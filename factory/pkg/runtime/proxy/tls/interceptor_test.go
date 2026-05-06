package tls

import (
	"crypto/tls"
	"testing"
)

func TestInterceptor_TLSConfig(t *testing.T) {
	// Create a dummy CA manager and cert generator for testing
	caManager, _ := NewCAManager("", "", "/tmp/dummy-ca.crt")
	cg := NewCertGenerator(caManager)
	interceptor := NewInterceptor(cg)

	config := interceptor.TLSConfig()
	if config == nil {
		t.Fatal("expected TLS config, got nil")
	}

	if config.GetCertificate == nil {
		t.Error("expected GetCertificate callback to be set")
	}

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS1.2, got %v", config.MinVersion)
	}

	// Test the callback wrapper (basic test to ensure it delegates correctly)
	hello := &tls.ClientHelloInfo{ServerName: "test.com"}
	cert, err := config.GetCertificate(hello)
	if err != nil {
		t.Errorf("GetCertificate callback failed: %v", err)
	}
	if cert == nil {
		t.Error("expected certificate from callback, got nil")
	}
}
