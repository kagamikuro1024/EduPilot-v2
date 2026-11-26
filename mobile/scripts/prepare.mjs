// Sinh `www/` (webDir bắt buộc có) từ APP_SERVER_URL: một trang trống và trang lỗi «Thử lại» trỏ về ứng dụng.
import { mkdirSync, writeFileSync } from "node:fs";

const url = process.env.APP_SERVER_URL;
if (!url) {
  console.error("Đặt APP_SERVER_URL trước (vd. APP_SERVER_URL=https://edupilot.example.com).");
  process.exit(1);
}
let u;
try {
  u = new URL(url);
} catch {
  console.error(`APP_SERVER_URL không phải URL hợp lệ: ${url}`);
  process.exit(1);
}
// http:// chỉ để chạy với server LAN (chủ dự án 27/11, P-11-153): capacitor.config.js bật cleartext đúng khi URL là http. Trang http không phải secure context ngoài localhost ⇒ WebCrypto / getUserMedia có thể không dùng được.
if (u.protocol !== "https:" && u.protocol !== "http:") {
  console.error(`APP_SERVER_URL phải là http(s) (nhận ${u.protocol}//${u.host}).`);
  process.exit(1);
}
mkdirSync("www", { recursive: true });
writeFileSync("www/index.html", `<!doctype html><meta charset="utf-8"><title>EduPilot</title>`);
// Trang lỗi WebView: nền / chữ / nút theo token sáng của web (D59: --ep-canvas #FFFFFF, --ep-ink #231F20, --ep-red #CE1628); color-scheme chỉ-sáng để WebView ép tối không đổi màu.
// «Thử lại» là liên kết (href đã escape), không phải onclick: JSON.stringify trong thuộc tính "…" làm gãy thuộc tính và nút không chạy.
const href = u.href.replaceAll("&", "&amp;");
writeFileSync(
  "www/error.html",
  `<!doctype html><html lang="vi"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><title>EduPilot</title>
<style>html{color-scheme:light;background:#FFFFFF}body{font:16px system-ui;margin:0;min-height:100vh;display:grid;place-content:center;gap:16px;justify-items:center;background:#FFFFFF;color:#231F20}a{display:grid;place-items:center;min-height:44px;padding:0 24px;border-radius:8px;background:#CE1628;color:#FFFFFF;text-decoration:none}</style>
<p>Không kết nối được máy chủ.</p><a href="${href}">Thử lại</a></html>`,
);
console.log(`www/ sẵn sàng cho ${u.origin}`);
