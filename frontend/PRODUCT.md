# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- **Sinh viên** của một học phần đại cương (tải thiết kế 1.000 SV / 20 lớp). Phần lớn dùng điện thoại, giữa giờ học hoặc buổi tối. Việc chính: biết hôm nay nên làm gì, hỏi bài, hỏi riêng về điểm / chuyên cần / quy chế áp vào mình, nộp bài, luyện đề, xem kết quả.
- **Trợ giảng (TA) và giảng viên (TEACHER)**, chủ yếu trên desktop. Việc chính: xử lý hộp thư hỗ trợ (câu AI không chắc), điểm danh tại lớp (cả trên điện thoại), xác nhận câu trả lời AI trong Threads, duyệt bài AI chấm nháp, sổ điểm và công thức điểm, tài liệu, ngân hàng câu hỏi, xem lớp đang yếu ở đâu. Chỉ TEACHER xác nhận công thức, công bố và chốt điểm.
- **Admin**: mở lớp, gán giảng viên, cấu hình LLM, quan sát vận hành AI, tích hợp.

## Product Purpose

Nền tảng vận hành lớp học có AI cho một học phần: sinh viên hỏi → AI trả lời hoặc chuyển giảng viên → bài nộp được chấm nháp → giảng viên duyệt, công bố → điểm danh, điểm cộng, điểm thành phần gộp thành điểm cuối kỳ theo quy chế → giảng viên thấy lớp yếu ở đâu để cập nhật giáo trình. Thành công: sinh viên mở app biết ngay việc kế tiếp; giảng viên chỉ phải quyết những việc cần phán đoán của người.

## Positioning

- **AI chỉ nháp, người quyết**: AI trả lời, chấm nháp, trích công thức; giảng viên xác nhận, công bố, chốt.
- **Che danh tính hai chiều**: tên và MSSV được thay placeholder quanh mọi lời gọi LLM; tường lửa chặn câu hỏi riêng tư lọt ra kênh công khai (chat riêng vs Threads).
- **Điểm tính bằng code thuần từ quy chế môn học**, không LLM tính hay làm tròn điểm.

## Operating Context

Một học phần, nhiều lớp cùng mã học phần; giảng viên có thể dạy nhiều lớp (bộ chọn lớp). Tuần học, buổi học, điểm danh tại lớp, hạn nộp bài, thi giữa kỳ / cuối kỳ. Điểm chính thức vẫn là điểm trong hệ thống quản lý đào tạo của trường.

## Capabilities and Constraints

Mô-đun M0–M14 (`docs/PRD.md`): lớp học + mã tham gia, chat riêng, Threads + xác nhận, escalation + mail, luyện đề, CRM điểm danh / phát biểu / điểm cộng, hồ sơ 360, sổ điểm + công thức, bài tập + chấm + phúc khảo, tài liệu, thư viện, lịch, cấu hình LLM, quan sát, "Hôm nay". Giao diện tiếng Việt. Sinh viên không bao giờ thấy từ kỹ thuật AI (RAG, PII, provider, fallback, trace, confidence), điểm nháp, ghi chú quan sát, nhãn rủi ro của chính mình. Không gamify (không streak, xếp hạng, vòng tiến độ, confetti). Nội dung chat riêng không hiện cho giảng viên khi chưa escalate.

## Brand Commitments

Tên EduPilot; logo mark + wordmark ở `frontend/public/brand/`. Hệ thiết kế "Red Thread / Academic Instrument" do chủ dự án cung cấp (`docs/design/DESIGN.md`, token `frontend/src/shared/styles/tokens.css`): đỏ + trắng, đỏ là tín hiệu không phải nền, một họ chữ Be Vietnam Pro.

## Evidence on Hand

Chưa có người dùng thật, chưa có số đo thật. Dữ liệu demo là mô phỏng hoàn toàn (D44): môn An ninh mạng, 2 lớp × 30 sinh viên tên giả, 3 PDF môn học ở `seed/documents/`. Không được bịa số liệu đánh giá hay lời chứng thực.

## Product Principles

1. Mở app, hiểu điều gì quan trọng bây giờ, làm xong một việc rõ ràng, rồi đi.
2. Việc cần phán đoán của người luôn đến tay người; AI không tự công bố hay tự xác nhận.
3. Riêng tư theo mặc định: thông tin cá nhân không rời kênh riêng, không tới LLM ở dạng định danh.
4. Không làm mất chữ người dùng đã gõ; thao tác đảo ngược được thì không hỏi xác nhận.

## Accessibility & Inclusion

Màn của sinh viên, `/attendance`, `/inbox` phải dùng tốt ở 375 px; vùng chạm ≥ 44 px; focus luôn thấy; tôn trọng `prefers-reduced-motion`.
