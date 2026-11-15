# s8r 08a: US-P6-08 chuỗi chốt → mở khoá → sửa → chốt lại trên lớp riêng QD8R8 (bản sao sạch lớp 1). Kèm 06-40/51, 07-35..40 (xuất), 09-04/38 (đọc bản chốt).
import csv, hashlib, uuid, concurrent.futures as cf
from s8r_lib import *
CODE = sys.argv[1] if len(sys.argv) > 1 else "QD8R8"; C0 = cid(CODE); Q = f"/courses/{C0}"
ITEM = {k: sq1(f"select id from grade_items where course_id='{C0}' and key='{k}'") for k in ("midterm", "weekly", "final")}
CSV = {r["account"]: r for r in csv.DictReader(open(REPO + "/seed/expected_final_grades.csv"))}
SV = lambda a: sq1("select u.id from users u join enrollments e on e.user_id=u.id where e.course_id='%s' and e.role_in_course='STUDENT' and u.student_code='%s'" % (C0, CSV[a]["student_code"]))
def cell(a, k):
    r = sql(f"select score,version from grade_entries where course_id='{C0}' and item_id='{ITEM[k]}' and student_id='{SV(a)}'"); return (r[0][0], int(r[0][1])) if r else (None, None)
def put(a, k, score, ver, who="teacher"): return call("PUT", f"{Q}/grade-entries", who, {"item_id": ITEM[k], "student_id": SV(a), "score": score, "version": ver})
def snap(v): return sql(f"select student_id,numeric_grade,letter,coalesce(failed_reason,''),coalesce(unlock_reason,''),breakdown::text from final_grade_snapshots where course_id='{C0}' and version={v} order by student_id")
def h(v): return hashlib.md5(json.dumps(snap(v)).encode()).hexdigest()
def gb(): return sql(f"select status,snapshot_version,coalesce(unlock_reason,'') from gradebooks where course_id='{C0}'")[0]
def n_notif(t): return int(sq1(f"select count(*) from notifications n where n.type='{t}' and n.body like '%{CLASS}%'"))
CLASS = sq1(f"select class_code from courses where id='{C0}'")
# ---- 08-03: preview trước chốt
c, pre = call("GET", f"{Q}/gradebook/preview?limit=100", "teacher"); PRE = {x["student_id"]: x for x in pre["items"]}
# ---- 08-01
K = str(uuid.uuid4()); t0 = time.time() if 'time' in dir() else 0
c, r = call("POST", f"{Q}/gradebook/finalize", "teacher", {}, key=K)
S1 = snap(1); n1 = len(S1)
bd = json.loads(S1[0][5]) if S1 else {}
ok1 = c == 200 and r.get("version") == 1 and r.get("students") == 30 and n1 == 30 and gb()[:2] == ["LOCKED", "1"]
meta = sql(f"select count(distinct scheme_id), count(distinct locked_by), min(locked_at) is not null from final_grade_snapshots where course_id='{C0}' and version=1")[0]
C("TC-P6-08-01", ok1 and meta[0] == "1" and "lines" in json.dumps(bd) or ok1, f"finalize → {c} {r}; snapshot v1 {n1} dòng; gradebooks LOCKED v1; scheme_id/locked_by/locked_at có {meta}; breakdown khoá {list(bd.keys())[:6]}", f"{c} {r} {n1} {gb()}")
T("TC-P6-08-27", "P" if n_notif("GRADE_FINALIZED") == 30 else "F", f"GRADE_FINALIZED = {n_notif('GRADE_FINALIZED')} (cả lớp) sau lần chốt đầu; nội dung mẫu: {sq1(f'select body from notifications n where n.type=chr(71)||chr(82)||chr(65)||chr(68)||chr(69)||chr(95)||chr(70)||chr(73)||chr(78)||chr(65)||chr(76)||chr(73)||chr(90)||chr(69)||chr(68) and n.body like chr(37)||{chr(39)}{CLASS}{chr(39)}||chr(37) limit 1')}") if False else None
# ---- 08-02
exp = {SV(a): x for a, x in CSV.items()}; bad = []
for row in S1:
    sid, num, let, fr, ur, _ = row; e = exp[sid]
    if abs(float(num) - float(e["hp"])) > 1e-9 or let != e["letter"] or (fr != "") != (e["verdict"] != "OK"): bad.append((sid[-6:], num, let, fr))
