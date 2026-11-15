import { expect, test } from "@playwright/test";
import { cid, go, log, login, shot } from "./real-lib";
test("attendance 30 students with banner (TC-P6-05-19/20)", async ({ page }) => {
  const C = cid("QD8F3"); let dialogs = 0; page.on("response", (r) => { if (r.status() >= 400) log("us05att30", `HTTP ${r.status()} ${r.url()}`); }); page.on("dialog", () => dialogs++);
  await login(page, process.env.GV ?? "gv.s8f"); await go(page, `/attendance?course=${C}`); await page.waitForTimeout(7000);
  const banner = await page.getByText("Công thức điểm chưa xác nhận", { exact: true }).count();
  const rows = page.locator("[data-part=att-row]"); await rows.first().waitFor({ timeout: 45000 }); await page.waitForTimeout(1500); const n = await rows.count(); log("us05att30", `banner=${banner} rows=${n}`);
  await shot(page, "us05-att30-before");
  const t0 = Date.now();
  await rows.first().click({ position: { x: 5, y: 5 } }).catch(() => undefined);
  const keys = ["1", "1", "2", "3", "4"]; 
  for (let i = 0; i < n; i++) { await page.keyboard.press(keys[i % keys.length]); await page.waitForTimeout(40); }
  const marked = await page.locator("[data-part=att-status][aria-checked=true], [data-part=att-status][data-state=on], [data-part=att-status][aria-pressed=true]").count();
  log("us05att30", `marked(selectors)=${marked} state=${await page.locator("[data-part=att-save-state]").innerText()}`);
  await page.getByRole("button", { name: "Lưu điểm danh" }).click(); await page.waitForTimeout(2500);
  const el = (Date.now() - t0) / 1000; log("us05att30", `elapsed=${el}s dialogs=${dialogs} state=${await page.locator("[data-part=att-save-state]").innerText()} banner-after=${await page.getByText("Công thức điểm chưa xác nhận", { exact: true }).count()}`);
  await shot(page, "us05-att30-after");
});
