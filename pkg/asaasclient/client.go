// Package asaasclient is a small, resource-agnostic HTTP client for the
// Asaas API v3. Every endpoint is reachable through Do (and its Get/Post/
// Put/Delete shortcuts); responses are returned as raw JSON so callers can
// pass them through untouched.
package asaasclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultTimeout bounds a single HTTP round trip.
	DefaultTimeout = 60 * time.Second
	// MaxPageSize is the largest "limit" accepted by Asaas list endpoints.
	MaxPageSize = 100

	defaultUserAgent = "asaas-cli"
	maxResponseBytes = 32 << 20
)

// Client talks to one Asaas account in one environment.
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	userAgent  string
}

// Option customizes a Client.
type Option func(*Client) error

// WithBaseURL overrides the environment base URL (proxies, mocks, tests).
func WithBaseURL(raw string) Option {
	return func(c *Client) error {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("asaasclient: invalid base URL %q", raw)
		}
		c.baseURL = u
		return nil
	}
}

// WithHTTPClient replaces the underlying *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = h
		return nil
	}
}

// WithUserAgent sets the User-Agent header (required by Asaas for newer accounts).
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		c.userAgent = ua
		return nil
	}
}

// New builds a Client for the given key and environment.
func New(apiKey string, env Environment, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("asaasclient: API key is required")
	}
	if !env.Valid() {
		return nil, fmt.Errorf("asaasclient: invalid environment %q", env)
	}
	base, err := url.Parse(env.BaseURL())
	if err != nil {
		return nil, err
	}
	c := &Client{
		baseURL:    base,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: DefaultTimeout},
		userAgent:  defaultUserAgent,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// BaseURL returns the base URL requests are sent to.
func (c *Client) BaseURL() string { return c.baseURL.String() }

// Response is a successful (2xx) API response.
type Response struct {
	StatusCode int
	Header     http.Header
	// Body is the raw JSON body; nil when the API returned no content.
	Body json.RawMessage
}

// Do sends a request to path (relative to the /v3 base, e.g. "/customers").
// body may be nil, []byte / json.RawMessage (sent verbatim) or any value
// that encoding/json can marshal.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any) (*Response, error) {
	u, err := c.resolve(path, query)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		payload, err := encodeBody(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	return c.send(ctx, method, u, reader, "application/json")
}

// File is one file part of a multipart/form-data request.
type File struct {
	Field    string // form field name, e.g. "certificateFile"
	Filename string
	Content  io.Reader
}

// PostMultipart sends a multipart/form-data POST, used by the few Asaas
// endpoints that accept file uploads (e.g. /fiscalInfo).
func (c *Client) PostMultipart(ctx context.Context, path string, fields map[string]string, files []File) (*Response, error) {
	u, err := c.resolve(path, nil)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := mw.WriteField(k, fields[k]); err != nil {
			return nil, err
		}
	}
	for _, f := range files {
		part, err := mw.CreateFormFile(f.Field, f.Filename)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(part, f.Content); err != nil {
			return nil, fmt.Errorf("asaasclient: reading %s: %w", f.Filename, err)
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return c.send(ctx, http.MethodPost, u, &buf, mw.FormDataContentType())
}

func (c *Client) send(ctx context.Context, method string, u *url.URL, body io.Reader, contentType string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("access_token", c.apiKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Method: method, URL: u.String(), Err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &NetworkError{Method: method, URL: u.String(), Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, newAPIError(method, u.Path, resp.StatusCode, raw)
	}

	raw = bytes.TrimSpace(raw)
	out := &Response{StatusCode: resp.StatusCode, Header: resp.Header}
	if len(raw) == 0 {
		return out, nil
	}
	if !json.Valid(raw) {
		return nil, &ResponseError{StatusCode: resp.StatusCode, Message: "response body is not valid JSON", Body: truncate(string(raw), 2048)}
	}
	out.Body = raw
	return out, nil
}

// Get issues a GET request.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*Response, error) {
	return c.Do(ctx, http.MethodGet, path, query, nil)
}

// Post issues a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body any) (*Response, error) {
	return c.Do(ctx, http.MethodPost, path, nil, body)
}

// Put issues a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body any) (*Response, error) {
	return c.Do(ctx, http.MethodPut, path, nil, body)
}

// Delete issues a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

// Page is the envelope Asaas uses for every list endpoint.
type Page struct {
	Object     string            `json:"object"`
	HasMore    bool              `json:"hasMore"`
	TotalCount int               `json:"totalCount"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
	Data       []json.RawMessage `json:"data"`
}

// ListAll walks every page of a list endpoint (starting at the "offset" in
// query, if any) and returns a single Page holding all items.
func (c *Client) ListAll(ctx context.Context, path string, query url.Values) (*Page, error) {
	q := url.Values{}
	for k, v := range query {
		q[k] = append([]string(nil), v...)
	}
	start, _ := strconv.Atoi(q.Get("offset"))
	q.Set("limit", strconv.Itoa(MaxPageSize))

	all := &Page{Object: "list", Offset: start, Data: []json.RawMessage{}}
	for offset := start; ; {
		q.Set("offset", strconv.Itoa(offset))
		resp, err := c.Get(ctx, path, q)
		if err != nil {
			return nil, err
		}
		var page Page
		if err := json.Unmarshal(resp.Body, &page); err != nil {
			return nil, &ResponseError{StatusCode: resp.StatusCode, Message: "response is not a list page: " + err.Error(), Body: truncate(string(resp.Body), 2048)}
		}
		all.Data = append(all.Data, page.Data...)
		all.TotalCount = page.TotalCount
		if !page.HasMore || len(page.Data) == 0 {
			break
		}
		offset += len(page.Data)
	}
	all.Limit = len(all.Data)
	return all, nil
}

// resolve joins path onto the base URL. Only relative API paths are
// accepted so the access token is never sent to another host.
func (c *Client) resolve(path string, query url.Values) (*url.URL, error) {
	p := strings.TrimLeft(strings.TrimSpace(path), "/")
	p = strings.TrimPrefix(p, "v3/")
	ref, err := url.Parse(p)
	if err != nil {
		return nil, fmt.Errorf("asaasclient: invalid path %q: %w", path, err)
	}
	if ref.Scheme != "" || ref.Host != "" {
		return nil, fmt.Errorf("asaasclient: path %q must be relative to the API base (e.g. /customers)", path)
	}
	for _, seg := range strings.Split(ref.Path, "/") {
		if seg == ".." {
			return nil, fmt.Errorf("asaasclient: path %q must not contain '..'", path)
		}
	}

	base := *c.baseURL
	base.Path = strings.TrimRight(base.Path, "/") + "/"
	base.RawPath = ""
	u := base.ResolveReference(ref)

	q := u.Query()
	for k, vs := range query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u, nil
}

func encodeBody(body any) ([]byte, error) {
	switch b := body.(type) {
	case json.RawMessage:
		return b, nil
	case []byte:
		return b, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("asaasclient: encoding request body: %w", err)
	}
	return payload, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
