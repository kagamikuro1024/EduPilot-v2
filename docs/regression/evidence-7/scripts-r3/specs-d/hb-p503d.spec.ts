import { test, type Page } from "@playwright/test";
import { go, FX, log, psql, newUser, expect } from "../specs/lib";
const FX5 = FX("r4fx") as { X: string; S: string[] }; const X = FX5.X; const [, , , , , , s6, , s8] = FX5.S;
const L = (s: string) => log("p503d", s);
async function hbPage(browser: any, user: string, path: string) {
  const { ctx, page } = await newUser(browser, user); const beats: Array<{ t: number; fake: number; st?: number }> = [];
  page.on("request", (r) => { if (r.url().includes("/activity/heartbeat") && r.method() === "POST") beats.push({ t: Date.now(), fake: -1 }); });
  page.on("response", (r) => { if (r.url().includes("/activity/heartbeat") && r.request().method() === "POST") { const x = beats.find((b) => b.st === undefined); if (x) x.st = r.status(); } });
  await page.addInitScript(() => { Math.random = () => 0; (window as any).__bt = []; const of = window.fetch; window.fetch = function (...a: any[]) { if (String(a[0]).includes("/activity/heartbeat")) (window as any).__bt.push(Date.now()); return (of as any).apply(this, a); }; });
  await page.clock.install({ time: new Date() }); await page.goto(path); await page.waitForTimeout(2500); return { ctx, page, beats };
}
const bt = (p: Page) => p.evaluate(() => (window as any).__bt as number[]);
test("08 đo từng nhịp", async ({ browser }) => {
  const { ctx, page, beats } = await hbPage(browser, s8, `/threads?course=${X}`);
  const t0 = await page.evaluate(() => Date.now());
  for (let i = 0; i < 13; i++) { await page.clock.runFor(10_000); await page.waitForTimeout(400); }
  const tt = (await bt(page)).map((x) => Math.round((x - t0) / 1000)); L(`08 nhịp theo giây giả lập sau t0 (fetch trong trang): ${JSON.stringify(tt)}; số request ngoài: ${beats.length}`);
  for (let i = 0; i < 18; i++) { await page.clock.runFor(10_000); await page.waitForTimeout(300); }
  const tt2 = (await bt(page)).map((x) => Math.round((x - t0) / 1000)); L(`08 sau 310 s: ${JSON.stringify(tt2)}`);
  await page.mouse.click(300, 300); await page.clock.runFor(1000); await page.waitForTimeout(1200); const tt3 = (await bt(page)).map((x) => Math.round((x - t0) / 1000)); L(`08 chạm lại: ${JSON.stringify(tt3)}`);
  await ctx.close();
});

