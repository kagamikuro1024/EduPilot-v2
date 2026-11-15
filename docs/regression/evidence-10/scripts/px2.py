from pa1_lib import *
def run(q): return subprocess.run(["docker", "exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], capture_output=True, text=True)
sid = sql("select id from users where email='sv.gioi@edupilot.local'")[0][0]
ins = lambda s, t, c=C1: f"insert into practice_attempts(course_id, student_id, mode, topic, item_count) values ('{c}','{s}','TOPIC','{t}',5);"
moi = sql("select id from users where email='sv.moi@edupilot.local'")[0][0]
r = run(f"begin; {ins(moi, 'x')} rollback;")
ok("TC-01-03 SV chưa vào lớp → 23503", "23503" in r.stderr or "foreign key" in r.stderr.lower(), r.stderr[:150])
r = run(f"begin; {ins(sid, 'QCU1')} {ins(sid, 'QCU1')} rollback;"); r2 = run(f"begin; {ins(sid, 'QCU1')} {ins(sid, 'QCU2')} rollback;")
ok("TC-01-06 trùng lượt dở cùng chủ đề → 23505; khác chủ đề được", "23505" in r.stderr and "ERROR" not in r2.stderr, f"{r.stderr[:90]} | {r2.stderr[:60]}")
