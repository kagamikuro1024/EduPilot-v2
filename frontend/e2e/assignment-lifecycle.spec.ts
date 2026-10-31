import { expect, test, type Page, type Route } from "@playwright/test";
import { A1, STUDENT_B_ID, SUB1, UPLOAD_ID, assignment, assignmentsApi, mineOut, studentApi, studentItem, unmatchedApi } from "./support/assignment-fixtures";
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

// ---- US-P7-03: trang nộp bài của sinh viên ---------------------------------------------------------------------------------
const STUDENT_URL = `/assignments/${A1}`;
const PDF = { name: "bt03.pdf", mimeType: "application/pdf", buffer: Buffer.from("%PDF-1.7\nbai lam cua sinh vien\n") };
const picker = (page: Page) => page.getByLabel("Chọn tệp bài nộp");
const statusLine = (page: Page) => page.locator("[data-part=submit-status]");

async function openStudent(page: Page, context: Parameters<typeof asDemo>[0], opts: Parameters<typeof studentApi>[1] = {}) {
  await asDemo(context, "student");
  const api = await studentApi(page, opts);
  await page.goto(STUDENT_URL);
  await expect(page.getByRole("heading", { name: "Bài tập 03 · Phân tích nhật ký", level: 1 })).toBeVisible();
  return api;
}

test("student page states: bốn trạng thái, đề bài Markdown, tiêu chí, không mục sidebar", async ({ page, context }, info) => {
  desktopOnly(info);
  await openStudent(page, context);
  await expect(page.locator("[data-part=due-left]")).toHaveText(/^(còn|đã quá hạn) /);
  await expect(page.locator("[data-part=markdown] li").first()).toHaveText("Nêu ba dấu hiệu");
  await expect(page.locator("main dt").first()).toHaveText("Phát hiện dấu hiệu");
  await expect(page.locator("main dd").first()).toHaveText("6 điểm");
  await expect(page.locator("[data-part=dropzone]")).toBeVisible();
  await expect(statusLine(page)).toHaveText("Chưa nộp");
  await expect(page.getByRole("button", { name: "Nộp bài" })).toBeVisible();
  await expect(page.locator("main [data-ep-panel]")).toHaveCount(1);
  await expect(page.locator("nav a[href^='/assignments']")).toHaveCount(0); // không có mục sidebar
  await expect(page.locator("main [data-tone=strong]")).toHaveCount(0);

  for (const [state, text] of [
    ["SUBMITTED", "Đã nộp Thứ Sáu, 23/10 · 08:10 · Nộp muộn 1 ngày"],
    ["GRADING", "Đang chấm"],
    ["GRADED", "Đã có điểm"],
  ] as const) {
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await studentApi(page, { mine: mineOut({ state, submission: { version: 1, file_name: "bt03.pdf", received_at: "2026-10-23T01:10:00.000Z", late_days: 1 } }) });
    await page.reload();
    await expect(statusLine(page)).toHaveText(text);
    await expect(page.getByRole("button", { name: "Nộp lại" })).toBeVisible();
  }
  const body = await page.locator("main").innerText();
  expect(body).not.toMatch(/điểm nháp|AI|tiến độ chấm/i);
});

test("dropzone rejects before upload: sai đuôi / quá 20 MB / rỗng báo tại chỗ, không có presign", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await openStudent(page, context);
  const seen: string[] = [];
  page.on("request", (r) => seen.push(r.url()));
  const err = page.locator("[data-part=file-error]");
  await picker(page).setInputFiles({ name: "bai.exe", mimeType: "application/octet-stream", buffer: Buffer.from("MZ") });
  await expect(err).toHaveText("Chỉ nhận PDF, DOCX, TXT, MD");
  await picker(page).setInputFiles({ name: "to.pdf", mimeType: "application/pdf", buffer: Buffer.alloc(20 * 1024 * 1024 + 1) });
  await expect(err).toHaveText("Tệp lớn hơn 20 MB");
  await picker(page).setInputFiles({ name: "rong.txt", mimeType: "text/plain", buffer: Buffer.alloc(0) });
  await expect(err).toHaveText("Tệp rỗng");
  await expect(page.getByRole("button", { name: "Nộp bài" })).toBeDisabled();
  expect(seen.filter((u) => u.includes("presign"))).toHaveLength(0);
  expect(api.presigns()).toHaveLength(0);
});

