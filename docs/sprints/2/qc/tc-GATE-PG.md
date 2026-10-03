# QC test case — GATE-PG (cổng nghiệm thu phase PG + "Bạn tự kiểm")
Nguồn: `docs/phases/PG.md` mục "Cổng nghiệm thu" và "Bạn tự kiểm" (nguyên văn, từng dòng → bước đo được) + `US.md` v1.1 US-PG-07 AC19 (bản mở rộng của cổng có hai cấu hình tag) + AC6/AC16 (lệnh cụ thể). Hộp đen cho đến khi chấm; riêng TC-GATE-30 QC **đọc diff** (đúng yêu cầu PG.md dòng cuối "Bạn tự kiểm" — phase 2 mới được đọc mã).
Công cụ: **S** = `scripts/gate-pg.sh` (hàm `tc_gate_NN` ↔ `TC-GATE-NN`; `bash docs/sprints/2/qc/scripts/gate-pg.sh [--list | NN …]`), **M** = `scripts/sse-browser-cut.mjs`, `scripts/idem-two-tabs.mjs` (chạy bằng eval, trình duyệt Chromium thật), **T** = tay, `scripts/diff-review.sh` (gom ứng viên cho TC-GATE-30), `scripts/lib.sh` (dùng chung), `scripts/pg01.sh … pg07.sh` (từng story).
Chạy ở gốc worktree `/Users/kuro/Documents/TA_Agent_v2-s2` (nhánh `sprint/2-pg`) **sau khi 7 story đều có `report-US-PG-0N.md` PASS** (cổng chạy trọn = US-PG-07 AC19). Máy QC: Docker (colima) + `go` + `pnpm` + `k6` + `gh` + `golangci-lint` + `sqlc` + `jq` + `curl` + `openssl`.
Mọi TC có một kết luận PASS/FAIL. Công cụ thiếu (`golangci-lint`, `sqlc`, `k6`) → TC đó FAIL "KHÔNG KIỂM ĐƯỢC" (QC không tự cài, không đoán).
Thứ tự: **A** (01 → 13) rồi **B** (20 → 30) rồi 99 (dọn). B-20 xoá volume Postgres: chạy sau A, vì A-10 dựng stack trên volume cũ; mọi TC sau 20 chạy trên DB mới.

