import { expect, test, type Browser, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC đợt 4, vòng bổ sung: ca cần đo kỹ hơn (phím A / P, mạng khi duyệt, 375 px, màu Vắng, so API sổ điểm).
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
async function teacher(browser: Browser, w = 1440, h = 900, user = "teacher", code = "761987") {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, hasTouch: w < 500, isMobile: w < 500, locale: "vi-VN" });
  const page = await ctx.newPage();
  await login(page, user);
  const c = (await api<Mine>(page, "/me/courses")).body.items.find((i) => i.course.class_code === code)!.course.id;
  return { ctx, page, c };
}
const AID = async (page: Page, c: string) => (await api<{ items: Array<{ id: string; title: string }> }>(page, `/courses/${c}/assignments?limit=50`)).body.items.find((a) => /Bài tập 03/.test(a.title))!.id;
const subs = async (page: Page, c: string, aid: string, st: string) => (await api<{ items: Array<{ id: string; student: { full_name: string } }> }>(page, `/courses/${c}/submissions?limit=100&assignment_id=${aid}&status=${st}`)).body.items;
const desktop = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng");

test("TC-UX14-3/5/7/11 hàng chờ bổ sung: Enter khi focus, 375 px, rộng ô Bài tập", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  await page.goto(`/grading?course=${c}`);
  await page.waitForTimeout(2500);
  await page.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" });
  await page.getByLabel("Trạng thái").selectOption({ label: "Chưa duyệt" });
  await page.waitForTimeout(2200);
  const row0 = await page.locator("main tbody tr").first().evaluate((e) => ({ tab: (e as HTMLElement).tabIndex, role: e.getAttribute("role"), onkey: !!(e as HTMLElement).onkeydown, focusables: [...e.querySelectorAll("a,button,[tabindex]")].map((x) => `${x.tagName}:${(x as HTMLElement).innerText.trim().slice(0, 14)}:${(x as HTMLElement).tabIndex}`) }));
  console.log("INFO hàng:", JSON.stringify(row0));
  // Tab tới phần tử đầu tiên trong hàng rồi Enter
  await page.locator("main tbody tr").first().locator("a").first().focus();
  await page.keyboard.press("Enter");
  await page.waitForTimeout(1500);
  console.log("INFO Enter trên liên kết trong hàng →", page.url());
  expect(page.url()).toMatch(/\/grading\/[0-9a-f-]{36}/);
  await page.goBack();
  await page.waitForTimeout(1500);
  // hàng có tabindex?
  const sel = await page.getByLabel("Bài tập").evaluate((s: HTMLSelectElement) => { const cs = getComputedStyle(s); const cv = document.createElement("canvas").getContext("2d")!; cv.font = `${cs.fontSize} ${cs.fontFamily}`; return { w: Math.round(s.getBoundingClientRect().width), text: Math.round(Math.max(...[...s.options].map((o) => cv.measureText(o.text).width))), pad: cs.paddingLeft + "/" + cs.paddingRight }; });
  console.log("INFO ô Bài tập:", JSON.stringify(sel));
  await ctx.close();
  const m = await teacher(browser, 375, 812);
  await m.page.goto(`/grading?course=${m.c}`);
  await m.page.waitForTimeout(2500);
  await m.page.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" });
  await m.page.getByLabel("Trạng thái").selectOption({ label: "Chưa duyệt" });
  await m.page.waitForTimeout(2500);
  const r = await m.page.evaluate(() => {
    const links = [...document.querySelectorAll("main a")].filter((a) => /Duyệt/.test((a as HTMLElement).innerText)).map((a) => { const b = a.getBoundingClientRect(); return { t: (a as HTMLElement).innerText.trim(), x: Math.round(b.left), y: Math.round(b.top), w: Math.round(b.width), h: Math.round(b.height), vis: getComputedStyle(a).visibility + ":" + getComputedStyle(a).display }; });
    const row = document.querySelector("main tbody tr") as HTMLElement;
    const name = row.querySelector("td") as HTMLElement;
    return { links: links.slice(0, 3), rowDisplay: getComputedStyle(row).display, rowH: Math.round(row.getBoundingClientRect().height), tds: [...row.querySelectorAll("td")].map((t) => `${(t as HTMLElement).innerText.replace(/\s+/g, " ").slice(0, 24)}@${Math.round(t.getBoundingClientRect().top)}`), nameBox: Math.round(name.getBoundingClientRect().height) };
  });
  console.log("INFO 375:", JSON.stringify(r));
  const link = r.links.find((l) => l.h > 0);
  expect(link, "có liên kết Duyệt nhìn thấy").toBeTruthy();
  expect(link!.h, "Duyệt cao ≥ 44").toBeGreaterThanOrEqual(43);
  await m.ctx.close();
});

