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
const composer = (p: Page) => p.getByRole("textbox", { name: "Câu hỏi của bạn" });
const api = (page: Page, path: string) =>
  page.evaluate(async (p) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await fetch(`/api/v1${p}`, { headers: { authorization: `Bearer ${a.access_token}` } });
    const text = await r.text();
    return { status: r.status, text, body: (() => { try { return JSON.parse(text); } catch { return null; } })() };
  }, path);

test("TC-UX13-19/20/22/23 nháp 2 s, mất mạng giữa stream, phân quyền", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(240000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "sv.kha@edupilot.local");
  await page.goto("/chat");
  await page.waitForTimeout(4000);
  // 19: nháp ở phiên trống
  await composer(page).fill("nháp QC hai giây");
  await page.waitForTimeout(2400);
  await page.reload();
  await page.waitForTimeout(4000);
  console.log("INFO UX13-19 phiên trống → tải lại, nháp:", JSON.stringify(await composer(page).inputValue()));
  // 19b: nháp trong phiên có tin
  await page.getByRole("button", { name: /^Phiên trước/ }).first().click().catch(() => undefined);
  const row = page.locator("[data-part=chat-history] li").first();
  await row.locator("button, a").first().click().catch(() => undefined);
  await page.waitForTimeout(2500);
  await composer(page).fill("nháp trong phiên cũ");
  await page.waitForTimeout(2400);
  const url = page.url();
  await page.reload();
  await page.waitForTimeout(4000);
  console.log("INFO UX13-19b phiên cũ", url.slice(-30), "→ tải lại, nháp:", JSON.stringify(await composer(page).inputValue()));
  await composer(page).fill("");
  // 20: mất mạng giữa stream
  await page.goto("/chat"); await page.waitForTimeout(3500);
  await composer(page).fill("Giải thích chi tiết quy chế điểm quá trình và cách tính điểm chuyên cần của lớp này");
  await page.getByRole("button", { name: "Gửi", exact: true }).click();
  await page.waitForTimeout(150);
  await ctx.setOffline(true);
  await page.waitForTimeout(2500);
  const banner = (await page.locator("body").innerText()).match(/[^\n]*(mất mạng|Mất kết nối|offline)[^\n]*/i)?.[0];
  await ctx.setOffline(false);
  await page.waitForTimeout(9000);
  const last = (await page.locator("[data-part=chat-thread]").innerText()).replace(/\s+/g, " ").slice(-300);
  console.log("INFO UX13-20 dải mất mạng:", JSON.stringify(banner), "| sau nối lại, cuối hội thoại:", JSON.stringify(last), "| Thử lại:", await page.getByRole("button", { name: "Thử lại" }).count());
  // lấy id phiên của SV A
  const cs = await api(page, "/me/courses");
  const cid = cs.body.items.find((i: any) => i.course.class_code === "761987").course.id;
  const ss = await api(page, `/chat/sessions?course_id=${cid}&limit=50`);
  const ids: string[] = ss.body.items.map((x: any) => x.id);
  console.log("INFO số phiên SV A:", ids.length);
  const ctx2 = await browser.newContext({ ignoreHTTPSErrors: true });
  const p2 = await ctx2.newPage();
  await login(p2, "sv.gioi@edupilot.local");
  const res: string[] = [];
  for (const id of ids.slice(0, 3)) { const r = await api(p2, `/chat/sessions/${id}/messages?limit=5`); res.push(`${r.status}${r.text.includes("QC") || r.text.includes("\"content\"") ? " LỘ" : ""}`); }
  console.log("INFO UX13-23 SV B đọc phiên SV A:", JSON.stringify(res));
  for (const [who, mail] of [["GV", "teacher@edupilot.local"], ["TA", "ta@edupilot.local"]] as const) {
    const c3 = await browser.newContext({ ignoreHTTPSErrors: true }); const p3 = await c3.newPage();
    await login(p3, mail);
    const r3: string[] = [];
    for (const id of ids.slice(0, 2)) { const r = await api(p3, `/chat/sessions/${id}/messages?limit=5`); r3.push(`${r.status}${r.text.includes("\"content\"") ? " LỘ" : ""}`); }
    await p3.goto("/chat"); await p3.waitForTimeout(3500);
    console.log(`INFO UX13-22 ${who}: API phiên SV`, JSON.stringify(r3), "| /chat:", JSON.stringify((await p3.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 160)));
    await c3.close();
  }
  await ctx2.close(); await ctx.close();
});
