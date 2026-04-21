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

package spec

import (
	"embed"
	"reflect"
	"testing"
)

//go:embed TEMPLATE.md
var templateFS embed.FS

const (
	fmValid = `---
name: myspec
deps:
  - dep1
  - dep2
---
`
	fmNoDeps = `---
name: myspec
---
`
	fmEmptyDeps = `---
name: myspec
deps: []
---
`
	title        = "# My Spec Title\n"
	overview     = "## Overview\nThis is the overview.\n"
	goals        = "## Goals\n- Goal 1\n- Goal 2\n"
	nonGoals     = "## Non-Goals\n- Non-Goal 1\n"
	requirements = "## Key Requirements\nSpecial considerations here.\n"
	design       = "## Design\nDetailed design.\n"
	examples     = "## Examples\nSome examples.\n"
	tests        = "## Tests\nDescribes tests.\n"
)

func TestParse_ValidCases(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		expected *Spec
	}{
		{
			name:     "All sections",
			markdown: fmValid + title + overview + goals + nonGoals + requirements + design + examples + tests,
			expected: &Spec{
				Name:         "myspec",
				Deps:         []string{"dep1", "dep2"},
				Title:        "My Spec Title",
				Overview:     "This is the overview.\n",
				Goals:        []string{"Goal 1", "Goal 2"},
				NonGoals:     []string{"Non-Goal 1"},
				Requirements: "Special considerations here.\n",
				Design:       "Detailed design.\n",
				Examples:     "Some examples.\n",
				Tests:        "Describes tests.\n",
			},
		},
		{
			name:     "No Examples",
			markdown: fmValid + title + overview + goals + nonGoals + requirements + design + tests,
			expected: &Spec{
				Name:         "myspec",
				Deps:         []string{"dep1", "dep2"},
				Title:        "My Spec Title",
				Overview:     "This is the overview.\n",
				Goals:        []string{"Goal 1", "Goal 2"},
				NonGoals:     []string{"Non-Goal 1"},
				Requirements: "Special considerations here.\n",
				Design:       "Detailed design.\n",
				Tests:        "Describes tests.\n",
			},
		},
		{
			name:     "No Key Requirements",
			markdown: fmValid + title + overview + goals + nonGoals + design + examples + tests,
			expected: &Spec{
				Name:     "myspec",
				Deps:     []string{"dep1", "dep2"},
				Title:    "My Spec Title",
				Overview: "This is the overview.\n",
				Goals:    []string{"Goal 1", "Goal 2"},
				NonGoals: []string{"Non-Goal 1"},
				Design:   "Detailed design.\n",
				Examples: "Some examples.\n",
				Tests:    "Describes tests.\n",
			},
		},
		{
			name:     "No Key Requirements, No Examples",
			markdown: fmValid + title + overview + goals + nonGoals + design + tests,
			expected: &Spec{
				Name:     "myspec",
				Deps:     []string{"dep1", "dep2"},
				Title:    "My Spec Title",
				Overview: "This is the overview.\n",
				Goals:    []string{"Goal 1", "Goal 2"},
				NonGoals: []string{"Non-Goal 1"},
				Design:   "Detailed design.\n",
				Tests:    "Describes tests.\n",
			},
		},
		{
			name:     "No Deps in frontmatter",
			markdown: fmNoDeps + title + overview + goals + nonGoals + requirements + design + examples + tests,
			expected: &Spec{
				Name:         "myspec",
				Title:        "My Spec Title",
				Overview:     "This is the overview.\n",
				Goals:        []string{"Goal 1", "Goal 2"},
				NonGoals:     []string{"Non-Goal 1"},
				Requirements: "Special considerations here.\n",
				Design:       "Detailed design.\n",
				Examples:     "Some examples.\n",
				Tests:        "Describes tests.\n",
			},
		},
		{
			name:     "Empty Deps in frontmatter",
			markdown: fmEmptyDeps + title + overview + goals + nonGoals + requirements + design + examples + tests,
			expected: &Spec{
				Name:         "myspec",
				Deps:         []string{},
				Title:        "My Spec Title",
				Overview:     "This is the overview.\n",
				Goals:        []string{"Goal 1", "Goal 2"},
				NonGoals:     []string{"Non-Goal 1"},
				Requirements: "Special considerations here.\n",
				Design:       "Detailed design.\n",
				Examples:     "Some examples.\n",
				Tests:        "Describes tests.\n",
			},
		},
		{
			name:     "Sub-sections allowed",
			markdown: fmValid + title + overview + goals + nonGoals + requirements + "## Design\nDetailed design.\n### Sub-section\nContent\n" + examples + tests,
			expected: &Spec{
				Name:         "myspec",
				Deps:         []string{"dep1", "dep2"},
				Title:        "My Spec Title",
				Overview:     "This is the overview.\n",
				Goals:        []string{"Goal 1", "Goal 2"},
				NonGoals:     []string{"Non-Goal 1"},
				Requirements: "Special considerations here.\n",
				Design:       "Detailed design.\n### Sub-section\nContent\n",
				Examples:     "Some examples.\n",
				Tests:        "Describes tests.\n",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expected.RawMarkdown = tc.markdown
			spec, err := Parse([]byte(tc.markdown))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if !reflect.DeepEqual(spec, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, spec)
			}
		})
	}
}

