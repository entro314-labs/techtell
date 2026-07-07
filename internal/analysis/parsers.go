package analysis

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"regexp"
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
	"pyproject.toml":   ParsePyProjectToml,
	"Pipfile":          ParsePipfile,
	"uv.lock":          ParseUVLock,
	"Gemfile":          ParseGemfile,
	"composer.json":    ParseComposerJSON,
	"pom.xml":          ParsePomXML,
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
				deps = append(deps, parts[1])
			}
		} else if parts[0] == ")" && inRequire {
			inRequire = false
		} else if inRequire {
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

	keywords := []string{"implementation", "api", "compileOnly", "runtimeOnly", "testImplementation", "classpath", "kapt", "annotationProcessor"}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		for _, kw := range keywords {
			if !strings.HasPrefix(line, kw) {
				continue
			}
			start := strings.IndexAny(line, "'\"")
			if start == -1 {
				continue
			}
			end := strings.LastIndexAny(line, "'\"")
			if end <= start {
				continue
			}
			depStr := line[start+1 : end]
			deps = append(deps, depStr)
			parts := strings.Split(depStr, ":")
			if len(parts) >= 2 {
				deps = append(deps, parts[1])
			}
		}
	}
	return deps, nil
}

// ParsePyProjectToml parses pyproject.toml dependency entries.
func ParsePyProjectToml(content []byte) ([]string, error) {
	var deps []string
	scanner := bufio.NewScanner(bytes.NewReader(content))

	inDeps := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.Trim(line, "[]")
			inDeps = section == "project.dependencies" ||
				section == "project.optional-dependencies" ||
				section == "tool.poetry.dependencies" ||
				section == "tool.poetry.group.dev.dependencies" ||
				section == "dependency-groups"
			continue
		}
		if !inDeps {
			continue
		}
		if idx := strings.Index(line, "="); idx != -1 {
			name := strings.TrimSpace(line[:idx])
			name = strings.Trim(name, "\"'")
			if name != "" && name != "python" {
				deps = append(deps, name)
			}
			continue
		}
		line = strings.Trim(line, ",")
		line = strings.Trim(line, "\"'")
		if line == "" {
			continue
		}
		splitters := []string{"==", ">=", "<=", "~=", "!=", ";", " "}
		name := line
		for _, s := range splitters {
			if idx := strings.Index(name, s); idx != -1 {
				name = name[:idx]
			}
		}
		name = strings.TrimSpace(name)
		if name != "" {
			deps = append(deps, name)
		}
	}
	return deps, nil
}

// ParsePipfile parses Pipfile package sections.
func ParsePipfile(content []byte) ([]string, error) {
	var deps []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inDeps := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.Trim(line, "[]")
			inDeps = section == "packages" || section == "dev-packages"
			continue
		}
		if !inDeps {
			continue
		}
		if idx := strings.Index(line, "="); idx != -1 {
			name := strings.TrimSpace(line[:idx])
			name = strings.Trim(name, "\"'")
			if name != "" {
				deps = append(deps, name)
			}
		}
	}
	return deps, nil
}

// ParseUVLock parses uv.lock package entries.
func ParseUVLock(content []byte) ([]string, error) {
	var deps []string
	re := regexp.MustCompile(`(?m)^name\s*=\s*["']([^"']+)["']`)
	for _, match := range re.FindAllSubmatch(content, -1) {
		if len(match) > 1 {
			deps = append(deps, string(match[1]))
		}
	}
	return deps, nil
}

// ParseGemfile parses gem declarations in Gemfile.
func ParseGemfile(content []byte) ([]string, error) {
	var deps []string
	re := regexp.MustCompile(`(?m)^\s*gem\s+["']([^"']+)["']`)
	for _, match := range re.FindAllSubmatch(content, -1) {
		if len(match) > 1 {
			deps = append(deps, string(match[1]))
		}
	}
	return deps, nil
}

// ParseComposerJSON parses PHP composer.json dependencies.
func ParseComposerJSON(content []byte) ([]string, error) {
	var c struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if err := json.Unmarshal(content, &c); err != nil {
		return nil, err
	}
	deps := make([]string, 0, len(c.Require)+len(c.RequireDev))
	for k := range c.Require {
		deps = append(deps, k)
	}
	for k := range c.RequireDev {
		deps = append(deps, k)
	}
	return deps, nil
}

// ParsePomXML parses Maven pom.xml dependency identifiers.
func ParsePomXML(content []byte) ([]string, error) {
	type dependency struct {
		ArtifactID string `xml:"artifactId"`
		GroupID    string `xml:"groupId"`
	}
	type pom struct {
		Dependencies []dependency `xml:"dependencies>dependency"`
	}
	var p pom
	if err := xml.Unmarshal(content, &p); err != nil {
		return nil, err
	}
	deps := make([]string, 0, len(p.Dependencies)*2)
	for _, d := range p.Dependencies {
		if d.ArtifactID != "" {
			deps = append(deps, d.ArtifactID)
		}
		if d.GroupID != "" {
			deps = append(deps, d.GroupID)
		}
	}
	return deps, nil
}
