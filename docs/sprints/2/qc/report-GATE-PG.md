# QC report — GATE-PG · sprint 2 (PG Nền Go) · Kết luận: **PASS**

Worktree `TA_Agent_v2-s2`, nhánh `sprint/2-pg` @ `4bf829c` (CI xanh run `37055504838`). Máy: colima, curl 8.7.1, sqlc 1.31.1. Đo 2026-10-03. Script: `gate-pg.sh` (lượt đầu → sau triage + sửa script: 15/25 → 25/25). Nhật ký thô: `run-logs/`.
**Tóm tắt:** 25/25 TC PASS, **0 FAIL** sau vòng sửa 1 (lỗi sản phẩm BUG-PG-1/2/3/4/6 dev đã sửa; BUG-PG-5, bearerAuth, `time.Sleep` drain: PM chốt #12 — spec v1.5; TC-PG05-41: #13).

## Kết luận cổng PG: **PASS**
Vòng sửa 1 (dev `3f1a39d`…`8d8f3c4`; PM #12, #13): 8 TC FAIL của vòng 1 đã chạy lại **PASS** (01-76, 02-48, 05-41, 05-68, 06-05, 06-18, 06-30, 07-45) và hồi quy toàn bộ:

| Story | TC | PASS | FAIL |
| --- | --- | --- | --- |
| US-PG-01 | 85 | 85 | 0 |
| US-PG-02 | 70 | 70 | 0 |
| US-PG-03 | 108 | 108 | 0 |
| US-PG-04 | 58 | 58 | 0 |
| US-PG-05 | 74 | 74 | 0 |
| US-PG-06 | 38 | 38 | 0 |
| US-PG-07 | 69 | 69 | 0 |
| GATE-PG | 25 | 25 | 0 |

Hồi quy vòng 2 đã chạy: `pg05.sh` đủ (72 + TC chậm 25, 27), `pg06.sh` đủ (37 + 38 tay), `pg07.sh` đủ (67 + 44/63 chép), `gate-pg.sh` đủ (22 tự động + 25, 27 trình duyệt thật + 30 đọc diff) — gồm `--scale gateway=2`, tắt một gateway giữa stream (TC-GATE-28/29), rút mạng SSE (23/24/25), Idempotency-Key gửi đôi (26/27), `down -v` rồi `up` (20). Story 01–04 giữ kết quả vòng 1 (đã PASS), riêng TC-PG01-76 và TC-PG02-48 chạy lại. CI `sprint/2-pg` xanh (xem TC-PG07-39). Lưu ý: TC-PG07-43/44 chấm ở vòng 1, nhánh `ci/sqlc-drift` nay đã xoá (TC-PG07-45 PASS).

## TC
| TC-id | PASS/FAIL | AC | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-GATE-01 | PASS | dòng 1 `go vet ./...` | go vet ./... rc = [0] ; go vet -tags testroutes ./... rc = [0] (+1 dòng ok) |
| TC-GATE-02 | PASS | dòng 1 `golangci-lint run` | golangci-lint run rc = [0] ; golangci-lint run --build-tags testroutes rc = [0] |
| TC-GATE-03 | PASS | dòng 1 `go test -race ./...` (đúng PG.md, **không** tag) | go test -race ./... (PG.md, không tag) rc = [0] ; FAIL/SKIP trong -json = [0] — _sửa script QC: đếm skip mức test (gói không test phát skip mức gói — #10)_ |
| TC-GATE-04 | PASS | dòng 1 bản có tag (AC19, `make test`) | go test -race -tags testroutes ./... rc = [0] ; số test PASS = 357 (≥ 100) (+18 dòng ok) — _sửa script QC: như 03_ |
| TC-GATE-05 | PASS | dòng 2 `sqlc diff` | sqlc diff rc = [0] ; sqlc diff không in gì = [] |
| TC-GATE-06 | PASS | dòng 3 contract | contract không tag rc = [0] ; contract có tag rc = [0] (+2 dòng ok) |
| TC-GATE-07 | PASS | dòng 4 `./internal/httpapi/...` "cursor, Idempotency-Key gửi đôi → một bản ghi, 409 version, ETag" | go test ./internal/httpapi -run 'TestCursor_NoSkipNoDup\|TestCursor_SameTimestamp\|TestCursor_InsertAndDeleteDur ; --- PASS: TestCursor_NoSkipNoDup (+10 dòng ok) — _sửa script QC: #10(2) gói httpapi riêng_ |
| TC-GATE-08 | PASS | dòng 5 sse + auth | go test ./internal/httpapi/sse/... -run 'TestSSE_Headers\|TestSSE_LastEventID_NoLossNoDup\|TestSSE_ReconnectRace ; --- PASS: TestSSE_Headers (+12 dòng ok) |
| TC-GATE-09 | PASS | dòng 6 `pnpm -C frontend build` | pnpm -C frontend build rc = [0] |
| TC-GATE-10 | PASS | dòng 7 `up -d --scale gateway=2` | up -d --scale gateway=2 --wait rc = [0] ; số container gateway = [2] (+2 dòng ok) |
| TC-GATE-11 | PASS | dòng 8 `curl -fsSk https://localhost/api/v1/healthz` "qua Caddy, 200 ở cả hai bản gateway" | curl -fsSk healthz = [{"status":"ok"}] ; số X-Instance-Id khác nhau = [2] (+2 dòng ok) |
| TC-GATE-12 | PASS | dòng 9 `k6 run benchmarks/load/smoke.js` "p95 trong SLO ở SYSTEM_DESIGN §5" | k6 rc = [0] ; số dòng p(95)<300 trong smoke.js = 3 (≥ 3) (+3 dòng ok) — _sửa script QC: #10(1) `paste -sd" " -`_ |
| TC-GATE-13 | PASS | AC14 (biến thể ghi) | k6 -e TEST_ROUTES=1 rc = [0] ; k6 TEST_ROUTES=1 ở bản mặc định phải DỪNG (rc≠0) = [107] (≠ [0]) (+1 dòng ok) |
| TC-GATE-20 | PASS | mục 1 "Xoá sạch volume DB rồi `pnpm dev`: migration chạy từ 00001 lên hết, không cần thao tác tay" | volume postgres đã xoá = [0] ; readyz 200 sau pnpm dev (≤ 300 s) = [200] (+4 dòng ok) — _sửa script QC: #10(1)_ |
| TC-GATE-21 | PASS | mục 2 "Bỏ một biến môi trường bắt buộc: gateway chết ngay lúc khởi động và nói đúng tên biến thiếu" (tiến trình trần) | DATABASE_URL: rc = [1] ; DATABASE_URL: ms tới khi thoát = 26 (≤ 1000) (+19 dòng ok) |
| TC-GATE-22 | PASS | mục 2 (compose) "… không chết lúc đang phục vụ" | gateway thiếu JWT_SECRET_KEY: rc = [1] ; log nêu tên biến = [JWT_SECRET_KEY] (+4 dòng ok) — _sửa script QC: jq bỏ qua dòng "Container …"_ |
| TC-GATE-23 | PASS | mục 3 "Mở một stream SSE thử, rút mạng 10 giây rồi cắm lại … không mất sự kiện nào" (mô phỏng phía client, tự động) | đã nhận id e5 ~ /^[0-9]+-[0-9]+$/ ; đã nhận n 1..5 = [1 2 3 4 5] (+5 dòng ok) — _sửa script QC: #10(1) + `wait` pid subshell + `sse_ns` bỏ "data: "_ |
| TC-GATE-24 | PASS | mục 3 (rút mạng thật) | gateway nhận lại qua Caddy ≤ 40 s sau khi cắm = [200] ; đã nhận trước khi rút: 1 2 = [1 2] (+2 dòng ok) — _sửa script QC: như 23_ |
| TC-GATE-25 | PASS | mục 3c | `sse-browser-cut.mjs` (Chrome thật): 9/9 PASS — nhận đúng 1…30, không trùng, 4 lần nối lại gửi `Last-Event-ID`, không `resync`. |
| TC-GATE-26 | PASS | mục 4 "Gửi đúp … `Idempotency-Key` bằng hai tab" (hai tiến trình song song) | một bản ghi duy nhất = [1] ; hai thân giống hệt (byte) = [same] (+3 dòng ok) |
| TC-GATE-27 | PASS | mục 4b | `idem-two-tabs.mjs` (hai tab thật): 5/5 PASS — 1 bản ghi, hai response 201 giống hệt, một bên `Idempotent-Replayed: true`. |
| TC-GATE-28 | PASS | mục 5 "Tắt một trong hai bản gateway giữa lúc đang có stream: phiên kế tiếp vẫn chạy, không mất đăng nhập" (bản mang stream) | client nhận event: shutdown = [1] ; shutdown reason = [server_shutdown] (+4 dòng ok) — _sửa script QC: như 23 + `--filter id=` (#10(3))_ |
| TC-GATE-29 | PASS | mục 5 (bản không mang stream) | dòng không-200 trong 200 request = 1 (≤ 4) ; 60 dòng cuối 0 lỗi = [0] (+3 dòng ok) — _sửa script QC: như 28_ |
| TC-GATE-30 | PASS | mục 6 | Đọc lại diff từ `4bf829c` đến HEAD: 7 tệp (header `X-EP-Draining` `middleware.go`, `server.go` đặt header lúc tắt, `sse/handler.go` `startFrom`, `depProbeTimeout` 400 ms) — đạt; ứng viên `diff-review.sh` giữ nguyên như vòng 1 và đều đã giải thích (`time.Sleep(drain)` là ngoại lệ hợp lệ theo #12). |
| TC-GATE-99 | PASS | dọn | chế độ cuối = [default] ; 2 gateway healthy = [2] (+1 dòng ok) |

## Bước tay đã làm
- TC-GATE-23/24 (rút mạng 10 s; `docker network disconnect`): PASS bằng `gate-pg.sh`. TC-GATE-25 (Chrome thật offline 10 s, `sse-browser-cut.mjs`): **9/9 PASS** — nhận đúng 1…30, không trùng, 4 lần nối lại gửi `Last-Event-ID`, không `resync`. TC-GATE-26/27 (Idempotency-Key gửi đôi): 2 tiến trình + 2 tab thật (`idem-two-tabs.mjs`) **5/5 PASS** — 1 bản ghi, 2 response 201 giống hệt, một bên `Idempotent-Replayed: true`. TC-GATE-28/29 (tắt một gateway giữa stream / không mang stream): PASS. TC-GATE-20 (`down -v` rồi `up`, migration chạy hết) và `--scale gateway=2`: PASS. TC-GATE-30 (đọc diff):
  - đọc chọn lọc 20 tệp sản xuất, 0 ứng viên chưa giải thích: `time.Sleep(drain)` `server.go:72` (cửa sổ drain khi tắt êm — hợp lệ, nhưng checklist mục 14 cấm `time.Sleep` → đề nghị BA ghi ngoại lệ), `panic(rec)` `middleware.go:63` (chỉ khi `http.ErrAbortHandler` — đúng mục 11), `float64` SSE score `handler.go` (hạn kết nối, không phải điểm), `fmt.Fprintln` `auth/cli.go` (lệnh `token` in ra stdout/stderr — cố ý).
  - xác nhận bằng `file:dòng`: HS256 cố định `jwt.go:105` (`WithValidMethods`), leeway `jwt.go:109`, `jti` `crypto/rand` `jwt.go:4`; `RequireRole` so khớp bằng map `middleware.go:70-81`; Principal chỉ từ context `auth.go:32`; idempotency lock `SetNX` `idempotency.go:152`; `X-Forwarded-For` chỉ tin `TRUSTED_PROXY_CIDRS` `ratelimit.go:139-160`; các mục còn lại (bcrypt, cursor, CORS, ETag, CourseAccessGuard) được kiểm hộp đen ở US-PG-03/04 và `diff-review.sh`. **Hạn chế:** không đọc từng dòng 8,5 nghìn dòng diff.

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
