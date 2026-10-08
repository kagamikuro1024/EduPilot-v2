# Báo cáo QC — US-PU-06 (khung vẽ từ máy chủ, Lighthouse LCP / TBT về `error`)
**Cập nhật 2026-10-09 (#15): TC-13 PASS. Còn FAIL 3 TC (TC-06, TC-12, TC-19); chờ dev "US-PU-06: fix B1/B2/B4".**
**Kết luận: FAIL 4 TC (TC-06, TC-12, TC-13 theo chữ AC3, TC-19), 3 TC KHÔNG KIỂM ĐƯỢC (TC-04, TC-05, TC-18/20 chưa chạy); còn lại PASS.** Bản chấm `9dca5ad` (`origin/sprint/5-pe`; mã dev = `799a236`, spec `FEAT-ui-foundation` v1.5 / PM #14). QC tự đo trên macOS arm64 (benchmarkIndex 3570–3840), Lighthouse 12.6.1 (`lhci`), Chromium headless-shell của Playwright, **không dùng số của dev**. Không có lỗ hổng bảo mật; không hồi quy ảnh mốc, axe, đăng nhập / phiên.

## Lỗi / lệch
- **B1 (TC-06 — FAIL, AC5).** TBT trung vị 12× **> 170 ở `/gradebook`**: lần 1 (5 lượt) `82 85 183 190 194` ⇒ trung vị **183**; lần 2 (7 lượt) `160 167 170 172 183 221 235` ⇒ **172**. `/chat` sát ngưỡng: `158 163 164 171 177` ⇒ 164; lần 2 `93 167 169 173 180 181 186` ⇒ 173. Dev ghi `/gradebook` 82 (chỉ đúng với nhóm lượt nhanh); phân bố hai đỉnh (≈ 85 / ≈ 190). *Repro:* build `pnpm build:gate`; chạy `node lighthouse-api.mjs` + `next start -p 3310`; `pnpm exec lhci collect --config=<rc tối giản> --url=http://localhost:3310/gradebook -n 7 --settings.throttling.cpuSlowdownMultiplier=12 --settings.extraHeaders='{"Cookie":"lh_role=teacher"}'`; trích `audits.total-blocking-time.numericValue`. Các route còn lại đạt: `/` 99, `/threads` 102, `/inbox` 132, `/settings/llm` 119 (lần 1). `/dev/ui` 187 (chỉ ghi số, `warn`).
- **B2 (TC-12 — FAIL, AC2).** 6 route trả `Cache-Control: s-maxage=31536000` (+ `x-nextjs-cache: HIT`, trang tiền kết xuất tĩnh); AC2 cấm `s-maxage` / `public`. *Repro:* `curl -s -D - -o /dev/null http://localhost:3310/chat | grep -i cache-control`. **Rủi ro thực: thấp** — HTML tĩnh không chứa tên / email / MSSV / token / `Set-Cookie` (0 khớp ở các mẫu còn lại); lệch là do Next tiền kết xuất, không phải chủ ý. Nếu BA chấp nhận thì sửa chữ AC2; nếu không thì dev phải đổi `Cache-Control` của 6 route.
- **B3 (TC-13) — ĐÃ ĐÓNG theo #15 (PM 2026-10-09).** `"use client"` ở hai layout trước = 0, sau = 0; AC3 mới = không tăng + `h1` / mô tả vẽ từ máy chủ ⇒ PASS (bản chấm đầu đòi "giảm" nên ghi FAIL).
- **B4 (TC-19 — FAIL, AC13).** `docs/PROGRESS.md` dòng 47 vẫn ghi "Lighthouse (PM #15 / #14): `total-blocking-time` của 6 route người dùng đang `warn`…" (nợ chưa xoá / đánh dấu trả); dòng 8 còn "LCP + TBT `warn` tới US-PU-06". Handoff chưa có bảng `First Load JS` (chỉ có `JS` ở bảng đo) và chưa ghi `benchmarkIndex` từng route.
- **L1 (TC-04 / TC-05 — KHÔNG KIỂM ĐƯỢC).** CI GitHub **không chạy** từ `9bb8eef` đến HEAD: "The job was not started because recent account payments have failed or your spending limit needs to be increased" (annotation của `check-runs/113406919300`; cả Go và Frontend, kể cả commit chỉ có tài liệu). Lần xanh cuối `eb19e09` / `f7d821f` (trước `font-display: block`, lượt devtools và `lighthouserc` mới). Không có bằng chứng CI nào cho `error` LCP / TBT; **chủ dự án cần xử lý billing**. QC thay bằng chạy **chính hai lệnh của CI** (`lhci autorun` và `lhci autorun --config=lighthouserc.devtools.json`) trên máy: cả hai `rc=0`, chỉ một cảnh báo (`/dev/ui` LCP 2.937 ms, `warn` đúng #14).
- **L2 (cấu hình).** `lighthouserc.json` đặt `largest-contentful-paint: off` (không phải số ghi, AC4 nói "ghi số"); LCP chỉ được chấm ở `lighthouserc.devtools.json` (đúng #14). Lượt simulate đo được LCP **3,4–4,2 s** ở cả 7 URL (xem bảng) — không chấm theo #14.
- **L3 (TC-07 / TC-08).** `/dev/ui` LCP devtools **2.971 ms** (> 2.500, `warn` theo #14). FCP (devtools) 0,82–0,84 s cho 6 route người dùng: chữ hiện ≤ 1 s.
- **L4 (QC).** Lần đầu QC chạy `lhci --outputDir` làm ghi đè kết quả giữa các URL (cờ không có hiệu lực): đã bỏ số đó và chạy lại; số trong báo cáo là lượt chạy lại.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | 7 URL; `lighthouserc.devtools.json` (`throttlingMethod: devtools`, `onlyCategories: performance`, `numberOfRuns: 3`): 6 route `error` 2500 `median`, `/dev/ui` `warn` 2500 `median` (#14); `lighthouserc.json` giữ LCP `off` (L2) |
| 02 | PASS | `total-blocking-time`: 6 route `error` 200 `median`; `/dev/ui` `warn`; ngưỡng 200 giữ nguyên; CLS `error` 0,1, `resource-summary:script:size` `error` 256000 |
| 03 | PASS | `git diff 1e2083e` ở `lighthouserc.json`, `lighthouse-auth.cjs`, `lighthouse-api.mjs`, `ci.yml`: không đổi `cpuSlowdownMultiplier`, không `skipAudits` / `onlyAudits`, `numberOfRuns` 3; **thêm** bước `lighthouse ci (devtools, chỉ LCP)` và `include-hidden-files` |
| 04 | **KHÔNG KIỂM ĐƯỢC** (L1) | billing; thay bằng chạy hai lệnh CI tại máy (rc=0) |
| 05 | **KHÔNG KIỂM ĐƯỢC** (L1) | không có 3 lần chạy xanh |
| 06 | **FAIL** (B1) | 12×, từng route (trung vị, 5 lượt): `/` 99, `/chat` 164, `/threads` 102, `/inbox` 132, **`/gradebook` 183**, `/settings/llm` 119; `/dev/ui` 187 (ghi) |
| 07 | PASS | LCP **devtools 4×** (3 lượt, trung vị): `/` 1.457, `/chat` 1.488, `/threads` 1.499, `/inbox` 1.924, `/gradebook` 1.915, `/settings/llm` 1.498 (≤ 2.500); `/dev/ui` 2.971 (`warn`). Simulate 4×: 3.380 / 3.394 / 3.462 / 4.044 / 3.962 / 3.382 / 3.242 (không chấm, #14) |
| 08 | PASS | `font-display: block`; `next/font` tự host (preload mặc định), `subsets: ["vietnamese","latin"]`, **3 độ đậm** (400, 600, 700); FCP devtools 819–837 ms (≤ 1 s) |
| 09 | PASS | `tokens.spec` nằm trong 330 pass (khẳng định `block`) |
| 10 | PASS | 6 route: `<h1>` = 1 mỗi route, chữ tiếng Việt thật: "Việc cần xử lý hôm nay", "Chat riêng", "Threads", "Hộp thư hỗ trợ", "Sổ điểm", "Cấu hình LLM"; không "Đang tải…" |
| 11 | PASS | `shell.spec` / `account.spec` pass trong lượt đầy đủ; CLS = 0 ở cả 7 URL, mọi chế độ |
| 12 | **FAIL** (B2) | 0 khớp tên / email / MSSV / token / `Set-Cookie`; 1 khớp `s-maxage` ở **mỗi** route |
| 13 | PASS (chấm lại theo #15) | `"use client"` 0 → 0 ở hai layout (không tăng); `h1` + mô tả có trong HTML máy chủ 6 route (TC-10); JS truyền tải `/` 252.161, `/chat` 252.161, `/threads` 252.161, `/inbox` 237.215, `/gradebook` 251.827, `/settings/llm` 248.551, `/dev/ui` 200.622 (≤ 256.000, sát) |
| 14 | PASS | phần tử LCP ở khung máy chủ: `<p …PreShell…body>` (`/chat`), `<p …Layout…desc>` (`/threads`, `/gradebook`, `/settings/llm`, `/inbox`), `<h1 …page-title>` (`/`); `lcp.spec.ts › lcp before refresh` **12/12 pass** (2 dự án × 6 route) |
| 15 | PASS | `visual.spec.ts` trong `mcr.microsoft.com/playwright:v1.63.0-noble` (cài + `build:gate` trong container): **14 passed**, hai lần liên tiếp, không `--update-snapshots`; 0 ảnh mốc đổi |
| 16 | PASS | Playwright `--grep-invert "@real\|visual" --workers=2`: **330 pass**, 100 skip, 0 fail (gồm `a11y`, `account`, `class-join`, `today`, `shell`, `lcp`, `tokens`); `axe-allow.json` = 0 mục (không tăng) |
| 17 | PASS | không phụ thuộc mới (`git diff` `package.json`, `pnpm-lock.yaml` rỗng); `pnpm lint` rc=0; `ui-antipatterns.sh` rc=0, **19** `✓`; `ui-allow:` = 9; `build:gate` rc=0 |
| 18 | **KHÔNG KIỂM ĐƯỢC** | chưa chạy `audit-login.mjs` + `matrix()` trên stack seed (RAM / thứ tự story); `shell.spec` (`nav per role`, `route access`) pass |
| 19 | **FAIL** (B4) | `PROGRESS.md` dòng 8 và 47 chưa trả nợ; handoff thiếu `First Load JS` / `benchmarkIndex` từng route |
| 20 | KHÔNG KIỂM ĐƯỢC | chưa đi tay bằng `playwright-cli` (thăm dò + ảnh); sẽ làm khi có stack seed cho PE |

## Việc sau
- **Dev:** B1 (TBT `/gradebook` và `/chat` ở 12×), B2 (`Cache-Control` 6 route hoặc xin BA nới AC2), B4 (nợ PROGRESS + handoff).
- **BA:** AC3 ("giảm" khi trước đã 0, Q-QC-PU06-6) và AC2 (`s-maxage` của trang tiền kết xuất); Q-QC-PU06-5 (`h1` của `/` khung dài hơn "Chào {tên}").
- **PM / chủ dự án:** billing GitHub Actions (CI không chạy từ `9bb8eef`).
- **QC:** chấm lại TC-06 / 12 / 13 / 19 khi dev / BA xử lý; TC-04 / 05 khi CI chạy lại; TC-18 / 20 cùng stack seed của PE.
