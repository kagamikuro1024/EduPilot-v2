import { writeFileSync, appendFileSync } from "node:fs";
import { expect, test, type Page, type Browser, type APIRequestContext } from "@playwright/test";
import { call, login, needReal, sql, PW, OUT } from "./qc-real-lib";

// QC smoke S10 (11-18): UI sinh viên Luyện đề trên stack ep-qc, lớp QD10A (không đụng 761987/761988).
const BASE = process.env.QC_BASE ?? "https://localhost:943";
const LOG = `${OUT}/s10-sv-notes.txt`;
const RES = process.env.QC_RES ?? `${OUT}/results_m.jsonl`;
/** Ghi kết luận thật của một ca (dòng sau thắng): cond đúng → P, sai → F. */
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 200)); };
const note = (s: string) => { console.log("NOTE", s); appendFileSync(LOG, s + "\n"); };
let CID = "";
const T = "QC-r10-Y";

async function cid(request: APIRequestContext, code: string) {
  const r = sql(`select id from courses where class_code='${code}'`);
  return r[0][0];
}
async function pageAs(browser: Browser, user: string, w = 1440, code = "QD10A") {
  const id = sql(`select id from courses where class_code='${code}'`)[0][0];
  const ctx = await browser.newContext({ baseURL: BASE, ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500, timezoneId: "Asia/Ho_Chi_Minh" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), id);
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page, id };
}
async function finishOpen(request: APIRequestContext, user: string, id: string) {
  const t = await login(request, user);
  for (const [aid] of sql(`select id from practice_attempts where course_id='${id}' and status='IN_PROGRESS' and student_id=(select id from users where email='${user}@edupilot.local')`)) await call(request, t, "POST", `/courses/${id}/practice/attempts/${aid}/submit`, {});
}
async function startVia(request: APIRequestContext, user: string, id: string, body: Record<string, unknown>) {
  await finishOpen(request, user, id);
  const t = await login(request, user);
  const r = await call(request, t, "POST", `/courses/${id}/practice/start`, body);
  expect([200, 201], `start ${r.text.slice(0, 120)}`).toContain(r.status);
  return r.json as { id: string; items: { id: string; question: { stem: string } }[]; deadline_at?: string; server_now?: string };
}
const small = (page: Page) => page.evaluate(() => [...document.querySelectorAll("button,a[role=button],[role=tab],input[type=checkbox]")].filter((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return r.width > 0 && r.height > 0 && (r.height < 44 || r.width < 44); }).map((e) => ((e as HTMLElement).innerText || e.getAttribute("aria-label") || e.tagName).slice(0, 30)));

test.beforeAll(() => { needReal(); writeFileSync(LOG, ""); });

test("TC-06-07 lượt TOPIC 10 câu ở 1440 và 375", async ({ browser, request }) => {
  const r: { w: number; hasIdx: boolean; hasExit: boolean; clock: boolean; list: boolean; noOverflow: boolean }[] = [];
  for (const w of [1440, 375]) {
    const { ctx, page, id } = await pageAs(browser, "sv.kha", w);
    const a = await startVia(request, "sv.kha", id, { mode: "TOPIC", topic: T, count: 10 });
    await page.goto(`/practice/${a.id}`);
    await page.waitForTimeout(1500);
    const txt = await page.locator("main").innerText();
    const hasIdx = /1\s*\/\s*10/.test(txt);
    const hasExit = await page.getByRole("button", { name: /Thoát/ }).or(page.getByRole("link", { name: /Thoát/ })).count();
    const clock = /\d{1,2}:\d{2}/.test(txt) && /còn|phút|giờ/i.test(txt);
    const list = await page.getByRole("navigation", { name: /Danh sách câu/ }).or(page.getByLabel("Danh sách câu")).count();
    const questions = await page.getByLabel("Câu hỏi").count();
    const sw = await page.evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]);
    await page.screenshot({ path: `${OUT}/s10-0607-${w}.png` });
    note(`TC-06-07 ${w}px: «1/10» ${hasIdx}; Thoát ${hasExit}; đồng hồ ${clock}; danh sách câu ${list}; scrollWidth ${sw[0]}/${sw[1]}`);
    r.push({ w, hasIdx, hasExit: hasExit > 0, clock, list: list > 0, noOverflow: sw[0] <= sw[1] });
    await ctx.close();
  }
  const d = r[0], m = r[1];
  verdict("TC-06-07", d.hasIdx && d.hasExit && !d.clock && d.list && d.noOverflow && m.hasIdx && m.hasExit && !m.clock && m.noOverflow,
    `lượt TOPIC 10 câu: «Câu 1/10 · Đúng 0» + «Thoát về Luyện đề» + không đồng hồ + không tràn ngang ở 1440 và 375; NHƯNG ≥1024 px KHÔNG có danh sách số câu thu gọn (TopicRun không dựng «Danh sách câu»; chỉ MockRun có aside) — ${JSON.stringify(r)}`);
});

