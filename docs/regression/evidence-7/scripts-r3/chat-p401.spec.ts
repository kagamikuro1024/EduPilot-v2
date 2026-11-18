import { test } from "@playwright/test";
import { login, go, FX, shot, log, psql, newUser } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("p401", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const msgInfo = (page: any, marker: string) => page.evaluate((m: string) => {
  const els = Array.from(document.querySelectorAll("main *")).filter((e) => e.children.length === 0 && (e.textContent ?? "").includes(m));
  if (!els.length) return null;
  let node: Element = els[0];
  const all = Array.from(document.querySelectorAll("main *")).filter((e) => e.children.length === 0 && /QCUI-/.test(e.textContent ?? "")).map((e) => (e.textContent ?? "").match(/QCUI-[\w-]+/)![0]);
  while (node.parentElement) { const p = node.parentElement; const c = all.filter((x) => (p as HTMLElement).innerText.includes(x)).length; if (c > 1) break; node = p; }
  return { ask: node.querySelectorAll("[data-part=ask-staff]").length, status: Array.from(node.querySelectorAll("[data-part=ticket-status]")).map((e) => (e as HTMLElement).innerText), text: (node as HTMLElement).innerText.replace(/\n+/g, " | ").slice(0, 200) };
}, marker);
test("18-20", async ({ browser }) => {
  const f1 = FX("fx_ui1"), f2 = FX("fx_ui2");
  let { ctx, page } = await newUser(browser, "qc3s26"); await go(page, `/chat?session=${f1.sid}`);
  const res: any = {};
  for (const [k, m] of [["normal", "QCUI-NORMAL"], ["crisis", "QCUI-CRISIS"], ["other", "QCUI-OTHER"], ["small", "QCUI-SMALL"], ["failed", "QCUI-FAILED"], ["cancelled", "QCUI-CANCELLED"], ["streaming", "QCUI-STREAMING"], ["auto", "QCUI-AUTO"]]) res[k] = await msgInfo(page, m);
  await shot(page, "p401-18");
  const ok18 = res.normal?.ask === 1 && ["crisis", "other", "small", "failed", "cancelled", "streaming"].every((k) => res[k]?.ask === 0);
  R("TC-P4-01-18", ok18, `nút «Nhờ giảng viên hỗ trợ» dưới từng tin (1280×800): ${JSON.stringify(Object.fromEntries(Object.entries(res).map(([k, v]: any) => [k, v?.ask])))} (mong normal=1, còn lại 0)`);
  // 20: tooltip
  await ctx.close(); ({ ctx, page } = await newUser(browser, "qc3s26")); await go(page, `/chat?session=${f1.sid}`);
  const btn = page.locator("[data-part=ask-staff]").first();
  const title = await btn.getAttribute("title"); await btn.hover(); await page.waitForTimeout(800);
  const near = await btn.evaluate((b: HTMLElement) => (b.parentElement as HTMLElement).innerText.replace(/\n+/g, " | "));
  R("TC-P4-01-20", title === "Giảng viên sẽ đọc câu hỏi này và câu trả lời của AI" && !/mô tả|giải thích/i.test(near), `tooltip (title)=«${title}»; chữ quanh nút: «${near}» (không câu mô tả khác)`);
});
test("21", async ({ page }) => {
  const f = FX("fx_ui2").t21; await login(page, f.u); await go(page, `/chat?session=${f.sid}`);
  const b = page.locator("[data-part=ask-staff]").first();
  const dis = await b.isDisabled(); const title = await b.getAttribute("title"); await shot(page, "p401-21");
  R("TC-P4-01-21", (await b.count()) === 1 && dis && /5 câu hỏi chờ giảng viên/.test(title ?? ""), `SV có 5 ticket mở, tin AI thứ 6: nút hiện=${await b.count()}, disabled=${dis}, tooltip=«${title}»`);
});
test("22", async ({ page }) => {
  const f = FX("fx_ui2").t22; await login(page, f.u); await go(page, `/chat?session=${f.sid}`);
  let posts = 0; page.on("request", (r) => { if (r.method() === "POST" && /\/escalate$/.test(r.url())) posts++; });
  const b = page.locator("[data-part=ask-staff]").first();
  await page.evaluate(() => { (window as any).__t0 = 0; const obs = new MutationObserver(() => { if (document.querySelector("[data-part=ticket-status]") && !(window as any).__t1) (window as any).__t1 = performance.now(); }); obs.observe(document.body, { childList: true, subtree: true, characterData: true }); });
  await page.evaluate(() => { const bt = document.querySelector("[data-part=ask-staff]") as HTMLElement; (window as any).__t0 = performance.now(); bt.click(); bt.click(); bt.click(); });
  await page.waitForTimeout(2500);
  const d = await page.evaluate(() => (window as any).__t1 - (window as any).__t0);
  const nt = psql(`select count(*) from escalation_tickets where message_id='${f.am}'`);
  const st = await page.locator("[data-part=ticket-status]").first().innerText();
  R("TC-P4-01-22", posts === 1 && d <= 100 && nt === "1", `3 lần click liên tiếp (script, 0 ms giữa các lần): POST /escalate=${posts}; dòng trạng thái hiện sau ${d.toFixed(0)} ms (≤100); «${st}»; DB ticket=${nt}`);
});
test("23", async ({ page, context }) => {
  const f = FX("fx_ui2").t23; await login(page, f.u); await go(page, `/chat?session=${f.sid}`);
  await context.setOffline(true); await page.locator("[data-part=ask-staff]").first().click(); await page.waitForTimeout(1500);
  const off = await page.locator("[data-part=ticket-status]").first().innerText().catch(() => "(không có)");
  const n0 = psql(`select count(*) from escalation_tickets where message_id='${f.am}'`);
  await context.setOffline(false); await page.waitForTimeout(5000);
  const n1 = psql(`select count(*) from escalation_tickets where message_id='${f.am}'`);
  const on = await page.locator("[data-part=ticket-status]").first().innerText().catch(() => "(không có)");
  R("TC-P4-01-23", /Đang gửi/.test(off) && n0 === "0" && n1 === "1", `offline→bấm: «${off}» (DB ${n0} ticket); online sau 5 s: DB ${n1} ticket, giao diện «${on}»`);
});
test("24", async ({ page }) => {
  const f = FX("fx_ui2").t24; await login(page, f.u); await go(page, `/chat?session=${f.sid}`);
  await page.route("**/escalate", (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }));
  await page.locator("[data-part=ask-staff]").first().click(); await page.waitForTimeout(1500);
  const btn = await page.locator("[data-part=ask-staff]").count(); const txt = await page.locator("main").innerText();
  const retry = page.getByRole("button", { name: "Thử lại" });
  const n = await page.getByText("Chưa gửi được. Thử lại").count();
  await shot(page, "p401-24");
  R("TC-P4-01-24", btn === 1 && n >= 1 && (await retry.count()) >= 1, `chặn route 500: nút hoàn lại=${btn}, InlineNotice «Chưa gửi được. Thử lại»=${n}, nút «Thử lại»=${await retry.count()}`);
});
test("E1", async ({ browser }) => {
  const f = FX("fx_ui2").t25; const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: "https://localhost:1143", viewport: { width: 1280, height: 800 }, locale: "vi-VN" });
  const p1 = await ctx.newPage(), p2 = await ctx.newPage();
  await login(p1, f.u); await go(p1, `/chat?session=${f.sid}`); await go(p2, `/chat?session=${f.sid}`);
  await Promise.all([p1.locator("[data-part=ask-staff]").first().click(), p2.locator("[data-part=ask-staff]").first().click()]);
  await p1.waitForTimeout(3000);
  const n = psql(`select count(*) from escalation_tickets where message_id='${f.am}'`);
  await p1.reload(); await p2.reload(); await p1.waitForTimeout(2500);
  const s1 = await p1.locator("[data-part=ticket-status]").count(), s2 = await p2.locator("[data-part=ticket-status]").count();
  R("TC-P4-01-E1", n === "1" && s1 === 1 && s2 === 1, `hai tab cùng phiên bấm đồng thời: ticket=${n}; sau làm mới dòng trạng thái tab1=${s1}, tab2=${s2}`);
  await ctx.close();
});

test("19", async ({ browser }) => {
  const f = FX("fx_ui2").t19; const { ctx, page } = await newUser(browser, f.u); await go(page, `/chat?session=${f.sid}`);
  const r19 = await msgInfo(page, "QCUI-t19");
  R("TC-P4-01-19", r19?.ask === 0 && r19?.status.length === 1 && r19.status[0].trim() === "Đang chờ giảng viên · vừa gửi", `tin low_confidence đã tự mở ticket (ticket tạo ~20 s trước): nút=${r19?.ask}, dòng trạng thái=${JSON.stringify(r19?.status)}`);
  await ctx.close();
});
