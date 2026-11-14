# Máy chủ giả tương thích OpenAI cho QC (không 9router, không Gemini). Chế độ đọc từ /tmp/stub_mode mỗi yêu cầu.
#  Chung: allow | block | error | slow | invalid (lấp schema bằng giá trị mặc định / yes-no; xem sprint 10.5)
#  Riêng cho tác vụ `reflect` (lời nhắc hệ thống chứa «rút ra lưu ý chấm»): rf_ok | rf_bad_criterion | rf_bad_case | rf_wrong_direction |
#   rf_total_keyword | rf_pii_text | rf_dup_text | rf_low_support | rf_many | rf_empty | rf_fake_support  (cùng error / slow / invalid)
# Mọi yêu cầu ghi vào /tmp/stub_req.jsonl; đầu vào reflect gần nhất ghi vào /tmp/stub_reflect_in.json.
import json, os, re, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

def fill(schema, val):
    t = schema.get("type")
    if isinstance(t, list): t = [x for x in t if x != "null"][0]
    if t == "object" or "properties" in schema:
        return {k: fill(v, val) for k, v in schema.get("properties", {}).items()}
    if t == "boolean": return val
    if t in ("integer", "number"): return 0
    if t == "array": return []
    if "enum" in schema: return ("yes" if val and "yes" in schema["enum"] else "no" if "no" in schema["enum"] else schema["enum"][0])
    return ""

TEXTS = {
    "STRICTER": "Chấm chặt hơn khi bài chỉ nêu khái niệm chung mà chưa có ví dụ hay số liệu cụ thể cho {c}.",
    "LENIENT": "Chấp nhận cách diễn đạt khác của sinh viên cho {c} nếu ý chính đúng và có lập luận rõ ràng.",
    "CLARIFY": "Hiểu {c} theo nghĩa phân tích nguyên nhân gốc rễ, không chỉ mô tả hiện tượng bên ngoài.",
}
MANY = ["Ưu tiên bằng chứng định lượng khi đánh giá {c}.", "Cần nêu rõ tác nhân và phạm vi ảnh hưởng ở {c}.", "Không cho điểm tối đa {c} nếu thiếu đối chiếu thực tế.",
        "Đoạn lập luận ngắn gọn nhưng đúng trọng tâm vẫn đạt mức khá ở {c}.", "Phân biệt rõ giả định với kết luận trong {c}.", "Liên hệ vụ việc thật làm tăng giá trị bài ở {c}."]

def direction(c):
    ai, tc = float(c["ai_score"]), float(c["teacher_score"])
    return "STRICTER" if ai > tc else "LENIENT" if ai < tc else "CLARIFY"

