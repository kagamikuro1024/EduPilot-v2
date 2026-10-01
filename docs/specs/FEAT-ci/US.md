# FEAT-ci CI cho Go gateway và frontend mới
Nguồn: PRD §5, FLOWS — (hạ tầng), phase P0 lát L2; CLAUDE.md "Lệnh" (Go, Frontend); D46 (không còn job Python), D48 (phiên bản)

Lệnh `<ID>` = `databaseId` của run, lấy bằng `gh run list`.

## US-P0-03: Lập trình viên muốn mỗi lần push đều được kiểm tự động để không lỗi nào lọt vào nhánh chung
Ưu tiên: Must · Ước lượng: S · Sprint: 1

### Tiêu chí nghiệm thu
- AC1. Given HEAD của `sprint/1-p0-prep` đã push lên `origin` When GitHub Actions chạy `.github/workflows/ci.yml` Then có đúng 2 job — **Go** (`go vet ./...`, `golangci-lint run`, `go test -race ./...` trong `backend-go`) và **Frontend** (`pnpm install --frozen-lockfile`, `pnpm -C frontend lint`, `pnpm -C frontend build`, `bash scripts/ui-antipatterns.sh`) — cả hai xanh trên đúng commit HEAD.
  Kiểm:
  ```bash
  gh run list --workflow ci.yml --branch sprint/1-p0-prep --limit 1 --json databaseId,headSha,conclusion
  # headSha = $(git rev-parse origin/sprint/1-p0-prep); conclusion = success
  gh run view <ID> --json jobs --jq '.jobs[] | "\(.name) \(.conclusion)"'          # 2 dòng, success
  gh run view <ID> --json jobs --jq '.jobs[].steps[].name' | grep -ciE 'vet|golangci|race|lint|build|antipattern'   # ≥ 6
  ```
- AC2. Given workflow CI When đọc file Then không bước nào đọc, cài hay build gì trong `legacy/`, không dùng secret, không có khoá LLM (không thể gọi LLM thật).
  Kiểm: `grep -nE 'legacy|secrets\.|_API_KEY' .github/workflows/ci.yml` → không in gì.
- AC3 (nhánh lỗi — test Go đỏ). Given nhánh tạm `ci/red-check` có một test Go cố ý đỏ When CI chạy Then run `failure`, job Go `failure`, job Frontend vẫn chạy hết và `success`.
  Kiểm: dev push nhánh tạm; handoff ghi cho run này `<ID>`, `headSha`, tên nhánh (proposals #5). QC: `gh run view <ID> --json conclusion,headSha,headBranch,jobs --jq '.conclusion, .headSha, .headBranch, (.jobs[] | "\(.name) \(.conclusion)")'` → `failure`, `headSha`/`headBranch` khớp handoff, Go `failure`, Frontend `success`; nội dung commit (có test đỏ): `gh api repos/{owner}/{repo}/commits/<headSha> --jq '.files[].filename'` có file `_test.go`.
- AC4 (nhánh lỗi — phản mẫu UI). Given commit kế tiếp trên `ci/red-check` bỏ test đỏ và thêm một màu viết cứng vào `frontend/src/app/page.tsx` When CI chạy Then run `failure`, job Frontend `failure` ở bước phản mẫu UI, job Go `success`. Dev **chỉ xoá** `ci/red-check` sau khi QC báo đã chấm xong AC3 và AC4.
  Kiểm: như AC3 với `<ID>`, `headSha`, nhánh của run thứ hai ghi trong handoff; `gh api repos/{owner}/{repo}/commits/<headSha> --jq '.files[].filename'` có `frontend/src/app/page.tsx`. Sau khi QC chấm và dev xoá nhánh: `git ls-remote --heads origin ci/red-check` → không in gì. `grep -c continue-on-error .github/workflows/ci.yml` → `0`.
- AC5 (phân quyền). **Không áp dụng — chưa có API nghiệp vụ.** Lý do: story chỉ thêm workflow CI; không có endpoint, vai trò hay dữ liệu người dùng. Ràng buộc an toàn thay thế: workflow chỉ có quyền đọc mã (`permissions: contents: read`).
  Kiểm: `grep -nA1 '^permissions:' .github/workflows/ci.yml` → `contents: read`.

### Ngoài phạm vi của story này
- Job Python (D46: không còn service Python; script đánh giá P10 chạy tay).
- `sqlc diff`, contract test `internal/contract`, Playwright, Lighthouse, `govulncheck`: thêm khi phase tương ứng có mã để kiểm (CLAUDE.md "Lệnh").
- Deploy, build image đẩy lên registry.
- Sửa workflow `keep-huggingface-space-awake.yml` (xem FEAT-scaffold Q1).

### Phụ thuộc
- US-P0-02 (cần `backend-go/` và `frontend/` mới để có gì chạy). `gh` đã đăng nhập.
