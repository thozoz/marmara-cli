package api

import (
	"context"
	"encoding/json"

	"marmara-cli/internal/client"
)

const pathAvesisSearch = client.HostAvesis + "/proxy/search"

// currentUniversityId for Marmara in the AVESİS Elasticsearch proxy.
const avesisUniversityID = 65

// AvesisSearch queries the AVESİS publication/researcher index.
func (a *API) AvesisSearch(ctx context.Context, query string) (json.RawMessage, error) {
	return a.publicPost(ctx, pathAvesisSearch, map[string]any{
		"query":                query,
		"page":                 1,
		"currentUniversityId":  avesisUniversityID,
	})
}
