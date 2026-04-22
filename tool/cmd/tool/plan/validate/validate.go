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
	"os"
	"path/filepath"

	"github.com/ai-on-gke/ai-factory/tool/pkg/plan"
	"github.com/ai-on-gke/ai-factory/tool/pkg/spec"
	"github.com/spf13/cobra"
)

// Cmd represents the validate command.
var Cmd = &cobra.Command{
	Use:   "validate [plan-name]",
	Short: "Validates plans",
	Long:  `Validates that plans follow the right format and schema, and checks their dependencies and auxiliary files. Accepts a plan directory name under plans/.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planName := args[0]

		projectRoot, err := findProjectRoot(".")
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}

		planDir := filepath.Join(projectRoot, "plans", planName)

		// Validate plan.yaml
		planFilePath := filepath.Join(planDir, "plan.yaml")
		relPlanFilePath, err := filepath.Rel(projectRoot, planFilePath)
		if err != nil {
			relPlanFilePath = planFilePath
		}

		data, err := os.ReadFile(planFilePath)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "--- FAIL: validate %s\n    failed to read file: %v\n", relPlanFilePath, err)
			return fmt.Errorf("validation failed")
		}

		p, err := plan.Parse(data)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "--- FAIL: validate %s\n    parse error: %v\n", relPlanFilePath, err)
			return fmt.Errorf("validation failed")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "--- PASS: validate %s\n", relPlanFilePath)

		// Validate DAG
		if err := p.ValidateDAG(); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "--- FAIL: validate %s\n    DAG error: %v\n", relPlanFilePath, err)
			return fmt.Errorf("validation failed")
		}

		// Validate Spec Existence and Validity
		for _, task := range p.Tasks {
			specPath := filepath.Join(projectRoot, "specs", task.Spec+".md")
			data, err := os.ReadFile(specPath)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "--- FAIL: validate %s\n    failed to read spec %q: %v\n", relPlanFilePath, task.Spec, err)
				return fmt.Errorf("validation failed")
			}

			_, err = spec.Parse(data)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "--- FAIL: validate %s\n    spec %q referenced by task %q is invalid: %v\n", relPlanFilePath, task.Spec, task.Name, err)
				return fmt.Errorf("validation failed")
			}
		}

		// Validate Auxiliary Files
		if err := plan.ValidateAuxiliaryFiles(cmd.OutOrStdout(), planDir, projectRoot, p.Tasks); err != nil {
			return fmt.Errorf("validation failed")
		}

		fmt.Fprintln(cmd.OutOrStdout(), "PASS")
		return nil
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
