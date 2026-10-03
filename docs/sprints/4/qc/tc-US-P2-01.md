# QC test case — US-P2-01 (migration `00003` + `00004 auth_hardening`, `auth.Tokens`, lõi gửi mail: hàng đợi, thử lại, dead-letter)
Nguồn: `docs/specs/FEAT-account-security/US.md` US-P2-01 AC1–AC12 + `SRS.md` 5.2–5.6 (4 bảng, 3 enum), 6.5 (7 mẫu thư), 5.6 (token phát lúc gửi). Hộp đen: QC tự dựng DB / Mailpit và đo bằng `psql` + REST API của Mailpit; test Go của dev chạy thêm. Mọi đường dẫn tới liên kết chứa token ⇒ **token phải được băm trong DB** (nguyên tắc `AGENTS.md`).

Tiền điều kiện chung: worktree `TA_Agent_v2-s4` (`sprint/4-p2`); `source ~/.zprofile`; `export TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=unix://$HOME/.colima/default/docker.sock`; stack test (`docker-compose.test.yml`: Postgres, Redis, Mailpit, 2 gateway `testroutes`, Caddy, worker) chạy; biến `GW`, `MP`, `PW`, `PSQL`, `RDS`, `ORIGIN`, `j`, `login`, `mail_text`, `mail_token`, `mail_count`, `idem`, `age_token` đúng như "Quy ước kiểm chung" của `US.md`. Công cụ: **S** = shell (QC bọc `scripts/p201.sh`, hàm `tc_p201_NN`), **G** = `go test` của dev, **D** = đo qua DB / Mailpit / Redis, **T** = tay. Thiếu thành phần → FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P201-01 | AC1 | DB mới tới `00002` | **S** `goose … up`; `goose … status \| grep -cE '0000[34]_'`; `select max(version_id) from goose_db_version where is_applied` | `2` dòng; phiên bản `4` |
| TC-P201-02 | AC1 | – | **D** `information_schema.columns` + `pg_constraint` + `pg_indexes` của `auth_sessions`, `auth_tokens`, `login_attempts`, `mail_outbox`; so từng cột / CHECK / chỉ mục với SRS 5.2–5.6 (QC chép bảng SRS thành `expected-auth.tsv`); `\dT+` 3 enum `auth_token_kind`, `mail_status`, `login_outcome` | Khớp từng cột; đúng 4 bảng mới của `00004`; 3 enum đúng giá trị (`VERIFY_EMAIL, RESET_PASSWORD, INVITE`; `QUEUED, SENT, DEAD`; giá trị `login_outcome` theo SRS) |
| TC-P201-03 | AC1 | – | **S** `git diff --stat origin/main -- backend-go/db/migrations/0000[12]*`; `sha256sum` `00001*`,`00002*` so với sprint 3 | Rỗng / cùng hash; bảng của `00001`–`00002` không bị `ALTER` (`git diff` rỗng) |
| TC-P201-04 | AC1 | – | **S** `goose down` hai bước rồi `up` lại trên DB dev trống; `goose status` | `rc=0`; `up` lại sạch; không còn bảng mồ côi / enum treo sau `down` (`select count(*) from pg_type where typname in (…)` = 0 sau `down`) |
| TC-P201-05 | AC1 | – | **G** `go test ./internal/store/... -run TestAuthSchema -v` | `ok` |
| TC-P201-06 | AC2 | DB | **D** QC tự `INSERT` sai bằng `psql` cho **mọi** ca AC2 (≥ 12): `token_hash` 63 ký tự / không hex; trùng `token_hash`; `kind='X'`; vừa `used_at` vừa `revoked_at`; `refresh_hash` trùng / sai độ dài; `revoked_at` có thiếu `revoked_reason` và ngược lại; `absolute_expires_at < expires_at`; `to_addr='A@x.vn'`, `'abc'`; `status='X'`; trùng `dedupe_key`; `email_hash` không 64 hex | Mỗi ca bị từ chối `23514`/`23505`/`23502` đúng loại; 0 dòng lọt |
| TC-P201-07 | AC2 (ca lành) | – | **D** `INSERT` hợp lệ cho từng bảng | Thành công; ràng buộc không chặn nhầm |
| TC-P201-08 | AC2 | – | **G** `-run TestAuthConstraints -v` | `ok`; ≥ 12 ca |
| TC-P201-09 | AC3 (**token băm**) | stack, Mailpit | **D** kích hoạt đăng ký (US-P2-03) hoặc `Issue` qua test Go; lấy token từ thư (`mail_token`); `select count(*) from auth_tokens where token_hash = '<token rõ>'`; `where token_hash = encode(sha256('<token>'::bytea),'hex')` | Bản rõ **không** có ở DB (0 khớp); chỉ có `sha256` hex (1 khớp); token đúng **43 ký tự base64url** (`[A-Za-z0-9_-]{43}`) |
| TC-P201-10 | AC3 (quét toàn DB) | – | **D** `pg_dump` toàn bộ DB sau các luồng; `grep -c '<token rõ>'`; quét các cột text/jsonb của `auth_*`, `mail_outbox`, `outbox`, `audit_log`, `jobs` | **0** lần xuất hiện bản rõ ở bất kỳ nơi nào (cũng không ở `last_error`, `payload`, `details`) |
| TC-P201-11 | AC3 (log) | – | **D** `docker compose logs gateway worker caddy mailpit \| grep -c '<token rõ>'` (trừ **nội dung thư ở Mailpit**, là nơi duy nhất chứa token) | `0` ở log gateway / worker / caddy / postgres / redis |
| TC-P201-12 | AC3 | – | **D** `select count(*) from auth_tokens where length(token_hash) <> 64`; phát token mới cùng `(user, kind)`; xem token cũ | `0`; token cũ chưa dùng có `revoked_at`; token cũ **không** dùng được nữa |
| TC-P201-13 | AC3 | – | **D** độ ngẫu nhiên: phát 1.000 token, đếm trùng; kiểm bit phân bố sơ bộ | 1.000 khác nhau; không có tiền tố cố định |
| TC-P201-14 | AC3 | – | **G** `-race -run 'TestTokenIssueHashed\|TestTokenIssueRevokesPrevious\|TestTokenNeverLogged' -v` | `ok` |
| TC-P201-15 | AC4 | – | **D** gây lỗi sau khi xếp thư (route thử rollback) rồi đếm `mail_outbox`/`outbox`; rồi commit | Rollback: **0** dòng; commit: đúng **1** dòng `mail_outbox` + **1** `outbox` topic `mail.send` payload `{mail_id}` |
| TC-P201-16 | AC4 | – | **G** `go test ./internal/mail/... -run 'TestEnqueueRollback\|TestEnqueueCommit' -v` | `ok` |
| TC-P201-17 | AC5 | stack | **D** kích hoạt gửi `verify_email` tới `qc-a@example.test`; `mail_count`, đọc thư qua `$MP/message/<id>` | Thư đến đúng người; tiêu đề tiếng Việt; có **cả** `Text` lẫn `HTML`; `From`=`MAIL_FROM`; liên kết bắt đầu `APP_PUBLIC_URL`; `mail_outbox.status=SENT`, `sent_at` có; `grep -cE '\{\{\|<no value>\|%!'` = `0` |
| TC-P201-18 | AC5 | – | **G** `-tags integration -run TestSendViaMailpit -v` | `ok` |
| TC-P201-19 | AC6 | stack | **D** `docker compose stop mailpit`; kích hoạt thư; theo dõi `mail_outbox` / thời điểm thử (log worker) | Thử lại theo 1 s, 5 s, 30 s (3 lần sau lần đầu; ghi thời điểm, dung sai ±30 %); sau lần thứ 4: `status=DEAD`, `outbox.dead_at` có giá trị, tin vào `outbox.dispatch.dead` |
| TC-P201-20 | AC6 | – | **D** `select last_error` của dòng DEAD: độ dài, chứa `@` / `token=` / nội dung thư? | `≤ 1000` ký tự; **không** email / token / nội dung; `select count(*) from mail_outbox where last_error ~* '@\|token='` = `0` |
| TC-P201-21 | AC6 (hồi phục) | – | **D** bật Mailpit **trước** lần thử cuối | Thư đến đúng **1** lần (`mail_count`=1), `SENT` |
| TC-P201-22 | AC6 | – | **G** `-tags integration -run 'TestRetryThenDead\|TestRecoverMidRetry' -v` | `ok` |
| TC-P201-23 | AC7 | 2 worker | **D** giao hai lần cùng `mail.send` (`XADD` lại cùng `mail_id` vào stream; hoặc `kill -9` worker giữa lúc rồi `XAUTOCLAIM`); đếm thư ở Mailpit | **1** thư (khoá `mail_outbox.id`, `SENT` chặn); `dedupe_key` chặn xếp hai thư cùng loại cùng sự kiện (xếp lần 2 → không thêm dòng) |
| TC-P201-24 | AC7 | – | **G** `-tags integration -run 'TestRedeliverOnce\|TestDedupeKey' -v` | `ok` |
| TC-P201-25 | AC8 (**XSS**) | – | **D** tạo người dùng `full_name = <script>alert(1)</script> "Nguyễn"`; kích hoạt từng mẫu trong 7; đọc `HTML` và `Text` của thư | HTML thoát (`&lt;script&gt;`, `&quot;`); chữ thuần giữ nguyên văn; **tiêu đề** không chứa tên người; `grep -c '<script>'` trong HTML = `0` |
| TC-P201-26 | AC8 | – | **G** `-run 'TestTemplatesRender\|TestTemplatesEscape\|TestTemplateMissingVar' -v`; so `internal/mail/testdata/golden/*.txt` với lời văn SRS 6.5 (đọc mắt 3 mẫu) | `ok`; 7 mẫu × {HTML, chữ thuần}; mẫu thiếu biến → `DEAD` ngay, **không** thử lại |
| TC-P201-27 | AC9 | sau các luồng PU-03/04/06 | **D** quét `mail_outbox.payload` mọi dòng: khoá `token\|password\|link\|url` ; `grep -rn 'mail_outbox' backend-go/internal --include=*.go \| grep -E 'handler\|http' \| grep -v _test.go \| wc -l` | Không có khoá bí mật trong payload (chỉ `user_id`, `course_id`, `at`…); `0` handler; không API HTTP liệt kê `mail_outbox` (gọi `GET /api/v1/mail*`, `/admin/mail*` → 404) |
| TC-P201-28 | AC9 | – | **G** `-run TestPayloadHasNoSecrets` | `ok` |
| TC-P201-29 | AC10 | `smtpmock` hoặc SMTP giả của QC trả 550 / 451 | **D** SMTP giả: 550; rồi 451 | 550: `DEAD` **ngay lần đầu** (không đợi 3 lần), `last_error` là loại lỗi, log `warn` không có email; 451: thử lại như AC6; mẫu không tồn tại / email không hợp lệ → `DEAD` ngay |
| TC-P201-30 | AC10 | – | **G** `-tags integration -run 'TestPermanentFailureNoRetry\|TestTemporaryFailureRetries' -v` | `ok` |
| TC-P201-31 | AC11 | – | **S** `cd backend-go && sqlc generate && sqlc diff; echo rc=$?`; `grep -rnE '(SELECT\|INSERT\|UPDATE\|DELETE) ' backend-go/internal/auth backend-go/internal/mail --include=*.go \| grep -v _test.go \| wc -l` | `rc=0`; `0` |
| TC-P201-32 | AC12 | – | **S** `golangci-lint run ./...; echo rc=$?`; gieo tệp tạm `internal/zz/x.go` import truy vấn `auth_tokens` ngoài `internal/auth` → lint; xoá | `rc=0`; gieo → lint **đỏ** (depguard); xoá → sạch |
| TC-P201-33 | AC12 | – | **G** `go test ./internal/auth/... -run TestOnlyAuthPackageTouchesTokenTables`; **S** chỉ worker đăng ký handler `mail.send` (gateway không tiêu thụ stream) | `ok`; `redis-cli XINFO GROUPS` stream `mail.send`: consumer chỉ từ container worker |
| TC-P201-34 | tổng | – | **S** `go vet ./... && golangci-lint run && go test -race -count=1 ./... && go test -count=1 ./internal/contract/...; echo rc=$?` | `rc=0`; hợp đồng PG / LLM nguyên vẹn |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Token bản rõ ở DB / log / dump / `last_error` | 09–11, 20 |
| Thư mất khi rollback / gửi đôi khi giao lại | 15, 23 |
| SMTP chết → mất thư vĩnh viễn không dấu | 19–21 |
| Lỗi vĩnh viễn thử lại vô ích | 29 |
| XSS trong thư qua tên người dùng | 25 |
| Bí mật trong hàng đợi; API liệt kê hàng đợi | 27 |
| Bảng token bị truy cập ngoài `internal/auth` | 32, 33 |

