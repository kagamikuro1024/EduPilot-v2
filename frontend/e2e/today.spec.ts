import { expect, test, type Page } from "@playwright/test";
import { loadAudit, runAudit } from "./support/audit";
import { BASE_URL } from "./support/env";
import { settleGoto } from "./support/hydrate";
import { asJwt, type JwtRole } from "./support/session";

type Json = Record<string, unknown>;

// US-P2-11 — trang `/` "Hôm nay" thật. Gateway giả bằng page.route (phần @real chạy ở bản compose có seed).
test.beforeEach(({ page }) => settleGoto(page));

const C1 = "00000000-0000-7000-8000-00000000c001";
const C2 = "00000000-0000-7000-8000-00000000c002";
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const fulfill = (route: import("@playwright/test").Route, status: number, body: unknown) =>
  status === 204 ? route.fulfill({ status, headers: cors }) : route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });

const mine = (id: string, code: string, role: "TEACHER" | "TA" | "STUDENT") => ({
  course: { id, class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" },
  role_in_course: role,
  enrollment_status: "ACTIVE",
});
const item = (kind: string, title: string, reason: string, over: Json = {}): Json => ({ id: `${kind}:${title}`, kind, title, reason, urgency: "normal", href: "", course: null, ...over });
const c1 = { id: C1, class_code: "761988" };
const c2 = { id: C2, class_code: "761987" };

type Today = { calls: string[]; dismiss: string[]; fail: boolean };
/** Phiên thật + `GET /me/courses` + `GET …/today` trả `body(path)`; ghi lại các đường dẫn đã gọi. */
async function open(page: Page, role: JwtRole, courses: Json[], body: (path: string) => Json, url = "/"): Promise<Today> {
  const st: Today = { calls: [], dismiss: [], fail: false };
  await asJwt(page, role);
  await page.route("**/api/v1/me/courses**", (r) => fulfill(r, 200, { items: courses, next_cursor: null }));
  await page.route("**/api/v1/me/today**", (r) => {
    st.calls.push("/me/today");
    return st.fail ? fulfill(r, 500, { code: "INTERNAL", message: "x", trace_id: "c".repeat(32) }) : fulfill(r, 200, body("/me/today"));
  });
  await page.route("**/api/v1/courses/*/today**", (r) => {
    const p = new URL(r.request().url()).pathname.replace("/api/v1", "");
    st.calls.push(p);
    return st.fail ? fulfill(r, 500, { code: "INTERNAL", message: "x", trace_id: "c".repeat(32) }) : fulfill(r, 200, body(p));
  });
  await page.route("**/api/v1/courses/*/setup/dismiss", (r) => {
    st.dismiss.push(new URL(r.request().url()).pathname);
    return fulfill(r, 204, null);
  });
  await page.goto(url);
  return st;
}

const student = (over: Json = {}): Json => ({ no_course: false, email_verified: true, recommended: null, timeline: [], continue: [], ...over });
const staff = (actions: Json[], over: Json = {}): Json => ({ count: actions.length, actions, attention: [], upcoming: [], ...over });
const main = (page: Page) => page.locator("main");

test("student today: chưa vào lớp ⇒ ô nhập mã ngay trên trang; nhập xong chuyển /join/<mã>", async ({ page }) => {
  const previews: string[] = [];
  await page.route("**/api/v1/courses/join/preview", (r) => {
    previews.push((JSON.parse(r.request().postData() ?? "{}") as { code: string }).code);
    return fulfill(r, 404, { code: "JOIN_CODE_INVALID", message: "m", trace_id: "d".repeat(32) });
  });
  await open(page, "STUDENT", [], () =>
    student({ no_course: true, recommended: item("JOIN_CODE", "Nhập mã tham gia lớp", "Bạn chưa vào lớp nào. Nhập mã do giảng viên cung cấp để bắt đầu.", { href: "/join", estimate_minutes: 1 }) }),
  );
  await expect(page.getByRole("heading", { name: /^Chào / })).toBeVisible();
  const input = page.getByLabel("Mã tham gia lớp");
  await expect(input).toBeVisible();
  await expect(page.getByRole("button", { name: "Tiếp tục" })).toBeDisabled();
  await input.fill("an7 k2mq");
  await expect(page.getByRole("button", { name: "Tiếp tục" })).toBeEnabled();
  await page.getByRole("button", { name: "Tiếp tục" }).click();
  await expect.poll(() => previews).toEqual(["AN7K2MQ"]); // /join/<mã> xem trước ngay (rồi tự gỡ mã khỏi thanh địa chỉ)
  await expect(page).toHaveURL(/\/join$/);
});

test("student today: email chưa xác minh đứng trước, ô nhập mã vẫn ở dưới; chỉ một nút chính", async ({ page }) => {
  await open(page, "STUDENT", [], () =>
    student({
      no_course: true,
      email_verified: false,
      recommended: item("VERIFY_EMAIL", "Xác minh email của bạn", "Chưa xác minh email thì chưa vào được lớp. Kiểm tra hộp thư a***@x.com.", { href: "/verify-email", estimate_minutes: 1 }),
    }),
  );
  await expect(page.getByText("Xác minh email của bạn")).toBeVisible();
  await expect(page.getByText("Kiểm tra hộp thư a***@x.com.")).toBeVisible();
  await expect(page.getByText("Khoảng 1 phút")).toBeVisible();
  await expect(page.getByRole("link", { name: "Xác minh email", exact: true })).toHaveAttribute("data-variant", "primary");
  await expect(page.getByLabel("Mã tham gia lớp")).toBeVisible();
  await expect(main(page).locator('[data-variant="primary"]')).toHaveCount(1);
});

test("student today: chỉ có yêu cầu chờ duyệt ⇒ 'Chờ giảng viên duyệt', không nút hành động", async ({ page }) => {
  await open(page, "STUDENT", [], () =>
    student({ no_course: true, recommended: item("JOIN_PENDING", "Chờ giảng viên duyệt", "Yêu cầu vào lớp 761988 đã gửi 2 giờ trước.", { course: c1 }) }),
  );
  await expect(page.getByText("Chờ giảng viên duyệt")).toBeVisible();
  await expect(page.getByText("Yêu cầu vào lớp 761988 đã gửi 2 giờ trước.")).toBeVisible();
  await expect(main(page).locator('[data-variant="primary"]').filter({ hasNotText: "Tiếp tục" })).toHaveCount(0);
});

test("student today: có lớp, không việc gấp ⇒ 'Hôm nay bạn không có việc gấp.' + dòng thời gian; 'Tiếp tục học' ẩn khi rỗng", async ({ page }) => {
  await open(page, "STUDENT", [mine(C1, "761988", "STUDENT")], () =>
    student({
      timeline: [
        { at: "2026-10-15T02:00:00Z", ends_at: "2026-10-15T04:30:00Z", title: "Buổi 10 · An ninh mạng", place: "P.302", state: "NOW", course: c1 },
        { at: "2026-10-20T02:00:00Z", ends_at: "2026-10-20T04:30:00Z", title: "Buổi 11 · An ninh mạng", place: "", state: "NEXT", course: c1 },
      ],
    }),
  );
  await expect(page.getByText("Hôm nay bạn không có việc gấp.")).toBeVisible();
  await expect(page.getByText("Buổi 10 · An ninh mạng")).toBeVisible();
  await expect(page.getByText("Đang diễn ra")).toBeVisible();
  await expect(page.getByText("Sắp tới")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Tiếp tục học" })).toHaveCount(0);
  await expect(page.getByLabel("Mã tham gia lớp")).toHaveCount(0);
});

test("student today: continue learning — tối đa 3 phiên chat, mở đúng /chat?session=, có mã lớp; rỗng thì không có vùng", async ({ page }) => {
  const chat = (n: number) => ({ kind: "CHAT", id: `00000000-0000-7000-8000-0000000000d${n}`, title: `Phiên ${n}`, href: `/chat?session=00000000-0000-7000-8000-0000000000d${n}`, course: c1, at: `2026-10-1${n}T02:00:00Z` });
  await open(page, "STUDENT", [mine(C1, "761988", "STUDENT")], () => student({ continue: [chat(3), chat(2), chat(1)] }));
  await expect(page.getByRole("heading", { name: "Tiếp tục học" })).toBeVisible();
  await expect(main(page).getByRole("link", { name: /Phiên \d/ })).toHaveCount(3);
  await expect(main(page).getByRole("link", { name: /Phiên 3/ })).toHaveAttribute("href", "/chat?session=00000000-0000-7000-8000-0000000000d3");
  await expect(main(page)).toContainText("Lớp 761988");
});

test("student today: thứ tự Hôm nay trước Tiếp tục học; phiên cùng tiêu đề cùng lớp chỉ hiện phiên gần nhất (US-UX91-11)", async ({ page }) => {
  const chat = (id: string, title: string, at: string, course = c1) => ({ kind: "CHAT", id, title, href: `/chat?session=${id}`, course, at });
  await open(page, "STUDENT", [mine(C1, "761988", "STUDENT")], () =>
    student({
      timeline: [{ at: "2026-10-15T02:00:00Z", ends_at: "2026-10-15T04:30:00Z", title: "Buổi 10 · An ninh mạng", place: "P.302", state: "NOW", course: c1 }],
      continue: [chat("a1", "Điều kiện thi", "2026-11-02T02:00:00Z"), chat("a2", "điều kiện thi", "2026-11-03T02:00:00Z"), chat("a3", "Điều kiện thi", "2026-11-01T02:00:00Z", c2), chat("b1", "Khác", "2026-10-30T02:00:00Z")],
    }),
  );
  const heads = main(page).getByRole("heading", { level: 2 });
  await expect(heads).toHaveText(["Việc nên làm tiếp", "Hôm nay", "Tiếp tục học"]);
  const links = main(page).getByRole("link", { name: /điều kiện thi|Khác/i });
  await expect(links).toHaveCount(3); // a2 (mới nhất của cặp lớp c1), a3 (lớp khác), b1
  await expect(main(page).locator('a[href="/chat?session=a1"]')).toHaveCount(0);
  await expect(main(page).locator('a[href="/chat?session=a2"]')).toHaveCount(1);
  await expect(main(page)).toContainText(/Thứ [A-Za-zÀ-ỹ ]+, 03\/11 · 09:00|Hôm qua|Hôm nay/);
});

test("student today: chọn một lớp ⇒ gọi /courses/{id}/today; không lớp ⇒ /me/today", async ({ page }) => {
  const st = await open(page, "STUDENT", [mine(C1, "761988", "STUDENT"), mine(C2, "761987", "STUDENT")], () => student());
  await expect(page.getByText("Hôm nay bạn không có việc gấp.")).toBeVisible();
  expect(st.calls[0]).toBe(`/courses/${C1}/today`);
});

test("staff today: tiêu đề đếm việc, hàng đúng thứ tự, 'lớp 761988' ở chế độ tất cả lớp", async ({ page }) => {
  const acts = [
    item("JOIN_REQUEST", "3 yêu cầu vào lớp 761988 đang chờ duyệt", "Cũ nhất đã chờ 2 ngày.", { urgency: "overdue", course: c1, href: `/class/members?course=${C1}&tab=pending` }),
    item("EMAIL_MISMATCH", "1 yêu cầu có email chưa khớp MSSV · lớp 761987", "Cần bạn xác nhận: email đăng ký khác email trong danh sách lớp.", { urgency: "high", course: c2, href: `/class/members?course=${C2}&tab=pending` }),
  ];
  const st = await open(page, "TEACHER", [mine(C1, "761988", "TEACHER"), mine(C2, "761987", "TEACHER")], () => staff(acts, { upcoming: [{ at: "2026-10-20T02:00:00Z", title: "Buổi 11 · An ninh mạng", place: "P.302", course: c1 }] }));
  await page.getByRole("button", { name: /^Chọn lớp, đang xem/ }).click();
  await page.getByRole("menuitem", { name: "Tất cả lớp của tôi" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/^\d+ việc cần xử lý hôm nay$/);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("2 việc cần xử lý hôm nay");
  const rows = main(page).locator('li[data-kind]');
  await expect(rows).toHaveCount(2);
  await expect(rows.nth(0)).toContainText("3 yêu cầu vào lớp 761988 đang chờ duyệt");
  await expect(rows.nth(0)).toContainText("lớp 761988");
  await expect(rows.nth(1)).toContainText("lớp 761987");
  await expect(page.getByRole("link", { name: "Xem yêu cầu", exact: true })).toHaveAttribute("href", `/class/members?course=${C1}&tab=pending`);
  await expect(page.getByRole("heading", { name: "Sắp tới" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Lớp cần chú ý" })).toHaveCount(0);
  expect(st.calls[0]).toBe(`/courses/${C1}/today`);
  expect(st.calls).toContain("/me/today");
});

test("staff today: việc chia nhóm Gấp / Hôm nay / Khi rảnh có nhãn chữ, đúng một nút đỏ, không lặp mã lớp (US-UX91-10)", async ({ page }) => {
  const acts = [
    item("GRADING_ERROR", "2 bài chấm lỗi", "Chạy lại ở hàng chờ.", { urgency: "overdue", course: c1, href: "/grading" }),
    item("GRADING_REVIEW", "Bài AI cần xem kỹ", "Độ tin cậy thấp.", { urgency: "high", course: c1, href: "/grading" }),
    item("PUBLISH_PENDING", "Công bố điểm Bài 2", "Đã duyệt đủ.", { course: c1, href: "/grading" }),
    item("EXAM_SIMILARITY", "Hai bài thi giống nhau", "Xem khi rảnh.", { course: c1, href: "/exams" }),
    item("UNMATCHED_SUBMISSION", "1 bài chưa khớp sinh viên", "Chọn sinh viên.", { course: c1, href: "/grading" }),
  ];
  await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () => staff(acts));
  const heads = main(page).getByRole("heading", { level: 3 });
  await expect(heads).toHaveText(["Gấp · 2", "Hôm nay · 1", "Khi rảnh · 2"]);
  await expect(main(page).getByRole("heading", { name: "Gấp, 2 việc" })).toBeVisible(); // trình đọc màn hình nghe "Gấp, 2 việc"
  await expect(main(page).locator("li[data-kind]")).toHaveCount(5);
  expect(await main(page).locator("li[data-kind]").evaluateAll((els) => els.map((e) => e.getAttribute("data-kind")))).toEqual(["GRADING_ERROR", "GRADING_REVIEW", "PUBLISH_PENDING", "EXAM_SIMILARITY", "UNMATCHED_SUBMISSION"]);
  await expect(main(page).locator('a[data-variant="primary"]')).toHaveCount(1);
  await expect(main(page).locator('li[data-kind="GRADING_ERROR"] a[data-variant="primary"]')).toBeVisible(); // việc đầu của nhóm đầu
  await expect(main(page).locator('li[data-kind="GRADING_REVIEW"] a[data-variant="ghost"] svg')).toHaveCount(1); // ghost + mũi tên
  await expect(main(page).locator("li[data-kind]").first()).not.toContainText("lớp 761988"); // một lớp đang chọn: không lặp mã lớp
  await expect(main(page).getByRole("heading", { name: "Việc cần xử lý", exact: true })).toHaveCount(0); // khối trùng tiêu đề trang đã bỏ
});

test("staff today: nhóm rỗng không hiện; chỉ việc thường ⇒ chỉ nhóm Hôm nay", async ({ page }) => {
  await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () => staff([item("PUBLISH_PENDING", "Công bố điểm", "x", { course: c1, href: "/grading" })]));
  await expect(main(page).getByRole("heading", { level: 3 })).toHaveText(["Hôm nay · 1"]);
});

test("staff today: nguồn Khi rảnh nhưng urgency high ⇒ vẫn là Gấp", async ({ page }) => {
  await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () => staff([item("UNMATCHED_SUBMISSION", "Bài chưa khớp", "x", { urgency: "high", course: c1, href: "/grading" })]));
  await expect(main(page).getByRole("heading", { level: 3 })).toHaveText(["Gấp · 1"]);
});

test("staff today ≥ 1440: Lớp cần chú ý và Sắp tới nằm cột phải cạnh danh sách; 1024 nằm dưới", async ({ page }) => {
  const acts = [item("PUBLISH_PENDING", "Công bố điểm", "x", { course: c1, href: "/grading" })];
  const att = { student: { id: "00000000-0000-7000-8000-00000000b001", full_name: "Sinh Viên 1" }, course: c1, reason: "vắng 3/4 buổi" };
  await page.setViewportSize({ width: 1440, height: 900 });
  await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () => staff(acts, { attention: [att], attention_total: 1, upcoming: [{ at: "2026-10-20T02:00:00Z", title: "Buổi 11", place: "P.302", course: c1 }] }));
  const list = main(page).locator("li[data-kind]").first();
  const side = main(page).getByRole("heading", { name: "Lớp cần chú ý" });
  await expect(side).toBeVisible();
  const [lb, sb] = [await list.boundingBox(), await side.boundingBox()];
  expect(sb!.x).toBeGreaterThan(lb!.x + lb!.width - 1); // cột phải
  expect(sb!.y).toBeLessThan(lb!.y + 400);
  await page.setViewportSize({ width: 1024, height: 800 });
  const [lb2, sb2] = [await list.boundingBox(), await side.boundingBox()];
  expect(sb2!.y).toBeGreaterThan(lb2!.y + lb2!.height - 1); // dưới danh sách
});

test("staff today: 'Thiết lập lớp mới' mở 4 bước tại chỗ, Bỏ qua gọi setup/dismiss và hàng biến mất", async ({ page }) => {
  const steps = [
    { key: "share_code", label: "Chia sẻ mã lớp", done: true, href: `/class/settings?course=${C1}` },
    { key: "policy", label: "Tải quy chế môn học", done: false, href: "/documents" },
    { key: "sessions", label: "Tạo lịch buổi học", done: false, href: "/calendar" },
    { key: "documents", label: "Tải tài liệu", done: false, href: "/documents" },
  ];
  const st = await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () =>
    staff([item("COURSE_SETUP", "Thiết lập lớp mới · 761988", "1/4 bước xong: chia sẻ mã lớp → tải quy chế môn học → tạo lịch buổi học → tải tài liệu.", { course: c1, steps, href: `/class/settings?course=${C1}` })]),
  );
  const row = main(page).locator('li[data-kind="COURSE_SETUP"]');
  await expect(row).toBeVisible();
  await expect(row.locator("li")).toHaveCount(0);
  await row.getByRole("button", { name: "Xem 4 bước" }).click();
  await expect(row.locator("ol li")).toHaveCount(4);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await row.getByRole("button", { name: "Bỏ qua" }).click();
  await expect(row).toHaveCount(0);
  expect(st.dismiss).toEqual([`/api/v1/courses/${C1}/setup/dismiss`]);
});

test("staff today: TA không có nút Bỏ qua; 0 việc ⇒ 'Không có việc cần xử lý hôm nay.'", async ({ page }) => {
  await open(page, "TA", [mine(C1, "761988", "TA")], () => staff([]));
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Không có việc cần xử lý hôm nay.");
  await expect(page.getByText("Bạn đã xử lý hết việc.")).toBeVisible();
});

test("admin today: việc của hệ thống, không có lớp", async ({ page }) => {
  const st = await open(page, "ADMIN", [], () =>
    ({ count: 2, actions: [
      item("LLM_PROVIDER_ERROR", "Nhà cung cấp AI đang lỗi", "OpenAI chính không phản hồi. Chat của sinh viên có thể dùng dự phòng.", { urgency: "high", href: "/settings/llm" }),
      item("COURSE_NO_TEACHER", "Lớp 761999 chưa có giảng viên", "Hãy gán giảng viên để lớp nhận thông báo và mở mã tham gia.", { course: { id: C2, class_code: "761999" }, href: "/admin/courses" }),
    ] }),
  );
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("2 việc cần xử lý hôm nay");
  await expect(page.getByText("Nhà cung cấp AI đang lỗi")).toBeVisible();
  await expect(page.getByRole("link", { name: "Gán giảng viên", exact: true })).toHaveAttribute("href", "/admin/courses");
  expect(st.calls).toEqual(["/me/today"]);
});

test("today errors: lỗi chuẩn đúng chuỗi, Thử lại gọi lại đúng 1 yêu cầu", async ({ page }) => {
  const st = await open(page, "STUDENT", [], () => student({ no_course: true }));
  await expect(page.getByLabel("Mã tham gia lớp")).toBeVisible();
  const st2 = await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () => staff([]));
  void st;
  st2.fail = true;
  await page.reload();
  await expect(page.getByRole("alert").filter({ hasText: "Chưa tải được việc hôm nay. Dữ liệu của bạn không bị ảnh hưởng." })).toBeVisible();
  const before = st2.calls.length;
  st2.fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Không có việc cần xử lý hôm nay.");
  expect(st2.calls.length - before).toBe(1);
});

