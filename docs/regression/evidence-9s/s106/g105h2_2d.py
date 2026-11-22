# D09-05: ≥100 thân tới máy chủ giả từ chấm bài (INJECTION_FLAG) + Threads, mọi bài có tên đầy đủ/MSSV/email/điểm
dprov("gemini-decision-1"); time.sleep(4); inj(True); gate("THREADS_PRECHECK", True, [C]); gate("CHAT_ROUTING", True, [C]); redis_del("ep:llm:cb:*"); smode(mode="ok", noul=0.2)
essay = " ".join(["nội dung bài làm bình thường về kiểm thử phần mềm"] * 4)
tot = 0
for k in range(6):
    a = mk_assignment(f"g105h2d{k}", 20); subs = [x[0] for x in sql(f"select id from submissions where assignment_id='{a}' order by created_at")]
    sql("set session_replication_role=replica; " + "; ".join(f"update submissions set extracted_text=$$Họ tên: {NAMEK}\nMSSV: {MSK}\n{essay}. Liên hệ {NAMEG} MSSV {MSG} email sv.gioi@edupilot.local SĐT 0912345678, điểm {9.5 if i % 2 else 8.5}.$$ where id='{sid}'" for i, sid in enumerate(subs)))
    c, j, nn = grade_all_wait(a, 240); tot += len(subs); print("lô", k, "chấm", c, nn, "thân tới máy chủ", len(sreqs(0)), flush=True)
time.sleep(4); allr = sreqs(0); bad = {}
for r in allr:
    t = json.dumps(r["body"], ensure_ascii=False)
    for x in ("Vũ Hoàng", "Bùi Thanh", "Hoàng Giang", "Thanh Khải", "2022900", "@edupilot", "student_id", "9.5", "8.5", "0912345678", "Họ tên"):
        if x in t: bad[x] = bad.get(x, 0) + 1
    if re.search(r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}", t): bad["uuid"] = bad.get("uuid", 0) + 1
ph = sum(1 for r in allr if "[[SV_" in json.dumps(r["body"], ensure_ascii=False))
qs = {}
for r in allr: 
    for q in (r["body"].get("questions") or {}): qs.setdefault(q, []).append(len(r["body"].get("input", "")))
print("D09-05 tổng thân", len(allr), "vi phạm", bad, "| thân có placeholder [[SV_n]]:", ph, "| theo câu hỏi (số, max len):", {k: (len(v), max(v)) for k, v in qs.items()}, "| khoá header:", sorted(set(r["key"] for r in allr)), "URL:", sum(r["key_in_url"] for r in allr), "Authorization:", sum(r["auth_hdr"] for r in allr), "| pii_masked_count min/max:", sql(f"select min(pii_masked_count), max(pii_masked_count), count(*) from llm_audit where task='DECIDE' and cascade_gate='INJECTION_FLAG' and created_at>=now()-interval '15 minutes'"), flush=True)
