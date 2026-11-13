# Prompt cho `dev`: trang giới thiệu (landing) cho bản demo (OPS-VPS, chủ dự án yêu cầu 2026-11-01)

**Vấn đề:** mở `https://v2202609419695521098.happysrv.de` chưa đăng nhập thì chỉ thấy form đăng nhập, trông trống. Người xem demo (thầy hướng dẫn, hội đồng) không biết sản phẩm làm gì.

## Hành vi
- Chưa đăng nhập mà vào `/` → hiện trang giới thiệu (một trang, server render, không cần phiên). Đã đăng nhập vào `/` → "Hôm nay" như cũ. Link sâu khác (`/gradebook`…) khi chưa đăng nhập vẫn về `/login?next=` như hiện tại.
- Nút chính `Đăng nhập`, phụ `Tạo tài khoản` (`/register`). Trang `/login` có đường quay về trang giới thiệu (logo).

## Nội dung (tiếng Việt, ngắn, đúng sự thật; không câu mô tả thừa, luật chữ UI ở `~/.claude/CLAUDE.md`)
1. **Mở đầu:** EduPilot là gì trong một câu ("vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân"), nút Đăng nhập. Một ảnh màn hình thật (vd. "Hôm nay" của giảng viên hoặc màn duyệt bài hai cột).
2. **Ba điểm khác biệt có số đo thật** (lấy từ `docs/sprints/*/report.md`):
   - che tên / MSSV trước mọi lời gọi AI;
   - AI chỉ soạn nháp, giảng viên quyết, điểm do mã tính (30/30 khớp đáp án tính tay);
   - chấm hàng loạt không làm chậm chat.
3. **Cho giảng viên / cho sinh viên:** 4–6 tính năng mỗi bên, mỗi tính năng một dòng, kèm 2–3 ảnh màn hình thật (desktop + một ảnh 375 px).
4. **Chân trang:**
   - "Đồ án tốt nghiệp · Lê Văn Quang Trung";
   - link mã nguồn GitHub;
   - dòng nhỏ "Dữ liệu trên bản demo là dữ liệu mô phỏng".

   Không nêu tài khoản / mật khẩu demo trên trang công khai.

## Thiết kế
- Theo `docs/design/DESIGN.md` (Red Thread / Academic Instrument): token `--ep-*`, Be Vietnam Pro, đỏ là tín hiệu, không tường thẻ KPI, không card lồng card, không gradient / glass / emoji. Đây là trang "Persuade" nên được phép rộng và có chủ ý hơn màn ứng dụng, nhưng cùng một hệ chữ, màu.
- Nếu có skill `impeccable` thì đọc `skill://impeccable` (chế độ Persuade) trước khi làm.
- Ảnh: chụp từ stack VPS đã seed (1440 / 375), nén WebP, đặt `frontend/public/landing/`, `next/image` có kích thước. Không nhãn "Bản mô phỏng" trong ảnh.
- **Hiệu năng:** trang tĩnh, không gọi API; LCP ≤ 2,5 s đo trên VPS (Lighthouse), axe 0 lỗi, 375 px không cuộn ngang. `bash scripts/ui-antipatterns.sh` sạch (thêm ngoại lệ có lý do nếu luật áp cho màn ứng dụng không hợp với landing; ghi proposal).

## Kiểm
Test Playwright: chưa đăng nhập `/` → thấy tiêu đề + nút Đăng nhập; đã đăng nhập `/` → "Hôm nay"; `/gradebook` chưa đăng nhập → `/login?next=`. Chạy trên VPS theo luật mới. Deploy lên `/opt/edupilot`, gửi PM ảnh 1440 + 375. Handoff `docs/sprints/ops-vps/handoff/dev-landing.md`.
