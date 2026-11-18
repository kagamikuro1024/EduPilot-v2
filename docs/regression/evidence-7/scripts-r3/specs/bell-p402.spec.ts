import { test } from "@playwright/test";
import { login, go, FX, shot, log, psql, newUser } from "./lib";
import { execFileSync } from "node:child_process";
const R = (tc: string, ok: boolean, note: string) => log("p402", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const bell = (p: any) => p.locator('button[aria-label^="Thông báo"]');
const dot = async (p: any) => { const t = await p.locator("[data-part=bell-dot]").first().innerText().catch(() => "0"); return t.trim(); };
const api = (u: string, args: string) => execFileSync("python3", ["-c", `import sys; sys.path.insert(0,'/tmp/qc3e')\nfrom lib3 import *\n${args}`], { encoding: "utf8" }).trim();

test("06+12", async ({ browser }) => {
  const f = FX("fx_ui3");
  const t = await newUser(browser, "qc3t2"); let loads = 0; t.page.on("load", () => loads++);
  await go(t.page, "/"); await t.page.waitForTimeout(1500);
  const base = await dot(t.page);
  const s = await newUser(browser, f.b06.u); await go(s.page, `/chat?session=${f.b06.sid}`);
  const t0 = Date.now(); await s.page.locator("[data-part=ask-staff]").first().click();
  let seen = -1; for (let i = 0; i < 80; i++) { const d = await dot(t.page); if (d !== base && d !== "0") { seen = Date.now() - t0; break; } await t.page.waitForTimeout(100); }
  R("TC-P4-02-06", seen >= 0 && seen <= 5000 && loads === 1, `SV bấm «Nhờ giảng viên hỗ trợ»: chuông GV đổi «${base}» → «${await dot(t.page)}» sau ${seen} ms (≤5000), không reload (sự kiện load của trang GV: ${loads - 1} sau lần mở đầu)`);
  await s.ctx.close();
  // 12: offline 10 s, 3 ticket lúc offline
  await t.ctx.setOffline(true); await t.page.waitForTimeout(500);
  api("", `D=C7['D']; fx=json.load(open('/tmp/qc3e/fx_ui3.json'))\nfor am,u in zip(fx['ids'],('qc3s14','qc3s15','qc3s16')): esc(u,am)`);
  await t.page.waitForTimeout(10000); await t.ctx.setOffline(false); await t.page.waitForTimeout(35000);
  await bell(t.page).click(); await t.page.waitForTimeout(800);
  const items = await t.page.locator("[data-part=notifications] li").allInnerTexts();
  const nTicket = items.filter((x) => /câu hỏi đang chờ/.test(x));
  const db = psql(`select count(*)||'/'||max(title) from notifications where user_id=(select id from users where email='qc3t2@edupilot.local') and type='TICKET_NEW' and read_at is null`);
  await shot(t.page, "p402-12");
  R("TC-P4-02-12", nTicket.length === 1 && /4 câu hỏi/.test(nTicket[0]) , `offline 10 s, 3 ticket sinh lúc offline; sau nối lại ≤35 s: chuông có ${nTicket.length} dòng «câu hỏi đang chờ»: «${nTicket.map((x) => x.replace(/\n/g, " ")).join(" ; ")}» (không nhân đôi; mong 1 dòng gộp «4 câu hỏi…» gồm ticket 06); DB chưa đọc: ${db}`);
  await t.ctx.close();
});
test("23", async ({ browser }) => {
  const u = "qc3t2";
  const a = await newUser(browser, u); const b = await newUser(browser, u); const c = await newUser(browser, u);
  await go(a.page, "/"); await go(b.page, "/"); await go(c.page, "/"); await c.page.waitForTimeout(1500);
  const errs: string[] = []; c.page.on("pageerror", (e) => errs.push(String(e)));
  await bell(c.page).click(); await c.page.waitForTimeout(800);
  const rd = async () => ((await c.page.locator("[data-part=notifications] li").allInnerTexts()).find((x) => /câu hỏi đang chờ/.test(x)) ?? "").split("\n")[0];
  const before = await rd();
  const t0 = Date.now(); api("", `fx=json.load(open('/tmp/qc3e/fx_ui3.json')); print(esc('qc3s18',fx['ids23'][1])[0])`);
  let seen = -1; for (let i = 0; i < 400; i++) { if ((await rd()) !== before) { seen = Date.now() - t0; break; } await c.page.waitForTimeout(100); }
  const lbl = await bell(c.page).getAttribute("aria-label");
  R("TC-P4-02-23", seen >= 0 && seen <= 31000 && errs.length === 0, `3 tab cùng người dùng (SSE tối đa 2/người; tab 3 rơi về làm mới 30 s): dòng gộp ở tab 3 «${before}» → «${await rd()}» sau ${(seen / 1000).toFixed(1)} s (≤30 s), không lỗi trang (${errs.length}), nhãn chuông «${lbl}»`);
  await a.ctx.close(); await b.ctx.close(); await c.ctx.close();
});
test("25-27", async ({ browser }) => {
  const u = "qc3s29"; const s = await newUser(browser, u); const p = s.page; const toasts: string[] = [];
  await go(p, "/"); await p.waitForTimeout(1500);
  const d1 = await dot(p); await bell(p).click(); await p.waitForTimeout(800);
  const items = await p.locator("[data-part=notifications] li").all();
  const titles = await Promise.all(items.map((i) => i.innerText()));
  await shot(p, "p402-25");
  const order = titles.map((t) => (t.match(/QCN-(\d\d)/) ?? [])[1]);
  const newestFirst = order[0] === "12" || order.slice(0, 3).join() > order.slice(-3).join();
  // mục link rỗng -> '/'
  const li11 = p.locator('[data-note-id]', { hasText: "QCN-11" }); const href11 = await li11.locator("a").getAttribute("href");
  const li12 = p.locator('[data-note-id]', { hasText: "QCN-12" }); const href12 = await li12.locator("a").getAttribute("href");
  const reqs: string[] = []; p.on("request", (r) => { if (/notifications/.test(r.url()) && r.method() !== "GET") reqs.push(r.method() + " " + r.url().split("/api/v1")[1]); });
  await li12.locator("a").click(); await p.waitForTimeout(1500);
  const url = p.url(); const toastAll = await p.locator("[role=status],[data-part=toast],[role=alert],[class*=oast]").evaluateAll((els) => els.map((e) => (e as HTMLElement).tagName + "." + (e.getAttribute("data-part") ?? e.className).toString().slice(0, 40) + ":" + (e as HTMLElement).innerText.slice(0, 60))); const toast = toastAll.filter((x) => /oast/i.test(x));
  const d2 = await dot(p);
  R("TC-P4-02-25", d1 === "9+" && href11 === "/" && /\/calendar/.test(href12 ?? "") && !/course=/.test(url.split("?")[1] ?? "") && reqs.length >= 1 && toast.length === 0, `1280×800, 12 chưa đọc: chấm «${d1}» (mong 9+); thứ tự (QCN số): ${order.join(",")}; link rỗng → href «${href11}»; mục có ?course= dẫn tới «${href12}»; sau bấm URL «${url.replace("https://localhost:1143", "")}» (?course= được tiêu thụ); đánh dấu đã đọc lạc quan: ${reqs.join(" | ")}, chấm sau «${d2}»; toast: ${toast.length}; (phần tử role=status/alert khác: ${JSON.stringify(toastAll)})`);
  await s.ctx.close();
  // 26: 375px
  const m = await newUser(browser, u, { width: 375, height: 812 }); const q = m.page; await go(q, "/"); await q.waitForTimeout(1500);
  const bb = await bell(q).boundingBox(); await bell(q).click(); await q.waitForTimeout(800);
  const rects = await q.evaluate(() => Array.from(document.querySelectorAll("[data-part=notifications] li a")).map((a) => { const r = (a as HTMLElement).getBoundingClientRect(); return [Math.round(r.width), Math.round(r.height)]; }));
  const sw = await q.evaluate(() => [document.documentElement.scrollWidth, window.innerWidth]);
  const live = await q.locator('[data-part=bell-live]').getAttribute("aria-live");
  await shot(q, "p402-26");
  R("TC-P4-02-26", (bb?.width ?? 0) >= 44 && (bb?.height ?? 0) >= 44 && rects.every((r) => r[1] >= 44) && sw[0] <= sw[1] && live === "polite", `375×812: nút chuông ${bb?.width}×${bb?.height}; ${rects.length} mục, chiều cao nhỏ nhất ${Math.min(...rects.map((r) => r[1]))} px (≥44); scrollWidth ${sw[0]} ≤ innerWidth ${sw[1]}; aria-live="${live}" ở vùng số chưa đọc`);
  await m.ctx.close();
  // 27: mở danh sách, đang đọc mục thứ 3, đẩy thông báo mới
  const n = await newUser(browser, u); const w = n.page; await go(w, "/"); await w.waitForTimeout(1500);
  await bell(w).click(); await w.waitForTimeout(800);
  const third = w.locator("[data-part=notifications] li").nth(2); const top0 = await third.evaluate((e: HTMLElement) => e.getBoundingClientRect().top);
  const id3 = await third.getAttribute("data-note-id");
  api("", `A_=C7['A']; sid,um,am=mkchat('qc3s29',A_); s,d=esc('qc3s29',am); t=d['ticket']['id']; A('qc3t','POST',f'/courses/{A_}/tickets/{t}/claim'); print(A('qc3t','POST',f'/courses/{A_}/tickets/{t}/answer',{'body':'Chào em, trả lời để đẩy thông báo mới.'})[0])`);
  await w.waitForTimeout(6000);
  const first = await w.locator("[data-part=notifications] li").first().innerText();
  const top1 = await w.locator(`[data-note-id="${id3}"]`).evaluate((e: HTMLElement) => e.getBoundingClientRect().top);
  const cnt = await w.locator("[data-part=notifications] li").count();
  R("TC-P4-02-27", /trả lời|Giảng viên/i.test(first.split("\n")[0]) && Math.abs(top1 - top0) <= 2, `mở danh sách (mục 3 «${id3?.slice(0, 8)}» top=${top0.toFixed(1)}), đẩy thông báo mới (GV trả lời): mục đầu «${first.replace(/\n/g, " ").slice(0, 60)}», ${cnt} mục; mục đang đọc top sau=${top1.toFixed(1)} (lệch ${(top1 - top0).toFixed(1)} px, ≤2)`);
  await n.ctx.close();
});
