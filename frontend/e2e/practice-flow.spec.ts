import { expect, test, type Page } from "@playwright/test";
import { err, practiceApi } from "./support/practice-fixtures";
import { asDemo } from "./support/session";

// TC-12-10 / US-P9-12 AC3 (DEF-R10S-13), desktop + mobile: thi thử 20 câu → tải lại ở câu 8 → mạng tắt ở câu 12–14 → nộp → kết quả khớp đáp án TÍNH TAY dưới đây;
// luyện theo chủ đề 10 câu có phản hồi ngay; STUDENT B không mở được lượt của A. Máy chủ giả (support/practice-fixtures.ts): đáp án đúng luôn là phương án B.
test.beforeEach(async ({ context }) => { await asDemo(context, "student", { person: "sv-2" }); });

// Đáp án sinh viên chọn (A/B/C) cho câu 1…20. Tính tay: sai ở câu 3 (A), 9 (A), 13 (C), 17 (C) ⇒ 20 − 4 = 16 câu đúng ⇒ «Bạn đúng 16/20 câu», quy đổi 8,0/10.
const PICK = "BBABBBBBABBBCBBBCBBB".split("");
const CORRECT = 16;

async function open(page: Page, tab: RegExp, start: string) {
  await page.goto("/practice");
  await page.getByRole("tab", { name: tab }).click();
  await page.getByRole("button", { name: start }).click();
  await page.waitForURL(/\/practice\/att-1/);
}

test("thi thử 20 câu: tải lại ở câu 8, tắt/bật mạng ở câu 12–14, nộp → 16/20 theo đáp án tính tay", async ({ page, context }) => {
  await practiceApi(page, {}, { mockCount: 20 });
  await open(page, /Thi thử/, "Bắt đầu thi thử");
  const pick = (n: number) => page.getByRole("button", { name: new RegExp(`^Phương án ${PICK[n - 1]} của câu ${n}`) }).click();
  const next = () => page.getByRole("button", { name: "Câu sau" }).click();

  for (let n = 1; n <= 8; n++) {
    if (n > 1) await next();
    await pick(n);
  }
  await expect(page.getByRole("status").first()).toContainText("Đã lưu lúc");
  await page.reload();
  await expect(page.getByText("Câu 8/20")).toBeVisible();
  await expect(page.getByRole("button", { name: /^Phương án B của câu 8/ })).toHaveAttribute("aria-pressed", "true");

  for (let n = 9; n <= 11; n++) { await next(); await pick(n); }
  await context.setOffline(true);
  for (let n = 12; n <= 14; n++) { await next(); await pick(n); }
  await expect(page.getByText(/^Chưa gửi được/).first()).toBeVisible();
  await context.setOffline(false);
  await expect(page.getByRole("status").first()).toContainText("Đã lưu lúc", { timeout: 20_000 });
  for (let n = 15; n <= 20; n++) { await next(); await pick(n); }

  await page.getByRole("button", { name: "Nộp bài" }).first().click();
  await expect(page.getByRole("dialog")).toContainText("Bạn đã trả lời 20/20 câu");
  await page.getByRole("dialog").getByRole("button", { name: "Nộp bài" }).click();
  await expect(page.getByRole("heading", { name: `Bạn đúng ${CORRECT}/20 câu`, level: 1 })).toBeVisible();
  await expect(page.getByText("quy đổi 8,0/10")).toBeVisible();
});

test("luyện theo chủ đề 10 câu: phản hồi ngay sau mỗi câu, kết quả đúng/tổng", async ({ page }) => {
  await practiceApi(page, {}, { topicCount: 10 });
  await page.goto("/practice");
  await page.getByRole("listitem").filter({ hasText: "Hàm băm" }).getByRole("button").click();
  await page.waitForURL(/\/practice\/att-1/);
  for (let n = 1; n <= 10; n++) {
    const wrong = n === 4; // chọn A ở câu 4 ⇒ 9/10
    await page.getByRole("button", { name: new RegExp(`^Phương án ${wrong ? "A" : "B"} của câu ${n}`) }).click();
    await page.getByRole("button", { name: "Kiểm tra" }).click();
    await expect(page.getByText(wrong ? "Chưa đúng" : "Đúng rồi")).toBeVisible();
    await page.getByRole("button", { name: n === 10 ? "Xem kết quả" : "Câu tiếp theo" }).click();
  }
  await expect(page.getByRole("heading", { name: "Bạn đúng 9/10 câu", level: 1 })).toBeVisible();
});

test("STUDENT B không mở được lượt của A: 404 → «Không tìm thấy lượt luyện», không lộ câu hỏi", async ({ page, context }) => {
  let asB = false;
  await practiceApi(page, { "GET ": async (route, next) => (asB ? route.fulfill(err(404, "NOT_FOUND")) : next()) }, { mockCount: 5 });
  await open(page, /Thi thử/, "Bắt đầu thi thử");
  await expect(page.locator("[data-part=practice-mock]")).toBeVisible();
  asB = true; // người gọi bây giờ là sinh viên B (cùng đường dẫn lượt của A)
  await asDemo(context, "student", { person: "sv-3" });
  await page.reload();
  await expect(page.getByText("Không tìm thấy lượt luyện")).toBeVisible();
  await expect(page.locator("[data-part=practice-mock]")).toHaveCount(0);
  await expect(page.getByText(/Phương án [ABC] của câu/)).toHaveCount(0);
});
