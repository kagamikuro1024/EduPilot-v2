import type { Page } from "@playwright/test";

/**
 * Từ US-PU-06 vỏ và `/dev/ui` được vẽ từ máy chủ: `<main>` đã có trong HTML trước hydrate, nên đợi `<main>` không còn đủ — CI chậm hơn máy dev,
 * ca đo DOM ngay sau `goto` chạy giữa lúc hydrate và gặp "Element is not attached". React gắn khoá `__reactFiber$…` lên một nút DOM khi hydrate tới nút đó:
 * đợi cả `<main>` lẫn phần tử cuối cùng trong nó có khoá này. Trang không có React (404…) vẫn qua sau hạn ngắn.
 */
export function settleGoto(page: Page) {
  const goto = page.goto.bind(page);
  page.goto = (async (url: string, opts?: Parameters<Page["goto"]>[1]) => {
    const res = await goto(url, opts);
    await page.locator("main").first().waitFor({ state: "attached", timeout: 15_000 }).catch(() => undefined);
    await page
      .waitForFunction(
        () => {
          const hydrated = (el: Element | null) => !!el && Object.keys(el).some((k) => k.startsWith("__reactFiber"));
          const main = document.querySelector("main");
          if (!main) return true;
          const all = main.querySelectorAll("*");
          return hydrated(main) && (all.length === 0 || hydrated(all[all.length - 1]));
        },
        undefined,
        { timeout: 10_000 },
      )
      .catch(() => undefined);
    return res;
  }) as Page["goto"];
}
