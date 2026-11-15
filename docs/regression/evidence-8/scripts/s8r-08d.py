# s8r 08d: QD8R81 — chốt thất bại (thiếu điểm, EXAM chưa công bố, NO_POLICY), idempotency, RBAC, biên lý do mở khoá, SV rời lớp, audit fail-closed.
from s8r_g import *
IT = lambda k: ITEM[k]
nsnap = lambda: int(sq1(f"select count(*) from final_grade_snapshots where course_id='{C0}'"))
# ---- 08-06 / 05-16 NO_POLICY
c3 = cid("QD8F3"); c, r = call("POST", f"/courses/{c3}/gradebook/finalize", "gv.s8f", {})
C("TC-P6-08-06", c == 409 and r.get("code") == "SCHEME_NOT_CONFIRMED", f"finalize lớp NO_POLICY (QD8F3) → 409 {r.get('code')}", f"{c} {r}")
# ---- 08-43 thiếu Idempotency-Key
c1, r1 = call("POST", f"{Q}/gradebook/finalize", "teacher", {}, key=False); c2, r2 = call("POST", f"{Q}/gradebook/unlock", "teacher", {"reason": "Lý do đủ dài mười"}, key=False)
C("TC-P6-08-43", c1 in (400, 422) and c2 in (400, 422) and gb()[0] == "OPEN" and nsnap() == 0, f"finalize/unlock thiếu Idempotency-Key → {c1}/{c2} ({r1.get('code')}); sổ vẫn OPEN, 0 snapshot", f"{c1} {r1} {c2} {r2}")
# ---- 08-07 thiếu điểm
s30, v30 = cell("sv30", "final"); s27, v27 = cell("sv27", "final")
put("sv30", "final", None, v30); put("sv27", "final", None, v27)
c, r = fin(); d = r.get("details", {}) if isinstance(r, dict) else {}
ok = c == 409 and r.get("code") == "GRADES_INCOMPLETE" and d.get("total") == 2 and len(d.get("students", [])) == 2 and all(x.get("missing") == ["final"] for x in d["students"]) and nsnap() == 0 and gb()[0] == "OPEN"
C("TC-P6-08-07", ok, f"xoá cuối kỳ sv30, sv27 → finalize 409 GRADES_INCOMPLETE total={d.get('total')} students={json.dumps(d.get('students'))[:120]}; 0 snapshot; gradebooks OPEN", f"{c} {r}")
setc("sv30", "final", s30); setc("sv27", "final", s27)
# ---- 08-45 EXAM chưa công bố
exm = sql(f"select id from exams where course_id='{C0}' limit 1")[0][0]
sql(f"update exams set status='CLOSED', published_at=null where id='{exm}'")
c, r = fin(); ok = c == 409 and r.get("code") == "GRADES_INCOMPLETE" and nsnap() == 0 and r["details"]["total"] > 0
C("TC-P6-08-45", ok, f"đưa bài thi tuần 9 về CLOSED (chưa công bố) → finalize 409 GRADES_INCOMPLETE (total {r.get('details',{}).get('total')}, missing {r.get('details',{}).get('students',[{}])[0].get('missing')}), không phải chốt cả lớp 0", f"{c} {r}")
sql(f"update exams set status='PUBLISHED', published_at=now() where id='{exm}'")
# ---- 08-08 (LOAD1000 ở 08e)
# ---- 08-19/40 RBAC trước chốt (sổ đầy đủ)
rr = {}
for u in ("ta", "sv.gioi", "admin"):
    rr[u] = (call("POST", f"{Q}/gradebook/finalize", u, {})[0], call("POST", f"{Q}/gradebook/unlock", u, {"reason": "Lý do đủ dài mười ký tự"})[0])
