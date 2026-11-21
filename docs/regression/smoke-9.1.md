# Smoke hồi quy sprint 9.1 — kết quả (SmkS4 2026-11-21: chạy nốt N còn lại + chạy lại F đã sửa)
Mã đã chạy: 425396a7 (sprint/11-p10-eval) · ep-r + lớp sao sạch QR91A…H (evidence-9.1/scripts/r91-clone.sh + r91-fixgrades.sh + r91-fixseed.sh) + stack smk91 (UX09-16) · Người chạy: qc2 / SmkS · Ngày: 2026-11-21
Tổng TC cũ: 414 · Lỗi thời (LT, loại khỏi tổng): 53 · Tổng tính = 361 · Đạt: 361 · Trượt (F xác nhận): 0 · K: 0 · Chưa chạy (N): 0   (P+F+K+N = 361)
Cách chạy: `evidence-9.1/run8.sh <spec> <lớp> [grep]` (spec ở docs/regression/specs-qc2/qc2-r91, config qc2g; khoá /tmp/qc2-pw.lock qua scripts/pwlock.sh, xoá bộ đếm auth mỗi 3 s, log logs/*.rerun7*.log); `make_overrides7.py` ghi P từ ✓ thật + kết quả chấm tay `results-smkS.tsv` (P/F/LT, dòng sau thắng); `build_smoke91.py` dựng `s91-rows.json` (đọc TC từ docs/sprints/9.1/qc); `build_smoke91_md.py` dựng tệp này. Dữ liệu sạch: 761987 đã bị QC ghi đè nên chạy trên lớp sao QR91*.
Lượt SmkS4: LT mới có nguồn: TC-UX13-1 (3b259963 + P-107-15), TC-UX08-28 (US-08 AC7 sửa 10.7, P-107-12/13). Chạy lại DEF đã sửa: DEF-R9.1-06 (6d127b88, TC-UX12-2), DEF-R9.1-07 (TC-UX16-14 theo P-11-75, ≤ 200 ms), DEF-R9.1-08 (e0ac8dc9, TC-UX09-16), TC-UX16-10 theo P-11-64. Chi tiết từng ca ở cột Ghi chú.

## Tổng hợp theo US
| US | TC | P | F | LT | N |
| --- | ---: | ---: | ---: | ---: | ---: |
| UX02 | 14 | 14 | 0 | 0 | 0 |
| UX03 | 18 | 18 | 0 | 0 | 0 |
| UX04 | 23 | 23 | 0 | 0 | 0 |
| UX21 | 12 | 12 | 0 | 0 | 0 |
| UX01 | 20 | 20 | 0 | 0 | 0 |
| UX05 | 12 | 7 | 0 | 5 | 0 |
| UX06 | 11 | 11 | 0 | 0 | 0 |
| UX07 | 18 | 18 | 0 | 0 | 0 |
| UX10 | 29 | 29 | 0 | 0 | 0 |
| UX11 | 11 | 11 | 0 | 0 | 0 |
| UX14 | 14 | 14 | 0 | 0 | 0 |
| UX15 | 22 | 22 | 0 | 0 | 0 |
| UX16 | 24 | 24 | 0 | 0 | 0 |
| UX12 | 23 | 23 | 0 | 0 | 0 |
| UX13 | 28 | 27 | 0 | 1 | 0 |
| UX19 | 15 | 15 | 0 | 0 | 0 |
| UX20 | 11 | 11 | 0 | 0 | 0 |
| UX18 | 12 | 12 | 0 | 0 | 0 |
| UX17 | 15 | 15 | 0 | 0 | 0 |
| UX91 | 4 | 4 | 0 | 0 | 0 |
| UX08 | 57 | 16 | 0 | 41 | 0 |
| UX09 | 21 | 15 | 0 | 6 | 0 |
| **Cộng** | **414** | **361** | **0** | **53** | **0** |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-UX02-1 | P | rg còn 5 dòng ngoài 13 tệp liệt kê: shared/lib/datetime.ts (mô-đun định dạng), insights/KnowledgeGap.tsx:14-15, observability/AdminView.tsx:19,46 (hai tệp không thuộc 13 tệp của AC1); 13 tệp đã sạch. logs/grep-batch.log / grep-batch2.log; so sánh với main bằng `git diff 2c15dfce^1 2c15dfce` (1b72f54 không có merge-base với HEAD) |
| TC-UX02-2 | P | spec w1 |
| TC-UX02-3 | P | spec w1 |
| TC-UX02-4 | P | spec w1 |
| TC-UX02-5 | P | spec w12b-real |
| TC-UX02-6 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0misc-real ✓ (qc-uiux91-rs0misc-real.rerun7j.log) |
| TC-UX02-7 | P | spec w1 |
| TC-UX02-8 | P | spec w1 |
| TC-UX02-9 | P | spec w1 |
| TC-UX02-10 | P | frontend/e2e/datetime.spec.ts (5 ca: bảng AC2, 23:59/00:01 ICT, ranh giới nửa đêm, null → «—», múi giờ máy khác) xanh trên bản dựng cổng; không có vitest ở HEAD (script test đã bỏ) |
| TC-UX02-11 | P | frontend/e2e/datetime.spec.ts (5 ca: bảng AC2, 23:59/00:01 ICT, ranh giới nửa đêm, null → «—», múi giờ máy khác) xanh trên bản dựng cổng; không có vitest ở HEAD (script test đã bỏ) |
| TC-UX02-12 | P | chụp ở HEAD trên ep-r (r91shots, logs/qc-uiux91-r91shots-real.rerun7k/m.log; overflowOK): shots/threads-1440.png + threads-375.png, shots/chat-sv-1440.png + chat-sv-375.png, shots/inbox-1440.png + inbox-375.png; ảnh trước docs/sprints/9.1/audit/shots/{threads__teacher__1440,chat__student__1440,inbox__teacher__1440}.png. Đọc ảnh sau: ngày giờ dạng «Chủ Nhật, 19:25» / «Hôm nay, 00:42» (AC2), bố cục không đổi gì khác; ảnh mốc CI cùng đợt: shots/visual-4b6389f1/ |
| TC-UX02-13 | P | spec w1 |
| TC-UX02-PQ | P | spec w12b-real |
| TC-UX03-1 | P | spec w1 |
| TC-UX03-2 | P | spec w1 |
| TC-UX03-3 | P | spec w1 |
| TC-UX03-4 | P | spec w1 |
| TC-UX03-5 | P | đủ bộ playwright dev (build:gate @1409b04d, devspecs-r6.log): 271 ✓, 225 skip, 0 đỏ vì toBeDisabled (0 dòng khớp); 22 đỏ chỉ ở qc2-r4/r4ui-today (spec QC cần fixture /tmp/qc2-i, không phải spec dev). git diff e2e 9.1: toBeDisabled/Enabled chỉ đổi nhãn (ed0aa499 AC3 US-UX91-18) + 1 toBeEnabled thêm (8ae5393f), không nới |
| TC-UX03-6 | P | spec w1 |
| TC-UX03-7 | P | spec w1 |
| TC-UX03-8 | P | spec w1 |
| TC-UX03-9 | P | spec w1 |
| TC-UX03-10 | P | spec w1 |
| TC-UX03-11 | P | RERUN 425396a7: «Phiên mới» không thuộc nút khoá US-03: /chat trống không có nút «Phiên mới» (UX13-9: 0 nút, rerun7u ✓) và sau khi gửi 1 tin nút hiện, là nút phụ, không khoá (UX13-10, rerun7v) |
| TC-UX03-12 | P | spec w1 |
| TC-UX03-13 | P | spec w1 |
| TC-UX03-14 | P | spec w1 |
| TC-UX03-15 | P | spec w1 |
| TC-UX03-16 | P | đọc nguồn: lý do khoá là chuỗi tĩnh / chữ của mã: «Chưa có công thức điểm đã xác nhận» (Gradebook.tsx:357, mặc định; finalize_blocked_reason = hằng BlockedNoScheme backend-go/internal/grade/scheme.go:300), «Còn N mục chưa rõ: <tên khối công thức>» (GradeScheme.tsx:184, tên khối cố định BLOCKS.short), «Chọn một bài tập trước.» / «Chưa có bài nộp.» (QueueTab.tsx:248-250): không tên SV / MSSV / điểm; ca TC-6…8 hiển thị đúng ở rw12b/w1 (logs/…rerun7j, devspec-devui.log: nút khoá có lý do ✓) |
| TC-UX03-17 | P | spec w1 |
| TC-UX03-18 | P | bộ dev-ui.spec (build:gate sẵn có, ep-r không dùng) 25 ✓ / 0 đỏ (logs/devspec-devui.log): «nút khoá có lý do: focus / rê / chạm hiện lý do, bấm không kích hoạt; lý do mà không khoá thì bỏ qua» ✓ + «confirm: disabledReason khoá nút xác nhận…» ✓; /dev/ui bị 404 ở bản như production (NEXT_PUBLIC_DEV_TOOLS) nên ảnh chụp bằng spec dev; ảnh mốc dev-ui-*, gradebook-* KHÔNG đổi ở 4b6389f1 (ca đặc biệt dưới màn đầu, dev-wave-1.md); ảnh sau gradebook-1440/375 ở shots/ (nút «Chốt điểm» khoá kèm lý do) |
| TC-UX04-1 | P | spec w1 đỏ vì selector `class*=Button-module` không khớp bản dựng cổng (tên lớp băm); bản sửa qc2-r91-mock-extra.spec.ts (data-variant) xanh: 5 biến thể nút, transform 0,97 ≤ 100 ms, trở lại |
| TC-UX04-2 | P | spec w1 đỏ vì selector `class*=Button-module` không khớp bản dựng cổng (tên lớp băm); bản sửa qc2-r91-mock-extra.spec.ts (data-variant) xanh: 5 biến thể nút, transform 0,97 ≤ 100 ms, trở lại |
| TC-UX04-3 | P | RERUN 425396a7 sau 6d127b88 (DEF-R9.1-06): node docs/sprints/9.1/qc/scripts/hover-check.mjs → «quy tắc :hover: 85, ngoài @media (hover: hover): 0»; luật rê của .radio nay là .radio:not(:checked) trong @media (hover: hover) |
| TC-UX04-4 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0misc-real ✓ (qc-uiux91-rs0misc-real.rerun7j.log) |
| TC-UX04-5 | P | tokens.css:73-75 --ep-fast 140ms, --ep-normal 200ms, --ep-slow 200ms (≤ 200) |
| TC-UX04-6 | P | spec w1 |
| TC-UX04-7 | P | DESIGN.md:492 «- Drawer/dialog: ≤ 200ms.» đúng một dòng (không có khoảng trắng quanh «/» so với mẫu rg) |
| TC-UX04-8 | P | rg `transition: width` trong frontend/src: 0 kết quả |
| TC-UX04-9 | P | spec w1 |
| TC-UX04-10 | P | spec w1 |
| TC-UX04-11 | P | spec w1 |
| TC-UX04-12 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4c-real ✓ (qc-uiux91-rw4c-real.rerun7j.log) |
| TC-UX04-13 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4c-real ✓ (qc-uiux91-rw4c-real.rerun7j.log) |
| TC-UX04-14 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4c-real ✓ (qc-uiux91-rw4c-real.rerun7j.log) |
| TC-UX04-15 | P | git diff 2c15dfce^1 2c15dfce -- frontend/package.json: chỉ bỏ framer-motion, embla-carousel*; không thêm thư viện chuyển động |
| TC-UX04-16 | P | evidence-9.1/scripts/keyframes-check.mjs: keyframes ngoài landing chỉ animate opacity/transform (+ ngoại lệ ep-saved background-color, spinner, ba chấm, shimmer); không stagger ngoài landing. Landing có stagger/nhiều keyframes theo P-107-12 (trang giới thiệu được chuyển động marketing) |
| TC-UX04-17 | P | spec w12b-real |
| TC-UX04-18 | P | ep-motion-safe: SubmissionReview.tsx:498,534, DataTableVirtual.tsx:209, AttendanceGrid.tsx:123; base.css:52 luật lưới 1ms còn nguyên, :55 .ep-motion-safe 200/160 ms |
| TC-UX04-19 | P | qc-uiux91-rn4-real rerun7a reduce=true: ô vẫn đổi nền (61→222 ms ≈ 160 ms, ep-saved:200), tắt ≤ 200 ms sau đỉnh, transform = none (US-04 AC9) |
| TC-UX04-20 | P | spec w1 |
| TC-UX04-21 | P | spec w1 |
| TC-UX04-22 | P | visual.spec chạy trên Linux ở CI (repo kagamikuro1024/EduPilot-v2 run 44343596969 @777bb5e2: Frontend e2e Playwright 1/4-4/4 success, gh run view); ảnh mốc đổi trong sprint 9.1 (git log --grep ^9.1 -- visual.spec.ts-snapshots): inbox-1440 + threads-390 (4b6389f1), hom-nay-gv-*, attendance/grading-queue/grading-review/gradebook (f8a744bd), chat/insights/observability/threads-390 (ed0aa499), chat + grading-review (e050e566) — đều có dòng giải thích trong handoff/dev-wave-1/3/4/5 + dev-sua-ci; dev-ui-* không đổi (không cần :active tĩnh) |
| TC-UX04-PQ | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0pq-real ✓ (qc-uiux91-rs0pq-real.rerun7j.log) |
| TC-UX21-1 | P | spec w1 |
| TC-UX21-2 | P | spec w1 |
| TC-UX21-3 | P | spec w1 |
| TC-UX21-4 | P | spec w1 |
| TC-UX21-5 | P | spec w1 |
| TC-UX21-6 | P | spec w1 |
| TC-UX21-7 | P | spec w1 |
| TC-UX21-8 | P | spec w1 |
| TC-UX21-9 | P | spec w1 |
| TC-UX21-10 | P | spec w1 |
| TC-UX21-11 | P | frontend/e2e/landing.spec.ts: «không gọi API nào (kể cả refresh)» (dòng 17-35) + «có cờ ⇒ vẫn gọi làm mới phiên» (dòng 38); tệp này chạy xanh 21/21 trên bản dựng cổng (logs/devspecs.log) |
| TC-UX21-PQ | P | probe qc-uiux91-rzprobe4-real rerun7a: sv.gioi đăng nhập → `/` giữ nguyên, h1 «Chào Trung» (Hôm nay của SV) từ giây 1 đến 14, 3 yêu cầu (refresh, me/courses, courses/‹id›/today), không có /me/today của GV; GV cùng điều kiện: h1 «4 việc cần xử lý hôm nay». Đỏ lượt 6 là chụp lúc gateway vừa restart (h1 trang giới thiệu) |
| TC-UX01-1 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw2-real ✓ (qc-uiux91-rw2-real.rerun7j.log) |
| TC-UX01-2 | P | 28 mảnh kiểm của bảng tc-wave-2: 0 kết quả (rg -F, bỏ Demo*.tsx, test, spec) |
| TC-UX01-3 | P | spec w2-real |
| TC-UX01-4 | P | spec w12c-real,w5r-real |
| TC-UX01-5 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0misc-real ✓ (qc-uiux91-rs0misc-real.rerun7j.log) |
| TC-UX01-6 | P | diff 3 tệp Demo*.tsx (7 dòng): chỉ nhãn «Threads»→«Hỏi đáp công khai» (US-07 AC4) và progressStyle (US-04 AC4); không đổi description (như QC 9.1 đã ghi nhận) |
| TC-UX01-7 | P | lcp.spec.ts: 7 route (/, /chat, /threads, /me, /inbox, /gradebook, /settings/llm) ✓ + /assignments/<id> ✓, ứng viên LCP trước phiên = chữ PreShell, sau phiên không ứng viên mới (devspecs-r6.log) |
| TC-UX01-8 | P | RERUN 425396a7 trên ep-r (qc-uiux91-rx1-real, logs/qc-uiux91-rx1-real.rerun7w.log): LCP bằng PerformanceObserver (cùng cách audit/scripts/shoot.mjs), 1440 px, 3 lượt, trung vị; «trước» = audit 9.1 (docs/sprints/9.1/audit/data/*.json, cùng vai, trước khi sửa) → sau: GV / 268→152 ms, /threads 504→444, /inbox 1752→508, /gradebook 504→212, /settings/llm 488→68; SV /chat 524→104, /me 2008→108; cả 7 sau ≤ trước + 100 ms (hiệu −60…−1900 ms). Ghi chú: «trước» là số của audit (1 lượt/route, khác thời điểm và dữ liệu), không dựng lại bản trước 9.1 |
| TC-UX01-9 | P | spec w2-real |
| TC-UX01-10 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw2-real ✓ (qc-uiux91-rw2-real.rerun7j.log) |
| TC-UX01-11 | P | spec w12c-real,w5r-real |
| TC-UX01-12 | P | git diff 2c15dfce^1 2c15dfce -U0 -- frontend/src / rg '^\+.*(Info/CircleHelp)': 0 dòng (CircleHelp/Info xuất hiện ở HEAD là icon nav Luyện đề và icon thông báo ở sprint 10, không phải câu giải thích) |
| TC-UX01-13 | P | logs/antipatterns.log: PageHeader description tĩnh > 40 ký tự trên bản sao → ✗ «Câu mô tả tĩnh dưới tiêu đề», exit 1 |
| TC-UX01-14 | P | logs/antipatterns.log: description={"abc"} và description={dynamicVar} → không ✗, exit 0 |
| TC-UX01-15 | P | logs/antipatterns.log: --selftest = 32 / 32 phép bắt được (có phép «Câu mô tả tĩnh dưới tiêu đề») |
| TC-UX01-16 | P | spec w12c-real |
| TC-UX01-17 | P | spec w2-real |
| TC-UX01-18 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0pq-real ✓ (qc-uiux91-rs0pq-real.rerun7j.log) |
| TC-UX01-19 | P | ảnh sau ở HEAD 1440 + 375: shots/home-gv-*, home-sv-*, chat-sv-*, threads-*, inbox-*, gradebook-* (.png); ảnh trước audit/shots/{home__teacher,home__student,chat__student,threads__teacher,inbox__teacher,gradebook__teacher}__1440.png. Mất dòng phụ dưới h1 (threads-1440.png chỉ còn h1 «Hỏi đáp công khai»); visual.spec Linux ở CI xanh (run 44343596969 @777bb5e2) |
| TC-UX01-PQ | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0pq-real ✓ (qc-uiux91-rs0pq-real.rerun7j.log) |
| TC-UX05-4 | P | spec w2-real |
| TC-UX05-6 | P | spec w12b-real |
| TC-UX05-7 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0misc-real ✓ (qc-uiux91-rs0misc-real.rerun7j.log) |
| TC-UX05-8 | P | spec w12b-real |
| TC-UX05-9 | P | spec w2-real |
| TC-UX05-10 | P | spec w2-real |
| TC-UX05-12 | P | shots/inbox-1440.png + inbox-375.png (ep-r, HEAD); trước audit/shots/inbox__teacher__1440.png. Dữ liệu cố định của visual.spec có hộp thư nên ảnh mốc đổi ở 4b6389f1 (inbox-1440: 85732→85363 B, shots/visual-4b6389f1/) và dev giải thích ở handoff/dev-wave-1.md (dòng phụ hộp thư, US-01); CI Linux xanh |
| TC-UX06-1 | P | spec w2-real |
| TC-UX06-2 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw12b-real ✓ (qc-uiux91-rw12b-real.rerun7n.log) |
| TC-UX06-3 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs06-real ✓ (qc-uiux91-rs06-real.rerun7j.log) |
| TC-UX06-4 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs06-real ✓ (qc-uiux91-rs06-real.rerun7j.log) |
| TC-UX06-5 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs06-real ✓ (qc-uiux91-rs06-real.rerun7j.log) |
| TC-UX06-6 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs06-real ✓ (qc-uiux91-rs06-real.rerun7n.log) |
| TC-UX06-7 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs06-real ✓ (qc-uiux91-rs06-real.rerun7b.log) |
| TC-UX06-8 | P | spec w2-real |
| TC-UX06-9 | P | spec w2-real |
| TC-UX06-10 | P | visual.spec Linux xanh ở CI (run 44343596969 @777bb5e2). Ảnh đổi cùng commit story 6 (4b6389f1) là inbox-1440 + threads-390: xem shots/visual-4b6389f1/ — chỉ đổi nhãn menu (Threads→Hỏi đáp công khai, Insights→Lỗ hổng kiến thức, Analytics→Số liệu lớp học: US-07) và dòng phụ hộp thư (US-01); không ảnh nào đổi vì chọn lớp mặc định (US-06) |
| TC-UX06-PQ | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs06-real ✓ (qc-uiux91-rs06-real.rerun7b.log) |
| TC-UX07-1 | P | spec w2-real |
| TC-UX07-2 | P | spec w2-real |
| TC-UX07-3 | P | spec w2-real |
| TC-UX07-4 | P | spec w2-real |
| TC-UX07-5 | P | spec w2-real |
| TC-UX07-6 | P | spec w2-real |
| TC-UX07-7 | P | spec w2-real |
| TC-UX07-8 | P | RERUN 2026-11-16 sau 7322ac62 (DEF-R9.1-01): grep "Kiểm duyệt Threads" frontend/src = 0 |
| TC-UX07-9 | P | spec w12c-real |
| TC-UX07-10 | P | spec w12b-real |
| TC-UX07-11 | P | spec w2-real |
| TC-UX07-12 | P | spec w12b-real |
| TC-UX07-13 | P | spec w12b-real |
| TC-UX07-14 | P | diff shell.spec.ts của merge 9.1: chỉ đổi nhãn nav + chuyển sang toHaveText tự thử lại; không xoá/nới assertion |
| TC-UX07-15 | P | spec w2-real |
| TC-UX07-16 | P | rg DESIGN.md: còn duy nhất dòng 42 «Hỏi đáp công khai (Threads)» là ghi chú đổi tên có chủ ý |
| TC-UX07-17 | P | shots/threads-1440.png, home-gv-1440.png, chat-sv-1440.png, inbox-1440.png, gradebook-1440.png, settings-llm-1440.png (+375); sidebar có nhãn mới «Hỏi đáp công khai», «Lỗ hổng kiến thức», «Số liệu lớp học» (đọc settings-llm-1440.png); trước audit/shots/…__teacher__1440.png (nhãn Threads / Insights / Analytics) |
| TC-UX07-PQ | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0pq-real ✓ (qc-uiux91-rs0pq-real.rerun7j.log) |
| TC-UX10-1 | P | spec w3 |
| TC-UX10-2 | P | spec w3 |
| TC-UX10-3 | P | spec w3 |
| TC-UX10-4 | P | spec w3 |
| TC-UX10-5 | P | spec w3 |
| TC-UX10-6 | P | spec w3 |
| TC-UX10-7 | P | spec w3 |
| TC-UX10-8 | P | spec w3 |
| TC-UX10-9 | P | bash scripts/ui-antipatterns.sh trên cây HEAD: exit 0, không ✗ |
| TC-UX10-10 | P | spec w3 |
| TC-UX10-11 | P | spec w3 |
| TC-UX10-12 | P | spec w3 |
| TC-UX10-13 | P | spec w3 |
| TC-UX10-14 | P | spec w3 |
| TC-UX10-15 | P | spec w3 |
| TC-UX10-16 | P | spec w3 |
| TC-UX10-17 | P | spec w3 |
| TC-UX10-18 | P | spec w3 |
| TC-UX10-19 | P | spec w3 |
| TC-UX10-20 | P | spec w3 |
| TC-UX10-21 | P | spec w3 |
| TC-UX10-22 | P | spec w3 |
| TC-UX10-23 | P | spec w3 |
| TC-UX10-24 | P | spec w3 |
| TC-UX10-25 | P | spec w3 |
| TC-UX10-26 | P | today.spec.ts: 19 ✓ desktop, 0 ✘ (logs/devspecs-r6.log, build:gate @1409b04d); diff 9.1 (6a0e8584): +20 expect, −0; fixture có overdue/high/normal/EXAM_SIMILARITY (today.spec.ts:150-172); ảnh mốc hom-nay-gv-1440/390 có trong visual.spec.ts-snapshots |
| TC-UX10-27 | P | spec w3 |
| TC-UX10-28 | P | shots/hom-nay-gv-1440.png + hom-nay-gv-375.png (ep-r, HEAD, teacher; overflowOK); ảnh trước audit/shots/home__teacher__1440.png (+375): đủ ảnh 1440 / 375; ảnh mốc visual hom-nay-gv-1440/390 trong visual.spec.ts-snapshots, CI Linux xanh |
| TC-UX10-PQ | P | spec w3 |
| TC-UX11-1 | P | spec w3 |
| TC-UX11-2 | P | spec w3 |
| TC-UX11-3 | P | spec w3 |
| TC-UX11-4 | P | spec w3 |
| TC-UX11-5 | P | spec w3 |
| TC-UX11-6 | P | spec w3 |
| TC-UX11-7 | P | spec w3 |
| TC-UX11-8 | P | spec w3 |
| TC-UX11-9 | P | spec w3 |
| TC-UX11-10 | P | shots/home-sv-1440.png + home-sv-375.png (ep-r, HEAD, sv.gioi; đọc ảnh: «Chào Trung», Việc nên làm tiếp, Hôm nay, Tiếp tục học); trước audit/shots/home__student__1440.png + __375.png: đủ ảnh; CI Linux xanh (visual.spec) |
| TC-UX11-PQ | P | spec w3 |
| TC-UX14-1 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX14-2 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX14-3 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4b-real ✓ (qc-uiux91-rw4b-real.rerun7j.log) |
| TC-UX14-4 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX14-5 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4b-real ✓ (qc-uiux91-rw4b-real.rerun7j.log) |
| TC-UX14-6 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX14-7 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4b-real ✓ (qc-uiux91-rw4b-real.rerun7j.log) |
| TC-UX14-8 | P | qc-uiux91-rn4-real rerun7c (QR91H): chọn «Bài tập 04» (0 bài nộp) → «Chấm tất cả» aria-disabled=true + tooltip «Chưa có bài nộp.»; chọn «Bài tập 03» (có bài) → aria-disabled=null, bấm được |
| TC-UX14-9 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7j.log) |
| TC-UX14-10 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7j.log) |
| TC-UX14-11 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4b-real ✓ (qc-uiux91-rw4b-real.rerun7j.log) |
| TC-UX14-12 | P | shots/grading-queue-1440.png + grading-queue-375.png (ep-r, HEAD); trước audit/shots/grading__teacher__1440.png; ảnh mốc grading-queue-1440 đổi ở f8a744bd (CI Linux xanh) |
| TC-UX14-PQ | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7j.log) |
| TC-UX14-PG | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7j.log) |
| TC-UX15-1 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-2 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-3 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-4 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-5 | P | rw4 TC-UX15-4 chạy reduce=true: có đoạn tô sáng, giữ + tắt < 1,5 s (rerun7g ✓ 19,2 s) |
| TC-UX15-6 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-7 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-8 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4b-real ✓ (qc-uiux91-rw4b-real.rerun7j.log) |
| TC-UX15-9 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4b-real ✓ (qc-uiux91-rw4b-real.rerun7j.log) |
| TC-UX15-10 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-11 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-12 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-13 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4c-real ✓ (qc-uiux91-rw4c-real.rerun7j.log) |
| TC-UX15-14 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4g-real ✓ (qc-uiux91-rw4g-real.rerun7j.log) |
| TC-UX15-15 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-16 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-17 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-18 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-19 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7g.log) |
| TC-UX15-20 | P | assignment-lifecycle.spec.ts: 47 ✓ / 0 ✘ (devspecs-r6.log); diff 9.1: các expect bị bỏ đều thay bằng bản theo AC (cột «Duyệt», «Lệch … so với lượt 2», N/M, xung đột AC7); ảnh grading-review-1440 có |
| TC-UX15-21 | P | shots/grading-review-1440.png + grading-review-375.png (ep-r, HEAD; vào bằng liên kết «Duyệt»); trước audit/shots/grading-ID__teacher__1440.png (audit không chụp 375); ảnh mốc grading-review-1440 đổi ở f8a744bd + e050e566 (CI Linux xanh) |
| TC-UX15-PQ | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0pq-real ✓ (qc-uiux91-rs0pq-real.rerun7j.log) |
| TC-UX16-1 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4f-real ✓ (qc-uiux91-rw4f-real.rerun7j.log) |
| TC-UX16-2 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7i.log) |
| TC-UX16-3 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7i.log) |
| TC-UX16-4 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7i.log) |
| TC-UX16-5 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4f-real ✓ (qc-uiux91-rw4f-real.rerun7j.log) |
| TC-UX16-6 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7i.log) |
| TC-UX16-7 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7i.log) |
| TC-UX16-8 | P | spec w4 |
| TC-UX16-9 | P | spec w4,w4-real |
| TC-UX16-10 | P | P-11-64 (AC4 mới): lớp sao QR91B (30 SV đủ điểm, không SV Liệt): sổ điểm không có cột «Trạng thái», không có «Thiếu điểm: N» (logs/qc-uiux91-rzprobe5-real.rerun7b.log). Đối chứng QR91C (3 SV thiếu Cuối kỳ): cột hiện, 3 ô «Thiếu điểm» + «Liệt», tiêu đề phụ «Thiếu điểm: 3»; QR91A (1 SV Liệt): cột hiện với đúng 1 ô |
| TC-UX16-11 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7i.log) |
| TC-UX16-12 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4g-real ✓ (qc-uiux91-rw4g-real.rerun7j.log) |
| TC-UX16-13 | P | qc-uiux91-rn4-real rerun7a (QR91H): sửa ô sổ điểm → Enter giữ cột + xuống dòng kế (row 1), Tab sang ô nhập kế (cột 1→3, bỏ cột Kiểm tra hằng tuần chỉ đọc); animation ep-saved:200, nền oklab xanh 28→218 ms rồi về trong suốt, không transform |
| TC-UX16-14 | P | RERUN 425396a7 theo P-11-75 (PM 2026-11-21: AC6 sửa thành «đổi màu tức thì và tắt sau ≤ 200 ms»; DEF-R9.1-07 đóng): qc-uiux91-rn4-real rerun7a reduce=true: nền ô đổi 61→222 ms (≈ 161 ms ≤ 200 ms), animation ep-saved:200, transform = none (logs/qc-uiux91-rn4-real.rerun7a.log); mã không đổi từ lượt đó (chỉ tài liệu) |
| TC-UX16-15 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX16-16 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX16-17 | P | qc-uiux91-rn4-real rerun7d (lớp thử QR91D): bấm đúp xác nhận «Chốt điểm» → sổ LOCKED, snapshot_version = 1, đúng 1 POST finalize (Idempotency-Key ep-2f40332e…); không bản chốt thứ hai |
| TC-UX16-18 | P | spec w4 |
| TC-UX16-19 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX16-20 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX16-21 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4f-real ✓ (qc-uiux91-rw4f-real.rerun7j.log) |
| TC-UX16-22 | P | gradebook.spec.ts: 42 ✓ / 0 ✘; diff 9.1: số đệm 2 chữ số (8,5→8,50), cột Trạng thái «Đủ»→rỗng, «Đã công bố» → aria-label: đúng AC1/AC4, không nới; ảnh gradebook-1440/390 có |
| TC-UX16-23 | P | shots/gradebook-1440.png + gradebook-375.png (ep-r, HEAD); trước audit/shots/gradebook__teacher__1440.png (audit không chụp 375 của giảng viên); ảnh mốc gradebook-1440/390 đổi ở f8a744bd (CI Linux xanh) |
| TC-UX16-24 | P | RERUN7 trên lớp sao QR91G (seed gốc: điểm Giữa/Cuối kỳ IMPORT v1, điểm danh + phát biểu về bản /tmp/smk91-pristine.dump qua scripts/r91-fixgrades.sh + r91-fixseed.sh): rw4d khớp 30/30 (logs/qc-uiux91-rw4d-real.rerun7c.log). Lệch 28/30 ở QR91A là dữ liệu sao có phát biểu/điểm danh do QC ghi thêm |
| TC-UX12-1 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX12-2 | P | RERUN S9R sau sửa 6d127b88 (ep-r, lớp QR91A, ru1 «UX12-2», logs/qc-uiux91-ru1-real.rerun7z2.log ✓): chọn Muộn / Vắng phép / Vắng ở hàng 5 → viền ba chấm rgb(206,22,40) (đỏ) cả khi chuột xa lẫn khi rê lên đúng ô (trước sửa: lab(45 4 2) xám); 375 cảm ứng: (hover: hover) = false, 2 luật `.cell:hover .radio:not(:checked)` nằm trong @media (hover: hover) and (pointer: fine) ⇒ applies=false |
| TC-UX12-3 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4b-real ✓ (qc-uiux91-rw4b-real.rerun7j.log) |
| TC-UX12-4 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX12-5 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4b-real ✓ (qc-uiux91-rw4b-real.rerun7j.log) |
| TC-UX12-6 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX12-7 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX12-8 | P | rg "0[,.]25" frontend/src/features/attendance: 0 kết quả |
| TC-UX12-9 | P | RERUN S9R (rw4 «UX12-9/10/11/15/19», logs/qc-uiux91-rw4-real.rerun7z6.log ✓, 375×812): chọn 2 SV Vắng → thanh «Lưu điểm danh · 20 có mặt · 10 vắng» (vắng +2 so với 8 trước đó; vắng = tổng − Có mặt − Muộn, RealAttendance.tsx:283), position fixed, cao 44 px, đáy 748 < đỉnh thanh điều hướng 755. Spec đã sửa 3 lỗi kịch bản (nút ẩn trùng tên, đếm vắng gồm Vắng phép, hàng thật ở 375 là li) |
| TC-UX12-10 | P | RERUN S9R (cùng lượt z6, đã cuộn hết cỡ): thẻ cuối «Vũ Minh Long» cách thanh lưu 289 px (≥ 0), không bị che |
| TC-UX12-11 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4f-real ✓ (qc-uiux91-rw4f-real.rerun7j.log) |
| TC-UX12-12 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7j.log) |
| TC-UX12-13 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX12-14 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4f-real ✓ (qc-uiux91-rw4f-real.rerun7j.log) |
| TC-UX12-15 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4f-real ✓ (qc-uiux91-rw4f-real.rerun7j.log) |
| TC-UX12-16 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rn4-real ✓ (qc-uiux91-rn4-real.rerun7p.log) |
| TC-UX12-17 | P | qc-uiux91-rn4-real rerun7b (QR91H): TA sửa hàng 25 → Vắng, tự lưu; GV (trang cũ) sửa cùng hàng → Muộn: 409, hàng hiện «Phạm Quốc Bảo đã đánh Vắng», nút «Giữ của tôi» + «Lấy của Phạm Quốc Bảo», lựa chọn Muộn của GV còn |
| TC-UX12-18 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX12-19 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4f-real ✓ (qc-uiux91-rw4f-real.rerun7j.log) |
| TC-UX12-20 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7h.log) |
| TC-UX12-21 | P | attendance.spec.ts: 20 ✓ desktop (+1 mobile) / 0 ✘; diff 9.1: speak-count «1»→/^\+1\b/ (AC5), +3 expect; ảnh attendance-1440/390 có |
| TC-UX12-22 | P | shots/attendance-1440.png + attendance-375.png (ep-r, HEAD); trước audit/shots/attendance__teacher__1440.png + __375.png: đủ ảnh; ảnh mốc attendance-1440/390 đổi ở f8a744bd (CI Linux xanh) |
| TC-UX12-PQ | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw4-real ✓ (qc-uiux91-rw4-real.rerun7j.log) |
| TC-UX13-2 | P | scripts/ui-antipatterns.sh exit 0, 12 phép ✓ (logs/antipatterns-r6.log) + panels.spec.ts 59 ✓ / 0 ✘ (devspecs-r6.log; NEST = 0, TITLE = 0 mọi route × vai) |
| TC-UX13-3 | P | spec w5i-real |
| TC-UX13-4 | P | spec w5i-real |
| TC-UX13-5 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec ../docs/regression/specs-qc2/qc2-r91/qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7u.log) |
| TC-UX13-6 | P | RERUN 425396a7 (r91sv 7w, sv.kha): ⋯ → Xoá: hàng 49→48 ngay, 0 hộp thoại sau khi bấm Xoá, dòng «Đã xoá · Hoàn tác» hiện (logs/qc-uiux91-r91sv-real.rerun7w.log) |
| TC-UX13-7 | P | RERUN S9R (r91sv «UX13-6/7/8/10/15», logs/qc-uiux91-r91sv-real.rerun7z.log ✓, sv.kha): chấm theo AC3 US-UX91-13 (useUndoableAction hiện có: DELETE ngay rồi POST restore). Bấm Hoàn tác trong 5 s → hàng 49→50 trở lại đúng chỗ; yêu cầu ghi nhận: POST sessions, POST messages, DELETE sessions/‹id›, POST sessions/‹id›/restore. Lệch TC cũ («không DELETE») đã ghi ở defects-9.1.md mục SmkS4: TC ≠ AC, mã theo AC |
| TC-UX13-8 | P | RERUN S9R (cùng lượt z): Xoá, không Hoàn tác, đợi 6,5 s → DELETE tổng 2 (≥ 2, tức có DELETE của lần xoá thứ hai); tải lại → lịch sử không còn phiên vừa xoá (đối chiếu nội dung phiên) |
| TC-UX13-9 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec ../docs/regression/specs-qc2/qc2-r91/qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7u.log) |
| TC-UX13-10 | P | RERUN 425396a7 (r91sv 7v, sv.kha): sau khi gửi 1 tin nút «Phiên mới» hiện, data-variant=secondary, nền lab(100 0 0) (trắng, không --ep-red / rgb(206,22,40)); đo trước dòng che PII nên giữ dù ca PII còn chờ (logs/qc-uiux91-r91sv-real.rerun7v.log) |
| TC-UX13-11 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec ../docs/regression/specs-qc2/qc2-r91/qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7u.log) |
| TC-UX13-12 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec ../docs/regression/specs-qc2/qc2-r91/qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7u.log) |
| TC-UX13-13 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec ../docs/regression/specs-qc2/qc2-r91/qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7u.log) |
| TC-UX13-14 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5a-real ✓ (qc-uiux91-rw5a-real.rerun7j.log) |
| TC-UX13-15 | P | RERUN 425396a7 (r91sv 7w, sv.kha): gửi «Em tên Nguyễn Minh Trung, MSSV 20229002, email sv.kha@…» → dòng «Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI» (N = 2 ≥ 2) còn nguyên (logs/qc-uiux91-r91sv-real.rerun7w.log); lượt đầu với sv.gioi bị bộ chặn «Mình chỉ xem được dữ liệu của chính bạn» vì MSSV/email của người khác (đúng thiết kế) |
| TC-UX13-16 | P | RERUN 425396a7 trên ep-r (logs/ux13-16-pii-grep.log): câu «Em tên Bùi Thanh Khải, MSSV 20229002, email sv.kha@edupilot.local, hỏi điểm giữa kỳ» (tên thật của sv.kha) → SSE notice masked=3; llm_request_payloads: 0 dòng có tên / MSSV / email thật, câu hỏi gửi LLM = «Em tên [[SV_1]], MSSV [[MSSV_1]], email [[EMAIL_1]]…»; log gateway + worker (5 phút): 0 dòng chứa tên / MSSV / email. Ghi chú: tên gõ trong TC («Nguyễn Minh Trung») không có trong danh sách lớp của ep-r (seed khác) nên KHÔNG bị che (MSSV + email vẫn che, masked=2) — che tên dựa vào danh sách lớp |
| TC-UX13-17 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec ../docs/regression/specs-qc2/qc2-r91/qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7u.log) |
| TC-UX13-18 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5g-real ✓ (qc-uiux91-rw5g-real.rerun7j.log) |
| TC-UX13-19 | P | spec w5c-real |
| TC-UX13-20 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5g-real ✓ (qc-uiux91-rw5g-real.rerun7j.log) |
| TC-UX13-21 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5a-real ✓ (qc-uiux91-rw5a-real.rerun7j.log) |
| TC-UX13-22 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5g-real ✓ (qc-uiux91-rw5g-real.rerun7j.log) |
| TC-UX13-23 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5g-real ✓ (qc-uiux91-rw5g-real.rerun7j.log) |
| TC-UX13-24 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5a-real ✓ (qc-uiux91-rw5a-real.rerun7j.log) |
| TC-UX13-25 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5a-real ✓ (qc-uiux91-rw5a-real.rerun7j.log) |
| TC-UX13-26 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5a-real ✓ (qc-uiux91-rw5a-real.rerun7j.log) |
| TC-UX13-27 | P | private-chat.spec.ts: 24 ✓ / 0 ✘; diff 9.1 (ed0aa499, 3002ce33): +11 expect, −0; ảnh chat-1440/390 có |
| TC-UX13-28 | P | shots/chat-sv-1440.png + chat-sv-375.png (ep-r, HEAD, sv.gioi); trước audit/shots/chat__student__1440.png + __375.png: đủ ảnh; ảnh mốc chat-1440/390 đổi ở ed0aa499 + e050e566 (CI Linux xanh) |
| TC-UX19-1 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX19-2 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX19-3 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX19-4 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX19-5 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX19-6 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5f-real ✓ (qc-uiux91-rw5f-real.rerun7j.log) |
| TC-UX19-7 | P | spec w5e-real |
| TC-UX19-8 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX19-9 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX19-10 | P | spec w5i-real |
| TC-UX19-11 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5f-real ✓ (qc-uiux91-rw5f-real.rerun7j.log) |
| TC-UX19-12 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5f-real ✓ (qc-uiux91-rw5f-real.rerun7j.log) |
| TC-UX19-13 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX19-14 | P | shots/threads-1440.png + threads-375.png (ep-r, HEAD); trước audit/shots/threads__teacher__1440.png + threads__student__375.png; ảnh mốc threads-390 đổi ở 4b6389f1 (shots/visual-4b6389f1/) |
| TC-UX19-PG | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs19-real (qc-uiux91-rs19-real.rerun6.log) xanh |
| TC-UX20-1 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7b.log) |
| TC-UX20-2 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7b.log) |
| TC-UX20-3 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7b.log) |
| TC-UX20-4 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7b.log) |
| TC-UX20-5 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rs0misc-real ✓ (qc-uiux91-rs0misc-real.rerun7j.log) |
| TC-UX20-6 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5g-real ✓ (qc-uiux91-rw5g-real.rerun7j.log) |
| TC-UX20-7 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-rw5g-real ✓ (qc-uiux91-rw5g-real.rerun7j.log) |
| TC-UX20-8 | P | spec w5i-real |
| TC-UX20-9 | P | RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec qc-uiux91-r91sv-real ✓ (qc-uiux91-r91sv-real.rerun7b.log) |
| TC-UX20-10 | P | me.spec.ts: 16 ✓ / 0 ✘; diff 9.1: calc li 3→5 thành phần (AC1), +6 expect, câu rỗng đổi chữ theo AC2; không nới |
| TC-UX20-11 | P | shots/me-sv-375.png (+1440) (ep-r, HEAD, sv.gioi); trước audit/shots/me__student__375.png: đủ ảnh |
| TC-UX18-1 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs18-real (qc-uiux91-rs18-real.rerun6.log) xanh |
| TC-UX18-2 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs18-real (qc-uiux91-rs18-real.rerun6.log) xanh |
| TC-UX18-3 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs18-real (qc-uiux91-rs18-real.rerun6.log) xanh |
| TC-UX18-4 | P | RERUN S9R (ru1 «UX18-4/6/8/PQ», logs/qc-uiux91-ru1-real.rerun7z.log ✓, GV, lớp 761987): 22 công tắc, 100 % có tên «Dùng cho AI: bật/tắt» (7+4) hoặc «Hiện cho sinh viên: bật/tắt» (8+3); màu đường ray ::before bật rgb(35,31,32) khác tắt lab(87.1 3.7 2.0), đồng nhất mọi hàng (1 giá trị mỗi trạng thái) |
| TC-UX18-5 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs18-real (qc-uiux91-rs18-real.rerun6.log) xanh |
| TC-UX18-6 | P | RERUN S9R (cùng lượt z): 3 tài liệu «Hiện cho sinh viên: tắt» («QC Đáp án thiếu», «Đáp án đề mẫu luyện tập», «Đáp án đề tham khảo tuần 5») không có trong /library của sv.kha (274 ký tự) |
| TC-UX18-7 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs18-real (qc-uiux91-rs18-real.rerun6.log) xanh |
| TC-UX18-8 | P | RERUN S9R (cùng lượt z): SV /documents → «Bạn không có quyền xem màn này … dành cho giảng viên và trợ giảng»; API GET /courses/‹id›/documents = 403; /library mở được (274 ký tự) |
| TC-UX18-9 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs18-real (qc-uiux91-rs18-real.rerun6.log) xanh |
| TC-UX18-10 | P | shots/documents-1440.png + documents-375.png (ep-r, HEAD, teacher); trước audit/shots/documents__teacher__1440.png: đủ ảnh |
| TC-UX18-PQ | P | RERUN S9R (cùng lượt z): GV gọi GET /courses/01a1df30-…/documents của lớp mình không phụ trách → 403 |
| TC-UX18-ID | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs18-real (qc-uiux91-rs18-real.rerun6.log) xanh |
| TC-UX17-1 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-2 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-3 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-4 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-5 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-6 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-7 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-8 | P | RERUN 425396a7 (ru1 7u): admin /observability → hàng yêu cầu 01a1fbe8 → nhập lý do → «Xem chi tiết» thấy «Nội dung đã che [system] Bạn là trợ giảng…»; audit_log VIEW_DETAIL entity llm_request 4 → 5 (đúng +1, actor admin, reason «QC SmkS kiểm tra nội dung đã che») (logs/qc-uiux91-ru1-real.rerun7u.log) |
| TC-UX17-9 | P | RERUN 425396a7 (ru1 7u, lớp QR91A chưa có insight_reports): main = «Lỗ hổng kiến thức · Chưa có báo cáo · Tạo báo cáo mới»; đúng 1 nút «Tạo báo cáo mới» (ngoài ra chỉ 4 nút chọn kỳ 7/14/30 ngày/Toàn kỳ); EmptyState «Chưa có báo cáo» (logs/qc-uiux91-ru1-real.rerun7u.log) |
| TC-UX17-10 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-11 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-12 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-13 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX17-14 | P | shots/insights-1440.png, observability-1440.png (+375) (ep-r, HEAD, teacher); trước audit/shots/insights__teacher__1440.png, observability__teacher__1440.png: đủ ảnh; ảnh mốc insights-/observability- đổi ở ed0aa499 (CI Linux xanh) |
| TC-UX17-PQ | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs17-real (qc-uiux91-rs17-real.rerun6.log) xanh |
| TC-UX91-X1 | P | như UX08-37 (AC10a US-08) |
| TC-UX91-X2 | P | bộ playwright dev 271 ✓ / 0 đỏ spec dev (logs/devspecs-r6.log); scripts/ui-antipatterns.sh 12 phép ✓ (antipatterns-r6.log); eslint cây theo dõi: 36 lỗi đều ở e2e/qc2-* (spec QC), 0 ở src/ và spec dev (lint-tracked-r6.log); CI run 44343596969 @777bb5e2: Frontend (lint + build), 4 shard Playwright và 4 job Lighthouse (simulate A/B, devtools A/B) success |
| TC-UX91-X3 | P | git diff --stat 2c15dfce^1 2c15dfce: không có backend-go/db/migrations mới; không `fetch(` trần mới ngoài shared/api; Demo*.tsx chỉ đổi như UX01-6 |
| TC-UX91-X4 | P | spec w5h-real |
| TC-UX08-5 | P | RERUN 425396a7: hover-check.mjs 85 luật :hover, 0 ngoài @media (hover: hover); Landing.module.css sạch (cảm ứng: kiểm .radio ở TC-UX12-2) |
| TC-UX08-16 | P | spec w6-real |
| TC-UX08-17 | P | spec w6-real |
| TC-UX08-18 | P | spec w6-real |
| TC-UX08-19 | P | spec w6-real |
| TC-UX08-21 | P | spec w6-real |
| TC-UX08-37 | P | AC10a: story-chat/review/gradebook-v1-1280.webp chụp lại ở commit aa4343f8 (15/11, 10.7 lô 8–9) sau đợt 3–5; xem ảnh story-review: có «Đã duyệt 0/4», phím A, «Duyệt bài», nhãn nav mới (Hỏi đáp công khai, Lỗ hổng kiến thức) = UI hiện hành |
| TC-UX08-43 | P | Lighthouse mobile 3 lượt: LCP 2419/2190/2087 ms, CLS 0; desktop LCP 437 ms (lượt 1 lạnh 2177 ms), CLS 0; a11y 1,0. Máy chủ ép tải ~40 nên số mobile sát trần; ghi nhận p.lead là LCP ở 375 (DEF-R9.1-05) |
| TC-UX08-44 | P | logs/r91land.log: JS tĩnh route / = 214,5 KB gzip (11 tệp) ≤ 256 KB; script nội tuyến tự viết 0,48 KB gzip (≤ 5 KB, AC13) |
| TC-UX08-45 | P | git diff package.json merge 9.1: chỉ bỏ framer-motion/embla; curl / (không JS) có h1 «Vận hành lớp học có trợ lý AI» và nội dung chính (Server Component) |
| TC-UX08-48 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs0land-real (qc-uiux91-rs0land-real.rerun6.log) xanh |
| TC-UX08-50 | P | axe landing / ở 1440 và 375, 3 lượt mỗi cỡ: 6/6 ✓, 0 serious/critical (logs/qc-uiux91-r91axe-real.rerun4.log; DEF-R9.1-03 đã sửa theo P-11-40) |
| TC-UX08-51 | P | spec w6-real |
| TC-UX08-52 | P | spec w6-real |
| TC-UX08-54 | P | spec w6-real |
| TC-UX08-PQ | P | logs/r91land.log: khách mở /chat, /gradebook → /login?next=…; HTML `/` không có @edupilot.local, Edupilot#, Bearer/JWT, sk-…, MSSV 2022xxxx |
| TC-UX09-1 | P | ffprobe gioi-thieu-v1.mp4: duration 64,73 s (60–90) |
| TC-UX09-2 | P | ffprobe: h264 1440×900, không luồng audio (cả .mp4 và .webm vp9 1440×900) |
| TC-UX09-3 | P | ffmpeg trích 65 khung (1 khung/s), 22 khung đã xem qua ảnh ghép frames/sheet-1..3.png: thẻ tiêu đề → chat riêng (có che tên: «Đã ẩn 3 thông tin cá nhân trước khi gửi cho AI») → Hộp thư hỗ trợ → Chấm bài + duyệt bài → Sổ điểm + giải trình → Hôm nay → thẻ kết; con trỏ đỏ hiện; phụ đề tiếng Việt in vào hình, dấu đúng, không tràn. Người xem: qc2 qua công cụ đọc ảnh; người thật chấm cảm nhận mượt: ngoài phạm vi kiểm tự động |
| TC-UX09-7 | P | public/landing/: gioi-thieu-v1.mp4 1,06 MB, .webm 1,56 MB, .webp 51,7 KB (< 8 MB); curl -I https://localhost:1143/landing/gioi-thieu-v1.mp4 → 200, accept-ranges: bytes |
| TC-UX09-8 | P | logs/r91land.log: <video> trong hộp thoại có controls, preload=none, playsinline, poster=/landing/gioi-thieu-v1.webp, không autoplay; 0 request video trước khi bấm «Xem video» |
| TC-UX09-9 | P | Lighthouse mobile devtools-throttle 3 lượt LCP 2419/2190/2087 ms (≤ 2500), CLS 0, desktop 437 ms (+ 1 lượt lạnh 2177 ms khi máy tải 40): logs/lh-*.json; poster không còn là ứng viên LCP (US-09 AC5 sửa 10.7); ghi nhận: ở 375 phần tử LCP là p.lead chứ không phải h1 (DEF-R9.1-05) |
| TC-UX09-10 | P | spec w6-real |
| TC-UX09-11 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs0land-real (qc-uiux91-rs0land-real.rerun6.log) xanh |
| TC-UX09-12 | P | curl -H 'Range: bytes=0-99': mp4 và webm đều 206, accept-ranges: bytes, content-range: bytes 0-99/… |
| TC-UX09-13 | P | RERUN6 SmkR 1409b04d (ep-r): spec qc-uiux91-rs0land-real (qc-uiux91-rs0land-real.rerun6.log) xanh |
| TC-UX09-14 | P | scripts/landing-video/record.mjs + README: chỉ đọc; không gửi chat, không Duyệt/Công bố/sửa điểm (rg Duyệt/Công bố/Gửi/type(/Enter = 0) |
| TC-UX09-15 | P | tesseract OCR 65 khung (frames/ocr.txt, logs): không có email, localhost, .local, IP, 9router, khoá; MSSV 2022xxxx là dữ liệu seed mô phỏng (D44) |
| TC-UX09-16 | P | RERUN 425396a7 (sau e0ac8dc9, DEF-R9.1-08) trên stack seed sạch smk91 (lớp mặc định 761987, sv.kha có phiên chat): bash scripts/landing-video/run.sh (bản sao ~/Documents/smks-lv, không đụng frontend/public/landing) EP_BASE_URL=https://localhost:1343 → record.mjs chạy hết 8 cảnh (s1-chat qua được, trước đây timeout), encode xong, rc=0 (logs/ux09-16-landing-run-smk91-b.log). Quy cách so với tệp đang có ở frontend/public/landing: gioi-thieu-v1.mp4 h264 1440x900, 65,8 s / 1,07 MB (hiện có 64,7 s / 1,11 MB: +1,7 % / −3 %); webm 65,8 s / 1,61 MB (hiện 64,7 s / 1,64 MB); poster gioi-thieu-v1.webp 49 KB (hiện 52 KB; shots/ux09-16/); clip gioi-thieu-clip-v1.mp4/webm/webp 11,0 s, 1440x900, 262 KB (clip màn đầu đã bỏ khỏi trang ở 10.7 P-107-13 nên repo không còn tệp clip; script vẫn sinh). README nêu đủ lệnh |
| TC-UX09-19 | P | spec w6-real |
| TC-UX09-PQ | P | /landing/../package.json, /landing/%2e%2e/package.json, /landing/..%2f..%2fnext.config.ts, /edupilot/x.mp4: đều trả khung ứng dụng (HTML), không có nội dung tệp riêng; không tới MinIO |
