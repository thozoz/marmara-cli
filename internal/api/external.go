package api

import (
	"context"
	"encoding/json"

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

// Cafeteria returns the daily cafeteria menu (SKS).
func (a *API) Cafeteria(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathCafeteria, nil)
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

// Events returns university events (bonus; same ajax/service endpoint).
func (a *API) Events(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathAjaxSvc, map[string]string{"type": "etkinlik"})
}
