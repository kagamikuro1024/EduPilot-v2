import { expect, test } from "@playwright/test";
import { needReal, uiLogin } from "./qc-real-lib";

// QC sprint 11 — U1 (làm mới phiên sớm, 168c5eb2): đúng một lần, khoá giữa hai tab, không gọi khi chưa từng đăng nhập.
needReal();
const BASE = process.env.QC_BASE ?? "https://localhost:943";

test("@real U1-1 làm mới đúng MỘT lần khi tải trang, token không ở storage", async ({ page }) => {
  await uiLogin(page, "sv.kha");
  const calls: { t: number; status?: number }[] = [];
  page.on("request", (r) => { if (r.url().endsWith("/api/v1/auth/refresh") && r.method() === "POST") calls.push({ t: Date.now() }); });
  await page.goto("/settings");
  await page.waitForTimeout(3000);
  expect(calls.length, "một lần tải trang = đúng một POST /auth/refresh").toBe(1);
  await page.reload();
  await page.waitForTimeout(3000);
  expect(calls.length, "tải lại: thêm đúng một lần").toBe(2);
  const store = await page.evaluate(() => JSON.stringify({ l: { ...localStorage }, s: { ...sessionStorage } }));
  expect(store, "không token ở storage").not.toMatch(/eyJ[\w-]{10,}\.[\w-]{10,}/);
  expect(await page.evaluate(() => Object.keys(window).filter((k) => /^__epRefresh/.test(k))), "promise làm mới sớm đã bị xoá khỏi window sau khi dùng").toEqual([]);
});

test("@real U1-2 hai tab tải cùng lúc: không chạy chồng, phiên không bị thu hồi", async ({ browser }) => {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: BASE });
  const a = await ctx.newPage();
  await uiLogin(a, "sv.kha");
  const b = await ctx.newPage();
  const statuses: number[] = [];
  for (const p of [a, b]) p.on("response", (r) => { if (r.url().endsWith("/api/v1/auth/refresh")) statuses.push(r.status()); });
  await Promise.all([a.goto("/settings"), b.goto("/settings")]);
  await a.waitForTimeout(4000);
  expect(statuses.every((s) => s === 200), `mọi lần làm mới 200 (không 401 REFRESH_REUSE): ${statuses}`).toBe(true);
  await expect(a).not.toHaveURL(/\/login/);
  await expect(b).not.toHaveURL(/\/login/);
  // vẫn đăng nhập sau đó: tải lại cả hai
  await Promise.all([a.reload(), b.reload()]);
  await a.waitForTimeout(3000);
  await expect(a).not.toHaveURL(/\/login/);
  await expect(b).not.toHaveURL(/\/login/);
  expect(statuses.length, `số lần làm mới (${statuses.length}) ≥ 2 (mỗi tab) và mọi lần thành công`).toBeGreaterThanOrEqual(2);
  await ctx.close();
});

test("@real U1-3 máy chưa từng đăng nhập: `/` không gọi làm mới; route khác gọi", async ({ browser }) => {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: BASE });
  const p = await ctx.newPage();
  const calls: string[] = [];
  p.on("request", (r) => { if (r.url().endsWith("/api/v1/auth/refresh") && r.method() === "POST") calls.push(new URL(p.url() || BASE).pathname); });
  await p.goto("/");
  await p.waitForTimeout(2500);
  expect(calls, "`/` + chưa từng đăng nhập: không POST /auth/refresh").toEqual([]);
  await p.goto("/login");
  await p.waitForTimeout(2500);
  expect(calls.length, "/login: có gọi (như cũ)").toBeGreaterThanOrEqual(1);
  await ctx.close();
});
