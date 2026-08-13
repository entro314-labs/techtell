package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/entro314-labs/techtell/internal/analysis"
	"github.com/entro314-labs/techtell/internal/report"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type State int

const (
	StateWelcome State = iota
	StateScanning
	StateResults
	StateError
)

type errMsg error
type rootsFoundMsg struct {
	scanID uint64
	result discoveryResult
}
type scanUpdateMsg struct {
	scanID uint64
	path   string
}
type projectAnalyzedMsg struct {
	scanID   uint64
	project  analysis.Project
	warnings []string
}

// KeyMap defines keybindings
type KeyMap struct {
	Tab      key.Binding
	Enter    key.Binding
	Back     key.Binding
	Restart  key.Binding
	Save     key.Binding
	SaveHTML key.Binding
	Quit     key.Binding
}

var Keys = KeyMap{
	Tab: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "switch view"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/start"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "save md"),
	),
	SaveHTML: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "save html"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Enter, k.Back, k.Restart, k.Save, k.SaveHTML, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Enter, k.Back, k.Restart},
		{k.Save, k.SaveHTML, k.Quit},
	}
}

type Model struct {
	State     State
	ActiveTab int // 0: Dashboard, 1: Projects

	Spinner   spinner.Model
	Progress  progress.Model
	TextInput textinput.Model
	Table     table.Model
	Help      help.Model

	Report      *analysis.WorkspaceReport
	Err         error
	CurrentPath string

	LastScanned string
	SavedFile   string

	pendingPaths []string // Queue for scanning
	scanCh       chan string
	scanDone     chan discoveryResult
	scanCount    int
	scanID       uint64

	analyzer *analysis.Analyzer
	cancel   context.CancelFunc
	width    int
	height   int
}

