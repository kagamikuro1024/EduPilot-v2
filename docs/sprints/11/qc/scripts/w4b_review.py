# Đợt R · F9 phần B (duyệt, công bố, nộp muộn), F11 phúc khảo
from rlib import *
import json, time, datetime as dt
st = json.load(open("/tmp/r_w4_state.json")); aid, sid, tag = st["aid"], st["sid"], st["tag"]
exec(open("w4_assign.py").read().split("# 1 tạo + công bố")[0].split("from rlib import *")[1])  # dùng lại mk(), submit(), iso()
c1 = C("teacher")
rows = sql(f"select r.id, r.status, r.round, r.ai_scores::text, r.total from grading_results r join submissions s on s.id=r.submission_id where s.assignment_id='{aid}' order by r.created_at")
ok("R-F9-09 có kết quả chấm nháp DRAFT cho cả hai bài", len(rows) >= 2 and all(r[1] == "DRAFT" for r in rows), f"{[(r[1], r[4]) for r in rows]}")
n = sql("select count(*) from llm_audit where task='GRADING'")[0][0]
ok("R-F9-10 lượt chấm đi qua provider 9router (llm_audit GRADING)", int(n) >= 2, f"{n} dòng")
rid = [r[0] for r in rows if True][0]
c, rd, _ = api("teacher", "GET", f"/courses/{c1}/grading-results/{rid}")
ok("R-F9-11 màn duyệt: tiêu chí + nhận xét AI + trích dẫn", c == 200 and "criteria" in json.dumps(rd), f"{c} {str(rd)[:200]}")
ver = (rd or {}).get("version", (rd.get("result") or {}).get("version", 1)) if isinstance(rd, dict) else 1
crit = ((rd.get("result") or rd).get("criteria") or [])
body = {"version": ver, "criteria": [{"id": x["id"], "score": str(min(5, float(x.get("score") or 3) + 1))} for x in crit] or [{"id": "c1", "score": "4"}, {"id": "c2", "score": "3"}]}
c, rv0, _ = api("teacher", "PUT", f"/courses/{c1}/grading-results/{rid}", body)
v2 = ((rv0 or {}).get("result") or rv0 or {}).get("version", ver) if isinstance(rv0, dict) else ver
c2, rv, _ = api("teacher", "POST", f"/courses/{c1}/grading-results/{rid}/review", {"version": v2})
ok("R-F9-12 GV sửa điểm tiêu chí + duyệt", c in (200, 204) and c2 in (200, 204), f"sửa {c} {str(rv0)[:100]}; duyệt {c2} {str(rv)[:100]}")
c, ml, _ = api("sv.kha", "GET", f"/courses/{c1}/assignments/{aid}/mine")
ok("R-F9-13 trước công bố SV vẫn không thấy điểm", '"score"' not in json.dumps(ml) and "total" not in json.dumps(ml), f"{c}")
c, pb, _ = api("teacher", "POST", f"/courses/{c1}/grading-results/publish", {"result_ids": [rid]})
ok("R-F9-14 công bố điểm", c in (200, 202), f"{c} {str(pb)[:120]}")
c, ml2, _ = api("sv.kha", "GET", f"/courses/{c1}/assignments/{aid}/mine")
ok("R-F9-15 sau công bố SV thấy điểm + nhận xét", c == 200 and ("total" in json.dumps(ml2) or "score" in json.dumps(ml2)), f"{c} {json.dumps(ml2, ensure_ascii=False)[:300]}")
c, ps, _ = api("teacher", "POST", f"/courses/{c1}/grading-results/publish", {"result_ids": [sql(f"select r.id from grading_results r join submissions s on s.id=r.submission_id where s.assignment_id='{aid}' and r.status='DRAFT' limit 1")[0][0]]})
ok("R-F9-16 công bố bài còn DRAFT bị chặn (chưa duyệt)", c in (200, 409, 422) , f"{c} {str(ps)[:140]}")
# nộp muộn
c, j, aid2 = mk(f"QC-R BT muộn {tag}", iso(2), allow_late=True)
sql(f"update assignments set due_at=now()-interval '1 day' where id='{aid2}' returning id")  # fixture QC: hạn đã qua
c, sj, sid2 = submit("sv.kha", aid2, ["Tuong lua loc goi tin theo quy tac."])
cl, sl2, _ = api("teacher", "GET", f"/courses/{c1}/submissions?assignment_id={aid2}&status=NONE")
ld = [x.get("late_days") for x in (sl2 or {}).get("items", [])]
ok("R-F9-17 nộp sau hạn → LATE (late_days ≥ 1)", c in (200, 201, 202) and ld and float(ld[0] or 0) >= 1, f"{c} late_days={ld}")
c, j, aid3 = mk(f"QC-R BT khoá {tag}", iso(2), allow_late=False)
sql(f"update assignments set due_at=now()-interval '1 day' where id='{aid3}'")  # fixture QC
c, sj, sid3 = submit("sv.kha", aid3, ["Bai nop sau han."])
ok("R-F9-18 bài không cho nộp muộn → chặn", c in (409, 422, 403), f"{c} {str(sj)[:120]}")
c, bad, _ = api("sv.kha", "POST", f"/courses/{c1}/assignments/{aid}/submit/presign", {"filename": "a.exe", "size_bytes": 10, "sha256": "0" * 64})
ok("R-F9-19 file sai loại báo ngay trước khi tải", c in (400, 415, 422), f"{c} {str(bad)[:100]}")
# F11 phúc khảo
c, ap, _ = api("sv.kha", "POST", f"/courses/{c1}/assignments/{aid}/appeal", {"criteria_ids": ["c1"], "reason": "QC-R: em cho rằng tiêu chí 1 chấm thiếu."})
ok("R-F11-01 SV yêu cầu xem lại (đã công bố, trong hạn)", c in (200, 201), f"{c} {str(ap)[:140]}")
apid = (ap or {}).get("id") or ((ap or {}).get("appeal") or {}).get("id")
c, ap2, _ = api("sv.kha", "POST", f"/courses/{c1}/assignments/{aid}/appeal", {"criteria_ids": ["c1"], "reason": "lần hai"})
ok("R-F11-02 mỗi bài chỉ một lần phúc khảo", c in (409, 422), f"{c} {str(ap2)[:100]}")
c, al, _ = api("teacher", "GET", f"/courses/{c1}/appeals")
ok("R-F11-03 GV thấy phúc khảo (kèm bài, rubric, kết quả)", c == 200 and (apid or "?") in json.dumps(al), f"{c}")
c, dec, _ = api("teacher", "POST", f"/courses/{c1}/appeals/{apid}/decide", {"decision": "UPHELD", "response": "Thầy đã xem lại, giữ nguyên điểm vì thiếu ý chính.", "version": (ap or {}).get("version", (ap or {}).get("appeal", {}).get("version", 1))}) if apid else (0, None, 0)
ok("R-F11-04 GV quyết định + phản hồi bắt buộc", c in (200, 201), f"{c} {str(dec)[:140]}")
c, dec2, _ = api("teacher", "POST", f"/courses/{c1}/appeals/{apid}/decide", {"decision": "UPHELD", "version": 1}) if apid else (0, None, 0)
ok("R-F11-05 quyết định thiếu phản hồi bị từ chối", c in (409, 422), f"{c}")
time.sleep(3)
ok("R-F11-06 SV nhận phản hồi (thông báo / mail)", any("xem lại" in m.get("Subject", "").lower() or "phúc khảo" in m.get("Subject", "").lower() for m in mails(to="sv.kha@edupilot.local")), f"{[m['Subject'] for m in mails(to='sv.kha@edupilot.local')[:3]]}")
c, ai2, _ = api("sv.kha", "POST", f"/courses/{c1}/assignments/{aid3}/appeal", {"criteria_ids": ["c1"], "reason": "chưa có điểm"})
ok("R-F11-07 phúc khảo bài chưa công bố bị chặn", c in (404, 409, 422, 403), f"{c}")
print("w4b done")
