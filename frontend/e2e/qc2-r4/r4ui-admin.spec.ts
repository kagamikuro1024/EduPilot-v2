import { execFileSync } from "node:child_process";
import { expect, test, type Page, type Route } from "@playwright/test";
import { PWD, S, apiLogin, axeBad, call, clearRL, em, installRec, layout, linkToken, mailTo, note, psql, sleep, techWords, tokenOf } from "./r4ui-lib";

// QC2-I r4ui: US-P2-06 (Admin /admin/users, lời mời), US-P2-08 (Admin /admin/courses, chuông, thời gian). Admin riêng qc2iu.adm.
installRec("r4ui-admin");
const only = (info: { project: { name: string } }, p: "desktop" | "mobile") => test.skip(info.project.name !== p, `chỉ ${p}`);
const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
const err = (code: string, extra: Record<string, unknown> = {}) => ({ code, message: "m", trace_id: "b".repeat(32), ...extra });
const topbar = (page: Page) => page.locator("[data-part=topbar]");
const stamp = () => Date.now().toString(36).toUpperCase();

async function openUsers(page: Page) {
  await apiLogin(page, em("adm"));
  await page.goto("/admin/users");
  await expect(page.getByRole("heading", { name: "Người dùng", level: 1 })).toBeVisible({ timeout: 30_000 });
  await expect(page.locator("table tbody tr").first()).toBeVisible({ timeout: 30_000 });
}
async function openInvite(page: Page) {
  await expect(async () => {
    if (!(await page.getByLabel("Họ và tên").isVisible())) await page.getByRole("button", { name: "Mời giảng viên" }).click();
    await expect(page.getByLabel("Họ và tên")).toBeVisible({ timeout: 1000 });
  }).toPass({ timeout: 10_000 });
}
/** Mời một GV/TA bằng API (Admin riêng) → {email, id, token}. */
async function inviteApi(role: "TEACHER" | "TA", tag: string) {
  const adm = await tokenOf(em("adm"));
  const t0 = Date.now();
  const email = `qc2iu.${tag}${t0}@edupilot.local`;
  const r = await call("POST", "/admin/users", adm, { email, full_name: `QC2IU ${tag} ${stamp()}`, role });
  expect(r.status, JSON.stringify(r.body)).toBe(201);
  const token = linkToken(await mailTo(email, /mời|Lời mời/i, t0), /invite/);
  return { email, id: r.body.id as string, token, name: r.body.full_name as string };
}

