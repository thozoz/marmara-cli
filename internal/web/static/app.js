"use strict";

// ---- tiny helpers ----
const $ = (sel) => document.querySelector(sel);
const esc = (s) =>
  String(s ?? "").replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

async function api(path, opts) {
  const res = await fetch(path, opts);
  let data = null;
  try { data = await res.json(); } catch (_) {}
  return { ok: res.ok, status: res.status, data };
}

function fmt(n, d = 0) {
  const x = Number(n);
  return Number.isFinite(x) ? x.toFixed(d) : "";
}

// ---- view registry ----
// Each view: {label, endpoint, auth, render(data)->html}
const VIEWS = {
  summary:      { label: "Özet",              auth: true,  special: true },
  transcript:   { label: "Transkript",        endpoint: "transcript",    auth: true,  render: renderTranscript },
  grades:       { label: "Notlar",            endpoint: "grades",        auth: true,  render: renderGrades },
  schedule:     { label: "Ders Programı",     endpoint: "schedule",      auth: true,  render: renderSchedule },
  exams:        { label: "Sınavlar",          endpoint: "exams",         auth: true,  render: renderExams },
  profile:      { label: "Profil",            endpoint: "profile",       auth: true,  render: renderProfile },
  card:         { label: "Kart",              endpoint: "card",          auth: true,  render: renderCard },
  cafeteria:    { label: "Yemekhane",         endpoint: "cafeteria",     auth: false, render: renderCafeteria },
  calendar:     { label: "Akademik Takvim",   endpoint: "calendar",      auth: false, render: renderTitleList },
  news:         { label: "Haberler",          endpoint: "news",          auth: false, render: renderTitleList },
  announcements:{ label: "Duyurular",         endpoint: "announcements", auth: false, render: renderTitleList },
  events:       { label: "Etkinlikler",       endpoint: "events",        auth: false, render: renderTitleList },
  clubs:        { label: "Kulüpler",          endpoint: "clubs",         auth: false, render: renderClubs },
  "campus-maps":{ label: "Kampüs Haritası",   endpoint: "campus-maps",   auth: false, render: renderCampusMaps },
};

const NAV = [
  ["", ["summary"]],
  ["Akademik", ["transcript", "grades", "schedule", "exams"]],
  ["Kişisel", ["profile", "card"]],
  ["Kampüs", ["cafeteria", "calendar", "news", "announcements", "events", "clubs", "campus-maps"]],
];

// ---- renderers ----
function tableHTML(headers, rows, numCols = []) {
  const th = headers.map((h, i) =>
    `<th class="${numCols.includes(i) ? "num" : ""}">${esc(h)}</th>`).join("");
  const tr = rows.map((r) =>
    "<tr>" + r.map((c, i) =>
      `<td class="${numCols.includes(i) ? "num" : ""}">${c}</td>`).join("") + "</tr>").join("");
  return `<div class="tbl-wrap"><table><thead><tr>${th}</tr></thead><tbody>${tr}</tbody></table></div>`;
}

function renderTranscript(d) {
  const t = d?.Transcript?.Transcript;
  if (!t || !Array.isArray(t.Semesters)) return emptyMsg("Transkript verisi yok.");
  const o = t.Ogrenci || {};
  let html = `<div class="stat-row">
    <div class="stat"><div class="label">GANO</div><div class="value good">${fmt(t.Ortalama, 2)}</div></div>
    <div class="stat"><div class="label">Yüzlük</div><div class="value">${fmt(t.OrtalamaYuzlu, 1)}</div></div>
    <div class="stat"><div class="label">Kredi</div><div class="value">${fmt(t.TamamlananKredi)}</div></div>
    <div class="stat"><div class="label">ECTS</div><div class="value">${fmt(t.TamamlananEctsKredi)}</div></div>
  </div>
  <p class="muted">${esc(o.Ad || "")} · ${esc(o.Bolum || "")} · ${esc(o.OgrenciNo || "")}</p>`;

  for (const s of t.Semesters) {
    if (!s.DersAlinmismi) continue;
    const donem = s.TanscriptDonem ? s.TanscriptDonem.Aciklama : "";
    const rows = (s.Dersler || []).map((c) => {
      const n = c.DersNotu || {};
      return [
        `<span class="grade">${esc(c.DersKodu)}</span>`,
        esc(c.DersAdi),
        fmt(c.Kredi), fmt(c.ECTSKredi),
        `<span class="grade">${esc(n.HBN || "")}</span>`,
        fmt(n.BasariNotu),
      ];
    });
    html += `<div class="sem-head">${esc(s.Yil)} ${esc(donem)}
      <span class="yano">YANO ${fmt(s.YANO, 2)} · GANO ${fmt(s.GANO, 2)}</span></div>`;
    html += tableHTML(["Kod", "Ders", "Kr", "ECTS", "Harf", "Not"], rows, [2, 3, 5]);
  }
  return html;
}

