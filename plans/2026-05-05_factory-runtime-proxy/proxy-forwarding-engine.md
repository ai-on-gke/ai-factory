---
name: proxy-forwarding-engine
---
Implement the main request forwarding engine utilizing standard library `httputil.ReverseProxy`. The `Director` function must integrate policy matching and header injection. If no rule matches, terminate the request with a `403 Forbidden` response. Structure the proxy handler to leave a hook for future `ModifyResponse` logic. Provide table-driven end-to-end integration/unit tests with a mock upstream server.
