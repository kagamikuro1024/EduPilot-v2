import { expect, test } from "@playwright/test";
import { COURSE, CHECKED_IN_AT, TOKEN, checkinApi, json } from "./support/checkin-fixtures";

// DEF-MOB-02/03 (TC mobile pha 1, NET-02/NET-07): đường mã ngắn và đường token phải cùng một cách báo lỗi mạng; lượt gửi có trần 15 s.
test("DEF-MOB-02 mất mạng khi gửi mã ngắn: «Chưa có mạng.» + nút «Thử lại», giữ mã; gửi lại được", async ({ page, context }) => {
  let offline = true;
  const log = await checkinApi(page, context, {
    post: (r) => (offline ? r.abort("internetdisconnected") : r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 201))),
  });
  await page.goto(`/checkin?course=${COURSE}`);
  await page.getByLabel("Mã điểm danh").fill("abc 123");
  await page.getByRole("button", { name: "Điểm danh", exact: true }).first().click();
  const retry = page.getByRole("button", { name: "Thử lại", exact: true }).first();
  await expect(page.locator("[data-part=checkin]")).toContainText("Chưa có mạng.");
  await expect(retry).toBeVisible();
  await expect(page.getByLabel("Mã điểm danh")).toHaveValue("ABC 123"); // mã đã gõ không mất
  offline = false;
  await retry.click();
  await expect(page.locator("[data-part=done]")).toContainText("Đã điểm danh");
  expect(log.posts.length).toBeGreaterThan(1); // mất mạng có thể thử lại nội bộ (POST idempotent)
  expect(log.posts.at(-1)!.body.code).toBe(log.posts[0].body.code); // gửi lại đúng mã đã nhập
});

test("DEF-MOB-03 máy chủ treo khi POST điểm danh: trong 15 s thôi «đang gửi», báo lỗi + «Thử lại», giữ mã", async ({ page, context }) => {
  test.slow();
  let hang = true;
  await checkinApi(page, context, {
    post: async (r) => {
      if (hang) await new Promise((x) => setTimeout(x, 25_000)); // treo lâu hơn trần 15 s
      return r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 201)).catch(() => undefined);
    },
  });
  await page.goto(`/checkin?course=${COURSE}`);
  await page.getByLabel("Mã điểm danh").fill("abc 123");
  const t0 = Date.now();
  await page.getByRole("button", { name: "Điểm danh", exact: true }).first().click();
  await expect(page.locator("[data-part=checkin][data-state=sending]")).toHaveCount(1);
  const retry = page.getByRole("button", { name: "Thử lại", exact: true }).first();
  await expect(retry).toBeVisible({ timeout: 17_000 });
  const wait = Date.now() - t0;
  expect(wait).toBeLessThanOrEqual(16_000);
  await expect(page.locator("[data-part=checkin]")).toContainText("Chưa có mạng.");
  await expect(page.getByLabel("Mã điểm danh")).toHaveValue("ABC 123");
  hang = false;
  await retry.click();
  await expect(page.locator("[data-part=done]")).toContainText("Đã điểm danh");
});

test("token QR quá hạn 60 s: báo hết hạn và xoá ep.checkin khỏi sessionStorage", async ({ page, context }) => {
  await checkinApi(page, context);
  await page.goto("/");
  await page.evaluate(
    (c) => sessionStorage.setItem("ep.checkin", JSON.stringify({ c, token: "v1.dGFpLXRva2Vu.abc", at: Date.now() - 61_000 })),
    COURSE,
  );
  await page.goto(`/checkin?course=${COURSE}`);
  await expect(page.getByText("Mã đã hết hạn. Quét lại mã trên màn chiếu.")).toBeVisible();
  expect(await page.evaluate(() => sessionStorage.getItem("ep.checkin"))).toBeNull();
});

// DEF-MOB-05 (qc3 iOS lô B): tab đang mở /checkin còn stash cũ quá hạn, người dùng mở liên kết QR MỚI (chỉ đổi hash hoặc trang phục hồi từ bfcache).
test("DEF-MOB-05 #token mới trong cùng tab (đổi hash): gửi token MỚI, không báo «Mã đã hết hạn»", async ({ page, context }) => {
  const log = await checkinApi(page, context);
  await page.goto(`/checkin?c=${COURSE}`); // liên kết QR: `?c=` + `#token`
  await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
  await page.evaluate(
    (c) => sessionStorage.setItem("ep.checkin", JSON.stringify({ c, token: "v1.b2xk.abc", at: Date.now() - 61_000 })),
    COURSE,
  );
  await page.evaluate((t) => {
    location.hash = t; // cùng tab: chỉ đổi fragment, trang KHÔNG tải lại
  }, TOKEN);
  await expect(page.locator("[data-part=done]")).toContainText("Đã điểm danh");
  expect(log.posts).toHaveLength(1);
  expect(log.posts[0].body.token).toBe(TOKEN);
  expect(await page.evaluate(() => sessionStorage.getItem("ep.checkin"))).toBeNull();
  await expect(page.getByText("Mã đã hết hạn. Quét lại mã trên màn chiếu.")).toHaveCount(0);
});

test("DEF-MOB-05 #token mới khi Safari phục hồi trang từ bfcache (pageshow persisted)", async ({ page, context }) => {
  const log = await checkinApi(page, context);
  await page.goto(`/checkin?c=${COURSE}`);
  await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
  await page.evaluate(
    ({ c, t }) => {
      sessionStorage.setItem("ep.checkin", JSON.stringify({ c, token: "v1.b2xk.abc", at: Date.now() - 61_000 }));
      history.replaceState(null, "", `/checkin?c=${c}#${t}`); // bfcache: URL đã có token, KHÔNG có hashchange
      dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true })); // Safari phục hồi
    },
    { c: COURSE, t: TOKEN },
  );
  await expect(page.locator("[data-part=done]")).toContainText("Đã điểm danh");
  expect(log.posts).toHaveLength(1);
  expect(log.posts[0].body.token).toBe(TOKEN);
});
