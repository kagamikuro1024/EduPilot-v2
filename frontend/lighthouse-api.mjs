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

// `/assignments/[id]` thật (US-P7-03): `GET …/assignments/{aid}/mine` theo `MineOut`. Đề bài nguyên văn
// `docs/specs/FEAT-assignments/fixtures/de-bai-bt03.md`, tiêu chí của `rubric-bt03.json` (chỉ tên + điểm — sinh viên không thấy `guidance`).
const BT03_DESCRIPTION = `**Loại:** tự luận (ESSAY) · **Thang điểm:** 10 · **Hạn nộp:** 23:59 ngày hạn (giờ Việt Nam) · **Nộp muộn:** được, trừ 0,5 điểm mỗi ngày trễ, tối đa 2 ngày

## Đề bài

Phân tích chiến dịch tấn công chuỗi cung ứng nhằm vào phần mềm giám sát hạ tầng **SolarWinds Orion** (phát hiện tháng 12 năm 2020). Bài nộp dài khoảng một đến hai trang và trả lời bốn câu hỏi:

1. Ai là tác nhân và kẻ tấn công đã khai thác bề mặt tấn công nào?
2. Chuỗi tấn công diễn ra qua những bước nào?
3. Tác động của vụ việc là gì (dữ liệu, vận hành, niềm tin, chi phí)?
4. Tổ chức nên phòng thủ như thế nào để giảm rủi ro tương tự?

Mỗi nhận định cần nêu căn cứ cụ thể (tên công cụ, kỹ thuật, số liệu công khai). Nếu không tìm được số liệu thì nói rõ là chưa có số liệu, đừng đoán.

## Yêu cầu nộp

- Tệp PDF, DOCX, TXT hoặc MD, tối đa 20 MB.
- Không ghi họ tên hay MSSV trong thân bài (hệ thống chấm không đọc phần đó; giảng viên biết bài của ai qua tài khoản nộp).
- Nộp trong EduPilot, hoặc gửi thư tới hộp thư lớp với tiêu đề \`[BT03] <MSSV>\`, hoặc nộp qua ZIP do trợ giảng gom (tên tệp chứa MSSV).`;

const MINE = (aid) => ({
  assignment: {
    id: aid,
    title: "Bài tập 03 — Phân tích một vụ tấn công thực tế",
    description: BT03_DESCRIPTION,
    due_at: "2026-11-20T16:59:00Z",
    max_score: "10",
    criteria: [
      { id: "c1", name: "Xác định tác nhân và bề mặt tấn công", max_score: "2.5" },
      { id: "c2", name: "Phân tích tấn công", max_score: "2.5" },
      { id: "c3", name: "Đánh giá tác động", max_score: "2.5" },
      { id: "c4", name: "Biện pháp phòng thủ", max_score: "2.5" },
    ],
    allow_late: true,
    late_penalty_per_day: "0.5",
    late_max_days: 2,
    appeal_days: 7,
  },
  submission: null,
  state: "NOT_SUBMITTED",
});

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
    // /me thật (US-P5-06): ba API của chính sinh viên theo hợp đồng thật, để màn dựng đủ ba khối (không đo trang lỗi).
    if (/^\/courses\/[^/]+\/me\/attendance$/.test(path)) return send(200, { attendance: { recorded: 9, present: 6, late: 1, excused: 0, absent: 2, sessions: [{ session_no: 7, starts_at: "2026-10-08T02:00:00Z", status: "ABSENT" }, { session_no: 3, starts_at: "2026-09-10T02:00:00Z", status: "ABSENT" }] }, next_cursor: null });
    if (/^\/courses\/[^/]+\/me\/participation$/.test(path)) return send(200, { participation: { speak_count: 2, points_total: "0.50", events: [{ date: "2026-10-01", type: "SPEAK", points: "0.25" }, { date: "2026-09-17", type: "SPEAK", points: "0.25" }] }, next_cursor: null });
    if (/^\/courses\/[^/]+\/me\/study-time$/.test(path)) return send(200, { weeks: Array.from({ length: 8 }, (_, i) => ({ week_start: new Date(Date.UTC(2026, 7, 17 + i * 7)).toISOString().slice(0, 10), minutes: 120 + i * 10, by_area: { CHAT: 60 } })), this_week_minutes: 45, tracking: { enabled: true, locked: false } });
    const mine = /^\/courses\/[^/]+\/assignments\/([^/]+)\/mine$/.exec(path);
    if (mine) return send(200, MINE(mine[1]));
    // /settings/llm (đọc): dữ liệu như người dùng thật; không có thì màn dựng thông báo lỗi sau phiên và LCP đo trang lỗi.
    if (role === "admin" && path.startsWith("/admin/llm/")) {
      const model = { id: "m1", model: "gpt-4o-mini", kind: "chat", dims: null, price_in: "0.15", price_out: "0.60", enabled: true };
      const provider = { id: "pv1", type: "openai", name: "OpenAI", base_url: null, enabled: true, has_key: true, key_status: "ok", rpm_limit: 600, tpm_limit: null, last_test: { ok: true, at: "2026-10-01T00:00:00Z", error_kind: null }, circuit: "closed", models: [model], version: 1 };
      if (path === "/admin/llm/providers") return send(200, { items: [provider], env_fallback: { active: false, providers: [] } });
      if (path === "/admin/llm/routes") return send(200, { items: [{ task: "chat", lane: "INTERACTIVE", chain: [{ model_id: "m1", provider_id: "pv1", provider_name: "OpenAI", model: "gpt-4o-mini" }], params: {}, version: 1 }], embedding: null });
      if (path === "/admin/llm/budget") return send(200, { scope: "global", daily_limit: "10.00", monthly_limit: "200.00", spent_today: "1.20", spent_month: "40.00", pct_today: 12, pct_month: 20, state: "ok", version: 1 });
      if (path === "/admin/llm/usage") return send(200, { from: "2026-10-01T00:00:00Z", to: "2026-10-30T00:00:00Z", group: "task", items: [] });
    }
    if (path === "/notifications") return send(200, { items: [], next_cursor: null, unread_count: 0 });
    return send(404, { code: "NOT_FOUND", message: "x", trace_id: "0".repeat(32) });
  })
  .listen(PORT, () => console.log(`lighthouse fake api :${PORT}`));
