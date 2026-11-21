import { test, type Page } from "@playwright/test";
import { go, FX, shot, log, psql, newUser, expect } from "./lib";
import { execFileSync } from "node:child_process";
const F = FX("fx_ui8") as { C: string; S: string[]; T: string; th: Record<string, string> };
const FA = FX("fx_att") as { C: string; teacher: string };
const R = (tc: string, ok: boolean, note: string) => log("r7-ui", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("r7-ui", `INFO ${tc} ${note}`);
const py = (code: string) => execFileSync("python3", ["-c", `import sys; sys.path.insert(0,'/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-7/scripts-r7')\nfrom lib3 import *\n${code}`], { encoding: "utf8" }).trim();
const cls = async (p: Page) => ((await p.locator("[data-part=topbar]").innerText()).match(/QC3[A-Z0-9]+/) ?? [""])[0];
const cls0 = (p: Page) => p.addInitScript(() => { (window as any).__cls = 0; try { new PerformanceObserver((l) => { for (const e of l.getEntries() as any[]) if (!e.hadRecentInput) (window as any).__cls += e.value; }).observe({ type: "layout-shift", buffered: true }); } catch { /* */ } });
const getCls = (p: Page) => p.evaluate(() => (window as any).__cls as number);

test("P4-03-38 inbox trạng thái", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, F.T); await cls0(page);
  // chậm: GET tickets trễ 3,5 s
  let slow = true; await page.route(/\/tickets\?/, async (r) => { if (slow) await new Promise((x) => setTimeout(x, 3500)); await r.continue().catch(() => undefined); });
  await page.goto(`/inbox?course=${F.C}`); await page.waitForTimeout(1200);
  const skel = await page.locator("[class*=skeleton],[data-skeleton],[aria-busy=true]").count(); const loadTxt = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 160);
  await shot(page, "p7c-inbox-loading"); slow = false; await page.waitForTimeout(4500);
  const okCls = await cls(page); const radios = await page.getByRole("radio").allInnerTexts();
  // chuyển trạng thái: đo CLS quanh các lần đổi
  const c0 = await getCls(page); const seen: Record<string, string> = {};
  for (const name of radios) { await page.getByRole("radio", { name }).click(); await page.waitForTimeout(1500); seen[name] = (await page.locator("main").innerText().catch(() => "")).replace(/\n+/g, " | ").slice(0, 200); if (name === "Đã nhận") await shot(page, "r7-inbox-filter-empty"); }
  const c1 = await getCls(page);
  const rowsTxt = await page.locator("[data-part=inbox-list] [data-ticket-id]").allInnerTexts();
  // lỗi 500
  await page.unrouteAll({ behavior: "ignoreErrors" }); await page.route(/\/tickets\?/, (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi máy chủ" }) }));
  await page.goto(`/inbox?course=${F.C}`); await page.waitForTimeout(3500); const errTxt = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 300); const retry = await page.getByRole("button", { name: "Thử lại" }).count(); await shot(page, "p7c-inbox-error");
  await page.unrouteAll({ behavior: "ignoreErrors" });
  // rỗng: lớp không có ticket
  const e = await newUser(browser, FA.teacher); await go(e.page, `/inbox?course=${FA.C}`); await e.page.waitForTimeout(2500); const emptyTxt = (await e.page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 300); const seeAll = await e.page.getByRole("button", { name: "Xem tất cả" }).count() + await e.page.getByRole("link", { name: "Xem tất cả" }).count(); const emptyCls = await cls(e.page); await shot(e.page, "p7c-inbox-empty");
  const filtEmpty = /Không có câu hỏi trong bộ lọc này/.test(seen["Đã nhận"] ?? "") && !/Xem tất cả/.test(seen["Đã nhận"] ?? "");
  const okAll = filtEmpty && /Chưa có câu hỏi cần xử lý/.test(emptyTxt) && !/Chưa có câu hỏi cần xử lý\./.test(emptyTxt) && seeAll === 0;
  R("TC-P4-03-38", okCls === "QC3T1" && skel >= 1 && okAll && /Thử lại/.test(errTxt) && retry >= 1 && c1 - c0 <= 0.05 && rowsTxt.length > 0 && rowsTxt.every((t) => /\S/.test(t)), `[P-11-81] GV ${F.T}, lớp ${okCls}: bộ lọc rỗng «Đã nhận» (lớp còn ticket khác) = «${seen["Đã nhận"]}» (đúng chữ+không nút=${filtEmpty}); GET tickets trễ 3,5 s → khung xương ${skel} phần tử sau 1,2 s (main: «${loadTxt}»); bộ lọc ${JSON.stringify(radios)}: nội dung lần lượt ${JSON.stringify(seen)}; CLS cộng dồn khi đổi OPEN→ANSWERED→ALL: ${(c1 - c0).toFixed(4)} (≤ 0,05); GET tickets 500: «${errTxt.slice(0, 200)}», nút «Thử lại»=${retry}; lớp ${emptyCls} không có ticket: «${emptyTxt.slice(0, 200)}», nút/liên kết «Xem tất cả»=${seeAll}; dòng ticket có chữ: ${JSON.stringify(rowsTxt.slice(0, 3).map((t) => t.replace(/\n+/g, " ").slice(0, 80)))}`);
  await e.ctx.close(); await ctx.close();
});

