import { expect, test } from "@playwright/test";
import { json, practiceApi } from "./support/practice-fixtures";
import { asDemo } from "./support/session";

// DEF-R10S-20 (TC-12-17, luật 13): /practice/[id] không có ?course= → hỏi MỘT lần lớp của lượt rồi tải lượt đúng lớp; KHÔNG dò từng lớp của sinh viên (26 lớp ⇒ 26 GET tuần tự, LCP 18 s).
// Sinh viên sv-1 có hai lớp; lượt nằm ở lớp THỨ HAI (c002), nên cách dò cũ phải gọi lớp đầu (404) trước. Desktop + mobile.
const C1 = "00000000-0000-7000-8000-00000000c001";
const C2 = "00000000-0000-7000-8000-00000000c002";

test("mở /practice/[id] không ?course=: đúng 1 request tìm lượt, không dò lớp khác", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-1" });
  await practiceApi(page, {}, { topicCount: 5 });
  const probes: string[] = [];
  let locates = 0;
  await page.route(new RegExp(`/api/v1/courses/${C1}/practice/attempts/`), (r) => (r.request().method() === "OPTIONS" ? r.fallback() : r.fulfill(json({ code: "NOT_FOUND", message: "m", trace_id: "t".repeat(32) }, 404))));
  await page.route(/\/api\/v1\/me\/practice\/attempts\/[^/]+$/, (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill(json({}, 204));
    locates++;
    return r.fulfill(json({ course_id: C2 }));
  });
  page.on("request", (r) => /\/courses\/[^/]+\/practice\/attempts\/[^/]+$/.test(r.url()) && r.method() === "GET" && probes.push(r.url()));

  // tạo lượt ở lớp 2 qua gateway giả (start không phân biệt lớp), rồi mở thẳng bằng id, không ?course=
  await page.goto("/practice");
  await page.getByRole("listitem").filter({ hasText: "Hàm băm" }).getByRole("button").click();
  await page.waitForURL(/\/practice\/att-1/);
  await expect(page.getByRole("button", { name: /^Phương án B của câu 1/ })).toBeVisible(); // lần mở đầu (hint lớp 1 → 404 → hỏi lớp) xong hẳn rồi mới đếm
  probes.length = 0;
  locates = 0;
  await page.goto("/practice/att-1");
  await expect(page.getByRole("button", { name: /^Phương án B của câu 1/ })).toBeVisible();
  expect(locates).toBe(1);
  expect(probes.length).toBe(1);
  expect(probes[0]).toContain(`/courses/${C2}/`);
});

// Bắt đầu → /practice/<id>?course=… giữ tham số (không bị bỏ như deep link thường): đúng 0 request tìm lượt, 1 GET lượt; tải lại vẫn 0 request tìm lượt.
test("bắt đầu luyện: 0 request tìm lượt, ?course= ở lại URL, tải lại vẫn 0", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-1" });
  await practiceApi(page, {}, { topicCount: 5 });
  let locates = 0;
  const gets: string[] = [];
  await page.route(/\/api\/v1\/me\/practice\/attempts\/[^/]+$/, (r) => { locates++; return r.fulfill(json({ course_id: C1 })); });
  page.on("request", (r) => /\/courses\/[^/]+\/practice\/attempts\/[^/]+$/.test(r.url()) && r.method() === "GET" && gets.push(r.url()));
  await page.goto("/practice");
  await page.getByRole("listitem").filter({ hasText: "Hàm băm" }).getByRole("button").click();
  await page.waitForURL(/\/practice\/att-1/);
  await expect(page.getByRole("button", { name: /^Phương án B của câu 1/ })).toBeVisible();
  await page.waitForTimeout(500);
  expect(page.url()).toContain(`?course=${C1}`);
  expect(locates).toBe(0);
  expect(gets.length).toBe(1);
  await page.reload();
  await expect(page.getByRole("button", { name: /^Phương án B của câu 1/ })).toBeVisible();
  expect(locates).toBe(0);
});

// DEF-R10S-20 (3): URL đã có ?course= thì GET lượt chạy SONG SONG với /me/courses (LCP 4,4 s vì chờ /me/courses 3,7 s). /me/courses trễ 1,5 s: GET lượt phải bắt đầu TRƯỚC khi nó trả.
test("mở /practice/<id>?course=…: GET lượt không chờ /me/courses", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  await practiceApi(page, {}, { topicCount: 5 });
  await page.goto("/practice");
  await page.getByRole("listitem").filter({ hasText: "Hàm băm" }).getByRole("button").click();
  await page.waitForURL(/\/practice\/att-1\?course=/);

  let coursesDone = false;
  await page.route("**/api/v1/me/courses**", async (r) => { await new Promise((res) => setTimeout(res, 1500)); await r.fallback(); coursesDone = true; });
  let coursesDoneAtAttemptGet: boolean | null = null;
  page.on("request", (r) => { if (coursesDoneAtAttemptGet === null && /\/courses\/[^/]+\/practice\/attempts\/[^/]+$/.test(r.url()) && r.method() === "GET") coursesDoneAtAttemptGet = coursesDone; });
  await page.reload();
  await expect(page.getByRole("button", { name: /^Phương án B của câu 1/ })).toBeVisible();
  expect(coursesDoneAtAttemptGet).toBe(false);
});
