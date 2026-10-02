# QC report — US-PG-02 (schema + sqlc) · sprint 2 (PG Nền Go) · Kết luận: **PASS** (vòng sửa 1, 2026-10-03)

Worktree `TA_Agent_v2-s2`, nhánh `sprint/2-pg` @ `4bf829c` (CI xanh run `37055504838`). Máy: colima, curl 8.7.1, sqlc 1.31.1. Đo 2026-10-03. Script: `pg02.sh` (lượt đầu → sau triage + sửa script: 63/70 → 69/70). Nhật ký thô: `run-logs/`.
**Tóm tắt:** 70/70 TC PASS, **0 FAIL** sau vòng sửa 1 (lỗi sản phẩm BUG-PG-1/2/3/4/6 dev đã sửa; BUG-PG-5, bearerAuth, `time.Sleep` drain: PM chốt #12 — spec v1.5; TC-PG05-41: #13).

## TC
| TC-id | PASS/FAIL | AC | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-PG02-01 | PASS | AC1 | danh sách bảng public = [audit_log goose_db_version idempotency_keys jobs outbox users] ; bảng nghiệp vụ ngoài phạm vi (courses/enrollments/documents/notifications) = [0] |
| TC-PG02-02 | PASS | AC1 | danh sách enum = [job_status user_role user_status] ; user_role = [{ADMIN,TEACHER,TA,STUDENT}] (+2 dòng ok) |
| TC-PG02-03 | PASS | AC1 | extension vector = [vector] |
| TC-PG02-04 | PASS | AC2 | số cột users = [16] ; 16 cột users khớp SRS (tên, kiểu, nullable, mặc định) |
| TC-PG02-05 | PASS | AC2 | số cột users (information_schema) = [16] ; 6 cột P2/P5/P8 = [6] |
| TC-PG02-06 | PASS | AC2 | go test ./internal/store -run 'TestSchema_UsersColumns' rc = [0] ; --- PASS: TestSchema_UsersColumns |
| TC-PG02-07 | PASS | AC3 | email có chữ hoa 'A@X.com' → SQLSTATE 23514 ~ /23514:/ ; email có chữ hoa 'A@X.com' → thông báo ~ /users_email_lower_chk/ |
| TC-PG02-08 | PASS | AC3 | email trùng 'qc-dup@x.com' hai lần → SQLSTATE 23505 ~ /23505:/ ; email trùng 'qc-dup@x.com' hai lần → thông báo ~ /users_email_key/ |
| TC-PG02-09 | PASS | AC3 | role='SUPERUSER' (ngoài enum) → SQLSTATE 22P02 ~ /22P02:/ |
| TC-PG02-10 | PASS | AC3 | failed_logins = -1 → SQLSTATE 23514 ~ /23514:/ ; failed_logins = -1 → thông báo ~ /users_failed_logins_chk/ |
| TC-PG02-11 | PASS | AC3 | status='ACTIVE' và password_hash = '' (rỗng) → SQLSTATE 23514 ~ /23514:/ ; status='ACTIVE' và password_hash = '' (rỗng) → thông báo ~ /users_active_password_chk/ |
| TC-PG02-12 | PASS | AC3 (biên) | status='ACTIVE' và password_hash NULL → SQLSTATE 23514 ~ /23514:/ ; status='ACTIVE' và password_hash NULL → thông báo ~ /users_active_password_chk/ |
| TC-PG02-13 | PASS | AC3 | student_code cho vai TEACHER → SQLSTATE 23514 ~ /23514:/ ; student_code cho vai TEACHER → thông báo ~ /users_student_code_role_chk/ |
| TC-PG02-14 | PASS | AC2 / SRS 5.1 | version = 0 (SRS 5.1 CHECK version >= 1) → SQLSTATE 23514 ~ /23514:/ |
| TC-PG02-15 | PASS | AC3 (hợp lệ) | status='INVITED' không mật khẩu là hợp lệ không được lỗi !~ /ERROR/ ; status='INVITED' không mật khẩu là hợp lệ → id uuid v7 ~ /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f] |
| TC-PG02-16 | PASS | AC3 (hợp lệ) | status='ACTIVE' có mật khẩu bcrypt, email chữ thường không được lỗi !~ /ERROR/ ; status='ACTIVE' có mật khẩu bcrypt, email chữ thường → id uuid v7 ~ /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89 |
| TC-PG02-17 | PASS | AC3 (hợp lệ) + SRS 5.1 | student_code với role STUDENT không được lỗi !~ /ERROR/ ; student_code với role STUDENT → id uuid v7 ~ /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f] (+1 dòng ok) |
| TC-PG02-18 | PASS | AC3 | go test ./internal/store -run 'TestSchema_UsersConstraints' rc = [0] ; --- PASS: TestSchema_UsersConstraints |
| TC-PG02-19 | PASS | AC4 | số index = [17] ; danh sách index = [audit_log_actor_created_idx audit_log_course_created_idx audit_log_entity_idx audit_log_pke |
| TC-PG02-20 | PASS | AC4 | users_email_key UNIQUE (email) ~ /CREATE UNIQUE INDEX.*\(email\)/ ; users_email_key không partial !~ /WHERE/ (+19 dòng ok) |
| TC-PG02-21 | PASS | AC4 | cột id mặc định uuidv7() = [5] ; PK là cột id kiểu uuid (5 bảng) = [5] |
| TC-PG02-22 | PASS | AC4 / SRS 5.2 | số cột audit_log = [10] ; 10 cột audit_log khớp SRS (tên, kiểu, nullable, mặc định) (+1 dòng ok) |
| TC-PG02-23 | PASS | AC4 / SRS 5.3 | số cột outbox = [11] ; 11 cột outbox khớp SRS (tên, kiểu, nullable, mặc định) |
| TC-PG02-24 | PASS | AC4 / SRS 5.4 | số cột jobs = [10] ; 10 cột jobs khớp SRS (tên, kiểu, nullable, mặc định) |
| TC-PG02-25 | PASS | AC4 / SRS 5.5 | số cột idempotency_keys = [8] ; 8 cột idempotency_keys khớp SRS (tên, kiểu, nullable, mặc định) |
| TC-PG02-26 | PASS | AC4 | idempotency_keys_user_endpoint_key_key UNIQUE = [t] ; khoá đúng ba cột ~ /\(user_id, endpoint, key\)/ |
| TC-PG02-27 | PASS | AC4 (biên) | attempts = -1 → SQLSTATE 23514 ~ /23514:/ ; attempts = 5 → SQLSTATE 23514 ~ /23514:/ (+4 dòng ok) |
| TC-PG02-28 | PASS | AC4 | dispatched_at và dead_at cùng có giá trị → SQLSTATE 23514 ~ /23514:/ ; chỉ dispatched_at không được lỗi !~ /ERROR/ (+3 dòng ok) |
| TC-PG02-29 | PASS | AC4 (biên) | độ dài topic hợp lệ = [64] ; topic 64 ký tự (biên trên hợp lệ) không được lỗi !~ /ERROR/ (+13 dòng ok) |
| TC-PG02-30 | PASS | AC4 | SUCCEEDED mà finished_at NULL → SQLSTATE 23514 ~ /23514:/ ; FAILED mà finished_at NULL → SQLSTATE 23514 ~ /23514:/ (+6 dòng ok) |
| TC-PG02-31 | PASS | AC4 (biên) | progress = -1 → SQLSTATE 23514 ~ /23514:/ ; progress = 101 → SQLSTATE 23514 ~ /23514:/ (+4 dòng ok) |
| TC-PG02-32 | PASS | AC4 | trigger users_set_updated_at tồn tại = [1] ; hàm set_updated_at tồn tại = [1] (+3 dòng ok) |
| TC-PG02-33 | PASS | AC5 | INSERT audit_log không lỗi !~ /ERROR/ ; số dòng entity='qc-test' tăng 1 = [1] |
| TC-PG02-34 | PASS | AC5 (nhánh lỗi) | UPDATE → SQLSTATE 42501 ~ /42501:/ ; UPDATE → thông báo ~ /audit_log is append-only/ |
| TC-PG02-35 | PASS | AC5 (nhánh lỗi) | DELETE → SQLSTATE 42501 ~ /42501:/ ; DELETE → thông báo ~ /audit_log is append-only/ |
| TC-PG02-36 | PASS | AC5 (nhánh lỗi) | TRUNCATE → SQLSTATE 42501 ~ /42501:/ ; TRUNCATE → thông báo ~ /audit_log is append-only/ (+1 dòng ok) |
| TC-PG02-37 | PASS | AC5 / SRS 5.2 | audit_log_no_update BEFORE UPDATE OR DELETE ~ /BEFORE (UPDATE OR DELETE\|DELETE OR UPDATE) ON public.audit_log/ ; audit_log_no_update FOR EACH ROW ~ /FOR EACH ROW/ (+3 dòng ok) — _sửa script QC: regex nhận `DELETE OR UPDATE` (pg_get_triggerdef)_ |
| TC-PG02-38 | PASS | AC5 | go test ./internal/store -run 'TestSchema_AuditAppendOnly' rc = [0] ; --- PASS: TestSchema_AuditAppendOnly |
| TC-PG02-39 | PASS | AC6 | migrate down rc = [0] ; migrate up rc = [0] (+1 dòng ok) — _sửa script QC: bỏ dòng `\restrict` token ngẫu nhiên của pg_dump_ |
| TC-PG02-40 | PASS | AC6 | migrate down rc = [0] ; số bảng của 00001 còn lại sau down = [0] (+3 dòng ok) |
| TC-PG02-41 | PASS | AC6 | migrate up lần hai rc = [0] ; goose_db_version không thêm dòng = [2] (+1 dòng ok) |
| TC-PG02-42 | PASS | AC6 | goose_db_version.version_id mới nhất = [1] |
| TC-PG02-43 | PASS | AC6 | go test ./db -run 'TestMigrations_RoundTrip' rc = [0] ; --- PASS: TestMigrations_RoundTrip |
| TC-PG02-44 | PASS | AC7 | pnpm dev rc = [0] ; trạng thái service migrate = [migrate exited 0] (+3 dòng ok) |
| TC-PG02-45 | PASS | AC7 | migrate có FinishedAt ~ /^[0-9]{4}-/ ; StartedAt(/edupilot-gateway-1) > migrate.FinishedAt = [t] (+3 dòng ok) |
| TC-PG02-46 | PASS | AC7 | migrate DATABASE_URL trỏ thẳng postgres ~ /@postgres(:5432)?// ; migrate không trỏ pgbouncer !~ /pgbouncer/ (+2 dòng ok) |
| TC-PG02-47 | PASS | AC8 | sqlc diff rc = [0] ; sqlc diff không in gì (số dòng) = [0] — _sửa script QC: cài sqlc 1.31.1 (= CI)_ |
| TC-PG02-48 | PASS | AC8 | Spec v1.5 / #12: quét `float64` chỉ mã sản xuất (`grep … --include=*.go \| grep -v _test.go` = 0); enum Go = 3; `time.Time` ≥ 1. TC và `pg02.sh` đã sửa theo #12. |
| TC-PG02-49 | PASS | AC8 | OFFSET trong internal/store/queries = [0] ; số file queries/*.sql = 5 (≥ 1) (+1 dòng ok) |
| TC-PG02-50 | PASS | AC9 (nhánh lỗi) | sqlc diff rc (phải khác 0) = [1] (≠ [0]) ; diff nêu tên file sinh ra lệch ~ /\.go/ (+1 dòng ok) — _sửa script QC: cài sqlc 1.31.1 (= CI)_ |
| TC-PG02-51 | PASS | AC10 | go test ./internal/store -run 'TestVectorConventions' rc = [0] ; --- PASS: TestVectorConventions |
| TC-PG02-52 | PASS | AC10 | go test ./internal/platform/db -run 'TestVector_ThroughPgBouncer' rc = [0] ; --- PASS: TestVector_ThroughPgBouncer |
| TC-PG02-53 | PASS | AC10 | số dòng chứa halfvec_cosine_ops = 1 (≥ 1) ; mọi dòng halfvec_cosine_ops đều là chú thích SQL (bắt đầu bằng --) = [0] (+3 dòng ok) |
| TC-PG02-54 | PASS | AC11 | go test ./internal/platform/blob -run 'TestBlob_RoundTrip\|TestBlob_Presign\|TestBlob_InvalidKey\|TestBlob_Cancel ; --- PASS: TestBlob_RoundTrip (+3 dòng ok) |
| TC-PG02-55 | PASS | AC11 (hộp đen bổ sung) | BLOB_BUCKET trong .env.local ~ /^[a-z0-9][a-z0-9.-]{2,62}$/ ; MinIO /minio/health/live = [200] (+2 dòng ok) |
| TC-PG02-56 | PASS | AC12 | go test ./internal/platform/blob -run 'TestPresign_PublicHost\|TestPresign_NoNetwork' rc = [0] ; --- PASS: TestPresign_PublicHost (+1 dòng ok) |
| TC-PG02-57 | PASS | AC12 (hộp đen bổ sung) | BLOB_ENDPOINT của gateway = [minio:9000] ; BLOB_PUBLIC_ENDPOINT của gateway = [localhost:9000] (+1 dòng ok) |
| TC-PG02-58 | PASS | AC13 | go test ./internal/platform/outbox -run 'TestOutbox_SameTransaction_Commit\|TestOutbox_SameTransaction_Rollback ; --- PASS: TestOutbox_SameTransaction_Commit (+1 dòng ok) |
| TC-PG02-59 | PASS | AC13 (hộp đen) | dòng outbox topic qc.rollback.test = [0] ; dòng jobs kind qc.rollback = [0] (+2 dòng ok) |
| TC-PG02-60 | PASS | AC13 (hộp đen) | POST /_test/jobs trả job id ~ /^[0-9a-f-]{36}$/ ; số dòng jobs = [1] (+1 dòng ok) |
| TC-PG02-61 | PASS | AC14 | go test ./internal/platform/outbox -run 'TestOutbox_TwoWorkers_ExactlyOnce' rc = [0] ; --- PASS: TestOutbox_TwoWorkers_ExactlyOnce |
| TC-PG02-62 | PASS | AC14 (hộp đen) | số worker lúc chạy = [2] ; job nhận 202 = [100] (+9 dòng ok) |
| TC-PG02-63 | PASS | AC15 | go test ./internal/platform/outbox -run 'TestOutbox_RetryThenDead\|TestOutbox_PanicIsFailure\|TestOutbox_Unknown ; --- PASS: TestOutbox_RetryThenDead (+2 dòng ok) |
| TC-PG02-64 | PASS | AC15 (hộp đen, SRS 3.4) | tạo được dòng outbox topic lạ ~ /^[0-9a-f-]{36}$/ ; attempts (1 lần đầu + 3 lần thử lại) = [4] (+12 dòng ok) — _sửa script QC: `returning id` in thêm "INSERT 0 1" → `head -1`_ |
| TC-PG02-65 | PASS | AC15 / SRS 5.3 | last_error có nội dung ~ /^[1-9][0-9]*\\|/ ; độ dài last_error = 45 (≤ 1000) (+1 dòng ok) — _sửa script QC: như 64_ |
| TC-PG02-66 | PASS | AC16 | go test ./internal/platform/outbox -run 'TestOutbox_ConsumerCrash_Reclaimed\|TestOutbox_StaleEnqueuedRequeued'  ; --- PASS: TestOutbox_ConsumerCrash_Reclaimed (+1 dòng ok) |
| TC-PG02-67 | PASS | AC16 (hộp đen) | job nhận 202 = [30] ; dòng outbox job.enqueue mới = [30] (+5 dòng ok) |
| TC-PG02-68 | PASS | AC17 (ràng buộc thay thế) | dòng users có password_hash không phải bcrypt = [0] |
| TC-PG02-69 | PASS | AC17 (ràng buộc thay thế) | hai người dùng khác nhau dùng cùng khoá: hợp lệ !~ /ERROR/ ; chèn được 2 dòng ~ /^2$/ (+2 dòng ok) |
| TC-PG02-70 | PASS | AC1 / AC4 (v1.2 #Q-QC-02-3) | bảng thử _test_% đang tồn tại (điều kiện để phép đo có nghĩa) = 1 (≥ 1) ; bảng public sau khi loại _test_% = [audit_log goose_db_version idempotency_keys jobs outbox users] (+2 dòng ok) |

## Lỗi
_Vòng sửa 1: BUG-PG-1, 2, 3, 4, 6 dev đã sửa và QC chạy lại — **đóng**; BUG-PG-5 không phải lỗi (PM chốt #12, spec v1.5). Bảng dưới là hồ sơ vòng 1._

| Mã | Mức | Nơi | Bước tái hiện | Thấy | Mong đợi | AC / TC |
| --- | --- | --- | --- | --- | --- | --- |
| BUG-PG-1 | **trung bình** | Caddy (`Caddyfile`, `unhealthy_status 503`) | Tắt Redis; gọi `GET /api/v1/events` (hoặc `_test/error/503`) 6 lần liên tiếp qua `https://localhost` | Lần 1–2 `503` JSON `SERVICE_UNAVAILABLE`; lần 3 và 6 là `503` rỗng (không Content-Type, không `X-Instance-Id`, không `Retry-After`) vì Caddy coi cả hai gateway là hỏng | Mọi 503 của ứng dụng (trừ readyz) vẫn là JSON có `Retry-After`; Caddy không loại upstream vì 503 hợp lệ của ứng dụng | US-PG-05 AC15 · TC-PG05-68, US-PG-06 AC5 · TC-PG06-18 |
| BUG-PG-2 | trung bình-thấp | `internal/httpapi/sse/handler.go` | `Last-Event-ID: 99999999999999-0` rồi phát `test.future` | Có `ready`, không nhận sự kiện mới (id live nhỏ hơn Last-Event-ID nên bị loại) | Không `resync`, nhận sự kiện mới | US-PG-05 AC9 · TC-PG05-41 |
| BUG-PG-3 | thấp | `cmd/gateway/serve_test.go:130` (`TestServe_InvalidEnvNamesVariable/JWT_EXPIRATION`) | `go test -race ./...` nhiều lần | Đỏ chập chờn: "log lộ giá trị \"abc\"" vì `trace_id`/hostname ngẫu nhiên chứa "abc" (thấy 1/6 lần ở GATE-03) | Chỉ so giá trị trong trường `msg`, không so toàn dòng | US-PG-01 AC2 |
| BUG-PG-4 | thấp | `cmd/gateway` (chờ phụ thuộc lúc khởi động) | `REDIS_URL` cổng đóng, `STARTUP_TIMEOUT=3s` | 1 dòng warn `dependency not ready` (Redis) thay vì ≥ 2 | "Mỗi giây một dòng" (SRS 3.4) | US-PG-01 AC14 · TC-PG01-76 |
| BUG-PG-5 | thấp | `internal/store/vector_test.go:74,77` | `grep -rn float64 backend-go/internal/store` | 2 dòng `float64` (khoảng cách cosine, không phải điểm) | 0 dòng (TC) — hoặc BA chốt loại tệp `_test` | US-PG-02 AC8 · TC-PG02-48 |
| BUG-PG-6 | thấp (PM chốt) | `backend-go/api/openapi.yaml` | `grep -c _test`; tìm `bearerAuth` trong khối `jobs/{id}`, `events` | Chú thích dòng 7 có chữ `_test`; `bearerAuth` chỉ khai toàn cục (dòng 14) | 0 lần `_test`; khai tường minh theo AC7 (hoặc PM chấp nhận mặc định toàn cục) | US-PG-06 AC2, AC7 · TC-PG06-05, 30 |

Quan sát (không FAIL TC): worker xử lý việc **tuần tự** — việc 100 bước chặn việc mặc định sau nó ~17 s (TC-PG03-104 đã chỉnh thời gian chờ); `Vary` của gateway gửi thành hai dòng riêng.

## Sửa công cụ QC (theo góp ý #4–#10, PM #11) và lỗi script tìm thấy lúc chạy
Mỗi sửa ghi ở cột bằng chứng của TC tương ứng (_sửa script QC: …_). Tóm tắt: #4 literal 31 byte (`pg01.sh` 09); #5 `-w '%{http_code}'` (`pg07.sh` 15); #6 `</dev/null` (`lib.sh` `rl_reset`); #7 `--no-deps` (`pg07.sh` 54); #8 regex `9000(-[0-9]+)?->` (`pg07.sh` 56); #9 bỏ so gián tiếp + `':!legacy'` (`pg01.sh` 78/79, `pg07.sh` 58); #10 `paste -sd' ' -`, gói `httpapi` riêng, `--filter id=` (`gate-pg.sh`). Lỗi script mới (đo tay xác nhận, gateway đúng): `psql` in thêm "INSERT 0 1" (`pg02.sh` 64/65); `pg_dump` token `\restrict` ngẫu nhiên (39); `pg_get_triggerdef` đảo thứ tự sự kiện (37); `xargs` đưa biến môi trường sau `serve` (`pg04.sh` 41); `wait` trần chặn `sse_burst` (`sse-reconnect.sh`; 05/33/35/39); curl 8.7 in `200` khi hết `-m` giữa thân SSE (18/20/21/61); `Connection` bị Caddy bỏ (05-02); slog in level `ERROR` hoa (03-07); `Vary` hai dòng (03-76); mã dồn dòng (03-74); `wait` pid của subshell, `sse_ns` thiếu bỏ "data: ", log `compose run` có dòng không phải JSON (gate); `tab.run` truyền args sai (`sse-browser-cut.mjs`, `idem-two-tabs.mjs`). **Môi trường:** `~/.testcontainers.properties` (README), `brew install sqlc` (1.31.1 = CI); `run-all.sh` chết sau story 01 khi chạy nền — chạy từng script ở foreground.
