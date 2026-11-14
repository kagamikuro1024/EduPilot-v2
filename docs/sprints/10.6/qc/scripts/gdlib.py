import sys, os, json, time
sys.path.insert(0, os.path.expanduser("~/Documents/TA_Agent_qc106/docs/sprints/10.5/qc/scripts"))
from dmx import *
C1 = course("sv.kha", "761987")
A = sql(f"select id from assignments where course_id='{C1}' and title like 'Bài tập 03%' order by created_at limit 1")[0][0]   # BT03 lớp 1 (id đổi mỗi lần seed)
G = f"/courses/{C1}/assignments/{A}/guidance"
FL = f"/courses/{C1}/guidance-flags"

def flag(feature, enabled, user="teacher"):
    c, j, _ = api(user, "GET", FL)
    f = [x for x in (j.get("items") or j.get("flags") or j.get("features") or []) if x["feature"] == feature][0]
    return api(user, "PUT", f"{FL}/{feature}", {"enabled": enabled, "version": f["version"]})

def stub(mode): open("/tmp/stub_mode", "w").write(mode)

def route(task, model_id):
    c, j, _ = api("admin", "GET", "/admin/llm/routes")
    cur = [t for t in j["items"] if t["task"] == task][0]
    return api("admin", "PUT", "/admin/llm/routes", {"task": task, "chain": [model_id] if model_id else [], "version": cur.get("version", 0)})

def results():
    return sql(f"select r.id, r.version, r.status, r.round, r.guidance_version, r.run_state from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.round=(select max(round) from grading_results x where x.submission_id=r.submission_id) order by r.id")

def jobinfo(jid):
    return sql(f"select status, coalesce(result::text,''), coalesce(error::text,'') from jobs where id='{jid}'") [0]

def wait_job(jid, t=60):
    for _ in range(t):
        s = jobinfo(jid)
        if s[0] in ("DONE", "FAILED", "SUCCEEDED", "COMPLETED", "ERROR"): return s
        time.sleep(1)
    return jobinfo(jid)

def bump():
    sql(f"update grading_results set reviewed_at=now() where id=(select r.id from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status in ('REVIEWED','PUBLISHED') order by r.id limit 1)")
    sql("select 1")
    os.system(f"docker exec ep-qc-redis-1 redis-cli --scan --pattern 'ep:rl:reflect:*' | xargs -r docker exec -i ep-qc-redis-1 redis-cli del >/dev/null 2>&1")

def drafts(status="DRAFT"):
    return sql(f"select id, criterion_id, direction, support, left(text,70), array_length(source_result_ids,1), origin_job from grading_guidance_items where assignment_id='{A}' and status='{status}' order by created_at, id")

def reflect(mode, key):
    stub(mode); bump()
    c, j, _ = api("teacher", "POST", G + "/reflect", {}, idem=f"{key}-{int(time.time())}")
    if c != 202: return c, j, None
    return c, j, wait_job(j["job_id"])
