# Sprint 9.1: góp ý của đội

| # | Ai | Vấn đề | Đề xuất | Lý do + bằng chứng | Ảnh hưởng nếu không đổi | Quyết định PM |
| --- | --- | --- | --- | --- | --- | --- |
| P-91-1 | research (TL-2 đề xuất 4) | UX-16: "số cố định hai chữ số thập phân" làm tròn khi hiển thị (`7,915` → `7,92`); "cuộn theo trang" phá ảo hoá | Đệm số 0 tới số chữ số lẻ lớn nhất của cột, **không làm tròn**; vùng cuộn cao bằng khung nhìn, giữ ảo hoá | Điểm hiển thị phải đúng chuỗi decimal của máy chủ (nguyên tắc 5); `Gradebook.tsx` dòng 18 | Sổ điểm hiện số khác số đã tính | **CHẤP NHẬN** (PM 2026-11-08) |
| P-91-2 | research (TL-2 đề xuất 5) | Một số phần của mẫu BA thêm JS vào `/`, trùng thứ đã có hoặc cần API ngoài sprint giao diện | Bỏ: dự phòng `IntersectionObserver` cho `view()`; tab GV / SV ở trang giới thiệu (thay bằng hai cột, mỗi vai 4 mục); [ĐỔI DESIGN] nút `link` (dùng `ghost`); phép bắt tự động "Panel chứa khung có viền"; gợi ý "chủ đề sai nhiều" ở chat (ghi Nợ) | TL-2 | Phạm vi `FEAT-uiux-91` nhỏ lại, không mất mục tiêu | **CHẤP NHẬN** (PM 2026-11-08) |
