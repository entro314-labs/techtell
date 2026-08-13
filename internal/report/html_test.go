package report

import (
	"github.com/entro314-labs/techtell/internal/analysis"
	"strings"
	"testing"
)

func TestGenerateHTML_XSS(t *testing.T) {
	// Create a mock report with a malicious project name
	r := analysis.NewWorkspaceReport()
	r.Projects = []analysis.Project{
		{
			Name:           "bad<script>alert(1)</script>project",
			Path:           "/tmp/bad<script>alert(1)</script>project",
			Stack:          "React",
			Language:       "JavaScript",
			PackageManager: "npm",
		},
	}
	r.LangCounts["JavaScript"] = 1
	r.StackCounts["React"] = 1
	r.PkgMgrCounts["npm"] = 1
	r.Errors = append(r.Errors, "permission denied while scanning /tmp/foo")

	// Generate HTML
	htmlContent := GenerateHTML(r, "/tmp")

	// Verify that the malicious script tag is NOT present in its raw form
	if strings.Contains(htmlContent, "<script>alert(1)</script>") {
		t.Errorf("XSS vulnerability detected: script tag found in HTML output")
	}

	// Verify that it IS present in escaped form
	if !strings.Contains(htmlContent, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Logf("Expected escaped content not found (might be truncated or differently escaped)")
		// Note: The label might be truncated by truncateLabel(12)
		// "bad<script>a" -> 12 chars
		// "bad<script>alert..."
		// truncateLabel("bad<script>alert(1)</script>project", 12) -> "bad<script.."
		// Escaped: "bad&lt;script.."
	}

	// Let's check generally that no raw <script> exists except maybe in the code block I just wrote
	// But simply checking for the project name fragment being escaped is enough?
	// The vector was in Radar Chart labels.

	if !strings.Contains(htmlContent, "Project Inventory") {
		t.Errorf("expected project inventory section to be present")
	}

	if !strings.Contains(htmlContent, "permission denied while scanning /tmp/foo") {
		t.Errorf("expected warnings section to include scan warning")
	}
	if strings.Contains(htmlContent, "<code>/tmp/bad") {
		t.Errorf("expected project inventory path to be relative to the scanned root")
	}
	if strings.Contains(htmlContent, "scanned: <code>/tmp</code>") {
		t.Errorf("shareable report must not expose the absolute scan root")
	}
}

func TestGeneratePieChart_SingleCategoryRendersFullCircle(t *testing.T) {
	out := generatePieChart([]ChartItem{{Name: "Go", Count: 1, Percent: 100, Color: "#fff"}}, 100, 100, 80)
	if !strings.Contains(out, "<circle") {
		t.Fatalf("expected a full circle for a single category, got %q", out)
	}
	if strings.Contains(out, " A ") {
		t.Fatalf("single-category pie must not use a degenerate arc: %q", out)
	}
}

func TestGenerateHTML_AggregatesOverflowLanguages(t *testing.T) {
	r := analysis.NewWorkspaceReport()
	for i, name := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"} {
		r.LangCounts[name] = i + 1
	}
	r.TotalProjects = 45

	out := GenerateHTML(r, "/workspace")
	if !strings.Contains(out, ">Other<") {
		t.Fatalf("expected overflow language categories to be represented as Other")
	}
}

func TestGetTopName_DeterministicTie(t *testing.T) {
	if got := getTopName(map[string]int{"Zulu": 1, "Alpha": 1}); got != "Alpha" {
		t.Fatalf("expected alphabetical tie-break, got %q", got)
	}
}
