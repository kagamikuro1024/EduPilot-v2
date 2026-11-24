// Chạy: node --test deploy/caddy/blob-route.test.mjs
// DEF (QC sprint 11, F9/F11): tải PDF ở https://localhost phải đi qua Caddy cùng origin tới MinIO — Caddyfile local thiếu `handle /edupilot/*` thì PUT bị CSP / cổng chặn → «Chưa gửi được».
import { readFileSync } from "node:fs";
import test from "node:test";
import assert from "node:assert/strict";

const read = (f) => readFileSync(new URL(f, import.meta.url), "utf8");
for (const f of ["Caddyfile", "Caddyfile.vps"]) {
  test(`${f}: /edupilot/* → minio:9000`, () => {
    const s = read(f);
    assert.match(s, /handle \/edupilot\/\*\s*\{\s*reverse_proxy minio:9000/);
  });
}
test("dev.mjs: URL ký sẵn = origin HTTPS của stack (cùng origin, có cổng khi lệch)", () => {
  const s = read("../../scripts/dev.mjs");
  assert.match(s, /BLOB_PUBLIC_ENDPOINT \|\|= origin\.replace/);
  assert.match(s, /BLOB_PUBLIC_USE_SSL \|\|= "true"/);
});
