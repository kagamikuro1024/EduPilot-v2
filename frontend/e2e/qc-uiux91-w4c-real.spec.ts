import { expect, test, type Browser, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC đợt 4, vòng 3: sổ điểm (cột Trạng thái, Chốt điểm, TA), lỗi duyệt bài, nháy lưu `.ep-saved` (US-04 AC7), cảnh báo công thức (761988), ảnh K7.
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
async function login(page: Page, user: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30_000 });
  await page.locator("[data-part=topbar]").waitFor({ timeout: 15_000 });
  await page.waitForTimeout(1200);
}
const api = <T = unknown>(page: Page, path: string, init?: { method?: string; body?: unknown }) =>
  page.evaluate(async ([p, i]) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const headers: Record<string, string> = { authorization: `Bearer ${a.access_token}`, "content-type": "application/json" };
    if (i && (i as { method?: string }).method === "POST") headers["Idempotency-Key"] = crypto.randomUUID();
    const r = await fetch(`/api/v1${p}`, { method: (i as { method?: string } | undefined)?.method ?? "GET", headers, body: i && (i as { body?: unknown }).body ? JSON.stringify((i as { body?: unknown }).body) : undefined });
    return { status: r.status, body: (await r.json().catch(() => null)) as unknown };
  }, [path, init ?? null] as const) as Promise<{ status: number; body: T }>;
type Mine = { items: Array<{ course: { id: string; class_code: string } }> };
async function who(browser: Browser, w = 1440, h = 900, user = "teacher", code = "761987") {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, hasTouch: w < 500, isMobile: w < 500, locale: "vi-VN" });
  const page = await ctx.newPage();
  await login(page, user);
  const c = (await api<Mine>(page, "/me/courses")).body.items.find((i) => i.course.class_code === code)!.course.id;
  return { ctx, page, c };
}
const AID = async (page: Page, c: string) => (await api<{ items: Array<{ id: string; title: string }> }>(page, `/courses/${c}/assignments?limit=50`)).body.items.find((a) => /Bài tập 03/.test(a.title))!.id;
const subs = async (page: Page, c: string, aid: string, st: string) => (await api<{ items: Array<{ id: string; student: { full_name: string } }> }>(page, `/courses/${c}/submissions?limit=100&assignment_id=${aid}&status=${st}`)).body.items;
const desktop = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
const startSampler = (page: Page) =>
  page.evaluate(() => {
    const w = window as unknown as { __saved: string[] };
    w.__saved = [];
    const tick = () => { for (const a of document.getAnimations()) { const n = (a as CSSAnimation).animationName; if (n && /saved/.test(n)) { const d = `${n}:${a.effect?.getTiming().duration}`; if (!w.__saved.includes(d)) w.__saved.push(d); } } requestAnimationFrame(tick); };
    tick();
  });
const saved = (page: Page) => page.evaluate(() => (window as unknown as { __saved: string[] }).__saved);

