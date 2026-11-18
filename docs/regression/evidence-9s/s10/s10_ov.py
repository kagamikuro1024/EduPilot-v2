# Smoke S10: 02-06 (đáp án không vào lời nhắc/chat SV), 02-14 + 03-08 OVERLOADED thật (làm đầy 10 khe đồng thời toàn cục bằng 10 chat SV với stub chậm 45 s)
from pm import *
from pb import wait_job
import threading, subprocess
def mode(m): open("/tmp/stub10_mode", "w").write(m)
def nreq(): return sum(1 for _ in open("/tmp/stub10_req.jsonl")) if os.path.exists("/tmp/stub10_req.jsonl") else 0
stage = sys.argv[1]
if stage == "06":
    pat = r"1\. B\s+2\. A\s+3\. D\s+4\. C\s+5\. B"
    allk = sql(f"select a.task, count(*) from llm_request_payloads p join llm_audit a on a.id=p.audit_id where p.masked_prompt ~ $${pat}$$ group by 1")
    tot = sql("select count(*) from llm_request_payloads")[0][0]; sv = sql("select count(*) from llm_audit a join users u on u.id=a.user_id where u.role='STUDENT'")[0][0]
    L = [json.loads(l) for l in open("/tmp/stub10_req.jsonl")]; inlog = [x for x in L if re.search(pat, x["text"])]
    jobs = sql(f"select count(*) from jobs where kind='question.extract' and (payload_note is null or true)") if False else None
    # lệnh trích có đáp án: job đã chạy ở QD10A (stub ghi nhận lời nhắc tương ứng)
    nkey = int(sql("select count(*) from jobs where kind='question.extract' and created_at > now() - interval '3 hours' and result::text like '%DUPLICATE%'")[0][0])
    chk("TC-02-06", allk == [] and not inlog, f"bảng đáp án «1. B 2. A 3. D 4. C 5. B…» (dap-an-mau.pdf): có trong {len(allk)} nhóm task trong {tot} payload đã che ({allk}); có trong {len(inlog)}/{len(L)} lời nhắc stub (kể cả việc trích có answer_key_document_id) — đáp án được đọc bằng luật (ParseAnswerKey), KHÔNG vào lời nhắc nào, nên 0 lời gọi kênh SV chứa đáp án; llm_audit lời gọi của SV: {sv} dòng")
elif stage == "ov":
    C = C1; de = sql(f"select id from documents where course_id='{C1}' and title='QC S10 Đề mẫu trắc nghiệm'")[0][0]
    n0 = nreq(); mode("slow45"); out = []; users = ["sv.kha", "sv.gioi", "sv.nguyco", "sv.moi", "sv31", "sv32", "sv33", "sv34", "sv36", "sv37"]
    def chat(u):
        try:
            c, s, _ = api(u, "POST", "/chat/sessions", {"course_id": C1}); r = api(u, "POST", f"/chat/sessions/{s['id']}/messages", {"content": f"Lịch học tuần này của lớp mình thế nào {u}?"}, timeout=120); out.append((u, str(r[1])[:40]))
        except Exception as e: out.append((u, "ERR " + str(e)[:40]))
    ts = [threading.Thread(target=chat, args=(u,)) for u in users]; [t.start() for t in ts]; time.sleep(8)
    act = nreq() - n0; t1 = sql("select now()")[0][0]; nb0 = nreq()
    c, j, _ = api("teacher", "POST", f"/courses/{C1}/questions/extract", {"document_id": de}); print("extract", c, str(j)[:100])
    J = wait_job(j["job_id"], t=200) if c == 202 else {}
    lecs = sql(f"select id from documents where course_id='{C1}' and type='LECTURE' and status='READY' limit 1"); print("lecture", lecs)
    [t.join(timeout=100) for t in ts]; mode("ok")
    aud = sql(f"select status, attempts, error_kind, queue_wait_ms from llm_audit where task='QUESTION_GEN' and created_at >= '{t1}' order by created_at")
    print("trong lúc tải, yêu cầu tới stub:", act, "; job", J.get("status"), lastj(J.get("error"), 200), "audit", aud)
    chk("TC-02-14 (API)", J.get("status") in ("FAILED", "DEAD") and "OVERLOADED" in lastj(J.get("error"), 300), f"10 chat SV giữ 10 khe LLM (stub chậm 45 s; {act} lời gọi tới stub): trích đề → job {J.get('status')} lỗi {lastj(J.get('error'), 120)}; QUESTION_GEN audit {aud}")