func TestParse_InvalidCases(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
	}{
		{
			name:     "Missing Title",
			markdown: fmValid + overview + goals + nonGoals + requirements + design + examples + tests,
		},
		{
			name:     "Missing Overview",
			markdown: fmValid + title + goals + nonGoals + requirements + design + examples + tests,
		},
		{
			name:     "Missing Goals",
			markdown: fmValid + title + overview + nonGoals + requirements + design + examples + tests,
		},
		{
			name:     "Missing Non-Goals",
			markdown: fmValid + title + overview + goals + requirements + design + examples + tests,
		},
		{
			name:     "Missing Design",
			markdown: fmValid + title + overview + goals + nonGoals + requirements + examples + tests,
		},
		{
			name:     "Missing Tests",
			markdown: fmValid + title + overview + goals + nonGoals + requirements + design + examples,
		},
		{
			name:     "Wrong order (Goals before Overview)",
			markdown: fmValid + title + goals + overview + nonGoals + requirements + design + examples + tests,
		},
		{
			name:     "Wrong order (Design before Key Requirements)",
			markdown: fmValid + title + overview + goals + nonGoals + design + requirements + examples + tests,
		},
		{
			name:     "Wrong order (Tests before Design)",
			markdown: fmValid + title + overview + goals + nonGoals + requirements + tests + design + examples,
		},
		{
			name:     "Additional H2 section not allowed",
			markdown: fmValid + title + overview + goals + nonGoals + requirements + design + "## Extra Section\nContent\n" + examples + tests,
		},
		{
			name:     "Content before title",
			markdown: fmValid + "Content before title\n" + title + overview + goals + nonGoals + requirements + design + examples + tests,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.markdown))
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
		})
	}
}

func TestParse_Template(t *testing.T) {
	templateContent, err := templateFS.ReadFile("TEMPLATE.md")
	if err != nil {
		t.Fatalf("Failed to read embedded file: %v", err)
	}

	spec, err := Parse(templateContent)
	if err != nil {
		t.Fatalf("Parse template failed: %v", err)
	}

	expected := &Spec{
		Name:         "my-spec-name",
		Deps:         []string{"another-spec-this-depends-on"},
		Title:        "Title",
		Overview:     "A short description of the spec.\n",
		Requirements: "Optional: Particular considerations that should be followed when implementing.\n",
		Design:       "Section that includes detailed design guidance.\n",
		Examples:     "Optional: Few-shot examples to guide implementation.\n",
		Tests:        "Describes tests that must exist to validate the spec.\n",
		RawMarkdown:  string(templateContent),
	}

	if !reflect.DeepEqual(spec, expected) {
		t.Errorf("Template parsed spec does not match expected.\nExpected: %+v\nGot: %+v", expected, spec)
	}
}