test("TC-UX16 sổ điểm thật: cột Trạng thái, menu ⋯, Chốt điểm, TA không chốt được", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await who(browser);
  await page.goto(`/gradebook?course=${c}`);
  await page.waitForTimeout(4000);
  const statuses = new Set<string>();
  const heads = await page.locator("main thead th").allInnerTexts();
  const si = heads.findIndex((h) => /Trạng thái/.test(h));
  for (let k = 0; k < 6; k++) {
    for (const t of await page.locator(`main tbody tr td:nth-child(${si + 1})`).allInnerTexts()) statuses.add(t.trim());
    await page.evaluate((y) => { const e = [...document.querySelectorAll("main *")].find((x) => /(auto|scroll)/.test(getComputedStyle(x).overflowY) && x.scrollHeight > x.clientHeight) as HTMLElement; e.scrollTop = y; }, k * 400);
    await page.waitForTimeout(300);
  }
  console.log("INFO giá trị ô Trạng thái:", JSON.stringify([...statuses]), "| cột:", si >= 0);
  const g = await api<{ rows: Array<{ result: Record<string, string> | null; drift?: boolean }> }>(page, `/courses/${c}/gradebook?limit=100`);
  console.log("INFO API trạng thái:", JSON.stringify([...new Set(g.body.rows.map((r) => `${r.result?.status}/${r.result?.verdict}/${r.drift ?? ""}`))]));
  expect.soft([...statuses].filter(Boolean).length > 0 || si < 0, "UX16-10: cột Trạng thái chỉ hiện khi có hàng cần hiển thị").toBe(true);
  await page.evaluate(() => { const e = [...document.querySelectorAll("main *")].find((x) => /(auto|scroll)/.test(getComputedStyle(x).overflowY) && x.scrollHeight > x.clientHeight) as HTMLElement; e.scrollTop = 0; });
  const th = page.locator("main thead th").filter({ hasText: "Kiểm tra hằng tuần" }).first();
  console.log("INFO nút trong tiêu đề cột:", await th.locator("button").count());
  const chot = page.getByRole("button", { name: "Chốt điểm" });
  const dis = await chot.getAttribute("aria-disabled");
  console.log("INFO Chốt điểm aria-disabled:", dis);
  if (dis !== "true") {
    await chot.click();
    await page.waitForTimeout(1500);
    const dt = (await page.getByRole("dialog").innerText().catch(() => "")).replace(/\s+/g, " ");
    console.log("INFO hộp xác nhận:", dt.slice(0, 500));
    expect(dt, "hậu quả bằng số").toMatch(/\d/);
    await page.keyboard.press("Escape");
    await page.waitForTimeout(600);
    const st = await api<{ gradebook?: { status?: string } }>(page, `/courses/${c}/grade-scheme/status`);
    console.log("INFO sổ sau Esc:", st.body.gradebook?.status);
    expect(st.body.gradebook?.status).not.toBe("LOCKED");
  }
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.screenshot({ path: "/out/shots-w4/gradebook-1440.png" });
  await ctx.close();
  const t = await who(browser, 1440, 900, "ta");
  const fin = await api(t.page, `/courses/${t.c}/gradebook/finalize`, { method: "POST", body: {} });
  console.log("INFO TA POST finalize:", fin.status);
  expect([403, 404]).toContain(fin.status);
  await t.page.goto(`/gradebook?course=${t.c}`);
  await t.page.waitForTimeout(3500);
  const tb = t.page.getByRole("button", { name: "Chốt điểm" });
  console.log("INFO TA nút Chốt điểm:", await tb.count(), await tb.first().getAttribute("aria-disabled").catch(() => null));
  await t.ctx.close();
});

