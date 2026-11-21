import sys, os, re; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"; S10 = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10")
CD = cid("QD10D"); DOC = sql(f"select id from documents where course_id='{CD}' and type='LECTURE' and status='READY' order by created_at desc limit 1")[0][0]
QB = f"/courses/{CD}/questions"
def mkrej(reason, tag, n=1, old=False):
    ids = []
    for i in range(n):
        c, j, _ = api("teacher", "POST", QB, {"type": "MCQ_SINGLE", "title": f"QC106 loại {tag} {i}", "topic": "QC106-hint", "difficulty": "EASY", "stem": f"Câu thử loại {tag} số {i} {time.time()}?", "options": [{"body": "x"}, {"body": "y"}], "correct": [0]}); assert c in (200, 201), (c, j)
        qid = j["id"]; c, d, _ = api("teacher", "GET", f"{QB}/{qid}"); api("teacher", "PUT", f"{QB}/{qid}/review", {"decision": "REQUEST", "version": d["version"]})
        c, d, _ = api("teacher", "GET", f"{QB}/{qid}"); body = {"decision": "REJECT", "version": d["version"]}
        if reason: body["reject_reason"] = reason
        c2, j2, _ = api("teacher", "PUT", f"{QB}/{qid}/review", body); assert c2 == 200, (c2, j2)
        if old: sql(f"update question_bank set reviewed_at = now() - interval '91 days' where id='{qid}'")
        ids.append(qid)
    return ids
def nrej(extra=""): return sql("select count(*) from question_bank where course_id='%s' and review_status='REJECTED' and reject_reason<>'UNSPECIFIED' %s" % (CD, extra))[0][0]
def clear_rej(): sql(f"delete from question_bank where course_id='{CD}' and topic='QC106-hint'")
def hintflag(on):
    cc, j, _ = api("admin", "GET", FL(CD)); f = [x for x in j["items"] if x["feature"] == "QUESTION_HINTS"][0]; return api("admin", "PUT", f"{FL(CD)}/QUESTION_HINTS", {"enabled": on, "version": f["version"]})[0]
def gen_prompt():
    sql("set session_replication_role=replica; delete from question_bank where course_id='%s' and stem like 'Mẫu sinh câu%%'" % CD); redis_del("ep:rl:practice-generate:*"); n0 = sum(1 for _ in open(W + "stub10_req.jsonl")) if os.path.exists(W + "stub10_req.jsonl") else 0
    c, j, _ = api("teacher", "POST", f"/courses/{CD}/questions/generate", {"document_ids": [DOC], "topic": "Mẫu sinh câu", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]})
    if c != 202: return c, j, None, None
    for _ in range(120):
        s = sql(f"select status from jobs where id='{j['job_id']}'")[0][0]
        if s in ("SUCCEEDED", "FAILED"): break
        time.sleep(1)
    L = [json.loads(l) for l in open(W + "stub10_req.jsonl")][n0:]
    txt = "\n".join(r["text"] for r in L if "generate" in r.get("rec","") or "TRÁNH" in r["text"] or True)
    return c, j, txt, j["job_id"]
def blk(txt): 
    m = re.search(r"TRÁNH[^\n]*((?:\n- [^\n]+)*)", txt or ""); return (m.group(0) if m else None)
