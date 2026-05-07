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
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ai-on-gke/ai-factory/factory/pkg/runtime/proxy"
)

func TestE2EProxyFlow(t *testing.T) {
	// 1. Start a dummy upstream server
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Received-Auth", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from upstream"))
	}))
	defer upstream.Close()

	// 2. Prepare Proxy Config
	tempDir, err := os.MkdirTemp("", "e2e_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	secretPath := filepath.Join(tempDir, "secret")
	if err := os.WriteFile(secretPath, []byte("super-secret-token"), 0600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	trustBundleExportPath := filepath.Join(tempDir, "proxy-ca.crt")

	// Find an open port to avoid hardcoded port conflicts
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find open port: %v", err)
	}
	proxyPort := l.Addr().(*net.TCPAddr).Port
	l.Close()

	// Create proxy config
	configYAML := fmt.Sprintf(`
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpsPort: %d
  tls:
    trustBundleExportPath: %s
  rules:
    - name: rule1
      allowedURLs: ["https://example-upstream.test/*"]
      allowedVerbs: ["GET"]
      injection:
        header: Authorization
        placeholder: TOKEN
        secretFile: %s
`, proxyPort, trustBundleExportPath, secretPath)

	config, err := proxy.ParseConfig([]byte(configYAML))
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	
	proxyServer := proxy.NewServer(config)
	
	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	// Customize the transport to trust the upstream test server (proxy -> upstream)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // Proxy trusts upstream
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == "example-upstream.test:443" {
			addr = upstreamURL.Host
		}
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	}
	proxyServer.Proxy.Transport = transport

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := proxyServer.Start(ctx); err != nil {
			// Ignore normal shutdown errors
		}
	}()

	// Wait for server to start listening
	ctxWait, cancelWait := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelWait()
	for {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort), 50*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if ctxWait.Err() != nil {
			t.Fatalf("timeout waiting for proxy server to start listening")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 3. Prepare client trusting the proxy CA
	caCertPEM, err := os.ReadFile(trustBundleExportPath)
	if err != nil {
		t.Fatalf("failed to read proxy trust bundle: %v", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertPEM); !ok {
		t.Fatalf("failed to append certs to pool")
	}

	req, err := http.NewRequest("GET", "https://example-upstream.test/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer TOKEN")

	// Dial the proxy directly, but use the upstream's SNI and strictly verify against the proxy CA pool
	dialer := &tls.Dialer{
		Config: &tls.Config{
			RootCAs:    caCertPool,
			ServerName: "example-upstream.test", // explicitly set SNI to match upstream
		},
	}
	
	conn, err := dialer.DialContext(context.Background(), "tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to dial proxy: %v", err)
	}
	defer conn.Close()

	// Write the HTTP request to the TLS connection
	err = req.Write(conn)
	if err != nil {
		t.Fatalf("failed to write request: %v", err)
	}

	// Read the response cleanly via bufio
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got: %d", resp.StatusCode)
	}

	if auth := resp.Header.Get("X-Received-Auth"); auth != "Bearer super-secret-token" {
		t.Errorf("expected injected header, got: %q", auth)
	}
}
