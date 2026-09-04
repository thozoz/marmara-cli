// Package auth handles login against the native mobile Bearer API, local token
// caching and transparent refresh. Only tokens are persisted, never the raw
// password.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"marmara-cli/internal/client"
)

const (
	loginPath   = client.HostBYS + "/v2/Public/Authenticate/LoginMobil"
	refreshPath = client.HostBYS + "/v2/Public/Authenticate/MobilRefresh"
	revokePath  = client.HostBYS + "/v2/Public/Authenticate/RevokeToken"
)

// Token is the cached credential set.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// loginResponse mirrors the LoginMobil / MobilRefresh JSON shape. Field names
// on the real backend are PascalCase; tags cover common casings defensively.
type loginResponse struct {
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
	ExpiresIn    int    `json:"ExpiresIn"` // seconds
}

// Manager owns the token lifecycle.
type Manager struct {
	c    *client.Client
	path string
	tok  *Token
}

// NewManager returns a Manager backed by the on-disk cache at ~/.marmara/token.json.
func NewManager(c *client.Client) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	return &Manager{c: c, path: filepath.Join(home, ".marmara", "token.json")}, nil
}

// ErrNotLoggedIn is returned when no cached token is available.
var ErrNotLoggedIn = errors.New("not logged in: run `marmara login` first")

// Login authenticates with username/password and persists the resulting tokens.
func (m *Manager) Login(ctx context.Context, username, password string) error {
	var resp loginResponse
	err := m.c.Do(ctx, client.Request{
		Method: "POST",
		URL:    loginPath,
		Body:   map[string]string{"Username": username, "Password": password},
	}, &resp)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	if resp.AccessToken == "" {
		return errors.New("login failed: no access token returned (check credentials)")
	}
	return m.store(resp)
}

// Logout revokes the active token (best effort) and removes the local cache.
func (m *Manager) Logout(ctx context.Context) error {
	if tok, err := m.load(); err == nil && tok.AccessToken != "" {
		_ = m.c.Do(ctx, client.Request{
			Method: "POST",
			URL:    revokePath,
			Bearer: tok.AccessToken,
			Body:   map[string]string{"Token": tok.RefreshToken},
		}, nil)
	}
	if err := os.Remove(m.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove token cache: %w", err)
	}
	return nil
}

// AccessToken returns a currently-valid access token, refreshing if it is
// expired or within 60s of expiry.
func (m *Manager) AccessToken(ctx context.Context) (string, error) {
	tok, err := m.load()
	if err != nil {
		return "", err
	}
	if time.Now().Add(60 * time.Second).Before(tok.ExpiresAt) {
		return tok.AccessToken, nil
	}
	if err := m.refresh(ctx, tok); err != nil {
		return "", err
	}
	return m.tok.AccessToken, nil
}

// Refresh forces a token refresh using the stored refresh token.
func (m *Manager) refresh(ctx context.Context, tok *Token) error {
	if tok.RefreshToken == "" {
		return ErrNotLoggedIn
	}
	var resp loginResponse
	err := m.c.Do(ctx, client.Request{
		Method: "POST",
		URL:    refreshPath,
		Body:   map[string]string{"RefreshToken": tok.RefreshToken},
	}, &resp)
	if err != nil {
		return fmt.Errorf("token refresh failed (login again): %w", err)
	}
	if resp.AccessToken == "" {
		return errors.New("token refresh failed: no access token returned")
	}
	return m.store(resp)
}

func (m *Manager) store(resp loginResponse) error {
	expiresIn := resp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600 // sane default if backend omits it
	}
	tok := &Token{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	buf, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(m.path, buf, 0o600); err != nil {
		return fmt.Errorf("write token cache: %w", err)
	}
	m.tok = tok
	return nil
}

func (m *Manager) load() (*Token, error) {
	if m.tok != nil {
		return m.tok, nil
	}
	buf, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotLoggedIn
		}
		return nil, fmt.Errorf("read token cache: %w", err)
	}
	var tok Token
	if err := json.Unmarshal(buf, &tok); err != nil {
		return nil, fmt.Errorf("parse token cache: %w", err)
	}
	m.tok = &tok
	return &tok, nil
}
