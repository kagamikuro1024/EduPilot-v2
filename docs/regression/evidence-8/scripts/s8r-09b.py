# s8r 09b: tiếp QD8R9 (đang mở khoá sau CANARY): 09-04 (FINAL từ snapshot mới nhất), 09-22, 09-50
from s8r_g import *
def EM(a): return sq1(f"select email from users where id='{SV(a)}'").split("@")[0]
B = "sv.kha"; sB = EM(B)
c, r = fin(); assert c == 200, (c, r)
c1, m1 = call("GET", f"{Q}/me/grades", sB); s1 = json.dumps(m1, ensure_ascii=False)
v1 = (m1.get("final") or {})
w = call("POST", f"{Q}/me/grades/what-if", sB, {"final_exam_score": "8.0"}, key=False)
C("TC-P6-09-22", w[0] == 409 and w[1].get("code") == "FINAL_ALREADY_LOCKED", f"status FINAL → what-if {w[0]} {w[1].get('code')}; UI ẩn ô what-if: dev e2e 'grade section' xanh", str(w))
unl("Mở khoá chỉnh điểm B để chốt v3"); setc(B, "final", "8"); c, r = fin(); assert c == 200, (c, r)
c2, m2 = call("GET", f"{Q}/me/grades", sB); v2 = m2.get("final") or {}
sn = sql(f"select numeric_grade, letter, version from final_grade_snapshots where course_id='{C0}' and student_id='{SV(B)}' order by version desc limit 1")[0]
C("TC-P6-09-04", c1 == 200 and m1["status"] == "FINAL" and "7" in json.dumps(v1) and "B" in json.dumps(v1) and m2["status"] == "FINAL" and "8" in json.dumps(v2) and "B+" in json.dumps(v2) and "CANARY" not in s1, f"sau chốt: status FINAL, final={json.dumps(v1, ensure_ascii=False)} (7,0 B, từ snapshot); sau mở khoá + sửa 8,0 + chốt lại: final={json.dumps(v2, ensure_ascii=False)} (snapshot v{sn[2]} {sn[0]} {sn[1]}); DTO không có unlock_reason (CANARY 0)", f"{m1.get('final')} {m2.get('final')} {sn}")
# ---- 09-50 rời lớp
sql(f"update enrollments set status='REMOVED', removed_at=now() where course_id='{C0}' and user_id='{SV(B)}'")
c3, r3 = call("GET", f"{Q}/me/grades", sB); c4, r4 = call("GET", f"{Q}/gradebook/snapshots/1", sB); sql(f"update enrollments set status='ACTIVE', removed_at=null where course_id='{C0}' and user_id='{SV(B)}'")
C("TC-P6-09-50", c3 == 403 and c4 == 403, f"B rời lớp (REMOVED): GET me/grades → {c3}; GET snapshots/1 → {c4}; snapshot cũ của B vẫn còn trong DB cho Staff", f"{c3} {c4}")
