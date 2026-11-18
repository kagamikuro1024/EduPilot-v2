import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("02x"); T = tok("teacher")
a14 = psql("select a.id from assignments a where a.submission_code like 'Q214%' order by a.created_at desc limit 1"); C = psql(f"select course_id from assignments where id='{a14}'"); Q = f"/api/v1/courses/{C}"
rd = lambda: psql(f"select gr.late_days||'|'||gr.late_penalty from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a14}'")
def putdue(due):
    ad_ = call("GET", f"{Q}/assignments/{a14}", T)[1]; a_ = ad_.get("assignment", ad_); pb = {k_: a_[k_] for k_ in ("title", "description", "max_score", "allow_late", "late_penalty_per_day", "late_max_days", "appeal_days", "channels", "submission_code", "reference_answer") if k_ in a_}
    return call("PUT", f"{Q}/assignments/{a14}", T, {**pb, "due_at": due, "version": a_["version"]})
old = psql(f"select to_char(due_at at time zone 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"') from assignments where id='{a14}'"); print("   hạn cũ", old, "trước:", rd())
s, x, _ = putdue(time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 2 * 86400))); t0 = time.time(); seq = []
for _ in range(60): seq.append((round(time.time() - t0, 2), rd())); time.sleep(0.05)
print("   sau gia hạn:", s, seq[:3], seq[-1])
R.chk("TC-P7-02-14", f"gia hạn hạn nộp +2 ngày ({s}): sau 3 s đọc DB late_days|late_penalty = {seq[-1][1]} (kỳ vọng 0|0.00); lần đầu thấy {next((t for t, v in seq if v.startswith('0|')), None)}", s == 200 and seq[-1][1].startswith("0|0"), seq)
s, x, _ = putdue(old); time.sleep(2); R.chk("TC-P7-02-14", f"đưa hạn về cũ ({s}) → {rd()} (1|0.50)", rd().startswith("1|0.5"), rd())
sys.exit(R.end())
