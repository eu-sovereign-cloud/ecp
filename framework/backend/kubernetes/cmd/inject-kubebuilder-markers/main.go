// inject-kubebuilder-markers scans generated Go files for x-cel-* and
// x-kubebuilder-validation-* struct tags and injects the corresponding
// // +kubebuilder:validation:* marker comments so that controller-gen
// produces CRDs with the correct x-kubernetes-validations and constraint
// properties (MaxItems, MaxLength, etc.).
//
// Usage:
//
//	go run ./cmd/inject-kubebuilder-markers <dir>
//
// The tool modifies files in place. It is idempotent — running it twice
// produces the same output.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: inject-kubebuilder-markers <dir>")
		os.Exit(1)
	}
	dir := os.Args[1]

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	total := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		n, err := processFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error processing %s: %v\n", path, err)
			os.Exit(1)
		}
		if n > 0 {
			fmt.Printf("[KB] %s: injected %d marker(s)\n", path, n)
			total += n
		}
	}
	fmt.Printf("inject-kubebuilder-markers: %d marker(s) injected\n", total)
}

// ── CEL marker types ──────────────────────────────────────────────────────────

// celRule holds a parsed CEL validation rule extracted from struct tags.
type celRule struct {
	index           int
	rule            string
	message         string
	optionalOldSelf bool
}

// tagRe matches x-cel-rule-N:"..." and x-cel-message-N:"..." struct tags.
var tagRe = regexp.MustCompile(`(x-cel-(?:rule|message)-\d+):"((?:[^"\\]|\\.)*)"`)

// extractCELRules parses x-cel-rule-N / x-cel-message-N tags from a struct
// field line and returns them sorted by index.
func extractCELRules(line string) []celRule {
	matches := tagRe.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}

	ruleMap := map[int]*celRule{}

	for _, m := range matches {
		key := m[1]
		val := m[2]
		val = strings.ReplaceAll(val, `\"`, `"`)

		parts := strings.SplitN(key, "-", 4) // x-cel-rule-0 → [x, cel, rule, 0]
		if len(parts) != 4 {
			continue
		}
		kind := parts[2]
		idx, err := strconv.Atoi(parts[3])
		if err != nil {
			continue
		}

		if ruleMap[idx] == nil {
			ruleMap[idx] = &celRule{index: idx}
		}
		switch kind {
		case "rule":
			ruleMap[idx].rule = val
			if strings.Contains(val, "oldSelf.hasValue()") {
				ruleMap[idx].optionalOldSelf = true
			}
		case "message":
			ruleMap[idx].message = val
		}
	}

	rules := make([]celRule, 0, len(ruleMap))
	for _, r := range ruleMap {
		if r.rule == "" {
			continue
		}
		rules = append(rules, *r)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].index < rules[j].index })
	return rules
}

// formatCELMarker produces a kubebuilder XValidation marker string.
func formatCELMarker(r celRule) string {
	var b strings.Builder
	b.WriteString(`// +kubebuilder:validation:XValidation:rule="`)
	b.WriteString(r.rule)
	b.WriteString(`"`)
	if r.message != "" {
		b.WriteString(`,message="`)
		b.WriteString(r.message)
		b.WriteString(`"`)
	}
	if r.optionalOldSelf {
		b.WriteString(`,optionalOldSelf=true`)
	}
	return b.String()
}

// ── Validation marker types ───────────────────────────────────────────────────

// kbTagRe matches x-kubebuilder-validation-<name>:"<value>" struct tags.
// Values may be numeric (e.g., "64") or string (e.g., "ingress;egress" for enums).
var kbTagRe = regexp.MustCompile(`x-kubebuilder-validation-([a-z-]+):"([^"]*)"`)

// kbTagToMarker maps the kebab-case tag suffix to its kubebuilder marker name.
var kbTagToMarker = map[string]string{
	"max-items":        "MaxItems",
	"max-length":       "MaxLength",
	"min-items":        "MinItems",
	"min-length":       "MinLength",
	"minimum":          "Minimum",
	"maximum":          "Maximum",
	"max-properties":   "MaxProperties",
	"enum":             "Enum",
	"items-max-length": "items:MaxLength",
	"items-min-length": "items:MinLength",
	"items-maximum":    "items:Maximum",
	"items-minimum":    "items:Minimum",
	"pattern":          "Pattern",
}

// kbMarker holds a parsed kubebuilder validation constraint extracted from tags.
type kbMarker struct {
	name  string // e.g. "MaxItems"
	value string // e.g. "64"
}

// extractKBMarkers parses x-kubebuilder-validation-* tags from a struct field
// line and returns the corresponding markers in deterministic order.
func extractKBMarkers(line string) []kbMarker {
	matches := kbTagRe.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}

	var markers []kbMarker //nolint:prealloc
	for _, m := range matches {
		suffix := m[1] // e.g. "max-items"
		value := m[2]  // e.g. "64"
		name, ok := kbTagToMarker[suffix]
		if !ok {
			continue
		}
		markers = append(markers, kbMarker{name: name, value: value})
	}

	// Stable order: sort by marker name.
	sort.Slice(markers, func(i, j int) bool { return markers[i].name < markers[j].name })
	return markers
}

