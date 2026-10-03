# QC test case — US-P1-01 (migration `00002_llm`, `platform/crypto` AES-GCM, `llmconfig`)
Nguồn: `docs/specs/FEAT-llm-gateway/US.md` US-P1-01 AC1–AC12 + `SRS.md` 5.2 (5 bảng: cột, kiểu, CHECK), 5.3 (chỉ mục), 5.4 (định dạng bản mã), 6.3 (truy vấn `usage`), 8.3 (`FAKE_LLM_VALID_KEY`). Hộp đen: QC dựng DB **riêng** và tự kiểm bằng `psql`; test Go của dev chỉ chạy thêm, **không** thay phép đo của QC. Nền: stack sprint 2 (`docker-compose.test.yml`, project `edupilot`).

Tiền điều kiện chung: `source ~/.zprofile`; `export TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=unix://$HOME/.colima/default/docker.sock`; stack test chạy (`$C up -d --wait`, `C="docker compose --env-file .env.local -f docker-compose.test.yml -p edupilot"`); `PSQL="$C exec -T postgres psql -U edupilot -d edupilot -tA"`; `CANARY="sk-LEAK-CANARY-7f3a9c1e"`; `.env.local` có `APP_ENCRYPTION_KEY` (32 byte base64) và `JWT_SECRET_KEY`. Công cụ: **S** = shell (QC bọc `scripts/p101.sh`, hàm `tc_p101_NN`), **G** = `go test` (chạy test của dev), **D** = đối chứng độc lập (script riêng của QC, `openssl`/`python` tự tính), **T** = tay. Chưa có `llmconfig` → TC loại **G** FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P101-01 | AC1 | DB mới, goose tới `00001` | **S** `goose … up`; `goose … status \| grep -c '00002_llm'`; `$PSQL -c "select max(version_id) from goose_db_version where is_applied"` | `00002_llm` có 1 dòng; phiên bản = `2` |
| TC-P101-02 | AC1 | – | **S** `$PSQL -c "select count(*) from information_schema.tables where table_schema='public' and table_name like 'llm\_%'"`; liệt kê tên | `5`: `llm_providers`, `llm_models`, `llm_task_routes`, `llm_audit`, `llm_budgets` |
| TC-P101-03 | AC1 | – | **D** dump `information_schema.columns` (tên, kiểu, nullable, default) của 5 bảng và so từng cột với bảng SRS 5.2 (QC chép bảng SRS thành `expected-cols.tsv` lúc chạy) | Khớp **từng** cột (tên, kiểu, nullable); khoá chính `uuid` default `uuidv7()`; có `created_at`, `updated_at`; **0** khoá ngoại tới `users`/`courses` (`select count(*) from information_schema.table_constraints where constraint_type='FOREIGN KEY' and table_name like 'llm\_%'` chỉ gồm FK nội bộ giữa 5 bảng) |
| TC-P101-04 | AC1 | – | **S** `git diff --stat origin/main -- backend-go/db/migrations/00001*`; `sha256sum` `00001*` so với commit sprint 2 | Rỗng / cùng hash (migration `00001` không đổi) |
| TC-P101-05 | AC1 | – | **S** `goose … down` rồi `up` lại; `goose … status` | `down` xoá sạch 5 bảng, `up` lại được (phiên bản 2); dữ liệu `00001` còn nguyên |
| TC-P101-06 | AC1 | – | **G** `go test ./internal/store/... -run TestLLMSchema -v` | `ok`, không SKIP |
| TC-P101-07 | AC2 | DB có `00002` | **S** QC tự `INSERT` sai bằng `psql` cho **mọi** ca của AC2 (≥ 14): `type='x'`; `openai_compatible` không `base_url`; `kind='x'`; embedding không `dims`; `task='X'`; trùng `(task,fallback_order)`; `fallback_order=-1`; budget `scope='course'` không `course_id`; `scope='system'` có `course_id`; hai budget `system`; hai budget cùng `course_id`; `price_in=-1`; `price_out=-1`; `llm_audit.status='x'` | Mỗi ca bị từ chối với SQLSTATE `23514` (CHECK) / `23505` (UNIQUE) / `23502` (NOT NULL) đúng loại; **0** dòng lọt vào bảng |
| TC-P101-08 | AC2 (ca lành) | – | **S** `INSERT` hợp lệ cho từng bảng (mỗi `type`; 7 giá trị `task`; `scope='course'` với `course_id`) | Thành công (ràng buộc không chặn nhầm); sau đó dọn |
| TC-P101-09 | AC2 | – | **G** `go test ./internal/store/... -run TestLLMConstraints -v` | `ok`; bảng ≥ 14 ca |
| TC-P101-10 | AC3 | DB | **S** `$PSQL -c "select indexname from pg_indexes where tablename like 'llm\_%' order by 1"`; so danh sách với SRS 5.3 | Có đủ: `llm_audit` `(course_id, created_at desc) where course_id is not null`, `(created_at desc)`, `(trace_id)`, `(task, created_at desc)`; `llm_task_routes (task, fallback_order)` duy nhất; `llm_models (provider_id, model)` duy nhất; `llm_budgets` hai chỉ mục duy nhất từng phần |
| TC-P101-11 | AC3 | – | **S** nạp 20.000 dòng `llm_audit` giả (`generate_series`), `ANALYZE`, `EXPLAIN (FORMAT JSON)` truy vấn `usage` theo `task`, theo `day`, theo `course_id` (SRS 6.3) | Plan có `Index Scan`/`Bitmap Index Scan` trên đúng chỉ mục; **không** `Seq Scan` ở `llm_audit`; dọn sau |
| TC-P101-12 | AC3 (rule 13) | – | **S** `grep -n 'course_id' ` trong định nghĩa chỉ mục phức hợp bảng theo lớp | Chỉ mục phức hợp theo lớp **bắt đầu bằng** `course_id` |
| TC-P101-13 | AC4 | `sqlc` có | **S** `cd backend-go && sqlc generate && sqlc diff; echo rc=$?` | `rc=0` |
| TC-P101-14 | AC4 | – | **S** `grep -rnE '(SELECT\|INSERT\|UPDATE\|DELETE) ' internal/llmconfig/*.go internal/llm \| grep -v _test.go \| wc -l` | `0`; truy vấn nằm ở `internal/store/queries/llm*.sql` |
| TC-P101-15 | AC5 | – | **G** `go test -race ./internal/platform/crypto/... -v` | `ok`; có đủ `TestRoundtrip`, `TestNonceUnique1000`, `TestTamper` (≥ 4 vị trí byte), `TestAADMismatch`, `TestWrongKey`, `TestVector`; không SKIP |
| TC-P101-16 | AC5 (đối chứng độc lập) | `openssl`/Python có | **D** QC tự giải mã **một bản mã do Go tạo** (dump `api_key_enc` từ DB, khoá = `APP_ENCRYPTION_KEY`, AAD = id bản ghi theo SRS 5.4) bằng Python `cryptography` AESGCM (hoặc `openssl`), và ngược lại: QC tạo bản mã, Go giải được qua test tạm | Bản mã khớp định dạng `0x01 ‖ nonce(12) ‖ ct ‖ tag(16)` (byte đầu `01`, độ dài = 1+12+len(plain)+16); giải mã chéo hai chiều thành công, trả lại `$CANARY` (không chỉ tin vectơ của dev) |
| TC-P101-17 | AC5 | – | **D** tạo 1.000 bản mã cùng bản rõ qua dịch vụ (hoặc `crypto` test hook) và đếm khác nhau; sửa 1 byte ở nonce / ct / tag / byte phiên bản | 1.000 bản mã **khác nhau**; mỗi chỗ sửa → lỗi xác thực |
| TC-P101-18 | AC6 | `bin/gateway` build | **S** vòng `for v in "" "abc" "$(head -c 16 /dev/urandom \| base64)"`: `APP_ENCRYPTION_KEY="$v" timeout 10 ./bin/gateway serve >/tmp/o 2>&1; echo rc=$?`; `grep -c "$v" /tmp/o`; đo thời gian | Cả 3 lần `rc≠0` trong **≤ 5 s**; thông báo chứa "APP_ENCRYPTION_KEY không hợp lệ: cần 32 byte (base64)"; **không** in lại giá trị khoá (đếm `0`; ca rỗng bỏ qua); không có cổng nào mở |
| TC-P101-19 | AC6 | – | **S** khoá 32 byte hợp lệ nhưng chuỗi có xuống dòng cuối / khoảng trắng; khoá 33 byte; khoá `base64url` | Hành vi đúng SRS (chấp nhận hoặc từ chối nhất quán); không panic; ghi kết quả thật |
| TC-P101-20 | AC7 | dịch vụ có `Create` (qua test Go hoặc qua API P1-04 nếu đã có) | **S** tạo nhà cung cấp với khoá `$CANARY`; `$PSQL -c "select count(*) from llm_providers where position(convert_to('$CANARY','UTF8') in api_key_enc) > 0"`; `select data_type` cột `api_key_enc` | `0`; kiểu `bytea`; `grep -c CANARY` trên `pg_dump` toàn DB = `0`; log của gateway không chứa `$CANARY` |
| TC-P101-21 | AC7 (AAD) | – | **S** `UPDATE llm_providers SET id=<uuid mới> WHERE …` (hoặc hoán đổi `api_key_enc` giữa hai nhà cung cấp); rồi `List` | Nhà cung cấp đó có `key_status="unreadable"` (không panic); nhà khác **không** ảnh hưởng |
| TC-P101-22 | AC7 | – | **G** `go test ./internal/llmconfig/... -run TestKeyAtRest` | `ok` |
| TC-P101-23 | AC8 | – | **G** `go test ./internal/llmconfig/... -run 'TestNoKeyInOutputs\|TestRedacted'` | `ok` |
| TC-P101-24 | AC8 (QC tự quét) | – | **S** `grep -rn 'DecryptKey(' backend-go --include=*.go \| grep -v _test.go \| grep -v 'internal/llm/' \| grep -v 'internal/llmconfig/'`; viết test tạm: `fmt.Sprintf("%v %+v %#v")`, `json.Marshal`, `slog` JSON của `Provider` có khoá `$CANARY` và **4 ký tự cuối khoá** | Lệnh grep rỗng; test tạm: không `$CANARY`, không đuôi khoá trong mọi đầu ra; chỉ có `HasKey` và `KeyStatus`; `String()/GoString()/LogValue()` = `[REDACTED]`; xoá test tạm |
| TC-P101-25 | AC9 | – | **G** `go test ./internal/llmconfig/... -run 'TestVersion\|TestKeyKeepReplace\|TestDeleteInUse\|TestAuditLogRows\|TestLimits' -v` | `ok` cả 5 nhóm |
| TC-P101-26 | AC9 (QC tự đo `audit_log`) | – | **S** thực hiện tạo / sửa / xoá nhà cung cấp, mô hình, tuyến, ngân sách; đếm `select count(*) from audit_log` trước-sau; `details::text` chứa `$CANARY`? | Số dòng tăng **đúng 1** mỗi thao tác; có người làm, hành động, đối tượng; **không** có khoá trong `details` |
| TC-P101-27 | AC9 | – | **S** sửa với `version` cũ; sửa không `api_key`; sửa `api_key=""`; sửa `api_key="khác"`; xoá nhà cung cấp đang dùng bởi tuyến | `ErrVersionConflict` kèm giá trị hiện tại; không `api_key` ⇒ bản mã **giữ nguyên** (so `api_key_enc` trước-sau bằng `md5`); `""` ⇒ lỗi dữ liệu (không xoá ngầm); khác ⇒ bản mã **đổi** (nonce mới); xoá đang dùng ⇒ `ErrProviderInUse` kèm danh sách tác vụ |
| TC-P101-28 | AC9 | – | **S** xoá nhà cung cấp không dùng; hạn mức: tạo nhà cung cấp thứ 21, mô hình thứ 101 của một nhà | Xoá nhà cung cấp **cùng** mô hình của nó trong một giao dịch (không mồ côi); vượt hạn mức → lỗi rõ, không tạo |
| TC-P101-29 | AC10 | – | **S** `SetRoute` từng ca bất biến: chuỗi rỗng; 5 mô hình; mô hình sai loại (chat ↔ embedding); `EMBEDDING` 2 mô hình; `dims=768`; mô hình của nhà cung cấp đang tắt; trùng mô hình trong chuỗi; `temperature=2.1`, `max_tokens=40000`, `timeout_s=0`, `retries=6`, khoá lạ | Mỗi ca bị từ chối đúng lỗi (`ErrDimsMismatch`, `ErrRouteInvalid`…); không dòng nào được ghi (`count(*)` bảng tuyến không đổi); ca lành (`temperature=0`, `2`; `retries=0`, `5`; chuỗi 4 mô hình) được chấp nhận |
| TC-P101-30 | AC10 | – | **S** đổi mô hình `EMBEDDING` so với giá trị đang lưu | Kết quả mang cờ `ReindexRequired` (nêu theo US); không thao tác dữ liệu nhúng cũ |
| TC-P101-31 | AC10 | – | **G** `go test ./internal/llmconfig/... -run TestRouteRules -v` | `ok`; ≥ 12 ca |
| TC-P101-32 | AC11 | – | **G** `go test ./internal/llmconfig/... -run TestServiceRBAC -v`; QC tự gọi ma trận 4 vai × (8 hàm ghi / đọc) | Ghi: chỉ `ADMIN` được, còn lại `ErrForbidden`; đọc: `ADMIN`, `TEACHER` được, `TA`, `STUDENT` `ErrForbidden`; `actor` lấy từ ngữ cảnh, **không** từ tham số (thử truyền `actor` giả qua tham số → không có chỗ truyền / bị bỏ qua) |
| TC-P101-33 | AC12 | – | **S** `grep -rnE 'float(32\|64)' backend-go/internal/llmconfig backend-go/internal/llm/budget backend-go/internal/llm/cost --include=*.go \| grep -v _test.go \| wc -l`; `select data_type` của `price_*`, `daily_limit`, `monthly_limit` | `0`; mọi cột tiền `numeric`; Go dùng `decimal.Decimal` |
| TC-P101-34 | tổng | – | **S** `cd backend-go && go vet ./... && golangci-lint run && go test -race ./...; echo rc=$?` (không bao gồm test cần stack ngoài nếu dev ghi rõ) | `rc=0` |
| TC-P101-35 | tổng | – | **S** `git log --name-only` của story: không tệp ngoài vùng US (không `frontend/`, không `docs/specs/`) và không secret (`git grep -nE 'sk-[A-Za-z0-9]{10,}\|APP_ENCRYPTION_KEY=[A-Za-z0-9+/=]{20,}'` ngoài `.env.example` placeholder) | Không secret thật trong repo |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Khoá mã hoá thiếu / ngắn / sai base64 | 18, 19 |
| Khoá API rõ nằm ở DB / dump / log | 20 |
| Bản mã hoán đổi giữa bản ghi (AAD) | 21 |
| Khoá lộ qua `%v`, JSON, `slog` | 23, 24 |
| Xoá nhầm nhà cung cấp đang dùng; xoá ngầm khoá | 27, 28 |
| Dữ liệu sai lọt qua DB khi bỏ qua dịch vụ | 07, 29 |
| Quyền ở tầng dịch vụ | 32 |
| Tiền dùng `float64` | 33 |

