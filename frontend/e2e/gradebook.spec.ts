import { readFileSync } from "node:fs";
import path from "node:path";
import { expect, test, type Page } from "@playwright/test";
import { IMPACT, confirmedLop1, draftLop2, gradeApi } from "./support/grade-fixtures";
import { EXAMCOL, FINAL, WEEKLY, gradebookApi, student as gbStudent, type Opts as SheetOpts } from "./support/gradebook-fixtures";
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
  await expect(pane.getByRole("link", { name: "Mở trang 2" })).toHaveAttribute("href", /\/__pdf\/quy-che\.pdf\?sig=1#page=2$/);
  await block(page, "components").getByRole("button", { name: "Trang 1" }).click();
  await expect(page.locator("[data-part=citation-page]")).toHaveText("Trang 1");
  await expect(pane.locator("mark")).toContainText("Điểm quá trình gồm bài kiểm tra hằng tuần");
  expect(await page.locator("main").innerText()).not.toMatch(/AI rút ra|Chọn một mục|nháp do/i);
  await expect(page.locator("main svg[aria-label*='Thông tin'], main [data-icon=info]")).toHaveCount(0);
  // QC BUG-P6-04-1: mặc định là chữ để duyệt; ô nhập chỉ ở mục chưa rõ (ở đây: bước làm tròn) hoặc mục vừa bấm vào
  await expect(page.locator("main input:visible, main select:visible, main textarea:visible")).toHaveCount(1);
  await expect(block(page, "components")).toContainText("Kiểm tra hằng tuần · 20% · Quá trình");
  await block(page, "components").getByRole("button", { name: "Kiểm tra hằng tuần · 20% · Quá trình" }).click();
  await expect(block(page, "components").getByLabel("Trọng số")).toHaveValue("20");
  await block(page, "components").getByRole("button", { name: "Xong" }).click();
  await expect(page.locator("main input:visible, main select:visible, main textarea:visible")).toHaveCount(1);
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
  await block(page, "components").locator("[data-part=edit-value]").first().click();
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
  await block(page, "components").locator("[data-part=edit-value]").nth(1).click();
  const w = block(page, "components").getByLabel("Trọng số");
  await w.fill("15");
  await w.blur();
  await expect(page.getByText("Có người vừa sửa công thức. Đã tải bản mới.")).toBeVisible();
  await expect(block(page, "components")).toContainText("Kiểm tra hằng tuần · 30% · Quá trình");
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
  await block(page, "components").locator("[data-part=edit-value]").first().click();
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

// ---- US-P6-06: `/gradebook` THẬT -------------------------------------------------------------------------------------------------------------------
async function sheet(page: Page, context: Parameters<typeof asDemo>[0], opts: SheetOpts = {}, role: DemoRole = "teacher") {
  await asDemo(context, role);
  const api = await gradebookApi(page, opts);
  await page.goto("/gradebook");
  await expect(page.getByRole("heading", { name: "Sổ điểm", level: 1 })).toBeVisible();
  return api;
}
const cellOf = (page: Page, i: number, item: string) => page.locator(`input[data-cell="${gbStudent(i).id}:${item}"]`);

test("gradebook layout: một Panel, cột Sinh viên cố định, cột theo item + Điểm cộng… Trạng thái, không thẻ KPI", async ({ page, context }, info) => {
  desktopOnly(info);
  await page.setViewportSize({ width: 1440, height: 900 });
  await sheet(page, context);
  await expect(page.locator("[data-part=grade-row]").first()).toBeVisible();
  await expect(page.locator("main [data-ep-panel]")).toHaveCount(1);
  const heads = await page.locator("thead th").allInnerTexts();
  expect(heads.map((h) => h.split("\n")[0])).toEqual(["Sinh viên", "Kiểm tra tuần", "Thi cuối kỳ", "Bài thi giữa kỳ", "Điểm cộng", "Điểm trừ", "Quá trình", "Điểm học phần", "Điểm chữ", "Trạng thái", "Hành động"]);
  expect(await page.locator("thead th").first().evaluate((e) => getComputedStyle(e).position)).toBe("sticky");
  await expect(page.getByRole("link", { name: "Xem công thức" })).toHaveAttribute("href", "/gradebook/scheme");
  await expect(page.locator("[data-part=grade-row]").first().locator("[data-part=row-status]")).toHaveText("Thiếu điểm");
});

test("official note: đúng một dòng ghi chú cố định dưới bảng", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context);
  await expect(page.getByText("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.")).toHaveCount(1);
});

