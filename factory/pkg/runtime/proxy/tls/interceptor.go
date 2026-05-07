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

package tls

import (
	"crypto/tls"
)

// Interceptor manages the TLS configuration for transparent interception.
type Interceptor struct {
	CertGenerator *CertGenerator
}

// NewInterceptor creates a new TLS Interceptor.
func NewInterceptor(cg *CertGenerator) *Interceptor {
	return &Interceptor{
		CertGenerator: cg,
	}
}

// TLSConfig returns a tls.Config configured for transparent interception,
// using the GetCertificate callback to dynamically generate certificates based on SNI.
func (i *Interceptor) TLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: i.CertGenerator.GetCertificate,
		// Require SNI to be present to know what certificate to generate
		// Note: we handle empty SNI fallback in GetCertificate if necessary.
		MinVersion: tls.VersionTLS12,
	}
}
