# QC report — US-P7-09 (công bố hàng loạt, sổ điểm, sinh viên thấy kết quả) · Kết luận vòng 1: **FAIL** (AC1 và AC6: chưa có giao diện); **vòng 2 (`eaf827a8`): PASS** — xem mục 4
Ngày 2026-11-01. Commit kiểm: `sprint/9-p7` @ `c487e4fa` (máy chủ `578da354`). Bộ TC: `tc-US-P7-09.md`. Handoff: `dev-US-P7-09.md` (chỉ mục "Máy chủ"). CI `c487e4fa`: **success**.

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh`, `-race -tags integration`, mọi tên của dòng `Kiểm:`: assignment 22, grade 14, integration 1, today 4 | PASS 41/41 (lần đầu `TestMeGradesShowsAssignmentComponent` đỏ một lần; chạy lại 10 lần liên tiếp — 6 lần `-count=6` + 3 lượt runner + 1 lẻ — đều xanh; không bắt được thông điệp lỗi, xem BUG-P7-09-2). `TestTAReviewsCannotPublish` (nợ của US-P7-08) có ở đây và xanh |
| Hộp đen trên stack dựng lại từ `c487e4fa` (`scripts/p7-api-09.py`) | PASS 0 lỗi. Công bố bài DRAFT → 422 `NOT_REVIEWED` kèm ids; **all-or-nothing** (A REVIEWED + B DRAFT → 422, không bài nào PUBLISHED); TA / STUDENT / ADMIN công bố → 403, STUDENT / ADMIN xem `stats` → 403; TEACHER công bố 2 bài → 200 `published 2`; cùng `Idempotency-Key` → cùng kết quả, không thông báo thêm; công bố lại → 409 `ALREADY_PUBLISHED`; không ghi `grade_entries` cho item bài tập; độ lệch (`deviation`, `teacher_changed_criteria`) lưu lúc công bố; **mỗi người một chuông** `GRADE_PUBLISHED`, tổng 2; **đúng hai thư** (`Điểm bài tập đã được công bố`) tới `sv.gioi@`, `sv.kha@` (Mailpit), thư B ghi "4,00/10,00" + từng tiêu chí, không có canary nhận xét của A, cờ, `ai_`, token; `mine.result` của B có tổng, bốn tiêu chí, nhãn "đã được giảng viên duyệt", không khoá nháp / độ lệch; A thấy nhận xét của chính mình, không thấy của B; `/me/grades`, danh sách, chi tiết không lộ canary; `stats` `published 2`, tỷ lệ là chuỗi decimal; sửa sau công bố → 409 `RESULT_PUBLISHED`; chấm lại → 409 `ALREADY_REVIEWED`; "Hôm nay": TEACHER có `PUBLISH_PENDING` (TA không), mất sau khi công bố hết; B có `ASSIGNMENT_GRADED` (ở `recommended`); log gateway + worker không có canary |
| Playwright `assignment-lifecycle.spec.ts -g 'publish bulk confirm'` và `-g 'student sees published result'` | **không có test** → lỗi "No tests found" |
| Đột biến (`/tmp/qc-mut9`, đã xoá) U1 cho công bố cả bài chưa REVIEWED · U2 gửi thư cả tới email chưa xác minh | cả 2 **đỏ**: `TestPublishAllOrNothingExplicit`; `TestFeedbackMailToVerifiedEmail` |
| `go vet -tags testroutes ./...`; `go test -race -tags testroutes` `contract jobs cmd/... store db today` | xanh (OpenAPI 271 thao tác) |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 công bố hàng loạt — chỉ TEACHER, chỉ REVIEWED, một giao dịch, hậu quả bằng số | **FAIL** | Máy chủ PASS (hộp đen, U1, `TestPublishBulkOnlyReviewed`, `…AllOrNothingExplicit`, `…ScopeSkips`, `…Idempotent`, `…OnlyFromHandler`, `…OneTx`, `…ConcurrentOnce`). **Chưa có giao diện** `Công bố` (hộp thoại xác nhận nêu số bài / số thư) và test `publish bulk confirm` |
| AC2 vào sổ điểm P6 — đọc thẳng điểm đã công bố, không chép sang `grade_entries` | PASS | `TestAssignmentItemReadsPublishedDirect`, `TestPublishWritesNoGradeEntries`, `TestAssignmentItemBatchNoNPlusOne`, `TestAssignmentItemEqualsStudentView`; hộp đen: không có `grade_entries` bài tập. Hộp đen liên kết item `ASSIGNMENT` ở lược đồ điểm chưa dựng |
| AC3 sổ đã chốt — công bố không chặn, bản chốt không đổi, hiện lệch | PASS ở mức Go | hộp đen: công bố không bị chặn bởi sổ; kiểm chốt ở `grade` test |
| AC4 thư + chuông đúng người, đúng nội dung | PASS | hộp đen (Mailpit); U2; `TestFeedbackMail*`, `TestPublishNotifiesOnlyOwner`, `TestPublishOutboxReplayNoDuplicate` |
| AC5 độ lệch AI ↔ giảng viên lưu cho M13-A | PASS | cột có; `stats`; `TestDeviationStoredAtPublish`, `TestDeviationDemoStudentB` |
| AC6 sinh viên thấy điểm đã công bố | **FAIL** | API `mine.result` PASS (hộp đen). **Chưa có giao diện** phần kết quả ở `/assignments/[id]` và `/me`, và test `student sees published result` |
| AC7 sinh viên không bao giờ thấy nháp | PASS | `TestStudentNeverSeesDraftGrade`; hộp đen quét `mine`, danh sách, chi tiết, `me/grades`, thư, log |
| AC8 công bố dở, bài đổi giữa lúc công bố, Redis / SMTP chết | PASS ở mức Go | `TestPublishVersionConflict`, `…OutboxReplay…`; QC chưa dừng SMTP / Redis trên stack |
| AC9 phân quyền | PASS | hộp đen; `TestPublishRBAC`, `TestStatsRBAC`, `TestResultIDOR` |
| AC10 nguồn việc "Hôm nay" `PUBLISH_PENDING` (TEACHER), `ASSIGNMENT_GRADED` (STUDENT) | PASS | hộp đen; `TestProvider*` (4 test today) |

## 3. Lỗi / ghi nhận
- **BUG-P7-09-1 (Cao, AC1 + AC6):** chưa có giao diện công bố và giao diện kết quả của sinh viên; `grep` `Công bố` ở `features/grading` và `result` ở `features/assignments` rỗng; hai test Playwright theo `Kiểm:` chưa có. Dev giao frontend rồi QC chấm lại vòng 2 (máy chủ không cần kiểm lại ngoài hồi quy).
- **BUG-P7-09-2 (Thấp):** `TestMeGradesShowsAssignmentComponent` đỏ một lần khi chạy cùng 13 test khác trong một lượt, rồi xanh 10/10 lần sau; có thể phụ thuộc thời điểm / thứ tự. Nếu tái diễn, dev xem lại dữ liệu chung giữa các test.
- Ghi nhận: thân `POST …/grading-results/publish` có thêm `versions` (tuỳ chọn) ngoài SRS — dev đã ghi và có trong OpenAPI; `TestMineDTOAllowlist` cập nhật cho `result`.
- Đóng nợ: `TestTAReviewsCannotPublish` của US-P7-08 AC8 nay nằm ở đây, xanh; hộp đen TA công bố → 403.
- Chưa kiểm: liên kết item `ASSIGNMENT` ở `PUT grade-scheme` (`ASSIGNMENT_ALREADY_LINKED`, `ASSIGNMENT_SCALE_MISMATCH`) bằng hộp đen — chỉ Go; làm cùng US-S9-SEED.

## 4. Vòng 2 — `eaf827a8` (frontend) · Kết luận: **PASS**
CI `eaf827a8`: **success**. Bộ TC: `tc-US-P7-09.md` (không đổi).

| Lệnh | Kết quả |
| --- | --- |
| Playwright desktop `publish bulk confirm`, `publish bulk conflict`, `student sees published result`, `student grading shows no score` (+ `student page states`, `assignment a11y`, `grading a11y`) | 7/7 xanh; axe 8 lượt, 0 lỗi |
| Hồi quy `gradebook.spec.ts me.spec.ts today.spec.ts panels.spec.ts a11y.spec.ts` (desktop) | 141 passed, 3 skipped (đúng như dev) |
| `ui-antipatterns.sh`; grep `mock/assess` | rc 0; rỗng |
| **Đầu-cuối trên stack thật dựng lại từ `eaf827a8`**: 2 sinh viên nộp → nhập điểm tay → duyệt (API), rồi TEACHER công bố **bằng giao diện** | `/grading` → lọc `Đã duyệt` → `Chọn tất cả đã duyệt` → `Công bố`: hộp thoại "Công bố 2 bài · 2 sinh viên nhận thư · không tính vào sổ điểm" (`Để sau` / `Công bố`) → 2 kết quả `PUBLISHED`, 2 chuông `GRADE_PUBLISHED`, đúng 2 thư (Mailpit) tới `sv.gioi@`, `sv.kha@`. Sinh viên B mở `/assignments/{id}` 1440 và 375: khối "Kết quả" **4,00 / 10,00**, bốn tiêu chí `1,00 / 2,50`, nhận xét, nhãn "đã được giảng viên duyệt"; không có chữ nháp / AI / cờ; `scrollWidth = innerWidth` ở cả hai bề rộng. Ảnh: `shots/p709-queue-reviewed-1440.png`, `p709-publish-confirm-1440.png`, `p709-queue-after-publish-1440.png`, `p709-student-result-1440.png`, `-375.png` |

| AC | Vòng 2 |
| --- | --- |
| AC1 công bố hàng loạt (giao diện) | PASS — hộp thoại nêu số bài và số thư; công bố qua giao diện đúng hậu quả ở DB / chuông / thư; TA không thấy nút (`grading ta cannot see publish`, vòng US-P7-08) |
| AC6 sinh viên thấy điểm đã công bố | PASS — thấy điểm + tiêu chí + nhận xét; chưa công bố chỉ thấy "Đang chấm" (`student grading shows no score`) |
| AC2, AC3, AC4, AC5, AC7–AC10 | giữ PASS (máy chủ không đổi từ vòng 1) |

Ghi nhận (thấp): sau công bố `/assignments/{id}` vẫn hiện vùng thả "Chọn tệp / Kéo thả" cùng nút `Nộp lại` bị mờ kèm "Đã có điểm" — máy chủ khoá bằng `409 SUBMISSION_LOCKED`; nên ẩn vùng thả khi đã công bố. Liên kết item `ASSIGNMENT` ở công thức (`/gradebook/scheme`) chỉ kiểm bằng Go + Playwright (dev: `TestAssignmentItemLinkValidation`, spec gradebook) — chưa dựng trên stack thật (công thức lớp 1 đã `CONFIRMED`, sửa cần bản nháp mới) → làm cùng US-S9-SEED.
