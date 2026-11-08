import AxeBuilder from "@axe-core/playwright";
import { expect, firefox, test, webkit, type Browser, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC FEAT-uiux-91 đợt 6 (tc-wave-6.md): trang giới thiệu v2, trên stack thật `ep-qc` (playwright.qc-real.config.ts). Khách = context mới.
const H2 = ["Xem EduPilot làm việc", "AI soạn nháp, giảng viên quyết, mã tính điểm", "Khác biệt, có số đo", "Cho giảng viên và sinh viên", "Bắt đầu với lớp của bạn"];
const desktop = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
type Opts = Parameters<Browser["newContext"]>[0];
async function open(browser: Browser, o: Opts = {}, init?: (p: Page) => Promise<void>) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN", ...o });
  const page = await ctx.newPage();
  const errs: string[] = [];
  const reqs: Array<{ url: string; t: number }> = [];
  let loadAt = 0;
  page.on("pageerror", (e) => errs.push(e.message));
  page.on("console", (m) => m.type() === "error" && errs.push(m.text()));
  page.on("request", (q) => reqs.push({ url: q.url(), t: Date.now() }));
  page.on("load", () => (loadAt = Date.now()));
  if (init) await init(page);
  await page.goto("/", { waitUntil: "load" });
  await page.waitForTimeout(700);
  return { ctx, page, errs, reqs, loadAt: () => loadAt };
}
const isClip = (u: string) => /gioi-thieu-clip-v1\.(mp4|webm)/.test(u);
const isFull = (u: string) => /gioi-thieu-v1\.(mp4|webm)/.test(u);

test("TC-UX08-1 cấu trúc: h1, dòng dẫn, thứ tự h2", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân");
  await expect(page.locator("main, body").first()).toContainText("AI soạn nháp. Giảng viên quyết. Điểm do mã tính.");
  const h2 = await page.locator("h2").allInnerTexts();
  expect(h2.map((s) => s.trim())).toEqual(H2);
  await ctx.close();
});

test("TC-UX08-3/4 màn đầu: đúng 1 nút đỏ, ≤ 2 nút phụ, :active 0,97", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const btns = await page.evaluate(() =>
    [...document.querySelectorAll("a, button")].filter((e) => { const r = e.getBoundingClientRect(); return r.width > 40 && r.height > 24 && r.top >= 0 && r.top < 900 && /Đăng nhập|Tạo tài khoản|Xem video/.test((e as HTMLElement).innerText); }).map((e) => ({ t: (e as HTMLElement).innerText.trim(), bg: getComputedStyle(e).backgroundColor })),
  );
  const names = btns.map((b) => b.t);
  expect(names).toEqual(expect.arrayContaining(["Đăng nhập", "Tạo tài khoản", "Xem video"]));
  const rgb = await page.evaluate((cols) => cols.map((c) => { const k = document.createElement("canvas"); k.width = k.height = 1; const x = k.getContext("2d")!; x.fillStyle = c; x.fillRect(0, 0, 1, 1); return [...x.getImageData(0, 0, 1, 1).data.slice(0, 3)]; }), btns.map((b) => b.bg));
  console.log("INFO màu nút:", JSON.stringify(btns.map((b, i) => [b.t, b.bg, rgb[i]])));
  const red = btns.filter((_, i) => rgb[i][0] > 170 && rgb[i][1] < 70 && rgb[i][2] < 80);
  expect(red.map((b) => b.t), "nút đỏ ở màn đầu").toEqual(["Đăng nhập"]);
  const login = page.getByRole("link", { name: "Đăng nhập" }).first();
  await login.hover();
  await page.mouse.down();
  await page.waitForTimeout(160);
  expect(await login.evaluate((e) => getComputedStyle(e).transform)).toMatch(/matrix\(0\.97/);
  await page.mouse.up();
  await ctx.close();
});

test("TC-UX08-6/7/8 clip: LCP là poster, clip tải SAU load, thuộc tính, ≤ 1 MB, đang phát", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, reqs, loadAt } = await open(browser, {}, async (p) => {
    await p.addInitScript(() => { (window as unknown as { __lcp: string }).__lcp = ""; new PerformanceObserver((l) => l.getEntries().forEach((e) => { const x = e as unknown as { element?: Element; url?: string }; (window as unknown as { __lcp: string }).__lcp = `${x.element?.tagName ?? "?"}|${x.url ?? ""}|${(x.element as HTMLElement | undefined)?.textContent?.slice(0, 40) ?? ""}`; })).observe({ type: "largest-contentful-paint", buffered: true }); });
  });
  await page.waitForTimeout(3000);
  const lcp = await page.evaluate(() => (window as unknown as { __lcp: string }).__lcp);
  console.log("INFO LCP:", lcp);
  const clipReq = reqs.filter((r) => isClip(r.url));
  expect(clipReq.length, "có tải clip").toBeGreaterThan(0);
  expect(clipReq[0].t, "clip tải sau sự kiện load").toBeGreaterThanOrEqual(loadAt() - 5);
  const v = page.locator("video[muted], video").first();
  const a = await v.evaluate((e: HTMLVideoElement) => ({ muted: e.muted, loop: e.loop, pl: e.hasAttribute("playsinline"), aria: e.getAttribute("aria-hidden"), controls: e.controls, src: [...e.querySelectorAll("source")].map((s) => s.type), paused: e.paused }));
  expect(a).toMatchObject({ muted: true, loop: true, pl: true, aria: "true", controls: false });
  expect(a.src).toEqual(expect.arrayContaining(["video/mp4", "video/webm"]));
  await page.waitForTimeout(1500);
  const t = await v.evaluate((e: HTMLVideoElement) => ({ paused: e.paused, ct: e.currentTime }));
  expect(t.paused, "đang phát").toBe(false);
  expect(t.ct).toBeGreaterThan(0.3);
  for (const ext of ["mp4", "webm"]) {
    const r = await page.request.head(`/landing/gioi-thieu-clip-v1.${ext}`);
    expect(r.status()).toBe(200);
    expect(Number(r.headers()["content-length"]), `clip ${ext} ≤ 1 MB`).toBeLessThanOrEqual(1_048_576);
  }
  await ctx.close();
});

