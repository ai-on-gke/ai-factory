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
	"math/big"
	"net"
	"sync"
	"time"
)

// CertGenerator generates and caches leaf certificates signed by a root CA.
type CertGenerator struct {
	caManager *CAManager
	cache     map[string]*tls.Certificate
	mutex     sync.RWMutex
}

// NewCertGenerator creates a new CertGenerator.
func NewCertGenerator(caManager *CAManager) *CertGenerator {
	return &CertGenerator{
		caManager: caManager,
		cache:     make(map[string]*tls.Certificate),
	}
}

// GetCertificate returns a cached certificate for the given SNI or generates a new one.
func (cg *CertGenerator) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	serverName := hello.ServerName
	if serverName == "" {
		// If no SNI is provided, we can either reject or provide a fallback.
		// For transparent proxies, an empty SNI usually means we can't do much,
		// but let's provide a fallback to localhost to avoid panics.
		serverName = "localhost"
	}

	// Fast path: check cache with read lock
	cg.mutex.RLock()
	cert, ok := cg.cache[serverName]
	cg.mutex.RUnlock()
	if ok {
		return cert, nil
	}

	// Slow path: generate certificate with write lock
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	// Double-check cache in case another goroutine generated it while we waited for the lock
	cert, ok = cg.cache[serverName]
	if ok {
		return cert, nil
	}

	newCert, err := cg.generateLeafCert(serverName)
	if err != nil {
		return nil, err
	}

	cg.cache[serverName] = newCert
	return newCert, nil
}

func (cg *CertGenerator) generateLeafCert(serverName string) (*tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour), // Short-lived leaf certs
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(serverName); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{serverName}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, cg.caManager.Cert, &priv.PublicKey, cg.caManager.Key)
	if err != nil {
		return nil, err
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  priv,
	}

	return tlsCert, nil
}