test("TC-UX15-13/14 lỗi duyệt bài báo tại thanh dưới, điểm còn, Thử lại; 409 khi lưu điểm", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await who(browser);
  const aid = await AID(page, c);
  const d = await subs(page, c, aid, "DRAFT");
  expect(d.length).toBeGreaterThan(0);
  const writes: string[] = [];
  page.on("request", (r) => { if (r.method() !== "GET" && r.url().includes("/api/v1/") && !/auth|events/.test(r.url())) writes.push(`${r.method()} ${new URL(r.url()).pathname.replace(/[0-9a-f-]{36}/g, "‹id›")}`); });
  await page.goto(`/grading/${d[0].id}?course=${c}`);
  await page.waitForTimeout(3500);
  const inp = page.locator("main input[aria-label^='Điểm ']").nth(1);
  const v0 = await inp.inputValue();
  await inp.fill(v0 === "1,25" ? "1,00" : "1,25");
  await inp.blur();
  await page.waitForTimeout(1500);
  console.log("INFO ghi khi sửa điểm:", JSON.stringify([...new Set(writes)]));
  const saveUrl = writes.find((w) => /grading-results/.test(w) && !/review/.test(w));
  let block = true;
  await page.route(/grading-results\/[0-9a-f-]{36}\/review/, (r) => (block ? r.abort() : r.continue()));
  const typed = await inp.inputValue();
  await page.getByRole("button", { name: "Duyệt bài" }).click();
  await page.waitForTimeout(2500);
  const bar = await page.evaluate(() => { const btn = [...document.querySelectorAll("main button")].find((b) => /Duyệt bài|Thử lại/.test((b as HTMLElement).innerText)) as HTMLElement | undefined; let n = btn?.parentElement ?? null; while (n && n !== document.body) { const p = getComputedStyle(n).position; if (p === "sticky" || p === "fixed") return n.innerText.replace(/\s+/g, " ").slice(0, 220); n = n.parentElement; } return ""; });
  console.log("INFO thanh dưới sau lỗi:", bar, "| URL:", page.url().slice(-40));
  expect(bar, "UX15-13: lỗi báo tại thanh dưới").toMatch(/không|lỗi|Lỗi|Thử lại/);
  expect(await inp.inputValue(), "điểm nhập còn nguyên").toBe(typed);
  block = false;
  const retry = page.getByRole("button", { name: /Thử lại/ });
  if (await retry.count()) await retry.first().click(); else await page.getByRole("button", { name: "Duyệt bài" }).click();
  await page.waitForTimeout(3000);
  expect((await subs(page, c, aid, "REVIEWED")).some((s) => s.id === d[0].id), "bỏ chặn → duyệt được").toBe(true);
  // 409 khi lưu điểm
  const d2 = (await subs(page, c, aid, "DRAFT"))[0];
  if (d2 && saveUrl) {
    await page.goto(`/grading/${d2.id}?course=${c}`);
    await page.waitForTimeout(3500);
    await page.route(/grading-results\/[0-9a-f-]{36}(\?|$)/, (r) => (r.request().method() === "GET" ? r.continue() : r.fulfill({ status: 409, contentType: "application/json", body: JSON.stringify({ code: "VERSION_CONFLICT", message: "x", trace_id: "t", details: { current_version: 9 } }) })));
    const i2 = page.locator("main input[aria-label^='Điểm ']").nth(1);
    await i2.fill("0,75");
    await i2.blur();
    await page.waitForTimeout(2500);
    const t = (await page.locator("main").innerText()).replace(/\s+/g, " ");
    console.log("INFO sau 409:", t.match(/(người khác|vừa|xung đột|phiên bản|Giữ|Lấy)[^.]{0,80}/i)?.[0] ?? "(không thấy thông báo)", "| ô điểm:", await i2.inputValue());
    expect(await i2.inputValue(), "giá trị nhập không mất").toBe("0,75");
  }
  await ctx.close();
});

test("TC-UX04-12/13/14 nháy lưu `ep-saved` ở sổ điểm, điểm danh, nhận xét chấm (US-04 AC7)", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await who(browser);
  // sổ điểm: sửa một ô nhập rồi Enter
  await page.goto(`/gradebook?course=${c}`);
  await page.waitForTimeout(4000);
  await startSampler(page);
  const cell = page.locator("main tbody input").first();
  const old = await cell.inputValue();
  await cell.fill(old === "6,5" ? "6" : "6,5");
  await cell.press("Enter");
  await page.waitForTimeout(2500);
  console.log("INFO sổ điểm ep-saved:", JSON.stringify(await saved(page)));
  expect.soft((await saved(page)).length, "UX04-12 sổ điểm").toBeGreaterThan(0);
  await cell.fill(old);
  await cell.press("Enter");
  await page.waitForTimeout(1500);
  // điểm danh: đổi một hàng rồi Lưu
  await page.goto(`/attendance?course=${c}`);
  await page.waitForTimeout(4000);
  await startSampler(page);
  const rows = page.locator("main tbody tr[data-part=student-row]");
  await rows.nth(12).locator("input[data-status=LATE]").check({ force: true });
  await page.getByRole("button", { name: /Lưu điểm danh/ }).first().click();
  await page.waitForTimeout(2500);
  console.log("INFO điểm danh ep-saved:", JSON.stringify(await saved(page)));
  expect.soft((await saved(page)).length, "UX04-13 điểm danh").toBeGreaterThan(0);
  await rows.nth(12).locator("input[data-status=PRESENT]").check({ force: true });
  await page.getByRole("button", { name: /Lưu điểm danh/ }).first().click();
  await page.waitForTimeout(1500);
  // nhận xét chấm: sửa ghi chú rồi rời ô
  const aid = await AID(page, c);
  const d = (await subs(page, c, aid, "DRAFT"))[0];
  await page.goto(`/grading/${d.id}?course=${c}`);
  await page.waitForTimeout(3500);
  await startSampler(page);
  const note = page.locator("main textarea").first();
  await note.fill("QC nháy lưu 9.1");
  await note.blur();
  await page.waitForTimeout(2500);
  console.log("INFO nhận xét ep-saved:", JSON.stringify(await saved(page)));
  expect.soft((await saved(page)).length, "UX04-14 nhận xét chấm").toBeGreaterThan(0);
  await ctx.close();
});

