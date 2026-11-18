import { appendFileSync } from "node:fs";
import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Browser } from "@playwright/test";
import { call, login, needReal, sql, PW, OUT } from "./qc-real-lib";

// QC smoke S10: TC-12-13 (không dữ liệu mô phỏng), TC-12-19 (axe), TC-12-20 (reduced motion) trên ep-qc, lớp QD10A.
const BASE = process.env.QC_BASE ?? "https://localhost:943";
const RES = process.env.QC_RES ?? `${OUT}/results_m.jsonl`;
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 260)); };
test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
test.use({ actionTimeout: 30_000 });
test.beforeAll(() => needReal());
const ID = () => sql(`select id from courses where class_code='QD10A'`)[0][0];
async function pageAs(browser: Browser, user: string, w: number, reduced = false) {
  const ctx = await browser.newContext({ baseURL: BASE, ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", reducedMotion: reduced ? "reduce" : "no-preference", timezoneId: "Asia/Ho_Chi_Minh" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), ID());
  const page = await ctx.newPage();
  await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page };
}
async function openAttempt(request: import("@playwright/test").APIRequestContext, user: string, body: Record<string, unknown>) {
  const id = ID(); const t = await login(request, user);
  for (const [x] of sql(`select id from practice_attempts where course_id='${id}' and status='IN_PROGRESS' and student_id=(select id from users where email='${user}@edupilot.local')`)) await call(request, t, "POST", `/courses/${id}/practice/attempts/${x}/submit`, {});
  const r = await call(request, t, "POST", `/courses/${id}/practice/start`, body); expect([200, 201]).toContain(r.status); return r.json as { id: string };
}

test("TC-12-19 axe: /practice, /practice/[id], /practice/history, /questions — serious/critical = 0 ở 1440 và 375", async ({ browser, request }) => {
  test.setTimeout(240_000); const out: string[] = []; let all0 = true;
  const a = await openAttempt(request, "sv.kha", { mode: "TOPIC", topic: "QC-r10-Y", count: 5 });
  for (const w of [1440, 375]) {
    for (const [user, route] of [["sv.kha", "/practice"], ["sv.kha", `/practice/${a.id}`], ["sv.kha", "/practice/history"], ["teacher", "/questions"]] as const) {
      const { ctx, page } = await pageAs(browser, user, w);
      await page.goto(route); await page.waitForTimeout(2500);
      const ax = await new AxeBuilder({ page }).analyze(); const bad = ax.violations.filter((v) => v.impact === "serious" || v.impact === "critical").map((v) => `${v.id}:${v.nodes.length}`);
      out.push(`${route.replace(a.id, "[id]")}@${w}: ${bad.length ? bad.join(",") : "0"}`); if (bad.length) all0 = false;
      await ctx.close();
    }
  }
  verdict("TC-12-19 (axe)", all0, `axe serious/critical: ${out.join("; ")}`);
});

test("TC-12-20 reduced motion: không animation đang chạy ở các màn mới; đồng hồ vẫn chạy, chữ số cố định", async ({ browser, request }) => {
  test.setTimeout(240_000); const out: string[] = []; let calm = true;
  const a = await openAttempt(request, "sv.kha", { mode: "TOPIC", topic: "QC-r10-Y", count: 5 });
  const id = ID(); const t = await login(request, "sv.kha"); const mrun = await call(request, t, "POST", `/courses/${id}/practice/attempts/${a.id}/submit`, {});
  const m = await call(request, t, "POST", `/courses/${id}/practice/start`, { mode: "MOCK", count: 20, duration_minutes: 20, matrix: [{ topic: "QC-r10-Y", count: 12 }, { topic: "QC-r10-X", count: 8 }] }); expect([200, 201]).toContain(m.status);
  const mid = (m.json as { id: string }).id;
  const { ctx, page } = await pageAs(browser, "sv.kha", 1440, true);
  for (const route of ["/practice", `/practice/${mid}`, "/practice/history"]) {
    await page.goto(route); await page.waitForTimeout(2500);
    const run = await page.evaluate(() => document.getAnimations().filter((x) => x.playState === "running" && (x.effect?.getTiming().iterations === Infinity || (Number(x.effect?.getTiming().duration) || 0) > 1)).map((x) => `${(x as CSSAnimation).animationName ?? (x as CSSTransition).transitionProperty ?? "anim"}`));
    out.push(`${route.replace(mid, "[id]")}: animation đang chạy ${JSON.stringify(run)}`); if (run.length) calm = false;
  }
  await page.goto(`/practice/${mid}`); await page.waitForTimeout(1500);
  const clock = page.getByText(/\d{1,2}:\d{2}/).first(); const t1 = await clock.innerText(); await page.waitForTimeout(3200); const t2 = await clock.innerText();
  const fv = await clock.evaluate((e) => getComputedStyle(e).fontVariantNumeric); const tab = /tabular/.test(fv);
  out.push(`đồng hồ ${t1} → ${t2} (chạy ${t1 !== t2}); font-variant-numeric «${fv}»`);
  await call(request, t, "POST", `/courses/${id}/practice/attempts/${mid}/submit`, {}); void mrun;
  verdict("TC-12-20", calm && t1 !== t2 && tab, out.join("; ")); await ctx.close();
});

test("TC-12-13 không dữ liệu mô phỏng cứng: /practice, /practice/history, /questions khớp DB", async ({ browser }) => {
  test.setTimeout(180_000); const id = ID(); const rows: string[] = [];
  const dbTopics = sql(`select topic, count(*) from question_bank where course_id='${id}' and review_status='APPROVED' group by 1 order by 1`);
  const dbTotal = sql(`select count(*) from question_bank where course_id='${id}'`)[0][0]; const dbPend = sql(`select count(*) from question_bank where course_id='${id}' and review_status='PENDING'`)[0][0];
  const sv = await pageAs(browser, "sv.kha", 1440); await sv.page.goto("/practice"); await sv.page.waitForTimeout(3000);
  const ptxt = await sv.page.locator("main").innerText(); await sv.page.screenshot({ path: `${OUT}/s10-1213-practice.png` });
  const shown = dbTopics.filter(([tp]) => ptxt.includes(tp)).length; const fake = ["Hàm băm", "Mã hoá đối xứng", "Giao thức TLS"].filter((x) => ptxt.includes(x));
  rows.push(`/practice: ${shown}/${dbTopics.length} chủ đề DB hiện; chuỗi mô phỏng cũ ${JSON.stringify(fake)}`);
  const tch = await pageAs(browser, "teacher", 1440); await tch.page.goto("/questions"); await tch.page.waitForTimeout(3500);
  const qtxt = await tch.page.locator("main").innerText(); await tch.page.screenshot({ path: `${OUT}/s10-1213-questions.png` });
  const nums = [...qtxt.matchAll(/(\d+)\s*(?:câu|câu hỏi)/gi)].map((x) => x[1]); const hasDb = nums.includes(dbTotal) || nums.includes(dbPend) || qtxt.includes(dbTotal);
  rows.push(`/questions: số trong trang ${JSON.stringify(nums.slice(0, 8))}; DB tổng ${dbTotal}, PENDING ${dbPend} (khớp: ${hasDb}); đầu trang «${qtxt.replace(/\n/g, " ").slice(0, 120)}»`);
  verdict("TC-12-13", shown >= Math.min(3, dbTopics.length) && fake.length === 0 && hasDb, rows.join("; "));
  await sv.ctx.close(); await tch.ctx.close();
});
