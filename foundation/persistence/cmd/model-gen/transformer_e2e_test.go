package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden files")

// testOutFile registers a unique output path inside dir and removes it on cleanup.
func testOutFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}

func TestTransformFile_Valid(t *testing.T) {
	const (
		srcPath    = "testdata/valid/schema.go"
		goldenPath = "testdata/valid/zz_generated_schema.go.golden"
	)
	outPath := testOutFile(t, "testdata/valid", "zz_test_schema.go")

	if err := transformFile(srcPath, outPath); err != nil {
		t.Fatalf("transformFile: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if *update {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with -update to create it): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("generated output differs from golden\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRun_Valid(t *testing.T) {
	outDir := t.TempDir()
	const goldenPath = "testdata/valid/zz_generated_schema.go.golden"

	if err := run("testdata/valid", outDir); err != nil {
		t.Fatalf("run: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, "zz_generated_schema.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("run output differs from golden\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTransformFile_ParseError(t *testing.T) {
	src := testOutFile(t, t.TempDir(), "bad.go")
	if err := os.WriteFile(src, []byte("this is not valid go"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := testOutFile(t, t.TempDir(), "out.go")
	err := transformFile(src, out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRun_MissingDir(t *testing.T) {
	err := run("/nonexistent/does/not/exist", t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