test("submit confirms with server timestamp: presign → PUT kho tệp → submit, xác nhận bằng giờ máy chủ", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await openStudent(page, context);
  await picker(page).setInputFiles(PDF);
  await expect(page.locator("[data-part=picked]")).toHaveText("bt03.pdf");
  await page.getByRole("button", { name: "Nộp bài" }).click();
  await expect(statusLine(page)).toHaveText("Đã nhận bài lúc 08:10 Thứ Sáu, 23/10");
  const sent = JSON.parse(api.presigns()[0].body) as { filename: string; size_bytes: number; sha256: string };
  expect(sent.filename).toBe("bt03.pdf");
  expect(sent.size_bytes).toBe(PDF.buffer.length);
  expect(sent.sha256).toMatch(/^[0-9a-f]{64}$/);
  expect(api.blobPuts).toHaveLength(1);
  expect(api.blobPuts[0].headers["x-amz-checksum-sha256"]).toBeTruthy();
  expect(api.blobPuts[0].headers["content-type"]).toBe("application/pdf");
  const submits = api.submits();
  expect(submits).toHaveLength(1);
  expect(JSON.parse(submits[0].body)).toEqual({ upload_id: UPLOAD_ID });
  expect(submits[0].headers["idempotency-key"]).toBeTruthy();
});

test("submit offline retry: kho tệp hỏng giữ nguyên tệp đã chọn, Thử lại xin presign mới", async ({ page, context }, info) => {
  desktopOnly(info);
  let down = true;
  const api = await openStudent(page, context, { storage: (r) => (down ? r.abort("connectionfailed") : r.fulfill({ status: 200, body: "" })) });
  await picker(page).setInputFiles(PDF);
  await page.getByRole("button", { name: "Nộp bài" }).click();
  await expect(page.getByText("Chưa gửi được")).toBeVisible({ timeout: 15_000 });
  await expect(page.locator("[data-part=picked]")).toHaveText("bt03.pdf");
  expect(api.submits()).toHaveLength(0);
  down = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(statusLine(page)).toHaveText("Đã nhận bài lúc 08:10 Thứ Sáu, 23/10");
  expect(api.presigns().length).toBeGreaterThanOrEqual(2);
  expect(api.submits()).toHaveLength(1);
});

test("submit double click: bấm đúp Nộp bài chỉ tạo một bản nộp", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await openStudent(page, context, {
    h: {
      "POST assignments/:id/submit": async (r, dflt) => {
        await new Promise((res) => setTimeout(res, 800));
        await dflt(r);
      },
    },
  });
  await picker(page).setInputFiles(PDF);
  await page.getByRole("button", { name: "Nộp bài" }).dblclick();
  await expect(statusLine(page)).toHaveText("Đã nhận bài lúc 08:10 Thứ Sáu, 23/10", { timeout: 15_000 });
  await page.waitForTimeout(800);
  expect(api.submits()).toHaveLength(1);
  expect(api.presigns()).toHaveLength(1);
});

test("submit resume complete: vé còn hạn hiện Chưa hoàn tất nộp, Hoàn tất dùng lại upload_id", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await openStudent(page, context);
  await page.evaluate(
    ([key, id]) => localStorage.setItem(key, JSON.stringify({ upload_id: id, filename: "bt03.pdf", size: 30, ts: Date.now() })),
    [`ep:submit:00000000-0000-7000-8000-00000000c001:${A1}`, UPLOAD_ID],
  );
  await page.reload();
  await expect(page.getByText("Chưa hoàn tất nộp · bt03.pdf")).toBeVisible();
  await page.getByRole("button", { name: "Hoàn tất" }).click();
  await expect(statusLine(page)).toHaveText("Đã nhận bài lúc 08:10 Thứ Sáu, 23/10");
  expect(api.presigns()).toHaveLength(0);
  expect(JSON.parse(api.submits()[0].body)).toEqual({ upload_id: UPLOAD_ID });
  expect(await page.evaluate((k) => localStorage.getItem(k), `ep:submit:00000000-0000-7000-8000-00000000c001:${A1}`)).toBeNull();
});

