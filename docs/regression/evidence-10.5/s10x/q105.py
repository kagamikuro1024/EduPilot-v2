# Trợ giúp QC smoke 10.5 trên ep-qc (HTTPS :1143). Dùng qlib của evidence-9s/s10 (đã trỏ ep-qc), ghi kết quả vào results-s10x.jsonl.
import os, sys, json, time, subprocess
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import qlib
from qlib import raw, api, sse, sql, pct, CTX, tok
qlib.RES = os.path.join(HERE, "results-s10x.jsonl")
RES = qlib.RES
_B = qlib.B

def rec(tc, kq, note):
    """kq ∈ P/F/N; ghi dòng cuối của TC."""
    print(f"{kq} {tc} {note}"[:300])
    with open(RES, "a", encoding="utf8") as f: f.write(json.dumps({"tc": tc, "kq": kq, "note": note}, ensure_ascii=False) + "\n")

def adm(method, path, body=None, **kw): return api("admin", method, path, body, **kw)
def fake(**kw):
    """POST /_test/llm/fake (ADMIN, route thử): đổi bản giả lúc chạy, không restart."""
    c, j, _ = api("admin", "POST", "/_test/llm/fake", kw, idem=False); assert c == 200, (c, j); return j
def fake_reset():
    return fake(decision_mode="", decision_p50_ms=426, decision_p95_ms=682, decision_error_rate=0, decision_prob=0.25)
def stats():
    c, j, _ = api("admin", "GET", "/_test/llm/stats"); return j
def payloads():
    c, j, _ = api("admin", "GET", "/_test/llm/payloads"); return j["items"]
def payload_reset():
    return api("admin", "POST", "/_test/llm/payloads/reset", {})
def now_db(): return sql("select clock_timestamp()")[0][0]
def gates():
    c, j, _ = adm("GET", "/admin/llm/decision"); return j
def sys_gate(gate, enabled=None, **kw):
    g = [x for x in gates()["gates"] if x["gate"] == gate][0]
    body = {"version": g["version"], **kw}
    if enabled is not None: body["enabled"] = enabled
    c, j, _ = adm("PUT", f"/admin/llm/decision/gates/{gate}", body); assert c == 200, (c, j); return j
def course_gate(cid, gate, enabled):
    c, j, _ = adm("GET", f"/courses/{cid}/decision-gates")
    g = [x for x in j["items"] if x["gate"] == gate][0]
    if g["enabled"] == enabled: return g
    c, j, _ = adm("PUT", f"/courses/{cid}/decision-gates/{gate}", {"enabled": enabled, "version": g["version"]}); assert c == 200, (c, j); return j
def set_gates(cid, on=(), system_on=None, wait=1.6):
    """Hệ thống: bật đúng các cổng trong system_on (mặc định = on); lớp cid: bật đúng các cổng trong on."""
    system_on = on if system_on is None else system_on
    for g in ("THREADS_PRECHECK", "INJECTION_FLAG", "CHAT_ROUTING"):
        cur = [x for x in gates()["gates"] if x["gate"] == g][0]["enabled"]
        if cur != (g in system_on): sys_gate(g, g in system_on)
        if cid: course_gate(cid, g, g in on)
    time.sleep(wait)
def all_off(cid=None):
    set_gates(cid, (), ())
def decide_rows(since, cid=None, extra="", wait=4.0):
    time.sleep(wait)  # llm_audit ghi bất đồng bộ (bộ đệm)
    w = f" and course_id='{cid}'" if cid else ""
    return sql(f"select cascade_gate,coalesce(cascade_path,''),status,coalesce(error_kind,''),lane,latency_ms,provider,model from llm_audit where task='DECIDE' and created_at>='{since}'{w} {extra} order by created_at")
def audit_rows(since, cid=None, task=None, wait=4.0):
    time.sleep(wait)
    w = f" and course_id='{cid}'" if cid else ""; t = f" and task='{task}'" if task else ""
    return sql(f"select task,status,coalesce(cascade_gate,''),coalesce(cascade_path,''),lane,latency_ms from llm_audit where created_at>='{since}'{w}{t} order by created_at")
def cid_of(code): return sql(f"select id from courses where class_code='{code}'")[0][0]
def uid_of(email): return sql(f"select id from users where email='{email}'")[0][0]
def post_thread(user, cid, title, body, **kw):
    t0 = time.time(); c, j, ms = api(user, "POST", f"/courses/{cid}/threads", {"title": title, "body": body}, timeout=60, **kw); return c, j, ms
def err_code(j):
    if isinstance(j, dict): return j.get("code") or (j.get("error") or {}).get("code")
def chat(user, cid, text):
    c, ss, _ = api(user, "POST", "/chat/sessions", {"course_id": cid}); assert c in (200, 201), (c, ss)
    st, t1, t2, body = sse(user, f"/chat/sessions/{ss['id']}/messages", {"content": text}); return st, t1, t2, body, ss["id"]

# ---- chấm lại / bài nộp của lớp QD105A ----
def subs(cid):
    """{email: submission_id} của bài tập đo ở lớp cid."""
    return {r[0]: r[1] for r in sql(f"select u.email,s.id from submissions s join users u on u.id=s.student_id where s.course_id='{cid}' order by 1")}
def regrade(cid, sid, user="teacher", wait=240):
    """POST chấm lại; đợi round mới DONE/FAILED. Trả (http, round, state, total, flags)."""
    n0 = int(sql(f"select coalesce(max(round),0) from grading_results where submission_id='{sid}'")[0][0])
    c, j, _ = api(user, "POST", f"/courses/{cid}/submissions/{sid}/grade", {})
    if c != 202: return c, None, None, None, str(j)[:150]
    t0 = time.time()
    while time.time() - t0 < wait:
        time.sleep(2)
        r = sql(f"select round,run_state,total,flags from grading_results where submission_id='{sid}' and round>{n0} order by round desc limit 1")
        if r and r[0][1] in ("DONE", "FAILED"): return c, int(r[0][0]), r[0][1], r[0][2], r[0][3]
    return c, None, "TIMEOUT", None, ""
def flag_of(r): return "INJECTION_SUSPECTED" in (r[4] or "")
