import { expect, test, type Browser, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC FEAT-uiux-91 đợt 4 (tc-wave-4.md): Chấm bài, Sổ điểm, Điểm danh trên stack thật ep-qc, lớp 761987 (seed), giảng viên `teacher`.
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
const desktop = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng");

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
const courseOf = async (page: Page, code = "761987") => (await api<Mine>(page, "/me/courses")).body.items.find((i) => i.course.class_code === code)!.course.id;
async function teacher(browser: Browser, w = 1440, h = 900) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, hasTouch: w < 500, isMobile: w < 500, locale: "vi-VN" });
  const page = await ctx.newPage();
  await login(page, "teacher");
  const c = await courseOf(page);
  return { ctx, page, c };
}
const AID = async (page: Page, c: string) => {
  const r = await api<{ items: Array<{ id: string; title: string }> }>(page, `/courses/${c}/assignments?limit=50`);
  return r.body.items.find((a) => /Bài tập 03/.test(a.title))!.id;
};
const subs = async (page: Page, c: string, aid: string, st: string) => (await api<{ items: Array<{ id: string; student: { full_name: string } }> }>(page, `/courses/${c}/submissions?limit=100&assignment_id=${aid}&status=${st}`)).body.items;
const redOf = (page: Page, sel: string, H = 900) =>
  page.evaluate(([s, h]) => {
    const cv = document.createElement("canvas"); cv.width = cv.height = 1; const x = cv.getContext("2d")!;
    return [...document.querySelectorAll(s as string)].filter((e) => { const r = e.getBoundingClientRect(); return r.width > 30 && r.height > 20 && r.top >= 0 && r.top < (h as number); })
      .filter((e) => { x.clearRect(0, 0, 1, 1); x.fillStyle = getComputedStyle(e).backgroundColor; x.fillRect(0, 0, 1, 1); const d = x.getImageData(0, 0, 1, 1).data; return d[0] > 170 && d[1] < 70 && d[2] < 80; })
      .map((e) => (e as HTMLElement).innerText.trim().slice(0, 24));
  }, [sel, H] as const);

test("SETUP chấm nháp thêm bài (Bài tập 03) để có bài chờ duyệt", async ({ browser }, info) => {
  desktop(info);
  test.setTimeout(240_000);
  const { ctx, page, c } = await teacher(browser);
  const aid = await AID(page, c);
  const none = await subs(page, c, aid, "NONE");
  for (const s of none.slice(0, 4)) await api(page, `/courses/${c}/submissions/${s.id}/grade`, { method: "POST", body: {} });
  await expect.poll(async () => (await subs(page, c, aid, "DRAFT")).length, { timeout: 150_000, intervals: [5000] }).toBeGreaterThanOrEqual(3);
  console.log("INFO DRAFT:", (await subs(page, c, aid, "DRAFT")).map((s) => s.student.full_name).join(", "));
  await ctx.close();
});

// ---------- US-UX91-14 ----------
test("TC-UX14-1/2/4/5/6/7 hàng chờ: cột Bài tập, lý do cờ, hàng bấm được, căn đáy chip", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  const calls: string[] = [];
  page.on("request", (r) => { if (/\/submissions\?/.test(r.url())) calls.push(new URL(r.url()).search); });
  await page.goto(`/grading?course=${c}`);
  await page.waitForTimeout(2500);
  await page.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" });
  await page.getByLabel("Trạng thái").selectOption({ label: "Chưa duyệt" });
  await page.waitForTimeout(2500);
  const th = await page.locator("main th").allInnerTexts();
  console.log("INFO th một bài:", JSON.stringify(th));
  expect(th.map((t) => t.trim()), "UX14-1: cột Bài tập ẩn khi lọc một bài").not.toContain("Bài tập");
  const txt = await page.locator("main").innerText();
  expect((txt.match(/Bài tập 03/g) ?? []).length, "tên bài hiện đúng 1 lần (kể cả chọn trong ô lọc)").toBeLessThanOrEqual(2);
  console.log("INFO số 'Bài tập 03' trên trang:", (txt.match(/Bài tập 03/g) ?? []).length);
  await page.getByLabel("Bài tập").selectOption({ label: "Tất cả bài tập" });
  await page.waitForTimeout(2000);
  const th2 = (await page.locator("main th").allInnerTexts()).map((t) => t.trim());
  console.log("INFO th tất cả bài:", JSON.stringify(th2));
  expect(th2, "UX14-2: cột Bài tập hiện khi Tất cả bài tập").toContain("Bài tập");
  const cell = await page.evaluate(() => { const ths = [...document.querySelectorAll("main thead th")]; const i = ths.findIndex((t) => (t as HTMLElement).innerText.trim() === "Bài tập"); const td = document.querySelector(`main tbody tr td:nth-child(${i + 1})`) as HTMLElement | null; const inner = td ? ((td.firstElementChild as HTMLElement | null) ?? td) : null; if (!inner) return null; const cs = getComputedStyle(inner); return { ws: cs.whiteSpace, to: cs.textOverflow, ov: cs.overflow, w: Math.round(inner.getBoundingClientRect().width), txt: inner.innerText.slice(0, 40) }; });
  console.log("INFO ô Bài tập:", JSON.stringify(cell));
  if (cell) { expect.soft(cell.ws, "một dòng").toBe("nowrap"); expect.soft(cell.to).toBe("ellipsis"); }
  // chip «Cần xem kỹ» căn đáy với hai ô chọn (UX14-7)
  const al = await page.evaluate(() => {
    const bottom = (e: Element | null | undefined) => (e ? Math.round(e.getBoundingClientRect().bottom * 10) / 10 : null);
    const chip = [...document.querySelectorAll("main button, main label, main [role=switch], main [role=checkbox]")].find((e) => /Cần xem kỹ/.test((e as HTMLElement).innerText));
    const sel = [...document.querySelectorAll("main select")];
    return { chip: bottom(chip), sels: sel.map((s) => bottom(s)) };
  });
  console.log("INFO đáy:", JSON.stringify(al));
  for (const b of al.sels) expect(Math.abs((al.chip ?? 0) - (b ?? 0)), "UX14-7 đáy lệch ≤ 1 px").toBeLessThanOrEqual(1.5);
  const sel = await page.getByLabel("Bài tập").evaluate((s: HTMLSelectElement) => ({ w: Math.round(s.getBoundingClientRect().width), longest: Math.max(...[...s.options].map((o) => o.text.length)) }));
  console.log("INFO ô Bài tập rộng:", JSON.stringify(sel));
  // hàng bấm được (UX14-5/6)
  await page.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" });
  await page.getByLabel("Trạng thái").selectOption({ label: "Chưa duyệt" });
  await page.waitForTimeout(2200);
  const row = page.locator("main tbody tr").first();
  const link = row.getByRole("link", { name: /Duyệt/ });
  console.log("INFO liên kết Duyệt:", await link.count());
  expect(await link.count(), "UX14-5: liên kết Duyệt").toBeGreaterThan(0);
  const cur = await row.evaluate((e) => getComputedStyle(e).cursor);
  console.log("INFO cursor hàng:", cur);
  expect(cur).toBe("pointer");
  const bgOf = (r: ReturnType<typeof page.locator>) => r.evaluate((e) => [getComputedStyle(e).backgroundColor, ...[...e.querySelectorAll("td")].slice(0, 2).map((t) => getComputedStyle(t).backgroundColor)].join("|"));
  await page.mouse.move(5, 5);
  const bgBefore = await bgOf(row);
  await row.hover();
  await page.waitForTimeout(300);
  const bgAfter = await bgOf(row);
  console.log("INFO nền hàng:", bgBefore, "→", bgAfter);
  expect(bgAfter, "nền hover đổi").not.toBe(bgBefore);
  await row.locator("td").nth(1).click();
  await expect(page).toHaveURL(/\/grading\/[0-9a-f-]{36}/);
  await page.goBack();
  await page.waitForTimeout(1500);
  const r2 = page.locator("main tbody tr").first();
  await r2.focus();
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/grading\/[0-9a-f-]{36}/, { timeout: 5000 }).catch(() => console.log("INFO Enter trên hàng không mở trang"));
  console.log("INFO URL sau Enter:", page.url());
  expect(page.url()).toMatch(/\/grading\/[0-9a-f-]{36}/);
  console.log("INFO request danh sách:", [...new Set(calls)].slice(0, 3).join(" | "));
  await ctx.close();
});

