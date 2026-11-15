import { test } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.7 — TL 14.7: /settings sinh viên khi /api/v1 lỗi: mỗi mục có tối đa 1 alert; Nhắc 24 giờ + Thời gian học không ra nút / ô dùng được trên dữ liệu chưa tải.
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-set";
mkdirSync(OUT, { recursive: true });
for (const w of [1440, 375]) {
  test(`settings sv lỗi @${w}`, async ({ browser }) => {
    test.setTimeout(120_000);
    const ctx = await browser.newContext({ viewport: { width: w, height: w === 1440 ? 900 : 812 }, locale: "vi-VN", hasTouch: w < 500, ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    await page.goto(`${BASE}/login`);
    await page.locator("input[type=email]").fill("sv.kha@edupilot.local");
    await page.locator("input[type=password]").fill(process.env.QC_PW ?? "");
    await page.keyboard.press("Enter");
    await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
    await page.locator("[data-part=topbar]").waitFor();
    await page.goto(`${BASE}/`, { waitUntil: "domcontentloaded" });
    await page.waitForTimeout(800);
    let fail = false;
    const paths = new Set<string>();
    await page.route("**/api/v1/**", async (route) => {
      const p = new URL(route.request().url()).pathname;
      if (!fail || p.startsWith("/api/v1/auth/") || p === "/api/v1/me" || p.startsWith("/api/v1/me/courses")) return route.continue();
      if (route.request().method() === "GET") paths.add(p);
      return route.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi máy chủ." }) });
    });
    fail = true;
    await page.goto(`${BASE}/settings`, { waitUntil: "domcontentloaded" });
    await page.waitForTimeout(2500);
    const r = await page.evaluate(() => {
      const main = document.querySelector("main")!;
      const live = (e: Element) => (e as HTMLElement).offsetParent !== null && !(e as HTMLButtonElement).disabled && e.getAttribute("aria-disabled") !== "true";
      const secs = Array.from(main.querySelectorAll("[data-ep-section]")).map((s) => ({
        h2: s.querySelector("h2")?.textContent?.trim(),
        alerts: s.querySelectorAll("[role=alert]").length,
        retry: Array.from(s.querySelectorAll("[role=alert] button")).map((b) => b.textContent?.trim()),
        live: Array.from(s.querySelectorAll("button, input, select, textarea, [role=switch]")).filter((e) => live(e) && !e.closest("[role=alert]")).map((e) => (e.getAttribute("aria-label") || e.textContent || (e as HTMLInputElement).name || e.tagName).trim().slice(0, 28)),
      }));
      return { secs, alertsTotal: main.querySelectorAll("[role=alert]").length };
    });
    console.log(`QCINFO settings-sv@${w} GET hỏng khác nhau=${paths.size} ${JSON.stringify(paths.size ? [...paths] : [])} :: ${JSON.stringify(r)}`);
    await page.screenshot({ path: `${OUT}/settings-sv-error-${w}.png`, fullPage: true });
    await ctx.close();
  });
}
