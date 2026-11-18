# Smoke S10 — US-P9-03 sinh câu với stub LLM: TC-03-03/05/06/09/13 (QD10A, hạn mức generate 5 việc/lớp/giờ). CẦN route QUESTION_GEN=Q10-stub + khoá toàn cục.
from pm import *
from pb import wait_job, wait_doc
SUF = str(int(time.time()))[-6:]
def mode(m): open("/tmp/stub10_mode", "w").write(m)
def nreq(): return sum(1 for _ in open("/tmp/stub10_req.jsonl")) if os.path.exists("/tmp/stub10_req.jsonl") else 0
def upb(b, fname, mime, title, typ, C):
    sha = hashlib.sha256(b).hexdigest()
    c, pre, _ = api("teacher", "POST", f"/courses/{C}/uploads/presign", {"purpose": "document", "filename": fname, "mime_type": mime, "size_bytes": len(b), "sha256": sha}); assert c == 200, (c, pre)
    urllib.request.urlopen(urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"]), context=CTX).read()
    c, d, _ = api("teacher", "POST", f"/courses/{C}/uploads/complete", {"upload_id": pre["upload_id"], "title": title, "type": typ}); assert c in (200, 202), (c, d)
    return d["document"]["id"]
ROOT = os.path.expanduser("~/Documents/TA_Agent_qc106/")
def lec(name, title): return upb(open(ROOT + "seed/demo/" + name, "rb").read() + f"\n%QC-S10-GEN-{SUF}\n".encode(), name, "application/pdf", title, "LECTURE", C1)
def gen(body, user="teacher"): return api(user, "POST", f"/courses/{C1}/questions/generate", body)
def stems(): return [(r[0], r[1]) for r in sql(f"select stem, review_status from question_bank where course_id='{C1}' and origin='GENERATED'")]
stage = sys.argv[1]
if stage == "a":
    L1 = lec("quy-che-lop1.pdf", f"QC S10 bài giảng {SUF}"); L2 = lec("quy-che-lop2.pdf", f"QC S10 bài giảng ẩn {SUF}")
    print("READY", wait_doc(L1, C=C1)["status"], wait_doc(L2, C=C1)["status"]); mode("ok")
    c, d, _ = api("teacher", "GET", f"/courses/{C1}/documents/{L2}"); c2, d2, _ = api("teacher", "PATCH", f"/courses/{C1}/documents/{L2}", {"visible_to_students": False, "version": d["version"]}); print("ẩn", c2, sql(f"select visible_to_students from documents where id='{L2}'"))
    json.dump({"L1": L1, "L2": L2}, open("/tmp/s10_gen_docs.json", "w"))
    # TC-03-09: dựng 3 câu «X» ở 3 trạng thái (biến thể hoa/thường, dấu cách, số thứ tự đầu) rồi sinh
    base = ["Mẫu sinh câu 1: Giao thức nào thiết lập kết nối trước khi truyền dữ liệu?", "Mẫu sinh câu 2: Thiết bị nào chuyển khung theo địa chỉ MAC?", "Mẫu sinh câu 3: Tầng nào của OSI chịu trách nhiệm định tuyến?"]
    var = ["  mẫu SINH   câu 1:  giao thức nào thiết lập kết nối trước khi truyền dữ liệu?  ", "12. " + base[1].upper().replace("MẪU SINH CÂU 2", "Mẫu sinh câu 2"), "3) " + base[2]]
    ids = []
    for i, st in enumerate(var):
        c, j, _ = mkq("teacher", {"type": "MCQ_SINGLE", "title": f"QC 03-09 {i}", "topic": "QC-r10-A", "difficulty": "EASY", "stem": st.strip(), "options": [{"body": "x"}, {"body": "y"}], "correct": [0]}); assert c in (200, 201), (c, j); ids.append(j["id"])
    review(ids[0], "REQUEST"); review(ids[0], "REJECT"); review(ids[1], "REQUEST"); review(ids[1], "APPROVE")
    print("trạng thái X:", sql("select stem, review_status from question_bank where id in ('%s')" % "','".join(ids)))
    n0 = len(stems()); c, j, _ = gen({"document_ids": [L1], "topic": "Mẫu sinh câu", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]}); print("gen", c, j); J = wait_job(j["job_id"]); r = J.get("result") or {}
    sk = r.get("skipped", []); n1 = len(stems())
    chk("TC-03-09", J.get("status") == "SUCCEEDED" and len([s for s in sk if s.get("reason") == "DUPLICATE"]) == 3 and len(r.get("created", [])) == 2 and n1 - n0 == 2, f"3 câu X ở REJECTED/APPROVED/PENDING (biến thể hoa/thường/dấu cách/số thứ tự đầu): sinh 5 → created={len(r.get('created', []))}, skipped={[(s.get('number'), s.get('reason')) for s in sk]}; GENERATED {n0}→{n1}")
    json.dump({"L1": L1, "L2": L2, "j09": j["job_id"]}, open("/tmp/s10_gen_docs.json", "w"))
