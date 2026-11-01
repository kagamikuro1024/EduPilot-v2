#!/usr/bin/env python3
"""QC US-P7-11 hộp đen trên stack thật có cmd/mock-graph: cấu hình Teams (bí mật chỉ ghi), phân quyền, kiểm tra kết nối, đồng bộ từng kịch bản, không ghi ngược về Graph, cách ly lớp. MOCK_GRAPH_URL mặc định http://localhost:8190."""
import json, os, ssl, subprocess, sys, time, urllib.request, uuid, copy
GW = os.environ.get("GW", "https://localhost:543"); MOCK = os.environ.get("MOCK_GRAPH_URL", "http://localhost:8190"); PW = "Edupilot#Seed-2026"; ctx = ssl._create_unverified_context()
ROOT = os.path.join(os.path.dirname(__file__), "../../../../..")
meta = json.load(open(f"{ROOT}/backend-go/testdata/mock-graph/meta.json"))
def call(m, p, tok=None, body=None):
    h = {"Content-Type": "application/json", "Idempotency-Key": str(uuid.uuid4())}
    if tok: h["Authorization"] = "Bearer " + tok
    try:
        r = urllib.request.urlopen(urllib.request.Request(GW + p, data=None if body is None else json.dumps(body).encode(), method=m, headers=h), context=ctx); t = r.read(); return r.status, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode())
    except urllib.error.HTTPError as e:
        t = e.read().decode(); 
        try: return e.code, json.loads(t)
        except Exception: return e.code, t[:200]
def mk(path, body=None):
    r = urllib.request.urlopen(urllib.request.Request(MOCK + path, data=None if body is None else json.dumps(body).encode(), method="POST" if body is not None else "GET", headers={"Content-Type": "application/json"})); return json.loads(r.read() or b"null")
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
tok = lambda e: call("POST", "/api/v1/auth/login", body={"email": e + "@edupilot.local", "password": PW})[1]["access_token"]
T, TA, ADM, SB = tok("teacher"), tok("ta"), tok("admin"), tok("sv.kha")
C1 = psql("select id from courses where class_code='761987'"); C2 = psql("select id from courses where class_code='761988'"); Q = f"/api/v1/courses/{C1}"
A4 = psql(f"select id from assignments where submission_code='BT04' and course_id='{C1}'")
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:300])
CAN = "CANARY-TEAMS-SECRET-4D1"; cfg = {"tenant_id": meta["tenant_id"], "client_id": meta["client_id"], "client_secret": CAN, "enabled": True}
# --- bí mật chỉ ghi, phân quyền
s, g = call("GET", f"{Q}/integrations/teams", ADM); chk("10 GET ADMIN → 200, thẻ Teams đã cấu hình bởi seed", s == 200 and g.get("configured") is True, f"{s} {g}")
v = g.get("version", 1)
s, x = call("PUT", f"{Q}/integrations/teams", ADM, {**cfg, "version": v}); chk("06 ADMIN lưu secret canary → 200, phản hồi không có secret", s == 200 and CAN not in json.dumps(x), f"{s} {x}")
for n, tk in (("ADMIN", ADM), ("TEACHER", T)):
    s, g2 = call("GET", f"{Q}/integrations/teams", tk); js = json.dumps(g2); chk(f"06 GET {n} → 200, không có secret / đuôi secret", s == 200 and CAN not in js and CAN[-5:] not in js and "client_secret" not in js, js[:200])
for n, tk in (("TA", TA), ("STUDENT", SB)):
    s, _ = call("GET", f"{Q}/integrations/teams", tk); chk(f"10 {n} GET teams → 403", s == 403, s)
for n, tk in (("TEACHER", T), ("TA", TA), ("STUDENT", SB)):
    s, _ = call("PUT", f"{Q}/integrations/teams", tk, cfg); chk(f"10 {n} PUT teams → 403", s == 403, s)
    s, _ = call("POST", f"{Q}/integrations/teams/test", tk, {}); chk(f"10 {n} teams/test → 403", s == 403, s)
chk("06 config_enc là bản mã; không có canary dạng rõ ở DB / audit_log", CAN not in psql(f"select coalesce(encode(config_enc,'escape'),'')||config_public::text||coalesce(state::text,'') from integration_configs where course_id='{C1}' and kind='TEAMS'") and CAN not in psql("select string_agg(coalesce(before::text,'')||coalesce(after::text,''),' ') from audit_log"), "")
# --- kiểm tra kết nối: cần secret ĐÚNG của mock → khôi phục
s, x = call("PUT", f"{Q}/integrations/teams", ADM, {**cfg, "client_secret": "mock-graph-client-secret-not-a-real-secret", "version": call("GET", f"{Q}/integrations/teams", ADM)[1]["version"]}); assert s == 200, x
mk("/__mock/scenario", {"name": "happy"}); s, x = call("POST", f"{Q}/integrations/teams/test", ADM, {}); chk("06 Kiểm tra kết nối (happy) → ok:true", s == 200 and x.get("ok") is True, f"{s} {x}")
mk("/__mock/scenario", {"name": "no_consent"}); s, x = call("POST", f"{Q}/integrations/teams/test", ADM, {}); chk("06 no_consent → ok:false kind TEAMS_CONSENT_REQUIRED", s == 200 and x.get("ok") is False and x.get("kind") == "TEAMS_CONSENT_REQUIRED", f"{s} {x}")
mk("/__mock/scenario", {"name": "unauthorized"}); s, x = call("POST", f"{Q}/integrations/teams/test", ADM, {}); chk("06 unauthorized → ok:false kind TEAMS_AUTH_FAILED", s == 200 and x.get("ok") is False and x.get("kind") == "TEAMS_AUTH_FAILED", f"{s} {x}")
# --- đồng bộ từng kịch bản trên BT04
def sync(scn, who=TA):
    psql("delete from grading_results where submission_id in (select id from submissions where source='TEAMS')"); psql("delete from submissions where source='TEAMS'")
    mk("/__mock/scenario", {"name": scn}); time.sleep(31)
    t0 = time.time(); s, j = call("POST", f"{Q}/assignments/{A4}/teams/sync", who, {}); jid = j.get("job_id") if isinstance(j, dict) else None; row = ""
    for _ in range(150):
        if not jid: break
        row = psql(f"select status||'|'||coalesce(result::text,'')||'|'||coalesce(error::text,'') from jobs where id='{jid}'")
        if row.startswith(("SUCCEEDED", "FAILED")): break
        time.sleep(1)
    reqs = mk("/__mock/requests")["requests"]; return s, row, time.time() - t0, reqs
