import { expect, test, type Page, type Route } from "@playwright/test";
import { CHECKED_IN_AT, COURSE, TOKEN, checkinApi, json } from "./support/checkin-fixtures";

// US-AS-15 (P-11-130): /checkin dùng được trên mọi điện thoại. Playwright giả lập bề rộng / hướng; không giả được bàn phím ảo, tai thỏ thật
// (đó là phần kiểm tay trên máy thật của QC, android-checklist.md). Đo trên project desktop (tự đặt viewport).
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt viewport");
});

const GOOD = `ORIGIN/checkin?c=${COURSE}#${TOKEN}`;
/** App giả: Capacitor + camera canvas + BarcodeDetector (canvas = phép thử khả dụng; video = khung camera lấy từ hàng đợi `window.__qr`). */
const NATIVE = (opts: { gum?: string; repeat?: string } = {}) => `(() => {
  window.__stopped = 0; window.__gum = 0; window.__qr = [];
  window.Capacitor = { isNativePlatform: () => true, isPluginAvailable: () => false };
  navigator.mediaDevices.getUserMedia = async () => {
    window.__gum++;
    const fail = window.__gumFail ?? ${JSON.stringify(opts.gum ?? "")};
    if (fail) throw new DOMException("x", fail);
    const c = document.createElement("canvas"); c.width = c.height = 16;
    const s = c.captureStream(5); c.getContext("2d").fillRect(0, 0, 16, 16);
    s.getTracks().forEach((t) => { const st = t.stop.bind(t); t.stop = () => { window.__stopped++; st(); }; });
    return s;
  };
  window.BarcodeDetector = class {
    static async getSupportedFormats() { return ["qr_code"]; }
    async detect(src) {
      if (src instanceof HTMLCanvasElement) return [{ rawValue: "probe" }];
      const rep = ${JSON.stringify(opts.repeat ?? "")}.replace("ORIGIN", location.origin);
      if (rep) return [{ rawValue: rep }];
      const v = window.__qr.shift(); return v ? [{ rawValue: v.replace("ORIGIN", location.origin) }] : [];
    }
  };
})()`;

const SIZES = [[360, 740], [390, 844], [411, 842], [430, 932]] as const;
const LANDSCAPE = [[667, 375], [844, 390], [915, 412]] as const;

type St = "code" | "native" | "sending" | "done" | "scanning" | "error" | "limited";
async function reach(page: Page, context: Parameters<typeof checkinApi>[1], st: St) {
  const never = () => new Promise<void>(() => undefined);
  const post = st === "sending" ? never : st === "limited" ? (r: Route) => r.fulfill({ status: 429, headers: { "Retry-After": "30", "content-type": "application/json" }, body: JSON.stringify({ code: "RATE_LIMITED", message: "m", trace_id: "c".repeat(32) }) }) : st === "error" ? (r: Route) => r.abort("connectionfailed") : undefined;
  if (st === "native" || st === "scanning" || st === "error") await page.addInitScript(NATIVE());
  const log = await checkinApi(page, context, post ? { post } : {});
  if (st === "code" || st === "native" || st === "scanning") await page.goto("/checkin");
  else await page.goto(`/checkin?c=${COURSE}#${TOKEN}`);
  if (st === "code") await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
  if (st === "native") await expect(page.getByRole("button", { name: "Quét mã điểm danh" }).first()).toBeVisible();
  if (st === "scanning") {
    await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
    await expect(page.getByRole("button", { name: "Thoát" })).toBeVisible();
  }
  if (st === "done") await expect(page.locator("[data-part=done]")).toBeVisible();
  if (st === "error") await expect(page.getByText("Chưa có mạng.")).toBeVisible();
  if (st === "limited") await expect(page.locator("[data-part=limited]")).toBeVisible();
  if (st === "sending") await expect(page.locator("[data-part=checkin][data-state=sending]")).toBeVisible();
  return log;
}

const STATES: St[] = ["code", "native", "sending", "done", "scanning", "error", "limited"];

