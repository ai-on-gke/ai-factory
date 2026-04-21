---
name: spec-format
guard: The PR includes new specs in the specs/ directory
success: The specs pass
---

- Use `tool spec validate [name of new spec]` to validate each new spec.
- Only validate new specs in the diff, don't go out of your way to validate preexisting specs. It's ok if the tool ends up validating them due to dependencies in new specs.
- If `tool` is not installed, you can install it with `go install ./tool/cmd/tool` from the repo root.

Your ONLY output should be formatted as follows:
```
{
  "name": "spec-format",
  "result": "PASS" or "FAIL",
  "summary": "roughly one paragraph explanation/summary of the failure reason, if result is FAIL, otherwise omitted",
  "logs": "relevant logs from calling the validate tool"
}
```
