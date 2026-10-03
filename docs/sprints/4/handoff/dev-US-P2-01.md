# DEV handoff — US-P2-01 (`00003` + `00004`, `internal/mail`, token một lần)
Nhánh `sprint/4-p2` (xếp chồng trên `sprint/3-pu-p1`, đã merge `a363512`). Không có endpoint HTTP mới → `openapi.yaml`/contract không đổi.

## Làm gì
- `db/migrations/00003_course_foundation.sql` (dạng cuối theo `FEAT-course-foundation/SRS.md` 5: 7 enum, 8 bảng, trigger `set_updated_at`, `Down` đầy đủ) và `00004_auth_hardening.sql` (3 enum, 4 bảng) — **cùng một commit** (QUESTIONS Q1). `00001`/`00002` không bị sửa. `db/migrations_test.go`: version 2→4, `status` nêu 4 tệp, `down` xoá hết bảng của 00003/00004 (không nới lỏng gì).
- `internal/store/queries/{auth,mail}.sql` + sqlc: `InsertAuthToken`, `RevokeUnusedAuthTokens`, `GetAuthTokenByHash`, `InsertMailOutbox` (`ON CONFLICT (dedupe_key) … DO NOTHING`), `LockMailOutbox` (`FOR UPDATE`), `MarkMailSent`, `RecordMailFailure`, `MarkMailDead`.
- `internal/auth/tokens.go`: `Tokens.Issue(ctx, tx, user, kind, ttl, createdBy)` — 32 byte `crypto/rand` → 43 ký tự base64url, DB chỉ giữ `HashToken` (sha256 hex); thu hồi token cùng `(user,kind)` chưa dùng; chạy trong tx của người gọi.
- `internal/mail`: `Enqueue(ctx, tx, Message)` (cùng tx ghi `mail_outbox` + `outbox` topic `mail.send` `{mail_id}`; từ chối khoá payload `token|password|secret|link|url`; `dedupe_key` trùng → `queued=false`), `Render` (7 mẫu SRS 6.5: `text/template` + `html/template`, thiếu/rỗng biến → `ErrMissingVar`), `SMTP` sender (go-mail; 5xx = vĩnh viễn, còn lại tạm thời; lỗi chỉ là nhãn `smtp_permanent_550`/`smtp_temporary_451`/`smtp_unavailable`/`smtp_timeout`…, không chứa email/token), `Handler.Handle` (khoá dòng `FOR UPDATE`, SENT/DEAD → no-op; phát token **lúc gửi** trong cùng tx; gửi lỗi → rollback tx nên token chưa tới người nhận bị huỷ; lần thứ 4 hoặc lỗi vĩnh viễn → `DEAD`).
- `cmd/worker/registry.go`: đăng ký `mail.send`. Config: `APP_PUBLIC_URL`, `SMTP_HOST/PORT/USER/PASS/TLS`, `MAIL_FROM`, `MAIL_SEND_TIMEOUT`, `VERIFY/RESET/INVITE_TOKEN_TTL` (+ test `TestLoad_Mail`); `.env.example`, `docker-compose.local.yml` (worker nhận biến mail, `depends_on` mailpit).
- `testutil`: `Mailpit(t)` (container dùng chung, tên cố định) và `FakeSMTP` (550/451/dừng-bật cổng).
- `Makefile`: `lint` thêm `vet`/`golangci-lint` với `-tags integration`; `test` thêm `-tags integration ./internal/mail/...`.

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `TestAuthSchema` PASS (cột/kiểu/null/mặc định 4 bảng, 3 enum, 15 chỉ mục, FK). `TestMigrations_RoundTrip` PASS: version `4`, `up` lần hai no-op, `down` về 0 rồi `up` giống hệt. `git diff -- backend-go/db/migrations/0000[12]*` rỗng |
| 2 | `TestAuthConstraints` 23 ca (≥ 12): 23514 / 23505 / 22P02 (enum). Lưu ý: `char(64)` đệm khoảng trắng nên chuỗi ngắn bị CHECK từ chối (23514), không phải 22001 |
| 3 | `TestTokenIssueHashed`, `TestTokenIssueRevokesPrevious`, `TestTokenNeverLogged` PASS `-race` (quét mọi cột text/jsonb/char của toàn schema `public` + log của consumer, cả đường lỗi lẫn thành công: 0 lần) |
| 4 | `TestEnqueueRollback`, `TestEnqueueCommit` PASS |
| 5 | `TestSendViaMailpit` PASS (Mailpit testcontainers, REST): tiêu đề, `From`, người nhận, Text + HTML, liên kết `https://localhost/verify-email?token=<43>`, `sha256(token)` khớp `auth_tokens`, bản rõ không có ở `mail_outbox`/`outbox`, 0 `{{`/`<no value>`/`%!` |
| 6 | `TestRetryThenDead` (SMTP chết: `attempts=4`, `DEAD`, `outbox.dead_at`, `outbox.dispatch.dead` có 1 tin, `last_error=smtp_unavailable`, quét `@\|token=` = 0, token phát lúc gửi bị huỷ), `TestRecoverMidRetry` (bật lại SMTP sau lần thất bại đầu → đúng 1 thư) PASS |
| 7 | `TestRedeliverOnce` (4 goroutine giao lại sau khi SENT → Mailpit 1 thư), `TestDedupeKey` PASS |
| 8 | `TestTemplatesRender` (7 mẫu × {tiêu đề, chữ thuần, HTML}, golden `internal/mail/testdata/golden/*.txt`), `TestTemplatesEscape` (`<script>` → `&lt;script&gt;`, chữ thuần nguyên văn, tiêu đề không có tên), `TestTemplateMissingVar` (mọi biến bắt buộc × {thiếu, rỗng}) PASS |
| 9 | `TestPayloadHasNoSecrets` PASS (7 khoá bị từ chối; không dòng nào có khoá `token\|password\|link\|url`); `grep -rn mail_outbox internal --include=*.go \| grep -E 'handler\|http' \| grep -v _test` = 0 |
| 10 | `TestPermanentFailureNoRetry` (550 → `DEAD` ngay, `attempts=1`, outbox `dispatched`, mẫu lạ/thiếu biến cũng DEAD ngay), `TestTemporaryFailureRetries` (451 → QUEUED rồi SENT) PASS |
| 11 | `sqlc diff` rc=0; grep SQL thô trong `internal/auth`,`internal/mail` = 0 (`TestOnlyAuthPackageTouchesTokenTables`) |
| 12 | `TestOnlyAuthPackageTouchesTokenTables` PASS; `golangci-lint` 0 issues (mặc định, `testroutes`, `integration`) |

