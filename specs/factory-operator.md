---
name: factory-operator
deps:
  - reverse-proxy
  - agent-image
---

# Factory Operator

## Overview

A Kubernetes controller that implements a CRD-based interface for controlling the factory. It launches Pods running an agent harness to execute work.

## Goals

- Implement a pure Go Kubernetes controller.
- Use a single multidoc YAML configuration file loaded locally.
- Launch secure, isolated agent Pods using the `gvisor` runtime.
- Strictly enforce network isolation for agent Pods, allowing egress only to a designated reverse proxy.
- Provision and manage storage for agent work using PVCs.

## Non-Goals

- Implementation of the reverse proxy itself.
- Implementation of the agent container image itself.
- Applying configuration resources directly to a Kubernetes API server.

## Key Requirements

- Everything must be written in pure Go as much as possible, following excellent Go style.
- The command `factory operator` should be implemented under a new root-level `factory/` directory, similar to the `tool/` directory layout.
- The operator accepts a single flag `--config` pointing to a multidoc YAML file containing Kubernetes Resource Model resources (e.g., `FactoryConfig`, `OperatorConfig`).
- Launched Pods must use the `gvisor` runtime class.
- A NetworkPolicy must be created or ensured to block all agent Pod network access except to a TLS-terminating MITM reverse proxy endpoint.
- Agent Pods must perform an init container self-test: verify that internal DNS and 8.8.8.8 are unreachable, and verify the reverse proxy is reachable.
- The operator provisions a PVC for storage, and the init container must be configurable to clone work into this storage before starting the agent.

## Design

The `factory operator` will act as a standalone binary that watches or processes local configuration to orchestrate agent jobs as Kubernetes Pods.

1. **Command Structure**: 
   A new command hierarchy under `factory/` will be created (e.g., `factory/cmd/factory/factory.go` or `factory/operator`).
2. **Configuration Loader**: 
   Parse the file specified by `--config` as a stream of YAML documents, decoding them into strongly typed Go structs representing resources like `FactoryConfig` and `OperatorConfig`. No other command-line flags will be supported for the operator.
3. **Pod Provisioning**:
   The operator generates Pod manifests with the following characteristics:
   - `runtimeClassName: gvisor`
   - A Volume mounted to the containers, backed by a PVC created by the operator.
   - **Init Container**: 
     - Evaluates network restrictions by attempting to ping `8.8.8.8` and resolve cluster-internal DNS. It must fail if either succeeds.
     - Pings or makes a test request to the reverse proxy to ensure connectivity.
     - Clones the required repository or work items into the mounted PVC.
   - **Main Container**:
     - Uses the `agent-image`.
     - Mounts the PVC.
4. **Network Security**:
   The operator will also generate or assume the existence of a strict `NetworkPolicy` for the namespace/labels of the agent Pods, blocking all egress traffic except for the IP/port of the reverse proxy.

## Tests

- **Unit Tests**: Full unit test coverage for the configuration parser, ensuring valid and invalid multidoc YAML files are handled correctly.
- **Manifest Generation Tests**: Verify that the generated Pod manifests include the `gvisor` runtime class, the proper init containers, and the correct volume mounts.
- **Style and Standard**: Tests must be as rigorous and well-structured as those in the Go standard library.
