#!/usr/bin/env python3
"""QC US-P7-03 hộp đen (máy chủ) trên stack thật (TC-P7-03-03…17). Cần stack đã seed. In PASS/FAIL; thoát 1 nếu lỗi."""
import copy, hashlib, json, os, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:543"); PW = os.environ.get("PW", "Edupilot#Seed-2026"); ctx = ssl._create_unverified_context()
FX = os.path.join(os.path.dirname(__file__), "../../../../specs/FEAT-assignments/fixtures")
def raw(m, url, h=None, data=None):
    try:
        r = urllib.request.urlopen(urllib.request.Request(url, data=data, method=m, headers=h or {}), context=ctx); t = r.read(); return r.status, t, r.headers
    except urllib.error.HTTPError as e: return e.code, e.read(), e.headers
def call(m, p, tok=None, body=None, key=None):
    h = {"Content-Type": "application/json"}
    if tok: h["Authorization"] = "Bearer " + tok
    if key: h["Idempotency-Key"] = key
    s, t, hd = raw(m, GW + p, h, None if body is None else json.dumps(body).encode())
    return s, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode()), hd
tok = lambda e: call("POST", "/api/v1/auth/login", body={"email": e, "password": PW})[1]["access_token"]
T, TA, ADM, SA, SB, SD, SC = (tok(x + "@edupilot.local") for x in ("teacher", "ta", "admin", "sv.gioi", "sv.kha", "sv.moi", "sv.nguyco"))
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
C1 = psql("select id from courses where class_code='761987'"); C2 = psql("select id from courses where class_code='761988'"); Q = f"/api/v1/courses/{C1}"
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else x)
idem = lambda: str(uuid.uuid4())
base = json.load(open(f"{FX}/create-bt03.json"))
def mk(code, over=None, publish=True):
    b = copy.deepcopy(base); b.update({"submission_code": code, "due_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 3 * 86400))}); b.update(over or {})
    s, a, _ = call("POST", f"{Q}/assignments", T, b, idem()); a = a.get("assignment", a); aid = a["id"]
    if publish: call("POST", f"{Q}/assignments/{aid}/publish", T, {}, idem())
    return aid
def put_file(aid, tk, name, data, key=None):
    sha = hashlib.sha256(data).hexdigest()
    s, pre, _ = call("POST", f"{Q}/assignments/{aid}/submit/presign", tk, {"filename": name, "size_bytes": len(data), "sha256": sha})
    if s != 200: return s, pre, None
    s2, _, _ = raw(pre["method"], pre["url"], pre["headers"], data)
    s3, sub, hd = call("POST", f"{Q}/assignments/{aid}/submit", tk, {"upload_id": pre["upload_id"]}, key or idem())
    return s3, sub, (pre, s2)
PDF = b"%PDF-1.4\n" + "Bài làm: tác nhân APT29 cấy SUNBURST.\n".encode() * 5 + b"%%EOF"; MD = ("Bài làm Markdown. " * 20).encode()
A = mk("QC3A"); print("   BT QC3A", A)
# --- 03/04: kiểm tra phía máy chủ ở presign
pre = lambda name, size, sha="0" * 64, tk=SB: call("POST", f"{Q}/assignments/{A}/submit/presign", tk, {"filename": name, "size_bytes": size, "sha256": sha})
for n, a, want in [("đuôi .exe", ("a.exe", 100), "FILE_TYPE_NOT_ALLOWED"), ("20 MiB + 1", ("a.pdf", 20 * 1024 * 1024 + 1), "FILE_TOO_LARGE"), ("rỗng", ("a.pdf", 0), "FILE_EMPTY")]:
    s, x, _ = pre(*a); chk(f"03 presign {n} → 422 {want}", s == 422 and want in json.dumps(x), f"{s} {str(x)[:120]}")
s, x, _ = pre("a.PDF", 100); chk("03 đuôi .PDF hoa được nhận", s == 200, f"{s} {str(x)[:100]}")
s, x, _ = put_file(A, SB, "fake.pdf", b"MZ\x90\x00" + b"x" * 200); chk("04 .pdf nội dung MZ → 422 FILE_CONTENT_MISMATCH", s == 422 and "FILE_CONTENT_MISMATCH" in json.dumps(x), f"{s} {str(x)[:100]}")
s, x, _ = put_file(A, SB, "bad.txt", b"abc\x00def"); chk("04 .txt có NUL → 422", s == 422, f"{s} {str(x)[:100]}")
s, x, _ = put_file(A, SB, "nodocx.docx", b"khong phai zip"); chk("04 .docx không phải zip → 422", s == 422, f"{s} {str(x)[:100]}")
# --- 05: luồng thật, version, idempotent
k = idem(); s, x, (p1, putst) = put_file(A, SB, "bai1.pdf", PDF, k); chk("05 nộp lần 1 → 201, PUT 200", s in (200, 201) and putst == 200, f"{s} {putst} {str(x)[:140]}")
s2, x2, _ = call("POST", f"{Q}/assignments/{A}/submit", SB, {"upload_id": p1["upload_id"]}, k); chk("07 gửi lại cùng Idempotency-Key → cùng kết quả", s2 in (200, 201) and json.dumps(x2, sort_keys=True) == json.dumps(x, sort_keys=True), f"{s2} {str(x2)[:100]}")
chk("05 đúng 1 phiên bản, source IN_APP, MATCHED, student = B", psql(f"select count(*)||'/'||max(version)||'/'||max(source)||'/'||max(match_status) from submissions where assignment_id='{A}'") == "1/1/IN_APP/MATCHED")
chk("05 submitted_at do máy chủ (≈ now)", psql(f"select (abs(extract(epoch from now()-submitted_at)) < 30) from submissions where assignment_id='{A}' limit 1") == "t")
chk("05 sinh viên của bài = B (từ JWT)", psql(f"select u.email from submissions s join users u on u.id=s.student_id where s.assignment_id='{A}'") == "sv.kha@edupilot.local")
s, x, _ = put_file(A, SB, "bai2.md", MD); chk("10 nộp lại → phiên bản 2, bản 1 còn", s in (200, 201) and psql(f"select string_agg(version::text, ',' order by version) from submissions where assignment_id='{A}'") == "1,2", f"{s} {str(x)[:100]}")
# --- 08/17: mine
s, m, hd = call("GET", f"{Q}/assignments/{A}/mine", SB); txt = json.dumps(m, ensure_ascii=False)
chk("17 mine: no-store", "no-store" in (hd.get("Cache-Control") or ""), str(hd.get("Cache-Control")))
bad = [w for w in ("ai_scores", "flags", "confidence", "run_state", "reference_answer", "guidance", "ai_total", "pass_detail") if w in txt]; chk("17 mine: không khoá nháp / đáp án", s == 200 and not bad, f"{s} {bad} {txt[:200]}")
print("   mine keys:", sorted(m.keys()) if isinstance(m, dict) else m)
s, m2, _ = call("GET", f"{Q}/assignments/{A}/mine?student_id=" + psql("select id from users where email='sv.gioi@edupilot.local'"), SB); chk("16 mine bỏ qua ?student_id (vẫn của B)", s == 200 and json.dumps(m2, sort_keys=True) == json.dumps(m, sort_keys=True))
for n, tk, want in [("TA gọi mine", TA, 403), ("TEACHER gọi mine", T, 403), ("ADMIN gọi mine", ADM, 403)]:
    s, _, _ = call("GET", f"{Q}/assignments/{A}/mine", tk); chk(f"16 {n} → {want}", s == want, str(s))
for n, m_, p_ in [("submissions", "GET", f"{Q}/submissions")]:  # grade-all thuộc US-P7-07 (route chưa có)
    s, _, _ = call(m_, p_, SB, {} if m_ == "POST" else None, idem() if m_ == "POST" else None); chk(f"16 STUDENT gọi {n} → 403", s == 403, str(s))
# --- 17: nháp đặt canary rồi quét mine
sid = psql(f"select id from submissions where assignment_id='{A}' and version=2")
psql(f"insert into grading_results(course_id,submission_id,run_state,status,flags,ai_scores,ai_total_before_late,criteria,overall_feedback,confidence) values ('{C1}','{sid}','DONE','DRAFT','{{NEEDS_REVIEW}}','{{\"c1\":\"7.77\"}}',7.77,'[{{\"id\":\"c1\",\"comment\":\"CANARY-DRAFT-7K3\"}}]','CANARY-DRAFT-7K3',0.5)")
for p_ in (f"{Q}/assignments/{A}/mine", f"{Q}/assignments", f"{Q}/assignments/{A}", f"{Q}/me/grades"):
    s, x, _ = call("GET", p_, SB); t = json.dumps(x, ensure_ascii=False); chk(f"17 DRAFT có canary → B GET …{p_[len(Q):]} không lộ", s == 200 and "CANARY-DRAFT-7K3" not in t and "7.77" not in t and "7,77" not in t and "ai_scores" not in t, f"{s} {t[:160]}")
s, xd, _ = call("POST", f"{Q}/assignments/{A}/submit/presign", SB, {"filename": "x.pdf", "size_bytes": 10, "sha256": "0" * 64})
# --- 11: REVIEWED/PUBLISHED → SUBMISSION_LOCKED câu chung (so với DRAFT)
time.sleep(62)  # cửa sổ hạn mức presign 10 / phút / người
msgs = {}
for st in ("DRAFT", "REVIEWED", "PUBLISHED"):
    cur = psql(f"select id from submissions where assignment_id='{A}' order by version desc limit 1")  # khoá tính theo bản MỚI NHẤT
    psql(f"update grading_results set submission_id='{cur}' where submission_id='{sid}'"); sid = cur
    TID = psql("select id from users where email='teacher@edupilot.local'")
    stmt = {"DRAFT": f"update grading_results set status='DRAFT', published_at=NULL, published_by=NULL where submission_id='{sid}'",
            "REVIEWED": f"update grading_results set status='REVIEWED', published_at=NULL, published_by=NULL where submission_id='{sid}'",
            "PUBLISHED": f"update grading_results set status='PUBLISHED', published_at=now(), published_by='{TID}' where submission_id='{sid}'"}[st]
    psql(stmt)
    s, x, _ = put_file(A, SB, "lai.md", MD + st.encode()); msgs[st] = (s, json.dumps(x, ensure_ascii=False)[:300])
    print("   nộp lại khi", st, "→", msgs[st][0], msgs[st][1][:140])
chk("11 DRAFT: nộp lại được (v3)", msgs["DRAFT"][0] in (200, 201))
chk("11 REVIEWED / PUBLISHED → 409 SUBMISSION_LOCKED", msgs["REVIEWED"][0] == 409 and "SUBMISSION_LOCKED" in msgs["REVIEWED"][1] and msgs["PUBLISHED"][0] == 409 and "SUBMISSION_LOCKED" in msgs["PUBLISHED"][1])
bw = [w for w in ("duyệt", "chấm", "REVIEWED", "PUBLISHED", "điểm") if w in msgs["REVIEWED"][1] or w in msgs["PUBLISHED"][1]]; chk("11 câu lỗi CHUNG (không nêu duyệt / chấm / điểm / trạng thái)", not bw and json.loads(msgs["REVIEWED"][1][:300].rsplit(", \"trace_id\"",1)[0] + "}")["message"] == json.loads(msgs["PUBLISHED"][1][:300].rsplit(", \"trace_id\"",1)[0] + "}")["message"], str(bw))
# --- 12: muộn
def due_ago(aid, interval, allow=True, days=2):
    psql(f"update assignments set due_at = now() - interval '{interval}', allow_late = {str(allow).lower()}, late_max_days = {days}, late_penalty_per_day = 0.5 where id='{aid}'")
rows = []
for iv, want in [("8 hours 11 minutes", "LATE1"), ("40 hours", "LATE2"), ("47 hours 59 minutes", "LATE2"), ("48 hours 1 minute", "CLOSED"), ("1 minute", "LATE1")]:
    L = mk("QL" + str(len(rows)) , {"allow_late": True, "late_max_days": 2}); due_ago(L, iv); s, x, _ = put_file(L, SC, "m.md", MD)
    st = psql(f"select count(*) from submissions where assignment_id='{L}'")
    got = "CLOSED" if s == 409 and "SUBMISSION_CLOSED" in json.dumps(x) else None
    s_m, mm, _ = call("GET", f"{Q}/assignments/{L}/mine", SC); t = json.dumps(mm, ensure_ascii=False); import re as _re
    ld = _re.search(r'"late_days":\s*(\d+)', t); got = got or ("LATE" + (ld.group(1) if ld else "?"))
    chk(f"12 trễ {iv} → {want}", got == want and (st == "0") == (want == "CLOSED"), f"{got} s={s} rows={st} {t[:120]}"); rows.append(L)
L = mk("QL9", {"allow_late": False}); psql(f"update assignments set due_at = now() - interval '1 minute', allow_late=false, late_max_days=0 where id='{L}'")
s, x, _ = put_file(L, SC, "n.md", MD); chk("12 allow_late = false sau hạn → 409 SUBMISSION_CLOSED, không bản ghi", s == 409 and "SUBMISSION_CLOSED" in json.dumps(x) and psql(f"select count(*) from submissions where assignment_id='{L}'") == "0", f"{s} {str(x)[:100]}")
psql(f"update assignments set due_at = now() + interval '5 days' where id='{rows[0]}'"); s_m, mm, _ = call("GET", f"{Q}/assignments/{rows[0]}/mine", SC); chk("13 gia hạn hạn nộp → late_days hiển thị 0 (tính lại)", ('"late_days": 0' in json.dumps(mm) or "late_days" not in json.dumps(mm)), json.dumps(mm)[:160])
chk("13 submissions không có cột late_days / is_late", psql("select count(*) from information_schema.columns where table_name='submissions' and column_name in ('late_days','is_late')") == "0")
# --- 15: bài chưa công bố / kênh tắt / lớp khác
U = mk("QC3U", publish=False); s, x, _ = put_file(U, SC, "a.md", MD); chk("15 bài chưa công bố → 404 ASSIGNMENT_NOT_FOUND", s == 404 and "ASSIGNMENT_NOT_FOUND" in json.dumps(x), f"{s} {str(x)[:100]}")
CH = mk("QC3C", {"channels": ["UPLOAD"]}); s, x, _ = put_file(CH, SC, "a.md", MD); chk("15 kênh IN_APP tắt → 409 CHANNEL_NOT_ALLOWED", s == 409 and "CHANNEL_NOT_ALLOWED" in json.dumps(x), f"{s} {str(x)[:100]}")
s, x, _ = call("POST", f"/api/v1/courses/{C2}/assignments/{A}/submit/presign", SB, {"filename": "a.md", "size_bytes": 10, "sha256": "0" * 64}); chk("15 B gọi route lớp 2 (không thuộc) → 403", s == 403, str(s))
s, x, _ = call("POST", f"{Q}/assignments/{A}/submit/presign", SD, {"filename": "a.md", "size_bytes": 10, "sha256": "0" * 64}); chk("15 sinh viên chưa vào lớp → 403", s == 403, str(s))
# --- 06: rate limit presign 10 / phút / người
codes = [call("POST", f"{Q}/assignments/{A}/submit/presign", SA, {"filename": "r.md", "size_bytes": 10, "sha256": "0" * 64})[0] for _ in range(12)]
chk("08 presign 10 lượt / phút / người: có 429 sau lượt thứ 10", codes[:10].count(200) >= 9 and 429 in codes[10:], str(codes))
s, _, _ = call("POST", f"{Q}/assignments/{A}/submit/presign", SB, {"filename": "r.md", "size_bytes": 10, "sha256": "0" * 64}); print("   người khác không bị ảnh hưởng (B):", s)
# --- 19: Hôm nay
td = call("GET", f"{Q}/today", SA)[1]; print("   today sv.gioi kinds:", [i.get("kind") for i in td.get("items", [])] if isinstance(td, dict) else td)
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