test("assignment page 375: một cột, vùng nộp ≥ 48 px, nút đủ rộng, không cuộn ngang", async ({ page, context }, info) => {
  desktopOnly(info);
  await page.setViewportSize({ width: 375, height: 812 });
  await openStudent(page, context, { mine: mineOut(), items: [studentItem()] });
  const over = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(over).toBeLessThanOrEqual(0);
  const drop = await page.locator("[data-part=dropzone]").boundingBox();
  expect(drop!.height).toBeGreaterThanOrEqual(48);
  const btn = await page.getByRole("button", { name: "Nộp bài" }).boundingBox();
  expect(btn!.height).toBeGreaterThanOrEqual(44);
  expect(btn!.width).toBeGreaterThanOrEqual(160);
  await expect(statusLine(page)).toHaveAttribute("aria-live", "polite");
});

// ---- US-P7-04: tab `Chưa khớp` ---------------------------------------------------------------------------------------------
const UNMATCHED_URL = "/grading?tab=unmatched";
const uRows = (page: Page) => page.locator("[data-part=unmatched-row]");

async function openUnmatched(page: Page, context: Parameters<typeof asDemo>[0], opts: Parameters<typeof unmatchedApi>[1] = {}) {
  await asDemo(context, "teacher");
  const api = await unmatchedApi(page, opts);
  await page.goto(UNMATCHED_URL);
  await expect(page.getByRole("heading", { name: "Chấm bài", level: 1 })).toBeVisible();
  return api;
}

