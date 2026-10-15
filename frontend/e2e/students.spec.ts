import { expect, test, type Page } from "@playwright/test";
import { panelViolations as violations } from "./support/panels";
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

test("students no real class: không còn màn mô phỏng (AC10), chưa có lớp → câu rỗng và không gọi API sinh viên", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const calls: string[] = [];
  await page.route("**/api/v1/me/courses**", (r) => (r.request().method() === "OPTIONS" ? r.fulfill({ status: 204, headers: json({}).headers }) : r.fulfill(json({ items: [], next_cursor: null }))));
  await page.route(`**/api/v1/courses/**/students**`, (r) => { calls.push(r.request().url()); return r.fulfill(json(pageOf([]))); });
  await page.goto("/students");
  await expect(page.getByRole("heading", { name: "Sinh viên" })).toBeVisible();
  await expect(page.getByText("Chưa có lớp nào.")).toBeVisible();
  expect(calls).toEqual([]);
});

// ===== US-P5-05: hồ sơ 360 thật =====
const SUB = "00000000-0000-7000-8000-0000000000a0"; // `sub` của JWT giả (support/session.ts)
const UID = "00000000-0000-7000-8000-00000000d001";
const profileBody = (over: Record<string, unknown> = {}) => ({
  student: { id: UID, full_name: "Sinh Viên 001", student_code: "B22DCCN101", joined_at: "2026-09-01T00:00:00Z" },
  risk: { level: "AT_RISK", sentence: "Cần chú ý: vắng 3/4 buổi trong 2 tuần gần nhất; học trên EduPilot khoảng 20 phút mỗi tuần.", reasons: [{ code: "ATTENDANCE_LOW", text: "vắng 3/4 buổi trong 2 tuần gần nhất" }], computed_at: "2026-10-12T19:00:00Z" },
  attendance: { recorded: 9, present: 6, late: 1, excused: 0, absent: 2 },
  participation: { speak_count: 2, points_total: "0.50" },
  study: { minutes_7d: 45, tracked: true, weeks: Array.from({ length: 8 }, (_, i) => ({ week_start: `2026-08-${String(17 + i * 7).padStart(2, "0")}`, minutes: 20 + i * 5 })) },
  activity: { asks_last_week: 3, posts_last_week: 1 },
  ...over,
});
const obs = (i: number, over: Record<string, unknown> = {}) => ({ id: `00000000-0000-7000-8000-00000000e00${i}`, author: { id: SUB, full_name: "Giảng viên" }, kind: "NOTE", tags: [], body: `Ghi chú ${i}`, ai_edited: false, version: 1, created_at: "2026-10-10T03:00:00Z", updated_at: "2026-10-10T03:00:00Z", ...over });

