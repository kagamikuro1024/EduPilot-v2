import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10")); import qlib
PID = "01a1deef-8e87-74a9-b3d2-aad764ed3dc8"
def pset(en):
    cur = [x for x in api("admin", "GET", "/admin/llm/providers")[1]["items"] if x["id"] == PID][0]
    return api("admin", "PUT", f"/admin/llm/providers/{PID}", {**{k: cur.get(k) for k in ("name", "type", "base_url", "rpm_limit", "tpm_limit") if cur.get(k) is not None}, "enabled": en, "version": cur["version"]})[0]
def chat():
    cs, sj, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": C}); return qlib.sse("sv.kha", f"/chat/sessions/{sj['id']}/messages", {"content": "Nói giúp mình về mấy cái bảo mật mạng căn bản trong môn này nhé"})[0]
def run():
    set_route("CHAT", "QC fake", "fake-chat")
    try:
        gate("CHAT_ROUTING", True, [C]); redis_del("ep:llm:cb:*"); fake(decision_mode="")
        s0 = now(); chat(); time.sleep(3); base = rows(s0, "and cascade_gate='CHAT_ROUTING'")
        pset(False); out = []
        for wait in (3, 20, 40):
            time.sleep(wait); s = now(); chat(); time.sleep(3); out.append((wait, [(x[1], x[2]) for x in rows(s, "and cascade_gate='CHAT_ROUTING'")]))
        print("base", [(x[1], x[2]) for x in base]); print("tắt", out)
        chk5("TC-D01-25-chat", all(o[1] == [] for o in out), f"provider tắt + CHAT_ROUTING bật: chat sau +3 s / +23 s / +63 s: DECIDE {out}; trước khi tắt: {[(x[1], x[2]) for x in base]}")
    finally:
        print("bật lại", pset(True)); off(); set_route("CHAT", "QC 9router", "ag/gemini-3.8-flash")
with dwin("QC3 105 D01-25 chat"): run()