test("08b một lần runFor 130 s như bản cũ", async ({ browser }) => {
  const { ctx, page, beats } = await hbPage(browser, s8, `/threads?course=${X}`);
  const t0 = await page.evaluate(() => Date.now());
  await page.clock.runFor(130_000); await page.waitForTimeout(1500);
  L(`08b runFor(130 s) một lần: nhịp theo giây giả lập (fetch trong trang) ${JSON.stringify((await bt(page)).map((x) => Math.round((x - t0) / 1000)))}; request ngoài ${beats.length}`);
  await page.clock.runFor(180_000); await page.waitForTimeout(1500);
  L(`08b thêm runFor(180 s): ${JSON.stringify((await bt(page)).map((x) => Math.round((x - t0) / 1000)))}`);
  await ctx.close();
});
test("10 mạng đứt, 403/422 (đo từng nhịp)", async ({ browser }) => {
  const { ctx, page, beats } = await hbPage(browser, s6, `/chat?course=${X}`); const t0 = await page.evaluate(() => Date.now());
  const step = async (sec: number) => { for (let i = 0; i < sec / 5; i++) { await page.keyboard.press("Shift"); await page.clock.runFor(5000); await page.waitForTimeout(150); } };
  const seq = async () => (await bt(page)).map((x) => Math.round((x - t0) / 1000));
  await step(20); const a = (await seq()).length;
  await ctx.setOffline(true); await step(95); const b = (await seq()).length; const offSeq = await seq();
  await ctx.setOffline(false); await step(35); const c = (await seq()).length;
  let mode = 403; await page.route(/\/activity\/heartbeat/, (r) => r.fulfill({ status: mode, contentType: "application/json", body: JSON.stringify({ code: mode === 403 ? "FORBIDDEN" : "VALIDATION_FAILED", message: "x" }) }));
  await step(65); const d = (await seq()).length; const alert = await page.locator("[role=alert],[data-ep-toast],[class*=toast]").count(); const dlg = await page.getByRole("dialog").count();
  await page.unrouteAll(); await step(65); const e = (await seq()).length;
  L(`10 nhịp(giây giả lập)=${JSON.stringify(await seq())}; trước offline ${a}; sau 95 s offline ${b} (+${b - a}); online +35 s ${c} (+${c - b}); sau 403 (65 s) ${d} (+${d - c}); alert/toast=${alert}, dialog=${dlg}; bỏ chặn 65 s: ${e} (+${e - d}); offSeq=${JSON.stringify(offSeq)}`);
  // 422 ở tab mới
  await ctx.close();
});
test("08c nhật ký mạng", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, s8); const net: string[] = []; let tt0 = 0;
  page.on("response", (r) => { if (/\/api\/v1\//.test(r.url())) net.push(`${Math.round((Date.now() - tt0) / 1000)}s ${r.request().method()} ${r.url().replace(/^.*\/api\/v1/, "").slice(0, 60)} ${r.status()}`); });
  await page.addInitScript(() => { Math.random = () => 0; }); await page.clock.install({ time: new Date() }); tt0 = Date.now(); await page.goto(`/threads?course=${X}`); await page.waitForTimeout(2500);
  await page.clock.runFor(130_000); await page.waitForTimeout(1500);
  L(`08c mạng (giây giả lập): ${net.join(" ; ")}`); await ctx.close();
});
for (const chunk of [20_000, 30_000, 65_000]) test(`08d chunk ${chunk}`, async ({ browser }) => {
  const { ctx, page } = await hbPage(browser, s8, `/threads?course=${X}`); const t0 = await page.evaluate(() => Date.now());
  for (let i = 0; i < Math.ceil(130_000 / chunk); i++) { await page.clock.runFor(chunk); await page.waitForTimeout(500); }
  L(`08d chunk ${chunk / 1000}s: ${JSON.stringify((await bt(page)).map((x) => Math.round((x - t0) / 1000)))}`); await ctx.close();
});
test("10e chi tiết", async ({ browser }) => {
  const mk = async () => { const x = await hbPage(browser, s6, `/chat?course=${X}`); const t0 = await x.page.evaluate(() => Date.now()); const ev: string[] = [];
    x.page.on("requestfailed", (r) => { if (r.url().includes("/activity/heartbeat")) ev.push("fail"); }); x.page.on("response", (r) => { if (r.url().includes("/activity/heartbeat")) ev.push(String(r.status())); });
    const step = async (sec: number) => { for (let i = 0; i < sec / 5; i++) { await x.page.keyboard.press("Shift"); await x.page.clock.runFor(5000); await x.page.waitForTimeout(150); } };
    return { ...x, t0, ev, step }; };
  const a = await mk(); await a.step(20); await a.ctx.setOffline(true); await a.step(65); const alerts = await a.page.locator("[role=alert],[data-ep-toast],[class*=toast],[data-part=offline-banner]").evaluateAll((els) => els.map((e) => `${e.tagName}[${e.getAttribute("role") ?? ""}|${e.getAttribute("data-part") ?? ""}]: ${(e as HTMLElement).innerText.slice(0, 60)}`));
  await a.ctx.setOffline(false); await a.step(70); const alerts2 = await a.page.locator("[role=alert],[data-ep-toast],[class*=toast],[data-part=offline-banner]").evaluateAll((els) => els.map((e) => `${e.tagName}: ${(e as HTMLElement).innerText.slice(0, 60)}`));
  L(`10e offline: sự kiện nhịp=${JSON.stringify(a.ev)}; phần tử alert khi offline=${JSON.stringify(alerts)}; sau online=${JSON.stringify(alerts2)}; số fetch trong trang=${JSON.stringify((await bt(a.page)).map((x) => Math.round((x - a.t0) / 1000)))}`); await a.ctx.close();
  for (const code of [403, 422]) {
    const b = await mk(); await b.page.route(/\/activity\/heartbeat/, (r) => r.fulfill({ status: code, contentType: "application/json", body: JSON.stringify({ code: code === 403 ? "FORBIDDEN" : "VALIDATION_FAILED", message: "x" }) }));
    await b.step(95); const al = await b.page.locator("[role=alert],[data-ep-toast],[class*=toast]").evaluateAll((els) => els.map((e) => (e as HTMLElement).innerText.slice(0, 60))); const dlg = await b.page.getByRole("dialog").count();
    L(`10e ${code}: sự kiện=${JSON.stringify(b.ev)} (kỳ vọng đúng 1 rồi ngừng); alert/toast=${JSON.stringify(al)}; dialog=${dlg}; số fetch=${(await bt(b.page)).length}`); await b.ctx.close();
  }
});
