# Plan File Format

This directory contains the parser for the plan file format used by the `tool plan` command.
The plan file format is a sequence of YAML documents separated by `---`.

## File Structure

A plan file named `plan.yaml` must be located in a directory named like `plans/YYYY-MM-DD_plan-name/` in the project root.

Each file contains one or more YAML task objects.

### Task Object Fields

Each task object must have the following fields:

- `name`: A short string identifying the task.
- `spec`: The name of the single spec this task is related to.
- `deps`: A list of other task names this task depends on. This must form a Directed Acyclic Graph (DAG).
- `out`: A list of files (relative paths from the project root) that must be created or modified.

Example:
```yaml
name: task-a
spec: my-spec
deps: []
out: [file1.go]
---
name: task-b
spec: my-spec
deps: [task-a]
out: [file2.go]
```

### Auxiliary Files

Optionally, the plan directory can contain auxiliary `task-name.md` files.
These files provide a brief paragraph or so of guidance on the task.
The frontmatter for these files must contain a `name` field that must match both the filename and the name of the related task.

## Usage

To parse a plan file, use the `Parse` function:

```go
import "github.com/ai-on-gke/ai-factory/tool/pkg/plan"

// ...

data, err := os.ReadFile("plan.yaml")
if err != nil {
    // handle error
}

p, err := plan.Parse(data)
if err != nil {
    // handle error
}

fmt.Printf("Parsed Plan: %+v\n", p)
```
