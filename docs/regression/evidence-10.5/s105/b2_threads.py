# B2: Threads — lớp 2 không DECIDE (D03-17), vai (D03-18), sửa/bình luận/precheck (D03-11), song song (D03-20), chat race (D05-19), cổng chat tắt/lớp 2 (D05-17/18), làn (D01-11)
import os
import sys, os; from q105 import *
from concurrent.futures import ThreadPoolExecutor
A, B = cid_of("QD105A"), cid_of("QD105B")
PRIV = [json.loads(l) for l in open(os.path.expanduser("~/Documents/TA_Agent_qc106/benchmarks/decision/data/threads.jsonl")) if '"private"' in l]; PRIV = [r for r in PRIV if not r['body'].startswith('Cậu bạn hay ngồi cạnh')]  # 1/10 bài «private» bị luật bắt sẵn (PERSONAL_QUESTION): bỏ khỏi tập qua-luật
_n = [0]
def SUBTLE(k):
    _n[0] += 1; r = PRIV[_n[0] % len(PRIV)]
    return (f"{r['title']} ({k}{int(time.time()*1000)%100000})", r["body"])
VAGUE = "Nói giúp mình về mấy cái bảo mật mạng căn bản trong môn này nhé"
SPECIFIC = "Điểm chuyên cần của tôi bao nhiêu rồi?"
w = sys.argv[1]
if w == "t":
    fake_reset(); fake(decision_mode="always_yes")
    set_gates(A, on=("THREADS_PRECHECK",))
    T0 = SUBTLE("a")
    s = now_db(); c1, j1, _ = post_thread("sv.kha", A, *T0); r1 = decide_rows(s, A)
    s2 = now_db(); c2, j2, _ = post_thread("sv38", B, *T0); r2 = decide_rows(s2, B)
    ok = c1 == 422 and len(r1) == 1 and r1[0][:3] == ["THREADS_PRECHECK", "FAST", "ok"] and c2 == 201 and not r2
    rec("TC-D03-17", "P" if ok else "F", f"cổng chỉ lớp QD105A, decision giả luôn «có»: sv.kha lớp 1 → {c1} {err_code(j1)}, DECIDE {[x[:3] for x in r1]}; sv38 lớp 2 cùng nội dung → {c2}, DECIDE {len(r2)} (đường cũ)")
    res = {}
    for u in ("teacher", "ta"):
        s = now_db(); c, j, _ = post_thread(u, A, *T0[:0], *(T0[0] + " " + u, T0[1])); r = decide_rows(s, A)
        res[u] = (c, err_code(j), [x[:3] for x in r])
    ok = all(v[0] == 422 and v[2] == [["THREADS_PRECHECK", "FAST", "ok"]] for v in res.values())
    rec("TC-D03-18", "P" if ok else "F", f"teacher@, ta@ lớp 1 đăng cùng bài (decision giả «có»): {res} → đi qua cổng như SV")
    # D03-11: sửa + bình luận qua cổng; precheck thì không
    fake(decision_mode="always_no"); c, j, _ = post_thread("sv.kha", A, *SUBTLE("e")); assert c == 201, (c, j)
    tid = (j.get("thread") or j)["id"]; ver = (j.get("thread") or j).get("version", 1)
    fake(decision_mode="always_yes")
    s = now_db(); ce, je, _ = api("sv.kha", "PUT", f"/courses/{A}/threads/{tid}", {"title": "Sửa chuyện lớp mình", "body": SUBTLE("sua")[1], "version": ver}); re_ = decide_rows(s, A)
    s = now_db(); cc, jc, _ = api("sv.gioi", "POST", f"/courses/{A}/threads/{tid}/posts", {"body": SUBTLE("bl")[1]}); rc = decide_rows(s, A)
    s = now_db(); cp, jp, _ = api("sv.kha", "POST", f"/courses/{A}/threads/precheck", {"title": SUBTLE("p")[0], "body": SUBTLE("p")[1]}); rp = decide_rows(s, A)
    ok = ce == 422 and len(re_) == 1 and cc == 422 and len(rc) == 1 and cp == 200 and not rp
    rec("TC-D03-11", "P" if ok else "F", f"đăng ok (decision «không») rồi bật «có»: PUT sửa bài → {ce} {err_code(je)} DECIDE {len(re_)}; bình luận → {cc} {err_code(jc)} DECIDE {len(rc)}; precheck → {cp} DECIDE {len(rp)} (precheck không qua decision)")
    fake_reset()
