package sharedutil

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// RemoteHost extracts the host part from an HTTP request's remote address.
func RemoteHost(addr string) string {
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// IsLocalExecutionRequest checks if the request originates from localhost.
func IsLocalExecutionRequest(r *http.Request) bool {
	host := strings.ToLower(RemoteHost(r.RemoteAddr))
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == ""
}

// WithLocalCORS wraps a handler with permissive CORS headers for local use.
func WithLocalCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ContainsBlockedCommand checks if a shell command contains dangerous patterns.
func ContainsBlockedCommand(cmd string) bool {
	blocked := []string{
		"rm -rf /", "rm -rf /*", "mkfs.", "dd if=", ":(){ :|:& };:",
		"chmod -R 000 /", "shutdown -h now", "reboot", "init 0", "poweroff",
	}
	lower := strings.ToLower(strings.TrimSpace(cmd))
	for _, b := range blocked {
		if strings.Contains(lower, b) {
			return true
		}
	}
	return false
}

// NewProxyClient creates an HTTP client with only a connect timeout.
func NewProxyClient(connectTimeout time.Duration) *http.Client {
	if connectTimeout <= 0 {
		connectTimeout = 5 * time.Second
	}
	return &http.Client{
		Timeout: connectTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   connectTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}
}
