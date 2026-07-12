package feishusvc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	lark "github.com/larksuite/oapi-sdk-go/v3"
)

var (
	currentGOOS   = runtime.GOOS
	currentGOARCH = runtime.GOARCH

	userHomeDirFn                = os.UserHomeDir
	keychainStatFn               = os.Stat
	exportKeychainCertificatesFn = func(ctx context.Context, keychainPath string) ([]byte, error) {
		output, err := exec.CommandContext(ctx, "security", "find-certificate", "-a", "-p", keychainPath).CombinedOutput()
		if err != nil {
			text := strings.TrimSpace(string(output))
			if text == "" {
				return nil, err
			}
			return nil, fmt.Errorf("%w: %s", err, text)
		}
		return output, nil
	}

	feishuNetworkCompatOnce sync.Once
	feishuNetworkCompatErr  error
	feishuTLSConfigOnce     sync.Once
	feishuTLSConfigValue    *tls.Config
	feishuTLSConfigErr      error
)

var newFeishuLarkClient = buildFeishuLarkClient

func buildFeishuLarkClient(cfg Config) (*lark.Client, error) {
	if err := installFeishuNetworkCompat(); err != nil {
		return nil, err
	}
	return lark.NewClient(
		cfg.AppID,
		cfg.AppSecret,
		lark.WithHttpClient(newFeishuHTTPClient(0)),
	), nil
}

func installFeishuNetworkCompat() error {
	if !needsFeishuTLSCompatibility() {
		return nil
	}
	feishuNetworkCompatOnce.Do(func() {
		tlsConfig, err := feishuTLSConfig()
		if err != nil {
			feishuNetworkCompatErr = err
			return
		}
		if tlsConfig == nil {
			return
		}

		http.DefaultTransport = cloneTransportWithTLS(http.DefaultTransport, tlsConfig)
		if http.DefaultClient != nil && http.DefaultClient.Transport != nil {
			http.DefaultClient.Transport = cloneTransportWithTLS(http.DefaultClient.Transport, tlsConfig)
		}
		ws.DefaultDialer = cloneWebsocketDialer(ws.DefaultDialer, tlsConfig)
	})
	return feishuNetworkCompatErr
}

func needsFeishuTLSCompatibility() bool {
	return strings.EqualFold(strings.TrimSpace(currentGOOS), "darwin") && strings.EqualFold(strings.TrimSpace(currentGOARCH), "amd64")
}

func feishuTLSConfig() (*tls.Config, error) {
	if !needsFeishuTLSCompatibility() {
		return nil, nil
	}
	feishuTLSConfigOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pool, err := loadDarwinCertPool(ctx)
		if err != nil {
			feishuTLSConfigErr = err
			return
		}
		feishuTLSConfigValue = &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    pool,
		}
	})
	if feishuTLSConfigValue == nil {
		return nil, feishuTLSConfigErr
	}
	return feishuTLSConfigValue.Clone(), feishuTLSConfigErr
}

func newFeishuHTTPClient(timeout time.Duration) *http.Client {
	transport := cloneTransportWithTLS(http.DefaultTransport, nil)
	if tlsConfig, err := feishuTLSConfig(); err == nil && tlsConfig != nil {
		transport = cloneTransportWithTLS(http.DefaultTransport, tlsConfig)
	}
	client := &http.Client{Transport: transport}
	if timeout > 0 {
		client.Timeout = timeout
	}
	return client
}

func loadDarwinCertPool(ctx context.Context) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	loaded := 0
	var problems []string
	for _, keychain := range darwinKeychainPaths() {
		if strings.TrimSpace(keychain) == "" {
			continue
		}
		if _, err := keychainStatFn(keychain); err != nil {
			continue
		}
		pemData, err := exportKeychainCertificatesFn(ctx, keychain)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", keychain, err))
			continue
		}
		if !pool.AppendCertsFromPEM(pemData) {
			problems = append(problems, fmt.Sprintf("%s: no certificates found", keychain))
			continue
		}
		loaded++
	}
	if loaded > 0 {
		return pool, nil
	}
	if len(problems) == 0 {
		return nil, fmt.Errorf("load macOS trust store: no readable keychains")
	}
	return nil, fmt.Errorf("load macOS trust store: %s", strings.Join(problems, "; "))
}

func darwinKeychainPaths() []string {
	paths := []string{
		"/System/Library/Keychains/SystemRootCertificates.keychain",
		"/Library/Keychains/System.keychain",
	}
	home, err := userHomeDirFn()
	if err == nil && strings.TrimSpace(home) != "" {
		paths = append(paths, filepath.Join(home, "Library", "Keychains", "login.keychain-db"))
	}
	return paths
}

func cloneTransportWithTLS(base http.RoundTripper, tlsConfig *tls.Config) *http.Transport {
	var transport *http.Transport
	if existing, ok := base.(*http.Transport); ok && existing != nil {
		transport = existing.Clone()
	} else {
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:   true,
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	}
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig.Clone()
	}
	return transport
}

func cloneWebsocketDialer(base *ws.Dialer, tlsConfig *tls.Config) *ws.Dialer {
	if base == nil {
		base = ws.DefaultDialer
	}
	clone := &ws.Dialer{
		NetDial:           base.NetDial,
		NetDialContext:    base.NetDialContext,
		NetDialTLSContext: base.NetDialTLSContext,
		Proxy:             base.Proxy,
		HandshakeTimeout:  base.HandshakeTimeout,
		ReadBufferSize:    base.ReadBufferSize,
		WriteBufferSize:   base.WriteBufferSize,
		WriteBufferPool:   base.WriteBufferPool,
		Subprotocols:      append([]string{}, base.Subprotocols...),
		EnableCompression: base.EnableCompression,
		Jar:               base.Jar,
	}
	if tlsConfig != nil {
		clone.TLSClientConfig = tlsConfig.Clone()
	} else if base.TLSClientConfig != nil {
		clone.TLSClientConfig = base.TLSClientConfig.Clone()
	}
	return clone
}
