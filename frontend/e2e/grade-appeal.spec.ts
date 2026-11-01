import { expect, test, type Page } from "@playwright/test";
import { A1, RECEIVED, SUB_B, appealFx, gradingApi, mineOut, mineResult, resultView, studentApi } from "./support/assignment-fixtures";
import { BASE_URL } from "./support/env";
import { asDemo } from "./support/session";
import { FINAL, WEEKLY, gradebookApi, student as gbStudent } from "./support/gradebook-fixtures";

// US-P7-10: phúc khảo điểm bài tập. Gateway giả theo hợp đồng thật (support/assignment-fixtures.ts).
const desktopOnly = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "đo một lần");
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const OPEN_WINDOW = "2026-12-20T07:00:00.000Z";

const published = (appeal: { state: "NONE" | "OPEN" | "UPHELD" | "ADJUSTED"; closes_at: string | null; response: string | null }) =>
  mineOut({
    state: "GRADED",
    submission: { version: 1, file_name: "bt03.pdf", received_at: RECEIVED, late_days: 1 },
    result: mineResult({ appeal }),
  });

async function openStudent(page: Page, context: Parameters<typeof asDemo>[0], opts: Parameters<typeof studentApi>[1] = {}) {
  await asDemo(context, "student");
  const api = await studentApi(page, opts);
  await page.goto(`/assignments/${A1}`);
  await expect(page.locator("[data-part=result]")).toBeVisible();
  return api;
}

test("student files appeal: chọn tiêu chí theo tên, lý do, gửi một lần", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await openStudent(page, context, { mine: published({ state: "NONE", closes_at: OPEN_WINDOW, response: null }) });
  await page.getByRole("button", { name: "Yêu cầu xem lại" }).click();
  const form = page.locator("[data-part=appeal-form]");
  await expect(form.getByRole("checkbox", { name: "Đề xuất xử lý" })).toBeVisible();
  const send = page.getByRole("button", { name: "Gửi", exact: true });
  await expect(send).toBeDisabled(); // chưa chọn tiêu chí, chưa đủ lý do
  await form.getByRole("checkbox", { name: "Đề xuất xử lý" }).check();
  await form.getByRole("textbox").fill("Em đã nêu bước khoá tài khoản ở đoạn cuối, mong thầy xem lại.");
  await expect(form).toContainText("/1000");
  await send.dblclick();
  await expect.poll(() => api.calls.filter((c) => c.path.endsWith("/appeal")).length).toBe(1);
  const sent = JSON.parse(api.calls.find((c) => c.path.endsWith("/appeal"))!.body) as { criteria_ids: string[]; reason: string };
  expect(sent.criteria_ids).toEqual(["c2"]);
  expect(sent.reason.length).toBeGreaterThan(10);
  await expect(page.locator("[data-part=appeal-sent]")).toContainText("Đã gửi yêu cầu xem lại");
  await expect(page.getByRole("button", { name: "Yêu cầu xem lại" })).toHaveCount(0);
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);
});

test("appeal window and states: hết hạn hoặc đã gửi thì không có nút, đã quyết thì hiện phản hồi", async ({ page, context }, info) => {
  desktopOnly(info);
  await openStudent(page, context, { mine: published({ state: "NONE", closes_at: "2026-10-01T07:00:00.000Z", response: null }) });
  await expect(page.getByRole("button", { name: "Yêu cầu xem lại" })).toHaveCount(0);

  await page.unrouteAll({ behavior: "ignoreErrors" });
  await studentApi(page, { mine: published({ state: "OPEN", closes_at: OPEN_WINDOW, response: null }) });
  await page.reload();
  await expect(page.locator("[data-part=appeal-sent]")).toBeVisible();
  await expect(page.getByRole("button", { name: "Yêu cầu xem lại" })).toHaveCount(0);

  await page.unrouteAll({ behavior: "ignoreErrors" });
  await studentApi(page, { mine: published({ state: "ADJUSTED", closes_at: OPEN_WINDOW, response: "Thầy đã cộng thêm 1 điểm cho tiêu chí 2." }) });
  await page.reload();
  await expect(page.locator("[data-part=appeal-decision]")).toContainText("Đã sửa");
  await expect(page.locator("[data-part=appeal-response]")).toContainText("Thầy đã cộng thêm 1 điểm");
});

test("appeal already filed: gửi lần hai bị từ chối bằng chữ, chữ đang gõ còn nguyên", async ({ page, context }, info) => {
  desktopOnly(info);
  await openStudent(page, context, {
    mine: published({ state: "NONE", closes_at: OPEN_WINDOW, response: null }),
    h: { "POST assignments/:id/appeal": (r) => r.fulfill(json({ code: "APPEAL_ALREADY_FILED", message: "x", trace_id: "t" }, 409)) },
  });
  await page.getByRole("button", { name: "Yêu cầu xem lại" }).click();
  const form = page.locator("[data-part=appeal-form]");
  await form.getByRole("checkbox", { name: "Phát hiện dấu hiệu" }).check();
  await form.getByRole("textbox").fill("Em muốn thầy xem lại tiêu chí đầu tiên của bài.");
  await page.getByRole("button", { name: "Gửi", exact: true }).click();
  await expect(page.getByText("Bạn đã gửi phúc khảo cho bài này.")).toBeVisible();
  await expect(form.getByRole("textbox")).toHaveValue("Em muốn thầy xem lại tiêu chí đầu tiên của bài.");
});

