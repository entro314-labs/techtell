package analysis

import (
	"context"
	"encoding/json"
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
func (a *Analyzer) InspectProject(path string) (Project, []string) {
	name := filepath.Base(path)
	return identProject(path, name, a.exclude)
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
		if info, err := os.Stat(filepath.Join(path, m)); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}

type projectContext struct {
	files            []string
	sourceFiles      []string
	content          map[string]string // dependency manifest filename -> content
	signatureContent map[string]string
	dependencies     map[string]struct{}
	warnings         []string
}

func loadProjectContext(root string, excludes []string) *projectContext {
	ctx := &projectContext{
		content:          make(map[string]string),
		signatureContent: make(map[string]string),
		dependencies:     make(map[string]struct{}),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		ctx.warnings = append(ctx.warnings, fmt.Sprintf("reading project root %s failed: %v", root, err))
		return ctx
	}

	fileSet := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		ctx.files = append(ctx.files, e.Name())
		fileSet[e.Name()] = struct{}{}
	}

	depCandidates := []string{
		"package.json", "go.mod", "Cargo.toml",
		"requirements.txt", "pyproject.toml", "Pipfile",
		"build.gradle", "build.gradle.kts", "pom.xml", "cpanfile", "Gemfile", "composer.json",
	}

	for _, d := range depCandidates {
		if _, ok := fileSet[d]; !ok {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, d))
		if err != nil {
			ctx.warnings = append(ctx.warnings, fmt.Sprintf("reading dependency manifest %s failed: %v", filepath.Join(root, d), err))
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
			ctx.warnings = append(ctx.warnings, fmt.Sprintf("parsing dependency manifest %s failed: %v", filepath.Join(root, d), err))
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
			ctx.warnings = append(ctx.warnings, fmt.Sprintf("reading signature source %s failed: %v", filepath.Join(root, name), err))
			continue
		}
		if len(b) > 256*1024 {
			continue
		}
		ctx.signatureContent[name] = string(b)
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			ctx.warnings = append(ctx.warnings, fmt.Sprintf("checking project source %s failed: %v", path, walkErr))
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if isExcluded(d.Name(), excludes) || isProjectRoot(path) {
				return filepath.SkipDir
			}
			return nil
		}
		ctx.sourceFiles = append(ctx.sourceFiles, d.Name())
		return nil
	})
	if err != nil {
		ctx.warnings = append(ctx.warnings, fmt.Sprintf("walking project source %s failed: %v", root, err))
	}

	return ctx
}

func identProject(root string, name string, excludes []string) (Project, []string) {
	p := Project{
		Name:           name,
		Path:           root,
		Stack:          "Unknown",
		Language:       "Unknown",
		PackageManager: "Unknown",
	}

	ctx := loadProjectContext(root, excludes)

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
			for _, f := range ctx.sourceFiles {
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
			for _, content := range ctx.signatureContent {
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
	return p, ctx.warnings
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

	var packageManagerSignals []string
	p.PackageManager, packageManagerSignals = detectPackageManager(p.Language, ctx)
	if len(packageManagerSignals) > 1 {
		ctx.warnings = append(ctx.warnings, fmt.Sprintf(
			"project %s has conflicting package manager signals (%s); selected %s",
			p.Path, strings.Join(packageManagerSignals, ", "), p.PackageManager,
		))
	}
	if hasAnyFile(ctx, "mise.toml", ".mise.toml") {
		p.Tags = unique(append(p.Tags, "mise"))
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

func detectPackageManager(language string, ctx *projectContext) (string, []string) {
	if language == "JavaScript" || language == "TypeScript" {
		var declared string
		if raw, ok := ctx.content["package.json"]; ok {
			var pkg struct {
				PackageManager string `json:"packageManager"`
			}
			if err := json.Unmarshal([]byte(raw), &pkg); err == nil {
				if name := strings.TrimSpace(strings.SplitN(pkg.PackageManager, "@", 2)[0]); name != "" {
					declared = strings.ToLower(name)
				}
			}
		}
		signals := packageManagerSignals(ctx, []struct {
			name  string
			files []string
		}{
			{name: "bun", files: []string{"bun.lock", "bun.lockb"}},
			{name: "pnpm", files: []string{"pnpm-lock.yaml"}},
			{name: "yarn", files: []string{"yarn.lock"}},
			{name: "npm", files: []string{"package-lock.json", "npm-shrinkwrap.json"}},
		})
		if declared != "" {
			signals = uniqueStrings(append(signals, declared))
			sort.Strings(signals)
			return declared, signals
		}
		if len(signals) > 0 {
			return signals[0], signals
		}
	}

	if language == "Python" {
		signals := make([]string, 0, 5)
		if hasFile(ctx, "uv.lock") {
			signals = append(signals, "uv")
		}
		if hasFile(ctx, "Pipfile") {
			signals = append(signals, "pipenv")
		}
		if hasFile(ctx, "poetry.lock") || contentContains(ctx, "pyproject.toml", "[tool.poetry]") {
			signals = append(signals, "poetry")
		}
		if hasFile(ctx, "pdm.lock") || contentContains(ctx, "pyproject.toml", "[tool.pdm]") {
			signals = append(signals, "pdm")
		}
		if hasFile(ctx, "requirements.txt") {
			signals = append(signals, "pip")
		}
		signals = uniqueStrings(signals)
		if len(signals) > 0 {
			return signals[0], signals
		}
	}

	switch {
	case hasFile(ctx, "go.mod"):
		return "go mod", []string{"go mod"}
	case hasFile(ctx, "Cargo.toml"):
		return "cargo", []string{"cargo"}
	case hasFile(ctx, "pom.xml"):
		return "maven", []string{"maven"}
	case hasAnyFile(ctx, "build.gradle", "build.gradle.kts", "gradlew"):
		return "gradle", []string{"gradle"}
	case hasFile(ctx, "Gemfile"):
		return "bundler", []string{"bundler"}
	case hasFile(ctx, "composer.json"):
		return "composer", []string{"composer"}
	default:
		return "Unknown", nil
	}
}

func packageManagerSignals(ctx *projectContext, markers []struct {
	name  string
	files []string
}) []string {
	var signals []string
	for _, marker := range markers {
		if hasAnyFile(ctx, marker.files...) {
			signals = append(signals, marker.name)
		}
	}
	return signals
}

func contentContains(ctx *projectContext, filename, needle string) bool {
	content, ok := ctx.content[filename]
	return ok && strings.Contains(strings.ToLower(content), strings.ToLower(needle))
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