type discoveryResult struct {
	paths    []string
	warnings []string
	err      error
}

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = StylePrimaryStyle()

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60

	ti := textinput.New()
	ti.Placeholder = "Enter path to scan (default: current directory)"
	ti.Focus()
	ti.CharLimit = 4096
	ti.Width = 40

	// Initialize Table
	columns := []table.Column{
		{Title: "Project Name", Width: 25},
		{Title: "Stack", Width: 20},
		{Title: "Language", Width: 10},
		{Title: "Package Manager", Width: 15},
		{Title: "Tags", Width: 25},
		{Title: "Path", Width: 25},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	sTable := table.DefaultStyles()
	sTable.Header = sTable.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	sTable.Selected = sTable.Selected.
		Foreground(ColorSecondary).
		Bold(true).
		BorderLeft(false)
	t.SetStyles(sTable)

	wd, _ := os.Getwd()

	return Model{
		State:       StateWelcome,
		ActiveTab:   0,
		Spinner:     s,
		Progress:    prog,
		TextInput:   ti,
		Table:       t,
		Help:        help.New(),
		CurrentPath: wd,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.Spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global Quits
		if key.Matches(msg, Keys.Quit) {
			return m, tea.Quit
		}

		// Handle specific state keys
		switch m.State {
		case StateWelcome:
			switch {
			case key.Matches(msg, Keys.Enter):
				path := m.TextInput.Value()
				if path == "" {
					var err error
					wd, err := os.Getwd()
					if err != nil {
						m.Err = fmt.Errorf("failed to get current directory: %w", err)
						m.State = StateError
						return m, nil
					}
					path = wd
				}
				if abs, err := filepath.Abs(path); err == nil {
					path = abs
				}
				m.CurrentPath = path
				m.State = StateScanning
				m.ActiveTab = 0
				m.scanID++
				m.scanCount = 0
				m.Report = nil
				m.Err = nil
				m.SavedFile = ""
				m.LastScanned = ""
				m.pendingPaths = nil
				m.Table.SetRows(nil)
				if m.Progress.Width == 0 {
					m.Progress.Width = 60
				}
				m.analyzer = analysis.NewAnalyzer(path, nil)

				// Create cancellable context
				ctx, cancel := context.WithCancel(context.Background())
				m.cancel = cancel

				// START DISCOVERY (and restart spinner loop)
				return m, tea.Batch(startDiscovery(ctx, &m, m.scanID), m.Spinner.Tick)
			}
		case StateScanning:
			switch {
			case key.Matches(msg, Keys.Back):
				if m.cancel != nil {
					m.cancel()
					m.cancel = nil
				}
				m.scanID++
				m.State = StateWelcome
				m.Report = nil
				m.pendingPaths = nil
				m.Table.SetRows(nil)
				m.LastScanned = ""
				m.TextInput.Focus()
				return m, nil
			}
		case StateResults:
			switch {
			case key.Matches(msg, Keys.Save):
				filename := fmt.Sprintf("techtell_report_%d.md", time.Now().Unix())
				content := report.GenerateMarkdown(m.Report, m.CurrentPath)
				if err := report.SaveToFile(filename, content); err != nil {
					m.Err = err
					m.State = StateError
				} else {
					m.SavedFile = filename
				}
				return m, nil
			case key.Matches(msg, Keys.SaveHTML):
				filename := fmt.Sprintf("techtell_report_%d.html", time.Now().Unix())
				content := report.GenerateHTML(m.Report, m.CurrentPath)
				if err := report.SaveHTMLToFile(filename, content); err != nil {
					m.Err = err
					m.State = StateError
				} else {
					m.SavedFile = filename
				}
				return m, nil
			case key.Matches(msg, Keys.Back):
				if m.cancel != nil {
					m.cancel()
					m.cancel = nil
				}
				m.scanID++
				m.State = StateWelcome
				m.Report = nil
				m.pendingPaths = nil
				m.SavedFile = ""
				m.Table.SetRows(nil)
				m.TextInput.SetValue("")
				m.TextInput.Focus()
				return m, nil
			case key.Matches(msg, Keys.Tab):
				m.ActiveTab = 1 - m.ActiveTab // Toggle 0 <-> 1
				return m, nil
			}

			// Forward keys to Table ONLY if in Projects tab
			if m.ActiveTab == 1 {
				m.Table, cmd = m.Table.Update(msg)
				return m, cmd
			}
		case StateError:
			switch {
			case key.Matches(msg, Keys.Restart), key.Matches(msg, Keys.Back), key.Matches(msg, Keys.Enter):
				if m.cancel != nil {
					m.cancel()
					m.cancel = nil
				}
				m.scanID++
				m.State = StateWelcome
				m.Report = nil
				m.pendingPaths = nil
				m.Err = nil
				m.SavedFile = ""
				m.LastScanned = ""
				m.Table.SetRows(nil)
				m.TextInput.SetValue("")
				m.TextInput.Focus()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.Progress.Width = msg.Width - 10
		if m.Progress.Width < 1 {
			m.Progress.Width = 1
		}

		// Resize Table dynamically to fit
		// Title(3) + Tab(2) + Help(2) + CardBorder(2) = 9 lines fixed overhead
		// Plus standard padding/margins ~2-3
		// Safety = 14
		tblHeight := m.height - 14
		if tblHeight < 5 {
			tblHeight = 5
		}

		tableWidth := m.width - 4
		if tableWidth < 1 {
			tableWidth = 1
		}
		m.Table.SetWidth(tableWidth)
		m.Table.SetHeight(tblHeight)

		// Dynamic column width
		availableWidth := m.width - 4 - 10 // buffer
		if availableWidth < 80 {
			availableWidth = 80
		}

		// Fixed cols: Stack(15), Lang(10), Package Manager(12), Tags(20) = 57
		// Remainder split between Name (40%) and Path (60%)
		rem := availableWidth - 57
		nameCol := int(float64(rem) * 0.4)
		pathCol := int(float64(rem) * 0.6)

		cols := m.Table.Columns()
		if len(cols) == 6 {
			cols[0].Width = nameCol // Name
			cols[1].Width = 15      // Stack
			cols[2].Width = 10      // Lang
			cols[3].Width = 12      // Package Manager
			cols[4].Width = 20      // Tags
			cols[5].Width = pathCol // Path
			m.Table.SetColumns(cols)
		}

		m.Help.Width = m.width
		return m, nil

	case tea.MouseMsg:
		if m.State == StateResults && m.ActiveTab == 1 {
			m.Table, cmd = m.Table.Update(msg)
			return m, cmd
		}

	// --- Progress Flow ---
	case scanUpdateMsg:
		if msg.scanID != m.scanID || m.State != StateScanning {
			return m, nil
		}
		m.LastScanned = msg.path
		m.scanCount++

		// Keep listening
		return m, listenForActivity(msg.scanID, m.scanCh, m.scanDone)

	case rootsFoundMsg:
		if msg.scanID != m.scanID || m.State != StateScanning {
			return m, nil
		}
		res := msg.result
		if res.err != nil {
			m.Err = res.err
			m.State = StateError
			m.cancel = nil
			return m, nil
		}

		paths := res.paths
		count := len(paths)
		if count == 0 {
			m.Report = analysis.NewWorkspaceReport()
			m.Report.Errors = append(m.Report.Errors, res.warnings...)
			m.State = StateResults
			m.cancel = nil
			m.Table.SetRows(nil)
			return m, nil
		}

		m.Report = analysis.NewWorkspaceReport()
		m.Report.TotalProjects = count
		m.Report.Errors = append(m.Report.Errors, res.warnings...)
		m.pendingPaths = paths

		// Start processing chain
		return m, processNextProject(msg.scanID, m.analyzer, m.pendingPaths)

	case projectAnalyzedMsg:
		if msg.scanID != m.scanID || m.State != StateScanning || m.Report == nil {
			return m, nil
		}
		p := msg.project
		// Add to report
		m.Report.Projects = append(m.Report.Projects, p)
		m.Report.Errors = append(m.Report.Errors, msg.warnings...)
		m.Report.StackCounts[p.Stack]++
		m.Report.LangCounts[p.Language]++
		m.Report.PkgMgrCounts[p.PackageManager]++
		m.LastScanned = p.Name

		// Remove from pending
		if len(m.pendingPaths) > 0 {
			m.pendingPaths = m.pendingPaths[1:]
		}

		// Update Progress
		current := float64(len(m.Report.Projects))
		total := float64(m.Report.TotalProjects)
		pct := current / total

		cmd := m.Progress.SetPercent(pct)

		// Continue?
		if len(m.pendingPaths) > 0 {
			return m, tea.Batch(cmd, inspectProjectCmd(msg.scanID, m.analyzer, m.pendingPaths[0]))
		}

		// Done
		sort.Slice(m.Report.Projects, func(i, j int) bool {
			if m.Report.Projects[i].Name == m.Report.Projects[j].Name {
				return m.Report.Projects[i].Path < m.Report.Projects[j].Path
			}
			return m.Report.Projects[i].Name < m.Report.Projects[j].Name
		})
		m.State = StateResults
		m.cancel = nil

		// Populate Table
		var rows []table.Row
		for _, p := range m.Report.Projects {
			relPath, _ := filepath.Rel(m.CurrentPath, p.Path)
			if relPath == "" || relPath == "." {
				relPath = "/"
			}
			tags := strings.Join(p.Tags, ", ")
			rows = append(rows, table.Row{p.Name, p.Stack, p.Language, p.PackageManager, tags, relPath})
		}
		m.Table.SetRows(rows)

		return m, cmd

	case errMsg:
		m.Err = msg
		m.State = StateError
		return m, nil

	// Standard bubbles updates
	case spinner.TickMsg:
		if m.State == StateScanning {
			var cmd1 tea.Cmd
			m.Spinner, cmd1 = m.Spinner.Update(msg)

			// Time-based Pulse for Discovery Phase (when Report is nil)
			if m.Report == nil {
				// Loop 0.0 -> 1.0 every 1.5 seconds
				period := int64(1500 * time.Millisecond)
				now := time.Now().UnixNano()
				pct := float64(now%period) / float64(period)
				cmd2 := m.Progress.SetPercent(pct)
				return m, tea.Batch(cmd1, cmd2)
			}

			return m, cmd1
		}
	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		m.Progress = progressModel.(progress.Model)
		return m, cmd
	}

	// Update components
	switch m.State {
	case StateWelcome:
		m.TextInput, cmd = m.TextInput.Update(msg)
		return m, cmd
		// case StateScanning: // Handled in Batch above
	}

	return m, nil
}

// Helper to start discovery and return the first listener cmd
func startDiscovery(ctx context.Context, m *Model, scanID uint64) tea.Cmd {
	m.scanCh = make(chan string)
	m.scanDone = make(chan discoveryResult, 1)

	go func() {
		defer close(m.scanCh)
		paths, err := m.analyzer.DiscoverProjects(ctx, m.scanCh)
		warnings := append([]string(nil), m.analyzer.Warnings...)
		m.scanDone <- discoveryResult{paths: paths, warnings: warnings, err: err}
		close(m.scanDone)
	}()

	return listenForActivity(scanID, m.scanCh, m.scanDone)
}

func listenForActivity(scanID uint64, ch <-chan string, done <-chan discoveryResult) tea.Cmd {
	return func() tea.Msg {
		select {
		case path, ok := <-ch:
			if !ok {
				// Channel closed. Wait for done.
				res := <-done
				return rootsFoundMsg{scanID: scanID, result: res}
			}
			return scanUpdateMsg{scanID: scanID, path: path}
		case res := <-done:
			return rootsFoundMsg{scanID: scanID, result: res}
		}
	}
}

func inspectProjectCmd(scanID uint64, a *analysis.Analyzer, path string) tea.Cmd {
	return func() tea.Msg {
		p, warnings := a.InspectProject(path)
		return projectAnalyzedMsg{scanID: scanID, project: p, warnings: warnings}
	}
}

func processNextProject(scanID uint64, a *analysis.Analyzer, paths []string) tea.Cmd {
	if len(paths) == 0 {
		return nil
	}
	return inspectProjectCmd(scanID, a, paths[0])
}

func StylePrimaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorPrimary)
}
