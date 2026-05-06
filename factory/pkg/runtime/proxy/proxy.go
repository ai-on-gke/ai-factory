package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
)

// Server represents the egress reverse proxy server.
type Server struct {
	Config *ProxyConfig
	Proxy  *httputil.ReverseProxy
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

	return &Server{
		Config: config,
		Proxy:  proxy,
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
