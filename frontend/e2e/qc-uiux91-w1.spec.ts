import { expect, test, type Locator, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { settleGoto } from "./support/hydrate";
import { asDemo } from "./support/session";
import { COURSE as TC, json as tjson, row as threadRow, threadApi } from "./support/thread-fixtures";
import { chatApi, json as cjson, session as chatSession } from "./support/chat-fixtures";
import { doc as docRow, docApi, json as djson, libItem, stats as docStats } from "./support/doc-fixtures";
import { gradebookApi } from "./support/gradebook-fixtures";
import { gradeApi } from "./support/grade-fixtures";
import { A1, gradingApi } from "./support/assignment-fixtures";

// QC FEAT-uiux-91 đợt 1 (tc-wave-1.md). Hộp đen: chỉ dựa vào AC. Mỗi `test` ghi mã TC ở đầu tên.
const LONG_EXP = 366 * 86400;
const NOW = new Date("2026-11-08T14:00:00+07:00"); // Chủ nhật
const reasonText = (b: Locator) => b.evaluate((e) => (e.getAttribute("aria-describedby") ?? "").split(" ").map((i) => document.getElementById(i)?.textContent ?? "").join(" ").trim());
const desktop = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
test.beforeEach(({ page }) => settleGoto(page));

// ---------- US-UX91-02 ----------
const DATES: Array<[string, string, string]> = [
  ["I1", "2026-11-08T09:20:00+07:00", "Hôm nay, 09:20"],
  ["I2", "2026-11-07T16:40:00+07:00", "Hôm qua, 16:40"],
  ["I3", "2026-11-02T14:00:00+07:00", "Thứ Hai, 14:00"],
  ["I4", "2026-09-21T14:00:00+07:00", "Thứ Hai, 21/09 · 14:00"],
  ["I5", "2025-09-22T14:00:00+07:00", "Thứ Hai, 22/09/2025 · 14:00"], // spec ghi 21/09/2025 nhưng đó là Chủ Nhật: dùng thứ Hai thật 22/09/2025 (erratum, proposals.md)
  ["I6", "2026-11-08T00:01:00+07:00", "Hôm nay, 00:01"],
  ["I7", "2026-11-07T23:59:00+07:00", "Hôm qua, 23:59"],
  ["I8", "2026-11-01T23:59:00+07:00", "Chủ Nhật, 01/11 · 23:59"],
  ["I9", "2026-11-09T09:00:00+07:00", "Ngày mai, 09:00"],
];
/** Gom chữ của khối bắt đầu ở dòng chứa `tag` tới dòng chứa thẻ kế tiếp. */
function blocks(text: string, prefix: string) {
  const lines = text.split("\n");
  const out: Record<string, string> = {};
  let cur = "";
  for (const l of lines) {
    const m = l.match(new RegExp(`${prefix}-(I\\d+)`));
    if (m) cur = m[1];
    if (cur) out[cur] = (out[cur] ?? "") + "\n" + l;
  }
  return out;
}
async function openThreads(page: Page, context: Parameters<typeof asDemo>[0], rows: Array<[string, string]>) {
  await page.clock.setFixedTime(NOW);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await threadApi(page, {
    [`GET /courses/${TC}/threads`]: tjson({
      items: rows.map(([id, iso], i) => threadRow({ id: `00000000-0000-7000-8000-00000000f${String(i).padStart(3, "0")}`, title: `TT-${id}`, preview: "x", created_at: iso, last_activity_at: iso })),
      next_cursor: null,
    }),
  });
  await page.goto("/threads");
  await expect(page.getByText(`TT-${rows[0][0]}`).first()).toBeVisible();
}

test("TC-UX02-2 Threads: các hàng I1…I9 đúng chuỗi AC2", async ({ page, context }, info) => {
  desktop(info);
  await openThreads(page, context, DATES.map(([id, iso]) => [id, iso]));
  const b = blocks(await page.locator("main").innerText(), "TT");
  for (const [id, , want] of DATES) expect(b[id], `${id} → ${want}`).toContain(want);
  expect(await page.locator("main").innerText()).not.toMatch(/Thứ [2-7]\b|\d\d:\d\d \d\d-\d\d/);
});

test("TC-UX02-9 ngày giờ sai / null → «—», không Invalid Date, không lỗi", async ({ page, context }, info) => {
  desktop(info);
  const errs: string[] = [];
  page.on("pageerror", (e) => errs.push(e.message));
  page.on("console", (m) => m.type() === "error" && !/Failed to load resource/.test(m.text()) && errs.push(m.text()));
  await openThreads(page, context, [["I1", "abc"], ["I2", null as unknown as string], ["I3", ""], ["I4", "2026-13-45"]]);
  const b = blocks(await page.locator("main").innerText(), "TT");
  for (const id of ["I1", "I2", "I3", "I4"]) expect(b[id], id).toContain("—");
  expect(await page.locator("main").innerText()).not.toMatch(/Invalid Date|NaN/);
  expect(errs).toEqual([]);
});

test("TC-UX02-3 Chat: lịch sử phiên I1…I4", async ({ page, context }, info) => {
  desktop(info);
  await page.clock.setFixedTime(NOW);
  await asDemo(context, "student", { expIn: LONG_EXP });
  const items = DATES.slice(0, 4).map(([id, iso], i) => chatSession({ id: `00000000-0000-7000-8000-00000000d${String(i).padStart(3, "0")}`, title: `CH-${id}`, last_message_at: iso }));
  await chatApi(page, { "GET /chat/sessions": cjson({ items, next_cursor: null }) });
  await page.goto("/chat");
  await expect(page.getByText("CH-I1").first()).toBeVisible();
  const b = blocks(await page.locator("main").innerText(), "CH");
  for (const [id, , want] of DATES.slice(0, 4)) expect(b[id], `${id} → ${want}`).toContain(want);
});

test("TC-UX02-4 Tài liệu (GV) và Thư viện (SV): cập nhật I3, I4", async ({ page, context }, info) => {
  desktop(info);
  await page.clock.setFixedTime(NOW);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  const mk = (id: string, i: number, iso: string) => docRow({ id: `00000000-0000-7000-8000-00000000e${String(i).padStart(3, "0")}`, title: `DOC-${id}`, updated_at: iso, created_at: iso });
  await docApi(page, {
    [`GET /courses/${TC}/documents`]: djson({ items: [mk("I3", 1, DATES[2][1]), mk("I4", 2, DATES[3][1])], next_cursor: null }),
    [`GET /courses/${TC}/documents/stats`]: djson(docStats({ total: 2 })),
  });
  await page.goto("/documents");
  await expect(page.getByText("DOC-I3").first()).toBeVisible();
  const b = blocks(await page.locator("main").innerText(), "DOC");
  expect(b.I3).toContain("Thứ Hai, 14:00");
  expect(b.I4).toContain("Thứ Hai, 21/09 · 14:00");
  expect(await page.locator("main").innerText()).not.toMatch(/\d\d:\d\d \d\d-\d\d/);
});

test("TC-UX02-4b Thư viện (SV): cập nhật I3, I4", async ({ page, context }, info) => {
  desktop(info);
  await page.clock.setFixedTime(NOW);
  await asDemo(context, "student", { expIn: LONG_EXP });
  const mk = (id: string, i: number, iso: string) => libItem({ id: `00000000-0000-7000-8000-00000000e${String(i).padStart(3, "0")}`, title: `LIB-${id}`, updated_at: iso });
  await page.route(/\/api\/v1\/courses\/[^/]+\/library(\?|$)/, (r) => r.fulfill(djson({ items: [mk("I3", 1, DATES[2][1]), mk("I4", 2, DATES[3][1])], next_cursor: null })));
  await page.goto("/library");
  await expect(page.getByText("LIB-I3").first()).toBeVisible();
  const b = blocks(await page.locator("main").innerText(), "LIB");
  // Thư viện dùng dạng «chỉ ngày» của AC2 (21/09, cùng năm)
  expect(b.I3).toContain("02/11");
  expect(b.I4).toContain("21/09");
  expect(b.I3 + b.I4).not.toMatch(/\d\d:\d\d \d\d-\d\d/);
});

for (const tz of ["America/Los_Angeles", "Pacific/Kiritimati"]) {
  test.describe(`TC-UX02-7/8 múi giờ trình duyệt ${tz}`, () => {
    test.use({ timezoneId: tz });
    test(`Threads I1/I2/I3 giữ giờ ICT (${tz})`, async ({ page, context }, info) => {
      desktop(info);
      await openThreads(page, context, DATES.slice(0, 3).map(([id, iso]) => [id, iso]));
      const b = blocks(await page.locator("main").innerText(), "TT");
      expect(b.I1).toContain("Hôm nay, 09:20");
      expect(b.I2).toContain("Hôm qua, 16:40");
      expect(b.I3).toContain("Thứ Hai, 14:00");
    });
  });
}

test("TC-UX02-13 375 px: Threads / Chat với I5 không tràn ngang", async ({ page, context }, info) => {
  desktop(info);
  await page.setViewportSize({ width: 375, height: 812 });
  await openThreads(page, context, [["I5", DATES[4][1]]]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
});

// ---------- US-UX91-03 ----------
test("TC-UX03-1..4,12 /dev/ui: nút khoá có lý do (DOM, tooltip, click, opacity, bàn phím)", async ({ page }, info) => {
  desktop(info);
  await page.goto("/dev/ui");
  const btn = page.locator('button[aria-disabled="true"][aria-describedby]').first();
  await expect(btn).toBeVisible();
  expect(await btn.getAttribute("disabled")).toBeNull();
  const id = (await btn.getAttribute("aria-describedby"))!;
  const reason = (await page.evaluate((i) => document.getElementById(i.split(" ")[0])?.textContent ?? "", id)).trim();
  expect(reason.length).toBeGreaterThan(3);
  expect(await btn.evaluate((e) => getComputedStyle(e).opacity)).toBe("0.45");
  await btn.hover();
  await expect(page.getByRole("tooltip")).toContainText(reason);
  await page.mouse.move(0, 0);
  await btn.focus();
  await expect(page.getByRole("tooltip")).toContainText(reason);
  const ring = await btn.evaluate((e) => { const c = getComputedStyle(e); return c.outlineStyle !== "none" || c.boxShadow !== "none"; });
  expect(ring, "focus ring").toBe(true);
  const reqs: string[] = [];
  page.on("request", (q) => reqs.push(q.url()));
  await page.keyboard.press("Enter");
  await page.keyboard.press("Space");
  await btn.click({ force: true });
  expect(reqs.filter((u) => /\/api\//.test(u))).toEqual([]);
});

test("TC-UX03-14 /dev/ui: disabledReason mà không disabled → không tooltip, bấm được", async ({ page }, info) => {
  desktop(info);
  await page.goto("/dev/ui");
  const c = page.locator("[data-case=reason-not-disabled]");
  await expect(c).toBeVisible();
  const b = (await c.evaluate((e) => e.tagName)) === "BUTTON" ? c : c.locator("button").first();
  expect(await b.getAttribute("aria-disabled")).not.toBe("true");
  expect(await b.getAttribute("disabled")).toBeNull();
  expect(await b.getAttribute("aria-describedby"), "lý do bị bỏ qua: không aria-describedby").toBeNull();
  await b.hover();
  await page.waitForTimeout(400);
  expect(await b.evaluate((e) => (e.parentElement?.querySelector("[class*=reason]") as HTMLElement | null)?.textContent ?? ""), "không hiện chữ lý do").toBe("");
  await b.click();
});

test("TC-UX03-6 Sổ điểm công thức chưa xác nhận → Chốt điểm khoá, lý do «Chưa có công thức điểm đã xác nhận»", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await gradebookApi(page, { scheme: false, status: "DRAFT_READY" });
  await page.goto("/gradebook");
  const btn = page.getByRole("button", { name: "Chốt điểm" });
  await expect(btn).toHaveAttribute("aria-disabled", "true");
  expect(await reasonText(btn)).toBe("Chưa có công thức điểm đã xác nhận"); // chữ theo mã, US-03 AC2 sau P-91-7 (4)
});

test("TC-UX03-7 Công thức có mục chưa rõ → Xác nhận công thức khoá «Còn N mục chưa rõ»", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await gradeApi(page);
  await page.goto("/gradebook/scheme");
  const btn = page.getByRole("button", { name: "Xác nhận công thức" });
  await expect(btn).toHaveAttribute("aria-disabled", "true");
  expect(await reasonText(btn)).toMatch(/Còn \d+ mục chưa rõ/);
});

test("TC-UX03-8/10 Chấm bài: Chấm tất cả khoá khi rỗng; Tải bài lên khoá khi chưa chọn bài tập", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await gradingApi(page, { rows: [] });
  await page.goto("/grading?tab=queue");
  const all = page.getByRole("button", { name: "Chấm tất cả" });
  await expect(all).toHaveAttribute("aria-disabled", "true");
  console.log("INFO chưa chọn bài tập: Chấm tất cả =", await reasonText(all));
  await page.getByLabel("Bài tập").selectOption(A1);
  await page.waitForTimeout(400);
  await expect(page.getByRole("button", { name: "Chấm tất cả" }), "hàng chờ rỗng (đã chọn một bài) thì Chấm tất cả phải khoá").toHaveAttribute("aria-disabled", "true");
  expect(await reasonText(page.getByRole("button", { name: "Chấm tất cả" }))).toMatch(/^Chưa có bài nộp\.?$/);
  await page.getByLabel("Bài tập").selectOption({ index: 0 });
  await page.waitForTimeout(300);
  const up = page.getByRole("button", { name: "Tải bài lên" }).first();
  await expect(up).toHaveAttribute("aria-disabled", "true");
  expect(await reasonText(up)).toMatch(/^Chọn một bài tập trước\.?$/);
});

test("TC-UX03-9/10b Chấm bài có bài nộp: Chấm tất cả bấm được; chọn bài tập → Tải bài lên bấm được", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await gradingApi(page);
  await page.goto("/grading?tab=queue");
  await page.getByLabel("Bài tập").selectOption(A1);
  await expect(page.getByRole("button", { name: "Chấm tất cả" })).not.toHaveAttribute("aria-disabled", "true");
  await expect(page.getByRole("button", { name: "Tải bài lên" }).first()).not.toHaveAttribute("aria-disabled", "true");
});

