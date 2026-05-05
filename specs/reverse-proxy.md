---
name: reverse-proxy
deps:
---

# Reverse Proxy

## Overview

A TLS-terminating MITM reverse proxy that acts as the sole external gateway for agent Pods, intercepting, authenticating, and observing their requests.

## Goals

- Securely host credentials on behalf of the agent, keeping them invisible to the agent Pod.
- Provide a secure, isolated channel for necessary external access, specifically an LLM API.
- Support Model Context Protocol (MCP) tools for operations like creating Pull Requests.
- Allow task-specific or sandbox-specific security policies.

## Non-Goals

- General-purpose HTTP proxying for unrestricted internet access.
- Complex routing outside of predefined upstream targets.

## Key Requirements

- Pure Go implementation, following excellent Go style.
- Must terminate TLS to inspect egress traffic.
- Must allow direct proxying to LLM APIs (e.g., `https://generativelanguage.googleapis.com`).
- All other operations must go through MCP tools served by the reverse proxy.
- Security policies must be configurable per task/sandbox.
- Must securely host and inject credentials (e.g., GitHub tokens) without exposing them to the agent.

## Design

1. **Proxy Server Architecture**:
   - A Go-based HTTP/HTTPS server using `net/http` and custom TLS configuration to act as a MITM proxy.
   - Generates and serves custom certificates for the agent Pods to trust, enabling TLS termination.
2. **Traffic Routing and Interception**:
   - Maintains an allowlist of external domains (e.g., `generativelanguage.googleapis.com`). Traffic to these domains is logged and forwarded, potentially injecting API keys stored securely in the proxy.
   - For all other requests, particularly those interacting with systems like GitHub, the proxy blocks direct access and instead serves or routes traffic to MCP tools.
3. **Credential Management**:
   - The proxy handles authentication on behalf of the agent. For example, if an MCP tool initiates a GitHub PR, the proxy uses its securely stored tokens to authenticate the request.
4. **Policy Configuration**:
   - The proxy reads configuration defining the security policy for a given sandbox (identifiable via headers, source IPs, or dedicated endpoints), restricting the tools and endpoints available to that specific agent session.

## Tests

- **Unit Tests**: Coverage for request routing, TLS certificate generation, and header injection.
- **Proxy Simulation**: Tests establishing a local TLS connection through the proxy to ensure credentials are injected correctly and unauthorized domains are blocked.
- **Style and Standard**: High-quality, Go-standard-library level testing for all logic.
