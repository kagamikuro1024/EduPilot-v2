#!/usr/bin/env python3
"""TC-P7-03-06 (MinIO từ chối PUT sai checksum / Content-Length / URL quá 10 phút), 03-09 (student_id trong thân submit bị bỏ qua)."""
import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("03x"); code = lambda: str(int(time.time()) % 100000)
C = mkcourse("QD9I", ["sv.gioi", "sv.kha"]); Q = f"/api/v1/courses/{C}"; T = tok("teacher"); SA, SB = tok("sv.gioi"), tok("sv.kha")
aid = mkassign(Q, T, "Q3X" + code())
def pres(tk, data, name="x.md"):
    s, pre, _ = call("POST", f"{Q}/assignments/{aid}/submit/presign", tk, {"filename": name, "size_bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}); assert s == 200, (s, pre); return pre
nsub = lambda: int(psql(f"select count(*) from submissions where assignment_id='{aid}'"))
data = ("Nội dung bài làm. SUNBURST. " * 10).encode()
# 03-09
SAid = psql("select id from users where email='sv.gioi@edupilot.local'"); SBid = psql("select id from users where email='sv.kha@edupilot.local'")
pre = pres(SB, data); raw(pre["method"], pre["url"], pre["headers"], data)
s, x, _ = call("POST", f"{Q}/assignments/{aid}/submit", SB, {"upload_id": pre["upload_id"], "student_id": SAid, "user_id": SAid}, key=idem())
owner = psql(f"select student_id from submissions where assignment_id='{aid}'")
R.chk("TC-P7-03-09", f"B gửi submit kèm student_id/user_id của A → {s}; bài thuộc B ({owner == SBid}), A có {psql(f'''select count(*) from submissions where assignment_id='{aid}' and student_id='{SAid}' ''')} bài", s in (200, 201) and owner == SBid, (s, x))
# 03-06 a) checksum sai
base0 = nsub()
pre = pres(SA, data); h = dict(pre["headers"]); ck = [k for k in h if "checksum" in k.lower()]; print("   header presign:", sorted(h)[:8])
h2 = dict(h)
for k in ck: h2[k] = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
s_, b_, _ = raw(pre["method"], pre["url"], h2, data); R.chk("TC-P7-03-06", f"PUT MinIO với x-amz-checksum-sha256 sai ({ck}) → {s_}", 400 <= s_ < 500, (s_, b_[:100]))
s, x, _ = call("POST", f"{Q}/assignments/{aid}/submit", SA, {"upload_id": pre["upload_id"]}, key=idem()); R.chk("TC-P7-03-06", f"submit sau đó (object không có) → {s} {x.get('code') if isinstance(x, dict) else x}", s >= 400 and nsub() == base0, (s, x))
# b) Content-Length lệch (thân ngắn hơn)
pre = pres(SA, data); s_, b_, _ = raw(pre["method"], pre["url"], pre["headers"], data[:-5]); R.chk("TC-P7-03-06", f"PUT với thân ngắn hơn size đã ký (Content-Length lệch) → {s_}", 400 <= s_ < 500, (s_, b_[:100]))
s, x, _ = call("POST", f"{Q}/assignments/{aid}/submit", SA, {"upload_id": pre["upload_id"]}, key=idem()); R.chk("TC-P7-03-06", f"submit sau đó → {s}; không bài nộp mới ({nsub()} = {base0})", s >= 400 and nsub() == base0, (s, x))
# c) quá 10 phút
pre = pres(SA, data); t0 = time.time(); print("   URL expires:", [p for p in pre["url"].split("&") if "Expires" in p]); time.sleep(620)
s_, b_, _ = raw(pre["method"], pre["url"], pre["headers"], data); R.chk("TC-P7-03-06", f"PUT sau {time.time() - t0:.0f} s (>10 phút) → {s_}", 400 <= s_ < 500, (s_, b_[:100]))
s, x, _ = call("POST", f"{Q}/assignments/{aid}/submit", SA, {"upload_id": pre["upload_id"]}, key=idem()); R.chk("TC-P7-03-06", f"submit sau URL hết hạn → {s}; không bài nộp mới ({nsub()} = {base0})", s >= 400 and nsub() == base0, (s, x))
sys.exit(R.end())
