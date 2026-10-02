# DEV handoff — US-PROTO-04 Hiểu lớp + hệ thống
Nhánh `sprint/1.5-mock-ui` · commit: `US-PROTO-04: ...`

## Đã làm (theo thứ tự lát dọc)
- Mock data: `frontend/src/mock/{insights,analytics,system}.ts` (bộ báo cáo lỗ hổng kiến thức 2 lớp, telemetry 50 yêu cầu observability, cấu hình 3 provider LLM và tích hợp Mail/IMAP/Teams).
- Màn hình tính năng (`frontend/src/features/`):
  - `today/AdminHome.tsx`: ActionList việc vận hành (Gemini lỗi 3 lần trong 15p; sắp chạm 80% ngân sách tháng; 2 việc dead-letter; lớp 761988 chưa có giảng viên hoạt động); bấm việc → đúng màn; rỗng 'Hệ thống đang vận hành bình thường'.
  - `insights/`: lớp 1: chủ đề A (AES/CBC) và B (Hàm băm/chữ ký) đứng đầu, câu mẫu đã ẩn danh (không tên, không MSSV 2022xxxx); chủ đề < 3 SV hỏi không có câu mẫu; mục Tài liệu chưa đề cập. Lớp 2: chủ đề C (Tường lửa) đứng đầu. Tạo báo cáo mới có tiến độ → báo cáo 29/10; Tạo thread ghim ghi vào `KEYS.insightThreads`; Tạo buổi ôn tập ghi vào `KEYS.calendarExtras`.
  - `analytics/`: hoạt động học, hỗ trợ (ticket, thời gian trả lời), bảo vệ thông tin cá nhân, chất lượng chấm (tỷ lệ GV sửa điểm AI); **chi phí chỉ GV thấy, TA không thấy (chuỗi 'chi phí' không xuất hiện trong HTML của TA)**. Đổi khoảng 7/30 ngày; biểu đồ SVG sạch sẽ, không thẻ KPI.
  - `observability/`: dải trạng thái (1.284 yêu cầu hôm nay, p95 3,1s, lỗi 0,6%, model dự phòng 1,2%, chi phí 42.000đ), bảng 50 yêu cầu. Admin: mở Drawer bắt buộc nhập lý do kiểm toán trước → hiện nội dung đã che (`[[SV_1]]`), link chi tiết, ghi 'Đã ghi nhật ký kiểm toán'. **GV: chỉ thấy dải + số tổng hợp lớp mình, hàng không mở được và chuỗi `[[SV_` không xuất hiện trong HTML của GV.**
  - `settings/llm`: 3 provider, bảng tác vụ → model, chuỗi dự phòng, embedding tách riêng, ngân sách 62%. Admin: Test kết nối theo hàng (412ms / lỗi mẫu có cách khắc phục), đổi model CHAT tại chỗ + Hoàn tác, khoá API chỉ ghi ('••••3f9a · đã kết nối'). **GV: chỉ xem, không có nút Test kết nối và không có nút sửa.**
  - `settings/integrations`: Mail (đã kết nối, kiểm 08:55), IMAP (chưa cấu hình), Teams (cần quản trị viên trường đồng ý). Gửi thư thử / Kiểm tra kết quả tại chỗ (Admin; GV chỉ xem).
  - `admin/courses`: 2 lớp 761987 và 761988; Mở lớp (form tại chỗ) gán GV; Lưu trữ lớp qua ConfirmIrreversible.
  - `admin/users`: lọc theo vai (GV, TA, Admin, 57 SV); Mời giảng viên (link mời 72h); khoá/mở khoá tài khoản + Hoàn tác; không có tính năng tạo SV.
- Routes app: `frontend/src/app/(app)/{insights,analytics,observability,settings/llm,settings/integrations,admin/courses,admin/users}/page.tsx`.

