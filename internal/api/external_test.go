package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const sampleDay = `{"date":"2026-10-19","meals":{"Mercimek Çorbası":110,"İzmir Köfte":340}}`

func TestNormalizeCafeteria_SKSArray(t *testing.T) {
	raw := json.RawMessage(`[` + sampleDay + `]`)
	out, err := NormalizeCafeteria(raw)
	if err != nil {
		t.Fatalf("NormalizeCafeteria: %v", err)
	}
	var days []map[string]any
	if err := json.Unmarshal(out, &days); err != nil {
		t.Fatalf("output is not an array: %v (%s)", err, out)
	}
	if len(days) != 1 || days[0]["date"] != "2026-10-19" {
		t.Fatalf("unexpected days: %s", out)
	}
}

// The BYS fallback wraps the same array under "Yemekler"; unwrapping it is what
// keeps --table and the dashboard working (issue #10).
func TestNormalizeCafeteria_BYSEnvelope(t *testing.T) {
	raw := json.RawMessage(`{"Yemekler":[` + sampleDay + `],"Error":null,"CustomData":null}`)
	out, err := NormalizeCafeteria(raw)
	if err != nil {
		t.Fatalf("NormalizeCafeteria: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(out)), "[") {
		t.Fatalf("output should be a bare array, got %s", out)
	}
	var days []map[string]any
	if err := json.Unmarshal(out, &days); err != nil {
		t.Fatalf("output is not an array: %v (%s)", err, out)
	}
	if len(days) != 1 || days[0]["date"] != "2026-10-19" {
		t.Fatalf("unexpected days: %s", out)
	}
}

func TestNormalizeCafeteria_RejectsUnusable(t *testing.T) {
	cases := map[string]string{
		"empty array":       `[]`,
		"empty yemekler":    `{"Yemekler":[],"Error":null}`,
		"missing yemekler":  `{"Error":null}`,
		"upstream error":    `{"Error":{"Message":"Servis kapalı"}}`,
		"invalid json":      `not json`,
		"unexpected scalar": `42`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NormalizeCafeteria(json.RawMessage(raw)); err == nil {
				t.Fatalf("NormalizeCafeteria(%s) = nil error, want error", raw)
			}
		})
	}
}

func TestCafeteria_FallsBackToBYSWhenSKSFails(t *testing.T) {
	var bysHit bool
	a := stubAPI(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Host {
		case "sks.marmara.edu.tr":
			return jsonResponse(503, `service unavailable`), nil
		case "bys.marmara.edu.tr":
			bysHit = true
			if r.Method != http.MethodGet {
				t.Errorf("BYS cafeteria must be GET, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/v2/Public/Mobil/Yemekhane") {
				t.Errorf("unexpected BYS path %q", r.URL.Path)
			}
			return jsonResponse(200, `{"Yemekler":[`+sampleDay+`],"Error":null,"CustomData":null}`), nil
		}
		t.Fatalf("unexpected host %q", r.URL.Host)
		return nil, nil
	})

	out, err := a.Cafeteria(context.Background())
	if err != nil {
		t.Fatalf("Cafeteria: %v", err)
	}
	if !bysHit {
		t.Fatal("BYS fallback was not called")
	}
	var days []map[string]any
	if err := json.Unmarshal(out, &days); err != nil || len(days) != 1 {
		t.Fatalf("fallback output should be a normalised 1-day array, got %s (err %v)", out, err)
	}
}

func TestCafeteria_PrefersSKS(t *testing.T) {
	var bysHit bool
	a := stubAPI(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "bys.marmara.edu.tr" {
			bysHit = true
		}
		return jsonResponse(200, `[`+sampleDay+`]`), nil
	})

	out, err := a.Cafeteria(context.Background())
	if err != nil {
		t.Fatalf("Cafeteria: %v", err)
	}
	if bysHit {
		t.Error("BYS fallback should not be used when SKS succeeds")
	}
	if !strings.HasPrefix(strings.TrimSpace(string(out)), "[") {
		t.Fatalf("output should be a bare array, got %s", out)
	}
}

func TestCafeteria_BothSourcesFail(t *testing.T) {
	a := stubAPI(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(500, `boom`), nil
	})
	_, err := a.Cafeteria(context.Background())
	if err == nil {
		t.Fatal("expected error when both sources fail")
	}
	if !strings.Contains(err.Error(), "SKS") || !strings.Contains(err.Error(), "BYS") {
		t.Fatalf("error should name both sources: %v", err)
	}
}
