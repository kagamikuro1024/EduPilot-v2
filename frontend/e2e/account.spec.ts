import { expect, test, type Page, type Route } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { loadAudit, runAudit } from "./support/audit";
import { asDemo, sessionBody, type JwtRole } from "./support/session";

// US-P2-02: đăng nhập thật, làm mới, khôi phục khi tải lại. Backend được giả lập TỪNG TRANG bằng page.route (không dùng máy chủ giả
// dùng chung). Ca @real (gateway + Mailpit thật) không chạy ở CI.
const CORS = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (route: Route, status: number, body: unknown, extra: Record<string, string> = {}) =>
  route.fulfill({ status, contentType: "application/json", headers: { ...CORS, ...extra }, body: JSON.stringify(body) });
const err = (code: string, message = "m", extra: Record<string, unknown> = {}) => ({ code, message, trace_id: "b".repeat(32), ...extra });

type Mock = { refresh: number; login: number; logout: number; loginBodies: string[]; active: { role: JwtRole; email: string; fullName: string } | null };

/** Dựng phiên giả lập: chưa đăng nhập → refresh 401; đăng nhập đúng (mật khẩu `pw`) → phiên; sau đó refresh thành công cho tới khi đăng xuất. */
async function mockAuth(page: Page, who: { role: JwtRole; email: string; fullName?: string; password?: string; loggedIn?: boolean; delayLogin?: number; refreshDelay?: number }): Promise<Mock> {
  const m: Mock = { refresh: 0, login: 0, logout: 0, loginBodies: [], active: who.loggedIn ? { role: who.role, email: who.email, fullName: who.fullName ?? "Người Thử" } : null };
  const pw = who.password ?? "Edupilot#2026-demo";
  await page.route("**/api/v1/auth/refresh", async (route) => {
    m.refresh++;
    if (who.refreshDelay) await new Promise((r) => setTimeout(r, who.refreshDelay));
    if (!m.active) return json(route, 401, err("UNAUTHENTICATED"));
    return json(route, 200, sessionBody(m.active.role, { email: m.active.email, fullName: m.active.fullName }));
  });
  await page.route("**/api/v1/auth/login", async (route) => {
    m.login++;
    m.loginBodies.push(route.request().postData() ?? "");
    if (who.delayLogin) await new Promise((r) => setTimeout(r, who.delayLogin));
    const b = JSON.parse(route.request().postData() ?? "{}") as { email?: string; password?: string };
    if (b.email?.trim().toLowerCase() !== who.email || b.password !== pw) return json(route, 401, err("INVALID_CREDENTIALS", "Email hoặc mật khẩu không đúng."));
    m.active = { role: who.role, email: who.email, fullName: who.fullName ?? "Người Thử" };
    return json(route, 200, sessionBody(who.role, { email: who.email, fullName: who.fullName }), { "Set-Cookie": "ep_rt=x; Path=/api/v1/auth; HttpOnly" });
  });
  await page.route("**/api/v1/auth/logout", (route) => {
    m.logout++;
    m.active = null;
    return route.fulfill({ status: 204, headers: CORS });
  });
  return m;
}

const fillLogin = async (page: Page, email: string, pw = "Edupilot#2026-demo") => {
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Mật khẩu", { exact: true }).fill(pw);
  await page.getByRole("button", { name: "Đăng nhập", exact: true }).click();
};

test.beforeEach(async ({ context }) => {
  await context.clearCookies();
});

test("login page: nhãn, thuộc tính nhập, một nút primary, logo, liên kết, không từ kỹ thuật", async ({ page }, info) => {
  await mockAuth(page, { role: "STUDENT", email: "sv.gioi@edupilot.local" });
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Đăng nhập EduPilot" })).toBeVisible();
  const email = page.getByLabel("Email");
  await expect(email).toHaveAttribute("autocomplete", "username");
  await expect(email).toHaveAttribute("inputmode", "email");
  const pw = page.getByLabel("Mật khẩu", { exact: true });
  await expect(pw).toHaveAttribute("autocomplete", "current-password");
  await expect(pw).toHaveAttribute("type", "password");
  await page.getByRole("button", { name: "Hiện mật khẩu" }).click();
  await expect(pw).toHaveAttribute("type", "text");
  await page.getByRole("button", { name: "Ẩn mật khẩu" }).click();
  const submit = page.getByRole("button", { name: "Đăng nhập", exact: true });
  await expect(submit).toHaveCount(1);
  await expect(submit).toHaveAttribute("data-variant", "primary");
  await expect(page.getByRole("link", { name: "Quên mật khẩu?" })).toHaveAttribute("href", "/forgot-password");
  await expect(page.getByRole("link", { name: "Chưa có tài khoản? Đăng ký" })).toHaveAttribute("href", "/register");
  const h = await page.locator("[data-part=brand]").evaluate((e) => e.getBoundingClientRect().height);
  expect(Math.round(h)).toBe(info.project.name === "mobile" ? 32 : 40);
  const body = (await page.locator("main").innerText()).toLowerCase();
  for (const w of ["token", "session", "refresh", "jwt", "bcrypt", "oidc"]) expect(body).not.toContain(w);
  await expect(page.getByText("Cần giúp? Liên hệ giảng viên hoặc quản trị viên của bạn.")).toBeVisible();
});

test("login page: 375 px không tràn ngang, vùng chạm ≥ 44 px, AUDIT/TOUCH sạch", async ({ page }) => {
  await mockAuth(page, { role: "STUDENT", email: "sv.gioi@edupilot.local" });
  await page.setViewportSize({ width: 375, height: 800 });
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Đăng nhập EduPilot" })).toBeVisible();
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(a.cut).toEqual([]);
  const small = (await page.evaluate(TOUCH_SRC)) as unknown[];
  expect(small).toEqual([]);
});

test("login page: sai ⇒ dòng lỗi role=alert, giữ email, xoá mật khẩu; đang gửi chỉ một request", async ({ page }) => {
  const m = await mockAuth(page, { role: "STUDENT", email: "sv.gioi@edupilot.local", delayLogin: 400 });
  await page.goto("/login");
  await page.getByLabel("Email").fill("  Sv.Gioi@Edupilot.Local ");
  await page.getByLabel("Mật khẩu", { exact: true }).fill("sai-mat-khau-1");
  const submit = page.getByRole("button", { name: "Đăng nhập", exact: true });
  await submit.click();
  await expect(submit).toBeDisabled();
  await submit.click({ force: true, noWaitAfter: true }).catch(() => undefined); // đang gửi: không gửi lần hai
  await expect(page.getByRole("alert").filter({ hasText: "Email hoặc mật khẩu không đúng." })).toBeVisible();
  expect(m.login).toBe(1);
  await expect(page.getByLabel("Email")).toHaveValue("Sv.Gioi@Edupilot.Local");
  await expect(page.getByLabel("Mật khẩu", { exact: true })).toHaveValue("");
  await expect(page).toHaveURL(/\/login$/);
});

