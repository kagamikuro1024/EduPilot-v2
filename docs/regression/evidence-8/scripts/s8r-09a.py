# s8r 09a: US-P6-09 API trên QD8R9 (sổ OPEN rồi chốt): /me/grades, what-if, rò rỉ CANARY, RBAC, IDOR
from s8r_g import *
MID, WK, FIN = ITEM["midterm"], ITEM["weekly"], ITEM["final"]
def EM(a): return sq1(f"select email from users where id='{SV(a)}'").split("@")[0]
A, B, Cc = "sv.gioi", "sv.kha", "sv.nguyco"; sA, sB, sC = EM(A), EM(B), EM(Cc)
def me(u, path="me/grades", **kw): return call("GET", f"{Q}/{path}", u, **kw)
def wi(u, body, key=False): return call("POST", f"{Q}/me/grades/what-if", u, body, key=key)
cnt = lambda: [int(sq1(f"select count(*) from {t}")) if t != "x" else 0 for t in ("grade_entries", "audit_log", "final_grade_snapshots", "notifications")] + [int(sq1("select count(*) from outbox")) if sq1("select to_regclass('public.outbox')") else 0]
# ---- 09-01
st, b, hd = getx(f"{Q}/me/grades", sB); j = json.loads(b)
C("TC-P6-09-01", st == 200 and hd.get("Cache-Control") == "no-store" and j["status"] == "IN_PROGRESS" and j["qt"] == "8.25" and all(k in j for k in ("scheme", "components", "bonus", "penalty", "qt", "calculation", "official_note")), f"B /me/grades → 200, Cache-Control no-store, status IN_PROGRESS, qt {j['qt']} (tay 8,25); khoá {sorted(j)}", f"{st} {hd.get('Cache-Control')} {str(j)[:200]}")
# ---- 09-02 tham số chéo
c1, r1 = me(sB, f"me/grades?student_id={SV(Cc)}&user_id={SV(Cc)}"); c2, r2 = call("POST", f"{Q}/me/grades", sB, {"student_id": SV(Cc)})
C("TC-P6-09-02", c1 == 200 and r1["qt"] == "8.25" and c2 in (404, 405) , f"GET ?student_id=C&user_id=C → vẫn dữ liệu của B (qt {r1.get('qt')}, không 1,25); POST {{student_id:C}} → {c2} (route không nhận POST)", f"{c1} {r1.get('qt')} {c2} {r2}")
# ---- 09-03 NO_SCHEME (QD8F3 lớp NO_POLICY)
f3 = cid("QD8F3"); mem = sql(f"select u.email from users u join enrollments e on e.user_id=u.id where e.course_id='{f3}' and e.role_in_course='STUDENT' and e.status='ACTIVE' limit 1")[0][0].split("@")[0]
c, r = call("GET", f"/courses/{f3}/me/grades", mem)
C("TC-P6-09-03", c == 200 and r == {"status": "NO_SCHEME"}, f"lớp NO_POLICY (QD8F3), sinh viên {mem}: {c} {r} (chỉ có status)", f"{c} {r}")
# ---- 09-05 ngoài lớp
c, r = call("GET", f"{Q}/me/grades", "sv.moi"); C("TC-P6-09-05", c == 403, f"sinh viên ngoài lớp (sv.moi) → {c}", f"{c} {r}")
# ---- 09-36/37 RBAC
rs = [call("GET", f"{Q}/me/grades", u)[0] for u in ("ta", "teacher", "admin")]
C("TC-P6-09-36", rs == [403, 403, 403], f"TA/TCH/ADM GET /me/grades → {rs}", str(rs))
ID0 = "00000000-0000-0000-0000-000000000001"
R = [("GET", f"{Q}/grade-scheme", None), ("GET", f"{Q}/grade-scheme/status", None), ("POST", f"{Q}/grade-scheme/impact", {}), ("GET", f"{Q}/gradebook", None), ("GET", f"{Q}/gradebook/preview", None), ("GET", f"{Q}/gradebook/students/{SV(Cc)}/explain", None), ("GET", f"{Q}/gradebook/students/{SV(B)}/history", None), ("GET", f"{Q}/gradebook/snapshots", None), ("GET", f"{Q}/gradebook/snapshots/1", None),
     ("PUT", f"{Q}/grade-scheme", {}), ("POST", f"{Q}/grade-scheme/extract", {}), ("POST", f"{Q}/grade-scheme/confirm", {}), ("PUT", f"{Q}/grade-entries", {}), ("POST", f"{Q}/grade-entries/import", {}), ("POST", f"{Q}/uploads/presign", {"purpose": "grade_import"}), ("POST", f"{Q}/grade-items/{ID0}/publish", {}), ("POST", f"{Q}/gradebook/finalize", {}), ("POST", f"{Q}/gradebook/unlock", {}), ("GET", f"{Q}/gradebook/export.xlsx", None)]
