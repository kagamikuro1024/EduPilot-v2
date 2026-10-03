import { expect, test, type Page } from "@playwright/test";
import { settleGoto } from "./support/hydrate";
import { loadAudit, runAudit } from "./support/audit";
import { BASE_URL } from "./support/env";
import { asJwt, type JwtRole } from "./support/session";

// US-P2-07 — bộ chọn lớp từ `GET /me/courses`, màn "Bạn chưa vào lớp nào", ánh xạ lớp mô phỏng. Gateway giả bằng page.route (phần @real chạy ở bản compose có seed).

test.beforeEach(({ page }) => settleGoto(page));

const C1 = "00000000-0000-7000-8000-00000000c001";
const C2 = "00000000-0000-7000-8000-00000000c002";
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };

const item = (id: string, code: string, role: "TEACHER" | "TA" | "STUDENT", status: "ACTIVE" | "PENDING" = "ACTIVE") => ({
  course: { id, class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" },
  role_in_course: role,
  enrollment_status: status,
});

/** Phiên thật của `role` + `GET /me/courses` trả đúng `items`; trả danh sách URL `/api/v1/courses/…` đã bị gọi (AC12: phải rỗng). */
async function login(page: Page, role: JwtRole, items: unknown[]) {
  await asJwt(page, role);
  const courseCalls: string[] = [];
  await page.route("**/api/v1/me/courses**", (route) => route.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ items, next_cursor: null }) }));
  await page.route("**/api/v1/courses/**", (route) => {
    courseCalls.push(route.request().url());
    return route.fulfill({ status: 403, contentType: "application/json", headers: cors, body: "{}" });
  });
  return courseCalls;
}

const picker = (page: Page) => page.getByRole("button", { name: /^Chọn lớp, đang xem/ });

async function options(page: Page) {
  await picker(page).click();
  const panel = page.getByRole("dialog", { name: "Chọn lớp" });
  await expect(panel).toBeVisible();
  return panel;
}

test("course picker: giảng viên 2 lớp thấy cả hai, 'Tất cả lớp của tôi' và 'Quản lý lớp này'", async ({ page }) => {
  await login(page, "TEACHER", [item(C1, "761987", "TEACHER"), item(C2, "761988", "TEACHER")]);
  await page.goto("/");
  await expect(picker(page)).toContainText("761987");
  const panel = await options(page);
  await expect(panel.getByRole("menuitem", { name: /761987 · An ninh mạng/ })).toBeVisible();
  await expect(panel.getByRole("menuitem", { name: /761988 · An ninh mạng/ })).toBeVisible();
  await expect(panel.getByRole("menuitem", { name: "Tất cả lớp của tôi" })).toBeVisible();
  await panel.getByRole("menuitem", { name: "Quản lý lớp này" }).click();
  await expect(page).toHaveURL(/\/class\/members$/);
});

test("course picker: sinh viên thấy lớp ACTIVE của mình, lớp PENDING không chọn được, không có 'Tất cả lớp'", async ({ page }) => {
  await login(page, "STUDENT", [item(C1, "761987", "STUDENT"), item(C2, "761988", "STUDENT")]);
  await page.goto("/");
  const both = await options(page);
  await expect(both.getByRole("menuitem", { name: /761987 · An ninh mạng/ })).toBeVisible();
  await expect(both.getByRole("menuitem", { name: /761988 · An ninh mạng/ })).toBeVisible();
  await expect(both.getByRole("menuitem", { name: "Tất cả lớp của tôi" })).toHaveCount(0);
  await expect(both.getByRole("menuitem", { name: "Quản lý lớp này" })).toHaveCount(0);
  await expect(both.getByRole("menuitem", { name: "Tham gia lớp bằng mã" })).toBeVisible();
});

test("course picker: sinh viên 1 lớp ACTIVE + 1 PENDING chỉ thấy lớp ACTIVE", async ({ page }) => {
  await login(page, "STUDENT", [item(C1, "761987", "STUDENT"), item(C2, "761988", "STUDENT", "PENDING")]);
  await page.goto("/");
  const panel = await options(page);
  await expect(panel.getByRole("menuitem", { name: /761987/ })).toBeVisible();
  await expect(panel.getByRole("menuitem", { name: /761988/ })).toHaveCount(0);
});

