# Lỗi hồi quy tổng hợp — qc2, mã sprint/11-p10-eval, 2026-11-15

Nguồn: `defects-<sprint>.md` (chi tiết từng lỗi) và `smoke-<sprint>.md` (từng ca). Mỗi lỗi ghi: mức, sprint gốc, có phải hồi quy do sprint sau không.

## Ca trượt (ca · sprint · triệu chứng)

| Sprint | Ca | Triệu chứng |
|---|---|---|
| 1.5 | TC-00-36 | DEF-R1.5-02: TA tại «/» tràn ngang +9 px (390) / +24 px (375) — nút hồ sơ ở topbar |
| 1.5 | TC-00-57 | DEF-R1.5-01: đã đăng nhập mở đường dẫn không có thật → /login?next=…, không có màn «Không tìm thấy trang» |
| 1.5 | TC-00-59 | DEF-R1.5-01: không thấy màn 404 để bấm «Về Hôm nay» |
| 1 | P0-01/TC-07 | DEF-R1-01: DEMO_SCRIPT trích attendance-checkin/-display/checkin-code/audit/insights.spec.ts nhưng FLOWS.md không có mục tương ứng |
| 1 | P0-02/TC-21 | DEF-R2-01: `redocly lint` openapi.yaml 1 lỗi (analytics/exams/compare: parameters null) |
| 10.5 | TC-D07-02 | bảng «Quyết định nhanh» nằm TRÊN bảng «7 ngày qua» (AdminView.tsx:58 trước :65); US-DM-07 AC1: dưới bảng 7 ngày · **DEF-R10.5-01** |
| 10.6 | TC-G03-24 | TA/SV/ADMIN 403, không token 401, cờ tắt 409 đúng; nhưng reflect qua đường lớp 2 (cờ lớp 2 tắt) → 409 GUIDANCE_DISABLED trước 404 (ghi nhận 1 c |
| 10.6 | TC-G05-24 | SV regrade 403, TA 202; qua đường lớp 2 → 409 GUIDANCE_DISABLED (cờ lớp 2 tắt) thay vì 404 — ghi nhận 1 của summary 10.6, có sẵn · **DEF-R10.6-01 |
| 10 | TC-12-14 | DEF-R10-01: go test ./internal/contract/... không xanh (7 test đỏ; 500 INTERNAL ở POST …/posts/{id}/report, /auth/verify-email, thiếu quan sát); ngoài phạm  |
| 2 | TC-PG02-60 | CẦN TRIAGE: dòng outbox job.enqueue cùng xmin với dòng jobs (cùng transaction): thực tế [0] phải ≥ 1 |
| 2 | TC-PG06-01 | DEF-R2-01: redocly lint openapi.yaml 1 lỗi (analytics/exams/compare: parameters null) |
| 2 | TC-PG06-10 | DEF-R2-02: `go test ./internal/contract` rc 1 (TestStatusParity trượt 2/2 lần) |
| 2 | TC-PG06-11 | DEF-R2-02: gồm TestStatusParity |
| 3 | TC-GATEPU-01 | lint 5 cảnh báo (TC-PU01-05); pbuild/gbuild rc=0, 0 "Failed to load font" (DEF-R3-03) |
| 3 | TC-P101-24 | DecryptKey( ngoài llm/llmconfig: 2 chỗ (httpapi/llmhttp/admin.go:242, verify.go:302) → DEF-R3-04 (DEF-R3-04) |
| 3 | TC-P102-14 | lỗi 503: INTERACTIVE gọi nhà 2 lần (1+1 thử lại), BATCH 15 lần (kỳ vọng 3) (DEF-R3-05) |
| 3 | TC-P103-19 | sau chuỗi lỗi: circuit=open; lời gọi kế tiếp chặn ngay (503 LLM_UNAVAILABLE, 42242 ms, 0 request tới stub) (DEF-R3-05) |
| 3 | TC-P103-20 | bán mở sau 30 s: 3 lời gọi đồng thời → stub nhận 6 (≤ 1 thăm dò × 3 lần thử); [503, 503, 503] (DEF-R3-05) |
| 3 | TC-P105-44 | chữ «EMBEDDING» thô hiện ở bảng «Mức dùng theo tác vụ» (taskLabel không có nhãn cho EMBEDDING) → DEF-R3-01 (DEF-R3-01) |
| 3 | TC-PU01-05 | pnpm lint rc=0 nhưng 5 cảnh báo (cells.tsx:39, QueueTab.tsx:115/123, Landing.tsx:48, LlmSettings.tsx:13); nền sprint 3 = 0 cảnh báo → DEF-R3-03 (DEF-R3-03) |
| 4 | TC-P208-02 | class_code trùng(hoa-thường) 201; còn lại [422, 422, 422, 422, 422, 422] (DEF-R4-01) |
| 4 | TC-P210-03 | tệp `.xlsx` thực là văn bản (text.xlsx) được nhận 200 (server đọc theo nội dung như CSV); fake-ext/png-as/cut.xlsx → 422 UNSUPPORTED_FILE đúng → |
| 6 | TC-P3-01-49 | Vai DB ứng dụng edupilot rolsuper=t (như BUG-1 gốc); DEF-R6-01 |
| 6 | TC-P3-02-21 | Duyệt/vào lớp không vô hiệu ep:roster (chỉ member_changed khi mời ra/hoàn tác) → tên SV mới không bị chặn ≤1 giờ; DEF-R6-02 |
| 7 | TC-P4-02-03 | go test FAIL: ok  	github.com/edupilot/backend-go/internal/calendar	(cached) FAIL |
| 7 | TC-20 | wall run PR 43712425198 = 15,97 phút (12:09:46→12:25:44), run push e0bf1fc9 = 16,7 phút; job dài nhất Playwright 1/3 = 15,4 phút (shard 2/3 = 6,5, 3/3 = 2,7: lệ |
| 8 | TC-P6-01-20 | weekly A 10, B 8, C 0 và 24 dòng MCQ khớp CSV (đọc qua explain), nhưng giải trình C không có dòng 'không dự → 0 theo quy chế' (DEF-R8-s8core-01) |
| 8 | TC-P6-02-15 | quét float64: internal/seedtool/examhistory.go:130 đưa điểm thi qua float64(c)*2.5 (seed tool); mã sản phẩm sạch |
| 8 | TC-P6-02-25 | C: KT=0, status COMPLETE, nhưng dòng KT chỉ expr '0' (filled_by_user, không quote điều 1.4), không có chữ 'không dự → 0 theo quy chế' (DEF-R8-s8core-01 |
| 8 | TC-P6-02-42 | 200 {'students_total': 30, 'students_computable': 30, 'changed_vs_confirmed': 21, 'entries_carried': 60, 'entries_dropped': [], 'samples': [{'student': {'i |
| 8 | TC-P6-05-03 | status={"state": "DRAFT_UNCLEAR", "unclear_count": 3, "has_policy": true, "pending_draft": true, "gradebook": {"status": "OPEN", "snapsho |
| 8 | TC-P6-05-23 | GS=[{"id": "GRADE_SCHEME_UNCONFIRMED:01a1dff6-f018-7550-b9bb-abfaa2cf3d9d", "kind": setup=[] |
| 8 | TC-P6-05-24 | ['Đã có quy chế, chưa trích công thức.'] |
| 9.5 | TC-N01-12 | DEF-R9.5-16: check-meta nêu đúng tên 7 trường thiếu nhưng `demo_stack:"unknown"` vẫn «meta đủ» (không kiểm giá trị) |
| 9.5 | TC-N01-17 | DEF-R9.5-01: BASE=http://[::1]:3000 và host.docker.internal:3000 KHÔNG bị refuseDemo (chỉ lỗi kết nối) |
| 9.5 | TC-N02-02 | DEF-R9.5-17: trên ep-r 4/7 check-*-seed đạt (chat, crm, docs, grade ×3); assignments/exam/knowledge đỏ do dữ liệu bị ca khác thêm (BT03 đã chấm, 10 c� |
| 9.5 | TC-N02-03 | DEF-R9.5-17: chạy lại seed.mjs trên ep-r: mã 1 «422 IDEMPOTENCY_KEY_REUSED» ở bước 15 (6 điểm giữa kỳ bị ca khác sửa); lượt chạy sau bước |
| 9.5 | TC-N02-06 | DEF-R9.5-18: đăng nhập 4 vai OK, sai mật khẩu 401; GV/TA/SV có việc đúng nguồn; Admin Hôm nay = 0 việc (nguồn Admin chỉ phát sinh khi lỗi hệ th |
| 9.5 | TC-N02-09 | DEF-R9.5-19: bỏ SEED_DEFAULT_PASSWORD vẫn chạy tiếp (đọc .env.local/.env.example), không «Thiếu SEED_DEFAULT_PASSWORD» |
| 9.5 | TC-N02-11 | DEF-R9.5-19: mật khẩu rỗng / toàn khoảng trắng không bị coi là chưa đặt (rơi về .env.local) |
| 9.5 | TC-N04-12 | DEF-R9.5-03: load-a/c không có mục docker stats / ghi 'không OOM, CPU<95 %' |
| 9.5 | TC-N05-03 | DEF-R9.5-04: không có 00018_indexes_tuning.sql (P-95-8 dev: «không tạo», cột quyết định PM để trống) |
| 9.5 | TC-N06-04 | DEF-R9.5-20: Redis tắt → gửi chat trả 200 suy giảm sau 39,5 s (> 10 s; dev ghi ≈ 19 s) = TL-2d còn mở; tạo phiên chat 503 sau 1,9 s |
| 9.5 | TC-N07-10 | DEF-R9.5-08: báo cáo/handoff không ghi lần đo bị nhiễu (CPU nền > 30 %) |
| 9.5 | TC-N08-01 | DEF-R9.5-09: meta.config của 12/13 tệp real-llm chỉ có `note`, không có chuỗi model thực đo và RPM |
| 9.5 | TC-N08-06 | DEF-R9.5-10: 15 lượt mask nhưng không có tỷ lệ placeholder giữ/khôi phục (≥ 95 %) và số lần mô hình đổi placeholder; report.md «chưa so khớ |
| 9.5 | TC-N08-11 | DEF-R9.5-11: R5 Threads 3/3 AI SKIPPED (embedding giả); khủng hoảng chỉ 1 mẫu (Q24), không đủ ≥ 3 mẫu/việc |
| 9.5 | TC-N08-16 | DEF-R9.5-12: real_llm.py không có --resume / không ghi «chưa hoàn tất — hết hạn mức»; 429 Retry-After 120 → chờ 60 s ×5 (≈300 s) rồi LỖI rc=2, |
| 9.5 | TC-N08-17 | DEF-R9.5-12: không có --resume (README nói có); xem TC-N08-16 |
| 9.5 | TC-N09-01 | DEF-R9.5-13: báo cáo research không có mục riêng NFR-02/03 và không đủ «6 phần» ở từng mục |
| 9.5 | TC-N09-04 | DEF-R9.5-14: số R2–R4 trong research report (MAE 1,19; token đầu p95 9,5 s…) không có trong tệp nguồn cùng tên (nay chứa số 9router: MAE 0,688; first |
| 9.5 | TC-N09-05 | DEF-R9.5-14: 3/15 số mẫu không khớp tệp nguồn (R2 vòng 3, R4 first_token, R4 sự kiện đầu 15 vs 14 ms) |
| 9.5 | TC-N09-07 | DEF-R9.5-15: danh sách «câu cần vá» chỉ nêu tệp, không có tệp:dòng |
| 9.5 | TC-N09-11 | DEF-R9.5-07: câu «MacBook Apple Silicon, Colima 6 CPU / 10 GB» chỉ 1 lần; hạn chế loopback/cận dưới không ở từng mục |
| 9.5 | TC-N09-13 | DEF-R9.5-14: FAIL «1.000 bài lần 1» (322/1000, 678 FAILED) và run1 không có ở bảng tổng của research report (bảng ghi «chưa có kết quả») |
| 9.5 | TC-N10-06 | DEF-R9.5-06: PROGRESS.md không có dòng Nợ TL-2c/d/e/f dù techlead.md ghi «chuyển Nợ» |
| 9.5 | TC-N10-10 | DEF-R9.5-05: 9.5 sửa test có sẵn (bf9aa0bb 2 test guard; 811afebd kỳ vọng 429) |
| 9 | TC-P7-01-G06 | TestAssignmentsRBACMatrix không có trong repo (DEF-R9-01); contract -tags testroutes chạy lại lúc máy rảnh: ok 54s (lần đỏ trước = do tải ~20) |
| 9 | TC-P7-02-G10 | TestAssignmentsRBACMatrix: không có trong repo |
| 9 | TC-P7-03-G03 | TestSubmitIdempotent: không có trong repo; Playwright mock 1 mẫu -g xanh |
| 9 | TC-P7-03-G07 | TestSubmitClosed: không có trong repo |
| 9 | TC-P7-06-G11 | TestGradingResultsRBAC: không có trong repo; TestNoDraftCanaryToStudent: không có trong repo |
| 9 | TC-P7-08-G08 | TestTAReviewsCannotPublish: không có trong repo; Playwright mock 1 mẫu -g xanh |
| 9.5 | TC-N01-12 | DEF-R9.5-16: check-meta nêu đúng tên 7 trường thiếu nhưng `demo_stack:"unknown"` vẫn «meta đủ» (không kiểm giá trị) |
| 9.5 | TC-N01-17 | DEF-R9.5-01: BASE=http://[::1]:3000 và host.docker.internal:3000 KHÔNG bị refuseDemo (chỉ lỗi kết nối) |
| 9.5 | TC-N02-02 | DEF-R9.5-17: trên ep-r 4/7 check-*-seed đạt (chat, crm, docs, grade ×3); assignments/exam/knowledge đỏ do dữ liệu bị ca khác thêm (BT03 đã chấm, 10 c� |
| 9.5 | TC-N02-03 | DEF-R9.5-17: chạy lại seed.mjs trên ep-r: mã 1 «422 IDEMPOTENCY_KEY_REUSED» ở bước 15 (6 điểm giữa kỳ bị ca khác sửa); lượt chạy sau bước |
| 9.5 | TC-N02-06 | DEF-R9.5-18: đăng nhập 4 vai OK, sai mật khẩu 401; GV/TA/SV có việc đúng nguồn; Admin Hôm nay = 0 việc (nguồn Admin chỉ phát sinh khi lỗi hệ th |
| 9.5 | TC-N02-09 | DEF-R9.5-19: bỏ SEED_DEFAULT_PASSWORD vẫn chạy tiếp (đọc .env.local/.env.example), không «Thiếu SEED_DEFAULT_PASSWORD» |
| 9.5 | TC-N02-11 | DEF-R9.5-19: mật khẩu rỗng / toàn khoảng trắng không bị coi là chưa đặt (rơi về .env.local) |
| 9.5 | TC-N04-12 | DEF-R9.5-03: load-a/c không có mục docker stats / ghi 'không OOM, CPU<95 %' |
| 9.5 | TC-N05-03 | DEF-R9.5-04: không có 00018_indexes_tuning.sql (P-95-8 dev: «không tạo», cột quyết định PM để trống) |
| 9.5 | TC-N06-04 | DEF-R9.5-20: Redis tắt → gửi chat trả 200 suy giảm sau 39,5 s (> 10 s; dev ghi ≈ 19 s) = TL-2d còn mở; tạo phiên chat 503 sau 1,9 s |
| 9.5 | TC-N07-10 | DEF-R9.5-08: báo cáo/handoff không ghi lần đo bị nhiễu (CPU nền > 30 %) |
| 9.5 | TC-N08-01 | DEF-R9.5-09: meta.config của 12/13 tệp real-llm chỉ có `note`, không có chuỗi model thực đo và RPM |
| 9.5 | TC-N08-06 | DEF-R9.5-10: 15 lượt mask nhưng không có tỷ lệ placeholder giữ/khôi phục (≥ 95 %) và số lần mô hình đổi placeholder; report.md «chưa so khớ |
| 9.5 | TC-N08-11 | DEF-R9.5-11: R5 Threads 3/3 AI SKIPPED (embedding giả); khủng hoảng chỉ 1 mẫu (Q24), không đủ ≥ 3 mẫu/việc |
| 9.5 | TC-N08-16 | DEF-R9.5-12: real_llm.py không có --resume / không ghi «chưa hoàn tất — hết hạn mức»; 429 Retry-After 120 → chờ 60 s ×5 (≈300 s) rồi LỖI rc=2, |
| 9.5 | TC-N08-17 | DEF-R9.5-12: không có --resume (README nói có); xem TC-N08-16 |
| 9.5 | TC-N09-01 | DEF-R9.5-13: báo cáo research không có mục riêng NFR-02/03 và không đủ «6 phần» ở từng mục |
| 9.5 | TC-N09-04 | DEF-R9.5-14: số R2–R4 trong research report (MAE 1,19; token đầu p95 9,5 s…) không có trong tệp nguồn cùng tên (nay chứa số 9router: MAE 0,688; first |
| 9.5 | TC-N09-05 | DEF-R9.5-14: 3/15 số mẫu không khớp tệp nguồn (R2 vòng 3, R4 first_token, R4 sự kiện đầu 15 vs 14 ms) |
| 9.5 | TC-N09-07 | DEF-R9.5-15: danh sách «câu cần vá» chỉ nêu tệp, không có tệp:dòng |
| 9.5 | TC-N09-11 | DEF-R9.5-07: câu «MacBook Apple Silicon, Colima 6 CPU / 10 GB» chỉ 1 lần; hạn chế loopback/cận dưới không ở từng mục |
| 9.5 | TC-N09-13 | DEF-R9.5-14: FAIL «1.000 bài lần 1» (322/1000, 678 FAILED) và run1 không có ở bảng tổng của research report (bảng ghi «chưa có kết quả») |
| 9.5 | TC-N10-06 | DEF-R9.5-06: PROGRESS.md không có dòng Nợ TL-2c/d/e/f dù techlead.md ghi «chuyển Nợ» |
| 9.5 | TC-N10-10 | DEF-R9.5-05: 9.5 sửa test có sẵn (bf9aa0bb 2 test guard; 811afebd kỳ vọng 429) |

## Chi tiết theo sprint

---
<!-- defects-1.md -->
### DEF-R1-01 — DEMO_SCRIPT nêu spec E2E không có trong FLOWS.md
- Mức: Thấp (tài liệu)
- Sprint gốc: 1 · TC gốc: US-P0-01/TC-07 · Hồi quy do sprint sau?: CÓ (DEMO v2 bd032bf2 + sprint 10.8 thêm spec, không cập nhật FLOWS)
- Môi trường: be0c0d96, không cần stack
- Bước tái hiện: 1. `bash docs/sprints/1/qc/scripts/tc-US-P0-01.sh TC-07`
- Kỳ vọng (AC7): mọi `*.spec.ts` nêu ở DEMO_SCRIPT có trong FLOWS.md (đúng mục flow) · Thực tế: THIẾU audit.spec.ts, insights.spec.ts, checkin-code.spec.ts; attendance-checkin/-display nằm ở mục tự phục vụ chứ không ở F7
- Bằng chứng: docs/regression/evidence-1/run-01.log · Nghi ngờ mã: docs/DEMO_SCRIPT.md dòng 116, 162; docs/FLOWS.md mục F7 (dòng ~104–112)
- Chạy lại lần 2: ổn định

---
<!-- defects-1.5.md -->
### DEF-R1.5-01 — Đã đăng nhập mà mở đường dẫn không có thật bị đưa về /login thay vì màn «Không tìm thấy trang»
- Mức: Trung bình
- Sprint gốc: 1.5 · TC gốc: US-PROTO-00/TC-00-57, TC-00-59 (AC13 / FR-X16) · Hồi quy do sprint sau?: CÓ (cổng AuthGate + `(app)/[...slug]` ở sprint P1-PU; không rõ commit) → KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, vai sv.kha (và ta/teacher/admin), 1440 px
- Bước tái hiện: 1. Đăng nhập sv.kha tại https://localhost:1143/login. 2. `page.goto('/me')` → vẫn đăng nhập (POST /auth/refresh 200). 3. `page.goto('/khong-co-trang')` (hoặc /threads/khong-co, /assignments/khong-co) → URL thành `/login?next=%2Fkhong-co-trang`, không có lời gọi /auth/refresh, trang là màn Đăng nhập. Script: evidence-1.5/scripts/dbg-404.mjs.
- Kỳ vọng: màn «Không tìm thấy trang» + nút «Về Hôm nay» (frontend/src/app/(app)/not-found.tsx, app/not-found.tsx) · Thực tế: màn Đăng nhập
- Bằng chứng: docs/regression/evidence-1.5/matrix-*.json (mục notfound) · Nghi ngờ mã: frontend/src/shared/session/AuthGate.tsx:36-49 (bounce → /login?next=) khi pathname không thuộc route đã biết; frontend/src/app/(app)/[...slug]/page.tsx
- Chạy lại lần 2: ổn định (tái hiện 3/3)

### DEF-R1.5-02 — TA: nút hồ sơ ở topbar tràn ngang ở 375 và 390 px
- Mức: Thấp (phụ thuộc dữ liệu: TA là thành viên lớp có tên dài, vd «QC-AS3r108 · QC điểm danh QC-AS3r108»)
- Sprint gốc: 1.5 · TC gốc: US-PROTO-00/TC-00-36 (AC10: không tràn ngang) · Hồi quy do sprint sau?: KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, ta@edupilot.local, trang «/»
- Bước tái hiện: 1. Đăng nhập ta@edupilot.local. 2. Viewport 375×812 (390×812) → `document.documentElement.scrollWidth - innerWidth` = 24 (9). 3. Phần tử tràn: `AppShell topEnd` / `Menu root` / `profile` (right = 399). GV cùng viewport: 0; 414 px: 0.
- Kỳ vọng: không cuộn ngang ở 375/390 · Thực tế: tràn 24/9 px · Bằng chứng: evidence-1.5/matrix-ta.json (overflow), evidence-1.5/scripts/dbg-overflow.mjs · Nghi ngờ mã: frontend/src/shared/shell/AppShell (khối topEnd không co lại khi bộ chọn lớp dài)
- Chạy lại lần 2: ổn định

---
<!-- defects-2.md -->
### DEF-R2-01 — `openapi.yaml` không qua `redocly lint` (1 lỗi struct)
- Mức: Trung bình
- Sprint gốc: 2 (US-PG-06; cũng sprint 1 US-P0-02/TC-21) · TC gốc: TC-PG06-01, P0-02/TC-21 · Hồi quy do sprint sau?: CÓ (sprint 11 đợt 2, commit 0d38d7a5 «US-P10-06 API thống kê điểm thi»)
- Môi trường: be0c0d96, không cần stack
- Bước tái hiện: 1. `cd` gốc repo. 2. `pnpm exec redocly lint backend-go/api/openapi.yaml` → rc 1, «10370:7 error struct Expected type `ParameterList` (array) but got `null`».
- Kỳ vọng: 0 lỗi lint · Thực tế: 1 lỗi + 21 cảnh báo; thao tác `compareExamAnalytics` (GET /api/v1/analytics/exams/compare) khai báo `parameters:` rỗng
- Bằng chứng: evidence-1/run-02-tc21.log · Nghi ngờ mã: backend-go/api/openapi.yaml:10370 (xoá khoá `parameters:` hoặc điền danh sách)
- Chạy lại lần 2: ổn định

### DEF-R2-02 — `go test ./internal/contract` đỏ: TestStatusParity / TestSpec_EveryDocumentedStatusExercised
- Mức: Cao (hợp đồng API↔spec; có một 500 INTERNAL thay vì 404)
- Sprint gốc: 2 (US-PG-06) · TC gốc: TC-PG06-10, TC-PG06-11 (TC-PG06-28 lỗi thời cách chọn test) · Hồi quy do sprint sau?: CÓ (scenarios_exam_test.go thuộc sprint 9.x; spec mở rộng 298 route)
- Môi trường: be0c0d96, testcontainers (Colima), máy tải cao (load ~20)
- Bước tái hiện: 1. `cd backend-go && go test -race -count=1 -tags testroutes -v -run TestStatusParity ./internal/contract`. Lượt đầy đủ: `scenarios_exam_test.go:606: GET …/exams/{eid}/attempts/{aid}/code/{pid}/submissions → 500 (cần 404): {"code":"INTERNAL"…}`; lượt chạy riêng: `scenarios_test.go:441: POST /api/v1/admin/llm/providers → 504 (cần 201) DEADLINE_EXCEEDED` (nghi tải máy). `TestSpec_EveryDocumentedStatusExercised`: nhiều trạng thái khai báo (DELETE /admin/llm/providers/{id}: 204/401/403/404/409; DELETE /chat/sessions/{sid}…) «không test nào sinh ra».
- Kỳ vọng: TestStatusParity, TestSpec_EveryDocumentedStatusExercised PASS · Thực tế: FAIL cả hai lần (chạy riêng và cả bộ)
- Bằng chứng: /tmp/qc2-A-statusparity.log; evidence-2/run-logs/06.log · Nghi ngờ mã: backend-go/internal/contract/scenarios_exam_test.go:606 (500 thay vì 404 ở submissions mã lập trình), contract/exempt.go (thiếu miễn trừ cho route mới)
- Chạy lại lần 2: trượt (2/2)
- **Phân tích SmkJ (23:40, chạy riêng `TestStatusParity` với env CI `TESTCONTAINERS_RYUK_DISABLED=true EP_SKIP_TIMING=1 -race -tags testroutes -count=1`, load máy 18–20, VM Colima load 43–62):** lượt 23:33 trượt ở `scenarios_test.go:501 PUT /admin/llm/routes → 504 DEADLINE_EXCEEDED` (không tới được dòng 606; log gateway của rig bị `io.Discard` ở `rig_test.go`, không có dòng log/stack để trích). Mỗi lượt đỏ ở MỘT yêu cầu khác nhau (441 → 501 → 606; DEF-R10-01 cũng khác endpoint mỗi lần) ⇒ không phải một lỗi xác định. Nguyên nhân gốc đọc từ mã: `rig_test.go:66 REQUEST_TIMEOUT=1s` + Postgres/Redis/MinIO testcontainers DÙNG CHUNG toàn máy (tên cố định, Reuse) với mọi agent đang chạy `go test` ⇒ một câu SQL > 1 s là hết hạn. Hết hạn chưa ghi gì → `timeoutMiddleware` trả 504 (đúng spec); NHƯNG 27 chỗ `apierr.Internal` ở các handler (vd. `examhttp/questions.go:99`, mặc định của `h.fail`) ghi 500 INTERNAL cho mọi lỗi lạ, kể cả `context deadline exceeded` từ DB, TRƯỚC khi middleware kịp xét (`!sw.wrote` sai) ⇒ cùng nguyên nhân cho ra 500 (606, DEF-R10-01: `POST …/report`, `/auth/verify-email`) hoặc 504 (441, 501). Kết luận: **do tải** ở mức kiểm thử (rig 1 s); kèm lỗi nhẹ thật của sản phẩm: lỗi DB do hết hạn bị gói thành 500 thay vì 504 (SRS 3.4) ở handler không dùng ánh xạ `DeadlineExceeded` (chỉ `llmhttp/verify.go`, `observabilityhttp`, `todayhttp` có). Chưa có lượt xanh trên máy rảnh (tải không xuống dưới 8) — lượt kế tiếp: xem mục «lượt chạy» ở smoke-2.md.

---
<!-- defects-3.md -->
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

---
<!-- defects-4.md -->
### DEF-R4-01 — Mã lớp phân biệt hoa/thường: tạo được hai lớp `SK20D91` và `sk20d91`
- Mức: Trung bình
- Sprint gốc: 4 · TC gốc: TC-P208-02 (AC1: `class_code` trùng khác hoa-thường → 409) · Hồi quy do sprint sau?: KHÔNG (có sẵn: `normClass` không đổi chữ từ US-P2-08 commit 3651adab; index `courses_class_code_key` không `lower()`; báo cáo gốc ghi PASS nhưng không tái hiện)
- Môi trường: ep-r, be0c0d96, ADMIN
- Bước tái hiện: 1. `POST /admin/courses` `{"subject_code":"INT1006","class_code":"SK20D91","name":"x","semester":"2026-2027-HK1"}` → 201; 2. lặp với `class_code":"sk20d91"` (key khác) → 201 (kỳ vọng 409 `details.field=class_code`). `select class_code from courses where class_code ilike 'sk20d91'` → 2 dòng.
- Kỳ vọng: 409 CONFLICT · Thực tế: 201, hai lớp có mã chỉ khác hoa/thường (SV nhập mã lớp dễ nhầm; tìm lớp theo mã không rõ ràng)
- Bằng chứng: `evidence-4/p2c.out` (TC-P208-02) · Nghi ngờ mã: `backend-go/internal/course/admin.go:114-120` (`normClass`), `backend-go/db/migrations/00003_course_foundation.sql:37,49`
- Chạy lại lần 2: ổn định (tạo lại với mã khác: cùng kết quả). Đã lưu trữ cả hai lớp QC sau khi đo.

### DEF-R4-02 — Import roster nhận tệp `.xlsx` thực chất là văn bản (TC-P210-03)
- Mức: Thấp
- Sprint gốc: 4 · TC gốc: TC-P210-03 (AC1/AC13 magic bytes) · Hồi quy do sprint sau?: KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, GV của lớp
- Bước tái hiện: tạo `text.xlsx` bằng `gen.py` (nội dung CSV, đuôi .xlsx); `POST /courses/{id}/roster/import?dry_run=true` multipart `file=@text.xlsx` + `Idempotency-Key` → 200 (đọc như CSV). `fake-ext.csv` (ELF), `png-as.csv`, `cut.xlsx` → 422 UNSUPPORTED_FILE đúng.
- Kỳ vọng: 422 UNSUPPORTED_FILE (đuôi/ nội dung không khớp magic bytes) · Thực tế: 200
- Bằng chứng: `evidence-4/p2h.out` · Nghi ngờ mã: bộ đọc roster ở `backend-go/internal/course/` (nhận diện theo nội dung, bỏ qua đuôi)
- Chạy lại lần 2: ổn định (không có hại: dữ liệu vẫn qua kiểm tra dòng)

---
<!-- defects-5.md -->
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

## Ghi nhận không tính lỗi
- Go `TestRunningNotCancelled` đỏ 1 lần khi máy tải nặng (lease 30 s hết hạn), PASS 2/2 khi chạy riêng — chập chờn do tải.
- TC-PE02-37: fork bomb A8 ra `WA` (FORKED=0) thay vì `TLE` như gốc; bất biến `procPeak==1`/0 tiến trình mồ côi thoả.
- Today bỏ provider `staff`/`assignment` khi stack quá tải (timeout) — degrade có chủ đích.

---
<!-- defects-5.5.md -->
### Không có lỗi riêng cho 5.5 (phần đã chạy). DEF-R5-01 (AppShell axe) ghi ở defects-5.md — áp cho mọi màn 5.5.

---
<!-- defects-6.md -->
### DEF-R6-01 — Vai DB của ứng dụng là siêu quyền
- Mức: Thấp
- Sprint gốc: 6 · TC gốc: TC-P3-01-49 · Hồi quy do sprint sau?: KHÔNG (có sẵn — chính là BUG-1 trong `docs/sprints/6/qc/report-US-P3-01.md`)
- Môi trường: ep-r, be0c0d96; gateway nối Postgres bằng vai `edupilot`
- Bước tái hiện: 1. `docker exec ep-r-postgres-1 psql -U edupilot -d edupilot -Atc "select rolsuper,rolbypassrls from pg_roles where rolname='edupilot'"` → `t|t`.
- Kỳ vọng: vai ứng dụng không `rolsuper`; Thực tế: `rolsuper=t`. Trigger append-only (`pii_events`, `audit_log`) vẫn chặn UPDATE/DELETE với 42501 nên không mất bất biến.
- Bằng chứng: lệnh trên; `docs/regression/smoke-6.md` · Nghi ngờ mã: cấu hình compose (user Postgres = user ứng dụng), không phải mã Go
- Chạy lại lần 2: ổn định

### DEF-R6-02 — Bộ nhớ đệm roster PII không bị vô hiệu khi sinh viên vào lớp (duyệt hoặc vào bằng mã)
- Mức: Cao (tên SV mới vào lớp không được che trong Threads/chat tới 1 giờ ⇒ lộ họ tên người khác ở kênh công khai)
- Sprint gốc: 6 · TC gốc: TC-P3-02-21 · Hồi quy do sprint sau?: KHÔNG RÕ (có vẻ có sẵn: `members.go` ở db7c187b đã chỉ phát `course.member_changed` ở mời ra / hoàn tác; duyệt phát `course.join_decided`, vào tự động không phát gì; worker chỉ gắn `roster.Invalidate` vào `course.member_changed`, `cmd/worker/registry.go:129`)
- Môi trường: ep-r, be0c0d96; GV `teacher`, SV; lớp E6C1 (`class_code=E6C1`)
- Bước tái hiện: 1. Bật `require_approval` (PUT `/courses/{id}/join-settings`). 2. `redis-cli DEL ep:roster:<course>`; SV A `POST …/threads/precheck` để dựng cache (không có SV B). 3. SV B `POST /courses/join` (PENDING) rồi GV `POST /courses/{id}/members/{B}/approve` (200). 4. `redis-cli EXISTS ep:roster:<course>` sau 10 s → vẫn `1`; `precheck` với «Đỗ Bảo Hải …» (tên SV B) → `allowed:true`. Tương tự khi lớp không cần duyệt (vào mã). Mời ra thì cache xoá sau 0,5 s (đúng).
- Kỳ vọng (SRS P3-02 AC3 / TC-P3-02-21): cache roster xoá ≤ 5 s sau khi thành viên đổi, tên SV mới bị chặn; Thực tế: cache sống đến hết TTL 1 giờ.
- Bằng chứng: lệnh trên (chạy hai lần, cùng kết quả); outbox chỉ có `course.member_changed` cho thao tác mời ra · Nghi ngờ mã: `backend-go/internal/course/members.go:222` (approve chỉ ghi `TopicJoinDecided`), `backend-go/internal/course/join.go:185-269` (không ghi sự kiện khi ACTIVE ngay), `cmd/worker/registry.go:129`
- Chạy lại lần 2: ổn định

---
<!-- defects-7.md -->
### DEF-R7-01 — CI chạy đủ job vượt ngưỡng cứng 10 phút của US-CI-02
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-20 (US-CI-02) · Hồi quy do sprint sau?: CÓ (các sprint 9.x–11 thêm nhiều spec Playwright vào cùng 3 shard; `.github/workflows/ci.yml` vẫn `shard: [1,2,3]`)
- Môi trường: GitHub Actions, nhánh sprint/11-p10-eval @be0c0d96 (run PR 43712425198), `gh run view`
- Bước tái hiện: 1. `gh run view 43712425198 --json jobs,createdAt,updatedAt`. 2. wall = 12:09:46 → 12:25:44 = 15,97 phút; job «Frontend e2e (Playwright 1/3)» 15,4 phút, 2/3 6,5 phút, 3/3 2,7 phút; Lighthouse simulate 10,3 phút, Go rest-1 9,5 phút. Run push e0bf1fc9 (`43710464639`) cũng 16,7 phút.
- Kỳ vọng (PM 2026-10-13): lượt chạy đủ wall ≤ 10 phút và nhanh hơn mốc cũ 13 m 43 s; Thực tế: ~16 phút, chậm hơn mốc cũ.
- Bằng chứng: `/tmp/qc2e/ci-run.json`, `/tmp/qc2e/ci-log.txt` · Nghi ngờ: `.github/workflows/ci.yml:235-253` (3 shard không cân bằng theo thời gian), job Lighthouse simulate 10,3 phút
- Chạy lại lần 2: ổn định (hai run liên tiếp ≥ 16 phút)

---
<!-- defects-8.md -->


### DEF-R8-00 — (SmkD) Xem trước sổ lớp 1 lệch 7/30 — xem S8Core để kết luận
- Triệu chứng: `p6-preview-vs-csv.py --by-account` 23/30 (sv10 qt 8,5 vs 6,0; sv23 hp_raw 4,1 vs 4,0). hand-calc 30/30 và go test grade xanh.

---
<!-- defects-9.md -->
### DEF-R9-01 — Tên test ở dòng `Kiểm:` của US không có trong repo (hoặc ở gói khác)
- Mức: Thấp
- Sprint gốc: 9 · TC gốc: TC-P7-01-G06, TC-P7-02-G10, TC-P7-03-G03, TC-P7-03-G07, TC-P7-06-G11, TC-P7-08-G08 · Hồi quy do sprint sau?: KHÔNG (có sẵn — BUG-P7-03-3 / handoff dev-US-P7-0{1,2,6}: `TestAssignmentsRBACMatrix`, `TestSubmitIdempotent`, `TestSubmitClosed`, `TestGradingResultsRBAC`, `TestNoDraftCanaryToStudent` chưa bao giờ có)
- Môi trường: ep-r/be0c0d96, `go test -tags integration[,testroutes] -run <tên> -v <gói>`
- Bước tái hiện: chạy lệnh ở hàng `-G` tương ứng; `logs/g-go.jsonl`, `logs/g-go-fix.jsonl`
- Kỳ vọng: test được nêu tên tồn tại và PASS · Thực tế: `no tests to run` / không có hàm
- Bằng chứng: `docs/regression/evidence-9/logs/TC_P7_*_G*.log`
- Chạy lại lần 2: ổn định

### DEF-R9-02 — ĐÃ LOẠI: `go test ./internal/contract/...` đỏ lúc tải cao
- Chạy lại `go test -count=1 -tags testroutes ./internal/contract/...` lúc máy rảnh: ok (25 PASS, rc=0, logs/contract-rerun.log). Lần đỏ trước do timeout/504 khi tải ~20 → môi trường, không phải lỗi.


---
<!-- defects-9.5.md -->
# Lỗi hồi quy sprint 9.5 — qc2, mã be0c0d96 (sprint/11-p10-eval), 2026-11-15

Chạy trên ep-r (dữ liệu dùng chung, nhiễm bởi ca khác) và stack riêng ep-r95-* (postgres `ep-r95-pg`, gateway/worker/redis/mail/docling riêng, DB `edupilot_t1_r95`). Chạy lại lần 2 của mọi ca F chạy được: ổn định, trừ khi ghi khác. Không phát hiện lỗi nào do sprint sau gây ra; mọi lỗi dưới đây có sẵn từ sprint gốc 9.5 (hoặc từ spec/TC) trừ khi ghi khác.

### DEF-R9.5-01 — `refuseDemo` bỏ lọt `[::1]` và `host.docker.internal`
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N01-17 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: ep-r, be0c0d96, người chạy script
- Bước tái hiện: `BASE=http://[::1]:3000 PHASE=seed SEED_DEFAULT_PASSWORD=x node deploy/vps-llm.mjs`; tương tự `http://host.docker.internal:3000`. (Port 3000 không có dịch vụ nên chỉ lỗi kết nối, không ghi gì.) Ngoài ra lỗi từ chối in stack trace Node nhiều dòng thay vì 1 dòng.
- Kỳ vọng (theo TC): cả 5 biến thể bị từ chối bằng cùng cơ chế · Thực tế: `127.0.0.1`, `localhost`, `/` cuối, `https://localhost` bị từ chối; `[::1]` và `host.docker.internal` đi tiếp tới `fetch`
- Bằng chứng: docs/regression/evidence-9.5/n01-refuse.log · Nghi ngờ mã: benchmarks/lib/meta.mjs:10-16 (`DEMO_PORTS`, so `hostname` chỉ với `localhost`/`127.0.0.1`)
- Chạy lại lần 2: ổn định

### DEF-R9.5-02 — `canary-scan.sh` không xoá nhà cung cấp canary
- Mức: Thấp
- Sprint gốc: 9.5 (script US-P1-04) · TC gốc: TC-N01-18 · Hồi quy do sprint sau?: KHÔNG RÕ (phản hồi tạo nhà cung cấp nay có `models[].id`)
- Môi trường: ep-r, be0c0d96, ADMIN
- Bước tái hiện: chạy `scripts/canary-scan.sh` với `A="Authorization: Bearer <ADMIN>"`; `GET /admin/llm/providers` thấy `CN0`, `CN1` còn lại
- Kỳ vọng: 7 dòng đếm = 0 và dọn dẹp · Thực tế: 7 dòng = 0 (đạt) nhưng `DELETE` nhận id của mô hình (regex `.*"id"` tham lam lấy id cuối) nên 2 nhà cung cấp rác còn trong cấu hình toàn cục (QC đã xoá tay)
- Bằng chứng: docs/regression/evidence-9.5/n01-canary.log · Nghi ngờ mã: scripts/canary-scan.sh:19 (`sed -n 's/.*"id":"…"/…'`)
- Chạy lại lần 2: ổn định

### DEF-R9.5-03 — load-a/c không có mẫu `docker stats` / ghi «không OOM, CPU < 95 %»
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N04-12 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: ep-r, be0c0d96
- Bước tái hiện: `grep -il 'oom\|docker stats\|cpu_pct' benchmarks/reports/load-*.json` → 0 tệp; không dòng nào trong báo cáo nói không có OOM / CPU ≥ 95 % suốt 60 s
- Kỳ vọng: có mục mẫu hoặc sự kiện · Thực tế: không có · Bằng chứng: n04-misc.log · Nghi ngờ mã: benchmarks/load/run-t1.mjs (không lấy docker stats)
- Chạy lại lần 2: ổn định

### DEF-R9.5-04 — Không có `00018_indexes_tuning.sql`, đề xuất P-95-8 (dev) chưa có quyết định PM
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N05-03 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: ep-r, be0c0d96
- Bước tái hiện: `ls backend-go/db/migrations` (00016, 00017, 00019…); `docs/sprints/9.5/proposals.md` dòng P-95-8 đầu: cột «Quyết định PM» trống
- Kỳ vọng (theo TC/AC2): thêm 00018 · Thực tế: không có; EXPLAIN T1 ≤ 13 ms nên dev đề xuất không tạo. QC rerun EXPLAIN: 9/10 truy vấn ≤ 11 ms, `rag_hybrid` 68,8 ms (dev 13,1 ms) do tải chung
- Bằng chứng: docs/regression/evidence-9.5/explain-r95.json, n05-explain-run.log · Nghi ngờ mã: –
- Chạy lại lần 2: ổn định (8 TC kèm theo ghi LT)

### DEF-R9.5-05 — 9.5 sửa test có sẵn
- Mức: Trung bình
- Sprint gốc: 9.5 · TC gốc: TC-N10-10 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: be0c0d96, git
- Bước tái hiện: `git log --no-merges --format='%h %s' ff00b5aa..2d730a4e -- '*_test.go'` → `bf9aa0bb` («sửa 2 test guard của main»), `811afebd` (đổi kỳ vọng 429 trong provider_test.go)
- Kỳ vọng: chỉ thêm test hồi quy; sửa test có sẵn phải báo PM · Thực tế: 429 có P-95-8 (PM duyệt TL-2j); 2 test guard (decision-gates ADMIN-only, KnowledgeSimilarPosts) không có quyết định PM nêu rõ
- Bằng chứng: n08-n10-scans.log · Nghi ngờ mã: backend-go/internal/llm/provider/provider_test.go, backend-go/internal/decision/*_test.go
- Chạy lại lần 2: ổn định

### DEF-R9.5-06 — PROGRESS.md thiếu dòng Nợ cho TL-2c/d/e/f
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N10-06 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `grep -n 'TL-2[cdef]' docs/PROGRESS.md` → 0 dòng, dù techlead.md ghi «chuyển Nợ (PROGRESS.md)»
- Kỳ vọng: mỗi lỗi vừa/thấp có dòng Nợ · Thực tế: chỉ có Lighthouse /assignments/[id] (sprint 11)
- Bằng chứng: n08-n10-scans.log · Chạy lại lần 2: ổn định

### DEF-R9.5-07 — Báo cáo cuối không nêu hạn chế Mac ở từng mục
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N09-11 (TC-N04-15) · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `grep -c 'MacBook Apple Silicon, Colima 6 CPU / 10 GB' docs/research/2026-11-14-nfr-real-llm.md` → 1; `grep -in 'cận dưới\|loopback' docs/research/…nfr-real-llm.md docs/sprints/9.5/report.md` → 0 (chỉ ở techlead.md:74, dev-lat-2.md)
- Kỳ vọng: ở mọi mục có số · Thực tế: một lần · Bằng chứng: n08-n10-scans.log · Chạy lại lần 2: ổn định

### DEF-R9.5-08 — Không ghi lần đo bị nhiễu
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N07-10 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `grep -niE 'nhiễu|CPU nền' docs/research/2026-11-14-nfr-real-llm.md docs/sprints/9.5/handoff/*.md` → 0
- Kỳ vọng: ghi lần nhiễu (CPU nền > 30 %) và đo lại · Thực tế: không có · Chạy lại lần 2: ổn định

### DEF-R9.5-09 — `meta.config` của tệp real-llm thiếu chuỗi model và RPM
- Mức: Trung bình
- Sprint gốc: 9.5 · TC gốc: TC-N08-01 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `jq -c '.meta.config' benchmarks/reports/real-llm/*.json` → 12/13 tệp chỉ có `note` «khối meta bổ sung khi gộp 9.5»; model nằm trong `audit.models`
- Kỳ vọng: chuỗi model thực đo + RPM trong meta · Thực tế: không · Bằng chứng: n08-recompute.log · Nghi ngờ mã: benchmarks/real-llm/real_llm.py (không ghi meta), commit 2d730a4e
- Chạy lại lần 2: ổn định

### DEF-R9.5-10 — R3 (mask) thiếu tỷ lệ placeholder và số lần mô hình đổi placeholder
- Mức: Trung bình
- Sprint gốc: 9.5 · TC gốc: TC-N08-06 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: đọc benchmarks/reports/real-llm/mask.json (15 lượt) và report.md §3: «chưa so khớp tự động tên người khác»
- Kỳ vọng: ≥ 95 % giữ + khôi phục (tính lại), có số lần đổi/dịch/bỏ placeholder · Thực tế: không có hai số này · Chạy lại lần 2: ổn định

### DEF-R9.5-11 — R5 không đủ mẫu / AI Threads không có đầu ra
- Mức: Trung bình
- Sprint gốc: 9.5 · TC gốc: TC-N08-11 · Hồi quy do sprint sau?: KHÔNG (có sẵn; P-11-19 đã xếp hàng)
- Bước tái hiện: `jq` threads.json: 3/3 `ai_state=SKIPPED` (`LOW_SCORE`, embedding giả); chat.json chỉ có Q24 là tin khủng hoảng
- Kỳ vọng: ≥ 3 mẫu mỗi việc, khủng hoảng luôn đường an toàn · Thực tế: 1 mẫu khủng hoảng, 0 bài AI trả lời · Chạy lại lần 2: ổn định

### DEF-R9.5-12 — `real_llm.py` không có `--resume`, không dừng sớm khi hết hạn mức
- Mức: Cao
- Sprint gốc: 9.5 · TC gốc: TC-N08-16, TC-N08-17 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: máy chủ giả (docs/regression/evidence-9.5/scripts/fake-gateway.mjs)
- Bước tái hiện: `MODE=429 RA=120 node …/fake-gateway.mjs`; `BASE=http://127.0.0.1:18099 python3 benchmarks/real-llm/real_llm.py setup`
- Kỳ vọng: dừng lượt, ghi «chưa hoàn tất — hết hạn mức», lưu tiến độ, chạy tiếp bằng `--resume` · Thực tế: chờ 60 s × 5 (≈ 300 s, bỏ qua `Retry-After` 120) rồi «LỖI … 429» rc=2; không lưu tiến độ; `grep resume benchmarks/real-llm/*.py` → 0 (README mục `--resume` nói có)
- Bằng chứng: n08-harness-429.log, n08-fake429.log · Nghi ngờ mã: benchmarks/real-llm/real_llm.py:100-125, 172-177 (`backoff` tối đa 60 s, `tries=6`)
- Chạy lại lần 2: ổn định

### DEF-R9.5-13 — Báo cáo research thiếu mục NFR-02/03 và «6 phần»
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N09-01 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `grep -n '^##' docs/research/2026-11-14-nfr-real-llm.md` → (a)(b)(c)(d), Baseline, Chaos, EXPLAIN, LLM; không có seed T0/T1
- Chạy lại lần 2: ổn định

### DEF-R9.5-14 — Báo cáo research lệch tệp nguồn sau khi đo lại qua 9router
- Mức: Cao
- Sprint gốc: 9.5 · TC gốc: TC-N09-04, TC-N09-05, TC-N09-13 · Hồi quy do sprint sau?: KHÔNG (có sẵn; P-95-8 đo lại R1–R6 nhưng chỉ report.md được cập nhật)
- Bước tái hiện: so docs/research/2026-11-14-nfr-real-llm.md (R2 vòng 3 «MAE 1,19, 3/12», R4 «token đầu p50 7,1 s, p95 9,5 s», sự kiện đầu 15 ms, «1.000 bài chưa có kết quả») với benchmarks/reports/real-llm/grade-report.json (MAE 0,688; 9/12), chat.json (first_token p50 21, p95 4.217 ms; first_event p95 14 ms) và load-c-1000-9router-run{1-hong,2}.json (FAIL 322/1000 rồi PASS 1000/1000 trong 1,68 h)
- Kỳ vọng: 100 % số truy được về tệp nguồn; mọi FAIL có ở bảng tổng · Thực tế: các số cũ (Gemini trực tiếp) chỉ còn trong lan-1-khong-hop-le/; run1 FAIL vắng ở bảng tổng
- Bằng chứng: n08-recompute.log · Chạy lại lần 2: ổn định

### DEF-R9.5-15 — «Câu cần vá» không có tệp:dòng
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N09-07 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: mục «Danh sách câu cần vá» của báo cáo research chỉ nêu tên 3 tệp
- Chạy lại lần 2: ổn định

### DEF-R9.5-16 — `check-meta.mjs` không kiểm giá trị `demo_stack`
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N01-12 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: sao chép load-d.json, đặt `meta.demo_stack="unknown"`, `node benchmarks/check-meta.mjs <thư mục>` → «meta đủ»
- Kỳ vọng: mã ≠ 0 · Thực tế: mã 0 · Nghi ngờ mã: benchmarks/check-meta.mjs:15 (chỉ kiểm `=== undefined || ""`)
- Bằng chứng: n01-n02-quick.log · Chạy lại lần 2: ổn định

### DEF-R9.5-17 — Seed không chạy lại được sau khi dữ liệu bị sửa; check-*-seed đếm tuyệt đối
- Mức: Trung bình
- Sprint gốc: 8/9 (seed.mjs bước 15, US-S8-SEED) · TC gốc: TC-N02-02, TC-N02-03 · Hồi quy do sprint sau?: KHÔNG RÕ (môi trường ep-r nhiễu: 7 agent cùng ghi)
- Môi trường: ep-r, be0c0d96
- Bước tái hiện: `node scripts/seed.mjs` trên ep-r → «Seed lỗi: POST …/grade-entries/import → 422 IDEMPOTENCY_KEY_REUSED» (preview: 24/30 dòng giữa kỳ không đổi, 6 dòng bị ca khác sửa lúc 12:41 UTC; seed commit lại với cùng khoá Idempotency + upload_id mới). `check-assignments-seed.mjs bt03|counts`, `check-exam-seed.mjs bank`, `check-knowledge-seed.mjs` đỏ vì BT03 đã chấm/clone, 10 câu MCQ_MULTI QC thêm, 429 hạn mức
- Kỳ vọng: mã 0, N1 = N2 · Thực tế: mã 1 ở bước 15; lượt chạy tiếp (bước 16–20 trước đó chưa chạy) tạo thêm 43 attendance_checkins, 32 practice_attempts
- Bằng chứng: n02-check-seed.log, n02-seed-rerun1.log, n02-seed-rerun3-diff-course12.txt · Nghi ngờ mã: scripts/seed.mjs:1093-1096 (`importColumn`, khoá cố định theo sha tệp)
- Chạy lại lần 2: ổn định (cùng lỗi 422 ở 2 lượt chạy)

### DEF-R9.5-18 — Hôm nay của Admin trống ngay sau seed
- Mức: Thấp
- Sprint gốc: 9.5 (spec AC3) · TC gốc: TC-N02-06 · Hồi quy do sprint sau?: KHÔNG RÕ
- Bước tái hiện: đăng nhập `admin@`, `GET /me/today` → `{"count":0,"actions":[]}`; teacher 36 việc, ta 19, sv.kha/sv.gioi có `recommended`, sv.moi `JOIN_CODE`
- Kỳ vọng: mỗi vai ≥ 1 việc · Thực tế: Admin 0 (nguồn Admin chỉ phát sinh khi LLM lỗi / ngân sách / dead-letter / lớp không GV) · Nghi ngờ mã: backend-go/internal/today/providers.go:410-432
- Chạy lại lần 2: ổn định

### DEF-R9.5-19 — Seed không từ chối khi thiếu hoặc rỗng `SEED_DEFAULT_PASSWORD`
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N02-09, TC-N02-11 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `env -u SEED_DEFAULT_PASSWORD API_URL=http://127.0.0.1:1/api/v1 node scripts/seed.mjs`; hoặc `SEED_DEFAULT_PASSWORD="   "` → vẫn chạy bước 1 (đọc `.env.local`/`.env.example`)
- Kỳ vọng: mã ≠ 0 «Thiếu SEED_DEFAULT_PASSWORD» trước khi ghi · Thực tế: tiếp tục · Nghi ngờ mã: scripts/seed.mjs:44,56 (`setting()` rơi về tệp env; không `trim()`)
- Bằng chứng: n01-n02-quick.log · Chạy lại lần 2: ổn định

### DEF-R9.5-20 — Redis tắt: chat treo 39,5 s (TL-2d còn mở)
- Mức: Trung bình
- Sprint gốc: 9.5 (TL-2d) · TC gốc: TC-N06-04 · Hồi quy do sprint sau?: KHÔNG (có sẵn, đã ghi ≈ 19 s)
- Môi trường: stack riêng ep-r95 (gateway + redis + 2 worker), DB T1
- Bước tái hiện: `docker stop ep-r95-redis`; SV `POST /chat/sessions/{sid}/messages` → 200 suy giảm sau 39,5 s; `/readyz` 503 (đúng); tạo phiên 503 sau 1,9 s
- Kỳ vọng: ≤ 10 s, thông điệp «đang bận, thử lại» · Thực tế: 39,5 s · Bằng chứng: n06-redis-chat.log · Chạy lại lần 2: chưa lặp (1 lượt)

---
<!-- defects-10.md -->
### DEF-R10-01 — go test ./internal/contract không xanh (TC-12-14)
- Mức: Trung bình
- Sprint gốc: 10 · TC gốc: TC-12-14 · Hồi quy do sprint sau?: KHÔNG RÕ (7 test đỏ nằm ở thread/auth/results, không ở practice; nghi quá tải môi trường hoặc lỗi 500 của sprint 11/10.8)
- Môi trường: máy QC, Colima 10 GB chung với ~40 container ep-*, be0c0d96
- Bước tái hiện: 1. `cd backend-go && go test ./internal/contract/ -run 'TestStudentNeverSeesConfidence|TestChatThreadsContract' -v`  2. so với `go test ./internal/contract/ -run 'TestNoAnswerLeak|TestGolden_Auth'` (xanh).
- Kỳ vọng (TC-12-14): contract test xanh · Thực tế: 500 INTERNAL ở `POST /api/v1/courses/{id}/posts/{id}/report` (scenarios_thread_test.go:247) và `POST /api/v1/auth/verify-email` (scenarios_test.go:389); chạy cả gói: 7 test đỏ.
- Bằng chứng: evidence-10/logs/contract-fail.log, contract-fail2.log, go-tests.log · Nghi mã: backend-go/internal/contract/scenarios_thread_test.go:247, scenarios_test.go:389
- Chạy lại lần 2: trượt (khác endpoint mỗi lần ⇒ có thể chập chờn do tải)

Ghi nhận không phải lỗi sản phẩm: (1) check-practice-seed.mjs đỏ do dữ liệu seed lớp 1 bị ca QC đổi; (2) sập Postgres ep-r lúc 12:59 UTC do nhóm khác nạp T1 vào cùng container; (3) specs `qc-s10-*` gốc giả định lớp mặc định 761987 nhưng sv.kha nay thuộc nhiều lớp (đã khắc phục ở bản sao).

---
<!-- defects-10.5.md -->
### DEF-R10.5-01 — Bảng «Quyết định nhanh» ở /observability nằm trên bảng «7 ngày qua»
- Mức: Thấp
- Sprint gốc: 10.5 · TC gốc: TC-D07-02 (US-DM-07 AC1: «một bảng … dưới bảng 7 ngày») · Hồi quy do sprint sau?: CÓ — sprint 11 viết lại màn «Quan sát AI» (`AdminView.tsx`, các commit 7993ff72 merge wip/11-wave1, 78b0a2be «Section/Page/EmptyState»); ở 10.5 (`ObservabilityScreen.tsx:159`) bảng nằm cuối trang
- Môi trường: ep-r, be0c0d96, ADMIN, 1440 và 375
- Bước tái hiện: 1. Đăng nhập `admin@edupilot.local`. 2. Mở `/observability`. 3. Thứ tự các khối: «Thư chưa gửi được» → «Quyết định nhanh» → «7 ngày qua» → «Chi phí theo tác vụ» → «Yêu cầu». (`frontend/e2e/qc-dm-ui-real.spec.ts`, ảnh `evidence-10.5/shots/observability-admin-1440.png`)
- Kỳ vọng (theo AC): bảng «Quyết định nhanh» dưới bảng 7 ngày  · Thực tế: trên bảng 7 ngày (`frontend/src/features/observability/AdminView.tsx:58` trước `:65`)
- Bằng chứng: `evidence-10.5/pw-dm.log`, `evidence-10.5/shots/observability-admin-1440.png`  · Nghi ngờ mã: `AdminView.tsx:58-65`
- Chạy lại lần 2: ổn định

### Mục TC-D04 (cổng chèn lệnh) — không có lỗi sản phẩm
- TC-D04-01, D04-09 ghi `LỖI THỜI (P-11-26)`: `FAKE_DECISION_PROB` chỉ ghi đè đầu vào NGOÀI `decision-replay.json`; sau P-11-26 replay chứa mẫu chấm của seed (20 FALLBACK + 1 FAST ở P=0,9) nên kịch bản `dm_c.py` (14 cờ mới ở P=0,9) không còn dựng được. Cần dev/PM cho cách điều khiển mới nếu muốn kiểm OR ba tầng.
- TC-D04-02/03 PASS: lỗi decision (circuit_open) → chấm xong 22/22, điểm y hệt giữa cổng tắt / bật / lỗi cùng điều kiện. Chênh điểm so với round 1 trong `dm_c.py` KHÔNG do cổng (cổng tắt cũng chênh y hệt): round 1 chấm lúc route GRADING do nhóm khác đang đổi.
- Lưu ý môi trường: Postgres vào recovery giữa lượt B5a (nhóm khác chaos) làm hỏng một lượt; đã chạy lại ở B6.

---
<!-- defects-10.6.md -->
### DEF-R10.6-01 — reflect / regrade-stale qua lớp có cờ tắt trả 409 GUIDANCE_DISABLED trước 404
- Mức: Thấp
- Sprint gốc: 10.6 · TC gốc: TC-G03-24, TC-G05-24 · Hồi quy do sprint sau?: KHÔNG (có sẵn; summary 10.6 «Ghi nhận (1)» đã nêu, PM chưa đổi)
- Môi trường: ep-r, be0c0d96, vai teacher (GV lớp 761987 và lớp 761988), 1440
- Bước tái hiện: 1. `teacher` đăng nhập; lấy `C2` = id lớp 761988, `A` = id «Bài tập 03» của lớp 761987 (không thuộc lớp 761988). 2. Cờ `GRADING_GUIDANCE` của lớp 761988 đang tắt. 3. `POST /courses/$C2/assignments/$A/guidance/reflect` (Idempotency-Key bất kỳ) và `POST /courses/$C2/assignments/$A/regrade-stale`. (Script: `docs/regression/evidence-10.6/scripts/gd_k.py`, `gd_d.py`)
- Kỳ vọng (theo TC): `404` (bài tập không thuộc lớp)  · Thực tế: `409 GUIDANCE_DISABLED` (kiểm cờ lớp chạy trước kiểm sở hữu bài tập); bật cờ lớp 761988 thì ra 404 (suy luận, chưa chạy)
- Bằng chứng: `docs/regression/evidence-10.5/batchB1.log` (G03-24), `docs/regression/evidence-10.6/gd_d.log` (G05-24)  · Nghi ngờ mã: `internal/httpapi/assignmenthttp/guidance.go` (thứ tự guard cờ → tải bài tập)
- Chạy lại lần 2: ổn định (2/2)

### Ghi nhận không phải lỗi sản phẩm (đã loại)
- TC-G03-22 (reflect `error` / `invalid` → FAILED sau 3 lần): hành vi đổi theo P-11-23 (`llm.AwaitOutage`: mất nhà cung cấp không tiêu lượt, hoãn tới OutageMax) → job FAILED sau ≈ 10 phút, vẫn 0 DRAFT, mã `REFLECT_FAILED`. Ghi `LỖI THỜI (P-11-23)`.
- TC-G01-05 «lớp lạ → 404»: GV/TA/SV không thuộc lớp nhận 403 từ CourseAccessGuard (không đổi từ 10-2026; tc-00 sprint 10: «403 hoặc 404»); ADMIN nhận 404.
- `go test -tags testroutes ./internal/contract` đỏ ở `TestNoAnswerLeak` khi chạy trong shell đã export `QC_*`/`SEED_DEFAULT_PASSWORD`; chạy trong môi trường sạch: PASS (cả gói 57 s). Do môi trường chạy của QC, không phải lỗi.

---
<!-- defects-10.8.md -->
# Lỗi hồi quy sprint 10.8 (tạm, cập nhật dần) — be0c0d96, ep-r, qc2/SmkH

Chưa có lỗi sản phẩm nào được xác nhận (trượt 2 lần). Các lượt F đã gặp và đã quy về môi trường / dữ liệu lặp, chạy lại đều ra P (không phải lỗi):

| TC | Triệu chứng lần đầu | Nguyên nhân | Chạy lại |
| --- | --- | --- | --- |
| TC-AS03-07 | (c) trả `422 CHECKIN_TOKEN_INVALID` thay vì `409 SELF_CHECKIN_DISABLED` | lớp thử còn một cửa sổ đang mở; theo D-AS-06 (US-AS-01 AC5) cửa sổ đang mở vẫn nhận lượt dù tắt chính sách — hành vi đúng | P trên lớp không có cửa sổ mở |
| TC-AS03-18 | `me/attendance` rỗng | chưa đóng cửa sổ nên chưa có `attendance_records` | P sau khi đóng |
| TC-AS04-10 | K2 đầu tiên bị `DEVICE_NEW` ở lượt chạy 2 | dùng lại SV đã có khoá từ lượt chạy 1 | P với SV mới |
| TC-AS10-08b | grep log khớp `10.77` | khớp nhầm mili-giây của dấu thời gian | P với mẫu chính xác (0 dòng) |
| TC-AS12-15, TC-AS12-12…14 | 422 / dòng DB rỗng | token xoay 5 s hết hạn vì stack chậm, hoặc PostgreSQL đang phục hồi sau sự cố (nhóm khác) | P khi chạy lại |
| TC-AS06-06/08 | `/me/today` không có mục | cache Hôm nay 60 s + fixture SQL không xoá cache; mục tính trong hạn 60 ms của provider | P sau chờ hết TTL |
