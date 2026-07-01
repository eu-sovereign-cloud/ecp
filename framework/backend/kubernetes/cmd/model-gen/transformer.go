package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func transformFile(srcPath, outPath, packageName string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// a. Rename package to the requested package name
	file.Name.Name = packageName

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

	// f. Inject kubebuilder annotations after the "package <name>" line
	out := injectKubebuilderHeader(buf.Bytes(), packageName)

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

// injectKubebuilderHeader inserts kubebuilder marker comments after the "package <name>" line.
func injectKubebuilderHeader(src []byte, packageName string) []byte {
	const block = "\n// +kubebuilder:object:generate=true\n// +kubebuilder:object:root=false\n"
	target := []byte("package " + packageName + "\n")
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

// buildTypeNamesFromFile parses the Go file at path and returns a map of all
// declared type names (TypeSpec) and constant type names (ValueSpec with
// explicit Type that is an *ast.Ident).
func buildTypeNamesFromFile(path string) (map[string]bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	names := map[string]bool{}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				names[s.Name.Name] = true
			case *ast.ValueSpec:
				if s.Type != nil {
					if ident, ok := s.Type.(*ast.Ident); ok {
						names[ident.Name] = true
					}
				}
			}
		}
	}
	return names, nil
}

// collectLocalTypeRefs walks a TypeSpec's type expression recursively and
// returns the names of all identifiers that appear in type positions.
// SelectorExpr nodes (already qualified) are not descended into.
func collectLocalTypeRefs(ts *ast.TypeSpec) []string {
	var refs []string
	var walk func(expr ast.Expr)
	walk = func(expr ast.Expr) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.Ident:
			refs = append(refs, e.Name)
		case *ast.StarExpr:
			walk(e.X)
		case *ast.ArrayType:
			walk(e.Elt)
		case *ast.MapType:
			walk(e.Key)
			walk(e.Value)
		case *ast.StructType:
			if e.Fields == nil {
				return
			}
			for _, field := range e.Fields.List {
				walk(field.Type)
			}
		case *ast.SelectorExpr:
			// Already qualified — do not descend.
		case *ast.InterfaceType:
			// Not descended for local refs.
		}
	}
	walk(ts.Type)
	return refs
}

// computeIncludeSet computes the transitive closure of types reachable from
// rootTypes via local (non-shared) references defined in the file.
func computeIncludeSet(file *ast.File, rootTypes, sharedTypes map[string]bool) map[string]bool {
	// Build name → TypeSpec index.
	typeIndex := map[string]*ast.TypeSpec{}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			typeIndex[ts.Name.Name] = ts
		}
	}

	included := map[string]bool{}
	queue := []string{}
	for name := range rootTypes {
		if !included[name] {
			included[name] = true
			queue = append(queue, name)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		ts, ok := typeIndex[current]
		if !ok {
			continue
		}
		for _, ref := range collectLocalTypeRefs(ts) {
			if sharedTypes[ref] || included[ref] {
				continue
			}
			if _, defined := typeIndex[ref]; defined {
				included[ref] = true
				queue = append(queue, ref)
			}
		}
	}
	return included
}

// filterFileDecls removes declarations not in includeTypes from file.Decls in place.
// For TYPE decls: keep only TypeSpec whose Name is in includeTypes.
// For CONST decls: keep only ValueSpec whose explicit Type ident is in includeTypes.
// GenDecls that become empty are removed entirely.
func filterFileDecls(file *ast.File, includeTypes map[string]bool) {
	filtered := file.Decls[:0]
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			// Keep non-GenDecl nodes (e.g. FuncDecl).
			filtered = append(filtered, decl)
			continue
		}

		switch genDecl.Tok {
		case token.TYPE:
			kept := genDecl.Specs[:0]
			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if ok && includeTypes[ts.Name.Name] {
					kept = append(kept, spec)
				}
			}
			genDecl.Specs = kept
			if len(genDecl.Specs) > 0 {
				filtered = append(filtered, genDecl)
			}

		case token.CONST:
			kept := genDecl.Specs[:0]
			for _, spec := range genDecl.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if vs.Type == nil {
					continue
				}
				ident, ok := vs.Type.(*ast.Ident)
				if ok && includeTypes[ident.Name] {
					kept = append(kept, spec)
				}
			}
			genDecl.Specs = kept
			if len(genDecl.Specs) > 0 {
				filtered = append(filtered, genDecl)
			}

		default:
			filtered = append(filtered, decl)
		}
	}
	file.Decls = filtered
}