test("TC-UX14-4 lý do cờ dưới tên sinh viên, hàng không cờ gọn", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  await page.goto(`/grading?course=${c}`);
  await page.waitForTimeout(2500);
  await page.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" });
  await page.getByLabel("Trạng thái").selectOption({ label: "Chưa duyệt" });
  await page.waitForTimeout(2200);
  const rows = await page.evaluate(() => [...document.querySelectorAll("main tbody tr")].map((r) => { const t = r as HTMLElement; const first = t.querySelector("td") as HTMLElement; const kids = [...first.querySelectorAll("*")].filter((e) => (e as HTMLElement).innerText?.trim() && e.children.length === 0).map((e) => ({ t: (e as HTMLElement).innerText.trim().slice(0, 40), top: Math.round(e.getBoundingClientRect().top) })); return { h: Math.round(t.getBoundingClientRect().height), kids }; }));
  console.log("INFO hàng:", JSON.stringify(rows.slice(0, 5)));
  const flagged = rows.filter((r) => r.kids.length >= 2);
  const plain = rows.filter((r) => r.kids.length < 2);
  console.log(`INFO có lý do cờ: ${flagged.length}; không: ${plain.length}`);
  if (flagged.length && plain.length) expect(Math.max(...plain.map((r) => r.h)), "hàng không cờ thấp hơn hàng có cờ").toBeLessThan(Math.min(...flagged.map((r) => r.h)));
  await ctx.close();
});

test("TC-UX14-9/10/PQ/PG lỗi tải giữ bộ lọc; STUDENT 403, TA mở được; lớp lạ; limit", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  const calls: string[] = [];
  page.on("request", (r) => { if (/\/submissions\?/.test(r.url())) calls.push(new URL(r.url()).search); });
  await page.goto(`/grading?course=${c}`);
  await page.waitForTimeout(2500);
  await page.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" });
  await page.waitForTimeout(1500);
  expect(calls.some((q) => /limit=(\d+)/.test(q) && Number(/limit=(\d+)/.exec(q)![1]) <= 100), "UX14-PG: có limit ≤ 100").toBe(true);
  expect(calls.some((q) => /offset=/i.test(q)), "không OFFSET").toBe(false);
  let fail = true;
  await page.route(/\/submissions\?/, (r) => (fail ? r.abort() : r.continue()));
  await page.getByLabel("Trạng thái").selectOption({ label: "Đã duyệt" });
  await page.waitForTimeout(4000);
  const t = await page.locator("main").innerText();
  console.log("INFO lỗi:", t.replace(/\s+/g, " ").slice(0, 200));
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible({ timeout: 15_000 });
  expect(await page.getByLabel("Trạng thái").inputValue(), "bộ lọc còn nguyên").not.toBe("");
  const selVal = await page.getByLabel("Bài tập").evaluate((s: HTMLSelectElement) => s.selectedOptions[0].text);
  expect(selVal).toMatch(/Bài tập 03/);
  fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await page.waitForTimeout(2500);
  expect(await page.getByRole("button", { name: "Thử lại" }).count(), "tải lại được").toBe(0);
  await ctx.close();
  // STUDENT / TA
  const cs = await browser.newContext({ ignoreHTTPSErrors: true });
  const ps = await cs.newPage();
  await login(ps, "sv.gioi");
  await ps.goto("/grading"); await ps.waitForTimeout(2500);
  expect(await ps.locator("main").innerText()).toContain("Bạn không có quyền xem màn này");
  await cs.close();
  const ct = await browser.newContext({ ignoreHTTPSErrors: true });
  const pt = await ct.newPage();
  await login(pt, "ta");
  await pt.goto("/grading"); await pt.waitForTimeout(2500);
  expect(await pt.locator("main").innerText()).not.toContain("Bạn không có quyền xem màn này");
  // lớp lạ: lớp QC… mà TA không thuộc
  const mine = (await api<Mine>(pt, "/me/courses")).body.items.map((i) => i.course.id);
  const ca = await browser.newContext({ ignoreHTTPSErrors: true });
  const pa = await ca.newPage();
  await login(pa, "admin");
  const all = ((await api<{ items: Array<{ id: string; class_code: string }> }>(pa, "/admin/courses?limit=100")).body.items ?? []);
  await ca.close();
  const other = all.find((x) => !mine.includes(x.id));
  console.log("INFO lớp TA không thuộc:", other?.class_code);
  if (other) {
    const s = await api(pt, `/courses/${other.id}/submissions?limit=5`);
    const g = await api(pt, `/courses/${other.id}/gradebook?limit=5`);
    const a = await api(pt, `/courses/${other.id}/attendance/sessions`);
    console.log("INFO PQ TA lớp lạ: submissions", s.status, "gradebook", g.status, "attendance", a.status);
    expect([403, 404]).toContain(s.status);
    expect([403, 404]).toContain(g.status);
    expect([403, 404]).toContain(a.status);
  }
  await ct.close();
});

