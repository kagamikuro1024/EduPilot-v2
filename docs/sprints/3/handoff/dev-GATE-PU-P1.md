# DEV handoff — cổng PU + P1 (chạy ở HEAD nhánh `sprint/3-pu-p1`)
Máy: Mac arm64; Chrome for Testing (Playwright chromium 1243); Go test qua colima. Đã dọn container `edupilot-test-*`, server :3310 / :3312.

## Cổng PU (`docs/phases/PU.md`)
| Lệnh | Kết quả thật |
| --- | --- |
| `pnpm -C frontend lint && pnpm -C frontend build` | `eslint .` rc=0; build thường OK, `/dev/*` 404, `grep 'DEV_AUTH\|Dán token\|state-cell' .next/static` = 0 |
| `bash scripts/ui-antipatterns.sh` (+ `--selftest`) | 19 ✓, 0 ✗; `19 / 19 phép bắt được`; `lint-selftest.sh` `7 / 7` + `19 / 19 phép ui-antipatterns bắt được`; `ui-allow:` = 9 (≤ 10) |
| `playwright test ui-foundation.spec.ts` (bản `build:gate`) | **14 passed**, 4 skipped (ca chỉ-desktop ở dự án mobile); gồm 3 ca chỉ-bàn-phím `/chat` + `/threads` + 640 px |
| `playwright test` (cả bộ, không retry) | **155 passed, 85 skipped**; `visual` 14/14, `a11y` 102 lượt quét, critical 0 · serious 0 · moderate 0 · minor 0 |
| `lhci autorun` | **PASS có điều kiện** (PM quyết #26): bỏ phông 500 rồi đo lại ⇒ LCP 2709–3390 ms (`/threads` 2709 · `/chat` 3164 · `/dev/ui` 3189 · `/gradebook` 3238 · `/inbox` 3242 · `/` 3249 · `/settings/llm` 3390) vẫn > 2500 ⇒ LCP `warn` tạm, `lhci assert` rc=0, Nợ ở `docs/PROGRESS.md`, cổng P2 đưa về `error`. Số đo trước khi bỏ phông 500: Trung vị 3 lần, mobile mặc định: LCP `/` 2865 · `/chat` 3315 · `/threads` 2936 · `/inbox` 3388 · `/gradebook` 2866 · `/settings/llm` 3542 · `/dev/ui` 3324 ms (ngưỡng 2500); CLS = 0 mọi route (≤ 0,1); TBT 14–69 ms (≤ 200); JS truyền 197–230 KB (≤ 256.000 B). Không nới ngưỡng — góp ý #26 chờ PM |
| `cd backend-go && go test ./internal/contract/...` | ok (golden không đổi; không commit PU nào sửa `backend-go/`, chỉ các commit P1) |

## Cổng P1 (`docs/phases/P1.md`)
| Lệnh | Kết quả thật |
| --- | --- |
| `grep -rn "openai\.\|anthropic\.\|genai\." backend-go --include=*.go \| grep -v internal/llm/` | không in gì (rc=1 = sạch) |
| `go test -race ./internal/llm/...` | ok (llm, cost, fake, provider, scheduler) |
| `go test -race ./internal/llm/scheduler/...` | ok |
| `go test ./internal/llm -run TestProviderContract -v` | `fake`, `fake-replay`, **`openai`, `gemini` PASS** (bản ghi thật, xem `dev-US-P1-02.md`); `anthropic` SKIP "BLOCKED: cần khoá thật để ghi" |
| `go test -race ./internal/llmconfig/... ./internal/platform/...` | ok |
| `make eval` | **hoãn theo plan sprint 3** (cần RAG + golden set, P3 / P10); không có target `eval` ở repo. BA đã xác nhận "ngoài phạm vi, không FAIL" (Q-QC-GATEP1-1). Nợ cho `docs/PROGRESS.md` (P3 / P10) |

Thêm: `go vet` ×3 tags sạch, `golangci-lint` ×3 0 issues, `sqlc diff` rc=0, `go test -race -count=1 -tags testroutes ./...` toàn bộ ok.

## "Bạn tự kiểm" còn lại (mắt người / stack thật — QC & chủ dự án)
- Mọi bước tay của P1 (thêm 2 nhà thật, đổi CHAT, tắt nhà chính, bơm 200 việc BATCH, tắt hết nhà) cần stack compose + khoá thật — chưa chạy ở máy dev.
- PU: mở `/dev/ui` cạnh `edupilot-ui-v3.html`; 10 câu `AGENT_PROMPT.md`; `/chat` trên điện thoại thật.
- Ca `@real` của PU-03 (AC6, 8, 11, 13, 16, 17) và P1-05 (hai tab, khoá sai, 403) chưa chạy.
- CI GitHub (`ci.yml`) đã viết đủ bước nhưng chưa có lượt chạy; bước `lhci` sẽ đỏ cho tới khi PM quyết #26.
