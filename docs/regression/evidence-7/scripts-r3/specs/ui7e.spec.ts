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


test("07-15 chat nháp", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, F.S[2]); await go(page, `/threads?course=${F.C}`);
  await page.getByRole("button", { name: "Đặt câu hỏi" }).first().click(); await page.waitForTimeout(500);
  const body = "em không muốn sống nữa, mọi chuyện quá nặng nề"; await page.getByRole("textbox", { name: /^Tiêu đề/ }).fill("QC3 nháp chat"); await page.getByRole("textbox", { name: /^Nội dung/ }).fill(body);
  await page.getByRole("button", { name: "Đăng", exact: true }).click(); await page.getByRole("dialog").getByRole("button", { name: "Mở chat riêng" }).click(); await page.waitForURL(/\/chat/, { timeout: 15000 }); await page.waitForTimeout(3000);
  const vals = await page.locator("textarea,input[type=text]").evaluateAll((e: any[]) => e.map((x) => x.value)); const txt = (await page.locator("main").innerText()).replace(/\n+/g, " | ");
  R("TC-P4-07-15b", vals.some((v: string) => v.includes("không muốn sống nữa")) || /không muốn sống nữa/.test(txt), `«Mở chat riêng» → ${page.url().replace(/^https:\/\/[^/]+/, "")}: giá trị ô nhập ${JSON.stringify(vals.map((v: string) => v.slice(0, 60)))}; main «${txt.slice(0, 200)}»`); await ctx.close();
});
test("47 menu thứ hai của GV", async ({ browser }) => {
  const th = py(`s,d=A('${F.S[0]}','POST','/courses/${F.C}/threads',{'title':'QC3T1 AI hai','body':'Nội dung câu hỏi thử menu AI.','tags':[]}); print(d['thread']['id'])`);
  psql(`insert into forum_posts(course_id,thread_id,author_id,kind,verification_state,citations,confidence,body,ai_body) values('${F.C}','${th}',null,'AI','PENDING','[]',0.5,'Câu trả lời AI hai','Câu trả lời AI hai')`);
  const t = await newUser(browser, F.T); await go(t.page, `/threads/${th}?course=${F.C}`); await t.page.waitForTimeout(1500);
  const ov = t.page.locator("[data-part=ai-post]").getByRole("button", { name: /Tuỳ chọn|Thêm|…/ }); const n = await ov.count(); const names = await ov.evaluateAll((e: any[]) => e.map((x) => x.getAttribute("aria-label")));
  let items: string[] = []; if (n >= 2) { await ov.nth(1).click(); await t.page.waitForTimeout(500); items = await t.page.getByRole("menuitem").allInnerTexts(); }
  R("TC-P4-06-47c", items.some((x) => /Loại khỏi tri thức/.test(x)), `GV ${F.T}: bài AI «Chờ xác nhận»: ${n} nút … trong khối AI ${JSON.stringify(names)}; menu của nút thứ hai: ${JSON.stringify(items)}`); await t.ctx.close();
});
