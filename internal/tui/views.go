package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var s string
	switch m.State {
	case StateWelcome:
		s = m.viewWelcome()
	case StateScanning:
		s = m.viewScanning()
	case StateResults:
		s = m.viewResults()
	case StateError:
		s = m.viewError()
	default:
		s = "Unknown state"
	}

	// Center everything roughly
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s)
}

func (m Model) viewWelcome() string {
	return StyleContainer.Render(
		fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			StyleTitle.Render("Welcome to TechTell"),
			"Enter Dev Folder path to analyze (leave empty for current):",
			m.TextInput.View(),
		),
	)
}

func (m Model) viewScanning() string {
	// Truncate logic
	maxWidth := m.width - 20
	if maxWidth < 40 {
		maxWidth = 40
	}

	currentPath := truncateHead(m.CurrentPath, maxWidth)
	lastScanned := truncateHead(m.LastScanned, maxWidth-10)

	// Status text
	var action string
	var details string

	if m.Report != nil && m.Report.TotalProjects > 0 {
		// Phase 2: Inspection
		action = fmt.Sprintf("Processing %d projects...", m.Report.TotalProjects)
		details = fmt.Sprintf("Analyzing: %s", lastScanned)

		return StyleContainer.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleTitle.Render("TechTell Analysis"),
				"",
				m.Spinner.View()+" "+action,
				details,
				"",
				m.Progress.View(),
			),
		)
	}

	// Phase 1: Discovery
	action = "Scanning workspace..."
	details = fmt.Sprintf("Found: %s", lastScanned)

	return StyleContainer.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			StyleTitle.Render("TechTell Analysis"),
			"",
			m.Spinner.View()+" "+action,
			StyleFocused.Render(currentPath),
			"",
			details,
			"",
			// Persistent Progress Bar
			m.Progress.View(),
		),
	)
}

func truncateHead(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "..." + s[len(s)-(max-3):]
}

func (m Model) viewResults() string {
	if m.Report == nil {
		return "No results."
	}

	// --- Tabs ---
	tabDashboard := " Dashboard "
	tabInventory := " Projects "

	if m.ActiveTab == 0 {
		tabDashboard = StyleActiveTab.Render(tabDashboard)
		tabInventory = StyleTab.Render(tabInventory)
	} else {
		tabDashboard = StyleTab.Render(tabDashboard)
		tabInventory = StyleActiveTab.Render(tabInventory)
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Bottom, tabDashboard, tabInventory)
	tabBar = lipgloss.NewStyle().BorderBottom(true).BorderForeground(ColorGray).Width(m.width - 4).Render(tabBar)

	var contentView string

	if m.ActiveTab == 0 { // Dashboard
		// --- Stats Row ---
		totalProjects := StyleStatBox.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleStatLabel.Render("PROJECTS"),
				StyleStatValue.Render(fmt.Sprintf("%d", m.Report.TotalProjects)),
			),
		)

		topLang, _ := topCategory(m.Report.LangCounts)
		topStack, _ := topCategory(m.Report.StackCounts)
		topPM, _ := topCategory(m.Report.PkgMgrCounts)

		primaryLang := StyleStatBox.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleStatLabel.Render("TOP LANG"),
				StyleStatValue.Render(topLang),
			),
		)

		primaryStack := StyleStatBox.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleStatLabel.Render("TOP STACK"),
				StyleStatValue.Render(topStack),
			),
		)

		primaryPM := StyleStatBox.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleStatLabel.Render("PKG MGR"),
				StyleStatValue.Render(topPM),
			),
		)

		headerRow := lipgloss.JoinHorizontal(lipgloss.Top, totalProjects, primaryLang, primaryStack, primaryPM)

		// --- Charts Section ---
		chartWidth := (m.width - 10) / 2
		if chartWidth < 30 {
			chartWidth = 30
		}

		// Stack Chart
		stackChart := StyleCard.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				StyleHeader.Render("Stack Distribution"),
				BarChart(m.Report.StackCounts, chartWidth),
			),
		)

		// Language Chart
		langChart := StyleCard.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				StyleHeader.Render("Language DNA"),
				BarChart(m.Report.LangCounts, chartWidth),
			),
		)

		chartsRow := lipgloss.JoinHorizontal(lipgloss.Top, stackChart, langChart)

		contentView = lipgloss.JoinVertical(lipgloss.Center, headerRow, chartsRow)

	} else { // Inventory (Table)
		contentView = StyleCard.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				m.Table.View(),
			),
		)
	}

	// Header
	title := StyleTitle.Render("Results for: " + filepath.Base(m.CurrentPath))

	// Footer / Help
	helpView := m.Help.View(Keys)

	// Notifications
	var notifications []string
	if len(m.Report.Errors) > 0 {
		notifications = append(notifications, StyleError.Render(fmt.Sprintf("%d warnings encountered.", len(m.Report.Errors))))
	}
	if m.SavedFile != "" {
		notifications = append(notifications, StyleSuccess.Render(fmt.Sprintf("Report saved to: %s", m.SavedFile)))
	}
	notify := strings.Join(notifications, "\n")

	// Assemble
	final := lipgloss.JoinVertical(lipgloss.Center,
		title,
		tabBar,
		StyleContent.Render(contentView),
		notify,
		StyleContainer.Render(helpView),
	)

	return final
}

func (m Model) viewError() string {
	return StyleContainer.Render(
		fmt.Sprintf(
			"%s\n\nError: %v\n\n%s",
			StyleTitle.Render("Error"),
			m.Err,
			StyleHeader.Render("(r/enter/esc) restart, (q) quit"),
		),
	)
}

func StyleSecondary(s string) string {
	return lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(s)
}
