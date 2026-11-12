// Dùng chung cho mọi lệnh đo (FEAT-nfr-eval US-NFR-01): khối `meta` của tệp kết quả + chặn ghi nhầm lên stack demo.
import { execFileSync } from "node:child_process";

export const MACHINE = "MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB, k6 trong VM 1,5 CPU";
export const META_FIELDS = ["commit", "timestamp", "machine", "runs", "config"];
/** Tệp kết quả k6 (`load-*.json`) cần thêm hai trường này (P-95-5). */
export const K6_FIELDS = ["k6_cpus", "demo_stack"];

/** Cổng của stack demo (project `edupilot`, không lệch cổng). Lệnh GHI dữ liệu từ chối chạy ở đây. */
const DEMO_PORTS = new Set(["", "80", "443", "3000"]);
export function refuseDemo(base) {
  const u = new URL(base);
  if (["localhost", "127.0.0.1"].includes(u.hostname) && DEMO_PORTS.has(u.port)) {
    throw new Error(`Từ chối: ${base} là stack demo. Dùng stack -p ep-<vai> với EP_PORT_OFFSET ≠ 0.`);
  }
}

/** `config`: số bản gateway / worker, provider, model, RPM, quy mô dữ liệu… (tự do, nhưng không được chứa khoá). */
export function meta(config, runs = 1) {
  let commit = "unknown";
  try { commit = execFileSync("git", ["rev-parse", "--short", "HEAD"], { encoding: "utf8" }).trim(); } catch { /* ngoài git */ }
  return { commit, timestamp: new Date().toISOString(), machine: MACHINE, runs, config };
}
