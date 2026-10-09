import { expect, test } from "@playwright/test";
import { asDemo, sessionBody, type DemoRole } from "./support/session";
import { BASE_URL } from "./support/env";

// US-PU-06 AC8: lần vẽ chữ đầu tiên là chữ của khung máy chủ (PreShell) và SAU KHI có phiên không có ứng viên LCP nào mới.
// Mạng giả 1,6 Mbps / 150 ms + CPU 4× như Lighthouse để phông chữ về muộn (phông đổi làm chữ rộng hơn ⇒ ứng viên mới — lỗi từng gặp).
// `/auth/refresh` bị giữ cho tới khi test thả: ứng viên đầu phải vẽ trước phiên.
test.describe.configure({ retries: 0 });
test.beforeEach(({}, info) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng 412"));

const CASES: Array<[string, DemoRole]> = [["/", "student"], ["/chat", "student"], ["/threads", "student"], ["/inbox", "teacher"], ["/gradebook", "teacher"], ["/settings/llm", "admin"]];
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
