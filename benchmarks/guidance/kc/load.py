#!/usr/bin/env python3
"""Nạp bộ bài vào stack test bằng API THẬT (kiem-chung.md §1.3). Chỉ thư viện chuẩn, như một client ngoài (D46): không ghi DB.

  SEED_DEFAULT_PASSWORD=… python3 benchmarks/guidance/kc/load.py            nạp lớp KC106 + BT-A (60 bài) + BT-B (24 bài), bật cờ GRADING_GUIDANCE
  … load.py --limit 3                                                         chạy thử: chỉ nộp 3 bài đầu của mỗi bài tập
Biến: API_URL (https://localhost/api/v1), SEED_DEFAULT_PASSWORD (bắt buộc), KC_CLASS (KC106), KC_DATA.
Thứ tự nộp: ZIP BT-A kc001–kc020 (tập học) → ZIP BT-A kc021–kc060 (tập đo) → ZIP BT-B. KHÔNG bấm chấm (thuộc quy trình ở §2).
Chạy lại không tạo đôi: khoá idempotency cố định, và chỉ nộp những sinh viên chưa có bài MATCHED.
"""
import argparse, hashlib, io, json, os, re, time, uuid, zipfile
from datetime import datetime, timedelta, timezone
import urllib.request
from common import CLASS, PASSWORD, CODE, CTX, DATA, N_B, N_LEARN, N_STUDENTS, bt_students, call, die, find_assignment, list_all, login, wait_job

SEMESTER = "2026-2027-HK1"
STATUSES = ["NONE", "RUNNING", "DRAFT", "REVIEWED", "PUBLISHED", "FAILED"]
TITLE = {"bt-a": "Sự cố lộ dữ liệu ở doanh nghiệp vừa và nhỏ", "bt-b": "Xác thực một yếu tố và đa yếu tố"}


def multipart(field, filename, content, ctype):
    b = uuid.uuid4().hex
    body = (f"--{b}\r\nContent-Disposition: form-data; name=\"{field}\"; filename=\"{filename}\"\r\nContent-Type: {ctype}\r\n\r\n").encode() + content + f"\r\n--{b}--\r\n".encode()
    return body, {"Content-Type": f"multipart/form-data; boundary={b}"}


def put_blob(pre, data):
    h = {k: v for k, v in (pre.get("headers") or {}).items() if k.lower() != "content-length"}
    with urllib.request.urlopen(urllib.request.Request(pre["url"], data=data, headers=h, method=pre.get("method", "PUT")), context=CTX, timeout=120) as r:
        if r.status >= 300:
            die(f"PUT tệp → {r.status}")


def matched_codes(token, course, asg):
    seen = {}
    for st in STATUSES:
        for it in list_all(f"/courses/{course}/submissions?assignment_id={asg}&match_status=MATCHED&status={st}", token):
            if it.get("student"):
                seen[it["student"]["student_code"]] = it["id"]
    return seen


def hkey(prefix, body):
    """Khoá idempotency gắn với nội dung: sửa nội dung rồi chạy lại không đụng IDEMPOTENCY_KEY_REUSED."""
    return f"{prefix}-{hashlib.sha256(json.dumps(body, sort_keys=True).encode()).hexdigest()[:10]}"


MAILPIT = os.environ.get("MAILPIT_URL", "http://localhost:8025").rstrip("/")


def mailpit(route):
    with urllib.request.urlopen(MAILPIT + route, timeout=30) as r:
        return json.load(r)


def mail_token(addr, pattern, timeout=90):
    """Token trong thư mới nhất tới `addr` có liên kết khớp `pattern` (Mailpit)."""
    end = time.time() + timeout
    while time.time() < end:
        msgs = mailpit(f"/api/v1/search?query=to:{addr}&limit=20").get("messages") or []
        for m in sorted(msgs, key=lambda m: m["Created"], reverse=True):
            mt = re.search(pattern, mailpit(f"/api/v1/message/{m['ID']}").get("Text") or "")
            if mt:
                return mt.group(1)
        time.sleep(0.5)
    die(f"Không thấy thư tới {addr} ({MAILPIT}); worker gửi thư / MAILPIT_URL đúng chưa?")


def ensure_accounts(adm, students):
    """Roster nhập xong thì mỗi email được MỜI (thư «Bạn được thêm vào lớp», liên kết /invite/<token>): nhận lời mời đặt mật khẩu = tài khoản ACTIVE + vào lớp với đúng MSSV → bài nộp mới khớp sinh viên.
    Trạng thái lấy từ API Admin, KHÔNG đăng nhập thử: đăng nhập sai bị tính vào giới hạn thất bại theo IP (AUTH_LOGIN_IP_FAIL_PER_15M = 30) và chặn cả lần chạy."""
    todo = []
    for s in students:
        _, r = call("GET", f"/admin/users?q={s['email']}&limit=10", adm)
        u = next((x for x in r["items"] if x["email"] == s["email"]), None)
        if u is None:
            die(f"{s['email']} chưa có tài khoản sau khi nhập roster: roster nhập chưa xong?")
        if u["status"] == "INVITED":
            todo.append(s)
    for s in todo:
        call("POST", "/auth/accept-invite", body={"token": mail_token(s["email"], r"/invite/([A-Za-z0-9_-]+)"), "password": PASSWORD}, ok=(200, 204))
    print(f"tài khoản: {len(todo)} nhận lời mời / {len(students)}")