test("course picker: ?course= của lớp không thuộc người này bị bỏ, về lớp đầu", async ({ page }) => {
  await login(page, "STUDENT", [item(C1, "761987", "STUDENT")]);
  await page.goto(`/?course=${C2}`);
  await expect(picker(page)).toContainText("761987");
  await expect(page).toHaveURL(`${BASE_URL}/`);
});

test("course picker: ?course= của lớp thuộc người này đè lựa chọn và được nhớ ở localStorage", async ({ page }) => {
  await login(page, "TEACHER", [item(C1, "761987", "TEACHER"), item(C2, "761988", "TEACHER")]);
  await page.goto(`/?course=${C2}`);
  await expect(picker(page)).toContainText("761988");
  await expect(page).toHaveURL(`${BASE_URL}/`);
  expect(await page.evaluate(() => localStorage.getItem("ep:ui:course"))).toBe(C2);
  await page.reload();
  await expect(picker(page)).toContainText("761988");
});

test("course picker: không lớp nào → 'Chưa có lớp'; chữ dài bị cắt có title; không tràn ở 1440 / 390", async ({ page }) => {
  const { AUDIT_SRC } = await loadAudit();
  await login(page, "STUDENT", [item(C1, "761987", "STUDENT")]);
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/");
    await expect(picker(page)).toContainText("761987");
    const a = await runAudit(page, AUDIT_SRC);
    expect(a.ell, `ell @${width}`).toEqual([]);
    expect(a.ox, `ox @${width}`).toBe(0);
  }
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await login(page, "STUDENT", []);
  await page.goto("/");
  await expect(page.getByRole("button", { name: /^Chọn lớp, đang xem Chưa có lớp/ })).toBeVisible();
});

test("no course screen: SV chưa có lớp ACTIVE thấy cùng một màn ở 7 route và không gọi API lớp", async ({ page }) => {
  const courseCalls = await login(page, "STUDENT", []);
  const routes = ["/chat", "/threads", "/practice", "/library", "/calendar", "/me", "/assignments/a1"];
  for (const route of routes) {
    await page.goto(route);
    await expect(page.getByRole("heading", { name: "Bạn chưa vào lớp nào" }), route).toBeVisible();
    await expect(page.getByText("Nhập mã tham gia do giảng viên cung cấp để dùng tính năng này."), route).toBeVisible();
    await expect(page.locator("main").getByRole("link", { name: "Tham gia lớp bằng mã" }), route).toHaveAttribute("href", "/join");
    await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" }), route).toHaveCount(0);
  }
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Bạn chưa vào lớp nào" })).toHaveCount(0);
  await page.goto("/join");
  await expect(page.getByRole("heading", { name: "Bạn chưa vào lớp nào" })).toHaveCount(0);
  expect(courseCalls).toEqual([]);
});

test("mock course mapping: giảng viên chọn 761988 → màn mô phỏng dùng lớp int1006-2, không gọi API lớp", async ({ page }) => {
  const courseCalls = await login(page, "TEACHER", [item(C1, "761987", "TEACHER"), item(C2, "761988", "TEACHER")]);
  await page.goto("/inbox");
  await expect(page.getByText(/An ninh mạng – 761987/).first()).toBeVisible();
  const panel = await options(page);
  await panel.getByRole("menuitem", { name: /761988 · An ninh mạng/ }).click();
  await expect(page.getByText(/An ninh mạng – 761988/).first()).toBeVisible();
  await expect(page.getByText(/An ninh mạng – 761987/)).toHaveCount(0);
  expect(courseCalls).toEqual([]);
});

test("mock course mapping: lớp có class_code lạ dùng lớp mô phỏng đầu", async ({ page }) => {
  await login(page, "TEACHER", [item(C1, "999999", "TEACHER")]);
  await page.goto("/inbox");
  await expect(picker(page)).toContainText("999999");
  await expect(page.getByText(/An ninh mạng – 761987/).first()).toBeVisible();
});

test("@real course picker với seed: GV 2 lớp, SV A 2 lớp, SV B 1 lớp, SV D chưa có lớp", async () => {
  test.skip(true, "@real — cần compose + seed (US-P2-12); chạy tay theo handoff dev-US-P2-07");
});