// AC1 + AC13: 4 bề rộng × 7 trạng thái — không cuộn ngang, nút chính trong khung nhìn, mọi điều khiển ở nội dung + thanh dưới ≥ 44 px, kề nhau ≥ 8 px
for (const [w, h] of SIZES) {
  for (const st of STATES) {
    test(`checkin mobile @${w}: ${st} — không cuộn ngang, nút chính trong màn, vùng chạm ≥ 44 px`, async ({ page, context }) => {
      await page.setViewportSize({ width: w, height: h });
      await reach(page, context, st);
      const m = await page.evaluate(() => {
        const r = (e: Element) => e.getBoundingClientRect();
        const vis = (e: Element) => { const b = r(e); const cs = getComputedStyle(e); return b.width > 0 && b.height > 0 && cs.visibility !== "hidden"; };
        const ctl = [...document.querySelectorAll("main button, main a, main input, [data-part=bottom-nav] a, [data-part=bottom-nav] button, [data-part=checkin-dock] button")].filter(vis);
        const small = ctl.filter((e) => r(e).width < 43.5 || r(e).height < 43.5).map((e) => `${e.tagName}:${(e.textContent || e.getAttribute("aria-label") || "").trim().slice(0, 20)}:${Math.round(r(e).width)}x${Math.round(r(e).height)}`);
        const dock = document.querySelector("[data-part=checkin-dock] button");
        const nav = document.querySelector("[data-part=bottom-nav]");
        const dockBox = dock ? r(dock) : null;
        return {
          sw: document.documentElement.scrollWidth, iw: innerWidth, small,
          dockIn: dockBox ? dockBox.top >= 0 && dockBox.bottom <= innerHeight : true,
          dockOverNav: dockBox && nav ? dockBox.bottom > r(nav).top + 0.5 : false,
          contentFits: document.documentElement.scrollHeight <= innerHeight + 1,
        };
      });
      expect(m.sw, "cuộn ngang").toBeLessThanOrEqual(m.iw);
      expect(m.small, "điều khiển < 44 px").toEqual([]);
      expect(m.dockIn, "nút chính trong khung nhìn").toBe(true);
      expect(m.dockOverNav, "dock đè thanh dưới").toBe(false);
      if (st !== "scanning") expect(m.contentFits, "một màn, không cuộn dọc").toBe(true);
    });
  }
}

// AC2: ngang — không tràn, khung camera co theo chiều cao khả dụng, Thoát / nút chính không bị thanh dưới che
for (const [w, h] of LANDSCAPE) {
  for (const st of ["code", "scanning"] as const) {
    test(`checkin landscape @${w}x${h}: ${st} — không tràn, camera trong khung nhìn, nút không bị thanh dưới che`, async ({ page, context }) => {
      await page.setViewportSize({ width: w, height: h });
      await reach(page, context, st);
      const m = await page.evaluate(() => {
        const r = (s: string) => document.querySelector(s)?.getBoundingClientRect();
        const nav = r("[data-part=bottom-nav]");
        const exit = [...document.querySelectorAll("button")].find((b) => b.textContent?.trim() === "Thoát")?.getBoundingClientRect();
        const dock = r("[data-part=checkin-dock] button");
        const video = r("[data-part=scanner] video");
        const navTop = nav && nav.height > 0 ? nav.top : innerHeight;
        return { sw: document.documentElement.scrollWidth, iw: innerWidth, videoBottom: video?.bottom ?? 0, exitBottom: exit?.bottom ?? 0, dockBottom: dock?.bottom ?? 0, navTop, ih: innerHeight };
      });
      expect(m.sw).toBeLessThanOrEqual(m.iw);
      expect(m.dockBottom).toBeLessThanOrEqual(m.navTop + 0.5);
      if (st === "scanning") {
        expect(m.videoBottom, "khung camera tràn đáy").toBeLessThanOrEqual(m.navTop);
        expect(m.exitBottom, "Thoát bị che").toBeLessThanOrEqual(m.navTop);
      }
    });
  }
}