## File đổi
`frontend/src/features/{insights,analytics,observability,settings,admin}/**`
`frontend/src/features/today/AdminHome.tsx`
`frontend/src/mock/{insights,analytics,system}.ts`
`frontend/src/app/(app)/{insights,analytics,observability,settings/llm,settings/integrations,admin/courses,admin/users}/page.tsx`
`docs/sprints/1.5/shots/04-*.png`

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile
pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh
F=http://localhost:3000 bash docs/sprints/1.5/qc/scripts/proto-curl.sh tc_04_01 tc_04_02 tc_04_03 tc_04_04 tc_04_06
```

## Test đã chạy và kết quả
- `proto-curl.sh tc_04_01 tc_04_02 tc_04_03 tc_04_04 tc_04_06`: PASS 100% (/insights không lộ tên/MSSV; GV không thấy `[[SV_` ở /observability; GV không thấy 'Test kết nối' ở /settings/llm; TA không thấy 'chi phí' ở /analytics; phân quyền chặn đúng).
- Trình duyệt: ảnh chụp desktop 1440 và mobile 390 tại `docs/sprints/1.5/shots/04-*.png`.

## AC tự đánh giá
04-AC1 ✓ · 04-AC2 ✓ · 04-AC3 ✓ · 04-AC4 ✓ · 04-AC5 ✓ · 04-AC6 ✓.

## Nợ / chưa làm / cần hỏi
- Chi tiết yêu cầu ở Observability của Admin mở drawer hiển thị log kiểm toán mô phỏng.

## Cập nhật theo spec v4 (Proposals #16)
- Phân định rõ các khối cấu hình thành các panel sạch sẽ (`configPanel`, `integration`): viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc `var(--ep-radius-md)`, padding `var(--ep-space-6)` ở `/settings/llm` và `/settings/integrations`.
- Danh sách chủ đề ở `/insights` được đóng gói thành các panel độc lập rõ ràng (`.topic`, `.gap`).

## v5 / v5.1 — bổ sung

**v5:** 04-AC7 `/settings/llm` lưới 3 cột (`provider-status`, `provider-action` thẳng hàng), xếp chồng < 720 px.

**v5.1 (#19):**

| E | Đã sửa | AC |
| --- | --- | --- |
| E6, E25 | `ticketStats` là nguồn duy nhất: "Chờ > 24 giờ" = 3 khớp hộp thư; "AI tự trả lời" = `aiShare` (98% / 99%) | 04-AC8 |
| E17 | `Tạo thread ghim` idempotent; nút thành "Đã ghim · Xem thread" | 04-AC9 |
| E30 | `ASSIGNED_AT` (28/10 16:40) là nguồn mốc phân công → "hôm qua 16:40" | 04-AC10 |
| E35 | `/settings/llm` khoảng cách giữa các `settings-section` đều | 04-AC11 |
| E37 | Biểu đồ "Hoạt động học" có trục, điểm (`chart-point`), nhãn trục, giá trị cao nhất / thấp nhất | 04-AC12 |

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

## Sửa lỗi QC v5 vòng 1 (US-PROTO-04)

| Lỗi | Đã sửa | Cách tự kiểm |
| --- | --- | --- |
| BUG-v5-04-2 `/analytics` 7→30 ngày lệch | `piiChannels` theo [7, 30 ngày]; `piiDetected` = tổng các kênh | 30 ngày: 19+7+5 = 31; lớp 761988: 3+2 = 5 |
| BUG-v5-04-1 lý do xem nội dung không báo lỗi tại ô | `Field error` + `aria-invalid` khi trống sau Tab hoặc < 10 ký tự (trả lời câu hỏi: nút vô hiệu **chưa đủ**, nay có dòng lỗi) | Mở hàng, gõ 4 khoảng trắng, Tab |
| BUG-v5-04-3 trạng thái rỗng chung | Rỗng riêng + một hành động cho `/settings/llm`, `/settings/integrations` | `?state=empty` |
| BUG-v5-04-7 số đếm không khớp rỗng | Admin Hôm nay / Lớp học / Người dùng: 0 khi `state=empty` | '0 việc đang chờ', '0 lớp đang chạy', '0 tài khoản' |
| BUG-v5-04-4/5 /admin/users 720, /observability 375 | Gốc chung ở 00 (DataTable cuộn + `sectionHead` xuống dòng) | AUDIT |
| BUG-v5-04-6 gạch thừa trong panel embedding | `.configPanel dl` không kẻ hàng đầu/cuối | `/settings/llm` |
| BUG-v5-DEMO-10 khối rỗng ~270 px trắng | `EmptyState` `align-content: start` | `/inbox` lớp 761988: tiêu đề và câu giải thích cách nhau 31 px |

**Tự kiểm (build production :3400, trình duyệt headless):** `regress-v24.mjs` 30/30 PASS · `audit.mjs` 495/495 PASS (SV 165, TA 106, GV 170, Admin 54) và `states:true` GV + Admin 544/544 PASS · `demo-run.mjs` 21/21 hàng PASS, 173 s (< 13:45) · `proto-curl.sh all` 497 PASS / 0 FAIL · `pnpm lint` sạch · `pnpm build` OK · `bash scripts/ui-antipatterns.sh` 0 ✗. Commit: `4f7b5b5` (00) · `5086445` (02) · `821be9d` (01) · `0fc7ffb` (03) · `94dbf38` (04) · `6a043ae`.
