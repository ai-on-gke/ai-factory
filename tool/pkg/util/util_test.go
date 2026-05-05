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

package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock project structure
	// tmpDir/
	//   spec.yaml
	//   subdir1/
	//     subdir2/

	if err := os.WriteFile(filepath.Join(tmpDir, "spec.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "subdir1", "subdir2"), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		startDir string
		expected string
	}{
		{
			name:     "from root",
			startDir: tmpDir,
			expected: tmpDir,
		},
		{
			name:     "from subdir1",
			startDir: filepath.Join(tmpDir, "subdir1"),
			expected: tmpDir,
		},
		{
			name:     "from subdir2",
			startDir: filepath.Join(tmpDir, "subdir1", "subdir2"),
			expected: tmpDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := FindProjectRoot(tt.startDir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if root != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, root)
			}
		})
	}
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Without spec.yaml, FindProjectRoot returns current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	root, err := FindProjectRoot(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != cwd {
		t.Errorf("expected %q, got %q", cwd, root)
	}
}
