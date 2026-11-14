# 10.6 pha 2 — F: Hoàn tác = decide RESET (P-106-8, TC-G04-19…30 phần API/DB). Tiền đề: stack ở HEAD có RESET; route GRADING → stub; cờ bật.
# python gd_f.py
from gdlib import *
import concurrent.futures
sql(f"delete from grading_guidance_items where assignment_id='{A}'")  # kể cả mục đã áp dụng: tránh trùng chữ làm reflect bỏ mục
stub("rf_many"); c, j, jb = reflect("rf_many", "qc-gdf-many1"); stub("allow")
print("reflect:", jb[0], jb[1][:80])
D = lambda iid: f"{G}/items/{iid}/decide"
def ov(): return api("teacher", "GET", G)[1]
def it(i): return [x for x in ov()["items"] if x["id"] == i][0]
def row(i): return sql(f"select status, decided_by is null, approved_in_version, pending_retire, text, coalesce(text_original,'') from grading_guidance_items where id='{i}'")[0]
def asg_v(): return sql(f"select guidance_version from assignments where id='{A}'")[0][0]
def nver(): return sql(f"select count(*) from grading_guidance_versions where assignment_id='{A}'")[0][0]
dr = [x["id"] for x in ov()["items"] if x["status"] == "DRAFT"]
ok("G04-F0 có ≥ 6 DRAFT", len(dr) >= 6, str(len(dr)))
v0, n0 = asg_v(), nver()
# --- G04-20 + G04-26: duyệt có sửa chữ → RESET giữ chữ đã sửa, text_original = bản AI
i1 = dr[0]; ai_text = it(i1)["text"]; v = it(i1)["version"]
c, j, _ = api("teacher", "POST", D(i1), {"decision": "APPROVE", "version": v, "text": "Ưu tiên ví dụ thực tế đã sửa tay khi chấm tiêu chí này."}); ok("G04-19 APPROVE (có sửa chữ) → 200 APPROVED", c == 200 and j["status"] == "APPROVED", str(c))
v = it(i1)["version"]
c, j, _ = api("teacher", "POST", D(i1), {"decision": "RESET", "version": v}); ok("G04-20 RESET → 200, về DRAFT", c == 200 and j["status"] == "DRAFT", f"{c} {json.dumps(j)[:140]}")
st, dnull, aiv, pr, txt, orig = row(i1)
ok("G04-20 decided_by NULL, approved_in_version NULL", dnull == "t" and aiv in ("", None), str(row(i1)))
ok("G04-26 giữ chữ đã sửa; text_original = bản AI", txt.startswith("Ưu tiên ví dụ thực tế đã sửa tay") and orig == ai_text, f"{txt[:40]} | {orig[:40]}")
ok("G04-20 phiên bản không đổi (assignments, grading_guidance_versions)", asg_v() == v0 and nver() == n0, f"{asg_v()} {nver()}")
act = sql(f"select string_agg(action, ',' order by created_at) from audit_log where entity_id='{i1}'")[0][0]; print("INFO audit:", act)
ok("G04-20 audit_log có approve + reset", "approve" in (act or "") and "reset" in (act or ""), str(act))
# --- G04-21 Bỏ → RESET
i2 = dr[1]; v = it(i2)["version"]
c, j, _ = api("teacher", "POST", D(i2), {"decision": "REJECT", "version": v}); ok("G04-21 REJECT → 200", c == 200 and j["status"] == "REJECTED", str(c))
c, j, _ = api("teacher", "POST", D(i2), {"decision": "RESET", "version": it(i2)["version"]}); ok("G04-21 RESET từ REJECTED → DRAFT", c == 200 and j["status"] == "DRAFT" and row(i2)[1] == "t", f"{c}")
# --- RESET trên DRAFT: 200 không đổi
v = it(i2)["version"]
c, j, _ = api("teacher", "POST", D(i2), {"decision": "RESET", "version": v}); ok("G04-23 RESET trên DRAFT → 200 không đổi", c == 200 and j["status"] == "DRAFT", f"{c} {json.dumps(j)[:100]}")
# --- G04-22 (API): duyệt, đợi, RESET vẫn được (menu Đặt lại) 
i3 = dr[2]; v = it(i3)["version"]; api("teacher", "POST", D(i3), {"decision": "APPROVE", "version": v}); time.sleep(6)
c, j, _ = api("teacher", "POST", D(i3), {"decision": "RESET", "version": it(i3)["version"]}); ok("G04-22 sau 6 s RESET vẫn 200 (Đặt lại cho tới khi Áp dụng)", c == 200 and j["status"] == "DRAFT", str(c))
# --- G04-25 hai tab
i4 = dr[3]; v = it(i4)["version"]
c, a, _ = api("teacher", "POST", D(i4), {"decision": "APPROVE", "version": v}); vA = a["version"]
c, b, _ = api("teacher", "POST", D(i4), {"decision": "RESET", "version": vA})           # tab B đặt lại
c, b, _ = api("teacher", "POST", D(i4), {"decision": "REJECT", "version": b["version"]})  # tab B bỏ
c, j, _ = api("teacher", "POST", D(i4), {"decision": "RESET", "version": vA})           # tab A Hoàn tác với version cũ
ok("G04-25 RESET version cũ → 409 VERSION_CONFLICT, không ghi đè quyết định của tab B", c == 409 and (j or {}).get("code") == "VERSION_CONFLICT" and row(i4)[0] == "REJECTED", f"{c} {(j or {}).get('code')} {row(i4)[0]}")
# --- RBAC RESET
i5 = dr[4]; v = it(i5)["version"]; api("teacher", "POST", D(i5), {"decision": "APPROVE", "version": v}); v = it(i5)["version"]
for u, tag in (("ta", "TA"), ("sv.kha", "SV"), ("admin", "ADMIN")):
    c, _, _ = api(u, "POST", D(i5), {"decision": "RESET", "version": v}); ok(f"G04-RBAC {tag} RESET → 403", c == 403, str(c))
