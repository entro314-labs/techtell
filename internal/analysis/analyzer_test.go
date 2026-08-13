package analysis

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func contains(slice []string, want string) bool {
	for _, v := range slice {
		if v == want {
			return true
		}
	}
	return false
}

func TestInspectProject_DoesNotFalsePositiveOnSubstringDependency(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "node-app")

	writeFile(t, filepath.Join(project, "package.json"), `{"dependencies":{"next-auth":"5.0.0"}}`)
	writeFile(t, filepath.Join(project, "index.js"), `console.log("ok")`)

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	if p.Stack != "Node.js Project" {
		t.Fatalf("expected stack Node.js Project, got %q", p.Stack)
	}
	if contains(p.Tags, "Next.js") {
		t.Fatalf("unexpected Next.js tag: %v", p.Tags)
	}
}

func TestInspectProject_NextOverridesReact(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "next-app")

	writeFile(t, filepath.Join(project, "package.json"), `{"dependencies":{"next":"15.0.0","react":"19.0.0"}}`)
	writeFile(t, filepath.Join(project, "next.config.js"), `module.exports = {}`)
	writeFile(t, filepath.Join(project, "page.tsx"), `export default function Page(){return null}`)

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	if p.Stack != "Next.js App" {
		t.Fatalf("expected stack Next.js App, got %q (tags=%v)", p.Stack, p.Tags)
	}
}

func TestInspectProject_GoLanguageAndPackageManager(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "go-app")

	writeFile(t, filepath.Join(project, "go.mod"), "module example.com/go-app\n\ngo 1.26\n")
	writeFile(t, filepath.Join(project, "main.go"), "package main\nfunc main(){}\n")

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	if p.Language != "Go" {
		t.Fatalf("expected language Go, got %q", p.Language)
	}
	if p.PackageManager != "go mod" {
		t.Fatalf("expected package manager go mod, got %q", p.PackageManager)
	}
}

func TestInspectProject_BuildToolTagsApplied(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "vite-app")

	writeFile(t, filepath.Join(project, "package.json"), `{"name":"vite-app"}`)
	writeFile(t, filepath.Join(project, "vite.config.ts"), `export default {}`)
	writeFile(t, filepath.Join(project, "package-lock.json"), `{}`)

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	if !contains(p.Tags, "Vite") {
		t.Fatalf("expected Vite tag, got %v", p.Tags)
	}
}

func TestDiscoverProjects_FindsNestedProjectsEvenWithRootMarkers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Dockerfile"), "FROM alpine:latest\n")
	writeFile(t, filepath.Join(root, "service-a", "go.mod"), "module example.com/a\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "service-a", "main.go"), "package main\nfunc main(){}\n")

	a := NewAnalyzer(root, nil)
	paths, err := a.DiscoverProjects(context.Background(), nil)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	rel := make([]string, 0, len(paths))
	for _, p := range paths {
		r, _ := filepath.Rel(root, p)
		rel = append(rel, r)
	}

	if !contains(rel, ".") {
		t.Fatalf("expected root project in discovery, got %v", rel)
	}
	if !contains(rel, "service-a") {
		t.Fatalf("expected nested service-a project in discovery, got %v", rel)
	}
}

func TestInspectProject_RailsSubpathSignature(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "rails-app")
	writeFile(t, filepath.Join(project, "Gemfile"), "source 'https://rubygems.org'\n")
	writeFile(t, filepath.Join(project, "config", "routes.rb"), "Rails.application.routes.draw do\nend\n")

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	if p.Stack != "Rails" {
		t.Fatalf("expected Rails stack, got %q (tags=%v)", p.Stack, p.Tags)
	}
}

func TestInspectProject_ContentSignature(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "scripted")
	writeFile(t, filepath.Join(project, "go.mod"), "module example.com/scripted\ngo 1.26\n")
	writeFile(t, filepath.Join(project, "script.sh"), "#!/usr/bin/env bash\ncurl https://example.com\n")

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	if !contains(p.Tags, "curl") {
		t.Fatalf("expected curl tag, got %v", p.Tags)
	}
}

func TestInspectProject_ContentSignatureDoesNotScanManifestText(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "curl-wrapper")
	writeFile(t, filepath.Join(project, "package.json"), `{"name":"curl-wrapper","dependencies":{"jq":"1.0.0"}}`)

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if contains(p.Tags, "curl") || contains(p.Tags, "jq") {
		t.Fatalf("manifest text must not be interpreted as shell-tool usage: %v", p.Tags)
	}
}

