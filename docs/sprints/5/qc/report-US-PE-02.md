# Báo cáo QC — US-PE-02 (máy chấm go-judge trong sandbox, hàng đợi riêng)
**Kết luận: PASS có điều kiện** — không FAIL; điều kiện: (1) máy QC chỉ có arm64 + `-no-seccomp` (D58): **seccomp ở amd64 và job CI `judge-attacks` KHÔNG KIỂM ĐƯỢC** (CI không chạy vì billing); (2) các TC cần handler `Chạy thử` / `Nộp lời giải` / Today (TC-27, 31, 53 …) chờ PE-06 / PE-08 — dev ghi đúng trong handoff; (3) cảnh báo L1 (A16). Bản chấm `c5f93d7` (mã PE-02 = `9a7ad13`). QC **tự viết mã tấn công** (`docs/sprints/5/qc/scripts/attacks/a01…a20`) và driver `p502-attacks.py` gọi **thẳng go-judge** theo SRS 4.5.2 (không dùng `internal/judge` của dev); judge là ảnh dựng từ `deploy/judge` chạy trong mạng `internal` riêng của QC (`qc5_judge_net`, `--privileged --cgroupns=host`, `cpus=2`, `mem=2g`, `JUDGE_PARALLELISM=2`, `-no-seccomp`), client là container `curl` cùng mạng. Máy: colima 4 CPU / 4 GiB, arm64.

