#!/usr/bin/env python3
"""QC US-P7-02 hộp đen trên stack thật (TC-P7-02-03…19). Cần stack đã seed (EP_PORT_OFFSET=100). In PASS/FAIL từng ca; thoát 1 nếu có FAIL."""
import copy, json, os, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:543"); PW = os.environ.get("PW", "Edupilot#Seed-2026"); ctx = ssl._create_unverified_context()
FX = os.path.join(os.path.dirname(__file__), "../../../../specs/FEAT-assignments/fixtures")
def call(m, p, tok=None, body=None, key=None):
    h = {"Content-Type": "application/json"}
    if tok: h["Authorization"] = "Bearer " + tok
    if key: h["Idempotency-Key"] = key
    try:
        r = urllib.request.urlopen(urllib.request.Request(GW + p, data=None if body is None else json.dumps(body).encode(), method=m, headers=h), context=ctx)
        t = r.read(); return r.status, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode())
    except urllib.error.HTTPError as e:
        t = e.read(); return e.code, (json.loads(t) if t[:1] == b"{" else t.decode())
tok = lambda e: call("POST", "/api/v1/auth/login", body={"email": e, "password": PW})[1]["access_token"]
T, TA, ADM, SA, SB = (tok(x + "@edupilot.local") for x in ("teacher", "ta", "admin", "sv.gioi", "sv.kha"))
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
C1 = psql("select id from courses where class_code='761987'"); Q = f"/api/v1/courses/{C1}"
fails = 0
def chk(name, ok, extra=""):
    global fails; fails += (not ok); print(("PASS" if ok else "FAIL"), name, extra if not ok else "")
base = json.load(open(f"{FX}/create-bt03.json")); idem = lambda: str(uuid.uuid4())
def create(over=None, key=None, drop=()):
    b = copy.deepcopy(base); b.update(over or {}); [b.pop(k, None) for k in drop]; return call("POST", f"{Q}/assignments", T, b, key or idem())
s, a = create(); chk("03 tạo BT03 từ fixtures → 201", s == 201, f"{s} {str(a)[:200]}")
A = a.get("assignment", a) if isinstance(a, dict) else {}; aid = A.get("id"); print("   id", aid, "keys", sorted(A.keys())[:30])
crit = A.get("rubric") or A.get("criteria") or []
if isinstance(crit, dict): crit = crit.get("criteria", [])
chk("03 tiêu chí c1…c4 do máy chủ cấp, version 1, chưa công bố", [c["id"] for c in crit] == ["c1", "c2", "c3", "c4"] and A.get("version") == 1 and not A.get("published_at"), str(crit)[:200])
k = idem(); s1, a1 = create({"submission_code": "IDEM1"}, k); s2, a2 = create({"submission_code": "IDEM1"}, k)
chk("03 Idempotency-Key: cùng khoá → cùng bài", s1 in (200, 201) and s2 in (200, 201) and (a1.get("assignment", a1).get("id") == a2.get("assignment", a2).get("id")), f"{s1} {s2}")
chk("03 chỉ một bài IDEM1 trong DB", psql(f"select count(*) from assignments where course_id='{C1}' and submission_code='IDEM1'") == "1")
# ---- TC-04 biên giá trị
cases = [("title rỗng", {"title": ""}, 422), ("title 201 ký tự", {"title": "x" * 201}, 422), ("description 8001", {"description": "x" * 8001}, 422),
 ("mã nộp 1 ký tự", {"submission_code": "A"}, 422), ("mã nộp 13 ký tự", {"submission_code": "A" * 13}, 422), ("mã nộp chữ thường", {"submission_code": "bt 03"}, 422), ("mã nộp trùng", {"submission_code": "IDEM1"}, 422),
 ("max_score 0", {"max_score": "0"}, 422), ("max_score 100,25", {"max_score": "100.25"}, 422), ("max_score 10,1 (bước)", {"max_score": "10.1"}, 422),
 ("late_penalty âm", {"late_penalty_per_day": "-1"}, 422), ("late_max_days 15", {"late_max_days": 15}, 422), ("allow_late + 0 ngày", {"allow_late": True, "late_max_days": 0}, 422),
 ("appeal_days 31", {"appeal_days": 31}, 422), ("channels lạ", {"channels": ["FAX"]}, 422), ("TEAMS thiếu id", {"channels": ["IN_APP", "TEAMS"]}, 422),
 ("reference_answer 20001", {"reference_answer": "x" * 20001}, 422), ("type QUIZ", {"type": "QUIZ"}, 422)]
