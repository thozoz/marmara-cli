package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
)

// tab builds a tab-aligned table. header may be empty. Each row is a slice of
// cells joined by tabs.
func tab(header []string, rows [][]string) string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	if len(header) > 0 {
		fmt.Fprintln(w, strings.Join(header, "\t"))
		sep := make([]string, len(header))
		for i, h := range header {
			sep[i] = strings.Repeat("-", len(h))
		}
		fmt.Fprintln(w, strings.Join(sep, "\t"))
	}
	for _, r := range rows {
		fmt.Fprintln(w, strings.Join(r, "\t"))
	}
	w.Flush()
	return strings.TrimRight(b.String(), "\n")
}

// ---- transcript ----

type transcriptEnvelope struct {
	Transcript struct {
		Transcript transcriptBody `json:"Transcript"`
	} `json:"Transcript"`
}

type transcriptBody struct {
	Ortalama            float64 `json:"Ortalama"`
	OrtalamaYuzlu       float64 `json:"OrtalamaYuzlu"`
	TamamlananKredi     float64 `json:"TamamlananKredi"`
	TamamlananEctsKredi float64 `json:"TamamlananEctsKredi"`
	Ogrenci             struct {
		Ad        string `json:"Ad"`
		Bolum     string `json:"Bolum"`
		OgrenciNo string `json:"OgrenciNo"`
	} `json:"Ogrenci"`
	Semesters []struct {
		Yil           int     `json:"Yil"`
		GANO          float64 `json:"GANO"`
		YANO          float64 `json:"YANO"`
		DersAlinmismi bool    `json:"DersAlinmismi"`
		TanscriptDonem *struct {
			Aciklama string `json:"Aciklama"`
		} `json:"TanscriptDonem"`
		Dersler []struct {
			DersKodu  string  `json:"DersKodu"`
			DersAdi   string  `json:"DersAdi"`
			Kredi     float64 `json:"Kredi"`
			ECTSKredi float64 `json:"ECTSKredi"`
			DersNotu  *struct {
				HBN        string  `json:"HBN"`
				BasariNotu float64 `json:"BasariNotu"`
			} `json:"DersNotu"`
		} `json:"Dersler"`
	} `json:"Semesters"`
}

