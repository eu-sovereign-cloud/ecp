package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultQualifyPkgPath  = "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	defaultQualifyPkgAlias = "schemav1"
)

// hardcodedSharedTypes are always treated as shared regardless of --shared-types-source.
var hardcodedSharedTypes = []string{
	"Cidr", "IPVersion", "Zone", "VolumeReference", "VolumeReferenceType",
}

func main() {
	var (
		schemaDir         string
		outputDir         string
		packageName       string
		schemaFile        string
		outputFile        string
		rootTypesFlag     string
		sharedTypesSource string
		qualifyPkgPath    string
		qualifyPkgAlias   string
	)

	cmd := &cobra.Command{
		Use:   "model-gen",
		Short: "Generate Kubernetes-compatible type definitions from go-sdk schema files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if schemaFile != "" && schemaDir != "" {
				return fmt.Errorf("--schema-file and --schema-dir are mutually exclusive")
			}
			if schemaFile != "" {
				return runSingle(schemaFile, outputFile, packageName, rootTypesFlag, sharedTypesSource, qualifyPkgAlias, qualifyPkgPath)
			}
			return run(schemaDir, outputDir, packageName)
		},
	}

	// Directory-mode flags
	cmd.Flags().StringVar(&schemaDir, "schema-dir", "", "directory containing go-sdk schema .go files")
	cmd.Flags().StringVar(&outputDir, "output-dir", "generated/types", "output directory for generated files")

	// Single-file-mode flags
	cmd.Flags().StringVar(&schemaFile, "schema-file", "", "path to a single schema .go file (alternative to --schema-dir)")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "path to a single output file (alternative to --output-dir)")
	cmd.Flags().StringVar(&rootTypesFlag, "root-types", "", "comma-separated type names to generate (e.g. BlockStorageSpec,BlockStorageStatus)")
	cmd.Flags().StringVar(&sharedTypesSource, "shared-types-source", "", "path to a file whose types are treated as shared (qualified with external pkg alias)")
	cmd.Flags().StringVar(&qualifyPkgPath, "qualify-pkg-path", defaultQualifyPkgPath, "import path for the shared types package")
	cmd.Flags().StringVar(&qualifyPkgAlias, "qualify-pkg-alias", defaultQualifyPkgAlias, "alias for the shared types package")

	// Common flags
	cmd.Flags().StringVar(&packageName, "package-name", "types", "Go package name for generated files")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(schemaDir, outputDir, packageName string) error {
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return fmt.Errorf("read schema dir: %w", err)
	}

	err = os.MkdirAll(outputDir, 0755) // #nosec G301 -- 0755 is correct for generated directories //nolint:gosec
	if err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		srcPath := filepath.Join(schemaDir, entry.Name())
		outPath := filepath.Join(outputDir, "zz_generated_"+entry.Name())

		if err := transformFile(srcPath, outPath, packageName); err != nil {
			return fmt.Errorf("transform %s: %w", entry.Name(), err)
		}
		fmt.Printf("✅ Generated %s\n", outPath)
	}
	return nil
}

func runSingle(schemaFile, outputFile, packageName, rootTypesFlag, sharedTypesSource, qualifyAlias, qualifyPkgPath string) error {
	if outputFile == "" {
		base := filepath.Base(schemaFile)
		outputFile = "zz_generated_" + base
	}

	// No root-types filtering: fall back to the existing whole-file transform.
	if rootTypesFlag == "" {
		return transformFile(schemaFile, outputFile, packageName)
	}

	// Build root types set.
	rootTypes := map[string]bool{}
	for name := range strings.SplitSeq(rootTypesFlag, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			rootTypes[name] = true
		}
	}

	// Build shared types set.
	sharedTypes := map[string]bool{}
	for _, name := range hardcodedSharedTypes {
		sharedTypes[name] = true
	}
	if sharedTypesSource != "" {
		extra, err := buildTypeNamesFromFile(sharedTypesSource)
		if err != nil {
			return fmt.Errorf("read shared-types-source: %w", err)
		}
		for name := range extra {
			sharedTypes[name] = true
		}
	}

	return transformFileSingle(schemaFile, outputFile, packageName, rootTypes, sharedTypes, qualifyAlias, qualifyPkgPath)
}
