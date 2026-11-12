#!/usr/bin/env node
// Kiểm khối `meta` của mọi `benchmarks/reports/**/*.json` (US-NFR-01 AC3). Thoát mã 1 nếu thiếu trường hoặc `machine` không đúng.
//   node benchmarks/check-meta.mjs [thư-mục]
import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { META_FIELDS, K6_FIELDS, MACHINE } from "./lib/meta.mjs";

const root = path.resolve(process.argv[2] ?? path.join(path.dirname(new URL(import.meta.url).pathname), "reports"));
const files = (d) => readdirSync(d).flatMap((n) => { const p = path.join(d, n); return statSync(p).isDirectory() ? files(p) : p.endsWith(".json") ? [p] : []; });
const LEGACY = /^(e1|pe-exam-submit-2026-10-09-.+)\.json$/; // số đo trước sprint 9.5, không có `meta`
let bad = 0;
for (const f of files(root).filter((x) => !LEGACY.test(path.basename(x)))) {
  const m = JSON.parse(readFileSync(f, "utf8")).meta;
  const miss = [...META_FIELDS, ...(path.basename(f).startsWith("load-") ? K6_FIELDS : [])].filter((k) => m?.[k] === undefined || m[k] === "");
  if (miss.length || m.machine !== MACHINE) { bad++; console.error(`${path.relative(process.cwd(), f)}: ${miss.length ? `thiếu ${miss.join(", ")}` : "machine sai"}`); }
}
console.log(bad ? `${bad} tệp không hợp lệ` : "meta đủ");
process.exit(bad ? 1 : 0);
