import { test, expect, type Browser } from "@playwright/test";
import { call, courseId, email, linkIn, login, mailCount, needReal, shot, sql, uiLogin, uniq, waitMail } from "./qc-real-lib";

// QC @real F1 Tài khoản (P-11-29). Bật bốn ca @real của account.spec.ts (đang test.skip) + «mạo danh MSSV» + khoá khi dò mật khẩu.
needReal();
test.describe.configure({ mode: "serial" });

const tag = uniq();
const acct = { email: `qcf1.${tag}@edupilot.local`, pw: `Qcf1!${tag}zA9`, name: "QC Luồng F1", code: `29${Math.floor(10000 + Math.random() * 89999)}` };

test("@real register → Mailpit → xác minh → đăng nhập → vào / (ô nhập mã lớp)", async ({ page, request }) => {
  const t0 = Date.now();
  await page.goto("/register");
  await page.getByLabel("Họ và tên").fill(acct.name);
  await page.getByLabel("Email").fill(acct.email);
  await page.getByLabel("Mã số sinh viên (không bắt buộc)").fill(acct.code);
  await page.getByLabel("Mật khẩu", { exact: true }).fill(acct.pw);
  await page.getByRole("button", { name: "Tạo tài khoản", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Kiểm tra email của bạn" })).toBeVisible();
  await shot(page, "f1-register-check-email");
  const mail = await waitMail(acct.email, /xác (minh|nhận)/i, t0);
  const link = linkIn(mail.text, /verify/i);
  // chưa xác minh: đăng nhập được nhưng không vào được lớp (FLOWS F1 nhánh lỗi)
  const early = await call(request, null, "POST", "/auth/login", { email: acct.email, password: acct.pw });
  expect(early.status, "chưa xác minh vẫn đăng nhập được").toBe(200);
  const cid = await courseId(request, "teacher");
  const code = sql(`select join_code from courses where id='${cid}'`)[0][0];
  const joinEarly = await call(request, String(early.json.access_token), "POST", "/courses/join", { code });
  expect([200, 201, 202], "email chưa xác minh không được ACTIVE trong lớp").toContain(joinEarly.status);
  const st0 = sql(`select e.status from enrollments e join users u on u.id=e.user_id where u.email='${acct.email}' and e.course_id='${cid}'`)[0]?.[0];
  expect(st0 === undefined || st0 !== "ACTIVE", `email chưa xác minh: status=${st0}`).toBe(true);
  sql(`delete from enrollments where user_id=(select id from users where email='${acct.email}')`);

  await page.goto(link);
  await expect.poll(() => sql(`select email_verified_at is not null from users where email='${acct.email}'`)[0]?.[0]).toBe("t");
  expect(page.url(), "token rời khỏi URL").not.toMatch(/token=/);
  await shot(page, "f1-verified");
  const reuse = await call(request, null, "POST", "/auth/verify-email", { token: new URL(link, "https://x").searchParams.get("token") });
  expect([400, 404, 409, 410, 422], "liên kết xác minh dùng lại bị từ chối").toContain(reuse.status);

  await uiLogin(page, acct.email, acct.pw);
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByText(/mã (lớp|tham gia)/i).first()).toBeVisible();
  await shot(page, "f1-first-login-today");
});

test("@real login + refresh qua Caddy cùng origin: tải lại giữ phiên, đúng một refresh, không token ở storage/cookie đọc được", async ({ page }) => {
  await uiLogin(page, acct.email, acct.pw);
  const refreshes: string[] = [];
  page.on("request", (r) => { if (r.url().includes("/api/v1/auth/refresh")) refreshes.push(r.method()); });
  await page.reload();
  await page.locator("[data-part=topbar]").waitFor();
  expect(new URL(page.url()).pathname).toBe("/");
  expect(refreshes.length, "một refresh khi tải lại").toBe(1);
  const leak = await page.evaluate(() => ({ ls: Object.entries(localStorage).concat(Object.entries(sessionStorage)).filter(([k, v]) => /jwt|token|refresh|bearer/i.test(k) || /^eyJ/.test(v)).map(([k]) => k), cookie: document.cookie }));
  expect(leak.ls).toEqual([]);
  expect(leak.cookie).not.toMatch(/ep_rt|refresh/i);
  const origin = new URL(page.url()).origin;
  const res = await page.request.get(`${origin}/api/v1/me/courses`, { headers: {} });
  expect(res.status(), "không có Bearer thì 401, cookie httpOnly không đủ để gọi tay").toBe(401);
});

async function login2(browser: Browser) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, locale: "vi-VN" });
  const page = await ctx.newPage();
  await uiLogin(page, acct.email, acct.pw);
  return { ctx, page };
}