## Bảng A — Cổng nghiệm thu (PG.md)
| TC-id | PG.md | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-GATE-01 | dòng 1 `go vet ./...` | Docker không cần | **S** `gate-pg.sh 01` — `cd backend-go && go vet ./... ; go vet -tags testroutes ./...` | Cả hai rc=0; không in dòng nào (0 cảnh báo) — AC19 đòi cả hai cấu hình biên dịch sạch |
| TC-GATE-02 | dòng 1 `golangci-lint run` | `golangci-lint` có trên máy | **S** `gate-pg.sh 02` — `golangci-lint run` rồi `golangci-lint run --build-tags testroutes` | Cả hai rc=0. Thiếu công cụ → FAIL "KHÔNG KIỂM ĐƯỢC" |
| TC-GATE-03 | dòng 1 `go test -race ./...` (đúng PG.md, **không** tag) | Docker chạy (testcontainers) | **S** `gate-pg.sh 03` — `go test -race -count=1 -timeout 40m -json ./...` | rc=0; số dòng `Action:"fail"` hoặc `"skip"` = 0 |
| TC-GATE-04 | dòng 1 bản có tag (AC19, `make test`) | Docker chạy | **S** `gate-pg.sh 04` — `go test -race -count=1 -tags testroutes -json ./...` | rc=0; ≥ 100 test PASS; **0 skip** (không Docker ⇒ phải fail, không skip — SRS 9.1); mỗi gói của SRS 9.2 (`internal/platform/{config,log,otel,redis,db,blob,outbox}`, `internal/store`, `internal/httpapi`, `internal/httpapi/sse`, `internal/jobs`, `internal/auth`, `internal/contract`, `cmd/gateway`, `cmd/worker`, `db`) ≥ 1 test PASS (bắt trường hợp gói "no test files" lọt qua) |
| TC-GATE-05 | dòng 2 `sqlc diff` | `sqlc` có trên máy | **S** `gate-pg.sh 05` — `cd backend-go && sqlc diff` | rc=0 và **không in gì** |
| TC-GATE-06 | dòng 3 contract | Docker chạy | **S** `gate-pg.sh 06` — `go test -count=1 ./internal/contract/...` (không tag, đúng PG.md) rồi `-tags testroutes` | Cả hai rc=0; đầu ra không có `no tests to run` / `no test files` |
| TC-GATE-07 | dòng 4 `./internal/httpapi/...` "cursor, Idempotency-Key gửi đôi → một bản ghi, 409 version, ETag" | Docker chạy | **S** `gate-pg.sh 07` — `gt ./internal/httpapi/... 'TestCursor_NoSkipNoDup\|TestCursor_SameTimestamp\|TestCursor_InsertAndDeleteDuringScan\|TestIdempotency_DoubleSend_OneRecord\|TestIdempotency_ReplayIdentical\|TestIdempotency_Concurrent50\|TestOptimisticLock_Stale409\|TestOptimisticLock_Concurrent20\|TestETag_NotModified\|TestETag_ChangesAfterUpdate'` rồi `go test -race ./internal/httpapi/...` | Mỗi tên có `--- PASS:` (không skip, không "no tests"); rc=0. Bốn thứ PG.md liệt kê (cursor, gửi đôi, 409 version, ETag) mỗi thứ ≥ 1 test PASS có tên |
| TC-GATE-08 | dòng 5 sse + auth | Docker chạy | **S** `gate-pg.sh 08` — `gt ./internal/httpapi/sse/... 'TestSSE_Headers\|TestSSE_LastEventID_NoLossNoDup\|TestSSE_ReconnectRace\|TestSSE_CrossInstance\|TestSSE_GatewayShutdownFailover\|TestSSE_MaxTwoPerUser'`; `gt ./internal/auth/... 'TestJWT_Claims\|TestVerify_Table\|TestRBAC_Matrix\|TestCourseAccessGuard_DefaultDenyAll\|TestBcrypt_Format'`; rồi `go test -race ./internal/httpapi/sse/... ./internal/auth/...` | Mỗi tên `--- PASS:`; rc=0 |
| TC-GATE-09 | dòng 6 `pnpm -C frontend build` | `pnpm install` đã chạy | **S** `gate-pg.sh 09` — `pnpm -C frontend build` | rc=0 (PG không đụng UI nhưng `NEXT_PUBLIC_API_URL` đổi sang `https://localhost` không được làm vỡ build) |
| TC-GATE-10 | dòng 7 `up -d --scale gateway=2` | Docker chạy; image mặc định | **S** `gate-pg.sh 10` — `$C up -d --build --force-recreate --scale gateway=2 --wait` | rc=0; đúng 2 container `gateway`; `mode_now`=`default` (route thử 404); `migrate exited 0` |
| TC-GATE-11 | dòng 8 `curl -fsSk https://localhost/api/v1/healthz` "qua Caddy, 200 ở cả hai bản gateway" | Sau TC-GATE-10 | **S** `gate-pg.sh 11` — `curl -fsSk $GW/api/v1/healthz`; 40 lần đọc `X-Instance-Id` + mã | Thân `{"status":"ok"}`; 40/40 là 200; đúng 2 giá trị `X-Instance-Id`, mỗi giá trị đếm > 0 (cả hai bản đều trả 200) |
| TC-GATE-12 | dòng 9 `k6 run benchmarks/load/smoke.js` "p95 trong SLO ở SYSTEM_DESIGN §5" | Sau TC-GATE-11, `k6` có | **S** `gate-pg.sh 12` — `k6 run -e BASE=$GW -e TOKEN=… benchmarks/load/smoke.js` | rc=0; mọi dòng ngưỡng ✓ (không ✗); trong `smoke.js` ngưỡng p(95) chỉ gồm `<300` (≥ 3 lần) và `<500` — **không nới**; `RestartCount`+`StartedAt` của mọi gateway/worker **không đổi** trong lúc đo |
| TC-GATE-13 | AC14 (biến thể ghi) | `tmode` rồi `dmode` | **S** `gate-pg.sh 13` — `k6 run -e TEST_ROUTES=1 …` ở chế độ test; rồi cùng lệnh ở chế độ mặc định | Chế độ test: rc=0, p95 ghi ≤ 500 ms. Chế độ mặc định: rc≠0 và thông báo nêu rõ route thử vắng (`_test`/`testroutes`/`TEST_ROUTES`/404) |