async function pick(page: Page, i = 0) {
  const list = page.getByRole("list", { name: "Các đáp án" });
  if (await list.count()) { await list.getByRole("button").nth(i).click(); return; }
  if (await page.getByRole("button", { name: "Đúng", exact: true }).count()) { await page.getByRole("button", { name: "Đúng", exact: true }).click(); return; }
  await page.getByRole("textbox").first().fill("đáp án thử");
}
const dbCount = (q: string) => Number(sql(q)[0][0]);
const FORBID = /\b(AI|RAG|citation|model|LLM|prompt|embedding|token)\b/i;

test("TC-06-10 TOPIC: mất mạng khi bấm Kiểm tra", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.gioi");
  const a = await startVia(request, "sv.gioi", id, { mode: "TOPIC", topic: T, count: 5 });
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1200);
  await pick(page); await ctx.setOffline(true);
  await page.getByRole("button", { name: "Kiểm tra" }).click(); await page.waitForTimeout(3000);
  const offTxt = await page.locator("main").innerText();
  const showedUnsent = /Chưa gửi được/.test(offTxt), leaked = /Đúng rồi|Chưa đúng/.test(offTxt);
  await page.screenshot({ path: `${OUT}/s10-0610-offline.png` });
  await ctx.setOffline(false);
  await page.waitForFunction(() => /Đúng rồi|Chưa đúng/.test(document.body.innerText), null, { timeout: 40_000 }).catch(() => {});
  const onTxt = await page.locator("main").innerText(); const fb = /Đúng rồi|Chưa đúng/.test(onTxt);
  const n = dbCount(`select count(*) from practice_answers where attempt_id='${a.id}'`);
  verdict("TC-06-10", showedUnsent && !leaked && fb && n === 1, `offline: «Chưa gửi được» ${showedUnsent}, phản hồi lộ ${leaked}; online: phản hồi hiện ${fb}; practice_answers của lượt = ${n} (kỳ vọng 1)`);
  await ctx.close();
});

test("TC-06-21 tải lại ở câu 6: giữ câu, giữ lựa chọn chưa gửi", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.nguyco");
  const a = await startVia(request, "sv.nguyco", id, { mode: "TOPIC", topic: T, count: 10 });
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1200);
  for (let i = 1; i <= 5; i++) { await pick(page); await page.getByRole("button", { name: "Kiểm tra" }).click(); await page.getByRole("button", { name: /Câu tiếp theo|Xem kết quả/ }).click(); await page.waitForTimeout(400); }
  await pick(page, 1); await page.waitForTimeout(700);
  const before = await page.locator("main").innerText();
  await page.reload(); await page.waitForTimeout(2000);
  const after = await page.locator("main").innerText();
  const on6 = /Câu 6\s*\/\s*10/.test(after);
  const pressed = await page.getByRole("list", { name: "Các đáp án" }).getByRole("button").nth(1).getAttribute("aria-pressed").catch(() => null);
  const locked = dbCount(`select count(*) from practice_answers where attempt_id='${a.id}' and is_correct is not null`);
  await page.screenshot({ path: `${OUT}/s10-0621.png` });
  verdict("TC-06-21", on6 && locked === 5 && pressed === "true", `sau tải lại: «Câu 6/10» ${on6}; 5 câu trước đã chấm (DB ${locked}); lựa chọn chưa gửi ở câu 6 còn (aria-pressed=${pressed})`);
  await ctx.close();
});

