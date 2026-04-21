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

package plan

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v2"
)

// Task represents a task in a plan.
type Task struct {
	Name string   `yaml:"name"`
	Spec string   `yaml:"spec"`
	Deps []string `yaml:"deps"`
	Out  []string `yaml:"out"`
}

// Plan represents a parsed plan.
type Plan struct {
	Tasks []Task
}

// Parse parses a plan from a byte slice.
func Parse(data []byte) (*Plan, error) {
	plan := &Plan{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var task Task
		err := decoder.Decode(&task)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode task: %w", err)
		}
		// Basic validation
		if task.Name == "" {
			return nil, fmt.Errorf("task must have a name")
		}
		if task.Spec == "" {
			return nil, fmt.Errorf("task must refer to a spec")
		}
		plan.Tasks = append(plan.Tasks, task)
	}
	return plan, nil
}

// ValidateDAG checks that the tasks form a Directed Acyclic Graph.
func (p *Plan) ValidateDAG() error {
	adj := make(map[string][]string)
	for _, t := range p.Tasks {
		adj[t.Name] = t.Deps
	}

	visited := make(map[string]int) // 0: unvisited, 1: visiting, 2: visited

	var hasCycle func(u string) bool
	hasCycle = func(u string) bool {
		visited[u] = 1
		for _, v := range adj[u] {
			if visited[v] == 1 {
				return true // Cycle found
			}
			if visited[v] == 0 {
				if hasCycle(v) {
					return true
				}
			}
		}
		visited[u] = 2
		return false
	}

	for _, t := range p.Tasks {
		if visited[t.Name] == 0 {
			if hasCycle(t.Name) {
				return fmt.Errorf("cyclic dependency found in tasks starting from %q", t.Name)
			}
		}
	}
	return nil
}

// ValidateAuxiliaryFiles checks that all .md files in the directory match a task name.
func ValidateAuxiliaryFiles(out io.Writer, dir string, projectRoot string, tasks []Task) error {
	taskNames := make(map[string]bool)
	for _, t := range tasks {
		taskNames[t.Name] = true
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
	)

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir, file.Name())
		relPath, err := filepath.Rel(projectRoot, filePath)
		if err != nil {
			relPath = filePath
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(out, "--- FAIL: validate %s\n    failed to read file: %v\n", relPath, err)
			return err
		}

		context := parser.NewContext()
		markdown.Parser().Parse(text.NewReader(data), parser.WithContext(context))
		metaData := meta.Get(context)

		name, ok := metaData["name"].(string)
		if !ok {
			err := fmt.Errorf("auxiliary file %q missing 'name' in frontmatter", file.Name())
			fmt.Fprintf(out, "--- FAIL: validate %s\n    %v\n", relPath, err)
			return err
		}

		expectedName := strings.TrimSuffix(file.Name(), ".md")
		if name != expectedName {
			err := fmt.Errorf("auxiliary file %q name in frontmatter (%q) does not match filename (%q)", file.Name(), name, expectedName)
			fmt.Fprintf(out, "--- FAIL: validate %s\n    %v\n", relPath, err)
			return err
		}

		if !taskNames[name] {
			err := fmt.Errorf("auxiliary file %q refers to unknown task %q", file.Name(), name)
			fmt.Fprintf(out, "--- FAIL: validate %s\n    %v\n", relPath, err)
			return err
		}

		fmt.Fprintf(out, "--- PASS: validate %s\n", relPath)
	}
	return nil
}
