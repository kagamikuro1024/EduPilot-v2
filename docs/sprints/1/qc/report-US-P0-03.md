# QC report — US-P0-03  · Kết luận: PASS
Nhánh `sprint/1-p0-prep`, spec `docs/specs/FEAT-ci/` v2, handoff `docs/sprints/1/handoff/dev-US-P0-03.md`. Chạy 2026-10-01. TC: `docs/sprints/1/qc/tc-US-P0-03.md` (16 TC). Kết quả: `docs/sprints/1/qc/run-US-P0-03.log` — **70 PASS, 1 FAIL đang chờ** (TC-12 dòng "nhánh `ci/red-check` đã xoá": spec v2 chỉ cho xoá **sau khi QC chấm**, nên FAIL này là trạng thái chờ, không phải lỗi).

> **QC đã chấm xong AC3 và AC4. PM có thể cho dev xoá `ci/red-check`** (`git push origin --delete ci/red-check`). Sau đó chạy lại `tc-US-P0-03.sh TC-12` để chốt `git ls-remote --heads origin ci/red-check` rỗng.

## Cổng nghiệm thu đã chạy (lệnh → PASS/FAIL)
| Lệnh | KQ | Đầu ra chính |
| --- | --- | --- |
| `gh run list --workflow ci.yml --branch sprint/1-p0-prep` → run của HEAD | PASS | `36879278517`, `headSha=75ddd58ec2b2…` = `origin/sprint/1-p0-prep`, `success`, event `push` |
| `gh run view 36879278517 --json jobs` | PASS | `Frontend success`, `Go success` (đúng 2 job) |
| `… steps[].name \| grep -ciE 'vet\|golangci\|race\|lint\|build\|antipattern'` | PASS | 7 (≥ 6) |
| `grep -nE 'legacy\|secrets\.\|_API_KEY' ci.yml` | PASS | không in gì |
| `grep -c continue-on-error ci.yml` | PASS | 0 |
| `grep -nA1 '^permissions:' ci.yml` | PASS | `contents: read` |
| Chạy cục bộ đúng lệnh của CI (TC-15): `go vet`, `golangci-lint run`, `go test -race`, `pnpm install --frozen-lockfile`, `lint`, `build`, `ui-antipatterns.sh` | PASS | exit 0 cả hai nhóm |

## AC
| AC | Cách kiểm | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| AC1 | TC-01…04, 06 | PASS | Run của đúng HEAD (`75ddd58`, sau ghim `ubuntu-24.04` — proposals #11) xanh, 2 job, 7 bước khớp. Tĩnh: push (mọi nhánh) + pull_request; Go: `working-directory: backend-go`, `go-version-file`, thứ tự `go vet` < `golangci-lint` < `go test -race`, golangci-lint ghim `v2.14.0`; Frontend: pnpm theo `packageManager` (`pnpm@12.8.1`), Node 24 + cache pnpm, `--frozen-lockfile`, lint/build/antipatterns. Log run có `golangci-lint run` và `go test -race` thật |
| AC2 | TC-05, 06 | PASS | Không `legacy`/`secrets.`/`_API_KEY`/biến khoá LLM; log run không nhắc `legacy/`, không API key; `GITHUB_TOKEN` chỉ `Contents: read`, `Metadata: read` |
| AC3 | TC-07, 08, 09 | PASS | Run `36878548063`, `headSha=8deac88…`, nhánh `ci/red-check`: `failure`; `Go failure` ở bước `go test -race` (log có `FAIL`), `Frontend success` chạy hết, không bước bị skipped/cancelled; commit chỉ thêm `backend-go/internal/platform/red_test.go` |
| AC4 | TC-10, 11, 12 | PASS (chờ xoá nhánh) | Run `36878726051`, `headSha=1efb08a…`: `failure`; `Frontend failure` đúng bước `ui antipatterns` (install/lint/build trước đó `success`); `Go success`; commit xoá `red_test.go` và sửa `frontend/src/app/page.tsx` thêm màu cứng. `continue-on-error` = 0. Phần "`ls-remote` rỗng" chỉ kiểm được sau khi dev xoá nhánh — spec v2 |
| AC5 | TC-13 | PASS (n/a có ràng buộc thay thế) | `permissions: contents: read` ở mức workflow; không `write` ở đâu; job không ghi đè |

## Lỗi
Không có lỗi.

## Kiểm chéo
- Lan phạm vi: dưới `.github/` chỉ thêm `workflows/ci.yml`; `keep-huggingface-space-awake.yml` không đổi; đúng 2 workflow (TC-14). Không secret trong dòng thêm.
- Hai job độc lập: không `needs`; run đỏ ở AC3 vẫn chạy hết job Frontend, run đỏ ở AC4 vẫn chạy hết job Go.
- Tái lập: `pnpm install --frozen-lockfile` chạy cục bộ không lỗi → lockfile khớp.
- Thư viện: action ghim theo major (`@v6/@v7/@v9`), không `@main`; không theo SHA (dev đã ghi nợ). Runner ghim `ubuntu-24.04`.
- Ghi chú: golangci-lint chạy qua `golangci/golangci-lint-action@v9` (bản ghim `v2.14.0`) thay vì lệnh `golangci-lint run` trực tiếp; log chứng minh action chạy đúng `golangci-lint run`, nên tính là đạt FR-2.
- TC đã sửa khi chạy (lỗi của TC, không phải của dev; xem "Lịch sử sửa TC"): TC-04, 06, 12, 16.
- Mọi commit tài liệu mới đẩy lên nhánh sẽ sinh run mới; TC-01 luôn chấm run của HEAD tại thời điểm chạy.

## Đề nghị
PASS. PM cho dev xoá `ci/red-check`; QC chạy lại `TC-12` để đóng AC4 hoàn toàn.