Cổng: `make -C backend-go lint test sqlc-check` xanh (toàn bộ gói ok; `-race`).

## Lệch spec / nợ
- **AC12 `depguard`:** depguard chặn theo *gói import*, không theo *bảng/tên truy vấn*; `internal/store` được mọi module dùng nên không cấm được. Thay bằng `TestOnlyAuthPackageTouchesTokenTables` (quét mã nguồn ngoài `auth`/`store`: `auth_tokens|auth_sessions|AuthToken|AuthSession` = 0). Ghi `proposals.md`.
- **AC6 "dừng/bật container Mailpit":** container Mailpit dùng chung giữa các gói test nên không dừng được; dùng `FakeSMTP` (dừng/bật đúng cổng) cho lỗi/hồi phục, Mailpit thật cho `TestSendViaMailpit`/`TestRedeliverOnce`. `smtpmock` thay bằng `FakeSMTP` tự viết (không thêm thư viện).
- Handler giữ khoá dòng qua một lần SMTP (≤ `MAIL_SEND_TIMEOUT`) để không gửi đôi; đủ cho T1. Nếu commit lỗi sau khi SMTP đã nhận thì thư có thể đi hai lần (at-least-once). `ponytail:` trong mã.
- Văn bản mẫu ghi cứng "24 giờ / 30 phút / 72 giờ" như SRS 6.5; đổi `*_TOKEN_TTL` không đổi chữ trong thư (kể cả dev rút gọn).
- Chưa chạy `docker compose up` với worker mới (không cần cho story này; US-P2-02 sẽ dựng stack).
