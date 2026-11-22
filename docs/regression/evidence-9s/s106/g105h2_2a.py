# D01-04 / D01-15 / D02-05 / D01-10 / D03-16
gate("THREADS_PRECHECK", True, [C])
def gl(): return subprocess.run(["bash", "-c", "(docker logs --tail 3000 ep-qc-gateway-1; docker logs --tail 3000 ep-qc-worker-1) 2>&1"], capture_output=True, text=True).stdout
print("provider khoá DB →", dprov("gemini-decision-1", key="qc-db-key")[0], flush=True); time.sleep(4); redis_del("ep:llm:cb:*")
t0 = time.time(); [newpost(100 + i) for i in range(2)]; k1 = [r["key"] for r in sreqs(t0)]
sql(f"UPDATE llm_providers SET api_key_enc=NULL WHERE id='{PID}'"); r = dprov("gemini-decision-1"); time.sleep(4); t1 = time.time(); [newpost(110 + i) for i in range(2)]; k2 = [r2["key"] for r2 in sreqs(t1)]
L = gl(); apij = json.dumps(api("admin", "GET", "/admin/llm/providers")[1])
na = sql("select count(*) from llm_audit a where a::text like '%qc-db-key%' or a::text like '%test-key%'")[0][0]
print("D01-04 khoá DB:", k1, "| xoá khoá DB (SQL UPDATE llm_providers SET api_key_enc=NULL + PUT lưu lại →", r[0], ") → khoá gửi đi:", k2, "| log gateway+worker chứa qc-db-key:", L.count("qc-db-key"), "test-key:", L.count("test-key"), "| API providers chứa:", apij.count("qc-db-key"), apij.count("test-key"), "| llm_audit chứa:", na, flush=True)
# D01-15
res = {}
for mode in ("429", "500", "safety", "hang"):
    smode(mode=mode); redis_del("ep:llm:cb:*"); s = now(); t = time.time(); c, j, ms = newpost(200); res[mode] = (c, ms, st(s))
    print("D01-15", mode, c, ms, "ms", res[mode][2], flush=True)
smode(mode="ok"); redis_del("ep:llm:cb:*"); [stop(p) for p in STUBS]; STUBS.clear(); time.sleep(1); s = now(); c, j, ms = newpost(201); res["mạng"] = (c, ms, st(s)); print("D01-15 lỗi mạng (máy chủ giả tắt)", c, ms, "ms", res["mạng"][2], flush=True)
STUBS.append(start_stub()); time.sleep(1); redis_del("ep:llm:cb:*")
print("D01-15 SELECT DISTINCT status:", sql("select distinct status from llm_audit order by 1"), flush=True)
# D02-05
smode(mode="safety"); redis_del("ep:llm:cb:*"); s = now(); c, j, ms = newpost(300, title="QC105 an toàn 300"); d = st(s)
print("D02-05 safety → post", c, {k: j.get(k) for k in list(j)[:10]} if isinstance(j, dict) else j, d, flush=True)
smode(mode="ok"); redis_del("ep:llm:cb:*")
# D01-10 / D03-16 (tường lửa Threads chặn MSSV/email/SĐT/tên đầy đủ bằng 422 PII_DETECTED trước DECIDE; chỉ tên rút gọn lọt qua tới DECIDE)
ev0 = int(sql("select count(*) from pii_events")[0][0]); smode(mode="ok"); redis_del("ep:llm:cb:*")
for i, body in enumerate(["Khải ơi và Giang ơi, cho mình hỏi hạn nộp bài tuần này nhé", "Hỏi hạn nộp bài: bạn Khải và bạn Giang có biết không?"]):
    t0 = time.time(); s = now(); c, j, ms = newpost(400 + i, body, f"QC105 PII {400+i}"); d = drows2(s, "THREADS_PRECHECK"); rq = sreqs(t0)
    txt = json.dumps([r["body"] for r in rq], ensure_ascii=False); bad = [x for x in ("Khải", "Giang", NAMEK, MSK, NAMEG, MSG) if x in txt]
    print("D01-10 post", c, "yêu cầu tới máy chủ", len(rq), "chữ cấm trong thân:", bad, "pii_masked_count", [x[5] for x in d], "| input:", (rq[0]["body"].get("input") if rq else None), flush=True)
e1 = int(sql("select count(*) from pii_events")[0][0]); c2, j2, _ = newpost(410, f"Mình là {NAMEK} MSSV {MSK}", "QC105 PII 410")
print("D03-16 pii_events thêm (2 bài lọt)", e1 - ev0, "| bài MSSV bị chặn:", c2, "pii_events thêm", int(sql("select count(*) from pii_events")[0][0]) - e1, "| cột pii_events:", [x[0] for x in sql("select column_name from information_schema.columns where table_name='pii_events'")], "| cột chữ của llm_audit:", [x[0] for x in sql("select column_name from information_schema.columns where table_name='llm_audit' and data_type in ('text','jsonb')")], flush=True)