if w == "p":
    # D03-20: 10 request song song khi decision chậm (p50 1.500 ms > hạn 800 ms của cổng)
    fake_reset(); fake(decision_p50_ms=1500, decision_p95_ms=2000); set_gates(A, on=("THREADS_PRECHECK",))
    users = ["sv.kha", "sv.gioi", "sv.nguyco", "sv31", "sv32", "sv33", "sv04", "sv05", "sv06", "sv07"]
    for u in users: tok(u)
    clean = lambda u, i: (f"Hỏi ôn tập chương {i} {u} {int(time.time()*1000)%100000}", f"Mình muốn hỏi cách ôn tập chương {i} của môn an ninh mạng sao cho hiệu quả, bạn nào có kinh nghiệm chỉ giúp với ạ ({u}).")
    s = now_db(); t0 = time.time()
    def one(a):
        i, u = a; ti = clean(u, i); c, j, ms = post_thread(u, A, *ti, idem=f"qc105-par-{i}-{int(t0)}"); return (u, ti[0], c, err_code(j), round(ms))
    with ThreadPoolExecutor(10) as ex: res = list(ex.map(one, enumerate(users)))
    wall = time.time() - t0
    rows = decide_rows(s, A, wait=6)
    titles = [r[1] for r in res]; n201 = sum(r[2] == 201 for r in res)
    created = int(sql(f"select count(distinct title) from forum_threads where course_id='{A}' and created_at>='{s}'")[0][0])
    total = int(sql(f"select count(*) from forum_threads where course_id='{A}' and created_at>='{s}'")[0][0])
    ok = all(r[2] in (201, 422) for r in res) and total == n201 == created and len(rows) == 10 and all(r[1] == "FALLBACK" for r in rows) and wall < 60
    rec("TC-D03-20", "P" if ok else "F", f"10 đăng song song, decision giả p50 1.500 ms (hạn cổng 800 ms): kết quả {sorted(set((r[2]) for r in res))} (201×{n201}), threads tạo {total} = số 201, tiêu đề duy nhất {created}, DECIDE {len(rows)} dòng {sorted(set((r[1], r[2], r[3]) for r in rows))}, tổng {wall:.1f}s, từng request {min(r[4] for r in res)}–{max(r[4] for r in res)} ms; không deadlock")
    print(res)
    fake_reset()
    # D05-17 / D05-18 / D05-19
    set_gates(A, on=("THREADS_PRECHECK", "INJECTION_FLAG"), system_on=("THREADS_PRECHECK", "INJECTION_FLAG", "CHAT_ROUTING"))
    s = now_db(); st, _, _, _, _ = chat("sv.kha", A, VAGUE); r = decide_rows(s, A)
    rec("TC-D05-17", "P" if st == 200 and not r else "F", f"cổng CHAT_ROUTING tắt ở lớp (hệ thống bật, 2 cổng kia bật): chat mơ hồ → HTTP {st}, DECIDE = {len(r)}")
    set_gates(A, on=("CHAT_ROUTING",), system_on=("CHAT_ROUTING",))
    s = now_db(); st, _, _, _, _ = chat("sv38", B, VAGUE); r2 = decide_rows(s, B); r1 = decide_rows(s, A, wait=0)
    rec("TC-D05-18", "P" if st == 200 and not r2 and not r1 else "F", f"CHAT_ROUTING bật chỉ ở QD105A: sv38 (lớp 2) hỏi cùng tin mơ hồ → HTTP {st}, DECIDE lớp 2 = {len(r2)}, lớp 1 trong khoảng đó = {len(r1)}")
    # race: 5 cặp song song (tin mơ hồ ↔ tin cụ thể), mỗi tin một người; kiểm DECIDE theo user_id
    vag = ["sv.gioi", "sv.nguyco", "sv31", "sv32", "sv33"]; spec = ["sv04", "sv05", "sv06", "sv07", "sv08"]
    s = now_db()
    def go(a):
        u, t = a; return (u, chat(u, A, t)[0])
    jobs = [(u, VAGUE) for u in vag] + [(u, SPECIFIC) for u in spec]
    with ThreadPoolExecutor(10) as ex: res = list(ex.map(go, jobs))
    time.sleep(1)
    per = {r[0]: int(r[1]) for r in sql(f"select u.email,count(*) from llm_audit a join users u on u.id=a.user_id where a.task='DECIDE' and a.course_id='{A}' and a.created_at>='{s}' group by 1") } if False else {}
    rows = sql(f"select split_part(u.email,'@',1),count(*) from llm_audit a join users u on u.id=a.user_id where a.task='DECIDE' and a.course_id='{A}' and a.created_at>='{s}' group by 1")
    time.sleep(3); rows = sql(f"select split_part(u.email,'@',1),count(*) from llm_audit a join users u on u.id=a.user_id where a.task='DECIDE' and a.course_id='{A}' and a.created_at>='{s}' group by 1")
    per = {r[0]: int(r[1]) for r in rows}
    ok = all(c == 200 for _, c in res) and all(per.get(u) == 1 for u in vag) and all(u not in per for u in spec)
    rec("TC-D05-19", "P" if ok else "F", f"10 tin gửi cùng lúc (5 mơ hồ, 5 cụ thể — mỗi tin một SV/phiên): HTTP {sorted(set(c for _, c in res))}; DECIDE theo người: mơ hồ {[per.get(u, 0) for u in vag]}, cụ thể {[per.get(u, 0) for u in spec]} (đúng 1 / 0): không lẫn nhãn giữa phiên")
    all_off(A)
