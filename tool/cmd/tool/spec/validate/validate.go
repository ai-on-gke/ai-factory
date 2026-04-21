//  Copyright 2026 Google LLC
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package validate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-on-gke/ai-factory/tool/pkg/spec"
	"github.com/spf13/cobra"
)

// Cmd represents the validate command.
var Cmd = &cobra.Command{
	Use:   "validate [spec-name]",
	Short: "Validates specs",
	Long:  `Validates that specs follow the right format and schema, and checks their dependencies. Accepts a spec name or a relative path under specs/.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		
		projectRoot, err := findProjectRoot(".")
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}

		if !strings.HasSuffix(name, ".md") {
			name = name + ".md"
		}

		var filePath string
		if strings.Contains(name, "/") {
			// It's a path, check safety
			if filepath.IsAbs(name) || strings.HasPrefix(name, "..") || strings.Contains(name, "../") {
				return fmt.Errorf("invalid path %q. Only relative paths under specs/ are allowed, no '..' or absolute paths", name)
			}
			filePath = filepath.Join(projectRoot, "specs", name)
		} else {
			filePath = filepath.Join(projectRoot, "specs", name)
		}

		visited := make(map[string]bool)
		results := make(map[string]bool)

		validateSpec(cmd.OutOrStdout(), filePath, projectRoot, visited, results)

		allPassed := true
		for _, passed := range results {
			if !passed {
				allPassed = false
				break
			}
		}

		if allPassed {
			fmt.Fprintln(cmd.OutOrStdout(), "PASS")
			return nil
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "FAIL")
			return fmt.Errorf("validation failed")
		}
	},
}

func findProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "spec.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Fallback to current directory if spec.yaml is not found anywhere
			cwd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			return cwd, nil
		}
		dir = parent
	}
}

func validateSpec(out io.Writer, filePath string, projectRoot string, visited map[string]bool, results map[string]bool) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Fprintf(out, "--- FAIL: validate %s\n    failed to get absolute path: %v\n", filePath, err)
		results[filePath] = false
		return
	}

	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		relPath = absPath
	}

	if visited[absPath] {
		return
	}
	visited[absPath] = true

	data, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Fprintf(out, "--- FAIL: validate %s\n    failed to read file: %v\n", relPath, err)
		results[relPath] = false
		return
	}

	s, err := spec.Parse(data)
	if err != nil {
		fmt.Fprintf(out, "--- FAIL: validate %s\n    parse error: %v\n", relPath, err)
		results[relPath] = false
		return
	}

	// Check that s.Name matches the filename without .md
	expectedName := strings.TrimSuffix(filepath.Base(absPath), ".md")
	if s.Name != expectedName {
		fmt.Fprintf(out, "--- FAIL: validate %s\n    name in frontmatter (%q) does not match filename (%q)\n", relPath, s.Name, expectedName)
		results[relPath] = false
		return
	}

	fmt.Fprintf(out, "--- PASS: validate %s\n", relPath)
	results[relPath] = true

	for _, dep := range s.Deps {
		depPath := filepath.Join(projectRoot, "specs", dep+".md")
		validateSpec(out, depPath, projectRoot, visited, results)
	}
}
