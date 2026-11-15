# State nhóm E — slice E_s7_p4b (SliceE) — dừng theo lệnh PM

Kết quả đã ghi: /tmp/qc2e/res/E_s7_p4b.jsonl (một TC có thể có nhiều dòng, lấy dòng cuối). Không có lỗi (F) nào đang mở; defects/E_s7_p4b.md chưa tạo (chưa có DEF).

## Đã xong (24/80), tất cả P
TC-P4-05-02, TC-P4-05-05, TC-P4-05-06, TC-P4-05-17, TC-P4-05-18, TC-P4-05-19, TC-P4-05-21, TC-P4-05-22, TC-P4-05-23, TC-P4-05-24, TC-P4-05-25, TC-P4-05-37, TC-P4-05-40, TC-P4-05-41, TC-P4-05-45, TC-P4-06-09, TC-P4-06-12, TC-P4-06-13, TC-P4-06-14, TC-P4-06-15, TC-P4-06-20, TC-P4-06-22, TC-P4-07-24, TC-P4-07-27

## Đã chạy dữ liệu nhưng CHƯA ghi rec (kết quả thấy được, cần chốt)
- TC-P4-05-21 và TC-P4-05-24: ĐÃ rec P (24: lần gọi /me/today ADMIN đầu rỗng do cache, lần sau có mục bậc 45).
- TC-P4-05-26: đã dịch thông báo stalled hôm nay sang ngày 20261113 (dedupe_key + created_at -2d); chờ tick xem có thông báo mới 20261115. Cần: đổi dedupe_key/created_at của thông báo stalled thành ngày 20261113, chờ tick → thêm 1 (20261115); lặp cho 20261114 → tổng 3, mỗi ngày 1.
- TC-P4-05-16: go test escalation PASS + s18 (T5 14:00 → stage 1) + s19 (T6 12:00 CN → stage 0); còn có thể thêm ca T7 10:00 → P.
- TC-P4-05-07: ticket t7 (id trong /tmp/qc2e/e7/s2.json) vẫn stage 0 sau nhiều tick (P tạm); cần kiểm /me/today teacher: mục TICKET lớp E7C1 không overdue.
- s4a.py (kiểm duyệt thread) bị kill giữa chừng: 06-12, 06-13, 06-14, 06-20, 06-22, 06-09 đã P; chưa ghi 06-48, 06-49, 06-51, 06-52, 06-08 (script nằm /tmp/qc2e/e7/s4a.py — sửa chạy lại từ phần "# 48"; thread tạo trước đó chỉ là dữ liệu test lớp E7C1).

## Chưa làm
TC-P4-05-07, TC-P4-05-11, TC-P4-05-13, TC-P4-05-14, TC-P4-05-15, TC-P4-05-16, TC-P4-05-20, TC-P4-05-26, TC-P4-05-27, TC-P4-05-28, TC-P4-05-30, TC-P4-05-33, TC-P4-05-34, TC-P4-05-42, TC-P4-05-44, TC-P4-05-46, TC-P4-05-49, TC-P4-05-50, TC-P4-06-08, TC-P4-06-25, TC-P4-06-26, TC-P4-06-28, TC-P4-06-29, TC-P4-06-30, TC-P4-06-33, TC-P4-06-34, TC-P4-06-36, TC-P4-06-38, TC-P4-06-39, TC-P4-06-40, TC-P4-06-41, TC-P4-06-42, TC-P4-06-43, TC-P4-06-44, TC-P4-06-45, TC-P4-06-47, TC-P4-06-48, TC-P4-06-49, TC-P4-06-50, TC-P4-06-51, TC-P4-06-52, TC-P4-06-53, TC-P4-07-03, TC-P4-07-04, TC-P4-07-05, TC-P4-07-10, TC-P4-07-11, TC-P4-07-15, TC-P4-07-16, TC-P4-07-17, TC-P4-07-18, TC-P4-07-19, TC-P4-07-20, TC-P4-07-21, TC-P4-07-22, TC-P4-07-23

## Ghi chú môi trường
- Lớp riêng: E7C1, E7C2 (/tmp/qc2e/e7/courses.json); lớp SLA E7S18/19/21/22/23 (dùng GV e7gv, TA e7ta tạo bằng SQL, mật khẩu seed). Đã lão hoá auth_sessions của e7gv/e7ta (4 ngày): KHÔNG gọi API bằng e7gv/e7ta nữa nếu còn muốn thử 05-26.
- Thư viện: /tmp/qc2e/e7/e7lib.py; script s1*.py (SLA), s2*.py (vòng đời ticket + dữ liệu cho autoclose), s3*.py (tri thức), s4a.py (kiểm duyệt).
- TC-05-13/14/15 (job ticket.autoclose chạy mỗi giờ): dữ liệu đã dựng trong s2.json (a8 ANSWERED 8 ngày, a6 trả lời lại, o8 OPEN, c8 CLAIMED quá 7 ngày); cần chờ job (worker/judge restart làm lệch mốc giờ) rồi kiểm.
- Tri thức lớp: embedding giả chỉ truy xuất được khi câu hỏi trùng đúng chuỗi đoạn "Câu hỏi: …\nTrả lời: …" (ghi vào note 05-41).
- TC-05-20/50 (scale worker, kill worker) cần khoá global; 05-42/06-44 (embedding chết) cũng cần đổi cấu hình toàn cục; chưa làm.
- UI/Playwright (05-11/15/27/28/30, 06-34/47/50/53, 07-15) chưa chạy; phải qua /tmp/qc2e/heavy.sh.

---
## Dừng lượt 3 (qc2/SmkE, theo lệnh PM «tối đa 2 agent»)
- Đã hợp nhất mọi kết quả `/tmp/qc2e/res/*.jsonl` vào smoke-6/7.md. Slice A (P8-01/02), B (P3-01..04), D (P4-01..04), F (P5-01..03) bị dừng sớm vì tải máy 40+; xem `/tmp/qc2e/slices/*.md` để biết ca còn lại (so với `res/*.jsonl`).
- TC-P3-02-21 (DEF-R6-02): retest chờ ảnh ep-r mới (script `/tmp/qc2e/s6_roster2.py`, 3 ca: vào mã / duyệt / mời ra).
- TC-P8-01-05, TC-P4-05-49 trả về N (không phải F).
