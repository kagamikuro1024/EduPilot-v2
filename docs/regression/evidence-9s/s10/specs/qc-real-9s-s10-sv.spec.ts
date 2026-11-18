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
  await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); }); // hộp thoại thời gian học lần đầu (P-11-5)
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

test.use({ actionTimeout: 30_000 });
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

async function step(page: Page, name: string, fn: () => Promise<unknown>) { try { await fn(); } catch (e) { await page.screenshot({ path: `${OUT}/s10-fail-${name}.png` }); throw new Error(`${name}: ${String(e).slice(0, 160)}`); } }
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
  const notice = offTxt.replace(/\n/g, " ").match(/Không kết nối[^.]*\.[^.]*\./)?.[0] ?? "";
  await ctx.setOffline(false);
  await page.waitForFunction(() => /Đúng rồi|Chưa đúng/.test(document.body.innerText), null, { timeout: 25_000 }).catch(() => {});
  const auto = /Đúng rồi|Chưa đúng/.test(await page.locator("main").innerText());
  const nAuto = dbCount(`select count(*) from practice_answers where attempt_id='${a.id}'`);
  if (!auto) { await page.getByRole("button", { name: "Kiểm tra" }).click(); await page.waitForTimeout(2500); }
  const fb = /Đúng rồi|Chưa đúng/.test(await page.locator("main").innerText());
  const n = dbCount(`select count(*) from practice_answers where attempt_id='${a.id}'`);
  verdict("TC-06-10", showedUnsent && !leaked && auto && n === 1, `offline: thông báo tại câu «${notice}» (TC kỳ vọng chữ «Chưa gửi được»: ${showedUnsent}); phản hồi lộ khi offline ${leaked}; khi có mạng tự gửi lại ${auto} (DB sau 25 s = ${nAuto}); bấm Kiểm tra thủ công → phản hồi ${fb}, practice_answers = ${n}`);
  await ctx.close();
});

test("TC-06-21 tải lại ở câu 6: giữ câu, giữ lựa chọn chưa gửi", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.nguyco");
  const a = await startVia(request, "sv.nguyco", id, { mode: "TOPIC", topic: T, count: 10 });
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1200);
  for (let i = 1; i <= 5; i++) { await pick(page); await page.getByRole("button", { name: "Kiểm tra" }).click(); await page.getByRole("button", { name: /Câu tiếp theo|Xem kết quả/ }).click(); await page.waitForTimeout(400); }
  await pick(page, 1); await page.waitForTimeout(700);
  const enabledBefore = await page.getByRole("button", { name: "Kiểm tra" }).isEnabled();
  await page.reload(); await page.waitForTimeout(2000);
  const after = await page.locator("main").innerText();
  const on6 = /Câu 6\s*\/\s*10/.test(after);
  const pressed = await page.getByRole("list", { name: "Các đáp án" }).getByRole("button").nth(1).getAttribute("aria-pressed").catch(() => null);
  const locked = dbCount(`select count(*) from practice_answers where attempt_id='${a.id}' and is_correct is not null`);
  await page.screenshot({ path: `${OUT}/s10-0621.png` });
  const enabledAfter = await page.getByRole("button", { name: "Kiểm tra" }).isEnabled();
  verdict("TC-06-21", enabledBefore && on6 && locked === 5 && enabledAfter, `trước tải lại đã chọn (Kiểm tra bật=${enabledBefore}); sau tải lại: «Câu 6/10» ${on6}; 5 câu trước đã chấm (DB ${locked}); lựa chọn chưa gửi còn (Kiểm tra bật=${enabledAfter}, aria-pressed=${pressed})`);
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
    if (i === 1) { await step(page, "c1-pick", () => page.getByRole("list", { name: "Các đáp án" }).getByRole("button").first().tap({ timeout: 8000 })); await step(page, "c1-check", () => page.getByRole("button", { name: "Kiểm tra" }).tap({ timeout: 8000 })); await page.waitForTimeout(600); texts.push(await page.locator("body").innerText()); }
    else { await step(page, `c${i}-pick`, () => page.getByRole("list", { name: "Các đáp án" }).getByRole("button").first().tap({ timeout: 8000 })); await step(page, `c${i}-check`, () => page.getByRole("button", { name: "Kiểm tra" }).tap({ timeout: 8000 })); }
    const bar = await page.getByRole("button", { name: /Câu tiếp theo|Xem kết quả/ }).evaluate((e) => { let n: HTMLElement | null = e as HTMLElement; while (n) { const p = getComputedStyle(n).position; if (p === "fixed" || p === "sticky") return p; n = n.parentElement; } return "static"; });
    if (bar !== "static") sticky = true;
    await step(page, `c${i}-next`, () => page.getByRole("button", { name: /Câu tiếp theo|Xem kết quả/ }).tap({ timeout: 8000 })); await page.waitForTimeout(500);
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

