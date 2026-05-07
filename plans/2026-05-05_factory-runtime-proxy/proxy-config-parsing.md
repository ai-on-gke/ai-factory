---
name: proxy-config-parsing
---
Implement the Go types mapping to the `ProxySpec` custom Kubernetes Resource Model (KRM). Use `sigs.k8s.io/yaml` for parsing. Add strict validation logic for required fields (`listenAddress`, rules, etc.). Include table-driven unit tests verifying correct unmarshaling and error handling for malformed inputs.
