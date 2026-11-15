# 10.6 pha 2 — G: L5 — M6 lý do loại câu hỏi (G09), M4 câu tương tự (G08), M2 ca hồi quy (G07). python gd_g.py
from gdlib import *
import re, subprocess, concurrent.futures
C2 = sql("select id from courses where class_code='761988'")[0][0]
# ============ G09 — lý do loại câu hỏi
QB = f"/courses/{C1}/questions"
pend = sql(f"select id, version from question_bank where course_id='{C1}' and review_status='PENDING' order by id limit 14")
ok("G09-00 có ≥ 12 câu PENDING", len(pend) >= 12, str(len(pend)))
reasons = ["MO_HO", "SAI_DAP_AN", "TRUNG_LAP", "LAC_DE", "KHO_DE_BAT_THUONG", "SAI_DINH_DANG", "UNSPECIFIED"]
for (qid, ver), rs in zip(pend[:7], reasons):
    c, j, _ = api("teacher", "PUT", f"{QB}/{qid}/review", {"decision": "REJECT", "reject_reason": rs, "version": int(ver)})
    got = sql(f"select review_status, reject_reason from question_bank where id='{qid}'")[0]
    ok(f"G09-01 REJECT reject_reason={rs} → lưu đúng", c == 200 and got == ["REJECTED", rs], f"{c} {got}")
qid, ver = pend[7]; c, j, _ = api("teacher", "PUT", f"{QB}/{qid}/review", {"decision": "REJECT", "version": int(ver)})
rr = sql(f"select coalesce(reject_reason,'<NULL>') from question_bank where id='{qid}'")[0][0]
ok("G09-01 REJECT không gửi lý do → UNSPECIFIED (US-GD-09 AC1)", c == 200 and rr == "UNSPECIFIED", f"{c} lưu = {rr}")
qid, ver = pend[8]; c, j, _ = api("teacher", "PUT", f"{QB}/{qid}/review", {"decision": "REJECT", "reject_reason": "XYZ", "version": int(ver)}); ok("G09-02 lý do lạ → 422", c == 422, str(c))
c, j, _ = api("teacher", "PUT", f"{QB}/{qid}/review", {"decision": "APPROVE", "reject_reason": "MO_HO", "version": int(ver)}); ok("G09-02 reject_reason kèm APPROVE → 422", c == 422, f"{c} {(j or {}).get('code')}")
ok("G09-02 DB CHECK: reject_reason chỉ khi REJECTED", sql(f"select count(*) from question_bank where reject_reason is not null and review_status <> 'REJECTED'")[0][0] == "0", "")
ids = [str(x[0]) for x in pend[9:12]]
c, j, _ = api("teacher", "POST", f"{QB}/bulk", {"action": "REJECT", "ids": ids, "reason": "TRUNG_LAP"}); print("INFO bulk reject:", c, json.dumps(j)[:200])
ok("G09-02 bulk reject + reason → lưu lý do cho cả lô", c == 200 and all(sql(f"select reject_reason from question_bank where id='{i}'")[0][0] == "TRUNG_LAP" for i in ids), str(c))
c, j, _ = api("teacher", "POST", f"{QB}/bulk", {"action": "REJECT", "ids": [str(pend[12][0])], "reason": "XYZ"}); ok("G09-02 bulk reason lạ → 422 hoặc từng id ok:false (không loại câu)", (c == 422) or (c == 200 and all(not x["ok"] for x in j["results"]) and sql(f"select review_status from question_bank where id='{pend[12][0]}'")[0][0] == "PENDING"), f"{c} {json.dumps(j)[:120]}")
# G09-11 phân quyền / không lộ cho SV
for u, exp, tag in (("sv.kha", 403, "SV"),):
    c, j, _ = api(u, "GET", QB); ok(f"G09-11 {tag} GET questions → {exp}", c == exp, str(c))
    c, j, _ = api(u, "GET", f"{QB}/gen-stats"); ok(f"G09-11 {tag} gen-stats → {exp}", c in (exp, 400, 404, 422), str(c))
