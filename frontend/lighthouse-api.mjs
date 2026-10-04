// Gateway GIẢ tối thiểu cho Lighthouse CI (US-P2-12 AC10 bỏ phiên mô phỏng bằng cookie): trả phiên JWT giả theo cookie `lh_role` (đặt bởi
// lighthouse-auth.cjs), lớp của vai đó và "Hôm nay" rỗng. Mọi đường khác → 404 JSON. Chỉ dùng để đo; không phải gateway thật.
import http from "node:http";

const PORT = Number(process.env.LH_API_PORT || 3312);
const ORIGIN = process.env.LH_ORIGIN || "http://localhost:3310";
const b64 = (o) => Buffer.from(JSON.stringify(o)).toString("base64url");

const EMAIL = { student: "sv.kha@edupilot.local", teacher: "teacher@edupilot.local", ta: "ta@edupilot.local", admin: "admin@edupilot.local" };
const C1 = "00000000-0000-7000-8000-00000000c001";
const C2 = "00000000-0000-7000-8000-00000000c002";
const item = (id, code, role) => ({
  course: { id, class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" },
  role_in_course: role,
  enrollment_status: "ACTIVE",
});
const COURSES = {
  student: [item(C1, "761987", "STUDENT")],
  teacher: [item(C1, "761987", "TEACHER"), item(C2, "761988", "TEACHER")],
  ta: [item(C1, "761987", "TA")],
  admin: [],
};
const TODAY = {
  student: { no_course: false, email_verified: true, recommended: null, timeline: [], continue: [] },
  teacher: { count: 0, actions: [], attention: [], upcoming: [] },
  ta: { count: 0, actions: [], attention: [], upcoming: [] },
  admin: { count: 0, actions: [] },
};

function roleOf(req) {
  const m = /(?:^|; )lh_role=([a-z]+)/.exec(req.headers.cookie || "");
  return m && EMAIL[m[1]] ? m[1] : null;
}

http
  .createServer((req, res) => {
    const cors = { "Access-Control-Allow-Origin": ORIGIN, "Access-Control-Allow-Credentials": "true", "Access-Control-Allow-Methods": "GET,POST,PUT,PATCH,DELETE,OPTIONS", "Access-Control-Allow-Headers": "authorization,content-type,idempotency-key,x-request-id,if-none-match", Vary: "Origin" };
    const send = (status, body) => {
      res.writeHead(status, { ...cors, "Content-Type": "application/json" });
      res.end(body === undefined ? "" : JSON.stringify(body));
    };
    if (req.method === "OPTIONS") return send(204);
    const path = new URL(req.url, "http://x").pathname.replace(/^\/api\/v1/, "");
    const role = roleOf(req);
    if (path === "/auth/refresh") {
      if (!role) return send(401, { code: "UNAUTHENTICATED", message: "x", trace_id: "0".repeat(32) });
      const now = Math.floor(Date.now() / 1000);
      const jwt = `${b64({ alg: "HS256", typ: "JWT" })}.${b64({ sub: "00000000-0000-7000-8000-0000000000a0", role: role.toUpperCase(), email: EMAIL[role], sid: "00000000-0000-7000-8000-0000000000b0", iat: now, exp: now + 3600 })}.c2ln`;
      return send(200, { access_token: jwt, token_type: "Bearer", expires_in: 3600, user: { id: "00000000-0000-7000-8000-0000000000a0", email: EMAIL[role], full_name: "Người Thử", role: role.toUpperCase(), status: "ACTIVE", email_verified: true } });
    }
    if (!role) return send(401, { code: "UNAUTHENTICATED", message: "x", trace_id: "0".repeat(32) });
    if (path === "/me/courses") return send(200, { items: COURSES[role], next_cursor: null });
    if (path === "/me/today" || /^\/courses\/[^/]+\/today$/.test(path)) return send(200, TODAY[role]);
    if (path === "/notifications") return send(200, { items: [], next_cursor: null, unread_count: 0 });
    return send(404, { code: "NOT_FOUND", message: "x", trace_id: "0".repeat(32) });
  })
  .listen(PORT, () => console.log(`lighthouse fake api :${PORT}`));
