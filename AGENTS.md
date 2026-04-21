# AGENTS.md

This file contains notes and instructions for AI coding agents (like yourself) working on the `ai-factory` project. The overarching goal of this experiment is to achieve **self-assembly** and autonomous evolution.

## Instructions for Agents

1. **Read this file first:** Whenever you start a new task, review this document to understand the project's current state, architecture, and established patterns.
2. **Update this file:** If you make architectural decisions, create new components, or learn something important about the project's setup, **you must update `AGENTS.md`** to share this knowledge with future agents. Self-assembly relies on shared memory.
3. **Use `SOUL.md`:** You'll find a `SOUL.md` file in this repository. Use it to record high-level principles, overarching goals, or "personality" constraints that should guide the ai-factory's evolution.
4. **Organize your thoughts:** Feel free to create other markdown files in a `docs/agents/` or similar directory if you need to organize your memory, thoughts, or ideas for complex tasks.
5. **Components:** Software components intended for installation on Kubernetes are organized under the `components/` directory. Each component should have its own installation logic (e.g., `components/<name>/install`), which can then be invoked by the main `components/install` script.

## Current Architecture

* **Target Environment:** Google Kubernetes Engine (GKE).
* **Component Management:** The `components/` directory contains all sub-components. The master install script is `components/install`.
* **Agent Sandbox:** We are using `agent-sandbox` (from `https://github.com/kubernetes-sigs/agent-sandbox`) installed via the `components/agent-sandbox/install` script. It installs the "extension" manifests (SandboxWarmPool, SandboxClaim, SandboxTemplate). Images are pushed to GCR using `gcr.io/$(gcloud config get project)/`.

6. **Agent Definitions:** Agents are defined in the `.agents/` directory. Each agent has a subdirectory with an `agent.md` file that specifies its instructions and metadata. The file format is Markdown with YAML frontmatter. The frontmatter MUST contain `name` and `description` fields, and should also specify `model` and `tools`. The `name` MUST match the directory name. The body of the file is the system prompt/instructions for the agent. The top-level agent is responsible for scanning and orchestrating these agents.
7. **Event Triggers:** Agents can be triggered by GitHub events. For example, assigning an issue to the robot triggers it to solve the issue. Requesting a PR review from the robot triggers the `reviewer` agent to auto-review and approve the PR.
