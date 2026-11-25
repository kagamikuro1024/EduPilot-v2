import { readFileSync, writeFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
import { needReal, noOverflow, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; aid: string; res: Record<string, string>; sub: Record<string, string>; aid2: string };
const out: Record<string, unknown> = {};
const save = () => writeFileSync("/tmp/q9s/ui1.json", JSON.stringify(out, null, 1));
async function open(page: import("@playwright/test").Page, user: string, path: string) {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, user);
  await page.goto(path);
  await page.locator("main").waitFor(); await page.waitForTimeout(3500);
}
test("@real UI1 teacher queue + review", async ({ page }) => {
  await open(page, "teacher", `/grading?course=${S.course}`);
  await shot(page, "p708-queue-1440");
  out.heads = await page.locator("main [role=columnheader], main th").allInnerTexts();
  out.selects = await page.locator("main select").evaluateAll((els) => els.map((e) => ({ label: (e as HTMLSelectElement).getAttribute("aria-label") ?? (e.closest("label")?.textContent ?? ""), value: (e as HTMLSelectElement).value, text: (e as HTMLSelectElement).selectedOptions[0]?.textContent })));
  out.chips = await page.locator("main [aria-pressed=true], main [aria-selected=true], main [aria-checked=true]").evaluateAll((els) => els.map((e) => (e.textContent ?? "").trim().slice(0, 40)));
  const rows = page.locator("main [role=row], main tbody tr");
  out.rowCount = await rows.count();
  out.rows = await rows.evaluateAll((els) => els.slice(0, 14).map((e) => (e as HTMLElement).innerText.replace(/\s*\n+\s*/g, " | ").slice(0, 230)));
  const q = await page.locator("main").innerText();
  out.tech = ["trace", "model", "flags", "run_state", "NEEDS_REVIEW", "ai_scores"].filter((w) => q.includes(w));
  out.flagTexts = ["Hai lượt chấm lệch hơn 1 điểm", "Bài lạc đề", "Nghi chèn lệnh", "Thiếu nội dung", "Trích dẫn không có trong bài", "AI không chắc", "Nhận xét giống đáp án chuẩn", "Thư nộp quá hạn tối đa"].filter((t) => q.includes(t));
  out.sourceLabels = ["Trong EduPilot", "ZIP", "Thư", "Teams"].filter((t) => q.includes(t));
  // review of B (sv.kha)
  await page.goto(`/grading/${S.sub["sv.kha"]}?course=${S.course}`);
  await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  await shot(page, "p708-review-B");
  const rv = await page.locator("main").innerText(); out.reviewText = rv.slice(0, 200) + " … " + rv.slice(-1800);
  out.btns = await page.locator("main button").allInnerTexts();
  out.marks = await page.locator("main mark").count();
  out.quoteEls = await page.locator("main [data-part*=quote], main [data-part*=evidence], main button:has-text('“')").count();
  out.inputs = await page.locator("main textarea, main input").evaluateAll((els) => els.map((e) => ({ t: (e as HTMLInputElement).type, a: e.getAttribute("aria-label"), v: (e as HTMLInputElement).value.slice(0, 20) })));
  out.notices = await page.locator("main [role=status], main [class*=Notice]").allInnerTexts();
  out.layout = await page.evaluate(() => { const g = (s: string) => { const e = document.querySelector(s) as HTMLElement | null; if (!e) return null; const r = e.getBoundingClientRect(); return { x: Math.round(r.x), w: Math.round(r.width) }; }; return { main: g("[class*=splitMain]"), aside: g("[class*=splitAside]") }; });
  save();
  await noOverflow(page, true);
});
test("@real UI1 TA không có Công bố", async ({ page }) => {
  await open(page, "ta", `/grading?course=${S.course}`);
  out.taQueueBtns = await page.locator("main button").allInnerTexts();
  out.taPublishInDom = await page.getByText(/Công bố/).count();
  await page.goto(`/grading/${S.sub["sv.kha"]}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3000);
  out.taReviewBtns = await page.locator("main button").allInnerTexts(); out.taReviewPublish = await page.getByText(/Công bố/).count();
  save();
});
