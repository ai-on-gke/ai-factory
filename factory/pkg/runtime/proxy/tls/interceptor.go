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
