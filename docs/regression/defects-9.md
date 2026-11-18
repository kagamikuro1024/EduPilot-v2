### DEF-R9-01 — Tên test ở dòng `Kiểm:` của US không có trong repo (hoặc ở gói khác)
- Mức: Thấp
- Sprint gốc: 9 · TC gốc: TC-P7-01-G06, TC-P7-02-G10, TC-P7-03-G03, TC-P7-03-G07, TC-P7-06-G11, TC-P7-08-G08 · Hồi quy do sprint sau?: KHÔNG (có sẵn — BUG-P7-03-3 / handoff dev-US-P7-0{1,2,6}: `TestAssignmentsRBACMatrix`, `TestSubmitIdempotent`, `TestSubmitClosed`, `TestGradingResultsRBAC`, `TestNoDraftCanaryToStudent` chưa bao giờ có)
- Môi trường: ep-r/be0c0d96, `go test -tags integration[,testroutes] -run <tên> -v <gói>`
- Bước tái hiện: chạy lệnh ở hàng `-G` tương ứng; `logs/g-go.jsonl`, `logs/g-go-fix.jsonl`
- Kỳ vọng: test được nêu tên tồn tại và PASS · Thực tế: `no tests to run` / không có hàm
- Bằng chứng: `docs/regression/evidence-9/logs/TC_P7_*_G*.log`
- Chạy lại lần 2: ổn định
- **ĐÓNG theo P-11-42 (2026-11-16):** 5/6 hàng đổi tên test trỏ tới theo ánh xạ dev2, chạy lại PASS (`evidence-9/logs/p11-42.log`); TC-P7-03-G03 chờ `TestSubmitIdempotent` (dev2 đang viết).

### DEF-R9-02 — ĐÃ LOẠI: `go test ./internal/contract/...` đỏ lúc tải cao
- Chạy lại `go test -count=1 -tags testroutes ./internal/contract/...` lúc máy rảnh: ok (25 PASS, rc=0, logs/contract-rerun.log). Lần đỏ trước do timeout/504 khi tải ~20 → môi trường, không phải lỗi.


## Smoke 9 — phiên Smoke9Runner (dc67bd93, ep-qc, 2026-11-18)
| ID | Mức | TC | Mô tả / tái hiện | Bằng chứng |
|---|---|---|---|---|
| DEF-R9S-01 | Thấp | TC-P7-09-M6 | Đột biến bỏ `AND status='REVIEWED'` ở UPDATE công bố (store/review.sql.go `PublishMark`) sống: toàn bộ test Publish* xanh (tầng Go đã chặn bằng khoá + kiểm trạng thái, SQL chỉ là lớp phòng thủ thứ hai) — thiếu test cho lớp SQL | logs/mut-TC-P7-09-M6.log, mut-TC-P7-09-M6b.log |
| DEF-R9S-02 | Thấp | TC-P7-06-15, TC-P7-07-06 | Hàng chờ `/grading` bài FAILED chỉ hiện «Chấm lỗi», không nêu nguyên nhân bằng chữ; chữ «Hết ngân sách AI hôm nay — chấm lại ngày mai hoặc chấm tay» không có trong frontend/mã. Tái hiện: lớp có ngân sách ngày hết → grade-all → bài FAILED LLM_BUDGET_EXHAUSTED; API trả flags [] + error_code; QueueTab chỉ dựng chữ từ FLAG_TEXT. (Chưa chụp UI — kiểm bằng API + đọc mã) | logs/q9_06b.log, logs/q9_07a.log |
| DEF-R9S-03 | Thấp | TC-P7-07-M4 | Đột biến XADD trước COMMIT (Signal gọi trong enqueue trước tx.Commit) sống: nhóm test TestGradeAll*/GradeOne/Retry/DeadLetter/Queue xanh; TestGradeAllEnqueuesOnlyUngraded không có queue/Redis trong rig nên XLen=0 vô nghĩa; Tick tự nạp lại che lỗi | logs/mut-TC-P7-07-M4.log |

Đóng (xác nhận 2026-11-19 @6abb09af): DEF-R9S-01 (test SQL mới đỏ khi đột biến), DEF-R9S-02 (UI nêu nguyên nhân, 1440+390 px).

| DEF-R9S-04 | Thấp | TC-P7-07-01 | `POST …/assignments/{aid}/grade-all` khi Redis dừng trả 503 SERVICE_UNAVAILABLE và không tạo dòng QUEUED (TC/US AC1: vẫn 202, dòng QUEUED ghi trong giao dịch không cần Redis). Nghi middleware Idempotency-Key cần Redis. Tái hiện: `docker stop ep-qc-redis-1`; grade-all với Idempotency-Key | logs/q9_07c.log |
| DEF-R9S-05 | Thấp | TC-P7-08-04 | Màn duyệt `/grading/{id}` 1440 px: cột trái 576 px / cột phải 576 px (49 % / 49 %), TC yêu cầu 52–58 % / 42–48 % | shots/p708-review-1440.png, logs/ui-dump08.json |
| DEF-R9S-06 | Thấp | TC-P7-07-13 | Không có `/metrics`; `grading_jobs_total`, `grading_dead_total`, `grading_duration_seconds`, `grading_queue_depth` chỉ nằm trong bộ nhớ (`Metrics.Snapshot` không ai gọi), `/observability` không đọc dead-letter (US-P7-07 AC10) | logs/q9_07a2.log |
