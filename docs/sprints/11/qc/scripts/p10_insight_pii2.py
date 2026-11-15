# TC-04-15 (đo delta): pii_masked_count của lời gọi đặt tên tăng đúng bằng số PII QC thêm vào (3 câu × 1 email), payload không rò.
import subprocess, time, uuid, json
from p10lib import *
c1 = course("teacher", "761987"); OBS = "/admin/observability"
def run():
    subprocess.run(["docker", "exec", "ep-qc-redis-1", "sh", "-c", "redis-cli --scan --pattern 'ep:rl:insight-generate:*' | xargs -r redis-cli del"], capture_output=True)
    sql(f"update insight_reports set data_watermark = data_watermark - interval '1 day' where course_id='{c1}'")
    t0 = sql("select clock_timestamp()")[0][0]
    c, j, _, _ = api("teacher", "POST", f"/courses/{c1}/insights/knowledge-gap/generate", {"period_days": 14}, headers={"Idempotency-Key": "qc-15b-" + uuid.uuid4().hex}); assert c == 202, (c, j)
    for _ in range(60):
        cc, jj, _, _ = api("teacher", "GET", f"/jobs/{j['job_id']}")
        if jj.get("status") in ("SUCCEEDED", "FAILED", "DEAD"): break
        time.sleep(3)
    time.sleep(1)
    return sql(f"select id, pii_masked_count from llm_audit where task='INSIGHT' and created_at >= '{t0}'::timestamptz - interval '2 seconds' order by created_at desc")
b = run(); print("baseline", b)
mails = [f"qcpii{uuid.uuid4().hex[:6]}@example.com" for _ in range(3)]
for u, m in zip(("sv.kha", "sv.gioi", "sv.nguyco"), mails):
    c, s, _, _ = api(u, "POST", "/chat/sessions", {"course_id": c1}); api(u, "POST", f"/chat/sessions/{s['id']}/messages", {"content": f"Hỏi về lịch học, thư gửi tới {m} nhé"}); time.sleep(1.5)
a = run(); print("sau", a)
base = max(int(r[1]) for r in b); after = max(int(r[1]) for r in a)
leaks = []
for r in a:
    c, d, _, _ = api("admin", "POST", f"{OBS}/requests/{r[0]}/open", {"reason": "Kiểm PII payload TC-04-15 (QC)"}); t = json.dumps(d, ensure_ascii=False)
    leaks += [m for m in mails if m in t] + (["example.com"] if "example.com" in t else [])
ok("TC-04-15 pii_masked_count của lời gọi đặt tên tăng đúng bằng số PII mới (3 email) và payload không rò", after - base == 3 and not leaks, f"baseline={base} sau={after} (Δ={after-base}, kỳ vọng 3) rò={leaks}")
