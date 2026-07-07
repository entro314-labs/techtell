package report

import (
	"strings"
	"testing"

	"github.com/entro314-labs/techtell/internal/analysis"
)

func TestGenerateMarkdown_IncludesPackageManagersAndWarnings(t *testing.T) {
	r := analysis.NewWorkspaceReport()
	r.TotalProjects = 1
	r.Projects = []analysis.Project{{
		Name:           "demo",
		Path:           "/workspace/demo",
		Stack:          "Node.js Project",
		Language:       "JavaScript",
		PackageManager: "NPM",
		Tags:           []string{"Vite"},
	}}
	r.LangCounts["JavaScript"] = 1
	r.StackCounts["Node.js Project"] = 1
	r.PkgMgrCounts["NPM"] = 1
	r.Errors = append(r.Errors, "permission denied while scanning /workspace/secret")

	out := GenerateMarkdown(r, "/workspace")
	for _, want := range []string{"### Package Managers", "| NPM | 1 |", "## ⚠️ Warnings", "permission denied while scanning /workspace/secret"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected markdown output to contain %q", want)
		}
	}
}
