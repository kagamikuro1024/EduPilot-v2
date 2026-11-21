import { writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { courseId, go, mainText, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
test("probe", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(280_000);
  const out: Record<string, unknown> = {};
  const gv = await open(browser, "teacher");
  const c = await courseId(gv.page, "761987");
  for (const p of ["/insights", "/observability", "/documents", "/threads", "/gradebook", "/attendance", "/grading?tab=queue"]) {
    await go(gv.page, p, c, 3000);
    out[p] = { text: (await mainText(gv.page)).slice(0, 1800), panels: await gv.page.evaluate(() => [...document.querySelectorAll("[data-ep-panel]")].map((e) => ({ h: (e.querySelector("h2,h3")?.textContent ?? "").slice(0, 40), w: Math.round(e.getBoundingClientRect().width) }))), h: await gv.page.evaluate(() => [...document.querySelectorAll("main h1,main h2,main h3")].map((e) => e.tagName + ":" + (e.textContent ?? "").slice(0, 50))) };
  }
  await gv.ctx.close();
  const sv = await open(browser, "sv.gioi");
  const c2 = await courseId(sv.page, "761987");
  for (const p of ["/me", "/chat", "/threads"]) {
    await go(sv.page, p, c2, 3000);
    out["sv" + p] = { text: (await mainText(sv.page)).slice(0, 1500), panels: await sv.page.evaluate(() => [...document.querySelectorAll("[data-ep-panel]")].length), h: await sv.page.evaluate(() => [...document.querySelectorAll("main h1,main h2,main h3")].map((e) => e.tagName + ":" + (e.textContent ?? "").slice(0, 50))) };
  }
  await sv.ctx.close();
  writeFileSync("/tmp/qc2-s91/probe.json", JSON.stringify(out, null, 1));
});
