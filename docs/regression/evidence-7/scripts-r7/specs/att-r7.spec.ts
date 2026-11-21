import { test, type Page, type BrowserContext } from "@playwright/test";
import { login, go, FX, shot, log, psql, newUser, expect } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("r7-att", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("r7-att", `INFO ${tc} ${note}`);
const F = FX("fx_att") as { C: string; C0: string; CE: string; S: string[]; teacher: string; ta: string };
const ses = (C: string, startMin = -30, durMin = 90) =>
  psql(`insert into class_sessions(course_id,session_no,starts_at,ends_at,room) select '${C}',coalesce(max(session_no),0)+1,now()+interval '${startMin} minutes'+random()*interval '1 second',now()+interval '${startMin + durMin} minutes','P.QC' from class_sessions where course_id='${F.C}' returning id`.replace("where course_id='" + F.C + "'", `where course_id='${C}'`)).split("\n")[0];
const rows = (p: Page) => p.locator("[data-part=student-row]");
const rad = (p: Page, i: number, st: string) => rows(p).nth(i).locator(`input[data-status=${st}]`);
const stateTxt = (p: Page) => p.locator("[data-part=att-save-state]").innerText();
const openAtt = async (p: Page, C: string, S?: string) => { await go(p, `/attendance?course=${C}${S ? `&session=${S}` : ""}`); await p.locator("[data-part=student-row]").first().waitFor({ timeout: 30000 }); };
const puts = (p: Page) => { const a: Array<{ body: any; t: number }> = []; p.on("request", (r) => { if (r.method() === "PUT" && /\/sessions\/[^/]+\/attendance$/.test(r.url())) a.push({ body: JSON.parse(r.postData() ?? "{}"), t: Date.now() }); }); return a; };
const count = (S: string, st: string) => psql(`select count(*) from attendance_records where session_id='${S}' and status='${st}'`);
const nameOf = async (p: Page, i: number) => (await rows(p).nth(i).locator(".ep-item-title").first().innerText()).trim();

test("33 Lưu điểm danh khi mất mạng xếp hàng complete:true", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, F.teacher); const S = ses(F.C); const reqs = puts(page); await openAtt(page, F.C, S);
  const done0 = psql(`select attendance_done_at is not null from class_sessions where id='${S}'`);
  await ctx.setOffline(true); await page.waitForTimeout(500);
  await rad(page, 8, "LATE").click(); await page.waitForTimeout(400);
  const bar = page.locator("[data-part=att-save-bar] button"); const barDis = await bar.isDisabled(); const barTxt = (await bar.innerText()).trim();
  const hdr = page.getByRole("button", { name: /^Lưu điểm danh/ }); const hdrDis = (await hdr.count()) ? await hdr.first().isDisabled() : null;
  let queued = false; if (hdrDis === false) { await hdr.first().click(); queued = true; }
  await page.waitForTimeout(800); const stOff = (await stateTxt(page).catch(() => "")).trim(); const bnr = (await page.locator("[data-part=offline-banner]").innerText().catch(() => "")).trim();
  const doneOff = psql(`select attendance_done_at is not null from class_sessions where id='${S}'`);
  await shot(page, "r7-p502-33-offline");
  await ctx.setOffline(false); await page.getByText(/Đã lưu \d\d:\d\d/).waitFor({ timeout: 30000 }).catch(() => undefined); await page.waitForTimeout(4000);
  const doneAt = psql(`select attendance_done_at is not null from class_sessions where id='${S}'`); const late = count(S, "LATE");
  const puts1 = reqs.map((r) => ({ complete: !!r.body.complete, n: (r.body.records ?? []).length }));
  await shot(page, "r7-p502-33-online");
  R("TC-P5-02-33", done0 === "f" && queued && doneOff === "f" && doneAt === "t" && late === "1" && puts1.some((x) => x.complete), `[DEF-R7-08, 131a5868] buổi mới ${S.slice(-6)} (done_at ban đầu=${done0}); setOffline(true), đổi 1 hàng → LATE; nút thanh dưới «${barTxt}» khoá=${barDis}; nút đầu trang «Lưu điểm danh» khoá=${hdrDis}; ${queued ? "đã bấm Lưu khi offline" : "KHÔNG bấm được khi offline"}; trạng thái «${stOff}», banner «${bnr}»; done_at lúc offline=${doneOff}; setOffline(false) → «Đã lưu»; sau 4 s: done_at đặt=${doneAt}, DB LATE=${late}; PUT gửi=${JSON.stringify(puts1)} (có complete:true=${puts1.some((x) => x.complete)})`);
  await ctx.close();
});

test("33b Lưu điểm danh offline khi không còn thay đổi chờ", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, F.teacher); const S = ses(F.C); const reqs = puts(page); await openAtt(page, F.C, S);
  await rad(page, 2, "ABSENT").click(); await page.getByText(/Đã lưu \d\d:\d\d/).waitFor({ timeout: 30000 }); await page.waitForTimeout(500);
  await ctx.setOffline(true); await page.waitForTimeout(500);
  const hdr = page.getByRole("button", { name: /^Lưu điểm danh/ }); const hdrDis = (await hdr.count()) ? await hdr.first().isDisabled() : null;
  let queued = false; if (hdrDis === false) { await hdr.first().click(); queued = true; }
  await page.waitForTimeout(800); const doneOff = psql(`select attendance_done_at is not null from class_sessions where id='${S}'`);
  await ctx.setOffline(false); await page.waitForTimeout(8000); const doneAt = psql(`select attendance_done_at is not null from class_sessions where id='${S}'`);
  const puts1 = reqs.map((r) => ({ complete: !!r.body.complete, n: (r.body.records ?? []).length }));
  R("TC-P5-02-33b", queued && doneOff === "f" && doneAt === "t", `buổi ${S.slice(-6)}: đã lưu 1 thay đổi khi có mạng; setOffline(true) không còn thay đổi chờ; nút đầu trang khoá=${hdrDis}, ${queued ? "đã bấm" : "không bấm được"}; done_at lúc offline=${doneOff}; có mạng 8 s: done_at=${doneAt}; PUT=${JSON.stringify(puts1)}`);
  await ctx.close();
});
