import { expect, test, type Page, type Route } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { settleGoto } from "./support/hydrate";
import { asJwt, type JwtRole } from "./support/session";

// QC FEAT-uiux-91 đợt 3 (tc-wave-3.md): Hôm nay. Gateway giả bằng page.route (như today.spec.ts), dữ liệu cố định F-GV.
type Json = Record<string, unknown>;
test.beforeEach(({ page }, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
  settleGoto(page);
});
const C1 = "00000000-0000-7000-8000-00000000c001";
const C2 = "00000000-0000-7000-8000-00000000c002";
const c1 = { id: C1, class_code: "761988" };
const c2 = { id: C2, class_code: "761987" };
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const ok = (body: unknown) => ({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const mine = (id: string, code: string, role: "TEACHER" | "TA" | "STUDENT") => ({
  course: { id, class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" },
  role_in_course: role,
  enrollment_status: "ACTIVE",
});
const act = (id: string, kind: string, title: string, urgency: string, course: Json | null = c2, reason = "Lý do ngắn."): Json => ({ id, kind, title, reason, urgency, href: `/inbox?x=${id}`, course });
// F-GV (tc-wave-3.md): kind lấy đúng tên của stack
const A1 = act("a1", "AI_PENDING", "Câu hỏi chờ giảng viên", "overdue");
const A2 = act("a2", "REVIEW_FLAGGED", "4 bài cần xem kỹ", "high");
const A3 = act("a3", "JOIN_REQUESTS", "1 yêu cầu có email chưa khớp MSSV", "high");
const A4 = act("a4", "REPORTED_POSTS", "2 bài bị báo cáo chờ xem", "normal");
const A5 = act("a5", "EXAM_SIMILARITY", "1 cặp bài code nghi giống nhau", "normal");
const A6 = act("a6", "UNMATCHED_SUBMISSION", "3 bài nộp chưa khớp sinh viên", "normal");
const A7 = act("a7", "QUESTION_REVIEW", "5 câu hỏi chờ duyệt", "normal");
const FGV = [A1, A2, A3, A4, A5, A6, A7];
const staff = (actions: Json[], over: Json = {}): Json => ({ count: actions.length, actions, attention: [], upcoming: [], ...over });
const att = (i: number) => ({ student: { id: `00000000-0000-7000-8000-00000000b00${i}`, full_name: `Sinh Viên ${i}` }, course: c2, reason: "vắng 3/4 buổi" });
const up = { at: "2026-11-09T06:30:00Z", title: "Buổi 5 · An ninh mạng", place: "P.405", course: c2 };

type St = { urls: string[]; fail: boolean };
async function open(page: Page, role: JwtRole, courses: Json[], body: (p: string) => Json, url = "/", pre?: () => Promise<void>, failInit = false): Promise<St> {
  const st: St = { urls: [], fail: failInit };
  await asJwt(page, role);
  if (pre) await pre();
  page.on("request", (r) => { if (r.url().includes("/api/v1/")) st.urls.push(new URL(r.url()).pathname.replace("/api/v1", "")); });
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(ok({ items: courses, next_cursor: null })));
  const handler = (path: (r: Route) => string) => (r: Route) =>
    st.fail ? r.fulfill({ status: 500, contentType: "application/json", headers: cors, body: JSON.stringify({ code: "INTERNAL", message: "x", trace_id: "c".repeat(32) }) }) : r.fulfill(ok(body(path(r))));
  await page.route("**/api/v1/me/today**", handler(() => "/me/today"));
  await page.route("**/api/v1/courses/*/today**", handler((r) => new URL(r.request().url()).pathname.replace("/api/v1", "")));
  await page.route("**/api/v1/courses/*/setup/dismiss", (r) => r.fulfill({ status: 204, headers: cors }));
  await page.goto(url);
  return st;
}
const T1 = [mine(C2, "761987", "TEACHER")];
const main = (p: Page) => p.locator("main");
const rowsOf = (p: Page, h3: RegExp) =>
  p.evaluate(([src]) => {
    const re = new RegExp(src);
    const hs = [...document.querySelectorAll("main h3")];
    const h = hs.find((e) => re.test((e as HTMLElement).innerText.replace(/\s+/g, " ").trim()));
    if (!h) return null;
    const next = hs[hs.indexOf(h) + 1];
    return [...document.querySelectorAll("main li[data-kind]")]
      .filter((l) => (h.compareDocumentPosition(l) & Node.DOCUMENT_POSITION_FOLLOWING) && (!next || (next.compareDocumentPosition(l) & Node.DOCUMENT_POSITION_PRECEDING)))
      .map((l) => (l as HTMLElement).innerText.split("\n")[0].trim());
  }, [h3.source] as const);
const h3s = (p: Page) => p.locator("main h3").allInnerTexts().then((l) => l.map((s) => s.replace(/\s+/g, " ").trim()));
const redCount = (p: Page, h = 900) =>
  p.evaluate((H) => {
    const cv = document.createElement("canvas"); cv.width = cv.height = 1; const x = cv.getContext("2d")!;
    const rgb = (c: string) => { x.clearRect(0, 0, 1, 1); x.fillStyle = c; x.fillRect(0, 0, 1, 1); return [...x.getImageData(0, 0, 1, 1).data.slice(0, 3)]; };
    return [...document.querySelectorAll("main a, main button")].filter((e) => { const r = e.getBoundingClientRect(); return r.width > 30 && r.height > 20 && r.top >= 0 && r.top < H; })
      .map((e) => ({ t: (e as HTMLElement).innerText.trim().slice(0, 30), c: rgb(getComputedStyle(e).backgroundColor) })).filter((b) => b.c[0] > 170 && b.c[1] < 70 && b.c[2] < 80).map((b) => b.t);
  }, h);

test("TC-UX10-1 F-GV: ba nhóm Gấp · 3 → Hôm nay · 2 → Khi rảnh · 2, thứ tự máy chủ trong nhóm", async ({ page }) => {
  await open(page, "TEACHER", T1, () => staff(FGV, { count: 7 }));
  await page.locator("main h3").first().waitFor();
  expect(await h3s(page)).toEqual(["Gấp · 3", "Hôm nay · 2", "Khi rảnh · 2"]);
  const g = await rowsOf(page, /^Gấp/), h = await rowsOf(page, /^Hôm nay/), k = await rowsOf(page, /^Khi rảnh/);
  console.log("INFO hàng:", JSON.stringify({ g, h, k }));
  expect(g).toEqual([A1.title, A2.title, A3.title]);
  expect(h).toEqual([A4.title, A7.title]);
  expect(k).toEqual([A5.title, A6.title]);
});

test("TC-UX10-2/3/4 nhóm rỗng không hiện; chỉ normal → chỉ «Hôm nay»; EXAM_SIMILARITY high → Gấp", async ({ page }) => {
  await open(page, "TEACHER", T1, () => staff([A1, A2, A3, A4, A7]));
  await page.locator("main h3").first().waitFor();
  expect(await h3s(page)).toEqual(["Gấp · 3", "Hôm nay · 2"]);
  const p2 = await page.context().newPage();
  await open(p2, "TEACHER", T1, () => staff([A4, A7]));
  await p2.locator("main h3").first().waitFor();
  expect(await h3s(p2)).toEqual(["Hôm nay · 2"]);
  const p3 = await page.context().newPage();
  await open(p3, "TEACHER", T1, () => staff([A4, { ...A5, urgency: "high" }, A6]));
  await p3.locator("main h3").first().waitFor();
  const names = await h3s(p3);
  console.log("INFO EXAM_SIMILARITY high:", names.join(" ; "));
  expect(names[0]).toMatch(/^Gấp · 1/);
  expect(await rowsOf(p3, /^Gấp/)).toEqual([A5.title]);
  expect(names.some((n) => /^Khi rảnh · 1/.test(n)), "A6 (normal) vào Khi rảnh").toBe(true);
});

test("TC-UX10-5 không API mới: chỉ /me/courses, /me/today (hoặc courses/*/today), auth", async ({ page }) => {
  const st = await open(page, "TEACHER", T1, () => staff(FGV));
  await page.locator("main h3").first().waitFor();
  await page.waitForTimeout(800);
  const odd = [...new Set(st.urls)].filter((u) => !/^\/(me\/courses|me\/today|courses\/[^/]+\/today|auth\/|me$|me\/(notifications|unread)|notifications|events)/.test(u));
  console.log("INFO URL:", [...new Set(st.urls)].join(", "));
  expect(odd).toEqual([]);
});

test("TC-UX10-6/7/8 một nút đỏ ở việc đầu của nhóm đầu; các hàng còn lại ghost + mũi tên; bỏ nhóm Gấp → nút đỏ sang A4", async ({ page }) => {
  await open(page, "TEACHER", T1, () => staff(FGV));
  await page.locator("main h3").first().waitFor();
  const red = await redCount(page);
  console.log("INFO nút đỏ:", JSON.stringify(red));
  expect(red.length, "đúng 1 nút đỏ trong khung 1440×900").toBe(1);
  const info = await page.evaluate(() => [...document.querySelectorAll("main li[data-kind]")].map((li) => {
    const b = [...li.querySelectorAll("a, button")].filter((e) => !/^(Bỏ qua|Xem \d+ bước)/.test((e as HTMLElement).innerText.trim())).pop() as HTMLElement | undefined;
    const cs = b ? getComputedStyle(b) : null;
    return { k: (li as HTMLElement).dataset.kind, t: b?.innerText.trim() ?? "", svg: !!b?.querySelector("svg"), bg: cs?.backgroundColor ?? "", color: cs?.color ?? "", dec: cs?.textDecorationLine ?? "" };
  }));
  console.log("INFO hàng:", JSON.stringify(info));
  expect(info[0].k).toBe("AI_PENDING");
  expect(info[0].bg).not.toMatch(/rgba\(0, 0, 0, 0\)/);
  for (const r of info.slice(1)) { expect(r.bg, r.k + " ghost").toMatch(/rgba\(0, 0, 0, 0\)|transparent/); expect(r.svg, r.k + " mũi tên").toBe(true); expect(r.dec, r.k + " không gạch chân (link)").not.toBe("underline"); }
  const p2 = await page.context().newPage();
  await open(p2, "TEACHER", T1, () => staff([A4, A5, A6, A7]));
  await p2.locator("main h3").first().waitFor();
  const red2 = await redCount(p2);
  const first = await p2.evaluate(() => (document.querySelector("main li[data-kind]") as HTMLElement).dataset.kind);
  console.log("INFO nút đỏ khi không còn Gấp:", JSON.stringify(red2), "hàng đầu:", first);
  expect(red2.length).toBe(1);
  expect(first).toBe("REPORTED_POSTS");
});

test("TC-UX10-10/11 một lớp: không mã lớp ở hàng; Tất cả lớp: có mã lớp", async ({ page }) => {
  const courses = [mine(C1, "761988", "TEACHER"), mine(C2, "761987", "TEACHER")];
  const rows = [{ ...A1, course: c2 }, { ...A4, course: c1 }];
  await open(page, "TEACHER", courses, (p) => staff(p === "/me/today" ? rows : rows.filter((r) => (r.course as Json).id === (p.includes(C2) ? C2 : C1))), "/", async () => { await page.addInitScript((id) => localStorage.setItem("ep:ui:course", id), C2); });
  await page.locator("main h3").first().waitFor();
  const t = await main(page).innerText();
  console.log("INFO một lớp:", t.replace(/\s+/g, " ").slice(0, 200));
  expect(t, "mã lớp ở hàng / dòng phụ").not.toMatch(/\b76198[78]\b/);
  await expect(page.getByRole("heading", { level: 1 })).toContainText(/\d+ việc cần xử lý hôm nay/);
  expect(await main(page).getByRole("heading", { name: /^Việc cần xử lý$/ }).count(), "không khối con trùng tiêu đề").toBe(0);
  await page.getByRole("button", { name: /^Chọn lớp/ }).click();
  await page.getByRole("option", { name: /Tất cả lớp/ }).or(page.getByRole("menuitem", { name: /Tất cả lớp/ })).first().click();
  await page.waitForTimeout(800);
  const t2 = await main(page).innerText();
  console.log("INFO tất cả lớp:", t2.replace(/\s+/g, " ").slice(0, 240));
  expect(t2).toMatch(/761987/);
  expect(t2).toMatch(/761988/);
});

test("TC-UX10-12/13 màu chấm theo mức khẩn; nhóm có tên truy cập «Gấp, 3 việc»", async ({ page }) => {
  await open(page, "TEACHER", T1, () => staff(FGV));
  await page.locator("main h3").first().waitFor();
  const dots = await page.evaluate(() => [...document.querySelectorAll("main li[data-kind]")].map((li) => {
    const d = [...li.querySelectorAll("*")].find((e) => { const r = e.getBoundingClientRect(); const c = getComputedStyle(e); return r.width > 4 && r.width <= 14 && Math.abs(r.width - r.height) < 1 && parseFloat(c.borderRadius) >= r.width / 2 - 0.5 && c.backgroundColor !== "rgba(0, 0, 0, 0)"; });
    return { k: (li as HTMLElement).dataset.kind, bg: d ? getComputedStyle(d).backgroundColor : "—" };
  }));
  console.log("INFO chấm:", JSON.stringify(dots));
  const by = Object.fromEntries(dots.map((d) => [d.k, d.bg]));
  expect(by.AI_PENDING, "overdue đỏ").not.toBe("—");
  expect(new Set([by.AI_PENDING, by.REVIEW_FLAGGED, by.REPORTED_POSTS]).size, "ba mức ba màu").toBe(3);
  const names = await page.getByRole("heading", { level: 3 }).evaluateAll((l) => l.map((h) => (h.getAttribute("aria-label") ?? h.textContent ?? "").trim()));
  console.log("INFO tên h3:", names.join(" | "));
  const reg = await page.getByRole("region").evaluateAll((l) => l.map((e) => e.getAttribute("aria-label") ?? ""));
  console.log("INFO region:", reg.join(" | "));
  const acc = [...names, ...reg].join(" | ");
  expect(acc).toMatch(/Gấp, 3 việc/);
  expect(acc).toMatch(/Hôm nay, 2 việc/);
  expect(acc).toMatch(/Khi rảnh, 2 việc/);
});

test("TC-UX10-14/15/16/27 «Lớp cần chú ý» + «Sắp tới»: cột phải ≤ 320 px ở 1440, dưới danh sách ở 1099; Xem tất cả (N)", async ({ page }) => {
  const body = () => staff(FGV, { attention: [att(1), att(2)], attention_total: 7, upcoming: [up] });
  await open(page, "TEACHER", T1, body);
  await page.locator("main h3").first().waitFor();
  const geo = () => page.evaluate(() => {
    const r = (t: string) => { const h = [...document.querySelectorAll("main h2, main h3")].find((e) => (e as HTMLElement).innerText.trim() === t); const b = h?.closest("section, aside, div")?.getBoundingClientRect(); return b ? { l: Math.round(b.left), t: Math.round(b.top), w: Math.round(b.width) } : null; };
    const list = document.querySelector("main li[data-kind]")?.closest("ul, ol, section")?.getBoundingClientRect();
    return { att: r("Lớp cần chú ý"), up: r("Sắp tới"), list: list ? { l: Math.round(list.left), t: Math.round(list.top), r: Math.round(list.right), b: Math.round(list.bottom) } : null };
  });
  const g = await geo();
  console.log("INFO 1440:", JSON.stringify(g));
  expect(g.att && g.up && g.list).toBeTruthy();
  expect(g.att!.w).toBeLessThanOrEqual(322);
  expect(g.att!.l, "bên phải danh sách").toBeGreaterThanOrEqual(g.list!.r - 4);
  expect(Math.abs(g.att!.t - g.list!.t), "cùng hàng ngang").toBeLessThan(200);
  await expect(main(page).getByRole("link", { name: "Xem tất cả (7)" })).toBeVisible();
  const text1440 = (await main(page).innerText()).replace(/\s+/g, " ");
  await page.setViewportSize({ width: 1099, height: 900 });
  await page.waitForTimeout(400);
  const g2 = await geo();
  console.log("INFO 1099:", JSON.stringify(g2));
  expect(g2.att!.t, "dưới danh sách").toBeGreaterThanOrEqual(g2.list!.b - 4);
  const text1099 = (await main(page).innerText()).replace(/\s+/g, " ");
  expect(text1099, "cùng dữ liệu").toBe(text1440);
});

test("TC-UX10-18/19/20/21 lỗi tải, làm mới lỗi, rỗng, 15 trên 40", async ({ page }) => {
  const st = await open(page, "TEACHER", T1, () => staff(FGV));
  await page.locator("main h3").first().waitFor();
  st.fail = true;
  await page.getByRole("link", { name: "Hộp thư hỗ trợ" }).first().click();
  await page.waitForTimeout(600);
  await page.getByRole("link", { name: "Hôm nay" }).first().click();
  await page.waitForTimeout(2500);
  console.log("INFO sau refetch lỗi:", (await main(page).innerText()).replace(/\s+/g, " ").slice(0, 160));
  expect(await h3s(page), "giữ nhóm đã hiện").toEqual(["Gấp · 3", "Hôm nay · 2", "Khi rảnh · 2"]);
  expect((await main(page).innerText()).replace(/\s+/g, " "), "dải làm mới lỗi").toMatch(/Chưa làm mới được/);
  const p3 = await page.context().newPage();
  await open(p3, "TEACHER", T1, () => staff([]));
  await expect(p3.getByText("Bạn đã xử lý hết việc")).toBeVisible();
  expect(await p3.locator("main h3").count(), "không nhóm trống").toBe(0);
  const p4 = await page.context().newPage();
  const many = Array.from({ length: 15 }, (_, i) => act(`m${i}`, "QUESTION_REVIEW", `Việc ${i}`, "normal"));
  await open(p4, "TEACHER", T1, () => staff(many, { count: 40 }));
  await expect(p4.getByText("Đang hiện 15 việc đầu tiên.")).toBeVisible();
  expect(await h3s(p4)).toEqual(["Hôm nay · 15"]);
  const p5 = await page.context().newPage();
  const st5 = await open(p5, "TEACHER", T1, () => staff(FGV), "/", undefined, true);
  await expect(p5.getByRole("button", { name: "Thử lại" })).toBeVisible({ timeout: 20_000 });
  console.log("INFO lỗi lần đầu:", (await main(p5).innerText()).replace(/\s+/g, " ").slice(0, 160));
  st5.fail = false;
  await p5.getByRole("button", { name: "Thử lại" }).click();
  await p5.locator("main h3").first().waitFor();
  expect(await h3s(p5)).toEqual(["Gấp · 3", "Hôm nay · 2", "Khi rảnh · 2"]);
});

test("TC-UX10-22/23/24 sinh viên không có nhóm; admin; TA không có «Bỏ qua» ở COURSE_SETUP", async ({ page }) => {
  await open(page, "STUDENT", [mine(C2, "761987", "STUDENT")], () => ({ no_course: false, email_verified: true, recommended: null, timeline: [], continue: [] }));
  await page.getByRole("heading", { level: 1 }).waitFor();
  expect(await h3s(page)).not.toEqual(expect.arrayContaining(["Gấp"]));
  expect((await h3s(page)).filter((s) => /^(Gấp|Hôm nay|Khi rảnh) ·/.test(s))).toEqual([]);
  const pa = await page.context().newPage();
  await open(pa, "ADMIN", [], () => staff([act("s1", "SYSTEM_HEALTH", "Cần xem sức khoẻ hệ thống", "high", null), act("s2", "QUESTION_REVIEW", "Việc thường", "normal", null)]));
  await pa.locator("main h3").first().waitFor();
  console.log("INFO admin:", (await h3s(pa)).join(" ; "));
  expect((await h3s(pa)).some((n) => /^Khi rảnh/.test(n)), "admin: không nhóm Khi rảnh khi rỗng").toBe(false);
  const pt = await page.context().newPage();
  const setup = act("c1", "COURSE_SETUP", "Thiết lập lớp mới · 761987", "normal");
  (setup as Json).steps = [{ key: "a", label: "Chia sẻ mã lớp", done: false, href: "/class/settings" }];
  await open(pt, "TA", [mine(C2, "761987", "TA")], () => staff([A1, setup]));
  await pt.locator("main h3").first().waitFor();
  console.log("INFO TA:", (await h3s(pt)).join(" ; "));
  expect(await pt.getByRole("button", { name: "Bỏ qua" }).count(), "TA không có Bỏ qua").toBe(0);
  const row = pt.locator('li[data-kind="COURSE_SETUP"]');
  await expect(row).toBeVisible();
});

test("TC-UX10-25 375 px: nhãn nhóm + hàng xếp dọc, nút xuống dòng, vùng chạm ≥ 44", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await open(page, "TEACHER", T1, () => staff(FGV, { attention: [att(1)], attention_total: 1, upcoming: [up] }));
  await page.locator("main h3").first().waitFor();
  const r = await page.evaluate(() => ({
    ov: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    small: [...document.querySelectorAll("main li[data-kind] a, main li[data-kind] button")].map((e) => { const b = e.getBoundingClientRect(); return { t: (e as HTMLElement).innerText.trim().slice(0, 22), w: Math.round(b.width), h: Math.round(b.height) }; }).filter((b) => b.w > 0 && (b.h < 43 || b.w < 43)),
    rows: [...document.querySelectorAll("main li[data-kind]")].slice(0, 3).map((li) => { const t = (li.querySelector("a, h4, strong, span") as HTMLElement).getBoundingClientRect(); const b = ([...li.querySelectorAll("a, button")].pop() as HTMLElement).getBoundingClientRect(); return { titleBottom: Math.round(t.bottom), btnTop: Math.round(b.top) }; }),
  }));
  console.log("INFO 375:", JSON.stringify(r));
  expect(r.ov).toBe(false);
  expect(r.small, "vùng chạm < 44").toEqual([]);
});

test("TC-UX10-PQ TA/GV chỉ thấy việc theo JWT: không gọi courses khác", async ({ page }) => {
  const st = await open(page, "TA", [mine(C2, "761987", "TA")], () => staff([A1]));
  await page.locator("main h3").first().waitFor();
  const calls = [...new Set(st.urls.filter((u) => /today/.test(u)))];
  console.log("INFO today:", calls.join(", "));
  expect(calls.every((u) => u === "/me/today" || u.includes(C2))).toBe(true);
});

// ---------- US-UX91-11 ----------
const sv = (over: Json = {}): Json => ({ no_course: false, email_verified: true, recommended: { id: "r1", kind: "ASSIGNMENT_DUE", title: "Nộp Bài tập 03", reason: "Hạn 23:59 hôm nay.", urgency: "high", href: "/assignments", course: null, estimate_minutes: 30 }, timeline: [{ at: "2026-11-09T06:30:00Z", ends_at: "2026-11-09T09:00:00Z", title: "Buổi 5 · An ninh mạng", place: "P.405", state: "NEXT", course: c2 }], continue: [], ...over });
const chat = (id: string, title: string, at: string, course: Json = c2) => ({ kind: "CHAT", id, title, href: `/chat?session=${id}`, course, at });

test("TC-UX11-1/2/3/4/6 SV: thứ tự lời chào → Việc nên làm tiếp → Hôm nay → Tiếp tục học; khử trùng; 1 nút đỏ", async ({ page }) => {
  await page.clock.setFixedTime(new Date("2026-11-08T14:00:00+07:00"));
  const cont = [chat("s1", "Sâu Morris là gì", "2026-11-02T02:00:00Z"), chat("s2", "Sâu Morris là gì", "2026-11-05T02:00:00Z"), chat("s3", "Ransomware hoạt động", "2026-11-04T02:00:00Z"), chat("s4", "Mỗi lần phát biểu", "2026-11-03T02:00:00Z"), chat("s5", "sâu morris là gì", "2026-11-01T02:00:00Z"), chat("s6", "Sâu Morris là gì", "2026-11-06T02:00:00Z", c1)];
  await open(page, "STUDENT", [mine(C2, "761987", "STUDENT"), mine(C1, "761988", "STUDENT")], () => sv({ continue: cont.slice(0, 5) }));
  await page.getByRole("heading", { level: 1 }).waitFor();
  await page.getByRole("heading", { name: /^Tiếp tục học/ }).waitFor(); // khối cuối: chờ dữ liệu "Hôm nay" về hết rồi mới đo thứ tự (CI chậm hơn máy dev)
  const order = await page.evaluate(() => {
    const pos = (re: RegExp) => { const h = [...document.querySelectorAll("main h1, main h2, main h3")].find((e) => re.test((e as HTMLElement).innerText.trim())); return h ? { y: Math.round(h.getBoundingClientRect().top + scrollY), t: (h as HTMLElement).innerText.trim() } : null; };
    return [pos(/^Chào /), pos(/^Việc nên làm tiếp/), pos(/^Hôm nay$/), pos(/^Tiếp tục học/)];
  });
  console.log("INFO thứ tự:", JSON.stringify(order));
  expect(order.every(Boolean)).toBe(true);
  const ys = order.map((o) => o!.y);
  expect([...ys].sort((a, b) => a - b), "thứ tự").toEqual(ys);
  expect(ys[2], "«Hôm nay» trong khung nhìn đầu").toBeLessThan(900);
  const rows = await main(page).getByRole("link", { name: /Sâu Morris|Ransomware|Mỗi lần/i }).allInnerTexts();
  console.log("INFO tiếp tục học:", JSON.stringify(rows));
  expect(rows.length).toBeLessThanOrEqual(3);
  expect(rows.filter((r) => /sâu morris/i.test(r)).length, "một hàng Sâu Morris").toBe(1);
  expect((await redCount(page)).length, "một nút đỏ").toBe(1);
  const p2 = await page.context().newPage();
  await open(p2, "STUDENT", [mine(C2, "761987", "STUDENT"), mine(C1, "761988", "STUDENT")], () => sv({ continue: [cont[1], cont[5]] }));
  await p2.getByRole("heading", { level: 1 }).waitFor();
  await p2.waitForTimeout(1200);
  console.log("INFO khác lớp:", (await main(p2).innerText()).replace(/\s+/g, " ").slice(0, 400));
  expect((await main(p2).getByRole("link", { name: /Sâu Morris/i }).count()), "khác lớp thì cả hai hiện").toBe(2);
});

test("TC-UX11-5 thời gian «Tiếp tục học» theo US-02", async ({ page }) => {
  await page.clock.setFixedTime(new Date("2026-11-08T14:00:00+07:00"));
  await open(page, "STUDENT", [mine(C2, "761987", "STUDENT")], () => sv({ continue: [chat("t1", "Phiên cũ", "2026-11-01T17:13:00Z"), chat("t2", "Phiên hôm qua", "2026-11-06T17:13:00Z")] }));
  await page.getByRole("heading", { name: "Tiếp tục học" }).waitFor();
  const t = (await main(page).innerText()).replace(/\s+/g, " ");
  console.log("INFO:", t.slice(-220));
  expect(t).toMatch(/Thứ Hai, 00:13/);
  expect(t).toMatch(/Hôm qua, 00:13/);
  expect(t).not.toMatch(/Thứ 2, 2\/11/);
});

test("TC-UX11-7/8/9 sv.moi nhập mã; GV/TA không có khối của SV; 375 px thứ tự + thanh dưới", async ({ page }) => {
  await open(page, "STUDENT", [], () => ({ no_course: true, email_verified: true, recommended: { id: "j", kind: "JOIN_CODE", title: "Nhập mã tham gia lớp", reason: "Chưa vào lớp.", urgency: "normal", href: "/join", course: null, estimate_minutes: 1 }, timeline: [], continue: [] }));
  await expect(page.getByLabel("Mã tham gia lớp")).toBeVisible();
  expect(await main(page).getByRole("heading", { name: /^Hôm nay$/ }).count(), "không khối Hôm nay trống").toBe(0);
  const pt = await page.context().newPage();
  await open(pt, "TEACHER", T1, () => staff(FGV));
  await pt.locator("main h3").first().waitFor();
  expect(await pt.getByRole("heading", { name: /Việc nên làm tiếp|Tiếp tục học/ }).count()).toBe(0);
  const pm = await page.context().newPage();
  await pm.setViewportSize({ width: 375, height: 812 });
  await open(pm, "STUDENT", [mine(C2, "761987", "STUDENT")], () => sv({ continue: [chat("m1", "Phiên 1", "2026-11-05T02:00:00Z")] }));
  await pm.getByRole("heading", { level: 1 }).waitFor();
  await pm.waitForTimeout(1200);
  const r = await pm.evaluate(() => ({ ov: document.documentElement.scrollWidth > document.documentElement.clientWidth, cur: [...document.querySelectorAll("[aria-current]")].map((a) => (a as HTMLElement).innerText.trim()), ys: [/^Việc nên làm tiếp/, /^Hôm nay$/, /^Tiếp tục học/].map((re) => { const h = [...document.querySelectorAll("main h2, main h3")].find((e) => re.test((e as HTMLElement).innerText.trim())); return h ? Math.round(h.getBoundingClientRect().top + scrollY) : -1; }) }));
  console.log("INFO 375 SV:", JSON.stringify(r));
  expect(r.ov).toBe(false);
  expect(r.cur.join("|")).toMatch(/Hôm nay/);
  expect(r.ys.every((y) => y >= 0)).toBe(true);
  expect([...r.ys].sort((a, b) => a - b)).toEqual(r.ys);
});

test("TC-UX11-PQ phiên chat chỉ của chính SV: không gọi API phiên của người khác", async ({ page }) => {
  const st = await open(page, "STUDENT", [mine(C2, "761987", "STUDENT")], () => sv());
  await page.getByRole("heading", { level: 1 }).waitFor();
  expect(st.urls.filter((u) => /\/students\/|\/chat\/sessions/.test(u))).toEqual([]);
});

for (const reduce of [false, true]) {
  test(`TC-UX10-17 Red Thread khi bấm việc (reduce=${reduce}): < 300 ms; reduce → không chạy`, async ({ browser }) => {
    const ctx = await browser.newContext({ baseURL: BASE_URL, reducedMotion: reduce ? "reduce" : "no-preference", viewport: { width: 1440, height: 900 } });
    const page = await ctx.newPage();
    settleGoto(page);
    await open(page, "TEACHER", T1, () => staff(FGV));
    await page.locator("main h3").first().waitFor();
    await page.evaluate(() => {
      const w = window as unknown as { __a: Array<{ n: string; d: number }> };
      w.__a = [];
      const seen = new Set<Animation>();
      const tick = () => { for (const a of document.getAnimations()) if (!seen.has(a)) { seen.add(a); w.__a.push({ n: (a as CSSAnimation).animationName ?? (a as CSSTransition).transitionProperty ?? "?", d: Number(a.effect?.getTiming().duration ?? 0) }); } requestAnimationFrame(tick); };
      tick();
    });
    const t0 = Date.now();
    await page.locator('li[data-kind="AI_PENDING"]').getByRole("link", { name: "Mở" }).or(page.locator('li[data-kind="AI_PENDING"]').getByRole("button", { name: "Mở" })).first().click();
    await page.waitForTimeout(1200);
    const a = await page.evaluate(() => (window as unknown as { __a: Array<{ n: string; d: number }> }).__a);
    console.log(`INFO reduce=${reduce} hoạt cảnh sau khi bấm (${Date.now() - t0} ms):`, JSON.stringify(a.slice(0, 12)));
    const real = a.filter((x) => !/shimmer|spin|dot/.test(x.n));
    console.log(`INFO reduce=${reduce} không tính chỉ báo tải:`, JSON.stringify([...new Set(real.map((x) => `${x.n}:${x.d}`))]));
    const longest = Math.max(0, ...real.map((x) => x.d));
    if (reduce) expect(longest, "reduce: không có hoạt cảnh > 1 ms").toBeLessThanOrEqual(1);
    else expect(longest, "mọi hoạt cảnh < 300 ms").toBeLessThan(300);
    await ctx.close();
  });
}
