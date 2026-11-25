import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string };
test("@real UII 02-17 drawer: Tính vào điểm", async ({ page }) => {
  const out: Record<string, unknown> = {};
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "teacher");
  await page.goto(`/grading?course=${S.course}`); await page.getByText("Chấm bài").first().waitFor(); await page.waitForTimeout(2000);
  await page.getByRole("tab", { name: "Bài tập" }).click(); await page.waitForTimeout(1500);
  const rows = page.locator("main tr, main [role=row]").filter({ hasText: "22 / 40" }); const n = await rows.count(); out.rows22 = n; out.tries = [];
  for (let i = 0; i < n; i++) {
    await rows.nth(i).click(); await page.waitForTimeout(3500);
    const t = await page.locator("dialog[open]").innerText(); const line = t.split("\n").filter((l) => /tính vào điểm|Tính vào/i.test(l)); (out.tries as unknown[]).push(line);
    if (line.some((l) => l.startsWith("Tính vào điểm"))) { out.gradeLine = line; await shot(page, "p702-grade-link"); break; }
    await page.keyboard.press("Escape"); await page.waitForTimeout(800);
  }
  writeFileSync("/tmp/q9s/uii.json", JSON.stringify(out, null, 1)); return;
  await shot(page, "p702-grade-link");
  writeFileSync("/tmp/q9s/uii.json", JSON.stringify(out, null, 1));
});
