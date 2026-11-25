#!/usr/bin/env python3
"""Máy chủ giả decision (QC 10.5): POST /decisions. Chế độ đọc từ /tmp/qc105/stub_mode mỗi lần gọi: yes|no|mid|error|slow|bad|safety.
Ghi /tmp/qc105/stub.jsonl: {t, key_sha (sha256 của header x-goog-api-key, KHÔNG ghi khoá), path, body}. Cổng 9807 (lắng nghe 0.0.0.0 để container tới bằng host.docker.internal)."""
import json, hashlib, time, sys, os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
LOG = "/tmp/qc105/stub.jsonl"; MODE = "/tmp/qc105/stub_mode"
class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_POST(self):
        n = int(self.headers.get("content-length") or 0); raw = self.rfile.read(n)
        try: body = json.loads(raw)
        except Exception: body = {"raw": raw[:200].decode("utf8", "replace")}
        key = self.headers.get("x-goog-api-key", "")
        open(LOG, "a").write(json.dumps({"t": time.time(), "path": self.path, "key_sha": hashlib.sha256(key.encode()).hexdigest()[:16], "has_key": bool(key), "body": body}, ensure_ascii=False) + "\n")
        mode = open(MODE).read().strip() if os.path.exists(MODE) else "mid"
        if mode == "slow": time.sleep(5)
        if mode == "error": return self._j(503, {"error": {"code": "unavailable"}})
        if mode == "safety": return self._j(400, {"error": {"code": "safety", "message": "blocked"}})
        if mode == "bad": return self._j(200, {"answers": {}})
        p = {"yes": 0.99, "no": 0.01, "mid": 0.25, "slow": 0.25}.get(mode, 0.25)
        ans = {}
        for qid, q in (body.get("questions") or {}).items():
            if q.get("type") == "noul": ans[qid] = {"type": "noul", "noul": p, "confidence": max(p, 1 - p)}
            else:
                lab = sorted((q.get("criteria") or {"A": ""}))[0]; ans[qid] = {"type": "choice", "choice": lab, "confidence": 0.5, "probabilities": {lab: 0.5}}
        self._j(200, {"answers": ans, "usage": {"total_input_tokens": max(1, len(json.dumps(body)) // 4)}})
    def _j(self, c, o):
        b = json.dumps(o).encode(); self.send_response(c); self.send_header("content-type", "application/json"); self.send_header("content-length", str(len(b))); self.end_headers(); self.wfile.write(b)
ThreadingHTTPServer(("0.0.0.0", int(sys.argv[1]) if len(sys.argv) > 1 else 9807), H).serve_forever()
