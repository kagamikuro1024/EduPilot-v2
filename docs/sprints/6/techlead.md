# Sprint 6: kênh Tech Lead

## TL-CI (PM, 2026-10-11): rút thời gian CI

**Câu hỏi:** CI đang mất khoảng 17–27 phút mỗi lượt. Đổi gì để nhanh hơn mà không giảm độ phủ test, không bỏ `-race`, không bỏ cổng nào?

**Kết luận trước:**
- Hai lượt gần nhất mất **27,4 phút**. Đường găng là job `Frontend`, nối tiếp Playwright 12,2 phút, Lighthouse simulate 4,2 phút và Lighthouse devtools 9,2 phút. Job `Go` chỉ mất 8,6 phút.
- Hai việc đầu đáng làm ngay, rẻ và không đụng test:
  - (1) Lọc theo đường dẫn: push chỉ đổi backend thì không chạy phần frontend, lượt còn **≈ 8,6 phút**.
  - (2) Tách `Frontend` thành các job chạy song song: lượt đổi frontend còn **≈ 15 phút**.
- Làm thêm (3) shard Playwright và chia Lighthouse, cùng (4)–(5) cho Go, thì cả hai loại lượt về **≈ 6–8 phút**.

Nguồn số liệu:
- `.github/workflows/ci.yml` trên `origin/sprint/6-p3-p8`.
- Run `38106540620` (success) và `38104867232` (Go đỏ ở `TestPrecheckRateLimit`), đo bằng `gh run view <id> --json jobs` và `gh run view 38106540620 --log`.
- `gh cache list`.

### 1. Thời gian từng bước (hai lượt gần nhất)

| Job / bước | Run 38106540620 | Run 38104867232 | Ghi chú |
| --- | --- | --- | --- |
| **Frontend (tổng)** | **27 m 22 s** | **27 m 29 s** | đường găng |
| checkout + pnpm + node + install | 21 s | 19 s | cache pnpm trúng |
| lint / ui antipatterns / lint selftest | 19 s / 0 s / 13 s | 19 s / 0 s / 14 s | |
| build (production) / build (gate) | 21 s / 9 s | 21 s / 9 s | |
| cache + cài Chromium | 4 s + 11 s | 3 s + 10 s | cache trúng |
| **playwright test** | **12 m 12 s** | **12 m 22 s** | 822 test, 2 worker: 580 passed, 240 skipped, 2 flaky |
| **lighthouse ci (simulate)** | **4 m 11 s** | **4 m 13 s** | 7 URL × 3 lượt ≈ 34,5 s / URL; thu đủ mọi nhóm mục, chỉ chấm hiệu năng |
| **lighthouse ci (devtools, LCP)** | **9 m 14 s** | **9 m 16 s** | 7 URL × 3 lượt ≈ 79 s / URL |
| **Go (tổng)** | **8 m 35 s** | **11 m 40 s** (đỏ) | |
| setup-go | 14 s | 12 s | "Cache hit", nhưng xem mục 2.4 |
| go vet / vet testroutes | 61 s / 14 s | 86 s / 19 s | lượt 1 tải lại **37 module** sau khi "cache trúng" |
| golangci-lint ×2 | 10 s + 10 s | 15 s + 14 s | |
| setup sqlc + sqlc diff | 3 s | 3 s | |
| **go test -race (testroutes)** | **5 m 54 s** | 9 m 00 s (đỏ) | 42 gói; dài nhất: `course` 187 s, `auth` 187 s, `exam` 123 s, `contract` 102 s, `today` 44 s; tổng giờ chạy các gói 1.091 s |
| go test đo thời gian (`-p 1 -count=3`) | 38 s | bỏ qua | |
| **Judge attacks (tổng)** | **2 m 35 s** | **2 m 30 s** | dựng image judge + 15 đòn tấn công |

Hai mươi lượt gần nhất trên nhánh: các lượt chạy hết (không bị huỷ) mất 10–27 phút; lượt đỏ ở Go vẫn chờ `Frontend` chạy xong mới kết thúc. Từ 2026-10-10, trong các commit đụng mã (không tính `docs/`): **16 commit chỉ đổi backend**, 1 commit chỉ đổi frontend, 16 commit đổi cả hai.

### 2. Thay đổi đề xuất (xếp theo lợi ích / chi phí)