test("cell edit keys: Enter lưu và xuống ô dưới, Tab lưu và sang phải, Esc hoàn tác, xoá ô = xoá điểm; lỗi giữ chữ", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context);
  const a = cellOf(page, 1, WEEKLY);
  await a.fill("8,5");
  await a.press("Enter");
  await expect.poll(() => api.puts().length).toBe(1);
  expect(JSON.parse(api.puts()[0].body)).toEqual({ item_id: WEEKLY, student_id: gbStudent(1).id, score: "8.5", version: null });
  await expect(cellOf(page, 2, WEEKLY)).toBeFocused();
  await expect(a).toHaveValue("8,5");
  const b = cellOf(page, 2, WEEKLY);
  await b.fill("7");
  await b.press("Tab");
  await expect(cellOf(page, 2, FINAL)).toBeFocused();
  await expect.poll(() => api.puts().length).toBe(2);
  const c = cellOf(page, 2, FINAL);
  await c.fill("9");
  await c.press("Escape");
  await expect(c).toHaveValue("");
  expect(api.puts()).toHaveLength(2);
  // lỗi: giữ chữ, không gửi
  await a.fill("8,333");
  await a.press("Enter");
  await expect(page.getByRole("alert").filter({ hasText: "Điểm có quá 2 chữ số thập phân" })).toBeVisible();
  await expect(a).toHaveValue("8,333");
  await a.fill("11");
  await a.press("Enter");
  await expect(page.getByRole("alert").filter({ hasText: "Điểm ngoài thang 0–10" })).toBeVisible();
  expect(api.puts()).toHaveLength(2);
  // xoá ô
  await a.fill("");
  await a.press("Enter");
  await expect.poll(() => api.puts().length).toBe(3);
  expect(JSON.parse(api.puts()[2].body)).toMatchObject({ score: null, version: 1 });
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);
});

test("cell edit instant: cột tính hiện … trong lúc chờ rồi đổi theo phản hồi của máy chủ", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context, { h: { "PUT grade-entries": async (r, next) => { await new Promise((res) => setTimeout(res, 700)); return next(r); } } });
  const row = page.locator("[data-part=grade-row]").first();
  await cellOf(page, 1, WEEKLY).fill("8");
  await cellOf(page, 1, WEEKLY).press("Tab");
  await cellOf(page, 1, FINAL).fill("6");
  await cellOf(page, 1, FINAL).press("Enter");
  await expect(row.locator("[data-part=row-status]")).toHaveText("…");
  await expect(row.locator("[data-part=row-status]")).toHaveText("Đủ", { timeout: 5000 });
  await expect(row.locator("td").nth(7)).toHaveText("7");
});

test("cell conflict choose: người sau nhận 409, hiện giá trị người kia, chọn Giữ của tôi hoặc Dùng của họ", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context);
  const key = `${gbStudent(1).id}:${WEEKLY}`;
  api.entries.set(key, { score: "6", version: 1, source: "MANUAL" });
  await page.reload();
  await expect(cellOf(page, 1, WEEKLY)).toHaveValue("6");
  api.entries.set(key, { score: "7.5", version: 2, source: "MANUAL" }); // người kia vừa nhập
  const a = cellOf(page, 1, WEEKLY);
  await expect(a).toHaveValue("6");
  await a.fill("9");
  await a.press("Enter");
  const pop = page.locator("[data-part=cell-conflict]");
  await expect(pop).toContainText("Thầy Nam vừa nhập 7,5 · 2 phút trước");
  await pop.getByRole("button", { name: "Giữ của tôi" }).click();
  await expect.poll(() => api.entries.get(key)?.score).toBe("9");
  expect(api.entries.get(key)?.version).toBe(3);
  await expect(page.locator("[data-part=grade-row]").first().locator("[data-part=row-status]")).not.toHaveText("…");
  // Dùng của họ
  api.entries.set(key, { score: "5", version: 4, source: "MANUAL" });
  await a.fill("8");
  await a.press("Enter");
  await expect(pop).toBeVisible();
  await pop.getByRole("button", { name: "Dùng của họ" }).click();
  await expect(pop).toHaveCount(0);
  await expect(a).toHaveValue("5");
});

