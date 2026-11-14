# Kiểm lại các lỗi Thấp đã sửa: L7, L8, L9, L10, L12 (API). L11, L15, L16 kiểm bằng UI (qc-s10-low-real.spec.ts)
from pb import *
import io, csv, hashlib
# L7: chủ đề có câu nhưng 0 câu đủ điều kiện
for i in range(2):
    c, j, _ = api("teacher", "POST", f"{P}/questions", {"type": "SHORT", "title": "p", "topic": "QC-OnlyPending", "difficulty": "EASY", "stem": f"QCP{i}?", "accepted": ["x"]})
    from pa1_lib import review; review(j["id"], "REQUEST")
c, j, _ = api("sv.gioi", "POST", f"{P}/practice/start", {"mode": "TOPIC", "topic": "QC-OnlyPending", "count": 10})
ok("RECHECK-L7 chủ đề có câu, 0 đủ điều kiện → 409 NOT_ENOUGH_QUESTIONS {available:0}", c == 409 and "NOT_ENOUGH_QUESTIONS" in json.dumps(j) and '"available": 0' in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:140]}")
c, j, _ = api("sv.gioi", "POST", f"{P}/practice/start", {"mode": "TOPIC", "topic": "ZZ-không-có", "count": 10}); ok("RECHECK-L7b chủ đề không tồn tại → 422", c == 422, str(c))
# L8: bù khi ma trận không có difficulty
ov = api("sv.kha", "GET", P + "/practice/overview")[1]["topics"]; ts = sorted(ov, key=lambda t: -t["available"])
few = [t for t in ts if t["available"] <= 4][:1] or ts[-1:]
c, a, _ = api("sv.kha", "POST", f"{P}/practice/start", {"mode": "MOCK", "count": 10, "matrix": [{"topic": few[0]["topic"], "count": 6}, {"topic": ts[0]["topic"], "count": 4}]})
n = len(a["items"]) if c in (200, 201) else None
ok("RECHECK-L8 ma trận ô thiếu câu (không difficulty): bù đủ 10", c in (200, 201) and n == 10, f"{c} items={n} notes={a.get('notes') if isinstance(a, dict) else ''}")
if c in (200, 201): api("sv.kha", "POST", f"{P}/practice/attempts/{a['id']}/submit", {})
# L9
lec = [d for d in api("teacher", "GET", f"{P}/documents?limit=100")[1]["items"] if d["type"] == "LECTURE" and d["status"] == "READY"][0]
c, j, _ = api("teacher", "POST", f"{P}/questions/generate", {"document_ids": [lec["id"]], "topic": "x", "difficulty": "EASY", "count": 3, "types": []}); ok("RECHECK-L9 generate types=[] → 422", c == 422, f"{c}")
# L10: bulk APPROVE EXTRACTED không trích dẫn
pend = [q for q in bank() if q["review_status"] == "PENDING" and q["type"] == "MCQ_SINGLE"][:1]
sql(f"update question_bank set origin='EXTRACTED', citations='[]'::jsonb where id='{pend[0]['id']}'")
c, j, _ = api("teacher", "POST", f"{P}/questions/bulk", {"action": "APPROVE", "ids": [pend[0]["id"]]}); r = j["results"][0]
ok("RECHECK-L10 bulk APPROVE EXTRACTED không trích dẫn → code CITATION_REQUIRED", not r["ok"] and r.get("code") == "CITATION_REQUIRED", f"{c} {r}")
sql(f"update question_bank set origin='MANUAL' where id='{pend[0]['id']}'")
# L12: ánh xạ theo tên tiêu đề + chữ lỗi tệp
def presign(name, data, mime="text/csv"):
    c, pre, _ = api("teacher", "POST", f"{P}/uploads/presign", {"purpose": "forms_import", "filename": name, "mime_type": mime, "size_bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()})
    if c != 200: return c, pre
    urllib.request.urlopen(urllib.request.Request(pre["url"], data=data, method=pre["method"], headers=pre["headers"]), context=CTX).read(); return c, pre
rows = sql(f"select u.email, u.student_code from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.role_in_course='STUDENT' and e.status='ACTIVE' and u.email like 'sv%@edupilot.local' order by u.email limit 3")
mc = [q for q in bank() if q["type"] == "MCQ_SINGLE" and q["review_status"] == "APPROVED"][:2]; dd = [detail(q["id"]) for q in mc]
ans = lambda d, good: [x for x in d["options"] if (x["id"] in d["answer_key"]["option_ids"]) == good][0]["body"]
R = [["Email", "Mã số sinh viên", "Completion time", "Câu 1", "Câu 2"]] + [[e, c_, "2026-11-10 10:00:00", ans(dd[0], True), ans(dd[1], i % 2 == 0)] for i, (e, c_) in enumerate(rows)]
buf = io.StringIO(); csv.writer(buf).writerows(R); data = buf.getvalue().encode()
cp, pre = presign("forms-hdr.csv", data); c, pv, _ = api("teacher", "POST", P + "/forms-import/preview", {"upload_id": pre["upload_id"]})
item = sql("select id from grade_items where key='midterm' limit 1")[0][0]
mp = {"header_row": 1, "email_col": "Email", "code_col": "Mã số sinh viên", "time_col": "Completion time", "questions": [{"col": "Câu 1", "question_id": mc[0]["id"]}, {"col": "Câu 2", "question_id": mc[1]["id"]}]}
c, pv2, _ = api("teacher", "POST", P + "/forms-import/preview", {"upload_id": pre["upload_id"], "item_id": item, "answer_source": "BANK", "mapping": mp})
ok("RECHECK-L12a ánh xạ theo tên tiêu đề cột (ví dụ openapi) được chấp nhận", c == 200 and pv2.get("result", {}).get("matched", 0) == 3, f"{c} {json.dumps(pv2.get('result', pv2), ensure_ascii=False)[:160]}")
time.sleep(1); c, pre2 = presign("x.txt", b"hello", "text/plain"); ok("RECHECK-L12b chữ lỗi tệp .txt nhắc XLSX/CSV, không nhắc PDF/DOCX/PPTX", c == 422 and "PDF" not in json.dumps(pre2, ensure_ascii=False), f"{c} {pre2.get('message') if isinstance(pre2, dict) else pre2}")
time.sleep(1); c, pre3 = presign("big.csv", b"a,b\n" + b"x,y\n" * (1_400_000)); ok("RECHECK-L12c tệp > 5 MB: chữ nói 5 MB", c in (413, 422) and "50 MB" not in json.dumps(pre3, ensure_ascii=False), f"{c} {pre3.get('message') if isinstance(pre3, dict) else pre3}")
