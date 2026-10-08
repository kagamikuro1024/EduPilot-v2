# DEV handoff — US-PE-05 (làm bài trắc nghiệm: bắt đầu / làm tiếp, xáo theo lượt, tự lưu, một nơi ghi, nộp, tự nộp hết giờ, giao diện `/exams/[id]/take`)
Nhánh `sprint/5-pe`, spec `FEAT-weekly-exam` v1.6. Không có migration mới (bảng `exam_attempts`, `exam_answers`, `exam_events` ở `00006`). `openapi.yaml`: 96 → **102** thao tác (+ #29 bắt đầu, #30 `attempts/mine`, #31 lưu đáp án, #39 `takeover`, #40 nộp, #41 xem kết quả; #32–38 (chạy thử / nộp lời giải code) thuộc PE-06, #42–43 (`/me/exam-lock`) thuộc PE-07).

## Làm gì
- **Backend `internal/exam`**: `attempts.go` (`StartAttempt` 201 mới / 200 làm tiếp, `MyAttempt` 3 dạng, `Takeover`, `SaveAnswers`, `SubmitAttempt`, `finish`, `gradeMCQ`, `AutoSubmitDue`, `AttemptResult`), `dto_student.go` (`AttemptSeed` = SHA-256 `attempt_id` → PCG, Fisher–Yates; `BuildStudentView(src, shuffleQ, shuffleO, seed, saved)`), `tick.go` (bước 3 `AutoSubmitDue`). SQL ở `internal/store/queries/exam_attempt.sql` (`AnswersUpsert` bằng `jsonb_to_recordset`). Cấu hình `EXAM_GRACE_SECONDS` (10) · `EXAM_SAVE_RATE_PER_MIN` (240) · `EXAM_TAB_STALE` (20 s) — `.env.example` + `docker-compose.local.yml`.
- **`internal/httpapi/examhttp/attempts.go`**: `MountAttempts`; header `X-Exam-Tab`; thân lưu ≤ 64 KiB; `exempt.go` miễn mã 429 cho `PUT …/answers`. Worker: topic `exam.attempt_started` / `exam.attempt_submitted` → xoá cache "Hôm nay" (`today.Topics()` 11 → 13).
- **Frontend**: `shared/lib/examClock.ts` (độ lệch = `server_time` − trung điểm lúc gửi / nhận, mỗi lần đổi ≤ 1 s), `shared/lib/saveQueue.ts` (hàng đợi lưu: giữ ở `localStorage`, gộp 2 s, lùi 1-2-4-8-15 s, lỗi cố định 4xx dừng), `shared/domain/ExamTimer`, `features/exam/take/{ExamTake,TakeRunning,useWriter,useAnswers,takeApi}`, route `/exams/[id]/take`. Bài code chỉ hiện màn hình hẹp (giao diện code ở PE-06).
- **Thiết kế**: MCQ chấm NGAY lúc nộp (`GRADED`, `auto_score` + `breakdown`); bài có câu code → `GRADING`, chốt ở PE-06. Sau nộp `attempts/mine` chỉ trả tóm tắt (không đề). Giới hạn lưu fail-open khi Redis lỗi. `ErrInvalidOption` khi chấm = bỏ trống.

## AC (kết quả thật)
| AC | Kết quả |
| --- | --- |
| 1–2 Bắt đầu / làm tiếp / một lượt | `TestStartAttempt*`, `TestResumeSameAttempt`, `TestClockNotResetOnReload`, `TestOneAttemptRaceStart` (20 yêu cầu song song → 1 lượt), `TestStartAfterSubmit409` PASS |
| 4 Không lộ đáp án | `TestAttemptPayloadWhitelist`, `TestAttemptPayloadNoAnswerCanary` (canary trong đáp án / giải thích / test ẩn), `testdata/student_dto_allowlist.json` (19 DTO) PASS |
| 5 Xáo | `TestShuffle*` (6 ca: ổn định theo `attempt_id`, khác giữa sinh viên, tắt xáo giữ nguyên, đáp án ghim, lưu theo `option_id`) PASS |
| 6 Lưu | `TestSave*` (Diff / Validation / RateLimit (Redis thật) / AfterDeadlineGrace / Other404), `TestGraceAcceptsLastSave` PASS |
| 7 Tự nộp hết giờ | `TestAutoSubmit*` (4 ca) PASS |
| 8 Nộp | `TestSubmit*` (4 ca: idempotent, `answered/total`, chấm MCQ, lượt đã nộp) PASS |
| 9 Một nơi ghi | `TestSingleWriterTab`, `TestTakeoverFlipsWriter`, `TestStaleTabCannotOverwrite`, `TestWriterSeenThrottle`, `TestTakeoverReloadMeta` PASS; e2e `take: two tabs`, `take: duplicate tab`, `take: reload writer` PASS |
| 13 | `TestStudentGuardStates` (chưa mở / đã đóng / PENDING / REMOVED), `TestRemovedMidExam403`, `TestResultBeforePublish409`, `TestOtherStudentAttempt404`, `TestAfterSubmitSummaryOnly`; hợp đồng `examAttemptScenarios` PASS |
| 14 | `openapi.yaml` 102 thao tác; `go test ./internal/contract/...` (`-tags testroutes`) PASS |
| 3 Đồng hồ | `exam-clock.spec.ts` (8 ca: lệch +5 phút / −5 phút, mỗi lần đổi ≤ 1 s, định dạng, mốc 5 / 1 phút) + e2e `take: clock skew`, `take: timeout autosubmit` PASS |
| 10 Hàng đợi lưu | `exam-save-queue.spec.ts` (7 ca: gộp, lùi, lỗi cố định dừng, khôi phục sau tải lại, online / offline) + e2e `take: offline mid exam` PASS |
| 11–12 Giao diện | e2e `take: submit confirm and after` (hộp xác nhận nêu số câu, "Làm tiếp", nộp lại cùng `Idempotency-Key`, màn sau nộp không điểm / đúng sai), `take: mcq 375` (một câu mỗi lần, phím `A–D` / `1–4` / `←→`, vùng chạm = `[]`, `AUDIT_SRC` = `{ox:0,cut:[],ell:[]}`) PASS |

## Lệnh tự kiểm
- `cd backend-go && make lint sqlc-check` → 0 issues ×3, `sqlc diff` sạch. `make test`: lượt đầy đủ có 1 ca đỏ `TestJobs_Lifecycle` (gói `internal/jobs`, không đụng PE-05; chạy riêng `-count=3` xanh → nhiễu khi tải nặng, thêm vào nhóm ca chập chờn cùng `TestTodayQueryBudget` / `TestStaffTodayCountAndCap` / mail `TestRetryThenDead`); phần integration mail / auth xanh (288 ca); `go test -tags integration ./internal/exam -run TestTickLeaderLock` xanh.
- Frontend: `pnpm lint` sạch, `tsc --noEmit` sạch, `ui-antipatterns.sh` 0 ✗, `lint-selftest.sh` 7/7 + 19/19; `playwright test --grep-invert "@real|visual" --workers=2` → **403 pass, 0 đỏ** (103 skip theo dự án).

## Nợ / chuyển giao
- **Khoá chat `ep:exam_lock` lúc bắt đầu CHƯA làm** (SRS AC1 nói bắt đầu đặt khoá; thuộc PE-07 AC1 cùng `GET /me/exam-lock` #43) — PE-07 nối vào `StartAttempt` / `finish`.
- Dòng "Giảng viên đã gia hạn…" cho sinh viên đang làm chưa hiện (đồng hồ dài ra đúng ở lần lưu kế).
- Kiểm thử `X-Exam-Tab` / người ghi chạy ở gói thường (không tách `-tags integration`); k6 `autosave` p95 → PE-09.
- Outbox `exam.attempt_*` hiện chỉ xoá cache "Hôm nay".
- Chưa chạy: `@real`, ảnh chụp thị giác, curl / psql thủ công trên stack thật, GitHub CI (hết hạn mức thanh toán).
- Sửa kèm: QC US-PE-03 B1 (NUL → 500) đã xử lý ở tầng chung `httpx.DecodeJSON` (commit `203334e`), không thuộc phạm vi story này.
