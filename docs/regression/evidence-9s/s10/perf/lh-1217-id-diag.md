# R10S-20 — /practice/[id] LCP 18,4 s (mã 0bdef404, ep-qc, sv.kha, 1 lượt devtools; file chi tiết lh-1217-id-diag.json)
Phần tử LCP: h1.ep-page-title (PreShell). FCP 859 ms. Tài liệu HTML 912 ms; chunk/css xong ~3,9 s; 3 yêu cầu _rsc xong ~4,3 s.
Waterfall tới /api/v1/courses/*/practice/attempts/{id}: từ 3,9 s tới 18,4 s có 26 yêu cầu GET LIÊN TIẾP (mỗi cái chờ cái trước, ~570 ms/cái dưới devtools throttle), 25 cái trả 404 ở 25 lớp KHÁC (sv.kha có 26 lớp), cái cuối (lớp đúng, QD10A) trả 200 lúc 18,4 s.
=> trang tìm lớp của lượt luyện bằng cách thử lần lượt từng lớp của người dùng; h1 (LCP) chỉ vẽ lại/ổn định sau khi có lớp.
Đo curl (máy rảnh, 5 lần): GET attempts/{id} đúng lớp 69/16/10/14/16 ms; lớp khác (404) 26/21/29/12/11 ms → máy chủ nhanh; thời gian mất ở chuỗi tuần tự phía trình duyệt.
Số liệu 3 lượt devtools: 18.164 / 18.139 / 18.292 ms; LCP simulate 3.154 ms; TBT 35 ms.
