### DEF-R5-01 — Nút đầu trang AppShell: axe `label-content-name-mismatch` (chỉ khi bật thẻ wcag21a)
- Mức: Thấp (ghi nhận, KHÔNG phải hồi quy của sprint sau; ca TC liên quan được chấm lại)
- Sprint gốc: 5 · TC gốc: TC-PE03-52 (axe wcag2a/2aa/21aa/22aa 0 serious) · Hồi quy do sprint sau?: KHÔNG — `aria-label` của nút tìm nhanh / chuông / hồ sơ có từ `f7a15c7d` (US-PROTO-00, 2026-10-02) và `c8c3b70b` (2026-10-02), TRƯỚC sprint 5 (10-03…10-10); `axe-allow.json` không có mục `label-content-name-mismatch`. Quy tắc này thuộc thẻ `wcag21a`, không nằm trong bộ thẻ của TC/dev (`a11y.spec.ts` TAGS = wcag2a, wcag2aa, wcag21aa, wcag22aa) nên lượt đầu của QC (thêm wcag21a) báo thừa.
- Môi trường: ep-r, be0c0d96, GV r5t 1440; script `evidence-5/ui/ui_pe.cjs`
- Kỳ vọng: 0 serious với bộ thẻ TC. · Thực tế: với bộ thẻ TC → 0 (`/questions`); thêm wcag21a → `button.search` aria-label "Tìm nhanh hoặc đi đến" ≠ chữ "…" nhìn thấy, chuông "Thông báo, N chưa đọc" ≠ "9+", hồ sơ "Tài khoản: Tên" ≠ "Tên" (WCAG 2.5.3). Nghi mã: `frontend/src/shared/shell/AppShell.tsx:245-266`.
- Chạy lại lần 2: ổn định. Khuyến nghị (PM quyết): đưa chữ nhìn thấy vào đầu aria-label.

### DEF-R5-02 — Vùng cuộn ở màn làm bài code không focus được bằng bàn phím
- Mức: Thấp
- Sprint gốc: 5 · TC gốc: TC-PE06-08 (axe 1024/1440 0 `serious`) · Hồi quy do sprint sau?: KHÔNG RÕ (gốc PE-06 chỉ chạy axe qua spec dev có `axe-allow`)
- Môi trường: ep-r, be0c0d96, SV r5s12, 1440, màn `/exams/<id>/take` câu code sau khi `Chạy thử` có kết quả
- Bước tái hiện: 1. SV mở bài MIXED, chuyển tới câu code, bấm `Chạy thử`. 2. axe → `scrollable-region-focusable` ×1 (vùng cuộn của khối kết quả / soạn mã không có `tabindex=0`).
- Kỳ vọng: 0 serious · Thực tế: 1 vùng cuộn không tới được bằng Tab. · Bằng chứng: `evidence-5/logs` (ui_pe, lượt SV 1440) · Nghi ngờ mã: khối kết quả chạy thử / editor ở `frontend/src/features/exam` (chưa định vị dòng)
- Chạy lại lần 2: ổn định

### DEF-R5-03 — Ngân hàng câu hỏi: nháp tự lưu nhưng KHÔNG khôi phục khi mở lại (mất chữ gõ dở)
- Mức: Trung bình (S2: vi phạm luật "không bao giờ mất chữ", AC13 US-PE-03)
- Sprint gốc: 5 · TC gốc: TC-PE03-49 · Hồi quy do sprint sau?: KHÔNG — lỗi từ b8e30b42 (2026-10-09, US-PE-03 frontend)
- Môi trường: ep-r, GV r5t, Chrome 1440, /questions lớp R5C1 · script `evidence-5/ui/ui_q2.cjs` (ONLY=gv)
- Bước tái hiện: 1. Mở câu AI "Chờ duyệt" → Chỉnh sửa. 2. Gõ thêm chữ vào Đề bài, đợi 3 s (localStorage `ep:draft:<uid>:question:<id>` có ghi). 3. Đóng tab đột ngột, mở tab mới, vào lại câu → Chỉnh sửa.
- Kỳ vọng: banner "Đã khôi phục phần bạn đang sửa dở" + chữ còn. · Thực tế: không banner, ô về bản máy chủ; chữ gõ dở mất (khoá nháp vẫn nằm trong localStorage).
- Nghi ngờ mã: `QuestionForm.tsx` tính `restored = useMemo(..., [])` từ `saved.value` ngay lần render đầu, nhưng `useAutosaveDraft` chỉ nạp giá trị trong `useEffect` (value = "" lúc đầu) ⇒ `restored` luôn null.
- Chạy lại lần 2: ổn định.

