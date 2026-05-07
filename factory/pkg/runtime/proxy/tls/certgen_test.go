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
	"os"
	"path/filepath"
	"testing"
)

func TestCertGenerator_GetCertificate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "certgen_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	exportPath := filepath.Join(tempDir, "ca.crt")
	caManager, err := NewCAManager("", "", exportPath)
	if err != nil {
		t.Fatalf("failed to create ca manager: %v", err)
	}

	certGen := NewCertGenerator(caManager)

	// Generate a cert
	hello := &tls.ClientHelloInfo{ServerName: "example.com"}
	cert1, err := certGen.GetCertificate(hello)
	if err != nil {
		t.Fatalf("failed to get certificate: %v", err)
	}
	if cert1 == nil {
		t.Fatal("expected a certificate, got nil")
	}

	// Request the same SNI, should hit cache
	cert2, err := certGen.GetCertificate(hello)
	if err != nil {
		t.Fatalf("failed to get certificate second time: %v", err)
	}
	if cert1 != cert2 {
		t.Error("expected cached certificate, got different instance")
	}

	// Request different SNI, should generate new
	hello2 := &tls.ClientHelloInfo{ServerName: "test.com"}
	cert3, err := certGen.GetCertificate(hello2)
	if err != nil {
		t.Fatalf("failed to get certificate for test.com: %v", err)
	}
	if cert3 == cert1 {
		t.Error("expected new certificate, got same instance")
	}

	// Empty SNI fallback
	helloEmpty := &tls.ClientHelloInfo{}
	certEmpty, err := certGen.GetCertificate(helloEmpty)
	if err != nil {
		t.Fatalf("failed to get certificate for empty SNI: %v", err)
	}
	if certEmpty == nil {
		t.Fatal("expected a certificate for empty SNI")
	}
}
