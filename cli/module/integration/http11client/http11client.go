package http11client

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

type Options struct {
	Timeout               time.Duration
	ConnectTimeout        time.Duration
	KeepAlive             time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	ResponseHeaderTimeout time.Duration
	TLSHandshakeTimeout   time.Duration
	TLSClientConfig       *tls.Config
}

func NewClient(opts Options) *http.Client {
	return &http.Client{
		Transport: NewTransport(opts),
		Timeout:   opts.Timeout,
	}
}

func NewTransport(opts Options) *http.Transport {
	keepAlive := opts.KeepAlive
	if keepAlive <= 0 {
		keepAlive = 60 * time.Second
	}
	maxIdleConns := opts.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 100
	}
	maxIdleConnsPerHost := opts.MaxIdleConnsPerHost
	if maxIdleConnsPerHost <= 0 {
		maxIdleConnsPerHost = 100
	}
	idleConnTimeout := opts.IdleConnTimeout
	if idleConnTimeout <= 0 {
		idleConnTimeout = 90 * time.Second
	}
	tlsHandshakeTimeout := opts.TLSHandshakeTimeout
	if tlsHandshakeTimeout <= 0 {
		tlsHandshakeTimeout = 10 * time.Second
	}

	dialer := &net.Dialer{
		Timeout:   opts.ConnectTimeout,
		KeepAlive: keepAlive,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if tc, ok := conn.(*net.TCPConn); ok {
				_ = tc.SetNoDelay(true)
			}
			return conn, nil
		},
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       idleConnTimeout,
		ResponseHeaderTimeout: opts.ResponseHeaderTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ForceAttemptHTTP2:     false,
		// Keep ALPN from upgrading the client to HTTP/2.
		TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
	}
	if opts.TLSClientConfig != nil {
		transport.TLSClientConfig = opts.TLSClientConfig.Clone()
	}
	return transport
}
