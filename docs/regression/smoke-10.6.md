# Smoke hồi quy sprint 10.6 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm G, SmkG) · Ngày: 2026-11-15
Tổng TC cũ: 263 · Đạt: 142 · Trượt: 0 · Không chạy (K, gồm lỗi thời): 121   (P+F+K = 263)
Tách K: K(a/b/c) = 2 (a=0, b=2, c=0) · lỗi thời = 1 · **chưa chạy (mẫu) = 118** (ngoài mẫu smoke «1 chính + 1 lỗi/phân quyền mỗi US»; chờ PM quyết có chạy nốt)

Cách chạy (bản sao script ở `docs/regression/evidence-10.6/scripts/`, sửa cho ep-r): `gd_a`,`gd_c`,`gd_d`,`gd_f`,`gd_g`,`gd_h`,`gd_i`,`gd_j`(mới, DB),`gd_k`(mới, reflect) + `gd_b2` (G03-16/18/22); stub OpenAI-compatible `stub_llm.py` cổng 8778 trỏ route GRADING dưới khoá toàn cục (cửa sổ W1/W2/B1, đã hoàn); `gd_e1/e2` với `GUIDANCE_ENABLED=false` (restart gateway+worker dưới khoá, đã hoàn mặc định); Go: `TestEssayPromptUnchangedAtV0`, `TestGuidance*`, `TestReflect*`, `TestRegress*`, `TestPublishMixedGuidance`, `TestRejectReasonHints`, `TestRosterDiacriticSubset` (+`TestRoster*`,`TestDetect*`) và `go test -tags testroutes ./internal/contract/...` PASS; Playwright `qc-gd-today/ui-real` (9 ca xanh, ảnh `shots/`), `qc2g-class-settings` + `qc-gd-ux/undo-real` KHÔNG ra kết quả hợp lệ (stack bị chaos của nhóm khác + thiếu dữ liệu nháp) → các ca UI ghi «chưa chạy (mẫu)». Dữ liệu BT03 lớp 761987 đã bị đổi (duyệt/công bố/chấm lại) — nhóm khác dùng chung cần lưu ý. Chưa chạy: `GUIDANCE_MIN_RESULTS=abc` (TC-G01-08), `regress run`, `run.sh` E3 (cần nhãn người → K b), migrate up/down/up trên DB trống.

