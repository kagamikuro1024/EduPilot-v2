# Smoke S10: TC-10-10/11/12 search_library không lộ EXAM_PAPER/ANSWER_KEY khi SV đang thi (lớp QD10A, chat thật qua 9router)
from pm import *
import hashlib, urllib.request, re as _re
import minipdf
def upfile(name, text, title, typ, mime="application/pdf"):
    b = minipdf.make(text.split("\n")); sha = hashlib.sha256(b).hexdigest()
    c, pre, _ = api("teacher", "POST", f"{P}/uploads/presign", {"purpose": "document", "filename": name, "mime_type": mime, "size_bytes": len(b), "sha256": sha}); assert c == 200, (c, pre)
    r = urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"]); urllib.request.urlopen(r, context=CTX).read()
    c, d, _ = api("teacher", "POST", f"{P}/uploads/complete", {"upload_id": pre["upload_id"], "title": title, "type": typ}); assert c in (200, 202), (c, d)
    return d["document"]["id"]
def wait_ready(ids):
    for _ in range(60):
        st = [sql(f"select status from documents where id='{i}'")[0][0] for i in ids]
        if all(s == "READY" for s in st): return st
        time.sleep(4)
    return st
def ask(u, text):
    c, s, _ = api(u, "POST", "/chat/sessions", {"course_id": C1}); r = api(u, "POST", f"/chat/sessions/{s['id']}/messages", {"content": text}, timeout=120)[1]
    return r if isinstance(r, str) else json.dumps(r, ensure_ascii=False)
def tokens(sse): return "".join(json.loads(m.group(1)).get("t", "") for m in _re.finditer(r"event: token\ndata: (\{.*?\})\n", sse))
def cites(sse):
    m = _re.search(r'event: done\ndata: (\{.*?\})\n\n', sse, _re.S); return json.loads(m.group(1)).get("citations", []) if m else []
