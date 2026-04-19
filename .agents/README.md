# Agents Directory

This directory contains the definitions for agents in the `ai-factory`.
Each agent should have its own subdirectory containing an `agent.md` file that defines its instruction and metadata.

## Structure

```
.agents/
└── <agent-name>/
    └── agent.md
```

## `agent.md` Format

The `agent.md` file should contain a YAML frontmatter section that encodes metadata (like scheduling), followed by the Markdown instructions for the agent.

Example:

```markdown
---
schedule: "every 8 hours"
---
# Agent Instructions
...
```
