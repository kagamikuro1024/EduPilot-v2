# 10.6 pha 2 — C: duyệt / Áp dụng / Cho nghỉ (US-GD-04). Tiền đề: gd_a.py (8 bài duyệt có sửa), route GRADING → stub, cờ GRADING_GUIDANCE bật.
# python gd_c.py
from gdlib import *
import concurrent.futures
sql("delete from grading_guidance_items"); sql("delete from grading_guidance_versions where assignment_id='" + A + "'")
sql(f"update assignments set guidance_version=0 where id='{A}'")
c, j, jb = reflect("rf_many", "qc-gdc-many1"); print("reflect rf_many:", jb[0], jb[1][:120])
stub("allow")
items = drafts()
ok("G04-00 có ≥ 6 DRAFT để thao tác", len(items) >= 6, str(len(items)))
ids = [x[0] for x in items]
def get_items():
    c, j, _ = api("teacher", "GET", G); return j
def item(iid):
    return [x for x in get_items()["items"] if x["id"] == iid][0]
D = lambda iid: f"{G}/items/{iid}/decide"
# --- RBAC (G04-11)
v = item(ids[0])["version"]
for u, tag, exp in (("ta", "TA", 403), ("sv.kha", "SV", 403), ("admin", "ADMIN", 403)):
    c, _, _ = api(u, "POST", D(ids[0]), {"decision": "APPROVE", "version": v}); ok(f"G04-11 {tag} decide → {exp}", c == exp, str(c))
    c, _, _ = api(u, "POST", G + "/apply", {"expected_version": 0}, idem="qc-rbac-" + tag.lower() + "00"); ok(f"G04-11 {tag} apply → {exp}", c == exp, str(c))
c, _, _ = api("ta", "GET", G); ok("G04-11 TA xem danh sách → 200", c == 200, str(c))
c, _, _ = api("sv.kha", "GET", G); ok("G04-11 SV GET → 403", c == 403, str(c))
c, _, _ = api("admin", "GET", G); print("INFO ADMIN GET tổng quan", c)
C2 = sql("select id from courses where class_code='761988'")[0][0]
c, _, _ = api("teacher", "GET", f"/courses/{C2}/assignments/{A}/guidance"); ok("G04-11 bài tập của lớp 1 qua đường lớp 2 → 404", c == 404, str(c))
# --- G04-03 duyệt
c, j, _ = api("teacher", "POST", D(ids[0]), {"decision": "APPROVE", "version": v}); ok("G04-03 APPROVE → 200 APPROVED chờ áp dụng", c == 200 and j["status"] == "APPROVED" and j.get("approved_in_version") is None, f"{c} {json.dumps(j)[:140]}")
au = sql("select action, \"after\"::text from audit_log where entity_id='%s' order by created_at desc limit 1" % ids[0])
print("INFO audit:", au)
c, j, _ = api("teacher", "POST", D(ids[0]), {"decision": "APPROVE", "version": v}); ok("G04-05 version cũ → 409", c == 409, f"{c} {(j or {}).get('code')}")
nv = item(ids[0])["version"]
c, j, _ = api("teacher", "POST", D(ids[0]), {"decision": "APPROVE", "version": nv}); ok("G04-05 APPROVE lại → 409 ITEM_NOT_DRAFT", c == 409 and (j or {}).get("code") == "ITEM_NOT_DRAFT", f"{c} {(j or {}).get('code')}")
# --- G04-04 Bỏ
v1 = item(ids[1])["version"]
c, j, _ = api("teacher", "POST", D(ids[1]), {"decision": "REJECT", "version": v1}); ok("G04-04 REJECT → 200 REJECTED", c == 200 and j["status"] == "REJECTED", f"{c}")
# --- G04-06 sửa chữ khi duyệt
v2 = item(ids[2])["version"]; orig = item(ids[2])["text"]
c, j, _ = api("teacher", "POST", D(ids[2]), {"decision": "APPROVE", "version": v2, "text": "Ưu tiên bằng chứng cụ thể khi chấm tiêu chí này."}); ok("G04-06a sửa chữ hợp lệ → 200, text_original = bản gốc", c == 200 and j["text"].startswith("Ưu tiên") and j.get("text_original") == orig, f"{c} {json.dumps(j)[:160]}")
for tag, body in (("b từ khoá «tổng»", "Cần tính tổng điểm chặt hơn."), ("c PII (tên)", "Bài của Bùi Thanh Khải thường thiếu ví dụ."), ("c2 PII (MSSV)", "Sinh viên 20229002 hay viết ngắn."), ("d rỗng", ""), ("d dài > 400", "a" * 401)):
    vv = item(ids[3])["version"]
    c, j, _ = api("teacher", "POST", D(ids[3]), {"decision": "APPROVE", "version": vv, "text": body}); ok(f"G04-06{tag} → 422", c == 422, f"{c} {(j or {}).get('code')}")