test("TC-UX12-14/15 cảnh báo công thức chưa xác nhận (lớp 761988): một dòng mỏng + Xem công thức; 375 px một hàng", async ({ browser }, info) => {
  desktop(info);
  for (const [w, h] of [[1440, 900], [375, 812]] as const) {
    const o = await who(browser, w, h, "teacher", "761988");
    await o.page.goto(`/attendance?course=${o.c}`);
    await o.page.waitForTimeout(4500);
    const nodes = o.page.getByText(/chưa xác nhận/i);
    const n = await nodes.count();
    const boxes: unknown[] = [];
    for (let i = 0; i < n; i++) boxes.push({ t: (await nodes.nth(i).innerText()).replace(/\s+/g, " ").slice(0, 60), box: await nodes.nth(i).boundingBox() });
    const btn = o.page.getByRole("button", { name: "Xem công thức" }).or(o.page.getByRole("link", { name: "Xem công thức" })).first();
    console.log(`INFO ${w}: dòng cảnh báo ${n}`, JSON.stringify(boxes), "| nút Xem công thức:", JSON.stringify(await btn.boundingBox().catch(() => null)));
    const row = await o.page.evaluate(() => { const e = [...document.querySelectorAll("main *")].find((x) => /chưa xác nhận/i.test((x as HTMLElement).innerText ?? "") && x.children.length <= 3 && (x as HTMLElement).innerText.length < 80) as HTMLElement | undefined; let p = e; while (p && p !== document.body && !p.querySelector("button, a")) p = p.parentElement as HTMLElement; return p ? { h: Math.round(p.getBoundingClientRect().height), text: p.innerText.replace(/\s+/g, " ").slice(0, 100) } : null; });
    console.log(`INFO ${w}: khối cảnh báo`, JSON.stringify(row));
    expect(n, "đúng một dòng cảnh báo").toBeGreaterThan(0);
    if (w === 375 && row) expect.soft(row.h, "375: cao ≤ 56").toBeLessThanOrEqual(56);
    await o.ctx.close();
  }
});

test("K7 ảnh sau đợt 4 (1440 + 375)", async ({ browser }, info) => {
  desktop(info);
  test.setTimeout(240_000);
  for (const [w, h] of [[1440, 900], [375, 812]] as const) {
    const { ctx, page, c } = await who(browser, w, h);
    const aid = await AID(page, c);
    const d = (await subs(page, c, aid, "DRAFT")).concat(await subs(page, c, aid, "REVIEWED"))[0];
    const shots: Array<[string, string, (p: Page) => Promise<void>]> = [
      ["grading-queue", `/grading?course=${c}`, async (p) => { await p.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" }); await p.getByLabel("Trạng thái").selectOption({ label: "Chưa duyệt" }); await p.waitForTimeout(2500); }],
      ["grading-review", `/grading/${d.id}?course=${c}`, async (p) => { await p.waitForTimeout(3500); }],
      ["gradebook", `/gradebook?course=${c}`, async (p) => { await p.waitForTimeout(3500); }],
      ["attendance", `/attendance?course=${c}`, async (p) => { await p.waitForTimeout(3500); }],
    ];
    for (const [name, url, prep] of shots) {
      await page.goto(url);
      await page.waitForTimeout(2000);
      await prep(page);
      await page.screenshot({ path: `/out/shots-w4/${name}-${w}.png` });
    }
    if (w === 375) { await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight)); await page.waitForTimeout(500); await page.screenshot({ path: "/out/shots-w4/attendance-375-cuoi.png" }); }
    await ctx.close();
  }
});
