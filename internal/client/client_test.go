package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBYSLoginError(t *testing.T) {
	broken := `{"Error":{"Message":"BYS kullan�c� ad� veya �ifre hatal� olabilir."}}`
	for _, host := range []string{"bys.marmara.edu.tr", "example.com"} {
		t.Run(host, func(t *testing.T) {
			c := &Client{http: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(broken))}, nil
			})}}
			err := c.Do(context.Background(), Request{Method: "POST", URL: "https://" + host + "/login"}, nil)
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected APIError, got %v", err)
			}
			want := broken
			if host == "bys.marmara.edu.tr" {
				want = `{"Error":{"Message":"BYS kullanıcı adı veya şifre hatalı olabilir."}}`
			}
			if apiErr.Body != want {
				t.Fatalf("body = %q, want %q", apiErr.Body, want)
			}
		})
	}
}

func TestTruncateUnicode(t *testing.T) {
	if got := truncate("şifre", 1); got != "ş..." {
		t.Fatalf("truncate = %q", got)
	}
	if got := truncate("Türkçe", 6); got != "Türkçe" {
		t.Fatalf("truncate = %q", got)
	}
}
