# Spec File Format

This directory contains the parser for the spec file format used for the `tool spec` command.
The spec file format is Markdown with YAML frontmatter.

## File Structure

A spec file must have the following structure:

### Frontmatter

The file must start with a YAML frontmatter block containing the following fields:

- `name`: The name of the spec (string).
- `deps`: A list of other spec names this spec depends on (list of strings). These are relative paths in the project root's `specs/` directory.

Example:
```yaml
---
name: my-feature-spec
deps:
  - core-spec
  - auth-spec
---
```

### Markdown Sections

After the frontmatter, the file must contain the following sections in **exactly** this order:

1.  **`# Title`**: The title of the spec (H1 header).
2.  **`## Overview`**: A short description of the spec.
3.  **`## Goals`**: A bulleted list of design goals.
4.  **`## Non-Goals`**: A bulleted list of anti-goals.
5.  **`## Key Requirements`** *(Optional)*: Particular considerations that should be followed when implementing.
6.  **`## Design`**: Section that includes detailed design guidance.
7.  **`## Examples`** *(Optional)*: Few-shot examples to guide implementation.
8.  **`## Tests`**: Describes tests that must exist to validate the spec.

## Usage

To parse a spec file, use the `Parse` function provided by the `parser` package:

```go
import "github.com/ai-on-gke/SubstrATE/hack/pkg/spec/parser"

// ...

data, err := os.ReadFile("myspec.md")
if err != nil {
    // handle error
}

spec, err := parser.Parse(data)
if err != nil {
    // handle error
}

fmt.Printf("Parsed Spec: %+v\n", spec)
```