// AC3: viewport-fit=cover (CSS đã dùng env(safe-area-inset-*)) + thanh trên chừa vùng an toàn trên / ngang
test("checkin: viewport-fit=cover; thanh trên và khung chừa vùng an toàn", async ({ page, context }) => {
  await checkinApi(page, context);
  await page.goto("/checkin");
  await expect(page.locator('meta[name=viewport]')).toHaveAttribute("content", /viewport-fit=cover/);
  const css = await page.evaluate(async () => {
    const t = document.querySelector("header, [data-part=topbar]");
    return t ? getComputedStyle(t).paddingTop : "none";
  });
  expect(css).toBeTruthy();
  const src = await page.evaluate(() => [...document.styleSheets].flatMap((s) => { try { return [...s.cssRules].map((r) => r.cssText); } catch { return []; } }).join("\n"));
  expect(src, "thanh trên dùng safe-area-inset-top").toMatch(/safe-area-inset-top/);
  expect(src, "lề ngang dùng safe-area-inset-left/right").toMatch(/safe-area-inset-(left|right)/);
});

// AC14: lỗi camera tiếng Việt, không lộ tên lỗi kỹ thuật
for (const [err, line] of [["NotAllowedError", "Chưa cấp quyền camera."], ["NotFoundError", "Không mở được camera."], ["NotReadableError", "Không mở được camera."], ["AbortError", "Không mở được camera."]] as const) {
  test(`checkin camera ${err} ⇒ «${line}» + ô nhập mã, không lộ mã kỹ thuật`, async ({ page, context }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.addInitScript(NATIVE({ gum: err }));
    await checkinApi(page, context);
    await page.goto("/checkin");
    await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
    await expect(page.getByText(line, { exact: true })).toBeVisible();
    await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
    const body = await page.locator("body").innerText();
    expect(body).not.toMatch(/NotAllowed|NotFound|NotReadable|AbortError|undefined|\[object/);
  });
}

// AC7a: Back đang quét ⇒ về ô nhập mã, camera tắt, vẫn ở /checkin
test("checkin: Back đang quét thoát khung quét về màn nhập mã, tắt camera, không rời /checkin", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.addInitScript(NATIVE());
  await checkinApi(page, context);
  await page.goto("/checkin");
  await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
  await expect(page.getByRole("button", { name: "Thoát" })).toBeVisible();
  await page.goBack();
  await expect(page.getByRole("button", { name: "Thoát" })).toHaveCount(0);
  await expect(page).toHaveURL(/\/checkin$/);
  await expect(page.getByRole("button", { name: "Quét mã điểm danh" }).first()).toBeVisible();
  expect(await page.evaluate(() => (window as unknown as { __stopped: number }).__stopped)).toBeGreaterThan(0);
});

// AC8: bị từ chối rồi cấp lại ở Cài đặt, quay lại trang ⇒ nút quét trở lại, dòng «Chưa cấp quyền camera.» biến mất, không cần tải lại
test("checkin: từ chối camera rồi cấp lại — trang hiện lại (visibilitychange) ⇒ nút «Quét mã điểm danh» trở lại, hết dòng từ chối", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.addInitScript(NATIVE({ gum: "NotAllowedError" }));
  await checkinApi(page, context);
  await page.goto("/checkin");
  await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
  await expect(page.getByText("Chưa cấp quyền camera.")).toBeVisible();
  expect(await page.evaluate(() => (window as unknown as { __gum: number }).__gum)).toBe(1);
  await page.evaluate(() => {
    (window as unknown as { __gumFail: string }).__gumFail = "";
    Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await expect(page.getByText("Chưa cấp quyền camera.")).toHaveCount(0);
  await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
  await expect(page.getByRole("button", { name: "Thoát" })).toBeVisible();
  expect(await page.evaluate(() => (window as unknown as { __gum: number }).__gum), "không tự xin lại quyền khi chỉ hiện lại trang").toBe(2);
});

// AC9: mã quét lặp lại → đúng một POST; mất mạng → «Chưa có mạng.» + Thử lại gửi đúng token, không đăng xuất
test("checkin: quét cùng mã nhiều lần ⇒ một POST; mất mạng ⇒ Thử lại gửi lại đúng token một lần", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.addInitScript(NATIVE({ repeat: GOOD }));
  let fail = true;
  const log = await checkinApi(page, context, { post: (r) => (fail ? r.abort("connectionfailed") : r.fulfill(json({ checked_in_at: "2026-10-07T00:03:12.000Z", round: 1 }, 201))) });
  await page.goto("/checkin");
  await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
  await expect(page.getByText("Chưa có mạng.")).toBeVisible();
  await page.waitForTimeout(800);
  // apiClient tự gửi lại lỗi mạng với CÙNG Idempotency-Key: «một POST» là một khoá, dù detector còn thấy mã
  const keys = () => new Set(log.posts.map((p) => p.headers["idempotency-key"]));
  expect(keys().size, "một lượt gửi cho mỗi token dù detector còn thấy mã").toBe(1);
  fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
  expect(log.posts.at(-1)!.body.token).toBe(TOKEN);
  expect(log.posts.filter((p) => p.headers["idempotency-key"] === log.posts.at(-1)!.headers["idempotency-key"]).length).toBeGreaterThanOrEqual(1);
  expect(page.url()).not.toContain("/login");
});

