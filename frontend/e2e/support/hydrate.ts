import type { Page } from "@playwright/test";

/**
 * Từ US-P2-02 vỏ ứng dụng chỉ dựng SAU hydrate (phiên đọc ở client), nên `goto` xong (sự kiện load) chưa chắc đã có DOM.
 * Bọc `page.goto` để đợi `<main>` xuất hiện: các ca đo DOM ngay sau `goto` không còn đua với hydrate (CI chậm hơn máy dev).
 * Trang không có <main> (404…) vẫn qua sau hạn ngắn.
 */
export function settleGoto(page: Page) {
  const goto = page.goto.bind(page);
  page.goto = (async (url: string, opts?: Parameters<Page["goto"]>[1]) => {
    const res = await goto(url, opts);
    await page.locator("main").first().waitFor({ state: "attached", timeout: 15_000 }).catch(() => undefined);
    return res;
  }) as Page["goto"];
}
