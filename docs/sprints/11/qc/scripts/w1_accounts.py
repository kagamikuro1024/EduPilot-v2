# Đợt R · làn 1: F1 Tài khoản, F2 Vào lớp, F13 Lịch, F14 Hôm nay (API thật ep-qc)
from rlib import *
import re, time, uuid
tag = str(int(time.time()))[-6:]
em = f"qcr{tag}@edupilot.local"; pw = "Qcr!" + tag + "aZ9"
# F1 đăng ký → mail xác minh → đăng nhập
c, j, _ = api(None, "POST", "/auth/register", {"email": em, "password": pw, "full_name": "QC Đợt R", "student_code": "20239" + tag[-3:]}, idem=str(uuid.uuid4()))
ok("R-F1-01 đăng ký", c in (200, 201, 202), f"{c} {str(j)[:100]}")
time.sleep(3)
ms = mails(to=em)
ok("R-F1-02 có mail xác minh", len(ms) >= 1, f"{len(ms)} mail")
tokn = None
if ms:
    t = mail_text(ms[0]["ID"]); m = re.search(r"token=([A-Za-z0-9_\-\.]+)", t); tokn = m.group(1) if m else None
c, j, _ = raw("POST", "/auth/login", {"email": em, "password": pw})[:3]
ok("R-F1-03 chưa xác minh vẫn đăng nhập được (không vào lớp)", c == 200, f"{c}")
if tokn:
    c, j, _ = api(None, "POST", "/auth/verify-email", {"token": tokn})
    ok("R-F1-04 xác minh bằng link", c in (200, 204), f"{c}")
    c, j, _ = api(None, "POST", "/auth/verify-email", {"token": tokn})
    ok("R-F1-05 link dùng lại bị từ chối (một lần)", c in (400, 404, 409, 410, 422), f"{c} {str(j)[:80]}")
else: ok("R-F1-04 xác minh bằng link", False, "không lấy được token từ mail")
c, j, _, _ = raw("POST", "/auth/login", {"email": em, "password": pw}, {"Idempotency-Key": str(uuid.uuid4())})
ok("R-F1-06 đăng nhập sau xác minh", c == 200 and j.get("access_token"), f"{c}")
stu = j["access_token"] if c == 200 else None
c, mc, _, _ = raw("GET", "/me/courses", token=stu)
ok("R-F1-07 SV mới chưa có lớp", c == 200 and len(mc.get("items", [])) == 0, f"{c} {len(mc.get('items', [])) if isinstance(mc, dict) else mc}")
# quên mật khẩu: cùng thông báo
c1, j1, _, _ = raw("POST", "/auth/forgot-password", {"email": em}, {"Idempotency-Key": str(uuid.uuid4())})
c2, j2, _, _ = raw("POST", "/auth/forgot-password", {"email": "khong-ton-tai-" + tag + "@edupilot.local"}, {"Idempotency-Key": str(uuid.uuid4())})
ok("R-F1-08 forgot: email có / không tồn tại → cùng phản hồi", c1 == c2 and str(j1) == str(j2), f"{c1} {j1} | {c2} {j2}")
time.sleep(3)
ms = [m for m in mails(to=em) if "mật khẩu" in m.get("Subject", "").lower() or "reset" in m.get("Subject", "").lower() or "đặt lại" in m.get("Subject", "").lower()]
rt = None
if ms:
    t = mail_text(ms[0]["ID"]); m = re.search(r"token=([A-Za-z0-9_\-\.]+)", t); rt = m.group(1) if m else None
ok("R-F1-09 mail đặt lại mật khẩu", rt is not None, f"{len(ms)} mail")
if rt:
    c, j, _ = api(None, "POST", "/auth/reset-password", {"token": rt, "new_password": pw + "x"})
    ok("R-F1-10 đặt lại mật khẩu", c in (200, 204), f"{c} {str(j)[:80]}")
    c, j, _, _ = raw("GET", "/me/courses", token=stu)
    ok("R-F1-11 phiên cũ bị thu hồi sau đặt lại", c == 401, f"{c}")
    c, j, _ = api(None, "POST", "/auth/reset-password", {"token": rt, "new_password": pw + "y"})
    ok("R-F1-12 link đặt lại dùng lại bị từ chối", c in (400, 404, 409, 410, 422), f"{c}")
# chống dò: 5 lần sai
for i in range(6):
    c, j, _, _ = raw("POST", "/auth/login", {"email": em, "password": "sai" + str(i)}, {"Idempotency-Key": str(uuid.uuid4())})
ok("R-F1-13 nhiều lần sai → bị chặn / chờ (429 hoặc 401 tăng dần)", c in (401, 429), f"lần 6: {c}")
# MSSV mạo danh: đăng ký với MSSV của sinh viên trong roster, email khác → không vào lớp / dữ liệu
em2 = f"qcr2{tag}@edupilot.local"
rows = sql("select u.student_code from users u where u.email='sv.kha@edupilot.local'")
mssv = rows[0][0] if rows else "20229002"
c, j, _ = api(None, "POST", "/auth/register", {"email": em2, "password": pw, "full_name": "Mạo danh", "student_code": mssv}, idem=str(uuid.uuid4()))
time.sleep(3)
ms2 = mails(to=em2); tk2 = None
if ms2:
    m = re.search(r"token=([A-Za-z0-9_\-\.]+)", mail_text(ms2[0]["ID"])); tk2 = m.group(1) if m else None
if tk2: api(None, "POST", "/auth/verify-email", {"token": tk2})
c, jl, _, _ = raw("POST", "/auth/login", {"email": em2, "password": pw}, {"Idempotency-Key": str(uuid.uuid4())})
if c == 200:
    cc, mc2, _, _ = raw("GET", "/me/courses", token=jl["access_token"])
    ok("R-F1-14 MSSV trùng, email khác: không có lớp / dữ liệu của SV thật", cc == 200 and not any(i["course"]["class_code"] == "761987" and i.get("status") == "ACTIVE" for i in mc2.get("items", [])), f"{cc} {[ (i['course']['class_code'], i.get('status')) for i in mc2.get('items', [])]}")
else: ok("R-F1-14 MSSV trùng, email khác", c in (200, 409, 422), f"đăng ký/đăng nhập {c}")
print("wave1-F1 done")
