// node --test mobile/scripts/prepare.test.mjs — prepare.mjs nhận APP_SERVER_URL http(s) (D-AS-04; http mở theo P-11-153).
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, mkdtempSync, readFileSync } from "node:fs";
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
  const page = path.join(cwd, "www", "error.html");
  return { ...r, wrote: existsSync(page), html: existsSync(page) ? readFileSync(page, "utf8") : "" };
};

for (const bad of ["ftp://edupilot.example.com", "file:///etc/passwd", "không-phải-url"]) {
  test(`từ chối ${bad}`, () => {
    const r = run(bad);
    assert.equal(r.status, 1);
    assert.match(r.stderr, /http\(s\)|URL hợp lệ/);
    assert.equal(r.wrote, false);
  });
}
test("http LAN hợp lệ → sinh www/, nút «Thử lại» trỏ đúng URL", () => {
  const r = run("http://192.168.41.4");
  assert.equal(r.status, 0, r.stderr);
  assert.match(r.html, /<a href="http:\/\/192\.168\.41\.4\/">Thử lại<\/a>/);
});
test("thiếu APP_SERVER_URL → lỗi", () => assert.equal(run(undefined).status, 1));
test("https hợp lệ → sinh www/", () => {
  const r = run("https://demo.trycloudflare.com");
  assert.equal(r.status, 0, r.stderr);
  assert.equal(r.wrote, true);
});

// error.html là trang lỗi WebView: nền và chữ đặt rõ theo token sáng (D59: --ep-canvas, --ep-ink, --ep-red) và khai chỉ-sáng để WebView ép tối không đổi màu.
test("error.html: nền + chữ theo token sáng, khai color-scheme light, chữ tiếng Việt", () => {
  const { html } = run("https://demo.trycloudflare.com");
  assert.match(html, /<meta name="color-scheme" content="light">/);
  assert.match(html, /background:#FFFFFF/i);
  assert.match(html, /color:#231F20/i);
  assert.match(html, /Không kết nối được máy chủ\./);
  assert.match(html, /<a href="https:\/\/demo\.trycloudflare\.com\/">Thử lại<\/a>/); // nút chạy được: href đúng, không onclick lồng dấu nháy
});
