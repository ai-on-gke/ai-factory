package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"

	"net/http"
	"net/http/httptest"
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

	// Create a client to talk to the upstream from the proxy's director/transport
	// Actually, the reverse proxy uses the default transport. Let's make sure it trusts the upstream
	// By default httptest server uses a self-signed cert.
	// We need to customize the proxy's transport to trust the upstream httptest server.
	
	// Create proxy config
	configYAML := fmt.Sprintf(`
apiVersion: factory.ai.gke.io/v1alpha1
kind: ProxySpec
spec:
  listen:
    address: 127.0.0.1
    httpsPort: 8444
  tls:
    trustBundleExportPath: %s
  rules:
    - name: rule1
      allowedURLs: ["%s/*"]
      allowedVerbs: ["GET"]
      injection:
        header: Authorization
        placeholder: TOKEN
        secretFile: %s
`, trustBundleExportPath, upstream.URL, secretPath)

	config, err := proxy.ParseConfig([]byte(configYAML))
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Override port to use dynamic allocation
	config.Spec.Listen.HTTPSPort = 0 // Actually we have to use httptest server to get a dynamic port or let net.Listen pick one.
	
	proxyServer := proxy.NewServer(config)
	
	// Customize the transport to trust the upstream test server
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // Just for this test
	proxyServer.Proxy.Transport = transport

	// Start proxy manually on a random port
	// Since proxy.Server.Start uses hardcoded ports, we'll start a test server directly
	
	// We need to initialize the TLS interceptor manually since NewServer does it but we want httptest.NewUnstartedServer
	// Actually, proxy.Server.Start takes care of creating the server.
	// Let's use a fixed high port for testing.
	config.Spec.Listen.HTTPSPort = 8444
	proxyServer = proxy.NewServer(config)
	proxyServer.Proxy.Transport = transport

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := proxyServer.Start(ctx); err != nil {
			// Ignore normal shutdown errors
		}
	}()

	// Wait for server to start and CA to be written
	time.Sleep(500 * time.Millisecond)

	// 3. Prepare client trusting the proxy CA
	caCertPEM, err := os.ReadFile(trustBundleExportPath)
	if err != nil {
		t.Fatalf("failed to read proxy trust bundle: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertPEM)

	// Extract the host:port from the upstream URL
	// The client needs to connect to the proxy, but request the upstream URL
	// In a transparent proxy, the client connects to the proxy's IP/port, but sends SNI for the upstream
	// To simulate this in Go, we set the DialTLSContext to connect to the proxy instead.


	
	// Make a custom transport that dials the proxy directly but does a TLS handshake with the upstream's SNI
	
	// Better approach for transparent proxy testing:
	// Use a custom dialer that resolves the upstream domain to the proxy's IP
	
	req, err := http.NewRequest("GET", upstream.URL+"/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer TOKEN")

	// Need to dial the proxy manually
	dialer := &tls.Dialer{
		Config: &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: true, // We dial 127.0.0.1 directly
		},
	}
	
	conn, err := dialer.DialContext(context.Background(), "tcp", "127.0.0.1:8444")
	if err != nil {
		t.Fatalf("failed to dial proxy: %v", err)
	}
	defer conn.Close()

	// Write the HTTP request to the TLS connection
	err = req.Write(conn)
	if err != nil {
		t.Fatalf("failed to write request: %v", err)
	}

	// Read the response
	// Note: this is a simple test, real clients use robust HTTP parsers over the connection
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("failed to read response: %v", err)
	}

	responseStr := string(buf[:n])
	
	if !bytes.Contains(buf[:n], []byte("200 OK")) {
		t.Errorf("expected 200 OK, got: %s", responseStr)
	}

	if !bytes.Contains(buf[:n], []byte("X-Received-Auth: Bearer super-secret-token")) {
		t.Errorf("expected injected header, got: %s", responseStr)
	}
}
