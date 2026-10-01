# DEV handoff — US-P0-03
Nhánh `sprint/1-p0-prep` · commit mã cuối: `53770ee50e67b47a040e3a5ca5b315a22a308ee2` (sau đó chỉ có commit tài liệu). Đã push.

## Đã làm (theo thứ tự lát dọc)
- `package.json` gốc: thêm `"packageManager": "pnpm@12.8.1"`; `pnpm-lock.yaml` cập nhật (pnpm 12 ghi `packageManagerDependencies`, `--frozen-lockfile` đòi có). `frontend/Dockerfile` lấy bản pnpm từ trường này → gỡ nợ "pnpm ghim tay" của US-P0-02 (CI và Dockerfile cùng một nguồn).
- `.github/workflows/ci.yml`: `push` + `pull_request`; `permissions: contents: read`; hai job độc lập `Go` (setup-go theo `backend-go/go.mod`, `go vet`, `golangci-lint` ghim `v2.14.0` = bản trên máy dev, `go test -race`) và `Frontend` (pnpm theo `packageManager`, Node 24 + cache pnpm, `pnpm install --frozen-lockfile`, `lint`, `build`, `ui antipatterns`). Không secret, không `continue-on-error`, không nhắc `legacy`.
- Không migration/store/frontend mới/seed.

## File đổi
`.github/workflows/ci.yml`, `package.json`, `pnpm-lock.yaml`, `frontend/Dockerfile`.

## Lệnh QC chạy để kiểm
(`source ~/.zprofile` mỗi shell)
```bash
gh run list --workflow ci.yml --branch sprint/1-p0-prep --limit 1 --json databaseId,headSha,conclusion   # headSha = git rev-parse origin/sprint/1-p0-prep; success
gh run view <ID> --json jobs --jq '.jobs[] | "\(.name) \(.conclusion)"'                                  # Frontend success / Go success
gh run view <ID> --json jobs --jq '.jobs[].steps[].name' | grep -ciE 'vet|golangci|race|lint|build|antipattern'   # ≥ 6 (thực tế 7)
grep -nE 'legacy|secrets\.|_API_KEY' .github/workflows/ci.yml          # không in gì
grep -c continue-on-error .github/workflows/ci.yml                      # 0
grep -nA1 '^permissions:' .github/workflows/ci.yml                      # contents: read
gh run view 36878548063 --json conclusion,headSha,headBranch,jobs --jq '.conclusion, .headSha, .headBranch, (.jobs[] | "\(.name) \(.conclusion)")'
gh run view 36878726051 --json conclusion,headSha,headBranch,jobs --jq '.conclusion, .headSha, .headBranch, (.jobs[] | "\(.name) \(.conclusion)")'
gh api repos/kagamikuro1024/TA_Agent/commits/8deac88facf8a65da8f412f165fb9d9a4b7ce2f7 --jq '.files[].filename'
gh api repos/kagamikuro1024/TA_Agent/commits/1efb08a19f7d06291f65c1e0204bcfd1ecb68a0a --jq '.files[].filename'
```

## Run bằng chứng (proposals #5)
| Mục | `<ID>` | `headSha` | Nhánh | Kết quả |
| --- | --- | --- | --- | --- |
| AC1 (mới nhất, sau ghim `ubuntu-24.04`, proposals #11) | 36879160770 | a44586a22dd1960c1812eac0e84500ff35aac494 | sprint/1-p0-prep | success; Go success, Frontend success |
| AC1 (trước khi ghim runner) | 36878528413 | 53770ee50e67b47a040e3a5ca5b315a22a308ee2 | sprint/1-p0-prep | success; Go success, Frontend success |
| AC1 (lần đầu, trước khi thêm cache go.sum) | 36878347940 | 93742140d8e71b92d8761a57ec3c73eee20bc4c5 | sprint/1-p0-prep | success |
| AC3 (test Go đỏ) | 36878548063 | 8deac88facf8a65da8f412f165fb9d9a4b7ce2f7 | ci/red-check | failure; Go failure, Frontend success; commit có `backend-go/internal/platform/red_test.go` |
| AC4 (màu viết cứng) | 36878726051 | 1efb08a19f7d06291f65c1e0204bcfd1ecb68a0a | ci/red-check | failure; Go success, Frontend failure ở bước `ui antipatterns`; commit có `frontend/src/app/page.tsx` (và xoá `red_test.go`) |

**Nhánh `ci/red-check` vẫn còn trên origin** — dev chỉ xoá (`git push origin --delete ci/red-check`) sau khi PM báo QC đã chấm AC3/AC4.

## Test đã chạy và kết quả
Bốn run ở bảng trên (kết quả thật từ `gh run view`). Trước khi push: `docker build -f frontend/Dockerfile .` với Dockerfile mới → thành công.

## AC tự đánh giá
AC1 ✓ · AC2 ✓ (grep rỗng) · AC3 ✓ · AC4 ✓ (phần "xoá nhánh + `ls-remote` rỗng" chờ QC) · AC5 ✓ (`permissions: contents: read`; `continue-on-error` = 0).

## Nợ / chưa làm / cần hỏi
- Runner đã ghim `ubuntu-24.04` (proposals #11).
- Action ghim theo major (`@v7`, `@v6`, `@v9`), không theo SHA.
- OpenAPI lint chưa vào CI (đúng SRS mục 10).
