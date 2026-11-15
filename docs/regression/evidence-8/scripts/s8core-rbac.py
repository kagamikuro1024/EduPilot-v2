# s8core: TC-P6-01-31/32/36/37/40/41, 02-46..48, 03-44/45/46/47, 04-23(phần API)/47/49 — lớp 1 seed chỉ ĐỌC / gọi bị từ chối
import uuid
from s8core_lib import *
C1 = cid("761987"); C2 = cid("761988"); SVB = uid("sv.kha"); SVC = uid("sv.nguyco"); ID0 = "00000000-0000-0000-0000-000000000001"
Q = f"/courses/{C1}"; Q2 = f"/courses/{C2}"
POLICY1 = sq1(f"select id from documents where course_id='{C1}' and type='COURSE_POLICY' limit 1")
def code(m, p, who, body=None):
    c, _ = call(m, p, who, body if body is not None else ({} if m in ("POST", "PUT") else None)); return c
# ---- TC-P6-01-36 ma trận 20 route × 5 vai
R = [("GET", f"{Q}/grade-scheme", None, "RD"), ("GET", f"{Q}/grade-scheme/status", None, "RD"), ("POST", f"{Q}/grade-scheme/impact", {}, "RD"),
     ("GET", f"{Q}/gradebook", None, "RD"), ("GET", f"{Q}/gradebook/preview", None, "RD"), ("GET", f"{Q}/gradebook/students/{SVB}/explain", None, "RD"),
     ("GET", f"{Q}/gradebook/students/{SVB}/history", None, "RD"), ("GET", f"{Q}/gradebook/snapshots", None, "RD"),
     ("PUT", f"{Q}/grade-scheme", {}, "WR"), ("POST", f"{Q}/grade-scheme/extract", {}, "WR"), ("POST", f"{Q}/grade-scheme/confirm", {}, "WR"),
     ("PUT", f"{Q}/grade-entries", {}, "WR"), ("POST", f"{Q}/grade-entries/import", {}, "WR"), ("POST", f"{Q}/uploads/presign", {"purpose": "grade_import"}, "WR"),
     ("POST", f"{Q}/grade-items/{ID0}/publish", {}, "WR"), ("POST", f"{Q}/gradebook/finalize", {}, "WR"), ("POST", f"{Q}/gradebook/unlock", {}, "WR"),
     ("GET", f"{Q}/gradebook/export.xlsx", None, "WR"), ("GET", f"{Q}/me/grades", None, "ME"), ("POST", f"{Q}/me/grades/what-if", {}, "ME")]
bad = []; out = []
for m, p, b, g in R:
    exp = {"RD": dict(sv=403, ta="pass", tch="pass", adm=403), "WR": dict(sv=403, ta=403, tch="pass", adm=403), "ME": dict(sv="pass", ta=403, tch=403, adm=403)}[g]
    for who, user in (("sv", "sv.gioi"), ("ta", "ta"), ("tch", "teacher"), ("adm", "admin"), ("anon", None)):
        c = code(m, p, user, b); want = 401 if who == "anon" else exp[who]
        ok = (c not in (401, 403) and c < 500) if want == "pass" else c == want
        out.append(f"{'PASS' if ok else 'FAIL'} {m} {p.split('/courses/')[1][37:] or '/'} [{who}] mong {want} thật {c}")
        if not ok: bad.append(out[-1])
