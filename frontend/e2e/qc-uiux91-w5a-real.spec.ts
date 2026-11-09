/* eslint-disable @typescript-eslint/no-explicit-any */
import { test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
const BAD = /\b(RAG|PII|provider|fallback|trace|confidence|token|LLM|embedding|prompt|model)\b/i;

async function login(page: Page, email: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}
const composer = (p: Page) => p.getByRole("textbox", { name: "Câu hỏi của bạn" });
const send = async (p: Page, text: string) => {
  await composer(p).fill(text);
  await p.getByRole("button", { name: "Gửi", exact: true }).click();
};
const rect = (l: ReturnType<Page["locator"]>) => l.evaluate((e) => { const r = e.getBoundingClientRect(); const c = getComputedStyle(e); return { x: r.x, y: r.y, w: r.width, h: r.height, r: r.right, b: r.bottom, radius: c.borderTopLeftRadius, shadow: c.boxShadow, bg: c.backgroundColor, bl: c.borderLeftWidth, br: c.borderRightWidth, bt: c.borderTopWidth, bb: c.borderBottomWidth, ff: e.tagName }; });

test("TC-UX13 chat 1440: bố cục, lịch sử, gợi ý, Phiên mới, xoá có hoàn tác, PII", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(240000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  const reqs: string[] = [];
  page.on("request", (r) => { if (/\/chat\//.test(r.url()) && r.method() !== "GET") reqs.push(`${r.method()} ${new URL(r.url()).pathname.replace(/[0-9a-f-]{36}/g, "‹id›")}`); });
  await login(page, "sv.kha@edupilot.local");
  await page.goto("/chat");
  await page.waitForTimeout(4500);

  // --- phiên trống
  console.log("INFO UX13-9 nút «Phiên mới» khi phiên trống:", await page.getByRole("button", { name: "Phiên mới" }).count());
  const sug = page.locator("[data-part=chat-suggestions] li");
  const sugTexts = await sug.allInnerTexts();
  console.log("INFO UX13-11 gợi ý:", JSON.stringify(sugTexts));
  const sugStyle = await page.locator("[data-part=chat-suggestions] li").first().evaluate((e) => { const b = (e.querySelector("button") ?? e) as HTMLElement; const c = getComputedStyle(b); return { tag: b.tagName, radius: c.borderTopLeftRadius, bg: c.backgroundColor, border: c.borderTopWidth + " " + c.borderBottomWidth, h: Math.round(b.getBoundingClientRect().height) }; });
  console.log("INFO UX13-11 kiểu gợi ý:", JSON.stringify(sugStyle));
  const intro = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  console.log("INFO UX13-17 có «chỉ bạn đọc được» trong main:", /chỉ bạn đọc được/.test(intro), "| body:", /chỉ bạn đọc được/.test(await page.locator("body").innerText()));
  console.log("INFO UX13-14 từ kỹ thuật 1440:", JSON.stringify(intro.match(BAD)));
  const inner = await rect(page.locator("[data-part=chat-inner]"));
  const main = await rect(page.locator("main"));
  const hist = await rect(page.locator("[data-part=chat-history]"));
  const panel = await page.locator("[data-part=chat-inner]").evaluate((e) => { let n: HTMLElement | null = e.parentElement; const out: unknown[] = []; while (n && n.tagName !== "MAIN") { const r = n.getBoundingClientRect(); const c = getComputedStyle(n); out.push({ cls: n.className.toString().slice(0, 40), w: Math.round(r.width), x: Math.round(r.x), radius: c.borderTopLeftRadius, bw: c.borderTopWidth, shadow: c.boxShadow === "none" ? "none" : "shadow" }); n = n.parentElement; } return out; });
  console.log("INFO UX13-1 inner", JSON.stringify(inner), "| main", JSON.stringify({ x: main.x, r: main.r }), "| history", JSON.stringify(hist), "| tổ tiên", JSON.stringify(panel));
  const composerBox = await rect(page.locator("[data-part=chat-composer]"));
  const lastSug = await rect(sug.last());
  console.log("INFO UX13-13 gap trên ô nhập", Math.round(composerBox.y - lastSug.b), "/ khung", Math.round(inner.h), "=", ((composerBox.y - lastSug.b) / inner.h).toFixed(2));
  const histRows = page.locator("[data-part=chat-history] li");
  console.log("INFO UX13-4 hàng lịch sử:", await histRows.count(), JSON.stringify(await histRows.allInnerTexts()));
  console.log("INFO UX13-4 nút trên hàng:", JSON.stringify(await histRows.first().locator("button").evaluateAll((bs) => bs.map((b) => `${b.getAttribute("aria-label")}|${Math.round(b.getBoundingClientRect().width)}x${Math.round(b.getBoundingClientRect().height)}`))));

  // --- bấm gợi ý: điền, không gửi
  const before = reqs.length;
  await sug.nth(1).locator("button, a, span").first().click().catch(async () => sug.nth(1).click());
  await page.waitForTimeout(600);
  console.log("INFO UX13-12 ô nhập sau bấm gợi ý:", JSON.stringify(await composer(page).inputValue()), "| POST mới:", reqs.length - before);
  await composer(page).fill("");

  // --- thu gọn lịch sử, CLS
  await page.evaluate(() => { (window as any).__cls = 0; new PerformanceObserver((l) => { for (const e of l.getEntries() as any[]) if (!e.hadRecentInput) (window as any).__cls += e.value; }).observe({ type: "layout-shift", buffered: false }); });
  const wBefore = (await rect(page.locator("[data-part=chat-inner]"))).w;
  await page.getByRole("button", { name: "Thu gọn lịch sử" }).click();
  await page.waitForTimeout(900);
  const wAfter = (await rect(page.locator("[data-part=chat-inner]"))).w;
  console.log("INFO UX13-3 thu gọn: rộng", Math.round(wBefore), "→", Math.round(wAfter), "| CLS", await page.evaluate(() => (window as any).__cls), "| nút mở lại:", await page.getByRole("button", { name: /^Phiên trước/ }).count());
  await page.getByRole("button", { name: /^Phiên trước/ }).first().click().catch(() => undefined);
  await page.waitForTimeout(500);

  // --- gửi 1 tin → Phiên mới
  await send(page, "Điểm quá trình của tôi hiện là bao nhiêu?");
  await page.waitForTimeout(6000);
  const pm = page.getByRole("button", { name: "Phiên mới" });
  console.log("INFO UX13-10 «Phiên mới» sau 1 tin:", await pm.count(), (await pm.count()) ? JSON.stringify(await rect(pm.first())) : "");

  // --- PII
  await pm.first().click();
  await page.waitForTimeout(800);
  await send(page, "Em tên Nguyễn Minh Trung, MSSV 20229002, email sv.kha@edupilot.local, hỏi điểm giữa kỳ");
  await page.waitForTimeout(8000);
  const txt = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  console.log("INFO UX13-15 dòng che:", JSON.stringify(txt.match(/Đã ẩn \d+ thông tin cá nhân trước khi gửi cho AI/)), "| lộ tên/MSSV/email trong màn:", /Nguyễn Minh Trung|20229002|sv\.kha@/.test(txt));
  console.log("INFO UX13-14 từ kỹ thuật sau gửi:", JSON.stringify(txt.match(BAD)));
  await page.screenshot({ path: "/out/shots-w5/chat-after-send-1440.png" });

  // --- xoá có hoàn tác (phiên PII vừa tạo là hàng đầu)
  await page.waitForTimeout(500);
  const rows = page.locator("[data-part=chat-history] li");
  const n0 = await rows.count();
  const t0 = (await rows.first().innerText()).replace(/\s+/g, " ");
  await rows.first().hover();
  await rows.first().getByRole("button", { name: /Thao tác với phiên/ }).click();
  const dlgBefore = await page.getByRole("dialog").count();
  await page.getByRole("menuitem", { name: "Xoá" }).click();
  await page.waitForTimeout(400);
  const n1 = await rows.count();
  const undo = page.getByRole("button", { name: "Hoàn tác" });
  console.log("INFO UX13-6 hàng", n0, "→", n1, "| hộp thoại:", dlgBefore, await page.getByRole("dialog").count(), "| Hoàn tác:", await undo.count(), "| dòng:", JSON.stringify((await page.locator("main").innerText()).match(/Đã xoá[^\n]*/)));
  const dels0 = reqs.filter((x) => x.startsWith("DELETE")).length;
  await undo.first().click();
  await page.waitForTimeout(1500);
  console.log("INFO UX13-7 sau Hoàn tác: hàng", await rows.count(), "| DELETE:", reqs.filter((x) => x.startsWith("DELETE")).length - dels0, "| hàng đầu giữ nguyên:", (await rows.first().innerText()).replace(/\s+/g, " ") === t0);
  await rows.first().getByRole("button", { name: /Thao tác với phiên/ }).click();
  await page.getByRole("menuitem", { name: "Xoá" }).click();
  const tDel = Date.now();
  let delAt = -1;
  for (let i = 0; i < 40 && delAt < 0; i++) { await page.waitForTimeout(250); if (reqs.filter((x) => x.startsWith("DELETE")).length - dels0 > 0) delAt = Date.now() - tDel; }
  console.log("INFO UX13-8 DELETE sau", delAt, "ms");
  await page.reload();
  await page.waitForTimeout(4000);
  console.log("INFO UX13-8 sau tải lại: hàng", await page.locator("[data-part=chat-history] li").count(), "(trước", n0, ") | phiên đã xoá còn:", (await page.locator("[data-part=chat-history]").innerText()).includes(t0.split(" ")[0]));
  console.log("INFO reqs", JSON.stringify(reqs));
  await ctx.close();
});

test("TC-UX13-18/21/22/23 chat: gõ khi đang trả lời, gửi lại cùng Idempotency-Key, phân quyền", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(240000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "sv.kha@edupilot.local");
  await page.goto("/chat");
  await page.waitForTimeout(4000);
  // 18: gõ tiếp khi đang trả lời
  await send(page, "Tuần này tôi có những buổi học nào?");
  let stopSeen = 0, typed = false;
  for (let i = 0; i < 30; i++) {
    stopSeen = Math.max(stopSeen, await page.getByRole("button", { name: "Dừng" }).count());
    if (!typed) { await composer(page).fill("tin nháp gõ trong lúc AI trả lời"); typed = (await composer(page).inputValue()).length > 0; }
    await page.waitForTimeout(100);
  }
  console.log("INFO UX13-18 nút Dừng từng hiện:", stopSeen, "| gõ được:", typed);
  await page.waitForTimeout(5000);
  // 19: nháp tự lưu 2 s
  await composer(page).fill("nháp QC 2 giây");
  await page.waitForTimeout(2300);
  await page.reload();
  await page.waitForTimeout(3500);
  console.log("INFO UX13-19 sau tải lại, nháp:", JSON.stringify(await composer(page).inputValue()));
  await composer(page).fill("");
  // 21: chặn POST → gửi → bỏ chặn → Gửi lại
  const keys: string[] = []; const posts: string[] = [];
  page.on("request", (r) => { if (r.method() === "POST" && /\/chat\/sessions\/[0-9a-f-]+\/messages$/.test(r.url())) { keys.push(r.headers()["idempotency-key"] ?? "∅"); posts.push(r.url()); } });
  const blocker = (route: import("@playwright/test").Route) => route.abort();
  await page.route(/\/chat\/sessions(\/[0-9a-f-]+)?\/messages$/, (r) => (r.request().method() === "POST" ? blocker(r) : r.continue()));
  await send(page, "Tin QC kiểm gửi lại");
  await page.waitForTimeout(2500);
  const kept = await composer(page).inputValue();
  const retryBtn = page.getByRole("button", { name: "Gửi lại" });
  console.log("INFO UX13-21 sau lỗi: ô nhập", JSON.stringify(kept), "| Gửi lại:", await retryBtn.count(), "| alert:", JSON.stringify(await page.getByRole("alert").allInnerTexts()));
  await page.unroute(/\/chat\/sessions(\/[0-9a-f-]+)?\/messages$/);
  if (await retryBtn.count()) { await retryBtn.first().click(); await page.waitForTimeout(8000); }
  console.log("INFO UX13-21 khoá idempotency:", JSON.stringify(keys), "| cùng khoá:", new Set(keys.filter((k) => k !== "∅")).size === 1 && keys.length >= 2);
  const cnt = await page.evaluate(async () => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const h = { authorization: `Bearer ${a.access_token}` };
    const m = await (await fetch("/api/v1/me/courses", { headers: h })).json();
    const c = m.items.find((i: any) => i.course.class_code === "761987").course.id;
    const s = await (await fetch(`/api/v1/chat/sessions?course_id=${c}&limit=50`, { headers: h })).json();
    let n = 0; const ids: string[] = [];
    for (const x of s.items) { const ms = await (await fetch(`/api/v1/chat/sessions/${x.id}/messages?limit=100`, { headers: h })).json(); n += ms.items.filter((q: any) => q.role === "USER" && /Tin QC kiểm gửi lại/.test(q.content ?? "")).length; ids.push(x.id); }
    return { n, ids, token: a.access_token };
  });
  console.log("INFO UX13-21 số tin «Tin QC kiểm gửi lại» đã ghi:", cnt.n);
  // 23: SV khác gọi phiên của SV A
  const ctx2 = await browser.newContext({ ignoreHTTPSErrors: true });
  const p2 = await ctx2.newPage();
  await login(p2, "sv.gioi@edupilot.local");
  const res = await p2.evaluate(async (ids) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const h = { authorization: `Bearer ${a.access_token}` };
    const out: string[] = [];
    for (const id of ids.slice(0, 3)) { const r = await fetch(`/api/v1/chat/sessions/${id}/messages?limit=5`, { headers: h }); out.push(`${r.status}:${(await r.text()).includes("Tin QC") || (await r.clone().text()).includes("content") && r.status === 200 ? "LỘ" : "ok"}`); }
    return out;
  }, cnt.ids);
  console.log("INFO UX13-23 SV khác đọc phiên của SV A:", JSON.stringify(res));
  // 22: giảng viên / TA
  for (const [who, mail] of [["GV", "teacher@edupilot.local"], ["TA", "ta@edupilot.local"]] as const) {
    const c3 = await browser.newContext({ ignoreHTTPSErrors: true }); const p3 = await c3.newPage();
    await login(p3, mail);
    const r3 = await p3.evaluate(async (ids) => {
      const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
      const h = { authorization: `Bearer ${a.access_token}` };
      const out: string[] = [];
      for (const id of ids.slice(0, 2)) { const r = await fetch(`/api/v1/chat/sessions/${id}/messages?limit=5`, { headers: h }); out.push(String(r.status)); }
      return out;
    }, cnt.ids);
    await p3.goto("/chat"); await p3.waitForTimeout(3500);
    console.log(`INFO UX13-22 ${who}: API phiên SV`, JSON.stringify(r3), "| /chat:", (await p3.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 200));
    await c3.close();
  }
  await ctx2.close(); await ctx.close();
});

test("TC-UX13-24/14/25 chat 375: bảng phiên, ô nhập, vùng chạm, hoạt cảnh", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(180000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  const page = await ctx.newPage();
  await login(page, "sv.kha@edupilot.local");
  await page.goto("/chat");
  await page.waitForTimeout(4500);
  const sw = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth }));
  const btn = page.getByRole("button", { name: /^Phiên trước/ });
  console.log("INFO UX13-24 tràn ngang:", JSON.stringify(sw), "| nút Phiên trước:", await btn.count(), JSON.stringify(await btn.first().boundingBox()));
  const comp = await composer(page).boundingBox();
  const nav = await page.locator("nav").last().boundingBox().catch(() => null);
  console.log("INFO UX13-24 ô nhập", JSON.stringify(comp), "| thanh dưới", JSON.stringify(nav));
  const txt = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  console.log("INFO UX13-14 từ kỹ thuật 375:", JSON.stringify(txt.match(BAD)), "| «chỉ bạn đọc»:", /chỉ bạn đọc được/.test(txt));
  // mô phỏng bàn phím ảo: thu nhỏ visualViewport không giả lập được trong Playwright; đo vị trí ô nhập so với đáy khung nhìn khi thu viewport
  await page.setViewportSize({ width: 375, height: 480 });
  await page.waitForTimeout(600);
  console.log("INFO UX13-24 khi khung nhìn thấp 480: ô nhập", JSON.stringify(await composer(page).boundingBox()));
  await page.setViewportSize({ width: 375, height: 812 });
  await page.waitForTimeout(400);
  await btn.first().tap();
  await page.waitForTimeout(700);
  const dlg = page.getByRole("dialog");
  const db = await dlg.first().boundingBox().catch(() => null);
  console.log("INFO UX13-24 bảng phiên:", await dlg.count(), JSON.stringify(db), "| hàng:", await dlg.first().locator("button, a, [role=option], li").count());
  const hits = await dlg.first().locator("button, a, [role=menuitem], [role=option]").evaluateAll((es) => es.slice(0, 12).map((e) => `${(e as HTMLElement).innerText.trim().slice(0, 18)}:${Math.round(e.getBoundingClientRect().width)}x${Math.round(e.getBoundingClientRect().height)}`));
  console.log("INFO UX13-24 vùng chạm trong bảng:", JSON.stringify(hits));
  await page.screenshot({ path: "/out/shots-w5/chat-sheet-375.png" });
  await page.keyboard.press("Escape");
  await ctx.close();
});