function renderGrades(d) {
  const list = d?.OgrenciDersNotListesi || [];
  if (!list.length) return emptyMsg("Aktif dönem notu yok (dönem başlayınca dolacak).");
  const rows = list.map((g) => [
    `<span class="grade">${esc(g.DersKodu || "")}</span>`,
    esc(g.DersAdi || ""),
    `<span class="grade">${esc(g.HBN || g.HarfNotu || "")}</span>`,
    fmt(g.BasariNotu),
  ]);
  return tableHTML(["Kod", "Ders", "Harf", "Not"], rows, [3]);
}

const SCHEDULE_DAYS = {
  1: "Pazartesi",
  2: "Salı",
  3: "Çarşamba",
  4: "Perşembe",
  5: "Cuma",
  6: "Cumartesi",
  7: "Pazar",
};

function renderSchedule(d) {
  const list = d?.OgrenciDersProgramListesi || [];
  if (!list.length) return emptyMsg("Aktif dönem ders programı yok (dönem başlayınca dolacak).");
  const sorted = [...list].sort((a, b) => {
    if ((a.Gun || 0) !== (b.Gun || 0)) return (a.Gun || 0) - (b.Gun || 0);
    return (a.Baslangic || "").localeCompare(b.Baslangic || "");
  });
  const rows = sorted.map((s) => [
    esc(SCHEDULE_DAYS[s.Gun] || `Gün ${s.Gun ?? ""}`),
    `${esc(s.Baslangic || "")}${s.Bitis ? " - " + esc(s.Bitis) : ""}`,
    `<span class="code">${esc(s.DersKodu || "")}</span>`,
    esc(s.DersAdi || ""),
    esc((s.Derslik || s.DerslikAdi || "").trim()),
    esc((s.OgretimUyesi || "").trim()),
  ]);
  return tableHTML(["Gün", "Saat", "Kod", "Ders", "Derslik", "Öğretim Üyesi"], rows);
}

function renderExams(d) {
  const list = d?.OgrenciDersSinavListesi || [];
  if (!list.length) return emptyMsg("Aktif dönem sınavı yok (dönem başlayınca dolacak).");
  return rawJSON(d);
}

function renderProfile(d) {
  const fields = [
    ["Ad", "Ad"], ["Soyad", "Soyad"], ["Birim", "Birim"], ["AltBirim", "Alt Birim"],
    ["Email", "E-posta"], ["DanismanEMail", "Danışman"], ["KullaniciTuru", "Tür"],
  ];
  const rows = fields
    .filter(([k]) => d[k] != null && String(d[k]).trim() !== "")
    .map(([k, lab]) => `<dt>${esc(lab)}</dt><dd>${esc(d[k])}</dd>`);
  if (!rows.length) return rawJSON(d);
  return `<dl class="kv">${rows.join("")}</dl>`;
}

function renderCard(d) {
  const list = d?.Entities || [];
  if (!list.length) return emptyMsg("Kart bulunamadı.");
  const rows = list.map((c) => [
    `<span class="grade">${esc(c.KARTNO || "")}</span>`,
    c.AKTIF ? "aktif" : "pasif",
  ]);
  return tableHTML(["Kart No", "Durum"], rows);
}

// Menu variants present on a cafeteria day, in display order.
const MENU_VARIANTS = [
  ["meals", "Standart"],
  ["meals_alternative", "Alternatif"],
  ["meals_vejetaryen", "Vejetaryen"],
  ["meals_vegan", "Vegan"],
];

function menuVariantHTML(meals) {
  const entries = Object.entries(meals || {}).sort((a, b) => a[0].localeCompare(b[0], "tr"));
  if (!entries.length) return "";
  let total = 0;
  const rows = entries.map(([name, cal]) => {
    total += Number(cal || 0);
    return `<li><span>${esc(name)}</span><span class="kcal">${esc(cal)}</span></li>`;
  }).join("");
  return `<ul class="menu">${rows}</ul><div class="menu-total">Toplam ${total} kcal</div>`;
}

