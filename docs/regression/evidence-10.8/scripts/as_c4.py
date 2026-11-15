# TC-AS06-08 lặp: hai cửa sổ có cờ (w1 đẩy lùi 8 ngày bằng SQL, w2 để nguyên làm đối chứng — provider Hôm nay có hạn 60 ms nên một lần gọi rỗng chưa chứng minh gì)
from asl import *
S=st();C=S["cid"]
def items():
    for _ in range(6):
        j=api(TEACHER,"GET","/me/today")[1]; it=[x for x in j["actions"]+j["attention"] if x["kind"]=="CHECKIN_REVIEW"]
        if it: return it
        time.sleep(2)
    return []
ws=[]
for k in range(2):
    s,w,d=neww(C); [checkin(u,C,{"token":tokenof(d)}) for u in ("sv.gioi","sv.kha")]; close(C,w); ws.append((s,w))
sql(f"update checkin_windows set closed_at = now() - interval '8 days' where id='{ws[0][1]}'")
time.sleep(63); it=items(); t=json.dumps(it, ensure_ascii=False)
ok("TC-AS06-08 (fixture SQL, chờ TTL 63 s) cửa sổ đóng > 7 ngày → mục biến mất; đối chứng cửa sổ 2 còn", f"session={ws[1][0]}" in t and f"session={ws[0][0]}" not in t, t[:300])
for k in (0,1):
    for i in [x[0] for x in sql(f"select id from attendance_checkins where window_id='{ws[k][1]}'")]: api(TEACHER, "POST", f"/courses/{C}/checkins/{i}/clear-flags", None)
