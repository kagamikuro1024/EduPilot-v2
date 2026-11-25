import { expect, test } from "@playwright/test";
import { asDemo, sessionBody, type DemoRole } from "./support/session";
import { BASE_URL } from "./support/env";
import { mockMe } from "./support/meMock";
import { mockLlmApi } from "./support/staffMock";
import { gradebookApi } from "./support/gradebook-fixtures";
import { practiceApi } from "./support/practice-fixtures";

// US-PU-06 AC8: lần vẽ chữ đầu tiên là chữ của khung máy chủ (PreShell) và SAU KHI có phiên không có ứng viên LCP nào mới.
// Mạng giả 1,6 Mbps / 150 ms + CPU 4× như Lighthouse để phông chữ về muộn (phông đổi làm chữ rộng hơn ⇒ ứng viên mới — lỗi từng gặp).
// `/auth/refresh` bị giữ cho tới khi test thả: ứng viên đầu phải vẽ trước phiên.
test.describe.configure({ retries: 0 });
test.beforeEach(({}, info) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng 412"));

const CASES: Array<[string, DemoRole]> = [["/", "student"], ["/chat", "student"], ["/threads", "student"], ["/me", "student"], ["/inbox", "teacher"], ["/gradebook", "teacher"], ["/settings/llm", "admin"], ["/practice", "student"], ["/practice/history", "student"]];
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };

for (const [path, role] of CASES) {
  test(`lcp before refresh ${path}`, async ({ page, context }) => {
    test.setTimeout(60_000);
    await page.setViewportSize({ width: 412, height: 823 });
    await asDemo(context, role);
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    await page.route("**/api/v1/auth/refresh", async (route) => {
      await gate;
      await route.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(sessionBody(role === "student" ? "STUDENT" : role === "teacher" ? "TEACHER" : "ADMIN", {})) });
    });
    if (path.startsWith("/practice")) await practiceApi(page); // DEF-R10S-20: có dữ liệu thật của màn Luyện đề
    if (path === "/me") await mockMe(page); // có dữ liệu thật của màn: khối chữ nào lớn hơn h1 sau phiên sẽ hiện ra
    if (path === "/settings/llm") await mockLlmApi(page, { noProviders: true }); // trạng thái T1 (chưa có nhà cung cấp): câu giải thích dài từng là ứng viên LCP mới; không giả API thì hiện thông báo lỗi lớn hơn h1 (proposal #1)
    if (path === "/gradebook") await gradebookApi(page, { warnings: ["PENALTY_NOT_IN_SCHEME", "BONUS_NOT_IN_SCHEME"] }); // T1: có điểm cộng / trừ ghi tay — câu cảnh báo dài từng là ứng viên LCP mới (P-11-6a)
    const cdp = await context.newCDPSession(page);
    await cdp.send("Network.enable");
    await cdp.send("Network.emulateNetworkConditions", { offline: false, latency: 150, downloadThroughput: (1.6 * 1024 * 1024) / 8, uploadThroughput: (750 * 1024) / 8 });
    await cdp.send("Emulation.setCPUThrottlingRate", { rate: 4 });
    await page.addInitScript(() => {
      const w = window as unknown as { __lcp: Array<{ t: number; size: number; fromPre: boolean }> };
      w.__lcp = [];
      new PerformanceObserver((l) => l.getEntries().forEach((e) => w.__lcp.push({ t: Math.round(e.startTime), size: (e as unknown as { size: number }).size, fromPre: !!(e as unknown as { element?: Element }).element?.closest("[data-part=pre-shell]") }))).observe({ type: "largest-contentful-paint", buffered: true });
    });
    const read = () => page.evaluate(() => (window as unknown as { __lcp: Array<{ t: number; size: number; fromPre: boolean }> }).__lcp);

    await page.goto(path);
    await expect(page.locator("[data-part=pre-shell] h1")).toBeVisible();
    await page.waitForTimeout(2500); // phông chữ về hết trong lúc phiên còn bị giữ
    const before = await read();
    expect(before.length, "có ứng viên LCP trước phiên").toBeGreaterThan(0);
    expect(before.every((c) => c.fromPre), "mọi ứng viên trước phiên là chữ của PreShell").toBe(true);

    release();
    await expect(page.locator("[data-part=bottom-nav] a").first()).toBeVisible({ timeout: 30_000 }); // khung thật đã dựng
    await page.waitForTimeout(1500);
    const after = await read();
    expect(after, `không có ứng viên LCP mới sau phiên (trước: ${JSON.stringify(before)}, sau: ${JSON.stringify(after)})`).toEqual(before);
  });
}

