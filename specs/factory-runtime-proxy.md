---
name: factory-runtime-proxy
deps: []
---

# Factory Runtime Proxy

## Overview

The `factory runtime proxy` command implements a lightweight, secure egress reverse proxy designed for the `ai-factory` environment. Its primary responsibility is to intercept outgoing HTTP/HTTPS requests from components or agent sandboxes, validate them against a strict egress policy, and inject sensitive authentication tokens into specified headers. To ensure maximum security, maintainability, and portability, the proxy is implemented in pure Go using only the Go standard library.

## Goals

- Implement a minimal-dependency egress proxy using the Go standard library (`net/http`, etc.) and a lightweight YAML parser.
- Configure the proxy using a single command-line flag: `--config <path>`.
- Define the egress policy via an on-disk Kubernetes Resource Model (KRM) YAML file.
- Support fine-grained egress matching based on explicit URLs and paths using shell-style wildcards (`path.Match`) and HTTP verbs.
- Perform conditional header injection by replacing a pre-configured placeholder with a secret value read from an on-disk file.
- Design the architecture to leave open the opportunity for future ingress policy or ingress filtering, including inspecting and filtering response packets.
- Ensure the initial implementation strictly blocks direct ingress traffic, while naturally allowing response packets for established egress connections to pass through.

## Non-Goals

- Implementing generic ingress routing, complex load balancing, or path re-writing (though basic ingress filtering architecture should be left open for the future).
- Providing dynamic configuration reloading without restarting the proxy process.
- Implementing caching, rate-limiting, or proxy chaining.
- Supporting non-HTTP/HTTPS layer 4 protocols.

## Key Requirements

- **Minimal Dependencies**: Use the Go standard library for proxying (`net/http` and `httputil.ReverseProxy`). The `sigs.k8s.io/yaml` package (the same parser used by upstream Kubernetes) is required for parsing the YAML configuration.
- **Strict Flag Validation**: The proxy must accept exactly one flag (`--config`). Any other flags or positional arguments must result in an immediate error and usage printout.
- **Token Refreshment**: Secret files must be read dynamically on each request (or not heavily cached, some caching is ok though, for say 10 minutes, configurable) to automatically respect token rotations on disk (e.g., Kubernetes projected volumes).
- **Table-driven tests**: Use table-driven test style for both unit and integration tests. Generally test to the same level of quality as something in the Go standard library.
- **Common security mistakes**: Generally ensure that common security mistakes don't allow bypassing the proxy (e.g. failure to block path traversals, case sensitive policy that fails to identify case-insensitive variants, etc).

## Design

### Command Line Interface
The proxy command will be invoked as follows:
```bash
factory runtime proxy --config /path/to/config.yaml
```
The Go `flag` package will be used to enforce that `--config` is provided and that no unexpected arguments are present.

### Configuration Model (KRM)
To maintain KRM compatibility, the configuration file will be written in YAML format. The structure maps to a custom Kubernetes Resource Model:

```yaml
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listenAddress: 127.0.0.1:8080
  rules:
    - name: allow-github-api
      allowedURLs:
        - https://api.github.com/repos/kubernetes-sigs/agent-sandbox/*
      allowedVerbs:
        - GET
        - POST
      injection:
        header: Authorization
        placeholder: GITHUB_TOKEN_PLACEHOLDER
        secretFile: /var/run/secrets/github/token
```

### Request Handling and Forwarding Engine
1. **Initialization**: The proxy reads and parses the YAML configuration file into Go structures using the permitted YAML parser. It then starts an HTTP server listening on `spec.listenAddress`.
2. **Request Interception**: The server handler intercepts incoming HTTP requests. The proxy functions as a forward/egress proxy, expecting either absolute URLs in the request line or standard proxy request formatting.
3. **Policy Matching**:
   - For each request, the proxy iterates through `spec.rules`.
   - It validates if `req.Method` matches one of the elements in `allowedVerbs`.
   - It validates if `req.URL.String()` matches one of the patterns in `allowedURLs` using shell-style wildcard matching via the standard library's `path.Match` capability.
   - If no rule matches, the proxy terminates the request immediately, returning `403 Forbidden`.
4. **Header Injection**:
   - If a rule matches and contains an `injection` block, the proxy inspects the request headers for the configured `header` name.
   - If the header exists and matches or contains the configured `placeholder`, the proxy reads the secret token from `secretFile`, trims any trailing whitespace, and replaces the placeholder with the token value.
   - Note: The placeholder must not contain spaces or prefixes like "Bearer " (e.g. use `GITHUB_TOKEN_PLACEHOLDER` instead of `Bearer GITHUB_TOKEN_PLACEHOLDER`), otherwise the substring replacement will inadvertently strip the prefix.
   - If the placeholder is not present, the proxy does not inject the token (as per the requirement to only inject when the pre-configured placeholder value is set).
5. **Upstream Proxying**:
   - The proxy leverages `httputil.ReverseProxy` from the standard library.
   - The `Director` function modifies the request to point to the target upstream URL, updates the headers, and cleans up hop-by-hop headers as standard.
6. **Ingress and Response Handling**:
   - The initial implementation will not expose any listener for direct ingress traffic (other than the proxy port intended for internal clients), effectively blocking direct external ingress.
   - Responses to valid egress requests are automatically handled and allowed through by `httputil.ReverseProxy`.
   - The design must allow future insertion of a `ModifyResponse` function in the `httputil.ReverseProxy` to enable filtering or policy enforcement on response packets.

## Examples

### Scenario 1: Successful Injection
- **Configuration**: As shown in the Design section.
- **Secret File Content (`/var/run/secrets/github/token`)**: `ghp_secret123456`
- **Incoming Request**:
  ```http
  GET https://api.github.com/repos/kubernetes-sigs/agent-sandbox/issues HTTP/1.1
  Host: api.github.com
  Authorization: Bearer GITHUB_TOKEN_PLACEHOLDER
  ```
- **Processed Request Forwarded Upstream**:
  ```http
  GET /repos/kubernetes-sigs/agent-sandbox/issues HTTP/1.1
  Host: api.github.com
  Authorization: Bearer ghp_secret123456
  ```

### Scenario 2: Request Blocked by Policy
- **Incoming Request**:
  ```http
  DELETE https://api.github.com/repos/kubernetes-sigs/agent-sandbox/issues/1 HTTP/1.1
  Host: api.github.com
  Authorization: Bearer GITHUB_TOKEN_PLACEHOLDER
  ```
- **Proxy Response**: `403 Forbidden` (since `DELETE` is not in `allowedVerbs`).

## Tests

### Unit Tests
- **`TestParseConfig`**: Verifies correct unmarshaling of the KRM YAML configuration and validation of required fields (`listenAddress`, `allowedURLs`, `allowedVerbs`).
- **`TestPolicyMatcher`**: Evaluates various URLs and HTTP verbs against policy rules to ensure correct allow/deny outcomes, including shell-style wildcard path behavior (`*`) via `path.Match`.
- **`TestHeaderInjection`**: Mocks the file system or reads temporary secret files to verify that placeholders within specific headers are accurately replaced by the secret value, while ensuring requests without placeholders remain untouched.

### Integration Tests
- **`TestProxyFlow`**: Launches a mock upstream HTTP server and an instance of the proxy server. Verifies that a complete end-to-end round-trip successfully validates the egress policy, injects the token into the request headers, and forwards the request to the mock upstream server correctly.
Uses distinct test cases to ensure positive and negative (error case) coverage.