## Bảng B — "Bạn tự kiểm" (PG.md)
| TC-id | PG.md | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-GATE-20 | mục 1 "Xoá sạch volume DB rồi `pnpm dev`: migration chạy từ 00001 lên hết, không cần thao tác tay" | Sau A | **S** `gate-pg.sh 20` — `pnpm dev:down; docker volume rm edupilot_postgres_data; pnpm dev` (nền, không stdin) | Volume mất rồi dựng lại; `readyz` 200 ≤ 300 s; `migrate exited 0`; `goose_db_version` ≥ 2 dòng đã áp; 5 bảng nền + `goose_db_version` (không bảng khác); đầu ra `pnpm dev` không có lời nhắc thao tác tay (`[y/N]`, "Press", "please run"…); ghi thời gian tới readyz |
| TC-GATE-21 | mục 2 "Bỏ một biến môi trường bắt buộc: gateway chết ngay lúc khởi động và nói đúng tên biến thiếu" (tiến trình trần) | Binary mặc định build | **S** `gate-pg.sh 21` — với từng biến trong 7 biến (`DATABASE_URL REDIS_URL JWT_SECRET_KEY BLOB_ENDPOINT BLOB_BUCKET BLOB_ACCESS_KEY BLOB_SECRET_KEY`), `env -i` + 6 biến còn lại, `gateway serve` | Mỗi biến: rc=1; thoát ≤ 1000 ms (đo `now_ms`, không dùng `date +%N` vì macOS); dòng JSON `missing` chứa đúng tên biến |
| TC-GATE-22 | mục 2 (compose) "… không chết lúc đang phục vụ" | Stack chạy, 2 gateway | **S** `gate-pg.sh 22` — sao lưu `.env.local`, xoá `JWT_SECRET_KEY`, `$C run --rm --no-deps --name qc-gw-noenv gateway serve`, khôi phục `.env.local` | rc=1; log `missing` = `["JWT_SECRET_KEY"]` (nêu đúng tên); container thử không còn chạy (không mở cổng); **hai gateway đang phục vụ không đổi `RestartCount`/`StartedAt`** và `readyz` vẫn 200; `.env.local` khôi phục `cmp` y hệt |
| TC-GATE-23 | mục 3 "Mở một stream SSE thử, rút mạng 10 giây rồi cắm lại … không mất sự kiện nào" (mô phỏng phía client, tự động) | Chế độ test | **S** `gate-pg.sh 23` — curl stream; phát n=1..5; **huỷ curl** (rút mạng); phát n=6..12 (≥ 10 s); mở lại với `Last-Event-ID: <id của n=5>` cùng token | Trước: nhận `1 2 3 4 5`. Sau nối lại: nhận **đúng** `6 7 8 9 10 11 12` theo thứ tự; không nhận lại 5; không trùng; không `event: resync`; byte đầu `retry: 3000` |
| TC-GATE-24 | mục 3 (rút mạng thật) | Chế độ test; network compose `edupilot_default` | **S** `gate-pg.sh 24` — stream qua Caddy; phát n=1,2; `docker network disconnect edupilot_default <caddy>` 10 s rồi `docker network connect --alias caddy …`; phát n=3,4; nối lại với `Last-Event-ID` | `readyz` qua Caddy 200 ≤ 40 s sau khi cắm; trước: `1 2`; sau nối lại: `3 4` (không lặp 2); gateway/worker `RestartCount` = 0. **Nếu `network connect` không phục hồi cổng đã công bố** thì TC này "KHÔNG KIỂM ĐƯỢC" và TC-GATE-25 là bằng chứng chính thức (xem Điểm khó kiểm 3) |
| TC-GATE-25 | mục 3 "trình duyệt tự nối lại bằng Last-Event-ID" (Chromium thật) | Chế độ test; `browser` của eval | **M** `sse-browser-cut.mjs` — fetch+ReadableStream trong tab; phát n=1..30 mỗi 1 s từ `curl` ngoài; ở n=8 bật OFFLINE (CDP) + abort, ở n=18 tắt offline (= 10 s) | `table()` in `ok=true`: K1 stream 200; K2 30 POST 2xx; K3 nhận đúng `1..30` thứ tự; K4 không trùng; K5 ≥ 1 lần nối lại; K6 `Last-Event-ID` gửi lúc nối lại = id của n=7/8; K7 ≥ 1 lỗi fetch (mạng đứt thật); K8 không `resync`; K9 mọi lần nối lại HTTP 200. **Tay** (hàm script `manual`) vì cần `browser` |
| TC-GATE-26 | mục 4 "Gửi đúp … `Idempotency-Key` bằng hai tab" (hai tiến trình song song) | Chế độ test | **S** `gate-pg.sh 26` — hai `curl -X POST` song song cùng khoá + cùng thân; nếu một bên 409 thì gửi lại sau `Retry-After` | Đúng 1 dòng `_test_items`; hai thân giống **từng byte** (`cmp`); hai status giống nhau; đúng **một** response có `Idempotent-Replayed: true`; hai `Content-Type` giống nhau |
| TC-GATE-27 | mục 4 (hai tab trình duyệt thật) | Chế độ test; `browser` | **M** `idem-two-tabs.mjs` — mở 2 tab, `Promise.all` gửi cùng khoá bằng `fetch` | `ok=true`: I1 1 bản ghi; I2 hai status giống nhau và 2xx; I3 hai thân giống hệt; I4 Content-Type giống; I5 đúng một `Idempotent-Replayed: true` **đọc được từ trình duyệt** (chứng minh `Access-Control-Expose-Headers`). **Tay** (cần `browser`) |
| TC-GATE-28 | mục 5 "Tắt một trong hai bản gateway giữa lúc đang có stream: phiên kế tiếp vẫn chạy, không mất đăng nhập" (bản mang stream) | Chế độ test, 2 gateway | **S** `gate-pg.sh 28` — mở stream, xác định bản mang stream bằng `X-Instance-Id`, phát n=1..3, `docker stop` đúng container đó, phát n=4..6, nối lại với `Last-Event-ID` + **cùng token** | Client nhận `event: shutdown` `{"reason":"server_shutdown"}`; `docker stop` ≤ 30 s (không bị SIGKILL); `GET /api/v1/jobs/<UX>` cùng token → **404, không 401**; nối lại nhận **đúng `4 5 6`**; stack khôi phục 2 gateway |
| TC-GATE-29 | mục 5 (bản không mang stream) | Chế độ test, 2 gateway | **S** `gate-pg.sh 29` — stream mở ở bản A; chạy 200 GET healthz (10/s); ở giây 5 `docker stop` bản **B**; phát n=99 | ≤ 4 dòng không-200 trong 200, **0 lỗi trong 60 dòng cuối**; stream cũ vẫn sống và nhận n=99; **không** có `event: shutdown` ở stream; khôi phục 2 gateway |
| TC-GATE-30 | mục 6 "Đọc toàn bộ diff của `internal/auth` và `httpapi/`" | Sau mọi TC khác | **T** `bash scripts/diff-review.sh` (gom ứng viên) + đọc diff `git diff $(git merge-base HEAD origin/main)..HEAD -- backend-go/internal/auth backend-go/internal/httpapi` theo checklist dưới | Mỗi mục checklist = "đạt" kèm `file:dòng` hoặc `BUG-n`; **0 ứng viên chưa giải thích** từ `diff-review.sh` |
| TC-GATE-99 | dọn | cuối cùng | **S** `gate-pg.sh 99` — `dmode` | Chế độ `default`; 2 gateway healthy; không container nào `exited` ngoài `migrate` |

