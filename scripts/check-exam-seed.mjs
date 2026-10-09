// Kiểm dữ liệu mẫu của US-PE-09 bằng API thật, từng tài khoản (SRS FEAT-weekly-exam 4.11).
//   node scripts/check-exam-seed.mjs bank|scores|demo     (cần API_URL, SEED_DEFAULT_PASSWORD như seed.mjs)
//   bank   : 20 câu trắc nghiệm APPROVED (12/4/4), 5 AI_DRAFT PENDING, 2 bài code (2 mẫu + 3 ẩn, trọng số, lời giải đã xác minh); lớp 761988 không có.
//   scores : chờ hai bài PUBLISHED rồi so `auto_score` với seed/expected_exam_scores.csv từng dòng và verdict từng bài code → `matched 30/30`.
//   demo   : A thấy điểm cao nhất + đáp án; C "không làm bài"; D chưa vào lớp không có bài thi; GV có bảng điểm, phân bố, câu đúng ≤ 40 %, ĐÚNG 1 cặp giống nhau + việc "Hôm nay";
//            TA không thấy `flags`; sinh viên nộp CE thấy lỗi biên dịch ở kết quả của chính mình.
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
function envFileValue(key) {
  for (const f of [".env.local", ".env.example"]) {
    const p = path.join(root, f);
    if (!existsSync(p)) continue;
    const line = readFileSync(p, "utf8").split("\n").find((l) => l.startsWith(`${key}=`));
    if (line) return line.slice(key.length + 1).trim();
  }
  return undefined;
}
const setting = (k, d) => process.env[k] || envFileValue(k) || d;
const API = setting("API_URL", "https://localhost/api/v1").replace(/\/$/, "");
const PASSWORD = setting("SEED_DEFAULT_PASSWORD", "");
if (/^https:\/\/(localhost|127\.0\.0\.1)/.test(API)) process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
const ORIGIN = new URL(API).origin;
const DOMAIN = "@edupilot.local";
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const failures = [];
const must = (cond, msg) => {
  if (!cond) failures.push(msg);
  return cond;
};

async function call(method, route, { token, json } = {}) {
  const headers = { Accept: "application/json", Origin: ORIGIN };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (json !== undefined) headers["Content-Type"] = "application/json";
  if (method === "POST") headers["Idempotency-Key"] = `chk-${crypto.randomUUID()}`;
  for (let i = 0; ; i++) {
    const res = await fetch(API + route, { method, headers, body: json === undefined ? undefined : JSON.stringify(json) });
    const body = res.status === 204 ? null : await res.json().catch(() => null);
    if (res.status === 429 && i < 4) {
      await sleep(2000);
      continue;
    }
    return { status: res.status, body };
  }
}
const tokens = {};
async function login(key) {
  if (tokens[key]) return tokens[key];
  const r = await call("POST", "/auth/login", { json: { email: key + DOMAIN, password: PASSWORD } });
  if (r.status !== 200) throw new Error(`Không đăng nhập được ${key}: ${r.status}`);
  tokens[key] = r.body;
  return r.body;
}
const tok = async (k) => (await login(k)).access_token;
async function listAll(route, token) {
  const out = [];
  let cursor = "";
  for (let i = 0; i < 30; i++) {
    const r = await call("GET", `${route}${route.includes("?") ? "&" : "?"}limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`, { token });
    if (r.status !== 200) throw new Error(`GET ${route} → ${r.status}`);
    out.push(...r.body.items);
    if (!r.body.next_cursor) break;
    cursor = r.body.next_cursor;
  }
  return out;
}
async function courseIds() {
  const t = await tok("teacher");
  const r = await call("GET", "/me/courses?limit=50", { token: t });
  const by = Object.fromEntries(r.body.items.map((x) => [x.course.class_code, x.course.id]));
  return { c1: by["761987"], c2: by["761988"], t };
}