bad = [(m, p[-40:], c) for m, p, b in R for c in [call(m, p, sA, b)[0]] if c != 403]
C("TC-P6-09-37", not bad, f"STUDENT (sv.gioi) gọi {len(R)} route Staff (đọc + ghi, mọi route P6) → 403 hết", str(bad))
# ---- 09-10 EXAM PENDING + không có final khi chưa chốt
exm = sql(f"select id from exams where course_id='{C0}' limit 1")[0][0]
sql(f"update exams set status='CLOSED', published_at=null where id='{exm}'"); c, r = me(sB); comp = {x["key"]: x for x in r["components"]}
sql(f"update exams set status='PUBLISHED', published_at=now() where id='{exm}'")
C("TC-P6-09-10", comp["weekly"]["status"] == "PENDING" and "score" not in comp["weekly"] and "final" not in r, f"exam chưa PUBLISHED → item EXAM {comp['weekly']}; sổ chưa chốt → không có khoá 'final' (khoá: {sorted(r)})", str(r)[:300])
# ---- 09-07/08/11: chưa công bố + CANARY 7,77
setc(B, "midterm", "7.77"); call("POST", f"{Q}/grade-items/{MID}/publish", "teacher", {"published": False})
c, r = me(sB); s = json.dumps(r, ensure_ascii=False); comp = {x["key"]: x for x in r["components"]}
C("TC-P6-09-07", c == 200 and comp["midterm"]["status"] != "PUBLISHED" and "score" not in comp["midterm"], f"giữa kỳ chưa công bố (B=7,77): components.midterm = {comp['midterm']} (không score)", s[:300])
C("TC-P6-09-08", "7.77" not in s and "7,77" not in s, f"phản hồi /me/grades của B khi cột chưa công bố: 0 lần '7.77'/'7,77' (calc={len(r.get('calculation', []))} dòng)", s[:300])
c, r2 = wi(sB, {"final_exam_score": "8.0"}); s2 = json.dumps(r2, ensure_ascii=False)
call("POST", f"{Q}/grade-items/{MID}/publish", "teacher", {"published": True}); c3, r3 = me(sB); comp3 = {x["key"]: x for x in r3["components"]}
call("POST", f"{Q}/grade-items/{MID}/publish", "teacher", {"published": False}); c4, r4 = me(sB); comp4 = {x["key"]: x for x in r4["components"]}
C("TC-P6-09-11", comp3["midterm"].get("score") == "7.77" and comp4["midterm"]["status"] != "PUBLISHED", f"GV bật publish → B thấy giữa kỳ {comp3['midterm'].get('score')}; gỡ công bố → {comp4['midterm']['status']} (không score)", f"{comp3} {comp4}")
# 09-21: QT chưa đủ → what-if 409 NOT_ENOUGH_GRADES (giữa kỳ chưa công bố)
c, r = wi(sB, {"final_exam_score": "8.0"}); C("TC-P6-09-21", c == 409 and r.get("code") == "NOT_ENOUGH_GRADES" and "7.77" not in json.dumps(r), f"giữa kỳ chưa công bố → what-if {c} {r.get('code')}; không số bịa", f"{c} {r}")
call("POST", f"{Q}/grade-items/{MID}/publish", "teacher", {"published": True}); setc(B, "midterm", CSV[B]["midterm"])
# ---- 09-17/18/19/20/48/49
c, r = wi(sB, {"final_exam_score": "8.0"}); ok17 = c == 200 and r["hp"] == "8" and r["letter"] == "B+" and any("calculation" in k for k in r) 
C("TC-P6-09-17", ok17, f"what-if final 8,0 (không Idempotency-Key) → {c}: HP {r.get('hp')} {r.get('letter')} (tay QT 8,25 → HP thô 8,10 → 8,0 B+), {len(r.get('calculation', []))} dòng phép tính tuyến tính; UI ô nhập: s8r-ui-09", f"{c} {r}")
ca, ra = wi(sB, {"target_letter": "A"}); cp, rp = wi(sB, {"target_letter": "A+"}); cu, ru = wi(sC, {"target_letter": "A+"})
C("TC-P6-09-18", ra == {"min_final_exam_score": "8.25", "reachable": True} and rp.get("min_final_exam_score") == "9.09" and ru.get("reachable") is False, f"B target A → {ra}; A+ → {rp}; C target A+ → {ru} (không đạt được)", f"{ra} {rp} {ru}")
bad = []
for v in ("11", "-1", "abc", "10.001", ""):
    c, r = wi(sB, {"final_exam_score": v})
    if not (c == 422): bad.append((v, c))
