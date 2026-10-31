import { expect, test, type Page, type Route } from "@playwright/test";
import { assignment, assignmentsApi } from "./support/assignment-fixtures";
import { BASE_URL } from "./support/env";
import { asDemo, type DemoRole } from "./support/session";

// US-P7-02: tab `Bài tập` của `/grading` THẬT. Gateway giả theo hợp đồng thật (support/assignment-fixtures.ts).
const desktopOnly = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "đo một lần");
const URL = "/grading?tab=assignments";
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });

async function open(page: Page, context: Parameters<typeof asDemo>[0], opts: Parameters<typeof assignmentsApi>[1] = {}, role: DemoRole = "teacher") {
  await asDemo(context, role);
  const api = await assignmentsApi(page, opts);
  await page.goto(URL);
  await expect(page.getByRole("heading", { name: "Chấm bài", level: 1 })).toBeVisible();
  return api;
}

const rows = (page: Page) => page.locator("[data-part=assignment-row]");
const drawer = (page: Page) => page.getByRole("dialog");

test("assignments tab layout: Tabs, một Panel + bảng năm cột, Tạo bài tập chỉ của giảng viên, TA chỉ xem", async ({ page, context, browser }, info) => {
  desktopOnly(info);
  await open(page, context);
  await expect(page.getByRole("tablist", { name: "Phần của màn chấm bài" })).toBeVisible();
  await expect(page.locator("main [data-ep-panel]")).toHaveCount(1);
  await expect(page.locator("main table th")).toHaveText(["Tiêu đề", "Hạn", "Trạng thái", "Đã nộp / Tổng", "Đã chấm / Đã công bố"]);
  await expect(rows(page)).toHaveCount(2);
  await expect(rows(page).nth(0)).toContainText("Bài tập 03 · Phân tích nhật ký");
  await expect(rows(page).nth(0)).toContainText("Đã công bố");
  await expect(rows(page).nth(0)).toContainText("24 / 30");
  await expect(rows(page).nth(0)).toContainText("20 / 12");
  await expect(rows(page).nth(1)).toContainText("Nháp");
  await expect(page.getByRole("button", { name: "Tạo bài tập" })).toBeVisible();
  // không thẻ KPI, không câu mô tả dưới tiêu đề
  await expect(page.locator("main [data-tone=strong]")).toHaveCount(0);
  const text = await page.locator("main").innerText();
  expect(text).not.toMatch(/AI chấm nháp|Khi sinh viên nộp bài|Lưu nháp để soạn tiếp/);
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);

  await rows(page).nth(0).click();
  await expect(drawer(page).getByLabel("Tiêu đề")).toHaveValue("Bài tập 03 · Phân tích nhật ký");
  await expect(drawer(page).locator("[data-part=grade-link]")).toContainText("Tính vào điểm: Kiểm tra hằng tuần · weekly");
  await expect(drawer(page).getByRole("link", { name: "Mở công thức" })).toHaveAttribute("href", "/gradebook/scheme");
  // đã công bố: điểm tối đa và mã nộp thư khoá
  await expect(drawer(page).getByLabel("Điểm tối đa")).toBeDisabled();
  await expect(drawer(page).getByLabel("Mã nộp thư")).toBeDisabled();
  await expect(drawer(page).locator("[data-part=rubric-total]")).toContainText("10 / 10");

  // TA: cùng màn, chỉ xem — không có `Tạo bài tập`, không có `Lưu`, vẫn gợi ý được tiêu chí
  const taCtx = await browser.newContext();
  const taPage = await taCtx.newPage();
  await asDemo(taCtx, "ta");
  const taApi = await assignmentsApi(taPage);
  await taPage.goto(URL);
  await expect(taPage.locator("main table th")).toHaveCount(5);
  await expect(taPage.getByRole("button", { name: "Tạo bài tập" })).toHaveCount(0);
  await rows(taPage).nth(0).click();
  await expect(drawer(taPage).getByLabel("Tiêu đề")).toBeDisabled();
  await expect(drawer(taPage).getByRole("button", { name: "Lưu" })).toHaveCount(0);
  await expect(drawer(taPage).getByRole("button", { name: "Công bố bài" })).toHaveCount(0);
  await expect(drawer(taPage).getByRole("button", { name: "Gợi ý tiêu chí" })).toBeEnabled();
  await drawer(taPage).getByRole("button", { name: "Gợi ý tiêu chí" }).click();
  await expect(drawer(taPage).getByLabel("Tên tiêu chí 2")).toHaveValue("Phân tích nguyên nhân");
  await expect(drawer(taPage).locator("[data-part=rubric-total]")).toContainText("10 / 10");
  expect(taApi.puts()).toHaveLength(0); // gợi ý KHÔNG lưu
  await taCtx.close();
});

