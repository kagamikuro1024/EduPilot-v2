/* eslint-disable @typescript-eslint/no-explicit-any */
import { test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";

async function login(page: Page, email: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}
const api = (page: Page, path: string) =>
  page.evaluate(async (p) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await fetch(`/api/v1${p}`, { headers: { authorization: `Bearer ${a.access_token}` } });
    const text = await r.text();
    return { status: r.status, text, body: (() => { try { return JSON.parse(text); } catch { return null; } })() };
  }, path);
const cid = async (page: Page, code: string) => (await api(page, "/me/courses")).body.items.find((i: any) => i.course.class_code === code).course.id as string;
const fresh = async (browser: import("@playwright/test").Browser, mail: string, vp = { width: 1440, height: 900 }) => { const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: vp }); const page = await ctx.newPage(); page.setDefaultTimeout(10000); await login(page, mail); return { ctx, page }; };

test("TC-UX20-6/7 /me rỗng và lỗi", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(100000);
  const { ctx, page } = await fresh(browser, "sv.kha@edupilot.local");
  const real = await api(page, `/courses/${await cid(page, "761987")}/me/grades`);
  const g = real.body;
  const empty = { ...g, qt: null, final: null, bonus: null, penalty: null, calculation: [], components: g.components.map((c: any) => ({ key: c.key, name: c.name, weight: c.weight, status: "PENDING" })) };
  await page.route(/\/courses\/[0-9a-f-]+\/me\/grades(\?.*)?$/, (r) => r.fulfill({ json: empty }));
  await page.goto("/me"); await page.waitForTimeout(4500);
  console.log("INFO UX20-6 rỗng:", JSON.stringify((await page.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 300)));
  await page.screenshot({ path: "/out/shots-w5/me-empty-1440.png" });
  await page.unroute(/\/courses\/[0-9a-f-]+\/me\/grades(\?.*)?$/);
  await page.route(/\/courses\/[0-9a-f-]+\/me\/grades(\?.*)?$/, (r) => r.abort());
  await page.goto("/me"); await page.waitForTimeout(4500);
  console.log("INFO UX20-7 lỗi:", JSON.stringify((await page.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 300)), "| Thử lại:", await page.getByRole("button", { name: /Thử lại/ }).count());
  await ctx.close();
});

test("TC-UX13-18/22/23 chat: kéo cuộn khi stream; phân quyền", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(150000);
  const { ctx, page } = await fresh(browser, "sv.kha@edupilot.local");
  const c = await cid(page, "761987");
  const ss = await api(page, `/chat/sessions?course_id=${c}&limit=50`);
  const ids: string[] = ss.body.items.map((x: any) => x.id);
  console.log("INFO số phiên SV A:", ids.length);
  const { ctx: c2, page: p2 } = await fresh(browser, "sv.gioi@edupilot.local");
  const res: string[] = [];
  for (const id of ids.slice(0, 3)) { const r = await api(p2, `/chat/sessions/${id}/messages?limit=5`); res.push(`${r.status}${r.text.includes("\"content\"") ? " LỘ" : ""}`); }
  console.log("INFO UX13-23 SV B đọc phiên SV A:", JSON.stringify(res));
  await c2.close();
  for (const [who, mail] of [["GV", "teacher@edupilot.local"], ["TA", "ta@edupilot.local"]] as const) {
    const { ctx: c3, page: p3 } = await fresh(browser, mail);
    const r3: string[] = [];
    for (const id of ids.slice(0, 2)) { const r = await api(p3, `/chat/sessions/${id}/messages?limit=5`); r3.push(`${r.status}${r.text.includes("\"content\"") ? " LỘ" : ""}`); }
    await p3.goto("/chat"); await p3.waitForTimeout(3500);
    console.log(`INFO UX13-22 ${who}: API phiên SV`, JSON.stringify(r3), "| /chat:", JSON.stringify((await p3.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 200)));
    await c3.close();
  }
  await ctx.close();
});

test("TC-UX13-20 mất mạng giữa stream", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(120000);
  const { ctx, page } = await fresh(browser, "sv.kha@edupilot.local");
  await page.goto("/chat"); await page.waitForTimeout(3500);
  const comp = page.getByRole("textbox", { name: "Câu hỏi của bạn" });
  await comp.fill("Giải thích chi tiết quy chế điểm quá trình và cách tính điểm chuyên cần của lớp này");
  await page.getByRole("button", { name: "Gửi", exact: true }).click();
  await page.waitForTimeout(120);
  await ctx.setOffline(true);
  await page.waitForTimeout(3000);
  const banner = (await page.locator("body").innerText()).match(/[^\n]*(mất mạng|Mất kết nối|offline|Chưa có mạng)[^\n]*/i)?.[0];
  await ctx.setOffline(false);
  await page.waitForTimeout(9000);
  const last = (await page.locator("[data-part=chat-thread]").innerText()).replace(/\s+/g, " ").slice(-260);
  console.log("INFO UX13-20 dải mất mạng:", JSON.stringify(banner), "| cuối hội thoại:", JSON.stringify(last), "| Thử lại:", await page.getByRole("button", { name: "Thử lại" }).count());
  await ctx.close();
});

