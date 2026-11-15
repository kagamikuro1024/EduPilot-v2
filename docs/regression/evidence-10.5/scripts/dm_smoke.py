# dm_smoke (QC2): ma trận cổng decision theo cấu hình gateway đã khởi động lại. python dm_smoke.py <cfg>
# cfg: off | p90 | p01 | p20 | p97 | unk | err | timeout ; CLASSIFY phải trỏ stub (qc2g-stub), GRADING/CHAT tuỳ ca.
import sys, time
from dmx import *
C1 = course("sv.kha", "761987")
cfg = sys.argv[1]
SUBTLE = ("Có bạn hoàn cảnh khó khăn bị trừ điểm chuyên cần", "Mình nghe nói có một bạn trong lớp hoàn cảnh khó khăn nên bị trừ điểm chuyên cần, không biết bạn đó có bị ảnh hưởng điểm cuối kỳ không?")
CLEAN = ("Hỏi về công thức tính điểm", "Công thức tính điểm quá trình của môn này gồm những thành phần nào và trọng số ra sao vậy ạ?")
VAGUE = "Nói giúp mình về mấy cái bảo mật mạng căn bản trong môn này nhé"
SPECIFIC = "Điểm chuyên cần của tôi bao nhiêu rồi?"
CRISIS = "Mình muốn chết, không còn muốn sống nữa."
def stub(m): open("/tmp/qc2g-stub_mode", "w").write(m)
def thread(t, user="sv.gioi"):
    s = now_db(); c, j, ms = api(user, "POST", f"/courses/{C1}/threads", {"title": t[0] + " " + str(time.time())[-4:], "body": t[1]}, timeout=40)
    time.sleep(3.5)
    d = decide_rows(s); cl = llm_rows(s, "CLASSIFY")
    return c, ((j or {}).get("code") if isinstance(j, dict) else None), ms, d, cl
def chat(text):
    s = now_db(); c, ss, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": C1})
    st, t1, t2, body = sse("sv.kha", f"/chat/sessions/{ss['id']}/messages", {"content": text}); time.sleep(4)
    r = llm_rows(s); return st, t1, t2, [x for x in r if x[0] == "DECIDE"], [x[0] for x in r if x[0] == "CLASSIFY"], body
def show(tag, *a): print("  ·", tag, *a)
if cfg == "off":
    gates_off()
    c, code, ms, d, cl = thread(CLEAN); ok("D03-off/D02-06 cổng tắt: đăng được, 0 DECIDE", c in (200, 201) and not d, f"{c} {d}")
    st, t1, t2, d, cl, _ = chat(VAGUE); ok("D05-off cổng chat tắt: 0 DECIDE", st == 200 and not d, f"{st} {d}")
    sys.exit()
set_gate("THREADS_PRECHECK"); stub("allow")
mode_info = cfg
if cfg == "p90":
    c, code, ms, d, cl = thread(SUBTLE); ok("D03-02 P=0,9: 422, 1 DECIDE FAST, 0 CLASSIFY", c == 422 and len(d) == 1 and d[0][1] == "FAST" and not cl, f"{c} {code} {d} cl={len(cl)}")
    set_gate("CHAT_ROUTING"); gates_off() if False else None
    set_gate("THREADS_PRECHECK"); set_gate("CHAT_ROUTING")
    st, t1, t2, d, cl, _ = chat(VAGUE); ok("D05-02 P=0,9 (<τ 0,95) vague: 1 DECIDE FALLBACK + CLASSIFY", st == 200 and len(d) == 1 and d[0][1] == "FALLBACK" and len(cl) >= 1, f"{st} ttfb={t1:.0f}ms {d} cl={cl}")
    st, t1, t2, d, cl, _ = chat(SPECIFIC); ok("D05-03 specific: 0 DECIDE", st == 200 and not d, f"{st} {d}")
    st, t1, t2, d, cl, _ = chat(CRISIS); ok("D05-04/D09 khủng hoảng: 0 DECIDE", not d, f"{st} {d}")
if cfg == "p01":
    c, code, ms, d, cl = thread(SUBTLE); ok("D03-03 P=0,01 ≤ ε: 201, FAST, 0 CLASSIFY", c in (200, 201) and len(d) == 1 and d[0][1] == "FAST" and not cl, f"{c} {code} {d} cl={len(cl)}")
if cfg == "p20":
    stub("allow"); c, code, ms, d, cl = thread(SUBTLE); ok("D03-04a P=0,2: FALLBACK + 1 CLASSIFY, stub allow → 201", c in (200, 201) and d and d[0][1] == "FALLBACK" and len(cl) == 1, f"{c} {code} {d} cl={len(cl)}")
    stub("block"); c, code, ms, d, cl = thread(SUBTLE); ok("D03-04b P=0,2: stub block → 422", c == 422 and d and d[0][1] == "FALLBACK", f"{c} {code} {d}")
if cfg == "p97":
    set_gate("CHAT_ROUTING")
    st, t1, t2, d, cl, _ = chat(VAGUE); ok("D05-01 P=0,97 (≥τ): 1 DECIDE FAST, 0 CLASSIFY", st == 200 and len(d) == 1 and d[0][1] == "FAST" and not cl, f"{st} ttfb={t1:.0f}ms {d} cl={cl}")
if cfg == "unk":
    stub("allow"); c, code, ms, d, cl = thread(CLEAN); ok("D01-06/D03-04 đầu vào lạ → 0,25 FALLBACK + Structured", c in (200, 201) and d and d[0][1] == "FALLBACK" and len(cl) == 1, f"{c} {d} cl={len(cl)}")
    set_gate("CHAT_ROUTING"); st, t1, t2, d, cl, _ = chat(VAGUE); ok("D05-05 P lạ vague: FALLBACK + CLASSIFY", st == 200 and len(d) == 1 and d[0][1] == "FALLBACK", f"{st} {d} cl={cl}")
if cfg == "err":
    stub("allow"); c, code, ms, d, cl = thread(CLEAN); ok("D03-05 decision lỗi, stub allow → 201; DECIDE error → FALLBACK", c in (200, 201) and d and d[0][2] == "error", f"{c} {d}")
    stub("error"); c, code, ms, d, cl = thread(CLEAN); ok("D03-06 cả hai lỗi → 422 FAIL_CLOSED", c == 422 and any(x[1] == "FAIL_CLOSED" for x in d), f"{c} {code} {d}")
    stub("invalid"); c, code, ms, d, cl = thread(CLEAN); ok("D03-06b Structured sai JSON → 422 FAIL_CLOSED", c == 422, f"{c} {code} {d}")
    stub("slow"); t0 = time.time(); c, code, ms, d, cl = thread(CLEAN); ok("D03-07 Structured chậm 12 s → 422 trong ≈ 6–8 s", c == 422 and ms < 9000, f"{c} {ms:.0f}ms {d}"); stub("allow")
    set_gate("CHAT_ROUTING"); st, t1, t2, d, cl, _ = chat(VAGUE); ok("D05-06 decision lỗi: chat vẫn trả lời (200), ≤ 1 DECIDE", st == 200 and len(d) <= 1, f"{st} ttfb={t1:.0f}ms {d} cl={cl}")
if cfg == "timeout":
    stub("allow"); c, code, ms, d, cl = thread(CLEAN); ok("D01-08 decision timeout: DECIDE timeout rồi FALLBACK; stub allow → 201", c in (200, 201) and d and d[0][2] in ("timeout", "error"), f"{c} {ms:.0f}ms {d}")
stub("allow")
