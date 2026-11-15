# s8r-stub: nhà cung cấp OpenAI-compatible giả cho trích công thức (US-P6-03). usage: python3 s8r-stub.py 9761
# trạng thái đọc từ /tmp/qc2-s8r/stub-mode.json {"mode": lop1|lop2|invalid|fail503|custom, "custom": {...}, "delay": giây}; ghi mọi yêu cầu vào /tmp/qc2-s8r/stub-req.jsonl; GET /__count
import json, sys, time, threading, os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
FX = os.path.expanduser("~/Documents/TA_Agent_qc2/docs/specs/FEAT-gradebook/fixtures")
N = {"n": 0}; L = threading.Lock()
def mode(): 
    try: return json.load(open("/tmp/qc2-s8r/stub-mode.json"))
    except Exception: return {"mode": "lop1"}
class H(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    def log_message(self, *a): pass
    def _send(self, code, obj=None, raw=None):
        b = raw if raw is not None else json.dumps(obj).encode()
        self.send_response(code); self.send_header("content-type", "application/json"); self.send_header("content-length", str(len(b))); self.end_headers(); self.wfile.write(b)
    def do_GET(self):
        if self.path == "/__count": return self._send(200, {"n": N["n"]})
        if self.path.endswith("/models"): return self._send(200, {"object": "list", "data": [{"id": "m-ext", "object": "model"}]})
        self._send(404, {"error": "nf"})
    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("content-length") or 0))
        if self.path == "/__reset": N["n"] = 0; return self._send(200, {"ok": 1})
        try: j = json.loads(body)
        except Exception: j = {}
        with L: N["n"] += 1
        m = mode()
        with open("/tmp/qc2-s8r/stub-req.jsonl", "a") as f: f.write(json.dumps({"t": time.time(), "mode": m.get("mode"), "req": j}, ensure_ascii=False) + "\n")
        if m.get("delay"): time.sleep(float(m["delay"]))
        k = m.get("mode")
        if k == "fail503": return self._send(503, {"error": {"message": "down"}})
        if k == "invalid": content = "đây không phải JSON {"
        elif k == "custom": content = json.dumps(m["custom"], ensure_ascii=False)
        else: content = open(f"{FX}/extraction-{k}.json", encoding="utf-8").read()
        return self._send(200, {"id": "c", "object": "chat.completion", "created": 1, "model": j.get("model", "m-ext"), "choices": [{"index": 0, "message": {"role": "assistant", "content": content}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150}})
ThreadingHTTPServer(("0.0.0.0", int(sys.argv[1])), H).serve_forever()
