import { test, expect } from "@playwright/test";
import { call, login, needReal, shot, sql, uiLogin, uniq, noOverflow } from "./qc-real-lib";

// QC @real (P-11-29): bật ba ca @real của class-join.spec.ts (bộ chọn lớp theo seed, nhập danh sách + chia sẻ nguồn, vào lớp end-to-end 375×812)
// và ca @real của today.spec.ts. Lớp thử tạo mới (QF<tag>) để không đụng dữ liệu 761987 / 761988; xong lưu trữ lớp.
needReal();
test.describe.configure({ mode: "serial", timeout: 240_000 });
const tag = uniq();
const S: { cid: string; code: string } = { cid: "", code: "" };

test("@real bộ chọn lớp theo seed: GV 2 lớp, SV A/B ≥ 1 lớp ACTIVE, SV D (sv.moi) chưa có lớp → màn «Bạn chưa vào lớp nào»", async ({ browser, request }) => {
  const count = async (u: string) => (((await call(request, await login(request, u), "GET", "/me/courses")).json.items ?? []) as { status?: string }[]).filter((i) => (i.status ?? "ACTIVE") === "ACTIVE").length;
  const nT = await count("teacher");
  expect(nT, "GV có đúng 2 lớp (761987, 761988)").toBe(2);
  expect(await count("sv.gioi")).toBeGreaterThanOrEqual(1);
  expect(await count("sv.kha")).toBeGreaterThanOrEqual(1);
  const nD = await count("sv.moi");
  for (const [u, w] of [["teacher", 1440], ["sv.moi", 1440], ["sv.moi", 375]] as const) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN" });
    const p = await ctx.newPage();
    await uiLogin(p, u);
    if (u === "teacher") {
      await p.getByRole("button", { name: /lớp/i }).first().click().catch(() => undefined);
      await expect(p.getByText(/Tất cả lớp của tôi/)).toBeVisible();
      await expect(p.getByText(/761987/).first()).toBeVisible();
      await expect(p.getByText(/761988/).first()).toBeVisible();
    } else {
      expect(nD, "seed: sv.moi (D) cố ý chưa vào lớp nào").toBe(0);
      await expect(p.getByText(/Bạn chưa vào lớp nào|mã (lớp|tham gia)/i).first()).toBeVisible();
      await noOverflow(p);
    }
    await shot(p, `f2-picker-${u}-${w}`);
    await ctx.close();
  }
});

test("@real nhập danh sách: GV mở lớp mới, nhập roster-2-loi.csv → báo đúng dòng 7 và 19, nút «Nhập 28 sinh viên»; TA không thấy tab", async ({ browser, request }) => {
  const admin = await login(request, "admin");
  const tid = (((await call(request, admin, "GET", "/admin/users?q=teacher@&limit=5")).json.items ?? []) as { id: string; email: string }[]).find((u) => u.email === "teacher@edupilot.local")!.id;
  const mk = await call(request, admin, "POST", "/admin/courses", { subject_code: "QCF", class_code: `QF${tag}`.toUpperCase().slice(0, 12), name: `QC luồng F2 ${tag}`, semester: "2026-2027-HK1", capacity: 60, teacher_id: tid });
  expect([200, 201], mk.text.slice(0, 200)).toContain(mk.status);
  S.cid = String(mk.json.id);
  const jc = (await call(request, await login(request, "teacher"), "GET", `/courses/${S.cid}/join-code`)).json;
  S.code = String(jc.code ?? jc.join_code);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const p = await ctx.newPage();
  await uiLogin(p, "teacher");
  await p.goto(`/class/members?course=${S.cid}`);
  await p.getByRole("tab", { name: "Nhập danh sách" }).click();
  await p.getByLabel("Gửi thư mời cho sinh viên chưa có tài khoản").uncheck();
  await p.getByLabel("Tệp danh sách lớp").setInputFiles("e2e/fixtures/roster-2-loi.csv");
  await p.getByRole("button", { name: "Xem trước" }).click();
  await expect(p.getByRole("cell", { name: 'Dòng 7 · Email · "Email không đúng dạng."' })).toBeVisible();
  await expect(p.getByRole("cell", { name: 'Dòng 19 · Họ và tên · "Thiếu họ và tên."' })).toBeVisible();
  await expect(p.getByText(/28 sẽ được nhập/)).toBeVisible();
  await shot(p, "f2-roster-preview");
  await p.getByRole("button", { name: "Nhập 28 sinh viên" }).click();
  await expect(p.getByRole("status").filter({ hasText: "Đã nhập 28 sinh viên · 2 dòng lỗi chưa nhập." })).toBeVisible({ timeout: 60_000 });
  expect(Number(sql(`select count(*) from roster_entries where course_id='${S.cid}'`)[0]?.[0] ?? sql(`select count(*) from enrollments where course_id='${S.cid}'`)[0][0])).toBeGreaterThanOrEqual(28);
  await shot(p, "f2-roster-imported");
  await ctx.close();
  // TA (lớp 761987) không thấy tab nhập danh sách
  const ta = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const tp = await ta.newPage();
  await uiLogin(tp, "ta");
  await tp.goto("/class/members");
  await expect(tp.getByRole("tab", { name: "Nhập danh sách" })).toHaveCount(0);
  await ta.close();
});

