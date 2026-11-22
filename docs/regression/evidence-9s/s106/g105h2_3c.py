# D03-19 + D05-15: khoá trống / hết hạn mức / mạng lỗi (máy chủ giả HTTP không chạy) — Threads và chat
STUBS and [stop(p) for p in STUBS]; STUBS.clear()   # mạng lỗi = máy chủ giả tắt
import statistics
dprov("gemini-decision-1"); time.sleep(4); gate("THREADS_PRECHECK", True, [C]); gate("CHAT_ROUTING", True, [C]); redis_del("ep:llm:cb:*"); redis_del("ep:decision:real:*")
CLEAN = ("Mình hỏi về cách nộp bài tập tuần này ạ?", "Mình muốn hỏi hạn nộp bài tập tuần này và cách nộp qua hệ thống ạ?")
PRIV = ("Điểm của em bao nhiêu", "Cho em hỏi điểm giữa kỳ của em là bao nhiêu và em có bị cảnh báo học vụ không ạ?")
def chat20(label, n=20):
    ms = []; codes = []
    for i in range(n):
        u = "sv.kha" if i % 2 == 0 else "sv.gioi"; cs, sj, _ = api(u, "POST", "/chat/sessions", {"course_id": C})
        t = time.time(); st = qlib.sse(u, f"/chat/sessions/{sj['id']}/messages", {"content": "Cách trình bày báo cáo thế nào " + str(i) + label})[0]; ms.append((time.time() - t) * 1000); codes.append(st); time.sleep(2.5)
    return codes, round(statistics.median(ms)), round(sorted(ms)[int(0.95 * (len(ms) - 1))])
def posts(tag):
    r = []
    for k, (ti, bo) in (("sạch", CLEAN), ("riêng tư", PRIV)): c, j, ms = post("sv.kha", C, (f"QC105 {tag} {k} {time.time()}", bo)); r.append((k, c, (j or {}).get("code") if c != 201 else "")); 
    return r
res = {}
# baseline chat: cổng tắt
gate("CHAT_ROUTING", False, [C]); redis_del("ep:llm:cb:*"); res["tắt"] = chat20("t"); print("chat cổng tắt (mã đủ?, p50, p95 ms):", set(res["tắt"][0]), res["tắt"][1:], flush=True)
gate("CHAT_ROUTING", True, [C])
# A: khoá trống (model gemini-decision-1, không khoá DB, GEMINI_API_KEY rỗng)
s = now(); pa = posts("A"); ca = chat20("A"); time.sleep(3); na = sql(f"select count(*) from llm_audit where task='DECIDE' and created_at>='{s}'")[0][0]; print("A khoá trống: posts", pa, "DECIDE", na, "chat", set(ca[0]), ca[1:], flush=True)
# B: mạng lỗi (khoá DB có; máy chủ giả tắt)
sql(f"UPDATE llm_providers SET api_key_enc=NULL WHERE id='{PID}'"); dprov("gemini-decision-1", key="qc-db-key"); time.sleep(4); redis_del("ep:llm:cb:*"); s = now(); pb = posts("B"); cb = chat20("B"); time.sleep(3)
rows = sql(f"select status, coalesce(error_kind,''), count(*) from llm_audit where task='DECIDE' and created_at>='{s}' group by 1,2"); print("B mạng lỗi: posts", pb, "DECIDE", rows, "chat", set(cb[0]), cb[1:], flush=True)
# C: hết hạn mức (đặt bộ đếm ngày vượt trần 1000)
import datetime
day = (datetime.datetime.utcnow() + datetime.timedelta(hours=7)).strftime("%Y%m%d"); os.system(f"docker exec ep-qc-redis-1 redis-cli set ep:decision:real:{day} 100000 EX 3600 >/dev/null"); redis_del("ep:llm:cb:*"); s = now(); pc = posts("C"); cc = chat20("C"); time.sleep(3)
rows = sql(f"select status, coalesce(error_kind,''), count(*) from llm_audit where task='DECIDE' and created_at>='{s}' group by 1,2"); print("C hết hạn mức: posts", pc, "DECIDE", rows, "chat", set(cc[0]), cc[1:], flush=True)
# D: mất cả hai (Structured/CLASSIFY lỗi + decision lỗi)
redis_del("ep:decision:real:*"); redis_del("ep:llm:cb:*"); fake(error_rate=1.0); s = now(); pd = posts("D"); fake(error_rate=0.0)
print("D mất cả hai nhà cung cấp: posts", pd, "DECIDE", sql(f"select cascade_path, status from llm_audit where task='DECIDE' and created_at>='{s}'"), flush=True)
json.dump({"base": res["tắt"], "A": [pa, na, ca], "B": [pb, cb], "C": [pc, cc], "D": pd}, open("/Users/kuro/.config/edupilot-team/qc3-work/d0319.json", "w"), default=str)
sql(f"UPDATE llm_providers SET api_key_enc=NULL WHERE id='{PID}'")
