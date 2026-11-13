#!/usr/bin/env node
// US-NFR-05: EXPLAIN (ANALYZE, BUFFERS) của 10 truy vấn nặng nhất (xếp theo total_exec_time của pg_stat_statements sau k6 (a)–(d), xem docs/research/…-explain.md)
// trên DB edupilot_t1, tham số đại diện (lớp T1-01, SV đầu, giảng viên t1gv1). Mỗi truy vấn chạy 1 lần làm nóng + 5 lần đo, lấy trung vị.
//   COMPOSE_PROJECT_NAME=ep-t1 node benchmarks/explain.mjs before|after   → benchmarks/reports/explain.json (khoá `before` / `after`)
// Chỉ stack -p ep-*, DB edupilot_t1; chỉ đọc (mỗi truy vấn trong giao dịch read-only).
import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { meta } from "./lib/meta.mjs";

const label = process.argv[2];
if (!["before", "after"].includes(label)) { console.error("Dùng: explain.mjs before|after"); process.exit(1); }
const project = process.env.COMPOSE_PROJECT_NAME ?? "";
if (!/^ep-/.test(project)) { console.error("Từ chối: COMPOSE_PROJECT_NAME phải là stack đo ep-<vai>."); process.exit(1); }
const psql = (sql) => {
  const r = spawnSync("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "exec", "-T", "postgres", "psql", "-U", "edupilot", "-d", "edupilot_t1", "-v", "ON_ERROR_STOP=1", "-qAt"], { input: sql, encoding: "utf8", maxBuffer: 1 << 26 });
  if (r.status !== 0) { console.error(r.stderr); process.exit(1); }
  return r.stdout.trim();
};
const one = (sql) => psql(sql).split("\n")[0];
const C = one("select id from courses where class_code = 'T1-01'");
const S = one(`select user_id from enrollments where course_id = '${C}' and role_in_course = 'STUDENT' order by user_id limit 1`);
const T = one(`select user_id from enrollments where course_id = '${C}' and role_in_course = 'TEACHER' limit 1`);
const IDS = one(`select array_agg(user_id::text) from (select user_id from enrollments where course_id = '${C}' and role_in_course = 'STUDENT' order by user_id limit 50) x`).replace(/[{}]/g, "").split(",");
const ids = `array[${IDS.map((i) => `'${i}'`).join(",")}]::uuid[]`;
const V = `(select embedding from content_chunks where course_ids @> array['${C}'::uuid] and embedding is not null limit 1)`;
const sch = one(`select id from grade_schemes where course_id = '${C}' limit 1`);

const risk = (extra) => `from v_student_360 v where v.course_id = '${C}' and (not false or v.risk_level = 'HIGH') and (not false or v.absent >= 3) and (not false or (v.last_week_asks + v.last_week_posts = 0 and v.study_seconds_7d < 3600)) and ('' = '' or true)${extra ?? ""}`;
export const QUERIES = {
  rag_hybrid: `WITH q AS (SELECT ${V} AS v, vn_bigram_query('tấn công chuỗi cung ứng phần mềm') AS t),
vec AS (SELECT c.id, ROW_NUMBER() OVER (ORDER BY c.embedding <=> q.v) AS r FROM content_chunks c JOIN documents d ON d.id = c.document_id, q
  WHERE c.course_ids @> ARRAY['${C}'::uuid] AND d.status = 'READY' AND d.use_for_rag AND d.type <> 'ANSWER_KEY' AND c.audience::text = ANY(ARRAY['ALL']) AND c.embedding IS NOT NULL ORDER BY c.embedding <=> q.v LIMIT 30),
kw AS (SELECT c.id, ROW_NUMBER() OVER (ORDER BY ts_rank_cd(c.tsv, q.t) DESC) AS r FROM content_chunks c JOIN documents d ON d.id = c.document_id, q
  WHERE c.course_ids @> ARRAY['${C}'::uuid] AND d.status = 'READY' AND d.use_for_rag AND d.type <> 'ANSWER_KEY' AND c.audience::text = ANY(ARRAY['ALL']) AND c.embedding IS NOT NULL AND c.tsv @@ q.t ORDER BY ts_rank_cd(c.tsv, q.t) DESC LIMIT 30),
fused AS (SELECT x.id, SUM(1.0 / (60 + x.r)) AS s FROM (SELECT vec.id, vec.r FROM vec UNION ALL SELECT kw.id, kw.r FROM kw) x GROUP BY x.id ORDER BY s DESC LIMIT 8)
SELECT c.id, c.document_id, d.title, c.page_no, c.text, (1 - (c.embedding <=> (SELECT v FROM q)))::float8 AS cosine, f.s::float8 AS score FROM fused f JOIN content_chunks c ON c.id = f.id JOIN documents d ON d.id = c.document_id WHERE c.course_ids @> ARRAY['${C}'::uuid] ORDER BY f.s DESC`,
  students_list_360: `with base as (select v.student_id, v.full_name, v.student_code, v.recorded, v.present, v.late, v.excused, v.absent, v.speak_count, v.points_total, v.study_seconds_7d, v.last_week_asks, v.last_week_posts, v.risk_level, v.risk_computed_at,
  (case v.risk_level when 'AT_RISK' then 2 when 'WATCH' then 1 else 0 end)::int as risk_rank, vn_fold(v.full_name)::text as name_fold, (u.tracking_notice_ack_at is not null and (not c.allow_study_time_opt_out or coalesce(us.track_study_time, true)))::bool as tracked
  from v_student_360 v join users u on u.id = v.student_id join courses c on c.id = v.course_id left join user_settings us on us.user_id = v.student_id where v.course_id = '${C}')
select * from base order by risk_rank desc, name_fold, student_id limit 51`,
  students_count_360: `select count(*)::int from v_student_360 v where v.course_id = '${C}'`,
  participation_events: `select pe.student_id, u.full_name, (pe.type = 'PENALTY')::bool as is_penalty from participation_events pe join enrollments en on en.course_id = pe.course_id and en.user_id = pe.student_id and en.role_in_course = 'STUDENT' and en.status = 'ACTIVE'
  join users u on u.id = pe.student_id where pe.course_id = '${C}' and pe.points <> 0 and pe.type in ('SPEAK','BONUS','PENALTY') group by pe.student_id, u.full_name, (pe.type = 'PENALTY') order by vn_fold(u.full_name), pe.student_id`,
  study_time_weekly: `select (date_trunc('week', date::timestamp))::date as week_start, (sum(active_seconds) / 60)::int as minutes from study_time_daily where course_id = '${C}' and student_id = '${S}' and date >= current_date - 105 and date < current_date group by 1 order by 1`,
  roster_page: `select e.user_id as student_id, u.full_name, e.student_code_snapshot as student_code, vn_fold(u.full_name)::text as fold_name from enrollments e join users u on u.id = e.user_id where e.course_id = '${C}' and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' order by vn_fold(u.full_name), e.user_id limit 50`,
  attendance_me: `select s.id as session_id, s.session_no, s.starts_at, a.status, a.note from attendance_records a join class_sessions s on s.course_id = a.course_id and s.id = a.session_id where a.course_id = '${C}' and a.student_id = '${S}' order by s.starts_at desc, s.id desc limit 50`,
  today_assignments: `select 'DUE'::text as kind, a.id, a.course_id, a.title, a.due_at as at, a.allow_late, a.late_max_days from assignments a where a.course_id = any(array['${C}']::uuid[]) and a.published_at is not null and 'IN_APP' = any(a.channels) and a.due_at <= now() + interval '14 days'
  and (a.due_at > now() or (a.allow_late and a.due_at + make_interval(days => a.late_max_days::int) > now())) and not exists (select 1 from submissions s where s.course_id = a.course_id and s.assignment_id = a.id and s.student_id = '${S}' and s.match_status = 'MATCHED')
union all select 'GRADED', a.id, a.course_id, a.title, r.published_at, a.allow_late, a.late_max_days from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
  where s.course_id = any(array['${C}']::uuid[]) and s.student_id = '${S}' and s.match_status = 'MATCHED' and r.status = 'PUBLISHED' and r.published_at > now() - interval '30 days' and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id) order by kind, at, id`,
  library_list: `select d.id, d.title, d.type, d.filename, d.week_no, d.updated_at, (d.use_for_rag and exists (select 1 from content_chunks c where c.document_id = d.id and c.course_ids @> array['${C}'::uuid] and c.embedding is not null))::bool as can_ask_ai from documents d
  where (d.course_id = '${C}' or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = '${C}')) and d.status = 'READY' and d.visible_to_students and d.type <> 'ANSWER_KEY' order by d.updated_at desc, d.id desc limit 30`,
  gradebook_signals: `select e.user_id as student_id, coalesce(p.bonus, 0)::numeric(8,2) as bonus_points, coalesce(p.penalty, 0)::numeric(8,2) as penalty_event_points, coalesce(a.absent, 0)::int as absent from enrollments e
  left join (select pe.student_id, sum(pe.points) filter (where pe.type in ('SPEAK','BONUS')) as bonus, abs(sum(pe.points) filter (where pe.type = 'PENALTY')) as penalty from participation_events pe where pe.course_id = '${C}' and pe.student_id = any(${ids}) group by pe.student_id) p on p.student_id = e.user_id
  left join (select ar.student_id, count(*) as absent from attendance_records ar where ar.course_id = '${C}' and ar.status = 'ABSENT' and ar.student_id = any(${ids}) group by ar.student_id) a on a.student_id = e.user_id
  where e.course_id = '${C}' and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' and e.user_id = any(${ids})`,
};

const out = {};
for (const [name, q] of Object.entries(QUERIES)) {
  const runs = [];
  let plan;
  for (let i = 0; i < 6; i++) {
    const js = JSON.parse(psql(`begin read only; explain (analyze, buffers, format json) ${q}; rollback;`).replace(/^BEGIN\n?/m, "").replace(/\nROLLBACK$/, ""))[0];
    if (i > 0) runs.push(js["Execution Time"]);
    plan = js.Plan; plan.__buf = js.Plan["Shared Hit Blocks"] + js.Plan["Shared Read Blocks"];
  }
  runs.sort((a, b) => a - b);
  const nodes = []; const walk = (n) => { nodes.push(`${n["Node Type"]}${n["Relation Name"] ? `(${n["Relation Name"]}${n["Index Name"] ? `:${n["Index Name"]}` : ""})` : ""}`); (n.Plans ?? []).forEach(walk); };
  walk(plan);
  out[name] = { ms_median: Number(runs[2].toFixed(3)), ms_runs: runs.map((x) => Number(x.toFixed(3))), rows: plan["Actual Rows"], buffers: plan.__buf, seq_scans: nodes.filter((n) => n.startsWith("Seq Scan")), nodes: [...new Set(nodes)] };
  console.log(`${name}: ${out[name].ms_median} ms · ${out[name].buffers} buffers · seq: ${out[name].seq_scans.join(", ") || "-"}`);
}
const idx = psql(`select indexrelname || '|' || relname || '|' || pg_relation_size(indexrelid) from pg_stat_user_indexes`);
const file = "benchmarks/reports/explain.json";
const doc = existsSync(file) ? JSON.parse(readFileSync(file, "utf8")) : { meta: meta({ db: "edupilot_t1", compose_project: project }), before: null, after: null };
doc[label] = { at: new Date().toISOString(), queries: out, index_bytes: Object.fromEntries(idx.split("\n").map((l) => { const [i, r, b] = l.split("|"); return [`${r}.${i}`, Number(b)]; })) };
mkdirSync("benchmarks/reports", { recursive: true });
writeFileSync(file, JSON.stringify(doc, null, 2));