### Bước tay TC-GATE-25 (Chromium thật qua eval)
1. `ensure_mode test` (bằng `bash -c 'source docs/sprints/2/qc/scripts/lib.sh; ensure_mode test'`); lấy token: `tok STUDENT 00000000-0000-7000-8000-000000000001 --ttl 10m`.
2. Trong eval JS: `const m = await import('/Users/kuro/Documents/TA_Agent_v2-s2/docs/sprints/2/qc/scripts/sse-browser-cut.mjs'); const rows = await m.default(browser, { base: 'https://localhost', token: '<token>', cwd: '/Users/kuro/Documents/TA_Agent_v2-s2' }); console.log(m.table(rows));`
3. Kỳ vọng K1–K9 đều PASS (≈ 40 s). Lưu đầu ra vào `qc/report-GATE-PG.md`.
4. Nếu `page.createCDPSession` hoặc `Security.setIgnoreCertificateErrors` không có ở bản trình duyệt này → ghi "KHÔNG KIỂM ĐƯỢC" kèm thông báo lỗi; bằng chứng thay thế là TC-GATE-23.

### Bước tay TC-GATE-27 (hai tab)
1. Chế độ test. Token như trên.
2. `const m = await import('…/idem-two-tabs.mjs'); console.log(m.table(await m.default(browser, { base: 'https://localhost', token: '<token>', cwd: '/Users/kuro/Documents/TA_Agent_v2-s2' })));`
3. Kỳ vọng I1–I5 PASS (`ok=true`).

