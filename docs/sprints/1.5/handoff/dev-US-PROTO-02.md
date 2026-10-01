# DEV handoff — US-PROTO-02 Giảng viên: vận hành lớp
Nhánh `sprint/1.5-mock-ui` · commit: `US-PROTO-02: ...`

## Đã làm (theo thứ tự lát dọc)
- Mock data: `frontend/src/mock/{roster,staff}.ts`, điền `TICKETS_SEED` trong `frontend/src/mock/support.ts` (5 ticket mở + 1 đã có người nhận + ticket D3 khi SV B gửi).
- Màn hình tính năng (`frontend/src/features/`):
  - `today/StaffHome.tsx` + `StaffHome.module.css`: '6 việc cần xử lý hôm nay' theo F14 (ticket chờ > 72h đứng đầu, điểm danh buổi 10 lớp 761987, BT03 có 4 bài cần xem kỹ, 1 câu AI chờ xác nhận, thiết lập lớp mới 761988, 3 yêu cầu vào lớp chờ duyệt). Ghi tên lớp ở từng việc; việc đã xử lý biến khỏi danh sách; khối 'Lớp cần chú ý' (C + 2 SV khác); dải lịch sắp tới; không thẻ số liệu/biểu đồ ở khung nhìn đầu; nút đỏ chỉ việc khẩn nhất. TA thấy bản của TA (không có việc thiết lập lớp).
  - `inbox/`: split 380px (danh sách → chi tiết trên mobile). 5 ticket mở + 1 đã nhận (Phạm Quốc Bảo nhận lúc 09:12 thay nút Nhận, 409 giả lập) + ticket D3. Nhận → Claimed; trả lời D4 → Answered + dòng gửi thư thông báo tới email SV (mô phỏng); tuỳ chọn Lưu thành tri thức; ghi `answer` vào ticket.
  - `students/`: tìm kiếm tên/MSSV, chip lọc (Cần chú ý, Vắng nhiều, Điểm giảm, Ít hoạt động); bảng SV 30 người theo lớp; đọc attendanceStats + attendance state. `students/[id]` (Hồ sơ 360): câu rủi ro tiếng Việt thường, 5 tab, ghi chú PrivateMark thêm tại chỗ + Hoàn tác, nút Nhắn riêng.
  - `attendance/`: buổi 10 hôm nay 09:00–11:30, 30 SV mặc định Có mặt; chọn buổi 1–15; phím tắt ↑/↓ chọn hàng, 1 Có mặt, 2 Muộn, 3 Vắng phép, 4 Vắng, P +phát biểu (+0,25); chạm ≥44px trên mobile; mỗi thao tác hiện 'Đã đánh vắng · Hoàn tác' 5s + lưu 09:21; công tắc Giả lập mất mạng → hàng đợi đồng bộ; Lưu điểm danh → 'Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng' và ghi finalized=true, làm `/me` của B lên 8,5.
  - `members/`: lớp 2: 24 thành viên + 3 chờ duyệt (+ SV D); Duyệt / Từ chối tại hàng + Hoàn tác; GV: Tạo lại mã (qua ConfirmIrreversible), Sao chép link, Mời trợ giảng; TA: xem và duyệt, không có nút 'Tạo lại mã' hay 'Mời ra khỏi lớp'.
- Routes app: `frontend/src/app/(app)/{inbox,students,students/[id],attendance,class/members}/page.tsx`.

## File đổi
`frontend/src/features/{inbox,students,attendance,members}/**`
`frontend/src/features/today/StaffHome.tsx`, `StaffHome.module.css`
`frontend/src/mock/{support,roster,staff}.ts`
`frontend/src/app/(app)/{inbox,students,students/[id],attendance,class/members}/page.tsx`
`docs/sprints/1.5/shots/02-*.png`

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile
pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh
F=http://localhost:3000 bash docs/sprints/1.5/qc/scripts/proto-curl.sh tc_02_01 tc_02_07
```

## Test đã chạy và kết quả
- `proto-curl.sh tc_02_01 tc_02_07`: PASS 100% (mọi route mở được với GV; SV/Admin bị chặn; TA không thấy 'Tạo lại mã' và 'Mời ra khỏi lớp').
- Đã chạy điểm danh buổi 10 bàn phím và chạm, dưới 60s.
- Trình duyệt: ảnh chụp desktop 1440 và mobile 390 tại `docs/sprints/1.5/shots/02-*.png`.

## AC tự đánh giá
02-AC1 ✓ · 02-AC2 ✓ · 02-AC3 ✓ · 02-AC4 ✓ · 02-AC5 ✓ · 02-AC6 ✓ · 02-AC7 ✓.

## Nợ / chưa làm / cần hỏi
- Công tắc "Giả lập mất mạng" là mô phỏng phía client qua state cờ tạm.

## Cập nhật theo spec v4 (Proposals #15, #16)
- Phân định rõ 2 panel độc lập ở `/inbox` (FR-X11, 02-AC5): cột danh sách ticket (bên trái, max 380px) và panel chi tiết ticket (bên phải), đều có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc `var(--ep-radius-md)`, padding đầy đủ, cuộn riêng biệt.
- Thao tác kiểm duyệt câu trả lời AI ở `/threads/[id]` cho GV/TA (02-AC8): Chỉnh sửa câu trả lời inline → Lưu và xác nhận (`Đã được giảng viên sửa & xác nhận`), hiển thị đối chiếu câu trả lời AI gốc; Xác nhận (`Đã được giảng viên xác nhận`); Loại khỏi tri thức kèm thanh Hoàn tác.
