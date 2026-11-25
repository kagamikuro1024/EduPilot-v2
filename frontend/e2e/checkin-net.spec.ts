import { expect, test } from "@playwright/test";
import { COURSE, CHECKED_IN_AT, TOKEN, checkinApi, item, json } from "./support/checkin-fixtures";
import { asDemo } from "./support/session";

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
    (c) => sessionStorage.setItem("ep.checkin", JSON.stringify({ c, token: "e7c1a1c2-1111-4c3d-8e9f-0a1b2c3d4e5f.3.ABCDEFGHIJKLMNOP", at: Date.now() - 61_000 })),
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
      sessionStorage.setItem("ep.checkin", JSON.stringify({ c, token: "e7c1a1c2-1111-4c3d-8e9f-0a1b2c3d4e5f.3.ABCDEFGHIJKLMNOP", at: Date.now() - 61_000 }));
      history.replaceState(null, "", `/checkin?c=${c}#${t}`); // bfcache: URL đã có token, KHÔNG có hashchange
      dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true })); // Safari phục hồi
    },
    { c: COURSE, t: TOKEN },
  );
  await expect(page.locator("[data-part=done]")).toContainText("Đã điểm danh");
  expect(log.posts).toHaveLength(1);
  expect(log.posts[0].body.token).toBe(TOKEN);
});

test("DEF-MOB-05b skip link «Bỏ qua điều hướng» ở /checkin: fragment #main không bị coi là token (0 POST, focus tới #main, ô mã còn)", async ({ page, context }) => {
  const log = await checkinApi(page, context);
  await page.goto(`/checkin?course=${COURSE}`);
  const field = page.getByLabel("Mã điểm danh");
  await expect(field).toBeVisible();
  await page.keyboard.press("Tab"); // skip link chỉ hiện khi được focus bằng bàn phím
  const skip = page.getByRole("link", { name: "Bỏ qua điều hướng" });
  await expect(skip).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.locator("#main")).toBeFocused();
  expect(page.url()).toContain("#main"); // không bị replaceState xoá
  expect(log.posts).toHaveLength(0);
  await expect(field).toBeVisible(); // ô mã không biến mất
});

test("DEF-MOB-07 token cũ 91 s bị xoá khi màn dựng lại (đăng nhập chen giữa): vẫn hiện «Mã đã hết hạn» một lần, sessionStorage sạch, 0 POST", async ({ page, context }) => {
  const log = await checkinApi(page, context);
  await page.goto(`/checkin?course=${COURSE}`);
  await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
  await page.evaluate(
    (c) => sessionStorage.setItem("ep.checkin", JSON.stringify({ c, token: "e7c1a1c2-1111-4c3d-8e9f-0a1b2c3d4e5f.3.ABCDEFGHIJKLMNOP", at: Date.now() - 91_000 })),
    COURSE,
  );
  await page.goto("/"); // rời màn rồi quay lại = màn dựng lại (như trang đăng nhập chen giữa)
  await page.goto(`/checkin?course=${COURSE}`);
  await expect(page.getByText("Mã đã hết hạn. Quét lại mã trên màn chiếu.")).toBeVisible();
  expect(await page.evaluate(() => sessionStorage.getItem("ep.checkin"))).toBeNull(); // token bị xoá (yêu cầu 60 s)
  await page.goto("/");
  await page.goto(`/checkin?course=${COURSE}`);
  await expect(page.getByText("Mã đã hết hạn. Quét lại mã trên màn chiếu.")).toBeVisible(); // vẫn báo, không rơi về ô nhập mã
  await expect(page.getByLabel("Mã điểm danh")).toHaveCount(0);
  expect(log.posts).toHaveLength(0);
  await page.goto("/");
  await page.goto(`/checkin?course=${COURSE}`);
  await expect(page.getByLabel("Mã điểm danh")).toBeVisible(); // câu chỉ hiện MỘT lần
});

// DEF-MOB-07b (qc3 iOS ERR-06): đăng XUẤT trước, mở liên kết QR MỚI (nên /login giữ `next=`), chờ lâu, rồi đăng nhập.
test("DEF-MOB-07b đăng xuất → mở link QR mới → đứng ở /login → đăng nhập: câu «Mã đã hết hạn», 0 POST", async ({ browser }) => {
  const context = await browser.newContext(); // ngữ cảnh mới = chưa đăng nhập
  const page = await context.newPage();
  const posts: string[] = [];
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items: [item(COURSE, "761988")], next_cursor: null })));
  await page.route(`**/api/v1/courses/${COURSE}`, (r) => r.fulfill(json({ id: COURSE, class_code: "761988", attendance: { self_checkin: true, require: "WEB", geo_check: false } })));
  await page.route(`**/api/v1/courses/${COURSE}/checkins`, (r) => {
    posts.push(r.request().postData() ?? "");
    return r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 201));
  });
  await page.goto(`/checkin?c=${COURSE}#${TOKEN}`);
  await expect(page).toHaveURL(/\/login/, { timeout: 20_000 });
  // «chờ 91 s»: già đi mốc `at` của token thay vì ngủ thật (cùng một trạng thái vào lúc đăng nhập)
  await page.evaluate(() => {
    const raw = sessionStorage.getItem("ep.checkin");
    if (!raw) return;
    const o = JSON.parse(raw) as { at: number };
    sessionStorage.setItem("ep.checkin", JSON.stringify({ ...o, at: Date.now() - 91_000 }));
  });
  await asDemo(context, "student"); // đăng nhập xong; redirect sau đăng nhập = next= của /login
  await page.goto(`/checkin?c=${COURSE}`);
  await expect(page.getByText("Mã đã hết hạn. Quét lại mã trên màn chiếu.")).toBeVisible();
  await expect(page.getByLabel("Mã điểm danh")).toHaveCount(0);
  expect(posts).toHaveLength(0);
  expect(await page.evaluate(() => sessionStorage.getItem("ep.checkin"))).toBeNull();
  await context.close();
});

