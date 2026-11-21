// Chạy: node --test deploy/caddy/log-redact.test.mjs   (cần docker; tự kéo ảnh caddy:2; không có docker thì bỏ qua)
// DEF-R4-05 (TC-P201-11, US-P2-04 AC13): token mời / mã tham gia không được lọt vào log truy cập qua `request>uri` VÀ
// các tiêu đề Next.js (Next-Url, Next-Router-State-Tree) / Referer khi điều hướng RSC rời trang /invite/<token>.
import { execFileSync, spawnSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import test from "node:test";
import assert from "node:assert/strict";

const docker = spawnSync("docker", ["image", "inspect", "caddy:2"]).status === 0 || spawnSync("docker", ["pull", "-q", "caddy:2"]).status === 0;
const SECRET = "tkn-AbC123_secret-value-0123456789abcdefghi";

for (const f of ["Caddyfile", "Caddyfile.vps"]) {
  test(`${f}: log truy cập che token trong uri + tiêu đề Next.js / Referer`, { skip: !docker }, async () => {
    const src = readFileSync(new URL(f, import.meta.url), "utf8");
    const i = src.indexOf("\tlog {");
    const block = src.slice(i, src.indexOf("\n\t}\n", i) + 4);
    const name = `ep-caddy-logtest-${process.pid}`;
    const dir = execFileSync("mktemp", ["-d"]).toString().trim();
    writeFileSync(`${dir}/Caddyfile`, `{\n\tauto_https off\n}\n:8080 {\n${block}\trespond ok\n}\n`);
    try {
      // docker cp thay cho -v: thư mục tạm của macOS không được Colima chia sẻ
      execFileSync("docker", ["create", "--name", name, "-p", "127.0.0.1::8080", "caddy:2"]);
      execFileSync("docker", ["cp", `${dir}/Caddyfile`, `${name}:/etc/caddy/Caddyfile`]);
      execFileSync("docker", ["start", name]);
      const port = execFileSync("docker", ["port", name, "8080/tcp"]).toString().trim().split(":").pop();
      for (let n = 0; n < 40; n++) {
        if (spawnSync("curl", ["-s", "-o", "/dev/null", `http://127.0.0.1:${port}/`]).status === 0) break;
        await new Promise((r) => setTimeout(r, 250));
      }
      const tree = encodeURIComponent(JSON.stringify(["", { children: ["invite", { children: [["token", SECRET, "d"], {}] }] }]));
      execFileSync("curl", ["-s", "-o", "/dev/null", `http://127.0.0.1:${port}/?_rsc=1`, "-H", `Next-Url: /invite/${SECRET}`, "-H", `Next-Router-State-Tree: ${tree}`, "-H", `Referer: https://localhost/invite/${SECRET}`]);
      execFileSync("curl", ["-s", "-o", "/dev/null", `http://127.0.0.1:${port}/invite/${SECRET}?token=${SECRET}`]);
      const log = execFileSync("docker", ["logs", name], { stdio: ["ignore", "pipe", "pipe"] }).toString();
      const entries = log.split("\n").filter((l) => l.includes('"request"'));
      assert.ok(entries.length >= 2, log);
      assert.ok(!log.includes(SECRET) && !log.includes(encodeURIComponent(SECRET)), "token còn trong log truy cập");
    } finally {
      spawnSync("docker", ["rm", "-f", name]);
    }
  });
}