test("P4-06-47 chữ Loại khỏi tri thức / Không hiển thị", async ({ browser }) => {
  const th = py(`s,d=A('${F.S[0]}','POST','/courses/${F.C}/threads',{'title':'QC3T1 bài có câu trả lời AI','body':'Nội dung câu hỏi để thử loại câu trả lời AI.','tags':[]}); print(d['thread']['id'])`);
  psql(`insert into forum_posts(course_id,thread_id,author_id,kind,verification_state,citations,confidence,body,ai_body) values('${F.C}','${th}',null,'AI','PENDING','[]',0.5,'Câu trả lời AI QC3T1','Câu trả lời AI QC3T1')`);
  const t = await newUser(browser, F.T); await go(t.page, `/threads/${th}?course=${F.C}`); await t.page.waitForTimeout(1500); const c = await cls(t.page);
  const btn = await t.page.getByRole("button", { name: "Loại khỏi tri thức" }).count(); const mit = await (async () => { const m = t.page.getByRole("button", { name: "Thêm hành động" }); if (!(await m.count())) return [] as string[]; await m.first().click(); const r = await t.page.getByRole("menuitem").allInnerTexts(); await t.page.keyboard.press("Escape"); return r; })();
  const tTxt = (await t.page.locator("main").innerText()).replace(/\n+/g, " | "); await shot(t.page, "p7c-ai-teacher");
  if (btn) await t.page.getByRole("button", { name: "Loại khỏi tri thức" }).first().click(); else if (mit.some((x) => /Loại khỏi tri thức/.test(x))) { await t.page.getByRole("button", { name: "Thêm hành động" }).first().click(); await t.page.getByRole("menuitem", { name: /Loại khỏi tri thức/ }).click(); }
  await t.page.waitForTimeout(1500); const dlg = await t.page.getByRole("dialog").innerText().catch(() => ""); if (dlg) await t.page.getByRole("dialog").getByRole("button").last().click().catch(() => undefined); await t.page.waitForTimeout(2000);
  const st = psql(`select verification_state from forum_posts where thread_id='${th}' and kind='AI'`);
  const s = await newUser(browser, F.S[1]); await go(s.page, `/threads/${th}?course=${F.C}`); await s.page.waitForTimeout(1500); const sTxt = (await s.page.locator("main").innerText()).replace(/\n+/g, " | "); await shot(s.page, "p7c-ai-student-rejected");
  R("TC-P4-06-47(chữ)", c === "QC3T1" && (btn >= 1 || mit.some((x) => /Loại khỏi tri thức/.test(x))) && !/Câu trả lời AI QC3T1/.test(sTxt), `GV ${F.T} lớp ${c}: bài AI «Chờ xác nhận»: nút «Loại khỏi tri thức»=${btn}, menu … ${JSON.stringify(mit)}; hộp xác nhận «${dlg.replace(/\n+/g, " | ")}»; sau khi loại DB verification_state=${st}; SV ${F.S[1]} xem bài: «${sTxt.slice(0, 260)}» — SV không thấy bài AI đã loại («Không hiển thị» theo DESIGN §13)=${!/Câu trả lời AI QC3T1/.test(sTxt)} (DESIGN §13: Reject → SV «Không hiển thị», GV «Loại khỏi tri thức»)`);
  await s.ctx.close(); await t.ctx.close();
});


const menuBtn = (p: Page) => p.locator('button[aria-label="Tuỳ chọn bài"]').first();
const items = async (p: Page) => { await menuBtn(p).click(); await p.waitForTimeout(500); const t = await p.getByRole("menuitem").allInnerTexts(); return t; };