test("@real vào lớp end-to-end 375×812: D xem trước và vào lớp, GV duyệt, tạo lại mã (mã cũ chết ngay), 6 mã sai ⇒ 429", async ({ browser, request }) => {
  const teacher = await login(request, "teacher");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, locale: "vi-VN" });
  const p = await ctx.newPage();
  await uiLogin(p, "sv.moi");
  await p.goto("/join");
  await p.getByLabel("Mã tham gia").fill(S.code);
  await p.getByRole("button", { name: "Xem lớp" }).click();
  await expect(p.locator("[data-part=join-preview]")).toContainText(`QC luồng F2 ${tag}`);
  await noOverflow(p);
  await shot(p, "f2-join-preview-375");
  await p.getByRole("button", { name: "Tham gia lớp" }).click();
  await expect(p.getByRole("status").filter({ hasText: /Bạn đã vào lớp|chờ duyệt|Đã gửi yêu cầu/ })).toBeVisible();
  await shot(p, "f2-join-done-375");
  const st = sql(`select e.status from enrollments e join users u on u.id=e.user_id where u.email='sv.moi@edupilot.local' and e.course_id='${S.cid}'`)[0]?.[0];
  expect(["ACTIVE", "PENDING"], `trạng thái vào lớp: ${st}`).toContain(st);
  if (st === "PENDING") {
    const mem = (((await call(request, teacher, "GET", `/courses/${S.cid}/members?status=PENDING`)).json.items ?? []) as { id: string; user_id?: string }[])[0];
    const ap = await call(request, teacher, "POST", `/courses/${S.cid}/members/${mem.id}/approve`, {});
    expect([200, 204], ap.text.slice(0, 150)).toContain(ap.status);
  }
  expect(sql(`select e.status from enrollments e join users u on u.id=e.user_id where u.email='sv.moi@edupilot.local' and e.course_id='${S.cid}'`)[0][0]).toBe("ACTIVE");
  // tạo lại mã: mã cũ chết ngay
  const old = S.code;
  const rg = await call(request, teacher, "POST", `/courses/${S.cid}/join-code/regenerate`, {});
  expect([200, 201], rg.text.slice(0, 150)).toContain(rg.status);
  expect(String(rg.json.code ?? rg.json.join_code)).not.toBe(old);
  const dead = await call(request, await login(request, "sv.gioi"), "POST", "/courses/join/preview", { code: old });
  expect([404, 422], "mã cũ chết ngay").toContain(dead.status);
  // 6 mã sai liên tiếp ⇒ 429
  const codes: number[] = [];
  const tk = await login(request, "sv.nguyco");
  for (let i = 0; i < 8; i++) codes.push((await call(request, tk, "POST", "/courses/join", { code: `ZZZZZ${i}2`.slice(0, 7) })).status);
  expect(codes.includes(429), `mã trả về: ${codes}`).toBe(true);
  await ctx.close();
  const arch = await call(request, await login(request, "admin"), "POST", `/admin/courses/${S.cid}/archive`, {});
  expect([200, 204]).toContain(arch.status);
  const after = await call(request, await login(request, "sv.gioi"), "POST", "/courses/join/preview", { code: String(rg.json.code ?? rg.json.join_code) });
  expect([404, 409, 410, 422], "lớp đã lưu trữ: mã tham gia tắt").toContain(after.status);
});

test("@real Hôm nay (seed): SV chưa có lớp thấy ô nhập mã; SV có lớp ≤ 1 nút chính; GV: «N việc cần xử lý hôm nay»", async ({ browser }) => {
  for (const [u, check] of [["sv.moi", "code"], ["sv.kha", "student"], ["teacher", "staff"]] as const) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
    const p = await ctx.newPage();
    await uiLogin(p, u);
    await p.goto("/");
    if (check === "code") await expect(p.getByText(/mã (lớp|tham gia)/i).first()).toBeVisible();
    if (check === "student") {
      expect(await p.locator('main [data-variant="primary"]:visible').count(), "≤ 1 nút chính").toBeLessThanOrEqual(1);
      await expect(p.getByText(/Buổi \d+/).first()).toBeVisible();
    }
    if (check === "staff") await expect(p.getByRole("heading", { name: /^\d+ việc cần xử lý hôm nay$/ })).toBeVisible();
    await shot(p, `f14-today-${u}`);
    await ctx.close();
  }
});
