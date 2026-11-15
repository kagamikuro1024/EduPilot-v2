# QC sprint 11 — US-P10-03 cảnh báo (worker ep-qc, OBS_ALERT_TICK=20s). Ghi DB: llm_audit tổng hợp (trace_id 'qc-synth-*') + dời hàng thật 3 giờ (khôi phục cuối), llm_budgets, notifications.
import sys
from p10lib import *
TYPES = "('LLM_ERROR_RATE','LLM_BUDGET_REACHED')"
c1, c2 = course("teacher", "761987"), course("teacher", "761988")
ADM = [r[0] for r in sql("select id from users where role='ADMIN' and status='ACTIVE'")]
moved = []

def isolate():
    global moved
    moved += [r[0] for r in sql("update llm_audit set created_at = created_at - interval '3 hours' where created_at > now() - interval '70 minutes' and trace_id not like 'qc-synth%' returning id")]

def restore():
    if moved:
        sql("update llm_audit set created_at = created_at + interval '3 hours' where id in (%s)" % ",".join(f"'{i}'" for i in moved))

def clean():
    sql("delete from llm_audit where trace_id like 'qc-synth%'"); sql(f"delete from notifications where type in {TYPES}")

def rows(n, status="ok", course=None, at="now() - interval '5 minutes'", cost=0):
    cid = f"'{course}'" if course else "null"
    sql("insert into llm_audit(task,lane,provider,model,latency_ms,status,trace_id,created_at,course_id,cost_est) select 'CHAT','INTERACTIVE','fake','fake-chat',100,'%s','qc-synth-'||g,%s,%s,%s from generate_series(1,%d) g" % (status, at, cid, cost, n))

def notes(typ):
    return int(sql(f"select count(*) from notifications where type='{typ}'")[0][0])

def wait_note(typ, want=True, t=60):
    end = time.time() + t
    while time.time() < end:
        if notes(typ) > 0 and want: return True
        time.sleep(3)
    return notes(typ) > 0

def case(tc, build, expect, typ="LLM_ERROR_RATE", per_admin=True):
    clean(); isolate(); build(); got = wait_note(typ, expect, 60 if expect else 50)
    n = notes(typ)
    ok(tc, (n == len(ADM) if per_admin else n > 0) if expect else n == 0, f"thông báo={n} admin={len(ADM)}")
    return n

