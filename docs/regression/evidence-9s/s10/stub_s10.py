#!/usr/bin/env python3
# Máy chủ giả tương thích OpenAI cho QC sprint 10: phát lại bản ghi seed/demo/fake_llm.json (khớp mọi chuỗi `match` trong lời nhắc);
# chế độ đọc từ /tmp/stub10_mode mỗi yêu cầu: ok | error (500) | overloaded (429) | slow (12 s) | invalid (không phải JSON) | ungrounded
# Mọi yêu cầu ghi /tmp/stub10_req.jsonl (t, mode, tên bản ghi khớp, độ dài).  Chạy: python3 stub_s10.py 9761
import json, os, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
REC = json.load(open(os.path.expanduser("~/Documents/TA_Agent_qc106/seed/demo/fake_llm.json")))["recordings"]
def match(text):
    for r in REC:
        ms = r["match"] if isinstance(r["match"], list) else [r["match"]]
        if all(m in text for m in ms): return r
class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers(); self.wfile.write(json.dumps({"data": [{"id": "stub10"}]}).encode())
    def do_POST(self):
        n = int(self.headers.get("content-length") or 0); body = json.loads(self.rfile.read(n) or b"{}")
        mode = open("/tmp/stub10_mode").read().strip() if os.path.exists("/tmp/stub10_mode") else "ok"
        text = "\n".join(str(m.get("content", "")) for m in body.get("messages", []))
        rec = match(text)
        with open("/tmp/stub10_req.jsonl", "a") as f: f.write(json.dumps({"t": time.time(), "mode": mode, "rec": rec and rec["name"], "len": len(text), "maxpage": max([int(x) for x in __import__("re").findall(r"\[Trang (\d+)\]", text)] or [0]), "text": text[:6000]}, ensure_ascii=False) + "\n")
        if mode == "error": self.send_response(500); self.end_headers(); self.wfile.write(b'{"error":{"message":"stub error"}}'); return
        if mode == "overloaded": self.send_response(429); self.send_header("retry-after", "1"); self.end_headers(); self.wfile.write(b'{"error":{"message":"overloaded","type":"rate_limit"}}'); return
        if mode.startswith("slow"): time.sleep(int(mode[4:] or 12))
        if mode == "invalid": content = "khong phai json {{"
        elif rec: r = rec["response"]; content = r if isinstance(r, str) else json.dumps(r, ensure_ascii=False)
        elif "Bạn trích câu hỏi trắc nghiệm" in text: content = '{"questions": []}'
        else: content = "Bản ghi giả: không có bản ghi khớp lời nhắc."
        out = {"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "stub10", "choices": [{"index": 0, "message": {"role": "assistant", "content": content}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}}
        if body.get("stream"):
            self.send_response(200); self.send_header("content-type", "text/event-stream"); self.end_headers()
            ch = {"id": "x", "object": "chat.completion.chunk", "model": "stub10", "choices": [{"index": 0, "delta": {"role": "assistant", "content": content}, "finish_reason": None}]}
            self.wfile.write(("data: " + json.dumps(ch, ensure_ascii=False) + "\n\n").encode())
            fin = {"id": "x", "object": "chat.completion.chunk", "model": "stub10", "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}], "usage": out["usage"]}
            self.wfile.write(("data: " + json.dumps(fin) + "\n\ndata: [DONE]\n\n").encode()); return
        self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers(); self.wfile.write(json.dumps(out, ensure_ascii=False).encode())
if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", int(sys.argv[1]) if len(sys.argv) > 1 else 9761), H).serve_forever()
