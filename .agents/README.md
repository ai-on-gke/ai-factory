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

The `agent.md` file should contain a YAML frontmatter section that encodes metadata, followed by the Markdown instructions for the agent.

Example:

```markdown
---
name: my-agent
description: A helpful agent
model: gemini-3.1-pro
tools: [Read, Bash]
---
# Agent Instructions
...
```
