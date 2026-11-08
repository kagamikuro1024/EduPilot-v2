# QC đợt 2 — Chữ và điều hướng (US-UX91-01, 05, 06, 07)

**Kết luận: FAIL.** US-06 FAIL (AC3). US-05 PASS. US-07 PASS (ở `62f4833b`). US-01: các TC đã chạy PASS, còn 7 TC chưa chạy (liệt kê cuối file), chưa kết luận.

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
