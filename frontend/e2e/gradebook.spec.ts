import { expect, test, type Page } from "@playwright/test";
import { IMPACT, confirmedLop1, draftLop2, gradeApi } from "./support/grade-fixtures";
import { BASE_URL } from "./support/env";
import { asDemo, type DemoRole } from "./support/session";

// US-P6-04: `/gradebook/scheme` THẬT. Gateway giả theo hợp đồng thật (support/grade-fixtures.ts).
const desktopOnly = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "đo một lần");
const URL = "/gradebook/scheme";

async function open(page: Page, context: Parameters<typeof asDemo>[0], opts: Parameters<typeof gradeApi>[1] = {}, role: DemoRole = "teacher") {
  await asDemo(context, role);
  const api = await gradeApi(page, opts);
  await page.goto(URL);
  await expect(page.getByRole("heading", { name: "Công thức điểm", level: 1 })).toBeVisible();
  return api;
}
const block = (page: Page, id: string) => page.locator(`[data-block=${id}]`);
const confirmBtn = (page: Page) => page.getByRole("button", { name: "Xác nhận công thức" });

test("scheme review layout: một Panel, sáu khối đúng thứ tự, cột trích dẫn mở đúng trang, không câu mô tả", async ({ page, context }, info) => {
  desktopOnly(info);
  await page.setViewportSize({ width: 1440, height: 900 });
  await open(page, context);
  await expect(page.locator("main [data-ep-panel]")).toHaveCount(1);
  await expect(page.locator("[data-block] h3")).toHaveText(["Thành phần", "Điểm cộng", "Điểm trừ", "Làm tròn", "Thang chữ", "Liệt"]);
  const pane = page.locator("[data-part=citation-pane]");
  await expect(pane).toBeVisible();
  await block(page, "BONUS").getByRole("button", { name: "Trang 2" }).click();
  await expect(page.locator("[data-part=citation-page]")).toHaveText("Trang 2");
  await expect(pane.locator("mark")).toContainText("Mỗi lần phát biểu được cộng 0,25 điểm");
  await block(page, "components").getByRole("button", { name: "Trang 1" }).click();
  await expect(page.locator("[data-part=citation-page]")).toHaveText("Trang 1");
  await expect(pane.locator("mark")).toContainText("Điểm quá trình gồm bài kiểm tra hằng tuần");
  expect(await page.locator("main").innerText()).not.toMatch(/AI rút ra|Chọn một mục|nháp do/i);
  await expect(page.locator("main svg[aria-label*='Thông tin'], main [data-icon=info]")).toHaveCount(0);
});

test("unclear blocks confirm: mục chưa rõ ngay tại khối, nút khoá có lý do ở chính nút, điền xong thì mở", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  await expect(block(page, "ROUNDING")).toContainText("Quy chế chưa nêu cách làm tròn");
  const btn = confirmBtn(page);
  await expect(btn).toHaveAttribute("aria-disabled", "true");
  await btn.focus();
  await expect(page.getByRole("tooltip")).toHaveText("Còn 1 mục chưa rõ: làm tròn");
  await btn.click({ force: true });
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await block(page, "ROUNDING").getByRole("combobox").selectOption("0.1");
  await expect(block(page, "ROUNDING")).not.toContainText("Quy chế chưa nêu cách làm tròn");
  await expect(btn).not.toHaveAttribute("aria-disabled", "true");
  expect(api.puts()).toHaveLength(1);
  const sent = JSON.parse(api.puts()[0].body) as { rules: Array<{ kind: string; params: { step: string } }> };
  expect(sent.rules.find((r) => r.kind === "ROUNDING")?.params.step).toBe("0.1");
});

test("scheme edit optimistic: số sai báo tại ô và giữ chữ; số đúng lưu ngay khi rời ô, version + 1; 409 hoàn lại kèm dòng báo", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  const weight = block(page, "components").getByLabel("Trọng số").first();
  for (const bad of ["abc", "-5", "101"]) {
    await weight.fill(bad);
    await weight.blur();
    await expect(block(page, "components").getByRole("alert")).toHaveText("Nhập số từ 0 đến 100");
    await expect(weight).toHaveValue(bad);
  }
  expect(api.puts()).toHaveLength(0);
  await weight.fill("25,5");
  await weight.blur();
  await expect(block(page, "components")).toContainText("Tổng 105,5%");
  await expect.poll(() => api.puts().length).toBe(1);
  const first = JSON.parse(api.puts()[0].body) as { version: number; components: Array<{ weight: string }> };
  expect(first.version).toBe(1);
  expect(first.components[0].weight).toBe("25.5");
  await expect(block(page, "components")).toContainText("Tổng trọng số là 105.5");
  await block(page, "components").getByLabel("Trọng số").nth(1).fill("14,5");
  await block(page, "components").getByLabel("Trọng số").nth(1).blur();
  await expect.poll(() => api.puts().length).toBe(2);
  expect((JSON.parse(api.puts()[1].body) as { version: number }).version).toBe(2);
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);
});