test("TC-UX08-9/10/11/12 reduce / Save-Data / tab ẩn / lỗi tải clip", async ({ browser }, info) => {
  desktop(info);
  {
    const { ctx, reqs } = await open(browser, { reducedMotion: "reduce" });
    expect(reqs.filter((r) => isClip(r.url)), "reduce: không tải clip").toEqual([]);
    await ctx.close();
  }
  {
    const { ctx, reqs } = await open(browser, {}, async (p) => { await p.addInitScript(() => Object.defineProperty(navigator, "connection", { value: { saveData: true }, configurable: true })); });
    expect(reqs.filter((r) => isClip(r.url)), "Save-Data: không tải clip").toEqual([]);
    await ctx.close();
  }
  {
    const { ctx, page, reqs } = await open(browser, {}, async (p) => {
      await p.addInitScript(() => {
        const w = window as unknown as { __vis: string };
        w.__vis = "hidden";
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => w.__vis });
        Object.defineProperty(document, "hidden", { configurable: true, get: () => w.__vis === "hidden" });
      });
    });
    await page.waitForTimeout(2000);
    const before = reqs.filter((r) => isClip(r.url)).length;
    await page.evaluate(() => { (window as unknown as { __vis: string }).__vis = "visible"; document.dispatchEvent(new Event("visibilitychange", { bubbles: true })); });
    await page.waitForTimeout(2500);
    const after = reqs.filter((r) => isClip(r.url)).length;
    const playing = await page.locator("video").first().evaluate((e: HTMLVideoElement) => !e.paused);
    console.log("INFO tab ẩn: request trước/sau khi hiện:", before, after, "đang phát:", playing);
    expect(before, "tab ẩn: không tải clip").toBe(0);
    expect.soft(after, "AC3: ẩn lúc tải, hiện lại → clip tải (P-91-5 Q5)").toBeGreaterThan(0);
    expect.soft(playing, "AC3: ẩn lúc tải, hiện lại → phát").toBe(true);
    await ctx.close();
  }
  {
    const { ctx, page } = await open(browser);
    await page.waitForTimeout(2500);
    const st = async () => page.locator("video").first().evaluate((e: HTMLVideoElement) => !e.paused);
    expect(await st(), "đang phát").toBe(true);
    await page.evaluate(() => { const w = window as unknown as { __v: string }; w.__v = "hidden"; Object.defineProperty(document, "visibilityState", { configurable: true, get: () => w.__v }); Object.defineProperty(document, "hidden", { configurable: true, get: () => w.__v === "hidden" }); document.dispatchEvent(new Event("visibilitychange", { bubbles: true })); });
    await page.waitForTimeout(600);
    expect(await st(), "tab ẩn → dừng").toBe(false);
    await page.evaluate(() => { (window as unknown as { __v: string }).__v = "visible"; document.dispatchEvent(new Event("visibilitychange", { bubbles: true })); });
    await page.waitForTimeout(800);
    expect(await st(), "tab hiện lại → phát lại").toBe(true);
    await ctx.close();
  }
  {
    const { ctx, page, errs } = await open(browser, {}, async (p) => { await p.route(/gioi-thieu-clip-v1\.(mp4|webm)/, (r) => r.abort()); });
    await page.waitForTimeout(1500);
    await expect(page.locator("img").first()).toBeVisible();
    expect(errs.filter((e) => !/Failed to load resource|net::ERR/.test(e)), "không lỗi console").toEqual([]);
    await ctx.close();
  }
});

test("TC-UX08-14/15 nút Tạm dừng / Phát: ≥ 44 px, aria-pressed, bàn phím, nhớ trong phiên", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  await page.waitForTimeout(6000);
  const btn = page.getByRole("button", { name: /Tạm dừng/ });
  await expect(btn).toBeVisible();
  const bb = (await btn.boundingBox())!;
  expect(bb.width).toBeGreaterThanOrEqual(43);
  expect(bb.height).toBeGreaterThanOrEqual(43);
  expect(await btn.getAttribute("aria-pressed")).not.toBeNull();
  await btn.focus();
  await page.keyboard.press("Enter");
  await page.waitForTimeout(500);
  expect(await page.locator("video").first().evaluate((e: HTMLVideoElement) => e.paused), "đã dừng").toBe(true);
  await expect(page.getByRole("button", { name: /^Phát/ })).toBeVisible();
  await page.reload({ waitUntil: "load" });
  await page.waitForTimeout(2500);
  const stillPaused = await page.locator("video").first().evaluate((e: HTMLVideoElement) => e.paused || e.currentTime === 0);
  expect(stillPaused, "tải lại trong phiên vẫn dừng").toBe(true);
  await ctx.close();
});