// DEF-MOB-08 (qc2 GEO-05): lớp bật geofence, hộp quyền vị trí không trả lời / giữ lâu ⇒ VẪN gửi điểm danh với geo.denied=true (AC4/AC5).
test("DEF-MOB-08 geofence + hộp quyền vị trí KHÔNG bao giờ trả lời: vẫn đúng 1 POST với geo.denied, «Đã điểm danh», không «Chưa có mạng.»", async ({ page, context }) => {
  test.slow();
  const log = await checkinApi(page, context, { geoCheck: true });
  await page.addInitScript(() => {
    // WebView không gọi callback (hộp quyền treo) — trần 8 s của trình duyệt cũng không cứu
    navigator.geolocation.getCurrentPosition = () => undefined;
  });
  await page.goto(`/checkin?course=${COURSE}`);
  await page.getByLabel("Mã điểm danh").fill("abc 123");
  const t0 = Date.now();
  await page.getByRole("button", { name: "Điểm danh", exact: true }).first().click();
  await expect(page.locator("[data-part=done]")).toContainText("Đã điểm danh", { timeout: 25_000 });
  expect(Date.now() - t0).toBeLessThanOrEqual(22_000);
  await expect(page.getByText("Chưa có mạng.")).toHaveCount(0);
  expect(log.posts).toHaveLength(1);
  expect(log.posts[0].body.geo).toMatchObject({ denied: true });
});

test("DEF-MOB-08 geofence + hộp quyền vị trí giữ 16 s rồi cấp: không «Chưa có mạng.», đúng 1 POST, «Đã điểm danh»", async ({ page, context }) => {
  test.slow();
  const log = await checkinApi(page, context, { geoCheck: true });
  await page.addInitScript(() => {
    navigator.geolocation.getCurrentPosition = (ok: PositionCallback) => {
      setTimeout(() => ok({ coords: { latitude: 10, longitude: 106, accuracy: 12 } as GeolocationCoordinates } as GeolocationPosition), 16_000);
    };
  });
  await page.goto(`/checkin?course=${COURSE}`);
  await page.getByLabel("Mã điểm danh").fill("abc 123");
  await page.getByRole("button", { name: "Điểm danh", exact: true }).first().click();
  await expect(page.locator("[data-part=done]")).toContainText("Đã điểm danh", { timeout: 25_000 });
  await expect(page.getByText("Chưa có mạng.")).toHaveCount(0);
  expect(log.posts).toHaveLength(1);
  expect(log.posts[0].body.geo).toMatchObject({ denied: true }); // trần định vị 10 s ⇒ coi như từ chối, không chờ 16 s
});

// DEF-MOB-09 (qc2 GEO-05): lớp geofence — hộp quyền vị trí phải mở NGAY khi vào màn (AC4), không chen vào lúc mã đang bay (mã rotate 10 s).
test("DEF-MOB-09 geofence: geolocation được gọi TRƯỚC khi bấm gửi (câu «kiểm tra vị trí» hiện ở ô nhập mã), POST vẫn xong", async ({ page, context }) => {
  test.slow();
  const log = await checkinApi(page, context, { geoCheck: true });
  await page.addInitScript(() => {
    (window as unknown as { __geo: number }).__geo = 0;
    navigator.geolocation.getCurrentPosition = (ok: PositionCallback) => {
      (window as unknown as { __geo: number }).__geo++;
      setTimeout(() => ok({ coords: { latitude: 10, longitude: 106, accuracy: 12 } as GeolocationCoordinates } as GeolocationPosition), 16_000);
    };
  });
  await page.goto(`/checkin?course=${COURSE}`);
  await expect(page.getByLabel("Mã điểm danh")).toBeVisible();
  await expect(page.getByText("Lớp này kiểm tra vị trí khi điểm danh.")).toBeVisible(); // câu hiện ở màn nhập mã
  await expect.poll(() => page.evaluate(() => (window as unknown as { __geo: number }).__geo)).toBeGreaterThan(0); // xin quyền TRƯỚC khi bấm
  await page.getByLabel("Mã điểm danh").fill("abc 123");
  const t0 = Date.now();
  await page.getByRole("button", { name: "Điểm danh", exact: true }).first().click();
  await expect(page.locator("[data-part=done]")).toContainText("Đã điểm danh", { timeout: 20_000 });
  expect(Date.now() - t0).toBeLessThanOrEqual(15_000); // không chờ thêm hộp quyền lúc gửi
  expect(log.posts).toHaveLength(1);
  expect(log.posts[0].body.geo).toMatchObject({ denied: true }); // trần định vị 10 s đã chốt trước khi bấm
});