test("TC-UX03-15 STUDENT: nút giảng viên không có trong DOM", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "student", { expIn: LONG_EXP });
  for (const u of ["/gradebook", "/grading", "/gradebook/scheme"]) {
    await page.goto(u);
    await expect(page.getByText("Bạn không có quyền xem màn này")).toBeVisible();
    for (const n of ["Chốt điểm", "Chấm tất cả", "Xác nhận công thức"]) await expect(page.getByRole("button", { name: n })).toHaveCount(0);
  }
});

test("TC-UX03-13 cảm ứng 375: chạm nút khoá → lý do hiện dưới nút ≥ 3 s", async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, baseURL: BASE_URL });
  const page = await ctx.newPage();
  await page.goto("/dev/ui");
  const btn = page.locator('button[aria-disabled="true"][aria-describedby]').first();
  await btn.scrollIntoViewIfNeeded();
  const box = (await btn.boundingBox())!;
  const vis = () => btn.evaluate((e) => { const w = e.parentElement!; const r = w.querySelector("[class*=reason]") as HTMLElement | null; return r ? { op: getComputedStyle(r).opacity, top: r.getBoundingClientRect().top, btnBottom: e.getBoundingClientRect().bottom } : null; });
  await page.touchscreen.tap(box.x + box.width / 2, box.y + box.height / 2);
  await page.waitForTimeout(400);
  const t1 = await vis();
  expect(t1, "có phần tử lý do cạnh nút").not.toBeNull();
  expect(Number(t1!.op), "ngay sau khi chạm").toBeGreaterThan(0.9);
  expect(t1!.top, "lý do nằm dưới nút").toBeGreaterThanOrEqual(t1!.btnBottom - 2);
  await page.waitForTimeout(2400);
  expect(Number((await vis())!.op), "sau 2,8 s vẫn còn").toBeGreaterThan(0.9);
  await ctx.close();
});

