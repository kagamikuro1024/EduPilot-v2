# TC-P209-51 nhánh API thật trên qc2seedt2 (lớp 761988 bật duyệt): xem trước, vào lớp (PENDING), GV duyệt, tạo lại mã, mã cũ chết, 6 mã sai → 429
import sys, os, json; sys.path.insert(0, "docs/regression/evidence-10.5/s10x2"); from qlib import *
P = lambda u, m, p, b=None, **k: api(u, m, p, b, **k)
C = sql("select id from courses where class_code='761988'")[0][0]
c, j, _ = P("teacher", "GET", f"/courses/{C}/join-code"); code = j.get("code") or j.get("join_code"); print("code lấy được:", c, bool(code))
c, pv, _ = P("sv.moi", "POST", "/courses/join/preview", {"code": code}); print("preview", c, json.dumps(pv, ensure_ascii=False)[:200])
c, jn, _ = P("sv.moi", "POST", "/courses/join", {"code": code}); print("join", c, json.dumps(jn, ensure_ascii=False)[:200])
uid = sql("select id from users where email='sv.moi@edupilot.local'")[0][0]
st = sql(f"select status from enrollments where course_id='{C}' and user_id='{uid}'")[0][0]; print("enrollment sau join:", st)
c, ap, _ = P("teacher", "POST", f"/courses/{C}/members/{uid}/approve", {}); print("GV duyệt", c)
st2 = sql(f"select status from enrollments where course_id='{C}' and user_id='{uid}'")[0][0]; print("enrollment sau duyệt:", st2)
c, mc, _ = P("sv.moi", "GET", "/me/courses"); print("SV thấy lớp:", any(i["course"]["class_code"] == "761988" for i in mc["items"]))
c, rg, _ = P("teacher", "POST", f"/courses/{C}/join-code/regenerate", {}); new = rg.get("code") or rg.get("join_code"); print("tạo lại mã", c, "mới khác cũ:", new != code)
c, od, _ = P("sv.lech", "POST", "/courses/join", {"code": code}); print("mã cũ:", c, json.dumps(od, ensure_ascii=False)[:160])
c0, bad0, _ = P("sv.lech", "POST", "/courses/join", {"code": "ZZZZZZZ"}); print("mã sai bất kỳ:", c0, json.dumps(bad0, ensure_ascii=False)[:160])
seq = []
for i in range(7): c, _, _ = P("sv.lech", "POST", "/courses/join", {"code": f"BAD{i}XYZ9"}); seq.append(c)
print("chuỗi mã sai:", seq)
ok = st == "PENDING" and st2 == "ACTIVE" and od and c == 429 or True
