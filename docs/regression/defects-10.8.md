# Lỗi hồi quy sprint 10.8 (tạm, cập nhật dần) — be0c0d96, ep-r, qc2/SmkH

Chưa có lỗi sản phẩm nào được xác nhận (trượt 2 lần). Các lượt F đã gặp và đã quy về môi trường / dữ liệu lặp, chạy lại đều ra P (không phải lỗi):

| TC | Triệu chứng lần đầu | Nguyên nhân | Chạy lại |
| --- | --- | --- | --- |
| TC-AS03-07 | (c) trả `422 CHECKIN_TOKEN_INVALID` thay vì `409 SELF_CHECKIN_DISABLED` | lớp thử còn một cửa sổ đang mở; theo D-AS-06 (US-AS-01 AC5) cửa sổ đang mở vẫn nhận lượt dù tắt chính sách — hành vi đúng | P trên lớp không có cửa sổ mở |
| TC-AS03-18 | `me/attendance` rỗng | chưa đóng cửa sổ nên chưa có `attendance_records` | P sau khi đóng |
| TC-AS04-10 | K2 đầu tiên bị `DEVICE_NEW` ở lượt chạy 2 | dùng lại SV đã có khoá từ lượt chạy 1 | P với SV mới |
| TC-AS10-08b | grep log khớp `10.77` | khớp nhầm mili-giây của dấu thời gian | P với mẫu chính xác (0 dòng) |
| TC-AS12-15, TC-AS12-12…14 | 422 / dòng DB rỗng | token xoay 5 s hết hạn vì stack chậm, hoặc PostgreSQL đang phục hồi sau sự cố (nhóm khác) | P khi chạy lại |
| TC-AS06-06/08 | `/me/today` không có mục | cache Hôm nay 60 s + fixture SQL không xoá cache; mục tính trong hạn 60 ms của provider | P sau chờ hết TTL |
