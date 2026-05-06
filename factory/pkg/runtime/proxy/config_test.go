package proxy

import (
	"fmt"
	"os"
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
  listen:
    address: 127.0.0.1
    httpPort: 8080
  rules:
    - name: allow-github-api
      allowedURLs:
        - https://api.github.com/repos/kubernetes-sigs/agent-sandbox/*
      allowedVerbs:
        - GET
        - POST
      injection:
        header: Authorization
        placeholder: GITHUB_TOKEN_PLACEHOLDER
        secretFile: %s
`,
			expectError: false,
		},
		{
			name: "missing listen address",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    httpPort: 8080
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
`,
			expectError: true,
			errorMsg:    "listen address is required",
		},
		{
			name: "missing ports",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
`,
			expectError: true,
			errorMsg:    "at least one of listen.httpPort or listen.httpsPort must be specified",
		},
		{
			name: "missing allowedURLs",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpPort: 8080
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
  listen:
    address: 127.0.0.1
    httpPort: 8080
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
  listen:
    address: 127.0.0.1
    httpPort: 8080
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
  listen:
    address: 127.0.0.1
    httpPort: 8080
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
  listen:
    address: 127.0.0.1
    httpPort: 8080
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
  listen:
    address: 127.0.0.1
    httpPort: 8080
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
			name: "injection placeholder with spaces",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpPort: 8080
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
      injection:
        header: Auth
        placeholder: Bearer TOKEN
        secretFile: /test
`,
			expectError: true,
			errorMsg:    "should not contain spaces",
		},
		{
			name: "malformed yaml",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpPort: 8080
  rules:
    - name: rule1
      allowedURLs: "*"
`,
			expectError: true,
			errorMsg:    "failed to unmarshal config",
		},
		{
			name: "missing tls when httpsPort specified",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpsPort: 8443
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
`,
			expectError: true,
			errorMsg:    "tls configuration is required when httpsPort is specified",
		},
		{
			name: "missing trustBundleExportPath",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpsPort: 8443
  tls: {}
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
`,
			expectError: true,
			errorMsg:    "tls.trustBundleExportPath is required",
		},
		{
			name: "missing caKeyFile",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpsPort: 8443
  tls:
    trustBundleExportPath: /tmp/ca.crt
    caCertFile: /tmp/ca.crt
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
`,
			expectError: true,
			errorMsg:    "tls.caKeyFile is required when tls.caCertFile is specified",
		},
		{
			name: "missing caCertFile",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpsPort: 8443
  tls:
    trustBundleExportPath: /tmp/ca.crt
    caKeyFile: /tmp/ca.key
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
`,
			expectError: true,
			errorMsg:    "tls.caCertFile is required when tls.caKeyFile is specified",
		},
		{
			name: "valid tls config",
			input: `
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpPort: 8080
    httpsPort: 8443
  tls:
    trustBundleExportPath: /tmp/ca.crt
    caCertFile: /tmp/ca.crt
    caKeyFile: /tmp/ca.key
  rules:
    - name: rule1
      allowedURLs: ["*"]
      allowedVerbs: ["GET"]
`,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			if strings.Contains(input, "%s") {
				f, err := os.CreateTemp("", "secret")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				defer os.Remove(f.Name())
				f.WriteString("secret")
				f.Close()
				input = fmt.Sprintf(input, f.Name())
			}
			_, err := ParseConfig([]byte(input))
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
