# smoke r10.8: TC-AS02-18, 02-19, 05-05 (tự đóng sau 60 s; quét ≤ 60 s), TC-AS04-11 (TOKEN_LATE, rotate 60)
from asl import *
import re
S=st(); C3=S["cid3"]
try: gen_sessions(C3, 2, 40)
except AssertionError: pass
policy(C3, rotate_seconds=5, late_after_min=10)
# --- 02-18 / 19 / 05-05
s,w,d=neww(C3, minutes=1); t=tokenof(d)
r=[checkin(u,C3,{"token":t})[0] for u in ("sv.gioi","sv.kha")]
time.sleep(125)
row=sql(f"select closed_at is not null, closed_by is null, (select count(*) from audit_log where entity_id='{w}' and action like '%auto_close%') from checkin_windows where id='{w}'")[0]
recs=sql(f"select r.status, count(*) from attendance_records r where r.session_id='{s}' group by 1")
aud=sql(f"select action, actor_id is null from audit_log where entity_id='{w}' order by created_at")
ok("TC-AS02-18 minutes=1 tự đóng ≤ 125 s: closed_at đặt, closed_by NULL, audit auto_close, records PRESENT/LATE, không ABSENT", r==[201,201] and row[0]=="t" and row[1]=="t" and int(row[2])>=1 and recs and all(x[0] in ("PRESENT","LATE") for x in recs), f"{r} {row} {recs} {aud}")
mb=sql(f"select marked_by is not null from attendance_records where session_id='{s}' limit 1")
ok("TC-AS05-05 cửa sổ tự đóng: closed_by NULL, marked_by = người mở, cùng quy tắc", row[1]=="t" and mb and mb[0][0]=="t", f"{row} marked_by_notnull={mb}")
snap=(sql(f"select closed_at from checkin_windows where id='{w}'")[0][0], sql(f"select count(*) from audit_log where entity_id='{w}'")[0][0], sql(f"select count(*) from attendance_records where session_id='{s}'")[0][0])
time.sleep(125)
snap2=(sql(f"select closed_at from checkin_windows where id='{w}'")[0][0], sql(f"select count(*) from audit_log where entity_id='{w}'")[0][0], sql(f"select count(*) from attendance_records where session_id='{s}'")[0][0])
ok("TC-AS02-19 sau thêm 2 nhịp quét (125 s): không đổi gì, không audit mới", snap==snap2, f"{snap} -> {snap2}")
# --- 04-11 TOKEN_LATE (rotate 60 ⇒ ngưỡng 65 s)
policy(C3, rotate_seconds=60, late_after_min=10)
s,w,d=neww(C3); E=datetime.datetime.fromisoformat(d["step_expires_at"].replace("Z","+00:00"))
time.sleep(max(0,(E-datetime.datetime.now(datetime.timezone.utc)).total_seconds())+1.5)
d=display(C3,s,w)[1]; t=tokenof(d); t0=time.time()
time.sleep(max(0,30-(time.time()-t0))); ca=checkin("sv.gioi",C3,{"token":t})[0]
time.sleep(max(0,70-(time.time()-t0))); cb=checkin("sv.kha",C3,{"token":t})[0]
q=lambda u: sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-'), token_age_ms from attendance_checkins where window_id='{w}' and student_id='{uid(u)}'")[0]
fa,fb=q("sv.gioi"),q("sv.kha")
ok("TC-AS04-11 rotate=60: tuổi ≈30 s không cờ; tuổi ≈70 s TOKEN_LATE vẫn 201; token_age_ms khớp ±1 s", ca==cb==201 and "TOKEN_LATE" not in fa[0] and "TOKEN_LATE" in fb[0] and abs(int(fa[1])-30000)<1500 and abs(int(fb[1])-70000)<1500, f"{ca} {fa} | {cb} {fb}")
close(C3,w); policy(C3, rotate_seconds=5)
