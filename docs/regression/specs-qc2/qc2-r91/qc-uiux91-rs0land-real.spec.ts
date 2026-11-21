/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkR (qc2, lượt 6): trang giới thiệu `/` trên Firefox / WebKit / không script: TC-UX08-28, UX08-48, UX09-11, UX09-13.
import { chromium, firefox, test, webkit, expect, type Browser, type Page } from "@playwright/test";
test.skip(!process.env.QC_REAL, "QC_REAL");
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");
const BASE = process.env.QC_BASE ?? "https://localhost:1143";
const hiddenBlocks = (page: Page) => page.evaluate(() => {
  const out: string[] = [];
  for (const e of document.querySelectorAll("main *, body > *")) {
    const el = e as HTMLElement; if (!el.innerText?.trim() || el.closest("dialog, [hidden], [inert], [aria-hidden=true], .sr-only, noscript") || el.tagName === "SCRIPT" || el.tagName === "STYLE") continue;
    if (el.children.length > 0 && [...el.children].some((c) => (c as HTMLElement).innerText?.trim())) continue; // chỉ lá chứa chữ
    const cs = getComputedStyle(el);
    if (cs.opacity === "0" || cs.visibility === "hidden" || cs.display === "none") out.push(`${el.tagName}.${el.className.toString().slice(0, 24)}:${el.innerText.trim().slice(0, 24)}`);
  }
  return out;
});
const scrollAll = async (page: Page) => { const h = await page.evaluate(() => document.documentElement.scrollHeight); for (let y = 0; y < h; y += 500) { await page.evaluate((v) => window.scrollTo(0, v), y); await page.waitForTimeout(120); } await page.waitForTimeout(1800); };

for (const [name, bt] of [["firefox", firefox], ["webkit", webkit]] as const) {
  test(`TC-UX08-28 ${name} 1440: không có animation-timeline ⇒ trang hiện đủ nội dung, không khối ẩn, không tràn ngang`, async ({}, info) => {
    D(info); test.setTimeout(180_000);
    const b: Browser = await bt.launch();
    const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, baseURL: BASE });
    const page = await ctx.newPage();
    const errs: string[] = []; page.on("pageerror", (e) => errs.push(e.message));
    await page.goto("/", { waitUntil: "load" }); await page.waitForTimeout(1500);
    const sup = await page.evaluate(() => CSS.supports("animation-timeline: view()"));
    console.log(`INFO UX08-28 ${name}: CSS.supports(animation-timeline: view()) =`, sup, "| ua", await page.evaluate(() => navigator.userAgent.slice(-40)));
    await expect(page.locator("h1").first()).toBeVisible();
    await scrollAll(page);
    const hid = await hiddenBlocks(page);
    console.log(`INFO UX08-28 ${name} khối ẩn:`, JSON.stringify(hid));
    expect(hid, "mọi khối chữ hiện (opacity > 0, không ẩn)").toEqual([]);
    const secs = await page.locator("main section, main > div > section").count();
    expect(secs, "có các mục của trang").toBeGreaterThanOrEqual(5);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    expect(errs, "không pageerror").toEqual([]);
    await b.close();
  });
}

test("TC-UX08-48 Chromium tắt JavaScript: hiện đủ nội dung, không khối ẩn vĩnh viễn (opacity 0 / visibility hidden)", async ({}, info) => {
  D(info); test.setTimeout(120_000);
  const b = await chromium.launch();
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, javaScriptEnabled: false, viewport: { width: 1440, height: 900 }, baseURL: BASE });
  const page = await ctx.newPage();
  await page.goto("/", { waitUntil: "load" }); await page.waitForTimeout(2500);
  await expect(page.locator("h1").first()).toBeVisible();
  const txt = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  const hid = await hiddenBlocks(page);
  console.log("INFO UX08-48 JS tắt: độ dài chữ", txt.length, "| khối ẩn:", JSON.stringify(hid));
  expect(txt.length, "đủ nội dung").toBeGreaterThan(800);
  expect(txt).toContain("Vận hành lớp học có trợ lý AI");
  expect(hid, "không khối ẩn vĩnh viễn").toEqual([]);
  await b.close();
});

