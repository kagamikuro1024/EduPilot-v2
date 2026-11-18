# 04-31 + 04-34: UI tải lên hỏng / trích FAILED. Giữ khoá, UTILITY → stub 'invalid' trong lúc Playwright chạy.
import subprocess, os
exec(open('s8r-03i.py').read().split('os.makedirs("/tmp/qc2-s8r"')[0])
os.makedirs("/tmp/qc2-s8r", exist_ok=True)
if not acquire(): print("không lấy được khoá"); sys.exit(2)
stub = subprocess.Popen([sys.executable, os.path.dirname(os.path.abspath(__file__)) + "/s8r-stub.py", "9764"]); time.sleep(1.5)
logl("LOCK — 04-31/04-34 UI: UTILITY → S8R-stub(invalid); ≤ 6 phút")
try:
    prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub2"]
    if not prov:
        c, pr = call("POST", "/admin/llm/providers", "admin", {"type": "openai_compatible", "name": "S8R-stub2", "base_url": "http://host.docker.internal:9764/v1", "api_key": "good-key", "models": [{"model": "m-ext", "kind": "chat", "price_in": "0", "price_out": "0"}]}); print("create prov", c)
        prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub2"]
    print("setroute", setroute("UTILITY", prov[0]["models"][0]["id"])[0]); setmode("invalid")
    env = dict(os.environ, QC2G_DIR="e2e/qc2-r8", QC_REAL="1", NODE_TLS_REJECT_UNAUTHORIZED="0", QC_REPO=REPO, QC_PDF="/tmp/qc2-pdf34.pdf")
    r = subprocess.run("npx playwright test -c playwright.qc2g.config.ts real-04b -g 04-34", shell=True, cwd=REPO + "/frontend", env=env, capture_output=True, text=True, timeout=250)
    print("\n".join(l for l in r.stdout.split("\n") if l.startswith(("P ", "F ", "  ✓", "  ✘")) or "Error" in l)[:3000])
finally:
    stub.terminate(); setmode("lop1")
    print("restore", setroute("UTILITY", REAL_MODEL)[0]); logl("UNLOCK — UTILITY hoàn lại (04-31/34)")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
