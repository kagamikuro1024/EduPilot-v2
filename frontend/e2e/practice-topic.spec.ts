import { expect, test, type Page } from "@playwright/test";
import { practiceApi } from "./support/practice-fixtures";
import { asDemo } from "./support/session";

// US-P9-06 AC3 / AC4 / AC10 / AC12 trên lượt TOPIC (DEF-R10S-04/05/06/07). Gateway giả có trạng thái; chạy cả desktop 1440 và mobile 390.
test.beforeEach(async ({ context }) => {
  await asDemo(context, "student", { person: "sv-2" });
});

async function startTopic(page: Page, down: { on: boolean } = { on: false }) {
  // máy chủ giả luôn "trả lời được" kể cả khi context offline (route fulfill): mô phỏng mất mạng bằng cách huỷ yêu cầu PUT
  const api = await practiceApi(page, { "PUT answers": (route, next) => (down.on ? route.abort("internetdisconnected") : next()) });
  await page.goto("/practice");
  await page.getByRole("listitem").filter({ hasText: "Hàm băm" }).getByRole("button").click();
  await page.waitForURL(/\/practice\/att-1/);
  await expect(page.getByText("Câu 1 về Hàm băm?")).toBeVisible();
  return api;
}

test("TOPIC mất mạng: «Chưa gửi được», không hiện phản hồi, có mạng thì tự gửi lại đúng một lần", async ({ page, context }) => {
  const down = { on: false };
  const api = await startTopic(page, down);
  await page.getByRole("button", { name: /^Phương án B của câu 1/ }).click();
  down.on = true;
  await context.setOffline(true);
  await page.getByRole("button", { name: "Kiểm tra" }).click();
  await expect(page.getByText("Chưa gửi được. Bài vẫn giữ trên máy bạn.")).toBeVisible();
  await expect(page.getByText("Đúng rồi")).toHaveCount(0);
  down.on = false;
  await context.setOffline(false);
  await expect(page.getByText("Đúng rồi")).toBeVisible({ timeout: 15_000 });
  await expect(page.getByText("Chưa gửi được")).toHaveCount(0);
  expect(api.attempts.get("att-1")!.items[0].answer!.version).toBe(1); // máy chủ ghi đúng một lần (lần đầu bị huỷ giữa đường)
});

test("TOPIC tải lại: lựa chọn chưa gửi ở câu hiện tại còn (nháp phía máy)", async ({ page }) => {
  await startTopic(page);
  await page.getByRole("button", { name: /^Phương án B của câu 1/ }).click();
  await page.reload();
  await expect(page.getByRole("button", { name: /^Phương án B của câu 1/ })).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("button", { name: "Kiểm tra" })).toBeEnabled();
});

test("TOPIC: thanh hành động cố định dưới màn hình, nút ≥ 44 px", async ({ page }) => {
  await startTopic(page);
  const bar = page.locator("[class*=actionBar]");
  await expect(bar).toHaveCSS("position", "sticky");
  const vh = page.viewportSize()!.height;
  const box = (await bar.boundingBox())!;
  expect(box.y + box.height).toBeLessThanOrEqual(vh + 1);
  const btn = (await page.getByRole("button", { name: "Kiểm tra" }).boundingBox())!;
  if (page.viewportSize()!.width < 720) expect(btn.height).toBeGreaterThanOrEqual(44); // vùng chạm chỉ đòi ở mobile (AC12)
});

test("TOPIC: danh sách số câu thu gọn chỉ từ 1024 px", async ({ page }) => {
  await startTopic(page);
  const list = page.getByRole("list", { name: "Danh sách câu" });
  if ((page.viewportSize()?.width ?? 0) >= 1024) {
    await expect(list.getByRole("button")).toHaveCount(5);
    await expect(list.getByRole("button", { name: "Câu 1" })).toHaveAttribute("aria-current", "step");
    await expect(list.getByRole("button", { name: "Câu 3" })).toBeDisabled(); // chưa tới câu 3
  } else {
    await expect(list).toHaveCount(0);
  }
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
});
