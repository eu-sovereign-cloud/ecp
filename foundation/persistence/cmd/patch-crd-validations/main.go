// patch-crd-validations patches generated Kubernetes CRDs with CEL validations.
//
// CRDs and Go structs in this repo are generated via controller-gen. We keep the
// source-of-truth in Go types, but some validations are easier to express as
// x-kubernetes-validations (CEL). Since the CRDs are regenerated frequently, we
// apply these validations as a post-generation patch step.
//
// This tool:
//   - reads one or more rules from a YAML file
//   - patches the specified CRD YAML files
//   - injects x-kubernetes-validations at the requested OpenAPI schema location
//
// It uses yaml.Node for parsing (to locate insertion points via line numbers)
// but inserts validation snippets as text to preserve the original YAML formatting.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RulesFile is the top-level structure of the rules YAML file.
type RulesFile struct {
	Rules []Rule `yaml:"rules"`
}

// Rule describes a single CEL validation to inject into a CRD.
type Rule struct {
	Name        string           `yaml:"name"`
	File        string           `yaml:"file"`
	SpecPath    string           `yaml:"specPath"`
	Validations []map[string]any `yaml:"validations"`
}

func main() {
	rulesPath := flag.String("rules", "", "Path to YAML file describing validations to apply")
	rootDir := flag.String("root", "", "Root directory containing generated CRD YAMLs")
	dryRun := flag.Bool("dry-run", false, "Don't write files, only report what would change")
	flag.Parse()

	if *rulesPath == "" || *rootDir == "" {
		fmt.Fprintln(os.Stderr, "error: --rules and --root are required")
		os.Exit(1)
	}

	rules, err := loadRules(*rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	totalPatched := 0
	for _, rule := range rules {
		filePath := filepath.Join(*rootDir, rule.File)

		if *dryRun {
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Printf("[DRY-RUN] rule '%s': file not found: %s\n", rule.Name, filePath)
				continue
			}
			fmt.Printf("[DRY-RUN] would patch %s\n", filePath)
			continue
		}

		patched, err := patchFile(filePath, rule)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error patching %s: %v\n", filePath, err)
			os.Exit(1)
		}
		if patched {
			fmt.Printf("[PATCHED] %s\n", filePath)
			totalPatched++
		}
	}

	fmt.Printf("patched_files=%d rules=%d\n", totalPatched, len(rules))
}

func loadRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from CLI flags
	if err != nil {
		return nil, fmt.Errorf("reading rules file: %w", err)
	}

	var rf RulesFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("parsing rules file: %w", err)
	}

	for _, r := range rf.Rules {
		if r.Name == "" {
			r.Name = "unnamed"
		}
		if r.File == "" {
			return nil, fmt.Errorf("rule '%s' missing 'file'", r.Name)
		}
		if r.SpecPath == "" {
			return nil, fmt.Errorf("rule '%s' missing 'specPath'", r.Name)
		}
		if len(r.Validations) == 0 {
			return nil, fmt.Errorf("rule '%s' missing 'validations'", r.Name)
		}
	}

	return rf.Rules, nil
}

// specPathToSchemaPath converts e.g. "sizeGB" to "properties.spec.properties.sizeGB"
// and "storage.sizeGB" to "properties.spec.properties.storage.properties.sizeGB".
func specPathToSchemaPath(specPath string) string {
	parts := strings.Split(specPath, ".")
	var filtered []string
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	return "properties.spec.properties." + strings.Join(filtered, ".properties.")
}

// insertionPoint holds information about where to insert validations in the text.
type insertionPoint struct {
	// afterLine is the 1-based line number after which to insert (the last line of the target mapping's content).
	afterLine int
	// indent is the number of spaces to use for the x-kubernetes-validations key (same as sibling keys).
	indent int
}

func patchFile(path string, rule Rule) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("[WARN] rule '%s': file not found: %s\n", rule.Name, path)
		return false, nil
	}

	data, err := os.ReadFile(path) //nolint:gosec // path comes from CLI flags
	if err != nil {
		return false, fmt.Errorf("reading file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	schemaPath := specPathToSchemaPath(rule.SpecPath)

	docs, err := decodeAllDocs(data)
	if err != nil {
		return false, fmt.Errorf("decoding YAML: %w", err)
	}

	points := findAllInsertionPoints(docs, schemaPath)
	if len(points) == 0 {
		return false, nil
	}

	// Insert at each point in reverse line order to preserve line numbers
	for i := len(points) - 1; i >= 0; i-- {
		p := points[i]
		snippet := renderValidationSnippet(p.indent, rule.Validations)
		lines = insertLines(lines, p.afterLine, snippet)
	}

	result := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(result), 0o600); err != nil { //nolint:gosec // path comes from CLI flags
		return false, fmt.Errorf("writing file: %w", err)
	}

	return true, nil
}

// findAllInsertionPoints scans all CRD documents for locations where
// x-kubernetes-validations should be injected.
func findAllInsertionPoints(docs []*yaml.Node, schemaPath string) []insertionPoint {
	var points []insertionPoint

	for _, doc := range docs {
		if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
			continue
		}
		root := doc.Content[0]
		if root.Kind != yaml.MappingNode {
			continue
		}
		if getScalarValue(root, "kind") != "CustomResourceDefinition" {
			continue
		}

		specNode := getMappingValue(root, "spec")
		if specNode == nil {
			continue
		}
		versionsNode := getSequenceValue(specNode, "versions")
		if versionsNode == nil {
			continue
		}

		for _, ver := range versionsNode.Content {
			if p, ok := findInsertionPointForVersion(ver, schemaPath); ok {
				points = append(points, p)
			}
		}
	}

	return points
}

