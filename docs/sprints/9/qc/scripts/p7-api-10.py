#!/usr/bin/env python3
"""QC US-P7-10 hộp đen (máy chủ: phúc khảo bài tập) trên stack thật. Dựng: 3 sinh viên nộp → nhập điểm tay → duyệt → công bố (API); rồi gửi / quyết định phúc khảo."""
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
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:300])
base = json.load(open(f"{FX}/create-bt03.json")); base["due_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 3 * 86400)); base["submission_code"] = "QC10" + str(int(time.time()) % 10000); base["appeal_days"] = 7
s, a, _ = call("POST", f"{Q}/assignments", T, base, idem()); A = a.get("assignment", a)["id"]; call("POST", f"{Q}/assignments/{A}/publish", T, {}, idem())
def submit(tk, name, data):
    s, pre, _ = call("POST", f"{Q}/assignments/{A}/submit/presign", tk, {"filename": name, "size_bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}); raw(pre["method"], pre["url"], pre["headers"], data)
    return call("POST", f"{Q}/assignments/{A}/submit", tk, {"upload_id": pre["upload_id"]}, idem())
MD = lambda t: (f"Bài làm {t}. SUNBURST. " * 12).encode()
for tk, n in ((SA, "a"), (SB, "b"), (SC, "c")): s, x, _ = submit(tk, f"{n}.md", MD(n)); assert s in (200, 201), (s, x)
call("POST", f"{Q}/assignments/{A}/grade-all", T, {}, idem())
for _ in range(60):
    if psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{A}' and gr.run_state in ('QUEUED','RUNNING')") == "0": break
    time.sleep(2)
R = {r.split("|")[1]: r.split("|")[0] for r in psql(f"select gr.id||'|'||u.email from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{A}'").splitlines()}
rA, rB, rC = R["sv.gioi@edupilot.local"], R["sv.kha@edupilot.local"], R["sv.nguyco@edupilot.local"]
ver = lambda rid: int(psql(f"select version from grading_results where id='{rid}'"))
def edit(rid, sc): return call("PUT", f"{Q}/grading-results/{rid}", T, {"version": ver(rid), "criteria": [{"id": f"c{i+1}", "score": s_} for i, s_ in enumerate(sc)], "overall_feedback": "Nhận xét tay"})
for r, sc in ((rA, ["2.5"] * 4), (rB, ["1", "1", "1", "1"]), (rC, ["2", "2", "2", "2"])):
    assert edit(r, sc)[0] == 200; call("POST", f"{Q}/grading-results/{r}/review", T, {"version": ver(r)}, idem())
s, x, _ = call("POST", f"{Q}/grading-results/publish", T, {"result_ids": [rA, rB, rC]}, idem()); assert s == 200, (s, x)
print("   bài", A, "đã công bố 3 kết quả")
if os.environ.get("STOP_AFTER_PUBLISH"): open("/tmp/p710-ids", "w").write(f"{C1} {A} {rA} {rB} {rC}"); print("   DỪNG sau khi công bố (STOP_AFTER_PUBLISH)"); sys.exit(0)
tk0 = psql("select count(*) from escalation_tickets"); llm0 = psql("select count(*) from llm_audit")
appeal = lambda tk, body: call("POST", f"{Q}/assignments/{A}/appeal", tk, body, idem())
for n, body, code in [("lý do 9 ký tự", {"criteria_ids": ["c2"], "reason": "x" * 9}, "REASON"), ("lý do 1.001 ký tự", {"criteria_ids": ["c2"], "reason": "x" * 1001}, "REASON"), ("không chọn tiêu chí", {"criteria_ids": [], "reason": "x" * 20}, ""), ("tiêu chí lạ", {"criteria_ids": ["zz"], "reason": "x" * 20}, "CRITERION_UNKNOWN")]:
    s, x, _ = appeal(SB, body); chk(f"02 {n} → 422", s == 422 and (not code or code in json.dumps(x)), f"{s} {str(x)[:120]}")
s, x, _ = appeal(SB, {"criteria_ids": ["c2"], "reason": "x" * 10}); chk("01 B gửi (lý do 10 ký tự, tiêu chí c2) → 201 OPEN", s == 201 and x["appeal"]["status"] == "OPEN", f"{s} {str(x)[:140]}"); apB = x["appeal"]["id"] if s == 201 else None
s, x, _ = appeal(SB, {"criteria_ids": ["c2"], "reason": "x" * 20}); chk("03 gửi lần hai → 409 APPEAL_ALREADY_FILED", s == 409 and "APPEAL_ALREADY_FILED" in json.dumps(x), f"{s} {str(x)[:120]}")
chk("01 một dòng grade_appeals, không escalation_tickets, 0 lời gọi LLM", psql(f"select count(*) from grade_appeals where assignment_id='{A}'") == "1" and psql("select count(*) from escalation_tickets") == tk0 and psql("select count(*) from llm_audit") == llm0, (psql("select count(*) from escalation_tickets"), tk0))
# --- hạn: mốc -7 ngày
psql(f"update grading_results set published_at = now() - interval '7 days 1 minute' where id='{rA}'"); s, x, _ = appeal(SA, {"criteria_ids": ["c1"], "reason": "x" * 20}); chk("03 quá hạn −7 ngày −1 phút → 409 APPEAL_WINDOW_CLOSED kèm closes_at", s == 409 and "APPEAL_WINDOW_CLOSED" in json.dumps(x), f"{s} {str(x)[:160]}")
psql(f"update grading_results set published_at = now() - interval '6 days 23 hours 59 minutes' where id='{rC}'"); s, x, _ = appeal(SC, {"criteria_ids": ["c3"], "reason": "y" * 20}); chk("03 đúng hạn −7 ngày +1 phút → 201", s == 201, f"{s} {str(x)[:140]}"); apC = x["appeal"]["id"] if s == 201 else None
# --- chưa công bố → 404
A2base = copy.deepcopy(base); A2base["submission_code"] = "QC10U" + str(int(time.time()) % 1000)
s, a2, _ = call("POST", f"{Q}/assignments", T, A2base, idem()); A2 = a2.get("assignment", a2)["id"]; call("POST", f"{Q}/assignments/{A2}/publish", T, {}, idem())
s, x, _ = call("POST", f"{Q}/assignments/{A2}/appeal", SB, {"criteria_ids": ["c1"], "reason": "x" * 20}, idem()); chk("03 bài chưa công bố / chưa có kết quả → 404 ASSIGNMENT_NOT_FOUND", s == 404, f"{s} {str(x)[:100]}")
# --- phân quyền
for n, tk, want in (("STUDENT", SB, 403), ("ADMIN", ADM, 403), ("TA", TA, 200), ("TEACHER", T, 200)):
    s, _, _ = call("GET", f"{Q}/appeals?assignment_id={A}", tk); chk(f"10 {n} GET appeals → {want}", s == want, s)
for n, tk in (("STUDENT A", SA), ("ADMIN", ADM)):
    s, _, _ = call("GET", f"{Q}/appeals/{apB}", tk); chk(f"10 {n} đọc phúc khảo của B → 403", s == 403, s)
for n, tk in (("TA", TA), ("STUDENT", SB), ("ADMIN", ADM)):
    s, _, _ = call("POST", f"{Q}/appeals/{apB}/decide", tk, {"decision": "UPHELD", "response": "z" * 20, "version": 1}, idem()); chk(f"10 {n} decide → 403", s == 403, s)
s, _, _ = call("POST", f"{Q}/assignments/{A}/appeal", T, {"criteria_ids": ["c1"], "reason": "x" * 20}, idem()); chk("10 TEACHER gửi phúc khảo → 403", s == 403, s)
# --- Hôm nay: APPEAL, overdue sau 49 giờ
psql(f"update grade_appeals set created_at = now() - interval '49 hours' where id='{apB}'"); time.sleep(1)
td = call("GET", f"{Q}/today", T)[1]; it = [i for i in td.get("actions", []) + td.get("attention", []) if i.get("kind") == "APPEAL"]; print("   today APPEAL:", json.dumps(it, ensure_ascii=False)[:300])
chk("06 TEACHER: đúng một mục APPEAL, urgency overdue khi chờ > 48 giờ, chữ '… yêu cầu xem lại điểm …'", len(it) == 1 and it[0].get("urgency") == "overdue" and "yêu cầu xem lại điểm" in it[0].get("title", "") + it[0].get("reason", ""), it)
tdTA = call("GET", f"{Q}/today", TA)[1]; chk("06 TA không có APPEAL (chỉ TEACHER)", not [i for i in tdTA.get("actions", []) + tdTA.get("attention", []) if i.get("kind") == "APPEAL"])
n_ntf = psql("select count(*) from notifications where type='APPEAL_NEW'"); print("   chuông APPEAL_NEW:", n_ntf)
# --- quyết định
s, d, _ = call("GET", f"{Q}/appeals/{apB}", T); print("   chi tiết:", json.dumps(d, ensure_ascii=False)[:400]); vB = d.get("version", 1) if isinstance(d, dict) else 1
dec = lambda ap, body, v=None, tk=None: call("POST", f"{Q}/appeals/{ap}/decide", tk or T, {"version": v or vB, **body}, idem())
s, x, _ = dec(apB, {"decision": "UPHELD", "response": "z" * 9}); chk("03 phản hồi 9 ký tự → 422 RESPONSE_REQUIRED", s == 422 and "RESPONSE_REQUIRED" in json.dumps(x), f"{s} {str(x)[:100]}")
s, x, _ = dec(apB, {"decision": "ADJUSTED", "response": "Đã xem lại bài.", "criteria_scores": [{"id": "c2", "score": "1"}]}); chk("03 ADJUSTED không đổi tiêu chí nào → 422 NO_CHANGE", s == 422 and "NO_CHANGE" in json.dumps(x), f"{s} {str(x)[:100]}")
pub0 = psql(f"select published_at::text from grading_results where id='{rB}'"); v0 = ver(rB); ai0 = psql(f"select coalesce(ai_scores::text,'null') from grading_results where id='{rB}'")
res = []; body = {"decision": "ADJUSTED", "response": "Đã xem lại: tiêu chí 2 được cộng thêm.", "criteria_scores": [{"id": "c2", "score": "1.5"}]}
th = [threading.Thread(target=lambda: res.append(dec(apB, body, tk=T)[:2])) for _ in range(2)]; [t.start() for t in th]; [t.join() for t in th]
codes = sorted(r[0] for r in res); chk("11 hai lần quyết cùng lúc → một 200, một 409 APPEAL_ALREADY_DECIDED", codes == [200, 409] and any("APPEAL_ALREADY_DECIDED" in json.dumps(r[1]) for r in res if r[0] == 409), [r for r in res])
chk("03 ADJUSTED: kết quả sửa tại chỗ (tổng 4,00 → 4,50), vẫn PUBLISHED, version + 1, giữ published_at và ai_scores", psql(f"select status||'|'||total from grading_results where id='{rB}'") == "PUBLISHED|4.50" and ver(rB) == v0 + 1 and psql(f"select published_at::text from grading_results where id='{rB}'") == pub0 and psql(f"select coalesce(ai_scores::text,'null') from grading_results where id='{rB}'") == ai0, psql(f"select status||'|'||total from grading_results where id='{rB}'"))
chk("03 score_after ghi, status ADJUSTED", psql(f"select status||'|'||coalesce(score_after::text,'null') from grade_appeals where id='{apB}'") in ("ADJUSTED|4.50", "ADJUSTED|4.5"), psql(f"select status||'|'||coalesce(score_after::text,'null') from grade_appeals where id='{apB}'"))
s, vc, _ = call("GET", f"{Q}/appeals/{apC}", T); vC = vc["version"]
s, x, _ = dec(apC, {"decision": "UPHELD", "response": "Giữ nguyên điểm sau khi xem lại."}, vC); chk("03 UPHELD → 200, điểm giữ 8,00", s == 200 and psql(f"select total from grading_results where id='{rC}'") == "8.00", f"{s} {str(x)[:100]}")
s, x, _ = dec(apC, {"decision": "UPHELD", "response": "Giữ nguyên điểm sau khi xem lại."}, vC); chk("11 quyết lần hai → 409", s == 409, f"{s} {str(x)[:100]}")
time.sleep(8)
# --- B thấy phản hồi; thông báo; thư
s, m, _ = call("GET", f"{Q}/assignments/{A}/mine", SB); ap = (m.get("result") or {}).get("appeal"); print("   mine.result.appeal:", ap)
chk("06 B: mine.result thấy 4,50 và appeal {state, response}", s == 200 and ap and ap.get("response") and "4,50" in json.dumps(m["result"], ensure_ascii=False).replace(".", ","), f"{s} {str(ap)[:160]}")
s, mA, _ = call("GET", f"{Q}/assignments/{A}/mine", SA); chk("10 A không thấy phản hồi / lý do của B", "Đã xem lại: tiêu chí 2" not in json.dumps(mA, ensure_ascii=False))
chk("06 chuông APPEAL_REPLIED tới đúng B và C (mỗi người một)", psql("select string_agg(u.email, ',' order by u.email) from notifications n join users u on u.id=n.user_id where n.type='APPEAL_REPLIED'") == "sv.kha@edupilot.local,sv.nguyco@edupilot.local", psql("select string_agg(u.email, ',' order by u.email) from notifications n join users u on u.id=n.user_id where n.type='APPEAL_REPLIED'"))
msgs = [m for m in json.loads(subprocess.run(["curl", "-s", "http://localhost:8125/api/v1/messages?limit=50"], capture_output=True, text=True).stdout)["messages"] if "xem lại" in m.get("Subject", "") or "phúc khảo" in m.get("Subject", "").lower()]
to = sorted(t["Address"] for m in msgs for t in m["To"]); print("   thư phúc khảo:", [(m["Subject"], m["To"][0]["Address"]) for m in msgs])
chk("06 thư tới đúng 2 người (B, C), mỗi người một", to == ["sv.kha@edupilot.local", "sv.nguyco@edupilot.local"], to)
for m in msgs:
    if m["To"][0]["Address"] == "sv.kha@edupilot.local":
        mj = json.loads(subprocess.run(["curl", "-s", f"http://localhost:8125/api/v1/message/{m['ID']}"], capture_output=True, text=True).stdout); txt = mj.get("Text", "")
        print("   thư B:", txt[:240].replace("\n", " | "))
        chk("06 thư B: 'Đã sửa điểm', 4,00 → 4,50, phản hồi nguyên văn, không nhắc AI / nháp", "Đã sửa điểm" in txt and "4,00" in txt and "4,50" in txt and "tiêu chí 2 được cộng thêm" in txt and "AI" not in txt and "nháp" not in txt, txt[:200])
# --- stats
s, st, _ = call("GET", f"{Q}/assignments/{A}/stats", T); print("   stats:", json.dumps(st, ensure_ascii=False)[:260])
chk("08 stats: appealed 2, adjusted 1, appeal_rate 2/3, success 1/2 (chuỗi decimal)", s == 200 and st["appealed"] == 2 and st["adjusted"] == 1 and st["appeal_rate"].startswith("0.666") and st["appeal_success_rate"] in ("0.5000", "0.5"), st)
# --- cô lập với hàng chờ ticket
chk("09 không dòng escalation_tickets nào sinh ra suốt vòng đời", psql("select count(*) from escalation_tickets") == tk0, psql("select count(*) from escalation_tickets"))
chk("05 0 lời gọi LLM suốt vòng đời phúc khảo (llm_audit không tăng)", psql("select count(*) from llm_audit") == llm0, (psql("select count(*) from llm_audit"), llm0))
for p_ in (f"{Q}/appeals/{apB}/claim", f"{Q}/appeals/{apB}/answer"):
    s, _, _ = call("POST", p_, T, {}, idem()); chk(f"09 route cũ {p_.split('/')[-1]} với id phúc khảo → 404 / 405", s in (404, 405), s)
logs = subprocess.run("docker logs edupilot100-gateway-1 2>&1; docker logs edupilot100-worker-1 2>&1", shell=True, capture_output=True, text=True).stdout
chk("log không chứa lý do / phản hồi phúc khảo", "tiêu chí 2 được cộng thêm" not in logs and "x" * 20 not in logs)
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
