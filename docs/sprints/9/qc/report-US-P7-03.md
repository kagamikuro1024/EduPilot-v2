# QC report — US-P7-03 (nộp bài của sinh viên: presign, sniff, phiên bản, nộp muộn, `/assignments/[id]`) · Kết luận vòng 1: **FAIL** (chỉ AC9: LCP — PM #10 xác nhận 6,1 s, chưa đạt; mọi AC còn lại PASS)
Ngày 2026-10-31. Commit kiểm: `sprint/9-p7` @ `e424a20b`. Bộ TC: `tc-US-P7-03.md`. Handoff: `dev-US-P7-03.md`. CI `e424a20b`: **success**.

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh` theo gói, `-race` (không `-tags integration`/`testroutes` đúng theo handoff; QC dùng `integration` / `testroutes`): assignment 28, calendar 3, httpapi 3, today 1 | PASS 35/35 |
| Hộp đen trên stack seed thật (`scripts/p7-api-03.py`) | PASS toàn bộ ca: presign `.exe` / 20 MiB+1 / rỗng → 422 đúng mã, `.PDF` hoa nhận; `.pdf` nội dung MZ, `.txt` có NUL, `.docx` không phải zip → 422 sau khi PUT; nộp 201, PUT 200, `submitted_at` do máy chủ, sinh viên = B từ JWT, `source=IN_APP`, `MATCHED`; cùng `Idempotency-Key` → cùng kết quả, một bản ghi; nộp lại → v2, v1 còn; `mine` `no-store`, không khoá nháp / đáp án, bỏ qua `?student_id=`; TA / TEACHER / ADMIN gọi `mine` 403; STUDENT gọi `submissions` 403; **canary nháp** (`CANARY-DRAFT-7K3`, 7,77) ở `grading_results` DRAFT không lộ ở `mine`, danh sách, chi tiết, `me/grades`; khoá khi bản **mới nhất** `REVIEWED` / `PUBLISHED` → 409 `SUBMISSION_LOCKED` cùng một câu, DRAFT nộp lại được; muộn 8h11′ → 1 ngày, 40h → 2, 47h59′ → 2, 48h01′ → `SUBMISSION_CLOSED` không bản ghi, 1′ → 1; `allow_late=false` sau hạn → 409 đóng; gia hạn hạn nộp → `late_days` tính lại về 0, bảng không có cột `late_days` / `is_late`; chưa công bố → 404; kênh IN_APP tắt → 409 `CHANNEL_NOT_ALLOWED`; lớp khác / SV chưa vào lớp → 403; presign vượt 10 lượt/phút → 429, người khác không bị ảnh hưởng |
| Playwright desktop: `student page states`, `dropzone rejects before upload`, `submit confirms…`, `submit offline retry`, `submit double click`, `submit resume complete`, `assignment page 375`, `assignment a11y` (`--workers=2 --retries=0`) | 8/8 xanh; axe 0 lỗi (critical / serious / moderate / minor) |
| `lcp.spec.ts -g '/assignments'` | **"No tests found"** — đã bỏ khỏi `Kiểm:` theo #10; chưa có đo thay thế (xem AC9) |
| `ui-antipatterns.sh`; grep `mock|assess` | rc 0; rỗng |
| Đột biến (`/tmp/qc-mut9`, đã xoá) M1 `ceil` → `floor` · M2 khoá bỏ `PUBLISHED` · M3 sniff text bỏ NUL · M4 sniff PDF nới · M5 bỏ advisory lock | cả 5 **đỏ**: `TestLateWindowTable` / `TestLateDaysCeil` / `TestResubmitLateByOwnTime`; `TestResubmitAfterReviewedRejected`; `TestSubmitContentSniff` (M3, M4); `TestSubmitVersionAdvisoryLock` |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 `/assignments/[id]` bốn trạng thái, đề Markdown, không sidebar | PASS | `student page states` |
| AC2 kiểm tệp trước khi tải (đuôi, 20 MiB, rỗng) + kiểm máy chủ | PASS | `dropzone rejects…` không có request `presign`; API ba mã 422 |
| AC3 presign → PUT kho → submit, checksum ký, định danh từ JWT, Idempotency, rate limit | PASS | API; `TestPresignChecksumSigned`, `TestSubmitIdentityFromJWT` |
| AC4 nộp lại tạo phiên bản, khoá khi `REVIEWED` / `PUBLISHED` (câu chung) | PASS | M2; câu lỗi cùng chữ cho hai trạng thái; xem BUG-P7-03-2 |
| AC5 nộp muộn `⌈…⌉`, không lưu `late_days` | PASS | M1; API bảng biên |
| AC6 mạng chập chờn: giữ tệp, thử lại, bấm đúp, hoàn tất dùng lại `upload_id` | PASS | ba spec |
| AC7 bài chưa công bố / đóng / kênh tắt / lớp khác | PASS | API |
| AC8 `mine` chỉ của chính mình, allowlist, `no-store` | PASS | API; `TestMineDTOAllowlist` |
| AC9 375 px, trạng thái, trợ năng, **LCP ≤ 2,5 s** | **FAIL** | 375 px / a11y xanh; LCP: PM #10 ghi số đo hiện **6,1 s** (lỗi hiệu năng thật); dòng `Kiểm:` đổi sang Lighthouse CI devtools nhưng `lighthouserc.devtools.json` @ `e424a20b` chưa có URL `/assignments/{id}` (8 URL cũ). Chưa kiểm được → FAIL |
| AC10 danh sách sinh viên chỉ bài đã công bố; lịch / Hôm nay | PASS | `TestStudentListPublishedOnly`, `TestCalendarDeadlineUnion`, `TestTodayQueryBudget`; nháp BT04 không lọt |

## 3. Lỗi / ghi nhận
- **BUG-P7-03-1 (Cao, AC9 — hiệu năng):** LCP `/assignments/[id]` 6,1 s > 2,5 s (số của PM, #10). Dev sửa mã và thêm URL vào `lighthouserc.devtools.json`; QC đo lại bằng `lhci autorun --config=lighthouserc.devtools.json` (median 3 lượt).
- **BUG-P7-03-2 (Thấp, chữ ≠ SRS):** `SUBMISSION_LOCKED` hiện "Bài của bạn đang được xử lý, chưa nộp lại được." (`apierr/assignments.go`, `shared/i18n/vi.ts`); SRS ghi "Bài đã được nhận, không nộp lại được". Câu chung, không lộ trạng thái → không FAIL; thống nhất chữ với BA.
- **BUG-P7-03-3 (Thấp, tên test ≠ US):** `TestSubmitClosed`, `TestSubmitIdempotent` ở dòng `Kiểm:` không có trong repo (Go bỏ qua im lặng); `TestSubmitContentSniff` chỉ là hàm gọi `TestSubmitSniff`. Hành vi đóng / idempotent đã kiểm bằng API và `TestClosedAfterCutoff` / `TestSubmitFlow`.
- Phần tiếp: US-P7-04 chưa có handoff nên chưa chấm.

## 4. Cập nhật 2026-11-01 (HEAD `61c35b39`, không có vòng mới của story) · Kết luận giữ: **FAIL**
- **AC9 (LCP) vẫn FAIL** theo proposals #10: dev tối ưu 6.146 → 4.723 ms, chưa đạt 2.500 ms; PM ghi chờ chủ dự án chọn phương án ngày 01/11. QC không đo lại bằng LHCI (URL `/assignments/{id}` không có trong ma trận, theo #10).
- **BUG-P7-03-4 (Trung bình, AC4 "nhân viên xem được lịch sử"):** SRS mục 6 hàng 15 `GET …/submissions/{id}` (A, T — "gồm các phiên bản") **chưa có**: `GET /courses/{id}/submissions/{sid}` trả `404` cho mọi vai (QC phát hiện qua ma trận `p7-rbac.sh`; OpenAPI không có thao tác này). Hiện nhân viên chỉ thấy bản mới nhất ở hàng chờ và số bản ở hộp thoại gán (`…/versions?student_id=`). Dev thêm route hoặc BA / PM bỏ hàng 15 khỏi SRS bằng proposal.
- BUG-P7-03-1, -2, -3 giữ nguyên (LCP; câu `SUBMISSION_LOCKED` khác SRS; tên test ≠ US).
