package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSpecPathToSchemaPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sizeGB", "properties.spec.properties.sizeGB"},
		{"storage.sizeGB", "properties.spec.properties.storage.properties.sizeGB"},
		{"a.b.c", "properties.spec.properties.a.properties.b.properties.c"},
	}
	for _, tt := range tests {
		got := specPathToSchemaPath(tt.input)
		if got != tt.want {
			t.Errorf("specPathToSchemaPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLoadRules(t *testing.T) {
	rulesYAML := `rules:
  - name: test-rule
    file: storage/test.yaml
    specPath: sizeGB
    validations:
      - rule: "self > 0"
        message: "must be positive"
`
	path := filepath.Join(t.TempDir(), "rules.yaml")
	if err := os.WriteFile(path, []byte(rulesYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	rules, err := loadRules(path)
	if err != nil {
		t.Fatalf("loadRules() error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	if rules[0].Name != "test-rule" {
		t.Errorf("name = %q, want %q", rules[0].Name, "test-rule")
	}
	if rules[0].File != "storage/test.yaml" {
		t.Errorf("file = %q, want %q", rules[0].File, "storage/test.yaml")
	}
	if rules[0].SpecPath != "sizeGB" {
		t.Errorf("specPath = %q, want %q", rules[0].SpecPath, "sizeGB")
	}
	if len(rules[0].Validations) != 1 {
		t.Fatalf("got %d validations, want 1", len(rules[0].Validations))
	}
}

func TestLoadRulesMissingFields(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		errIs string
	}{
		{
			name:  "missing file",
			yaml:  "rules:\n  - name: r1\n    specPath: x\n    validations:\n      - rule: \"self > 0\"\n",
			errIs: "missing 'file'",
		},
		{
			name:  "missing specPath",
			yaml:  "rules:\n  - name: r1\n    file: f.yaml\n    validations:\n      - rule: \"self > 0\"\n",
			errIs: "missing 'specPath'",
		},
		{
			name:  "missing validations",
			yaml:  "rules:\n  - name: r1\n    file: f.yaml\n    specPath: x\n",
			errIs: "missing 'validations'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "rules.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := loadRules(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errIs) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errIs)
			}
		})
	}
}

const minimalCRD = `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test-resources.example.com
spec:
  group: example.com
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              sizeGB:
                description: Size in GB.
                type: integer
        type: object
`

func mustParseValidations(t *testing.T, yamlStr string) []yaml.Node {
	t.Helper()
	var nodes []yaml.Node
	if err := yaml.Unmarshal([]byte(yamlStr), &nodes); err != nil {
		t.Fatalf("failed to parse validations: %v", err)
	}
	return nodes
}

func TestPatchFileInjectsValidations(t *testing.T) {
	dir := t.TempDir()
	crdPath := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(crdPath, []byte(minimalCRD), 0o600); err != nil {
		t.Fatal(err)
	}

	rule := Rule{
		Name:     "test-rule",
		File:     "test.yaml",
		SpecPath: "sizeGB",
		Validations: mustParseValidations(t, `
- rule: "self > 0"
  message: "must be positive"
`),
	}

	patched, err := patchFile(crdPath, rule)
	if err != nil {
		t.Fatalf("patchFile() error: %v", err)
	}
	if !patched {
		t.Fatal("patchFile() returned false, expected true")
	}

	data, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "x-kubernetes-validations:") {
		t.Error("patched file missing x-kubernetes-validations")
	}
	if !strings.Contains(content, "self > 0") {
		t.Error("patched file missing validation rule")
	}
	if !strings.Contains(content, "must be positive") {
		t.Error("patched file missing validation message")
	}
}

func TestPatchFileIdempotent(t *testing.T) {
	dir := t.TempDir()
	crdPath := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(crdPath, []byte(minimalCRD), 0o600); err != nil {
		t.Fatal(err)
	}

	rule := Rule{
		Name:     "test-rule",
		File:     "test.yaml",
		SpecPath: "sizeGB",
		Validations: mustParseValidations(t, `
- rule: "self > 0"
  message: "must be positive"
`),
	}

	// First patch
	if _, err := patchFile(crdPath, rule); err != nil {
		t.Fatalf("first patchFile() error: %v", err)
	}

	afterFirst, _ := os.ReadFile(crdPath)

	// Second patch should not modify
	patched, err := patchFile(crdPath, rule)
	if err != nil {
		t.Fatalf("second patchFile() error: %v", err)
	}
	if patched {
		t.Error("second patchFile() returned true, expected false (idempotent)")
	}

	afterSecond, _ := os.ReadFile(crdPath)
	if !bytes.Equal(afterFirst, afterSecond) {
		t.Error("file content changed on second patch")
	}
}

func TestPatchFileMissingFile(t *testing.T) {
	rule := Rule{
		Name:     "test-rule",
		File:     "nonexistent.yaml",
		SpecPath: "sizeGB",
		Validations: mustParseValidations(t, `
- rule: "self > 0"
  message: "must be positive"
`),
	}

	patched, err := patchFile("/tmp/nonexistent.yaml", rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patched {
		t.Error("expected false for missing file")
	}
}

func TestMergeValidations(t *testing.T) {
	target := &yaml.Node{Kind: yaml.MappingNode}

	validations := mustParseValidations(t, `
- rule: "self > 0"
  message: "must be positive"
- rule: "self < 100"
  message: "must be under 100"
`)

	changed := mergeValidations(target, validations)
	if !changed {
		t.Error("expected mergeValidations to return true")
	}

	seq := getSequenceValue(target, "x-kubernetes-validations")
	if seq == nil {
		t.Fatal("x-kubernetes-validations sequence not found")
	}
	if len(seq.Content) != 2 {
		t.Errorf("got %d validations, want 2", len(seq.Content))
	}

	// Merging again should not change anything
	changed2 := mergeValidations(target, validations)
	if changed2 {
		t.Error("expected mergeValidations to return false on duplicate")
	}
}
