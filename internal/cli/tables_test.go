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

func TestRenderProfile(t *testing.T) {
	raw := json.RawMessage(`{
		"Ad": "Ayşe", "Soyad": "Yılmaz", "OgrenciNo": "111111111",
		"Birim": "Mühendislik Fakültesi", "BirimKodu": "BLM",
		"Email": "ayse@marmara.edu.tr", "DanismanEMail": "hoca@marmara.edu.tr",
		"ResimPath": "111111111.jpg", "KullaniciTuru": "Öğrenci",
		"OgretimYili": 2026, "OgretimDonemi": 100001
	}`)
	out, err := renderProfile(raw)
	if err != nil {
		t.Fatalf("renderProfile: %v", err)
	}
	for _, want := range []string{
		"Ayşe Yılmaz", "111111111", "Mühendislik Fakültesi", "BLM",
		"ayse@marmara.edu.tr", "hoca@marmara.edu.tr", "Öğrenci",
		"2026 Güz", "https://bys.marmara.edu.tr/upload/ogrfoto/111111111.jpg",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("profile table missing %q:\n%s", want, out)
		}
	}
}

func TestRenderProfile_NoFields(t *testing.T) {
	if _, err := renderProfile(json.RawMessage(`{"Bilinmeyen":"x"}`)); err == nil {
		t.Error("expected error for a profile with no known fields")
	}
}

func TestTermLabel(t *testing.T) {
	cases := []struct {
		yil, donem int
		want       string
	}{
		{2026, 100001, "2026 Güz"},
		{2026, 100002, "2026 Bahar"},
		{2026, 100003, "2026 Yaz"},
		{2026, 100009, "2026 (dönem 100009)"},
		{0, 0, ""},
		{2026, 0, ""},
	}
	for _, c := range cases {
		if got := termLabel(c.yil, c.donem); got != c.want {
			t.Errorf("termLabel(%d,%d) = %q, want %q", c.yil, c.donem, got, c.want)
		}
	}
}

func TestRenderCafeteria(t *testing.T) {
	raw := json.RawMessage(`[{"date":"2026-10-19","meals":{"Çorba":110,"Köfte":340}}]`)
	out, err := renderCafeteria(raw)
	if err != nil {
		t.Fatalf("renderCafeteria: %v", err)
	}
	if !strings.Contains(out, "2026-10-19") || !strings.Contains(out, "450 kcal") {
		t.Errorf("unexpected table:\n%s", out)
	}
	if _, err := renderCafeteria(json.RawMessage(`[]`)); err == nil {
		t.Error("expected error for an empty menu")
	}
}
