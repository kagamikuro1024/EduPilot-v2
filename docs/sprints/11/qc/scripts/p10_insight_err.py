# TC-04-22: nhà cung cấp NAME lỗi (địa chỉ không nghe) → job DONE, tên tạm theo từ khoá, named_by fallback, xếp hạng giữ nguyên
from p10lib import *
import subprocess
from llm_route import mk, provs, model_id
import llm_route
c1 = course("teacher", "761987")
bad = mk("QC lỗi", {"type": "openai_compatible", "base_url": "http://127.0.0.1:9/v1", "api_key": "x", "enabled": True, "skip_verify": True, "models": [{"model": "bad-model", "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]})
bad = provs()["QC lỗi"]
c, rt, _, _ = api("admin", "GET", "/admin/llm/routes"); ins = [r for r in rt["items"] if r["task"] == "INSIGHT"][0]
c, j, _, _ = api("admin", "PUT", "/admin/llm/routes", {"task": "INSIGHT", "chain": [model_id(bad, "bad-model")], "params": ins["params"], "version": ins["version"]}); print("route", c)
try:
    sql(f"update insight_reports set data_watermark = data_watermark - interval '1 day' where course_id='{c1}'")
    subprocess.run(["docker", "exec", "ep-qc-redis-1", "sh", "-c", "redis-cli --scan --pattern 'ep:rl:insight-generate:*' | xargs -r redis-cli del"], capture_output=True)
    time.sleep(3)
    prev = sql(f"select count(*) from insight_reports where course_id='{c1}'")[0][0]
    c, j, _, _ = api("teacher", "POST", f"/courses/{c1}/insights/knowledge-gap/generate", {"period_days": 120}); print(c, j)
    end = time.time() + 240
    while time.time() < end:
        jj = api("teacher", "GET", f"/jobs/{j['job_id']}")[1]
        if jj["status"] in ("SUCCEEDED", "FAILED", "DEAD"): break
        time.sleep(3)
    ok("TC-04-22 NAME lỗi: job không treo, hoàn tất", jj["status"] == "SUCCEEDED", str(jj)[:160])
    rep = json.loads(sql(f"select payload::text from insight_reports where course_id='{c1}' order by created_at desc limit 1")[0][0])
    nb = {t.get("named_by") for t in rep["topics"]}
    ok("TC-04-22 named_by = fallback, khuyến nghị rỗng", nb == {"fallback"} and all(not t.get("recommendation") for t in rep["topics"]), f"{nb}")
    ref = json.loads(sql(f"select payload::text from insight_reports where course_id='{c1}' and payload->>'period_days'='120' order by created_at desc offset 1 limit 1")[0][0])
    ok("TC-04-22 xếp hạng giữ nguyên (cùng thứ tự điểm)", [t["score"] for t in rep["topics"]][:5] == [t["score"] for t in ref["topics"]][:5], "")
finally:
    c, rt, _, _ = api("admin", "GET", "/admin/llm/routes"); ins = [r for r in rt["items"] if r["task"] == "INSIGHT"][0]
    fk = provs()["QC fake"]
    api("admin", "PUT", "/admin/llm/routes", {"task": "INSIGHT", "chain": [model_id(fk, "fake-chat")], "params": ins["params"], "version": ins["version"]})