test("explain student B: Giải trình từ menu dòng, từng dòng phép tính, căn cứ mở đoạn trích, dòng do giảng viên đặt", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context);
  await page.getByRole("button", { name: /^Hành động — Sinh Viên 0002/ }).click();
  await page.getByRole("menuitem", { name: "Giải trình" }).click();
  const calc = page.locator("[data-part=grade-calculation]");
  await expect(calc).toContainText("QT₀");
  await expect(calc).toContainText("= 7,75");
  await expect(calc).toContainText("= 0,50");
  await expect(calc).toContainText("do giảng viên đặt");
  await calc.getByRole("button", { name: /Căn cứ · tr\. 2/ }).click();
  await expect(calc).toContainText("Mỗi lần phát biểu được cộng 0,25 điểm.");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.getByRole("button", { name: /^Hành động — Sinh Viên 0001/ }).click();
  await page.getByRole("menuitem", { name: "Lịch sử" }).click();
  await expect(page.locator("[data-part=history-list]")).toContainText("weekly: 8 → 9");
  expect(await page.getByRole("button", { name: "Lịch sử", exact: true }).count()).toBe(0);
});

test("virtualized 1000: DOM ≤ 60 dòng, cuộn tải trang kế theo con trỏ", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context, { students: 1000 });
  expect(await page.locator("[data-part=grade-row]").count()).toBeLessThanOrEqual(60);
  const scroller = page.locator("[data-part=virtual-scroll]");
  for (let i = 0; i < 6; i++) {
    await scroller.evaluate((el) => { el.scrollTop = el.scrollHeight; });
    await page.waitForTimeout(150);
  }
  await expect.poll(() => api.calls.filter((c) => c.path.includes("gradebook?") && c.path.includes("cursor=")).length).toBeGreaterThan(0);
  expect(await page.locator("[data-part=grade-row]").count()).toBeLessThanOrEqual(60);
  expect(api.calls.every((c) => !c.path.includes("limit=1000"))).toBe(true);
});

test("type then scroll keeps text: gõ dở ở dòng 5, cuộn xa rồi cuộn về, chữ còn", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context, { students: 300 });
  const a = cellOf(page, 5, WEEKLY);
  await a.fill("7,5");
  const scroller = page.locator("[data-part=virtual-scroll]");
  await scroller.evaluate((el) => { el.scrollTop = 48 * 250; });
  await expect(page.locator(`input[data-cell="${gbStudent(5).id}:${WEEKLY}"]`)).toHaveCount(0);
  await scroller.evaluate((el) => { el.scrollTop = 0; });
  await expect(cellOf(page, 5, WEEKLY)).toHaveValue("7,5");
});

test("gradebook states: chưa công thức, lớp rỗng, lỗi có Thử lại", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context, { scheme: false });
  await expect(page.getByText("Chưa có công thức điểm.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Thiết lập công thức" })).toBeVisible();
  await expect(page.getByText("Công thức điểm chưa xác nhận")).toBeVisible();
  await expect(page.locator("input[data-cell]")).toHaveCount(0);

  const p2 = await context.newPage();
  await gradebookApi(p2, { students: 0 });
  await p2.goto("/gradebook");
  await expect(p2.getByText("Lớp chưa có sinh viên.")).toBeVisible();

  const p3 = await context.newPage();
  let n = 0;
  await gradebookApi(p3, { h: { "GET gradebook": (r, next) => (++n === 1 ? r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x", trace_id: "t" }), headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true" } }) : next(r)) } });
  await p3.goto("/gradebook");
  await p3.getByRole("button", { name: "Thử lại" }).click();
  await expect(p3.locator("[data-part=grade-row]").first()).toBeVisible();
});

test("offline cell: mất mạng giữ chữ, nhãn Chưa lưu tại ô, tự gửi lại khi có mạng", async ({ page, context }, info) => {
  desktopOnly(info);
  let n = 0;
  const api = await sheet(page, context, { h: { "PUT grade-entries": (r, next) => (++n === 1 ? r.abort("connectionfailed") : next(r)) } });
  const a = cellOf(page, 1, WEEKLY);
  await a.fill("8");
  await a.press("Enter");
  await expect(page.getByText("Chưa lưu", { exact: true })).toBeVisible();
  await expect(a).toHaveValue("8");
  await expect(page.getByText("Chưa lưu", { exact: true })).toHaveCount(0, { timeout: 10_000 });
  expect(api.puts().length).toBeGreaterThanOrEqual(2);
  expect(JSON.parse(api.puts().at(-1)!.body)).toMatchObject({ score: "8", version: null });
});

