#!/usr/bin/env python3
"""Proxy đổi tên máy: http://ep.localhost:5800 → http://localhost:580 (stack EduPilot), chỉ cho PoC persona.

Vì sao cần: auth_sessions.load_for của eval-insights dò tệp phiên theo «host bỏ dần nhãn đầu» với range(len(parts) - 1),
nên host một nhãn như `localhost` không bao giờ khớp → persona luôn vào trang đăng nhập. `ep.localhost` có hai nhãn,
Chrome tự trỏ *.localhost về 127.0.0.1 và coi là ngữ cảnh an toàn (cookie Secure vẫn chạy). Caddy của EduPilot chỉ
phục vụ site `localhost` trên HTTP, nên proxy đặt lại Host / Origin và sửa Location khi trả về. Chỉ thư viện chuẩn,
truyền thân theo luồng (SSE vẫn chạy).
  python3 host_proxy.py   (PROXY_PORT=5800, UPSTREAM=localhost:580)
"""
import http.client, os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT = int(os.environ.get("PROXY_PORT", "5800"))
UP = os.environ.get("UPSTREAM", "localhost:580")
PUB = f"ep.localhost:{PORT}"
HOP = {"connection", "keep-alive", "transfer-encoding", "upgrade", "proxy-connection", "te", "trailer"}


class H(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.0"  # đóng kết nối sau mỗi phản hồi: thân truyền thẳng, không cần chunk lại

    def log_message(self, *a):
        pass

    def _go(self):
        body = self.rfile.read(int(self.headers.get("Content-Length") or 0)) or None
        h = {k: v.replace(PUB, UP) for k, v in self.headers.items() if k.lower() not in HOP}
        h["Host"] = UP
        c = http.client.HTTPConnection(UP, timeout=300)
        c.request(self.command, self.path, body, h)
        r = c.getresponse()
        self.send_response(r.status, r.reason)
        for k, v in r.getheaders():
            if k.lower() in HOP or k.lower() == "content-length" and r.status == 200 and "event-stream" in (r.getheader("Content-Type") or ""):
                continue
            self.send_header(k, v.replace(UP, PUB) if k.lower() == "location" else v)
        self.end_headers()
        while chunk := r.read1(65536) if hasattr(r, "read1") else r.read(65536):
            self.wfile.write(chunk)
            self.wfile.flush()
        c.close()

    do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = do_HEAD = do_OPTIONS = _go


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", PORT), H).serve_forever()
