#!/usr/bin/env python3
"""TC-P7-11-02 (paged + lần hai), 11-08 (happy rồi throttle), 11-10 (tắt mock-graph giữa chừng → UNREACHABLE, chạy lại). Lớp QD9T (thẻ Teams + BT04 do q9p11 dựng)."""
import sys, os, json, urllib.request; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("11b"); MOCK = "http://localhost:8590"
def mk(path, body=None):
    r = urllib.request.urlopen(urllib.request.Request(MOCK + path, data=None if body is None else json.dumps(body).encode(), method="POST" if body is not None else "GET", headers={"Content-Type": "application/json"})); return json.loads(r.read() or b"{}")
C1 = course("QD9T"); Q = f"/api/v1/courses/{C1}"; TA = tok("ta"); A4 = psql(f"select id from assignments where submission_code='BT04' and course_id='{C1}'")
def clear(): psql(f"delete from grading_results where submission_id in (select id from submissions where source='TEAMS' and course_id='{C1}')"); psql(f"delete from submissions where source='TEAMS' and course_id='{C1}'")
lock = lambda: redis("keys", "ep:lock:teams*")
def sync(scn=None, wait=True, keep=False, who=None):
    if scn: mk("/__mock/scenario", {"name": scn})
    if not keep: clear()
    time.sleep(31)
    t0 = time.time(); s, j, _ = call("POST", f"{Q}/assignments/{A4}/teams/sync", who or TA, {}); jid = j.get("job_id") if isinstance(j, dict) else None; row = ""; prog = []
    for _ in range(200):
        if not jid or not wait: break
        row = psql(f"select status||'|'||progress||'|'||coalesce(result::text,'')||'|'||coalesce(error::text,'') from jobs where id='{jid}'"); prog.append(row.split("|")[1] if row else "")
        if row.startswith(("SUCCEEDED", "FAILED")): break
        time.sleep(0.7)
    return s, j, row, time.time() - t0, prog, jid
cnt = lambda: psql(f"select match_status||':'||count(*) from submissions where source='TEAMS' and course_id='{C1}' group by match_status order by match_status").replace("\n", " ")
# ---- 11-02
s, j, row, dt, prog, _ = sync("paged"); rq = mk("/__mock/requests"); pages = sum(1 for r in rq["requests"] if "/submissions" in r["path"]); print("   paged:", row[:160], cnt(), "trang submissions:", pages, "yêu cầu:", len(rq["requests"]))
R.chk("TC-P7-11-02", f"paged (30 bản ghi / trang 10): {row.split('|')[0]} {cnt()} — {pages} yêu cầu trang bài nộp", row.startswith("SUCCEEDED") and "MATCHED:24" in cnt() and "UNMATCHED:1" in cnt() and pages >= 3, (row, cnt()))
before = psql(f"select count(*) from submissions where source='TEAMS' and course_id='{C1}'")
s, j, row2, dt, prog, _ = sync(None, keep=True); after = psql(f"select count(*) from submissions where source='TEAMS' and course_id='{C1}'"); print("   lần hai:", row2[:200], before, "→", after)
R.chk("TC-P7-11-02", f"đồng bộ lần hai ngay: {before} → {after} bài (0 bài mới, UNIQUE (course_id, source, external_ref, external_ts))", before == after == "25", (before, after, row2))
# ---- 11-08
s, j, row, dt, prog, _ = sync("happy"); print("   happy:", row[:120], f"{dt:.1f}s", "tiến độ:", sorted(set(prog), key=lambda x: int(x or 0))[:12])
R.chk("TC-P7-11-08", f"happy: {row.split('|')[0]} đủ 25 ({cnt()}), tiến độ tăng dần {sorted(set(prog), key=lambda x: int(x or 0))}", row.startswith("SUCCEEDED") and '"fetched": 25' in row, row)
s, j, row, dt, prog, _ = sync("throttle"); rq = mk("/__mock/requests"); print("   throttle:", row[:140], f"{dt:.1f}s", "throttled:", rq.get("throttled"))
R.chk("TC-P7-11-08", f"throttle: {row.split('|')[0]} đủ 25 ({cnt()}) sau {rq.get('throttled')} lần 429, {dt:.1f} s, tiến độ không đứng: {sorted(set(prog), key=lambda x: int(x or 0))}", row.startswith("SUCCEEDED") and '"fetched": 25' in row and (rq.get("throttled") or 0) >= 2, (row, rq.get("throttled")))
# ---- 11-10 tắt mock-graph giữa chừng
mk("/__mock/scenario", {"name": "slow"}); clear(); time.sleep(31)
s, j, _ = call("POST", f"{Q}/assignments/{A4}/teams/sync", TA, {}); jid = j.get("job_id"); time.sleep(3)
subprocess.run(["docker", "stop", "ep-qc-mock-graph-1"], capture_output=True)
try:
    row = ""
    for _ in range(150):
        row = psql(f"select status||'|'||coalesce(result::text,'')||'|'||coalesce(error::text,'') from jobs where id='{jid}'")
        if row.startswith(("SUCCEEDED", "FAILED")): break
        time.sleep(1)
    print("   mock-graph tắt giữa chừng:", row[:200], "bài đã về:", cnt(), "khoá:", repr(lock()))
    R.chk("TC-P7-11-10", f"tắt mock-graph giữa chừng → {row[:80]}; khoá ep:lock:teams* rỗng ({lock()!r})", row.startswith("FAILED") and "UNREACHABLE" in row.upper() and lock() == "", row)
finally:
    subprocess.run(["docker", "start", "ep-qc-mock-graph-1"], capture_output=True)
    for _ in range(40):
        if "healthy" in subprocess.run(["docker", "ps", "--filter", "name=ep-qc-mock-graph-1", "--format", "{{.Status}}"], capture_output=True, text=True).stdout: break
        time.sleep(2)
mk("/__mock/scenario", {"name": "happy"}); s, j, row, dt, prog, _ = sync("happy", keep=True)
R.chk("TC-P7-11-10", f"chạy lại sau khi mock-graph bật: {row.split('|')[0]}, {cnt()} (đủ 25, không đôi, không panic), khoá {lock()!r}", row.startswith("SUCCEEDED") and psql(f"select count(*) from submissions where source='TEAMS' and course_id='{C1}'") == "25" and lock() == "", row)
mk("/__mock/scenario", {"name": "happy"})
sys.exit(R.end())
