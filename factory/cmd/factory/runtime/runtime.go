// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"github.com/ai-on-gke/ai-factory/factory/cmd/factory/runtime/proxy"
	"github.com/spf13/cobra"
)

// Cmd represents the runtime command.
var Cmd = &cobra.Command{
	Use:   "runtime",
	Short: "Manage the factory runtime",
	Long:  `Subcommands for managing the factory runtime.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(proxy.Cmd)
}
