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
    // canvas = phép thử khả dụng (QR mẫu vẽ tại chỗ, P-11-131); video = khung camera
    window.BarcodeDetector = class { async detect(src) { if (src instanceof HTMLCanvasElement) return [{ rawValue: "probe" }]; const v = queue.shift(); return v ? [{ rawValue: v }] : []; } };
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

// DEF-R10.8-04 (TC-AS11-07): BarcodeDetector có nhưng không đọc được qr_code ⇒ không có nút quét, ô «Mã điểm danh» hiện ngay.
for (const [name, patch] of [
  ["getSupportedFormats() rỗng", `window.BarcodeDetector.getSupportedFormats = async () => [];`],
  ["getSupportedFormats() không có qr_code", `window.BarcodeDetector.getSupportedFormats = async () => ["code_128", "ean_13"];`],
  ["detect() ném lỗi dù báo qr_code (máy thiếu GMS, «not operational»)", `window.BarcodeDetector.getSupportedFormats = async () => ["qr_code"]; window.BarcodeDetector.prototype.detect = async () => { throw new DOMException("Barcode Detector is not operational", "NotSupportedError"); };`],
  ["detect() không đọc ra QR mẫu", `window.BarcodeDetector.getSupportedFormats = async () => ["qr_code"]; window.BarcodeDetector.prototype.detect = async () => [];`],
  ["hàm dựng ném lỗi", `window.BarcodeDetector = class { constructor() { throw new TypeError("formats không hỗ trợ"); } };`],
] as const) {
  test(`checkin scan: ${name} ⇒ không nút «Quét mã điểm danh», ô nhập mã hiện ngay`, async ({ page, context }) => {
    await page.addInitScript(stub([]) + ";" + patch);
    await checkinApi(page, context);
    await page.goto("/checkin");
    await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
    await expect(page.getByRole("button", { name: "Quét mã điểm danh" })).toHaveCount(0);
  });
}

test("checkin scan: getSupportedFormats() có qr_code ⇒ vẫn có nút quét", async ({ page, context }) => {
  await page.addInitScript(stub([]) + `;window.BarcodeDetector.getSupportedFormats = async () => ["qr_code"];`);
  await checkinApi(page, context);
  await page.goto("/checkin");
  await expect(page.getByRole("button", { name: "Quét mã điểm danh" }).first()).toBeVisible();
});