isolate()
try:
    case("TC-03-01 30 yêu cầu, 3 lỗi (10%) → 1 thông báo / ADMIN", lambda: (rows(27), rows(3, "error")), True)
    l = sql(f"select link from notifications where type='LLM_ERROR_RATE' limit 1")[0][0]
    ok("TC-03-01b liên kết /observability + lọc Có vấn đề", "/observability" in l and "problem" in l, l)
    time.sleep(45)
    ok("TC-03-02 chạy lại ≥ 2 tick cùng cửa sổ → vẫn 1 thông báo / ADMIN", notes("LLM_ERROR_RATE") == len(ADM), str(notes("LLM_ERROR_RATE")))
    case("TC-03-03a 30 yêu cầu, 1 lỗi (3,3%) → không", lambda: (rows(29), rows(1, "error")), False)
    case("TC-03-03b 30 yêu cầu, 2 lỗi (6,7%) → có", lambda: (rows(28), rows(2, "error")), True)
    case("TC-03-03c 40 yêu cầu, 2 lỗi (đúng 5,0%) → không", lambda: (rows(38), rows(2, "error")), False)
    case("TC-03-04 cancelled + degraded không tính lỗi → không", lambda: (rows(24), rows(3, "cancelled"), rows(3, "degraded")), False)
    case("TC-03-05a 19 yêu cầu, 100% lỗi → không", lambda: rows(19, "error"), False)
    case("TC-03-05b 20 yêu cầu, 100% lỗi → có", lambda: rows(20, "error"), True)
    # dedupe 03-08/09/10
    clean(); isolate(); rows(27); rows(3, "error"); wait_note("LLM_ERROR_RATE")
    sql("update notifications set created_at = now() - interval '5 hours 59 minutes' where type='LLM_ERROR_RATE'"); time.sleep(45)
    ok("TC-03-08 +5h59 không thêm", notes("LLM_ERROR_RATE") == len(ADM), str(notes("LLM_ERROR_RATE")))
    sql("update notifications set created_at = now() - interval '6 hours 1 minutes' where type='LLM_ERROR_RATE'")
    t0 = time.time()
    while time.time() - t0 < 60 and notes("LLM_ERROR_RATE") <= len(ADM): time.sleep(3)
    ok("TC-03-09 sau > 6 giờ tái phát → thông báo mới", int(sql("select count(*) from notifications where type='LLM_ERROR_RATE' and created_at > now() - interval '5 minutes'")[0][0]) == len(ADM), str(notes("LLM_ERROR_RATE")))
    # budget
    clean(); st = {}
    def day_spent(where=""):
        return float(sql("select coalesce(sum(cost_est),0) from llm_audit where (created_at at time zone 'Asia/Ho_Chi_Minh')::date = (now() at time zone 'Asia/Ho_Chi_Minh')::date " + where)[0][0])
    def b_sys():
        spent = day_spent(); st["X"] = max(1000, int(spent) + 10); rows(1, cost=round(st["X"] - spent, 4))
        sql("delete from llm_budgets where scope='system'"); sql(f"insert into llm_budgets(scope, daily_limit) values ('system', {st['X']})")
    case("TC-03-06a chi phí ngày = hạn mức hệ thống → LLM_BUDGET_REACHED «toàn hệ thống»", b_sys, True, "LLM_BUDGET_REACHED")
    X = st["X"]
    b = sql("select body from notifications where type='LLM_BUDGET_REACHED' limit 1")[0][0]
    ok("TC-03-06b nội dung nêu «toàn hệ thống»", "toàn hệ thống" in b, b)
    sql(f"update llm_budgets set daily_limit = {round(X / 0.99 + 0.01, 2)} where scope='system'"); sql(f"delete from notifications where type in {TYPES}"); time.sleep(50)
    ok("TC-03-06c 0,99·X → không", notes("LLM_BUDGET_REACHED") == 0, str(notes("LLM_BUDGET_REACHED")))
    sql("delete from llm_budgets where scope='system'"); sql(f"delete from notifications where type in {TYPES}")
    s2 = float(sql(f"select coalesce(sum(cost_est),0) from llm_audit where course_id='{c2}' and (created_at at time zone 'Asia/Ho_Chi_Minh')::date = (now() at time zone 'Asia/Ho_Chi_Minh')::date")[0][0])
    L = s2 + 300; sql(f"insert into llm_budgets(scope, course_id, daily_limit) values ('course','{c2}', {L})")
    rows(1, course=c2, at="(date_trunc('day', now() at time zone 'Asia/Ho_Chi_Minh') at time zone 'Asia/Ho_Chi_Minh') - interval '30 minutes'", cost=5000)
    time.sleep(50)
    ok("TC-03-07a dòng 23:30 VN hôm qua không tính vào hôm nay → không thông báo", notes("LLM_BUDGET_REACHED") == 0, str(notes("LLM_BUDGET_REACHED")))
    rows(1, course=c2, at="(date_trunc('day', now() at time zone 'Asia/Ho_Chi_Minh') at time zone 'Asia/Ho_Chi_Minh') + interval '30 minutes'", cost=300)
    ok("TC-03-07b dòng 00:30 VN hôm nay tính vào hôm nay → có, nêu lớp 761988", wait_note("LLM_BUDGET_REACHED") and "761988" in sql("select body from notifications where type='LLM_BUDGET_REACHED' limit 1")[0][0], "")
    # 03-10 phạm vi khác: thêm hạn mức lớp 761987 hạ thấp → thông báo riêng
    s1 = float(sql(f"select coalesce(sum(cost_est),0) from llm_audit where course_id='{c1}' and (created_at at time zone 'Asia/Ho_Chi_Minh')::date = (now() at time zone 'Asia/Ho_Chi_Minh')::date")[0][0])
    sql(f"insert into llm_budgets(scope, course_id, daily_limit) values ('course','{c1}', {max(1, round(s1, 2) - 0.01 if s1 > 0.02 else 0.01)})")
    rows(1, course=c1, cost=1)
    time.sleep(50)
    ok("TC-03-10 phạm vi lớp khác cùng loại → thông báo riêng (2 phạm vi)", int(sql("select count(distinct dedupe_key) from notifications where type='LLM_BUDGET_REACHED'")[0][0]) >= 2, str(sql("select count(*) from notifications where type='LLM_BUDGET_REACHED'")))
    # 03-12 vai khác + 03-14 hai ADMIN
    clean(); isolate(); rows(27); rows(3, "error"); wait_note("LLM_ERROR_RATE")
    # 03-14: hai ADMIN
    for u in ("teacher", "ta", "sv.kha"):
        c, j, _, _ = api(u, "GET", "/notifications?limit=50")
        ok(f"TC-03-12 {u} không thấy cảnh báo AI", c == 200 and not any(n["type"] in ("LLM_ERROR_RATE", "LLM_BUDGET_REACHED") for n in j["items"]), str(c))
    c, j, _, _ = api("admin", "GET", "/notifications?limit=50"); mine = [n for n in j["items"] if n["type"] == "LLM_ERROR_RATE"]
    ok("TC-03-01c admin thấy ở chuông", len(mine) == 1, str(len(mine)))
    clean(); isolate(); sql("update users set role='ADMIN' where email='ta@edupilot.local'")
    ADM[:] = [r[0] for r in sql("select id from users where role='ADMIN' and status='ACTIVE'")]
    rows(27); rows(3, "error"); wait_note("LLM_ERROR_RATE"); time.sleep(5)
    ok("TC-03-14a hai ADMIN đều nhận", notes("LLM_ERROR_RATE") == 2 and len(ADM) == 2, f"{notes('LLM_ERROR_RATE')} admin={len(ADM)}")
    c, j, _, _ = api("ta", "GET", "/notifications?limit=50"); nid = [n for n in j["items"] if n["type"] == "LLM_ERROR_RATE"][0]["id"]
    api("ta", "POST", f"/notifications/{nid}/read", {})
    c, j2, _, _ = api("admin", "GET", "/notifications?limit=50")
    ok("TC-03-14b một người đọc không ảnh hưởng người kia", [n for n in j2["items"] if n["type"] == "LLM_ERROR_RATE"][0].get("read_at") in (None, False), str(c))
finally:
    sql("update users set role='TA' where email='ta@edupilot.local'")
    clean(); sql("delete from llm_budgets where scope in ('system','course')"); restore()
    print("restored", len(moved))