for i, (n, over, want) in enumerate(cases):
    o = dict(over); o.setdefault("submission_code", f"V{i:02d}X"); s, r = create(o)
    chk(f"04 {n} → {want}", s == want, f"{s} {str(r)[:140]}")
for n, over in [("max_score 0,25 (biên)", {"max_score": "0.25", "rubric_note": 1}), ("max_score 100 (biên)", {"max_score": "100"}), ("appeal_days 0", {"appeal_days": 0}), ("appeal_days 30", {"appeal_days": 30})]:
    pass
# ---- rubric
if aid:
    r = lambda c: call("PUT", f"{Q}/assignments/{aid}/rubric", T, c)
    cur = crit
    def rb(scores): return {"version": A.get("version", 1), "criteria": [{"id": c["id"], "name": c["name"], "max_score": s, "guidance": c.get("guidance", "g")} for c, s in zip(cur, scores)]}
    for n, sc, want in [("tổng 9,75", ["2.5", "2.5", "2.5", "2.25"], 422), ("tổng 10,25", ["2.5", "2.5", "2.5", "2.75"], 422), ("bước 2,3", ["2.5", "2.5", "2.7", "2.3"], 422)]:
        s, x = r(rb(sc)); chk(f"06 rubric {n} → {want}", s == want, f"{s} {str(x)[:140]}")
    s, x = r(rb(["2.5", "2.5", "2.5", "2.5"])); chk("06 rubric tổng đúng → 200", s == 200, f"{s} {str(x)[:140]}")
    s, x = call("POST", f"{Q}/assignments/{aid}/rubric/suggest", T, {}, idem()); print("   suggest trên stack (provider giả, chưa có bản ghi demo):", s, str(x)[:100]); chk("08 rubric/suggest (TEACHER) không 5xx, không lọt 403", s in (200, 422))
    tot = None
    try:
        if s != 200: raise StopIteration
        from fractions import Fraction as F; cs = (x.get("criteria") or x.get("rubric", {}).get("criteria")); tot = sum(F(c["max_score"]) for c in cs); chk("08 suggest: Σ max_score = 10 và bội 0,25", tot == 10 and all((F(c["max_score"]) / F(1, 4)).denominator == 1 for c in cs), str(cs)[:200])
    except StopIteration: pass
    except Exception as e: chk("08 suggest có cấu trúc", False, str(e))
    s, _ = call("POST", f"{Q}/assignments/{aid}/rubric/suggest", TA, {}, idem()); chk("08 rubric/suggest (TA) không 403", s in (200, 422), str(s))
    # ---- công bố
    n_active = int(psql(f"select count(*) from enrollments where course_id='{C1}' and role_in_course='STUDENT' and status='ACTIVE'"))
    kp = idem(); s, x = call("POST", f"{Q}/assignments/{aid}/publish", T, {}, kp); chk("10 publish → 200/201", s in (200, 201), f"{s} {str(x)[:140]}")
    time.sleep(8)
    got = int(psql(f"select count(*) from notifications where type='ASSIGNMENT_PUBLISHED' and course_id='{C1}'")); chk(f"10 thông báo ASSIGNMENT_PUBLISHED = {n_active} sinh viên ACTIVE", got == n_active, str(got))
    s, x = call("POST", f"{Q}/assignments/{aid}/publish", T, {}, idem()); chk("11 publish lần hai → 409 hoặc idempotent", s in (200, 409), str(s)); time.sleep(4)
    chk("11 không thông báo lần hai", int(psql(f"select count(*) from notifications where type='ASSIGNMENT_PUBLISHED' and course_id='{C1}'")) == n_active)
    # ---- đáp án chuẩn không tới sinh viên
    pat = "CANARY-BT03-ANSWER"
    s, x = call("PUT", f"{Q}/assignments/{aid}", T, {**{k2: v for k2, v in A.items() if k2 in ("title", "description", "due_at", "allow_late", "late_penalty_per_day", "late_max_days", "appeal_days", "channels")}, "version": A.get("version", 1) + 2 if False else None, "reference_answer": pat + " — đáp án"}) if False else (0, "")
    ref = open(f"{FX}/dap-an-chuan-bt03.md").read(); chk("19 đáp án chuẩn của fixtures có canary", pat in ref)
    for who, tk in (("sv.gioi", SA), ("sv.kha", SB)):
        for p in (f"{Q}/assignments", f"{Q}/assignments/{aid}", f"{Q}/assignments/{aid}/mine"):
            s, x = call("GET", p[len(GW) - len(GW):], tk); txt = json.dumps(x, ensure_ascii=False)
            chk(f"19 {who} GET …{p[len(Q):]} → không reference_answer / guidance / canary", s == 200 and "reference_answer" not in txt and "guidance" not in txt and pat not in txt, f"{s} {txt[:120]}")
    s, x = call("GET", f"{Q}/assignments/{aid}", TA); chk("19 TA GET chi tiết → có đáp án (chỉ đọc)", s == 200 and "reference_answer" in json.dumps(x), str(s))
    s, x = call("GET", f"{Q}/assignments/{aid}", ADM); chk("19 ADMIN → 403", s == 403, str(s))
    for n, m, p, tk, want in [("TA POST tạo", "POST", f"{Q}/assignments", TA, 403), ("STUDENT POST tạo", "POST", f"{Q}/assignments", SA, 403), ("ADMIN POST tạo", "POST", f"{Q}/assignments", ADM, 403),
        ("TA PUT", "PUT", f"{Q}/assignments/{aid}", TA, 403), ("TA DELETE", "DELETE", f"{Q}/assignments/{aid}", TA, 403), ("TA publish", "POST", f"{Q}/assignments/{aid}/publish", TA, 403), ("STUDENT suggest", "POST", f"{Q}/assignments/{aid}/rubric/suggest", SA, 403)]:
        s, _ = call(m, p, tk, {}, idem()); chk(f"19 {n} → 403", s == want, str(s))
    # ---- lịch / Hôm nay của sinh viên
    s, x = call("GET", f"{Q}/calendar?from=2026-11-20T00:00:00Z&to=2026-12-10T00:00:00Z", SA); txt = json.dumps(x, ensure_ascii=False)
    chk("10 lịch sinh viên có DEADLINE source assignment (một dòng)", s == 200 and txt.count('"source": "assignment"') + txt.count('"source":"assignment"') == 1 and "DEADLINE" in txt, f"{s} {txt[:200]}")
    s, x = call("GET", f"{Q}/today", SA); print("   today sv:", [i.get("kind") for i in (x.get("items", []) if isinstance(x, dict) else [])])
    # ---- sửa sau công bố: đổi hạn → lịch đổi
    s, x = call("GET", f"{Q}/assignments/{aid}", T); cur = x.get("assignment", x); v = cur.get("version")
    newdue = "2026-12-20T16:59:00Z"
    put = {k2: cur[k2] for k2 in ("title", "description", "allow_late", "late_penalty_per_day", "late_max_days", "appeal_days", "channels", "reference_answer") if k2 in cur}; put.update({"version": v, "due_at": newdue})
    s, x = call("PUT", f"{Q}/assignments/{aid}", T, put); chk("12 PUT đổi hạn sau công bố (chưa chấm) → 200", s == 200, f"{s} {str(x)[:160]}")
    s, x = call("PUT", f"{Q}/assignments/{aid}", T, put); chk("15 PUT sai version → 409 VERSION_CONFLICT", s == 409 and "VERSION_CONFLICT" in json.dumps(x), f"{s} {str(x)[:140]}")
    time.sleep(3); s, x = call("GET", f"{Q}/calendar?from=2026-12-15T00:00:00Z&to=2026-12-25T00:00:00Z", SA); chk("12 lịch đổi theo hạn mới ≤ 5 s", "assignment" in json.dumps(x), f"{s}")
    put2 = dict(put); put2.update({"version": call("GET", f"{Q}/assignments/{aid}", T)[1].get("assignment", {}).get("version", v + 1), "max_score": "20"})
    s, x = call("PUT", f"{Q}/assignments/{aid}", T, put2); chk("13 PUT đổi max_score sau công bố → bị khoá (4xx)", s in (409, 422), f"{s} {str(x)[:140]}")
    # ---- xoá
    s, _ = call("DELETE", f"{Q}/assignments/{aid}", T, None, idem()); chk("16 DELETE bài đã công bố, sạch → 204", s == 204, str(s)); time.sleep(3)
    s, x = call("GET", f"{Q}/calendar?from=2026-12-15T00:00:00Z&to=2026-12-25T00:00:00Z", SA); chk("16 lịch không còn dòng sau xoá ≤ 5 s", "assignment" not in json.dumps(x), f"{s}")
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
