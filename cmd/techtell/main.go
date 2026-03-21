package main

import (
	"fmt"
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

func buildVersion() string {
	if version == "dev" {
		return "dev (built from source)"
	}
	return fmt.Sprintf("%s (commit: %s, built: %s, by: %s)", version, commit, date, builtBy)
}

func main() {
	if slices.Contains(os.Args[1:], "--version") || slices.Contains(os.Args[1:], "-v") {
		fmt.Printf("techtell %s\n", buildVersion())
		return
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
