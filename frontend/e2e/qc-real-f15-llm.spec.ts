import { test, expect } from "@playwright/test";
import { call, login, needReal, shot, uiLogin, uniq, type Json } from "./qc-real-lib";

// QC @real (P-11-29): bật ca @real của settings-llm.spec.ts — hai tab, khoá sai, 403 khi ép bằng API.
// Khoá sai cần gateway ep-qc chạy với FAKE_LLM_VALID_KEY=<khoá đúng> và QC_FAKE_VALID_KEY cùng giá trị khi chạy spec.
needReal();
test.describe.configure({ mode: "serial" });
const tag = uniq();

test("@real 403: TEACHER / TA / STUDENT không đọc, không ghi cấu hình LLM (API) và /settings/llm là trang chặn quyền (UI)", async ({ browser, request }) => {
  for (const u of ["teacher", "ta", "sv.kha"]) {
    const tk = await login(request, u);
    for (const [m, path, body] of [["GET", "/admin/llm/providers"], ["GET", "/admin/llm/routes"], ["POST", "/admin/llm/providers", { name: "x", type: "fake", api_key: "k" }]] as const) {
      const r = await call(request, tk, m, path, body);
      expect(r.status, `${u} ${m} ${path}`).toBe(403);
    }
  }
  const anon = await call(request, null, "GET", "/admin/llm/providers");
  expect(anon.status).toBe(401);
  for (const u of ["teacher", "sv.kha"]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
    const p = await ctx.newPage();
    const calls: number[] = [];
    p.on("response", (r) => { if (r.url().includes("/api/v1/admin/llm/providers")) calls.push(r.status()); });
    await uiLogin(p, u);
    await p.goto("/settings/llm");
    await expect(p.getByText(/không có quyền|Bạn không thể mở/i).first()).toBeVisible();
    expect(calls.filter((c) => c === 200), "không có lời gọi 200 tới cấu hình LLM").toEqual([]);
    await shot(p, `f15-settings-llm-403-${u}`);
    await ctx.close();
  }
});

test("@real hai tab: cùng sửa một nhà cung cấp, tab sau nhận 409 (khoá lạc quan) và giao diện nạp lại giá trị mới", async ({ browser, request }) => {
  const admin = await login(request, "admin");
  const mk = await call(request, admin, "POST", "/admin/llm/providers", { name: `QC F15 ${tag}`, type: "fake", api_key: `qc-${tag}`, enabled: true });
  expect([200, 201], mk.text.slice(0, 200)).toContain(mk.status);
  const id = String(mk.json.id);
  const v = Number(mk.json.version ?? 1);
  const a = await call(request, admin, "PATCH", `/admin/llm/providers/${id}`, { enabled: false, version: v });
  expect(a.status, a.text.slice(0, 200)).toBe(200);
  const b = await call(request, admin, "PATCH", `/admin/llm/providers/${id}`, { enabled: true, version: v });
  expect(b.status, "tab sau dùng version cũ").toBe(409);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const p = await ctx.newPage();
  await uiLogin(p, "admin");
  await p.goto("/settings/llm");
  await expect(p.getByText(`QC F15 ${tag}`).first()).toBeVisible();
  await shot(p, "f15-settings-llm-admin");
  await ctx.close();
  const del = await call(request, admin, "DELETE", `/admin/llm/providers/${id}`);
  expect([200, 204]).toContain(del.status);
});

test("@real khoá sai: gateway từ chối khoá không hợp lệ (422 PROVIDER_AUTH_FAILED tại ô api_key), khoá đúng lưu được, khoá không lộ ở giao diện", async ({ page, request }) => {
  test.skip(!process.env.QC_FAKE_VALID_KEY, "cần gateway ep-qc có FAKE_LLM_VALID_KEY và QC_FAKE_VALID_KEY (K — chưa kiểm được nếu thiếu)");
  const admin = await login(request, "admin");
  const bad = await call(request, admin, "POST", "/admin/llm/providers", { name: `QC F15 sai ${tag}`, type: "fake", api_key: `sai-${tag}`, enabled: true });
  expect(bad.status, bad.text.slice(0, 200)).toBe(422);
  expect(JSON.stringify((bad.json as Json).details ?? [])).toMatch(/api_key/);
  expect(bad.text, "thân lỗi không lặp lại khoá").not.toContain(`sai-${tag}`);
  const good = await call(request, admin, "POST", "/admin/llm/providers", { name: `QC F15 đúng ${tag}`, type: "fake", api_key: process.env.QC_FAKE_VALID_KEY, enabled: true });
  expect([200, 201], good.text.slice(0, 200)).toContain(good.status);
  expect(good.text, "phản hồi không trả khoá").not.toContain(String(process.env.QC_FAKE_VALID_KEY));
  await uiLogin(page, "admin");
  await page.goto("/settings/llm");
  const html = await page.evaluate(() => document.body.innerHTML + JSON.stringify({ ...localStorage }) + document.cookie);
  expect(html).not.toContain(String(process.env.QC_FAKE_VALID_KEY));
  await call(request, admin, "DELETE", `/admin/llm/providers/${String(good.json.id)}`);
});