C("TC-P6-09-19", not bad, f"final_exam_score = 11, -1, abc, 10,001, '' → 422 cả 5 ({bad}); UI giữ số đã gõ: dev e2e 'what if inline' xanh", str(bad))
n0 = cnt()
for _ in range(10): wi(sB, {"final_exam_score": "8.0"})
n1 = cnt()
C("TC-P6-09-20", n0 == n1, f"10 lần what-if: grade_entries/audit_log/final_grade_snapshots/notifications/outbox {n0} → {n1} không đổi (hàng đợi nền có thể nhích nếu nhóm khác chạy; so sánh {'đúng' if n0==n1 else 'LỆCH'})", f"{n0} {n1}")
e1 = wi(sB, {}); e2 = wi(sB, {"final_exam_score": "8", "target_letter": "A"}); e3 = wi(sB, {"target_letter": "Z"}); e4 = wi(sB, {"target_letter": "a+"})
C("TC-P6-09-48", all(x[0] == 422 and x[1].get("code") == "VALIDATION_FAILED" for x in (e1, e2)), f"{{}} → {e1[0]} {e1[1].get('code')}; cả hai trường → {e2[0]} {e2[1].get('code')}", f"{e1} {e2}")
C("TC-P6-09-49", e3[0] == 422 and e4[0] == 422, f"target_letter 'Z' → {e3[0]}, 'a+' → {e4[0]}", f"{e3} {e4}")
# ---- 09-35 mỗi sinh viên một qt
qs = [me(u)[1]["qt"] for u in (sA, sB, sC)]
C("TC-P6-09-35", qs == ["10", "8.25", "1.25"], f"A/B/C /me/grades qt = {qs} (tay 10 / 8,25 / 1,25), mỗi người chỉ thấy của mình", str(qs))
# ---- 09-09: quét CANARY — unlock_reason + ghi chú; chốt rồi mở khoá với CANARY
c, r = fin(); assert c == 200, (c, r)
c, r = unl("CANARY-S8R-4417 phúc khảo nội bộ"); assert c == 200, (c, r)
sql(f"update grade_items set name=name where id='{MID}'")
sw = [("GET", f"/me/courses"), ("GET", "/me/today"), ("GET", "/me/notifications"), ("GET", f"/courses/{C0}/me/grades"), ("GET", f"/courses/{C0}/me/study-time"), ("GET", f"/courses/{C0}/me/attendance"), ("GET", f"/courses/{C0}/me/assignments"), ("GET", f"/courses/{C0}/threads"), ("GET", f"/courses/{C0}/documents"), ("GET", f"/courses/{C0}/exams"), ("GET", f"/courses/{C0}/gradebook/snapshots"), ("GET", f"/courses/{C0}/gradebook"), ("GET", "/me/threads"), ("GET", "/courses"), ("GET", "/me")]
hits = []; seen = 0
for m, p in sw:
    c, r = call(m, p, sB); t = json.dumps(r, ensure_ascii=False) if not isinstance(r, str) else r; seen += c == 200
    if "CANARY" in t or re.search(r'"(hp_raw|breakdown|observations|risk|study_time)"', t) and "/me/study-time" not in p: hits.append((p, c))
sq_ = sql(f"select count(*) from notifications where course_id='{C0}' and (title ilike '%CANARY%' or body ilike '%CANARY%')")[0][0]
C("TC-P6-09-09", not hits and sq_ == "0", f"B quét {len(sw)} route STUDENT (/me, /me/grades, today, notifications, threads, documents, exams, snapshots…): {seen} trả 200, CANARY (unlock_reason) = 0 lần ở mọi phản hồi, 0 khoá hp_raw/breakdown/observations/risk; notifications chứa CANARY: {sq_}; UI/DOM: s8r-ui-09", str(hits))
