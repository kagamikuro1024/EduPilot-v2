# Ghi chú cho quyển đồ án: sprint 7 (P4 Escalation + Mail, P5 CRM + 360)

PM bổ sung dần trong sprint; bản đầy đủ viết lúc kết sprint.

## Tin nhắn khủng hoảng: giới hạn đã biết (US-P4-07 AC7)
- Bộ nhận diện chỉ dựa trên luật từ khoá hai tầng: cụm mơ hồ khi bỏ dấu chỉ khớp trên chữ có dấu; bản không dấu chỉ khớp khi có ngôi thứ nhất đứng trước. Không dùng mô hình phân loại (D46, D47).
- **Bỏ sót** cách nói rất gián tiếp, ví dụ "em mệt quá rồi, muốn ngủ mãi".
- **Bắt nhầm có chủ đích** lời nói quá, ví dụ "deadline này làm em muốn chết": vẫn trả câu hỗ trợ, vì ưu tiên không bỏ sót hơn trả lời đúng mọi lần (QUESTIONS Q10, chọn an toàn hơn đủ).
- Đây là công cụ hỗ trợ, **không thay chuyên gia** tâm lý; hệ thống không tự báo cho ai, chỉ đưa thông tin hỗ trợ và gợi ý nói chuyện với giảng viên.
- Đo bằng bộ 20 câu dương + 17 câu âm tự soạn (D44), gồm các ca bắt nhầm của sprint 6 ("từ từ", "từ hai", "kết thúc tất cả các tiến trình con").

## Quyết định kỹ thuật và số đo
- **Thư "đúng một lần" là không thể ở phía gửi SMTP** (TLR-1): worker chết sau khi máy chủ thư nhận dữ liệu mà trước khi ghi `SENT` thì thư sẽ gửi lại. Giải pháp: `Message-ID` tất định theo dòng `mail_outbox`, khử trùng ở phía nhận; ngữ nghĩa là "ít nhất một lần" + khử trùng. Đo: thư trả lời tới Mailpit sau 2 s, đúng 1 thư kể cả khi giết worker giữa chừng.
- **Thông báo tức thì qua nhiều bản gateway:** dùng lại một luồng SSE `/events` (Redis Stream đệm + pub/sub) thay vì mở luồng thứ hai (hạn 2 kết nối / người). Sự kiện sinh ở bản B tới người nối ở bản A sau 14 ms; chuông giảng viên nổi ≤ 5 s kể từ lúc sinh viên gửi câu hỏi.
- **SLA tính theo ngày làm việc** (gói `workhours`, T2–T6, giờ Việt Nam), đồng hồ đếm từ `waiting_since` và đặt lại khi sinh viên hỏi tiếp (TLR-3: đếm từ `created_at` làm ticket mở lại nhảy thẳng lên mức báo động).
- **Hồ sơ 360 và dữ liệu nhạy cảm:** mỗi lần mở hồ sơ ghi `audit_log` trong cùng giao dịch đọc (lỗi thì trả 503, không đọc), `audit_log` có trigger chặn sửa / xoá; sinh viên không có đường đọc ghi chú, nhãn rủi ro; nhận xét tổng hợp AI chỉ gửi số liệu, không tên / MSSV.
- **Thời gian học:** heartbeat chỉ `INCRBY` Redis (k6 10 req/s p95 8,82 ms), worker cuộn lên DB bằng ghi tuyệt đối `GREATEST` (idempotent); idle 2 phút ngừng đếm, tab ẩn không đếm; `internal/grade` bị cấm phụ thuộc (test kiến trúc kiểm cả phụ thuộc bắc cầu). Giới hạn: số đo chỉ là thời gian mở trang có thao tác, không đo được sự chú ý thật; mất Redis thì số của ngày thấp hơn thực tế.
- **At-risk** vật chất hoá ban đêm (`student_risk`) để danh sách 1.000 sinh viên trả trang đầu p95 ≈ 49 ms; một tuần "yếu" khi ≥ 2 tín hiệu thấp và có tín hiệu neo (chuyên cần), tránh gắn nhãn sinh viên đi học đủ nhưng ít dùng ứng dụng.
- **Điểm danh:** 30 sinh viên ở 375 px theo kịch bản 4,4 s, bàn phím ≈ 3 s; một `PUT` một giao dịch cho cả lưới, khoá lạc quan trả 409 rõ ràng.

## Khó khăn và cách giải quyết
- **CI chập chờn** là nút thắt lớn nhất của sprint: test chờ đồng hồ thật, chuỗi canary hex trùng UUID ngẫu nhiên, thứ tự `map`, rò database test (≈ 25 GB mỗi lượt, đầy đĩa hai lần). Sửa ở gốc từng loại thay vì thêm retry; sau đó 6/6 lượt nguội xanh, mỗi lượt dưới 9 phút.
- **Lỗi chỉ lộ trên hệ thống thật:** nhận xét AI luôn 503 vì sai làn ưu tiên, kho tri thức không bao giờ nạp vì thiếu danh tính lớp, seed sinh 126 ticket tự mở. Cả ba có test đơn vị xanh nhưng sai ở tích hợp; bài học: test phải đi qua cổng LLM thật với provider giả.
