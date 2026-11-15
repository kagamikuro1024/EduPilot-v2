import AxeBuilder from "@axe-core/playwright";
import { test } from "@playwright/test";
test.skip(!process.env.QC_REAL, "QC_REAL");
for (const [w, h] of [[1440, 900], [375, 812]] as const) {
  test(`r91 axe landing ${w}`, async ({ browser }, info) => {
    test.skip(info.project.name !== "desktop");
    const ctx = await browser.newContext({ viewport: { width: w, height: h }, ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    await page.goto("/", { waitUntil: "load" });
    await page.waitForTimeout(700);
    const r = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
    const out = r.violations.map((v) => ({ id: v.id, impact: v.impact, nodes: v.nodes.map((n) => ({ target: n.target, html: n.html.slice(0, 120), msg: (n.any[0]?.message ?? "").slice(0, 200), data: n.any[0]?.data })) }));
    console.log("AXE " + w + " " + JSON.stringify(out));
    await page.screenshot({ path: `/tmp/qc2-s91/landing-${w}.png`, fullPage: false });
    await ctx.close();
  });
}
