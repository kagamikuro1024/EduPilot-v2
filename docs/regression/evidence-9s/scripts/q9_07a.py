#!/usr/bin/env python3
"""TC-P7-07-01,02,06,07,10,13 (hàng đợi chấm, không cần dừng dịch vụ). Provider stub Q9 dưới khoá global; lớp QD9G (3 sinh viên + sv04..sv27)."""
import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("07a"); KEY = str(int(time.time()))
SV = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 28)]
C = mkcourse("QD9G", SV); Q = f"/api/v1/courses/{C}"; T = tok("teacher"); TA = tok("ta")
body = lambda u, extra="": (f"Bài làm của {u}. Mã độc SUNBURST chờ khoảng 10 ngày rồi mới gửi tín hiệu. {extra} " * 6).encode()
def fresh(code, n=3, extra="", users=None):
    aid = mkassign(Q, T, code + str(int(time.time()) % 100000))
    for u in (users or SV[:n]): s, x, _ = submit(Q, aid, tok(u), "b.md", body(u, extra)); assert s in (200, 201), (u, s, x)
    return aid
def rows(aid): return psql(f"select gr.run_state||':'||coalesce(gr.error_code,'-')||':'||gr.attempts from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{aid}' order by 1").splitlines()
def calls(aid): return int(psql(f"select count(*) from llm_audit where task='GRADING' and created_at > now() - interval '30 minutes'")) # thô, chỉ để tham khảo
with stub_ctx():
    # --- 01: 24 bài, 202 + queued, 24 dòng QUEUED/RUNNING/DONE round=1 ngay sau 202, hàng Redis
    aid = fresh("Q701", 24)
    n0 = len(reqs()); t0 = time.time()
    s, g, _ = call("POST", f"{Q}/assignments/{aid}/grade-all", T, {}); jid = g.get("job_id")
    cnt = int(psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{aid}' and gr.round=1"))
    R.chk("TC-P7-07-01", "202 {job_id, queued:24}", s == 202 and g.get("queued") == 24 and jid, (s, g))
    R.chk("TC-P7-07-01", f"ngay sau 202: {cnt} dòng grading_results round=1 (đủ 24 trước khi chấm xong)", cnt == 24, cnt)
    seen = set()
    for _ in range(80):
        s2, j, _ = call("GET", f"/api/v1/jobs/{jid}", T); 
        if isinstance(j, dict): seen.add(json.dumps({k: j.get(k) for k in ("status", "progress", "result")}, ensure_ascii=False))
        if isinstance(j, dict) and j.get("status") in ("SUCCEEDED", "DONE", "COMPLETED", "FAILED"): break
        time.sleep(1.5)
    R.note("TC-P7-07-01", "job: " + " | ".join(sorted(seen))[:900]); print("   job cuối:", json.dumps(j, ensure_ascii=False)[:400])
    R.chk("TC-P7-07-01", "xong: result {graded, failed}", isinstance(j, dict) and isinstance(j.get("result"), dict) and "graded" in j["result"] and "failed" in j["result"], j)
    wait_done(aid, 120); st = rows(aid)
    R.chk("TC-P7-07-01", f"24 DONE: {set(st)}", len(st) == 24 and all(x.startswith("DONE") for x in st), st)
    # --- 02: grade-all lần hai: queued 0 + chữ; bài mới chưa có kết quả
    s, g, _ = call("POST", f"{Q}/assignments/{aid}/grade-all", T, {})
    R.chk("TC-P7-07-02", f"không còn bài chưa chấm → {s} queued {g.get('queued') if isinstance(g, dict) else g}", s in (200, 202) and isinstance(g, dict) and g.get("queued") == 0, (s, g))
    print("   thân:", json.dumps(g, ensure_ascii=False)[:300])
    s, x, _ = submit(Q, aid, tok("sv.gioi"), "b.md", body("sv.gioi", "bản hai khác")); 
    s, g, _ = call("POST", f"{Q}/assignments/{aid}/grade-all", T, {})
    R.chk("TC-P7-07-02", f"bài nộp mới (v2) chưa có kết quả → chỉ 1 bài xếp: queued {g.get('queued') if isinstance(g, dict) else g}", s == 202 and g.get("queued") == 1, (s, g))
    wait_done(aid, 60)
    # --- 07: lỗi vĩnh viễn: BUDGET (HTTP... không giả lập qua stub) → dùng GRADING_OUTPUT_INVALID (invalid) và TEXT_TOO_LONG (bài 60.001 ký tự)
    a7 = mkassign(Q, T, "Q707" + str(int(time.time()) % 100000))
    submit(Q, a7, tok("sv.gioi"), "b.md", body("sv.gioi", "[[S9G invalid]]"))
    long = (("abcdefgh " * 7000)[:60001]).encode()
    submit(Q, a7, tok("sv.kha"), "long.md", long)
    call("POST", f"{Q}/assignments/{a7}/grade-all", T, {}); wait_done(a7, 150); time.sleep(8)
    r7 = psql(f"select u.email||'|'||gr.run_state||'|'||coalesce(gr.error_code,'-')||'|'||gr.attempts from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{a7}' order by 1").splitlines(); print("  ", r7)
    R.chk("TC-P7-07-07", "invalid → FAILED GRADING_OUTPUT_INVALID, attempts=1", any("sv.gioi" in x and "FAILED|GRADING_OUTPUT_INVALID|1" in x for x in r7), r7)
    R.chk("TC-P7-07-07", "bài 61.000 ký tự → FAILED TEXT_TOO_LONG, attempts=1", any("sv.kha" in x and "FAILED|TEXT_TOO_LONG|1" in x for x in r7), r7)
    # --- 06: lỗi tạm 404 (non-outage) 3 lần rồi OK; lỗi liên tục → FAILED; một bài lỗi không chặn cả lô
    a6 = mkassign(Q, T, "Q706" + str(int(time.time()) % 100000))
    submit(Q, a6, tok("sv.gioi"), "b.md", body("sv.gioi", "[[S9G fail=1:404:k706a%s]]" % KEY + ""))   # lỗi tạm 1 lần (lượt chấm gọi 2 pass song song → đếm theo key)
    submit(Q, a6, tok("sv.kha"), "b.md", body("sv.kha", "[[S9G fail=99:404:k706b%s]]" % KEY + ""))    # lỗi liên tục
    submit(Q, a6, tok("sv.nguyco"), "b.md", body("sv.nguyco"))                              # lành
    g0 = psql("select count(*) from llm_audit"); t0 = time.time()
    call("POST", f"{Q}/assignments/{a6}/grade-all", T, {})
    samples = []
    for _ in range(70):
        st = psql(f"select u.email||'|'||gr.run_state||'|'||gr.attempts||'|'||coalesce(to_char(gr.not_before - now(),'SS.MS'),'-') from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{a6}' order by 1")
        samples.append((round(time.time() - t0, 1), st.replace("\n", " ; ")))
        if "FAILED" in st and "DONE" in st and "QUEUED" not in st and "RUNNING" not in st: break
        time.sleep(1.5)
    for t, x in samples[::4] + samples[-1:]: print("   t=%s %s" % (t, x))
    fin = psql(f"select u.email||'|'||gr.run_state||'|'||gr.attempts||'|'||coalesce(gr.error_code,'-') from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{a6}' order by 1").splitlines(); print("  ", fin)
    R.chk("TC-P7-07-06", "bài lành DONE; bài lỗi tạm 1 lần rồi OK (attempts>=2, DONE); bài lỗi liên tục FAILED attempts=3", any("sv.nguyco" in x and "|DONE|" in x for x in fin) and any("sv.gioi" in x and "|DONE|" in x for x in fin) and any("sv.kha" in x and "|FAILED|3|" in x for x in fin), fin)
    R.chk("TC-P7-07-06", "hết lượt: FAILED có mã nguyên nhân (không INTERNAL rỗng)", any("sv.kha" in x and "FAILED" in x and not x.endswith("|-") for x in fin), fin)
    rid = psql(f"select gr.id from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{a6}' and u.email='sv.kha@edupilot.local'")
    s, x, _ = call("POST", f"{Q}/grading-results/{rid}/retry", T, {}); st = psql(f"select run_state from grading_results where id='{rid}'")
    R.chk("TC-P7-07-06", f"retry → {s}, run_state sau retry {st} (QUEUED/RUNNING/FAILED lại)", s in (200, 202) and st in ("QUEUED", "RUNNING", "FAILED"), (s, x))
    dead = redis("XLEN", "ep:grading.dead"); R.chk("TC-P7-07-06", f"ep:grading.dead XLEN={dead} (chỉ đếm)", dead.isdigit() and int(dead) >= 1, dead)
    s, it, _ = today("teacher", C); kinds = [i.get("kind") for i in it]
    R.chk("TC-P7-07-06", f"Hôm nay TEACHER có GRADING_ERROR: {kinds[:8]}", "GRADING_ERROR" in kinds, kinds)
    # --- 10: B nộp lại khi bản 1 còn QUEUED (6 bài chậm chiếm hết worker, B xếp sau cùng)
    a10 = mkassign(Q, T, "Q710" + str(int(time.time()) % 100000))
    for u in ["sv04", "sv05", "sv06", "sv07", "sv08", "sv09"]: submit(Q, a10, tok(u), "b.md", body(u, "[[S9G slow=12]]"))
    submit(Q, a10, tok("sv.kha"), "b.md", body("sv.kha", "ONLYV1TOKEN"))
    call("POST", f"{Q}/assignments/{a10}/grade-all", T, {}); time.sleep(2)
    s1 = psql(f"select gr.run_state from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{a10}' and u.email='sv.kha@edupilot.local' and s.version=1")
    submit(Q, a10, tok("sv.kha"), "b.md", body("sv.kha", "bản hai ONLYV2TOKEN"))
    s, g, _ = call("POST", f"{Q}/assignments/{a10}/grade-all", T, {}); print("   grade-all sau nộp lại:", s, g)
    wait_done(a10, 150); time.sleep(4)
    rr = psql(f"select s.version||'|'||gr.run_state||'|'||coalesce(gr.error_code,'-') from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{a10}' and u.email='sv.kha@edupilot.local' order by s.version").splitlines()
    v1calls = len([r for r in reqs() if "ONLYV1TOKEN" in json.dumps(r["body"], ensure_ascii=False)]); v2calls = len([r for r in reqs() if "ONLYV2TOKEN" in json.dumps(r["body"], ensure_ascii=False)])
    print("   v1 lúc nộp lại:", s1, "| kết quả:", rr, "| lời gọi LLM v1/v2:", v1calls, v2calls)
    R.chk("TC-P7-07-10", f"v1 đang {s1} khi nộp lại → kết quả v1 {rr[0]} (SUPERSEDED), 0 lời gọi LLM cho v1 ({v1calls})", s1 == "QUEUED" and rr[0].startswith("1|FAILED|SUPERSEDED") and v1calls == 0, (s1, rr, v1calls))
    R.chk("TC-P7-07-10", f"chỉ v2 được chấm (DONE) với {v2calls} lời gọi", rr[1].startswith("2|DONE") and v2calls == 2, (rr, v2calls))
    s, qd, _ = call("GET", f"{Q}/submissions?assignment_id={a10}&limit=100", T); items = qd.get("items", []) if isinstance(qd, dict) else []
    kha = [i for i in items if "kha" in json.dumps(i, ensure_ascii=False)]; print("   hàng chờ dòng của B:", json.dumps(kha, ensure_ascii=False)[:300])
    R.chk("TC-P7-07-10", f"hàng chờ chỉ có v2 của B ({[k.get('version') for k in kha]})", [k.get("version") for k in kha] == [2], kha)
    # --- 13: bộ đếm + nhật ký
    for sv in ("ep-qc-gateway-1", "ep-qc-worker-1"):
        lg = subprocess.run(["docker", "logs", sv, "--since", "20m"], capture_output=True, text=True); txt = lg.stdout + lg.stderr
        bad = [k for k in ("SUNBURST", "Bài làm của sv", "20229001", "Trần Văn") if k in txt]
        trace = len(re.findall(r'"trace_id"', txt))
        R.chk("TC-P7-07-13", f"{sv}: log 20 phút: trace_id {trace} dòng, canary/văn bản bài/MSSV {bad}", not bad and trace > 0, bad)
sys.exit(R.end())
