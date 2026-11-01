#!/usr/bin/env node
// Dựng seed/demo/quy-che-lop{1,2}.pdf từ docs/specs/FEAT-gradebook/fixtures/quy-che-lop{1,2}.md (US-S8-SEED AC1/AC4).
// Chữ giữ nguyên từng câu; mỗi dấu `<!-- trang N -->` là một ngắt trang cứng; PDF có lớp chữ thật (Chromium của Playwright — đã có trong bảng thư viện).
// Chạy lại khi BA đổi .md: `node scripts/make-policy-pdf.mjs`. Kết quả được commit (seed không cần trình duyệt).
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const { chromium } = createRequire(path.join(root, "frontend", "package.json"))("@playwright/test");
const esc = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");

function html(md) {
  const body = md
    .slice(md.indexOf("<!-- trang 1 -->")) // chú thích của BA ở đầu tệp (có chứa chính dấu trang làm ví dụ) không vào PDF
    .split("\n")
    .map((l) => {
      const m = /^<!--\s*trang\s+(\d+)\s*-->/.exec(l);
      if (m) return m[1] === "1" ? "" : '<div class="pb"></div>';
      if (l.startsWith("## ")) return `<h2>${esc(l.slice(3))}</h2>`;
      if (l.startsWith("# ")) return `<h1>${esc(l.slice(2))}</h1>`;
      return l.trim() ? `<p>${esc(l)}</p>` : "";
    })
    .join("\n");
  return `<!doctype html><meta charset="utf-8"><style>
    body { font: 10.5pt/1.45 "Be Vietnam Pro", "DejaVu Sans", Arial, sans-serif; margin: 0; font-kerning: none; }
    h1 { font-size: 17pt; margin: 0 0 8pt; } h2 { font-size: 13pt; margin: 14pt 0 5pt; } p { margin: 0 0 26pt; }
    .pb { break-after: page; height: 0; }
  </style>${body}`;
}

const browser = await chromium.launch();
try {
  mkdirSync(path.join(root, "seed", "demo"), { recursive: true });
  // `node scripts/make-policy-pdf.mjs v2` chỉ dựng bản 2 của lớp 1 (US-S9-SEED AC4; nguồn của BA ở FEAT-assignments), không ghi đè hai bản cũ.
  const jobs = process.argv[2] === "v2" ? [["1-v2", "docs/specs/FEAT-assignments/fixtures/quy-che-lop1-v2.md"]] : [[1, "docs/specs/FEAT-gradebook/fixtures/quy-che-lop1.md"], [2, "docs/specs/FEAT-gradebook/fixtures/quy-che-lop2.md"]];
  for (const [n, src] of jobs) {
    const md = readFileSync(path.join(root, src), "utf8");
    const page = await browser.newPage();
    await page.setContent(html(md));
    const pdf = await page.pdf({ format: "A4", margin: { top: "18mm", bottom: "18mm", left: "20mm", right: "20mm" } });
    writeFileSync(path.join(root, "seed", "demo", `quy-che-lop${n}.pdf`), pdf);
    await page.close();
    console.log(`quy-che-lop${n}.pdf ${pdf.length} byte`);
  }
} finally {
  await browser.close();
}
