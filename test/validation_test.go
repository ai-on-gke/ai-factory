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

package test

import (
	"bytes"
	"testing"

	planvalidate "github.com/ai-on-gke/ai-factory/tool/pkg/plan/validate"
	specvalidate "github.com/ai-on-gke/ai-factory/tool/pkg/spec/validate"
	"github.com/ai-on-gke/ai-factory/tool/pkg/util"
)

func TestValidateSpecs(t *testing.T) {
	projectRoot, err := util.FindProjectRoot(".")
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	var buf bytes.Buffer
	err = specvalidate.All(&buf, projectRoot)
	if err != nil {
		t.Fatalf("Spec validation failed:\n%s\nError: %v", buf.String(), err)
	}
}

func TestValidatePlans(t *testing.T) {
	projectRoot, err := util.FindProjectRoot(".")
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	var buf bytes.Buffer
	err = planvalidate.All(&buf, projectRoot)
	if err != nil {
		t.Fatalf("Plan validation failed:\n%s\nError: %v", buf.String(), err)
	}
}
