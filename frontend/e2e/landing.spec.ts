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

test("landing anonymous: tiêu đề, nút Đăng nhập / Tạo tài khoản, chân trang, không gọi API nào ngoài refresh, không tài khoản demo", async ({ page }) => {
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
  await expect(page.locator("[data-part=landing] h2")).toHaveText(["AI soạn nháp, giảng viên quyết, mã tính điểm", "Khác biệt, có số đo", "Cho giảng viên và sinh viên", "Bắt đầu với lớp của bạn"]);
  await expect(page.locator("[data-part=sidebar]:visible, [data-part=topbar]:visible")).toHaveCount(0); // không khung ứng dụng
  expect(api.filter((p) => !p.endsWith("/auth/refresh"))).toEqual([]);
  await page.getByRole("link", { name: "Đăng nhập" }).first().click();
  await expect(page).toHaveURL(`${BASE_URL}/login`);
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

test("landing video: không tải video khi chưa bấm (preload=none), có poster, không tự phát, có lời thay cho video không tiếng", async ({ page }) => {
  const media: string[] = [];
  page.on("request", (q) => /\/landing\/gioi-thieu-v1\.(mp4|webm)/.test(q.url()) && media.push(new URL(q.url()).pathname));
  await anonymous(page);
  await page.goto("/");
  const video = page.locator("[data-part=landing] video");
  await expect(video).toHaveAttribute("preload", "none");
  await expect(video).toHaveAttribute("poster", "/landing/gioi-thieu-v1.webp");
  expect(await video.evaluate((v: HTMLVideoElement) => v.autoplay || !v.paused)).toBe(false);
  await page.waitForLoadState("networkidle");
  expect(media).toEqual([]);
  await expect(page.locator("#video-loi")).toContainText("AI không tự công bố");
  await expect(video).toHaveAttribute("aria-describedby", "video-loi");
  const box = page.locator("[data-part=landing] video >> xpath=..");
  await expect(box).not.toHaveAttribute("data-playing", ""); // nút phát vẽ sẵn đang hiện
  await video.evaluate((v: HTMLVideoElement) => v.play()); // bấm vào video là hành vi của trình duyệt; ở đây chỉ kiểm nút phát vẽ sẵn ẩn / hiện theo sự kiện
  await expect(box).toHaveAttribute("data-playing", "");
  await video.evaluate((v: HTMLVideoElement) => v.pause());
  await expect(box).not.toHaveAttribute("data-playing", "");
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

test("landing vai: mặc định Giảng viên; chọn Sinh viên (chuột hoặc mũi tên) thì đổi nội dung, không cần JS", async ({ page }) => {
  await anonymous(page);
  await page.goto("/");
  const gv = page.getByText("Việc cần xử lý xếp theo mức khẩn");
  const sv = page.getByText("Hỏi về điểm, chuyên cần, lịch");
  await expect(gv).toBeVisible();
  await expect(sv).toBeHidden();
  await page.getByText("Sinh viên", { exact: true }).click();
  await expect(sv).toBeVisible();
  await expect(gv).toBeHidden();
  await page.getByRole("radio", { name: "Giảng viên" }).focus();
  await page.keyboard.press("ArrowRight");
  await expect(sv).toBeVisible();
});