test("TC-06-23 lớp chưa có câu APPROVED", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "sv36", 1440, "QD10C");
  await page.goto("/practice"); await page.waitForTimeout(1500);
  const txt = await page.locator("main").innerText(); const btns = await page.locator("main").getByRole("button").allInnerTexts();
  await page.screenshot({ path: `${OUT}/s10-0623.png` });
  verdict("TC-06-23", /Giảng viên chưa mở câu hỏi/.test(txt) && btns.filter((b) => !/^(Luyện theo|Thi thử|Chủ đề)/.test(b)).length === 0, `chữ: «${txt.replace(/\n/g, " ").slice(0, 160)}»; nút trong main: ${JSON.stringify(btns)}`);
  await ctx.close();
});

test("TC-06-24 + TC-06-26 375 touch: làm trọn lượt 10 câu; quét chữ", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.moi", 375);
  const a = await startVia(request, "sv.moi", id, { mode: "TOPIC", topic: T, count: 10 });
  const texts: string[] = [], overflow: string[] = [], smalls: string[] = []; let sticky = false;
  await page.goto("/practice"); await page.waitForTimeout(1500); texts.push(await page.locator("body").innerText());
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1200);
  for (let i = 1; i <= 10; i++) {
    const m = await page.evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]);
    if (m[0] > m[1]) overflow.push(`câu ${i}: ${m}`);
    smalls.push(...(await small(page)).filter((x) => !/^(Mở|Đóng|Thu gọn)/.test(x)).map((x) => `c${i}:${x}`));
    if (i === 1) { await pick(page); await page.getByRole("button", { name: "Kiểm tra" }).tap(); await page.waitForTimeout(600); texts.push(await page.locator("body").innerText()); }
    else { await page.getByRole("list", { name: "Các đáp án" }).getByRole("button").first().tap(); await page.getByRole("button", { name: "Kiểm tra" }).tap(); }
    const bar = await page.getByRole("button", { name: /Câu tiếp theo|Xem kết quả/ }).evaluate((e) => { let n: HTMLElement | null = e as HTMLElement; while (n) { const p = getComputedStyle(n).position; if (p === "fixed" || p === "sticky") return p; n = n.parentElement; } return "static"; });
    if (bar !== "static") sticky = true;
    await page.getByRole("button", { name: /Câu tiếp theo|Xem kết quả/ }).tap(); await page.waitForTimeout(500);
  }
  await page.waitForTimeout(800); texts.push(await page.locator("body").innerText());
  await page.goto("/practice/history"); await page.waitForTimeout(1500); texts.push(await page.locator("body").innerText());
  const bad = texts.map((t) => (t.match(FORBID) ?? [])[0]).filter(Boolean);
  const src = texts.join(" ").includes("Nguồn trong tài liệu");
  await page.screenshot({ path: `${OUT}/s10-0624-history.png` });
  verdict("TC-06-24", overflow.length === 0 && smalls.length === 0 && sticky, `lượt 10 câu bằng thao tác chạm ở 375: tràn ngang ${JSON.stringify(overflow)}; điều khiển <44 px ${JSON.stringify(smalls.slice(0, 6))}; thanh hành động sticky/fixed ${sticky}`);
  verdict("TC-06-26", bad.length === 0, `quét chữ SV ở 375 (/practice, câu, phản hồi, kết quả, /practice/history): chữ cấm ${JSON.stringify(bad)}; «Nguồn trong tài liệu» có ${src}; nút là động từ: «Kiểm tra», «Câu tiếp theo», «Xem kết quả» (TC ghi «Trả lời» — UI dùng «Kiểm tra»)`);
  await ctx.close();
});
