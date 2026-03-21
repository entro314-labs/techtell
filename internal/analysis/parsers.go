package analysis

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

// ContentParser parses file content and returns a list of dependencies
type ContentParser func(content []byte) ([]string, error)

var Parsers = map[string]ContentParser{
	"package.json":     ParsePackageJSON,
	"go.mod":           ParseGoMod,
	"requirements.txt": ParseRequirementsTxt,
	"Cargo.toml":       ParseCargoToml,
	"build.gradle":     ParseGradle,
	"build.gradle.kts": ParseGradle,
}

// ParsePackageJSON parses Node.js package.json
func ParsePackageJSON(content []byte) ([]string, error) {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, err
	}

	deps := make([]string, 0, len(pkg.Dependencies)+len(pkg.DevDependencies))
	for k := range pkg.Dependencies {
		deps = append(deps, k)
	}
	for k := range pkg.DevDependencies {
		deps = append(deps, k)
	}
	return deps, nil
}

// ParseGoMod parses Go go.mod
func ParseGoMod(content []byte) ([]string, error) {
	var deps []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		if parts[0] == "require" {
			if len(parts) > 1 && parts[1] == "(" {
				inRequire = true
				continue
			}
			if len(parts) >= 2 {
				// Single line require: require github.com/foo/bar v1.0.0
				deps = append(deps, parts[1])
			}
		} else if parts[0] == ")" && inRequire {
			inRequire = false
		} else if inRequire {
			// Inside require block: github.com/foo/bar v1.0.0
			if len(parts) >= 1 {
				deps = append(deps, parts[0])
			}
		}
	}
	return deps, nil
}

// ParseRequirementsTxt parses Python requirements.txt
func ParseRequirementsTxt(content []byte) ([]string, error) {
	var deps []string
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Remove version specifiers (e.g., "flask==2.0.1" -> "flask")
		// Also handle "flask>=2.0" or "flask~=2.0"
		// Simple approach: split by typical version chars
		splitters := []string{"==", ">=", "<=", "~=", "!=", ";", " "}
		pkg := line
		for _, s := range splitters {
			if idx := strings.Index(pkg, s); idx != -1 {
				pkg = pkg[:idx]
			}
		}
		if pkg != "" {
			deps = append(deps, pkg)
		}
	}
	return deps, nil
}

// ParseCargoToml parses Rust Cargo.toml
func ParseCargoToml(content []byte) ([]string, error) {
	var deps []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inDependencies := false

	// Very naive TOML parser for dependencies
	// Looks for [dependencies] or [dev-dependencies] section
	// Then reads keys until next section

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.Trim(line, "[]")
			if section == "dependencies" || section == "dev-dependencies" || strings.HasSuffix(section, ".dependencies") {
				inDependencies = true
			} else {
				inDependencies = false
			}
			continue
		}

		if inDependencies {
			// parse "name = ..." or "name = { ... }"
			if idx := strings.Index(line, "="); idx != -1 {
				name := strings.TrimSpace(line[:idx])
				deps = append(deps, name)
			}
		}
	}
	return deps, nil
}

// ParseGradle parses build.gradle and build.gradle.kts
func ParseGradle(content []byte) ([]string, error) {
	var deps []string
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Naive Gradle parser
	// Looks for: implementation 'com.example:lib:1.0'
	// or: api("com.example:lib:1.0")
	// or: classpath "com.android.tools.build:gradle:7.0.0"

	keywords := []string{"implementation", "api", "compileOnly", "runtimeOnly", "testImplementation", "classpath", "kapt", "annotationProcessor"}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		for _, kw := range keywords {
			if strings.HasPrefix(line, kw) {
				// Extract string content: '...' or "..."
				start := strings.IndexAny(line, "'\"")
				if start != -1 {
					end := strings.LastIndexAny(line, "'\"")
					if end > start {
						depStr := line[start+1 : end]
						deps = append(deps, depStr)
					}
				}
			}
		}
	}
	return deps, nil
}