test("unmatched assign: chọn bài + sinh viên gợi ý rồi Gán, 409 và 403 báo bằng chữ", async ({ page, context }, info) => {
  desktopOnly(info);
  let slow = false;
  let conflict = false;
  let forbidden = false;
  const api = await openUnmatched(page, context, {
    h: {
      "PUT submissions/:id/match": async (r, dflt) => {
        if (forbidden) return r.fulfill({ status: 403, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true" }, body: JSON.stringify({ code: "FORBIDDEN", message: "x", trace_id: "t" }) });
        if (conflict) return r.fulfill({ status: 409, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true" }, body: JSON.stringify({ code: "VERSION_CONFLICT", message: "x", trace_id: "t" }) });
        if (slow) await new Promise((res) => setTimeout(res, 800));
        await dflt(r);
      },
    },
  });
  await expect(page.locator("main table th")).toHaveText(["Tệp", "Nguồn", "Nhận lúc", "Lý do", "Gợi ý"]);
  await expect(uRows(page)).toHaveCount(2);
  await expect(uRows(page).nth(0)).toContainText("bt03-uyen.pdf");
  await expect(uRows(page).nth(0)).toContainText("Thư");
  await expect(uRows(page).nth(0)).toContainText("Không có MSSV trong tên tệp");
  await expect(uRows(page).nth(0)).toContainText("Trần Thu Uyên");
  expect(await page.locator("main").innerText()).not.toMatch(/phiên bản|Bài nộp chưa khớp của lớp này/);

  // 403 (TA sau khi đã công bố điểm): lý do bằng chữ, dòng vẫn còn
  forbidden = true;
  await uRows(page).nth(0).click();
  const box = page.locator("[data-part=assign-box]");
  await expect(box.getByLabel("Bài tập")).toHaveValue(A1);
  await expect(box.getByLabel("Sinh viên", { exact: true })).toHaveValue(STUDENT_B_ID);
  await box.getByRole("button", { name: "Gán" }).click();
  await expect(page.getByText("Chỉ giảng viên sửa sau khi đã công bố điểm")).toBeVisible();
  await expect(uRows(page)).toHaveCount(2);

  // 409: dòng báo + tải lại danh sách
  forbidden = false;
  conflict = true;
  await box.getByRole("button", { name: "Gán" }).click();
  await expect(page.getByText("Bài này vừa được xử lý ở nơi khác")).toBeVisible();

  // thành công: bấm đúp chỉ một PUT, dòng biến mất ngay
  conflict = false;
  slow = true;
  await expect(box).toBeVisible(); // khu gán vẫn mở sau lỗi: lựa chọn không mất
  await box.getByRole("button", { name: "Gán" }).dblclick();
  await expect(uRows(page)).toHaveCount(1, { timeout: 15_000 });
  await page.waitForTimeout(800);
  const ok = api.matches().filter((c) => !c.body.includes("\"version\":0"));
  expect(ok.filter((c) => JSON.parse(c.body).version === 1_800_000_000_000_001).length).toBeGreaterThanOrEqual(1);
  const last = JSON.parse(api.matches()[api.matches().length - 1].body) as { assignment_id: string; student_id: string; version: number };
  expect(last).toEqual({ assignment_id: A1, student_id: STUDENT_B_ID, version: 1_800_000_000_000_001 });
  expect(api.matches().filter((c) => c.path.includes(SUB1)).length).toBe(3); // 403 + 409 + một lần thành công (bấm đúp = một yêu cầu)
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);
});

test("unmatched dismiss: lý do rỗng không gửi, bỏ qua xong có Hoàn tác trong 5 giây", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await openUnmatched(page, context);
  await uRows(page).nth(1).click();
  const box = page.locator("[data-part=assign-box]");
  await box.getByRole("button", { name: "Bỏ qua" }).click();
  const confirm = box.getByRole("button", { name: "Xác nhận bỏ qua" });
  await expect(confirm).toBeDisabled();
  expect(api.dismisses()).toHaveLength(0);
  await box.getByLabel("Lý do bỏ qua").fill("Nộp nhầm lớp");
  await confirm.click();
  await expect(uRows(page)).toHaveCount(1);
  const sent = JSON.parse(api.dismisses()[0].body) as { reason: string; version: number };
  expect(sent).toEqual({ reason: "Nộp nhầm lớp", version: 1_800_000_000_000_002 });
  const line = page.locator("[data-part=undo-line], p", { hasText: "Đã bỏ qua" }).first();
  await expect(line).toBeVisible();
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect(uRows(page)).toHaveCount(2);
  expect(api.undismisses()).toHaveLength(1);
});

test("unmatched states: khung xương, rỗng, lỗi kèm Thử lại", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  await page.route(/\/api\/v1\/courses\/[^/]+\/submissions(\?|$)/, async (r) => {
    await new Promise((res) => setTimeout(res, 2000));
    return r.fulfill(json({ items: [], next_cursor: null }));
  });
  await page.goto(UNMATCHED_URL);
  await expect(page.getByLabel("Đang tải")).toBeVisible();
  await page.unrouteAll({ behavior: "ignoreErrors" });

  await unmatchedApi(page, { rows: [] });
  await page.reload();
  await expect(page.getByText("Không có bài chưa khớp.")).toBeVisible();
  await page.unrouteAll({ behavior: "ignoreErrors" });

  const api = await unmatchedApi(page, { h: { "GET submissions": (r) => r.fulfill(json({ code: "INTERNAL", message: "x", trace_id: "t" }, 500)) } });
  await page.reload();
  const notice = page.locator("main [role=alert]");
  await expect(notice).toContainText("Có lỗi xảy ra");
  const before = api.calls.length;
  await notice.getByRole("button", { name: "Thử lại" }).click();
  await expect.poll(() => api.calls.length).toBeGreaterThan(before);
});