// formatKBMarker produces a +kubebuilder:validation:<Name>=<value> string.
// Pattern values are backtick-quoted as required by controller-gen.
func formatKBMarker(m kbMarker) string {
	if m.name == "Pattern" {
		return fmt.Sprintf("// +kubebuilder:validation:%s=`%s`", m.name, m.value)
	}
	return fmt.Sprintf("// +kubebuilder:validation:%s=%s", m.name, m.value)
}

// ── Default marker type ───────────────────────────────────────────────────────

// kbDefaultTagRe matches x-kubebuilder-default:"<value>" struct tags.
var kbDefaultTagRe = regexp.MustCompile(`x-kubebuilder-default:"([^"]*)"`)

// extractKBDefault parses an x-kubebuilder-default tag from a struct field line.
func extractKBDefault(line string) (string, bool) {
	m := kbDefaultTagRe.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// formatDefaultMarker produces a +kubebuilder:default=<value> string.
// Booleans and numbers are unquoted; all other values are double-quoted.
func formatDefaultMarker(value string) string {
	if value == "true" || value == "false" {
		return "// +kubebuilder:default=" + value
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "// +kubebuilder:default=" + value
	}
	return fmt.Sprintf(`// +kubebuilder:default="%s"`, value)
}

// ── File processing ───────────────────────────────────────────────────────────

// kbValidationPrefix is the common prefix for all injected kubebuilder validation markers.
// Used for idempotency: any line matching this prefix is stripped before re-injecting.
const kbValidationPrefix = "// +kubebuilder:validation:"

// kbDefaultPrefix is the prefix for injected kubebuilder default markers.
const kbDefaultPrefix = "// +kubebuilder:default"

// processFile reads a Go file, injects kubebuilder markers above fields that
// have x-cel-* or x-kubebuilder-validation-* struct tags, and writes the file
// back. Returns the number of markers injected.
func processFile(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path) // #nosec:G304 - path comes from os.ReadDir //nolint:gosec
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	injected := 0

	for i, line := range lines {
		// Preserve blank lines immediately before an already-injected marker
		// (idempotency: don't re-process them on second run).
		if strings.TrimSpace(line) == "" && i+1 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(next, kbValidationPrefix) || strings.HasPrefix(next, kbDefaultPrefix) {
				out = append(out, line)
				continue
			}
		}

		hasCEL := strings.Contains(line, "x-cel-rule-")
		hasKB := strings.Contains(line, "x-kubebuilder-validation-")
		hasDefault := strings.Contains(line, "x-kubebuilder-default")

		if !hasCEL && !hasKB && !hasDefault {
			out = append(out, line)
			continue
		}

		celRules := extractCELRules(line)
		kbMarkers := extractKBMarkers(line)
		defaultVal, hasDefaultVal := extractKBDefault(line)

		if len(celRules) == 0 && len(kbMarkers) == 0 && !hasDefaultVal {
			out = append(out, line)
			continue
		}

		indent := leadingWhitespace(line)

		// Remove any previously injected markers immediately above this field.
		out = removePriorMarkers(out)

		// Inject default marker first.
		if hasDefaultVal {
			out = append(out, indent+formatDefaultMarker(defaultVal))
			injected++
		}

		// Inject kubebuilder validation markers (MaxItems, MaxLength, …).
		//
		// controller-gen validates MaxLength/MinLength by checking schema.Type==string.
		// For fields typed via a cross-module string alias (e.g. schemav1.Zone = string)
		// the type resolves to "" in controller-gen's schema pass, causing a false
		// validation failure. The Type marker has ApplyPriorityDefault-1, so it is
		// always applied before other markers regardless of source order. Emit
		// Type=string / items:Type=string whenever length constraints are present so
		// the schema type is set explicitly before MaxLength/MinLength run their check.
		needTypeString := false
		needItemsTypeString := false
		for _, m := range kbMarkers {
			if m.name == "MaxLength" || m.name == "MinLength" {
				needTypeString = true
			}
			if m.name == "items:MaxLength" || m.name == "items:MinLength" {
				needItemsTypeString = true
			}
		}
		// Also check whether Type markers are already present (idempotency guard).
		for _, m := range kbMarkers {
			if m.name == "Type" {
				needTypeString = false
			}
			if m.name == "items:Type" {
				needItemsTypeString = false
			}
		}
		if needTypeString {
			out = append(out, indent+formatKBMarker(kbMarker{name: "Type", value: "string"}))
			injected++
		}
		if needItemsTypeString {
			out = append(out, indent+formatKBMarker(kbMarker{name: "items:Type", value: "string"}))
			injected++
		}
		for _, m := range kbMarkers {
			out = append(out, indent+formatKBMarker(m))
			injected++
		}

		// Inject CEL markers after.
		for _, r := range celRules {
			out = append(out, indent+formatCELMarker(r))
			injected++
		}

		out = append(out, line)
	}

	if injected == 0 {
		return 0, nil
	}

	return injected, os.WriteFile(path, []byte(strings.Join(out, "\n")), info.Mode()) //nolint:gosec
}

// removePriorMarkers strips any trailing kubebuilder marker lines from the
// output buffer (used for idempotency). Covers both validation and default markers.
func removePriorMarkers(out []string) []string {
	for len(out) > 0 {
		last := strings.TrimSpace(out[len(out)-1])
		if strings.HasPrefix(last, kbValidationPrefix) || strings.HasPrefix(last, kbDefaultPrefix) {
			out = out[:len(out)-1]
			continue
		}
		break
	}
	return out
}

// leadingWhitespace returns the whitespace prefix of a line.
func leadingWhitespace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}
