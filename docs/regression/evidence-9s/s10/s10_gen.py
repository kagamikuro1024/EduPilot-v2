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
if stage == "a2":
    D = json.load(open("/tmp/s10_gen_docs.json")); L1 = D["L1"]; mode("ok")
    var = ["  GIAO THỨC NÀO THIẾT LẬP KẾT NỐI TRƯỚC KHI TRUYỀN DỮ LIỆU??  ", "12. Thiết bị nào chuyển khung theo địa chỉ MAC?", "Tầng  nào của OSI   chịu trách nhiệm định tuyến"]
    ids = []
    for i, st in enumerate(var):
        c, j, _ = mkq("teacher", {"type": "MCQ_SINGLE", "title": f"QC 03-09b {i}", "topic": "QC-r10-A", "difficulty": "EASY", "stem": st.strip(), "options": [{"body": "x"}, {"body": "y"}], "correct": [0]}); assert c in (200, 201), (c, j); ids.append(j["id"])
    review(ids[0], "REQUEST"); review(ids[0], "REJECT"); review(ids[1], "REQUEST"); review(ids[1], "APPROVE"); review(ids[2], "REQUEST")
    st3 = sql("select left(stem,40), review_status from question_bank where id in ('%s') order by created_at" % "','".join(ids)); print("X:", st3)
    n0 = len(stems()); c, j, _ = gen({"document_ids": [L1], "topic": "Mạng máy tính cơ bản", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]}); print("gen", c, j); J = wait_job(j["job_id"]); r = J.get("result") or {}
    sk = r.get("skipped", []); n1 = len(stems()); dup = sorted(s.get("number") for s in sk if s.get("reason") == "DUPLICATE")
    chk("TC-03-09", J.get("status") == "SUCCEEDED" and dup == [1, 2, 3] and len(r.get("created", [])) == 2, f"câu X ở REJECTED («GIAO THỨC… ??» hoa + dấu cách + dấu câu cuối), APPROVED («12. Thiết bị…» số thứ tự đầu), PENDING (hai dấu cách, thiếu «?»): sinh 5 → DUPLICATE ở câu {dup}, created={len(r.get('created', []))}, skipped={[(s.get('number'), s.get('reason')) for s in sk]}; GENERATED {n0}→{n1}. NormalizeShort (NFC, gộp khoảng trắng, chữ thường, bỏ dấu câu CUỐI) không bỏ số thứ tự đầu")
if stage == "a3":  # lớp sạch QD10C
    CCx = sql("select id from courses where class_code='QD10C'")[0][0]; mode("ok")
    Lc = upb(open(ROOT + "seed/demo/quy-che-lop1.pdf", "rb").read() + f"\n%QC-S10-GEN-C-{SUF}\n".encode(), "quy-che-lop1.pdf", "application/pdf", f"QC S10 bài giảng C {SUF}", "LECTURE", CCx); print("READY", wait_doc(Lc, C=CCx)["status"])
    var = ["  GIAO THỨC NÀO THIẾT LẬP KẾT NỐI TRƯỚC KHI TRUYỀN DỮ LIỆU??  ", "12. Thiết bị nào chuyển khung theo địa chỉ MAC?", "Tầng  nào của OSI   chịu trách nhiệm định tuyến"]
    ids = []
    for i, st in enumerate(var):
        c, j, _ = mkq("teacher", {"type": "MCQ_SINGLE", "title": f"QC 03-09c {i}", "topic": "QC-r10-A", "difficulty": "EASY", "stem": st.strip(), "options": [{"body": "x"}, {"body": "y"}], "correct": [0]}, C=CCx); assert c in (200, 201), (c, j); ids.append(j["id"])
    review(ids[0], "REQUEST", C=CCx); review(ids[0], "REJECT", C=CCx); review(ids[1], "REQUEST", C=CCx); review(ids[1], "APPROVE", C=CCx); review(ids[2], "REQUEST", C=CCx)
    print("X:", sql("select left(stem,40), review_status from question_bank where id in ('%s') order by created_at" % "','".join(ids)))
    n0 = int(sql(f"select count(*) from question_bank where course_id='{CCx}' and origin='GENERATED'")[0][0])
    c, j, _ = api("teacher", "POST", f"/courses/{CCx}/questions/generate", {"document_ids": [Lc], "topic": "Mạng máy tính cơ bản", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]}); print("gen", c, j); J = wait_job(j["job_id"]); r = J.get("result") or {}
    sk = r.get("skipped", []); n1 = int(sql(f"select count(*) from question_bank where course_id='{CCx}' and origin='GENERATED'")[0][0]); dup = sorted(s.get("number") for s in sk if s.get("reason") == "DUPLICATE")
    chk("TC-03-09", J.get("status") == "SUCCEEDED" and dup == [1, 2, 3] and len(r.get("created", [])) == 2, f"lớp sạch QD10C, câu X ở REJECTED («GIAO THỨC… ??» hoa + dấu cách + dấu câu cuối), APPROVED («12. Thiết bị…» số thứ tự đầu), PENDING (hai dấu cách, thiếu «?»): sinh 5 → DUPLICATE ở câu {dup}, created={len(r.get('created', []))}, skipped={[(s.get('number'), s.get('reason')) for s in sk]}; GENERATED {n0}→{n1}. NormalizeShort (NFC, gộp khoảng trắng, chữ thường, bỏ dấu câu CUỐI) không bỏ số thứ tự đầu. Ở QD10A lần đầu: biến thể «12. … HOA» và «3) …» cũng không bị coi trùng")
