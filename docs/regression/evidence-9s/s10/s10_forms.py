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
G = {r[0]: r[1] for r in sql(f"select key,id from grade_items where course_id='{C1}'")}; G1, G2 = G["g1"], G["g2"]
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
def entries(item): return {r[0]: r[1] for r in sql(f"select u.email, e.score from grade_entries e join users u on u.id=e.student_id where e.item_id='{item}' and e.source='QUIZ'")}
T0 = sql("select now()")[0][0]
import time as _t
RUN = _t.strftime("%Y-%m-%d %H:%M", _t.gmtime())   # dấu thời gian riêng mỗi lần chạy (tránh replay theo băm tệp)
if __name__ == "__main__":
    # TC-11-02 hai kiểu ánh xạ
    data = mkcsv([row(STU[0][0], STU[0][1], RUN+":05", 3)]); c, pre = presign("teacher", "f02.csv", data)
    c1, pv, _ = preview(pre)
    mpm = {"header_row": 1, "email_col": "Email", "code_col": "Mã số sinh viên", "questions": [{"col": f"Câu {i+1}", "accepted": [ans(DD[i], True)]} for i in range(5)]}
    c2, pv2, _ = preview(pre, mp=mpm, src="MANUAL")
    chk("TC-11-02", c1 == 200 and c2 == 200 and "result" in pv and "result" in pv2, f"BANK {c1} {len(pv.get('result', {}).get('items', []))} khớp; MANUAL (accepted nhập tay) {c2} {len(pv2.get('result', {}).get('items', []))} khớp")
    # TC-11-04 tính tay ≥ 5 SV + làm tròn
    R = [row(STU[i][0], STU[i][1], f"{RUN}:1{i}", i) for i in range(6)]
    c, pre = presign("teacher", "f04.csv", mkcsv(R)); c1, pv, _ = preview(pre); cc, j, _ = confirm(pre); got = entries(G1)
    exp = {STU[i][0]: i * 2.0 for i in range(6)}  # k đúng/5 × 10 = 2k
    okk = cc in (200, 201) and all(abs(float(got.get(e, -9)) - v) < 0.005 for e, v in exp.items())
    chk("TC-11-04", okk, f"6 SV: tính tay k/5×10 = {list(exp.values())}; ghi vào grade_entries(QUIZ) = {[got.get(e) for e in exp]} (cc={cc})")
    # làm tròn 3 câu: 1/3, 2/3
    mp3 = dict(MPB, questions=MPB["questions"][:3]); R3 = [row(STU[6][0], STU[6][1], RUN+":02", 1), row(STU[7][0], STU[7][1], RUN+":02", 2)]
    c, pre3 = presign("teacher", "f04b.csv", mkcsv(R3)); preview(pre3, item=G2, mp=mp3); cc3, j3, _ = confirm(pre3, item=G2, mp=mp3); g2 = entries(G2)
    chk("TC-11-04 làm tròn", cc3 in (200, 201) and abs(float(g2[STU[6][0]]) - 3.33) < 0.005 and abs(float(g2[STU[7][0]]) - 6.67) < 0.005, f"1/3×10 → {g2.get(STU[6][0])}, 2/3×10 → {g2.get(STU[7][0])} (kỳ vọng 3,33 và 6,67 decimal)")
    # TC-11-05 khớp danh tính
    n_users = sql("select count(*) from users")[0][0]; n_enr = sql("select count(*) from enrollments")[0][0]
    SV36 = [x for x in sql(f"select u.email,u.student_code from users u where email='sv36@edupilot.local'")][0]
    R = [row(STU[0][0], "", RUN+":01", 5), row("", STU[1][1], RUN+":01", 5), row("", "99999999", RUN+":01", 5), row(STU[2][0], STU[3][1], RUN+":01", 5), row("khong.ton.tai@x.vn", "", RUN+":01", 5), row("", SV36[1], RUN+":01", 5)]
    c, pre = presign("teacher", "f05.csv", mkcsv(R)); c1, pv, _ = preview(pre); res = pv["result"]
    why = [(u["reason"]) for u in res["unmatched_rows"]]
    chk("TC-11-05", res["matched"] == 2 and why.count("MISMATCH") == 1 and why.count("NOT_FOUND") == 3 and sql("select count(*) from users")[0][0] == n_users and sql("select count(*) from enrollments")[0][0] == n_enr, f"khớp {res['matched']} (email đã xác minh + MSSV danh sách); không khớp: {why} (email đã xác minh + MSSV của SV khác trong danh sách → MISMATCH; MSSV không có / email lạ / MSSV tự khai của SV vào lớp bằng mã (joined_via=CODE) → NOT_FOUND); số tài khoản/ghi danh không đổi")
    # TC-11-06 dòng trùng
    R = [row(STU[4][0], STU[4][1], "2026-10-01 09:00:00", 0), row(STU[4][0], STU[4][1], RUN+":06", 4)]
    c, pre = presign("teacher", "f06.csv", mkcsv(R)); c1, pv, _ = preview(pre); res = pv["result"]; it = [x for x in res["items"] if x["student"]["student_code"] == STU[4][1]]
    chk("TC-11-06", len(it) == 1 and it[0]["correct"] == 4 and res["duplicates"] >= 1, f"dòng mới hơn (4/5 đúng) thắng: {[ (x['correct'], x['score']) for x in it]}, duplicates={res['duplicates']}")
    # TC-11-10 đáp án không khớp
    R = [row(STU[5][0], STU[5][1], RUN+":01", 5)]; R[0][5] = "không phải đáp án nào"
    c, pre = presign("teacher", "f10.csv", mkcsv(R)); c1, pv, _ = preview(pre); it = pv["result"]["items"][0]
    chk("TC-11-10", it["correct"] == 4 and it["mismatch"] == 1, f"ô không khớp phương án nào: đúng {it['correct']}/{it['total']}, mismatch={it['mismatch']} (không đoán)")
    # TC-11-16 SV chỉ thấy khi công bố
    def sv_has(item_name):
        c, j, _ = api("sv.nguyco", "GET", f"/courses/{C1}/me/grades"); return item_name in json.dumps(j, ensure_ascii=False), j
    sql(f"update grade_items set published_at=null where id='{G1}'")
    uid_ = sql("select id from users where email='sv.nguyco@edupilot.local'")[0][0]
    sql(f"delete from grade_entries where item_id='{G1}' and student_id='{uid_}'")
    R = [row("sv.nguyco@edupilot.local", "", RUN+":03", 5)]; c, pre = presign("teacher", "f16.csv", mkcsv(R)); preview(pre); confirm(pre)
    c0, j0, _ = api("sv.nguyco", "GET", f"/courses/{C1}/me/grades"); before = json.dumps(j0, ensure_ascii=False)
    cp, jp, _ = api("teacher", "POST", f"{P}/grade-items/{G1}/publish", {"published": True})
    c1_, j1, _ = api("sv.nguyco", "GET", f"/courses/{C1}/me/grades"); after = json.dumps(j1, ensure_ascii=False)
    chk("TC-11-16", c0 == 200 and cp in (200, 201) and before != after and "Forms G1" not in before, f"trước công bố me/grades không đổi theo điểm Forms; publish (route cũ) → {cp}; sau công bố me/grades KHÁC trước: {before != after}; trích: {[l for l in (j1.get('calculation') or []) if 'Forms' in json.dumps(l, ensure_ascii=False) or l.get('value') in ('10','10.00',10)][:2]}")
    # TC-11-17 audit_log + TC-11-13 blob/log
    al = sql(f"select action, entity, left(after::text,300) from audit_log where course_id='{C1}' and created_at > '{T0}' and (action ilike '%forms%' or entity ilike '%forms%') order by created_at desc limit 3")
    chk("TC-11-17", len(al) >= 1, f"audit_log: {al[:2]}")
    # TC-11-14 llm_audit
    nl = sql(f"select count(*) from llm_audit where created_at > '{T0}' and course_id='{C1}'")[0][0]
    chk("TC-11-14 (DB)", nl == "0", f"llm_audit của lớp trong cả lô nhập Forms: {nl} dòng")
    # TC-11-15 sổ khoá
    c, pre = presign("teacher", "f15.csv", mkcsv([row(STU[0][0], STU[0][1], RUN+":04", 5)])); preview(pre)
    sql(f"update gradebooks set status='LOCKED', snapshot_version=1 where course_id='{C1}'")
    cL, jL, _ = confirm(pre)
    cU, jU, _ = api("teacher", "POST", f"{P}/gradebook/unlock", {"reason": "QC smoke 10: mở khoá để nhập Forms"}); cc2, j2, _ = confirm(pre)
    chk("TC-11-15", cL == 409 and cc2 in (200, 201), f"sổ LOCKED: import → {cL} {(jL or {}).get('code')}; mở khoá có lý do ({cU}) → import {cc2}")
    print("done forms")
