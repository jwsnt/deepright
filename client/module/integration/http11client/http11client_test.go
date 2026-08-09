package http11client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientUsesHTTP11AgainstHTTP2Server(t *testing.T) {
	var seenProto string
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenProto = r.Proto
		_, _ = io.WriteString(w, "ok")
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	serverTransport, ok := server.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("server client transport type = %T", server.Client().Transport)
	}

	client := NewClient(Options{
		Timeout:         time.Second,
		TLSClientConfig: serverTransport.TLSClientConfig,
	})
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("client get: %v", err)
	}
	defer resp.Body.Close()

	if seenProto != "HTTP/1.1" {
		t.Fatalf("request protocol = %q, want HTTP/1.1", seenProto)
	}
}

func TestNewTransportDisablesHTTP2Negotiation(t *testing.T) {
	transport := NewTransport(Options{})
	if transport.ForceAttemptHTTP2 {
		t.Fatalf("ForceAttemptHTTP2 = true, want false")
	}
	if len(transport.TLSNextProto) != 0 {
		t.Fatalf("TLSNextProto length = %d, want 0", len(transport.TLSNextProto))
	}
}
