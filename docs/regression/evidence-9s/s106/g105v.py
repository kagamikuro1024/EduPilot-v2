import sys, os, json, time, subprocess, urllib.request, ssl; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from s3lib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"; STUB_MODE = W + "stub_dec_mode.json"; STUB_REQ = W + "stub_dec_req.jsonl"
def smode(**k): open(STUB_MODE, "w").write(json.dumps(k))
def sreqs(t0=0): return [r for r in (json.loads(l) for l in open(STUB_REQ) if l.strip()) if r["t"] >= t0]
def sse(user, sid, content):
    rq = urllib.request.Request(A + f"/chat/sessions/{sid}/messages", data=json.dumps({"content": content}).encode(), method="POST", headers={"content-type": "application/json", "authorization": "Bearer " + tok(user), "accept": "text/event-stream", "idempotency-key": str(__import__("uuid").uuid4())})
    t = time.time(); txt = ""
    try:
        with urllib.request.urlopen(rq, context=CTX, timeout=60) as r: txt = r.read().decode()
    except urllib.error.HTTPError as e: return e.code, e.read().decode()[:200], 0
    return 200, txt, round((time.time() - t) * 1000)
def chat(user, C, msg):
    cs, sj, _ = api(user, "POST", "/chat/sessions", {"course_id": C}); return sse(user, sj["id"], msg)
def evs(txt):
    out = []; cur = None
    for l in txt.splitlines():
        if l.startswith("event:"): cur = l[6:].strip()
        elif l.startswith("data:") and cur: out.append((cur, l[5:].strip()[:300]))
    return out
def setup(on):
    C1 = cid("761987"); pr = [p for p in api("admin", "GET", "/admin/llm/providers")[1]["items"] if p["type"] == "decision"][0]
    body = {"type": "decision", "name": pr["name"], "base_url": pr.get("base_url"), "enabled": True, "rpm_limit": pr.get("rpm_limit"), "tpm_limit": pr.get("tpm_limit"), "version": pr["version"], "models": [{"model": m, "kind": "decision", "price_in": "0", "price_out": "0", "enabled": m == ("gemini-decision-1" if on else "decision-fake")} for m in ("decision-fake", "gemini-decision-1")]}
    print("provider", api("admin", "PUT", f"/admin/llm/providers/{pr['id']}", body)[0])
    g = [x for x in api("admin", "GET", "/admin/llm/decision")[1]["gates"] if x["gate"] == "CHAT_ROUTING"][0]; print("gate sys", api("admin", "PUT", "/admin/llm/decision/gates/CHAT_ROUTING", {"enabled": on, "version": g["version"]})[0])
    cg = [x for x in api("admin", "GET", f"/courses/{C1}/decision-gates")[1]["items"] if x["gate"] == "CHAT_ROUTING"][0]; print("gate lớp", api("admin", "PUT", f"/courses/{C1}/decision-gates/CHAT_ROUTING", {"enabled": on, "version": cg["version"]})[0]); time.sleep(4); return C1
if __name__ == "__main__":
    st = subprocess.Popen(["python3", os.path.join(os.path.dirname(os.path.abspath(__file__)), "stub_decision.py"), "9770"]); time.sleep(1); open(STUB_REQ, "w").close()
    try:
        C1 = setup(True)
        import re
        def ask(user, msg, lab, on):
            smode(mode="ok", choice=lab, conf=0.99); t0 = time.time(); c, txt, ms = chat(user, C1, msg); rq = sreqs(t0); body = "".join(json.loads("\"" + x + "\"") for x in re.findall(r'"t":"((?:[^"\\]|\\.)*)"', txt))
            print(f"[{user}|{lab}|{'bật' if on else 'tắt'}] «{msg}» → decision {len(rq)}; trả lời: {body[:700]!r}", flush=True); return body
        ka = ask("sv.kha", "Còn phần giữa kỳ của em thì sao nhỉ", "PERSONAL_GRADE", True)
        ga = ask("sv.gioi", "Còn phần giữa kỳ của em thì sao nhỉ", "PERSONAL_GRADE", True)
        kb = ask("sv.kha", "Còn phần giữa kỳ của bạn 20229001 thì sao nhỉ", "PERSONAL_GRADE", True)
        setup(False)
        kr = ask("sv.kha", "Điểm giữa kỳ của em thế nào", "PERSONAL_GRADE", False)
        print("kha(bật, nhãn) == kha(luật, tắt):", ka == kr, "| kha ≠ gioi:", ka != ga, flush=True)
        setup(True)
    finally:
        st.terminate(); print("tắt cổng:"); setup(False)
