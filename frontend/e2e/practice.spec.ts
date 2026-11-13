import { expect, test, type Page } from "@playwright/test";
import { COURSE, err, practiceApi } from "./support/practice-fixtures";
import { asDemo } from "./support/session";

// US-P9-05…09: `/practice` THẬT (gateway giả có trạng thái, support/practice-fixtures.ts). Bản chạy với gateway thật là việc của QC.
test.beforeEach(async ({ context }, info) => {
  test.skip(info.project.name !== "desktop", "mobile có ca 375 riêng");
  await asDemo(context, "student", { person: "sv-2" });
});


test("home: lượt đang dở, khuyến nghị, chủ đề, chủ đề ít câu bị khoá kèm lý do", async ({ page }) => {
  await practiceApi(page);
  await page.goto("/practice");
  await expect(page.getByRole("heading", { name: "Luyện đề", level: 1 })).toBeVisible();
  await expect(page.getByText("Luyện Mật mã khối")).toBeVisible();
  const row = page.getByRole("listitem").filter({ hasText: "Ít câu" });
  await expect(row.getByRole("button")).toBeDisabled();
  await expect(row.getByRole("button")).toHaveAttribute("title", "Cần ít nhất 5 câu");
  await page.getByRole("tab", { name: /Thi thử/ }).click();
  await page.getByLabel("Số câu").fill("3");
  await expect(page.getByRole("button", { name: "Bắt đầu thi thử" })).toBeDisabled();
});

test("topic run: chấm ngay, khoá câu đã trả lời, giải thích thêm, kết quả đúng/tổng; start gửi Idempotency-Key + X-Practice-Tab", async ({ page }) => {
  const api = await practiceApi(page);
  await page.goto("/practice");
  await page.getByRole("listitem").filter({ hasText: "Hàm băm" }).getByRole("button").click();
  await page.waitForURL(/\/practice\/att-1/);
  const start = api.calls.find((c) => c.method === "POST" && c.path === "/start")!;
  expect(start.headers["idempotency-key"]).toBeTruthy();
  expect(start.headers["x-practice-tab"]).toMatch(/^[0-9a-f-]{36}$/);

  // câu 1: sai → phản hồi + đáp án đúng; không đổi được
  await page.getByRole("button", { name: /^A\s*Phương án A/ }).click();
  await page.getByRole("button", { name: "Kiểm tra" }).click();
  await expect(page.getByText("Chưa đúng")).toBeVisible();
  await expect(page.getByText("Đáp án đúng: Phương án B của câu 1")).toBeVisible();
  await expect(page.getByRole("button", { name: /^A\s*Phương án A/ })).toBeDisabled();
  await page.getByRole("button", { name: "Giải thích thêm" }).click();
  await expect(page.getByText("Giải thích thêm: B đúng vì khoá đối xứng.")).toBeVisible();
  await page.getByRole("button", { name: "Câu tiếp theo" }).click();

  // các câu còn lại: đúng cả → "Xem kết quả"
  for (let i = 2; i <= 5; i++) {
    await page.getByRole("button", { name: new RegExp(`^B\\s*Phương án B của câu ${i}`) }).click();
    await page.getByRole("button", { name: "Kiểm tra" }).click();
    await expect(page.getByText("Đúng rồi")).toBeVisible();
    await page.getByRole("button", { name: i === 5 ? "Xem kết quả" : "Câu tiếp theo" }).click();
  }
  await expect(page.getByRole("heading", { name: "Bạn đúng 4/5 câu", level: 1 })).toBeVisible();
  await expect(page.getByText("quy đổi 8,0/10")).toBeVisible();
  await expect(page.getByText("Câu cần ôn (1)")).toBeVisible();
  expect(api.calls.filter((c) => c.method === "PUT")).toHaveLength(5); // mỗi câu đúng một lần ghi
});

async function startMock(page: Page, api: Awaited<ReturnType<typeof practiceApi>>) {
  await page.goto("/practice");
  await page.getByRole("tab", { name: /Thi thử/ }).click();
  await page.getByRole("button", { name: "Bắt đầu thi thử" }).click();
  await page.waitForURL(/\/practice\/att-1/);
  await expect(page.locator("[data-part=practice-mock]")).toBeVisible();
  return api;
}

