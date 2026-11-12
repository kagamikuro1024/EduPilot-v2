#!/usr/bin/env node
// Sinh dữ liệu T1 (SYSTEM_DESIGN §1.2; FEAT-nfr-eval US-NFR-03): 1.000 SV · 20 lớp (≈ 50 SV/lớp) · 5 TA · 3 GV · một học kỳ 15 tuần.
// Ghi THẲNG vào Postgres bằng SQL (generate_series, tất định theo SEED_RNG) — không qua API nghiệp vụ nên không chạy được ở dev-seed (`seed.mjs` ~ 60 tài khoản).
// Chỉ chạy trên stack đo -p ep-* có DB TÊN `edupilot_t1` (không phải `edupilot` của demo):
//   POSTGRES_DB=edupilot_t1 DATABASE_URL=postgres://edupilot:edupilot-dev@postgres:5432/edupilot_t1?sslmode=disable \
//   PGBOUNCER_URL=postgres://edupilot:edupilot-dev@pgbouncer:6432/edupilot_t1?sslmode=disable COMPOSE_PROJECT_NAME=ep-t1 EP_PORT_OFFSET=400 pnpm dev
//   (DATABASE_URLS của pgbouncer đọc POSTGRES_DB nên pgbouncer cũng trỏ vào edupilot_t1.)
// Cần admin đã có (`gateway admin create` hoặc bước 1 của seed.mjs): mật khẩu mọi tài khoản T1 = mật khẩu của admin (băm bcrypt đúng thuật toán của `auth`, dùng lại một bản băm).
//   COMPOSE_PROJECT_NAME=ep-t1 SEED_DEFAULT_PASSWORD=… node scripts/seed-t1.mjs [--reset] [--scale 0.5] [--no-vectors]
//   --reset       xoá sạch dữ liệu T1 (mọi lớp, mọi tài khoản trừ ADMIN) rồi sinh lại
//   --scale f     thu nhỏ (0 < f ≤ 1): 1.000·f SV, 20·f lớp; ghi quy mô thực vào báo cáo; không được tuyên bố "đạt T1" cho phần đã thu nhỏ
//   --no-vectors  bỏ 20.000 đoạn tài liệu có vector (nhanh hơn, nhưng RAG / HNSW không đo được)
// Kết quả: benchmarks/reports/seed-t1.json (có khối `meta`). Tài khoản: t1sv0001…@edupilot.local, t1ta1…5, t1gv1…3 (@edupilot.local, dữ liệu mô phỏng D44).
// Thử không cần stack: T1_PSQL="docker exec -i <container-postgres> psql -U postgres -d edupilot_t1 -v ON_ERROR_STOP=1 -qAt".
import { spawnSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { meta } from "../benchmarks/lib/meta.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const argv = process.argv.slice(2);
const RESET = argv.includes("--reset");
const VECTORS = !argv.includes("--no-vectors");
const scaleAt = argv.indexOf("--scale");
const SCALE = scaleAt >= 0 ? Number(argv[scaleAt + 1]) : 1;
const RNG = process.env.SEED_RNG || "20261029";
const die = (msg) => { console.error(msg); process.exit(1); };

if (!(SCALE > 0 && SCALE <= 1)) die("--scale phải trong (0, 1].");
const S = Math.round(1000 * SCALE), C = Math.round(20 * SCALE);
if (C < 1 || S % C !== 0) die(`Quy mô ${SCALE} cho ${S} SV / ${C} lớp không chia đều.`);
const PER = S / C;
if (!process.env.SEED_DEFAULT_PASSWORD) die("Thiếu SEED_DEFAULT_PASSWORD (mật khẩu của admin trong stack đo).");

// ---- chạy psql -----------------------------------------------------------------------------------------------------------
const project = process.env.COMPOSE_PROJECT_NAME ?? "";
let psql;
if (process.env.T1_PSQL) psql = process.env.T1_PSQL.split(/\s+/);
else {
  if (!/^ep-/.test(project)) die("Từ chối: COMPOSE_PROJECT_NAME phải là stack đo ep-<vai> (không phải stack demo).");
  psql = ["docker", "compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "exec", "-T", "postgres", "psql", "-U", "edupilot", "-d", "edupilot_t1", "-v", "ON_ERROR_STOP=1", "-qAt"];
}
function sql(text, what = "truy vấn") {
  const r = spawnSync(psql[0], psql.slice(1), { cwd: root, input: text, encoding: "utf8", maxBuffer: 1 << 26 });
  if (r.status !== 0) die(`Lỗi khi ghi ${what} (dữ liệu dở dang — chạy lại với --reset):\n${(r.stderr || r.error?.message || "").split("\n").slice(0, 5).join("\n")}`);
  return r.stdout.trim();
}

if (sql("select current_database()", "kiểm DB") !== "edupilot_t1") die("Từ chối: DB phải tên edupilot_t1 (dựng stack đo với POSTGRES_DB=edupilot_t1).");
if (Number(sql("select count(*) from users where role = 'ADMIN' and password_hash is not null", "kiểm admin")) < 1) die("Chưa có admin: chạy `gateway admin create` (SEED_DEFAULT_PASSWORD) trước.");
const have = Number(sql("select count(*) from courses where class_code like 'T1-%'", "kiểm dữ liệu cũ"));
if (have > 0 && !RESET) die(`Đã có ${have} lớp T1. Dùng --reset để xoá và sinh lại.`);
if (RESET) {
  console.log("--reset: xoá dữ liệu T1…");
  // Vài bảng chỉ-thêm (audit_log, pii_events, final_grade_snapshots) chặn TRUNCATE bằng trigger mà CASCADE chạm tới; DB T1 dùng một lần nên tắt trigger (replica) chỉ cho lệnh TRUNCATE.
  sql("begin; set local session_replication_role = replica; truncate courses cascade; commit; delete from users where role <> 'ADMIN';", "reset");
}

// ---- các bước ----------------------------------------------------------------------------------------------------------------
// Tiền đề dùng chung: st = sinh viên (n theo email), co = lớp (n theo mã lớp), mỗi SV thuộc lớp ((n-1)/PER)+1. `r` = SEED_RNG vào mọi băm → tất định.
const PRE = `with st as (select id, row_number() over (order by email) n from users where role = 'STUDENT' and email like 't1sv%'),
 co as (select id, row_number() over (order by class_code) n from courses where class_code like 'T1-%'),
 tch as (select course_id, user_id from enrollments where role_in_course = 'TEACHER' and course_id in (select id from co))`;
const H = (expr) => `abs(hashtext(${expr} || '${RNG}'))`;
const TA_AREAS = "array['CHAT','THREADS','PRACTICE','LIBRARY','CALENDAR']";

const steps = [
  ["users", `insert into users (email, password_hash, full_name, role, student_code, email_verified_at, status)
select e, (select password_hash from users where role = 'ADMIN' and password_hash is not null order by created_at limit 1), nm, role::user_role, code, now(), 'ACTIVE'
from (
  select 't1gv' || g || '@edupilot.local' e, 'Giảng viên T1 ' || g nm, 'TEACHER' role, null::text code from generate_series(1, 3) g
  union all select 't1ta' || g || '@edupilot.local', 'Trợ giảng T1 ' || g, 'TA', null from generate_series(1, 5) g
  union all select 't1sv' || lpad(g::text, 4, '0') || '@edupilot.local', 'Sinh viên T1 ' || lpad(g::text, 4, '0'), 'STUDENT', 'T1' || lpad(g::text, 8, '0') from generate_series(1, ${S}) g
) x;`],
  ["courses", `insert into courses (subject_code, class_code, name, semester, join_code, created_by)
select 'T1', 'T1-' || lpad(c::text, 2, '0'), 'Lớp T1 số ' || c, '2026-2027-HK1', 'TTTTT' || substr('ABCDEFGHJKMNPQRSTUVWXYZ23456789', c, 1) || 'M', (select id from users where email = 't1gv1@edupilot.local')
from generate_series(1, ${C}) c;`],
  ["enrollments", `${PRE}
insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot)
select co.id, st.id, 'STUDENT'::enrollment_role, 'ACTIVE'::enrollment_status, 'ROSTER'::enrollment_joined_via, u.student_code from st join users u on u.id = st.id join co on co.n = ((st.n - 1) / ${PER}) + 1
union all select co.id, u.id, 'TEACHER', 'ACTIVE', 'ADMIN', null from co join users u on u.email = 't1gv' || (((co.n - 1) % 3) + 1) || '@edupilot.local'
union all select co.id, u.id, 'TA', 'ACTIVE', 'ADMIN', null from co join users u on u.email = 't1ta' || (((co.n - 1) % 5) + 1) || '@edupilot.local';`],
  ["class_sessions", `${PRE}
insert into class_sessions (course_id, session_no, starts_at, ends_at, room, topic)
select co.id, k, ((current_date - 105 + (k - 1) * 7)::timestamp + time '07:00') at time zone 'Asia/Ho_Chi_Minh', ((current_date - 105 + (k - 1) * 7)::timestamp + time '09:30') at time zone 'Asia/Ho_Chi_Minh', 'P' || (100 + co.n), 'Buổi ' || k
from co, generate_series(1, 15) k;`],
  ["attendance_records", `${PRE}
insert into attendance_records (course_id, session_id, student_id, status, marked_by)
select cs.course_id, cs.id, st.id, (case when h < 6 then 'ABSENT' when h < 10 then 'LATE' else 'PRESENT' end)::attendance_status, tch.user_id
from st join co on co.n = ((st.n - 1) / ${PER}) + 1 join class_sessions cs on cs.course_id = co.id join tch on tch.course_id = co.id,
 lateral (select ${H("st.n || ':' || cs.session_no")} % 100 h) x;`],
  ["participation_events", `${PRE}
insert into participation_events (course_id, student_id, session_id, type, points, client_event_id, created_by)
select cs.course_id, st.id, cs.id, 'SPEAK', 0.25, gen_random_uuid(), tch.user_id
from st join co on co.n = ((st.n - 1) / ${PER}) + 1 join class_sessions cs on cs.course_id = co.id join tch on tch.course_id = co.id
where ${H("'p' || st.n || ':' || cs.session_no")} % 100 < 12;`],
  ["study_time_daily", `${PRE}
insert into study_time_daily (course_id, student_id, date, area, active_seconds)
select co.id, st.id, current_date - 105 + d, a, 60 + ${H("st.n || a || d")} % 3000
from st join co on co.n = ((st.n - 1) / ${PER}) + 1, generate_series(0, 104) d, unnest(${TA_AREAS}) a;`],
  ["chat_sessions", `${PRE}
insert into chat_sessions (course_id, user_id, channel, title, last_message_at, created_at, updated_at)
select co.id, st.id, 'PRIVATE', 'Phiên ' || k, ts, ts, ts
from st join co on co.n = ((st.n - 1) / ${PER}) + 1, generate_series(1, 5) k,
 lateral (select (current_date - 105 + (${H("st.n || 's' || k")} % 104)::int)::timestamp + interval '10 hours' ts) x;`],
  ["chat_messages", `with q as (
  select cs.id sid, cs.course_id, cs.user_id, n, uuidv7() uid, uuidv7() aid, cs.created_at + n * interval '3 minutes' ts
  from chat_sessions cs join courses c on c.id = cs.course_id and c.class_code like 'T1-%', generate_series(1, 9) n),
 u as (insert into chat_messages (id, course_id, session_id, user_id, role, content, created_at, updated_at)
  select uid, course_id, sid, user_id, 'USER', 'Câu hỏi mô phỏng số ' || n || ' về nội dung buổi học.', ts, ts from q returning 1)
insert into chat_messages (id, course_id, session_id, user_id, role, content, reply_to, intent, confidence, completed_at, created_at, updated_at)
select aid, course_id, sid, user_id, 'ASSISTANT', 'Trả lời mô phỏng số ' || n || ' dựa trên tài liệu của lớp.', uid, 'KNOWLEDGE', 0.82, ts + interval '8 seconds', ts + interval '8 seconds', ts + interval '8 seconds' from q;`],
  ["grade_schemes", `${PRE}
insert into grade_schemes (course_id, scheme_no, status, source_title, confirmed_by, confirmed_at, created_by)
select tch.course_id, 1, 'CONFIRMED', 'Quy chế mô phỏng T1', tch.user_id, now(), tch.user_id from tch;
insert into grade_components (course_id, scheme_id, key, name, grp, weight, position)
select s.course_id, s.id, c.key, c.name, c.grp::grade_group, c.w, c.pos from grade_schemes s join courses co on co.id = s.course_id and co.class_code like 'T1-%',
 (values ('midterm', 'Giữa kỳ', 'QT', 20, 0), ('weekly', 'Kiểm tra hằng tuần', 'QT', 20, 1), ('final', 'Thi cuối kỳ', 'CK', 60, 2)) c(key, name, grp, w, pos);
insert into grade_rules (course_id, scheme_id, kind, applicable, params)
select s.course_id, s.id, r.kind::grade_rule_kind, r.app, r.p::jsonb from grade_schemes s join courses co on co.id = s.course_id and co.class_code like 'T1-%',
 (values ('BONUS', false, '{}'), ('PENALTY', false, '{}'), ('ROUNDING', true, '{"step":"0.5","mode":"HALF_UP"}'),
  ('LETTER_SCALE', true, '{"bands":[{"min":"8.5","letter":"A"},{"min":"7","letter":"B"},{"min":"5.5","letter":"C"},{"min":"4","letter":"D"},{"min":"0","letter":"F"}]}')) r(kind, app, p);
insert into grade_items (course_id, scheme_id, component_id, key, name, source, published_at, position)
select k.course_id, k.scheme_id, k.id, k.key, k.name, 'MANUAL', now(), k.position from grade_components k join courses co on co.id = k.course_id and co.class_code like 'T1-%';
insert into gradebooks (course_id) select id from courses where class_code like 'T1-%';`],
  ["grade_entries", `${PRE}
insert into grade_entries (course_id, item_id, student_id, score, entered_by)
select i.course_id, i.id, st.id, 4 + (${H("st.n || i.key")} % 25) * 0.25, tch.user_id
from st join co on co.n = ((st.n - 1) / ${PER}) + 1 join grade_items i on i.course_id = co.id join tch on tch.course_id = co.id;`],
  ["assignments", `${PRE}
insert into assignments (course_id, title, description, max_score, due_at, allow_late, submission_code, reference_answer, rubric, published_at, created_by)
select tch.course_id, 'Bài tập mô phỏng ' || k, 'Phân tích một vụ tấn công theo đề mô phỏng ' || k, 10, now() + (case k when 1 then -70 when 2 then -42 when 3 then -14 else 2 end) * interval '1 day', false, 'BT' || k,
 'Đáp án tham khảo mô phỏng.', '[{"name":"Tác nhân và bề mặt","max_score":"2.5","guidance":"Nêu đủ tác nhân và bề mặt."},{"name":"Phân tích","max_score":"2.5","guidance":"Chuỗi bước có căn cứ."},{"name":"Biện pháp","max_score":"2.5","guidance":"Biện pháp cụ thể."},{"name":"Trình bày","max_score":"2.5","guidance":"Mạch lạc."}]'::jsonb,
 now() - interval '80 days', tch.user_id
from tch, generate_series(1, 4) k;`],
  ["submissions", `${PRE}
insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, extracted_text, extracted_at, extract_status, match_status, match_method, submitted_at, received_at)
select a.course_id, a.id, e.user_id, 'IN_APP', 1, 'seed-t1/' || a.id || '/' || e.user_id, 'bai-' || a.submission_code || '.txt', 1500, 'text/plain',
 md5(a.id::text || e.user_id::text) || md5(e.user_id::text || a.id::text), repeat('Phân tích mô phỏng về tấn công chuỗi cung ứng phần mềm. ', 12), now(), 'OK', 'MATCHED', 'IN_APP', t, t
from assignments a join enrollments e on e.course_id = a.course_id and e.role_in_course = 'STUDENT' join co on co.id = a.course_id,
 lateral (select case when a.due_at < now() then a.due_at - interval '1 day' - ${H("e.user_id::text || a.id::text")} % 48 * interval '1 hour' else now() - ${H("e.user_id::text || a.id::text")} % 48 * interval '1 hour' end t) x;
insert into grading_results (course_id, submission_id, round, run_state, status, attempts, criteria, total, confidence, reviewed_by, reviewed_at, published_by, published_at)
select s.course_id, s.id, 1, 'DONE', 'PUBLISHED', 1, '[]', 4 + (${H("s.id::text")} % 25) * 0.25, 0.8, tch.user_id, now(), tch.user_id, now()
from submissions s join assignments a on a.course_id = s.course_id and a.id = s.assignment_id and a.submission_code <> 'BT4' join enrollments tch on tch.course_id = s.course_id and tch.role_in_course = 'TEACHER';`],
];
if (VECTORS) {
  steps.push(["documents", `${PRE}
insert into documents (course_id, title, type, filename, mime_type, size_bytes, sha256, blob_key, status, page_count, week_no, uploaded_by)
select tch.course_id, 'Tài liệu mô phỏng ' || k, 'LECTURE', 'tai-lieu-' || k || '.pdf', 'application/pdf', 200000, md5(tch.course_id::text || k) || md5(k || tch.course_id::text), 'seed-t1/' || tch.course_id || '/' || k, 'READY', 20, k, tch.user_id
from tch, generate_series(1, 5) k;
insert into document_courses (document_id, course_id, shared_by) select id, course_id, uploaded_by from documents where blob_key like 'seed-t1/%';`]);
  steps.push(["content_chunks", `insert into content_chunks (document_id, course_ids, audience, ord, page_no, heading, text, token_count, embedding)
select d.id, array[d.course_id], 'ALL', o, 1 + o / 10, 'Mục ' || o,
 'Đoạn mô phỏng ' || o || ' của ' || d.title || ': an toàn thông tin, phishing, ransomware, quy chế học vụ và điều kiện dự thi.', 40,
 ('[' || array_to_string(array(select ((hashtext(d.id::text || o || g || '${RNG}') % 2000) / 1000.0)::numeric(5, 3) from generate_series(1, 1536) g), ',') || ']')::vector
from documents d, generate_series(0, 199) o where d.blob_key like 'seed-t1/%';`]);
}

const t0 = Date.now();
const durations = {};
for (const [name, text] of steps) {
  const s = Date.now();
  sql(`begin;\n${text}\ncommit;`, name);
  durations[name] = Math.round((Date.now() - s) / 1000);
  console.log(`${name}: ${durations[name]} s`);
}
sql("analyze", "analyze");

const tables = ["users", "courses", "enrollments", "class_sessions", "attendance_records", "participation_events", "study_time_daily", "chat_sessions", "chat_messages", "grade_components", "grade_entries", "assignments", "submissions", "grading_results", "documents", "content_chunks"];
const rows = Object.fromEntries(sql(tables.map((t) => `select '${t}', count(*) from ${t}`).join(" union all ") + ";").split("\n").map((l) => l.split("|")).map(([t, n]) => [t, Number(n)]));
/** RAM của container postgres ngay sau seed (AC7); chỉ có khi chạy qua compose. */
function pgMem() {
  if (process.env.T1_PSQL) return null;
  const id = spawnSync("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "ps", "-q", "postgres"], { cwd: root, encoding: "utf8" }).stdout.trim();
  return id ? spawnSync("docker", ["stats", "--no-stream", "--format", "{{.MemUsage}}", id], { encoding: "utf8" }).stdout.trim() : null;
}
const out = {
  meta: meta({ scale: SCALE, students: S, courses: C, per_course: PER, vectors: VECTORS, seed_rng: RNG, db: "edupilot_t1", provider: "none", compose_project: project || "(T1_PSQL)" }),
  seconds_total: Math.round((Date.now() - t0) / 1000), seconds_by_step: durations, rows,
  db_mb: Math.round(Number(sql("select pg_database_size('edupilot_t1')")) / 1048576),
  postgres_mem: pgMem(),
  note: SCALE < 1 ? "Đã thu nhỏ: không được tuyên bố đạt T1 cho phần này." : "Quy mô T1 đầy đủ.",
};
const file = path.join(root, "benchmarks", "reports", "seed-t1.json");
mkdirSync(path.dirname(file), { recursive: true });
writeFileSync(file, JSON.stringify(out, null, 2) + "\n");
console.log(`Xong sau ${out.seconds_total} s, ${out.db_mb} MB → ${path.relative(root, file)}`);