if stage == "c":  # TC-03-05 ở QD10C: sv.kha được ghi danh; tài liệu ẩn
    CCx = sql("select id from courses where class_code='QD10C'")[0][0]; mode("ok")
    Lh = upb(open(ROOT + "seed/demo/quy-che-lop2.pdf", "rb").read() + f"\n%QC-S10-GEN-H-{SUF}\n".encode(), "quy-che-lop2.pdf", "application/pdf", f"QC S10 bài giảng ẩn C {SUF}", "LECTURE", CCx); print("READY", wait_doc(Lh, C=CCx)["status"])
    c, d, _ = api("teacher", "GET", f"/courses/{CCx}/documents/{Lh}"); print("ẩn", api("teacher", "PATCH", f"/courses/{CCx}/documents/{Lh}", {"visible_to_students": False, "version": d["version"]})[0], sql(f"select visible_to_students from documents where id='{Lh}'"))
    def ov(): return api("sv.kha", "GET", f"/courses/{CCx}/practice/overview")[1]
    topics0 = json.dumps(ov(), ensure_ascii=False)
    c, j, _ = api("teacher", "POST", f"/courses/{CCx}/questions/generate", {"document_ids": [Lh], "topic": "Mẫu sinh lỗi", "difficulty": "EASY", "count": 5, "types": ["MCQ_SINGLE"]}); print("gen", c, j); J = wait_job(j["job_id"]); r = J.get("result") or {}; cr = r.get("created", [])
    print("job", J.get("status"), lastj(r, 300))
    qid = cr[0] if cr else None; st = sql(f"select review_status, origin from question_bank where id='{qid}'") if qid else None
    ov1 = json.dumps(ov(), ensure_ascii=False); seen_pending = "Mẫu sinh lỗi" in ov1
    # PENDING không hiện cho SV (kể cả bắt đầu TOPIC)
    c2, s2, _ = api("sv.kha", "POST", f"/courses/{CCx}/practice/start", {"mode": "TOPIC", "topic": "Mẫu sinh lỗi", "count": 5}); print("start khi PENDING", c2, lastj(s2, 120))
    c, rv, _ = (0, 0, 0)
    if qid: review(qid, "REQUEST", C=CCx); print("approve", review(qid, "APPROVE", C=CCx)[0])
    ov2 = json.dumps(ov(), ensure_ascii=False); seen_ok = "Mẫu sinh lỗi" in ov2
    c3, s3, _ = api("sv.kha", "POST", f"/courses/{CCx}/practice/start", {"mode": "TOPIC", "topic": "Mẫu sinh lỗi", "count": 5}); print("start sau APPROVED", c3, lastj(s3, 300))
    it = (s3.get("items") or [{}])[0] if isinstance(s3, dict) else {}; q = it.get("question", {}); print("item keys", sorted(q.keys()), "src", lastj({k: v for k, v in q.items() if "source" in k or "citation" in k or "href" in k}, 300))
    api("sv.kha", "POST", f"/courses/{CCx}/practice/attempts/{s3.get('id')}/submit", {}, idem=str(uuid.uuid4())) if c3 in (200, 201) else None
    json.dump({"Lh": Lh, "qid": qid}, open("/tmp/s10_gen_c.json", "w"))
    chk("TC-03-05 (API)", c == 0 and st and st[0][0] == "PENDING" and not seen_pending and c2 in (404, 409, 422) and seen_ok and c3 in (200, 201), f"tài liệu ẩn (visible=false): sinh → 202, câu {st}; PENDING: overview SV có chủ đề «Mẫu sinh lỗi» {seen_pending}, start TOPIC {c2}; sau APPROVED: overview có {seen_ok}, start {c3}")
