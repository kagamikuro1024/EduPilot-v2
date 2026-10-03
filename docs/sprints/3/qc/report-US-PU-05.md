# Báo cáo QC — US-PU-05 (cổng tự động: ảnh mốc, axe, Lighthouse, CI)
**Kết luận: FAIL** — AC7/AC8 (CI) không đạt. Phần cổng chạy ở máy (AC1–AC6, AC9–AC13) đạt; LCP theo #26 là "PASS có điều kiện".
Bản chấm: `a363512` (commit chứa #25/#26/#28) + commit QC sau đó; `build:gate` cổng riêng 3510/3512; Apple M3 Pro 11 nhân, Chrome for Testing 150.

## Lỗi
**BUG-PU05-1 (CI đỏ ở HEAD, AC7).** `gh run list --workflow ci.yml --branch sprint/3-pu-p1`: run `37130666133` (`a363512`), `37131124648` (`3ea1922`), `37131196419` (`8aeca57`) đều `failure`. Job Go `success`; job Frontend chỉ đỏ ở bước `playwright test`: **14 failed / 141 passed / 83 skipped**, đúng 14 ca `visual.spec.ts` (7 route × 2 bề rộng), ví dụ `13029 pixels (ratio 0.02) are different`. Nguyên nhân (suy luận, [INFERENCE]): ảnh mốc sinh trên macOS, `snapshotPathTemplate` không có hậu tố hệ điều hành (#27) nên runner Linux so với ảnh macOS → render chữ khác. Bước `lighthouse ci` bị `skipped`. Tái hiện: xem log run `37131196419` (`gh run view 37131196419 --log-failed`). Sửa: sinh ảnh mốc trong cùng môi trường CI (container Playwright / `--update-snapshots` trên runner), hoặc ghim phông + hậu tố nền tảng. Đề nghị dev.
Hệ quả: TC-PU05-28/29 FAIL (CI không xanh ở HEAD); TC-PU05-31 (artifact khi lỗi) có cấu hình `upload reports` chạy `success`, chưa tải thử.
**AC8:** nhánh `ci/ui-drift` chưa tạo (`git ls-remote --heads origin 'ci/*'` rỗng) → TC-PU05-32/33/34 FAIL "KHÔNG KIỂM ĐƯỢC"; TC-35 N/A. Q-QC-PU05-1/-2: chờ BA.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01, 05 | PASS | `visual.spec.ts`: 3 lần liên tiếp `14 passed` (6,0 / 5,4 / 5,4 s), rc=0 |
| 02 | PASS | 14 tệp đúng tên `<route>-{1440,390}.png`, kích thước 1440×900 / 390×844 |
| 03 | PASS | `maxDiffPixelRatio 0.005`, `threshold 0.2`, `reducedMotion: reduce`, `setFixedTime` 2026-10-29 09:20 +07, `timezoneId Asia/Ho_Chi_Minh`, `document.fonts.ready`, `retries: 0` |
| 04 | PASS | QC dịch `.btn` 3 px (`top:3px`) → build → `visual.spec.ts` **14 failed** (`24614 pixels, ratio 0.02`); hoàn tác → 14 passed; `git status src` rỗng |
| 06 | PASS | `inbox-1440/390`, `threads-1440` đổi ở `a363512` (#26, có giải thích trong handoff); `settings-llm-*` ở `c9e909d` (#28, P1-05) |
| 07 | PASS | `shots/before` 12 ảnh, commit `2dc2180` (03/10 12:49) sớm hơn commit PU-05 đầu `349f260`/`a363512` |
| 08–11, 41 | PASS | `ui-foundation.spec.ts` + full suite `155 passed / 85 skipped` (ca chỉ-desktop ở dự án mobile), 0 đỏ; QC đo 375×330 `/chat`: ô nhập trong vùng nhìn, không bị thanh dưới che, gõ được; 640 px dùng `AUDIT` sạch (đã chạy trong audit) |
| 12 | PASS | `a11y.spec.ts` rc=0, "AXE: 102 lượt quét … critical 0, serious 0, moderate 0, minor 0" |
| 14 | PASS | `axe-allow.json` = `[]` |
| 15 | PASS | `axe-core` 4.13 chạy riêng (wcag2a/2aa/21aa/22aa), `/`(SV, GV), `/chat`, `/gradebook`, `/dev/ui` × {1440, 390}: **0 vi phạm** mọi lượt |
| 16 | PASS | 20 phần tử × 4 màn (SV `/`, `/chat`, GV `/gradebook`, `/inbox`), công thức WCAG: min **4,65 : 1**, 0 dưới ngưỡng; `--ep-ink-3` = rgb(116,105,104) trên trắng **5,3 : 1** (#25) |
| 17 | PASS | `TOUCH_SRC` 375: 4 vai `/` và `/dev/ui` đều `[]` |
| 18 | PASS | bỏ `aria-label` nút chuông (`NotificationPopover.tsx`, tệp tạm) → build → `a11y.spec.ts` **FAIL** `button-name` ×48 (24 lượt critical); hoàn tác → rc=0 (102 lượt, 0). Lưu ý: lượt đầu cho xanh giả vì server 3510 cũ còn sống (`reuseExistingServer`) — QC đã diệt và chạy lại |
| 19–21 | PASS có điều kiện | `lhci autorun` rc=0 (cần `CHROME_PATH`), 7 URL × 3 = 21 báo cáo. Trung vị: LCP `/` 3237, `/chat` 3158, `/threads` 3232, `/inbox` 3234, `/gradebook` 3235, `/settings/llm` 3382, `/dev/ui` 3184 ms (**> 2500**); CLS 0; TBT 12–77 ms; JS 197–230 KB; perf 0,92–0,94; mobile, 4G chậm (RTT 150, 1638 kbps) + CPU 4×. LCP vượt ngưỡng, `assert` đang `warn` (#26: PASS có điều kiện, nợ ghi PROGRESS, cổng P2 phải về `error`) |
| 23, 24 | PASS | `ui-antipatterns.sh` 19 ✓ / 0 ✗, tên 19 phép gồm "Emoji làm icon chức năng", "Token / bí mật ghi vào storage"; `--selftest` 19/19; `lint-selftest` 7/7 |
| 25, 26 | PASS | tự gieo 10 mẫu: màu cứng, bo 16 px, bóng, cỡ chữ 13 px, `fetch(`, emoji 🔍, `localStorage.setItem('token'`, "streak 🔥", Tailwind `bg-gray-100 rounded-2xl`, `z-index:999` → mỗi mẫu bị đúng phép tương ứng; ca lành (`border-radius:0/50%`, `z-index:1`, "streak" trong `docs/`) rc=0 |
| 27 | PASS | `git status --short frontend/src` = 0 sau selftest và sau gieo |
| 30 | PASS | `grep legacy|secrets\.` ci.yml = 0; thứ tự bước đúng (install → lint → antipatterns → selftest → build → build:gate → cache → chromium → playwright `--grep-invert @real` → lhci → upload) |
| 36 | PASS | 0 commit `US-PU` đụng `backend-go/` |
| 37 | PASS | `go test -count=1 -v -tags testroutes ./internal/contract/...` rc=0, 11 `PASS`, 0 SKIP |
| 38 | PASS | `audit.mjs` SV 165 / GV 170 / TA 106 / Admin 54 / spec 179 = **674 PASS, FAIL 0**; `sweep.mjs only:'student'` 42 hàng, 0 xấu; `proto-curl.sh all` **493 PASS / 0 FAIL**. Thấp hơn nền 680 là 6 hàng `04-AC7/04-AC11` đo bố cục **mock** của `/settings/llm` (nay là màn thật, thay bằng TC-P105); công cụ đã bỏ các phép đó, không nới ngưỡng nào |
| 40 | PASS | 7 route SV: 0 từ kỹ thuật (`RAG|PII|trace|provider|fallback|embedding|prompt|LLM|token|API|…`), nút là động từ ("Gửi", "Phiên mới", "Đặt câu hỏi", "Luyện 10 câu", "Thêm vào lịch") |
| 43, 44 | PASS | `eyJ…` trong `e2e`/`lighthouserc.json` = 0; ảnh/dữ liệu giả |
| 09, 10, 13 (tay), 22, 39 | chưa chạy tay | đi tay bàn phím `/chat` `/threads` (QC dựa 3 ca `ui-foundation.spec.ts` pass), so sánh kích thước chip, 10 câu §22: gộp ở GATE-PU (TC-GATEPU-12) |

## Việc sau
Dev: BUG-PU05-1 (ảnh mốc cho CI Linux), tạo nhánh `ci/ui-drift` với 3 run (sau khi CI xanh). Sau đó QC chấm lại AC7/AC8.