### DEF-R5-04 — Ngân hàng câu hỏi: Enter trên hàng mở Drawer rồi Drawer đóng ngay (bàn phím không dùng được)
- Mức: Trung bình (S2: luồng chính chỉ bàn phím, AC13)
- Sprint gốc: 5 · TC gốc: TC-PE03-51 · Hồi quy do sprint sau?: KHÔNG RÕ (DataTable + Dialog từ sprint 2)
- Môi trường: ep-r, GV r5t, Chrome 1440 (`ui_q2.cjs` ONLY=gv2)
- Bước tái hiện: Tab tới hàng (`tr[tabindex=0]`) → Enter. Drawer mở rồi đóng trong cùng một lần nhấn (MutationObserver: dialog[open] 1 → 0; focus về body). Click chuột mở bình thường.
- Nghi ngờ: `onKeyDown` của hàng mở Drawer và chuyển focus tới nút "Đóng"; sự kiện keypress của chính phím Enter đó bấm luôn nút "Đóng". [INFERENCE: Chrome thật cũng vậy; đã tái hiện bằng Playwright keyboard.press.]
- Các phần khác của TC đạt: 5 mốc bề rộng 1440/1024/720/719/390 không tràn ngang, không cắt chữ, ≥720 bảng, <720 danh sách.

### DEF-R5-05 — `go test ./internal/contract/...` không tag testroutes đỏ (lệnh trong TC tổng và `scripts/gate-pe.sh`) — ĐÃ SỬA (commit 562f5a45, P-11-50/DEF-R2-04): chạy lại ở HEAD qc2: không tag ok 39 s, có tag ok
- Mức: Thấp (script/tài liệu; CI chạy với `-tags testroutes` nên xanh)
- TC gốc: TC-PE01-39, PE02-57 (và các TC tổng PE03-55…PE08-68 cùng lệnh) · Hồi quy do sprint sau?: CÓ (contract_test cần bản dựng testroutes sau P3/P10)
- Thực tế: 4 test đỏ khi thiếu tag; với `-tags testroutes` ok (31 s) và `-race` ok (80 s). Khuyến nghị: thêm `-tags testroutes` vào gate-pe.sh + TC.

## Ghi nhận không tính lỗi
- Go `TestRunningNotCancelled` đỏ 1 lần khi máy tải nặng (lease 30 s hết hạn), PASS 2/2 khi chạy riêng — chập chờn do tải.
- TC-PE02-37: fork bomb A8 ra `WA` (FORKED=0) thay vì `TLE` như gốc; bất biến `procPeak==1`/0 tiến trình mồ côi thoả.
- Today bỏ provider `staff`/`assignment` khi stack quá tải (timeout) — degrade có chủ đích.

## Lượt SmkM (qc2, 2026-11-16, HEAD sprint/11 + merge)
Bằng chứng: `evidence-5/results-q2.jsonl` (bản ghi cuối thắng), `evidence-5/ui/ui_m.cjs`, `ui_m2.cjs`, `scripts/s20*.py`, ảnh `evidence-5/ui/shots/m-*.png`.

### DEF-R5-06 — Lần nộp code: nhãn «Đã nhận, đang xếp hàng» (QUEUED) không bao giờ hiện
- TC: TC-PE06-27 · Mức: Thấp · Sprint gốc: 5 (US-PE-06 AC8, commit 2b74e9f7) · Hồi quy do sprint sau: KHÔNG (chưa từng có)
- Tái hiện: SV nộp lời giải khi hàng chấm đầy / chặn. `CodeResults.tsx:91 submissionStatus()` trả «Đang chấm» cho cả QUEUED và RUNNING. Quan sát thật: «Đang chấm» (0,2 s) → «Biên dịch được · 2/2 test mẫu đúng».
- Kỳ vọng (US AC8): QUEUED «Đã nhận, đang xếp hàng» → RUNNING «Đang chấm» → DONE.