// AC12: kết quả báo MỘT lần qua vùng aria-live
test("checkin: «Đã điểm danh» nằm trong vùng aria-live polite", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await checkinApi(page, context);
  await page.goto(`/checkin?c=${COURSE}#${TOKEN}`);
  await expect(page.locator('[data-part=done]')).toBeVisible();
  const live = await page.locator("[data-part=done]").evaluate((e) => !!e.closest('[aria-live="polite"], [role="status"]'));
  expect(live).toBe(true);
});

// AC12 (DEF-MOB-15): «Đang gửi» là chữ thật; lỗi là role=alert (không lồng trong vùng polite); đếm ngược 429 không làm vùng thông báo đổi mỗi giây
test("checkin AC12: đang gửi nói «Đang gửi», không còn nhãn «Đang tải»", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await reach(page, context, "sending");
  await expect(page.getByRole("status").filter({ hasText: /^Đang gửi$/ })).toHaveCount(1);
  await expect(page.locator('[data-part=checkin] [aria-label="Đang tải"]')).toHaveCount(0);
});

test("checkin AC12: câu lỗi là role=alert, ở ô nhập mã và ở màn token, không nằm trong vùng aria-live", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await checkinApi(page, context, { post: (r) => r.fulfill(json({ code: "CHECKIN_TOKEN_INVALID", message: "m", trace_id: "d".repeat(32) }, 400)) });
  await page.goto("/checkin");
  await page.getByLabel("Mã điểm danh").fill("ABC123");
  await page.getByRole("button", { name: "Điểm danh" }).click();
  const a = page.getByRole("alert").filter({ hasText: "Mã không đúng hoặc đã hết hạn." });
  await expect(a).toHaveCount(1);
  expect(await a.evaluate((e) => !!e.parentElement?.closest('[aria-live="polite"]'))).toBe(false);
});

test("checkin AC12: lỗi mạng ở màn token là role=alert, không nằm trong vùng aria-live", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await reach(page, context, "error");
  const b = page.getByRole("alert").filter({ hasText: "Chưa có mạng." });
  await expect(b).toHaveCount(1);
  expect(await b.evaluate((e) => !!e.closest('[aria-live="polite"]'))).toBe(false);
});

