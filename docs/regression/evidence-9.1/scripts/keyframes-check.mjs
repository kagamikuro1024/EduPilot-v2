// TC-UX04-16: liệt kê @keyframes + thuộc tính được animate; quét delay so le. Chạy: node keyframes-check.mjs frontend/src
import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
const root = process.argv[2] ?? "frontend/src"; const files = [];
(function walk(d) { for (const f of readdirSync(d)) { const p = path.join(d, f); statSync(p).isDirectory() ? walk(p) : /\.css$/.test(f) && files.push(p); } })(root);
for (const f of files) {
  const src = readFileSync(f, "utf8").replace(/\/\*[\s\S]*?\*\//g, "");
  const re = /@keyframes\s+([\w-]+)\s*\{/g; let m;
  while ((m = re.exec(src))) {
    let i = re.lastIndex, depth = 1, body = "";
    while (i < src.length && depth) { const c = src[i++]; if (c === "{") depth++; else if (c === "}") depth--; body += c; }
    const props = new Set([...body.matchAll(/([a-z-]+)\s*:/g)].map((x) => x[1]));
    const line = src.slice(0, m.index).split("\n").length;
    console.log(`${f}:${line} ${m[1]} -> ${[...props].join(",")}`);
  }
  for (const d of src.matchAll(/(animation-delay|transition-delay)[^;]*(calc\(|nth-child|var\(--i)[^;]*/g)) console.log(`STAGGER ${f}: ${d[0].slice(0, 100)}`);
}