const CLOCK = /(?<![\d:])(\d{1,2}):(\d{2})(?![\d:])/;
async function mockStart(request: APIRequestContext, user: string, id: string, n = 20, minutes = 20) { const matrix = n <= 15 ? [{ topic: T, count: n }] : [{ topic: T, count: 12 }, { topic: "QC-r10-X", count: n - 12 }]; return startVia(request, user, id, { mode: "MOCK", count: n, duration_minutes: minutes, matrix }); } // toàn câu trắc nghiệm 1 đáp án: chọn bằng nút
async function getAttempt(request: APIRequestContext, user: string, id: string, aid: string) { const r = await call(request, await login(request, user), "GET", `/courses/${id}/practice/attempts/${aid}`); return r.json as { deadline_at: string; server_now: string; items: { id: string }[] }; }
async function mockPick(page: Page, i = 0) { await step(page, "mockpick", () => page.getByRole("list", { name: "Các đáp án" }).getByRole("button").nth(i).click({ timeout: 8000 })); }

test("TC-07-04 đồng hồ theo máy chủ dù máy khách lệch +5 phút", async ({ browser, request }) => {
  const res: string[] = []; let okAll = true;
  for (const w of [1440, 375]) {
    const { ctx, page, id } = await pageAs(browser, "sv.kha", w);
    const a = await mockStart(request, "sv.kha", id);
    await page.addInitScript(() => { const off = 5 * 60 * 1000; const RD = Date; class D extends RD { constructor(...a: unknown[]) { if (a.length === 0) super(RD.now() + off); else super(...(a as [number])); } static now() { return RD.now() + off; } } (window as unknown as { Date: typeof Date }).Date = D as unknown as typeof Date; });
    const tF = Date.now(); const at = await getAttempt(request, "sv.kha", id, a.id);
    await page.goto(`/practice/${a.id}`); await page.waitForTimeout(2500);
    const txt = await page.locator("main").innerText(); const m = txt.match(CLOCK); const tR = Date.now();
    const exp = (Date.parse(at.deadline_at) - Date.parse(at.server_now)) / 1000 - (tR - tF) / 1000;
    const shown = m ? Number(m[1]) * 60 + Number(m[2]) : NaN; const diff = Math.abs(shown - exp);
    const tab = await page.evaluate((re) => { const rx = new RegExp(re); for (const e of document.querySelectorAll("*")) { if (e.children.length === 0 && rx.test((e as HTMLElement).innerText ?? "")) { return getComputedStyle(e).fontVariantNumeric; } } return ""; }, CLOCK.source);
    res.push(`${w}px: hiển thị ${m?.[0]} (${shown}s) vs máy chủ ${exp.toFixed(1)}s, chênh ${diff.toFixed(1)}s; font-variant-numeric «${tab}»`);
    await page.screenshot({ path: `${OUT}/s10-0704-${w}.png` });
    if (!(diff <= 2) || !/tabular/.test(tab)) okAll = false;
    await ctx.close();
  }
  verdict("TC-07-04", okAll, `máy khách lệch +5 phút (Date ghi đè): ${res.join(" | ")}`);
});

