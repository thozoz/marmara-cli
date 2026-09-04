// Package web serves a local browser dashboard over the same API layer the CLI
// uses. It binds to localhost only and reuses the shared token cache, so a
// single login (CLI or the web form) authorizes everything.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"time"

	"marmara-cli/internal/api"
	"marmara-cli/internal/auth"
	"marmara-cli/internal/client"
)

//go:embed static/*
var staticFS embed.FS

// Server wires HTTP routes to the API layer.
type Server struct {
	api  *api.API
	auth *auth.Manager
	mux  *http.ServeMux
}

// dataFn is an API method returning raw JSON for a given context.
type dataFn func(ctx context.Context) (json.RawMessage, error)

// New builds the server and its routes.
func New() (*Server, error) {
	c := client.New()
	m, err := auth.NewManager(c)
	if err != nil {
		return nil, err
	}
	s := &Server{api: api.New(c, m), auth: m, mux: http.NewServeMux()}
	s.routes()
	return s, nil
}

// ListenAndServe binds to 127.0.0.1:port only (never exposed off-host).
func (s *Server) ListenAndServe(port string) error {
	srv := &http.Server{
		Addr:              "127.0.0.1:" + port,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

func (s *Server) routes() {
	sub, _ := fs.Sub(staticFS, "static")
	s.mux.Handle("/", http.FileServer(http.FS(sub)))

	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/logout", s.handleLogout)
	s.mux.HandleFunc("/api/status", s.handleStatus)

	simple := map[string]dataFn{
		"profile":       s.api.Profile,
		"card":          s.api.Card,
		"grades":        s.api.Grades,
		"transcript":    s.api.Transcript,
		"schedule":      s.api.Schedule,
		"exams":         s.api.Exams,
		"features":      s.api.Features,
		"risk-report":   s.api.RiskReport,
		"cafeteria":     s.api.Cafeteria,
		"clubs":         s.api.Clubs,
		"club-events":   s.api.ClubEvents,
		"calendar":      s.api.Calendar,
		"campus-maps":   s.api.CampusMaps,
		"news":          s.api.News,
		"announcements": s.api.Announcements,
		"events":        s.api.Events,
	}
	for name, fn := range simple {
		fn := fn
		s.mux.HandleFunc("/api/"+name, func(w http.ResponseWriter, r *http.Request) {
			s.serveData(w, r, fn)
		})
	}

	s.mux.HandleFunc("/api/grade-detail", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("ders_id")
		if id == "" {
			writeErr(w, http.StatusBadRequest, "ders_id required")
			return
		}
		s.serveData(w, r, func(ctx context.Context) (json.RawMessage, error) { return s.api.GradeDetail(ctx, id) })
	})
}

func (s *Server) serveData(w http.ResponseWriter, r *http.Request, fn dataFn) {
	raw, err := fn(r.Context())
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, auth.ErrNotLoggedIn) {
			status = http.StatusUnauthorized
		}
		writeErr(w, status, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(raw)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := s.auth.Login(r.Context(), body.Username, body.Password); err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	_ = s.auth.Logout(r.Context())
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AccessToken(r.Context())
	writeJSON(w, map[string]bool{"loggedIn": err == nil})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
