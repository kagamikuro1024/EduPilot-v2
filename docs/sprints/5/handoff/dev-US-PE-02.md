# DEV handoff — US-PE-02 (máy chấm code: go-judge trong sandbox, hàng đợi riêng)
Nhánh `sprint/5-pe`, spec `FEAT-weekly-exam` v1.6 (SRS 4.5). Không có migration mới (cột `lease_until`, `next_attempt_at`, `fail_count`, `enqueued_at` đã có từ `00006`).

## Làm gì
- **`deploy/judge/`**: `Dockerfile` (debian trixie-slim + `g++`/`gcc` 14.2.0, `criyle/go-judge:v1.13.0`), `start.sh` (thoát 1 nếu `JUDGE_TOKEN` < 16 ký tự; **lọc token khỏi log của chính go-judge** — go-judge in `AuthToken` ở dòng "Config loaded" và "Attach token auth", đo thấy 2 dòng lộ token trước khi lọc).
- **`internal/judge`**: `Client` (biên dịch 1 lần → chạy từng test, `copyOutCached` + `DELETE /file`, semaphore, hạn mỗi lời gọi, token không vào log), `MapRunStatus` (bảng SRS 4.5.4, `IE` cho lỗi hệ thống), `Check` (EXACT / TOKENS / FLOAT_EPS, Go thuần), `Client.Judge` (mọi test, chỉ test `approved`, Σweight = 0 → `ConfigProblem`), `Queue` (consumer: tín hiệu Stream, nhận việc / thuê / gia hạn / thử lại ở DB bằng `now()` của Postgres, chỗ `RUN` dành riêng, thay bản cũ `SUPERSEDED`, tick của leader đưa lại hàng đến hạn / hết thuê / mất tín hiệu, thăm dò `ep:judge:up`), `SettingsFromEnv` + `CheckMemLimit` (công thức `P×(512+256)+512` MiB).
- **`cmd/worker`**: outbox topic `judge.enqueue` → `Queue.HandleEnqueue`; consumer chỉ chạy khi `JUDGE_CONSUMER=true`; thiếu / ngắn `JUDGE_TOKEN` hoặc `JUDGE_MEM_LIMIT` thấp → log nêu tên biến, thoát 1; `/readyz` mới (`{"status":"ok","judge":"up|down|off"}`, luôn 200); `/healthz` không đổi, không phụ thuộc judge.
- **Compose**: dịch vụ `judge` (build `deploy/judge`, `privileged`, `cgroup: host`, `shm_size: 256m`, `cpus`/`mem_limit` từ `JUDGE_CPUS`/`JUDGE_MEM_LIMIT`, không `ports`, không volume, env chỉ `JUDGE_*`), mạng `judge_net` (`internal: true`) chỉ gắn `judge` + `worker`; `worker` đợi `judge` healthy. `.env.example` có `JUDGE_*`. `docker-compose.test.yml` chỉ là overlay lên `local` nên thừa hưởng dịch vụ `judge`.
- **SQL**: `internal/store/queries/judge.sql` (claim / renew / finish / fail / requeue … tất cả ở sqlc, thời gian dùng `now()` của DB).
- **CI**: job `judge-attacks` (ubuntu-24.04 = amd64, **không** `-no-seccomp`). Makefile: `make test-judge` (build image + chạy toàn bộ test `-tags integration` của `internal/judge`; arm64: `JUDGE_EXTRA_ARGS=-no-seccomp make test-judge`).

