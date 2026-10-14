import { expect, test, type Page } from "@playwright/test";
import { asDemo } from "./support/session";
import { COURSE, err, json } from "./support/cal-fixtures";

// US-P5-04: `/students` THẬT. Gateway giả theo hợp đồng thật `GET /courses/{id}/students`; `calls` ghi query để kiểm tìm / lọc / con trỏ.
const stu = (i: number, over: Record<string, unknown> = {}) => ({
  student: { id: `00000000-0000-7000-8000-00000000d${String(i).padStart(3, "0")}`, full_name: `Sinh Viên ${String(i).padStart(3, "0")}`, student_code: `B22DCCN${100 + i}` },
  attendance: { recorded: 9, present: 6, late: 1, excused: 0, absent: 2 },
  participation: { speak_count: 2, points_total: "0.50" },
  study: { minutes_7d: 45, tracked: true },
  activity: { asks_last_week: 3, posts_last_week: 0 },
  risk: { level: "NONE", computed_at: "2026-10-12T19:00:00Z" },
  ...over,
});

type Api = { calls: URLSearchParams[] };
async function open(page: Page, handler: (q: URLSearchParams) => unknown | null, url = "/students"): Promise<Api> {
  const api: Api = { calls: [] };
  await page.route(`**/api/v1/courses/${COURSE}/students**`, (r) => {
    const q = new URL(r.request().url()).searchParams;
    api.calls.push(q);
    const body = handler(q);
    return body === null ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(json(body));
  });
  await page.goto(url);
  return api;
}
const pageOf = (items: unknown[], next: string | null = null, total = items.length) => ({ items, next_cursor: next, total });
const rows = (page: Page) => page.locator("[role=row], tr").filter({ has: page.getByText(/Sinh Viên \d+/) });

test.beforeEach(async ({ context }) => {
  await asDemo(context, "teacher");
});

test("students list filters search keep state: cột đúng định dạng, chip + ô tìm đi vào URL và còn khi quay lại từ hồ sơ", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const list = [stu(1, { risk: { level: "AT_RISK", computed_at: "2026-10-12T19:00:00Z" } }), stu(2, { risk: { level: "WATCH", computed_at: "2026-10-12T19:00:00Z" } }), stu(3, { study: { minutes_7d: 0, tracked: false }, attendance: { recorded: 0, present: 0, late: 0, excused: 0, absent: 0 } })];
  const api = await open(page, () => pageOf(list));
  await expect(page.getByRole("heading", { name: "Sinh viên" })).toBeVisible();
  await expect(page.getByText("3 sinh viên")).toBeVisible();
  const r1 = page.getByRole("row", { name: /Sinh Viên 001/ });
  await expect(r1).toContainText("Vắng 2/9");
  await expect(r1).toContainText("2 · +0,50");
  await expect(r1).toContainText("45 phút · 3 câu hỏi");
  await expect(r1).toContainText("Cần chú ý");
  await expect(page.getByRole("row", { name: /Sinh Viên 002/ })).toContainText("Theo dõi");
  const r3 = page.getByRole("row", { name: /Sinh Viên 003/ });
  await expect(r3).toContainText("—"); // chưa ghi buổi nào / không đo thời gian
  await expect(r3).not.toContainText("Cần chú ý");
  await expect(page.getByRole("columnheader", { name: /Điểm/ })).toHaveCount(0); // điểm thuộc P6
  await expect(page.getByRole("button", { name: "Điểm giảm" })).toHaveCount(0);
  expect(api.calls[0].get("limit")).toBe("100");

  await page.getByRole("button", { name: "Cần chú ý" }).click();
  await expect(page).toHaveURL(/filter=attention/);
  await expect.poll(() => api.calls.at(-1)?.get("filter")).toBe("attention");
  await page.getByPlaceholder("Tìm theo tên hoặc MSSV").fill("van an");
  await expect.poll(() => api.calls.at(-1)?.get("q")).toBe("van an");
  await expect(page).toHaveURL(/q=van\+an|q=van%20an/);

  await page.getByRole("row", { name: /Sinh Viên 001/ }).click();
  await expect(page).toHaveURL(/\/students\/00000000-0000-7000-8000-00000000d001$/);
  await page.goBack();
  await expect(page.getByPlaceholder("Tìm theo tên hoặc MSSV")).toHaveValue("van an");
  await expect(page.getByRole("button", { name: "Cần chú ý", pressed: true })).toBeVisible();
});

test("students deep link filter: ?filter= từ Hôm nay bật chip tương ứng và gọi API với bộ lọc đó", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const api = await open(page, () => pageOf([stu(1, { risk: { level: "AT_RISK", computed_at: null } })]), "/students?filter=attention");
  await expect(page.getByRole("button", { name: "Cần chú ý", pressed: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Vắng nhiều", pressed: false })).toBeVisible();
  expect(api.calls[0].get("filter")).toBe("attention");
});

test("students states: lỗi có Thử lại; rỗng khi lọc có Bỏ lọc; lớp chưa có sinh viên", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  let n = 0;
  await open(page, (q) => {
    if (++n === 1) return null;
    return q.get("filter") ? pageOf([]) : pageOf([stu(1)]);
  });
  const retry = page.getByRole("button", { name: "Thử lại" });
  await expect(retry).toBeVisible();
  await retry.click();
  await expect(page.getByRole("row", { name: /Sinh Viên 001/ })).toBeVisible();
  await page.getByRole("button", { name: "Vắng nhiều" }).click();
  await expect(page.getByText("Không có sinh viên nào khớp.")).toBeVisible();
  await page.getByRole("button", { name: "Bỏ lọc" }).click();
  await expect(page.getByRole("row", { name: /Sinh Viên 001/ })).toBeVisible();
  await expect(page).not.toHaveURL(/filter=/);
});

test("students virtual cursor: 1.000 dòng chỉ dựng phần nhìn thấy, cuộn tới đáy xin trang kế bằng con trỏ", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const all = Array.from({ length: 250 }, (_, i) => stu(i + 1));
  const api = await open(page, (q) => {
    const from = q.get("cursor") ? Number(q.get("cursor")) : 0;
    const items = all.slice(from, from + 100);
    return pageOf(items, from + 100 < all.length ? String(from + 100) : null, all.length);
  });
  await expect(page.getByText("250 sinh viên")).toBeVisible();
  expect(await rows(page).count()).toBeLessThan(60); // ảo hoá: không dựng đủ 100 dòng
  const scroller = page.locator("[data-part=virtual-scroll]");
  await scroller.evaluate((el) => el.scrollTo(0, el.scrollHeight));
  await expect.poll(() => api.calls.map((c) => c.get("cursor"))).toContain("100");
  await expect
    .poll(async () => {
      await scroller.evaluate((el) => el.scrollTo(0, el.scrollHeight)); // trang mới dựng xong thì đáy dịch xuống: cuộn lại tới khi xin trang kế
      return api.calls.map((c) => c.get("cursor"));
    })
    .toContain("200");
});

test("students demo session: phiên mô phỏng (không lớp thật) vẫn dùng màn mô phỏng", async ({ context, page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  void context;
  await page.goto("/students");
  await expect(page.getByRole("heading", { name: "Sinh viên" })).toBeVisible();
});
