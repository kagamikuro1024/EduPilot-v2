import { expect, test, type Page } from "@playwright/test";
import { settleGoto } from "./support/hydrate";
import { loadAudit, runAudit } from "./support/audit";
import { BASE_URL } from "./support/env";
import { asJwt, type JwtRole } from "./support/session";

type Json = Record<string, unknown>;

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

// ===== US-P2-08: /admin/courses và chuông thật =====

const fulfill = (route: import("@playwright/test").Route, status: number, body: unknown) =>
  status === 204 ? route.fulfill({ status, headers: cors }) : route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const apiErr = (code: string, details?: unknown, message = "m") => ({ code, message, trace_id: "b".repeat(32), details });

const course = (id: string, code: string, over: Json = {}): Json => ({
  id, class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE",
  teacher: { id: "t1", full_name: "Trần Văn Giảng" }, assistants_count: 1, students_active: 30, students_pending: 2, capacity: 30, version: 1, ...over,
});
const person = (id: string, name: string, status = "ACTIVE") => ({ id, email: `${id}@x.test`, full_name: name, role: "TEACHER", status, last_login_at: null, version: 1 });

type CoursesLog = { post: string[]; keys: Array<string | undefined>; put: string[]; assign: string[]; archive: string[] };
type CoursesHandlers = {
  post?: (body: Json, n: number) => { status: number; body: unknown };
  put?: (id: string, body: Json) => { status: number; body: unknown };
  assign?: (id: string, body: Json) => { status: number; body: unknown };
  archive?: (id: string) => { status: number; body: unknown };
};

async function openCourses(page: Page, rows: Json[], h: CoursesHandlers = {}) {
  await asJwt(page, "ADMIN");
  await page.route("**/api/v1/me/courses**", (r) => fulfill(r, 200, { items: [], next_cursor: null }));
  await page.route("**/api/v1/notifications**", (r) => fulfill(r, 200, { items: [], next_cursor: null, unread_count: 0 }));
  await page.route("**/api/v1/admin/users**", (r) => {
    const role = new URL(r.request().url()).searchParams.get("role");
    return fulfill(r, 200, { items: role === "TA" ? [{ ...person("ta1", "Lê Trợ Giảng"), role: "TA" }] : [person("t1", "Trần Văn Giảng"), person("t2", "Phạm Giảng Viên")], next_cursor: null });
  });
  const log: CoursesLog = { post: [], keys: [], put: [], assign: [], archive: [] };
  let current = rows;
  let posts = 0;
  await page.route("**/api/v1/admin/courses**", async (r) => {
    const req = r.request();
    const parts = new URL(req.url()).pathname.replace("/api/v1/admin/courses", "").split("/").filter(Boolean);
    const body = JSON.parse(req.postData() ?? "{}") as Json;
    if (req.method() === "GET") return fulfill(r, 200, { items: current, next_cursor: null });
    if (req.method() === "POST" && parts.length === 0) {
      log.post.push(req.postData() ?? "");
      log.keys.push(req.headers()["idempotency-key"]);
      const out = h.post?.(body, ++posts) ?? { status: 201, body: { ...course("new", String(body.class_code)), teacher: null } };
      if (out.status === 201) current = [course("new", String(body.class_code), { teacher: body.teacher_id ? { id: String(body.teacher_id), full_name: "Trần Văn Giảng" } : null, students_active: 0, students_pending: 0, capacity: null }), ...current];
      return fulfill(r, out.status, out.body);
    }
    if (req.method() === "PUT") {
      log.put.push(`${parts[0]} ${req.postData()}`);
      const out = h.put?.(parts[0], body) ?? { status: 200, body: course(parts[0], "761987") };
      return fulfill(r, out.status, out.body);
    }
    if (parts[1] === "assign") {
      log.assign.push(`${parts[0]} ${req.postData()}`);
      const out = h.assign?.(parts[0], body) ?? { status: 200, body: { teacher: { id: "t2", full_name: "Phạm Giảng Viên" }, assistants: [], changed: { teacher: true, added_ta: [], removed_ta: [] } } };
      return fulfill(r, out.status, out.body);
    }
    if (parts[1] === "archive") {
      log.archive.push(parts[0]);
      const out = h.archive?.(parts[0]) ?? { status: 200, body: course(parts[0], "761987", { status: "ARCHIVED" }) };
      if (out.status === 200) current = current.map((c) => (c.id === parts[0] ? { ...c, status: "ARCHIVED" } : c));
      return fulfill(r, out.status, out.body);
    }
    return r.fallback();
  });
  await page.goto("/admin/courses");
  await expect(page.getByRole("heading", { name: "Lớp học", level: 1 })).toBeVisible();
  return log;
}

