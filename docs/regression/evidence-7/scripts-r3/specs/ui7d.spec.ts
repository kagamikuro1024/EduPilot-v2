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


test("38b lớp 2 rỗng", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, F.T); await go(page, `/inbox?course=${F.C}`); await page.getByRole("radio", { name: "Đã nhận" }).click(); await page.waitForTimeout(2000);
  const c = await cls(page); const txt = (await page.locator("main").innerText()).replace(/\n+/g, " | "); const all = await page.getByRole("button", { name: "Xem tất cả" }).count() + await page.getByRole("link", { name: "Xem tất cả" }).count(); await shot(page, "p7c-inbox-empty2");
  R("TC-P4-03-38b", false, `lớp ${c} (có ticket ở trạng thái khác), bộ lọc «Đã nhận» rỗng: «${txt.slice(0, 260)}», «Xem tất cả»=${all}`); await ctx.close();
});
test("47b SV thấy bài AI bị loại", async ({ browser }) => {
  const th = psql(`select thread_id from forum_posts where kind='AI' and verification_state='REJECTED' and course_id='${F.C}' limit 1`);
  const s = await newUser(browser, F.S[1]); await go(s.page, `/threads/${th}?course=${F.C}`); await s.page.waitForTimeout(1500); const c = await cls(s.page); const txt = (await s.page.locator("main").innerText()).replace(/\n+/g, " | "); await shot(s.page, "p7c-ai-student-rejected2");
  R("TC-P4-06-47b", /Không hiển thị/.test(txt), `SV ${F.S[1]} lớp ${c}, bài AI đã bị GV loại (verification_state=REJECTED): «${txt.slice(0, 300)}»`); await s.ctx.close();
});
test("07-15 b", async ({ browser }) => {
  const u = F.S[3]; const { ctx, page } = await newUser(browser, u); await go(page, `/threads?course=${F.C}`); const c = await cls(page);
  const pre: number[] = []; page.on("request", (r) => { if (/\/threads\/precheck$/.test(r.url()) && r.method() === "POST") pre.push(Date.now()); });
  await page.getByRole("button", { name: "Đặt câu hỏi" }).first().click(); await page.waitForTimeout(500);
  const title = `QC3 nháp khủng hoảng ${Date.now()}`; const ti = page.getByRole("textbox", { name: /^Tiêu đề/ }); const ta = page.getByRole("textbox", { name: /^Nội dung/ }); await ti.fill(title);
  const body = "em không muốn sống nữa, mọi chuyện quá nặng nề"; await ta.click(); const tt = Date.now(); await ta.pressSequentially(body, { delay: 60 }); const typeMs = Date.now() - tt;
  await page.waitForTimeout(1800); const nPre = pre.length;
  await page.getByRole("button", { name: "Đăng", exact: true }).click(); await page.waitForTimeout(1500);
  const dlg = page.getByRole("dialog"); const dTxt = (await dlg.innerText().catch(() => "")).replace(/\n+/g, " | "); const hasPrivate = await dlg.getByRole("button", { name: "Mở chat riêng" }).count(); const hasBack = await dlg.getByRole("button", { name: "Quay lại sửa" }).count(); await shot(page, "p7c-crisis-dialog");
  const dbN = psql(`select (select count(*) from forum_threads where title='${title}')+(select count(*) from forum_posts where body like '%em không muốn sống nữa, mọi chuyện%')+(select count(*) from chat_messages where content like '%em không muốn sống nữa, mọi chuyện%')`);
  await dlg.getByRole("button", { name: "Quay lại sửa" }).click(); await page.waitForTimeout(600); const kept = await ta.inputValue(); const keptT = await ti.inputValue();
  await page.getByRole("button", { name: "Đăng", exact: true }).click(); await page.waitForTimeout(1500); await page.getByRole("dialog").getByRole("button", { name: "Mở chat riêng" }).click(); await page.waitForURL(/\/chat/, { timeout: 15000 }).catch(() => undefined); await page.waitForTimeout(2500);
  const chatUrl = page.url().replace(/^https:\/\/[^/]+/, ""); const chatTxt = (await page.locator("main").innerText().catch(() => "")).replace(/\n+/g, " | ");
  R("TC-P4-07-15", c === "QC3T1" && hasPrivate === 1 && hasBack === 1 && kept === body && keptT === title && dbN === "0" && nPre <= 2 && /em không muốn sống nữa/.test(chatTxt), `SV ${u} lớp ${c}: gõ ${body.length} ký tự trong ${typeMs} ms → số request precheck = ${nPre} (debounce 800 ms); bấm Đăng → hộp «${dTxt.slice(0, 220)}»: «Mở chat riêng»=${hasPrivate}, «Quay lại sửa»=${hasBack}; DB (forum_threads+forum_posts+chat_messages chứa nháp) = ${dbN}; «Quay lại sửa» → Nội dung giữ=${kept === body}, Tiêu đề giữ=${keptT === title}; «Mở chat riêng» → ${chatUrl}, nháp có trong chat=${/em không muốn sống nữa/.test(chatTxt)}`);
  await ctx.close();
});
