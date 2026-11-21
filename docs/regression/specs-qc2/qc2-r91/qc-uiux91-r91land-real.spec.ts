import { gzipSync } from "node:zlib";
import { expect, test } from "@playwright/test";
test.skip(!process.env.QC_REAL, "QC_REAL");
const desk = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");

test("TC-UX08-PQ khách /chat, /gradebook → /login?next=; HTML trang giới thiệu không dữ liệu cá nhân", async ({ browser, request }, info) => {
  desk(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  for (const p of ["/chat", "/gradebook"]) {
    await page.goto(p);
    await page.waitForURL((u) => u.pathname.startsWith("/login"), { timeout: 20_000 });
    expect(page.url()).toContain("next=");
    expect(await page.locator("body").innerText()).not.toMatch(/Sổ điểm|Chat riêng có nguồn trả lời/);
  }
  const html = await (await request.get("/")).text();
  expect(html).not.toMatch(/@edupilot\.local|Edupilot#|Bearer |eyJhbGciOi|sk-[A-Za-z0-9]{10}/);
  expect(html).not.toMatch(/\b2022\d{4}\b/); // MSSV
  await ctx.close();
});

test("TC-UX09-8 video đầy đủ: controls, preload=none, playsinline, poster, không autoplay; không request trước khi bấm", async ({ browser }, info) => {
  desk(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  const vids: string[] = [];
  page.on("request", (q) => /gioi-thieu-v1\.(mp4|webm)/.test(q.url()) && vids.push(q.url()));
  await page.goto("/", { waitUntil: "load" });
  await page.waitForTimeout(1500);
  expect(vids, "request video trước khi bấm").toEqual([]);
  const attrs0 = await page.evaluate(() => { const v = document.querySelector("video"); return v ? { controls: v.hasAttribute("controls"), preload: v.getAttribute("preload"), playsinline: v.hasAttribute("playsinline"), poster: v.getAttribute("poster"), autoplay: v.hasAttribute("autoplay") } : null; });
  await page.getByRole("button", { name: /Xem video/ }).first().click();
  await page.waitForTimeout(800);
  const attrs = attrs0 ?? await page.evaluate(() => { const v = document.querySelector("video"); return v ? { controls: v.hasAttribute("controls"), preload: v.getAttribute("preload"), playsinline: v.hasAttribute("playsinline"), poster: v.getAttribute("poster"), autoplay: v.hasAttribute("autoplay") } : null; });
  const a2 = await page.evaluate(() => { const v = document.querySelector("video"); return v ? { controls: v.hasAttribute("controls"), preload: v.getAttribute("preload"), playsinline: v.hasAttribute("playsinline"), poster: v.getAttribute("poster"), autoplay: v.hasAttribute("autoplay") } : null; });
  console.log("INFO video attrs", JSON.stringify({ attrs0, a2, attrs }));
  const a = a2 ?? attrs;
  expect(a).not.toBeNull();
  expect(a!.controls).toBe(true); expect(a!.preload).toBe("none"); expect(a!.playsinline).toBe(true); expect(a!.autoplay).toBe(false); expect(a!.poster).toBeTruthy();
  await ctx.close();
});

for (const [w, h] of [[1440, 900], [375, 812]] as const) {
  test(`TC-UX08-44 JS gzip route / + LCP ${w}`, async ({ browser, request }, info) => {
    desk(info);
    const html = await (await request.get("/")).text();
    const urls = [...new Set([...html.matchAll(/\/_next\/static\/[^"' ]+\.js/g)].map((m) => m[0]))];
    let total = 0; const big: string[] = [];
    for (const u of urls) { const b = await (await request.get(u)).body(); const g = gzipSync(b).length; total += g; if (g > 40_000) big.push(`${u.split("/").pop()}=${(g / 1024).toFixed(1)}KB`); }
    const inl = [...html.matchAll(/<script(?![^>]*src)[^>]*>([\s\S]*?)<\/script>/g)].map((m) => m[1]);
    const inline = inl.filter((x) => !x.includes("__next_f")).reduce((s, x) => s + gzipSync(Buffer.from(x)).length, 0); // JS tự viết nội tuyến; không tính dữ liệu RSC (__next_f)
    const flight = inl.filter((x) => x.includes("__next_f")).reduce((s, x) => s + gzipSync(Buffer.from(x)).length, 0);
    console.log(`INFO JS gz ${w}: ${urls.length} tệp tĩnh ${(total / 1024).toFixed(1)} KB + inline ${(inline / 1024).toFixed(2)} KB (+ dữ liệu RSC ${(flight / 1024).toFixed(1)} KB, không phải JS); lớn: ${big.join(" ")}`);
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h } });
    const page = await ctx.newPage();
    await page.addInitScript(() => { (window as any).__lcp = []; new PerformanceObserver((l) => { for (const e of l.getEntries()) (window as any).__lcp.push({ t: Math.round(e.startTime), tag: (e as any).element?.tagName, cls: ((e as any).element?.className ?? "").toString().slice(-30), txt: ((e as any).element?.textContent ?? "").slice(0, 40) }); }).observe({ type: "largest-contentful-paint", buffered: true }); (window as any).__cls = 0; new PerformanceObserver((l) => { for (const e of l.getEntries() as any) if (!e.hadRecentInput) (window as any).__cls += e.value; }).observe({ type: "layout-shift", buffered: true }); });
    await page.goto("/", { waitUntil: "load" });
    await page.waitForTimeout(3000);
    const m = await page.evaluate(() => ({ lcp: (window as any).__lcp, cls: (window as any).__cls }));
    console.log(`INFO LCP ${w}: ${JSON.stringify(m)}`);
    expect(total + inline).toBeLessThanOrEqual(256 * 1024);
    await ctx.close();
  });
}
