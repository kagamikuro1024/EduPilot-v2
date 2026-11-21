# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: activity.spec.ts >> heartbeat cleanup on route change: rời khu vực đo → dọn bộ hẹn giờ, không nhịp nào nữa
- Location: e2e/activity.spec.ts:90:1

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: 1
Received: 3
```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - alert [ref=e2]: Hôm nay · EduPilot
  - generic [ref=e3]:
    - link "Bỏ qua điều hướng" [ref=e4] [cursor=pointer]:
      - /url: "#main"
    - banner [ref=e5]:
      - button "Chọn lớp, đang xem 761987 · An ninh mạng" [ref=e7] [cursor=pointer]:
        - generic "761987" [ref=e8]
        - generic "761987 · An ninh mạng" [ref=e9]: · An ninh mạng
      - generic [ref=e12]:
        - button "Tìm nhanh hoặc đi đến… ⌘ K" [ref=e13] [cursor=pointer]:
          - generic [ref=e17]: Tìm nhanh hoặc đi đến…
          - generic [ref=e18]:
            - generic [ref=e19]: ⌘
            - generic [ref=e20]: K
        - button "Thông báo" [ref=e23] [cursor=pointer]
        - 'button "Tài khoản: Trần Thu Uyên" [ref=e28] [cursor=pointer]':
          - generic [aria-hidden] [ref=e29]: U
          - generic [ref=e30]:
            - generic [ref=e31]: "Tài khoản:"
            - text: Trần Thu Uyên
    - complementary [ref=e34]:
      - link "EduPilot — Hôm nay" [ref=e35] [cursor=pointer]:
        - /url: /
      - navigation "Điều hướng chính" [ref=e36]:
        - list [ref=e38]:
          - listitem [ref=e39]:
            - link "Hôm nay" [ref=e40] [cursor=pointer]:
              - /url: /
          - listitem [ref=e45]:
            - link "Chat riêng" [ref=e46] [cursor=pointer]:
              - /url: /chat
          - listitem [ref=e50]:
            - link "Hỏi đáp công khai" [ref=e51] [cursor=pointer]:
              - /url: /threads
          - listitem [ref=e56]:
            - link "Bài tập" [ref=e57] [cursor=pointer]:
              - /url: /assignments
          - listitem [ref=e63]:
            - link "Luyện đề" [ref=e64] [cursor=pointer]:
              - /url: /practice
          - listitem [ref=e69]:
            - link "Bài thi" [ref=e70] [cursor=pointer]:
              - /url: /exams
          - listitem [ref=e75]:
            - link "Thư viện" [ref=e76] [cursor=pointer]:
              - /url: /library
          - listitem [ref=e80]:
            - link "Lịch" [ref=e81] [cursor=pointer]:
              - /url: /calendar
          - listitem [ref=e85]:
            - link "Kết quả của tôi" [ref=e86] [cursor=pointer]:
              - /url: /me
      - button "Thu gọn thanh bên" [expanded] [ref=e91] [cursor=pointer]:
        - generic [ref=e95]: Thu gọn
    - main [ref=e96]:
      - generic [ref=e97]:
        - generic [ref=e98]:
          - heading "Chào Uyên" [level=1] [ref=e100]
          - generic [ref=e101]: Thứ Bảy, 21/11
        - group "Đang tải" [ref=e102]
```

# Test source

