#!/usr/bin/env python3
"""QC US-S9-SEED AC2 độc lập: `Chấm tất cả` Bài tập 03 trên stack seed (provider giả) rồi so TỪNG Ô với fixtures/expected_grading.json (BA) theo seed-plan.csv. Không dùng check-assignments-seed.mjs."""
import csv, json, os, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:543"); PW = "Edupilot#Seed-2026"; ctx = ssl._create_unverified_context()
FX = os.path.join(os.path.dirname(__file__), "../../../../specs/FEAT-assignments/fixtures")
def call(m, p, tok=None, body=None):
    h = {"Content-Type": "application/json", "Idempotency-Key": str(uuid.uuid4())}
    if tok: h["Authorization"] = "Bearer " + tok
    try:
        r = urllib.request.urlopen(urllib.request.Request(GW + p, data=None if body is None else json.dumps(body).encode(), method=m, headers=h), context=ctx); t = r.read(); return r.status, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode())
    except urllib.error.HTTPError as e: return e.code, e.read().decode()[:300]
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
T = call("POST", "/api/v1/auth/login", body={"email": "teacher@edupilot.local", "password": PW})[1]["access_token"]
C1 = psql("select id from courses where class_code='761987'"); A = psql(f"select id from assignments where submission_code='BT03' and course_id='{C1}'"); Q = f"/api/v1/courses/{C1}"
print("   grading_results trước:", psql("select count(*) from grading_results"))
llm0 = int(psql("select count(*) from llm_audit"))
if psql("select count(*) from grading_results") == "0":
    s, g = call("POST", f"{Q}/assignments/{A}/grade-all", T, {}); print("   grade-all:", s, g); assert s == 202 and g["queued"] == 22, g
else: print("   đã chấm sẵn — chỉ đối chiếu")
for _ in range(120):
    if psql("select count(*) from grading_results where run_state in ('QUEUED','RUNNING')") == "0": break
    time.sleep(2)
exp = {x["id"]: x["expected"] for x in json.load(open(f"{FX}/expected_grading.json"))}
plan = {r["account"]: r for r in csv.DictReader(open(f"{FX}/seed-plan.csv"))}
fails = 0; checked = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else x)
rows = psql("select u.email, gr.run_state, coalesce(gr.error_code,''), coalesce(gr.ai_total_before_late::text,''), coalesce(gr.total::text,''), coalesce(gr.flags::text,''), coalesce(gr.ai_scores::text,''), coalesce(gr.late_penalty::text,'') from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id order by u.email")
got = {}
for r in rows.splitlines():
    f = r.split("|"); got[f[0].split("@")[0]] = f
chk("22 kết quả, 22 sinh viên MATCHED", len(got) == 22, len(got))
for acc, r in plan.items():
    if r["initial_match"] != "MATCHED": continue
    smp = r["sample_v2"] or r["sample_v1"]; e = exp[smp]; f = got.get(acc)
    if not f: chk(f"{acc} ({smp}) có kết quả", False); continue
    state, err, before, total, flags, aiscores = f[1], f[2], f[3], f[4], f[5], f[6]
    if smp == "S4":  # bài trống: 0 lời gọi, không 0 điểm tự động
        chk(f"{acc} S4 trống: {state}/{err}", state in ("DONE", "FAILED"), f); continue
    h = float(r["hours_vs_due"] or 0); late = min(-(-h // 24), 2) if h > 0 else 0; want_total = max(0.0, float(e["total_before_late"]) - 0.5 * late)  # trừ muộn 0,5 × min(⌈giờ ÷ 24⌉, 2) (README fixtures)
    ok = state == "DONE" and float(total or -1) == want_total and float(before or -1) == float(e["total_before_late"]); checked += 1
    sc = json.loads(aiscores) if aiscores.startswith("{") else {}
    ok_c = all(abs(float(sc.get(k, "nan")) - float(v)) < 1e-9 for k, v in e["criteria_scores"].items()) if sc else False
    fl = sorted(json.loads(flags.replace("{", "[").replace("}", "]").replace("NEEDS_REVIEW", '"NEEDS_REVIEW"').replace("LOW_CONFIDENCE", '"LOW_CONFIDENCE"').replace("TOTAL_MISMATCH", '"TOTAL_MISMATCH"').replace("EVIDENCE_NOT_FOUND", '"EVIDENCE_NOT_FOUND"').replace("INJECTION_SUSPECTED", '"INJECTION_SUSPECTED"').replace("ANSWER_ECHO", '"ANSWER_ECHO"').replace("SCORE_ADJUSTED", '"SCORE_ADJUSTED"').replace("OFF_TOPIC", '"OFF_TOPIC"').replace("TOO_SHORT", '"TOO_SHORT"')) if flags else []) if flags else []
    chk(f"{acc} {smp}: Σ {total} (chờ {want_total}), trước muộn {before} (chờ {e['total_before_late']}), tiêu chí AI khớp", ok and ok_c, f)
    want_fl = sorted(e["flags"]); chk(f"{acc} {smp}: cờ {fl} (chờ {want_fl})", fl == want_fl, f"{fl} {want_fl}")
chk("so ô: đã đối chiếu ≥ 20 bài", checked >= 20, checked)
llm = int(psql("select count(*) from llm_audit")) - llm0; print("   lời gọi LLM mới:", llm)
chk("sv24 (S4 trống) 0 lời gọi LLM; số lời gọi = 2 × (số bài có chữ) ± bài hỏng", llm in (42, 44), llm)
chk("mọi lời gọi chấm làn BATCH", psql("select count(*) from llm_audit where task='GRADING' and lane<>'BATCH'") == "0")
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
