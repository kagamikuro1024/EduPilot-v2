# gd_b2: bản sửa cho ep-r: G03-16 PII với roster thật; G03-22 chờ lâu (retry + backoff); G03-18 trần 8
from gdlib import *
import json
n, m, e = sql("select full_name, student_code, email from users where email='sv.kha@edupilot.local'")[0]
json.dump({"name": n, "mssv": m, "email": e}, open("/tmp/qc2g-stub_pii.json", "w"), ensure_ascii=False)
def clear(): sql("delete from grading_guidance_items")
clear(); c, j, jb = reflect("rf_pii_text", "qc-rpt-b2-0001")
ok("G03-16 [rf_pii_text, roster thật] bỏ mọi mục PII (0 DRAFT)", jb[0] == "SUCCEEDED" and len(drafts()) == 0, f"{jb[0]} drafts={len(drafts())} {jb[1][:140]}")
clear(); c, j, jb = reflect("rf_many", "qc-rmn-b2-0001")
ok("G03-18 [rf_many] tối đa 8 DRAFT", jb[0] == "SUCCEEDED" and len(drafts()) <= 8, f"{jb[0]} drafts={len(drafts())}")
for mode in ("error", "invalid"):
    clear(); stub(mode); bump()
    c, j, _ = api("teacher", "POST", G + "/reflect", {}, idem=f"qc-b2-{mode}-{int(time.time())}")
    jb = wait_job(j["job_id"], 300) if c == 202 else (c, "", "")
    a = sql("select count(*) from llm_audit where task='GRADING' and created_at > now() - interval '5 minutes' and status <> 'ok'")[0][0]
    ok(f"G03-22 [{mode}] retry rồi job FAILED, 0 DRAFT (chờ ≤ 300 s)", jb[0] == "FAILED" and len(drafts()) == 0, f"{jb[0]} {jb[2][:80]} llm_audit lỗi 5ph={a}")
stub("allow")
