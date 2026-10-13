# DEV handoff — US-CI-02 (TL-CI #3 shard Playwright + chia Lighthouse, #5 tách Go)
Nhánh / commit cuối: `sprint/7-p4-p5` @ `089d239` (commit `US-CI-02: …`). Chỉ đổi `.github/workflows/ci.yml` (không đụng `frontend/lighthouserc*.json`, test, `playwright.config.ts`).

## Đã làm
**#5 Go.** Job `Go` tách thành:
- `Go (static)`: `go vet`, `go vet -tags testroutes`, golangci-lint ×2 (`v2.14.0`), `sqlc diff` (`1.31.1`). Một lần mỗi lượt.
- `Go test (course)`, `Go test (auth)`, `Go test (rest)` (matrix `fail-fast: false`). Mỗi dòng: `go list -tags testroutes ./...` rồi lọc: `course` = `/internal/course(/|$)`, `auth` = `/internal/auth(/|$)`, `rest` = **mọi gói còn lại** (`grep -Ev`, tính lúc chạy nên gói mới tự rơi vào `rest`; gói con `internal/auth/x` rơi vào `auth`). Dòng không có gói nào → đỏ (tránh `go test` chạy nhầm thư mục hiện tại). Cả ba: `go test -race -tags testroutes -v`, env `TESTCONTAINERS_RYUK_DISABLED=true`, `EP_SKIP_TIMING=1`. Bước đo thời gian (`-p 1 -count=3 -tags testroutes -run 'TimingEqualized$'`, không `-race`) chạy ở dòng `course` (`./internal/course/`) và `auth` (`./internal/auth/`), không ở `rest`.
- `-v` mới thêm để đếm test: cuối bước in `go test [dòng] test cấp cao nhất: PASS=… SKIP=… FAIL=…`. Tổng ba dòng phải bằng `GO_T`. Không có bước "kiểm hợp A∪B∪C" riêng: `rest` là phần bù của `course` + `auth` trên cùng danh sách `go list` nên hợp = tất cả theo cấu trúc; dòng rỗng thì đỏ.
- Cache Go (TL-CI #4 giữ nguyên ý): `go-mod-<go.sum>` do `Go (static)` ghi; `go-build-<go.sum>-static-<run_id>` do `static` ghi; `go-build-<go.sum>-rest-<run_id>` do **chỉ** dòng `rest` ghi (bước `cache/save` cuối job, `if: success()`); `course` / `auth` / `Judge attacks` chỉ khôi phục (`restore-keys` `…-rest-` rồi `…-` để lượt đầu còn dùng bản cũ của `main`). Mỗi lượt ghi ≈ 905 MiB (2 × 452 MiB) thay vì 453 MiB.

**#3 Frontend.**
- `e2e`: matrix `shard: [1, 2, 3]`, `playwright test --grep-invert @real --shard=<i>/3`; `workers: 2`, `retries: 1` không đổi. Artifact lỗi `e2e-reports-<i>`.
- `lhci`: matrix 3 job: `simulate` (`lighthouserc.json`, đủ 7 URL), `devtools A` (`lighthouserc.devtools.json` + `--collect.url` cho `/ /chat /threads /inbox`), `devtools B` (`/gradebook /settings/llm /dev/ui`). **Không tạo tệp cấu hình con a/b:** `--collect.url` thay cả mảng `url` của tệp cấu hình (đã thử local: 2 URL → chỉ 2 URL được đo), nên ngưỡng, `numberOfRuns: 3`, `assertMatrix` (kể cả `dev/ui` = `warn`) là đúng một bản duy nhất. Artifact lỗi `lhci-reports-simulate` / `-devtools-a` / `-devtools-b` (`include-hidden-files`). Bước benchmark chẩn đoán đọc đúng thư mục của dòng (`.lighthouseci` hoặc `.lighthouseci-devtools`).
- **Không làm** `onlyCategories: ["performance"]` ở simulate (TL-CI #3c, tuỳ chọn): wall đã dưới ngưỡng, không cần đổi cái được đo.
- Mọi job mới có `timeout-minutes` (static 20, go-test 30, e2e 25, lhci 25).

## File đổi
`.github/workflows/ci.yml`; `docs/sprints/7/proposals.md` (dòng 4); handoff này.

## Chứng minh (repo `kagamikuro1024/EduPilot-v2`)
Bản nền = cùng cây mã, workflow cũ (TL-CI #1 #2 #4) + chỉ thêm `-v` và dòng đếm ở bước Go (nhánh tạm `tmp/ci-base-us-ci-02`, đã xoá): run `38400101715`. Bản mới: run `38400115687` @ `089d239`. `git diff` giữa hai cây chỉ là `ci.yml`. Mỗi bên chạy 2 lượt (attempt 1 cache Go mới đổi khoá, chỉ có bản cũ của `main` để khôi phục; attempt 2 ấm).

**(a) Số test bằng nhau.**
| | Nền (a1 / a2) | Mới (a1 / a2) | Chuẩn local `$S` |
| --- | --- | --- | --- |
| Go top-level test: PASS + SKIP | 1429 + 7 = **1436** / 1429 + 7 | course 156+1, auth 147+3, rest 1126+3 → 1429 + 7 = **1436** / như nhau | `go test -tags testroutes -list '.*' ./... \| grep -c '^Test'` = 1436 |
| Gói (`go list -tags testroutes ./...`) | 60 | 1 + 1 + 58 = 60 | 60 |
| Playwright (822 test, `--grep-invert @real`) | 582 passed + 240 skipped = **822** | a1: 270+203+108 = 581 passed + **1 flaky** (shard 1) + (3+71+166) = 240 skipped = **822**; a2: 271+203+108 = 582 passed, 240 skipped = 822 | `playwright test --list --grep-invert @real` = 822; `--shard=i/3` = 281 / 281 / 280 tổng không lọc `@real` |
| Lighthouse | simulate 7 URL × 3 = 21; devtools 7 × 3 = 21 | simulate 7 URL / 21 lượt; devtools A 4 URL / 12 lượt + B 3 URL / 9 lượt = 21 | — |
Bước đo thời gian: `auth` `ok 24.487s`, `course` `ok` (chạy, không skip) ở run `38400115687`.

**(b) Wall (từ lúc run bắt đầu đến job cuối kết thúc).**
| Lượt | Nền | Mới |
| --- | --- | --- |
| attempt 1 | 15 m 01 s | **8 m 04 s** |
| attempt 2 (ấm) | 14 m 56 s | **7 m 24 s** |
Job dài nhất ở bản mới (a1 / a2): `Playwright 1/3` 470 s / 426 s, `lhci devtools A` 379 s / 413 s, `Go test (rest)` 312 s / 244 s, `lhci simulate` 328 s / 317 s, `Go (static)` 67 s / 65 s, `Playwright 2/3` 268 s / 258 s, `3/3` 137 s / 158 s. Nền (a1): `lhci` 888 s, `Playwright` 743 s, `Go` 534 s. Số shard **không cân**: `--shard` chia theo (dự án, tệp) nên shard 1 mất 6,0–6,5 phút, shard 3 mất 1,3–1,5 phút; còn dư địa nếu cần `--shard=i/4`.

**(c) Làm đỏ từng shard / dòng → CI đỏ** (nhánh tạm `tmp/ci-red-us-ci-02`, run `38400320967`, đã xoá). Thêm `00-red-a.spec.ts` + `zz-red-a.spec.ts` (assert sai; `--list` xác nhận phủ cả 3 shard), `TestZZRed` (`t.Fatal`) ở `internal/course`, `internal/auth`, `internal/today` (thuộc `rest`), hàm thừa `zzUnusedRed` (golangci `unused`), `maxNumericValue` TBT `1` ở simulate và LCP `1` ở devtools. Kết quả: `Playwright 1/3`, `2/3`, `3/3` = failure (1/2/1 test đỏ); `Go test (course)`, `(auth)`, `(rest)` = failure (`--- FAIL: TestZZRed`); `Go (static)` = failure (`func zzUnusedRed is unused`); `lhci simulate`, `devtools A`, `devtools B` = failure (`Assertion failed` trên TBT / LCP); workflow = failure. Đã hoàn tác bằng xoá nhánh.

**Xanh ở `089d239`:** run `38400115687` attempt 1 success; attempt 2 chỉ `Judge attacks` đỏ (xem Nợ #1) nên chạy lại `--failed` → attempt 3 success.

**Tên check đổi (branch protection):** `main` **không có** branch protection (`gh api …/branches/main/protection` → 404 "Branch not protected"). Tên mới nếu sau này đặt: `Go (static)`, `Go test (course)`, `Go test (auth)`, `Go test (rest)`, `Frontend e2e (Playwright 1/3)`, `… 2/3`, `… 3/3`, `Frontend Lighthouse (simulate)`, `Frontend Lighthouse (devtools A)`, `Frontend Lighthouse (devtools B)`. Giữ nguyên: `Changes`, `Frontend (lint + build)`, `Judge attacks (amd64, seccomp bật)`. Đã bỏ: `Go`, `Frontend e2e (Playwright)`, `Frontend Lighthouse`.

## Lệnh QC chạy
```bash
gh run view 38400115687 -R kagamikuro1024/EduPilot-v2 --json jobs --jq '.jobs[]|[.name,.conclusion,.startedAt,.completedAt]|@tsv'
gh run view 38400115687 -R kagamikuro1024/EduPilot-v2 --log > run.log
sed 's/\x1b\[[0-9;]*m//g' run.log | grep -E 'go test \[|[0-9]+ (passed|skipped|flaky|failed)|Running Lighthouse|Checking assertions'
cd backend-go && go list -tags testroutes ./... | wc -l && go test -tags testroutes -list '.*' ./... | grep -c '^Test'
cd ../frontend && pnpm exec playwright test --list --grep-invert @real | tail -1
git diff --stat 1518f83 089d239 -- .github frontend backend-go
```
Lưu ý TC-07 ("ở CI không có `-v`"): bản này có `-v` ở bước `go test -race`; TC-18 ("tệp con a/b") và TC-12 (danh sách tệp cấu hình): chỉ còn `lighthouserc.json` + `lighthouserc.devtools.json`, nửa URL nằm ở `matrix.paths` trong `ci.yml`.

## AC tự đánh giá
#3 shard + chia Lighthouse ✓ · #5 static song song + matrix `-race -tags testroutes` + `rest` tính lúc chạy ✓ · không bỏ test / `-race` / bước đo thời gian / lượt Lighthouse, `numberOfRuns` 3, `workers` 2 ✓ · (a) ✓ · (b) ✓ (≤ 10 phút cả hai lượt, nhanh hơn nền) · (c) ✓.

## Nợ / chưa làm / cần hỏi
1. **Data race thật trong `internal/judge/queue.go`** (`defaults()` ghi từ `Run` và `HandleEnqueue`) làm `Judge attacks` đỏ chập chờn (`TestJudgeRetryThenDead`, run `38400115687` attempt 2). Ngoài phạm vi sprint 7 → `proposals.md` dòng 4, chờ PM.
2. Shard Playwright lệch (1,3 → 6,5 phút). Đường găng hiện là `Playwright 1/3` và `lhci devtools A`; muốn cân thì thử `--shard=i/4`.
3. Cache Go ghi 2 × 452 MiB mỗi lượt (≈ 0,9 GB); 10 GB repo giữ khoảng 8–9 lượt gần nhất.
4. TL-CI #6 (kéo trước image testcontainers) chưa làm; ≤ 0,5 phút, không đo được.

## Proposals #4 (race `judge.Queue.defaults`) — đã sửa
- Gốc: `defaults()` ghi lười từ `Run` / `HandleEnqueue` / `Tick` / `ProbeNow`. Nay `judge.NewQueue(q)` gọi `defaults()` đúng một lần khi dựng (không `sync.Once`); bốn hàm trên chỉ đọc. Đã chuyển 3 nơi dựng qua `NewQueue`: `cmd/worker/judge.go`, `cmd/worker/worker_test.go`, rig `internal/judge/queue_integration_test.go`.
- Test mới `TestQueueDefaultsNoRace` (`-race -tags integration`): `Run` + 8 `HandleEnqueue` song song trên Queue chỉ có trường bắt buộc. Trên mã cũ (worktree tạm) báo `WARNING: DATA RACE` tại `queue.go:81–125`; trên mã mới PASS.
- Chạy: `go test -race -tags 'integration testroutes' ./internal/judge -run 'TestJudge|TestQueue'` (11 pass, gồm `TestJudgeRetryThenDead`), `go test -race -tags testroutes ./internal/judge ./cmd/worker` (35 pass). Commit `fix(judge): race defaults (#4)`.
- Shard lệch: đã thử `--shard=i/4` (run `38407209313`, nhánh tạm đã xoá): `Playwright 1/4` 398 s (so 3 shard: 426–470 s), 4/4 116 s; đường găng chuyển sang `lhci devtools A` (365 s) và `Go test (rest)` 407 s (một lượt). Lợi < 1 phút, không đủ để đổi → giữ 3 shard.

## Vòng 2 (sau QC FAIL vòng 1, BUG-1: `Go test (rest)` đỏ ngẫu nhiên trên lượt nguội)
**Nguyên nhân gốc (đo, không suy đoán).** `go test ./...` mặc định `-p` = số CPU (4): bốn gói chạy cùng lúc, mỗi gói dựng container Postgres / Redis và chạy `-race` (chậm 5–10×) trên runner 4 nhân; lượt nguội còn biên dịch `-race` chồng lên. Máy quá tải làm lộ các test có chờ theo thời gian (đỏ tại: `TestCancelReachesProvider`, `TestStreamEventOrder`, `TestStreamMaxDuration`, `TestOutbox_RetryThenDead`, `TestJobs_Lifecycle`, `TestRosterInvalidatedOnMemberChange`, `TestShutdown_ForcedAfterTimeout`). Không phải một lỗi của gói nào; cũng không do chia dòng `course` / `auth`. Workflow cũ "2/2 xanh" là mẫu quá nhỏ.

**Phép đo**: cùng mã, dòng `rest` (`-race -tags testroutes -v`), mỗi cấu hình chạy song song N lượt trên runner mới, cache chỉ lấy từ `main` (nguội). Nhánh thử đã xoá.

| Cấu hình | Số lượt | Xanh | Đỏ | Thời gian một lượt |
| --- | --- | --- | --- | --- |
| `-p` mặc định (hiện trạng) | 6 | 3 | 3 | 7–8 phút |
| biên dịch trước (`-exec /bin/true`) rồi chạy | 6 | 4 | 2 | 8–9 phút |
| `-p 2` (4 mẫu) | 30 | 24 | 6 | 7–9 phút |
| `-p 2 -parallel 2` | 8 | 7 | 1 | 8–9 phút |
| **`-p 1`** | 8 | **8** | 0 | 10–12 phút |
| `-p 1 -parallel 2` | 8 | 8 | 0 | 10–13 phút |
| **`-p 1` đúng `ci.yml` đã sửa (xác nhận)** | 8 | **8** | 0 | 10–12,5 phút |

→ Tổng `-p 1`: **16/16 lượt nguội xanh** (yêu cầu ≥ 5 liên tiếp). Không thêm retry, không bỏ / sửa test nào.

**Sửa**: `.github/workflows/ci.yml` dòng `go test`: thêm `-p 1` (kèm chú thích số đo). Thời gian dòng `rest` tăng ~3–4 phút (7–8 → 10–12,5); vẫn là hai dòng song song với Playwright (≈ 9 phút) nên tổng CI tăng ~2–3 phút. Nếu cần rút lại: tách `rest` làm hai dòng (hai runner, mỗi dòng `-p 1`) — chưa làm vì chưa cần.

**Còn nợ (không chặn)**: các test chờ theo thời gian vẫn mong manh khi máy quá tải (ví dụ `TestCancelReachesProvider`: luồng giả tự xong sau 20 ms nên `provAbort` có thể không bao giờ bật nếu `Cancel` tới trễ). Đã có bản sửa xác định (luồng giả chờ tới khi `ctx` huỷ) nhưng chưa áp vì sửa test cũ phải qua proposal; với `-p 1` không còn lượt đỏ nào nên không đề xuất lúc này.

**QC chạy lại**: tạo nhánh mới từ `sprint/7-p4-p5` (cache nguội), đổi một tệp Go, push ≥ 5 lần / mở PR; dòng `Go test (rest)` phải xanh cả 5.

## Vòng 3 (#19: bỏ `-p 1`, sửa test chờ theo đồng hồ thật ở gốc)
`-p 1` chỉ che triệu chứng và đẩy `Go test (rest)` lên 10–12 phút (vượt ngưỡng TC-20). Đã **bỏ `-p 1`** (`ci.yml`) và sửa từng test đã thấy đỏ trên lượt nguội, không nới assertion, không retry, không bỏ test (commit ghi `#19`):

| Test | Nguyên nhân (đã thấy trong log CI) | Sửa |
| --- | --- | --- |
| `TestCancelReachesProvider` (`chat`) | luồng giả tự xong sau ~30 ms; `Cancel` tới trễ → trạng thái `DONE`, `provAbort` không bật | luồng giả `hold` (chờ tới khi `ctx` huỷ) + chờ sự kiện `aborted` (kênh đóng một lần) thay cho đọc cờ |
| `TestStreamEventOrder` (`chat`) | kiểm "đang STREAMING" sau `Send`, nhưng luồng 15 ms đã xong dưới tải | `gate` giữ luồng tới khi kiểm xong trạng thái |
| `TestStreamMaxDuration` (`chat`) | đọc `provAbort` ngay khi `send` trả về, trước khi goroutine giả thấy `ctx` huỷ | chờ `aborted` |
| `TestJobs_Lifecycle` (`jobs`) | thấy `RUNNING` rồi đọc `progress` trước khi mốc 25 % ghi xong | handler đóng kênh `reported` sau `Progress(25)`; test chờ kênh |
| `TestShutdown_ForcedAfterTimeout` (`httpapi`) | handler "chậm" chờ `time.After`/`ctx` của request; dưới tải request kết thúc trước hạn → `Serve = nil` | handler chặn trên kênh (không theo đồng hồ / `ctx`), `entered` báo request đã vào |
| `TestOutbox_RetryThenDead`, `TestOutbox_UnknownTopicDead` | `dead_at` ghi **trước** `XADD` dead-letter (hai hệ lưu trữ); test đọc stream ngay sau `waitDead` → 0 tin | chờ `XLEN(dead) == 1` rồi mới đọc |
| `TestRosterInvalidatedOnMemberChange` (`privacy`) | Redis chỉ có 30 ms/lệnh (`redisOpTimeout`); quá hạn → dựng lại từ DB → "cache cũ" biến mất | `Roster.OpTimeout` (0 = 30 ms như cũ); test đặt 10 s |
| `TestMaskConcurrentSameSession` (`privacy`) | cùng cơ chế: lệnh Redis quá 30 ms → ánh xạ bộ nhớ cho một số lệnh → hai email cùng `[[EMAIL_1]]` | `Masker.RedisOpTimeout` (0 = 30 ms); test đặt 10 s |
| `TestAIConfirmInvalidatedOnDecision` (`today`) | mỗi Provider của Hôm nay có hạn 150 ms (đúng SRS) → dưới tải Provider bị bỏ → danh sách rỗng | `TODAY_PROVIDER_TIMEOUT` (mặc định 150 ms); rig test đặt 30 s |
| `private-chat.spec` "auto ticket" (e2e) | "3 giờ trước" tính theo ngày lịch giờ VN → "hôm qua 21:18" trong 3 giờ đầu sau nửa đêm | đồng hồ cố định giữa ngày |
| `data-layer.spec` "sse parse + singleton" (e2e) | **lỗi thật**: chuông mở `/events` cả khi chưa có phiên | `useSSE({enabled})`, chuông chỉ nghe khi có phiên; `asDemo`/`asJwt` giả `/events` |

Mặc định sản phẩm giữ nguyên (30 ms / 50 ms / 150 ms); chỉ test tiêm hạn dài qua trường cấu hình. Hai test đo thời gian thuần (`TestDetectLinearTime`, nhóm `*TimingEqualized`) đã chạy riêng, tuần tự (`EP_SKIP_TIMING=1` ở bước chung) — không đổi.

**Đo (cold, không `-p 1`, `Go test (rest)` = `-race -tags testroutes -v`)**: vòng 1 (trước sửa Masker) 5/6 xanh (đỏ 1: `TestMaskConcurrentSameSession`); vòng 2 (sau mọi sửa) **6/6 xanh**, mỗi lượt 5,8–8,2 phút: runs `38435086769`, `38435093256`, `38435101291`, `38435108199`, `38435117559`, `38435123057` (nhánh tạm `tmp/p19b-*`, đã xoá). Máy dev (11 nhân, 24 vòng lặp bận chạy nền): 20 lượt các test trên đều xanh (`TestJobs_Lifecycle` đỏ trên mã cũ đúng như CI).
