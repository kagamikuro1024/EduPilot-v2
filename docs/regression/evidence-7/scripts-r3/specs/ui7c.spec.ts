import { test, type Page } from "@playwright/test";
import { go, FX, shot, log, psql, newUser, expect } from "./lib";
import { execFileSync } from "node:child_process";
const F = FX("fx_ui8") as { C: string; S: string[]; T: string; th: Record<string, string> };
const FA = FX("fx_att") as { C: string; teacher: string };
const R = (tc: string, ok: boolean, note: string) => log("p7c", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("p7c", `INFO ${tc} ${note}`);
const py = (code: string) => execFileSync("python3", ["-c", `import sys; sys.path.insert(0,'/tmp/qc3e')\nfrom lib3 import *\n${code}`], { encoding: "utf8" }).trim();
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
  for (const name of radios) { await page.getByRole("radio", { name }).click(); await page.waitForTimeout(1500); seen[name] = (await page.locator("[data-part=inbox-list]").innerText().catch(() => "")).replace(/\n+/g, " | ").slice(0, 120); }
  const c1 = await getCls(page);
  const rowsTxt = await page.locator("[data-part=inbox-list] [data-ticket-id]").allInnerTexts();
  // lỗi 500
  await page.unrouteAll({ behavior: "ignoreErrors" }); await page.route(/\/tickets\?/, (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi máy chủ" }) }));
  await page.goto(`/inbox?course=${F.C}`); await page.waitForTimeout(3500); const errTxt = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 300); const retry = await page.getByRole("button", { name: "Thử lại" }).count(); await shot(page, "p7c-inbox-error");
  await page.unrouteAll({ behavior: "ignoreErrors" });
  // rỗng: lớp không có ticket
  const e = await newUser(browser, FA.teacher); await go(e.page, `/inbox?course=${FA.C}`); await e.page.waitForTimeout(2500); const emptyTxt = (await e.page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 300); const seeAll = await e.page.getByRole("button", { name: "Xem tất cả" }).count() + await e.page.getByRole("link", { name: "Xem tất cả" }).count(); const emptyCls = await cls(e.page); await shot(e.page, "p7c-inbox-empty");
  const okAll = /Không có câu hỏi nào đang chờ/.test(emptyTxt) && seeAll >= 1;
  R("TC-P4-03-38", okCls === "QC3T1" && skel >= 1 && okAll && /Thử lại/.test(errTxt) && retry >= 1 && c1 - c0 <= 0.05 && rowsTxt.length > 0 && rowsTxt.every((t) => /\S/.test(t)), `GV ${F.T}, lớp ${okCls}: GET tickets trễ 3,5 s → khung xương ${skel} phần tử sau 1,2 s (main: «${loadTxt}»); bộ lọc ${JSON.stringify(radios)}: nội dung lần lượt ${JSON.stringify(seen)}; CLS cộng dồn khi đổi OPEN→ANSWERED→ALL: ${(c1 - c0).toFixed(4)} (≤ 0,05); GET tickets 500: «${errTxt.slice(0, 200)}», nút «Thử lại»=${retry}; lớp ${emptyCls} không có ticket: «${emptyTxt.slice(0, 200)}», nút/liên kết «Xem tất cả»=${seeAll}; dòng ticket có chữ: ${JSON.stringify(rowsTxt.slice(0, 3).map((t) => t.replace(/\n+/g, " ").slice(0, 80)))}`);
  await e.ctx.close(); await ctx.close();
});

const menuBtn = (p: Page) => p.locator('button[aria-label="Tuỳ chọn bài"]').first();
const items = async (p: Page) => { await menuBtn(p).click(); await p.waitForTimeout(500); const t = await p.getByRole("menuitem").allInnerTexts(); return t; };

test("P4-06-34 xoá có xác nhận", async ({ browser }) => {
  const o = await newUser(browser, F.S[0]); await go(o.page, `/threads/${F.th.del}?course=${F.C}`); await o.page.waitForTimeout(1200);
  const c = await cls(o.page); await menuBtn(o.page).click(); await o.page.getByRole("menuitem", { name: /Xoá/ }).click(); await o.page.waitForTimeout(600);
  const dlg = (await o.page.getByRole("dialog").innerText()).replace(/\n+/g, " | "); const n = psql(`select count(*) from forum_posts where thread_id='${F.th.del}' and deleted_at is null`).trim(); const del0 = psql(`select (deleted_at is not null)::text from forum_threads where id='${F.th.del}'`);
  await shot(o.page, "p7c-delete-confirm"); await o.page.getByRole("button", { name: "Để sau" }).click();
  R("TC-P4-06-34", c === "QC3T1" && /Câu hỏi và 3 bình luận sẽ bị xoá/.test(dlg) && del0 === "false", `chủ bài ${F.S[0]} (lớp ${c}): menu … → Xoá → hộp «${dlg}» (ConfirmIrreversible: nút «Để sau» / «Xoá»); bài có 3 bình luận (DB ${n}); bấm «Để sau» → bài chưa xoá (deleted_at ${del0})`);
  await o.ctx.close();
});

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