async function openCourseForm(page: Page) {
  await expect(async () => {
    if (!(await page.getByLabel("Mã lớp", { exact: true }).isVisible())) await page.getByRole("button", { name: "Mở lớp" }).first().click();
    await expect(page.getByLabel("Mã lớp", { exact: true })).toBeVisible({ timeout: 1000 });
  }).toPass({ timeout: 10_000 });
}

test("admin courses page: bảng đúng cột, một nút chính, khung mở tại chỗ, gửi xong có dòng tĩnh, bấm đúp một lớp, lưu trữ có xác nhận nêu số", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung; 375 px kiểm trong ca riêng");
  const log = await openCourses(page, [course("c1", "761987"), course("c2", "761988", { teacher: null, students_active: 12, capacity: null })]);
  const open = page.getByRole("button", { name: "Mở lớp" });
  await expect(open).toHaveCount(1);
  await expect(open).toHaveAttribute("data-variant", "primary");
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  for (const h of ["Mã lớp", "Học phần", "Giảng viên", "Sĩ số", "Trạng thái"]) await expect(page.getByRole("columnheader", { name: h })).toBeVisible();
  await expect(page.locator("table tbody tr")).toHaveCount(2);
  await expect(page.locator("table").getByText("30 / 30")).toBeVisible();
  await expect(page.locator("table tbody tr").filter({ hasText: "761988" }).getByText("Chưa gán")).toBeVisible();
  await expect(page.getByText("2 lớp đang chạy")).toBeVisible();

  await openCourseForm(page);
  await expect(page.getByRole("dialog")).toHaveCount(0); // khung mở tại chỗ, không modal
  await expect(page.getByRole("button", { name: "Mở lớp" })).toHaveCount(1); // nút chính ở đầu trang ẩn khi khung mở: vẫn đúng một
  await page.getByLabel("Mã lớp", { exact: true }).fill("999001");
  await page.getByLabel("Giảng viên").selectOption({ label: "Trần Văn Giảng" });
  await page.getByLabel("Lê Trợ Giảng").check();
  const submit = page.getByRole("button", { name: "Mở lớp" });
  await submit.dblclick();
  await expect(page.getByRole("status").filter({ hasText: "Đã mở lớp. Đã gửi thông báo phân công cho Trần Văn Giảng." })).toBeVisible();
  expect(log.post).toHaveLength(1);
  expect(JSON.parse(log.post[0])).toEqual({ subject_code: "INT1006", class_code: "999001", name: "An ninh mạng", semester: "2026-2027-HK1", teacher_id: "t1", ta_ids: ["ta1"] });
  expect(log.keys[0]).toMatch(/^ep-/);
  await expect(page.locator("table tbody tr")).toHaveCount(3);

  // lưu trữ: hộp xác nhận chỉ xuất hiện ở đây, nêu số
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.locator("table tbody tr").filter({ hasText: "761987" }).getByRole("button", { name: /Hành động cho lớp 761987/ }).click();
  await page.getByRole("menuitem", { name: "Lưu trữ lớp" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(1);
  await expect(page.getByText("Lưu trữ lớp 761987: 30 sinh viên chỉ còn quyền đọc; mã tham gia ngừng hoạt động.")).toBeVisible();
  await page.getByRole("dialog").getByRole("button", { name: "Lưu trữ lớp" }).click();
  await expect.poll(() => log.archive).toEqual(["c1"]);
  await expect(page.getByRole("status").filter({ hasText: "Đã lưu trữ lớp 761987." })).toBeVisible();
  await expect(page.locator("table tbody tr").filter({ hasText: "761987" }).getByText("Đã lưu trữ")).toBeVisible();
  await expect(page.getByRole("button", { name: /Hành động cho lớp 761987/ })).toHaveCount(0);
  expect((await page.locator("main").innerText()).toLowerCase()).not.toContain("join_code");
});

test("admin courses page: sửa và gán lại dùng PUT / assign; rỗng có lời mời mở lớp; không có đường thêm sinh viên; 375 px sạch", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung; 375 px kiểm trong ca riêng");
  const log = await openCourses(page, [course("c1", "761987")]);
  const row = page.locator("table tbody tr").filter({ hasText: "761987" });
  await row.getByRole("button", { name: /Hành động cho lớp/ }).click();
  await page.getByRole("menuitem", { name: "Sửa" }).click();
  await page.getByLabel("Tên lớp").fill("Tên đã sửa");
  await page.getByRole("button", { name: "Lưu thay đổi" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã lưu thay đổi của lớp." })).toBeVisible();
  expect(JSON.parse(log.put[0].split(" ").slice(1).join(" "))).toMatchObject({ name: "Tên đã sửa", class_code: "761987", capacity: 30, version: 1 });

  await row.getByRole("button", { name: /Hành động cho lớp/ }).click();
  await page.getByRole("menuitem", { name: "Gán lại" }).click();
  await page.getByLabel("Giảng viên").selectOption({ label: "Phạm Giảng Viên" });
  await page.getByRole("button", { name: "Gán lại" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gán lại lớp. Đã gửi thông báo phân công cho Phạm Giảng Viên." })).toBeVisible();
  expect(JSON.parse(log.assign[0].split(" ").slice(1).join(" "))).toEqual({ teacher_id: "t2" });
  expect(log.assign[0]).not.toContain("ta_ids"); // không đụng tới trợ giảng khi không chọn

  for (const w of [/Thêm sinh viên/, /Ghi danh/]) await expect(page.getByRole("button", { name: w })).toHaveCount(0);
  await page.setViewportSize({ width: 375, height: 800 });
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("admin courses page: chưa có lớp → lời mời mở lớp đầu tiên", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung");
  await openCourses(page, []);
  await expect(page.getByText("Chưa có lớp nào. Mở lớp đầu tiên.")).toBeVisible();
});

test("admin courses errors: mã lớp trùng tại ô, 422 theo ô giữ chữ, gửi lại cùng khoá, sai version, sai vai, lưu trữ lại", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung");
  const log = await openCourses(page, [course("c1", "761987"), course("c2", "761988")], {
    post: (b, n) => {
      if (b.class_code === "TRUNG") return { status: 409, body: apiErr("CONFLICT", { field: "class_code" }) };
      if (b.class_code === "SAIVAI") return { status: 422, body: apiErr("VALIDATION_FAILED", [{ field: "teacher_id", code: "ROLE_MISMATCH", message: "Người này không phải giảng viên." }]) };
      if (b.class_code === "TAMTHOI" && n < 4) return { status: 503, body: apiErr("SERVICE_UNAVAILABLE") };
      return { status: 201, body: course("new", String(b.class_code)) };
    },
    put: (id) => ({ status: 409, body: apiErr("VERSION_CONFLICT", { current_version: 5, current: course(id, "761987", { version: 5 }) }) }),
    archive: () => ({ status: 409, body: apiErr("COURSE_ARCHIVED") }),
  });
  await openCourseForm(page);
  const code = page.getByLabel("Mã lớp", { exact: true });
  const name = page.getByLabel("Tên lớp");
  await code.fill("TRUNG");
  await name.fill("Tên giữ nguyên");
  await page.getByRole("button", { name: "Mở lớp" }).click();
  await expect(page.getByText("Mã lớp này đã có.")).toBeVisible();
  await expect(code).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveValue("Tên giữ nguyên");

  await code.fill("SAIVAI");
  await page.getByRole("button", { name: "Mở lớp" }).click();
  await expect(page.getByText("Người này không phải giảng viên.")).toBeVisible();
  await expect(name).toHaveValue("Tên giữ nguyên");

  await code.fill("TAMTHOI");
  await page.getByRole("button", { name: "Mở lớp" }).click();
  const retry = page.getByRole("alert").getByRole("button", { name: "Gửi lại" });
  await expect(retry).toBeVisible();
  const before = log.keys.length;
  await retry.click();
  await expect.poll(() => log.keys.length).toBe(before + 1);
  expect(log.keys.at(-1)).toBe(log.keys.at(-2)); // cùng Idempotency-Key
  await expect(page.getByRole("status").filter({ hasText: "Đã mở lớp." })).toBeVisible(); // gửi lại thành công: một lớp

  // sai version khi sửa
  const row = page.locator("table tbody tr").filter({ hasText: "761987" });
  await row.getByRole("button", { name: /Hành động cho lớp/ }).click();
  await page.getByRole("menuitem", { name: "Sửa" }).click();
  await page.getByLabel("Tên lớp").fill("Bản của tôi");
  await page.getByRole("button", { name: "Lưu thay đổi" }).click();
  await expect(page.getByText("Lớp này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?")).toBeVisible();
  await expect(page.getByLabel("Tên lớp")).toHaveValue("Bản của tôi");
  await page.getByRole("button", { name: "Giữ thay đổi của tôi" }).click();
  await page.getByRole("button", { name: "Lưu thay đổi" }).click();
  await expect.poll(() => log.put.length).toBe(2);
  expect(log.put[1]).toContain('"version":5');
  await page.getByRole("button", { name: "Huỷ" }).click();

  // lưu trữ lớp đã lưu trữ
  await page.locator("table tbody tr").filter({ hasText: "761988" }).getByRole("button", { name: /Hành động cho lớp/ }).click();
  await page.getByRole("menuitem", { name: "Lưu trữ lớp" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Lưu trữ lớp" }).click();
  await expect(page.getByRole("dialog").getByRole("alert")).toContainText("Lớp này đã được lưu trữ.");
});

test("bell real: chấm theo unread_count, mở khung không xoá chấm, bấm mục đánh dấu đã đọc và đi tới link, làm mới 30 s", async ({ page }) => {
  await page.clock.install();
  await asJwt(page, "TEACHER");
  await page.route("**/api/v1/me/courses**", (r) => fulfill(r, 200, { items: [item(C1, "761987", "TEACHER")], next_cursor: null }));
  const now = new Date();
  const mk = (id: string, title: string, read: boolean, ageMin = 1) => ({
    id, type: "COURSE_ASSIGNED", title, body: "b", link: `/class/settings?course=${C1}`, course_id: C1, read_at: read ? now.toISOString() : null, created_at: new Date(now.getTime() - ageMin * 60_000).toISOString(),
  });
  let state = [mk("n1", "Bạn được phân công lớp An ninh mạng – 761987", false, 5)];
  const reads: string[] = [];
  let lists = 0;
  await page.route("**/api/v1/notifications**", async (r) => {
    const req = r.request();
    if (req.method() === "POST") {
      const id = new URL(req.url()).pathname.split("/").at(-2) ?? "";
      reads.push(id);
      state = state.map((n) => (n.id === id ? { ...n, read_at: new Date().toISOString() } : n));
      return fulfill(r, 204, null);
    }
    lists++;
    return fulfill(r, 200, { items: state, next_cursor: null, unread_count: state.filter((n) => !n.read_at).length });
  });
  await page.goto("/");
  const dot = page.locator("[data-part=bell-dot]");
  await expect(dot).toHaveCount(1);
  await page.getByRole("button", { name: /^Thông báo, 1 chưa đọc/ }).click();
  const panel = page.locator("[data-part=notifications]");
  await expect(panel).toContainText("Bạn được phân công lớp An ninh mạng – 761987");
  await expect(panel).toContainText("Phân công lớp · 5 phút trước");
  await expect(dot).toHaveCount(1); // mở khung không xoá chấm

  // làm mới 30 s: thông báo mới xuất hiện
  const before = lists;
  state = [mk("n2", "Thông báo thứ hai", false, 0), ...state];
  await page.clock.runFor(31_000);
  await expect.poll(() => lists).toBeGreaterThan(before);
  await expect(page.getByRole("button", { name: /^Thông báo, 2 chưa đọc/ })).toBeVisible();

  await panel.getByRole("link", { name: /Bạn được phân công lớp An ninh mạng/ }).click();
  await expect.poll(() => reads).toEqual(["n1"]);
  // `?course=` là lớp của mình ⇒ được chọn rồi BỎ khỏi URL (SRS 4.9); đường dẫn đúng là link của thông báo
  await expect(page).toHaveURL(`${BASE_URL}/class/settings`);
  await expect(picker(page)).toContainText("761987");
  await expect(page.getByRole("button", { name: /^Thông báo, 1 chưa đọc/ })).toBeVisible();
});

test("bell real: lỗi tải → dòng 'Chưa tải được thông báo.' không chặn app; hết chưa đọc thì chấm biến", async ({ page }) => {
  await asJwt(page, "TEACHER");
  await page.route("**/api/v1/me/courses**", (r) => fulfill(r, 200, { items: [], next_cursor: null }));
  await page.route("**/api/v1/notifications**", (r) => fulfill(r, 503, apiErr("SERVICE_UNAVAILABLE")));
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 }).first()).toBeVisible();
  await page.getByRole("button", { name: /^Thông báo/ }).click();
  await expect(page.locator("[data-part=notifications]")).toContainText("Chưa tải được thông báo.");
  await expect(page.locator("[data-part=bell-dot]")).toHaveCount(0);
});

test("@real course picker với seed: GV 2 lớp, SV A 2 lớp, SV B 1 lớp, SV D chưa có lớp", async () => {
  test.skip(true, "@real — cần compose + seed (US-P2-12); chạy tay theo handoff dev-US-P2-07");
});
