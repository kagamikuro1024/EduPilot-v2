#!/usr/bin/env python3
"""TC-P7-09-10/14/16/17/18, 10-02/03/08/10, 02-14 (API). Lớp QD9H. Stub dưới khoá global."""
import sys, os, threading; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("09x"); code = lambda: str(int(time.time()) % 100000)
ALL = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 41)]
C = mkcourse("QD9K" + str(int(time.time()) % 1000), ALL); Q = f"/api/v1/courses/{C}"; T = tok("teacher"); TA = tok("ta"); AD = tok("admin"); T2 = tok("qct2")
SA, SB, SC = tok("sv.gioi"), tok("sv.kha"), tok("sv.nguyco")
ver = lambda rid: int(psql(f"select version from grading_results where id='{rid}'"))
def edit(rid, sc, fb="Nhận xét"): return call("PUT", f"{Q}/grading-results/{rid}", T, {"version": ver(rid), "criteria": [{"id": f"c{i+1}", "score": s} for i, s in enumerate(sc)], "overall_feedback": fb})
review = lambda rid, tk=None: call("POST", f"{Q}/grading-results/{rid}/review", tk or T, {"version": ver(rid)})
pub = lambda tk, body: call("POST", f"{Q}/grading-results/publish", tk, body)
def stats(aid, tk): return call("GET", f"{Q}/assignments/{aid}/stats", tk)
def strip(x): 
    d = json.loads(json.dumps(x)); d.pop("trace_id", None); return json.dumps(d, sort_keys=True, ensure_ascii=False)
