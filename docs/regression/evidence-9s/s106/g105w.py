import sys, os, json, time, re, unicodedata, subprocess; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from s3lib import *
sys.path.insert(0, "/Users/kuro/checkouts/qc2seedt3/benchmarks"); import eval_pii as E
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
ROOT = "/Users/kuro/checkouts/qc2seedt3/benchmarks/pii/"
roster = json.load(open(ROOT + "roster_seed.json", encoding="utf8")); roster = {r["n"]: r for r in roster}
dev = [r for r in (json.loads(l) for l in open(ROOT + "e1_dataset.jsonl", encoding="utf8")) if r["split"] == "dev"]
C1 = cid("761987")
def gate(on, model=None):
    g = [x for x in api("admin", "GET", "/admin/llm/decision")[1]["gates"] if x["gate"] == "THREADS_PRECHECK"][0]; api("admin", "PUT", "/admin/llm/decision/gates/THREADS_PRECHECK", {"enabled": on, "version": g["version"]})
    cg = [x for x in api("admin", "GET", f"/courses/{C1}/decision-gates")[1]["items"] if x["gate"] == "THREADS_PRECHECK"][0]; api("admin", "PUT", f"/courses/{C1}/decision-gates/THREADS_PRECHECK", {"enabled": on, "version": cg["version"]})
    if model:
        pr = [p for p in api("admin", "GET", "/admin/llm/providers")[1]["items"] if p["type"] == "decision"][0]
        body = {"type": "decision", "name": pr["name"], "base_url": pr.get("base_url"), "enabled": True, "rpm_limit": pr.get("rpm_limit"), "tpm_limit": pr.get("tpm_limit"), "version": pr["version"], "models": [{"model": m, "kind": "decision", "price_in": "0", "price_out": "0", "enabled": m == model} for m in ("decision-fake", "gemini-decision-1")]}
        print("provider", model, api("admin", "PUT", f"/admin/llm/providers/{pr['id']}", body)[0])
    time.sleep(4)
def run(label):
    out = {}; t = tok("sv.kha"); t0 = time.time()
    for it in dev:
        text = E.fill(it["text"], roster)
        if label == "C": json.dump({"mode": "ok", "noul": 0.99 if it["expect_block"] else 0.01}, open(W + "stub_dec_mode.json", "w"))
        for _ in range(20):
            st, body, hdr = raw("POST", f"/courses/{C1}/threads", {"title": f"Đo E1 dev {it['id']} ({label})", "body": text}, t)
            if st == 429: time.sleep(5); continue
            break
        out[it["id"]] = (st, (body or {}).get("code") if isinstance(body, dict) else None); time.sleep(0.2)
    json.dump(out, open(W + f"e1p_{label}.json", "w")); print(label, "xong", round(time.time() - t0), "s", flush=True); return out
if __name__ == "__main__":
    phase = sys.argv[1]
    if phase == "A": gate(False); run("A")
    if phase == "B": gate(True, "decision-fake"); run("B"); gate(False)
    if phase == "C": gate(True, "gemini-decision-1"); run("C"); gate(False, "decision-fake")
