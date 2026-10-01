# FEAT-prototype-ui — Câu hỏi mở

Không câu nào chặn dev: mỗi câu có mặc định, prototype làm theo mặc định cho tới khi có trả lời.

| # | Câu hỏi | Phương án BA đề xuất | Trả lời của chủ dự án | Ngày |
| --- | --- | --- | --- | --- |
| Q1 | (= `FEAT-demo-script` Q3, chưa trả lời) Câu trả lời dưới ngưỡng độ tin cậy: hệ thống **tự** chuyển giảng viên (PRD M3, FLOWS F5) hay chờ sinh viên bấm `Nhờ giảng viên hỗ trợ` (DESIGN §14.2)? Prototype phải chọn một để diễn bước 3 của kịch bản. | **Mặc định:** tự chuyển (theo PRD, PRD thắng về hành vi — INTEGRATION mục 1); chỗ nút đổi thành "Đang chờ giảng viên · vừa gửi". Nút `Nhờ giảng viên hỗ trợ` vẫn có ở mọi câu trả lời khác. | **Tự chuyển giảng viên** (như đề xuất). | 2026-10-01 |
| Q2 | `/analytics` có mục "chi phí chỉ cho vai được phép" (DESIGN §14.25) nhưng PRD §3 không nói TA có được xem chi phí LLM không. | **Mặc định:** GV và Admin thấy; TA không thấy (PRD §3 chỉ cho TA "–" ở dòng cấu hình LLM / observability). | **Giảng viên + Admin**; TA không thấy. | 2026-10-01 |
| Q3 | (v5, #18) Threads hệ cũ có lượt xem, lượt thích và "câu trả lời được chấp nhận". Có mang sang prototype không? | **Mặc định: không mang.** Lượt thích / lượt xem là số đếm thi đua (CLAUDE.md "không gamify", DESIGN §21); trạng thái câu trả lời đã có nhãn xác nhận của giảng viên thay cho "được chấp nhận". Prototype làm theo mặc định, không chặn dev. | | |
| Q4 | (v5, #18) Câu hỏi Threads mà AI không có mẫu (D49 "chưa đủ chắc chắn"): có mở ticket ở `/inbox` như chat riêng không? | **Mặc định: không mở ticket.** Thread vẫn công khai, chuyển cho giảng viên qua "Hôm nay" với nhãn "Cần giảng viên trả lời" và trả lời ngay trong thread (cả lớp cùng xem). `/inbox` giữ cho câu hỏi riêng tư từ chat riêng (PRD M3, FLOWS F5). Không chặn dev. | | |
