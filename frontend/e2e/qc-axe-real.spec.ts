import AxeBuilder from "@axe-core/playwright";
import { test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.7 (P-107-15) — axe ở 390 (và 1440) trên các route vừa sửa: serious/critical = 0; vùng cuộn ngang có tabindex (scrollable-region-focusable).
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const WHO: Record<string, { user: string; routes: string[] }> = {
  teacher: { user: "teacher", routes: ["/observability", "/documents"] },
  admin: { user: "admin", routes: ["/observability"] },
  student: { user: "sv.kha", routes: ["/chat"] },
};
for (const [role, cfg] of Object.entries(WHO)) {
  for (const w of [390, 1440]) {
    test(`axe ${role} @${w}`, async ({ browser }) => {
      test.setTimeout(150_000);
      const ctx = await browser.newContext({ viewport: { width: w, height: w === 390 ? 844 : 900 }, locale: "vi-VN", ignoreHTTPSErrors: true });
      const page = await ctx.newPage();
      await page.goto(`${BASE}/login`);
      await page.locator("input[type=email]").fill(`${cfg.user}@edupilot.local`);
      await page.locator("input[type=password]").fill(process.env.QC_PW ?? "");
      await page.keyboard.press("Enter");
      await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
      const tb = page.locator("[data-part=topbar]"); await tb.waitFor();
      if (role !== "admin" && !(await tb.innerText()).includes("761987")) { await tb.locator("button").filter({ hasText: /\d{3,}|KC106|QC/ }).first().click(); await page.getByText("761987").first().click(); await page.waitForTimeout(1500); }
      for (const route of cfg.routes) {
        await page.goto(`${BASE}${route}`, { waitUntil: "domcontentloaded" });
        await page.waitForTimeout(3000);
        const ax = await new AxeBuilder({ page }).analyze();
        const bad = ax.violations.filter((v) => v.impact === "serious" || v.impact === "critical").map((v) => `${v.id}:${v.nodes.length}`);
        const scroll = await page.evaluate(() => Array.from(document.querySelectorAll<HTMLElement>("main *")).filter((e) => e.scrollWidth > e.clientWidth + 1 && ["auto", "scroll"].includes(getComputedStyle(e).overflowX)).map((e) => ({ tag: e.tagName, tabindex: e.getAttribute("tabindex"), role: e.getAttribute("role"), label: e.getAttribute("aria-label") ?? e.getAttribute("aria-labelledby"), cls: e.className.toString().slice(-18) })));
        console.log(`QCAC AXE-${role}${route}@${w} ${bad.length === 0 ? "PASS" : "FAIL"} serious/critical=${JSON.stringify(bad)} vùng cuộn ngang=${JSON.stringify(scroll)}`);
      }
      await ctx.close();
    });
  }
}
