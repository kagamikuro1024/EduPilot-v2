# DEV handoff — US-PU-05 (cổng tự động: ảnh mốc, axe, Lighthouse, CI)
Nhánh `sprint/3-pu-p1`. Góp ý #25–#28. Chạy: `pnpm -C frontend build:gate && pnpm -C frontend exec playwright test` (Next :3310, gateway giả :3312).

## Làm gì
- `e2e/visual.spec.ts`: 7 route × 2 bề rộng = **14 ảnh** (`e2e/visual.spec.ts-snapshots/<route>-<w>.png`, không hậu tố hệ điều hành — `snapshotPathTemplate`); `page.clock.setFixedTime` 29/10/2026 09:20 +07, `reducedMotion: reduce`, `document.fonts.ready`, `maxDiffPixelRatio 0.005`, `threshold 0.2`, `retries: 0`. `/settings/llm` chụp Admin bằng token dev dán trong test.
- `e2e/a11y.spec.ts` + `e2e/support/routes.ts` + `e2e/axe-allow.json` (`[]`): axe (wcag2a/2aa/21aa/22aa) trên mọi route nav của 4 vai + route chi tiết đại diện + `/login`, `/dev/ui`, `/dev/data` ở 1440 và 390; chặn critical/serious; báo cáo `test-results/axe-report.json` và dòng `AXE: N lượt quét …`.
- `e2e/ui-foundation.spec.ts`: 3 ca **chỉ bàn phím** (`tabTo` chỉ nhấn Tab, không `.focus()`): `/chat` (gõ, Enter, `Dừng`, nguồn mở sẵn → Enter đóng, lịch sử phiên, về composer), `/threads` (mở thread, `Đặt câu hỏi`, gõ tiêu đề / chọn chủ đề bằng phím / nội dung, gửi, `VerificationState` "Chờ xác nhận"), 640 px (`AUDIT` `{ox:0,cut:[],ell:[]}`, bảng lịch sử phiên mở / `Esc` trả focus, composer không bị thanh dưới che); mỗi bước là `test.step`.
- `lighthouserc.json` + `lighthouse-auth.cjs` (cookie vai theo URL): 7 URL :3310, `numberOfRuns: 3`, mobile mặc định, 4 assert (LCP ≤ 2500, CLS ≤ 0,1, TBT ≤ 200, script ≤ 256000, trung vị), `upload: filesystem`; `.lighthouseci` vào `.gitignore`. Phụ thuộc dev mới: `@axe-core/playwright`, `@lhci/cli`.
- `.github/workflows/ci.yml` job Frontend (timeout 25 phút): install → lint → `ui-antipatterns` → `lint-selftest` → build → `build:gate` → cache + cài chromium → `playwright test --grep-invert @real` → `lhci autorun` → tải báo cáo khi lỗi; `playwright.config.ts`: CI `workers 2`, `retries 1`, reporter html. Không `legacy/`, không `secrets.*`.
- Sửa để đạt axe (đều thật, không ngoại lệ): `--ep-ink-3` 58 % → 53 % (#25); `StatusStrip` không đặt `<p>` trong `<dl>`; `Skeleton` `role=group`; hàng tải của `CommandPalette` bỏ `aria-label` trên `<li>`.
- Sửa lỗi gặp khi dựng cổng: `session.tsx` kẹp `setTimeout` ≤ 2³¹−1 ms (token hạn dài bị xoá ngay); `CommandPalette` đóng qua `onCancel` + state (sự kiện `close` đến muộn làm không mở lại được), đặt lại ô tìm khi **đóng**, bỏ dấu cả chữ `Đ` hoa.

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `visual.spec.ts`: **14 passed**, `ls …-snapshots \| wc -l` = 14; chạy lại liên tiếp không lệch. Ảnh mốc `/settings/llm` hiện là bản **mock** (cập nhật có giải thích khi P1-05 vào) |
| 2 | `docs/sprints/3/qc/shots/before/` có 12 ảnh (QC) |
| 3 | `keyboard-only` 3 ca pass (lặp 5× không chập chờn); 640 px `AUDIT` sạch |
| 4 | `a11y.spec.ts`: **102 lượt quét** (route × vai × bề rộng), critical 0, serious 0, moderate 0, minor 0; `jq length e2e/axe-allow.json` = 0 |
| 5 | (**cập nhật sau PM quyết #26**: bỏ phông 500 ⇒ LCP trung vị `/` 3249 · `/chat` 3164 · `/threads` 2709 · `/inbox` 3242 · `/gradebook` 3238 · `/settings/llm` 3390 · `/dev/ui` 3189 ms, phông truyền 64–79 KB (trước 98 KB); CLS 0, TBT 16–83 ms, JS 197–230 KB đạt. LCP vẫn > 2500 ⇒ assertion LCP chuyển `warn` TẠM (ngưỡng giữ 2500), `lhci assert` rc=0; Nợ ghi `docs/PROGRESS.md`; cổng P2 phải đưa về `error`. Cổng PU ghi "PASS có điều kiện".) — số đo trước đó: `lhci autorun` rc=1. Trung vị 3 lần (mobile mặc định): LCP `/` 3473 · `/chat` 3326 · `/threads` 3546 · `/inbox` 3407 · `/gradebook` 2864 · `/settings/llm` 3320 · `/dev/ui` 3338 ms (**> 2500**); CLS 0 mọi route; TBT 17–83 ms; JS 196–228 KB (≤ 256 KB). Máy: Mac arm64, Chrome for Testing (Playwright chromium 1243), tải máy 8–10 do QC chạy song song; LCP quan sát thật 0,49 s, ước lượng mô phỏng do ~365 KB tải ở 1,47 Mbps. **Không nới ngưỡng — #26 chờ PM** |
| 6 | `ui-antipatterns.sh` 19 ✓; `--selftest` 19 / 19 |
| 7 | `ci.yml` đã viết đủ bước; **chưa có lượt chạy trên GitHub** (không `gh run` ở máy này) |
| 8 | Nhánh `ci/ui-drift`: **chưa tạo** (QC chứng minh cổng đỏ; dev chỉ xoá sau khi QC chấm) |
| 9 | các commit PU không sửa `backend-go/` (ngoài P1-xx); `go test ./internal/contract/...` chạy ở cổng cuối |
| 12 | cổng PU gộp chạy ở bước "Cổng PU + P1" |
| 13 | `git grep -nE 'eyJ[A-Za-z0-9_-]{20,}\.' -- frontend/e2e frontend/lighthouserc.json` = 0 (token test dựng tại chỗ, `sig` giả) |

## Cổng frontend đã chạy
`eslint .` 0 · `tsc` sạch · `ui-antipatterns` 0 ✗ · `lint-selftest` 7/7, 19/19 · `ui-allow:` = 10 · `build:gate` + `env -u CI playwright test`: **134 passed, 66 skipped** (ca chỉ-desktop ở dự án mobile), không retry.

## Nợ
- LCP (#26) chờ quyết định PM; CI bước `lhci` sẽ đỏ tới khi đó.
- Ảnh mốc `settings-llm-*` cập nhật khi US-P1-05 vào (có giải thích).

## Thực hiện quyết định PM (#25, #26, #28)
- **#25:** `docs/design/DESIGN.md` bảng token: `--ep-ink-3` = `oklch(53% 0.014 25)` / `#746968` (trước `oklch(58%…)` / `#958a8c`), ghi lý do AA 4,5 : 1.
- **#26:** `layout.tsx` chỉ nạp phông 400 / 600 / 700 (subset `vietnamese` + `latin`); chữ 500 hiển thị bằng 400 ⇒ ảnh mốc `inbox-1440`, `inbox-390`, `threads-1440` lệch quá ngưỡng và được cập nhật (`--update-snapshots`, đổi chỉ do độ đậm chữ 500 → 400). Đo lại LHCI: số ở bảng AC5 trên. LCP vẫn > 2500 ⇒ `lighthouserc.json` LCP `warn` (ngưỡng 2500 giữ nguyên), `lhci assert` rc=0; Nợ + cổng P2 ghi ở `docs/PROGRESS.md`.
- **#28:** ảnh mốc `settings-llm-1440|390.png` đã là màn **thật** (cập nhật ở commit US-P1-05); sau thay đổi phông vẫn khớp, không đổi thêm.

## Sửa lỗi QC vòng 1 (BUG-PU05-1, GATE-PU TC-09, GATE-P1 TC-11/16) — nhánh `sprint/3-pu-p1`
**CI xanh ở HEAD:** run `37138882189` (`1701731`): Go `success`, Frontend `success`. (`10ecf58` chỉ là commit tài liệu của PM; run `37141206998` đang chạy khi viết.)

| Lỗi CI | Gốc rễ (bằng chứng) | Sửa |
| --- | --- | --- |
| 14 ảnh mốc `visual.spec.ts` lệch (BUG-PU05-1) | ảnh sinh trên macOS, CI là Linux | Sinh lại **trong image chính thức** `mcr.microsoft.com/playwright:v1.63.0-noble` (đúng phiên bản `@playwright/test` 1.63.0, #27): `docker run … pnpm install --frozen-lockfile && pnpm build && pnpm build:gate && playwright test visual.spec.ts --update-snapshots`; 14 ảnh, commit `e819d93`. **Lưu ý cho QC:** chạy `visual.spec.ts` trên macOS giờ LỆCH các ảnh Linux này (chạy trong cùng image docker, hoặc dựa CI) |
| Go `TestSSE_ReconnectRace`: "connection refused 127.0.0.1:41749/33945" | `internal/testutil` dùng chung container (tên cố định, `Reuse`) giữa các gói chạy song song; **Ryuk** (bật mặc định trên CI) giết container khi tiến trình đã *tạo* nó thoát ⇒ gói khác mất Redis giữa chừng. Local luôn đặt `TESTCONTAINERS_RYUK_DISABLED=true` nên không thấy. (Không liên quan `docker volume prune` — CI không chạy lệnh đó) | `ci.yml` bước `go test` đặt `TESTCONTAINERS_RYUK_DISABLED=true` (`bbe5b7f`). Chạy lại: `go test -race -count=8 -run TestSSE_ReconnectRace ./internal/httpapi/sse` local 8/8 ok |
| `lighthouse ci`: "Chrome installation not found" → rồi "No usable sandbox" | runner không có Chrome cho lhci; khi có `puppeteerScript` thì `chromeFlags` bị bỏ qua | `CHROME_PATH` = Chromium của Playwright vừa cài (`8dfdac1`); `puppeteerLaunchOptions.args = [--no-sandbox, --headless=new]` (`ca1cb96`) |
| lhci TBT `/dev/ui` > 200 | chẩn đoán in `benchmarkIndex` trên CI: ~2200; `/dev/ui` **TBT 544–929 ms**, 6 route còn lại 67–111 ms. CPU profile 4×: khung xương quét bằng `background-position` bắt main thread vẽ lại hàng chục khung (`(program)` 571 ms) | Khung xương quét bằng `transform` (compositor) → `(program)` 160 ms (`376f794`, cải thiện cả production). Phần còn lại là chi phí DOM của trang danh mục dev: trong container Linux BI 3600 còn 125–238 ms, khi tranh CPU 424 ms ⇒ phép đo này đo máy. **Góp ý #34 (PM ACCEPTED):** TBT riêng của `/dev/ui` là `warn` (`assertMatrix`), 6 route người dùng giữ `error`; CLS và JS của `/dev/ui` vẫn `error` |

## `ci/ui-drift` (AC8) — một nhánh, ba commit, ba run riêng (Q11); **chưa xoá nhánh, chờ QC chấm**
Tạo từ HEAD xanh `1701731`. Mỗi commit hoàn tác lỗi trước rồi đưa lỗi mới để mỗi run chứng minh đúng một cổng.
| Lỗi | `<ID>` | `headSha` | Kết quả |
| --- | --- | --- | --- |
| (a) `color: #c81d32` trong `features/chat/drift.module.css` | `37139595531` | `12638d78f7b87bbcf5413b23b9e36c4148af0e27` | `failure`, `ci/ui-drift`, **Frontend failure ở bước `ui antipatterns`**, Go success |
| (b) `.btn { position: relative; top: 3px }` | `37140069267` | `4081b0b3ef075fe7d8f30cbe75b6050678574cd6` | `failure`, **Frontend failure ở `playwright test`**: `visual.spec.ts` `gradebook @390` và `dev-ui @390` đỏ (153 pass); Go success |
| (c) bỏ `aria-label` nút chuông | `37140802309` | `38073d08dcd8ac7df395bb5847ae7533e111321c` | `failure`, **Frontend failure ở `playwright test`**: `a11y.spec.ts` "axe: student" đỏ (+ `shell.spec` topbar/notifications, `settings-llm` axe); Go success |
Ghi chú trung thực: run `37140689355` (`09ae2efb`) là lần đẩy (c) sai — `git revert` của (b) khôi phục luôn `drift.module.css` nên đỏ ở `ui antipatterns`; đã bỏ tệp đó ở `38073d0` và chạy lại ⇒ (c) hợp lệ là `37140802309`. Độ lệch 3 px chỉ làm 2/14 ảnh đỏ trên Linux (ngưỡng `maxDiffPixelRatio 0.005`; QC đo 14/14 ở macOS vì ảnh mốc cũ khác nền tảng). Kiểm: `gh run view <ID> --json conclusion,headBranch,jobs --jq '.conclusion, .headBranch, (.jobs[]|"\(.name) \(.conclusion)")'`. Dev **không** xoá `ci/ui-drift` cho tới khi QC chấm; xoá bằng `git push origin --delete ci/ui-drift`.

## GATE-P1 TC-16 — `llmload` báo `interactive_wait_p95_ms=1430` (> 500) và hành vi BATCH hết hạn ↔ cầu dao
**Giải thích số đo (sai điều kiện đo, không sai Scheduler).** Tái hiện bằng gateway `testroutes` + `fake` 5–15 s + `LLM_MAX_CONCURRENCY=10`:
| `LLM_DEFAULT_RPM` | `interactive_wait_p95_ms` | tất cả 25 chờ (ms) | `batch_peak` |
| --- | --- | --- | --- |
| 60 (mặc định) | **1326** / 1336 (hai lượt; QC 1430) | `0 0 0 1 1 1 1 2 2 2 2 3 4 4 4 5 6 17 168 309 448 934 1326 2221 3143` | 5 |
| 6000 (+ `LLM_DEFAULT_TPM=100000000`) | **1** | `0 … 0 1 1 1 1 1 1 1 1` | 5 |
Cơ chế: 5 BATCH + 5 CHAT với `fake` ~10 s = **1 yêu cầu/giây = đúng `LLM_DEFAULT_RPM=60`**. Khi đó `queue_wait_ms` của CHAT là thời gian chờ **token RPM** (≤ 1 s mỗi token, BATCH tranh token), không phải hàng đợi đồng thời. Số tự đo của QC (CHAT ≤ 4 đồng thời ⇒ 0,9 yêu cầu/giây < 60 RPM) không chạm trần nên 2 ms. **Sửa công cụ:** `cmd/llmload` ghi điều kiện đo (RPM ≥ 6000, TPM ≥ 10⁷) ở đầu tệp và in gợi ý khi p95 vượt ngưỡng. Lệnh đo đúng: `LLM_DEFAULT_RPM=6000 LLM_DEFAULT_TPM=100000000 … gateway serve` rồi `go run ./cmd/llmload --batch 200 --chat 25` → rc=0.
*Ghi chú thiết kế (không sửa, cần BA/PM nếu muốn):* khi RPM của nhà thực sự cạn, INTERACTIVE vẫn chờ ≤ 1 token (BATCH không bị chặn cướp token lúc hàng INTERACTIVE rỗng). Giữ riêng một phần RPM cho INTERACTIVE chưa có trong SRS 4.3.

**Hết hạn BATCH ↔ cầu dao (xác nhận và SỬA — đúng như QC nghi).** Trước: khi hạn của chính yêu cầu (BATCH 120 s) hết lúc nhà cung cấp còn đang trả lời, lỗi ctx `DeadlineExceeded` được chuẩn hoá thành `TIMEOUT` và `runChain`/`stream` gọi `BreakerReport` ⇒ 5 BATCH hết hạn liên tiếp **mở mạch** nhà (đúng hiện tượng QC: BATCH sau đó nhận 503, CHAT sang đường rút gọn; sau lượt đầu tôi thấy `batch_rejected=9` khi chạy lại ngay). Sau: `call.ownDeadlineExpired()` — hạn của CHÍNH yêu cầu không tính vào mạch; timeout do nhà (hạn từng lần gọi, `LLM_REQUEST_TIMEOUT`) và mọi lỗi 5xx/mạng thật vẫn tính. Test `TestOwnDeadlineNotCountedToBreaker` (hai ca; đỏ trước khi sửa: `[TIMEOUT]`, xanh sau). `go test -race ./internal/llm/...` ok; golangci-lint 0 issues.
