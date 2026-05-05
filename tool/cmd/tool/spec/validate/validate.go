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
	"path/filepath"
	"strings"

	pkgvalidate "github.com/ai-on-gke/ai-factory/tool/pkg/spec/validate"
	"github.com/ai-on-gke/ai-factory/tool/pkg/util"
	"github.com/spf13/cobra"
)

// Cmd represents the validate command.
var allFlag bool

func init() {
	Cmd.Flags().BoolVar(&allFlag, "all", false, "Validate all specs in the specs/ directory")
}

// Cmd represents the validate command.
var Cmd = &cobra.Command{
	Use:   "validate [spec-name]",
	Short: "Validates specs",
	Long:  `Validates that specs follow the right format and schema, and checks their dependencies. Accepts a spec name or a relative path under specs/.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if allFlag {
			if len(args) > 0 {
				return fmt.Errorf("accepts 0 arg(s) when --all is used, received %d", len(args))
			}
			return nil
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := util.FindProjectRoot(".")
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}

		if allFlag {
			if err := pkgvalidate.All(cmd.OutOrStdout(), projectRoot); err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "FAIL")
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "PASS")
			return nil
		}

		name := args[0]

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

		pkgvalidate.Spec(cmd.OutOrStdout(), filePath, projectRoot, visited, results)

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
