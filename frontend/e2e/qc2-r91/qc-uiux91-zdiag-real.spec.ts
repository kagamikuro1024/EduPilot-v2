import { test } from "@playwright/test";
test("diag", async ({ browser }) => {
  const page = await (await browser.newContext({ ignoreHTTPSErrors: true })).newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("sv.gioi@edupilot.local");
  await page.locator("input[type=password]").fill("Edupilot#Seed-2026");
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30_000 });
  for (const r of ["/documents", "/inbox"]) { await page.goto(r); await page.waitForTimeout(3000); console.log("DIAG", r, page.url(), JSON.stringify((await page.locator("main").innerText()).slice(0, 400))); }
});