test("drawer offline draft: chữ đã gõ giữ nguyên sau khi tải lại, mất mạng thì xếp hàng và tự gửi lại", async ({ page, context }, info) => {
  desktopOnly(info);
  let down = false;
  const api = await open(page, context, {
    h: { "PUT assignments/:id": (r, dflt) => (down ? r.abort("connectionfailed") : dflt(r)) },
  });
  const TEXT = "Phân tích mẫu mã độc và nộp báo cáo kèm ảnh chụp màn hình.";
  await rows(page).nth(1).click();
  await expect(drawer(page).getByLabel("Tiêu đề")).toHaveValue("Bài tập 04 · Mã độc");
  await drawer(page).getByLabel("Mô tả").fill(TEXT);
  await page.waitForTimeout(2300); // tự lưu nháp mỗi 2 giây

  await page.reload();
  await rows(page).nth(1).click();
  await expect(drawer(page).getByText("Đã khôi phục phần bạn đang soạn dở.")).toBeVisible();
  await expect(drawer(page).getByLabel("Mô tả")).toHaveValue(TEXT);

  down = true;
  await drawer(page).getByRole("button", { name: "Lưu" }).click();
  await expect(drawer(page).getByText("Mất mạng. Sẽ gửi lại khi có mạng.")).toBeVisible({ timeout: 15_000 });
  await expect(drawer(page).getByLabel("Mô tả")).toHaveValue(TEXT);
  const tried = api.puts().length; // lần gửi hỏng cũng được ghi lại: đếm từ mốc này
  down = false;
  await expect(drawer(page).getByText("Mất mạng. Sẽ gửi lại khi có mạng.")).toHaveCount(0, { timeout: 20_000 });
  expect(api.puts().length).toBeGreaterThan(tried);
  const sent = JSON.parse(api.puts()[api.puts().length - 1].body) as { description: string; version: number };
  expect(sent.description).toBe(TEXT);
  expect(sent.version).toBe(1);
});

test("drawer conflict: 409 VERSION_CONFLICT hiện dòng báo + Giữ của tôi / Dùng của họ", async ({ page, context }, info) => {
  desktopOnly(info);
  const theirs = assignment({ title: "Bài tập 03 · bản của họ", version: 9 });
  let n = 0;
  const api = await open(page, context, {
    h: {
      "PUT assignments/:id": (r) => {
        n += 1;
        if (n === 1) return r.fulfill(json({ code: "VERSION_CONFLICT", message: "x", trace_id: "t", details: { current: theirs } }, 409));
        const body = JSON.parse(r.request().postData() ?? "{}") as Record<string, unknown>;
        return r.fulfill(json({ ...theirs, ...body, version: 10 }));
      },
    },
  });
  await rows(page).nth(0).click();
  await drawer(page).getByLabel("Tiêu đề").fill("Bài tập 03 · bản của tôi");
  await drawer(page).getByRole("button", { name: "Lưu" }).click();
  await expect(drawer(page).getByText("Bài tập vừa được người khác sửa")).toBeVisible();
  await expect(drawer(page).getByRole("button", { name: "Giữ của tôi" })).toBeVisible();
  await expect(drawer(page).getByRole("button", { name: "Dùng của họ" })).toBeVisible();
  await expect(drawer(page).getByLabel("Tiêu đề")).toHaveValue("Bài tập 03 · bản của tôi");

  await drawer(page).getByRole("button", { name: "Giữ của tôi" }).click();
  await expect(drawer(page).getByText("Bài tập vừa được người khác sửa")).toHaveCount(0);
  const puts = api.puts().filter((c) => !c.path.endsWith("/rubric"));
  expect(puts).toHaveLength(2);
  const second = JSON.parse(puts[1].body) as { version: number; title: string };
  expect(second.version).toBe(9);
  expect(second.title).toBe("Bài tập 03 · bản của tôi");
});

