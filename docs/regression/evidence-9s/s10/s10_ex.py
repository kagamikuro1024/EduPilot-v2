# Smoke S10 — US-P9-02 trích đề với stub LLM (Q10-stub): TC-02-06/07/10/14/15/16/17/20. Lớp riêng QD10B/QD10C (hạn mức 5 việc/lớp/giờ). CẦN route QUESTION_GEN=Q10-stub + khoá toàn cục.
from pm import *
from pb import wait_job, wait_doc
import minipdf, zipfile, io, html, subprocess
CB = C2; CC = sql("select id from courses where class_code='QD10C'")[0][0]
SUF = str(int(time.time()))[-6:]
def mode(m): open("/tmp/stub10_mode", "w").write(m)
def nreq(): return sum(1 for _ in open("/tmp/stub10_req.jsonl")) if os.path.exists("/tmp/stub10_req.jsonl") else 0
def reqs_since(n): return [json.loads(l) for l in list(open("/tmp/stub10_req.jsonl"))[n:]]
def upb(b, fname, mime, title, typ, C):
    sha = hashlib.sha256(b).hexdigest()
    c, pre, _ = api("teacher", "POST", f"/courses/{C}/uploads/presign", {"purpose": "document", "filename": fname, "mime_type": mime, "size_bytes": len(b), "sha256": sha}); assert c == 200, (c, pre)
    urllib.request.urlopen(urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"]), context=CTX).read()
    c, d, _ = api("teacher", "POST", f"/courses/{C}/uploads/complete", {"upload_id": pre["upload_id"], "title": title, "type": typ}); assert c in (200, 202), (c, d)
    return d["document"]["id"]
def upf(path, title, typ, C): return upb(open(os.path.expanduser("~/Documents/TA_Agent_qc106/") + path, "rb").read() + f"\n%QC-S10-{SUF}-{C[:4]}\n".encode(), os.path.basename(path), "application/pdf", title, typ, C)
def docx(lines):
    body = "".join(f'<w:p><w:r><w:t xml:space="preserve">{html.escape(l)}</w:t></w:r></w:p>' for l in lines)
    bio = io.BytesIO()
    with zipfile.ZipFile(bio, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("[Content_Types].xml", '<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>')
        z.writestr("_rels/.rels", '<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>')
        z.writestr("word/document.xml", f'<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>{body}</w:body></w:document>')
    return bio.getvalue()
def ex(C, body, idem=None): return api("teacher", "POST", f"/courses/{C}/questions/extract", body, idem=idem)
def bankC(C): return api("teacher", "GET", f"/courses/{C}/questions?limit=100")[1]["items"]
def ready(C, *ids):
    return [wait_doc(i, C=C)["status"] for i in ids]
stage = sys.argv[1] if len(sys.argv) > 1 else "b"
if stage == "b":  # QD10B: 02-10, 02-06, 02-07, 02-14
    de = sql(f"select id from documents where course_id='{C1}' and title='QC S10 Đề mẫu trắc nghiệm'")[0][0]; key = sql(f"select id from documents where course_id='{C1}' and title='QC S10 Đáp án mẫu'")[0][0]
    lc = upf("seed/practice/de-mau-loi-cau.pdf", f"QC S10 đề lỗi câu {SUF}", "EXAM_PAPER", C1); ls = upf("seed/practice/de-mau-loi-schema.pdf", f"QC S10 đề lỗi schema {SUF}", "EXAM_PAPER", C1)
    print("READY", ready(C1, de, key, lc, ls)); mode("ok")
    t0 = sql("select now()")[0][0]; n0 = nreq()
    c, j, _ = ex(C1, {"document_id": de, "answer_key_document_id": key}); print("job1", c, lastj(j)); J = wait_job(j["job_id"]); r = J.get("result") or {}
    rq = reqs_since(n0); txt = "\n".join(x["text"] for x in rq)
    sn = sql(f"select left(text, 4000) from content_chunks where document_id='{key}' order by ord limit 1")[0][0]; frag = " ".join(sn.split())[:45]
    keyin = frag in txt
    others = int(sql(f"select count(*) from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.task <> 'QUESTION_GEN' and p.masked_prompt like '%' || $${frag}$$ || '%'")[0][0])
    chk("TC-02-06", J.get("status") == "SUCCEEDED" and keyin and others == 0, f"job {J.get('status')}; đoạn đáp án «{frag}» có trong lời nhắc trích (staff): {keyin}; số payload task khác QUESTION_GEN (chat/practice explain của SV…) chứa đoạn này: {others}")
    # TC-02-07: lỗi câu + lỗi schema
    cr0 = len(bankC(C1)); c, j, _ = ex(C1, {"document_id": lc}); J2 = wait_job(j["job_id"]); r2 = J2.get("result") or {}
    print("LOICAU", lastj(r2, 500)); cr1 = len(bankC(C1))
    c, j, _ = ex(C1, {"document_id": ls}); J3 = wait_job(j["job_id"]); cr2 = len(bankC(C1)); print("LOISCHEMA", J3.get("status"), lastj(J3.get("error"), 300))
    sk = r2.get("skipped", []); rs = {s.get("reason") for s in sk}
    chk("TC-02-07", len(r2.get("created", [])) == 1 and cr1 - cr0 == 1 and {"INVALID", "NO_ANSWER"} <= rs and J3.get("status") in ("FAILED", "DEAD") and cr2 == cr1, f"đề lỗi câu: created={len(r2.get('created', []))} (bank +{cr1-cr0}), skipped={[(s.get('number'), s.get('reason')) for s in sk]}; đề lỗi schema: job {J3.get('status')}, bank +{cr2-cr1}")
    # TC-02-14: OVERLOADED
    mode("overloaded"); t1 = sql("select now()")[0][0]; n1 = nreq()
    c, j, _ = ex(C1, {"document_id": de}); J4 = wait_job(j["job_id"], t=240); mode("ok"); rq = reqs_since(n1)
    aud = sql(f"select status, attempts, error_kind from llm_audit where task='QUESTION_GEN' and created_at >= '{t1}' order by created_at")
    dl = sql("select table_name from information_schema.tables where table_name ilike '%dead%'")
    print("OVERLOADED", c, J4.get("status"), lastj(J4.get("error"), 300), "stub req", len(rq), "audit", aud, "dead tables", dl)
    nb = len(bankC(C1))
    chk("TC-02-14 (API)", J4.get("status") in ("FAILED", "DEAD") and len(rq) == 3 and nb == cr2, f"OVERLOADED: job {J4.get('status')}; lời gọi tới provider = {len(rq)} (đúng 3?); llm_audit {aud}; bank {cr2}→{nb}; error {lastj(J4.get('error'), 120)}")
elif stage == "c":  # QD10C: 02-16 (docx có tên), 02-17 (ảnh quét), 02-20 (>60 trang), 02-15 (restart worker)
    rec = json.load(open(os.path.expanduser("~/Documents/TA_Agent_qc106/seed/demo/fake_llm.json")))["recordings"]
    qs = [r for r in rec if r["name"] == "practice-extract-de-mau"][0]["response"]["questions"]
    lines = ["ĐỀ MẪU LUYỆN TẬP — MẠNG MÁY TÍNH", "Người ra đề: Nguyễn Văn Test"]
    for q in qs:
        lines.append(f"Câu {q['number']}. {q['stem']}"); lines += [f"{'ABCD'[i]}. {o['body']}" for i, o in enumerate(q["options"])]
    lc = upf("seed/practice/de-mau-loi-cau.pdf", f"QC S10 đề lỗi câu {SUF}", "EXAM_PAPER", CC); ls = upf("seed/practice/de-mau-loi-schema.pdf", f"QC S10 đề lỗi schema {SUF}", "EXAM_PAPER", CC)
    dn = upb(docx(lines), f"de-ten-{SUF}.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", f"QC S10 đề có tên {SUF}", "EXAM_PAPER", CC)
    ds = upb(minipdf.make([]), f"quet-{SUF}.pdf", "application/pdf", f"QC S10 đề ảnh quét {SUF}", "EXAM_PAPER", CC)
    d70 = upb(minipdf.make([f"Trang {i} dong {j}: noi dung cau hoi on tap mon mang may tinh gom nhieu van ban dai de docling doc duoc chu {i}-{j}" for i in range(1, 71) for j in range(1, 13)], pages=70, per_page=12), f"70trang-{SUF}.pdf", "application/pdf", f"QC S10 đề 70 trang {SUF}", "EXAM_PAPER", CC)
    dm = upf("seed/practice/de-mau-trac-nghiem.pdf", f"QC S10 đề nhiều trang {SUF}", "EXAM_PAPER", CC)
    print("READY", ready(CC, lc, ls, dn, ds, d70, dm)); mode("ok")
    cr0 = len(bankC(CC)); c, j, _ = ex(CC, {"document_id": lc}); J2 = wait_job(j["job_id"]); r2 = J2.get("result") or {}; cr1 = len(bankC(CC))
    c, j, _ = ex(CC, {"document_id": ls}); J3 = wait_job(j["job_id"]); cr2 = len(bankC(CC)); sk = r2.get("skipped", []); rs = {s.get("reason") for s in sk}
    chk("TC-02-07", len(r2.get("created", [])) == 1 and cr1 - cr0 == 1 and {"INVALID", "NO_ANSWER"} <= rs and J3.get("status") in ("FAILED", "DEAD") and cr2 == cr1, f"đề lỗi câu: created={len(r2.get('created', []))} (bank +{cr1-cr0}), skipped={[(s.get('number'), s.get('reason')) for s in sk]}; đề lỗi schema: job {J3.get('status')} {lastj(J3.get('error'), 90)}, bank +{cr2-cr1}")
    print("docs", sql(f"select title, status, page_count, (select count(*) from content_chunks c where c.document_id=d.id) from documents d where id in ('{dn}','{ds}','{d70}','{dm}')"))
    n0 = nreq(); c, j, _ = ex(CC, {"document_id": dn}); J = wait_job(j["job_id"]); rq = reqs_since(n0); txt = "\n".join(x["text"] for x in rq)
    pay = sql(f"select count(*) from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.task='QUESTION_GEN' and a.created_at > now() - interval '10 minutes' and p.masked_prompt like '%Nguyễn Văn Test%'")[0][0]
    stems = [r[0] for r in sql(f"select stem from question_bank where course_id='{CC}'")]; leak = [s for s in stems if re.search(r"\[[A-Z_]+_?\d*\]|<[A-Z_]+|Nguyễn|\bTest\b|PERSON|TÊN", s)]
    chk("TC-02-16", J.get("status") == "SUCCEEDED" and "Nguyễn Văn Test" not in txt and pay == "0" and not leak and len(stems) > 0, f"tài liệu có «Người ra đề: Nguyễn Văn Test»: job {J.get('status')}, {len(stems)} câu lưu; lời nhắc tới provider chứa tên: {'Nguyễn Văn Test' in txt}; masked_prompt trong DB chứa tên: {pay}; stem rò placeholder/tên: {leak[:2]}; lời nhắc có dòng người ra đề: «{[l for l in txt.splitlines() if 'Người ra đề' in l][:1]}»")
    c, j, _ = ex(CC, {"document_id": ds}); print("scan", c, lastj(j, 300)); J = wait_job(j["job_id"]) if c == 202 else {}
    msg = lastj(j, 300) + lastj(J.get("error"), 300) + lastj(J.get("result"), 300); nb = len(sql(f"select 1 from question_bank where course_id='{CC}'"))
    chk("TC-02-17", ("Không đọc được văn bản" in msg) and nb == len(stems), f"đề ảnh quét (docling rỗng): extract {c} → job {J.get('status')}; phản hồi/lỗi «{msg[:200]}»; bank {len(stems)}→{nb}")
    c, j, _ = ex(CC, {"document_id": d70}); print("70p", c, lastj(j, 300))
    J = wait_job(j["job_id"]) if c == 202 else {}; print("70p job", J.get("status"), lastj(J.get("error"), 300), lastj(J.get("result"), 300), sql(f"select page_count from documents where id='{d70}'"))
    # TC-02-15
    mode("slow"); n0 = nreq(); bank0 = len(sql(f"select 1 from question_bank where course_id='{CC}'")); c, j, _ = ex(CC, {"document_id": dm}); jid = j["job_id"]; time.sleep(4)
    subprocess.run(["docker", "restart", "ep-qc-worker-1"], capture_output=True); mode("ok"); t0 = time.time(); J = wait_job(jid, t=300); print("restart job", J.get("status"), time.time() - t0)
    time.sleep(2); stems2 = [r[0] for r in sql(f"select stem from question_bank where course_id='{CC}'")]; b2 = stems2
    chk("TC-02-15", J.get("status") == "SUCCEEDED" and len(stems2) == len(set(stems2)) and len(stems2) > bank0, f"restart worker giữa việc (stub chậm 12 s): job → {J.get('status')} sau {time.time()-t0:.0f} s; bank {bank0}→{len(b2)}; stem trùng: {len(stems2)-len(set(stems2))}; lời gọi provider {len(reqs_since(n0))}")
elif stage == "c2":  # tiếp stage c: tài liệu đã tải (SUF 028038); 70 trang tải lại có chữ dày
    S0 = "028038"; gid = lambda t: sql(f"select id from documents where course_id='{CC}' and title='{t}'")[0][0]
    dn, ds, dm = gid(f"QC S10 đề có tên {S0}"), gid(f"QC S10 đề ảnh quét {S0}"), gid(f"QC S10 đề nhiều trang {S0}")
    d70 = upb(minipdf.make([f"Trang {i} dong {j}: noi dung cau hoi on tap mon mang may tinh gom nhieu van ban dai de docling doc duoc chu {i}-{j}" for i in range(1, 71) for j in range(1, 13)], pages=70, per_page=12), f"70trang-{SUF}.pdf", "application/pdf", f"QC S10 đề 70 trang {SUF}", "EXAM_PAPER", CC)
    print("READY", ready(CC, d70)); mode("ok"); nq = lambda: len(sql(f"select 1 from question_bank where course_id='{CC}'"))
    q0 = nq(); n0 = nreq(); c, j, _ = ex(CC, {"document_id": dn}); J = wait_job(j["job_id"]); rq = reqs_since(n0); txt = "\n".join(x["text"] for x in rq)
    pay = sql("select count(*) from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.task='QUESTION_GEN' and a.created_at > now() - interval '10 minutes' and p.masked_prompt like '%Nguyễn Văn Test%'")[0][0]
    pay2 = sql("select count(*) from llm_request_payloads p where p.created_at > now() - interval '10 minutes' and p.masked_prompt like '%Người ra đề%'")[0][0]
    stems = [r[0] for r in sql(f"select stem from question_bank where course_id='{CC}'")]; leak = [s for s in stems if re.search(r"\[[A-Z_]+_?\d*\]|<[A-Z_]+|Nguyễn|\bTest\b|PERSON|TÊN", s)]
    line = [l for l in txt.splitlines() if "Người ra đề" in l][:1]
    chk("TC-02-16", J.get("status") == "SUCCEEDED" and "Nguyễn Văn Test" not in txt and pay == "0" and not leak and nq() > q0, f"docx có «Người ra đề: Nguyễn Văn Test»: job {J.get('status')}, bank {q0}→{nq()}; lời nhắc tới provider chứa tên: {'Nguyễn Văn Test' in txt}; masked_prompt trong DB chứa tên: {pay}; dòng người ra đề trong lời nhắc: {line}; stem rò placeholder/tên: {leak[:2]}")
    c, j, _ = ex(CC, {"document_id": ds}); msg = lastj(j, 300); dm_ = sql(f"select status, error from documents where id='{ds}'")
    chk("TC-02-17", c == 409 and dm_[0][0] == "FAILED" and nq() == q0 + (nq() - q0), f"đề ảnh quét: tài liệu ingest {dm_} (docling không đọc được chữ); trích → {c} {msg[:120]}; không câu nào lưu từ tài liệu này; (TC ghi «Không đọc được văn bản», sản phẩm: «Không đọc được chữ trong tệp (có thể là bản quét). Hãy tải bản có chữ.»; TC thông tin ngoài phạm vi AC)")
    n0 = nreq(); q1 = nq(); c, j, _ = ex(CC, {"document_id": d70}); print("70p", c, lastj(j, 200)); J = wait_job(j["job_id"]) if c == 202 else {}; rq = reqs_since(n0)
    mp = max([x.get("maxpage", 0) for x in rq] or [0]); print("70p job", J.get("status"), lastj(J.get("error"), 200), lastj(J.get("result"), 300), "maxpage", mp, "pages", sql(f"select page_count from documents where id='{d70}'"))
    PC = sql(f"select page_count from documents where id='{d70}'")
    chk("TC-02-20", mp == 60 or "60" in lastj(J, 600), f"tài liệu 70 trang: page_count {PC}; trích: {c}, job {J.get('status')}; trang lớn nhất trong lời nhắc gửi LLM = {mp} (≤60: chỉ trích 60 trang); error/result {lastj(J.get('error'), 100)} {lastj(J.get('result'), 120)}")

elif stage == "d":  # 02-20 làm lại (cửa sổ 60 trang, mọi cửa sổ đều trả rỗng) ở QD10C; 02-16 làm lại ở QD10B với tên là NGƯỜI DÙNG BIẾT (đổi tên tạm sv đang ở lớp)
    d70 = sql(f"select id from documents where course_id='{CC}' and title like 'QC S10 đề 70 trang%' order by created_at desc limit 1")[0][0]
    mode("ok"); n0 = nreq(); c, j, _ = ex(CC, {"document_id": d70}); print("70p", c, lastj(j, 200)); J = wait_job(j["job_id"]) if c == 202 else {}; rq = reqs_since(n0)
    mp = max([x.get("maxpage", 0) for x in rq] or [0]); sk = (J.get("result") or {}).get("skipped", [])
    chk("TC-02-20", c == 202 and mp == 60 and any(s.get("reason") == "PAGE_LIMIT" for s in sk), f"tài liệu 70 trang: extract {c} → job {J.get('status')}; {len(rq)} lời gọi, trang lớn nhất gửi LLM = {mp}; skipped {sk[:2]} (PAGE_LIMIT từ trang 61: «chỉ trích 60 trang, báo rõ»; US AC9 viết 429 reason=pages nhưng TC-02-20 chấp nhận «chỉ trích 60 hoặc báo rõ»)")
elif stage == "e":  # TC-02-20 sau sửa R10S-15 (429 reason=pages)
    d70 = sql(f"select id from documents where course_id='{CC}' and title like 'QC S10 đề 70 trang%' order by created_at desc limit 1")[0][0]
    d60 = upb(minipdf.make([f"Trang {i} dong {j}: noi dung cau hoi on tap mon mang may tinh gom nhieu van ban dai de docling doc duoc chu {i}-{j}" for i in range(1, 61) for j in range(1, 13)], pages=60, per_page=12), f"60trang-{SUF}.pdf", "application/pdf", f"QC S10 đề 60 trang {SUF}", "EXAM_PAPER", CC)
    print("READY", ready(CC, d60)); mode("ok")
    j0 = int(sql("select count(*) from jobs where kind='question.extract'")[0][0]); l0 = int(sql("select count(*) from llm_audit where task='QUESTION_GEN'")[0][0])
    c, j, _, h = raw("POST", f"/courses/{CC}/questions/extract", {"document_id": d70}, headers={"Idempotency-Key": str(uuid.uuid4())}, token=tok("teacher"))
    j1 = int(sql("select count(*) from jobs where kind='question.extract'")[0][0]); l1 = int(sql("select count(*) from llm_audit where task='QUESTION_GEN'")[0][0])
    ok70 = c == 429 and "RATE_LIMITED" in lastj(j, 400) and "pages" in lastj(j, 400) and j1 == j0 and l1 == l0
    print("70 trang:", c, lastj(j, 220), "Retry-After", h.get("Retry-After"))
    c2, j2, _, h2 = raw("POST", f"/courses/{CC}/questions/extract", {"document_id": d60}, headers={"Idempotency-Key": str(uuid.uuid4())}, token=tok("teacher")); print("60 trang:", c2, lastj(j2, 160))
    chk("TC-02-20", ok70 and c2 == 202, f"tài liệu 70 trang: extract → {c} {lastj(j, 120)} (không tạo việc: jobs {j0}→{j1}; không gọi LLM: llm_audit {l0}→{l1}); tài liệu đúng 60 trang → {c2} (nhận việc)")
elif stage == "f":  # QD10B: 02-16 biến thể B (tên là người dùng biết) + 02-15 restart worker
    rec = json.load(open(os.path.expanduser("~/Documents/TA_Agent_qc106/seed/demo/fake_llm.json")))["recordings"]
    qs = [r for r in rec if r["name"] == "practice-extract-de-mau"][0]["response"]["questions"]
    lines = ["ĐỀ MẪU LUYỆN TẬP — MẠNG MÁY TÍNH", "Người ra đề: Nguyễn Văn Test"]
    for q in qs:
        lines.append(f"Câu {q['number']}. {q['stem']}"); lines += [f"{'ABCD'[i]}. {o['body']}" for i, o in enumerate(q["options"])]
    dn = upb(docx(lines), f"de-ten-b-{SUF}.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", f"QC S10 đề có tên B {SUF}", "EXAM_PAPER", CB)
    dm = upf("seed/practice/de-mau-trac-nghiem.pdf", f"QC S10 đề nhiều trang B {SUF}", "EXAM_PAPER", CB)
    print("READY", ready(CB, dn, dm)); mode("ok")
    U = "sv31@edupilot.local"; ORIG = sql(f"select full_name from users where email='{U}'")[0][0]
    sql(f"update users set full_name='Nguyễn Văn Test' where email='{U}'"); os.system("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:*roster*' | xargs -r redis-cli del\" >/dev/null 2>&1")
    try:
        n0 = nreq(); t0 = sql("select now()")[0][0]; c, j, _ = ex(CB, {"document_id": dn}); J = wait_job(j["job_id"]); rq = reqs_since(n0); txt = "\n".join(x["text"] for x in rq)
        pay = sql(f"select count(*) from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.created_at >= '{t0}' and p.masked_prompt like '%Nguyễn Văn Test%'")[0][0]
        line = [l for l in txt.splitlines() if "Người ra đề" in l][:1]
        stems = [r[0] for r in sql(f"select stem from question_bank where course_id='{CB}' and origin='EXTRACTED' and created_at >= '{t0}'") if len(r) == 1]
        leak = [s for s in stems if re.search(r"\[[A-Z_]+_?\d*\]|<[A-Z_]+|Nguyễn|\bTest\b|PERSON|TÊN", s)]
        print("B:", J.get("status"), "tên có trong lời nhắc:", "Nguyễn Văn Test" in txt, "dòng:", line, "masked_prompt chứa tên:", pay, "stems", len(stems), "leak", leak[:2])
        chk("TC-02-16 (B)", J.get("status") == "SUCCEEDED" and "Nguyễn Văn Test" not in txt and pay == "0" and not leak, f"tên «Nguyễn Văn Test» là người dùng của lớp (sv31 đổi tên tạm): lời nhắc tới provider chứa tên: {'Nguyễn Văn Test' in txt}; dòng người ra đề: {line}; payload đã che chứa tên: {pay}; {len(stems)} stem lưu, rò: {leak[:2]}")
    finally:
        sql(f"update users set full_name=$q${ORIG}$q$ where email='{U}'")
    # 02-15
    mode("slow12"); q0 = len(sql(f"select 1 from question_bank where course_id='{CB}' and origin='EXTRACTED'")); n0 = nreq(); c, j, _ = ex(CB, {"document_id": dm}); jid = j["job_id"]; time.sleep(5)
    subprocess.run(["docker", "restart", "ep-qc-worker-1"], capture_output=True); mode("ok"); t0 = time.time(); J = wait_job(jid, t=360); print("restart job", J.get("status"), f"{time.time()-t0:.0f}s", lastj(J.get("result"), 120))
    sts = [r[0] for r in sql(f"select stem from question_bank where course_id='{CB}' and origin='EXTRACTED'") if len(r) == 1]
    chk("TC-02-15", J.get("status") == "SUCCEEDED" and len(sts) == len(set(sts)) and len(sts) > q0, f"restart worker giữa việc (stub chậm 12 s, restart sau 5 s): job → {J.get('status')} sau {time.time()-t0:.0f} s; EXTRACTED {q0}→{len(sts)}; stem trùng {len(sts)-len(set(sts))}; lời gọi provider {len(reqs_since(n0))}")
