# s8core: dựng lớp riêng cho nhóm trích công thức / giao diện: tạo lớp + tải quy chế COURSE_POLICY (docling) và chờ READY. Dùng: s8core-prep.py <MÃ_LỚP> <pdf> <tiêu đề> [type=COURSE_POLICY]
import hashlib, sys, time, urllib.request
from s8core_lib import *
CODE, PDF, TITLE = sys.argv[1:4]; TYPE = sys.argv[4] if len(sys.argv) > 4 else "COURSE_POLICY"
TID = sq1("select id from users where email='teacher@edupilot.local'"); TAID = sq1("select id from users where email='ta@edupilot.local'")
cidx = sq1(f"select id from courses where class_code='{CODE}'")
if not cidx:
    c, r = call("POST", "/admin/courses", "admin", {"subject_code": "QCD", "class_code": CODE, "name": f"QC s8core {CODE}", "semester": "2026-2027-HK1", "capacity": 60, "teacher_id": TID}); assert c in (200, 201), (c, r); cidx = r["id"]
    print("assign", call("POST", f"/admin/courses/{cidx}/assign", "admin", {"ta_ids": [TAID]})[0])
    for a in ("sv.gioi", "sv.kha", "sv.nguyco"):
        sql(f"insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{cidx}', id,'STUDENT','ACTIVE','ROSTER',student_code from users where email='{a}@edupilot.local'")
open(f"/tmp/s8core/{CODE}.id", "w").write(cidx); print(CODE, cidx)
if PDF != "-":
    b = open(PDF, "rb").read() + f"\n%QC-s8core-{CODE}\n".encode(); sha = hashlib.sha256(b).hexdigest()
    c, pre = call("POST", f"/courses/{cidx}/uploads/presign", "teacher", {"purpose": "document", "filename": PDF.split("/")[-1], "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha}); assert c == 200, (c, pre)
    req = urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"])
    with urllib.request.urlopen(req, context=CTX, timeout=120) as r: print("PUT", r.status)
    c, done = call("POST", f"/courses/{cidx}/uploads/complete", "teacher", {"upload_id": pre["upload_id"], "title": TITLE, "type": TYPE}); print(c, done); jid = done.get("job_id")
    for _ in range(90):
        d = sql(f"select status from documents where course_id='{cidx}' and title='{TITLE}'")
        if d and d[0][0] in ("READY", "FAILED"): break
        time.sleep(5)
    print("doc", sql(f"select id,status,type,category from documents where course_id='{cidx}'"))
