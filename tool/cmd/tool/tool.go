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

// tool is the toolbox for the factory
package main

import (
	"fmt"
	"os"

	"github.com/ai-on-gke/ai-factory/tool/cmd/tool/plan"
	"github.com/ai-on-gke/ai-factory/tool/cmd/tool/spec"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tool",
	Short: "tool is the toolbox for the factory",
	Long:  `tool is the toolbox for the factory`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func main() {
	rootCmd.AddCommand(spec.Cmd)
	rootCmd.AddCommand(plan.Cmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
