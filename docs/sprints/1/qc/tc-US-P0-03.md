# QC test case — US-P0-03
Nguồn: `docs/specs/FEAT-ci/US.md` + `SRS.md`. Viết trước khi có code, không đọc code của dev. Base sprint `1b72f54`.

**Chạy:** `RUN_RED_GO=<ID> RUN_RED_UI=<ID> bash docs/sprints/1/qc/scripts/tc-US-P0-03.sh TC-01 …` (hai ID lấy từ handoff dev; TC-07…11 cần chúng). Cần `git gh(đã login) jq go golangci-lint pnpm node`.
Mọi TC dựa vào bằng chứng thật: run GitHub Actions của đúng commit; không suy từ file workflow.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-01 | AC1 | HEAD `sprint/1-p0-prep` đã push | `…sh TC-01` (`gh run list --workflow ci.yml --branch sprint/1-p0-prep --json databaseId,headSha,conclusion`, chọn run có `headSha = origin/sprint/1-p0-prep`) | Có run đúng HEAD; `conclusion=success`; `event=push` |
| TC-02 | AC1 | TC-01 pass | `TC-02` (`gh run view <ID> --json jobs --jq '.jobs[]\|"\(.name) \(.conclusion)"'`) | Đúng 2 dòng: `Frontend success`, `Go success` |
| TC-03 | AC1 (FR-7) | TC-01 pass | `TC-03` (`gh run view <ID> --json jobs --jq '.jobs[].steps[].name' \| grep -ciE 'vet\|golangci\|race\|lint\|build\|antipattern'`) | ≥ 6; job Go có `go vet`, `golangci-lint`, `go test -race`; job Frontend có install, lint, build, ui antipatterns; không bước nào bị `skipped` |
| TC-04 | AC1 (FR-1…4, FR-7) tĩnh | file có | `TC-04` | Trigger `push` (không lọc nhánh/đường dẫn) + `pull_request`; Go: `working-directory: backend-go`, `go-version-file: backend-go/go.mod`, thứ tự bước `go vet` < `golangci-lint` (action ghim `version`, log phải có `golangci-lint run`) < `go test -race`, golangci-lint ghim bản (không `latest`); Frontend: `packageManager` ghim ở `package.json`, Node 24, cache pnpm, `--frozen-lockfile`, đúng 3 lệnh `pnpm -C frontend lint/build`, `bash scripts/ui-antipatterns.sh`; không `needs`; action không `@main/@master`; không `pull_request_target`; tên job `Go`, `Frontend` |
| TC-05 | AC2 | – | `TC-05` (`grep -nE 'legacy\|secrets\.\|_API_KEY' .github/workflows/ci.yml`) | Không in gì; thêm: không tên biến khoá LLM, không deploy/push image |
| TC-06 | AC2 runtime | TC-01 pass | `TC-06` (`gh run view <ID> --log`) | Log không nhắc `legacy/`, không `API_KEY`; có `golangci-lint run` và `go test -race`; `GITHUB_TOKEN Permissions` không có `write` |
| TC-07 | AC3 | `RUN_RED_GO` | `TC-07` | Run `failure`, nhánh `ci/red-check`, event push; job `Go=failure`, `Frontend=success` |
| TC-08 | AC3 | `RUN_RED_GO` | `TC-08` (`--log-failed`, `gh api …/commits/<sha>`) | Bước Go đỏ là `go test -race` (không phải vet/lint); Frontend chạy hết, không bước `skipped/cancelled/failure`; log có `FAIL` của test Go; commit đỏ chỉ thêm/sửa `*_test.go` |
| TC-09 | AC3/AC4 | – | `TC-09` | `ci/red-check` có ≥ 2 sha khác nhau đã chạy CI (hai commit liên tiếp, hai run) |
| TC-10 | AC4 | `RUN_RED_UI` | `TC-10` | Run `failure` trên `ci/red-check`; `Frontend=failure` **ở bước ui antipatterns**; install/lint/build của Frontend `success`; `Go=success` |
| TC-11 | AC4 | `RUN_RED_UI` | `TC-11` (`gh api …/commits/<sha>`) | Commit 2: sửa `frontend/src/app/page.tsx` thêm màu viết cứng (`#xxx`/`rgb(`) và bỏ test Go đỏ |
| TC-12 | AC4 | sau khi dev xoá nhánh | `TC-12` (`git ls-remote --heads origin ci/red-check`; `grep -c continue-on-error`) | Không in gì; `0` |
| TC-13 | AC5 | – | `TC-13` (`grep -nA1 '^permissions:' ci.yml`) | `contents: read` ở mức workflow; không `write` ở đâu; job không ghi đè `permissions` |
| TC-14 | Chéo: lan phạm vi (FR, §8 "không phá cái đang chạy") | – | `TC-14` (`git diff --name-only 1b72f54 HEAD -- .github`) | Chỉ có `.github/workflows/ci.yml`; `keep-huggingface-space-awake.yml` không đổi; đúng 2 workflow; không secret trong dòng thêm |
| TC-15 | Chéo: tái lập cục bộ | toolchain như CI | `TC-15` | Ba lệnh Go và bốn lệnh Frontend của CI chạy cục bộ đều exit 0 (lockfile khớp → `--frozen-lockfile` không lỗi) |
| TC-16 | Chéo: sạch nhánh | – | `TC-16` | Không còn commit chưa push (AC1 cần HEAD đã push); xoá nhánh tạm chỉ kiểm ở TC-12 sau khi QC chấm |

