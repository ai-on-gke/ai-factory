// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-on-gke/ai-factory/tool/pkg/spec"
)

// Spec validates a spec file and its dependencies.
func Spec(out io.Writer, filePath string, projectRoot string, visited map[string]bool, results map[string]bool) {
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
		Spec(out, depPath, projectRoot, visited, results)
	}
}

// All validates all specs in the project's specs/ directory.
func All(out io.Writer, projectRoot string) error {
	specsDir := filepath.Join(projectRoot, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read specs directory: %v", err)
	}

	allPassed := true
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if strings.EqualFold(entry.Name(), "readme.md") || strings.EqualFold(entry.Name(), "template.md") {
			continue
		}

		visited := make(map[string]bool)
		results := make(map[string]bool)
		filePath := filepath.Join(specsDir, entry.Name())

		Spec(out, filePath, projectRoot, visited, results)

		for _, passed := range results {
			if !passed {
				allPassed = false
				break
			}
		}
	}

	if !allPassed {
		return fmt.Errorf("spec validation failed")
	}
	return nil
}
