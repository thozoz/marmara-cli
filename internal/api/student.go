package api

import (
	"context"
	"encoding/json"

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
	pathPrep       = client.HostBYS + "/v2/Public/Mobil/Ogrenci/HazirlikBilgileri"
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

// Schedule returns the weekly lecture timetable.
func (a *API) Schedule(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathSchedule, map[string]any{})
}

// Exams returns midterm/final/resit exam schedules.
func (a *API) Exams(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathExams, map[string]any{})
}

// Prep returns prep/language school status.
func (a *API) Prep(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathPrep, map[string]any{})
}
