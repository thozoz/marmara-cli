package api

import (
	"context"
	"encoding/json"

	"marmara-cli/internal/client"
)

// Support ticket system (Destek).
const (
	pathTicketAttrs  = client.HostDestek + "/Issue/GetApplicationAttributes"
	pathTicketCreate = client.HostDestek + "/Issue/CreateIssueMobil"
)

// TicketAttributes returns the ticket form metadata (departments, custom fields).
func (a *API) TicketAttributes(ctx context.Context) (json.RawMessage, error) {
	return a.publicGet(ctx, pathTicketAttrs, nil)
}

// CreateTicket files a support ticket. This is a real side effect on Marmara's
// production system; callers must gate it behind explicit user confirmation.
func (a *API) CreateTicket(ctx context.Context, subject, body string) (json.RawMessage, error) {
	return a.publicPost(ctx, pathTicketCreate, map[string]any{
		"Subject": subject,
		"Konu":    subject,
		"Body":    body,
		"Mesaj":   body,
	})
}
