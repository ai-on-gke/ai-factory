// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"net/http"
	"path"
	"strings"
)

// MatchRequest evaluates an incoming HTTP request against a list of proxy rules.
// It returns the matching rule if one exists, or nil otherwise.
func MatchRequest(req *http.Request, rules []ProxyRule) *ProxyRule {
	// Prevent path traversal and bypass attacks.
	// We require the client to send a strictly canonical path. If path.Clean
	// alters the path (other than stripping a trailing slash), it contains
	// non-canonical elements like "//", "/./", or "/../", and we reject it.
	cleanedPath := path.Clean(req.URL.Path)
	if cleanedPath != "/" && strings.HasSuffix(req.URL.Path, "/") {
		cleanedPath += "/"
	}
	if cleanedPath != req.URL.Path {
		return nil
	}

	// Normalize URL scheme and host for case-insensitive matching.
	// We copy the URL so we don't modify the original request.
	// Per RFC 3986, scheme and host are case-insensitive. Normalizing them
	// prevents bypasses or brittle rules. The path remains case-sensitive.
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
