// Chuyển <text> trong logo SVG thành <path> bằng Be Vietnam Pro Bold (SVG nạp qua <img> không dùng được webfont — M19).
// Dùng: node outline-logo.mjs <BeVietnamPro-Bold.ttf> <vào.svg> <ra.svg>   (cần `opentype.js`)
import fs from 'fs';
import opentype from 'opentype.js';

const [, , fontPath, src, out] = process.argv;
const font = opentype.parse(fs.readFileSync(fontPath).buffer);
const svg = fs.readFileSync(src, 'utf8').replace(/<text ([^>]*)>([^<]*)<\/text>/g, (_, attrs, text) => {
  const a = Object.fromEntries([...attrs.matchAll(/([\w-]+)="([^"]*)"/g)].map((m) => [m[1], m[2]]));
  const size = Number(a['font-size']);
  const path = font.getPath(text, Number(a.x), Number(a.y), size, { kerning: true, letterSpacing: Number(a['letter-spacing'] ?? 0) / size });
  return `<path fill="${a.fill}" d="${path.toPathData(2)}"/>`;
});
fs.writeFileSync(out, svg);
