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
