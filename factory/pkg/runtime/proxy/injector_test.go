package proxy

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestInjectHeader(t *testing.T) {
	// Create a temporary directory for secret files
	tmpDir := t.TempDir()
	secretFilePath := filepath.Join(tmpDir, "token")
	err := os.WriteFile(secretFilePath, []byte("my-secret-token\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	rule := &ProxyRule{
		Name: "test-rule",
		Injection: &HeaderInjection{
			Header:      "Authorization",
			Placeholder: "PLACEHOLDER",
			SecretFile:  secretFilePath,
			SecretValue: "my-secret-token",
		},
	}

	tests := []struct {
		name          string
		rule          *ProxyRule
		initialHeader string
		expectHeader  string
		expectError   bool
	}{
		{
			name:          "successful injection",
			rule:          rule,
			initialHeader: "Bearer PLACEHOLDER",
			expectHeader:  "Bearer my-secret-token",
		},
		{
			name:          "placeholder not present",
			rule:          rule,
			initialHeader: "Bearer OTHER_TOKEN",
			expectHeader:  "Bearer OTHER_TOKEN",
		},
		{
			name:          "header not present",
			rule:          rule,
			initialHeader: "",
			expectHeader:  "",
		},
		{
			name: "no injection configured",
			rule: &ProxyRule{Name: "no-injection"},
			initialHeader: "Bearer PLACEHOLDER",
			expectHeader:  "Bearer PLACEHOLDER",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			if tc.initialHeader != "" {
				req.Header.Set("Authorization", tc.initialHeader)
			}

			err := InjectHeader(req, tc.rule)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				actualHeader := req.Header.Get("Authorization")
				if actualHeader != tc.expectHeader {
					t.Errorf("expected header %q, got %q", tc.expectHeader, actualHeader)
				}
			}
		})
	}
}
