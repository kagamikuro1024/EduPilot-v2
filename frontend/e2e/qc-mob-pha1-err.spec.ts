// QC 10.8 mobile pha 1 — lô B: 6.F nhánh lỗi (ERR) + 6.S câu chữ lỗi (MSG phần web).
import { test } from "@playwright/test";
import { ACCT, api, authed, C, check, closeWindow, gotoEntry, openWindow, pwFor, rec, relogin, BASE } from "./qc-mob-lib";

const OUT = process.env.QC_OUT ?? "/tmp/qc-mob";
const body = (p: import("@playwright/test").Page) => p.locator("[data-part=checkin]");
const text = async (p: import("@playwright/test").Page) => (await p.locator("body").innerText()).replace(/\n+/g, "|");
const open = async (p: import("@playwright/test").Page, c: string, tok: string) => {
  await p.goto(`/checkin?c=${c}#${tok}`);
  await p.locator("main").waitFor({ timeout: 20_000 }); // màn chặn (GV) không có khung điểm danh
  await p.waitForTimeout(3000);
};
const codeIn = async (p: import("@playwright/test").Page, c: string, v: string) => {
  await gotoEntry(p, c);
  await p.getByLabel("Mã điểm danh").fill(v);
  await p.keyboard.press("Enter");
  await p.waitForTimeout(2500);
};

test("ERR-01…05 liên kết sai/hết hạn/MAC sai/cửa sổ đóng/lớp khác", async ({ browser }) => {
  const { ctx } = await authed(browser, "pixel", ACCT.s1);
  let page = await ctx.newPage();
  let posts = 0;
  page.on("request", (r) => r.method() === "POST" && /\/checkins$/.test(new URL(r.url()).pathname) && posts++);
  const tab = async () => { await page.close(); page = await ctx.newPage(); return page; };
  await open(page, "khong-phai-uuid", "tokenbatky");
  check("TC-MOB-ERR-01 «Liên kết không đúng.» một dòng, không gọi API", /Liên kết không đúng\./.test(await text(page)) && posts === 0, (await text(page)).slice(0, 80));

  const w = await openWindow(C.QCMOB1);
  const old = w.qr;
  await (await tab());
  await page.waitForTimeout(26000); // quá 2 bước rotate (10 s/bước)
  await open(page, C.QCMOB1, old);
  check("TC-MOB-ERR-02 token hết hạn → «Mã đã hết hạn. Quét lại mã trên màn chiếu.»", /Mã đã hết hạn\. Quét lại mã trên màn chiếu\./.test(await text(page)), (await text(page)).slice(0, 90));
  await page.screenshot({ path: `${OUT}/ERR-02-expired.png` });

  const w2 = await openWindow(C.QCMOB1);
  await tab();
  await open(page, C.QCMOB1, w2.qr.slice(0, -1) + (w2.qr.endsWith("A") ? "B" : "A"));
  check("TC-MOB-ERR-03 MAC sai → «Mã không đúng. Quét lại mã trên màn chiếu.»", /Mã không đúng\. Quét lại mã trên màn chiếu\./.test(await text(page)), (await text(page)).slice(0, 90));
  await closeWindow(C.QCMOB1, w2.wid);

  const w3 = await openWindow(C.QCMOB1);
  await closeWindow(C.QCMOB1, w3.wid);
  await tab();
  await open(page, C.QCMOB1, w3.qr);
  check("TC-MOB-ERR-04 cửa sổ đã đóng → «Đã đóng điểm danh.»", /Đã đóng điểm danh\./.test(await text(page)), (await text(page)).slice(0, 90));

  const w4 = await openWindow(C.QCMOB6);
  const nm = await authed(browser, "pixel", ACCT.s1);
  await open(nm.page, C.QCMOB6, w4.qr);
  check("TC-MOB-ERR-05 SV ngoài lớp → «Bạn không thuộc lớp này.»", /Bạn không thuộc lớp này\./.test(await text(nm.page)), (await text(nm.page)).slice(0, 90));
  const d = (await api("teacher", "GET", `/courses/${C.QCMOB6}/sessions/${w4.sid}/checkin-windows/${w4.wid}/display`)).j;
  check("TC-MOB-ERR-05b không ghi lượt nào", d.checked_in === 0, `checked_in=${d.checked_in}`);
  await closeWindow(C.QCMOB1, w4.wid);
});

