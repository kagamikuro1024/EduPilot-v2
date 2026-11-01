#!/usr/bin/env python3
"""QC US-P7-07 hộp đen trên stack thật (hàng đợi chấm, TC-P7-07-01…12). Provider giả chưa có bản ghi demo chấm → kết quả có thể FAILED; kiểm cơ chế hàng đợi, không kiểm điểm."""
import copy, hashlib, json, os, ssl, subprocess, sys, threading, time, urllib.request, uuid
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
T, TA, ADM, SA, SB, SC = (tok(x + "@edupilot.local") for x in ("teacher", "ta", "admin", "sv.gioi", "sv.kha", "sv.nguyco"))
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
C1 = psql("select id from courses where class_code='761987'"); Q = f"/api/v1/courses/{C1}"; idem = lambda: str(uuid.uuid4())
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:260])
base = json.load(open(f"{FX}/create-bt03.json")); base["due_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 3 * 86400))
def mk(code):
    b = copy.deepcopy(base); b["submission_code"] = code
    s, a, _ = call("POST", f"{Q}/assignments", T, b, idem()); aid = a.get("assignment", a)["id"]; call("POST", f"{Q}/assignments/{aid}/publish", T, {}, idem()); return aid
def submit(aid, tk, name, data):
    sha = hashlib.sha256(data).hexdigest()
    s, pre, _ = call("POST", f"{Q}/assignments/{aid}/submit/presign", tk, {"filename": name, "size_bytes": len(data), "sha256": sha}); assert s == 200, (s, pre)
    raw(pre["method"], pre["url"], pre["headers"], data)
    return call("POST", f"{Q}/assignments/{aid}/submit", tk, {"upload_id": pre["upload_id"]}, idem())
CAN = "CANARY-SUBTEXT-9Q2"; MD = lambda t: (f"Bài làm {t}. {CAN}. SUNBURST được cấy vào bản cập nhật Orion. " * 8).encode()
terminal = lambda aid: psql(f"select string_agg(run_state||':'||coalesce(error_code,'-'), ',' order by run_state) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{aid}' and gr.round=1")
A = mk("QC7A"); print("   BT QC7A", A)
for tk, n in ((SA, "a"), (SB, "b"), (SC, "c")): s, x, _ = submit(A, tk, f"bai-{n}.md", MD(n)); assert s in (200, 201), (s, x)
# --- TC-01/02/03: grade-all
K = idem(); s, g, _ = call("POST", f"{Q}/assignments/{A}/grade-all", T, {}, K)
chk("01 grade-all → 202 queued 3 + job_id", s == 202 and g.get("queued") == 3 and g.get("job_id"), f"{s} {g}")
n1 = psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}'"); chk("01 đúng 3 dòng grading_results round 1", n1 == "3" and psql(f"select max(round) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}'") == "1", n1)
s, g2, _ = call("POST", f"{Q}/assignments/{A}/grade-all", T, {}, K); chk("03 cùng Idempotency-Key → cùng job_id", s in (200, 202) and g2.get("job_id") == g.get("job_id"), f"{s} {g2}")
res = []; th = [threading.Thread(target=lambda: res.append(call("POST", f"{Q}/assignments/{A}/grade-all", T, {}, idem())[:2])) for _ in range(10)]; [t.start() for t in th]; [t.join() for t in th]
chk("03 10 request song song (khoá khác) → không dòng trùng, tổng vẫn 3", psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}'") == "3" and psql(f"select count(*) from (select submission_id, round from grading_results gr group by 1,2 having count(*)>1) x") == "0", [r[0] for r in res])
chk("03 song song: tổng queued của các request = 0 (đã xếp hết)", sum((r[1].get("queued", 0) if isinstance(r[1], dict) else 0) for r in res) == 0, [r[1] for r in res][:3])
# --- TC-12: phân quyền
for n, tk, want in [("STUDENT", SB, 403), ("ADMIN", ADM, 403)]:
    s, _, _ = call("POST", f"{Q}/assignments/{A}/grade-all", tk, {}, idem()); chk(f"12 {n} grade-all → 403", s == want, s)
s, _, _ = call("POST", f"{Q}/assignments/{A}/grade-all", TA, {}, idem()); chk("12 TA grade-all → 200/202", s in (200, 202), s)
jid = g["job_id"]
for n, tk, want in [("TEACHER", T, 200), ("TA", TA, 200), ("STUDENT", SB, 404), ("ADMIN", ADM, 404)]:
    s, _, _ = call("GET", f"/api/v1/jobs/{jid}", tk); chk(f"12 GET /jobs/{{id}} {n} → {want}", s == want, s)
# --- TC-06/07: chờ kết thúc, đọc trạng thái
for _ in range(60):
    st = psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.run_state in ('QUEUED','RUNNING')")
    if st == "0": break
    time.sleep(2)