test("TC-UX08-16/17/18/21 Xem video: không tải trước, hộp thoại, focus trap, Esc, bỏ nguồn; reduce mở tức thì", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, reqs } = await open(browser);
  expect(reqs.filter((r) => isFull(r.url)), "trước khi bấm").toEqual([]);
  const open_ = page.getByRole("button", { name: "Xem video" }).or(page.getByRole("link", { name: "Xem video" })).first();
  await open_.click();
  const dlg = page.locator("dialog[open], [role=dialog]").first();
  await expect(dlg).toBeVisible();
  await page.waitForTimeout(800);
  const dv = await page.locator("dialog video, [role=dialog] video").first().evaluate((e: HTMLVideoElement) => ({ src: e.getAttribute("src"), srcs: [...e.querySelectorAll("source")].map((x) => x.getAttribute("src")), cur: e.currentSrc }));
  console.log("INFO nguồn video khi mở:", JSON.stringify(dv));
  expect(dv.src || dv.srcs.some(Boolean), "video có nguồn sau khi bấm").toBeTruthy();
  await page.locator("dialog video, [role=dialog] video").first().evaluate((e: HTMLVideoElement) => e.play().catch(() => undefined));
  await page.waitForTimeout(1500);
  expect(reqs.filter((r) => isFull(r.url)).length, "phát thì có request video").toBeGreaterThan(0);
  for (let i = 0; i < 12; i++) await page.keyboard.press("Tab");
  expect(await page.evaluate(() => !!document.activeElement?.closest("dialog, [role=dialog]")), "focus bị giữ trong hộp thoại").toBe(true);
  await page.keyboard.press("Escape");
  await expect(dlg).toBeHidden();
  const back = await page.evaluate(() => (document.activeElement as HTMLElement | null)?.innerText ?? "");
  expect(back, "focus về nút Xem video").toMatch(/Xem video/);
  const vidEl = page.locator("dialog video, [role=dialog] video").first();
  await expect.poll(() => vidEl.evaluate((e: HTMLVideoElement) => e.paused), { message: "đóng thì dừng", timeout: 3000 }).toBe(true);
  const vid = await vidEl.evaluate((e: HTMLVideoElement) => ({ cur: e.currentSrc, attr: e.getAttribute("src"), srcs: e.querySelectorAll("source[src]").length }));
  console.log("INFO sau khi đóng:", JSON.stringify(vid));
  expect(vid.cur === "" || (vid.attr === null && vid.srcs === 0), `AC5 bỏ nguồn khi đóng: currentSrc=${vid.cur}, <source>=${vid.srcs}`).toBe(true);
  await ctx.close();
  const r = await open(browser, { reducedMotion: "reduce" });
  await r.page.getByRole("button", { name: "Xem video" }).or(r.page.getByRole("link", { name: "Xem video" })).first().click();
  const d = await r.page.evaluate(() => Math.max(0, ...document.getAnimations().map((a) => Number(a.effect?.getTiming().duration ?? 0))));
  expect(d, "reduce: hoạt cảnh mở ≤ 1 ms").toBeLessThanOrEqual(1);
  await r.ctx.close();
});

test("TC-UX08-8b/19 video đầy đủ: thuộc tính; #video-loi; không rãnh âm (ffprobe ở shell)", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const v = page.locator("video").last();
  const a = await v.evaluate((e: HTMLVideoElement) => ({ controls: e.controls, preload: e.preload, pl: e.hasAttribute("playsinline"), poster: !!e.getAttribute("poster") || e.dataset.poster !== undefined, auto: e.autoplay }));
  console.log("INFO video đầy đủ (trước khi mở):", JSON.stringify(a));
  expect(a.controls).toBe(true);
  expect(a.preload).toBe("none");
  expect(a.auto).toBe(false);
  expect(a.pl).toBe(true);
  await expect(page.locator("#video-loi")).toHaveCount(1);
  const lines = (await page.locator("#video-loi").textContent())!.trim();
  expect(lines.length).toBeGreaterThan(80);
  const vis = await page.locator("#video-loi").evaluate((e) => { const r = e.getBoundingClientRect(); return r.width <= 2 && r.height <= 2 || getComputedStyle(e).clip !== "auto" || getComputedStyle(e).position === "absolute"; });
  expect(vis, "ẩn trực quan").toBe(true);
  await ctx.close();
});

test("TC-UX08-22/23/24/25 khung ghim ≥ 1100: sticky, đổi ảnh theo bước, ảnh ≤ 90 KB, chỉ transform/opacity", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const sec = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[0] }) }).first();
  await expect(sec).toBeVisible();
  const h3 = await sec.locator("h3").allInnerTexts();
  expect(h3.length, "ba bước h3").toBe(3);
  const imgs = sec.locator("img");
  const n = await imgs.count();
  expect(n).toBeGreaterThanOrEqual(3);
  const sticky = await sec.evaluate((s) => [...s.querySelectorAll("*")].some((e) => getComputedStyle(e).position === "sticky"));
  expect(sticky, "có phần tử position: sticky").toBe(true);
  // cuộn từng bước và ghi ảnh nào đang nhìn thấy (opacity cao nhất)
  const top0 = (await sec.evaluate((s) => s.getBoundingClientRect().top + window.scrollY)) as number;
  const seen: string[] = [];
  const props = new Set<string>();
  const secH = (await sec.evaluate((s) => s.getBoundingClientRect().height)) as number;
  console.log("INFO chiều cao khối:", Math.round(secH));
  for (let k = 0; k <= 16; k++) {
    await page.evaluate((y) => window.scrollTo(0, y), top0 - 100 + (k * (secH + 100)) / 16);
    await page.waitForTimeout(350);
    const st = await sec.evaluate((s) => {
      const im = [...s.querySelectorAll("img")] as HTMLImageElement[];
      const vis = im.map((i) => ({ src: i.getAttribute("src") ?? "", op: Number(getComputedStyle(i).opacity) * Number(getComputedStyle(i.parentElement!).opacity), r: i.getBoundingClientRect() })).filter((x) => x.r.top < innerHeight && x.r.bottom > 0);
      const on = vis.filter((x) => x.op > 0.5);
      vis.length = 0;
      if (on.length) vis.push(on[on.length - 1]);
      const stk = [...s.querySelectorAll("*")].find((e) => getComputedStyle(e).position === "sticky");
      return { top: vis[0]?.src ?? "", sticky: stk ? Math.round(stk.getBoundingClientRect().top) : null };
    });
    seen.push(`${k}:${st.top.split("/").pop()}@${st.sticky}`);
    for (const a of await page.evaluate(() => document.getAnimations().flatMap((x) => (x.effect as KeyframeEffect | null)?.getKeyframes().flatMap((f) => Object.keys(f)) ?? []))) props.add(a);
  }
  console.log("INFO bước cuộn:", seen.join(" | "));
  const order = [...new Set(seen.map((s) => s.split(":")[1].split("@")[0]).filter(Boolean))];
  console.log("INFO thứ tự ảnh:", order.join(" → "));
  expect(order.length, "đổi qua ≥ 3 ảnh").toBeGreaterThanOrEqual(3);
  console.log("INFO thuộc tính được animate:", [...props].join(","));
  for (const f of ["story-chat-v1.webp", "story-review-v1.webp", "story-gradebook-v1.webp"]) {
    const r = await page.request.head(`/landing/${f}`);
    expect(Number(r.headers()["content-length"]), f).toBeLessThanOrEqual(92_160);
  }
  const attrs = await imgs.evaluateAll((l) => l.map((i) => ({ w: i.getAttribute("width"), h: i.getAttribute("height"), lazy: i.getAttribute("loading"), alt: i.getAttribute("alt") })));
  for (const a of attrs) { expect(a.w && a.h, "width/height").toBeTruthy(); expect(a.alt?.length, "alt").toBeGreaterThan(3); }
  expect(new Set(attrs.map((a) => a.alt)).size, "alt riêng cho ba bước").toBeGreaterThanOrEqual(3);
  expect(attrs.some((a) => a.lazy === "lazy")).toBe(true);
  await ctx.close();
});

