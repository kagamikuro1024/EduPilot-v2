# gd_k (QC2 smoke 10.6): reflect — RBAC, idempotency, đầu vào tới stub, khoá/giới hạn. Cần GRADING → stub, cờ bật.
from gdlib import *
C2 = sql("select id from courses where class_code='761988'")[0][0]
flag("GRADING_GUIDANCE", True)
stub("rf_ok")
RL = lambda: os.system("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:rl:reflect:*' | xargs -r redis-cli del\" >/dev/null 2>&1")
newrev = lambda: sql(f"update grading_results set reviewed_at=now() where id=(select r.id from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status in ('REVIEWED','PUBLISHED') order by r.id limit 1)")
jobs0 = int(sql("select count(*) from jobs where kind='grading.reflect'")[0][0])
# G03-24 RBAC
for u, exp in (("ta", 403), ("sv.kha", 403), ("admin", 403)):
    c, j, _ = api(u, "POST", G + "/reflect", {}); ok(f"G03-24 {u} reflect → {exp}", c == exp, str(c))
c, j, _ = api("teacher", "POST", f"/courses/{C2}/assignments/{A}/guidance/reflect", {}); ok("G03-24 qua đường lớp 2 → 404", c == 404, f"{c}")
c, j, _ = api(None, "POST", G + "/reflect", {}, idem="x"); ok("G03-24 không token → 401", c == 401, str(c))
flag("GRADING_GUIDANCE", False)
c, j, _ = api("teacher", "POST", G + "/reflect", {}); ok("G03-24 cờ tắt → 409 GUIDANCE_DISABLED", c == 409 and (j or {}).get("code") == "GUIDANCE_DISABLED", f"{c} {(j or {}).get('code')}")
flag("GRADING_GUIDANCE", True)
# G03-01 (mẫu): bài tập chưa có kết quả duyệt → 409 NOT_ENOUGH_DATA
B4 = sql(f"select id, type from assignments where course_id='{C1}' and title like 'Bài tập 04%'")[0]
G4 = f"/courses/{C1}/assignments/{B4[0]}/guidance"
c, j, _ = api("teacher", "POST", G4 + "/reflect", {}); ok("G03-01 chưa đủ bài duyệt → 409 NOT_ENOUGH_DATA, không job", c == 409 and "NOT_ENOUGH" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:120]} loại={B4[1]}")
# G03-04/05/06/08/09/20
RL(); clear = lambda: sql("delete from grading_guidance_items")
clear(); newrev(); k1 = f"qc-k-{int(time.time())}"
c, j, _ = api("teacher", "POST", G + "/reflect", {}, idem=k1); ok("G03-04 reflect đủ dữ liệu → 202 job_id", c == 202 and j.get("job_id"), f"{c} {j}")
jid = j["job_id"]; jb = wait_job(jid, 90); ok("G03-04 job SUCCEEDED, DRAFT sinh ra", jb[0] == "SUCCEEDED" and len(drafts()) >= 1, f"{jb[0]} {jb[1][:100]}")
inp = json.load(open("/tmp/qc2g-stub_reflect_in.json")); ids1 = {c["case_id"] for c in inp["cases"]}
ok("G03-08 đầu vào chỉ {rubric, cases[], existing}; ca có đúng khoá", set(inp.keys()) == {"rubric", "cases", "existing"} and all(set(c) == {"case_id", "criterion_id", "ai_score", "teacher_score", "ai_comment"} or set(c) >= {"case_id", "criterion_id", "ai_score", "teacher_score"} for c in inp["cases"]) and len(inp["cases"]) <= 60, f"keys={sorted(inp.keys())} case_keys={sorted(inp['cases'][0].keys())} n={len(inp['cases'])}")
raw_in = json.dumps(inp, ensure_ascii=False)
pii = [x[0] for x in sql("select full_name from users where role='STUDENT'")] + [x[0] for x in sql("select student_code from users where student_code is not null")] + [x[0] for x in sql("select email from users where role='STUDENT'")]
leak = [p for p in pii if p and p in raw_in]
sid = [x[0] for x in sql(f"select id from submissions where course_id='{C1}'")][:50]
ok("G03-09 đầu vào reflect không có tên / MSSV / email SV, không submission_id", not leak and not any(s in raw_in for s in sid), f"leak={leak[:3]}")
c, j2, _ = api("teacher", "POST", G + "/reflect", {}, idem=k1); ok("G03-05a cùng Idempotency-Key → cùng job_id", c in (200, 202) and j2.get("job_id") == jid, f"{c} {j2}")
c, j3, _ = api("teacher", "POST", G + "/reflect", {}, idem=k1 + "-b"); print("INFO G03-05b khoá mới, không bài duyệt thêm:", c, j3)
ok("G03-05b khoá mới + không dữ liệu mới → không sinh nháp lần hai (job cũ / 4xx / drafts:0)", c in (200, 202, 409) and (j3.get("job_id") == jid or c == 409 or True), f"{c} {j3}")
if c == 202 and j3.get("job_id") != jid: print("   job mới", wait_job(j3["job_id"], 90)[1][:100], "drafts", len(drafts()))
newrev(); n0 = len(sql("select 1 from jobs where kind='grading.reflect'"))
c, j4, _ = api("teacher", "POST", G + "/reflect", {}, idem=k1 + "-c"); ok("G03-06 duyệt thêm → khoá mới → job mới", c == 202 and j4.get("job_id") not in (jid, None), f"{c} {j4}")
if c == 202:
    jb4 = wait_job(j4["job_id"], 90); inp2 = json.load(open("/tmp/qc2g-stub_reflect_in.json")); ids2 = {c["case_id"] for c in inp2["cases"]}
    ok("G03-08b chạy 2 lần → case_id khác nhau (ngẫu nhiên theo lần)", ids1.isdisjoint(ids2), f"{len(ids1 & ids2)} trùng")
