import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
CD = cid("QD10D"); QB = f"/courses/{CD}/questions"; W = "/Users/kuro/.config/edupilot-team/qc3-work/"
sql(f"delete from question_bank where course_id='{CD}' and topic='QC106-ui'")
cc, j, _ = api("admin", "GET", FL(CD)); f = [x for x in j["items"] if x["feature"] == "QUESTION_HINTS"][0]; print("flag", api("admin", "PUT", f"{FL(CD)}/QUESTION_HINTS", {"enabled": True, "version": f["version"]})[0])
ids = {}
for k in ("a", "b", "c"):
    c, j, _ = api("teacher", "POST", QB, {"type": "MCQ_SINGLE", "title": f"QC106ui {k}", "topic": "QC106-ui", "difficulty": "EASY", "stem": f"QC106ui câu {k} {time.time()}?", "options": [{"body": "x"}, {"body": "y"}], "correct": [0]}); qid = j["id"]
    c, d, _ = api("teacher", "GET", f"{QB}/{qid}"); print(api("teacher", "PUT", f"{QB}/{qid}/review", {"decision": "REQUEST", "version": d["version"]})[0]); ids[k] = qid
json.dump({"cid": CD, "ids": ids, "stems": {k: sql(f"select stem from question_bank where id='{v}'")[0][0] for k, v in ids.items()}}, open(W + "g09ui.json", "w"), ensure_ascii=False); print(ids)
