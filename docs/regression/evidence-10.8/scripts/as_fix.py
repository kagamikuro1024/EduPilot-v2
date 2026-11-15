# smoke r10.8: chạy lại các ca trượt do môi trường / dữ liệu lặp (xem smoke-10.8.md)
import re
from asl import *
import subprocess
S=st();C=S["cid"];C2=S["cid2"]
# --- 03-07 (c): lớp tắt self_checkin, KHÔNG có cửa sổ mở
for s_ in sessions(TEACHER,C2):
    c,j,_=api(TEACHER,"GET",f"/courses/{C2}/sessions/{s_['id']}/checkin-windows")
    for x in (j or {}).get("items",[]):
        if not x["closed_at"]: close(C2,x["id"])
policy(C2, self_checkin=False)
ca = raw("POST", f"/courses/{C}/checkins", {"token": "x"}, {"Idempotency-Key": str(uuid.uuid4())})[0]
cb, jb, _ = checkin("sv.moi", C, {"token": "x"}); cc, jc, _ = checkin("sv.gioi", C2, {"token": "x"})
policy(C, rotate_seconds=20, late_after_min=10, geofence=None, campus_cidrs=[], require="WEB")
s,w,d=neww(C); t=tokenof(d); close(C,w); cd, jd, _ = checkin("sv.gioi", C, {"token": t[:-1] + ("A" if t[-1]!="A" else "B")})
ok("TC-AS03-07 thứ tự kiểm: 401 → 403 → 409 SELF_CHECKIN_DISABLED → (đóng+MAC sai) 422 CHECKIN_TOKEN_INVALID", ca==401 and cb==403 and cc==409 and jc.get("code")=="SELF_CHECKIN_DISABLED" and cd==422 and jd.get("code")=="CHECKIN_TOKEN_INVALID", f"{ca} {cb} {cc}:{jc.get('code')} | {cd}:{jd.get('code')} (chạy lại trên lớp không có cửa sổ mở)")
# --- 03-18 me/attendance sau đóng
policy(C, rotate_seconds=20, late_after_min=10, require="WEB")
s,w,d=neww(C); t=tokenof(d); checkin("sv.gioi",C,{"token":t}); close(C,w)
c,m,_=api("sv.gioi","GET",f"/courses/{C}/me/attendance"); items=m["attendance"]["sessions"]; row=[x for x in items if x.get("checked_in_at")]; txt=json.dumps(m)
ok("TC-AS03-18 me/attendance: buổi đã điểm danh (sau đóng) có checked_in_at; không cờ/mức/thiết bị", c==200 and bool(row) and not re.search(r'flags|assurance|device|geo_(?!check)', txt), f"{row[:1]} keys={list(items[0]) if items else None}")
# --- 04-10 DEVICE_NEW với SV chưa từng có khoá
u="sv11"; K1,K2=Key(),Key()
fs=[]
for k in (K1,K2,K2):
    s,w,d=neww(C); t=tokenof(d); r=checkin(u,C,{"token":t,"device":k.dev(u,token=t)})[0]
    fs.append((r, sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-') from attendance_checkins where window_id='{w}' and student_id='{uid(u)}'")[0][0])); close(C,w)
ok("TC-AS04-10 DEVICE_NEW: K1 lần đầu không cờ; K2 đầu: DEVICE_NEW; K2 sau: không", fs[0][0]==fs[1][0]==fs[2][0]==201 and "DEVICE_NEW" not in fs[0][1] and "DEVICE_NEW" in fs[1][1] and "DEVICE_NEW" not in fs[2][1], str(fs))
# --- 10-08b
logs = subprocess.run("docker logs ep-r-gateway-1 2>&1 | grep -c -E '10\\.7727|106\\.6602|\"lat\"|\"lng\"|lat=|lng='", shell=True, capture_output=True, text=True).stdout.strip()
ok("TC-AS10-08b log gateway không chứa toạ độ đã gửi (khớp chính xác 10.7727 / 106.6602)", logs=="0", logs)
# --- 06-03 EXPLAIN truy vấn thật
print(subprocess.run("grep -n 'AttListSession\\|const attList' ../../../../backend-go/internal/store/attendance_checkin.sql.go | head",shell=True,capture_output=True,text=True,cwd=os.path.dirname(os.path.abspath(__file__))).stdout)
