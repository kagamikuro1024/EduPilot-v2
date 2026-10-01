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

## v5 / v5.1 — bổ sung

**v5:** 02-AC9 `/inbox` `?ticket=` + `← Hộp thư`; 02-AC10 ô trả lời cách khối thông tin ≤ 24 px; 02-AC11 thread mới của SV thành việc "Câu hỏi mới" ở Hôm nay + chuông.

**v5.1 (#19):**

| E | Đã sửa | AC |
| --- | --- | --- |
| E3 | Thẻ 761988 mang `?course=int1006-2`; `useCourseDeepLink` đặt lớp rồi bỏ tham số; tab `Chờ duyệt` mở sẵn | 02-AC12 |
| E9 | Thẻ "chờ 3 ngày 4 giờ" mở đúng `/inbox?ticket=tk-5`, danh sách cuộn tới hàng | 02-AC13 |
| E19 | Badge Hộp thư / Chấm bài tính từ `ticketStats` + `reviewPending`, giảm ngay, ẩn khi 0, cả sidebar và thanh dưới | 02-AC14 |
| E7, E8 | `/attendance` 390 / 375: ô Buổi ≥ 160 px, công tắc hàng riêng, hàng SV 2 dòng, 4 nút ≥ 44 px, chọn = tick + đậm | 02-AC15 |
| E18 | Trạng thái lưu về đúng sau offline / `Lưu điểm danh`; nút `Đã lưu` trung tính; không còn "· 0 thay đổi" | 02-AC16 |
| E16 | `Lưu và xác nhận` khoá khi rỗng / không đổi, kèm câu lý do | 02-AC17 |
| E12 | Một tập "Cần chú ý" (`attentionSet`): chip, bộ lọc, cột Rủi ro, Hôm nay | 02-AC18 |
| E28 | Thứ tự `sv-n` ở mọi màn (`rosterOf`) | 02-AC19 |
| E29 | Thẻ "N bài cần xem kỹ" nêu đúng lý do thật (N8) | 02-AC20 |
| – | Thứ tự việc Hôm nay + `data-part=today-task` / `data-task-id` (J3) | 02-AC21 |

### AUDIT v5.1 (prod/dev server :3300, trình duyệt headless)

Chạy `AUDIT` + `TOUCH` + `LEFT` (US.md "Quy ước kiểm chung") trên mọi route × vai: 135 lượt đo (SV 15 route × 1440/390/375; TA 17 + GV 20 + Admin 6 route × 1440/390 (+ 375 cho `/attendance`, `/inbox`)).

| Vai | Số lượt | `ox` ≠ 0 | `cut` | `ell` | `TOUCH` (≤ 390) |
| --- | --- | --- | --- | --- | --- |
| Sinh viên | 45 | 0 | 0 | 0 | 0 |
| Trợ giảng | 36 | 0 | 0 | 0 | – |
| Giảng viên | 42 | 0 | 0 | 0 | 0 (`/attendance`, `/inbox`) |
| Admin | 12 | 0 | 0 | 0 | – |

`LEFT` (`[data-part=page-title]`): mọi route = **240** ở 1440, **16** ở 390 và 375. `curl` bộ `proto-curl.sh all` (bản đã sửa lỗi biến `$s»` của bash UTF-8): 496 PASS, 1 FAIL giả ở `tc_04_08` — xem góp ý #23 (`grep -c` đếm dòng, HTML SSR chỉ một dòng; `grep -o … | wc -l` ra đúng 3). `pnpm -C frontend lint`, `tsc --noEmit`, `bash scripts/ui-antipatterns.sh` sạch.
Ảnh: `docs/sprints/1.5/shots/v5/` (thread, danh sách, Hôm nay GV, inbox 1440/375, điểm danh 390, chat 390/900/1440, SV D chưa vào lớp, `/settings/llm`, luyện đề 390, analytics).