## Câu hỏi cho BA / PM
- **Q-QC-P101-1** — AC7 / AC9 gọi qua "dịch vụ" nhưng chưa có handler (P1-04): QC chỉ tạo dữ liệu qua test Go / tệp Go tạm `internal/llmconfig/qc_probe_test.go` (xoá sau). Chấp nhận, hay dev cung cấp lệnh CLI (`gateway llmconfig …`)? — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Chấp nhận. US-P1-01 không có handler / CLI (handler ở US-P1-04). QC dùng test Go của dev hoặc `qc_probe_test.go` tạm (xoá sau). Đã ghi vào "Quy ước kiểm chung" (spec v1.2). Không thêm CLI.
- **Q-QC-P101-2** — TC-P101-19: SRS chưa nói rõ khoá `base64url` / có khoảng trắng: QC ghi hành vi thực và hỏi BA. — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Spec đã sửa (v1.2, US-P1-01 AC6 + SRS 5.6): chỉ nhận **base64 chuẩn có đệm** (RFC 4648 §4, `+` `/`), cắt khoảng trắng / xuống dòng **hai đầu**; base64url, không đệm, khoảng trắng ở giữa → **từ chối lúc khởi động**, không tự sửa. Ca kiểm cụ thể đã thêm vào AC6.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-llm-gateway, APPROVED 2026-10-03).

Tổng: 35 TC.