## AC (kết quả thật)
| AC | Kết quả |
| --- | --- |
| 1 | `docker compose -f local -f test config` → `privileged:true, cgroup:"host", ports:null, shm_size:268435456, cpus:2, mem_limit:2147483648`, env = `JUDGE_EXTRA_ARGS, JUDGE_PARALLELISM, JUDGE_TOKEN`; dịch vụ khác có `privileged` = **0**; `/opt/go-judge -version` = `v1.13.0`; `g++ --version` = `g++ (Debian 14.2.0-19) 14.2.0`; `-enable-debug` không bật. `TestJudgeMemLimitCheck` (10 ca: P2+2g, P2+2048m, P4+3584m đạt; P2+1g, P4+2g, rác → lỗi nêu `JUDGE_MEM_LIMIT`) PASS |
| 2 | Stack thật: `getent hosts postgres redis minio gateway caddy mailpit` từ `judge` → **0** tên phân giải; `1.1.1.1:53` → `Network is unreachable` → `FAIL`; đối chứng `worker:8081` → `OPEN`; `judge_net` có **2** container (`judge`, `worker`); gateway chỉ ở `default` (`judge:5050` từ netns gateway: `bad address`); `judge` 0 volume, 0 cổng publish. A10 / A10b qua đường chấm thật PASS (xem AC11) |
| 3 | Từ container tạm trong `judge_net`: `POST /run` không token → **401**; có token → **400** (không phải 401); `GET /version` → **200**. Token trong log: worker **0**, gateway **0**, judge **0** (sau khi sửa `start.sh`; trước đó **2**). `TestClientRequiresToken`, `TestTokenNeverLogged` PASS |
| 4 | `TestClientCompileRequest`, `TestClientRunRequest`, `TestClientDeletesCachedFile`, `TestClientDeadlineAndSemaphore` (máy chủ giả httptest, `-race`) PASS; `grep -rnE 'os/exec\|syscall\.Exec\|exec\.Command' internal/judge internal/exam` ngoài `_test.go` = **0** |
| 5 | `TestVerdictMapping` PASS (gồm ca `lstat stdout: operation not permitted` → `IE`) |
| 6 | `TestCheckerExact`, `TestCheckerTokens`, `TestCheckerFloatEps` PASS. Lưu ý: `FLOAT_EPS` cần `float64` ở **bộ so khớp** (`internal/judge`); lệnh `grep float64` của US-PE-01 AC7 chỉ quét `internal/quiz internal/exam` = 0 |
| 7 | `TestSubmissionResultAssembly`, `TestNoEarlyStop`, `TestZeroTotalWeight`, `TestOnlyApprovedTests` PASS |
| 8 | **8/8** PASS với Postgres + Redis thật, judge giả: `TestJudgeConsumerIdempotent`, `TestJudgeLeaseNoDoubleRun`, `TestJudgeLongRunNotReclaimed` (chạy 2,7 s ≫ thuê 0,6 s, `fail_count`=0), `TestJudgeLeaseExpiryRequeue`, `TestJudgeBackoffNextAttemptAt`, `TestJudgeRetryThenDead` (4 lần → `ERROR` + `IE`, `passed_weight` NULL, 1 mục ở `judge.submit.dead`, rồi chấm lại → `DONE`), `TestJudgeKillWorkerMidRun`, `TestJudgeRunLaneReserved` |
| 9 | `TestSupersedeQueued` (bản 1 → `SUPERSEDED` không chạy sandbox; `RUN` không bị thay), `TestRunningNotCancelled` PASS |
| 10 | **Một phần.** `TestJudgeUpKeyLifecycle` (đặt khoá TTL ≤ 30 s, xoá khi lỗi, đặt lại khi sống) PASS. **Chưa**: `TestRunWhenJudgeDown503`, `TestSubmitWhenJudgeDownQueued`, `TestIENeverScoresZero` ở tầng gateway / `internal/exam` — cần handler `Chạy thử` / `Nộp lời giải` (US-PE-06) và chấm điểm (US-PE-08); gateway chỉ đọc `ep:judge:up`. Phía hàng đợi "IE không điểm 0" đã được kiểm ở `TestJudgeRetryThenDead` |
| 11 | `TestSandboxAttacks` PASS 10,9 s, 15 dòng bên dưới + `A6 đồng thời x2 [MLE MLE] container sống`; `RestartCount` = 0 trước và sau; bài đúng của sinh viên khác AC ≤ 0,1 s trong mọi ca |
| 12 | Máy đo: colima 4 CPU / 4 GiB (arm64, `-no-seccomp`), container `judge` `cpus=4` `mem_limit=3584m`, `JUDGE_PARALLELISM=4`. **60 bài `<bits/stdc++.h>` × 10 test: 17,3 s (p95 17,1 s), 60 AC, 0 `IE`**; **240 bài: 51,4 s (≈ 280 bài/phút), 240 AC**. `JUDGE_PARALLELISM=2` (`2g`, 2 CPU): **60 bài: 21,1 s (p95 20,5 s)** — chỉ ghi số đo. Cách đo: chèn thẳng 60 / 240 dòng `code_submissions` (mỗi bài một lượt, 10 test) vào Postgres của stack thật, để tick của worker đưa vào Stream (đi qua cả đường tick). `TestRunLatencyIdle`: **p50 564 ms, p95 749 ms**, max 763 ms (20 lượt). **Chưa** `k6 … SCENARIO=judge_burst` — cần API nộp bài (US-PE-06); kịch bản làm ở US-PE-09 |
| 13 | `TestNoSourceInLogs` (canary trong mã nguồn / test, cả đường lỗi; không có token) PASS, `TestCompileLogScrubbed` PASS |
| 14 | `TestJudgeProbe` (đồng hồ giả: dưới 60 s không cảnh báo; quá 60 s `warn` **đúng một lần** dù thăm dò nhiều lần; đợt sập mới cảnh báo lại), `TestReadyzJudgeInfoOnly` (200 với `up` / `down` / `off`, kể cả khi DB đóng) PASS. Stack thật: `/readyz` → `{"status":"ok","judge":"up"}`. Phần "Today hiện việc `EXAM_GRADE_ERROR`" thuộc nguồn việc Today của US-PE-08 |
| 15 | arm64 + `-no-seccomp`: 15/15 (xem AC11). **amd64 (seccomp bật) chưa chạy được ở local** — CI job `judge-attacks` đã thêm nhưng GitHub Actions không chạy được (billing). Không tự tắt seccomp ở amd64 |

