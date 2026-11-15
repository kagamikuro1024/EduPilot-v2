from pb import *
de = sql("select id from documents where title='Đề mẫu luyện tập — Mạng máy tính' limit 1")[0][0]; dk = sql("select id from documents where title='Đáp án đề mẫu luyện tập' limit 1")[0][0]
codes = []
for i in range(7):
    c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": de, "answer_key_document_id": dk}); codes.append(c)
    if c == 429: print("429 sau", i + 1, "việc; Retry-After", (j or {}).get("retry_after")); break
ok("TC-02-19 việc thứ 6+ trong giờ → 429 RATE_LIMITED (cửa sổ giờ vừa đặt lại do Redis khởi động lại bởi nhóm khác)", 429 in codes and codes.index(429) <= 5, f"{codes}")