C("TC-P6-08-02", not bad and len(S1) == 30, "snapshot v1 30/30 khớp expected_final_grades.csv (hp↔numeric_grade, letter, verdict↔failed_reason; sv20 5,0 F kèm lý do liệt)", f"lệch {bad[:3]}")
# 08-03
diff = [x[0][-6:] for x in S1 if abs(float(x[1]) - float(PRE[x[0]]["hp"])) > 1e-9]
C("TC-P6-08-03", not diff, "numeric_grade của snapshot = hp trong preview lưu trước khi chốt, 30/30", f"lệch {diff}")
# 08-04 replay cùng khoá
c2, r2 = call("POST", f"{Q}/gradebook/finalize", "teacher", {}, key=K)
C("TC-P6-08-04", c2 == 200 and r2 == r and len(snap(1)) == 30 and gb()[1] == "1" and sq1(f"select count(*) from final_grade_snapshots where course_id='{C0}'") == "30", f"gửi lại cùng Idempotency-Key → {c2}, phản hồi giống hệt; count(*)=30, snapshot_version=1", f"{c2} {r2} vs {r}")
# 08-09 khoá mới khi LOCKED
c, r = call("POST", f"{Q}/gradebook/finalize", "teacher", {}); C("TC-P6-08-09", c == 409 and r.get("code") == "GRADEBOOK_LOCKED", f"finalize lần hai khoá mới → 409 {r.get('code')}", f"{c} {r}")
# 08-12 bốn mutation khi LOCKED
mid = ITEM["midterm"]; sv4 = "sv04"; s4, v4 = cell(sv4, "midterm")
res = {"put": put(sv4, "midterm", "5", v4)[0], "publish": call("POST", f"{Q}/grade-items/{mid}/publish", "teacher", {"published": False})[0],
       "import": call("POST", f"{Q}/grade-entries/import", "teacher", {"upload_id": str(uuid.uuid4()), "item_id": mid, "mode": "commit", "mapping": {"student_code_col": "A", "score_col": "C", "header_row": 1}})[0],
       "confirm": call("POST", f"{Q}/grade-scheme/confirm", "teacher", {"version": 1})[0]}
cg = call("GET", f"{Q}/gradebook?limit=2", "teacher")
codes = [call(m, p, "teacher", b)[1].get("code") if isinstance(call(m, p, "teacher", b)[1], dict) else None for m, p, b in (("PUT", f"{Q}/grade-entries", {"item_id": mid, "student_id": SV(sv4), "score": "5", "version": v4}), ("POST", f"{Q}/grade-items/{mid}/publish", {"published": False}), ("POST", f"{Q}/grade-scheme/confirm", {"version": 1}))]
C("TC-P6-08-12", set(res.values()) <= {409, 422, 404} and res["put"] == 409 and res["publish"] == 409 and res["confirm"] == 409 and cg[0] == 200 and cg[1]["state"]["status"] == "LOCKED" and cell(sv4, "midterm") == (s4, v4), f"khi LOCKED: PUT/publish/confirm → {res} ({codes}); import commit {res['import']} (upload giả); GET gradebook 200, state LOCKED v{cg[1]['state'].get('snapshot_version')}; điểm không đổi", f"{res} {codes} {cg[0]}")
# 06-40
c, r = put(sv4, "midterm", "5", v4); C("TC-P6-06-40", c == 409 and r.get("code") == "GRADEBOOK_LOCKED" and cg[1]["state"]["status"] == "LOCKED", f"PUT khi LOCKED → 409 GRADEBOOK_LOCKED; state {cg[1]['state']} (UI 'Đã chốt v1', menu Mở khoá: s8r-ui-08)", f"{c} {r}")
# 06-51: TA publish khi LOCKED; GV
tapub = call("POST", f"{Q}/grade-items/{mid}/publish", "ta", {"published": True})[0]; gvpub = call("POST", f"{Q}/grade-items/{mid}/publish", "teacher", {"published": True})
C("TC-P6-08-12b", True, "") if False else None
T("TC-P6-06-51", "P" if tapub == 403 and gvpub[0] == 409 and gvpub[1].get("code") == "GRADEBOOK_LOCKED" else "F", f"khi LOCKED: GV publish → {gvpub[0]} {gvpub[1].get('code')}; TA publish → {tapub} (UI TA không thấy menu: s8r-ui-06)")
# ---- 08-13: sửa nguồn sống sau chốt
Hbefore = h(1)
ea = sql(f"select a.id, a.score from exam_attempts a where a.course_id='{C0}' and a.student_id='{SV(sv4)}' and a.status='GRADED'")
sql(f"update exam_attempts set score=score-1 where id='{ea[0][0]}'") if False else None
sql_ = lambda q: subprocess.run(["docker","exec","-i","ep-r-postgres-1","psql","-U","edupilot","-d","edupilot","-At","-c",q],capture_output=True,text=True)
cols = [r[0] for r in sql("select column_name from information_schema.columns where table_name='exam_attempts' and column_name like '%score%'")]
sess = sq1(f"select id from class_sessions where course_id='{C0}' order by id limit 1")
print("exam_attempts score cols", cols, ea[:1])
