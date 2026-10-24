package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"marmara-cli/internal/client"
)

// Common (Ortak) authenticated endpoints.
const (
	pathProfile = client.HostBYS + "/v2/Public/Mobil/Ortak/KullaniciBilgileri"
	pathCard    = client.HostBYS + "/v2/Public/Mobil/Ortak/GetKartBilgileri"
)

// StudentPhotoBase is where BYS serves student profile photos. The profile
// response carries only a file name/path, so callers join it with this base.
const StudentPhotoBase = client.HostBYS + "/upload/ogrfoto/"

// Profile returns the authenticated user's profile (name, email, photo path, type).
func (a *API) Profile(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathProfile, map[string]any{})
}

// Card returns campus card status and numbers.
func (a *API) Card(ctx context.Context) (json.RawMessage, error) {
	return a.authPost(ctx, pathCard, map[string]any{})
}

// ProfileInfo is a typed view over the KullaniciBilgileri response.
//
// The upstream JSON is not documented and field names drift between app
// versions, so parsing is deliberately defensive: every field has candidate
// aliases and simply stays empty when none of them is present. Consumers that
// need the untouched payload should keep using Profile().
type ProfileInfo struct {
	Ad            string
	Soyad         string
	AdSoyad       string
	OgrenciNo     string
	Birim         string
	AltBirim      string
	BirimKodu     string
	Email         string
	KullaniciTuru string
	PhotoPath     string
	AdvisorName   string
	AdvisorEmail  string
	OgretimYili   int
	OgretimDonemi int
}

// ParseProfile decodes a raw KullaniciBilgileri response into ProfileInfo.
func ParseProfile(raw json.RawMessage) (ProfileInfo, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ProfileInfo{}, fmt.Errorf("decode profile: %w", err)
	}
	if len(m) == 0 {
		return ProfileInfo{}, fmt.Errorf("decode profile: empty response")
	}
	var p ProfileInfo
	p.Ad = firstStringField(m, "Ad", "AD")
	p.Soyad = firstStringField(m, "Soyad", "SOYAD")
	p.AdSoyad = firstStringField(m, "AdSoyad", "AdSoyadi", "Ad_Soyad", "AdiSoyadi")
	if p.AdSoyad == "" {
		p.AdSoyad = strings.TrimSpace(p.Ad + " " + p.Soyad)
	}
	p.OgrenciNo = firstStringField(m, "OgrenciNo", "OgrenciNumarasi", "OgrenciNoStr", "Numara")
	p.Birim = firstStringField(m, "Birim", "BirimAdi", "BirimAd")
	p.AltBirim = firstStringField(m, "AltBirim", "AltBirimAdi", "Bolum", "BolumAdi")
	p.BirimKodu = firstStringField(m, "BirimKodu", "BirimKod")
	p.Email = firstStringField(m, "Email", "EMail", "Eposta", "EPosta", "Mail")
	p.KullaniciTuru = firstStringField(m, "KullaniciTuru", "KullaniciTipi", "Turu")

	// Photo: the mobile app stores a relative file name and joins it with
	// /upload/ogrfoto/. Older builds used a different key name.
	p.PhotoPath = firstStringField(m, "ResimPath", "OgrenciFotograf", "OgrenciFotografi", "Resim", "Fotograf", "Foto")

	// Advisor: DanismanEMail is present in the mobile Dart snapshot; the
	// display name has no confirmed upstream key, so accept the plausible ones.
	p.AdvisorEmail = firstStringField(m, "DanismanEMail", "DanismanMail", "AnadalDanismanMail", "DanismanEposta")
	p.AdvisorName = firstStringField(m, "DanismanAdSoyad", "DanismanAdi", "DanismanAd", "Danisman", "Danismani")

	p.OgretimYili = firstIntField(m, "OgretimYili")
	p.OgretimDonemi = firstIntField(m, "OgretimDonemi")
	return p, nil
}

// PhotoURL returns an absolute URL for the student's profile photo, or "" when
// the response carries no photo. Absolute upstream URLs are returned as-is.
func (p ProfileInfo) PhotoURL() string {
	path := strings.TrimSpace(p.PhotoPath)
	if path == "" {
		return ""
	}
	if u, err := url.Parse(path); err == nil && u.Scheme != "" && u.Host != "" {
		return path
	}
	// Strip a leading slash so JoinPath treats it as relative to the base.
	path = strings.TrimLeft(path, "/")
	if strings.HasPrefix(path, "upload/") || strings.HasPrefix(path, "v2/upload/") {
		return client.HostBYS + "/" + path
	}
	return StudentPhotoBase + url.PathEscape(path)
}

// firstStringField returns the first non-empty string value found under keys.
func firstStringField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch s := v.(type) {
		case string:
			if t := strings.TrimSpace(s); t != "" {
				return t
			}
		case float64:
			// Some numeric-ish fields (student number) arrive as JSON numbers.
			if s != 0 {
				return fmt.Sprintf("%.0f", s)
			}
		}
	}
	return ""
}

// firstIntField returns the first positive integer value found under keys.
func firstIntField(m map[string]any, keys ...string) int {
	for _, k := range keys {
		switch v := m[k].(type) {
		case float64:
			if v > 0 {
				return int(v)
			}
		case int:
			if v > 0 {
				return v
			}
		case string:
			var n int
			if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &n); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}