test("appeal offline draft: mất mạng không mất chữ, mở lại vẫn còn và gửi lại được", async ({ page, context }, info) => {
  desktopOnly(info);
  let offline = true;
  const api = await openStudent(page, context, {
    mine: published({ state: "NONE", closes_at: OPEN_WINDOW, response: null }),
    h: { "POST assignments/:id/appeal": (r, pass) => (offline ? r.abort("internetdisconnected") : pass(r)) },
  });
  await page.getByRole("button", { name: "Yêu cầu xem lại" }).click();
  const form = page.locator("[data-part=appeal-form]");
  await form.getByRole("checkbox", { name: "Đề xuất xử lý" }).check();
  await form.getByRole("textbox").fill("Em nghĩ phần đề xuất xử lý đã đủ ba bước như yêu cầu.");
  await page.getByRole("button", { name: "Gửi", exact: true }).click();
  await expect(page.getByRole("button", { name: "Gửi lại" })).toBeVisible();
  await page.waitForTimeout(2300);

  await page.reload();
  await expect(page.locator("[data-part=result]")).toBeVisible();
  await page.getByRole("button", { name: "Yêu cầu xem lại" }).click();
  await expect(page.locator("[data-part=appeal-form]").getByRole("textbox")).toHaveValue("Em nghĩ phần đề xuất xử lý đã đủ ba bước như yêu cầu.");

  offline = false;
  await page.locator("[data-part=appeal-form]").getByRole("checkbox", { name: "Đề xuất xử lý" }).check();
  await page.getByRole("button", { name: "Gửi", exact: true }).click();
  await expect(page.locator("[data-part=appeal-sent]")).toBeVisible();
  expect(api.calls.filter((c) => c.path.endsWith("/appeal")).length).toBeGreaterThanOrEqual(1);
});

