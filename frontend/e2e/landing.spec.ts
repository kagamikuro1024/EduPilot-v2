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
  await expect(page.getByRole("link", { name: "Tạo tài khoản" })).toHaveAttribute("href", "/register");
  await expect(page.getByText("Đồ án tốt nghiệp · Lê Văn Quang Trung")).toBeVisible();
  await expect(page.getByRole("link", { name: "Mã nguồn trên GitHub" })).toHaveAttribute("href", /github\.com\/kagamikuro1024\/EduPilot-v2$/);
  await expect(page.getByText("Dữ liệu trên bản demo là dữ liệu mô phỏng.")).toBeVisible();
  const text = await page.locator("body").innerText();
  expect(text).not.toMatch(/@edupilot\.local|Edupilot#|mật khẩu mặc định/i);
  await expect(page.locator("[data-part=landing] h2")).toHaveText(["Khác biệt, có số đo", "Cho giảng viên", "Cho sinh viên"]);
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
