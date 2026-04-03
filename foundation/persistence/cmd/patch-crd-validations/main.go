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
	Name        string      `yaml:"name"`
	File        string      `yaml:"file"`
	SpecPath    string      `yaml:"specPath"`
	Validations []yaml.Node `yaml:"validations"`
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

	// NOTE: multiple rules targeting the same file will each read, parse, patch,
	// encode, and write the file separately. If this becomes a performance concern,
	// group rules by file and apply all rules in a single read-patch-write cycle.
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

func patchFile(path string, rule Rule) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("[WARN] rule '%s': file not found: %s\n", rule.Name, path)
		return false, nil
	}

	data, err := os.ReadFile(path) //nolint:gosec // path comes from CLI flags
	if err != nil {
		return false, fmt.Errorf("reading file: %w", err)
	}

	docs, err := decodeAllDocs(data)
	if err != nil {
		return false, fmt.Errorf("decoding YAML: %w", err)
	}

	schemaPath := specPathToSchemaPath(rule.SpecPath)
	touched := false

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
			modified, err := patchVersion(ver, schemaPath, rule.Validations)
			if err != nil {
				return false, fmt.Errorf("rule '%s': %w", rule.Name, err)
			}
			if modified {
				touched = true
			}
		}
	}

	if !touched {
		return false, nil
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	for _, doc := range docs {
		if err := enc.Encode(doc); err != nil {
			return false, fmt.Errorf("encoding YAML: %w", err)
		}
	}
	if err := enc.Close(); err != nil {
		return false, fmt.Errorf("closing encoder: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil { //nolint:gosec // path comes from CLI flags
		return false, fmt.Errorf("writing file: %w", err)
	}

	return true, nil
}

// patchVersion patches a single CRD version entry. Returns true if modified.
func patchVersion(ver *yaml.Node, schemaPath string, validations []yaml.Node) (bool, error) {
	if ver.Kind != yaml.MappingNode {
		return false, nil
	}
	schemaNode := getMappingValue(ver, "schema")
	if schemaNode == nil {
		return false, nil
	}
	openAPISchema := getMappingValue(schemaNode, "openAPIV3Schema")
	if openAPISchema == nil {
		return false, nil
	}

	target, err := walkToMapping(openAPISchema, schemaPath)
	if err != nil {
		return false, err
	}

	return mergeValidations(target, validations), nil
}

// mergeValidations appends validation entries to the x-kubernetes-validations list
// on the target node, skipping duplicates. Returns true if any were added.
func mergeValidations(target *yaml.Node, validations []yaml.Node) bool {
	valList := ensureSequenceKey(target, "x-kubernetes-validations")
	if valList == nil {
		return false
	}

	changed := false
	for i := range validations {
		v := &validations[i]
		if containsNode(valList, v) {
			continue
		}
		valList.Content = append(valList.Content, deepCopyNode(v))
		changed = true
	}
	return changed
}

// containsNode checks if a sequence already contains a node with identical content.
func containsNode(seq *yaml.Node, candidate *yaml.Node) bool {
	candidateBytes := marshalNode(candidate)
	for _, existing := range seq.Content {
		if bytes.Equal(marshalNode(existing), candidateBytes) {
			return true
		}
	}
	return false
}

func marshalNode(n *yaml.Node) []byte {
	b, _ := yaml.Marshal(n)
	return b
}

func deepCopyNode(n *yaml.Node) *yaml.Node {
	b, _ := yaml.Marshal(n)
	var out yaml.Node
	_ = yaml.Unmarshal(b, &out)
	if out.Kind == yaml.DocumentNode && len(out.Content) > 0 {
		return out.Content[0]
	}
	return &out
}

// ensureSequenceKey ensures a key exists in a mapping node and its value is a sequence.
func ensureSequenceKey(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			val := mapping.Content[i+1]
			if val.Kind == yaml.SequenceNode {
				return val
			}
			if val.Tag == "!!null" || (val.Kind == yaml.ScalarNode && val.Value == "") {
				newSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
				mapping.Content[i+1] = newSeq
				return newSeq
			}
			return nil
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
	valNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	mapping.Content = append(mapping.Content, keyNode, valNode)
	return valNode
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
// Returns an error if a segment in the path does not exist.
// NOTE: this only supports mapping keys (e.g. "properties.spec.properties.sizeGB").
// Array element schemas via "items" are not supported. If needed, add handling
// for an "items" segment similar to the Python version's _walk_schema_path.
func walkToMapping(root *yaml.Node, schemaPath string) (*yaml.Node, error) {
	node := root
	for _, seg := range strings.Split(schemaPath, ".") {
		if seg == "" {
			continue
		}
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("expected mapping at segment %q, got node kind %d", seg, node.Kind)
		}
		next := getMappingValue(node, seg)
		if next == nil {
			return nil, fmt.Errorf("segment %q not found in schema path %q", seg, schemaPath)
		}
		node = next
	}
	return node, nil
}