```ts
  1   | import { expect, test, type BrowserContext, type Page } from "@playwright/test";
  2   | import { BASE_URL } from "./support/env";
  3   | import { asDemo } from "./support/session";
  4   | 
  5   | // US-P5-03: heartbeat thời gian học, hộp thoại minh bạch, công tắc đo ở /settings và /class/settings. Gateway giả theo hợp đồng thật (openapi.yaml);
  6   | // đồng hồ giả của Playwright điều khiển chu kỳ 30 s / idle 120 s.
  7   | const C1 = "00000000-0000-7000-8000-00000000c001";
  8   | const CORS = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "authorization,content-type,x-request-id,idempotency-key,accept", "Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS" };
  9   | const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: CORS, body: JSON.stringify(body) });
  10  | const item = (role: string) => ({ course: { id: C1, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: role, enrollment_status: "ACTIVE" });
  11  | 
  12  | type Beat = { course_id: string; area: string; tab_id: string };
  13  | 
  14  | test.beforeEach(async ({}, info) => {
  15  |   test.skip(info.project.name !== "desktop", "ca desktop");
  16  | });
  17  | 
  18  | /** Vào trang với đồng hồ giả; `acked` = đã bấm "Đã hiểu". Pha ngẫu nhiên của hook = 0 nên nhịp đầu rơi ngay. Trả mảng thân các nhịp đã gửi. */
  19  | async function open(page: Page, context: BrowserContext, path: string, o: { acked?: boolean; role?: "student" | "ta" | "teacher"; optOut?: boolean } = {}) {
  20  |   const beats: Beat[] = [];
  21  |   const acked = { v: o.acked ?? true };
  22  |   await asDemo(context, o.role ?? "student", { person: "sv-2" });
  23  |   await page.addInitScript(() => { Math.random = () => 0; });
  24  |   await page.clock.install({ time: new Date() });
  25  |   await page.route("**/api/v1/me/courses**", (r) => (r.request().method() === "OPTIONS" ? r.fulfill({ status: 204, headers: CORS }) : r.fulfill(json({ items: [item(o.role === "ta" ? "TA" : o.role === "teacher" ? "TEACHER" : "STUDENT")], next_cursor: null, tracking_notice_acked: acked.v, study_time_opt_out_available: o.optOut ?? true }))));
  26  |   await page.route("**/api/v1/activity/heartbeat", (r) => {
  27  |     if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: CORS });
  28  |     beats.push(JSON.parse(r.request().postData() ?? "{}"));
  29  |     return r.fulfill({ status: 204, headers: CORS });
  30  |   });
  31  |   await page.route("**/api/v1/me/tracking-notice/ack", (r) => {
  32  |     if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: CORS });
  33  |     acked.v = true;
  34  |     return r.fulfill({ status: 204, headers: CORS });
  35  |   });
  36  |   const loaded = page.waitForResponse((r) => r.url().includes("/me/courses"));
  37  |   await page.goto(path);
  38  |   await loaded;
  39  |   await page.waitForTimeout(300); // thời gian THẬT để React dựng lại hook sau khi lớp / cờ đã về; đồng hồ giả chỉ chạy khi test ra lệnh
  40  |   return beats;
  41  | }
  42  | 
  43  | /** Yêu cầu giả được xử lý bất đồng bộ (thời gian THẬT): đợi số nhịp đạt `n`, hoặc — khi khẳng định "không thêm" — đợi một nhịp thật rồi đọc. */
  44  | const settled = async (beats: Beat[], n: number) => expect.poll(() => beats.length).toBe(n);
  45  | const quiet = async (page: Page, beats: Beat[], n: number) => {
  46  |   await page.waitForTimeout(300);
> 47  |   expect(beats.length).toBe(n);
      |                        ^ Error: expect(received).toBe(expected) // Object.is equality
  48  | };
  49  | 
  50  | const hide = (page: Page, hidden: boolean) =>
  51  |   page.evaluate((h) => {
  52  |     Object.defineProperty(document, "visibilityState", { configurable: true, get: () => (h ? "hidden" : "visible") });
  53  |     document.dispatchEvent(new Event("visibilitychange"));
  54  |   }, hidden);
  55  | 
  56  | test("heartbeat hidden tab stops: ẩn tab → không nhịp nào; hiện lại + thao tác → gửi tiếp", async ({ page, context }) => {
  57  |   const beats = await open(page, context, "/chat");
  58  |   await page.clock.runFor(1_000);
  59  |   await settled(beats, 1);
  60  |   await hide(page, true);
  61  |   await page.clock.runFor(150_000);
  62  |   await quiet(page, beats, 1);
  63  |   await hide(page, false);
  64  |   await page.keyboard.press("Shift");
  65  |   await page.clock.runFor(1_000);
  66  |   await settled(beats, 2);
  67  | });
  68  | 
  69  | test("heartbeat idle 2min stops: không thao tác 2 phút → ngừng; nhịp cuối trước mốc idle", async ({ page, context }) => {
  70  |   const beats = await open(page, context, "/threads");
  71  |   await page.clock.runFor(130_000);
  72  |   await page.waitForTimeout(300);
  73  |   const atIdle = beats.length;
  74  |   expect(atIdle).toBeGreaterThanOrEqual(4);
  75  |   expect(atIdle).toBeLessThanOrEqual(5);
  76  |   await page.clock.runFor(240_000);
  77  |   await quiet(page, beats, atIdle);
  78  | });
  79  | 
  80  | test("heartbeat resumes on input: nghỉ quá 2 phút rồi chạm / gõ → nhịp tiếp trong ≤ 1 s", async ({ page, context }) => {
  81  |   const beats = await open(page, context, "/library");
  82  |   await page.clock.runFor(200_000);
  83  |   await page.waitForTimeout(300);
  84  |   const before = beats.length;
  85  |   await page.mouse.click(300, 300);
  86  |   await page.clock.runFor(1_000);
  87  |   await settled(beats, before + 1);
  88  | });
  89  | 
  90  | test("heartbeat cleanup on route change: rời khu vực đo → dọn bộ hẹn giờ, không nhịp nào nữa", async ({ page, context }) => {
  91  |   const beats = await open(page, context, "/chat");
  92  |   await page.clock.runFor(1_000);
  93  |   await settled(beats, 1);
  94  |   await page.locator("[data-part=sidebar]").getByRole("link", { name: "Hôm nay" }).first().click();
  95  |   await page.keyboard.press("Shift");
  96  |   await page.clock.runFor(100_000);
  97  |   await quiet(page, beats, 1);
  98  | });
  99  | 
  100 | test("heartbeat areas only: chỉ course_id, area, tab_id; area đúng khu vực của trang", async ({ page, context }) => {
  101 |   const beats = await open(page, context, "/chat");
  102 |   await page.clock.runFor(1_000);
  103 |   for (const [path, area] of [["/threads", "THREADS"], ["/library", "LIBRARY"], ["/calendar", "CALENDAR"]] as const) {
  104 |     await page.locator("[data-part=sidebar]").locator(`a[href="${path}"]`).click();
  105 |     await page.keyboard.press("Shift");
  106 |     await page.clock.runFor(31_000);
  107 |     await expect.poll(() => beats.at(-1)?.area, { message: path }).toBe(area);
  108 |   }
  109 |   for (const b of beats) {
  110 |     expect(Object.keys(b).sort()).toEqual(["area", "course_id", "tab_id"]);
  111 |     expect(b.course_id).toBe(C1);
  112 |     expect(["CHAT", "THREADS", "PRACTICE", "LIBRARY", "CALENDAR"]).toContain(b.area);
  113 |     expect(b.tab_id).toMatch(/^[0-9a-f-]{36}$/);
  114 |   }
  115 |   expect(new Set(beats.map((b) => b.tab_id)).size).toBeGreaterThanOrEqual(1);
  116 | });
  117 | 
  118 | test("heartbeat not on exam or me: /me, /exams, Hôm nay không gửi nhịp", async ({ page, context }) => {
  119 |   const beats = await open(page, context, "/me");
  120 |   await page.clock.runFor(100_000);
  121 |   await page.goto("/exams");
  122 |   await page.clock.runFor(100_000);
  123 |   await page.goto("/");
  124 |   await page.clock.runFor(100_000);
  125 |   await page.waitForTimeout(300);
  126 |   expect(beats).toEqual([]);
  127 | });
  128 | 
  129 | test("tracking notice first login: chưa Đã hiểu → hộp thoại, KHÔNG nhịp; Đã hiểu → POST ack, hộp thoại tắt, nhịp bắt đầu", async ({ page, context }) => {
  130 |   const beats = await open(page, context, "/chat", { acked: false });
  131 |   const dialog = page.getByRole("dialog", { name: "Thời gian học trên EduPilot" });
  132 |   await expect(dialog).toBeVisible();
  133 |   await expect(dialog).toContainText("EduPilot ghi thời gian bạn dùng Chat, Hỏi đáp công khai, Luyện đề, Thư viện và Lịch khi tab đang mở và bạn đang thao tác.");
  134 |   await expect(dialog).toContainText("Không ghi nội dung bạn xem hay gõ.");
  135 |   await expect(dialog).toContainText("Giảng viên xem được; không dùng để tính điểm.");
  136 |   await expect(dialog).toContainText("Bạn có thể tắt ở Cài đặt.");
  137 |   await page.clock.runFor(100_000);
  138 |   expect(beats).toEqual([]);
  139 |   const ack = page.waitForRequest((r) => r.url().endsWith("/me/tracking-notice/ack") && r.method() === "POST");
  140 |   await dialog.getByRole("button", { name: "Đã hiểu" }).click();
  141 |   await ack;
  142 |   await expect(page.getByRole("dialog")).toHaveCount(0);
  143 |   await page.keyboard.press("Shift");
  144 |   await page.clock.runFor(1_000);
  145 |   await settled(beats, 1);
  146 | });
  147 | 
```