test("scheme edit conflict: 409 VERSION_CONFLICT nạp bản hiện tại và báo bằng InlineNotice", async ({ page, context }, info) => {
  desktopOnly(info);
  const theirs = draftLop2({ version: 5, components: draftLop2().components.map((c) => (c.key === "weekly" ? { ...c, weight: "30" } : c)) });
  await open(page, context, { h: { "PUT grade-scheme": (r) => r.fulfill({ status: 409, contentType: "application/json", body: JSON.stringify({ code: "VERSION_CONFLICT", message: "x", trace_id: "t", details: { current_version: 5, current: theirs } }), headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true" } }) } });
  const w = block(page, "components").getByLabel("Trọng số").nth(1);
  await w.fill("15");
  await w.blur();
  await expect(page.getByText("Có người vừa sửa công thức. Đã tải bản mới.")).toBeVisible();
  await expect(block(page, "components").getByLabel("Trọng số").first()).toHaveValue("30");
});

test("impact preview: ba sinh viên mẫu trước → sau; lớp chưa đủ điểm hiện dòng chưa đủ điểm", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, { draft: draftLop2({ unclear_items: [] }), confirmed: confirmedLop1() });
  const rows = page.locator("[data-part=impact-samples] li");
  await expect(rows).toHaveCount(3);
  await expect(rows.nth(0)).toContainText("Sinh Viên 06");
  await expect(rows.nth(0)).toContainText("10 (A) → 9,8 (A)");
  await expect(rows.nth(2)).toContainText("3 (F) → 3,2 (F)");
});

test("impact preview empty: chưa đủ điểm", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, { impact: { ...IMPACT, students_computable: 0, changed_vs_confirmed: 0, samples: [] } });
  await expect(page.locator("[data-part=impact-empty]")).toHaveText("Chưa đủ điểm để xem thử");
});

test("confirm irreversible: hộp thoại nêu ba con số, gửi version + Idempotency-Key, xong đổi chữ tại chỗ (không toast)", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context, { draft: draftLop2({ unclear_items: [] }), confirmed: confirmedLop1() });
  await confirmBtn(page).click();
  const dlg = page.getByRole("dialog");
  await expect(dlg).toContainText("Áp cho 30 sinh viên · điểm học phần của 12 sinh viên sẽ đổi so với công thức hiện tại · 60 điểm đã nhập được giữ · 0 điểm đã nhập bị bỏ");
  await dlg.getByRole("button", { name: "Xác nhận công thức" }).click();
  await expect(page.locator("[data-part=confirmed-line]")).toContainText("Đã xác nhận ·");
  const post = api.calls.find((c) => c.method === "POST" && c.path.endsWith("/confirm"));
  expect(post?.headers["idempotency-key"]).toMatch(/^ep-/);
  expect(JSON.parse(post?.body ?? "{}")).toEqual({ version: 1 });
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.locator("[role=status]").filter({ hasText: /Thành công/ })).toHaveCount(0);
});

test("confirm irreversible dropped: số điểm bị bỏ và lý do hiện trong hộp thoại", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, { draft: draftLop2({ unclear_items: [] }), impact: { ...IMPACT, entries_carried: 30, entries_dropped: [{ item_key: "midterm", count: 30, reason: "MAX_CHANGED" }] } });
  await confirmBtn(page).click();
  await expect(page.getByRole("dialog")).toContainText("30 điểm đã nhập bị bỏ (midterm: 30, đổi thang)");
});

test("new policy diff: có bản nháp cạnh bản đã xác nhận → chế độ so sánh do máy chủ tính", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context, { draft: draftLop2({ unclear_items: [] }), confirmed: confirmedLop1() });
  await page.getByRole("button", { name: "So sánh" }).click();
  await expect(page.locator("[data-part=diff-rows]")).toContainText("midterm: 20% → 15%");
  await expect(page.getByText("Tổng trọng số 100% → 95%")).toBeVisible();
  expect(api.calls.some((c) => c.path.endsWith("grade-scheme?diff=confirmed"))).toBe(true);
  await page.getByRole("button", { name: "Bản nháp" }).click();
  await expect(page.locator("[data-block]")).toHaveCount(6);
});

test("scheme offline retry: mất mạng giữ chữ, báo Chưa lưu 1–2 từ, tự gửi lại khi có mạng", async ({ page, context }, info) => {
  desktopOnly(info);
  let n = 0;
  const api = await open(page, context, { h: { "PUT grade-scheme": (r, next) => (++n === 1 ? r.abort("connectionfailed") : next(r)) } });
  const w = block(page, "components").getByLabel("Trọng số").first();
  await w.fill("25");
  await w.blur();
  await expect(page.locator("[data-part=save-state]")).toHaveText("Chưa lưu");
  await expect(w).toHaveValue("25");
  await expect(page.locator("[data-part=save-state]")).toHaveCount(0, { timeout: 10_000 });
  expect(api.puts().length).toBeGreaterThanOrEqual(2);
  expect((JSON.parse(api.puts().at(-1)!.body) as { components: Array<{ weight: string }> }).components[0].weight).toBe("25");
});

