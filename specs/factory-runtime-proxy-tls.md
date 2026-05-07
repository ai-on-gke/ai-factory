---
name: factory-runtime-proxy-tls
deps:
  - factory-runtime-proxy
---

# Factory Runtime Proxy TLS Termination

## Overview

This specification outlines the addition of TLS termination to the `factory-runtime-proxy`. The primary motivation is to allow the proxy to inspect and mutate outbound HTTPS requests (e.g., injecting authentication headers) originating from the runtime environment before they are forwarded to external destinations.

## Goals

* Terminate TLS connections in the egress proxy to enable inspection and modification of HTTPS requests.
* Support transparent TLS interception, where clients initiate TLS directly without sending explicit `CONNECT` requests.
* Dynamically generate leaf certificates on the fly using a Certificate Authority (CA).
* Automatically write a client trust bundle (public CA certificate) to a configurable local file path so clients can trust the proxy.

## Non-Goals

* Supporting explicit proxying (e.g., where clients are explicitly configured with an `HTTP_PROXY` or `HTTPS_PROXY` and send `CONNECT` requests).
* Handling non-TLS traffic on the TLS-designated listener.
* Implementing the actual header injection logic itself; this spec focuses purely on the TLS interception capability required to make header injection possible.

## Key Requirements

* The proxy must use a TLS listener designed for transparent interception.
* The proxy must inspect the Server Name Indication (SNI) from the `ClientHello` message to identify the destination domain.
* The proxy must generate and sign leaf certificates on the fly for the extracted SNI domain.
* The proxy must accept a CA or generate one on startup, and it MUST export the CA's public certificate to a configurable local file path.
* If `httpsPort` is configured in the `listen` block, the `tls` configuration block MUST be required and valid.

## Design

### 1. Transparent Interception via SNI
In a transparent interception scenario, the client behaves as if it is connecting directly to the destination server, initiating a standard TLS handshake. 
* **Implementation Guidance**: Manual byte-peeking is not required. The proxy should leverage the Go standard library's `crypto/tls` package by configuring a `tls.Server` with a custom `tls.Config{GetCertificate: ...}` callback.
* When a client connects, Go natively parses the `ClientHello` message and passes a `tls.ClientHelloInfo` struct to this callback.
* The proxy will extract the destination domain directly from the `ClientHelloInfo.ServerName` field (the SNI) to use for both certificate generation and upstream routing.

### 2. Dynamic Certificate Generation
To successfully complete the TLS handshake with the client, the proxy must present a valid certificate for the requested SNI hostname.
* The proxy will maintain a root CA, which can either be provided via configuration or generated dynamically at startup.
* **Implementation Guidance**: Inside the `GetCertificate` callback, the proxy should use the standard library's `crypto/x509.CreateCertificate` to generate a new leaf certificate for the SNI hostname, signed by the root CA's private key.
* To optimize latency and avoid expensive cryptographic operations on every connection, the proxy MUST implement an in-memory thread-safe cache for these dynamically generated leaf certificates, keyed by the SNI hostname.

### 3. Client Trust Bundle
For the client to trust the dynamically generated leaf certificates, the client environment must trust the root CA used by the proxy.
* During initialization, the proxy MUST write its root CA public certificate (the trust bundle) to a configurable local file path.
* This file can then be mounted into client sandboxes or referenced by environment variables (like `SSL_CERT_FILE` or `SSL_CERT_DIR`), ensuring that client applications can make outbound HTTPS requests without encountering certificate validation errors.

### 4. Integration with Egress Flow
* Once the client-facing TLS connection is terminated, the proxy has access to the plaintext HTTP request.
* The proxy will route this request through its internal middleware chain (where header injection and other mutations can occur).
* Finally, the proxy will initiate a new, outbound TLS connection to the actual upstream destination indicated by the SNI, forwarding the modified request.

## Examples

### Configuration Model (KRM)
To support both plaintext HTTP and TLS-terminated HTTPS transparently, the configuration schema should be updated to use a structured `listen` block and extended with a `tls` block. 
*Note: If `httpsPort` is specified, the `tls` configuration block MUST be provided and valid.*

```yaml
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpPort: 8080
    httpsPort: 8443
  tls:
    # REQUIRED: Where the proxy will write the public CA certificate so clients can trust it.
    trustBundleExportPath: /var/run/proxy/ca.crt
    # OPTIONAL: If provided, use this CA instead of generating a new one dynamically on startup.
    caCertFile: /etc/proxy/certs/ca.crt
    caKeyFile: /etc/proxy/certs/ca.key
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

## Tests

* **Certificate Generation Tests**: Unit tests verifying that leaf certificates are correctly generated, signed by the CA, and include the correct Subject Alternative Names (SANs) based on the SNI.
* **Trust Bundle Export Tests**: Tests verifying that the CA public certificate is correctly written to the configured output path upon proxy initialization.
* **Transparent Interception Tests**: Integration tests simulating a client connecting directly to the proxy using TLS, verifying that the proxy accurately extracts the SNI, serves a valid dynamically generated certificate, and successfully completes the TLS handshake.
* **End-to-End Proxy Tests**: An E2E test verifying that an HTTPS request through the transparent proxy successfully terminates TLS, exposes plaintext for modification, and successfully completes the round trip to an upstream server.
