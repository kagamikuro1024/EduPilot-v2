import { test } from "@playwright/test";
import { go, FX, log, newUser, shot } from "../specs/lib";
const FP = FX("fx_p5") as { C: string; SV: string };
test("hdr", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3pg"); await go(page, `/students/${FP.SV}?course=${FP.C}`); await page.getByRole("tab", { name: "Tổng quan" }).waitFor();
  const r = await page.evaluate(() => { const tl = document.querySelector("[role=tablist]") as HTMLElement; return Array.from(document.querySelectorAll("main button, main a")).filter((b) => b.getBoundingClientRect().bottom <= tl.getBoundingClientRect().top).map((b) => `${b.tagName}|${(b as HTMLElement).innerText.trim()}|${b.getAttribute("aria-label")}|${b.getAttribute("data-variant") ?? b.className.toString().slice(0, 50)}|${Math.round(b.getBoundingClientRect().x)},${Math.round(b.getBoundingClientRect().y)}`); });
  log("p505h", "hdr: " + JSON.stringify(r)); await shot(page, "p505-hdr"); await ctx.close();
});
