# QC test case — US-PE-02 (sandbox chấm code `go-judge`, `internal/judge`, hàng đợi `judge.submit`, ca tấn công)
Nguồn: `docs/specs/FEAT-weekly-exam/US.md` US-PE-02 AC1–AC15 + `SRS.md` 4.5.1–4.5.7 (container, REST, giới hạn, phân loại, checker, **bảng tấn công A1–A15**, hàng đợi), 8.1–8.2 (biến môi trường, compose), 8.4. Research `docs/research/2026-10-03-code-judge.md`. Hộp đen: QC dựng stack có `judge`, **tự viết mã tấn công** (`docs/sprints/5/qc/scripts/attacks/*.c|cpp`, A1–A15 + A16–A20 bổ sung; không dùng mã của dev) và nộp qua **đường chấm thật**.

Tiền điều kiện chung: `source ~/.zprofile`; `export TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=unix://$HOME/.colima/default/docker.sock`; stack riêng của QC dựng từ `docker-compose.test.yml` **với tên dự án và cổng riêng** (không chạm `edupilot-test-*` của phiên khác): Postgres, Redis, gateway `testroutes`, worker, `judge` (`build: deploy/judge`); `JUDGE_EXTRA_ARGS=-no-seccomp` trên colima arm64 (D58). Biến quy ước như `US.md`. **Đường nộp thật:** QC tạo bài code qua API (`POST …/questions`, `PUT …/questions/{qid}/code`, `POST …/testcases`, duyệt), bài thi (`POST …/exams`, `PUT …/items`, `POST …/schedule`), SV bắt đầu lượt (`POST …/attempts`) rồi `POST …/code/{itemId}/submit`; hàm `submit_src <tệp> <ngôn-ngữ>` của QC gói các bước này (lưu ở `scripts/p502-lib.sh` khi viết lúc chạy). Nếu API soạn đề / làm bài (US-PE-03/04/06) chưa có, TC-PE02-24…40 chạy lại ở cổng PE; TC hạ tầng (01–10) và test Go chạy ngay. Công cụ: **S** = shell, **D** = SQL, **G** = `go test`, **R** = Redis CLI, **P** = Python.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PE02-01 | AC1 | stack | **S** `docker compose -f docker-compose.test.yml config --format json \| jq '.services.judge \| {privileged, cgroup, ports, shm_size, build, command}'`; đếm dịch vụ tên `judge` | đúng **1** dịch vụ; `privileged:true`, `cgroup:"host"`, `ports:null`, `shm_size:"256m"`, `build` trỏ `deploy/judge`, `cpus`/`mem_limit` có giá trị (mặc định 2.0 / 2g) |
| TC-PE02-02 | AC1 | – | **S** `… jq '[.services \| to_entries[] \| select(.key!="judge") \| select(.value.privileged==true)] \| length'`; môi trường của `judge`: `… jq '.services.judge.environment'` | `0`; mọi biến của `judge` bắt đầu `JUDGE_`; `-enable-debug` **không** có trong lệnh; `-no-seccomp` chỉ khi `JUDGE_EXTRA_ARGS` đặt (đọc `deploy/judge/start.sh` và `.env.example`) |
| TC-PE02-03 | AC1 | – | **S** `docker compose exec judge /opt/go-judge -version`; `docker compose exec judge g++ --version \| head -1`; `gcc --version \| head -1`; `cat /etc/os-release \| grep VERSION_CODENAME` | `v1.13.0`; chứa `14.2.0` (g++ và gcc); `trixie` |
| TC-PE02-04 | AC1 | – | **S** `docker inspect -f '{{.HostConfig.Privileged}} {{.HostConfig.CgroupnsMode}}' <gateway|worker>` | `false`, không `host` cho gateway và worker |
| TC-PE02-05 | AC2 (mạng) | stack | **S** từ trong `judge`: `for h in postgres redis minio gateway caddy mailpit; do docker compose exec -T judge sh -c "getent hosts $h \|\| echo NOHOST"; done \| grep -vc NOHOST`; `wget -qO- -T3 http://1.1.1.1 \|\| echo FAIL`; `nc -zv -w2 <IP của postgres> 5432` | `0` phân giải được; `FAIL`; kết nối IP trực tiếp cũng **thất bại** (không chỉ DNS) |
| TC-PE02-06 | AC2 | – | **S** `docker network inspect <proj>_judge_net \| jq '.[0].Internal, (.[0].Containers\|length)'`; `docker compose exec gateway sh -c 'getent hosts judge \|\| echo NOHOST'` (gateway không gắn `judge_net`); worker: `wget -qO- http://judge:5050/version`; `docker compose config --format json \| jq '.services.judge.volumes // [] \| length'` | `true`, `2` (`judge` + `worker`); gateway `NOHOST`; worker nhận `v1.13.0`; `0` volume |
| TC-PE02-07 | AC3 (token) | `JUDGE_TOKEN` ≥ 16 ký tự | **S** container tạm trong `judge_net`: `POST /run` không token / token sai / token đúng; `GET /version` không token | `401`, `401`, `200`/`400` (không 401), `200` |
| TC-PE02-08 | AC3 | – | **S** khởi động worker với `JUDGE_TOKEN` thiếu; với 15 ký tự (QC dùng `run.py` bọc timeout) | thoát `rc=1` nêu **`JUDGE_TOKEN`**; 16 ký tự → chạy |
| TC-PE02-09 | AC3 (log) | sau mọi TC | **S** `docker compose logs gateway worker judge \| grep -c "$JUDGE_TOKEN"`; `pg_dump` / Redis `--scan` | `0` mọi nơi |
| TC-PE02-10 | AC3 | – | **G** `-run 'TestClientRequiresToken|TestTokenNeverLogged' -v` | `ok` |
| TC-PE02-11 | AC4 | – | **G** `go test -race ./internal/judge/... -run 'TestClientCompileRequest|TestClientRunRequest|TestClientDeletesCachedFile|TestClientDeadlineAndSemaphore' -v`; QC tự đọc thân yêu cầu mà máy chủ giả ghi lại và so với SRS 4.5.2 (cờ biên dịch, `procLimit`, `copyOutCached`, `clockLimit = 3 × cpu`) | `ok`; thân khớp từng trường |
| TC-PE02-12 | AC4 | – | **S** `grep -rnE 'os/exec\|syscall\.Exec\|exec\.Command' backend-go/internal/judge backend-go/internal/exam \| grep -v _test.go \| wc -l`; gieo tệp tạm `internal/judge/zz.go` dùng `exec.Command("gcc")` → test / lint | `0`; gieo → test cấm **đỏ** (nếu có) hoặc `golangci-lint` đỏ; xoá → xanh |
| TC-PE02-13 | AC4 (không ghi đĩa) | – | **S** chạy một lượt chấm; `docker diff <worker>` / `find /tmp -newer` trong container worker; đo `du` thư mục dữ liệu | worker không tạo tệp mới ngoài log; không `gcc` trong ảnh worker (`docker compose exec worker which gcc g++` → không có) |
| TC-PE02-14 | AC5 (phân loại) | – | **S** QC nộp các mã **do QC viết** (`a01`…`a20`, một bài đúng `ok.c`, một bài WA, một CE `int main(){ x }`) và ghi verdict; so với bảng 4.5.4 | `AC` / `WA` / `TLE` / `MLE` / `OLE` / `RE` / `CE` đúng; **`IE` chỉ khi** lỗi hệ thống |
| TC-PE02-15 | AC5 (`IE` ≠ `WA`) | – | **S** làm judge lỗi: `docker compose stop judge` giữa lúc chấm; hoặc proxy giả (Bun) giữa worker và judge trả 500 / `Internal Error` / `File Error` / `lstat stdout: operation not permitted` | verdict `IE` (không `RE`, không `WA`, **không 0 điểm**); bài ở lại chờ, thử lại |
| TC-PE02-16 | AC5 | – | **G** `-run TestVerdictMapping -v` | `ok`; ≥ 16 ca (có ca `lstat stdout`) |
| TC-PE02-17 | AC6 (checker) | bài có 3 loại checker | **S** nộp cùng một chương trình in `"1.0000001\n"` rồi `"1e3\n"` so với `expected` `1` / `1000`: `EXACT`, `TOKENS`, `FLOAT_EPS` (`eps=1e-6`); CRLF; khoảng trắng cuối dòng; thiếu dòng trống cuối; dòng thừa; `NaN`, `Inf`; đầu ra rỗng ↔ `expected` rỗng; Unicode khoảng trắng (U+00A0, U+2003) cho `TOKENS` | đúng bảng 4.5.5: `EXACT` = chuẩn hoá CRLF + khoảng trắng cuối dòng + dòng trống cuối rồi so byte; `TOKENS` tách theo khoảng trắng Unicode; `FLOAT_EPS` `|a−b| ≤ eps×max(1,|b|)` (biên: đúng bằng eps → đạt; vượt → `WA`); `NaN`/`Inf` so chuỗi; rỗng ↔ rỗng `AC`; QC ghi bảng 20 ca |
| TC-PE02-18 | AC6 | – | **G** `-run 'TestCheckerExact|TestCheckerTokens|TestCheckerFloatEps' -v` | `ok`; ≥ 36 ca |
| TC-PE02-19 | AC6 (so khớp trong worker) | – | **S** chặn bằng `tcpdump` / log judge: khi chấm có lệnh thứ hai (checker) gửi tới judge không? đếm `POST /run` mỗi bài (1 biên dịch + N test) | đúng `1 + N` lời gọi `/run`, không lệnh so khớp; `DELETE /file/{id}` sau chấm |
| TC-PE02-20 | AC7 (kết quả bài) | bài 6 test trọng số `[1,1,1,2,0,3]`; mã đúng test 1, 2, 4; test 3 TLE; test 5 WA; test 6 RE | **D** `select verdict, passed_weight, total_weight, compile_ok, tests_version, status, jsonb_array_length(results) from code_submissions where id=<S>` + đọc `results` | chạy đủ **6** test (không dừng sớm); `passed_weight=4`, `total_weight=7`; `verdict` = verdict của test lỗi đầu tiên theo `position` (`TLE`); test `weight=0` vẫn chạy và hiện; `status=DONE`; mỗi phần tử có `test_id, position, is_sample, verdict, time_ms, memory_kb, weight` |
| TC-PE02-21 | AC7 | – | **S** bài có test **chưa duyệt** (`approved=false`) và test duyệt; nộp; `Σweight = 0` (mọi test trọng số 0) | chỉ test `approved=true` được chấm; `Σweight=0` → `IE` + log cấu hình, **không** chia cho 0, không điểm |
| TC-PE02-22 | AC7 | – | **G** `-run 'TestSubmissionResultAssembly|TestNoEarlyStop|TestZeroTotalWeight|TestOnlyApprovedTests' -v` | `ok` |
| TC-PE02-23 | AC8 (hàng đợi) | stack | **R** `XINFO GROUPS judge.submit`; `XINFO STREAM judge.run`; nộp 1 bài; `D` bảng `outbox` topic `judge.enqueue` ghi cùng transaction (tắt worker, nộp, đếm `code_submissions` và `outbox`) | nhóm `judge` có; outbox có đúng 1 dòng / bài nộp; bài ghi cùng lúc với outbox (không có bài mà thiếu outbox) |
| TC-PE02-24 | AC8 (idempotent) | – | **R** giao lại tin cũ: `XADD judge.submit * submission_id <S>` hai lần sau khi `DONE` | không chạy lại sandbox (đếm `POST /run` không tăng); kết quả không đổi; 1 dòng kết quả |
| TC-PE02-25 | AC8 (giết worker) | bài 10 test, mỗi test 1 s | **S** `kill -9` worker khi đang chấm; bật lại; đợi `JUDGE_CLAIM_IDLE` | bài được chấm lại **đúng một** kết quả cuối (không dòng đôi, điểm như lượt chạy trọn); tin được `XAUTOCLAIM` |
| TC-PE02-26 | AC8 (retry + dead) | proxy lỗi / judge dừng | **S** dừng judge; nộp; theo dõi `attempts` và thời điểm | thử lại 1 s / 5 s / 30 s (±30 %), lần 4: `status=ERROR`, `verdict=IE`, `judge.submit.dead` có 1 tin (không chứa mã nguồn), `XACK` |
| TC-PE02-27 | AC8 (làn `run`) | hàng `judge.submit` đầy (60 bài) | **S** trong lúc đó SV khác `Chạy thử` | `Chạy thử` xong ≤ 5 s (không chờ hàng nộp); `judge.run` có goroutine riêng |
| TC-PE02-28 | AC8 | – | **G** `-tags integration -run 'TestJudgeConsumerIdempotent|TestJudgeRetryThenDead|TestJudgeKillWorkerMidRun|TestJudgeRunLaneReserved' -v` | `ok` |
| TC-PE02-29 | AC9 (SUPERSEDED) | 1 worker, hàng có 20 bài khác | **S** SV nộp lần 1 (QUEUED), lần 2; xem trạng thái sau khi consumer nhận | lần 1 `SUPERSEDED`, **0** lời gọi sandbox cho lần 1; lần 2 `DONE`; lần `RUNNING` không bị huỷ; `Chạy thử` không bao giờ `SUPERSEDED` |
| TC-PE02-30 | AC9 | – | **G** `-run 'TestSupersedeQueued|TestRunningNotCancelled' -v` | `ok` |
| TC-PE02-31 | AC10 (judge chết) | `docker compose stop judge` | **S** `Chạy thử`: đo thời gian; `Nộp lời giải`; trạng thái lượt; bật judge; `POST …/regrade` | `503 JUDGE_UNAVAILABLE` + `retry_after` 5…30 trong **≤ 3 s** (không treo); nộp vẫn `202`, `QUEUED`; judge lên → tự chấm; sau dead-letter `IE`, lượt ở `GRADING` (không `GRADED`, không điểm 0); `regrade` chấm lại mọi `IE` |
| TC-PE02-32 | AC10 | – | **G** `-tags integration -run 'TestRunWhenJudgeDown503|TestSubmitWhenJudgeDownQueued|TestIENeverScoresZero' -v` | `ok` |
| TC-PE02-33 | AC11 A1, A2 | judge sống | **S** nộp `a01`, `a02` (bài giới hạn 1.000 ms, 256 MiB) | A1 `TLE` (CPU ≈ 1.000 ms ± 200); A2 `TLE`, stdout bị cắt; RAM `judge` ổn định (`docker stats` mẫu mỗi 1 s) |
| TC-PE02-34 | AC11 A3, A4 | – | **S** `a03`, `a04` (output limit mặc định 1.024 KiB); đo đĩa máy chủ trước / sau (`docker system df`, `du` volume) | A3 `OLE`; A4 `OLE` hoặc `TLE`; RAM / đĩa đổi ≤ 10 MiB |
| TC-PE02-35 | AC11 A5, A6 | – | **S** `a05`, `a06`; `docker inspect RestartCount` | A5 `MLE`; A6 `MLE` hoặc `RE`; container không khởi động lại |
| TC-PE02-36 | AC11 A7, A18 | – | **S** `a07`, `a18` | `RE` (`Signalled`) cả hai; không treo |
| TC-PE02-37 | AC11 A8, A17 | – | **S** `a08`, `a17`; sau 30 s đếm tiến trình trong `judge` (`docker top`) và trên máy chủ | không tạo được tiến trình / luồng thứ hai (`FORKED=0`, `THREADS=0` hoặc `RE`/`TLE`); **0** tiến trình mồ côi |
| TC-PE02-38 | AC11 A9 | – | **S** `a09` (8 đường, gồm 5 đường của SRS + `/proc/self/environ`, `/etc/hostname`, `/proc/mounts`) | 5 đường SRS **in `CHAN`**; ghi lại 3 đường thêm (đọc được cái nào → báo dev, **FAIL** nếu lộ `JUDGE_TOKEN`, biến môi trường hay secret) |
| TC-PE02-39 | AC11 A10, A10b | – | **S** `a10`, `a10b` | `socket` được, `connect=-1` cho `1.1.1.1:53`, `postgres:5432`, `redis:6379`, `gateway`, `host.docker.internal`, `172.17.0.1:22`, `127.0.0.1:5050` (không gọi được ngay cả go-judge nội bộ từ mã SV) |
| TC-PE02-40 | AC11 A11, A12 | – | **S** `a11`, `a12` (đo thời gian thực) | `CE` trong ≤ **25 s**; bài khác không bị chậm (xem TC-PE02-44) |
| TC-PE02-41 | AC11 A13, A15 | – | **S** `a13`; rồi cặp `a15-leak-write` → ngay sau `a15-leak-read` (cùng bài, hai bản nộp liên tiếp, cùng worker song song 1) | A13 bị chặn / `MLE` / `RE`; đĩa máy chủ ≤ 10 MiB; A15 lượt đọc in `CHAN` ×3 (**không** `LEAKED`) |
| TC-PE02-42 | AC11 A14, A16 | – | **S** `a14`, `a16` | không tạo được tiến trình con; `ptrace`/`mount`/`unshare`/`chroot` đều thất bại (`-1`); `RE`/`WA`; không đọc `/etc/shadow` |
| TC-PE02-43 | AC11 A19, A20 | – | **S** `a19`, `a20` (stdin hết rồi `scanf` lặp) | `TLE` (đồng hồ ≤ 3 × CPU); không treo worker; `IE` = 0 |
| TC-PE02-44 | AC11 (điều kiện chung) | – | **S** chạy **song song** từng ca tấn công A1…A20 với một bài đúng của SV khác (nộp lặp 5 lần); đo thời gian từ nộp tới `DONE` | bài đúng luôn `AC` trong **≤ 20 s**; `RestartCount` trước = sau; bộ nhớ `judge` về ≤ 100 MiB trong 30 s; đĩa ≤ 10 MiB; `docker top`/`ps` máy chủ: không tiến trình lạ |
| TC-PE02-45 | AC11 | – | **G** `-tags integration -run TestSandboxAttacks -v -timeout 10m`; QC đếm dòng `A<n> <verdict> <dấu hiệu>` | `ok`, **15** dòng; so với bảng đo của QC ở TC-PE02-33…43 (không chỉ tin test của dev) |
| TC-PE02-46 | AC12 (thông lượng) | `JUDGE_PARALLELISM=2` rồi `4` | **S** 60 bài `<bits/stdc++.h>` (1 biên dịch + 10 test) nộp cùng lúc bằng script QC; đo thời gian tới `DONE` của bài cuối và p95 | ≤ **60 s** cả hai cấu hình (PoC `4` = 18,2 s); `IE` = 0; ghi cấu hình máy |
| TC-PE02-47 | AC12 | – | **S** `Chạy thử` lúc rảnh ×20 (biên dịch + 3 test mẫu) | p95 ≤ **5 s** |
| TC-PE02-48 | AC12 | – | **S** 240 bài trong 5 phút (k6 hoặc script QC) | hết trong ≤ **8 phút**; ghi bài/phút |
| TC-PE02-49 | AC12 | – | **S** `k6 run benchmarks/load/exam-submit.js --env SCENARIO=judge_burst` | ngưỡng `judge_done_p95 < 60000`, `judge_ie == 0` |
| TC-PE02-50 | AC13 (log) | canary `QC-CANARY-<uuid>` trong mã nguồn, input, expected, `stderr` (in ra `stderr`), tên SV | **S** chạy bài đúng, WA, CE, RE, tấn công; `docker compose logs gateway worker judge \| grep -c QC-CANARY`; `select compile_log from code_submissions` | `0` ở log; `compile_log` thay đường dẫn bằng `main.c`/`main.cpp`, ≤ 8 KiB; log chỉ có `submission_id`, `attempt_id`, `exam_id`, `verdict`, thời gian, `trace_id` |
| TC-PE02-51 | AC13 | – | **G** `-tags integration -run 'TestNoSourceInLogs|TestCompileLogScrubbed' -v` | `ok` |
| TC-PE02-52 | AC14 (sức khoẻ) | – | **S** dừng judge; `curl localhost:8081/healthz`; `curl localhost:8081/readyz \| jq .judge`; đợi > 60 s xem log `warn` (đếm) | `healthz` 200 không phụ thuộc judge; `readyz` HTTP **200** với `judge:"down"`; log `warn` đúng **1** lần; bật lại → `up` trong ≤ 10 s |
| TC-PE02-53 | AC14 | – | **S** khi `down` > 60 s và có bài chờ chấm: Today của GV/TA | có việc `EXAM_GRADE_ERROR` |
| TC-PE02-54 | AC14 | – | **G** `-run 'TestJudgeProbe|TestReadyzJudgeInfoOnly' -v` | `ok` |
| TC-PE02-55 | AC15 (seccomp) | arm64 colima | **S** `.env.local` có `JUDGE_EXTRA_ARGS=-no-seccomp`; `TestSandboxAttacks` | `ok` 15 ca |
| TC-PE02-56 | AC15 (CI amd64) | – | **S** `grep -n 'judge-attacks' .github/workflows/ci.yml`; `grep -c 'no-seccomp' .github/workflows/ci.yml docker-compose.test.yml`; `gh run list --workflow ci.yml --branch sprint/5-pe --limit 1 --json conclusion,headSha` | job `judge-attacks` có, runner amd64; `-no-seccomp` **không** có ở CI; `conclusion=success` ở HEAD (QC không có amd64: ghi "đọc cấu hình + CI") |
| TC-PE02-57 | tổng | – | **S** `go vet ./... && golangci-lint run && go test -race -count=1 ./internal/judge/... ./internal/exam/... && go test -count=1 ./internal/contract/...` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Sandbox ra mạng / chạm dịch vụ nội bộ | 05, 06, 39 |
| Token judge rò hoặc thiếu | 07–10 |
| `IE` bị tính thành 0 điểm / `WA` | 15, 21, 31 |
| Mất bài khi worker chết giữa lúc chấm | 25, 26 |
| Nộp lại làm tốn tài nguyên | 29 |
| Judge chết lúc làm bài | 31, 52, 53 |
| 15 ca tấn công + 5 ca QC thêm | 33–45 |
| Tải dồn cuối giờ | 46–49 |
| Mã nguồn / test lọt log | 50, 51 |

## Câu hỏi cho BA / PM
- **Q-QC-PE02-1** — AC11 A9 chỉ đòi chặn 5 đường; QC thử thêm `/proc/self/environ`, `/etc/hostname`, `/proc/mounts`. Nếu đọc được những tệp này nhưng không có bí mật, QC ghi cảnh báo (không FAIL)? — *chờ BA*.
- **Q-QC-PE02-2** — AC12 "240 bài / 5 phút hết trong ≤ 8 phút" trên máy dev 11 nhân Apple M3: QC đo trên cấu hình `JUDGE_PARALLELISM=2` và `4` và ghi cấu hình máy; chấp nhận số đo trên colima làm chuẩn (CI amd64 không kiểm)? — *chờ BA*.
- **Q-QC-PE02-3** — TC-PE02-56: QC không có runner amd64; "đọc cấu hình CI + `gh run` xanh" có đủ cho AC15? — *chờ BA*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-weekly-exam, APPROVED). Mã tấn công: `docs/sprints/5/qc/scripts/attacks/`.

Tổng: 57 TC.
