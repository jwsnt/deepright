package standalone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"integration/http11client"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const Endpoint = "/api/standalone"

type Snapshot struct {
	Enabled        bool `json:"enabled"`
	StartupEnabled bool `json:"startupEnabled"`
	Overridden     bool `json:"overridden"`
	RuntimeOnly    bool `json:"runtimeOnly"`
}

type State struct {
	mu              sync.RWMutex
	startupEnabled  bool
	overrideSet     bool
	overrideEnabled bool
}

type apiResponse struct {
	Status  int      `json:"status"`
	Content string   `json:"content,omitempty"`
	Data    Snapshot `json:"data"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(startupEnabled bool) *State {
	return &State{startupEnabled: startupEnabled}
}

func (s *State) Enabled() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.overrideSet {
		return s.overrideEnabled
	}
	return s.startupEnabled
}

func (s *State) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{RuntimeOnly: true}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	enabled := s.startupEnabled
	overridden := false
	if s.overrideSet {
		enabled = s.overrideEnabled
		overridden = true
	}
	return Snapshot{
		Enabled:        enabled,
		StartupEnabled: s.startupEnabled,
		Overridden:     overridden,
		RuntimeOnly:    true,
	}
}

func (s *State) Set(enabled bool) Snapshot {
	if s == nil {
		return Snapshot{Enabled: enabled, RuntimeOnly: true}
	}
	s.mu.Lock()
	s.overrideSet = true
	s.overrideEnabled = enabled
	s.mu.Unlock()
	return s.Snapshot()
}

func (s *State) Reset() Snapshot {
	if s == nil {
		return Snapshot{RuntimeOnly: true}
	}
	s.mu.Lock()
	s.overrideSet = false
	s.overrideEnabled = false
	s.mu.Unlock()
	return s.Snapshot()
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if httpClient == nil {
		httpClient = http11client.NewClient(http11client.Options{Timeout: 5 * time.Second})
	}
	return &Client{BaseURL: baseURL, HTTPClient: httpClient}
}

func (c *Client) Get(ctx context.Context) (Snapshot, error) {
	return c.do(ctx, http.MethodGet, Endpoint, nil)
}

func (c *Client) Set(ctx context.Context, enabled bool) (Snapshot, error) {
	payload := map[string]any{"enabled": enabled}
	path := Endpoint
	if enabled {
		path = Endpoint + "=true"
	} else {
		path = Endpoint + "=false"
	}
	return c.do(ctx, http.MethodPost, path, payload)
}

func (c *Client) Reset(ctx context.Context) (Snapshot, error) {
	return c.do(ctx, http.MethodDelete, Endpoint, nil)
}

func (c *Client) do(ctx context.Context, method, path string, payload map[string]any) (Snapshot, error) {
	if c == nil {
		return Snapshot{}, fmt.Errorf("standalone client is nil")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		return Snapshot{}, fmt.Errorf("standalone base URL is empty")
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return Snapshot{}, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return Snapshot{}, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http11client.NewClient(http11client.Options{Timeout: 5 * time.Second})
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return Snapshot{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Snapshot{}, err
	}

	var decoded apiResponse
	if len(bytes.TrimSpace(data)) != 0 {
		if err := json.Unmarshal(data, &decoded); err != nil {
			return Snapshot{}, err
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || decoded.Status != 0 {
		message := strings.TrimSpace(decoded.Content)
		if message == "" {
			message = strings.TrimSpace(string(data))
		}
		if message == "" {
			message = resp.Status
		}
		return Snapshot{}, fmt.Errorf(message)
	}

	return decoded.Data, nil
}
