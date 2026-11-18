#!/usr/bin/env python3
"""TC-P7-08 phần API/DB (02 chữ cờ, 03 hiệu năng 1.000 bài, 06 pass_detail, 07 sửa điểm, 08 trừ muộn tay / version, 10 duyệt, 11 NOT_READY/SUPERSEDED, 12 RBAC, 14 Hôm nay). Dựng dữ liệu UI cho spec Playwright (/tmp/q9s/seed08.json)."""
import sys, os, statistics, threading; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("08"); code = lambda: str(int(time.time()) % 100000)
ALL = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 41)]
C = mkcourse("QD9U", ALL); Q = f"/api/v1/courses/{C}"; T = tok("teacher"); TA = tok("ta"); AD = tok("admin"); SB = tok("sv.kha"); SA = tok("sv.gioi")
body = lambda u, x="": (f"Bài làm của {u}. Mã độc SUNBURST chờ khoảng 10 ngày. {x} " * 6).encode()
ver = lambda rid: int(psql(f"select version from grading_results where id='{rid}'"))
def put(rid, crit, tk=None, fb="Nhận xét", extra=None, v=None):
    b = {"version": v or ver(rid), "criteria": [{"id": f"c{i+1}", "score": s} for i, s in enumerate(crit)], "overall_feedback": fb}; b.update(extra or {}); return call("PUT", f"{Q}/grading-results/{rid}", tk or T, b)
