---
name: proxy-main-cli
---
Implement the main entrypoint and CLI command flag validation using the standard library `flag` package. Enforce that exactly one flag (`--config`) is accepted, and reject any other flags or positional arguments with a clear usage error. Start the HTTP server on the configured `listenAddress`.
