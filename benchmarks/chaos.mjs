#!/usr/bin/env node
// US-NFR-06: hỗn loạn nhỏ trên stack đo ep-*: (1) kill -9 một worker giữa lượt chấm, (2) dừng Redis 70 s trong lúc chấm + chat, (3) dừng docling-serve 2 phút (cần stack dựng với profile ingest).
// Đối chiếu cuối mỗi tình huống: việc vào = việc xong + dead-letter, 0 việc thất lạc, 0 bài chấm hai lần.
//   COMPOSE_PROJECT_NAME=ep-t1 EP_PORT_HTTPS=843 SEED_DEFAULT_PASSWORD=… node benchmarks/chaos.mjs [worker|redis|docling] …   → benchmarks/reports/chaos.json (ghi thêm từng tình huống)
// Định nghĩa (P-95-7 Q18):
//   việc vào        = số bài nộp đã `grade-all` (grading_results vòng 1 được tạo, run_state QUEUED lúc đầu) / số tài liệu được ingest
//   xong            = grading_results.run_state = DONE (bài) ; documents.status = READY (tài liệu)
//   dead-letter     = grading_results.run_state = FAILED có error_code (+ XLEN của stream ep:grading.dead) ; jobs.status = FAILED
//   mồ côi          = run_state IN (QUEUED, RUNNING) còn sau thời hạn chờ (10 phút) — không consumer nào nhận; hoặc XPENDING của nhóm `grading` > 0 sau khi xong
//   chấm hai lần    = có submission_id nào có > 1 dòng grading_results cùng round
// Lệnh dừng / giết nhắm theo project (`docker compose -p …` / nhãn com.docker.compose.project), không theo tên chung. Stack demo được kiểm trước / sau.
import { spawnSync } from "node:child_process";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { meta } from "./lib/meta.mjs";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
const project = process.env.COMPOSE_PROJECT_NAME ?? "";
const port = process.env.EP_PORT_HTTPS ?? "";
const PW = process.env.SEED_DEFAULT_PASSWORD;
const die = (m) => { console.error(m); process.exit(1); };
if (!/^ep-/.test(project)) die("Từ chối: COMPOSE_PROJECT_NAME phải là stack đo ep-<vai>.");
if (!port || port === "443" || !PW) die("Cần EP_PORT_HTTPS (≠ 443) và SEED_DEFAULT_PASSWORD.");
const BASE = `https://localhost:${port}/api/v1`;
const run = (cmd, args, input) => { const r = spawnSync(cmd, args, { encoding: "utf8", input }); if (r.status !== 0) throw new Error(`${cmd} ${args.join(" ")}: ${r.stderr}`); return r.stdout.trim(); };
const compose = (...a) => run("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-f", "benchmarks/load/pg-stats.yml", "-f", "benchmarks/load/k6-stack.yml", "-p", project, ...a]);
const psql = (sql) => run("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "exec", "-T", "postgres", "psql", "-U", "edupilot", "-d", "edupilot_t1", "-qAt", "-v", "ON_ERROR_STOP=1"], sql);
const redis = (...a) => run("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "exec", "-T", "redis", "redis-cli", ...a]);
const sleep = (s) => new Promise((r) => setTimeout(r, s * 1000));
const demoState = () => run("docker", ["ps", "-aq", "--filter", "label=com.docker.compose.project=edupilot"]).split("\n").filter(Boolean).map((id) => run("docker", ["inspect", "-f", "{{.State.Status}}", id])).join(",");

async function api(path, tok, init = {}) { const r = await fetch(`${BASE}${path}`, { ...init, headers: { "Content-Type": "application/json", ...(tok ? { Authorization: `Bearer ${tok}` } : {}), ...(init.headers ?? {}) } }); return r; }
const login = async (u) => (await (await api("/auth/login", null, { method: "POST", body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }) })).json()).access_token;

