// Package client provides a small HTTP helper for talking to Marmara's
// backends. It centralises base URLs, JSON encoding/decoding, bearer-token
// injection and a consistent error type so the api package stays thin.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Base hosts. Only student-facing / public endpoints are used.
const (
	HostBYS    = "https://bys.marmara.edu.tr"
	HostSKS    = "https://sks.marmara.edu.tr"
	HostESKS   = "https://esks.marmara.edu.tr"
	HostTakvim = "https://takvim.marmara.edu.tr"
	HostWWW    = "https://www.marmara.edu.tr"
	HostDestek = "https://destek.marmara.edu.tr"
	HostAvesis = "https://avesis.marmara.edu.tr"
)

// defaultUserAgent is honest about being an unofficial client. Override with
// MARMARA_USER_AGENT if the backend ever rejects it.
const defaultUserAgent = "marmara-cli/0.1 (+unofficial)"

// Client is a thin wrapper over http.Client.
type Client struct {
	http *http.Client
	ua   string
}

// New returns a Client with sensible timeouts.
func New() *Client {
	ua := defaultUserAgent
	if v := os.Getenv("MARMARA_USER_AGENT"); v != "" {
		ua = v
	}
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
		ua:   ua,
	}
}

// APIError is returned for any non-2xx response.
type APIError struct {
	Status int
	Body   string
	URL    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("http %d from %s: %s", e.Status, e.URL, truncate(e.Body, 300))
}

// IsUnauthorized reports whether the error is a 401, used to trigger a token
// refresh-and-retry in the auth layer.
func IsUnauthorized(err error) bool {
	if ae, ok := err.(*APIError); ok {
		return ae.Status == http.StatusUnauthorized
	}
	return false
}

// Request describes a single call.
type Request struct {
	Method  string
	URL     string
	Bearer  string            // optional access token
	Body    any               // optional; JSON-encoded if non-nil
	Query   map[string]string // optional query params
	Headers map[string]string // optional extra headers
}

// Do executes req and decodes a 2xx JSON response into out (out may be nil to
// discard the body). Non-2xx responses become *APIError.
func (c *Client) Do(ctx context.Context, req Request, out any) error {
	var bodyReader io.Reader
	if req.Body != nil {
		buf, err := json.Marshal(req.Body)
		if err != nil {
			return fmt.Errorf("encode body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	if len(req.Query) > 0 {
		q := httpReq.URL.Query()
		for k, v := range req.Query {
			q.Set(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	httpReq.Header.Set("User-Agent", c.ua)
	httpReq.Header.Set("Accept", "application/json")
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if req.Bearer != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Bearer)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // cap at 8 MiB

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: string(raw), URL: httpReq.URL.String()}
	}

	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response from %s: %w", httpReq.URL.String(), err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
