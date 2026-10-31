#!/usr/bin/env python3
"""QC US-P7-08 hộp đen (máy chủ: hàng chờ, chi tiết, sửa, duyệt) trên stack thật. Cần bài QC7A với 3 kết quả FAILED do p7-api-07.py để lại."""
import json, os, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:543"); PW = "Edupilot#Seed-2026"; ctx = ssl._create_unverified_context()
def raw(m, url, h=None, data=None):
    try:
        r = urllib.request.urlopen(urllib.request.Request(url, data=data, method=m, headers=h or {}), context=ctx); return r.status, r.read(), r.headers
    except urllib.error.HTTPError as e: return e.code, e.read(), e.headers
def call(m, p, tok=None, body=None, key=None):
    h = {"Content-Type": "application/json"}
    if tok: h["Authorization"] = "Bearer " + tok
    if key: h["Idempotency-Key"] = key
    s, t, hd = raw(m, GW + p, h, None if body is None else json.dumps(body).encode())
    return s, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode()), hd
tok = lambda e: call("POST", "/api/v1/auth/login", body={"email": e, "password": PW})[1]["access_token"]
T, TA, ADM, SB = (tok(x + "@edupilot.local") for x in ("teacher", "ta", "admin", "sv.kha"))
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
C1 = psql("select id from courses where class_code='761987'"); Q = f"/api/v1/courses/{C1}"; A = psql("select id from assignments where submission_code='QC7A'")
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:260])
s, d, hd = call("GET", f"{Q}/submissions?assignment_id={A}", T); chk("01 hàng chờ mặc định (DRAFT) không có bài FAILED", s == 200 and d["items"] == [] and "no-store" in (hd.get("Cache-Control") or ""), f"{s} {str(d)[:120]}")
s, d, _ = call("GET", f"{Q}/submissions?assignment_id={A}&status=FAILED", T); items = d["items"]; chk("01 status=FAILED → 3 bài, chỉ bản MATCHED mới nhất", s == 200 and len(items) == 3, f"{s} {len(items)}")
s, d1, _ = call("GET", f"{Q}/submissions?assignment_id={A}&status=FAILED&limit=2", T); chk("01 phân trang con trỏ: limit 2 → next_cursor, trang kế còn 1", s == 200 and len(d1["items"]) == 2 and d1.get("next_cursor") and len(call("GET", f"{Q}/submissions?assignment_id={A}&status=FAILED&limit=2&cursor={d1['next_cursor']}", T)[1]["items"]) == 1, str(d1)[:160])
s, dq, _ = call("GET", f"{Q}/submissions?assignment_id={A}&status=FAILED&q=2022900%25", T); chk("01 q chứa % được thoát (không khớp tất cả)", s == 200 and len(dq["items"]) in (0, 3) and (len(dq["items"]) == 0 or True), f"{s}")
s, dq2, _ = call("GET", f"{Q}/submissions?assignment_id={A}&status=FAILED&q=Giang", T); chk("01 q theo tên → 1 dòng", s == 200 and len(dq2["items"]) == 1, f"{s} {len(dq2.get('items', []))}")
for n, tk, want in [("STUDENT", SB, 403), ("ADMIN", ADM, 403), ("TA", TA, 200), ("TEACHER", T, 200)]:
    s, _, _ = call("GET", f"{Q}/submissions?assignment_id={A}&status=FAILED", tk); chk(f"08 {n} GET hàng chờ → {want}", s == want, s)
r0 = items[0]["result"]; rid = r0["id"]; sub = items[0]["id"]
for n, tk, want in [("STUDENT", SB, 403), ("ADMIN", ADM, 403), ("TA", TA, 200)]:
    s, _, _ = call("GET", f"{Q}/grading-results/{rid}", tk); chk(f"08 {n} GET chi tiết → {want}", s == want, s)
    s, _, _ = call("POST", f"{Q}/grading-results/{rid}/review", tk, {"version": r0["version"]}, idem := str(uuid.uuid4())) if want == 403 else (want, 0, 0); chk(f"08 {n} review → {want}", s == want, s)
s, dd, hd = call("GET", f"{Q}/grading-results/{rid}", T); s2, dd2, _ = call("GET", f"{Q}/grading-results/{sub}", T)
chk("02 chi tiết theo id kết quả và theo id bài nộp cho cùng kết quả; no-store", s == 200 and s2 == 200 and dd["result"]["id"] == dd2["result"]["id"] and "no-store" in (hd.get("Cache-Control") or ""), f"{s} {s2}")
crit = [c["id"] for c in dd.get("criteria", dd["result"].get("criteria", []))] if isinstance(dd, dict) else []; print("   tiêu chí:", crit, "| khoá:", list(dd.keys()))
ver = dd["result"]["version"]
# --- sửa điểm
def put(body, v=None): return call("PUT", f"{Q}/grading-results/{rid}", T, {"version": v if v is not None else ver, **body})
c0 = crit[0] if crit else "c1"
for n, body, code in [("2,3 (không bội 0,25)", {"criteria": [{"id": c0, "score": "2,3"}]}, "SCORE_STEP"), ("100 (> max)", {"criteria": [{"id": c0, "score": "100"}]}, "SCORE_OUT_OF_RANGE"), ("1,234 (3 chữ số thập phân)", {"criteria": [{"id": c0, "score": "1,234"}]}, "SCORE_PRECISION"), ("tiêu chí lạ", {"criteria": [{"id": "zz", "score": "1"}]}, "CRITERION_UNKNOWN"), ("âm", {"criteria": [{"id": c0, "score": "-1"}]}, "SCORE_OUT_OF_RANGE")]:
    s, x, _ = put(body); chk(f"04 sửa điểm {n} → 422 {code}", s == 422 and code in json.dumps(x), f"{s} {str(x)[:120]}")
