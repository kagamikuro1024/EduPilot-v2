import { expect, test } from "@playwright/test";
import { CHECKED_IN_AT, COURSE, SUB, TOKEN, checkinApi, err, json } from "./support/checkin-fixtures";

// US-AS-09: /checkin đường web. Gateway giả theo hợp đồng thật.
const LINK = `/checkin?c=${COURSE}#${TOKEN}`;

test("checkin web: đã đăng nhập, liên kết QR → gửi MỘT lần, «Đã điểm danh 07:03», hash bị xoá, khoá thiết bị không xuất được, ký đúng chuỗi", async ({ page, context }) => {
  const log = await checkinApi(page, context);
  await page.goto(LINK);
  await expect(page.locator("[data-part=done]")).toHaveText("Đã điểm danh 07:03");
  expect(log.posts).toHaveLength(1);
  const b = log.posts[0].body as { token: string; device: { kind: string; public_key: string; signature: string } };
  expect(b.token).toBe(TOKEN);
  expect(b.device.kind).toBe("WEB_KEY");
  expect(await page.evaluate(() => location.hash)).toBe("");
  expect(log.posts[0].headers["idempotency-key"]).toBeTruthy();

  // khoá trong IndexedDB: extractable === false (AC3)
  const extractable = await page.evaluate(
    () =>
      new Promise<boolean | null>((res) => {
        const r = indexedDB.open("edupilot.device.v1");
        r.onsuccess = () => {
          const g = r.result.transaction("keys").objectStore("keys").get("web");
          g.onsuccess = () => res(g.result ? g.result.privateKey.extractable : null);
        };
      }),
  );
  expect(extractable).toBe(false);

  // chữ ký ECDSA P-256 của `edupilot-checkin-v1|{token}|{user_id}` kiểm được bằng khoá công khai gửi lên (r‖s 64 byte)
  const ok = await page.evaluate(
    async ({ pub, sig, msg }) => {
      const bin = (s: string) => Uint8Array.from(atob(s), (c) => c.charCodeAt(0));
      const key = await crypto.subtle.importKey("spki", bin(pub), { name: "ECDSA", namedCurve: "P-256" }, false, ["verify"]);
      const raw = bin(sig);
      return { len: raw.length, valid: await crypto.subtle.verify({ name: "ECDSA", hash: "SHA-256" }, key, raw, new TextEncoder().encode(msg)) };
    },
    { pub: b.device.public_key, sig: b.device.signature, msg: `edupilot-checkin-v1|${TOKEN}|${SUB}` },
  );
  expect(ok).toEqual({ len: 64, valid: true });

  // token KHÔNG nằm ở URL / header của bất kỳ yêu cầu /api/v1 nào (AC9)
  expect(log.urls.some((u) => u.includes(TOKEN))).toBe(false);
  expect(log.headers.some((h) => h.includes(TOKEN))).toBe(false);
});

test("checkin web: mở lại cùng liên kết → 200 cùng giờ, không lỗi; lần sau cùng khoá thiết bị (một khoá cho cả trình duyệt)", async ({ page, context }) => {
  let first = "";
  const log = await checkinApi(page, context, { post: (r) => r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 201)) });
  await page.goto(LINK);
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
  first = (log.posts[0].body.device as { public_key: string }).public_key;
  // «mở lại»: trang mới (cùng trình duyệt ⇒ cùng IndexedDB); goto cùng URL chỉ đổi hash nên không tải lại trang
  const p2 = await context.newPage();
  const log2 = await checkinApi(p2, context, { post: (r) => r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 200)) });
  await p2.goto(LINK);
  await expect(p2.locator("[data-part=done]")).toContainText("07:03");
  expect((log2.posts[0].body.device as { public_key: string }).public_key).toBe(first);
});

