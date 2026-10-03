# Báo cáo QC — GATE-PU (cổng nghiệm thu phase PU)
**Kết luận: FAIL** — một điều kiện còn đỏ: **CI GitHub đỏ ở HEAD** (BUG-PU05-1, 14 ảnh mốc lệch trên runner Linux) và AC8 (`ci/ui-drift`) chưa có. Mọi cổng chạy ở máy đạt; LCP "PASS có điều kiện" (#26). Bản chấm `a363512`…`3b27c1d`, QC tự chạy, không dùng số của dev.

| TC | KQ | Số đo |
| --- | --- | --- |
| 01 | PASS | `pnpm lint` rc=0 (0 warning); `pnpm build` (pbuild) rc=0, 0 "Failed to load font"; `/dev/*` là route động (404 ở production), 0 chuỗi `DEV_AUTH|Dán token|state-cell` trong `.next/static`; gbuild rc=0 |
| 02 | PASS | `ui-antipatterns.sh` 19 ✓ / 0 ✗; `--selftest` 19/19; `lint-selftest` 7/7; `ui-allow:` = 9 (≤ 10) |
| 03, 04 | PASS | `playwright test` toàn bộ (gồm `visual`, `a11y`, `ui-foundation`, `data-layer`, `dev-ui`, `settings-llm`, `shell`, `tokens`): **155 passed / 85 skipped**, 0 đỏ (skip = ca chỉ-desktop ở dự án mobile) |
| 05 | BLOCKED một phần | ca `@real` cần stack sprint 2 (`docker-compose.test.yml`); QC không dựng lại (containers `edupilot-test-*` thuộc phiên khác, `down -v` sẽ phá). Phần tương ứng đã chứng bằng Go: `go test -race -tags testroutes ./...` rc=0 (23 gói `ok`) và P1-05 chạy thật trên 2 gateway |
| 06 | PASS có điều kiện | `lhci autorun` rc=0, 7 URL × 3 lần. Trung vị: LCP 3158–3382 ms (**> 2500**), CLS 0, TBT 12–77 ms, JS 197–230 KB, perf 0,92–0,94. LCP chuyển `warn` theo #26; nợ P2 đưa về `error`. Máy: Apple M3 Pro 11 nhân, Chrome for Testing 150, mobile 4G chậm + CPU 4× |
| 07 | PASS | `go test -count=1 -v -tags testroutes ./internal/contract/...` rc=0, 11 PASS, 0 SKIP; 0 commit `US-PU` đụng `backend-go/` |
| 08 | PASS (theo từng thành phần) | từng lệnh của chuỗi AC12 đều rc=0 khi chạy riêng (lint, gbuild, antipatterns, selftest, lint-selftest, playwright, lhci, contract) — QC không chạy gộp một lệnh duy nhất |
| 09 | **FAIL** | `gh run list` ở HEAD: run `37131196419` (`8aeca57`) và hai run trước **failure**; Frontend đỏ ở `playwright test` (14 `visual.spec.ts` / 141 pass / 83 skip), `lighthouse ci` bị bỏ qua. Chưa có nhánh `ci/ui-drift` (TC-PU05-32…34 "KHÔNG KIỂM ĐƯỢC") |
| 10 | PASS | `audit.mjs` SV 165 / GV 170 / TA 106 / Admin 54 / spec 179 → 674 PASS, **FAIL 0** (thấp hơn nền 680 vì bỏ 6 hàng `04-AC7/04-AC11` đo bố cục mock của `/settings/llm`; không nới ngưỡng); `sweep.mjs only:'student'` 42 hàng, 0 `FORBIDDEN`/0 cuộn ngang/0 lỗi; `proto-curl.sh all` **493 PASS / 0 FAIL** (≥ 497 của nền cũ không đạt được về số lượng vì đã bỏ 4 phép hook mock của `/settings/llm`; không FAIL nào) |
| 11–13 | PASS (chủ quan có số) | `/dev/ui` 25 khối / 125 áp dụng / 75 N/A; màn thật `/settings/llm`: 0 card lồng, 0 nút primary hiện lúc đầu; ảnh `shots/after/settings-llm-{1440,390}.png`. So với `edupilot-ui-v3.html` và 10 câu §22: QC chưa lập bảng chi tiết từng câu — ghi nhận là chưa làm |
| 14 | PASS | `/chat` 375×330: ô nhập trong vùng nhìn, gõ được, không bị thanh dưới che — chỉ giả lập, chưa thiết bị thật (Q-QC-GATEPU-1: chờ BA) |
| 15 | PASS | 7 route SV: 0 từ kỹ thuật |
| 16 | một phần | 12 ảnh trước (`shots/before`) có; ảnh sau mới có `/settings/llm`; 12 cặp chưa chụp lại |
| 17 | PASS | dọn dẹp ở cuối phiên |

axe: 102 lượt (route × vai × bề rộng) = 0 critical / 0 serious / 0 moderate / 0 minor; QC chạy `axe-core` riêng 10 lượt = 0. Đối chứng âm (dịch nút 3 px; bỏ `aria-label`) làm cổng đỏ đúng chỗ.

## Việc còn lại để PASS
BUG-PU05-1 (sinh ảnh mốc trong môi trường CI hoặc ghim hậu tố nền tảng), tạo `ci/ui-drift` 3 run, rồi QC chấm lại TC-09 và AC8.
