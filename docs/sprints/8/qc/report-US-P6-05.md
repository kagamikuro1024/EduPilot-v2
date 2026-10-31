# QC report — US-P6-05 (trạng thái công thức, nhắc hằng tuần, "Hôm nay" `GRADE_SCHEME_UNCONFIRMED`, AI nói chưa có công thức) · Kết luận: **FAIL** (vòng 1: AC2 và AC3 chưa kiểm được — màn `/gradebook` và route `finalize` chưa có; AC1, AC4–AC9 PASS; đề nghị PM chấp nhận proposals #18 như #8 → PASS có điều kiện)
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `f07c2af6` (mã story ở `05633308`). Bộ TC: `tc-US-P6-05.md` (chữ TC chỉ đổi 1 dòng theo #11). Handoff: `dev-US-P6-05.md`. CI: run `41188894701` (`f07c2af6`) **đang chạy lúc QC chốt**; run trước `41188589228` bị huỷ do commit mới (Go static / auth / course / Judge xanh). QC chạy lại toàn bộ bằng chứng trên máy QC (Go + Playwright, mục 1).

## 1. Cổng nghiệm thu đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p6-go-tests.sh ./internal/grade "TestStatusStates\|TestStatusPendingDraft\|TestStatusBlocksFinalizeWithoutScheme\|TestReminder*(5)\|TestSourceDocumentDeletedKeepsConfirmed\|TestRecomputeOnDocumentChanged" -race` | PASS 10/10 (không SKIP, không DATA RACE) |
| `go test ./internal/grading -run TestStatusRBAC` | PASS (route nằm ở gói `grading`, không phải `grade`; 11/11 test AC1/3/8/9 tính cả hai gói) |
| `p6-go-tests.sh ./internal/today "TestProviderGradeSchemeUnconfirmed\|TestProviderClearedOnConfirm\|TestProviderGradeSkipsArchived\|TestTodayQueryBudget\|TestStaffTodayCountAndCap\|TestProviderOverdueAfter48h"` | PASS 6/6 (gồm ba test cũ đã sửa theo #13) |
| `p6-go-tests.sh ./internal/agent "TestGradeFormulaNoScheme\|TestPersonalGradeNoScheme\|TestWhatIfNoScheme"`; `./internal/escalation "TestAutoNoFormulaTicket"` | PASS 3/3; PASS 1/1 |
| `go test -count=1 -race -tags testroutes ./internal/contract` (số thao tác 228, #12) | PASS (21) |
| `go test -count=1 -tags testroutes ./internal/integration -run 'TestNoUnscopedChunkQuery\|TestAllCourseRoutesGuarded'`; `./internal/rag -run TestKnowledgeDocExcludedEverywhere`; `./internal/today` (cả gói) | PASS (3, 1, ok 35 s) — ba test cũ đỏ ở US-P6-03 nay xanh |
| `golangci-lint` các gói đụng tới + `cmd/...`; `go vet ./...`; `sqlc diff`; `go test ./db/...` | PASS cả bốn; `grep -ci 'ALTER TABLE' 00016_gradebook.sql` = 0; `unclear_count` nằm trong `CREATE` (không `ALTER`, #16) |
| `grep 'float64' backend-go/internal/grade` (mã không-test) | rỗng |
| Playwright (máy QC, bản dựng `build:gate`) `attendance class-join private-chat calendar lcp settings-llm shell a11y` `--workers=2 --retries=0` | PASS: **174 passed**, 130 skipped (ca một dự án), 0 failed, 0 flaky; `ui-antipatterns.sh` rc=0; `pnpm lint` sạch |
| Đột biến (bản sao `/tmp/qc-mut`, đã xoá) | P1 thang `DRAFT_UNCLEAR`→luôn "chưa rõ": **đỏ** (`TestStatusStates`, `TestRecomputeOnDocumentChanged`); P3 TA bị loại khỏi Provider: **đỏ** (`TestProviderGradeSchemeUnconfirmed`); P4 bỏ `finalize_blocked_reason`: **đỏ** (`TestStatusBlocksFinalizeWithoutScheme`); P2 bỏ khoá khử trùng thông báo: **XANH** — xem BUG-P6-05-1 |
| QC probe `scripts/p6_qc_probe_remind_test.go.txt` (chạy tạm rồi xoá): gọi `RemindUnconfirmed` hai lần cùng tuần ISO | PASS: đúng 1 thông báo — khử trùng ở mức thông báo **có hiệu lực** (TC-P6-05-45) |

## 2. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01–08 trạng thái, `pending_draft`, `finalize_blocked_reason`, RBAC `status` | PASS ở mức test + contract (200 Teacher/TA, 401, 403 Student/Admin, id lạ) | `TestStatusStates` (5 trạng thái), `…PendingDraft`, `…BlocksFinalizeWithoutScheme`, `TestStatusRBAC`; P1, P4 làm đỏ. Qua stack đầy đủ (`j $Q2/grade-scheme/status` → `NO_POLICY`): chờ US-S8-SEED |
| 09–14 banner `/gradebook` | **KHÔNG KIỂM ĐƯỢC** (chưa có màn `/gradebook` thật) | handoff: làm ở US-P6-06 |
| 15–18 nút `Chốt điểm` khoá + tooltip; `POST finalize` lớp 2 → `409` | **KHÔNG KIỂM ĐƯỢC** (chưa có nút, chưa có route `finalize`) | handoff: làm ở US-P6-08; chỉ phần khoá `finalize_blocked_reason` ở API `status` có (TC-04) |
| 19, 21–22 banner `/attendance` | PASS | `attendance.spec.ts` "attendance notice" (3 ca) trong 174 passed; một `InlineNotice` "Công thức điểm chưa xác nhận" + `Xem công thức`, không chặn điểm danh; `CONFIRMED` không hiện |
| 20 điểm danh 30 SV ≤ 60 s khi có banner | PASS gián tiếp (ca điểm danh trong `attendance.spec.ts` vẫn xanh); bấm giờ tay chưa làm | |
| 23–27 "Hôm nay" `GRADE_SCHEME_UNCONFIRMED` (bậc 70, TEACHER + TA, không STUDENT/ADMIN, lý do theo `state`, biến mất khi xác nhận) | PASS | `TestProviderGradeSchemeUnconfirmed`, `…ClearedOnConfirm`, `…SkipsArchived`; P3 làm đỏ; thứ tự kind trong `TestProviderOverdueAfter48h` (`GRADE_SCHEME_UNCONFIRMED` giữa bậc 40–45 và `COURSE_SETUP`) khớp bảng xếp hạng; ≤ 5 s chỉ kiểm bằng invalidation (không đo giây trên stack) |
| 28 lớp lưu trữ không có mục | PASS | `TestProviderGradeSkipsArchived` |
| 29 test Provider | PASS | |
| 30–36 chuông hằng tuần: một lần/tuần, trước 08:00 chưa nhắc, tuần kế gửi tiếp, `CONFIRMED` không gửi, chạy bù, chỉ TEACHER | PASS | `TestReminderWeeklyOncePerWeek` (07:59 → không; 08:00 → có; tick sau không; tuần kế → 2), `…CatchUp`, `…StopsWhenConfirmed`, `…SkipsArchived`, `…RecipientsTeacherOnly` |
| 37–41 AI: câu mẫu khi chưa `CONFIRMED`, 0 lời gọi LLM, không bịa số, ticket `AUTO_NO_FORMULA` chỉ ở `GRADE_FORMULA` | PASS ở mức test | `TestGradeFormulaNoScheme`, `TestPersonalGradeNoScheme`, `TestWhatIfNoScheme`, `TestAutoNoFormulaTicket`; hội thoại thật trên stack: chờ US-S8-SEED / `gate-P6` |
| 42–43 quy chế bị xoá sau xác nhận | PASS | `TestSourceDocumentDeletedKeepsConfirmed` (vẫn `CONFIRMED`, `source_title` còn, `has_policy=false`) |
| 44 `status` lớp khác | PASS theo #11: guard dùng chung `403` | `TestStatusRBAC` ("GV lớp khác" → 403 `FORBIDDEN`) |
| 45 khoá Redis mất, khử trùng thông báo | PASS bằng probe QC | probe PASS; **test của dev không phủ** (P2 xanh) |
| 46 hai worker cùng chạy | PASS ở mức thiết kế (khoá leader `ep:lock:grade.remind` + `SET NX`); không chạy hai tiến trình thật | |

## 3. AC
| AC | Kết quả |
| --- | --- |
| AC1 trạng thái công thức | PASS |
| AC2 banner `/gradebook` (TEACHER, TA) | **KHÔNG KIỂM ĐƯỢC → FAIL** (chờ US-P6-06; proposals #18) |
| AC3 khoá `Chốt điểm` kèm lý do + `409` | **KHÔNG KIỂM ĐƯỢC → FAIL** (chờ US-P6-08; proposals #18). Phần dữ liệu `finalize_blocked_reason` ở `GET …/status` PASS |
| AC4 cảnh báo `/attendance` | PASS |
| AC5 Provider "Hôm nay" | PASS |
| AC6 chuông hằng tuần | PASS |
| AC7 AI nói thật + ticket | PASS (mức test) |
| AC8 quy chế bị xoá | PASS |
| AC9 phân quyền (status, chuông chỉ TEACHER) | PASS |

## 4. Lỗi
- **BUG-P6-05-1 (Thấp, khoảng trống bảo vệ):** bỏ `DedupeKey` của thông báo ở `RemindUnconfirmed` (đột biến P2) **không làm test nào đỏ**: `TestReminderWeeklyOncePerWeek` chỉ kiểm tầng `ReminderJob` (`SET NX` Redis), không kiểm "gọi `RemindUnconfirmed` hai lần cùng tuần → một thông báo". Hành vi hiện tại đúng (probe QC PASS). Đề nghị dev **thêm** một ca (không sửa test cũ): `RemindUnconfirmed` hai lần cùng tuần ISO → `count = 1` (nguyên văn probe ở `scripts/p6_qc_probe_remind_test.go.txt`). Không chặn story.
- **BUG-P6-05-2 (Quy trình, Thấp — đã tự khai):** handoff ghi `#13` "phần test CHƯA push, chờ TL" nhưng `staff_test.go` đã nằm trong commit `05633308` (`git log -- backend-go/internal/today/staff_test.go`). Nội dung đúng với #13 ACCEPTED có điều kiện (165 = 55+55+55; kind giữa `EMAIL_MISMATCH` và `COURSE_SETUP`; ngân sách 6 → 7 một lớp; ca 2 lớp có `DRAFT` ≤ 7 truy vấn theo TL-1 — ca đó có mặt). Ghi nhận handoff lệch thực tế; không cần hoàn tác.

## 5. Kiểm phản mẫu UI / luật mở rộng / phân quyền
- Luật 15 (không cache điểm/trạng thái): `GET …/status` `no-store` (test); luật 13 (N+1): `StatusBatch` một truy vấn theo lô, `TestTodayQueryBudget` ≤ 7 với 2 lớp có bản nháp (#16/TL-1).
- Luật chữ UI: banner `/attendance` ngắn, một dòng; không toast.
- Test cũ bị sửa: `internal/today/staff_test.go` (#13 ACCEPTED), `internal/contract/contract_test.go` 226 → 228 (#12 ACCEPTED), `internal/store/gradebook_schema_test.go` thêm cột `unclear_count` (test của chính US-P6-01), `e2e/class-join.spec.ts` + `e2e/support/session.ts` (#17 ACCEPTED, BUG-S8-01). Không nới assertion ngoài các số đó.
- `00016_gradebook.sql` sửa tại chỗ thêm cột `unclear_count` — luật 6 chỉ cấm sửa migration đã merge; chưa merge, #16 ACCEPTED.
- Không thư viện mới; không secret; không PII.

## Đề nghị
1. PM chấp nhận proposals #18 (chuyển nơi kiểm AC2 → US-P6-06, AC3 → US-P6-08) → US-P6-05 PASS; QC chạy TC-09…18 ở hai story đó.
2. Dev thêm ca bảo vệ khử trùng chuông (BUG-P6-05-1); sửa mô tả #13 trong handoff cho khớp thực tế.
3. QC chấm lại sau khi CI `41188894701` xong (nếu đỏ do ca ngoài story, ghi vào report).
