"""D05-14: các câu routeCases (agent_test.go) qua chat THẬT, cổng CHAT_ROUTING tắt rồi bật (decision thật); so khối trả lời với kỳ vọng theo intent. Chỉ chạy trên qc2seedt3."""
import sys, os, json, time, importlib.util, concurrent.futures as cf
os.environ.setdefault("API_URL", "https://localhost:4743/api/v1")
spec = importlib.util.spec_from_file_location("run", os.path.expanduser("~/checkouts/qc2seedt3/benchmarks/decision/run.py")); R = importlib.util.module_from_spec(spec); spec.loader.exec_module(R)
W = os.path.expanduser("~/.config/edupilot-team/qc3-work/")
cases = json.load(open(W + "route61.json"))
extra = [(r["intent"], r["text"]) for r in R.jsonl("chat.jsonl") if r["stratum"] == "paraphrase"]
KIND = {"Attendance": "attendance", "Participation": "participation", "Grade": "grade_summary", "ExamSchedule": "exam_schedule", "Upcoming": "upcoming_events", "Library": "library_results", "WhatIf": "what_if"}
PK = {"PERSONAL_ATTENDANCE": "attendance", "PERSONAL_PARTICIPATION": "participation", "PERSONAL_GRADE": "grade_summary", "EXAM_SCHEDULE": "exam_schedule", "UPCOMING_EVENTS": "upcoming_events", "LIBRARY_SEARCH": "library_results"}
items = [("route", i, q, KIND.get(i)) for i, q in cases] + [("paraphrase", i, q, PK.get(i)) for i, q in extra]
c1 = R.course_id(); USERS = ["sv.gioi", "sv.kha", "sv.nguyco"]
def run_mode(on):
    R.set_mode(c1, on); o0 = R.obs(); out = [None] * len(items)
    def work(u):
        tk = R.token(u)
        for n in range(u_idx[u], len(items), len(USERS)):
            r = R.ask(tk, c1, items[n][2]); out[n] = r
    u_idx = {u: k for k, u in enumerate(USERS)}
    with cf.ThreadPoolExecutor(len(USERS)) as ex: list(ex.map(work, USERS))
    o1 = R.obs(); return out, R.path_delta(o0, o1, "CHAT_ROUTING")
res = {}
for on in (False, True):
    out, d = run_mode(on); res["on" if on else "off"] = {"out": out, "paths": d}; print("xong", "bật" if on else "tắt", d, flush=True)
R.set_mode(c1, False)
json.dump({"items": items, "res": res}, open(W + "d0514.json", "w"), ensure_ascii=False)
def ok(r, exp, kind): 
    if "error" in r: return None
    return (exp in r["kinds"]) if exp else None
for grp in ("route", "paraphrase"):
    for m in ("off", "on"):
        rows = [(it, r) for it, r in zip(items, res[m]["out"]) if it[0] == grp and it[3]]
        n = sum(1 for it, r in rows if "error" not in r); c = sum(1 for it, r in rows if ok(r, it[3], None))
        print(f"{grp} {m}: đúng {c}/{len(rows)} (có kỳ vọng khối), lỗi {sum(1 for it,r in rows if 'error' in r)}")
diff = [(it[2][:50], it[1], res['off']['out'][k].get('kinds'), res['on']['out'][k].get('kinds')) for k, it in enumerate(items) if res['off']['out'][k].get('kinds') != res['on']['out'][k].get('kinds')]
print("khác giữa tắt / bật:", len(diff)); [print("  ", d) for d in diff[:20]]