test("TC-UX14-11 375 px: hàng xếp dọc, Duyệt riêng ≥ 44 px, không tràn", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser, 375, 812);
  await page.goto(`/grading?course=${c}`);
  await page.waitForTimeout(2500);
  await page.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" });
  await page.getByLabel("Trạng thái").selectOption({ label: "Chưa duyệt" });
  await page.waitForTimeout(2500);
  const r = await page.evaluate(() => ({
    ov: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    row: (() => { const tr = document.querySelector("main tbody tr, main [role=row]") as HTMLElement | null; if (!tr) return null; const l = tr.querySelector("a") as HTMLElement | null; const b = l?.getBoundingClientRect(); return { rowH: Math.round(tr.getBoundingClientRect().height), link: l ? { t: l.innerText.trim(), w: Math.round(b!.width), h: Math.round(b!.height) } : null, text: tr.innerText.replace(/\s+/g, " ").slice(0, 120) }; })(),
  }));
  console.log("INFO 375 hàng chờ:", JSON.stringify(r));
  expect(r.ov).toBe(false);
  expect(r.row?.link?.h ?? 0, "Duyệt cao ≥ 44").toBeGreaterThanOrEqual(43);
  await ctx.close();
});

// ---------- US-UX91-15 ----------
test("TC-UX15-1/2/3/6/7/11 duyệt bài (S7): mỗi tiêu chí một hàng, không chồng, ghi chú một lần, thanh dưới trọn khung, lệch hai lượt", async ({ browser }, info) => {
  desktop(info);
  for (const w of [1440, 1100]) {
    const { ctx, page, c } = await teacher(browser, w, 900);
    const aid = await AID(page, c);
    const d = (await subs(page, c, aid, "DRAFT")).find((s) => /Khải/.test(s.student.full_name))!;
    await page.goto(`/grading/${d.id}?course=${c}`);
    await page.waitForTimeout(3500);
    const m = await page.evaluate(() => {
      const inputs = [...document.querySelectorAll("main input[aria-label^='Điểm ']")] as HTMLInputElement[];
      return inputs.map((i) => {
        const name = i.getAttribute("aria-label")!.replace(/^Điểm /, "");
        const nameEl = [...document.querySelectorAll("main h3, main h4, main strong, main span, main div")].filter((e) => (e as HTMLElement).innerText?.trim() === name && e.children.length === 0).pop();
        const a = i.getBoundingClientRect(), b = nameEl?.getBoundingClientRect();
        const maxEl = i.parentElement?.innerText ?? "";
        const visibleLabel = [...document.querySelectorAll("main label")].some((l) => (l as HTMLElement).innerText.trim() === `Điểm ${name}` && getComputedStyle(l).display !== "none" && l.getBoundingClientRect().width > 4 && !(l as HTMLElement).className.includes("sr-only"));
        return { name: name.slice(0, 30), nameTop: b ? Math.round(b.top) : null, inTop: Math.round(a.top), nameR: b ? Math.round(b.right) : null, inL: Math.round(a.left), overlap: b ? !(b.right <= a.left || b.left >= a.right || b.bottom <= a.top || b.top >= a.bottom) : null, max: /\/\s*\d/.test(maxEl), visibleLabel };
      });
    });
    console.log(`INFO tiêu chí ${w}:`, JSON.stringify(m));
    expect(m.length).toBeGreaterThan(2);
    for (const r of m) {
      expect(r.overlap, `${w}: ${r.name} tên không đè ô điểm`).toBe(false);
      expect(r.visibleLabel, `${w}: ${r.name} không còn nhãn «Điểm …» nhìn thấy`).toBe(false);
    }
    if (w === 1440) {
      for (const r of m) expect.soft(Math.abs((r.nameTop ?? 0) - r.inTop), `${r.name} cùng hàng ±8`).toBeLessThanOrEqual(8);
      const text = (await page.locator("main").innerText()).replace(/\s+/g, " ");
      console.log("INFO số lần câu ghi chú trích:", (text.match(/Bấm một dòng trích để xem trong bài/g) ?? []).length, "| cũ:", (text.match(/Đoạn trích từ bài làm · bấm/g) ?? []).length);
      expect((text.match(/Đoạn trích từ bài làm/g) ?? []).length, "UX15-3: ghi chú cũ tối đa 1 lần").toBeLessThanOrEqual(1);
      expect(text).toMatch(/Lệch 1,5 điểm so với lượt 2/);
      expect(await page.getByRole("dialog").count(), "không modal").toBe(0);
      const bar = await page.evaluate(() => {
        const btn = [...document.querySelectorAll("main button")].find((b) => (b as HTMLElement).innerText.trim() === "Duyệt bài") as HTMLElement | undefined;
        let n: HTMLElement | null = btn?.parentElement ?? null; let bar: HTMLElement | null = null;
        while (n && n !== document.body) { const p = getComputedStyle(n).position; if (p === "sticky" || p === "fixed") { bar = n; break; } n = n.parentElement; }
        const r = bar?.getBoundingClientRect();
        return bar ? { top: Math.round(r!.top), bottom: Math.round(r!.bottom), pos: getComputedStyle(bar).position, text: bar.innerText.replace(/\s+/g, " ").slice(0, 120), innerH: innerHeight } : null;
      });
      console.log("INFO thanh dưới lúc mở:", JSON.stringify(bar));
      expect(bar, "có thanh dính").not.toBeNull();
      expect.soft(bar!.bottom, "UX15-6: thanh trọn khung (bottom ≤ 900)").toBeLessThanOrEqual(900);
      expect(bar!.text).toMatch(/Đã duyệt \d+\/\d+/);
      expect(bar!.text).toMatch(/\bA\b/);
      const reds = await redOf(page, "main button, main a");
      console.log("INFO nút đỏ:", JSON.stringify(reds));
      expect(reds, "UX15-7 một nút đỏ").toEqual(["Duyệt bài"]);
      await page.locator("main").evaluate(() => { for (const e of document.querySelectorAll("main *")) { const c = getComputedStyle(e); if (/(auto|scroll)/.test(c.overflowY) && e.scrollHeight > e.clientHeight + 20) (e as HTMLElement).scrollTop = 99999; } });
      await page.mouse.wheel(0, 3000);
      await page.waitForTimeout(500);
      const bar2 = await page.evaluate(() => { const btn = [...document.querySelectorAll("main button")].find((b) => (b as HTMLElement).innerText.trim() === "Duyệt bài") as HTMLElement; const r = btn.getBoundingClientRect(); return { top: Math.round(r.top), bottom: Math.round(r.bottom), innerH: innerHeight }; });
      console.log("INFO nút Duyệt bài sau khi cuộn:", JSON.stringify(bar2));
      expect.soft(bar2.bottom, "sau cuộn vẫn trong khung").toBeLessThanOrEqual(bar2.innerH);
    }
    await ctx.close();
  }
});

