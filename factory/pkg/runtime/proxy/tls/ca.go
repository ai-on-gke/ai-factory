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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CAManager handles loading or generating the root CA and exporting the trust bundle.
type CAManager struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}

// NewCAManager creates a new CAManager, loading an existing CA if certFile and keyFile
// are provided, or generating a new dynamic CA otherwise. It then exports the trust
// bundle to the exportPath.
func NewCAManager(certFile, keyFile, exportPath string) (*CAManager, error) {
	manager := &CAManager{}

	if certFile != "" && keyFile != "" {
		if err := manager.loadCA(certFile, keyFile); err != nil {
			return nil, fmt.Errorf("failed to load existing CA: %w", err)
		}
	} else {
		if err := manager.generateCA(); err != nil {
			return nil, fmt.Errorf("failed to generate dynamic CA: %w", err)
		}
	}

	if err := manager.exportTrustBundle(exportPath); err != nil {
		return nil, fmt.Errorf("failed to export trust bundle: %w", err)
	}

	return manager, nil
}

func (m *CAManager) loadCA(certFile, keyFile string) error {
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	
	cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return err
	}

	privateKey, ok := tlsCert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("expected RSA private key")
	}

	m.Cert = cert
	m.Key = privateKey
	return nil
}

func (m *CAManager) generateCA() error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}

	m.Cert = cert
	m.Key = priv
	return nil
}

func (m *CAManager) exportTrustBundle(exportPath string) error {
	if err := os.MkdirAll(filepath.Dir(exportPath), 0755); err != nil {
		return err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: m.Cert.Raw,
	})

	return os.WriteFile(exportPath, certPEM, 0644)
}
