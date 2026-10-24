package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"marmara-cli/internal/client"
)

// Public JSON on separate hosts — no auth required.
const (
	pathCafeteria = client.HostSKS + "/yemek/json"
	pathClubs     = client.HostESKS + "/kulup/kuluplist.json"
	pathClubEvs   = client.HostESKS + "/kulup/kulupeglist.json"
	pathCalendar  = client.HostTakvim + "/service/events"
	pathMaps      = client.HostWWW + "/ajax/campusmaps"
	pathMapTypes  = client.HostWWW + "/ajax/campusmaptypes"
	pathAjaxSvc   = client.HostWWW + "/ajax/service"
)

// pathCafeteriaBYS is the mobile app's cafeteria endpoint on BYS. It serves the
// same menu as SKS but wrapped in an object ({"Yemekler": [...]}), and it is
// reachable when the SKS host is down. Must be a GET; POST returns 405.
const pathCafeteriaBYS = client.HostBYS + "/v2/Public/Mobil/Yemekhane"

// Cafeteria returns the daily cafeteria menu.
//
// SKS is the primary source. If it times out, fails DNS, returns HTTP 5xx or
// sends an unusable body, the BYS mobile endpoint is used as a fallback. Both
// shapes are normalised to the flat array of day objects that callers (CLI
// --table, web dashboard, MCP) expect.
func (a *API) Cafeteria(ctx context.Context) (json.RawMessage, error) {
	raw, sksErr := a.publicGet(ctx, pathCafeteria, nil)
	if sksErr == nil {
		if menu, err := NormalizeCafeteria(raw); err == nil {
			return menu, nil
		} else {
			sksErr = fmt.Errorf("unusable SKS response: %w", err)
		}
	}

	bysRaw, bysErr := a.publicGet(ctx, pathCafeteriaBYS, nil)
	if bysErr != nil {
		return nil, fmt.Errorf("cafeteria unavailable: SKS failed (%v); BYS fallback also failed: %w", sksErr, bysErr)
	}
	menu, err := NormalizeCafeteria(bysRaw)
	if err != nil {
		return nil, fmt.Errorf("cafeteria unavailable: SKS failed (%v); BYS fallback returned an unusable body: %w", sksErr, err)
	}
	return menu, nil
}

// NormalizeCafeteria turns either supported cafeteria payload into a plain JSON
// array of day menus.
//
// Supported shapes:
//   - SKS: a bare array of days ([{date, meals, ...}]).
//   - BYS: an object wrapping that same array under "Yemekler"
//     ({"Yemekler": [...], "Error": null, "CustomData": null}).
func NormalizeCafeteria(raw json.RawMessage) (json.RawMessage, error) {
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("decode menu: %w", err)
	}
	switch v := probe.(type) {
	case []any:
		if len(v) == 0 {
			return nil, errors.New("empty menu")
		}
		// Already the canonical shape: hand back the original bytes so field
		// order and number formatting match what the server sent.
		return raw, nil
	case map[string]any:
		if msg := upstreamErrorMessage(v); msg != "" {
			return nil, fmt.Errorf("upstream error: %s", msg)
		}
		days, ok := v["Yemekler"].([]any)
		if !ok || len(days) == 0 {
			return nil, errors.New("no Yemekler array in response")
		}
		out, err := json.Marshal(days)
		if err != nil {
			return nil, fmt.Errorf("normalize menu: %w", err)
		}
		return out, nil
	default:
		return nil, errors.New("unexpected menu payload (want an array or an object with Yemekler)")
	}
}

// upstreamErrorMessage extracts a human-readable message from BYS's JSON error
// envelope ({"Error":{"Message":"..."}}), returning "" when there is none.
func upstreamErrorMessage(m map[string]any) string {
	e, ok := m["Error"].(map[string]any)
	if !ok {
		return ""
	}
	if s, ok := e["Message"].(string); ok {
		return s
	}
	return ""
}

// Clubs returns the active student club list (ESKS).
func (a *API) Clubs(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathClubs, nil)
}

// ClubEvents returns upcoming club events (ESKS).
func (a *API) ClubEvents(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathClubEvs, nil)
}

// Calendar returns academic calendar dates and exam periods.
func (a *API) Calendar(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathCalendar, nil)
}

// CampusMaps returns campus GPS markers and buildings.
func (a *API) CampusMaps(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathMaps, nil)
}

// CampusMapTypes returns building category filters.
func (a *API) CampusMapTypes(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathMapTypes, nil)
}

// News returns recent university news.
func (a *API) News(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathAjaxSvc, map[string]string{"type": "haber"})
}

// Announcements returns official announcements.
func (a *API) Announcements(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathAjaxSvc, map[string]string{"type": "duyuru"})
}

// Events returns university events.
func (a *API) Events(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathAjaxSvc, map[string]string{"type": "etkinlik"})
}
