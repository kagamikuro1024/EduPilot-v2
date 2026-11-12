#!/usr/bin/env node
// Chạy một kịch bản T1 bằng k6 TRONG VM Colima (TL-1 §5): container `grafana/k6` dùng chung mạng của container `caddy` của stack đo (BASE = https://localhost, không qua
// bộ chuyển tiếp cổng của macOS nên không cộng độ trễ của nó), giới hạn --cpus 1.5 --memory 1.5g để phần còn lại cho stack xấp xỉ S1 (4 vCPU).
//   COMPOSE_PROJECT_NAME=ep-t1 SEED_DEFAULT_PASSWORD=… node benchmarks/load/run-t1.mjs browse|dashboard [--duration 5m] [--vus 300] [--students 1000]
// Ghi benchmarks/reports/load-<kịch bản>.json (khối `meta` đủ trường, thêm k6_cpus và demo_stack). Chỉ chạy trên stack -p ep-*.
import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { meta } from "../lib/meta.mjs";

const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const [scenario, ...rest] = process.argv.slice(2);
const opt = (name, d) => (rest.includes(name) ? rest[rest.indexOf(name) + 1] : d);
const die = (m) => { console.error(m); process.exit(1); };
const project = process.env.COMPOSE_PROJECT_NAME ?? "";
if (!["browse", "dashboard"].includes(scenario)) die("Dùng: run-t1.mjs browse|dashboard [--duration 5m] [--vus 300] [--students 1000]");
if (!/^ep-/.test(project)) die("Từ chối: COMPOSE_PROJECT_NAME phải là stack đo ep-<vai> (không phải stack demo).");
if (!process.env.SEED_DEFAULT_PASSWORD) die("Thiếu SEED_DEFAULT_PASSWORD.");

const run = (cmd, args) => spawnSync(cmd, args, { cwd: root, encoding: "utf8" });
const caddy = run("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "ps", "-q", "caddy"]).stdout.trim();
if (!caddy) die(`Không thấy container caddy của project ${project} (stack đã chạy chưa?).`);
const demo = run("docker", ["ps", "-q", "--filter", "label=com.docker.compose.project=edupilot"]).stdout.trim() ? "running" : "stopped";
const cpus = "1.5";
const gw = Number(run("docker", ["ps", "-q", "--filter", `label=com.docker.compose.project=${project}`, "--filter", "label=com.docker.compose.service=gateway"]).stdout.split("\n").filter(Boolean).length);
const wk = Number(run("docker", ["ps", "-q", "--filter", `label=com.docker.compose.project=${project}`, "--filter", "label=com.docker.compose.service=worker"]).stdout.split("\n").filter(Boolean).length);
const duration = opt("--duration", "5m");
const m = { ...meta({ scenario, duration, vus: Number(opt("--vus", 300)), students: Number(opt("--students", 1000)), gateways: gw, workers: wk, provider: "fake", k6_cpus: cpus, demo_stack: demo, compose_project: project }), k6_cpus: cpus, demo_stack: demo };
mkdirSync(path.join(root, "benchmarks", "reports"), { recursive: true });
const r = spawnSync("docker", [
  "run", "--rm", "--network", `container:${caddy}`, "--cpus", cpus, "--memory", "1.5g",
  "-v", `${path.join(root, "benchmarks")}:/b`, "-w", "/b/reports", "grafana/k6", "run", "/b/load/t1.js",
  "-e", "BASE=https://localhost", "-e", `SCENARIO=${scenario}`, "-e", `DURATION=${duration}`, "-e", `VUS=${opt("--vus", 300)}`, "-e", `STUDENTS=${opt("--students", 1000)}`,
  "-e", `SEED_DEFAULT_PASSWORD=${process.env.SEED_DEFAULT_PASSWORD}`, "-e", `OUT=/b/reports/load-${scenario}.json`, "-e", `META=${JSON.stringify(m)}`,
], { cwd: root, stdio: "inherit" });
process.exit(r.status ?? 1);
