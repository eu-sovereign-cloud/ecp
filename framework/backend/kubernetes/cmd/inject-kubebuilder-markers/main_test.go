package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── CEL extraction tests (ported from inject-cel-markers) ────────────────────

func TestExtractCELRules(t *testing.T) {
	line := `	SizeGB int ` + "`" + `json:"sizeGB" x-cel-rule-0:"!oldSelf.hasValue() || self >= oldSelf.value()" x-cel-message-0:"spec.sizeGB cannot be decreased" x-cel-rule-1:"self > 0" x-cel-message-1:"spec.sizeGB must be greater than 0"` + "`"

	rules := extractCELRules(line)
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}
	if rules[0].rule != "!oldSelf.hasValue() || self >= oldSelf.value()" {
		t.Errorf("rule[0].rule = %q", rules[0].rule)
	}
	if !rules[0].optionalOldSelf {
		t.Error("rule[0].optionalOldSelf should be true")
	}
	if rules[1].rule != "self > 0" {
		t.Errorf("rule[1].rule = %q", rules[1].rule)
	}
	if rules[1].optionalOldSelf {
		t.Error("rule[1].optionalOldSelf should be false")
	}
}

func TestExtractCELRulesNoTags(t *testing.T) {
	line := `	Name string ` + "`" + `json:"name"` + "`"
	if got := extractCELRules(line); len(got) != 0 {
		t.Fatalf("got %d rules, want 0", len(got))
	}
}

func TestFormatCELMarker(t *testing.T) {
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
			if got := formatCELMarker(tt.rule); got != tt.want {
				t.Errorf("got:\n  %s\nwant:\n  %s", got, tt.want)
			}
		})
	}
}

// ── Kubebuilder validation marker tests ──────────────────────────────────────

func TestExtractKBMarkers(t *testing.T) {
	line := `	Items []Foo ` + "`" + `json:"items,omitempty" x-kubebuilder-validation-max-items:"64"` + "`"

	markers := extractKBMarkers(line)
	if len(markers) != 1 {
		t.Fatalf("got %d markers, want 1", len(markers))
	}
	if markers[0].name != "MaxItems" {
		t.Errorf("name = %q, want MaxItems", markers[0].name)
	}
	if markers[0].value != "64" {
		t.Errorf("value = %q, want 64", markers[0].value)
	}
}

func TestExtractKBMarkersMaxLength(t *testing.T) {
	line := `	Protocol string ` + "`" + `json:"protocol,omitempty" x-kubebuilder-validation-max-length:"7"` + "`"

	markers := extractKBMarkers(line)
	if len(markers) != 1 {
		t.Fatalf("got %d markers, want 1", len(markers))
	}
	if markers[0].name != "MaxLength" {
		t.Errorf("name = %q, want MaxLength", markers[0].name)
	}
	if markers[0].value != "7" {
		t.Errorf("value = %q, want 7", markers[0].value)
	}
}

func TestExtractKBMarkersNoTags(t *testing.T) {
	line := `	Name string ` + "`" + `json:"name"` + "`"
	if got := extractKBMarkers(line); len(got) != 0 {
		t.Fatalf("got %d markers, want 0", len(got))
	}
}

func TestExtractKBMarkersAllTypes(t *testing.T) {
	tests := []struct {
		tag      string
		wantName string
		wantVal  string
	}{
		{`x-kubebuilder-validation-min-length:"4"`, "MinLength", "4"},
		{`x-kubebuilder-validation-min-items:"1"`, "MinItems", "1"},
		{`x-kubebuilder-validation-minimum:"0"`, "Minimum", "0"},
		{`x-kubebuilder-validation-maximum:"65535"`, "Maximum", "65535"},
		{`x-kubebuilder-validation-max-properties:"9"`, "MaxProperties", "9"},
		{`x-kubebuilder-validation-maximum:"200000000000"`, "Maximum", "200000000000"},
		{`x-kubebuilder-validation-enum:"ingress;egress"`, "Enum", "ingress;egress"},
		{`x-kubebuilder-validation-enum:"virtio"`, "Enum", "virtio"},
		{`x-kubebuilder-validation-enum:"none;stable;preview"`, "Enum", "none;stable;preview"},
		{`x-kubebuilder-validation-items-min-length:"1"`, "items:MinLength", "1"},
		{`x-kubebuilder-validation-items-max-length:"256"`, "items:MaxLength", "256"},
		{`x-kubebuilder-validation-items-minimum:"1"`, "items:Minimum", "1"},
		{`x-kubebuilder-validation-items-maximum:"65535"`, "items:Maximum", "65535"},
		{`x-kubebuilder-validation-pattern:"^[a-z]+$"`, "Pattern", "^[a-z]+$"},
	}
	for _, tt := range tests {
		line := "\tField int `json:\"f\" " + tt.tag + "`"
		markers := extractKBMarkers(line)
		if len(markers) != 1 {
			t.Errorf("tag %q: got %d markers, want 1", tt.tag, len(markers))
			continue
		}
		if markers[0].name != tt.wantName || markers[0].value != tt.wantVal {
			t.Errorf("tag %q: got {%s %s}, want {%s %s}", tt.tag, markers[0].name, markers[0].value, tt.wantName, tt.wantVal)
		}
	}
}

