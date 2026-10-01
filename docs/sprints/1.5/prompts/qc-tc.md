# Prompt cho `qc` — test case prototype (pha 1)

Đọc `docs/team/CONTEXT.md`, spec `docs/specs/FEAT-prototype-ui/` (v2 APPROVED). Dev đang thi công US-PROTO-01…04 song song; US-PROTO-00 đã xong (handoff `docs/sprints/1.5/handoff/dev-US-PROTO-00.md`).

## Việc
1. Viết TC hộp đen từ AC, không đọc code: `docs/sprints/1.5/qc/tc-US-PROTO-0N.md` cho N = 0…4. Mỗi AC ≥ 1 TC; có TC phân quyền (ma trận SRS mục 2: gõ thẳng URL bằng cookie vai khác), TC nhánh lỗi (`?state=…`), TC 375/390 px cho màn SV + `/attendance` + `/inbox`, TC "sinh viên không thấy từ kỹ thuật AI".
2. Một TC xuyên suốt: **đi trọn `docs/DEMO_SCRIPT.md` bước 1–7** trên prototype, một trình duyệt, đổi vai (bỏ Mailpit theo FR-X7), kiểm các con số SRS 4.1 (8,3 → 8,5 → 8,8; What-if 8,3).
3. Script tự động được (Playwright duyệt route + chụp ảnh 1440/390) đặt ở `docs/sprints/1.5/qc/scripts/`; không đưa vào `frontend/` (prototype không có Playwright cố định — SRS mục 9). Ảnh chụp của QC vào `docs/sprints/1.5/qc/shots/`.
4. Ngay khi PM báo "US-PROTO-0N có handoff", chạy TC của story đó → `docs/sprints/1.5/qc/report-US-PROTO-0N.md`. Có thể chạy US-PROTO-00 ngay bây giờ (đã có handoff).

Chất lượng giao diện tính như AC: kiểm DESIGN.md §21 (phản mẫu) và §22 (định nghĩa xong) bằng mắt trên ảnh chụp; lỗi thẩm mỹ rõ ràng (vỡ bố cục, chữ tràn, khoảng trắng hụt, đỏ lạm dụng) ghi BUG mức trung bình.

Commit chỉ `docs/sprints/1.5/qc/**`. Trả lời ≤ 6 dòng sau khi viết xong TC + chạy US-PROTO-00, rồi chờ PM.
