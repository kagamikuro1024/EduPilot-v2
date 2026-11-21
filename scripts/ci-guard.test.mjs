// DEF-R7-06 / DEF-R7-07 (US-CI-02): mọi job CI có timeout-minutes; shard e2e rỗng không chạy gì (không chạy cả bộ); mỗi tệp spec thuộc đúng một shard.
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";

const shard = (i, n) => execFileSync("node", ["frontend/e2e/support/shard.mjs", String(i), String(n)], { encoding: "utf8" });

test("shard rỗng in ra ĐÚNG rỗng (mapfile không được ra một phần tử rỗng → playwright chạy cả bộ)", () => {
  assert.equal(shard(200, 200), "");
});

test("mỗi tệp spec thuộc đúng một shard", () => {
  const all = [1, 2, 3, 4].flatMap((i) => shard(i, 4).split("\n").filter(Boolean));
  assert.ok(all.length > 100);
  assert.equal(new Set(all).size, all.length);
});

test("mọi job trong ci.yml khai báo timeout-minutes", () => {
  const jobs = readFileSync(".github/workflows/ci.yml", "utf8").split(/^jobs:\s*$/m)[1].split(/^  ([a-z0-9-]+):\s*$/m);
  const missing = [];
  for (let i = 1; i < jobs.length; i += 2) if (!/^ {4}timeout-minutes:\s*\d+/m.test(jobs[i + 1])) missing.push(jobs[i]);
  assert.deepEqual(missing, []);
});

test("push chỉ trên main: nhánh có PR không chạy hai lượt (push + pull_request) cho cùng một SHA", () => {
  const on = readFileSync(".github/workflows/ci.yml", "utf8").split(/^on:\s*$/m)[1].split(/^\S/m)[0];
  assert.match(on, /^ {2}push:\s*\n(?: {4}#.*\n)* {4}branches:\s*\[main\]/m);
});