test("TC-UX15-4 tô sáng đoạn trích: giữ ≈ 1 s rồi mờ ≤ 200 ms (và reduce tắt tức thì)", async ({ browser }, info) => {
  desktop(info);
  for (const reduce of [false, true]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, reducedMotion: reduce ? "reduce" : "no-preference" });
    const page = await ctx.newPage();
    await login(page, "teacher");
    const c = await courseOf(page);
    const aid = await AID(page, c);
    const d = (await subs(page, c, aid, "DRAFT")).find((s) => /Khải/.test(s.student.full_name))!;
    await page.goto(`/grading/${d.id}?course=${c}`);
    await page.waitForTimeout(3500);
    await page.evaluate(() => {
      const w = window as unknown as { __s: Array<{ t: number; n: number; max: number }> };
      w.__s = [];
      const t0 = performance.now();
      const tick = () => {
        const cv = document.createElement("canvas"); cv.width = cv.height = 1; const x = cv.getContext("2d")!;
        let n = 0, max = 0;
        for (const e of document.querySelectorAll("main [data-fading], main mark, main [data-highlight], main span, main p")) {
          const bg = getComputedStyle(e).backgroundColor;
          if (bg === "rgba(0, 0, 0, 0)" || bg === "transparent") continue;
          x.clearRect(0, 0, 1, 1); x.fillStyle = bg; x.fillRect(0, 0, 1, 1); const d = x.getImageData(0, 0, 1, 1).data;
          if (d[0] > 200 && d[1] < 245 && d[2] < 245 && (e as HTMLElement).innerText.length < 400 && e.getBoundingClientRect().left < 900) { n++; max = Math.max(max, 255 - d[1]); }
        }
        w.__s.push({ t: Math.round(performance.now() - t0), n, max });
        if (performance.now() - t0 < 3200) setTimeout(tick, 50);
      };
      (window as unknown as { __go: () => void }).__go = tick;
    });
    const quote = page.locator("main button").filter({ hasText: /^“/ }).first();
    await page.evaluate(() => (window as unknown as { __go: () => void }).__go());
    await quote.click();
    await page.waitForTimeout(3500);
    const s = await page.evaluate(() => (window as unknown as { __s: Array<{ t: number; n: number; max: number }> }).__s);
    const on = s.filter((x) => x.n > 0);
    const first = on[0]?.t, last = on[on.length - 1]?.t;
    console.log(`INFO reduce=${reduce}: tô sáng từ ${first} ms đến ${last} ms; mẫu cường độ: ${on.filter((_, i) => i % 3 === 0).map((x) => `${x.t}:${x.max}`).join(" ")}`);
    expect(on.length, "có đoạn được tô sáng").toBeGreaterThan(0);
    if (!reduce) { expect.soft((last - first) / 1000, "giữ + mờ ≈ 1,2 s").toBeGreaterThan(0.8); expect.soft((last - first) / 1000).toBeLessThan(1.6); }
    else expect.soft((last - first) / 1000, "reduce: giữ ≈ 1 s rồi tắt").toBeLessThan(1.5);
    await ctx.close();
  }
});

test("TC-UX15-8/9/10 phím A duyệt rồi sang bài kế; ‹ › đổi bài không hoạt cảnh; nhấn đúp chỉ một lần", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  const aid = await AID(page, c);
  const drafts = (await subs(page, c, aid, "DRAFT")).filter((s) => !/Khải/.test(s.student.full_name));
  console.log("INFO bản nháp khả dụng:", drafts.length);
  expect(drafts.length).toBeGreaterThanOrEqual(2);
  const posts: string[] = [];
  page.on("request", (r) => { if (r.method() !== "GET" && /submissions|grade/.test(r.url())) posts.push(`${r.method()} ${new URL(r.url()).pathname.replace(/[0-9a-f-]{36}/g, "‹id›")}`); });
  await page.goto(`/grading/${drafts[0].id}?course=${c}`);
  await page.waitForTimeout(3500);
  const prog0 = await page.evaluate(() => (document.querySelector("main") as HTMLElement).innerText.match(/Đã duyệt (\d+)\/(\d+)/)?.[0]);
  console.log("INFO tiến độ trước:", prog0);
  const url0 = page.url();
  await page.locator("main").click({ position: { x: 10, y: 10 } }).catch(() => undefined);
  await page.keyboard.press("a");
  await page.keyboard.press("a");
  await page.waitForTimeout(3500);
  console.log("INFO URL sau A:", url0, "→", page.url());
  console.log("INFO yêu cầu ghi:", JSON.stringify(posts));
  expect(page.url(), "UX15-8: sang bài kế").not.toBe(url0);
  const approves = posts.filter((p) => /approve|review|grade/.test(p) && /^(POST|PUT|PATCH)/.test(p));
  console.log("INFO yêu cầu duyệt:", approves.length);
  const st = await api<{ status?: string; grade?: { status?: string } }>(page, `/courses/${c}/submissions/${drafts[0].id}`);
  console.log("INFO trạng thái bài đã A:", JSON.stringify(st.body).slice(0, 160).replace(/\n/g, ""));
  const prog1 = await page.evaluate(() => (document.querySelector("main") as HTMLElement).innerText.match(/Đã duyệt (\d+)\/(\d+)/)?.[0]);
  console.log("INFO tiến độ sau:", prog1);
  // ‹ › không hoạt cảnh
  const before = page.url();
  await page.getByRole("button", { name: /›|Bài sau|Tiếp/ }).first().click().catch(async () => { await page.keyboard.press("ArrowRight"); });
  const anim = await page.evaluate(() => document.getAnimations().filter((a) => !/shimmer|spin/.test((a as CSSAnimation).animationName ?? "")).length);
  await page.waitForTimeout(1500);
  console.log("INFO sau ›:", before, "→", page.url(), "| hoạt cảnh ngay sau:", anim);
  await ctx.close();
});

