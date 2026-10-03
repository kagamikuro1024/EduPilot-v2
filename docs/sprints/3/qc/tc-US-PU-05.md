# QC test case — US-PU-05 (cổng tự động: ảnh mốc, bàn phím, axe, Lighthouse CI, phản mẫu, CI)
Nguồn: `docs/specs/FEAT-ui-foundation/US.md` US-PU-05 AC1–AC13 + `SRS.md` 4.3 (19 phép `ui-antipatterns.sh`), 8.2 (hiệu năng), 8.3 (ảnh mốc), 8.4 (trợ năng), 8.5 (LHCI), 8.6 (CI). Hộp đen. Nền: `audit-baseline.md`; ảnh trước: `shots/before/` (12 ảnh).

Tiền điều kiện chung: `gbuild && $FE exec next start -p 3300`; `gh` đã đăng nhập (đọc CI, đẩy nhánh `ci/ui-drift` — **nhánh tạm do dev tạo ở handoff, QC chỉ đọc**; QC không đẩy nhánh code). Công cụ: **P** = Playwright (`$PW <tệp>`), **L** = `lhci autorun`, **S** = shell, **A** = Eval trình duyệt thật, **G** = `gh`, **T** = tay. Thiếu tệp spec / `lighthouserc.json` → FAIL "KHÔNG KIỂM ĐƯỢC". Số đo Lighthouse **không bao giờ nới ngưỡng**: không đạt do máy → ghi số đo thực + cấu hình máy, vẫn FAIL, báo PM.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PU05-01 | AC1 | `gbuild` | **P** `$PW visual.spec.ts; echo rc=$?` | `rc=0`, `14 passed`, 0 skip |
| TC-PU05-02 | AC1 | – | **S** `ls frontend/e2e/visual.spec.ts-snapshots \| wc -l`; tên theo 7 route × 2 bề rộng | `14`; đủ `/`, `/chat`, `/threads`, `/inbox`, `/gradebook`, `/settings/llm`, `/dev/ui` × {1440 × 900, 390 × 844}; kích thước ảnh đúng bề rộng |
| TC-PU05-03 | AC1 | – | **S** đọc cấu hình: `maxDiffPixelRatio`, `threshold`, `reducedMotion`, đồng hồ cố định (SRS 8.3) | `maxDiffPixelRatio ≤ 0.005`, `threshold ≤ 0.2` (không nới); `page.clock.setFixedTime` 2026-10-29 09:20 `Asia/Ho_Chi_Minh`; `reducedMotion: 'reduce'`; chờ `document.fonts.ready` |
| TC-PU05-04 | AC1 (đối chứng âm) | cây sạch | **S** QC dịch một nút 3 px bằng CSS tạm trong tệp ở `frontend/src/shared/ui/` (không commit) → `gbuild` → `$PW visual.spec.ts`; rồi `git checkout -- <tệp>` | Có ≥ 1 ảnh **FAIL** với diff > ngưỡng; hoàn tác → `rc=0` lại; `git status` sạch |
| TC-PU05-05 | AC1 (ổn định) | – | **S** chạy `$PW visual.spec.ts` **3 lần liên tiếp** | 3/3 `14 passed` (không flaky); ghi thời gian |
| TC-PU05-06 | AC1 (cập nhật ảnh) | – | **S** `git log --follow -p -- frontend/e2e/visual.spec.ts-snapshots \| grep -c '^commit'`; đọc handoff từng lần cập nhật | Mỗi lần cập nhật ảnh mốc có lời giải thích trong handoff; nếu không → FAIL (QC từ chối) |
| TC-PU05-07 | AC2 | – | **S** `ls docs/sprints/3/qc/shots/before \| wc -l`; tên mẫu `<route>-<w>.png`; `git log -1 --format=%H -- docs/sprints/3/qc/shots/before` so với commit PU đầu tiên | `12`; ảnh trước **sớm hơn** commit story PU đầu tiên (`git log --reverse --grep='^PU' -1`); `/dev/ui` không có ảnh trước |
| TC-PU05-08 | AC3 | `gbuild` | **P** `$PW ui-foundation.spec.ts; echo rc=$?`; in tên `test.step` | `rc=0`; có các bước: `/chat` gõ, Enter, `Dừng`, mở nguồn, lịch sử phiên, về composer; `/threads` danh sách, mở thread, `Đặt câu hỏi`, gõ, gửi, đọc `VerificationState` |
| TC-PU05-09 | AC3 | SV B | **A** QC tự đi `/chat` **chỉ bàn phím** (`Tab`, `Enter`, `Esc`, mũi tên; không chuột): gõ câu hỏi → gửi → `Dừng` khi đang trả lời → mở một nguồn → mở lịch sử → quay lại composer | Mỗi bước focus **nhìn thấy** (ghi `boxShadow` ≠ none ở phần tử focus); không bẫy focus ngoài hộp thoại; `Esc` thoát overlay; hoàn thành mọi bước |
| TC-PU05-10 | AC3 | SV B | **A** như 09 ở `/threads`: danh sách → mở thread → `Đặt câu hỏi` → gõ → gửi → đọc `VerificationState` | Hoàn thành chỉ bàn phím; `VerificationState` đọc được (chữ + `role`/`aria`) |
| TC-PU05-11 | AC3 (640 px) | – | **A** `/chat` và `/threads` ở 640: `AUDIT_SRC`; composer + nút chính tới được bằng `Tab`; `elementFromPoint` ở tâm composer | `AUDIT` sạch (`ox:0`, `cut:[]`, `ell:[]`); composer/nút chính tới được; không phần tử bị thanh dưới che |
| TC-PU05-12 | AC4 | `gbuild` | **P** `$PW a11y.spec.ts; echo rc=$?` | `rc=0`; báo cáo in tổng số route × vai đã quét |
| TC-PU05-13 | AC4 | – | **S** đếm số route trong `routes.ts` (theo ma trận SRS 7.5) × vai được mở × 2 bề rộng + `/dev/ui`; so với số QC đọc ở báo cáo | Số lượt quét ≥ số route × vai hợp lệ (QC tính độc lập từ SRS 7.5, không từ báo cáo) |
| TC-PU05-14 | AC4 | – | **S** `jq length frontend/e2e/axe-allow.json`; xem từng mục (luật, route, lý do, người duyệt) | ≤ `3`; mọi mục đủ bốn trường không rỗng |
| TC-PU05-15 | AC4 | – | **A** QC chạy `axe-core` riêng (tag `wcag2a,wcag2aa,wcag21aa,wcag22aa`) trên 4 route tuỳ chọn × 2 bề rộng (`/`, `/chat`, `/gradebook`, `/dev/ui`) | 0 `critical`, 0 `serious` (ngoài `axe-allow.json`); `moderate`/`minor` ghi báo cáo |
| TC-PU05-16 | AC4 (tương phản) | – | **A** tính tương phản chữ/nền 20 phần tử ngẫu nhiên ở 3 route (công thức WCAG) | Chữ thường ≥ 4,5 : 1; chữ lớn / thành phần ≥ 3 : 1 |
| TC-PU05-17 | AC4 (vùng chạm) | 375 | **A** `TOUCH_SRC` trên `/dev/ui` và 4 vai `/` | `[]` |
| TC-PU05-18 | AC4 (đối chứng âm) | cây sạch | **S** QC bỏ `aria-label` khỏi một nút icon (tệp tạm, không commit) → `gbuild` → `$PW a11y.spec.ts`; hoàn tác | `a11y.spec.ts` **FAIL** (nêu luật `button-name`); hoàn tác → `rc=0` |
| TC-PU05-19 | AC5 | `gbuild` + `lhci` | **L** `$FE exec lhci autorun; echo rc=$?` | `rc=0`; `.lighthouseci/` có 7 URL × 3 lần = 21 báo cáo; `jq '.[].summary' .lighthouseci/manifest.json` có `performance` |
| TC-PU05-20 | AC5 | – | **S** đọc `frontend/lighthouserc.json`: `numberOfRuns`, ngưỡng, `puppeteerScript`, cấu hình mạng/CPU | `numberOfRuns: 3`; LCP ≤ 2500, CLS ≤ 0,1, TBT ≤ 200, `resource-summary:script:size ≤ 256000`; cấu hình mobile mặc định (4G chậm + CPU 4×); **không** nới (so với SRS 8.2); `puppeteerScript` nạp phiên theo vai |
| TC-PU05-21 | AC5 | – | **S** trung vị 3 lần của 7 route: LCP, CLS, TBT, JS truyền (từ `.lighthouseci/lhr-*.json`) | Mọi route: LCP ≤ 2,5 s, CLS ≤ 0,1, TBT ≤ 200 ms, JS ≤ 250 KB nén; **ghi bảng số đo + cấu hình máy** vào report; vượt → FAIL (không nới) |
| TC-PU05-22 | AC5 | – | **S** đối chứng với `audit-baseline` / `next build` log: JS mỗi route sau PU-03 (TanStack Query) so với nền | Tăng nhưng ≤ 250 KB; ghi mức tăng |
| TC-PU05-23 | AC6 | – | **S** `bash scripts/ui-antipatterns.sh \| grep -c '^✓'`; `… \| grep -c '^✗'`; liệt kê 19 tên phép và so với SRS 4.3 | `19`; `0`; tên phép khớp danh sách SRS 4.3 (không thêm bớt, nền cũ 11 phép đều còn) |
| TC-PU05-24 | AC6 | – | **S** `bash scripts/ui-antipatterns.sh --selftest; echo rc=$?` | `rc=0`, in `19 / 19 phép bắt được` (gồm "emoji làm icon chức năng", "token / bí mật trong storage") |
| TC-PU05-25 | AC6 (QC tự gieo) | cây sạch | **S** QC tự gieo **từng** vi phạm vào tệp tạm trong `frontend/src` (một lượt một phép; ít nhất 8 phép: màu cứng, bo 16 px, bóng ngoài danh sách, cỡ chữ lạ, `fetch(` ngoài `shared/data/`, emoji `🔍` làm icon nút, `localStorage.setItem('token',…)`, gamification `streak`) rồi `bash scripts/ui-antipatterns.sh; echo rc=$?`; xoá tệp | Mỗi lượt `rc≠0` và nêu đúng phép; sau xoá `rc=0`; không dựa vào `--selftest` của dev |
| TC-PU05-26 | AC6 | – | **S** ca lành không báo nhầm: `border-radius: 0`, `50%`, `z-index: 1`, chữ "streak" trong `docs/` (ngoài `frontend/src`) | Không bị bắt |
| TC-PU05-27 | AC6 | – | **S** `UI_SRC=<thư mục tạm> bash scripts/ui-antipatterns.sh --selftest` có thể chạy từ thư mục khác; để lại `frontend/src` sạch | `git status --short frontend \| wc -l` = `0` sau selftest |
| TC-PU05-28 | AC7 | `gh` | **G** `gh run list --workflow ci.yml --branch sprint/3-pu-p1 --limit 1 --json databaseId,headSha,conclusion`; so `headSha` với `git rev-parse origin/sprint/3-pu-p1` | `headSha` trùng HEAD đẩy; `conclusion=success` |
| TC-PU05-29 | AC7 | – | **G** `gh run view <ID> --json jobs --jq '.jobs[].steps[].name'`; `grep -ciE 'playwright\|lhci\|antipatterns\|selftest'` | ≥ `4`; thứ tự bước: cài (cache pnpm) → lint → antipatterns → selftest → build → cài chromium → playwright `--grep-invert @real` (7 tệp) → lhci |
| TC-PU05-30 | AC7 | – | **S** `grep -nE 'legacy\|secrets\.' .github/workflows/ci.yml`; job **Go** không đổi so với sprint 2 (`git diff <sprint2-tip> -- .github/workflows/ci.yml` chỉ thêm job Frontend) | Không in gì; job Go còn nguyên; ci.yml không đọc `legacy/` |
| TC-PU05-31 | AC7 | – | **G** `gh run download <ID> -n <artifact>` trên một run thất bại (từ AC8) | Có artifact báo cáo (Playwright HTML / `.lighthouseci`) khi thất bại |
| TC-PU05-32 | AC8 (a) | nhánh `ci/ui-drift` do dev tạo | **G** `gh run view <ID_a> --json conclusion,headBranch,jobs --jq '.conclusion, .headBranch, (.jobs[] \| "\(.name) \(.conclusion)")'`; `gh run view <ID_a> --log-failed \| grep -c 'ui-antipatterns'` | `failure`, `ci/ui-drift`, `Frontend failure`, `Go success`; thất bại **ở bước `ui-antipatterns`** (không bước khác); diff của commit là `color: #c81d32` trong CSS Module ngoài `shared/styles/` (`git show <headSha>`) |
| TC-PU05-33 | AC8 (b) | – | **G** như 32 cho run (b) | `failure` ở bước Playwright `visual.spec.ts`; commit lệch nút 3 px |
| TC-PU05-34 | AC8 (c) | – | **G** như 32 cho run (c) | `failure` ở bước `a11y.spec.ts`; commit bỏ `aria-label` |
| TC-PU05-35 | AC8 (dọn) | sau khi QC chấm | **S** `git ls-remote --heads origin ci/ui-drift` (sau khi dev xoá) | Không in gì; handoff ghi `<ID>`, `headSha`, tên nhánh |
| TC-PU05-36 | AC9 | – | **S** `git log --format=%H --grep='^PU:' \| xargs -n1 git show --name-only --format= \| grep -c '^backend-go/'` | `0` |
| TC-PU05-37 | AC9 | testcontainers | **S** `cd backend-go && go test -count=1 ./internal/contract/...; echo rc=$?` (`TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=unix://$HOME/.colima/default/docker.sock`) | `rc=0`; không `--- SKIP` bất ngờ (so sprint 2) |
| TC-PU05-38 | AC10 | bản gate | **A** `audit.mjs` bốn vai + spec như baseline; `sweep.mjs only:'student'`; ghi `audit-log.md` | FAIL 0 mỗi lượt; PASS ≥ nền (680); `FORBIDDEN`=0; số liệu sau story vào `docs/sprints/3/qc/audit-log.md` |
| TC-PU05-39 | AC11 | `/dev/ui` + 6 route AC1 | **T** trả lời 10 câu nghiệm thu cuối `docs/design/AGENT_PROMPT.md` cho từng màn đã dựng | Mọi câu "có"; bảng ghi `report-US-PU-05.md`; ảnh đính kèm |
| TC-PU05-40 | AC11 | – | **T** đọc to mọi chuỗi của màn Sinh viên (`/`, `/chat`, `/threads`, `/practice`, `/library`, `/calendar`, `/me`) | Không từ kỹ thuật (bảng `DESIGN.md` §13); nút là động từ |
| TC-PU05-41 | AC11 | giả lập 375 × 330 | **A** `/chat`: focus ô nhập, `isIntersectingViewport`, gõ, gửi | Ô nhập với tới được; không bị thanh dưới che; (nếu có thiết bị thật: chụp ảnh, ghi kiểu máy) |
| TC-PU05-42 | AC12 | nhánh `sprint/3-pu-p1` | **S** `$FE lint && gbuild && bash scripts/ui-antipatterns.sh && bash scripts/ui-antipatterns.sh --selftest && bash scripts/lint-selftest.sh && $PW --grep-invert @real && $FE exec lhci autorun && (cd backend-go && go test -count=1 ./internal/contract/...); echo rc=$?` | `rc=0`; QC chạy **tự**, không chép từ handoff |
| TC-PU05-43 | AC13 | – | **S** `git grep -nE 'eyJ[A-Za-z0-9_-]{20,}\.' -- frontend/e2e frontend/lighthouserc.json \| wc -l`; đọc `playwright.config.ts` và `lighthouserc.json` tìm bí mật (`secret`, `key=`, mật khẩu) | `0`; không bí mật; token test sinh tại chỗ (`tok`) |
| TC-PU05-44 | AC13 | – | **S** ảnh mốc / dữ liệu mô phỏng: tìm tên / MSSV / email thật trong `frontend/e2e/**` và ảnh (OCR mắt trên 3 ảnh) | Chỉ dữ liệu giả (D44); không dữ liệu sinh viên thật |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Cổng xanh giả (không bắt được vi phạm) | 04, 18, 25, 32–34 |
| Ảnh mốc bị cập nhật để cho xanh | 06 |
| Ảnh mốc flaky | 05 |
| Ngưỡng Lighthouse bị nới | 20, 21 |
| CI xanh nhưng ở commit cũ | 28 |
| Nhánh thử còn sót trên remote | 35 |
| Bí mật / dữ liệu thật trong cổng | 43, 44 |
| Backend bị đụng | 36, 37 |

## Câu hỏi cho BA / PM
- **Q-QC-PU05-1** — TC-PU05-32…34: nhánh `ci/ui-drift` chứa 3 commit / 3 run riêng, hay một nhánh một run? AC8 viết số ít. QC chờ handoff nêu 3 `<ID>` — *chờ trả lời*.
- **Q-QC-PU05-2** — TC-PU05-28 cần CI xanh ở HEAD: nhánh `sprint/3-pu-p1` đã có `ci.yml` chạy trên GitHub của dự án. Nếu runner `lhci` không chạy được (thiếu Chrome), QC tính FAIL, không nới — đồng ý? — *chờ trả lời*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md (FEAT-ui-foundation, APPROVED 2026-10-03).

Tổng: 44 TC.
