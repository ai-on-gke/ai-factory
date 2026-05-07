---
name: proxy-tls-ca
---
Implement a CA manager for the proxy. This component should handle loading an existing CA from configured files or generating a new root CA dynamically. It must also handle exporting the CA's public certificate to the configured `trustBundleExportPath` so clients can trust the proxy.
