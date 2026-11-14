#!/usr/bin/env node
// Dựng PDF mẫu cho luyện đề (US-P9-02, bàn giao QC P-10-7) từ seed/practice/de-mau.json — nguồn duy nhất của đề + đáp án:
//   seed/practice/de-mau-trac-nghiem.pdf   24 câu, 6 câu / trang (4 trang)
//   seed/practice/dap-an-mau.pdf           bảng "N. X" đủ 24 câu
//   seed/practice/dap-an-thieu.pdf         thiếu câu 22 và 24 (→ NO_ANSWER_MATCH)
//   seed/practice/de-mau-loi-cau.pdf       5 câu; tiêu đề khớp bản ghi lỗi mức câu của seed/demo/fake_llm.json
//   seed/practice/de-mau-loi-schema.pdf    tiêu đề khớp bản ghi sai schema (job FAILED, không lưu câu nào)
// Chromium của Playwright (đã có trong bảng thư viện). Kết quả được commit; chạy lại: `node scripts/make-practice-pdf.mjs`.
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const { chromium } = createRequire(path.join(root, "frontend", "package.json"))("@playwright/test");
const src = JSON.parse(readFileSync(path.join(root, "seed", "practice", "de-mau.json"), "utf8"));
const esc = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
const L = "ABCD";
const paper = (title, qs, perPage) => `<h1>${esc(title)}</h1>` + qs.map((q, i) => (i && i % perPage === 0 ? '<div class="pb"></div>' : "") +
  `<p class="q">Câu ${q.number}. ${esc(q.stem)}</p>` + q.options.map((o, k) => `<p class="o">${L[k]}. ${esc(o)}</p>`).join("")).join("\n");
const key = (title, qs) => `<h1>${esc(title)}</h1>` + qs.map((q) => `<p class="o">${q.number}. ${L[q.correct]}</p>`).join("\n");
const html = (body) => `<!doctype html><meta charset="utf-8"><style>
  body { font: 10.5pt/1.4 "Be Vietnam Pro", "DejaVu Sans", Arial, sans-serif; margin: 0; font-kerning: none; }
  h1 { font-size: 16pt; margin: 0 0 14pt; } p { margin: 0; } .q { margin: 12pt 0 3pt; font-weight: 600; } .o { margin: 0 0 2pt 14pt; } .pb { break-after: page; height: 0; }
</style>${body}`;

const qs = src.questions;
const bad = qs.slice(0, 5);
const files = {
  "de-mau-trac-nghiem": html(paper(src.title, qs, src.per_page)),
  "dap-an-mau": html(key("ĐÁP ÁN ĐỀ MẪU — MẠNG MÁY TÍNH", qs)),
  "dap-an-thieu": html(key("ĐÁP ÁN ĐỀ MẪU THIẾU — MẠNG MÁY TÍNH", qs.filter((q) => q.number !== 22 && q.number !== 24))),
  "de-mau-loi-cau": html(paper("ĐỀ MẪU LỖI CÂU — MẠNG MÁY TÍNH", bad, 8)),
  "de-mau-loi-schema": html(paper("ĐỀ MẪU LỖI SCHEMA — MẠNG MÁY TÍNH", bad.slice(0, 2), 8)),
};
const browser = await chromium.launch();
try {
  for (const [name, h] of Object.entries(files)) {
    const page = await browser.newPage();
    await page.setContent(h);
    const pdf = await page.pdf({ format: "A4", margin: { top: "18mm", bottom: "18mm", left: "20mm", right: "20mm" } });
    writeFileSync(path.join(root, "seed", "practice", `${name}.pdf`), pdf);
    console.log(`${name}.pdf ${pdf.length} byte`);
    await page.close();
  }
} finally {
  await browser.close();
}