test("scheme rbac: TA đọc được nhưng không có nút xác nhận, tải quy chế, ô điền; sinh viên bị chặn", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, { draft: draftLop2({ unclear_items: [] }) }, "ta");
  await expect(block(page, "components")).toContainText("Kiểm tra hằng tuần · 20% · Quá trình");
  await expect(confirmBtn(page)).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Tải quy chế" })).toHaveCount(0);
  await expect(page.locator("main input:not([type=hidden]), main select, main textarea")).toHaveCount(0);
  const stu = await context.newPage();
  await asDemo(context, "student");
  await gradeApi(stu);
  await stu.goto(URL);
  await expect(stu.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
  expect(await stu.content()).not.toContain("Thi cuối kỳ");
});

test("scheme states: rỗng có Tải quy chế, đang tải có khung xương, lỗi có nguyên nhân và Thử lại", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, { draft: null, confirmed: null });
  await expect(page.getByText("Lớp chưa có quy chế.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Tải quy chế" }).first()).toBeVisible();

  let n = 0;
  const p2 = await context.newPage();
  await gradeApi(p2, { h: { "GET grade-scheme": async (r, next) => { if (++n === 1) return r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x", trace_id: "t" }), headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true" } }); await new Promise((res) => setTimeout(res, 1500)); return next(r); } } });
  await p2.goto(URL);
  const retry = p2.getByRole("button", { name: "Thử lại" });
  await expect(retry).toBeVisible();
  await retry.click();
  await expect(p2.locator("[data-ep-skeleton], [aria-busy=true]").first()).toBeVisible();
  await expect(p2.getByRole("heading", { name: "Thành phần" })).toBeVisible();
});

test("scheme mobile: 375 px một cột, trích dẫn bằng Drawer, không tràn ngang, nút chính ≥ 44 px", async ({ page, context }, info) => {
  desktopOnly(info);
  await page.setViewportSize({ width: 375, height: 812 });
  await open(page, context, { draft: draftLop2({ unclear_items: [] }) });
  await expect(page.locator("[data-part=citation-pane]")).toHaveCount(0);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  const box = await confirmBtn(page).boundingBox();
  expect(box?.height ?? 0).toBeGreaterThanOrEqual(44);
  await block(page, "BONUS").getByRole("button", { name: "Trang 2" }).click();
  const drawer = page.getByRole("dialog");
  await expect(drawer).toContainText("Mỗi lần phát biểu được cộng 0,25 điểm");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
});

const PDF = { name: "quy-che.pdf", mimeType: "application/pdf", buffer: Buffer.from("%PDF-1.4\n% quy che\n") };
const okJson = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true" }, body: JSON.stringify(body) });

test("new policy upload: tải → đọc → trích có tiến độ, xong có bản nháp; trích dùng Idempotency-Key", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context, { draft: null, confirmed: null });
  let extracts = 0;
  let polls = 0;
  await page.route("**/api/v1/courses/*/uploads/presign", (r) => r.fulfill(okJson({ upload_id: "u1", method: "PUT", url: `${BASE_URL}/__blob/u1`, headers: {} })));
  await page.route("**/__blob/**", (r) => r.fulfill({ status: 200, body: "" }));
  await page.route("**/api/v1/courses/*/uploads/complete", (r) => r.fulfill(okJson({ job_id: "j1", document: { id: "00000000-0000-7000-8000-0000000007d1" } }, 202)));
  await page.route("**/api/v1/jobs/j1", (r) => r.fulfill(okJson({ id: "j1", status: "SUCCEEDED", progress: 100 })));
  await page.route("**/api/v1/jobs/j2", (r) => r.fulfill(okJson({ id: "j2", status: ++polls < 2 ? "RUNNING" : "SUCCEEDED", progress: polls < 2 ? 40 : 100 })));
  await page.route("**/api/v1/courses/*/grade-scheme/extract", (r) => {
    extracts++;
    api.set({ draft: draftLop2() });
    return r.fulfill(okJson({ job_id: "j2" }, 202));
  });
  await page.setInputFiles("[data-part=policy-input]", PDF);
  await expect(page.locator("[data-part=policy-progress]")).toBeVisible();
  await expect(page.locator("[data-block]")).toHaveCount(6, { timeout: 15_000 });
  expect(extracts).toBe(1);
  await expect(page.locator("[data-part=policy-progress]")).toHaveCount(0);
});

test("new policy upload failed: tải hỏng giữ tên tệp và có Tải lại", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, { draft: null, confirmed: null });
  await page.route("**/api/v1/courses/*/uploads/presign", (r) => r.fulfill(okJson({ upload_id: "u1", method: "PUT", url: `${BASE_URL}/__blob/u1`, headers: {} })));
  await page.route("**/__blob/**", (r) => r.abort("connectionfailed"));
  await page.setInputFiles("[data-part=policy-input]", PDF);
  await expect(page.getByRole("alert").filter({ hasText: "quy-che.pdf" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Tải lại" })).toBeVisible();
});