## Lỗi / lệch
- **L1 (TC-42 — cảnh báo, không FAIL).** Với `-no-seccomp` (arm64, D58) `a16-ptrace-mount.c` in `ptrace=0 mount=-1 unshare=0 chroot=-1`: **`ptrace(PTRACE_TRACEME)` và `unshare(CLONE_NEWUSER)` thành công** trong sandbox; `mount` và `chroot` bị chặn. TC mong cả bốn `-1` (ca A16 do QC thêm, không thuộc 15 ca của AC11). Đây là hệ quả của việc tắt seccomp, không phải lỗi mã; **phải kiểm lại ở amd64 (seccomp bật)** trước khi dùng thật.
- **L2 (QC).** Ba mã tấn công đầu của QC lỗi biên dịch / bị trình biên dịch tối ưu hoá mất ý đồ (`a06` bị `-O2` xoá `malloc+memset` ⇒ lần đầu ra `AC`; `a08`, `a16` thiếu `#define _GNU_SOURCE`/`sys/types.h`); QC đã sửa nguồn của mình và chạy lại — các số dưới đây là lượt chạy lại.
- **L3 (TC-12).** Không gieo tệp `exec.Command` để thử "đỏ"; chỉ kiểm `grep` = 0 (QC chưa thử gieo).
- **L4 (CI).** Như báo cáo PE-01: GitHub Actions không chạy được (billing) ⇒ TC-56 chỉ đọc cấu hình.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `docker compose -f docker-compose.local.yml -f docker-compose.test.yml config`: đúng 1 dịch vụ `judge`: `privileged:true`, `cgroup:"host"`, `ports:null`, `shm_size` 256 MiB, `cpus:2`, `mem_limit` 2 GiB |
| 02 | PASS | dịch vụ khác có `privileged:true` = **0**; env của `judge` chỉ `JUDGE_EXTRA_ARGS`, `JUDGE_PARALLELISM`, `JUDGE_TOKEN`; `start.sh` không `-enable-debug`; `-no-seccomp` chỉ qua `JUDGE_EXTRA_ARGS` |
| 03 | PASS (một phần) | ảnh `deploy/judge` dựng thành công; `go-judge` v1.13.0 khởi động; (`g++`/`gcc` 14.2.0 và `trixie`: đọc `Dockerfile`, biên dịch C/C++ thật chạy được) |
| 04 | PASS | `docker compose config`: gateway và worker không `privileged`, không `cgroup: host`; worker không có `gcc` / `g++` trong Dockerfile |
| 05 | PASS | `a10b`: `postgres`, `redis`, `gateway`, `host.docker.internal`… đều `NOHOST`; `a10`: `socket=3 connect=-1` tới `1.1.1.1:53` (mạng `internal`) |
| 06 | PASS | `judge_net` `Internal=true`, thành viên đúng 2 (`judge`, `worker`); gateway không ở `judge_net`; `judge` không `ports`, 0 volume |
| 07 | PASS | `POST /run` không token `401`, token sai `401`, token đúng `400` (không 401); `GET /version` không token `200` |
| 08 | PASS (dev) | `TestJudgeMemLimitCheck`, khởi động worker thiếu / ngắn `JUDGE_TOKEN` (10 ca) PASS trong lượt tích hợp; `start.sh` thoát 1 khi < 16 ký tự |
| 09 | PASS (một phần) | log `qc5-judge` sau toàn bộ lượt tấn công: token **0** lần (go-judge in `"token":"***"`); log gateway / worker: dev `TestTokenNeverLogged` PASS |
| 10 | PASS | `TestClientRequiresToken`, `TestTokenNeverLogged` PASS |
| 11 | PASS | `TestClientCompileRequest`, `TestClientRunRequest`, `TestClientDeletesCachedFile`, `TestClientDeadlineAndSemaphore` PASS (`-race`) |
| 12 | PASS (L3) | `grep -rnE 'os/exec\|syscall\.Exec\|exec\.Command' internal/judge internal/exam` ngoài test = **0** |
| 13 | PASS (một phần) | worker không có trình biên dịch (đọc Dockerfile); không đo `docker diff` |
| 14 | PASS | 20 mã QC chạy trực tiếp: AC / TLE / MLE / OLE / RE / CE đúng bảng 4.5.4; `IE` = 0 |
| 15 | PASS (dev) | `TestVerdictMapping` PASS (có ca `lstat stdout: operation not permitted` → `IE`); QC không tự làm proxy lỗi |
| 16 | PASS | `TestVerdictMapping` PASS |
| 17 | PASS (dev) | QC chưa tự nộp ca checker; `TestCheckerExact`, `TestCheckerTokens`, `TestCheckerFloatEps` PASS |
| 18 | PASS | như 17 |
| 19–32 | PASS (dev, một phần) | `TestJudgeConsumerIdempotent`, `TestJudgeLeaseNoDoubleRun`, `TestJudgeLongRunNotReclaimed`, `TestJudgeLeaseExpiryRequeue`, `TestJudgeBackoffNextAttemptAt`, `TestJudgeRetryThenDead`, `TestSupersedeQueued`, `TestRunningNotCancelled`, `TestSubmissionResultAssembly`/`NoEarlyStop`/`ZeroTotalWeight`/`OnlyApprovedTests`, `TestJudgeUpKeyLifecycle` PASS (Postgres + Redis thật, judge giả). **Chờ:** `TestRunWhenJudgeDown503`, `TestSubmitWhenJudgeDownQueued`, `TestIENeverScoresZero` (PE-06 / PE-08, TC-27, 31, 32); QC không kiểm tay `kill -9` worker (TC-25) ở lượt này |
| 33 | PASS | A1 `TLE` (CPU 1001 ms); A2 `TLE` (CPU 1004 ms, stdout bị cắt); RAM `judge` ≈ 53–63 MiB |
| 34 | PASS | A3 `OLE` (5 MB), A4 `OLE` (1 GB; CPU 21 ms) |
| 35 | PASS | A5 `MLE` (256 MiB), A6 `MLE` (2 GiB, sau khi QC sửa mã); `RestartCount` = **0** |
| 36 | PASS | A7 `RE` (Signalled), A18 `RE` (tràn stack, 8 MiB) |
| 37 | PASS | A8 `FORKED=0` (fork bị chặn); A17 `THREADS=0`; `procPeak` = 1 |
| 38 | PASS | A9: `CHAN /etc/shadow`, `/etc/passwd`, `/proc/1/environ`, `/opt/go-judge…` …; không lộ biến môi trường / token (đã kiểm các đường in ra) |
| 39 | PASS | A10 `connect=-1`; A10b mọi tên `NOHOST` |
| 40 | PASS | A11 `CE` (`cc1` bị kill, ≈ 0,9 s); A12 `CE` do `TLE` biên dịch sau **10,1 s** (≤ 25 s) |
| 41 | PASS | A13 `MLE` (ghi 1 GB vào `/tmp` bị chặn ở bộ nhớ); A15: bản **ghi** `WROTE /tmp/leak`, `/w/leak`, `./leak`; bản **đọc** ngay sau `CHAN` ×3 (không `LEAKED`) |
| 42 | PASS có điều kiện (L1) | A14 `RE`, không `/bin/sh`; A16 `mount=-1 chroot=-1` nhưng `ptrace=0`, `unshare=0` (arm64 `-no-seccomp`) |
| 43 | PASS | A19 `TLE` sau 3,0 s (= 3 × CPU 1 s); A20 `TLE` (CPU 1004 ms); `IE` = 0 |
| 44 | PASS | 10 bài đúng (`a+b`) chạy **song song với 16 ca tấn công** (A1, A2, A4, A5, A6, A8, A19, A20 × 2): 10 / 10 `AC`, chờ tối đa **3,5 s**; `RestartCount` = 0, RAM `judge` về 63 MiB |
| 45 | PASS | `TestSandboxAttacks` PASS trong lượt tích hợp (dev, 15 ca); bảng đo của QC ở TC-33…43 khớp |
| 46 | PASS | **60 bài × (1 biên dịch `<bits/stdc++.h>` + 10 test)** cùng lúc, `JUDGE_PARALLELISM=2`: tổng **26,4 s**, p95 26,3 s, 60 `AC`, 0 `IE` (≤ 60 s) |
| 47 | KHÔNG KIỂM ĐƯỢC | `Chạy thử` chưa có handler (PE-06) |
| 48 | PASS | **240 bài** × (1 + 10): **111,9 s** (≈ 129 bài/phút), 240 `AC` (≤ 8 phút). Chỉ là driver QC vào go-judge, không gồm hàng đợi / worker |
| 49 | KHÔNG KIỂM ĐƯỢC | `benchmarks/load/exam-submit.js` chưa có |
| 50 | PASS (một phần) | log `qc5-judge` 0 token; canary trong mã / log gateway–worker: `TestNoSourceInLogs` PASS (QC chưa tự gieo canary) |
| 51 | PASS | `TestNoSourceInLogs`, `TestCompileLogScrubbed` PASS |
| 52 | PASS (dev) | `TestJudgeProbe`, `TestReadyzJudgeInfoOnly` PASS; QC chưa tự `docker stop judge` trên stack đầy đủ |
| 53 | KHÔNG KIỂM ĐƯỢC | nguồn việc `EXAM_GRADE_ERROR` của "Hôm nay" thuộc story sau |
| 54 | PASS | như 52 |
| 55 | PASS | arm64 + `-no-seccomp`: `TestSandboxAttacks` PASS |
| 56 | KHÔNG KIỂM ĐƯỢC | job `judge-attacks` có trong `ci.yml`, runner `ubuntu-24.04`, không `-no-seccomp`; **CI không chạy** (billing); amd64 / seccomp chưa kiểm |
| 57 | PASS | `go vet` rc=0; `golangci-lint` 0 issues; `sqlc diff` rc=0; `go test -race -tags integration ./...` rc=0 (783 PASS, 0 FAIL, 3 SKIP) — xem `report-US-PE-01.md` |

## Việc sau
- **Dev / chủ dự án:** chạy `TestSandboxAttacks` + A16 (ptrace / unshare) trên amd64 với seccomp bật (CI `judge-attacks` khi billing được xử lý).
- **QC:** chấm lại TC-27, 31, 32, 47, 49, 53 khi có handler (PE-06 / PE-08); TC-25 (`kill -9` worker) và TC-15 (proxy lỗi) tay trên stack đầy đủ ở cổng PE. Scripts: `scripts/p502-attacks.py`, `scripts/p502-mix.py`, `scripts/attacks/`.
