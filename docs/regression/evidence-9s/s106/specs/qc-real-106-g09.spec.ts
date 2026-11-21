import { appendFileSync, readFileSync } from "node:fs";
import { test, type Browser } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
const RES = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_106.jsonl";
const D = JSON.parse(readFileSync("/Users/kuro/.config/edupilot-team/qc3-work/g09ui.json", "utf8")) as { cid: string; ids: Record<string,string>; stems: Record<string,string> };
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 380)); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 20_000 });
const CID = () => D.cid;
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
const st = (k: string) => sql(`select review_status||'/'||coalesce(reject_reason,'-') from question_bank where id='${D.ids[k]}'`)[0][0];
async function pick(page: any, k: string) { const row = page.getByRole("row").filter({ hasText: "QC106ui " + k }); await row.getByRole("checkbox").check(); }
test("TC-G09-03/04 Loại + lý do chọn nhanh", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto("/questions"); await page.waitForTimeout(3000);
  const dlg: string[] = []; page.on("dialog", (d: any) => { dlg.push(d.message()); d.dismiss(); });
  // a: loại, bỏ qua lý do
  await pick(page, "a"); await page.getByRole("button", { name: "Loại", exact: true }).click(); await page.waitForTimeout(1500);
  const chips = await page.getByRole("group", { name: "Lý do loại" }).getByRole("button").allInnerTexts(); const undoTxt = await page.getByRole("button", { name: "Hoàn tác" }).count(); const dialogs = await page.getByRole("dialog").count();
  const hidden = await page.getByRole("row").filter({ hasText: "QC106ui a" }).count(); const during = st("a"); await page.screenshot({ path: `${OUT}/s106-g0903.png` }); await page.waitForTimeout(6000); const afterA = st("a");
  // b: loại + chọn Mơ hồ
  await pick(page, "b"); await page.getByRole("button", { name: "Loại", exact: true }).click(); await page.getByRole("group", { name: "Lý do loại" }).getByRole("button", { name: "Mơ hồ" }).click(); const pressed = await page.getByRole("button", { name: "Mơ hồ" }).getAttribute("aria-pressed"); await page.waitForTimeout(6000); const afterB = st("b");
  // c: loại + chọn lý do rồi Hoàn tác trong 5 s
  await pick(page, "c"); await page.getByRole("button", { name: "Loại", exact: true }).click(); await page.getByRole("group", { name: "Lý do loại" }).getByRole("button", { name: "Lạc đề" }).click(); await page.getByRole("button", { name: "Hoàn tác" }).first().click(); await page.waitForTimeout(6500); const afterC = st("c"); const backC = await page.getByRole("row").filter({ hasText: "QC106ui c" }).count();
  verdict("TC-G09-03", chips.length === 6 && undoTxt >= 1 && dialogs === 0 && dlg.length === 0 && hidden === 0 && during.startsWith("PENDING"), `chọn câu PENDING → «Loại» một thao tác: dãy ${chips.length} nhãn ${JSON.stringify(chips)}, nút «Hoàn tác» ${undoTxt}, hộp thoại ${dialogs}/${dlg.length}, hàng ẩn ngay (${hidden === 0}), DB trong lúc chờ 5 s = ${during}`);
  verdict("TC-G09-04", afterA === "REJECTED/UNSPECIFIED" && afterB === "REJECTED/MO_HO" && pressed === "true" && afterC.startsWith("PENDING") && afterC.endsWith("/-") && backC === 1, `bỏ qua lý do → ${afterA}; chọn «Mơ hồ» (aria-pressed ${pressed}) → ${afterB}; chọn «Lạc đề» rồi «Hoàn tác» trong 5 s → ${afterC}, hàng quay lại ${backC === 1}`); await ctx.close();
});
