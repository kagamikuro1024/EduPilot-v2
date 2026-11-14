# 10.6 pha 2 — D: số bài bản cũ, số đo, chấm lại (ghim + khối lưu ý trong prompt), công bố miễn. Tiền đề: gd_c.py xong (assignment guidance_version = 3).
# python gd_d.py   (route GRADING → stub: chấm giả hợp lệ, ghi system prompt vào /tmp/stub_grading_sys.jsonl)
from gdlib import *
import concurrent.futures, hashlib
open("/tmp/stub_grading_sys.jsonl", "w").close(); stub("allow")
GV = sql(f"select guidance_version from assignments where id='{A}'")[0][0]; print("assignment guidance_version:", GV)
c, ov, _ = api("teacher", "GET", G); st = ov["stale"]
rows = sql(f"select r.status, r.guidance_version, count(*) from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.round=(select max(round) from grading_results x where x.submission_id=r.submission_id) group by 1,2 order by 1,2"); print("hiện hành theo (status, gv):", rows)
ok("G05-05 stale.count = số nháp chưa sửa bản cũ (8); reviewed = 8; editing = 0", st["count"] == 8 and st["reviewed"] == 8 and st["editing"] == 0, json.dumps(st))
# --- số đo trước chấm lại (G06-01): đối chiếu SQL
c, sj, _ = api("teacher", "GET", G + "/stats"); print("stats:", json.dumps(sj)[:300])
rs = sql(f"select r.criteria::text, r.ai_scores::text, r.ai_total_before_late, r.total, r.late_penalty from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status in ('REVIEWED','PUBLISHED') and r.round=(select max(round) from grading_results x where x.submission_id=r.submission_id)")
from decimal import Decimal as Dn
sc = sa = sb = Dn(0); nc = ned = 0
for crit, ai, aitot, tot, lp in rs:
    cr = json.loads(crit); ais = json.loads(ai); fin_total = Dn(0)
    for x in cr:
        if x.get("score") is None or x["id"] not in ais: continue
        a, f = Dn(ais[x["id"]]), Dn(x["score"]); sc += abs(a - f); nc += 1; fin_total += f; ned += (a != f)
    d = Dn(aitot) - fin_total; sa += abs(d); sb += d
n = len(rs)
exp = (n, f"{sc / nc:.2f}", f"{sa / n:.2f}", f"{sb / n:.2f}", ned, f"{Dn(ned) / nc:.2f}")
it = [x for x in sj["items"] if x["version"] == 0][0]
got = (it["n"], it["mae_criterion"], it["mae_total"], it["bias_total"], it["edited_criteria"], it["edited_rate"])
ok("G06-01 số đo v0 khớp SQL độc lập (n, MAE tiêu chí, MAE tổng, lệch, tiêu chí sửa, tỉ lệ)", exp == got, f"api={got} sql={exp}")
# tổng chốt theo cột total + late_penalty để so
alt = sum(abs(Dn(aitot) - (Dn(tot) + Dn(lp))) for _, _, aitot, tot, lp in rs) / n
print(f"INFO mae_total theo total+late_penalty = {alt:.2f} (api {it['mae_total']}) — khác nhau chỉ khi có trừ muộn / chỉnh tổng tay")
# --- RBAC chấm lại (G05-24)
R = f"/courses/{C1}/assignments/{A}/regrade-stale"
c, _, _ = api("sv.kha", "POST", R, {}, idem="qc-rg-sv00001"); ok("G05-24 SV regrade → 403", c == 403, str(c))
C2 = sql("select id from courses where class_code='761988'")[0][0]
c, _, _ = api("teacher", "POST", f"/courses/{C2}/assignments/{A}/regrade-stale", {}, idem="qc-rg-c200001"); ok("G05-24 qua lớp 2 → 404", c == 404, str(c))
snap = lambda: hashlib.md5(json.dumps(sql(f"select r.id, r.status, r.total, r.criteria::text, r.round, r.guidance_version, r.ai_scores::text from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status in ('REVIEWED','PUBLISHED') order by r.id")).encode()).hexdigest()
before = snap(); old_rows = sql(f"select r.id, r.round, r.ai_scores::text, r.ai_comments::text, r.guidance_version from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status='DRAFT' order by r.id")
c, j, _ = api("ta", "POST", R, {}, idem="qc-rg-ta000001")
ok("G05-06/24 TA regrade → 202 {job_id, queued:8, skipped:{reviewed:8}}", c == 202 and j["queued"] == 8 and j["skipped"]["reviewed"] == 8 and j["skipped"]["edited"] == 0, f"{c} {json.dumps(j)[:200]}")
for _ in range(120):
    q = sql(f"select count(*) from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.run_state in ('QUEUED','RUNNING')")[0][0]
    if q == "0": break
    time.sleep(1)
