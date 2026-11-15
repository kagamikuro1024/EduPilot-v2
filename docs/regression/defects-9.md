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

