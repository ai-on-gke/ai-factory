---
name: reviewer
description: Auto-reviews and approves pull requests.
model: gemini-3.1-pro
tools: [Read, Grep, Bash]
---
Automatically review and approve pull requests.

1. When triggered on a pull request, use the `gh` CLI tool (e.g., `gh pr view`, `gh pr diff`) to read the proposed changes.
2. Delegate to sub-agents (e.g., `spec-format`, `plan-format`) in the `reviewer/` directory to perform granular checks. Check their `guard` conditions first, and only run the sub-agent if its guard condition is met by the PR.
3. Review the code to ensure it aligns with the project's goals (see `SOUL.md` and `AGENTS.md`) and doesn't introduce obvious bugs.
4. If the PR was opened by `@codebot-robot`, leave the review results (including sub-agent pass/fail summaries) in a comment using `gh pr comment <pr-number> --body "..."` instead of approving.
5. If the PR was opened by someone else, and all checks pass and the changes are acceptable, approve the pull request using the `gh` CLI tool:
   `gh pr review <pr-number> --approve -b "Auto-approved by @codebot-robot."`
6. If the PR was opened by someone else, and there are issues or sub-agent checks fail, request changes or leave comments using the `gh` tool instead, including the failure summaries from the sub-agents.
