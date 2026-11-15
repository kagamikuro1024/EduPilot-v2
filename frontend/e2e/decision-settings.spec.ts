import { expect, test, type Page } from "@playwright/test";
import { acquireFakeApi, releaseFakeApi } from "./support/fake-api-lock";
import { DEC, decision, decisionCourse, decisionGate, decisionProvider, err, getLog, openLlm, provider } from "./support/llm-fixtures";

// US-DM-06: khối «Quyết định nhanh» ở /settings/llm — máy chủ giả theo hợp đồng thật (SRS FEAT-decision-model 6). Không gọi gateway thật.
const P = "/api/v1/admin/llm";
test.describe.configure({ mode: "serial" });
test.beforeAll(acquireFakeApi);
test.afterAll(releaseFakeApi);
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng; máy chủ giả dùng chung");
});

const open = (page: Page, over: Record<string, unknown[]> = {}) =>
  openLlm(page, "ADMIN", { [`GET ${P}/providers`]: [{ body: { items: [provider(), decisionProvider()], env_fallback: { active: false, providers: [] } } }], ...over } as never);
const rows = (page: Page) => page.locator("[data-part=decision-gate]");
const puts = (page: Page, path: string) => getLog(page, path).then((l) => l.filter((x) => x.method === "PUT"));
const THREADS = `${P}/decision/gates/THREADS_PRECHECK`;

test("vị trí và hàng: một khối sau Chuỗi dự phòng; provider «Quyết định» · Giả; ba cổng theo thứ tự bảng; không câu mô tả", async ({ page }) => {
  await open(page);
  await expect(rows(page)).toHaveCount(3);
  const heads = await page.locator("[data-part=settings-section] h2:visible").allTextContents();
  expect(heads.slice(2, 5)).toEqual(["Chuỗi dự phòng", "Quyết định nhanh", "Mô hình tìm kiếm tài liệu"]);
  const block = page.locator("[data-part=decision-section]");
  await expect(block.locator("[data-part=decision-provider-status]")).toHaveText("Giả");
  await expect(rows(page)).toHaveCount(3);
  await expect(rows(page).locator("label")).toHaveText(["Kiểm duyệt Threads", "Cờ chèn lệnh khi chấm", "Định tuyến câu hỏi chat"]);
  await expect(rows(page).locator("[data-part=decision-gate-state]")).toHaveText(["Tắt", "Tắt", "Tắt"]);
  // nhà «Quyết định» không lẫn vào danh sách nhà cung cấp
  await expect(page.locator("[data-part=provider-row]")).toHaveCount(1);
  expect(await block.getByRole("button", { name: "Test kết nối" }).count()).toBe(1);
  expect(await block.locator("p").evaluateAll((ps) => ps.filter((p) => /^(Cổng|Bật|Tắt)\b.{40,}/.test(p.textContent ?? "")).length)).toBe(0);
});

test("công tắc cổng: lạc quan + «Hoàn tác» ghi ngay; không toast «Thành công»", async ({ page }) => {
  await open(page, {
    [`PUT ${THREADS}`]: [
      { body: decisionGate("THREADS_PRECHECK", { enabled: true, version: 2 }) },
      { body: decisionGate("THREADS_PRECHECK", { enabled: false, version: 3 }) },
    ],
  });
  const row = rows(page).first();
  await row.getByRole("switch", { name: "Kiểm duyệt Threads" }).click();
  await expect(row.getByRole("switch")).toHaveAttribute("aria-checked", "true");
  await expect(row.getByText("Đã bật Kiểm duyệt Threads")).toBeVisible();
  await expect(row.locator("[data-part=decision-gate-state]")).toContainText("Bật ở 0 lớp");
  await expect(page.locator('[role="status"]:has-text("Thành công")')).toHaveCount(0);
  expect(JSON.parse((await puts(page, THREADS))[0].body)).toEqual({ enabled: true, version: 1 });
  await row.getByRole("button", { name: "Hoàn tác" }).click();
  await expect(row.getByRole("switch")).toHaveAttribute("aria-checked", "false");
  await expect.poll(async () => (await puts(page, THREADS)).length).toBe(2);
  expect(JSON.parse((await puts(page, THREADS))[1].body)).toEqual({ enabled: false, version: 2 });
});