function dayMenuHTML(day) {
  const parts = MENU_VARIANTS
    .filter(([k]) => day[k] && Object.keys(day[k]).length)
    .map(([k, label]) => `<div class="variant"><h4>${label}</h4>${menuVariantHTML(day[k])}</div>`);
  return `<div class="menu-grid">${parts.join("")}</div>`;
}

function renderCafeteria(d) {
  if (!Array.isArray(d) || !d.length) return emptyMsg("Menü yok.");
  // Newest first so the latest menu is at the top (no scrolling to find it).
  const days = [...d].sort((a, b) => String(b.date).localeCompare(String(a.date)));
  const cards = days.map((day) =>
    `<div class="card wide"><h3>${esc(day.date)}</h3>${dayMenuHTML(day)}</div>`);
  return `<div class="stack">${cards.join("")}</div>`;
}

// pickMenuDay returns today's menu if present, else the nearest upcoming day,
// else the most recent one — never the oldest.
function pickMenuDay(days) {
  const sorted = [...days].sort((a, b) => String(a.date).localeCompare(String(b.date)));
  const t0 = todayISO();
  return sorted.find((x) => x.date === t0)
    || sorted.find((x) => x.date > t0)
    || sorted[sorted.length - 1];
}

function firstStr(o, keys) {
  for (const k of keys) if (o[k] && String(o[k]).trim() !== "") return o[k];
  return "";
}
function renderTitleList(d) {
  const items = Array.isArray(d) ? d : (d && Array.isArray(d.data) ? d.data : []);
  const rows = items.map((it) => {
    const title = firstStr(it, ["title", "summary", "baslik", "name"]);
    if (!title) return null;
    const date = firstStr(it, ["date", "datetime", "start", "tarih", "startdate", "start_date"]);
    const link = firstStr(it, ["link", "url"]);
    const tCell = link ? `<a class="link" href="${esc(link)}" target="_blank" rel="noopener">${esc(title)}</a>` : esc(title);
    return [esc(date), tCell];
  }).filter(Boolean);
  if (!rows.length) return emptyMsg("Kayıt yok.");
  return tableHTML(["Tarih", "Başlık"], rows);
}

function renderClubs(d) {
  const list = Array.isArray(d) ? d : (d?.data || []);
  if (!list.length) return emptyMsg("Kulüp yok.");
  const cards = list.map((c) => {
    const name = firstStr(c, ["clubAdi", "clubName", "adi", "title", "name"]) || ("Kulüp #" + esc(c.clubID || ""));
    return `<div class="card"><h3>${esc(name)}</h3>
      <div class="meta">Üye: ${esc(c.clubUyeSayisi || "?")} · Kuruluş: ${esc(c.clubKurulusTarihi || "?")}</div></div>`;
  });
  return `<div class="cards">${cards.join("")}</div>`;
}

function renderCampusMaps(d) {
  const list = d?.data || [];
  if (!list.length) return emptyMsg("Kayıt yok.");
  const rows = list.map((m) => [
    esc(m.campus || ""), esc(m.type || ""), esc(m.title || ""),
    m.latitude && m.longitude
      ? `<a class="link" target="_blank" rel="noopener" href="https://maps.google.com/?q=${esc(m.latitude)},${esc(m.longitude)}">harita</a>`
      : "",
  ]);
  return tableHTML(["Yerleşke", "Tür", "Bina", ""], rows);
}

// ---- summary ----
function todayISO() {
  const d = new Date();
  return d.getFullYear() + "-" + String(d.getMonth() + 1).padStart(2, "0") + "-" + String(d.getDate()).padStart(2, "0");
}
function miniList(items, limit) {
  const rows = items.slice(0, limit).map((it) => {
    const title = firstStr(it, ["title", "summary", "baslik", "name"]);
    if (!title) return "";
    const date = firstStr(it, ["date", "datetime", "start", "tarih", "startdate", "start_date"]);
    const link = firstStr(it, ["link", "url"]);
    const t = link ? `<a class="link" href="${esc(link)}" target="_blank" rel="noopener">${esc(title)}</a>` : esc(title);
    const dt = date ? `<span class="mini-date">${esc(String(date).slice(0, 10))}</span>` : "";
    return `<li>${dt}<span>${t}</span></li>`;
  }).filter(Boolean).join("");
  return rows ? `<ul class="mini">${rows}</ul>` : `<div class="empty">Kayıt yok.</div>`;
}