## Nhánh lỗi
AC3 → TC-07, 08, 09; AC4 → TC-10, 11, 12. (SRS mục 3: "Không áp dụng", nhánh lỗi ở AC3/AC4.)

## Phân quyền
AC5 "không áp dụng" — ràng buộc thay thế: TC-13 (`permissions: contents: read`), TC-05/TC-06 (không secret, không khoá LLM), TC-04 (không `pull_request_target`).

## Kiểm chéo bắt buộc (QC.md mục 3–4)
Phân quyền vai trò / PII / idempotency / phân trang / 375 px / migration: **không áp dụng** (chỉ thêm workflow). Áp dụng: lan phạm vi và secret (TC-14), `legacy/` không bị đụng (TC-05, TC-06), thư viện lạ (TC-04: action ghim, không `@main`), "không bỏ test cho xanh" (TC-08/11: commit đỏ chỉ đụng test cố ý; `continue-on-error` = 0 ở TC-12).

## Điểm khó kiểm / giả định
- AC1 chỉ chấm được sau khi dev push HEAD cuối và run xong; mọi commit mới (kể cả tài liệu) làm TC-01 FAIL cho tới khi có run mới.
- AC3/AC4: nhánh bị xoá ở AC4 nên nội dung commit chỉ kiểm được qua `gh api …/commits/<sha>`; cần handoff ghi `headSha` + nhánh (proposals #5).
- AC3 nói "Frontend vẫn chạy hết": TC-08 kiểm không có bước skipped/cancelled/failure; phụ thuộc hai job độc lập (TC-04 `needs` = 0).
- `next/font/google` cần mạng ở runner (SRS §8): nếu Frontend đỏ vì mạng thì báo PM, không tính xanh.
- Cần PATH có `gh` đã đăng nhập (proposals #6).

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-01 · TC-12, TC-16: nhánh `ci/red-check` chỉ bị xoá sau khi QC chấm AC3/AC4 — spec đổi theo proposals #5 (FEAT-ci v2). TC-04: thứ tự bước kiểm theo tên bước vì golangci-lint chạy qua action (lỗi của TC, không phải spec). TC-06: thêm kiểm log có `golangci-lint run`.

- 2026-10-01 · TC-06: lọc quyền `write` chỉ trong khối `GITHUB_TOKEN Permissions` (trước đó khớp nhầm dòng `Cache mode: write`) — lỗi của TC.