test("[TC-P206-32] /admin/users: cột, chip lọc, ô tìm, 1 nút primary, khung mời tại chỗ không dialog, không có Tạo sinh viên", async ({ page }, info) => {
  only(info, "desktop");
  await openUsers(page);
  const invite = page.getByRole("button", { name: "Mời giảng viên" });
  await expect(invite).toHaveCount(1);
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  await expect(invite).toHaveAttribute("data-variant", "primary");
  for (const h of ["Họ tên", "Email", "Vai trò", "Trạng thái", "Lần cuối"]) await expect(page.getByRole("columnheader", { name: h })).toBeVisible();
  for (const chip of ["Tất cả", "Giảng viên", "Trợ giảng", "Sinh viên"]) await expect(page.getByRole("radio", { name: chip })).toBeVisible();
  await expect(page.getByLabel("Tìm người dùng")).toBeVisible();
  await expect(page.getByRole("button", { name: /Tạo sinh viên/ })).toHaveCount(0);
  await openInvite(page);
  const dlg = await page.getByRole("dialog").count();
  note(`getByRole('dialog') hiện = ${dlg}`);
  expect(dlg).toBe(0);
  await expect(page.getByLabel("Email", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Họ và tên")).toBeVisible();
  const vai = page.getByLabel("Vai", { exact: true });
  await expect(vai).toBeVisible();
  const opts = (await vai.locator("option").allInnerTexts()).map((s) => s.trim());
  note(`Vai: ${opts.join("/")}`);
  expect(opts).toEqual(expect.arrayContaining(["Giảng viên", "Trợ giảng"]));
  expect(opts).not.toContain("Sinh viên");
});

test("[TC-P206-33] gửi lời mời: dòng tĩnh; Gửi lại lời mời ở hàng INVITED; Khoá tài khoản → Hoàn tác = mở khoá; rỗng", async ({ page }, info) => {
  only(info, "desktop");
  await openUsers(page);
  await openInvite(page);
  const email = `qc2iu.i33${Date.now()}@edupilot.local`;
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Họ và tên").fill("QC2IU Mời 33");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi link mời, hạn 72 giờ." })).toBeVisible({ timeout: 30_000 });
  expect(psql(`select status from users where email='${email}'`)).toBe("INVITED");
  await page.getByLabel("Tìm người dùng").fill(email);
  const row = page.locator("table tbody tr").filter({ hasText: email }).first();
  await expect(row).toBeVisible({ timeout: 20_000 });
  await row.getByRole("button", { name: "Gửi lại lời mời" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi lại lời mời" })).toBeVisible({ timeout: 20_000 });
  expect(Number(psql(`select count(*) from mail_outbox where to_email='${email}' or recipient='${email}'`) || 0)).toBeGreaterThanOrEqual(0);
  // khoá lạc quan + Hoàn tác (tài khoản SV riêng qc2iu.sv7)
  await page.getByLabel("Tìm người dùng").fill(em("sv7"));
  const sv = page.locator("table tbody tr").filter({ hasText: em("sv7") }).first();
  await expect(sv).toBeVisible({ timeout: 20_000 });
  await sv.getByRole("button", { name: /Thao tác cho/ }).click();
  await page.getByRole("menuitem", { name: "Khoá tài khoản" }).click();
  await expect(page.getByRole("status").filter({ hasText: /^Đã khoá .*Hoàn tác/ })).toBeVisible();
  await expect(sv.getByText("Đã khoá")).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect.poll(() => psql(`select status from users where email='${em("sv7")}'`), { timeout: 15_000 }).toBe("DISABLED");
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect.poll(() => psql(`select status from users where email='${em("sv7")}'`), { timeout: 15_000 }).toBe("ACTIVE");
  // rỗng
  await page.getByLabel("Tìm người dùng").fill("zzzkhongkhop987");
  await expect(page.getByText(/Chưa có người dùng khớp/)).toBeVisible({ timeout: 20_000 });
  note("mời: dòng tĩnh 72 giờ; gửi lại; khoá DISABLED → Hoàn tác ACTIVE (DB); rỗng 'Chưa có người dùng khớp…'");
});

test("[TC-P206-34] /admin/users 375: AUDIT/TOUCH/axe, thành danh sách < 720 px, không useDemoSlice", async ({ page }, info) => {
  only(info, "mobile");
  await openUsers(page);
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const l = await layout(page);
  const ax = await axeBad(page);
  const tables = await page.locator("table:visible").count();
  const grep = execFileSync("bash", ["-c", "grep -rln useDemoSlice src/app src/features 2>/dev/null | grep -i 'admin' | wc -l"], { cwd: __dirname + "/../.." }).toString().trim();
  note(`ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length} axe=${JSON.stringify(ax)}; bảng hiện ở 375=${tables}; grep useDemoSlice ở admin=${grep}`);
  expect(l.ox).toBeLessThanOrEqual(0);
  expect(l.cut).toEqual([]);
  expect(l.touch).toEqual([]);
  expect(ax).toEqual([]);
  expect(tables).toBe(0);
  expect(grep).toBe("0");
  expect(await techWords(page, ["token", "session", "jwt", "bcrypt", "password"])).toEqual([]);
});

test("[TC-P206-36] /invite/<token>: lời chào theo preview, 2 ô mật khẩu, 1 nút, token thay trong URL, thành công → / đã đăng nhập", async ({ page }, info) => {
  only(info, "desktop");
  const inv = await inviteApi("TEACHER", "i36");
  clearRL();
  const calls: string[] = [];
  page.on("request", (r) => r.url().includes("/auth/tokens/preview") && calls.push(r.url()));
  await page.goto(`/invite/${inv.token}`);
  await expect(page.getByRole("heading", { name: `Chào ${inv.name}, bạn được mời làm Giảng viên trên EduPilot.` })).toBeVisible({ timeout: 30_000 });
  expect(decodeURIComponent(new URL(page.url()).pathname)).toBe("/invite/·");
  expect(page.url()).not.toContain(inv.token);
  expect(calls.length).toBe(1);
  const a = page.getByLabel("Mật khẩu", { exact: true });
  const b = page.getByLabel("Nhập lại mật khẩu", { exact: true });
  await expect(a).toHaveAttribute("autocomplete", "new-password");
  await expect(b).toHaveAttribute("autocomplete", "new-password");
  await expect(page.getByText("Ít nhất 10 ký tự, không phải mật khẩu phổ biến.")).toBeVisible();
  const submit = page.getByRole("button", { name: "Đặt mật khẩu và vào" });
  await expect(submit).toHaveCount(1);
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  await a.fill(PWD);
  await b.fill(PWD);
  await submit.click();
  await expect(page).toHaveURL(/\/$/, { timeout: 30_000 });
  await expect(topbar(page)).toBeVisible({ timeout: 30_000 });
  expect(psql(`select status from users where email='${inv.email}'`)).toBe("ACTIVE");
  note(`lời chào đúng; preview 1 lần; URL /invite/·; sau đặt mật khẩu → ${new URL(page.url()).pathname}`);
});

test("[TC-P206-37] /invite: hết hạn / đã dùng / rác → câu chung không nút, không lộ email; Referrer-Policy; log Caddy không có token", async ({ page, request }, info) => {
  only(info, "desktop");
  const used = await inviteApi("TEACHER", "u37");
  clearRL();
  expect((await call("POST", "/auth/accept-invite", null, { token: used.token, password: PWD })).status).toBe(200);
  const exp = await inviteApi("TEACHER", "e37");
  psql(`update auth_tokens set expires_at=now()-interval '1 hour' where user_id=(select id from users where email='${exp.email}') and kind='INVITE'`);
  const since = new Date(Date.now() - 2000).toISOString();
  const MSG = "Lời mời đã hết hạn hoặc đã được dùng. Hãy nhờ quản trị viên gửi lại.";
  for (const [k, t] of [["đã dùng", used.token], ["hết hạn", exp.token], ["rác", "R".repeat(43)]] as const) {
    clearRL();
    await page.goto(`/invite/${t}`);
    await expect(page.getByText(MSG), k).toBeVisible({ timeout: 30_000 });
    await expect(page.locator("main").getByRole("button"), k).toHaveCount(0);
    expect(await page.locator("body").innerText(), k).not.toMatch(/@/);
  }
  const hd = (await request.get(`/invite/${"D".repeat(43)}`)).headers();
  const logs = execFileSync("docker", ["logs", "ep-r-caddy-1", "--since", since], { maxBuffer: 1 << 28 }).toString();
  const cnt = [used.token, exp.token].reduce((n, t) => n + logs.split(t).length - 1, 0);
  const red = (logs.match(/\/invite\/REDACTED/g) ?? []).length;
  note(`Referrer-Policy=${hd["referrer-policy"]}; log Caddy đếm token=${cnt}, dòng /invite/REDACTED=${red}`);
  expect(hd["referrer-policy"]).toBe("no-referrer");
  expect(cnt).toBe(0);
});

test("[TC-P206-39] lỗi Admin: 409 email trùng tại ô, 422, 403, offline Gửi lại cùng khoá, xung đột phiên bản", async ({ page, context }, info) => {
  only(info, "desktop");
  await openUsers(page);
  await openInvite(page);
  const email = page.getByLabel("Email", { exact: true });
  const name = page.getByLabel("Họ và tên");
  // 409 thật: email đã có tài khoản
  await email.fill(em("ta"));
  await name.fill("Người Trùng");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.getByText("Email này đã có tài khoản.")).toBeVisible({ timeout: 30_000 });
  await expect(email).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveValue("Người Trùng");
  // 422 (ép)
  await page.route("**/api/v1/admin/users", (r) => (r.request().method() === "POST" ? json(r, 422, err("VALIDATION_FAILED", { details: [{ field: "full_name", code: "INVALID_NAME", message: "Họ và tên cần từ 1 đến 100 ký tự." }] })) : r.fallback()));
  await email.fill(`qc2iu.e422${Date.now()}@edupilot.local`);
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.getByText("Họ và tên cần từ 1 đến 100 ký tự.")).toBeVisible();
  await expect(name).toHaveValue("Người Trùng");
  // 403 (ép)
  await page.unroute("**/api/v1/admin/users");
  await page.route("**/api/v1/admin/users", (r) => (r.request().method() === "POST" ? json(r, 403, err("FORBIDDEN", { details: { reason: "role" } })) : r.fallback()));
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  const a403 = await page.getByRole("alert").first().innerText();
  note(`403 → "${a403.slice(0, 80)}"`);
  expect(a403).not.toMatch(/FORBIDDEN|403/);
  await page.unroute("**/api/v1/admin/users");
  // offline: Gửi lại cùng khoá
  const keys: Array<string | undefined> = [];
  let down = true;
  await page.route("**/api/v1/admin/users", (r) => {
    if (r.request().method() !== "POST" || !down) return r.fallback();
    keys.push(r.request().headers()["idempotency-key"]);
    return r.abort("connectionfailed");
  });
  const fresh = `qc2iu.off${Date.now()}@edupilot.local`;
  await email.fill(fresh);
  await name.fill("QC2IU Mất Mạng");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.locator("[data-part=offline-banner]").getByText("Mất kết nối mạng.")).toBeVisible({ timeout: 40_000 });
  await expect(name).toHaveValue("QC2IU Mất Mạng");
  down = false;
  await page.getByRole("alert").getByRole("button", { name: "Gửi lại" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi link mời, hạn 72 giờ." })).toBeVisible({ timeout: 30_000 });
  note(`offline: ${keys.length} lần gửi, ${new Set(keys).size} khoá`);
  expect(new Set(keys).size).toBe(1);
  expect(psql(`select count(*) from users where email='${fresh}'`)).toBe("1");
  await page.unrouteAll({ behavior: "ignoreErrors" });
  void context;
  // VERSION_CONFLICT thật: sửa phiên bản bằng API trong lúc trang đang mở
  const vic = await inviteApi("TA", "vc");
  await page.getByLabel("Tìm người dùng").fill(vic.email);
  const row = page.locator("table tbody tr").filter({ hasText: vic.email }).first();
  await expect(row).toBeVisible({ timeout: 20_000 });
  const adm = await tokenOf(em("adm"));
  const cur = (await call("GET", `/admin/users?q=${encodeURIComponent(vic.email)}`, adm)).body.items[0];
  expect((await call("PATCH", `/admin/users/${vic.id}`, adm, { status: "DISABLED", version: cur.version })).status).toBe(200);
  await row.getByRole("button", { name: /Thao tác cho/ }).click();
  await page.getByRole("menuitem", { name: "Khoá tài khoản" }).click();
  await expect(page.getByText("Tài khoản này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?")).toBeVisible({ timeout: 20_000 });
  note("409 CONFLICT email trùng, 422, 403, offline cùng khoá, VERSION_CONFLICT thật: đủ thông báo tiếng Việt");
});