test("checkin AC12: đếm ngược 429 không làm vùng thông báo đổi mỗi giây; báo lúc bắt đầu và lúc hết", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  let n = 0;
  await checkinApi(page, context, { post: (r) => (++n === 1 ? r.fulfill(json({ code: "RATE_LIMITED", message: "m", trace_id: "e".repeat(32), retry_after: 4 }, 429)) : r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 201))) });
  await page.goto("/checkin");
  await page.getByLabel("Mã điểm danh").fill("ABC123");
  await page.getByRole("button", { name: "Điểm danh" }).click();
  const live = page.getByRole("status").filter({ hasText: /^Thử lại sau 4 giây\.$/ });
  await expect(live).toHaveCount(1);
  const muts = await page.evaluate(
    (ms) =>
      new Promise<number>((res) => {
        let c = 0;
        const sel = '[aria-live]:not([aria-live=off]), [role=status], [role=alert]';
        const mo = new MutationObserver((list) => {
          for (const m of list) if ((m.target as Node).parentElement?.closest(sel) || (m.target as Element).closest?.(sel)) c++;
        });
        mo.observe(document.body, { subtree: true, childList: true, characterData: true });
        setTimeout(() => (mo.disconnect(), res(c)), ms);
      }),
    2000,
  );
  expect(muts).toBe(0);
  await expect(page.getByRole("status").filter({ hasText: /^Thử lại$/ })).toHaveCount(1, { timeout: 5000 });
});

test("checkin AC12 (DEF-MOB-15b): đếm ngược 429 — cây trợ năng không đổi trong 3 s (không nút nào đổi tên / mô tả, không nút nào đổi nội dung ngoài aria-hidden)", async ({ page, context }) => {
  for (const [w, h] of [[390, 844], [1440, 900]] as const) {
    await page.setViewportSize({ width: w, height: h });
    await checkinApi(page, context, { post: (r) => r.fulfill(json({ code: "RATE_LIMITED", message: "m", trace_id: "f".repeat(32), retry_after: 9 }, 429)) });
    await page.goto("/checkin");
    await page.getByLabel("Mã điểm danh").fill("ABC123");
    await page.getByRole("button", { name: "Điểm danh" }).last().click();
    await expect(page.locator("[data-part=limited]")).toBeVisible();
    await expect(page.locator("[data-part=limited]")).toHaveText(/Thử lại sau [1-9] giây\./); // chữ nhìn thấy vẫn có
    const before = await page.locator("body").ariaSnapshot();
    const muts = await page.evaluate(
      (ms) =>
        new Promise<string[]>((res) => {
          const out: string[] = [];
          const hidden = (n: Node) => !!(n instanceof Element ? n : n.parentElement)?.closest('[aria-hidden="true"]');
          const mo = new MutationObserver((list) => {
            for (const m of list) {
              if (m.type === "attributes" && !/^aria-|^title$|^role$/.test(m.attributeName ?? "")) continue;
              if (!hidden(m.target)) out.push(`${m.type}:${m.attributeName ?? ""}:${(m.target as Element).nodeName ?? ""}`);
            }
          });
          mo.observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true });
          setTimeout(() => (mo.disconnect(), res(out)), ms);
        }),
      3000,
    );
    expect(muts, `${w}px`).toEqual([]);
    expect(await page.locator("body").ariaSnapshot(), `${w}px`).toBe(before);
  }
});

test("checkin AC12 (DEF-MOB-15c): đổi trạng thái (429, lỗi, xong) không để focus rơi về body — focus nằm trong khu checkin hoặc ở nút còn lại", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const focusOk = () => page.evaluate(() => { const a = document.activeElement; return !!a && a !== document.body && !!a.closest("[data-part=checkin], [data-part=checkin-dock]"); });
  for (const [status, body, wait] of [[429, { code: "RATE_LIMITED", message: "m", trace_id: "a".repeat(32), retry_after: 9 }, "[data-part=limited]"], [400, { code: "CHECKIN_TOKEN_INVALID", message: "m", trace_id: "a".repeat(32) }, "[role=alert]"], [201, { checked_in_at: CHECKED_IN_AT, round: 1 }, "[data-part=done]"]] as const) {
    await checkinApi(page, context, { post: (r) => r.fulfill(json(body, status)) });
    await page.goto("/checkin");
    await page.getByLabel("Mã điểm danh").fill("ABC123");
    await page.getByRole("button", { name: "Điểm danh" }).last().click();
    await expect(page.locator(wait).first()).toBeVisible();
    expect(await focusOk(), `HTTP ${status}`).toBe(true);
  }
});

