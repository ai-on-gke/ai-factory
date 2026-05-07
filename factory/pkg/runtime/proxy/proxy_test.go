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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProxyFlow(t *testing.T) {
	// Setup a mock upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		w.Header().Set("X-Received-Auth", auth)
		fmt.Fprintln(w, "Hello from upstream")
	}))
	defer upstream.Close()

	// Create a temp secret file
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "token")
	err := os.WriteFile(secretFile, []byte("my-secret-token"), 0644)
	if err != nil {
		t.Fatalf("failed to create secret file: %v", err)
	}

	// Create config
	cfg := &ProxyConfig{
		Spec: ProxySpec{
			Listen: ListenConfig{
				Address:  "127.0.0.1",
				HTTPPort: 8080,
			},
			Rules: []ProxyRule{
				{
					Name: "allow-upstream",
					AllowedURLs: []string{
						upstream.URL + "/*",
					},
					AllowedVerbs: []string{"GET"},
					Injection: &HeaderInjection{
						Header:      "Authorization",
						Placeholder: "PLACEHOLDER",
						SecretFile:  secretFile,
						SecretValue: "my-secret-token",
					},
				},
			},
		},
	}

	proxyServer := NewServer(cfg)
	proxyTestServer := httptest.NewServer(proxyServer)
	defer proxyTestServer.Close()

	tests := []struct {
		name               string
		method             string
		urlPath            string // relative to upstream, or absolute
		initialAuth        string
		expectStatus       int
		expectBody         string
		expectUpstreamAuth string
	}{
		{
			name:               "successful proxy and injection",
			method:             "GET",
			urlPath:            "/test",
			initialAuth:        "Bearer PLACEHOLDER",
			expectStatus:       http.StatusOK,
			expectBody:         "Hello from upstream\n",
			expectUpstreamAuth: "Bearer my-secret-token",
		},
		{
			name:         "forbidden url",
			method:       "GET",
			urlPath:      "http://example.com/test",
			initialAuth:  "Bearer PLACEHOLDER",
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "forbidden verb",
			method:       "POST",
			urlPath:      "/test",
			initialAuth:  "Bearer PLACEHOLDER",
			expectStatus: http.StatusForbidden,
		},
		{
			name:               "no placeholder, no injection",
			method:             "GET",
			urlPath:            "/test2",
			initialAuth:        "Bearer OTHER",
			expectStatus:       http.StatusOK,
			expectBody:         "Hello from upstream\n",
			expectUpstreamAuth: "Bearer OTHER",
		},
	}

	// Create a client that uses the proxyTestServer as proxy
	proxyURL, _ := url.Parse(proxyTestServer.URL)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := tc.urlPath
			if !strings.HasPrefix(targetURL, "http") {
				targetURL = upstream.URL + tc.urlPath
			}

			req, _ := http.NewRequest(tc.method, targetURL, nil)
			if tc.initialAuth != "" {
				req.Header.Set("Authorization", tc.initialAuth)
			}

			res, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != tc.expectStatus {
				t.Errorf("expected status %d, got %d", tc.expectStatus, res.StatusCode)
			}

			if tc.expectStatus == http.StatusOK {
				body, _ := io.ReadAll(res.Body)
				if string(body) != tc.expectBody {
					t.Errorf("expected body %q, got %q", tc.expectBody, string(body))
				}

				upstreamAuth := res.Header.Get("X-Received-Auth")
				if upstreamAuth != tc.expectUpstreamAuth {
					t.Errorf("expected upstream auth %q, got %q", tc.expectUpstreamAuth, upstreamAuth)
				}
			}
		})
	}
}