test("gradebook rbac: TA đọc bảng, giải trình, lịch sử nhưng không có ô nhập, menu cột, Chốt điểm; sổ đã chốt khoá mọi ô", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context, {}, "ta");
  await expect(page.locator("input[data-cell]")).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^Cột / })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^Hành động — Sinh Viên 0001/ })).toBeVisible();

  const p2 = await context.newPage();
  await gradebookApi(p2, { locked: true });
  await p2.goto("/gradebook");
  await expect(p2.locator("[data-part=row-status]").first()).toHaveText("Đã chốt v2");
  await expect(p2.locator("input[data-cell]")).toHaveCount(0);
});

test("publish column: menu cột công bố / gỡ; cột bài thi không có menu", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context);
  const head = page.locator("thead th").nth(1);
  await expect(head).toContainText("Chưa công bố");
  await head.getByRole("button", { name: "Cột Kiểm tra tuần" }).click();
  await page.getByRole("menuitem", { name: "Công bố cho sinh viên" }).click();
  await expect(head).toContainText("Đã công bố");
  expect(api.calls.some((c) => c.method === "POST" && c.path.endsWith("/publish") && JSON.parse(c.body).published === true)).toBe(true);
  await expect(page.locator("thead th").nth(3).getByRole("button")).toHaveCount(0);
  await head.getByRole("button", { name: "Cột Kiểm tra tuần" }).click();
  await expect(page.getByRole("menuitem", { name: "Gỡ công bố" })).toBeVisible();
  expect(EXAMCOL).toBeTruthy();
});

test("penalty not in scheme notice: một dòng cảnh báo không chặn, mở xem sinh viên liên quan; không có sự kiện thì không hiện", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context, { warnings: ["PENALTY_NOT_IN_SCHEME"] });
  const note = page.getByText("Có điểm trừ ghi tay không được tính theo công thức");
  await expect(note).toHaveCount(1);
  await page.getByRole("button", { name: "Xem", exact: true }).click();
  await expect(page.getByText("Sinh Viên 0001").first()).toBeVisible();
  await expect(cellOf(page, 1, WEEKLY)).toBeVisible();
  const p2 = await context.newPage();
  await gradebookApi(p2, {});
  await p2.goto("/gradebook");
  await expect(p2.getByText(/không được tính theo công thức/)).toHaveCount(0);
});

test("gradebook banner: công thức chưa xác nhận → Chốt điểm khoá có lý do ở chính nút; TA có Xem công thức", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context, { scheme: false, status: "DRAFT_READY" });
  const btn = page.getByRole("button", { name: "Chốt điểm" });
  await expect(btn).toHaveAttribute("aria-disabled", "true");
  await btn.focus();
  await expect(page.getByRole("tooltip")).toHaveText("Chưa có công thức điểm đã xác nhận");
  await expect(page.getByRole("link", { name: "Xác nhận công thức" })).toBeVisible();
  const p2 = await context.newPage();
  await asDemo(context, "ta");
  await gradebookApi(p2, { scheme: false, status: "DRAFT_READY" });
  await p2.goto("/gradebook");
  await expect(p2.getByRole("link", { name: "Xem công thức" }).first()).toBeVisible();
  await expect(p2.getByRole("button", { name: "Chốt điểm" })).toHaveCount(0);
});

