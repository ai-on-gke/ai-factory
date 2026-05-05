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

	"github.com/ai-on-gke/ai-factory/tool/pkg/plan"
	"github.com/ai-on-gke/ai-factory/tool/pkg/spec"
)

// Plan validates a plan directory and returns an error if validation fails.
func Plan(out io.Writer, planDir string, projectRoot string) error {
	// Validate plan.yaml
	planFilePath := filepath.Join(planDir, "plan.yaml")
	relPlanFilePath, err := filepath.Rel(projectRoot, planFilePath)
	if err != nil {
		relPlanFilePath = planFilePath
	}

	data, err := os.ReadFile(planFilePath)
	if err != nil {
		fmt.Fprintf(out, "--- FAIL: validate %s\n    failed to read file: %v\n", relPlanFilePath, err)
		return fmt.Errorf("validation failed")
	}

	p, err := plan.Parse(data)
	if err != nil {
		fmt.Fprintf(out, "--- FAIL: validate %s\n    parse error: %v\n", relPlanFilePath, err)
		return fmt.Errorf("validation failed")
	}

	fmt.Fprintf(out, "--- PASS: validate %s\n", relPlanFilePath)

	// Validate DAG
	if err := p.ValidateDAG(); err != nil {
		fmt.Fprintf(out, "--- FAIL: validate %s\n    DAG error: %v\n", relPlanFilePath, err)
		return fmt.Errorf("validation failed")
	}

	// Validate Spec Existence and Validity
	for _, task := range p.Tasks {
		specPath := filepath.Join(projectRoot, "specs", task.Spec+".md")
		data, err := os.ReadFile(specPath)
		if err != nil {
			fmt.Fprintf(out, "--- FAIL: validate %s\n    failed to read spec %q: %v\n", relPlanFilePath, task.Spec, err)
			return fmt.Errorf("validation failed")
		}

		_, err = spec.Parse(data)
		if err != nil {
			fmt.Fprintf(out, "--- FAIL: validate %s\n    spec %q referenced by task %q is invalid: %v\n", relPlanFilePath, task.Spec, task.Name, err)
			return fmt.Errorf("validation failed")
		}
	}

	// Validate Auxiliary Files
	if err := plan.ValidateAuxiliaryFiles(out, planDir, projectRoot, p.Tasks); err != nil {
		return fmt.Errorf("validation failed")
	}

	return nil
}

// All validates all plans in the project's plans/ directory.
func All(out io.Writer, projectRoot string) error {
	plansDir := filepath.Join(projectRoot, "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read plans directory: %v", err)
	}

	allPassed := true
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		planDir := filepath.Join(plansDir, entry.Name())
		if err := Plan(out, planDir, projectRoot); err != nil {
			allPassed = false
		}
	}

	if !allPassed {
		return fmt.Errorf("plan validation failed")
	}
	return nil
}
