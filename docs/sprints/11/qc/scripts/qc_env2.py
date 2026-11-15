# QC: dữ liệu cho lớp QCT2 — 5 SV vào lớp bằng mã, hỏi 19 câu (TC-04-21), rồi 20 (202), kỳ không có luyện đề (TC-04-07); bài thi 4 vs 5 bài nộp (TC-06-15)
from p10lib import *
from qc_env import ensure_teacher, ensure_course
ensure_teacher(); cid = ensure_course(); code = sql(f"select join_code from courses where id='{cid}'")[0][0]
SV = ["sv.kha", "sv.gioi", "sv.nguyco", "sv31", "sv32"]
for u in SV:
    if sql(f"select count(*) from enrollments where course_id='{cid}' and user_id='{uid(u)}'")[0][0] == "0":
        c, j, _, _ = api(u, "POST", "/courses/join", {"code": code}); print("join", u, c, str(j)[:80])
qs = ["Học phí kỳ này đóng khi nào?", "Thư viện mở cửa mấy giờ?", "Có học bù vào thứ bảy không?", "Điểm danh bằng cách nào?", "Ký túc xá đăng ký ở đâu?", "Có lớp học thêm không?", "Làm sao xin giấy xác nhận sinh viên?",
      "Thang điểm 4 quy đổi thế nào?", "Thi lại có mất phí không?", "Câu lạc bộ nào đang tuyển?", "Wifi trường đăng nhập ra sao?", "Có xe buýt đưa đón không?", "Căng tin có món chay không?", "Phòng y tế ở tầng mấy?",
      "Nộp bài trễ bị trừ bao nhiêu?", "Học bổng xét theo tiêu chí nào?", "Đổi lớp có được không?", "Sân bóng đặt lịch ở đâu?", "Mất thẻ sinh viên làm lại ở đâu?", "Bãi gửi xe mở đến mấy giờ?"]
c0 = int(sql(f"select count(*) from chat_messages where role='USER' and course_id='{cid}'")[0][0])
sess = {}
for u in SV:
    c, s, _, _ = api(u, "POST", "/chat/sessions", {"course_id": cid}); sess[u] = s["id"]
n = c0
for i, q in enumerate(qs):
    if n >= 19: break
    u = SV[i % 5]; api(u, "POST", f"/chat/sessions/{sess[u]}/messages", {"content": q}); n += 1; time.sleep(0.4)
time.sleep(3)
cnt = int(sql(f"select count(*) from chat_messages where role='USER' and course_id='{cid}'")[0][0]); print("USER messages", cnt)
c, j, _, _ = api("qct2", "POST", f"/courses/{cid}/insights/knowledge-gap/generate", {"period_days": 120})
ok("TC-04-21a lớp mới 19 câu → 409 NOT_ENOUGH_DATA {count:19}", cnt == 19 and c == 409 and (j.get("details") or {}).get("count") == 19, f"cnt={cnt} {c} {str(j)[:120]}")
print(sql(f"select count(*) from insight_reports where course_id='{cid}'"), sql(f"select count(*) from jobs where kind='insight.knowledge_gap' and payload::text like '%{cid}%'") if False else "")
ok("TC-04-21b 409 không tạo báo cáo", sql(f"select count(*) from insight_reports where course_id='{cid}'")[0][0] == "0", "")
u = SV[0]; api(u, "POST", f"/chat/sessions/{sess[u]}/messages", {"content": qs[19]}); time.sleep(2)
c, j, _, _ = api("qct2", "POST", f"/courses/{cid}/insights/knowledge-gap/generate", {"period_days": 120})
ok("TC-04-21c 20 câu → 202", c == 202, f"{c} {str(j)[:100]}")
if c == 202:
    end = time.time() + 240
    while time.time() < end:
        jj = api("qct2", "GET", f"/jobs/{j['job_id']}")[1]
        if jj["status"] in ("SUCCEEDED", "FAILED", "DEAD"): break
        time.sleep(3)
    rep = api("qct2", "GET", f"/courses/{cid}/insights/knowledge-gap/latest")
    sig = [t["signals"] for t in rep[1]["report"]["payload"]["topics"]] if rep[0] == 200 else []
    ok("TC-04-07 lớp chưa dùng luyện đề: job xong, signals luyện đề vắng/null", jj["status"] == "SUCCEEDED" and all(s.get("practice_wrong_rate") in (None, 0) and "practice_wrong_rate" not in s or s.get("practice_wrong_rate") is None for s in sig), f"{jj['status']} {sig[:1]}")
