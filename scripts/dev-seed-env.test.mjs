// Chạy: node --test scripts/dev-seed-env.test.mjs
// DEF-Q2-DEV-01: `pnpm dev` với EP_PORT_OFFSET ≠ 0 phải truyền cổng mock-imap / mock-graph (đã cộng offset) cho seed.
// docker và node (seed) giả trong PATH: không dựng stack, không chạy seed thật.
import { spawnSync } from "node:child_process";
import { chmodSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import assert from "node:assert/strict";
import test from "node:test";

test("EP_PORT_OFFSET=500 → seed nhận EP_PORT_MOCK_IMAP=8591, EP_PORT_MOCK_GRAPH=8590", () => {
  const dir = mkdtempSync(path.join(tmpdir(), "dev-seed-"));
  const dump = path.join(dir, "seed-env.txt");
  writeFileSync(path.join(dir, "docker"), "#!/bin/sh\nexit 0\n");
  writeFileSync(path.join(dir, "node"), `#!/bin/sh\necho "$EP_PORT_MOCK_IMAP $EP_PORT_MOCK_GRAPH" > ${dump}\n`);
  chmodSync(path.join(dir, "docker"), 0o755);
  chmodSync(path.join(dir, "node"), 0o755);
  const env = { PATH: `${dir}:${process.env.PATH}`, HOME: process.env.HOME, EP_PORT_OFFSET: "500", SEED_ON_EMPTY_DB: "true" };
  const r = spawnSync(process.execPath, ["scripts/dev.mjs"], { env, encoding: "utf8" });
  assert.equal(r.status, 0, r.stderr);
  assert.equal(readFileSync(dump, "utf8").trim(), "8591 8590");
});
