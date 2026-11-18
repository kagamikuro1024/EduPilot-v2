#!/usr/bin/env python3
"""TC-P7-09-09: SMTP chết lúc công bố (dừng Mailpit) → công bố vẫn thành công; thư gửi lại sau khi SMTP hồi phục; replay outbox không trùng chuông/thư."""
import sys, os, urllib.request; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("09m"); ALL = ["sv.gioi", "sv.kha", "sv.nguyco"]
C = mkcourse("QD9N" + str(int(time.time()) % 1000), ALL); Q = f"/api/v1/courses/{C}"; T = tok("teacher")
ver = lambda rid: int(psql(f"select version from grading_results where id='{rid}'"))
with stub_ctx():
    S = seed_bt03(Q, T, "Q9M" + str(int(time.time()) % 100000), ["sv.gioi", "sv.kha"])
rids = list(S["res"].values())
for r in rids:
    call("POST", f"{Q}/grading-results/{r}/review", T, {"version": ver(r)})
print("   REVIEWED:", psql(f"select status from grading_results where id in ({','.join(repr(r) for r in rids)})").replace("\n", ","))
mail = lambda: json.loads(urllib.request.urlopen("http://localhost:8525/api/v1/messages?limit=200").read())
def mails(addr): return sum(1 for m in mail()["messages"] if any(addr in t["Address"] for t in m["To"]) and "Bài tập" in m["Subject"])
users = [f"{a}@edupilot.local" for a in S["res"]]
n0 = {u: mails(u) for u in users}
while subprocess.run(["mkdir", "/tmp/qc-global.lock"], capture_output=True).returncode: time.sleep(5)
try:
    subprocess.run(["docker", "stop", "ep-qc-mailpit-1"], capture_output=True, check=True)
    s, x, _ = call("POST", f"{Q}/grading-results/publish", T, {"result_ids": rids}, idem()); print("   publish khi SMTP chết:", s, str(x)[:100])
    R.chk("TC-P7-09-09", f"công bố khi Mailpit dừng → 200, published {x.get('published') if isinstance(x, dict) else x}", s == 200 and x.get("published") == 2, (s, x))
    time.sleep(25)
    st = psql(f"select status||':'||attempts from mail_outbox where dedupe_key like 'assignment.published:%' and dedupe_key like any (array[{','.join(repr('%'+r+'%') for r in rids)}])").replace("\n", " ")
    bell = psql(f"select count(*) from notifications where type='ASSIGNMENT_GRADED' and dedupe_key like any (array[{','.join(repr('%'+r+'%') for r in rids)}])")
    st_res = psql(f"select status from grading_results where id in ({','.join(repr(r) for r in rids)})").replace("\n", ",")
    print("   khi SMTP chết: kết quả", st_res, "| mail_outbox", st, "| chuông", bell)
    R.chk("TC-P7-09-09", f"SMTP chết: kết quả {st_res}; thư chưa gửi (mail_outbox: {st}); chuông trong ứng dụng vẫn có ({bell})", "PUBLISHED" in st_res and "SENT" not in st, (st, bell, st_res))
finally:
    subprocess.run(["docker", "start", "ep-qc-mailpit-1"], capture_output=True)
    for _ in range(30):
        try: mail(); break
        except Exception: time.sleep(2)
    subprocess.run(["rmdir", "/tmp/qc-global.lock"], capture_output=True)
for _ in range(60):
    sent = psql(f"select count(*) from mail_outbox where status='SENT' and dedupe_key like any (array[{','.join(repr('%'+r+'%') for r in rids)}])")
    if sent == "2": break
    time.sleep(3)
n1 = {u: mails(u) for u in users}; print("   sau hồi phục: SENT", sent, "thư mới:", {u: n1[u] - n0[u] for u in users})
R.chk("TC-P7-09-09", f"Mailpit bật lại → thư gửi lại: mail_outbox SENT {sent}/2, thư mới tới Mailpit {[n1[u] - n0[u] for u in users]}", sent == "2" and all(n1[u] - n0[u] == 1 for u in users), (sent, n0, n1))
# replay outbox
b0 = psql(f"select count(*) from notifications where type='ASSIGNMENT_GRADED' and dedupe_key like any (array[{','.join(repr('%'+r+'%') for r in rids)}])"); m0 = psql(f"select count(*) from mail_outbox where dedupe_key like any (array[{','.join(repr('%'+r+'%') for r in rids)}])")
psql(f"update outbox set dispatched_at=null, enqueued_at=null, next_attempt_at=now(), attempts=0 where topic like 'assignment.result_published%' and payload::text like any (array[{','.join(repr('%'+r+'%') for r in rids)}])"); time.sleep(25)
b1 = psql(f"select count(*) from notifications where type='ASSIGNMENT_GRADED' and dedupe_key like any (array[{','.join(repr('%'+r+'%') for r in rids)}])"); m1 = psql(f"select count(*) from mail_outbox where dedupe_key like any (array[{','.join(repr('%'+r+'%') for r in rids)}])")
n2 = {u: mails(u) for u in users}; print("   replay: chuông", b0, "→", b1, "| mail_outbox", m0, "→", m1, "| thư", {u: n2[u] - n1[u] for u in users})
R.chk("TC-P7-09-09", f"replay outbox (dispatched_at=null): chuông {b0}→{b1}, mail_outbox {m0}→{m1}, thư mới {[n2[u]-n1[u] for u in users]} (không trùng)", b0 == b1 and m0 == m1 and all(n2[u] == n1[u] for u in users), (b0, b1, m0, m1))
sys.exit(R.end())
