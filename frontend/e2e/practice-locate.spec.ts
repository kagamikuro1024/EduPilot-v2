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
  await page.goto(`/practice?course=${C2}`);
  await page.evaluate(([c]) => localStorage.setItem("ep:ui:course", c), [C2]);
  await page.goto("/practice");
  await page.getByRole("listitem").filter({ hasText: "Hàm băm" }).getByRole("button").click();
  await page.waitForURL(/\/practice\/att-1/);
  probes.length = 0;
  locates = 0;
  await page.goto("/practice/att-1");
  await expect(page.getByRole("button", { name: /^Phương án B của câu 1/ })).toBeVisible();
  expect(locates).toBe(1);
  expect(probes.length).toBe(1);
  expect(probes[0]).toContain(`/courses/${C2}/`);
});
