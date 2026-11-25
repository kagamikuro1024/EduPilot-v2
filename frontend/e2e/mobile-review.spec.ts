import { expect, test } from "@playwright/test";
import { checkinApi } from "./support/checkin-fixtures";
import { practiceApi } from "./support/practice-fixtures";
import { attApi } from "./support/att-fixtures";
import { asDemo } from "./support/session";

// Soát giao diện điện thoại 25/11/2026 (docs/design/review/2026-11-25-mobile.md): M1, M2, M3, M6, M12. Desktop project, tự đặt viewport.
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt viewport");
});
const redish = (bg: string) => { const [r, g] = bg.match(/\d+/g)!.map(Number); return r - g >= 40; };

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
    const st = await btn.evaluate((e) => { const c = getComputedStyle(e); return { bg: c.backgroundColor, op: c.opacity }; });
    expect(redish(st.bg), `nền ${st.bg}`).toBe(false);
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
  const y0 = (await pre.locator("h1").boundingBox())!.y;
  release();
  await expect(page.locator("[data-part=bottom-nav]")).toBeVisible();
  const y1 = (await page.locator("main h1").boundingBox())!.y;
  expect(Math.abs(y1 - y0)).toBeLessThanOrEqual(2);
});