### Checklist TC-GATE-30 (đọc diff `internal/auth`, `internal/httpapi` — phần mọi phase sau kế thừa)
Mỗi dòng: đạt / BUG-n, kèm `file:dòng`.
1. Danh tính chỉ từ `Principal` trong `context.Context`; không biến toàn cục giữ trạng thái người dùng; không đọc `user_id`/`student_code` từ body/query (luật 2, US-PG-04 AC12).
2. JWT: thuật toán cố định HS256 (không `none`, không đọc `alg` từ header để chọn khoá), kiểm `iss`/`aud`/`nbf`/`exp` + leeway 5 s, `jti` từ `crypto/rand`; không `ParseUnverified`.
3. 401/403/404: mọi lỗi đi qua **một** hàm viết lỗi; thân không có chi tiết nội bộ; `WWW-Authenticate` đúng.
4. `CourseAccessGuard`: mặc định từ chối tất cả (kể cả ADMIN); không cache kết quả giữa các request; resolver lỗi → 503, không cho qua.
5. `RequireRole`: so khớp chính xác, không `strings.Contains`; claim thắng DB.
6. bcrypt: `CompareHashAndPassword`, không so chuỗi; cost từ cấu hình; > 72 byte → `ErrPasswordTooLong`.
7. Không ghi log token/mật khẩu/hash/thân request/`Idempotency-Key`; `jti` rút gọn ≤ 8 ký tự.
8. Cursor: so sánh hàng `(created_at, id)`, không `OFFSET`; giải mã chặt (v=1, `t` số, `i` uuid).
9. Idempotency: khoá logic `(user_id, endpoint, key)`; 5xx/429 không lưu; Redis chết → 503 fail-closed; ghi Redis + bảng; `lock` NX EX 30.
10. Rate limit: fail-open có log hạn chế 1 lần/10 s; `X-Forwarded-For` chỉ tin khi nguồn thuộc `TRUSTED_PROXY_CIDRS`; health được miễn.
11. Recover: không nuốt `http.ErrAbortHandler`; log stack + `trace_id`; thân trả về chung.
12. Mọi truy vấn có `ctx`; không `context.Background()` trong handler; lỗi bọc `%w`; handler mỏng, logic ở service.
13. CORS: không `*`, `Vary: Origin`, preflight 204; ETag `W/"v<version>"`; `If-None-Match` trên POST/PUT bị bỏ qua.
14. Mã sản xuất không ghi đĩa cục bộ (`os.Create`…), không `time.Sleep`, không `fmt.Print`.

## Nhánh lỗi (mỗi tình huống có TC)
| Tình huống | TC |
| --- | --- |
| Thiếu biến bắt buộc (PG.md "Bạn tự kiểm" 2) | TC-GATE-21 (7 biến), TC-GATE-22 (compose) |
| Cổng nghiệm thu đỏ vì thiếu công cụ | TC-GATE-02, 05, 12, 13 (FAIL "KHÔNG KIỂM ĐƯỢC") |
| Test bị skip thay vì fail khi thiếu Docker | TC-GATE-03, 04 (0 skip) |
| Rút mạng 10 s giữa SSE | TC-GATE-23, 24, 25 |
| Gửi đúp Idempotency-Key | TC-GATE-26, 27 |
| Tắt một gateway giữa stream (bản mang stream / bản không mang) | TC-GATE-28, 29 |
| k6 ở bản mặc định không có route thử | TC-GATE-13 |

## Phân quyền
Cổng không có endpoint phân quyền riêng; phân quyền được chấm trong `pg03.sh`, `pg04.sh`, `pg05.sh`. Cổng kiểm thêm: TC-GATE-28 và 29 dùng **cùng một token** ở hai bản gateway (không 401), TC-GATE-30 mục 1–5 đọc diff phần phân quyền, TC-GATE-10 chứng minh bản mặc định không có route thử.

