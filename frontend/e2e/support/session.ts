import { BASE_URL } from "./env";
import type { BrowserContext, Page } from "@playwright/test";

export type DemoRole = "student" | "ta" | "teacher" | "admin";

// Tài khoản seed (US-P2-12) ↔ người mock: màn mô phỏng lấy NGƯỜI từ email của phiên (src/mock/identity.ts). Không còn cookie phiên mô phỏng (AC10).
const SEED_EMAIL: Record<string, string> = {
  "sv-1": "sv.gioi@edupilot.local", "sv-2": "sv.kha@edupilot.local", "sv-3": "sv.nguyco@edupilot.local", "sv-4": "sv.moi@edupilot.local",
  ta: "ta@edupilot.local", teacher: "teacher@edupilot.local", admin: "admin@edupilot.local",
};
// Lớp THẬT tương ứng lớp mô phỏng (class_code 761987 ↔ int1006-1, 761988 ↔ int1006-2); mã sinh viên mô phỏng → lớp họ học.
const COURSE_ID = { "761987": "00000000-0000-7000-8000-00000000c001", "761988": "00000000-0000-7000-8000-00000000c002" } as const;
const STUDENT_CLASSES: Record<string, Array<keyof typeof COURSE_ID>> = { "sv-1": ["761987", "761988"], "sv-2": ["761987"], "sv-3": ["761987"], "sv-4": [] };

const mineItem = (code: keyof typeof COURSE_ID, role: "TEACHER" | "TA" | "STUDENT") => ({
  course: { id: COURSE_ID[code], class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" },
  role_in_course: role,
  enrollment_status: "ACTIVE",
});

/**
 * Đăng nhập GIẢ bằng JWT cho màn mô phỏng (thay bộ đổi vai bằng cookie đã bị xoá — US-P2-12 AC10): `POST /auth/refresh` trả phiên của
 * tài khoản seed tương ứng vai / người, `GET /me/courses` trả lớp thật tương ứng. Gateway không chạy: mọi lời gọi khác bị bỏ qua.
 */
export async function asDemo(context: BrowserContext, role: DemoRole, opts: { person?: string } = {}) {
  const person = role === "student" ? (opts.person ?? "sv-2") : role;
  const email = SEED_EMAIL[person];
  const jwtRole: JwtRole = role === "student" ? "STUDENT" : (role.toUpperCase() as JwtRole);
  const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
  const json = (body: unknown) => ({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  await context.route("**/api/v1/auth/refresh", (r) => r.fulfill(json(sessionBody(jwtRole, { email }))));
  await context.route("**/api/v1/auth/logout", (r) => r.fulfill({ status: 204, headers: cors }));
  const items =
    role === "student" ? (STUDENT_CLASSES[person] ?? []).map((c) => mineItem(c, "STUDENT"))
    : role === "ta" ? [mineItem("761987", "TA")]
    : role === "teacher" ? [mineItem("761987", "TEACHER"), mineItem("761988", "TEACHER")]
    : [];
  await context.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items, next_cursor: null })));
  const today = role === "student" ? { no_course: items.length === 0, email_verified: true, recommended: null, timeline: [], continue: [] }
    : role === "admin" ? { count: 0, actions: [] }
    : { count: 0, actions: [], attention: [], upcoming: [] };
  await context.route("**/api/v1/me/today**", (r) => r.fulfill(json(today)));
  await context.route("**/api/v1/courses/*/today**", (r) => r.fulfill(json(today)));
  await context.route("**/api/v1/notifications**", (r) => r.fulfill(json({ items: [], next_cursor: null, unread_count: 0 })));
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
