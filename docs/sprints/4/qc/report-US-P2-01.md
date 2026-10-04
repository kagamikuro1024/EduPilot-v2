# Báo cáo QC — US-P2-01 (migration `00003`+`00004`, `auth.Tokens`, lõi gửi mail)
**Kết luận: PASS.** 34 TC: 31 PASS, 3 PASS có ghi chú (TC-15, TC-32, TC-04). Không lỗi. Bản chấm `876d695` (`sprint/4-p2`), stack riêng của QC: Postgres pgvector 18 + Redis 8 + Mailpit (container `qcp2-*`), 2 gateway `testroutes` + **worker thật** (`SMTP_HOST` trỏ Mailpit; ghi đè sang SMTP giả cho TC-29), đo bằng `psql`, REST Mailpit, Redis CLI; test Go của dev chạy thêm. Q-QC-P201-1/-2: BA đã trả lời (Mailpit là nơi duy nhất chứa token rõ; `SMTP_HOST/PORT` ghi đè được) — QC làm theo.

| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `goose` `0003_`, `0004_` đã áp (2 dòng), phiên bản **4** |
| 02 | PASS | `scripts/p201-schema.py` (QC chép bảng SRS 5.1–5.5 thành kỳ vọng): 4 bảng × {cột/kiểu/null/mặc định}, chỉ mục theo tên (15), nội dung CHECK (định nghĩa `pg_get_constraintdef`), 3 enum đúng giá trị và thứ tự, FK (`sessions`/`tokens` → `users` CASCADE; `login_attempts` không FK), trigger `set_updated_at`: **66 PASS / 0 FAIL** |
| 03 | PASS | `git diff --stat origin/sprint/3-pu-p1 -- db/migrations/0000[12]*` rỗng |
| 04 | PASS* | `TestMigrations_RoundTrip` PASS (down về 0 rồi up giống hệt). *QC không chạy `goose down` tay: `gateway migrate` không có lệnh down; dựa test Go |
| 05, 08, 14, 16, 18, 22, 24, 26, 28, 30 | PASS | `go test -race -tags integration ./db/... ./internal/store/... ./internal/auth/... ./internal/mail/...`: **78 `--- PASS`**, 0 FAIL, 0 SKIP (4 gói `ok`) |
| 06 | PASS | 20 INSERT sai + 4 trùng/FK, mỗi ca **đúng SQLSTATE**: `23514` (hash 63 ký tự / không hex, used+revoked, refresh sai độ dài, revoked thiếu reason và ngược lại, reason ngoài danh sách, absolute < expires, prev sai, email_hash sai, to_addr hoa / không `@`, template sai, attempts 5, SENT thiếu `sent_at`, payload không phải object, `last_error` 1001), `22P02` (enum sai ×3), `23505` (trùng token_hash, refresh_hash, dedupe_key), `23503` (user không tồn tại) |
| 07 | PASS | 4 INSERT lành (token đã dùng, phiên đã thu hồi, THROTTLED, SENT với attempts=4) đều qua |
| 09 | PASS | thư verify: token 43 ký tự base64url; `token_hash = sha256(token)` 1 dòng; `token_hash = <bản rõ>` 0 dòng; `length(token_hash)<>64` 0 |
| 10, 11 | PASS | sau mọi luồng (kể cả SMTP chết, 1.000 thư): `pg_dump` chứa token rõ **0**; log 2 gateway + worker **0**; khoá Redis **0** |
| 12 | PASS | phát lại: 2 dòng, 1 bị thu hồi, token cũ `revoked_at` có |
| 13 | PASS | 1.000 thư → 1.000 token **khác nhau**, 997 tiền tố 3 ký tự khác nhau, tần suất ký tự 599–756 (kỳ vọng ≈ 672), ký tự cuối chỉ thuộc 16 giá trị (32 byte → 43 ký tự) — đúng; `auth_tokens` 1.000 hash khác nhau, 1 còn hiệu lực |
| 15 | PASS* | không có route thử rollback: dựa `TestEnqueueRollback`/`TestEnqueueCommit` PASS |
| 17 | PASS | thư tới sau **208 ms**: Subject "Xác minh email EduPilot của bạn", From `no-reply@edupilot.local`, To đúng, Text 418 B + HTML 919 B, liên kết `APP_PUBLIC_URL/verify-email?token=…`; 0 `{{`/`<no value>`/`%!`; `mail_outbox.status=SENT`, `sent_at` có, outbox `dispatched` |
| 19 | PASS | dừng Mailpit: lần thử tại 0,5 s / 2,0 s / 7,6 s / 38,1 s (khoảng cách 1,5 / 5,5 / 30,5 s ≈ 1 / 5 / 30 s + chu kỳ thăm dò 0,5 s), sau lần 4: `DEAD`, `outbox.dead_at` có, stream Redis `outbox.dispatch.dead` có 1 tin |
| 20 | PASS | `last_error=smtp_unavailable` (16 ký tự); `last_error ~ '@|token='` = 0; token phát lúc gửi bị huỷ (0 dòng còn lại) |
| 21 | PASS | bật Mailpit sau 3,5 s: `SENT attempts=2`, đúng **1** thư |
| 23 | PASS | thêm 4 dòng `outbox` `mail.send` cùng `mail_id` sau khi đã SENT: vẫn **1** thư, **1** token, 5/5 dòng outbox `dispatched`; xếp lần 2 cùng `dedupe_key` → 0 dòng thêm (`ON CONFLICT DO NOTHING`) |
| 25 | PASS | tên `<script>alert(1)</script> "Nguyễn" & <b>x</b>` qua 7 mẫu: HTML `<script>` = 0, `&lt;script&gt;` có, `&quot;` có, `<img src=x onerror>` ở tên lớp → `&lt;img`; **tiêu đề** không chứa tên; chữ thuần giữ nguyên văn |
| 27 | PASS | khoá `token|password|link|url` trong `mail_outbox.payload` = 0; `GET /mail`, `/admin/mail`, `/mail_outbox`, `/admin/mail_outbox` → 404; `TestPayloadHasNoSecrets` PASS |
| 29 | PASS | SMTP giả (Bun, đổi `SMTP_PORT`): **550** → `DEAD` ngay sau 1 lần (519 ms, 1 kết nối, `smtp_permanent_550`), log worker không có email; **451** → thử 4 lần (0,5 / 2,0 / 7,5 / 37,9 s) rồi `DEAD smtp_temporary_451`; mẫu lạ / thiếu biến / `payload_invalid` / `user_missing` → `DEAD` ngay, 0 kết nối SMTP |
| 31 | PASS | `sqlc generate && sqlc diff` rc=0; SQL thô trong `internal/auth`, `internal/mail` = 0 |
| 32 | PASS có ghi chú | `golangci-lint` 0 issues. **Lệch spec đã ghi bởi dev**: depguard không cấm được theo bảng; thay bằng `TestOnlyAuthPackageTouchesTokenTables`. QC gieo `internal/zz/x.go` chứa `auth_tokens` → test **đỏ**; xoá → xanh. Chấp nhận (proposals dev) |
| 33 | PASS | chỉ `cmd/worker/registry.go` đăng ký `mail.send`; gateway không đăng ký |
| 34 | PASS | `go vet` ok, `golangci-lint` 0 issues, `go test -race -count=1 -tags testroutes,integration ./...` rc=0 (24 gói `ok`), `internal/contract` rc=0 |

## Ghi chú
- `mail_outbox.attempts` chỉ đếm lần **thất bại** (gửi thành công lần đầu: `attempts=0`) — khớp ví dụ của dev; ghi để BA đối chiếu với SRS.
- Lần gửi giữ khoá dòng `FOR UPDATE` suốt một lần SMTP (≤ 10 s): chấp nhận cho T1 (dev ghi `ponytail:`).

## Chấm lại sau góp ý #1, #2 (PM ACCEPTED, 2026-10-04)
- TC-32 (AC12c): chấm theo TC mới — `TestOnlyAuthPackageTouchesTokenTables` ok; gieo `internal/zz/x.go` chứa `auth_tokens` → đỏ, xoá → xanh (đã làm ở lần chấm đầu) → **PASS** (bỏ chữ "có ghi chú").
- TC-19/21/29 (#2): đã chấm bằng Mailpit riêng của QC và SMTP giả của QC → **PASS** theo TC mới.
