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

func TestGenerateMarkdown_EscapesTableContentAndUsesRelativePaths(t *testing.T) {
	r := analysis.NewWorkspaceReport()
	r.TotalProjects = 1
	r.Projects = []analysis.Project{{
		Name:           "demo|name",
		Path:           "/workspace/nested/demo|name",
		Stack:          "Node|Runtime",
		Language:       "TypeScript",
		PackageManager: "pnpm",
		Tags:           []string{"tag|value"},
	}}
	r.LangCounts["TypeScript"] = 1
	r.StackCounts["Node|Runtime"] = 1
	r.PkgMgrCounts["pnpm"] = 1

	out := GenerateMarkdown(r, "/workspace")
	if strings.Contains(out, "Scanned Root:** `/workspace`") {
		t.Fatalf("shareable report must not expose the absolute scan root: %s", out)
	}
	for _, want := range []string{"demo\\|name", "Node\\|Runtime", "tag\\|value", "`nested/demo\\|name`"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected escaped markdown to contain %q:\n%s", want, out)
		}
	}
}

func TestGetTop_DeterministicTie(t *testing.T) {
	name, count := getTop(map[string]int{"Zulu": 2, "Alpha": 2})
	if name != "Alpha" || count != 2 {
		t.Fatalf("expected alphabetical tie-break, got %q (%d)", name, count)
	}
}
