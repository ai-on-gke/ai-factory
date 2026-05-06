package tls

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestCAManager_Generate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ca_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	exportPath := filepath.Join(tempDir, "ca.crt")

	manager, err := NewCAManager("", "", exportPath)
	if err != nil {
		t.Fatalf("failed to generate CA: %v", err)
	}

	if manager.Cert == nil {
		t.Error("expected cert to be generated")
	}
	if manager.Key == nil {
		t.Error("expected key to be generated")
	}
	if !manager.Cert.IsCA {
		t.Error("expected cert to be a CA")
	}

	// Verify export
	exportedBytes, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read exported cert: %v", err)
	}

	block, _ := pem.Decode(exportedBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("failed to decode PEM block containing certificate")
	}

	exportedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse exported cert: %v", err)
	}

	if !exportedCert.Equal(manager.Cert) {
		t.Error("exported cert does not match generated cert")
	}
}

func TestCAManager_Load(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ca_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// First generate a CA to load
	origExportPath := filepath.Join(tempDir, "orig_ca.crt")
	origManager, err := NewCAManager("", "", origExportPath)
	if err != nil {
		t.Fatalf("failed to generate orig CA: %v", err)
	}

	// Write key to file for loading
	keyPath := filepath.Join(tempDir, "orig_ca.key")
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(origManager.Key),
	})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	// Now load it
	newExportPath := filepath.Join(tempDir, "new_ca.crt")
	manager, err := NewCAManager(origExportPath, keyPath, newExportPath)
	if err != nil {
		t.Fatalf("failed to load CA: %v", err)
	}

	if !manager.Cert.Equal(origManager.Cert) {
		t.Error("loaded cert does not match original cert")
	}

	// Verify it exported again
	if _, err := os.Stat(newExportPath); os.IsNotExist(err) {
		t.Error("expected new trust bundle to be exported")
	}
}