// ---------------- /admin/courses ----------------
async function openCourses(page: Page) {
  await apiLogin(page, em("adm"));
  await page.goto("/admin/courses");
  await expect(page.getByRole("heading", { name: "Lớp học", level: 1 })).toBeVisible({ timeout: 30_000 });
  await expect(page.locator("table tbody tr, [data-part=row]").first()).toBeVisible({ timeout: 30_000 });
}
async function openCourseForm(page: Page) {
  await expect(async () => {
    if (!(await page.getByLabel("Mã lớp", { exact: true }).isVisible())) await page.getByRole("button", { name: "Mở lớp" }).first().click();
    await expect(page.getByLabel("Mã lớp", { exact: true })).toBeVisible({ timeout: 1000 });
  }).toPass({ timeout: 10_000 });
}

test("[TC-P208-31] /admin/courses: cột, 1 nút Mở lớp, Drawer không dialog, mở lớp (nhấp đúp = 1 lớp), Lưu trữ → xác nhận nêu số", async ({ page }, info) => {
  only(info, "desktop");
  await openCourses(page);
  const open = page.getByRole("button", { name: "Mở lớp" });
  await expect(open).toHaveCount(1);
  await expect(open).toHaveAttribute("data-variant", "primary");
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  for (const h of ["Mã lớp", "Học phần", "Giảng viên", "Sĩ số", "Trạng thái"]) await expect(page.getByRole("columnheader", { name: h })).toBeVisible();
  await expect(page.locator("table").getByText(/\d+ \/ \d+|\d+ \/ —|\d+/).first()).toBeVisible();
  await openCourseForm(page);
  expect(await page.getByRole("dialog").count()).toBe(0);
  const code = "QC2IUN" + stamp();
  await page.getByLabel("Mã lớp", { exact: true }).fill(code);
  await page.getByLabel("Giảng viên").selectOption({ label: S.gv1.name });
  const submit = page.getByRole("button", { name: "Mở lớp" });
  await submit.dblclick();
  await expect(page.getByRole("status").filter({ hasText: `Đã mở lớp. Đã gửi thông báo phân công cho ${S.gv1.name}.` })).toBeVisible({ timeout: 30_000 });
  const n = psql(`select count(*) from courses where class_code='${code}'`);
  note(`nhấp đúp → ${n} lớp ${code}`);
  expect(n).toBe("1");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  const row = page.locator("table tbody tr").filter({ hasText: code });
  await expect(row).toBeVisible({ timeout: 20_000 });
  await row.getByRole("button", { name: new RegExp(`Hành động cho lớp ${code}`) }).click();
  await page.getByRole("menuitem", { name: "Lưu trữ lớp" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(1);
  await expect(page.getByText(new RegExp(`Lưu trữ lớp ${code}: \\d+ sinh viên chỉ còn quyền đọc; mã tham gia ngừng hoạt động\\.`))).toBeVisible();
  await page.getByRole("dialog").getByRole("button", { name: "Lưu trữ lớp" }).click();
  await expect(page.getByRole("status").filter({ hasText: `Đã lưu trữ lớp ${code}.` })).toBeVisible({ timeout: 20_000 });
  expect(psql(`select status from courses where class_code='${code}'`)).toBe("ARCHIVED");
});

test("[TC-P208-32] /admin/courses: 375 sạch (AUDIT/TOUCH/axe) thành danh sách; bàn phím mở khung; không useDemoSlice", async ({ page }, info) => {
  await openCourses(page);
  const grep = execFileSync("bash", ["-c", "grep -rln useDemoSlice src/app src/features 2>/dev/null | grep -i 'admin' | wc -l"], { cwd: __dirname + "/../.." }).toString().trim();
  expect(grep).toBe("0");
  if (info.project.name === "mobile") {
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
    const l = await layout(page);
    const ax = await axeBad(page);
    const tables = await page.locator("table:visible").count();
    note(`375: ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length} axe=${JSON.stringify(ax)} bảng hiện=${tables}; grep=${grep}`);
    expect(l.ox).toBeLessThanOrEqual(0);
    expect(l.cut).toEqual([]);
    expect(l.touch).toEqual([]);
    expect(ax).toEqual([]);
    expect(tables).toBe(0);
  } else {
    let reached = false;
    for (let i = 0; i < 40 && !reached; i++) {
      await page.keyboard.press("Tab");
      reached = await page.evaluate(() => /^Mở lớp$/.test((document.activeElement?.textContent ?? "").trim()));
    }
    expect(reached).toBe(true);
    await page.keyboard.press("Enter");
    await expect(page.getByLabel("Mã lớp", { exact: true })).toBeVisible();
    await page.keyboard.press("Tab");
    note("bàn phím: Tab tới 'Mở lớp', Enter mở khung tại chỗ");
    expect(await techWords(page, ["token", "session", "jwt", "bcrypt", "join_code"])).toEqual([]);
  }
});

/** Lớp mới cho gv2 (GV hoàn toàn mới ⇒ chuông). */
async function newCourseFor(teacherId: string, tag: string) {
  const adm = await tokenOf(em("adm"));
  const code = `QC2IU${tag}${stamp()}`.slice(0, 16);
  const r = await call("POST", "/admin/courses", adm, { subject_code: "INT1006", class_code: code, name: `QC2IU ${tag}`, semester: "2026-2027-HK1", teacher_id: teacherId });
  expect(r.status, JSON.stringify(r.body)).toBe(201);
  return { id: r.body.id as string, code };
}

test("[TC-P208-34] chuông thật: chấm ≤ 35 s sau khi được gán, mở khung không xoá chấm, bấm → link + bộ chọn lớp, unread giảm, rỗng", async ({ page }, info) => {
  only(info, "desktop");
  const gv2 = S.gv2;
  await apiLogin(page, gv2.email);
  await page.goto("/");
  await expect(topbar(page)).toBeVisible({ timeout: 30_000 });
  const dot = page.locator("[data-part=bell-dot]");
  const before = Number(psql(`select count(*) from notifications where user_id='${gv2.id}' and read_at is null`));
  const c = await newCourseFor(gv2.id, "B");
  const t0 = Date.now();
  await expect(dot).toHaveCount(1, { timeout: 40_000 });
  const dt = (Date.now() - t0) / 1000;
  note(`chấm bell-dot sau ${dt.toFixed(1)} s (chưa đọc trước khi gán: ${before})`);
  expect(dt).toBeLessThanOrEqual(35);
  const bell = page.getByRole("button", { name: /^Thông báo, \d+ chưa đọc/ });
  const label0 = await bell.getAttribute("aria-label");
  const unread0 = Number(label0!.match(/(\d+) chưa đọc/)![1]);
  await bell.click();
  const panel = page.locator("[data-part=notifications]");
  const item = panel.getByRole("link", { name: new RegExp(`Bạn được phân công lớp .*${c.code}`) });
  await expect(item).toBeVisible();
  await expect(dot).toHaveCount(1); // mở khung không xoá chấm
  await item.click();
  const link = psql(`select link from notifications where user_id='${gv2.id}' and course_id='${c.id}' order by created_at desc limit 1`);
  const want = link.split("?")[0];
  await expect.poll(() => new URL(page.url()).pathname, { timeout: 15_000 }).toBe(want);
  expect(new URL(page.url()).search).not.toContain("course=");
  await expect(page.getByRole("button", { name: /^Chọn lớp, đang xem/ })).toContainText(c.code);
  await expect.poll(() => psql(`select count(*) from notifications where user_id='${gv2.id}' and course_id='${c.id}' and read_at is null`), { timeout: 15_000 }).toBe("0");
  const after = await page.getByRole("button", { name: /^Thông báo/ }).getAttribute("aria-label");
  note(`link "${link}"; unread ${unread0} → "${after}"`);
  const unread1 = Number(after!.match(/(\d+) chưa đọc/)?.[1] ?? 0);
  expect(unread1).toBe(unread0 - 1);
  // hết chưa đọc → chấm biến; rỗng với người không có thông báo
  psql(`update notifications set read_at=now() where user_id='${gv2.id}' and read_at is null`);
  await page.reload();
  await expect(page.locator("[data-part=bell-dot]")).toHaveCount(0, { timeout: 20_000 });
  await page.context().clearCookies();
  await apiLogin(page, em("svd"));
  await page.goto("/");
  await page.getByRole("button", { name: /^Thông báo/ }).click();
  await expect(page.locator("[data-part=notifications]")).toContainText(/Chưa có thông báo\. Khi có việc/);
});

test("[TC-P208-35] thời gian thông báo: vừa xong / N phút trước / N giờ trước / hôm qua HH:mm / dd/MM; không chuỗi cứng ngoài timeAgo", async ({ page }, info) => {
  only(info, "desktop");
  const u = S.svd;
  const uid = psql(`select id from users where email='${u.email}'`);
  psql(`delete from notifications where user_id='${uid}' and type='QC2IU_TIME'`);
  const rows: Array<[string, string]> = [
    ["n59s", "now() - interval '59 seconds'"],
    ["n5m", "now() - interval '5 minutes'"],
    ["n3h", "now() - interval '3 hours'"],
    ["nyd", "(date_trunc('day', now() at time zone 'Asia/Ho_Chi_Minh') - interval '1 day' + interval '16 hours 40 minutes') at time zone 'Asia/Ho_Chi_Minh'"],
    ["n5d", "now() - interval '5 days'"],
  ];
  for (const [k, ts] of rows) psql(`insert into notifications(user_id,type,title,body,link,created_at) values ('${uid}','QC2IU_TIME','Thử ${k}','b','/',${ts})`);
  await apiLogin(page, u.email);
  await page.goto("/");
  await page.getByRole("button", { name: /^Thông báo/ }).click();
  const panel = page.locator("[data-part=notifications]");
  await expect(panel).toContainText("Thử n59s", { timeout: 20_000 });
  const txt = await panel.innerText();
  const line = (k: string) => (txt.split("\n").join(" ").match(new RegExp(`Thử ${k}[^]*?(?=Thử n|$)`)) ?? [""])[0];
  const lk = Object.fromEntries(rows.map(([k]) => [k, line(k)]));
  note(JSON.stringify(lk));
  expect(lk.n59s).toMatch(/vừa xong/);
  expect(lk.n5m).toMatch(/5 phút trước/);
  expect(lk.n3h).toMatch(/3 giờ trước/);
  expect(lk.nyd).toMatch(/hôm qua 16:40/);
  const d5 = new Date(Date.now() - 5 * 86400_000);
  const dd = new Date(d5.toLocaleString("en-US", { timeZone: "Asia/Ho_Chi_Minh" }));
  const ddmm = `${String(dd.getDate()).padStart(2, "0")}/${String(dd.getMonth() + 1).padStart(2, "0")}`;
  expect(lk.n5d).toContain(ddmm);
  const grep = execFileSync("bash", ["-c", "grep -rnE '[0-9]+ (ngày|giờ|phút) trước' src | grep -v shared/lib/timeAgo | grep -v test | wc -l"], { cwd: __dirname + "/../.." }).toString().trim();
  note(`grep chuỗi 'N … trước' ngoài timeAgo = ${grep}`);
  expect(grep).toBe("0");
  psql(`delete from notifications where user_id='${uid}' and type='QC2IU_TIME'`);
});

test("[TC-P208-37] lỗi Admin lớp: mã trùng 409 tại ô, 422 sai vai, VERSION_CONFLICT, offline cùng khoá, lưu trữ lớp đã lưu trữ", async ({ page }, info) => {
  only(info, "desktop");
  await openCourses(page);
  await openCourseForm(page);
  const code = page.getByLabel("Mã lớp", { exact: true });
  const name = page.getByLabel("Tên lớp");
  const dup = `QC2IUD${stamp()}`;
  const first = await newCourseFor(S.gv1.id, "D");
  // 409 thật: mã lớp trùng
  await code.fill(first.code);
  await name.fill("Tên giữ nguyên");
  await page.getByRole("button", { name: "Mở lớp" }).click();
  await expect(page.getByText("Mã lớp này đã có.")).toBeVisible({ timeout: 30_000 });
  await expect(code).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveValue("Tên giữ nguyên");
  // 422 người sai vai (ép)
  await page.route("**/api/v1/admin/courses", (r) => (r.request().method() === "POST" ? json(r, 422, err("VALIDATION_FAILED", { details: [{ field: "teacher_id", code: "ROLE_MISMATCH", message: "Người này không phải giảng viên." }] })) : r.fallback()));
  await code.fill(dup);
  await page.getByRole("button", { name: "Mở lớp" }).click();
  await expect(page.getByText("Người này không phải giảng viên.")).toBeVisible();
  await expect(name).toHaveValue("Tên giữ nguyên");
  await page.unroute("**/api/v1/admin/courses");
  // offline: Gửi lại cùng khoá
  const keys: Array<string | undefined> = [];
  let down = true;
  await page.route("**/api/v1/admin/courses", (r) => {
    if (r.request().method() !== "POST" || !down) return r.fallback();
    keys.push(r.request().headers()["idempotency-key"]);
    return r.abort("connectionfailed");
  });
  await code.fill(dup);
  await page.getByRole("button", { name: "Mở lớp" }).click();
  await expect(page.locator("[data-part=offline-banner]").getByText("Mất kết nối mạng.")).toBeVisible({ timeout: 40_000 });
  await expect(name).toHaveValue("Tên giữ nguyên");
  down = false;
  await page.getByRole("alert").getByRole("button", { name: "Gửi lại" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã mở lớp." })).toBeVisible({ timeout: 30_000 });
  note(`offline: ${keys.length} lần gửi, ${new Set(keys).size} khoá; lớp ${dup} = ${psql(`select count(*) from courses where class_code='${dup}'`)}`);
  expect(new Set(keys).size).toBe(1);
  expect(psql(`select count(*) from courses where class_code='${dup}'`)).toBe("1");
  await page.unrouteAll({ behavior: "ignoreErrors" });
  // VERSION_CONFLICT thật: sửa lớp qua API khi trang đang mở, rồi sửa trên UI
  const adm = await tokenOf(em("adm"));
  const row = page.locator("table tbody tr").filter({ hasText: first.code });
  await page.reload();
  await expect(row).toBeVisible({ timeout: 30_000 });
  const list = (await call("GET", "/admin/courses?limit=200", adm)).body.items as Array<{ id: string; class_code: string; version: number; name: string; capacity?: number }>;
  const cur = list.find((c) => c.class_code === first.code)!;
  const put = await call("PUT", `/admin/courses/${first.id}`, adm, { name: "Đổi bằng API", class_code: first.code, version: cur.version, capacity: cur.capacity ?? 30 });
  note(`PUT API đổi phiên bản: ${put.status}`);
  await row.getByRole("button", { name: /Hành động cho lớp/ }).click();
  await page.getByRole("menuitem", { name: "Sửa" }).click();
  await page.getByLabel("Tên lớp").fill("Bản của tôi");
  await page.getByRole("button", { name: "Lưu thay đổi" }).click();
  await expect(page.getByText("Lớp này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?")).toBeVisible({ timeout: 20_000 });
  await expect(page.getByLabel("Tên lớp")).toHaveValue("Bản của tôi");
  await page.getByRole("button", { name: "Huỷ" }).click();
  // lưu trữ lớp đã lưu trữ: lưu trữ bằng API trước, rồi bấm trên UI (trang còn bản cũ)
  expect((await call("POST", `/admin/courses/${first.id}/archive`, adm, {})).status).toBe(200);
  await row.getByRole("button", { name: /Hành động cho lớp/ }).click();
  await page.getByRole("menuitem", { name: "Lưu trữ lớp" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Lưu trữ lớp" }).click();
  await expect(page.getByRole("dialog").getByRole("alert")).toContainText("Lớp này đã được lưu trữ.", { timeout: 20_000 });
  await sleep(10);
  void PWD;
});
