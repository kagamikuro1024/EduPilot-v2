# s8r 08d: QD8R81 — chốt thất bại (thiếu điểm, EXAM chưa công bố, NO_POLICY), idempotency, RBAC, biên lý do mở khoá, SV rời lớp, audit fail-closed.
from s8r_g import *
IT = lambda k: ITEM[k]
nsnap = lambda: int(sq1(f"select count(*) from final_grade_snapshots where course_id='{C0}'"))
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
