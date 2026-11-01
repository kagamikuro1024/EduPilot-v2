// Dữ liệu Bài tập 03 / 04 của seed sprint 9 (US-S9-SEED): đọc ĐÚNG các tệp đáp án của BA ở docs/specs/FEAT-assignments/fixtures — không chép số liệu vào mã.
// Dùng chung cho `seed.mjs` và `check-assignments-seed.mjs`.
import { readFileSync } from "node:fs";
import path from "node:path";

export const FIX = (root) => path.join(root, "docs", "specs", "FEAT-assignments", "fixtures");
const read = (root, ...p) => readFileSync(path.join(FIX(root), ...p), "utf8");

/** seed-plan.csv → [{account, sample1, channel, hours, sample2, match, final, note}] (dòng "(ngoài lớp)" và sinh viên không nộp giữ nguyên). */
export function loadPlan(root) {
  const lines = read(root, "seed-plan.csv").trim().split("\n");
  return lines.slice(1).map((l) => {
    const c = [];
    for (const m of l.matchAll(/("([^"]|"")*"|[^,]*)(,|$)/g)) {
      if (m[0] === "" ) break;
      c.push(m[1].replace(/^"|"$/g, "").replace(/""/g, '"'));
      if (m[3] === "") break;
    }
    return { account: c[0], sample1: c[1], channel: c[2], hours: c[3] === "" ? null : Number(c[3]), sample2: c[4], match: c[5], final: c[6], note: c[7] ?? "" };
  });
}

export const expectedGrading = (root) => JSON.parse(read(root, "expected_grading.json"));
/** Nội dung bài làm của một mẫu (S1…S7). */
export function sampleText(root, id) {
  const s = expectedGrading(root).find((e) => e.id === id);
  if (!s) throw new Error(`Không có mẫu ${id} trong expected_grading.json`);
  return { file: path.basename(s.file), text: read(root, s.file) };
}

/** Bỏ khối chú thích HTML `<!-- … -->` (ghi chú nội bộ của BA ở đầu tệp đề bài / đáp án): không nạp vào `description` / `reference_answer` (BUG-S9-02, proposals #17). */
export const stripHtmlComments = (s) => (typeof s === "string" ? s.replace(/<!--[\s\S]*?-->/g, "").replace(/\n{3,}/g, "\n\n").trim() : s);

export function createBody(root) {
  const b = JSON.parse(read(root, "create-bt03.json"));
  b.description = stripHtmlComments(b.description);
  b.reference_answer = stripHtmlComments(b.reference_answer);
  if (Array.isArray(b.rubric)) b.rubric = b.rubric.map((c) => ({ ...c, name: stripHtmlComments(c.name), guidance: stripHtmlComments(c.guidance) }));
  return b;
}
export const teamsMeta = (root) => JSON.parse(readFileSync(path.join(root, "backend-go", "testdata", "mock-graph", "meta.json"), "utf8"));
export const TEAMS_TITLE_04 = "Bài tập 04 — Phân tích một vụ tấn công thực tế (Teams)";

export const FOREIGN_MAIL = { from: "ban.la@example.org", subject: "[BT03] 20250001", sample: "S3", hours: -20, message_id: "seed-bt03-foreign@edupilot.test" };
/** sv29: thư sai mẫu (không có mã bài), người gửi dùng email cá nhân không có trong hệ thống. */
export const PERSONAL_MAIL = (account) => ({ from: `${account}.ca-nhan@mail.example`, subject: "Bài tập 3 của em", message_id: `seed-bt03-${account}@edupilot.test` });
