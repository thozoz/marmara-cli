package api

import (
	"context"
	"encoding/json"

	"marmara-cli/internal/client"
)

// Common (Ortak) authenticated endpoints.
const (
	pathProfile = client.HostBYS + "/v2/Public/Mobil/Ortak/KullaniciBilgileri"
	pathCard    = client.HostBYS + "/v2/Public/Mobil/Ortak/GetKartBilgileri"
)

// Profile returns the authenticated user's profile (name, email, photo path, type).
func (a *API) Profile(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathProfile, map[string]any{})
}

// Card returns campus card status and numbers.
func (a *API) Card(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathCard, map[string]any{})
}