type Call = { method: string; path: string; body: string; key: string | null };
type PApi = { calls: Call[]; count: (m: string, tail: string) => number };
async function profileApi(page: Page, h: Record<string, (c: Call) => unknown> = {}): Promise<PApi> {
  const calls: Call[] = [];
  const notes = [obs(1)];
  await page.route(`**/api/v1/courses/${COURSE}/students/${UID}/**`, (r) => {
    const req = r.request();
    if (req.method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    const path = new URL(req.url()).pathname.replace(`/api/v1/courses/${COURSE}/students/${UID}`, "");
    const c: Call = { method: req.method(), path, body: req.postData() ?? "", key: req.headers()["idempotency-key"] ?? null };
    calls.push(c);
    const custom = h[`${c.method} ${path.split("?")[0].replace(/(\/observations)\/[0-9a-f-]{36}$/, "$1")}`];
    if (custom) {
      const out = custom(c);
      if (out === null) return r.fulfill(err(500, "INTERNAL"));
      if (out !== undefined) return r.fulfill(out as never);
    }
    if (path.startsWith("/profile")) return r.fulfill(json(profileBody()));
    if (path.startsWith("/attendance")) return r.fulfill(json({ items: [{ session: { id: "00000000-0000-7000-8000-0000000005a1", session_no: 3, starts_at: "2026-10-07T02:00:00Z" }, status: "ABSENT", note: "Ốm" }], next_cursor: null }));
    if (c.method === "GET" && path.startsWith("/observations")) return r.fulfill(json({ items: notes, next_cursor: null }));
    if (c.method === "POST" && path.startsWith("/observations")) { const b = JSON.parse(c.body) as Record<string, unknown>; const n = obs(notes.length + 1, { ...b, id: `00000000-0000-7000-8000-00000000e00${notes.length + 1}` }); notes.unshift(n); return r.fulfill(json(n, 201)); }
    if (c.method === "PUT") { const b = JSON.parse(c.body) as { body: string }; notes[0] = { ...notes[0], body: b.body, version: 2 }; return r.fulfill(json(notes[0])); }
    if (path.startsWith("/ai-summary")) return r.fulfill(json({ draft: { summary: "Sinh viên đi học chưa đều.", strengths: ["Hay phát biểu"], concerns: ["Vắng nhiều"], suggestions: ["Nhắn riêng hỏi lý do"] } }));
    if (path.startsWith("/outreach")) return r.fulfill(json({ ticket: { id: "00000000-0000-7000-8000-0000000007a1", kind: "TEACHER_OUTREACH", status: "ANSWERED", created_at: "2026-10-14T03:00:00Z" } }, 201));
    return r.fulfill(err(404, "NOT_FOUND"));
  });
  return { calls, count: (m, tail) => calls.filter((c) => c.method === m && c.path.startsWith(tail)).length };
}

test("profile tabs and header: tên, MSSV, lớp, câu nhận định, hai hành động phụ; năm tab", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await profileApi(page);
  await page.goto(`/students/${UID}`);
  await expect(page.getByRole("heading", { name: "Sinh Viên 001" })).toBeVisible();
  await expect(page.getByText(/B22DCCN101/)).toBeVisible();
  await expect(page.getByText(/^Cần chú ý: vắng 3\/4 buổi/)).toBeVisible();
  await expect(page.getByRole("button", { name: "Thêm ghi chú" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Nhắn sinh viên" })).toBeVisible();
  await expect(page.locator("main .ep-btn[data-variant=primary], main button[data-variant=primary]")).toHaveCount(0);
  for (const t of ["Tổng quan", "Chuyên cần", "Điểm", "Hoạt động học", "Ghi chú"]) await expect(page.getByRole("tab", { name: t })).toBeVisible();
  await expect(page.getByText("Có mặt 7/9 buổi, vắng 2; 2 lần phát biểu; tuần này học 45 phút.")).toBeVisible();
  await page.getByRole("tab", { name: "Chuyên cần" }).click();
  await expect(page.getByRole("row", { name: /Buổi 3.*Vắng.*Ốm/ })).toBeVisible();
  await page.getByRole("tab", { name: "Điểm" }).click();
  await expect(page.getByText("Lớp chưa có công thức điểm đã xác nhận.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Xác nhận công thức" })).toBeVisible();
  await page.getByRole("tab", { name: "Hoạt động học" }).click();
  await expect(page.getByText("1 bài đăng").or(page.getByText("Bài đăng tuần trước"))).toBeVisible();
});

test("profile no kpi cards: tab Tổng quan là một đoạn chữ + danh sách term/value, không thẻ số liệu", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await profileApi(page);
  await page.goto(`/students/${UID}`);
  await expect(page.getByRole("tab", { name: "Tổng quan", selected: true })).toBeVisible();
  await expect(page.locator("main dl")).toHaveCount(1);
  await expect(page.locator("main [class*=kpi], main [data-part=kpi]")).toHaveCount(0);
});

test("profile panels per tab: mỗi tab đúng một Panel, không lồng, không nhiều chữ đậm (#29, chuyển từ panels.spec)", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await profileApi(page);
  await page.goto(`/students/${UID}`);
  await page.locator("main [data-ep-panel]:visible").first().waitFor();
  for (const tab of ["Tổng quan", "Chuyên cần", "Điểm", "Hoạt động học", "Ghi chú"]) {
    await page.getByRole("tab", { name: tab }).click();
    await page.waitForTimeout(250);
    expect(await violations(page), tab).toMatchObject({ panels: 1, nest: 0, title: 0, strongMax: 0 });
  }
});

test("profile open three times three requests: mỗi lần mở trang đúng MỘT request, không tự gọi lại", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const api = await profileApi(page);
  for (let i = 0; i < 3; i++) {
    await page.goto(`/students/${UID}`);
    await expect(page.getByRole("heading", { name: "Sinh Viên 001" })).toBeVisible();
    await page.getByRole("tab", { name: "Chuyên cần" }).click(); // đổi tab không gọi lại hồ sơ
    await page.getByRole("tab", { name: "Tổng quan" }).click();
    await page.evaluate(() => window.dispatchEvent(new Event("focus"))); // đổi cửa sổ không gọi lại
  }
  expect(api.count("GET", "/profile")).toBe(3);
});

test("profile states and offline error: lỗi nói dữ liệu không mất, có Thử lại; sau đó tải được", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  let n = 0;
  const api = await profileApi(page, { "GET /profile": () => (++n === 1 ? null : undefined) });
  await page.goto(`/students/${UID}`);
  await expect(page.getByText(/không bị mất/)).toBeVisible();
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByRole("heading", { name: "Sinh Viên 001" })).toBeVisible();
  expect(api.count("GET", "/profile")).toBe(2);
});

test("observation draft survives offline: lưu lỗi mạng giữ chữ trong ô và báo; lưu lại được", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  let n = 0;
  const api = await profileApi(page, { "POST /observations": () => (++n === 1 ? null : undefined) });
  await page.goto(`/students/${UID}`);
  await page.getByRole("tab", { name: "Ghi chú" }).click();
  const box = page.getByLabel("Ghi chú mới");
  await box.fill("Hay đến muộn buổi sáng");
  await page.getByRole("button", { name: "Lưu ghi chú" }).click();
  await expect(page.getByText(/Chữ của bạn vẫn còn trong ô/)).toBeVisible();
  await expect(box).toHaveValue("Hay đến muộn buổi sáng");
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(box).toHaveValue("");
  await expect(page.locator("[data-part=observation]").first()).toContainText("Hay đến muộn buổi sáng");
  expect(api.count("POST", "/observations")).toBe(2);
});

