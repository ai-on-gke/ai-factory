---
name: proxy-tls-certgen
---
Create a component that dynamically generates leaf certificates signed by the root CA. It must implement a thread-safe in-memory cache keyed by the SNI hostname to optimize performance and avoid unnecessary cryptographic operations on every connection.
