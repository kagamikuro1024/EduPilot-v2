import { BASE_URL } from "./env";
import type { BrowserContext, Page } from "@playwright/test";

export type DemoRole = "student" | "ta" | "teacher" | "admin";

/** Đặt phiên MÔ PHỎNG (cookie ep_demo_*) trước khi mở trang — như bộ đổi vai của prototype 1.5. */
export async function asDemo(context: BrowserContext, role: DemoRole, opts: { person?: string } = {}) {
  const url = BASE_URL;
  await context.addCookies([
    { name: "ep_demo_role", value: role, url },
    ...(role === "student" ? [{ name: "ep_demo_person", value: opts.person ?? "sv-2", url }] : []),
  ]);
}

export type JwtRole = "ADMIN" | "TEACHER" | "TA" | "STUDENT";

/** JWT giả (chữ ký không kiểm ở client): chỉ để frontend giải mã vai / email / hạn. */
export function makeJwt(role: JwtRole | string, opts: { email?: string; expIn?: number; sid?: string } = {}) {
  const b64 = (o: unknown) => Buffer.from(JSON.stringify(o)).toString("base64url");
  return `${b64({ alg: "HS256", typ: "JWT" })}.${b64({
    sub: "00000000-0000-7000-8000-0000000000a0", role, email: opts.email ?? "admin@ptit.edu.vn", sid: opts.sid ?? "00000000-0000-7000-8000-0000000000b0",
    iat: Math.floor(Date.now() / 1000), exp: Math.floor(Date.now() / 1000) + (opts.expIn ?? 900),
  })}.c2ln`;
}

export const sessionBody = (role: JwtRole, opts: { email?: string; fullName?: string; expIn?: number } = {}) => {
  const email = opts.email ?? `${role.toLowerCase()}@ptit.edu.vn`;
  return {
    access_token: makeJwt(role, { email, expIn: opts.expIn }), token_type: "Bearer", expires_in: 900,
    user: { id: "00000000-0000-7000-8000-0000000000a0", email, full_name: opts.fullName ?? "Người Thử", role, status: "ACTIVE", email_verified: true },
  };
};

/**
 * Giả lập phiên đăng nhập thật cho MỘT trang: `POST /auth/refresh` (lúc tải trang) trả về phiên của `role`; `POST /auth/logout` → 204.
 * Không dùng máy chủ giả dùng chung nên không cần khoá. Trả bộ đếm lời gọi refresh để test khẳng định.
 */
export async function asJwt(page: Page, role: JwtRole, opts: { email?: string; fullName?: string; expIn?: number } = {}) {
  const calls = { refresh: 0, logout: 0 };
  const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
  await page.route("**/api/v1/auth/refresh", (route) => {
    calls.refresh++;
    return route.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(sessionBody(role, opts)) });
  });
  await page.route("**/api/v1/auth/logout", (route) => {
    calls.logout++;
    return route.fulfill({ status: 204, headers: cors });
  });
  return calls;
}