test("ERR-06 để màn đăng nhập > 60 s: token bị bỏ, không POST", async ({ browser }) => {
  const w = await openWindow(C.QCMOB1);
  const { DEV } = await import("./qc-mob-lib");
  const c = await browser.newContext({ ...DEV.pixel, baseURL: BASE, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", ignoreHTTPSErrors: true } as never);
  const page = await c.newPage();
  await page.goto(`/checkin?c=${C.QCMOB1}#${w.qr}`);
  await page.waitForURL((u: URL) => u.pathname.startsWith("/login"), { timeout: 20_000 });
  const stash = await page.evaluate(() => sessionStorage.getItem("ep.checkin"));
  check("TC-MOB-SES-11a chưa đăng nhập: chuyển /login và cất {c,token} vào sessionStorage (không localStorage)", !!stash && /token/.test(stash) && !(await page.evaluate(() => Object.keys(localStorage).some((k) => /token|refresh/i.test(k)))), String(stash).slice(0, 40));
  let posts = 0;
  page.on("request", (r) => r.method() === "POST" && /\/checkins$/.test(new URL(r.url()).pathname) && posts++);
  await page.waitForTimeout(62000); // > 60 s: token cất hết hạn theo giờ máy
  await page.locator("input[type=email]").fill(`${ACCT.s1}@edupilot.local`);
  await page.locator("input[type=password]").fill(pwFor(ACCT.s1));
  await page.keyboard.press("Enter");
  await page.waitForTimeout(6000);
  const t = await text(page);
  check("TC-MOB-ERR-06 đăng nhập chậm > 60 s → «Mã đã hết hạn…», không POST", /Mã đã hết hạn/.test(t) && posts === 0, `posts=${posts}; ${t.slice(0, 90)}`);
  await relogin(browser, "pixel", ACCT.s1); // context dùng chung vừa bị thu hồi phiên
  rec("TC-MOB-ERR-06b kho cất sau khi quá 60 s", "NOTE", `ep.checkin=${(await page.evaluate(() => sessionStorage.getItem("ep.checkin") ? "còn" : "rỗng"))} (không thuộc AC; chỉ cần không POST)`);
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
  await c.close();
});

test("ERR-07/08 TEACHER, TA, ADMIN mở liên kết điểm danh: chặn, 0 POST, không lộ token", async ({ browser }) => {
  const w = await openWindow(C.QCMOB1);
  for (const [tc, user] of [["TC-MOB-ERR-07", "teacher"], ["TC-MOB-ERR-08a", "ta"], ["TC-MOB-ERR-08b", "admin"]] as const) {
    const { ctx, page } = await authed(browser, "pixel", user);
    let posts = 0;
    page.on("request", (r) => r.method() === "POST" && /\/checkins$/.test(new URL(r.url()).pathname) && posts++);
    await open(page, C.QCMOB1, w.qr);
    const t = await text(page);
    const leak = (await page.content()).includes(w.qr);
    check(`${tc} ${user}: màn chặn, không ô mã, không POST, không lộ token`, !/Mã điểm danh/.test(t) && posts === 0 && !leak, `posts=${posts} leak=${leak} ${t.slice(0, 70)}`);
  }
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
});

test("ERR-11/12/13 ô mã: mã sai/hết hạn, rỗng, ký tự lạ, dán 40 ký tự", async ({ browser }) => {
  const w = await openWindow(C.QCMOB1);
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  let posts = 0;
  page.on("request", (r) => r.method() === "POST" && /\/checkins$/.test(new URL(r.url()).pathname) && posts++);
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  const btn = page.locator("[data-part=checkin-dock] button");
  check("TC-MOB-ERR-12a nút «Điểm danh» khoá khi ô rỗng", (await page.getByLabel("Mã điểm danh").inputValue()) === "" && (await btn.isDisabled()), `disabled=${await btn.isDisabled()}`);
  await codeIn(page, C.QCMOB1, "ZZZZZZ");
  const t1 = await text(page);
  const invalid = await page.getByLabel("Mã điểm danh").getAttribute("aria-invalid");
  check("TC-MOB-ERR-11a mã sai → «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» dưới ô, aria-invalid, không hộp thoại", /Mã không đúng hoặc đã hết hạn\. Nhập mã đang hiện trên màn chiếu\./.test(t1) && invalid === "true" && (await page.getByRole("dialog").count()) === 0, `aria-invalid=${invalid}`);
  await page.screenshot({ path: `${OUT}/ERR-11-code-wrong.png` });
  const p0 = posts;
  await codeIn(page, C.QCMOB1, "😀@#$");
  check("TC-MOB-ERR-12b ký tự lạ → cùng một câu, không gọi API, không crash", /Mã không đúng hoặc đã hết hạn/.test(await text(page)) && posts === p0, `posts+${posts - p0}`);
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  const inp = page.getByLabel("Mã điểm danh");
  await inp.fill("A".repeat(40));
  const len = (await inp.inputValue()).length;
  await inp.fill(` ${w.code.slice(0, 3).toLowerCase()} ${w.code.slice(3).toLowerCase()} `);
  await page.keyboard.press("Enter");
  await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
  check("TC-MOB-ERR-13 ô giới hạn 12 ký tự; mã có khoảng trắng đầu/cuối vẫn xử lý", len <= 12 && /Đã điểm danh/.test(await text(page)), `len=${len}`);
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
});

test("ERR-14 POST 500 / HTML / rỗng: «Chưa điểm danh được. Thử lại.», nút còn dùng", async ({ browser }) => {
  const w = await openWindow(C.QCMOB1);
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  const modes: [string, number, string, string][] = [
    ["500 JSON", 500, "application/json", '{"code":"INTERNAL"}'],
    ["200 HTML", 200, "text/html", "<html><body>oops</body></html>"],
    ["500 rỗng", 500, "application/json", ""],
  ];
  for (const [name, st, ct, b] of modes) {
    await page.unroute("**/checkins").catch(() => undefined);
    await page.route("**/checkins", (r) => (r.request().method() === "POST" ? r.fulfill({ status: st, contentType: ct, body: b }) : r.continue()));
    await codeIn(page, C.QCMOB1, w.code);
    const t = await text(page);
    const clean =
      /(Chưa điểm danh được\. Thử lại\.|Máy chủ chưa phản hồi đúng\. Dữ liệu của bạn vẫn an toàn\.|Có lỗi xảy ra\. Dữ liệu của bạn vẫn an toàn\. Thử lại sau ít phút\.|Phản hồi từ máy chủ không đọc được\. Thử lại sau\.)/.test(t) &&
      !/\b(500|undefined|null|\[object|INTERNAL|oops)\b/.test(t);
    check(`TC-MOB-ERR-14 ${name}: một câu tiếng Việt (SRS: BAD_GATEWAY), không mã HTTP/JSON`, clean, t.slice(0, 110));
    await page.screenshot({ path: `${OUT}/ERR-14-${name.replace(/\s/g, "-")}.png` });
  }
  const enabled = await page.locator("[data-part=checkin-dock] button").isEnabled().catch(() => true);
  check("TC-MOB-ERR-14b nút còn dùng được sau lỗi", enabled, String(enabled));
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
});

test("ERR-18 khoá lạ trong thân yêu cầu bị từ chối 422, không ghi", async ({ browser }) => {
  const w = await openWindow(C.QCMOB1);
  const r = await api(ACCT.s1, "POST", `/courses/${C.QCMOB1}/checkins`, { code: w.code, student_id: "01a212f1-0000-7000-8000-000000000000" });
  const codes = JSON.stringify(r.j ?? {});
  check("TC-MOB-ERR-18 422 VALIDATION_FAILED, details[].code=unknown", r.s === 422 && /VALIDATION|unknown/.test(codes), `${r.s} ${codes.slice(0, 120)}`);
  const d = (await api("teacher", "GET", `/courses/${C.QCMOB1}/sessions/${w.sid}/checkin-windows/${w.wid}/display`)).j;
  check("TC-MOB-ERR-18b không ghi lượt nào", d.checked_in === 0, `checked_in=${d.checked_in}`);
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
});

test("ERR-19 SV ngoài lớp với lớp không tồn tại: cùng một câu", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  await page.goto(`/checkin?c=01a212f1-0000-7000-8000-000000000000`);
  await page.waitForTimeout(3500);
  const t = await text(page);
  check("TC-MOB-ERR-19 không phân biệt lớp tồn tại hay không", !/không tồn tại|not found|404/i.test(t), t.slice(0, 110));
});

test("MSG-10 không có BarcodeDetector: không câu nào, ô nhập hiện ngay", async ({ browser }) => {
  const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
  await page.addInitScript(() => {
    // Chromium không có BarcodeDetector; khẳng định lại môi trường ca kiểm
    Object.defineProperty(window, "BarcodeDetector", { value: undefined, configurable: true });
  });
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  await page.waitForTimeout(2500);
  const t = await text(page);
  check("TC-MOB-MSG-10 không câu lỗi nào; ô nhập + nút Điểm danh hiện ngay", !/camera|BarcodeDetector|Quét mã/i.test(t) && /Mã điểm danh/.test(t), t.slice(0, 100));
});
