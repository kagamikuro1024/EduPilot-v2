# Prompt cho `qc`: thu bằng chứng UI/UX (sprint 9.1, pha 1)

Đọc `docs/sprints/9.1/plan.md`.
- **Bản demo:** `https://v2202609419695521098.happysrv.de`. **Tài khoản** `teacher@`, `ta@`, `admin@`, `sv.gioi@`, `sv.kha@`, `sv.moi@` (đuôi `@edupilot.local`), mật khẩu `Edupilot#Seed-2026`.
- **Dùng** `playwright-cli` trên Mac, chỉ trình duyệt (luật `CONTEXT.md` mục 4).
- **Không làm hỏng dữ liệu demo:** không chốt điểm, không công bố, không xoá. Ảnh của các trạng thái phá dữ liệu chụp trên stack test của bạn (`-p ep-qc` trên VPS qua `ssh -L`) hoặc bỏ qua và ghi chú.

## Việc
1. **Chụp mọi route của `frontend/src/app`** (danh sách ở plan) với **từng vai được phép**: 1440 × 900, full page.
   - Ở 375 × 812: mọi màn sinh viên, `/attendance`, `/inbox`, trang giới thiệu, đăng nhập.
   - Tên tệp: `docs/sprints/9.1/audit/shots/<route>__<vai>__<cỡ>.png`.
2. **Trạng thái:** tải (chậm mạng), rỗng (lớp 2 / `sv.moi`), lỗi (offline), dữ liệu đầy, 1–2 hộp thoại / drawer chính mỗi màn, focus bàn phím (Tab) ở 3 màn chính.
3. **Mỗi màn ghi vào `docs/sprints/9.1/audit/evidence.md`:**
   - route, vai;
   - lỗi console;
   - axe (số lỗi theo mức);
   - thời gian thấy nội dung (ước lượng);
   - cuộn ngang ở 375;
   - mọi thứ trông lạ, khó hiểu hay chậm, một dòng mỗi điều. Chỉ ghi quan sát, không đề xuất; đề xuất là việc của BA.
4. **Quay 2 video ngắn** (`playwright-cli` có ghi hình): một luồng giảng viên chấm bài, một luồng sinh viên hỏi chat. Đặt ở `audit/video/`; dùng để đánh giá chuyển động.

Commit `docs/sprints/9.1/audit/**` (ảnh PNG nén, tổng < 60 MB), push `sprint/9.1-uiux`, báo PM ≤ 4 dòng. Không mở subagent.
