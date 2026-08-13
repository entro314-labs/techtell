package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseCLIAction(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    cliAction
		wantErr bool
	}{
		{name: "run default", args: nil, want: actionRun},
		{name: "version long", args: []string{"--version"}, want: actionVersion},
		{name: "version short", args: []string{"-v"}, want: actionVersion},
		{name: "help long", args: []string{"--help"}, want: actionHelp},
		{name: "help short", args: []string{"-h"}, want: actionHelp},
		{name: "help alt", args: []string{"-help"}, want: actionHelp},
		{name: "unknown", args: []string{"--versoin"}, want: actionRun, wantErr: true},
		{name: "multiple", args: []string{"--help", "--version"}, want: actionRun, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCLIAction(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseCLIAction(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("parseCLIAction(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildVersion(t *testing.T) {
	origVersion := version
	origCommit := commit
	origDate := date
	origBuiltBy := builtBy
	t.Cleanup(func() {
		version = origVersion
		commit = origCommit
		date = origDate
		builtBy = origBuiltBy
	})

	version = "dev"
	if got := buildVersion(); got != "dev (built from source)" {
		t.Fatalf("unexpected dev version string: %q", got)
	}

	version = "v1.2.3"
	commit = "abc123"
	date = "2026-01-01"
	builtBy = "ci"
	got := buildVersion()
	for _, part := range []string{"v1.2.3", "abc123", "2026-01-01", "ci"} {
		if !strings.Contains(got, part) {
			t.Fatalf("expected %q in buildVersion output %q", part, got)
		}
	}
}

func TestPrintHelp(t *testing.T) {
	var buf bytes.Buffer
	if err := printHelp(&buf); err != nil {
		t.Fatalf("printHelp failed: %v", err)
	}
	out := buf.String()
	for _, snippet := range []string{"Usage:", "--version", "--help", "Start interactive TUI"} {
		if !strings.Contains(out, snippet) {
			t.Fatalf("missing %q in help output: %q", snippet, out)
		}
	}
}
