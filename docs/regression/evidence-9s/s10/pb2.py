from pb import *
import json
def ex(body, user="teacher", C=C1, idem=None, tabs=None): return api(user, "POST", f"/courses/{C}/questions/extract", body, idem=idem)
D = {t: docid(t) for t in ["QC Đề mẫu", "QC Đáp án mẫu", "QC Đáp án thiếu", "QC Đề lỗi câu", "QC Đề lỗi schema"]}
de, dk = D["QC Đề mẫu"]["id"], D["QC Đáp án mẫu"]["id"]
c, lst, _ = api("teacher", "GET", f"{P}/documents?limit=100"); lec = next(d for d in lst["items"] if d["type"] in ("LECTURE",) and d["status"] == "READY")
# --- âm tính (không xếp việc)
c, j, _ = ex({"document_id": lec["id"]}); ok("TC-02-11 LECTURE → 422 WRONG_DOCUMENT_TYPE", c == 422 and "WRONG_DOCUMENT_TYPE" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:120]}")
c, j, _ = ex({"document_id": de, "answer_key_document_id": lec["id"]}); ok("TC-02-13a answer_key sai loại → 422 WRONG_DOCUMENT_TYPE", c == 422 and "WRONG_DOCUMENT_TYPE" in json.dumps(j), f"{c}")
c, j, _ = ex({"document_id": str(uuid.uuid4())}); ok("TC-02-12 document_id lạ → 404", c == 404, str(c))
c2d = None
if C2:
    c, l2, _ = api("teacher", "GET", f"/courses/{C2}/documents?limit=100"); 
    d2 = next((d for d in l2["items"]), None)
    if d2:
        c, j, _ = ex({"document_id": d2["id"]}); ok("TC-02-12 tài liệu lớp 2 từ lớp 1 → 404 (không lộ)", c == 404, f"{c}")
        c, j, _ = ex({"document_id": de, "answer_key_document_id": d2["id"]}); ok("TC-02-13b answer_key lớp khác → 404", c == 404, f"{c}")
c, j, _ = ex({"document_id": de}, user="sv.kha"); ok("TC-02-P1 STUDENT → 403", c == 403, str(c))
c, j, _ = ex({"document_id": de}, user="admin"); ok("TC-02-P2 ADMIN → 403", c == 403, str(c))
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": de}, idem=False); ok("TC-02-P4 thiếu Idempotency-Key → 4xx", c in (400, 422), str(c))
if C2: c, j, _ = ex({"document_id": de}, C=C2); ok("TC-02-P5 GV lớp 1 gọi trên lớp 2 / tài liệu lớp 1", c in (403, 404), f"{c}")
# --- việc 1 (đã chạy ở bước trước, không đáp án): lấy job gần nhất bằng cách chạy lại với khoá mới sẽ tốn hạn mức → đọc DB
rows = sql(f"select id, status, left(result::text,200) from jobs where kind like '%extract%' order by created_at desc limit 3") if False else []
# --- việc 2: có bảng đáp án
k = str(uuid.uuid4())
c, j, _ = ex({"document_id": de, "answer_key_document_id": dk}, idem=k); ok("TC-02-01 extract → 202 {job_id}", c == 202 and j.get("job_id"), f"{c} {j}")
jid = j["job_id"]; c, jr, _ = ex({"document_id": de, "answer_key_document_id": dk}, idem=k)
ok("TC-02-08 cùng Idempotency-Key → cùng job_id", jr.get("job_id") == jid, f"{c} {jr}")
J = wait_job(jid); res = J.get("result") or {}
print("JOB", json.dumps(J, ensure_ascii=False)[:500])
cr, sk = res.get("created") or [], res.get("skipped") or []
ok("TC-02-01b job xong: created+skipped=24", len(cr) + len(sk) == 24, f"status={J.get('status')} created={len(cr)} skipped={len(sk)}")
ex_q = [q for q in bank() if q["origin"] == "EXTRACTED"]
print("EXTRACTED", len(ex_q), {q["review_status"] for q in ex_q})
ok("TC-02-02 ≥20 câu EXTRACTED PENDING", len(ex_q) >= 20 and all(q["review_status"] == "PENDING" for q in ex_q), f"{len(ex_q)} câu")
# đối chiếu bảng đáp án
exp = json.load(open(ROOT + "seed/practice/de-mau.json"))["questions"]
bystem = {}
miss = 0; bad = []; nocit = 0; goodn = 0
for q in ex_q:
    d = detail(q["id"]); ak = d.get("answer_key") or {}
    if not d.get("citations") or not all(isinstance(c.get("page"), int) and c.get("document_id") == de for c in d["citations"]): nocit += 1
    bystem[d["stem"]] = (d, ak)
