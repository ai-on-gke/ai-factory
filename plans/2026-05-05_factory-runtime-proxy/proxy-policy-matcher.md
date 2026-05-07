---
name: proxy-policy-matcher
---
Implement the policy matching logic. For an incoming HTTP request, iterate over the configured rules, validating the HTTP verb against `allowedVerbs` and the full request URL against `allowedURLs` patterns using shell-style wildcard matching via `path.Match`. Explicitly prevent common security mistakes by blocking path traversals (e.g., using '..' or '.') and handling case-sensitive policy bypasses within the matching logic. Include table-driven unit tests covering various allowed and denied scenarios.