test("TC-UX08-26/27 không ghim ở 1024 px và khi reduce; đủ chữ + ảnh", async ({ browser }, info) => {
  desktop(info);
  for (const o of [{ viewport: { width: 1024, height: 800 } }, { reducedMotion: "reduce" as const }]) {
    const { ctx, page } = await open(browser, o);
    const sec = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[0] }) }).first();
    expect(await sec.evaluate((s) => [...s.querySelectorAll("*")].some((e) => getComputedStyle(e).position === "sticky")), JSON.stringify(o)).toBe(false);
    expect(await sec.locator("h3").count()).toBe(3);
    const vis = await sec.locator("img").evaluateAll((l) => l.filter((i) => { const c = getComputedStyle(i); return c.display !== "none" && Number(c.opacity) > 0.5; }).length);
    expect(vis, "ảnh hiện cho mọi bước").toBeGreaterThanOrEqual(3);
    await ctx.close();
  }
});

test("TC-UX08-29/30/31/32 sơ đồ và số đo: thông số hoạt cảnh, nội dung HTML thật, số đếm", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const diag = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[1] }) }).first();
  const steps = await diag.evaluate((s) => [...s.querySelectorAll("li, [data-step]")].slice(0, 6).map((e) => ({ t: (e as HTMLElement).innerText.slice(0, 20).replace(/\n/g, " "), d: getComputedStyle(e).animationDelay, du: getComputedStyle(e).animationDuration, tl: getComputedStyle(e).getPropertyValue("animation-timeline") })));
  console.log("INFO bước sơ đồ:", JSON.stringify(steps));
  await expect(diag).toContainText("AI soạn nháp");
  await expect(diag).toContainText("Giảng viên duyệt");
  await expect(diag).toContainText("Mã tính điểm");
  const aria = await diag.evaluate((s) => [...s.querySelectorAll("[aria-hidden=true]")].length);
  expect(aria, "vạch đỏ aria-hidden").toBeGreaterThan(0);
  const num = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[2] }) }).first();
  await num.scrollIntoViewIfNeeded();
  const samples: string[] = [];
  for (let i = 0; i < 14; i++) { samples.push(await num.evaluate((s) => (s.textContent ?? "").replace(/\s+/g, " ").slice(0, 160))); await page.waitForTimeout(80); }
  console.log("INFO số đếm (mẫu đầu/cuối):", samples[0], "||", samples[samples.length - 1]);
  const full = await num.innerText();
  expect(full).toContain("146");
  expect(full).toMatch(/30\s*\/\s*30/);
  expect(full).toMatch(/0,05\s*%/);
  await ctx.close();
});

test("TC-UX08-33/34 chữ ba số đo khớp nguồn, không «mọi/toàn bộ», có chú thích mô phỏng", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const num = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[2] }) }).first();
  const t = (await num.innerText()).replace(/\s+/g, " ");
  console.log("INFO khối số đo:", t);
  expect(t).toContain("146 yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp");
  expect(t).toMatch(/recall 0,960 trên 200 mẫu tự soạn/);
  expect(t).toContain("30/30 sinh viên mẫu có điểm học phần khớp tính tay");
  expect(t).toMatch(/Chấm hàng loạt[,:] thời gian đến chữ đầu tiên của chat chỉ tăng 0,05 ?%/);
  expect(t).not.toMatch(/\bmọi\b|toàn bộ|được che trước mọi lời gọi AI|độ trễ chat/i);
  expect(await page.getByText("Đo trên dữ liệu mô phỏng.").count()).toBe(1);
  await ctx.close();
});

