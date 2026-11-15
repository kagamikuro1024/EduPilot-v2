# gd_j (QC2 smoke): kiểm DB không cần LLM: G02-02/03/07/09, G10-xx lược đồ. Insert thử phải bị CHECK/UNIQUE chặn; thành công ngoài ý muốn → dọn.
from gdlib import *
def fails(q):
    try: sql(q); return None
    except RuntimeError as e: return str(e).replace("\n", " ")
U = sql("select id from users where email='teacher@edupilot.local'")[0][0]
base = f"insert into grading_guidance_items(course_id,assignment_id,criterion_id,text,direction,support"
e = fails(base + f",status,approved_in_version,retired_in_version,decided_by) values ('{C1}','{A}','c2','x','STRICTER',2,'RETIRED',2,2,'{U}')")
ok("G02-03a retired_in_version <= approved_in_version bị CHECK chặn", e is not None and "check" in e.lower(), e or "chèn được")
e = fails(base + f",status) values ('{C1}','{A}','c2','x','STRICTER',2,'APPROVED')")
ok("G02-03b APPROVED không có decided_by bị CHECK chặn", e is not None and "check" in e.lower(), e or "chèn được")
q = sql(f"select id from question_bank where course_id='{C1}' and review_status<>'REJECTED' limit 1")[0][0]
e = fails(f"update question_bank set reject_reason='MO_HO' where id='{q}'")
ok("G02-03c reject_reason khi review_status<>'REJECTED' bị CHECK chặn", e is not None and "check" in e.lower(), e or "ghi được")
rid = sql(f"select id from grading_results where course_id='{C1}' and run_state='DONE' limit 1")[0][0]
for col, val in (("ai_scores", "'{}'::jsonb"), ("ai_comments", "'{}'::jsonb"), ("guidance_version", "9")):
    e = fails(f"update grading_results set {col}={val} where id='{rid}'")
    ok(f"G02-07 UPDATE {col} của kết quả DONE bị trigger chặn", e is not None, e or "ghi được")
e = fails(f"update grading_results set updated_at=now() where id='{rid}'")
ok("G02-07 UPDATE cột khác vẫn được", e is None, e or "")
r = sql(f"select ai_comments::text from grading_results where run_state='DONE' and ai_comments is not null limit 1")
j = json.loads(r[0][0]) if r else {}
ok("G02-09 ai_comments dạng {criteria:{id:chữ}, overall}", set(j.keys()) >= {"criteria", "overall"} and isinstance(j.get("criteria"), dict), str(list(j.keys())))
n = sql(f"select count(*) from grading_results where ai_comments is null and run_state='DONE'")[0][0]
print("INFO kết quả DONE không có ai_comments:", n)
d = sql("select count(*) from grading_regression_cases")[0][0]
e = fails(f"insert into grading_regression_cases(course_id,kind,source_id,criterion_id,expected) select course_id,kind,source_id,criterion_id,expected from grading_regression_cases limit 1")
ok("G07 UNIQUE (course,kind,source,criterion) chặn trùng", e is not None and ("unique" in e.lower() or "duplicate" in e.lower()), e or "chèn được")
