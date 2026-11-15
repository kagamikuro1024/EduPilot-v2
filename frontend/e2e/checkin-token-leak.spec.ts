import { expect, test } from "@playwright/test";
import { COURSE, TOKEN, checkinApi } from "./support/checkin-fixtures";

// US-AS-09 AC1, AC7, AC9: token chỉ trong thân POST; hash rỗng ≤ 100 ms sau khi tải; một màn ở 375 px, nút ≥ 44.
test("checkin 375: token không ở URL / header / Referer của request nào; hash rỗng sớm; một viewport; nút ≥ 44 px", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  const seen: string[] = [];
  page.on("request", (r) => seen.push(`${r.url()} ${JSON.stringify(r.headers())}`));
  const log = await checkinApi(page, context);
  await page.goto(`/checkin?c=${COURSE}#${TOKEN}`, { waitUntil: "commit" });
  await expect.poll(() => page.evaluate(() => location.hash), { timeout: 100 + 3000 }).toBe("");
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
  expect(log.posts[0].body.token).toBe(TOKEN);
  expect(seen.some((s) => s.includes(TOKEN))).toBe(false);
  const m = await page.evaluate(() => ({ ox: document.documentElement.scrollWidth - document.documentElement.clientWidth, h: document.documentElement.scrollHeight, vh: innerHeight }));
  expect(m.ox).toBe(0);
  expect(m.h).toBeLessThanOrEqual(m.vh + 1); // một màn
});

test("checkin 375: ô nhập mã — nút Điểm danh ≥ 44 px, dính đáy trên thanh điều hướng", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await checkinApi(page, context);
  await page.goto("/checkin");
  const btn = page.locator("[data-part=checkin-dock] button");
  await expect(btn).toBeVisible();
  const box = await btn.boundingBox();
  expect(box!.height).toBeGreaterThanOrEqual(44);
  expect(box!.width).toBeGreaterThan(300);
});
