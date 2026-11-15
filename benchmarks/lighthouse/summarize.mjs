#!/usr/bin/env node
// Trung vị LCP / TBT / CLS / kích thước script mỗi route từ benchmarks/.lighthouse-raw/{simulate,devtools}/*.json → benchmarks/reports/lighthouse.json (có `meta`).
// Ngưỡng hiện hành (không nới): LCP ≤ 2.500 ms (devtools), TBT ≤ 200 ms (mô phỏng), CLS ≤ 0,1, script ≤ 256 KB.
import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { meta } from "../lib/meta.mjs";
const med = (a) => { const s = [...a].sort((x, y) => x - y); return s[Math.floor(s.length / 2)]; };
const out = {};
for (const mode of ["simulate", "devtools"]) {
  const dir = `benchmarks/.lighthouse-raw/${mode}`;
  if (!existsSync(dir)) continue;
  const by = {};
  for (const f of readdirSync(dir).filter((x) => x.endsWith(".json"))) {
    const l = JSON.parse(readFileSync(`${dir}/${f}`, "utf8"));
    const route = new URL(l.finalDisplayedUrl ?? l.requestedUrl).pathname;
    const a = l.audits;
    (by[route] ??= []).push({ lcp: a["largest-contentful-paint"].numericValue, tbt: a["total-blocking-time"].numericValue, cls: a["cumulative-layout-shift"].numericValue, script: a["resource-summary"].details.items.find((i) => i.resourceType === "script")?.transferSize ?? 0 });
  }
  out[mode] = Object.fromEntries(Object.entries(by).map(([r, xs]) => [r, { runs: xs.length, lcp_ms: Math.round(med(xs.map((x) => x.lcp))), tbt_ms: Math.round(med(xs.map((x) => x.tbt))), cls: Number(med(xs.map((x) => x.cls)).toFixed(3)), script_kb: Math.round(med(xs.map((x) => x.script)) / 1024) }]));
}
writeFileSync("benchmarks/reports/lighthouse.json", JSON.stringify({ meta: meta({ thresholds: { lcp_ms_devtools: 2500, tbt_ms_simulate: 200, cls: 0.1, script_kb: 250 } }), ...out }, null, 2));
for (const [m, rs] of Object.entries(out)) { console.log(m); for (const [r, v] of Object.entries(rs)) console.log(`  ${r.padEnd(16)} LCP ${v.lcp_ms} TBT ${v.tbt_ms} CLS ${v.cls} script ${v.script_kb} KB`); }