def assignment_body(bt, rub):
    due = (datetime.now(timezone.utc) + timedelta(days=30)).strftime("%Y-%m-%dT%H:%M:%SZ")
    return {"title": TITLE[bt], "description": rub["topic"][bt], "type": "ESSAY", "max_score": "10", "due_at": due, "allow_late": False, "late_penalty_per_day": "0", "late_max_days": 0,
            "appeal_days": 7, "channels": ["UPLOAD"], "submission_code": CODE[bt], "rubric": rub["rubric"],
            "reference_answer": "Bài đạt yêu cầu nêu rõ vấn đề, có tình huống cụ thể, biện pháp khả thi gắn với nguyên nhân, trình bày rõ ràng."}


def pending(token, course, asg, students, limit):
    have = matched_codes(token, course, asg)
    return [s for s in students[: limit or None] if s["code"] not in have]


def send_zip(token, course, asg, bt, part, todo):
    if not todo:
        print(f"  {bt} {part}: đã đủ, bỏ qua")
        return
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as z:
        for s in todo:
            z.writestr(f"{s['code']}.txt", (DATA / bt / f"{s['code']}.txt").read_text(encoding="utf8"))
    data = buf.getvalue()
    sha = hashlib.sha256(data).hexdigest()
    _, pre = call("POST", f"/courses/{course}/uploads/presign", token, {"purpose": "submission_upload", "filename": f"{bt}-{part}.zip", "mime_type": "application/zip", "size_bytes": len(data), "sha256": sha}, key=f"kc-presign-{bt}-{part}-{sha[:16]}")
    put_blob(pre, data)
    st, out = call("POST", f"/courses/{course}/assignments/{asg}/submissions/upload", token, {"upload_id": pre["upload_id"]}, key=f"kc-zip-{bt}-{part}-{sha[:16]}")
    if st == 202:
        out = wait_job(token, out["job_id"]).get("result") or out
    print(f"  {bt} {part}: {len(todo)} bài → {json.dumps(out, ensure_ascii=False)[:200]}")


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--limit", type=int, default=0, help="chỉ nộp N bài đầu của mỗi phần (chạy thử)")
    a = ap.parse_args()
    if not (DATA / "rubric.json").exists():
        die(f"Chưa có {DATA}/rubric.json: chạy gen.py trước")
    rub = json.loads((DATA / "rubric.json").read_text(encoding="utf8"))
    adm, _ = login("admin")
    teacher, tid = login("teacher")
    row = next((c for c in list_all("/admin/courses", adm) if c["class_code"] == CLASS), None)
    if not row:
        body = {"subject_code": "INT1006", "class_code": CLASS, "name": "An ninh mạng (kiểm chứng)", "semester": SEMESTER, "join_code": "KCJPMTW"}
        _, row = call("POST", "/admin/courses", adm, body, key=hkey(f"kc-course-{CLASS}", body))
    course = row["id"]
    full = next(c for c in list_all("/admin/courses", adm) if c["id"] == course)
    if (full.get("teacher") or {}).get("id") != tid:
        call("POST", f"/admin/courses/{course}/assign", adm, {"teacher_id": tid})
    # roster 60 dòng (số dòng chạy thử cũng đủ 60: roster chỉ là danh sách, không tốn lời gọi LLM)
    csv_text = "Email,Họ và tên,MSSV\n" + "".join(f"{s['email']},{s['name']},{s['code']}\n" for s in bt_students("bt-a"))
    body, h = multipart("file", f"roster-{CLASS}.csv", csv_text.encode("utf8"), "text/csv")
    st, r = call("POST", f"/courses/{course}/roster/import", teacher, key=hkey(f"kc-roster-{CLASS}", {"csv": csv_text}), raw=body, headers=h, ok=(200, 201, 202, 409))
    print(f"roster: {st}")
    asg = {}
    for bt in ("bt-a", "bt-b"):
        have = next((x for x in list_all(f"/courses/{course}/assignments", teacher) if x["submission_code"] == CODE[bt]), None)
        if not have:
            body = assignment_body(bt, rub)
            _, have = call("POST", f"/courses/{course}/assignments", teacher, body, key=f"kc-create-{CLASS}-{bt}-{uuid.uuid4().hex[:8]}")  # due_at đổi mỗi lần chạy; chỉ tạo khi chưa có
        if not have.get("published_at"):
            call("POST", f"/courses/{course}/assignments/{have['id']}/publish", teacher, key=f"kc-publish-{CLASS}-{bt}")
        asg[bt] = have["id"]
    _, flags = call("GET", f"/courses/{course}/guidance-flags", teacher)
    cur = next(f for f in flags["items"] if f["feature"] == "GRADING_GUIDANCE")
    if not cur["enabled"]:
        call("PUT", f"/courses/{course}/guidance-flags/GRADING_GUIDANCE", teacher, {"enabled": True, "version": cur["version"]})
    sa = bt_students("bt-a")
    plan = [("bt-a", "hoc", sa[:N_LEARN]), ("bt-a", "do", sa[N_LEARN:]), ("bt-b", "doichung", bt_students("bt-b"))]
    plan = [(bt, part, pending(teacher, course, asg[bt], ss, a.limit)) for bt, part, ss in plan]
    ensure_accounts(adm, {s["code"]: s for _, _, todo in plan for s in todo}.values())
    for bt, part, todo in plan:
        send_zip(teacher, course, asg[bt], bt, part, todo)
    want = {"bt-a": N_STUDENTS, "bt-b": N_B}
    for bt, n in want.items():
        got = len(matched_codes(teacher, course, asg[bt]))
        exp = n if not a.limit else min(n, a.limit * (2 if bt == "bt-a" else 1))
        print(f"{bt}: {got}/{exp} bài MATCHED")
        if got < exp:
            die(f"{bt}: chỉ {got}/{exp} bài khớp sinh viên")
    print(f"Lớp {CLASS} sẵn sàng (cờ GRADING_GUIDANCE bật). Chưa chấm: bước 2 của quy trình do giaovien làm.")


if __name__ == "__main__":
    main()