test("drawer double submit: bấm đúp Tạo chỉ gửi một POST", async ({ page, context }, info) => {
  desktopOnly(info);
  const slow = async (r: Route, dflt: (r2: Route) => Promise<void> | void) => {
    await new Promise((res) => setTimeout(res, 800));
    await dflt(r);
  };
  const api = await open(page, context, { h: { "POST assignments": slow, "POST assignments/:id/publish": slow } });
  await page.getByRole("button", { name: "Tạo bài tập" }).click();
  const d = drawer(page);
  await d.getByLabel("Tiêu đề").fill("Bài tập 05 · Tường lửa");
  await d.getByLabel("Mã nộp thư").fill("BT05");
  await d.getByLabel("Hạn nộp").fill("2026-12-20T14:00");
  await d.getByLabel("Tên tiêu chí 1").fill("Báo cáo đủ ý");
  await d.getByLabel("Điểm tiêu chí 1").fill("10");
  await d.getByRole("button", { name: "Tạo", exact: true }).dblclick();
  await expect.poll(() => api.posts().length, { timeout: 15_000 }).toBe(1);
  await page.waitForTimeout(1000);
  expect(api.posts()).toHaveLength(1);
  expect(api.posts()[0].headers["idempotency-key"]).toBeTruthy();
  const made = JSON.parse(api.posts()[0].body) as { submission_code: string; rubric: Array<{ max_score: string }> };
  expect(made.submission_code).toBe("BT05");
  expect(made.rubric[0].max_score).toBe("10");

  // `Công bố` cũng chỉ gửi đúng một POST; hộp xác nhận nêu hậu quả bằng số
  await page.keyboard.press("Escape");
  await rows(page).filter({ hasText: "Bài tập 04" }).click();
  await d.getByRole("button", { name: "Công bố bài" }).click();
  const box = page.getByRole("dialog").filter({ hasText: "Công bố cho 30 sinh viên" });
  await expect(box).toContainText("hạn");
  await box.getByRole("button", { name: "Công bố bài" }).dblclick();
  const publishes = () => api.posts().filter((c) => c.path.endsWith("/publish"));
  await expect.poll(() => publishes().length, { timeout: 15_000 }).toBe(1);
  await page.waitForTimeout(1000);
  expect(publishes()).toHaveLength(1);
});

test("assignments tab states: khung xương, rỗng, lỗi kèm Thử lại", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  await page.route(/\/api\/v1\/courses\/[^/]+\/assignments(\?|$)/, async (r) => {
    await new Promise((res) => setTimeout(res, 2000));
    return r.fulfill(json({ items: [], next_cursor: null }));
  });
  await page.goto(URL);
  await expect(page.getByLabel("Đang tải")).toBeVisible();
  await page.unrouteAll({ behavior: "ignoreErrors" });

  await assignmentsApi(page, { items: [] });
  await page.reload();
  await expect(page.getByText("Lớp chưa có bài tập.")).toBeVisible();
  await expect(page.locator("[data-ep-panel]").getByRole("button", { name: "Tạo bài tập" })).toBeVisible();
  await page.unrouteAll({ behavior: "ignoreErrors" });

  const api = await assignmentsApi(page, { h: { "GET assignments": (r) => r.fulfill(json({ code: "INTERNAL", message: "x", trace_id: "t" }, 500)) } });
  await page.reload();
  const notice = page.locator("main [role=alert]");
  await expect(notice).toContainText("Có lỗi xảy ra");
  const before = api.calls.length;
  await notice.getByRole("button", { name: "Thử lại" }).click();
  await expect.poll(() => api.calls.length).toBeGreaterThan(before);
  await expect(page.locator("[data-part=assignment-row]")).toHaveCount(0);
  // bài tập đã công bố dẫn từ lịch sang đúng tab này
  expect(page.url()).toContain("tab=assignments");
});
