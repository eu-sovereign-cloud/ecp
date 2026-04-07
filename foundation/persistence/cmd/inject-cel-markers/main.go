// inject-cel-markers scans generated Go files for x-cel-* struct tags and
// injects // +kubebuilder:validation:XValidation marker comments so that
// controller-gen produces CRDs with x-kubernetes-validations natively.
//
// Usage:
//
//	go run ./cmd/inject-cel-markers <dir>
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
		fmt.Fprintln(os.Stderr, "usage: inject-cel-markers <dir>")
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
			fmt.Printf("[CEL] %s: injected %d marker(s)\n", path, n)
			total += n
		}
	}
	fmt.Printf("inject-cel-markers: %d marker(s) injected\n", total)
}

// celRule holds a parsed CEL validation rule extracted from struct tags.
type celRule struct {
	index           int
	rule            string
	message         string
	optionalOldSelf bool
}

// markerPrefix is what we look for to detect already-injected markers.
const markerPrefix = "// +kubebuilder:validation:XValidation:"

// processFile reads a Go file, injects kubebuilder XValidation markers above
// fields that have x-cel-* struct tags, and writes the file back. Returns the
// number of markers injected.
func processFile(path string) (int, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from CLI arg
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	var out []string
	injected := 0

	for i, line := range lines {
		// Skip lines that are already markers (for idempotency).
		if strings.TrimSpace(line) == "" && i+1 < len(lines) && strings.Contains(strings.TrimSpace(lines[i+1]), markerPrefix) {
			out = append(out, line)
			continue
		}

		if !strings.Contains(line, "x-cel-rule-") {
			out = append(out, line)
			continue
		}

		rules := extractCELRules(line)
		if len(rules) == 0 {
			out = append(out, line)
			continue
		}

		// Detect leading indentation of the field line.
		indent := leadingWhitespace(line)

		// Remove any previously injected markers immediately above this field
		// (for idempotency on re-runs).
		out = removePriorMarkers(out)

		// Inject marker comments.
		for _, r := range rules {
			marker := formatMarker(r)
			out = append(out, indent+marker)
			injected++
		}

		out = append(out, line)
	}

	if injected == 0 {
		return 0, nil
	}

	return injected, os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644) //nolint:gosec
}

// tagRe matches a single struct tag key:"value" pair, handling escaped quotes
// in values. It captures the key and value groups.
var tagRe = regexp.MustCompile(`(x-cel-(?:rule|message)-\d+):"((?:[^"\\]|\\.)*)"`)

// extractCELRules parses x-cel-rule-N and x-cel-message-N tags from a struct
// field line and returns sorted rules.
func extractCELRules(line string) []celRule {
	matches := tagRe.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}

	ruleMap := map[int]*celRule{}

	for _, m := range matches {
		key := m[1]
		val := m[2]
		// Unescape backslash-escaped quotes that may appear in tag values.
		val = strings.ReplaceAll(val, `\"`, `"`)

		parts := strings.SplitN(key, "-", 4) // x-cel-rule-0 → [x, cel, rule, 0]
		if len(parts) != 4 {
			continue
		}
		kind := parts[2] // "rule" or "message"
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
			if strings.Contains(val, "oldSelf") {
				ruleMap[idx].optionalOldSelf = true
			}
		case "message":
			ruleMap[idx].message = val
		}
	}

	// Collect and sort by index.
	var rules []celRule
	for _, r := range ruleMap {
		if r.rule == "" {
			continue // skip incomplete entries
		}
		rules = append(rules, *r)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].index < rules[j].index })
	return rules
}

// formatMarker produces a kubebuilder XValidation marker string.
func formatMarker(r celRule) string {
	var b strings.Builder
	b.WriteString(markerPrefix)
	b.WriteString(`rule="`)
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

// removePriorMarkers strips any trailing XValidation marker lines from the
// output buffer (used for idempotency).
func removePriorMarkers(out []string) []string {
	for len(out) > 0 {
		last := strings.TrimSpace(out[len(out)-1])
		if strings.HasPrefix(last, markerPrefix) {
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
