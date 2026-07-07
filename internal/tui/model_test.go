package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdate_ErrorStateRestart(t *testing.T) {
	m := InitialModel()
	m.State = StateError
	m.Err = errors.New("boom")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	next := updated.(Model)

	if next.State != StateWelcome {
		t.Fatalf("expected StateWelcome, got %v", next.State)
	}
	if next.Err != nil {
		t.Fatalf("expected cleared error, got %v", next.Err)
	}
}

func TestUpdate_ScanningBackCancels(t *testing.T) {
	m := InitialModel()
	m.State = StateScanning
	cancelled := false
	m.cancel = func() { cancelled = true }

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(Model)

	if !cancelled {
		t.Fatal("expected cancel function to be called")
	}
	if next.State != StateWelcome {
		t.Fatalf("expected StateWelcome, got %v", next.State)
	}
}

func TestUpdate_RootsFoundCarriesWarnings(t *testing.T) {
	m := InitialModel()
	m.State = StateScanning

	updated, _ := m.Update(rootsFoundMsg(discoveryResult{warnings: []string{"perm denied"}}))
	next := updated.(Model)

	if next.State != StateResults {
		t.Fatalf("expected StateResults, got %v", next.State)
	}
	if next.Report == nil {
		t.Fatal("expected report to be initialized")
	}
	if len(next.Report.Errors) != 1 || next.Report.Errors[0] != "perm denied" {
		t.Fatalf("expected warnings to propagate into report errors, got %v", next.Report.Errors)
	}
}
