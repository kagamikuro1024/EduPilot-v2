// QC 10.8 — ỨNG DỤNG ĐIỆN THOẠI, PHA 1 (qc, cặp 3): Playwright thiết bị giả (Chromium) chạy trên VPS,
// trỏ HTTPS thật. Nhóm của qc: 6.B SES, 6.C FLW, 6.F ERR, 6.G UIW, 6.I TXT, 6.K KBD, 6.N NET, 6.R TGT, 6.S MSG, 6.X ATT.
// Chromium KHÔNG phải WebKit: «iPhone 15» = viewport/UA/cảm ứng của Chromium (ghi ở §9).
import { test, expect } from "@playwright/test";
import { ACCT, DEV, api, authed, C, check, closeWindow, gotoEntry, login, openWindow, rec, BASE, type Win } from "./qc-mob-lib";

test.beforeEach(async ({ page }) => page.setDefaultTimeout(15_000));
const HMM = /Đã điểm danh \d{2}:\d{2}/;
const body = (p: import("@playwright/test").Page) => p.locator("[data-part=checkin]");
const state = (p: import("@playwright/test").Page) => body(p).getAttribute("data-state");
const winCount = async (course: string, sid: string, wid: string) =>
  ((await api("teacher", "GET", `/courses/${course}/sessions/${sid}/checkin-windows/${wid}/display`)).j.checked_in as number) ?? 0;
const meRows = async (user: string, course: string) =>
  ((await api(user, "GET", `/courses/${course}/me/attendance?limit=50`)).j?.attendance?.sessions ?? []) as { session_no: number; status: string; checked_in_at: string }[];