test("TC-UX09-11 375 hasTouch: hộp thoại video, điều khiển gốc, vùng chạm nút đóng ≥ 44 px, video co theo bề rộng", async ({}, info) => {
  D(info); test.setTimeout(180_000);
  const b = await chromium.launch();
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, baseURL: BASE });
  const page = await ctx.newPage();
  await page.goto("/", { waitUntil: "load" }); await page.waitForTimeout(1500);
  await page.getByRole("button", { name: /Xem video/ }).first().tap();
  const v = page.locator("dialog[open] video, [role=dialog] video").first();
  await expect(v).toBeVisible({ timeout: 15_000 });
  await page.waitForTimeout(1200);
  const m = await v.evaluate((e: HTMLVideoElement) => { const r = e.getBoundingClientRect(); return { controls: e.controls, w: Math.round(r.width), h: Math.round(r.height), left: Math.round(r.left), right: Math.round(r.right), ready: e.readyState }; });
  console.log("INFO UX09-11 video:", JSON.stringify(m));
  expect(m.controls, "điều khiển gốc").toBe(true);
  expect(m.right, "không tràn ngang").toBeLessThanOrEqual(375); expect(m.left).toBeGreaterThanOrEqual(0);
  expect(m.h, "đủ cao cho thanh điều khiển gốc (≥ 44)").toBeGreaterThanOrEqual(44);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  // phát / dừng bằng bàn phím trên điều khiển gốc
  await v.focus(); await page.keyboard.press("Space"); await page.waitForTimeout(1500);
  const playing = await v.evaluate((e: HTMLVideoElement) => !e.paused && e.currentTime > 0);
  console.log("INFO UX09-11 phát bằng Space:", playing);
  expect(playing, "phát được bằng điều khiển gốc").toBe(true);
  await page.keyboard.press("Space"); await page.waitForTimeout(400);
  expect(await v.evaluate((e: HTMLVideoElement) => e.paused), "dừng được").toBe(true);
  const close = page.locator("dialog[open], [role=dialog]").first().getByRole("button", { name: /Đóng/ }).first();
  const cb = (await close.boundingBox())!;
  console.log("INFO UX09-11 nút đóng:", JSON.stringify(cb));
  expect(cb.width, "nút đóng ≥ 44").toBeGreaterThanOrEqual(44); expect(cb.height).toBeGreaterThanOrEqual(44);
  await b.close();
});

for (const [name, bt] of [["chromium", chromium], ["webkit", webkit]] as const) {
  test(`TC-UX09-13 ${name}: phát rồi currentTime = 30 ⇒ phát tiếp từ 30 s`, async ({}, info) => {
    D(info); test.setTimeout(240_000);
    const b = await bt.launch(name === "chromium" ? { args: ["--autoplay-policy=no-user-gesture-required"] } : {});
    const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, baseURL: BASE });
    const page = await ctx.newPage();
    await page.goto("/", { waitUntil: "load" }); await page.waitForTimeout(1200);
    await page.getByRole("button", { name: /Xem video/ }).first().click();
    const v = page.locator("dialog[open] video, [role=dialog] video").first();
    await expect(v).toBeVisible({ timeout: 15_000 });
    const r = await v.evaluate(async (e: HTMLVideoElement) => {
      const w = (ev: string) => new Promise<void>((res) => e.addEventListener(ev, () => res(), { once: true }));
      e.muted = true;
      await e.play().catch(() => undefined);
      await new Promise((res) => setTimeout(res, 1200));
      const t0 = e.currentTime; const dur = e.duration;
      const seeked = w("seeked"); e.currentTime = 30; await Promise.race([seeked, new Promise((res) => setTimeout(res, 8000))]);
      const t1 = e.currentTime;
      await new Promise((res) => setTimeout(res, 2500));
      return { t0, dur, t1, t2: e.currentTime, paused: e.paused, ended: e.ended, err: e.error?.code ?? null };
    });
    console.log(`INFO UX09-13 ${name}:`, JSON.stringify(r));
    expect(r.err).toBeNull(); expect(r.t0, "đã phát trước khi tua").toBeGreaterThan(0);
    expect(r.t1, "tua tới 30 s").toBeGreaterThanOrEqual(29.5);
    expect(r.t2, "phát tiếp từ 30 s (không quay về 0)").toBeGreaterThan(r.t1 + 1);
    expect(r.paused).toBe(false);
    await b.close();
  });
}
