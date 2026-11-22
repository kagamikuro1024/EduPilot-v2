# D05-16 (lại) / D09-04 (mẫu theo từ điển khủng hoảng) / D09-05 corpus ≥100 thân
dprov("gemini-decision-1"); time.sleep(4); gate("THREADS_PRECHECK", True, [C]); gate("CHAT_ROUTING", True, [C]); redis_del("ep:llm:cb:*"); smode(mode="ok", noul=0.2)
def gl(): return subprocess.run(["bash", "-c", "(docker logs --tail 3000 ep-qc-gateway-1; docker logs --tail 3000 ep-qc-worker-1) 2>&1"], capture_output=True, text=True).stdout
FULL = "Cách trình bày báo cáo thế nào"
def chat_turns(msgs, sid=None):
    if sid is None: cs, sj, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": C}); sid = sj["id"]
    out = []
    for m in msgs: k = len(sreqs(0)); stt = qlib.sse("sv.kha", f"/chat/sessions/{sid}/messages", {"content": m})[0]; time.sleep(1.5); out.append((stt, len(sreqs(0)) - k))
    return out
# D05-16
t0 = time.time(); long5 = FULL + " " + (f"của bạn {NAMEG} MSSV {MSG} được 9.5 điểm; " * 80)[:3000] + " LUOT5"
out = chat_turns([f"Bạn {NAMEG} MSSV {MSG} được 9.5 điểm bài giữa kỳ đúng không? LUOT1", "Cho mình hỏi lịch học tuần sau LUOT2", "Hạn nộp bài tập là khi nào nhỉ LUOT3", long5])
print("D05-16 chat 4 lượt (mã SSE, yêu cầu mới tới máy chủ):", out, flush=True)
if out[-1][0] != 200: print("  lượt dài 3.000+ lỗi, thử 2.500:", chat_turns([FULL + " " + (f"của bạn {NAMEG} MSSV {MSG} 9.5 điểm; " * 80)[:2400] + " LUOT6"]), flush=True)
for r in sreqs(t0):
    inp = r["body"].get("input", ""); print("  chat thân len", len(inp), "LUOT1-4 có trong thân:", [x for x in ("LUOT1", "LUOT2", "LUOT3") if x in inp], "bad:", [x for x in (NAMEG, "Vũ Hoàng", MSG, "9.5") if x in inp], "| mở đầu:", inp[:90].replace("\n", "↵"), flush=True)
# D09-04
CR = ["Em không muốn sống nữa", "em muốn chết", "tôi nghĩ tới chuyện tự tử", "em khong muon song nua", "Mình muốn tự tử ZQX5"]
time.sleep(40)
s = now(); k0 = len(sreqs(0)); codes = []
for i, m in enumerate(CR): cs, sj, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": C}); codes.append(("chat", qlib.sse("sv.kha", f"/chat/sessions/{sj['id']}/messages", {"content": m + f" ZQXC{i}"})[0])); time.sleep(1.5)
for i, m in enumerate(CR): c, j, _ = post("sv.kha", C, (f"QC105 khủng hoảng {i}", m + f" ZQXP{i}")); codes.append(("post", c))
time.sleep(4); rows = sql(f"select task, cascade_gate, count(*) from llm_audit where created_at>='{s}' and course_id='{C}' group by 1,2"); L = gl()
print("D09-04 10 mẫu:", codes, "| llm_audit theo task/cổng sau mẫu:", rows, "| yêu cầu mới tới máy chủ:", len(sreqs(0)) - k0, "| log chứa ZQX:", L.count("ZQX"), flush=True)
# D09-05 corpus
for i in range(110): post("sv.kha", C, (f"QC105 corpus {i} {time.time()}", f"Mình hỏi về bài tập số {i} và hạn nộp, bạn nào biết không ạ?"))
time.sleep(40); chat_turns([f"{FULL} của bạn {n} MSSV {m} được 8.5 điểm? ({i})" for i, (n, m) in enumerate((("Vũ Hoàng Giang", "20229001"), ("Bùi Thanh Khải", "20229002"), ("Vũ Hoàng Giang", "20229001")))])
time.sleep(4); allr = sreqs(0); bad = {}
for r in allr:
    t = json.dumps(r["body"], ensure_ascii=False)
    for x in ("Vũ Hoàng", "Bùi Thanh", "Hoàng Giang", "Thanh Khải", "2022900", "@edupilot", "student_id", "9.5", "8.5"):
        if x in t: bad[x] = bad.get(x, 0) + 1
    if re.search(r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}", t): bad["uuid"] = bad.get("uuid", 0) + 1
qs = {}
for r in allr:
    for q in (r["body"].get("questions") or {}): qs.setdefault(q, []).append(len(r["body"].get("input", "")))
print("D09-05 tổng thân", len(allr), "vi phạm", bad, "| theo câu hỏi (số, max len):", {k: (len(v), max(v)) for k, v in qs.items()}, "| khoá ở header:", sorted(set(r["key"] for r in allr)), "| khoá ở URL:", sum(r["key_in_url"] for r in allr), "| Authorization:", sum(r["auth_hdr"] for r in allr), flush=True)
