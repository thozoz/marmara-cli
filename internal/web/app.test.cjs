const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
const path = require("node:path");

// Load the real renderers/router without running the browser's event wiring.
const source = fs.readFileSync(path.join(__dirname, "static/app.js"), "utf8")
  .split('$("#login-form").addEventListener')[0];
function app() {
  const elements = new Map();
  const timers = new Map();
  let timerId = 0;
  const context = vm.createContext({
    Intl, Date,
    document: {
      querySelector(selector) {
        if (!elements.has(selector)) elements.set(selector, { innerHTML: "", hidden: false });
        return elements.get(selector);
      },
      querySelectorAll: () => [],
    },
    setInterval(fn) { timers.set(++timerId, fn); return timerId; },
    clearInterval(id) { timers.delete(id); },
  });
  vm.runInContext(source, context);
  return { context, elements, timers };
}
const lesson = (start, end, name = "Ders") => ({
  Gun: 3, Baslangic: start, Bitis: end, DersKodu: "MAT2019", DersAdi: name,
});
const schedule = { OgrenciDersProgramListesi: [lesson("15:00", "15:50"), lesson("16:00", "16:50")] };

test("lesson starts inclusively, ends exclusively; break sits between lessons", () => {
  const { context: c } = app();
  const render = (time) => c.renderTodaySchedule(schedule, new Date(`2026-09-30T${time}+03:00`));
  assert.match(render("15:00:00"), /current-lesson.*aria-current="time"/);
  assert.match(render("15:49:59"), /Bitmesine 1 dk kaldı/);
  const pause = render("15:50:00");
  assert.doesNotMatch(pause, /aria-current="time"/);
  assert.match(pause, /Sıradaki derse 10 dk kaldı/);
  assert.ok(pause.indexOf('class="break-row"') > pause.indexOf("15:00–15:50"));
  assert.ok(pause.indexOf('class="break-row"') < pause.indexOf("16:00–16:50"));
  assert.doesNotMatch(render("14:59:00"), /Teneffüs/);
  assert.doesNotMatch(render("16:00:00"), /Teneffüs/);
  assert.doesNotMatch(render("16:50:00"), /Teneffüs|aria-current="time"/);
});

test("Istanbul date changes at 21:00 UTC; Sunday wraps to Monday", () => {
  const { context: c } = app();
  assert.equal(c.scheduleClock(new Date("2026-09-30T20:59:59Z")).weekday, 3);
  assert.equal(c.scheduleClock(new Date("2026-09-30T21:00:00Z")).weekday, 4);
  assert.equal(c.scheduleClock(new Date("2026-09-30T21:00:00Z")).minutes, 0);
  const html = c.renderTodaySchedule({ OgrenciDersProgramListesi: [{ ...lesson("08:30", "09:20"), Gun: 1 }] }, new Date("2026-10-04T12:00:00Z"));
  assert.match(html, /Bugün ders yok/);
  assert.match(html, /Sıradaki ders günü: Pazartesi/);
});

test("overlapping lessons suppress breaks; invalid slots cannot create breaks", () => {
  const { context: c } = app();
  assert.equal(c.findScheduleBreak([lesson("15:00", "15:50"), lesson("15:40", "16:20"), lesson("16:30", "17:00")], 15 * 60 + 55), null);
  assert.equal(c.findScheduleBreak([lesson("bad", "15:50"), lesson("16:00", "16:50")], 15 * 60 + 55), null);
  assert.equal(c.scheduleMinutes("24:00"), null);
  assert.equal(c.scheduleMinutes("15:60"), null);
  assert.equal(c.sortScheduleEntries([lesson("10:00", "10:50"), lesson("9:00", "9:50")])[0].Baslangic, "9:00");
});

test("room codes and escaped API text are preserved", () => {
  const { context: c } = app();
  assert.equal(c.summaryRoom({ Derslik: "Külliye-T2-Elektronik Lab.-RTE.T2.113", DerslikAdi: "(Elektronik Lab.)" }), "RTE.T2.113 · Elektronik Lab.");
  assert.equal(c.summaryRoom({ Derslik: "T4-Z07" }), "T4-Z07");
  const html = c.scheduleRows([lesson("15:00", "15:50", '<img src=x onerror="alert(1)">')], 15 * 60);
  assert.doesNotMatch(html, /<img/);
  assert.match(html, /&lt;img/);
});

test("a delayed summary cannot replace a newer view or restart its timer", async () => {
  const { context: c, elements, timers } = app();
  let resolve;
  const pending = new Promise((r) => { resolve = r; });
  c.api = () => pending;
  const oldView = c.selectView("summary");
  c.api = async () => ({ ok: true, status: 200, data: {} });
  await c.selectView("grades");
  const newerHTML = elements.get("#panel").innerHTML;
  resolve({ ok: true, status: 200, data: {} });
  await oldView;
  assert.equal(elements.get("#panel").innerHTML, newerHTML);
  assert.equal(elements.get("#panel-title").textContent, "Notlar");
  assert.equal(timers.size, 0);
});

test("summary clock updates stop on navigation and login screen", async () => {
  const { context: c, timers } = app();
  c.api = async () => ({ ok: true, status: 200, data: {} });
  await c.selectView("summary");
  assert.equal(timers.size, 1);
  await c.selectView("schedule");
  assert.equal(timers.size, 0);
  await c.selectView("summary");
  c.showLogin();
  assert.equal(timers.size, 0);
});

