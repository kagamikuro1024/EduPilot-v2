import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; sub: Record<string, string> };
const E = JSON.parse(readFileSync("/tmp/q9s/ui3.json", "utf8")) as { empty: string };
const out: Record<string, unknown> = {};
test("@real UI3 trạng thái rỗng / lỗi / nhãn sửa", async ({ page }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), E.empty);
  await uiLogin(page, "teacher");
  await page.goto(`/grading?course=${E.empty}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  out.empty = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 600); await shot(page, "p708-empty");
  await page.evaluate((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await page.route(/\/api\/v1\/courses\/[^/]+\/submissions(\?|$)/, (r) => r.abort());
  await page.goto(`/grading?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(5000);
  out.error = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 600); out.retryBtn = await page.getByRole("button", { name: /Thử lại/ }).count(); await shot(page, "p708-error");
  await page.unroute(/\/api\/v1\/courses\/[^/]+\/submissions(\?|$)/);
  await page.goto(`/grading/${S.sub["sv04"]}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  out.editedLabel = await page.getByText(/Đã sửa bởi giảng viên/).count();
  writeFileSync("/tmp/q9s/ui3out.json", JSON.stringify(out, null, 1));
});