## Câu hỏi cho BA / PM
- **Q-QC-P201-1** — `token` ở Mailpit là nơi duy nhất chứa bản rõ (SRS 5.6): QC chấp nhận Mailpit là ngoại lệ hợp lệ khi quét "0 lần" (chỉ quét DB, log). Đúng? — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Đúng: bản rõ chỉ tồn tại trong nội dung thư gửi đi (Mailpit) và bộ nhớ consumer; QC quét "0 lần" ở DB / log / Redis / outbox / audit, không quét Mailpit (v1.1, AC3 + SRS 5.6).
- **Q-QC-P201-2** — AC10 "550 vĩnh viễn": Mailpit không mô phỏng; QC cần SMTP giả riêng (Bun `net.createServer` nói SMTP tối thiểu) và cấu hình `SMTP_HOST` trỏ vào. Có biến cho phép thay `SMTP_HOST` ở stack test? — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Có: `SMTP_HOST` / `SMTP_PORT` của worker ghi đè được bằng compose override (`up -d --force-recreate --no-deps worker`) ở stack test — đã ghi vào SRS 8.1 (ghi chú cho QC, v1.1). Dev đảm bảo worker đọc hai biến này lúc khởi động.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1 (FEAT-account-security, APPROVED 2026-10-03).

Tổng: 34 TC.
