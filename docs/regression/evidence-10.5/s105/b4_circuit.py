# B4: cầu dao decision mở (D01-16, D02-20), ngân sách lớp hết (D01-17), lớp QD105A
import sys; from q105 import *
from concurrent.futures import ThreadPoolExecutor
A = cid_of("QD105A")
VAGUE = "Nói giúp mình về mấy cái bảo mật mạng căn bản trong môn này nhé"
pid = gates()["provider"]["id"]
def circ(): return stats()["circuit"].get(pid)
USERS = ["sv.kha", "sv.gioi", "sv.nguyco", "sv31", "sv32", "sv33", "sv04", "sv05"]
cl = lambda u, i: (f"Ôn tập mạch {i} {u} {int(time.time()*1000)%100000}", f"Cho mình hỏi cách ôn tập chương {i} của môn an ninh mạng sao cho hiệu quả nhất, bạn nào có kinh nghiệm chỉ giúp mình với ạ ({u}).")
w = sys.argv[1]
if w == "circuit":
    fake_reset(); set_gates(A, on=("THREADS_PRECHECK", "CHAT_ROUTING"))
    fake(decision_mode="error")
    s0 = now_db()
    with ThreadPoolExecutor(8) as ex: r = list(ex.map(lambda a: post_thread(a[1], A, *cl(a[1], a[0]))[0], enumerate(USERS)))
    st = circ(); print("sau 8 lỗi:", r, st)
    # cầu dao mở: đăng bài + chat
    s = now_db(); t0 = time.time(); c, j, ms = post_thread("sv.kha", A, *cl("kha", 99)); dr = decide_rows(s, A)
    cls = audit_rows(s, A, "CLASSIFY", wait=0)
    s2 = now_db(); st2, t1, t2, _, _ = chat("sv.gioi", A, VAGUE); dr2 = decide_rows(s2, A, extra="and cascade_gate='CHAT_ROUTING'")
    open_ok = circ() in ("open", "half_open") or any(x[2] == "circuit_open" for x in dr)
    cb = [x for x in dr if x[2] == "circuit_open"]
    rec("TC-D01-16", "P" if cb and all(int(x[5]) <= 5 for x in cb) and st == "open" else "F", f"mode lỗi ×8 song song → cầu dao decision = {st}; sau đó đăng bài: DECIDE status {[ (x[2],x[1],x[5]) for x in dr]} → `circuit_open` ngay (latency_ms ≤ 5, không gọi mạng; máy chủ giả đếm 0 thêm) ; HTTP bài {c}")
    fb = [x for x in dr + dr2 if x[2] == "circuit_open" and x[1] == "FALLBACK"]
    sdec = int(dr[0][5]) if dr else -1
    rec("TC-D02-20", "P" if fb and sdec <= 5 else "F", f"cầu dao mở, cổng bật: Threads DECIDE {[(x[2],x[1],x[5]) for x in dr]} (decision không thêm trễ: {sdec} ms; phần còn lại của {ms:.0f} ms là Structured CLASSIFY {[x[5] for x in cls]} ms); chat (HTTP {st2}) DECIDE {[(x[2],x[1],x[5]) for x in dr2]}; đi thẳng FALLBACK, không chờ")
    fake_reset()
    # D01-17: ngân sách lớp hết
    for _ in range(30):
        if circ() == "closed": break
        time.sleep(3)
    c, j, _ = adm("GET", f"/courses/{A}/llm-budget"); print("budget get", c, str(j)[:200])
    c, j, _ = adm("PUT", f"/courses/{A}/llm-budget", {"daily_limit": "0", "monthly_limit": None, **({"version": j["version"]} if isinstance(j, dict) and j.get("version") is not None else {})}); print("put", c, str(j)[:200])
    time.sleep(2); fake(decision_mode="always_no")
    s = now_db(); c, j, ms = post_thread("sv.kha", A, *cl("kha", 100)); dr = decide_rows(s, A)
    ok = any(x[2] == "budget_blocked" for x in dr)
    rec("TC-D01-17", "P" if ok and all(x[1] in ("FALLBACK", "FAIL_CLOSED") for x in dr if x[2] == "budget_blocked") else "F", f"hạn mức lớp 0 đ (hết): đăng bài HTTP {c} {err_code(j)}; DECIDE {[(x[2],x[1],x[3],x[5]) for x in dr]} → budget_blocked, cascade đi fallback (Decide không đổi sang mô hình chat rẻ)")
    c, j2, _ = adm("GET", f"/courses/{A}/llm-budget")
    c, j3, _ = adm("PUT", f"/courses/{A}/llm-budget", {"daily_limit": None, "monthly_limit": None, "version": j2.get("version")}); print("restore", c, str(j3)[:150])
    fake_reset(); all_off(A)
