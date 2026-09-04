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

var prettyFlag bool

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

// dataCmd is a helper for the common "call one API method, print JSON" pattern.
func dataCmd(use, short string, fn func(ctx context.Context, a *api.API) (json.RawMessage, error)) *cobra.Command {
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
			return emit(raw)
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

	root.AddCommand(
		loginCmd(),
		logoutCmd(),
		// Authenticated: common
		dataCmd("profile", "Show your profile", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Profile(ctx) }),
		dataCmd("card", "Campus card balance and gate logs", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Card(ctx) }),
		// Authenticated: student
		gradesCmd(),
		dataCmd("transcript", "Full academic transcript", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Transcript(ctx) }),
		dataCmd("schedule", "Weekly lecture timetable", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Schedule(ctx) }),
		dataCmd("exams", "Exam schedule (midterm/final/resit)", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Exams(ctx) }),
		// Public BYS
		dataCmd("features", "App feature toggles", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Features(ctx) }),
		dataCmd("risk-report", "Health/risk report", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.RiskReport(ctx) }),
		directoryCmd(),
		// Public external hosts
		dataCmd("cafeteria", "Daily cafeteria menu", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Cafeteria(ctx) }),
		dataCmd("clubs", "Active student clubs", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Clubs(ctx) }),
		dataCmd("club-events", "Upcoming club events", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.ClubEvents(ctx) }),
		dataCmd("calendar", "Academic calendar and exam periods", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Calendar(ctx) }),
		campusMapsCmd(),
		dataCmd("news", "Recent university news", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.News(ctx) }),
		dataCmd("announcements", "Official announcements", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Announcements(ctx) }),
		dataCmd("events", "University events", func(ctx context.Context, a *api.API) (json.RawMessage, error) { return a.Events(ctx) }),
		// Support tickets
		ticketCmd(),
		// AVESİS
		avesisCmd(),
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