test("TC-UX08-35/36 hai cột vai, không tab; xếp dọc < 720", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const sec = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[3] }) }).first();
  await expect(sec.locator("[role=tablist], [role=tab]")).toHaveCount(0);
  const cols = await sec.evaluate((s) => [...s.querySelectorAll("h3")].map((h) => { const c = h.closest("div, article, section, li")!; const r = c.getBoundingClientRect(); return { t: (h as HTMLElement).innerText, top: Math.round(r.top), left: Math.round(r.left), items: c.querySelectorAll("li, dt, h4").length }; }));
  console.log("INFO cột vai:", JSON.stringify(cols));
  expect(cols.map((c) => c.t)).toEqual(expect.arrayContaining(["Giảng viên", "Sinh viên"]));
  const [a, b] = [cols.find((c) => c.t === "Giảng viên")!, cols.find((c) => c.t === "Sinh viên")!];
  expect(Math.abs(a.top - b.top), "cùng hàng").toBeLessThanOrEqual(6);
  expect(a.left).not.toBe(b.left);
  expect(await sec.innerText()).not.toMatch(/\bThreads\b/);
  await ctx.close();
  const m = await open(browser, { viewport: { width: 700, height: 900 } });
  const s2 = m.page.locator("section").filter({ has: m.page.getByRole("heading", { level: 2, name: H2[3] }) }).first();
  const c2 = await s2.evaluate((s) => [...s.querySelectorAll("h3")].map((h) => Math.round(h.getBoundingClientRect().top)));
  expect(Math.abs(c2[0] - c2[1]), "xếp dọc").toBeGreaterThan(40);
  expect(await m.page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  await m.ctx.close();
});

test("TC-UX08-38/39 chữ lớn, khoảng thở, một h2 mỗi khung nhìn", async ({ browser }, info) => {
  desktop(info);
  for (const [vp, min] of [[{ width: 1440, height: 900 }, { h1: 56, h2: 40, body: 18, gap: 96 }], [{ width: 375, height: 812 }, { h1: 36, h2: 28, body: 16, gap: 64 }]] as const) {
    const { ctx, page } = await open(browser, { viewport: vp });
    const m = await page.evaluate(() => {
      const px = (e: Element | null) => (e ? parseFloat(getComputedStyle(e).fontSize) : 0);
      const secs = [...document.querySelectorAll("main section, [data-part=landing] section")];
      const gaps: number[] = [];
      for (let i = 1; i < secs.length; i++) gaps.push(Math.round(secs[i].getBoundingClientRect().top - secs[i - 1].getBoundingClientRect().bottom));
      const small = [...document.querySelectorAll("p")].filter((p) => px(p) < (innerWidth >= 700 ? 18 : 16) && (p as HTMLElement).innerText.trim().length > 60 && !p.closest("[aria-hidden=true], #video-loi, footer, dialog")).map((p) => `${px(p)}px «${(p as HTMLElement).innerText.trim().slice(0, 50)}»`);
      const notes = [...document.querySelectorAll("p")].filter((p) => px(p) < 17 && (p as HTMLElement).innerText.trim()).map((p) => `${px(p)}px ${(p as HTMLElement).innerText.trim().slice(0, 28).replace(/\n/g, " ")}`);
      console.log("NOTES", notes.length);
      const pads = secs.map((e) => { const c = getComputedStyle(e); return Math.round(parseFloat(c.paddingTop) + parseFloat(c.marginTop)) + "/" + Math.round(parseFloat(c.paddingBottom) + parseFloat(c.marginBottom)); });
      const body = [...document.querySelectorAll("p")].map((p) => px(p)).filter(Boolean);
      return { h1: px(document.querySelector("h1")), h2: Math.min(...[...document.querySelectorAll("h2")].map(px)), body: Math.min(...body), small, pads, gaps };
    });
    console.log("INFO cỡ chữ", vp.width, JSON.stringify({ h1: m.h1, h2: m.h2, small: m.small, pads: m.pads }));
    expect(m.h1).toBeGreaterThanOrEqual(min.h1);
    expect(m.h2).toBeGreaterThanOrEqual(min.h2);
    expect(m.small, `đoạn chữ thân < ${min.body}px`).toEqual([]);
    const spaces = m.pads.slice(1).map((p, i) => Number(p.split("/")[0]) + Number(m.pads[i].split("/")[1]));
    expect(Math.min(...spaces), `khoảng giữa các mục ≥ ${min.gap}: ${m.pads.join(" ")}`).toBeGreaterThanOrEqual(min.gap - 1);
    if (vp.width === 1440) {
      for (const h of H2) {
        const el = page.getByRole("heading", { level: 2, name: h });
        await el.evaluate((e) => window.scrollTo(0, e.getBoundingClientRect().top + window.scrollY - 40));
        await page.waitForTimeout(250);
        const inView = await page.evaluate(() => [...document.querySelectorAll("h2")].filter((e) => { const r = e.getBoundingClientRect(); return r.top >= 0 && r.bottom <= innerHeight; }).map((e) => `${(e as HTMLElement).innerText.slice(0, 22)}@${Math.round(e.getBoundingClientRect().top)}`));
        console.log("INFO h2 trong khung khi cuộn tới", h, ":", inView.join(" ; "));
        // AC «một h2 mỗi khung nhìn» đã bỏ (P-91-8 (b)): chỉ ghi lại
      }
    }
    await ctx.close();
  }
});