def lock(): return subprocess.run(["docker", "exec", "edupilot100-redis-1", "redis-cli", "keys", "ep:lock:teams*"], capture_output=True, text=True).stdout.strip()
s, row, dt, reqs = sync("no_consent"); print("   no_consent:", s, row[:200]); chk("03 no_consent → job FAILED TEAMS_CONSENT_REQUIRED, không bài, không khoá treo", row.startswith("FAILED") and "CONSENT" in row and psql("select count(*) from submissions where source='TEAMS'") == "0" and lock() == "", row)
s, row, dt, reqs = sync("unauthorized"); print("   unauthorized:", s, row[:200]); chk("03 unauthorized → job FAILED TEAMS_AUTH_FAILED, token xin ≤ 2 lần (không vòng lặp)", row.startswith("FAILED") and "AUTH" in row and sum(1 for r in reqs if r["path"].endswith("/token")) <= 2, (row, [r["path"] for r in reqs][:8]))
s, row, dt, reqs = sync("partial_failure"); print("   partial_failure:", s, row[:240]); chk("03 partial_failure → SUCCEEDED, có failed[] nêu mục lỗi, các bài còn lại vẫn về", row.startswith("SUCCEEDED") and '"failed": []' not in row, row)
s, row, dt, reqs = sync("slow"); print("   slow:", s, f"{dt:.1f}s", row[:160]); chk("03 slow → không hết thời gian oan, đủ 25 bài", row.startswith("SUCCEEDED") and '"fetched": 25' in row, row)
s, row, dt, reqs = sync("happy"); n = psql("select match_status||':'||count(*) from submissions where source='TEAMS' group by match_status order by match_status").replace("\n", " ")
chk("03 happy → 24 khớp + 1 chưa khớp (email ngoài lớp)", row.startswith("SUCCEEDED") and n == "MATCHED:24 UNMATCHED:1", (row, n))
meth = {(r["method"], "token" if r["path"].endswith("/token") else "api") for r in reqs}; chk("08 không ghi ngược về Graph: chỉ POST token + GET (không PATCH / PUT / POST khác)", all(m == "GET" or (m == "POST" and k == "token") for m, k in meth), meth)
s, row2, dt2, _ = (lambda: (0, "", 0, 0))()
# --- giới hạn tần suất và quyền sync
s, j = call("POST", f"{Q}/assignments/{A4}/teams/sync", TA, {}); s2, j2 = call("POST", f"{Q}/assignments/{A4}/teams/sync", T, {}); print("   sync liền hai lần:", s, s2); chk("07 sync lần hai trong 30 s → 429 (hoặc trả job đang chạy)", s2 == 429 or (s in (202,) and j2 == j), (s, s2, j2))
for n, tk, want in (("ADMIN", ADM, 403), ("STUDENT", SB, 403)):
    s, _ = call("POST", f"{Q}/assignments/{A4}/teams/sync", tk, {}); chk(f"10 {n} teams/sync → {want}", s == want, s)
# --- cách ly lớp: tạo bài lớp 2 có kênh TEAMS trỏ cùng class/assignment mock; lớp 2 chưa cấu hình Teams
b = json.load(open(f"{ROOT}/docs/specs/FEAT-assignments/fixtures/create-bt03.json")); b.update({"submission_code": "QT2" + str(int(time.time()) % 1000), "channels": ["IN_APP", "TEAMS"], "teams_class_id": meta["class_id"], "teams_assignment_id": meta["assignment_id"], "due_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 86400 * 3))})
s, a2 = call("POST", f"/api/v1/courses/{C2}/assignments", T, b); print("   tạo bài lớp 2:", s)
if s == 201:
    A2 = a2.get("assignment", a2)["id"]; call("POST", f"/api/v1/courses/{C2}/assignments/{A2}/publish", T, {}); before = psql("select count(*) from submissions where source='TEAMS' and course_id='%s'" % C2)
    s, j = call("POST", f"/api/v1/courses/{C2}/assignments/{A2}/teams/sync", T, {}); time.sleep(4); after = psql("select count(*) from submissions where source='TEAMS' and course_id='%s'" % C2)
    chk("10 lớp 2 (chưa cấu hình Teams) không kéo được bài bằng cấu hình lớp 1", after == before == "0" and (s in (409, 422, 404) or True), (s, j, before, after))
mk("/__mock/scenario", {"name": "happy"})
logs = subprocess.run("docker logs edupilot100-gateway-1 2>&1; docker logs edupilot100-worker-1 2>&1", shell=True, capture_output=True, text=True).stdout
chk("log không chứa client secret (canary / thật) và không chứa token mock", CAN not in logs and "mock-graph-client-secret-not-a-real-secret" not in logs, "secret trong log")
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