func renderTranscript(raw json.RawMessage) (string, error) {
	var env transcriptEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	t := env.Transcript.Transcript
	if len(t.Semesters) == 0 {
		return "", errors.New("no transcript data")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s) — %s\n", t.Ogrenci.Ad, t.Ogrenci.OgrenciNo, t.Ogrenci.Bolum)
	fmt.Fprintf(&b, "GANO %.2f  |  %.1f/100  |  %.0f kredi / %.0f ECTS tamamlandı\n",
		t.Ortalama, t.OrtalamaYuzlu, t.TamamlananKredi, t.TamamlananEctsKredi)

	for _, s := range t.Semesters {
		if !s.DersAlinmismi {
			continue
		}
		donem := ""
		if s.TanscriptDonem != nil {
			donem = s.TanscriptDonem.Aciklama
		}
		fmt.Fprintf(&b, "\n%d %s — YANO %.2f (GANO %.2f)\n", s.Yil, donem, s.YANO, s.GANO)
		var rows [][]string
		for _, d := range s.Dersler {
			hbn, not := "", ""
			if d.DersNotu != nil {
				hbn = d.DersNotu.HBN
				not = fmt.Sprintf("%.0f", d.DersNotu.BasariNotu)
			}
			rows = append(rows, []string{d.DersKodu, d.DersAdi, fmt.Sprintf("%.0f", d.Kredi), fmt.Sprintf("%.0f", d.ECTSKredi), hbn, not})
		}
		b.WriteString(tab([]string{"Kod", "Ders", "Kr", "ECTS", "Harf", "Not"}, rows))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// ---- grades (current term) — best effort; shape mirrors transcript courses ----

func renderGrades(raw json.RawMessage) (string, error) {
	var env struct {
		OgrenciDersNotListesi []struct {
			DersKodu   string  `json:"DersKodu"`
			DersAdi    string  `json:"DersAdi"`
			HBN        string  `json:"HBN"`
			HarfNotu   string  `json:"HarfNotu"`
			BasariNotu float64 `json:"BasariNotu"`
			Ortalama   float64 `json:"Ortalama"`
		} `json:"OgrenciDersNotListesi"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	if len(env.OgrenciDersNotListesi) == 0 {
		return "", errors.New("no grades")
	}
	var rows [][]string
	for _, g := range env.OgrenciDersNotListesi {
		harf := g.HBN
		if harf == "" {
			harf = g.HarfNotu
		}
		rows = append(rows, []string{g.DersKodu, g.DersAdi, harf, fmt.Sprintf("%.0f", g.BasariNotu)})
	}
	return tab([]string{"Kod", "Ders", "Harf", "Not"}, rows), nil
}

// ---- cafeteria ----

func renderCafeteria(raw json.RawMessage) (string, error) {
	var days []struct {
		Date  string         `json:"date"`
		Meals map[string]int `json:"meals"`
	}
	if err := json.Unmarshal(raw, &days); err != nil {
		return "", err
	}
	if len(days) == 0 {
		return "", errors.New("no menu")
	}
	var rows [][]string
	for _, d := range days {
		var items []string
		var kcal int
		for name, c := range d.Meals {
			items = append(items, name)
			kcal += c
		}
		sort.Strings(items)
		rows = append(rows, []string{d.Date, strings.Join(items, ", "), fmt.Sprintf("%d kcal", kcal)})
	}
	return tab([]string{"Tarih", "Menü", "Toplam"}, rows), nil
}

// ---- campus card ----

func renderCard(raw json.RawMessage) (string, error) {
	var env struct {
		Entities []struct {
			KARTNO string  `json:"KARTNO"`
			AKTIF  bool    `json:"AKTIF"`
			Bakiye float64 `json:"Bakiye"`
		} `json:"Entities"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	if len(env.Entities) == 0 {
		return "", errors.New("no cards")
	}
	var rows [][]string
	for _, c := range env.Entities {
		durum := "pasif"
		if c.AKTIF {
			durum = "aktif"
		}
		rows = append(rows, []string{c.KARTNO, durum, fmt.Sprintf("%.2f", c.Bakiye)})
	}
	return tab([]string{"Kart No", "Durum", "Bakiye"}, rows), nil
}

// ---- profile ----

func renderProfile(raw json.RawMessage) (string, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", err
	}
	// Show a curated set of known fields, in order, when present.
	fields := []struct{ key, label string }{
		{"Ad", "Ad"},
		{"Soyad", "Soyad"},
		{"Birim", "Birim"},
		{"AltBirim", "Alt Birim"},
		{"Email", "E-posta"},
		{"DanismanEMail", "Danışman"},
		{"KullaniciTuru", "Tür"},
	}
	var rows [][]string
	for _, f := range fields {
		if v, ok := m[f.key]; ok {
			s := fmt.Sprintf("%v", v)
			if strings.TrimSpace(s) != "" && s != "<nil>" {
				rows = append(rows, []string{f.label, s})
			}
		}
	}
	if len(rows) == 0 {
		return "", errors.New("no profile fields")
	}
	return tab(nil, rows), nil
}

// ---- generic title list (news, announcements, events, calendar) ----

func renderTitleList(raw json.RawMessage) (string, error) {
	// Two shapes: {"data":[...]} or a bare [...] array. Each item may use
	// "title" or "summary" for its label and "date"/"start"/etc. for a date.
	extract := func(items []map[string]any) (string, error) {
		var rows [][]string
		for _, it := range items {
			title := firstString(it, "title", "summary", "baslik", "name")
			if title == "" {
				continue
			}
			date := firstString(it, "date", "datetime", "start", "tarih", "startdate", "start_date")
			rows = append(rows, []string{date, title})
		}
		if len(rows) == 0 {
			return "", errors.New("no titled items")
		}
		return tab([]string{"Tarih", "Başlık"}, rows), nil
	}

	var wrapped struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Data) > 0 {
		return extract(wrapped.Data)
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return extract(arr)
	}
	return "", errors.New("unrecognized list shape")
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}