// ---- US-P6-07: nhập / xuất XLSX ------------------------------------------------------------------------------------------------------------------
const XLSX = { name: "giua-ky.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: Buffer.from("PK\x03\x04 thử") };

async function openImport(page: Page) {
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  await page.getByRole("menuitem", { name: "Nhập từ XLSX" }).click();
  return page.getByRole("dialog", { name: "Nhập điểm từ XLSX" });
}

test("import xlsx flow: ba bước, nút chính theo bước, ghi xong bảng đổi không cần tải lại, công bố mặc định tắt", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context);
  const dlg = await openImport(page);
  await expect(dlg.getByRole("button", { name: "Tiếp tục" })).toBeDisabled();
  await dlg.locator("[data-part=import-input]").setInputFiles(XLSX);
  await expect(dlg.locator("[data-part=import-file]")).toHaveText("giua-ky.xlsx");
  await dlg.getByRole("button", { name: "Tiếp tục" }).click();
  // bước 2: gợi ý theo tên cột
  await expect(dlg.getByLabel("Cột MSSV")).toHaveValue("A");
  await expect(dlg.getByLabel("Cột chứa điểm")).toHaveValue("C");
  await dlg.getByRole("button", { name: "Xem trước" }).click();
  // bước 3
  await expect(dlg.locator("[data-part=import-summary]")).toHaveText("3 dòng hợp lệ · 1 ghi đè · 0 không đổi · 0 lỗi");
  await expect(dlg.locator("[data-part=import-sample]")).toContainText("Sinh Viên 0001");
  await expect(dlg.getByLabel("Công bố cột cho sinh viên")).not.toBeChecked();
  await expect(dlg.getByRole("button", { name: "Ghi 3 dòng" })).toBeVisible();
  expect(api.calls.filter((c) => c.path.endsWith("/grade-entries/import") && JSON.parse(c.body).mode === "commit")).toHaveLength(0); // xem trước không ghi
  await dlg.getByRole("button", { name: "Ghi 3 dòng" }).click();
  await expect(dlg).toHaveCount(0);
  await expect(cellOf(page, 2, WEEKLY)).toHaveValue("8");
  const commit = api.calls.find((c) => c.path.endsWith("/grade-entries/import") && JSON.parse(c.body).mode === "commit")!;
  expect(commit.idem).not.toBe("");
  expect(JSON.parse(commit.body)).toMatchObject({ skip_errors: false, mapping: { header_row: 1, student_code_col: "A", score_col: "C" } });
  expect(api.calls.some((c) => c.path.endsWith("/publish"))).toBe(false);
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);
});

test("import errors listed: dòng lỗi theo hàng bằng chữ tiếng Việt, nút Bỏ qua N dòng lỗi và ghi M dòng gửi skip_errors", async ({ page, context }, info) => {
  desktopOnly(info);
  const errors = [
    { row: 3, student_code: "99999999", code: "STUDENT_NOT_IN_COURSE", message: "MSSV không có trong lớp" },
    { row: 5, student_code: "20229002", code: "SCORE_OUT_OF_RANGE", message: "Điểm ngoài thang 0–10" },
    { row: 6, student_code: "20229777", code: "STUDENT_CODE_UNVERIFIED", message: "MSSV chưa có trong danh sách lớp" },
  ];
  const api = await sheet(page, context, { importOut: { rows_ok: 1, errors } });
  const dlg = await openImport(page);
  await dlg.locator("[data-part=import-input]").setInputFiles(XLSX);
  await dlg.getByRole("button", { name: "Tiếp tục" }).click();
  await dlg.getByRole("button", { name: "Xem trước" }).click();
  const list = dlg.locator("[data-part=import-errors]");
  await expect(list.locator("li")).toHaveText(["Dòng 3 · 99999999 · MSSV không có trong lớp", "Dòng 5 · 20229002 · Điểm ngoài thang 0–10", "Dòng 6 · 20229777 · MSSV chưa có trong danh sách lớp"]);
  expect(await dlg.innerText()).not.toMatch(/STUDENT_NOT_IN_COURSE|SCORE_OUT_OF_RANGE|STUDENT_CODE_UNVERIFIED/);
  await dlg.getByRole("button", { name: "Bỏ qua 3 dòng lỗi và ghi 1 dòng" }).click();
  await expect(dlg).toHaveCount(0);
  const commit = api.calls.find((c) => c.path.endsWith("/grade-entries/import") && JSON.parse(c.body).mode === "commit")!;
  expect(JSON.parse(commit.body).skip_errors).toBe(true);
});

test("import upload failed: giữ tên tệp và có Tải lại; chọn cột và hạng mục không mất", async ({ page, context }, info) => {
  desktopOnly(info);
  let down = true;
  await sheet(page, context, { h: { "POST uploads/presign": (r, next) => (down ? r.abort("connectionfailed") : next(r)) } });
  const dlg = await openImport(page);
  await dlg.getByLabel("Nhập vào").selectOption({ label: "Thi cuối kỳ" });
  await dlg.locator("[data-part=import-input]").setInputFiles(XLSX);
  await dlg.getByRole("button", { name: "Tiếp tục" }).click();
  await expect(dlg.getByRole("alert")).toBeVisible();
  await expect(dlg.locator("[data-part=import-file]")).toHaveText("giua-ky.xlsx");
  down = false;
  await dlg.getByRole("button", { name: "Tải lại" }).click();
  await expect(dlg.getByLabel("Cột MSSV")).toBeVisible();
});

