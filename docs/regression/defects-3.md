### DEF-R3-01 — Bảng «Mức dùng theo tác vụ» hiện chữ thô «EMBEDDING»
- Mức: Thấp
- Sprint gốc: 3 · TC gốc: TC-P105-44 (AC11 không từ kỹ thuật) · Hồi quy do sprint sau?: KHÔNG RÕ (nhãn tác vụ nằm từ US-P1-05; chữ thô chỉ lộ khi `llm_audit` có dòng EMBEDDING; sprint sau thêm lời gọi nhúng thật nên mới thấy)
- Môi trường: ep-r, be0c0d96, ADMIN, 1440 và 390
- Bước tái hiện: 1. đăng nhập admin@edupilot.local; 2. mở `/settings/llm`; 3. cuộn tới «Mức dùng và ngân sách» → bảng «Mức dùng theo tác vụ». (`GET /admin/llm/usage?group=task` có key `EMBEDDING`.)
- Kỳ vọng: mọi tác vụ có nhãn tiếng Việt, không từ kỹ thuật (regex `/\b(RAG|PII|trace|fallback|embedding|prompt)\b/i` = 0 khớp ở văn bản hiển thị) · Thực tế: hàng «EMBEDDING | 173 lượt | …»
- Bằng chứng: `evidence-3/shots/settings-llm-1440.png`; DOM: `TD:EMBEDDING @usage-section` · Nghi ngờ mã: `frontend/src/features/settings/llm/labels.ts:13` (`taskLabel` chỉ biết 6 tác vụ + DECIDE; `features/observability/api.ts:44` đã có nhãn «Đánh chỉ mục»)
- Chạy lại lần 2: ổn định