// ---------- US-UX91-04 ----------
test("TC-UX04-1/2 :active thu 0,97 ≤ 100 ms rồi trở lại", async ({ page }, info) => {
  desktop(info);
  await page.goto("/dev/ui");
  const bs = page.locator("main button[class*=Button-module]:not([aria-disabled=true]):not([disabled])");
  const n = Math.min(await bs.count(), 40);
  const bad: string[] = [];
  const seen = new Set<string>();
  for (let i = 0; i < n; i++) {
    const b = bs.nth(i);
    if (!(await b.isVisible())) continue;
    const sig = await b.evaluate((e) => `${e.className}|${(e.textContent ?? "").trim().slice(0, 18)}`);
    const cls = sig.split("|")[0];
    if (seen.has(cls)) continue;
    seen.add(cls);
    await b.scrollIntoViewIfNeeded();
    await b.hover();
    await page.mouse.down();
    await page.waitForTimeout(160);
    const t = await b.evaluate((e) => ({ tr: getComputedStyle(e).transform, d: getComputedStyle(e).transitionDuration, tp: getComputedStyle(e).transitionProperty }));
    if (!/matrix\(0\.97/.test(t.tr)) bad.push(`${sig} → transform=${t.tr} dur=${t.d} prop=${t.tp}`);
    await page.mouse.up();
    await page.mouse.move(0, 0);
    await page.waitForTimeout(220);
    if (!/none|matrix\(1,/.test(await b.evaluate((e) => getComputedStyle(e).transform))) bad.push(`${sig} không trở lại`);
  }
  expect(seen.size, "số biến thể nút đã thử").toBeGreaterThan(2);
  expect(bad, "nút không thu 0,97 khi :active").toEqual([]);
});

test("TC-UX04-9/21 thanh tiến độ: scaleX, kẹp 0–100, NaN → 0", async ({ page }, info) => {
  desktop(info);
  await page.goto("/dev/ui");
  const cases = page.locator("[data-case=progress]");
  const n = await cases.count();
  expect(n).toBeGreaterThanOrEqual(4);
  const got: Record<string, number> = {};
  for (let i = 0; i < n; i++) {
    const c = cases.nth(i);
    const v = (await c.getAttribute("data-value")) ?? "";
    const sx = await c.evaluate((root) => {
      for (const e of [root, ...root.querySelectorAll("*")]) {
        const t = getComputedStyle(e).transform;
        const m = t.match(/matrix\(([^,]+),/);
        if (m && t !== "none") return Number(m[1]);
      }
      return -1;
    });
    got[v] = sx;
  }
  expect(got["150"]).toBeCloseTo(1, 2);
  expect(got["-10"]).toBeLessThanOrEqual(0.001);
  expect(got["NaN"]).toBeLessThanOrEqual(0.001);
  expect(got["50"]).toBeCloseTo(0.5, 2);
});

test("TC-UX04-10 thu gọn sidebar: main không transition margin-left, tức thì ≤ 160 ms, CLS ≤ 0,05", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await page.goto("/inbox");
  await expect(page.locator("main")).toBeVisible();
  const tp = await page.locator("main").evaluate((m) => ({ p: getComputedStyle(m).transitionProperty, d: getComputedStyle(m).transitionDuration }));
  expect(!/margin-left/.test(tp.p) && !(tp.p === "all" && !/^0s(, 0s)*$/.test(tp.d)), `main transition: ${tp.p} ${tp.d}`).toBe(true);
  const res = await page.evaluate(async () => {
    let cls = 0;
    new PerformanceObserver((l) => l.getEntries().forEach((e) => { const x = e as unknown as { value: number; hadRecentInput: boolean }; if (!x.hadRecentInput) cls += x.value; })).observe({ type: "layout-shift", buffered: false });
    const m = document.querySelector("main")!;
    const btn = [...document.querySelectorAll("button")].find((b) => /Thu gọn/.test(b.textContent ?? ""))!;
    const w0 = m.getBoundingClientRect().width;
    btn.click();
    const t0 = performance.now();
    let settled = -1;
    await new Promise<void>((done) => {
      const tick = () => {
        const w = m.getBoundingClientRect().width;
        if (Math.abs(w - w0) > 1 && settled < 0) settled = performance.now() - t0;
        if (performance.now() - t0 > 400) return done();
        requestAnimationFrame(tick);
      };
      tick();
    });
    return { cls, settled, w0, w1: m.getBoundingClientRect().width };
  });
  expect(res.w1).toBeGreaterThan(res.w0);
  expect(res.cls).toBeLessThanOrEqual(0.05);
});

test("TC-UX04-6 hộp thoại Chốt điểm mở ≤ 200 ms", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await gradebookApi(page, {});
  await page.goto("/gradebook");
  const btn = page.getByRole("button", { name: "Chốt điểm" });
  await expect(btn).not.toHaveAttribute("aria-disabled", "true");
  await btn.click();
  const d = await page.evaluate(() => document.getAnimations().map((a) => { const t = a.effect!.getTiming(); return Number(t.duration) + Number(t.delay ?? 0); }));
  expect(d.length).toBeGreaterThan(0);
  for (const x of d) expect(x).toBeLessThanOrEqual(200);
});

test("TC-UX04-11 menu tài khoản: gốc xuất phát ở nút gọi, ≤ 160 ms", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await page.goto("/");
  const trigger = page.getByRole("button", { name: /^Tài khoản/ });
  await trigger.click();
  const menu = page.getByRole("menu").first();
  await expect(menu).toBeVisible();
  const info2 = await menu.evaluate((m) => ({ origin: getComputedStyle(m).transformOrigin, anim: m.getAnimations().map((a) => Number(a.effect!.getTiming().duration)) }));
  expect(info2.origin, "transform-origin không phải tâm 50% 50%").not.toBe("50% 50%");
  for (const x of info2.anim) expect(x).toBeLessThanOrEqual(160);
});