c, j, _ = api("teacher", "POST", D(ids[3]), {"decision": "APPROVE", "version": item(ids[3])["version"], "text": "x"}); print("INFO text 1 ký tự:", c)
print("INFO mục 3 vẫn DRAFT:", item(ids[3])["status"])
# --- G04-07 apply
c, j, _ = api("teacher", "POST", G + "/apply", {"expected_version": 0}, idem="qc-apply-0001")
ok("G04-07 apply expected 0 → 200 {version:1, stale}", c == 200 and j["version"] == 1 and "stale" in j, f"{c} {json.dumps(j)[:200]}")
r = sql(f"select guidance_version from assignments where id='{A}'")[0][0]; ok("G04-07 assignments.guidance_version = 1", r == "1", r)
n_app = sql(f"select count(*) from grading_guidance_items where assignment_id='{A}' and approved_in_version=1")[0][0]; ok("G04-07 2 mục approved_in_version=1", n_app == "2", n_app)
vr = sql(f"select item_count from grading_guidance_versions where assignment_id='{A}' and version=1"); ok("G04-07 hàng grading_guidance_versions v1 (item_count=2)", vr == [["2"]], str(vr))
q = sql(f"select count(*) from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.run_state in ('QUEUED','RUNNING')")[0][0]; ok("G04-07 không tự chấm lại (0 việc chấm)", q == "0", q)
c, j2, _ = api("teacher", "POST", G + "/apply", {"expected_version": 0}, idem="qc-apply-0001"); ok("G04-10 cùng Idempotency-Key → trả lại kết quả cũ", c == 200 and j2 == j, f"{c} {json.dumps(j2)[:100]}")
c, j2, _ = api("teacher", "POST", G + "/apply", {"expected_version": 0}, idem="qc-apply-0002"); ok("G04-08 expected 0 lại → 409 VERSION_CONFLICT", c == 409 and (j2 or {}).get("code") == "VERSION_CONFLICT", f"{c} {(j2 or {}).get('code')}")
c, j2, _ = api("teacher", "POST", G + "/apply", {"expected_version": 1}, idem="qc-apply-0003"); ok("G04-08 không còn gì → 409 NOTHING_TO_APPLY", c == 409 and (j2 or {}).get("code") == "NOTHING_TO_APPLY", f"{c} {(j2 or {}).get('code')}")
c, j2, _ = api("teacher", "POST", G + "/apply", {}, idem="qc-apply-0004"); ok("G04-08 thiếu expected_version → 422", c == 422, str(c))
print("INFO stale sau apply:", json.dumps(get_items()["stale"]), "counts", get_items()["counts"])
# --- G04-09 hai apply đồng thời
vv = item(ids[4])["version"]; api("teacher", "POST", D(ids[4]), {"decision": "APPROVE", "version": vv})
def ap(k): return api("teacher", "POST", G + "/apply", {"expected_version": 1}, idem=k)[0]
with concurrent.futures.ThreadPoolExecutor(2) as ex: rs = sorted(ex.map(ap, ["qc-par-00001", "qc-par-00002"]))
ok("G04-09 hai apply đồng thời: một 200, một 409", rs == [200, 409], str(rs))
vs = sql(f"select version from grading_guidance_versions where assignment_id='{A}' order by version"); ok("G04-09 phiên bản không trùng (v1, v2)", vs == [["1"], ["2"]], str(vs))
# --- G04-13 cho nghỉ
first = ids[0]; vf = item(first)["version"]
c, j, _ = api("teacher", "POST", D(first), {"decision": "RETIRE", "version": vf}); ok("G04-13 RETIRE mục đã áp dụng → 200 pending_retire", c == 200 and j.get("pending_retire") is True and j["status"] == "APPROVED", f"{c} {json.dumps(j)[:140]}")
gv = sql(f"select count(*) from grading_guidance_items where assignment_id='{A}' and approved_in_version<=2 and retired_in_version is null and status in ('APPROVED','RETIRED')")[0][0]; print("INFO còn hiệu lực ở v2 trước apply:", gv)
c, j, _ = api("teacher", "POST", G + "/apply", {"expected_version": 2}, idem="qc-apply-0005"); ok("G04-13 apply → v3, mục RETIRED retired_in_version=3", c == 200 and j["version"] == 3 and sql(f"select status, retired_in_version from grading_guidance_items where id='{first}'") == [["RETIRED", "3"]], f"{c} {json.dumps(j)[:100]}")
c, j, _ = api("teacher", "POST", D(ids[2]), {"decision": "RETIRE", "text": "x", "version": item(ids[2])["version"]}); ok("G04-06e RETIRE kèm text → 422", c == 422, str(c))
print("RESULT lưu ý hiện tại:", [(x["criterion_id"], x["status"], x["approved_in_version"], x["retired_in_version"]) for x in get_items()["items"]])
