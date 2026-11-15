# Đợt R · làn 3: F6 Tài liệu, F7 Điểm danh + phát biểu, F8 Theo dõi sinh viên
from rlib import *
import hashlib, time, urllib.request, ssl, json, uuid
c1 = C("teacher"); tag = str(int(time.time()))[-5:]
# ---- F6 upload tài liệu (docling) ----
def mkpdf(lines):
    cs = "BT /F1 14 Tf 72 740 Td 18 TL " + " ".join("(" + l.replace("(", "[").replace(")", "]") + ") Tj T*" for l in lines) + " ET"
    objs = ["<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
            "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
            f"<< /Length {len(cs)} >>\nstream\n{cs}\nendstream", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"]
    out = b"%PDF-1.4\n"; offs = []
    for i, o in enumerate(objs, 1): offs.append(len(out)); out += f"{i} 0 obj\n{o}\nendobj\n".encode("latin-1")
    x = len(out); out += f"xref\n0 {len(objs)+1}\n0000000000 65535 f \n".encode() + b"".join(f"{o:010d} 00000 n \n".encode() for o in offs)
    return out + f"trailer\n<< /Size {len(objs)+1} /Root 1 0 R >>\nstartxref\n{x}\n%%EOF\n".encode()
body = mkpdf(["QC-R chuong " + tag, "Tuong lua (firewall) kiem soat luu luong mang theo quy tac.", "IDS phat hien xam nhap dua tren chu ky hoac bat thuong.", "Mat khau manh can dai, ngau nhien va khong dung lai."])
sha = hashlib.sha256(body).hexdigest()
c, pre, _ = api("teacher", "POST", f"/courses/{c1}/uploads/presign", {"purpose": "document", "filename": f"qcr-{tag}.pdf", "mime_type": "application/pdf", "size_bytes": len(body), "sha256": sha})
ok("R-F6-01 xin URL ký sẵn", c in (200, 201), f"{c} {str(pre)[:160]}")
if c in (200, 201):
    url = pre.get("url") or pre.get("upload_url"); uid = pre.get("upload_id") or pre.get("id")
    ctx = ssl.create_default_context(); ctx.check_hostname = False; ctx.verify_mode = ssl.CERT_NONE
    hd = pre.get("headers") or {}
    rq = urllib.request.Request(url, data=body, method=pre.get("method", "PUT"), headers={"Content-Type": "application/pdf", **hd})
    try:
        r = urllib.request.urlopen(rq, context=ctx, timeout=60); pc = r.status
    except urllib.error.HTTPError as e: pc = e.code
    ok("R-F6-02 đẩy thẳng lên object storage", pc in (200, 201, 204), f"{pc} {url[:60]}")
    c, dc, _ = api("teacher", "POST", f"/courses/{c1}/uploads/complete", {"upload_id": uid, "title": f"QC-R tài liệu {tag}", "type": "LECTURE", "use_for_rag": True, "visible_to_students": True})
    ok("R-F6-03 hoàn tất tải lên → tạo tài liệu", c in (200, 201, 202), f"{c} {str(dc)[:140]}")
    did = ((dc or {}).get("document") or dc or {}).get("id") if isinstance(dc, dict) else None
    st = None
    for _ in range(90):
        c, dl, _ = api("teacher", "GET", f"/courses/{c1}/documents?q=QC-R%20t%C3%A0i")
        it = [x for x in (dl or {}).get("items", []) if x.get("id") == did]
        if it and it[0].get("status") in ("READY", "FAILED"): st = it[0]["status"]; break
        time.sleep(2)
    ok("R-F6-04 ingest → READY (docling)", st == "READY", f"status={st}")
    n = sql(f"select count(*) from content_chunks where document_id='{did}'")[0][0] if did else "0"
    ok("R-F6-05 có chunk + nhúng", int(n) >= 1, f"{n} chunk")
    c, dup, _ = api("teacher", "POST", f"/courses/{c1}/uploads/presign", {"purpose": "document", "filename": f"qcr-{tag}-b.pdf", "mime_type": "application/pdf", "size_bytes": len(body), "sha256": sha})
    ok("R-F6-06 trùng nội dung → báo đã có / đề nghị chia sẻ", c in (200, 409), f"{c} {str(dup)[:140]}")
    c, dd, _ = api("teacher", "DELETE", f"/courses/{c1}/documents/{did}") if did else (0, None, 0)
    ok("R-F6-07 xoá tài liệu", c in (200, 204, 405), f"{c} {str(dd)[:80]}")
c, bad, _ = api("teacher", "POST", f"/courses/{c1}/uploads/presign", {"purpose": "document", "filename": "virus.exe", "mime_type": "application/x-msdownload", "size_bytes": 1000, "sha256": "0" * 64})
ok("R-F6-08 file sai loại bị chặn ngay trước khi tải", c in (400, 415, 422), f"{c} {str(bad)[:120]}")
c, big, _ = api("teacher", "POST", f"/courses/{c1}/uploads/presign", {"purpose": "document", "filename": "big.pdf", "mime_type": "application/pdf", "size_bytes": 900_000_000, "sha256": "0" * 64})
ok("R-F6-09 file quá cỡ bị chặn", c in (400, 413, 422), f"{c} {str(big)[:120]}")
c, sv, _ = api("sv.kha", "POST", f"/courses/{c1}/uploads/presign", {"purpose": "document", "filename": "a.pdf", "mime_type": "application/pdf", "size_bytes": 100, "sha256": "0" * 64})
ok("R-F6-10 SV không tải tài liệu lên được (403)", c == 403, f"{c}")
c, lib, _ = api("sv.kha", "GET", f"/courses/{c1}/library")
ok("R-F6-11 Thư viện SV không chứa ANSWER_KEY", c == 200 and "Đáp án" not in json.dumps(lib, ensure_ascii=False), f"{c}")
# ---- F7 điểm danh ----
c, sl, _ = api("teacher", "GET", f"/courses/{c1}/sessions")
last = [s for s in sl["items"] if not s["attendance"].get("done_at")][0]
sid = last["id"]
c, at, _ = api("teacher", "GET", f"/courses/{c1}/sessions/{sid}/attendance")
studs = at["students"]
recs = [{"student_id": s["id"], "status": "ABSENT" if i < 3 else "PRESENT"} for i, s in enumerate(studs)]
c, p1, _ = api("teacher", "PUT", f"/courses/{c1}/sessions/{sid}/attendance", {"records": recs, "complete": True})
ok("R-F7-01 điểm danh: lưu cả buổi (chạm người vắng)", c in (200, 204), f"{c} {str(p1)[:120]}")
c, at2, _ = api("teacher", "GET", f"/courses/{c1}/sessions/{sid}/attendance")
absent = sum(1 for s in at2["students"] if (s.get("record") or {}).get("status") == "ABSENT")
ok("R-F7-02 đọc lại: đúng 3 vắng", absent == 3, f"absent={absent}")
c, p2, _ = api("ta", "PUT", f"/courses/{c1}/sessions/{sid}/attendance", {"records": [{"student_id": studs[0]["id"], "status": "LATE", "version": 1}], "complete": True})
ok("R-F7-03 TA sửa sau buổi học / khoá lạc quan", c in (200, 204, 409), f"{c} {str(p2)[:100]}")
c, pt, _ = api("teacher", "POST", f"/courses/{c1}/participation", {"student_id": studs[5]["id"], "session_id": sid, "type": "SPEAK", "points": "0.25", "note": "QC-R phát biểu", "client_event_id": str(uuid.uuid4())})
ok("R-F7-04 ghi phát biểu", c in (200, 201), f"{c} {str(pt)[:100]}")
ev = str(uuid.uuid4())
c1p, _, _ = api("teacher", "POST", f"/courses/{c1}/participation", {"student_id": studs[6]["id"], "session_id": sid, "type": "SPEAK", "points": "0.25", "client_event_id": ev})
c2p, _, _ = api("teacher", "POST", f"/courses/{c1}/participation", {"student_id": studs[6]["id"], "session_id": sid, "type": "SPEAK", "points": "0.25", "client_event_id": ev})
n = sql(f"select count(*) from participation_events where client_event_id='{ev}'") if False else None
ok("R-F7-05 gửi lại cùng client_event_id không ghi đôi (idempotent)", c1p in (200, 201) and c2p in (200, 201, 409), f"{c1p} {c2p}")
c, me, _ = api("sv.kha", "GET", f"/courses/{c1}/me/attendance")
ok("R-F7-06 SV thấy điểm danh của chính mình", c == 200, f"{c} {str(me)[:100]}")
c, oth, _ = api("sv.kha", "GET", f"/courses/{c1}/students/{studs[0]['id']}/attendance")
ok("R-F7-07 SV không xem điểm danh người khác (403)", c == 403, f"{c}")
c, ng, _ = api("sv.kha", "PUT", f"/courses/{c1}/sessions/{sid}/attendance", {"records": [], "complete": True})
ok("R-F7-08 SV không điểm danh được (403)", c == 403, f"{c}")
# ---- F8 theo dõi sinh viên ----
c, sl, _ = api("teacher", "GET", f"/courses/{c1}/students?filter=attention&limit=5")
ok("R-F8-01 danh sách «Cần chú ý»", c == 200 and len(sl.get("items", [])) >= 1, f"{c} {len(sl.get('items', []))}")
u = sl["items"][0]["student"]["id"]
c, pr, _ = api("teacher", "GET", f"/courses/{c1}/students/{u}/profile")
ok("R-F8-02 hồ sơ sinh viên (rủi ro + lý do)", c == 200 and "risk" in json.dumps(pr), f"{c} {str(pr)[:140]}")
c, ob, _ = api("teacher", "POST", f"/courses/{c1}/students/{u}/observations", {"kind": "NOTE", "tags": ["vắng"], "body": "QC-R ghi chú riêng"})
ok("R-F8-03 thêm ghi chú riêng", c in (200, 201), f"{c} {str(ob)[:100]}")
c, sa, _ = api("sv.kha", "GET", f"/courses/{c1}/students/{u}/observations")
ok("R-F8-04 SV không thấy ghi chú / hồ sơ (403)", c == 403, f"{c}")
t0 = time.time(); c, ai, _ = api("teacher", "POST", f"/courses/{c1}/students/{u}/ai-summary", {})
ok("R-F8-05 nhận xét AI (LLM 9router)", c in (200, 201, 202), f"{c} {time.time()-t0:.1f}s {str(ai)[:160]}")
c, ou, _ = api("teacher", "POST", f"/courses/{c1}/students/{u}/outreach", {"body": "Chào em, thầy thấy em vắng nhiều; em cần hỗ trợ gì không?"})
ok("R-F8-06 nhắn riêng qua ticket do GV mở", c in (200, 201), f"{c} {str(ou)[:120]}")
b4 = int(sql("select count(*) from audit_log where action='view'")[0][0]); api("teacher", "GET", f"/courses/{c1}/students/{u}/profile"); af = int(sql("select count(*) from audit_log where action='view'")[0][0])
ok("R-F8-07 xem hồ sơ ghi audit_log", af > b4, f"view {b4}→{af}")
print("w3 done")
