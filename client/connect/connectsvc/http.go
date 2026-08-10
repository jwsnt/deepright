package connectsvc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServiceURL = "http://deepright.cn"
	healthPath        = "/api/connect/health"
	metaPath          = "/api/connect/meta"
	requestPath       = "/api/connect/request"
	responsePath      = "/api/connect/response"
)

type apiEnvelope struct {
	Status  int             `json:"status"`
	Content string          `json:"content"`
	Data    json.RawMessage `json:"data"`
}

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	baseURL = normalizeServiceBaseURL(baseURL)
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func DefaultServiceBaseURL() string {
	if env := strings.TrimSpace(osEnv("CONNECT_ADDR")); env != "" {
		return normalizeServiceBaseURL(env)
	}
	return defaultServiceURL
}

func ServiceBaseURLFromFlags(flags map[string]string) string {
	if host := FirstValue(flags, "host"); host != "" {
		return normalizeServiceBaseURL(host)
	}
	if addr := FirstValue(flags, "addr"); addr != "" {
		return normalizeServiceBaseURL(addr)
	}
	if port := FirstValue(flags, "port"); port != "" {
		return normalizeServiceBaseURL("127.0.0.1:" + port)
	}
	return DefaultServiceBaseURL()
}

func ListenAddrFromFlags(flags map[string]string) string {
	if addr := FirstValue(flags, "addr"); addr != "" {
		return normalizeListenAddr(addr)
	}
	if port := FirstValue(flags, "port"); port != "" {
		return "127.0.0.1:" + strings.TrimSpace(port)
	}
	return "127.0.0.1:18080"
}

func (c *APIClient) Health() error {
	return c.doJSON(http.MethodGet, healthPath, nil, nil)
}

