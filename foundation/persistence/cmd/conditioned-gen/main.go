package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var (
		paths       []string
		headerFile  string
		outFilename string
	)

	cmd := &cobra.Command{
		Use:   "conditioned-gen",
		Short: "Generate Conditioned interface methods for +ecp:conditioned-marked types",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(paths) == 0 {
				return fmt.Errorf("at least one path must be specified")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			g, err := newGenerator(headerFile, outFilename)
			if err != nil {
				return err
			}
			return g.run(paths)
		},
	}

	cmd.Flags().StringSliceVar(&paths, "paths", nil,
		"package path pattern(s) to process; may be repeated or comma-separated (e.g. ./v1/...)")
	_ = cmd.MarkFlagRequired("paths")

	cmd.Flags().StringVar(&headerFile, "header-file", "",
		"path to a boilerplate header file to prepend to generated files")

	cmd.Flags().StringVar(&outFilename, "output-filename", "zz_generated.conditions.go",
		"name of the generated file written into each target package")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
