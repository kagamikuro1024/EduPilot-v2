import sys, os, re; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
import gdlib
from gdlib import tok
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10")); import qlib
def chat(user, text):
    c, s, _ = api(user, "POST", "/chat/sessions", {"course_id": C}); sid = s["id"]
    st, t1, t2, body = qlib.sse(user, f"/chat/sessions/{sid}/messages", {"content": text}); return st, round(t1), round(t2), body
def evs(body): return re.findall(r"^event: (\S+)", body, re.M)
def drows(s, gate="CHAT_ROUTING"): time.sleep(4); return sql(f"select cascade_path,status,coalesce(error_kind,''),latency_ms from llm_audit where task='DECIDE' and cascade_gate='{gate}' and created_at>='{s}' order by created_at")
VAGUE = "Nói giúp mình về mấy cái bảo mật mạng căn bản trong môn này nhé"
CRISIS = ["Mình muốn chết, không còn muốn sống nữa.", "Mình không muốn sống nữa", "Tôi muốn tự tử", "Mình chán sống quá, muốn kết thúc tất cả", "Mình muốn biến mất khỏi thế giới này mãi mãi"]
def run():
    set_route("CHAT", "QC fake", "fake-chat")
    off(); redis_del("ep:llm:cb:*"); fake(decision_mode="", decision_p50_ms=0, decision_p95_ms=0)
    st0, a0, b0, body0 = chat("sv.kha", VAGUE); print("off", st0, a0, b0, evs(body0)[:4])
    gate("CHAT_ROUTING", True, [C]); time.sleep(1)
    s = now(); st1, a1, b1, body1 = chat("sv.kha", VAGUE); d1 = drows(s); print("on", st1, a1, b1, evs(body1)[:4], d1)
    chk5("TC-D05-03", st1 == 200 and evs(body1)[:1] == ["received"] and a1 <= 300 and evs(body0)[:1] == ["received"], f"CHAT_ROUTING bật (QD105A, provider fake cho CHAT/CLASSIFY): sự kiện SSE đầu «{(evs(body1) or ['-'])[0]}» sau {a1} ms (tắt cổng: «{(evs(body0) or ['-'])[0]}» sau {a0} ms); DECIDE {[(x[0], x[1]) for x in d1]}")
    # D05-05: decision chậm 3000 → bỏ sau ~800 ms
    fake(decision_p50_ms=3000, decision_p95_ms=3500); s = now(); st2, a2, b2, body2 = chat("sv.kha", VAGUE); d2 = drows(s); fake(decision_p50_ms=0, decision_p95_ms=0)
    chk5("TC-D05-05", st2 == 200 and any(x[1] == "timeout" for x in d2) and b2 < b0 + 2500, f"decision_p50=3000 ms: chat hoàn tất {b2} ms (cổng tắt {b0} ms), sự kiện đầu {a2} ms; DECIDE {[(x[0], x[1], x[3]) for x in d2]} (timeout cổng 800 ms → dùng luật; không treo)")
    # D05-10: khủng hoảng
    s = now(); res = [chat("sv.kha", t) for t in CRISIS]; time.sleep(3); dd = sql(f"select count(*) from llm_audit where task='DECIDE' and created_at>='{s}'")[0][0]; ll = sql(f"select count(*) from llm_audit where task in ('CHAT','CLASSIFY','UTILITY') and created_at>='{s}'")[0][0]
    chk5("TC-D05-10", int(dd) == 0 and all(r[0] == 200 for r in res), f"5 câu khủng hoảng, cổng CHAT_ROUTING bật: DECIDE = {dd}, lời gọi CHAT/CLASSIFY/UTILITY = {ll}, HTTP {[r[0] for r in res]}; phản hồi có sự kiện {sorted(set(sum([evs(r[3]) for r in res], [])))[:6]}")
    # D05-04: decision p50 600 song song nhúng
    fake(decision_p50_ms=600, decision_p95_ms=700); s = now(); st4, a4, b4, body4 = chat("sv.kha", VAGUE); d4 = drows(s); fake(decision_p50_ms=0, decision_p95_ms=0)
    lat = [int(float(x[3])) for x in d4]
    note5("TC-D05-04", "N", f"Chỉ đo được một phần: decision_p50=600 ms → chat {b4} ms (cổng tắt {b0} ms), DECIDE latency {lat}; chưa tách mốc nhúng / intent (không có log trace); TestOneEmbedPerMessage PASS (agent)") if False else None
    print("D05-04", b4, b0, lat)
with dwin("QC3 105 chat"):
    try: run()
    finally: set_route("CHAT", "QC 9router", "ag/gemini-3.8-flash")
