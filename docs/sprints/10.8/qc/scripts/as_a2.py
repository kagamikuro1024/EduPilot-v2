# bổ sung US-AS-02: sửa các ca đã ghi sai trong as_a (lỗi test, không phải lỗi sản phẩm) + tự đóng (chờ ~125 s) + lưu trữ
from asl import *
S = st(); C, C2 = S["cid"], S["cid2"]
s2 = fresh_session(C)
r4 = [api(TEACHER, "POST", f"/courses/{C}/sessions/{s2}/checkin-windows", {"minutes": m})[0] for m in (0, 61, -1)] + [api(TEACHER, "POST", f"/courses/{C}/sessions/{s2}/checkin-windows", {"minutes": "x"})[0]]
n = sql(f"select count(*) from checkin_windows where session_id='{s2}'")[0][0]
ok("TC-AS02-04 (chạy lại) minutes 0/61/-1/'x' → 422, không tạo cửa sổ", all(x == 422 for x in r4) and n == "0", f"{r4} n={n}")
s = fresh_session(C); c, j, _ = openw(C, s); w = j["id"]; c, d, _ = display(C, s, w)
ok("TC-AS02-10 (chạy lại) short_code nhóm 3+3, không I L O U", c == 200 and len(d["short_code"].replace(" ", "")) == 6 and d["short_code"][3] == " " and not set(d["short_code"]) & set("ILOU"), d["short_code"])
close(C, w)
# tự đóng: minutes=1
sa = fresh_session(C); c, j, _ = openw(C, sa, minutes=1); wa = j["id"]; c, d, _ = display(C, sa, wa)
ck = checkin("sv.gioi", C, {"token": tokenof(d)}); print("điểm danh", ck[0])
t0 = time.time()
while time.time() - t0 < 135:
    r = sql(f"select closed_at is not null, closed_by is null from checkin_windows where id='{wa}'")[0]
    if r[0] == "t": break
    time.sleep(5)
dt = time.time() - t0
au = sql(f"select action from audit_log where entity_id='{wa}'")
rec = sql(f"select u.email, r.status from attendance_records r join users u on u.id=r.student_id where r.session_id='{sa}'")
ok("TC-AS02-18 tự đóng ≤ 125 s sau mở: closed_by NULL, audit auto_close", r[0] == "t" and r[1] == "t" and any("auto_close" in x[0] for x in au), f"{dt:.0f}s audit={au} records={rec}")
ok("TC-AS02-18b ghi attendance_records cho SV đã điểm danh, không ABSENT", len(rec) == 1 and rec[0][1] in ("PRESENT", "LATE"), str(rec))
nb = sql(f"select count(*) from audit_log where entity_id='{wa}'")[0][0]; time.sleep(125)
na = sql(f"select count(*) from audit_log where entity_id='{wa}'")[0][0]
ok("TC-AS02-19 hai nhịp quét sau đó không đổi gì", nb == na, f"{nb}->{na}")
