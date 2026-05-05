---
name: agent-image
deps:
---

# Agent Image

## Overview

The container image containing `gemini-cli` and the necessary environment to execute work as an agent inside a secure Kubernetes Pod.

## Goals

- Provide a consistent, minimal execution environment.
- Package `gemini-cli` alongside necessary code manipulation and testing tools.
- Ensure compatibility with strict network and runtime constraints.

## Non-Goals

- Packaging unnecessary tools or generic utilities not required for agent tasks.
- Managing its own credentials.

## Key Requirements

- Assume execution on GKE within a `gvisor` runtime.
- Image must be configured to route all external traffic through the designated TLS-terminating reverse proxy.
- Must trust the custom CA certificates provided by the reverse proxy.
- Must lack direct access to credentials; all privileged operations must rely on the reverse proxy or MCP tools.

## Design

1. **Base Image**:
   - Use a secure, minimal base image (e.g., Alpine or a minimal Debian-based image) to reduce attack surface.
2. **Tooling Integration**:
   - Install the `gemini-cli` binary and its prerequisites.
   - Include standard utilities needed for codebase interaction (e.g., `git`, language runtimes like Go, Python, or Node.js depending on the factory's target projects).
3. **Network Configuration**:
   - Set environment variables like `HTTPS_PROXY` and `HTTP_PROXY` by default, or provide an entrypoint script that configures them based on Pod environment variables.
   - Include logic in the entrypoint or base configuration to append the reverse proxy's custom CA certificate to the system's trusted certificate store, ensuring seamless TLS connections to intercepted endpoints.

## Tests

- **Image Build Tests**: Validate the Dockerfile builds successfully and passes vulnerability scans.
- **Execution Tests**: Verify that `gemini-cli` and standard tools (like `git`) are executable and in the `PATH`.
- **Certificate Loading**: Tests confirming that if a CA certificate is injected (e.g., via volume mount in the Pod), it is correctly loaded into the trust store by the image's initialization sequence.
