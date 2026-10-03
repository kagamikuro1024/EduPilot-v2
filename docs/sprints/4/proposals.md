# Sprint 4 — Góp ý của đội

PM quyết (`docs/team/PM.md` §5). Spec đã duyệt chỉ đổi qua góp ý ở đây.

| # | Ai | Vấn đề | Đề xuất | Lý do + bằng chứng | Ảnh hưởng nếu không đổi | Quyết định PM | Ngày |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Dev (US-P2-01) | AC12 yêu cầu luật `depguard` "gói khác không import `store` truy vấn bảng token/phiên". depguard chỉ chặn theo gói import; `internal/store` là gói chung mọi module phải import (sqlc sinh một gói) nên không cấm được theo bảng. | Đổi cách kiểm AC12(c) thành `TestOnlyAuthPackageTouchesTokenTables` (quét mã nguồn ngoài `auth`/`store`, 0 chỗ chạm `auth_tokens\|auth_sessions\|AuthToken\|AuthSession`). Đã làm vậy. | `golangci-lint` 0 issues; test PASS. Chia `store` thành gói con theo bảng sẽ phá cấu hình sqlc (`package: store`, một `out`). | Không ảnh hưởng nếu chấp nhận; nếu muốn depguard thật thì phải tách sqlc thành nhiều gói. | | 2026-10-03 |
| 2 | Dev (US-P2-01) | AC6 ghi "dừng / bật container Mailpit"; container Mailpit dùng chung giữa các gói `go test` (testutil giữ container tên cố định) nên dừng nó làm hỏng test khác. AC10 nêu `smtpmock` (thư viện ngoài bảng ARCHITECTURE). | Dùng `testutil.FakeSMTP` tự viết (~100 dòng, dừng/bật đúng cổng, trả 550/451) cho các ca lỗi; Mailpit thật cho ca gửi thành công. Đã làm vậy. | `TestRetryThenDead`, `TestRecoverMidRetry`, `TestPermanentFailureNoRetry`, `TestTemporaryFailureRetries` PASS; không thêm thư viện. | Không. | | 2026-10-03 |
