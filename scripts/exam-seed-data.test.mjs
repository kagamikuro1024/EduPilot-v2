// Chạy: node --test scripts/exam-seed-data.test.mjs
// DEF-R5-14 (TC-PE09-07, US-PE-09 AC2): lượt code của seed minh hoạ đủ AC, WA (một phần), CE, TLE; bảng mong đợi khớp mô hình.
import { readFileSync } from "node:fs";
import test from "node:test";
import assert from "node:assert/strict";
import { CODE_ATTEMPTS, SOLUTIONS, expectedCode } from "./exam-seed-data.mjs";

const verdicts = CODE_ATTEMPTS.flatMap((c) => [expectedCode(0, c.gcd).verdict, expectedCode(1, c.words).verdict]);

test("seed có đủ kết luận AC, WA, CE, TLE", () => {
  for (const v of ["AC", "WA", "CE", "TLE"]) assert.ok(verdicts.includes(v), `thiếu ${v}: ${[...new Set(verdicts)]}`);
});

test("mọi lời giải được tham chiếu đều tồn tại", () => {
  for (const c of CODE_ATTEMPTS) for (const k of [c.gcd, c.words]) if (k) assert.ok(SOLUTIONS[k], k);
});

test("seed/expected_exam_scores.csv ghi đúng kết luận từng lượt code", () => {
  const rows = readFileSync(new URL("../seed/expected_exam_scores.csv", import.meta.url), "utf8").split("\n").filter((l) => l.startsWith("CODE,"));
  for (const c of CODE_ATTEMPTS) {
    const want = [["gcd-lon-nhat", expectedCode(0, c.gcd).verdict], ["dem-tu", expectedCode(1, c.words).verdict]].map(([s, v]) => `${s}:${v || "NONE"}`).join(";");
    assert.ok(rows.some((l) => l.split(",")[1] === c.who && l.endsWith(`,${want}`)), `${c.who}: ${want}`);
  }
});
