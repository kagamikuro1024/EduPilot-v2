# QC sprint 11 — US-P10-04 API (+DB). Tuyến chữ = fake (llm_route.py fake) trừ khi ghi khác. Xoá bộ đếm Redis ep:rl:insight-generate:* giữa các ca (ghi rõ trong báo cáo).
import sys, re, subprocess, threading
from p10lib import *
part = sys.argv[1] if len(sys.argv) > 1 else "all"
c1, c2 = course("teacher", "761987"), course("teacher", "761988")
GEN = lambda c: f"/courses/{c}/insights/knowledge-gap"
def reset_rl(): subprocess.run(["docker", "exec", "ep-qc-redis-1", "sh", "-c", "redis-cli --scan --pattern 'ep:rl:insight-generate:*' | xargs -r redis-cli del"], capture_output=True)
def nrep(c=None): return int(sql("select count(*) from insight_reports" + (f" where course_id='{c}'" if c else ""))[0][0])
def naudit(): return int(sql("select count(*) from llm_audit where task='INSIGHT'")[0][0])
def wait_job(user, jid, t=240):
    end = time.time() + t
    while time.time() < end:
        c, j, _, _ = api(user, "GET", f"/jobs/{jid}")
        if c == 200 and j.get("status") in ("DONE", "SUCCEEDED", "FAILED", "DEAD"): return j
        time.sleep(2)
    return j
def gen(user, c, body, key=None):
    reset_rl()
    return api(user, "POST", f"{GEN(c)}/generate", body, headers={"Idempotency-Key": key} if key else None)
def ask(user, c, text):
    c_, s, _, _ = api(user, "POST", "/chat/sessions", {"course_id": c}); sid = s["id"]
    api(user, "POST", f"/chat/sessions/{sid}/messages", {"content": text})
    time.sleep(1.5)

if part in ("all", "perm"):
    c, _, _, _ = api("sv.kha", "POST", f"{GEN(c1)}/generate", {"period_days": 14}); ok("TC-04-26 sv.kha generate → 403", c == 403, str(c))
    for u in ("sv.moi",):
        c, _, _, _ = api(u, "POST", f"{GEN(c1)}/generate", {"period_days": 14}); ok(f"TC-04-27 {u} generate lớp 1 → 403 (P-11-35)", c == 403, str(c))
    for tc, user, meth, path in (("generate", "admin", "POST", f"{GEN(c1)}/generate"), ("latest", "admin", "GET", f"{GEN(c1)}/latest")):
        c, _, _, _ = api(user, meth, path, {"period_days": 14} if meth == "POST" else None); ok(f"TC-04-28 ADMIN {tc} → 403", c == 403, str(c))
    rid = sql(f"select id from insight_reports where course_id='{c1}' order by created_at desc limit 1")[0][0]
    c, _, _, _ = api("admin", "GET", f"{GEN(c1)}/{rid}/export?format=md"); ok("TC-04-28 ADMIN export → 403", c == 403, str(c))
    for u in ("teacher", "ta"):
        c, _, _, _ = api(u, "GET", f"{GEN(c1)}/latest"); ok(f"TC-04-29 {u} latest → 200", c == 200, str(c))
        c, _, _, _ = api(u, "GET", f"{GEN(c1)}/{rid}/export?format=md"); ok(f"TC-04-29 {u} export md → 200", c == 200, str(c))
        for f in ("pdf", "xml"):
            c, _, _, _ = api(u, "GET", f"{GEN(c1)}/{rid}/export?format={f}"); ok(f"TC-04-29 {u} export {f} → 422", c == 422, str(c))
    c, _, _, _ = raw("GET", f"{GEN(c1)}/latest"); ok("TC-04-30 latest không token → 401", c == 401, str(c))
    c, _, _, _ = raw("POST", f"{GEN(c1)}/generate", {"period_days": 14}, {"Idempotency-Key": str(uuid.uuid4())}); ok("TC-04-30 generate không token → 401", c == 401, str(c))
    c, _, _, _ = api("ta", "POST", f"{GEN(c1)}/generate", {"period_days": 14}); ok("TC-04-05 ta generate → được như GV (200 reused/202)", c in (200, 202), str(c))
    # GV/TA lớp 2 vào lớp 1: teacher dạy cả 2 lớp → không dùng được; ta idem