test("P4-06-47 chữ Loại khỏi tri thức / Không hiển thị", async ({ browser }) => {
  const th = py(`s,d=A('${F.S[0]}','POST','/courses/${F.C}/threads',{'title':'QC3T1 bài có câu trả lời AI','body':'Nội dung câu hỏi để thử loại câu trả lời AI.','tags':[]}); print(d['thread']['id'])`);
  psql(`insert into forum_posts(course_id,thread_id,author_id,kind,verification_state,citations,confidence,body,ai_body) values('${F.C}','${th}',null,'AI','PENDING','[]',0.5,'Câu trả lời AI QC3T1','Câu trả lời AI QC3T1')`);
  const t = await newUser(browser, F.T); await go(t.page, `/threads/${th}?course=${F.C}`); await t.page.waitForTimeout(1500); const c = await cls(t.page);
  const btn = await t.page.getByRole("button", { name: "Loại khỏi tri thức" }).count(); const mit = await (async () => { const m = t.page.locator("[data-part=ai-post] button[aria-label=\"Tuỳ chọn bài\"]"); if (!(await m.count())) return [] as string[]; await m.first().click(); const r = await t.page.getByRole("menuitem").allInnerTexts(); await t.page.keyboard.press("Escape"); return r; })();
  const tTxt = (await t.page.locator("main").innerText()).replace(/\n+/g, " | "); await shot(t.page, "p7c-ai-teacher");
  if (btn) await t.page.getByRole("button", { name: "Loại khỏi tri thức" }).first().click(); else if (mit.some((x) => /Loại khỏi tri thức/.test(x))) { await t.page.locator("[data-part=ai-post] button[aria-label=\"Tuỳ chọn bài\"]").first().click(); await t.page.getByRole("menuitem", { name: /Loại khỏi tri thức/ }).click(); }
  await t.page.waitForTimeout(1500); const dlg = await t.page.getByRole("dialog").innerText().catch(() => ""); if (dlg) await t.page.getByRole("dialog").getByRole("button").last().click().catch(() => undefined); await t.page.waitForTimeout(2000);
  const st = psql(`select verification_state from forum_posts where thread_id='${th}' and kind='AI'`);
  const s = await newUser(browser, F.S[1]); await go(s.page, `/threads/${th}?course=${F.C}`); await s.page.waitForTimeout(1500); const sTxt = (await s.page.locator("main").innerText()).replace(/\n+/g, " | "); await shot(s.page, "p7c-ai-student-rejected");
  R("TC-P4-06-47(chữ)", c === "QC3T1" && (btn >= 1 || mit.some((x) => /Loại khỏi tri thức/.test(x))) && /Không hiển thị/.test(sTxt), `GV ${F.T} lớp ${c}: bài AI «Chờ xác nhận»: nút «Loại khỏi tri thức»=${btn}, menu … ${JSON.stringify(mit)}; hộp xác nhận «${dlg.replace(/\n+/g, " | ")}»; sau khi loại DB verification_state=${st}; SV ${F.S[1]} xem bài: «${sTxt.slice(0, 260)}» — có chữ «Không hiển thị»=${/Không hiển thị/.test(sTxt)} (DESIGN §13: Reject → SV «Không hiển thị», GV «Loại khỏi tri thức»)`);
  await s.ctx.close(); await t.ctx.close();
});

test("P4-06-50 báo cáo gửi lại sau mất kết nối", async ({ browser }) => {
  const rep = F.S[9]; const th3 = py(`s,d=A('${F.S[4]}','POST','/courses/${F.C}/threads',{'title':'QC3T1 bài báo cáo ba','body':'Nội dung bài dùng thử báo cáo khi mất kết nối.','tags':[]}); print(d['thread']['id'])`); const s = await newUser(browser, rep); await go(s.page, `/threads/${th3}?course=${F.C}`); await s.page.waitForTimeout(1200); const c = await cls(s.page);
  await s.page.context().route(/\/threads\/[^/]+\/report$/, async (r) => { if (r.request().method() === "POST") { await r.fetch(); await r.abort("failed"); } else await r.continue(); });
  await menuBtn(s.page).click(); await s.page.getByRole("menuitem", { name: /Báo cáo/ }).click(); await s.page.getByRole("button", { name: "Gửi báo cáo" }).click(); await s.page.waitForTimeout(3500);
  const n1 = psql(`select count(*) from post_reports where thread_id='${th3}' and reporter_id=(select id from users where email='${rep}@edupilot.local')`); const err1 = (await s.page.locator("main").innerText()).replace(/\n+/g, " | ").match(/(Không kết nối|Có lỗi|Chưa gửi|Không gửi)[^|]*/)?.[0] ?? "";
  await s.page.context().unroute(/\/threads\/[^/]+\/report$/); await s.page.getByRole("button", { name: "Gửi báo cáo" }).click().catch(() => undefined); await s.page.waitForTimeout(2500);
  const txt = (await s.page.locator("main").innerText()).replace(/\n+/g, " | "); const alerts = await s.page.getByRole("alert").allInnerTexts(); const n2 = psql(`select count(*) from post_reports where thread_id='${th3}' and reporter_id=(select id from users where email='${rep}@edupilot.local')`);
  await shot(s.page, "p7c-report-409");
  R("TC-P4-06-50", c === "QC3T1" && /Bạn đã báo cáo bài này/.test(txt) && n1 === "1" && n2 === "1" && !alerts.some((a) => /lỗi/i.test(a)), `lớp ${c}; ${rep}: yêu cầu báo cáo tới máy chủ nhưng phản hồi bị cắt (DB ${n1} báo cáo; giao diện «${err1}»); gửi lại cùng người → 409 ALREADY_REPORTED: giao diện có «Bạn đã báo cáo bài này»=${/Bạn đã báo cáo bài này/.test(txt)}, ô alert ${JSON.stringify(alerts)}; DB ${n2} báo cáo (không nhân đôi)`);
  await s.ctx.close();
});

