# QC đợt 2 — Chữ và điều hướng (US-UX91-01, 05, 06, 07)

**Kết luận (kiểm lại 2026-11-09 ở `6a0e8584`): FAIL.** US-05 PASS. US-07 PASS. US-01 FAIL (AC1 #3, AC5 #34). US-06 FAIL (AC3, lớp đã lưu trữ).
Lượt 1 (ở `62f4833b`) là FAIL; bảng lượt 1 giữ nguyên bên dưới làm lịch sử.

## Kiểm lại ở `6a0e8584` (2026-11-09)
Stack `-p ep-qc` dựng lại từ `6a0e8584`. Spec: `qc-uiux91-w2-real.spec.ts` (40 ca), `qc-uiux91-w12b-real.spec.ts`, `qc-uiux91-w12c-real.spec.ts`; chạy `--workers=1`. Ba ca (UX05-11, UX06-1, UX06-3/4/5) đỏ ở lần chạy chuỗi vì `/me/courses` trả lỗi giữa chừng (nhiễu như lượt 1); chạy riêng cả ba pass.
| TC | Trước | Nay | Ghi chú |
| --- | --- | --- | --- |
| UX06-6 | FAIL | PASS | `ep:ui:course` = UUID lạ → xoá (`null` sau đăng nhập), chọn lớp theo AC2, không `pageerror` (GV và SV) |
| UX06-7 | chưa chạy | **FAIL (Thấp)** | Admin mở lớp thử `QC196964` cho `teacher`, đặt `ep:ui:course` = lớp đó, admin lưu trữ (200). `/me/courses` vẫn trả lớp với `status: ARCHIVED`; `ep:ui:course` giữ nguyên, lớp lưu trữ vẫn là lớp đang chọn. AC3 ghi «lớp không còn trong `me/courses` (đã lưu trữ, bị gỡ)» → cần BA chốt: lớp `ARCHIVED` còn trong danh sách thì có được chọn không (P-91-9) |
| UX06-PQ | chưa chạy | PASS | SV `sv.gioi` với `ep:ui:course` = lớp mà SV không thuộc: `GET /courses/‹id›/today` = 403, giá trị bị xoá, vào lớp của mình |
| UX01-2 | chưa chạy | PASS | `grep` 27 chuỗi «Xoá» của bảng, trừ `Demo*` và `mock/`: 0 kết quả |
| UX01-4 | chưa chạy | **FAIL (Thấp)** | #3: không có tooltip nút `Mở lớp` («Người được gán nhận thông báo kèm mã tham gia.» không có trong mã; nút không `title`, không `aria-describedby`). #4, #18, #20, #25, #28, #29 (`AN7K2MQ`), #42 đạt |
| UX01-5 | chưa chạy | PASS | #48 «24 sinh viên» (không mã lớp), #11 «buổi 4 · Hôm qua, 13:30–16:00 · P.405» (không mã lớp), MSSV còn |
| UX01-8 | chưa chạy | PASS | `lcp.spec.ts` 7 route pass trong suite |
| UX01-9 | chưa chạy | PASS có giới hạn | `PIIChannelDialog.tsx:34`: «Hỏi đáp công khai là nơi cả lớp cùng đọc…»; không dựng được hộp thoại trên stack thật |
| UX01-11 | chưa chạy | **FAIL (Thấp)** | #34 (`Switch` «Cần giảng viên duyệt»): chỉ `title` native, không `aria-describedby` (dev đã tự khai ở đợt 1: không hiện khi focus bàn phím). #4 có `aria-describedby`. Chữ ≤ 12 từ (6–10 từ) |
| UX01-16 | chưa chạy | PASS | `/students` CLS 0,015, `/class/members` CLS 0,001; không dòng trống dưới `h1` |
| UX07-9 | chưa chạy | PASS | Thông báo thật của GV (13), TA (6), SV (7; loại `TICKET_*`, `POST_REPORTED`, `AI_PENDING`, `JOIN_REQUEST`…): không «Threads» / «Insights» / «Analytics» |
Các TC khác của đợt 2 chạy lại đều PASS (UX01-1, 3, 6, 7, 10, 12–15, 17–19, PQ; UX05-1…12; UX06-1…5, 8, 9; UX07-1…8, 10…17, PQ).

---
# Lượt 1 ở `62f4833b` (lịch sử)

## Môi trường
Như `wave-1.md`: commit `62f4833b` (chứa `4b6389f1` đợt 1 + 2), VPS `/srv/work/qc`, stack `-p ep-qc` (seed thật, tài khoản `admin|teacher|ta|sv.gioi@edupilot.local`), Caddy `https://localhost:14443`.
Spec QC: `frontend/e2e/qc-uiux91-w2-real.spec.ts` (40 ca, stack thật), `qc-uiux91-w12b-real.spec.ts` (7 ca). Lần chạy đầy đủ: 34 pass, 6 fail. 5 trong 6 ca fail xuất hiện khi chạy liền nhiều đăng nhập (nghi giới hạn đăng nhập của gateway, [INFERENCE]); **chạy lại riêng cả 5 đều pass**. Ca còn lại (UX06-6/7) là lỗi sản phẩm.
Lúc `4b6389f1`: US-07 AC4 FAIL vì `Landing.tsx:46` còn `["Threads", …]` (dev handoff nói đã đổi, commit không có). Đã sửa ở đợt 6 (`Landing.tsx:55` «Hỏi đáp công khai»); grep ở `62f4833b` không còn chuỗi hiển thị «Threads», chỉ còn chú thích mã.

## Kết quả theo AC

### US-UX91-01 — Bỏ câu mô tả thừa, thay bằng tooltip / meta
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX01-1 | 1 | PASS | 20 màn × vai (admin `/admin/courses`, `/admin/users`, `/observability`; teacher `/analytics`, `/exams`, `/insights`, `/class/*`, `/questions`; student `/exams`, `/join`, `/me`, `/practice*`), cả trạng thái mở biểu mẫu: không còn câu cũ |
| UX01-3 | 1, 3 | PASS | Khung máy chủ (`curl /`, `/threads`, …): không câu phụ, dùng «Hỏi đáp công khai». Ở `4b6389f1` FAIL do Landing |
| UX01-6 | 2 | PASS | Diff 3 tệp `Demo*.tsx` (7 dòng) chỉ đổi nhãn «Threads» → «Hỏi đáp công khai» (US-07 AC4) và `progressStyle` (US-04 AC4); không đổi `description`. TC ghi «không đổi» quá cứng, ghi nhận theo US-07 AC4 / US-04 AC4 |
| UX01-7 | 3 | PASS | `lcp.spec.ts` 7 route pass |
| UX01-10 | 4 | PASS | Hộp xác nhận `Chốt điểm` còn mô tả hệ quả |
| UX01-12 | 5 | PASS | Không thêm icon `Info` / `CircleHelp` |
| UX01-13 | 6 | PASS | Tệp `.tsx` có `<PageHeader description="…dài hơn 40 ký tự">` → `ui-antipatterns.sh` bắt (exit 1, «Câu mô tả tĩnh dưới tiêu đề») |
| UX01-14 | 6 | PASS | `description` ≤ 40 ký tự / biểu thức động không bị bắt (exit 0) |
| UX01-15 | 6 | PASS | `--selftest` 23 / 23 phép bắt được |
| UX01-17 | 8 | PASS | 375 px, 7 route: không dòng trống dưới `h1`, không tràn ngang |
| UX01-18 | 9 | PASS | `/observability`, `/analytics` không câu mô tả (qua UX01-1) |
| UX01-19 | 10 | PASS | ảnh sau `shots/wave-2/{home,chat,threads,inbox,gradebook}-{1440,375}.png`; mắt thường: tiêu đề + nội dung, không câu phụ |
| UX01-PQ | — | PASS | SV `/observability`, `/analytics` → 403 giữ nguyên |
| `ui-antipatterns.sh` | — | PASS | Sạch ở `62f4833b` |
| UX01-2, 4, 5, 8, 9, 11, 16 | 1, 2, 3, 4, 5, 7 | CHƯA CHẠY | Xem cuối file |

### US-UX91-05 — Trạng thái rỗng: PASS
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX05-1, 2, 3 | 1 | PASS | Hộp thư rỗng: `EmptyState` «Chưa có câu hỏi nào cần thầy/cô xử lý.», không «Chọn một câu hỏi», nút ghost căn trái |
| UX05-4 | 2 | PASS | Chi tiết ticket không còn «Đã chờ …» / «Trạng thái …» lặp |
| UX05-5, 6 | 3 | PASS | `/grading` chưa có bài nộp: `EmptyState` + `Tải bài lên`; TA thấy nút (1) và không bị chặn |
| UX05-7, 8 | 4 | PASS | Lớp chưa có công thức (761988): một nút chính `Thiết lập công thức`, không câu «Điểm chính thức…». Lớp 761987: câu đó hiện đúng một lần ở chân bảng |
| UX05-9 | 5 | PASS | Lỗi tải: `ApiErrorNotice` + `Thử lại`, không `EmptyState` |
| UX05-10 | 6 | PASS | SV `/inbox`, `/grading`, `/gradebook` → «Bạn không có quyền xem màn này», không `EmptyState` của giảng viên |
| UX05-11 | 7 | PASS | 375 px: không tràn, nút ≥ 44 px |
| UX05-12 | 8 | PASS | ảnh `inbox-1440`, `inbox-375` |

### US-UX91-06 — Nhớ lớp, lớp mặc định: FAIL
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX06-1, 2 | 1 | PASS | Chọn lớp, đăng xuất / đăng nhập lại, hoặc tải lại: lớp được giữ (`ep:ui:course`) |
| UX06-3, 4, 5 | 2 | PASS | Lớp mặc định = lớp đầu tiên có việc trong `today` (teacher: 761988) |
| UX06-6 | 3 | **FAIL (Thấp)** | `ep:ui:course` = UUID lạ: không lỗi, không trang trống, chọn lớp theo AC2, nhưng giá trị **không bị xoá**: sau 4 s `localStorage['ep:ui:course']` vẫn là `00000000-0000-7000-8000-00000000dead`. AC3 ghi «bỏ giá trị». Tái hiện: `addInitScript` đặt giá trị, đăng nhập teacher, đọc `localStorage` |
| UX06-7 | 3 | CHƯA CHẠY | Lớp đã lưu trữ (cần admin lưu trữ lớp trên `ep-qc`); cùng nguyên nhân với UX06-6 nên đang tính theo UX06-6 |
| UX06-8 | 4 | PASS | SV / TA: bộ chọn lớp chỉ có lớp của mình, không mục quản lý |
| UX06-9 | 5 | PASS | 375 px: mục chọn lớp cao ≥ 44 px |
| UX06-10 | 6 | PASS | `visual.spec.ts` pass |
| UX06-PQ | — | CHƯA CHẠY | SV với `ep:ui:course` là lớp không thuộc |

### US-UX91-07 — Đổi tên: PASS
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX07-1, 2, 3 | 1 | PASS | Sidebar + `⌘K` theo vai: SV có «Hỏi đáp công khai», không «Lỗ hổng kiến thức»; GV / TA có đủ ba mục; admin không có |
| UX07-4 | 1 | PASS | 375 px: thanh dưới + «Thêm» |
| UX07-5, 6 | 2 | PASS | `/threads`, `/insights`, `/analytics`: tiêu đề = nhãn nav, `<title>`, HTML máy chủ |
| UX07-7 | 3 | PASS | `/threads/‹id›`: liên kết quay lại |
| UX07-8 | 4 | PASS ở `62f4833b` | `rg -w Threads|Insights|Analytics`: chỉ còn chú thích mã và `mock/*.ts`. **FAIL ở `4b6389f1`** (`Landing.tsx:46`) |
| UX07-10 | 4 | PASS | GV `/`: không «Threads» |
| UX07-9 | 4 | CHƯA CHẠY | Chữ thông báo `AI_PENDING` / `POST_REPORTED` |
| UX07-11 | 5 | PASS | URL cũ còn dùng được |
| UX07-12 | 6 | PASS | Thu gọn sidebar: có tên đầy đủ (`aria-label` + `title`) |
| UX07-13 | 6 | PASS | 375 px: «Hỏi đáp» (8 ký tự), không nhãn bị cắt |
| UX07-14 | 7 | PASS | `shell.spec.ts` chỉ đổi nhãn nav; không nới assertion |
| UX07-15 | 8 | PASS | Số mục nav theo vai không đổi (SV 8, TA 12, GV 15, admin 6) |
| UX07-16 | 9 | PASS | `DESIGN.md` §14.3 / §14.21 / §14.25 và danh sách nav dùng tên mới (dòng 42 còn «Hỏi đáp công khai (Threads)» trong chú giải nhóm, không phải nhãn hiển thị) |
| UX07-17 | 9 | PASS | ảnh sau `threads-sv-*`, `home-*`, `chat-*`, `inbox-*`, `gradebook-*` |
| UX07-PQ | — | PASS | `/insights`, `/analytics` mở thẳng vẫn vào đúng màn |

## Chưa chạy (làm ở vòng kiểm sửa)
UX01-2 (rg bảng «mảnh kiểm»), UX01-4 (đối chiếu chuỗi thay thế / tooltip), UX01-5 (dòng «Giữ»), UX01-8 (số đo LCP trước / sau), UX01-9 (PII `PIIChannel` ở `/threads`), UX01-11 (tooltip ≤ 12 từ), UX01-16 (`meta` rỗng, CLS), UX06-7, UX06-PQ, UX07-9.

## File
Chạy lại: `QC_REAL=1 pnpm exec playwright test -c playwright.qc-real.config.ts <spec> --project=desktop` (không đặt `QC_REAL` thì spec `*-real` tự bỏ qua, không lẫn vào suite CI).
Spec: `frontend/e2e/qc-uiux91-w2-real.spec.ts`, `qc-uiux91-w12b-real.spec.ts`, `playwright.qc-real.config.ts`. Ảnh: `docs/sprints/9.1/qc/shots/wave-2/`. Log: VPS `/srv/work/qc/out/{w2b,w12b,full}.log`.