new = sql(f"select r.id, r.round, r.run_state, r.guidance_version, r.ai_comments is not null, r.status from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.round > 11 order by r.id")
print("round mới:", [(x[1], x[2], x[3], x[4], x[5]) for x in new][:3], len(new))
ok("G05-06 8 kết quả round+1, DONE, ghim guidance_version = bản hiện hành, có ai_comments", len(new) == 8 and all(x[2] == "DONE" and x[3] == GV and x[4] == "t" for x in new), str(len(new)))
ok("G05-08/13 bài REVIEWED / PUBLISHED không đổi (điểm, tiêu chí, round, ghim, ai_scores)", snap() == before, "")
same = all(sql(f"select ai_scores::text, ai_comments::text, guidance_version from grading_results where id='{o[0]}'")[0] == [o[2], o[3], o[4]] for o in old_rows)
ok("G05-21 round cũ giữ nguyên ai_scores / ai_comments / guidance_version", same, "")
c, j2, _ = api("teacher", "POST", R, {}, idem="qc-rg-tea00001"); ok("G05-06 chạy lại: 200 queued 0 (không còn bài bản cũ chưa sửa)", c == 200 and j2["queued"] == 0, f"{c} {json.dumps(j2)[:120]}")
dup = sql(f"select count(*) from (select submission_id, round from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' group by 1,2 having count(*)>1) t")[0][0]; ok("G05-20 không round trùng (UNIQUE)", dup == "0", dup)
# --- khối lưu ý trong prompt (G05-01/02/04): lấy từ stub
sys_rows = [json.loads(l)["system"] for l in open("/tmp/stub_grading_sys.jsonl")]
print("lời gọi chấm tới stub:", len(sys_rows))
blk = [s for s in sys_rows if "Lưu ý của giảng viên (bản v" in s]
ok(f"G05-01 mọi lời gọi chấm lại có khối «Lưu ý của giảng viên (bản v{GV})»", len(sys_rows) > 0 and len(blk) == len(sys_rows) and all(f"(bản v{GV})" in s for s in blk), f"{len(blk)}/{len(sys_rows)}")
import re
def block(s):
    m = re.search(r"Lưu ý của giảng viên \(bản v\d+\)[^\n]*\n((?:\[[^\n]*\n?)+)", s); return m.group(0) if m else ""
ok("G05-04 khối lưu ý giống hệt giữa các bài cùng bản (lượt 1)", len({block(s) for s in blk if block(s)}) <= 2, str(len({block(s) for s in blk})))  # lượt 1 và lượt 2 (đảo thứ tự)
if blk:
    s0 = blk[0]; print("--- khối lưu ý mẫu ---\n" + block(s0))
    ok("G05-01 khối nằm sau RUBRIC, trước ĐÁP ÁN CHUẨN, trong system", s0.index("RUBRIC (thứ tự chấm)") < s0.index("Lưu ý của giảng viên") < (s0.index("ĐÁP ÁN CHUẨN") if "ĐÁP ÁN CHUẨN" in s0 else 10**9), "")
    ok("G05-01 khối chỉ là chữ lưu ý đang hiệu lực v3 (không mục RETIRED/DRAFT/REJECTED)", all(t not in s0 for t in [x[0] for x in sql(f"select text from grading_guidance_items where assignment_id='{A}' and status in ('REJECTED','DRAFT','RETIRED')")]), "")
    ok("G05-27 khối không chứa số điểm tổng / cụm trừ / làm tròn", not re.search(r"(tổng điểm|trừ|làm tròn|phạt)", block(s0).lower()), "")
# --- công bố (G05-13/14/18): ghim v0 đã duyệt, bản hiện hành 3
api("admin", "GET", "/health") if False else None
res = sql(f"select r.id from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status='REVIEWED' and r.round=(select max(round) from grading_results x where x.submission_id=r.submission_id) order by r.id")
rid = [x[0] for x in res]; print("REVIEWED hiện hành:", len(rid))
PUB = f"/courses/{C1}/grading-results/publish"
# cờ tắt trước: công bố 2 bài → không exempt đếm
api("teacher", "GET", FL); fl = flag("GRADING_GUIDANCE", False)[0]
c, j, _ = api("teacher", "POST", PUB, {"result_ids": rid[:2]}, idem="qc-pub-off0001"); print("PUBLISH cờ tắt:", c, json.dumps(j)[:200])
ok("G05-18 cờ tắt: 200, không 409, exempt rỗng/0", c == 200 and sum((j.get("exempt") or {}).values()) == 0, json.dumps(j.get("exempt")))
flag("GRADING_GUIDANCE", True)
c, j, _ = api("teacher", "POST", PUB, {"result_ids": rid[2:]}, idem="qc-pub-on00001"); print("PUBLISH cờ bật:", c, json.dumps(j)[:260])
ex = (j or {}).get("exempt", {})
exp_e = sum(1 for r in rid[2:] if len(sql(f"select 1 from grading_results where id='{r}' and criteria::text <> '' and exists (select 1 from jsonb_each_text(ai_scores) a join jsonb_array_elements(criteria) c on c->>'id'=a.key where (c->>'score')::numeric <> a.value::numeric)")) > 0)
ok("G05-14 cờ bật: bài đã duyệt ghim v0 vẫn công bố, KHÔNG 409 MIXED_GUIDANCE; exempt đếm đúng EDITED / ACCEPTED_AS_IS", c == 200 and ex.get("EDITED") == exp_e and ex.get("ACCEPTED_AS_IS") == len(rid[2:]) - exp_e, f"api={ex} kỳ vọng EDITED={exp_e} AS_IS={len(rid[2:]) - exp_e}")
au = sql("select \"after\"::text from audit_log where action='grading_result.publish' order by created_at desc limit 1")[0][0]; print("INFO audit publish:", au[:200])
ok("G05-14 audit_log có guidance_exempt", "guidance_exempt" in sql("select \"after\"::text from audit_log where \"after\"::text like '%guidance_exempt%' order by created_at desc limit 1")[0][0] if sql("select count(*) from audit_log where \"after\"::text like '%guidance_exempt%'")[0][0] != "0" else False, "")
c, j, _ = api("teacher", "POST", PUB, {"result_ids": [r for r in [sql(f"select r.id from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status='DRAFT' and r.round=(select max(round) from grading_results x where x.submission_id=r.submission_id) limit 1")[0][0]]]}, idem="qc-pub-draft001"); ok("G05-15 công bố bài nháp → 422 NOT_REVIEWED", c == 422 and "NOT_REVIEWED" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
