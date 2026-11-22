import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
t0 = time.time()
while True:
    try: os.mkdir("/tmp/qc-global.lock"); break
    except FileExistsError:
        if time.time() - t0 > 600: raise SystemExit("LOCKED")
        time.sleep(5)
open("/tmp/qc-global.lock/owner", "w").write("QC3 G08-17 route EMBEDDING lỗi")
def prov(n): return [x for x in api("admin", "GET", "/admin/llm/providers")[1]["items"] if x["name"] == n][0]
def setp(n, en):
    p = prov(n); body = {"type": p["type"], "name": p["name"], "base_url": p["base_url"], "enabled": en, "rpm_limit": p.get("rpm_limit"), "tpm_limit": p.get("tpm_limit"), "version": p["version"], "models": [{"model": m["model"], "kind": m["kind"], "dims": m.get("dims"), "price_in": m.get("price_in", "0"), "price_out": m.get("price_out", "0"), "enabled": True} for m in p["models"]]}
    return api("admin", "PUT", f"/admin/llm/providers/{p['id']}", body)[0]
flag("THREAD_SUGGEST", True)
try:
    print("bật QC lỗi embed", setp("QC lỗi embed", True), "route EMBEDDING →", set_route("EMBEDDING", "QC lỗi embed", "bad-embed")); time.sleep(3)
    c, j, ms = api("sv.kha", "POST", f"/courses/{CG0}/threads/similar-confirmed", {"q": "Câu hỏi hoàn toàn mới chưa từng nhúng số " + str(time.time())}); print("API khi nhúng lỗi", c, j, round(ms) if ms else ms)
    n = sql("select count(*) from llm_audit where task='EMBEDDING' and created_at > now() - interval '1 minute' and status <> 'ok'")[0][0]; print("llm_audit EMBEDDING lỗi trong phút:", n)
    os.system(f'cd {os.path.dirname(os.path.abspath(__file__))} && ./runpw.sh qc-real-106-g08.spec.ts "TC-G08-17" 150 2>&1 | tail -12')
finally:
    print("route EMBEDDING về fake-eval", set_route("EMBEDDING", "fake-eval", "fake-embed"), "tắt QC lỗi embed", setp("QC lỗi embed", False)); redis_del("ep:llm:cb:*")
    import shutil; shutil.rmtree("/tmp/qc-global.lock", ignore_errors=True)
