import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; aid: string };
const out: Record<string, unknown> = {};
const lines = (t: string) => t.split("\n").map((s) => s.trim()).filter(Boolean);
test("@real UI5 sv30 trạng thái; GV/TA tab Bài tập", async ({ page }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "sv30");
  for (const w of [1440, 375]) {
    await page.setViewportSize({ width: w, height: 900 });
    await page.goto(`/assignments/${S.aid}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3000);
    const L = lines(await page.locator("main").innerText()); out[`sv30_${w}_head`] = L.slice(0, 3); out[`sv30_${w}_tail`] = L.slice(-14);
    out[`sv30_${w}_crit`] = L.filter((l) => /\/\s?2,5|điểm tối đa|tối đa/i.test(l)).slice(0, 8);
    await shot(page, `p703-sv30-${w}`);
  }
  await page.setViewportSize({ width: 1440, height: 900 });
  const inp = page.locator("input[type=file]").first();
  await inp.setInputFiles({ name: "bai.md", mimeType: "text/markdown", buffer: Buffer.from("# Bài\n" + "Nội dung bài làm. SUNBURST. ".repeat(12)) }); await page.waitForTimeout(600);
  await page.getByRole("button", { name: /Nộp bài/ }).first().click(); await page.waitForTimeout(6000);
  const L2 = lines(await page.locator("main").innerText()); out.afterSubmitTail = L2.slice(-10); out.afterSubmitHead = L2.slice(0, 5);
  out.badAfter = ["12/85", "lượt 2", "nháp", "cờ", "confidence", "ai_scores"].filter((w) => L2.join(" ").toLowerCase().includes(w));
  await shot(page, "p703-sv30-after");
  // GV / TA tab Bài tập
  for (const u of ["teacher", "ta"]) {
    const p = await page.context().newPage(); await p.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
    await uiLogin(p, u); await p.goto(`/grading?course=${S.course}`); await p.locator("main").waitFor(); await p.waitForTimeout(3000);
    out[`${u}_tabs`] = await p.getByRole("tab").allInnerTexts(); out[`${u}_defaultTab`] = await p.locator("[role=tab][aria-selected=true]").allInnerTexts();
    await p.getByRole("tab", { name: "Bài tập" }).click(); await p.waitForTimeout(2500);
    out[`${u}_panel`] = lines(await p.locator("main").innerText()).slice(0, 40);
    out[`${u}_createBtn`] = await p.getByRole("button", { name: /Tạo bài tập/ }).count();
    out[`${u}_panels`] = await p.locator("[role=tabpanel]:visible").count();
    await shot(p, `p702-tabs-${u}`); await p.close();
  }
  writeFileSync("/tmp/q9s/ui5.json", JSON.stringify(out, null, 1));
});