func (c *APIClient) CreateMeta(input MetaInput) (*Meta, error) {
	values := url.Values{}
	values.Set("key", input.Key)
	values.Set("meta", input.Meta)
	values.Set("stream", strconv.FormatBool(input.Stream))
	values.Set("callback", input.Callback)
	values.Set("agent", input.AgentID)
	values.Set("chatId", input.ChatID)
	values.Set("model", input.Model)
	values.Set("thinking", strconv.FormatBool(input.Thinking))
	values.Set("verify", strconv.FormatBool(input.Verify))
	values.Set("router_disable", strconv.FormatBool(input.RouterDisable))
	var item Meta
	if err := c.doJSON(http.MethodPost, metaPath, values, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *APIClient) UpdateMeta(key string, patch MetaUpdate) (*Meta, error) {
	values := url.Values{}
	values.Set("key", key)
	if patch.Meta != nil {
		values.Set("meta", *patch.Meta)
	}
	if patch.Stream != nil {
		values.Set("stream", strconv.FormatBool(*patch.Stream))
	}
	if patch.Callback != nil {
		values.Set("callback", *patch.Callback)
	}
	if patch.AgentID != nil {
		values.Set("agent", *patch.AgentID)
	}
	if patch.ChatID != nil {
		values.Set("chatId", *patch.ChatID)
	}
	if patch.Model != nil {
		values.Set("model", *patch.Model)
	}
	if patch.Thinking != nil {
		values.Set("thinking", strconv.FormatBool(*patch.Thinking))
	}
	if patch.Verify != nil {
		values.Set("verify", strconv.FormatBool(*patch.Verify))
	}
	if patch.RouterDisable != nil {
		values.Set("router_disable", strconv.FormatBool(*patch.RouterDisable))
	}
	var item Meta
	if err := c.doJSON(http.MethodPut, metaPath, values, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *APIClient) DeleteMeta(key string) (*Meta, error) {
	values := url.Values{}
	values.Set("key", key)
	var item Meta
	if err := c.doJSON(http.MethodDelete, metaPath, values, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *APIClient) GetMeta(key string, includeDeleted bool) (*Meta, error) {
	values := url.Values{}
	values.Set("key", key)
	if includeDeleted {
		values.Set("includeDeleted", "true")
	}
	var item Meta
	if err := c.doJSON(http.MethodGet, metaPath, values, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *APIClient) GetMetaConfig(key string, includeDeleted bool) (*MetaConfig, error) {
	values := url.Values{}
	values.Set("key", key)
	values.Set("view", "config")
	if includeDeleted {
		values.Set("includeDeleted", "true")
	}
	var item MetaConfig
	if err := c.doJSON(http.MethodGet, metaPath, values, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *APIClient) ListMeta(includeDeleted bool) ([]Meta, error) {
	values := url.Values{}
	if includeDeleted {
		values.Set("includeDeleted", "true")
	}
	var items []Meta
	if err := c.doJSON(http.MethodGet, metaPath, values, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *APIClient) ListMetaConfig(includeDeleted bool) ([]MetaConfig, error) {
	values := url.Values{}
	values.Set("view", "config")
	if includeDeleted {
		values.Set("includeDeleted", "true")
	}
	var items []MetaConfig
	if err := c.doJSON(http.MethodGet, metaPath, values, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *APIClient) AddRequest(input RequestInput) (*Request, error) {
	values := url.Values{}
	if strings.TrimSpace(input.Key) != "" {
		values.Set("key", input.Key)
	}
	if strings.TrimSpace(input.ExternalID) != "" {
		values.Set("externalId", input.ExternalID)
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		content = input.Request
	}
	values.Set("content", content)
	if strings.TrimSpace(input.Artifacts) != "" {
		values.Set("artifacts", input.Artifacts)
	}
	original := strings.TrimSpace(input.Original)
	if original == "" {
		original = input.RawRequest
	}
	if strings.TrimSpace(original) != "" {
		values.Set("original", original)
	}
	if strings.TrimSpace(input.ResponseSchema) != "" {
		values.Set("schema", input.ResponseSchema)
	}
	if strings.TrimSpace(input.MessageSnapshot) != "" {
		values.Set("messageSnapshot", input.MessageSnapshot)
	}
	if input.Status != nil {
		values.Set("status", strconv.Itoa(*input.Status))
	}
	if strings.TrimSpace(input.CreatedAt) != "" {
		values.Set("created", input.CreatedAt)
	}
	var item Request
	if err := c.doJSON(http.MethodPost, requestPath, values, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *APIClient) ListRequests(filter RequestFilter) ([]Request, error) {
	values := url.Values{}
	if strings.TrimSpace(filter.Key) != "" {
		values.Set("key", filter.Key)
	}
	if filter.AfterID > 0 {
		values.Set("afterId", strconv.Itoa(filter.AfterID))
	}
	if filter.BeforeID > 0 {
		values.Set("beforeId", strconv.Itoa(filter.BeforeID))
	}
	if filter.Limit > 0 {
		values.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Status != nil {
		values.Set("status", strconv.Itoa(*filter.Status))
	}
	var items []Request
	if err := c.doJSON(http.MethodGet, requestPath, values, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *APIClient) UpdateRequestStatus(input RequestStatusUpdate) (*Request, error) {
	values := url.Values{}
	values.Set("requestId", strconv.Itoa(input.ID))
	if strings.TrimSpace(input.Key) != "" {
		values.Set("key", input.Key)
	}
	values.Set("toStatus", strconv.Itoa(input.To))
	if input.From != nil {
		values.Set("fromStatus", strconv.Itoa(*input.From))
	}
	if input.Strict {
		values.Set("strict", "true")
	}
	var item Request
	if err := c.doJSON(http.MethodPut, requestPath, values, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *APIClient) AddResponse(input ResponseInput) (*Response, error) {
	values := url.Values{}
	if strings.TrimSpace(input.Key) != "" {
		values.Set("key", input.Key)
	}
	values.Set("requestId", strconv.Itoa(input.RequestID))
	values.Set("response", input.Response)
	if strings.TrimSpace(input.Artifacts) != "" {
		values.Set("artifacts", input.Artifacts)
	}
	var item Response
	if err := c.doJSON(http.MethodPost, responsePath, values, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *APIClient) ListResponses(filter ResponseFilter) ([]Response, error) {
	values := url.Values{}
	if strings.TrimSpace(filter.Key) != "" {
		values.Set("key", filter.Key)
	}
	if filter.RequestID > 0 {
		values.Set("requestId", strconv.Itoa(filter.RequestID))
	}
	if filter.AfterID > 0 {
		values.Set("afterId", strconv.Itoa(filter.AfterID))
	}
	if filter.Limit > 0 {
		values.Set("limit", strconv.Itoa(filter.Limit))
	}
	var items []Response
	if err := c.doJSON(http.MethodGet, responsePath, values, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *APIClient) doJSON(method, path string, values url.Values, out any) error {
	baseURL := normalizeServiceBaseURL(c.BaseURL)
	target := strings.TrimRight(baseURL, "/") + path
	if len(values) > 0 {
		target += "?" + values.Encode()
	}
	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("connect service is not running at %s; start it first with ./connect start: %w", baseURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err == nil && (env.Status != 0 || len(env.Data) > 0 || env.Content != "") {
		if env.Status != 0 {
			if env.Content == "" {
				env.Content = "unknown connect service error"
			}
			return fmt.Errorf(env.Content)
		}
		if out != nil && len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return err
			}
		}
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		text := strings.TrimSpace(string(body))
		if text == "" {
			text = resp.Status
		}
		return fmt.Errorf(text)
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func (c *APIClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	c.HTTPClient = &http.Client{Timeout: 3 * time.Second}
	return c.HTTPClient
}

func NewHTTPHandler(svc *Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(healthPath, handleHealth())
	mux.HandleFunc(metaPath, handleMetaHTTP(svc))
	mux.HandleFunc(requestPath, handleRequestHTTP(svc))
	mux.HandleFunc(responsePath, handleResponseHTTP(svc))
	return mux
}

type ServiceProvider func(*http.Request) (*Service, func(), error)

func NewDynamicHTTPHandler(provider ServiceProvider) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(healthPath, handleHealth())
	mux.HandleFunc(metaPath, dynamicServiceHandler(provider, handleMetaHTTP))
	mux.HandleFunc(requestPath, dynamicServiceHandler(provider, handleRequestHTTP))
	mux.HandleFunc(responsePath, dynamicServiceHandler(provider, handleResponseHTTP))
	return mux
}

func dynamicServiceHandler(provider ServiceProvider, builder func(*Service) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if provider == nil {
			writeAPIErrorWithCode(w, http.StatusInternalServerError, fmt.Errorf("connect service is not initialized"))
			return
		}
		svc, release, err := provider(r)
		if err != nil {
			writeAPIErrorWithCode(w, http.StatusInternalServerError, err)
			return
		}
		if svc == nil {
			writeAPIErrorWithCode(w, http.StatusInternalServerError, fmt.Errorf("connect service is not initialized"))
			return
		}
		if release != nil {
			defer release()
		}
		builder(svc).ServeHTTP(w, r)
	}
}

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": 0})
	}
}

func handleMetaHTTP(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			item, err := svc.CreateMeta(MetaInput{
				Key:           q.Get("key"),
				Meta:          q.Get("meta"),
				Stream:        queryBool(q.Get("stream")),
				Callback:      q.Get("callback"),
				AgentID:       FirstValueFromQuery(q, "agent", "agentId"),
				ChatID:        FirstValueFromQuery(q, "chat", "chatId"),
				Model:         q.Get("model"),
				Thinking:      queryBool(q.Get("thinking")),
				Verify:        queryBool(q.Get("verify")),
				RouterDisable: resolveRouterDisableQueryValue(q, true),
			})
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, item)
		case http.MethodPut:
			key := FirstValueFromQuery(q, "key")
			patch := MetaUpdate{}
			if _, ok := q["meta"]; ok {
				value := q.Get("meta")
				patch.Meta = &value
			}
			if _, ok := q["stream"]; ok {
				value := queryBool(q.Get("stream"))
				patch.Stream = &value
			}
			if _, ok := q["callback"]; ok {
				value := q.Get("callback")
				patch.Callback = &value
			}
			if value := FirstValueFromQuery(q, "agent", "agentId"); value != "" {
				patch.AgentID = &value
			}
			if value := FirstValueFromQuery(q, "chat", "chatId"); value != "" {
				patch.ChatID = &value
			}
			if _, ok := q["model"]; ok {
				value := q.Get("model")
				patch.Model = &value
			}
			if _, ok := q["thinking"]; ok {
				value := queryBool(q.Get("thinking"))
				patch.Thinking = &value
			}
			if _, ok := q["verify"]; ok {
				value := queryBool(q.Get("verify"))
				patch.Verify = &value
			}
			if value, ok := resolveRouterDisableQueryPatch(q); ok {
				patch.RouterDisable = &value
			}
			item, err := svc.UpdateMeta(key, patch)
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, item)
		case http.MethodDelete:
			item, err := svc.DeleteMeta(FirstValueFromQuery(q, "key"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, item)
		case http.MethodGet:
			if key := FirstValueFromQuery(q, "key"); key != "" {
				includeDeleted := queryBool(FirstValueFromQuery(q, "includeDeleted", "include-deleted"))
				if strings.EqualFold(FirstValueFromQuery(q, "view"), "config") {
					item, err := svc.GetMetaConfig(key, includeDeleted)
					if err != nil {
						writeAPIError(w, err)
						return
					}
					writeAPIData(w, item)
					return
				}
				item, err := svc.GetMeta(key, includeDeleted)
				if err != nil {
					writeAPIError(w, err)
					return
				}
				writeAPIData(w, item)
				return
			}
			if strings.EqualFold(FirstValueFromQuery(q, "view"), "config") {
				items, err := svc.ListMetaConfig(queryBool(FirstValueFromQuery(q, "includeDeleted", "include-deleted")))
				if err != nil {
					writeAPIError(w, err)
					return
				}
				writeAPIData(w, items)
				return
			}
			items, err := svc.ListMeta(queryBool(FirstValueFromQuery(q, "includeDeleted", "include-deleted")))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, items)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleRequestHTTP(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			status, err := parseOptionalIntValue(q.Get("status"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			item, err := svc.AddRequest(RequestInput{
				Key:             FirstValueFromQuery(q, "key"),
				ExternalID:      FirstValueFromQuery(q, "externalId", "external-id"),
				Content:         FirstValueFromQuery(q, "content"),
				Request:         q.Get("request"),
				Artifacts:       q.Get("artifacts"),
				Original:        FirstValueFromQuery(q, "original"),
				RawRequest:      FirstValueFromQuery(q, "rawRequest", "raw-request"),
				ResponseSchema:  FirstValueFromQuery(q, "schema"),
				MessageSnapshot: FirstValueFromQuery(q, "messageSnapshot", "message-snapshot"),
				Status:          status,
				CreatedAt:       FirstValueFromQuery(q, "created"),
			})
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, item)
		case http.MethodPut:
			requestID, err := parseRequiredIntValue(FirstValueFromQuery(q, "requestId", "request-id"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			toStatus, err := parseRequiredIntValue(FirstValueFromQuery(q, "toStatus", "to-status"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			fromStatus, err := parseOptionalIntValue(FirstValueFromQuery(q, "fromStatus", "from-status"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			item, err := svc.UpdateRequestStatus(RequestStatusUpdate{
				ID:     requestID,
				Key:    FirstValueFromQuery(q, "key"),
				From:   fromStatus,
				To:     toStatus,
				Strict: queryBool(FirstValueFromQuery(q, "strict")),
			})
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, item)
		case http.MethodGet:
			afterID, err := parseOptionalIntValue(FirstValueFromQuery(q, "afterId", "after-id"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			beforeID, err := parseOptionalIntValue(FirstValueFromQuery(q, "beforeId", "before-id"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			limit, err := parseOptionalIntValue(q.Get("limit"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			status, err := parseOptionalIntValue(q.Get("status"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			size := 100
			if limit != nil {
				size = *limit
			}
			items, err := svc.ListRequests(RequestFilter{
				Key:      FirstValueFromQuery(q, "key"),
				AfterID:  intPtrValue(afterID),
				BeforeID: intPtrValue(beforeID),
				Limit:    size,
				Status:   status,
			})
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, items)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleResponseHTTP(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			requestID, err := parseRequiredIntValue(FirstValueFromQuery(q, "requestId", "request-id"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			item, err := svc.AddResponse(ResponseInput{
				Key:       FirstValueFromQuery(q, "key"),
				RequestID: requestID,
				Response:  q.Get("response"),
				Artifacts: q.Get("artifacts"),
			})
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, item)
		case http.MethodGet:
			requestID, err := parseOptionalIntValue(FirstValueFromQuery(q, "requestId", "request-id"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			afterID, err := parseOptionalIntValue(FirstValueFromQuery(q, "afterId", "after-id"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			limit, err := parseOptionalIntValue(q.Get("limit"))
			if err != nil {
				writeAPIError(w, err)
				return
			}
			size := 100
			if limit != nil {
				size = *limit
			}
			items, err := svc.ListResponses(ResponseFilter{
				Key:       FirstValueFromQuery(q, "key"),
				RequestID: intPtrValue(requestID),
				AfterID:   intPtrValue(afterID),
				Limit:     size,
			})
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeAPIData(w, items)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func writeAPIData(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"status": 0, "data": data})
}

func writeAPIError(w http.ResponseWriter, err error) {
	writeAPIErrorWithCode(w, http.StatusBadRequest, err)
}

func writeAPIErrorWithCode(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": 1, "content": err.Error()})
}

func queryBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func resolveRouterDisableQueryValue(values url.Values, fallback bool) bool {
	if _, ok := values["router_disable"]; ok {
		return queryBool(values.Get("router_disable"))
	}
	return fallback
}

func resolveRouterDisableQueryPatch(values url.Values) (bool, bool) {
	if _, ok := values["router_disable"]; ok {
		return queryBool(values.Get("router_disable")), true
	}
	return false, false
}

func ParseFormBoolValue(form url.Values, key string, fallback bool) (bool, bool) {
	values, ok := form[key]
	if !ok || len(values) == 0 {
		return fallback, false
	}
	raw := strings.TrimSpace(values[len(values)-1])
	if raw == "" {
		return fallback, true
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return fallback, true
	}
}

func ResolveRouterDisableFormValue(form url.Values, fallback bool) bool {
	if value, ok := ParseFormBoolValue(form, "router_disable", fallback); ok {
		return value
	}
	return fallback
}

func ParsePluginConfigRequest(r *http.Request) (MetaInput, error) {
	if r == nil {
		return MetaInput{}, fmt.Errorf("request is required")
	}
	if r.ContentLength != 0 && strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		return parsePluginConfigJSONRequest(r)
	}
	if err := r.ParseForm(); err != nil {
		return MetaInput{}, err
	}
	first := func(keys ...string) string {
		for _, key := range keys {
			value := strings.TrimSpace(r.FormValue(key))
			if value != "" {
				return value
			}
		}
		return ""
	}
	boolValue := func(key string, fallback bool) bool {
		raw := strings.TrimSpace(r.FormValue(key))
		if raw == "" {
			return fallback
		}
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		default:
			return fallback
		}
	}
	return MetaInput{
		Key:           first("key"),
		Meta:          first("meta"),
		Stream:        boolValue("stream", false),
		AgentID:       first("agentId", "agent"),
		ChatID:        first("chatId", "chat"),
		Model:         first("model"),
		Thinking:      boolValue("thinking", false),
		Verify:        boolValue("verify", false),
		RouterDisable: ResolveRouterDisableFormValue(r.Form, true),
	}, nil
}

func parsePluginConfigJSONRequest(r *http.Request) (MetaInput, error) {
	var payload struct {
		Key           string          `json:"key"`
		Meta          json.RawMessage `json:"meta"`
		Stream        bool            `json:"stream"`
		AgentID       string          `json:"agentId"`
		Agent         string          `json:"agent"`
		ChatID        string          `json:"chatId"`
		Chat          string          `json:"chat"`
		Model         string          `json:"model"`
		Thinking      bool            `json:"thinking"`
		Verify        bool            `json:"verify"`
		RouterDisable *bool           `json:"router_disable"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		return MetaInput{}, fmt.Errorf("invalid plugin config json: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return MetaInput{}, fmt.Errorf("invalid plugin config json: multiple values")
		}
		return MetaInput{}, fmt.Errorf("invalid plugin config json: %w", err)
	}

	meta := strings.TrimSpace(string(payload.Meta))
	if len(meta) >= 2 && meta[0] == '"' {
		if err := json.Unmarshal(payload.Meta, &meta); err != nil {
			return MetaInput{}, fmt.Errorf("invalid plugin config meta: %w", err)
		}
		meta = strings.TrimSpace(meta)
	}
	if meta == "" {
		meta = "{}"
	}
	if !json.Valid([]byte(meta)) {
		return MetaInput{}, fmt.Errorf("meta must be valid json")
	}

	routerDisable := true
	if payload.RouterDisable != nil {
		routerDisable = *payload.RouterDisable
	}
	agentID := strings.TrimSpace(payload.AgentID)
	if agentID == "" {
		agentID = strings.TrimSpace(payload.Agent)
	}
	chatID := strings.TrimSpace(payload.ChatID)
	if chatID == "" {
		chatID = strings.TrimSpace(payload.Chat)
	}
	return MetaInput{
		Key:           strings.TrimSpace(payload.Key),
		Meta:          meta,
		Stream:        payload.Stream,
		AgentID:       agentID,
		ChatID:        chatID,
		Model:         strings.TrimSpace(payload.Model),
		Thinking:      payload.Thinking,
		Verify:        payload.Verify,
		RouterDisable: routerDisable,
	}, nil
}

func ParsePluginActionRequest(r *http.Request) (string, map[string]string, error) {
	if err := r.ParseForm(); err != nil {
		return "", nil, err
	}
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" {
		return "", nil, fmt.Errorf("key is required")
	}
	flags := make(map[string]string)
	for key, values := range r.Form {
		if key == "key" || len(values) == 0 {
			continue
		}
		flags[key] = strings.TrimSpace(values[len(values)-1])
	}
	return key, flags, nil
}

func parseRequiredIntValue(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("integer value is required")
	}
	return strconv.Atoi(raw)
}

func parseOptionalIntValue(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("integer value is invalid: %s", raw)
	}
	return &value, nil
}

func intPtrValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func FirstValueFromQuery(q url.Values, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(q.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func normalizeServiceBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultServiceURL
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return strings.TrimRight(raw, "/")
	}
	return "http://" + strings.TrimRight(raw, "/")
}

func normalizeListenAddr(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "127.0.0.1:18080"
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		if parsed, err := url.Parse(raw); err == nil && strings.TrimSpace(parsed.Host) != "" {
			return parsed.Host
		}
	}
	return raw
}

var osEnv = func(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
