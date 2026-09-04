package api

import (
	"context"
	"encoding/json"

	"marmara-cli/internal/client"
)

// Unauthenticated endpoints under /v2/Public/Mobil/*.
const (
	pathFeatures = client.HostBYS + "/v2/Public/Mobil/GetFeatures"
	pathRisk     = client.HostBYS + "/v2/Public/Mobil/GetRecentRiskReport"
)

// Features returns app feature toggles.
func (a *API) Features(ctx context.Context) (json.RawMessage, error) {
	return a.publicPost(ctx, pathFeatures, map[string]any{})
}

// RiskReport returns the health/risk report payload.
func (a *API) RiskReport(ctx context.Context) (json.RawMessage, error) {
	return a.publicPost(ctx, pathRisk, map[string]any{})
}