for (const code of ["LOGIN_THROTTLED", "RATE_LIMITED"]) {
  test(`login throttled (${code}): đếm ngược mỗi giây, nút khoá kèm lý do rồi mở lại, giữ chữ đã gõ, không lộ mã`, async ({ page }) => {
    await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
    let calls = 0;
    await page.route("**/api/v1/auth/login", (r) => {
      calls++;
      return calls === 1 ? json(r, 429, err(code, "m", { retry_after: 3 }), { "Retry-After": "3" }) : json(r, 401, err("INVALID_CREDENTIALS"));
    });
    await page.goto("/login");
    await fillLogin(page, "Khong.Biet@Sv.Example", "mat-khau-dang-go");
    const line = page.getByRole("status").filter({ hasText: /^Bạn đã thử quá nhiều lần\. Thử lại sau 00:0[123]\.$/ });
    await expect(line).toBeVisible();
    const submit = page.getByRole("button", { name: "Đăng nhập", exact: true });
    await expect(submit).toBeDisabled();
    await expect(submit).toHaveAttribute("aria-describedby", "login-wait"); // lý do khoá
    await expect(page.getByLabel("Email")).toHaveValue("Khong.Biet@Sv.Example");
    await expect(page.getByLabel("Mật khẩu", { exact: true })).toHaveValue("mat-khau-dang-go"); // giữ nguyên, không buộc gõ lại
    await expect(page.getByRole("link", { name: "Quên mật khẩu?" })).toBeVisible();
    expect(await page.locator("body").innerText()).not.toMatch(/LOGIN_THROTTLED|RATE_LIMITED/);
    await expect(submit).toBeEnabled({ timeout: 4_000 }); // sau ~3 s
    await expect(line).toHaveCount(0);
    await submit.click();
    await expect(page.getByRole("alert").filter({ hasText: "Email hoặc mật khẩu không đúng." })).toBeVisible();
    expect(calls).toBe(2);
  });
}

test("login page: phiên bị thu hồi ⇒ dòng thông báo", async ({ page }) => {
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
  await page.goto("/login?revoked=refresh_reuse");
  await expect(page.getByText("Bạn đã bị đăng xuất. Hãy đăng nhập lại.")).toBeVisible();
  await page.goto("/login?revoked=password_changed");
  await expect(page.getByText("Bạn đã bị đăng xuất vì mật khẩu của tài khoản vừa được đổi.")).toBeVisible();
});

test("open redirect: next lạ bị bỏ, location.origin không đổi", async ({ page }) => {
  for (const evil of ["//evil.example", "https://evil.example", "/\\evil.example", "javascript:alert(1)", "/%2F%2Fevil.example"]) {
    await mockAuth(page, { role: "STUDENT", email: "sv.gioi@edupilot.local" });
    await page.goto(`/login?next=${encodeURIComponent(evil)}`);
    await fillLogin(page, "sv.gioi@edupilot.local");
    await expect(page.locator("[data-part=topbar]")).toBeVisible();
    expect(new URL(page.url()).origin).toBe(BASE_URL);
    expect(new URL(page.url()).pathname).toBe("/");
    await page.unroute("**/api/v1/auth/**");
    await page.context().clearCookies();
  }
});

test("next hợp lệ: quay lại đúng trang sau đăng nhập; vào route cần phiên khi chưa đăng nhập ⇒ /login?next=", async ({ page }) => {
  await mockAuth(page, { role: "STUDENT", email: "sv.gioi@edupilot.local" });
  await page.goto("/threads");
  await expect(page).toHaveURL(/\/login\?next=%2Fthreads$/);
  await fillLogin(page, "sv.gioi@edupilot.local");
  await expect(page).toHaveURL(/\/threads$/);
});

test("no token in storage: không có JWT / refresh ở storage, cookie; tải lại ⇒ đúng một refresh, không nháy /login", async ({ page }) => {
  const m = await mockAuth(page, { role: "STUDENT", email: "sv.gioi@edupilot.local", refreshDelay: 150 });
  const visited: string[] = [];
  page.on("framenavigated", (f) => f === page.mainFrame() && visited.push(new URL(f.url()).pathname));
  await page.goto("/login");
  await fillLogin(page, "sv.gioi@edupilot.local");
  await expect(page.locator("[data-part=topbar]")).toBeVisible();
  const scan = () =>
    page.evaluate(() => {
      const bad = /eyJ[\w-]{10,}\.|^[A-Za-z0-9_-]{43}$/;
      const hits: string[] = [];
      for (const store of [localStorage, sessionStorage]) for (const [k, v] of Object.entries(store)) if (/token|jwt|refresh|access/i.test(k) || bad.test(v)) hits.push(`${k}=${v.slice(0, 20)}`);
      if (document.cookie.includes("ep_rt")) hits.push("cookie ep_rt");
      return hits;
    });
  expect(await scan()).toEqual([]);

  const before = m.refresh;
  visited.length = 0;
  await page.reload();
  await expect(page.locator("[data-part=topbar]")).toBeVisible();
  expect(m.refresh - before).toBe(1);
  expect(visited.filter((p) => p === "/login")).toEqual([]);
  expect(await scan()).toEqual([]);
});

test("nav per role after login: Sinh viên 7 · TA 12 · Giảng viên 15 · Admin 6 mục", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "thanh bên chỉ có ở desktop");
  const want: Array<[JwtRole, string, number]> = [["STUDENT", "sv.gioi@edupilot.local", 7], ["TA", "ta@edupilot.local", 12], ["TEACHER", "teacher@edupilot.local", 15], ["ADMIN", "admin@edupilot.local", 6]];
  for (const [role, email, n] of want) {
    await page.context().clearCookies();
    await page.unroute("**/api/v1/auth/**");
    await mockAuth(page, { role, email });
    await page.goto("/login");
    await fillLogin(page, email);
    await expect(page.locator("[data-part=sidebar] nav a")).toHaveCount(n);
    await page.getByRole("button", { name: /^Tài khoản: / }).click();
    await expect(page.getByText("Đổi vai")).toHaveCount(0);
    await page.keyboard.press("Escape");
  }
});

test("mock identity from session: sv.kha ⇒ Trần Thu Uyên; email lạ ⇒ tên thật; không đọc cookie ep_demo_*", async ({ page, context }) => {
  await context.addCookies([{ name: "ep_demo_person", value: "sv-1", url: BASE_URL }]); // không có ep_demo_role ⇒ không phải phiên mô phỏng
  await mockAuth(page, { role: "STUDENT", email: "sv.kha@edupilot.local", fullName: "Tên Thật Khác" });
  await page.goto("/login");
  await fillLogin(page, "sv.kha@edupilot.local");
  await expect(page.getByRole("button", { name: /^Tài khoản: Trần Thu Uyên/ })).toBeVisible();
  await page.unroute("**/api/v1/auth/**");
  await context.clearCookies();
  await mockAuth(page, { role: "STUDENT", email: "an.nguyen@sv.example", fullName: "Nguyễn Thị An", loggedIn: true });
  await page.goto("/");
  await expect(page.getByRole("button", { name: /^Tài khoản: Nguyễn Thị An/ })).toBeVisible();
});

