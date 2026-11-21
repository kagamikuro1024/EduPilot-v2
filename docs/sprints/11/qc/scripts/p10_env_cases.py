# TC-04-23 (lỗi trước NAME: embedding hỏng), TC-04-24 (LLM chậm 60 s), TC-02-10 (ghi payload lỗi), TC-03-11 (worker tắt)
import sys, subprocess, threading, http.server, socketserver, json as _j
from p10lib import *
import llm_route
from llm_route import mk, provs
part = sys.argv[1]
c1 = course("teacher", "761987"); GEN = f"/courses/{c1}/insights/knowledge-gap"
def rl(): subprocess.run(["docker", "exec", "ep-qc-redis-1", "sh", "-c", "redis-cli --scan --pattern 'ep:rl:insight-generate:*' | xargs -r redis-cli del"], capture_output=True)
def fresh(): sql(f"update insight_reports set data_watermark = data_watermark - interval '1 day' where course_id='{c1}'"); rl()
def job(j, t=300):
    end = time.time() + t
    while time.time() < end:
        r = api("teacher", "GET", f"/jobs/{j}")[1]
        if r["status"] in ("SUCCEEDED", "FAILED", "DEAD"): return r
        time.sleep(3)
    return r
def setroute(task, prov, model):
    c, rt, _, _ = api("admin", "GET", "/admin/llm/routes"); r = [x for x in rt["items"] if x["task"] == task][0]
    mid = [m for m in prov["models"] if m["model"] == model][0]["id"] if prov else None
    return api("admin", "PUT", "/admin/llm/routes", {"task": task, "chain": [mid] if mid else [], "params": r["params"], "version": r["version"]})
if part == "p423":
    bad = mk("QC lỗi embed", {"type": "openai_compatible", "base_url": "http://127.0.0.1:9/v1", "api_key": "x", "enabled": True, "skip_verify": True, "models": [{"model": "bad-embed", "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0", "enabled": True}]})
    bad = provs()["QC lỗi embed"]; print("route", setroute("EMBEDDING", bad, "bad-embed")[0])
    try:
        before = sql(f"select count(*) from insight_reports where course_id='{c1}'")[0][0]; old = api("teacher", "GET", f"{GEN}/latest")[1]["report"]["id"]
        fresh(); time.sleep(3); c, j, _, _ = api("teacher", "POST", f"{GEN}/generate", {"period_days": 14}); print(c, j)
        r = job(j["job_id"]) if c == 202 else {"status": f"HTTP {c}"}
        ok("TC-04-23 lỗi trước NAME (embedding hỏng): job FAILED, không bản ghi mới, latest vẫn báo cáo cũ", r["status"] in ("FAILED", "DEAD") and sql(f"select count(*) from insight_reports where course_id='{c1}'")[0][0] == before and api("teacher", "GET", f"{GEN}/latest")[1]["report"]["id"] == old, f"{r['status']} {str(r.get('error') or '')[:80]} rep {before}->{sql(f'select count(*) from insight_reports where course_id={chr(39)}{c1}{chr(39)}')[0][0]}")
    finally:
        print("restore", setroute("EMBEDDING", None, None)[0])
if part == "p424":
    class H(http.server.BaseHTTPRequestHandler):
        def do_POST(self):
            n = int(self.headers.get("content-length", 0)); self.rfile.read(n); time.sleep(60)
            b = _j.dumps({"id": "x", "object": "chat.completion", "choices": [{"index": 0, "message": {"role": "assistant", "content": "{}"}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}}).encode()
            try: self.send_response(200); self.send_header("content-type", "application/json"); self.send_header("content-length", str(len(b))); self.end_headers(); self.wfile.write(b)
            except Exception: pass
        def log_message(self, *a): pass
    srv = socketserver.ThreadingTCPServer(("0.0.0.0", 28999), H); srv.daemon_threads = True; threading.Thread(target=srv.serve_forever, daemon=True).start()
    slow = mk("QC chậm", {"type": "openai_compatible", "base_url": "http://host.docker.internal:28999/v1", "api_key": "x", "enabled": True, "skip_verify": True, "models": [{"model": "slow-model", "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]})
    slow = provs()["QC chậm"]; print("route", setroute("INSIGHT", slow, "slow-model")[0])
    try:
        fresh(); time.sleep(3); c, j, _, _ = api("teacher", "POST", f"{GEN}/generate", {"period_days": 14}); print(c, j)
        t0 = time.time(); lat = []
        while time.time() - t0 < 50:
            for u in ("sv.kha",):
                s = api(u, "POST", "/chat/sessions", {"course_id": c1})[1]["id"]; t = time.time(); r = api(u, "POST", f"/chat/sessions/{s}/messages", {"content": "Tỷ lệ điểm quá trình và điểm cuối kỳ là bao nhiêu phần trăm?"}); lat.append((time.time() - t) * 1000)
            time.sleep(4)
        lat.sort(); print("chat trong lúc job chậm:", [int(x) for x in lat])
        lane = sql("select lane from llm_audit where task='INSIGHT' order by created_at desc limit 1")
        ok("TC-04-24 LLM chậm 60 s: chat vẫn ≤ 3 s (p95), job BATCH không treo chat", lat and lat[int(0.95 * len(lat)) - 1] <= 3000, f"n={len(lat)} p95={lat[int(0.95*len(lat))-1]:.0f} ms max={lat[-1]:.0f}")
        r = job(j["job_id"], 200); ok("TC-04-24b job chậm vẫn kết thúc (không treo)", r["status"] in ("SUCCEEDED", "FAILED", "DEAD"), r["status"])
    finally:
        fk = provs()["QC 9router"]; print("restore", setroute("INSIGHT", fk, "ag/gemini-3.8-flash")[0]); srv.shutdown()