for (const [vp, tag] of [[{ width: 1280, height: 800 }, "1280"], [{ width: 375, height: 812 }, "375"]] as const) test(`P4-06-47 menu bài ${tag}`, async ({ browser }) => {
  const rep = tag === "1280" ? F.S[1] : F.S[2]; const th = tag === "1280" ? F.th.rep : F.th.rep2;
  const s = await newUser(browser, rep, vp); await go(s.page, `/threads/${th}?course=${F.C}`); await s.page.waitForTimeout(1200);
  const c = await cls(s.page); const it = await items(s.page);
  const trg = await menuBtn(s.page).boundingBox(); await shot(s.page, `p7c-menu-student-${tag}`);
  const mi = await s.page.getByRole("menuitem").first().boundingBox();
  await s.page.getByRole("menuitem", { name: /Báo cáo/ }).click(); await s.page.waitForTimeout(500);
  const sel = s.page.getByLabel("Lý do báo cáo"); const opts = await sel.locator("option").allInnerTexts(); const tag0 = await sel.evaluate((e) => e.tagName);
  await s.page.getByRole("button", { name: "Gửi báo cáo" }).click(); const t0 = Date.now(); await s.page.getByText(/Đã báo cáo/).first().waitFor({ timeout: 8000 }); const undo = await s.page.getByRole("button", { name: "Hoàn tác" }).count();
  const line = (await s.page.getByText(/Đã báo cáo/).first().innerText()).replace(/\n+/g, " "); await shot(s.page, `p7c-reported-${tag}`);
  await s.page.getByText(/Đã báo cáo/).first().waitFor({ state: "detached", timeout: 9000 }).catch(() => undefined); const gone = Date.now() - t0;
  const ov = await s.page.evaluate(() => ({ sw: document.documentElement.scrollWidth, iw: window.innerWidth }));
  // chủ bài
  const own = th === F.th.rep ? F.S[4] : F.S[5]; const o = await newUser(browser, own, vp); await go(o.page, `/threads/${th}?course=${F.C}`); await o.page.waitForTimeout(1200); const ownIt = await items(o.page); await o.ctx.close();
  // GV: sau khi bị tự ẩn (3 báo cáo) có Giữ bài
  py(`import time\nQ='/courses/${F.C}'\nfor u in ('${F.S[6]}','${F.S[7]}','${F.S[8]}'):\n    A(u,'POST',f'{Q}/threads/${th}/report',{'reason':'SPAM'})\ntime.sleep(1)`);
  const t = await newUser(browser, F.T, vp); await go(t.page, `/threads/${th}?course=${F.C}`); await t.page.waitForTimeout(1200); const tIt = await items(t.page); const hid = (await t.page.locator("main").innerText()).match(/Đã ẩn[^\n]*/)?.[0] ?? ""; await shot(t.page, `p7c-menu-teacher-${tag}`);
  const ok = c === "QC3T1" && it.length === 1 && /Báo cáo/.test(it[0]) && tag0 === "SELECT" && opts.length >= 3 && undo >= 1 && /Đã báo cáo/.test(line) && gone >= 4000 && gone <= 6500 && ownIt.some((x) => /Sửa/.test(x)) && ownIt.some((x) => /Xoá/.test(x)) && tIt.some((x) => /Ẩn bài|Bỏ ẩn/.test(x)) && tIt.some((x) => /Ghim/.test(x)) && tIt.some((x) => /Giữ bài/.test(x)) && ov.sw <= ov.iw && (tag === "1280" || ((trg?.height ?? 0) >= 44 && (mi?.height ?? 0) >= 44));
  R(`TC-P4-06-47(${tag})`, ok, `[${tag}px] lớp ${c}; SV ${rep}: menu … = ${JSON.stringify(it)} (Báo cáo là mục menu, nút … ${trg?.width.toFixed(0)}×${trg?.height.toFixed(0)}, mục đầu cao ${mi?.height.toFixed(0)}); chọn lý do bằng <${tag0}> ${opts.length} lựa chọn ${JSON.stringify(opts)}; sau gửi: «${line}» + «Hoàn tác»=${undo}, biến mất sau ${gone} ms; scrollWidth ${ov.sw} ≤ ${ov.iw}; chủ bài ${own}: ${JSON.stringify(ownIt)}; GV ${F.T} sau 3 báo cáo (bài tự ẩn «${hid}»): ${JSON.stringify(tIt)}`);
  await t.ctx.close(); await s.ctx.close();
});

