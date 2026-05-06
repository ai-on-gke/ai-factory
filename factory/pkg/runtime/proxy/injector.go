package proxy

import (
	"net/http"
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

	newValue := strings.Replace(headerValue, placeholder, rule.Injection.SecretValue, 1)
	req.Header.Set(headerName, newValue)

	return nil
}
