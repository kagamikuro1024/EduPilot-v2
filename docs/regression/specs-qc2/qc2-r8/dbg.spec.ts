import { test } from "@playwright/test";
import { cid, go, login } from "./lib";
test("dbg enter", async ({ page }) => {
  await login(page, "teacher"); await go(page, `/gradebook?course=${cid("QD8R6U")}`);
  page.on("request", (q) => { if (q.method() !== "GET") console.log("REQ", q.method(), q.url().slice(-60)); });
  const rows = page.locator("[data-part=grade-row]"); const c0 = rows.nth(0).locator("input[data-cell]").first();
  console.log("rows", await rows.count(), "c0", await c0.inputValue(), await c0.getAttribute("data-cell"), "disabled", await c0.isDisabled(), "readonly", await c0.getAttribute("readonly"));
  await c0.focus(); await c0.fill("8,5"); console.log("after fill", await c0.inputValue()); await c0.press("Enter"); await page.waitForTimeout(3000); console.log("after enter", await c0.inputValue(), JSON.stringify(await page.getByRole("alert").allInnerTexts()));
});
