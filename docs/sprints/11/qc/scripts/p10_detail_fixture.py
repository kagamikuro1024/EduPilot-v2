# P-11-15 ca 4 (QC): nhân bản công thức đã xác nhận của 761987 sang lớp QCT3 (DB), «Giữa kỳ» thành 2 cột MANUAL → trung bình có `detail`; sv31 có điểm cả hai cột + các cột khác. Xoá: python3 p10_detail_fixture.py clean
import sys
from p10lib import *
C1 = course("teacher", "761987"); C3 = sql("select id from courses where class_code='QCT3'")[0][0]
S = uid("sv31"); T = uid("qct2")
if len(sys.argv) > 1 and sys.argv[1] == "clean":
    for t in ("grade_entries", "grade_items", "grade_rules", "grade_components", "grade_schemes"): sql(f"delete from {t} where course_id='{C3}'")
    sql(f"delete from gradebooks where course_id='{C3}'"); sql(f"delete from enrollments where course_id='{C3}' and role_in_course='STUDENT'"); print("đã dọn"); sys.exit()
if sql(f"select count(*) from grade_schemes where course_id='{C3}'")[0][0] != "0": print("đã có"); sys.exit()
sql(f"insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{C3}', id, 'STUDENT','ACTIVE','ADMIN', student_code from users where id='{S}' on conflict do nothing")
sid = sql(f"insert into grade_schemes (course_id,scheme_no,status,source_title,confirmed_by,confirmed_at,created_by,version,unclear_count) select '{C3}',1,status,source_title,'{T}',now(),'{T}',version,0 from grade_schemes where course_id='{C1}' and status='CONFIRMED' limit 1 returning id")[0][0]
sql(f"insert into gradebooks (course_id) values ('{C3}') on conflict do nothing")
cmap = {}
for r in sql(f"select id, key, name, grp, weight, position, citations, filled_by_user from grade_components where course_id='{C1}' and scheme_id=(select id from grade_schemes where course_id='{C1}' and status='CONFIRMED' limit 1)"):
    nid = sql(f"insert into grade_components (course_id,scheme_id,key,name,grp,weight,position,citations,filled_by_user) select '{C3}','{sid}',key,name,grp,weight,position,citations,filled_by_user from grade_components where id='{r[0]}' returning id")[0][0]; cmap[r[1]] = nid
sql(f"insert into grade_rules (course_id,scheme_id,kind,applicable,params,source_quote,source_page,filled_by_user) select '{C3}','{sid}',kind,applicable,params,source_quote,source_page,filled_by_user from grade_rules where scheme_id=(select id from grade_schemes where course_id='{C1}' and status='CONFIRMED' limit 1)")
items = {}
def item(comp, key, name, pos):
    i = sql(f"insert into grade_items (course_id,scheme_id,component_id,key,name,max_score,source,position,published_at) values ('{C3}','{sid}','{cmap[comp]}','{key}','{name}',10,'MANUAL',{pos},now()) returning id")[0][0]; items[key] = i
item("midterm", "midterm_1", "Giữa kỳ lần 1", 1); item("midterm", "midterm_2", "Giữa kỳ lần 2", 2); item("weekly", "weekly", "Kiểm tra hằng tuần", 3); item("final", "final", "Thi cuối kỳ", 4)
for k, v in (("midterm_1", 7), ("midterm_2", 9), ("weekly", 8), ("final", 7.5)):
    sql(f"insert into grade_entries (course_id,item_id,student_id,score,source,entered_by) values ('{C3}','{items[k]}','{S}',{v},'MANUAL','{T}')")
print("QCT3 scheme", sid, "items", list(items))
c, j, _, _ = api("sv31", "GET", f"/courses/{C3}/me/grades"); print(c, [ (l["label"], l.get("detail")) for l in (j.get("calculation") or [])])
