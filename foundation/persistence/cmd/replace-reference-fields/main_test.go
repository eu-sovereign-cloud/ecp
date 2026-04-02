package main

import "testing"

func TestReplaceReferences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "replaces Reference in struct fields",
			input: `package foo

type BlockStorage struct {
	Sku  Reference ` + "`" + `json:"sku"` + "`" + `
	Pool Reference ` + "`" + `json:"pool"` + "`" + `
}`,
			want: `package foo

type BlockStorage struct {
	Sku  ReferenceObject ` + "`" + `json:"sku"` + "`" + `
	Pool ReferenceObject ` + "`" + `json:"pool"` + "`" + `
}`,
		},
		{
			name: "does not replace Reference in type declaration",
			input: `package foo

type Reference struct {
	ID string
}`,
			want: `package foo

type Reference struct {
	ID string
}`,
		},
		{
			name: "does not replace Reference in comments",
			input: `package foo

type Foo struct {
	// Reference is the link
	Sku Reference
}`,
			want: `package foo

type Foo struct {
	// Reference is the link
	Sku ReferenceObject
}`,
		},
		{
			name: "does not replace Reference outside structs",
			input: `package foo

var x Reference

type Foo struct {
	Sku Reference
}`,
			want: `package foo

var x Reference

type Foo struct {
	Sku ReferenceObject
}`,
		},
		{
			name: "handles multiple structs",
			input: `package foo

type A struct {
	F1 Reference
}

type B struct {
	F2 Reference
}`,
			want: `package foo

type A struct {
	F1 ReferenceObject
}

type B struct {
	F2 ReferenceObject
}`,
		},
		{
			name:  "skips empty lines and closing braces",
			input: "package foo\n\ntype A struct {\n\n\tF1 Reference\n\n}",
			want:  "package foo\n\ntype A struct {\n\n\tF1 ReferenceObject\n\n}",
		},
		{
			name: "skips union keyword",
			input: `package foo

type A struct {
	union
	F1 Reference
}`,
			want: `package foo

type A struct {
	union
	F1 ReferenceObject
}`,
		},
		{
			name: "does not replace partial matches like MyReference",
			input: `package foo

type A struct {
	F1 MyReference
	F2 ReferenceID
	F3 Reference
}`,
			want: `package foo

type A struct {
	F1 MyReference
	F2 ReferenceID
	F3 ReferenceObject
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceReferences(tt.input)
			if got != tt.want {
				t.Errorf("replaceReferences():\ngot:\n%s\n\nwant:\n%s", got, tt.want)
			}
		})
	}
}
