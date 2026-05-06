package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func transformFile(srcPath, outPath string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// a. Rename package to "types"
	file.Name.Name = "types"

	// b. Replace time.Time → metav1.Time and update imports
	if replaceTimeType(file) {
		astutil.DeleteImport(fset, file, "time")
		astutil.AddNamedImport(fset, file, "metav1", "k8s.io/apimachinery/pkg/apis/meta/v1")
		ast.SortImports(fset, file)
	}

	// c. Replace map[string]interface{} → map[string]string
	replaceMapInterface(file)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		mutateGenDecl(genDecl)
	}

	// Format the AST
	var buf bytes.Buffer
	if err = format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("format: %w", err)
	}

	// f. Inject kubebuilder annotations after the "package types" line
	out := injectKubebuilderHeader(buf.Bytes())

	return os.WriteFile(outPath, out, 0o644) // #nosec G306 -- 0644 is correct for generated source files //nolint:gosec
}

func mutateGenDecl(genDecl *ast.GenDecl) {
	if genDecl == nil {
		return
	}

	// Walk struct types for d/e transformations
	for _, spec := range genDecl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}
		// d. Add json:"-" tag to union fields that have no tag
		addUnionTags(st)

		if ts.Name.Name == "StatusCondition" {
			// e. Add counter field to the StatusCondition struct
			addOccurrencesField(st)
			continue
		}
	}
}

// replaceTimeType replaces all time.Time references with metav1.Time and returns
// true if any replacement was made.
func replaceTimeType(file *ast.File) bool {
	replaced := false
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		sel, ok := cursor.Node().(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "time" || sel.Sel.Name != "Time" {
			return true
		}

		cursor.Replace(
			&ast.SelectorExpr{
				X: &ast.Ident{
					Name: "metav1",
				},
				Sel: &ast.Ident{
					Name: "Time",
				},
			},
		)
		replaced = true

		return true
	}, nil)

	return replaced
}

// replaceMapInterface replaces map[string]interface{} with map[string]string.
func replaceMapInterface(file *ast.File) {
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		m, ok := cursor.Node().(*ast.MapType)
		if !ok {
			return true
		}
		iface, ok := m.Value.(*ast.InterfaceType)
		if !ok || iface.Methods == nil || len(iface.Methods.List) > 0 {
			return true
		}
		m.Value = &ast.Ident{Name: "string"}
		return true
	}, nil)
}

// addUnionTags adds a json:"-" struct tag to any field named "union" that has no tag.
func addUnionTags(st *ast.StructType) {
	for _, field := range st.Fields.List {
		if field.Tag != nil {
			continue
		}
		for _, name := range field.Names {
			if strings.EqualFold(name.Name, "union") {
				field.Tag = &ast.BasicLit{
					Kind:  token.STRING,
					Value: "`json:\"-\"`",
				}
			}
		}
	}
}

// addOccurrencesField appends a `Occurrences int \`json:"-"\“ field to the given struct.
func addOccurrencesField(st *ast.StructType) {
	st.Fields.List = append(st.Fields.List, &ast.Field{
		Names: []*ast.Ident{{Name: "Occurrences"}},
		Type:  &ast.Ident{Name: "int"},
		Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`json:\"occurrences\"`"},
	})
}

// injectKubebuilderHeader inserts kubebuilder marker comments after the "package types" line.
func injectKubebuilderHeader(src []byte) []byte {
	const block = "\n// +kubebuilder:object:generate=true\n// +kubebuilder:object:root=false\n"
	target := []byte("package types\n")
	idx := bytes.Index(src, target)
	if idx < 0 {
		return src
	}
	insert := idx + len(target)
	out := make([]byte, 0, len(src)+len(block))
	out = append(out, src[:insert]...)
	out = append(out, block...)
	out = append(out, src[insert:]...)
	return out
}