test("TC-UX13-25/26 hoạt cảnh tin mới và reduce", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(120000);
  for (const reduce of [false, true]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, reducedMotion: reduce ? "reduce" : "no-preference" });
    const page = await ctx.newPage();
    await login(page, "sv.kha@edupilot.local");
    await page.goto("/chat");
    await page.waitForTimeout(4000);
    await page.evaluate(() => {
      (window as any).__anims = [];
      const mo = new MutationObserver((ms) => { for (const m of ms) for (const n of m.addedNodes) if (n instanceof HTMLElement) requestAnimationFrame(() => { const a = [n, ...n.querySelectorAll("*")].flatMap((e) => (e as HTMLElement).getAnimations?.() ?? []); for (const x of a) (window as any).__anims.push({ name: (x as any).animationName ?? "transition", prop: (x as any).transitionProperty ?? "", dur: (x.effect?.getTiming().duration as number) ?? 0, kf: (x.effect as KeyframeEffect)?.getKeyframes?.().map((k) => Object.keys(k).filter((q) => !["offset", "easing", "composite", "computedOffset"].includes(q)).join("+")).join(" | ") ?? "" }); }); });
      mo.observe(document.querySelector("[data-part=chat-thread]")!, { childList: true, subtree: true });
    });
    await send(page, "Điểm quá trình của tôi hiện là bao nhiêu?");
    await page.waitForTimeout(7000);
    const a = await page.evaluate(() => (window as any).__anims);
    const act = await page.getByRole("button", { name: "Gửi", exact: true }).evaluate(() => { const rule = [...document.styleSheets].flatMap((s) => { try { return [...s.cssRules]; } catch { return []; } }).filter((r) => r.cssText.includes(":active") && r.cssText.includes("scale")).length; return rule; });
    console.log(`INFO UX13-${reduce ? 26 : 25} hoạt cảnh (reduce=${reduce}):`, JSON.stringify(a.slice(0, 8)), "| số luật :active có scale:", act);
    await ctx.close();
  }
});
