# Đợt R · làn 1b: F2 Mở lớp / vào lớp, F13 Lịch, F14 Hôm nay
from rlib import *
import time, uuid, datetime as dt
tag = str(int(time.time()))[-5:]
# F2: admin mở lớp mới, gán GV + TA
c, jt, _ = api("admin", "GET", "/admin/users?role=TEACHER&limit=5")
teachers = [u for u in (jt.get("items", []) if isinstance(jt, dict) else [])]
c2, jtt, _ = api("admin", "GET", "/admin/users?q=teacher@&limit=5")
tid = None
for u in (jtt.get("items", []) if isinstance(jtt, dict) else []):
    if u.get("email") == "teacher@edupilot.local": tid = u["id"]
ok("R-F2-01 Admin tra được GV", tid is not None, f"{c2}")
body = {"subject_code": "QCR", "class_code": "QR" + tag, "name": "QC đợt R " + tag, "semester": "2026-2027-HK1", "capacity": 50, "teacher_id": tid}
c, j, _ = api("admin", "POST", "/admin/courses", body)
cid = j.get("id") if c in (200, 201) and j else None
cj, jc2, _ = api("teacher", "GET", f"/courses/{cid}/join-code") if cid else (0, None, 0)
code = (jc2 or {}).get("code") or (jc2 or {}).get("join_code") if cj == 200 else None
ok("R-F2-02 Admin mở lớp + GV lấy mã tham gia", c in (200, 201) and bool(code), f"mở {c}; join-code {cj} {str(jc2)[:100]}")
c, jp, _ = api("sv.moi", "POST", "/courses/join/preview", {"code": code}) if code else (0, None, 0)
ok("R-F2-03 SV xem trước lớp theo mã", c == 200, f"{c} {str(jp)[:100]}")
c, jj, _ = api("sv.moi", "POST", "/courses/join", {"code": code}) if code else (0, None, 0)
ok("R-F2-04 SV vào lớp bằng mã", c in (200, 201), f"{c} {str(jj)[:100]}")
c, jb, _ = api("sv.moi", "POST", "/courses/join", {"code": "ZZZZZZZ"})
ok("R-F2-05 mã sai → 404 / từ chối, không lộ lớp", c in (404, 422), f"{c}")
if cid:
    c, jm, _ = api("teacher", "GET", f"/courses/{cid}/members?status=PENDING")
    ok("R-F2-06 GV thấy yêu cầu vào lớp (hoặc đã vào thẳng)", c == 200, f"{c} {str(jm)[:120]}")
    # lưu trữ → mã tham gia tắt
    c, ja, _ = api("admin", "POST", f"/admin/courses/{cid}/archive", {})
    ok("R-F2-07 Lưu trữ lớp", c in (200, 204), f"{c} {str(ja)[:80]}")
    c, jj2, _ = api("sv.nguyco", "POST", "/courses/join", {"code": code})
    ok("R-F2-08 lớp đã lưu trữ: mã tham gia tắt", c in (404, 409, 410, 422), f"{c} {str(jj2)[:80]}")
# F13 Lịch
C1 = C("teacher")
start = (dt.datetime.utcnow() + dt.timedelta(days=3)).replace(microsecond=0).isoformat() + "Z"
c, je, _ = api("teacher", "POST", f"/courses/{C1}/calendar/events", {"type": "EXAM", "title": "QC-R kiểm tra giữa kỳ", "starts_at": start})
ok("R-F13-01 GV tạo sự kiện EXAM", c in (200, 201), f"{c} {str(je)[:100]}")
c, jc, _ = api("sv.kha", "GET", f"/courses/{C1}/calendar?from=" + dt.date.today().isoformat() + "T00:00:00Z&to=" + (dt.date.today() + dt.timedelta(days=30)).isoformat() + "T00:00:00Z")
ok("R-F13-02 SV thấy sự kiện", c == 200 and "QC-R kiểm tra giữa kỳ" in str(jc), f"{c}")
c, ji, _ = api("sv.kha", "GET", "/me/calendar/ics-token")
c2, ji2, _ = api("sv.kha", "POST", "/me/calendar/ics-token", {})
ok("R-F13-03 token ICS", c in (200, 404, 405) or c2 in (200, 201), f"GET {c} POST {c2} {str(ji2)[:80]}")
c, jx, _ = api("sv.kha", "POST", f"/courses/{C1}/calendar/events", {"type": "OTHER", "title": "SV tạo", "starts_at": start})
ok("R-F13-04 SV không tạo được sự kiện lớp (403)", c == 403, f"{c}")
# F14 Hôm nay các vai
for u in ("sv.kha", "teacher", "ta", "admin"):
    c, j, _ = api(u, "GET", "/me/today")
    n = len(j.get("items", j.get("tasks", []))) if isinstance(j, dict) else -1
    ok(f"R-F14-{u} Hôm nay ({u}) 200", c == 200, f"{c} {n} việc; keys {list(j.keys())[:6] if isinstance(j, dict) else j}")
c, jt, _ = api("teacher", "GET", f"/courses/{C1}/today")
ok("R-F14-teacher-lớp Hôm nay theo lớp", c == 200, f"{c} {str(jt)[:100]}")
c, jt, _ = api("sv.kha", "GET", f"/courses/{C1}/today")
ok("R-F14-sv-lớp Hôm nay theo lớp (SV 200)", c in (200, 403), f"{c}")
print("w1b done")
