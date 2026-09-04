// Package cli defines the cobra command tree. Read commands print raw JSON to
// stdout by default so an AI agent or script can parse them; --pretty
// re-indents the JSON for humans.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"marmara-cli/internal/api"
	"marmara-cli/internal/auth"
	"marmara-cli/internal/client"

	"github.com/spf13/cobra"
)

// json0 is a short alias for json.RawMessage used across command signatures.
type json0 = json.RawMessage

// renderer turns a raw JSON response into a human-readable table. Returning an
// error makes output fall back to JSON.
type renderer func(json.RawMessage) (string, error)

var (
	prettyFlag bool
	tableFlag  bool
)

// deps builds the API + auth manager shared by all commands.
func deps() (*api.API, *auth.Manager, error) {
	c := client.New()
	m, err := auth.NewManager(c)
	if err != nil {
		return nil, nil, err
	}
	return api.New(c, m), m, nil
}

// emit writes JSON to stdout, optionally re-indented.
func emit(raw json.RawMessage) error {
	if !prettyFlag {
		_, err := fmt.Fprintln(os.Stdout, string(raw))
		return err
	}
	var buf any
	if err := json.Unmarshal(raw, &buf); err != nil {
		// Not valid JSON object/array — print as-is.
		_, err := fmt.Fprintln(os.Stdout, string(raw))
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(buf)
}

// output prints raw as a table when --table is set and a renderer is available
// (falling back to JSON if the renderer errors), otherwise as JSON.
func output(raw json.RawMessage, r renderer) error {
	if tableFlag && r != nil {
		if s, err := r(raw); err == nil {
			_, err := fmt.Fprintln(os.Stdout, s)
			return err
		}
	}
	return emit(raw)
}

// dataCmd is a helper for the common "call one API method, print output" pattern.
// r may be nil (no table renderer; always JSON).
func dataCmd(use, short string, fn func(ctx context.Context, a *api.API) (json.RawMessage, error), r renderer) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			a, _, err := deps()
			if err != nil {
				return err
			}
			raw, err := fn(cmd.Context(), a)
			if err != nil {
				return err
			}
			return output(raw, r)
		},
	}
}

// NewRoot assembles the full command tree.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "marmara",
		Short:         "Unofficial Marmara Üniversitesi CLI (personal use)",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&prettyFlag, "pretty", false, "pretty-print JSON output")
	root.PersistentFlags().BoolVar(&tableFlag, "table", false, "human-readable table output (falls back to JSON if unsupported)")

	root.AddCommand(
		loginCmd(),
		logoutCmd(),
		// Authenticated: common
		dataCmd("profile", "Show your profile", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Profile(ctx) }, renderProfile),
		dataCmd("card", "Campus card balance and gate logs", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Card(ctx) }, renderCard),
		// Authenticated: student
		gradesCmd(),
		dataCmd("transcript", "Full academic transcript", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Transcript(ctx) }, renderTranscript),
		dataCmd("schedule", "Weekly lecture timetable", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Schedule(ctx) }, nil),
		dataCmd("exams", "Exam schedule (midterm/final/resit)", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Exams(ctx) }, nil),
		// Public BYS
		dataCmd("features", "App feature toggles", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Features(ctx) }, nil),
		dataCmd("risk-report", "Health/risk report", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.RiskReport(ctx) }, nil),
		// Public external hosts
		dataCmd("cafeteria", "Daily cafeteria menu", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Cafeteria(ctx) }, renderCafeteria),
		dataCmd("clubs", "Active student clubs", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Clubs(ctx) }, nil),
		dataCmd("club-events", "Upcoming club events", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.ClubEvents(ctx) }, nil),
		dataCmd("calendar", "Academic calendar and exam periods", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Calendar(ctx) }, renderTitleList),
		campusMapsCmd(),
		dataCmd("news", "Recent university news", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.News(ctx) }, renderTitleList),
		dataCmd("announcements", "Official announcements", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Announcements(ctx) }, renderTitleList),
		dataCmd("events", "University events", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Events(ctx) }, renderTitleList),
		// MCP server mode
		mcpCmd(),
	)
	return root
}

// Execute runs the root command.
func Execute() {
	if err := NewRoot().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
