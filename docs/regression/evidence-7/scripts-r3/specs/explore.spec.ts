import { test } from "@playwright/test";
import { login, go, FX, log } from "./lib";
test("explore chat", async ({ page }) => {
  const fx = FX("fx_ui1");
  await login(page, "qc3s26");
  await go(page, `/chat?session=${fx.sid}`);
  const out = await page.evaluate(() => {
    const markers = ["NORMAL","CRISIS","OTHER","SMALL","FAILED","CANCELLED","STREAMING","AUTO"];
    const res: any = {};
    for (const m of markers) {
      const els = Array.from(document.querySelectorAll("main *")).filter((e) => e.children.length === 0 && (e.textContent ?? "").includes("QCUI-" + m));
      if (!els.length) { res[m] = "not-rendered"; continue; }
      let node: Element = els[0];
      while (node.parentElement) { const p = node.parentElement; const c = markers.filter((x) => (p as HTMLElement).innerText.includes("QCUI-" + x)).length; if (c > 1) break; node = p; }
      res[m] = { ask: node.querySelectorAll("[data-part=ask-staff]").length, status: node.querySelectorAll("[data-part=ticket-status]").length, text: (node as HTMLElement).innerText.slice(0, 120) };
    }
    return res;
  });
  log("explore", JSON.stringify(out, null, 1));
  log("explore", "MSG TEXT=" + (await page.locator("main").innerText()).slice(0, 1500));
});
