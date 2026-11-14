# Máy chủ giả tương thích OpenAI để QC điều khiển kết quả của lời gọi Structured (không 9router, không Gemini).
# Chế độ đọc từ /tmp/stub_mode mỗi yêu cầu: allow | block | error | slow | invalid. Ghi yêu cầu vào /tmp/stub_req.jsonl.
import json, time, sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

def fill(schema, val):
    t = schema.get("type")
    if isinstance(t, list): t = [x for x in t if x != "null"][0]
    if t == "object" or "properties" in schema:
        return {k: fill(v, val) for k, v in schema.get("properties", {}).items()}
    if t == "boolean": return val
    if t in ("integer", "number"): return 0
    if t == "array": return []
    if "enum" in schema: return schema["enum"][0]
    return ""

class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers()
        self.wfile.write(json.dumps({"data": [{"id": "stub-chat"}]}).encode())
    def do_POST(self):
        n = int(self.headers.get("content-length") or 0); body = json.loads(self.rfile.read(n) or b"{}")
        mode = (open("/tmp/stub_mode").read().strip() if __import__("os").path.exists("/tmp/stub_mode") else "allow")
        with open("/tmp/stub_req.jsonl", "a") as f: f.write(json.dumps({"t": time.time(), "mode": mode, "body": body}, ensure_ascii=False) + "\n")
        if mode == "error":
            self.send_response(500); self.end_headers(); self.wfile.write(b'{"error":{"message":"stub error"}}'); return
        if mode == "slow": time.sleep(12)
        schema = (((body.get("response_format") or {}).get("json_schema") or {}).get("schema")) or {}
        if not schema:  # schema nằm trong lời nhắc hệ thống (chế độ json_object)
            import re
            for m in body.get("messages", []):
                x = re.search(r"JSON Schema sau[^\n]*\n(\{.*\})", m.get("content", ""))
                if x:
                    try: schema = json.loads(x.group(1)); break
                    except Exception: pass
        if mode == "invalid": content = "khong phai json {{"
        else: content = json.dumps(fill(schema, mode == "block"), ensure_ascii=False)
        out = {"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "stub-chat",
               "choices": [{"index": 0, "message": {"role": "assistant", "content": content}, "finish_reason": "stop"}],
               "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}}
        self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers(); self.wfile.write(json.dumps(out).encode())

if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", int(sys.argv[1]) if len(sys.argv) > 1 else 8777), H).serve_forever()
