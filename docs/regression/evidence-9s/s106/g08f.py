import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10")); import qlib
Q0 = "Cách phân biệt tấn công chuỗi cung ứng với tấn công trực tiếp là gì"; SIM = f"/courses/{CG0}/threads/similar-confirmed"
def run():
    for t in ("CHAT", "CLASSIFY", "UTILITY"): set_route(t, "QC fake", "fake-chat")
    flag("THREAD_SUGGEST", True)
    try:
        cs, sj, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": CG0}); sid = sj["id"]
        st = qlib.sse("sv.kha", f"/chat/sessions/{sid}/messages", {"content": Q0 + " ZQTK91A"})[0]; time.sleep(2)
        msgs = api("sv.kha", "GET", f"/chat/sessions/{sid}/messages")[1]; msgs = msgs["items"] if isinstance(msgs, dict) else msgs
        am = [m for m in msgs if m["role"] == "ASSISTANT" and (m.get("can_escalate") or m.get("ticket"))]
        print("chat", st, "tin trợ lý có thể chuyển:", len(am))
        ec, ej, _ = api("sv.kha", "POST", f"/chat/messages/{am[0]['id']}/escalate", {}); tk = ej["ticket"]; print("escalate", ec, tk["status"])
        cl = api("teacher", "POST", f"/courses/{CG0}/tickets/{tk['id']}/claim", {})[0]
        ANS = "ZQ-TK-91: tấn công chuỗi cung ứng nhắm vào nhà cung cấp phần mềm trung gian, khác với tấn công trực tiếp vào hệ thống đích."
        ac = api("teacher", "POST", f"/courses/{CG0}/tickets/{tk['id']}/answer", {"body": ANS}, idem=str(uuid.uuid4()))[0]
        kc, kj, _ = api("teacher", "POST", f"/courses/{CG0}/tickets/{tk['id']}/knowledge", {"question": Q0 + "?", "answer": ANS}); print("claim", cl, "answer", ac, "knowledge", kc, str(kj)[:120])
        tc, tj, _ = api("sv.kha", "POST", f"/courses/{CG0}/threads", {"title": Q0 + " ? QCG08F" + str(int(time.time()))[-4:], "body": "Em muốn hiểu rõ hơn về chuỗi cung ứng phần mềm."}, idem=str(uuid.uuid4())); T0 = tj["thread"]["id"]
        sql(f"insert into forum_posts (course_id, thread_id, kind, body, verification_state, citations, confidence) select '{CG0}', '{T0}', 'AI', 'Tấn công chuỗi cung ứng nhắm vào nhà cung cấp phần mềm trung gian, khác với tấn công trực tiếp vào hệ thống đích.', 'PENDING', '[]', 0.7")
        pid = sql(f"select id from forum_posts where thread_id='{T0}' and kind='AI'")[0][0]; vc = api("teacher", "POST", f"/courses/{CG0}/posts/{pid}/verify", {})[0]
        time.sleep(35)
        it = sql(f"select k.id, k.source_type, c.id, (c.embedding is not null), left(regexp_replace(c.text, '[[:space:]|]+', ' ', 'g'), 600) from knowledge_items k join content_chunks c on c.id=k.chunk_id where k.source_type='TICKET' and k.source_id='{tk['id']}'")
        print("mục kho TICKET:", [(x[1], x[3]) for x in it])
        pch = sql(f"select c.id, left(regexp_replace(c.text, '[[:space:]|]+', ' ', 'g'), 1500) from knowledge_items k join content_chunks c on c.id=k.chunk_id where k.source_type='POST' and k.source_id='{pid}'")[0]
        sql(f"update content_chunks set embedding=(select embedding from content_chunks where id='{pch[0]}') where id='{it[0][2]}'")  # đối chứng: ép mục TICKET giống hệt mục POST (cos = 1) để chứng minh bộ lọc loại TICKET
        qtxt = pch[1]
        res = []
        for q in (qtxt, Q0 + "?", "ZQ-TK-91 tấn công chuỗi cung ứng"):
            c, j, _ = api("sv.kha", "POST", SIM, {"q": q}); res.append((c, len(j), "ZQ-TK-91" in json.dumps(j, ensure_ascii=False)))
        pos = sql(f"select count(*) from knowledge_items k join forum_posts p on p.id=k.source_id where k.source_type='POST' and k.course_id='{CG0}'")[0][0]
        cosv = sql(f"select 1 - (embedding <=> (select embedding from content_chunks where id='{it[0][2]}')) from content_chunks c where c.id=(select c2.id from knowledge_items k2 join content_chunks c2 on c2.id=k2.chunk_id where k2.source_type='POST' and k2.course_id='{CG0}' limit 1)")[0][0] if it else None
        ids_ = [x["thread_id"] for x in api("sv.kha", "POST", SIM, {"q": qtxt})[1]]; cosp = sql(f"select 1 - (a.embedding <=> b.embedding) from content_chunks a, content_chunks b where a.id='{it[0][2]}' and b.id=(select k.chunk_id from knowledge_items k where k.source_type='POST' and k.source_id='{pid}')")
        print("đối chứng: thread POST mới", T0 in ids_, "cos(TICKET, POST)", cosp, "verify", vc, "| similar-confirmed (mã, số mục, lộ ZQ-TK-91):", res, "| mục POST trong lớp:", pos)
        chk("TC-G08-06", bool(it) and it[0][3] == "t" and T0 in ids_ and all(r[0] == 200 and not r[2] for r in res), f"luồng thật: SV hỏi riêng (chat, provider giả) → escalate (ticket {tk['status']}) → GV nhận + trả lời có chữ «ZQ-TK-91» → «Lưu vào kho tri thức» (POST /tickets/{{id}}/knowledge {kc}) → mục kho source_type TICKET có embedding (embedding ép bằng SQL = embedding của mục POST đối chứng, cos 1); POST similar-confirmed với: chính nội dung mục (cos≈1), câu hỏi gốc, «ZQ-TK-91 …» → {[(r[0], r[1]) for r in res]} mục; chữ «ZQ-TK-91» có trong phản hồi: {[r[2] for r in res]}; đối chứng: thread POST nội dung gần giống (cos {cosp}) CÓ trong gợi ý = {T0 in ids_}, mục TICKET không (Q-G-5)")
    finally:
        for t in ("CHAT", "CLASSIFY", "UTILITY"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
with window(): run()
