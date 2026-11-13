#!/usr/bin/env node
// P-95-7 Q20: chạy TRƯỚC `docker compose -p ep-research down -v` (llm_audit mất theo volume). Thoát mã ≠ 0 nếu thiếu thứ gì.
//   COMPOSE_PROJECT_NAME=ep-research node benchmarks/real-llm/export-evidence.mjs
// Ghi vào benchmarks/reports/real-llm/:
//   (1) kết quả thô từng mẫu: `real_llm.py` đã ghi *.json (extract, grade-*, mask, chat, threads, misc) — script chỉ kiểm có mặt và có `meta`;
//   (2) llm-audit-summary.json: llm_audit TỔNG HỢP theo ngày (giờ VN) / model / fallback_index / task / status / error_kind (đếm, token, độ trễ; KHÔNG payload — bảng không có cột payload);
//   (3) masked-payloads.json: ≥ 15 payload ĐÃ CHE lấy từ TestNoPayloadLeak (đường che thật + provider giả ghi lại), vì gateway không lưu payload gửi nhà cung cấp.
import { spawnSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import path from "node:path";
import { meta } from "../lib/meta.mjs";

const project = process.env.COMPOSE_PROJECT_NAME ?? "";
const die = (m) => { console.error(m); process.exit(1); };
if (!/^ep-/.test(project)) die("Từ chối: COMPOSE_PROJECT_NAME phải là stack đo ep-<vai>.");
const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const out = path.join(root, "benchmarks", "reports", "real-llm");

const raw = existsSync(out) ? readdirSync(out).filter((f) => f.endsWith(".json") && !["llm-audit-summary.json", "masked-payloads.json"].includes(f)) : [];
const need = ["extract.json", "mask.json", "chat.json"];
const missing = need.filter((f) => !raw.includes(f));
if (missing.length) die(`Thiếu kết quả thô: ${missing.join(", ")} (chạy real_llm.py trước).`);

const q = `begin read only; select coalesce(json_agg(t), '[]') from (
  select (created_at at time zone 'Asia/Ho_Chi_Minh')::date as day, provider, model, fallback_index, task, status, coalesce(error_kind, '') as error_kind,
         count(*) as calls, sum(tokens_in) as tokens_in, sum(tokens_out) as tokens_out, round(avg(latency_ms)) as avg_latency_ms, max(latency_ms) as max_latency_ms
  from llm_audit group by 1, 2, 3, 4, 5, 6, 7 order by 1, 2, 3, 4, 5) t; rollback;`;
const r = spawnSync("docker", ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, "exec", "-T", "postgres", "psql", "-U", "edupilot", "-d", "edupilot", "-qAt", "-v", "ON_ERROR_STOP=1"], { cwd: root, input: q, encoding: "utf8" });
if (r.status !== 0) die(`Không đọc được llm_audit: ${r.stderr}`);
const rows = JSON.parse(r.stdout.replace(/^BEGIN\n?/m, "").replace(/\nROLLBACK\s*$/, "").trim());
const perDay = {};
for (const x of rows) if (x.task !== "EMBEDDING") perDay[x.day] = (perDay[x.day] ?? 0) + x.calls;
writeFileSync(path.join(out, "llm-audit-summary.json"), JSON.stringify({ meta: meta({ compose_project: project, source: "llm_audit (read-only, không payload)" }), generation_calls_per_day: perDay, rows }, null, 2));

const tmp = path.join(out, ".masked.tmp");
const g = spawnSync("go", ["test", "-count=1", "./internal/integration", "-run", "TestNoPayloadLeak"], { cwd: path.join(root, "backend-go"), env: { ...process.env, EXPORT_MASKED_PAYLOADS: tmp }, encoding: "utf8" });
if (g.status !== 0 || !existsSync(tmp)) die(`Xuất payload đã che thất bại:\n${(g.stdout + g.stderr).slice(-600)}`);
const body = JSON.parse(readFileSync(tmp, "utf8")); rmSync(tmp);
writeFileSync(path.join(out, "masked-payloads.json"), JSON.stringify({ meta: meta({ source: "TestNoPayloadLeak" }), ...body }, null, 2));
console.log(`đã xuất: ${raw.length} kết quả thô · llm_audit ${rows.length} dòng tổng hợp (${Object.entries(perDay).map(([d, n]) => `${d}: ${n}`).join(", ") || "rỗng"}) · ${body.count} payload che`);
