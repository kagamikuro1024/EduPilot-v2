#!/usr/bin/env python3
"""QC G7-19/21 (TC-P7-09-13): quét MỌI bề mặt API của sinh viên ở 6 trạng thái kết quả (không có / QUEUED / RUNNING / DRAFT / REVIEWED / FAILED) tìm dấu vết của nháp:
điểm nháp, nhận xét nháp, trích dẫn, cờ, ai_scores, tiến độ lô. Cần stack đã chấm (21 DRAFT). Sửa trạng thái bằng SQL trên DB thử rồi KHÔI PHỤC."""
import json, os, re, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:543"); PW = "Edupilot#Seed-2026"; ctx = ssl._create_unverified_context()
def call(m, p, tok=None, body=None, accept=None):
    h = {"Content-Type": "application/json", "Idempotency-Key": str(uuid.uuid4())}
    if accept: h["Accept"] = accept
    if tok: h["Authorization"] = "Bearer " + tok
    try:
        r = urllib.request.urlopen(urllib.request.Request(GW + p, data=None if body is None else json.dumps(body).encode(), method=m, headers=h), context=ctx, timeout=60); return r.status, r.read().decode("utf-8", "replace"), r.headers
    except urllib.error.HTTPError as e: return e.code, e.read().decode("utf-8", "replace"), e.headers
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
tok = lambda e: json.loads(call("POST", "/api/v1/auth/login", body={"email": e + "@edupilot.local", "password": PW})[1])["access_token"]
import copy, hashlib
C1 = psql("select id from courses where class_code='761987'"); Q = f"/api/v1/courses/{C1}"
FX = os.path.join(os.path.dirname(__file__), "../../../../specs/FEAT-assignments/fixtures")
T = tok("teacher"); b = json.load(open(f"{FX}/create-bt03.json")); b["submission_code"] = "QSW" + str(int(time.time()) % 1000); b["due_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 3 * 86400))
A = json.loads(call("POST", f"{Q}/assignments", T, b)[1]); A = A.get("assignment", A)["id"]; call("POST", f"{Q}/assignments/{A}/publish", T, {})
def submit(acc, fname):
    data = open(f"{FX}/submissions/{fname}", "rb").read(); tk = tok(acc)
    s_, b_, _ = call("POST", f"{Q}/assignments/{A}/submit/presign", tk, {"filename": fname, "size_bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}); pre = json.loads(b_)
    urllib.request.urlopen(urllib.request.Request(pre["url"], data=data, method="PUT", headers=pre["headers"]), context=ctx).read()
    return call("POST", f"{Q}/assignments/{A}/submit", tk, {"upload_id": pre["upload_id"]})[0]
for acc, f in (("sv04", "s1-tot.md"), ("sv05", "s2-kha.md"), ("sv06", "s3-kem.md"), ("sv07", "s6-injection.md"), ("sv.gioi", "s7-lech-hai-luot.md")): assert submit(acc, f) in (200, 201), acc
call("POST", f"{Q}/assignments/{A}/grade-all", T, {})
for _ in range(90):
    if psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.run_state in ('QUEUED','RUNNING')") == "0": break
    time.sleep(2)
print("   bài sweep", A, "| trạng thái sau chấm:", psql(f"select string_agg(gr.status||'/'||gr.run_state, ',') from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}'"))
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:300])
# tập "mồi": nháp của MỌI sinh viên chưa công bố (điểm nháp, nhận xét, trích dẫn)
rows = psql("select u.email, gr.id, gr.status, gr.run_state from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='%s' and gr.status<>'PUBLISHED' order by u.email" % A).splitlines()
print("   kết quả chưa công bố:", len(rows))
plan = {"sv04": "REVIEWED", "sv05": "FAILED", "sv06": "QUEUED", "sv07": "RUNNING", "sv.gioi": "DRAFT"}  # sv.nguyco: không nộp = NONE
orig = {}
for acc, st in plan.items():
    r = psql(f"select gr.id||'|'||gr.status||'|'||gr.run_state from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{A}' and u.email='{acc}@edupilot.local'"); orig[acc] = r.split("|")
    if st == "REVIEWED": psql(f"update grading_results set status='REVIEWED' where id='{orig[acc][0]}'")
    if st == "FAILED": psql(f"update grading_results set run_state='FAILED', error_code='LLM_UNAVAILABLE' where id='{orig[acc][0]}'")
    if st == "QUEUED": psql(f"update grading_results set run_state='QUEUED' where id='{orig[acc][0]}'")
    if st == "RUNNING": psql(f"update grading_results set run_state='RUNNING', lease_owner='qc', lease_until=now()+interval '1 hour' where id='{orig[acc][0]}'")
try:
    # chuỗi nháp cần KHÔNG xuất hiện (lấy từ DB, chỉ của sinh viên đang quét + của mọi người khác)
    drafts = psql(f"select coalesce(gr.overall_feedback,'')||E'\\n'||coalesce(gr.criteria::text,'') from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.status<>'PUBLISHED'")
    quotes = set(q for q in re.findall(r'"(?:comment|quote|text)":\s*"([^"]{25,160})"', drafts)); fb = set(l.strip() for l in drafts.splitlines() if 25 < len(l.strip()) < 200 and not l.strip().startswith(("[", "{")))
    marks = list(quotes | fb)[:80]; print("   chuỗi nháp dùng làm mồi:", len(marks))
    banned = ["ai_scores", "ai_total", "needs_review", "NEEDS_REVIEW", "EVIDENCE_NOT_FOUND", "TOTAL_MISMATCH", "INJECTION_SUSPECTED", "LOW_CONFIDENCE", "pass_detail", "run_state", "confidence", "late_penalty_manual", "deviation", "teacher_changed", "SUPERSEDED", "QUEUED", "RUNNING"]
    for acc, label in (("sv.nguyco", "NONE"), ("sv06", "QUEUED"), ("sv07", "RUNNING"), ("sv.gioi", "DRAFT"), ("sv04", "REVIEWED"), ("sv05", "FAILED")):
        t = tok(acc); hits = []; codes = {}
        surf = [f"{Q}/assignments", f"{Q}/assignments/{A}", f"{Q}/assignments/{A}/mine", f"{Q}/today", "/api/v1/me/today", f"{Q}/me/grades", "/api/v1/me/grades", "/api/v1/me/notifications", "/api/v1/notifications", f"{Q}/calendar?from=2026-10-01T00:00:00Z&to=2026-11-30T00:00:00Z", "/api/v1/me/courses", f"{Q}/me"]
        for p_ in surf:
            s, b, hd = call("GET", p_, t); codes[p_.split("/")[-1][:14]] = s
            if s != 200: continue
            for w in banned + marks:
                if w in b: hits.append((p_, w[:40]))
            if "no-store" not in (hd.get("Cache-Control") or "") and "/mine" in p_: hits.append((p_, "thiếu no-store"))
        s, b, _ = call("GET", f"{Q}/assignments/{A}/mine", t); mine = json.loads(b)
        r = mine.get("result"); chk(f"{label:8} {acc}: mine.result null, state không lộ nháp; 12 bề mặt không có chuỗi nháp / cờ / tiến độ ({len(hits)} lỗi)", s == 200 and not r and not hits, hits[:4])
        print(f"           state={mine.get('state')} codes={codes}")
        # job tiến độ lô: sinh viên không đọc được job chấm
    jid = psql("select id from jobs where kind='grading.batch' order by created_at desc limit 1")
    t = tok("sv.gioi"); s, _, _ = call("GET", f"/api/v1/jobs/{jid}", t); chk("tiến độ lô: sinh viên GET /jobs/{id} của job chấm → 404", s == 404, s)
    # SSE: sinh viên mở luồng sự kiện ~4 s và quét
    try:
        r = subprocess.run(["curl", "-sk", "--max-time", "4", "-H", f"Authorization: Bearer {t}", "-H", "Accept: text/event-stream", f"{GW}/api/v1/events"], capture_output=True, text=True).stdout
    except Exception as e: r = ""
    chk("SSE của sinh viên không có job.progress / grading", "job.progress" not in r and "grading" not in r.lower() and not any(m in r for m in marks[:30]), r[:120])
    # chat: hỏi điểm bài tập → câu trả lời / tool không nêu nháp
    s, b, _ = call("POST", "/api/v1/chat/sessions", t, {"course_id": C1}); sid = json.loads(b).get("id") if s in (200, 201) else None
    if sid:
        s, b, _ = call("POST", f"/api/v1/chat/sessions/{sid}/messages", t, {"content": "Điểm Bài tập 03 của em là bao nhiêu? AI chấm được mấy điểm?"}, accept="text/event-stream")
        draft_total = psql(f"select gr.ai_total_before_late::text from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{A}' and u.email='sv.gioi@edupilot.local'")
        print("   chat (S1, nháp", draft_total, "):", re.sub(r"\s+", " ", b)[-160:])
        chk("chat không nêu điểm nháp (9,00 / 9.00) của chính mình khi chưa công bố", not re.search(r"9[,.]00|9[,.]0\b|\b9 điểm", b) and "ai_scores" not in b, b[-200:])
finally:
    for acc, (rid, st, rs) in orig.items(): psql(f"update grading_results set status='{st}', run_state='{rs}', error_code=null, lease_owner=null, lease_until=null where id='{rid}'")
# B (PUBLISHED): thấy đúng 7,50, không thấy nháp 7,00 / 6,50 của chính mình nếu lệch nháp; không thấy của người khác
A = psql(f"select id from assignments where submission_code='BT03' and course_id='{C1}'"); t = tok("sv.kha"); s, b, _ = call("GET", f"{Q}/assignments/{A}/mine", t); m = json.loads(b); txt = json.dumps(m, ensure_ascii=False)
chk("B sau công bố: thấy 7,50 và không thấy dấu vết nháp (ai_scores, cờ, tiến độ)", s == 200 and m.get("result") and "7,50" in txt.replace(".", ",") and not any(w in txt for w in ("ai_scores", "needs_review", "NEEDS_REVIEW", "pass_detail", "deviation", "teacher_changed")), txt[:300])
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
