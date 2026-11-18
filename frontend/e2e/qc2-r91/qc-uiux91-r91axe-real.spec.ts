import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
test.skip(!process.env.QC_REAL, "QC_REAL");
// TC-UX08-50 (US-08 AC16): axe 0 serious/critical trên `/` ngay sau tải (+700 ms), 3 lượt mỗi cỡ
for (const [w, h] of [[1440, 900], [375, 812]] as const) {
  for (const k of [1, 2, 3]) {
    test(`TC-UX08-50 axe landing ${w} lượt ${k}`, async ({ browser }, info) => {
      test.skip(info.project.name !== "desktop");
      const ctx = await browser.newContext({ viewport: { width: w, height: h }, ignoreHTTPSErrors: true });
      const page = await ctx.newPage();
      await page.goto("/", { waitUntil: "load" });
      await page.waitForTimeout(700);
      const r = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
      const bad = r.violations.filter((v) => v.impact === "serious" || v.impact === "critical");
      console.log("AXE " + w + " " + JSON.stringify(bad.map((v) => ({ id: v.id, n: v.nodes.length, t: v.nodes.slice(0, 3).map((n) => n.target) }))));
      await ctx.close();
      expect(bad.map((v) => `${v.id}×${v.nodes.length}`)).toEqual([]);
    });
  }
}