### DEF-R5-07 — Tín hiệu liêm chính cuối cùng bị mất khi đóng tab (không gửi lúc pagehide)
- TC: TC-PE07-18 · Mức: Thấp (tín hiệu tham khảo, không đổi điểm) · Sprint gốc: 5 (237f40af US-PE-07) · Sau đó: KHÔNG sửa (4831150a chỉ thêm tuỳ chọn keepalive vào apiClient)
- Tái hiện: SV dán nội dung ở ô mã (PASTE vào bộ đệm), rồi rời trang (`location=about:blank`) trước nhịp 15 s: `exam_events` 5 → 5 (lô cuối không tới). `useIntegrityEvents.ts` flush() gọi `apiClient.post` không `keepalive`/`sendBeacon`.
- Kỳ vọng (US AC3): gửi gộp ~15 s VÀ khi `pagehide` (sendBeacon / fetch keepalive). Phần đạt: lô 15 s gửi đúng (PASTE chỉ `chars`+`item_id`, không lộ nội dung).

### DEF-R5-08 — Màn kết quả SV thiếu dòng tóm tắt và từng câu không gập mặc định
- TC: TC-PE08-31 · Mức: Thấp · Sprint gốc: 5 (ff926faf US-PE-08) · Hồi quy do sprint sau: KHÔNG
- `StudentResult.tsx` chỉ có «9,00 / 10,00» + danh sách câu mở sẵn; không có «Bạn đúng 8/10 câu trắc nghiệm · bài lập trình: p/t test ẩn đạt»; câu không gập (chỉ khối «Mã của bạn» gập). SRS dòng 1098 và US AC7 vẫn ghi dòng này (chưa vá theo luật chữ UI 09/10/2026 nếu là chủ ý — PM quyết: vá spec hoặc thêm dòng).

### DEF-R5-09 — Giao diện Staff `/exams/[id]/results` lệch US AC10 ở 3 điểm
- TC: TC-PE08-38 · Mức: Thấp · Sprint gốc: 5
- (1) Drawer không có verdict từng test ẩn (chỉ «Test ẩn: đạt 2 trên 2»); SRS 4.x `breakdown` chỉ lưu `{passed,total}` nên US AC10 và SRS mâu thuẫn — BA/PM quyết. (2) Bài chưa ai làm: hiện cả lớp «Chưa bắt đầu», không có «Chưa có sinh viên nào bắt đầu làm bài.». (3) 390 px: AUDIT_SRC `cut`=40 (cột «Nộp lúc/Tín hiệu» ngoài khung cuộn ngang của bảng; không tràn trang).
- Phần đạt: cuộn 5.001 dòng p95 khung 16,8 ms; TA không thấy Tín hiệu / tab liêm chính.

### DEF-R5-10 — Sửa test sau đóng: bài cũ không hiện «đã chấm theo bộ test cũ»
- TC: TC-PE03-33 · Mức: Thấp · Sprint gốc: 5 (US-PE-03 AC8) · Hồi quy: KHÔNG
- GV sửa test ẩn sau khi bài PUBLISHED: 200, `tests_version` 7→8, điểm giữ nguyên tới regrade, TA 403, câu đang ở bài OPEN → 409 QUESTION_IN_USE đều đúng; nhưng không có trường/chữ «đã chấm theo bộ test cũ» ở API kết quả Staff, giao diện hay mã Go.

### DEF-R5-11 — Chấm trắc nghiệm đồng bộ không phát sự kiện SSE `exam.attempt`
- TC: TC-PE08-04 · Mức: Thấp · Sprint gốc: 5 (US-PE-08 AC2) · Hồi quy: KHÔNG RÕ
- Nộp bài MCQ-only: 200 trong 0,02 s, GRADED + `breakdown` đúng; nhưng outbox của lượt chỉ có `exam.attempt_started`, `exam.attempt_submitted`; không có `exam.attempt_graded` → chủ lượt không nhận SSE `exam.attempt` (kết nối /api/v1/events chỉ nhận `ready`).

