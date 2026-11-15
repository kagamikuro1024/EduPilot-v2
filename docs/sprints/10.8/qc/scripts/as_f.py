# US-AS-13 — giữ 30 ngày (purge) bằng fixture closed_at + `gateway admin purge-checkins`
from asl import *
import subprocess, concurrent.futures as cf
S = st(); C = S["cid"]
policy(C, rotate_seconds=5, late_after_min=10, geofence=None, campus_cidrs=[], require="WEB")
def mk(users, keys, minutes=None):
    s, w, d = neww(C); t = tokenof(d)
    for u, k in zip(users, keys): checkin(u, C, {"token": t, **({"device": k.dev(u, token=t)} if k else {})})
    return s, w
Ka1, Ka2, Kboth = Key(), Key(), Key()
sA, wA = mk(["sv.gioi", "sv.kha", "sv.nguyco"], [Ka1, Kboth, Ka2]); close(C, wA)
sB, wB = mk(["sv.kha", "sv04", "sv05", "sv06"], [Kboth, None, None, None]); close(C, wB)
sC, wC = mk(["sv07", "sv08"], [None, None])  # cửa sổ C đang mở
sql(f"update checkin_windows set opened_at=now()-interval '31 days', closes_at=now()-interval '31 days'+interval '10 minutes', closed_at=now()-interval '31 days'+interval '9 minutes' where id='{wA}'")
sql(f"update checkin_windows set opened_at=now()-interval '29 days', closes_at=now()-interval '29 days'+interval '10 minutes', closed_at=now()-interval '29 days'+interval '9 minutes' where id='{wB}'")
sql(f"update attendance_checkins set created_at=now()-interval '31 days' where window_id='{wA}'"); sql(f"update attendance_checkins set created_at=now()-interval '29 days' where window_id='{wB}'")
sql("update user_devices set created_at = now() - interval '31 days' where id in (select distinct device_id from attendance_checkins where window_id in ('%s','%s') and device_id is not null)" % (wA, wB))
cnt = lambda w: int(sql(f"select count(*) from attendance_checkins where window_id='{w}'")[0][0])
devs = lambda: int(sql(f"select count(*) from user_devices where id in (select distinct device_id from attendance_checkins where window_id in ('{wA}','{wB}','{wC}') and device_id is not null)")[0][0])
orph_before = int(sql("select count(*) from user_devices d where not exists (select 1 from attendance_checkins a where a.device_id=d.id)")[0][0])
print("trước:", cnt(wA), cnt(wB), cnt(wC), "devices A∪B:", devs())
dev_a_only = sql(f"select d.id from user_devices d where d.id in (select device_id from attendance_checkins where window_id='{wA}' and device_id is not null) and not exists (select 1 from attendance_checkins x where x.device_id=d.id and x.window_id<>'{wA}')")
run = lambda: subprocess.run("docker compose -p ep-qc exec -T gateway /gateway admin purge-checkins", shell=True, capture_output=True, text=True, cwd=os.path.expanduser("~/Documents/TA_Agent_qc5"))
r1 = run(); print(r1.stdout.strip(), r1.stderr.strip()[:200])
ok("TC-AS13-01a xoá lượt của cửa sổ closed_at > 30 ngày (A: 3 → 0); B (29 ngày) giữ 4; C (đang mở) giữ 2", cnt(wA) == 0 and cnt(wB) == 4 and cnt(wC) == 2, f"A={cnt(wA)} B={cnt(wB)} C={cnt(wC)} | {r1.stdout.strip()}")
still = sql(f"select count(*) from user_devices where id in ({','.join(repr(x[0]) for x in dev_a_only)})")[0][0] if dev_a_only else "0"
ok("TC-AS13-01b thiết bị chỉ cửa sổ A dùng (mồ côi quá 30 ngày) bị xoá; thiết bị còn dùng ở B (Kboth) giữ", still == "0" and int(sql(f"select count(*) from user_devices where key_sha256=decode('{hashlib.sha256(base64.b64decode(Kboth.pub)).hexdigest()}','hex')")[0][0]) == 1, f"a_only còn {still}")
ok("TC-AS13-01c cửa sổ A vẫn còn (giữ headcount; không dữ liệu cá nhân)", sql(f"select count(*) from checkin_windows where id='{wA}'")[0][0] == "1", "")
# 03: hai lần liên tiếp / song song
r2 = run(); 
with cf.ThreadPoolExecutor(2) as ex: rr = list(ex.map(lambda _: run(), range(2)))
ok("TC-AS13-03 chạy purge lần 2 và 2 tiến trình song song: không lỗi, kết quả không đổi, cửa sổ mở/B giữ nguyên", all(x.returncode == 0 for x in [r2] + rr) and cnt(wB) == 4 and cnt(wC) == 2 and "0 lượt" in r2.stdout, f"{r2.stdout.strip()} | {[x.stdout.strip() for x in rr]}")
# 02: sau purge: lượt cũ không còn
c, j = rows(C, sA); ok("TC-AS13-02a Staff danh sách lượt của buổi A (đã purge) trống", c == 200 and j["items"] == [], str(len(j.get("items", []))))
# 08 log: không student_id/tên
lg = subprocess.run("docker logs ep-qc-gateway-1 2>&1 | grep -i purge | tail -5; docker logs ep-qc-worker-1 2>&1 | grep -i 'attendance.purge' | tail -3", shell=True, capture_output=True, text=True).stdout
ok("TC-AS13-08 log purge chỉ có số hàng; không student_id / tên", not re.search(r"student_id|Nguyễn|sv\.|@edupilot", lg) if (re := __import__("re")) else 0, lg.strip()[:200] or "(không có dòng log purge trong gateway/worker — CLI in ra stdout)")
# 07: schema
cols = sql("select table_name||'.'||column_name from information_schema.columns where table_name in ('attendance_checkins','user_devices') and (column_name ~* '(^|_)(lat|lng|lon|ip|ua|imei|serial|name|label_text)($|_)' or column_name ~* 'coord')")
ok("TC-AS13-07 không cột toạ độ / IP / tên máy ở attendance_checkins, user_devices", not cols, str(cols))
