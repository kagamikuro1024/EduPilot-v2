/* eslint-disable @typescript-eslint/no-explicit-any */
import { test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";

async function login(page: Page, email: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}
const api = (page: Page, path: string) =>
  page.evaluate(async (p) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await fetch(`/api/v1${p}`, { headers: { authorization: `Bearer ${a.access_token}` } });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, path);
const courseId = async (page: Page, code: string) => {
  const m = await api(page, "/me/courses");
  return m.body.items.find((i: { course: { class_code: string } }) => i.course.class_code === code).course.id as string;
};

type Row = { id: string; title: string; preview: string; tags: string[]; week_no: number | null; author: { full_name: string; role: string; is_me: boolean }; answer_state: string | null; reply_count: number; last_activity_at: string; created_at: string };
const mk = (i: number, title: string, preview: string, st: string | null): Row => ({ id: `00000000-0000-4000-8000-${String(i).padStart(12, "0")}`, title, preview, tags: [], week_no: 3, author: { full_name: `Tác giả ${i}`, role: "STUDENT", is_me: false }, answer_state: st, reply_count: 0, last_activity_at: "2026-11-08T08:10:00Z", created_at: "2026-11-08T08:10:00Z" });
const FIX_PREV = ["- - Hướng dẫn nộp bài", "· Phisher - Individual uses email…", "* Ví dụ", "# Tiêu đề lớn", "> trích dẫn", "- - - ·", "Dòng một của đoạn dài.\nDòng hai " + "chữ ".repeat(80)];
const fixture = (pending: number): Row[] => {
  const r: Row[] = [];
  for (let i = 0; i < pending; i++) r.push(mk(i + 1, `T${i + 1}`, FIX_PREV[i] ?? `Nội dung thường ${i + 1}`, "PENDING"));
  r.push(mk(101, "T8 xác nhận", "đã xác nhận", "VERIFIED"), mk(102, "T9 sửa", "đã sửa", "CORRECTED"), mk(103, "T10 loại", "đã loại", "REJECTED"));
  return r;
};
const mock = async (page: Page, rows: Row[]) => {
  await page.route(/\/api\/v1\/courses\/[0-9a-f-]+\/threads(\?.*)?$/, (r) => (r.request().method() === "GET" ? r.fulfill({ json: { items: rows, pinned: [], next_cursor: null } }) : r.continue()));
};
const rowsOf = (page: Page) => page.locator("[data-part=thread-list] > li");

test("TC-UX19 giảng viên: đoạn trích, nhãn đa số, nhãn từng thread (dữ liệu giả)", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(120000);
  for (const pending of [17, 10]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
    const page = await ctx.newPage();
    await login(page, "teacher@edupilot.local");
    await mock(page, fixture(pending));
    await page.goto("/threads");
    await page.waitForTimeout(3500);
    const rows = rowsOf(page);
    const data = await rows.evaluateAll((els) => els.map((e) => ({ title: (e.querySelector("a span") as HTMLElement | null)?.innerText ?? "", prev: (e.querySelector("p") as HTMLElement | null)?.innerText ?? "", meta: (e.querySelector("[class*=meta]") as HTMLElement | null)?.innerText.replace(/\n/g, " | ") ?? "" })));
    console.log(`INFO UX19 pending=${pending}/${pending + 3}:`, JSON.stringify(data.slice(0, 7)));
    console.log(`INFO UX19 pending=${pending} T8–T10:`, JSON.stringify(data.slice(-3)));
    console.log(`INFO UX19 pending=${pending} dòng đầu:`, JSON.stringify((await page.locator("main").innerText()).match(/\d+ câu chờ xác nhận/)), "| số hàng có «Chờ xác nhận»:", data.filter((d) => d.meta.includes("Chờ xác nhận")).length, "| nhãn khác:", JSON.stringify(data.filter((d) => /Đã được|Đã sửa|Đã loại/.test(d.meta)).map((d) => d.title)));
    const t7 = data[6].prev;
    console.log("INFO UX19-2 T7:", t7.length, JSON.stringify(t7.slice(0, 30)), t7.endsWith("…"));
    if (pending === 17) await page.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/threads-gv-fixture-1440.png" });
    await ctx.close();
  }
});

