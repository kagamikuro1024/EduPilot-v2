import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def run():
    # ===== G02-05: cờ tắt → ghim 0, prompt không khối =====
    a = mk_assignment("g0205", 3); flag("GRADING_GUIDANCE", False); stub("allow"); t0 = time.time()
    c, j, nn = grade_all_wait(a); rs = results(a); P = [p for p in stub_prompts(t0) if not p[3]]
    chk("TC-G02-05", all(r[4] == "0" for r in rs) and len(rs) == 3 and P and not any(p[1] for p in P), f"cờ tắt, chấm 3 bài: guidance_version kết quả = {sorted({r[4] for r in rs})}; {len(P)} lời gọi chấm tới stub, có khối «Lưu ý của giảng viên» = {sum(1 for p in P if p[1])}")
    # ===== G02-16: UPDATE guidance_version khi RUNNING cho phép, sau DONE chặn =====
    rid = rs[0][0]
    sql(f"update grading_results set run_state='RUNNING', lease_owner='qc', lease_until=now()+interval '10 minutes' where id='{rid}'")
    try: sql(f"update grading_results set guidance_version=1 where id='{rid}'"); okr = True
    except Exception as e: okr = False; err1 = str(e)[:120]
    sql(f"update grading_results set run_state='DONE', lease_owner=null, lease_until=null where id='{rid}'")
    try: sql(f"update grading_results set guidance_version=2 where id='{rid}'"); blocked = False; err2 = ""
    except Exception as e: blocked = True; err2 = str(e).strip().replace("\n", " ")[:160]
    v = sql(f"select guidance_version, run_state from grading_results where id='{rid}'")[0]
    chk("TC-G02-16", okr and blocked and v[0] == "1", f"RUNNING: UPDATE guidance_version 0→1 {'cho phép' if okr else 'bị chặn'}; sau DONE: UPDATE →2 {'bị chặn: ' + err2 if blocked else 'KHÔNG bị chặn'}; cuối cùng guidance_version={v[0]} run_state={v[1]}")
    # ===== G02-06: đổi bản giữa lượt 1 và lượt 2 =====
    a, ids, info = guided("g0206", 12, 8, 2); print("guided", info)
    flag("GRADING_GUIDANCE", True)
    sql(f"""create temp table tn as select * from submissions where assignment_id='{a}' order by created_at limit 1;
update tn set id=uuidv7(), sha256=md5(random()::text)||md5(random()::text), blob_key='qc106/'||md5(random()::text), external_ref=null, student_id=(select user_id from enrollments e where e.course_id=tn.course_id and e.role_in_course='STUDENT' and e.status='ACTIVE' and not exists (select 1 from submissions x where x.assignment_id=tn.assignment_id and x.student_id=e.user_id) limit 1);
insert into submissions select * from tn;""")
    more = [x["id"] for x in items_of(a, "DRAFT")]; [decide(a, i) for i in more[:2]]
    pend = len(items_of(a, "APPROVED")); stub("slow"); t0 = time.time()
    c, j, _ = api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/grade-all", {}); print("grade-all", c, lastj(j, 80))
    time.sleep(3); cA, jA, _ = apply_v(a, 1); print("apply trong lúc chấm", cA, lastj(jA, 120))
    for _ in range(90):
        r = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.run_state='DONE'")[0][0]
        if int(r) == 13: break
        time.sleep(2)
    stub("allow"); rs = results(a); newr = [x for x in rs if x[4] != "0"]; gv = sql(f"select guidance_version from assignments where id='{a}'")[0][0]
    P = [p for p in stub_prompts(t0) if not p[3] and p[1]]
    vers = sorted({p[2] for p in P}); lastres = sql(f"select r.guidance_version, r.round, r.attempts from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' order by r.created_at desc limit 1")[0]
    chk("TC-G02-06", cA == 200 and gv == "2" and lastres[0] == "1" and vers == ["1"], f"mục chờ áp dụng khi bắt đầu: {pend}; apply v1→v2 trong lúc chấm (stub chậm) → {cA}; assignments.guidance_version={gv}; kết quả bài mới: guidance_version={lastres[0]} round={lastres[1]} attempts={lastres[2]}; mọi {len(P)} lời gọi chấm của bài mới dùng khối bản {vers} (không trộn)")
with window(): run()