c, _, _ = raw("GET", QB)[:3]; ok("G09-11 không token → 401", c == 401, str(c))
c, j, _ = api("teacher", "GET", f"/courses/{C2}/questions"); print("INFO teacher lớp 2 (là GV lớp 2):", c)
# SV lấy câu qua luyện đề: reject_reason không bao giờ trong DTO SV
leak = 0
for p in (f"/courses/{C1}/practice/questions", f"/courses/{C1}/practice", f"/courses/{C1}/practice/quizzes"):
    c, j, _ = api("sv.kha", "GET", p)
    if "reject_reason" in json.dumps(j): leak += 1
ok("G09-11 DTO SV không có reject_reason", leak == 0, str(leak))
# ============ G08 — câu tương tự
SIM = f"/courses/{C1}/threads/similar-confirmed"
src = sql(f"select t.id, t.title, regexp_replace(left(c.text, 1500), '[[:space:]|]+', ' ', 'g') from knowledge_items k join forum_posts p on p.id = k.source_id join forum_threads t on t.id = p.thread_id join content_chunks c on c.id = k.chunk_id where k.course_id='{C1}' and k.source_type='POST' limit 5")
print("INFO thread có tri thức POST:", [(s[1][:40]) for s in src])
flag("THREAD_SUGGEST", True)
if src:
    tid, title = src[0][0], src[0][1]
    c, j, ms = api("sv.kha", "POST", SIM, {"q": src[0][2][:1500]}); ok("G08-01 q = đúng chữ đã nhúng của thread xác nhận → có thread đó (≤ 3, DTO {thread_id,title})", c == 200 and any(x["thread_id"] == str(tid) for x in j) and len(j) <= 3 and all(set(x) == {"thread_id", "title"} for x in j), f"{c} {json.dumps(j)[:200]}")
    c, j, _ = api("sv.kha", "POST", SIM, {"q": "ab"}); ok("G08-02 q < 3 từ → [] hoặc 422, không 500", (c == 200 and j == []) or c == 422, f"{c} {json.dumps(j)[:60]}")
    c, j, _ = api("sv.kha", "POST", SIM, {"q": ""}); ok("G08-02 q rỗng → không 500", c in (200, 422), str(c))
    c, j, _ = api("sv.kha", "POST", SIM, {"q": "x " * 3000}); ok("G08-02 q 6.000 ký tự → không 500", c in (200, 413, 422), str(c))
    c, j, _ = api("sv.kha", "POST", SIM, {"q": "ZQ-L2-không-có-thật lạ chưa từng xuất hiện trong lớp nào"}); ok("G08-15 chuỗi lạ → [] (không lộ lớp khác)", c == 200 and j == [], f"{c} {json.dumps(j)[:60]}")
    c, _, _ = api("sv.kha", "POST", f"/courses/{C2}/threads/similar-confirmed", {"q": title}); ok("G08-16 SV lớp 1 gọi qua lớp 2 → 404/403", c in (403, 404), str(c))
    c, _, _ = api("sv31", "POST", SIM, {"q": title}); ok("G08-16 SV chỉ lớp 2 gọi lớp 1 → 404/403", c in (403, 404), str(c))
    for u in ("ta", "teacher"):
        c, _, _ = api(u, "POST", SIM, {"q": title}); ok(f"G08-16 {u} → 200", c == 200, str(c))
    c, _, _ = raw("POST", SIM, {"q": title})[:3]; ok("G08-16 không token → 401", c == 401, str(c))
    c, j, _ = api("sv.kha", "GET", SIM + "?q=x"); ok("G08-20 route GET cũ → 404/405", c in (404, 405), str(c))
    # G08-09 / 11: nhúng và độ trễ
    t0 = now_db(); lat = []
    for i in range(20):
        c, j, ms = api("sv.kha", "POST", SIM, {"q": title + (" " * 0) if i % 2 == 0 else title + f" biến thể số {i}"}); lat.append(ms)
    time.sleep(4)
    emb = sql(f"select count(*) from llm_audit where task='EMBEDDING' and created_at >= '{t0}'")[0][0]
    chat = sql(f"select count(*) from llm_audit where task in ('CHAT','UTILITY','GRADING','CLASSIFY') and created_at >= '{t0}'")[0][0]
    ok("G08-09 20 lần gợi ý: nhúng ≤ 11 (cùng chuỗi dùng cache), 0 lời gọi sinh chữ", int(emb) <= 11 and chat == "0", f"nhúng={emb} sinh chữ={chat}")
    lat.sort(); ok("G08-11 p95 ≤ 300 ms (20 lượt, embedding giả)", lat[int(len(lat) * 0.95) - 1] <= 300, f"p95={lat[int(len(lat)*0.95)-1]:.0f} ms p50={lat[len(lat)//2]:.0f}")
    # G08-19: chữ đang soạn không có ở log
    secret = "MSSV 20201234 zqsecret-gd08"
    api("sv.kha", "POST", SIM, {"q": secret}); time.sleep(3)
    logs = subprocess.run("docker logs ep-qc-gateway-1 --since 2m 2>&1; docker logs ep-qc-caddy-1 --since 2m 2>&1", shell=True, capture_output=True, text=True).stdout
    ok("G08-19 chữ đang soạn không có trong log gateway / Caddy", "zqsecret" not in logs and "20201234" not in logs, "")
    pe = sql("select count(*) from pii_events where created_at > now() - interval '2 minutes'")[0][0]; ok("G08-14 không ghi pii_events cho chuỗi đang soạn", pe == "0", pe)
    c, j, hd = raw("POST", SIM, {"q": title}, token=tok("sv.kha"))[:2] + (None,)
    r = raw("POST", SIM, {"q": title}, token=tok("sv.kha")); ok("G08-19 Cache-Control: no-store", "no-store" in str(r[3]).lower(), str({k: v for k, v in r[3].items() if k.lower() == 'cache-control'}))
    # G08-10 rate limit 30/phút
    codes = [api("sv.kha", "POST", SIM, {"q": f"gợi ý giới hạn tốc độ {i}"})[0] for i in range(35)]
    ok("G08-10 35 lần / phút: có 429", 429 in codes, f"{codes.count(200)}×200 {codes.count(429)}×429")
    c, _, _ = api("sv.gioi", "POST", SIM, {"q": title}); ok("G08-10 người khác không bị ảnh hưởng", c == 200, str(c))
    # G08-12 cờ tắt
    flag("THREAD_SUGGEST", False); t1 = now_db()
    c, j, _ = api("sv.kha", "POST", SIM, {"q": title + " cờ tắt"}); time.sleep(3)
    ok("G08-12 cờ tắt → [] và 0 nhúng", c == 200 and j == [] and sql(f"select count(*) from llm_audit where task='EMBEDDING' and created_at >= '{t1}'")[0][0] == "0", f"{c} {json.dumps(j)[:40]}")
    flag("THREAD_SUGGEST", True)
