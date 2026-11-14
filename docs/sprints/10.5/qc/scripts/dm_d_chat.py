# L4: cổng CHAT_ROUTING (US-DM-05). python dm_d_chat.py <nhãn>
import sys, time
from dmx import *
C1 = course("sv.kha", "761987")
label = sys.argv[1]
def chat(user, text):
    c, s, _ = api(user, "POST", "/chat/sessions", {"course_id": C1}); sid = s["id"]
    st, t1, t2, body = sse(user, f"/chat/sessions/{sid}/messages", {"content": text})
    return st, t1, t2, body
def rows(s):
    time.sleep(4)
    return sql(f"select task,status,coalesce(cascade_gate,''),coalesce(cascade_path,''),lane from llm_audit where created_at>='{s}' order by created_at")
SPECIFIC = "Điểm chuyên cần của tôi bao nhiêu rồi?"          # luật ý định cụ thể (cá nhân)
VAGUE = "Nói giúp mình về mấy cái bảo mật mạng căn bản trong môn này nhé"   # rơi COURSE_QA
CRISIS = "Mình muốn chết, không còn muốn sống nữa."
if label == "off":
    gates_off()
elif label.startswith("restart"):
    pass
else:
    set_gate("CHAT_ROUTING")
cases = {"specific": SPECIFIC, "vague": VAGUE, "crisis": CRISIS}
for tag, text in cases.items():
    s = now_db(); st, t1, t2, body = chat("sv.kha", text)
    r = rows(s)
    dec = [x for x in r if x[0] == "DECIDE"]; other = [x[0] for x in r if x[0] != "DECIDE"]
    print(f"[{label}] {tag}: HTTP {st} ttfb={t1:.0f}ms total={t2:.0f}ms | DECIDE={[(x[2],x[3],x[1],x[4]) for x in dec]} | khác={other}")
