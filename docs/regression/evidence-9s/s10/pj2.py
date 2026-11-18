# TC-02-12/13b trên ep-r: tài liệu riêng của lớp 2 (tải lên lớp 2) dùng từ lớp 1 → 404 (document_courses: tài liệu dùng chung đã có từ sau sprint 10)
from pb import *
C2 = course("teacher", "QD10B")
d2 = upload("seed/demo/quy-che-lop1-v2.pdf", "QC r10 đề riêng lớp 2", "EXAM_PAPER", C=C2); e2 = d2["document"]["id"]
k2 = upload("seed/demo/quy-che-lop2.pdf", "QC r10 đáp án riêng lớp 2", "ANSWER_KEY", C=C2)["document"]["id"]
print(sql(f"select id, course_id from documents where id in ('{e2}','{k2}')"), C2)
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": e2}); ok("TC-02-12 EXAM_PAPER riêng lớp 2 gọi từ lớp 1 → 404", c == 404, f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": docid("Đề mẫu luyện tập — Mạng máy tính")["id"] if False else sql("select id from documents where title='Đề mẫu luyện tập — Mạng máy tính' limit 1")[0][0], "answer_key_document_id": k2}); ok("TC-02-13b answer_key riêng lớp 2 → 404", c == 404, f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
