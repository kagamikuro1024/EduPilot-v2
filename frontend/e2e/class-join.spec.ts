import { expect, test, type Page } from "@playwright/test";
import { settleGoto } from "./support/hydrate";
import { loadAudit, runAudit } from "./support/audit";
import { BASE_URL } from "./support/env";
import { asJwt, sessionBody, type JwtRole } from "./support/session";

type Json = Record<string, unknown>;

// US-P2-07 — bộ chọn lớp từ `GET /me/courses`, màn "Bạn chưa vào lớp nào", ánh xạ lớp mô phỏng. Gateway giả bằng page.route (phần @real chạy ở bản compose có seed).

test.beforeEach(({ page }) => settleGoto(page));

const C1 = "00000000-0000-7000-8000-00000000c001";
const C2 = "00000000-0000-7000-8000-00000000c002";
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };

const item = (id: string, code: string, role: "TEACHER" | "TA" | "STUDENT", status: "ACTIVE" | "PENDING" = "ACTIVE", courseStatus: "ACTIVE" | "ARCHIVED" = "ACTIVE") => ({
  course: { id, class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: courseStatus },
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

test("course picker: mặc định bỏ qua lớp đã lưu trữ; chọn tay một lớp thì tải lại vẫn giữ", async ({ page }) => {
  await login(page, "TEACHER", [item(C1, "900001", "TEACHER", "ACTIVE", "ARCHIVED"), item(C2, "761988", "TEACHER")]);
  await page.goto("/");
  await expect(picker(page)).toContainText("761988"); // lớp đầu danh sách đã lưu trữ nhưng không là mặc định
  const panel = await options(page);
  await panel.getByRole("menuitem", { name: /900001/ }).click();
  await expect(picker(page)).toContainText("900001");
  await page.reload();
  await expect(picker(page)).toContainText("900001");
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

// ===== US-P2-09: /join, /class/settings, /class/members =====

const NEW_CODE = "BX4P9TW";
type Handler = { status: number; body: unknown; headers?: Record<string, string> };

async function openJoin(page: Page, h: { preview?: (code: string) => Handler; join?: (code: string) => Handler; verified?: boolean } = {}) {
  await login(page, "STUDENT", []);
  const calls = { preview: [] as string[], join: [] as string[] };
  await page.route("**/api/v1/courses/join/preview", (r) => {
    const code = (JSON.parse(r.request().postData() ?? "{}") as { code: string }).code;
    calls.preview.push(code);
    const out = h.preview?.(code) ?? { status: 200, body: { name: "An ninh mạng", class_code: "761988", semester: "2026-2027-HK1", teachers: [{ full_name: "TS. Lê Thu Hà" }], state: "OPEN" } };
    return fulfill(r, out.status, out.body);
  });
  await page.route("**/api/v1/courses/join", (r) => {
    const code = (JSON.parse(r.request().postData() ?? "{}") as { code: string }).code;
    calls.join.push(code);
    const out = h.join?.(code) ?? { status: 200, body: { course_id: C2, status: "ACTIVE", already_member: false } };
    return fulfill(r, out.status, out.body);
  });
  return calls;
}

test("join page: ô nhập tự viết hoa và bỏ ký tự cấm, một nút chính, xem trước rồi tham gia", async ({ page }) => {
  const calls = await openJoin(page);
  await page.goto("/join");
  await expect(page.getByRole("heading", { name: "Tham gia lớp", level: 1 })).toBeVisible();
  const input = page.getByLabel("Mã tham gia");
  await input.fill(" bx 4p9 tw0 ILo1");
  await expect(input).toHaveValue(NEW_CODE);
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  const look = page.getByRole("button", { name: "Xem lớp" });
  await expect(look).toHaveAttribute("data-variant", "primary");
  await look.click();
  await expect(page.locator("[data-part=join-preview]")).toContainText("An ninh mạng · 761988 · TS. Lê Thu Hà · HK1 2026–2027");
  expect(calls.preview).toEqual([NEW_CODE]);
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1); // chỉ còn `Tham gia lớp`
  await page.getByRole("button", { name: "Tham gia lớp" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Bạn đã vào lớp." })).toBeVisible();
  expect(calls.join).toEqual([NEW_CODE]);
  expect((await page.locator("main").innerText()).toLowerCase()).not.toMatch(/\b(rag|pii|token|jwt|provider)\b/);
});

test("join page: cần duyệt, chờ duyệt, sai mã, đầy lớp, bị giới hạn đếm ngược, chưa xác minh email", async ({ page }) => {
  const calls = await openJoin(page, {
    preview: (code) => {
      if (code === "AAAAAAA") return { status: 404, body: apiErr("JOIN_CODE_INVALID") };
      if (code === "BBBBBBB") return { status: 429, body: { ...apiErr("RATE_LIMITED"), retry_after: 461 } };
      if (code === "CCCCCCC") return { status: 403, body: apiErr("EMAIL_NOT_VERIFIED") };
      if (code === "DDDDDDD") return { status: 200, body: { name: "An ninh mạng", class_code: "761988", semester: "2026-2027-HK1", teachers: [], state: "FULL" } };
      return { status: 200, body: { name: "An ninh mạng", class_code: "761988", semester: "2026-2027-HK1", teachers: [{ full_name: "TS. Lê Thu Hà" }], state: "REQUIRES_APPROVAL" } };
    },
    join: () => ({ status: 200, body: { course_id: C2, status: "PENDING", already_member: false } }),
  });
  await page.goto("/join");
  const input = page.getByLabel("Mã tham gia");
  const look = page.getByRole("button", { name: "Xem lớp" });

  await input.fill("AAAAAAA");
  await look.click();
  await expect(page.getByText("Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.")).toBeVisible();
  await expect(input).toHaveAttribute("aria-invalid", "true");

  await input.fill("CCCCCCC");
  await look.click();
  await expect(page.getByText("Hãy xác minh email trước khi vào lớp.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Gửi lại email xác minh" })).toHaveAttribute("href", "/verify-email");

  await input.fill("DDDDDDD");
  await look.click();
  await expect(page.locator("[data-part=join-preview]").getByText("Lớp đã đủ sĩ số. Hãy báo giảng viên.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Tham gia lớp" })).toHaveCount(0);

  await input.fill("BBBBBBB");
  await look.click();
  await expect(page.getByText(/Bạn đã thử quá nhiều lần\. Thử lại sau 07:4\d\./)).toBeVisible();
  await expect(look).toBeDisabled();

  await input.fill("EEEEEEE"); // sửa mã không bỏ khoá khi còn đếm ngược
  await expect(look).toBeDisabled();
  expect(calls.preview).toEqual(["AAAAAAA", "CCCCCCC", "DDDDDDD", "BBBBBBB"]);
});

test("join page: lớp cần duyệt hiện câu cần duyệt và kết quả chờ duyệt", async ({ page }) => {
  await openJoin(page, {
    preview: () => ({ status: 200, body: { name: "An ninh mạng", class_code: "761988", semester: "2026-2027-HK1", teachers: [], state: "REQUIRES_APPROVAL" } }),
    join: () => ({ status: 200, body: { course_id: C2, status: "PENDING", already_member: false } }),
  });
  await page.goto("/join");
  await page.getByLabel("Mã tham gia").fill(NEW_CODE);
  await page.getByRole("button", { name: "Xem lớp" }).click();
  await expect(page.getByText("Lớp này cần giảng viên duyệt.")).toBeVisible();
  await page.getByRole("button", { name: "Tham gia lớp" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi yêu cầu, chờ giảng viên duyệt." })).toBeVisible();
});

test("join page: 375 px sạch (tràn ngang, vùng chạm, chữ bị cắt)", async ({ page }) => {
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  await openJoin(page);
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("/join");
  await page.getByLabel("Mã tham gia").fill(NEW_CODE);
  await page.getByRole("button", { name: "Xem lớp" }).click();
  await expect(page.locator("[data-part=join-preview]")).toBeVisible();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(a.cut).toEqual([]);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("join link login redirect: /join/<mã> chưa đăng nhập → /login?next=… → quay lại, xem trước ngay, mã rời khỏi URL", async ({ page }) => {
  let logged = false;
  const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
  await page.route("**/api/v1/auth/refresh", (r) => (logged ? fulfill(r, 200, sessionBody("STUDENT")) : fulfill(r, 401, apiErr("UNAUTHENTICATED"))));
  await page.route("**/api/v1/auth/login", (r) => {
    logged = true;
    return r.fulfill({ status: 200, contentType: "application/json", headers: { ...cors, "Set-Cookie": "ep_rt=x; Path=/api/v1/auth; HttpOnly" }, body: JSON.stringify(sessionBody("STUDENT")) });
  });
  await page.route("**/api/v1/me/courses**", (r) => fulfill(r, 200, { items: [], next_cursor: null }));
  await page.route("**/api/v1/notifications**", (r) => fulfill(r, 200, { items: [], next_cursor: null, unread_count: 0 }));
  const previews: string[] = [];
  await page.route("**/api/v1/courses/join/preview", (r) => {
    previews.push((JSON.parse(r.request().postData() ?? "{}") as { code: string }).code);
    return fulfill(r, 200, { name: "An ninh mạng", class_code: "761988", semester: "2026-2027-HK1", teachers: [{ full_name: "TS. Lê Thu Hà" }], state: "OPEN" });
  });
  await page.goto(`/join/${NEW_CODE}`);
  await expect(page).toHaveURL(`${BASE_URL}/login?next=%2Fjoin%2F${NEW_CODE}`);
  await page.getByLabel("Email").fill("sv.moi@edupilot.local");
  await page.getByLabel("Mật khẩu", { exact: true }).fill("Edupilot#2026-demo");
  await page.getByRole("button", { name: "Đăng nhập", exact: true }).click();
  await expect(page.locator("[data-part=join-preview]")).toContainText("An ninh mạng · 761988");
  expect(previews).toEqual([NEW_CODE]);
  await expect(page).toHaveURL(`${BASE_URL}/join`); // mã không còn trong thanh địa chỉ
});

test("join page: /join/<mã> gửi Referrer-Policy no-referrer và Cache-Control no-store", async ({ request }) => {
  const res = await request.get(`/join/${NEW_CODE}`);
  expect(res.headers()["referrer-policy"]).toBe("no-referrer");
  expect(res.headers()["cache-control"]).toContain("no-store");
  const res2 = await request.get("/join");
  expect(res2.headers()["referrer-policy"]).toBe("no-referrer");
});

// --- /class/settings ---

type Info = { join_code: string; join_url: string; enabled: boolean; expires_at: string | null; require_approval: boolean; allowed_email_domain: string | null; capacity: number | null; active_students: number; pending: number; version: number };
const info = (over: Partial<Info> = {}): Info => ({ join_code: "AN7K2MQ", join_url: "https://localhost/join/AN7K2MQ", enabled: true, expires_at: null, require_approval: false, allowed_email_domain: null, capacity: null, active_students: 24, pending: 2, version: 1, ...over });

async function openSettings(page: Page, role: "TEACHER" | "TA", h: { put?: (b: Json) => Handler; before?: () => Promise<void> } = {}) {
  await login(page, role, [item(C1, "761988", role)]);
  await h.before?.();
  let cur = info();
  const log = { put: [] as Json[], regen: 0, gets: 0 };
  await page.route(`**/api/v1/courses/${C1}/join-code`, (r) => { log.gets++; return fulfill(r, 200, cur); });
  await page.route(`**/api/v1/courses/${C1}/join-code/regenerate`, (r) => {
    log.regen++;
    cur = info({ join_code: NEW_CODE, join_url: `https://localhost/join/${NEW_CODE}`, version: cur.version + 1 });
    return fulfill(r, 200, { join_code: NEW_CODE, join_url: cur.join_url, version: cur.version });
  });
  await page.route(`**/api/v1/courses/${C1}/join-settings`, (r) => {
    const b = JSON.parse(r.request().postData() ?? "{}") as Json;
    log.put.push(b);
    const out = h.put?.(b) ?? { status: 200, body: (cur = info({ ...cur, enabled: b.enabled as boolean, require_approval: b.require_approval as boolean, capacity: b.capacity as number | null, allowed_email_domain: b.allowed_email_domain as string | null, version: cur.version + 1 })) };
    return fulfill(r, out.status, out.body);
  });
  await page.goto("/class/settings");
  await expect(page.getByRole("heading", { name: "Mã và cài đặt tham gia", level: 1 })).toBeVisible();
  return log;
}

test("class settings page: mã lớn, sao chép, tạo lại mã có xác nhận nêu số, lưu cài đặt một nút chính", async ({ page, context }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const log = await openSettings(page, "TEACHER");
  await expect(page.locator("[data-part=join-code]")).toHaveText("AN7K2MQ");
  await page.getByRole("button", { name: "Sao chép mã" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã sao chép" })).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe("AN7K2MQ");
  await page.getByRole("button", { name: "Sao chép liên kết" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe("https://localhost/join/AN7K2MQ");

  const regen = page.getByRole("button", { name: "Tạo lại mã" });
  await expect(regen).not.toHaveAttribute("data-variant", "primary");
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1); // `Lưu cài đặt`
  await regen.click();
  const dlg = page.getByRole("dialog");
  await expect(dlg).toContainText("Tạo lại mã cho lớp 761988?");
  await expect(dlg).toContainText("Mã cũ sẽ ngừng hoạt động ngay. 24 sinh viên đang ở trong lớp không bị ảnh hưởng.");
  await dlg.getByRole("button", { name: "Tạo lại mã" }).click();
  await expect(page.locator("[data-part=join-code]")).toHaveText(NEW_CODE);
  expect(log.regen).toBe(1);

  await page.getByRole("switch", { name: "Cần giảng viên duyệt" }).click();
  await page.getByLabel("Giới hạn sĩ số").fill("40");
  await page.getByLabel("Giới hạn tên miền email").fill("ptit.edu.vn");
  await page.getByRole("button", { name: "Lưu cài đặt" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã lưu cài đặt." })).toBeVisible();
  expect(log.put[0]).toMatchObject({ enabled: true, require_approval: true, capacity: 40, allowed_email_domain: "ptit.edu.vn", expires_at: null, version: 2 });
});

test("class settings page: lỗi tại ô và xung đột phiên bản", async ({ page }) => {
  let n = 0;
  const log = await openSettings(page, "TEACHER", {
    put: () => {
      n++;
      if (n === 1) return { status: 422, body: apiErr("VALIDATION_FAILED", [{ field: "capacity", code: "CAPACITY_BELOW_ACTIVE", message: "Sĩ số không được nhỏ hơn số sinh viên đang học." }]) };
      if (n === 2) return { status: 409, body: apiErr("VERSION_CONFLICT", { current_version: 7, current: info({ version: 7, capacity: 99 }) }) };
      return { status: 200, body: info({ version: 8 }) };
    },
  });
  await page.getByLabel("Giới hạn sĩ số").fill("3");
  await page.getByRole("button", { name: "Lưu cài đặt" }).click();
  await expect(page.getByText("Sĩ số không được nhỏ hơn số sinh viên đang học.")).toBeVisible();
  await expect(page.getByLabel("Giới hạn sĩ số")).toHaveValue("3"); // chữ đã gõ giữ nguyên
  await page.getByRole("button", { name: "Lưu cài đặt" }).click();
  await expect(page.getByText("Cài đặt này vừa được người khác đổi. Giữ thay đổi của bạn hay dùng bản mới?")).toBeVisible();
  await page.getByRole("button", { name: "Giữ thay đổi của tôi" }).click();
  await expect.poll(() => log.put.length).toBe(3);
  expect(log.put[2]).toMatchObject({ version: 7 });
});

test("class settings page: TA thấy mã nhưng chỉ đọc; sinh viên bị chặn và không gọi join-code", async ({ page }) => {
  await openSettings(page, "TA");
  await expect(page.locator("[data-part=join-code]")).toHaveText("AN7K2MQ");
  await expect(page.getByRole("button", { name: "Tạo lại mã" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Lưu cài đặt" })).toHaveCount(0);
  await expect(page.getByText("Chỉ giảng viên được thay đổi.")).toBeVisible();
  await expect(page.getByLabel("Giới hạn sĩ số")).toBeDisabled();
});

test("class settings page: sinh viên → màn chặn quyền, 0 yêu cầu tới join-code", async ({ page }) => {
  await login(page, "STUDENT", [item(C1, "761988", "STUDENT")]);
  const hits: string[] = [];
  await page.route("**/api/v1/courses/**", (r) => { hits.push(r.request().url()); return fulfill(r, 403, apiErr("FORBIDDEN")); });
  await page.goto("/class/settings");
  await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
  expect(hits).toEqual([]);
});

// --- /class/members ---

type Mem = { user_id: string; full_name: string; email: string; student_code: string; role_in_course: string; status: string; joined_via: string; warning: string | null; status_changed_at: string };
const mem = (id: string, name: string, status = "ACTIVE", over: Partial<Mem> = {}): Mem => ({
  user_id: id, full_name: name, email: `${id}@sv.example`, student_code: `B20DC${id.replace(/\D/g, "").padStart(5, "0")}`, role_in_course: "STUDENT", status, joined_via: "CODE", warning: null, status_changed_at: "2026-10-03T02:00:00Z", ...over,
});

async function openMembers(page: Page, role: "TEACHER" | "TA", rows: Mem[], url = "/class/members") {
  await login(page, role, [item(C1, "761988", role)]);
  const state = { rows, ops: [] as string[], bodies: [] as Json[] };
  await page.route(`**/api/v1/courses/${C1}/members**`, (r) => {
    const req = r.request();
    const u = new URL(req.url());
    const parts = u.pathname.split("/").filter(Boolean); // api v1 courses id members [uid [op]]
    const uid = parts[5];
    if (req.method() === "GET" && !uid) {
      const st = u.searchParams.get("status"), ro = u.searchParams.get("role"), q = (u.searchParams.get("q") ?? "").toLowerCase();
      const items = state.rows.filter((m) => (!st || m.status === st) && (!ro || m.role_in_course === ro) && (!q || m.full_name.toLowerCase().includes(q)));
      const students = state.rows.filter((m) => m.role_in_course === "STUDENT");
      return fulfill(r, 200, { items, next_cursor: null, counts: { active: students.filter((m) => m.status === "ACTIVE").length, pending: students.filter((m) => m.status === "PENDING").length } });
    }
    const op = req.method() === "DELETE" ? "delete" : (parts[6] ?? "");
    state.ops.push(`${op} ${uid}`);
    state.bodies.push(JSON.parse(req.postData() || "{}") as Json);
    const to = op === "approve" || op === "undo" ? (op === "approve" ? "ACTIVE" : "PENDING") : "REMOVED";
    state.rows = state.rows.map((m) => (m.user_id === uid ? { ...m, status: to } : m));
    return fulfill(r, 200, { user_id: uid, status: to, previous_status: null, warning: null });
  });
  await page.route(`**/api/v1/courses/${C1}/assistant-candidates**`, (r) => fulfill(r, 200, { items: [{ id: "ta9", full_name: "Lê Trợ Giảng", email: "ta9@x.test" }] }));
  await page.goto(url);
  await expect(page.getByRole("heading", { name: "Thành viên lớp", level: 1 })).toBeVisible();
  return state;
}

test("class members page: bảng thành viên, tìm, đếm tab, mời ra có Hoàn tác (chỉ giảng viên)", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung; 375 px kiểm trong ca riêng");
  const st = await openMembers(page, "TEACHER", [mem("s1", "Nguyễn Văn An"), mem("s2", "Trần Thị Bình"), mem("s3", "Lê Cường", "PENDING")]);
  await expect(page.locator("table tbody tr")).toHaveCount(2);
  await expect(page.getByRole("tab", { name: /Thành viên/ })).toContainText("2");
  await expect(page.getByRole("tab", { name: /Chờ duyệt/ })).toContainText("1");
  await page.getByLabel("Tìm thành viên").fill("bình");
  await expect(page.locator("table tbody tr")).toHaveCount(1);
  await page.getByLabel("Tìm thành viên").fill("");
  const row = page.locator("table tbody tr").filter({ hasText: "Nguyễn Văn An" });
  await row.getByRole("button", { name: /Thao tác cho Nguyễn Văn An/ }).click();
  await page.getByRole("menuitem", { name: "Mời ra khỏi lớp" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã mời Nguyễn Văn An ra khỏi lớp" })).toBeVisible();
  await expect(page.locator("table tbody tr").filter({ hasText: "Nguyễn Văn An" })).toHaveCount(0);
  expect(st.ops).toEqual(["delete s1"]);
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect.poll(() => st.ops).toEqual(["delete s1", "undo s1"]);
  await expect(page.getByRole("tab", { name: /Trợ giảng/ })).toBeVisible();
});

test("class members page: chọn nhiều → Duyệt N, Hoàn tác gọi undo cho từng người; từ chối; ?tab=pending", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung");
  const st = await openMembers(page, "TA", [mem("p1", "Chờ Một", "PENDING"), mem("p2", "Chờ Hai", "PENDING"), mem("p3", "Chờ Ba", "PENDING"), mem("s1", "Đã Vào")], "/class/members?tab=pending");
  await expect(page.getByRole("tab", { name: /Chờ duyệt/ })).toHaveAttribute("aria-selected", "true");
  await expect(page.locator("table tbody tr")).toHaveCount(3);
  await expect(page.getByRole("tab", { name: /Trợ giảng/ })).toHaveCount(0); // TA không quản lý trợ giảng
  await page.getByRole("checkbox", { name: /Chọn tất cả/i }).check();
  await page.getByRole("button", { name: "Duyệt 3" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã duyệt 3 yêu cầu" })).toBeVisible();
  await expect(page.locator("table tbody tr")).toHaveCount(0);
  expect(st.ops.sort()).toEqual(["approve p1", "approve p2", "approve p3"]);
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect.poll(() => st.ops.filter((o) => o.startsWith("undo")).sort()).toEqual(["undo p1", "undo p2", "undo p3"]);
  await expect(page.locator("table tbody tr")).toHaveCount(3);
  await page.locator("table tbody tr").filter({ hasText: "Chờ Một" }).getByRole("button", { name: "Từ chối" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã từ chối Chờ Một" })).toBeVisible();
  expect(st.ops.at(-1)).toBe("reject p1");
});

test("class members page: hàng email chưa khớp MSSV — giảng viên xác nhận ngắn rồi duyệt; TA không thấy nút Duyệt", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung");
  const rows = [mem("m1", "Trùng Mssv", "PENDING", { warning: "EMAIL_MISMATCH" })];
  const st = await openMembers(page, "TEACHER", rows, "/class/members?tab=pending");
  const row = page.locator("table tbody tr").filter({ hasText: "Trùng Mssv" });
  await expect(row.getByText("Email chưa khớp MSSV")).toBeVisible();
  await row.getByRole("button", { name: "Duyệt" }).click();
  await expect(page.getByText("Email của người này khác email trong danh sách lớp. Chỉ duyệt nếu bạn chắc đúng là sinh viên này.")).toBeVisible();
  expect(st.ops).toEqual([]); // chưa gọi gì cho tới khi xác nhận
  await page.getByRole("button", { name: "Duyệt Trùng Mssv" }).click();
  await expect.poll(() => st.ops).toEqual(["approve m1"]);
  expect(st.bodies[0]).toEqual({ confirm_mismatch: true });
});

test("class members page: TA không có nút Duyệt ở hàng email chưa khớp", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung");
  await openMembers(page, "TA", [mem("m1", "Trùng Mssv", "PENDING", { warning: "EMAIL_MISMATCH" })], "/class/members?tab=pending");
  const row = page.locator("table tbody tr").filter({ hasText: "Trùng Mssv" });
  await expect(row.getByText("Email chưa khớp MSSV")).toBeVisible();
  await expect(row.getByRole("button", { name: /Duyệt/ })).toHaveCount(0);
  await expect(row.getByText("Chờ giảng viên duyệt")).toBeVisible();
});

test("class members page: rỗng, tab Trợ giảng (thêm / bớt), sinh viên bị chặn, 375 px sạch", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung");
  const st = await openMembers(page, "TEACHER", [mem("t1", "Trợ Giảng Một", "ACTIVE", { role_in_course: "TA" })], "/class/members?tab=pending");
  await expect(page.getByText("Chưa có yêu cầu nào chờ duyệt.")).toBeVisible();
  const puts: Json[] = [];
  await page.route(`**/api/v1/courses/${C1}/assistants`, (r) => { puts.push(JSON.parse(r.request().postData() ?? "{}") as Json); return fulfill(r, 200, { teacher: null, assistants: [], changed: { teacher: false, added_ta: [], removed_ta: [] } }); });
  await page.getByRole("tab", { name: "Trợ giảng" }).click();
  await expect(page.locator("table tbody tr").filter({ hasText: "Trợ Giảng Một" })).toBeVisible();
  await page.getByLabel("Chọn trợ giảng").selectOption("ta9");
  await page.getByRole("button", { name: "Thêm" }).click();
  await expect.poll(() => puts).toEqual([{ ta_ids: ["t1", "ta9"] }]);
  await page.locator("table tbody tr").filter({ hasText: "Trợ Giảng Một" }).getByRole("button", { name: "Bớt" }).click();
  await expect.poll(() => puts.at(-1)).toEqual({ ta_ids: [] });
  void st;
  await page.setViewportSize({ width: 375, height: 800 });
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("class members page: sinh viên → màn chặn quyền", async ({ page }) => {
  await login(page, "STUDENT", [item(C1, "761988", "STUDENT")]);
  await page.goto("/class/members");
  await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
});

const REPORT = {
  total: 30, created_users: 28, linked_existing: 0, already_member: 0, pending_unverified: 0, skipped_removed: 0, dry_run: true,
  errors: [
    { row: 7, field: "email", code: "INVALID_EMAIL", message: "Email không đúng dạng." },
    { row: 19, field: "full_name", code: "MISSING_NAME", message: "Thiếu họ và tên." },
  ],
};

test("roster import ui: chọn tệp → xem trước đúng dòng 7 và 19 → Nhập 28 sinh viên; TA không thấy tab", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "375 px kiểm ở ca sau");
  await openMembers(page, "TEACHER", [mem("s1", "Nguyễn Văn An")]);
  const calls: { dry: boolean; key: string | undefined; ctype: string | undefined; invites: boolean }[] = [];
  await page.route(`**/api/v1/courses/${C1}/roster/import**`, (r) => {
    const req = r.request();
    const dry = new URL(req.url()).searchParams.get("dry_run") === "true";
    calls.push({ dry, key: req.headers()["idempotency-key"], ctype: req.headers()["content-type"], invites: (req.postData() ?? "").includes("true") });
    return fulfill(r, 200, dry ? REPORT : { ...REPORT, dry_run: false });
  });
  await page.getByRole("tab", { name: "Nhập danh sách" }).click();
  await expect(page.getByText("Thả tệp CSV hoặc XLSX vào đây.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Chọn tệp" })).toBeVisible();
  await expect(page.getByLabel("Gửi thư mời cho sinh viên chưa có tài khoản")).toBeChecked();
  await expect(page.getByRole("button", { name: "Xem trước" })).toHaveCount(0);
  await page.getByLabel("Tệp danh sách lớp").setInputFiles("e2e/fixtures/roster-2-loi.csv");
  await page.getByRole("button", { name: "Xem trước" }).click();
  await expect(page.getByRole("cell", { name: 'Dòng 7 · Email · "Email không đúng dạng."' })).toBeVisible();
  await expect(page.getByRole("cell", { name: 'Dòng 19 · Họ và tên · "Thiếu họ và tên."' })).toBeVisible();
  await expect(page.getByText(/28 sẽ được nhập/)).toBeVisible();
  const go = page.getByRole("button", { name: "Nhập 28 sinh viên" });
  await expect(go).toBeEnabled();
  expect(await page.getByRole("button", { name: "Xem trước" }).count()).toBe(0);
  await go.click();
  await expect(page.getByRole("status").filter({ hasText: "Đã nhập 28 sinh viên · 2 dòng lỗi chưa nhập." })).toBeVisible();
  expect(calls.map((c) => c.dry)).toEqual([true, false]);
  expect(calls[1].key).toMatch(/^ep-/);
  expect(calls[1].ctype).toContain("multipart/form-data; boundary=");
  await page.getByRole("tab", { name: "Thành viên" }).click();
  await expect(page.getByRole("tab", { name: "Nhập danh sách" })).toBeVisible();
});

test("roster import ui: TA không thấy tab", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "luồng giống nhau ở 375 px");
  await openMembers(page, "TA", [mem("s1", "Nguyễn Văn An")]);
  await expect(page.getByRole("tab", { name: "Nhập danh sách" })).toHaveCount(0);
});

test("roster import ui: tệp sai loại / quá lớn báo tại vùng thả; mất mạng giữ tệp và Gửi lại dùng cùng khoá", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "luồng giống nhau ở 375 px");
  await openMembers(page, "TEACHER", [mem("s1", "Nguyễn Văn An")], "/class/members?tab=import");
  const input = page.getByLabel("Tệp danh sách lớp");
  await input.setInputFiles({ name: "danh-sach.pdf", mimeType: "application/pdf", buffer: Buffer.from("%PDF-1.7") });
  await expect(page.getByText("Tệp phải là CSV hoặc XLSX.")).toBeVisible();
  await input.setInputFiles({ name: "lon.csv", mimeType: "text/csv", buffer: Buffer.alloc(2 * 1024 * 1024 + 1, 97) });
  await expect(page.getByText("Tệp quá lớn. Tối đa 2 MB.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Xem trước" })).toHaveCount(0);

  // 422 của máy chủ: nêu tên cột thiếu.
  await page.route(`**/api/v1/courses/${C1}/roster/import**`, (r) => fulfill(r, 422, apiErr("VALIDATION_FAILED", [{ field: "file", code: "MISSING_COLUMN", message: "Thiếu cột MSSV." }])));
  await input.setInputFiles({ name: "thieu.csv", mimeType: "text/csv", buffer: Buffer.from("Email,Họ và tên\na@x.test,A\n") });
  await page.getByRole("button", { name: "Xem trước" }).click();
  await expect(page.getByText("Thiếu cột MSSV.")).toBeVisible();
  await page.unroute(`**/api/v1/courses/${C1}/roster/import**`);

  // Mất mạng khi nhập thật: giữ tệp, "Gửi lại" dùng đúng khoá cũ, thành công thì hết thông báo.
  const keys: (string | undefined)[] = [];
  let down = true;
  await page.route(`**/api/v1/courses/${C1}/roster/import**`, (r) => {
    const req = r.request();
    if (new URL(req.url()).searchParams.get("dry_run") === "true") return fulfill(r, 200, { ...REPORT, errors: [], created_users: 2, total: 2 });
    keys.push(req.headers()["idempotency-key"]);
    if (down) return r.abort("internetdisconnected");
    return fulfill(r, 200, { ...REPORT, errors: [], created_users: 2, total: 2, dry_run: false });
  });
  await input.setInputFiles({ name: "hai.csv", mimeType: "text/csv", buffer: Buffer.from("Email,Họ và tên,MSSV\na@x.test,A,B20DC00001\nb@x.test,B,B20DC00002\n") });
  await page.getByRole("button", { name: "Xem trước" }).click();
  await page.getByRole("button", { name: "Nhập 2 sinh viên" }).click();
  await expect(page.getByText("Mất kết nối. Tệp vẫn được giữ")).toBeVisible();
  await expect(page.getByText("hai.csv")).toBeVisible();
  down = false;
  await page.getByRole("button", { name: "Gửi lại" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã nhập 2 sinh viên." })).toBeVisible();
  expect(keys.length).toBeGreaterThanOrEqual(2);
  expect(new Set(keys).size).toBe(1);
});

test("roster import ui: 375 px không tràn ngang", async ({ page }, info) => {
  test.skip(info.project.name !== "mobile", "chỉ dự án 375 px");
  await openMembers(page, "TEACHER", [mem("s1", "Nguyễn Văn An")], "/class/members?tab=import");
  await page.route(`**/api/v1/courses/${C1}/roster/import**`, (r) => fulfill(r, 200, REPORT));
  await page.getByLabel("Tệp danh sách lớp").setInputFiles("e2e/fixtures/roster-2-loi.csv");
  await page.getByRole("button", { name: "Xem trước" }).click();
  await expect(page.getByRole("button", { name: "Nhập 28 sinh viên" })).toBeVisible();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(a.cut).toEqual([]);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("share sources section: chỉ hiện khi có nguồn đủ điều kiện; dùng lại tại chỗ, không toast; TA không thấy", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "luồng giống nhau ở 375 px");
  let sources: Json[] = [{ id: C2, class_code: "761987", name: "An ninh mạng", semester: "2026-2027-HK1", documents: 6 }];
  const shares: Json[] = [];
  const mock = async () => {
  await page.route(`**/api/v1/courses/${C1}/share-sources`, (r) => fulfill(r, 200, { items: sources }));
  await page.route(`**/api/v1/courses/${C1}/share-from`, (r) => {
    shares.push(JSON.parse(r.request().postData() ?? "{}") as Json);
    return fulfill(r, 200, { shared: { documents: shares.length === 1 ? 6 : 0 }, skipped: [] });
  });
  };
  await openSettings(page, "TEACHER", { before: mock });
  await expect(page.getByRole("heading", { name: "Dùng lại nội dung từ lớp khác" })).toBeVisible();
  await expect(page.getByText(/Lớp 761987 · An ninh mạng/)).toBeVisible();
  await page.getByRole("button", { name: "Dùng lại tài liệu" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã dùng lại 6 tài liệu từ lớp 761987." })).toBeVisible();
  expect(shares).toEqual([{ source_course_id: C2, what: ["documents"] }]);
  await page.getByRole("button", { name: "Dùng lại tài liệu" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Lớp 761987 chưa có tài liệu để dùng lại." })).toBeVisible();
  await expect(page.locator("main [role=alert]")).toHaveCount(0);

  sources = [];
  await page.reload();
  await expect(page.getByRole("heading", { name: "Mã và cài đặt tham gia", level: 1 })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Dùng lại nội dung từ lớp khác" })).toHaveCount(0);
});

test("share sources section: TA không gọi share-sources và không thấy mục", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "luồng giống nhau ở 375 px");
  let called = 0;
  await openSettings(page, "TA", { before: async () => { await page.route(`**/api/v1/courses/${C1}/share-sources`, (r) => { called++; return fulfill(r, 200, { items: [] }); }); } });
  await expect(page.getByRole("heading", { name: "Dùng lại nội dung từ lớp khác" })).toHaveCount(0);
  expect(called).toBe(0);
});

test("@real course picker với seed: GV 2 lớp, SV A 2 lớp, SV B 1 lớp, SV D chưa có lớp", async () => {
  test.skip(true, "@real — cần compose + seed (US-P2-12); chạy tay theo handoff dev-US-P2-07");
});

test("@real roster import ui + share sources section (seed): GV lớp 2 nhập roster-2-loi.csv thấy dòng 7 và 19, nút Nhập 28 sinh viên; TA không thấy tab; GV khác học phần không thấy mục chia sẻ", async () => {
  test.skip(true, "@real — cần compose + seed (US-P2-12)");
});

test("@real join flow end to end (375 × 812): D xem trước và vào lớp 2, giảng viên duyệt, tạo lại mã, 6 mã sai ⇒ 429", async () => {
  test.skip(true, "@real — cần compose + seed (US-P2-12); phần tương đương chạy được: TestJoinGuessSimulation, TestRegenerateOldCodeDeadImmediately, TestApproveReject và các ca 'join page' / 'class members page'");
});
