package main

import (
	"net/http"
	"testing"
	"time"
)

func TestCreateCliGetHTTPClientForcesHTTP11(t *testing.T) {
	client := createCliGetHTTPClient(&Config{
		HTTPTimeout:       45000,
		HTTPConnTimeout:   15000,
		HTTPSocketTimeout: 45000,
		IdleTimeout:       90,
	})

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", client.Transport)
	}
	if transport.ForceAttemptHTTP2 {
		t.Fatalf("ForceAttemptHTTP2 = true, want false")
	}
	if len(transport.TLSNextProto) != 0 {
		t.Fatalf("TLSNextProto length = %d, want 0", len(transport.TLSNextProto))
	}
	if client.Timeout != 45*time.Second {
		t.Fatalf("timeout = %s, want %s", client.Timeout, 45*time.Second)
	}
}
