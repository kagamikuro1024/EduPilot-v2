#!/usr/bin/env python3
"""QC US-P7-09 hộp đen (máy chủ: công bố hàng loạt, thông báo, mine.result, stats, Hôm nay) trên stack thật. Provider giả → kết quả chấm FAILED; QC nhập điểm tay rồi duyệt / công bố."""
import copy, hashlib, json, os, ssl, subprocess, sys, time, urllib.request, uuid
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
T, TA, ADM, SA, SB = (tok(x + "@edupilot.local") for x in ("teacher", "ta", "admin", "sv.gioi", "sv.kha"))
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
C1 = psql("select id from courses where class_code='761987'"); Q = f"/api/v1/courses/{C1}"; idem = lambda: str(uuid.uuid4())
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:300])
base = json.load(open(f"{FX}/create-bt03.json")); base["due_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 3 * 86400)); base["submission_code"] = "QC9" + str(int(time.time()) % 100000)
s, a, _ = call("POST", f"{Q}/assignments", T, base, idem()); A = a.get("assignment", a)["id"]; call("POST", f"{Q}/assignments/{A}/publish", T, {}, idem())
def submit(tk, name, data):
    s, pre, _ = call("POST", f"{Q}/assignments/{A}/submit/presign", tk, {"filename": name, "size_bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}); raw(pre["method"], pre["url"], pre["headers"], data)
    return call("POST", f"{Q}/assignments/{A}/submit", tk, {"upload_id": pre["upload_id"]}, idem())
CAN = "CANARY-PUB-5M1"; MD = lambda t: (f"Bài làm {t}. SUNBURST. " * 12).encode()
for tk, n in ((SA, "a"), (SB, "b")): s, x, _ = submit(tk, f"{n}.md", MD(n)); assert s in (200, 201), (s, x)
call("POST", f"{Q}/assignments/{A}/grade-all", T, {}, idem())
for _ in range(60):
    if psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.run_state in ('QUEUED','RUNNING')") == "0": break
    time.sleep(2)
rows = psql(f"select gr.id||'|'||u.email||'|'||gr.version from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{A}'").splitlines(); R = {r.split('|')[1]: r.split('|') for r in rows}; print("   kết quả:", {k: v[0][:8] for k, v in R.items()})
rA, rB = R["sv.gioi@edupilot.local"][0], R["sv.kha@edupilot.local"][0]
def edit(rid, sc, fb):
    v = int(psql(f"select version from grading_results where id='{rid}'")); return call("PUT", f"{Q}/grading-results/{rid}", T, {"version": v, "criteria": [{"id": f"c{i+1}", "score": s_} for i, s_ in enumerate(sc)], "overall_feedback": fb})
s, x, _ = edit(rA, ["2.5"] * 4, f"Nhận xét A {CAN}"); chk("00 nhập điểm tay A (tổng 10)", s == 200, f"{s} {str(x)[:120]}")
s, x, _ = edit(rB, ["1", "1", "1", "1"], "Nhận xét B"); chk("00 nhập điểm tay B (tổng 4)", s == 200, f"{s} {str(x)[:120]}")
ver = lambda rid: int(psql(f"select version from grading_results where id='{rid}'"))
# --- chưa duyệt: không công bố được; sinh viên không thấy gì
pub = lambda tk, body, key=None: call("POST", f"{Q}/grading-results/publish", tk, body, key or idem())
s, x, _ = pub(T, {"result_ids": [rA]}); chk("01 công bố bài DRAFT → 422 NOT_REVIEWED kèm ids", s == 422 and "NOT_REVIEWED" in json.dumps(x), f"{s} {str(x)[:160]}")
s, m, _ = call("GET", f"{Q}/assignments/{A}/mine", SB); chk("07 DRAFT: mine.result null cho sinh viên", s == 200 and not m.get("result") and "CANARY" not in json.dumps(m), f"{s} {str(m)[:160]}")
call("POST", f"{Q}/grading-results/{rA}/review", T, {"version": ver(rA)}, idem())
s, x, _ = pub(T, {"result_ids": [rA, rB]}); chk("01 all-or-nothing: A REVIEWED + B DRAFT → 422 NOT_REVIEWED [B], không bài nào PUBLISHED", s == 422 and rB in json.dumps(x) and psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.status='PUBLISHED'") == "0", f"{s} {str(x)[:200]}")
td = call("GET", f"{Q}/today", T)[1]; kinds = [i.get("kind") for i in td.get("actions", []) + td.get("attention", [])]; print("   today TEACHER:", [k for k in kinds if k and ("PUBLISH" in k or "GRADING" in k)])
chk("10 'Hôm nay' TEACHER có PUBLISH_PENDING khi có bài REVIEWED chưa công bố", "PUBLISH_PENDING" in kinds, kinds)
td = call("GET", f"{Q}/today", TA)[1]; chk("10 TA không có PUBLISH_PENDING (chỉ TEACHER)", "PUBLISH_PENDING" not in [i.get("kind") for i in td.get("actions", []) + td.get("attention", [])])
call("POST", f"{Q}/grading-results/{rB}/review", T, {"version": ver(rB)}, idem())
# --- phân quyền
for n, tk in (("TA", TA), ("STUDENT", SB), ("ADMIN", ADM)):
    s, _, _ = pub(tk, {"result_ids": [rA, rB]}); chk(f"09 {n} công bố → 403", s == 403, s)
for n, tk in (("STUDENT", SB), ("ADMIN", ADM)):
    s, _, _ = call("GET", f"{Q}/assignments/{A}/stats", tk); chk(f"09 {n} GET stats → 403", s == 403, s)
# --- công bố
K = idem(); n0 = psql("select count(*) from notifications where type='GRADE_PUBLISHED'")
s, x, _ = pub(T, {"result_ids": [rA, rB]}, K); chk("01 TEACHER công bố 2 bài REVIEWED → 200 published 2", s == 200 and x.get("published") == 2, f"{s} {str(x)[:160]}")
s, x2, _ = pub(T, {"result_ids": [rA, rB]}, K); chk("01 cùng Idempotency-Key → cùng kết quả, không thông báo thêm", s == 200 and x2 == x, f"{s} {str(x2)[:120]}")
s, x3, _ = pub(T, {"result_ids": [rA]}); chk("01 công bố lại bài đã PUBLISHED → 409 ALREADY_PUBLISHED", s == 409 and "ALREADY_PUBLISHED" in json.dumps(x3), f"{s} {str(x3)[:140]}")
chk("05 độ lệch lưu lúc công bố (A: AI NULL → có cột deviation)", psql(f"select count(*) from information_schema.columns where table_name='grading_results' and column_name in ('deviation','teacher_changed_criteria','published_deviation')") != "", "")
print("   cột độ lệch:", psql("select string_agg(column_name, ',') from information_schema.columns where table_name='grading_results' and (column_name like '%dev%' or column_name like '%changed%')"))
chk("02 không ghi grade_entries cho bài tập", psql("select count(*) from grade_entries ge join grade_items gi on gi.id=ge.item_id where gi.assignment_id is not null") == "0", psql("select count(*) from grade_entries ge join grade_items gi on gi.id=ge.item_id where gi.assignment_id is not null"))
time.sleep(8)
# --- thông báo + thư: đúng người
for em, who in (("sv.gioi", "A"), ("sv.kha", "B")):
    n = psql(f"select count(*) from notifications nt join users u on u.id=nt.user_id where u.email='{em}@edupilot.local' and nt.type='GRADE_PUBLISHED' and nt.created_at > now() - interval '3 minutes'"); chk(f"04 {who} nhận đúng 1 chuông GRADE_PUBLISHED", n == "1", n)
chk("04 tổng chuông mới = 2 (không ai nhận bài của người khác)", int(psql("select count(*) from notifications where type='GRADE_PUBLISHED'")) - int(n0) == 2)
def mp(path): return json.loads(subprocess.run(["curl", "-s", "http://localhost:8125/api/v1/" + path], capture_output=True, text=True).stdout)
pubm = [m for m in mp("messages?limit=50")["messages"] if "Điểm bài tập đã được công bố" in m.get("Subject", "")]; to = sorted(t["Address"] for m in pubm for t in m["To"]); print("   thư công bố:", to)
chk("04 đúng 2 thư công bố, tới sv.gioi@ và sv.kha@ (email đã xác minh), mỗi người một thư", to == ["sv.gioi@edupilot.local", "sv.kha@edupilot.local"], to)
ms = subprocess.run(["curl", "-s", "http://localhost:8125/api/v1/messages?limit=20"], capture_output=True, text=True).stdout
try:
    msgs = pubm; mine = [m for m in msgs if any("sv.kha" in t.get("Address", "") for t in m.get("To", []))]
    mid = mine[0]["ID"]; body = subprocess.run(["curl", "-s", f"http://localhost:8125/api/v1/message/{mid}"], capture_output=True, text=True).stdout; mj = json.loads(body); txt = mj.get("Text", "") + mj.get("HTML", "")
    print("   thư B:", mj.get("Subject"), "|", txt[:200].replace("\n", " "))
    chk("04 thư của B không chứa điểm nháp / cờ / nhận xét của A (canary) / khoá nội bộ", CAN not in txt and "ai_" not in txt and "NEEDS_REVIEW" not in txt and "token" not in txt.lower(), txt[:200])
    chk("04 thư của B có điểm 4 / 10 (dấu phẩy chuẩn hoá)", "4" in txt and "10" in txt)
except Exception as e: chk("04 đọc thư Mailpit", False, e)
# --- mine.result
s, mB, hd = call("GET", f"{Q}/assignments/{A}/mine", SB); jb = json.dumps(mB, ensure_ascii=False)
chk("06 mine.result của B có tổng 4 / 10, bốn tiêu chí, nhãn duyệt", s == 200 and mB.get("result") and "đã được giảng viên duyệt" in jb and len(mB["result"].get("criteria", [])) == 4, f"{s} {jb[:300]}")
bad = [w for w in ("ai_scores", "flags", "confidence", "run_state", "ai_total", "pass_detail", "NEEDS_REVIEW", "deviation", CAN) if w in jb]; chk("07 mine của B không khoá nháp / độ lệch / canary của A", not bad, bad)
s, mA, _ = call("GET", f"{Q}/assignments/{A}/mine", SA); chk("06 A thấy kết quả của A (có canary nhận xét của chính mình), không thấy của B", s == 200 and CAN in json.dumps(mA, ensure_ascii=False) and "Nhận xét B" not in json.dumps(mA, ensure_ascii=False))
for p_ in (f"{Q}/me/grades", f"{Q}/assignments", f"{Q}/assignments/{A}"):
    s, x, _ = call("GET", p_, SB); t = json.dumps(x, ensure_ascii=False); chk(f"07 B GET …{p_[len(Q):]} không lộ canary / nháp của A", s == 200 and CAN not in t and "ai_scores" not in t, f"{s} {t[:120]}")
s, st, _ = call("GET", f"{Q}/assignments/{A}/stats", T); print("   stats:", json.dumps(st, ensure_ascii=False)[:300]); chk("05 stats TEACHER: published 2, tỷ lệ là chuỗi decimal / null", s == 200 and st.get("published") == 2, f"{s} {st}")
# --- sau công bố
s, x, _ = call("PUT", f"{Q}/grading-results/{rA}", T, {"version": ver(rA), "overall_feedback": "đổi"}); chk("08 sửa sau công bố → 409 RESULT_PUBLISHED", s == 409 and "RESULT_PUBLISHED" in json.dumps(x), f"{s} {str(x)[:120]}")
sid = psql(f"select submission_id from grading_results where id='{rA}'"); s, x, _ = call("POST", f"{Q}/submissions/{sid}/grade", T, {}, idem()); chk("08 chấm lại bài đã công bố → 409 ALREADY_REVIEWED", s == 409 and "ALREADY_REVIEWED" in json.dumps(x), f"{s} {str(x)[:120]}")
td = call("GET", f"{Q}/today", SB)[1]; kinds = [(i.get("kind"), i.get("title")) for i in td.get("actions", []) + td.get("attention", []) + ([td["recommended"]] if td.get("recommended") else [])]; print("   today B:", [k for k in kinds if k[0] and "GRADED" in k[0]])
chk("10 'Hôm nay' của B có ASSIGNMENT_GRADED", any(k[0] == "ASSIGNMENT_GRADED" for k in kinds), kinds)
tdt = call("GET", f"{Q}/today", T)[1]; chk("10 PUBLISH_PENDING biến mất sau khi công bố hết", "PUBLISH_PENDING" not in [i.get("kind") for i in tdt.get("actions", []) + tdt.get("attention", [])])
logs = subprocess.run("docker logs edupilot100-gateway-1 2>&1; docker logs edupilot100-worker-1 2>&1", shell=True, capture_output=True, text=True).stdout
chk("11 log gateway + worker không chứa canary nhận xét", CAN not in logs, "canary trong log")
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
