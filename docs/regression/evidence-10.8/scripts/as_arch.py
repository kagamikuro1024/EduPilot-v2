from asl import *
S=st();C2=S["cid2"]
policy(C2, rotate_seconds=5)
s, w, d = neww(C2); t = tokenof(d)
c, j, _ = api(ADMIN, "POST", f"/admin/courses/{C2}/archive", None); print("archive", c)
v=api(TEACHER, "GET", f"/courses/{C2}")[1]["settings"]["version"]
r = {"checkin": checkin("sv.gioi", C2, {"token": t}), "settings": api(TEACHER, "PUT", f"/courses/{C2}/settings", {"attendance": {"self_checkin": True}, "version": v}),
     "open": openw(C2, sessions(TEACHER, C2)[5]["id"]), "close": close(C2, w), "absent": api(TEACHER, "POST", f"/courses/{C2}/sessions/{s}/absent-unchecked", None, idem=str(uuid.uuid4())),
     "headcount": api(TEACHER, "PATCH", f"/courses/{C2}/checkin-windows/{w}", {"headcount": 3}), "spot": api(TEACHER, "POST", f"/courses/{C2}/checkin-windows/{w}/spot-check", {"n": 3})}
codes = {k: (v[0], v[1].get("code") if isinstance(v[1], dict) else None) for k, v in r.items()}
ok("TC-AS01-09/02-23/03-19/05-14 lớp lưu trữ: 6 ghi → 409 COURSE_ARCHIVED (spot-check chỉ đọc: 200)", all(v == (409, "COURSE_ARCHIVED") for k, v in codes.items() if k != "spot") and codes["spot"][0] == 200, str(codes))