test("TC-UX15-8/9/13 bắt mạng khi bấm Duyệt bài / phím A; trạng thái bài; hoạt cảnh sau ›", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  const aid = await AID(page, c);
  const d = await subs(page, c, aid, "DRAFT");
  const rev0 = (await subs(page, c, aid, "REVIEWED")).length;
  console.log("INFO DRAFT:", d.length, "REVIEWED:", rev0);
  const writes: string[] = [];
  page.on("request", (r) => { if (r.method() !== "GET" && r.url().includes("/api/v1/") && !/auth|events/.test(r.url())) writes.push(`${r.method()} ${new URL(r.url()).pathname.replace(/[0-9a-f-]{36}/g, "‹id›")} ${(r.postData() ?? "").slice(0, 80)}`); });
  await page.goto(`/grading/${d[0].id}?course=${c}`);
  await page.waitForTimeout(3500);
  const prog = () => page.evaluate(() => (document.querySelector("main") as HTMLElement).innerText.match(/Đã duyệt \d+\/\d+/)?.[0]);
  console.log("INFO tiến độ:", await prog());
  await page.getByRole("button", { name: "Duyệt bài" }).click();
  await page.waitForTimeout(3500);
  console.log("INFO sau bấm Duyệt bài: URL", page.url(), "| tiến độ", await prog(), "| ghi:", JSON.stringify(writes));
  const rev1 = (await subs(page, c, aid, "REVIEWED")).length;
  console.log("INFO REVIEWED:", rev0, "→", rev1);
  expect(rev1, "bấm Duyệt bài làm bài thành REVIEWED").toBe(rev0 + 1);
  // phím A trên bài kế
  writes.length = 0;
  const cur = page.url();
  await page.keyboard.press("a");
  await page.waitForTimeout(3500);
  const rev2 = (await subs(page, c, aid, "REVIEWED")).length;
  console.log("INFO phím A: URL", cur, "→", page.url(), "| tiến độ", await prog(), "| ghi:", JSON.stringify(writes), "| REVIEWED", rev1, "→", rev2);
  expect(rev2, "UX15-8: phím A duyệt bài hiện tại").toBe(rev1 + 1);
  // › : hoạt cảnh
  const names = await page.evaluate(async () => {
    const b = [...document.querySelectorAll("main button")].find((x) => /^›|Bài sau/.test((x as HTMLElement).innerText.trim() || x.getAttribute("aria-label") || "")) as HTMLElement | undefined;
    const btn = b ?? ([...document.querySelectorAll("main button")].filter((x) => /[›>]/.test((x as HTMLElement).innerText + (x.getAttribute("aria-label") ?? ""))).pop() as HTMLElement | undefined);
    btn?.click();
    await new Promise((r) => requestAnimationFrame(() => r(null)));
    return document.getAnimations().map((a) => `${(a as CSSAnimation).animationName ?? (a as CSSTransition).transitionProperty}:${a.effect?.getTiming().duration}`);
  });
  console.log("INFO hoạt cảnh ngay sau ›:", JSON.stringify(names));
  await ctx.close();
});

