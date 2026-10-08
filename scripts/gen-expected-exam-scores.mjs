// Tính `seed/expected_exam_scores.csv` (US-PE-09 AC3) bằng mô hình ĐỘC LẬP với gateway: số hữu tỉ BigInt, quy tắc PARTIAL và bước 0,01 viết lại từ SRS (không gọi mã Go).
// Mỗi dòng có cột `formula` (từng câu / từng bài) để đối chiếu bằng tay hoặc bằng bảng tính. Chạy lại phải cho tệp y hệt: `node scripts/gen-expected-exam-scores.mjs --check`.
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { CODE, CODE_ATTEMPTS, MCQ_STUDENTS, QUESTIONS, earnedMcq, expectedCode, mcqAnswers, round2 } from "./exam-seed-data.mjs";

const out = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "seed", "expected_exam_scores.csv");
const gcd = (a, b) => (b === 0n ? a : gcd(b, a % b));
const add = ([an, ad], [bn, bd]) => {
  const n = an * bd + bn * ad;
  const d = ad * bd;
  const g = gcd(n < 0n ? -n : n, d) || 1n;
  return [n / g, d / g];
};

const rows = [["exam", "account", "label", "kind", "expected_score", "formula", "expected_verdict"]];
for (const s of MCQ_STUDENTS) {
  let raw = [0n, 1n];
  const parts = [];
  for (const a of mcqAnswers(s.id)) {
    const e = earnedMcq(QUESTIONS[a.qi], a.ans);
    raw = add(raw, e);
    parts.push(e[1] === 1n ? String(e[0]) : `${e[0]}/${e[1]}`);
  }
  // thang 10, tổng điểm các câu = 10 ⇒ điểm = raw × 10 ÷ 10
  rows.push(["MCQ", s.who, String(s.id), "MCQ", round2(raw[0] * 10n, raw[1] * 10n), `(${parts.join("+")}) x 10 / 10`, ""]);
}
for (const c of CODE_ATTEMPTS) {
  let raw = [0n, 1n];
  const parts = [];
  const verdicts = [];
  [c.gcd, c.words].forEach((sol, i) => {
    const e = expectedCode(i, sol);
    raw = add(raw, [5n * BigInt(e.passed), BigInt(e.total)]);
    parts.push(`5 x ${e.passed}/${e.total}`);
    verdicts.push(`${CODE[i].slug}:${sol == null ? "NONE" : e.verdict}`);
  });
  rows.push(["CODE", c.who, c.label, "CODE", round2(raw[0] * 10n, raw[1] * 10n), `(${parts.join(" + ")}) x 10 / 10`, verdicts.join(";")]);
}
const text = rows.map((r) => r.join(",")).join("\n") + "\n";
if (process.argv.includes("--check")) {
  if (readFileSync(out, "utf8") !== text) {
    console.error("seed/expected_exam_scores.csv lệch mô hình: chạy node scripts/gen-expected-exam-scores.mjs");
    process.exit(1);
  }
  console.log(`OK ${rows.length - 1} dòng`);
} else {
  writeFileSync(out, text);
  console.log(`Đã ghi ${out} (${rows.length - 1} dòng)`);
}
