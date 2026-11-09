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
    return { status: r.status, body: await r.json().catch(() => null) };
  }, path);
const cid = async (page: Page, code: string) => (await api(page, "/me/courses")).body.items.find((i: any) => i.course.class_code === code).course.id as string;

test("TC-UX18 Tài liệu lớp 761987 (tài liệu của chính lớp): công tắc, lỗi + Thử lại, bấm nhanh", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(240000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "teacher@edupilot.local");
  const c = await cid(page, "761987");
  await page.evaluate((id) => localStorage.setItem("ep:ui:course", id), c);
  await page.goto("/documents");
  await page.waitForTimeout(5000);
  const rows = await page.locator("main tbody tr").evaluateAll((trs) => trs.map((tr) => { const tds = [...tr.querySelectorAll("td")]; return { title: (tds[0] as HTMLElement).innerText.replace(/\n/g, " | "), type: (tds[1] as HTMLElement).innerText.trim(), sw: (tds[1] as HTMLElement).scrollWidth + "/" + (tds[1] as HTMLElement).clientWidth }; }));
  console.log("INFO UX18 761987 hàng:", JSON.stringify(rows));
  const sws = page.getByRole("switch");
  const info1 = await sws.evaluateAll((els) => els.map((e) => ({ name: (e.getAttribute("aria-label") ?? e.getAttribute("aria-labelledby") ?? (e as HTMLElement).innerText).slice(0, 60), on: e.getAttribute("aria-checked"), dis: (e as HTMLButtonElement).disabled || e.getAttribute("aria-disabled"), bg: getComputedStyle(e).backgroundColor, op: getComputedStyle(e).opacity })));
  const named = await Promise.all((await sws.all()).slice(0, 6).map(async (s) => (await s.evaluate((e) => (e as any).ariaLabel ?? "")) || ""));
  console.log("INFO UX18-4 công tắc 761987:", JSON.stringify(info1.slice(0, 8)), "| tên truy cập:", JSON.stringify(named));
  const nameOf = async (i: number) => page.evaluate((idx) => { const e = document.querySelectorAll("[role=switch]")[idx] as HTMLElement; const lb = e.getAttribute("aria-labelledby"); return e.getAttribute("aria-label") ?? (lb ? lb.split(" ").map((x) => document.getElementById(x)?.textContent).join(" ") : (e.closest("label")?.textContent ?? "")); }, i);
  console.log("INFO UX18-4 tên:", JSON.stringify([await nameOf(0), await nameOf(1), await nameOf(2)]));
  await page.screenshot({ path: "/out/shots-w5/documents-761987-1440.png" });
  const first = page.getByRole("switch").first();
  const before = await first.getAttribute("aria-checked");
  await page.route(/\/api\/v1\/courses\/[0-9a-f-]+\/documents\/[0-9a-f-]+/, (r) => (["PATCH", "PUT", "POST"].includes(r.request().method()) ? r.abort() : r.continue()));
  await first.click({ timeout: 8000 });
  await page.waitForTimeout(2500);
  console.log("INFO UX18-7 công tắc:", before, "→", await first.getAttribute("aria-checked"), "| lỗi:", JSON.stringify((await page.getByRole("alert").allInnerTexts()).slice(0, 2)), "| Thử lại:", await page.getByRole("button", { name: "Thử lại" }).count());
  await page.unroute(/\/api\/v1\/courses\/[0-9a-f-]+\/documents\/[0-9a-f-]+/);
  if (await page.getByRole("button", { name: "Thử lại" }).count()) { await page.getByRole("button", { name: "Thử lại" }).first().click(); await page.waitForTimeout(2500); console.log("INFO UX18-7 sau Thử lại:", await first.getAttribute("aria-checked")); }
  if ((await first.getAttribute("aria-checked")) !== before) { await first.click(); await page.waitForTimeout(1500); }
  const sw2 = page.getByRole("switch").nth(1);
  const b2 = await sw2.getAttribute("aria-checked");
  for (let i = 0; i < 5; i++) await sw2.click({ delay: 10, timeout: 8000 });
  await page.waitForTimeout(3500);
  const fin = await sw2.getAttribute("aria-checked");
  await page.reload(); await page.waitForTimeout(4500);
  const after = await page.getByRole("switch").nth(1).getAttribute("aria-checked");
  console.log("INFO UX18-ID 5 lần bấm:", b2, "→ UI", fin, "→ sau tải lại", after);
  if (after !== b2) { await page.getByRole("switch").nth(1).click(); await page.waitForTimeout(1500); }
  await ctx.close();
  // 375
  const c3 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  const p3 = await c3.newPage();
  await login(p3, "teacher@edupilot.local");
  await p3.evaluate((id) => localStorage.setItem("ep:ui:course", id), c);
  await p3.goto("/documents"); await p3.waitForTimeout(5000);
  console.log("INFO UX18-5/9 375 761987: tràn", await p3.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth), "| công tắc (hộp công tắc / nhãn bọc):", JSON.stringify(await p3.getByRole("switch").evaluateAll((els) => els.slice(0, 4).map((e) => { const l = e.closest("label") ?? e; const r = l.getBoundingClientRect(); const s = e.getBoundingClientRect(); return `sw ${Math.round(s.width)}x${Math.round(s.height)} / bọc ${Math.round(r.width)}x${Math.round(r.height)}`; }))));
  await p3.screenshot({ path: "/out/shots-w5/documents-761987-375.png", fullPage: true });
  await c3.close();
});

test("TC-UX19-7 bảng bộ lọc 375: trượt từ đáy", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  const page = await ctx.newPage();
  await login(page, "sv.kha@edupilot.local");
  await page.goto("/threads"); await page.waitForTimeout(4500);
  await page.getByRole("button", { name: /^Bộ lọc/ }).first().tap();
  await page.waitForTimeout(900);
  const box = await page.evaluate(() => { const d = document.querySelector("dialog[open], [role=dialog]") as HTMLElement; const out: unknown[] = []; let e: HTMLElement | null = d; for (let i = 0; i < 3 && e; i++) { const r = e.getBoundingClientRect(); out.push({ tag: e.tagName, cls: e.className.toString().slice(0, 30), y: Math.round(r.y), b: Math.round(r.bottom), h: Math.round(r.height), radius: getComputedStyle(e).borderTopLeftRadius }); e = e.firstElementChild as HTMLElement | null; } return out; });
  console.log("INFO UX19-7 khung:", JSON.stringify(box));
  await page.screenshot({ path: "/out/shots-w5/threads-sheet-375.png" });
  await ctx.close();
});
