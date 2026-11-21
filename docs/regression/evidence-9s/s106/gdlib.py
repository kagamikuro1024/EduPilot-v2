# Sprint 10.6 (US-GD-*) trên ep-qc: lớp QD9G (bản sao BT03 do Smoke9Runner dựng; KHÔNG đụng 761987/761988). Kết quả ghi results_106.jsonl.
import sys, os, json, time, re, uuid, subprocess
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10"))
from qlib import *
RM = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results_106.jsonl")
def res(tc, kq, note=""):
    print(f"{kq} {tc} {note}"[:300]); open(RM, "a", encoding="utf8").write(json.dumps({"tc": tc, "kq": kq, "note": note}, ensure_ascii=False) + "\n")
def chk(tc, cond, note=""): res(tc, "P" if cond else "F", note); return cond
def lastj(j, n=160): return json.dumps(j, ensure_ascii=False)[:n]
def cid(code): return sql(f"select id from courses where class_code='{code}'")[0][0]
CG0 = cid("QD9G"); CG = CG0   # lớp QD9G có bài BT03 với kết quả có điểm tiêu chí; bài tập thử do QC dựng thêm
def assignments(min_subs=10):
    return [r[0] for r in sql(f"select a.id from assignments a where a.course_id='{CG}' and a.type='ESSAY' and (select count(*) from submissions s where s.assignment_id=a.id) >= {min_subs} order by (select count(*) from submissions s where s.assignment_id=a.id) desc, a.id")]
def G(a, c=CG): return f"/courses/{c}/assignments/{a}/guidance"
def FL(c=CG): return f"/courses/{c}/guidance-flags"
def flag(feature, enabled, c=CG, user="teacher"):
    cc, j, _ = api(user, "GET", FL(c)); f = [x for x in (j.get("items") or []) if x["feature"] == feature][0]
    return api(user, "PUT", f"{FL(c)}/{feature}", {"enabled": enabled, "version": f["version"]})
def stub(mode): open("/Users/kuro/.config/edupilot-team/qc3-work/stub_mode", "w").write(mode)
def results(a): return sql(f"select r.id, r.version, r.status, r.round, r.guidance_version, r.run_state from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{a}' order by r.created_at")
def wait_job(jid, t=60):
    for _ in range(t):
        r = sql(f"select status, coalesce(result::text,''), coalesce(error::text,'') from jobs where id='{jid}'")[0]
        if r[0] in ("DONE", "FAILED", "SUCCEEDED", "COMPLETED", "ERROR", "DEAD"): return r
        time.sleep(1)
    return r
def redis_del(pat): os.system(f"docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern '{pat}' | xargs -r redis-cli del\" >/dev/null 2>&1")

import contextlib
def _prov(): return {p["name"]: p for p in api("admin", "GET", "/admin/llm/providers")[1]["items"]}
def set_route(task, name, model):
    p = _prov()[name]; mid = [m["id"] for m in p["models"] if m["model"] == model][0]
    r = [x for x in api("admin", "GET", "/admin/llm/routes")[1]["items"] if x["task"] == task][0]
    return api("admin", "PUT", "/admin/llm/routes", {"task": task, "chain": [mid], "params": r["params"], "version": r["version"]})[0]
