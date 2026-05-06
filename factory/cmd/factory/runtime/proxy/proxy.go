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

package proxy

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"

	proxycore "github.com/ai-on-gke/ai-factory/factory/pkg/runtime/proxy"
)

// Cmd represents the proxy command
var Cmd = &cobra.Command{
	Use:                "proxy",
	Short:              "Run the egress reverse proxy",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		fs := flag.NewFlagSet("proxy", flag.ContinueOnError)
		fs.SetOutput(io.Discard) // prevent default error printing so we can print custom usage

		configPath := fs.String("config", "", "path to config file")

		err := fs.Parse(args)
		if err != nil || *configPath == "" || len(fs.Args()) > 0 {
			fmt.Println("Usage: factory runtime proxy --config <path>")
			os.Exit(1)
		}

		cfg, err := proxycore.LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		server := proxycore.NewServer(cfg)
		ctx := cmd.Context()
		if err := server.Start(ctx); err != nil {
			log.Fatalf("Proxy server failed: %v", err)
		}
	},
}
