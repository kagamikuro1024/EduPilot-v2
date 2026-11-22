import sys, os, re, time, json, subprocess; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
import gdlib
W = "/Users/kuro/.config/edupilot-team/qc3-work/"; ENVD = "/Users/kuro/Documents/ep-qc-env/"; EVD = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-10.5/env/"
STUB_MODE = W + "stub_dec_mode.json"; STUB_REQ = W + "stub_dec_req.jsonl"; PID = "01a1deef-8e87-74a9-b3d2-aad764ed3dc8"
def F2(extra=None):
    base = f"--env-file .env.local -f docker-compose.local.yml -f docker-compose.test.yml -f /Users/kuro/Documents/TA_Agent_qc106/docs/sprints/10.6/qc/scripts/kc/docker-compose.kc.yml -f {ENVD}ep-qc.override.yml"
    return base + (f" -f {extra}" if extra else "") + " -p ep-qc"
def write_override(name, env):
    y = "services:\n" + "".join(f"  {s}:\n    environment:\n" + "".join(f"      {k}: \"{v}\"\n" for k, v in env.items()) for s in ("gateway", "worker"))
    open(ENVD + name, "w").write(y); return ENVD + name
def envswap(extra=None):
    cmd = f". {ENVD}env.sh; export COMPOSE_PROFILES=ingest FAKE_LLM_VALID_KEY= SEED_ON_EMPTY_DB=true; cd /Users/kuro/Documents/TA_Agent_qc106 && docker compose {F2(extra)} up -d --no-deps --force-recreate gateway worker 2>&1 | tail -2"
    out = subprocess.run(["bash", "-c", cmd], capture_output=True, text=True).stdout
    for _ in range(60):
        time.sleep(3)
        r = subprocess.run(["curl", "-sk", "-m", "3", "https://localhost:943/api/v1/readyz"], capture_output=True, text=True).stdout
        if '"status":"ready"' in r: break
    time.sleep(4); return r
def snap(tag):
    for s in ("gateway", "worker"):
        o = subprocess.run(["bash", "-c", f"docker inspect ep-qc-{s}-1 --format '{{{{range .Config.Env}}}}{{{{println .}}}}{{{{end}}}}' | sed -E 's/((KEY|SECRET|PASSWORD|TOKEN|DSN|DATABASE_URL)[A-Z_]*)=.*/\\1=<ẩn>/I' | sort > {EVD}{s}-{tag}.env"], capture_output=True, text=True)
def start_stub(port=9770):
    open(STUB_MODE, "w").write('{"mode":"ok","noul":0.25}'); open(STUB_REQ, "w").close()
    p = subprocess.Popen(["python3", os.path.join(os.path.dirname(os.path.abspath(__file__)), "stub_decision.py"), str(port)], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL); time.sleep(1); return p
def smode(**k): open(STUB_MODE, "w").write(json.dumps(k))
def sreqs(since=0): return [r for r in (json.loads(l) for l in open(STUB_REQ) if l.strip()) if r["t"] >= since]
def dprov(model, key=None, enabled=True):
    cur = [x for x in api("admin", "GET", "/admin/llm/providers")[1]["items"] if x["id"] == PID][0]
    body = {"type": "decision", "name": cur["name"], "base_url": cur.get("base_url"), "enabled": enabled, "rpm_limit": cur.get("rpm_limit"), "tpm_limit": cur.get("tpm_limit"), "version": cur["version"],
            "models": [{"model": m, "kind": "decision", "price_in": "0", "price_out": "0", "enabled": m == model} for m in ("decision-fake", "gemini-decision-1")]}
    if key is not None: body["api_key"] = key
    c, j, _ = api("admin", "PUT", f"/admin/llm/providers/{PID}", body); return c, lastj(j, 200)
def drows2(s, gate=None): time.sleep(3); g = f"and cascade_gate='{gate}'" if gate else ""; return sql(f"select cascade_gate,cascade_path,status,coalesce(error_kind,''),coalesce(latency_ms,0),coalesce(pii_masked_count,0),coalesce(provider,''),coalesce(model,'') from llm_audit where task='DECIDE' and created_at>='{s}' {g} order by created_at")
