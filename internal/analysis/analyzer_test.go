package analysis

import (
	"context"
	"os"
	"path/filepath"
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

	p := NewAnalyzer(root, nil).InspectProject(project)

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

	p := NewAnalyzer(root, nil).InspectProject(project)

	if p.Stack != "Next.js App" {
		t.Fatalf("expected stack Next.js App, got %q (tags=%v)", p.Stack, p.Tags)
	}
}

func TestInspectProject_GoLanguageAndPackageManager(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "go-app")

	writeFile(t, filepath.Join(project, "go.mod"), "module example.com/go-app\n\ngo 1.26\n")
	writeFile(t, filepath.Join(project, "main.go"), "package main\nfunc main(){}\n")

	p := NewAnalyzer(root, nil).InspectProject(project)

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

	p := NewAnalyzer(root, nil).InspectProject(project)

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

	p := NewAnalyzer(root, nil).InspectProject(project)

	if p.Stack != "Rails" {
		t.Fatalf("expected Rails stack, got %q (tags=%v)", p.Stack, p.Tags)
	}
}

func TestInspectProject_ContentSignature(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "scripted")
	writeFile(t, filepath.Join(project, "go.mod"), "module example.com/scripted\ngo 1.26\n")
	writeFile(t, filepath.Join(project, "script.sh"), "#!/usr/bin/env bash\ncurl https://example.com\n")

	p := NewAnalyzer(root, nil).InspectProject(project)

	if !contains(p.Tags, "curl") {
		t.Fatalf("expected curl tag, got %v", p.Tags)
	}
}
