#!/usr/bin/env python3
# Tải nền cho TC-02-18: xếp việc extract liên tục (QD10A/B/C) trong ~N giây; ghi số việc xếp được và trạng thái.
import sys, os, time
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from pm import *
DUR = int(sys.argv[1]); t0 = time.time(); sent = []
CL = {c: sql(f"select id from courses where class_code='{c}'")[0][0] for c in ("QD10A", "QD10B", "QD10C")}
DOC = {c: sql(f"select id from documents where course_id='{i}' and type='EXAM_PAPER' and status='READY' and title ilike '%nhiều trang%' order by created_at desc limit 1") or sql(f"select id from documents where course_id='{i}' and type='EXAM_PAPER' and status='READY' order by created_at desc limit 1") for c, i in CL.items()}
k = 0
while time.time() - t0 < DUR:
    c = ("QD10A", "QD10B", "QD10C")[k % 3]; k += 1
    if not DOC[c]: continue
    r = api("teacher", "POST", f"/courses/{CL[c]}/questions/extract", {"document_id": DOC[c][0][0]})
    sent.append((c, r[0])); print(round(time.time() - t0), c, r[0], flush=True)
    time.sleep(6)
print("SENT", sent)