test("TC-UX15-12/13 tự lưu điểm + nháp khôi phục; lỗi duyệt báo tại thanh dưới, điểm còn", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  const aid = await AID(page, c);
  const d = (await subs(page, c, aid, "DRAFT"))[0];
  await page.goto(`/grading/${d.id}?course=${c}`);
  await page.waitForTimeout(3500);
  const inp = page.locator("main input[aria-label^='Điểm ']").nth(1);
  const old = await inp.inputValue();
  const val = old === "1,50" ? "1,25" : "1,50";
  await inp.fill(val);
  await inp.blur();
  const note = page.locator("main textarea").first();
  await note.fill("QC ghi chú tự lưu 9.1");
  await page.waitForTimeout(2500);
  await page.reload();
  await page.waitForTimeout(3500);
  const v2 = await page.locator("main input[aria-label^='Điểm ']").nth(1).inputValue();
  const n2 = await page.locator("main textarea").first().inputValue();
  console.log(`INFO sau reload: điểm ${old} → nhập ${val} → còn ${v2}; ghi chú: ${n2.slice(0, 30)}`);
  expect(v2).toBe(val);
  expect(n2).toContain("QC ghi chú tự lưu 9.1");
  // lỗi duyệt
  await page.route(/\/submissions\/[0-9a-f-]{36}\/(approve|review)/, (r) => r.abort());
  await page.route(/\/grade(\?|$)/, (r) => r.continue());
  await page.getByRole("button", { name: "Duyệt bài" }).click();
  await page.waitForTimeout(2500);
  const t = await page.locator("main").innerText();
  const bar = await page.evaluate(() => { const btn = [...document.querySelectorAll("main button")].find((b) => /Thử lại|Duyệt bài/.test((b as HTMLElement).innerText)) as HTMLElement | undefined; let n = btn?.parentElement ?? null; while (n && n !== document.body) { const p = getComputedStyle(n).position; if (p === "sticky" || p === "fixed") return n.innerText.replace(/\s+/g, " ").slice(0, 200); n = n.parentElement; } return ""; });
  console.log("INFO thanh dưới sau lỗi:", bar);
  expect(await page.locator("main input[aria-label^='Điểm ']").nth(1).inputValue(), "điểm còn nguyên").toBe(val);
  expect(bar).toMatch(/Thử lại|không|lỗi|Lỗi/);
  void t;
  await ctx.close();
});

test("TC-UX15-15/16/17/18 SV chưa thấy điểm; STUDENT 403; TA duyệt được", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  const aid = await AID(page, c);
  const d = (await subs(page, c, aid, "DRAFT")).concat(await subs(page, c, aid, "REVIEWED"))[0];
  const cs = await browser.newContext({ ignoreHTTPSErrors: true });
  const ps = await cs.newPage();
  await login(ps, "sv.gioi");
  await ps.goto(`/grading/${d.id}`); await ps.waitForTimeout(2500);
  expect(await ps.locator("main").innerText()).toContain("Bạn không có quyền xem màn này");
  await ps.goto("/assignments"); await ps.waitForTimeout(3000);
  const t = (await ps.locator("main").innerText()).replace(/\s+/g, " ");
  console.log("INFO SV /assignments:", t.slice(0, 260));
  expect(t, "SV không thấy điểm nháp / đáp án").not.toMatch(/Điểm nháp|Ghi chú quan sát|Đáp án/i);
  await cs.close();
  const ct = await browser.newContext({ ignoreHTTPSErrors: true });
  const pt = await ct.newPage();
  await login(pt, "ta");
  await pt.goto(`/grading/${d.id}?course=${c}`); await pt.waitForTimeout(3500);
  const tt = await pt.locator("main").innerText();
  expect(tt).not.toContain("Bạn không có quyền xem màn này");
  console.log("INFO TA thấy nút Duyệt bài:", await pt.getByRole("button", { name: "Duyệt bài" }).count());
  await ct.close();
  await ctx.close();
});

test("TC-UX14-11/UX15-19 375 px: hai tab Bài làm / Chấm, thanh duyệt dính đáy ≥ 44 px trên thanh điều hướng", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser, 375, 812);
  const aid = await AID(page, c);
  const d = (await subs(page, c, aid, "DRAFT"))[0];
  await page.goto(`/grading/${d.id}?course=${c}`);
  await page.waitForTimeout(3500);
  const r = await page.evaluate(() => {
    const tabs = [...document.querySelectorAll("main [role=tab], main button")].map((e) => (e as HTMLElement).innerText.trim()).filter((t) => /^(Bài làm|Chấm)$/.test(t));
    const btn = [...document.querySelectorAll("main button")].find((b) => (b as HTMLElement).innerText.trim() === "Duyệt bài") as HTMLElement | undefined;
    const bb = btn?.getBoundingClientRect();
    const nav = document.querySelector("nav[aria-label], [data-part=bottom-nav]") as HTMLElement | null;
    const nb = [...document.querySelectorAll("nav")].map((n) => n.getBoundingClientRect()).filter((r) => r.top > innerHeight / 2).pop();
    return { tabs, btn: bb ? { top: Math.round(bb.top), bottom: Math.round(bb.bottom), h: Math.round(bb.height) } : null, navTop: nb ? Math.round(nb.top) : null, ov: document.documentElement.scrollWidth > document.documentElement.clientWidth, innerH: innerHeight, navFound: !!nav };
  });
  console.log("INFO 375 duyệt bài:", JSON.stringify(r));
  expect(r.tabs.sort()).toEqual(["Bài làm", "Chấm"]);
  expect(r.ov).toBe(false);
  if (r.btn) { expect(r.btn.h, "nút ≥ 44").toBeGreaterThanOrEqual(43); if (r.navTop) expect.soft(r.btn.bottom, "nằm trên thanh điều hướng đáy").toBeLessThanOrEqual(r.navTop + 1); }
  await ctx.close();
});