# ============ G07 — ca hồi quy
RC = f"select kind, criterion_id, expected::text, source_id::text from grading_regression_cases where course_id='{C1}'"
sql(f"delete from grading_regression_cases where course_id='{C1}'")
def edit_review(rid, edits):
    c, j, _ = api("teacher", "GET", f"/courses/{C1}/grading-results/{rid}"); v = j["result"]["version"]
    if edits:
        c, j2, _ = api("teacher", "PUT", f"/courses/{C1}/grading-results/{rid}", {"version": v, "criteria": edits}); v = j2["result"]["version"]
    return api("teacher", "POST", f"/courses/{C1}/grading-results/{rid}/review", {"version": v})[0]
cand = sql(f"select r.id, r.ai_scores->>'c2', r.ai_scores->>'c3' from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status='DRAFT' and r.run_state='DONE' and r.round=(select max(round) from grading_results x where x.submission_id=r.submission_id) and (r.ai_scores->>'c2')::numeric >= 1.0 and (r.ai_scores->>'c3')::numeric <= 3.0 order by r.id limit 4")
print("INFO ứng viên:", cand)
flag_admin = lambda en: api("admin", "PUT", FL + "/REGRESSION_CAPTURE", {"enabled": en, "version": [x for x in api("admin", "GET", FL)[1]["items"] if x["feature"] == "REGRESSION_CAPTURE"][0]["version"]})[0]
# G07-04: cờ tắt → 0 hàng
flag_admin(False); r0, a2, a3 = cand[0]
c = edit_review(r0, [{"id": "c2", "score": f"{float(a2) - 1.0:.2f}"}]); ok("G07-04 cờ tắt: review có sửa 1,0 → 0 hàng ca", c == 200 and sql(RC) == [], f"{c} {sql(RC)}")
# G07-01: cờ bật; sửa c2 lệch đúng 1,0 (biên) và c3 lệch 0,5 (< 1,0)
flag_admin(True)
r1, a2, a3 = cand[1]
c = edit_review(r1, [{"id": "c2", "score": f"{float(a2) - 1.0:.2f}"}, {"id": "c3", "score": f"{float(a3) + 0.5:.2f}"}])
rows = sql(RC); print("INFO ca:", rows)
ok("G07-01 lệch đúng 1,0 ghi 1 hàng GRADING_EDIT (c2); lệch 0,5 (c3) không ghi", c == 200 and len(rows) == 1 and rows[0][0] == "GRADING_EDIT" and rows[0][1] == "c2" and rows[0][3] == str(r1), f"{c} {rows}")
exp = json.loads(rows[0][2]) if rows else {}
ok("G07-03 expected chỉ số / enum, không chuỗi ≥ 40 ký tự, không văn bản", all(not (isinstance(v, str) and len(v) >= 40) for v in exp.values()) and len(rows[0][2]) < 300 if rows else False, rows[0][2] if rows else "")
print("INFO expected:", exp)
# G07-05: sửa lại cùng tiêu chí → một hàng, expected mới nhất (đưa về nháp bằng SQL không; review lần nữa idempotent)
c, j, _ = api("teacher", "GET", f"/courses/{C1}/grading-results/{r1}"); v = j["result"]["version"]
c2_, j2, _ = api("teacher", "PUT", f"/courses/{C1}/grading-results/{r1}", {"version": v, "criteria": [{"id": "c2", "score": f"{float(a2) + 1.0:.2f}"}]}); print("INFO sửa lại sau review:", c2_, (j2 or {}).get("code"))
rows2 = sql(RC); ok("G07-05 sửa lại cùng tiêu chí: vẫn 1 hàng", len(rows2) == 1, str(len(rows2)))
# G07-01 thêm: bài khác lệch 1,5 → hàng mới; tổng 2 hàng
r2, a2, a3 = cand[2]; edit_review(r2, [{"id": "c2", "score": f"{float(a2) - 1.0:.2f}"}]); ok("G07-01 bài thứ hai → tổng 2 hàng", len(sql(RC)) == 2, str(len(sql(RC))))
# G07-13 phân quyền: số ca
for p in (f"/courses/{C1}/regression-cases", f"/courses/{C1}/guidance/regression", f"/courses/{C1}/assignments/{A}/guidance/stats"):
    c, j, _ = api("teacher", "GET", p); print("INFO GET", p[-34:], c, json.dumps(j)[:100])
# G07-11 export bị từ chối
env = dict(os.environ, DATABASE_URL="postgres://x", REGRESSION_EXPORT_ALLOWED="false")
r = subprocess.run("cd ~/Documents/TA_Agent_qc5/backend-go && go run ./cmd/regress export --course %s 2>&1; echo exit=$?" % C1, shell=True, capture_output=True, text=True, env=env)
ok("G07-11 `regress export` mặc định từ chối (thoát ≠ 0)", "exit=0" not in r.stdout, r.stdout.strip().splitlines()[-2:][0][:140] if r.stdout.strip() else "")
print("INFO regress export:", r.stdout.strip()[-200:])
flag_admin(False)
