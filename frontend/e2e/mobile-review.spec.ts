import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { checkinApi } from "./support/checkin-fixtures";
import { practiceApi } from "./support/practice-fixtures";
import { attApi } from "./support/att-fixtures";
import { chatApi } from "./support/chat-fixtures";
import { asDemo } from "./support/session";

// Soát giao diện điện thoại 25/11/2026 (docs/design/review/2026-11-25-mobile.md): M1, M2, M3, M6, M12. Desktop project, tự đặt viewport.
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt viewport");
});
// màu tính toán có thể là rgb() hoặc lab(): quy về RGB bằng canvas rồi so kênh đỏ với xanh lá
const toRgb = (bg: string) => { const c = document.createElement("canvas"); c.width = c.height = 1; const g = c.getContext("2d")!; g.fillStyle = bg; g.fillRect(0, 0, 1, 1); const d = g.getImageData(0, 0, 1, 1).data; return [d[0], d[1]]; };
const isRed = (page: Page, bg: string) => page.evaluate(`(${toRgb.toString()})(${JSON.stringify(bg)})`).then((v) => { const [r, g] = v as number[]; return r - g >= 40; });

for (const w of [360, 375, 390, 430]) {
  test(`M1 @${w}: thanh trên hiện TÊN lớp, không chỉ mã`, async ({ page, context }) => {
    await page.setViewportSize({ width: w, height: 812 });
    await checkinApi(page, context);
    await page.goto("/checkin");
    const pick = page.getByRole("button", { name: /^Chọn lớp/ });
    expect(await pick.evaluate((e) => (e as HTMLElement).innerText)).toContain("An ninh mạng"); // innerText: chỉ chữ nhìn thấy
    expect(await pick.evaluate((e) => (e as HTMLElement).innerText)).not.toContain("761988");
    expect(await pick.evaluate((e) => e.scrollWidth <= e.clientWidth + 1)).toBe(true);
  });
}

for (const w of [375, 430]) {
  test(`M6 @${w}: nút chính bị khoá nền trung tính, không đỏ mờ`, async ({ page, context }) => {
    await page.setViewportSize({ width: w, height: 812 });
    await checkinApi(page, context);
    await page.goto("/checkin");
    const btn = page.locator("[data-part=checkin-dock] button");
    await expect(btn).toBeDisabled();
    await page.waitForTimeout(400); // chuyển màu 150 ms xong
    const st = await btn.evaluate((e) => { const c = getComputedStyle(e); return { bg: c.backgroundColor, op: c.opacity }; });
    expect(await isRed(page, st.bg), `nền ${st.bg}`).toBe(false);
    expect(st.op).toBe("1");
  });
}

// M2: điện thoại xoay ngang (máy cảm ứng, thấp) ⇒ khung dưới đáy, không thanh bên chỉ-icon; điều khiển của khung ≥ 44 px
test.describe("M2 điện thoại ngang 844×390", () => {
  test.use({ viewport: { width: 844, height: 390 }, hasTouch: true, isMobile: true });
  test("khung dưới đáy thay thanh bên, điều khiển khung ≥ 44 px", async ({ page, context }) => {
    await asDemo(context, "teacher");
    await attApi(page);
    await page.goto("/attendance");
    await expect(page.locator("[data-part=bottom-nav]")).toBeVisible();
    await expect(page.locator("#sidebar-nav")).toBeHidden();
    const small = await page.evaluate(() => [...document.querySelectorAll("header button, header a, [data-part=bottom-nav] a, [data-part=bottom-nav] button")]
      .map((e) => ({ t: (e.textContent || e.getAttribute("aria-label") || "").trim().slice(0, 18), w: Math.round(e.getBoundingClientRect().width), h: Math.round(e.getBoundingClientRect().height) }))
      .filter((b) => b.w > 0 && (b.w < 43.5 || b.h < 43.5)));
    expect(small).toEqual([]);
  });
});

// M3: màn tài khoản ở điện thoại ngang — form vào màn
for (const path of ["/login", "/register", "/forgot-password"]) {
  test(`M3 ${path} @844×390: ô đầu tiên trong 170 px đầu, nút chính trong khung nhìn`, async ({ page }) => {
    await page.setViewportSize({ width: 844, height: 390 });
    await page.goto(path);
    const first = await page.locator("main input").first().boundingBox();
    expect(first!.y).toBeLessThanOrEqual(path === "/forgot-password" ? 215 : 170);
    if (path === "/register") return; // nhiều trường: chỉ đòi ô đầu vào màn, cuộn tới nút
    const btn = await page.locator("main button[type=submit], main form button").last().boundingBox();
    expect(btn!.y + btn!.height).toBeLessThanOrEqual(390);
  });
}