/** BT4 của các lớp [from, to) (theo class_code T1-NN), chưa có kết quả chấm; trả [{course, assignment, teacher}] */
function pick(from, to) {
  const rows = psql(`select c.id, a.id, u.email from courses c join assignments a on a.course_id = c.id and a.submission_code = 'BT4' join enrollments e on e.course_id = c.id and e.role_in_course = 'TEACHER' join users u on u.id = e.user_id
    where c.class_code between 'T1-${String(from).padStart(2, "0")}' and 'T1-${String(to - 1).padStart(2, "0")}' and not exists (select 1 from grading_results r join submissions s on s.id = r.submission_id where s.assignment_id = a.id) order by c.class_code`);
  return rows.split("\n").filter(Boolean).map((l) => { const [course, assignment, email] = l.split("|"); return { course, assignment, teacher: email.split("@")[0] }; });
}
async function gradeAll(targets) {
  let queued = 0;
  for (const t of targets) {
    const tok = await login(t.teacher);
    const r = await api(`/courses/${t.course}/assignments/${t.assignment}/grade-all`, tok, { method: "POST", body: "{}", headers: { "Idempotency-Key": `chaos-${Date.now()}-${t.course.slice(0, 8)}` } });
    if (r.status !== 202) throw new Error(`grade-all ${t.course} → ${r.status}`);
    queued += (await r.json()).queued;
  }
  return queued;
}
const ids = (targets) => targets.map((t) => `'${t.assignment}'`).join(",");
function reconcile(targets, waitedSeconds, dead0 = 0) {
  const where = `submission_id in (select id from submissions where assignment_id in (${ids(targets)}))`;
  const states = Object.fromEntries(psql(`select run_state, count(*) from grading_results where ${where} group by 1`).split("\n").filter(Boolean).map((l) => l.split("|")).map(([k, v]) => [k, Number(v)]));
  const twice = Number(psql(`select count(*) from (select submission_id, round from grading_results where ${where} group by 1, 2 having count(*) > 1) x`));
  const missing = Number(psql(`select count(*) from submissions s where s.assignment_id in (${ids(targets)}) and s.match_status = 'MATCHED' and not exists (select 1 from grading_results r where r.submission_id = s.id)`));
  const pending = Number(redis("xpending", "ep:grading", "grading").split("\n")[0] || 0);
  const dead = Number(redis("xlen", "ep:grading.dead") || 0);
  const vao = Object.values(states).reduce((a, b) => a + b, 0) + missing;
  const orphan = (states.QUEUED ?? 0) + (states.RUNNING ?? 0);
  return { in: vao, done: states.DONE ?? 0, failed_dead_letter: states.FAILED ?? 0, orphan, graded_twice: twice, never_enqueued: missing, xpending: pending, dead_stream_new: dead - dead0, waited_seconds: waitedSeconds, pass: vao === (states.DONE ?? 0) + (states.FAILED ?? 0) && orphan === 0 && twice === 0 && missing === 0 && pending === 0 };
}
async function waitDone(targets, maxSeconds) {
  const t0 = Date.now();
  while ((Date.now() - t0) / 1000 < maxSeconds) {
    const left = Number(psql(`select count(*) from grading_results where run_state in ('QUEUED','RUNNING') and submission_id in (select id from submissions where assignment_id in (${ids(targets)}))`));
    if (left === 0) break;
    await sleep(5);
  }
  return Math.round((Date.now() - t0) / 1000);
}
const containerOf = (svc) => run("docker", ["ps", "-q", "--filter", `label=com.docker.compose.project=${project}`, "--filter", `label=com.docker.compose.service=${svc}`]).split("\n").filter(Boolean);

const which = process.argv.slice(2).length ? process.argv.slice(2) : ["worker", "redis"];
const file = "benchmarks/reports/chaos.json";
const doc = existsSync(file) ? JSON.parse(readFileSync(file, "utf8")) : { meta: meta({ compose_project: project, note: "tải: chấm 150 bài (3 lớp × BT4) + chat; provider giả trễ 2 s mỗi lượt chấm (K6_GRADE_LAT=2000-2000)" }), scenarios: {} };
const demoBefore = demoState();
const dead0 = () => Number(redis("xlen", "ep:grading.dead") || 0); // `ep:grading.dead` tồn tại từ trước các lượt chaos: chỉ đếm phần tăng thêm

if (which.includes("worker")) {
  const targets = pick(2, 5);
  const d0 = dead0();
  const queued = await gradeAll(targets);
  await sleep(20);
  const victim = containerOf("worker")[0];
  run("docker", ["kill", "-s", "KILL", victim]); // một container của project, theo id
  const killedAt = new Date().toISOString();
  const waited = await waitDone(targets, 600);
  doc.scenarios.worker_kill9 = { at: killedAt, queued, workers_before: containerOf("worker").length + 1, ...reconcile(targets, waited, d0) };
  console.log("worker_kill9", JSON.stringify(doc.scenarios.worker_kill9));
}
if (which.includes("redis")) {
  const targets = pick(5, 8);
  const d0 = dead0();
  const queued = await gradeAll(targets);
  await sleep(15);
  const t0 = new Date().toISOString();
  compose("stop", "redis");
  const sv = await login("t1sv0001").catch(() => null); // login cần Redis (giới hạn tần suất): ghi nhận hành vi, không bắt buộc thành công
  const during = [];
  const probe = async () => { const r = await api("/me/today", sv ?? "x"); during.push(r.status); };
  for (let i = 0; i < 7; i++) { await probe().catch(() => during.push("lỗi mạng")); await sleep(10); }
  compose("start", "redis");
  await sleep(10);
  const waited = await waitDone(targets, 600);
  doc.scenarios.redis_down_70s = { at: t0, queued, statuses_during_outage: [...new Set(during)], status_5xx_raw_during: during.filter((s) => typeof s === "number" && s >= 500 && s !== 503).length, ...reconcile(targets, waited, d0) };
  console.log("redis_down_70s", JSON.stringify(doc.scenarios.redis_down_70s));
}
doc.demo_stack = { before: demoBefore, after: demoState(), note: "trạng thái các container của project edupilot trước / sau chaos (không bị đụng)" };
writeFileSync(file, JSON.stringify(doc, null, 2));
process.exit(Object.values(doc.scenarios).every((s) => s.pass) ? 0 : 1);