test("409 VERSION_CONFLICT: tải lại hàng và báo tại chỗ, không đè im lặng", async ({ page }) => {
  await open(page, { [`PUT ${THREADS}`]: [err(409, "VERSION_CONFLICT", { details: { current_version: 5 } })] });
  const row = rows(page).first();
  await row.getByRole("switch", { name: "Kiểm duyệt Threads" }).click();
  await expect(row.getByText("Cài đặt vừa được người khác đổi. Đã tải lại.")).toBeVisible();
  await expect.poll(async () => (await getLog(page, `${P}/decision`)).filter((l) => l.method === "GET").length).toBeGreaterThanOrEqual(2);
});

test("tooltip hệ quả chỉ khi provider THẬT; bản giả không có", async ({ page }) => {
  await open(page);
  await expect(rows(page).locator("[role=tooltip]")).toHaveCount(0);
  await open(page, { [`GET ${P}/decision`]: [{ body: decision({ provider: { id: DEC.prov, model: "gemini-decision-1", status: "connected", fake: false }, real_allowed: true }) }] });
  await expect(rows(page).first().locator("[role=tooltip]")).toHaveText("Gửi nội dung đã che tới nhà cung cấp");
});

test("cài đặt nâng cao: ngưỡng / thời hạn có placeholder khoảng; 422 báo tại ô; dấu phẩy được nhận", async ({ page }) => {
  await open(page, {
    [`PUT ${THREADS}`]: [
      { status: 422, body: { code: "VALIDATION_FAILED", message: "m", trace_id: "c".repeat(32), details: [{ field: "threshold", code: "OUT_OF_RANGE", message: "Ngưỡng từ 0.01 đến 0.20." }] } },
      { body: decisionGate("THREADS_PRECHECK", { threshold: "0.100", version: 2 }) },
    ],
  });
  const row = rows(page).first();
  await row.getByRole("button", { name: "Cài đặt nâng cao" }).click();
  const th = row.getByLabel("Ngưỡng ε");
  await expect(th).toHaveAttribute("placeholder", "0,01–0,20");
  await expect(row.getByLabel("Thời hạn (ms)")).toHaveAttribute("placeholder", "200–5000");
  await th.fill("0,9");
  await row.getByRole("button", { name: "Lưu", exact: true }).click();
  await expect(row.getByText("Ngưỡng từ 0.01 đến 0.20.")).toBeVisible();
  await th.fill("0,1");
  await row.getByRole("button", { name: "Lưu", exact: true }).click();
  await expect(row.getByText("Đã lưu cài đặt nâng cao")).toBeVisible();
  const last = JSON.parse((await puts(page, THREADS)).at(-1)!.body);
  expect(last).toEqual({ threshold: "0.1", timeout_ms: 800, version: 1 });
});

