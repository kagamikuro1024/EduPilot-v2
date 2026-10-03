import { expect, test, type Page, type Route } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { loadAudit, runAudit } from "./support/audit";
import { sessionBody, type JwtRole } from "./support/session";

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

test("login page: bị chờ ⇒ đếm ngược và khoá nút", async ({ page }) => {
  await page.route("**/api/v1/auth/refresh", (r) => json(r, 401, err("UNAUTHENTICATED")));
  await page.route("**/api/v1/auth/login", (r) => json(r, 429, err("LOGIN_THROTTLED", "m", { retry_after: 32 }), { "Retry-After": "32" }));
  await page.goto("/login");
  await fillLogin(page, "a@b.example", "x");
  await expect(page.getByRole("alert").filter({ hasText: /^Bạn đã thử quá nhiều lần\. Thử lại sau 00:3[12]\.$/ })).toBeVisible();
  await expect(page.getByRole("button", { name: "Đăng nhập", exact: true })).toBeDisabled();
});

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

test("@real login + refresh qua Caddy cùng origin (cần stack Go: docker-compose.test.yml)", async () => {
  test.skip(true, "@real: cần stack Go + Caddy trên https://localhost — QC chạy tay");
});

test("@real register verify login (cần stack Go + Mailpit: đăng ký → đọc Mailpit → xác minh → đăng nhập → vào /)", async () => {
  test.skip(true, "@real: cần stack Go + Mailpit trên https://localhost — QC chạy tay");
});
