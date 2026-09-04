// Package mcpserver exposes the same Marmara operations as an MCP stdio server
// so an MCP-aware agent (Claude Code/Desktop, etc.) can call them directly.
package mcpserver

import (
	"context"
	"encoding/json"

	"marmara-cli/internal/api"
	"marmara-cli/internal/auth"
	"marmara-cli/internal/client"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Serve builds the tool set and blocks serving over stdio.
func Serve() error {
	c := client.New()
	m, err := auth.NewManager(c)
	if err != nil {
		return err
	}
	a := api.New(c, m)

	s := server.NewMCPServer("marmara", "0.1.0")
	register(s, a)
	return server.ServeStdio(s)
}

// simpleFn is an api method taking only a context.
type simpleFn func(ctx context.Context, a *api.API) (json.RawMessage, error)

// addSimple registers a no-argument tool.
func addSimple(s *server.MCPServer, a *api.API, name, desc string, fn simpleFn) {
	tool := mcp.NewTool(name, mcp.WithDescription(desc))
	s.AddTool(tool, func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := fn(ctx, a)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(raw)), nil
	})
}

func register(s *server.MCPServer, a *api.API) {
	// Authenticated — student's own data.
	addSimple(s, a, "profile", "Get your Marmara profile (name, email, type).", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Profile(ctx) })
	addSimple(s, a, "card", "Campus card balance and gate-pass logs.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Card(ctx) })
	addSimple(s, a, "grades", "Your grade list.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Grades(ctx) })
	addSimple(s, a, "transcript", "Full academic transcript.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Transcript(ctx) })
	addSimple(s, a, "schedule", "Weekly lecture timetable.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Schedule(ctx) })
	addSimple(s, a, "exams", "Exam schedule (midterm/final/resit).", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Exams(ctx) })

	// Grade detail (takes an id).
	gradeDetail := mcp.NewTool("grade_detail",
		mcp.WithDescription("Grade component breakdown for one course."),
		mcp.WithString("ders_id", mcp.Description("OgrenciDersId from the grades list."), mcp.Required()),
	)
	s.AddTool(gradeDetail, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("ders_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := a.GradeDetail(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(raw)), nil
	})

	// Public — no auth.
	addSimple(s, a, "features", "App feature toggles.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Features(ctx) })
	addSimple(s, a, "risk_report", "Health/risk report.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.RiskReport(ctx) })
	addSimple(s, a, "cafeteria", "Daily cafeteria menu.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Cafeteria(ctx) })
	addSimple(s, a, "clubs", "Active student clubs.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Clubs(ctx) })
	addSimple(s, a, "club_events", "Upcoming club events.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.ClubEvents(ctx) })
	addSimple(s, a, "calendar", "Academic calendar and exam periods.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Calendar(ctx) })
	addSimple(s, a, "campus_maps", "Campus GPS markers and buildings.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.CampusMaps(ctx) })
	addSimple(s, a, "campus_map_types", "Campus building categories.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.CampusMapTypes(ctx) })
	addSimple(s, a, "news", "Recent university news.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.News(ctx) })
	addSimple(s, a, "announcements", "Official announcements.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Announcements(ctx) })
	addSimple(s, a, "events", "University events.", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Events(ctx) })

	// Grade detail (takes an id) is registered above.
}