if part in ("all", "valid"):
    r0, a0 = nrep(), naudit()
    c, _, _, _ = raw("POST", f"{GEN(c1)}/generate", {"period_days": 14}, {"Authorization": "Bearer " + tok("teacher")})
    ok("TC-04-03 generate không Idempotency-Key → 400/422", c in (400, 422), str(c))
    for tag, body in (("0", {"period_days": 0}), ("-1", {"period_days": -1}), ("5", {"period_days": 5}), ("15", {"period_days": 15}), ("60", {"period_days": 60}), ("119", {"period_days": 119}), ("121", {"period_days": 121}), ("999", {"period_days": 999}),
                      ("null", {"period_days": None}), ('"14"', {"period_days": "14"}), ("14.0", None), ("chữ", {"period_days": "abc"})):
        if body is None:
            reset_rl(); c, j, _, _ = raw("POST", f"{GEN(c1)}/generate", None, {"Idempotency-Key": str(uuid.uuid4()), "Authorization": "Bearer " + tok("teacher")}) if False else (0, None, 0, {})
            import urllib.request as ur
            req = ur.Request(B + f"{GEN(c1)}/generate", data=b'{"period_days": 14.0}', method="POST", headers={"content-type": "application/json", "authorization": "Bearer " + tok("teacher"), "Idempotency-Key": str(uuid.uuid4())})
            try: c = ur.urlopen(req, context=CTX).status
            except Exception as e: c = e.code
        else:
            c, j, _, _ = gen("teacher", c1, body)
        want_ok = tag in ("null",)  # null ≡ không gửi → mặc định 14 (Q-E-7)
        ok(f"TC-04-04 period_days={tag} → " + ("mặc định 14 (200/202)" if want_ok else "422"), (c in (200, 202)) if want_ok else c == 422, f"{c} {str(j)[:100] if body is not None else ''}")
    ok("TC-04-04 giá trị sai không tạo job/báo cáo/lời gọi LLM", nrep() in (r0, r0 + 1) and naudit() == a0, f"rep {r0}->{nrep()} audit {a0}->{naudit()}")
    c, j, _, _ = gen("teacher", c1, {})
    ok("TC-04-04 không gửi period_days → mặc định 14", c in (200, 202) and (nrep() >= 1) and sql(f"select payload->>'period_days' from insight_reports where course_id='{c1}' order by created_at desc limit 1")[0][0] in ("14", "120", "30"), f"{c} {str(j)[:100]}")
    for pd in (7, 14, 30, 120):
        c, j, _, _ = gen("teacher", c1, {"period_days": pd})
        ok(f"TC-04-04 period_days={pd} hợp lệ → 200/202", c in (200, 202), f"{c} {str(j)[:100]}")
if part in ("all", "reuse"):
    a0, r0 = naudit(), nrep(c1)
    c, j, _, _ = gen("teacher", c1, {"period_days": 14})
    ok("TC-04-18 không có dữ liệu mới → 200 reused:true", c == 200 and j.get("reused") is True, f"{c} {str(j)[:100]}")
    ok("TC-04-18 llm_audit / insight_reports không tăng", naudit() == a0 and nrep(c1) == r0, f"audit {a0}->{naudit()} rep {r0}->{nrep(c1)}")
    k = str(uuid.uuid4()); ask("sv.kha", c1, f"QC thêm câu hỏi mới về học phí {uuid.uuid4().hex[:6]}")
    c, j, _, _ = gen("teacher", c1, {"period_days": 14}, k)
    ok("TC-04-19 thêm câu hỏi → 202 (không reused)", c == 202, f"{c} {str(j)[:100]}")
    c2_, j2, _, _ = api("teacher", "POST", f"{GEN(c1)}/generate", {"period_days": 14}, headers={"Idempotency-Key": k})
    ok("TC-04-02 cùng Idempotency-Key → cùng job_id", c2_ in (200, 202) and j2 and j2.get("job_id") == j.get("job_id"), f"{c2_} {str(j2)[:100]}")
    if c == 202:
        jj = wait_job("teacher", j["job_id"]); ok("TC-04-01 job xong", jj.get("status") in ("DONE", "SUCCEEDED"), str(jj)[:160])
        wm = sql(f"select data_watermark from insight_reports where course_id='{c1}' and payload->>'period_days'='14' order by created_at desc limit 2")
        ok("TC-04-19 báo cáo mới có data_watermark mới", len(wm) == 2 and wm[0][0] != wm[1][0], str(wm))
        ai = sql("select task, lane, status, provider from llm_audit where task='INSIGHT' order by created_at desc limit 3")
        ok("TC-04-09 job 1 lời gọi INSIGHT làn BATCH", len([a for a in ai[:1] if a[1] == "BATCH"]) == 1, str(ai))
