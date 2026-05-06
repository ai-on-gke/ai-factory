package proxy

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// InjectHeader inspects the request and injects the secret token if the configured placeholder is present.
func InjectHeader(req *http.Request, rule *ProxyRule) error {
	if rule == nil || rule.Injection == nil {
		return nil
	}

	headerName := rule.Injection.Header
	placeholder := rule.Injection.Placeholder
	
	headerValue := req.Header.Get(headerName)
	if !strings.Contains(headerValue, placeholder) {
		// Placeholder not present, no injection needed
		return nil
	}

	secretData, err := os.ReadFile(rule.Injection.SecretFile)
	if err != nil {
		return fmt.Errorf("failed to read secret file %q: %w", rule.Injection.SecretFile, err)
	}

	// Trim trailing whitespace
	secretValue := strings.TrimRight(string(secretData), "\r\n\t ")

	newValue := strings.Replace(headerValue, placeholder, secretValue, 1)
	req.Header.Set(headerName, newValue)

	return nil
}
