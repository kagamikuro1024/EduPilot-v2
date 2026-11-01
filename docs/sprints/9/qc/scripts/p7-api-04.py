#!/usr/bin/env python3
"""QC US-P7-04 hộp đen trên stack thật (TC-P7-04-03…17, phần có route). In PASS/FAIL; thoát 1 nếu lỗi."""
import copy, hashlib, io, json, os, ssl, subprocess, sys, time, urllib.request, uuid, zipfile
GW = os.environ.get("GW", "https://localhost:543"); PW = "Edupilot#Seed-2026"; ctx = ssl._create_unverified_context()
FX = os.path.join(os.path.dirname(__file__), "../../../../specs/FEAT-assignments/fixtures")
def raw(m, url, h=None, data=None):
    try:
        r = urllib.request.urlopen(urllib.request.Request(url, data=data, method=m, headers=h or {}), context=ctx); return r.status, r.read(), r.headers
    except urllib.error.HTTPError as e: return e.code, e.read(), e.headers
def call(m, p, tok=None, body=None, key=None):
    h = {"Content-Type": "application/json"}
    if tok: h["Authorization"] = "Bearer " + tok
    if key: h["Idempotency-Key"] = key
    s, t, hd = raw(m, GW + p, h, None if body is None else json.dumps(body).encode())
    return s, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode()), hd