async function renderSummary() {
  setPanel(`<div class="loading">Yükleniyor…</div>`);
  const [tr, caf, ex, prof, card, news, ann, cal] = await Promise.all([
    api("/api/transcript"), api("/api/cafeteria"), api("/api/exams"),
    api("/api/profile"), api("/api/card"),
    api("/api/news"), api("/api/announcements"), api("/api/calendar"),
  ]);
  if ([tr, caf, ex, prof, card].some((r) => r.status === 401)) { showLogin(); return; }

  let html = "";

  // GANO stats
  const t = tr.data?.Transcript?.Transcript;
  if (t) {
    html += `<div class="stat-row">
      <div class="stat"><div class="label">GANO</div><div class="value good">${fmt(t.Ortalama, 2)}</div></div>
      <div class="stat"><div class="label">Yüzlük</div><div class="value">${fmt(t.OrtalamaYuzlu, 1)}</div></div>
      <div class="stat"><div class="label">Kredi</div><div class="value">${fmt(t.TamamlananKredi)}</div></div>
      <div class="stat"><div class="label">ECTS</div><div class="value">${fmt(t.TamamlananEctsKredi)}</div></div>
    </div>`;
  }

  // Menu (all variants + kcal) — full width on top. Show today if available,
  // otherwise the most recent published day.
  const days = Array.isArray(caf.data) ? caf.data : [];
  const day = days.length ? pickMenuDay(days) : null;
  const isToday = day && day.date === todayISO();
  html += `<div class="section"><h2>${isToday ? "Bugün " : ""}Yemekhane${day ? ` <span class="h2-date">${esc(day.date)}</span>` : ""}</h2>`;
  html += day ? dayMenuHTML(day) : `<div class="empty">Menü yok.</div>`;
  html += `</div>`;

  // Build each widget as its own section, then lay them out in fixed columns
  // (short cards stacked in the first column) so the row fills evenly.
  const p = prof.data || {};
  const line = (lab, v) => v ? `<dt>${lab}</dt><dd>${esc(v)}</dd>` : "";
  const profileSec = `<div class="section"><h2>Profil</h2>` +
    (p.Ad ? `<dl class="kv flat">
      ${line("Ad", [p.Ad, p.Soyad].filter(Boolean).join(" "))}
      ${line("Bölüm", p.Birim)}
      ${line("Danışman", p.DanismanEMail)}
      ${line("E-posta", p.Email)}
    </dl>` : `<div class="empty">—</div>`) + `</div>`;

  const cards = card.data?.Entities || [];
  let cardBody;
  if (cards.length) {
    const primary = cards.find((c) => c.AKTIF) || cards[0];
    const totalActive = cards.filter((c) => c.AKTIF).length;
    const totalPassive = cards.length - totalActive;
    const counts = [];
    if (totalActive > 0) counts.push(`${totalActive} aktif`);
    if (totalPassive > 0) counts.push(`${totalPassive} pasif`);
    cardBody = `<div class="big-num" style="font-size:1.4rem;letter-spacing:0.04em"><span class="grade">${esc(primary.KARTNO || "")}</span></div>` +
      `<div class="muted">${primary.AKTIF ? "aktif kart" : "pasif kart"}</div>` +
      (cards.length > 1 ? `<div class="muted" style="margin-top:6px">${cards.length} kart · ${counts.join(" · ")}</div>` : "");
  } else cardBody = `<div class="empty">Kart yok.</div>`;
  const cardSec = `<div class="section"><h2>Kart</h2>${cardBody}</div>`;

  const exList = ex.data?.OgrenciDersSinavListesi || [];
  const examsSec = `<div class="section"><h2>Yaklaşan Sınavlar</h2>` +
    (exList.length ? renderExams(ex.data) : `<div class="empty">Aktif dönem sınavı yok.</div>`) + `</div>`;

  const calItems = (Array.isArray(cal.data) ? cal.data : (cal.data?.data || []));
  const upcoming = calItems.filter((it) => {
    const d = firstStr(it, ["startdate", "start_date", "date", "start"]);
    return d && String(d).slice(0, 10) >= todayISO();
  });
  const calSec = `<div class="section"><h2>Akademik Takvim</h2>${miniList(upcoming.length ? upcoming : calItems, 6)}</div>`;

  const annItems = ann.data?.data || (Array.isArray(ann.data) ? ann.data : []);
  const annSec = `<div class="section"><h2>Duyurular</h2>${miniList(annItems, 6)}</div>`;

  const newsItems = news.data?.data || (Array.isArray(news.data) ? news.data : []);
  const newsSec = `<div class="section"><h2>Haberler</h2>${miniList(newsItems, 6)}</div>`;

  // 4 columns: [profil+kart+sınavlar] · takvim · duyurular · haberler
  html += `<div class="dash">
    <div class="col">${profileSec}${cardSec}${examsSec}</div>
    <div class="col">${calSec}</div>
    <div class="col">${annSec}</div>
    <div class="col">${newsSec}</div>
  </div>`;

  setPanel(html);
}