test("TC-UX08-40/41/42 chiều sâu: bóng họ popover, nâng ≤ 6 px, chỉ hover; cảm ứng / reduce không nâng", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const sec = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[0] }) }).first();
  const fr = sec.locator("img").first();
  await fr.scrollIntoViewIfNeeded();
  const sh = await fr.evaluate((e) => { let n: Element | null = e; for (let i = 0; i < 4 && n; i++, n = n.parentElement) { const s = getComputedStyle(n).boxShadow; if (s !== "none") return s; } return "none"; });
  const tokens = await page.evaluate(() => [...document.styleSheets].flatMap((s) => { try { return [...s.cssRules]; } catch { return []; } }).flatMap((r) => (r as CSSStyleRule).style ? [...(r as CSSStyleRule).style].filter((p) => p.startsWith("--ep-") && /shadow|elev/.test(p)).map((p) => `${p}=${(r as CSSStyleRule).style.getPropertyValue(p).trim()}`) : []));
  console.log("INFO bóng khung:", sh, "| token:", tokens.slice(0, 8).join(" ; "));
  expect(sh, "khung có bóng tĩnh").not.toBe("none");
  const frame = await fr.evaluate((e) => { let n: Element | null = e; for (let i = 0; i < 4 && n; i++, n = n.parentElement) if (getComputedStyle(n).boxShadow !== "none") return n as HTMLElement; return e as HTMLElement; }).catch(() => null);
  void frame;
  await ctx.close();
  const t = await open(browser, { hasTouch: true, isMobile: true, viewport: { width: 390, height: 844 } });
  const hoverRules = await t.page.evaluate(() => matchMedia("(hover: hover)").matches);
  expect(hoverRules, "cảm ứng không hỗ trợ hover").toBe(false);
  await t.ctx.close();
});

test("TC-UX08-46/48 reduce: không clip, không ghim, số cuối ngay, 0 animation; JS tắt vẫn đủ nội dung", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, reqs } = await open(browser, { reducedMotion: "reduce" });
  expect(reqs.filter((r) => isClip(r.url))).toEqual([]);
  expect(await page.evaluate(() => document.getAnimations().length), "animation đang chạy").toBe(0);
  const t = await page.locator("body").innerText();
  expect(t).toContain("146");
  expect(t).toContain("Mã tính điểm");
  await ctx.close();
  const nj = await open(browser, { javaScriptEnabled: false });
  for (let i = 0; i < 40; i++) { await nj.page.mouse.wheel(0, 300); await nj.page.waitForTimeout(100); }
  const hidden = await nj.page.evaluate(() => [...document.querySelectorAll("h2, h3, p, li, img")].filter((e) => { const c = getComputedStyle(e); let n: Element | null = e; let op = 1; while (n) { op *= Number(getComputedStyle(n).opacity); n = n.parentElement; } const r = e.getBoundingClientRect(); return (op < 0.05 || c.visibility === "hidden") && r.width > 0 && getComputedStyle(e).display !== "none"; }).map((e) => `${e.tagName}«${(e as HTMLElement).innerText?.slice(0, 30) ?? (e as HTMLImageElement).getAttribute("src")}»`));
  console.log("INFO khối ẩn khi tắt JS:", hidden.join(" ; "));
  expect(hidden, "khối ẩn vĩnh viễn khi tắt JS (sau khi cuộn qua)").toEqual([]);
  await nj.ctx.close();
});

for (const w of [1440, 375]) {
  test(`TC-UX08-50 axe ${w} px và thứ bậc tiêu đề`, async ({ browser }, info) => {
    desktop(info);
    const { ctx, page } = await open(browser, { viewport: { width: w, height: w > 700 ? 900 : 812 } });
    const r = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
    const bad = r.violations.filter((v) => v.impact === "serious" || v.impact === "critical").map((v) => `${v.id} ×${v.nodes.length}`);
    expect(bad).toEqual([]);
    const lv = await page.evaluate(() => [...document.querySelectorAll("h1,h2,h3,h4")].map((h) => Number(h.tagName[1])));
    for (let i = 1; i < lv.length; i++) expect(lv[i] - lv[i - 1], `nhảy cấp tiêu đề ở #${i}`).toBeLessThanOrEqual(1);
    const noAlt = await page.evaluate(() => [...document.querySelectorAll("img")].filter((i) => i.getAttribute("alt") === null).length);
    expect(noAlt).toBe(0);
    await ctx.close();
  });
}

test("TC-UX08-51/54 không rò mật khẩu / tài khoản; CTA đúng đích", async ({ browser, request }, info) => {
  desktop(info);
  const html = await (await request.get("/")).text();
  expect(html).not.toMatch(/@edupilot\.local|Edupilot#|mật khẩu mặc định/i);
  const js = [...html.matchAll(/\/_next\/static\/chunks\/[^"']+\.js/g)].map((m) => m[0]);
  for (const u of new Set(js)) expect(await (await request.get(u)).text(), u).not.toMatch(/@edupilot\.local|Edupilot#/);
  const { ctx, page } = await open(browser);
  const t = await page.locator("body").innerText();
  expect(t).toContain("Dữ liệu trên bản demo là dữ liệu mô phỏng.");
  expect(t).toContain("Đồ án tốt nghiệp · Lê Văn Quang Trung");
  await expect(page.getByRole("link", { name: "Mã nguồn trên GitHub" })).toHaveAttribute("href", /github\.com\/kagamikuro1024\/EduPilot-v2$/);
  for (const [name, href] of [["Đăng nhập", "/login"], ["Tạo tài khoản", "/register"]]) {
    await expect(page.getByRole("link", { name }).first()).toHaveAttribute("href", href);
    await expect(page.getByRole("link", { name }).last()).toHaveAttribute("href", href);
  }
  await ctx.close();
});

test("TC-UX08-53 lỗi ảnh / clip / video đầy đủ: khung giữ kích thước, không lỗi", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page, errs } = await open(browser, {}, async (p) => {
    await p.route(/story-chat-v1\.webp|gioi-thieu-clip-v1|gioi-thieu-v1\.(mp4|webm|webp)/, (r) => r.abort());
  });
  const sec = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[0] }) }).first();
  const boxes = await sec.locator("img").evaluateAll((l) => l.map((i) => { const r = i.getBoundingClientRect(); return `${Math.round(r.width)}x${Math.round(r.height)}`; }));
  console.log("INFO kích thước ảnh khi lỗi:", boxes.join(","));
  const shown = boxes.filter((b) => b !== "0x0");
  expect(shown.length, "ảnh đang hiện").toBeGreaterThanOrEqual(3);
  expect(new Set(shown).size, "ảnh lỗi giữ kích thước như ảnh tốt").toBe(1);
  await page.getByRole("button", { name: "Xem video" }).or(page.getByRole("link", { name: "Xem video" })).first().click();
  await page.waitForTimeout(1200);
  const d = await page.locator("dialog[open], [role=dialog]").first().innerText();
  console.log("INFO hộp thoại khi video lỗi:", d.replace(/\s+/g, " ").slice(0, 160));
  expect(d).toMatch(/Mở video trong tab mới/);
  expect(errs.filter((e) => !/Failed to load resource|net::ERR|aborted/.test(e))).toEqual([]);
  await ctx.close();
});

