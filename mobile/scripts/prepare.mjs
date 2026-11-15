// Sinh `www/` (webDir bắt buộc có) từ APP_SERVER_URL: một trang trống và trang lỗi «Thử lại» trỏ về ứng dụng.
import { mkdirSync, writeFileSync } from "node:fs";

const url = process.env.APP_SERVER_URL;
if (!url) {
  console.error("Đặt APP_SERVER_URL trước (vd. APP_SERVER_URL=https://edupilot.example.com).");
  process.exit(1);
}
const u = new URL(url);
if (u.protocol !== "https:" && !["localhost", "127.0.0.1", "10.0.2.2"].includes(u.hostname)) {
  console.error("APP_SERVER_URL phải là HTTPS (getUserMedia và WebCrypto cần ngữ cảnh an toàn).");
  process.exit(1);
}
mkdirSync("www", { recursive: true });
writeFileSync("www/index.html", `<!doctype html><meta charset="utf-8"><title>EduPilot</title>`);
writeFileSync(
  "www/error.html",
  `<!doctype html><html lang="vi"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>EduPilot</title>
<style>body{font:16px system-ui;margin:0;min-height:100vh;display:grid;place-content:center;gap:16px;justify-items:center;color:#1a1a1a}button{min-height:44px;padding:0 24px;font:inherit;border:0;border-radius:8px;background:#CE1628;color:#fff}</style>
<p>Không kết nối được máy chủ.</p><button onclick="location.href=${JSON.stringify(url)}">Thử lại</button></html>`,
);
console.log(`www/ sẵn sàng cho ${u.origin}`);
