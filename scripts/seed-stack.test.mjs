// Chạy: node --test scripts/seed-stack.test.mjs
// DEF-R105S-05: seed từ chối khi lệnh admin và API_URL không cùng DB.
import test from "node:test";
import assert from "node:assert/strict";
import { stackMismatch } from "./seed-stack.mjs";

const base = { apiId: "01a2-A", adminTarget: "docker compose -p ep-qc exec gateway", api: "https://localhost:4643/api/v1" };

test("cùng id lớp → cho chạy", () => {
  assert.equal(stackMismatch({ ...base, adminOut: "seed-identity: 01a2-A" }), null);
});
test("id lớp khác (DB khác) → từ chối, nêu project đích", () => {
  const m = stackMismatch({ ...base, adminOut: "seed-identity: 01a2-B" });
  assert.match(m, /KHÁC/);
  assert.match(m, /ep-qc/);
});
test("lệnh admin lỗi / không có lớp → từ chối", () => {
  assert.match(stackMismatch({ ...base, adminOut: null }), /không có/);
});