## Ghi nhận môi trường (không tính lỗi)
- TC-PE04-40 (lượt đầu ghi F): LỖI ĐO, không phải lỗi sản phẩm. Kịch bản s20a_today.py lấy thời điểm outbox exam.closed khi dòng đó chưa có (trả 0) nên tính «đóng 1794837114.25 s» (epoch trừ 0). Đo lại từng sự kiện so với dòng outbox: lên lịch 0,62 s, mở 0,63 s, bắt đầu 1,35 s, nộp 0,58 s, đóng 0,57 s sau sự kiện (s20a_today2.py) → P (≤2 s). Một lượt đo đóng bài trễ 66 s do bộ gửi outbox bị nhóm QC khác khởi động lại — môi trường.
- Bộ gửi outbox trễ 66 s đúng một lần (13:55:58Z) khi nhóm QC khác khởi động lại tiến trình trong cửa sổ khoá; các lần khác 0,45–0,5 s.
- Worker ghi `exam.tick lỗi` mỗi 5 s từ 22:03 hôm qua cho lượt 01a1e01f… của bài «QC E7 khoá chat» (tổng điểm các mục bằng 0 — dữ liệu chèn bằng SQL của nhóm QC sprint 7; không tạo được qua API).
- `go test -race ./...` toàn kho khi máy tải 20–30: nhiều gói hết 10 phút (activity, assignment, auth, course, crm, exam, grade…) và contract đỏ do tải; chạy lại gói exam / contract không-tag / có-tag riêng thì xanh.

## Từ nhóm đo hiệu năng/tĩnh (SmkM.PerfStaticQC; chi tiết `evidence-5/defects-draft-perf.md`)
### DEF-R5-12 — `pnpm lint` ở frontend rc=1 (TC-PU06-17, TC-UI7-06)
- Mức: Trung bình (CI chạy đúng lệnh này sẽ đỏ) · Nguyên nhân: 32 lỗi + 15 cảnh báo hoàn toàn ở 10 tệp QC đã commit `frontend/e2e/qc2-*`; mã sản phẩm lint rc=0 · Sprint gốc: không phải sprint 5 (tệp QC đợt hồi quy) · Đề xuất: sửa/loại `qc2-*` khỏi lint hoặc `eslint-ignore`.
### Ghi nhận Thấp — `next build` cảnh báo `metadataBase` (trang giới thiệu thêm ở sprint sau).


## Chạy lại sau sửa của dev (qc2/SmkP, 2026-11-18, ep-r dựng lại @634d59c5 rồi @88ab83e4)
Tất cả đạt (bản ghi mới trong `evidence-5/results-q2.jsonl`, bản ghi F cũ giữ nguyên): DEF-R5-03 (TC-PE03-49) ĐÃ SỬA 46bf5626; DEF-R5-04 (TC-PE03-51) ĐÃ SỬA 4cd9c07e; DEF-R5-06 (TC-PE06-27) ĐÃ SỬA 0e52f1e5; DEF-R5-07 (TC-PE07-18) ĐÃ SỬA d7fecdc2 (đóng tab thật bằng page.close({runBeforeUnload:true}); biến thể browserContext.close() không phát pagehide — hạn chế của Playwright, không phải lỗi); DEF-R5-08 (TC-PE08-31) ĐÃ SỬA 473472ec (P-11-60); DEF-R5-09 (TC-PE08-38) ĐÃ SỬA 247bf4e4 (P-11-59); DEF-R5-10 (TC-PE03-33) ĐÃ SỬA dd160463; DEF-R5-11 (TC-PE08-04) ĐÃ SỬA 3f748ec2. Còn mở: DEF-R5-12 (lint spec QC, agent chính).

## Lượt bù qc3 (2026-11-21, ep-r + qc3s @425396a7)

