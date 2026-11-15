# Kiểm lại D-AS-06: tắt self_checkin giữa buổi không chặn cửa sổ đang mở
from asl import *
S = st(); C = S["cid"]
policy(C, rotate_seconds=5, late_after_min=10)
s, w, d = neww(C); t = tokenof(d)
policy(C, self_checkin=False)
c1, j1, _ = checkin("sv.gioi", C, {"token": t}); c2, j2, _ = checkin("sv.kha", C, {"code": display(C, s, w)[1]["short_code"]}); dd = display(C, s, w)
ok("TC-AS01-10a (kiểm lại D-AS-06) tắt self_checkin khi cửa sổ đang mở: SV vẫn điểm danh được (token và mã ngắn → 201) tới khi đóng; rotate vẫn ghim 5 s", c1 == 201 and c2 == 201 and dd[0] == 200 and dd[1]["rotate_seconds"] == 5, f"{c1} {c2} {dd[0]}")
c, j, _ = openw(C, fresh_session(C)); ok("TC-AS01-10b (kiểm lại) cửa sổ MỚI khi tắt → vẫn 409 SELF_CHECKIN_DISABLED", c == 409 and j.get("code") == "SELF_CHECKIN_DISABLED", f"{c} {j.get('code')}")
close(C, w); time.sleep(1)
s3, _, _ = None, None, None
c3, j3, _ = checkin("sv.nguyco", C, {"token": t}); ok("TC-AS01-10d sau khi ĐÓNG cửa sổ (lớp vẫn tắt) → 409 (WINDOW_CLOSED hoặc SELF_CHECKIN_DISABLED), không 201", c3 == 409, f"{c3} {j3.get('code')}")
c4, j4, _ = checkin("sv.nguyco", S["cid2"], {"token": t}); ok("TC-AS01-10e lớp khác dùng token này → không 201", c4 != 201, str(c4))
policy(C, rotate_seconds=5, late_after_min=10)
