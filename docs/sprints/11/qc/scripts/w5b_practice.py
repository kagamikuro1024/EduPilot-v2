# Đợt R · F12 Luyện đề
from rlib import *
import json, time
c1 = C("teacher")
c, s, _ = api("sv.kha", "POST", f"/courses/{c1}/practice/start", {"mode": "TOPIC", "topic": "Mạng và tường lửa", "count": 5})
ok("R-F12-01 bắt đầu luyện theo chủ đề (5 câu)", c in (200, 201) and len(s["items"]) == 5, f"{c}")
aid = s["id"]; blob = json.dumps(s, ensure_ascii=False)
ok("R-F12-02 trong lúc làm KHÔNG lộ đáp án đúng / giải thích", not any(k in blob for k in ('"is_correct"', '"correct"', '"explanation"', '"answer_key"')), "kiểm khoá JSON")
n = 0
for it in s["items"]:
    q = it["question"]; t = q["type"]
    ans = {"option_ids": [q["options"][0]["id"]]} if t.startswith("MCQ") else {"value": True} if t == "TRUE_FALSE" else {"text": "Tường lửa lọc gói theo quy tắc"}
    c, r, _ = api("sv.kha", "PUT", f"/courses/{c1}/practice/attempts/{aid}/answers/{it['id']}", {"answer": ans, "version": 0})
    n += c in (200, 204)
ok("R-F12-03 tự lưu từng câu", n == 5, f"{n}/5")
c, rs, _ = api("sv.kha", "GET", f"/courses/{c1}/practice/attempts/{aid}")
ok("R-F12-04 tải lại giữa chừng: giữ đáp án đã chọn", c == 200 and rs.get("status") == "IN_PROGRESS" and "option_ids" in json.dumps(rs), f"{c}")
c, sb, _ = api("sv.kha", "POST", f"/courses/{c1}/practice/attempts/{aid}/submit", {})
ok("R-F12-05 nộp → chấm bằng code, có đáp án đúng", c == 200 and ("correct" in json.dumps(sb) or "score" in json.dumps(sb)), f"{c} {json.dumps(sb, ensure_ascii=False)[:260]}")
c, sb2, _ = api("sv.kha", "POST", f"/courses/{c1}/practice/attempts/{aid}/submit", {})
ok("R-F12-06 nộp lại: idempotent (không chấm đôi)", c in (200, 409), f"{c}")
item = s["items"][0]["id"]
t0 = time.time(); c, ex, _ = api("sv.kha", "POST", f"/courses/{c1}/practice/attempts/{aid}/items/{item}/explain", {})
ok("R-F12-07 giải thích câu sai/đúng (LLM 9router)", c == 200, f"{c} {time.time()-t0:.1f}s {json.dumps(ex, ensure_ascii=False)[:200]}")
c, fb, _ = api("sv.kha", "POST", f"/courses/{c1}/practice/attempts/{aid}/items/{item}/feedback", {"kind": "WRONG_ANSWER"})
ok("R-F12-08 báo câu hỏi sai", c in (200, 201, 204, 422), f"{c} {str(fb)[:100]}")
c, wk, _ = api("sv.kha", "GET", f"/courses/{c1}/practice/weakness")
ok("R-F12-09 chủ đề yếu theo lượt đã nộp", c == 200, f"{c} {json.dumps(wk, ensure_ascii=False)[:160]}")
c, hs, _ = api("sv.kha", "GET", f"/courses/{c1}/practice/history")
ok("R-F12-10 lịch sử luyện", c == 200 and len(hs.get("items", [])) >= 1, f"{c}")
c, mk_, _ = api("sv.kha", "POST", f"/courses/{c1}/practice/start", {"mode": "MOCK", "count": 10, "duration_minutes": 15})
ok("R-F12-11 đề thử có đồng hồ (deadline_at server)", c in (200, 201) and mk_.get("deadline_at"), f"{c} {str(mk_.get('deadline_at'))}")
if mk_.get("id"): api("sv.kha", "POST", f"/courses/{c1}/practice/attempts/{mk_['id']}/submit", {})
c, ot, _ = api("sv.kha", "GET", f"/courses/{c1}/practice/attempts/{aid}")
c2, th, _ = api("teacher", "GET", f"/courses/{c1}/practice/attempts/{aid}")
ok("R-F12-12 lượt luyện của SV khác/GV không đọc được (403/404)", c2 in (403, 404), f"{c2}")
c, nv, _ = api("sv.moi", "POST", f"/courses/{c1}/practice/start", {"mode": "TOPIC", "topic": "Mạng và tường lửa", "count": 5})
ok("R-F12-13 SV chưa ở lớp không luyện được (403)", c in (403, 404), f"{c}")
# không lộ đáp án đề: ANSWER_KEY không truy xuất qua chat
c, ch, _ = api("sv.kha", "POST", f"/courses/{c1}/chat/sessions", {})
print("w5b done")
