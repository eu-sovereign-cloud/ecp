package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden files")

// testOutFile registers a unique output filename for the given package directory
// and cleans up the generated file after the test completes.
func testOutFile(t *testing.T, pkgDir string) string {
	t.Helper()
	safe := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	name := "zz_test_" + safe + ".go"
	t.Cleanup(func() { _ = os.Remove(filepath.Join(pkgDir, name)) })
	return name
}

// captureLog redirects the default logger's output to a buffer for the duration
// of the test, restoring it on cleanup.
func captureLog(t *testing.T) *strings.Builder {
	t.Helper()
	var buf strings.Builder
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })
	return &buf
}

func TestRun_ValidPackage(t *testing.T) {
	const pkgDir = "testdata/valid"
	outName := testOutFile(t, pkgDir)

	g, err := newGenerator("", outName)
	if err != nil {
		t.Fatalf("newGenerator: %v", err)
	}
	if err := g.run([]string{"./testdata/valid"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(pkgDir, outName))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}

	const goldenPath = "testdata/valid/zz_generated.conditions.golden"
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
	if string(got) != string(want) {
		t.Errorf("generated output differs from golden\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRun_NoMarker(t *testing.T) {
	const pkgDir = "testdata/no-marker"
	outName := testOutFile(t, pkgDir)

	g, err := newGenerator("", outName)
	if err != nil {
		t.Fatalf("newGenerator: %v", err)
	}
	if err := g.run([]string{"./testdata/no-marker"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, outName)); err == nil {
		t.Errorf("expected no output file, but %s exists", filepath.Join(pkgDir, outName))
	}
}

func TestRun_NoStatus(t *testing.T) {
	outName := testOutFile(t, "testdata/no-status")
	logBuf := captureLog(t)

	g, err := newGenerator("", outName)
	if err != nil {
		t.Fatalf("newGenerator: %v", err)
	}
	err = g.run([]string{"./testdata/no-status"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "one or more packages failed") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(logBuf.String(), "no Status field") {
		t.Errorf("expected log to contain 'no Status field', got: %s", logBuf.String())
	}
}

func TestRun_WrongStatusPkg(t *testing.T) {
	outName := testOutFile(t, "testdata/wrong-status-pkg")
	logBuf := captureLog(t)

	g, err := newGenerator("", outName)
	if err != nil {
		t.Fatalf("newGenerator: %v", err)
	}
	err = g.run([]string{"./testdata/wrong-status-pkg"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "one or more packages failed") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(logBuf.String(), "status field type must live in") {
		t.Errorf("expected log to mention wrong package, got: %s", logBuf.String())
	}
}

func TestRun_NonStruct(t *testing.T) {
	outName := testOutFile(t, "testdata/non-struct")
	logBuf := captureLog(t)

	g, err := newGenerator("", outName)
	if err != nil {
		t.Fatalf("newGenerator: %v", err)
	}
	err = g.run([]string{"./testdata/non-struct"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "one or more packages failed") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(logBuf.String(), "non-struct type") {
		t.Errorf("expected log to mention non-struct type, got: %s", logBuf.String())
	}
}

func TestRun_WithHeader(t *testing.T) {
	const pkgDir = "testdata/valid"
	outName := testOutFile(t, pkgDir)

	headerFile := filepath.Join(t.TempDir(), "header.txt")
	const headerContent = "// Copyright (c) 2025 Test\n"
	if err := os.WriteFile(headerFile, []byte(headerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := newGenerator(headerFile, outName)
	if err != nil {
		t.Fatalf("newGenerator: %v", err)
	}
	if err := g.run([]string{"./testdata/valid"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(pkgDir, outName))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if !strings.Contains(string(got), headerContent) {
		t.Errorf("generated file does not contain header %q", headerContent)
	}
}
