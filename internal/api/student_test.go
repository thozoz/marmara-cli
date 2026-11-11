package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"marmara-cli/internal/auth"
	"marmara-cli/internal/client"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// stubAPI returns an *api.API whose HTTP transport is replaced by fn and whose
// auth manager reads a fake, non-expiring token from a temp HOME.
func stubAPI(t *testing.T, fn roundTripFunc) *API {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir reads this on Windows
	dir := filepath.Join(home, ".marmara")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	buf, err := json.Marshal(auth.Token{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "token.json"), buf, 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	c := client.NewWithHTTPClient(&http.Client{Transport: fn})
	m, err := auth.NewManager(c)
	if err != nil {
		t.Fatalf("auth manager: %v", err)
	}
	return New(c, m)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestParseDersID(t *testing.T) {
	good := map[string]int64{"42": 42, "  7  ": 7, "123456": 123456}
	for in, want := range good {
		got, err := ParseDersID(in)
		if err != nil {
			t.Fatalf("ParseDersID(%q) error: %v", in, err)
		}
		if got != want {
			t.Fatalf("ParseDersID(%q) = %d, want %d", in, got, want)
		}
	}
	for _, in := range []string{"", "   ", "abc", "12abc", "0", "-3", "1.5"} {
		if _, err := ParseDersID(in); err == nil {
			t.Fatalf("ParseDersID(%q) = nil error, want error", in)
		}
	}
}

// The upstream ASP.NET model binder needs an integer; sending a quoted string is
// the bug described in issue #9.
func TestGradeDetail_SendsIntegerOgrenciDersId(t *testing.T) {
	var sentBody string
	a := stubAPI(t, func(r *http.Request) (*http.Response, error) {
		buf, _ := io.ReadAll(r.Body)
		sentBody = string(buf)
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want Bearer test-token", got)
		}
		return jsonResponse(200, `{"SinavNotlariListesi":[]}`), nil
	})

	if _, err := a.GradeDetail(context.Background(), 123456); err != nil {
		t.Fatalf("GradeDetail: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(sentBody), &payload); err != nil {
		t.Fatalf("decode sent body %q: %v", sentBody, err)
	}
	id, ok := payload["OgrenciDersId"]
	if !ok {
		t.Fatalf("payload missing OgrenciDersId: %s", sentBody)
	}
	if _, isString := id.(string); isString {
		t.Fatalf("OgrenciDersId sent as a JSON string, want number: %s", sentBody)
	}
	if num, isNum := id.(float64); !isNum || num != 123456 {
		t.Fatalf("OgrenciDersId = %#v, want number 123456", id)
	}
}

func TestGradeDetail_RejectsNonPositiveID(t *testing.T) {
	a := &API{}
	for _, id := range []int64{0, -1} {
		if _, err := a.GradeDetail(context.Background(), id); err == nil {
			t.Fatalf("GradeDetail(%d) = nil error, want error", id)
		}
	}
}

// A 500 from NotDetayGetir is the past-semester case in issue #3; the error must
// explain that only active-term courses work, while staying unwrappable to the
// original APIError.
func TestGradeDetail_ExplainsHTTP500(t *testing.T) {
	a := stubAPI(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(500, `{"Error":{"Code":"Exception","Message":"An error occurred while processing your request."}}`), nil
	})
	_, err := a.GradeDetail(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"999", "active term", "HTTP 500"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not mention %q", msg, want)
		}
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Error("wrapped original APIError should stay unwrappable")
	}
}

func TestGradeDetail_ExplainsUnknownEnrolment(t *testing.T) {
	a := stubAPI(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(400, `{"Error":{"Message":"Bu Öğrenci İstenilen Dersi Almamaktadır."}}`), nil
	})
	_, err := a.GradeDetail(context.Background(), 5)
	if err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleTerm_Validation(t *testing.T) {
	a := &API{}

	testCases := []struct {
		name  string
		yil   int
		donem int
	}{
		{"zero year", 0, 100001},
		{"negative year", -1, 100001},
		{"zero term", 2026, 0},
		{"negative term", 2026, -1},
		{"both zero", 0, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := a.ScheduleTerm(context.Background(), tc.yil, tc.donem)
			if err == nil {
				t.Fatalf("expected error for yil=%d donem=%d, got nil", tc.yil, tc.donem)
			}
			if !strings.Contains(err.Error(), "must be positive") {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}
