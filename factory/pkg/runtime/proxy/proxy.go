package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/ai-on-gke/ai-factory/factory/pkg/runtime/proxy/tls"
)

// Server represents the egress reverse proxy server.
type Server struct {
	Config         *ProxyConfig
	Proxy          *httputil.ReverseProxy
	tlsInterceptor *tls.Interceptor
}

// NewServer creates a new Server instance based on the provided configuration.
func NewServer(config *ProxyConfig) *Server {
	director := func(req *http.Request) {
		// For an egress proxy, the request URL should already contain the target.
		// We ensure the Scheme and Host are set.
		if req.URL.Scheme == "" {
			if req.TLS != nil {
				req.URL.Scheme = "https"
			} else {
				req.URL.Scheme = "http"
			}
		}
		if req.URL.Host == "" {
			req.URL.Host = req.Host
		}
	}

	proxy := &httputil.ReverseProxy{
		Director: director,
		ModifyResponse: func(res *http.Response) error {
			// Hook for future response filtering logic
			return nil
		},
	}

	var tlsInterceptor *tls.Interceptor
	if config.Spec.TLS != nil {
		caManager, err := tls.NewCAManager(config.Spec.TLS.CACertFile, config.Spec.TLS.CAKeyFile, config.Spec.TLS.TrustBundleExportPath)
		if err != nil {
			log.Fatalf("failed to initialize CA manager: %v", err) // TODO: Better error handling, maybe return error
		}
		cg := tls.NewCertGenerator(caManager)
		tlsInterceptor = tls.NewInterceptor(cg)
	}

	return &Server{
		Config:         config,
		Proxy:          proxy,
		tlsInterceptor: tlsInterceptor,
	}
}

// Start starts the proxy server(s) based on the configuration.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 2)

	if s.Config.Spec.Listen.HTTPPort != 0 {
		httpServer := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", s.Config.Spec.Listen.Address, s.Config.Spec.Listen.HTTPPort),
			Handler: s,
		}
		go func() {
			log.Printf("Starting HTTP proxy on %s", httpServer.Addr)
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("http server error: %w", err)
			}
		}()
		go func() {
			<-ctx.Done()
			httpServer.Shutdown(context.Background())
		}()
	}

	if s.Config.Spec.Listen.HTTPSPort != 0 && s.tlsInterceptor != nil {
		httpsServer := &http.Server{
			Addr:      fmt.Sprintf("%s:%d", s.Config.Spec.Listen.Address, s.Config.Spec.Listen.HTTPSPort),
			Handler:   s,
			TLSConfig: s.tlsInterceptor.TLSConfig(),
		}
		go func() {
			log.Printf("Starting HTTPS proxy on %s", httpsServer.Addr)
			// Pass empty cert/key paths because we are using GetCertificate in TLSConfig
			if err := httpsServer.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("https server error: %w", err)
			}
		}()
		go func() {
			<-ctx.Done()
			httpsServer.Shutdown(context.Background())
		}()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Give servers a little time to shutdown cleanly before returning
		time.Sleep(100 * time.Millisecond)
		return nil
	}
}

// ServeHTTP handles incoming requests, applying policy matching and header injection.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Block CONNECT requests since we don't support TLS tunneling yet.
	if req.Method == http.MethodConnect {
		http.Error(w, "CONNECT method not supported", http.StatusMethodNotAllowed)
		return
	}

	rule := MatchRequest(req, s.Config.Spec.Rules)
	if rule == nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := InjectHeader(req, rule); err != nil {
		log.Printf("Header injection error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	s.Proxy.ServeHTTP(w, req)
}
