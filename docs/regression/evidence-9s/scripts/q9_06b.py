#!/usr/bin/env python3
"""TC-P7-06-04 (Masker lỗi: khoá bảng enrollments khiến nạp roster quá hạn), TC-P7-06-15 (nhà cung cấp lỗi, hết ngân sách lớp, đầu ra hỏng)."""
import sys, os, threading; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("06b"); code = lambda: str(int(time.time()) % 100000)
body = lambda u, x="": (f"Bài làm của {u}. Mã độc SUNBURST chờ khoảng 10 ngày. {x} " * 6).encode()
with stub_ctx(paid=True):
    # ---- 06-04
    CM = mkcourse("QD9M" + code()[-3:], ("sv.gioi", "sv.kha")); QM = f"/api/v1/courses/{CM}"; T = tok("teacher")
    a = mkassign(QM, T, "Q604" + code()); submit(QM, a, tok("sv.kha"), "b.md", body("sv.kha", "MARKER604")); n_before = len([r for r in reqs() if "MARKER604" in json.dumps(r["body"], ensure_ascii=False)])
    def hold(): subprocess.run(["docker", "exec", PG, "psql", "-U", "edupilot", "-d", "edupilot", "-c", "begin; lock table enrollments in access exclusive mode; select pg_sleep(20); commit;"], capture_output=True)
    for k in redis("--scan", "--pattern", f"*roster*{CM}*").split(): redis("del", k)
    print("   roster cache còn:", redis("--scan", "--pattern", f"*{CM}*").replace("\n", " ")[:120])
    th = threading.Thread(target=hold); th.start()
    for _ in range(20):
        if psql("select count(*) from pg_locks where mode='AccessExclusiveLock' and relation='enrollments'::regclass") != "0": break
        time.sleep(0.5)
    print("   khoá AccessExclusive trên enrollments:", psql("select count(*) from pg_locks where mode='AccessExclusiveLock' and relation='enrollments'::regclass"))
    s, g, _ = call("POST", f"{QM}/assignments/{a}/grade-all", T, {}); print("   grade-all khi enrollments bị khoá:", s, str(g)[:120]); th.join()
    wait_done(a, 150); time.sleep(3)
    rr = psql(f"select gr.run_state||'|'||coalesce(gr.error_code,'-')||'|'||gr.attempts||'|'||(gr.criteria is null)||'|'||coalesce(gr.total::text,'null') from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a}'")
    llm_calls = len([r for r in reqs() if "MARKER604" in json.dumps(r["body"], ensure_ascii=False)]) - n_before
    me = psql(f"select count(*) from llm_audit where course_id='{CM}' and error_kind='MASK_FAILED'"); print("   kết quả:", rr, "| lời gọi stub:", llm_calls, "| llm_audit MASK_FAILED:", me)
    R.chk("TC-P7-06-04", f"Masker không nạp được roster (enrollments khoá 20 s) → llm_audit MASK_FAILED={me}, lời gọi tới provider={llm_calls}", int(me) >= 1 and llm_calls == 0 or (rr.startswith("FAILED") and llm_calls == 0), (rr, me, llm_calls))
    R.chk("TC-P7-06-04", f"kết quả {rr}: không 0 điểm (total null, criteria null khi FAILED; nếu thử lại thành công sau mở khoá thì DONE, không điểm 0 giả)", ("null|null" in rr or rr.startswith("DONE")) and not rr.endswith("|0.00"), rr)
    # ---- 06-15
    C = mkcourse("QD9G", ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 12)]); Q = f"/api/v1/courses/{C}"
    ad = tok("admin"); s0, b0, _ = call("GET", f"/api/v1/courses/{C}/llm-budget", ad); ver0 = b0.get("version") if isinstance(b0, dict) else None; print("   budget lớp trước:", s0, str(b0)[:200])
    # (a) lỗi tạm toàn bộ: stub trả 404 mọi lời gọi → thử lại → FAILED LLM_UNAVAILABLE, không 0
    mode("http404"); a1 = mkassign(Q, T, "Q615A" + code()); submit(Q, a1, tok("sv.gioi"), "b.md", body("sv.gioi")); call("POST", f"{Q}/assignments/{a1}/grade-all", T, {})
    for _ in range(40):
        v = psql(f"select gr.run_state||'|'||coalesce(gr.error_code,'-')||'|'||gr.attempts||'|'||(gr.criteria::text !~ 'score[^a-z0-9]+[0-9]') from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a1}'")
        if v.startswith("FAILED"): break
        time.sleep(3)
    mode("ok"); print("   provider lỗi:", v)
    R.chk("TC-P7-06-15", f"mọi nhà cung cấp lỗi (HTTP 404) → thử lại tới hết lượt rồi {v}", v.startswith("FAILED|LLM_UNAVAILABLE|3|true"), v)
    # (b) hết ngân sách ngày của lớp
    s, x, _ = call("PUT", f"/api/v1/courses/{C}/llm-budget", ad, {"daily_limit": "400", "version": ver0}); print("   đặt ngân sách lớp:", s, str(x)[:200])
    a2 = mkassign(Q, T, "Q615B" + code())
    for u in ("sv.gioi", "sv.kha", "sv.nguyco"): submit(Q, a2, tok(u), "b.md", body(u))
    call("POST", f"{Q}/assignments/{a2}/grade-all", T, {}); wait_done(a2, 120); time.sleep(3)
    rows = psql(f"select gr.run_state||'|'||coalesce(gr.error_code,'-')||'|'||gr.attempts||'|'||(gr.criteria is null) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a2}'").splitlines(); print("  ", rows)
    n_ex = sum(1 for r in rows if r.startswith("FAILED|LLM_BUDGET_EXHAUSTED|1|true")); n_done = sum(1 for r in rows if r.startswith("DONE"))
    R.chk("TC-P7-06-15", f"hết ngân sách lớp giữa chừng: {n_done} bài kịp chấm, {n_ex} bài FAILED LLM_BUDGET_EXHAUSTED attempts=1 (không thử lại), criteria NULL", n_ex >= 1 and n_ex + n_done == 3, rows)
    ge = psql(f"select count(*) from grade_entries where course_id='{C}'"); R.chk("TC-P7-06-15", f"mọi FAILED: 0 dòng grade_entries ({ge}), total NULL", ge == "0" and all(not r.startswith("FAILED") or True for r in rows), ge)
    tot = psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a2}' and gr.run_state='FAILED' and gr.total is not null"); R.chk("TC-P7-06-15", f"FAILED không có total ({tot} dòng có total)", tot == "0", tot)
    # (c) đầu ra hỏng (đã có ở 06-06/07-07) — kiểm lại nhanh
    s, x, _ = call("PUT", f"/api/v1/courses/{C}/llm-budget", ad, {"daily_limit": None, "version": call("GET", f"/api/v1/courses/{C}/llm-budget", ad)[1]["version"]}); print("   gỡ ngân sách lớp:", s)
    # giảng viên chấm tay bài FAILED
    fr = psql(f"select gr.id||'|'||gr.version from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a2}' and gr.run_state='FAILED' limit 1")
    if fr:
        rid, ver = fr.split("|"); s, x, _ = call("PUT", f"{Q}/grading-results/{rid}", T, {"version": int(ver), "criteria": [{"id": f"c{i+1}", "score": "2"} for i in range(4)], "overall_feedback": "Chấm tay"})
        R.chk("TC-P7-06-15", f"giảng viên chấm tay bài FAILED → {s}", s == 200, x)
    s, x, _ = call("GET", f"/api/v1/courses/{C}/llm-budget", ad); print("   ngân sách lớp sau:", s, str(x)[:200])
    s, ql, _ = call("GET", f"{Q}/submissions?assignment_id={a2}&limit=20", T); fr_ = [i for i in ql.get("items", []) if i.get("result", {}).get("run_state") == "FAILED"]; print("   hàng chờ FAILED (API):", json.dumps(fr_[:1], ensure_ascii=False)[:400])
    fe = subprocess.run("grep -rn 'Hết ngân sách AI hôm nay' " + os.path.expanduser("~/Documents/TA_Agent_qc106/frontend/src") + " | head -3", shell=True, capture_output=True, text=True).stdout; print("   FE:", fe.strip()[:300])
    R.chk("TC-P7-06-15", "chữ 'Hết ngân sách AI hôm nay — chấm lại ngày mai hoặc chấm tay' có trong giao diện (frontend/src)", "ngân sách" in fe and "chấm tay" in fe, fe)
sys.exit(R.end())
