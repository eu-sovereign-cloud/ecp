package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestHasMarker(t *testing.T) {
	cases := []struct {
		name   string
		groups []*ast.CommentGroup
		want   bool
	}{
		{
			name:   "nil slice",
			groups: nil,
			want:   false,
		},
		{
			name:   "empty slice",
			groups: []*ast.CommentGroup{},
			want:   false,
		},
		{
			name:   "nil group element",
			groups: []*ast.CommentGroup{nil},
			want:   false,
		},
		{
			name:   "exact marker",
			groups: []*ast.CommentGroup{{List: []*ast.Comment{{Text: "// +ecp:conditioned"}}}},
			want:   true,
		},
		{
			name:   "marker with space suffix",
			groups: []*ast.CommentGroup{{List: []*ast.Comment{{Text: "// +ecp:conditioned extra"}}}},
			want:   true,
		},
		{
			name:   "marker with colon suffix",
			groups: []*ast.CommentGroup{{List: []*ast.Comment{{Text: "// +ecp:conditioned:key=val"}}}},
			want:   true,
		},
		{
			name:   "marker with equals suffix",
			groups: []*ast.CommentGroup{{List: []*ast.Comment{{Text: "// +ecp:conditioned=value"}}}},
			want:   true,
		},
		{
			name:   "unrelated comment",
			groups: []*ast.CommentGroup{{List: []*ast.Comment{{Text: "// +kubebuilder:object:root=true"}}}},
			want:   false,
		},
		{
			name:   "partial prefix only",
			groups: []*ast.CommentGroup{{List: []*ast.Comment{{Text: "// +ecp:condition"}}}},
			want:   false,
		},
		{
			name: "marker in second group",
			groups: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// unrelated"}}},
				{List: []*ast.Comment{{Text: "// +ecp:conditioned"}}},
			},
			want: true,
		},
		{
			name: "marker as second comment in group",
			groups: []*ast.CommentGroup{{List: []*ast.Comment{
				{Text: "// unrelated"},
				{Text: "// +ecp:conditioned"},
			}}},
			want: true,
		},
		{
			// TrimPrefix("//") is a no-op for text without "//" prefix, so bare
			// marker text also matches — documents implementation behaviour.
			name:   "bare marker text without comment slashes",
			groups: []*ast.CommentGroup{{List: []*ast.Comment{{Text: "+ecp:conditioned"}}}},
			want:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasMarker(tc.groups); got != tc.want {
				t.Errorf("hasMarker() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFindField(t *testing.T) {
	statusVar := types.NewVar(token.NoPos, nil, "Status", types.Typ[types.String])
	specVar := types.NewVar(token.NoPos, nil, "Spec", types.Typ[types.Int])

	empty := types.NewStruct(nil, nil)
	one := types.NewStruct([]*types.Var{statusVar}, nil)
	two := types.NewStruct([]*types.Var{specVar, statusVar}, nil)

	cases := []struct {
		name      string
		s         *types.Struct
		field     string
		wantNil   bool
		wantField string
	}{
		{
			name:    "empty struct",
			s:       empty,
			field:   "Status",
			wantNil: true,
		},
		{
			name:      "found only field",
			s:         one,
			field:     "Status",
			wantField: "Status",
		},
		{
			name:      "found second field",
			s:         two,
			field:     "Status",
			wantField: "Status",
		},
		{
			name:    "nonexistent field",
			s:       one,
			field:   "Conditions",
			wantNil: true,
		},
		{
			name:    "case-sensitive mismatch",
			s:       one,
			field:   "status",
			wantNil: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findField(tc.s, tc.field)
			if tc.wantNil {
				if got != nil {
					t.Errorf("findField() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("findField() = nil, want %q", tc.wantField)
			}
			if got.Name() != tc.wantField {
				t.Errorf("findField().Name() = %q, want %q", got.Name(), tc.wantField)
			}
		})
	}
}

func TestReadHeader(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		got, err := readHeader("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("reads file content", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "header.txt")
		const content = "// Copyright 2025\n"
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := readHeader(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != content {
			t.Errorf("got %q, want %q", got, content)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		if _, err := readHeader("/nonexistent/does/not/exist.txt"); err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestPackageDir(t *testing.T) {
	cases := []struct {
		name            string
		goFiles         []string
		compiledGoFiles []string
		wantDir         string
		wantErr         bool
	}{
		{
			name:    "no files",
			wantErr: true,
		},
		{
			name:    "uses first GoFile",
			goFiles: []string{"/some/pkg/foo.go", "/some/pkg/bar.go"},
			wantDir: "/some/pkg",
		},
		{
			name:            "falls back to CompiledGoFiles",
			compiledGoFiles: []string{"/other/pkg/baz.go"},
			wantDir:         "/other/pkg",
		},
		{
			name:            "GoFiles takes precedence",
			goFiles:         []string{"/primary/foo.go"},
			compiledGoFiles: []string{"/secondary/bar.go"},
			wantDir:         "/primary",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkg := &packages.Package{
				GoFiles:         tc.goFiles,
				CompiledGoFiles: tc.compiledGoFiles,
			}
			got, err := packageDir(pkg)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantDir {
				t.Errorf("packageDir() = %q, want %q", got, tc.wantDir)
			}
		})
	}
}