test("observation conflict: sửa đồng thời → hiện bản của người kia, Giữ bản của tôi gửi lại với version mới", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  let n = 0;
  const api = await profileApi(page, {
    "PUT /observations": (c) => {
      if (++n > 1) return undefined;
      void c;
      return { status: 409, contentType: "application/json", headers: json({}).headers, body: JSON.stringify({ code: "VERSION_CONFLICT", message: "m", trace_id: "a".repeat(32), details: { current_version: 5, current: obs(1, { author: { id: "x", full_name: "Trợ giảng Lan" }, body: "Bản của Lan", version: 5 }) } }) };
    },
  });
  await page.goto(`/students/${UID}`);
  await page.getByRole("tab", { name: "Ghi chú" }).click();
  await page.locator("[data-part=observation]").first().getByRole("button", { name: "Sửa" }).click();
  await page.getByLabel("Sửa ghi chú").fill("Bản của tôi");
  await page.getByRole("button", { name: "Lưu", exact: true }).click();
  await expect(page.getByText(/Trợ giảng Lan vừa sửa/)).toContainText("Bản của Lan");
  await page.getByRole("button", { name: "Giữ bản của tôi" }).click();
  await page.getByRole("button", { name: "Lưu", exact: true }).click();
  await expect.poll(() => api.calls.filter((c) => c.method === "PUT").map((c) => (JSON.parse(c.body) as { version: number }).version)).toEqual([1, 5]);
});

test("ai summary edit then save: bản nháp có nhãn AI, sửa rồi mới lưu; ai_edited đúng; lỗi AI có Thử lại và không mất ghi chú đang gõ", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  let n = 0;
  const api = await profileApi(page, {
    "POST /ai-summary": () => (++n === 1 ? { status: 503, contentType: "application/json", headers: json({}).headers, body: JSON.stringify({ code: "LLM_UNAVAILABLE", message: "m", trace_id: "a".repeat(32) }) } : undefined),
  });
  await page.goto(`/students/${UID}`);
  await page.getByRole("tab", { name: "Ghi chú" }).click();
  await page.getByLabel("Ghi chú mới").fill("Đang gõ dở");
  await page.getByRole("button", { name: "Soạn nhận xét tổng hợp" }).click();
  await expect(page.getByText("Chưa soạn được nhận xét. Thử lại.")).toBeVisible();
  await expect(page.getByLabel("Ghi chú mới")).toHaveValue("Đang gõ dở");
  await page.getByRole("button", { name: "Thử lại" }).click();
  const draft = page.getByLabel("Nhận xét tổng hợp (bản nháp AI)");
  await expect(draft).toHaveValue(/Sinh viên đi học chưa đều\./);
  expect(api.calls.some((c) => c.method === "POST" && c.path.startsWith("/observations"))).toBe(false); // chưa lưu
  await draft.fill("Đã sửa bởi tôi");
  await page.getByRole("button", { name: "Lưu nhận xét" }).click();
  await expect.poll(() => api.calls.find((c) => c.method === "POST" && c.path.startsWith("/observations"))?.body).toContain('"ai_edited":true');
  expect(JSON.parse(api.calls.find((c) => c.method === "POST" && c.path.startsWith("/observations"))!.body)).toMatchObject({ kind: "AI_SUMMARY", body: "Đã sửa bởi tôi", ai_edited: true });
  await expect(page.locator("[data-part=observation]").first()).toContainText("Nhận xét tổng hợp · Đã sửa bởi giảng viên");
});

test("outreach reaches student chat: gửi một tin với Idempotency-Key; lỗi mạng giữ chữ và gửi lại dùng cùng khoá", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  let n = 0;
  const api = await profileApi(page, { "POST /outreach": () => (++n === 1 ? null : undefined) });
  await page.goto(`/students/${UID}`);
  await page.getByRole("button", { name: "Nhắn sinh viên" }).click();
  const box = page.getByLabel("Tin nhắn tới Sinh Viên 001");
  await box.fill("Em ghé gặp thầy nhé.");
  await page.getByRole("button", { name: "Gửi", exact: true }).click();
  await expect(page.getByText("Chưa gửi được tin.")).toBeVisible();
  await expect(box).toHaveValue("Em ghé gặp thầy nhé.");
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Đã gửi tin tới Sinh Viên 001")).toBeVisible();
  const posts = api.calls.filter((c) => c.method === "POST" && c.path.startsWith("/outreach"));
  expect(posts).toHaveLength(2);
  expect(posts[0].key).toBeTruthy();
  expect(posts[1].key).toBe(posts[0].key);
  expect(JSON.parse(posts[1].body)).toEqual({ body: "Em ghé gặp thầy nhé." });
});
