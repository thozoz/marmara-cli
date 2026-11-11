package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"marmara-cli/internal/api"
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
		Yil            int     `json:"Yil"`
		GANO           float64 `json:"GANO"`
		YANO           float64 `json:"YANO"`
		DersAlinmismi  bool    `json:"DersAlinmismi"`
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

// ---- exams ----

func renderExams(raw json.RawMessage) (string, error) {
	var env struct {
		List []struct {
			DersKodu       string `json:"dersKodu"`
			DersAdi        string `json:"dersAdi"`
			SinavTuru      string `json:"sinavTuru"`
			SinavTarihi    string `json:"sinavTarihi"`
			SinavSaati     string `json:"sinavSaati"`
			DerslikKodu    string `json:"derslikKodu"`
			SinavCondition string `json:"sinavCondition"`
		} `json:"OgrenciDersSinavListesi"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	if len(env.List) == 0 {
		return "", errors.New("no exams")
	}
	rows := make([][]string, 0, len(env.List))
	for _, e := range env.List {
		durum := "yaklaşıyor"
		if e.SinavCondition == "passed" {
			durum = "yapıldı"
		}
		rows = append(rows, []string{examDate(e.SinavTarihi), uniqCSV(e.SinavSaati), e.DersKodu, e.DersAdi, e.SinavTuru, uniqCSV(e.DerslikKodu), durum})
	}
	// Chronological, so the next exams end up right above the prompt.
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i][0]+" "+fmt.Sprintf("%5s", rows[i][1]) < rows[j][0]+" "+fmt.Sprintf("%5s", rows[j][1])
	})
	return tab([]string{"Tarih", "Saat", "Kod", "Ders", "Tür", "Derslik", "Durum"}, rows), nil
}

// examDate turns BYS's "16.11.2026, Monday" into a sortable "2026-11-16".
func examDate(s string) string {
	if len(s) >= 10 && s[2] == '.' && s[5] == '.' {
		return s[6:10] + "-" + s[3:5] + "-" + s[0:2]
	}
	return s
}

// uniqCSV collapses BYS's per-room repeats ("10:30,10:30") to unique values.
func uniqCSV(s string) string {
	var out []string
	seen := map[string]bool{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}

func renderGrades(raw json.RawMessage) (string, error) {
	var env struct {
		OgrenciDersNotListesi []struct {
			DersKodu   string  `json:"DersKodu"`
			DersAdi    string  `json:"DersAdi"`
			HBN        string  `json:"HBN"`
			HarfNotu   string  `json:"HarfNotu"`
			BN         string  `json:"BN"` // grade as sent by BYS: "93", or "-" before grading
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
		not := strings.TrimSpace(g.BN)
		if not == "" {
			not = fmt.Sprintf("%.0f", g.BasariNotu)
		}
		rows = append(rows, []string{g.DersKodu, g.DersAdi, strings.TrimSpace(harf), not})
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
			KARTNO string `json:"KARTNO"`
			AKTIF  bool   `json:"AKTIF"`
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
		rows = append(rows, []string{c.KARTNO, durum})
	}
	return tab([]string{"Kart No", "Durum"}, rows), nil
}

// ---- schedule (weekly timetable) ----

type scheduleEnvelope struct {
	OgrenciDersProgramListesi []scheduleEntry `json:"OgrenciDersProgramListesi"`
}

type scheduleEntry struct {
	Gun          int    `json:"Gun"`
	Baslangic    string `json:"Baslangic"`
	Bitis        string `json:"Bitis"`
	DersKodu     string `json:"DersKodu"`
	DersAdi      string `json:"DersAdi"`
	Derslik      string `json:"Derslik"`
	DerslikAdi   string `json:"DerslikAdi"`
	OgretimUyesi string `json:"OgretimUyesi"`
}

var scheduleDays = map[int]string{
	1: "Pazartesi",
	2: "Salı",
	3: "Çarşamba",
	4: "Perşembe",
	5: "Cuma",
	6: "Cumartesi",
	7: "Pazar",
}

func renderSchedule(raw json.RawMessage) (string, error) {
	var env scheduleEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	if len(env.OgrenciDersProgramListesi) == 0 {
		return "", errors.New("no schedule entries")
	}

	items := make([]scheduleEntry, len(env.OgrenciDersProgramListesi))
	copy(items, env.OgrenciDersProgramListesi)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Gun != items[j].Gun {
			return items[i].Gun < items[j].Gun
		}
		if items[i].Baslangic != items[j].Baslangic {
			return items[i].Baslangic < items[j].Baslangic
		}
		return items[i].DersKodu < items[j].DersKodu
	})

	var rows [][]string
	for _, it := range items {
		gun := scheduleDays[it.Gun]
		if gun == "" {
			gun = fmt.Sprintf("Gün %d", it.Gun)
		}
		saat := it.Baslangic
		if it.Bitis != "" {
			saat = fmt.Sprintf("%s-%s", it.Baslangic, it.Bitis)
		}
		derslik := strings.TrimSpace(it.Derslik)
		if derslik == "" {
			derslik = strings.TrimSpace(it.DerslikAdi)
		}
		rows = append(rows, []string{
			gun,
			saat,
			it.DersKodu,
			it.DersAdi,
			derslik,
			strings.TrimSpace(it.OgretimUyesi),
		})
	}
	return tab([]string{"Gün", "Saat", "Kod", "Ders", "Derslik", "Öğretim Üyesi"}, rows), nil
}

// ---- profile ----

func renderProfile(raw json.RawMessage) (string, error) {
	p, err := api.ParseProfile(raw)
	if err != nil {
		return "", err
	}
	// Curated set of known fields, in order, shown only when present.
	fields := []struct{ label, value string }{
		{"Ad Soyad", p.AdSoyad},
		{"Öğrenci No", p.OgrenciNo},
		{"Birim", p.Birim},
		{"Alt Birim", p.AltBirim},
		{"Birim Kodu", p.BirimKodu},
		{"E-posta", p.Email},
		{"Danışman", p.AdvisorName},
		{"Danışman E-posta", p.AdvisorEmail},
		{"Tür", p.KullaniciTuru},
		{"Dönem", termLabel(p.OgretimYili, p.OgretimDonemi)},
		{"Fotoğraf", p.PhotoURL()},
	}
	var rows [][]string
	for _, f := range fields {
		if strings.TrimSpace(f.value) != "" {
			rows = append(rows, []string{f.label, f.value})
		}
	}
	if len(rows) == 0 {
		return "", errors.New("no profile fields")
	}
	return tab(nil, rows), nil
}

// termLabel formats a term as "2026 Güz" style when it can, else "".
func termLabel(yil, donem int) string {
	if yil <= 0 || donem <= 0 {
		return ""
	}
	// BYS term codes observed: ...001 Güz, ...002 Bahar (e.g. 100001 / 100002).
	name := ""
	switch donem % 10 {
	case 1:
		name = "Güz"
	case 2:
		name = "Bahar"
	case 3:
		name = "Yaz"
	}
	if name == "" {
		return fmt.Sprintf("%d (dönem %d)", yil, donem)
	}
	return fmt.Sprintf("%d %s", yil, name)
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