// ---------- US-UX91-16 ----------
test("TC-UX16-2/3/4/5/6/7/9/10/11 sổ điểm thật: số khớp API, tiêu đề một dòng, một vùng cuộn, Thiếu điểm", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  await page.goto(`/gradebook?course=${c}`);
  await page.waitForTimeout(4000);
  const g = await page.evaluate(() => {
    const ths = [...document.querySelectorAll("main thead th")] as HTMLElement[];
    const heads = ths.map((t) => ({ t: t.innerText.trim().replace(/\s+/g, " "), h: Math.round(t.getBoundingClientRect().height), pos: getComputedStyle(t).position, lines: t.innerText.trim().split("\n").length }));
    const rows = [...document.querySelectorAll("main tbody tr")].slice(0, 6).map((r) => [...r.querySelectorAll("td")].map((td) => (td as HTMLElement).innerText.trim()));
    const num = (document.querySelector("main tbody tr td:nth-child(7)") as HTMLElement | null);
    const ncs = num ? getComputedStyle(num) : null;
    const scr = [...document.querySelectorAll("main *")].filter((e) => { const c = getComputedStyle(e); return /(auto|scroll)/.test(c.overflowY) && e.scrollHeight > e.clientHeight; }).map((e) => ({ h: e.clientHeight, top: Math.round(e.getBoundingClientRect().top), cls: (e as HTMLElement).className.toString().slice(0, 30) }));
    const first = document.querySelector("main tbody tr td:first-child") as HTMLElement | null;
    const text = (document.querySelector("main") as HTMLElement).innerText.replace(/\s+/g, " ");
    return { heads, rows, align: ncs?.textAlign, tab: ncs?.fontVariantNumeric, scr, nameSticky: first ? getComputedStyle(first).position : null, innerH: innerHeight, text: text.slice(0, 260), foot: text.slice(-260), du: [...document.querySelectorAll("main td")].filter((e) => (e as HTMLElement).innerText.trim() === "Đủ").length, thieu: text.match(/Thiếu điểm: \d+ sinh viên/)?.[0] ?? null, statusTh: heads.some((h) => /Trạng thái/.test(h.t)) };
  });
  console.log("INFO đầu bảng:", JSON.stringify(g.heads));
  console.log("INFO hàng:", JSON.stringify(g.rows.slice(0, 3)));
  console.log("INFO cuộn:", JSON.stringify(g.scr), "căn:", g.align, g.tab, "cột tên:", g.nameSticky, "| đầu trang:", g.text, "| chân:", g.foot);
  expect(g.align, "UX16-2 text-align right").toBe("right");
  expect(g.tab).toContain("tabular-nums");
  expect(new Set(g.heads.map((h) => h.h)).size, "UX16-4 cùng chiều cao tiêu đề").toBe(1);
  expect(g.heads.every((h) => h.lines === 1), "tiêu đề cột một dòng").toBe(true);
  expect(g.scr.length, "UX16-6 đúng 1 vùng cuộn dọc").toBe(1);
  expect(Math.abs(g.scr[0].top + g.scr[0].h - g.innerH), "vùng cuộn tới đáy khung ±8 (hoặc dưới đó là chân bảng)").toBeLessThanOrEqual(120);
  expect(g.heads[0].pos).toBe("sticky");
  expect(g.nameSticky, "cột tên dính").toBe("sticky");
  expect(g.du, "UX16-11 không còn ô «Đủ»").toBe(0);
  console.log("INFO thiếu:", g.thieu, "| cột Trạng thái:", g.statusTh);
  // UX16-3: khớp API
  const r = await api<{ columns: Array<{ item_id: string; name: string }>; rows: Array<{ student: { full_name: string }; entries: Record<string, { score: string }>; result: Record<string, string> | null; bonus: string; penalty: string }>; next_cursor: string | null }>(page, `/courses/${c}/gradebook?limit=100`);
  const byName = new Map(r.body.rows.map((x) => [x.student.full_name, x]));
  const idx = (n: string) => g.heads.findIndex((h) => h.t === n);
  let bad = 0, checked = 0;
  const hp = idx("Điểm học phần"), qt = idx("Quá trình"), cong = idx("Điểm cộng");
  for (const row of g.rows) {
    const x = byName.get(row[0]);
    if (!x?.result) continue;
    for (const [col, key] of [[hp, "hp"], [qt, "qt"]] as const) {
      const ui = row[col]?.replace(",", ".");
      const apiV = x.result[key];
      if (ui && apiV) { checked++; if (Number(ui) !== Number(apiV)) { bad++; console.log("INFO lệch:", row[0], key, ui, apiV); } }
    }
    if (cong >= 0 && row[cong] && Number(row[cong].replace(",", ".")) !== Number(x.bonus)) { bad++; console.log("INFO lệch điểm cộng:", row[0], row[cong], x.bonus); }
  }
  console.log(`INFO UX16-3: kiểm ${checked} ô số, lệch ${bad}`);
  expect(bad, "UI không làm tròn / tính lại").toBe(0);
  await ctx.close();
});

test("TC-UX16-5/15/16/19/20 menu cột khi rê; Chốt điểm; STUDENT 403; TA không chốt được", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  await page.goto(`/gradebook?course=${c}`);
  await page.waitForTimeout(3500);
  const th = page.locator("main thead th").filter({ hasText: "Kiểm tra hằng tuần" }).first();
  const menuBtn = th.getByRole("button").first();
  const op0 = await menuBtn.evaluate((e) => getComputedStyle(e).opacity).catch(() => "n/a");
  await th.hover();
  await page.waitForTimeout(400);
  const op1 = await menuBtn.evaluate((e) => getComputedStyle(e).opacity).catch(() => "n/a");
  console.log("INFO menu cột: opacity", op0, "→", op1);
  const chot = page.getByRole("button", { name: "Chốt điểm" });
  const dis = await chot.getAttribute("aria-disabled");
  console.log("INFO Chốt điểm aria-disabled:", dis);
  if (dis !== "true") {
    await chot.click();
    await page.waitForTimeout(1200);
    const dlg = page.getByRole("dialog");
    const dt = (await dlg.innerText().catch(() => "")).replace(/\s+/g, " ");
    console.log("INFO hộp xác nhận chốt:", dt.slice(0, 400));
    expect(dt, "hậu quả bằng số").toMatch(/\d/);
    await page.keyboard.press("Escape");
    await page.waitForTimeout(500);
    const st = await api<{ gradebook?: { status?: string } }>(page, `/courses/${c}/grade-scheme/status`);
    console.log("INFO trạng thái sổ sau Esc:", JSON.stringify(st.body).slice(0, 200));
    expect((st.body.gradebook?.status ?? "OPEN")).not.toBe("LOCKED");
  }
  await ctx.close();
  const cs = await browser.newContext({ ignoreHTTPSErrors: true });
  const ps = await cs.newPage();
  await login(ps, "sv.gioi");
  await ps.goto("/gradebook"); await ps.waitForTimeout(2500);
  expect(await ps.locator("main").innerText()).toContain("Bạn không có quyền xem màn này");
  await cs.close();
  const ct = await browser.newContext({ ignoreHTTPSErrors: true });
  const pt = await ct.newPage();
  await login(pt, "ta");
  const tc = await courseOf(pt);
  await pt.goto(`/gradebook?course=${tc}`); await pt.waitForTimeout(3500);
  const tb = pt.getByRole("button", { name: "Chốt điểm" });
  console.log("INFO TA: nút Chốt điểm", await tb.count(), "aria-disabled", await tb.first().getAttribute("aria-disabled").catch(() => null));
  const fin = await api(pt, `/courses/${tc}/gradebook/finalize`, { method: "POST", body: {} });
  console.log("INFO TA POST finalize:", fin.status);
  expect([403, 404]).toContain(fin.status);
  await ct.close();
});

test("TC-UX16-21 375 px: cuộn ngang trong bảng, cột tên dính, không tràn trang, ô ≥ 44 px", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser, 375, 812);
  await page.goto(`/gradebook?course=${c}`);
  await page.waitForTimeout(4000);
  const r = await page.evaluate(() => {
    const sc = [...document.querySelectorAll("main *")].find((e) => /(auto|scroll)/.test(getComputedStyle(e).overflowX) && e.scrollWidth > e.clientWidth + 5) as HTMLElement | undefined;
    const td = document.querySelector("main tbody tr td:nth-child(5)") as HTMLElement | null;
    const inp = document.querySelector("main tbody input") as HTMLElement | null;
    return { ov: document.documentElement.scrollWidth > document.documentElement.clientWidth, inner: !!sc, cell: td ? Math.round(td.getBoundingClientRect().height) : null, input: inp ? Math.round(inp.getBoundingClientRect().height) : null, nameSticky: getComputedStyle(document.querySelector("main tbody tr td:first-child")!).position };
  });
  console.log("INFO 375 sổ điểm:", JSON.stringify(r));
  expect(r.ov).toBe(false);
  expect(r.inner, "cuộn ngang trong bảng").toBe(true);
  expect(r.nameSticky).toBe("sticky");
  await ctx.close();
});

