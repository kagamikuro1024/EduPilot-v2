import { expect, test } from "@playwright/test";
import { COURSE, TOKEN, checkinApi, err, json } from "./support/checkin-fixtures";

// US-AS-11 AC4, AC6: trong app Android `/checkin` gọi plugin EduPilotAttest (giả ở đây bằng `window.Capacitor`). Không dùng WebCrypto.
const LINK = `/checkin?c=${COURSE}#${TOKEN}`;
const SPKI = "U1BLSQ=="; // chuỗi bất kỳ: máy chủ giả không kiểm
const stub = () =>
  `(() => {
    const calls = [];
    window.__plugin = calls;
    window.Capacitor = {
      isNativePlatform: () => true,
      isPluginAvailable: (n) => n === "EduPilotAttest",
      nativePromise: async (p, m, o) => {
        calls.push({ p, m, o });
        if (m === "registerDevice") return { public_key: "${SPKI}", attestation_chain: ["Q0VSVA=="] };
        if (m === "sign") return { signature: "REVSU0lH", public_key: "${SPKI}" };
        if (m === "isMockLocation") return { mock: true };
        throw new Error("?");
      },
    };
  })()`;

test("checkin app: challenge → registerDevice → POST /me/devices → ký bằng Keystore, gửi ANDROID_KEY; lần sau không đăng ký lại", async ({ page, context }) => {
  await page.addInitScript(stub());
  const reg: unknown[] = [];
  const log = await checkinApi(page, context);
  await page.route("**/api/v1/me/devices/challenge", (r) => r.fulfill(json({ nonce: "bm9uY2U" })));
  await page.route("**/api/v1/me/devices", (r) => {
    reg.push(JSON.parse(r.request().postData() ?? "{}"));
    return r.fulfill(json({ key_id: "KID-1" }, 201));
  });
  await page.goto(LINK);
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
  expect(log.posts[0].body.device).toEqual({ kind: "ANDROID_KEY", key_id: "KID-1", signature: "REVSU0lH" });
  expect(reg).toEqual([{ kind: "ANDROID_KEY", public_key: SPKI, attestation_chain: ["Q0VSVA=="] }]);
  const calls = await page.evaluate(() => (window as unknown as { __plugin: Array<{ m: string; o?: { nonce?: string; message?: string } }> }).__plugin);
  expect(calls.map((c) => c.m)).toEqual(["registerDevice", "sign"]);
  expect(calls[0].o?.nonce).toBe("bm9uY2U");
  expect(calls[1].o?.message).toMatch(/^edupilot-checkin-v1\|v1\./);

  // trang mới: key_id đã cất ⇒ chỉ ký
  const p2 = await context.newPage();
  await p2.addInitScript(stub());
  const log2 = await checkinApi(p2, context);
  await p2.route("**/api/v1/me/devices**", (r) => r.fulfill(err(500, "INTERNAL"))); // không được gọi
  await p2.goto(LINK);
  await expect(p2.locator("[data-part=done]")).toContainText("07:03");
  expect(log2.posts[0].body.device).toMatchObject({ kind: "ANDROID_KEY", key_id: "KID-1" });
});

test("checkin app: emulator — đăng ký 422 ATTESTATION_FAILED → vẫn ký Keystore, gửi WEB_KEY (mức 2); không thử lại trong phiên", async ({ page, context }) => {
  await page.addInitScript(stub());
  let regCalls = 0;
  const log = await checkinApi(page, context);
  await page.route("**/api/v1/me/devices/challenge", (r) => r.fulfill(json({ nonce: "bm9uY2U" })));
  await page.route("**/api/v1/me/devices", (r) => {
    regCalls++;
    return r.fulfill(err(422, "ATTESTATION_FAILED"));
  });
  await page.goto(LINK);
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
  expect(log.posts[0].body.device).toEqual({ kind: "WEB_KEY", public_key: SPKI, signature: "REVSU0lH" });
  expect(regCalls).toBe(1);
});

test("checkin app: plugin lỗi → không gửi device (mức 1), vẫn điểm danh; lớp kiểm tra vị trí → geo.mock lấy từ plugin", async ({ page, context }) => {
  await page.addInitScript(`(() => { window.Capacitor = { isNativePlatform: () => true, isPluginAvailable: () => true, nativePromise: async (p, m) => { if (m === "isMockLocation") return { mock: true }; throw new Error("keystore"); } }; })()`);
  await context.grantPermissions(["geolocation"]);
  await context.setGeolocation({ latitude: 21.005, longitude: 105.843, accuracy: 20 });
  const log = await checkinApi(page, context, { geoCheck: true });
  await page.route("**/api/v1/me/devices/challenge", (r) => r.fulfill(json({ nonce: "bm9uY2U" })));
  await page.goto(LINK);
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
  expect(log.posts[0].body.device).toBeUndefined();
  expect(log.posts[0].body.geo).toMatchObject({ lat: 21.005, lng: 105.843, mock: true });
});
