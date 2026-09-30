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
