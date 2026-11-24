// Chạy: node --test scripts/seed-gateway-down.test.mjs
// DEF-Q2-S8-03 (TC-P6-SEED-35): gateway tắt → thoát lỗi rõ, nêu API_URL và việc cần làm, không treo.
import { spawn } from "node:child_process";
import assert from "node:assert/strict";
import test from "node:test";

test("gateway tắt: «Seed lỗi: Gateway <API_URL> chưa sẵn sàng…», rc=1", async () => {
  const api = "http://127.0.0.1:1/api/v1";
  const p = spawn("node", ["scripts/seed.mjs"], { env: { PATH: process.env.PATH, HOME: process.env.HOME, API_URL: api, SEED_DEFAULT_PASSWORD: "x" } });
  let out = "";
  p.stdout.on("data", (d) => (out += d));
  p.stderr.on("data", (d) => (out += d));
  const rc = await new Promise((r) => p.on("close", r));
  assert.match(out, /Seed lỗi: Gateway http:\/\/127\.0\.0\.1:1\/api\/v1 chưa sẵn sàng/);
  assert.notEqual(rc, 0);
});
