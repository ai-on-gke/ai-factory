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

	pkgvalidate "github.com/ai-on-gke/ai-factory/tool/pkg/plan/validate"
	"github.com/ai-on-gke/ai-factory/tool/pkg/util"
	"github.com/spf13/cobra"
)

// Cmd represents the validate command.
var allFlag bool

func init() {
	Cmd.Flags().BoolVar(&allFlag, "all", false, "Validate all plans in the plans/ directory")
}

// Cmd represents the validate command.
var Cmd = &cobra.Command{
	Use:   "validate [plan-name]",
	Short: "Validates plans",
	Long:  `Validates that plans follow the right format and schema, and checks their dependencies and auxiliary files. Accepts a plan directory name under plans/.`,
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

		planName := args[0]

		planDir := filepath.Join(projectRoot, "plans", planName)

		if err := pkgvalidate.Plan(cmd.OutOrStdout(), planDir, projectRoot); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "PASS")
		return nil
	},
}
