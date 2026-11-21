import { test } from "@playwright/test";
import { clearRL } from "./r4ui-lib";
// SmkT2: thăm dò DOM thật của các vai seed (chỉ in) để viết kiểm chứng GATEP2-26/27/28/30
const SEED = "Edupilot#Seed-2026";
for (const who of ["admin", "ta", "teacher", "sv.gioi", "sv.kha", "sv.moi"])
  test(`[EXPLORE ${who}]`, async ({ page }, info) => {
    if (info.project.name !== "desktop") test.skip();
    clearRL();
    const t0 = Date.now();
    await page.goto("/login");
    await page.getByLabel("Email").fill(`${who}@edupilot.local`);
    await page.getByLabel("Mật khẩu", { exact: true }).fill(SEED);
    await page.getByRole("button", { name: "Đăng nhập", exact: true }).click();
    await page.locator("[data-part=topbar]").waitFor({ timeout: 30000 });
    const tTop = Date.now() - t0;
    await page.waitForTimeout(2500);
    const nav = await page.locator("[data-part=sidebar] nav a").allInnerTexts();
    const prim = await page.locator('main [data-variant="primary"]:visible').allInnerTexts();
    const picker = page.getByRole("button", { name: /^Chọn lớp/ });
    let opts: string[] = [];
    if (await picker.count()) { await picker.first().click(); await page.waitForTimeout(600); opts = await page.getByRole("option").allInnerTexts().catch(() => []); if (!opts.length) opts = await page.getByRole("menuitem").allInnerTexts().catch(() => []); await page.keyboard.press("Escape"); }
    console.log("EXPLORE", who, JSON.stringify({ url: page.url().replace("https://localhost:1143", ""), ms: tTop, nav: nav.length, navText: nav.map((x) => x.replace(/\n/g, " ")), prim, picker: await picker.allInnerTexts(), opts, main: (await page.locator("main").innerText()).slice(0, 500).replace(/\n+/g, " | ") }));
  });
