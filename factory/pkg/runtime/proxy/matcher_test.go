package proxy

import (
	"net/http"
	"net/url"
	"testing"
)

func TestMatchRequest(t *testing.T) {
	rules := []ProxyRule{
		{
			Name: "github-api",
			AllowedURLs: []string{
				"https://api.github.com/repos/kubernetes-sigs/*",
			},
			AllowedVerbs: []string{"GET", "POST"},
		},
		{
			Name: "google-api",
			AllowedURLs: []string{
				"https://*.googleapis.com/*",
			},
			AllowedVerbs: []string{"GET"},
		},
	}

	tests := []struct {
		name         string
		method       string
		requestURL   string
		expectMatch  bool
		expectedRule string
	}{
		{
			name:         "exact match GET",
			method:       "GET",
			requestURL:   "https://api.github.com/repos/kubernetes-sigs/agent-sandbox",
			expectMatch:  true,
			expectedRule: "github-api",
		},
		{
			name:         "exact match POST",
			method:       "POST",
			requestURL:   "https://api.github.com/repos/kubernetes-sigs/test",
			expectMatch:  true,
			expectedRule: "github-api",
		},
		{
			name:         "verb not allowed",
			method:       "DELETE",
			requestURL:   "https://api.github.com/repos/kubernetes-sigs/test",
			expectMatch:  false,
		},
		{
			name:         "url not allowed",
			method:       "GET",
			requestURL:   "https://api.github.com/repos/other-org/test",
			expectMatch:  false,
		},
		{
			name:         "wildcard host match",
			method:       "GET",
			requestURL:   "https://storage.googleapis.com/bucket",
			expectMatch:  true,
			expectedRule: "google-api",
		},
		{
			name:         "case-insensitive host bypass attempt",
			method:       "GET",
			requestURL:   "https://API.GITHUB.COM/repos/kubernetes-sigs/test",
			expectMatch:  true,
			expectedRule: "github-api",
		},
		{
			name:         "path traversal blocked ..",
			method:       "GET",
			requestURL:   "https://api.github.com/repos/kubernetes-sigs/../other-org/test",
			expectMatch:  false,
		},
		{
			name:         "path traversal blocked .",
			method:       "GET",
			requestURL:   "https://api.github.com/repos/kubernetes-sigs/./test",
			expectMatch:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, _ := url.Parse(tc.requestURL)
			req := &http.Request{
				Method: tc.method,
				URL:    u,
				Host:   u.Host,
			}

			matchedRule := MatchRequest(req, rules)

			if tc.expectMatch {
				if matchedRule == nil {
					t.Fatalf("expected match for rule %q, got nil", tc.expectedRule)
				}
				if matchedRule.Name != tc.expectedRule {
					t.Errorf("expected matched rule %q, got %q", tc.expectedRule, matchedRule.Name)
				}
			} else {
				if matchedRule != nil {
					t.Fatalf("expected no match, got rule %q", matchedRule.Name)
				}
			}
		})
	}
}
