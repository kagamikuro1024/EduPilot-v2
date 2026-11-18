import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string };
test("@real UI7 TA bấm Công bố", async ({ page }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "ta"); await page.goto(`/grading?course=${S.course}`); await page.getByText("Chấm bài").first().waitFor(); await page.waitForTimeout(4000);
  const reqs: string[] = []; page.on("response", (r) => { if (/publish/.test(r.url())) reqs.push(`${r.request().method()} ${r.url().split("/api/v1")[1]} ${r.status()}`); });
  const b = page.locator("main button", { hasText: "Công bố" }).first();
  const info = await b.evaluate((e) => { const r = e.getBoundingClientRect(); const cs = getComputedStyle(e); let p: HTMLElement | null = e as HTMLElement; const anc: string[] = []; for (let i = 0; i < 4 && p; i++, p = p.parentElement) anc.push(p.tagName + "." + String(p.className).slice(0, 60)); return { vis: cs.visibility, disp: cs.display, w: Math.round(r.width), h: Math.round(r.height), y: Math.round(r.y), ariaHidden: e.closest("[aria-hidden=true]") !== null, inert: e.closest("[inert]") !== null, dis: (e as HTMLButtonElement).disabled, anc }; });
  const visible = await b.isVisible(); await shot(page, "p708-ta-publish");
  writeFileSync("/tmp/q9s/ui7.json", JSON.stringify({ info, visible, reqs }, null, 1)); return;
  writeFileSync("/tmp/q9s/ui7.json", JSON.stringify({ ctx, reqs, dialogs: await page.locator("[role=dialog], [role=alertdialog]").allInnerTexts(), alerts: await page.locator("[role=alert], [role=status]").allInnerTexts() }, null, 1));
});
