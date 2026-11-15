import { test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");

// QC sprint 10, TC-06-23: lớp chưa có câu APPROVED → /practice (SV lớp 761988, 375 và 1440).
const OUT = process.env.QC_OUT ?? "/tmp/qc10-shots";
for (const w of [1440, 375]) {
  test(`practice lớp rỗng ${w}`, async ({ browser }) => {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
    const page = await ctx.newPage();
    await page.goto("/login");
    await page.locator("input[type=email]").fill("sv.gioi@edupilot.local");
    await page.locator("input[type=password]").fill(process.env.QC_PW ?? "");
    await page.keyboard.press("Enter");
    await page.waitForURL((u) => !u.pathname.startsWith("/login"));
    await page.getByRole("button", { name: /Chọn lớp/ }).click();
    await page.getByText("761988").first().click();
    await page.waitForTimeout(800);
    await page.goto("/practice"); await page.waitForTimeout(1500);
    await page.screenshot({ path: `${OUT}/practice-empty-${w}.png` });
    console.log("TEXT", w, JSON.stringify((await page.locator("main").innerText()).slice(0, 300)));
    await ctx.close();
  });
}
