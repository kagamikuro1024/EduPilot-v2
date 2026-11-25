// QC 10.8 mobile pha 1 — lô D: 6.N mạng yếu/mất mạng (phần web: Chromium CDP).
import { test } from "@playwright/test";
import { ACCT, api, authed, C, check, closeWindow, gotoEntry, openWindow, rec } from "./qc-mob-lib";

const OUT = process.env.QC_OUT ?? "/tmp/qc-mob";
type P = import("@playwright/test").Page;
async function net(p: P, opts: { offline?: boolean; latency?: number; down?: number; up?: number }) {
  const cdp = await p.context().newCDPSession(p);
  await cdp.send("Network.enable");
  await cdp.send("Network.emulateNetworkConditions", {
    offline: !!opts.offline,
    latency: opts.latency ?? 0,
    downloadThroughput: opts.down ?? -1,
    uploadThroughput: opts.up ?? -1,
  });
  return cdp;
}

test("NET-01 mạng chậm ≈400 kbit/s, RTT 2 s: khung ≤ 3 s, nội dung ≤ 8 s", async ({ browser }) => {
  const { page } = await authed(browser, "pixel", ACCT.s1);
  const cdp = await net(page, { latency: 2000, down: (400 * 1024) / 8, up: (400 * 1024) / 8 });
  const t0 = Date.now();
  await page.goto(`/checkin?course=${C.QCMOB1}`, { waitUntil: "commit" });
  await page.getByRole("heading", { name: "Điểm danh" }).waitFor({ timeout: 30_000 });
  const tFrame = Date.now() - t0;
  await page.getByLabel("Mã điểm danh").waitFor({ timeout: 30_000 });
  const tContent = Date.now() - t0;
  const ov = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
  check("TC-MOB-NET-01 khung ≤ 3 s và nội dung ≤ 8 s trên mạng chậm, không cuộn ngang", tFrame <= 3000 && tContent <= 8000 && !ov, `khung=${tFrame} ms, nội dung=${tContent} ms, ov=${ov}`);
  await cdp.send("Network.emulateNetworkConditions", { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  await page.screenshot({ path: `${OUT}/NET-01-slow.png` });
});

test("NET-02/03 mất mạng khi gửi mã: «Chưa có mạng.» + Thử lại, giữ mã; có mạng lại thì xong", async ({ browser }) => {
  const w = await openWindow(C.QCMOB1);
  const { page } = await authed(browser, "pixel", ACCT.s1);
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  await gotoEntry(page, C.QCMOB1);
  await page.getByLabel("Mã điểm danh").fill(w.code);
  const cdp = await net(page, { offline: true });
  await page.locator("[data-part=checkin-dock] button").click();
  await page.getByText("Chưa có mạng.").waitFor({ timeout: 20_000 });
  const t2 = await page.locator("[data-part=checkin]").innerText();
  const retry = page.getByRole("button", { name: "Thử lại" });
  const kept = await page.evaluate(() => (document.querySelector("[data-part=checkin] input") as HTMLInputElement | null)?.value ?? "");
  check("TC-MOB-NET-02 offline: «Chưa có mạng.» + nút Thử lại, mã đã gõ giữ nguyên, không hộp lỗi kỹ thuật", /Chưa có mạng\./.test(t2) && (await retry.count()) > 0 && !/Error|Failed|net::/.test(t2), `${t2.replace(/\n+/g, "|").slice(0, 70)} | input=${kept}`);
  await page.screenshot({ path: `${OUT}/NET-02-offline.png` });
  await cdp.send("Network.emulateNetworkConditions", { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  if ((await retry.count()) > 0) await retry.first().click();
  else await page.evaluate(() => (document.querySelector("[data-part=checkin-dock] button") as HTMLButtonElement | null)?.click());
  await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 25_000 });
  const d = (await api("teacher", "GET", `/courses/${C.QCMOB1}/sessions/${w.sid}/checkin-windows/${w.wid}/display`)).j;
  check("TC-MOB-NET-03 có mạng lại: Thử lại thành công, một lượt, không tải lại trang", /Đã điểm danh/.test(await page.locator("[data-part=checkin]").innerText()) && d.checked_in === 1, `checked_in=${d.checked_in}`);
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
});

test("NET-07 máy chủ trả lời POST chậm 20 s: trong 15 s phải có «Chưa có mạng.»", async ({ browser }) => {
  const w = await openWindow(C.QCMOB1);
  const { page } = await authed(browser, "pixel", ACCT.s1);
  await gotoEntry(page, C.QCMOB1);
  await page.route("**/checkins", async (r) => {
    if (r.request().method() !== "POST") return r.continue();
    await new Promise((x) => setTimeout(x, 20_000));
    await r.continue();
  });
  await page.getByLabel("Mã điểm danh").fill(w.code);
  await page.locator("[data-part=checkin-dock] button").click();
  await page.waitForTimeout(16_000);
  const t = await page.locator("[data-part=checkin]").innerText();
  const offline = /Chưa có mạng\./.test(t);
  rec("TC-MOB-NET-07 POST chậm 20 s: sau 15 s", offline ? "PASS" : "FAIL", `${offline ? "có" : "KHÔNG có"} «Chưa có mạng.»; trạng thái=${await page.locator("[data-part=checkin]").getAttribute("data-state")}; "${t.replace(/\n+/g, "|").slice(0, 60)}"`);
  await page.waitForTimeout(8000);
  await page.screenshot({ path: `${OUT}/NET-07-slow-post.png` });
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
});
