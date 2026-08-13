package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/entro314-labs/techtell/internal/analysis"
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

	updated, _ := m.Update(rootsFoundMsg{scanID: m.scanID, result: discoveryResult{warnings: []string{"perm denied"}}})
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

func TestUpdate_IgnoresStaleProjectAfterCancellation(t *testing.T) {
	m := InitialModel()
	m.State = StateScanning
	m.scanID = 4
	m.Report = analysis.NewWorkspaceReport()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(Model)
	if next.State != StateWelcome || next.scanID != 5 {
		t.Fatalf("expected cancellation to invalidate scan, got state=%v scanID=%d", next.State, next.scanID)
	}

	updated, _ = next.Update(projectAnalyzedMsg{scanID: 4, project: analysis.Project{Name: "stale"}})
	next = updated.(Model)
	if next.State != StateWelcome || next.Report != nil {
		t.Fatalf("stale project result changed cancelled state: %+v", next)
	}
}

func TestUpdate_IgnoresStaleDiscoveryAfterNewScanGeneration(t *testing.T) {
	m := InitialModel()
	m.State = StateScanning
	m.scanID = 2
	m.Report = analysis.NewWorkspaceReport()

	updated, _ := m.Update(rootsFoundMsg{scanID: 1, result: discoveryResult{paths: []string{"/old"}}})
	next := updated.(Model)
	if next.State != StateScanning || next.Report.TotalProjects != 0 {
		t.Fatalf("stale discovery changed active scan: state=%v report=%+v", next.State, next.Report)
	}
}

func TestUpdate_EmptyResultClearsPreviousRows(t *testing.T) {
	m := InitialModel()
	m.State = StateScanning
	m.scanID = 1
	m.Table.SetRows([]table.Row{{"old"}})

	updated, _ := m.Update(rootsFoundMsg{scanID: 1, result: discoveryResult{}})
	next := updated.(Model)
	if got := len(next.Table.Rows()); got != 0 {
		t.Fatalf("expected empty scan to clear old rows, got %d", got)
	}
}
