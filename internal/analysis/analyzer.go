package analysis

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Project represents a single detected project
type Project struct {
	Name           string
	Path           string
	Stack          string // e.g. "Next.js", "Tauri", "Go CLI"
	Language       string // e.g. "TypeScript", "Go"
	PackageManager string // e.g. "npm", "yarn", "pnpm", "cargo", "go mod"
	Tags           []string
}

// WorkspaceReport aggregates findings across all projects
type WorkspaceReport struct {
	TotalProjects int
	Projects      []Project
	StackCounts   map[string]int
	LangCounts    map[string]int
	PkgMgrCounts  map[string]int
	Errors        []string
}

func NewWorkspaceReport() *WorkspaceReport {
	return &WorkspaceReport{
		StackCounts:  make(map[string]int),
		LangCounts:   make(map[string]int),
		PkgMgrCounts: make(map[string]int),
	}
}

// Analyzer walks the directory and builds a report
type Analyzer struct {
	rootPath string
	exclude  []string
	Warnings []string
}

func NewAnalyzer(root string, exclude []string) *Analyzer {
	if len(exclude) == 0 {
		exclude = []string{
			".git", "node_modules", "dist", "build", "vendor",
			".idea", ".vscode", "target", "__pycache__",
			".next", ".astro", ".output", ".cache", "tmp",
			"deps", "third_party", "examples", "samples",
		}
	}
	return &Analyzer{
		rootPath: root,
		exclude:  exclude,
	}
}

// DiscoverProjects scans the workspace for project roots
func (a *Analyzer) DiscoverProjects(ctx context.Context, updateChan chan<- string) ([]string, error) {
	var paths []string
	var lastUpdate time.Time

	// Fast walk
	err := filepath.WalkDir(a.rootPath, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			// Log permission errors or access issues but continue walking
			a.Warnings = append(a.Warnings, fmt.Sprintf("checking path %s failed: %v", path, err))
			return nil
		}

		if d.IsDir() {
			// Send update (Throttled to max 20fps to keep UI responsive)
			if updateChan != nil && time.Since(lastUpdate) > 50*time.Millisecond {
				select {
				case updateChan <- path:
					lastUpdate = time.Now()
				default:
					// Drop update if channel full or blocked
				}
			}

			// Exclude logic
			if isExcluded(d.Name(), a.exclude) {
				return filepath.SkipDir
			}

			// Marker check
			if isProjectRoot(path) {
				paths = append(paths, path)

				// Smart Recursion:
				// If this project does NOT look like a monorepo, stop recursing.
				// This prevents finding "CMakeLists.txt" deep inside a "package.json" project.
				if !isMonorepo(path) {
					return filepath.SkipDir
				}
			}
		}
		return nil
	})
	return paths, err
}

func isMonorepo(path string) bool {
	// Check for explicit workspace markers
	if _, err := os.Stat(filepath.Join(path, "pnpm-workspace.yaml")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(path, "lerna.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(path, "rush.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(path, "go.work")); err == nil {
		return true
	}

	// Check package.json for "workspaces"
	if content, err := os.ReadFile(filepath.Join(path, "package.json")); err == nil {
		if strings.Contains(string(content), "\"workspaces\"") {
			return true
		}
	}

	// Check for common monorepo folders? (packages, apps, libs)
	// Only if they exist and are directories?
	// Let's be conservative. If it has "packages" folder, suspect monorepo.
	if info, err := os.Stat(filepath.Join(path, "packages")); err == nil && info.IsDir() {
		return true
	}
	if info, err := os.Stat(filepath.Join(path, "apps")); err == nil && info.IsDir() {
		return true
	}

	return false
}

// InspectProject analyzes a single project path
func (a *Analyzer) InspectProject(path string) Project {
	name := filepath.Base(path)
	return identProject(path, name)
}

func isExcluded(name string, excludes []string) bool {
	for _, ex := range excludes {
		if name == ex {
			return true
		}
	}
	return false
}

func isProjectRoot(path string) bool {
	markers := []string{
		"go.mod",
		"package.json",
		"Cargo.toml",
		"requirements.txt", "pyproject.toml", "Pipfile", "uv.lock", "setup.py",
		"pom.xml", "mix.exs", "docker-compose.yml", "Dockerfile",
		"mise.toml", ".mise.toml",
		"CMakeLists.txt",
	}
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(path, m)); err == nil {
			return true
		}
	}
	return false
}

