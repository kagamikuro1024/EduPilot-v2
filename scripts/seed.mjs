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
const STEPS = 9;

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
      : spawnSync("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", "edupilot", "exec", "-T", "gateway", "/gateway", "admin", "create", ...args], { cwd: root, input: PASSWORD + "\n", encoding: "utf8" });
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
  const n = await call("GET", "/notifications?limit=100", { token: ctx.tokens.teacher });
  for (const x of n.body.items) {
    if (!x.read_at && !(x.type === "COURSE_ASSIGNED" && x.course_id === ctx.ids["761988"])) {
      await call("POST", `/notifications/${x.id}/read`, { token: ctx.tokens.teacher });
    }
  }
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
  console.log(`Seed xong trong ${Math.round((Date.now() - started) / 1000)} s.`);
  console.log("  2 lớp: 761987 (mã AN7K2MQ, 30 sinh viên + 1 chờ duyệt EMAIL_MISMATCH), 761988 (mã BX4P9TW, 24 sinh viên + 3 chờ duyệt, tối đa 30)");
  console.log("  60 tài khoản (1 Admin, 1 giảng viên, 1 TA, 57 sinh viên); mật khẩu = SEED_DEFAULT_PASSWORD");
  console.log("  Tài khoản mẫu: admin@ teacher@ ta@ sv.gioi@ sv.kha@ sv.nguyco@ sv.moi@ (@edupilot.local)");
}

main().catch((e) => {
  console.error("Seed lỗi:", mask(e.message));
  process.exit(1);
});
