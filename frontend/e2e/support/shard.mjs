#!/usr/bin/env node
// Chia tệp spec cho N shard CI theo THỜI GIAN (LPT: tệp nặng nhất vào shard đang nhẹ nhất), không theo số test như `--shard` của Playwright
// (DEF-R7-01: chia theo số test dồn frame / a11y / panels / exam vào shard 1 → 17 phút). In đường dẫn tệp của shard, mỗi dòng một tệp.
//   node e2e/support/shard.mjs <shard 1..N> <N>     |     node e2e/support/shard.mjs --report <N>
// Trọng số = giây CPU của tệp ở lượt CI thật (`shard-weights.json`, làm mới khi cơ cấu đổi nhiều); tệp mới / chưa có số lấy 2 s. Mọi tệp *.spec.ts đều có đúng một shard.
import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, ".."); // frontend/e2e
const specs = (d) => readdirSync(d).flatMap((n) => { const p = path.join(d, n); return statSync(p).isDirectory() ? (n === "node_modules" || n.endsWith("-snapshots") ? [] : specs(p)) : n.endsWith(".spec.ts") ? [p] : []; });
const rel = specs(root).map((p) => path.relative(root, p)).sort();
const known = JSON.parse(readFileSync(path.join(here, "shard-weights.json"), "utf8"));
const w = (f) => known[f] ?? 2; // tệp chưa có số: ở lượt đo không có dòng nào (ca bỏ qua / rất nhanh) → trọng số nhỏ

export function plan(n) {
  const bins = Array.from({ length: n }, () => ({ sum: 0, files: [] }));
  for (const f of [...rel].sort((a, b) => w(b) - w(a) || a.localeCompare(b))) {
    const b = bins.reduce((m, x) => (x.sum < m.sum ? x : m));
    b.sum += w(f);
    b.files.push(f);
  }
  return bins;
}

const [a, b] = process.argv.slice(2);
if (a === "--report") {
  plan(Number(b)).forEach((x, i) => console.log(`shard ${i + 1}: ${Math.round(x.sum)} s, ${x.files.length} tệp`));
} else {
  const i = Number(a), n = Number(b);
  if (!(i >= 1 && i <= n)) { console.error("dùng: shard.mjs <1..N> <N>"); process.exit(2); }
  console.log(plan(n)[i - 1].files.map((f) => `e2e/${f}`).join("\n"));
}
