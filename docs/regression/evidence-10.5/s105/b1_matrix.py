# B1: ma trận bật/tắt cổng (D02-07/08/09), lớp 2 không DECIDE (D03-17, D05-18), vai (D03-18), sửa/bình luận/precheck (D03-11). Lớp riêng QD105A/B.
import sys; from q105 import *
A, B = cid_of("QD105A"), cid_of("QD105B"); SA = subs(A)
VAGUE = "Nói giúp mình về mấy cái bảo mật mạng căn bản trong môn này nhé"
TXT = lambda k: (f"Hỏi về ôn tập {k} {int(time.time()*1000)%100000}", f"Cho mình hỏi cách ôn tập môn an ninh mạng hiệu quả nhất trước kỳ thi cuối kỳ là gì vậy ạ? ({k})")
def work(user, cid, tag):
    """đăng bài + chat + chấm lại (chỉ lớp A); trả số DECIDE theo loại."""
    s = now_db(); out = {}
    c, j, ms = post_thread(user, cid, *TXT(tag)); out["thread"] = (c, err_code(j))
    st, t1, t2, body, _ = chat(user, cid, VAGUE); out["chat"] = st
    if cid == A and user == "sv.kha":
        out["grade"] = regrade(A, SA["sv.kha@edupilot.local"])[2]
    time.sleep(3)
    rows = decide_rows(s, cid)
    return out, rows
if sys.argv[1] == "m":
    fake_reset(); fake(decision_mode="always_no")
    ALL = ("THREADS_PRECHECK", "INJECTION_FLAG", "CHAT_ROUTING")
    # D02-07: hệ thống BẬT, lớp 1 chưa bật
    set_gates(A, on=(), system_on=ALL); out, rows = work("sv.kha", A, "s1")
    rec("TC-D02-07", "P" if not rows and out["thread"][0] == 201 and out["chat"] == 200 and out["grade"] == "DONE" else "F", f"hệ thống bật 3 cổng, lớp QD105A tắt: đăng bài {out['thread']}, chat {out['chat']}, chấm lại {out['grade']}; DECIDE của lớp = {len(rows)}")
    # D02-08: lớp BẬT, hệ thống TẮT
    set_gates(A, on=ALL, system_on=()); out, rows = work("sv.kha", A, "s2")
    rec("TC-D02-08", "P" if not rows and out["thread"][0] == 201 and out["chat"] == 200 and out["grade"] == "DONE" else "F", f"lớp bật 3 cổng, hệ thống tắt: đăng bài {out['thread']}, chat {out['chat']}, chấm lại {out['grade']}; DECIDE = {len(rows)}")
    # D02-09: cả hai bật ở lớp 1 (lớp 2 chỉ bật hệ thống)
    set_gates(A, on=ALL, system_on=ALL); out, rows = work("sv.kha", A, "s3")
    gates_seen = sorted({r[0] for r in rows}); outb, rowsb = work("sv38", B, "s3b")
    ok = gates_seen == sorted(ALL) and not rowsb and out["thread"][0] == 201
    rec("TC-D02-09", "P" if ok else "F", f"lớp 1 (cả hai bật): DECIDE {len(rows)} dòng, cổng {gates_seen}, {[ (r[1],r[2]) for r in rows]}; lớp 2 (chỉ hệ thống): đăng {outb['thread']}, chat {outb['chat']} → DECIDE {len(rowsb)}")
    print(out, outb)