| # | Thay đổi | Tiết kiệm (wall) | Chi phí | Rủi ro và cách giữ |
| --- | --- | --- | --- | --- |
| 1 | **Lọc theo đường dẫn ở mức job.** Thêm job `changes` (~10 s) chạy `git diff --name-only` giữa `github.event.before` và `github.sha`, trả hai cờ. `fe` khi đổi `frontend/**`, `pnpm-lock.yaml`, `package.json`, `pnpm-workspace.yaml`, `scripts/ui-antipatterns.sh`, `scripts/lint-selftest.sh`. `go` khi đổi `backend-go/**`, `deploy/judge/**`. Đổi `.github/workflows/**` thì bật cả hai. Các job phía frontend có `if: needs.changes.outputs.fe == 'true'`; `Go` và `Judge attacks` có `if: …go…`. **Luôn chạy đủ** ở `pull_request` vào `main` và khi `before` là `000…0` (nhánh mới / force-push). | Push chỉ đổi backend: **27,4 → ≈ 8,6 phút** (−19 phút, khoảng nửa số push của sprint). Push chỉ đổi frontend: không chờ Go. | ≈ 25 dòng YAML, không thêm action ngoài (`git diff` sẵn có; không dùng `dorny/paths-filter`, để khỏi thêm phụ thuộc ngoài bảng `ARCHITECTURE.md`). | Đã kiểm: frontend **không đọc tệp nào** của `backend-go/` (chỉ là chú thích "hợp đồng thật" trong `features/*/api.ts`, `e2e/support/*`); Playwright chạy với máy chủ giả (`--grep-invert @real`). Cổng không mất: PR vào `main` luôn chạy đủ. Job bị bỏ qua sẽ hiện "skipped", tức success nếu branch protection đòi check theo tên. Nếu sau này frontend sinh kiểu từ `openapi.yaml`, thêm `backend-go/api/openapi.yaml` vào bộ lọc `fe`. |
| 2 | **Tách `Frontend` thành ba job song song:** `fe-check` (lint, antipatterns, lint selftest, build production), `e2e` (build:gate + Playwright), `lhci` (build:gate + hai lượt Lighthouse). Mỗi job tự cài (≈ 1,2–1,5 phút, cache pnpm/Chromium đã trúng). | Lượt đổi frontend: **27,4 → ≈ 15 phút** (`lhci` ≈ 1,5 + 4,2 + 9,2). | Chép khối setup sang ba job (hoặc composite action cục bộ `.github/actions/fe-setup`). Tốn thêm ≈ 3 phút runner, miễn phí với repo public. | Không đổi test nào. `upload reports` phải có ở cả `e2e` và `lhci` (thư mục khác nhau). Đổi tên job thì phải sửa tên check bắt buộc của branch protection `main`, nếu có. |
| 3 | **Chia đường găng còn lại của frontend:** (a) Playwright `--shard=${{ matrix.shard }}/3`, giữ `workers: 2`, `retries: 1`; (b) Lighthouse thành matrix: `simulate` 1 job và `devtools` 2 job, mỗi job một nửa URL (hai tệp cấu hình con `lighthouserc.devtools.a.json` / `.b.json` cùng ngưỡng, cùng `numberOfRuns: 3`); (c) cấu hình simulate thêm `"onlyCategories": ["performance"]`, vì hiện thu cả a11y / SEO / best-practices mà chỉ chấm TBT / CLS / dung lượng JS. | `e2e` ≈ 1,5 + 12,2 / 3 ≈ **5,6 phút**; `lhci-devtools` ≈ 1,5 + 4,6 ≈ **6,1 phút**; `lhci-simulate` ≈ 1,5 + 4,2, còn khoảng 3–3,5 nếu (c) cắt 20–30 % [SUY LUẬN: chưa đo phần thời gian của các nhóm không chấm]. Lượt đổi frontend: **≈ 15 → ≈ 6–7 phút**. | Matrix 3 dòng + tách URL; ≈ 20 dòng. Runner: e2e 3 × 5,6 ≈ 17 phút, lhci ≈ 17 phút (miễn phí, repo public). | Không giảm độ phủ: shard chia 822 test, mỗi test vẫn chạy đúng một lần. Lighthouse vẫn 7 URL × 3 lượt, vẫn trung vị, ngưỡng không đổi. **Không** giảm `numberOfRuns` 3 → 1: trung vị của 3 lượt là điều kiện chống chập chờn của cổng US-PU-06; đổi là đổi AC, phải qua `proposals.md`. Phải đặt `include-hidden-files` cho thư mục báo cáo của từng shard / từng nửa URL. |
| 4 | **Sửa cache Go.** Khoá `setup-go-…-<hash go.sum>` tạo ngày 2026-10-09 **thiếu module**: lượt 38106540620 "Cache hit" (224 MB) mà `go vet` vẫn `go: downloading` 37 module. Cache cũng **không bao giờ được ghi lại** ("Cache hit occurred on the primary key … not saving cache") nên `GOCACHE` cũng cũ từ 09/10. Nguyên nhân [SUY LUẬN, khớp mốc giờ]: `Judge attacks` dùng cùng khoá, xong sớm (2,5 phút) nên ghi trước, chỉ với module của `./internal/judge`. Sửa: `Judge attacks` đặt `cache: false` (hoặc chỉ khôi phục); job Go tự quản bằng `actions/cache`: `GOMODCACHE` khoá theo `go.sum`; `GOCACHE` khoá `go-build-<go.sum>-<run_id>` với `restore-keys: go-build-<go.sum>-` để lượt sau dùng bản gần nhất. | ≈ 45–70 s ở `go vet` (bỏ tải module) + một phần biên dịch `-race` ở bước test [SUY LUẬN: log không tách biên dịch / chạy; trần ≈ 5 m 54 s − 187 s]. Ước **−1 đến −2 phút** cho job Go. | ≈ 10 dòng. Mỗi lượt ghi thêm ≈ 230 MB cache (giới hạn 10 GB / repo; GitHub tự xoá bản cũ). | Không đụng test. Cache sai chỉ làm chậm, không làm sai kết quả (Go tự kiểm hash). |
| 5 | **Tách job Go:** `go-static` (vet ×2, golangci-lint ×2, sqlc diff ≈ 1,7 phút) song song với `go-test` matrix 3 dòng, cả ba đều `-race -tags testroutes`: (A) `./internal/course/...` + bước đo thời gian của `course`; (B) `./internal/auth/...` + bước đo thời gian của `auth`; (C) **phần còn lại**, tính lúc chạy bằng `go list ./... \| grep -v -e /internal/course -e /internal/auth`. Thêm một bước kiểm hợp của A, B, C bằng đúng `go list ./...`, để gói mới không bị bỏ sót. | Job Go: 8,6 → **≈ 5–6 phút** (dòng C: `exam` 123 s là gói dài nhất, các gói còn lại chia 4 nhân; mỗi dòng còn tự biên dịch ≈ 1,5–2 phút [SUY LUẬN]). Cần làm khi frontend đã về ≈ 6–7 phút, lúc đó Go thành đường găng. | ≈ 25 dòng. Runner: ≈ 3 × 5 phút. | Giữ `-race`, giữ `TESTCONTAINERS_RYUK_DISABLED`, `EP_SKIP_TIMING`, bước đo thời gian `-p 1 -count=3` (chia theo gói `auth` / `course`). Mỗi dòng matrix là một máy riêng, container `Reuse` không bị chia sẻ chéo, nên còn ít rủi ro "connection refused" hơn hiện nay. |
| 6 (tuỳ chọn) | Kéo trước image testcontainers (`pgvector/pgvector:pg18`, `redis:8`, `pgsty/minio:…`, `axllent/mailpit:v1.27`, image pgbouncer) bằng `docker pull … &` ngay đầu job test, chạy song song với setup / biên dịch. | ≤ 0,5 phút [SUY LUẬN: test không in log testcontainers khi xanh nên không đo được thời gian kéo] | 1 bước. | Danh sách image phải khớp `internal/testutil/containers.go`, `pgbouncer.go`; lệch chỉ mất phần lợi, không hỏng test. |

