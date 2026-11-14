from pb import *
import io, csv
def presign(user, name, data, mime="text/csv", C=C1):
    sha = hashlib.sha256(data).hexdigest()
    c, pre, _ = api(user, "POST", f"/courses/{C}/uploads/presign", {"purpose": "forms_import", "filename": name, "mime_type": mime, "size_bytes": len(data), "sha256": sha})
    if c != 200: return c, pre
    r = urllib.request.Request(pre["url"], data=data, method=pre["method"], headers=pre["headers"]); urllib.request.urlopen(r, context=CTX).read()
    return c, pre
rows = sql(f"select u.email, u.student_code from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.role_in_course='STUDENT' and e.status='ACTIVE' and u.email like 'sv%@edupilot.local' order by u.email limit 8")
print(rows)
mc = [q for q in BANK if q["type"] == "MCQ_SINGLE" and q["review_status"] == "APPROVED"][:5]
dd = [detail(q["id"]) for q in mc]
def ans(d, ok_):
    ids = d["answer_key"]["option_ids"]; o = [x for x in d["options"] if (x["id"] in ids) == ok_][0]; return o["body"]
hdr = ["Email", "Mã số sinh viên", "Completion time"] + [f"Câu {i+1}" for i in range(5)]
def mkrow(email, code, ts, nok): return [email, code, ts] + [ans(d, i < nok) for i, d in enumerate(dd)]
R = [hdr]
exp = {}
for i, (em, code) in enumerate(rows[:6]): R.append(mkrow(em, code, f"2026-11-1{i} 10:0{i}:00", i)); exp[em] = i
R.append(mkrow("", rows[6][1], "2026-11-13 09:00:00", 5)); exp["MSSV-only:" + rows[6][1]] = 5
R.append(mkrow("", "99999999", "2026-11-13 09:00:00", 5))  # MSSV không có
R.append(mkrow("khong.ton.tai@x.vn", "", "2026-11-13 09:00:00", 5))  # email lạ
R.append(mkrow(rows[0][0], rows[1][1], "2026-11-13 09:00:00", 5))  # email A + MSSV B → MISMATCH
R.append(mkrow(rows[2][0], rows[2][1], "2026-11-01 09:00:00", 0))  # trùng sv: dòng cũ (điểm thấp) – dòng mới nhất (i=2 → 2 đúng) phải thắng
R[1 + 5][3] = "không phải đáp án nào"  # ô không khớp
buf = io.StringIO(); csv.writer(buf).writerows(R); data = buf.getvalue().encode()
G = [c for c in api("teacher", "GET", P + "/gradebook")[1]["columns"] if c["source"] == "MANUAL"]; midterm = G[0]; print("cột", midterm["name"], midterm["published"])
c, pre = presign("teacher", "forms-ok.csv", data); ok("TC-11-01a presign forms_import (TEACHER) 200", c == 200, f"{c}")
mp = {"header_row": 1, "email_col": "Email", "code_col": "Mã số sinh viên", "time_col": "Completion time", "questions": [{"col": f"Câu {i+1}", "question_id": mc[i]["id"]} for i in range(5)]}
c, pv, _ = api("teacher", "POST", P + "/forms-import/preview", {"upload_id": pre["upload_id"]}); print("PREVIEW", c, json.dumps(pv, ensure_ascii=False)[:400])
ok("TC-11-01 preview: cột + 20 dòng đầu, không ghi", c == 200, f"{c} keys={list(pv) if isinstance(pv, dict) else ''}")
ge0 = sql("select count(*) from grade_entries")[0][0]
# TC-11-12: tệp khác chưa preview → 409
c2, pre2 = presign("teacher", "forms-nopreview.csv", data + b"\n")
c, j, _ = api("teacher", "POST", P + "/forms-import", {"upload_id": pre2["upload_id"], "item_id": midterm["item_id"], "answer_source": "BANK", "mapping": mp}); ok("TC-11-12 chưa preview → 409 PREVIEW_REQUIRED", c == 409 and "PREVIEW_REQUIRED" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
# preview kết quả (có mapping)
c, pv2, _ = api("teacher", "POST", P + "/forms-import/preview", {"upload_id": pre["upload_id"], "item_id": midterm["item_id"], "answer_source": "BANK", "mapping": mp}); print("PREVIEW2", c, json.dumps(pv2, ensure_ascii=False)[:900])
ok("TC-11-08 Xem trước kết quả: điểm + đúng/tổng + chưa khớp, chưa ghi", c == 200 and sql("select count(*) from grade_entries")[0][0] == ge0, f"{c}")
mp = {"header_row": 1, "email_col": "A", "code_col": "B", "time_col": "C", "questions": [{"col": "DEFGH"[i], "question_id": mc[i]["id"]} for i in range(5)]}
c, pv2, _ = api("teacher", "POST", P + "/forms-import/preview", {"upload_id": pre["upload_id"], "item_id": midterm["item_id"], "answer_source": "BANK", "mapping": mp}); print("PREVIEW3", c, {k:(v if not isinstance(v,(list,dict)) else len(v)) for k,v in pv2.items()})
for k,v in pv2.items():
    if k not in ("columns","rows"): print("  ",k, json.dumps(v,ensure_ascii=False)[:700])
res = pv2["result"]; 
print("ITEMS", [(i["student"]["student_code"], i["correct"], i["total"], i["score"], i["mismatch"]) for i in res["items"]])
print("OTHER", {k: v for k, v in res.items() if k != "items"})
before = sql(f"select count(*) from grade_entries where item_id='{midterm['item_id']}' and source='QUIZ'")[0][0]
pub0 = sql(f"select published_at from grade_items where id='{midterm['item_id']}'")[0][0]
k = str(uuid.uuid4())
c, j, _ = api("teacher", "POST", P + "/forms-import", {"upload_id": pre["upload_id"], "item_id": midterm["item_id"], "answer_source": "BANK", "mapping": mp}, idem=k)
print("CONFIRM", c, json.dumps(j, ensure_ascii=False)[:500])
after = sql(f"select count(*) from grade_entries where item_id='{midterm['item_id']}' and source='QUIZ'")[0][0]
pub1 = sql(f"select published_at from grade_items where id='{midterm['item_id']}'")[0][0]
ok("TC-11-03 confirm ghi grade_entries(source=QUIZ), không đổi published_at", c in (200, 201) and int(after) > int(before) and pub0 == pub1, f"{c} QUIZ entries {before}→{after}; published_at {pub0}→{pub1}")
c, j2, _ = api("teacher", "POST", P + "/forms-import", {"upload_id": pre["upload_id"], "item_id": midterm["item_id"], "answer_source": "BANK", "mapping": mp}, idem=str(uuid.uuid4()))
ok("TC-11-07 nhập lại cùng tệp/cột (khoá mới) → replayed, không ghi thêm", c in (200, 201) and j2.get("replayed") is True and sql(f"select count(*) from grade_entries where item_id='{midterm['item_id']}' and source='QUIZ'")[0][0] == after, f"{c} replayed={j2.get('replayed')}")
for u in ["ta", "sv.kha", "admin"]:
    cp, pr = presign(u, "x.csv", data)
    ok(f"TC-11-11 {u} presign forms_import → 403", cp == 403, f"{cp}")
    c1, _, _ = api(u, "POST", P + "/forms-import/preview", {"upload_id": pre["upload_id"]}); c2, _, _ = api(u, "POST", P + "/forms-import", {"upload_id": pre["upload_id"], "item_id": midterm["item_id"], "answer_source": "BANK", "mapping": mp})
    ok(f"TC-11-11 {u} preview/confirm → 403", c1 == 403 and c2 == 403, f"{c1}/{c2}")
# tệp xấu
bad = [("forms-bad.txt", b"hello\x00\x01", "text/plain"), ("forms-nocol.csv", "Họ tên,Điểm\nA,1\n".encode(), "text/csv")]
for nm, b, mt in bad:
    cp, pr = presign("teacher", nm, b, mt)
    if cp != 200: ok(f"TC-11-09 {nm} bị từ chối ngay ở presign", True, f"presign {cp}"); continue
    c, j, _ = api("teacher", "POST", P + "/forms-import/preview", {"upload_id": pr["upload_id"]}); 
    ok(f"TC-11-09 {nm} → 422 kèm vị trí", c == 422, f"{c} {json.dumps(j, ensure_ascii=False)[:160]}")
    if nm == "forms-nocol.csv":
        c, j, _ = api("teacher", "POST", P + "/forms-import/preview", {"upload_id": pr["upload_id"], "item_id": midterm["item_id"], "answer_source": "BANK", "mapping": {"header_row": 1, "email_col": "A", "questions": [{"col": "B", "question_id": mc[0]["id"]}]}}); print("NOCOL", c, json.dumps(j, ensure_ascii=False)[:200])
big = "Email,Cau 1\n".encode() + b"a@b.c,x\n" * 700000
cp, pr = presign("teacher", "forms-big.csv", big[: 5 * 1024 * 1024 + 10]); ok("TC-11-09 tệp > 5 MB bị từ chối ở presign", cp in (413, 422, 400), f"{cp}")