SUF = str(int(time.time()))[-5:]
DE, KEY = f"ZQ-DE-THI-17-{SUF}", f"ZQ-DAP-AN-17-{SUF}"
d1 = upfile(f"zq-de-{SUF}.pdf", f"DE THI THU CHU DE MANG\nMa de: {DE}\nCau 1. Giao thuc nao thiet lap ket noi truoc khi truyen du lieu? A. UDP B. TCP C. ICMP D. ARP\n", f"{DE} đề thi", "EXAM_PAPER")
d2 = upfile(f"zq-key-{SUF}.pdf", f"DAP AN\nMa dap an: {KEY}\nCau 1: B (TCP)\n", f"{KEY} đáp án", "ANSWER_KEY")
sql(f"update documents set use_for_rag=true where id in ('{d1}','{d2}')")  # ANSWER_KEY visible để chứng minh chặn theo audience GRADING, không theo cờ hiển thị
print("docs", wait_ready([d1, d2]), sql(f"select d.type, c.audience, count(*) from content_chunks c join documents d on d.id=c.document_id where d.id in ('{d1}','{d2}') group by 1,2"))
# TC-10-10: ngoài giờ thi — SV hỏi tìm tài liệu
sql(f"update exams set status='CLOSED', publish_hold=true where course_id='{C1}' and status='OPEN'")
sql(f"update exam_attempts set status='GRADING', submit_reason='MANUAL', submitted_at=now() where course_id='{C1}' and status='IN_PROGRESS'")
a1 = ask("sv.kha", f"Tìm tài liệu {DE}"); a2 = ask("sv.kha", f"Tìm tài liệu {KEY}")
t1, t2 = tokens(a1), tokens(a2); c1_, c2_ = cites(a1), cites(a2)
print("Q1", t1[:160], [x.get("document_id") for x in c1_]); print("Q2", t2[:160], [x.get("document_id") for x in c2_])
AUD = sql("select d.type, c.audience, count(*) from content_chunks c join documents d on d.id=c.document_id where d.id in ('%s','%s') group by 1,2" % (d1, d2))
chk("TC-10-10", DE in t1 and KEY not in t2 and d2 not in json.dumps(c2_), f"ngoài giờ thi: hỏi mã đề → câu trả lời nêu EXAM_PAPER ({DE in t1}: «{t1[:90]!r}»); hỏi mã đáp án → trả lời «{t2[:80]!r}» (lộ mã đáp án: {KEY in t2}, ANSWER_KEY trong trích dẫn: {d2 in json.dumps(c2_)}); audience: {AUD}")
# TC-10-11/12: sv.kha đang có lượt bài thi PE IN_PROGRESS
now = __import__("datetime").datetime.now(__import__("datetime").timezone.utc)
mq = [q for q in bank() if q["type"] == "MCQ_SINGLE" and q["review_status"] == "APPROVED"][:3]
c, e, _ = api("teacher", "POST", f"{P}/exams", {"title": f"QC S10 PE {SUF}", "opens_at": (now + __import__("datetime").timedelta(days=3)).isoformat(), "closes_at": (now + __import__("datetime").timedelta(days=3, hours=2)).isoformat(), "duration_minutes": 60})
c, j, _ = api("teacher", "PUT", f"{P}/exams/{e['id']}/items", {"items": [{"question_id": q["id"], "points": "1"} for q in mq], "version": e["version"]})
sql(f"update exams set status='OPEN', opens_at=now()-interval '10 minutes', closes_at=now()+interval '2 hours', duration_minutes=60 where id='{e['id']}'")
import uuid as _u; XT = str(_u.uuid4()); ca, at, _ = api("sv.kha", "POST", f"{P}/exams/{e['id']}/attempts", {}, headers={"X-Exam-Tab": XT}); print("thi", ca)
b1 = ask("sv.kha", f"Tìm tài liệu {DE}"); b2 = ask("sv.kha", f"Tìm tài liệu {KEY}")
LOCK = "EXAM_IN_PROGRESS" in b1 and "EXAM_IN_PROGRESS" in b2
ct = json.dumps(cites(b1) + cites(b2)); tt = tokens(b1) + tokens(b2)
print("CHAT_LOCK", LOCK, b1[:120])
chk("TC-10-11 (chat)", ca in (200, 201) and d1 not in ct and d2 not in ct and DE not in tt and KEY not in tt, f"sv.kha đang thi (attempt {ca}; chat bị khoá EXAM_IN_PROGRESS: {LOCK}): hỏi mã đề / mã đáp án → trích dẫn có đề/đáp án: {d1 in ct}/{d2 in ct}; câu trả lời chứa mã: {DE in tt}/{KEY in tt}; trả lời: «{tt[:120]}»")
o1 = ask("sv.gioi", f"Tìm tài liệu {DE}"); oc = json.dumps(cites(o1))
chk("TC-10-12", DE in tokens(o1), f"SV khác (sv.gioi, không có lượt thi) trong lúc sv.kha đang thi: tìm được đề ({DE in tokens(o1)}); trả lời «{tokens(o1)[:100]}»")
# sau giờ thi: nộp/đóng → tìm lại được
cs, js, _ = api("sv.kha", "POST", f"{P}/exams/{e['id']}/attempts/{at['attempt']['id']}/submit", {}, idem=str(_u.uuid4()), headers={"X-Exam-Tab": XT}); print("nộp", cs, str(js)[:120])
time.sleep(1); a3 = ask("sv.kha", f"Tìm tài liệu {DE}"); print("sau thi:", DE in tokens(a3)); SAU = DE in tokens(a3)
chk("TC-10-11 (sau giờ thi)", SAU, f"sau khi nộp bài thi: sv.kha tìm lại được EXAM_PAPER qua chat: {SAU}")
sql(f"update exams set status='PUBLISHED', published_at=now(), publish_hold=false where id='{e['id']}'")
print("done chat")
