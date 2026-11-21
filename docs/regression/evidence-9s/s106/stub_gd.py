# Máy chủ giả tương thích OpenAI cho QC (không 9router, không Gemini). Chế độ đọc từ /Users/kuro/.config/edupilot-team/qc3-work/stub_mode mỗi yêu cầu.
#  Chung: allow | block | error | slow | invalid (lấp schema bằng giá trị mặc định / yes-no; xem sprint 10.5)
#  Riêng cho tác vụ `reflect` (lời nhắc hệ thống chứa «rút ra lưu ý chấm»): rf_ok | rf_bad_criterion | rf_bad_case | rf_wrong_direction |
#   rf_total_keyword | rf_pii_text | rf_dup_text | rf_low_support | rf_many | rf_empty | rf_fake_support  (cùng error / slow / invalid)
# Mọi yêu cầu ghi vào /Users/kuro/.config/edupilot-team/qc3-work/stub_req.jsonl; đầu vào reflect gần nhất ghi vào /Users/kuro/.config/edupilot-team/qc3-work/stub_reflect_in.json.
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
        items = items[:8]  # schema strict maxItems = GUIDANCE_MAX_ITEMS (8): vượt thì cổng gateway chặn trước
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
        pii = json.load(open("/Users/kuro/.config/edupilot-team/qc3-work/stub_pii.json")) if os.path.exists("/Users/kuro/.config/edupilot-team/qc3-work/stub_pii.json") else {}
        (cid, d), ids = good[0]
        for t in (f"Bài của {pii.get('name', 'Nguyễn Văn Kha')} thường thiếu ví dụ cụ thể.", f"Sinh viên mã {pii.get('mssv', '20229001')} hay viết quá ngắn.", f"Liên hệ {pii.get('email', 'kha@edupilot.local')} để hỏi lại ý.", "Gọi 0912345678 nếu cần làm rõ."):
            items.append({"criterion_id": cid, "text": t, "direction": d, "case_ids": ids[:3]})
    elif mode == "rf_dup_text":
        (cid, d), ids = good[0]
        base = existing[0]["text"] if existing else TEXTS[d].format(c=cid)
        items.append({"criterion_id": existing[0]["criterion_id"] if existing else cid, "text": base, "direction": d, "case_ids": ids[:3]})
    elif mode == "rf_ok_plus1":
        for (cid, d), ids in good: items.append({"criterion_id": cid, "text": TEXTS[d].format(c=cid), "direction": d, "case_ids": ids[:5]})
        items = items[:2] + [{"criterion_id": good[0][0][0], "text": "Tính tổng điểm của bài theo trọng số cho đúng.", "direction": good[0][0][1], "case_ids": good[0][1][:3]}]
    elif mode in ("rf_kw_exempt", "rf_kw_forbidden", "rf_mix14", "rf_n3"):
        crit = []
        for c in cases:
            if c["criterion_id"] not in crit: crit.append(c["criterion_id"])
        def ids_of(cid): return [c["case_id"] for c in cases if c["criterion_id"] == cid]
        def dir_of(cid): return direction([c for c in cases if c["criterion_id"] == cid][0])
        c2 = good[0][0][0]; c1 = [x for x in crit if x != c2][0]
        def it(cid, t, ids=None): return {"criterion_id": cid, "text": t, "direction": dir_of(cid) if ids is None else good[0][0][1], "case_ids": (ids or ids_of(cid))[:3]}
        d2 = good[0][0][1]; g2 = good[0][1]
        if mode == "rf_kw_exempt":
            for t in ("Tổng quát hoá kết luận nhưng nêu rõ giả định.", "Khuyến khích tổng hợp nhiều nguồn khi lập luận.", "Tổng quan ngắn gọn rồi đi vào phân tích."):
                items.append({"criterion_id": c2, "text": t, "direction": d2, "case_ids": g2[:3]})
        elif mode == "rf_kw_forbidden":
            for t in ("Tính tổng điểm của bài theo trọng số cho đúng.", "Tổng quát nhưng đừng quên trừ điểm khi thiếu ví dụ."):
                items.append({"criterion_id": c2, "text": t, "direction": d2, "case_ids": g2[:3]})
        elif mode in ("rf_mix14", "rf_n3"):
            for t in ("Tính tổng điểm của bài theo trọng số cho đúng.", "Tổng quát nhưng đừng quên trừ điểm khi thiếu ví dụ."):
                items.append({"criterion_id": c2, "text": t, "direction": d2, "case_ids": g2[:3]})
            mixed = None
            for cid in crit:
                ca = [c["case_id"] for c in cases if c["criterion_id"] == cid and direction(c) == "STRICTER"]; cb = [c["case_id"] for c in cases if c["criterion_id"] == cid and direction(c) == "LENIENT"]
                if ca and cb: mixed = (cid, ca[0], cb[0]); break
            if mixed: items.append({"criterion_id": mixed[0], "text": "Nêu rõ bằng chứng cho mỗi nhận định trong phần phân tích.", "direction": "STRICTER", "case_ids": [mixed[1], mixed[2]]})
    return items

