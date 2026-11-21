import { expect, test } from "@playwright/test";
import { apiLogin, em } from "./r4ui-lib";

test("[DBG offline invite]", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  await apiLogin(page, em("adm"));
  await page.goto("/admin/users");
  await expect(page.getByRole("heading", { name: "Người dùng", level: 1 })).toBeVisible({ timeout: 30_000 });
  await page.getByRole("button", { name: "Mời giảng viên" }).click();
  const keys: string[] = [];
  await page.route("**/api/v1/admin/users", (r) => {
    if (r.request().method() !== "POST") return r.fallback();
    keys.push(String(r.request().headers()["idempotency-key"]));
    return r.abort("connectionfailed");
  });
  await page.getByLabel("Email", { exact: true }).fill(`qc2iu.dbg${Date.now()}@edupilot.local`);
  await page.getByLabel("Họ và tên").fill("QC2IU Dbg");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  for (const t of [1000, 4000, 12000, 30000]) {
    await page.waitForTimeout(t === 1000 ? 1000 : t - [1000, 4000, 12000, 30000][[1000, 4000, 12000, 30000].indexOf(t) - 1]);
    const html = await page.locator("form").first().innerHTML();
    const alerts = await page.getByRole("alert").allInnerTexts();
    const banner = await page.locator("[data-part=offline-banner]").innerText();
    console.log(`T+${t}ms keys=${keys.length} alerts=${JSON.stringify(alerts)} banner=${JSON.stringify(banner)} btnGuiLai=${await page.getByRole("button", { name: "Gửi lại" }).count()} html=${html.replace(/\s+/g, " ").slice(-700)}`);
  }
});
