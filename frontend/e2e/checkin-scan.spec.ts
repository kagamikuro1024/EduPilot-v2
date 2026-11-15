import { expect, test } from "@playwright/test";
import { COURSE, TOKEN, checkinApi } from "./support/checkin-fixtures";

// US-AS-11: đường quét trong app (getUserMedia + BarcodeDetector). Chromium headless không có BarcodeDetector nên giả cả hai;
// đo: mã đúng → gửi token, tắt camera; mã lạ bị bỏ qua im lặng; từ chối camera → nhập mã.
const stub = (qr: string[], deny = false) =>
  `(() => {
    window.__stopped = 0;
    window.Capacitor = { isNativePlatform: () => true, isPluginAvailable: () => false };
    navigator.mediaDevices.getUserMedia = async () => {
      if (${deny}) throw new DOMException("denied", "NotAllowedError");
      const c = document.createElement("canvas");
      c.width = c.height = 16;
      const s = c.captureStream(5);
      c.getContext("2d").fillRect(0, 0, 16, 16);
      s.getTracks().forEach((t) => { const st = t.stop.bind(t); t.stop = () => { window.__stopped++; st(); }; });
      return s;
    };
    const queue = ${JSON.stringify(qr)}.map((v) => v.replace("ORIGIN", location.origin));
    window.BarcodeDetector = class { async detect() { const v = queue.shift(); return v ? [{ rawValue: v }] : []; } };
  })()`;

test("checkin scan: mã lạ bị bỏ qua, mã đúng → gửi token một lần và tắt camera", async ({ page, context }) => {
  const good = `ORIGIN/checkin?c=${COURSE}#${TOKEN}`;
  await page.addInitScript(stub(["https://evil.example/checkin?c=" + COURSE + "#" + TOKEN, "không phải URL", good]));
  const log = await checkinApi(page, context);
  await page.goto("/checkin");
  await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
  expect(log.posts).toHaveLength(1);
  expect(log.posts[0].body.token).toBe(TOKEN);
  expect(await page.evaluate(() => (window as unknown as { __stopped: number }).__stopped)).toBeGreaterThan(0);
});

test("checkin scan: từ chối camera → «Chưa cấp quyền camera.» + ô nhập mã", async ({ page, context }) => {
  await page.addInitScript(stub([], true));
  await checkinApi(page, context);
  await page.goto("/checkin");
  await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
  await expect(page.getByText("Chưa cấp quyền camera.")).toBeVisible();
  await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
});
