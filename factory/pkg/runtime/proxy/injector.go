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
