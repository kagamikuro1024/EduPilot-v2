// node --test mobile/scripts/prepare.test.mjs — prepare.mjs chỉ nhận APP_SERVER_URL https (D-AS-04).
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const script = path.join(path.dirname(fileURLToPath(import.meta.url)), "prepare.mjs");
const run = (url) => {
  const cwd = mkdtempSync(path.join(tmpdir(), "prep-"));
  const env = { ...process.env };
  if (url === undefined) delete env.APP_SERVER_URL;
  else env.APP_SERVER_URL = url;
  const r = spawnSync("node", [script], { cwd, env, encoding: "utf8" });
  return { ...r, wrote: existsSync(path.join(cwd, "www", "error.html")) };
};

for (const bad of ["http://localhost:580", "http://127.0.0.1", "http://10.0.2.2:3000", "http://edupilot.example.com", "không-phải-url"]) {
  test(`từ chối ${bad}`, () => {
    const r = run(bad);
    assert.equal(r.status, 1);
    assert.match(r.stderr, /HTTPS|URL hợp lệ/);
    assert.equal(r.wrote, false);
  });
}
test("http báo gợi ý tunnel", () => assert.match(run("http://localhost:580").stderr, /tunnel/));
test("thiếu APP_SERVER_URL → lỗi", () => assert.equal(run(undefined).status, 1));
test("https hợp lệ → sinh www/", () => {
  const r = run("https://demo.trycloudflare.com");
  assert.equal(r.status, 0, r.stderr);
  assert.equal(r.wrote, true);
});