test("checkin web: mã hết hạn / không đúng / đã đóng / ngoài lớp → một câu theo US-AS-09 AC4; mất mạng → «Chưa có mạng.» + Thử lại", async ({ page, context }) => {
  const codes = ["CHECKIN_TOKEN_EXPIRED", "CHECKIN_TOKEN_INVALID", "WINDOW_CLOSED"];
  const want = ["Mã đã hết hạn. Quét lại mã trên màn chiếu.", "Mã không đúng. Quét lại mã trên màn chiếu.", "Đã đóng điểm danh."];
  for (const [i, code] of codes.entries()) {
    const p = await context.newPage();
    await checkinApi(p, context, { post: (r) => r.fulfill(err(code === "WINDOW_CLOSED" ? 409 : 422, code)) });
    await p.goto(LINK);
    await expect(p.getByText(want[i], { exact: true })).toBeVisible();
    await p.close();
  }
  const p = await context.newPage();
  await checkinApi(p, context, { post: (r) => r.fulfill(json({ code: "FORBIDDEN", message: "m", trace_id: "b".repeat(32), details: { reason: "course" } }, 403)) });
  await p.goto(LINK);
  await expect(p.getByText("Bạn không thuộc lớp này.", { exact: true })).toBeVisible();
  await p.close();

  // mất mạng: apiClient tự thử lại 2 lần (cùng Idempotency-Key) rồi mới báo; «Thử lại» gửi lại và thành công
  let n = 0;
  await checkinApi(page, context, { post: (r) => (++n <= 3 ? r.abort("internetdisconnected") : r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 201))) });
  await page.goto(LINK);
  await expect(page.getByText("Chưa có mạng.", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Thử lại" }).first().click();
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
});

test("checkin web: 429 → «Thử lại sau n giây.» đếm ngược, nút khoá có lý do; rồi tự gửi lại", async ({ page, context }) => {
  let n = 0;
  await checkinApi(page, context, {
    post: (r) => (++n === 1 ? r.fulfill(json({ code: "RATE_LIMITED", message: "m", trace_id: "b".repeat(32), retry_after: 3 }, 429)) : r.fulfill(json({ checked_in_at: CHECKED_IN_AT, round: 1 }, 201))),
  });
  await page.goto(LINK);
  await expect(page.locator("[data-part=limited]")).toHaveText(/Thử lại sau [1-3] giây\./);
  await expect(page.locator("[data-part=done]")).toContainText("07:03", { timeout: 9000 });
});

test("checkin web: lớp kiểm tra vị trí → câu «Lớp này kiểm tra vị trí khi điểm danh.» hiện TRƯỚC khi hỏi định vị; từ chối quyền vẫn điểm danh với geo.denied", async ({ page, context }) => {
  await context.grantPermissions([]); // từ chối định vị
  let lineSeenBeforeGeo = false;
  await page.addInitScript(() => {
    const real = navigator.geolocation.getCurrentPosition.bind(navigator.geolocation);
    navigator.geolocation.getCurrentPosition = (ok, bad, opt) => {
      (window as unknown as { __geoAskedWithLine: boolean }).__geoAskedWithLine = Boolean(document.querySelector("[data-part=geo-line]"));
      real(ok, bad, opt);
    };
  });
  const log = await checkinApi(page, context, { geoCheck: true });
  await page.goto(LINK);
  await expect(page.locator("[data-part=done]")).toContainText("07:03");
  lineSeenBeforeGeo = await page.evaluate(() => (window as unknown as { __geoAskedWithLine: boolean }).__geoAskedWithLine);
  expect(lineSeenBeforeGeo).toBe(true);
  expect(log.posts[0].body.geo).toEqual({ denied: true });
});

test("checkin web: chưa đăng nhập → /login?next= (không có token trong URL), token ở sessionStorage; không có WebCrypto → vẫn điểm danh, không device", async ({ page, context }) => {
  // chưa đăng nhập: không asDemo
  await page.goto(LINK);
  await expect(page).toHaveURL(/\/login\?next=/);
  expect(page.url()).not.toContain(TOKEN);
  const stash = await page.evaluate(() => JSON.parse(sessionStorage.getItem("ep.checkin") ?? "null"));
  expect(stash).toMatchObject({ c: COURSE, token: TOKEN });
  expect(Date.now() - stash.at).toBeLessThan(60_000);

  // không WebCrypto: gửi không `device`
  const p = await context.newPage();
  await p.addInitScript(() => Object.defineProperty(window, "isSecureContext", { value: false }));
  const log = await checkinApi(p, context);
  await p.goto(LINK);
  await expect(p.locator("[data-part=done]")).toContainText("07:03");
  expect(log.posts[0].body.device).toBeUndefined();
});

test("checkin web: giảng viên → màn chặn quyền, 0 yêu cầu /courses/{id}/checkins; c không phải UUID → «Liên kết không đúng.»", async ({ page, context }) => {
  const log = await checkinApi(page, context, { role: "teacher" });
  await page.goto(LINK);
  await expect(page.getByText("Bạn không có quyền xem màn này")).toBeVisible();
  expect(log.posts).toHaveLength(0);

  const p = await context.newPage();
  const log2 = await checkinApi(p, context);
  await p.goto(`/checkin?c=khong-phai-uuid#${TOKEN}`);
  await expect(p.getByText("Liên kết không đúng.", { exact: true })).toBeVisible();
  expect(log2.posts).toHaveLength(0);
});
