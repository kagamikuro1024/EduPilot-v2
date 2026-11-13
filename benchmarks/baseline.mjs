#!/usr/bin/env node
// US-NFR-07 AC1–AC2: số đo nền của stack đo: RAM / CPU nghỉ từng container (`docker stats`), kích thước image, thời gian `up` → `/api/v1/readyz` = 200 (lạnh, 3 lần, trung vị).
// AC3 (p95 ≥ 12 endpoint) do `benchmarks/load/run-t1.mjs baseline` ghi `load-baseline-p95.json`; script này gộp vào `benchmarks/reports/baseline.json`.
//   COMPOSE_PROJECT_NAME=ep-t1 EP_PORT_HTTPS=843 node benchmarks/baseline.mjs [--no-startup]
// "Lạnh" = `compose down` (giữ volume, KHÔNG -v) rồi `up -d`; mất vài phút nên có cờ bỏ. Chỉ stack -p ep-*.
import { spawnSync } from "node:child_process";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { meta } from "./lib/meta.mjs";

const project = process.env.COMPOSE_PROJECT_NAME ?? "";
if (!/^ep-/.test(project)) { console.error("Từ chối: COMPOSE_PROJECT_NAME phải là stack đo ep-<vai>."); process.exit(1); }
const port = process.env.EP_PORT_HTTPS ?? "";
if (!port || port === "443") { console.error("Cần EP_PORT_HTTPS của stack đo (≠ 443)."); process.exit(1); }
const sh = (cmd, args, opt = {}) => { const r = spawnSync(cmd, args, { encoding: "utf8", ...opt }); if (r.status !== 0) throw new Error(`${cmd} ${args.join(" ")}: ${r.stderr}`); return r.stdout.trim(); };
const compose = (...a) => sh("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-f", "benchmarks/load/pg-stats.yml", "-p", project, ...a]);
const toMiB = (s) => { const m = /([\d.]+)(KiB|MiB|GiB)/.exec(s); return m ? Number(m[1]) * { KiB: 1 / 1024, MiB: 1, GiB: 1024 }[m[2]] : 0; };

const stats = sh("docker", ["stats", "--no-stream", "--format", "{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}", ...sh("docker", ["ps", "--filter", `label=com.docker.compose.project=${project}`, "--format", "{{.Names}}"]).split("\n")]).split("\n").map((l) => { const [name, cpu, mem] = l.split("|"); return { name, cpu_pct: Number(cpu.replace("%", "")), mem_mib: Number(toMiB(mem.split("/")[0]).toFixed(1)) }; });
const images = {};
for (const svc of ["gateway", "worker", "frontend"]) {
  const id = sh("docker", ["ps", "--filter", `label=com.docker.compose.project=${project}`, "--filter", `label=com.docker.compose.service=${svc}`, "--format", "{{.Image}}"]).split("\n")[0];
  images[svc] = { image: id, mb: Number((Number(sh("docker", ["image", "inspect", "-f", "{{.Size}}", id])) / 1e6).toFixed(1)) };
}
const startup = [];
if (!process.argv.includes("--no-startup")) {
  for (let i = 0; i < 3; i++) {
    compose("down");
    const t0 = Date.now();
    compose("up", "-d");
    for (;;) { const r = spawnSync("curl", ["-ks", "-o", "/dev/null", "-w", "%{http_code}", `https://localhost:${port}/api/v1/readyz`], { encoding: "utf8" }); if (r.stdout.trim() === "200") break; if (Date.now() - t0 > 300000) throw new Error("readyz không lên sau 5 phút"); spawnSync("sleep", ["1"]); }
    startup.push(Number(((Date.now() - t0) / 1000).toFixed(1)));
  }
}
const sorted = [...startup].sort((a, b) => a - b);
const p95 = existsSync("benchmarks/reports/load-baseline-p95.json") ? JSON.parse(readFileSync("benchmarks/reports/load-baseline-p95.json", "utf8")).endpoints : null;
const doc = { meta: meta({ compose_project: project, note: "stack T1 (DB edupilot_t1) lúc nghỉ; không so với hệ cũ (D45) hay VPS cũ" }), idle: { containers: stats, total_mem_mib: Number(stats.reduce((s, c) => s + c.mem_mib, 0).toFixed(1)) }, images, startup_seconds: { runs: startup, median: sorted[Math.floor(sorted.length / 2)] ?? null }, p95_endpoints: p95 };
writeFileSync("benchmarks/reports/baseline.json", JSON.stringify(doc, null, 2));
const md = ["# Số đo nền (MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB)", "", "| Container | CPU nghỉ % | RAM MiB |", "| --- | ---: | ---: |", ...stats.map((c) => `| ${c.name} | ${c.cpu_pct} | ${c.mem_mib} |`), `| **Tổng** | | **${doc.idle.total_mem_mib}** |`, "", "| Image | MB |", "| --- | ---: |", ...Object.entries(images).map(([k, v]) => `| ${k} | ${v.mb} |`), "", `Khởi động lạnh (up → readyz 200): ${startup.join(" / ") || "chưa đo"} s; trung vị ${doc.startup_seconds.median ?? "-"} s.`, ""];
if (p95) md.push("| Endpoint | p50 ms | p95 ms | p99 ms | Đạt |", "| --- | ---: | ---: | ---: | --- |", ...Object.entries(p95).map(([k, v]) => `| ${k} | ${v.p50.toFixed(1)} | ${v.p95.toFixed(1)} | ${v.p99.toFixed(1)} | ${v.ok ? "PASS" : "FAIL"} |`), "");
writeFileSync("benchmarks/reports/baseline.md", md.join("\n"));
console.log(md.join("\n"));
