# Triage 118 F — sprint 9.1 @ be0c0d96 (qc2, 2026-11-15) — lượt 1, chưa chạy lại hết

Nguồn: `logs/*.log` + `*.rerun.log`; `/tmp/qc2-s91/red-map.json` (test đỏ → TC). Kết luận (b) dựa vào văn bản US/proposal, (c) dựa vào dữ liệu ep-r; nhóm (c) phải chạy lại tới P/F (đang làm), chưa phải lỗi mã.

| Nhóm | Số TC | Mã ca | Triệu chứng | Kết luận |
| --- | ---: | --- | --- | --- |
| G1 Landing đổi bởi 10.7 | 36 | UX08-1,3,4,6-12,14,15,22-27,29-36,38-42,46,53,56; UX09-17,18 | h1 «Vận hành lớp học có trợ lý AI» (8 từ), 2 nút Đăng nhập + Xem video, hết clip/poster, mục ghim/số đo/vai đổi, không `view-timeline` mà IntersectionObserver, nút Tạm dừng 32 px | (b) LT: `docs/specs/FEAT-uiux-91/US.md` US-08 AC1-4,6,8,10,13,15,21 + US-09 AC2,5 «Sửa 10.7»; P-107-9/11/12/13. Cần viết lại TC theo AC mới (spec bổ sung `qc2-r91/`) → ca nào còn nghĩa sẽ chạy lại, ca hết nghĩa ghi LT |
| G2 Trạng thái rỗng | 5 | UX05-1,2,3,5,11 | EmptyState chỉ còn tiêu đề «Chưa có câu hỏi cần xử lý» (mất câu «Khi AI không đủ chắc chắn…») | (b) LT: P-107-2 (rỗng = tiêu đề ngắn + tối đa 1 nút, luật chữ UI); DESIGN §15, UX.md đã vá |
| G3 Dữ liệu ep-r bị đổi / đua | 54 | UX12-1..20, UX14-*, UX15-*, UX16-5/15/16/19/20/24, UX04-12/13/14 | 125 bản nháp chấm / 33 SV / grade_entries có MANUAL v9 trên 761987 (do spec cũ + 7 agent khác); «Cannot read 'id'», bài trùng 7 dòng, 0 hàng chờ, điểm API ≠ CSV 7/30 | (c) môi trường: chạy lại trên lớp riêng hậu tố `r91`; chưa có bằng chứng lỗi mã |
| G4 Phiên/đăng nhập/timeout | 18 | UX01-1,5; UX06-2..7,PQ; UX13-14,18,21-25; UX19-6,12 | `waitForURL` đăng nhập 30 s, «Cannot read 'find'» (me/courses lỗi lúc gateway `draining`/NOT_READY), topbar mặc định lớp QD8C của agent khác (teacher có 20 lớp) | (c) môi trường: chạy lại (đang chạy w2); UX06-3/4/5 cần tài khoản GV riêng 2 lớp |
| G5 Trình duyệt thiếu | 4 | UX08-28,48; UX09-11,13 | WebKit/Firefox chưa cài | (c) đã `playwright install webkit firefox`; chạy lại |
| G6 Lỗi mã nghi | 1 | UX08-50 | axe `color-contrast` ×5 (1440) / ×12 (375) trên `/` mới; US-08 AC16 (0 serious/critical) còn hiệu lực | (a) nghi hồi quy sprint 10.7 lô 9 (`Landing.module.css`): xác nhận lại ở lượt chạy axe mới |