test("TC-UX19 thật: giảng viên, SV 375 bộ lọc, thời gian, phân trang, 403, PII", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(200000);
  // giảng viên: dữ liệu thật
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  const urls: string[] = [];
  page.on("request", (r) => { if (/\/courses\/[0-9a-f-]+\/threads\?/.test(r.url())) urls.push(new URL(r.url()).search); });
  await login(page, "teacher@edupilot.local");
  await page.goto("/threads");
  await page.waitForTimeout(4000);
  const rows = rowsOf(page);
  const data = await rows.evaluateAll((els) => els.map((e) => ({ prev: (e.querySelector("p") as HTMLElement | null)?.innerText ?? "", meta: (e.querySelector("[class*=meta]") as HTMLElement | null)?.innerText.replace(/\n/g, " | ") ?? "" })));
  console.log("INFO UX19-1 thật: đoạn trích:", JSON.stringify(data.map((d) => d.prev.slice(0, 40))));
  console.log("INFO UX19-1 bắt đầu bằng ký tự thừa:", JSON.stringify(data.filter((d) => /^[-*·#>•–—]/.test(d.prev)).map((d) => d.prev.slice(0, 30))));
  console.log("INFO UX19-8 thời gian:", JSON.stringify(data.map((d) => d.meta.split(" | ").pop())), "| dạng cũ:", data.some((d) => /\d\d:\d\d \d\d-\d\d/.test(d.meta)));
  console.log("INFO UX19-PG request:", JSON.stringify(urls.slice(0, 3)));
  console.log("INFO UX19-13 h1:", JSON.stringify(await page.locator("main h1").allInnerTexts()));
  const cid = await courseId(page, "761988");
  const list = await api(page, `/courses/${cid}/threads?limit=5`);
  const tid = list.body.items[0].id as string;
  await ctx.close();
  // SV 375
  const c2 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  const p2 = await c2.newPage();
  await login(p2, "sv.kha@edupilot.local");
  await p2.goto("/threads");
  await p2.waitForTimeout(4000);
  const fb = p2.getByRole("button", { name: /^Bộ lọc/ });
  console.log("INFO UX19-7 nút:", await fb.count(), JSON.stringify(await fb.first().boundingBox()), "| svg:", await fb.first().locator("svg").count(), "| chữ:", await fb.first().innerText());
  await fb.first().tap();
  await p2.waitForTimeout(700);
  const dlg = p2.getByRole("dialog").first();
  console.log("INFO UX19-7 bảng:", JSON.stringify(await dlg.boundingBox()), "| viewport h 812");
  const sels = dlg.locator("select");
  console.log("INFO UX19-7 số ô lọc:", await sels.count());
  await sels.nth(0).selectOption({ index: 2 }).catch(() => undefined);
  await sels.nth(1).selectOption({ index: 1 }).catch(() => undefined);
  await p2.waitForTimeout(500);
  await dlg.getByRole("button", { name: "Xong" }).tap().catch(() => undefined);
  await p2.waitForTimeout(800);
  console.log("INFO UX19-7 sau đặt 2 bộ lọc:", JSON.stringify(await fb.first().innerText()), JSON.stringify(await fb.first().boundingBox()));
  await p2.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/threads-sv-375.png" });
  console.log("INFO UX19-6 SV: danh sách có nội dung AI chưa xác nhận?", JSON.stringify((await rowsOf(p2).first().innerText()).replace(/\s+/g, " ").slice(0, 200)));
  // PII dialog: SV viết bài có MSSV
  await c2.close();
  const c3 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const p3 = await c3.newPage();
  await login(p3, "sv.kha@edupilot.local");
  await p3.goto("/threads");
  await p3.waitForTimeout(3500);
  await p3.getByRole("button", { name: "Đặt câu hỏi" }).first().click();
  const form = p3.locator("[data-part=thread-form]");
  await form.getByRole("textbox").first().fill("Hỏi về điểm của em");
  await form.getByRole("textbox").nth(1).fill("Em là 20229002, email sv.kha@edupilot.local, xin hỏi điểm của em");
  await form.getByRole("button", { name: /Đăng|Gửi/ }).first().click();
  await p3.waitForTimeout(2500);
  console.log("INFO UX19-12 hộp thoại:", await p3.getByRole("dialog").count(), JSON.stringify((await p3.getByRole("dialog").first().innerText().catch(() => "")).replace(/\s+/g, " ").slice(0, 300)), "| status:", JSON.stringify(await p3.locator("[data-part=pii-notice]").allInnerTexts()));
  await p3.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/threads-pii-1440.png" });
  await c3.close();
  // 403 SV không thuộc lớp
  const c4 = await browser.newContext({ ignoreHTTPSErrors: true });
  const p4 = await c4.newPage();
  await login(p4, "sv.moi@edupilot.local");
  const r1 = await api(p4, `/courses/${cid}/threads/${tid}`);
  const r2 = await api(p4, `/courses/${cid}/threads?limit=5`);
  console.log("INFO UX19-11 SV không thuộc lớp:", r1.status, r2.status);
  await c4.close();
});

test("TC-UX20 Kết quả của tôi: một khối, số khớp API, chưa công bố, 375", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(180000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "sv.kha@edupilot.local");
  const cid = await courseId(page, "761987");
  const g = await api(page, `/courses/${cid}/me/grades`);
  console.log("INFO UX20-3 API me/grades:", JSON.stringify(g.body).slice(0, 1200));
  await page.goto("/me");
  await page.waitForTimeout(4500);
  const dom = await page.evaluate(() => {
    const m = document.querySelector("main")!;
    return { h1: [...m.querySelectorAll("h1")].map((x) => x.textContent), h2: [...m.querySelectorAll("h2")].map((x) => x.textContent), dl: m.querySelectorAll("dl").length, panels: m.querySelectorAll("[data-part=panel], section").length, lis: [...m.querySelectorAll("li")].slice(0, 8).map((x) => (x as HTMLElement).innerText.replace(/\n/g, " ⏎ ")), buttons: [...m.querySelectorAll("button")].map((b) => (b as HTMLElement).innerText.trim()).filter(Boolean), sub: [...m.querySelectorAll("h1 + *")].map((x) => (x as HTMLElement).innerText.slice(0, 60)) };
  });
  console.log("INFO UX20-1 DOM:", JSON.stringify(dom));
  const txt = (await page.locator("main").innerText()).replace(/\n+/g, " ⏎ ");
  console.log("INFO UX20-1 «Căn cứ» số lần:", (txt.match(/Căn cứ/g) ?? []).length, "| «Giữa kỳ · 20 %» lặp:", (txt.match(/Giữa kỳ/g) ?? []).length, "| mã lớp trần:", /^761987$/m.test(await page.locator("main").innerText()), "| đã công bố nút xem lại:", JSON.stringify(txt.match(/Yêu cầu xem lại/g)));
  // nửa dưới: bài tập / điểm nháp
  console.log("INFO UX20-4 từ nháp / rủi ro / ghi chú:", JSON.stringify(txt.match(/nháp|rủi ro|ghi chú quan sát|DRAFT/gi)));
  await page.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/me-1440.png", fullPage: true });
  // SV chưa có điểm
  const c2 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const p2 = await c2.newPage();
  await login(p2, "sv.moi@edupilot.local");
  await p2.goto("/me");
  await p2.waitForTimeout(4500);
  console.log("INFO UX20-6 sv.moi /me:", JSON.stringify((await p2.locator("main").innerText()).replace(/\n+/g, " ⏎ ").slice(0, 500)));
  await p2.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/me-moi-1440.png" });
  // 403 SV A gọi dữ liệu SV B
  const cs = await api(p2, "/me/courses");
  console.log("INFO UX20-8 sv.moi lớp:", JSON.stringify(cs.body.items?.map((i: any) => i.course.class_code)));
  const adm = await browser.newContext({ ignoreHTTPSErrors: true }); const pa = await adm.newPage(); await login(pa, "teacher@edupilot.local");
  const us = await api(pa, `/courses/${cid}/students?limit=5`);
  const uid = us.body?.items?.[0]?.id ?? us.body?.items?.[0]?.user_id ?? us.body?.items?.[0]?.student?.id;
  console.log("INFO UX20-8 mẫu SV B uid:", uid, "| keys:", JSON.stringify(Object.keys(us.body?.items?.[0] ?? {})));
  const x1 = await api(page, `/courses/${cid}/gradebook/students/${uid}/explain`);
  const x2 = await api(page, `/courses/${cid}/me/grades?user_id=${uid}`);
  console.log("INFO UX20-8 SV A gọi explain của B:", x1.status, "| me/grades?user_id=B:", x2.status, JSON.stringify(x2.body).slice(0, 80));
  await adm.close(); await c2.close(); await ctx.close();
  // 375
  const c3 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  const p3 = await c3.newPage();
  await login(p3, "sv.kha@edupilot.local");
  await p3.goto("/me");
  await p3.waitForTimeout(4500);
  console.log("INFO UX20-9 375: scrollHeight", await p3.evaluate(() => document.documentElement.scrollHeight), "| tràn ngang:", await p3.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth));
  await p3.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/me-375-full.png", fullPage: true });
  await c3.close();
});

test("TC-UX18 Tài liệu: nhãn Loại, Chia sẻ từ, công tắc, 375, SV, lớp lạ", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(180000);
  for (const [w, h] of [[1440, 900], [1100, 800]] as const) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h } });
    const page = await ctx.newPage();
    await login(page, "teacher@edupilot.local");
    await page.goto("/documents");
    await page.waitForTimeout(4500);
    const cells = await page.locator("main tbody tr").evaluateAll((trs) => trs.map((tr) => { const tds = [...tr.querySelectorAll("td")]; const sel = tds[1]; return { type: sel?.innerText.trim(), sw: Math.round(sel?.scrollWidth ?? 0), cw: Math.round(sel?.clientWidth ?? 0), title: (tds[0] as HTMLElement)?.innerText.replace(/\n/g, " | ") }; }));
    console.log(`INFO UX18-1 @${w} Loại:`, JSON.stringify(cells.map((c) => `${c.type}:${c.sw}/${c.cw}`)));
    console.log(`INFO UX18-2 @${w} «Chia sẻ từ»:`, JSON.stringify(cells.map((c) => c.title.match(/Chia sẻ từ [^|]*/)?.[0] ?? "—")));
    const sw = await page.locator("main tbody tr [role=switch]").evaluateAll((els) => els.map((e) => ({ name: e.getAttribute("aria-label") ?? "", on: e.getAttribute("aria-checked"), dis: (e as HTMLButtonElement).disabled || e.getAttribute("aria-disabled"), w: Math.round(e.getBoundingClientRect().width), h: Math.round(e.getBoundingClientRect().height), bg: getComputedStyle(e).backgroundColor, op: getComputedStyle(e).opacity })));
    console.log(`INFO UX18-4 @${w} công tắc:`, JSON.stringify(sw));
    if (w === 1440) {
      await page.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/documents-1440.png" });
      // UX18-7: chặn API, đổi công tắc
      const first = page.locator("main tbody tr [role=switch]").first();
      const before = await first.getAttribute("aria-checked");
      await page.route(/\/api\/v1\/courses\/[0-9a-f-]+\/documents\/[0-9a-f-]+/, (r) => (r.request().method() === "PATCH" || r.request().method() === "PUT" || r.request().method() === "POST" ? r.abort() : r.continue()));
      await first.click();
      await page.waitForTimeout(2500);
      console.log("INFO UX18-7 công tắc:", before, "→", await first.getAttribute("aria-checked"), "| lỗi:", JSON.stringify((await page.getByRole("alert").allInnerTexts()).slice(0, 2)), "| Thử lại:", await page.getByRole("button", { name: "Thử lại" }).count());
      await page.unroute(/\/api\/v1\/courses\/[0-9a-f-]+\/documents\/[0-9a-f-]+/);
      if (await page.getByRole("button", { name: "Thử lại" }).count()) { await page.getByRole("button", { name: "Thử lại" }).first().click(); await page.waitForTimeout(2000); console.log("INFO UX18-7 sau Thử lại:", await first.getAttribute("aria-checked")); }
      const cid = await courseId(page, "761988");
      const d = await api(page, `/courses/${cid}/documents?limit=50`);
      console.log("INFO UX18 tài liệu API:", JSON.stringify((d.body?.items ?? []).map((x: any) => [x.title.slice(0, 18), x.doc_type ?? x.type, x.shared_from ?? "∅", x.ai_enabled, x.visible_to_students])));
      const dstate = await api(page, `/courses/${cid}/documents?limit=50`);
      void dstate;
      // đổi lại công tắc đã đổi (nếu Thử lại đã áp dụng)
      if ((await first.getAttribute("aria-checked")) !== before) await first.click();
      // ID: bấm nhanh 5 lần
      const sw2 = page.locator("main tbody tr [role=switch]").nth(1);
      const b2 = await sw2.getAttribute("aria-checked");
      for (let i = 0; i < 5; i++) { await sw2.click({ delay: 10 }); }
      await page.waitForTimeout(3000);
      const fin = await sw2.getAttribute("aria-checked");
      await page.reload(); await page.waitForTimeout(3500);
      console.log("INFO UX18-ID 5 lần bấm:", b2, "→ giao diện", fin, "→ sau tải lại", await page.locator("main tbody tr [role=switch]").nth(1).getAttribute("aria-checked"), "(kỳ vọng đảo vì 5 lần lẻ)");
      // trả lại nếu lệch
      if ((await page.locator("main tbody tr [role=switch]").nth(1).getAttribute("aria-checked")) !== b2) await page.locator("main tbody tr [role=switch]").nth(1).click();
    }
    await ctx.close();
  }
  // 375
  const c3 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  const p3 = await c3.newPage();
  await login(p3, "teacher@edupilot.local");
  await p3.goto("/documents");
  await p3.waitForTimeout(4500);
  console.log("INFO UX18-5/9 375: tràn ngang", await p3.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth), "| công tắc:", JSON.stringify(await p3.locator("[role=switch]").evaluateAll((els) => els.slice(0, 4).map((e) => `${Math.round(e.getBoundingClientRect().width)}x${Math.round(e.getBoundingClientRect().height)}`))));
  await p3.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/documents-375.png" });
  await c3.close();
  // SV, lớp lạ
  const c4 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const p4 = await c4.newPage();
  await login(p4, "sv.kha@edupilot.local");
  await p4.goto("/documents"); await p4.waitForTimeout(3000);
  const cid = await courseId(p4, "761987");
  console.log("INFO UX18-8 SV /documents:", JSON.stringify((await p4.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 120)), "| API:", (await api(p4, `/courses/${cid}/documents?limit=5`)).status);
  const lib = await api(p4, `/courses/${cid}/library?limit=100`);
  console.log("INFO UX18-6 SV thư viện:", lib.status, JSON.stringify((lib.body?.items ?? []).map((x: any) => x.title ?? x.name).filter(Boolean)));
  await p4.goto("/library"); await p4.waitForTimeout(3500);
  console.log("INFO UX18-6 SV /library có «Đáp án»:", /Đáp án/.test(await p4.locator("main").innerText()));
  await c4.close();
  const c5 = await browser.newContext({ ignoreHTTPSErrors: true });
  const p5 = await c5.newPage();
  await login(p5, "ta@edupilot.local");
  const adm = await browser.newContext({ ignoreHTTPSErrors: true }); const pa = await adm.newPage(); await login(pa, "teacher@edupilot.local");
  const cArch = await courseId(pa, "QC196964");
  console.log("INFO UX18-PQ TA /documents lớp lạ:", (await api(p5, `/courses/${cArch}/documents?limit=5`)).status, "| /insights:", (await api(p5, `/courses/${cArch}/insights`)).status);
  await adm.close(); await c5.close();
});