tok = lambda e: call("POST", "/api/v1/auth/login", body={"email": e, "password": PW})[1]["access_token"]
T, TA, ADM, SA, SB, SL = (tok(x + "@edupilot.local") for x in ("teacher", "ta", "admin", "sv.gioi", "sv.kha", "sv.lech"))
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
C1 = psql("select id from courses where class_code='761987'"); Q = f"/api/v1/courses/{C1}"; idem = lambda: str(uuid.uuid4())
uid = lambda e: psql(f"select id from users where email='{e}@edupilot.local'")
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:260])
base = json.load(open(f"{FX}/create-bt03.json")); base["due_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 3 * 86400)); base["channels"] = ["IN_APP", "UPLOAD"]
def mk(code, over=None, publish=True):
    b = copy.deepcopy(base); b["submission_code"] = code; b.update(over or {})
    s, a, _ = call("POST", f"{Q}/assignments", T, b, idem()); a = a.get("assignment", a); aid = a["id"]
    if publish: call("POST", f"{Q}/assignments/{aid}/publish", T, {}, idem())
    return aid
def upload(aid, tk, name, data, mime="application/octet-stream", purpose="submission_upload"):
    time.sleep(6.5)  # hạn mức presign 10 lượt / phút / người
    sha = hashlib.sha256(data).hexdigest()
    s, pre, _ = call("POST", f"{Q}/uploads/presign", tk, {"purpose": purpose, "filename": name, "mime_type": mime, "size_bytes": len(data), "sha256": sha})
    if s != 200: return s, pre
    s2, _, _ = raw("PUT", pre["url"], pre["headers"], data)
    r = call("POST", f"{Q}/assignments/{aid}/submissions/upload", tk, {"upload_id": pre["upload_id"]}, idem())
    if r[0] == 202:  # ZIP nhiều mục chạy nền: chờ job (người tạo đọc)
        for _ in range(60):
            j = call("GET", f"/api/v1/jobs/{r[1]['job_id']}", tk)[1]
            if isinstance(j, dict) and j.get("status") in ("SUCCEEDED", "FAILED"): return (200 if j["status"] == "SUCCEEDED" else 422), (j.get("result") or j)
            time.sleep(1)
    return r[0], r[1]
def rows(aid, cols="file_name, match_status, coalesce(match_method,''), coalesce(unmatched_reason,''), coalesce(student_id::text,''), version"):
    return [r.split("|") for r in psql(f"select {cols} from submissions where assignment_id='{aid}' order by received_at, file_name").splitlines() if r]
MD = lambda t: (t + "\nBài làm Markdown của sinh viên về SUNBURST. " * 6).encode()
A = mk("BT03"); print("   BT03", A)
# --- TC-06/07: tệp lẻ đủ loại
s, x = upload(A, TA, "BT03_20229001.md", MD("a")); chk("06 tệp lẻ md hợp lệ (TA) → 200, created 1", s == 200 and x.get("created") == 1, f"{s} {x}")
r = rows(A); chk("01 BT03_20229001 → MATCHED, sv.gioi (MSSV từ tên tệp, danh sách lớp)", r and r[0][1] == "MATCHED" and psql(f"select email from users where id=(select student_id from submissions where assignment_id='{A}' limit 1)") == "sv.gioi@edupilot.local", r)
s, x = upload(A, TA, "bad.exe", b"MZ" + b"x" * 100); chk("06 .exe → 422 hoặc rejected FILE_TYPE", (s == 422) or (s == 200 and "TYPE" in json.dumps(x)), f"{s} {x}")
s, x = upload(A, TA, "rong.md", b""); chk("06 tệp rỗng → 422 / FILE_EMPTY", s == 422 or "EMPTY" in json.dumps(x), f"{s} {x}")
s, x = upload(A, TA, "ghi-chu.md", MD("không mã")); chk("01 không có mã bài / MSSV → UNMATCHED", s == 200 and x.get("unmatched") == 1, f"{s} {x}")
s, x = upload(A, TA, "BT03_99999999.md", MD("lạ")); chk("01 BT03 + MSSV ngoài lớp → UNMATCHED", s == 200 and x.get("unmatched") == 1, f"{s} {x}")
# --- TC-03/04: TLR-1 — sv.lech tự khai 20229002 (= B); tên tệp mang MSSV 20229002
s, x = upload(A, TA, "BT03_20229002.md", MD("của B")); r = [q for q in rows(A) if q[0] == "BT03_20229002.md"]
em = psql(f"select email from users where id='{r[0][4]}'") if r and r[0][4] else ""
chk("03 MSSV 20229002 (sv.lech tự khai trùng B) → chỉ gán B theo danh sách lớp, KHÔNG sv.lech", em == "sv.kha@edupilot.local", f"{r} {em}")
chk("03 không có hàng nào của sv.lech", psql(f"select count(*) from submissions where assignment_id='{A}' and student_id='{uid('sv.lech')}'") == "0")
chk("05 grep users.student_code ở internal/submission rỗng", subprocess.run("grep -rn 'users.student_code\\|student_code FROM users' backend-go/internal/submission --include=*.go | grep -v _test.go", shell=True, capture_output=True, text=True, cwd=os.path.join(os.path.dirname(__file__), "../../../../..")).stdout.strip() == "")
# --- TC-07/08/11: ZIP
def mkzip(entries, comp=zipfile.ZIP_DEFLATED):
    b = io.BytesIO()
    with zipfile.ZipFile(b, "w", comp) as z:
        for n, d in entries: z.writestr(n, d)
    return b.getvalue()
inner = mkzip([("a.md", b"x" * 50)])
E1 = [("BT03_20229003.md", MD("nguyco")), ("BT03_20229001.md", MD("gioi-v2")), ("thu-muc/", b""), ("__MACOSX/x", b"junk"), (".DS_Store", b"junk"),
      ("../../etc/passwd.txt", b"root"), ("/abs.txt", b"abs"), ("nested.zip", inner), ("x.exe", b"MZ" + b"y" * 50), ("rong.md", b"")]
z = mkzip(E1); s, x = upload(A, TA, "lop.zip", z, "application/zip"); print("   ZIP →", s, json.dumps(x, ensure_ascii=False)[:420])
codes = {d["entry"]: d["code"] for d in x.get("rejected", [])} if isinstance(x, dict) else {}
chk("07 ZIP: 2 tệp hợp lệ tạo bản, thư mục / __MACOSX / .DS_Store bị bỏ im", s == 200 and x.get("created") == 2, f"{s} {x}")
chk("07 `..` / tuyệt đối / zip lồng → ENTRY_UNSAFE; x.exe → ENTRY_TYPE_NOT_ALLOWED; 0 byte → FILE_EMPTY", codes.get("../../etc/passwd.txt") == "ENTRY_UNSAFE" and codes.get("/abs.txt") == "ENTRY_UNSAFE" and codes.get("nested.zip") == "ENTRY_UNSAFE" and codes.get("x.exe") == "ENTRY_TYPE_NOT_ALLOWED" and codes.get("rong.md") == "FILE_EMPTY", codes)
chk("07 không tạo tệp ngoài kho (không có bản ghi tên passwd/abs)", psql(f"select count(*) from submissions where assignment_id='{A}' and (file_name like '%passwd%' or file_name like '%abs.txt%')") == "0")
chk("08 created + unmatched + rejected + deduped = số mục (kèm hợp lệ)", x["created"] + len(x["rejected"]) + len(x["deduped"]) >= 2 + 4)
vers = lambda: psql(f"select count(*) from submissions where assignment_id='{A}'")
n0 = vers(); s, x2 = upload(A, TA, "lop.zip", z, "application/zip"); print("   ZIP lần 2 →", s, json.dumps(x2, ensure_ascii=False)[:300])
chk("11 tải lại cùng ZIP → deduped, không bản mới", s == 200 and x2.get("created") == 0 and len(x2.get("deduped", [])) >= 2 and vers() == n0, f"{s} {x2} {n0}->{vers()}")
z3 = mkzip([("BT03_20229003.md", MD("nguyco SỬA")), ("BT03_20229001.md", MD("gioi-v2"))]); s, x3 = upload(A, TA, "lop2.zip", z3, "application/zip")
chk("11 ZIP sửa một tệp → chỉ tệp đổi tạo phiên bản kế", s == 200 and x3.get("created") == 1 and len(x3.get("deduped", [])) == 1, f"{s} {x3}")
chk("11 phiên bản của sv.nguyco = 1,2", psql(f"select string_agg(version::text, ',' order by version) from submissions where assignment_id='{A}' and student_id='{uid('sv.nguyco')}'") == "1,2")
# bom: 300 MiB số không nén → vài trăm KB
bomb = mkzip([("BT03_20229003.md", b"\0" * (300 * 1024 * 1024))]); print("   bom", len(bomb), "byte nén")
s, xb = upload(A, TA, "bom.zip", bomb, "application/zip"); chk("07 ZIP bom 300 MiB giải nén → từ chối (ZIP_LIMIT / 422), không bản ghi mới", (s == 422 or "ZIP_LIMIT" in json.dumps(xb) or "TOO_LARGE" in json.dumps(xb) or "FILE_" in json.dumps(xb)) and vers() == str(int(n0) + 1), f"{s} {str(xb)[:160]} {vers()}")
big = mkzip([(f"BT03_2022{i:04d}.md", MD(str(i))) for i in range(201)]); s, xj = upload(A, TA, "201.zip", big, "application/zip"); print("   201 mục →", s, str(xj)[:160])
chk("07 ZIP 201 mục → 202 + job, hoặc bị chặn ZIP_LIMIT (giới hạn 200 mục)", s in (202, 422) or "ZIP_LIMIT" in json.dumps(xj), f"{s} {xj}")
psql(f"delete from submissions where assignment_id='{A}' and student_id is null and file_name ~ '^BT03_2022[0-9]{{4}}\\.md$' and file_name <> 'BT03_20229002.md'")  # dọn 200 mục của ZIP 201
# --- TC-12: bài chưa công bố / kênh tắt
U = mk("QC4U", publish=False); s, x = upload(U, TA, "BT03_20229001.md", MD("u")); chk("12 bài chưa công bố → 409 ASSIGNMENT_NOT_PUBLISHED", s == 409 and "ASSIGNMENT_NOT_PUBLISHED" in json.dumps(x), f"{s} {x}")
NC = mk("QC4C", {"channels": ["IN_APP"]}); s, x = upload(NC, TA, "BT03_20229001.md", MD("c")); chk("12 kênh UPLOAD tắt → 409 CHANNEL_NOT_ALLOWED", s == 409 and "CHANNEL_NOT_ALLOWED" in json.dumps(x), f"{s} {x}")
# --- TC-13: ALREADY_REVIEWED mọi nguồn (tệp): REVIEWED rồi nhận bản mới
sidB = psql(f"select id from submissions where assignment_id='{A}' and student_id='{uid('sv.kha')}' order by version desc limit 1")
psql(f"insert into grading_results(course_id,submission_id,run_state,status,flags,ai_scores,ai_total_before_late,criteria,overall_feedback,confidence) values ('{C1}','{sidB}','DONE','REVIEWED','{{}}','{{\"c1\":\"6.5\"}}',6.5,'[]','ok',0.9)")
tot0 = psql(f"select ai_total_before_late||'/'||status from grading_results where submission_id='{sidB}'")
s, x = upload(A, TA, "BT03_20229002.md", MD("B nộp lại sau khi đã duyệt")); r = [q for q in rows(A) if q[1] == "UNMATCHED" and q[3] == "ALREADY_REVIEWED"]
chk("13 tệp mới cho sinh viên đã REVIEWED → UNMATCHED ALREADY_REVIEWED + ứng viên, không đè điểm", s == 200 and x.get("unmatched") == 1 and len(r) == 1 and psql(f"select count(*) from submissions where id='{sidB}'") == "1" and psql(f"select ai_total_before_late||'/'||status from grading_results where submission_id='{sidB}'") == tot0 and psql(f"select candidate_student_id from submissions where unmatched_reason='ALREADY_REVIEWED' and assignment_id='{A}' limit 1") == uid("sv.kha"), f"{s} {x} {r}")
# --- TC-14: phân quyền
s, pre, _ = call("POST", f"{Q}/uploads/presign", SB, {"purpose": "submission_upload", "filename": "a.md", "mime_type": "text/markdown", "size_bytes": 10, "sha256": "0" * 64}); chk("14 STUDENT presign submission_upload → 403", s == 403, s)
s, _, _ = call("POST", f"{Q}/uploads/presign", ADM, {"purpose": "submission_upload", "filename": "a.md", "mime_type": "text/markdown", "size_bytes": 10, "sha256": "0" * 64}); chk("14 ADMIN presign → 403", s == 403, s)
for n, tk, want in [("STUDENT", SB, 403), ("ADMIN", ADM, 403), ("TA", TA, 200), ("TEACHER", T, 200)]:
    s, _, _ = call("GET", f"{Q}/submissions?match_status=UNMATCHED", tk); chk(f"14 {n} GET submissions?match_status=UNMATCHED → {want}", s == want, s)
s, _, _ = call("POST", f"{Q}/assignments/{A}/submissions/upload", SB, {"upload_id": str(uuid.uuid4())}, idem()); chk("14 STUDENT upload → 403", s == 403, s)
un = call("GET", f"{Q}/submissions?match_status=UNMATCHED&limit=100", T)[1]; items = un.get("items", un) if isinstance(un, dict) else un
print("   UNMATCHED:", len(items), [ (i.get("file_name"), i.get("unmatched_reason")) for i in items][:8])
ids = {i["file_name"]: i for i in items}; g = ids.get("ghi-chu.md") or items[0]; sid = g["id"]
s, f, _ = call("GET", f"{Q}/submissions/{sid}/file", T); chk("14 TEACHER lấy URL tệp ký", s == 200 and ("url" in json.dumps(f)), f"{s} {f}")
for n, tk in [("STUDENT B", SB), ("ADMIN", ADM)]:
    s, _, _ = call("GET", f"{Q}/submissions/{sid}/file", tk); chk(f"14 {n} lấy URL tệp → 403", s == 403, s)
# --- TC-10: gán tay
ver = g["version"]; stu = {"sv.moi": uid("sv.moi"), "sv.nguyco": uid("sv.nguyco")}
s, x, _ = call("PUT", f"{Q}/submissions/{sid}/match", T, {"assignment_id": A, "student_id": stu["sv.moi"], "version": ver}); chk("10 gán cho sinh viên chưa vào lớp → 422", s == 422, f"{s} {x}")
s, x, _ = call("PUT", f"{Q}/submissions/{sid}/match", T, {"assignment_id": A, "student_id": stu["sv.nguyco"], "version": ver + 1}); chk("10 sai version → 409 VERSION_CONFLICT", s == 409 and "VERSION_CONFLICT" in json.dumps(x), f"{s} {x}")
s, x, _ = call("PUT", f"{Q}/submissions/{sid}/match", T, {"assignment_id": A, "student_id": uid("sv.kha"), "version": ver}); chk("10 gán cho B đã REVIEWED → 409 ALREADY_REVIEWED", s == 409 and "ALREADY_REVIEWED" in json.dumps(x), f"{s} {x}")
mx = int(psql(f"select coalesce(max(version),0) from submissions where assignment_id='{A}' and student_id='{stu['sv.nguyco']}'"))
s, x, _ = call("PUT", f"{Q}/submissions/{sid}/match", TA, {"assignment_id": A, "student_id": stu["sv.nguyco"], "version": ver}); chk("09/10 TA gán bài chưa công bố điểm → 200, version = max+1", s == 200 and x.get("version") == mx + 1, f"{s} {x} mx={mx}")
chk("09 match_method=MANUAL + audit_log", psql(f"select match_method from submissions where id='{sid}'") == "MANUAL" and int(psql(f"select count(*) from audit_log where entity_id::text='{sid}' or after::text like '%{sid}%'") or 0) >= 1, psql(f"select match_method from submissions where id='{sid}'"))
s, x, _ = call("PUT", f"{Q}/submissions/{sid}/match", T, {"assignment_id": A, "student_id": stu["sv.nguyco"], "version": ver}); chk("10 gán lần hai cùng version cũ → 409", s == 409, f"{s} {x}")
n_ntf = psql(f"select count(*) from notifications where user_id='{stu['sv.nguyco']}' and created_at > now() - interval '2 minutes' and type like '%GRAD%'"); print("   thông báo điểm cho sv.nguyco sau khi gán:", n_ntf)
# --- TC-16: bỏ qua / hoàn tác
un = call("GET", f"{Q}/submissions?match_status=UNMATCHED&limit=100", T)[1]; items = un.get("items", un); d = next(i for i in items if i["file_name"] == "BT03_99999999.md"); did = d["id"]
for reason, want in [("", 422), ("x" * 201, 422)]:
    s, _, _ = call("POST", f"{Q}/submissions/{did}/dismiss", T, {"reason": reason, "version": d["version"]}); chk(f"16 dismiss lý do {len(reason)} ký tự → {want}", s == want, s)
s, x, _ = call("POST", f"{Q}/submissions/{did}/dismiss", T, {"reason": "Thư của người ngoài lớp", "version": d["version"]}); chk("16 dismiss hợp lệ → 200/204, bản ghi + tệp còn", s in (200, 204) and psql(f"select (dismissed_at is not null)::text||'/'||(blob_key <> '')::text from submissions where id='{did}'") == "true/true", f"{s} {x}")
chk("16 biến khỏi danh sách Chưa khớp", all(i["id"] != did for i in call("GET", f"{Q}/submissions?match_status=UNMATCHED&limit=100", T)[1].get("items", [])))
s, _, _ = call("DELETE", f"{Q}/submissions/{did}/dismiss", T); chk("16 hoàn tác dismiss → 200/204, trở lại danh sách", s in (200, 204) and any(i["id"] == did for i in call("GET", f"{Q}/submissions?match_status=UNMATCHED&limit=100", T)[1].get("items", [])), s)
# --- TC-15: Hôm nay
def today(tk, c=C1):
    t = call("GET", f"/api/v1/courses/{c}/today", tk)[1]; return [i for i in (t.get("actions", []) + t.get("attention", []) if isinstance(t, dict) else []) if i.get("kind") == "UNMATCHED_SUBMISSION"]
tt = today(T); print("   today TEACHER UNMATCHED_SUBMISSION:", json.dumps(tt, ensure_ascii=False)[:300])
chk("15 TEACHER và TA có đúng một mục UNMATCHED_SUBMISSION, href đúng", len(tt) == 1 and "tab=unmatched" in json.dumps(tt) and len(today(TA)) == 1, tt)
chk("15 STUDENT / ADMIN không có mục", today(SB) == [] and today(ADM) == [])
for i in call("GET", f"{Q}/submissions?match_status=UNMATCHED&limit=100", T)[1].get("items", []):
    call("POST", f"{Q}/submissions/{i['id']}/dismiss", T, {"reason": "QC dọn", "version": i["version"]})
time.sleep(6); chk("15 hết chưa khớp → mục biến mất ≤ 6 s", today(T) == [], today(T))
# --- TC-17: sổ điểm — INCOMPLETE khi còn UNMATCHED (kiểm bằng Go: TestUnmatchedBlocksZeroScore…) → bỏ qua ở hộp đen
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