for e in exp:
    hit = next(((d, ak) for s, (d, ak) in bystem.items() if e["stem"][:30] in s or s[:30] in e["stem"]), None)
    if not hit: miss += 1; continue
    d, ak = hit; pos = [o["position"] - 1 for o in d["options"] if o["id"] in (ak.get("option_ids") or [])]
    cor = e["correct"] if isinstance(e["correct"], list) else [e["correct"]]
    if sorted(pos) != sorted(cor): bad.append((e["stem"][:30], pos, cor))
    else: goodn += 1
ok("TC-02-05 đáp án đúng khớp bảng 24/24 theo số câu", goodn == 24 and not bad, f"khớp {goodn}/24, thiếu {miss}, lệch {bad[:3]}")
ok("TC-02-02c mọi câu có trích dẫn {document_id=đề, page nguyên}", nocit == 0 and len(ex_q) > 0, f"câu thiếu/ sai trích dẫn: {nocit}/{len(ex_q)}")
# TC-02-09 khoá mới cùng tài liệu
c, j, _ = ex({"document_id": de, "answer_key_document_id": dk}); J2 = wait_job(j["job_id"]); r2 = J2.get("result") or {}
n2 = len([q for q in bank() if q["origin"] == "EXTRACTED"])
ok("TC-02-09 trích lại khoá mới: không thêm câu; skipped DUPLICATE", c == 202 and n2 == len(ex_q) and all(s.get("reason") == "DUPLICATE" for s in r2.get("skipped", [])) and len(r2.get("skipped", [])) > 0, f"{c} EXTRACTED {len(ex_q)}→{n2}; skipped={len(r2.get('skipped',[]))} reasons={ {s.get('reason') for s in r2.get('skipped',[])} }")
# TC-02-07: lỗi câu
c, j, _ = ex({"document_id": D["QC Đề lỗi câu"]["id"]}); J3 = wait_job(j["job_id"]); r3 = J3.get("result") or {}
print("LOICAU", json.dumps(r3, ensure_ascii=False)[:600])
ok("TC-02-07a đề lỗi câu: 1 câu lưu, skipped INVALID/NO_ANSWER", c == 202 and len(r3.get("created", [])) == 1 and {s["reason"] for s in r3.get("skipped", [])} >= {"INVALID", "NO_ANSWER"}, f"created={len(r3.get('created',[]))} skipped={[(s.get('number'), s.get('reason')) for s in r3.get('skipped',[])]}")
c, j, _ = ex({"document_id": D["QC Đề lỗi schema"]["id"]}); J4 = wait_job(j["job_id"])
cnt = len(bank()); print("LOISCHEMA", json.dumps(J4, ensure_ascii=False)[:500])
ok("TC-02-07b lỗi schema: job FAILED, 0 câu lưu", J4.get("status") in ("FAILED", "DEAD") and len(J4.get("result", {}).get("created", []) if J4.get("result") else []) == 0, f"status={J4.get('status')} error={json.dumps(J4.get('error'), ensure_ascii=False)[:120]}")
# TC-02-19: việc thứ 6 trong giờ
c, j, _ = ex({"document_id": D["QC Đáp án thiếu"]["id"] and de, "answer_key_document_id": D["QC Đáp án thiếu"]["id"]})
print("6th", c, j)
c, j2, _, = ex({"document_id": de, "answer_key_document_id": dk})
ok("TC-02-19 việc thứ 6+ trong giờ → 429 RATE_LIMITED", c == 429 and "RATE_LIMITED" in json.dumps(j2), f"{c} {json.dumps(j2, ensure_ascii=False)[:100]}")