class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers()
        self.wfile.write(json.dumps({"data": [{"id": "stub-chat"}]}).encode())
    def do_POST(self):
        n = int(self.headers.get("content-length") or 0); body = json.loads(self.rfile.read(n) or b"{}")
        mode = (open("/Users/kuro/.config/edupilot-team/qc3-work/stub_mode").read().strip() if os.path.exists("/Users/kuro/.config/edupilot-team/qc3-work/stub_mode") else "allow")
        with open("/Users/kuro/.config/edupilot-team/qc3-work/stub_req.jsonl", "a") as f: f.write(json.dumps({"t": time.time(), "mode": mode, "body": body}, ensure_ascii=False) + "\n")
        if "[[QCERR]]" in json.dumps(body, ensure_ascii=False) and "rút ra lưu ý chấm" not in json.dumps(body, ensure_ascii=False):  # bài có dấu [[QCERR]] luôn lỗi 500
            out = {"id": "x", "object": "chat.completion", "created": 0, "model": "stub-chat", "choices": [{"index": 0, "message": {"role": "assistant", "content": "khong phai json {{"}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}}
            self.send_response(200); self.send_header("content-type", "application/json"); self.end_headers(); self.wfile.write(json.dumps(out).encode()); return
        if mode == "error":
            self.send_response(500); self.end_headers(); self.wfile.write(b'{"error":{"message":"stub error"}}'); return
        if mode.endswith("_slow"): time.sleep(7); mode = mode[:-5]
        if mode == "slow": time.sleep(12)
        msgs = body.get("messages", [])
        system = " ".join(m.get("content", "") for m in msgs if m.get("role") == "system")
        if mode == "invalid": content = "khong phai json {{"
        elif "rút ra lưu ý chấm" in system:
            user = " ".join(m.get("content", "") for m in msgs if m.get("role") == "user")
            x = re.search(r"<<<[^\n]*\n(\{.*\})\n[^\n]*>>>", user, re.S)
            data = json.loads(x.group(1)) if x else {}
            json.dump(data, open("/Users/kuro/.config/edupilot-team/qc3-work/stub_reflect_in.json", "w"), ensure_ascii=False)
            content = json.dumps({"items": reflect_items(mode if mode.startswith("rf_") else "rf_ok", data)}, ensure_ascii=False)
        elif "RUBRIC (thứ tự chấm)" in system:  # chấm bài tự luận: ghi system vào /Users/kuro/.config/edupilot-team/qc3-work/stub_grading_sys.jsonl, trả đầu ra hợp lệ
            with open("/Users/kuro/.config/edupilot-team/qc3-work/stub_grading_sys.jsonl", "a") as f: f.write(json.dumps({"t": time.time(), "system": system}, ensure_ascii=False) + "\n")
            sm = re.search(r"JSON Schema sau[^\n]*\n(\{.*\})", system)
            ids = json.loads(sm.group(1))["properties"]["criteria"]["items"]["properties"]["id"]["enum"] if sm else []
            content = json.dumps({"criteria": [{"id": i, "score": "1.00", "evidence_quotes": [], "comment": f"Nhận xét giả của stub cho {i}."} for i in ids],
                                  "overall_feedback": "Nhận xét tổng quát giả của stub.", "confidence": 0.8, "flags": [], "total": f"{len(ids):.2f}"}, ensure_ascii=False)
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
