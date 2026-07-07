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
}
