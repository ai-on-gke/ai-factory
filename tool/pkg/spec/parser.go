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
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Spec represents a parsed spec file.
type Spec struct {
	Name         string   `yaml:"name"`
	Deps         []string `yaml:"deps"`
	Title        string
	Overview     string
	Goals        []string
	NonGoals     []string
	Requirements string
	Design       string
	Examples     string
	Tests        string
	RawMarkdown  string
}

// Parse parses a spec file from bytes.
func Parse(data []byte) (*Spec, error) {
	markdown := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
	)

	context := parser.NewContext()
	doc := markdown.Parser().Parse(text.NewReader(data), parser.WithContext(context))

	metaData := meta.Get(context)

	spec := &Spec{}
	spec.RawMarkdown = string(data)

	if name, ok := metaData["name"].(string); ok {
		spec.Name = name
	}
	if deps, ok := metaData["deps"].([]interface{}); ok {
		spec.Deps = []string{}
		for _, dep := range deps {
			if depStr, ok := dep.(string); ok {
				spec.Deps = append(spec.Deps, depStr)
			}
		}
	}

	expectedSections := []string{
		"Title",
		"Overview",
		"Goals",
		"Non-Goals",
		"Key Requirements",
		"Design",
		"Examples", // Optional
		"Tests",
	}

	currentSectionIdx := -1

	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		switch node := n.(type) {
		case *ast.Heading:
			text := strings.TrimSpace(headingText(node, data))
			if node.Level == 1 {
				if currentSectionIdx != -1 {
					return nil, fmt.Errorf("unexpected h1 heading %q after start", text)
				}
				spec.Title = text
				currentSectionIdx = 0 // Title
			} else if node.Level == 2 {
				foundIdx := -1
				for i, sec := range expectedSections {
					if text == sec {
						foundIdx = i
						break
					}
				}
				if foundIdx == -1 {
					return nil, fmt.Errorf("unknown section %q", text)
				}

				// Validate order
				if foundIdx <= currentSectionIdx {
					return nil, fmt.Errorf("section %q out of order (current: %s)", text, expectedSections[currentSectionIdx])
				} else if foundIdx > currentSectionIdx+1 {
					// Allow skipping Key Requirements (idx 4) or Examples (idx 6)
					if currentSectionIdx == 3 && foundIdx == 5 {
						// OK: Skipped Key Requirements
					} else if currentSectionIdx == 5 && foundIdx == 7 {
						// OK: Skipped Examples
					} else {
						return nil, fmt.Errorf("skipped required section before %q", text)
					}
				}

				currentSectionIdx = foundIdx
			} else {
				// Level > 2, treat as content
				if currentSectionIdx == -1 {
					return nil, fmt.Errorf("content found before Title")
				}
				appendContent(spec, currentSectionIdx, node, data)
			}
		default:
			if currentSectionIdx == -1 {
				return nil, fmt.Errorf("content found before Title")
			}
			appendContent(spec, currentSectionIdx, node, data)
		}
	}

	// Check if we reached the end and found all required sections.
	// Tests is required and is index 7.
	for i := currentSectionIdx + 1; i < len(expectedSections); i++ {
		if i == 4 || i == 6 { // Key Requirements and Examples are optional
			continue
		}
		return nil, fmt.Errorf("missing required section %q", expectedSections[i])
	}

	return spec, nil
}

func headingText(n *ast.Heading, source []byte) string {
	var buf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
		}
	}
	return buf.String()
}

func nodeRawContent(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	if h, ok := n.(*ast.Heading); ok {
		buf.WriteString(strings.Repeat("#", h.Level))
		buf.WriteByte(' ')
	}
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		buf.Write(segment.Value(source))
	}
	buf.WriteByte('\n')
	return buf.String()
}

func listItems(list *ast.List, source []byte) []string {
	var items []string
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		if li, ok := c.(*ast.ListItem); ok {
			// Extract text from list item
			var buf bytes.Buffer
			for child := li.FirstChild(); child != nil; child = child.NextSibling() {
				lines := child.Lines()
				for i := 0; i < lines.Len(); i++ {
					segment := lines.At(i)
					buf.Write(segment.Value(source))
				}
			}
			items = append(items, strings.TrimSpace(buf.String()))
		}
	}
	return items
}
func appendContent(spec *Spec, idx int, node ast.Node, data []byte) {
	switch idx {
	case 1: // Overview
		spec.Overview += nodeRawContent(node, data)
	case 2: // Goals
		if list, ok := node.(*ast.List); ok {
			spec.Goals = append(spec.Goals, listItems(list, data)...)
		}
	case 3: // Non-Goals
		if list, ok := node.(*ast.List); ok {
			spec.NonGoals = append(spec.NonGoals, listItems(list, data)...)
		}
	case 4: // Key Requirements
		spec.Requirements += nodeRawContent(node, data)
	case 5: // Design
		spec.Design += nodeRawContent(node, data)
	case 6: // Examples
		spec.Examples += nodeRawContent(node, data)
	case 7: // Tests
		spec.Tests += nodeRawContent(node, data)
	}
}
