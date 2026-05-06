package proxy

import (
	"net/http"
	"path"
	"strings"
)

// MatchRequest evaluates an incoming HTTP request against a list of proxy rules.
// It returns the matching rule if one exists, or nil otherwise.
func MatchRequest(req *http.Request, rules []ProxyRule) *ProxyRule {
	// Prevent path traversal attacks
	if strings.Contains(req.URL.Path, "..") {
		return nil
	}
	// Prevent /./ bypasses
	if strings.Contains(req.URL.Path, "/./") || req.URL.Path == "/." || strings.HasSuffix(req.URL.Path, "/.") {
		return nil
	}

	// Normalize URL scheme and host for case-insensitive matching.
	// We copy the URL so we don't modify the original request.
	normalizedURL := *req.URL
	normalizedURL.Scheme = strings.ToLower(normalizedURL.Scheme)
	normalizedURL.Host = strings.ToLower(normalizedURL.Host)

	// If it's a proxy request without an absolute URI in the request line,
	// req.URL.Host and Scheme might be empty, so we populate them from req.Host.
	if normalizedURL.Host == "" {
		normalizedURL.Host = strings.ToLower(req.Host)
	}
	if normalizedURL.Scheme == "" {
		if req.TLS != nil {
			normalizedURL.Scheme = "https"
		} else {
			normalizedURL.Scheme = "http"
		}
	}
	
	reqURLString := normalizedURL.String()
	verb := strings.ToUpper(req.Method)

	for i := range rules {
		rule := &rules[i]

		// Check verbs
		verbMatch := false
		for _, allowedVerb := range rule.AllowedVerbs {
			if strings.ToUpper(allowedVerb) == verb {
				verbMatch = true
				break
			}
		}
		if !verbMatch {
			continue
		}

		// Check URLs
		urlMatch := false
		for _, allowedURL := range rule.AllowedURLs {
			matched, err := path.Match(allowedURL, reqURLString)
			if err == nil && matched {
				urlMatch = true
				break
			}
		}

		if urlMatch {
			return rule
		}
	}

	return nil
}