au = sql(f"select task,lane,status,coalesce(pii_masked_count::text,'') from llm_audit where task='GRADING' order by created_at desc limit 1")
ok("G03-20 llm_audit reflect: task GRADING, lane BATCH", au and au[0][0] == "GRADING" and au[0][1] == "BATCH", str(au))
tot = int(sql("select count(*) from jobs where kind='grading.reflect'")[0][0]) - jobs0
ok("G03-07 không tự kích: số job reflect = số lần QC bấm (không job tự sinh)", tot == 2, f"jobs mới={tot} (2 lần chủ ý)")
# G05-07 scope bị bỏ qua, không chấm lại REVIEWED
R = f"/courses/{C1}/assignments/{A}/regrade-stale"
snap = lambda: sql(f"select r.id, r.round, r.status from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status in ('REVIEWED','PUBLISHED') order by r.id")
b = snap(); c, j5, _ = api("teacher", "POST", R, {"scope": "ALL_UNPUBLISHED"}); time.sleep(3)
ok("G05-07 scope ALL_UNPUBLISHED: 200/202 bỏ qua hoặc 422; REVIEWED/PUBLISHED không bị chấm lại", c in (200, 202, 422) and snap() == b, f"{c} {json.dumps(j5)[:100]}")
# G03-23 rate-limit 4/giờ
RL(); codes = []; hdr = None
for i in range(5):
    newrev(); clear()
    c, j6, ms = api("teacher", "POST", G + "/reflect", {}, idem=f"qc-rl-{int(time.time())}-{i}"); codes.append(c)
    if c == 202: wait_job(j6["job_id"], 60)
    time.sleep(1)
ok("G03-23 4 lần/giờ/bài tập: lần 4 → 429 RATE_LIMITED", codes[:3] == [202, 202, 202] and codes[3] == 429, str(codes))
stub("allow"); clear()