test("teacher decides appeal: sửa điểm kèm phản hồi bắt buộc, xác nhận không quyết lại", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  const api = await gradingApi(page, { views: { [SUB_B]: resultView({ appeal: appealFx() }) } });
  await page.goto(`/grading/${SUB_B}`);
  const block = page.locator("[data-part=appeal]");
  await expect(block.locator("[data-part=appeal-who]")).toContainText("Trần Thu Uyên");
  await expect(block.locator("[data-part=appeal-criteria]")).toHaveText("Phân tích tấn công");
  await expect(block.locator("[data-part=appeal-reason]")).toContainText("leo thang đặc quyền");
  await expect(block.locator("[data-part=appeal-scores]")).toContainText("Điểm đã công bố 6,5 · AI chấm nháp 7");

  await block.getByRole("button", { name: "Sửa điểm" }).click();
  await expect(page.getByRole("button", { name: "Quyết định" })).toBeDisabled(); // chưa có phản hồi
  await block.locator('[data-part=appeal-score][data-criterion="c2"]').fill("2,25");
  await block.locator("[data-part=appeal-response-input]").fill("Thầy đồng ý phần leo thang đặc quyền, đã cộng thêm điểm.");
  await page.getByRole("button", { name: "Quyết định" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText("không quyết lại được");
  await dialog.getByRole("button", { name: "Quyết định" }).dblclick();
  await expect.poll(() => api.calls.filter((c) => c.path.endsWith("/decide")).length).toBe(1);
  const sent = JSON.parse(api.calls.find((c) => c.path.endsWith("/decide"))!.body) as { decision: string; criteria_scores: Array<{ id: string; score: string }>; version: number };
  expect(sent.decision).toBe("ADJUSTED");
  expect(sent.criteria_scores).toEqual([{ id: "c2", score: "2.25" }]);
  await expect(block.locator("[data-part=appeal-decision]")).toContainText("Đã sửa");
  await expect(block.locator("[data-part=appeal-response]")).toContainText("đã cộng thêm điểm");
  await expect(page.getByRole("button", { name: "Sửa điểm" })).toHaveCount(0);
});

test("ta cannot decide appeal: trợ giảng chỉ đọc khối phúc khảo", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "ta");
  await gradingApi(page, { views: { [SUB_B]: resultView({ appeal: appealFx() }) } });
  await page.goto(`/grading/${SUB_B}`);
  const block = page.locator("[data-part=appeal]");
  await expect(block.locator("[data-part=appeal-reason]")).toBeVisible();
  await expect(block.getByRole("button", { name: "Sửa điểm" })).toHaveCount(0);
  await expect(block.getByRole("button", { name: "Giữ nguyên điểm" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Quyết định" })).toHaveCount(0);
});

test("appeal decided twice: 409 nói bằng chữ", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  await gradingApi(page, {
    views: { [SUB_B]: resultView({ appeal: appealFx() }) },
    h: { "POST appeals/:id/decide": (r) => r.fulfill(json({ code: "APPEAL_ALREADY_DECIDED", message: "x", trace_id: "t" }, 409)) },
  });
  await page.goto(`/grading/${SUB_B}`);
  const block = page.locator("[data-part=appeal]");
  await block.getByRole("button", { name: "Giữ nguyên điểm" }).click();
  await block.locator("[data-part=appeal-response-input]").fill("Thầy giữ nguyên điểm vì bài chưa nêu đủ bước.");
  await page.getByRole("button", { name: "Quyết định" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Quyết định" }).click();
  await expect(page.getByText("Phúc khảo này đã có quyết định.")).toBeVisible();
});

// US-P7-10 AC11: phúc khảo sửa điểm khi sổ đã CHỐT — sổ đánh dấu lệch, giảng viên mở khoá rồi chốt lại, sinh viên được báo.
test("appeal locked gradebook drift: sửa điểm khi sổ đã chốt ⇒ Khác bản chốt, Mở khoá → Chốt điểm, sinh viên nhận thông báo", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  const grading = await gradingApi(page, { views: { [SUB_B]: resultView({ appeal: appealFx() }) } });
  const book = await gradebookApi(page, { students: 3, locked: true, drift: true });

  // 1) quyết định ADJUSTED trong khi sổ đang khoá
  await page.goto(`/grading/${SUB_B}`);
  const block = page.locator("[data-part=appeal]");
  await block.getByRole("button", { name: "Sửa điểm" }).click();
  await block.locator('[data-part=appeal-score][data-criterion="c2"]').fill("2,25");
  await block.locator("[data-part=appeal-response-input]").fill("Thầy đồng ý phần leo thang đặc quyền, đã cộng thêm điểm.");
  await page.getByRole("button", { name: "Quyết định" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Quyết định" }).click();
  await expect(block.locator("[data-part=appeal-decision]")).toContainText("Đã sửa");
  expect(JSON.parse(grading.calls.find((c) => c.path.endsWith("/decide"))!.body).decision).toBe("ADJUSTED");

  // 2) sổ điểm: dòng của sinh viên đó lệch bản chốt
  await page.goto("/gradebook");
  await expect(page.locator("[data-part=grade-row]").first()).toBeVisible();
  await expect(page.locator("[data-part=row-drift]")).toHaveText("Khác bản chốt");
  await expect(page.locator("[data-part=row-drift]")).toHaveCount(1);

  // 3) Mở khoá (có lý do) → Chốt điểm
  for (let i = 1; i <= 3; i++) {
    book.entries.set(`${gbStudent(i).id}:${WEEKLY}`, { score: "8", version: 1, source: "MANUAL" });
    book.entries.set(`${gbStudent(i).id}:${FINAL}`, { score: "7", version: 1, source: "MANUAL" });
  }
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  await page.getByRole("menuitem", { name: "Mở khoá" }).click();
  const unlock = page.getByRole("dialog", { name: "Mở khoá" });
  await unlock.getByLabel("Lý do").fill("Phúc khảo bài tập 03: đã cộng điểm tiêu chí 2");
  await unlock.getByRole("button", { name: "Mở khoá" }).click();
  await expect(unlock).toHaveCount(0);
  expect(JSON.parse(book.calls.find((c) => c.path.endsWith("/gradebook/unlock"))!.body).reason).toContain("Phúc khảo");

  await page.getByRole("button", { name: "Chốt điểm" }).click();
  const dlg = page.getByRole("dialog", { name: "Chốt điểm" });
  await dlg.getByRole("button", { name: "Chốt điểm" }).click();
  await expect(dlg).toHaveCount(0);
  await expect(page.getByText(/^Đã chốt v3 · \d{1,2}[:/]\d{2}/)).toBeVisible();

  // 4) sinh viên nhận thông báo điểm đổi
  const p2 = await context.newPage();
  await asDemo(context, "student");
  await p2.route("**/api/v1/notifications**", (r) =>
    r.fulfill(
      json({
        items: [{ id: "n1", type: "GRADE_CHANGED", title: "Điểm Bài tập 03 đã đổi sau phúc khảo", body: null, link: `/assignments/${A1}`, course_id: null, read_at: null, created_at: new Date().toISOString() }],
        next_cursor: null,
        unread_count: 1,
      }),
    ),
  );
  await p2.goto("/");
  await p2.getByRole("button", { name: "Thông báo, 1 chưa đọc" }).click();
  await expect(p2.locator("[data-part=notifications]")).toContainText("Điểm Bài tập 03 đã đổi sau phúc khảo");
  await p2.close();
});