**Tổng hợp (wall time ước lượng):**

| Kịch bản | Hiện nay | Sau #1 | Sau #1–#2 | Sau #1–#5 |
| --- | --- | --- | --- | --- |
| Push chỉ đổi backend | 27,4 phút | ≈ 8,6 | ≈ 8,6 | ≈ 5–6 |
| Push đổi frontend (hoặc cả hai) | 27,4 phút | 27,4 | ≈ 15 | ≈ 6–7 |
| PR vào `main` (chạy đủ) | 27,4 phút | 27,4 | ≈ 15 | ≈ 6–7 |

**Không đề xuất:**
- Bỏ `-race`, bỏ bước đo thời gian, bỏ một lượt Lighthouse, hay giảm `numberOfRuns` (đều là cổng hoặc AC).
- Tăng `workers` Playwright lên 4 trên một máy. Runner 4 nhân phải chạy cả `next start` lẫn máy chủ giả, nên dễ thêm test chập chờn; shard sang máy khác an toàn hơn.

**Ảnh hưởng:**
- Chỉ đổi `.github/workflows/ci.yml` (cộng hai tệp `lighthouserc.devtools.*.json` nếu làm #3b, một composite action nếu làm #2).
- Không đổi `ARCHITECTURE.md` hay `DECISIONS.md`.
- Nếu `main` có branch protection theo tên check thì phải đổi danh sách check bắt buộc (`Go`, `Frontend` → tên job mới).
- Cách lùi: các thay đổi độc lập, revert từng phần được.

**Đề xuất cho PM (một dòng chép vào `proposals.md` nếu cần):** CI: làm #1 (lọc đường dẫn, PR vào `main` luôn chạy đủ), #2 (tách `Frontend`), #4 (sửa cache Go) ngay; #3, #5 khi #1–#2 đã chạy ổn một tuần; mục tiêu ≤ 8 phút mỗi lượt, giữ nguyên test, `-race` và cổng Lighthouse.
