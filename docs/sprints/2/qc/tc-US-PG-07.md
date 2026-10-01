# QC test case — US-PG-07 (hạ tầng chạy: compose, Caddy, PgBouncer, image, CI, k6)
Nguồn: `docs/specs/FEAT-pg-foundation/US.md` v1.2 (US-PG-07 AC1…AC20, gồm dòng "Làm rõ (QC questions #…)") + `SRS.md` mục 3.4, 4.7, 8.1–8.6, 9.3; `QUESTIONS.md` Q11–Q14; `docs/sprints/2/qc/questions.md` (trả lời BA 02/10); `docs/sprints/2/proposals.md` góp ý #2 (PM ACCEPTED 01/10). Hộp đen (không đọc mã của dev). Công cụ: **S** = `scripts/pg07.sh` (hàm `tc_pg07_MM` ↔ `TC-PG07-MM`), **T** = tay. Chạy `bash docs/sprints/2/qc/scripts/pg07.sh [MM …|--list]` ở gốc worktree sau `pnpm dev`.

Tiền điều kiện chung của story: colima chạy (≥ 4 CPU, 8 GB); `.env.local` đầy đủ; stack dựng bằng `$C up -d --scale gateway=2 --wait`; **trạng thái chuẩn = chế độ default** (image `edupilot-gateway` / `edupilot-worker`, không route thử, 2 bản gateway). Mọi TC đổi chế độ, tắt service hoặc sửa `.env.local` đều khôi phục trạng thái chuẩn **ngay sau lệnh đo, trước khi chấm**. Công cụ cần có trên máy QC: `docker compose` v2 (có `--format json`), `jq`, `curl`, `openssl`, `nc`, `gh` (đã `gh auth login`), `k6`, `golangci-lint`, `go`. Chứng cứ ghi ở `$QC_OUT/`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PG07-01 | AC1 | – | **S** `pg07.sh 01` — `$C config --services \| sort \| paste -sd' ' -` | Đúng chuỗi `caddy frontend gateway mailpit migrate minio pgbouncer postgres redis worker` (10 tên) |
| TC-PG07-02 | AC1 | – | **S** `02` — `$C config --services \| grep -ciE 'python\|docling'`; `grep -ci docling docker-compose.local.yml` | `0`; `0` |
| TC-PG07-03 | AC2 | Máy sạch: script tự `$C down --remove-orphans` (giữ volume) | **S** `03` — `$C up -d --scale gateway=2 --wait`, đo thời gian | `rc=0`; thời gian ghi ra log `$QC_OUT/tc-PG07-03.log`; `readyz` 200 trong ≤ 120 s |
| TC-PG07-04 | AC2 | Sau TC-03 | **S** `04` — `$C ps -a --format '{{.Service}} {{.State}} {{.Health}} {{.ExitCode}}' \| sort` | Đúng **2** dòng `gateway running healthy 0`; mỗi dòng `postgres/redis/minio/mailpit/pgbouncer/caddy/worker/frontend running healthy 0` = 1; `migrate` → `exited 0` |
| TC-PG07-05 | AC2 | Sau TC-03 | **S** `05` — `docker inspect -f '{{.State.StartedAt}}'` của mọi gateway + worker so với `{{.State.FinishedAt}}` của migrate (chuẩn hoá về chuỗi số) | Mọi `StartedAt` > `FinishedAt` của migrate |
| TC-PG07-06 | AC2 | – | **S** `06` — `$C config --format json \| jq .services.<s>.depends_on.<d>.condition` | gateway.migrate = worker.migrate = `service_completed_successfully`; gateway.pgbouncer = gateway.redis = `service_healthy` |
| TC-PG07-07 | AC3 | – | **S** `07` — `curl -fsSk $GW/api/v1/healthz` | Mã `200`; `.status` = `ok` |
| TC-PG07-08 | AC3 | frontend healthy | **S** `08` — `curl -sk $GW/ \| grep -c '<html'` | ≥ `1`; `content-type` khớp `text/html` |
| TC-PG07-09 | AC3 | – | **S** `09` — `curl -sI http://localhost/` | Dòng đầu khớp `HTTP/… (301\|308)`; `location` = `https://localhost/` |
| TC-PG07-10 | AC3 | – | **S** `10` — `curl -sk -D-` (GET) **và** `curl -skI` (HEAD) tới `/api/v1/healthz` | GET: đúng `2` header `strict-transport-security` + `x-content-type-options: nosniff`, HSTS có `max-age=31536000`; HEAD: cũng `2` (mã HEAD được in ra để đối chiếu) |
| TC-PG07-11 | AC3 | – | **S** `11` — `curl -sk -D- -o /dev/null -H 'Accept-Encoding: gzip, zstd' $GW/` | `content-encoding` ∈ {`zstd`,`gzip`} |
| TC-PG07-12 | AC3 | Token STUDENT | **S** `12` — cùng `Accept-Encoding: gzip, zstd` tới `/api/v1/healthz` và `/api/v1/events` | JSON nhỏ: `0` header `content-encoding`; SSE: `content-type` khớp `text/event-stream` và `0` header `content-encoding` |
| TC-PG07-13 | AC3 / SRS 8.2 | – | **S** `13` — đếm header `^server:` ở `/api/v1/healthz` và `/` | `0` và `0` (chỉ thị `-Server` của Caddyfile có hiệu lực) |
| TC-PG07-14 | AC4 | Caddy chạy | **S** `14` — `openssl s_client -connect localhost:443 -servername localhost \| openssl x509 -noout -issuer -subject -dates -text` | ≥ 1 dòng `Caddy Local Authority`; ≥ 1 `DNS:localhost`; có dòng `notAfter=` |
| TC-PG07-15 | AC5 | `--scale gateway=2` | **S** `15` — 40 lần `curl -sk -D- -o /dev/null $GW/api/v1/healthz`, gom mã + `X-Instance-Id` | `40` lần `200`, `0` lần khác 200; đúng **2** giá trị `X-Instance-Id`; giá trị ít nhất xuất hiện ≥ 1 lần |
| TC-PG07-16 | AC5 | Sau TC-15 | **S** `16` — 12 lần đọc `X-Instance-Id`, so với `docker inspect -f '{{.Config.Hostname}} {{.Name}} {{.Id}}'` của `$C ps -q gateway` | ≥ 2 container; **mọi** `X-Instance-Id` khớp hostname/tên/id của một container gateway thật |
| TC-PG07-17 | AC5 | Token ký **một** lần | **S** `17` — 20 lần `GET /api/v1/jobs/<UX>` cùng token qua Caddy | `20` lần `404`; `0` lần `401` |
| TC-PG07-18 | AC6 (nhánh lỗi) | 2 bản gateway; ~2 phút | **S** `18` — ×3 vòng: 200 `GET /api/v1/healthz` nhịp 0,1 s; giây 5 `docker stop $($C ps -q gateway \| head -1)`; sau mỗi vòng `$C up -d --scale gateway=2 --wait gateway` | "Lỗi" = mọi response không-200 **hoặc** kết nối `000` (US.md 07-AC6 v1.2). Mỗi vòng: số dòng không-`200` ≤ **4** (2 % của 200) **và** 60 dòng cuối có **0** dòng không-200. TC PASS chỉ khi cả **3** vòng đạt |
| TC-PG07-19 | AC6 | Token ký trước khi tắt | **S** `19` — `jobs/<UX>` trước khi tắt → `docker stop` 1 bản → chờ 7 s → gọi lại **cùng token** → khôi phục `--scale gateway=2` | Trước: `404`; sau: `404` (không bao giờ `401` — không mất đăng nhập) |
| TC-PG07-20 | AC6 | – | **S** `20` — `docker stop` 1 bản rồi `$C up -d --scale gateway=2 --wait gateway` | Lại đúng **2** dòng `gateway running healthy 0` |
| TC-PG07-21 | AC7 / SRS 8.3 | – | **S** `21` — `$C exec -T postgres psql "$PGB" -Atc 'SHOW CONFIG'` | `pool_mode\|transaction`, `listen_port\|6432`, `max_client_conn\|200`, `default_pool_size\|20`, `reserve_pool_size\|5`, `reserve_pool_timeout\|3`, `query_wait_timeout\|30`, `server_lifetime\|1800`, `server_idle_timeout\|600`, `auth_type\|scram-sha-256`, `max_prepared_statements\|0`, `ignore_startup_parameters` chứa `extra_float_digits` |
| TC-PG07-22 | AC7 / AC17 | – | **S** `22` — `$C ps -a --format '{{.Service}} {{.Ports}}'` hàng pgbouncer; `nc -z -w 2 127.0.0.1 5433` (đối chứng dương); `nc -z -w 2 127.0.0.1 6432` | `0` dòng có `->`; 5433 mở được; **6432 thất bại** |
| TC-PG07-23 | AC7 | `PGBOUNCER_STATS_*` trong `.env.local` | **S** `23` — `$C exec -T postgres psql "$PGB" -Atc 'SHOW VERSION'` | rc = 0; chuỗi trả về chứa `pgbouncer` |
| TC-PG07-24 | AC8 | – | **S** `24` — `$C logs --no-log-prefix <svc> \| jq -R 'fromjson? \| select(.msg=="config loaded").db_via' \| sort -u` | gateway = `pgbouncer`; worker = `pgbouncer` (đúng một giá trị mỗi bên) |
| TC-PG07-25 | AC8 | Gateway đã phục vụ ≥ 1 request chạm DB | **S** `25` — `psql "$PGB" -Atc 'SHOW CLIENTS'` | ≥ **3** dòng client `edupilot`; **0** dòng có `migrate` |
| TC-PG07-26 | AC8 | – | **S** `26` — `$C config --format json` env của `migrate`; rồi `$C up -d --force-recreate migrate` + 12 lần lấy mẫu `SHOW CLIENTS` | env chứa `postgres:5432`, **không** chứa `pgbouncer:6432`; `0`/12 lần thấy client migrate; migrate vẫn `exited 0` |
| TC-PG07-27 | AC9 | Docker cho testcontainers | **S** `27` — `gt ./internal/platform/db 'TestPgBouncer_TransactionMode\|TestPgBouncer_NoPreparedStatements\|TestVector_ThroughPgBouncer'` | rc = 0 và có đủ **3** dòng `--- PASS:` (không SKIP, không "no tests to run") |
| TC-PG07-28 | AC9 (hộp đen) | Chế độ default; ~2 phút | **S** `28` — 20 vòng × 50 request song song `GET /api/v1/jobs/<UX>` (có token ⇒ có truy vấn DB qua PgBouncer), `rl_reset` giữa các vòng | `1000` mã `404`; `$C logs --since 10m gateway \| grep -ciE 'prepared statement\|bind message\|unnamed prepared'` = `0` |
| TC-PG07-29 | AC10 | – | **S** `29` — `docker build -q --target gateway\|worker -t edupilot-<t> backend-go`, đo giây | rc = 0 cả hai; cùng một `backend-go/Dockerfile` (≥ 2 dòng `FROM`); thời gian build in ra báo cáo |
| TC-PG07-30 | AC10 | Sau TC-29 | **S** `30` — `docker image inspect -f '{{.Size}}'` | `edupilot-gateway` và `edupilot-worker` mỗi cái ≤ **39.999.999** byte (< 40.000.000) |
| TC-PG07-31 | AC10 | Sau TC-29 | **S** `31` — `docker image inspect -f '{{.Config.User}}'` | Khớp `^(nonroot\|65532)(:(nonroot\|65532))?$` cho cả hai image |
| TC-PG07-32 | AC10 | Sau TC-29 | **S** `32` — `docker run --rm --entrypoint sh edupilot-<t> -c true` và `--entrypoint /bin/sh` | Cả 4 lệnh **thất bại** (rc ≠ 0 — không có shell trong image) |
| TC-PG07-33 | AC10 / SRS 8.4 | – | **S** `33` — `$C config --format json \| jq` các khoá `.services.migrate.image`, `.entrypoint`, `.command` | `image` = `edupilot-gateway`; `entrypoint` = `["/gateway","migrate"]`; `command` = `["up"]` (đúng ba giá trị, không chấp nhận dạng khác) |
| TC-PG07-34 | AC11a | Stack chạy | **S** `34` — `docker inspect -f '{{.HostConfig.ReadonlyRootfs}}' $($C ps -q gateway worker)` | ≥ 3 container; **0** container khác `true` |
| TC-PG07-35 | AC11b | – | **S** `35` — `grep -rnE 'os\.(Create\|CreateTemp\|WriteFile\|OpenFile\|Mkdir\|MkdirAll\|MkdirTemp)\|ioutil\.(WriteFile\|TempFile)' internal cmd --include=*.go \| grep -v _test.go` | `0` dòng |
| TC-PG07-36 | AC11c | `golangci-lint` đã cài | **S** `36` — `cd backend-go && golangci-lint run` | rc = 0; `0` dòng `*.go:<số>` báo lỗi |
| TC-PG07-37 | AC11b (động) | Stack chạy | **S** `37` — 60 request `healthz` + 1 request `jobs` rồi `docker diff <container>` cho mọi gateway + worker | `0` dòng thay đổi hệ thống file ở mỗi container |
| TC-PG07-38 | AC12 | `gh auth` xong; `git fetch origin sprint/2-pg` | **S** `38` — `gh run list --workflow ci.yml --branch sprint/2-pg --limit 1 --json databaseId,headSha,conclusion` | `headSha` = `git rev-parse origin/sprint/2-pg`; `conclusion` = `success` |
| TC-PG07-39 | AC12 | Sau TC-38 | **S** `39` — `gh run view <ID> --json jobs --jq '.jobs[].steps[].name' \| grep -ciE 'sqlc\|race\|vet\|golangci'` | ≥ **6** |
| TC-PG07-40 | AC12 | Sau TC-38 | **S** `40` — đếm trong danh sách bước: `vet`, `golangci`, `sqlc` | `vet` ≥ **2**, `golangci` ≥ **2** (hai cấu hình tag), `sqlc` ≥ 1 |
| TC-PG07-41 | AC12 | Sau TC-38 | **S** `41` — `gh run view <ID> --json jobs --jq '.jobs[] \| "\(.name)\|\(.conclusion)"'` | Có ≥ 1 job tên `Frontend`; **0** job `Frontend` khác `success`; **0** job `Go` khác `success` |
| TC-PG07-42 | AC12 | – | **S** `42` — `grep -c testroutes .github/workflows/ci.yml`; `grep -nE 'legacy\|secrets\.' .github/workflows/ci.yml` | ≥ `3`; **không in gì** (`0` dòng) |
| TC-PG07-43 | AC13 (nhánh lỗi) | Nhánh `ci/sqlc-drift` **còn tồn tại**; `RUN_ID=<id>` nếu biết | **S** `RUN_ID=<id> pg07.sh 43` — `gh run view <ID> --json conclusion,headSha,headBranch,jobs` | `conclusion` = `failure`; `headBranch` = `ci/sqlc-drift`; job `Go` = `failure`, job `Frontend` = `success`; ≥ 1 bước thất bại có tên chứa `sqlc` |
| TC-PG07-44 | AC13 (xác thực) | Như TC-43 | **S** `44` — `gh run view <ID> --json headSha` so `git ls-remote --heads origin ci/sqlc-drift`; `git diff --name-only origin/sprint/2-pg...<sha>` | `headSha` = SHA đầu nhánh trên origin; ≥ 1 tệp `internal/store/queries/*.sql`; **0** tệp đổi ngoài thư mục đó |
| TC-PG07-45 | AC13 | Chỉ chạy **sau** khi TC-43, TC-44 đã chấm xong | **T** `45` (script in `manual`) — xem "Bước tay TC-PG07-45" | `git ls-remote --heads origin ci/sqlc-drift` **không in gì** |
| TC-PG07-46 | AC14 | `k6` đã cài; chế độ default | **S** `46` — `k6 run -e BASE=https://localhost -e TOKEN="$(tok STUDENT $U1)" benchmarks/load/smoke.js` | `rc=0`; **0** dòng `✗`; ≥ 3 dòng `✓`; ghi vào báo cáo p95 của `healthz`/`readyz`/`jobs404` (≤ 300 ms), `http_req_failed` < 0,005, `checks` > 0,99 |
| TC-PG07-47 | AC14 | – | **S** `47` — `grep -c 'p(95)<300' benchmarks/load/smoke.js`; `grep -oE 'p\(95\)<[0-9]+' \| sort -u` | ≥ `3`; tập ngưỡng đúng bằng `p(95)<300,p(95)<500` (**không nới SLO**); có `rate<0.005` và `rate>0.99` |
| TC-PG07-48 | AC14 | `tmode` (image `*-test`) | **S** `48` — `k6 run … -e TEST_ROUTES=1 …` rồi `ensure_mode default` | `rc=0`; **0** dòng `✗`; có ngưỡng `p(95)<500` của kịch bản ghi; p95 ghi vào báo cáo |
| TC-PG07-49 | AC14 | Chế độ default | **S** `49` — `k6 run … -e TEST_ROUTES=1 …` trên stack mặc định | `rc ≠ 0`; đầu ra chứa đúng chuỗi `TEST_ROUTES=1 cần stack dựng bằng docker-compose.test.yml` (`grep -cF` ≥ 1) |
| TC-PG07-50 | AC15 | Dev đã nộp baseline | **S** `50` — `grep -ciE 'RAM\|khởi động\|image\|p95' benchmarks/reports/pg-baseline.md`; `jq .startup_ms` của log `gateway ready` | Tệp tồn tại; ≥ `4` dòng khớp; ≥ 4 con số; `startup_ms` khớp `^[0-9]+$` |
| TC-PG07-51 | AC15 | Stack rảnh; TC chạy 60 s | **S** `51` — `sleep 60; docker stats --no-stream --format '{{.Name}} {{.MemUsage}}'` + `docker image inspect .Size` + p95 từ `$QC_OUT/k6.txt` | Đo được cả 4 số (RAM gateway, RAM worker, 2 kích thước image, p95); **chỉ ghi nhận** số đo vào report, không FAIL vì lệch so với `pg-baseline.md` (PM chốt góp ý #2) |
| TC-PG07-52 | AC16 (nhánh lỗi) | `.env.local` được sao lưu + `trap` khôi phục | **S** `52` — xoá dòng `JWT_SECRET_KEY=` khỏi `.env.local`; `$C up -d --force-recreate gateway`; `sleep 8`; đọc log + exit code; **khôi phục `.env.local` rồi mới chấm** | Log có `missing` chứa `JWT_SECRET_KEY`; exit code của container gateway = `1`; `cmp` xác nhận `.env.local` y hệt bản gốc |
| TC-PG07-53 | AC16 | Như TC-52; TC chạy ~90 s | **S** `53` — như trên, `sleep 60` rồi `docker inspect -f '{{.RestartCount}}'` + `{{.State.Running}}` | `RestartCount` lớn nhất ≤ **3**; **0** container gateway còn `Running=true` (không lặp vô hạn) |
| TC-PG07-54 | AC16 | Như TC-52 | **S** `54` — ghi `FinishedAt` của migrate trước/sau sự cố; `$C ps -a` hàng migrate | `FinishedAt` **không đổi**; migrate = `exited 0` (không bị ảnh hưởng) |
| TC-PG07-55 | AC17 (phân quyền / mạng) | – | **S** `55` — `$C ps -a --format '{{.Service}} {{.Ports}}' \| grep -E '^(gateway\|worker\|pgbouncer\|migrate) ' \| grep -c -- '->'` | `0` |
| TC-PG07-56 | AC17 / SRS 8.4 | – | **S** `56` — bảng cổng công bố của toàn stack | Tập service có `->` đúng bằng `caddy frontend mailpit minio postgres redis`; caddy có `:80->` và `:443->`; postgres `5433->`, redis `6380->`, minio `9000->`, mailpit `8025->`, frontend `3000->` |
| TC-PG07-57 | AC17 | – | **S** `57` — `grep -nE '(PASSWORD\|SECRET\|KEY)[A-Z_]*: *[^$ ]' docker-compose.local.yml` (và `.test.yml`) | `0` dòng ở cả hai tệp (mọi giá trị nhạy cảm là `${BIEN}`) |
| TC-PG07-58 | AC17 / 01-AC15 | – | **S** `58` — chấm chính: `grep -E '^[A-Z_]*(PASSWORD\|SECRET\|ACCESS_KEY)[A-Z_]*=' .env.example \| grep -vc -- '-dev'` và `grep -cE` cùng mẫu; phụ: so `JWT_SECRET_KEY` của `.env.example` với `.env.local`, `git check-ignore .env.local`, `git grep -nE 'JWT_SECRET_KEY=[^ ]{20,}' -- ':!.env.example' ':!docs'` | Chấm chính: `grep -vc -- '-dev'` = `0` và số biến bí mật đếm được ≥ `1` (không đo rỗng) — mọi giá trị bí mật ở `.env.example` chứa `-dev` (PM chốt góp ý #2). Phụ: hai giá trị **khác nhau**; `.env.local` bị git bỏ qua; `0` dòng lộ bí mật; `.env.example` vẫn liệt kê khoá `JWT_SECRET_KEY` |
| TC-PG07-59 | AC17 | – | **S** `59` — `curl http://localhost:8080/healthz`, `:8081`; đối chứng `rawhttp gateway 8080 'GET /healthz …'` từ container postgres | `000` và `000` từ host; trong mạng compose vẫn `HTTP/1.1 200` (chứng minh `000` không phải do gateway chết) |
| TC-PG07-60 | AC18 | – | **S** `60` — `$RDS config get appendonly\|appendfsync\|maxmemory-policy` | `yes`, `everysec`, `noeviction` |
| TC-PG07-61 | AC18 | – | **S** `61` — `$C config --volumes`; `docker volume ls` | Có volume tên chứa `postgres`, `redis`, `minio`, và `caddy_data`; ≥ 4 volume thật tên `edupilot_*` |
| TC-PG07-62 | AC18 | Redis chạy; khôi phục ngay sau lệnh đo | **S** `62` — `SET ep:qc:pg07:aof v1 EX 600`; `sleep 2`; `$C restart redis`; chờ `PONG` + `readyz`; `GET` lại rồi `DEL` | Giá trị vẫn là `v1` (AOF `everysec` giữ dữ liệu qua restart) |
| TC-PG07-63 | AC19 | Cổng PG do GATE chấm (sau khi các TC khác của story xong) | **T** `63` — chạy `bash docs/sprints/2/qc/scripts/gate-pg.sh 01 02 03 04 05 06 07 08 09 10 11 12 13` (đúng chuỗi lệnh của AC19: vet, golangci-lint 2 cấu hình, test -race -tags, sqlc diff, contract có/không tag, httpapi/sse/auth -race, `pnpm -C frontend build`, `dmode`, healthz qua Caddy, k6) | Cả 13 TC-GATE-01…13 PASS (xem `tc-GATE-PG.md`); AC19 không chấm lại ở story này, chỉ chép kết luận |
| TC-PG07-64 | AC20 | – | **S** `64` — `grep -cE '^FROM .* AS (gateway-test\|worker-test)$' backend-go/Dockerfile`; `grep -c testroutes` ; `grep -cE '^FROM .* AS (gateway\|worker)$'` | `2`; `2` (đúng hai dòng `go build -tags testroutes`); `2` (hai target mặc định) |
| TC-PG07-65 | AC20 | – | **S** `65` — `grep -c 'gateway-test\|worker-test' docker-compose.local.yml` | `0` |
| TC-PG07-66 | AC20 | – | **S** `66` — `$CT config --services`; `$CT config \| grep -cE 'image: edupilot-(gateway\|worker)-test'`, `… 'target: (gateway\|worker)-test'`; `jq` env của gateway/worker | Cùng **10** service như AC1 (override không thêm service); `2`; `2`; `APP_ENV` = `test` ở cả gateway và worker |
| TC-PG07-67 | AC20 | Dựng lại ảnh (~vài phút) | **S** `67` — `tmode` → `GET /api/v1/_test/whoami`; `dmode` → gọi lại (kết thúc ở dmode) | `200` ở image `*-test`; **`404`** ở image mặc định |
| TC-PG07-68 | AC20 | Chế độ test rồi trả về default | **S** `68` — `docker image inspect -f '{{.Size}}' edupilot-gateway-test\|worker-test` | Cả hai ≤ **39.999.999** byte |
| TC-PG07-69 | AC20 | – | **S** `69` — `grep -cE 'docker +push\|--push\|docker/build-push-action' .github/workflows/ci.yml`; `grep -nE '(gateway\|worker)-test' ci.yml \| grep -ci push` | `0`; `0` (CI không đẩy image `-test` đi đâu) |

### Bước tay TC-PG07-45 (xoá nhánh `ci/sqlc-drift` sau khi chấm)
1. Chạy và chấm xong `bash docs/sprints/2/qc/scripts/pg07.sh 43 44`; chép `databaseId`, `headSha` vào report pha 2.
2. Báo PM/dev: "QC đã chấm AC13, dev xoá được nhánh `ci/sqlc-drift`".
3. Sau khi dev báo đã xoá: `git fetch -q --prune origin && git ls-remote --heads origin ci/sqlc-drift` → **không in gì** → PASS. Còn in dòng SHA → FAIL (chưa xoá).
4. Nếu nhánh đã bị xoá **trước** bước 1 (TC-43/44 không tìm được run) → FAIL cả TC-43, TC-44, TC-45, ghi lý do "nhánh bị xoá sớm hơn lúc QC chấm".

### Bước tay TC-PG07-63
1. Chạy `gate-pg.sh 01 … 13` (hoặc lấy kết quả đã chạy ở `tc-GATE-PG.md`).
2. Chép PASS/FAIL từng TC-GATE-01…13 + lệnh vào report pha 2 của US-PG-07, mục AC19; không chạy hai lần để tránh lệch kết quả.

## Nhánh lỗi
SRS 3.4 không có hàng nào dành riêng cho hạ tầng chạy; ba hàng dưới đây được US-PG-07 kiểm ở mức hạ tầng (bản gốc ở US-PG-01), cộng ba nhánh lỗi do chính AC của story định nghĩa.

| Tình huống (SRS 3.4 / AC) | TC-id |
| --- | --- |
| Thiếu biến bắt buộc → thoát mã 1, log `missing:[…]` (ở tầng compose) | TC-PG07-52, 53, 54 |
| SIGTERM / mất một bản gateway → không mất phiên, không lỗi sau 6 s | TC-PG07-18, 19, 20 |
| DB qua PgBouncer transaction mode → không lỗi prepared statement / giao thức | TC-PG07-27, 28 |
| AC6 — tắt một container gateway giữa 200 request | TC-PG07-18, 19, 20 |
| AC13 — CI chặn lệch `sqlc` (nhánh `ci/sqlc-drift` đỏ đúng job, đúng bước) | TC-PG07-43, 44, 45 |
| AC16 — thiếu env trong compose (exit 1, ≤ 3 lần restart, migrate không ảnh hưởng) | TC-PG07-52, 53, 54 |
| AC14 — `TEST_ROUTES=1` trên stack mặc định (route thử 404) → k6 dừng có thông báo | TC-PG07-49 |

Story không sở hữu mã lỗi nào trong SRS 6.1 (không có handler nghiệp vụ); các mã lỗi HTTP chạm tới ở đây chỉ là `200`/`404` của endpoint nền (TC-17, TC-28) và `000` ở mức TCP (TC-59). Mã lỗi `NOT_READY`, `DEADLINE_EXCEEDED`… thuộc US-PG-01/03.

## Phân quyền
AC17 là mục phân quyền của story — **bề mặt mạng** thay cho vai trò (story hạ tầng, chưa có endpoint cần vai trò):
- Chỉ `caddy` công bố cổng ứng dụng ra host: TC-PG07-55, 56; PgBouncer đóng hoàn toàn: TC-PG07-22.
- Gateway/worker không chạm được từ host (`8080`, `8081` → `000`) nhưng vẫn sống trong mạng compose: TC-PG07-59.
- Không bí mật trong compose, mọi giá trị bí mật ở `.env.example` chứa `-dev` (đo trực tiếp), `.env.local` không vào git: TC-PG07-57, 58 (kiểm chéo US-PG-01 AC15).
- Không trạng thái phiên: cùng một token dùng được ở cả hai bản gateway, không bao giờ `401`: TC-PG07-17, 19.
- Image chạy bằng người dùng không phải root, không shell, rootfs chỉ-đọc: TC-PG07-31, 32, 34, 37.

## Điểm khó kiểm
- **TC-03 "máy sạch"**: script `$C down --remove-orphans` (**giữ** volume) rồi `up --wait` — xoá volume (`-v`) sẽ mất dữ liệu migrate/seed và làm các TC sau phải dựng lại từ đầu, nên "máy sạch" ở đây = không còn container cũ. Cold build trên colima 4 CPU mất **vài phút**; `up --wait` không bị `timeout` chặn (macOS không có `timeout`), thời gian được in ra.
- **TC-18 (AC6) chậm và dễ flaky**: 200 request nhịp 0,1 s + độ trễ curl ⇒ **≈ 22–30 s mỗi vòng**, 3 vòng kèm 3 lần `up --wait` ≈ 2–3 phút. Giảm flaky bằng: lặp 3 vòng (PASS chỉ khi cả 3 đạt), `rl_reset` đầu mỗi vòng, `--max-time 5` cho mỗi curl, khôi phục scale ngay sau mỗi vòng. `000` (lỗi kết nối) **tính là lỗi** theo US.md 07-AC6 v1.2 ("lỗi" = mọi response không-200 hoặc kết nối `000`).
- **HEAD vs GET (AC3)**: lệnh của AC dùng `curl -skI` (HEAD). Nếu gateway chỉ đăng ký `GET` thì HEAD có thể trả `405` trong khi header của Caddy vẫn có; TC-10 đo bằng **cả** GET (`-D-`) và HEAD và in mã HEAD ra để phân biệt "thiếu header" với "route không nhận HEAD".
- **openssl trên macOS là LibreSSL**: không có cờ `-ext`; TC-14 dùng `-text` rồi `grep 'DNS:localhost'`.
- **`nc`**: dùng `nc -z -w 2`; TC-22 có **đối chứng dương** (cổng 5433 phải mở) để một `nc` hỏng không bị đọc nhầm thành "cổng đã đóng".
- **Mốc thời gian docker** (TC-05): `StartedAt`/`FinishedAt` có số chữ số nano khác nhau → chuẩn hoá bằng `tr -d '-:TZ.'` + đệm `0` rồi so chuỗi (bash 3.2 không có so sánh thời gian sẵn, không dùng `date +%s%N`).
- **`$C config --format json`** cần Docker Compose ≥ v2.21 (TC-06, 26, 33, 66). Thiếu thì TC FAIL vì lý do công cụ — ghi rõ phiên bản `docker compose version` vào report.
- **Giới hạn tốc độ**: TC-28 bắn 1.000 request nên tự `rl_reset` giữa 20 vòng thay vì đặt `RATE_LIMIT_IP_PER_MIN` bằng `gw_env` — compose có thể **không chuyển tiếp** biến này vào container (SRS 8.1 liệt kê biến nhưng compose quyết định), nên cách qua Redis chắc chắn hơn.
- **k6**: phải cài sẵn (`brew install k6`); chứng chỉ Caddy là TLS nội bộ nên `smoke.js` phải tự đặt `insecureSkipTLSVerify: true` — QC chạy **nguyên văn** lệnh của AC, nếu k6 chết vì TLS thì đó là lỗi của `smoke.js` (FAIL AC14), không phải TC. Số đo trên máy dev (colima 4 CPU) có thể không đạt — theo Q14, **không nới ngưỡng**, chỉ ghi số + cấu hình máy vào report.
- **`gh`** phải đã `gh auth login` và có quyền đọc Actions; TC-43/44 còn phụ thuộc nhánh `ci/sqlc-drift` **chưa bị xoá** (nếu dev xoá sớm → FAIL, không có cách kiểm thay thế).
- **TC-52…54 (AC16)** mỗi TC tự dựng lại sự cố nên mỗi TC mất 1–2 phút; `.env.local` được `cp` ra `$QC_TMP`, có `trap … INT TERM`, khôi phục **trước** khi chấm và `cmp` để chắc chắn không để lại rác. Không dùng `sed -i` (macOS cần hậu tố).
- **TC-51** ngủ 60 s theo AC ("sau 60 s rảnh"); `docker stats` liệt kê mọi container của máy nên lọc theo tên — nếu máy chạy stack khác cùng tên thì số đo sai, ghi chú khi chấm. Lệch so với `pg-baseline.md` **không FAIL**, chỉ ghi số đo vào report (PM chốt góp ý #2).
- **TC-67, 68** dựng lại image (`tmode`/`dmode`) nên chậm; cả hai kết thúc ở **chế độ default** để TC sau chạy tiếp đúng trạng thái chuẩn.
- **TC-64** dùng `grep -cE '^FROM .* AS (…)$'` **nguyên văn** của AC: nhạy chữ hoa `AS` và không chấp nhận khoảng trắng cuối dòng. Dev viết `as` thường → TC FAIL theo đúng lệnh của AC (báo như lỗi, không tự nới regex).
- **TC-61** tên volume: SRS 8.2 chốt `caddy_data`; tên của postgres/redis/minio chỉ ghi "volume tên cố định" nên TC kiểm theo **chuỗi con** (`postgres`, `redis`, `minio`) qua `$C config --volumes` cộng tiền tố thật `edupilot_*` của `docker volume ls`.
- **TC-36 (AC11)** chỉ chạy `golangci-lint run` ở cấu hình **không tag**; cấu hình `--build-tags testroutes` thuộc AC12 (kiểm qua danh sách bước CI, TC-40).
- Tương thích bash 3.2: không `mapfile`, không mảng kết hợp, không `${x,,}`, không `date +%s%N` (dùng `now_ms` của `lib.sh`), không `timeout` trên host (dùng `curl --max-time`; `timeout` chỉ dùng trong container postgres qua `rawhttp`).

## Câu hỏi cho PM
- **Q-QC-07-1** — đã trả lời (BA, spec v1.2): chấm theo AC10, service `migrate` phải đúng `image: edupilot-gateway`, `entrypoint: ["/gateway","migrate"]`, `command: ["up"]` (SRS 8.4 đã sửa cho khớp); TC-PG07-33 đã sửa.
- **Q-QC-07-2** — đã trả lời (BA, spec v1.2; PM chốt góp ý #2): baseline chỉ cần đủ 4 số, QC đo lại ghi số của mình vào report, không FAIL vì lệch; TC-PG07-50, TC-PG07-51 đã sửa (bỏ ghi chú chờ dung sai).
- **Q-QC-07-3** — đã trả lời (BA, spec v1.2; PM chốt góp ý #2): đo trực tiếp `.env.example` — mọi giá trị bí mật chứa `-dev`, kiểm bằng `grep -E '^[A-Z_]*(PASSWORD|SECRET|ACCESS_KEY)[A-Z_]*=' .env.example | grep -vc -- '-dev'` → `0`; TC-PG07-58 đã sửa.
- **Q-QC-07-4** — đã trả lời (BA, spec v1.2): "lỗi" trong cửa sổ tắt = mọi response không-200 hoặc kết nối `000`, ≤ 4 trên 200 request và 0 sau 6 s; TC-PG07-18 giữ cách đo, đã ghi nguồn.
- **Q-QC-07-5** — đã trả lời (BA, spec v1.2): trên stack mặc định `k6 run -e TEST_ROUTES=1 …` phải `rc ≠ 0` và in đúng chuỗi `TEST_ROUTES=1 cần stack dựng bằng docker-compose.test.yml`; TC-PG07-49 đã sửa.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-02 — viết lần đầu theo US.md v1.1 (đã gồm góp ý #1 route thử = build tag `testroutes`; `tmode`/`dmode`/`CT`); không có TC nào bị sửa vì chưa có TC cũ.
- 2026-10-02 — spec v1.2 (commit 02a4435; QC questions #Q-QC-07-1, #Q-QC-07-4, #Q-QC-07-5): TC-PG07-33 sửa: đòi đúng ba giá trị `image=edupilot-gateway`, `entrypoint=["/gateway","migrate"]`, `command=["up"]` (bỏ cách chấm lỏng "đối số chứa migrate"); TC-PG07-18 sửa: ghi nguồn định nghĩa "lỗi" = không-200 hoặc `000` (ngưỡng và cách đo không đổi); TC-PG07-49 sửa: đòi đúng chuỗi `TEST_ROUTES=1 cần stack dựng bằng docker-compose.test.yml` bằng `grep -cF`, bỏ regex lỏng `_test|route thử|test route|404`.
- 2026-10-02 — spec v1.2 (commit 02a4435; QC questions #Q-QC-07-2, #Q-QC-07-3; góp ý #2 PM ACCEPTED 01/10): TC-PG07-58 sửa: chấm chính bằng phép đo trực tiếp `.env.example` (`grep -vc -- '-dev'` = `0`, kèm `grep -cE` ≥ 1 để không đo rỗng), giữ các kiểm gián tiếp làm phụ; TC-PG07-50, TC-PG07-51 sửa: bỏ ghi chú "chờ trả lời"/dung sai, ghi rõ chỉ ghi nhận số đo, không FAIL vì lệch.

Tổng: **69 TC** (67 tự động **S**, 2 tay **T**: TC-PG07-45, TC-PG07-63 — TC-63 chấm bằng TC-GATE-01…13). Bản v1.2 không thêm/xoá TC, chỉ siết 5 TC.