// AC11 + AC5: PWA iPhone — biểu tượng, tên, toàn màn, thanh trạng thái khớp thanh trên trắng
test("checkin: có apple-touch-icon, tên «EduPilot», chạy toàn màn (iOS) và theme-color khớp thanh trên", async ({ page, context }) => {
  await checkinApi(page, context);
  await page.goto("/checkin");
  await expect(page.locator('link[rel="apple-touch-icon"]')).toHaveAttribute("href", /icon-192\.png/);
  await expect(page.locator('meta[name="apple-mobile-web-app-title"]')).toHaveAttribute("content", "EduPilot");
  await expect(page.locator('meta[name="apple-mobile-web-app-status-bar-style"]')).toHaveAttribute("content", "default");
  expect(await page.locator('meta[name$="mobile-web-app-capable"]').count()).toBeGreaterThan(0);
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute("content", /^#fff(fff)?$/i);
  const m = await (await page.request.get("/manifest.webmanifest")).json();
  expect(m.display).toBe("standalone");
  expect(m.theme_color.toLowerCase()).toBe("#ffffff");
});

// AC3 (đo thật): Chromium cho giả lập vùng an toàn (CDP) — tai thỏ trên 47, thanh cử chỉ dưới 34; ngang: tai thỏ trái / phải 44.
// Một cách duy nhất cho PWA iOS và app Android: viewport-fit=cover + env(safe-area-inset-*) ở khung; Capacitor SystemBars `insetsHandling: "css"` cấp giá trị inset.
async function withInsets(page: Page, insets: { top: number; bottom: number; left: number; right: number }) {
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("Emulation.setSafeAreaInsetsOverride", { insets });
}
test("checkin @390 dọc có tai thỏ + thanh cử chỉ: thanh trên, thanh dưới và dock chừa vùng an toàn", async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await withInsets(page, { top: 47, bottom: 34, left: 0, right: 0 });
  await checkinApi(page, context);
  await page.goto("/checkin");
  await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
  await page.getByLabel("Mã điểm danh").fill("ABC123");
  const m = await page.evaluate(() => {
    const r = (s: string) => document.querySelector(s)!.getBoundingClientRect();
    const top = r("header, [data-part=topbar]");
    return { topbarH: Math.round(top.height), brand: Math.round(r("header a, header button").top), nav: Math.round(r("[data-part=bottom-nav]").height), navBottom: Math.round(r("[data-part=bottom-nav]").bottom), dockBottom: Math.round(r("[data-part=checkin-dock] button").bottom), navTop: Math.round(r("[data-part=bottom-nav]").top), ih: innerHeight };
  });
  expect(m.topbarH).toBeGreaterThanOrEqual(56 + 47);
  expect(m.brand, "điều khiển trên cùng không nằm dưới tai thỏ").toBeGreaterThanOrEqual(47);
  expect(m.nav).toBeGreaterThanOrEqual(56 + 34);
  expect(m.dockBottom, "dock nằm trên thanh dưới (đã gồm vùng an toàn)").toBeLessThanOrEqual(m.navTop);
});
test("checkin ngang @844x390 có tai thỏ trái / phải 44: nội dung và thanh dưới lùi vào", async ({ page, context }) => {
  await page.setViewportSize({ width: 667, height: 375 });
  await withInsets(page, { top: 0, bottom: 21, left: 44, right: 44 });
  await checkinApi(page, context);
  await page.goto("/checkin");
  await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
  const m = await page.evaluate(() => {
    const els = [...document.querySelectorAll("main button, main input, h1, [data-part=checkin-dock] button")].map((e) => e.getBoundingClientRect()).filter((b) => b.width > 0 && b.height > 0); // bỏ phần tử ẩn (nút chính ở đầu trang ẩn khi dock hiện)
    return { minLeft: Math.round(Math.min(...els.map((b) => b.left))), maxRight: Math.round(Math.max(...els.map((b) => b.right))), iw: innerWidth, sw: document.documentElement.scrollWidth };
  });
  expect(m.minLeft).toBeGreaterThanOrEqual(44);
  expect(m.maxRight).toBeLessThanOrEqual(m.iw - 44);
  expect(m.sw).toBeLessThanOrEqual(m.iw);
});
