package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gobwas/glob"
)

var (
	labelExpr   = regexp.MustCompile(`^\s*([a-zA-Z0-9_.*/-]+)\s*(==|!=|>=|<=|=|>|<)\s*([^,]+)\s*$`)
	k8sSelector = regexp.MustCompile(`^\s*(!?[a-zA-Z0-9_.-/]+)(\s+(in|notin)\s+\([^)]*\)|(\s*(==|=|!=)\s*[a-zA-Z0-9_.-/]+)?)?\s*$`)
)

type compiledLabelFilter struct {
	key, value       string
	op               string
	numValue         float64
	isNumeric        bool
	hasKeyWildcard   bool
	hasValueWildcard bool
	keyGlob          glob.Glob
	valueGlob        glob.Glob
}

// K8sSelectorForAPI extracts a subset of label selectors that are safe to pass to the Kubernetes API.
// It only includes selectors (=, ==, != and set selectors) and ignores numeric and * ones.
func K8sSelectorForAPI(rawSelector string) string {
	if rawSelector == "" {
		return ""
	}
	parts := strings.Split(rawSelector, ",")
	var safeParts []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if k8sSelector.MatchString(trimmed) {
			safeParts = append(safeParts, trimmed)
		}
	}
	return strings.Join(safeParts, ",")
}

// MatchLabels checks if a map of labels matches a raw selector string.
// The selector supports numeric comparisons (>, <, >=, <=),
// and wildcards (*) in keys and values for equality checks.
func MatchLabels(labels map[string]string, rawSelector string) (matched bool, k8sHandled bool, err error) {
	if rawSelector == "" {
		return true, false, nil
	}

	filters, err := compileSelector(rawSelector)
	if err != nil {
		return false, false, err
	}
	if len(filters) == 0 { // keep only K8s-compatible selectors
		return false, true, nil
	}

	for _, f := range filters {
		if !matchFilter(labels, f) {
			return false, false, nil // All filters must match
		}
	}

	return true, false, nil
}

func compileSelector(sel string) ([]compiledLabelFilter, error) {
	parts := strings.Split(sel, ",")
	filters := make([]compiledLabelFilter, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Skip selectors handled by the K8s API server.
		if k8sSelector.MatchString(part) {
			continue
		}

		submatch := labelExpr.FindStringSubmatch(part)
		if len(submatch) != 4 {
			return nil, fmt.Errorf("invalid label filter segment: %q", part)
		}

		op := submatch[2]

		cf := compiledLabelFilter{
			key:   strings.TrimSpace(submatch[1]),
			value: strings.TrimSpace(submatch[3]),
			op:    op,
		}
		if num, err := strconv.ParseFloat(cf.value, 64); err == nil {
			if cf.op == ">" || cf.op == "<" || cf.op == ">=" || cf.op == "<=" {
				cf.isNumeric = true
				cf.numValue = num
			}
		}

		// Precompile globs for wildcard keys (any op) and values (equality op only)
		if strings.Contains(cf.key, "*") {
			cf.hasKeyWildcard = true
			cf.keyGlob = glob.MustCompile(cf.key)
		}
		if !cf.isNumeric && cf.op == "=" && strings.Contains(cf.value, "*") {
			cf.hasValueWildcard = true
			cf.valueGlob = glob.MustCompile(cf.value)
		}

		filters = append(filters, cf)
	}
	return filters, nil
}

func matchFilter(labels map[string]string, compiledLabel compiledLabelFilter) bool {
	for labelKey, labelValue := range labels {
		var keyMatch bool
		if compiledLabel.hasKeyWildcard {
			keyMatch = compiledLabel.keyGlob.Match(labelKey)
		} else {
			keyMatch = compiledLabel.key == labelKey
		}
		if !keyMatch {
			continue
		}

		if compiledLabel.isNumeric {
			labelVal, err := strconv.ParseFloat(labelValue, 64)
			if err != nil {
				continue // Not a number, cannot compare
			}
			if evaluateNumericComparison(compiledLabel, labelVal) {
				return true
			}
		}
		if compiledLabel.op == "=" { // Equality or wildcard equality
			var valMatch bool
			if compiledLabel.hasValueWildcard {
				valMatch = compiledLabel.valueGlob.Match(labelValue)
			} else {
				valMatch = compiledLabel.value == labelValue
			}
			if valMatch {
				return true
			}
		}
	}
	return false
}

func evaluateNumericComparison(compiledLabel compiledLabelFilter, value float64) bool {
	switch compiledLabel.op {
	case ">":
		if value > compiledLabel.numValue {
			return true
		}
	case "<":
		if value < compiledLabel.numValue {
			return true
		}
	case ">=":
		if value >= compiledLabel.numValue {
			return true
		}
	case "<=":
		if value <= compiledLabel.numValue {
			return true
		}
	}
	return false
}
