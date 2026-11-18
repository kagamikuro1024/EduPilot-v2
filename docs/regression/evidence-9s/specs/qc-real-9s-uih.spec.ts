import { writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, sql, uiLogin } from "./qc-real-lib";

needReal();
test("@real UIH 08-03 hàng chờ 1.000 bài: DOM và cuộn", async ({ page }) => {
  const out: Record<string, unknown> = {};
  const C = sql("select id from courses where class_code='QD9P'")[0][0];
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), C);
  await uiLogin(page, "teacher");
  await page.goto(`/grading?course=${C}`); await page.getByText("Chấm bài").first().waitFor(); await page.waitForTimeout(4000);
  await page.locator("main select").last().selectOption({ label: "Chưa duyệt" }).catch(() => null); await page.waitForTimeout(2500);
  out.rowsInDom = await page.locator("main [role=row], main tbody tr").count();
  const sc = await page.evaluate(() => { const els = Array.from(document.querySelectorAll("main *")).filter((e) => (e as HTMLElement).scrollHeight > (e as HTMLElement).clientHeight + 200 && getComputedStyle(e).overflowY !== "visible"); const e = els.sort((a, b) => b.scrollHeight - a.scrollHeight)[0] as HTMLElement | undefined; return e ? { cls: String(e.className).slice(0, 40), sh: e.scrollHeight, ch: e.clientHeight } : null; });
  out.scroller = sc;
  const fps = await page.evaluate(async () => {
    const els = Array.from(document.querySelectorAll("main *")).filter((e) => (e as HTMLElement).scrollHeight > (e as HTMLElement).clientHeight + 200 && getComputedStyle(e).overflowY !== "visible"); const e = (els.sort((a, b) => (b as HTMLElement).scrollHeight - (a as HTMLElement).scrollHeight)[0] ?? document.scrollingElement) as HTMLElement;
    const frames: number[] = []; let last = performance.now(); let y = 0; await new Promise<void>((res) => { const step = (t: number) => { frames.push(t - last); last = t; y += 40; e.scrollTop = y; if (y < 6000 && frames.length < 400) requestAnimationFrame(step); else res(); }; requestAnimationFrame(step); });
    frames.shift(); const avg = frames.reduce((a, b) => a + b, 0) / frames.length; frames.sort((a, b) => a - b); return { n: frames.length, avgMs: +avg.toFixed(1), p95Ms: +frames[Math.floor(frames.length * .95)].toFixed(1), fps: +(1000 / avg).toFixed(1), rowsAfter: document.querySelectorAll("main [role=row], main tbody tr").length };
  });
  out.fps = fps; await shot(page, "p708-1000-scroll");
  writeFileSync("/tmp/q9s/uih.json", JSON.stringify(out, null, 1));
});
