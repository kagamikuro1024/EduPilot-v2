import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; sub: Record<string, string>; sub2: Record<string, string> };
const E = JSON.parse(readFileSync("/tmp/q9s/ui08e.json", "utf8")) as { sub: string; quote: string };
const F = JSON.parse(readFileSync("/tmp/q9s/ui08f.json", "utf8")) as { sub: Record<string, string> };
const out: Record<string, unknown> = {};
const save = () => writeFileSync("/tmp/q9s/uie.json", JSON.stringify(out, null, 1));
const go = async (page: import("@playwright/test").Page, sub: string) => { await page.goto(`/grading/${sub}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500); };
test("@real UIE 08-05 + 08-09", async ({ page, context }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "teacher");
  await go(page, E.sub);
  const q = page.locator("main button", { hasText: "😀" }).first(); out.quoteBtns = await page.locator("main button", { hasText: "😀" }).count();
  await q.click(); await page.waitForTimeout(900);
  const mk = page.locator("main mark"); out.marks = await mk.count(); out.markText = (await mk.first().innerText().catch(() => "")); out.markEqQuote = out.markText === E.quote;
  out.markInViewport = await mk.first().evaluate((e) => { const r = e.getBoundingClientRect(); return r.top >= 0 && r.bottom <= window.innerHeight; }).catch(() => null);
  await shot(page, "p708-highlight-emoji");
  // 08-05 cuộn: trích dẫn thấp nhất của bài mẫu S7 (sv.kha, lớp QD9V)
  await go(page, S.sub2["sv.kha"]);
  const btns = page.locator("main button", { hasText: "“" }); const n = await btns.count(); await btns.nth(n - 1).click(); await page.waitForTimeout(900);
  out.last = { n, mark: await page.locator("main mark").count(), inView: await page.locator("main mark").first().evaluate((e) => { const r = e.getBoundingClientRect(); return r.top >= 0 && r.bottom <= window.innerHeight; }).catch(() => null), scroll: await page.evaluate(() => Array.from(document.querySelectorAll("main *")).filter((e) => (e as HTMLElement).scrollTop > 0).map((e) => String((e as HTMLElement).className).slice(0, 40) + ":" + (e as HTMLElement).scrollTop)) };
  // 08-09 nhãn «Đã sửa bởi giảng viên» (GV thấy) và nháp theo từng bài khi chuyển bài
  out.editedLabelTeacher = await page.getByText("Đã sửa bởi giảng viên").count();
  const ov = page.locator("main textarea").last(); const mark = "NHAP-A-" + Date.now(); await ov.fill(mark); await page.waitForTimeout(1500);
  await go(page, S.sub["sv07"]); const ov2 = page.locator("main textarea").last(); out.otherEmpty = !(await ov2.inputValue()).includes("NHAP-A-");
  await context.setOffline(true); await page.waitForTimeout(2500); await context.setOffline(false);
  await go(page, S.sub2["sv.kha"]); out.draftBack = (await page.locator("main textarea").last().inputValue()).includes(mark);
  save();
});
test("@real UIE 08-09 nhãn không lộ cho sinh viên", async ({ page }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "sv.kha"); await page.goto(`/assignments/${(JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { aid2: string }).aid2}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  out.editedLabelStudent = await page.getByText("Đã sửa bởi giảng viên").count(); save();
});
test("@real UIE 08-11 RUNNING + nộp lại", async ({ page }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "teacher");
  await go(page, F.sub["sv34"]);
  const btn = page.getByRole("button", { name: "Duyệt bài" }); out.runningText = (await page.locator("main").innerText()).split("\n").filter((l) => /Đang chấm|chưa duyệt được/.test(l)); out.approveDisabled = await btn.isDisabled(); out.approveTitle = await btn.getAttribute("title"); await shot(page, "p708-running");
  await go(page, F.sub["sv33"]); out.beforeBanner = (await page.locator("main").innerText()).includes("Sinh viên vừa nộp lại");
  save();
  for (let i = 0; i < 40; i++) { if (await page.getByText("Sinh viên vừa nộp lại").count()) break; await page.waitForTimeout(3000); }
  out.banner = await page.getByText(/Sinh viên vừa nộp lại/).allInnerTexts(); out.approveAfter = await page.getByRole("button", { name: "Duyệt bài" }).isDisabled().catch(() => null); await shot(page, "p708-superseded"); save();
});