func TestExtractKBMarkersUnknownTag(t *testing.T) {
	line := `	Name string ` + "`" + `json:"name" x-kubebuilder-validation-unknown:"99"` + "`"
	if got := extractKBMarkers(line); len(got) != 0 {
		t.Fatalf("got %d markers for unknown tag, want 0", len(got))
	}
}

func TestFormatKBMarker(t *testing.T) {
	tests := []struct {
		m    kbMarker
		want string
	}{
		{kbMarker{"MaxItems", "64"}, "// +kubebuilder:validation:MaxItems=64"},
		{kbMarker{"MaxLength", "7"}, "// +kubebuilder:validation:MaxLength=7"},
		{kbMarker{"MinItems", "1"}, "// +kubebuilder:validation:MinItems=1"},
		{kbMarker{"MinLength", "4"}, "// +kubebuilder:validation:MinLength=4"},
		{kbMarker{"Minimum", "0"}, "// +kubebuilder:validation:Minimum=0"},
		{kbMarker{"Maximum", "65535"}, "// +kubebuilder:validation:Maximum=65535"},
		{kbMarker{"MaxProperties", "9"}, "// +kubebuilder:validation:MaxProperties=9"},
		{kbMarker{"Enum", "ingress;egress"}, "// +kubebuilder:validation:Enum=ingress;egress"},
		{kbMarker{"items:MaxLength", "45"}, "// +kubebuilder:validation:items:MaxLength=45"},
		{kbMarker{"items:MinLength", "1"}, "// +kubebuilder:validation:items:MinLength=1"},
		{kbMarker{"items:Maximum", "65535"}, "// +kubebuilder:validation:items:Maximum=65535"},
		{kbMarker{"items:Minimum", "1"}, "// +kubebuilder:validation:items:Minimum=1"},
		{kbMarker{"Pattern", "^[a-z]+$"}, "// +kubebuilder:validation:Pattern=`^[a-z]+$`"},
	}
	for _, tt := range tests {
		if got := formatKBMarker(tt.m); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

// ── processFile tests ─────────────────────────────────────────────────────────

const testGoFileCEL = `package types

type BlockStorageSpec struct {
	SizeGB int ` + "`" + `json:"sizeGB" x-cel-rule-0:"self > 0" x-cel-message-0:"spec.sizeGB must be greater than 0"` + "`" + `
}
`

func TestProcessFileCEL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileCEL), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	if n != 1 {
		t.Errorf("injected %d markers, want 1", n)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `// +kubebuilder:validation:XValidation:rule="self > 0"`) {
		t.Error("missing XValidation marker")
	}
}

const testGoFileKB = `package types

type NetworkSpec struct {
	AdditionalCidrs []Cidr ` + "`" + `json:"additionalCidrs,omitempty" x-kubebuilder-validation-max-items:"10"` + "`" + `
	Protocol string ` + "`" + `json:"protocol,omitempty" x-kubebuilder-validation-max-length:"7"` + "`" + `
}
`

func TestProcessFileKBMarkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileKB), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	// MaxItems=10 (no length → no Type=string) + Type=string + MaxLength=7 = 3 total.
	if n != 3 {
		t.Errorf("injected %d markers, want 3", n)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "// +kubebuilder:validation:MaxItems=10") {
		t.Error("missing MaxItems marker")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:Type=string") {
		t.Error("missing Type=string marker (injected alongside MaxLength)")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:MaxLength=7") {
		t.Error("missing MaxLength marker")
	}
}

const testGoFileBoth = `package types

type Foo struct {
	Items []Bar ` + "`" + `json:"items,omitempty" x-kubebuilder-validation-max-items:"64" x-cel-rule-0:"self.size() <= 64" x-cel-message-0:"too many items"` + "`" + `
}
`

func TestProcessFileBothMarkerTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileBoth), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	if n != 2 {
		t.Errorf("injected %d markers, want 2", n)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	lines := strings.Split(content, "\n")
	var fieldIdx int
	for i, l := range lines {
		if strings.Contains(l, "Items []Bar") {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 2 {
		t.Fatal("expected at least 2 marker lines before Items field")
	}
	// KB marker before CEL marker.
	if !strings.Contains(lines[fieldIdx-2], "MaxItems") {
		t.Errorf("expected MaxItems marker 2 lines before field, got: %q", lines[fieldIdx-2])
	}
	if !strings.Contains(lines[fieldIdx-1], "XValidation") {
		t.Errorf("expected XValidation marker 1 line before field, got: %q", lines[fieldIdx-1])
	}
}

func TestProcessFileIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileBoth), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := processFile(path); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)

	n, err := processFile(path)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)

	if !bytes.Equal(first, second) {
		t.Error("file changed on second run — not idempotent")
	}
	if n != 2 {
		t.Errorf("second run injected %d markers, want 2", n)
	}
}

const testGoFileItemsMinLength = `package types

type Resources struct {
	Resources []string ` + "`" + `json:"resources,omitempty" x-kubebuilder-validation-items-min-length:"1" x-kubebuilder-validation-items-max-length:"128"` + "`" + `
}
`

// testGoFileArrayPrimitiveConstraints represents a []string field with both array-level max-items
// and items-level max-length — the pattern produced after fixing inline-primitive array specs.
const testGoFileArrayPrimitiveConstraints = `package types

type KubernetesClusterSpec struct {
	RestrictKubernetesApi []string ` + "`" + `json:"restrictKubernetesApi,omitempty" x-kubebuilder-validation-items-max-length:"45" x-kubebuilder-validation-max-items:"100"` + "`" + `
}
`

func TestProcessFileArrayPrimitiveConstraints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileArrayPrimitiveConstraints), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	// items:Type=string (injected before items:MaxLength) + MaxItems + items:MaxLength = 3 total.
	if n != 3 {
		t.Errorf("injected %d markers, want 3", n)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "// +kubebuilder:validation:MaxItems=100") {
		t.Error("missing MaxItems marker")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:items:Type=string") {
		t.Error("missing items:Type=string marker (injected alongside items:MaxLength)")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:items:MaxLength=45") {
		t.Error("missing items:MaxLength marker")
	}

	lines := strings.Split(content, "\n")
	var fieldIdx int
	for i, l := range lines {
		if strings.Contains(l, "RestrictKubernetesApi") {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 3 {
		t.Fatal("expected at least 3 marker lines before field")
	}
	// Emit order: items:Type=string (fieldIdx-3), then sorted markers: MaxItems (M < i) → items:MaxLength.
	if !strings.Contains(lines[fieldIdx-2], "MaxItems") {
		t.Errorf("expected MaxItems 2 lines before field, got: %q", lines[fieldIdx-2])
	}
	if !strings.Contains(lines[fieldIdx-1], "items:MaxLength") {
		t.Errorf("expected items:MaxLength 1 line before field, got: %q", lines[fieldIdx-1])
	}
}

func TestProcessFileItemsMinLength(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileItemsMinLength), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	// items:Type=string (injected alongside items length constraints) + items:MaxLength + items:MinLength = 3.
	if n != 3 {
		t.Errorf("injected %d markers, want 3", n)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "// +kubebuilder:validation:items:Type=string") {
		t.Error("missing items:Type=string marker (injected alongside items:MinLength)")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:items:MinLength=1") {
		t.Error("missing items:MinLength marker")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:items:MaxLength=128") {
		t.Error("missing items:MaxLength marker")
	}
}