def reflect_items(mode, data):
    cases = data.get("cases", []); existing = data.get("existing", [])
    groups = {}
    for c in cases: groups.setdefault((c["criterion_id"], direction(c)), []).append(c["case_id"])
    good = [(k, v) for k, v in groups.items() if len(v) >= 2 and k[1] != "CLARIFY"]
    items = []
    if mode == "rf_empty": return []
    if mode == "rf_ok":
        for (cid, d), ids in good: items.append({"criterion_id": cid, "text": TEXTS[d].format(c=cid), "direction": d, "case_ids": ids[:5]})
    elif mode == "rf_many":
        for (cid, d), ids in good:
            for i, t in enumerate(MANY): items.append({"criterion_id": cid, "text": t.format(c=cid) + f" Mẫu {i}.", "direction": d, "case_ids": ids[:3]})
        items = items[:12]
    elif mode == "rf_bad_criterion":
        (cid, d), ids = good[0]; items.append({"criterion_id": "c99", "text": "Tiêu chí không có trong rubric cần chú ý thêm ví dụ.", "direction": d, "case_ids": ids[:3]})
    elif mode == "rf_bad_case":
        (cid, d), ids = good[0]; items.append({"criterion_id": cid, "text": TEXTS[d].format(c=cid), "direction": d, "case_ids": ids[:3] + ["khong-co-trong-dau-vao"]})
    elif mode == "rf_wrong_direction":
        (cid, d), ids = good[0]; items.append({"criterion_id": cid, "text": TEXTS[d].format(c=cid), "direction": "LENIENT" if d == "STRICTER" else "STRICTER", "case_ids": ids[:3]})
    elif mode == "rf_low_support":
        (cid, d), ids = good[0]
        other = [c["case_id"] for c in cases if c["criterion_id"] == cid and direction(c) != d][:1]
        items.append({"criterion_id": cid, "text": TEXTS[d].format(c=cid), "direction": d, "case_ids": ids[:1] + other})
    elif mode == "rf_fake_support":
        (cid, d), ids = good[0]; items.append({"criterion_id": cid, "text": TEXTS[d].format(c=cid), "direction": d, "case_ids": ids[:1] + ["khong-co"], "support": 99})
    elif mode == "rf_total_keyword":
        (cid, d), ids = good[0]
        for t in ("Tổng điểm của bài nên được tính chặt hơn.", "Trừ muộn theo quy định nhà trường cho mọi bài.", "Làm tròn lên khi gần ngưỡng.", "Phạt nặng bài sao chép.", "TONG diem phai khop rubric.", "Cần LAM TRON theo 0,5."):
            items.append({"criterion_id": cid, "text": t, "direction": d, "case_ids": ids[:3]})
    elif mode == "rf_pii_text":
        pii = json.load(open("/tmp/stub_pii.json")) if os.path.exists("/tmp/stub_pii.json") else {}
        (cid, d), ids = good[0]
        for t in (f"Bài của {pii.get('name', 'Nguyễn Văn Kha')} thường thiếu ví dụ cụ thể.", f"Sinh viên mã {pii.get('mssv', '20229001')} hay viết quá ngắn.", f"Liên hệ {pii.get('email', 'kha@edupilot.local')} để hỏi lại ý.", "Gọi 0912345678 nếu cần làm rõ."):
            items.append({"criterion_id": cid, "text": t, "direction": d, "case_ids": ids[:3]})
    elif mode == "rf_dup_text":
        (cid, d), ids = good[0]
        base = existing[0]["text"] if existing else TEXTS[d].format(c=cid)
        items.append({"criterion_id": existing[0]["criterion_id"] if existing else cid, "text": base, "direction": d, "case_ids": ids[:3]})
    return items

class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers()
        self.wfile.write(json.dumps({"data": [{"id": "stub-chat"}]}).encode())
    def do_POST(self):
        n = int(self.headers.get("content-length") or 0); body = json.loads(self.rfile.read(n) or b"{}")
        mode = (open("/tmp/stub_mode").read().strip() if os.path.exists("/tmp/stub_mode") else "allow")
        with open("/tmp/stub_req.jsonl", "a") as f: f.write(json.dumps({"t": time.time(), "mode": mode, "body": body}, ensure_ascii=False) + "\n")
        if mode == "error":
            self.send_response(500); self.end_headers(); self.wfile.write(b'{"error":{"message":"stub error"}}'); return
        if mode == "slow": time.sleep(12)
        msgs = body.get("messages", [])
        system = " ".join(m.get("content", "") for m in msgs if m.get("role") == "system")
        if mode == "invalid": content = "khong phai json {{"
        elif "rút ra lưu ý chấm" in system:
            user = " ".join(m.get("content", "") for m in msgs if m.get("role") == "user")
            x = re.search(r"<<<[^\n]*\n(\{.*\})\n[^\n]*>>>", user, re.S)
            data = json.loads(x.group(1)) if x else {}
            json.dump(data, open("/tmp/stub_reflect_in.json", "w"), ensure_ascii=False)
            content = json.dumps({"items": reflect_items(mode if mode.startswith("rf_") else "rf_ok", data)}, ensure_ascii=False)
        else:
            schema = (((body.get("response_format") or {}).get("json_schema") or {}).get("schema")) or {}
            if not schema:
                for m in msgs:
                    x = re.search(r"JSON Schema sau[^\n]*\n(\{.*\})", m.get("content", ""))
                    if x:
                        try: schema = json.loads(x.group(1)); break
                        except Exception: pass
            content = json.dumps(fill(schema, mode == "block"), ensure_ascii=False)
        out = {"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "stub-chat",
               "choices": [{"index": 0, "message": {"role": "assistant", "content": content}, "finish_reason": "stop"}],
               "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}}
        self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers(); self.wfile.write(json.dumps(out).encode())

if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", int(sys.argv[1]) if len(sys.argv) > 1 else 8777), H).serve_forever()
