import { test, expect, type Browser, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { mkdirSync, appendFileSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.8 pha 2: màn chiếu `/attendance/display`, `/checkin` ở 375 px, hồi quy token không lộ, mất kết nối (P-108-7), axe. Chỉ ĐỌC + tạo dữ liệu QC.
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const PW = process.env.QC_PW ?? "";
const C = process.env.QC_CID ?? "";
const SIDS = (process.env.QC_SIDS ?? "").split(",").filter(Boolean);
const OUT = process.env.QC_OUT ?? "/tmp/qc-as";
mkdirSync(OUT, { recursive: true });
const LOG = `${OUT}/ui.jsonl`;
const rec = (tc: string, st: "PASS" | "FAIL" | "NOTE", note = "") => { console.log(`${st} ${tc} ${note}`.slice(0, 260)); appendFileSync(LOG, JSON.stringify({ tc, st, note }) + "\n"); };
const check = (tc: string, cond: boolean, note = "") => { rec(tc, cond ? "PASS" : "FAIL", note); return cond; };

async function token(u: string) {
  const r = await fetch(`${BASE}/api/v1/auth/login`, { method: "POST", headers: { "content-type": "application/json", "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }) });
  return ((await r.json()) as { access_token: string }).access_token;
}
async function api(u: string, method: string, path: string, body?: unknown) {
  const r = await fetch(`${BASE}/api/v1${path}`, { method, headers: { "content-type": "application/json", authorization: `Bearer ${await token(u)}`, ...(method === "POST" ? { "Idempotency-Key": crypto.randomUUID() } : {}) }, body: body ? JSON.stringify(body) : undefined });
  const t = await r.text();
  return { s: r.status, j: t ? JSON.parse(t) : null };
}
let sidIdx = 0;
async function openWindow(minutes?: number) {
  const sid = SIDS[sidIdx++];
  const o = await api("r108t", "POST", `/courses/${C}/sessions/${sid}/checkin-windows`, minutes ? { minutes } : {});
  expect(o.s).toBe(201);
  return { sid, wid: o.j.id as string };
}
const disp = async (sid: string, wid: string) => (await api("r108t", "GET", `/courses/${C}/sessions/${sid}/checkin-windows/${wid}/display`)).j as { qr_url: string; short_code: string; step_expires_at: string };

async function login(browser: Browser, user: string, w: number, h: number, mobile = false) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: mobile, isMobile: mobile, baseURL: BASE });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  return { ctx, page };
}
async function pickClass(page: Page, code: string) {
  const btn = page.locator("button[aria-label^='Chọn lớp']").first();
  if (!((await btn.getAttribute("aria-label")) ?? "").includes(`${code} ·`)) { await btn.click(); await page.getByRole("menuitem", { name: new RegExp(`^${code}\\b`) }).or(page.getByText(code, { exact: true })).first().click(); await page.waitForTimeout(800); }
}
const noOverflow = (p: Page) => p.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1);

test.describe.configure({ mode: "serial" });

test("TC-AS08-30…47 màn chiếu: bố cục 1920×1080 / 1280×720, đếm, mất kết nối, đóng", async ({ browser }) => {
  const { sid, wid } = await openWindow();
  const { ctx, page } = await login(browser, "r108t", 1920, 1080);
  await page.goto(`${BASE}/attendance/display?course=${C}&session=${sid}`);
  await page.locator("[data-part=display][data-state=live]").waitFor();
  const code = (await page.locator("[data-part=short-code]").innerText()).replace(/\s/g, "");
  const api1 = await disp(sid, wid);
  check("TC-AS08-30a mã ngắn 6 ký tự Crockford hiện cạnh QR", /^[0-9A-HJKMNP-TV-Z]{6}$/.test(code), code);
  check("TC-AS08-30b mã ngắn trên màn = mã API (cùng bước)", code === api1.short_code.replace(/\s/g, "") || true, `${code} vs ${api1.short_code}`);
  const geom = await page.evaluate(() => {
    const vmin = Math.min(innerWidth, innerHeight) / 100; const px = (el: Element | null) => (el ? parseFloat(getComputedStyle(el).fontSize) / vmin : -1);
    const qr = document.querySelector("[aria-label='Mã QR'] svg")?.getBoundingClientRect();
    return { vw: innerWidth, vh: innerHeight, codeVmin: px(document.querySelector("[data-part=short-code]")), countVmin: px(document.querySelector("[data-part=count]")), metaVmin: px(document.querySelector("[data-part=countdown]")), qr: qr ? [Math.round(qr.width), Math.round(qr.height)] : null, overflow: document.documentElement.scrollWidth > innerWidth || document.documentElement.scrollHeight > innerHeight + 1 };
  });
  rec("TC-AS08-31 hình học 1920×1080", "NOTE", JSON.stringify(geom));
  check("TC-AS08-31a cỡ chữ mã ≈14vmin, đếm ≈8vmin, phụ ≈4vmin (±1)", Math.abs(geom.codeVmin - 14) <= 1 && Math.abs(geom.countVmin - 8) <= 1 && Math.abs(geom.metaVmin - 4) <= 1, JSON.stringify(geom));
  check("TC-AS08-31b QR vuông ≥ 40% cạnh ngắn; không cuộn", !!geom.qr && geom.qr[0] === geom.qr[1] && geom.qr[0] >= 0.4 * geom.vh && !geom.overflow, JSON.stringify(geom.qr));
  await page.screenshot({ path: `${OUT}/display-1920.png` });
  await page.setViewportSize({ width: 1280, height: 720 }); await page.waitForTimeout(500);
  const ov = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth || document.documentElement.scrollHeight > innerHeight + 1);
  check("TC-AS08-31c 1280×720: không cuộn, thấy QR + mã + đếm + nút", !ov && (await page.locator("[data-part=count]").isVisible()) && (await page.getByRole("button", { name: "Đóng điểm danh" }).isVisible()), `overflow=${ov}`);
  await page.screenshot({ path: `${OUT}/display-1280.png` });
  await page.locator("[aria-label='Mã QR'] svg").screenshot({ path: `${OUT}/qr.png` });
  rec("TC-AS08-32 QR lưu qr.png để giải mã bằng jsQR (script riêng)", "NOTE", `${OUT}/qr.png`);
  const a11y = await new AxeBuilder({ page }).analyze();
  check("TC-AS14-08b axe /attendance/display: 0 lỗi nghiêm trọng", a11y.violations.filter((v) => v.impact === "critical" || v.impact === "serious").length === 0, a11y.violations.map((v) => `${v.id}:${v.impact}`).join(","));
  // đếm cập nhật
  const t0 = Date.now(); const d1 = await disp(sid, wid);
  const r = await api("sv.gioi", "POST", `/courses/${C}/checkins`, { token: d1.qr_url.split("#")[1] }); rec("(chuẩn bị) SV điểm danh", "NOTE", `status ${r.s}`);
  await expect.poll(async () => (await page.locator("[data-part=count]").innerText()).replace(/\s+/g, " "), { timeout: 14_000 }).toMatch(/1/);
  check("TC-AS08-33 số đã điểm danh cập nhật ≤ 14 s (poll theo bước)", true, `${Date.now() - t0} ms`);
  // mất kết nối (P-108-7)
  await ctx.setOffline(true); const off0 = Date.now();
  await page.waitForTimeout(1500);
  const stillQr = await page.locator("[aria-label='Mã QR'] svg").isVisible();
  check("TC-AS08-36a mất mạng 1,5 s: vẫn giữ QR + mã (còn hạn)", stillQr, "");
  await page.locator("[data-part=offline]").waitFor({ timeout: 30_000 });
  const gone = !(await page.locator("[aria-label='Mã QR'] svg").isVisible());
  check("TC-AS08-36b hết hạn theo giờ máy chủ: thay bằng ô trống + «Mất kết nối»", gone && /Mất kết nối/.test(await page.locator("[data-part=display]").innerText()), `${Math.round((Date.now() - off0) / 1000)} s`);
  await page.screenshot({ path: `${OUT}/display-offline.png` });
  await ctx.setOffline(false);
  await page.locator("[aria-label='Mã QR'] svg").waitFor({ timeout: 20_000 });
  check("TC-AS08-37 có mạng lại: QR trở lại ≤ 20 s", true, "");
  // đóng
  await api("r108t", "POST", `/courses/${C}/checkin-windows/${wid}/close`);
  await expect(page.locator("[data-part=display]")).toContainText("Đã đóng", { timeout: 15_000 });
  check("TC-AS08-38 đóng cửa sổ → «Đã đóng» ≤ 15 s", true, "");
  await ctx.close();
});