type projectContext struct {
	files    []string
	content  map[string]string // filename -> content
	depFiles []string          // list of dependency file names found
}

func loadProjectContext(root string) *projectContext {
	ctx := &projectContext{
		content: make(map[string]string),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return ctx
	}

	for _, e := range entries {
		ctx.files = append(ctx.files, e.Name())
	}

	// Pre-load common dependency files
	depCandidates := []string{
		"package.json", "go.mod", "Cargo.toml",
		"requirements.txt", "pyproject.toml", "Pipfile",
		"build.gradle", "pom.xml", "cpanfile", "Gemfile", "composer.json",
	}

	for _, d := range depCandidates {
		// Only load if it exists in file list (avoid redundant stats)
		found := false
		for _, f := range ctx.files {
			if f == d {
				found = true
				break
			}
		}
		if found {
			ctx.depFiles = append(ctx.depFiles, d)
			b, err := os.ReadFile(filepath.Join(root, d))
			if err == nil {
				ctx.content[d] = string(b)
			}
		}
	}
	return ctx
}

func identProject(root string, name string) Project {
	p := Project{
		Name:           name,
		Path:           root,
		Stack:          "Unknown",
		Language:       "Unknown",
		PackageManager: "Unknown",
	}

	ctx := loadProjectContext(root)

	// Single pass filtering using Signatures
	for _, sig := range Signatures {
		matched := false

		switch sig.MatchType {
		case MatchTypeFile:
			for _, f := range ctx.files {
				if sig.Matches(f) {
					matched = true
					break
				}
			}
		case MatchTypeExtension:
			for _, f := range ctx.files {
				if sig.Matches(f) {
					matched = true
					break
				}
			}
		case MatchTypeDependency:
			// search in all loaded dependency content
			for _, content := range ctx.content {
				for _, id := range sig.Identifiers {
					if strings.Contains(content, id) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
		}

		if matched {
			applySignature(&p, sig)
		}
	}

	// Post-process / refinement
	refineProject(&p, ctx)

	return p
}

func applySignature(p *Project, sig TechSignature) {
	switch sig.Category {
	case "Language":
		if p.Language == "Unknown" {
			p.Language = sig.Name
		} else if p.Language != sig.Name {
			// Multi-language project strategy:
			// Maybe prioritize some or list both?
			// For simplicity, overwrite if "Unknown", otherwise keep or refined logic.
			// Let's bias towards specific project markers.
			// "Go Module" (File) > "Go" (Ext)
			if sig.MatchType == MatchTypeFile {
				p.Language = sig.Name
			}
		}
	case "Framework":
		if p.Stack == "Unknown" || p.Stack == "Node.js Project" || p.Stack == "Python Project" {
			p.Stack = sig.Name
		} else {
			// If we already have a stack, maybe append tag?
			// e.g. "Next.js" found, but already "React" found.
			// If sig is "Next.js", it overrides "React".
			// Naive: Just tag it.
			p.Tags = append(p.Tags, sig.Name)
		}
	case "Package Manager":
		p.PackageManager = sig.Name
	case "Library":
		p.Tags = append(p.Tags, sig.Name)
	case "Tool", "DevOps", "Cloud":
		p.Tags = append(p.Tags, sig.Name)
	}

	// Deduplicate tags
	p.Tags = unique(p.Tags)
}

func refineProject(p *Project, ctx *projectContext) {
	// 1. Language Fallbacks based on files
	if p.Language == "Unknown" {
		for _, f := range ctx.files {
			if strings.HasSuffix(f, ".js") {
				p.Language = "JavaScript"
				break
			}
		}
	}

	// 2. PackageManager Refinement
	// If mise is present, wrap it
	hasMise := false
	for _, f := range ctx.files {
		if f == "mise.toml" || f == ".mise.toml" {
			hasMise = true
			break
		}
	}
	if hasMise {
		if p.PackageManager == "Unknown" || p.PackageManager == "" {
			p.PackageManager = "mise"
		} else {
			p.PackageManager = "mise+" + p.PackageManager
		}
	}

	// 3. Stack Refinement
	// If we detected a framework like "Next.js", formatting standardizes it.
	switch p.Stack {
	case "Next.js":
		p.Stack = "Next.js App"
	case "Tauri":
		p.Stack = "Tauri App"
	}

	// Fallback for JS projects without frameworks detected
	if p.Language == "JavaScript" || p.Language == "TypeScript" {
		if p.Stack == "Unknown" {
			p.Stack = "Node.js Project"
		}
	}
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
