import { test } from "@playwright/test";
test("dbg", async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, ignoreHTTPSErrors: true, locale: "vi-VN" }); const page = await ctx.newPage();
  await page.goto("https://localhost:1143/login"); await page.getByLabel("Email").fill("r108t@edupilot.local"); await page.getByLabel("Mật khẩu", { exact: true }).fill("Edupilot#Seed-2026"); await page.getByRole("button", { name: "Đăng nhập" }).click(); await page.waitForTimeout(3000);
  const sid = process.env.SID!;
  await page.locator("button[aria-label^='Chọn lớp']").first().click(); await page.getByRole("menuitem", { name: /^QC-AS\b/ }).first().click(); await page.waitForTimeout(1200); console.log("AFTERPICK", await page.locator("button[aria-label^='Chọn lớp']").first().getAttribute("aria-label"));
  page.on("response", (r) => { if (r.url().includes("/api/")) console.log("RESP", r.status(), r.url().replace("https://localhost:1143", "").slice(0, 110)); });
  await page.goto(`https://localhost:1143/attendance?session=${sid}`); await page.waitForTimeout(6000);
  console.log("TOPBAR", (await page.locator("[data-part=topbar]").innerText()).replace(/\n/g, " | ").slice(0, 200));
  console.log("MAIN", (await page.locator("main").innerText()).replace(/\n/g, " | ").slice(0, 500));
  console.log("WL", await page.locator("[data-part=window-line]").count(), await page.locator("[data-part=window-line]").first().getAttribute("data-state").catch(() => "-"));
});
