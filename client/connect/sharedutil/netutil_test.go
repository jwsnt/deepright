package sharedutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRemoteHost(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{"empty", "", ""},
		{"ipv4 with port", "192.168.1.1:8080", "192.168.1.1"},
		{"ipv6 with port", "[::1]:8080", "::1"},
		{"hostname with port", "localhost:8080", "localhost"},
		{"no port", "192.168.1.1", "192.168.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoteHost(tt.addr); got != tt.want {
				t.Errorf("RemoteHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsLocalExecutionRequest(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want bool
	}{
		{"localhost", "localhost:8080", true},
		{"127.0.0.1", "127.0.0.1:8080", true},
		{"ipv6 local", "[::1]:8080", true},
		{"empty", "", true},
		{"remote ip", "10.0.0.1:8080", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{RemoteAddr: tt.addr}
			if got := IsLocalExecutionRequest(r); got != tt.want {
				t.Errorf("IsLocalExecutionRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithLocalCORS(t *testing.T) {
	handler := WithLocalCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tests := []struct {
		name   string
		method string
		want   int
	}{
		{"GET", "GET", http.StatusOK},
		{"POST", "POST", http.StatusOK},
		{"OPTIONS", "OPTIONS", http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, ts.URL, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.want {
				t.Errorf("%s status = %d, want %d", tt.method, resp.StatusCode, tt.want)
			}
		})
	}
}

func TestContainsBlockedCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"safe", "ls -la", false},
		{"safe empty", "", false},
		{"rm -rf /", "rm -rf /", true},
		{"rm -rf /*", "rm -rf /*", true},
		{"mkfs", "mkfs.ext4 /dev/sda", true},
		{"dd if=", "dd if=/dev/zero of=/dev/sda", true},
		{"fork bomb", ":(){ :|:& };:", true},
		{"reboot", "sudo reboot", true},
		{"partial rm check", "rm -rf /tmp/foo", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsBlockedCommand(tt.cmd); got != tt.want {
				t.Errorf("ContainsBlockedCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestNewProxyClient(t *testing.T) {
	tests := []struct {
		name           string
		connectTimeout time.Duration
	}{
		{"default timeout", 0},
		{"custom timeout", 10 * time.Second},
		{"negative", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewProxyClient(tt.connectTimeout)
			if client == nil {
				t.Fatal("NewProxyClient() returned nil")
			}
			if client.Timeout <= 0 {
				t.Errorf("NewProxyClient() timeout = %v, want > 0", client.Timeout)
			}
		})
	}
}
