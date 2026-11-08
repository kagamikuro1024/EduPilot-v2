# Câu hỏi của QC cho PM — TC FEAT-uiux-91 (2026-11-08)

QC không tự đoán hành vi sản phẩm: các chỗ dưới đây AC mơ hồ, thiếu hoặc không kiểm được hộp đen. Trong lúc chờ, TC ghi "theo thiết kế dev nêu ở handoff" hoặc chạy theo cách đề xuất; kết quả bị coi là **KHÔNG KIỂM ĐƯỢC** nếu không có trả lời.

| # | Story / AC | Vấn đề | QC đề xuất |
| --- | --- | --- | --- |
| Q1 | US-02 AC2 | Bảng định dạng không có hàng cho **ngày mai / tuần sau** (tương lai gần) và cho đúng 7 ngày trước (ranh giới của «trong 6 ngày trước») | Ngày mai → «Ngày mai, 09:00»; mốc 7 ngày trước đã thuộc «xa hơn, cùng năm»; BA bổ sung hai hàng vào AC2 |
| Q2 | US-03 AC2 vs US-13 AC4 | AC2 liệt kê `Phiên mới` trong các nút khoá có lý do, nhưng ghi «ẩn thay vì khoá», và US-13 AC4 nói **không** có nút khi phiên trống. Với `Tải bài lên`, AC chỉ nói «nêu điều kiện hiện có» mà không có chữ cụ thể | `Phiên mới`: bỏ khỏi danh sách nút khoá của US-03. `Tải bài lên`: BA ghi chữ lý do (ví dụ «Chọn một bài tập trước») |
| Q3 | US-03 AC5, US-04 AC11 | Hai AC đòi trạng thái không có cách kích hoạt từ giao diện thật: nút có `disabledReason` nhưng không `disabled`; thanh tiến độ nhận giá trị ngoài 0–100 | Dev trưng bày cả hai ca trong `/dev/ui` (và nêu vị trí ở handoff) để QC kiểm hộp đen; nếu không, hai AC chỉ kiểm bằng test đơn vị của dev |
| Q4 | US-15 AC5 | Cần bài có hai lượt chấm lệch > 1 điểm; seed hiện có chưa chắc có | Seed (hoặc dev) thêm một bài như vậy vào lớp 761987, ghi mã bài ở handoff |
| Q5 | US-08 AC3 | Tab ẩn: AC nói clip không tải và không phát, nhưng không nói gì khi tab trở lại hiện (phát tiếp hay không?) | Tab hiện → bắt đầu tải và phát (trừ `reduce` / `Save-Data`); BA ghi vào AC |
| Q6 | US-04 AC8 | «Mọi keyframe chỉ đổi `transform` / `opacity`» mâu thuẫn với hoạt cảnh khung xương `shimmer` (1400 ms, chạy lặp, đổi `background-position`) đã có ở `Feedback`; ngoại lệ chỉ nêu spinner `Button` và ba chấm Threads | Thêm `shimmer` (chỉ báo tải chạy liên tục) vào ngoại lệ, hoặc yêu cầu đổi sang `transform`; chủ dự án / PM quyết |
| Q7 | US-19 AC2 | «≥ 80 % thread cùng trạng thái» tính trên **trang đang hiển thị** hay toàn tab (nhiều trang cursor)? | Tính trên các thread đã tải trên màn (trang hiện có); dòng đầu nêu «{N} câu chờ xác nhận» cũng theo số đang tải |
| Q8 | US-07 AC1 | «admin: cả ba nếu có quyền» — nav admin hiện **không** có Hỏi đáp công khai / Lỗ hổng kiến thức / Số liệu lớp học | TC kiểm nhãn ở **nơi vai có mục đó** (SV: 1; GV / TA: 3; admin: 0 mục thuộc ba nhãn). Báo nếu muốn khác |
| Q9 | US-09 AC2 | «khung đầu ≈ khung cuối» (lặp liền mạch) không có ngưỡng | QC dùng SSIM ≥ 0,90 giữa khung đầu và cuối làm ngưỡng kiểm máy, kèm xem tay |
| Q10 | US-08 AC13 | «4G chậm (Lighthouse mobile, throttle thật)» chạy từ VPS tới chính nó không phản ánh mạng thật | Chấp nhận Lighthouse mô phỏng throttle trên VPS (≥ 3 lượt, trung vị) như số đo chính thức của QC; ghi rõ giới hạn trong báo cáo |
| Q11 | (quy ước tệp) | `QC.md` quy định `tc-<story>.md`; QC gộp theo đợt (`tc-wave-N.md`) để dùng chung tiền điều kiện | Chấp nhận gộp theo đợt (mỗi story một mục, có bảng phủ AC) |
| Q12 | US-22 | Chưa viết TC | Chờ TL-3 + Q-UX91-13…17 chốt và BA đổi sang APPROVED |

## Nhắc dev / PM (không phải câu hỏi)
- Stack test `-p ep-qc` cần được dựng với **commit của từng đợt**; dev ghi commit và cách dựng ở `handoff/dev-wave-<n>.md`.
- Mỗi đợt dev nêu rõ: tên mô-đun định dạng ngày giờ (US-02), vị trí `/dev/ui` của các ca đặc biệt (Q3), id bài / lớp dùng làm dữ liệu kiểm.
