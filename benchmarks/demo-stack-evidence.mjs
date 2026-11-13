#!/usr/bin/env node
// Bằng chứng dừng / chạy lại stack demo (P-95-7 Q8; meta.demo_stack = stopped|running của mọi load-*.json). Gọi sau khi `docker compose -p edupilot stop` và sau khi `start`:
//   node benchmarks/demo-stack-evidence.mjs stopped|started [ghi chú]
// Nối một mục vào benchmarks/reports/demo-stack.json: thời điểm, `docker compose ls -a`, State.Status / FinishedAt / StartedAt từng container của project `edupilot`.
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { meta } from "./lib/meta.mjs";

const event = process.argv[2];
if (!["stopped", "started"].includes(event)) { console.error("Dùng: demo-stack-evidence.mjs stopped|started [ghi chú]"); process.exit(1); }
const sh = (...a) => execFileSync(a[0], a.slice(1), { encoding: "utf8" }).trim();
const ids = sh("docker", "ps", "-aq", "--filter", "label=com.docker.compose.project=edupilot").split("\n").filter(Boolean);
const containers = ids.length ? sh("docker", "inspect", "-f", "{{.Name}}|{{.State.Status}}|{{.State.FinishedAt}}|{{.State.StartedAt}}", ...ids).split("\n").sort().map((l) => { const [name, status, finished_at, started_at] = l.split("|"); return { name: name.slice(1), status, finished_at, started_at }; }) : [];
const file = "benchmarks/reports/demo-stack.json";
const doc = existsSync(file) ? JSON.parse(readFileSync(file, "utf8")) : { meta: meta({ note: "Bằng chứng dừng / chạy lại stack demo (project `edupilot`) quanh các lượt đo 9.5" }), events: [] };
doc.events.push({ event, at: new Date().toISOString(), note: process.argv[3] ?? "", compose_ls: sh("docker", "compose", "ls", "-a"), containers });
mkdirSync("benchmarks/reports", { recursive: true });
writeFileSync(file, JSON.stringify(doc, null, 2));
console.log(`${event}: ${containers.filter((c) => c.status === (event === "stopped" ? "exited" : "running")).length}/${containers.length} container`);
