// Sprint 9.1 pha 1 — trang giới thiệu: chuyển động hiện có, ngân sách JS / video, prefers-reduced-motion. Chỉ đọc.
//   node landing.mjs   → ../data/landing.json + ../shots/landing-scroll-<chế độ>-<rộng>-<n>.png
import { chromium } from "/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright-core/index.mjs";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SHOTS = path.join(HERE, "../shots");
const DATA = path.join(HERE, "../data");
const B = process.env.DEMO_URL ?? "https://v2202609419695521098.happysrv.de";
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const browser = await chromium.launch();
const out = [];
for (const width of [1440, 375]) for (const rm of ["no-preference", "reduce"]) {
  const ctx = await browser.newContext({ viewport: { width, height: width > 700 ? 900 : 812 }, reducedMotion: rm, bypassCSP: true, locale: "vi-VN" });
  const page = await ctx.newPage();
  const bytes = { js: 0, css: 0, img: 0, media: 0, font: 0, total: 0 };
  page.on("response", async (r) => {
    try {
      await r.finished();
      const len = (await r.request().sizes()).responseBodySize; // byte truyền qua mạng (đã nén)
      const raw = (await r.body().catch(() => Buffer.alloc(0))).length; // sau giải nén
      if (r.request().resourceType() === "script") bytes.jsRaw = (bytes.jsRaw ?? 0) + raw;
      const t = r.request().resourceType();
      bytes.total += len;
      if (t === "script") bytes.js += len; else if (t === "stylesheet") bytes.css += len; else if (t === "image") bytes.img += len; else if (t === "media") bytes.media += len; else if (t === "font") bytes.font += len;
    } catch {}
  });
  await page.addInitScript(() => { window.__lcp = 0; new PerformanceObserver((l) => { for (const e of l.getEntries()) window.__lcp = Math.round(e.startTime); }).observe({ type: "largest-contentful-paint", buffered: true }); });
  await page.goto(B + "/", { waitUntil: "load" });
  await sleep(1500);
  const rec = { width, reducedMotion: rm, steps: [] };
  rec.static = await page.evaluate(() => ({
    lcp: window.__lcp, docH: document.documentElement.scrollHeight,
    videos: [...document.querySelectorAll("video")].map((v) => ({ preload: v.preload, autoplay: v.autoplay, muted: v.muted, controls: v.controls, poster: !!v.poster, sources: [...v.querySelectorAll("source")].map((s) => s.type) })),
    imgs: [...document.querySelectorAll("img")].map((i) => ({ lazy: i.loading, w: i.naturalWidth, alt: i.alt?.slice(0, 30) })).slice(0, 12),
    rmRules: [...document.styleSheets].reduce((n, ss) => { try { return n + [...ss.cssRules].filter((r) => r.media && /prefers-reduced-motion/.test(r.media.mediaText)).length; } catch { return n; } }, 0),
    scrollTimeline: [...document.styleSheets].some((ss) => { try { return [...ss.cssRules].some((r) => /animation-timeline|scroll\(|view\(/.test(r.cssText)); } catch { return false; } }),
    ioUsed: typeof IntersectionObserver !== "undefined",
  }));
  const H = rec.static.docH;
  let n = 0;
  for (let y = 0; y < Math.min(H, 4800); y += width > 700 ? 800 : 700) {
    await page.evaluate((yy) => window.scrollTo({ top: yy, behavior: "instant" }), y);
    await sleep(120);
    const early = await page.evaluate(() => document.getAnimations().length); // ngay sau khi cuộn tới
    await sleep(900);
    const late = await page.evaluate(() => document.getAnimations().length);
    rec.steps.push({ y, animsJustAfterScroll: early, animsAfter1s: late });
    if (n < 4) await page.screenshot({ path: path.join(SHOTS, `landing-scroll-${rm === "reduce" ? "reduced" : "motion"}-${width}-${n}.png`) });
    n++;
  }
  rec.bytes = bytes;
  rec.jsKB = Math.round(bytes.js / 1024);
  rec.jsRawKB = Math.round((bytes.jsRaw ?? 0) / 1024);
  rec.totalKB = Math.round(bytes.total / 1024);
  out.push(rec);
  console.log(width, rm, "JS", rec.jsKB, "KB, tổng", rec.totalKB, "KB, LCP", rec.static.lcp, "ms, scroll-timeline", rec.static.scrollTimeline, "rmRules", rec.static.rmRules, "videos", rec.static.videos.length);
  await ctx.close();
}
fs.writeFileSync(path.join(DATA, "landing.json"), JSON.stringify(out, null, 1));
await browser.close();
