package analysis

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
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
			if len(parts) >= 2 && !strings.Contains(line, "// indirect") {
				deps = append(deps, parts[1])
			}
		} else if parts[0] == ")" && inRequire {
			inRequire = false
		} else if inRequire && !strings.Contains(line, "// indirect") {
			if len(parts) >= 1 {
				deps = append(deps, parts[0])
			}
		}
	}
	return deps, scanner.Err()
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
		pkg := parsePythonRequirementName(line)
		if pkg != "" {
			deps = append(deps, pkg)
		}
	}
	return deps, scanner.Err()
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
	return deps, scanner.Err()
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
			depStr, ok := firstQuotedValue(line)
			if !ok {
				continue
			}
			deps = append(deps, depStr)
			parts := strings.Split(depStr, ":")
			if len(parts) >= 2 {
				deps = append(deps, parts[0])
				deps = append(deps, parts[1])
			}
		}

		if strings.HasPrefix(line, "id(") || strings.HasPrefix(line, "id ") {
			if pluginID, ok := firstQuotedValue(line); ok {
				deps = append(deps, pluginID)
			}
		}
	}
	return deps, scanner.Err()
}

func firstQuotedValue(line string) (string, bool) {
	start := strings.IndexAny(line, "'\"")
	if start < 0 {
		return "", false
	}
	quote := line[start]
	remainder := line[start+1:]
	end := strings.IndexByte(remainder, quote)
	if end < 0 {
		return "", false
	}
	return remainder[:end], true
}

// ParsePyProjectToml parses pyproject.toml dependency entries.
func ParsePyProjectToml(content []byte) ([]string, error) {
	var document map[string]any
	if err := toml.Unmarshal(content, &document); err != nil {
		return nil, err
	}

	deps := make([]string, 0)
	project := nestedMap(document, "project")
	appendRequirementList(&deps, project["dependencies"])
	appendRequirementGroups(&deps, project["optional-dependencies"])
	appendRequirementGroups(&deps, document["dependency-groups"])

	poetry := nestedMap(document, "tool", "poetry")
	appendDependencyKeys(&deps, poetry["dependencies"])
	if groups, ok := poetry["group"].(map[string]any); ok {
		for _, value := range groups {
			if group, ok := value.(map[string]any); ok {
				appendDependencyKeys(&deps, group["dependencies"])
			}
		}
	}

	return uniqueStrings(deps), nil
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
	return deps, scanner.Err()
}

func parsePythonRequirementName(spec string) string {
	name := strings.TrimSpace(spec)
	if name == "" || strings.HasPrefix(name, "-") {
		return ""
	}
	for _, delimiter := range []string{";", " @ ", "@", "[", "~=", "==", "!=", ">=", "<=", ">", "<", " "} {
		if idx := strings.Index(name, delimiter); idx >= 0 {
			name = name[:idx]
		}
	}
	return strings.TrimSpace(strings.Trim(name, "\"'"))
}

func nestedMap(root map[string]any, path ...string) map[string]any {
	current := root
	for _, part := range path {
		next, ok := current[part].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		current = next
	}
	return current
}

func appendRequirementList(deps *[]string, value any) {
	values, ok := value.([]any)
	if !ok {
		return
	}
	for _, item := range values {
		if spec, ok := item.(string); ok {
			if name := parsePythonRequirementName(spec); name != "" {
				*deps = append(*deps, name)
			}
		}
	}
}

func appendRequirementGroups(deps *[]string, value any) {
	groups, ok := value.(map[string]any)
	if !ok {
		return
	}
	for _, group := range groups {
		appendRequirementList(deps, group)
	}
}

func appendDependencyKeys(deps *[]string, value any) {
	dependencies, ok := value.(map[string]any)
	if !ok {
		return
	}
	for name := range dependencies {
		if name != "python" {
			*deps = append(*deps, name)
		}
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
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