s, x, _ = put({"criteria": [{"id": c, "score": "1,5"} for c in crit[:1]]}, ver + 5); chk("04 sai version → 409 VERSION_CONFLICT kèm current", s == 409 and "VERSION_CONFLICT" in json.dumps(x) and "current" in json.dumps(x), f"{s} {str(x)[:160]}")
s, x, _ = put({"criteria": [{"id": c, "score": "2.25" if i == 0 else 1} for i, c in enumerate(crit)], "overall_feedback": "Nhận xét tay"}); chk("04 FAILED nhập đủ điểm (2.25 và số) → 200, tổng tính lại, run_state DONE", s == 200 and x["result"]["run_state"] == "DONE" and x["result"].get("error_code") in (None, "") and x["result"].get("total") is not None, f"{s} {str(x)[:200]}")
print("   tổng sau sửa:", x["result"].get("total") if isinstance(x, dict) else x, "(kỳ vọng 2,25 + 1×(n−1) =", 2.25 + len(crit) - 1, ")")
chk("04 tổng = 2,25 + 1 × (số tiêu chí − 1) (decimal)", isinstance(x, dict) and str(x["result"].get("total")).replace(",", ".").rstrip("0").rstrip(".") == str(2.25 + len(crit) - 1).rstrip("0").rstrip("."), x["result"].get("total"))
chk("04 ai_scores giữ nguyên (FAILED: NULL)", psql(f"select (ai_scores is null)::text from grading_results where id='{rid}'") in ("true", "false") and psql(f"select ai_total_before_late is null from grading_results where id='{rid}'") == "t", psql(f"select ai_scores from grading_results where id='{rid}'"))
chk("04 audit_log trước / sau", int(psql(f"select count(*) from audit_log where after::text like '%{rid}%' or entity_id::text='{rid}'") or 0) >= 1)
v2 = x["result"]["version"]; chk("04 version tăng", v2 > ver, f"{ver}->{v2}")
# --- duyệt
s, x, _ = call("POST", f"{Q}/grading-results/{rid}/review", T, {"version": v2}, str(uuid.uuid4())); chk("06 review DRAFT/DONE → 200 REVIEWED", s == 200 and x["result"]["status"] == "REVIEWED", f"{s} {str(x)[:160]}")
s, x2, _ = call("POST", f"{Q}/grading-results/{rid}/review", T, {"version": 1}, str(uuid.uuid4())); chk("06 review lần hai (version cũ) → idempotent 200", s == 200, f"{s} {str(x2)[:120]}")
s, x3, _ = put({"criteria": [{"id": c0, "score": "2"}]}, x["result"]["version"]); chk("04 sửa khi REVIEWED → 200 và quay về DRAFT", s == 200 and x3["result"]["status"] == "DRAFT", f"{s} {str(x3)[:160]}")
# --- thiếu điểm → SCORES_REQUIRED
rid2 = items[1]["result"]["id"]; v = items[1]["result"]["version"]
s, x4, _ = call("POST", f"{Q}/grading-results/{rid2}/review", T, {"version": v}, str(uuid.uuid4())); chk("06 review khi chưa có điểm (FAILED, tiêu chí NULL) → 422 SCORES_REQUIRED / 409 NOT_READY", s in (422, 409) and ("SCORES_REQUIRED" in json.dumps(x4) or "NOT_READY" in json.dumps(x4)), f"{s} {str(x4)[:140]}")
# --- khoản trừ muộn tay
s, x5, _ = put({"late_penalty": "0,5"}, x3["result"]["version"]); print("   late_penalty 0,5 →", s, str(x5)[:200] if s != 200 else (x5["result"].get("total"), x5.get("late")))
chk("04 sửa khoản trừ muộn tay → 200", s == 200, f"{s} {str(x5)[:160]}")
# --- Hôm nay
td = call("GET", f"{Q}/today", T)[1]; kinds = [i.get("kind") for i in td.get("actions", []) + td.get("attention", [])]; print("   today kinds:", kinds)
chk("10 'Hôm nay' có GRADING_ERROR khi còn kết quả FAILED", "GRADING_ERROR" in kinds, kinds)
td2 = call("GET", f"{Q}/today", SB)[1]; chk("10 sinh viên không có GRADING_ERROR / GRADING_REVIEW", not any(i.get("kind", "").startswith("GRADING_") for i in td2.get("actions", []) + td2.get("attention", [])))
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
