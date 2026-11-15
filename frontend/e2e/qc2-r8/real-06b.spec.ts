import { expect, test, type Page, type BrowserContext } from "@playwright/test";
import { cid, go, log, login, shot, IDS } from "./lib";
test.setTimeout(150_000);
const mkp = async (browser: any, user: string) => { const c: BrowserContext = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p: Page = await c.newPage(); await login(p, user); return p; };
const rowsN = (p: Page) => p.locator("[data-part=grade-row]").count();
test("[TC-P6-06-31,TC-P6-06-32,TC-P6-06-33] LOAD1000: DOM ≤ 60 hàng, cuộn tải trang kế, ô đang gõ giữ chữ, Enter xuyên cửa sổ ảo hoá", async ({ browser }) => {
  const p = await mkp(browser, "teacher"); const reqs: string[] = []; p.on("request", (q) => { if (q.url().includes("/gradebook?")) reqs.push(q.url().split("?")[1]); });
  await p.addInitScript(() => { (window as any).__lt = []; new PerformanceObserver((l) => { for (const e of l.getEntries()) (window as any).__lt.push(e.duration); }).observe({ entryTypes: ["longtask"] }); });
  await go(p, `/gradebook?course=${cid("LOAD1000")}`); await p.locator("[data-part=grade-row]").first().waitFor({ timeout: 30000 });
  const scroller = () => p.evaluate((t) => { const r = document.querySelector("[data-part=grade-row]"); let e: any = r; while (e && !(e.scrollHeight > e.clientHeight + 5 && /(auto|scroll)/.test(getComputedStyle(e).overflowY))) e = e.parentElement; return e ? { h: e.scrollHeight, c: e.clientHeight } : null; }, 0);
  const setTop = (t: number) => p.evaluate((t) => { const r = document.querySelector("[data-part=grade-row]"); let e: any = r; while (e && !(e.scrollHeight > e.clientHeight + 5 && /(auto|scroll)/.test(getComputedStyle(e).overflowY))) e = e.parentElement; if (e) e.scrollTop = t < 0 ? e.scrollHeight : t; }, t);
  let maxDom = await rowsN(p); log("06b", `LOAD1000 hàng ban đầu ${maxDom} scroller ${JSON.stringify(await scroller())}`);
  // 06-32: gõ dở ở hàng 5
  const c5 = p.locator("[data-part=grade-row]").nth(4).locator("input[data-cell]").first(); const key5 = await c5.getAttribute("data-cell"); await c5.focus(); await c5.fill("7,");
  for (let i = 0; i < 40; i++) { await setTop(-1); await p.waitForTimeout(350); maxDom = Math.max(maxDom, await rowsN(p)); }
  const bottomRows = await rowsN(p), nextReq = reqs.filter((r) => /cursor=/.test(r)).length; log("06b", `sau cuộn cuối: ${bottomRows} hàng DOM, max ${maxDom}, yêu cầu có cursor ${nextReq}`);
  await setTop(0); await p.waitForTimeout(800);
  const back = p.locator(`input[data-cell="${key5}"]`); const v5 = await back.count() ? await back.inputValue() : "(không có trong DOM)"; log("06b", `ô hàng 5 sau cuộn về: '${v5}'`);
  const lt = await p.evaluate(() => (window as any).__lt as number[]); log("06b", `long task: ${lt.length} (max ${Math.max(0, ...lt).toFixed(0)} ms)`);
  expect(maxDom).toBeLessThanOrEqual(60); expect(nextReq).toBeGreaterThan(0); expect(v5).toBe("7,");
  await back.press("Escape").catch(() => undefined);
  // 06-33: Enter xuyên cuối cửa sổ ảo hoá
  const rows = p.locator("[data-part=grade-row]"); const last = (await rows.count()) - 1; const lc = rows.nth(last).locator("input[data-cell]").first(); await lc.focus();
  const ids: string[] = []; for (let i = 0; i < 6; i++) { ids.push((await p.evaluate(() => (document.activeElement as HTMLElement | null)?.getAttribute?.("data-cell") ?? "—")) as string); await p.keyboard.press("ArrowDown"); await p.waitForTimeout(400); }
  log("06b", `ArrowDown/focus: ${JSON.stringify(ids.map((x) => x.slice(0, 8)))}`); expect(new Set(ids).size).toBeGreaterThan(2);
  const ae = await p.evaluate(() => document.activeElement?.tagName); log("06b", `focus cuối: ${ae}`); expect(ae).toBe("INPUT");
  expect(await p.locator("thead th").first().evaluate((e) => getComputedStyle(e).position)).toBe("sticky"); await shot(p, "06-load1000");
});
test("[TC-P6-06-37] lớp NO_POLICY: EmptyState + Thiết lập công thức, không cột điểm", async ({ browser }) => {
  const p = await mkp(browser, "gv.s8f"); await go(p, `/gradebook?course=${cid("QD8F3")}`); await shot(p, "06-nopolicy");
  const t = (await p.locator("main").innerText()).replace(/\s+/g, " "); log("06b", `NO_POLICY: ${t.slice(0, 300)}`);
  expect(t).toContain("Chưa có công thức điểm"); await expect(p.getByRole("link", { name: "Thiết lập công thức" })).toBeVisible(); expect(await p.locator("input[data-cell]").count()).toBe(0); expect(await p.locator("thead th").count()).toBe(0);
  expect(t).toContain("Công thức điểm chưa xác nhận");
});
test("[TC-P6-06-38] lớp có công thức, 0 sinh viên", async ({ browser }) => {
  const p = await mkp(browser, "teacher"); await go(p, `/gradebook?course=${cid("QD8R6E")}`); const t = (await p.locator("main").innerText()).replace(/\s+/g, " "); log("06b", `0 SV: ${t.slice(0, 200)}`);
  expect(t).toContain("Lớp chưa có sinh viên"); expect(await p.locator("input[data-cell]").count()).toBe(0);
});
test("[TC-P6-06-53,TC-P6-06-54] cảnh báo điểm trừ / điểm cộng ghi tay không nằm trong công thức; lớp sạch không có", async ({ browser }) => {
  const p = await mkp(browser, "teacher"); const out: string[] = [];
  for (const [c, want] of [["QD8R6P", "điểm trừ"], ["QD8R6Q", "điểm cộng"], ["QD8R6W", ""]] as const) {
    await go(p, `/gradebook?course=${cid(c)}`); await p.locator("[data-part=grade-row]").first().waitFor({ timeout: 30000 });
    const n = p.getByText(/ghi tay không được tính theo công thức/); const cnt = await n.count(); const txt = cnt ? await n.first().innerText() : ""; out.push(`${c}: ${cnt} thông báo '${txt}'`);
    if (want) { expect(cnt).toBe(1); expect(txt.toLowerCase()).toContain(want); await shot(p, `06-warn-${c}`); const link = p.getByRole("button", { name: /xem|sinh viên liên quan|Xem/i }).first(); log("06b", `nút xem liên quan: ${await link.count()}`); } else expect(cnt).toBe(0);
  }
  log("06b", out.join(" | "));
});
test("[TC-P6-06-40,TC-P6-08-37,TC-P6-08-15] sổ LOCKED: Đã chốt vN, ô khoá, Mở khoá ở menu, hộp thoại mở khoá, không toast", async ({ browser }) => {
  const p = await mkp(browser, "teacher"); await go(p, `/gradebook?course=${cid("QD8R8")}`); await p.locator("[data-part=grade-row]").first().waitFor({ timeout: 30000 });
  const t = (await p.locator("main").innerText()).replace(/\s+/g, " "); log("06b", `LOCKED: ${t.slice(0, 220)}`); await shot(p, "08-locked");
  expect(t).toMatch(/Đã chốt v3/); const ed = await p.locator("input[data-cell]:not([disabled]):not([readonly])").count(); log("06b", `ô sửa được: ${ed}`); expect(ed).toBe(0);
  expect(await p.getByRole("status").filter({ hasText: "Thành công" }).count()).toBe(0);
  expect(await p.getByText("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.").count()).toBe(1);
  await p.getByRole("button", { name: "Thêm hành động" }).click(); const items = await p.getByRole("menuitem").allInnerTexts(); log("06b", `menu LOCKED: ${JSON.stringify(items)}`); expect(items.join("|")).toContain("Mở khoá");
  await p.getByRole("menuitem", { name: "Mở khoá" }).click(); const dlg = p.getByRole("dialog"); await expect(dlg).toBeVisible(); const dt = (await dlg.innerText()).replace(/\s+/g, " "); log("06b", `hộp thoại mở khoá: ${dt}`);
  expect(dt).toContain("Mở khoá bản chốt v3 của 30 sinh viên"); expect(dt).toContain("vẫn được giữ nguyên");
  const ta = dlg.locator("textarea, input[type=text]").first(); const ok = dlg.getByRole("button", { name: /Mở khoá/ }).last();
  await ta.fill("ngắn"); log("06b", `lý do ngắn → nút disabled=${await ok.isDisabled()}`); expect(await ok.isDisabled()).toBe(true);
  await ta.fill("Lý do mở khoá đủ dài"); log("06b", `lý do đủ dài → nút disabled=${await ok.isDisabled()}`); expect(await ok.isDisabled()).toBe(false);
  await p.keyboard.press("Escape");
});