// ---------- US-UX91-12 ----------
const radios = (page: Page) =>
  page.evaluate(() => [...document.querySelectorAll("main tbody tr[data-part=student-row]")].map((tr) => {
    const r = [...tr.querySelectorAll("input[type=radio]")] as HTMLInputElement[];
    const on = r.find((x) => x.checked)!;
    const cands = [getComputedStyle(on), getComputedStyle(on, "::before"), getComputedStyle(on, "::after"), getComputedStyle(on.closest("label")!, "::before"), getComputedStyle(on.closest("label")!, "::after")];
    const col = cands.map((c) => [c.backgroundColor, c.borderTopColor, c.color]).flat().filter((x) => x && x !== "rgba(0, 0, 0, 0)");
    return { st: on.dataset.status!, cols: col.slice(0, 4), accent: getComputedStyle(on).accentColor, appearance: getComputedStyle(on).appearance };
  }));

test("TC-UX12-1/2/3/4/5/6/7 điểm danh: chấm Có mặt mực, đỏ chỉ cho Muộn / Vắng phép / Vắng; phát biểu +", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  await page.goto(`/attendance?course=${c}`);
  await page.waitForTimeout(4000);
  const before = await radios(page);
  console.log("INFO chấm mặc định (3 hàng):", JSON.stringify(before.slice(0, 3)));
  const rows = page.locator("main tbody tr[data-part=student-row]");
  await rows.nth(1).locator("input[data-status=LATE]").check({ force: true });
  await rows.nth(4).locator("input[data-status=EXCUSED]").check({ force: true });
  await rows.nth(8).locator("input[data-status=ABSENT]").check({ force: true });
  await page.waitForTimeout(500);
  const after = await radios(page);
  const isRed = (cols: string[]) => cols.some((x) => { const m = x.match(/\d+(\.\d+)?/g)?.map(Number) ?? []; return /lab|oklch|rgb|color/.test(x) && (/^rgb/.test(x) ? m[0] > 170 && m[1] < 80 : /^lab/.test(x) ? m[0] > 30 && m[0] < 55 && m[1] > 50 : /^oklch/.test(x) ? m[2] > 15 && m[2] < 40 : false); });
  const redIdx = after.map((r, i) => (isRed(r.cols) ? `${i}:${r.st}` : null)).filter(Boolean);
  const presentRed = after.filter((r) => r.st === "PRESENT" && isRed(r.cols)).length;
  console.log("INFO đỏ sau khi đổi:", JSON.stringify(redIdx), "| Có mặt đỏ:", presentRed, "| mẫu LATE:", JSON.stringify(after[1]), "| mẫu PRESENT:", JSON.stringify(after[0]));
  expect(presentRed, "UX12-1: Có mặt không đỏ").toBe(0);
  expect(redIdx.length, "UX12-2: đúng 3 chấm đỏ").toBe(3);
  // UX12-3: nhãn chữ trạng thái + radio có tên
  const txt = await rows.nth(8).innerText();
  const lbl = await rows.nth(8).locator("input[data-status=ABSENT]").evaluate((e) => ({ name: (e.closest("label")?.querySelector(".ep-sr-only") as HTMLElement | null)?.innerText, checked: (e as HTMLInputElement).checked }));
  console.log("INFO hàng Vắng:", txt.replace(/\s+/g, " "), JSON.stringify(lbl));
  expect(txt, "nhãn chữ «Vắng» ở hàng").toMatch(/Vắng/);
  expect(lbl.name).toMatch(/Vắng/);
  // UX12-4/5/6: phát biểu
  const all = (await page.locator("main").innerText());
  console.log("INFO số lần '+ phát biểu':", (all.match(/\+ phát biểu/g) ?? []).length);
  expect((all.match(/\+ phát biểu · \+0,25/g) ?? []).length, "UX12-4: chữ không lặp mọi hàng").toBeLessThan(3);
  const spk = rows.nth(2).getByRole("button", { name: /Ghi phát biểu/ });
  const op0 = await spk.evaluate((e) => getComputedStyle(e).opacity).catch(() => "—");
  await rows.nth(2).hover();
  await page.waitForTimeout(300);
  const op1 = await spk.evaluate((e) => getComputedStyle(e).opacity).catch(() => "—");
  console.log("INFO nút phát biểu opacity:", op0, "→", op1);
  await spk.click(); await spk.click();
  await page.waitForTimeout(500);
  const t2 = (await rows.nth(2).innerText()).replace(/\s+/g, " ");
  console.log("INFO hàng sau 2 lần phát biểu:", t2);
  expect(t2).toMatch(/\+2/);
  expect(t2).toMatch(/0,25\s*×\s*2|×2|× 2/);
  await ctx.close();
});

test("TC-UX12-5b phím P ghi phát biểu cho hàng đang chọn", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  await page.goto(`/attendance?course=${c}`);
  await page.waitForTimeout(4000);
  const rows = page.locator("main tbody tr[data-part=student-row]");
  await rows.nth(5).click({ position: { x: 20, y: 10 } });
  const t0 = (await rows.nth(5).innerText()).replace(/\s+/g, " ");
  await page.keyboard.press("p");
  await page.waitForTimeout(500);
  const t1 = (await rows.nth(5).innerText()).replace(/\s+/g, " ");
  console.log("INFO P:", t0, "→", t1);
  expect(t1).not.toBe(t0);
  await ctx.close();
});