test("theo lớp: công tắc khoá khi cổng tắt; bật cổng thì bật được từng lớp; tìm theo tên / mã", async ({ page }) => {
  const COURSES = `${P}/decision/courses`;
  await open(page, { [`GET ${COURSES}`]: [{ body: { items: [decisionCourse(1), decisionCourse(2, { enabled: true, version: 3 })], next_cursor: null } }] });
  const row = rows(page).first();
  await row.getByRole("button", { name: "Theo lớp" }).click();
  const table = row.locator("[data-part=decision-courses]");
  await expect(table.getByRole("switch")).toHaveCount(2);
  await expect(table.getByRole("switch").first()).toBeDisabled();
  await expect(table.locator("[role=tooltip]").first()).toHaveText("Đang tắt toàn hệ thống");
  // cổng bật toàn hệ thống: mở khoá
  await open(page, {
    [`GET ${P}/decision`]: [{ body: decision({ gates: [decisionGate("THREADS_PRECHECK", { enabled: true, courses_enabled: 1 })] }) }],
    [`GET ${COURSES}`]: [{ body: { items: [decisionCourse(1), decisionCourse(2, { enabled: true, version: 3 })], next_cursor: null } }],
    [`PUT /api/v1/courses/${decisionCourse(1).course_id}/decision-gates/THREADS_PRECHECK`]: [{ body: { gate: "THREADS_PRECHECK", enabled: true, effective: true, version: 1 } }],
  });
  const r2 = rows(page).first();
  await expect(r2.locator("[data-part=decision-gate-state]")).toContainText("Bật ở 1 lớp");
  await r2.getByRole("button", { name: "Theo lớp" }).click();
  const t2 = r2.locator("[data-part=decision-courses]");
  await expect(t2.getByRole("switch").first()).toBeEnabled();
  await t2.getByRole("switch").first().click();
  await expect.poll(async () => (await puts(page, `/api/v1/courses/${decisionCourse(1).course_id}/decision-gates/THREADS_PRECHECK`)).length).toBe(1);
  expect(JSON.parse((await puts(page, `/api/v1/courses/${decisionCourse(1).course_id}/decision-gates/THREADS_PRECHECK`))[0].body)).toEqual({ enabled: true, version: 0 });
  await t2.getByLabel("Tìm lớp").fill("IT200");
  await expect.poll(async () => (await getLog(page, COURSES)).some((l) => l.search.includes("q=IT200"))).toBe(true);
});

test("chọn bản thật khi chưa được phép: báo tại chỗ, giữ nguyên bản giả (DECISION_REAL_DISABLED)", async ({ page }) => {
  await open(page, { [`PUT ${P}/providers/${DEC.prov}`]: [err(409, "DECISION_REAL_DISABLED")] });
  const block = page.locator("[data-part=decision-provider]");
  await block.getByRole("radio", { name: "gemini-decision-1" }).click();
  await expect(block.getByText("Chưa được phép dùng bản thật.")).toBeVisible();
  await expect(block.locator("[data-part=decision-provider-status]")).toHaveText("Giả");
  await expect(block.getByRole("radio", { name: "decision-fake" })).toHaveAttribute("aria-checked", "true");
});

test("chưa có provider decision: «Chưa cấu hình» + một nút Thêm", async ({ page }) => {
  await openLlm(page, "ADMIN", {
    [`GET ${P}/decision`]: [{ body: decision({ provider: null }) }],
    [`POST ${P}/providers`]: [{ status: 201, body: decisionProvider() }],
  });
  const block = page.locator("[data-part=decision-provider]");
  await expect(block.locator("[data-part=decision-provider-status]")).toHaveText("Chưa cấu hình");
  await expect(block.getByRole("button")).toHaveCount(1);
  await block.getByRole("button", { name: "Thêm" }).click();
  await expect.poll(async () => (await getLog(page, `${P}/providers`)).filter((l) => l.method === "POST").length).toBe(1);
  const body = JSON.parse((await getLog(page, `${P}/providers`)).filter((l) => l.method === "POST")[0].body);
  expect(body).toMatchObject({ type: "decision", name: "Quyết định" });
  expect(body.models.map((m: { model: string; enabled: boolean }) => [m.model, m.enabled])).toEqual([["decision-fake", true], ["gemini-decision-1", false]]);
});

test("TEACHER không thấy khối và không gọi API decision (AC8)", async ({ page }) => {
  await openLlm(page, "TEACHER");
  await expect(page.getByRole("heading", { name: "Kết nối nhà cung cấp", level: 2 })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Quyết định nhanh" })).toHaveCount(0);
  expect((await getLog(page, `${P}/decision`)).length).toBe(0);
});
