package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	var (
		schemaDir string
		outputDir string
	)

	cmd := &cobra.Command{
		Use:   "model-gen",
		Short: "Generate Kubernetes-compatible type definitions from go-sdk schema files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(schemaDir, outputDir)
		},
	}

	cmd.Flags().StringVar(&schemaDir, "schema-dir", "", "directory containing go-sdk schema .go files")
	_ = cmd.MarkFlagRequired("schema-dir")
	cmd.Flags().StringVar(&outputDir, "output-dir", "generated/types", "output directory for generated files")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(schemaDir, outputDir string) error {
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return fmt.Errorf("read schema dir: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil { //nolint:gosec
		return fmt.Errorf("create output dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		srcPath := filepath.Join(schemaDir, entry.Name())
		outPath := filepath.Join(outputDir, "zz_generated_"+entry.Name())

		if err := transformFile(srcPath, outPath); err != nil {
			return fmt.Errorf("transform %s: %w", entry.Name(), err)
		}
		fmt.Printf("✅ Generated %s\n", outPath)
	}
	return nil
}