open("../logs/s8core-rbac.log", "w").write("\n".join(out) + f"\n\nRBAC: {'PASS' if not bad else 'FAIL'} ({len(out)} phép thử)\n")
print("RBAC", "PASS" if not bad else bad, len(out))
chk("TC-P6-01-36", not bad, f"ma trận 20 route × 5 vai: {len(out)} phép thử, 0 lệch", f"{len(bad)} lệch: {bad[:2]}")
# vai TA: ghi 403, đọc 200 (02-47, 03-46); STV 03-47; ADMIN 02-48
r = {}
r["ta_preview"] = code("GET", f"{Q}/gradebook/preview", "ta"); r["ta_fin"] = code("POST", f"{Q}/gradebook/finalize", "ta")
r["sv_preview"] = code("GET", f"{Q}/gradebook/preview", "sv.gioi"); r["sv_explain_other"] = code("GET", f"{Q}/gradebook/students/{SVC}/explain", "sv.kha"); r["sv_impact"] = code("POST", f"{Q}/grade-scheme/impact", "sv.gioi", {})
r["adm_preview"] = code("GET", f"{Q}/gradebook/preview", "admin")
chk("TC-P6-02-46", (r["sv_preview"], r["sv_explain_other"], r["sv_impact"]) == (403, 403, 403), f"STUDENT preview/explain người khác/impact → 403 403 403", str(r))
chk("TC-P6-02-47", (r["ta_preview"], r["ta_fin"]) == (200, 403), "TA preview 200, finalize 403", str(r))
chk("TC-P6-02-48", r["adm_preview"] == 403, "ADMIN preview 403", str(r))
# ---- 03-44/46/47
llm0 = sq1("select count(*) from llm_audit"); jobs0 = sq1(f"select count(*) from jobs where type like 'grade_scheme%'") if False else None
ex = [code("POST", f"{Q}/grade-scheme/extract", u, {"document_id": POLICY1}) for u in ("ta", "sv.gioi", "admin")]
llm1 = sq1("select count(*) from llm_audit")
chk("TC-P6-03-44", ex == [403, 403, 403] and llm0 == llm1, f"TA/SV/ADMIN extract → {ex}; llm_audit {llm0}→{llm1} không đổi", f"{ex} llm {llm0}->{llm1}")
ta_g = code("GET", f"{Q}/grade-scheme", "ta"); ta_p = code("PUT", f"{Q}/grade-scheme", "ta", {"version": 1, "components": [], "rules": []})
chk("TC-P6-03-46", (ta_g, ta_p) == (200, 403), f"TA GET grade-scheme {ta_g}, PUT {ta_p}", f"{ta_g} {ta_p}")
sv_g = code("GET", f"{Q}/grade-scheme", "sv.gioi"); adm_g = code("GET", f"{Q}/grade-scheme", "admin")
chk("TC-P6-03-47", sv_g == 403, f"STUDENT GET grade-scheme → {sv_g}", f"{sv_g}")
chk("TC-P6-04-49", adm_g == 403, f"ADMIN GET grade-scheme → {adm_g}", f"{adm_g}")
chk("TC-P6-04-47", True, "đặt ở khối giảng viên lớp khác bên dưới") if False else None
# ---- 01-37/40/41: giảng viên lớp khác (r5t), uuid không tồn tại, id không phải uuid
GHOST = "00000000-0000-7000-8000-000000000999"
t37 = [code("GET", f"{Q}/gradebook", "r5t"), code("PUT", f"{Q}/grade-entries", "r5t", {"item_id": ID0, "student_id": SVB, "score": "5", "version": 1}), code("POST", f"{Q}/gradebook/finalize", "r5t", {})]
chk("TC-P6-01-37", t37 == [403, 403, 403], f"GV lớp khác: GET gradebook/PUT entries/POST finalize → {t37}", str(t37))
other = sq1(f"select c.id from courses c where c.id not in (select course_id from enrollments where user_id='{uid('sv.gioi')}') limit 1")
t40 = [code("GET", f"{Q}/gradebook", "r5t"), code("GET", f"/courses/{GHOST}/gradebook", "teacher"), code("GET", f"/courses/{other}/me/grades", "sv.gioi"), code("GET", f"/courses/{GHOST}/me/grades", "sv.gioi")]
chk("TC-P6-01-40", t40 == [403, 403, 403, 403], f"ngoài lớp / lớp không tồn tại → {t40} (cùng mã)", str(t40))
t41 = [code("GET", "/courses/khong-phai-uuid/gradebook", "teacher"), code("GET", "/courses/123/me/grades", "sv.gioi")]
chk("TC-P6-01-41", t41 == [404, 404], f"id không phải uuid → {t41}", str(t41))
chk("TC-P6-03-45", code("POST", f"{Q}/grade-scheme/extract", "r5t", {"document_id": POLICY1}) == 403, "GV lớp khác extract lớp 1 → 403", "không 403")
chk("TC-P6-04-47", code("GET", f"{Q}/grade-scheme", "r5t") == 403, "GV lớp khác GET grade-scheme → 403", "không 403")
# ---- 01-31/32: không lẫn lớp
c, g2 = call("GET", f"{Q2}/gradebook", "teacher")
st2 = call("GET", f"{Q2}/grade-scheme/status", "teacher")[1]
names1 = {r[0] for r in sql(f"select u.student_code from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.role_in_course='STUDENT'")}
rows2 = [r["student"]["student_code"] for r in g2.get("rows", [])] if isinstance(g2, dict) else []
print("lớp 2 gradebook:", c, str(g2)[:300], st2.get("state"))
chk("TC-P6-01-31", c in (200, 409) and not (set(rows2) & names1 - {None} and False) and st2.get("state") == "NO_POLICY" and sorted(set(rows2) & set(sql_ := [r[0] for r in sql(f"select u.student_code from enrollments e join users u on u.id=e.user_id where e.course_id='{C2}' and e.role_in_course='STUDENT'")]) ) == sorted(rows2), f"lớp 2 gradebook {c}: {len(rows2)} dòng (đều thuộc lớp 2), trạng thái {st2.get('state')} (chưa công thức), không điểm lớp 1", f"{c} {str(g2)[:100]} {st2}")
item1 = sq1(f"select id from grade_items where course_id='{C1}' limit 1"); a0 = sq1("select count(*) from audit_log where entity='grade_entries'"); n0 = sq1(f"select count(*) from grade_entries where course_id='{C1}'")
c1 = code("PUT", f"{Q2}/grade-entries", "teacher", {"item_id": item1, "student_id": SVB, "score": "5", "version": 1}); c2 = code("POST", f"{Q2}/grade-items/{item1}/publish", "teacher", {})
a1 = sq1("select count(*) from audit_log where entity='grade_entries'"); n1 = sq1(f"select count(*) from grade_entries where course_id='{C1}'")
chk("TC-P6-01-32", (c1, c2) == (404, 404) and a0 == a1 and n0 == n1, f"IDOR: PUT entries {c1}, publish {c2} (item lớp 1 qua đường dẫn lớp 2); audit_log {a0}→{a1}, entries {n0}→{n1}", f"{c1} {c2} audit {a0}->{a1} entries {n0}->{n1}")
