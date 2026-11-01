// MSSV của từng tài khoản seed (cùng bộ sinh với seed.mjs: lấy từ `GET /me/profile`-free nguồn — expected_final_grades_v2.csv của BA có đủ account → MSSV).
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
/** account → MSSV cho 30 sinh viên lớp 1. */
export function accountCodes() {
  const m = new Map();
  for (const l of readFileSync(path.join(root, "seed", "expected_final_grades_v2.csv"), "utf8").trim().split("\n").slice(1)) {
    const c = l.split(",");
    m.set(c[0], c[1]);
  }
  return m;
}
