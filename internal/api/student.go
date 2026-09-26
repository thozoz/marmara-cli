package api

import (
	"context"
	"encoding/json"
	"fmt"

	"marmara-cli/internal/client"
)

// Student endpoints under /v2/Public/Mobil/Ogrenci/*. The backend infers the
// student from the bearer token; these calls send an empty body and never take
// an arbitrary OgrenciId, by design (no cross-student lookups).

const (
	pathGrades     = client.HostBYS + "/v2/Public/Mobil/Ogrenci/NotListesiGetir"
	pathGradeDetl  = client.HostBYS + "/v2/Public/Mobil/Ogrenci/NotDetayGetir"
	pathTranscript = client.HostBYS + "/v2/Public/Mobil/Ogrenci/TranscriptGetir"
	pathSchedule   = client.HostBYS + "/v2/Public/Mobil/Ogrenci/DersProgramiGetir"
	pathExams      = client.HostBYS + "/v2/Public/Mobil/Ogrenci/SinavListesiGetir"
)

// Grades returns the student's grade list.
func (a *API) Grades(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathGrades, map[string]any{})
}

// GradeDetail returns the component breakdown for a single course enrolment.
// dersID is the OgrenciDersId from the grade list.
func (a *API) GradeDetail(ctx context.Context, dersID string) (json.RawMessage, error) {
	return a.authPost(ctx, pathGradeDetl, map[string]any{"OgrenciDersId": dersID})
}

// Transcript returns the full academic transcript.
func (a *API) Transcript(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathTranscript, map[string]any{})
}

// Schedule returns the weekly lecture timetable for the active term.
// The upstream BYS endpoint requires OgretimYili and OgretimDonemi; if omitted,
// it returns an empty list. This retrieves the current term from the user's
// Profile before requesting the schedule.
func (a *API) Schedule(ctx context.Context) (json.RawMessage, error) {
	profRaw, err := a.Profile(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch profile for schedule: %w", err)
	}

	var prof struct {
		OgretimYili   int `json:"OgretimYili"`
		OgretimDonemi int `json:"OgretimDonemi"`
	}
	if err := json.Unmarshal(profRaw, &prof); err != nil {
		return nil, fmt.Errorf("decode profile term: %w", err)
	}
	if prof.OgretimYili <= 0 || prof.OgretimDonemi <= 0 {
		return nil, fmt.Errorf("no active term found in profile (year %d, term %d)", prof.OgretimYili, prof.OgretimDonemi)
	}

	return a.ScheduleTerm(ctx, prof.OgretimYili, prof.OgretimDonemi)
}

// ScheduleTerm returns the weekly lecture timetable for a specific academic year and term.
func (a *API) ScheduleTerm(ctx context.Context, yil, donem int) (json.RawMessage, error) {
	if yil <= 0 || donem <= 0 {
		return nil, fmt.Errorf("invalid term parameters: OgretimYili (%d) and OgretimDonemi (%d) must be positive", yil, donem)
	}
	body := map[string]any{
		"OgretimYili":   yil,
		"OgretimDonemi": donem,
	}
	return a.authPost(ctx, pathSchedule, body)
}

// Exams returns midterm/final/resit exam schedules.
func (a *API) Exams(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathExams, map[string]any{})
}