with stub_ctx():
    S = seed_bt03(Q, T, "Q9X" + code(), ["sv.gioi", "sv.kha", "sv04", "sv05", "sv06"]); aid = S["aid"]; res = S["res"]; sub = S["sub"]
    rA, rB, r4, r5, r6 = (res[k] for k in ("sv.gioi", "sv.kha", "sv04", "sv05", "sv06"))
    ai = result(aid, "sv.kha")["ai_scores"]; base_before = result(aid, "sv.kha")["ai_total_before_late"]
    # ---- 09-10 mẫu số 0 trước công bố
    s, st0, _ = stats(aid, T); R.chk("TC-P7-09-10", f"stats khi chưa công bố: có đủ 7 khoá, tỷ lệ null (không NaN/0): {st0}", s == 200 and all(k in st0 for k in ("published", "mean_abs_deviation", "teacher_changed_rate", "appealed", "appeal_rate", "adjusted", "appeal_success_rate")) and st0["published"] == 0 and st0["mean_abs_deviation"] is None and st0["appeal_rate"] is None and st0["appeal_success_rate"] is None, st0)
    # B: 7,00 -> sửa c2 1,00 -> 2,00; duyệt A, B, sv05; sv04 để DRAFT
    s, x, _ = edit(rB, [ai["c1"], "2.00", ai["c3"], ai["c4"]]); R.chk("TC-P7-09-10", f"B sửa c2 → 200 (nháp {base_before})", s == 200, x)
    for r in (rA, rB, r5): review(r)
    # ---- 09-18 trước công bố
    def k(u):
        seen = {}
        for i in today(u, C)[1] + [x for x in today(u)[1] if (x.get("course") or {}).get("id") == C]: seen.setdefault((i["kind"], i["title"] + " | " + (i.get("reason") or "")), i.get("href"))
        return [(a_, b_, h_) for (a_, b_), h_ in seen.items()]
    kt, kta = k("teacher"), k("ta"); pp = [x for x in kt if x[0] == "PUBLISH_PENDING"]
    print("   TEACHER today:", kt); print("   TA today:", kta)
    R.chk("TC-P7-09-18", f"TEACHER có PUBLISH_PENDING «{pp[0][1] if pp else None}» href {pp[0][2] if pp else None}; TA không", bool(pp) and "3 bài đã duyệt chờ công bố" in pp[0][1] and pp[0][2] == "/grading" and not [x for x in kta if x[0] == "PUBLISH_PENDING"], (kt, kta))
    # ---- 09-17 ma trận publish / stats / đoán result id
    codes = {}
    for who, tk in (("TA", TA), ("STUDENT", SB), ("ADMIN", AD), ("GV lớp khác", T2)): codes[who] = pub(tk, {"result_ids": [r5]})[0]
    R.chk("TC-P7-09-17", f"publish: TA/STUDENT/ADMIN/GV lớp khác → {codes} (đều 403)", all(v == 403 for v in codes.values()), codes)
    sc = {w: stats(aid, tk)[0] for w, tk in (("TEACHER", T), ("TA", TA), ("STUDENT", SB), ("ADMIN", AD), ("GV khác", T2))}
    R.chk("TC-P7-09-17", f"stats: {sc} (TEACHER/TA 200, còn lại 403)", sc["TEACHER"] == 200 and sc["TA"] == 200 and all(sc[w] == 403 for w in ("STUDENT", "ADMIN", "GV khác")), sc)
    fake = str(uuid.uuid4()); s1, x1, _ = call("GET", f"{Q}/grading-results/{rB}", SA); s2, x2, _ = call("GET", f"{Q}/grading-results/{fake}", SA)
    R.chk("TC-P7-09-17", f"A đọc kết quả của B → {s1}; id không tồn tại → {s2}; thân giống nhau (trừ trace_id): {strip(x1) == strip(x2)}", s1 == s2 and s1 in (403, 404) and strip(x1) == strip(x2), (x1, x2))
    # ---- 09-14 đoán
    R.note("TC-P7-09-14", "B/ sv05 đoán…"); S5 = tok("sv05"); S4 = tok("sv04")
    own = {}
    for who, tk, r_, sb in (("sv04 DRAFT", S4, r4, sub["sv04"]), ("sv05 REVIEWED", S5, r5, sub["sv05"])):
        o = []
        o.append(call("GET", f"{Q}/grading-results/{r_}", tk)[0]); o.append(call("GET", f"{Q}/submissions/{sb}", tk)[0])
        for suffix, hd in (("?include=draft", None), ("?debug=1", None), ("", {"X-Include-Draft": "1"})):
            s, m, _ = call("GET", f"{Q}/assignments/{aid}/mine{suffix}", tk, hdr=hd); txt = json.dumps(m, ensure_ascii=False); o.append((s, not any(w in txt for w in ("ai_scores", "flags", "pass_detail", "confidence", "needs_review")) and not m.get("result")))
        own[who] = o
    print("   đoán:", own)
    R.chk("TC-P7-09-14", f"grading-results/{{id}} và submissions/{{id}} của chính mình → 403; ?include=draft, ?debug=1, X-Include-Draft bị bỏ qua (không lộ nháp/cờ): {own}", all(v[0] == 403 and v[1] == 403 and all(x == (200, True) for x in v[2:]) for v in own.values()), own)
    def resub(tk):
        d = ("Nộp lại. SUNBURST. " * 12).encode(); s, pre, _ = call("POST", f"{Q}/assignments/{aid}/submit/presign", tk, {"filename": "again.md", "size_bytes": len(d), "sha256": hashlib.sha256(d).hexdigest()})
        if s != 200: return s, pre, None
        raw(pre["method"], pre["url"], pre["headers"], d); return call("POST", f"{Q}/assignments/{aid}/submit", tk, {"upload_id": pre["upload_id"]}, idem())
    s4_, x4_, _ = resub(S4); s5_, x5_, _ = resub(S5); print("   nộp lại DRAFT:", s4_, str(x4_)[:120], "| REVIEWED:", s5_, str(x5_)[:160])
    R.chk("TC-P7-09-14", f"nộp lại: bài DRAFT → {s4_} {strip(x4_)[:100]} | bài REVIEWED → {s5_} {strip(x5_)[:100]}: thân lỗi giống hệt (chữ + mã): {strip(x4_) == strip(x5_)}", s4_ == s5_ and s5_ >= 400 and strip(x4_) == strip(x5_) and not any(w in json.dumps(x5_, ensure_ascii=False).lower() for w in ("reviewed", "draft", "nháp")), (s4_, x4_, s5_, x5_))
    # ---- 09-10 công bố B + A + sv05
    s, x, _ = pub(T, {"result_ids": [rA, rB, r5]}); R.chk("TC-P7-09-10", f"TEACHER công bố A,B,sv05 → {s}", s == 200 and x.get("published") == 3, x)
    dv = psql(f"select deviation::text||'|'||teacher_changed_criteria::text from grading_results where id='{rB}'"); print("   B deviation|changed:", dv)
    R.chk("TC-P7-09-10", f"B: deviation|teacher_changed_criteria = {dv} (1.00 và {{c2}})", dv == "1.00|{c2}", dv)
    for who, tk in (("TA", TA), ("TEACHER", T)):
        s, st, _ = stats(aid, tk); R.chk("TC-P7-09-10", f"stats {who} sau công bố: {st}", s == 200 and st["published"] == 3 and st["mean_abs_deviation"] not in (None, "NaN"), st)
    R.chk("TC-P7-09-10", f"sinh viên GET stats → 403", stats(aid, SB)[0] == 403)
    # ---- 09-18 sau công bố
    kb, ka = k("sv.kha"), k("sv.gioi"); k4 = k("sv04"); gB = [x for x in kb if x[0] == "ASSIGNMENT_GRADED"]; print("   B today:", kb, "| sv04 (chưa có điểm):", k4)
    R.chk("TC-P7-09-18", f"B có ASSIGNMENT_GRADED «{gB[0][1] if gB else None}» href {gB[0][2] if gB else None}; sv04 (chưa công bố) không có", bool(gB) and gB[0][2] == f"/assignments/{aid}" and "điểm" in gB[0][1].lower() and not [x for x in k4 if x[0] == "ASSIGNMENT_GRADED"], (kb, k4))
    # PUBLISH_PENDING còn vì sv... không còn bài REVIEWED: biến mất ≤ 5 s
    t0 = time.time(); gone = False
    while time.time() - t0 < 20:
        if not [x for x in k("teacher") if x[0] == "PUBLISH_PENDING"]: gone = True; break
        time.sleep(0.25)
    dt = time.time() - t0; R.chk("TC-P7-09-18", f"công bố hết → PUBLISH_PENDING biến mất sau {dt:.1f} s (≤ 5 s)", gone and dt <= 5.0, dt)
    T_AP = time.time()
    # ---- 10-02 ngưỡng lý do
    appeal = lambda tk, aid_, body: call("POST", f"{Q}/assignments/{aid_}/appeal", tk, body)
    for n, body, want in (("9 ký tự", {"criteria_ids": ["c2"], "reason": "x" * 9}, "REASON_LENGTH"), ("1.001 ký tự", {"criteria_ids": ["c2"], "reason": "x" * 1001}, "REASON_LENGTH"), ("không tiêu chí", {"criteria_ids": [], "reason": "x" * 20}, None), ("tiêu chí lạ", {"criteria_ids": ["c9"], "reason": "x" * 20}, "CRITERION_UNKNOWN")):
        s, x, _ = appeal(SB, aid, body); R.chk("TC-P7-10-02", f"{n} → {s} {x.get('code') if isinstance(x, dict) else ''}", s == 422 and (want is None or want in json.dumps(x)), x)
    s, x, _ = appeal(SA, aid, {"criteria_ids": ["c1"], "reason": "a" * 1000}); R.chk("TC-P7-10-02", f"1.000 ký tự hợp lệ → {s} 201", s == 201, x)
    s, x, _ = appeal(SB, aid, {"criteria_ids": ["c2"], "reason": "b" * 10}); R.chk("TC-P7-10-02", f"10 ký tự hợp lệ → {s} 201", s == 201, x)
    # ---- 10-08 chuông gộp / today TEACHER
    time.sleep(4)
    tn = psql(f"select string_agg(n.type||'|'||n.title, E'\\n' order by n.created_at) from notifications n join users u on u.id=n.user_id where u.email='teacher@edupilot.local' and n.type='APPEAL_NEW' and greatest(n.created_at, n.updated_at) > to_timestamp({T_AP})"); print("   chuông teacher:", tn)
    ta_ = [x for x in k("teacher") if x[0] == "APPEAL"]; print("   today APPEAL:", ta_)
    R.chk("TC-P7-10-08", f"TEACHER Hôm nay: đúng 1 mục APPEAL «{ta_[0][1] if ta_ else None}»; TA không có", len(ta_) == 1 and "2 yêu cầu xem lại điểm" in ta_[0][1] and not [x for x in k("ta") if x[0] == "APPEAL"], ta_)
    R.chk("TC-P7-10-08", f"chuông gộp TEACHER (không một chuông mỗi phúc khảo): {tn!r}", tn.count("\n") == 0 and "2" in tn.split("|")[-1], tn)
    # ---- 10-10 stats sau ADJUSTED
    apB = psql(f"select id from grade_appeals where student_id=(select id from users where email='sv.kha@edupilot.local') and assignment_id='{aid}'"); vB = psql(f"select version from grade_appeals where id='{apB}'")
    s, x, _ = call("POST", f"{Q}/appeals/{apB}/decide", T, {"version": int(vB), "decision": "ADJUSTED", "response": "Đã xem lại tiêu chí 2, điều chỉnh.", "criteria_scores": [{"id": "c2", "score": "2.50"}]}); print("   decide:", s, str(x)[:150])
    s, st, _ = stats(aid, T); print("   stats:", st)
    R.chk("TC-P7-10-10", f"1 phúc khảo ADJUSTED trên 3 bài công bố: {st}", st["appealed"] == 2 and st["adjusted"] == 1 and str(st["appeal_success_rate"]) in ("0.5", "0.50", "0.5000", "1/2") , st)
    # ---- B today có ASSIGNMENT_APPEAL_REPLY
    time.sleep(3); kb = k("sv.kha"); print("   B today sau quyết:", kb)
    rB_pub = psql(f"select id from grading_results where id='{rB}'"); psql(f"update grading_results set published_at = now() - interval '8 days' where id='{rB}'")
    t0 = time.time(); gone = False; rp = []
    while time.time() - t0 < 30:
        kb2 = k("sv.kha")
        if not [x for x in kb2 if x[0] == "ASSIGNMENT_GRADED"]: gone = True; rp = [x for x in kb2 if x[0] == "ASSIGNMENT_APPEAL_REPLY"]; break
        time.sleep(0.5)
    dtg = time.time() - t0; print("   sau 8 ngày: GRADED biến mất", gone, round(dtg, 1), "s; B today:", kb2)
    R.chk("TC-P7-09-18", f"hết 7 ngày kể từ công bố: ASSIGNMENT_GRADED của B biến mất sau {dtg:.1f} s (≤ 5 s kể từ khi cache nguội; cache 60 s)", gone, kb2)
    R.chk("TC-P7-10-08", f"B nhận ASSIGNMENT_APPEAL_REPLY (bậc 48) sau khi mục GRADED (bậc 43) hết hạn: {rp}", bool(rp) and rp[0][2] == f"/assignments/{aid}", kb2)
    # ---- 10-03 appeal_days = 0 / chưa công bố / giờ máy chủ
    a0 = mkassign(Q, T, "Q103" + code(), appeal_days=0)
    submit(Q, a0, SC, "c.md", ("Bài làm. SUNBURST. " * 12).encode()); call("POST", f"{Q}/assignments/{a0}/grade-all", T, {}); wait_done(a0, 90); time.sleep(1)
    rC = result(a0, "sv.nguyco")["id"]; edit_ = call("PUT", f"{Q}/grading-results/{rC}", T, {"version": ver(rC), "criteria": [{"id": f"c{i+1}", "score": "2"} for i in range(4)], "overall_feedback": "ok"}); review(rC)
    s, x, _ = appeal(SC, a0, {"criteria_ids": ["c1"], "reason": "x" * 20}); R.chk("TC-P7-10-03", f"bài chưa công bố (đã nộp) → {s} {x.get('code') if isinstance(x, dict) else ''}", s == 404 and "ASSIGNMENT_NOT_FOUND" in json.dumps(x), x)
    pub(T, {"result_ids": [rC]}); s, x, _ = appeal(SC, a0, {"criteria_ids": ["c1"], "reason": "x" * 20}); R.chk("TC-P7-10-03", f"appeal_days=0 → {s} {x.get('code') if isinstance(x, dict) else ''}", s == 409 and "APPEAL_DISABLED" in json.dumps(x), x)
    s, x, _ = appeal(SB, aid, {"criteria_ids": ["c3"], "reason": "x" * 20}); R.chk("TC-P7-10-03", f"lần hai (đã trả lời) → {s} {x.get('code') if isinstance(x, dict) else ''}", s == 409 and "APPEAL_ALREADY_FILED" in json.dumps(x), x)
    # giờ máy chủ: header Date lệch + ĐỒNG HỒ trình duyệt không ảnh hưởng: sv.gioi đã nộp; dùng sv04 chưa có công bố → công bố sv04 rồi đặt published_at -7d -1m, gửi kèm Date tương lai
    review(r4); pub(T, {"result_ids": [r4]}); psql(f"update grading_results set published_at = now() - interval '7 days 1 minute' where id='{r4}'")
    s, x, _ = call("POST", f"{Q}/assignments/{aid}/appeal", S4, {"criteria_ids": ["c1"], "reason": "x" * 20}, hdr={"Date": "Wed, 18 Nov 2026 00:00:00 GMT"}); R.chk("TC-P7-10-03", f"quá hạn −7 ngày −1 phút, gửi kèm header Date lùi → {s} {x.get('code') if isinstance(x, dict) else ''} (theo giờ máy chủ)", s == 409 and "APPEAL_WINDOW_CLOSED" in json.dumps(x), x)
    # appeal_days sửa sau công bố
    ad_ = call("GET", f"{Q}/assignments/{aid}", T)[1]; a_ = ad_.get("assignment", ad_); pb = {k_: a_[k_] for k_ in ("title", "description", "max_score", "allow_late", "late_penalty_per_day", "late_max_days", "appeal_days", "channels", "submission_code", "reference_answer", "due_at") if k_ in a_}; pb.update({"appeal_days": 30, "version": a_["version"]})
    s, x, _ = call("PUT", f"{Q}/assignments/{aid}", T, pb); s2, x2, _ = call("POST", f"{Q}/assignments/{aid}/appeal", S4, {"criteria_ids": ["c1"], "reason": "x" * 20})
    print("   appeal_days 7→30 sau công bố:", s, str(x)[:100], "| sv04 gửi (−7d−1m):", s2, str(x2)[:100])
    R.chk("TC-P7-10-03", f"appeal_days đổi 7→30 sau công bố: PUT {s}; sv04 (published −7 ngày −1 phút) gửi → {s2}", (s in (200,) and s2 == 201) or (s == 409), (x, x2))
    # ---- 02-14 gia hạn hạn nộp
    a14 = mkassign(Q, T, "Q214" + code()); submit(Q, a14, SC, "c.md", ("Bài làm. SUNBURST. " * 12).encode())
    psql(f"update assignments set due_at = now() - interval '16 hours' where id='{a14}'"); psql(f"update submissions set submitted_at = now() - interval '8 hours' where assignment_id='{a14}'")
    call("POST", f"{Q}/assignments/{a14}/grade-all", T, {}); wait_done(a14, 90); time.sleep(1); r14 = result(a14, "sv.nguyco"); print("   muộn:", r14["late_days"], r14["late_penalty"])
    R.chk("TC-P7-02-14", f"nộp lúc +8 giờ so với hạn → late_days {r14['late_days']}, trừ {r14['late_penalty']} (1 ngày / 0,50)", r14["late_days"] == 1 and float(r14["late_penalty"]) == 0.5, r14)
    ad_ = call("GET", f"{Q}/assignments/{a14}", T)[1]; a_ = ad_.get("assignment", ad_); keep = a_["due_at"]; pb = {k_: a_[k_] for k_ in ("title", "description", "max_score", "allow_late", "late_penalty_per_day", "late_max_days", "appeal_days", "channels", "submission_code", "reference_answer") if k_ in a_}
    newdue = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 2 * 86400)); s, x, _ = call("PUT", f"{Q}/assignments/{a14}", T, {**pb, "due_at": newdue, "version": a_["version"]}); time.sleep(1.5); r14b = result(a14, "sv.nguyco")
    R.chk("TC-P7-02-14", f"PUT due_at sang +2 ngày ({s}) → late_days {r14b['late_days']}, trừ {r14b['late_penalty']} (0 / 0)", s == 200 and r14b["late_days"] == 0 and float(r14b["late_penalty"]) == 0.0, (s, r14b["late_days"], r14b["late_penalty"]))
    ad_ = call("GET", f"{Q}/assignments/{a14}", T)[1]; a_ = ad_.get("assignment", ad_); s, x, _ = call("PUT", f"{Q}/assignments/{a14}", T, {**pb, "due_at": keep, "version": a_["version"]}); time.sleep(1.5); r14c = result(a14, "sv.nguyco")
    R.chk("TC-P7-02-14", f"đưa hạn về cũ ({s}) → trừ {r14c['late_penalty']} (0,50 lại)", s == 200 and r14c["late_days"] == 1 and float(r14c["late_penalty"]) == 0.5, (r14c["late_days"], r14c["late_penalty"]))
    # ---- 09-16 đồng thời: 10 bài, mỗi vòng 2 yêu cầu công bố song song
    S2 = seed_bt03(Q, T, "Q916" + code(), [f"sv{n:02d}" for n in range(7, 17)]); ok_ = dup_ = bad_ = 0; outb = []
    for acc, rid in S2["res"].items(): review(rid)
    for acc, rid in S2["res"].items():
        out = []
        def go(): out.append(pub(T, {"result_ids": [rid]}))
        th = [threading.Thread(target=go) for _ in range(3)]; [t.start() for t in th]; [t.join() for t in th]; cs = sorted(o[0] for o in out); outb.append(cs)
    n_pub = psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{S2['aid']}' and gr.status='PUBLISHED'"); n_ob = psql(f"select count(*) from outbox where topic='assignment.result_published' and payload->>'result_id' in ({','.join(repr(v) for v in S2['res'].values())})")
    print("   mã trả về mỗi vòng:", outb[:4], "published", n_pub, "outbox", n_ob)
    R.chk("TC-P7-09-16", f"10 vòng × 3 yêu cầu song song: mỗi vòng đúng 1 lần 200 (còn lại 409/skipped); {n_pub} PUBLISHED, {n_ob} sự kiện outbox (=10)", n_pub == "10" and n_ob == "10" and all(c.count(200) >= 1 for c in outb), outb)
sys.exit(R.end())