test("P4-07-15 khủng hoảng giữ nháp", async ({ browser }) => {
  const u = F.S[3]; const { ctx, page } = await newUser(browser, u); await go(page, `/threads?course=${F.C}`); const c = await cls(page);
  const pre: number[] = []; page.on("request", (r) => { if (/\/threads\/precheck$/.test(r.url()) && r.method() === "POST") pre.push(Date.now()); });
  await page.getByRole("button", { name: "Đặt câu hỏi" }).first().click(); await page.waitForTimeout(500);
  const title = `QC3 nháp khủng hoảng ${Date.now()}`; await page.getByLabel("Tiêu đề", { exact: true }).fill(title);
  const body = "em không muốn sống nữa, mọi chuyện quá nặng nề"; const ta = page.getByLabel("Nội dung", { exact: true }); await ta.click(); const tt = Date.now(); await ta.pressSequentially(body, { delay: 60 }); const typeMs = Date.now() - tt;
  await page.waitForTimeout(1800); const nPre = pre.length;
  await page.getByRole("button", { name: "Đăng", exact: true }).click(); await page.waitForTimeout(1500);
  const dlg = page.getByRole("dialog"); const dTxt = (await dlg.innerText().catch(() => "")).replace(/\n+/g, " | "); const hasPrivate = await dlg.getByRole("button", { name: "Mở chat riêng" }).count(); const hasBack = await dlg.getByRole("button", { name: "Quay lại sửa" }).count();
  const primary = await dlg.getByRole("button", { name: "Mở chat riêng" }).evaluate((e) => e.className).catch(() => ""); await shot(page, "p7c-crisis-dialog");
  const dbN = psql(`select (select count(*) from forum_threads where title='${title}')+(select count(*) from forum_posts where body like '%em không muốn sống nữa, mọi chuyện%')+(select count(*) from chat_messages where content like '%em không muốn sống nữa, mọi chuyện%')`);
  await dlg.getByRole("button", { name: "Quay lại sửa" }).click(); await page.waitForTimeout(600); const kept = await ta.inputValue(); const keptT = await page.getByLabel("Tiêu đề", { exact: true }).inputValue();
  await page.getByRole("button", { name: "Đăng", exact: true }).click(); await page.waitForTimeout(1500); await page.getByRole("dialog").getByRole("button", { name: "Mở chat riêng" }).click(); await page.waitForURL(/\/chat/, { timeout: 15000 }).catch(() => undefined); await page.waitForTimeout(2000);
  const chatUrl = page.url().replace(/^https:\/\/[^/]+/, ""); const chatTxt = (await page.locator("main").innerText()).replace(/\n+/g, " | ");
  R("TC-P4-07-15", c === "QC3T1" && hasPrivate === 1 && hasBack === 1 && kept === body && keptT === title && dbN === "0" && nPre <= 2 && /em không muốn sống nữa/.test(chatTxt), `SV ${u} lớp ${c}: gõ ${body.length} ký tự trong ${typeMs} ms → số request precheck = ${nPre} (debounce 800 ms, không phải ${body.length}); bấm Đăng → hộp «${dTxt.slice(0, 200)}»: «Mở chat riêng»=${hasPrivate} (class ${primary.includes("primary") ? "primary" : primary}), «Quay lại sửa»=${hasBack}; DB (forum_threads + forum_posts + chat_messages chứa nháp) = ${dbN}; «Quay lại sửa» → ô Nội dung giữ nguyên=${kept === body}, Tiêu đề giữ=${keptT === title}; «Mở chat riêng» → ${chatUrl}, nháp có trong chat=${/em không muốn sống nữa/.test(chatTxt)}`);
  await ctx.close();
});