### DEF-R3-02 — `docs/design/DESIGN_TOKENS.css` lệch `frontend/src/shared/styles/tokens.css` ở 2 token
- Mức: Thấp
- Sprint gốc: 3 · TC gốc: TC-PU01-01 (kit ⊆ tokens.css) · Hồi quy do sprint sau?: CÓ (góp ý #25 hạ `--ep-ink-3`; US-04 uiux-91 đặt `--ep-slow` 200 ms; kit không cập nhật)
- Môi trường: ep-r, be0c0d96, so tệp
- Bước tái hiện: so `--ep-ink-3` và `--ep-slow` ở hai tệp: kit `oklch(58% …)` / `250ms`; code `oklch(53% 0.014 25)` / `200ms`.
- Kỳ vọng: kit và tokens.css đồng bộ (TC cũ: mọi khai báo kit có mặt y nguyên) · Thực tế: lệch 2 giá trị (TC-PU01-01/03 đã xếp «LỖI THỜI» vì kit đổi hợp lệ 51→62 dòng ở P-107-8/10, P-108-8)
- Bằng chứng: `evidence-3/scripts/` (lệnh `comm` trong pu01.py) · Nghi ngờ mã: `docs/design/DESIGN_TOKENS.css`
- Chạy lại lần 2: ổn định

### DEF-R3-03 — `pnpm lint` có 5 cảnh báo (nền sprint 3 = 0)
- Mức: Thấp
- Sprint gốc: 3 · TC gốc: TC-PU01-05, TC-GATEPU-01 · Hồi quy do sprint sau?: CÓ (10.x)
- Môi trường: bản sao `frontend/` ở be0c0d96, `pnpm lint` rc=0
- Bước tái hiện: `cd frontend && pnpm lint` → `✖ 5 problems (0 errors, 5 warnings)`: `src/app/dev/ui/cells.tsx:39` (`text` không dùng), `features/grading/QueueTab.tsx:115,123` (react-hooks/exhaustive-deps `setFlagOnly`, `setStatus`), `features/landing/Landing.tsx:48` (`<img>`), `features/settings/llm/LlmSettings.tsx:13` (`s` không dùng)
- Kỳ vọng: 0 lỗi / 0 cảnh báo mới so nền · Thực tế: 5 cảnh báo
- Bằng chứng: `evidence-3/lint.out` · Nghi ngờ mã: các dòng trên
- Chạy lại lần 2: ổn định

### DEF-R3-04 — `DecryptKey(` được gọi ngoài `internal/llm/` và `internal/llmconfig/`
- Mức: Thấp
- Sprint gốc: 3 · TC gốc: TC-P101-24 (AC8) · Hồi quy do sprint sau?: KHÔNG (có sẵn từ US-P1-04, commit 25c0bf24 — sau lượt QC P1-01 vốn báo grep = 0)
- Môi trường: grep mã
- Bước tái hiện: `grep -rn 'DecryptKey(' backend-go --include=*.go | grep -v _test.go | grep -v internal/llm/ | grep -v internal/llmconfig/` → `internal/httpapi/llmhttp/admin.go:242`, `verify.go:302` (kiểm kết nối khi sửa nhà không đổi khoá)
- Kỳ vọng: 0 dòng (khoá rõ chỉ ở llm/llmconfig) · Thực tế: 2 dòng ở lớp HTTP; không có rò khoá (TC-P104-16/17 đều 0)
- Bằng chứng: kết quả grep · Nghi ngờ mã: như trên
- Chạy lại lần 2: ổn định

### DEF-R3-05 — Làn BATCH bị gọi lại/chờ bất thường trên ep-r (TC-P102-14, TC-P103-19, TC-P103-20)
- Mức: Trung bình (cần dev xác nhận; có thể do tải chung của nhóm khác trên làn BATCH)
- Sprint gốc: 3 · TC gốc: TC-P102-14 (AC6), TC-P103-19/20 (AC6 cầu dao) · Hồi quy do sprint sau?: KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, ADMIN, `/_test/llm/chat`, nhà stub OpenAI cục bộ `evidence-3/scripts/qstub.py` (model `m-503` trả 503), `LLM_MAX_CONCURRENCY=10`
- Bước tái hiện: `python3 evidence-3/scripts/p10x2.py` (cần khoá global): (1) đặt CHAT → nhà S3/m-503, GRADING → nhà S4/m-503; gọi CHAT ×1 → stub nhận 2 (đúng 1+1); gọi GRADING ×1 → stub nhận 15 lần (kỳ vọng 3: 1 + 2 thử lại); (2) sau ≥5 lỗi, circuit=open nhưng lời gọi GRADING kế tiếp trả 503 sau 42 242 ms (kỳ vọng chặn ngay <500 ms, 0 request tới nhà); (3) sau 30 s bán mở: 3 lời gọi đồng thời → stub nhận 6 (kỳ vọng ≤ 1 thăm dò × số lần thử).
- Kỳ vọng: BATCH thử lại ≤ 2 lần; cầu dao mở chặn ngay; bán mở 1 thăm dò · Thực tế: như trên (INTERACTIVE đúng)
- Bằng chứng: `evidence-3/p10x.out`, `/tmp/qc2-b/results-3-p10y*.json` · Nghi ngờ mã: `backend-go/internal/llm/scheduler/` (hàng đợi BATCH dùng chung giữa tiến trình, `LLM_BATCH_SHARE`), `internal/llm/cascade.go`; hoặc làn BATCH đang bận bởi chấm bài của nhóm khác (nhiễu môi trường) — chạy lại trên stack yên tĩnh để loại trừ
- Chạy lại lần 2: trượt ổn định (2 lần chạy cùng kết quả; lần chạy thứ 3 đang chờ khoá)

### Cập nhật lượt N=0 (2026-11-16, qc2/SmkN)
- DEF-R3-03 (lint 5 cảnh báo): ĐÃ HẾT trên mã hiện tại — eslint mã sản phẩm 0 vấn đề (bản sao /tmp/smk3-fe, loại e2e/qc2-*). Lưu ý: `pnpm lint` toàn cây đang báo 47 vấn đề (32 lỗi) nằm hoàn toàn trong spec QC đã commit `frontend/e2e/qc2-r4|r8|r10|r91` (đã báo PM).
- DEF-R3-05 (BATCH thử lại/cầu dao): lượt chạy lại 2026-11-16 (xem smoke-3: TC-P102-14, P103-19/20) = ĐẠT; trên stack cô lập cũng đạt (TC-P102-18, P103-22).
- DEF-R3-01 (EMBEDDING thô ở bảng mức dùng): CHƯA kiểm lại ở UI (TC-P105-44 giữ F); DEF-R3-02, DEF-R3-04 không đổi.

### DEF-R3-SMK-01 — RÚT LẠI (lỗi đo của QC, KHÔNG phải lỗi sản phẩm): «bucket RPM cho qua 50–65 lời gọi / 10 s»
- Trạng thái: ĐÓNG — TC-P103-08 chạy lại = ĐẠT (qc2/SmkO, 2026-11-16)
- Nguyên nhân gốc: `stubreset()` trong `evidence-3/scripts/smk3iso.py` chỉ xoá bộ đếm của stub cổng 9741/9742, còn ca TC-P103-08 dùng stub cổng 9751. Bộ đếm 9751 vì vậy cộng dồn cả các lời gọi chuẩn bị (khởi tạo nhà, vòng nạp cấu hình, lượt trước) rồi script in con số đó như «số lời gọi trong 10 s»: «65 trong giây đầu rồi 1/giây» = ~59 lời gọi cộng dồn + burst 6; «50 trong 10 s» cùng nguyên nhân. Sau mốc đó đúng 1 lời gọi/giây — tức bucket hoạt động đúng 60 rpm.
- Đọc mã (chỉ đọc) `backend-go/internal/llm/scheduler/redis.go` `bucketLua` + `scheduler.go` `tryGrant`: bucket khởi tạo `tokens = cap`, `cap = burst(rpm) = max(1, rpm/10)` = 6; mỗi lần gọi nạp thêm `(now-ts)*rpm/60000` rồi kẹp `min(cap, …)`; mỗi lời gọi trừ 1; thiếu token thì nhả chỗ và ngủ theo gợi ý (≤1 s). Đồng hồ lấy bằng `redis.call('TIME')`. Hàm khớp SRS 4.3.
- Đo lại sạch (`evidence-3/scripts/smk3-p10308.py`, `smk-p10308.txt`): nhà rpm_limit=60, 20 luồng qua 2 gateway, xoá đúng bộ đếm 9751: lượt 1 stub tích luỹ mỗi giây [6,7,8,9,10,11,12,13,14,15] = 15 tại t=10 s; lượt 2 (nghỉ 65 s) cũng 15; trần TC 16. Người gọi dư thừa chờ rồi nhận OVERLOADED sau 10 s (không RATE_LIMIT). Thêm TC-P103-09: nhà null/null trên gateway không đặt LLM_DEFAULT_* → cũng 15/10 s, TPM bucket cap 10.000 (= 10 % × 100.000). `TestTwoProcessesRPM` đạt.
- Hồi quy do sprint sau?: KHÔNG. `scheduler/redis.go` chỉ có 2 commit, cả hai ở sprint 3 (c9a1a04e US-P1-03 và 27d44ae8 sửa BUG-P103-1/2, 2026-10-03); `bucketLua`/`burst()` không đổi từ đó (git log, chỉ đọc). Sprint 10.5 chỉ đụng `scheduler.go` (thêm L1 Decide, 425d2a74).
- Bài học QC: dùng bộ đếm từ chính cổng stub của ca, hoặc đếm `llm_audit` trong cửa sổ; `smk3-p10308b.py` cũ giữ làm bằng chứng của lỗi đo.

## Lượt qc2/SmkP (2026-11-18, ep-r @88ab83e4)
- DEF-R3-01 (TC-P105-44): ĐÃ SỬA ffb46656 — chạy lại: Admin 1440/390 + Teacher 1440 ở /settings/llm (mở nâng cao + hộp thoại): 0 khớp `embedding|RAG|PII|trace|fallback|prompt`; hàng nay là «Đánh chỉ mục tài liệu» → TC-P105-44 = P (`evidence-3/ui/smkP_p10544.cjs`).
### DEF-R3-SMK-02 — Nhánh `ci/ui-drift` chưa được xoá (TC-PU05-35)
- Mức: Thấp · Sprint gốc: 3 (handoff PU-05: «dev xoá được») · Hồi quy: không
- Bước: `git ls-remote --heads origin ci/ui-drift` → `5e32bb50…` (EduPilot-v2); `git ls-remote --heads old-private ci/ui-drift` → `38073d08…` (TA_Agent_v2). Kỳ vọng: không in gì.
- Ghi chú: 3 run đỏ của nhánh (37139595531 / 37140069267 / 37140802309 ở repo TA_Agent_v2) vẫn đọc được bằng `gh run view -R kagamikuro1024/TA_Agent_v2`; các ca TC-PU05-29/31/32/33/34/28, GATEP1-11, GATEPU-09 đã đạt bằng chúng (CI sprint 3 nằm ở repo old-private TA_Agent_v2, không phải origin).

### DEF-R3-SMK-03 — [ĐÓNG @07a6b69c, TC PASS 2026-11-19] Sinh viên thấy «Chi tiết kỹ thuật … trace_id» ở màn lỗi /me và /threads (TC-PU05-40, TC-GATEPU-15)
- Mức: Trung bình (vi phạm luật «Sinh viên không bao giờ thấy từ kỹ thuật», chỉ lộ mã lỗi + trace_id, không dữ liệu cá nhân)
- Sprint gốc: 5/6 (US-P5-06 `/me` thật d6f61728; `RealThreads`) — hợp đồng từ sprint 3 (US-PU-03: `ApiErrorNotice` ghi «`trace_id` CHỈ khi `showTechnical` (TA / GV / Admin) — Sinh viên không bao giờ thấy»). Hồi quy do sprint sau? KHÔNG (từ khi có màn).
- Tái hiện (stack ep-r thật, sv.kha): mở /me hoặc /threads, chặn mọi GET /api/v1/* trả 500 {code:INTERNAL, trace_id}, tải lại. /me hiện «Không tải được kết quả của bạn. Dữ liệu của bạn không bị mất. Chi tiết kỹ thuật Thử lại»; mở «Chi tiết kỹ thuật» hiện «NOT_FOUND · trace_id 000…» (gate build, student). /threads cũng có phần «Chi tiết kỹ thuật».
- Kỳ vọng: STUDENT không có «Chi tiết kỹ thuật» (showTechnical chỉ cho TA/GV/Admin). Thực tế: `frontend/src/features/me/RealMe.tsx:51` và `features/threads/RealThreads.tsx:88` truyền `showTechnical` vô điều kiện (cả hai là màn của sinh viên). 5/7 route SV khác sạch (/, /chat, /practice, /library, /calendar).
- Bằng chứng: frontend/e2e/qc2-r8/real-tech.spec.ts (ep-r thật), frontend/e2e/qc2-r3/stu-tech.spec.ts (gate), evidence-3/gate-self/.


## Ca lỗi thời đã XOÁ khỏi smoke (lệnh chủ dự án; mỗi mã một dòng + nguồn)
- TC-GATEP1-10: smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.m
- TC-GATEPU-10: smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.m
- TC-P105-50: smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.m
- TC-PU01-01: LỖI THỜI: docs/design/DESIGN_TOKENS.css đổi hợp lệ sau sprint 3 (P-107-8/10 bộ màu ĐHBK, màn chiếu P-108-8); còn 2 token lệch giữa kit và tokens.css: --ep-ink-3 (kit 58 % / code 53 %, góp ý #25) và --
- TC-PU01-03: LỖI THỜI: DESIGN_TOKENS.css nay 62 dòng khai báo (sửa hợp lệ ở 51eab447 / ac670244 / 8999086f)
- TC-PU01-32: smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.m
- TC-PU01-33: smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.m
- TC-PU02-39: smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.m
- TC-PU04-59: smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.m
- TC-PU05-20: lighthouserc.json nay 9 URL × 5 lượt, LCP đo ở lượt devtools riêng (lighthouserc.devtools.json) — đổi hợp lệ ở US-PU-06 #14 (commit 77b4833e) và US-P5-06; không còn «7 URL × 3 lượt, LCP trong cùng ass
- TC-PU05-38: smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.m


## Ca lỗi thời đã XOÁ khỏi smoke (lệnh chủ dự án; mỗi mã một dòng + nguồn)
- TC-P104-40: golden PG không bị sửa, redocly lint rc=0 (đã chạy P); riêng «operationId = 18 / openapi.test = 18» LỖI THỜI: số thao tác đổi hợp lệ ở mọi sprint sau (a7a887f5 sinh contract; proposals sprint 8 #9/#12
- TC-PU01-21: LỖI THỜI: layout.tsx nay 3 độ đậm 400/600/700 (commit 1d6e611f «thực hiện góp ý #26 (phông 400/600/700, LCP warn tạm)»); tokens.spec.ts (QC cũ) kiểm 3 độ đậm — spec đổi hợp lệ