with stub_ctx():
    S = seed_bt03(Q, T, "Q8A" + code(), ["sv.gioi", "sv.kha", "sv04", "sv08", "sv10", "sv12", "sv14", "sv19", "sv24"]); aid = S["aid"]; res = S["res"]; sub = S["sub"]
    print("   seed:", json.dumps({k: v[:8] if isinstance(v, str) else v for k, v in S["res"].items()})[:300])
    flags = {a: (result(aid, a) or {}).get("flags") for a in res}; print("   cờ:", flags)
    B = res["sv.kha"]; rb = result(aid, "sv.kha")
    # --- 08-06 pass_detail lệch hai lượt
    pd = rb["pass_detail"]; print("   B pass_detail:", json.dumps(pd, ensure_ascii=False)[:300])
    R.chk("TC-P7-08-06", f"B (S7) cờ NEEDS_REVIEW và pass_detail lượt 1/lượt 2 tiêu chí 2 = {pd.get('c2') if isinstance(pd, dict) else pd}", "NEEDS_REVIEW" in rb["flags"], pd)
    # --- 08-02 chữ cờ (nguồn: FE FLAG_TEXT) + sinh thêm cờ
    fe = open(os.path.expanduser("~/Documents/TA_Agent_qc106/frontend/src/features/grading/gradingApi.ts"), encoding="utf-8").read()
    want = {"NEEDS_REVIEW": "Hai lượt chấm lệch hơn 1 điểm", "OFF_TOPIC": "Bài lạc đề", "INJECTION_SUSPECTED": "Nghi chèn lệnh", "EMPTY_SUBMISSION": "Thiếu nội dung", "EVIDENCE_NOT_FOUND": "Trích dẫn không có trong bài", "LOW_CONFIDENCE": "AI không chắc", "ANSWER_ECHO": "Nhận xét giống đáp án chuẩn", "LATE_AFTER_CUTOFF": "Thư nộp quá hạn tối đa"}
    miss = {k: v for k, v in want.items() if f'{k}: "{v}"' not in fe}; R.chk("TC-P7-08-02", f"FLAG_TEXT có đủ 8 câu đúng danh sách AC1 (EMPTY_SUBMISSION = «Thiếu nội dung» theo P-11-17 thay «Bài trống»); thiếu/sai: {miss}", not miss, miss)
    a2 = mkassign(Q, T, "Q802" + code(), allow_late=False); 
    for u, m in (("sv.gioi", "[[S9G conf=0.3]]"), ("sv.kha", "[[S9G echo=12]]"), ("sv.nguyco", "[[S9G flags=OFF_TOPIC]]")): submit(Q, a2, tok(u), "b.md", body(u, m))
    submit(Q, a2, tok("sv04"), "b.md", body("sv04")); psql(f"update assignments set due_at=now()-interval '3 days' where id='{a2}'"); psql(f"update submissions set submitted_at = now() - interval '1 day' where assignment_id='{a2}' and student_id=(select id from users where email='sv04@edupilot.local')")
    psql(f"update submissions set source='IMAP' where assignment_id='{a2}' and student_id=(select id from users where email='sv04@edupilot.local')")  # thư (IMAP) nộp sau hạn khi không cho nộp muộn
    call("POST", f"{Q}/assignments/{a2}/grade-all", T, {}); wait_done(a2, 120); time.sleep(2)
    f2 = {u: (result(a2, u) or {}).get("flags") for u in ("sv.gioi", "sv.kha", "sv.nguyco", "sv04")}; print("   cờ bổ sung:", f2)
    seen = set(sum([x for x in list(flags.values()) + list(f2.values()) if x], []))
    R.chk("TC-P7-08-02", f"cờ thực sinh ra ở dữ liệu thử: {sorted(seen)} ⊇ 8 loại cần kiểm", {"NEEDS_REVIEW", "OFF_TOPIC", "INJECTION_SUSPECTED", "EMPTY_SUBMISSION", "EVIDENCE_NOT_FOUND", "LOW_CONFIDENCE", "ANSWER_ECHO", "LATE_AFTER_CUTOFF"} <= seen, sorted(seen))
    # --- 08-07 sửa điểm
    cur = rb["criteria"] if "criteria" in rb else None; base = ["2.00", "1.00", "2.00", "2.00"]
    for val, want_ok, wcode in [("2,00", 200, None), ("2.25", 200, None), ("2,3", 422, "SCORE_STEP"), ("2,555", 422, "SCORE_PRECISION"), ("-1", 422, "SCORE_OUT_OF_RANGE"), ("2,75", 422, "SCORE_OUT_OF_RANGE"), ("abc", 422, None)]:
        s, x, _ = put(B, [base[0], val, base[2], base[3]]); R.chk("TC-P7-08-07", f"c2={val!r} → {s} {wcode or ''} ({str(x)[:90]})", s == want_ok and (wcode is None or wcode in json.dumps(x)), (s, x))
    s, x, _ = put(B, ["2.5", "2.00", "1.5", "2.0"]); r2 = result(aid, "sv.kha")
    tot_before = float(r2["ai_total_before_late"]) if r2["ai_total_before_late"] else None
    R.chk("TC-P7-08-07", f"tổng/dòng muộn tính lại ở máy chủ bằng decimal: ai_total_before_late={r2['ai_total_before_late']} (giữ nguyên), total={r2['total']}, late_penalty={r2['late_penalty']}", s == 200 and r2["total"] is not None and abs(float(r2["total"]) - (8.0 - float(r2["late_penalty"]))) < 1e-9, (s, r2["total"], r2["late_penalty"]))
    ai0 = rb["ai_scores"]; R.chk("TC-P7-08-07", f"ai_scores không đổi sau khi sửa ({ai0} → {r2['ai_scores']})", ai0 == r2["ai_scores"], r2["ai_scores"])
    # tái hiện đúng số TC: c2 1,00 → 2,00 trong khi các tiêu chí khác giữ nguyên bản AI
    aiB = rb["ai_scores"]; print("   AI B:", aiB, "total trước muộn", rb["ai_total_before_late"], "muộn", rb["late_penalty"], "total", rb["total"])
    s, x, _ = put(B, [aiB["c1"], "2.00", aiB["c3"], aiB["c4"]]); r3 = result(aid, "sv.kha")
    R.chk("TC-P7-08-07", f"đúng số TC (B): c2 1,00→2,00: tổng trước muộn {rb['ai_total_before_late']} → {float(rb['ai_total_before_late']) + 1:.2f}, trừ muộn {r3['late_penalty']}, tổng {r3['total']}", abs(float(r3["total"]) - (float(rb["ai_total_before_late"]) + 1 - float(r3["late_penalty"]))) < 1e-9 and float(r3["late_penalty"]) == 0.5, (rb["ai_total_before_late"], r3))
    # --- 08-08 trừ muộn tay + đổi hạn; version; sửa published
    s, x, _ = put(B, [aiB["c1"], "2.00", aiB["c3"], aiB["c4"]], extra={"late_penalty": "0"}); r4 = result(aid, "sv.kha"); print("   sửa muộn tay:", s, str(x)[:150], r4["late_penalty"], r4.get("late_penalty_manual"))
    R.chk("TC-P7-08-08", f"sửa tay Trừ nộp muộn 0,50 → 0: {s}, late_penalty={r4['late_penalty']}", s == 200 and float(r4["late_penalty"]) == 0.0, (s, x))
    ad_ = call("GET", f"{Q}/assignments/{aid}", T)[1]; a_ = ad_.get("assignment", ad_)
    due2 = (time.time() - 2 * 86400); newdue = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(due2 - 5 * 86400))  # đẩy hạn lùi 5 ngày → bài sẽ muộn hơn nếu tính lại
    pb = {k: a_[k] for k in ("title", "description", "max_score", "allow_late", "late_penalty_per_day", "late_max_days", "appeal_days", "channels", "submission_code", "reference_answer") if k in a_}; pb.update({"due_at": newdue, "version": a_["version"]})
    s, x, _ = call("PUT", f"{Q}/assignments/{aid}", T, pb); r5 = result(aid, "sv.kha"); print("   đổi hạn:", s, str(x)[:120], "late_penalty sau:", r5["late_penalty"])
    R.chk("TC-P7-08-08", f"đổi hạn không tính đè khoản trừ sửa tay (late_penalty vẫn {r5['late_penalty']}) [PUT hạn → {s}]", s == 200 and float(r5["late_penalty"]) == 0.0, (s, r5["late_penalty"]))
    s, x, _ = put(B, [aiB["c1"], "2.00", aiB["c3"], aiB["c4"]], v=1); R.chk("TC-P7-08-08", f"sai version → {s} VERSION_CONFLICT kèm giá trị của người kia", s == 409 and "VERSION_CONFLICT" in json.dumps(x), (s, str(x)[:200]))
    print("   thân 409:", json.dumps(x, ensure_ascii=False)[:300])
    # --- 08-10 duyệt
    s, x, _ = call("POST", f"{Q}/grading-results/{B}/review", T, {"version": ver(B)}); st = psql(f"select status||'|'||coalesce(reviewed_by::text,'-')||'|'||(reviewed_at is not null) from grading_results where id='{B}'")
    al = psql(f"select count(*) from audit_log where course_id='{C}' and entity_id='{B}' and action ilike '%review%'")
    R.chk("TC-P7-08-10", f"POST review → {s}; DB {st[:20]}…; audit_log review {al} dòng", s == 200 and st.startswith("REVIEWED") and int(al) >= 1, (s, x, st, al))
    s, x, _ = call("POST", f"{Q}/grading-results/{B}/review", T, {"version": ver(B)}); R.chk("TC-P7-08-10", f"duyệt lại → idempotent ({s})", s == 200, (s, str(x)[:150]))
    e = res["sv.gioi"]; s, x, _ = call("POST", f"{Q}/grading-results/{e}/review", T, {"version": ver(e)}); R.chk("TC-P7-08-10", f"duyệt bài còn cờ/không cờ được ({s})", s == 200, x)
    ev = res["sv.kha"]; s, lst, _ = call("GET", f"{Q}/submissions?assignment_id={aid}&needs_review=true&limit=50", T); ids = [i["result"]["id"] for i in lst.get("items", [])]
    # bài trống (S4 sv24): FAILED/EMPTY chưa điểm → SCORES_REQUIRED
    e24 = res["sv.nguyco"] if False else res["sv24"] if "sv24" in res else res.get("sv24")
    r24 = result(aid, "sv24"); print("   sv24 (S4 trống):", r24["run_state"], r24["status"], r24["total"], r24["criteria"][:1] if r24["criteria"] else None)
    s, x, _ = call("POST", f"{Q}/grading-results/{r24['id']}/review", T, {"version": r24["version"]}); R.chk("TC-P7-08-10", f"bài trống/FAILED chưa có điểm đủ → 422 SCORES_REQUIRED ({s}) — S4 có 0 điểm cờ EMPTY nên {'được duyệt' if s == 200 else s}", (s == 422 and "SCORES_REQUIRED" in json.dumps(x)) or (s == 200 and r24["total"] is not None), (s, x))
    fr = psql(f"select gr.id||'|'||gr.version from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id in (select id from assignments where course_id='{C}') and gr.run_state='FAILED' and gr.criteria::text !~ 'score[^a-z0-9]+[0-9]' limit 1")
    if fr:
        i_, v_ = fr.split("|"); s, x, _ = call("POST", f"{Q}/grading-results/{i_}/review", T, {"version": int(v_)}); R.chk("TC-P7-08-10", f"bài FAILED chưa điểm → 422 SCORES_REQUIRED ({s})", s == 422 and "SCORES_REQUIRED" in json.dumps(x), (s, x))
    # --- 08-11
    a11 = mkassign(Q, T, "Q811" + code()); 
    for u in ["sv04", "sv05", "sv06", "sv07", "sv09"]: submit(Q, a11, tok(u), "b.md", body(u, "[[S9G slow=12]]"))
    submit(Q, a11, tok("sv.kha"), "b.md", body("sv.kha", "V1"))
    call("POST", f"{Q}/assignments/{a11}/grade-all", T, {}); time.sleep(2); rq = result(a11, "sv.kha")
    s, x, _ = call("POST", f"{Q}/grading-results/{rq['id']}/review", T, {"version": rq["version"]}); R.chk("TC-P7-08-11", f"bài {rq['run_state']} → review {s} NOT_READY", rq["run_state"] in ("QUEUED", "RUNNING") and s == 409 and "NOT_READY" in json.dumps(x), (s, str(x)[:150]))
    wait_done(a11, 90); time.sleep(2); rd = result(a11, "sv.kha"); submit(Q, a11, tok("sv.kha"), "b.md", body("sv.kha", "V2 nộp lại"))
    s, x, _ = call("POST", f"{Q}/grading-results/{rd['id']}/review", T, {"version": rd["version"]}); R.chk("TC-P7-08-11", f"sinh viên nộp lại khi giảng viên mở bản cũ → review bản cũ {s} SUPERSEDED", s == 409 and "SUPERSEDED" in json.dumps(x), (s, str(x)[:200]))
    # --- 08-12 RBAC
    rid = res["sv04"]; tests = []
    for who, tk, want in (("TEACHER", T, 200), ("TA", TA, 200), ("STUDENT", SB, 403), ("ADMIN", AD, 403)):
        for p in (f"/grading-results/{rid}", f"/submissions?limit=5", f"/submissions/{sub['sv04']}", f"/submissions/{sub['sv04']}/file"):
            s, x, _ = call("GET", f"{Q}{p}", tk); tests.append((who, p.split("/")[1][:12], s, want)); 
    bad = [t for t in tests if t[2] != t[3] and not (t[1] == "submissions" and t[2] in (200, 302) and t[3] == 200)]
    print("   RBAC:", [(w, p, s) for w, p, s, _ in tests if _ != s][:6])
    R.chk("TC-P7-08-12", f"TA/TEACHER 200; STUDENT/ADMIN 403 cho grading-results, submissions, chi tiết, tệp ({len(tests)} ca, lệch {len(bad)})", not bad, bad)
    s, x, _ = call("POST", f"{Q}/grading-results/publish", TA, {"result_ids": [rid]}); R.chk("TC-P7-08-12", f"TA publish → {s} (403)", s == 403, x)
    s, x, _ = put(rid, ["2.5", "2.5", "2.5", "2.5"], tk=TA); R.chk("TC-P7-08-12", f"TA sửa điểm nháp → {s}", s == 200, x)
    s, x, _ = call("POST", f"{Q}/grading-results/{rid}/review", TA, {"version": ver(rid)}); R.chk("TC-P7-08-12", f"TA duyệt → {s}", s == 200, x)
    s, x, _ = call("GET", f"/api/v1/courses/{C}/grading-results/{rid}", tok("qct2")); R.chk("TC-P7-08-12", f"giảng viên lớp khác (qct2) → {s} 403", s == 403, x)
    # --- 08-14 Hôm nay
    n0 = {}
    for who, u in (("TEACHER", "teacher"), ("TA", "ta"), ("STUDENT", "sv.kha"), ("ADMIN", "admin")):
        s, it, j = today(u, C); ks = [(i["kind"], i["title"][:44], i.get("href")) for i in it]; n0[who] = ks; print("  ", who, s, ks[:6])
    fr_ = [k for k in n0["TEACHER"] if k[0] == "GRADING_ERROR"]; rv_ = [k for k in n0["TEACHER"] if k[0] == "GRADING_REVIEW"]
    R.chk("TC-P7-08-14", f"TEACHER: GRADING_REVIEW {rv_} (href /grading) và GRADING_ERROR {fr_} (href /grading?status=FAILED)", bool(rv_) and rv_[0][2] == "/grading" and (not fr_ or fr_[0][2] == "/grading?status=FAILED"), n0["TEACHER"])
    R.chk("TC-P7-08-14", "TA có cùng việc; STUDENT/ADMIN không có GRADING_*", any(k[0] == "GRADING_REVIEW" for k in n0["TA"]) and not any(k[0].startswith("GRADING_") for k in n0["STUDENT"] + n0["ADMIN"]), n0)
    # xoá hết bài cần xem kỹ → biến mất ≤ 5 s (duyệt hết)
    ids = [i["result"]["id"] for i in call("GET", f"{Q}/submissions?assignment_id={aid}&needs_review=true&limit=100", T)[1].get("items", []) if i["result"]["status"] == "DRAFT"]
    for i_ in ids:
        v_ = ver(i_); call("POST", f"{Q}/grading-results/{i_}/review", T, {"version": v_})
    gone = False; t0 = time.time()
    for _ in range(80):
        if not [i for i in today("teacher", C)[1] if i["kind"] == "GRADING_REVIEW"]: gone = True; break
        time.sleep(0.2)
    dt = time.time() - t0
    R.chk("TC-P7-08-14", f"duyệt hết bài cần xem kỹ → GRADING_REVIEW biến mất sau {dt:.1f} s kể từ lần duyệt cuối (≤5 s)", gone and dt <= 5.0, dt)
    json.dump({"course": C, "aid": aid, "res": res, "sub": sub}, open(f"{TMP}/seed08.json", "w"))
    # --- 08-03 hiệu năng 1.000 bài (SQL generate_series, lớp riêng)
    CP = mkcourse("QD9P", ALL); QP = f"/api/v1/courses/{CP}"
    if int(psql(f"select count(*) from submissions where course_id='{CP}'")) < 1000:
        tid = psql("select id from users where email='teacher@edupilot.local'")
        for i in range(25):
            ap = psql(f"insert into assignments (course_id, title, max_score, due_at, submission_code, created_by, published_at, rubric) values ('{CP}', 'Perf {i}', 10, now() - interval '5 days', 'P{i:02d}X{code()[-2:]}', '{tid}', now() - interval '6 days', '[{{\"id\":\"c1\",\"name\":\"A\",\"max_score\":\"10\",\"guidance\":\"\"}}]'::jsonb) returning id").splitlines()[0]
            psql(f"insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, extracted_text) select '{CP}', '{ap}', e.user_id, 'IN_APP', 1, 'k/'||e.user_id, 'a.txt', 100, 'text/plain', md5(e.user_id::text), 'MATCHED', 'IN_APP', now() - interval '6 days', 'x' from enrollments e where e.course_id='{CP}' and e.role_in_course='STUDENT'")
        psql(f"insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, flags) select s.course_id, s.id, 'DONE', 'DRAFT', '[]'::jsonb, 5, 5, case when (row_number() over ())%3=0 then array['EVIDENCE_NOT_FOUND']::text[] else '{{}}'::text[] end from submissions s where s.course_id='{CP}'")
    n = psql(f"select count(*) from grading_results where course_id='{CP}'"); print("   lớp perf:", n, "bài")
    tms = []
    for _ in range(40):
        t0 = time.time(); s, x, _ = call("GET", f"{QP}/submissions?limit=50&needs_review=true", T); tms.append((time.time() - t0) * 1000)
    tms.sort(); p95 = tms[int(len(tms) * 0.95) - 1]
    R.chk("TC-P7-08-03", f"{n} bài: GET submissions trang đầu p95={p95:.0f} ms (≤300), p50={statistics.median(tms):.0f} ms", p95 <= 300 and s == 200, tms[-3:])
    seen_ids = []; cur = None; pages = 0
    while True:
        s, x, _ = call("GET", f"{QP}/submissions?limit=100" + (f"&cursor={cur}" if cur else ""), T); seen_ids += [i["id"] for i in x["items"]]; pages += 1; cur = x.get("next_cursor")
        if not cur or pages > 30: break
    R.chk("TC-P7-08-03", f"phân trang con trỏ {pages} trang liên tiếp: {len(seen_ids)} dòng, trùng {len(seen_ids) - len(set(seen_ids))}, đủ 1.000 ({n})", len(seen_ids) == len(set(seen_ids)) == int(n), (len(seen_ids), len(set(seen_ids))))
    s, x, _ = call("GET", f"{QP}/submissions?limit=101", T); R.chk("TC-P7-08-03", f"limit=101 → {s} 422 (limit ≤ 100); không OFFSET (cursor)", s == 422, x)
    gen = psql("select is_generated from information_schema.columns where table_name='grading_results' and column_name='needs_review'"); R.chk("TC-P7-08-03", f"needs_review là cột sinh ({gen})", gen == "ALWAYS", gen)
    plan = psql(f"explain (analyze, costs off) select s.id from submissions s join grading_results r on r.course_id=s.course_id and r.submission_id=s.id where s.course_id='{CP}' and s.match_status='MATCHED' and r.needs_review order by s.submitted_at desc, s.id desc limit 50"); print(plan[:900])
    R.chk("TC-P7-08-03", "EXPLAIN (ANALYZE) truy vấn hàng chờ: không Seq Scan trên submissions/grading_results", "Seq Scan on submissions" not in plan and "Seq Scan on grading_results" not in plan, plan[:400])
sys.exit(R.end())
