import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { call, login, needReal, shot, sql, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string };
const out: Record<string, unknown> = {};
const lines = (t: string) => t.split("\n").map((s) => s.trim()).filter(Boolean);
const FX = "/Users/kuro/Documents/TA_Agent_qc9sub/docs/specs/FEAT-assignments/fixtures/create-bt03.json";
test("@real UID 03-14 mạng chập chờn khi nộp", async ({ page, request }) => {
  const t = await login(request, "teacher"); const b = JSON.parse(readFileSync(FX, "utf8")); const code = "QD" + String(Date.now()).slice(-8); b.submission_code = code; b.due_at = new Date(Date.now() + 3 * 86400_000).toISOString().replace(/\.\d+Z$/, "Z");
  const r = await call(request, t, "POST", `/courses/${S.course}/assignments`, b, { "Idempotency-Key": "k" + code }); const id = ((r.json.assignment ?? r.json) as { id: string }).id;
  await call(request, t, "POST", `/courses/${S.course}/assignments/${id}/publish`, {}, { "Idempotency-Key": "p" + code });
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "sv31");
  const tail = async () => lines(await page.locator("main").innerText()).slice(-7);
  const reqs: string[] = []; page.on("request", (q) => { if (/presign|\/submit(\?|$)/.test(q.url())) reqs.push(`${q.method()} ${q.url().split("/api/v1")[1]}`); });
  await page.goto(`/assignments/${id}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3000);
  const file = { name: "bai.md", mimeType: "text/markdown", buffer: Buffer.from("# Bài\n" + "Nội dung bài làm SUNBURST. ".repeat(14)) };
  // 1) PUT lên kho bị cắt
  let abortPut = true; await page.route((u) => !u.pathname.startsWith("/api/") && u.pathname.length > 3, async (rt) => { if (rt.request().method() === "PUT" && abortPut) await rt.abort("connectionreset"); else await rt.continue(); });
  await page.locator("input[type=file]").first().setInputFiles(file); await page.waitForTimeout(500);
  await page.getByRole("button", { name: /Nộp bài/ }).first().click(); await page.waitForTimeout(4500);
  out.afterPutAbort = await tail(); out.fileKept = (await page.locator("main").innerText()).includes("bai.md"); out.retryBtn = await page.getByRole("button", { name: /Thử lại/ }).count(); await shot(page, "p703-retry");
  const n1 = reqs.filter((x) => /presign/.test(x)).length;
  // 2) Thử lại → xin presign mới
  abortPut = false; if (await page.getByRole("button", { name: /Thử lại/ }).count()) { await page.getByRole("button", { name: /Thử lại/ }).first().click(); await page.waitForTimeout(6000); }
  out.presignBefore = n1; out.presignAfterRetry = reqs.filter((x) => /presign/.test(x)).length; out.afterRetry = await tail();
  out.subCount = sql(`select count(*) from submissions where assignment_id='${id}'`)[0][0]; out.reqs = reqs;
  // 3) PUT xong nhưng submit bị cắt (đóng tab) → mở lại
  const r2 = await call(request, t, "POST", `/courses/${S.course}/assignments`, { ...b, submission_code: code + "B" }, { "Idempotency-Key": "k2" + code }); const id2 = ((r2.json.assignment ?? r2.json) as { id: string }).id;
  await call(request, t, "POST", `/courses/${S.course}/assignments/${id2}/publish`, {}, { "Idempotency-Key": "p2" + code });
  await page.unroute((u) => true).catch(() => null);
  await page.route(/\/submit(\?|$)/, (rt) => rt.request().method() === "POST" ? rt.abort("connectionreset") : rt.continue());
  await page.goto(`/assignments/${id2}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(2500);
  await page.locator("input[type=file]").first().setInputFiles(file); await page.waitForTimeout(500);
  await page.getByRole("button", { name: /Nộp bài/ }).first().click(); await page.waitForTimeout(5000);
  out.afterSubmitAbort = await tail();
  await page.unroute(/\/submit(\?|$)/); await page.goto(`/assignments/${id2}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  out.reopened = await tail(); out.finishBtn = await page.getByRole("button", { name: /Hoàn tất/ }).count(); await shot(page, "p703-incomplete");
  if (out.finishBtn) { await page.getByRole("button", { name: /Hoàn tất/ }).first().click(); await page.waitForTimeout(5000); out.afterFinish = await tail(); out.sub2 = sql(`select count(*) from submissions where assignment_id='${id2}'`)[0][0]; }
  writeFileSync("/tmp/q9s/uid.json", JSON.stringify(out, null, 1));
});