async function bank() {
  const { c1, c2, t } = await courseIds();
  const qs = await listAll(`/courses/${c1}/questions`, t);
  const count = (f) => qs.filter(f).length;
  must(count((q) => q.type === "MCQ_SINGLE" && q.review_status === "APPROVED" && q.origin === "MANUAL") === 12, "MCQ_SINGLE APPROVED ≠ 12");
  must(count((q) => q.type === "MCQ_MULTI" && q.review_status === "APPROVED" && q.origin === "MANUAL") === 4, "MCQ_MULTI APPROVED ≠ 4");
  must(count((q) => q.type === "TRUE_FALSE" && q.review_status === "APPROVED" && q.origin === "MANUAL") === 4, "TRUE_FALSE APPROVED ≠ 4");
  must(count((q) => q.origin === "AI_DRAFT" && q.review_status === "PENDING") === 5, `AI_DRAFT PENDING ≠ 5 (có ${count((q) => q.origin === "AI_DRAFT")})`);
  const codes = qs.filter((q) => q.type === "CODE" && q.review_status === "APPROVED" && q.title.startsWith("Bài code — ")); // bỏ câu "Tải k6 …" do kịch bản tải tạo thêm
  must(codes.length === 2, `bài code APPROVED ≠ 2 (có ${codes.length})`);
  const wantWeights = { "Bài code — Ước chung lớn nhất": [1, 1, 2, 2, 2], "Bài code — Đếm từ": [1, 1, 1, 1, 1] };
  for (const c of codes) {
    const tests = (await call("GET", `/courses/${c1}/questions/${c.id}/testcases?limit=100`, { token: t })).body.items;
    must(tests.length === 5, `${c.title}: ${tests.length} test (cần 5)`);
    must(tests.filter((x) => x.is_sample).length === 2, `${c.title}: số test mẫu ≠ 2`);
    must(JSON.stringify(tests.map((x) => x.weight)) === JSON.stringify(wantWeights[c.title]), `${c.title}: trọng số ${tests.map((x) => x.weight)}`);
    must(tests.every((x) => x.approved), `${c.title}: còn test chưa duyệt`);
    const d = (await call("GET", `/courses/${c1}/questions/${c.id}`, { token: t })).body;
    must(d.code && d.code.reference_verified_version != null, `${c.title}: lời giải mẫu chưa xác minh`);
  }
  const other = await call("GET", `/courses/${c2}/questions?limit=100`, { token: t });
  must(other.status === 200 && other.body.items.length === 0, `lớp 761988 phải KHÔNG có câu hỏi (có ${other.body?.items?.length})`);
}

function parseCsv() {
  const lines = readFileSync(path.join(root, "seed", "expected_exam_scores.csv"), "utf8").trim().split("\n").slice(1);
  return lines.map((l) => {
    const [exam, account, label, kind, score, formula, verdict] = l.split(",");
    return { exam, account, label, kind, score, formula, verdict };
  });
}

async function exams(c1, t) {
  const list = await listAll(`/courses/${c1}/exams`, t);
  return { mcq: list.find((e) => e.title === "Kiểm tra tuần 9 — Mật mã"), code: list.find((e) => e.title === "Kiểm tra tuần 9 — Lập trình") };
}

async function waitPublished(c1, t, timeoutMs = 4 * 60_000) {
  const until = Date.now() + timeoutMs;
  for (;;) {
    const e = await exams(c1, t);
    if (e.mcq?.status === "PUBLISHED" && e.code?.status === "PUBLISHED") return e;
    if (Date.now() > until) throw new Error(`Hai bài thi chưa PUBLISHED sau ${timeoutMs / 1000}s: ${e.mcq?.status} / ${e.code?.status}`);
    await sleep(3000);
  }
}

async function accountIds(keys) {
  const admin = await tok("admin");
  const out = {};
  for (const k of keys) {
    const r = await call("GET", `/admin/users?q=${encodeURIComponent(k + DOMAIN)}&limit=10`, { token: admin });
    out[k] = r.body.items.find((u) => u.email === k + DOMAIN)?.id;
  }
  return out;
}

async function scores() {
  const { c1, t } = await courseIds();
  const e = await waitPublished(c1, t);
  const exp = parseCsv();
  const ids = await accountIds([...new Set(exp.map((x) => x.account))]);
  let matched = 0;
  for (const [kind, ex] of [["MCQ", e.mcq], ["CODE", e.code]]) {
    const rows = await listAll(`/courses/${c1}/exams/${ex.id}/results`, t);
    const byId = new Map(rows.map((r) => [r.student.id, r]));
    for (const x of exp.filter((r) => r.kind === kind)) {
      const row = byId.get(ids[x.account]);
      const got = row?.auto_score;
      if (!must(row && got === x.score, `${kind} ${x.account} (${x.label}): máy ${got ?? "—"} ≠ bảng tính ${x.score}  [${x.formula}]`)) continue;
      if (kind === "CODE") {
        const d = (await call("GET", `/courses/${c1}/exams/${ex.id}/results/${row.attempt_id}`, { token: t })).body;
        const items = d.items.filter((i) => i.type === "CODE");
        const got2 = x.verdict.split(";").map((v, i) => {
          const slug = v.split(":")[0];
          const idx = slug === "gcd-lon-nhat" ? 0 : 1;
          const it = items.find((i2) => i2.stem.startsWith(idx === 0 ? "Cho hai số nguyên" : "Đọc một dòng"));
          const subs = d.submissions.filter((s) => s.item_id === it?.item_id);
          const last = subs[subs.length - 1];
          return `${slug}:${last ? last.verdict : "NONE"}`;
        });
        if (!must(got2.join(";") === x.verdict, `CODE ${x.account}: verdict ${got2.join(";")} ≠ ${x.verdict}`)) continue;
      }
      matched++;
    }
  }
  console.log(`matched ${matched}/${exp.length}`);
  if (matched !== exp.length) failures.push(`chỉ khớp ${matched}/${exp.length}`);
}

