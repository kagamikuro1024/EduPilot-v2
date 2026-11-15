#!/usr/bin/env python3
"""Shim cục bộ để chạy eval-insights (nhánh uiux-crowd) bằng 9router mà không cần sửa mã của họ, không cần khoá trả phí.

eval-insights gọi «V-LLM» qua OpenAI chat completions + json_schema strict, lấy bearer từ một endpoint OAuth.
9router (ag/*) bỏ qua response_format, nên shim:
  POST /oauth/token            → trả bearer giả (eval-insights chỉ cần chuỗi có dạng JWT)
  POST /v1/chat/completions    → chép JSON schema vào lời nhắc hệ thống, bỏ khoá 9router không hiểu,
                                 ép stream:false, gửi 9router bằng khoá đọc từ ~/.9router (không in, không ghi tệp),
                                 rồi cắt {…} ra khỏi câu trả lời để phía gọi json.loads được.
Chạy: python3 router_shim.py  (cổng SHIM_PORT, mặc định 20129; model SHIM_MODEL ghi đè model phía gọi gửi)
Chỉ thư viện chuẩn. Ghi mỗi lời gọi (giây, token, model) vào shim-usage.jsonl cạnh tệp để tính chi phí.
"""
import base64, json, os, sqlite3, sys, time, urllib.error, urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ROUTER = os.environ.get("ROUTER_BASE", "http://127.0.0.1:20128/v1")
PORT = int(os.environ.get("SHIM_PORT", "20129"))
MODEL = os.environ.get("SHIM_MODEL", "")
USAGE = Path(__file__).with_name("shim-usage.jsonl")
_key = None


def key():
    global _key
    if not _key:
        db = sqlite3.connect(os.path.expanduser("~/.9router/db/data.sqlite"))
        _key = db.execute("select key from apiKeys where isActive=1 limit 1").fetchone()[0]
    return _key


def fake_jwt():
    seg = lambda o: base64.urlsafe_b64encode(json.dumps(o).encode()).decode().rstrip("=")
    return f"{seg({'alg': 'none'})}.{seg({'exp': int(time.time()) + 86400})}.x"


def extract(text):
    s = text.strip()
    if s.startswith("{") and s.endswith("}"):
        return s
    i, j = s.find("{"), s.rfind("}")
    return s[i:j + 1] if 0 <= i < j else s


class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def reply(self, code, obj):
        raw = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self):
        if self.path.rstrip("/").endswith("/models"):
            return self.reply(200, {"object": "list", "data": [{"id": MODEL or "shim", "object": "model"}]})
        self.reply(404, {"error": "not found"})

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length") or 0)) or b"{}")
        if self.path.endswith("/oauth/token"):
            return self.reply(200, {"access_token": fake_jwt(), "token_type": "bearer"})
        if not self.path.endswith("/chat/completions"):
            return self.reply(404, {"error": "not found"})
        rf = body.pop("response_format", None) or {}
        body.pop("chat_template_kwargs", None)
        body["stream"] = False
        if MODEL:
            body["model"] = MODEL
        schema = (rf.get("json_schema") or {}).get("schema")
        if schema or rf.get("type") == "json_object":
            note = "\n\nĐẦU RA: chỉ một đối tượng JSON hợp lệ, không markdown, không chữ nào khác."
            if schema:
                note += " Theo đúng JSON schema sau (đủ mọi khoá required, đúng enum):\n" + json.dumps(schema, ensure_ascii=False)
            msgs = body.get("messages") or []
            if msgs and msgs[0].get("role") == "system" and isinstance(msgs[0].get("content"), str):
                msgs[0]["content"] += note
            else:
                msgs.insert(0, {"role": "system", "content": note.strip()})
        t0 = time.time()
        req = urllib.request.Request(ROUTER + "/chat/completions", json.dumps(body).encode(),
                                     {"Authorization": f"Bearer {key()}", "Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=300) as r:
                data = json.loads(r.read())
        except urllib.error.HTTPError as e:
            return self.reply(e.code, {"error": e.read().decode(errors="replace")[:300]})
        except Exception as e:  # mạng / timeout → 502 để phía gọi retry
            return self.reply(502, {"error": type(e).__name__})
        msg = ((data.get("choices") or [{}])[0].get("message") or {})
        if (schema or rf) and isinstance(msg.get("content"), str):
            msg["content"] = extract(msg["content"])
        u = data.get("usage") or {}
        with USAGE.open("a") as f:
            f.write(json.dumps({"t": round(time.time() - t0, 2), "model": data.get("model"), "in": u.get("prompt_tokens"),
                                "out": u.get("completion_tokens"), "schema": bool(schema)}) + "\n")
        self.reply(200, data)


if __name__ == "__main__":
    print(f"shim :{PORT} → {ROUTER}", file=sys.stderr)
    ThreadingHTTPServer(("127.0.0.1", PORT), H).serve_forever()