// Route động `/assignments/<id>`: có PreShell theo tiền tố, nên chữ đầu tiên trước phiên là chữ của máy chủ (FCP không chờ phiên). Sau phiên đoạn đề bài là DỮ LIỆU của bài tập,
// lớn hơn h1 «Bài tập»: LCP của route này không đạt bằng đường (a) (US-P10-08 AC6), nên chỉ kiểm phần trước phiên.
test("lcp before refresh /assignments/<id>: ứng viên đầu là chữ PreShell", async ({ page, context }) => {
  test.setTimeout(60_000);
  await page.setViewportSize({ width: 412, height: 823 });
  await asDemo(context, "student");
  await page.route("**/api/v1/auth/refresh", () => new Promise(() => undefined)); // giữ mãi: chỉ xét trước phiên
  await page.addInitScript(() => {
    const w = window as unknown as { __lcp: Array<{ fromPre: boolean }> };
    w.__lcp = [];
    new PerformanceObserver((l) => l.getEntries().forEach((e) => w.__lcp.push({ fromPre: !!(e as unknown as { element?: Element }).element?.closest("[data-part=pre-shell]") }))).observe({ type: "largest-contentful-paint", buffered: true });
  });
  await page.goto("/assignments/00000000-0000-7000-8000-0000000a0001");
  await expect(page.locator("[data-part=pre-shell] h1")).toHaveText("Bài tập");
  await page.waitForTimeout(1500);
  const seen = await page.evaluate(() => (window as unknown as { __lcp: Array<{ fromPre: boolean }> }).__lcp);
  expect(seen.length).toBeGreaterThan(0);
  expect(seen.every((c) => c.fromPre)).toBe(true);
});

// DEF-R10S-20: `/practice/<id>` có PreShell theo tiền tố; chữ đầu tiên trước phiên là chữ máy chủ. Câu hỏi (dữ liệu) lớn hơn h1 nên LCP sau dữ liệu ghi nợ theo US-P10-08 AC6; chỉ kiểm phần trước phiên.
test("lcp before refresh /practice/<id>: ứng viên đầu là chữ PreShell", async ({ page, context }) => {
  test.setTimeout(60_000);
  await page.setViewportSize({ width: 412, height: 823 });
  await asDemo(context, "student");
  await page.route("**/api/v1/auth/refresh", () => new Promise(() => undefined));
  await page.addInitScript(() => {
    const w = window as unknown as { __lcp: Array<{ fromPre: boolean }> };
    w.__lcp = [];
    new PerformanceObserver((l) => l.getEntries().forEach((e) => w.__lcp.push({ fromPre: !!(e as unknown as { element?: Element }).element?.closest("[data-part=pre-shell]") }))).observe({ type: "largest-contentful-paint", buffered: true });
  });
  await page.goto("/practice/att-1?course=00000000-0000-7000-8000-00000000c001");
  await expect(page.locator("[data-part=pre-shell] h1")).toHaveText("Làm bài");
  await page.waitForTimeout(1500);
  const seen = await page.evaluate(() => (window as unknown as { __lcp: Array<{ fromPre: boolean }> }).__lcp);
  expect(seen.length).toBeGreaterThan(0);
  expect(seen.every((c) => c.fromPre)).toBe(true);
});

// LCP-2 (US-P10-08 AC6, TC-08-02, TC-12-17): `/assignments/<id>` và `/practice/<id>` có LCP là chữ DỮ LIỆU. Mã nội tuyến (earlyPaint.ts) gọi đúng một GET ngay khi lần làm mới sớm
// trả token và điền chữ thô vào ô PreShell, TRƯỚC khi JS nào chạy (ở đây mọi chunk JS bị chặn: chỉ HTML máy chủ + mã nội tuyến).
const EC = "00000000-0000-7000-8000-00000000c001";
for (const [path, api, body, expected] of [
  [`/assignments/00000000-0000-7000-8000-0000000a0001?course=${EC}`, `**/api/v1/courses/${EC}/assignments/00000000-0000-7000-8000-0000000a0001/mine`, { assignment: { description: "# Bài 03\n\n**Đề:** phân tích `nhật ký` máy chủ.\n\n- ý một" } }, "Bài 03 Đề: phân tích nhật ký máy chủ. • ý một"],
  [`/practice/00000000-0000-7000-8000-00000000d001?course=${EC}`, `**/api/v1/courses/${EC}/practice/attempts/00000000-0000-7000-8000-00000000d001`, { current_index: 1, items: [{ question: { stem: "câu 0" } }, { question: { stem: "Thiết bị nào chuyển khung theo MAC?" } }] }, "Thiết bị nào chuyển khung theo MAC?"],
] as const) {
  test(`early paint ${path.split("?")[0].replace(/[0-9a-f-]{36}/, "<id>")}: chữ dữ liệu hiện trong PreShell trước JS, token chỉ ở Authorization`, async ({ page, context }) => {
    await asDemo(context, "student");
    await page.route("**/_next/static/chunks/**", (r) => r.abort());
    await page.route("**/api/v1/auth/refresh", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(sessionBody("STUDENT", {})) }));
    let auth = "";
    await page.route(api, (r) => {
      auth = r.request().headers()["authorization"] ?? "";
      return r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
    });
    await page.goto(path);
    await expect(page.locator("[data-part=pre-shell] [data-part=early-body]")).toHaveText(expected);
    await expect(page.locator("[data-part=pre-shell] [data-part=early-body]")).toBeVisible();
    expect(auth).toMatch(/^Bearer .+/);
  });
}

test("early paint: không có ?course= thì ô trống ẩn (không gọi API đoán lớp)", async ({ page, context }) => {
  await asDemo(context, "student");
  await page.route("**/_next/static/chunks/**", (r) => r.abort());
  await page.route("**/api/v1/auth/refresh", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(sessionBody("STUDENT", {})) }));
  let called = 0;
  await page.route("**/api/v1/courses/**", (r) => { called++; return r.abort(); });
  await page.goto("/assignments/00000000-0000-7000-8000-0000000a0001");
  await page.waitForTimeout(800);
  await expect(page.locator("[data-part=pre-shell] [data-part=early-body]")).toBeHidden();
  expect(called).toBe(0);
});
