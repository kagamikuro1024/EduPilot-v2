import { appendFileSync } from "node:fs";
import { expect, test, type Browser, type Page } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
// QC smoke 10.6 (GD) trên ep-qc, lớp QD9G. Kết quả ghi QC_RES (jsonl {tc,kq,note}).
const BASE = process.env.QC_BASE ?? "https://localhost:943";
const RES = process.env.QC_RES ?? `${OUT}/results_106.jsonl`;
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 260)); };
test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
test.use({ actionTimeout: 30_000 });
const CID = () => sql(`select id from courses where class_code='QD9G'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: BASE, ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage();
  await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page };
}
const sw = (p: Page, name = "Lưu ý chấm") => p.getByRole("switch", { name });
const flagRow = () => (sql(`select enabled, version from course_guidance_flags where course_id='${CID()}' and feature='GRADING_GUIDANCE'`)[0] ?? ["f", "1"]);
const audits = () => Number(sql(`select count(*) from audit_log where entity like '%guidance%' and created_at > now() - interval '1 day'`)[0][0]);

test("TC-G01-14 công tắc lạc quan + Hoàn tác (không xác nhận, không toast), version +2, 2 dòng audit", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto("/class/settings"); await page.waitForTimeout(2500);
  const s0 = await sw(page).getAttribute("aria-checked"); const [e0, v0] = flagRow(); const a0 = audits();
  const dlgBefore = await page.getByRole("dialog").count();
  await sw(page).click(); const t0 = Date.now(); await page.waitForTimeout(250);
  const s1 = await sw(page).getAttribute("aria-checked"); const undoVis = await page.getByRole("button", { name: "Hoàn tác" }).count(); const dlg = await page.getByRole("dialog").count();
  const txt = await page.locator("main").innerText(); const toast = /Thành công|thành công/.test(await page.locator("body").innerText());
  await page.screenshot({ path: `${OUT}/s106-g0114-1.png` });
  await page.waitForTimeout(1500); const [e1, v1] = flagRow();
  await page.getByRole("button", { name: "Hoàn tác" }).click(); await page.waitForTimeout(2500);
  const s2 = await sw(page).getAttribute("aria-checked"); const [e2, v2] = flagRow(); const a2 = audits();
  verdict("TC-G01-14", s0 !== s1 && undoVis > 0 && dlg === dlgBefore && !toast && s2 === s0 && Number(v2) === Number(v0) + 2 && a2 - a0 === 2, `bấm «Lưu ý chấm»: ${s0}→${s1} sau ${Date.now() - t0 - 250} ms (lạc quan); dòng «Hoàn tác» ×${undoVis}; hộp thoại ${dlg}; toast «Thành công» ${toast}; DB sau bật enabled=${e1} v${v1}; Hoàn tác → ${s2}, enabled=${e2}, version ${v0}→${v2} (+${Number(v2) - Number(v0)}); audit_log +${a2 - a0}`);
  await ctx.close();
});

test("TC-G01-15 375 px: công tắc vùng chạm ≥ 44 px, không tràn ngang", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher", 375); await page.goto("/class/settings"); await page.waitForTimeout(2500);
  const rows = await page.locator("[data-part=guidance-flag]").evaluateAll((els) => els.map((e) => { const s = e.querySelector("[role=switch]") as HTMLElement; const r = s.getBoundingClientRect(); const lb = s.closest("label") ?? s.parentElement!; const lr = lb.getBoundingClientRect(); return { name: s.getAttribute("aria-label") ?? s.textContent, w: Math.round(r.width), h: Math.round(r.height), lw: Math.round(lr.width), lh: Math.round(lr.height) }; }));
  const ov = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1); await page.screenshot({ path: `${OUT}/s106-g0115.png` });
  const ok = rows.length >= 4 && rows.every((r) => Math.max(r.h, r.lh) >= 44 && Math.max(r.w, r.lw) >= 44) && !ov;
  verdict("TC-G01-15", ok, `375 px: ${rows.length} công tắc; vùng chạm (switch / hàng nhãn): ${JSON.stringify(rows.map((r) => `${r.w}x${r.h} / ${r.lw}x${r.lh}`))}; tràn ngang ${ov}`);
  await ctx.close();
});

test("TC-G01-17 hai tab cùng teacher: tab 2 lưu version cũ → 409, tải lại hàng + báo tại chỗ", async ({ browser }) => {
  const a = await pageAs(browser, "teacher"); const b = await pageAs(browser, "teacher");
  await a.page.goto("/class/settings"); await b.page.goto("/class/settings"); await a.page.waitForTimeout(2500); await b.page.waitForTimeout(1500);
  const initial = await sw(b.page).getAttribute("aria-checked");
  await sw(a.page).click(); await a.page.waitForTimeout(2500); // tab 1 đổi cờ → version +1
  const vNow = flagRow()[1]; const reqs: string[] = []; b.page.on("response", (r) => { if (/guidance-flags/.test(r.url()) && r.request().method() === "PUT") reqs.push(`${r.status()}`); });
  await sw(b.page).click(); await b.page.waitForTimeout(3000); await b.page.screenshot({ path: `${OUT}/s106-g0117.png` });
  const state = await sw(b.page).getAttribute("aria-checked"); const db = flagRow(); const txt = await b.page.locator("[data-part=guidance-flags]").innerText();
  verdict("TC-G01-17", reqs.includes("409") && state === `${db[0] === "t"}` && /đã thay đổi|tải lại|Thử lại|Chưa lưu/i.test(txt), `tab 2 PUT với version cũ → mã ${JSON.stringify(reqs)}; công tắc tab 2 sau đó ${state} (DB enabled=${db[0]}, v${db[1]}); chữ khối: «${txt.replace(/\n/g, " ").slice(0, 160)}»`);
  // hoàn nguyên: về trạng thái ban đầu
  const cur = await sw(b.page).getAttribute("aria-checked"); if (cur !== initial) { await sw(b.page).click(); await b.page.waitForTimeout(2000); }
  await a.ctx.close(); await b.ctx.close();
});
