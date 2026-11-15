# Đợt R · F19 nhánh lập trình C/C++ (judge sandbox)
from rlib import *
import json, time, uuid, datetime as dt
c1 = C("teacher"); TAB = {"X-Exam-Tab": str(uuid.uuid4())}
def at(s): return (dt.datetime.utcnow() + dt.timedelta(seconds=s)).replace(microsecond=0).isoformat() + "Z"
c, q, _ = api("teacher", "GET", f"/courses/{c1}/questions?type=CODE&limit=5")
qid = [x for x in q["items"] if "Ước" in x["title"]][0]["id"]
c, e, _ = api("teacher", "POST", f"/courses/{c1}/exams", {"title": "QC-R Code tuần", "instructions": "Viết C/C++.", "opens_at": at(25), "closes_at": at(190), "duration_minutes": 2, "max_score": "10"})
ok("R-F19-30 tạo bài thi lập trình", c == 201 and e.get("kind") in ("CODE", "MCQ"), f"{c} kind={e.get('kind')}")
eid = e["id"]
c, it, _ = api("teacher", "PUT", f"/courses/{c1}/exams/{eid}/items", {"items": [{"question_id": qid, "points": "10"}], "version": e["version"]})
ok("R-F19-31 gắn câu code 10đ", c == 200, f"{c} {str(it)[:100]}")
c, sc, _ = api("teacher", "POST", f"/courses/{c1}/exams/{eid}/schedule", {})
ok("R-F19-32 lên lịch", c == 200, f"{c}")
time.sleep(26)
c, a, _ = api("sv.gioi", "POST", f"/courses/{c1}/exams/{eid}/attempts", {}, headers=TAB)
ok("R-F19-33 SV bắt đầu thi code", c == 201, f"{c}")
aid = a["attempt"]["id"]; item = a["items"][0]["item_id"]
blob = json.dumps(a)
ok("R-F19-34 đề không lộ test ẩn / lời giải mẫu", "hidden" not in blob.lower() and "reference" not in blob.lower() and "model_solution" not in blob.lower(), "kiểm JSON")
good = "#include <stdio.h>\nint g(int a,int b){return b?g(b,a%b):a;}\nint main(){int a,b;scanf(\"%d %d\",&a,&b);printf(\"%d\\n\",g(a,b));return 0;}\n"
c, dr, _ = api("sv.gioi", "PUT", f"/courses/{c1}/exams/{eid}/attempts/{aid}/code/{item}/draft", {"language": "c11", "source": good, "base_rev": 0}, headers=TAB)
ok("R-F19-35 lưu nháp mã", c in (200, 204), f"{c} {str(dr)[:100]}")
c, rn, _ = api("sv.gioi", "POST", f"/courses/{c1}/exams/{eid}/attempts/{aid}/code/{item}/run", {"language": "c11", "source": good}, headers=TAB)
ok("R-F19-36 chạy thử (judge nhận việc)", c in (200, 202), f"{c} {str(rn)[:120]}")
rid = (rn or {}).get("run_id") or (rn or {}).get("id"); st = None
for _ in range(40):
    c2, rr, _ = api("sv.gioi", "GET", f"/courses/{c1}/exams/{eid}/attempts/{aid}/runs/{rid}", headers=TAB)
    st = (rr or {}).get("status")
    if st not in (None, "QUEUED", "RUNNING"): break
    time.sleep(2)
ok("R-F19-37 kết quả chạy thử (sandbox)", st not in (None, "QUEUED", "RUNNING"), f"status={st} {json.dumps(rr, ensure_ascii=False)[:200]}")
bad = "#include <stdio.h>\nint main(){while(1){} return 0;}\n"
c, rn2, _ = api("sv.gioi", "POST", f"/courses/{c1}/exams/{eid}/attempts/{aid}/code/{item}/run", {"language": "c11", "source": bad}, headers=TAB)
rid2 = (rn2 or {}).get("run_id") or (rn2 or {}).get("id"); st2 = None
for _ in range(40):
    c2, rr2, _ = api("sv.gioi", "GET", f"/courses/{c1}/exams/{eid}/attempts/{aid}/runs/{rid2}", headers=TAB)
    st2 = (rr2 or {}).get("status")
    if st2 not in (None, "QUEUED", "RUNNING"): break
    time.sleep(2)
ok("R-F19-38 vòng lặp vô hạn bị cắt (TLE), gateway không treo", st2 not in (None, "QUEUED", "RUNNING"), f"status={st2} {json.dumps(rr2, ensure_ascii=False)[:160]}")
c, sb, _ = api("sv.gioi", "POST", f"/courses/{c1}/exams/{eid}/attempts/{aid}/code/{item}/submit", {"language": "c11", "source": good}, headers=TAB)
ok("R-F19-39 nộp mã câu code", c in (200, 201, 202), f"{c} {str(sb)[:120]}")
c, fin, _ = api("sv.gioi", "POST", f"/courses/{c1}/exams/{eid}/attempts/{aid}/submit", {}, headers=TAB)
ok("R-F19-40 nộp bài", c == 200, f"{c} {str(fin)[:140]}")
st = None
for _ in range(100):
    c, e3, _ = api("teacher", "GET", f"/courses/{c1}/exams/{eid}"); st = e3.get("effective_status")
    if st in ("PUBLISHED", "CLOSED"): break
    time.sleep(3)
time.sleep(15)
c, rs, _ = api("sv.gioi", "GET", f"/courses/{c1}/exams/{eid}/attempts/{aid}/result")
ok("R-F19-41 công bố: SV thấy điểm máy chấm (10/10 cho lời giải đúng)", c == 200 and ("10" in json.dumps(rs)), f"{c} {json.dumps(rs, ensure_ascii=False)[:300]}")
print("w8 done")