// findInsertionPointForVersion checks a single CRD version entry for the target
// schema path and returns the insertion point if validations are needed.
func findInsertionPointForVersion(ver *yaml.Node, schemaPath string) (insertionPoint, bool) {
	if ver.Kind != yaml.MappingNode {
		return insertionPoint{}, false
	}
	schemaNode := getMappingValue(ver, "schema")
	if schemaNode == nil {
		return insertionPoint{}, false
	}
	openAPISchema := getMappingValue(schemaNode, "openAPIV3Schema")
	if openAPISchema == nil {
		return insertionPoint{}, false
	}

	target := walkToMapping(openAPISchema, schemaPath)
	if target == nil {
		return insertionPoint{}, false
	}

	// Already has validations — skip
	if getMappingValue(target, "x-kubernetes-validations") != nil || getSequenceValue(target, "x-kubernetes-validations") != nil {
		return insertionPoint{}, false
	}

	return findInsertionPoint(target)
}

// findInsertionPoint determines where to insert in the text based on the yaml.Node tree.
// It returns the line after which to insert and the indentation level.
func findInsertionPoint(target *yaml.Node) (insertionPoint, bool) {
	if target.Kind != yaml.MappingNode || len(target.Content) < 2 {
		return insertionPoint{}, false
	}

	// The indent for sibling keys is the column of the first key in the mapping.
	// yaml.Node.Column is 1-based.
	indent := target.Content[0].Column - 1

	// Find the last line used by this mapping's content.
	lastLine := lastLineOf(target)

	return insertionPoint{afterLine: lastLine, indent: indent}, true
}

// lastLineOf finds the deepest (highest line number) line in a node subtree.
func lastLineOf(n *yaml.Node) int {
	best := n.Line
	for _, c := range n.Content {
		if l := lastLineOf(c); l > best {
			best = l
		}
	}
	return best
}

// renderValidationSnippet produces the YAML text lines for x-kubernetes-validations.
func renderValidationSnippet(indent int, validations []map[string]any) []string {
	prefix := strings.Repeat(" ", indent)
	entryPrefix := strings.Repeat(" ", indent)

	var out []string
	out = append(out, prefix+"x-kubernetes-validations:")

	for _, v := range validations {
		first := true
		// Render in a stable order: rule, message, then remaining keys
		orderedKeys := orderValidationKeys(v)
		for _, key := range orderedKeys {
			val := v[key]
			rendered := renderValue(val)
			if first {
				out = append(out, entryPrefix+"- "+key+": "+rendered)
				first = false
			} else {
				out = append(out, entryPrefix+"  "+key+": "+rendered)
			}
		}
	}

	return out
}

// orderValidationKeys returns keys in a stable order: rule first, message second, rest alphabetically.
func orderValidationKeys(v map[string]any) []string {
	var keys []string
	hasRule := false
	hasMessage := false
	for k := range v {
		switch k {
		case "rule":
			hasRule = true
		case "message":
			hasMessage = true
		default:
			keys = append(keys, k)
		}
	}
	// Sort remaining keys
	sortStrings(keys)

	var result []string
	if hasRule {
		result = append(result, "rule")
	}
	if hasMessage {
		result = append(result, "message")
	}
	result = append(result, keys...)
	return result
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func renderValue(v any) string {
	switch val := v.(type) {
	case string:
		// Quote strings that contain special YAML characters
		if needsQuoting(val) {
			return fmt.Sprintf("%q", val)
		}
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func needsQuoting(s string) bool {
	if s == "" {
		return true
	}
	// Quote if contains characters that could confuse YAML parsing
	for _, c := range s {
		switch c {
		case ':', '{', '}', '[', ']', ',', '&', '*', '#', '?', '|', '-', '<', '>', '=', '!', '%', '@', '`', '"', '\'', '\\':
			return true
		}
	}
	// Quote YAML special values
	lower := strings.ToLower(s)
	switch lower {
	case "true", "false", "null", "yes", "no", "on", "off":
		return true
	}
	return false
}

func insertLines(lines []string, afterLine int, newLines []string) []string {
	// afterLine is 1-based; convert to 0-based index
	idx := afterLine
	if idx > len(lines) {
		idx = len(lines)
	}
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:idx]...)
	result = append(result, newLines...)
	result = append(result, lines[idx:]...)
	return result
}

func decodeAllDocs(data []byte) ([]*yaml.Node, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var docs []*yaml.Node
	for {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		docs = append(docs, &doc)
	}
	return docs, nil
}

// getScalarValue returns the string value for a scalar key in a mapping node.
func getScalarValue(mapping *yaml.Node, key string) string {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1].Value
		}
	}
	return ""
}

// getMappingValue returns the mapping node for a key, or nil.
func getMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key && mapping.Content[i+1].Kind == yaml.MappingNode {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// getSequenceValue returns the sequence node for a key, or nil.
func getSequenceValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key && mapping.Content[i+1].Kind == yaml.SequenceNode {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// walkToMapping navigates a dot-separated path through a yaml.Node tree.
// Unlike the Python version, this does NOT create intermediate nodes — it only reads.
func walkToMapping(root *yaml.Node, schemaPath string) *yaml.Node {
	node := root
	for _, seg := range strings.Split(schemaPath, ".") {
		if seg == "" {
			continue
		}
		if node.Kind != yaml.MappingNode {
			return nil
		}
		next := getMappingValue(node, seg)
		if next == nil {
			return nil
		}
		node = next
	}
	return node
}
