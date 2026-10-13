# QC report — US-CI-02 (TL-CI #3 shard Playwright + chia Lighthouse, #5 tách Go)
Ngày 2026-10-13. Commit kiểm: `sprint/7-p4-p5` @ `089d239` (workflow) và `7174df1` (handoff cuối). Repo `kagamikuro1024/EduPilot-v2`. Bộ TC: `tc-US-CI-02.md`, **không sửa chữ TC** (hai điểm dev nêu là "lệch" xử lý theo AC / ý định, mục 2).

## Kết luận: **FAIL** (1 lỗi mức CAO, BUG-1)
`Go test (rest)` đỏ ngẫu nhiên trên lượt chạy nguội: **7 / 9** lượt trên 6 nhánh tạm / PR của QC đỏ ở đúng dòng này (mỗi lần một test khác nhau, đều test dùng container + thời gian), trong khi workflow cũ (một job `Go`, cùng mã) **2 / 2 xanh** cùng điều kiện. Mọi thứ còn lại PASS. Một lượt đỏ ngẫu nhiên ở PR vào `main` là "phá cái đang chạy" (`CLAUDE.md` luật 8): chưa nhận.

## 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| TC-01 tổng test Playwright | PASS | `089d239` a1: 270+203+108 = 581 passed + 1 flaky (shard 1) = 582, skipped 3+71+166 = 240, tổng 822; a2: 271+203+108 = 582, skipped 240. Nền (run `38400101715`): 582 passed + 240 skipped = 822. `--list` ở máy QC = 822. 1 flaky ghi nhận, retry xanh |
| TC-02 hợp shard = tập cũ | PASS | `playwright test --list --grep-invert @real --shard=i/3` i=1..3: 274 / 274 / 274 test; dòng trùng giữa shard = 0; hợp sort = `--list` không shard (diff rỗng). N = 3 ≥ 2 (PM) |
| TC-03 giữ cấu hình | PASS | lệnh `playwright test --grep-invert @real --shard=${{matrix.shard}}/3`; `playwright.config.ts` không đổi (`workers: 2`, `retries: 1`); không `--max-failures`, `continue-on-error`, `\|\| true` |
| TC-04 artifact riêng | PASS | run đỏ `38406883998`: `e2e-reports-1/2/3`, `lhci-reports-simulate / -devtools-a / -devtools-b` (devtools-a/b 4,7 và 6,4 MB, có `.lighthouseci`) |
| TC-05 `-race -tags testroutes` | PASS | cả ba dòng: `go test -race -tags testroutes -v $pkgs`; `TESTCONTAINERS_RYUK_DISABLED`, `EP_SKIP_TIMING` có ở mỗi dòng; không `-short`, `-run`, `-failfast` |
| TC-06 tổng gói Go | PASS | `ok` mỗi dòng: course 1, auth 1, rest 58 → 60, trùng 0; bằng tập `ok` của nền (60, `diff` rỗng) và bằng `go list -tags testroutes ./...` (60). Lưu ý: `go list ./...` không tag = 59; dòng Go chuẩn trước tách đã chạy `-tags testroutes ./...` nên 60 là đúng tập |
| TC-07 tổng test Go | PASS (bằng chứng tương đương, xem 2a) | `go test -tags testroutes -list '.*' ./... \| grep -c '^Test'` = **1436**. CI: PASS+SKIP = 156+1, 147+3, 1126+3 = **1429 + 7 = 1436**, bằng nền 1429 + 7. FAIL = 0 ở cả hai lượt xanh |
| TC-08 gói mới tự rơi vào "rest" | PASS | nhánh tạm `tmp-qc-newpkg`: `internal/zzqcprobe` → `ok` đúng một lần, ở `Go test (rest)`; không sửa `ci.yml` |
| TC-09 gói con | PASS | `internal/auth/zzqcsub` chỉ ở `Go test (auth)`; `internal/course/zzqcsub` chỉ ở `Go test (course)`; không ở `rest` |
| TC-10 bước đo thời gian | PASS | `auth`: `ok …/internal/auth 24.487s`; `course`: `ok …/internal/course 1.832s` (không skip, `-p 1 -count=3`, không `-race`); `rest` không chạy bước này |
| TC-11 phần tĩnh | PASS | `Go (static)`: vet, vet testroutes, golangci-lint ×2 (`v2.14.0`), `sqlc diff` (`1.31.1`), một lần mỗi lượt, 56–124 s |
| TC-12 ngưỡng Lighthouse | PASS (bằng chứng tương đương, xem 2b) | `git diff 1518f83 089d239 -- frontend` rỗng: hai tệp `lighthouserc*.json` không đổi → mọi assertion, `numberOfRuns: 3`, `median`, `dev/ui = warn` giữ nguyên; không `onlyCategories` mới (dev không làm #3c) |
| TC-13 đủ URL | PASS | simulate 7 URL / 21 lượt; devtools A (`/ /chat /threads /inbox`) 4 URL / 12 lượt; B (`/gradebook /settings/llm /dev/ui`) 3 URL / 9 lượt. Hợp A∪B = 7 URL cũ, không giao nhau, 21 lượt |
| TC-14 quy tắc theo URL | PASS | run đỏ `38406883998` (ngưỡng LCP = 1): cả A và B báo `✘ largest-contentful-paint … assertion` → assertion khớp URL của cả hai nửa. `dev/ui` ở nửa B chung cùng `assertMatrix` (tệp không đổi) |
| TC-15 chạy hợp lệ | PASS | cả 3 job in `Healthcheck passed!`, `Done running Lighthouse!`, `All results processed!`; `CHROME_PATH` đặt |
| TC-16 test Playwright đỏ → CI đỏ | PASS | `tmp-qc-red`: `00-qc-red.spec.ts` + `zz-qc-red.spec.ts` (xác nhận `--list` phủ cả 3 shard); `Playwright 1/3, 2/3, 3/3` = failure (cả sau retry); workflow failure |
| TC-17 Go đỏ → CI đỏ | PASS | `TestZZQCRed` trong `course`, `auth`, `today`: `Go test (course)`, `(auth)`, `(rest)` failure, mỗi dòng báo `FAIL … internal/<gói>` |
| TC-18 Lighthouse vượt ngưỡng → đỏ | PASS | TBT = 1 ở simulate và LCP = 1 ở devtools: `lhci simulate`, `devtools A`, `devtools B` failure (`Assertion failed`) |
| TC-19 | KHÔNG ÁP DỤNG | không có bước kiểm hợp riêng; hợp = phần bù theo cấu trúc (cùng danh sách `go list`) và dòng rỗng thì đỏ; TC-08/09 chứng minh đủ |
| TC-20 thời gian (≤ 10 phút, nhanh hơn nền) | PASS (có điều kiện, xem BUG-1) | dev, push đổi `.github/` (chạy đủ), run `38400115687`: a1 **8 m 05 s**, a2 **7 m 24 s**. Nền cùng cây (`38400101715`): 15 m 01 s / 14 m 56 s. Lượt QC nguội: `tmp-qc-newpkg` 8 m 47 s, `tmp-qc-red` 8 m 45 s, `tmp-qc-ident2` 8 m 53 s; hai lượt `tmp-qc-pr` (push 10 m 14 s, `pull_request` 12 m 14 s) vượt 10 phút nhưng bị nhiễu: tôi chạy 6 nhánh cùng lúc → job chờ runner (job bắt đầu lệch), nên **không** chấm lượt này là vượt. Dài nhất: `Playwright 1/3` 470–538 s |
| TC-21 PR vào `main` chạy đủ | PASS | PR #1 (nháp, đã đóng): mọi job chạy, không `skipped`: `Changes`, `Go (static)`, `Go test` ×3, `Judge attacks`, `Frontend (lint + build)`, `Playwright` ×3, `Lighthouse` ×3. (Lượt đó đỏ do BUG-1) |
| TC-22 lọc đường dẫn | PASS | `tmp-qc-f1` chỉ đổi `backend-go/`: toàn bộ job frontend `skipped` (cả ma trận, một check mỗi ma trận, không mở dòng). `tmp-qc-f2` chỉ đổi `frontend/`: `Go (static)`, `Go test`, `Judge` `skipped` |
| TC-23 branch protection | PASS | `main` không có protection (`404 Branch not protected`), `rulesets` = 0. Tên check mới có trong handoff |
| TC-24 không bỏ cổng | PASS | `diff` workflow: không bước nào bị xoá; `-race`, đo thời gian, selftest, antipatterns, lint selftest, build, benchmark chẩn đoán, `Judge attacks` giữ nguyên. Không `continue-on-error`, `if: false`, `\|\| true` mới |
| TC-25 cache Go | PASS | không có `Unable to reserve` / `Failed to save`; `rest` ghi 452 MiB/lượt, `static` 452 MiB/lượt; tổng cache repo 4,1 GB / 10 GB (các nhánh tạm sẽ bị dọn) |
| TC-26 phạm vi diff | PASS (ghi chú) | `.github/workflows/ci.yml` là thay đổi của dev. Cùng khoảng `1518f83..HEAD` còn có `73a9cbe fix(judge): race defaults (#4)` (4 tệp Go + test) — không phải của dev CI, ghi theo `proposals.md` #4; test Go / Playwright / `playwright.config.ts` / ngưỡng không bị sửa |
| TC-27 an toàn workflow | PASS | `permissions` không đổi (`contents: read`); không `pull_request_target`, secret; `uses:` mới ghim `@v5`/`@v7` như cũ |
| TC-E1 flaky | PASS | a1 `1 flaky` ở shard 1, job xanh (retry hoạt động); test đỏ hai lần vẫn đỏ (TC-16) |
| TC-E2 timeout | PASS (ghi chú) | mọi job mới có `timeout-minutes` (static 20, go-test 30, e2e 25, lhci 25). `Judge attacks` và `Changes` không có (không phải job mới) |
| TC-E3 rerun failed | PASS | `gh run rerun --failed` chạy lại đúng job đỏ; job tổng xanh khi job đó xanh (run `38400115687` a3, `38409043382` a2). Các job xanh giữ nguyên |
| TC-E4 shard rỗng | PASS | `--shard=30/30` trên 822 test: 27 test, lệnh không lỗi; hợp lệ |

## 2. Hai điểm dev báo "lệch so với TC" — chấm theo ý định
a. **TC-07 (`-v`)**: TC nói "ở CI không có `-v`, đối chiếu theo gói". Bản mới có `-v` và in `PASS/SKIP/FAIL` mỗi dòng. Ý định TC là tổng test không giảm, không gói nào bị bỏ. Bằng chứng tương đương **mạnh hơn**: tổng test cấp cao nhất 1429 PASS + 7 SKIP = 1436, bằng nền (cùng cách đếm, run `38400101715`) và bằng `-list` ở máy QC; tập gói `ok` bằng nhau (TC-06). Đạt, không sửa TC.
b. **TC-12/14/18 (không có tệp a/b)**: dev dùng `--collect.url` trên cùng tệp cấu hình. Ý định TC là mọi assertion cũ vẫn chấm trên đủ URL. Bằng chứng: tệp cấu hình không đổi (diff rỗng); log chấm 4 + 3 URL, 12 + 9 lượt; run đỏ có ngưỡng hạ về 1 → cả hai nửa đỏ ở đúng assertion LCP, simulate đỏ ở TBT. Tức là assertion có hiệu lực ở cả hai nửa. Đạt, không sửa TC. (Muốn đổi chữ TC → ghi `proposals.md`, PM quyết; tôi không đề xuất.)

## 3. Lỗi
**BUG-1 (CAO): `Go test (rest)` đỏ ngẫu nhiên trên lượt nguội (nhánh mới / PR).**
- Tái hiện: tạo nhánh mới từ `sprint/7-p4-p5` (cache build chỉ lấy được từ `main`), thêm một tệp Go rỗng vào gói mới (`backend-go/internal/zzqcid/f.go`; cần đổi tệp ngoài `docs/` để CI chạy), push. Hoặc mở PR vào `main`.
- Quan sát (cùng mã, `-race -tags testroutes -v`, dòng `rest`):

| Run | Nhánh | `rest` | Test đỏ |
| --- | --- | --- | --- |
| `38406883998` | `tmp-qc-red` | đỏ | `TestCancelReachesProvider` (chat; kèm `TestZZQCRed` cố ý) |
| `38406888709` | `tmp-qc-newpkg` | đỏ | `TestExamResultTodayItems` (today) |
| `38406934888` | `tmp-qc-f1` | đỏ | `TestFromDraftNoMessageStored` (chat) |
| `38406944205` | `tmp-qc-pr` push | đỏ | `TestMaskMappingTTL` (privacy) |
| `38406956048` | `tmp-qc-pr` PR | đỏ | `TestOutbox_RetryThenDead` / `TestJobs_Lifecycle` (outbox, worker) |
| `38409043382` | `tmp-qc-ident` a1 / a2 | đỏ / xanh | `TestExamResultTodayItems` / — |
| `38410139811` | `tmp-qc-ident2` | đỏ | `TestCancelReachesProvider` |

  Tức **7 đỏ / 9 lượt**. Đối chứng: workflow cũ (một job `Go`, cùng mã + 1 gói rỗng): `tmp-qc-old` và `tmp-qc-old2` job `Go` **xanh 2/2**. Trên nhánh sprint (cache ấm) `rest` xanh 4/4 (dev `38400115687` a1/a2, `38407161411`, `38407209313`).
- Đặc điểm: mỗi lượt một test khác nhau; đều là test dùng Postgres/Redis container, có chờ theo thời gian ("`chat_sessions` should be zero but was 2", "mỗi lượt EXPIRE lại", "`EXAM_GRADE_ERROR` không có"). Đây là phụ thuộc thời gian / dùng chung container giữa các gói song song, lộ ra khi `rest` gom 58 gói chạy cùng lúc trên một máy. [SUY LUẬN, chưa chứng minh: nguyên nhân gốc; tôi không đọc code test.] Tách `course`/`auth` ra khỏi một job chung đã đổi nhịp chạy của 58 gói còn lại.
- Hậu quả: PR vào `main` đỏ ngẫu nhiên, không phải do mã của PR. Cách làm chạy "đỏ hai lần liên tiếp rồi xanh" che mất lỗi thật.
- Đề xuất cho dev: tái hiện bằng đúng bước trên (không cần thay mã); xác định test nào dùng chung container; hoặc chia `rest` thành hai dòng để giảm số gói song song; AC "không phá cái đang chạy" chỉ đạt khi `rest` xanh ổn định trên nhánh nguội (≥ 5/5). Đề xuất chép vào `proposals.md` nếu PM muốn xử lý ngoài story.

**Ghi chú (không phải lỗi của story):** `Judge attacks` đỏ chập chờn do data race trong `internal/judge/queue.go` (`proposals.md` #4) — đã sửa ở `73a9cbe`; lượt `tmp-qc-old` (mã trước sửa) đỏ ở đó đúng như dự báo. Shard Playwright lệch (1: 470–538 s, 3: 130–160 s); dev đã thử 4 shard, lợi < 1 phút, không đổi.

## 4. Dọn
Nhánh tạm `tmp-qc-red`, `-newpkg`, `-f1`, `-f2`, `-pr`, `-ident`, `-ident2`, `-old`, `-old2` đã xoá; PR #1 đóng; worktree tạm xoá.
