#!/usr/bin/env python3
"""Máy chủ giả OpenAI-compatible cho QC s9grading (US-P7-06/07/08). Ghi MỌI yêu cầu vào /tmp/q9s/req.jsonl (body nguyên văn, kể cả prompt đã che).
Hành vi chọn theo (a) tên model: g-main (mặc định) | g-timeout (ngủ 31 s, quá LLM_REQUEST_TIMEOUT 30 s) ; (b) dấu [[S9G k=v ...]] chèn trong bài làm:
  fail=N:CODE[:key]  N lần gọi đầu theo key trả HTTP CODE (404 = lỗi không-outage; 503 = outage)   slow=SEC  ngủ SEC giây rồi trả OK
  invalid            trả chuỗi không phải JSON                 echo=N  nhận xét tổng chép N từ liên tiếp của đáp án chuẩn (trong system)
  score=a,b,c,d      điểm từng tiêu chí (thứ tự c1…)              total=X  "total" LLM tự khai    quote=1  evidence_quotes = câu đầu bài (nguyên văn, không che)
  conf=0.9           confidence       flags=OFF_TOPIC     badpass2  lượt 2 trả JSON sai schema
(c) bài khớp mẫu S1–S7 của BA (expected_grading.json) → phát lại llm_pass1 / llm_pass2 (lượt 2 nhận ra qua thứ tự tiêu chí đảo)."""
import json, os, re, sys, time, threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
FX = "/Users/kuro/Documents/TA_Agent_qc9sub/docs/specs/FEAT-assignments/fixtures"
SAMPLES = []
for s in json.load(open(f"{FX}/expected_grading.json")):
    t = open(f"{FX}/{s['file']}", encoding="utf-8").read()
    SAMPLES.append((s["id"], t.strip()[60:130], s))
os.makedirs("/tmp/q9s", exist_ok=True); LOCK = threading.Lock(); CNT = {}
def marks(text):
    m = re.search(r"\[\[S9G ([^\]]*)\]\]", text); d = {}
    for kv in (m.group(1).split() if m else []):
        k, _, v = kv.partition("="); d[k] = v
    return d
def ids_of(system):
    return re.findall(r"^- (c\d+) — ", system, re.M)
class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def send(self, code, obj, raw=False):
        b = obj if raw else json.dumps(obj).encode(); self.send_response(code); self.send_header("content-type", "application/json"); self.end_headers(); self.wfile.write(b)
    def do_GET(self): self.send(200, {"data": [{"id": "g-main"}, {"id": "g-timeout"}]})
    def do_POST(self):
        n = int(self.headers.get("content-length") or 0); body = json.loads(self.rfile.read(n) or b"{}"); now = time.time()
        with LOCK, open("/tmp/q9s/req.jsonl", "a") as f: f.write(json.dumps({"t": now, "path": self.path, "model": body.get("model"), "body": body}, ensure_ascii=False) + "\n")
        mode = open("/tmp/q9s/mode").read().strip() if os.path.exists("/tmp/q9s/mode") else "ok"
        if body.get("model") == "g-timeout": time.sleep(31); return self.send(200, {"choices": []})
        if mode.startswith("http"): return self.send(int(mode[4:]), {"error": {"message": "s9g stub " + mode}})
        msgs = body.get("messages", []); system = " ".join(m["content"] for m in msgs if m["role"] == "system"); user = " ".join(m["content"] for m in msgs if m["role"] == "user")
        if "RUBRIC (thứ tự chấm)" not in system:  # tác vụ khác: trả JSON rỗng theo schema nếu có
            return self.send(200, self.wrap(json.dumps({})))
        m = re.search(r"<<<(BAI_LAM_[0-9A-F]+)\n(.*)\n\1>>>", user, re.S); text = m.group(2) if m else ""
        mk = marks(text); ids = ids_of(system)
        second = bool(re.search(r"Thứ tự tiêu chí trong kết quả: ([^.\n]*)", user)) and re.search(r"Thứ tự tiêu chí trong kết quả: ([^.\n]*)", user).group(1).split(", ")[0] != "c1"
        if "fail" in mk:
            pr = mk["fail"].split(":"); N, code = pr[0], (pr[1] if len(pr) > 1 else "500"); key = pr[2] if len(pr) > 2 else text[:40]
            with LOCK: CNT[key] = CNT.get(key, 0) + 1; c = CNT[key]
            if c <= int(N): return self.send(int(code), {"error": {"message": "s9g fail"}})
        if "slow" in mk: time.sleep(float(mk["slow"]))
        if "invalid" in mk or ("badpass2" in mk and second): return self.send(200, self.wrap("khong phai json {{"))
        for sid, key, s in SAMPLES:
            if key and key in text:
                out = s.get("llm_pass2" if second else "llm_pass1") or s.get("llm_pass1")
                if out is None: continue
                return self.send(200, self.wrap(json.dumps(out, ensure_ascii=False)))
        sc = [x for x in mk.get("score", "").split(",") if x]
        crit = []
        for i, c in enumerate(ids):
            q = [text.strip().split(".")[0].strip()] if "quote" in mk and text.strip() else []
            crit.append({"id": c, "score": sc[int(c[1:]) - 1] if sc else "1.00", "evidence_quotes": q, "comment": f"Nhận xét giả cho {c}."})
        fb = "Nhận xét tổng quát giả của stub."
        if "echo" in mk:
            ref = system.split("ĐÁP ÁN CHUẨN")[-1]; ref = ref.split("\n", 1)[1] if "\n" in ref else ref
            words = re.findall(r"\w+", ref, re.U); k = int(mk["echo"])
            start = next((i for i in range(len(words) - k) if not words[i].startswith("CANARY")), 0); fb = "Nhận xét: " + " ".join(words[start:start + k]) + " là điểm chính."
        if "echofold" in mk:
            import unicodedata
            ref = system.split("ĐÁP ÁN CHUẨN")[-1]; ref = ref.split("\n", 1)[1] if "\n" in ref else ref
            words = re.findall(r"\w+", ref, re.U); k = int(mk["echofold"])
            start = next((i for i in range(len(words) - k) if not words[i].startswith("CANARY")), 0)
            fb = "Nhận xét: " + unicodedata.normalize("NFD", " ".join(words[start:start + k])).encode("ascii", "ignore").decode().upper() + " là điểm chính."
        tot = mk.get("total") or "%.2f" % sum(float(c["score"]) for c in crit)
        out = {"criteria": crit, "overall_feedback": fb, "confidence": float(mk.get("conf", "0.9")), "flags": [x for x in mk.get("flags", "").split(",") if x], "total": tot}
        self.send(200, self.wrap(json.dumps(out, ensure_ascii=False)))
    def wrap(self, content):
        return {"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "g-main", "choices": [{"index": 0, "message": {"role": "assistant", "content": content}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150}}
if __name__ == "__main__": ThreadingHTTPServer(("0.0.0.0", int(sys.argv[1]) if len(sys.argv) > 1 else 9751), H).serve_forever()