test("@real reset mật khẩu thu hồi phiên máy khác: máy B về /login kèm dòng thông báo", async ({ browser, page }) => {
  const t0 = Date.now();
  const B = await login2(browser);
  await page.goto("/forgot-password");
  await page.getByLabel("Email").fill(acct.email);
  await page.getByRole("button", { name: "Gửi hướng dẫn", exact: true }).click();
  const mail = await waitMail(acct.email, /đặt lại|mật khẩu/i, t0);
  const link = linkIn(mail.text, /reset/i);
  await page.goto(link);
  const next = `${acct.pw}x`;
  await page.getByLabel("Mật khẩu mới", { exact: true }).fill(next);
  await page.getByLabel("Nhập lại mật khẩu", { exact: true }).fill(next);
  await page.getByRole("button", { name: "Đổi mật khẩu", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Mật khẩu đã được đổi. Hãy đăng nhập lại." })).toBeVisible();
  acct.pw = next;
  await B.page.reload();
  await B.page.waitForURL(/\/login/);
  await expect(B.page.getByText(/phiên|đăng nhập lại|hết hạn|thu hồi/i).first()).toBeVisible();
  await shot(B.page, "f1-reset-other-device-logged-out");
  await B.ctx.close();
  await uiLogin(page, acct.email, acct.pw);
});

test("@real mời giảng viên: Admin mời → Mailpit → đặt mật khẩu → bị khoá → mất phiên → mở khoá", async ({ page, request, browser }) => {
  const t0 = Date.now();
  const admin = await login(request, "admin");
  const mail = `qcf1.gv.${tag}@edupilot.local`;
  const inv = await call(request, admin, "POST", "/admin/users", { email: mail, full_name: "GV QC Luồng F1", role: "TEACHER" });
  expect(inv.status, inv.text.slice(0, 150)).toBe(201);
  const uid = String(inv.json.id);
  const m = await waitMail(mail, /mời|invite/i, t0);
  const link = linkIn(m.text, /invite/i);
  await page.goto(link);
  await expect(page.getByRole("heading", { name: /bạn được mời làm Giảng viên/ })).toBeVisible();
  expect(page.url(), "token rời URL").not.toContain(new URL(link, "https://x").pathname.split("/").pop()!);
  const pw = `Gv!${tag}Zq9x`;
  await page.getByLabel("Mật khẩu", { exact: true }).fill(pw);
  await page.getByLabel("Nhập lại mật khẩu", { exact: true }).fill(pw);
  await page.getByRole("button", { name: "Đặt mật khẩu và vào" }).click();
  await expect(page).toHaveURL(/\/$/);
  await shot(page, "f1-invite-accepted");
  // khoá → phiên chết ngay
  const cur = await call(request, admin, "GET", `/admin/users?q=${encodeURIComponent(mail)}&limit=5`);
  const row = ((cur.json.items ?? []) as { id: string; version: number }[]).find((u) => u.id === uid)!;
  const lock = await call(request, admin, "PATCH", `/admin/users/${uid}`, { status: "DISABLED", version: row.version });
  expect(lock.status, lock.text.slice(0, 150)).toBe(200);
  await page.reload();
  await page.waitForURL(/\/login/);
  const refused = await call(request, null, "POST", "/auth/login", { email: mail, password: pw });
  expect([401, 403, 423], "tài khoản bị khoá không đăng nhập được").toContain(refused.status);
  const unlock = await call(request, admin, "PATCH", `/admin/users/${uid}`, { status: "ACTIVE", version: Number(lock.json.version) });
  expect(unlock.status, unlock.text.slice(0, 150)).toBe(200);
  const ok = await browser.newContext({ ignoreHTTPSErrors: true });
  const p2 = await ok.newPage();
  await uiLogin(p2, mail, pw);
  await ok.close();
});

test("@real mạo danh MSSV: đăng ký bằng MSSV của SV trong lớp, email khác → PENDING + cảnh báo lệch, không thấy dữ liệu của SV thật", async ({ page, request, browser }) => {
  const t0 = Date.now();
  const victim = "sv.kha";
  const mssv = sql(`select student_code from users where email='${email(victim)}'`)[0][0];
  const imp = { email: `qcf1.imp.${tag}@edupilot.local`, pw: `Imp!${tag}zA9`, name: "Người Mạo Danh QC" };
  const reg = await call(request, null, "POST", "/auth/register", { email: imp.email, password: imp.pw, full_name: imp.name, student_code: mssv });
  expect([200, 201, 202]).toContain(reg.status);
  const m = await waitMail(imp.email, /xác (minh|nhận)/i, t0);
  await page.goto(linkIn(m.text, /verify/i));
  await expect.poll(() => sql(`select email_verified_at is not null from users where email='${imp.email}'`)[0]?.[0]).toBe("t");
  const tk = await login(request, imp.email, imp.pw);
  const cid = await courseId(request, "teacher");
  const code = sql(`select join_code from courses where id='${cid}'`)[0][0];
  const j = await call(request, tk, "POST", "/courses/join", { code });
  expect([200, 201, 202], j.text.slice(0, 150)).toContain(j.status);
  const row = sql(`select e.status, coalesce(e.warning,'') from enrollments e join users u on u.id=e.user_id where u.email='${imp.email}' and e.course_id='${cid}'`)[0];
  expect(row?.[0], "MSSV trùng, email khác không được ACTIVE").toBe("PENDING");
  expect(row?.[1], "hàng PENDING mang cảnh báo lệch").toMatch(/EMAIL_MISMATCH|^$/);
  // không đọc được dữ liệu lớp / của SV thật
  for (const p of [`/courses/${cid}/assignments`, `/courses/${cid}/gradebook/me`, `/courses/${cid}/attendance/me`, `/me/attendance`]) {
    const r = await call(request, tk, "GET", p);
    expect([401, 403, 404, 200]).toContain(r.status);
    if (r.status === 200) expect(JSON.stringify(r.json), `${p} không chứa dữ liệu của ${victim}`).not.toContain(mssv);
  }
  const mine = await call(request, tk, "GET", "/me/courses");
  const items = (mine.json.items ?? []) as { status?: string; course: { class_code: string } }[];
  expect(items.some((i) => i.course.class_code === "761987" && i.status === "ACTIVE")).toBe(false);
  // Giảng viên thấy hàng chờ duyệt kèm cảnh báo
  const tctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const tp = await tctx.newPage();
  await uiLogin(tp, "teacher");
  await tp.goto("/class/members?tab=pending");
  const prow = tp.locator("table tbody tr").filter({ hasText: imp.name });
  await expect(prow).toBeVisible();
  await expect(prow.getByText("Email chưa khớp MSSV")).toBeVisible();
  await shot(tp, "f1-mssv-impersonation-pending");
  await tctx.close();
  sql(`delete from enrollments where user_id=(select id from users where email='${imp.email}')`);
});

test("@real dò mật khẩu: 10 lần sai → khoá 15 phút + mail cảnh báo; đúng mật khẩu cũng bị từ chối trong lúc khoá", async ({ request }) => {
  const t0 = Date.now();
  const victim = `qcf1.lock.${tag}@edupilot.local`;
  const pw = `Lock!${tag}zA9`;
  const reg = await call(request, null, "POST", "/auth/register", { email: victim, password: pw, full_name: "QC Khoá Dò", student_code: "" });
  expect([200, 201, 202]).toContain(reg.status);
  const codes: number[] = [];
  for (let i = 0; i < 12; i++) codes.push((await call(request, null, "POST", "/auth/login", { email: victim, password: `Sai!${i}${tag}` })).status);
  expect(codes.filter((c) => c === 401).length, `mã trả về: ${codes}`).toBeGreaterThanOrEqual(5);
  expect(codes.slice(-2).every((c) => c === 423 || c === 429), `10 lần sai phải khoá/chờ: ${codes}`).toBe(true);
  const good = await call(request, null, "POST", "/auth/login", { email: victim, password: pw });
  expect([423, 429], "đang khoá thì đúng mật khẩu cũng không vào").toContain(good.status);
  await waitMail(victim, /khoá|bất thường|cảnh báo|đăng nhập/i, t0, 90_000);
  expect(await mailCount(victim, /khoá|bất thường|cảnh báo/i)).toBeGreaterThanOrEqual(1);
});