oth = (call("POST", f"{Q}/gradebook/finalize", "g7t", {})[0], call("POST", f"{Q}/gradebook/unlock", "g7t", {"reason": "Lý do đủ dài mười ký tự"})[0])
C("TC-P6-08-19", all(v == (403, 403) for v in rr.values()) and oth == (403, 403) and nsnap() == 0, f"TA/SV/ADMIN finalize+unlock → {rr}; GV lớp khác → {oth}; 0 snapshot", f"{rr} {oth}")
# ---- chốt thật rồi 08-40 sau chốt + 08-17 biên
c, r = fin(); assert c == 200, (c, r)
rr2 = {u: call("POST", f"{Q}/gradebook/finalize", u, {})[0] for u in ("ta", "sv.gioi", "admin", "g7t")}
C("TC-P6-08-40", set(rr2.values()) == {403}, f"sau chốt: finalize bởi TA/SV/ADMIN/GV lớp khác → {rr2}", str(rr2))
bad = {}
for name, body in (("thiếu", {}), ("rỗng", {"reason": ""}), ("9 ký tự", {"reason": "a" * 9}), ("501 ký tự", {"reason": "b" * 501}), ("10 dấu cách", {"reason": " " * 10})):
    c, r = call("POST", f"{Q}/gradebook/unlock", "teacher", body)
    if not (c == 422 and r.get("code") == "VALIDATION_FAILED"): bad[name] = (c, r.get("code"))
st = gb()[0]
c10, r10 = unl("c" * 10); ok10 = c10 == 200
c, r = fin(); assert c == 200, (c, r)
c500, r500 = unl("d" * 500); ok500 = c500 == 200
C("TC-P6-08-17", not bad and st == "LOCKED" and ok10 and ok500, f"thiếu/''/9/501/10 dấu cách → 422 VALIDATION_FAILED (5/5), sổ vẫn LOCKED; biên 10 ký tự → {c10}, 500 ký tự → {c500}", f"{bad} {st} {c10} {c500}")
# ---- 08-44 SV rời lớp: đang OPEN (mở bằng 500 ký tự) → sv12 REMOVED → chốt lại
sid12 = SV("sv12"); notif0 = int(sq1(f"select count(*) from notifications where user_id='{sid12}' and course_id='{C0}' and type like 'GRADE%'"))
sql(f"update enrollments set status='REMOVED', removed_at=now() where course_id='{C0}' and user_id='{sid12}'")
c, r = fin(); v = r.get("version")
n12 = int(sq1(f"select count(*) from final_grade_snapshots where course_id='{C0}' and student_id='{sid12}'")); nv = int(sq1(f"select count(*) from final_grade_snapshots where course_id='{C0}' and version={v}"))
nn12 = int(sq1(f"select count(*) from notifications where user_id='{sid12}' and course_id='{C0}' and type like 'GRADE%'"))
time.sleep(2); nn12b = int(sq1(f"select count(*) from notifications where user_id='{sid12}' and course_id='{C0}' and type like 'GRADE%'"))
C("TC-P6-08-44", c == 200 and nv == 29 and n12 == 2 and nn12b == notif0, f"sv12 rời lớp → chốt lại v{v}: {nv} snapshot (29, không có sv12); snapshot cũ của sv12 còn {n12} (v1,v2); thông báo GRADE* của sv12 {notif0}→{nn12b} (không nhận mới)", f"{c} {r} {nv} {n12} {notif0} {nn12b}")
sql(f"update enrollments set status='ACTIVE', removed_at=null where course_id='{C0}' and user_id='{sid12}'")
# ---- 08-46 audit fail-closed (trigger chỉ nhắm lớp này)
unl("Mở khoá thử audit fail-closed")
sql(f"create or replace function qc_s8_failaudit() returns trigger language plpgsql as $$ begin raise exception 'QC audit fail'; end $$")
sql(f"create trigger qc_s8_failaudit before insert on audit_log for each row when (new.course_id='{C0}') execute function qc_s8_failaudit()")
ns0 = nsnap(); c, r = fin(); g_ = gb()
sql("drop trigger qc_s8_failaudit on audit_log"); sql("drop function qc_s8_failaudit()")
C("TC-P6-08-46", c >= 500 and nsnap() == ns0 and g_[0] == "OPEN", f"audit_log chặn INSERT (trigger riêng lớp này, thay REVOKE để không ảnh hưởng nhóm khác) → finalize {c}; không snapshot mới ({ns0}→{nsnap()}), sổ vẫn OPEN; đã gỡ trigger", f"{c} {r} {ns0} {nsnap()} {g_}")
c, r = fin(); print("refinalize after audit restore", c, r.get("version"))
