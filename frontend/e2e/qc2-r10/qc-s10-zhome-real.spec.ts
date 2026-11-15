import { test } from "@playwright/test";
test.skip(!process.env.QC_REAL, "x");
for (const [w, mob] of [[375, true], [375, false], [1440, false]] as const) {
  test(`home ${w} mobile=${mob}`, async ({ browser }) => {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: 812 }, locale: "vi-VN", hasTouch: mob, isMobile: mob });
    const page = await ctx.newPage();
    await page.goto("/login");
    await page.locator("input[type=email]").fill("sv.kha@edupilot.local");
    await page.locator("input[type=password]").fill(process.env.QC_PW ?? "");
    await page.keyboard.press("Enter");
    await page.waitForURL((u) => !u.pathname.startsWith("/login"));
    console.log("QCH after-login", w, mob, page.url());
    for (const wait of [500, 3000, 6000]) { await page.waitForTimeout(wait); console.log("QCH", w, mob, page.url(), JSON.stringify(await page.locator("h1").first().innerText().catch(() => "?"))); }
    await page.goto("/practice"); await page.waitForTimeout(1500); await page.goto("/"); await page.waitForTimeout(3000);
    console.log("QCH after /practice → /", w, mob, page.url(), JSON.stringify(await page.locator("h1").first().innerText().catch(() => "?")));
    await ctx.close();
  });
}