// ---- shell / router ----
function emptyMsg(m) { return `<div class="empty">${esc(m)}</div>`; }
function rawJSON(d) { return `<pre class="raw">${esc(JSON.stringify(d, null, 2))}</pre>`; }
function setPanel(html) { $("#panel").innerHTML = html; }
function setTitle(t) { $("#panel-title").textContent = t; }

function buildNav() {
  const nav = $("#nav");
  nav.innerHTML = "";
  for (const [group, keys] of NAV) {
    if (group) {
      const g = document.createElement("div");
      g.className = "group"; g.textContent = group; nav.appendChild(g);
    }
    for (const key of keys) {
      const a = document.createElement("a");
      a.textContent = VIEWS[key].label; a.dataset.key = key;
      a.onclick = () => selectView(key);
      nav.appendChild(a);
    }
  }
}

async function selectView(key) {
  document.querySelectorAll(".sidebar a").forEach((a) =>
    a.classList.toggle("active", a.dataset.key === key));
  const v = VIEWS[key];
  setTitle(v.label);
  if (v.special) { renderSummary(); return; }

  setPanel(`<div class="loading">Yükleniyor…</div>`);
  const r = await api("/api/" + v.endpoint);
  if (r.status === 401) { showLogin(); return; }
  if (!r.ok) { setPanel(`<div class="err-box">Hata: ${esc(r.data?.error || r.status)}</div>`); return; }
  try { setPanel(v.render(r.data)); }
  catch (e) { setPanel(rawJSON(r.data)); }
}

// ---- theme ----
function systemTheme() {
  try { return matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"; }
  catch (_) { return "light"; }
}
function currentTheme() {
  return document.documentElement.getAttribute("data-theme") || systemTheme();
}
function applyThemeAttr(t) {
  if (t) document.documentElement.setAttribute("data-theme", t);
  else document.documentElement.removeAttribute("data-theme");
}
function updateThemeBtn() {
  const b = $("#theme");
  if (b) b.textContent = currentTheme() === "dark" ? "☀ Açık" : "☾ Koyu";
}
function initTheme() {
  try {
    const saved = localStorage.getItem("theme");
    if (saved === "dark" || saved === "light") applyThemeAttr(saved);
  } catch (_) {}
  updateThemeBtn();
}
function toggleTheme() {
  const next = currentTheme() === "dark" ? "light" : "dark";
  applyThemeAttr(next);
  try { localStorage.setItem("theme", next); } catch (_) {}
  updateThemeBtn();
}

// ---- auth ----
function showLogin() { $("#app").hidden = true; $("#login").hidden = false; }
function showApp() { $("#login").hidden = true; $("#app").hidden = false; }

async function boot() {
  buildNav();
  const st = await api("/api/status");
  if (st.data?.loggedIn) { showApp(); selectView("summary"); }
  else { showLogin(); }
}

$("#login-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const errEl = $("#login-err"); errEl.hidden = true;
  const r = await api("/api/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: $("#username").value, password: $("#password").value }),
  });
  if (r.ok) { $("#password").value = ""; showApp(); selectView("summary"); }
  else { errEl.textContent = r.data?.error || "Giriş başarısız."; errEl.hidden = false; }
});

$("#logout").addEventListener("click", async () => {
  await api("/api/logout", { method: "POST" });
  showLogin();
});

$("#theme").addEventListener("click", toggleTheme);

initTheme();
boot();
