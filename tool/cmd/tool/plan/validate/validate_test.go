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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func TestValidatePlan_Success(t *testing.T) {
	tmpDir := t.TempDir()

	if err := copyDir("testdata/success", tmpDir); err != nil {
		t.Fatal(err)
	}

	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origCwd)

	Cmd.SetArgs([]string{"2026-04-20_test-plan"}) // Plan name
	var out bytes.Buffer
	Cmd.SetOut(&out)

	err = Cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := out.String()
	t.Logf("Output:\n%s", output)

	expectedLogs := []string{
		"--- PASS: validate plans/2026-04-20_test-plan/plan.yaml",
		"--- PASS: validate plans/2026-04-20_test-plan/task-a.md",
		"PASS",
	}

	for _, log := range expectedLogs {
		if !strings.Contains(output, log) {
			t.Errorf("Expected log %q not found in output", log)
		}
	}
}

func TestValidatePlan_Cycle(t *testing.T) {
	tmpDir := t.TempDir()

	if err := copyDir("testdata/cycle", tmpDir); err != nil {
		t.Fatal(err)
	}

	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origCwd)

	Cmd.SetArgs([]string{"2026-04-20_cycle-plan"}) // Plan name
	var out bytes.Buffer
	Cmd.SetOut(&out)

	err = Cmd.Execute()
	if err == nil {
		t.Fatal("Expected command to fail, but it succeeded")
	}

	output := out.String()
	t.Logf("Output:\n%s", output)

	if !strings.Contains(output, "DAG error: cyclic dependency found in tasks") {
		t.Errorf("Expected DAG error message not found in output")
	}
	if !strings.Contains(output, "FAIL") {
		t.Errorf("Expected 'FAIL' in output")
	}
}
