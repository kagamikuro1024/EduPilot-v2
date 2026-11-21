// Chạy: node --test scripts/seed-if-empty.test.mjs
// DEF-R4-08 (TC-GATEP2-09, US-P2-12 AC7): `--if-empty` nhận ra DB đã seed dù có > 100 lớp (hai lớp seed không nằm ở trang đầu của danh sách).
import { spawn } from "node:child_process";
import { createServer } from "node:http";
import test from "node:test";
import assert from "node:assert/strict";

test("--if-empty: > 100 lớp, lớp seed nằm ngoài trang đầu → «Đã có dữ liệu, không seed.»", async () => {
  const courses = [...Array.from({ length: 150 }, (_, i) => `QC${String(i).padStart(4, "0")}`), "761987", "761988"].map((class_code) => ({ class_code }));
  const srv = createServer((req, res) => {
    const u = new URL(req.url, "http://x");
    const send = (code, body) => { res.writeHead(code, { "Content-Type": "application/json" }); res.end(JSON.stringify(body)); };
    if (u.pathname.endsWith("/auth/login")) return send(200, { access_token: "t" });
    if (u.pathname.endsWith("/admin/courses")) {
      const q = u.searchParams.get("q");
      const items = (q ? courses.filter((c) => c.class_code.startsWith(q)) : courses.slice(0, 100)).slice(0, Number(u.searchParams.get("limit") ?? 100));
      return send(200, { items, next_cursor: null });
    }
    send(404, { code: "NOT_FOUND" });
  });
  await new Promise((r) => srv.listen(0, "127.0.0.1", r));
  const env = { PATH: process.env.PATH, HOME: process.env.HOME, API_URL: `http://127.0.0.1:${srv.address().port}/api/v1`, SEED_DEFAULT_PASSWORD: "x" };
  const p = spawn("node", ["scripts/seed.mjs", "--if-empty"], { env });
  let out = "";
  p.stdout.on("data", (d) => (out += d));
  p.stderr.on("data", (d) => (out += d));
  const rc = await new Promise((r) => p.on("close", r));
  srv.close();
  assert.match(out, /Đã có dữ liệu, không seed\./);
  assert.equal(rc, 0);
});