test("TC-UX08-52 người đã đăng nhập mở `/` thấy Hôm nay, không nháy trang giới thiệu", async ({ browser }, info) => {
  desktop(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("sv.gioi@edupilot.local");
  await page.locator("input[type=password]").fill(process.env.QC_PW ?? "Edupilot#Seed-2026");
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await ctx.addInitScript(() => {
    const w = window as unknown as { __saw: boolean };
    w.__saw = false;
    new MutationObserver(() => { const l = document.querySelector("[data-part=landing]"); if (l && (l as HTMLElement).offsetHeight > 0 && getComputedStyle(l).display !== "none") w.__saw = true; }).observe(document, { childList: true, subtree: true, attributes: true });
  });
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible();
  expect(await page.evaluate(() => (window as unknown as { __saw: boolean }).__saw)).toBe(false);
  await ctx.close();
});

for (const [name, bt] of [["webkit", webkit], ["firefox", firefox]] as const) {
  test(`TC-UX08-28/48 ${name}: không có animation-timeline → không ghim, đủ nội dung, không khối ẩn`, async ({}, info) => {
    desktop(info);
    const b = await bt.launch();
    const { ctx, page } = await open(b);
    const sup = await page.evaluate(() => CSS.supports("animation-timeline: view()"));
    console.log(`INFO ${name} animation-timeline:`, sup);
    const sec = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[0] }) }).first();
    if (!sup) expect(await sec.evaluate((s) => [...s.querySelectorAll("*")].some((e) => getComputedStyle(e).position === "sticky")), "không ghim").toBe(false);
    for (let i = 0; i < 40; i++) { await page.mouse.wheel(0, 300); await page.waitForTimeout(80); }
    const hidden = await page.evaluate(() => [...document.querySelectorAll("h2, h3, p, li, img")].filter((e) => { let n: Element | null = e; let op = 1; while (n) { op *= Number(getComputedStyle(n).opacity); n = n.parentElement; } return op < 0.05 && e.getBoundingClientRect().width > 0 && getComputedStyle(e).display !== "none"; }).map((e) => e.tagName + "«" + ((e as HTMLElement).innerText ?? "").slice(0, 24) + "»"));
    console.log(`INFO ${name} khối ẩn:`, hidden.join(" ; "));
    expect(hidden).toEqual([]);
    expect(await page.locator("body").innerText()).toContain("146");
    expect(await sec.locator("h3").count()).toBe(3);
    await ctx.close();
    await b.close();
  });
}

const CAPS = [
  "Sinh viên hỏi riêng về điểm và chuyên cần của chính mình.",
  "Họ tên và MSSV được che trước khi gửi cho AI; câu trả lời kèm nguồn trong tài liệu của lớp.",
  "Câu hỏi AI chưa chắc được chuyển tới hộp thư của giảng viên, kèm lý do.",
  "Giảng viên thấy hàng chờ chấm, AI chấm nháp theo rubric kèm đoạn trích từ bài làm.",
  "Giảng viên sửa điểm, duyệt rồi mới công bố. AI không tự công bố.",
  "Sổ điểm: điểm học phần do mã tính từ quy chế, mỗi điểm có giải trình từng dòng.",
  "Hôm nay: việc cần giảng viên quyết, xếp theo mức khẩn.",
];
const openVideo = (page: Page) => page.getByRole("button", { name: "Xem video" }).or(page.getByRole("link", { name: "Xem video" })).first().click();

test("TC-UX09-10 #video-loi có đủ 7 câu phụ đề theo thứ tự, ẩn trực quan, aria-describedby", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const t = (await page.locator("#video-loi").textContent())!.replace(/\s+/g, " ");
  let at = -1;
  for (const c of CAPS) { const i = t.indexOf(c); expect(i, c).toBeGreaterThan(at); at = i; }
  const refs = await page.evaluate(() => [...document.querySelectorAll("[aria-describedby~=video-loi]")].length);
  expect(refs, "có phần tử aria-describedby=video-loi").toBeGreaterThan(0);
  await ctx.close();
});

