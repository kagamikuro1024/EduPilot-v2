#!/usr/bin/env node
// US-NFR-03 AC1, AC3, AC7: kiểm DB edupilot_t1 sau seed-t1 — số dòng đúng bảng SYSTEM_DESIGN §1.2 (±10 %), không dòng mồ côi (mọi dòng con có course_id trùng cha),
// mật khẩu mô phỏng băm (không cố định trong mã). Chỉ stack -p ep-*. `COMPOSE_PROJECT_NAME=ep-t1 node scripts/check-t1-seed.mjs` (thoát mã ≠ 0 nếu sai).
import { spawnSync } from "node:child_process";
const project = process.env.COMPOSE_PROJECT_NAME ?? "";
if (!/^ep-/.test(project)) { console.error("Từ chối: COMPOSE_PROJECT_NAME phải là stack đo ep-<vai>."); process.exit(1); }
const q = (sql) => { const r = spawnSync("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "exec", "-T", "postgres", "psql", "-U", "edupilot", "-d", "edupilot_t1", "-qAt", "-v", "ON_ERROR_STOP=1"], { input: sql, encoding: "utf8" }); if (r.status !== 0) { console.error(r.stderr); process.exit(1); } return r.stdout.trim(); };
let bad = 0;
const check = (name, ok, detail) => { console.log(`${ok ? "OK  " : "LỖI"} ${name}: ${detail}`); if (!ok) bad++; };
const near = (n, target) => Math.abs(n - target) <= target * 0.1;
const n = (sql) => Number(q(sql));

for (const [name, sql, target] of [["sinh viên", "select count(*) from users where role = 'STUDENT'", 1000], ["lớp", "select count(*) from courses", 20], ["TA", "select count(*) from users where role = 'TA'", 5], ["giảng viên", "select count(*) from users where role = 'TEACHER'", 3],
  ["chat_messages", "select count(*) from chat_messages", 90000], ["study_time_daily", "select count(*) from study_time_daily", 525000], ["bài nộp", "select count(*) from submissions", 4000]]) {
  const v = n(sql); check(name, near(v, target), `${v} (đích ${target} ±10 %)`);
}
// mọi dòng con có course_id đúng cha: đếm dòng mà cha cùng id nhưng khác lớp (FK phức hợp không thể vi phạm, kiểm lại bằng join)
for (const [child, parent, key] of [["submissions", "assignments", "assignment_id"], ["grade_entries", "grade_items", "item_id"], ["attendance_records", "class_sessions", "session_id"], ["chat_messages", "chat_sessions", "session_id"], ["content_chunks", "documents", "document_id"]]) {
  if (child === "content_chunks") { check(`${child} → ${parent}`, n(`select count(*) from content_chunks c join documents d on d.id = c.document_id where not (c.course_ids @> array[d.course_id])`) === 0, "đoạn nằm ngoài lớp của tài liệu: 0"); continue; }
  const v = n(`select count(*) from ${child} c join ${parent} p on p.id = c.${key} where c.course_id <> p.course_id`); check(`${child} → ${parent}`, v === 0, `${v} dòng lệch lớp`);
}
check("lớp mồ côi", n("select count(*) from courses c where not exists (select 1 from enrollments e where e.course_id = c.id and e.role_in_course = 'TEACHER')") === 0, "mọi lớp có giảng viên");
check("mật khẩu", n("select count(distinct password_hash) from users where email like 't1%@edupilot.local'") === 1 && n("select count(*) from users where email like 't1%@edupilot.local' and password_hash like '$2%'") > 1000, "một bản băm bcrypt dùng chung, không có mật khẩu rõ");
console.log(`DB: ${q("select pg_size_pretty(pg_database_size('edupilot_t1'))")}; chỉ mục: ${q("select pg_size_pretty(sum(pg_relation_size(indexrelid))) from pg_stat_user_indexes")}`);
process.exit(bad ? 1 : 0);