test("import publish option: bật Công bố cột thì ghi xong gọi công bố đúng cột", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context);
  const dlg = await openImport(page);
  await dlg.locator("[data-part=import-input]").setInputFiles(XLSX);
  await dlg.getByRole("button", { name: "Tiếp tục" }).click();
  await dlg.getByRole("button", { name: "Xem trước" }).click();
  await dlg.getByLabel("Công bố cột cho sinh viên").check();
  await dlg.getByRole("button", { name: "Ghi 3 dòng" }).click();
  await expect(dlg).toHaveCount(0);
  expect(api.calls.some((c) => c.method === "POST" && c.path.endsWith(`/grade-items/${WEEKLY}/publish`) && JSON.parse(c.body).published === true)).toBe(true);
});

test("export xlsx: Xuất XLSX tải tệp; TA và sổ đã chốt không có Nhập", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context);
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  const dl = page.waitForEvent("download");
  await page.getByRole("menuitem", { name: "Xuất XLSX" }).click();
  expect((await dl).suggestedFilename()).toBe("so-diem.xlsx");

  const p2 = await context.newPage();
  await gradebookApi(p2, { locked: true });
  await p2.goto("/gradebook");
  await p2.getByRole("button", { name: "Thêm hành động" }).click();
  await expect(p2.getByRole("menuitem", { name: "Nhập từ XLSX" })).toHaveCount(0);
  await expect(p2.getByRole("menuitem", { name: "Xuất XLSX" })).toBeVisible();

  const p3 = await context.newPage();
  await asDemo(context, "ta");
  await gradebookApi(p3, {});
  await p3.goto("/gradebook");
  await expect(p3.locator("[data-part=grade-row]").first()).toBeVisible();
  await expect(p3.getByRole("button", { name: "Thêm hành động" })).toHaveCount(0);
});

// ---- US-P6-08: chốt / mở khoá / lịch sử chốt -----------------------------------------------------------------------------------------------------
async function fill(api: Awaited<ReturnType<typeof gradebookApi>>, n: number, final = true) {
  for (let i = 1; i <= n; i++) {
    api.entries.set(`${gbStudent(i).id}:${WEEKLY}`, { score: "8", version: 1, source: "MANUAL" });
    if (final) api.entries.set(`${gbStudent(i).id}:${FINAL}`, { score: "7", version: 1, source: "MANUAL" });
  }
}

test("finalize flow: hậu quả bằng số, Idempotency-Key, sau chốt đầu bảng Đã chốt v1, ô khoá, Mở khoá ở menu, không toast", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context, { students: 3 });
  await fill(api, 3);
  await page.reload();
  await expect(page.locator("[data-part=grade-row]").first().locator("[data-part=row-status]")).toHaveText("Đủ");
  await page.getByRole("button", { name: "Chốt điểm" }).click();
  const dlg = page.getByRole("dialog", { name: "Chốt điểm" });
  await expect(dlg).toContainText("Chốt 3 sinh viên · phân bố: C 3 · 0 sinh viên liệt");
  await dlg.getByRole("button", { name: "Chốt điểm" }).click();
  await expect(dlg).toHaveCount(0);
  await expect(page.getByText(/^Đã chốt v1 · \d{1,2}[:/]\d{2}/)).toBeVisible();
  const call = api.calls.find((c) => c.method === "POST" && c.path.endsWith("/gradebook/finalize"))!;
  expect(call.idem).not.toBe("");
  await expect(page.locator("input[data-cell]")).toHaveCount(0);
  await expect(page.locator("[data-part=row-status]").first()).toHaveText("Đã chốt v1");
  await expect(page.getByRole("button", { name: "Chốt điểm" })).toHaveCount(0);
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  await expect(page.getByRole("menuitem", { name: "Mở khoá" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByText("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.")).toHaveCount(1);
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);
});

