# MÁY CHỦ GIẢ HTTP cho provider decision (gemini-decision-1) — chỉ QC, chạy trên host, KHÔNG ra Gemini thật.
# Giao thức đúng internal/llm/provider/decision.go: POST {base}/decisions, header x-goog-api-key, thân {model,input,questions}.
# Điều khiển bằng tệp ~/.config/edupilot-team/qc3-work/stub_dec_mode.json: {"mode": ok|429|500|safety|auth|hang|invalid|notfound, "noul": 0.25, "delay_ms": 0, "choice": null}
import http.server, json, os, sys, time
W = "/Users/kuro/.config/edupilot-team/qc3-work/"; PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 9770
def mode():
    try: return json.load(open(W + "stub_dec_mode.json"))
    except Exception: return {"mode": "ok"}
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self): self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers(); self.wfile.write(b'{"ok":true}')
    def do_POST(self):
        n = int(self.headers.get("content-length") or 0); raw = self.rfile.read(n) if n else b""; m = mode()
        try: body = json.loads(raw)
        except Exception: body = {"_raw": raw[:200].decode("utf8", "replace")}
        key = self.headers.get("x-goog-api-key") or ""
        with open(W + "stub_dec_req.jsonl", "a") as f: f.write(json.dumps({"t": time.time(), "path": self.path, "key_present": bool(key), "key": key if key.startswith(("test-key", "qc-db-key")) else ("<khác>" if key else ""), "key_is_test": key == "test-key", "auth_hdr": "Authorization" in self.headers, "key_in_url": "key=" in self.path, "mode": m.get("mode"), "body": body}, ensure_ascii=False) + "\n")
        if m.get("delay_ms"): time.sleep(m["delay_ms"] / 1000)
        md = m.get("mode", "ok")
        def err(code, obj, extra=None):
            b = json.dumps(obj).encode(); self.send_response(code)
            for k, v in (extra or {}).items(): self.send_header(k, v)
            self.send_header("content-type", "application/json"); self.send_header("content-length", str(len(b))); self.end_headers(); self.wfile.write(b)
        if self.path.rstrip("/").split("?")[0].endswith("/decisions") is False: return err(404, {"error": {"code": 404, "message": "no route"}})
        if md == "429": return err(429, {"error": {"code": 429, "message": "quota"}}, {"Retry-After": "2"})
        if md == "500": return err(500, {"error": {"code": 500, "message": "boom"}})
        if md == "safety": return err(400, {"error": {"code": "safety", "status": "INVALID_ARGUMENT", "message": "blocked"}})
        if md == "auth": return err(403, {"error": {"code": 403, "message": "key"}})
        if md == "notfound": return err(404, {"error": {"code": 404, "message": "model"}})
        if md == "hang": time.sleep(30); return err(200, {})
        if md == "invalid": b = b"khong phai json"; self.send_response(200); self.send_header("content-length", str(len(b))); self.end_headers(); self.wfile.write(b); return
        ans = {}
        for qid, q in (body.get("questions") or {}).items():
            if q.get("type") == "noul": ans[qid] = {"type": "noul", "noul": float(m.get("noul", 0.25))}
            else:
                labs = list((q.get("criteria") or {"A": 1, "B": 1}).keys()); lab = m.get("choice") if m.get("choice") in labs else labs[0]
                cf = float(m.get("conf", 0.9)); ans[qid] = {"type": "choice", "choice": lab, "confidence": cf, "probabilities": {l: (cf if l == lab else (1 - cf) / max(1, len(labs) - 1)) for l in labs}}
        out = json.dumps({"answers": ans, "usage": {"total_input_tokens": len(str(body.get("input", ""))) // 4 + 10}}).encode()
        self.send_response(200); self.send_header("content-type", "application/json"); self.send_header("content-length", str(len(out))); self.end_headers(); self.wfile.write(out)
http.server.ThreadingHTTPServer(("0.0.0.0", PORT), H).serve_forever()
