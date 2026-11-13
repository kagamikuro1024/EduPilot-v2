# Dev — sửa QC đợt 5 lượt 3

| Mục | Làm gì | Kiểm |
| --- | --- | --- |
| UX20-9 / AC8 (`/me` 375 cao 1.647 px) | `Me.module.css`, < 720 px: khoảng cách giữa các khối 32 → 24 px, tiêu đề khối → khối 16 → 12 px, đệm dòng `.dense` 8 → 4 px. Không bớt chữ nào | `me.spec.ts` «me 375 … gập sẵn» (fixture có khối Điểm, 9 buổi, 2 bài tập): 1.486 → **1.410 px** (−76). `sv.kha` có ≥ số dòng của fixture nên bớt ≥ 47 px → ước ≈ 1.570. QC đo lại trên `sv.kha` |
| UX91-X1 (`story-review-v1*.webp`) | Chụp lại từ màn `/grading/<id>` hiện tại (thanh «Duyệt bài», «Đã duyệt 0/4», trích dẫn dạng trích). Không có bản chạy nào còn bài nộp chưa duyệt (demo 0 bài nộp; `ep-qc` đã tắt) nên dựng 4 bài nộp + kết quả nháp mẫu (bài «Bài tập 03 — Phân tích một vụ tấn công thực tế», 4 tiêu chí, tổng 5,25) trực tiếp vào DB dev rồi chụp bằng gateway native + frontend dựng từ nhánh này | `story-review-v1.webp` 77,7 KB (≤ 90), `-800` 35,8 KB |

Ghi chú: bài làm trong ảnh là dữ liệu mẫu do dev viết (không phải bài sinh viên thật). Ảnh mã hoá bằng `cwebp`, không `encode.sh` (máy không có ffmpeg).
