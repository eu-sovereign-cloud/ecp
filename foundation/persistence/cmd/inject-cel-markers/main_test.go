package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCELRules(t *testing.T) {
	line := `	SizeGB int ` + "`" + `json:"sizeGB" x-cel-rule-0:"!oldSelf.hasValue() || self >= oldSelf.value()" x-cel-message-0:"spec.sizeGB cannot be decreased" x-cel-rule-1:"self > 0" x-cel-message-1:"spec.sizeGB must be greater than 0"` + "`"

	rules := extractCELRules(line)
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}

	if rules[0].rule != "!oldSelf.hasValue() || self >= oldSelf.value()" {
		t.Errorf("rule[0].rule = %q", rules[0].rule)
	}
	if rules[0].message != "spec.sizeGB cannot be decreased" {
		t.Errorf("rule[0].message = %q", rules[0].message)
	}
	if !rules[0].optionalOldSelf {
		t.Error("rule[0].optionalOldSelf should be true")
	}

	if rules[1].rule != "self > 0" {
		t.Errorf("rule[1].rule = %q", rules[1].rule)
	}
	if rules[1].message != "spec.sizeGB must be greater than 0" {
		t.Errorf("rule[1].message = %q", rules[1].message)
	}
	if rules[1].optionalOldSelf {
		t.Error("rule[1].optionalOldSelf should be false")
	}
}

func TestExtractCELRulesNoTags(t *testing.T) {
	line := `	Name string ` + "`" + `json:"name"` + "`"
	rules := extractCELRules(line)
	if len(rules) != 0 {
		t.Fatalf("got %d rules, want 0", len(rules))
	}
}

func TestExtractCELRulesIncomplete(t *testing.T) {
	// Only a rule, no message — should still extract (message is optional in marker).
	line := `	SizeGB int ` + "`" + `json:"sizeGB" x-cel-rule-0:"self > 0"` + "`"
	rules := extractCELRules(line)
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	if rules[0].message != "" {
		t.Errorf("expected empty message, got %q", rules[0].message)
	}
}

func TestFormatMarker(t *testing.T) {
	tests := []struct {
		name string
		rule celRule
		want string
	}{
		{
			name: "simple",
			rule: celRule{rule: "self > 0", message: "must be positive"},
			want: `// +kubebuilder:validation:XValidation:rule="self > 0",message="must be positive"`,
		},
		{
			name: "with oldSelf",
			rule: celRule{rule: "oldSelf == self", message: "immutable", optionalOldSelf: true},
			want: `// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="immutable",optionalOldSelf=true`,
		},
		{
			name: "no message",
			rule: celRule{rule: "self > 0"},
			want: `// +kubebuilder:validation:XValidation:rule="self > 0"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMarker(tt.rule)
			if got != tt.want {
				t.Errorf("got:\n  %s\nwant:\n  %s", got, tt.want)
			}
		})
	}
}

const testGoFile = `package types

// BlockStorageSpec References the SKU used for this block.
// If a reference to the source image is used as the base for creating this block storage.
type BlockStorageSpec struct {
	// SizeGB Size of the block storage in GB.
	SizeGB int ` + "`" + `json:"sizeGB" x-cel-rule-0:"!oldSelf.hasValue() || self >= oldSelf.value()" x-cel-message-0:"spec.sizeGB cannot be decreased" x-cel-rule-1:"self > 0" x-cel-message-1:"spec.sizeGB must be greater than 0"` + "`" + `

	// SkuRef Reference to the SKU of the block storage.
	SkuRef Reference ` + "`" + `json:"skuRef"` + "`" + `
}
`

func TestProcessFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_generated_block-storage.go")
	if err := os.WriteFile(path, []byte(testGoFile), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	if n != 2 {
		t.Errorf("injected %d markers, want 2", n)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Check markers are present.
	if !strings.Contains(content, `// +kubebuilder:validation:XValidation:rule="!oldSelf.hasValue() || self >= oldSelf.value()",message="spec.sizeGB cannot be decreased",optionalOldSelf=true`) {
		t.Error("missing first XValidation marker")
	}
	if !strings.Contains(content, `// +kubebuilder:validation:XValidation:rule="self > 0",message="spec.sizeGB must be greater than 0"`) {
		t.Error("missing second XValidation marker")
	}

	// Check markers appear before the field line.
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "SizeGB int") {
			if i < 2 {
				t.Fatal("expected at least 2 marker lines before SizeGB")
			}
			if !strings.Contains(lines[i-1], "+kubebuilder:validation:XValidation") {
				t.Errorf("line before SizeGB is not a marker: %q", lines[i-1])
			}
			if !strings.Contains(lines[i-2], "+kubebuilder:validation:XValidation") {
				t.Errorf("two lines before SizeGB is not a marker: %q", lines[i-2])
			}
			break
		}
	}

	// Check SkuRef was not touched.
	if strings.Contains(content, "SkuRef") {
		for i, line := range lines {
			if strings.Contains(line, "SkuRef Reference") {
				if i > 0 && strings.Contains(lines[i-1], "+kubebuilder") {
					t.Error("SkuRef should not have a marker")
				}
				break
			}
		}
	}
}

func TestProcessFileIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_generated_block-storage.go")
	if err := os.WriteFile(path, []byte(testGoFile), 0o644); err != nil {
		t.Fatal(err)
	}

	// First run.
	if _, err := processFile(path); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)

	// Second run.
	n, err := processFile(path)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)

	if string(first) != string(second) {
		t.Error("file changed on second run — not idempotent")
	}
	// On second run, markers already exist so they get removed and re-added.
	// The count should still be 2 since we re-inject them.
	if n != 2 {
		t.Errorf("second run injected %d markers, want 2", n)
	}
}

func TestProcessFileNoTags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.go")
	content := `package types

type Foo struct {
	Name string ` + "`" + `json:"name"` + "`" + `
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("injected %d markers, want 0", n)
	}
}
