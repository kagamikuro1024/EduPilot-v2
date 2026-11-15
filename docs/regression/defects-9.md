### DEF-R9-01 — Tên test ở dòng `Kiểm:` của US không có trong repo (hoặc ở gói khác)
- Mức: Thấp
- Sprint gốc: 9 · TC gốc: TC-P7-01-G06, TC-P7-02-G10, TC-P7-03-G03, TC-P7-03-G07, TC-P7-06-G11, TC-P7-08-G08 · Hồi quy do sprint sau?: KHÔNG (có sẵn — BUG-P7-03-3 / handoff dev-US-P7-0{1,2,6}: `TestAssignmentsRBACMatrix`, `TestSubmitIdempotent`, `TestSubmitClosed`, `TestGradingResultsRBAC`, `TestNoDraftCanaryToStudent` chưa bao giờ có)
- Môi trường: ep-r/be0c0d96, `go test -tags integration[,testroutes] -run <tên> -v <gói>`
- Bước tái hiện: chạy lệnh ở hàng `-G` tương ứng; `logs/g-go.jsonl`, `logs/g-go-fix.jsonl`
- Kỳ vọng: test được nêu tên tồn tại và PASS · Thực tế: `no tests to run` / không có hàm
- Bằng chứng: `docs/regression/evidence-9/logs/TC_P7_*_G*.log`
- Chạy lại lần 2: ổn định


### DEF-R9-02 — `go test ./internal/contract/...` đỏ ở HEAD (TC-P7-01-G06, một vế)
- Mức: Trung bình (chưa xác minh nguyên nhân)
- Sprint gốc: 9 · TC gốc: TC-P7-01-G06 · Hồi quy do sprint sau?: KHÔNG RÕ (chưa chạy lại với tập thẻ build đúng của gate-p6/p7; cả `-tags testroutes` lẫn không thẻ đều rc=1)
- Môi trường: ep-r, be0c0d96, máy dev (testcontainers/Colima)
- Bước tái hiện: `cd backend-go && go test -count=1 -tags testroutes ./internal/contract/...` → rc=1, `TestSpec_EveryDocumentedStatusExercised` báo các route `/api/v1/_test/*` (whoami, events, items) "khai báo status … nhưng không test nào sinh ra"; không thẻ: `TestStudentNeverSeesConfidence` "0 lời gọi chat"
- Kỳ vọng: xanh (mọi gate cũ chạy `-tags testroutes`) · Thực tế: đỏ
- Bằng chứng: `docs/regression/evidence-9/logs/contract-testroutes.log` (6.000 ký tự cuối), `TC_P7_01_G06.log`
- Chạy lại lần 2: ổn định (2 lần)
