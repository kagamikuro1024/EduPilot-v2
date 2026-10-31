# Ghi chú cho quyển đồ án: sprint 8 (P6 Sổ điểm và điểm cuối kỳ từ quy chế)

## Tách ba vai: AI đề xuất, người xác nhận, code tính
- **AI chỉ trích bản nháp.** AI đọc quy chế môn học (PDF) qua `llm.Structured` với schema strict. Mỗi mục có trích dẫn kèm số trang. Bộ kiểm bằng code thêm `unclear_items`, ví dụ quy chế lớp 2 thiếu quy tắc làm tròn. Mục "chưa rõ" chặn `Xác nhận công thức`.
- **Chỉ giảng viên xác nhận.** Trước khi xác nhận, giảng viên thấy hậu quả bằng số: bao nhiêu sinh viên đổi điểm, xem thử 3 sinh viên.
- **Code tính điểm:** `internal/grade` dùng `shopspring/decimal`, cấm `float64` (lệnh cổng kiểm). Không import gói LLM hay dữ liệu thời gian học (test kiến trúc kiểm). Mọi quy tắc đọc từ `grade_rules.params`.
- **Công thức tổng quát:** `QT = clamp(QT₀ + B − P, 0, 10)`; `HP_thô = w_QT·QT + w_CK·CK`. Chuẩn hoá độ chính xác máy (`Round(10)`) rồi làm tròn đến bước `step`, nửa lên, một lần ở cuối. Điểm chữ tính trên số đã làm tròn. Điều kiện liệt cho điểm chữ F nhưng giữ số đã tính.

## Đúng tuyệt đối và kiểm chứng độc lập
- **Đáp án 30 sinh viên** (`seed/expected_final_grades.csv`) tính tay từ quy chế mô phỏng, có cố ý các ca biên x,25 / x,5 / x,75, chạm trần điểm cộng, liệt.
- **Bốn lần tính độc lập cùng ra một kết quả:** đáp án tính tay; một chương trình kiểm tra độc lập dùng số hữu tỉ (`fractions.Fraction`); mã Go; người kiểm thử tính lại A, B, C. Kết quả 30/30 khớp.
- **Lỗi biên tìm được trước khi viết mã:**
  - Phép chia không hết (ví dụ 10/30) làm 7,25 thành 7,2499…, rồi làm tròn xuống. Sửa bằng chuẩn hoá độ chính xác trước bước làm tròn.
  - Bản in đáp án làm tròn kiểu "chẵn gần nhất" khiến 4 dòng sai.
- **Thiếu điểm** thì là `INCOMPLETE`, không tự cho 0, trừ khi quy chế ghi rõ. Số 0 chỉ áp khi nguồn điểm đã chốt, ví dụ bài thi đã công bố. Chốt điểm bị chặn khi còn sinh viên thiếu điểm.

## Bất biến và truy vết
- **Chốt điểm tạo snapshot bất biến.** Trigger chặn `UPDATE`, `DELETE`, `TRUNCATE`.
- **Mở khoá bắt buộc nhập lý do** và tạo phiên bản mới; sinh viên bị đổi điểm được báo; mọi thao tác ghi audit.
- **Sinh viên** chỉ thấy điểm học phần sau khi chốt, không thấy điểm nháp (cả qua API lẫn trợ lý AI). Tính thử ("nếu cuối kỳ được 8,0") không lưu.
- **Danh tính khi nhập XLSX:** chỉ khớp MSSV của danh sách lớp do giảng viên nạp. MSSV sinh viên tự khai khi vào lớp bằng mã không được tin.

## Số đo
- `impact` (xem hậu quả trước khi xác nhận) p95 7–8 ms, yêu cầu ≤ 300 ms.
- `/gradebook` ảo hoá 1.000 dòng.
- Seed chạy lại không đổi (idempotent).

## Giới hạn
- **Quy chế lớp 1 là bản mô phỏng** (D44), vì tài liệu quy chế thật có trong tay chỉ nói về học vụ.
- **Chưa đo độ chính xác trích công thức của LLM thật.** Kiểm thử dùng nhà cung cấp giả có bản ghi: chứng minh đường ống, bộ kiểm và trích dẫn, không chứng minh độ chính xác của mô hình.
- **Điểm chính thức** nằm ở hệ thống quản lý đào tạo của trường; màn hình và file xuất đều ghi rõ điều này.