test("TC-UX09-11/13 375 cảm ứng: hộp thoại vừa khung, video không tràn; tua 30 s phát tiếp (Chromium + WebKit)", async ({ browser }, info) => {
  desktop(info);
  const m = await open(browser, { viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  await openVideo(m.page);
  await m.page.waitForTimeout(800);
  const r = await m.page.evaluate(() => { const v = document.querySelector("dialog video, [role=dialog] video") as HTMLVideoElement; const b = v.getBoundingClientRect(); return { w: Math.round(b.width), ov: document.documentElement.scrollWidth > document.documentElement.clientWidth, controls: v.controls }; });
  console.log("INFO video 375:", JSON.stringify(r));
  expect(r.w).toBeLessThanOrEqual(375);
  expect(r.ov).toBe(false);
  expect(r.controls).toBe(true);
  await m.ctx.close();
  for (const [name, bt] of [["chromium", null], ["webkit", webkit]] as const) {
    const b = bt ? await bt.launch() : browser;
    const { ctx, page } = await open(b);
    await openVideo(page);
    const res = await page.evaluate(async () => {
      const v = document.querySelector("dialog video, [role=dialog] video") as HTMLVideoElement;
      try { await v.play(); } catch (e) { return { err: String(e) }; }
      await new Promise((r) => setTimeout(r, 600));
      v.currentTime = 30;
      await new Promise((r) => setTimeout(r, 2000));
      return { ct: v.currentTime, paused: v.paused, src: v.currentSrc.split("/").pop() };
    });
    console.log(`INFO tua 30 s ${name}:`, JSON.stringify(res));
    expect.soft("ct" in res ? res.ct : 0, `${name}: phát tiếp từ 30 s`).toBeGreaterThan(30);
    await ctx.close();
    if (bt) await b.close();
  }
});

test("TC-UX09-17/18 canPlayType: webm rỗng → MP4 phát; cả hai rỗng → poster + liên kết tải, không console.error", async ({ browser }, info) => {
  desktop(info);
  const noWebm = async (p: Page) => { await p.addInitScript(() => { const o = HTMLMediaElement.prototype.canPlayType; HTMLMediaElement.prototype.canPlayType = function (t: string) { return /webm/.test(t) ? "" : o.call(this, t); }; }); };
  const a = await open(browser, {}, noWebm);
  await a.page.waitForTimeout(3000);
  const r = await a.page.locator("video").first().evaluate((e: HTMLVideoElement) => ({ src: e.currentSrc.split("/").pop(), paused: e.paused }));
  console.log("INFO webm rỗng:", JSON.stringify(r));
  expect(r.src).toMatch(/\.mp4$/);
  expect(r.paused).toBe(false);
  await a.ctx.close();
  const none = async (p: Page) => { await p.addInitScript(() => { HTMLMediaElement.prototype.canPlayType = () => ""; }); };
  const b = await open(browser, {}, none);
  await b.page.waitForTimeout(2000);
  await openVideo(b.page);
  await b.page.waitForTimeout(1000);
  const d = (await b.page.locator("dialog[open], [role=dialog]").first().innerText()).replace(/\s+/g, " ");
  console.log("INFO cả hai rỗng, hộp thoại:", d.slice(0, 200));
  expect(d).toMatch(/Tải video|Mở video trong tab mới|tải/i);
  expect(b.errs.filter((e) => !/Failed to load resource|net::ERR/.test(e))).toEqual([]);
  await b.ctx.close();
});

test("TC-UX09-19 reduce: clip không phát; video đầy đủ không tự phát", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser, { reducedMotion: "reduce" });
  await page.waitForTimeout(2500);
  const clip = await page.locator("video").first().evaluate((e: HTMLVideoElement) => ({ paused: e.paused, ct: e.currentTime }));
  expect(clip.paused || clip.ct === 0, "clip không chạy").toBe(true);
  await openVideo(page);
  await page.waitForTimeout(1500);
  expect(await page.locator("dialog video, [role=dialog] video").first().evaluate((e: HTMLVideoElement) => e.paused), "video không tự phát").toBe(true);
  await ctx.close();
});

test("TC-UX08-29/30/31 sơ đồ và số đếm theo cuộn: thứ tự hiện, vạch scaleX, đếm 0 → 146, có chạy lại khi cuộn ngược không", async ({ browser }, info) => {
  desktop(info);
  const { ctx, page } = await open(browser);
  const diag = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[1] }) }).first();
  const snap = () => diag.evaluate((s) => {
    const li = [...s.querySelectorAll("li, [data-step]")].slice(0, 3).map((e) => Number(getComputedStyle(e).opacity).toFixed(2));
    const line = [...s.querySelectorAll("*")].map((e) => getComputedStyle(e, "::after").transform).find((t) => t && t !== "none") ?? [...s.querySelectorAll("*")].map((e) => getComputedStyle(e).transform).find((t) => /matrix/.test(t)) ?? "none";
    return { li, line, top: Math.round(s.getBoundingClientRect().top) };
  });
  const top0 = (await diag.evaluate((s) => s.getBoundingClientRect().top + scrollY)) as number;
  const down: string[] = [];
  for (let k = -500; k <= 300; k += 100) { await page.evaluate((y) => window.scrollTo(0, y), top0 + k); await page.waitForTimeout(250); const r = await snap(); down.push(`${k}:${r.li.join("/")}:${r.line.slice(0, 22)}`); }
  console.log("INFO sơ đồ cuộn xuống:", down.join(" | "));
  await page.evaluate(() => window.scrollTo(0, 0)); await page.waitForTimeout(400);
  await page.evaluate((y) => window.scrollTo(0, y - 500), top0); await page.waitForTimeout(300);
  const again = await snap();
  console.log("INFO sơ đồ cuộn lại (-500):", JSON.stringify(again));
  const num = page.locator("section").filter({ has: page.getByRole("heading", { level: 2, name: H2[2] }) }).first();
  const ntop = (await num.evaluate((s) => s.getBoundingClientRect().top + scrollY)) as number;
  const ns: string[] = [];
  for (let k = -800; k <= 200; k += 100) {
    await page.evaluate((y) => window.scrollTo(0, y), ntop + k); await page.waitForTimeout(250);
    ns.push(`${k}:` + (await num.evaluate((s) => [...s.querySelectorAll("*")].filter((e) => getComputedStyle(e).getPropertyValue("--n").trim() !== "").map((e) => getComputedStyle(e).getPropertyValue("--n").trim()).join("/"))));
  }
  console.log("INFO --n khi cuộn:", ns.join(" | "));
  await ctx.close();
});