## Tổng hợp theo US
| US | TC tổng | P | F | K(a/b/c)+lỗi thời | chưa chạy (mẫu) | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- |
| A. P-106-11 — tường lửa PII khớp tên roster: «dấu là tập con | 10 | 6 | 0 | 0 | 4 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| B. P-106-12 — cụm vô hại miễn từ khoá cấm | 12 | 10 | 0 | 0 | 2 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| C. P-106-12 — `dropped` theo tiêu chí + dòng «N lưu ý bị loạ | 14 | 1 | 0 | 0 | 13 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| D. UX theo danh mục chuẩn P-106-13 (nhật ký giaovien mô phỏn | 22 | 1 | 0 | 0 | 21 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-01 Cờ theo lớp | 18 | 14 | 0 | 0 | 4 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-02 Nền dữ liệu | 16 | 10 | 0 | 0 | 6 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-03 Rút lưu ý | 25 | 19 | 1 | 1 | 4 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-04 Duyệt / Áp dụng | 30 | 19 | 0 | 0 | 11 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-05 Ghim, chấm lại, chặn lô trộn | 27 | 16 | 1 | 0 | 10 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-06 Số đo và nhãn tiêu chí | 12 | 5 | 0 | 0 | 7 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-07 Ca hồi quy (theo tham chiếu) | 15 | 8 | 0 | 1 | 6 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-08 Câu tương tự lúc soạn (Threads) | 20 | 11 | 0 | 0 | 9 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-09 Lý do loại câu hỏi (mẫu âm) | 12 | 5 | 0 | 0 | 7 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-10 Seed, cổng, đo | 18 | 6 | 0 | 1 | 11 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-GD-11 Ranh giới an toàn (xuyên suốt) | 12 | 9 | 0 | 0 | 3 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |

## Kết quả từng ca (đã chạy: P/F; K gom theo dải)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-P11-01 | P | Go TestRosterDiacriticSubset + TestRoster*/TestDetect* PASS |
| TC-P11-02 | P | Go TestRosterDiacriticSubset + TestRoster*/TestDetect* PASS |
| TC-P11-03 | P | Go PASS + stack Threads: «Vu Minh Long», «vũ minh long», «VŨ MINH LONG» → 422 PII_DETECTED; «Vụ Minh Long» (khác dấu) và 7 cụm dương tính giả không bị chặn |
| TC-P11-04 | P | Go TestRosterDiacriticSubset + TestRoster*/TestDetect* PASS |
| TC-P11-05…TC-P11-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-P11-09 | P | Go TestRosterDiacriticSubset + TestRoster*/TestDetect* PASS |
| TC-P11-10 | P | TestRoster* / TestDetect* PASS (14 test); chỉ thêm test |
| TC-P12-01 | P | Go TestReflectKeywordExempt / TestReflectDroppedByCriterion PASS |
| TC-P12-02 | P | Go TestReflectKeywordExempt / TestReflectDroppedByCriterion PASS |
| TC-P12-03 | P | Go TestReflectKeywordExempt / TestReflectDroppedByCriterion PASS |
| TC-P12-04 | P | Go TestReflectKeywordExempt / TestReflectDroppedByCriterion PASS |
| TC-P12-05 | P | Go TestReflectKeywordExempt / TestReflectDroppedByCriterion PASS |
| TC-P12-06 | P | Go TestReflectKeywordExempt / TestReflectDroppedByCriterion PASS |
| TC-P12-07 | P | Go TestReflectKeywordExempt / TestReflectDroppedByCriterion PASS |
| TC-P12-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-P12-09 | P | stack decide: 3 chữ miễn → 200; 3 chữ cấm → 422 GUIDANCE_TEXT_REJECTED |
| TC-P12-10 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-P12-11 | P | Go TestEssayPromptUnchangedAtV0 (golden v0) PASS |
| TC-P12-12 | P | chữ cấm tính điểm vẫn bị chặn (P12-09); khối lưu ý không có tổng / trừ / làm tròn (G05-27) |
| TC-P12-13 | P | job reflect thật (9router): result = {drafts, dropped:[{criterion_id,reason,count}]}, reason ∈ enum; (validator openapi: chưa chạy) |
| TC-P12-14…TC-P12-26 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-UX-S21…TC-UX-17 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-UX-18 | P | gd_f.py (Hoàn tác = RESET) + gd_d.py + gd_a/c đã chạy API: PASS; qc-gd-undo-real.spec.ts: UI không chạy được (không có nháp lưu ý / stack bị chaos nhóm khác) — phần UI chưa chạy |
| TC-UX-19 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G01-01 | P | gd_a: teacher 4 cờ tắt (can_edit REGRESSION_CAPTURE=false), admin can_edit cả 4 |
| TC-G01-02 | P | teacher bật GRADING_GUIDANCE → 200, version+1, audit_log guidance_flag.set |
| TC-G01-03 | P | teacher PUT REGRESSION_CAPTURE → 403 |
| TC-G01-04 | P | admin PUT REGRESSION_CAPTURE → 200 |
| TC-G01-05 | P | TA/SV 403, không token 401; GV/lớp lạ → 403 (CourseAccessGuard chung từ 10-2026; tc-00 sprint 10 cho phép «403 hoặc 404»); admin lớp lạ → 404 |
| TC-G01-06 | P | TA GET/PUT cờ → 403; SV 403 |
| TC-G01-07 | P | gateway GUIDANCE_ENABLED=false: PUT cờ → 409 GUIDANCE_DISABLED; reflect → 409, 0 job mới (restart dưới khoá, đã hoàn) |
| TC-G01-08 | P | GUIDANCE_MIN_RESULTS=abc: gateway thoát mã 1, log nêu tên biến «GUIDANCE_MIN_RESULTS: cần số nguyên trong 1–1000»; env đã khôi phục (evidence-10.5/minres.log) |
| TC-G01-09 | P | cờ tắt: reflect/apply/regrade-stale → 409 GUIDANCE_DISABLED, overview enabled=false, dữ liệu cũ còn, màn duyệt không khối guidance (phần «prompt không khối» xem G05-26: chưa chạy) |
| TC-G01-10 | P | THREAD_SUGGEST tắt: similar-confirmed → [], 0 lượt nhúng của người gọi |
| TC-G01-11 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G01-12 | P | gd_g G07-04: cờ REGRESSION_CAPTURE tắt → 0 hàng ca mới |
| TC-G01-13 | P | qc-gd-ui + qc2g: 4 nhãn đúng, nhóm «Học từ lần sửa», công tắc «Ca kiểm thử» khoá «Chỉ quản trị viên bật», không câu mô tả, không tràn ngang |
| TC-G01-14…TC-G01-15 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G01-16 | P | version cũ 409; thiếu enabled / sai kiểu 422; cờ lạ 422; không 500 |
| TC-G01-17 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G01-18 | P | SV: 6 phản hồi không có guidance / ai_comments / first_reviewed |
| TC-G02-01 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G02-02 | P | gd_j + truy vấn schema: 4 bảng, 4 cột mới, FK phức hợp course_id, index bắt đầu course_id |
| TC-G02-03 | P | gd_j: retired<=approved, APPROVED thiếu decided_by, reject_reason khi chưa REJECTED đều bị CHECK chặn |
| TC-G02-04 | P | gd_d: chấm lại → kết quả mới ghim guidance_version = bản hiện hành (v3), prompt có khối «bản v3» trong 8/8 lời gọi |
| TC-G02-05…TC-G02-06 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G02-07 | P | gd_j: UPDATE ai_scores / ai_comments / guidance_version của kết quả DONE bị trigger chặn; cột khác vẫn ghi được |
| TC-G02-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G02-09 | P | gd_j: ai_comments {criteria, overall}; kết quả cũ do fake chấm không có ai_comments (NULL, không backfill) |
| TC-G02-10 | P | Go TestGuidanceSetAtVersion PASS |
| TC-G02-11 | P | Go TestGuidanceBlockDeterministic PASS |
| TC-G02-12 | P | Go TestEssayPromptUnchangedAtV0 PASS |
| TC-G02-13 | P | gd_c: TA/SV/ADMIN 403 mọi API guidance; bài tập qua đường lớp 2 → 404 (GET overview) |
| TC-G02-14 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G02-15 | P | git diff 6d6db6f7..b9a3bdd5: test cũ chỉ đổi tên/đếm theo 10.5 (TestInjectionFlagTiers→TestInjectionOR, +«Quyết định nhanh» ở settings-llm), không nới assertion |
| TC-G02-16 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G03-01 | P | bài tập chưa có kết quả duyệt (BT04) → 409 NOT_ENOUGH_DATA, 0 job (biên 4 bài: chưa chạy) |
| TC-G03-02…TC-G03-03 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G03-04 | P | stub rf_ok: 202 → job SUCCEEDED, 2 DRAFT, support do code đếm |
| TC-G03-05 | P | cùng Idempotency-Key → cùng job_id; khoá mới không bài mới → trả job cũ |
| TC-G03-06 | P | duyệt thêm → khoá mới → job mới |
| TC-G03-07 | P | không tự kích: số job reflect = số lần QC bấm |
| TC-G03-08 | P | đầu vào {rubric, cases[], existing}, ca {case_id, criterion_id, ai_score, teacher_score, ai_comment}; 2 lần → case_id khác nhau |
| TC-G03-09 | P | đầu vào không tên / MSSV / email / SĐT / submission_id / student_id (kiểm ≥4 ký tự); Go TestReflectInputNoSubmissionText + TestReflectInputNoIdentity PASS |
| TC-G03-10 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G03-11 | P | DRAFT hợp lệ, support = số ca (G03-04/11/19) |
| TC-G03-12 | P | criterion lạ: schema strict chặn trước → job FAILED, 0 DRAFT, không 500 |
| TC-G03-13 | P | bad_case: bỏ mục (dropped case); wrong_direction: bỏ mục (dropped support) — 0 DRAFT |
| TC-G03-14 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G03-15 | P | từ khoá tổng / trừ muộn / làm tròn: 6/6 bị bỏ (dropped keyword 6) |
| TC-G03-16 | P | stub pii_text với roster thật (tên, MSSV, email, SĐT): 4/4 bị bỏ (dropped pii 4); Go TestReflectOutputPII PASS |
| TC-G03-17 | P | chữ trùng mục đã có: bị bỏ (dropped duplicate 1) |
| TC-G03-18 | P | stub trả nhiều mục: đúng 8 DRAFT (trần) |
| TC-G03-19 | P | DRAFT.support = số id kết quả, source_result_ids không trùng users.id |
| TC-G03-20 | P | llm_audit reflect: task GRADING, lane BATCH, không nội dung |
| TC-G03-21 | P | stub rỗng → job SUCCEEDED, 0 DRAFT |
| TC-G03-22 | K (lỗi thời) | LỖI THỜI (P-11-23): lỗi nhà cung cấp (stub error/invalid) → hoãn không tính lượt (llm.AwaitOutage, OutageMax) nên job FAILED sau ≈10 phút thay vì 3 lần/≈30 s; đã quan sát: FAILED REFLECT_FAILED «Chưa rút được lưu ý. Thử lại.», 0 DRAFT |
| TC-G03-23 | P | 4 lần/giờ/bài tập: lần 4 → 429 RATE_LIMITED |
| TC-G03-24 | P | RERUN 2026-11-16 sau sửa DEF-R10.6-01 (dev, 1c0700e2): TA/SV/ADMIN 403, không token 401, cờ tắt 409 đúng; reflect với bài tập của lớp khác qua đường lớp 2 (cờ tắt) → 404 ASSIGNMENT_NOT_FOUND |
| TC-G03-25 | P | overview: enabled, eligible theo số bài duyệt (lượt đầu 0 bài → eligible=false) |
| TC-G04-01…TC-G04-02 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G04-03 | P | APPROVE → 200 APPROVED chờ áp dụng, audit |
| TC-G04-04 | P | REJECT → 200 REJECTED |
| TC-G04-05 | P | version cũ 409 VERSION_CONFLICT; APPROVE lại 409 ITEM_NOT_DRAFT |
| TC-G04-06 | P | sửa chữ hợp lệ 200 + text_original; «tổng», tên, MSSV, rỗng, >400 → 422 GUIDANCE_TEXT_REJECTED (ghi nhận cũ: 1 ký tự duyệt được) |
| TC-G04-07 | P | apply expected 0 → 200 {version:1, stale}; guidance_version=1; script đòi 2 mục nhưng thực có 3 APPROVED (kể cả mục «x» 1 ký tự) — kỳ vọng script lỗi thời, hành vi đúng |
| TC-G04-08 | P | expected 0 lại → 409 VERSION_CONFLICT; không còn gì → 409 NOTHING_TO_APPLY; thiếu expected_version → 422 |
| TC-G04-09 | P | hai apply đồng thời: [200, 409]; phiên bản không trùng |
| TC-G04-10 | P | cùng Idempotency-Key → trả lại kết quả cũ |
| TC-G04-11 | P | TA/SV/ADMIN decide + apply → 403; TA xem danh sách 200; SV GET 403; lớp 2 → 404 |
| TC-G04-12 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G04-13 | P | RETIRE mục đã áp dụng → pending_retire; apply → v3, RETIRED retired_in_version=3; RETIRE kèm text 422 |
| TC-G04-14…TC-G04-18 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G04-19 | P | gd_f: APPROVE có sửa chữ → 200 |
| TC-G04-20 | P | gd_f: RESET → DRAFT, decided_by NULL, phiên bản không đổi, audit approve+reset |
| TC-G04-21 | P | gd_f: RESET từ REJECTED → DRAFT |
| TC-G04-22 | P | gd_f: sau 6 s RESET vẫn 200 (đặt lại tới khi Áp dụng) |
| TC-G04-23 | P | gd_f: RESET trên DRAFT 200 không đổi; mục đã áp dụng → 409 ITEM_NOT_RESETTABLE |
| TC-G04-24 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G04-25 | P | gd_f: RESET version cũ → 409 VERSION_CONFLICT |
| TC-G04-26 | P | gd_f: giữ chữ đã sửa + text_original = bản AI |
| TC-G04-27…TC-G04-28 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G04-29 | P | gd_f: RESET mục chờ nghỉ → pending_retire=false |
| TC-G04-30 | P | gd_f: đua RESET × Áp dụng nhất quán (reset 200, apply 409) |
| TC-G05-01 | P | gd_d: 8/8 lời gọi chấm lại có khối «Lưu ý của giảng viên (bản v3)» sau RUBRIC, chỉ chữ lưu ý đang hiệu lực (không DRAFT/REJECTED/RETIRED) |
| TC-G05-02 | P | gd_d: 8/8 lời gọi chấm lại có khối «Lưu ý của giảng viên (bản v3)» sau RUBRIC, chỉ chữ lưu ý đang hiệu lực (không DRAFT/REJECTED/RETIRED) |
| TC-G05-03 | P | Go TestRegradeKeepsLatePolicy + TestRegradeNeverOverwritesTeacherScores PASS (phần API stub điểm lệch: chưa chạy) |
| TC-G05-04 | P | khối lưu ý giống hệt giữa các lời gọi cùng bản |
| TC-G05-05 | P | stale.count = số nháp chưa sửa (5), reviewed 8, editing 0 (script đòi 8 theo dữ liệu cũ; số khớp SQL) |
| TC-G05-06 | P | TA chấm lại → 202 queued = số nháp bản cũ; kết quả round+1 DONE ghim bản hiện hành, có ai_comments; chạy lại → queued 0 |
| TC-G05-07 | P | body scope bị bỏ qua (202), REVIEWED/PUBLISHED không bị chấm lại |
| TC-G05-08 | P | bài REVIEWED/PUBLISHED không đổi (điểm, tiêu chí, round, ghim, ai_scores) |
| TC-G05-09…TC-G05-12 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G05-13 | P | như G05-08 (publish) |
| TC-G05-14 | P | công bố bài duyệt bản cũ → 200 không 409; exempt EDITED/ACCEPTED_AS_IS đếm đúng; audit guidance_exempt |
| TC-G05-15 | P | công bố bài nháp → 422 NOT_REVIEWED |
| TC-G05-16 | P | Go TestPublishMixedGuidance PASS |
| TC-G05-17 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G05-18 | P | cờ tắt: công bố 200, exempt 0 |
| TC-G05-19 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G05-20 | P | không round trùng (UNIQUE) |
| TC-G05-21 | P | round cũ giữ nguyên ai_scores / ai_comments / guidance_version |
| TC-G05-22…TC-G05-23 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G05-24 | P | RERUN 2026-11-16 sau DEF-R10.6-01: regrade-stale với bài tập lớp khác qua đường lớp 2 → 404 ASSIGNMENT_NOT_FOUND; SV 403, TA 202 |
| TC-G05-25…TC-G05-26 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G05-27 | P | khối không chứa tổng điểm / trừ / làm tròn / phạt |
| TC-G06-01 | P | stats v0 khớp SQL độc lập (n, MAE tiêu chí, MAE tổng, lệch, tiêu chí sửa, tỉ lệ) |
| TC-G06-02 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G06-03 | P | gd_m: đụng DRAFT, tổng n các phiên bản không đổi |
| TC-G06-04 | P | bias_total = AI − chốt (AI > chốt → dương), khớp SQL |
| TC-G06-05…TC-G06-09 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G06-10 | P | SV stats 403, ADMIN 403, TA 200 (lớp 2 / không token: chưa chạy) |
| TC-G06-11 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G06-12 | P | benchmarks/guidance/run.py đọc teacher_label (cột c1..c4), không đọc expected cho MAE |
| TC-G07-01 | P | lệch ≥1,0 + review ghi 1 hàng GRADING_EDIT; bài thứ hai → 2 hàng |
| TC-G07-02 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G07-03 | P | expected = {ai_score, direction, rubric_hash, teacher_score}: chỉ số / enum / hash (script cũ đòi «không chuỗi ≥40» nhầm hash 64 ký tự — hành vi đúng SRS) |
| TC-G07-04 | P | cờ REGRESSION_CAPTURE tắt → 0 hàng mới |
| TC-G07-05 | P | sửa lại cùng tiêu chí → vẫn 1 hàng |
| TC-G07-06…TC-G07-10 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G07-11 | P | regress export mặc định từ chối, thoát 3 (Go TestRegressCLIRefusesExport PASS) |
| TC-G07-12 | K(b) | regress export thật chờ phép PM («KHÔNG KIỂM ĐƯỢC (chờ phép)») |
| TC-G07-13 | P | stats.regression_cases = SQL (2 = 2); teacher/TA 200, SV/ADMIN 403; GET /regression* → 404 |
| TC-G07-14 | P | gd_m: 0 ca hồi quy mồ côi (source_id không còn kết quả) |
| TC-G07-15 | P | Go TestRegressionCaseNoText + TestRegressionCaseIdempotent PASS |
| TC-G08-01 | P | POST similar-confirmed {q = chữ đã nhúng của thread xác nhận} → có thread đó (≤ 3, DTO {thread_id,title}) |
| TC-G08-02 | P | q < 3 từ → []; q rỗng không 500; q 6.000 ký tự → 422, không 500 |
| TC-G08-03…TC-G08-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G08-09 | P | 20 lần: 10 lượt nhúng (cache), 0 lời gọi sinh chữ |
| TC-G08-10 | P | 35 lần/phút: 28×200 + 7×429; người khác không bị ảnh hưởng |
| TC-G08-11 | P | p95 = 70 ms (20 lượt, embedding giả) |
| TC-G08-12 | P | THREAD_SUGGEST tắt → [] và 0 lượt nhúng của người gọi (lượt chạy đầu lệch do nhóm khác cùng stack; chạy lại riêng: 0) |
| TC-G08-13 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G08-14 | P | 0 hàng pii_events cho chuỗi đang soạn |
| TC-G08-15 | P | chuỗi lạ → [] (không lộ lớp khác) |
| TC-G08-16 | P | SV lớp 1 qua lớp 2 / SV chỉ lớp 2 qua lớp 1 → 403; TA, teacher 200; không token 401 |
| TC-G08-17…TC-G08-18 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G08-19 | P | chữ đang soạn không có ở log gateway / Caddy; Cache-Control: no-store |
| TC-G08-20 | P | route GET cũ → 404 |
| TC-G09-01 | P | 7 lý do lưu đúng; REJECT không lý do → UNSPECIFIED; có lý do (MO_HO) giữ nguyên; audit chỉ mang enum |
| TC-G09-02 | P | lý do lạ 422; reject_reason kèm APPROVE 422; CHECK DB; bulk có lý do lưu cả lô; bulk lý do lạ → ok:false |
| TC-G09-03…TC-G09-07 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G09-08 | P | Go TestRejectReasonHints PASS (phần API GenerateQuestions: chưa chạy) |
| TC-G09-09…TC-G09-10 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G09-11 | P | SV GET questions / gen-stats → 403, không token 401, DTO SV không có reject_reason |
| TC-G09-12 | P | bulk REJECT có / không lý do → UNSPECIFIED cả lô |
| TC-G10-01…TC-G10-06 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G10-07 | P | go test -tags testroutes ./internal/contract/... PASS (57 s, môi trường sạch); scripts/ui-antipatterns.sh exit 0 |
| TC-G10-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G10-09 | P | run.py đọc teacher_label (cột c1..c4); expected S7 = 7,00 còn nguyên (git diff chỉ thêm teacher_label ở expected_grading.json) |
| TC-G10-10 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G10-11 | P | run.py --labels thiếu → WAITING_HUMAN + lý do, thoát 3, không chạy giả |
| TC-G10-12 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G10-13 | K(b) | cần PM giao agent «Giáo viên» / người chấm nhãn E3 (chưa giao) |
| TC-G10-14 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G10-15 | P | docs: D61, D62 trong DECISIONS / ARCHITECTURE; FLOWS, PROGRESS có mục lưu ý chấm |
| TC-G10-16 | P | run.py khi gateway tắt: «Gateway chưa chạy», thoát 2 |
| TC-G10-17 | P | git diff gốc..10.6: test cũ chỉ đổi theo 10.5 / đếm; expected_grading.json chỉ thêm teacher_label, expected S7 = 7,00 giữ nguyên |
| TC-G10-18 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G11-01 | P | Go TestGuidanceVersionOnlyViaApply PASS |
| TC-G11-02 | P | gd_m: assignments.guidance_version không đổi khi đọc / stats |
| TC-G11-03 | P | Go TestGuidanceNoTotalsInPrompt PASS; gd_d: khối không có tổng/trừ/làm tròn; decide chữ cấm → 422 (API chữ tay đủ) |
| TC-G11-04 | P | gd_m: guidance_items + regression_cases không chứa tên SV; đầu vào reflect đã kiểm (G03-09) |
| TC-G11-05 | P | Go TestGuidanceIsolation PASS (cô lập hai bài tập qua API: chưa chạy) |
| TC-G11-06 | P | gd_m: SV quét 37 phản hồi: 0 trường lưu ý / phiên bản / cờ / số đo |
| TC-G11-07…TC-G11-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-G11-09 | P | git diff gốc..10.6: go.mod / go.sum / package.json không đổi; không thư viện bộ nhớ agent |
| TC-G11-10 | P | audit_log có từng quyết định: approve, edit_approve, reject, reset, retire, apply, guidance_flag.set, question.review |
| TC-G11-11 | P | log gateway / worker / caddy: 0 gói tới googleapis; LLM_DECISION_ALLOW_REAL=false |
| TC-G11-12 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |

## Trượt: bước tái hiện
### DEF-R10.6-01 — ĐÃ SỬA, rerun 2026-11-16: 404 ASSIGNMENT_NOT_FOUND trước 409 (TC-G03-24, TC-G05-24 đạt)

## Ca K và lý do (bảng gom)
| Mã loại | Ca | Lý do |
| --- | --- | --- |
| KL | TC-G03-22 | LỖI THỜI (P-11-23): lỗi nhà cung cấp (stub error/invalid) → hoãn không tính lượt (llm.AwaitOutage, OutageMax) nên job FAILED sau ≈10 phút thay vì 3 lần/≈30 s; đã quan sát: FAILED REFLECT_FAILED «Chưa rút được lưu ý. Thử lại.», 0 DRAFT |
| Kb | TC-G07-12 | regress export thật chờ phép PM («KHÔNG KIỂM ĐƯỢC (chờ phép)») |
| Kb | TC-G10-13 | cần PM giao agent «Giáo viên» / người chấm nhãn E3 (chưa giao) |

## Chưa chạy (mẫu)
118 ca: dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có script; liệt kê theo dải ở bảng trên (cột «chưa chạy (mẫu)»).

## Lượt 2026-11-18 — luật chủ dự án: ca «chặn» phải chạy tới kết luận thật (mã dc67bd93, stack ep-qc)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-G07-12 | N | «chờ phép PM» đã hết hiệu lực (lệnh 11-18): chạy regress export thật |
| TC-G10-13 | K(b) | cần chủ dự án / người chấm nhãn E3 (gán nhãn) — giữ K(b) theo lệnh 11-18 |
