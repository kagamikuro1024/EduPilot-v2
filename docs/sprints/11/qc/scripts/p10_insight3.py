# QC sprint 11 — TC-04-13 / 04-15 / 04-20 (API+DB). Chạy với route fake (đã có báo cáo lớp 1).
import json, subprocess
from p10lib import *
c1 = course("teacher", "761987")
GEN = f"/courses/{c1}/insights/knowledge-gap"
c, lat, _, _ = api("teacher", "GET", f"{GEN}/latest"); rep = lat["report"]; pl = rep["payload"]
# TC-04-13
bad = []
for t in pl["topics"]:
    if t.get("few_asked"): continue
    ex = t["examples"]
    stu = set()
    for e in ex:
        r = sql("select m.session_id, s.user_id from chat_messages m join chat_sessions s on s.id=m.session_id where s.course_id='%s' and m.role='USER' and m.content = $q$%s$q$ limit 1" % (c1, e))
        if not r:
            r = sql("select m.session_id, s.user_id from chat_messages m join chat_sessions s on s.id=m.session_id where s.course_id='%s' and m.role='USER' and m.content like $q$%s%%$q$ limit 1" % (c1, e[:40]))
        if not r:
            r = sql("select null, t.author_id from forum_threads t where t.course_id='%s' and (t.body like $q$%s%%$q$ or t.title like $q$%s%%$q$) limit 1" % (c1, e[:40], e[:40]))
        stu.add(r[0][1] if r else "?" + e[:20])
    if not (3 <= len(ex) <= 5) or len(stu) != len(ex) or any(s.startswith("?") for s in stu): bad.append((t["name"], len(ex), len(stu), [s for s in stu if s.startswith("?")]))
ok("TC-04-13 examples 3–5 câu, mỗi câu từ SV khác nhau (đối chiếu DB)", not bad and len(pl["topics"]) > 0, f"{len(pl['topics'])} chủ đề; lệch={bad[:3]}")
# TC-04-15
rows = sql("select id, pii_masked_count, trace_id from llm_audit where task='INSIGHT' order by created_at desc limit 3")
print(rows)
