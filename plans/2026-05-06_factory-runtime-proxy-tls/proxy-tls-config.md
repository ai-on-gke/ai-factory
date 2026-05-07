---
name: proxy-tls-config
---
Update the proxy configuration parsing logic to support a `tls` block and `httpsPort`. If `httpsPort` is provided, the `tls` block must be required and valid, including paths for trust bundle export, and optional CA cert/key files.
