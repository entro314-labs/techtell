package analysis

import (
	"path/filepath"
	"strings"
)

type MatchType string

const (
	MatchTypeFile       MatchType = "file"
	MatchTypeExtension  MatchType = "extension"
	MatchTypeContent    MatchType = "content"
	MatchTypeDependency MatchType = "dependency"
)

type TechSignature struct {
	Name        string
	Category    string // "Language", "Framework", "Package Manager", "Build Tool", "Tool", "Library", "DevOps", "Cloud"
	MatchType   MatchType
	Identifiers []string // filenames or extensions
}

var Signatures = []TechSignature{
	// Languages
	{Name: "Go", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".go"}},
	{Name: "Go", Category: "Language", MatchType: MatchTypeFile, Identifiers: []string{"go.mod"}},
	{Name: "JavaScript", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".js", ".jsx", ".mjs"}},
	{Name: "TypeScript", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".ts", ".tsx"}},
	{Name: "Python", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".py"}},
	{Name: "Python", Category: "Language", MatchType: MatchTypeFile, Identifiers: []string{"requirements.txt", "pyproject.toml", "Pipfile", "uv.lock"}},
	{Name: "Rust", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".rs"}},
	{Name: "Rust", Category: "Language", MatchType: MatchTypeFile, Identifiers: []string{"Cargo.toml"}},
	{Name: "Java", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".java"}},
	{Name: "Kotlin", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".kt", ".kts"}},
	{Name: "C++", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".cpp", ".cc", ".cxx", ".hpp"}},
	{Name: "C", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".c", ".h"}},
	{Name: "Ruby", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".rb"}},
	{Name: "Ruby", Category: "Language", MatchType: MatchTypeFile, Identifiers: []string{"Gemfile"}},
	{Name: "PHP", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".php"}},
	{Name: "HTML", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".html", ".htm"}},
	{Name: "CSS", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".css"}},
	{Name: "SASS/SCSS", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".sass", ".scss"}},
	{Name: "Lua", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".lua"}},
	{Name: "Shell", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".sh", ".bash", ".zsh"}},

	// Frameworks & Libraries (naive file-based detection)
	{Name: "React", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"react"}},
	{Name: "Next.js", Category: "Framework", MatchType: MatchTypeFile, Identifiers: []string{"next.config.js", "next.config.mjs", "next.config.ts"}},
	{Name: "Next.js", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"next"}},
	{Name: "Vue.js", Category: "Framework", MatchType: MatchTypeExtension, Identifiers: []string{".vue"}},
	{Name: "Vue.js", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"vue"}},
	{Name: "Svelte", Category: "Framework", MatchType: MatchTypeExtension, Identifiers: []string{".svelte"}},
	{Name: "Angular", Category: "Framework", MatchType: MatchTypeFile, Identifiers: []string{"angular.json"}},
	{Name: "Tailwind CSS", Category: "Framework", MatchType: MatchTypeFile, Identifiers: []string{"tailwind.config.js", "tailwind.config.ts"}},
	{Name: "Tailwind CSS", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"tailwindcss"}},
	{Name: "Django", Category: "Framework", MatchType: MatchTypeFile, Identifiers: []string{"manage.py"}}, // Heuristic
	{Name: "Django", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"django"}},
	{Name: "Flask", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"flask"}},
	{Name: "FastAPI", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"fastapi"}},
	{Name: "Rails", Category: "Framework", MatchType: MatchTypeFile, Identifiers: []string{"config/routes.rb"}}, // Tricky to match full path with just filename matchers, keeping simple for now
	{Name: "Gin", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"github.com/gin-gonic/gin"}},
	{Name: "Echo", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"github.com/labstack/echo/v4"}},
	{Name: "Fiber", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"github.com/gofiber/fiber/v2"}},
	{Name: "Cobra", Category: "Library", MatchType: MatchTypeDependency, Identifiers: []string{"github.com/spf13/cobra"}},
	{Name: "Bubbletea", Category: "Library", MatchType: MatchTypeDependency, Identifiers: []string{"github.com/charmbracelet/bubbletea"}},
	{Name: "Lipgloss", Category: "Library", MatchType: MatchTypeDependency, Identifiers: []string{"github.com/charmbracelet/lipgloss"}},

	// Rust / Tauri
	{Name: "Tauri", Category: "Framework", MatchType: MatchTypeFile, Identifiers: []string{"tauri.conf.json", "src-tauri/tauri.conf.json"}},
	{Name: "Tauri", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"tauri", "tauri-build"}},
	{Name: "Serde", Category: "Library", MatchType: MatchTypeDependency, Identifiers: []string{"serde"}},
	{Name: "Tokio", Category: "Library", MatchType: MatchTypeDependency, Identifiers: []string{"tokio"}},
	{Name: "Yew", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"yew"}},
	{Name: "Actix", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"actix-web"}},
	{Name: "Axum", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"axum"}},

	// Gradle / Java / Kotlin
	{Name: "Gradle", Category: "Build Tool", MatchType: MatchTypeFile, Identifiers: []string{"build.gradle", "build.gradle.kts", "gradlew"}},
	{Name: "Spring Boot", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"org.springframework.boot"}},
	{Name: "Ktor", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"ktor-server-core"}},
	{Name: "Android", Category: "Framework", MatchType: MatchTypeDependency, Identifiers: []string{"com.android.application", "com.android.library", "androidx.core"}},
	{Name: "Hibernate", Category: "Library", MatchType: MatchTypeDependency, Identifiers: []string{"hibernate-core"}},

	// Tools (Shell & Others)
	{Name: "Shell Script", Category: "Language", MatchType: MatchTypeExtension, Identifiers: []string{".sh", ".bash", ".zsh"}},
	{Name: "curl", Category: "Tool", MatchType: MatchTypeContent, Identifiers: []string{"curl"}},
	{Name: "wget", Category: "Tool", MatchType: MatchTypeContent, Identifiers: []string{"wget"}},
	{Name: "jq", Category: "Tool", MatchType: MatchTypeContent, Identifiers: []string{"jq"}},
	{Name: "Helm", Category: "Tool", MatchType: MatchTypeFile, Identifiers: []string{"Chart.yaml"}},
	{Name: "Kustomize", Category: "Tool", MatchType: MatchTypeFile, Identifiers: []string{"kustomization.yaml"}},
	{Name: "Terraform", Category: "Tool", MatchType: MatchTypeExtension, Identifiers: []string{".tf"}},

	// Tools & Configs
	{Name: "Docker", Category: "DevOps", MatchType: MatchTypeFile, Identifiers: []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml"}},
	{Name: "Git", Category: "Tool", MatchType: MatchTypeFile, Identifiers: []string{".gitignore"}},
	{Name: "GitHub Actions", Category: "DevOps", MatchType: MatchTypeFile, Identifiers: []string{".github"}},
	{Name: "Makefile", Category: "Build Tool", MatchType: MatchTypeFile, Identifiers: []string{"Makefile"}},
	{Name: "Vercel", Category: "Cloud", MatchType: MatchTypeFile, Identifiers: []string{"vercel.json"}},
	{Name: "Netlify", Category: "Cloud", MatchType: MatchTypeFile, Identifiers: []string{"netlify.toml"}},
	{Name: "Vite", Category: "Build Tool", MatchType: MatchTypeFile, Identifiers: []string{"vite.config.js", "vite.config.ts"}},
	{Name: "Vite", Category: "Build Tool", MatchType: MatchTypeDependency, Identifiers: []string{"vite"}},
}

// Matches checks if a given filename (base name) matches the signature
func (s TechSignature) Matches(filename string) bool {
	switch s.MatchType {
	case MatchTypeFile:
		normFile := filepath.ToSlash(filename)
		base := filepath.Base(normFile)
		for _, id := range s.Identifiers {
			normID := filepath.ToSlash(id)
			if strings.EqualFold(base, normID) || strings.EqualFold(normFile, normID) {
				return true
			}
		}
	case MatchTypeExtension:
		ext := filepath.Ext(filename)
		for _, id := range s.Identifiers {
			if strings.EqualFold(ext, id) {
				return true
			}
		}
	case MatchTypeDependency:
		// Not matched by filename
		return false
	}
	return false
}