test("profile renders photo URL, advisor and aliases; escapes text", () => {
  const { context: c } = app();
  const html = c.renderProfile({
    Ad: "Ayşe", Soyad: "Yılmaz", OgrenciNo: "111111111",
    Bolum: "Bilgisayar Mühendisliği",
    Mail: "ayse@marmara.edu.tr",
    DanismanEMail: "hoca@marmara.edu.tr",
    ResimPath: "111111111.jpg",
  });
  assert.match(html, /class="avatar"/);
  assert.match(html, /https:\/\/bys\.marmara\.edu\.tr\/upload\/ogrfoto\/111111111\.jpg/);
  assert.match(html, /Ayşe Yılmaz/);
  assert.match(html, /hoca@marmara\.edu\.tr/);
  assert.match(html, /Bilgisayar Mühendisliği/);
  const injected = c.renderProfile({ AdSoyad: '<img src=x onerror="alert(1)">' });
  assert.doesNotMatch(injected, /<img src=x/);
});

test("studentPhotoURL mirrors the Go helper", () => {
  const { context: c } = app();
  assert.equal(c.studentPhotoURL(""), "");
  assert.equal(c.studentPhotoURL("https://cdn.example/f.jpg"), "https://cdn.example/f.jpg");
  assert.equal(c.studentPhotoURL("f.jpg"), "https://bys.marmara.edu.tr/upload/ogrfoto/f.jpg");
  assert.equal(c.studentPhotoURL("upload/ogrfoto/x.jpg"), "https://bys.marmara.edu.tr/upload/ogrfoto/x.jpg");
  assert.equal(c.studentPhotoURL("/v2/upload/ogrfoto/y.jpg"), "https://bys.marmara.edu.tr/v2/upload/ogrfoto/y.jpg");
});

test("cafeteria renders the normalised array from either source", () => {
  const { context: c } = app();
  const day = { date: "2026-10-19", meals: { "Mercimek Çorbası": 110 }, meals_vegan: { Makarna: 270 } };
  const html = c.renderCafeteria([day]);
  assert.match(html, /2026-10-19/);
  assert.match(html, /Vegan/);
  assert.match(c.renderCafeteria([]), /Menü yok/);
});

test("exams render as a sorted, deduplicated, escaped table", () => {
  const { context: c } = app();
  const d = { OgrenciDersSinavListesi: [
    { dersKodu: "BLM1003.2", dersAdi: "Algoritma", sinavTarihi: "10.01.2026, Saturday", sinavSaati: "9:00", derslikKodu: "Z05", sinavCondition: "passed" },
    { dersKodu: "MAT2085.1", dersAdi: "<b>x</b>", sinavTarihi: "16.11.2026, Monday", sinavSaati: "10:30,10:30", derslikKodu: "Z05,Z06", sinavCondition: "future" },
  ] };
  const desc = c.renderExams(d);
  assert.ok(desc.indexOf("MAT2085.1") < desc.indexOf("BLM1003.2"), "newest first by default");
  const asc = c.renderExams(d, true);
  assert.ok(asc.indexOf("BLM1003.2") < asc.indexOf("MAT2085.1"), "soonest first when ascending");
  assert.match(desc, /2026-11-16/);
  assert.match(desc, /Z05, Z06/);
  assert.doesNotMatch(desc, /10:30, 10:30/);
  assert.match(desc, /yapıldı/);
  assert.doesNotMatch(desc, /<b>x<\/b>/);
  assert.match(c.renderExams({ OgrenciDersSinavListesi: [] }), /Sınav kaydı yok/);
});

test("exams page chips filter past/future with counts", () => {
  const { context: c, elements } = app();
  const d = { OgrenciDersSinavListesi: [
    { dersKodu: "OLD1", sinavTarihi: "10.01.2026, Saturday", sinavCondition: "passed" },
    { dersKodu: "NEW1", sinavTarihi: "20.11.2026, Friday", sinavCondition: "future" },
    { dersKodu: "NEW2", sinavTarihi: "16.11.2026, Monday", sinavCondition: "future" },
  ] };
  const page = c.renderExamsPage(d);
  assert.match(page, /Tümü<span class="n">3/);
  assert.match(page, /Yaklaşan<span class="n">2/);
  assert.match(page, /Geçmiş<span class="n">1/);
  c.filterExams("future");
  const fut = elements.get("#exams-list").innerHTML;
  assert.doesNotMatch(fut, /OLD1/);
  assert.ok(fut.indexOf("NEW2") < fut.indexOf("NEW1"), "upcoming soonest first");
  c.filterExams("past");
  assert.match(elements.get("#exams-list").innerHTML, /OLD1/);
  assert.doesNotMatch(elements.get("#exams-list").innerHTML, /NEW/);
});

test("summary upcoming exams: soonest first with days-left hint", () => {
  const { context: c } = app();
  const d = { OgrenciDersSinavListesi: [
    { dersKodu: "B", dersAdi: "Later", sinavTarihi: "20.11.2026, Friday", sinavSaati: "15:00", sinavTuru: "Ara Sınav 1", derslikKodu: "Z05,Z05" },
    { dersKodu: "A", dersAdi: "Tomorrow", sinavTarihi: "12.11.2026, Thursday", sinavSaati: "10:30,10:30", sinavTuru: "Ara Sınav 1", derslikKodu: "Z07" },
  ] };
  const html = c.renderUpcomingExams(d, "2026-11-11");
  assert.ok(html.indexOf("Tomorrow") < html.indexOf("Later"));
  assert.match(html, /yarın/);
  assert.match(html, /9 gün kaldı/);
  assert.match(html, /2 sınav/);
  assert.doesNotMatch(html, /Z05, Z05|10:30, 10:30/);
  assert.match(c.renderUpcomingExams({ OgrenciDersSinavListesi: [] }, "2026-11-11"), /Yaklaşan sınav yok/);
});
