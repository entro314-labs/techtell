package main

import (
	"fmt"
	"io"
	"os"

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

func parseCLIAction(args []string) (cliAction, error) {
	if len(args) == 0 {
		return actionRun, nil
	}
	if len(args) > 1 {
		return actionRun, fmt.Errorf("expected at most one argument, got %d", len(args))
	}
	switch args[0] {
	case "--version", "-v":
		return actionVersion, nil
	case "--help", "-h", "-help":
		return actionHelp, nil
	default:
		return actionRun, fmt.Errorf("unknown argument %q", args[0])
	}
}

func printHelp(w io.Writer) error {
	_, err := fmt.Fprint(w, `techtell - workspace tech DNA analyzer

Usage:
  techtell           Start interactive TUI
  techtell --version Show version
  techtell --help    Show help

In the TUI:
  Enter              Start scan
  Tab/Shift+Tab      Switch result views
  s                  Save Markdown report
  h                  Save HTML report
  Esc/Backspace      Back/Cancel
  q or Ctrl+C        Quit
`)
	return err
}

func main() {
	action, err := parseCLIAction(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	switch action {
	case actionVersion:
		fmt.Printf("techtell %s\n", buildVersion())
		return
	case actionHelp:
		if err := printHelp(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing help: %v\n", err)
			os.Exit(1)
		}
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
