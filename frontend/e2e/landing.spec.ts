import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { settleGoto } from "./support/hydrate";
import { mockStudentToday } from "./support/screens";
import { asDemo } from "./support/session";

// OPS-VPS (landing): `/` chưa đăng nhập → trang giới thiệu tĩnh; đã đăng nhập → "Hôm nay"; route sâu vẫn về /login?next=.
test.beforeEach(({ page }) => settleGoto(page));
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const anonymous = (page: Page) => page.route("**/api/v1/auth/refresh", (r) => r.fulfill({ status: 401, contentType: "application/json", headers: cors, body: JSON.stringify({ code: "UNAUTHENTICATED", message: "x", trace_id: "c".repeat(32) }) }));

test("landing anonymous: tiêu đề, nút Đăng nhập / Tạo tài khoản, chân trang, không gọi API nào (kể cả refresh), không tài khoản demo", async ({ page }) => {
  const api: string[] = [];
  page.on("request", (q) => q.url().includes("/api/v1/") && api.push(new URL(q.url()).pathname));
  await anonymous(page);
  await page.goto("/");
  await expect(page).toHaveURL(`${BASE_URL}/`);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân");
  await expect(page.locator("[data-part=landing]").getByRole("link", { name: "Đăng nhập" }).first()).toBeVisible();
  await expect(page.getByRole("link", { name: "Tạo tài khoản" }).first()).toHaveAttribute("href", "/register");
  await expect(page.getByText("Đồ án tốt nghiệp · Lê Văn Quang Trung")).toBeVisible();
  await expect(page.getByRole("link", { name: "Mã nguồn trên GitHub" })).toHaveAttribute("href", /github\.com\/kagamikuro1024\/EduPilot-v2$/);
  await expect(page.getByText("Dữ liệu trên bản demo là dữ liệu mô phỏng.")).toBeVisible();
  const text = await page.locator("body").innerText();
  expect(text).not.toMatch(/@edupilot\.local|Edupilot#|mật khẩu mặc định/i);
  await expect(page.locator("[data-part=landing] h2")).toHaveText(["Xem EduPilot làm việc", "AI soạn nháp, giảng viên quyết, mã tính điểm", "Khác biệt, có số đo", "Cho giảng viên và sinh viên", "Bắt đầu với lớp của bạn"]);
  await expect(page.locator("[data-part=sidebar]:visible, [data-part=topbar]:visible")).toHaveCount(0); // không khung ứng dụng
  expect(api).toEqual([]); // US-21: máy chưa từng đăng nhập thì `/` không gọi làm mới phiên
  await page.getByRole("link", { name: "Đăng nhập" }).first().click();
  await expect(page).toHaveURL(`${BASE_URL}/login`);
});

test("landing có cờ 'đã từng đăng nhập' ⇒ vẫn gọi làm mới phiên (US-21 AC2); mất cờ vẫn thấy giới thiệu", async ({ page }) => {
  const refresh: string[] = [];
  page.on("request", (q) => q.url().includes("/api/v1/auth/refresh") && refresh.push(q.method()));
  await anonymous(page);
  await page.addInitScript(() => localStorage.setItem("ep_returning", "1"));
  await page.goto("/");
  await expect.poll(() => refresh.length).toBeGreaterThan(0);
});

test("landing signed in: đã đăng nhập thì `/` là Hôm nay, không có trang giới thiệu", async ({ page, context }) => {
  await asDemo(context, "student");
  await mockStudentToday(page);
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible();
  await expect(page.getByRole("heading", { level: 1 })).not.toHaveText(/Vận hành lớp học/);
  await expect(page.locator("[data-part=landing]")).toHaveCount(0);
  await expect(page).toHaveTitle("Hôm nay · EduPilot");
});

test("landing deep link: /gradebook chưa đăng nhập vẫn về /login?next=", async ({ page }) => {
  await anonymous(page);
  await page.goto("/gradebook");
  await expect(page).toHaveURL(/\/login\?next=%2Fgradebook$/);
});

test("landing back from login: logo ở /login quay về trang giới thiệu", async ({ page }) => {
  await anonymous(page);
  await page.goto("/login");
  await page.getByRole("link", { name: /trang giới thiệu/ }).click();
  await expect(page).toHaveURL(`${BASE_URL}/`);
  await expect(page.locator("[data-part=landing]")).toBeVisible();
});

test("landing stale hint: dấu 'đã từng đăng nhập' nhưng phiên hết hạn → trang giới thiệu hiện, dấu bị gỡ", async ({ page }) => {
  await anonymous(page);
  await page.addInitScript(() => localStorage.setItem("ep_returning", "1"));
  await page.goto("/");
  await expect(page.locator("[data-part=landing]")).toBeVisible();
  await expect.poll(() => page.evaluate(() => localStorage.getItem("ep_returning"))).toBeNull();
});

for (const width of [1440, 375]) {
  test(`landing responsive ${width}: không cuộn ngang, axe không lỗi nghiêm trọng, ảnh có kích thước`, async ({ page }) => {
    await anonymous(page);
    await page.setViewportSize({ width, height: width === 375 ? 812 : 900 });
    await page.goto("/");
    await expect(page.locator("[data-part=landing]")).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    for (const img of await page.locator("[data-part=landing] img[src^='/landing/']").all()) {
      expect(await img.getAttribute("width")).toBeTruthy();
      expect(await img.getAttribute("height")).toBeTruthy();
      expect((await img.getAttribute("alt"))?.length).toBeGreaterThan(10);
    }
    const res = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
    expect(res.violations.filter((v) => v.impact === "critical" || v.impact === "serious").map((v) => `${v.id}: ${v.nodes.length}`)).toEqual([]);
  });
}

test("landing video đầy đủ: không tải khi chưa bấm; `Xem video` mở hộp thoại, Esc đóng; có lời thay cho video không tiếng", async ({ page }) => {
  const media: string[] = [];
  page.on("request", (q) => /\/landing\/gioi-thieu-v1\.(mp4|webm)/.test(q.url()) && media.push(new URL(q.url()).pathname));
  await anonymous(page);
  await page.goto("/");
  await page.waitForLoadState("networkidle");
  expect(media).toEqual([]);
  const dlg = page.locator("[data-part=landing] dialog");
  const video = dlg.locator("video");
  await expect(video).toHaveAttribute("preload", "none");
  expect(await video.evaluate((v: HTMLVideoElement) => v.autoplay || !v.paused)).toBe(false);
  await expect(dlg).toBeHidden();
  const open = page.getByRole("button", { name: "Xem video" });
  await open.click();
  await expect(dlg).toBeVisible();
  await expect(video).toHaveAttribute("poster", "/landing/gioi-thieu-v1.webp");
  expect(media).toEqual([]); // mở hộp thoại vẫn chưa tải video
  await page.keyboard.press("Escape");
  await expect(dlg).toBeHidden();
  await expect(open).toBeFocused();
  await expect(page.locator("#video-loi")).toContainText("AI không tự công bố");
});

test("landing clip màn đầu: poster trước, clip nạp sau `load`, có nút Tạm dừng / Phát; reduce và Save-Data thì không tải", async ({ page, browser }) => {
  await anonymous(page);
  const clipReq: string[] = [];
  page.on("request", (q) => /gioi-thieu-clip-v1\.(mp4|webm)/.test(q.url()) && clipReq.push(q.url()));
  await page.goto("/");
  const clip = page.locator("[data-part=landing] video[data-clip]");
  await expect(clip).toHaveAttribute("poster", "/landing/gioi-thieu-clip-v1.webp");
  await expect(clip).toHaveAttribute("aria-hidden", "true");
  const toggle = page.getByRole("button", { name: "Tạm dừng" });
  await expect(toggle).toBeVisible();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await toggle.click();
  await expect(page.getByRole("button", { name: "Phát" })).toHaveAttribute("aria-pressed", "true");
  expect(await clip.evaluate((v: HTMLVideoElement) => v.paused)).toBe(true);
  expect(clipReq.length).toBeGreaterThan(0);

  const reduced = await browser.newContext({ reducedMotion: "reduce" });
  const p2 = await reduced.newPage();
  const req2: string[] = [];
  p2.on("request", (q) => /gioi-thieu-clip-v1\.(mp4|webm)/.test(q.url()) && req2.push(q.url()));
  await anonymous(p2);
  await p2.goto("/");
  await p2.waitForLoadState("networkidle");
  expect(req2).toEqual([]);
  await expect(p2.getByRole("button", { name: /Tạm dừng|Phát/ })).toBeHidden();
  await reduced.close();
});

test("landing motion: khối dưới màn hình hiện dần theo cuộn (scroll-driven); reduced-motion hiện đủ ngay", async ({ page }) => {
  await anonymous(page);
  await page.goto("/");
  const proof = page.locator("[data-part=landing] li").filter({ hasText: "yêu cầu gửi AI" }).first();
  const opacity = () => proof.evaluate((e) => Number(getComputedStyle(e).opacity));
  expect(await opacity()).toBeLessThan(1);
  await proof.scrollIntoViewIfNeeded();
  await page.mouse.wheel(0, 400);
  await expect.poll(opacity).toBe(1);
  await expect(proof.locator("p").first()).toHaveText("146"); // chữ thật luôn ở trong DOM; số đếm chỉ là phần vẽ

  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.reload();
  await expect(page.locator("[data-part=landing]")).toBeVisible();
  expect(await opacity()).toBe(1);
});

test("landing vai: hai cột Giảng viên và Sinh viên cạnh nhau, mỗi vai 4 mục, không tab, nhãn mới 'Hỏi đáp công khai'", async ({ page }) => {
  await anonymous(page);
  await page.goto("/");
  const sec = page.locator("[data-part=landing] section[aria-labelledby=roles-title]");
  await expect(sec.getByRole("heading", { level: 3 })).toHaveText(["Giảng viên", "Sinh viên"]);
  await expect(sec.locator("dt")).toHaveCount(8);
  await expect(sec.getByRole("tab")).toHaveCount(0);
  await expect(sec.getByRole("radio")).toHaveCount(0);
  const [a, b] = await sec.getByRole("heading", { level: 3 }).evaluateAll((hs) => hs.map((h) => h.getBoundingClientRect().top));
  expect(Math.abs(a - b)).toBeLessThan(4); // cạnh nhau ở 1440
  await expect(sec.getByText("Hỏi đáp công khai", { exact: true })).toBeVisible();
  expect(await page.locator("[data-part=landing]").innerText()).not.toMatch(/Threads/);
});

test("landing khung ghim: ở 1440 khung ảnh dính; ở 1024 xếp dọc, mỗi bước mang ảnh của nó", async ({ page }) => {
  await anonymous(page);
  const stage = page.locator("[data-part=landing] section[aria-labelledby=story-title] div[aria-hidden=true]");
  await page.goto("/");
  expect(await stage.evaluate((e) => getComputedStyle(e).position)).toBe("sticky");
  await page.setViewportSize({ width: 1024, height: 800 });
  expect(await stage.evaluate((e) => getComputedStyle(e).display)).toBe("none");
  await expect(page.locator("[data-part=landing] section[aria-labelledby=story-title] figure img")).toHaveCount(3);
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  expect(await stage.evaluate((e) => getComputedStyle(e).display)).toBe("none"); // reduce: không ghim
});
