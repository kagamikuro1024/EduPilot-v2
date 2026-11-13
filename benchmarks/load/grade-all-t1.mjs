#!/usr/bin/env node
// US-NFR-04 AC9: chấm 1.000 bài THỜI GIAN THẬT (không ngoại suy): "Chấm tất cả" BT4 của cả 20 lớp T1 (20 × 50 bài), provider giả của WORKER trễ K6_GRADE_LAT (20 s mỗi lượt), hai lượt mỗi bài, 5 luồng BATCH.
// Chạy song song với `run-t1.mjs grading-baseline --duration 150m --tag 1000` (chat giữ nguyên độ trễ 300 ms ở gateway) để so TTFT chat lúc chấm với lúc nền.
//   COMPOSE_PROJECT_NAME=ep-t1 EP_PORT_HTTPS=843 SEED_DEFAULT_PASSWORD=… node benchmarks/load/grade-all-t1.mjs [--max-hours 4]
// Ghi benchmarks/reports/load-c-1000.json (khối `meta` đủ trường, tiến độ mỗi phút); thoát 0 khi mọi bài DONE.
import { spawnSync } from "node:child_process";
import { writeFileSync } from "node:fs";
import { meta } from "../lib/meta.mjs";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
const project = process.env.COMPOSE_PROJECT_NAME ?? "", port = process.env.EP_PORT_HTTPS ?? "", PW = process.env.SEED_DEFAULT_PASSWORD;
const die = (m) => { console.error(m); process.exit(1); };
if (!/^ep-/.test(project) || !port || port === "443" || !PW) die("Cần COMPOSE_PROJECT_NAME=ep-*, EP_PORT_HTTPS (≠ 443), SEED_DEFAULT_PASSWORD.");
const maxHours = Number(process.argv[process.argv.indexOf("--max-hours") + 1] || 4);
const BASE = `https://localhost:${port}/api/v1`;
const psql = (sql) => { const r = spawnSync("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "exec", "-T", "postgres", "psql", "-U", "edupilot", "-d", "edupilot_t1", "-qAt", "-v", "ON_ERROR_STOP=1"], { input: sql, encoding: "utf8" }); if (r.status !== 0) die(r.stderr); return r.stdout.trim(); };
const api = (path, tok, init = {}) => fetch(`${BASE}${path}`, { ...init, headers: { "Content-Type": "application/json", ...(tok ? { Authorization: `Bearer ${tok}` } : {}), ...(init.headers ?? {}) } });
const login = async (u) => (await (await api("/auth/login", null, { method: "POST", body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }) })).json()).access_token;

const targets = psql(`select c.id, a.id, u.email from courses c join assignments a on a.course_id = c.id and a.submission_code = 'BT4' join enrollments e on e.course_id = c.id and e.role_in_course = 'TEACHER' join users u on u.id = e.user_id
  where not exists (select 1 from grading_results r join submissions s on s.id = r.submission_id where s.assignment_id = a.id) order by c.class_code`).split("\n").filter(Boolean).map((l) => l.split("|"));
if (!targets.length) die("Không còn BT4 nào chưa chấm (xoá grading_results của BT4 để chạy lại).");
const started = Date.now();
let queued = 0;
for (const [course, assignment, email] of targets) {
  const r = await api(`/courses/${course}/assignments/${assignment}/grade-all`, await login(email.split("@")[0]), { method: "POST", body: "{}", headers: { "Idempotency-Key": `g1000-${course.slice(0, 8)}-${started}` } });
  if (r.status !== 202) die(`grade-all ${course} → ${r.status}`);
  queued += (await r.json()).queued;
}
const ids = targets.map((t) => `'${t[1]}'`).join(",");
const where = `submission_id in (select id from submissions where assignment_id in (${ids}))`;
const progress = [];
for (;;) {
  const st = Object.fromEntries(psql(`select run_state, count(*) from grading_results where ${where} group by 1`).split("\n").filter(Boolean).map((l) => l.split("|")).map(([k, v]) => [k, Number(v)]));
  progress.push({ minute: Math.round((Date.now() - started) / 60000), ...st });
  if (!(st.QUEUED || st.RUNNING)) break;
  if ((Date.now() - started) / 3.6e6 > maxHours) break;
  await new Promise((r) => setTimeout(r, 60000));
}
const last = progress.at(-1);
const twice = Number(psql(`select count(*) from (select submission_id, round from grading_results where ${where} group by 1, 2 having count(*) > 1) x`));
const hours = Number(((Date.now() - started) / 3.6e6).toFixed(2));
const demo = spawnSync("docker", ["ps", "-q", "--filter", "label=com.docker.compose.project=edupilot"], { encoding: "utf8" }).stdout.trim() ? "running" : "stopped";
const out = { meta: { ...meta({ scenario: "grade-1000", courses: targets.length, provider: "fake", worker_fake_latency: process.env.K6_GRADE_LAT ?? "(xem docker compose)", grading_workers: process.env.GRADING_WORKERS ?? "2 (mặc định)", compose_project: project }), k6_cpus: "1.5", demo_stack: demo }, queued, hours, done: last.DONE ?? 0, failed: last.FAILED ?? 0, orphan: (last.QUEUED ?? 0) + (last.RUNNING ?? 0), graded_twice: twice, slo_hours: 3, pass: (last.DONE ?? 0) === queued && hours <= 3 && twice === 0, progress };
writeFileSync("benchmarks/reports/load-c-1000.json", JSON.stringify(out, null, 2));
console.log(JSON.stringify({ ...out, progress: undefined }));
process.exit(out.pass ? 0 : 1);
