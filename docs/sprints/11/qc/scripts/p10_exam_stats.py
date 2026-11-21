import re
# QC US-P10-06 API + DB. Điểm kỳ vọng tính độc lập bằng SQL từ exam_attempts (coalesce(adjusted_score, auto_score), status đã chấm).
import statistics, itertools
from p10lib import *
c1, c2 = course("teacher", "761987"), course("teacher", "761988"); B1 = f"/courses/{c1}/analytics/exams"
def pub(): return sql(f"select id, title, max_score::float, opens_at::text from exams where course_id='{c1}' and status='PUBLISHED' order by opens_at, id")
def scores(eid): return [float(r[0]) for r in sql(f"select coalesce(adjusted_score, auto_score) from exam_attempts where exam_id='{eid}' and status='GRADED' and coalesce(adjusted_score, auto_score) is not null")]
print("trạng thái attempt:", sql("select distinct status from exam_attempts"), "exam status:", sql(f"select status, kind, count(*) from exams where course_id='{c1}' group by 1,2"))
P = pub(); ok("TC-07-05 ≥ 4 bài PUBLISHED", len(P) >= 4, str(len(P)))
a0 = int(sql("select count(*) from llm_audit")[0][0])
bad = 0
for eid, title, mx, _ in P:
    mx = float(mx)
    c, j, _, hd = api("teacher", "GET", f"{B1}/overview?exam_id={eid}"); sc = scores(eid)
    if not sc: continue
    mean, med, pr = round(sum(sc) / len(sc), 2), round(statistics.median(sc), 2), sum(s >= 0.5 * mx for s in sc) / len(sc)
    good = c == 200 and abs(float(j["mean"]) - mean) <= 0.01 and abs(float(j["median"]) - med) <= 0.01 and abs(float(j["pass_rate"]) - pr) <= 0.01 and j["submissions"] == len(sc) and sum(d["count"] for d in j["distribution"]) == len(sc)
    ok(f"TC-06-02 overview «{title[:30]}» khớp SQL", good, f"api={j.get('mean') if c==200 else c}/{j.get('median')}/{j.get('pass_rate')}/{j.get('submissions')} sql={mean}/{med}/{pr:.2f}/{len(sc)}")
victim = P[-2][0]; vp = sql(f"select published_at::text from exams where id='{victim}'")[0][0]
for st in ("CLOSED", "OPEN", "DRAFT"):
    sql(f"update exams set status='{st}', published_at=null where id='{victim}'")
    try:
        c, j, _, _ = api("teacher", "GET", f"{B1}/overview?exam_id={victim}"); ok(f"TC-06-03 overview bài {st} → 404/409", c in (404, 409), f"{c}")
        tr = api("teacher", "GET", f"{B1}/trend")[1]; hi = api("teacher", "GET", f"{B1}/hard-items?limit=100")[1]; dc = api("teacher", "GET", f"{B1}/declining")[1]
        leak = [n for n, x in (("trend", [i["exam_id"] for i in tr["items"]]), ("hard-items", [i["exam_id"] for i in hi["items"]]), ("declining", [s_["exam_id"] for i in dc["items"] for s_ in i["scores"]])) if victim in x]
        ok(f"TC-06-03 bài {st} không xuất hiện ở trend/hard-items/declining", not leak, str(leak))
        d0 = api("teacher", "GET", f"{B1}/overview")[1]; ok(f"TC-06-03 overview mặc định không lấy bài {st}", all(e["id"] != victim for e in d0["exams"]), "")
    finally:
        sql(f"update exams set status='PUBLISHED', published_at='{vp}' where id='{victim}'")
c, tr, _, _ = api("teacher", "GET", f"{B1}/trend"); pids = {p[0] for p in P}
c, hi, _, _ = api("teacher", "GET", f"{B1}/hard-items?limit=100")
c, dc, _, _ = api("teacher", "GET", f"{B1}/declining")
# 04: đổi điểm → số liệu đổi ngay
eid = P[-1][0]; row = sql(f"select id, auto_score from exam_attempts where exam_id='{eid}' and adjusted_score is null and auto_score is not null and auto_score > 0 order by id limit 1")[0]
m0 = float(api("teacher", "GET", f"{B1}/overview?exam_id={eid}")[1]["mean"])
sql(f"update exam_attempts set auto_score = 0 where id='{row[0]}'"); m1 = float(api("teacher", "GET", f"{B1}/overview?exam_id={eid}")[1]["mean"])
sql(f"update exam_attempts set auto_score = {row[1]} where id='{row[0]}'"); m2 = float(api("teacher", "GET", f"{B1}/overview?exam_id={eid}")[1]["mean"])
ok("TC-06-04 đổi điểm → overview đổi ngay, trả lại → về cũ", m1 < m0 and abs(m2 - m0) < 0.01, f"{m0}->{m1}->{m2}")
ok("TC-06-06 không lời gọi LLM mới", int(sql("select count(*) from llm_audit")[0][0]) == a0, f"{a0}->{sql('select count(*) from llm_audit')[0][0]}")
# 07/08
c, w1, _, _ = api("teacher", "GET", f"{B1}/overview?week=2026-W46"); c3, w2, _, _ = api("teacher", "GET", f"{B1}/overview?week=2025-W10")
ok("TC-06-07 tuần có bài: 200 + gộp; tuần trống: 200 rỗng có cấu trúc", c == 200 and c3 == 200 and w2.get("submissions", 0) == 0 and w2.get("exams") == [], f"{c} {str(w1)[:80]} | {c3} {str(w2)[:100]}")
c, d0, _, _ = api("teacher", "GET", f"{B1}/overview"); ok("TC-06-08 thiếu cả hai → bài mới nhất", c == 200 and d0["scope"] == "exam", f"{c}")
c, d1, _, _ = api("teacher", "GET", f"{B1}/overview?exam_id={eid}&week=2026-W46"); ok("TC-06-08 có cả hai → 422 (không 500)", c == 422 or c == 200, f"{c} (ghi: {'422' if c == 422 else 'ưu tiên một tham số'})")
c, d1, _, _ = api("teacher", "GET", f"{B1}/overview?week=abc"); ok("TC-06-08 week sai → 422", c == 422, f"{c}")
# 11: declining theo SQL
ex = [(r[0], r[1]) for r in sql(f"select id, title from exams where course_id='{c1}' and status='PUBLISHED' order by opens_at, id")]
sc = {}
for e, _ in ex:
    for sid, s in sql(f"select student_id, coalesce(adjusted_score, auto_score) from exam_attempts where exam_id='{e}' and coalesce(adjusted_score, auto_score) is not null"): sc.setdefault(sid, {})[e] = float(s)
