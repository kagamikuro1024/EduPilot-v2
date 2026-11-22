# D05-16 (lại): lượt 5 mơ hồ + dài; lịch sử lượt 1 có MSSV người khác + điểm
dprov("gemini-decision-1"); time.sleep(4); gate("CHAT_ROUTING", True, [C]); redis_del("ep:llm:cb:*"); smode(mode="ok", noul=0.2)
cs, sj, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": C}); sid = sj["id"]
def turn(m): k = len(sreqs(0)); stt = qlib.sse("sv.kha", f"/chat/sessions/{sid}/messages", {"content": m})[0]; time.sleep(2); return stt, len(sreqs(0)) - k
pad = ("Mình thấy phần này hơi khó hiểu và chưa rõ ràng lắm đối với mình " * 60)
res = []
for m in (f"Bạn {NAMEG} MSSV {MSG} được 9.5 điểm bài giữa kỳ đúng không? LUOT1", "Cho mình hỏi lịch học tuần sau LUOT2", "Hạn nộp bài tập là khi nào nhỉ LUOT3", "Cách trình bày báo cáo thế nào LUOT4"): res.append(turn(m))
for ln in (3000, 4000, 4900): 
    r = turn(("Cái kia thì sao nhỉ " + pad)[:ln] + " LUOT5-" + str(ln)); res.append((ln,) + r)
print("D05-16 lượt (mã SSE, yêu cầu mới):", res, flush=True)
for r in sreqs(0):
    inp = r["body"].get("input", ""); print("  chat thân len", len(inp), "lịch sử LUOT1-3:", [x for x in ("LUOT1", "LUOT2", "LUOT3", "LUOT4") if x in inp], "bad:", [x for x in (NAMEG, "Vũ Hoàng", MSG, "9.5") if x in inp], "| đầu:", inp[:60].replace("\n", "↵"), "| cuối:", inp[-22:], flush=True)
print("pii_masked_count CHAT_ROUTING:", sql("select pii_masked_count, status from llm_audit where task='DECIDE' and cascade_gate='CHAT_ROUTING' order by created_at desc limit 4"), flush=True)
