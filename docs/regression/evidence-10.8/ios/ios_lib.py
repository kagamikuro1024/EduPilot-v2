# TC-AS09-34 trên iOS Simulator: API ep-qc (https://localhost:943) bằng thư viện dmlib của QC (không dùng ep-r, không dùng sql()).
import os, sys, json, time, datetime, subprocess
os.environ.setdefault("QC_API", "https://localhost:943/api/v1")
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "scripts"))
import dmlib
dmlib.RES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results-ios.jsonl")
from dmlib import api
T, ADMIN = "teacher", "admin"
CODE_CLASS, JOIN = "QCIOS1", "QCKSTXW"
STATE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "state-ios.json")
def psql(q):
    r = subprocess.run(["docker", "exec", "-i", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-F", "|", "-c", q], capture_output=True, text=True)
    assert r.returncode == 0, r.stderr
    return [x.split("|") for x in r.stdout.strip().split("\n") if x]
def uid(u): return psql(f"select id from users where email='{u}@edupilot.local'")[0][0]
def setup():
    ex = psql(f"select id from courses where class_code='{CODE_CLASS}'")
    if ex: cid = ex[0][0]
    else:
        c, j, _ = api(ADMIN, "POST", "/admin/courses", {"subject_code": "QCIOS", "class_code": CODE_CLASS, "name": "QC iOS Simulator " + CODE_CLASS, "semester": "2026-2027-HK1", "teacher_id": uid(T), "join_code": JOIN})
        assert c == 201, (c, j); cid = j["id"]
    for u in ("sv.kha", "sv.gioi"):
        c, j, _ = api(u, "POST", "/courses/join", {"code": JOIN}); assert c in (200, 409), (u, c, j)
    t = datetime.date.today()
    if not psql(f"select 1 from class_sessions where course_id='{cid}' limit 1"):
        c, j, _ = api(T, "POST", f"/courses/{cid}/sessions/generate", {"weekdays": [1,2,3,4,5,6,7], "start_time": "00:05", "end_time": "23:50", "from": str(t - datetime.timedelta(days=0)), "to": str(t + datetime.timedelta(days=20))}); assert c == 201, (c, j)
    c, j, _ = api(T, "GET", f"/courses/{cid}"); v = j["settings"]["version"]
    c, j, _ = api(T, "PUT", f"/courses/{cid}/settings", {"attendance": {"self_checkin": True, "rotate_seconds": 60}, "version": v}); assert c == 200, (c, j)
    json.dump({"cid": cid}, open(STATE, "w")); return cid
def sessions(cid):
    c, j, _ = api(T, "GET", f"/courses/{cid}/sessions?limit=100"); assert c == 200, (c, j); return j["items"]
def open_window(cid, sid, minutes=30):
    c, j, _ = api(T, "POST", f"/courses/{cid}/sessions/{sid}/checkin-windows", {"minutes": minutes}); assert c == 201, (c, j)
    c, d, _ = api(T, "GET", f"/courses/{cid}/sessions/{sid}/checkin-windows/{j['id']}/display"); assert c == 200, (c, d)
    return j["id"], d
if __name__ == "__main__":
    cid = setup(); ss = sessions(cid); print(cid, len(ss), json.dumps(ss[0], ensure_ascii=False)[:300])
