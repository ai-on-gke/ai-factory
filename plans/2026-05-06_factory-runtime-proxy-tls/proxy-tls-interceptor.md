---
name: proxy-tls-interceptor
---
Implement the transparent TLS interception logic using Go's `tls.Server` and a custom `GetCertificate` callback. This callback will extract the SNI from the `ClientHelloInfo` and use the certgen component to provide a valid certificate. Integrate this into the main proxy egress flow so that once TLS is terminated, the request flows through the internal middleware chain and then establishes an outbound TLS connection to the destination.