@contextlib.contextmanager
def window(wait=600):
    """Giữ /tmp/qc-global.lock, bật stub GD (cổng 9763) + route GRADING → Q106-stub; trả lại 9router khi thoát."""
    t0 = time.time()
    while True:
        try: os.mkdir("/tmp/qc-global.lock"); break
        except FileExistsError:
            if time.time() - t0 > wait: raise RuntimeError("LOCKED")
            time.sleep(5)
    open("/tmp/qc-global.lock/owner", "w").write("QC3 10.6 stub %s" % time.strftime("%H:%M"))
    p = None
    try:
        if subprocess.run(["curl", "-s", "-m", "2", "http://localhost:9763/x"], capture_output=True).returncode != 0:
            p = subprocess.Popen(["python3", os.path.join(os.path.dirname(os.path.abspath(__file__)), "stub_gd.py"), "9763"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL); time.sleep(1)
        if "Q106-stub" not in _prov(): print("tạo provider", api("admin", "POST", "/admin/llm/providers", {"name": "Q106-stub", "type": "openai_compatible", "base_url": "http://host.docker.internal:9763/v1", "api_key": "stub", "enabled": True, "models": [{"model": "stub-chat", "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]})[0])
        stub("allow"); print("GRADING → stub", set_route("GRADING", "Q106-stub", "stub-chat")); redis_del("ep:llm:cb:*"); redis_del("ep:guidance:reflect:*")
        yield
    finally:
        stub("allow"); print("GRADING → 9router", set_route("GRADING", "QC 9router", "ag/gemini-3.8-flash"))
        if p: p.terminate()
        import shutil; shutil.rmtree("/tmp/qc-global.lock", ignore_errors=True)

def fresh(k=1):
    """Bài tập QD9P còn nguyên (40 kết quả DRAFT/DONE, chưa duyệt, chưa có lưu ý, version 0, chưa dùng trong tệp /Users/kuro/.config/edupilot-team/qc3-work/gd_used.json)."""
    used = set(json.load(open("/Users/kuro/.config/edupilot-team/qc3-work/gd_used.json"))) if os.path.exists("/Users/kuro/.config/edupilot-team/qc3-work/gd_used.json") else set()
    rows = [r[0] for r in sql(f"select a.id from assignments a where a.course_id='{CG}' and a.type='ESSAY' and a.guidance_version=0 and not exists (select 1 from grading_guidance_items i where i.assignment_id=a.id) and (select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id=a.id and r.status='DRAFT' and r.run_state='DONE')=40 order by a.id") if r[0] not in used]
    pick = rows[:k]; used |= set(pick); json.dump(sorted(used), open("/Users/kuro/.config/edupilot-team/qc3-work/gd_used.json", "w")); return pick[0] if k == 1 else pick
def done_results(a, n=None): 
    r = [x for x in results(a) if x[2] == "DRAFT" and x[5] == "DONE"]
    return r[:n] if n else r
def _sql_score(rid, crit, val):
    sql(f"update grading_results set criteria = (select jsonb_agg(case when e->>'id'='{crit}' then jsonb_set(jsonb_set(e,'{{score}}','\"{val}\"'),'{{edited}}','true') else e end order by ord) from jsonb_array_elements(criteria) with ordinality t(e,ord)), version = version where id='{rid}'")
def review_with(a, specs):
    """specs: [(idx, {crit: teacher_score_str}|None)] — sửa điểm tiêu chí (nếu có) rồi duyệt. Trả [(rid, ok)]."""
    rs = done_results(a); out = []
    for idx, edits in specs:
        rid = rs[idx][0]; c, j, _ = api("teacher", "GET", f"/courses/{CG}/grading-results/{rid}"); ver = j["result"]["version"]
        stepbad = {k: v for k, v in (edits or {}).items() if round(float(v) * 4) != float(v) * 4}
        for k, v in stepbad.items(): _sql_score(rid, k, v)
        edits = {k: v for k, v in (edits or {}).items() if k not in stepbad}
        if edits:
            c, j2, _ = api("teacher", "PUT", f"/courses/{CG}/grading-results/{rid}", {"version": ver, "criteria": [{"id": k, "score": v} for k, v in edits.items()]})
            if c != 200: out.append((rid, f"PUT {c} {lastj(j2, 100)}")); continue
            ver = j2["result"]["version"]
        c, j3, _ = api("teacher", "POST", f"/courses/{CG}/grading-results/{rid}/review", {"version": ver}); out.append((rid, c if c != 200 else "ok"))
    return out
def reset_reflect_limit(): redis_del("ep:rl:reflect:*"); redis_del("ep:llm:cb:*"); redis_del("ep:guidance:reflect:*")
def reflect(a, mode, key=None):
    stub(mode); reset_reflect_limit()
    c, j, _ = api("teacher", "POST", G(a) + "/reflect", {}, idem=key or str(uuid.uuid4()))
    if c != 202: return c, j, None
    return c, j, wait_job(j["job_id"])
def drafts(a, status="DRAFT"): return sql(f"select id, criterion_id, direction, support, left(text,60) from grading_guidance_items where assignment_id='{a}' and status='{status}' order by created_at")

SRC_A = "01a1ef59-643e-75f6-8bb4-eb13a7c7743d"  # BT03 QD9G (25 bài, rubric c1..c4)
def mk_assignment(tag, n=12, typ="ESSAY", src=SRC_A, late=False):
    """Nhân bản BT03 (rubric + n bài nộp) thành bài tập mới của QD9G; trả id. Chấm bằng grade-all trong window()."""
    new = str(uuid.uuid4())
    sql(f"""create temp table ta as select * from assignments where id='{src}';
update ta set due_at=case when {str(late).lower()} then now() - interval '5 days' else due_at end, id='{new}', title='QC106 {tag}', type='{typ}', guidance_version=0, published_at=null, version=1, submission_code='QC'||upper(substr(md5(random()::text),1,8)), teams_class_id=null, teams_assignment_id=null;
insert into assignments select * from ta;
create temp table ts as select * from submissions where assignment_id='{src}' and match_status='MATCHED' order by created_at limit {n};
update ts set received_at=case when {str(late).lower()} then now() - interval '3 days' else received_at end, id=uuidv7(), assignment_id='{new}', sha256=md5(random()::text)||md5(random()::text), blob_key='qc106/'||md5(random()::text), external_ref=null, version=1;
insert into submissions select * from ts;""")
    return new
def grade_all_wait(a, t=120):
    c, j, _ = api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/grade-all", {}); 
    for _ in range(t):
        n = sql(f"select count(*) filter (where r.run_state='DONE'), count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}'")[0]
        sub = sql(f"select count(*) from submissions where assignment_id='{a}'")[0][0]
        if n[0] == sub: break
        time.sleep(1)
    return c, j, n
def fxa(tag, n=12, typ="ESSAY", late=False):
    a = mk_assignment(tag, n, typ, late=late); c, j, nn = grade_all_wait(a); assert nn[0] == nn[1], (c, j, nn); return a

def D(a, iid): return f"{G(a)}/items/{iid}/decide"
def items_of(a, status=None):
    c, j, _ = api("teacher", "GET", G(a)); it = j.get("items") or []
    return [x for x in it if status is None or x["status"] == status]
def decide(a, iid, dec="APPROVE", text=None):
    it = [x for x in items_of(a) if x["id"] == iid][0]; b = {"decision": dec, "version": it["version"]}
    if text: b["text"] = text
    return api("teacher", "POST", D(a, iid), b)
def apply_v(a, expected, key=None): return api("teacher", "POST", G(a) + "/apply", {"expected_version": expected}, idem=key or str(uuid.uuid4()))
def guided(tag, n=12, k=8, approve=2, do_apply=True, late=False):
    """Bài tập đã có lưu ý: chấm n bài, duyệt k bài có sửa (c2,c3 → 0), reflect rf_many, duyệt `approve` mục, áp dụng → v1. Gọi TRONG window()."""
    flag("GRADING_GUIDANCE", True); a = fxa(tag, n, late=late)
    review_with(a, [(i, {"c2": "0.00", "c3": "0.00"}) for i in range(k)])
    c, j, jb = reflect(a, "rf_many"); stub("allow")
    ids = [x["id"] for x in items_of(a, "DRAFT")]
    for iid in ids[:approve]: decide(a, iid)
    v = None
    if do_apply: v = apply_v(a, 0)
    return a, ids, (c, jb and jb[0], v)
def stub_prompts(t0, marker=None):
    """Các yêu cầu GRADING (system chứa «Lưu ý» hoặc rubric) gửi tới stub kể từ t0: [(mode, có_khối, phiên_bản)]."""
    out = []
    for l in open("/Users/kuro/.config/edupilot-team/qc3-work/stub_req.jsonl"):
        r = json.loads(l)
        if r["t"] < t0: continue
        txt = "\n".join(m.get("content", "") if isinstance(m.get("content"), str) else "" for m in r["body"].get("messages", []))
        m = re.search(r"Lưu ý của giảng viên \(bản v(\d+)\)", txt)
        out.append((r["mode"], bool(m), m.group(1) if m else None, "rút ra lưu ý chấm" in txt))
    return out