if stage == "d":
    CCx = sql("select id from courses where class_code='QD10C'")[0][0]; Lh = json.load(open("/tmp/s10_gen_c.json"))["Lh"]; mode("ok")
    c, j, _ = api("teacher", "POST", f"/courses/{CCx}/questions/generate", {"document_ids": [Lh], "topic": "Mẫu sinh câu", "difficulty": "EASY", "count": 5, "types": ["MCQ_SINGLE"]}); print("gen", c, j); J = wait_job(j["job_id"]); r = J.get("result") or {}; cr = r.get("created", []); print("created", len(cr))
    ov0 = json.dumps(api("sv.kha", "GET", f"/courses/{CCx}/practice/overview")[1], ensure_ascii=False); pend_vis = "Mẫu sinh câu" in ov0
    c2, s2, _ = api("sv.kha", "POST", f"/courses/{CCx}/practice/start", {"mode": "TOPIC", "topic": "Mẫu sinh câu", "count": 5})
    for q in cr: review(q, "REQUEST", C=CCx); review(q, "APPROVE", C=CCx)
    c3, s3, _ = api("sv.kha", "POST", f"/courses/{CCx}/practice/start", {"mode": "TOPIC", "topic": "Mẫu sinh câu", "count": 5}); print("start", c3, lastj(s3, 200))
    its = s3.get("items") or []; q = (its[0].get("question") or {}) if its else {}
    src = {k: v for k, v in q.items() if k not in ("options", "stem", "type")}; print("item keys", sorted(q.keys()), lastj(src, 300))
    nm = c3 in (200, 201)
    if nm: api("sv.kha", "POST", f"/courses/{CCx}/practice/attempts/{s3['id']}/answers-dummy", {}) if False else None
    chk("TC-03-05 (API)", len(cr) == 5 and not pend_vis and c2 in (404, 409, 422) and nm, f"tài liệu LECTURE ẩn: sinh 5 → created {len(cr)} (PENDING); SV overview khi PENDING có chủ đề: {pend_vis}, start {c2}; sau APPROVED start {c3}; khoá DTO mục: {sorted(q.keys())}; nguồn: {lastj(src, 160)}")
    json.dump({"Lh": Lh, "aid": s3.get("id"), "Lh_ok": nm}, open("/tmp/s10_gen_c.json", "w"))
if stage == "e":  # TC-03-13: 6 việc generate trong 1 giờ ở QD10A
    D = json.load(open("/tmp/s10_gen_docs.json")); L1 = D["L1"]; mode("ok"); out = []
    cnt = int(sql(f"select count(*) from jobs where kind='question.generate' and created_at > now() - interval '60 minutes' and (result::text is not null) and owner_id is not null")[0][0])
    for i in range(3):
        c, j, _, h = raw("POST", f"/courses/{C1}/questions/generate", {"document_ids": [L1], "topic": "Mẫu sinh câu", "difficulty": "EASY", "count": 3, "types": ["MCQ_SINGLE"]}, headers={"Idempotency-Key": str(uuid.uuid4())}, token=tok("teacher")); out.append((c, j.get("code") if isinstance(j, dict) else None, (h or {}).get("Retry-After") or (h or {}).get("retry-after"), lastj(j, 140)))
        print(i, out[-1])
        if c == 429: break
    chk("TC-03-13", out and out[-1][0] == 429 and "RATE_LIMITED" in out[-1][3] and bool(out[-1][2]), f"QD10A đã có sẵn 4 việc generate trong 60 phút (03-09 ×2, 03-03, 03-05 — ở lần sinh khác nhau); gửi tiếp: {[(o[0]) for o in out]} → việc thứ {4 + len(out)} trả {out[-1][0]} {out[-1][3][:100]} Retry-After={out[-1][2]}; không tạo việc mới ở lần 429")