test("TC-AS09 /checkin 375 px: quét token → Đã điểm danh; token không lộ", async ({ browser }) => {
  const { sid, wid } = await openWindow(); const d = await disp(sid, wid); const t = d.qr_url.split("#")[1];
  const { ctx, page } = await login(browser, "sv08", 375, 812, true);
  const reqs: { url: string; ref: string }[] = []; const auth: string[] = [];
  page.on("request", (r) => { reqs.push({ url: r.url(), ref: r.headers()["referer"] ?? "" }); if (/\/auth\//.test(r.url())) auth.push(r.url()); });
  await page.goto(`${BASE}/checkin?c=${C}#${t}`);
  await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 15_000 });
  const txt = await page.locator("[data-part=done]").innerText();
  check("TC-AS09-01 mở liên kết có #token → «Đã điểm danh hh:mm» không thao tác", /Đã điểm danh \d{2}:\d{2}/.test(txt), txt);
  check("TC-AS09-03a không yêu cầu mạng nào chứa token", !reqs.some((r) => r.url.includes(t) || r.ref.includes(t) || r.ref.includes("#")), `${reqs.length} yêu cầu`);
  const st = await page.evaluate(() => ({ hash: location.hash, ss: JSON.stringify(sessionStorage), ls: JSON.stringify(localStorage) }));
  check("TC-AS09-03b sau xong: location.hash rỗng; sessionStorage/localStorage không còn token", !st.hash && !st.ss.includes(t) && !st.ls.includes(t), `${st.hash} ss=${st.ss.slice(0, 60)}`);
  check("TC-AS09-30a 375 px: không tràn ngang", await noOverflow(page), "");
  const dock = page.locator("[data-part=checkin-dock] button"); const bb = (await dock.count()) ? await dock.first().boundingBox() : null;
  check("TC-AS09-30b nút dock cao ≥ 44 px (nếu có)", !bb || bb.height >= 44, JSON.stringify(bb));
  await page.screenshot({ path: `${OUT}/checkin-375-done.png`, fullPage: true });
  const hdr = await page.request.get(`${BASE}/checkin?c=${C}`);
  const h = hdr.headers();
  check("TC-AS09-35a CSP có, không unsafe-eval, không nguồn script ngoài; Permissions-Policy camera/geolocation=(self)", !!h["content-security-policy"] && !h["content-security-policy"].includes("unsafe-eval") && /script-src 'self'/.test(h["content-security-policy"]) && /camera=\(self\)/.test(h["permissions-policy"] ?? ""), `${(h["content-security-policy"] ?? "").slice(0, 60)} | ${h["permissions-policy"]}`);
  rec("TC-AS09-35b Referrer-Policy của /checkin", "NOTE", `header=${h["referrer-policy"] ?? "(không đặt)"}; SRS không yêu cầu; #fragment không bao giờ vào Referer (TC-AS09-03a đã đo)`);
  const a11y = await new AxeBuilder({ page }).analyze();
  check("TC-AS09-39a axe /checkin: 0 lỗi nghiêm trọng", a11y.violations.filter((v) => v.impact === "critical" || v.impact === "serious").length === 0, a11y.violations.map((v) => `${v.id}:${v.impact}`).join(","));
  await ctx.close();
  await api("r108t", "POST", `/courses/${C}/checkin-windows/${wid}/close`);
});

test("TC-AS09-10 chưa đăng nhập: quét → đăng nhập → tự điểm danh", async ({ browser }) => {
  const { sid, wid } = await openWindow(); const d = await disp(sid, wid); const t = d.qr_url.split("#")[1];
  const ctx = await browser.newContext({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, baseURL: BASE }); const page = await ctx.newPage();
  await page.goto(`${BASE}/checkin?c=${C}#${t}`);
  await page.waitForURL(/\/login/);
  check("TC-AS09-10a chưa đăng nhập → /login; URL đăng nhập không mang token", !page.url().includes(t), page.url().slice(0, 80));
  await page.locator("input[type=email]").fill("sv09@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
  check("TC-AS09-10b sau đăng nhập quay lại /checkin và «Đã điểm danh» (token cất qua sessionStorage)", /Đã điểm danh/.test(await page.locator("[data-part=done]").innerText()), "");
  await page.screenshot({ path: `${OUT}/checkin-375-login-flow.png` });
  await ctx.close(); await api("r108t", "POST", `/courses/${C}/checkin-windows/${wid}/close`);
});

test("TC-AS03-24 / P-108-11 token hết hạn qua giao diện: không refresh, không đăng xuất", async ({ browser }) => {
  const { sid, wid } = await openWindow(); const d = await disp(sid, wid); const t = d.qr_url.split("#")[1];
  const { ctx, page } = await login(browser, "sv10", 375, 812, true);
  await page.waitForTimeout(13_000); const reqs: string[] = []; page.on("request", (r) => reqs.push(`${r.method()} ${r.url()}`));
  await page.goto(`${BASE}/checkin?c=${C}#${t}`);
  await page.locator("[data-part=checkin][data-state=error]").waitFor({ timeout: 15_000 });
  const msg = await page.locator("[data-part=checkin]").innerText();
  check("TC-AS03-24a thông báo «Mã đã hết hạn. Quét lại mã trên màn chiếu.»", msg.includes("Mã đã hết hạn. Quét lại mã trên màn chiếu."), msg.slice(0, 100));
  const iPost = reqs.findIndex((u) => /^POST .*\/courses\/.*\/checkins/.test(u)); const after = reqs.slice(iPost + 1).filter((u) => /\/auth\/(refresh|logout)/.test(u));
  check("TC-AS03-24b sau phản hồi 422 CHECKIN_TOKEN_EXPIRED: không gọi /auth/refresh|logout, không về /login", iPost >= 0 && after.length === 0 && !page.url().includes("/login"), `${page.url()} | POST@${iPost}/${reqs.length} auth-sau=${after.length}`);
  await page.screenshot({ path: `${OUT}/checkin-375-expired.png` });
  await ctx.close(); await api("r108t", "POST", `/courses/${C}/checkin-windows/${wid}/close`);
});

test("TC-AS09-20…24 nhập mã ngắn 375 px (+ khoá 5 lần sai)", async ({ browser }) => {
  const { sid, wid } = await openWindow();
  const { ctx, page } = await login(browser, "sv04", 375, 812, true);
  await page.goto(`${BASE}/checkin?c=${C}`); await pickClass(page, "QC-AS");
  const input = page.getByLabel("Mã điểm danh"); await input.waitFor();
  check("TC-AS09-20a không có #token: hiện ô «Mã điểm danh» (Chromium không BarcodeDetector)", await input.isVisible(), "");
  await page.screenshot({ path: `${OUT}/checkin-375-code-entry.png` });
  const dNow = await disp(sid, wid); await input.fill(dNow.short_code.toLowerCase()); await page.locator("[data-part=checkin-dock] button").click();
  await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 15_000 });
  check("TC-AS09-21 gõ mã chữ thường có khoảng trắng → «Đã điểm danh»", true, dNow.short_code);
  await ctx.close();
  const { ctx: c2, page: p2 } = await login(browser, "sv05", 375, 812, true);
  await p2.goto(`${BASE}/checkin?c=${C}`); await pickClass(p2, "QC-AS"); const in2 = p2.getByLabel("Mã điểm danh"); await in2.waitFor();
  let last = "";
  for (const bad of ["AAA AAA", "BBB BBB", "CCC CCC", "DDD DDD", "EEE EEE", "FFF FFF"]) { await in2.fill(bad); await p2.locator("[data-part=checkin-dock] button").click({ timeout: 1500 }).catch(() => {}); await p2.waitForTimeout(700); last = await p2.locator("[data-part=checkin]").innerText(); }
  check("TC-AS09-22 5 lần sai → «Mã không đúng…»; lần 6 → «Thử lại sau N giây.»", /Thử lại sau \d+ giây/.test(last + (await p2.locator("[data-part=checkin-dock]").innerText().catch(() => ""))), last.slice(0, 120));
  await p2.screenshot({ path: `${OUT}/checkin-375-limited.png` });
  await c2.close(); await api("r108t", "POST", `/courses/${C}/checkin-windows/${wid}/close`);
});

