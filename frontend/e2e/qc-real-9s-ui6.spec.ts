import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string };
const out: Record<string, unknown> = {};
for (const who of ["ta", "teacher"]) {
  test(`@real UI6 nút Công bố ở hàng chờ: ${who}`, async ({ page }) => {
    await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
    await uiLogin(page, who); await page.goto(`/grading?course=${S.course}`); await page.locator("main").waitFor(); await page.getByText("Chấm bài").first().waitFor(); await page.waitForTimeout(4000);
    out[`${who}_user`] = (await page.getByRole("button", { name: /Tài khoản/ }).first().innerText()).replace(/\n/g, " ");
    out[`${who}_btns`] = await page.locator("main button").allInnerTexts();
    out[`${who}_publish`] = await page.locator("main button", { hasText: "Công bố" }).count();
    out[`${who}_publishDisabled`] = await page.locator("main button", { hasText: "Công bố" }).first().isDisabled({ timeout: 2000 }).catch(() => null);
    writeFileSync(`/tmp/q9s/ui6-${who}.json`, JSON.stringify(out, null, 1));
  });
}