test("finalize incomplete list: 409 hiện danh sách sinh viên thiếu kèm liên kết tới ô, chữ tiếng Việt", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context, { students: 3 });
  await fill(api, 3, false);
  await page.reload();
  await expect(page.locator("[data-part=grade-row]").first()).toBeVisible();
  await page.getByRole("button", { name: "Chốt điểm" }).click();
  await page.getByRole("dialog", { name: "Chốt điểm" }).getByRole("button", { name: "Chốt điểm" }).click();
  const list = page.locator("[data-part=incomplete-list]");
  await expect(page.getByText("Còn 3 sinh viên chưa đủ điểm")).toBeVisible();
  await expect(list.locator("li")).toHaveCount(3);
  await expect(list.locator("li").first()).toHaveText("Sinh Viên 0001 — thiếu Thi cuối kỳ");
  expect(await page.locator("main").innerText()).not.toMatch(/GRADES_INCOMPLETE/);
  await list.getByRole("button", { name: /Sinh Viên 0002/ }).click();
  await expect(cellOf(page, 2, FINAL)).toBeFocused();
  await expect(page.getByText("Còn 3 sinh viên chưa đủ điểm")).toBeVisible();
});

test("unlock reason: hậu quả bằng số, lý do 10–500 ký tự khoá nút, mở khoá xong ô nhập lại được", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context, { locked: true });
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  await page.getByRole("menuitem", { name: "Mở khoá" }).click();
  const dlg = page.getByRole("dialog", { name: "Mở khoá" });
  await expect(dlg).toContainText("Mở khoá bản chốt v2 của 30 sinh viên. Bản v2 vẫn được giữ nguyên.");
  const ok = dlg.getByRole("button", { name: "Mở khoá" });
  await expect(ok).toBeDisabled();
  await dlg.getByLabel("Lý do").fill("123456789");
  await expect(ok).toBeDisabled();
  await dlg.getByLabel("Lý do").fill("Phúc khảo sv04: sửa điểm giữa kỳ");
  await expect(ok).toBeEnabled();
  await ok.click();
  await expect(dlg).toHaveCount(0);
  await expect(cellOf(page, 1, WEEKLY)).toBeVisible();
  const call = api.calls.find((c) => c.path.endsWith("/gradebook/unlock"))!;
  expect(call.idem).not.toBe("");
  expect(JSON.parse(call.body)).toEqual({ reason: "Phúc khảo sv04: sửa điểm giữa kỳ" });
});

test("snapshot history: Lịch sử chốt liệt kê các phiên bản kèm lý do mở khoá, chọn bản cũ xem điểm đã chốt; TA cũng xem được", async ({ page, context }, info) => {
  desktopOnly(info);
  await sheet(page, context, { locked: true, drift: true });
  await expect(page.locator("[data-part=row-drift]")).toHaveText("Khác bản chốt");
  await expect(page.locator("[data-part=row-drift]")).toHaveCount(1);
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  await page.getByRole("menuitem", { name: "Lịch sử chốt" }).click();
  const hist = page.locator("[data-part=snapshot-history]");
  await expect(hist.getByRole("button", { name: /^Bản chốt v2/ })).toBeVisible();
  await expect(hist).toContainText("Phúc khảo sv04");
  await hist.getByRole("button", { name: /^Bản chốt v1/ }).click();
  await expect(hist.getByRole("table", { name: "Điểm đã chốt v1" })).toContainText("Sinh Viên 0001");
  await page.keyboard.press("Escape");

  const p2 = await context.newPage();
  await asDemo(context, "ta");
  await gradebookApi(p2, { locked: true });
  await p2.goto("/gradebook");
  await p2.getByRole("button", { name: "Thêm hành động" }).click();
  await expect(p2.getByRole("menuitem", { name: "Lịch sử chốt" })).toBeVisible();
  await expect(p2.getByRole("menuitem", { name: "Mở khoá" })).toHaveCount(0);
  await expect(p2.getByRole("menuitem", { name: "Xuất XLSX" })).toHaveCount(0);
});