const testGoFileEnum = `package types

type ImageSpec struct {
	Boot string ` + "`" + `json:"boot,omitempty" x-kubebuilder-validation-enum:"UEFI;BIOS"` + "`" + `
}
`

func TestProcessFileEnum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileEnum), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	if n != 1 {
		t.Errorf("injected %d markers, want 1", n)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "// +kubebuilder:validation:Enum=UEFI;BIOS") {
		t.Error("missing Enum marker")
	}
}

const testGoFileMultipleCELRules = `package types

type BlockStorageSizeSpec struct {
	SizeGB int ` + "`" + `json:"sizeGB" x-cel-rule-0:"self > 0" x-cel-message-0:"must be positive" x-cel-rule-1:"self <= 1000000" x-cel-message-1:"too large"` + "`" + `
}
`

func TestProcessFileMultipleCELRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileMultipleCELRules), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	if n != 2 {
		t.Errorf("injected %d markers, want 2", n)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	lines := strings.Split(content, "\n")
	var fieldIdx int
	for i, l := range lines {
		if strings.Contains(l, "SizeGB int") {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 2 {
		t.Fatal("expected at least 2 marker lines before SizeGB field")
	}
	// rule-0 injected first, rule-1 second.
	if !strings.Contains(lines[fieldIdx-2], `rule="self > 0"`) {
		t.Errorf("expected rule-0 2 lines before field, got: %q", lines[fieldIdx-2])
	}
	if !strings.Contains(lines[fieldIdx-1], `rule="self <= 1000000"`) {
		t.Errorf("expected rule-1 1 line before field, got: %q", lines[fieldIdx-1])
	}
}

// ── default marker tests ──────────────────────────────────────────────────────

func TestExtractKBDefault(t *testing.T) {
	tests := []struct {
		line  string
		want  string
		found bool
	}{
		{
			line: "\tBoot string `json:\"boot\" x-kubebuilder-default:\"UEFI\"`",
			want: "UEFI", found: true,
		},
		{
			line: "\tEgressOnly bool `json:\"egressOnly\" x-kubebuilder-default:\"false\"`",
			want: "false", found: true,
		},
		{
			line: "\tLimit int `json:\"limit\" x-kubebuilder-default:\"1000\"`",
			want: "1000", found: true,
		},
		{
			line: "\tName string `json:\"name\"`",
			want: "", found: false,
		},
	}
	for _, tt := range tests {
		val, ok := extractKBDefault(tt.line)
		if ok != tt.found || val != tt.want {
			t.Errorf("line %q: got (%q, %v), want (%q, %v)", tt.line, val, ok, tt.want, tt.found)
		}
	}
}

func TestFormatDefaultMarker(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{"UEFI", `// +kubebuilder:default="UEFI"`},
		{"round-robin", `// +kubebuilder:default="round-robin"`},
		{"none", `// +kubebuilder:default="none"`},
		{"false", `// +kubebuilder:default=false`},
		{"true", `// +kubebuilder:default=true`},
		{"1000", `// +kubebuilder:default=1000`},
		{"0", `// +kubebuilder:default=0`},
	}
	for _, tt := range tests {
		if got := formatDefaultMarker(tt.value); got != tt.want {
			t.Errorf("value %q: got %q, want %q", tt.value, got, tt.want)
		}
	}
}

const testGoFilePattern = `package types

type StatusCondition struct {
	Reason string ` + "`" + `json:"reason,omitempty" x-kubebuilder-validation-max-length:"1024" x-kubebuilder-validation-pattern:"^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$"` + "`" + `
}
`

func TestProcessFilePattern(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFilePattern), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	// Type=string (injected before MaxLength) + MaxLength + Pattern = 3 total.
	if n != 3 {
		t.Errorf("injected %d markers, want 3", n)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "// +kubebuilder:validation:Type=string") {
		t.Error("missing Type=string marker (injected alongside MaxLength)")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:MaxLength=1024") {
		t.Error("missing MaxLength marker")
	}
	wantPattern := "// +kubebuilder:validation:Pattern=`^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$`"
	if !strings.Contains(content, wantPattern) {
		t.Errorf("missing Pattern marker; want %q", wantPattern)
	}

	// Emit order: Type=string (fieldIdx-3), then sorted markers: MaxLength (M=77) before Pattern (P=80).
	lines := strings.Split(content, "\n")
	var fieldIdx int
	for i, l := range lines {
		if strings.Contains(l, "Reason string") {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 3 {
		t.Fatal("expected at least 3 marker lines before Reason field")
	}
	if !strings.Contains(lines[fieldIdx-2], "MaxLength") {
		t.Errorf("expected MaxLength 2 lines before field, got: %q", lines[fieldIdx-2])
	}
	if !strings.Contains(lines[fieldIdx-1], "Pattern") {
		t.Errorf("expected Pattern 1 line before field, got: %q", lines[fieldIdx-1])
	}
}

const testGoFileDefault = `package types

type ImageSpec struct {
	Boot string ` + "`" + `json:"boot,omitempty" x-kubebuilder-default:"UEFI" x-kubebuilder-validation-enum:"UEFI;BIOS"` + "`" + `
}
`

func TestProcessFileDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileDefault), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	if n != 2 {
		t.Errorf("injected %d markers, want 2", n)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, `// +kubebuilder:default="UEFI"`) {
		t.Error("missing default marker")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:Enum=UEFI;BIOS") {
		t.Error("missing Enum marker")
	}

	// Default marker must appear before validation marker.
	lines := strings.Split(content, "\n")
	var fieldIdx int
	for i, l := range lines {
		if strings.Contains(l, "Boot string") {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 2 {
		t.Fatal("expected at least 2 marker lines before Boot field")
	}
	if !strings.Contains(lines[fieldIdx-2], "+kubebuilder:default") {
		t.Errorf("expected default marker 2 lines before field, got: %q", lines[fieldIdx-2])
	}
	if !strings.Contains(lines[fieldIdx-1], "+kubebuilder:validation") {
		t.Errorf("expected validation marker 1 line before field, got: %q", lines[fieldIdx-1])
	}
}

func TestProcessFileDefaultIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileDefault), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := processFile(path); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)

	n, err := processFile(path)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)

	if !bytes.Equal(first, second) {
		t.Error("file changed on second run — not idempotent")
	}
	if n != 2 {
		t.Errorf("second run injected %d markers, want 2", n)
	}
}

