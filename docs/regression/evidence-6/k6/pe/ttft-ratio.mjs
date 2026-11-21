// Góp ý #18: tỉ lệ p95 TTFT `mixed` / `chat` của từng cặp (-1, -2, -3) từ benchmarks/reports; đạt khi TRUNG VỊ ≤ 1,2.
import { readdirSync, readFileSync } from "node:fs";

const dir = "benchmarks/reports";
const files = readdirSync(dir);
const p95 = (kind, run) => {
  const f = files.filter((n) => new RegExp(`^pe-exam-submit-.*-${kind}-${run}\\.json$`).test(n)).sort().pop();
  if (!f) throw new Error(`thiếu báo cáo ${kind}-${run}`);
  return JSON.parse(readFileSync(`${dir}/${f}`, "utf8")).metrics.chat_ttft_ms["p(95)"];
};
const ratios = [1, 2, 3].map((i) => {
  const c = p95("chat", i), m = p95("mixed", i);
  console.log(`cặp ${i}: chat p95 ${c.toFixed(1)} ms · mixed p95 ${m.toFixed(1)} ms · tỉ lệ ${(m / c).toFixed(3)}`);
  return m / c;
});
const med = [...ratios].sort((a, b) => a - b)[1];
console.log(`trung vị tỉ lệ ${med.toFixed(3)} (ngưỡng 1,2)`);
process.exit(med <= 1.2 ? 0 : 1);
