# AI Factory

This project is an experiment in whether a coding agent can self-assemble. In other words, can a coding agent build an unattended coding-agent that can "brainstorm" ideas, open issues, create PRs to fix those issues, merge them automatically, and iterate towards an end state.

This project's end state is "self-hosting": building the coding agent that can perform these tasks autonomously.

We will rely on a Kubernetes cluster (a GKE cluster) and run agents in sandboxes provided by [agent-sandbox](https://github.com/kubernetes-sigs/agent-sandbox). We will initially use `gemini-cli` as our coding agent. We will assume this infrastructure exists initially, but will work towards building the infrastructure to run this experiment purely agentically.

## Repository Structure

- `.agents/`: Definitions and instructions for the various AI agents that operate within this repository (e.g., `builder`, `planner`, `reviewer`, `speccer`). Each agent has its own `agent.md` defining its persona, tools, and goals.
- `components/`: Software components intended for installation on Kubernetes (GKE). Each component has its own installation logic (e.g., `components/<name>/install`), which is invoked by the master `components/install` script. This includes infrastructure like `agent-sandbox` which provides the Kubernetes-native sandboxes where our AI agents execute.
- `tool/`: Go-based CLI tooling used within the repository, including tools for validating plans and specifications.
- `specs/` & `plans/`: Documents generated during the Spec-Driven Development process for complex features.
- `AGENTS.md`: Crucial instructions and architectural details for AI agents operating in this repository. Agents must read and update this file to share knowledge.
- `SOUL.md`: The core principles, goals, and "personality" constraints that guide the AI Factory's autonomous evolution.

## Architecture & Development Process

The AI Factory is designed as a set of Kubernetes-native workloads and operators running on Google Kubernetes Engine (GKE).

We follow a **Spec-Driven Development** process for complex features, handled entirely by interacting agents:
1. **Spec Generation:** The `speccer` agent generates specifications.
2. **Planning:** The `planner` agent creates detailed implementation plans.
3. **Execution:** The `builder` agent writes the code.
4. **Review:** The `reviewer` agent automatically reviews and approves pull requests.

Agents are triggered by GitHub events (e.g., assigning issues, requesting reviews).

## Contributing

This project is licensed under the [Apache 2.0 License](LICENSE).

**Note: We do not expect human contributions in this phase of the experiment.**

We follow [Google's Open Source Community Guidelines](https://opensource.google.com/conduct/).

## Disclaimer

This is not an officially supported Google product.

This project is not eligible for the Google Open Source Software Vulnerability Rewards Program.
Hi, I am codebot-robot