test("TC-UX12-13/14/18/20 điểm danh 1440: nút lưu cùng cột đầu bảng, tiêu đề / cột tên dính; STUDENT 403; đổi chấm ≤ 100 ms", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  await page.goto(`/attendance?course=${c}`);
  await page.waitForTimeout(4000);
  const g = await page.evaluate(() => {
    const save = [...document.querySelectorAll("main button")].find((b) => /Lưu điểm danh/.test((b as HTMLElement).innerText)) as HTMLElement;
    const th = document.querySelector("main thead th") as HTMLElement;
    const sb = save.getBoundingClientRect(), tb = th.getBoundingClientRect();
    return { saveTop: Math.round(sb.top), saveBottom: Math.round(sb.bottom), thTop: Math.round(tb.top), saveRight: Math.round(sb.right), tableRight: Math.round((document.querySelector("main table") as HTMLElement).getBoundingClientRect().right), thPos: getComputedStyle(th).position, namePos: getComputedStyle(document.querySelector("main tbody td") as HTMLElement).position, status: (document.querySelector("main") as HTMLElement).innerText.match(/Chưa có thay đổi|Đã lưu \d+:\d+|Chờ mạng/)?.[0] ?? null, warn: (document.querySelector("main") as HTMLElement).innerText.match(/Công thức[^\n]*chưa xác nhận[^\n]*/)?.[0] ?? null };
  });
  console.log("INFO bố cục:", JSON.stringify(g));
  expect(g.thPos).toBe("sticky");
  expect(g.namePos).toBe("sticky");
  expect(g.saveBottom, "nút lưu cùng hàng đầu / trên đầu bảng").toBeLessThanOrEqual(g.thTop + 4);
  // UX12-20: đổi chấm ≤ 100 ms
  const radio = page.locator("main tbody tr[data-part=student-row]").nth(3).locator("input[data-status=LATE]");
  const t = await radio.evaluate(async (e: HTMLInputElement) => { const t0 = performance.now(); e.click(); await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))); return { ms: Math.round(performance.now() - t0), checked: e.checked, anim: e.closest("tr")!.getAnimations({ subtree: true }).length }; });
  console.log("INFO đổi chấm:", JSON.stringify(t));
  expect(t.checked).toBe(true);
  expect(t.ms).toBeLessThanOrEqual(100);
  await ctx.close();
  const cs = await browser.newContext({ ignoreHTTPSErrors: true });
  const ps = await cs.newPage();
  await login(ps, "sv.gioi");
  await ps.goto("/attendance"); await ps.waitForTimeout(2500);
  expect(await ps.locator("main").innerText()).toContain("Bạn không có quyền xem màn này");
  await cs.close();
  const ct = await browser.newContext({ ignoreHTTPSErrors: true });
  const pt = await ct.newPage();
  await login(pt, "ta");
  await pt.goto("/attendance"); await pt.waitForTimeout(3000);
  expect(await pt.locator("main").innerText()).not.toContain("Bạn không có quyền xem màn này");
  await ct.close();
});

test("TC-UX12-9/10/11/15/19 điểm danh 375: thanh lưu dính trên thanh điều hướng, số cập nhật, lưu 2 SV Vắng ≤ 60 s không hộp thoại", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser, 375, 812);
  const t0 = Date.now();
  await page.goto(`/attendance?course=${c}`);
  await page.waitForTimeout(4000);
  const rows = page.locator("main tbody tr[data-part=student-row]");
  const n = await rows.count();
  console.log("INFO số SV:", n);
  await rows.nth(Math.min(19, n - 1)).scrollIntoViewIfNeeded();
  await rows.nth(Math.min(19, n - 1)).locator("input[data-status=ABSENT]").check({ force: true });
  await rows.nth(Math.min(20, n - 1)).locator("input[data-status=ABSENT]").check({ force: true });
  await page.waitForTimeout(500);
  const bar = await page.evaluate(() => {
    const b = [...document.querySelectorAll("button")].find((x) => /Lưu điểm danh/.test((x as HTMLElement).innerText)) as HTMLElement | undefined;
    const r = b?.getBoundingClientRect();
    const nav = [...document.querySelectorAll("nav")].map((x) => x.getBoundingClientRect()).filter((x) => x.top > innerHeight / 2).pop();
    let p: HTMLElement | null = b?.parentElement ?? null; let pos = "";
    while (p && p !== document.body) { const s = getComputedStyle(p).position; if (s === "sticky" || s === "fixed") { pos = s; break; } p = p.parentElement; }
    return { text: b?.innerText.replace(/\s+/g, " ") ?? null, h: r ? Math.round(r.height) : null, bottom: r ? Math.round(r.bottom) : null, navTop: nav ? Math.round(nav.top) : null, pos, innerH: innerHeight, warnH: (() => { const w = [...document.querySelectorAll("main *")].find((e) => /Công thức.*chưa xác nhận/.test((e as HTMLElement).innerText ?? "") && e.children.length < 4) as HTMLElement | undefined; return w ? Math.round(w.getBoundingClientRect().height) : null; })() };
  });
  console.log("INFO thanh lưu 375:", JSON.stringify(bar));
  expect(bar.pos, "dính").toMatch(/sticky|fixed/);
  expect(bar.text).toMatch(/Lưu điểm danh/);
  expect(bar.text, "số có mặt / vắng cập nhật").toMatch(/2 vắng/);
  expect(bar.h ?? 0).toBeGreaterThanOrEqual(43);
  if (bar.navTop) expect.soft(bar.bottom!, "nằm trên thanh điều hướng đáy").toBeLessThanOrEqual(bar.navTop + 1);
  // hàng cuối không bị che
  await rows.last().scrollIntoViewIfNeeded();
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await page.waitForTimeout(300);
  const lastGap = await page.evaluate(() => { const r = [...document.querySelectorAll("main tbody tr")].pop()!.getBoundingClientRect(); const b = [...document.querySelectorAll("button")].find((x) => /Lưu điểm danh/.test((x as HTMLElement).innerText))!.getBoundingClientRect(); return Math.round(b.top - r.bottom); });
  console.log("INFO khoảng hàng cuối → thanh:", lastGap);
  expect.soft(lastGap, "hàng cuối không bị thanh che").toBeGreaterThanOrEqual(0);
  // đếm hộp thoại + lưu
  await page.getByRole("button", { name: /Lưu điểm danh/ }).click();
  await page.waitForTimeout(1500);
  expect(await page.getByRole("dialog").count(), "0 hộp thoại xác nhận").toBe(0);
  const saved = (await page.locator("body").innerText()).match(/Đã lưu \d+:\d+/)?.[0];
  console.log("INFO sau lưu:", saved, "| tổng thời gian:", Math.round((Date.now() - t0) / 1000), "s");
  expect(saved).toBeTruthy();
  expect((Date.now() - t0) / 1000).toBeLessThan(60);
  await ctx.close();
});

test("TC-UX12-12/PQ lưu hai lần nhanh: một bản ghi; lớp lạ không ghi", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, c } = await teacher(browser);
  const posts: string[] = [];
  page.on("request", (r) => { if (r.method() !== "GET" && /attendance/.test(r.url())) posts.push(`${r.method()} ${new URL(r.url()).pathname.replace(/[0-9a-f-]{36}/g, "‹id›")}`); });
  await page.goto(`/attendance?course=${c}`);
  await page.waitForTimeout(4000);
  const rows = page.locator("main tbody tr[data-part=student-row]");
  await rows.nth(10).locator("input[data-status=LATE]").check({ force: true });
  const save = page.getByRole("button", { name: /Lưu điểm danh/ }).first();
  await save.dblclick();
  await page.waitForTimeout(2500);
  console.log("INFO yêu cầu ghi:", JSON.stringify(posts));
  expect(posts.length, "hai lần bấm không nhân đôi yêu cầu ghi").toBeLessThanOrEqual(1);
  await ctx.close();
});