elif stage == "b":
    D = json.load(open("/tmp/s10_gen_docs.json")); L1, L2 = D["L1"], D["L2"]; mode("ok")
    # TC-03-03: câu không căn cứ
    n0 = len(stems()); c, j, _ = gen({"document_ids": [L1], "topic": "Mẫu sinh lỗi", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]}); J = wait_job(j["job_id"]); r = J.get("result") or {}; sk = r.get("skipped", []); n1 = len(stems())
    print("03-03", c, lastj(r, 600))
    chk("TC-03-03", J.get("status") == "SUCCEEDED" and len(r.get("created", [])) == 1 and any(s.get("reason") in ("CITATION_REQUIRED", "INVALID") and s.get("number") == 2 for s in sk) and n1 - n0 == 1, f"bản ghi lỗi (6 câu): câu 2 dẫn nguồn C99 (không có trong chunk), câu 3 loại không yêu cầu, câu 4 hai đáp án đúng, câu 5 trùng câu 1, câu 6 vượt số lượng → created={len(r.get('created', []))}, skipped={[(s.get('number'), s.get('reason')) for s in sk]}; GENERATED {n0}→{n1}")
    # TC-03-06: payload
    pay = sql("select p.masked_prompt from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.task='QUESTION_GEN' and p.masked_prompt like '%Chủ đề: Mẫu sinh lỗi%' order by p.created_at desc limit 1")
    mp = pay[0][0] if pay else ""; pii = re.findall(r"[\w.]+@[\w.]+|\b\d{8}\b|Lê Thu Hà|Phạm Quốc Bảo|Đỗ Hữu Hương", mp)
    heads = [l for l in mp.splitlines() if l.startswith(("Chủ đề", "Độ khó", "Số câu", "Loại"))]
    chk("TC-03-06", bool(mp) and "Chủ đề: Mẫu sinh lỗi" in mp and not pii, f"payload generate (ADMIN, llm_request_payloads): {len(mp)} ký tự; các trường: {heads[:6]}; có chunk nội dung: {'[C1]' in mp}; email/MSSV/tên người dùng: {pii[:3]}")
    # TC-03-05: tài liệu ẩn
    sv0 = api("sv.kha", "GET", f"{P}/practice/overview")[1]; c, j, _ = gen({"document_ids": [L2], "topic": "Mẫu sinh câu", "difficulty": "EASY", "count": 5, "types": ["MCQ_SINGLE"]}); print("03-05 gen", c, lastj(j)); J = wait_job(j["job_id"]); r = J.get("result") or {}
    print("03-05 job", J.get("status"), lastj(r, 400))
    json.dump({**D, "j05": j.get("job_id"), "created05": r.get("created", [])}, open("/tmp/s10_gen_docs.json", "w"))
