package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func parseSource(t *testing.T, src string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parseSource: %v", err)
	}
	return fset, f
}

func formatFile(t *testing.T, fset *token.FileSet, f *ast.File) string {
	t.Helper()
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		t.Fatalf("formatFile: %v", err)
	}
	return buf.String()
}

func TestReplaceTimeType(t *testing.T) {
	t.Run("replaces time.Time and returns true", func(t *testing.T) {
		src := `package p
import "time"
type S struct {
	CreatedAt  time.Time  ` + "`json:\"createdAt\"`" + `
	DeletedAt *time.Time ` + "`json:\"deletedAt,omitempty\"`" + `
}`
		fset, f := parseSource(t, src)
		replaced := replaceTimeType(f)
		if !replaced {
			t.Fatal("expected replaced=true, got false")
		}
		out := formatFile(t, fset, f)
		if strings.Contains(out, "time.Time") {
			t.Errorf("output still contains time.Time:\n%s", out)
		}
		if !strings.Contains(out, "metav1.Time") {
			t.Errorf("output missing metav1.Time:\n%s", out)
		}
	})

	t.Run("returns false when no time.Time present", func(t *testing.T) {
		src := `package p
type S struct {
	Name string ` + "`json:\"name\"`" + `
}`
		_, f := parseSource(t, src)
		if replaceTimeType(f) {
			t.Error("expected replaced=false, got true")
		}
	})
}

func TestReplaceMapInterface(t *testing.T) {
	t.Run("replaces map[string]interface{} with map[string]string", func(t *testing.T) {
		src := `package p
type Extensions map[string]interface{}`
		fset, f := parseSource(t, src)
		replaceMapInterface(f)
		out := formatFile(t, fset, f)
		if strings.Contains(out, "interface{}") {
			t.Errorf("output still contains interface{}:\n%s", out)
		}
		if !strings.Contains(out, "map[string]string") {
			t.Errorf("output missing map[string]string:\n%s", out)
		}
	})

	t.Run("leaves map[string]int unchanged", func(t *testing.T) {
		src := `package p
type Counts map[string]int`
		fset, f := parseSource(t, src)
		replaceMapInterface(f)
		out := formatFile(t, fset, f)
		if !strings.Contains(out, "map[string]int") {
			t.Errorf("map[string]int was changed unexpectedly:\n%s", out)
		}
	})
}

// findStructType locates the first struct type declaration in a parsed file.
func findStructType(t *testing.T, f *ast.File) *ast.StructType {
	t.Helper()
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			return st
		}
	}
	t.Fatal("no struct type found in file")
	return nil
}

func TestAddUnionTags(t *testing.T) {
	t.Run("adds json:\"-\" to tagless union field", func(t *testing.T) {
		src := `package p
import "encoding/json"
type S struct {
	Name  string          ` + "`json:\"name\"`" + `
	union json.RawMessage
}`
		fset, f := parseSource(t, src)
		addUnionTags(findStructType(t, f))
		out := formatFile(t, fset, f)
		if !strings.Contains(out, `json:"-"`) {
			t.Errorf("union field missing json:\"-\" tag:\n%s", out)
		}
	})

	t.Run("does not overwrite existing tag on union field", func(t *testing.T) {
		src := `package p
import "encoding/json"
type S struct {
	union json.RawMessage ` + "`json:\"union\"`" + `
}`
		fset, f := parseSource(t, src)
		addUnionTags(findStructType(t, f))
		out := formatFile(t, fset, f)
		if strings.Contains(out, `json:"-"`) {
			t.Errorf("existing tag was overwritten:\n%s", out)
		}
	})

	t.Run("does not tag non-union fields", func(t *testing.T) {
		src := `package p
type S struct {
	Name string
}`
		fset, f := parseSource(t, src)
		addUnionTags(findStructType(t, f))
		out := formatFile(t, fset, f)
		if strings.Contains(out, "json:") {
			t.Errorf("unexpected tag added to non-union field:\n%s", out)
		}
	})
}

func TestAddOccurrencesField(t *testing.T) {
	src := `package p
type StatusCondition struct {
	Message string ` + "`json:\"message\"`" + `
}`
	fset, f := parseSource(t, src)
	addOccurrencesField(findStructType(t, f))
	out := formatFile(t, fset, f)
	if !strings.Contains(out, "Occurrences") {
		t.Errorf("Occurrences field not added:\n%s", out)
	}
	if !strings.Contains(out, `json:"occurrences"`) {
		t.Errorf("Occurrences field missing json tag:\n%s", out)
	}
}

func TestInjectKubebuilderHeader(t *testing.T) {
	t.Run("inserts block after package types line", func(t *testing.T) {
		src := []byte("package types\n\ntype S struct{}\n")
		out := injectKubebuilderHeader(src)
		if !bytes.Contains(out, []byte("// +kubebuilder:object:generate=true")) {
			t.Errorf("kubebuilder marker missing:\n%s", out)
		}
		idx := bytes.Index(out, []byte("package types\n"))
		markerIdx := bytes.Index(out, []byte("// +kubebuilder"))
		if markerIdx < idx {
			t.Errorf("kubebuilder marker appears before package line")
		}
	})

	t.Run("no-op when package types line absent", func(t *testing.T) {
		src := []byte("package other\n\ntype S struct{}\n")
		out := injectKubebuilderHeader(src)
		if !bytes.Equal(out, src) {
			t.Errorf("src was modified unexpectedly:\n%s", out)
		}
	})
}
