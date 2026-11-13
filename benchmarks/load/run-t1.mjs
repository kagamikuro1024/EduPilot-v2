#!/usr/bin/env node
// Chạy một kịch bản T1 bằng k6 TRONG VM Colima (TL-1 §5): container `grafana/k6` dùng chung mạng của container `caddy` của stack đo (BASE = https://localhost, không qua
// bộ chuyển tiếp cổng của macOS nên không cộng độ trễ của nó), giới hạn --cpus 1.5 --memory 1.5g để phần còn lại cho stack xấp xỉ S1 (4 vCPU).
//   COMPOSE_PROJECT_NAME=ep-t1 SEED_DEFAULT_PASSWORD=… node benchmarks/load/run-t1.mjs browse|dashboard [--duration 5m] [--vus 300] [--students 1000] [--tag x2]
//   … run-t1.mjs chat-first_event|chat-ttft            (chat.js, 50 tài khoản T1; gateway dựng kèm benchmarks/load/k6-stack.yml, K6_LAT=5000-15000 | 300-300)
//   … run-t1.mjs grading-baseline | grading-grading --course <id> --assignment <id> [--baseline <p95 TTFT ms>] [--vus 20]   (grading-vs-chat.js)
// Ghi benchmarks/reports/load-{a,b,c,d}[-phụ][-tag].json (a=browse, b=chat, c=grading, d=dashboard) (khối `meta` đủ trường, thêm k6_cpus và demo_stack). Chỉ chạy trên stack -p ep-*.
import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { meta } from "../lib/meta.mjs";

const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const [scenario, ...rest] = process.argv.slice(2);
const opt = (name, d) => (rest.includes(name) ? rest[rest.indexOf(name) + 1] : d);
const die = (m) => { console.error(m); process.exit(1); };
const project = process.env.COMPOSE_PROJECT_NAME ?? "";
const SCEN = {
  browse: { file: "load-a", script: "t1.js", env: (o) => ({ SCENARIO: "browse", DURATION: o("--duration", "5m"), VUS: o("--vus", 300), STUDENTS: o("--students", 1000) }) },
  dashboard: { file: "load-d", script: "t1.js", env: (o) => ({ SCENARIO: "dashboard", DURATION: o("--duration", "2m"), STUDENTS: o("--students", 1000) }) },
  "chat-first_event": { file: "load-b-first_event", script: "chat.js", env: () => ({ SCENARIO: "first_event", ACCOUNTS: "t1" }) },
  "chat-ttft": { file: "load-b-ttft", script: "chat.js", env: () => ({ SCENARIO: "ttft", ACCOUNTS: "t1" }) },
  "grading-baseline": { file: "load-c-baseline", script: "grading-vs-chat.js", env: (o) => ({ PHASE: "baseline", VUS: o("--vus", 20), ACCOUNTS: "t1", DURATION: o("--duration", "40s"), COURSE_ID: o("--course", ""), ASSIGNMENT_ID: o("--assignment", ""), CHAT_CLASS: o("--course", "") }) },
  "grading-grading": { file: "load-c-grading", script: "grading-vs-chat.js", env: (o) => ({ PHASE: "grading", VUS: o("--vus", 20), ACCOUNTS: "t1", DURATION: o("--duration", "40s"), COURSE_ID: o("--course", ""), ASSIGNMENT_ID: o("--assignment", ""), BASELINE_TTFT_MS: o("--baseline", ""), CHAT_CLASS: o("--course", "") }) },
}[scenario];
if (!SCEN) die("Dùng: run-t1.mjs browse|dashboard|chat-first_event|chat-ttft|grading-baseline|grading-grading [--tag x2] …");
if (!/^ep-/.test(project)) die("Từ chối: COMPOSE_PROJECT_NAME phải là stack đo ep-<vai> (không phải stack demo).");
if (!process.env.SEED_DEFAULT_PASSWORD) die("Thiếu SEED_DEFAULT_PASSWORD.");

const run = (cmd, args) => spawnSync(cmd, args, { cwd: root, encoding: "utf8" });
const caddy = run("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "ps", "-q", "caddy"]).stdout.trim();
if (!caddy) die(`Không thấy container caddy của project ${project} (stack đã chạy chưa?).`);
const demo = run("docker", ["ps", "-q", "--filter", "label=com.docker.compose.project=edupilot"]).stdout.trim() ? "running" : "stopped";
const cpus = "1.5";
const count = (svc) => Number(run("docker", ["ps", "-q", "--filter", `label=com.docker.compose.project=${project}`, "--filter", `label=com.docker.compose.service=${svc}`]).stdout.split("\n").filter(Boolean).length);
const tag = opt("--tag", "");
const out = `${SCEN.file}${tag ? `-${tag}` : ""}`;
const senv = SCEN.env(opt);
const m = { ...meta({ scenario, ...senv, gateways: count("gateway"), workers: count("worker"), provider: "fake", k6_cpus: cpus, demo_stack: demo, compose_project: project, tag }) };
mkdirSync(path.join(root, "benchmarks", "reports"), { recursive: true });
const r = spawnSync("docker", [
  "run", "--rm", "--network", `container:${caddy}`, "--cpus", cpus, "--memory", "1.5g",
  "-v", `${path.join(root, "benchmarks")}:/b`, "-w", "/b/reports", "grafana/k6", "run", `/b/load/${SCEN.script}`,
  "-e", "BASE=https://localhost", "-e", `SEED_DEFAULT_PASSWORD=${process.env.SEED_DEFAULT_PASSWORD}`, "-e", `OUT=/b/reports/${out}.json`, "-e", `META=${JSON.stringify(m)}`,
  ...Object.entries(senv).flatMap(([k, v]) => ["-e", `${k}=${v}`]),
], { cwd: root, stdio: ["ignore", "inherit", "inherit"] });
process.exit(r.status ?? 1);
