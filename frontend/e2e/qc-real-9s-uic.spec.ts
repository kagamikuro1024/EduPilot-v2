import { readFileSync, writeFileSync, existsSync } from "node:fs";
import { test } from "@playwright/test";
import { call, login, needReal, shot, sql, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string };
const PH = process.env.PHASE ?? "1"; const ST = "/tmp/q9s/uic-state.json";
const lines = (t: string) => t.split("\n").map((s) => s.trim()).filter(Boolean);
const FX = "/Users/kuro/Documents/TA_Agent_qc9sub/docs/specs/FEAT-assignments/fixtures/create-bt03.json";
test(`@real UIC 03-02 phase ${PH}`, async ({ page, request }) => {
  const out: Record<string, unknown> = existsSync(ST) && PH !== "1" ? JSON.parse(readFileSync(ST, "utf8")) : {};
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "sv30");
  const tail = async () => lines(await page.locator("main").innerText()).slice(-8);
  if (PH === "1") {
    const t = await login(request, "teacher"); const b = JSON.parse(readFileSync(FX, "utf8")); const code = "QC" + String(Date.now()).slice(-8); b.submission_code = code; b.due_at = new Date(Date.now() + 86400_000).toISOString().replace(/\.\d+Z$/, "Z");
    const r = await call(request, t, "POST", `/courses/${S.course}/assignments`, b, { "Idempotency-Key": "k" + code }); const id = ((r.json.assignment ?? r.json) as { id: string }).id;
    await call(request, t, "POST", `/courses/${S.course}/assignments/${id}/publish`, {}, { "Idempotency-Key": "p" + code });
    sql(`update assignments set due_at = now() - interval '1 day' where id='${id}'`); out.id = id;
    await page.goto(`/assignments/${id}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3000); out.s1 = await tail();
    await page.locator("input[type=file]").first().setInputFiles({ name: "bai.md", mimeType: "text/markdown", buffer: Buffer.from("# Bài\n" + "Nội dung bài làm SUNBURST. ".repeat(14)) }); await page.waitForTimeout(500);
    await page.getByRole("button", { name: /Nộp bài/ }).first().click(); await page.waitForTimeout(6000); out.s2 = await tail();
    out.s2bad = ["12/85", "lượt 2", "nháp", "cờ", "confidence", "ai_scores", "Cần xem kỹ"].filter((w) => (out.s2 as string[]).join(" ").includes(w)); await shot(page, "p703-state-submitted");
  } else {
    await page.goto(`/assignments/${out.id}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
    if (PH === "2") { out.s3 = await tail(); out.s3bad = ["12/85", "lượt 2", "nháp", "cờ", "confidence", "ai_scores", "Cần xem kỹ", "6,", "5,"].filter((w) => (out.s3 as string[]).join(" ").includes(w)); out.s3text = lines(await page.locator("main").innerText()).filter((l) => /chấm|Đang|điểm|Kết quả/.test(l)).slice(0, 6); await shot(page, "p703-state-grading"); }
    if (PH === "3") { out.s5 = lines(await page.locator("main").innerText()).filter((l) => /điểm|Điểm|Trừ|\/ 10|Kết quả/.test(l)).slice(0, 8); await shot(page, "p703-state-graded"); }
  }
  writeFileSync(ST, JSON.stringify(out, null, 1));
});
