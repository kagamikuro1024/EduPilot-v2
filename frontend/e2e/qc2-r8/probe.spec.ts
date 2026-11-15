import { test } from "@playwright/test";
import { cid, go, login, shot, IDS } from "./lib";
test("probe", async ({ page }) => {
  await login(page, "teacher"); await go(page, `/gradebook?course=${cid("QD8R6U")}`);
  console.log("H1", await page.locator("h1").allInnerTexts());
  console.log("BUTTONS", JSON.stringify((await page.getByRole("button").allInnerTexts()).slice(0, 25)), JSON.stringify(await page.getByRole("button").evaluateAll((els) => els.slice(0, 20).map((e) => e.getAttribute("aria-label")))));
  console.log("TH", JSON.stringify(await page.locator("thead th").allInnerTexts()));
  console.log("ROWS", await page.locator("[data-part=grade-row]").count(), "cells", await page.locator("input[data-cell]").count());
  const first = page.locator("input[data-cell]").first(); console.log("CELL", await first.getAttribute("data-cell"));
  console.log("PANELS", await page.locator("main [data-ep-panel]").count(), "NOTE", await page.getByText("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.").count());
  await shot(page, "probe-gradebook");
});
