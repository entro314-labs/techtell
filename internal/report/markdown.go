package report

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/entro314-labs/techtell/internal/analysis"
)

// GenerateMarkdown creates a markdown summary of the workspace analysis
func GenerateMarkdown(r *analysis.WorkspaceReport, rootPath string) string {
	var sb strings.Builder

	sb.WriteString("# TechTell Workspace Report\n\n")
	fmt.Fprintf(&sb, "**Scanned Root:** `%s`\n", rootPath)
	fmt.Fprintf(&sb, "**Date:** %s\n\n", time.Now().Format(time.RFC1123))

	// 1. Summary Stats
	sb.WriteString("## 📊 Executive Summary\n\n")
	fmt.Fprintf(&sb, "- **Total Projects Identified:** %d\n", r.TotalProjects)

	topLang, langC := getTop(r.LangCounts)
	fmt.Fprintf(&sb, "- **Dominant Language:** %s (%d projects)\n", topLang, langC)

	topStack, stackC := getTop(r.StackCounts)
	fmt.Fprintf(&sb, "- **Top Stack:** %s (%d projects)\n", topStack, stackC)

	sb.WriteString("\n---\n\n")

	// 2. DNA Breakdown
	sb.WriteString("## 🧬 Tech DNA\n\n")

	sb.WriteString("### Languages\n")
	sb.WriteString("| Language | Projects |\n|---|---|\n")
	for k, v := range r.LangCounts {
		fmt.Fprintf(&sb, "| %s | %d |\n", k, v)
	}
	sb.WriteString("\n")

	sb.WriteString("### Stacks\n")
	sb.WriteString("| Stack | Projects |\n|---|---|\n")
	for k, v := range r.StackCounts {
		fmt.Fprintf(&sb, "| %s | %d |\n", k, v)
	}
	sb.WriteString("\n")

	sb.WriteString("## 📂 Project Inventory\n\n")
	sb.WriteString("| Project Name | Stack | Language | Tags | Location |\n")
	sb.WriteString("|---|---|---|---|---|\n")

	for _, p := range r.Projects {
		cleanPath := strings.Replace(p.Path, rootPath, "", 1)
		cleanPath = strings.TrimPrefix(cleanPath, "/")
		if cleanPath == "" {
			cleanPath = "."
		}

		tags := strings.Join(p.Tags, ", ")
		fmt.Fprintf(&sb, "| **%s** | %s | %s | %s | `%s` |\n", p.Name, p.Stack, p.Language, tags, cleanPath)
	}

	return sb.String()
}

// SaveToFile writes content to a file
func SaveToFile(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

func getTop(m map[string]int) (string, int) {
	topK := "N/A"
	maxV := 0
	for k, v := range m {
		if v > maxV {
			maxV = v
			topK = k
		}
	}
	return topK, maxV
}