## Điểm khó kiểm
1. **PG.md và AC19 khác nhau**: PG.md viết lệnh không tag; US-PG-07 AC19 đòi hai cấu hình (có và không tag) cho `go vet`, `golangci-lint`, và `-tags testroutes` cho `go test`. TC chạy **cả hai** (TC-GATE-01…04, 06); bản không tag của `go test -race ./...` (TC-GATE-03) chỉ chấm rc=0 và 0 fail/skip.
2. **"Rút mạng" trên localhost**: tắt Wi-Fi không cắt `localhost`; nên dùng ba mức: (23) client huỷ + nối lại (xác định, tự động) — bằng chứng chuẩn; (24) `docker network disconnect` Caddy 10 s — gần thật hơn nhưng phụ thuộc Docker/colima phục hồi cổng công bố; (25) Chromium offline qua CDP + header `Last-Event-ID` thật + CORS preflight thật. TC-GATE-25 là bằng chứng "trình duyệt tự nối lại"; TC-GATE-23 là bằng chứng "không mất sự kiện" khi 25 không chạy được. Ghi rõ TC nào đã chạy.
3. Không có UI ở PG nên "trình duyệt tự nối lại" = mã client trong `sse-browser-cut.mjs` (fetch + ReadableStream, SRS mục 7); `EventSource` gốc không gửi được `Authorization` nên không dùng.
4. **TC-GATE-20** xoá dữ liệu Postgres (volume `edupilot_postgres_data`; tên volume theo project `edupilot`). Chạy trên máy QC/worktree này, không phải máy chủ dự án.
5. `pnpm dev` có thể chạy tiếp ở nền sau khi stack lên; script chỉ `kill` PID nó tạo. Nếu `pnpm dev` ở máy này là tiến trình chạy mãi → `readyz` 200 mới là điều kiện thành công.
6. `go test -json` ghi `Action:"skip"` cho test bị bỏ qua (kể cả `t.Skip` do thiếu Docker): TC-GATE-04 coi skip là FAIL (SRS 9.1: "Không Docker thì test **fail**, không skip, trừ `-short`").
7. Ngưỡng ≥ 100 test PASS ở TC-GATE-04 là sàn QC tự đặt vì spec không nêu tổng số test (tổng AC 105, mỗi AC ≥ 1 test Go) — nếu số thực tế thấp hơn thì đó là phát hiện, không phải lỗi script.
8. Số đo k6 phụ thuộc máy (colima 4 CPU): không đạt → FAIL kèm số thực; không nới SLO (plan sprint 2, Q14).
9. `diff-review.sh` dùng merge-base với `origin/main` làm mốc; nếu dev rebase thì đặt `BASE=<sha>`.

## Câu hỏi cho PM
- **Q-QC-GATE-1** — đã trả lời (PM chốt, góp ý #2 ACCEPTED 01/10): TC-GATE-23 hoặc TC-GATE-25 PASS là đủ cho dòng "rút mạng 10 giây"; TC-GATE-24 bổ trợ, nếu "không kiểm được" vì colima thì ghi rõ. TC không đổi.
- **Q-QC-GATE-2** — đã trả lời (PM chốt, góp ý #2): giữ sàn ≥ 100 test PASS, 0 skip, mỗi gói SRS 9.2 ≥ 1 test. TC-GATE-04 không đổi.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-02 — viết lần đầu theo `PG.md` + `US.md` v1.1 (đã gồm góp ý #1: route thử = build tag `testroutes`, `tmode`/`dmode`/`CT`, AC20). Không có TC cũ để sửa.
- 2026-10-02 — spec v1.2 (commit 02a4435) + góp ý #2 (PM ACCEPTED 01/10; QC questions #Q-QC-GATE-1, #Q-QC-GATE-2): không TC nào đổi nội dung — PM chốt đúng cách QC đã chọn (TC-GATE-23/25 đủ cho "rút mạng 10 s"; sàn ≥ 100 test PASS giữ). Ghi nhận để TC-GATE-24 "không kiểm được" không làm FAIL dòng "rút mạng" khi 23 hoặc 25 PASS.

Tổng: 25 TC (22 tự động bằng `gate-pg.sh`, 3 tay: TC-GATE-25, 27, 30 — ba TC này có script hỗ trợ; TC-GATE-24 bán tự động).
