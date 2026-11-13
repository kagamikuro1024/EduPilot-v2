// TC-UX04-3: mọi luật `:hover` trong frontend/src phải nằm trong `@media (hover: hover) and (pointer: fine)`.
// Chạy: node hover-check.mjs <thư mục frontend/src>
import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
const root = process.argv[2] ?? "frontend/src";
const files = [];
(function walk(d) { for (const f of readdirSync(d)) { const p = path.join(d, f); statSync(p).isDirectory() ? walk(p) : /\.css$/.test(f) && files.push(p); } })(root);
let total = 0;
const bad = [];
for (const f of files) {
  const src = readFileSync(f, "utf8").replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, " "));
  const stack = []; // mỗi phần tử: điều kiện @media của khối đang mở (hoặc "" nếu khối thường)
  let buf = "", line = 1, start = 1;
  for (const ch of src) {
    if (ch === "\n") line++;
    if (ch === "{") {
      const head = buf.trim();
      stack.push(head.startsWith("@media") || head.startsWith("@supports") ? head : "");
      if (/:hover/.test(head) && !head.startsWith("@")) {
        total++;
        const guarded = stack.some((s) => /hover:\s*hover/.test(s));
        if (!guarded) bad.push(`${f}:${line} ${head.slice(0, 90)}`);
      }
      buf = "";
    } else if (ch === "}") { stack.pop(); buf = ""; }
    else if (ch === ";") buf = "";
    else buf += ch;
  }
}
console.log(`quy tắc :hover: ${total}, ngoài @media (hover: hover): ${bad.length}`);
bad.forEach((b) => console.log("  " + b));
process.exit(bad.length ? 1 : 0);