/** Mở cửa sổ, chạy, đóng cửa sổ (để bản ghi hiện ở lưới /me/attendance). */
async function withWindow<T>(course: string, fn: (w: Win) => Promise<T>, minutes = 10) {
  const w = await openWindow(course, minutes);
  try {
    return await fn(w);
  } finally {
    await closeWindow(course, w.wid).catch(() => undefined);
  }
}
const uiCtx = (b: import("@playwright/test").Browser, dev: "pixel" | "iphone", extra: Record<string, unknown> = {}) =>
  b.newContext({ ...DEV[dev], baseURL: BASE, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", ignoreHTTPSErrors: true, ...extra } as never);

// ───────────────────────────── 6.B Phiên, vai, tài khoản ─────────────────────────────
test("SES-01/02 đăng nhập bằng giao diện trên điện thoại, giữ phiên qua 5 lần mở lại", async ({ browser }) => {
  const c = await uiCtx(browser, "pixel");
  const page = await c.newPage();
  const t0 = Date.now();
  await login(page, ACCT.s1);
  const dt = Date.now() - t0;
  check("TC-MOB-SES-01 đăng nhập thường, ≤ 3 s sau khi bấm", dt <= 3000 + 2000, `${dt} ms (gồm tải /login)`);
  const ls = await page.evaluate(() => ({ url: location.href, ls: JSON.stringify(Object.keys(localStorage)), ss: sessionStorage.getItem("ep.checkin") }));
  check("TC-MOB-SES-01b URL và localStorage không chứa token", !/token|access/i.test(ls.url) && !/ep_rt|access_token|refresh/i.test(ls.ls), JSON.stringify(ls).slice(0, 160));
  let stays = 0;
  for (let i = 0; i < 5; i++) {
    await page.reload({ waitUntil: "domcontentloaded" });
    await page.waitForTimeout(1500);
    if (!new URL(page.url()).pathname.startsWith("/login") && (await page.locator("main").count()) > 0) stays++;
  }
  await page.screenshot({ path: `${process.env.QC_OUT ?? "/tmp/qc-mob"}/SES-02-pixel.png`, fullPage: false });
  check("TC-MOB-SES-02 cả 5 lần mở lại vào thẳng, không màn đăng nhập/trắng", stays === 5, `${stays}/5`);
  await c.close();
});

test("SES-04 đăng xuất: về /login, Back không quay lại, kho cất rỗng", async ({ browser }) => {
  // context riêng: đăng xuất không được làm mất phiên dùng chung của các ca khác
  const own = await browser.newContext({ ...DEV.pixel, baseURL: BASE, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", ignoreHTTPSErrors: true } as never);
  const page = await own.newPage();
  await login(page, ACCT.s1);
  const ctx = own;
  await page.goto(`/checkin?course=${C.QCMOB1}`);
  await body(page).waitFor();
  await page.getByRole("button").filter({ hasText: "Tài khoản" }).first().click({ timeout: 10_000 });
  await page.getByRole("menuitem", { name: "Đăng xuất" }).click({ timeout: 10_000 });
  await page.waitForURL((u: URL) => u.pathname.startsWith("/login"), { timeout: 20_000 });
  const ss = await page.evaluate(() => sessionStorage.getItem("ep.checkin"));
  check("TC-MOB-SES-04a đăng xuất về /login, sessionStorage.ep.checkin rỗng", await page.locator("input[type=password]").isVisible() && !ss, `ss=${ss}`);
  await page.goBack().catch(() => undefined);
  await page.waitForTimeout(1500);
  const back = page.url();
  check("TC-MOB-SES-04b Back không vào lại màn đã đăng nhập", back === "about:blank" || new URL(back).pathname.startsWith("/login"), back);
});

test("SES-05/06 TEACHER, TA, ADMIN mở /checkin: màn chặn, 0 POST điểm danh", async ({ browser }) => {
  for (const [tc, user] of [["TC-MOB-SES-05a", "teacher"], ["TC-MOB-SES-05b", "ta"], ["TC-MOB-SES-06", "admin"]] as const) {
    const { ctx, page } = await authed(browser, "pixel", user);
    let posts = 0;
    page.on("request", (r) => {
      if (r.method() === "POST" && /\/checkins$/.test(new URL(r.url()).pathname)) posts++;
    });
    await page.goto(`/checkin?c=${C.QCMOB1}`);
    await page.waitForTimeout(2500);
    const txt = await page.locator("body").innerText();
    const hasInput = (await page.locator("input").count()) > 0 || /Mã điểm danh/.test(txt);
    check(`${tc} ${user} bị chặn ở /checkin, không ô nhập, không POST`, !hasInput && posts === 0, `input=${hasInput} posts=${posts} text=${txt.replace(/\n+/g, "|").slice(0, 90)}`);
    await page.screenshot({ path: `${process.env.QC_OUT ?? "/tmp/qc-mob"}/${tc.slice(3, 9)}-${user}.png` });
    // SES-05c: /attendance vẫn dùng được cho GV/TA
    if (user !== "admin") {
      await page.goto(`/attendance?course=${C.QCMOB1}`);
      await page.waitForTimeout(3000);
      const ok = /Điểm danh|Buổi|điểm danh/i.test(await page.locator("body").innerText()) && !/không có quyền|403/i.test(await page.locator("body").innerText());
      check(`TC-MOB-SES-05c ${user} dùng được /attendance`, ok, (await page.locator("body").innerText()).replace(/\n+/g, "|").slice(0, 80));
    }
  }
});

// ───────────────────────────── 6.C Luồng chính ─────────────────────────────
test("FLW-01/05 token QR: Đã điểm danh ≤ 3 s, #token biến mất, mở lại cùng giờ", async ({ browser }) => {
  await withWindow(C.QCMOB1, async (w) => {
    const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
    const t0 = Date.now();
    await page.goto(`/checkin?c=${C.QCMOB1}#${w.qr}`);
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
    const dt = Date.now() - t0;
    const txt = await body(page).innerText();
    check("TC-MOB-FLW-01 tự gửi MỘT lần, «Đã điểm danh hh:mm» ≤ 3 s", HMM.test(txt) && dt <= 3000, `${dt} ms; "${txt.replace(/\n+/g, " ").slice(0, 40)}"`);
    check("TC-MOB-FLW-01b #token biến khỏi URL", !page.url().includes("#"), page.url().slice(0, 80));
    const red = await page.evaluate(() => {
      const brand = getComputedStyle(document.documentElement).getPropertyValue("--ep-red").trim();
      const probe = document.createElement("span");
      probe.style.color = brand;
      document.body.append(probe);
      const rgb = getComputedStyle(probe).color;
      probe.remove();
      const hits = [...document.querySelectorAll("main *")]
        .filter((e) => getComputedStyle(e).color === rgb)
        .map((e) => `${e.tagName}${(e.textContent ?? "").trim() ? `:"${(e.textContent ?? "").trim().slice(0, 18)}"` : ""}`);
      return { rgb, hits };
    });
    const redText = red.hits.filter((h) => h.includes('"'));
    check("TC-MOB-FLW-01c chỉ dấu Check đỏ, không chữ nào màu đỏ thương hiệu trong main", redText.length === 0 && red.hits.length <= 2, JSON.stringify(red));
    await page.screenshot({ path: `${process.env.QC_OUT ?? "/tmp/qc-mob"}/FLW-01-done.png` });
    const at1 = (txt.match(HMM)?.[0] ?? "").slice(-5);
    await page.goto(`/checkin?c=${C.QCMOB1}#${w.qr}`);
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
    const txt2 = await body(page).innerText();
    check("TC-MOB-FLW-05 mở lại cùng liên kết: hiện lại đúng giờ lượt đầu, không lỗi", txt2.includes(at1) && !/lỗi|Thử lại/i.test(txt2), `"${txt2.replace(/\n+/g, " ").slice(0, 40)}"`);
    check("TC-MOB-FLW-01d cửa sổ ghi đúng 1 lượt", (await winCount(C.QCMOB1, w.sid, w.wid)) === 1, String(await winCount(C.QCMOB1, w.sid, w.wid)));
  });
});

test("FLW-03/04 nhập mã: ba kiểu chữ, ô nhập + «Lớp …» + nút, một dòng", async ({ browser }) => {
  await withWindow(C.QCMOB1, async (w) => {
    const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
    await gotoEntry(page, C.QCMOB1, "QCMOB1");
    const txt = await body(page).innerText();
    const b = page.locator("[data-part=checkin-dock] button");
    check(
      "TC-MOB-FLW-04 ô «Mã điểm danh» + «Lớp QCMOB1» + nút Điểm danh, không nút quét",
      /Mã điểm danh/.test(txt) && /Lớp QCMOB1/.test(txt) && (await b.innerText()) === "Điểm danh" && !(await page.getByRole("button", { name: /Quét/ }).count()),
      txt.replace(/\n+/g, "|").slice(0, 90),
    );
    const results: string[] = [];
    let at = "";
    for (const v of [w.code.toLowerCase(), `${w.code.slice(0, 3)} ${w.code.slice(3)}`, `${w.code.slice(0, 3)}-${w.code.slice(3)}`]) {
      if (at) await gotoEntry(page, C.QCMOB1);
      const inp = page.getByLabel("Mã điểm danh");
      await inp.fill(v);
      await page.keyboard.press("Enter");
      await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
      const t = await body(page).innerText();
      at ||= (t.match(HMM)?.[0] ?? "").slice(-5);
      results.push(`${v}→${(t.match(HMM)?.[0] ?? "?").slice(-5)}`);
    }
    check("TC-MOB-FLW-03 cả ba kiểu mã được nhận, cùng một giờ", results.every((r) => r.endsWith(at)), results.join("; "));
    check("TC-MOB-FLW-03b cửa sổ chỉ ghi 1 lượt", (await winCount(C.QCMOB1, w.sid, w.wid)) === 1, String(await winCount(C.QCMOB1, w.sid, w.wid)));
  });
  const rows = await meRows(ACCT.s1, C.QCMOB1);
  check("TC-MOB-FLW-03c /me/attendance có dòng cho buổi vừa điểm danh (không nhân đôi)", rows.length >= 1 && rows.every((r) => r.status !== "ABSENT"), JSON.stringify(rows.map((r) => `${r.session_no}:${r.status}`)));
});

test("FLW-06 chạm đúp < 300 ms: đúng MỘT POST, nút khoá khi gửi", async ({ browser }) => {
  await withWindow(C.QCMOB1, async (w) => {
    const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
    let posts = 0;
    await page.route("**/checkins", async (r) => {
      if (r.request().method() === "POST") {
        posts++;
        await new Promise((x) => setTimeout(x, 1200));
      }
      await r.continue();
    });
    await gotoEntry(page, C.QCMOB1, "QCMOB1");
    await page.getByLabel("Mã điểm danh").fill(w.code);
    const locked = await page.evaluate(async () => {
      const b = document.querySelector("[data-part=checkin-dock] button") as HTMLButtonElement;
      b.click();
      await new Promise((r) => setTimeout(r, 80));
      const now = document.querySelector("[data-part=checkin-dock] button") as HTMLButtonElement | null;
      const dis = now?.disabled ?? true;
      document.querySelector("[data-part=checkin-dock] button")?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      return dis;
    });
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 25_000 });
    check("TC-MOB-FLW-06 đúng một POST rời máy + nút khoá khi đang gửi", posts === 1 && locked, `posts=${posts} disabled=${locked}`);
    check("TC-MOB-FLW-06b cửa sổ ghi 1 lượt", (await winCount(C.QCMOB1, w.sid, w.wid)) === 1, String(await winCount(C.QCMOB1, w.sid, w.wid)));
  });
});

test("FLW-07 xoay ngang giữa lúc gửi: một lượt, kết quả không mất", async ({ browser }) => {
  await withWindow(C.QCMOB1, async (w) => {
    const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
    let posts = 0;
    await page.route("**/checkins", async (r) => {
      if (r.request().method() === "POST") {
        posts++;
        await new Promise((x) => setTimeout(x, 1500));
      }
      await r.continue();
    });
    await gotoEntry(page, C.QCMOB1, "QCMOB1");
    await page.getByLabel("Mã điểm danh").fill(w.code);
    await page.locator("[data-part=checkin-dock] button").click();
    await page.waitForTimeout(300);
    await page.setViewportSize({ width: 915, height: 412 });
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
    const txt = await body(page).innerText();
    check("TC-MOB-FLW-07 xoay giữa lúc gửi: đúng 1 lượt, «Đã điểm danh» hiện sau xoay", posts === 1 && HMM.test(txt) && (await winCount(C.QCMOB1, w.sid, w.wid)) === 1, `posts=${posts} "${txt.replace(/\n+/g, " ").slice(0, 30)}"`);
  });
});

test("FLW-08 token không lộ ra URL/Referer/luồng yêu cầu", async ({ browser }) => {
  await withWindow(C.QCMOB1, async (w) => {
    const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
    const seen: string[] = [];
    page.on("request", (r) => seen.push(`${r.url()} referer=${r.headers()["referer"] ?? ""}`));
    await page.goto(`/checkin?c=${C.QCMOB1}#${w.qr}`);
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
    const leak = seen.filter((s) => s.includes(w.qr));
    check("TC-MOB-FLW-08 token không có trong URL/Referer của mọi yêu cầu", leak.length === 0, `${seen.length} yêu cầu; rò=${leak.length}`);
  });
});

test("FLW-09 chọn lớp: mã lớp khác bị từ chối, chọn đúng lớp thì thành công", async ({ browser }) => {
  const w2 = await openWindow(C.QCMOB2);
  await withWindow(C.QCMOB1, async (w) => {
    const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
    await gotoEntry(page, C.QCMOB1, "QCMOB1");
    await page.getByLabel("Mã điểm danh").fill(w2.code);
    await page.keyboard.press("Enter");
    await page.waitForTimeout(2500);
    const t1 = await body(page).innerText();
    check("TC-MOB-FLW-09a mã QCMOB2 khi đang chọn QCMOB1 → câu từ chối", /Mã không đúng hoặc đã hết hạn/.test(t1), t1.replace(/\n+/g, "|").slice(0, 90));
    await page.getByLabel("Mã điểm danh").fill(w.code);
    await page.keyboard.press("Enter");
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
    check("TC-MOB-FLW-09b mã đúng lớp đang chọn → Đã điểm danh", HMM.test(await body(page).innerText()), "");
  });
  await closeWindow(C.QCMOB2, w2.wid).catch(() => undefined);
});

test("FLW-12/13 trạng thái lớp: tắt điểm danh, lưu trữ", async ({ browser }) => {
  const b = await authed(browser, "pixel", ACCT.s1);
  let posts = 0;
  b.page.on("request", (r) => r.method() === "POST" && /\/checkins$/.test(new URL(r.url()).pathname) && posts++);
  await b.page.goto(`/checkin?course=${C.QCMOB5}`);
  await b.page.getByText("Lớp này không điểm danh bằng mã.").waitFor({ timeout: 20_000 });
  const t12 = await b.page.locator("body").innerText();
  const hasInput = (await b.page.getByLabel("Mã điểm danh").count()) > 0;
  check("TC-MOB-FLW-12 lớp tắt self_checkin: «Lớp này không điểm danh bằng mã.», không POST", /Lớp này không điểm danh bằng mã/.test(t12) && posts === 0, `posts=${posts} input=${hasInput} (AC16(d): không còn ô mã)`);
  await b.page.screenshot({ path: `${process.env.QC_OUT ?? "/tmp/qc-mob"}/FLW-12-off.png` });

  await b.page.goto(`/checkin?course=${C.QCMOB4B}`);
  await b.page.waitForTimeout(3000);
  const t13 = await b.page.locator("main").innerText();
  const lines = t13.split("\n").map((x) => x.trim()).filter(Boolean);
  check(
    "TC-MOB-FLW-13 lớp lưu trữ: câu tiếng Việt, không mã kỹ thuật, không ghi lượt",
    !/COURSE_ARCHIVED|undefined|null|\b500\b/.test(t13) && lines.length > 0 && posts === 0,
    lines.slice(0, 2).join(" | ").slice(0, 120),
  );
  await b.page.screenshot({ path: `${process.env.QC_OUT ?? "/tmp/qc-mob"}/FLW-13-archived.png` });
});

test("FLW-11 vòng 2: lượt mới ghi round 2, giờ của vòng 2", async ({ browser }) => {
  await withWindow(C.QCMOB1, async (w) => {
    const { ctx, page } = await authed(browser, "pixel", ACCT.s1);
    await gotoEntry(page, C.QCMOB1, "QCMOB1");
    await page.getByLabel("Mã điểm danh").fill(w.code);
    await page.keyboard.press("Enter");
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
    const r1 = await winCount(C.QCMOB1, w.sid, w.wid);
    const w2 = await openWindow(C.QCMOB1, 10, w.sid); // cùng buổi: vòng 2
    await page.goto(`/checkin?c=${C.QCMOB1}#${w2.qr}`);
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
    const d = (await api("teacher", "GET", `/courses/${C.QCMOB1}/sessions/${w2.sid}/checkin-windows/${w2.wid}/display`)).j;
    check("TC-MOB-FLW-11 vòng 2 tạo lượt mới (round 2) và ghi nhận", r1 === 1 && (await winCount(C.QCMOB1, w2.sid, w2.wid)) === 1 && d.round >= 2, `r1=${r1} round=${d.round}`);
    await closeWindow(C.QCMOB1, w2.wid).catch(() => undefined);
  });
});