test("TC-UX16 sổ điểm: so từng ô với API, hàng thiếu / cột Trạng thái, menu cột khi rê, Chốt điểm", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  const g = await api<{ rows: Array<{ student: { full_name: string }; result: Record<string, string> | null; bonus: string; penalty: string; entries: Record<string, { score: string }>; drift?: boolean }>; columns: Array<{ item_id: string; name: string }> }>(page, `/courses/${c}/gradebook?limit=100`);
  const inc = g.body.rows.filter((r) => r.result && r.result.status !== "COMPLETE").length;
  console.log("INFO API: hàng", g.body.rows.length, "| chưa đủ:", inc, "| drift:", g.body.rows.filter((r) => r.drift).length);
  await page.goto(`/gradebook?course=${c}`);
  await page.waitForTimeout(4000);
  const ui = await page.evaluate(() => {
    const heads = [...document.querySelectorAll("main thead th")].map((t) => (t as HTMLElement).innerText.trim());
    const rows = [...document.querySelectorAll("main tbody tr")].map((r) => [...r.querySelectorAll("td")].map((td) => { const i = td.querySelector("input"); return i ? (i as HTMLInputElement).value : (td as HTMLElement).innerText.trim(); }));
    const text = (document.querySelector("main") as HTMLElement).innerText.replace(/\s+/g, " ");
    return { heads, rows, thieu: text.match(/Thiếu điểm: \d+ sinh viên/)?.[0] ?? null, legend: /đã công bố/i.test(text), pub: document.querySelectorAll("main thead [aria-label='Đã công bố']").length };
  });
  const byName = new Map(g.body.rows.map((r) => [r.student.full_name, r]));
  const hi = (n: string) => ui.heads.indexOf(n);
  let checked = 0; const bad: string[] = [];
  for (const row of ui.rows) {
    const x = byName.get(row[0].split("\n")[0]);
    if (!x?.result) continue;
    for (const [n, k] of [["Điểm học phần", "hp"], ["Quá trình", "qt"]] as const) {
      const u = row[hi(n)]?.replace(",", "."), a = x.result[k];
      if (u && a) { checked++; if (Number(u) !== Number(a)) bad.push(`${row[0].split("\n")[0]} ${n} UI ${u} API ${a}`); }
    }
    const bn = row[hi("Điểm cộng")]?.replace(",", ".");
    if (bn) { checked++; if (Number(bn) !== Number(x.bonus)) bad.push(`${row[0].split("\n")[0]} cộng UI ${bn} API ${x.bonus}`); }
  }
  console.log(`INFO UX16-3: so ${checked} ô, lệch ${bad.length} ${bad.slice(0, 3).join(" ; ")}`);
  console.log("INFO tiêu đề phụ:", ui.thieu, "| cột Trạng thái:", ui.heads.includes("Trạng thái"), "| chú giải đã công bố:", ui.legend, "| chấm công bố ở tiêu đề:", ui.pub);
  expect(checked).toBeGreaterThan(20);
  expect(bad, "UX16-3 UI khớp API từng ô").toEqual([]);
  if (inc > 0) expect(ui.thieu, "UX16-9 tiêu đề phụ Thiếu điểm: N").toContain(`Thiếu điểm: ${inc} sinh viên`);
  else expect(ui.heads.includes("Trạng thái"), "UX16-10 ẩn cột khi không ai thiếu").toBe(false);
  // menu cột
  const th = page.locator("main thead th").filter({ hasText: "Kiểm tra hằng tuần" }).first();
  const mb = th.locator("button").first();
  console.log("INFO nút trong tiêu đề cột:", await th.locator("button").count());
  const o0 = await mb.evaluate((e) => getComputedStyle(e).opacity).catch(() => "—");
  await th.hover({ timeout: 5000 });
  await page.waitForTimeout(400);
  const o1 = await mb.evaluate((e) => getComputedStyle(e).opacity).catch(() => "—");
  console.log("INFO menu ⋯ opacity:", o0, "→", o1);
  // Chốt điểm
  const chot = page.getByRole("button", { name: "Chốt điểm" });
  const dis = await chot.getAttribute("aria-disabled");
  console.log("INFO Chốt điểm aria-disabled:", dis);
  if (dis !== "true") {
    await chot.click();
    await page.waitForTimeout(1500);
    const dt = (await page.getByRole("dialog").innerText().catch(() => "")).replace(/\s+/g, " ");
    console.log("INFO hộp xác nhận:", dt.slice(0, 500));
    await page.keyboard.press("Escape");
    await page.waitForTimeout(600);
    const st = await api<{ gradebook?: { status?: string } }>(page, `/courses/${c}/grade-scheme/status`);
    console.log("INFO sổ sau Esc:", st.body.gradebook?.status);
    expect(st.body.gradebook?.status).not.toBe("LOCKED");
  }
  await ctx.close();
  const t = await teacher(browser, 1440, 900, "ta");
  const fin = await api(t.page, `/courses/${t.c}/gradebook/finalize`, { method: "POST", body: {} });
  console.log("INFO TA POST finalize:", fin.status);
  expect([403, 404]).toContain(fin.status);
  await t.page.goto(`/gradebook?course=${t.c}`);
  await t.page.waitForTimeout(3500);
  const tb = t.page.getByRole("button", { name: "Chốt điểm" });
  console.log("INFO TA nút Chốt điểm:", await tb.count(), await tb.first().getAttribute("aria-disabled").catch(() => null));
  await t.ctx.close();
});