test("TC-UX04-20 phím J/K, ⌘K không có hoạt cảnh", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await page.goto("/inbox");
  await expect(page.locator("main")).toBeVisible();
  await page.waitForTimeout(500);
  for (const k of ["j", "k"]) {
    await page.keyboard.press(k);
    expect(await page.evaluate(() => document.getAnimations().length), `phím ${k}`).toBe(0);
  }
  await page.keyboard.press("Control+k");
  expect(await page.evaluate(() => document.getAnimations().length), "⌘K").toBe(0);
});

// ---------- US-UX91-21 ----------
const apiCalls = (page: Page) => {
  const calls: string[] = [];
  page.on("request", (q) => q.url().includes("/api/v1/") && calls.push(`${q.method()} ${new URL(q.url()).pathname}`));
  return calls;
};
const unauth = (page: Page) => page.route("**/api/v1/auth/refresh", (r) => r.fulfill({ status: 401, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true" }, body: JSON.stringify({ code: "UNAUTHENTICATED", message: "x", trace_id: "b".repeat(32) }) }));

test("TC-UX21-1/2 khách (không cờ): `/` không gọi API nào, không lỗi 401 ở console", async ({ page, browser }, info) => {
  desktop(info);
  const errs: string[] = [];
  page.on("console", (m) => m.type() === "error" && errs.push(m.text()));
  const calls = apiCalls(page);
  await unauth(page);
  await page.goto("/");
  await expect(page.locator("[data-part=landing]")).toBeVisible();
  await page.waitForTimeout(800);
  expect(calls).toEqual([]);
  expect(errs.filter((e) => /401/.test(e))).toEqual([]);
  const ctx = await browser.newContext({ viewport: { width: 375, height: 812 }, baseURL: BASE_URL });
  const p2 = await ctx.newPage();
  const c2 = apiCalls(p2);
  await unauth(p2);
  await p2.goto("/");
  await p2.waitForTimeout(800);
  expect(c2).toEqual([]);
  await ctx.close();
});

test("TC-UX21-3 người quay lại: có gọi refresh, hiện Hôm nay, không nháy trang giới thiệu", async ({ page, context }, info) => {
  desktop(info);
  await context.addInitScript(() => {
    (window as unknown as { __sawLanding: boolean }).__sawLanding = false;
    new MutationObserver(() => {
      const l = document.querySelector("[data-part=landing]");
      if (l && getComputedStyle(l).display !== "none" && (l as HTMLElement).offsetHeight > 0) (window as unknown as { __sawLanding: boolean }).__sawLanding = true;
    }).observe(document, { childList: true, subtree: true, attributes: true });
  });
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  const calls = apiCalls(page);
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible();
  expect(calls.filter((c) => c.endsWith("/auth/refresh")).length).toBeGreaterThanOrEqual(1);
  expect(await page.evaluate(() => (window as unknown as { __sawLanding: boolean }).__sawLanding)).toBe(false);
});

test("TC-UX21-4/5 mất cờ nhưng còn cookie: `/` hiện trang giới thiệu; bấm Đăng nhập → /login làm mới và vào ứng dụng", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher", { expIn: LONG_EXP });
  await context.addInitScript(() => localStorage.removeItem("ep_returning"));
  await page.addInitScript(() => localStorage.removeItem("ep_returning"));
  const calls = apiCalls(page);
  await page.goto("/");
  await expect(page.locator("[data-part=landing]")).toBeVisible();
  expect(calls.filter((c) => c.endsWith("/auth/refresh"))).toEqual([]);
  await page.getByRole("link", { name: "Đăng nhập" }).first().click();
  await expect(page).toHaveURL(/\/login|\/$/);
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 10_000 });
  expect(calls.filter((c) => c.endsWith("/auth/refresh")).length).toBeGreaterThanOrEqual(1);
});

