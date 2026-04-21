---
name: plan-format
guard: The PR includes new plans in the plans/ directory
success: The plans pass validation
---

- Use `tool plan validate [plan-name]` to validate each new plan, where `plan-name` is the name of the directory inside `plans/` (e.g., `2026-04-20_test-plan`).
- Only validate new plans in the diff, don't go out of your way to validate preexisting plans.
- If `tool` is not installed, you can install it with `go install ./tool/cmd/tool` from the repo root.

Your ONLY output should be formatted as follows:
```
{
  "name": "plan-format",
  "result": "PASS" or "FAIL",
  "summary": "roughly one paragraph explanation/summary of the failure reason, if result is FAIL, otherwise omitted",
  "logs": "relevant logs from calling the validate tool"
}
```
