// Kiểm dữ liệu mẫu của Luyện đề (US-P9-12) bằng API thật, từng tài khoản.
//   node scripts/check-practice-seed.mjs     (cần API_URL, SEED_DEFAULT_PASSWORD như seed.mjs)
// Kiểm: ≥ 80 câu APPROVED (đủ loại, đủ độ khó) + 20 PENDING; Quiz xếp lịch; sinh viên thấy lượt + điểm yếu của MÌNH; B có điểm yếu An toàn ứng dụng web;
// C có lượt đang dở; giảng viên / TA KHÔNG đọc được lượt của sinh viên (403); sinh viên không gọi được /questions (403).
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
  const res = await fetch(API + route, { method, headers, body: json === undefined ? undefined : JSON.stringify(json) });
  return { status: res.status, body: res.status === 204 ? null : await res.json().catch(() => null) };
}
const tokens = {};
async function tok(key) {
  if (tokens[key]) return tokens[key];
  const r = await call("POST", "/auth/login", { json: { email: key + DOMAIN, password: PASSWORD } });
  if (r.status !== 200) throw new Error(`Không đăng nhập được ${key}: ${r.status}`);
  return (tokens[key] = r.body.access_token);
}
async function listAll(route, token) {
  const out = [];
  let cursor = "";
  for (let i = 0; i < 30; i++) {
    const r = await call("GET", `${route}${route.includes("?") ? "&" : "?"}limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`, { token });
    if (r.status !== 200) throw new Error(`${route} → ${r.status}`);
    out.push(...r.body.items);
    if (!r.body.next_cursor) break;
    cursor = r.body.next_cursor;
  }
  return out;
}

const teacher = await tok("teacher");
const courses = (await call("GET", "/me/courses", { token: teacher })).body;
const course = (courses.items ?? courses).find((c) => (c.course ?? c).class_code === "761987")?.course?.id;
if (!course) throw new Error("Không thấy lớp 761987");
const Q = `/courses/${course}/questions`;

const bank = await listAll(Q, teacher);
const approved = bank.filter((q) => q.review_status === "APPROVED" && !q.archived_at);
const pending = bank.filter((q) => q.review_status === "PENDING");
must(approved.length >= 80, `APPROVED ${approved.length} < 80`);
must(pending.length >= 20, `PENDING ${pending.length} < 20`);
for (const t of ["MCQ_SINGLE", "MCQ_MULTI", "TRUE_FALSE", "SHORT", "ESSAY"]) must(approved.some((q) => q.type === t), `thiếu loại ${t} APPROVED`);
for (const d of ["EASY", "MEDIUM", "HARD"]) must(approved.some((q) => q.difficulty === d), `thiếu độ khó ${d}`);
const topics = new Set(approved.map((q) => q.topic));
must(topics.size >= 8, `chỉ ${topics.size} chủ đề`);

const exams = await listAll(`/courses/${course}/exams`, teacher);
const quiz = exams.find((e) => e.title.startsWith("Quiz 01"));
must(quiz && ["SCHEDULED", "OPEN"].includes(quiz.effective_status), "Quiz 01 chưa xếp lịch");

// sinh viên thấy lượt của chính mình
let total = 0;
for (const who of ["sv.gioi", "sv.kha", "sv.nguyco", "sv04", "sv07", "sv12"]) {
  const t = await tok(who);
  const h = await listAll(`/courses/${course}/practice/history`, t);
  total += h.length;
  must(h.length >= 3, `${who}: ${h.length} lượt < 3`);
  const ov = await call("GET", `/courses/${course}/practice/overview`, { token: t });
  must(ov.status === 200 && ov.body.topics.length >= 5, `${who}: overview thiếu chủ đề`);
}
must(total >= 18, `tổng lượt của 6 sinh viên ${total} < 18`);
const B = await tok("sv.kha");
const wk = await call("GET", `/courses/${course}/practice/weakness`, { token: B });
must(wk.status === 200 && wk.body.topics.some((x) => x.topic === "An toàn ứng dụng web"), "B không có điểm yếu An toàn ứng dụng web");
const hB = await listAll(`/courses/${course}/practice/history`, B);
must(hB.some((r) => r.mode === "MOCK" && r.status !== "IN_PROGRESS"), "B thiếu lượt thi thử đã nộp");
const C = await tok("sv.nguyco");
must((await call("GET", `/courses/${course}/practice/overview`, { token: C })).body.in_progress.length >= 1, "C không có lượt đang dở");

// riêng tư: giảng viên / TA không đọc được lượt của sinh viên; sinh viên không vào /questions
const attemptB = hB[0].id;
for (const who of ["teacher", "ta"]) {
  const r = await call("GET", `/courses/${course}/practice/attempts/${attemptB}`, { token: await tok(who) });
  must(r.status === 403, `${who} đọc lượt của sinh viên → ${r.status} (cần 403)`);
}
must((await call("GET", Q, { token: B })).status === 403, "sinh viên gọi /questions không bị 403");
const other = await call("GET", `/courses/${course}/practice/attempts/${attemptB}`, { token: await tok("sv.gioi") });
must(other.status === 404, `sinh viên khác đọc lượt của B → ${other.status} (cần 404)`);

if (failures.length) {
  console.error("LỖI:\n- " + failures.join("\n- "));
  process.exit(1);
}
console.log(`OK: ${approved.length} APPROVED, ${pending.length} PENDING, Quiz ${quiz.effective_status}, ${total} lượt của 6 sinh viên mẫu.`);