### DEF-R5-13 — [ĐÃ SỬA 9754c523 (practice/extract.go sạch, chạy lại 2026-11-21 @aba738ef); cổng vẫn đỏ vì DEF-R5-18] `golangci-lint run` còn 1 vấn đề (gochecknoglobals) — cổng tổng đỏ (TC-PE04-50, PE05-60, PE06-47, PE08-68)
- Mức: Trung bình (`.github/workflows/ci.yml` chạy golangci-lint, bước này đỏ) · Sprint gốc: không phải sprint 5 (đưa vào ở sprint 11, commit e42d8b49 «DEF-R10S-12», 2026-11-19) · Hồi quy: có (@88ab83e4 báo «0 issues»)
- Tái hiện: `cd backend-go && golangci-lint run` → `internal/practice/extract.go:130:5: defaultRetryWait is a global variable (gochecknoglobals)`, `1 issues`, rc=1. Các bước còn lại của cổng đạt: `go vet ./...` rc=0, `sqlc diff` rc=0, `go test -race` exam/judge/quiz rc=0, `go test` contract (có và không `-tags testroutes`) rc=0, `pnpm lint` rc=0, `build:gate` rc=0.
- Nghi ngờ mã: `var defaultRetryWait = []time.Duration{…}` ở extract.go:130 (đổi thành hàm trả giá trị hoặc `//nolint:gochecknoglobals` có lý do). Log: `evidence-5/logs-r3/g1-static.log`.

### DEF-R5-14 — [ĐÃ SỬA d27b87b0, chạy lại 2026-11-21 @58448e0a: sv07 gcd TLE 0/8, matched 30/30] Seed PE không có lượt nộp code `TLE` (TC-PE09-07)
- Mức: Thấp · Sprint gốc: 5 (US-PE-09 AC2) · Hồi quy: KHÔNG
- US AC2 / TC: 6 bài code «AC đủ, một phần, `CE`, `TLE`, `WA`» + một cặp giống nhau. Seed sạch trên qc3s: verdict thực tế chỉ có AC (sv.gioi, sv05, sv07 dem-tu, sv.kha dem-tu), WA một phần (sv.kha gcd 6/8, sv04/sv06 dem-tu 3/5), CE (sv04 gcd), NONE (sv07 gcd); không có `TLE`. `seed/expected_exam_scores.csv` cũng không có TLE. Điểm 30/30 khớp, hai DB seed sạch độc lập cho điểm giống hệt.
- Nghi ngờ mã: `scripts/exam-seed-data.mjs` CODE_ATTEMPTS (`wordsLoop` thực ra AC, không có lời giải vòng lặp vô hạn). Cần BA/PM chọn: thêm lời giải TLE hoặc bỏ «TLE» khỏi US AC2.

### DEF-R5-15 — [SỬA CHƯA ĐỦ: d0d548e0 hết ở seedThreads, còn step19Practice:843 → DEF-R5-20; chạy lại 2026-11-21] Seed lần hai không in «Seed xong (không đổi)» (TC-PE09-12)
- Mức: Thấp · Sprint gốc: 5 (US-PE-09 AC5) nhưng lỗi nằm ở bước thêm sau · Hồi quy: không rõ
- Tái hiện (qc3s, DB đã seed): `node scripts/seed.mjs` lần 2, 3, 4 → «Seed xong trong 25 s.» (không có dòng «Seed xong (không đổi)»). Đếm dòng 85 bảng trước/sau: nghiệp vụ không đổi (users, courses, exams, question_bank, exam_attempts, code_submissions, forum_threads; Mailpit 168 = 168), chỉ tăng phiên đăng nhập / outbox / audit do chính seed đăng nhập.
- Nguyên nhân (bằng cách gắn dò vào bản sao seed.mjs): `ctx.changed = true` ở `seedThreads` (scripts/seed.mjs:1010): thread mẫu «Modern Network Security Threats» không khớp danh sách thread hiện có (11 thread thấy được, thread «· Phisher» bị ẩn ở bước 18) nên bị POST lại; Idempotency-Key trả dòng cũ nên không tạo thêm. Log: `evidence-5/logs-r3/pe09-12-*`.

### DEF-R5-16 — [N: PM 2026-11-21 — đo lại ở cửa sổ SEED cuối khi các stack khác dừng, P-11-76] Seed trên DB trống mất 652 s, trần ≤ 6 phút (TC-PE09-13)
- Mức: Thấp (cùng nhóm DEF-Q2-S4-01 / DEF-R4-08) · Sprint gốc: 5 (US-PE-09 AC5, ngân sách đổi ≤ 6 phút theo P-11-76) · Hồi quy: có thể do máy chung
- qc3s `up` → `seed`: «Seed xong trong 652 s» (log `evidence-5/logs-r3/seed3.log`, `seed-t0-clean.json`): bước 11 tài liệu docling thật 241 s, bước 20 luyện đề 204 s, bước 19 78 s, bước 10 (PE) 40 s; P2 + PE (bước 1–10) = 91 s. Máy chung tải 9–14 (không đạt «máy không chạy stack khác»; PM đo 314 s ngày 13/11). Cần đo lại trên máy trống; phần dài là các bước thêm sau sprint 5 (11, 19, 20).