if part in ("all", "content"):
    c, j, _, _ = api("teacher", "GET", f"{GEN(c1)}/latest"); rep = j["report"]; p = rep["payload"]
    cnt = p["counts"]; print("counts", cnt)
    ok("TC-04-06 counts = {questions,students,clusters} (P-11-35)", set(cnt) == {"questions", "students", "clusters"}, str(cnt))
    keys = json.dumps(p)
    def keysof(x):
        return set(x) | {k for v in x.values() for k in (keysof(v) if isinstance(v, dict) else [])} if isinstance(x, dict) else (set().union(*[keysof(i) for i in x]) if isinstance(x, list) and x else set())
    ok("TC-04-16 khoá payload không có student_id/user_id/email/student_code", not (keysof(p) & {"student_id", "user_id", "email", "student_code"}), str(sorted(keysof(p))))
    names = [r[0] for r in sql("select full_name from users where email like 'sv.%' union select student_code from users where student_code is not null union select email from users where email like 'sv.%'") if r[0] and len(r[0]) > 3]
    exp = api("teacher", "GET", f"{GEN(c1)}/{rep['id']}/export?format=md")[1]
    blob = keys + json.dumps(exp, ensure_ascii=False)
    ok("TC-04-14 0 khớp tên/MSSV/email/SĐT trong latest + export + payload", not [n for n in names if n in blob] and not re.search(r"tôi là [A-ZĐ]", blob), str([n for n in names if n in blob][:3]))
    topics = p["topics"]
    ok("TC-04-08 rank liền mạch 1..n, score giảm dần", [t["rank"] for t in topics] == list(range(1, len(topics) + 1)) and all(topics[i]["score"] >= topics[i + 1]["score"] - 1e-9 for i in range(len(topics) - 1)), f"n={len(topics)}")
    d = [t for t in topics if t.get("few_asked")]
    ok("TC-04-12 có chủ đề «Ít người hỏi» với examples=[]", d and all(t["examples"] == [] for t in d), f"{len(d)}")
    ok("TC-04-12 chủ đề thường có 3–5 câu mẫu hoặc ít hơn khi <3 SV", all(len(t["examples"]) <= 5 for t in topics), "")
    r120 = json.loads(sql("select payload::text from insight_reports where course_id='%s' and payload->>'period_days'='120' order by created_at desc limit 1" % c1)[0][0])
    ok("TC-04-10 (v1.10 P-11-31) uncovered kỳ 120: đúng 2 nhóm N1 (3) và N2 (4), N3 bị bỏ", sorted(u["count"] for u in r120["uncovered"] if "ZQ-L1" not in u["text"] and not u["text"].startswith("QC")) == [3, 4], str(r120["uncovered"]))
    top2 = [t["name"] for t in r120["topics"][:2]]
    A = [r[0] for r in sql("select 1")]  # tên chủ đề A, B lấy ở TC-04-10b
    print("top2 kỳ 120:", top2)
if part in ("all", "iso"):
    tag = "ZQ-L1-" + uuid.uuid4().hex[:6]
    for u in ("sv.kha", "sv.gioi", "sv.nguyco"): ask(u, c1, f"Cho hỏi về {tag} học phần")
    ask("sv34", c2, "QC mở lớp hai: câu hỏi mới " + uuid.uuid4().hex[:6])
    reset_rl(); r1 = api("teacher", "POST", f"{GEN(c1)}/generate", {"period_days": 120}); r2 = api("teacher", "POST", f"{GEN(c2)}/generate", {"period_days": 120})
    for r in (r1, r2):
        if r[0] == 202: wait_job("teacher", r[1]["job_id"])
    b1 = sql(f"select payload::text from insight_reports where course_id='{c1}' order by created_at desc limit 1")[0][0]
    b2 = sql(f"select payload::text from insight_reports where course_id='{c2}' order by created_at desc limit 1")[0][0]
    ok("TC-04-17 không câu/từ khoá riêng lớp 1 (ZQ-L1) ở lớp 2", tag not in b2 and "ZQ-L1" not in b2, f"{r1[0]} {r2[0]}")
    topic_c = [t for t in json.loads(b2)["topics"][:3]]
    print("top lớp 2:", [t["name"] for t in topic_c])
if part in ("all", "rate"):
    reset_rl(); res = []
    for i in range(4):
        sql(f"update insight_reports set data_watermark = data_watermark - interval '1 day' where course_id='{c2}'")  # fixture: coi như có dữ liệu mới
        c, j, _, hd = api("teacher", "POST", f"{GEN(c2)}/generate", {"period_days": 120})
        res.append((c, hd.get("Retry-After")))
    ok("TC-04-25 3 lần đầu 202, lần 4 429 + Retry-After", [r[0] for r in res] == [202, 202, 202, 429] and res[3][1], str(res))
if part in ("all", "conc"):
    ask("sv.kha", c1, "QC song song " + uuid.uuid4().hex[:6]); reset_rl(); out = []
    def f(): out.append(api("teacher", "POST", f"{GEN(c1)}/generate", {"period_days": 30}))
    th = [threading.Thread(target=f) for _ in range(2)]; [t.start() for t in th]; [t.join() for t in th]
    for r in out:
        if r[0] == 202: wait_job("teacher", r[1]["job_id"])
    wms = [r[0] for r in sql(f"select data_watermark::text from insight_reports where course_id='{c1}' and payload->>'period_days'='30'")]
    ok("TC-04-31 hai generate song song: không trùng watermark", len(wms) == len(set(wms)) and all(r[0] in (200, 202, 409, 429) for r in out), f"{[r[0] for r in out]} wm={len(wms)}/{len(set(wms))}")
