# Câu hỏi của QC cho PM — TC FEAT-uiux-91 (2026-11-08)

**Đã đóng cả 11 câu: PM chấp nhận mọi đề xuất của QC (P-91-5, `proposals.md`).** TC đã cập nhật theo đó; BA sửa `US.md` cho Q1, Q2, Q5, Q6.

| # | Story / AC | Chốt (P-91-5) | TC đã cập nhật |
| --- | --- | --- | --- |
| Q1 | US-02 AC2 | Thêm hàng «Ngày mai, 09:00»; mốc 7 ngày trước thuộc «xa hơn, cùng năm» | wave-1: I8, I9, TC-UX02-2 |
| Q2 | US-03 AC2 | Bỏ `Phiên mới` khỏi US-03; `Tải bài lên` khoá với «Chọn một bài tập trước» | wave-1: TC-UX03-10, 11 |
| Q3 | US-03 AC5, US-04 AC11 | Dev trưng bày hai ca ở `/dev/ui` | wave-1: tiêu đề, TC-UX03-14, TC-UX04-21 |
| Q4 | US-15 AC5 | Seed / dev thêm bài hai lượt lệch > 1 điểm ở lớp 761987 | wave-4: tiền điều kiện US-15 |
| Q5 | US-08 AC3 | Tab hiện lại → clip tải và phát (trừ `reduce` / `Save-Data`) | wave-6: TC-UX08-11 |
| Q6 | US-04 AC8 | `shimmer` khung xương thêm vào ngoại lệ | wave-1: TC-UX04-16 |
| Q7 | US-19 AC2 | Tính trên các thread đã tải trên màn | wave-5: TC-UX19-3 |
| Q8 | US-07 AC1 | Kiểm nhãn ở nơi vai có mục đó (admin: 0 mục) | wave-2: TC-UX07-3 |
| Q9 | US-09 AC2 | SSIM ≥ 0,90 giữa khung đầu và cuối + xem tay | wave-6: TC-UX09-5 |
| Q10 | US-08 AC13 | Lighthouse mô phỏng throttle trên VPS (≥ 3 lượt, trung vị), ghi rõ giới hạn | wave-6: TC-UX08-43 |
| Q11 | quy ước tệp | Gộp theo đợt `tc-wave-N.md` | — |
| Q12 | US-22 | Đã đóng: chủ dự án bỏ «Dùng thử» (P-91-4) | không có TC |

## Nhắc dev (không phải câu hỏi)
- Stack test `-p ep-qc` dựng với **commit của từng đợt**; dev ghi commit và cách dựng ở `handoff/dev-wave-<n>.md`.
- Mỗi đợt dev nêu: tên mô-đun định dạng ngày giờ (US-02), vị trí `/dev/ui` của các ca đặc biệt (Q3), id bài / lớp dùng làm dữ liệu kiểm.
