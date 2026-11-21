# Dựng lớp riêng QD105A (lớp 1: teacher@, ta@, SV) và QD105B (lớp 2: qct2@, sv38). KHÔNG đụng 761987/761988.
from q105 import *
def mk(code, teacher, tas, students, name):
    ex = sql(f"select id from courses where class_code='{code}'")
    if ex: cid = ex[0][0]; print("có sẵn", code, cid)
    else:
        tid = sql(f"select id from users where email='{teacher}@edupilot.local'")[0][0]
        c, j, _ = api("admin", "POST", "/admin/courses", {"subject_code": "QCD", "class_code": code, "name": name, "semester": "2026-2027-HK1", "capacity": 80, "teacher_id": tid})
        assert c in (200, 201), (c, j); cid = j["id"]
        if tas:
            c, j, _ = api("admin", "POST", f"/admin/courses/{cid}/assign", {"ta_ids": [sql(f"select id from users where email='{t}@edupilot.local'")[0][0] for t in tas]}); print("assign", c)
    jc = sql(f"select join_code from courses where id='{cid}'")[0][0]
    for s in students:
        c, j, _ = api(s, "POST", "/courses/join", {"code": jc})
        if c not in (200, 201, 409): print("join", s, c, j)
    print(code, cid)
mk("QD105A", "teacher", ["ta"], ["sv.kha", "sv.gioi", "sv.nguyco", "sv31", "sv32", "sv33"] + [f"sv{n:02d}" for n in range(4, 14)], "QC smoke10.5 QD105A")