async function demo() {
  const { c1, t } = await courseIds();
  const e = await waitPublished(c1, t);
  const ids = await accountIds(["sv.gioi", "sv.nguyco", "sv04", "sv.moi"]);
  // A: điểm cao nhất + đáp án
  const a = await tok("sv.gioi");
  const mine = await call("GET", `/courses/${c1}/exams/${e.mcq.id}/attempts/mine`, { token: a });
  const res = await call("GET", `/courses/${c1}/exams/${e.mcq.id}/attempts/${mine.body.attempt?.id}/result`, { token: a });
  must(res.status === 200 && res.body.score === "10.00", `A: điểm ${res.body?.score} (cần 10.00)`);
  must(res.body?.items?.every((i) => i.answer), "A: thiếu đáp án (reveal_answers)");
  const rows = await listAll(`/courses/${c1}/exams/${e.mcq.id}/results`, t);
  const top = Math.max(...rows.filter((r) => r.score).map((r) => Number(r.score)));
  must(top === 10, `điểm cao nhất của bảng ${top} ≠ 10`);
  // C không làm
  const cMine = await call("GET", `/courses/${c1}/exams/${e.mcq.id}/attempts/mine`, { token: await tok("sv.nguyco") });
  must(cMine.status === 200 && cMine.body.attempt === null, "C phải 'không làm bài' (attempt null)");
  must(rows.find((r) => r.student.id === ids["sv.nguyco"])?.status === "ABSENT", "C phải là ABSENT ở bảng điểm");
  // D chưa vào lớp: không có bài thi nào
  const dTok = await tok("sv.moi");
  const dc = await call("GET", "/me/courses?limit=50", { token: dTok });
  must(dc.body.items.length === 0, `D thuộc ${dc.body.items.length} lớp (cần 0)`);
  const dEx = await call("GET", `/courses/${c1}/exams?limit=50`, { token: dTok });
  must(dEx.status === 403 || dEx.status === 404, `D xem được bài thi (${dEx.status})`);
  // Giảng viên: phân bố, câu sai nhiều, cặp giống nhau
  const stats = (await call("GET", `/courses/${c1}/exams/${e.mcq.id}/stats`, { token: t })).body;
  must(stats.distribution.reduce((n, b) => n + b.count, 0) === 24, "phân bố: tổng ≠ 24 lượt");
  must(stats.hardest.some((h) => Number(h.correct_rate) <= 0.4), "Câu sai nhiều: không có câu đúng ≤ 40 %");
  const pairs = await listAll(`/courses/${c1}/exams/${e.code.id}/similarity?flagged=true`, t);
  must(pairs.length === 1, `cặp giống nhau gắn cờ: ${pairs.length} (cần ĐÚNG 1)`);
  const today = await call("GET", `/courses/${c1}/today`, { token: t });
  must(JSON.stringify(today.body).includes("EXAM_SIMILARITY"), "Hôm nay của giảng viên thiếu việc EXAM_SIMILARITY");
  // TA: có bảng điểm, không `flags`
  const ta = await tok("ta");
  const taRows = await listAll(`/courses/${c1}/exams/${e.mcq.id}/results`, ta);
  must(taRows.length > 0 && taRows.every((r) => !("flags" in r)), "TA thấy `flags` hoặc không có bảng điểm");
  // sinh viên nộp CE thấy lỗi biên dịch
  const s4 = await tok("sv04");
  const m4 = await call("GET", `/courses/${c1}/exams/${e.code.id}/attempts/mine`, { token: s4 });
  const r4 = await call("GET", `/courses/${c1}/exams/${e.code.id}/attempts/${m4.body.attempt?.id}/result`, { token: s4 });
  must(r4.body?.items?.some((i) => i.final_submission && i.final_submission.compile_ok === false && i.compile_log), "sv04 không thấy lỗi biên dịch ở kết quả của mình");
  void ids;
}

const mode = process.argv[2];
const run = { bank, scores, demo }[mode];
if (!run) {
  console.error("Cách dùng: node scripts/check-exam-seed.mjs bank|scores|demo");
  process.exit(2);
}
if (!PASSWORD) {
  console.error("Thiếu SEED_DEFAULT_PASSWORD");
  process.exit(2);
}
try {
  await run();
} catch (e) {
  failures.push(e.message);
}
if (failures.length) {
  console.error(`✗ ${mode}:\n  - ${failures.join("\n  - ")}`);
  process.exit(1);
}
console.log(`OK ${mode}`);
