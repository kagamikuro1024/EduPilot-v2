import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; aid: string; sub: Record<string, string>; sub2: Record<string, string> };
const out: Record<string, unknown> = {};
const save = () => writeFileSync("/tmp/q9s/ui9.json", JSON.stringify(out, null, 1));
test("@real UI9 10-04 phản hồi + 09-01 hộp xác nhận công bố", async ({ page }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "teacher");
  await page.goto(`/grading/${S.sub2["sv.kha"]}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  await page.getByRole("button", { name: "Giữ nguyên điểm" }).click(); await page.waitForTimeout(600);
  const t1 = await page.locator("main").innerText(); out.respLabel = (t1.match(/Phản hồi[^\n]*/) ?? [""])[0];
  const decide = page.getByRole("button", { name: "Quyết định" }); out.decideDisabledEmpty = await decide.isDisabled();
  const resp = page.getByRole("textbox", { name: /Phản hồi/ }); await resp.fill("ngắn"); out.decideDisabledShort = await decide.isDisabled();
  await resp.fill("Đã xem lại tiêu chí 2, điểm giữ nguyên."); out.decideDisabledOk = await decide.isDisabled();
  await page.getByRole("button", { name: "Sửa điểm" }).click(); await page.waitForTimeout(500);
  out.adjustedText = (await page.locator("main").innerText()).split("\n").filter((l) => /Sửa điểm|điểm mới|Điểm mới|Tổng/.test(l)).slice(0, 6);
  await shot(page, "p710-decide-form");
  // 09-01: chọn lẫn DRAFT + REVIEWED trong hàng chờ
  await page.goto(`/grading?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  out.statusOptions = await page.locator("main select").last().locator("option").allInnerTexts();
  await page.locator("main select").last().selectOption({ label: "Đã duyệt" }).catch(() => null); await page.waitForTimeout(1500);
  out.reviewedRows = await page.locator("main [role=row], main tbody tr").count();
  out.checkboxes = await page.locator("main input[type=checkbox]").count();
  const cb = page.locator("main input[type=checkbox]"); if (await cb.count()) await cb.first().check().catch(() => null); await page.waitForTimeout(600);
  out.publishBtns = await page.locator("main button", { hasText: "Công bố" }).evaluateAll((els) => els.map((e) => ({ t: (e.textContent ?? "").trim(), d: (e as HTMLButtonElement).disabled })));
  const pub = page.locator("main button", { hasText: "Công bố" }).first();
  if (await pub.isEnabled().catch(() => false)) { await pub.click(); await page.waitForTimeout(1000); out.dialog = (await page.locator("dialog[open]").innerText().catch(() => "")).replace(/\n+/g, " | "); out.dialogBtns = await page.locator("dialog[open] button").allInnerTexts(); await shot(page, "p709-confirm"); }
  save();
});
test("@real UI9 05-11 /settings/integrations", async ({ page }) => {
  for (const w of [1440, 390]) {
    await page.setViewportSize({ width: w, height: 900 });
    await uiLogin(page, "admin"); await page.goto(`/settings/integrations?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(4000);
    out[`int${w}`] = (await page.locator("main").innerText()).split("\n").map((s) => s.trim()).filter(Boolean).slice(0, 60);
    out[`int${w}btn`] = await page.locator("main button").allInnerTexts(); await shot(page, `p705-integrations-${w}`);
    out[`int${w}scroll`] = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth);
  }
  save();
});
