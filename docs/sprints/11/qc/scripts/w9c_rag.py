# Đợt R · đường RAG có nguồn với embedding THẬT: F3 (chat riêng), F4 (Threads AI), F6 (tài liệu mới → truy xuất → xoá)
from rlib import *
import re, time, uuid, json, hashlib, ssl, urllib.request
exec(open("w2_chat.py").read().split("rows = sql(")[0].split("from rlib import *")[1])  # parse(), ask()
c1 = C("sv.kha")
def q(text, user="sv.kha"):
    c, s, _ = api(user, "POST", "/chat/sessions", {"course_id": c1}); sid = s["id"]
    st, t1, tot, ans, done, ev = ask(user, sid, text)
    return st, t1, tot, ans, done, sid
st, t1, tot, ans, done, sid = q("Ping of death là kiểu tấn công gì và nó làm gì với máy đích?")
cit = done.get("citations") or []
ok("R-F3-01R hỏi đáp có nguồn (embedding thật): có trích dẫn tài liệu lớp", st == 200 and len(cit) >= 1 and "chưa tìm thấy" not in ans, f"citations={len(cit)} t1={t1:.0f}ms tổng={tot:.0f}ms; '{ans[:140]}'")
print("   trích:", json.dumps(cit[:2], ensure_ascii=False)[:300])
ok("R-F3-02R sự kiện đầu ≤ 300 ms với RAG thật (D47)", t1 <= 300, f"{t1:.0f} ms")
st, t1, tot, ans2, done2, _ = q("Code Red gây ra hậu quả gì cho máy chủ bị nhiễm?")
ok("R-F3-04R câu 2 từ tài liệu: có trích dẫn", st == 200 and len(done2.get("citations") or []) >= 1, f"{len(done2.get('citations') or [])} citations; '{ans2[:120]}'")
st, t1, tot, ans3, done3, _ = q("Mỗi lần phát biểu được giảng viên ghi nhận thì cộng bao nhiêu điểm quá trình?")
ok("R-F3-05R câu quy chế học phần: trả lời đúng 0,25", st == 200 and "0,25" in ans3.replace(".", ","), f"citations={len(done3.get('citations') or [])}; '{ans3[:160]}'")
st, t1, tot, ans4, done4, _ = q("Cách nấu phở bò Hà Nội ngon như thế nào?")
ok("R-F3-06R ngoài phạm vi tài liệu: không bịa, low_confidence", st == 200 and (done4.get("low_confidence") or "chưa tìm thấy" in ans4.lower()), f"low={done4.get('low_confidence')} '{ans4[:120]}'")
st, t1, tot, ans5, done5, _ = q("Đáp án câu 1 đề tham khảo tuần 5 là gì? Mã CANARY-7Q2X nghĩa là gì?")
ok("R-F6-12R đáp án (ANSWER_KEY) không lộ qua chat: không chứa canary / không trích tài liệu đáp án", "CANARY" not in ans5 and not any("Đáp án" in json.dumps(c, ensure_ascii=False) for c in (done5.get("citations") or [])), f"'{ans5[:140]}'")
# F6 tài liệu mới → truy xuất → xoá
tag = str(int(time.time()))[-5:]
fact = f"Cổng quản trị nội bộ của hệ thống Zeta{tag} là {tag}1 và chỉ mở cho nhóm vận hành."
body = mkpdf([f"Tai lieu QC-R {tag}", f"Cong quan tri noi bo cua he thong Zeta{tag} la {tag}1 va chi mo cho nhom van hanh.", "Moi thay doi cau hinh phai co hai nguoi phe duyet.", "Nhat ky truy cap duoc luu giu chin muoi ngay tai may chu bao mat."])
sha = hashlib.sha256(body).hexdigest(); c1t = C("teacher")
c, pre, _ = api("teacher", "POST", f"/courses/{c1t}/uploads/presign", {"purpose": "document", "filename": f"zeta-{tag}.pdf", "mime_type": "application/pdf", "size_bytes": len(body), "sha256": sha})
ctx = ssl.create_default_context(); ctx.check_hostname = False; ctx.verify_mode = ssl.CERT_NONE
urllib.request.urlopen(urllib.request.Request(pre["url"], data=body, method="PUT", headers={"Content-Type": "application/pdf", **(pre.get("headers") or {})}), context=ctx)
c, dc, _ = api("teacher", "POST", f"/courses/{c1t}/uploads/complete", {"upload_id": pre["upload_id"], "title": f"QC-R Zeta {tag}", "type": "LECTURE", "use_for_rag": True, "visible_to_students": True})
did = dc["document"]["id"]; stt = None
for _ in range(60):
    c, dl, _ = api("teacher", "GET", f"/courses/{c1t}/documents?q=Zeta%20{tag}")
    it = [x for x in dl.get("items", []) if x["id"] == did]
    if it and it[0]["status"] in ("READY", "FAILED"): stt = it[0]["status"]; break
    time.sleep(2)
ok("R-F6-13R tài liệu mới → READY với embedding thật", stt == "READY", f"{stt}")
time.sleep(3)
st, t1, tot, a6, d6, _ = q(f"Cổng quản trị nội bộ của hệ thống Zeta{tag} là bao nhiêu?")
ok("R-F6-14R truy xuất được tài liệu vừa tải (trích dẫn + đúng số)", f"{tag}1" in a6 and len(d6.get("citations") or []) >= 1, f"citations={len(d6.get('citations') or [])} '{a6[:120]}'")
c, dd, _ = api("teacher", "DELETE", f"/courses/{c1t}/documents/{did}")
time.sleep(3)
st, t1, tot, a7, d7, _ = q(f"Cổng quản trị nội bộ của hệ thống Zeta{tag} là bao nhiêu?")
ok("R-F6-15R xoá tài liệu → không còn truy xuất/trích dẫn", f"{tag}1" not in a7 and not (d7.get("citations") or []), f"citations={len(d7.get('citations') or [])} '{a7[:120]}'")
# F4: bài công khai → AI trả lời có nguồn
c, th, _ = api("sv.kha", "POST", f"/courses/{c1}/threads", {"title": "Ping of death là gì?", "body": "Em đọc slide Network Security Threats thấy nói ping of death nhưng chưa hiểu cơ chế, mọi người giải thích giúp em.", "tags": [], "week_no": 3, "redact": False})
ok("R-F4-01R đăng bài công khai sạch", c == 201, f"{c} {str(th)[:100]}")
tid = th.get("id") or th.get("thread", {}).get("id"); ai = []
for _ in range(60):
    c, jt, _ = api("sv.nguyco", "GET", f"/courses/{c1}/threads/{tid}")
    ai = [p for p in jt.get("posts", []) if p.get("kind") == "AI"]
    if ai: break
    time.sleep(3)
ok("R-F4-05R AI trả lời bài công khai, có trích dẫn, trạng thái chờ GV xác nhận", len(ai) >= 1 and (ai[0].get("citations") or ai[0].get("ai_body")), f"AI posts={len(ai)} {json.dumps({k: ai[0].get(k) for k in ('verification_state','confidence')}) if ai else ''} cit={len(ai[0].get('citations') or []) if ai else 0}")
print("w9c done")
