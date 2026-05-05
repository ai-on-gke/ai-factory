---
name: proxy-header-injector
---
Implement conditional header injection. If a rule matches and specifies an injection block, inspect the configured header. If it contains the expected placeholder, dynamically read the secret file, trim any trailing whitespace, and replace the placeholder with the secret value. Ensure that the secret is read dynamically on each request, subject to caching per spec, to handle rotations properly. Include table-driven unit tests to verify the header injection logic.
