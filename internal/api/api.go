// Package api wraps Marmara's student-facing and public endpoints. Every method
// returns raw JSON (json.RawMessage) because the upstream response shapes were
// reverse-engineered and may drift; callers (CLI, MCP) simply relay the JSON.
package api

import (
	"context"
	"encoding/json"

	"marmara-cli/internal/auth"
	"marmara-cli/internal/client"
)

// API bundles the HTTP client and auth manager.
type API struct {
	c    *client.Client
	auth *auth.Manager
}

// New wires an API from a client and auth manager.
func New(c *client.Client, m *auth.Manager) *API {
	return &API{c: c, auth: m}
}

// authPost performs an authenticated POST to a BYS Public endpoint, decoding the
// JSON body into raw. On a 401 it refreshes the token once and retries.
func (a *API) authPost(ctx context.Context, url string, body any) (json.RawMessage, error) {
	tok, err := a.auth.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	err = a.c.Do(ctx, client.Request{Method: "POST", URL: url, Bearer: tok, Body: body}, &raw)
	if client.IsUnauthorized(err) {
		// Force a refresh and retry once.
		tok, rerr := a.auth.AccessToken(ctx)
		if rerr != nil {
			return nil, rerr
		}
		raw = nil
		err = a.c.Do(ctx, client.Request{Method: "POST", URL: url, Bearer: tok, Body: body}, &raw)
	}
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// publicGet performs an unauthenticated GET, decoding JSON into raw.
func (a *API) publicGet(ctx context.Context, url string, query map[string]string) (json.RawMessage, error) {
	var raw json.RawMessage
	err := a.c.Do(ctx, client.Request{Method: "GET", URL: url, Query: query}, &raw)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// publicPost performs an unauthenticated POST, decoding JSON into raw.
func (a *API) publicPost(ctx context.Context, url string, body any) (json.RawMessage, error) {
	var raw json.RawMessage
	err := a.c.Do(ctx, client.Request{Method: "POST", URL: url, Body: body}, &raw)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
