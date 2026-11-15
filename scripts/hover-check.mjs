// Mọi luật `:hover` trong frontend/src phải nằm trong `@media (hover: hover) and (pointer: fine)` (FEAT-uiux-91 US-04 AC2): máy cảm ứng không dính trạng thái hover.
// Chạy: node scripts/hover-check.mjs [thư mục src]   → in `tệp:dòng luật` cho mỗi vi phạm, thoát 1 nếu có.
import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";

const root = process.argv[2] ?? "frontend/src";
const files = [];
(function walk(d) { for (const f of readdirSync(d)) { const p = path.join(d, f); statSync(p).isDirectory() ? walk(p) : /\.css$/.test(f) && files.push(p); } })(root);

const bad = [];
for (const f of files) {
  const src = readFileSync(f, "utf8").replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, " "));
  const stack = []; // điều kiện @media / @supports của khối đang mở ("" nếu khối thường)
  let buf = "", line = 1;
  for (const ch of src) {
    if (ch === "\n") line++;
    if (ch === "{") {
      const head = buf.trim();
      stack.push(head.startsWith("@media") || head.startsWith("@supports") ? head : "");
      if (/:hover/.test(head) && !head.startsWith("@") && !stack.some((s) => /hover:\s*hover/.test(s))) bad.push(`${f}:${line} ${head.slice(0, 90)}`);
      buf = "";
    } else if (ch === "}") { stack.pop(); buf = ""; }
    else if (ch === ";") buf = "";
    else buf += ch;
  }
}
for (const b of bad) console.log(b);
process.exit(bad.length ? 1 : 0);
