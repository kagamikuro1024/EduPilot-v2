import { appendFileSync, readFileSync } from "node:fs";
import { test, type Browser } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
const RES = process.env.QC_RES ?? `${OUT}/results_106.jsonl`;
const FX = JSON.parse(readFileSync("/Users/kuro/.config/edupilot-team/qc3-work/g05fx.json", "utf8")) as { a1: string; a2: string };
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 330)); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 25_000 });
const CID = () => sql(`select id from courses where class_code='QD9G'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
test("TC-G05-12 Chấm lại: tiến độ SSE, nút khoá lúc chạy", async ({ browser }) => {
  test.setTimeout(260_000);
  const { ctx, page } = await pageAs(browser, "teacher"); const evs: string[] = []; const jobReqs: string[] = [];
  page.on("response", (r) => { if (/\/events/.test(r.url())) evs.push(String(r.status())); if (/\/jobs\//.test(r.url())) jobReqs.push(r.url().replace(/^.*\/api\/v1/, "").slice(0, 60)); });
  await page.goto(`/grading?tab=queue&assignment=${FX.a1}`); await page.waitForTimeout(4000);
  const btn = page.getByRole("button", { name: "Chấm lại" }); const n0 = await btn.count(); const notice0 = (await page.locator("main").innerText()).match(/\d+ bài chấm theo bản cũ[^\n]*/)?.[0];
  await btn.first().click(); const samples: string[] = []; let disabled = false, busyTxt = "";
  for (let i = 0; i < 45; i++) { await page.waitForTimeout(3000); const b = page.getByRole("button", { name: /Chấm lại|Đang chấm lại/ }).first(); if (await b.count()) { const d = (await b.isDisabled()) || (await b.getAttribute("aria-busy")) === "true"; disabled = disabled || d; busyTxt = (await b.innerText()).replace(/\n/g, " "); } const t = await page.locator("main").innerText(); const rowsRun = (t.match(/Đang chấm/g) ?? []).length; if (i % 6 === 0) await page.screenshot({ path: `${OUT}/s106-g0512-${i}.png` }); const m = t.match(/(Đang chấm lại[^\n]*|Đã xếp[^\n]*|Đã chấm xong[^\n]*|\d+\s*\/\s*\d+[^\n]*)/); samples.push((m ? m[0].slice(0, 50) : "-") + ` [hàng «Đang chấm»=${rowsRun}]`); }
  await page.screenshot({ path: `${OUT}/s106-g0512.png` });
  const fin = sql(`select count(*) filter (where r.round=2 and r.run_state='DONE') from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${FX.a1}'`)[0][0];
  verdict("TC-G05-12", n0 > 0 && disabled && Number(fin) > 0, `/grading?assignment: «${notice0}» + nút «Chấm lại» ×${n0}; sau bấm: nút khoá/busy=${disabled} (chữ nút «${busyTxt}»); mẫu dòng mỗi 3 s: ${JSON.stringify([...new Set(samples)])}; yêu cầu /events ${JSON.stringify([...new Set(evs)])}, /jobs ${jobReqs.length}; cuối: round 2 DONE=${fin}`); await ctx.close();
});
test("TC-G05-17 công bố lô có bài miễn", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=queue&assignment=${FX.a2}&status=REVIEWED`); await page.waitForTimeout(4500);
  const rows = page.getByRole("row"); const nrows = await rows.count(); const cb = page.getByRole("checkbox"); const ncb = await cb.count(); console.log("rows", nrows, "checkbox", ncb, (await page.locator("main").innerText()).slice(0, 300).replace(/\n/g, " ¦ "));
  await cb.first().check().catch(() => {}); // chọn tất cả
  const pub = page.getByRole("button", { name: "Công bố" }).first(); await pub.waitFor(); const en = await pub.isEnabled(); await pub.click(); await page.waitForTimeout(1200);
  const dlg = page.getByRole("alertdialog").or(page.getByRole("dialog")).last(); const dt = (await dlg.innerText().catch(() => "")).replace(/\n/g, " ¦ "); await page.screenshot({ path: `${OUT}/s106-g0517-dlg.png` });
  const exemptInDlg = /bài chấm theo bản (lưu ý )?cũ/.test(dt);
  await dlg.getByRole("button", { name: /Công bố/ }).last().click(); await page.waitForTimeout(3500);
  const after = (await page.locator("main").innerText()).replace(/\n/g, " ¦ "); const line = after.match(/\d+ bài chấm theo bản[^¦]*/)?.[0] ?? "";
  const pubN = sql(`select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${FX.a2}' and r.status='PUBLISHED'`)[0][0]; await page.screenshot({ path: `${OUT}/s106-g0517-after.png` });
  verdict("TC-G05-17", exemptInDlg && /đã sửa tay/.test(line) && Number(pubN) >= 1, `chọn ${ncb} ô; nút «Công bố» bật=${en}; hộp xác nhận: «${dt.slice(0, 260)}»; dòng «bản cũ» TRONG hộp=${exemptInDlg}; sau xác nhận: «${line}»; PUBLISHED trong DB=${pubN}`); await ctx.close();
});
