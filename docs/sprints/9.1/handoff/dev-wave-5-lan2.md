# Dev — sửa QC đợt 5 lượt 2

| Mục | Làm gì | Kiểm |
| --- | --- | --- |
| UX13-19 (nháp sau tải lại) | `RealChat.tsx`: phiên đang xem được ghi vào URL (`/chat?session=…`, `history.replaceState`); tải lại mở đúng phiên, khoá nháp theo phiên nên nháp còn | `private-chat.spec.ts` «draft theo phiên» xanh |
| UX20-9 / P-91-14 | `RealMe.tsx`: `Fold` — «Bài tập» và «Thời gian học» gập sẵn ở < 720 px (nút `Mở` / `Gập` ở hàng tiêu đề), từ 720 px luôn mở | `me.spec.ts` «me 375 … gập sẵn»: fixture có khối Điểm 3 thành phần + 9 buổi + 2 bài tập → **1.486 px** (≤ 1.600). Chưa đo trên `sv.kha` của QC (2.059 px không gập): ước tính gập hai khối bớt ≈ 320 px → ≈ 1.740 px, **có thể vẫn > 1.600**; QC đo lại, nếu vượt thì cần gập thêm «Điểm cộng đã ghi» (đổi nội dung, cần PM) |
| UX91-X1 (ảnh khung ghim) | Chụp lại `story-chat-v1`, `story-gradebook-v1` (+ `-800`) từ UI 9.1 hiện tại trên dữ liệu demo `https://localhost` (chỉ đọc; frontend dựng từ nhánh này, `/api` chuyển tiếp sang demo). `record.mjs`: chọn phiên đầu của lịch sử (demo không còn phiên «Điều kiện để được dự thi»), bỏ qua ảnh duyệt khi hàng chờ trống | ảnh 34,5 / 43,2 KB (≤ 90) và 15,2 / 19,9 KB (≤ 20) |

**Chưa làm được — `story-review-v1*.webp` giữ ảnh cũ:** demo `https://localhost` hiện **0 bài nộp** ở cả hai lớp (`GET …/submissions?status=DRAFT|REVIEWED|PUBLISHED` đều rỗng), không có màn duyệt nào để chụp, và chụp chỉ đọc không tạo được dữ liệu. Cần một bản chạy có bài nộp chưa duyệt (seed đầy đủ bước chấm bài) rồi `STILLS_ONLY=1 bash scripts/landing-video/run.sh`. Ảnh mã hoá bằng `cwebp` (không có ffmpeg trên máy) nên dung lượng khác bản `encode.sh` một chút.