// qualifySharedTypeRefs replaces bare Ident references to shared types in type
// positions with SelectorExpr{X: alias, Sel: name}. Returns true if any
// replacement was made and adds the named import to the file.
func qualifySharedTypeRefs(fset *token.FileSet, file *ast.File, sharedTypes map[string]bool, alias, pkgPath string) bool {
	replaced := false

	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		ident, ok := cursor.Node().(*ast.Ident)
		if !ok || !sharedTypes[ident.Name] {
			return true
		}

		// Determine whether this Ident occupies a type position.
		inTypePosition := false
		switch parent := cursor.Parent().(type) {
		case *ast.Field:
			inTypePosition = parent.Type == cursor.Node()
		case *ast.TypeSpec:
			inTypePosition = parent.Type == cursor.Node()
		case *ast.ArrayType:
			inTypePosition = parent.Elt == cursor.Node()
		case *ast.MapType:
			inTypePosition = parent.Key == cursor.Node() || parent.Value == cursor.Node()
		case *ast.StarExpr:
			inTypePosition = parent.X == cursor.Node()
		case *ast.ValueSpec:
			inTypePosition = parent.Type == cursor.Node()
		}

		if !inTypePosition {
			return true
		}

		cursor.Replace(&ast.SelectorExpr{
			X:   &ast.Ident{Name: alias},
			Sel: &ast.Ident{Name: ident.Name},
		})
		replaced = true
		return true
	}, nil)

	if replaced {
		astutil.AddNamedImport(fset, file, alias, pkgPath)
	}
	return replaced
}

// mergeSiblingTypeDecls appends TYPE and CONST declarations from the other .go
// files in srcPath's directory (all part of the same go-sdk schema package)
// into file, so that root types referencing types defined in a sibling file can
// be resolved and inlined. Type specs already declared in file are skipped.
// Sibling files are visited in sorted order (os.ReadDir) for deterministic
// output.
func mergeSiblingTypeDecls(fset *token.FileSet, file *ast.File, srcPath string) error {
	defined := map[string]bool{}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				defined[ts.Name.Name] = true
			}
		}
	}

	dir := filepath.Dir(srcPath)
	base := filepath.Base(srcPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read schema dir: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == base ||
			!strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		sibling, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		for _, decl := range sibling.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || (genDecl.Tok != token.TYPE && genDecl.Tok != token.CONST) {
				continue
			}
			if genDecl.Tok == token.TYPE {
				kept := genDecl.Specs[:0]
				for _, spec := range genDecl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if ok && defined[ts.Name.Name] {
						continue // already declared in the primary file
					}
					if ok {
						defined[ts.Name.Name] = true
					}
					kept = append(kept, spec)
				}
				if len(kept) == 0 {
					continue
				}
				genDecl.Specs = kept
			}
			file.Decls = append(file.Decls, genDecl)
		}
	}
	return nil
}

// transformFileSingle parses a single schema file, filters it to the requested
// root types and their local dependency closure, qualifies shared type refs,
// applies the standard mutations, and writes the result to outPath.
func transformFileSingle(srcPath, outPath, packageName string, rootTypes, sharedTypes map[string]bool, qualifyAlias, qualifyPkgPath string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Merge type/const declarations from sibling files of the same package so
	// that root types referencing types defined in another file (e.g. a
	// SecurityGroupSpec embedding SecurityGroupRuleSpec from
	// security-group-rule.go) resolve. Only declarations reachable from the
	// root types survive the include-set filtering below, so slices without
	// cross-file references are unaffected.
	if err := mergeSiblingTypeDecls(fset, file, srcPath); err != nil {
		return fmt.Errorf("merge sibling decls: %w", err)
	}

	// a. Rename package.
	file.Name.Name = packageName

	// b. Compute transitive include set and filter declarations.
	includeTypes := computeIncludeSet(file, rootTypes, sharedTypes)
	filterFileDecls(file, includeTypes)

	// c. Replace time.Time → metav1.Time.
	if replaceTimeType(file) {
		astutil.DeleteImport(fset, file, "time")
		astutil.AddNamedImport(fset, file, "metav1", "k8s.io/apimachinery/pkg/apis/meta/v1")
		ast.SortImports(fset, file)
	}

	// d. Replace map[string]interface{} → map[string]string.
	replaceMapInterface(file)

	// e. Qualify shared type references.
	qualifySharedTypeRefs(fset, file, sharedTypes, qualifyAlias, qualifyPkgPath)

	// f. Apply standard struct mutations.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		mutateGenDecl(genDecl)
	}

	// Format the AST.
	var buf bytes.Buffer
	if err = format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("format: %w", err)
	}

	// g. Inject kubebuilder annotations.
	out := injectKubebuilderHeader(buf.Bytes(), packageName)

	return os.WriteFile(outPath, out, 0o644) // #nosec G306 -- 0644 is correct for generated source files //nolint:gosec
}
