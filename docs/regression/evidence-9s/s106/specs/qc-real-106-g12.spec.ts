import { appendFileSync, readFileSync } from "node:fs";
import { test, type Browser } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
const RES = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_106.jsonl";
const D = JSON.parse(readFileSync("/Users/kuro/.config/edupilot-team/qc3-work/g12_case.json", "utf8")) as { a: string; tcs: string[]; must: string[]; mustnot: string[]; user?: string; click?: boolean; post?: string; wait?: number };
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 380)); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 20_000 });
const CID = () => sql(`select id from courses where class_code='QD9G'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
test("P12 UI «N lưu ý bị loại»", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, D.user ?? "teacher"); const errs: string[] = []; page.on("pageerror", (e) => errs.push(e.message));
  await page.goto(`/grading?tab=assignments&open=${D.a}`); await page.waitForTimeout(5500);
  if (D.post === "old") await page.route(/\/jobs\/[0-9a-f-]+$/, async (route) => { const r = await route.fetch(); const j = await r.json(); if (j && j.status === "SUCCEEDED") j.result = { drafts: 2 }; await route.fulfill({ response: r, json: j }); });
  if (D.click) { const early = (await page.locator("[data-part=guidance-panel]").innerText().catch(() => "")); await page.getByRole("button", { name: "Rút lưu ý" }).click(); await page.waitForTimeout(D.wait ?? 9000); }
  const panel = page.locator("[data-part=guidance-panel]"); const pc = await panel.count(); const t = (pc ? await panel.first().innerText() : await page.locator("main").innerText()).replace(/\n+/g, " ¦ ");
  const lines = t.match(/\d+ lưu ý bị loại[^¦]*/g) ?? []; const ok = D.must.every((m) => new RegExp(m).test(t)) && D.mustnot.every((m) => !new RegExp(m).test(t)) && errs.length === 0;
  const icons = await panel.locator("svg[data-icon=info], [class*=info]").count().catch(() => 0);
  await page.screenshot({ path: `${OUT}/s106-p12-${D.tcs[0]}.png` });
  for (const tc of D.tcs) verdict(tc, ok, `UI ngăn «Lưu ý chấm» (panel ${pc}): dòng «bị loại» ${JSON.stringify(lines)}; cần ${JSON.stringify(D.must)}, cấm ${JSON.stringify(D.mustnot)}; lỗi JS ${errs.length}; icon info ${icons}; chữ ngăn: ${t.slice(0, 260)}`);
  await ctx.close();
});
