package runtimehost

import (
	"net/http"
	"testing"
)

func TestNewClientDefaultHTTPClientForcesHTTP11(t *testing.T) {
	client := NewClient("https://www.deepright.cn", nil)
	transport, ok := client.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", client.HTTPClient.Transport)
	}
	if transport.ForceAttemptHTTP2 {
		t.Fatalf("ForceAttemptHTTP2 = true, want false")
	}
	if len(transport.TLSNextProto) != 0 {
		t.Fatalf("TLSNextProto length = %d, want 0", len(transport.TLSNextProto))
	}
}
