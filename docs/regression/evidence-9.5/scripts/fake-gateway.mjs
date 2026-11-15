// QC2 r95 — máy chủ giả cho TC-N08-16/17/18: đăng nhập luôn 200, mọi lời gọi khác trả theo MODE.
//   MODE=429 (Retry-After: RA, mặc định 120) | 401 | drop (đóng kết nối). In mỗi yêu cầu ra stdout: `<giây> <METHOD> <path> → <mã>`.
import http from "node:http";
const MODE = process.env.MODE || "429", RA = process.env.RA || "120", PORT = Number(process.env.PORT || 18099);
const t0 = Date.now();
http.createServer((req, res) => {
  const log = (c) => console.log(`${((Date.now() - t0) / 1000).toFixed(1)} ${req.method} ${req.url} → ${c}`);
  req.resume();
  if (req.url.endsWith("/auth/login")) { log(200); res.writeHead(200, { "content-type": "application/json" }); return res.end(JSON.stringify({ access_token: "fake" })); }
  if (MODE === "drop") { log("drop"); return req.socket.destroy(); }
  const code = MODE === "401" ? 401 : 429;
  log(code);
  res.writeHead(code, { "content-type": "application/json", ...(code === 429 ? { "Retry-After": RA } : {}) });
  res.end(JSON.stringify({ code: code === 401 ? "UNAUTHENTICATED" : "RATE_LIMITED", message: "giả" }));
}).listen(PORT, "127.0.0.1", () => console.log("fake-gateway", MODE, PORT));