// ── items numeric constraints tests ──────────────────────────────────────────

const testGoFileItemsNumericConstraints = `package types

type PortsSpec struct {
	List []int32 ` + "`" + `json:"list,omitempty" x-kubebuilder-validation-items-minimum:"1" x-kubebuilder-validation-items-maximum:"65535" x-kubebuilder-validation-max-items:"100"` + "`" + `
}
`

func TestProcessFileItemsNumericConstraints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zz_test.go")
	if err := os.WriteFile(path, []byte(testGoFileItemsNumericConstraints), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := processFile(path)
	if err != nil {
		t.Fatalf("processFile() error: %v", err)
	}
	if n != 3 {
		t.Errorf("injected %d markers, want 3", n)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "// +kubebuilder:validation:MaxItems=100") {
		t.Error("missing MaxItems marker")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:items:Maximum=65535") {
		t.Error("missing items:Maximum marker")
	}
	if !strings.Contains(content, "// +kubebuilder:validation:items:Minimum=1") {
		t.Error("missing items:Minimum marker")
	}

	lines := strings.Split(content, "\n")
	var fieldIdx int
	for i, l := range lines {
		if strings.Contains(l, "List []int32") {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 3 {
		t.Fatal("expected at least 3 marker lines before List field")
	}
	// Sorted order: MaxItems (M=77) < items:Maximum (i=105) < items:Minimum.
	if !strings.Contains(lines[fieldIdx-3], "MaxItems") {
		t.Errorf("expected MaxItems 3 lines before field, got: %q", lines[fieldIdx-3])
	}
	if !strings.Contains(lines[fieldIdx-2], "items:Maximum") {
		t.Errorf("expected items:Maximum 2 lines before field, got: %q", lines[fieldIdx-2])
	}
	if !strings.Contains(lines[fieldIdx-1], "items:Minimum") {
		t.Errorf("expected items:Minimum 1 line before field, got: %q", lines[fieldIdx-1])
	}
}