func TestInspectProject_UsesNestedSourceFilesWithoutAssumingReact(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "solid-app")

	writeFile(t, filepath.Join(project, "package.json"), `{"packageManager":"pnpm@10.0.0","dependencies":{"solid-js":"1.0.0"}}`)
	writeFile(t, filepath.Join(project, "src", "app.tsx"), `export const App = () => null`)

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if p.Language != "TypeScript" {
		t.Fatalf("expected nested TypeScript source to be detected, got %q", p.Language)
	}
	if p.Stack != "Node.js Project" {
		t.Fatalf("expected TSX without React dependency to remain Node.js Project, got %q", p.Stack)
	}
	if p.PackageManager != "pnpm" {
		t.Fatalf("expected packageManager metadata to win, got %q", p.PackageManager)
	}
}

func TestInspectProject_PEP621DependenciesAndUVPackageManager(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "python-api")

	writeFile(t, filepath.Join(project, "pyproject.toml"), "[project]\nname = \"python-api\"\ndependencies = [\"fastapi[standard]>=0.100\"]\n")
	writeFile(t, filepath.Join(project, "uv.lock"), "version = 1\n")

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if p.Stack != "FastAPI" {
		t.Fatalf("expected FastAPI from PEP 621 dependencies, got %q", p.Stack)
	}
	if p.PackageManager != "uv" {
		t.Fatalf("expected uv package manager, got %q", p.PackageManager)
	}
}

func TestInspectProject_MiseIsToolNotPackageManager(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "go-app")
	writeFile(t, filepath.Join(project, "go.mod"), "module example.com/go-app\ngo 1.26\n")
	writeFile(t, filepath.Join(project, "mise.toml"), "[tools]\ngo = \"1.26\"\n")

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if p.PackageManager != "go mod" {
		t.Fatalf("expected go mod package manager, got %q", p.PackageManager)
	}
	if !contains(p.Tags, "mise") {
		t.Fatalf("expected mise tool tag, got %v", p.Tags)
	}
}

func TestInspectProject_ConflictingPackageManagerSignalsWarn(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "node-app")
	writeFile(t, filepath.Join(project, "package.json"), `{"packageManager":"pnpm@10.0.0"}`)
	writeFile(t, filepath.Join(project, "package-lock.json"), `{}`)
	writeFile(t, filepath.Join(project, "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if p.PackageManager != "pnpm" {
		t.Fatalf("expected declared pnpm selection, got %q", p.PackageManager)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "conflicting package manager signals") {
		t.Fatalf("expected conflict warning, got %v", warnings)
	}
}

func TestInspectProject_MalformedManifestProducesWarning(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "broken-node")
	writeFile(t, filepath.Join(project, "package.json"), `{not-json}`)

	_, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "parsing dependency manifest") {
		t.Fatalf("expected manifest parse warning, got %v", warnings)
	}
}

func TestInspectProject_GenericYAMLIsNotKubernetes(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "go-app")
	writeFile(t, filepath.Join(project, "go.mod"), "module example.com/go-app\ngo 1.26\n")
	writeFile(t, filepath.Join(project, "config.yaml"), "theme: dark\n")

	p, _ := NewAnalyzer(root, nil).InspectProject(project)
	if contains(p.Tags, "Kubernetes") {
		t.Fatalf("generic YAML must not imply Kubernetes: %v", p.Tags)
	}
}

func TestInspectProject_IndirectGoDependencyDoesNotSelectFramework(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "go-app")
	writeFile(t, filepath.Join(project, "go.mod"), "module example.com/go-app\ngo 1.26\nrequire github.com/gin-gonic/gin v1.0.0 // indirect\n")

	p, _ := NewAnalyzer(root, nil).InspectProject(project)
	if p.Stack == "Gin" || contains(p.Tags, "Gin") {
		t.Fatalf("indirect dependency must not define the project framework: %+v", p)
	}
}

func TestInspectProject_AndroidGradlePluginDetected(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "android-app")
	writeFile(t, filepath.Join(project, "build.gradle.kts"), `plugins {
    id("com.android.application") version "8.0.0"
}
`)
	writeFile(t, filepath.Join(project, "src", "main", "MainActivity.kt"), "class MainActivity\n")

	p, warnings := NewAnalyzer(root, nil).InspectProject(project)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if p.Stack != "Android" {
		t.Fatalf("expected Android stack, got %q (tags=%v)", p.Stack, p.Tags)
	}
	if p.Language != "Kotlin" {
		t.Fatalf("expected Kotlin language, got %q", p.Language)
	}
}
