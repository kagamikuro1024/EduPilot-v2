// Dữ liệu mẫu của P2 (FEAT-course-foundation US-P2-12, SRS 4.8): 2 lớp × 30 sinh viên dựng bằng CHÍNH API thật của gateway,
// nên chạy seed cũng là kiểm F1 + F2 đầu cuối. Không INSERT, không route _test, không ghi DB; chỉ `fetch` và `node:` chuẩn (Node >= 24 và Bun).
//
//   node scripts/seed.mjs [--if-empty] [--verbose]      pnpm seed
//
// Biến: API_URL (https://localhost/api/v1), MAILPIT_URL (http://localhost:8025), SEED_DEFAULT_PASSWORD, SEED_RNG (20261029),
//       SEED_BASE_DATE (hôm nay, YYYY-MM-DD), SEED_ADMIN_CMD (mặc định: docker compose exec gateway /gateway admin create).
// Idempotent: mỗi bước kiểm trạng thái trước khi làm; chạy lại không tạo bản ghi / thư mới.
import { spawnSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { ANSWER_KEY_LINES, EVENTS, EXAM_PAPER_LINES, THREADS_1, THREADS_2, chatQuestions, miniPdf } from "./chat-seed-data.mjs";
import { createHash } from "node:crypto";
import { CODE, CODE_ATTEMPTS, EXAM_MCQ, MCQ_STUDENTS, QUESTIONS, SOLUTIONS, mcqAnswers } from "./exam-seed-data.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const argv = process.argv.slice(2);
const IF_EMPTY = argv.includes("--if-empty");
const VERBOSE = argv.includes("--verbose");

// ---- cấu hình --------------------------------------------------------------------------------------------------------
function envFileValue(key) {
  for (const f of [".env.local", ".env.example"]) {
    const p = path.join(root, f);
    if (!existsSync(p)) continue;
    const line = readFileSync(p, "utf8").split("\n").find((l) => l.startsWith(`${key}=`));
    if (line) return line.slice(key.length + 1).trim();
  }
  return undefined;
}
const setting = (key, fallback) => process.env[key] || envFileValue(key) || fallback;

// Chặn production TRƯỚC mọi lời gọi mạng / lệnh.
if (setting("APP_ENV", "dev") === "production") {
  console.error("Seed bị chặn ở production.");
  process.exit(1);
}

const API = setting("API_URL", "https://localhost/api/v1").replace(/\/$/, "");
const MAILPIT = setting("MAILPIT_URL", "http://localhost:8025").replace(/\/$/, "");
const PASSWORD = setting("SEED_DEFAULT_PASSWORD", "");
const RNG_SEED = Number(setting("SEED_RNG", "20261029"));
if (!PASSWORD) {
  console.error("Thiếu SEED_DEFAULT_PASSWORD (xem .env.example).");
  process.exit(1);
}
const insecureTLS = /^https:\/\/(localhost|127\.0\.0\.1)/.test(API); // chứng chỉ nội bộ của Caddy (dev)
if (insecureTLS) process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
const ORIGIN = new URL(API).origin;

const DOMAIN = "@edupilot.local";
const SEMESTER = "2026-2027-HK1";
const STEPS = 13;

// ---- tiện ích --------------------------------------------------------------------------------------------------------
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
/** Che mật khẩu và mọi chuỗi giống token trong dòng log (AC11). */
const mask = (s) => String(s).split(PASSWORD).join("***").replace(/[A-Za-z0-9_-]{40,}/g, "***");
const step = (n, text) => console.log(`[${n}/${STEPS}] ${text}`);
const vlog = (...a) => VERBOSE && console.log("   ", mask(a.join(" ")));

class ApiFail extends Error {
  constructor(method, route, status, body) {
    super(`${method} ${route} → ${status} ${body?.code ?? ""}`);
    this.status = status;
    this.code = body?.code;
    this.body = body;
  }
}

/** Gọi API thật. Trả { status, body }. `ok` liệt kê các mã chấp nhận; mã khác ném ApiFail. 429 thử lại theo Retry-After (≤ 3 lần). */
async function call(method, route, { token, json, form, key, ok = [200, 201, 202, 204] } = {}) {
  for (let attempt = 0; ; attempt++) {
    const headers = { Accept: "application/json", Origin: ORIGIN };
    if (token) headers.Authorization = `Bearer ${token}`;
    if (key) headers["Idempotency-Key"] = key.replace(/[^A-Za-z0-9._:-]/g, "-"); // khoá chỉ nhận [A-Za-z0-9._:-]
    if (json !== undefined) headers["Content-Type"] = "application/json";
    const res = await fetch(API + route, { method, headers, body: form ?? (json === undefined ? undefined : JSON.stringify(json)), tls: { rejectUnauthorized: false } });
    let body = null;
    if (res.status !== 204) body = await res.json().catch(() => null);
    vlog(method, route, "→", res.status);
    if (res.status === 429 && attempt < 3) {
      await sleep(Math.min(10, Number(body?.retry_after ?? res.headers.get("Retry-After") ?? 2)) * 1000);
      continue;
    }
    if (!ok.includes(res.status)) throw new ApiFail(method, route, res.status, body);
    return { status: res.status, body };
  }
}

async function login(email) {
  const r = await call("POST", "/auth/login", { json: { email, password: PASSWORD }, ok: [200, 401, 403, 423, 429] });
  return r.status === 200 ? r.body : null;
}

// ---- Mailpit (chỉ để lấy liên kết từ thư) ----------------------------------------------------------------------------
async function mp(route) {
  const res = await fetch(MAILPIT + route);
  if (!res.ok) throw new Error(`Mailpit ${route} → ${res.status}`);
  return res.json();
}

/** Thư gửi tới `addr` có tiêu đề chứa `subject`, MỚI nhất trước. */
async function mailsTo(addr, subject) {
  const r = await mp(`/api/v1/search?query=${encodeURIComponent(`to:${addr}`)}&limit=50`);
  return (r.messages ?? []).filter((m) => m.Subject.includes(subject)).sort((a, b) => (a.Created < b.Created ? 1 : -1));
}

async function linkOf(msgId, re) {
  const m = await mp(`/api/v1/message/${msgId}`);
  return re.exec(m.Text ?? "")?.[1];
}

/** Chờ thư mới nhất (hoặc mới hơn `after`) rồi trích token bằng `re`. */
async function waitToken(addr, subject, re, { after = "", timeoutMs = 90_000 } = {}) {
  const until = Date.now() + timeoutMs;
  for (;;) {
    const found = (await mailsTo(addr, subject)).find((m) => m.Created > after);
    const tok = found && (await linkOf(found.ID, re));
    if (tok) return { token: tok, created: found.Created };
    if (Date.now() > until) throw new Error(`Không thấy thư "${subject}" tới ${addr} sau ${timeoutMs / 1000}s (worker gửi thư chưa chạy?)`);
    await sleep(400);
  }
}

const RE_INVITE = /\/invite\/([A-Za-z0-9_-]+)/;
const RE_VERIFY = /verify-email\?token=([A-Za-z0-9_-]+)/;

// ---- dữ liệu mẫu cố định theo hạt giống -----------------------------------------------------------------------------
function mulberry32(a) {
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
const rand = mulberry32(RNG_SEED);
const pick = (xs) => xs[Math.floor(rand() * xs.length)];
const HO = ["Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Huỳnh", "Phan", "Vũ", "Võ", "Đặng", "Bùi", "Đỗ", "Hồ", "Ngô", "Dương", "Lý"];
const DEM = ["Văn", "Thị", "Minh", "Quốc", "Hoàng", "Thanh", "Ngọc", "Đức", "Gia", "Hữu", "Bảo", "Khánh", "Tuấn", "Thu", "Mai", "Anh"];
const TEN = ["An", "Bình", "Chi", "Dũng", "Giang", "Hà", "Hải", "Hiếu", "Hương", "Khoa", "Lan", "Linh", "Long", "Nam", "Phúc", "Quân", "Sơn", "Thảo", "Trang", "Việt", "Yến"];
const usedNames = new Set();
const usedCodes = new Set(["20229001", "20229002", "20229003", "20229004"]);
function genName() {
  for (;;) {
    const n = `${pick(HO)} ${pick(DEM)} ${pick(TEN)}`;
    if (!usedNames.has(n)) {
      usedNames.add(n);
      return n;
    }
  }
}
function genCode() {
  for (;;) {
    const c = `2022${String(1000 + Math.floor(rand() * 8999))}`; // 2022 + 4 chữ số; không phải MSSV thật
    if (!usedCodes.has(c)) {
      usedCodes.add(c);
      return c;
    }
  }
}
const sv = (n) => `sv${String(n).padStart(2, "0")}`;

// Sinh viên có tên cố định (A, B, C, D, "mạo danh", "chưa xác minh").
const NAMED = {
  "sv.gioi": { name: "Vũ Hoàng Giang", code: "20229001" },
  "sv.kha": { name: "Bùi Thanh Khải", code: "20229002" },
  "sv.nguyco": { name: "Ngô Ngọc Cẩm", code: "20229003" },
  "sv.moi": { name: "Lý Thu Minh", code: "20229004" },
  "sv.lech": { name: "Dương Quang Lệch", code: "20229002" }, // MSSV trùng B: kẻ mạo danh
  "sv.chuaxm": { name: "Hồ Văn Chờ", code: genCode() },
};
const accounts = new Map(); // local-part → { email, name, code }
for (const [k, v] of Object.entries(NAMED)) accounts.set(k, { email: k + DOMAIN, ...v });
const roster1 = ["sv.gioi", "sv.kha", "sv.nguyco"];
for (let i = 4; i <= 30; i++) roster1.push(sv(i));
const only2 = [];
for (let i = 31; i <= 51; i++) only2.push(sv(i));
const pending2 = [sv(52), sv(53), sv(54)];
for (const k of [...roster1.slice(3), ...only2, ...pending2]) accounts.set(k, { email: k + DOMAIN, name: genName(), code: genCode() });
const A = accounts.get("sv.gioi").email;

// ---- ngày giờ (Asia/Ho_Chi_Minh) ------------------------------------------------------------------------------------
function baseDate() {
  const given = process.env.SEED_BASE_DATE;
  if (given) return given;
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Ho_Chi_Minh", year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date());
}
const addDays = (ymd, n) => {
  const d = new Date(`${ymd}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
};
const isoWeekday = (ymd) => ((new Date(`${ymd}T00:00:00Z`).getUTCDay() + 6) % 7) + 1; // 1 = thứ Hai

// ---- các bước --------------------------------------------------------------------------------------------------------
const ctx = { tokens: {}, ids: {} };

async function adminLogin() {
  return login("admin" + DOMAIN);
}

async function isSeeded() {
  const a = await adminLogin();
  if (!a) return false;
  const r = await call("GET", "/admin/courses?limit=100", { token: a.access_token });
  const codes = new Set(r.body.items.map((c) => c.class_code));
  return codes.has("761987") && codes.has("761988");
}

async function step1Admin() {
  step(1, "Admin đầu tiên (gateway admin create)…");
  let s = await adminLogin();
  if (!s) {
    const cmd = process.env.SEED_ADMIN_CMD;
    const args = ["--email", "admin" + DOMAIN, "--name", "Quản trị viên EduPilot"];
    const run = cmd
      ? spawnSync("sh", ["-c", `${cmd} ${args.map((a) => JSON.stringify(a)).join(" ")}`], { cwd: root, input: PASSWORD + "\n", encoding: "utf8" })
      : spawnSync("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", process.env.COMPOSE_PROJECT_NAME || "edupilot", "exec", "-T", "gateway", "/gateway", "admin", "create", ...args], { cwd: root, input: PASSWORD + "\n", encoding: "utf8" });
    if (run.status !== 0) throw new Error(`Không tạo được Admin (mã ${run.status}): ${mask(run.stderr || run.error?.message || "")}`);
    s = await adminLogin();
    if (!s) throw new Error("Đã tạo Admin nhưng không đăng nhập được.");
  }
  ctx.tokens.admin = s.access_token;
}

/** Trạng thái tài khoản theo Admin (KHÔNG đăng nhập thử: đăng nhập sai bị tính vào giới hạn thất bại theo IP). null = chưa có. */
async function userState(email) {
  const r = await call("GET", `/admin/users?q=${encodeURIComponent(email)}&limit=10`, { token: ctx.tokens.admin });
  return r.body.items.find((u) => u.email === email) ?? null;
}

async function mustLogin(email) {
  const s = await login(email);
  if (!s) throw new Error(`Không đăng nhập được ${email}`);
  return s;
}

async function loginOrAccept(email, name, role) {
  let user = await userState(email);
  if (!user) {
    const r = await call("POST", "/admin/users", { token: ctx.tokens.admin, key: `seed-invite-${email}`, json: { email, full_name: name, role }, ok: [201, 409] });
    user = r.status === 201 ? r.body : await userState(email);
  }
  if (user.status === "ACTIVE") return mustLogin(email);
  for (let attempt = 0; attempt < 3; attempt++) {
    const { token } = await waitToken(email, "Lời mời tham gia EduPilot", RE_INVITE);
    const acc = await call("POST", "/auth/accept-invite", { json: { token, password: PASSWORD }, ok: [200, 400, 403, 404, 410, 422] });
    if (acc.status === 200) return acc.body;
    await call("POST", `/admin/users/${user.id}/resend-invite`, { token: ctx.tokens.admin, ok: [200, 201, 202, 409, 422] }); // token cũ hết hiệu lực
    await sleep(500);
  }
  throw new Error(`Không nhận được lời mời cho ${email}`);
}

async function step2Staff() {
  step(2, "Mời giảng viên và trợ giảng, nhận lời mời…");
  const t = await loginOrAccept("teacher" + DOMAIN, "TS. Lê Thu Hà", "TEACHER");
  const a = await loginOrAccept("ta" + DOMAIN, "Phạm Quốc Bảo", "TA");
  ctx.tokens.teacher = t.access_token;
  ctx.tokens.ta = a.access_token;
  ctx.ids.teacher = t.user.id;
  ctx.ids.ta = a.user.id;
}

const COURSES = [
  { code: "761987", join: "AN7K2MQ", capacity: undefined },
  { code: "761988", join: "BX4P9TW", capacity: 30 },
];

async function step3Courses() {
  step(3, "Mở hai lớp 761987 / 761988, gán giảng viên và TA…");
  for (const c of COURSES) {
    const list = await call("GET", "/admin/courses?limit=100", { token: ctx.tokens.admin });
    let row = list.body.items.find((x) => x.class_code === c.code);
    if (!row) {
      const body = { subject_code: "INT1006", class_code: c.code, name: "An ninh mạng", semester: SEMESTER, join_code: c.join };
      if (c.capacity) body.capacity = c.capacity;
      const r = await call("POST", "/admin/courses", { token: ctx.tokens.admin, key: `seed-course-${c.code}`, json: body, ok: [201, 409] });
      row = r.status === 201 ? r.body : (await call("GET", "/admin/courses?limit=100", { token: ctx.tokens.admin })).body.items.find((x) => x.class_code === c.code);
    }
    ctx.ids[c.code] = row.id;
    const full = (await call("GET", "/admin/courses?limit=100", { token: ctx.tokens.admin })).body.items.find((x) => x.id === row.id);
    const wantTA = c.code === "761987";
    const hasTeacher = full.teacher?.id === ctx.ids.teacher;
    const hasTA = !wantTA || full.assistants_count >= 1;
    if (!hasTeacher || !hasTA) {
      await call("POST", `/admin/courses/${row.id}/assign`, { token: ctx.tokens.admin, json: { teacher_id: ctx.ids.teacher, ...(wantTA ? { ta_ids: [ctx.ids.ta] } : {}) } });
    }
  }
  // Thông báo nhận lớp tới qua worker (outbox): chờ đủ 2, đánh dấu đã đọc thông báo lớp 1.
  const until = Date.now() + 60_000;
  for (;;) {
    const n = await call("GET", "/notifications?limit=100", { token: ctx.tokens.teacher });
    const assigned = n.body.items.filter((x) => x.type === "COURSE_ASSIGNED");
    if (assigned.length >= 2) {
      const first = assigned.find((x) => x.course_id === ctx.ids["761987"]);
      if (first && !first.read_at) await call("POST", `/notifications/${first.id}/read`, { token: ctx.tokens.teacher });
      return;
    }
    if (Date.now() > until) throw new Error("Giảng viên chưa nhận đủ 2 thông báo nhận lớp (worker chạy chưa?)");
    await sleep(500);
  }
}

async function step4Sessions() {
  step(4, "Tạo buổi học (buổi hôm nay của hai lớp)…");
  const base = baseDate();
  const wd = isoWeekday(base);
  const plan = [
    { code: "761987", total: 15, from: addDays(base, -63), to: addDays(base, 35), start: "09:00", end: "11:30", room: "P.302" },
    { code: "761988", total: 6, from: addDays(base, -21), to: addDays(base, 14), start: "13:30", end: "16:00", room: "P.405" },
  ];
  for (const p of plan) {
    const id = ctx.ids[p.code];
    const have = await call("GET", `/courses/${id}/sessions?limit=100`, { token: ctx.tokens.teacher });
    if (have.body.items.length >= p.total) continue;
    await call("POST", `/courses/${id}/sessions/generate`, {
      token: ctx.tokens.teacher,
      key: `seed-sessions-${p.code}-${p.from}`,
      json: { weekdays: [wd], start_time: p.start, end_time: p.end, room: p.room, from: p.from, to: p.to, exclude_dates: [] },
    });
  }
}

async function loginAcceptRosterInvite(email) {
  const user = await userState(email);
  if (!user) throw new Error(`Import chưa tạo tài khoản ${email}`);
  if (user.status === "ACTIVE") return mustLogin(email);
  for (let attempt = 0; attempt < 3; attempt++) {
    const { token } = await waitToken(email, "Bạn được thêm vào lớp", RE_INVITE);
    const acc = await call("POST", "/auth/accept-invite", { json: { token, password: PASSWORD }, ok: [200, 400, 403, 404, 410, 422] });
    if (acc.status === 200) return acc.body;
    // Lời mời cũ không còn dùng được: đăng ký bằng chính email roster làm gateway gửi lại thư cho CHỦ hộp thư (không tạo gì mới).
    await call("POST", "/auth/register", { json: { email, password: PASSWORD, full_name: accounts.get(email.split("@")[0])?.name ?? "Sinh viên" } });
    await sleep(500);
  }
  throw new Error(`Không nhận được lời mời lớp cho ${email}`);
}

async function activeCount(courseId) {
  const r = await call("GET", `/courses/${courseId}/members?limit=1&status=ACTIVE&role=STUDENT`, { token: ctx.tokens.teacher });
  return r.body.counts.active;
}

async function step5Roster() {
  step(5, "Lớp 761987: nhập danh sách 30 sinh viên, nhận lời mời…");
  const id = ctx.ids["761987"];
  if ((await activeCount(id)) < 30) {
    const lines = ["Email,Họ và tên,MSSV", ...roster1.map((k) => `${accounts.get(k).email},${accounts.get(k).name},${accounts.get(k).code}`)];
    const form = new FormData();
    form.append("file", new Blob([lines.join("\n") + "\n"], { type: "text/csv" }), "roster-761987.csv");
    form.append("send_invites", "true");
    const r = await call("POST", `/courses/${id}/roster/import`, { token: ctx.tokens.teacher, form, key: "seed-roster-761987-v1" });
    if (r.body.errors.length > 0) throw new Error(`Import có ${r.body.errors.length} dòng lỗi`);
  }
  for (const k of roster1) {
    const s = await loginAcceptRosterInvite(accounts.get(k).email);
    ctx.tokens[k] = s.access_token;
  }
}

async function registerVerified(k, { verify = true } = {}) {
  const a = accounts.get(k);
  let user = await userState(a.email);
  if (!user) {
    await call("POST", "/auth/register", { json: { email: a.email, password: PASSWORD, full_name: a.name, student_code: a.code } });
    user = { status: "PENDING_VERIFICATION" };
  }
  if (user.status === "PENDING_VERIFICATION" && verify) {
    for (let attempt = 0; attempt < 3; attempt++) {
      const { token } = await waitToken(a.email, "Xác minh email", RE_VERIFY);
      const r = await call("POST", "/auth/verify-email", { json: { token }, ok: [200, 204, 400, 403, 404, 410, 422] });
      if (r.status < 300) break;
      await call("POST", "/auth/resend-verification", { json: { email: a.email }, ok: [200, 202, 204, 429] });
      await sleep(1000);
    }
  }
  const s = await login(a.email);
  if (verify && !s?.user.email_verified) throw new Error(`Chưa xác minh được email của ${k}`);
  return s;
}

async function step6Register() {
  step(6, "Đăng ký, xác minh email và đăng nhập các sinh viên còn lại…");
  for (const k of [...only2, ...pending2, "sv.moi", "sv.lech"]) ctx.tokens[k] = (await registerVerified(k)).access_token;
  await registerVerified("sv.chuaxm", { verify: false }); // cố ý KHÔNG xác minh (vẫn đăng nhập được)
}

async function join(k, code) {
  const r = await call("POST", "/courses/join", { token: ctx.tokens[k], json: { code }, ok: [200, 409] });
  return r;
}

async function step7Class2() {
  step(7, "Lớp 761988: 24 sinh viên vào bằng mã, bật duyệt, 3 sinh viên chờ duyệt…");
  const id = ctx.ids["761988"];
  const code = "BX4P9TW";
  for (const k of [...only2, "sv.gioi", sv(5), sv(6)]) {
    const r = await join(k, code);
    if (r.body?.status === "PENDING") {
      // Lớp đã bật duyệt ở lần chạy trước: người này thuộc 24 sinh viên chính thức ⇒ giảng viên duyệt.
      await call("POST", `/courses/${id}/members/${userId(k)}/approve`, { token: ctx.tokens.teacher, json: {}, ok: [200, 409] });
    }
  }
  const info = await call("GET", `/courses/${id}/join-code`, { token: ctx.tokens.teacher });
  if (!info.body.require_approval) {
    await call("PUT", `/courses/${id}/join-settings`, { token: ctx.tokens.teacher, json: { require_approval: true, version: info.body.version } });
  }
  for (const k of pending2) await join(k, code);
}

const ids = new Map();
function userId(k) {
  return ids.get(k);
}

async function collectIds() {
  for (const k of [...accounts.keys()]) {
    if (!ctx.tokens[k]) continue;
    const me = await call("GET", "/me/profile", { token: ctx.tokens[k] });
    ids.set(k, me.body.id);
  }
}

async function step8Mismatch() {
  step(8, "sv.lech (MSSV trùng B) vào lớp 761987 bằng mã…");
  await join("sv.lech", "AN7K2MQ"); // → PENDING + EMAIL_MISMATCH; sv.moi (D) cố ý không vào lớp nào
}

async function step9Dismiss() {
  step(9, "Giảng viên bỏ qua mục thiết lập của lớp 761987…");
  await call("POST", `/courses/${ctx.ids["761987"]}/setup/dismiss`, { token: ctx.tokens.teacher });
  // Giảng viên chỉ còn MỘT thông báo chưa đọc: nhận lớp 761988. Yêu cầu vào lớp sinh ra khi seed đọc như đã xem.
  // Thông báo JOIN_REQUEST do worker tạo bất đồng bộ (outbox, relay ≈ 0,5 s) SAU bước 7 / 8: chờ đủ (3 chờ duyệt lớp 2 + 1 lệch MSSV lớp 1) rồi mới đánh dấu đọc.
  let n;
  for (let i = 0; i < 40; i++) {
    n = await call("GET", "/notifications?limit=100", { token: ctx.tokens.teacher });
    if (n.body.items.filter((x) => x.type === "JOIN_REQUEST").length >= 4) break;
    await new Promise((r) => setTimeout(r, 500));
  }
  if (n.body.items.filter((x) => x.type === "JOIN_REQUEST").length < 4) throw new Error("Thông báo yêu cầu vào lớp chưa tới sau 20 s — worker có đang chạy không?");
  for (const x of n.body.items) {
    if (!x.read_at && !(x.type === "COURSE_ASSIGNED" && x.course_id === ctx.ids["761988"])) {
      await call("POST", `/notifications/${x.id}/read`, { token: ctx.tokens.teacher });
    }
  }
}


// ---- bước 10: ngân hàng câu hỏi + hai bài thi mẫu (US-PE-09; SRS FEAT-weekly-exam 4.11) -------------------------------------
// Khoá tự nhiên: `title` của câu / bài trong lớp. Mọi thứ đi qua HTTP API thật (không `_test`, không ghi DB). Seed tạo bài (mở sau ~12 s, đóng sau ~150 s), cho sinh viên làm ngay rồi THOÁT:
// bộ lập lịch của worker tự đóng, chấm và công bố. `scripts/check-exam-seed.mjs` chờ và kiểm.
const EXAM_MCQ_TITLE = "Kiểm tra tuần 9 — Mật mã";
const EXAM_CODE_TITLE = "Kiểm tra tuần 9 — Lập trình";

async function listAll(route, token) {
  const out = [];
  let cursor = "";
  for (let i = 0; i < 20; i++) {
    const r = await call("GET", `${route}${route.includes("?") ? "&" : "?"}limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`, { token });
    out.push(...r.body.items);
    if (!r.body.next_cursor) break;
    cursor = r.body.next_cursor;
  }
  return out;
}

async function waitJob(id, token, label, timeoutMs = 120_000) {
  const until = Date.now() + timeoutMs;
  for (;;) {
    const r = await call("GET", `/jobs/${id}`, { token });
    if (r.body.status === "SUCCEEDED") return r.body;
    if (r.body.status === "FAILED") throw new Error(`Việc ${label} lỗi: ${JSON.stringify(r.body.error)}`);
    if (Date.now() > until) throw new Error(`Việc ${label} quá ${timeoutMs / 1000}s (worker / máy chấm chạy chưa?)`);
    await sleep(700);
  }
}

async function approveQuestion(base, id, token) {
  const cur = await call("GET", `${base}/${id}`, { token });
  if (cur.body.review_status === "APPROVED") return;
  await call("PUT", `${base}/${id}/review`, { token, json: { decision: "APPROVE", version: cur.body.version } });
}

async function seedBank(course, token) {
  const base = `/courses/${course}/questions`;
  let have = new Map((await listAll(base, token)).map((x) => [x.title, x]));
  for (const q of QUESTIONS) {
    let row = have.get(q.title);
    if (!row) {
      const body = { type: q.type, title: q.title, topic: q.topic, difficulty: q.difficulty, stem: q.stem, explanation: q.explanation };
      if (q.type === "TRUE_FALSE") body.value = q.value;
      else {
        body.options = q.options.map((b) => ({ body: b }));
        body.correct = q.correct;
      }
      row = (await call("POST", base, { token, key: `seed-q-${q.title}`, json: body })).body;
    }
    await approveQuestion(base, row.id, token);
  }
  // 5 câu AI_DRAFT ở PENDING (nhà cung cấp `fake` của cổng LLM): chỉ gọi khi chưa đủ (khoá mới mỗi lần để không phát lại một việc đã FAILED)
  const all = await listAll(base, token);
  have = new Map(all.map((x) => [x.title, x]));
  const drafts = all.filter((x) => x.origin === "AI_DRAFT"); // theo danh sách, KHÔNG theo Map: `fake` đặt cùng một tiêu đề cho mọi câu
  // nhà cung cấp `fake` trả ĐÚNG 1 câu mỗi lần gọi → mỗi câu thiếu một việc gợi ý (count: 1)
  for (let n = drafts.length; n < 5; n++) {
    const r = await call("POST", `${base}/suggest`, { token, key: `seed-suggest-${Date.now()}-${n}`, json: { kind: "MCQ", topic: "Mạng máy tính", difficulty: "MEDIUM", count: 1 } });
    await waitJob(r.body.job_id, token, "gợi ý câu hỏi", 90_000);
  }
  // hai bài code: 2 test mẫu + 3 test ẩn, lời giải mẫu đã xác minh
  for (const p of CODE) {
    const title = `Bài code — ${p.title}`;
    let row = have.get(title);
    if (row?.review_status === "APPROVED") continue;
    if (!row) row = (await call("POST", base, { token, key: `seed-code-${p.slug}`, json: { type: "CODE", title, topic: p.topic, stem: p.stem } })).body;
    const cur = (await call("GET", `${base}/${row.id}`, { token })).body;
    await call("PUT", `${base}/${row.id}/code`, {
      token,
      json: { languages: p.languages, time_limit_ms: 2000, memory_limit_mb: 128, checker: "EXACT", starter_code: { cpp17: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  return 0;\n}\n" }, reference: { language: "cpp17", source: p.reference }, version: cur.version },
    });
    const existing = await call("GET", `${base}/${row.id}/testcases?limit=100`, { token });
    if (existing.body.items.length < p.tests.length) {
      for (const [i, t] of p.tests.entries()) await call("POST", `${base}/${row.id}/testcases`, { token, json: { name: t.name, input: t.input, expected: t.expected, is_sample: t.sample, weight: p.weights[i] } });
    }
    const all = (await call("GET", `${base}/${row.id}/testcases?limit=100`, { token })).body.items;
    await call("POST", `${base}/${row.id}/testcases/approve`, { token, json: { ids: all.map((t) => t.id) } });
    const v = await call("POST", `${base}/${row.id}/reference/verify`, { token, key: `seed-verify-${p.slug}` });
    const done = await waitJob(v.body.job_id, token, `xác minh lời giải ${p.slug}`);
    if (done.result && done.result.ok === false) throw new Error(`Lời giải mẫu ${p.slug} không đạt: ${JSON.stringify(done.result)}`);
    await approveQuestion(base, row.id, token);
  }
  return new Map((await listAll(base, token)).map((x) => [x.title, x]));
}

const tabId = () => crypto.randomUUID();
const KEYP = "seed-exam-";

async function takeAttempt(course, exam, who, plan) {
  const token = ctx.tokens[who];
  const tab = tabId();
  const base = `/courses/${course}/exams/${exam}/attempts`;
  const hdr = { "X-Exam-Tab": tab };
  const started = await callH("POST", base, { token, key: `${KEYP}start-${exam}-${who}`, headers: hdr });
  const attempt = started.body.attempt.id;
  await plan(started.body, { token, base: `${base}/${attempt}`, hdr });
  await callH("POST", `${base}/${attempt}/submit`, { token, key: `${KEYP}submit-${exam}-${who}`, headers: hdr });
}

/** như `call` nhưng có thêm header tuỳ ý (X-Exam-Tab). */
async function callH(method, route, { token, json, key, headers = {}, ok = [200, 201, 202, 204] }) {
  for (let attempt = 0; ; attempt++) {
    const h = { Accept: "application/json", Origin: ORIGIN, Authorization: `Bearer ${token}`, ...headers };
    if (key) h["Idempotency-Key"] = key.replace(/[^A-Za-z0-9._:-]/g, "-");
    if (json !== undefined) h["Content-Type"] = "application/json";
    const res = await fetch(API + route, { method, headers: h, body: json === undefined ? undefined : JSON.stringify(json), tls: { rejectUnauthorized: false } });
    const body = res.status === 204 ? null : await res.json().catch(() => null);
    if (res.status === 429 && attempt < 5) {
      await sleep(Math.min(20, Number(body?.retry_after ?? res.headers.get("Retry-After") ?? 2)) * 1000 + 200);
      continue;
    }
    if (!ok.includes(res.status)) throw new ApiFail(method, route, res.status, body);
    return { status: res.status, body };
  }
}

async function pool(items, n, fn) {
  const queue = [...items];
  await Promise.all(Array.from({ length: n }, async () => { for (let it = queue.shift(); it !== undefined; it = queue.shift()) await fn(it); }));
}

async function step10Exams() {
  step(10, "Ngân hàng câu hỏi, hai bài thi mẫu và lượt làm (lớp 761987)…");
  const course = ctx.ids["761987"];
  const t = ctx.tokens.teacher;
  const bank = await seedBank(course, t);
  const exams = await listAll(`/courses/${course}/exams`, t);
  if (exams.some((x) => x.title === EXAM_MCQ_TITLE) && exams.some((x) => x.title === EXAM_CODE_TITLE)) {
    console.log("   Đã có hai bài thi mẫu, không tạo thêm.");
    return;
  }
  const idOf = (title) => bank.get(title).id;
  // bài trắc nghiệm: 10 câu, mỗi câu 1 điểm (thang 10) — `points` do hàm createExam đặt 5 cho bài code; ở đây ghi đè bằng items riêng
  const mcqItems = EXAM_MCQ.map((qi) => idOf(QUESTIONS[qi].title));
  const base = `/courses/${course}/exams`;
  const now = Date.now();
  const mcq = (await call("POST", base, { token: t, key: `seed-exam-${EXAM_MCQ_TITLE}`, json: { title: EXAM_MCQ_TITLE, instructions: "Bài kiểm tra mẫu: 10 câu trắc nghiệm.", duration_minutes: 2, multi_scoring: "PARTIAL", opens_at: new Date(now + 14_000).toISOString(), closes_at: new Date(now + 150_000).toISOString() } })).body;
  await call("PUT", `${base}/${mcq.id}/items`, { token: t, json: { items: mcqItems.map((id) => ({ question_id: id, points: "1" })), version: mcq.version } });
  await call("POST", `${base}/${mcq.id}/schedule`, { token: t, json: {}, ok: [200] });
  const code = (await call("POST", base, { token: t, key: `seed-exam-${EXAM_CODE_TITLE}`, json: { title: EXAM_CODE_TITLE, instructions: "Bài kiểm tra mẫu: hai bài lập trình.", duration_minutes: 2, multi_scoring: "PARTIAL", opens_at: new Date(now + 14_000).toISOString(), closes_at: new Date(now + 160_000).toISOString() } })).body;
  await call("PUT", `${base}/${code.id}/items`, { token: t, json: { items: CODE.map((p) => ({ question_id: idOf(`Bài code — ${p.title}`), points: "5" })), version: code.version } });
  await call("POST", `${base}/${code.id}/schedule`, { token: t, json: {}, ok: [200] });
  // chờ tới giờ mở
  const wait = new Date(now + 15_500) - Date.now();
  if (wait > 0) await sleep(wait);

  // 24 lượt trắc nghiệm theo mẫu cố định
  await pool(MCQ_STUDENTS, 6, (s) =>
    takeAttempt(course, mcq.id, s.who, async (start, { token, base: ab, hdr }) => {
      const answers = [];
      for (const a of mcqAnswers(s.id)) {
        const q = QUESTIONS[a.qi];
        const item = start.items.find((x) => x.stem === q.stem);
        if (!item || a.ans == null) continue;
        answers.push({ item_id: item.item_id, answer: q.type === "TRUE_FALSE" ? { value: a.ans.value } : { option_ids: a.ans.idx.map((i) => item.options.find((o) => o.body === q.options[i]).id) } });
      }
      await callH("PUT", `${ab}/answers`, { token, headers: hdr, json: { items: answers } });
    }),
  );
  // 6 lượt code
  await pool(CODE_ATTEMPTS, 6, (c) =>
    takeAttempt(course, code.id, c.who, async (start, { token, base: ab, hdr }) => {
      for (const [i, sol] of [c.gcd, c.words].entries()) {
        if (!sol) continue;
        const item = start.items.find((x) => x.stem.startsWith(CODE[i].stem.slice(0, 20)));
        await callH("POST", `${ab}/code/${item.item_id}/submit`, { token, headers: hdr, key: `${KEYP}code-${code.id}-${c.who}-${i}`, json: { language: "cpp17", source: SOLUTIONS[sol] } });
        await sleep(i === 0 ? 15_500 : 0); // khoảng chờ giữa hai lần nộp của một sinh viên (EXAM_SUBMIT_COOLDOWN 15 s)
      }
    }),
  );
  ctx.changed = true;
  console.log("   Đã nộp 24 lượt trắc nghiệm + 6 lượt code; bộ lập lịch sẽ đóng và công bố (≤ 3 phút).");
}

// ---- bước 11: tài liệu + lịch (US-P8-03 AC17) ------------------------------------------------------------------------------
// Khoá tự nhiên: `title` của tài liệu / sự kiện trong lớp. Đi qua presign → PUT thẳng kho tệp → complete → việc nền (docling thật).
const DOCS = [
  { file: "Mordern_Network_Security_Threats.pdf", title: "Network Security Threats", type: "LECTURE", week: 3 },
  { file: "QMB12ch6b.pdf", title: "Forecasting (QMB ch. 6b)", type: "LECTURE", week: 6 },
  { file: "Quyche.pdf", title: "Quy chế học vụ", type: "COURSE_POLICY", week: null },
  { gen: EXAM_PAPER_LINES, title: "Đề tham khảo tuần 5", type: "EXAM_PAPER", week: 5, name: "de-tham-khao-tuan-5.pdf" },
  { gen: ANSWER_KEY_LINES, title: "Đáp án đề tham khảo tuần 5", type: "ANSWER_KEY", week: 5, name: "dap-an-tuan-5.pdf" },
];

async function uploadDoc(course, token, d) {
  const bytes = d.gen ? miniPdf(d.gen) : readFileSync(path.join(root, "seed", "documents", d.file));
  const sha = createHash("sha256").update(bytes).digest("hex");
  const filename = d.name ?? d.file;
  const pre = await call("POST", `/courses/${course}/uploads/presign`, { token, json: { purpose: "document", filename, mime_type: "application/pdf", size_bytes: bytes.length, sha256: sha } });
  const put = await fetch(pre.body.url, { method: pre.body.method, headers: pre.body.headers, body: bytes });
  if (!put.ok) throw new Error(`PUT tệp ${filename} → ${put.status}`);
  const json = { upload_id: pre.body.upload_id, title: d.title, type: d.type, ...(d.week ? { week_no: d.week } : {}) };
  const done = await call("POST", `/courses/${course}/uploads/complete`, { token, json, key: `seed-doc-${sha.slice(0, 16)}`, ok: [200, 202, 409] });
  return done.status === 409 ? null : done.body.job_id;
}

async function step11DocsEvents() {
  step(11, "Tài liệu (docling thật) và sự kiện lịch…");
  const c1 = ctx.ids["761987"], c2 = ctx.ids["761988"];
  const t = ctx.tokens.teacher;
  const have = new Map((await listAll(`/courses/${c1}/documents`, t)).map((d) => [d.title, d]));
  const todo = DOCS.filter((d) => !have.has(d.title));
  const jobs = [];
  await pool(todo, 1, async (d) => { // lần lượt: docling một luồng, nên đo được giây / trang từng tệp (`check-docs-seed.mjs timing`)
    const job = await uploadDoc(c1, t, d);
    if (job) jobs.push([job, d.title]);
    ctx.changed = true;
  });
  for (const [job, title] of jobs) await waitJob(job, t, `đọc tài liệu "${title}"`, 360_000);
  const docs = await listAll(`/courses/${c1}/documents`, t);
  const notReady = DOCS.filter((d) => docs.find((x) => x.title === d.title)?.status !== "READY");
  if (notReady.length > 0) throw new Error(`Tài liệu chưa READY: ${notReady.map((d) => d.title).join(", ")} (docling chạy chưa? \`docker compose --profile ingest up\`)`);
  const shared = await listAll(`/courses/${c2}/documents`, t);
  if (!shared.some((d) => d.title === DOCS[0].title)) {
    await call("POST", `/courses/${c2}/share-from`, { token: t, json: { source_course_id: c1, what: ["documents"] }, key: "seed-share-docs-761987-761988" });
    ctx.changed = true;
  }
  const base = baseDate();
  for (const [code, list] of Object.entries(EVENTS)) {
    const course = ctx.ids[code];
    const from = new Date(Date.now() - 86_400_000).toISOString();
    const to = new Date(Date.now() + 61 * 86_400_000).toISOString();
    const exist = new Set((await listAll(`/courses/${course}/calendar?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`, t)).map((e) => e.title));
    for (const e of list) {
      if (exist.has(e.title)) continue;
      const startsAt = new Date(`${addDays(base, e.days)}T${e.at}:00+07:00`).toISOString();
      await call("POST", `/courses/${course}/calendar/events`, { token: t, json: { type: e.type, title: e.title, starts_at: startsAt, location: e.location } });
      ctx.changed = true;
    }
  }
}

// ---- bước 12: ≈ 150 câu hỏi chat riêng bằng đúng luồng chat (US-P3-08 AC1) --------------------------------------------------
// Mỗi sinh viên của lớp 761987 một phiên (sv.gioi: 4 phiên, để "Tiếp tục học" có > 3 mục); chạy lại chỉ bổ sung phần thiếu.
const uuidOf = (s) => {
  const h = createHash("sha1").update(s).digest("hex");
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-5${h.slice(13, 16)}-a${h.slice(17, 20)}-${h.slice(20, 32)}`;
};

/** Gửi một tin và đọc SSE tới hết (done | error). 409 EXAM_IN_PROGRESS (đang có lượt thi): đợi tối đa 4 phút. */
async function sendChat(token, sid, content, key) {
  for (let attempt = 0; ; attempt++) {
    const res = await fetch(`${API}/chat/sessions/${sid}/messages`, {
      method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", Accept: "text/event-stream", Origin: ORIGIN, "Idempotency-Key": uuidOf(key) },
      body: JSON.stringify({ content }), tls: { rejectUnauthorized: false },
    });
    if (res.status === 429 && attempt < 8) {
      await sleep(3000);
      continue;
    }
    if (res.status === 409 && attempt < 50) {
      await sleep(5000);
      continue;
    }
    if (!res.ok) throw new Error(`POST messages → ${res.status} ${(await res.text()).slice(0, 120)}`);
    const text = await res.text();
    if (/event: error/.test(text)) throw new Error(`Chat báo lỗi: ${text.slice(text.indexOf("event: error"), text.indexOf("event: error") + 160)}`);
    return;
  }
}

async function userMessageCount(token, sid) {
  const items = await listAll(`/chat/sessions/${sid}/messages`, token);
  return items.filter((m) => m.role === "USER").length;
}

async function step12Chat() {
  step(12, "Câu hỏi chat riêng của sinh viên lớp 761987 (≈ 150, bằng luồng chat thật)…");
  const course = ctx.ids["761987"];
  const questions = chatQuestions(mulberry32(RNG_SEED ^ 0x5eed));
  const students = roster1;
  // kế hoạch: sv.gioi 4 phiên × 2 tin, mỗi người còn lại 1 phiên × 5 tin → 8 + 145 = 153
  const plan = students.map((k) => ({ k, sessions: k === "sv.gioi" ? [2, 2, 2, 2] : [5] }));
  let n = 0;
  const jobs = plan.map((p) => ({ ...p, from: (n += p.sessions.reduce((a, b) => a + b, 0)) - p.sessions.reduce((a, b) => a + b, 0) }));
  await pool(jobs, 4, async (j) => {
    const token = (await login(accounts.get(j.k).email)).access_token;
    const existing = (await listAll(`/chat/sessions?course_id=${course}`, token)).sort((a, b) => (a.id < b.id ? -1 : 1));
    let q = j.from;
    for (const [si, quota] of j.sessions.entries()) {
      let sess = existing[si];
      const had = sess ? await userMessageCount(token, sess.id) : 0;
      if (had >= quota) {
        q += quota;
        continue;
      }
      if (!sess) {
        sess = (await call("POST", "/chat/sessions", { token, json: { course_id: course }, key: `seed-chat-session-${j.k}-${si}` })).body;
        ctx.changed = true;
      }
      for (let m = had; m < quota; m++) {
        await sendChat(token, sess.id, questions[(q + m) % questions.length], `seed-chat-${j.k}-${si}-${m}`);
        ctx.changed = true;
      }
      q += quota;
    }
  });
}

// ---- bước 13: thread lớp (12 + 3) với đủ trạng thái bài AI (US-P3-08 AC2) ---------------------------------------------------
async function waitThreadAI(course, tid, token) {
  const until = Date.now() + 120_000;
  for (;;) {
    const v = (await call("GET", `/courses/${course}/threads/${tid}`, { token })).body;
    const ai = v.posts.find((p) => p.kind === "AI");
    if (ai || v.thread.ai_state === "SKIPPED") return { thread: v.thread, ai };
    if (Date.now() > until) throw new Error(`AI chưa trả lời thread ${tid} sau 120 s (consumer ep:ingest / worker chạy chưa?)`);
    await sleep(1500);
  }
}

async function seedThreads(course, plan, token) {
  const have = new Map((await listAll(`/courses/${course}/threads`, token)).map((t) => [t.title, t]));
  const out = [];
  for (const th of plan) {
    let row = have.get(th.title);
    if (!row) {
      const author = (await login(accounts.get(th.k).email)).access_token;
      row = (await call("POST", `/courses/${course}/threads`, { token: author, json: { title: th.title, body: th.body, week_no: th.week ?? null, tags: [] }, key: `seed-thread-${course}-${th.title}` })).body;
      ctx.changed = true;
    }
    out.push({ th, id: row.id });
  }
  return out;
}

async function step13Threads() {
  step(13, "Thread lớp 761987 (12) và 761988 (3), đủ trạng thái bài AI…");
  const t = ctx.tokens.teacher;
  const c1 = ctx.ids["761987"], c2 = ctx.ids["761988"];
  const made1 = await seedThreads(c1, THREADS_1, t);
  await seedThreads(c2, THREADS_2, t);
  const warn = [];
  for (const { th, id } of made1) {
    const { thread, ai } = await waitThreadAI(c1, id, t);
    if (th.want === "skip") {
      if (thread.ai_state !== "SKIPPED") warn.push(`"${th.title}" đáng lẽ AI bỏ qua nhưng đã trả lời`);
      continue;
    }
    if (!ai) {
      warn.push(`"${th.title}": AI bỏ qua (${thread.ai_state}) — không áp được quyết định ${th.want}`);
      continue;
    }
    if (ai.verification_state !== "PENDING" || th.want === "pending") continue; // đã quyết ở lần chạy trước, hoặc để chờ
    const base = `/courses/${c1}/posts/${ai.id}`;
    if (th.want === "verify") await call("POST", `${base}/verify`, { token: t });
    if (th.want === "correct") await call("POST", `${base}/correct`, { token: t, json: { body: `${ai.body}\n\nBổ sung của giảng viên: xem lại tài liệu tuần ${th.week}.`, version: ai.version } });
    if (th.want === "reject") await call("POST", `${base}/reject`, { token: t });
    ctx.changed = true;
  }
  for (const w of warn) console.log(`   CẢNH BÁO seed thread: ${w}`);
}

// ---- chạy ------------------------------------------------------------------------------------------------------------
async function main() {
  const started = Date.now();
  if (IF_EMPTY && (await isSeeded())) {
    console.log("Đã có dữ liệu, không seed.");
    return;
  }
  await step1Admin();
  await step2Staff();
  await step3Courses();
  await step4Sessions();
  await step5Roster();
  await step6Register();
  await collectIds();
  await step7Class2();
  await step8Mismatch();
  await step9Dismiss();
  await step10Exams();
  await step11DocsEvents();
  await step12Chat();
  await step13Threads();
  console.log(`Seed xong trong ${Math.round((Date.now() - started) / 1000)} s.`);
  if (!ctx.changed) console.log("Seed xong (không đổi)");
  console.log("  2 lớp: 761987 (mã AN7K2MQ, 30 sinh viên + 1 chờ duyệt EMAIL_MISMATCH), 761988 (mã BX4P9TW, 24 sinh viên + 3 chờ duyệt, tối đa 30)");
  console.log("  60 tài khoản (1 Admin, 1 giảng viên, 1 TA, 57 sinh viên); mật khẩu = SEED_DEFAULT_PASSWORD");
  console.log("  Lớp 761987: 20 câu trắc nghiệm APPROVED + 5 câu AI_DRAFT + 2 bài code; hai bài thi mẫu (24 lượt trắc nghiệm + 6 lượt code) tự đóng và công bố ≤ 3 phút — `node scripts/check-exam-seed.mjs bank|scores|demo`");
  console.log("  Lớp 761987: 5 tài liệu READY (1 quy chế scan, 2 bài giảng, 1 đề tham khảo, 1 đáp án canary) + 2 sự kiện thi; lớp 761988: bài giảng dùng chung + 1 sự kiện — `node scripts/check-docs-seed.mjs`");
  console.log("  ≈ 150 câu hỏi chat riêng (sinh viên lớp 761987), 12 thread lớp 761987 + 3 thread lớp 761988 — `node scripts/check-chat-seed.mjs [threads]`");
  console.log("  Tài khoản mẫu: admin@ teacher@ ta@ sv.gioi@ sv.kha@ sv.nguyco@ sv.moi@ (@edupilot.local)");
}

main().catch((e) => {
  console.error("Seed lỗi:", mask(e.message));
  process.exit(1);
});