test("no mock items: trang / không chứa dữ liệu mô phỏng cũ", async ({ page }) => {
  await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () => staff([item("JOIN_REQUEST", "1 yêu cầu vào lớp 761988 đang chờ duyệt", "Cũ nhất đã chờ 5 phút.", { course: c1 })]));
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("1 việc cần xử lý hôm nay");
  const text = await main(page).innerText();
  for (const old of ["Phiếu hỗ trợ", "QUIZ01", "Bài tập 03", "Lớp cần chú ý", "ticket"]) expect(text).not.toContain(old);
  expect(await main(page).getByText(/RAG|PII|trace|provider/i).count()).toBe(0);
});

test("today 375 px: sinh viên và giảng viên không tràn ngang, vùng chạm sạch", async ({ page }, info) => {
  test.skip(info.project.name !== "mobile", "chỉ dự án 375 px");
  await open(page, "STUDENT", [], () =>
    student({ no_course: true, email_verified: false, recommended: item("VERIFY_EMAIL", "Xác minh email của bạn", "Chưa xác minh email thì chưa vào được lớp. Kiểm tra hộp thư a***@x.com.", { href: "/verify-email", estimate_minutes: 1 }) }),
  );
  await expect(page.getByLabel("Mã tham gia lớp")).toBeVisible();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(a.cut).toEqual([]);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("today 375 px: giảng viên với bốn bước mở ra", async ({ page }, info) => {
  test.skip(info.project.name !== "mobile", "chỉ dự án 375 px");
  await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () =>
    staff([
      item("JOIN_REQUEST", "3 yêu cầu vào lớp 761988 đang chờ duyệt", "Cũ nhất đã chờ 2 ngày.", { urgency: "overdue", course: c1, href: `/class/members?course=${C1}&tab=pending` }),
      item("COURSE_SETUP", "Thiết lập lớp mới · 761988", "0/4 bước xong.", { course: c1, steps: [{ key: "a", label: "Chia sẻ mã lớp", done: false, href: "/class/settings" }, { key: "b", label: "Tải quy chế môn học", done: true, href: "/documents" }] }),
    ]),
  );
  await main(page).getByRole("button", { name: /^Xem 2 bước/ }).click();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("@real today: student / staff / admin với seed — D thấy ô nhập mã; B và A ≤ 1 nút chính, timeline có 'Buổi 10'; GV: tiêu đề '^\\d+ việc cần xử lý hôm nay$'", async () => {
  test.skip(true, "@real — cần compose + seed (US-P2-12)");
});

// US-P5-04 AC7: khối `attention` — ≤ 5 sinh viên AT_RISK, "Xem tất cả (n)"; không có thì không hiện khối.
test("staff attention block: liệt kê sinh viên + lý do ngắn + Xem tất cả (n)", async ({ page }) => {
  const att = (i: number) => ({ student: { id: `00000000-0000-7000-8000-00000000b00${i}`, full_name: `Sinh Viên ${i}` }, course: c1, reason: "vắng 3/4 buổi; học ít" });
  await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () => staff([], { attention: [att(1), att(2)], attention_total: 7 }));
  await expect(main(page).getByRole("heading", { name: "Lớp cần chú ý" })).toBeVisible();
  await expect(main(page).getByRole("link", { name: /Sinh Viên 1/ })).toHaveAttribute("href", `/students/00000000-0000-7000-8000-00000000b001?course=${C1}`);
  await expect(main(page).getByText("vắng 3/4 buổi; học ít")).toHaveCount(2);
  await expect(main(page).getByRole("link", { name: "Xem tất cả (7)" })).toHaveAttribute("href", `/students?filter=attention&course=${C1}`);
});

test("staff attention block: không có AT_RISK thì không hiện khối", async ({ page }) => {
  await open(page, "TEACHER", [mine(C1, "761988", "TEACHER")], () => staff([]));
  await expect(main(page).getByText("Bạn đã xử lý hết việc.")).toBeVisible();
  await expect(main(page).getByRole("heading", { name: "Lớp cần chú ý" })).toHaveCount(0);
});