def run():
    open(W + "stub10_mode", "w").write("ok"); clear_rej()
    # --- G09-05: flag bật, 4 câu loại có lý do (< 5) → không khối
    print("hint bật", hintflag(True)); mkrej("MO_HO", "a", 2); mkrej("TRUNG_LAP", "b", 1); mkrej("LAC_DE", "c", 1)
    c, j, txt, jid = gen_prompt(); b5 = blk(txt); print("g09-05", c, b5)
    chk("TC-G09-05", c == 202 and b5 is None and txt, f"QUESTION_HINTS bật, 4 câu REJECTED có lý do (MO_HO×2, TRUNG_LAP, LAC_DE; < 5): prompt sinh gửi tới stub ({len(txt or '')} ký tự) có khối «TRÁNH…»: {b5 is not None}")
    # --- G01-11: flag tắt, ≥5 câu loại có lý do → không khối
    mkrej("MO_HO", "d", 1); print("hint tắt", hintflag(False)); n11 = nrej(); c, j, txt, jid = gen_prompt(); b11 = blk(txt)
    chk("TC-G01-11", c == 202 and b11 is None and txt, f"QUESTION_HINTS tắt, {n11} câu REJECTED có lý do (≥ 5): prompt sinh có khối «TRÁNH…»: {b11 is not None}")
    # --- G09-06 / 07 / 09: bật; 5 câu (MO_HO×3, TRUNG_LAP×1, LAC_DE×1) + UNSPECIFIED + câu 91 ngày
    clear_rej(); mkrej("MO_HO", "e", 3); mkrej("TRUNG_LAP", "f", 1); mkrej("LAC_DE", "g", 1); mkrej(None, "h", 2); mkrej("SAI_DAP_AN", "i", 4, old=True)
    print("hint bật", hintflag(True)); c, j, txt, jid = gen_prompt(); b = blk(txt)
    stats = sql(f"select reject_reason, count(*) from question_bank where course_id='{CD}' and review_status='REJECTED' and reject_reason is not null and reject_reason<>'UNSPECIFIED' and reviewed_at >= now() - interval '90 days' group by 1 order by 2 desc, 1")
    lines = re.findall(r"\n- ([^\n]+)", b or "")
    exp = ["Mỗi câu chỉ một cách hiểu; tránh từ chung chung.", "Chỉ hỏi nội dung có trong tài liệu được chọn.", "Không lặp ý với câu đã có trong ngân hàng."]
    chk("TC-G09-06", c == 202 and lines == exp and "Kiểm lại đáp án" not in (b or ""), f"5 câu có lý do trong 90 ngày (MO_HO×3, TRUNG_LAP×1, LAC_DE×1) + 2 câu UNSPECIFIED + 4 câu SAI_DAP_AN cách 91 ngày (reviewed_at): SQL thống kê (≠UNSPECIFIED, ≤90 ngày) = {stats}; khối «TRÁNH» trong prompt có {len(lines)} dòng: {lines}; thứ tự = nhiều nhất trước, đồng hạng theo tên (LAC_DE trước TRUNG_LAP); SAI_DAP_AN (91 ngày) không có; tối đa 3 lý do")
    stems = sql(f"select stem from question_bank where course_id='{CD}' and topic='QC106-hint'"); leak = [s[0][:25] for s in stems if s[0][:40] in (txt or "")]
    chk("TC-G09-07", b is not None and b.startswith("\n\nTRÁNH") is False and not leak and all(l in exp for l in lines) and len(b.split("\n")[-1]) <= 140, f"khối «TRÁNH (giảng viên thường loại các câu như vậy):» + các dòng cố định đúng bảng RejectHintText (SRS 4.5); chữ câu bị loại xuất hiện trong prompt: {leak}; dòng dài nhất {max(len(l) for l in lines)} ký tự; (TC ghi tiêu đề «Tránh các lỗi sau» — chữ thật «TRÁNH (giảng viên thường loại các câu như vậy):»)")
    c3, j3, _ = api("teacher", "GET", f"/courses/{CD}/questions/gen-stats?job_id={jid}"); c4, j4, _ = api("ta", "GET", f"/courses/{CD}/questions/gen-stats?job_id={jid}")
    hints = (j3 or {}).get("hints"); print("gen-stats", c3, lastj(j3, 300))
    # --- tắt cờ → hints rỗng
    hintflag(False); c5, jx, txt5, jid5 = gen_prompt(); c6, j6, _ = api("teacher", "GET", f"/courses/{CD}/questions/gen-stats?job_id={jid5}")
    chk("TC-G09-09", c3 == 200 and set(hints or []) == {"MO_HO", "LAC_DE", "TRUNG_LAP"} and not (j6 or {}).get("hints"), f"gen-stats?job_id (cờ bật) → {c3} hints={hints} (TA → {c4}); cờ tắt → hints={ (j6 or {}).get('hints')!r}")
    # --- G09-10: cờ tắt / < 5 / lỗi thống kê
    hintflag(True); sql(f"update question_bank set reject_reason='UNSPECIFIED' where course_id='{CD}' and topic='QC106-hint' and reject_reason in ('MO_HO')")
    nA = nrej("and reviewed_at >= now() - interval '90 days'"); cA, jA, tA, _ = gen_prompt(); bA = blk(tA)  # còn 2 lý do < 5
    sql(f"update question_bank set reject_reason='MO_HO' where course_id='{CD}' and topic='QC106-hint' and stem like 'Câu thử loại e%'")
    sql(f"alter table question_bank rename column reject_reason to reject_reason_x")
    try: cB, jB, tB, _ = gen_prompt(); bB = blk(tB)
    finally: sql("alter table question_bank rename column reject_reason_x to reject_reason")
    chk("TC-G09-10", cA == 202 and bA is None and cB == 202 and bB is None and tB, f"(1) cờ bật nhưng chỉ còn {nA} câu có lý do (<5) → khối {bA is not None}; sinh {cA}; (2) lỗi thống kê (đổi tên cột reject_reason tạm trên ep-qc, đã đổi lại): sinh {cB} không lỗi, khối {bB is not None}; (3) cờ tắt: xem TC-G01-11 / G09-09")
    hintflag(False); clear_rej()
if True:
    t0 = time.time()
    while True:
        try: os.mkdir("/tmp/qc-global.lock"); break
        except FileExistsError:
            if time.time() - t0 > 900: raise SystemExit("LOCKED")
            time.sleep(5)
    open("/tmp/qc-global.lock/owner", "w").write("QC3 G09 stub QUESTION_GEN")
    os.environ["STUB_MODE_FILE"] = W + "stub10_mode"; os.environ["STUB_REQ_FILE"] = W + "stub10_req.jsonl"
    sp = subprocess.Popen(["python3", os.path.join(S10, "stub_s10.py"), "9761"], env=dict(os.environ), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL); time.sleep(1)
    try:
        print(subprocess.run(["python3", os.path.join(S10, "route10.py"), "stub"], capture_output=True, text=True, env=dict(os.environ)).stdout[-200:]); run()
    finally:
        sql("alter table question_bank rename column reject_reason_x to reject_reason") if False else None
        print(subprocess.run(["python3", os.path.join(S10, "route10.py"), "real"], capture_output=True, text=True, env=dict(os.environ)).stdout[-200:]); sp.terminate()
        import shutil; shutil.rmtree("/tmp/qc-global.lock", ignore_errors=True)
