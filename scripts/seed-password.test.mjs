// Chạy: node --test scripts/seed-password.test.mjs
// DEF-R9.5-19: seed từ chối SEED_DEFAULT_PASSWORD rỗng / trắng TRƯỚC mọi lời gọi mạng; không đặt thì đọc .env.local / .env.example.
import { spawnSync } from "node:child_process";
import test from "node:test";
import assert from "node:assert/strict";

const run = (extra, unset = []) => {
  const env = { PATH: process.env.PATH, HOME: process.env.HOME, API_URL: "http://127.0.0.1:1/api/v1", ...extra };
  for (const k of unset) delete env[k];
  return spawnSync("node", ["scripts/seed.mjs"], { env, encoding: "utf8", timeout: 20000 });
};
const refused = (r) => r.status !== 0 && /Thiếu SEED_DEFAULT_PASSWORD/.test(r.stderr);

test("đặt rỗng rõ ràng → từ chối (không rơi về .env.local)", () => {
  assert.ok(refused(run({ SEED_DEFAULT_PASSWORD: "" })));
});
test("đặt toàn khoảng trắng → từ chối", () => {
  assert.ok(refused(run({ SEED_DEFAULT_PASSWORD: "   " })));
});
test("không đặt → đọc .env.local / .env.example, đi tiếp tới bước mạng (không phải lỗi thiếu mật khẩu)", () => {
  const r = run({}, ["SEED_DEFAULT_PASSWORD"]);
  assert.ok(!refused(r), r.stderr);
});
