import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
def run():
    off(); redis_del("ep:llm:cb:*"); fake(decision_mode="", decision_error_rate=0.0)
    lat0 = []
    for _ in range(4): c, j, ms = post("sv.kha", C); lat0.append(round(ms)); time.sleep(0.3)
    gate('THREADS_PRECHECK', True, [C])
    # D01-17: ngân sách hết
    redis_del("ep:llm:cb:*"); fake(decision_mode=""); sql(f"insert into llm_budgets(scope, course_id, daily_limit) values ('course','{C}',0.00) on conflict (course_id) where scope='course' do update set daily_limit=0.00, version=llm_budgets.version+1"); os.system("docker exec ep-qc-redis-1 redis-cli publish ep:llm:cfg x >/dev/null"); time.sleep(34)
    s = now(); res = [post("sv.kha", C)[0] for _ in range(3)]; d = rows(s)
    sql(f"delete from llm_budgets where course_id='{C}' and scope='course'")
    chk5("TC-D01-17", any(x[2] == "budget_blocked" for x in d) and all(x[1] in ("FALLBACK", "FAIL_CLOSED") for x in d if x[2] == "budget_blocked"), f"ngân sách ngày lớp QD105A = 0 (llm_budgets; chờ 34 s cache hạn mức 30 s): 3 bài đăng → DECIDE {[(x[1], x[2]) for x in d]}; POST {res}; đã xoá ngân sách")
    off()
with dwin(): run()