test("two tabs refresh: hai tab cùng tải ⇒ refresh tuần tự, không chồng nhau", async ({ page, context }) => {
  let inflight = 0;
  let peak = 0;
  let total = 0;
  const handler = async (route: Route) => {
    total++;
    inflight++;
    peak = Math.max(peak, inflight);
    await new Promise((r) => setTimeout(r, 300));
    inflight--;
    return json(route, 200, sessionBody("STUDENT", { email: "sv.gioi@edupilot.local" }));
  };
  const other = await context.newPage();
  await page.route("**/api/v1/auth/refresh", handler);
  await other.route("**/api/v1/auth/refresh", handler);
  await Promise.all([page.goto("/"), other.goto("/")]);
  await expect(page.locator("[data-part=topbar]")).toBeVisible();
  await expect(other.locator("[data-part=topbar]")).toBeVisible();
  expect(total).toBeLessThanOrEqual(2);
  expect(peak).toBe(1);
});

test("refresh hỏng giữa chừng ⇒ /login?next=; phiên bị thu hồi ⇒ /login kèm thông báo", async ({ page }) => {
  let n = 0;
  await page.route("**/api/v1/auth/refresh", (r) => {
    n++;
    return n === 1 ? json(r, 200, sessionBody("ADMIN")) : json(r, 401, err("TOKEN_INVALID"));
  });
  await page.route("**/api/v1/admin/llm/**", (r) => json(r, 401, err("TOKEN_EXPIRED")));
  await page.goto("/settings/llm");
  await expect(page).toHaveURL(/\/login\?next=%2Fsettings%2Fllm$/);
  expect(n).toBe(2); // một lần lúc tải trang, đúng một lần khi 401 (không vòng lặp)

  await page.unroute("**/api/v1/admin/llm/**");
  await page.unroute("**/api/v1/auth/refresh");
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 200, sessionBody("ADMIN")));
  await page.route("**/api/v1/admin/llm/**", (r) => json(r, 401, err("SESSION_REVOKED", "m", { details: { reason: "password_changed" } })));
  await page.goto("/settings/llm");
  await expect(page).toHaveURL(/\/login\?.*revoked=password_changed/);
  await expect(page.getByText("Bạn đã bị đăng xuất vì mật khẩu của tài khoản vừa được đổi.")).toBeVisible();
});

const REGISTER_MSG = "Nếu email này dùng được, chúng tôi đã gửi thư xác nhận. Kiểm tra hộp thư của bạn.";

test("register page: nhãn, chú thích, autocomplete, một nút primary, lỗi từng ô, màn Kiểm tra email + đếm ngược gửi lại", async ({ page }) => {
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
  const posts: string[] = [];
  let resends = 0;
  await page.route("**/api/v1/auth/register", (r) => {
    const b = JSON.parse(r.request().postData() ?? "{}") as Record<string, string>;
    posts.push(JSON.stringify(b));
    if (b.full_name === "Lỗi") return json(r, 422, err("VALIDATION_FAILED", "m", { details: [{ field: "email", code: "INVALID_EMAIL", message: "Email chưa đúng dạng, ví dụ ten@truong.edu.vn." }, { field: "password", code: "PASSWORD_TOO_SHORT", message: "Mật khẩu cần ít nhất 10 ký tự và không quá 72 byte." }] }));
    return json(r, 202, { message: REGISTER_MSG });
  });
  await page.route("**/api/v1/auth/resend-verification", (r) => {
    resends++;
    return json(r, 202, { message: "ok" });
  });
  await page.goto("/register");
  await expect(page.getByRole("heading", { name: "Tạo tài khoản EduPilot" })).toBeVisible();
  await expect(page.getByLabel("Họ và tên")).toBeVisible();
  await expect(page.getByLabel("Email")).toHaveAttribute("autocomplete", "username");
  await expect(page.getByLabel("Mã số sinh viên (không bắt buộc)")).toBeVisible();
  await expect(page.getByText("Chỉ để giảng viên đối chiếu; không dùng để vào lớp.")).toBeVisible();
  await expect(page.getByLabel("Mật khẩu", { exact: true })).toHaveAttribute("autocomplete", "new-password");
  await expect(page.getByText("Ít nhất 10 ký tự, không phải mật khẩu phổ biến.")).toBeVisible();
  const submit = page.getByRole("button", { name: "Tạo tài khoản", exact: true });
  await expect(submit).toHaveCount(1);
  await expect(submit).toHaveAttribute("data-variant", "primary");

  await page.getByLabel("Họ và tên").fill("Lỗi");
  await page.getByLabel("Email").fill("sai");
  await page.getByLabel("Mật khẩu", { exact: true }).fill("ngan");
  await submit.click();
  await expect(page.getByLabel("Email")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByText("Email chưa đúng dạng, ví dụ ten@truong.edu.vn.")).toBeVisible();
  await expect(page.getByLabel("Mật khẩu", { exact: true })).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Họ và tên")).toHaveValue("Lỗi"); // không mất chữ đã gõ

  await page.getByLabel("Họ và tên").fill("Trần Thu Uyên");
  await page.getByLabel("Email").fill("uyen@sv.example");
  await page.getByLabel("Mã số sinh viên (không bắt buộc)").fill("20229002");
  await page.getByLabel("Mật khẩu", { exact: true }).fill("Edupilot#2026-demo");
  await submit.click();
  await expect(page.getByRole("heading", { name: "Kiểm tra email của bạn" })).toBeVisible();
  await expect(page.getByText("Nếu email này dùng được, chúng tôi đã gửi thư xác nhận.")).toBeVisible();
  expect(JSON.parse(posts.at(-1) ?? "{}")).toEqual({ email: "uyen@sv.example", password: "Edupilot#2026-demo", full_name: "Trần Thu Uyên", student_code: "20229002" });
  const resend = page.getByRole("button", { name: /^Gửi lại thư/ });
  await expect(resend).toBeDisabled();
  await expect(resend).toContainText(/\(\d+ giây\)/);
  expect(resends).toBe(0);
  const body = (await page.locator("main").innerText()).toLowerCase();
  for (const w of ["token", "session", "refresh", "jwt", "bcrypt"]) expect(body).not.toContain(w);
});