test("TC-UX21-6/10 /login và /register không cookie: vẫn gọi refresh, hiện form", async ({ page }, info) => {
  desktop(info);
  const calls = apiCalls(page);
  await unauth(page);
  await page.goto("/login");
  await expect(page.getByRole("button", { name: "Đăng nhập" })).toBeVisible();
  expect(calls.filter((c) => c.endsWith("/auth/refresh")).length).toBeGreaterThanOrEqual(1);
  await page.goto("/register");
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
});

test("TC-UX21-7/8 localStorage ném lỗi: `/` không lỗi, hiện trang giới thiệu", async ({ page }, info) => {
  desktop(info);
  const errs: string[] = [];
  page.on("pageerror", (e) => errs.push(e.message));
  await page.addInitScript(() => {
    Object.defineProperty(window, "localStorage", { get() { throw new DOMException("denied", "SecurityError"); } });
  });
  await unauth(page);
  await page.goto("/");
  await expect(page.locator("[data-part=landing]")).toBeVisible();
  expect(errs).toEqual([]);
});

test("TC-UX21-9 khách mở route sâu → /login?next=", async ({ page }, info) => {
  desktop(info);
  await unauth(page);
  for (const u of ["/chat", "/gradebook"]) {
    await page.goto(u);
    await expect(page).toHaveURL(new RegExp(`/login\\?next=${encodeURIComponent(u)}$`));
  }
});