### DEF-R5-17 — [ĐÃ SỬA 58448e0a, chạy lại 2026-11-21 @aba738ef: audit 115 hàng FAIL 0] AUDIT `cut` ≠ 0 ở bảng ảo hoá DataTable 390 px: TA/GV `/students`, `/gradebook` (TC-PE04-50…, TC-PU06-18)
- Mức: Thấp · Sprint gốc: không phải sprint 5 (DataTable ảo hoá, bảng điểm / sinh viên) · Hồi quy: không rõ
- Tái hiện: đăng nhập `teacher@` (hoặc `ta@`), 390 px, `/students` và `/gradebook`, dán AUDIT_SRC (docs/sprints/1.5/spec/US.md): `ox=0` nhưng `cut` = «Hoạt động học», «Rủi ro», «0 phút · 0 câu hỏi» (`/students`) và «Kiểm tra hằng tuần», «Thi cuối kỳ», «Điểm cộng» (`/gradebook`). Các phần tử nằm trong `DIV.DataTable…vscroll` (`overflow-x: auto`, `data-part=virtual-scroll`) đang cuộn ngang đúng ý, nhưng vùng này không có `data-scroll-x` nên phép đo tính là chữ bị cắt. 4/115 hàng FAIL (SV, admin: 0).
- Ghi chú kịch bản: bản `smkP_audit.cjs` cũ báo thêm 6 hàng `/settings/llm`, `/settings/integrations` do phần tử `<option>` (không có hộp vẽ); `evidence-5/ui-r3/audit.cjs` đã loại. Nghi ngờ mã: `shared/ui/DataTable.tsx` nhánh ảo hoá không gắn `useScrollRow` / `data-scroll-x` (DataTable.tsx:137). Log `evidence-5/logs-r3/audit.out`, `cutdbg.json`.

**Chạy lại DEF-R5-12 (qc3, 2026-11-21, @425396a7):** `pnpm -C frontend lint` rc=0 (0 lỗi, 1 cảnh báo ở `frontend/dbg91b.mjs`), 13 tệp `qc2-*` đã chuyển sang `docs/regression/specs-qc2` → ĐÃ SỬA; TC-PU06-17 = P.

**Ghi nhận chưa thành DEF (qc3, 2026-11-21):** TC-PE09-20 — `exam.spec.ts` (53 ca, 120 pass ở gate) chưa có ca đọc mọi phản hồi mạng của trình trình duyệt với canary (AC7 ý 11; chỉ có Go `TestNoAnswerLeak`) và chat-gate nằm ở `private-chat.spec.ts` (ý 9); chưa chấm, chờ lượt chạy `@real` + đối chiếu 12 ý.

### DEF-R5-18 — `golangci-lint run` đỏ lại: cmd/regress/main.go:213 `flipRight` biến toàn cục (cổng tổng TC-PE04-50, PE05-60, PE06-47, PE08-68)
- Mức: Trung bình (CI chạy golangci-lint) · Sprint gốc: không phải sprint 5 (commit 658e6719 «DEF-R106S-12/13/14», 2026-11-21) · Hồi quy: có
- Tái hiện (@aba738ef): `cd backend-go && golangci-lint run` → `cmd/regress/main.go:213:5: flipRight is a global variable (gochecknoglobals)` (`var flipRight, flipWrong = decimal.RequireFromString("0.25"), …("0.5")`), `1 issues`, rc=1. Còn lại đạt: go vet rc=0, sqlc diff rc=0, pnpm lint rc=0, audit 115 hàng FAIL 0. Log `evidence-5/logs-r3/g1r2-static-aba738ef.log`.

