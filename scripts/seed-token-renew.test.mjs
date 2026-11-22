// Chạy: node --test scripts/seed-token-renew.test.mjs
// DEF-Q2-S8-05: token admin hết hạn giữa chừng (401 TOKEN_EXPIRED) → seed đăng nhập lại và thử lại, không dừng.
import { spawn } from "node:child_process";
import { createServer } from "node:http";
import assert from "node:assert/strict";
import test from "node:test";

test("401 TOKEN_EXPIRED: đăng nhập lại đúng tài khoản rồi gọi tiếp bằng token mới", async () => {
  let logins = 0;
  const seen = [];
  const srv = createServer((req, res) => {
    const u = new URL(req.url, "http://x");
    const send = (code, body) => { res.writeHead(code, { "Content-Type": "application/json" }); res.end(JSON.stringify(body)); };
    if (u.pathname.endsWith("/auth/login")) return send(200, { access_token: `tok${++logins}` });
    if (u.pathname.endsWith("/admin/courses")) {
      const t = req.headers.authorization;
      seen.push(t);
      if (t === "Bearer tok1") return send(401, { code: "TOKEN_EXPIRED" });
      return send(200, { items: [{ class_code: u.searchParams.get("q") }], next_cursor: null });
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
  assert.equal(logins, 2);
  assert.deepEqual(seen.slice(0, 2), ["Bearer tok1", "Bearer tok2"]);
});
