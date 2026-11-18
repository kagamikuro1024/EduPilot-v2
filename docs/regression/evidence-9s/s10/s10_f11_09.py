# Smoke S10 lô Forms (US-P9-11) trên lớp QD10A: nhân bản công thức xác nhận của 761987 (đọc-chỉ) với 2 cột MANUAL, rồi chạy TC-11-xx qua API
import sys, io, csv, hashlib, urllib.request
from pm import *
T_ID = sql("select id from users where email='teacher@edupilot.local'")[0][0]
def fixture():
    if sql(f"select count(*) from grade_schemes where course_id='{C1}'")[0][0] != "0": return
    C0 = sql("select id from courses where class_code='761987'")[0][0]
    sid = sql(f"insert into grade_schemes (course_id,scheme_no,status,source_title,confirmed_by,confirmed_at,created_by,version,unclear_count) select '{C1}',1,status,source_title,'{T_ID}',now(),'{T_ID}',version,0 from grade_schemes where course_id='{C0}' and status='CONFIRMED' limit 1 returning id")[0][0]
    sql(f"insert into gradebooks (course_id) values ('{C1}') on conflict do nothing")
    cm = {}
    for r in sql(f"select id,key from grade_components where course_id='{C0}' and scheme_id=(select id from grade_schemes where course_id='{C0}' and status='CONFIRMED' limit 1)"):
        cm[r[1]] = sql(f"insert into grade_components (course_id,scheme_id,key,name,grp,weight,position,citations,filled_by_user) select '{C1}','{sid}',key,name,grp,weight,position,citations,filled_by_user from grade_components where id='{r[0]}' returning id")[0][0]
    sql(f"insert into grade_rules (course_id,scheme_id,kind,applicable,params,source_quote,source_page,filled_by_user) select '{C1}','{sid}',kind,applicable,params,source_quote,source_page,filled_by_user from grade_rules where scheme_id=(select id from grade_schemes where course_id='{C0}' and status='CONFIRMED' limit 1)")
    for k, nm, pos in (("g1", "Forms G1", 1), ("g2", "Forms G2", 2)):
        sql(f"insert into grade_items (course_id,scheme_id,component_id,key,name,max_score,source,position) values ('{C1}','{sid}','{cm['midterm']}','{k}','{nm}',10,'MANUAL',{pos})")
    print("scheme", sid)
fixture()
G = {r[0]: r[1] for r in sql(f"select key,id from grade_items where course_id='{C1}'")}; G1, G2 = G["g1"], G.get("g2") or G.get("g2x")
def presign(user, name, data, mime="text/csv", C=C1):
    sha = hashlib.sha256(data).hexdigest()
    c, pre, _ = api(user, "POST", f"/courses/{C}/uploads/presign", {"purpose": "forms_import", "filename": name, "mime_type": mime, "size_bytes": len(data), "sha256": sha})
    if c != 200: return c, pre
    r = urllib.request.Request(pre["url"], data=data, method=pre["method"], headers=pre["headers"]); urllib.request.urlopen(r, context=CTX).read(); return c, pre
STU = [r for r in sql(f"select u.email,u.student_code,u.full_name from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.role_in_course='STUDENT' and e.status='ACTIVE' and u.email like 'sv%@edupilot.local' order by u.email")]
print(len(STU), STU[:3])
MC = [q for q in bank() if q["type"] == "MCQ_SINGLE" and q["review_status"] == "APPROVED" and q["topic"] == "QC-r10-Y"][:5]
DD = [detail(q["id"]) for q in MC]
def ans(d, good): ids = d["answer_key"]["option_ids"]; return [o for o in d["options"] if (o["id"] in ids) == good][0]["body"]
HDR = ["Email", "Mã số sinh viên", "Completion time"] + [f"Câu {i+1}" for i in range(5)]
def row(em, code, ts, nok, n=5): return [em, code, ts] + [ans(d, i < nok) for i, d in enumerate(DD[:5])]
def mkcsv(rows): b = io.StringIO(); csv.writer(b).writerows([HDR] + rows); return b.getvalue().encode()
MPB = {"header_row": 1, "email_col": "Email", "code_col": "Mã số sinh viên", "time_col": "Completion time", "questions": [{"col": f"Câu {i+1}", "question_id": MC[i]["id"]} for i in range(5)]}
def preview(pre, item=G1, mp=MPB, src="BANK"): return api("teacher", "POST", P + "/forms-import/preview", {"upload_id": pre["upload_id"], "item_id": item, "answer_source": src, "mapping": mp})
def confirm(pre, item=G1, mp=MPB, src="BANK", idem=None): return api("teacher", "POST", P + "/forms-import", {"upload_id": pre["upload_id"], "item_id": item, "answer_source": src, "mapping": mp}, idem=idem or str(uuid.uuid4()))
# TC-11-09: ba tệp xấu → 422 kèm vị trí; tệp > 5 MB bị từ chối ở presign
import random, string
bad_txt = "Email,Mã số sinh viên,Completion time,Câu 1\nsv.kha@edupilot.local,\"001,2026-11-18 10:00,A\nrác không đúng định dạng\x00\x01;;;\n".encode()
c, pre = presign("teacher", "forms-bad.csv", bad_txt); print("bad presign", c, lastj(pre, 160))
r1 = (c, pre) if c != 200 else (lambda p: (p[0], p[1]))(preview(pre))
nocol = io.StringIO(); csv.writer(nocol).writerows([["Email", "Completion time", "Câu 1"], ["a@b.c", "2026-11-18 10:00", "x"]])
c2, pre2 = presign("teacher", "forms-nocol.csv", nocol.getvalue().encode()); r2 = preview(pre2)[:2] if c2 == 200 else (c2, pre2)
big = ("Email,Mã số sinh viên,Completion time," + ",".join(f"Câu {i}" for i in range(1, 6)) + "\n" + ("x@y.z,000,2026-11-18 10:00,a,b,c,d,e\n" * 160000)).encode()
c3, pre3 = presign("teacher", "forms-big.csv", big); print("bad preview", r1[0], lastj(r1[1], 400)); print("nocol preview", r2[0], lastj(r2[1], 500)); print("big size", len(big), "presign", c3, lastj(pre3, 200))
f = lambda r: lastj(r[1], 260)
ok1 = r1[0] in (400, 422) and re.search(r"dòng|cột|line|row|col|vị trí|position", f(r1), re.I)
ok2 = r2[0] in (400, 422) and re.search(r"dòng|cột|line|row|col|vị trí|position|Mã số", f(r2), re.I)
ok3 = c3 in (400, 413, 422)
chk("TC-11-09", bool(ok1 and ok2 and ok3), f"forms-bad.txt (nhị phân): {r1[0]} {f(r1)[:120]}; forms-nocol.csv (thiếu cột «Mã số sinh viên»): {r2[0]} {f(r2)[:120]}; forms-big.csv {len(big)//1024//1024}+ MB: presign {c3} {lastj(pre3, 100)}")