test("mock run: không lộ đúng / sai trước khi nộp, mất mạng giữ bài rồi tự gửi lại, tải lại còn nguyên lựa chọn, nộp có xác nhận", async ({ page, context }) => {
  const api = await practiceApi(page, {}, { mockCount: 5 });
  await startMock(page, api);
  const mock = page.locator("[data-part=practice-mock]");
  await expect(mock.getByText("Đáp án đúng")).toHaveCount(0);
  await expect(mock.getByRole("button", { name: "Giải thích thêm" })).toHaveCount(0);

  await mock.getByRole("button", { name: /^B\s*Phương án B của câu 1/ }).click();
  await expect(mock.getByRole("status").first()).toContainText("Đã lưu lúc");
  await expect(mock.getByText("Đáp án đúng")).toHaveCount(0); // vẫn không có phản hồi sau khi lưu

  // mất mạng: lựa chọn giữ trên máy, báo "Chưa gửi được 1 câu"
  await context.setOffline(true);
  await mock.getByRole("button", { name: "Câu sau" }).click();
  await mock.getByRole("button", { name: /^B\s*Phương án B của câu 2/ }).click();
  await expect(mock.getByText(/Chưa gửi được 1 câu/)).toBeVisible();
  await context.setOffline(false);
  await expect(mock.getByRole("status").first()).toContainText("Đã lưu lúc", { timeout: 20_000 });
  expect(api.attempts.get("att-1")!.items[1].answer).not.toBeNull(); // máy chủ đã nhận đúng lựa chọn ngoại tuyến

  await page.reload();
  await expect(page.locator("[data-part=practice-mock]")).toBeVisible();
  await page.getByRole("button", { name: "Câu sau" }).click();
  await expect(page.getByRole("button", { name: /^B\s*Phương án B của câu 2/ })).toHaveAttribute("aria-pressed", "true");

  await page.getByRole("button", { name: "Nộp bài" }).first().click();
  await expect(page.getByRole("dialog")).toContainText("Bạn đã trả lời 2/5 câu");
  await page.getByRole("dialog").getByRole("button", { name: "Nộp bài" }).click();
  await expect(page.getByRole("heading", { name: "Bạn đúng 2/5 câu", level: 1 })).toBeVisible();
  const submit = api.calls.find((c) => c.method === "POST" && c.path.endsWith("/submit"))!;
  expect(submit.headers["idempotency-key"]).toBeTruthy();
});

test("ATTEMPT_CLOSED: máy chủ đã đóng lượt (hết giờ) → báo hết giờ và nộp theo phần đã lưu, không mất bài", async ({ page }) => {
  const api = await practiceApi(page, { "PUT answers": (route) => { api.closeOnServer("att-1", true); return route.fulfill(err(409, "ATTEMPT_CLOSED")); } }, { mockCount: 5 });
  await startMock(page, api);
  await page.getByRole("button", { name: /^B\s*Phương án B của câu 1/ }).click();
  await expect(page.getByRole("heading", { name: /^Bạn đúng 0\/5 câu$/, level: 1 })).toBeVisible(); // tự nộp theo phần đã lưu (chưa có câu nào)
  await expect(page.getByText("Hết giờ — bài được nộp theo các câu đã lưu.")).toBeVisible();
});

test("TAB_TAKEN_OVER: tab khác giành quyền ghi → tab này chỉ xem, có Làm tiếp ở tab này", async ({ page }) => {
  const api = await practiceApi(page, { "PUT answers": (route) => route.fulfill(err(409, "TAB_TAKEN_OVER")) }, { mockCount: 5 });
  await startMock(page, api);
  await page.getByRole("button", { name: /^B\s*Phương án B của câu 1/ }).click();
  await expect(page.getByText("Bài đang mở ở tab khác")).toBeVisible();
  await expect(page.getByRole("button", { name: "Làm tiếp ở tab này" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Nộp bài" }).first()).toBeDisabled();
});

test("COURSE khớp: lời gọi API đi tới đúng lớp của sinh viên", async ({ page }) => {
  const api = await practiceApi(page);
  const seen: string[] = [];
  page.on("request", (r) => /\/practice\//.test(r.url()) && seen.push(r.url()));
  await page.goto("/practice");
  await expect(page.getByText("Luyện Mật mã khối")).toBeVisible();
  expect(seen.length).toBeGreaterThan(0);
  expect(seen.every((u) => u.includes(`/courses/${COURSE}/`))).toBe(true);
  void api;
});
