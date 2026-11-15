import { expect, type Browser, type BrowserContext, type Page } from "@playwright/test";
export const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
export async function login(page: Page, user: string) {
  for (let i = 0; i < 3; i++) {
    await page.goto("/login");
    await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
    await page.locator("input[type=password]").fill(PW);
    await page.keyboard.press("Enter");
    try { await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 40_000 }); await page.locator("[data-part=topbar]").waitFor({ timeout: 20_000 }); return; } catch { await page.waitForTimeout(3000); }
  }
  throw new Error("đăng nhập thất bại 3 lần: " + user);
}
export async function api<T = any>(page: Page, path: string, init?: { method?: string; body?: unknown; headers?: Record<string, string> }): Promise<{ status: number; body: T }> {
  return page.evaluate(async ({ path, init }) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const h: Record<string, string> = { authorization: `Bearer ${a.access_token}`, ...(init?.headers ?? {}) };
    if (init?.body !== undefined) { h["content-type"] = "application/json"; h["Idempotency-Key"] = crypto.randomUUID(); }
    const r = await fetch(`/api/v1${path}`, { method: init?.method ?? "GET", headers: h, body: init?.body !== undefined ? JSON.stringify(init.body) : undefined });
    let body: any = null; try { body = await r.json(); } catch { /* rỗng */ }
    return { status: r.status, body };
  }, { path, init });
}
export async function courseId(page: Page, code: string): Promise<string> {
  const r = await api(page, "/me/courses?limit=100");
  return r.body.items.find((i: any) => i.course.class_code === code).course.id as string;
}
export async function open(browser: Browser, user: string, w = 1440, h = 900): Promise<{ ctx: BrowserContext; page: Page; errs: string[] }> {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, hasTouch: w < 500, isMobile: w < 500, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  const page = await ctx.newPage();
  const errs: string[] = [];
  page.on("pageerror", (e) => errs.push(e.message));
  await login(page, user);
  return { ctx, page, errs };
}
export async function go(page: Page, url: string, course?: string, settle = 1500) {
  await page.goto(course ? `${url}${url.includes("?") ? "&" : "?"}course=${course}` : url);
  await page.locator("main").first().waitFor({ state: "attached" });
  await page.waitForFunction(() => (document.querySelector("main")?.innerText ?? "").trim().length > 20, undefined, { timeout: 25_000 }).catch(() => undefined);
  await page.waitForTimeout(settle);
}
export const mainText = (page: Page) => page.locator("main").innerText().then((t) => t.replace(/\s+/g, " "));
export const noOverflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth);
export const rect = (l: ReturnType<Page["locator"]>) => l.evaluate((e) => { const r = e.getBoundingClientRect(); return { x: r.x, y: r.y, w: r.width, h: r.height, r: r.right, b: r.bottom }; });
export { expect };
