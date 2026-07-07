package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/entro314-labs/techtell/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type cliAction int

const (
	actionRun cliAction = iota
	actionVersion
	actionHelp
)

func buildVersion() string {
	if version == "dev" {
		return "dev (built from source)"
	}
	return fmt.Sprintf("%s (commit: %s, built: %s, by: %s)", version, commit, date, builtBy)
}

func parseCLIAction(args []string) cliAction {
	if slices.Contains(args, "--version") || slices.Contains(args, "-v") {
		return actionVersion
	}
	if slices.Contains(args, "--help") || slices.Contains(args, "-h") || slices.Contains(args, "-help") {
		return actionHelp
	}
	return actionRun
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "techtell - workspace tech DNA analyzer")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  techtell           Start interactive TUI")
	fmt.Fprintln(w, "  techtell --version Show version")
	fmt.Fprintln(w, "  techtell --help    Show help")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "In the TUI:")
	fmt.Fprintln(w, "  Enter              Start scan")
	fmt.Fprintln(w, "  Tab/Shift+Tab      Switch result views")
	fmt.Fprintln(w, "  s                  Save Markdown report")
	fmt.Fprintln(w, "  h                  Save HTML report")
	fmt.Fprintln(w, "  Esc/Backspace      Back/Cancel")
	fmt.Fprintln(w, "  q or Ctrl+C        Quit")
}

func main() {
	switch parseCLIAction(os.Args[1:]) {
	case actionVersion:
		fmt.Printf("techtell %s\n", buildVersion())
		return
	case actionHelp:
		printHelp(os.Stdout)
		return
	case actionRun:
	}

	p := tea.NewProgram(
		tui.InitialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
