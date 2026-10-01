# Sprint 2 — Góp ý của đội

PM quyết (`docs/team/PM.md` §5). Spec đã duyệt chỉ đổi qua góp ý ở đây.

| # | Ai | Vấn đề | Đề xuất | Lý do + bằng chứng | Ảnh hưởng nếu không đổi | Quyết định PM | Ngày |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | PM (duyệt spec v1) | `QUESTIONS.md` Q4: route thử `/api/v1/_test/…` nằm trong binary production, chỉ khoá bằng `APP_ENV=test` | Khoá bằng **build tag `testroutes`**: file đăng ký route thử có `//go:build testroutes`; Dockerfile có target riêng `gateway-test` (`go build -tags testroutes`); compose dùng target đó qua override `docker-compose.test.yml` cho QC. Binary/image mặc định không chứa route thử. AC18 của US-PG-03 đổi thành: image mặc định → mọi `/api/v1/_test/*` 404; `go tool nm` của binary mặc định không có symbol của gói route thử | Cấu hình nhầm một biến env không được mở endpoint ghi dữ liệu ở production (nguyên tắc an toàn mặc định); chi phí: thêm một target Dockerfile | Một biến env sai ở PR = lỗ hổng | **ACCEPTED** (PM tự chốt, chủ dự án giao toàn quyền) | 01/10 |
