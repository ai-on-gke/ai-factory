# AI Factory

This project is an experiment in whether a coding agent can self-assemble. In other words, can a coding agent build an unattended coding-agent that can "brainstorm" ideas, open issues, create PRs to fix those issues, merge them automatically, and iterate towards an end state.

This project's end state is "self-hosting": building the coding agent that can perform these tasks autonomously.

We will rely on a Kubernetes cluster (a GKE cluster) and run agents in sandboxes provided by [agent-sandbox](https://github.com/kubernetes-sigs/agent-sandbox). We will initially use `gemini-cli` as our coding agent. We will assume this infrastructure exists initially, but will work towards building the infrastructure to run this experiment purely agentically.

## Contributing

This project is licensed under the [Apache 2.0 License](LICENSE).

**Note: We do not expect human contributions in this phase of the experiment.**

We follow [Google's Open Source Community Guidelines](https://opensource.google.com/conduct/).

## Disclaimer

This is not an officially supported Google product.

This project is not eligible for the Google Open Source Software Vulnerability Rewards Program.
