package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderSchedule(t *testing.T) {
	raw := json.RawMessage(`{
		"OgrenciDersProgramListesi": [
			{
				"Gun": 2,
				"Baslangic": "13:30",
				"Bitis": "14:20",
				"DersKodu": "CSE2025",
				"DersAdi": "Data Structures",
				"Derslik": "D101",
				"DerslikAdi": "",
				"OgretimUyesi": "Prof. Dr. Test"
			},
			{
				"Gun": 1,
				"Baslangic": "08:30",
				"Bitis": "09:20",
				"DersKodu": "MAT2085",
				"DersAdi": "Diferansiyel Denklemler",
				"Derslik": "T4-Z07",
				"DerslikAdi": "",
				"OgretimUyesi": "Dr. Öğr. Üyesi NEŞE ÖZDEMİR"
			}
		],
		"Error": null
	}`)

	out, err := renderSchedule(raw)
	if err != nil {
		t.Fatalf("renderSchedule returned error: %v", err)
	}

	// Should contain header columns
	for _, col := range []string{"Gün", "Saat", "Kod", "Ders", "Derslik", "Öğretim Üyesi"} {
		if !strings.Contains(out, col) {
			t.Errorf("output missing column %q", col)
		}
	}

	// Pazartesi (Gun 1) should appear before Salı (Gun 2)
	idxPazartesi := strings.Index(out, "Pazartesi")
	idxSali := strings.Index(out, "Salı")
	if idxPazartesi == -1 || idxSali == -1 {
		t.Fatalf("expected both Pazartesi and Salı in output: %s", out)
	}
	if idxPazartesi > idxSali {
		t.Errorf("expected Pazartesi before Salı in sorted output")
	}

	if !strings.Contains(out, "MAT2085") || !strings.Contains(out, "08:30-09:20") {
		t.Errorf("missing course details for MAT2085")
	}
}

func TestRenderSchedule_Empty(t *testing.T) {
	raw := json.RawMessage(`{"OgrenciDersProgramListesi": []}`)
	_, err := renderSchedule(raw)
	if err == nil {
		t.Fatal("expected error for empty schedule list, got nil")
	}
}

func TestRenderSchedule_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`invalid json`)
	_, err := renderSchedule(raw)
	if err == nil {
		t.Fatal("expected error for invalid json, got nil")
	}
}