// ---- US-S8-SEED: F10 đi trọn (AC4, AC7) ----------------------------------------------------------------------------------------------------------
test("lop2 demo step 6: tải quy chế lớp 2 (bản dựng thật) → một mục chưa rõ chặn xác nhận → điền làm tròn → xác nhận → banner mất, Chốt điểm mở", async ({ page, context }, info) => {
  desktopOnly(info);
  const pdf = { name: "quy-che-lop2.pdf", mimeType: "application/pdf", buffer: readFileSync(path.resolve(__dirname, "../../seed/demo/quy-che-lop2.pdf")) };
  expect(pdf.buffer.subarray(0, 5).toString()).toBe("%PDF-");
  const api = await open(page, context, { draft: null, confirmed: null });
  await page.route("**/api/v1/courses/*/uploads/presign", (r) => r.fulfill(okJson({ upload_id: "u1", method: "PUT", url: `${BASE_URL}/__blob/u1`, headers: {} })));
  await page.route("**/__blob/**", (r) => r.fulfill({ status: 200, body: "" }));
  await page.route("**/api/v1/courses/*/uploads/complete", (r) => r.fulfill(okJson({ job_id: "j1", document: { id: "00000000-0000-7000-8000-0000000007d1" } }, 202)));
  await page.route("**/api/v1/jobs/*", (r) => r.fulfill(okJson({ id: "j", status: "SUCCEEDED", progress: 100 })));
  await page.route("**/api/v1/courses/*/grade-scheme/extract", (r) => {
    api.set({ draft: draftLop2() });
    return r.fulfill(okJson({ job_id: "j2" }, 202));
  });
  await page.setInputFiles("[data-part=policy-input]", pdf);
  await expect(page.locator("[data-block]")).toHaveCount(6, { timeout: 15_000 });
  await expect(block(page, "ROUNDING")).toContainText("Quy chế chưa nêu cách làm tròn");
  await expect(confirmBtn(page)).toHaveAttribute("aria-disabled", "true");
  await block(page, "ROUNDING").getByRole("combobox").selectOption("0.1");
  await expect(confirmBtn(page)).not.toHaveAttribute("aria-disabled", "true");
  await confirmBtn(page).click();
  const dlg = page.getByRole("dialog");
  await expect(dlg).toContainText("Áp cho");
  await dlg.getByRole("button", { name: "Xác nhận công thức" }).click();
  await expect(page.locator("[data-part=confirmed-line]")).toContainText("Đã xác nhận ·");
  expect(api.calls.some((c) => c.method === "POST" && c.path.endsWith("/confirm"))).toBe(true);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await gradebookApi(page, { students: 3, scheme: true });
  await page.goto("/gradebook");
  await expect(page.locator("[data-part=grade-row]").first()).toBeVisible();
  await expect(page.getByText(/Chưa có công thức điểm đã xác nhận/)).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Chốt điểm" })).not.toHaveAttribute("aria-disabled", "true");
});

test("f10 main path: sửa ô → giải trình → chốt v1 → ô khoá → mở khoá có lý do → sửa → chốt lại v2; TA chỉ đọc", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await sheet(page, context, { students: 3 });
  await fill(api, 3);
  await page.reload();
  const c = cellOf(page, 1, FINAL);
  await c.fill("9");
  await c.press("Enter");
  await expect.poll(() => api.puts().length).toBe(1);
  await page.getByRole("button", { name: /^Hành động — Sinh Viên 0002/ }).click();
  await page.getByRole("menuitem", { name: "Giải trình" }).click();
  await expect(page.locator("[data-part=grade-calculation]")).toBeVisible();
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Chốt điểm" }).click();
  await page.getByRole("dialog", { name: "Chốt điểm" }).getByRole("button", { name: "Chốt điểm" }).click();
  await expect(page.getByText(/^Đã chốt v1 · /)).toBeVisible();
  await expect(page.locator("input[data-cell]")).toHaveCount(0);
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  await page.getByRole("menuitem", { name: "Mở khoá" }).click();
  const un = page.getByRole("dialog", { name: "Mở khoá" });
  await un.getByLabel("Lý do").fill("Phúc khảo sv01: sửa điểm cuối kỳ");
  await un.getByRole("button", { name: "Mở khoá" }).click();
  await expect(cellOf(page, 1, FINAL)).toBeVisible();
  const d = cellOf(page, 1, FINAL);
  await d.fill("8");
  await d.press("Enter");
  await expect.poll(() => api.puts().length).toBe(2);
  await page.getByRole("button", { name: "Chốt điểm" }).click();
  await page.getByRole("dialog", { name: "Chốt điểm" }).getByRole("button", { name: "Chốt điểm" }).click();
  await expect(page.getByText(/^Đã chốt v2 · /)).toBeVisible();
  expect(api.calls.filter((x) => x.path.endsWith("/gradebook/finalize"))).toHaveLength(2);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await asDemo(context, "ta");
  await gradebookApi(page, { students: 3, locked: true });
  await page.goto("/gradebook");
  await expect(page.locator("[data-part=grade-row]").first()).toBeVisible();
  await expect(page.locator("input[data-cell]")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Chốt điểm" })).toHaveCount(0);
});