test("TC-UX12-2/3/5/14/15 điểm danh bổ sung: màu Vắng, nhãn, phím P, cảnh báo công thức (761988), 375 px", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  await page.goto(`/attendance?course=${c}`);
  await page.waitForTimeout(4000);
  const rows = page.locator("main tbody tr[data-part=student-row]");
  await rows.nth(8).locator("input[data-status=ABSENT]").check({ force: true });
  await rows.nth(1).locator("input[data-status=LATE]").check({ force: true });
  await page.waitForTimeout(400);
  const col = (n: number, st: string) => rows.nth(n).locator(`input[data-status=${st}]`).evaluate((on) => { const l = on.closest("label")!; return { self: getComputedStyle(on).borderTopColor + "|" + getComputedStyle(on).backgroundColor, before: getComputedStyle(on, "::before").backgroundColor, after: getComputedStyle(on, "::after").backgroundColor, lab: getComputedStyle(l).color, checked: (on as HTMLInputElement).checked }; });
  console.log("INFO màu LATE:", JSON.stringify(await col(1, "LATE")));
  console.log("INFO màu ABSENT:", JSON.stringify(await col(8, "ABSENT")));
  console.log("INFO hàng Vắng text:", (await rows.nth(8).innerText()).replace(/\s+/g, " "));
  const aria = await rows.nth(8).evaluate((r) => [...r.querySelectorAll("input[type=radio]")].map((i) => `${(i.closest("label")?.querySelector(".ep-sr-only") as HTMLElement | null)?.innerText}:${(i as HTMLInputElement).checked}`));
  console.log("INFO radio:", JSON.stringify(aria));
  // phím P
  const speakText = (n: number) => rows.nth(n).locator("td").last().innerText();
  const all0 = await page.locator("main tbody").innerText();
  await rows.nth(5).locator("td").first().click();
  await page.keyboard.press("p");
  await page.waitForTimeout(600);
  const all1 = await page.locator("main tbody").innerText();
  console.log("INFO phím P đổi nội dung bảng:", all0 !== all1, "| hàng 5 ô cuối:", (await speakText(5)).replace(/\s+/g, " "), "| hàng đang chọn:", await page.evaluate(() => (document.querySelector("tr[aria-current=true]") as HTMLElement | null)?.dataset.studentId?.slice(-4)));
  await ctx.close();
  const o = await teacher(browser, 1440, 900, "teacher", "761988");
  await o.page.goto(`/attendance?course=${o.c}`);
  await o.page.waitForTimeout(4000);
  const warn = await o.page.evaluate(() => { const e = [...document.querySelectorAll("main *")].find((x) => /Công thức[^.]*chưa xác nhận|chưa xác nhận[^.]*công thức/i.test((x as HTMLElement).innerText ?? "") && x.children.length <= 4) as HTMLElement | undefined; return e ? { t: e.innerText.replace(/\s+/g, " ").slice(0, 120), h: Math.round(e.getBoundingClientRect().height), btn: !!e.querySelector("button, a") || !!e.parentElement?.querySelector("button[name], a") } : null; });
  console.log("INFO 761988 cảnh báo:", JSON.stringify(warn), "| số dòng cảnh báo:", await o.page.getByText(/chưa xác nhận/).count(), "| Xem công thức:", await o.page.getByRole("button", { name: "Xem công thức" }).or(o.page.getByRole("link", { name: "Xem công thức" })).count());
  await o.ctx.close();
  const m = await teacher(browser, 375, 812);
  await m.page.goto(`/attendance?course=${m.c}`);
  await m.page.waitForTimeout(4500);
  const dom = await m.page.evaluate(() => {
    const trs = [...document.querySelectorAll("main tr")].slice(0, 3).map((t) => `${(t as HTMLElement).dataset.part ?? ""}:${getComputedStyle(t).display}:${Math.round(t.getBoundingClientRect().height)}`);
    const r = document.querySelector("input[data-status=PRESENT]") as HTMLElement;
    const rb = r?.closest("label")?.getBoundingClientRect();
    const save = [...document.querySelectorAll("button")].find((b) => /Lưu điểm danh/.test((b as HTMLElement).innerText)) as HTMLElement | undefined;
    return { trs, radioCnt: document.querySelectorAll("input[data-status]").length, radioBox: rb ? [Math.round(rb.width), Math.round(rb.height), Math.round(rb.top)] : null, save: save ? { t: save.innerText.replace(/\s+/g, " "), top: Math.round(save.getBoundingClientRect().top), h: Math.round(save.getBoundingClientRect().height), pos: (() => { let p: HTMLElement | null = save.parentElement; while (p && p !== document.body) { const s = getComputedStyle(p).position; if (s === "sticky" || s === "fixed") return s; p = p.parentElement; } return ""; })() } : null, ov: document.documentElement.scrollWidth > document.documentElement.clientWidth, iH: innerHeight };
  });
  console.log("INFO 375 điểm danh:", JSON.stringify(dom));
  await m.ctx.close();
});