test("register page: 375 px không tràn ngang, vùng chạm ≥ 44 px", async ({ page }) => {
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
  await page.setViewportSize({ width: 375, height: 800 });
  await page.goto("/register");
  await expect(page.getByRole("heading", { name: "Tạo tài khoản EduPilot" })).toBeVisible();
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(a.cut).toEqual([]);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("verify page: xác minh tự động đúng một lần, token rời khỏi URL; thành công / đã dùng / hết hạn + gửi lại", async ({ page, request }) => {
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
  let calls = 0;
  let mode: "ok" | "used" | "expired" = "ok";
  const tokens: string[] = [];
  await page.route("**/api/v1/auth/verify-email", async (r) => {
    calls++;
    tokens.push((JSON.parse(r.request().postData() ?? "{}") as { token: string }).token);
    await new Promise((res) => setTimeout(res, 150));
    if (mode === "ok") return json(r, 200, { status: "verified" });
    return json(r, 410, err("LINK_INVALID", "m", { details: { reason: mode } }));
  });
  let resent = "";
  await page.route("**/api/v1/auth/resend-verification", (r) => {
    resent = r.request().postData() ?? "";
    return json(r, 202, { message: "ok" });
  });
  const T = "A".repeat(43);

  await page.goto(`/verify-email?token=${T}`);
  await expect(page.getByText("Đang xác minh email…")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Email đã được xác minh." })).toBeVisible();
  await expect(page.getByRole("link", { name: "Đăng nhập" })).toHaveAttribute("href", "/login");
  expect(page.url()).not.toContain("token");
  expect(await page.evaluate(() => location.search)).toBe("");
  expect(calls).toBe(1);
  expect(tokens).toEqual([T]);

  mode = "used";
  await page.goto(`/verify-email?token=${T}`);
  await expect(page.getByText("Liên kết này đã được dùng. Nếu bạn đã xác minh, hãy đăng nhập.")).toBeVisible();

  mode = "expired";
  await page.goto(`/verify-email?token=${T}`);
  await expect(page.getByRole("heading", { name: "Liên kết đã hết hạn." })).toBeVisible();
  await page.getByLabel("Email").fill("uyen@sv.example");
  await page.getByRole("button", { name: "Gửi lại thư" }).click();
  await expect(page.getByText("Nếu email này cần xác minh, chúng tôi đã gửi lại thư.")).toBeVisible();
  expect(JSON.parse(resent)).toEqual({ email: "uyen@sv.example" });
  await expect(page.getByRole("button", { name: /^Gửi lại thư/ })).toBeDisabled();

  await page.goto("/verify-email"); // thiếu token ⇒ trạng thái "không dùng được", không gọi máy chủ
  await expect(page.getByRole("heading", { name: "Liên kết không dùng được" })).toBeVisible();
  expect(calls).toBe(3);

  const res = await request.get("/verify-email");
  expect(res.headers()["referrer-policy"]).toBe("no-referrer");
  expect(res.headers()["cache-control"]).toContain("no-store");
});

const FORGOT_MSG = "Nếu email này có tài khoản, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu.";

test("forgot page: một ô email, một nút Gửi hướng dẫn, câu chung bất kể email, nút gửi lại khoá đếm ngược", async ({ page }) => {
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
  const bodies: string[] = [];
  await page.route("**/api/v1/auth/forgot-password", (r) => {
    bodies.push(r.request().postData() ?? "");
    return json(r, 202, { message: FORGOT_MSG });
  });
  await page.goto("/forgot-password");
  await expect(page.getByRole("heading", { name: "Quên mật khẩu" })).toBeVisible();
  await expect(page.getByLabel("Email")).toHaveCount(1);
  const send = page.getByRole("button", { name: "Gửi hướng dẫn", exact: true });
  await expect(send).toHaveCount(1);
  await expect(send).toHaveAttribute("data-variant", "primary");
  await expect(send).toBeDisabled(); // chưa nhập email
  await page.getByLabel("Email").fill("khong.co@sv.example");
  await send.click();
  await expect(page.getByText(FORGOT_MSG)).toBeVisible();
  expect(JSON.parse(bodies[0])).toEqual({ email: "khong.co@sv.example" });
  const again = page.getByRole("button", { name: /^Gửi lại hướng dẫn/ });
  await expect(again).toBeDisabled();
  await expect(again).toContainText(/\(\d+ giây\)/);
  await expect(page.getByRole("link", { name: "Quay lại đăng nhập" })).toHaveAttribute("href", "/login");
  const body = (await page.locator("main").innerText()).toLowerCase();
  for (const w of ["token", "session", "refresh", "jwt", "bcrypt"]) expect(body).not.toContain(w);
});

test("reset page: kiểm liên kết khi mở, token rời URL, form hai ô, lỗi mật khẩu tại ô, thành công; liên kết xấu", async ({ page }) => {
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
  const T = "B".repeat(43);
  let previewCalls = 0;
  let resetBody = "";
  let goodLink = true;
  await page.route("**/api/v1/auth/tokens/preview", async (r) => {
    previewCalls++;
    await new Promise((res) => setTimeout(res, 120));
    const b = JSON.parse(r.request().postData() ?? "{}") as { kind: string; token: string };
    expect(b.kind).toBe("RESET_PASSWORD");
    if (!goodLink) return json(r, 410, err("LINK_INVALID", "m", { details: { reason: "expired" } }));
    return json(r, 200, { valid: true, kind: "RESET_PASSWORD", expires_at: "2030-01-01T00:00:00Z" });
  });
  await page.route("**/api/v1/auth/reset-password", (r) => {
    resetBody = r.request().postData() ?? "";
    const b = JSON.parse(resetBody) as { new_password: string };
    if (b.new_password.length < 10) return json(r, 422, err("VALIDATION_FAILED", "m", { details: [{ field: "new_password", code: "PASSWORD_TOO_SHORT", message: "Mật khẩu cần ít nhất 10 ký tự và không quá 72 byte." }] }));
    return json(r, 200, { status: "password_reset" });
  });
  await page.goto(`/reset-password?token=${T}`);
  await expect(page.getByText("Đang kiểm tra liên kết…")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Đặt mật khẩu mới" })).toBeVisible();
  expect(await page.evaluate(() => location.search)).toBe("");
  expect(previewCalls).toBe(1);
  const a = page.getByLabel("Mật khẩu mới", { exact: true });
  const b = page.getByLabel("Nhập lại mật khẩu", { exact: true });
  await expect(a).toHaveAttribute("autocomplete", "new-password");
  await expect(a).toHaveAttribute("type", "password");
  await expect(b).toHaveAttribute("type", "password");
  await expect(page.getByText("Ít nhất 10 ký tự, không phải mật khẩu phổ biến.")).toBeVisible();
  const submit = page.getByRole("button", { name: "Đổi mật khẩu", exact: true });
  await expect(submit).toHaveAttribute("data-variant", "primary");

  await a.fill("ngan");
  await b.fill("khac");
  await submit.click();
  await expect(page.getByText("Hai mật khẩu chưa giống nhau.")).toBeVisible();
  expect(resetBody).toBe("");
  await b.fill("ngan");
  await submit.click();
  await expect(page.getByText("Mật khẩu cần ít nhất 10 ký tự và không quá 72 byte.")).toBeVisible();
  await expect(a).toHaveAttribute("aria-invalid", "true");
  await expect(a).toHaveValue("ngan"); // giữ chữ đã gõ; token còn trong bộ nhớ trang để nhập lại

  await a.fill("Mat-khau-moi-2026");
  await b.fill("Mat-khau-moi-2026");
  await submit.click();
  await expect(page.getByRole("heading", { name: "Mật khẩu đã được đổi. Hãy đăng nhập lại." })).toBeVisible();
  expect(JSON.parse(resetBody)).toEqual({ token: T, new_password: "Mat-khau-moi-2026" });
  await expect(page.getByRole("link", { name: "Đăng nhập" })).toHaveAttribute("href", "/login");

  goodLink = false;
  await page.goto(`/reset-password?token=${T}`);
  await expect(page.getByText("Liên kết đã hết hạn hoặc đã được dùng.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Yêu cầu liên kết mới" })).toHaveAttribute("href", "/forgot-password");
  await page.goto("/reset-password"); // thiếu token
  await expect(page.getByRole("link", { name: "Yêu cầu liên kết mới" })).toBeVisible();
  expect(previewCalls).toBe(2);
});

test("reset page: Referrer-Policy no-referrer, no-store (kể cả forgot không cần)", async ({ request }) => {
  const res = await request.get("/reset-password");
  expect(res.headers()["referrer-policy"]).toBe("no-referrer");
  expect(res.headers()["cache-control"]).toContain("no-store");
});

type Dev = { id: string; current: boolean; device_label: string; ip_masked: string; created_at: string; last_used_at: string };
const dev = (id: string, label: string, current = false, ip = "203.0.*.*", last = "2026-10-03T02:20:00Z"): Dev => ({ id, current, device_label: label, ip_masked: ip, created_at: "2026-10-01T02:20:00Z", last_used_at: last });

test("settings security: đổi mật khẩu (3 ô, 1 nút primary, lỗi tại ô), thiết bị: Thiết bị này, đăng xuất ngay + dòng tĩnh, xác nhận có số", async ({ page }) => {
  await mockAuth(page, { role: "STUDENT", email: "an.nguyen@sv.example", fullName: "Nguyễn Thị An", loggedIn: true });
  let list: Dev[] = [dev("s1", "Chrome trên macOS", true), dev("s2", "Safari trên iPhone", false, "113.190.*.*", "2026-10-02T14:05:00Z"), dev("s3", "Firefox trên Windows")];
  const calls: string[] = [];
  await page.route("**/api/v1/me/sessions", async (r) => {
    const m = r.request().method();
    calls.push(m);
    if (m === "GET") return json(r, 200, { items: list });
    const n = list.filter((d) => !d.current).length;
    list = list.filter((d) => d.current);
    return json(r, 200, { revoked: n });
  });
  await page.route("**/api/v1/me/sessions/*", (r) => {
    const id = r.request().url().split("/").pop();
    calls.push(`DELETE ${id}`);
    list = list.filter((d) => d.id !== id);
    return r.fulfill({ status: 204, headers: CORS });
  });
  let pwBody = "";
  await page.route("**/api/v1/me/password", (r) => {
    pwBody = r.request().postData() ?? "";
    const b = JSON.parse(pwBody) as { current_password: string };
    if (b.current_password !== "Edupilot#2026-demo") return json(r, 422, err("VALIDATION_FAILED", "m", { details: [{ field: "current_password", code: "WRONG_PASSWORD", message: "Mật khẩu hiện tại chưa đúng." }] }));
    list = list.filter((d) => d.current);
    return r.fulfill({ status: 204, headers: CORS });
  });

  await page.goto("/settings");
  await expect(page.getByRole("heading", { name: "Tài khoản và bảo mật" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Mật khẩu", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Thiết bị đang đăng nhập" })).toBeVisible();

  // thiết bị
  const rows = page.locator("ul li").filter({ hasText: /Lần cuối/ });
  await expect(rows).toHaveCount(3);
  await expect(rows.nth(0)).toContainText("Chrome trên macOS");
  await expect(rows.nth(0)).toContainText("Thiết bị này");
  await expect(rows.nth(0).getByRole("button")).toHaveCount(0); // hàng hiện tại không có menu
  await expect(rows.nth(1)).toContainText("113.190.*.*");
  await expect(rows.nth(1)).toContainText(/Lần cuối \d{2}:\d{2}/);
  const others = page.getByRole("button", { name: "Đăng xuất mọi thiết bị khác" });
  await expect(others).toBeVisible();
  await expect(others).not.toHaveAttribute("data-variant", "primary");

  // đăng xuất một thiết bị: thực hiện ngay, không hộp thoại, dòng tĩnh tại chỗ
  await rows.nth(1).getByRole("button", { name: /Thao tác cho Safari/ }).click();
  await page.getByRole("menuitem", { name: "Đăng xuất thiết bị này" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByText("Đã đăng xuất thiết bị này")).toBeVisible();
  expect(calls).toContain("DELETE s2");

  // đăng xuất mọi thiết bị khác: xác nhận nêu số
  await others.click();
  await expect(page.getByRole("dialog")).toContainText("Đăng xuất 1 thiết bị khác. Họ sẽ phải đăng nhập lại.");
  await page.getByRole("dialog").getByRole("button", { name: "Đăng xuất", exact: true }).click();
  await expect(page.getByText("Không có thiết bị nào khác.")).toBeVisible();
  await expect(others).toHaveCount(0);
  expect(calls).toContain("DELETE");

  // đổi mật khẩu
  const cur = page.getByLabel("Mật khẩu hiện tại", { exact: true });
  const nw = page.getByLabel("Mật khẩu mới", { exact: true });
  const again = page.getByLabel("Nhập lại mật khẩu mới", { exact: true });
  for (const f of [cur, nw, again]) await expect(f).toHaveAttribute("type", "password");
  await expect(nw).toHaveAttribute("autocomplete", "new-password");
  await expect(cur).toHaveAttribute("autocomplete", "current-password");
  const change = page.getByRole("button", { name: "Đổi mật khẩu", exact: true });
  await expect(change).toHaveAttribute("data-variant", "primary");
  await cur.fill("sai-mat-khau-hien-tai");
  await nw.fill("Mat-khau-moi-2026");
  await again.fill("Mat-khau-moi-2026");
  await change.click();
  await expect(page.getByText("Mật khẩu hiện tại chưa đúng.")).toBeVisible();
  await expect(cur).toHaveAttribute("aria-invalid", "true");
  await expect(nw).toHaveValue("Mat-khau-moi-2026"); // không mất chữ đã gõ
  await cur.fill("Edupilot#2026-demo");
  await change.click();
  await expect(page.getByRole("status").filter({ hasText: "Đã đổi mật khẩu. Các thiết bị khác đã bị đăng xuất." })).toBeVisible();
  await expect(cur).toHaveValue("");
  expect(JSON.parse(pwBody)).toEqual({ current_password: "Edupilot#2026-demo", new_password: "Mat-khau-moi-2026" });

  const text = (await page.locator("main").innerText()).toLowerCase();
  for (const w of ["session", "token", "refresh", "jwt", "bcrypt"]) expect(text).not.toContain(w);
});

test("settings security: khung xương, lỗi chuẩn, 375 px dạng danh sách không tràn", async ({ page }) => {
  await mockAuth(page, { role: "TEACHER", email: "teacher@edupilot.local", loggedIn: true });
  let fail = true;
  await page.route("**/api/v1/me/sessions", async (r) => {
    await new Promise((res) => setTimeout(res, 150));
    if (fail) return json(r, 500, err("INTERNAL"));
    return json(r, 200, { items: [dev("s1", "Chrome trên macOS", true)] });
  });
  await page.setViewportSize({ width: 375, height: 800 });
  await page.goto("/settings");
  await expect(page.getByText("Thiết bị đang đăng nhập")).toBeVisible();
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible({ timeout: 10_000 });
  fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Không có thiết bị nào khác.")).toBeVisible();
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(a.cut).toEqual([]);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("@real reset logs out other device (hai context: máy B bị đăng xuất, /login kèm dòng thông báo)", async () => {
  test.skip(true, "@real: cần stack Go + Mailpit trên https://localhost — QC chạy tay");
});

type AU = { id: string; email: string; full_name: string; role: string; status: string; last_login_at: string | null; version: number };
const ME = "00000000-0000-7000-8000-0000000000a0";
const au = (id: string, name: string, role: string, status: string, over: Partial<AU> = {}): AU => ({ id, email: `${id}@sv.example`, full_name: name, role, status, last_login_at: status === "INVITED" ? null : "2026-10-03T02:20:00Z", version: 1, ...over });

function usersFixture() {
  return [au(ME, "Quản Trị Thử", "ADMIN", "ACTIVE", { email: "admin@ptit.edu.vn" }), au("gv1", "Lê Thu Hà", "TEACHER", "ACTIVE"), au("gv2", "Phạm Quốc Bảo", "TA", "INVITED"), au("sv1", "Trần Thu Uyên", "STUDENT", "ACTIVE")];
}

/** Mở khung mời: bấm có thể đến trước lúc trang gắn xong trình xử lý — thử lại tới khi khung hiện. */
async function openInvite(page: Page) {
  await expect(async () => {
    if (!(await page.getByLabel("Họ và tên").isVisible())) await page.getByRole("button", { name: "Mời giảng viên" }).click();
    await expect(page.getByLabel("Họ và tên")).toBeVisible({ timeout: 1000 });
  }).toPass({ timeout: 10_000 });
}

async function openUsers(page: Page, handlers: { patch?: (id: string, body: Record<string, unknown>) => { status: number; body: unknown }; post?: (body: Record<string, string>, key: string | undefined) => { status: number; body: unknown }; resend?: (id: string) => { status: number; body: unknown } } = {}) {
  await mockAuth(page, { role: "ADMIN", email: "admin@ptit.edu.vn", loggedIn: true });
  let rows = usersFixture();
  const log = { patch: [] as string[], post: [] as string[], keys: [] as Array<string | undefined>, resend: [] as string[], list: [] as string[] };
  await page.route("**/api/v1/admin/users**", async (r) => {
    const req = r.request();
    const url = new URL(req.url());
    const parts = url.pathname.replace("/api/v1/admin/users", "").split("/").filter(Boolean);
    if (req.method() === "GET" && parts.length === 0) {
      log.list.push(url.search);
      const role = url.searchParams.get("role");
      const q = (url.searchParams.get("q") ?? "").toLowerCase();
      const items = rows.filter((u) => (!role || u.role === role) && (!q || u.full_name.toLowerCase().includes(q) || u.email.startsWith(q)));
      return json(r, 200, { items, next_cursor: null });
    }
    if (req.method() === "POST" && parts.length === 0) {
      const b = JSON.parse(req.postData() ?? "{}") as Record<string, string>;
      log.post.push(req.postData() ?? "");
      log.keys.push(req.headers()["idempotency-key"]);
      const out = handlers.post?.(b, req.headers()["idempotency-key"]) ?? { status: 201, body: au("new", b.full_name, b.role, "INVITED", { email: b.email }) };
      if (out.status === 201) rows = [...rows, out.body as AU];
      return json(r, out.status, out.body);
    }
    if (req.method() === "PATCH") {
      const id = parts[0];
      const b = JSON.parse(req.postData() ?? "{}") as Record<string, unknown>;
      log.patch.push(`${id} ${JSON.stringify(b)}`);
      const out = handlers.patch?.(id, b);
      if (out) return json(r, out.status, out.body);
      rows = rows.map((u) => (u.id === id ? { ...u, status: String(b.status), version: u.version + 1 } : u));
      return json(r, 200, rows.find((u) => u.id === id));
    }
    if (req.method() === "POST" && parts[1] === "resend-invite") {
      log.resend.push(parts[0]);
      const out = handlers.resend?.(parts[0]) ?? { status: 200, body: { expires_at: "2026-10-06T02:20:00Z" } };
      return json(r, out.status, out.body);
    }
    return r.fallback();
  });
  await page.goto("/admin/users");
  await expect(page.getByRole("heading", { name: "Người dùng", level: 1 })).toBeVisible();
  return log;
}

test("admin users page: bảng, một nút chính, lọc theo vai, tìm, khung mời mở tại chỗ, khoá lạc quan + Hoàn tác, gửi lại lời mời", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung; 375 px kiểm trong ca riêng");
  const log = await openUsers(page);
  const invite = page.getByRole("button", { name: "Mời giảng viên" });
  await expect(invite).toHaveCount(1);
  await expect(invite).toHaveAttribute("data-variant", "primary");
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  for (const h of ["Họ tên", "Email", "Vai trò", "Trạng thái", "Lần cuối"]) await expect(page.getByRole("columnheader", { name: h })).toBeVisible();
  await expect(page.locator("table tbody tr")).toHaveCount(4);
  await expect(page.locator("table").getByText("Lê Thu Hà")).toBeVisible();
  await expect(page.locator("table").getByText("Chờ nhận lời mời")).toBeVisible();
  await expect(page.locator("table").getByText("Chưa đăng nhập")).toBeVisible();
  await expect(page.getByRole("button", { name: /Tạo sinh viên/ })).toHaveCount(0);

  // lọc theo vai + tìm
  await page.getByRole("radio", { name: "Trợ giảng" }).click();
  await expect.poll(() => log.list.at(-1)).toContain("role=TA");
  await expect(page.locator("table").getByText("Phạm Quốc Bảo")).toBeVisible();
  await expect(page.locator("table").getByText("Lê Thu Hà")).toHaveCount(0);
  await page.getByRole("radio", { name: "Tất cả" }).click();
  await page.getByLabel("Tìm người dùng").fill("uyên");
  await expect.poll(() => log.list.at(-1)).toContain("q=uy");
  await expect(page.locator("table").getByText("Trần Thu Uyên")).toBeVisible();
  await page.getByLabel("Tìm người dùng").fill("");

  // gửi lại lời mời chỉ ở hàng INVITED
  const bao = page.locator("table tbody tr").filter({ hasText: "Phạm Quốc Bảo" }).first();
  await bao.getByRole("button", { name: "Gửi lại lời mời" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi lại lời mời" })).toBeVisible();
  expect(log.resend).toEqual(["gv2"]);
  await expect(page.getByRole("button", { name: "Gửi lại lời mời" })).toHaveCount(1);

  // không có menu ở hàng của chính mình; khoá hàng khác: lạc quan ngay + Hoàn tác = mở khoá
  const me = page.locator("table tbody tr").filter({ hasText: "Quản Trị Thử" }).first();
  await expect(me.getByRole("button", { name: /Thao tác cho/ })).toHaveCount(0);
  const ha = page.locator("table tbody tr").filter({ hasText: "Lê Thu Hà" }).first();
  await ha.getByRole("button", { name: /Thao tác cho Lê Thu Hà/ }).click();
  await page.getByRole("menuitem", { name: "Khoá tài khoản" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã khoá Lê Thu Hà" })).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0); // không hộp thoại xác nhận
  await expect(ha.getByText("Đã khoá")).toBeVisible();
  expect(log.patch[0]).toBe('gv1 {"status":"DISABLED","version":1}');
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect.poll(() => log.patch.length).toBe(2);
  expect(log.patch[1]).toBe('gv1 {"status":"ACTIVE","version":2}');
  await expect(ha.getByText("Đang dùng")).toBeVisible();
});

test("admin users page: khung mời mở tại chỗ (không dialog), gửi xong có dòng tĩnh, không từ kỹ thuật; 375 px sạch", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung; 375 px kiểm trong ca riêng");
  const log = await openUsers(page);
  await openInvite(page);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByLabel("Email", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Họ và tên")).toBeVisible();
  await expect(page.getByLabel("Vai", { exact: true })).toBeVisible();
  await page.getByLabel("Email", { exact: true }).fill("Gv.Moi@sv.example");
  await page.getByLabel("Họ và tên").fill("Giảng Viên Mới");
  await page.getByLabel("Vai", { exact: true }).selectOption("TA");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi link mời, hạn 72 giờ." })).toBeVisible();
  expect(JSON.parse(log.post[0])).toEqual({ email: "Gv.Moi@sv.example", full_name: "Giảng Viên Mới", role: "TA" });
  expect(log.keys[0]).toMatch(/^ep-/);
  const text = (await page.locator("main").innerText()).toLowerCase();
  for (const w of ["token", "session", "jwt", "bcrypt", "password"]) expect(text).not.toContain(w);

  await page.setViewportSize({ width: 375, height: 800 });
  // bố cục đổi sang danh sách sau khi đổi cỡ: đợi hết tràn ngang rồi mới đo (đo ngay có thể còn bố cục 1280 px)
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

test("admin users errors: email trùng tại ô, 422 tại ô, gửi lại cùng khoá, xung đột phiên bản, tự khoá, 403", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung; 375 px kiểm trong ca riêng");
  let postCalls = 0;
  const log = await openUsers(page, {
    post: (b) => {
      postCalls++;
      if (b.email.startsWith("trung@")) return { status: 409, body: { code: "CONFLICT", message: "m", trace_id: "b".repeat(32), details: { field: "email" } } };
      if (b.email.startsWith("sai@")) return { status: 422, body: { code: "VALIDATION_FAILED", message: "m", trace_id: "b".repeat(32), details: [{ field: "full_name", code: "INVALID_NAME", message: "Họ và tên cần từ 1 đến 100 ký tự." }] } };
      if (postCalls < 5) return { status: 503, body: { code: "SERVICE_UNAVAILABLE", message: "m", trace_id: "b".repeat(32) } };
      return { status: 201, body: au("ok1", b.full_name, b.role, "INVITED", { email: b.email }) };
    },
    patch: (id) => {
      if (id === "gv1") return { status: 409, body: { code: "VERSION_CONFLICT", message: "m", trace_id: "b".repeat(32), details: { current_version: 5, current: au("gv1", "Lê Thu Hà", "TEACHER", "ACTIVE", { version: 5 }) } } };
      if (id === "sv1") return { status: 409, body: { code: "CONFLICT", message: "m", trace_id: "b".repeat(32), details: { reason: "last_admin" } } };
      return { status: 403, body: { code: "FORBIDDEN", message: "m", trace_id: "b".repeat(32), details: { reason: "role" } } };
    },
  });
  await openInvite(page);
  const email = page.getByLabel("Email", { exact: true });
  const name = page.getByLabel("Họ và tên");
  await email.fill("trung@sv.example");
  await name.fill("Người Trùng");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.getByText("Email này đã có tài khoản.")).toBeVisible();
  await expect(email).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveValue("Người Trùng"); // nội dung khung giữ nguyên

  await email.fill("sai@sv.example");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.getByText("Họ và tên cần từ 1 đến 100 ký tự.")).toBeVisible();

  // lỗi tạm thời: Gửi lại dùng CÙNG Idempotency-Key
  await email.fill("tam@sv.example");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  const retry = page.getByRole("alert").getByRole("button", { name: "Gửi lại" });
  await expect(retry).toBeVisible();
  const before = log.keys.length;
  await retry.click();
  await expect.poll(() => log.keys.length).toBe(before + 1);
  expect(log.keys.at(-1)).toBe(log.keys.at(-2));
  await page.getByRole("button", { name: "Huỷ" }).click();

  // xung đột phiên bản
  const ha = page.locator("table tbody tr").filter({ hasText: "Lê Thu Hà" }).first();
  await ha.getByRole("button", { name: /Thao tác cho/ }).click();
  await page.getByRole("menuitem", { name: "Khoá tài khoản" }).click();
  await expect(page.getByText("Tài khoản này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?")).toBeVisible();
  await page.getByRole("button", { name: "Giữ thay đổi của tôi" }).click();
  await expect.poll(() => log.patch.some((p) => p.includes('"version":5'))).toBe(true);

  // quản trị viên cuối cùng / thiếu quyền
  const sv = page.locator("table tbody tr").filter({ hasText: "Trần Thu Uyên" }).first();
  await sv.getByRole("button", { name: /Thao tác cho/ }).click();
  await page.getByRole("menuitem", { name: "Khoá tài khoản" }).click();
  await expect(page.getByRole("alert").filter({ hasText: "Không thể khoá quản trị viên cuối cùng." })).toBeVisible();
});

test("admin users errors: mạng đứt khi mời ⇒ giữ chữ + banner ngoại tuyến + Gửi lại cùng khoá", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "bảng dùng chung; 375 px kiểm trong ca riêng");
  const log = await openUsers(page);
  const keys: Array<string | undefined> = [];
  let down = true;
  await page.route("**/api/v1/admin/users", (r) => {
    if (r.request().method() !== "POST" || !down) return r.fallback();
    keys.push(r.request().headers()["idempotency-key"]);
    return r.abort("connectionfailed");
  });
  await openInvite(page);
  await page.getByLabel("Email", { exact: true }).fill("mat.mang@sv.example");
  await page.getByLabel("Họ và tên").fill("Mất Mạng");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.locator("[data-part=offline-banner]").getByText("Mất kết nối mạng.")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole("alert").getByRole("button", { name: "Gửi lại" })).toBeVisible();
  await expect(page.getByLabel("Họ và tên")).toHaveValue("Mất Mạng"); // giữ chữ đã gõ
  down = false;
  await page.getByRole("alert").getByRole("button", { name: "Gửi lại" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi link mời, hạn 72 giờ." })).toBeVisible();
  expect(new Set(keys).size).toBe(1); // mọi lần gửi (kể cả tự thử lại) dùng một khoá
  expect(JSON.parse(log.post[0]).email).toBe("mat.mang@sv.example");
});

test("invite page: xem trước, token rời URL, hai ô mật khẩu, thành công vào /; liên kết hỏng", async ({ page }) => {
  const T = "C".repeat(43);
  let previews = 0;
  let good = true;
  let accepted = "";
  const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
  await page.route("**/api/v1/auth/tokens/preview", async (r) => {
    previews++;
    await new Promise((res) => setTimeout(res, 500));
    if (!good) return json(r, 410, err("LINK_INVALID", "m", { details: { reason: "used" } }));
    return json(r, 200, { valid: true, kind: "INVITE", expires_at: "2030-01-01T00:00:00Z", full_name: "Lê Thu Hà", role: "TEACHER" });
  });
  await page.route("**/api/v1/auth/accept-invite", (r) => {
    accepted = r.request().postData() ?? "";
    const b = JSON.parse(accepted) as { password: string };
    if (b.password.length < 10) return json(r, 422, err("VALIDATION_FAILED", "m", { details: [{ field: "password", code: "PASSWORD_TOO_SHORT", message: "Mật khẩu cần ít nhất 10 ký tự và không quá 72 byte." }] }));
    return r.fulfill({ status: 200, contentType: "application/json", headers: { ...cors, "Set-Cookie": "ep_rt=x; Path=/api/v1/auth; HttpOnly" }, body: JSON.stringify(sessionBody("TEACHER", { email: "teacher@ptit.edu.vn", fullName: "Lê Thu Hà" })) });
  });
  await page.goto(`/invite/${T}`);
  await expect(page.getByText("Đang kiểm tra lời mời…")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Chào Lê Thu Hà, bạn được mời làm Giảng viên trên EduPilot." })).toBeVisible();
  expect(decodeURIComponent(new URL(page.url()).pathname)).toBe("/invite/·");
  expect(page.url()).not.toContain(T);
  expect(previews).toBe(1);
  const a = page.getByLabel("Mật khẩu", { exact: true });
  const b = page.getByLabel("Nhập lại mật khẩu", { exact: true });
  await expect(a).toHaveAttribute("autocomplete", "new-password");
  await expect(a).toHaveAttribute("type", "password");
  await expect(page.getByText("Ít nhất 10 ký tự, không phải mật khẩu phổ biến.")).toBeVisible();
  const submit = page.getByRole("button", { name: "Đặt mật khẩu và vào" });
  await expect(submit).toHaveCount(1);
  await expect(submit).toHaveAttribute("data-variant", "primary");
  await a.fill("ngan");
  await b.fill("ngan");
  await submit.click();
  await expect(page.getByText("Mật khẩu cần ít nhất 10 ký tự và không quá 72 byte.")).toBeVisible();
  await a.fill("Mat-khau-nhan-moi-2026");
  await b.fill("Mat-khau-nhan-moi-2026");
  await submit.click();
  await expect(page).toHaveURL(/\/$/);
  expect(new URL(page.url()).pathname).toBe("/");
  expect(JSON.parse(accepted)).toEqual({ token: T, password: "Mat-khau-nhan-moi-2026" });

  good = false;
  await page.goto(`/invite/${T}`);
  await expect(page.getByText("Lời mời đã hết hạn hoặc đã được dùng. Hãy nhờ quản trị viên gửi lại.")).toBeVisible();
  await expect(page.locator("main").getByRole("button")).toHaveCount(0); // không có nút tự gửi lại
  await expect(page.locator("main").getByRole("link")).toHaveCount(0);
  expect(await page.locator("body").innerText()).not.toMatch(/@/); // không lộ email
});

test("invite page: Referrer-Policy no-referrer, no-store", async ({ request }) => {
  const res = await request.get(`/invite/${"D".repeat(43)}`);
  expect(res.headers()["referrer-policy"]).toBe("no-referrer");
  expect(res.headers()["cache-control"]).toContain("no-store");
});

test("@real invite flow (Admin mời → đọc Mailpit → đặt mật khẩu → khoá → bị đăng xuất → mở khoá)", async () => {
  test.skip(true, "@real: cần stack Go + Mailpit trên https://localhost — QC chạy tay");
});

test("@real login + refresh qua Caddy cùng origin (cần stack Go: docker-compose.test.yml)", async () => {
  test.skip(true, "@real: cần stack Go + Caddy trên https://localhost — QC chạy tay");
});

test("@real register verify login (cần stack Go + Mailpit: đăng ký → đọc Mailpit → xác minh → đăng nhập → vào /)", async () => {
  test.skip(true, "@real: cần stack Go + Mailpit trên https://localhost — QC chạy tay");
});

// US-P2-12 AC3: 7 tài khoản seed (đăng nhập giả bằng đúng email seed) thấy đúng số mục điều hướng: SV 7, TA 12, GV 15, Admin 6, SV chưa vào lớp 1.
test("seed accounts nav: số mục theo vai (7 / 12 / 15 / 6 / 1)", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "thanh bên chỉ có ở bề rộng desktop");
  const cases: Array<[Parameters<typeof asDemo>[1], string | undefined, number]> = [
    ["student", "sv-1", 7], ["student", "sv-2", 7], ["student", "sv-3", 7], ["student", "sv-4", 1], ["ta", undefined, 12], ["teacher", undefined, 15], ["admin", undefined, 6],
  ];
  for (const [role, person, want] of cases) {
    await context.clearCookies();
    await context.unrouteAll();
    await asDemo(context, role, { person });
    await page.goto("/");
    await expect(page.locator("[data-part=sidebar] nav a").first()).toBeVisible();
    await expect(page.locator("[data-part=sidebar] nav a"), `${role} ${person ?? ""}`).toHaveCount(want);
  }
});