### 15 dòng của `TestSandboxAttacks` (colima arm64, `-no-seccomp`)
```
A1 TLE vòng lặp vô hạn · A2 TLE in vô hạn · A3 OLE in 5 MB · A4 OLE in 1 GB · A5 MLE tới 512 MiB · A6 MLE 2 GB (+ x2 đồng thời: MLE MLE)
A7 RE NULL · A8 TLE fork bomb · A9 AC (5 đường tệp hệ thống CHAN) · A10 AC connect 1.1.1.1:53 = -1 · A10b AC connect -1 -1 -1 -1
A11 CE #include "/dev/random" · A12 CE bom constexpr · A13 WA ghi 1 GB vào /tmp (bị chặn) · A14 AC system() BLOCKED · A15 AC CHAN×2
```

## Lệnh tự kiểm
- `cd backend-go && make lint sqlc-check` → 0 issues ×3, `sqlc diff` sạch.
- `make test` → xanh, trừ **`internal/mail TestRetryThenDead` đỏ một lần** khi chạy cả bộ (bộ integration mail / auth); chạy riêng 2 lần PASS và chạy lại cả `./internal/mail/...` PASS — lỗi chập chờn dưới tải.
- `JUDGE_EXTRA_ARGS=-no-seccomp make test-judge` → `ok internal/judge 69.9s` (toàn bộ test `-tags integration`, judge thật).
- `go test ./cmd/worker -run 'TestJudgeMemLimitCheck|TestReadyzJudgeInfoOnly'` PASS.
- Tay: `EP_PORT_OFFSET=330 pnpm dev` (stack dựng được, `judge` healthy, worker healthy); các lệnh AC2 / AC3 ở bảng trên.

## Nợ / ghi chú
- **CI GitHub không chạy được** (billing) → AC15 amd64 chưa có bằng chứng; job `judge-attacks` chờ.
- Worker **chưa đọc test lớn lưu ở object storage** (`input_blob_key`): `BlobReader` có sẵn trong `Queue` nhưng chưa nối `platform/blob`; tới khi US-PE-03 lưu test > ngưỡng ra blob thì nối (một dòng ở `cmd/worker/judge.go`). Test lưu ở DB chấm bình thường.
- `k6` `judge_burst` + hợp đồng 503 `JUDGE_UNAVAILABLE` ở gateway: chờ US-PE-06 / US-PE-09 (xem AC10, AC12).
- `docker-compose.test.yml` là overlay: AC1 phải chạy với `-f docker-compose.local.yml -f docker-compose.test.yml` (như bảng trên).
- Ngưỡng `JUDGE_MEM_LIMIT` colima P=4 là `3584m`, nên tắt bớt gateway / frontend khi đo (đã làm: dừng `frontend caddy gateway mailpit minio`).