### DEF-R5-19 — `/chat` Sinh viên: CLS thật 0,16 > 0,1 (cột danh sách phiên biến mất làm cả cột chi tiết dịch) (TC-PU06-11, US-PU-06 AC1)
- Mức: Thấp · Sprint gốc: 5 (US-PU-06 AC1: CLS ≤ 0,1) · Hồi quy: có thể (đo tay cũ 0,0869 ở sprint 5)
- Tái hiện: ep-r @aba738ef, đăng nhập `sv.gioi@edupilot.local`, 1440 × 900, vào `/chat`, đo `PerformanceObserver('layout-shift')` (bỏ hadRecentInput): 0,1618 / 0,1560 / 0,1618. Một dịch 0,156 lúc ~370 ms: `DIV.Layout-module__…ldDetail` («Hỏi bất cứ điều gì về lớp») x 621 → 241, rộng 794 → 1174; kéo `Composer`, `UL…suggest` (3 LI). 14 route khác của SV/GV/Admin ≤ 0,0002.
- Nghi ngờ mã: bố cục hai cột của Chat (`Layout…ldDetail`, features/chat RealChat/ChatScreen): chỗ cột danh sách phiên chỉ được bỏ sau khi dữ liệu phiên về. Script `evidence-5/ui-r3/ui5d.cjs`.

### DEF-R5-20 — Seed lần hai vẫn không in «Seed xong (không đổi)»: `ctx.changed = true` vô điều kiện ở step19Practice (TC-PE09-12)
- Mức: Thấp · Sprint gốc: 5 (US-PE-09 AC5), lỗi ở bước Luyện đề thêm sau · Hồi quy: không rõ
- Tái hiện (qc3s, DB seed sạch @58448e0a): `node scripts/seed.mjs` lần hai → «Seed xong trong 23 s.»; gắn dò: `CHANGED-AT at step19Practice (scripts/seed.mjs:843)`: dòng `ctx.changed = true;` ngay sau khối «lượt đang dở của sv.nguyco» chạy cả khi không tạo gì (`attempts` = 0). Dữ liệu không đổi (đếm 85 bảng). Đề xuất: `if (attempts > 0 || …) ctx.changed = true`.
- DEF-R5-16 (652 s → 618 s ở lần đo sau, trần ≤ 6 phút theo P-11-76): vẫn vượt trên máy chung.

### DEF-R5-21 — Kết quả sinh viên `/exams/[id]/take`: bảng «Kết quả test mẫu» làm trang tràn ngang 287 px ở 375, 34 px ở 720 khi mở câu code (TC-PE09-25, US-PE-09 AC9)
- Mức: Thấp (nhìn thấy: bảng bị cắt trong thẻ, vẫn có «Vuốt ngang để xem thêm»; số đo trang vẫn dương) · Sprint gốc: 5 · Hồi quy: không rõ
- Tái hiện: ep-r @aba738ef, bài có câu code đã công bố, đăng nhập SV đã làm bài, 375 px (hoặc 720), mở các câu (summary) → `document.documentElement.scrollWidth − innerWidth` = 287 (375) / 34 (720); gập hết: 0. Phần tử vượt: `DIV.DataTable…root`, `…scrollHint`, `…scroll`, `TABLE` (right = 662 > 375). Spec `student result: 375 px …không tràn ngang` đạt vì dữ liệu giả không có bảng mẫu. Ảnh `ui-r3/shots/f19b-21-ketqua-375-mo.png`, `pe09-25-sv-*`; log `logs-r3/ui5e.log`, `ui5f` (`pe09-25-rows.json`).
- Nghi ngờ mã: `features/exam/results/ResultItems.tsx` (DataTable «Kết quả test mẫu») không bị ép rộng theo khung cha (thiếu `min-width: 0` / khung `data-scroll-x`).

### DEF-R5-22 — exam.spec.ts chưa phủ ý (9) chat-gate và (11) rò đáp án quét mạng của AC7 (TC-PE09-20)
- Mức: Thấp · Sprint gốc: 5 (US-PE-09 AC7) · Hồi quy: không
- `exam.spec.ts` có 46 ca (đạt hết) nhưng không có ca «trong giờ làm bài chat bị từ chối, sau nộp cho phép» (chỉ `private-chat.spec.ts` mock `EXAM_IN_PROGRESS`) và không ca nào đọc mọi phản hồi mạng của trình duyệt tìm canary (chỉ Go `TestNoAnswerLeak`). QC quét tay thay thế (TC-PE09-21): 0 rò trước công bố.