c, j, _ = api("teacher", "POST", D(i5), {"decision": "KHAC", "version": v}); ok("G04-RESET decision lạ → 422", c == 422, str(c))
# --- G04-30 đua RESET vs Áp dụng (i5 đang «Chờ áp dụng»)
cur = int(asg_v())
def do_reset(): return api("teacher", "POST", D(i5), {"decision": "RESET", "version": v})[0:2]
def do_apply(): return api("teacher", "POST", G + "/apply", {"expected_version": cur}, idem=f"qc-race-{int(time.time())}")[0:2]
with concurrent.futures.ThreadPoolExecutor(2) as ex:
    f1 = ex.submit(do_reset); f2 = ex.submit(do_apply); r1, r2 = f1.result(), f2.result()
print("INFO đua RESET/apply:", r1[0], (r1[1] or {}).get("code"), "|", r2[0], (r2[1] or {}).get("code"))
stt = row(i5); broken = sql(f"select count(*) from grading_guidance_items where assignment_id='{A}' and status='APPROVED' and approved_in_version is not null and decided_by is null")[0][0]
ok("G04-30 đua RESET × Áp dụng: không mục nửa vời; kết quả nhất quán", broken == "0" and ((r1[0] == 200 and stt[0] == "DRAFT") or (r1[0] == 409 and stt[0] == "APPROVED" and stt[2] not in ("", None))) , f"reset={r1[0]} apply={r2[0]} trạng thái={stt[:4]}")
# --- G04-23 sau Áp dụng: RESET → 409 ITEM_NOT_RESETTABLE
applied = sql(f"select id from grading_guidance_items where assignment_id='{A}' and status='APPROVED' and approved_in_version is not null and not pending_retire order by approved_in_version desc limit 1")
if not applied or stt[0] == "DRAFT":
    # dựng một mục đã áp dụng
    iX = dr[5]; vX = it(iX)["version"]; api("teacher", "POST", D(iX), {"decision": "APPROVE", "version": vX})
    api("teacher", "POST", G + "/apply", {"expected_version": int(asg_v())}, idem=f"qc-apply-x-{int(time.time())}")
    applied = [[iX]]
ia = applied[0][0]
c, j, _ = api("teacher", "POST", D(ia), {"decision": "RESET", "version": it(ia)["version"]}); ok("G04-23 RESET mục đã áp dụng → 409 ITEM_NOT_RESETTABLE (không phải ITEM_NOT_DRAFT)", c == 409 and (j or {}).get("code") == "ITEM_NOT_RESETTABLE", f"{c} {(j or {}).get('code')}")
retired = sql(f"select id from grading_guidance_items where assignment_id='{A}' and status='RETIRED' limit 1")
if retired:
    c, j, _ = api("teacher", "POST", D(retired[0][0]), {"decision": "RESET", "version": it(retired[0][0])["version"]}); ok("G04-29 RESET mục RETIRED → 409 ITEM_NOT_RESETTABLE", c == 409 and (j or {}).get("code") == "ITEM_NOT_RESETTABLE", f"{c} {(j or {}).get('code')}")
# --- G04-29 Cho nghỉ → RESET
c, j, _ = api("teacher", "POST", D(ia), {"decision": "RETIRE", "version": it(ia)["version"]}); ok("G04-29 RETIRE → chờ nghỉ", c == 200 and j["pending_retire"] is True, str(c))
c, j, _ = api("teacher", "POST", D(ia), {"decision": "RESET", "version": it(ia)["version"]}); r = row(ia)
ok("G04-29 RESET chờ nghỉ → pending_retire=false, vẫn APPROVED, approved_in_version giữ", c == 200 and j["pending_retire"] is False and r[0] == "APPROVED" and r[2] not in ("", None), f"{c} {r[:4]}")
