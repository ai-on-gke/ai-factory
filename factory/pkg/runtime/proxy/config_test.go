package proxy

import (
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listenAddress: 127.0.0.1:8080
  rules:
    - name: allow-github-api
      allowedURLs:
        - https://api.github.com/repos/kubernetes-sigs/agent-sandbox/*
      allowedVerbs:
        - GET
        - POST
      injection:
        header: Authorization
        placeholder: Bearer GITHUB_TOKEN_PLACEHOLDER
        secretFile: /var/run/secrets/github/token
`,
			expectError: false,
		},
		{
			name: "missing listenAddress",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
`,
			expectError: true,
			errorMsg:    "listenAddress is required",
		},
		{
			name: "missing allowedURLs",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listenAddress: 127.0.0.1:8080
  rules:
    - name: rule1
      allowedVerbs: ["GET"]
`,
			expectError: true,
			errorMsg:    "allowedURLs is required for rule rule1",
		},
		{
			name: "missing allowedVerbs",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listenAddress: 127.0.0.1:8080
  rules:
    - name: rule1
      allowedURLs: ["*"]
`,
			expectError: true,
			errorMsg:    "allowedVerbs is required for rule rule1",
		},
		{
			name: "invalid kind",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: Pod
spec:
  listenAddress: 127.0.0.1:8080
`,
			expectError: true,
			errorMsg:    "invalid kind: Pod",
		},
		{
			name: "invalid injection header",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listenAddress: 127.0.0.1:8080
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
      injection:
        placeholder: test
        secretFile: /test
`,
			expectError: true,
			errorMsg:    "injection header is required for rule rule1",
		},
		{
			name: "invalid injection placeholder",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listenAddress: 127.0.0.1:8080
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
      injection:
        header: Auth
        secretFile: /test
`,
			expectError: true,
			errorMsg:    "injection placeholder is required for rule rule1",
		},
		{
			name: "invalid injection secretFile",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listenAddress: 127.0.0.1:8080
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
      injection:
        header: Auth
        placeholder: test
`,
			expectError: true,
			errorMsg:    "injection secretFile is required for rule rule1",
		},
		{
			name: "malformed yaml",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listenAddress: 127.0.0.1:8080
  rules:
    - name: rule1
      allowedURLs: "*"
`,
			expectError: true,
			errorMsg:    "failed to unmarshal config",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(tc.input))
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