test("TC-UX17-8/9 admin xem prompt; lớp chưa có báo cáo", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(120000);
  const { ctx, page } = await fresh(browser, "teacher@edupilot.local");
  const c = await cid(page, "QC117283");
  await page.evaluate((id) => localStorage.setItem("ep:ui:course", id), c);
  await page.goto("/insights"); await page.waitForTimeout(4500);
  console.log("INFO UX17-9 lớp QC117283:", JSON.stringify((await page.locator("main").innerText()).replace(/\n+/g, " ⏎ ").slice(0, 300)), "| nút:", JSON.stringify(await page.locator("main button").allInnerTexts()));
  await page.screenshot({ path: "/out/shots-w5/insights-empty-qc-1440.png" });
  await ctx.close();
  const { ctx: c2, page: p2 } = await fresh(browser, "admin@edupilot.local");
  await p2.goto("/observability"); await p2.waitForTimeout(4500);
  const rows = p2.locator("main table").last().locator("tbody tr");
  console.log("INFO UX17-8 admin /observability:", JSON.stringify((await p2.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 200)), "| hàng:", await rows.count());
  if (await rows.count()) { await rows.first().click(); await p2.waitForTimeout(1200); console.log("INFO UX17-8 sau chọn hàng:", await p2.getByRole("dialog").count(), JSON.stringify((await p2.locator("body").innerText()).match(/(prompt|Prompt|Nội dung câu hỏi)[^\n]{0,60}/g)?.slice(0, 2))); }
  await c2.close();
});

test("TC-UX18-4/6/8/PQ tên công tắc, thư viện SV, quyền", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(150000);
  const { ctx, page } = await fresh(browser, "teacher@edupilot.local");
  const c = await cid(page, "761987");
  await page.evaluate((id) => localStorage.setItem("ep:ui:course", id), c);
  await page.goto("/documents"); await page.waitForTimeout(5000);
  const names = await page.getByRole("switch").evaluateAll((els) => els.slice(0, 10).map((e) => (e.getAttribute("aria-label") ?? e.getAttribute("aria-labelledby") ?? "").slice(0, 80)));
  console.log("INFO UX18-4 thuộc tính tên:", JSON.stringify(names));
  const bad = await page.getByRole("switch", { name: /Dùng cho AI: (bật|tắt)/ }).count();
  const bad2 = await page.getByRole("switch", { name: /Hiện cho sinh viên: (bật|tắt)/ }).count();
  console.log("INFO UX18-4 công tắc có tên «Dùng cho AI: bật/tắt»:", bad, "| «Hiện cho sinh viên: …»:", bad2, "| tổng:", await page.getByRole("switch").count());
  const a11y = await page.getByRole("switch").evaluateAll((els) => els.slice(0, 4).map((e) => e.getAttribute("aria-label")));
  console.log("INFO UX18-4 aria-label:", JSON.stringify(a11y));
  const sel = await page.locator("main tbody tr select").first().evaluate((s) => { const r = (s as HTMLSelectElement).getBoundingClientRect(); return { sw: s.scrollWidth, cw: s.clientWidth, w: Math.round(r.width), val: (s as HTMLSelectElement).selectedOptions[0]?.textContent, opts: [...(s as HTMLSelectElement).options].map((o) => o.textContent) }; }).catch(() => null);
  console.log("INFO UX18-1 ô Loại (select):", JSON.stringify(sel));
  await ctx.close();
  const { ctx: c2, page: p2 } = await fresh(browser, "sv.kha@edupilot.local");
  await p2.goto("/documents"); await p2.waitForTimeout(3000);
  console.log("INFO UX18-8 SV /documents:", JSON.stringify((await p2.locator("main").innerText().catch(() => "")).replace(/\s+/g, " ").slice(0, 120)));
  const cc = await cid(p2, "761987");
  console.log("INFO UX18-8 SV API documents:", (await api(p2, `/courses/${cc}/documents?limit=5`)).status);
  await p2.goto("/library"); await p2.waitForTimeout(4000);
  const lib = (await p2.locator("main").innerText()).replace(/\s+/g, " ");
  console.log("INFO UX18-6 SV /library:", lib.length, "| có «Đáp án»:", /Đáp án/.test(lib), "| có «Đề tham khảo tuần 5»:", /Đề tham khảo tuần 5/.test(lib), "|", JSON.stringify(lib.slice(0, 160)));
  await c2.close();
  const { ctx: c3, page: p3 } = await fresh(browser, "ta@edupilot.local");
  const { ctx: ca, page: pa } = await fresh(browser, "teacher@edupilot.local");
  const arch = await cid(pa, "QC196964"); await ca.close();
  console.log("INFO UX18-PQ TA /documents lớp lạ:", (await api(p3, `/courses/${arch}/documents?limit=5`)).status);
  await c3.close();
});