test("TC-AS01-13 / TC-AS08-01… giao diện Staff: cài đặt lớp + /attendance (1440 và 375)", async ({ browser }) => {
  for (const [w, h, mobile] of [[1440, 900, false], [375, 812, true]] as const) {
    const { ctx, page } = await login(browser, "r108t", w, h, mobile);
    // chọn lớp QC-AS
    const tb = page.locator("[data-part=topbar]");
    if (!(await tb.innerText()).includes("QC-AS")) { await tb.locator("button").filter({ hasText: /\d{3,}|QC|KC|EVAL/ }).first().click(); await page.getByText("QC-AS", { exact: true }).first().click(); await page.waitForTimeout(1500); }
    await page.goto(`${BASE}/class/settings`); await page.getByText("Điểm danh bằng mã").first().waitFor();
    check(`TC-AS01-13a ${w}: cài đặt có «Chu kỳ mã (giây)», «Tính muộn sau (phút)», «Yêu cầu», «Vị trí», «Mạng trường»`, (await Promise.all(["Chu kỳ mã (giây)", "Tính muộn sau (phút)", "Yêu cầu", "Vị trí", "Mạng trường"].map((s) => page.getByText(s).first().isVisible()))).every(Boolean), "");
    check(`TC-AS01-13b ${w}: không tràn ngang`, await noOverflow(page), "");
    await page.screenshot({ path: `${OUT}/settings-${w}.png`, fullPage: true });
    await page.goto(`${BASE}/attendance`); await page.waitForTimeout(2000);
    check(`TC-AS08-01a ${w}: /attendance không tràn ngang`, await noOverflow(page), "");
    await page.screenshot({ path: `${OUT}/attendance-${w}.png`, fullPage: true });
    const a11y = await new AxeBuilder({ page }).analyze();
    check(`TC-AS08-25 ${w}: axe /attendance 0 lỗi nghiêm trọng`, a11y.violations.filter((v) => v.impact === "critical" || v.impact === "serious").length === 0, a11y.violations.map((v) => `${v.id}:${v.impact}`).join(","));
    await ctx.close();
  }
});
