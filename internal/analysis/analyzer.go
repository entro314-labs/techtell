package analysis

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	a.Warnings = nil

	err := filepath.WalkDir(a.rootPath, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			// Log permission errors or access issues but continue walking.
			a.Warnings = append(a.Warnings, fmt.Sprintf("checking path %s failed: %v", path, err))
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if updateChan != nil && time.Since(lastUpdate) > 50*time.Millisecond {
			select {
			case updateChan <- path:
				lastUpdate = time.Now()
			default:
				// Drop update if channel is full or blocked.
			}
		}

		if isExcluded(d.Name(), a.exclude) {
			return filepath.SkipDir
		}

		if isProjectRoot(path) {
			paths = append(paths, path)
		}

		return nil
	})

	sort.Strings(paths)
	return paths, err
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
		"Gemfile", "composer.json",
		"pom.xml", "build.gradle", "build.gradle.kts", "mix.exs",
		"docker-compose.yml", "docker-compose.yaml", "Dockerfile",
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
	files        []string
	content      map[string]string // filename -> content (for content signatures)
	dependencies map[string]struct{}
}

func loadProjectContext(root string) *projectContext {
	ctx := &projectContext{
		content:      make(map[string]string),
		dependencies: make(map[string]struct{}),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return ctx
	}

	fileSet := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		ctx.files = append(ctx.files, e.Name())
		fileSet[e.Name()] = struct{}{}
	}

	depCandidates := []string{
		"package.json", "go.mod", "Cargo.toml",
		"requirements.txt", "pyproject.toml", "Pipfile", "uv.lock",
		"build.gradle", "build.gradle.kts", "pom.xml", "cpanfile", "Gemfile", "composer.json",
	}

	for _, d := range depCandidates {
		if _, ok := fileSet[d]; !ok {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, d))
		if err != nil {
			continue
		}
		raw := string(b)
		ctx.content[d] = raw

		parser, ok := Parsers[d]
		if !ok {
			continue
		}
		deps, err := parser(b)
		if err != nil {
			continue
		}
		for _, dep := range deps {
			norm := normalizeToken(dep)
			if norm != "" {
				ctx.dependencies[norm] = struct{}{}
			}
		}
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if _, alreadyLoaded := ctx.content[name]; alreadyLoaded {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".sh" && ext != ".bash" && ext != ".zsh" && ext != ".ps1" && name != "Makefile" && name != "Dockerfile" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		if len(b) > 256*1024 {
			continue
		}
		ctx.content[name] = string(b)
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

	for _, sig := range Signatures {
		matched := false

		switch sig.MatchType {
		case MatchTypeFile:
			for _, id := range sig.Identifiers {
				normID := filepath.ToSlash(id)
				if strings.Contains(normID, "/") || strings.Contains(normID, "\\") {
					if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(normID))); err == nil {
						matched = true
						break
					}
					continue
				}
				for _, f := range ctx.files {
					if strings.EqualFold(f, id) {
						matched = true
						break
					}
				}
				if matched {
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
			for _, id := range sig.Identifiers {
				normID := normalizeToken(id)
				if normID == "" {
					continue
				}
				if _, ok := ctx.dependencies[normID]; ok {
					matched = true
					break
				}
			}
		case MatchTypeContent:
			for _, content := range ctx.content {
				lowerContent := strings.ToLower(content)
				for _, id := range sig.Identifiers {
					if strings.Contains(lowerContent, strings.ToLower(id)) {
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

	refineProject(&p, ctx)
	return p
}

func applySignature(p *Project, sig TechSignature) {
	switch sig.Category {
	case "Language":
		if p.Language == "Unknown" || languagePriority(sig.Name) > languagePriority(p.Language) {
			p.Language = sig.Name
		}
	case "Framework":
		if p.Stack == "Unknown" || isGenericStack(p.Stack) {
			p.Stack = sig.Name
		} else if p.Stack != sig.Name {
			if frameworkPriority(sig.Name) > frameworkPriority(p.Stack) {
				p.Tags = append(p.Tags, p.Stack)
				p.Stack = sig.Name
			} else {
				p.Tags = append(p.Tags, sig.Name)
			}
		}
	case "Package Manager":
		if p.PackageManager == "Unknown" || p.PackageManager == "" {
			p.PackageManager = sig.Name
		} else if p.PackageManager != sig.Name {
			p.Tags = append(p.Tags, sig.Name)
		}
	case "Library", "Build Tool", "Tool", "DevOps", "Cloud":
		p.Tags = append(p.Tags, sig.Name)
	}

	p.Tags = unique(p.Tags)
}

func refineProject(p *Project, ctx *projectContext) {
	if p.Language == "Unknown" {
		switch {
		case hasFile(ctx, "go.mod"):
			p.Language = "Go"
		case hasFile(ctx, "Cargo.toml"):
			p.Language = "Rust"
		case hasAnyFile(ctx, "requirements.txt", "pyproject.toml", "Pipfile", "uv.lock"):
			p.Language = "Python"
		case hasFile(ctx, "package.json"):
			if hasFile(ctx, "tsconfig.json") {
				p.Language = "TypeScript"
			} else {
				p.Language = "JavaScript"
			}
		}
	}

	hasMise := hasAnyFile(ctx, "mise.toml", ".mise.toml")
	if hasMise {
		if p.PackageManager == "Unknown" || p.PackageManager == "" {
			p.PackageManager = "mise"
		} else {
			p.PackageManager = "mise+" + p.PackageManager
		}
	} else if p.PackageManager == "Unknown" || p.PackageManager == "" {
		switch {
		case hasFile(ctx, "go.mod"):
			p.PackageManager = "go mod"
		case hasFile(ctx, "Cargo.toml"):
			p.PackageManager = "cargo"
		case hasFile(ctx, "requirements.txt"):
			p.PackageManager = "pip"
		case hasFile(ctx, "Pipfile"):
			p.PackageManager = "pipenv"
		case hasFile(ctx, "uv.lock"):
			p.PackageManager = "uv"
		}
	}

	switch p.Stack {
	case "Next.js":
		p.Stack = "Next.js App"
	case "Tauri":
		p.Stack = "Tauri App"
	}

	if (p.Language == "JavaScript" || p.Language == "TypeScript") && p.Stack == "Unknown" {
		p.Stack = "Node.js Project"
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

func normalizeToken(s string) string {
	trimmed := strings.TrimSpace(strings.ToLower(s))
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Trim(trimmed, "\"'")
	return trimmed
}

func hasFile(ctx *projectContext, name string) bool {
	for _, f := range ctx.files {
		if strings.EqualFold(f, name) {
			return true
		}
	}
	return false
}

func hasAnyFile(ctx *projectContext, names ...string) bool {
	for _, n := range names {
		if hasFile(ctx, n) {
			return true
		}
	}
	return false
}

func isGenericStack(stack string) bool {
	return stack == "Node.js Project" || stack == "Python Project"
}

func languagePriority(lang string) int {
	switch lang {
	case "TypeScript":
		return 4
	case "JavaScript":
		return 3
	case "Go", "Rust", "Python", "Java", "Kotlin", "Ruby", "PHP", "C", "C++", "Shell", "Lua", "HTML", "CSS", "SASS/SCSS":
		return 3
	default:
		return 1
	}
}

func frameworkPriority(stack string) int {
	switch stack {
	case "React":
		return 1
	case "Node.js Project", "Python Project", "Unknown":
		return 0
	default:
		return 3
	}
}
