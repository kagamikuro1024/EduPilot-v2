# smoke r10.8: TC-AS14-12 điểm danh tay không đổi: PUT …/attendance cho cùng phản hồi (so cấu trúc/byte sau che id/thời gian) ở lớp TẮT và lớp BẬT self_checkin (buổi chưa có cửa sổ)
from asl import *
import re
S = st(); C, C3 = S["cid"], S["cid3"]
for _c in (C3, C):
    try: gen_sessions(_c, 300, 340)
    except AssertionError: pass
policy(C3, self_checkin=False); policy(C, rotate_seconds=20)
def run(cid):
    free = [s_["id"] for s_ in sessions(TEACHER, cid) if not sql(f"select 1 from checkin_windows where session_id='{s_['id']}'")][0]
    g = api(TEACHER, "GET", f"/courses/{cid}/sessions/{free}/attendance")[1]
    u = g["students"][0]["id"]
    out = []
    for body in ({"records": [{"student_id": u, "status": "PRESENT"}]}, {"records": [{"student_id": u, "status": "ABSENT"}], "complete": True}):
        c, j, _ = api(TEACHER, "PUT", f"/courses/{cid}/sessions/{free}/attendance", body)
        out.append((c, re.sub(r"[0-9a-f]{8}-[0-9a-f-]{27}|\d{4}-\d\d-\d\dT[\d:.]+Z?", "X", json.dumps(j, sort_keys=True, ensure_ascii=False)), sorted(j) if isinstance(j, dict) else None))
    return out, g
a, ga = run(C3); b, gb = run(C)
same = [x[0] == y[0] and x[2] == y[2] for x, y in zip(a, b)]
ok("TC-AS14-12 điểm danh tay (buổi không cửa sổ): PUT …/attendance cho cùng mã và cùng cấu trúc phản hồi ở lớp tắt / bật self_checkin; GET lưới cùng khoá", all(same) and sorted(ga) == sorted(gb) and a[0][0] == 200, f"tắt={[(x[0], x[2]) for x in a]} bật={[(y[0], y[2]) for y in b]}")
