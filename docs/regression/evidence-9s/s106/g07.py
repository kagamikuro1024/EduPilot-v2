import sys, os, statistics; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
TH = f"/courses/{CG0}/threads"
def cases(kind=None): return sql(f"select kind, source_id, criterion_id from grading_regression_cases where course_id='{CG0}'" + (f" and kind='{kind}'" if kind else ""))
def run():
    # bật REGRESSION_CAPTURE (chỉ ADMIN bật được)
    cc, j, _ = api("admin", "GET", FL()); f = [x for x in j["items"] if x["feature"] == "REGRESSION_CAPTURE"][0]
    print("REGRESSION_CAPTURE", api("admin", "PUT", f"{FL()}/REGRESSION_CAPTURE", {"enabled": True, "version": f["version"]})[0], "teacher tự bật:", api("teacher", "PUT", f"{FL()}/REGRESSION_CAPTURE", {"enabled": False, "version": f["version"]})[0])
    flag("GRADING_GUIDANCE", True)
    # ===== G07-02 =====
    a = sql("select a.id from assignments a where a.title='QC106 g0510' order by created_at desc limit 1")[0][0]
    r = sql(f"select r.id, u.email from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id join users u on u.id=s.student_id where s.assignment_id='{a}' and r.status='PUBLISHED' limit 1")[0]
    user = r[1].split("@")[0]; print("SV", user, "kq", r[0]); sql(f"update assignments set published_at=now() where id='{a}' and published_at is null")
    c, j, _ = api(user, "POST", f"/courses/{CG0}/assignments/{a}/appeal", {"criteria_ids": ["c2"], "reason": "Em xin xem lại tiêu chí phân tích tấn công vì bài có nêu đủ các bước."}); print("appeal", c, lastj(j, 160))
    cl, jl, _ = api("teacher", "GET", f"/courses/{CG0}/appeals?status=OPEN"); ap = [x for x in (jl.get("items") or []) if x.get("result_id") == r[0] or x.get("id") == (j.get("appeal") or {}).get("id")]
    apid = (j.get("appeal") or {}).get("id"); c2, d, _ = api("teacher", "GET", f"/courses/{CG0}/appeals/{apid}"); print("detail", c2, lastj(d, 200))
    ver = (d.get("appeal") or d).get("version", 1)
    c3, j3, _ = api("teacher", "POST", f"/courses/{CG0}/appeals/{apid}/decide", {"decision": "ADJUSTED", "response": "Đã xem lại: bài nêu đủ chuỗi bước nên tiêu chí này được nâng lên mức khá.", "criteria_scores": [{"id": "c2", "score": "2.00"}], "version": ver}); print("decide", c3, lastj(j3, 160))
    ap_cases = cases("GRADING_APPEAL")
    # thread: 3 bài AI PENDING chèn SQL (kênh công khai) → sửa / từ chối / duyệt
    ps = []
    for i in range(3):
        c0, j0, _ = api("sv.kha", "POST", TH, {"title": f"QC106 gợi ý G07-02 số {i} {int(time.time())}", "body": f"Em hỏi về cách làm phần {i} của bài tập 03 ạ, em chưa rõ yêu cầu của tiêu chí."}); t_ = j0["thread"]["id"]
        sql(f"insert into forum_posts (course_id, thread_id, kind, body, verification_state, citations, confidence) select '{CG0}', '{t_}', 'AI', 'Trả lời mẫu QC106 số {i}', 'PENDING', '[]', 0.60 where not exists (select 1 from forum_posts where thread_id='{t_}' and kind='AI')")
        ps.append(sql(f"select id from forum_posts where thread_id='{t_}' and kind='AI'")[0][0])
    vers = {p: sql(f"select version from forum_posts where id='{p}'")[0][0] for p in ps}
    cc1 = api("teacher", "POST", f"/courses/{CG0}/posts/{ps[0]}/correct", {"body": "Nội dung đã sửa bởi giảng viên QC106.", "version": int(vers[ps[0]])})
    cc2 = api("teacher", "POST", f"/courses/{CG0}/posts/{ps[1]}/reject", {})
    cc3 = api("teacher", "POST", f"/courses/{CG0}/posts/{ps[2]}/verify", {})
    print("thread ops", cc1[0], cc2[0], cc3[0], lastj(cc1[1], 100), lastj(cc2[1], 100))
    tc = cases(); tcs = {(x[0], x[1]) for x in tc}
    ok_corr = ("THREAD_CORRECTED", ps[0]) in tcs; ok_rej = ("THREAD_REJECTED", ps[1]) in tcs; ok_ver = not any(x[1] == ps[2] for x in tc)
    ok_ap = any(x[1] == r[0] and x[2] == "c2" for x in ap_cases)
    chk("TC-G07-02", c3 == 200 and ok_ap and ok_corr and ok_rej and ok_ver, f"cờ REGRESSION_CAPTURE bật (admin 200; teacher tự bật → 403?): phúc khảo ADJUSTED c2 → {c3}: ca GRADING_APPEAL source_id=kết quả, criterion=c2: {ok_ap}; bài AI sửa → {cc1[0]}: THREAD_CORRECTED source_id=bài {ok_corr}; từ chối → {cc2[0]}: THREAD_REJECTED {ok_rej}; xác nhận (verify) → {cc3[0]}: không ca {ok_ver}")
with window(): run()
