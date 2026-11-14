import sys, time
from dmx import *
C1 = course("sv.kha", "761987")
step = sys.argv[1]
if step == "mk":
    set_gate("THREADS_PRECHECK"); open("/tmp/stub_mode", "w").write("allow")
    c, j, _ = api("sv.gioi", "POST", f"/courses/{C1}/threads", {"title": "Hỏi về lịch học", "body": "Buổi học tuần sau có thay đổi phòng học so với thông báo trước không ạ?"})
    tid = (j.get("thread") or j).get("id"); print("thread", c, tid); open("/tmp/tid", "w").write(str(tid))
if step == "cm":
    tid = open("/tmp/tid").read()
    for body in ("Mình nghe nói có một bạn trong lớp hoàn cảnh khó khăn nên bị trừ điểm chuyên cần, không biết bạn đó có bị ảnh hưởng điểm cuối kỳ không?", "Cảm ơn bạn, mình sẽ kiểm tra lại lịch học."):
        s = now_db(); c, j, ms = api("sv.kha", "POST", f"/courses/{C1}/threads/{tid}/comments", {"body": body}); time.sleep(4)
        print("comment", c, (j or {}).get("code"), f"{ms:.0f}ms", [(r[1], r[2], r[3], r[4]) for r in sql(f"select created_at::time(2),task,status,coalesce(cascade_gate,''),coalesce(cascade_path,'') from llm_audit where created_at>='{s}' and task in ('DECIDE','CLASSIFY')")])
