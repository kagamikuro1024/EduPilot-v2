from asl import *
import json
ex = lambda code: (sql(f"select id from courses where class_code='{code}'") or [[None]])[0][0]
cid = ex("QC-AS") or mkcourse("QC-AS", "QCKASTV")
gen_sessions(cid, -2, 40)
c, j = policy(cid, rotate_seconds=5); print("policy", c)
ss = sessions(TEACHER, cid); print(len(ss), ss[0])
cid2 = ex("QC-AS2") or mkcourse("QC-AS2", "QCKASTW")
if not sessions(TEACHER, cid2): gen_sessions(cid2, 0, 1)
json.dump({"cid": cid, "cid2": cid2, "sessions": [s["id"] for s in ss]}, open(STATE, "w"))
print(json.dumps(ss[:2], ensure_ascii=False)[:600])