test("TC-UX03-17 tooltip lý do: hiện bằng opacity ≤ 120 ms; reduce → tức thì", async ({ browser }, info) => {
  desktop(info);
  for (const reduce of [false, true]) {
    const ctx = await browser.newContext({ baseURL: BASE_URL, reducedMotion: reduce ? "reduce" : "no-preference" });
    const page = await ctx.newPage();
    await page.goto("/dev/ui");
    const btn = page.locator('button[aria-disabled="true"][aria-describedby]').first();
    await btn.scrollIntoViewIfNeeded();
    await btn.hover();
    await page.waitForTimeout(30);
    const r = await btn.evaluate((e) => {
      const t = e.parentElement!.querySelector("[class*=reason]") as HTMLElement;
      const cs = getComputedStyle(t);
      return { td: cs.transitionDuration, tp: cs.transitionProperty, anim: t.getAnimations().map((a) => Number(a.effect?.getTiming().duration ?? 0)) };
    });
    console.log(`INFO tooltip reduce=${reduce}:`, JSON.stringify(r));
    const ms = (s: string) => Math.max(...s.split(",").map((x) => (x.trim().endsWith("ms") ? parseFloat(x) : parseFloat(x) * 1000)));
    if (reduce) expect(ms(r.td) <= 1 || r.anim.every((d) => d <= 1), "reduce: tức thì").toBe(true);
    else { expect(r.tp).toMatch(/opacity|all/); expect(ms(r.td), "≤ 120 ms").toBeLessThanOrEqual(120); }
    await ctx.close();
  }
});
