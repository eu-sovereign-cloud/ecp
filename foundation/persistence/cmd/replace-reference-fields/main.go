// replace-reference-fields replaces Reference field types with ReferenceObject
// only within struct definitions in generated Go files.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	structDeclRe = regexp.MustCompile(`^type\s+\w+\s+struct\b`)
	referenceRe  = regexp.MustCompile(`(?:^|[^\w])Reference(?:[^\w]|$)`)
	replaceRe    = regexp.MustCompile(`\bReference\b`)
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: replace-reference-fields <file>")
		os.Exit(1)
	}

	path := os.Args[1]
	data, err := os.ReadFile(path) //nolint:gosec // path comes from CLI args
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
		os.Exit(1)
	}

	lines := strings.Split(string(data), "\n")
	withinStruct := false
	braceDepth := 0

	for i, line := range lines {
		stripped := strings.TrimSpace(line)

		if !withinStruct {
			if structDeclRe.MatchString(stripped) {
				withinStruct = true
				braceDepth = strings.Count(stripped, "{") - strings.Count(stripped, "}")
				if braceDepth <= 0 {
					withinStruct = false
				}
			}
			continue
		}

		braceDepth += strings.Count(stripped, "{")
		braceDepth -= strings.Count(stripped, "}")

		if !strings.HasPrefix(stripped, "//") && stripped != "" && stripped != "union" && stripped != "}" {
			if referenceRe.MatchString(line) {
				lines[i] = replaceRe.ReplaceAllString(line, "ReferenceObject")
			}
		}

		if braceDepth <= 0 {
			withinStruct = false
		}
	}

	result := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(result), 0o600); err != nil { //nolint:gosec // path comes from CLI args
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
		os.Exit(1)
	}
}
