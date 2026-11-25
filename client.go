// Package sixtydb is the official Go SDK for the 60db API (https://60db.ai).
package sixtydb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production API endpoint.
const DefaultBaseURL = "https://api.60db.ai"

// Per-call timeouts from the API contract.
const (
	timeoutDefault = 60 * time.Second
	timeoutMemory  = 120 * time.Second
	timeoutLong    = 300 * time.Second
)

// Config configures a Client.
type Config struct {
	APIKey     string
	BaseURL    string       // defaults to DefaultBaseURL
	HTTPClient *http.Client // optional; defaults to a client without global timeout
}

// Client talks to the 60db API. Safe for concurrent use.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client

	Chat   *ChatService
	Memory *MemoryService
	Judge  *JudgeService
	Music  *MusicService
	Agents *AgentsService
	Dialer *DialerService
	// CallAnalysis is the Call analysis API (/deep-analytics); needs the key's call_analysis permission.
	CallAnalysis *CallAnalysisService
}

// NewClient creates a Client. APIKey is required.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("sixtydb: APIKey is required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{}
	}
	c := &Client{apiKey: cfg.APIKey, baseURL: strings.TrimRight(base, "/"), http: hc}
	c.Chat = &ChatService{c: c}
	c.Memory = &MemoryService{c: c}
	c.Judge = newJudgeService(c)
	c.Music = &MusicService{c: c}
	c.Agents = newAgentsService(c)
	c.Dialer = newDialerService(c)
	c.CallAnalysis = newCallAnalysisService(c)
	return c, nil
}

// Error is returned for every non-2xx response and in-stream failures.
type Error struct {
	Status  int    // HTTP status code
	Code    string // body.code, may be empty
	Field   string // body.field, may be empty
	Message string // body.message, else "HTTP <status>"
	Body    []byte // raw response body
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("sixtydb: %s (status %d, code %s)", e.Message, e.Status, e.Code)
	}
	return fmt.Sprintf("sixtydb: %s (status %d)", e.Message, e.Status)
}

func newHTTPError(status int, body []byte) *Error {
	e := &Error{Status: status, Message: fmt.Sprintf("HTTP %d", status), Body: body}
	var m map[string]any
	if json.Unmarshal(body, &m) == nil {
		if s, ok := m["message"].(string); ok && s != "" {
			e.Message = s
		}
		if s, ok := m["code"].(string); ok {
			e.Code = s
		}
		if s, ok := m["field"].(string); ok {
			e.Field = s
		}
	}
	return e
}

// PathEscape URL-encodes a path parameter.
func pathEscape(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), "+", "%2B")
}

// send performs a request. The caller must call cancel when done with resp.Body.
func (c *Client) send(ctx context.Context, method, path string, body io.Reader, contentType string, timeout time.Duration) (*http.Response, context.CancelFunc, error) {
	return c.sendHeaders(ctx, method, path, body, contentType, nil, timeout)
}

// sendHeaders is send with extra request headers; headers with an empty value are not sent.
func (c *Client) sendHeaders(ctx context.Context, method, path string, body io.Reader, contentType string,
	headers map[string]string, timeout time.Duration) (*http.Response, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "60db-go/0.1.0")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		return nil, nil, newHTTPError(resp.StatusCode, b)
	}
	return resp, cancel, nil
}

// raw performs a request and returns the response body and content type.
func (c *Client) raw(ctx context.Context, method, path string, payload any, timeout time.Duration) ([]byte, string, error) {
	return c.rawHeaders(ctx, method, path, payload, nil, timeout)
}

// rawHeaders is raw with extra request headers (e.g. Idempotency-Key).
func (c *Client) rawHeaders(ctx context.Context, method, path string, payload any, headers map[string]string, timeout time.Duration) ([]byte, string, error) {
	var body io.Reader
	ct := ""
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, "", err
		}
		body, ct = bytes.NewReader(b), "application/json"
	}
	return c.rawBodyHeaders(ctx, method, path, body, ct, headers, timeout)
}

func (c *Client) rawBody(ctx context.Context, method, path string, body io.Reader, ct string, timeout time.Duration) ([]byte, string, error) {
	return c.rawBodyHeaders(ctx, method, path, body, ct, nil, timeout)
}

func (c *Client) rawBodyHeaders(ctx context.Context, method, path string, body io.Reader, ct string,
	headers map[string]string, timeout time.Duration) ([]byte, string, error) {
	resp, cancel, err := c.sendHeaders(ctx, method, path, body, ct, headers, timeout)
	if err != nil {
		return nil, "", err
	}
	defer cancel()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return b, resp.Header.Get("Content-Type"), err
}

// parseJSON decodes a body into a map; non-object JSON is wrapped as {"data": v}.
func parseJSON(b []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(b)) == 0 {
		return map[string]any{}, nil
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("sixtydb: invalid JSON response: %w", err)
	}
	if m, ok := v.(map[string]any); ok {
		return m, nil
	}
	return map[string]any{"data": v}, nil
}

// do sends an optional JSON payload and returns the parsed JSON body.
func (c *Client) do(ctx context.Context, method, path string, payload any, timeout time.Duration) (map[string]any, error) {
	b, _, err := c.raw(ctx, method, path, payload, timeout)
	if err != nil {
		return nil, err
	}
	return parseJSON(b)
}

// doDataWith sends a request with extra headers and returns the body's `data` field.
func (c *Client) doDataWith(ctx context.Context, method, path string, payload any, headers map[string]string, timeout time.Duration) (any, error) {
	b, _, err := c.rawHeaders(ctx, method, path, payload, headers, timeout)
	if err != nil {
		return nil, err
	}
	m, err := parseJSON(b)
	if err != nil {
		return nil, err
	}
	return m["data"], nil
}

// idempotencyHeader sends key as Idempotency-Key; an empty key sends no header.
func idempotencyHeader(key string) map[string]string {
	return map[string]string{"Idempotency-Key": key}
}

// Page is a paged list: the rows (data) and the server's pagination (total, limit, offset).
type Page struct {
	Data       []any          `json:"data"`
	Pagination map[string]any `json:"pagination"`
}

// doPage GETs a paged-list envelope {data: [...], pagination: {...}} and keeps both.
func (c *Client) doPage(ctx context.Context, path string, timeout time.Duration) (*Page, error) {
	m, err := c.do(ctx, "GET", path, nil, timeout)
	if err != nil {
		return nil, err
	}
	data, _ := m["data"].([]any)
	if data == nil {
		data = []any{}
	}
	pg, _ := m["pagination"].(map[string]any)
	return &Page{Data: data, Pagination: pg}, nil
}

// doText GETs a text body (e.g. text/csv) as a UTF-8 string. Errors are the usual JSON envelope.
func (c *Client) doText(ctx context.Context, path string, timeout time.Duration) (string, error) {
	b, _, err := c.raw(ctx, "GET", path, nil, timeout)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// doBody is like do but with a prepared body (multipart).
func (c *Client) doBody(ctx context.Context, method, path string, body io.Reader, ct string, timeout time.Duration) (map[string]any, error) {
	b, _, err := c.rawBody(ctx, method, path, body, ct, timeout)
	if err != nil {
		return nil, err
	}
	return parseJSON(b)
}

// mergeParams builds a body map: base fields, then extra params (extra wins only for new keys).
func mergeParams(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range extra {
		out[k] = v
	}
	for k, v := range base {
		out[k] = v
	}
	return out
}

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
