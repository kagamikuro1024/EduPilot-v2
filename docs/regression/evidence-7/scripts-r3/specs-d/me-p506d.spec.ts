import { test } from "@playwright/test";
import { go, FX, log, newUser, shot } from "../specs/lib";
const FM = FX("fx_me") as { M: string }; const FP = FX("fx_p5") as { B: string; Cc: string };
const L = (s: string) => log("p506d", s);
test("probe Thấp + rỗng", async ({ browser }) => {
  const c = await newUser(browser, "qc3me03"); await go(c.page, `/me?course=${FM.M}`); await c.page.getByText("Kết quả của tôi").first().waitFor(); await c.page.waitForTimeout(2000);
  const hits = await c.page.evaluate(() => { const out: any[] = []; const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT); let n: Node | null; while ((n = w.nextNode())) { if (/thấp/i.test(n.textContent ?? "")) { const e = n.parentElement!; const cs = getComputedStyle(e); out.push({ text: (n.textContent ?? "").trim().slice(0, 80), tag: e.tagName, cls: String(e.className).slice(0, 80), color: cs.color, bg: cs.backgroundColor, inMain: !!e.closest("main"), ancestors: (() => { const a: string[] = []; let p: Element | null = e; for (let i = 0; i < 5 && p; i++, p = p.parentElement) a.push(p.tagName + (p.getAttribute("data-part") ? "[" + p.getAttribute("data-part") + "]" : "")); return a.join("<"); })() }); } } const attrs = Array.from(document.querySelectorAll("[title],[aria-label]")).filter((e) => /thấp/i.test((e.getAttribute("title") ?? "") + (e.getAttribute("aria-label") ?? ""))).map((e) => e.tagName + ":" + (e.getAttribute("title") ?? e.getAttribute("aria-label"))); return { out, attrs }; });
  L("06-15 «Thấp» trong DOM /me qc3me03: " + JSON.stringify(hits));
  L("06-15 main text: " + (await c.page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 900));
  await shot(c.page, "p506d-15"); await c.ctx.close();
  for (const [u, cid] of [["qc3p5s04", FP.B], ["qc3p5s05", FP.Cc]] as const) {
    const s = await newUser(browser, u); await go(s.page, `/me?course=${cid}`); await s.page.waitForTimeout(2500);
    L(`06-18 ${u}: ` + (await s.page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 700)); await shot(s.page, `p506d-18-${u}`); await s.ctx.close();
  }
});
