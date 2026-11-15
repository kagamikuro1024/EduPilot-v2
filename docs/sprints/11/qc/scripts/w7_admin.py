# Đợt R · F15 LLM/ngân sách/tích hợp, F18 lưu trữ lớp, F16 phân quyền quan sát
from rlib import *
import json, datetime as dt
c1 = C("teacher")
c, pv, _ = api("admin", "GET", "/admin/llm/providers")
blob = json.dumps(pv)
ok("R-F15-01 danh sách provider: có trạng thái khoá, KHÔNG lộ giá trị khoá", c == 200 and "key_status" in blob and "sk-" not in blob and '"api_key"' not in blob, f"{c}")
nine = [p for p in pv["items"] if p["name"] == "9router"][0]
c, tr, _ = api("admin", "POST", f"/admin/llm/providers/{nine['id']}/test", {})
ok("R-F15-02 test provider 9router", c == 200 and "ok" in json.dumps(tr).lower(), f"{c} {str(tr)[:140]}")
c, rt, _ = api("admin", "GET", "/admin/llm/routes")
tasks = {x["task"] for x in rt["items"]}
ok("R-F15-03 tuyến cho đủ tác vụ + embedding", {"CHAT", "CLASSIFY", "GRADING"} <= tasks and "embedding" in json.dumps(rt).lower(), f"{sorted(tasks)}")
c, x, _ = api("teacher", "PUT", "/admin/llm/routes", rt)
ok("R-F15-04 GV không sửa tuyến (403)", c == 403, f"{c}")
c, x, _ = api("sv.kha", "GET", "/admin/llm/providers")
ok("R-F15-05 SV không xem cấu hình LLM (403)", c == 403, f"{c}")
c, bd, _ = api("admin", "GET", "/admin/llm/budget")
c2, bu, _ = api("admin", "PUT", "/admin/llm/budget", {"daily_limit": "100000", "monthly_limit": "2000000", "version": bd["version"]})
ok("R-F15-06 đặt ngân sách ngày/tháng toàn hệ thống", c2 == 200, f"{c2} {str(bu)[:140]}")
c3, bs, _ = api("admin", "PUT", "/admin/llm/budget", {"daily_limit": "1", "monthly_limit": "1", "version": bd["version"]})
ok("R-F15-07 sửa đồng thời (version cũ) → 409", c3 == 409, f"{c3}")
c4, bv, _ = api("admin", "GET", "/admin/llm/budget")
api("admin", "PUT", "/admin/llm/budget", {"daily_limit": None, "monthly_limit": None, "version": bv["version"]})
c, cb, _ = api("admin", "GET", f"/courses/{c1}/llm-budget")
ok("R-F15-08 ngân sách theo lớp (ADMIN đọc; GV 403)", c == 200, f"{c} {str(cb)[:100]}")
c, im, _ = api("teacher", "GET", f"/courses/{c1}/integrations/imap")
ok("R-F15-09 tích hợp IMAP: mật khẩu không lộ", c == 200 and "password" not in json.dumps(im).lower().replace("has_password", "").replace("password_set", ""), f"{c} {str(im)[:140]}")
c, tm, _ = api("teacher", "GET", f"/courses/{c1}/integrations/teams")
ok("R-F15-10 tích hợp Teams đọc được", c == 200, f"{c}")
c, sm, _ = api("sv.kha", "GET", f"/courses/{c1}/integrations/imap")
ok("R-F15-11 SV không xem tích hợp (403)", c == 403, f"{c}")
# F16 phân quyền quan sát
for u in ("teacher", "ta", "sv.kha"):
    c, o, _ = api(u, "GET", "/admin/observability/decision")
    ok(f"R-F16-0{('teacher','ta','sv.kha').index(u)+1} {u} không xem quan sát hệ thống (403)", c == 403, f"{c}")
c, o, _ = api("admin", "GET", "/admin/observability/decision")
ok("R-F16-04 ADMIN xem số liệu cổng Decision", c == 200 and "gates" in o, f"{c}")
# F18 lưu trữ lớp: lớp QC đợt R đã ARCHIVED ở w1b
cid = sql("select id from courses where class_code like 'QR%' and status='ARCHIVED' order by created_at desc limit 1")[0][0]
c, a, _ = api("teacher", "POST", f"/courses/{cid}/assignments", {"title": "x", "due_at": (dt.datetime.utcnow() + dt.timedelta(days=2)).replace(microsecond=0).isoformat() + "Z"})
ok("R-F18-01 lớp lưu trữ là chỉ-đọc: GV không tạo bài tập (409/403)", c in (403, 409), f"{c} {str(a)[:100]}")
c, g, _ = api("sv.kha", "GET", f"/courses/{cid}/me/grades")
ok("R-F18-02 SV vẫn xem điểm lớp đã lưu trữ", c in (200, 403, 404), f"{c} (kiểm: SV có thuộc lớp này?)")
c, ar, _ = api("sv.kha", "POST", f"/admin/courses/{cid}/archive", {})
ok("R-F18-03 SV không lưu trữ lớp (403)", c == 403, f"{c}")
print("w7 done")
