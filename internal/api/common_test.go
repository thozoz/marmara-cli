package api

import (
	"encoding/json"
	"testing"
)

func TestParseProfile_KnownFields(t *testing.T) {
	raw := json.RawMessage(`{
		"Ad": "Ayşe", "Soyad": "Yılmaz", "OgrenciNo": "111111111",
		"Birim": "Mühendislik Fakültesi", "AltBirim": "Bilgisayar Mühendisliği",
		"BirimKodu": "BLM", "Email": "ayse@marmara.edu.tr",
		"KullaniciTuru": "Öğrenci", "ResimPath": "111111111.jpg",
		"DanismanEMail": "hoca@marmara.edu.tr",
		"OgretimYili": 2026, "OgretimDonemi": 100001
	}`)
	p, err := ParseProfile(raw)
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}
	if p.AdSoyad != "Ayşe Yılmaz" {
		t.Errorf("AdSoyad = %q", p.AdSoyad)
	}
	if p.OgrenciNo != "111111111" {
		t.Errorf("OgrenciNo = %q", p.OgrenciNo)
	}
	if p.AdvisorEmail != "hoca@marmara.edu.tr" {
		t.Errorf("AdvisorEmail = %q", p.AdvisorEmail)
	}
	if p.OgretimYili != 2026 || p.OgretimDonemi != 100001 {
		t.Errorf("term = %d/%d", p.OgretimYili, p.OgretimDonemi)
	}
	if want := "https://bys.marmara.edu.tr/upload/ogrfoto/111111111.jpg"; p.PhotoURL() != want {
		t.Errorf("PhotoURL = %q, want %q", p.PhotoURL(), want)
	}
}

// Field names drift between app versions, so aliases and numbers must be handled.
func TestParseProfile_AliasesAndNumbers(t *testing.T) {
	raw := json.RawMessage(`{
		"AdSoyad": "  Mehmet Kaya ",
		"OgrenciNumarasi": 222222222,
		"Bolum": "Elektrik Elektronik",
		"Mail": "mehmet@marmara.edu.tr",
		"DanismanMail": "dan@marmara.edu.tr",
		"OgrenciFotograf": "/upload/ogrfoto/222222222.jpg"
	}`)
	p, err := ParseProfile(raw)
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}
	if p.AdSoyad != "Mehmet Kaya" {
		t.Errorf("AdSoyad = %q", p.AdSoyad)
	}
	if p.OgrenciNo != "222222222" {
		t.Errorf("OgrenciNo = %q", p.OgrenciNo)
	}
	if p.Email != "mehmet@marmara.edu.tr" || p.AdvisorEmail != "dan@marmara.edu.tr" {
		t.Errorf("emails = %q / %q", p.Email, p.AdvisorEmail)
	}
	// An already-qualified upload path must not be double-prefixed.
	if want := "https://bys.marmara.edu.tr/upload/ogrfoto/222222222.jpg"; p.PhotoURL() != want {
		t.Errorf("PhotoURL = %q, want %q", p.PhotoURL(), want)
	}
}

func TestParseProfile_EmptyAndPhotoFallbacks(t *testing.T) {
	if _, err := ParseProfile(json.RawMessage(`{}`)); err == nil {
		t.Error("empty profile should error")
	}
	if _, err := ParseProfile(json.RawMessage(`null`)); err == nil {
		t.Error("null profile should error")
	}

	cases := map[string]string{
		"":                          "",
		"   ":                       "",
		"https://cdn.example/f.jpg": "https://cdn.example/f.jpg",
		"f.jpg":                     "https://bys.marmara.edu.tr/upload/ogrfoto/f.jpg",
		"folder/f.jpg":              "https://bys.marmara.edu.tr/upload/ogrfoto/folder%2Ff.jpg",
		"upload/ogrfoto/x.jpg":      "https://bys.marmara.edu.tr/upload/ogrfoto/x.jpg",
		"/v2/upload/ogrfoto/y.jpg":  "https://bys.marmara.edu.tr/v2/upload/ogrfoto/y.jpg",
	}
	for in, want := range cases {
		got := ProfileInfo{PhotoPath: in}.PhotoURL()
		if got != want {
			t.Errorf("PhotoURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstStringField_SkipsEmpty(t *testing.T) {
	m := map[string]any{"a": "  ", "b": nil, "c": "ok", "d": 0.0}
	if got := firstStringField(m, "a", "b", "c", "d"); got != "ok" {
		t.Fatalf("firstStringField = %q, want ok", got)
	}
	if got := firstStringField(m, "a", "b"); got != "" {
		t.Fatalf("firstStringField = %q, want empty", got)
	}
}
