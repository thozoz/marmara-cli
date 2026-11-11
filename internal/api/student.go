package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

// Grades returns the student's grade list for the active term.
// Like the schedule, NotListesiGetir returns an empty list without a term.
func (a *API) Grades(ctx context.Context) (json.RawMessage, error) {
	return a.activeTermPost(ctx, pathGrades)
}

// ParseDersID converts user-supplied input (CLI flag, web query, MCP argument)
// into an OgrenciDersId. The upstream model binder expects a JSON integer, so
// the value must be numeric; empty and non-numeric input is rejected here with
// a message that says where to get a valid id.
func ParseDersID(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("ders id is required (OgrenciDersId from `marmara grades`)")
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ders id %q: must be numeric (OgrenciDersId from `marmara grades`)", s)
	}
	if id <= 0 {
		return 0, fmt.Errorf("invalid ders id %d: must be positive", id)
	}
	return id, nil
}

// GradeDetail returns the component breakdown for a single course enrolment.
// dersID is the OgrenciDersId from the grade list and is sent as a JSON integer
// because the upstream ASP.NET model binder rejects a quoted string.
func (a *API) GradeDetail(ctx context.Context, dersID int64) (json.RawMessage, error) {
	if dersID <= 0 {
		return nil, fmt.Errorf("invalid ders id %d: must be positive", dersID)
	}
	raw, err := a.authPost(ctx, pathGradeDetl, map[string]any{"OgrenciDersId": dersID})
	if err != nil {
		return nil, explainGradeDetailError(dersID, err)
	}
	return raw, nil
}

// explainGradeDetailError turns the two known upstream failures into actionable
// messages. NotDetayGetir throws an unhandled exception (HTTP 500) upstream; it
// was observed live for both graded past-term and ungraded active-term ids, so
// the cause is on the BYS side and not something the caller can pick around.
func explainGradeDetailError(dersID int64, err error) error {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch {
	case apiErr.Status == 500:
		return fmt.Errorf(
			"grade detail unavailable for OgrenciDersId %d: BYS returned HTTP 500. "+
				"this is a backend error in NotDetayGetir, seen for both ungraded "+
				"active-term and archived courses; retry later: %w",
			dersID, err)
	case apiErr.Status == 400 && strings.Contains(apiErr.Body, "Bu Öğrenci İstenilen Dersi Almamaktadır"):
		return fmt.Errorf("grade detail unavailable for OgrenciDersId %d: the course enrolment does not belong to your active account/term: %w", dersID, err)
	}
	return err
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
	return a.activeTermPost(ctx, pathSchedule)
}

// activeTermPost posts the user's active term (from Profile) to a term-scoped
// endpoint. Schedule, grades and exams all return an empty list without it.
func (a *API) activeTermPost(ctx context.Context, url string) (json.RawMessage, error) {
	profRaw, err := a.Profile(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch profile for active term: %w", err)
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
	return a.authPost(ctx, url, map[string]any{"OgretimYili": prof.OgretimYili, "OgretimDonemi": prof.OgretimDonemi})
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

// Exams returns midterm/final/resit exam schedules for the active term.
func (a *API) Exams(ctx context.Context) (json.RawMessage, error) {
	return a.activeTermPost(ctx, pathExams)
}

// Exam filters for AllExams. BYS marks each exam with sinavCondition
// "passed" once it is over; anything else (e.g. "future") is still ahead.
const (
	ExamsAll    = ""
	ExamsPast   = "past"
	ExamsFuture = "future"
)

// AllExams returns the exams of every term the student enrolled in, taken from
// the transcript, in the same {"OgrenciDersSinavListesi": [...]} shape as
// Exams. Each exam is tagged with its OgretimYili/OgretimDonemi. filter is one
// of ExamsAll, ExamsPast or ExamsFuture.
func (a *API) AllExams(ctx context.Context, filter string) (json.RawMessage, error) {
	if filter != ExamsAll && filter != ExamsPast && filter != ExamsFuture {
		return nil, fmt.Errorf("invalid exam filter %q (want past or future)", filter)
	}
	trRaw, err := a.Transcript(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch transcript for terms: %w", err)
	}
	var tr struct {
		Transcript struct {
			Transcript struct {
				Semesters []struct {
					Yil            int  `json:"Yil"`
					DersAlinmismi  bool `json:"DersAlinmismi"`
					TanscriptDonem *struct {
						Kodid int `json:"Kodid"`
					} `json:"TanscriptDonem"`
				} `json:"Semesters"`
			} `json:"Transcript"`
		} `json:"Transcript"`
	}
	if err := json.Unmarshal(trRaw, &tr); err != nil {
		return nil, fmt.Errorf("decode transcript terms: %w", err)
	}

	all := []map[string]any{}
	for _, s := range tr.Transcript.Transcript.Semesters {
		if !s.DersAlinmismi || s.TanscriptDonem == nil || s.Yil <= 0 || s.TanscriptDonem.Kodid <= 0 {
			continue
		}
		donem := s.TanscriptDonem.Kodid
		raw, err := a.authPost(ctx, pathExams, map[string]any{"OgretimYili": s.Yil, "OgretimDonemi": donem})
		if err != nil {
			return nil, fmt.Errorf("exams for %d/%d: %w", s.Yil, donem, err)
		}
		var body struct {
			List []map[string]any `json:"OgrenciDersSinavListesi"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("decode exams for %d/%d: %w", s.Yil, donem, err)
		}
		for _, e := range body.List {
			past := e["sinavCondition"] == "passed"
			if (filter == ExamsPast && !past) || (filter == ExamsFuture && past) {
				continue
			}
			e["OgretimYili"], e["OgretimDonemi"] = s.Yil, donem
			all = append(all, e)
		}
	}
	return json.Marshal(map[string]any{"OgrenciDersSinavListesi": all})
}