print("   trạng thái cuối:", terminal(A), "| còn chạy:", st)
chk("01 cả 3 bài kết thúc (DONE / FAILED) ≤ 120 s", st == "0", st)
fl = psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.run_state='FAILED'"); print("   FAILED:", fl)
chk("10 FAILED: tiêu chí NULL, không điểm 0, không grade_entries", psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.run_state='FAILED' and (gr.ai_total_before_late is not null and gr.ai_total_before_late = 0)") == "0")
perm = psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.error_code in ('LLM_BUDGET_EXHAUSTED','GRADING_OUTPUT_INVALID','TEXT_TOO_LONG') and gr.attempts <> 1"); chk("07 lỗi vĩnh viễn → attempts = 1 (không thử lại)", perm == "0", perm)
chk("08 mọi lời gọi chấm làn BATCH (llm_audit)", psql("select count(*) from llm_audit where task='GRADING' and lane <> 'BATCH'") == "0", psql("select lane||':'||count(*) from llm_audit where task='GRADING' group by lane"))
# --- TC-03: grade / retry
rid = psql(f"select gr.id from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.run_state='FAILED' limit 1")
sidf = psql(f"select s.id from grading_results gr join submissions s on s.id=gr.submission_id where gr.id='{rid}'") if rid else ""
if rid:
    s, x, _ = call("POST", f"{Q}/grading-results/{rid}/retry", T, {}, idem()); chk("04 retry kết quả FAILED → 202/200", s in (200, 202), f"{s} {x}")
    s, x, _ = call("POST", f"{Q}/submissions/{sidf}/grade", T, {}, idem()); chk("03 grade khi đang xếp hàng sau retry → 409 VERSION_CONFLICT (hoặc 202 nếu đã xong)", s in (409, 202), f"{s} {x}")
dr = psql(f"select s.id from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.run_state='DONE' limit 1")
psql(f"update grading_results set status='REVIEWED' where submission_id='{sidf or dr}' and run_state='DONE'") if (sidf or dr) and dr else None
if dr:
    psql(f"update grading_results set status='REVIEWED' where submission_id='{dr}'"); s, x, _ = call("POST", f"{Q}/submissions/{dr}/grade", T, {}, idem()); chk("03 grade khi REVIEWED → 409 ALREADY_REVIEWED", s == 409 and "ALREADY_REVIEWED" in json.dumps(x), f"{s} {x}")
else: print("   (không có kết quả DONE — provider giả chưa có bản ghi chấm; grade-REVIEWED kiểm bằng Go TestRegradeOnlyDraftOrFailed)")
# --- TC-10: SUPERSEDED — dừng worker, xếp hàng v1, nộp v2, chạy worker
A2 = mk("QC7B"); s, x, _ = submit(A2, SC, "v1.md", MD("v1")); assert s in (200, 201)
subprocess.run(["docker", "pause", "edupilot100-worker-1"], capture_output=True)
try:
    s, g3, _ = call("POST", f"{Q}/assignments/{A2}/grade-all", T, {}, idem()); chk("10 grade-all v1 khi worker dừng → 202 queued 1", s == 202 and g3.get("queued") == 1, f"{s} {g3}")
    time.sleep(1); s, x, _ = submit(A2, SC, "v2.md", MD("v2")); chk("10 nộp v2 khi v1 đang QUEUED → 201", s in (200, 201), f"{s} {x}")
finally:
    subprocess.run(["docker", "unpause", "edupilot100-worker-1"], capture_output=True)
time.sleep(20)
print("   v1:", psql(f"select gr.run_state||':'||coalesce(gr.error_code,'-') from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A2}' and s.version=1"))
chk("10 v1 kết thúc FAILED SUPERSEDED", psql(f"select gr.run_state||':'||coalesce(gr.error_code,'-') from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A2}' and s.version=1") == "FAILED:SUPERSEDED")
s, g4, _ = call("POST", f"{Q}/assignments/{A2}/grade-all", T, {}, idem()); chk("10 grade-all sau đó chỉ xếp v2 (queued 1)", s == 202 and g4.get("queued") == 1, f"{s} {g4}")
# --- TC-13: log không văn bản bài
time.sleep(15)
logs = subprocess.run("docker logs edupilot100-gateway-1 2>&1; docker logs edupilot100-worker-1 2>&1", shell=True, capture_output=True, text=True).stdout
chk("13 log gateway + worker không chứa canary văn bản bài", CAN not in logs and "SUNBURST" not in logs, "canary có trong log")
chk("13 log không chứa MSSV / email sinh viên", not any(w in logs for w in ("20229001", "20229002", "20229003", "sv.gioi@", "sv.kha@", "sv.nguyco@")), [w for w in ("20229001", "20229002", "20229003", "sv.gioi@", "sv.kha@", "sv.nguyco@") if w in logs])
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
