import { readFileSync, writeFileSync } from "node:fs";
import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; aid: string; res: Record<string, string>; sub: Record<string, string> };
const out: Record<string, unknown> = {};
const save = () => writeFileSync("/tmp/q9s/ui2.json", JSON.stringify(out, null, 1));
test("@real UI2 review: trích dẫn, tự lưu, phím A, a11y", async ({ page, context }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "teacher");
  await page.goto(`/grading/${S.sub["sv04"]}?course=${S.course}`);
  await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  // 08-05: bấm trích dẫn
  const quote = page.locator("main button", { hasText: "“" }).first();
  const qtext = ((await quote.innerText()).replace(/[“”]/g, "")).trim();
  const doc = page.locator("[class*=doc]").first();
  const before = await doc.evaluate((e) => (e as HTMLElement).scrollTop);
  await quote.click(); await page.waitForTimeout(800);
  const hl = await page.evaluate((q) => { const marks = Array.from(document.querySelectorAll("mark, [data-highlight], [class*=highlight], [class*=Highlight], [class*=hl]")); return marks.map((m) => ({ tag: m.tagName, cls: String((m as HTMLElement).className).slice(0, 50), text: (m.textContent ?? "").slice(0, 80), match: q.startsWith((m.textContent ?? "").trim().slice(0, 30)) || (m.textContent ?? "").includes(q.slice(0, 30)) })); }, qtext);
  out.quote = qtext.slice(0, 70); out.highlights = hl; out.scrollBefore = before; out.scrollAfter = await doc.evaluate((e) => (e as HTMLElement).scrollTop);
  await shot(page, "p708-highlight");
  // 08-09: gõ nhận xét, tắt mạng, tải lại
  const overall = page.locator("main textarea").last(); const marker = "NHAN-XET-TAM-" + Date.now();
  await overall.click(); await overall.fill((await overall.inputValue()) + " " + marker);
  await context.setOffline(true); await page.waitForTimeout(3500);
  await context.setOffline(false); await page.reload(); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  const restored = await page.locator("main textarea").last().inputValue(); out.autosaveRestored = restored.includes(marker); out.restoredTail = restored.slice(-60);
  out.helperLabel = await page.getByText(/Đã sửa bởi giảng viên|bản nháp|Đã khôi phục/).allInnerTexts();
  // 08-10: phím A khi không gõ
  await page.locator("body").click({ position: { x: 5, y: 5 } }); await page.keyboard.press("Escape");
  const bar0 = await page.getByText(/Đã duyệt \d+\/\d+/).first().innerText();
  const url0 = page.url(); await page.keyboard.press("a"); await page.waitForTimeout(2500);
  out.bar0 = bar0; out.bar1 = await page.getByText(/Đã duyệt \d+\/\d+/).first().innerText().catch(() => "?"); out.urlChanged = page.url() !== url0;
  out.toasts = await page.getByText(/Thành công|thành công/).count();
  await shot(page, "p708-after-A");
  // 08-13: focus ring + axe
  await page.keyboard.press("Tab"); await page.keyboard.press("Tab");
  out.focusStyle = await page.evaluate(() => { const e = document.activeElement as HTMLElement; const cs = getComputedStyle(e); return { tag: e.tagName, outline: cs.outlineStyle + " " + cs.outlineWidth, shadow: cs.boxShadow.slice(0, 60) }; });
  const ax = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze(); out.axe = ax.violations.map((v) => ({ id: v.id, impact: v.impact, n: v.nodes.length }));
  save();
  expect(true).toBe(true);
});