// M12: mạng chậm — khung trước phiên có thanh trên + thanh đáy giữ chỗ + Skeleton; khi phiên về tiêu đề không nhảy chỗ
test("M12 @375: phiên về chậm ⇒ khung giữ thanh trên / đáy + Skeleton, h1 không dịch chỗ khi khung thật vào", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await asDemo(context, "student", { person: "sv-2" });
  await practiceApi(page);
  let release!: () => void;
  const gate = new Promise<void>((r) => (release = r));
  await context.route("**/api/v1/auth/refresh", async (r) => { await gate; await r.fallback(); });
  await page.goto("/practice");
  const pre = page.locator("[data-part=pre-shell]");
  await expect(pre).toBeVisible();
  await expect(page.locator("[data-part=pre-bottom]")).toBeVisible();
  await expect(pre.locator("[role=group][aria-busy=true]")).toBeVisible();
  expect(await pre.locator("header [data-ph]").count(), "thanh trên: ô giữ chỗ chọn lớp + 3 nút").toBe(4);
  expect(await page.locator("[data-part=pre-bottom] [data-ph]").count(), "thanh đáy: 5 mục giữ chỗ").toBe(10);
  const box = async (s: string) => (await page.locator(s).boundingBox())!;
  const y0 = (await pre.locator("h1").boundingBox())!.y;
  const bottom0 = await box("[data-part=pre-bottom]");
  const end0 = (await pre.locator("header i[data-ph]").last().boundingBox())!;
  release();
  await expect(page.locator("[data-part=bottom-nav]")).toBeVisible();
  await expect(page.locator("main h1")).toBeVisible();
  const y1 = (await page.locator("main h1").boundingBox())!.y;
  expect(Math.abs(y1 - y0)).toBeLessThanOrEqual(2);
  const bottom1 = await box("[data-part=bottom-nav]");
  expect([bottom1.y, bottom1.height]).toEqual([bottom0.y, bottom0.height]); // thanh đáy không dịch
  const prof = await box("[data-part=topbar] >> nth=0");
  expect(prof.height).toBe(56);
  expect(end0.y).toBeGreaterThanOrEqual(0);
});

// M9: /chat điện thoại — không khung Panel, ô soạn dính ngay trên thanh dưới (kể cả khi bàn phím co khung nhìn); ngang cũng vậy
async function openChat(page: Page, context: BrowserContext) {
  await asDemo(context, "student", { person: "sv-2", expIn: 366 * 86400 });
  await chatApi(page, {});
  await page.goto("/chat");
  await page.locator("[data-part=chat-thread]").waitFor();
}
const geom = (page: Page) =>
  page.evaluate(() => {
    const r = (s: string) => document.querySelector(s)!.getBoundingClientRect();
    const panel = document.querySelector("[data-part=chat-thread]")!.closest("[data-ep-panel]")!;
    const cs = getComputedStyle(panel);
    return { panelLeft: Math.round(panel.getBoundingClientRect().left), panelBorder: cs.borderTopWidth, composerBottom: Math.round(r("[data-part=chat-composer]").bottom), navTop: Math.round(r("[data-part=bottom-nav]").top), composerTop: Math.round(r("[data-part=chat-composer]").top), ih: innerHeight };
  });
test("M9 @375: Panel không khung, ô soạn dính đáy trên thanh dưới, bàn phím mở (khung nhìn thấp) vẫn dính", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await openChat(page, context);
  let g = await geom(page);
  expect(g.panelBorder).toBe("0px");
  expect(g.panelLeft).toBe(12); // vẫn trong cột 12 px của khung (frame.spec W3)
  expect(Math.abs(g.composerBottom - g.navTop)).toBeLessThanOrEqual(1);
  await page.setViewportSize({ width: 375, height: 420 }); // bàn phím ảo mở: khung nhìn co lại
  g = await geom(page);
  expect(Math.abs(g.composerBottom - g.navTop)).toBeLessThanOrEqual(1);
  expect(g.composerTop).toBeGreaterThan(56);
});
test.describe("M9 điện thoại ngang 844×390", () => {
  test.use({ viewport: { width: 844, height: 390 }, hasTouch: true, isMobile: true });
  test("ô soạn ở cuối luồng (không cố định); bàn phím mở (màn thấp) không đè thanh trên", async ({ page, context }) => {
    await openChat(page, context);
    expect(await page.locator("[data-part=chat-composer]").evaluate((e) => getComputedStyle(e).position)).toBe("static");
    for (const h of [390, 250]) { // 250: bàn phím ảo chiếm chỗ
      await page.setViewportSize({ width: 844, height: h });
      const m = await page.evaluate(() => {
        const c = document.querySelector("[data-part=chat-composer]")!.getBoundingClientRect();
        const t = document.querySelector("[data-part=topbar]")!.getBoundingClientRect();
        return { composerTop: Math.round(c.top), topbarBottom: Math.round(t.bottom), sw: document.documentElement.scrollWidth, iw: innerWidth };
      });
      expect(m.composerTop, `khung nhìn ${h}`).toBeGreaterThanOrEqual(m.topbarBottom);
      expect(m.sw).toBeLessThanOrEqual(m.iw);
    }
  });
  test("ô soạn đang focus trên cảm ứng ⇒ ẩn thanh điều hướng đáy, ô soạn sát đáy", async ({ page, context }) => {
    await page.setViewportSize({ width: 390, height: 844 }); // dọc: thanh đáy + ô soạn cố định
    await openChat(page, context);
    await expect(page.locator("[data-part=bottom-nav]")).toBeVisible();
    await page.getByRole("textbox", { name: "Câu hỏi của bạn" }).focus();
    await expect(page.locator("[data-part=bottom-nav]")).toBeHidden();
    const m = await page.evaluate(() => ({ bottom: Math.round(document.querySelector("[data-part=chat-composer]")!.getBoundingClientRect().bottom), ih: innerHeight }));
    expect(Math.abs(m.bottom - m.ih)).toBeLessThanOrEqual(1);
  });
});

// M1 (chỉnh): nút chọn lớp giãn hết phần trống, tên lớp không bị cắt sớm
test("M1 @375: nút chọn lớp giãn tới cụm tìm / chuông / hồ sơ", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await checkinApi(page, context);
  await page.goto("/checkin");
  const pick = page.getByRole("button", { name: /^Chọn lớp/ });
  await expect(pick).toBeVisible();
  const pb = (await pick.boundingBox())!;
  const bell = (await page.getByRole("button", { name: "Thông báo" }).boundingBox())!;
  const search = (await page.getByRole("button", { name: /Tìm nhanh/ }).boundingBox())!;
  expect(Math.min(bell.x, search.x) - (pb.x + pb.width)).toBeLessThanOrEqual(24);
});