test("TC-07-05 PUT answers kèm version + X-Practice-Tab; chữ «Đã lưu»", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.gioi");
  const a = await mockStart(request, "sv.gioi", id);
  const puts: { v: unknown; tab: string | undefined; idem: string | undefined }[] = [];
  page.on("request", (r) => { if (r.method() === "PUT" && /\/practice\/attempts\/.+\/answers\//.test(r.url())) puts.push({ v: (r.postDataJSON() as { version?: unknown })?.version, tab: r.headers()["x-practice-tab"], idem: r.headers()["idempotency-key"] }); });
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
  const t0 = Date.now(); const tps: number[] = [];
  page.on("request", (r) => { if (r.method() === "PUT" && /\/answers\//.test(r.url())) tps.push(Date.now() - t0); });
  await mockPick(page, 1); await page.waitForTimeout(1200);
  const st1 = await page.getByRole("status").first().innerText();
  await page.waitForTimeout(6000);
  const st = await page.getByRole("status").first().innerText();
  await page.getByRole("button", { name: "Câu sau" }).click(); await mockPick(page, 2); await page.waitForTimeout(7000);
  const st2 = await page.getByRole("status").first().innerText();
  verdict("TC-07-05", puts.length >= 2 && puts.every((p) => p.v !== undefined && !!p.tab) && /Đã lưu/.test(st), `PUT answers ${puts.length} lần (ms kể từ lúc chọn: ${JSON.stringify(tps)}): ${JSON.stringify(puts)}; trạng thái sau 1,2 s «${st1}», sau 7 s «${st}», sau câu 2 «${st2}»`);
  await ctx.close();
});

test("TC-07-09/10 offline 5 câu: đếm, IndexedDB, gửi lại đúng thứ tự", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.nguyco");
  const a = await mockStart(request, "sv.nguyco", id);
  const order: string[] = [];
  page.on("request", (r) => { if (r.method() === "PUT" && /\/answers\//.test(r.url())) order.push(r.url().split("/answers/")[1]); });
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
  await ctx.setOffline(true);
  const chosen: string[] = [];
  for (let i = 0; i < 5; i++) { await mockPick(page, 1); chosen.push(a.items[i].id); if (i < 4) await page.getByRole("button", { name: "Câu sau" }).click(); await page.waitForTimeout(300); }
  await page.waitForTimeout(1500);
  const txt = await page.locator("main").innerText(); const pend = txt.match(/Chưa gửi được (\d+) câu/)?.[1];
  const ls = await page.evaluate(() => Object.keys(localStorage).filter((k) => /practice|attempt|ep:/i.test(k)).map((k) => `${k}=${(localStorage.getItem(k) ?? "").length}B`));
  const idb = await page.evaluate(async () => { const dbs = await indexedDB.databases(); const out: Record<string, number> = {}; for (const d of dbs) { if (!d.name) continue; await new Promise<void>((res) => { const rq = indexedDB.open(d.name!); rq.onsuccess = () => { const db = rq.result; let n = 0; const names = [...db.objectStoreNames]; if (!names.length) { db.close(); res(); return; } const tx = db.transaction(names, "readonly"); names.forEach((s) => { const c = tx.objectStore(s).count(); c.onsuccess = () => { out[`${d.name}/${s}`] = c.result; }; }); tx.oncomplete = () => { db.close(); res(); }; }; rq.onerror = () => res(); }); } return out; });
  order.length = 0;
  await ctx.setOffline(false);
  await page.waitForFunction(() => !/Chưa gửi được \d+ câu/.test(document.body.innerText), null, { timeout: 40_000 }).catch(() => {});
  await page.waitForTimeout(2500);
  const gone = !/Chưa gửi được/.test(await page.locator("main").innerText());
  const nDb = dbCount(`select count(*) from practice_answers where attempt_id='${a.id}'`);
  const sent = order.filter((x, i) => order.indexOf(x) === i);
  const idbN = Object.values(idb).reduce((a, b) => a + b, 0);
  verdict("TC-07-09", pend === "5" && idbN >= 5, `offline, chọn 5 câu: thanh dưới «Chưa gửi được ${pend} câu» (kỳ vọng 5); IndexedDB mục = ${idbN} ${JSON.stringify(idb)}; localStorage ${JSON.stringify(ls)}`);
  verdict("TC-07-10", gone && nDb === 5 && JSON.stringify(sent.slice(0, 5)) === JSON.stringify(chosen), `online: thanh dưới biến mất ${gone}; practice_answers = ${nDb} (kỳ vọng 5); thứ tự gửi ${JSON.stringify(sent.map((x) => x.slice(-4)))} vs chọn ${JSON.stringify(chosen.map((x) => x.slice(-4)))}`);
  await ctx.close();
});

test("TC-07-12 cảnh báo «Còn 5 phút» một lần; tải lại không hiện lại", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.moi");
  const a = await mockStart(request, "sv.moi", id, 20, 20);
  sql(`update practice_attempts set deadline_at=now()+interval '5 minutes 8 seconds' where id='${a.id}'`);
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
  const before = (await page.getByRole("status").allInnerTexts()).join("|");
  await page.waitForFunction(() => /Còn 5 phút/.test(document.body.innerText), null, { timeout: 30_000 }).catch(() => {});
  const warned = /Còn 5 phút/.test(await page.locator("body").innerText());
  const modal = await page.getByRole("dialog").count();
  const red = await page.getByText("Còn 5 phút.").first().evaluate((e) => getComputedStyle(e).color).catch(() => "");
  await page.screenshot({ path: `${OUT}/s10-0712-warn.png` });
  await page.waitForTimeout(62_000);
  await page.reload(); await page.waitForTimeout(2500);
  const again = /Còn 5 phút/.test(await page.locator("body").innerText());
  verdict("TC-07-12", warned && modal === 0 && !again, `trước ngưỡng «${before}»; tới 5:00: cảnh báo ${warned} (màu ${red}), dialog ${modal}; tải lại khi còn ~4:00: hiện lại ${again}`);
  await ctx.close();
});

test("TC-07-15 + TC-07-21/22 nộp: xác nhận một lần; mất mạng khi nộp", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.kha");
  const a = await mockStart(request, "sv.kha", id, 5, 20);
  const subs: { idem?: string }[] = [];
  page.on("request", (r) => { if (r.method() === "POST" && /\/submit$/.test(r.url())) subs.push({ idem: r.headers()["idempotency-key"] }); });
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
  for (let i = 0; i < 5; i++) { await mockPick(page, 1); if (i < 4) await page.getByRole("button", { name: "Câu sau" }).click(); await page.waitForTimeout(400); }
  await page.waitForTimeout(1500);
  await page.getByRole("button", { name: "Nộp bài" }).first().click();
  const dlg = page.getByRole("dialog"); await dlg.waitFor();
  const dtxt = await dlg.innerText(); const n1 = await page.getByRole("dialog").count();
  verdict("TC-07-15", n1 === 1 && /không sửa được|không thể|không đảo ngược|không hoàn tác/i.test(dtxt) && subs.length === 0, `hộp xác nhận ${n1} lần, trước xác nhận POST submit = ${subs.length}; chữ: «${dtxt.replace(/\n/g, " ").slice(0, 160)}»`);
  // nhánh mất mạng
  await ctx.setOffline(true); await dlg.getByRole("button", { name: "Nộp bài" }).click(); await page.waitForTimeout(3000);
  const off = await page.locator("body").innerText(); const notice = /Chưa nộp được\. Đang thử lại/.test(off); await page.screenshot({ path: `${OUT}/s10-0721-offline.png` }); const bar = off.replace(/\n/g, " ").match(/(Chưa [^.]*\.[^.]*\.?)/g) ?? [];
  await ctx.setOffline(false);
  await page.waitForFunction(() => /Bạn đúng \d+\/\d+ câu/.test(document.body.innerText), null, { timeout: 60_000 }).catch(() => {});
  const done = /Bạn đúng \d+\/\d+ câu/.test(await page.locator("body").innerText());
  const idems = [...new Set(subs.map((s) => s.idem))];
  const nSub = dbCount(`select count(*) from practice_attempts where id='${a.id}' and status='SUBMITTED'`);
  verdict("TC-07-21", notice && done && idems.length === 1 && subs.length >= 2 && nSub === 1, `offline: «Chưa nộp được. Đang thử lại.» ${notice} (chữ thật: ${JSON.stringify(bar).slice(0, 160)}); có mạng: kết quả ${done}; POST submit ${subs.length} lần với ${idems.length} Idempotency-Key; lượt SUBMITTED ${nSub}`);
  await ctx.close();
});

test("TC-07-22 đóng tab khi chưa nộp được: bài không mất, gửi lại được", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.gioi");
  const a = await mockStart(request, "sv.gioi", id, 5, 20);
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
  await ctx.setOffline(true);
  for (let i = 0; i < 3; i++) { await mockPick(page, 1); if (i < 2) await page.getByRole("button", { name: "Câu sau" }).click(); await page.waitForTimeout(300); }
  await page.getByRole("button", { name: "Nộp bài" }).first().click(); await page.getByRole("dialog").getByRole("button", { name: "Nộp bài" }).click(); await page.waitForTimeout(1500);
  await page.close();
  const p2 = await ctx.newPage(); await p2.goto(`/practice/${a.id}`).catch(() => {}); await p2.waitForTimeout(1500);
  await ctx.setOffline(false); await p2.reload(); await p2.waitForTimeout(4000);
  const t2 = await p2.locator("body").innerText();
  const kept = dbCount(`select count(*) from practice_answers where attempt_id='${a.id}'`);
  verdict("TC-07-22", kept >= 3 || /Chưa gửi được|Đang|Bạn đúng/.test(t2), `đóng tab khi offline rồi mở lại + có mạng: DB practice_answers = ${kept} (đã chọn 3); chữ «${t2.replace(/\n/g, " ").slice(0, 140)}»`);
  await ctx.close();
});

test("TC-07-26 375 MOCK 20 câu: không tràn, nút ≥44, thanh dưới cố định, đồng hồ không nhảy", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.nguyco", 375);
  const a = await mockStart(request, "sv.nguyco", id, 20, 20);
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
  const over: string[] = [], sm: string[] = [], boxes: string[] = []; let fixed = false;
  for (let i = 1; i <= 20; i++) {
    const m = await page.evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]); if (m[0] > m[1]) over.push(`c${i}:${m}`);
    sm.push(...(await small(page)).filter((x) => !/^(Mở|Đóng|Thu gọn)/.test(x)).map((x) => `c${i}:${x}`));
    await page.getByRole("list", { name: "Các đáp án" }).getByRole("button").first().tap(); await page.waitForTimeout(200);
    const bx = await page.evaluate((re) => { const rx = new RegExp(re); for (const e of document.querySelectorAll("*")) { if (e.children.length === 0 && rx.test((e as HTMLElement).innerText ?? "")) { const r = e.getBoundingClientRect(); return `${Math.round(r.x)},${Math.round(r.y)},${Math.round(r.height)}`; } } return ""; }, CLOCK.source); boxes.push(bx);
    const pos = await page.getByRole("button", { name: /Câu sau|Nộp bài/ }).first().evaluate((e) => { let n: HTMLElement | null = e as HTMLElement; while (n) { const p = getComputedStyle(n).position; if (p === "fixed" || p === "sticky") return p; n = n.parentElement; } return "static"; }); if (pos !== "static") fixed = true;
    if (i < 20) await page.getByRole("button", { name: "Câu sau" }).tap();
    await page.waitForTimeout(150);
  }
  const jump = new Set(boxes.map((b) => b.split(",").slice(1).join(","))).size;
  verdict("TC-07-26", over.length === 0 && sm.length === 0 && fixed && jump <= 1, `375 MOCK 20 câu: tràn ngang ${JSON.stringify(over)}; điều khiển <44 px ${JSON.stringify(sm.slice(0, 6))}; thanh dưới sticky/fixed ${fixed}; vị trí/chiều cao đồng hồ khác nhau ${jump} giá trị`);
  await ctx.close();
});

test("DBG mock screen", async ({ browser, request }) => {
  const { ctx, page, id } = await pageAs(browser, "sv.gioi");
  const a = await mockStart(request, "sv.gioi", id, 5, 20);
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(2000);
  await page.screenshot({ path: `${OUT}/s10-dbg-mock.png` });
  console.log("OPT", await page.getByText("A đúng").first().evaluate((e) => { let n: HTMLElement = e as HTMLElement; for (let i = 0; i < 3; i++) n = n.parentElement as HTMLElement; return n.outerHTML.slice(0, 700); }));
  await ctx.close();
});

test("TC-09-02 + TC-09-05 + TC-09-19 + TC-09-20 lịch sử luyện tập", async ({ browser, request }) => {
  // 09-19: SV chưa có lượt nào (sv37)
  const { ctx, page, id } = await pageAs(browser, "sv37");
  await page.goto("/practice/history"); await page.waitForTimeout(2000);
  const t19 = await page.locator("main").innerText(); const btn19 = await page.getByRole("button", { name: "Luyện ngay" }).or(page.getByRole("link", { name: "Luyện ngay" })).count();
  await page.screenshot({ path: `${OUT}/s10-0919.png` });
  verdict("TC-09-19", /Chưa có lượt luyện/.test(t19) && btn19 > 0, `SV chưa luyện lượt nào: «${t19.replace(/\n/g, " ").slice(0, 120)}»; nút «Luyện ngay» ×${btn19}`);
  // 09-05: < 10 câu đã làm → khối «Chủ đề cần ôn» ẩn (không rỗng chữ)
  const a = await startVia(request, "sv37", id, { mode: "TOPIC", topic: T, count: 5 });
  const t = await login(request, "sv37");
  for (const it of a.items.slice(0, 3)) await call(request, t, "PUT", `/courses/${id}/practice/attempts/${a.id}/answers/${it.id}`, { answer: { option_ids: [(it as unknown as { question: { options: { id: string }[] } }).question.options[0].id] } });
  await call(request, t, "POST", `/courses/${id}/practice/attempts/${a.id}/submit`, {});
  await page.goto("/practice/history"); await page.waitForTimeout(2000);
  const t05 = await page.locator("main").innerText(); await page.screenshot({ path: `${OUT}/s10-0905.png` });
  verdict("TC-09-05", !/Chủ đề cần ôn/.test(t05), `3 câu đã làm (<10): «Chủ đề cần ôn» ${/Chủ đề cần ôn/.test(t05) ? "HIỆN" : "ẩn"}; chữ trang: «${t05.replace(/\n/g, " ").slice(0, 140)}»`);
  // 09-02: bấm một hàng → xem lại từng câu
  await page.getByRole("row").nth(1).click().catch(async () => { await page.locator("main a, main [role=button]").nth(1).click(); });
  await page.waitForTimeout(2000); const t02 = await page.locator("main").innerText(); await page.screenshot({ path: `${OUT}/s10-0902.png` });
  verdict("TC-09-02", /Đáp án|đáp án|Đúng|Sai|Chưa đúng/.test(t02) && /Giải thích|giải thích|Nguồn/.test(t02) || /Đáp án đúng/.test(t02), `bấm một hàng lịch sử → ${new URL(page.url()).pathname}: «${t02.replace(/\n/g, " ").slice(0, 200)}»`);
  // 09-20: route.abort history → PageState lỗi + Thử lại
  const H = (r: import("@playwright/test").Route) => r.abort();
  await page.route("**/api/v1/courses/*/practice/history*", H); await page.goto("/practice/history"); await page.waitForTimeout(9000);
  const te = await page.locator("main").innerText(); const nr = await page.getByRole("button", { name: "Thử lại" }).count();
  await page.unroute("**/api/v1/courses/*/practice/history*", H); if (nr) await page.getByRole("button", { name: "Thử lại" }).first().click(); await page.waitForTimeout(2500);
  const ok = !/Không kết nối|Thử lại/.test(await page.locator("main").innerText()); await page.screenshot({ path: `${OUT}/s10-0920.png` });
  verdict("TC-09-20", nr > 0 && ok, `abort API history: «${te.replace(/\n/g, " ").slice(0, 120)}» nút Thử lại ×${nr}; bấm → tải được ${ok}`);
  await ctx.close();
});

test("TC-12-11 luồng thi thử 20 câu: tải lại ở câu 8, tắt/bật mạng ở câu 12–14, nộp, kết quả so với đáp án tính tay", async ({ browser, request }) => {
  test.setTimeout(240_000);
  const { ctx, page, id } = await pageAs(browser, "sv.gioi", 1440);
  const a = await mockStart(request, "sv.gioi", id, 20, 30); const PACE = Number(process.env.QC_PACE ?? 1200); const reqs: string[] = [];
  page.on("response", (r) => { if (/practice\/attempts\/[^/]+\/(answers|submit)/.test(r.url())) reqs.push(`${r.request().method()} ${r.url().split("/").slice(-2).join("/").slice(-14)} ${r.status()}`); });
  page.on("requestfailed", (r) => { if (/practice\/attempts/.test(r.url())) reqs.push(`FAIL ${r.method()} ${r.failure()?.errorText}`); });
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
  // đáp án QC tính tay: câu i đúng khi i % 4 !== 0 → 15/20 (phương án đúng của kho QC có chữ «đúng» ở nhãn, sai có chữ «sai»)
  const want = (i: number) => i % 4 !== 0; let expected = 0; let idxAfter = 8; const log: string[] = [];
  for (let i = 1; i <= 20; i++) {
    if (i === 12 && !process.env.QC_NOOFF) { reqs.push("##offline"); await ctx.setOffline(true); }
    const list = page.getByRole("list", { name: "Các đáp án" }).getByRole("button");
    const texts = await list.allInnerTexts(); const k = texts.findIndex((t) => (want(i) ? /đúng/.test(t) : /sai/.test(t)));
    expect(k, `câu ${i}: phương án ${want(i) ? "đúng" : "sai"} trong ${JSON.stringify(texts)}`).toBeGreaterThanOrEqual(0);
    reqs.push(`>>click ${i}`); await list.nth(k).click({ timeout: 8000 }); if (want(i)) expected++;
    await page.waitForTimeout(PACE);
    if (i === 8 && !process.env.QC_NORELOAD) { await page.reload(); await page.waitForTimeout(2000); const cur = await page.locator("main").innerText(); const mm = cur.match(/(?:Câu\s*)?(\d+)\s*\/\s*20/); idxAfter = mm ? Number(mm[1]) : -1; const sel = await page.getByRole("list", { name: "Các đáp án" }).getByRole("button").nth(k).getAttribute("aria-pressed").catch(() => null); log.push(`reload@8: chỉ số câu sau tải lại = ${idxAfter} (kỳ vọng 8); lựa chọn câu 8 còn tô: ${sel}; đầu trang: «${cur.replace(/\n/g, " ").slice(0, 90)}»`); }
    if (i === 14) { reqs.push("##online"); await ctx.setOffline(false); await page.waitForTimeout(3000); log.push(`online@14: «${(await page.getByRole("status").first().innerText().catch(() => "")).slice(0, 60)}»`); }
    if (i < 20) await page.getByRole("button", { name: "Câu sau" }).click();
  }
  await page.getByRole("button", { name: "Nộp bài" }).first().click(); const dlg = page.getByRole("dialog"); await dlg.getByRole("button", { name: "Nộp bài" }).click(); await page.waitForTimeout(4000);
  const t = await page.locator("main").innerText(); const m = t.match(/đúng\s+(\d+)\s*\/\s*(\d+)/i); await page.screenshot({ path: `${OUT}/s10-1211-result.png` });
  const dbc = sql(`select correct_count from practice_attempts where id='${a.id}'`)[0]?.[0];
  console.log("REQS", PACE, reqs.length, JSON.stringify(reqs));
  verdict(PACE < 1000 || process.env.QC_NOOFF || process.env.QC_NORELOAD ? "DBG-12-11-var" : "TC-12-11", idxAfter === 8 && !!m && Number(m[1]) === expected && Number(m[2]) === 20 && String(dbc) === String(expected), `thi thử 20 câu (QC tính tay ${expected}/20): kết quả hiển thị «${m?.[0]}»; DB correct_count ${dbc}; ${log.join("; ")}; chữ kết quả: «${t.replace(/\n/g, " ").slice(0, 120)}»`);
  await ctx.close();
});

test("TC-06-20 explain lỗi (stub trả 500/429): chữ tại chỗ, phản hồi sẵn có không đổi, làm tiếp + nộp bình thường", async ({ browser, request }) => {
  test.setTimeout(150_000);
  const { ctx, page, id } = await pageAs(browser, "sv.kha", 1440);
  const a = await startVia(request, "sv.kha", id, { mode: "TOPIC", topic: T, count: 5 });
  const tk = await login(request, "sv.kha");
  await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
  const opts = page.getByRole("list", { name: "Các đáp án" }).getByRole("button");
  await opts.first().click(); await page.getByRole("button", { name: "Kiểm tra" }).click(); await page.waitForTimeout(2000);
  const before = (await page.locator("main").innerText()).replace(/Giải thích thêm|Thử lại|Chưa giải thích được\./g, "").trim();
  writeFileSync("/tmp/stub10_mode", "overloaded");
  await page.getByRole("button", { name: "Giải thích thêm" }).click(); await page.waitForTimeout(3500);
  const after = await page.locator("main").innerText(); await page.screenshot({ path: `${OUT}/s10-0620.png` });
  writeFileSync("/tmp/stub10_mode", "ok");
  const msg = /Chưa giải thích được\./.test(after); const retry = await page.getByRole("button", { name: "Thử lại" }).count();
  const kept = after.replace(/Giải thích thêm|Thử lại|Chưa giải thích được\./g, "").trim() === before;
  await page.getByRole("button", { name: /Câu tiếp theo|Câu sau/ }).first().click().catch(() => {}); await page.waitForTimeout(1200);
  const next = /2\s*\/\s*5/.test(await page.locator("main").innerText());
  const rs = await call(request, tk, "POST", `/courses/${id}/practice/attempts/${a.id}/submit`, {});
  verdict("TC-06-20", msg && retry > 0 && kept && next && [200, 201].includes(rs.status), `stub trả 429 (OVERLOADED) cho explain: «Chưa giải thích được.» tại chỗ ${msg} + nút «Thử lại» ×${retry}; phần đúng/sai + giải thích sẵn có không đổi ${kept}; làm tiếp sang câu 2/5 ${next}; submit ${rs.status}`);
  await ctx.close();
});

test("TC-03-05 UI: nguồn của câu sinh từ tài liệu ẩn chỉ là chữ, không bấm mở được", async ({ browser, request }) => {
  test.setTimeout(120_000);
  const id = sql(`select id from courses where class_code='QD10C'`)[0][0];
  const { ctx, page } = await pageAs(browser, "sv.kha", 1440, "QD10C");
  await startVia(request, "sv.kha", id, { mode: "TOPIC", topic: "Mẫu sinh câu", count: 5 }).then(async (a) => {
    await page.goto(`/practice/${a.id}`); await page.waitForTimeout(1500);
    await page.getByRole("list", { name: "Các đáp án" }).getByRole("button").first().click(); await page.getByRole("button", { name: "Kiểm tra" }).click(); await page.waitForTimeout(2500);
    const t = await page.locator("main").innerText(); await page.screenshot({ path: `${OUT}/s10-0305.png` });
    const links = await page.locator("main a[href*='/library']").count(); const txt = /Nguồn trong tài liệu/.test(t);
    verdict("TC-03-05", txt && links === 0, `câu sinh từ LECTURE ẩn (visible_to_students=false), sau khi trả lời: chữ «Nguồn trong tài liệu» ${txt}; liên kết tới /library ×${links}; chữ trang: «${t.replace(/\n/g, " ").slice(0, 200)}»`);
  });
  await ctx.close();
});

test("TC-08-11 nhận xét tự luận khi LLM lỗi (stub 500): «Chưa nhận xét được. Thử lại.», bài viết giữ nguyên", async ({ browser, request }) => {
  test.setTimeout(150_000);
  const id = sql(`select id from courses where class_code='QD10A'`)[0][0]; const U = "sv.moi";
  let a: { id: string; items: { id: string; question: { type: string; stem: string } }[] } | null = null; let iid = "";
  for (const [tp] of sql(`select topic from question_bank where course_id='${id}' and type='ESSAY' and review_status='APPROVED'`)) {
    for (let k = 0; k < 8 && !iid; k++) { const t = await startVia(request, U, id, { mode: "TOPIC", topic: tp, count: 20 }) as unknown as { id: string; items: { id: string; question: { type: string; stem: string } }[] }; const it = t.items.find((x) => x.question.type === "ESSAY"); if (it) { a = t; iid = it.id; } }
    if (iid) break;
  }
  expect(iid, "không có câu ESSAY").not.toBe("");
  const { ctx, page } = await pageAs(browser, U, 1440);
  const tk = await login(request, U); const TXT = "SQL injection là chèn lệnh SQL độc hại vào đầu vào. Cách chống: dùng prepared statement.";
  await call(request, tk, "PUT", `/courses/${id}/practice/attempts/${a!.id}/answers/${iid}`, { answer: { text: TXT } });
  await page.goto(`/practice/${a!.id}`); await page.waitForTimeout(1500);
  // tới câu ESSAY (có thể không phải câu 1)
  for (let n = 0; n < 25 && (await page.getByRole("button", { name: "Nhận xét" }).count()) === 0; n++) { const nx = page.getByRole("button", { name: /Câu tiếp theo|Câu sau|Kiểm tra/ }).first(); if (!(await nx.count())) break; await page.getByRole("list", { name: "Các đáp án" }).getByRole("button").first().click({ timeout: 1500 }).catch(() => {}); await nx.click().catch(() => {}); await page.waitForTimeout(500); }
  const has = await page.getByRole("button", { name: "Nhận xét" }).count(); const ta = await page.getByRole("textbox").first().inputValue().catch(() => "");
  writeFileSync("/tmp/stub10_mode", "error");
  if (has) await page.getByRole("button", { name: "Nhận xét" }).click(); await page.waitForTimeout(4000);
  const t = await page.locator("main").innerText(); await page.screenshot({ path: `${OUT}/s10-0811.png` }); writeFileSync("/tmp/stub10_mode", "ok");
  const r = await call(request, tk, "GET", `/courses/${id}/practice/attempts/${a!.id}`, undefined);
  const saved = JSON.stringify(r.json).includes("prepared statement"); const msg = /Chưa nhận xét được\./.test(t); const retry = await page.getByRole("button", { name: "Thử lại" }).count();
  verdict("TC-08-11", has > 0 && msg && retry > 0 && saved, `LLM lỗi: nút Nhận xét ×${has}; chữ «Chưa nhận xét được.» ${msg} + nút «Thử lại» ×${retry}; bài viết trong ô: «${ta.slice(0, 40)}», GET attempt vẫn có bài đã lưu: ${saved}; chữ trang: «${t.replace(/\n/g, " ").slice(-160)}»`);
  await ctx.close();
});