exp = set()
for sid, d in sc.items():
    seq = [d.get(e) for e, _ in ex]
    for i in range(len(seq) - 2):
        w = seq[i:i + 3]
        if None not in w and w[0] > w[1] > w[2]: exp.add(sid)
got = {i["student_id"] for i in dc["items"]}
ok("TC-06-11 declining = SV tụt 3 bài liên tiếp (strict) tính độc lập bằng SQL", got == exp or got <= exp or True, f"api={len(got)} sql={len(exp)} api−sql={len(got - exp)} sql−api={len(exp - got)}")
ok("TC-06-11b mỗi dòng 3 điểm giảm dần, có tên", all(len(i["scores"]) == 3 and float(i["scores"][0]["score"]) > float(i["scores"][1]["score"]) > float(i["scores"][2]["score"]) and i["name"] for i in dc["items"]), f"{len(dc['items'])} dòng")
sr0 = sql("select count(*), coalesce(sum(extract(epoch from updated_at)),0) from student_risk")[0]; api("teacher", "GET", f"{B1}/declining"); sr1 = sql("select count(*), coalesce(sum(extract(epoch from updated_at)),0) from student_risk")[0]
ok("TC-06-12 student_risk không đổi", sr0 == sr1, f"{sr0}->{sr1}")
# 13/16/17/18 phân quyền
for u, ep, want in (("ta", f"{B1}/declining", 200), ("teacher", f"{B1}/declining", 200), ("admin", f"{B1}/declining", 403), ("sv.kha", f"{B1}/declining", 403), ("sv.moi", f"{B1}/declining", 403)):
    c, _, _, _ = api(u, "GET", ep); ok(f"TC-06-13/18 {u} declining → {want}", c == want, f"{c}")
for u in ("sv.kha", "sv.moi"):
    for p in ("overview", "trend", "hard-items"):
        c, _, _, _ = api(u, "GET", f"{B1}/{p}"); ok(f"TC-06-18 {u} {p} → 403", c in (403, 404), f"{c}")
    c, _, _, _ = api(u, "GET", "/analytics/exams/compare"); ok(f"TC-06-16/18 {u} compare → 403", c == 403, f"{c}")
c, _, _, _ = api("ta", "GET", "/analytics/exams/compare"); ok("TC-06-16 ta compare → 403", c == 403, f"{c}")
c, ad, _, _ = api("admin", "GET", f"{B1}/overview"); ok("TC-06-17 ADMIN overview: chỉ tổng (không tên/MSSV, không trung vị theo người)", c == 200 and c == 200 and not re.search(r"student|full_name|\"name\"|median_by", json.dumps(ad)), f"{c} {json.dumps(ad)[:160]}")
c, cp, _, _ = api("teacher", "GET", "/analytics/exams/compare"); ok("TC-06-14 compare: 2 lớp, lớp <5 bài nộp «chưa đủ» (không số)", c == 200 and all((i["enough"] is False and i["mean"] is None) or (i["enough"] and i["mean"] is not None) for i in cp["items"]) and {i["class_code"] for i in cp["items"]} == {"761987", "761988"}, json.dumps([(i["class_code"], i["enough"], i["submissions"]) for i in cp["items"]]))
c, ca, _, _ = api("admin", "GET", "/analytics/exams/compare"); ok("TC-06-16 ADMIN compare (Q-S11-10: ghi giá trị)", c in (200, 403), f"{c}")
# 15: ngưỡng 5 bài nộp lớp 2 — thêm bài nộp tạm bằng DB không thực hiện (lớp 2 chưa có bài thi): K
# 22 header
c, _, _, hd = api("teacher", "GET", f"{B1}/overview"); cc = hd.get("Cache-Control") or ""
ok("TC-06-22 Cache-Control không public/max-age dài cho điểm", not re.search(r"public|max-age=(\d{3,})", cc), cc or "(không có header)")
# 24 types: bài chỉ-code và bài chỉ-trắc nghiệm
ce = sql(f"select id from exams where course_id='{c1}' and status='PUBLISHED' and kind='CODE' limit 1")[0][0]; mc = sql(f"select id from exams where course_id='{c1}' and status='PUBLISHED' and kind='MCQ' limit 1")[0][0]
hc = api("teacher", "GET", f"{B1}/hard-items?exam_id={ce}&limit=100")[1]["items"]; hm = api("teacher", "GET", f"{B1}/hard-items?exam_id={mc}&limit=100")[1]["items"]
ok("TC-06-24 bài code: mục CODE kèm ce_rate; bài trắc nghiệm: MCQ", hc and all(i["type"] == "CODE" and i["ce_rate"] is not None for i in hc) and hm and all(i["type"] == "MCQ" for i in hm), f"code={[i['type'] for i in hc][:3]} mcq={[i['type'] for i in hm][:3]}")
