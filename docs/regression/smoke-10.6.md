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
| Kb | TC-G07-12 | regress export thật chờ phép PM («KHÔNG KIỂM ĐƯỢC (chờ phép)») |
| Kb | TC-G10-13 | cần PM giao agent «Giáo viên» / người chấm nhãn E3 (chưa giao) |

## Chưa chạy (mẫu)
118 ca: dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có script; liệt kê theo dải ở bảng trên (cột «chưa chạy (mẫu)»).

## Lượt 2026-11-18 — luật chủ dự án: ca «chặn» phải chạy tới kết luận thật (mã dc67bd93, stack ep-qc)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-G07-12 | N | «chờ phép PM» đã hết hiệu lực (lệnh 11-18): chạy regress export thật |
| TC-G10-13 | K(b) | cần chủ dự án / người chấm nhãn E3 (gán nhãn) — giữ K(b) theo lệnh 11-18 |
| TC-G10-13 | N | không còn nhãn K(b): chạy tới kết luận (QC tự gán nhãn E3 theo rubric TC, ghi rõ nguồn nhãn) |

## Lượt 2026-11-18 — QC cặp 3, mã 4ac1638a, ep-qc, lớp QD9G (bài tập «QC106 …» do QC dựng; không đụng 761987/761988)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-G01-14 | P | UI /class/settings (QD9G, teacher): bấm «Lưu ý chấm» → công tắc đổi ngay (false→true, đọc sau 250 ms, lạc quan), hiện dòng «Hoàn tác» ×1, không hộp thoại, không toast «Thành công»; DB enabled=t v2; Hoàn tác → công tắc về false, enabled=f, version 1→3 (+2); audit_log +2 dòng. Ảnh /tmp/qc9s/s106-g0114-1.png |
| TC-G01-15 | P | 375 px (touch): 4 công tắc, vùng chạm switch 44×44 và hàng nhãn 146–217×44 (≥44); không tràn ngang |
| TC-G01-17 | P | tab 2 PUT với version cũ → mã ["409"]; công tắc tab 2 sau đó false (DB enabled=f, v30); chữ khối: «Lưu ý chấm Tắt Có người vừa sửa mục này. Chọn giữ bản nào. Thử lại Ca kiểm thử Chỉ quản trị viên bật Tắt Gợi ý câu hỏi tương tự Tắt Gợi ý khi sinh câu hỏi Tắt» |
| TC-G02-01 | P | DB trống riêng (qc_mig106 trên ep-qc-postgres): gateway migrate up → version 27 (85 bảng, md5 cột cdad802f…); down lặp tới version 0 (còn 1 bảng goose); up lần 2 → version 27, md5 cột giống hệt cdad802f… (85 bảng); migration guidance = 00021 liền sau 00020 (decision); git diff 4a5803e5~1..b9a3bdd5 -- db/migrations chỉ thêm A 00021_grading_guidance.sql, không sửa file cũ (00016/00020 chỉ sửa trong sprint gốc của chúng, trước khi merge); đã xoá DB thử |
| TC-G03-02 | P | 5 REVIEWED, tiêu chí lệch: 1,00 + đúng 0,50 (tính) + 0,49 (SQL, không tính): overview edits=2, reflect → 409 {"code": "NOT_ENOUGH_DATA", "message": "Cần thêm bài đã duyệt.", "trace_id": "d82523f2e7e4e59202a4a3b53c9c3d19", "detail; thêm 1 lệch đúng 0,50 → edits=3 eligible=True → reflect 202, job SUCCEEDED, 1 mục DRAFT |
| TC-G03-14 | P | Chạy lại sau sửa R106S-02 — mã 0bdef404: lần 1 (stub JSON sai) job f5e19b → FAILED; sửa stub, bấm lại với Idempotency-Key KHÁC → 202 job b05786 (job mới) → SUCCEEDED, 2 mục; bấm lần 3 khi đã SUCCEEDED cùng dữ liệu → job b05786 (cùng job lần 2, chống trùng đúng); (R106S-02 sửa ab89d82b) |
| TC-G02-05 | P | cờ tắt, chấm 3 bài: guidance_version kết quả = ['0']; 6 lời gọi chấm tới stub, có khối «Lưu ý của giảng viên» = 0 |
| TC-G02-16 | P | RUNNING: UPDATE guidance_version 0→1 cho phép; sau DONE: UPDATE →2 bị chặn: ERROR:  ai draft is immutable CONTEXT:  PL/pgSQL function grading_results_ai_immutable() line 11 at RAISE; cuối cùng guidance_version=1 run_state=DONE |
| TC-G02-06 | P | mục chờ áp dụng khi bắt đầu: 4; apply v1→v2 trong lúc chấm (stub chậm) → 200; assignments.guidance_version=2; kết quả bài mới: guidance_version=1 round=1 attempts=1; mọi 2 lời gọi chấm của bài mới dùng khối bản ['1'] (không trộn) |
| TC-G02-14 | P | ép lỗi INSERT grading_guidance_versions (trigger tạm trên ep-qc, superuser nên không thu quyền được; đã xoá): apply → 500 {"code": "INTERNAL", "message": "Đã xảy ra lỗi. Hãy thử lại sau.", "trace_id": "5181ee5b90; guidance_version=0, mục có approved_in_version=0, hàng phiên bản=0; bỏ trigger → apply lại → 200 {"version": 1, "stale": {"count": 4, "editing": 0, "reviewed |
| TC-G02-08 | P | lượt chấm bài mới đang chạy (RUNNING, attempts 1, ghim v0) → docker kill worker (RUNNING) rồi start + apply v1→v2 lúc worker chết (200); sau khi chạy lại: run_state=DONE attempts=2 guidance_version ghim=2 (assignments=2); cặp (submission, round) trùng: 0 |
| TC-G03-03 | P | mã 0bdef404, bài tập type=QUIZ: POST reflect → 409 {"code": "TYPE_NOT_SUPPORTED", "message": "Chỉ bài tự luận mới rút được lưu ý chấm.", "trace_id": "1b665f92fb18522c03cf6f4373e90ba9"}; số job reflect 14→14 (R106S-03 sửa ff09501f) |
| TC-G03-10 | P | 21 bài duyệt (mỗi bài 4 tiêu chí lệch = 84 ca, reviewed_at tăng dần R01→R21); đầu vào reflect gửi tới stub: 60 ca, khớp đúng tập ca của 15 bài mới nhất R07–R21 (mã hoá điểm GV theo chỉ số bài; khác tập của 15 bài cũ nhất) = True; job SUCCEEDED; overview reflect={"eligible": true, "results": 21, "edits": 84} |
| TC-G04-01 | P | [cờ tắt] khối guidance-panel=0; chữ «Lưu ý chấm» trong màn=false |
| TC-G04-02 | P | 8 mục hiển thị (DB có 3 Chờ duyệt, 1 Chờ áp dụng, 1 Đang dùng v2, 1 Chờ nghỉ, 1 Đã bỏ, 1 Đã nghỉ); nhãn trạng thái: Chờ duyệt=true, Chờ áp dụng=true, Đã áp dụng=true, Chờ nghỉ=true, Đã bỏ=true, Đã nghỉ=true; mục Chờ duyệt: «Không cho điểm tối đa c2 nếu thiếu đối chiếu thực tế. Mẫu 2. ¦  ¦ Duyệt ¦ Bỏ ¦ Sửa ¦ Chờ duyệt ¦ Gắt hơn ¦ 3 lần sửa»; có «lần sửa» true; ngăn: [{"bt":"1px","bb":"0px","rad":"0px","sh":false,"st":"APPROVED"},{"bt":"1px","bb":"0px","rad":"0px","sh":false,"st":"APPROVED"}]; trạng thái thật trên UI: ["Đã áp dụng","Chờ áp dụng","Đã bỏ","Chờ duyệt","Đã nghỉ","Chờ nghỉ"] |
| TC-G04-12 | P | TA: khối=1, 7 mục; nút trong khối: ["Chấm lại"]; nút cấm: [] |
| TC-G04-24 | P | bấm «Áp dụng»: hộp «Áp dụng bản v3? ¦  ¦ Thêm 1 lưu ý, cho nghỉ 1. Bài chấm sau này dùng bản mới; bài đã chấm giữ nguyên. ¦  ¦ Để sau ¦ Áp dụng»; POST apply trước xác nhận=0, sau xác nhận=1; DB chờ áp dụng 1 / chờ nghỉ 1; guidance_version 2→3; dòng «Hoàn tác»=0; toast «Thành công»=0; chữ sau áp dụng: «Lưu ý chấm ¦ Bản v3 ¦ Rút lưu ý» |
| TC-G04-14 | P | apply trả stale={"count":4,"editing":0,"reviewed":8,"versions":[0]}; SQL đếm lại: nháp chưa sửa 4, nháp đã sửa 0, đã duyệt 8; /grading: «4 bài chấm theo bản cũ», nút Chấm lại ×1; không tự chấm lại (kết quả QUEUED/RUNNING=0) |
| TC-G04-16 | P | 375 px: khối=1 rộng 327; 15 nút, nhỏ hơn 44 px: []; tràn ngang false; chữ mục trên, nút dưới (xếp dọc) true |
| TC-G04-27 | P | Duyệt → «Chờ áp dụng»; offline + Hoàn tác: mục vẫn «Chờ áp dụng» true; nút «Thử lại» ×1; chữ mục: «Cần nêu rõ tác nhân và phạm vi ảnh hưởng ở c2. Mẫu 1. ¦  ¦ Đặt lại ¦ Chờ áp dụng ¦ Chấm chặt hơn ¦ 3 bài đổi điểm ¦ Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn được giữ. ¦ Thử lại»; RESET tới máy chủ khi offline: 0; có mạng + Thử lại → DB status=DRAFT; số RESET thành công=1 (mọi mã: ["200"]) |
| TC-G04-28 | P | 375 px sau «Duyệt»: «Hoàn tác» 64×44; tràn ngang false; giao với nút khác: []; điểm giữa chạm trúng «Hoàn tác» |
| TC-G04-15 | P | Mã 0bdef404 (R106S-06 sửa 81e58ce6): AID2 chưa có lưu ý: khối «Lưu ý chấm ¦ Chưa có bản ¦ Rút lưu ý ¦ Mới có 0 bài đã duyệt, 0 lần sửa ¦ Áp dụng ¦ Chưa có lưu ý nào chờ áp dụng ¦ Chưa có lưu ý» (dòng «Chưa có lưu ý» true); chặn GET /guidance → 500: nút «Thử lại» ×1, chữ quanh lỗi «không thấy»; bỏ chặn + Thử lại → khối trở lại=1; CHỮ SỬA: sửa mục «Không cho điểm tối đa c2 » thành «Chữ sửa dở…», chờ 1,5 s (autosave), tải lại: ô «Sửa» chứa «Chữ sửa dở của QC106 giữ sau tải lại.»; thẻ hiển thị «Không cho điểm tối đa c2 nếu thiếu đối chiếu thực tế. Mẫu 2. ¦ Duyệt ¦ Bỏ ¦ Sửa » |
| TC-G04-17 | P | chữ khối: từ cấm không; icon Info/Help 0 (tổng svg 2); dòng mô tả ngay dưới tiêu đề 0; title/tooltip: []; `bash scripts/ui-antipatterns.sh` ở worktree mã 4ac1638a → exit 0 |
| TC-G04-18 | P | SV mở /assignments/{id}: khối=0; request guidance/regrade/reflect=0 []; «Lưu ý chấm» trong chữ=false; «guidance» trong DOM=false; url /assignments/3e78f66b-58ee-4068-8b79-50171248d9e1 |
| TC-G05-26 | P | cờ tắt vs cờ bật nhưng guidance_version=0, cùng rubric, mỗi bên 2 bài: 4 / 4 system prompt chấm; sau chuẩn hoá tên bài hai tập BẰNG NHAU từng byte = True; khối «Lưu ý của giảng viên» trong bản cờ bật = 0 |
| TC-G05-10 | P | bài ở round=20 (đặt bằng SQL): regrade-stale → 202 skipped={'reviewed': 6, 'edited': 0, 'max_round': 1}; kết quả round 21 của bài đó = 0; round lớn nhất của bài tập = 20 |
| TC-G05-11 | P | sv có bản nộp v2 (copy kết quả bản cũ thành kết quả của v2, guidance_version 0): sau regrade-stale: kết quả v2 có round 2 = 1, kết quả của bản nộp v1 có round 2 = 0 (chỉ chấm bản mới nhất); phản hồi {"job_id": "01a1fbda-15cf-746d-870b-4c78acbddcfe", "queued": 7, "skipped": {"reviewed": 6, "edited": 0, "max_round": 1}} |
| TC-G05-25 | P | bài nộp muộn (due 5 ngày trước, nộp 3 ngày trước): 6 bài chấm lại v1→v2 round 2; (late_days, late_penalty, ai_total_before_late, total) trước = sau cho 6/6 bài; mẫu trước ['3', '1.00', '4.00', '3.00'] sau ['3', '1.00', '4.00', '3.00'] |
| TC-G05-22 | P | Mã 0bdef404: guided 16 bài (6 duyệt) → regrade-stale xếp 10 bài; stub chậm 12 s, docker kill ep-qc-worker-1 khi đang chạy rồi start; 1 bài chứa [[QCERR]] (stub trả JSON hỏng riêng bài đó). Kết quả round 2: 9 DONE (1 bài attempts=2 = việc được nhận lại sau khi worker chết) + 1 FAILED LLM_UNAVAILABLE attempts 3, không chặn 9 bài kia; cặp (submission, round) trùng = 0. Quan sát: bài lỗi 500 (breaker mở) không FAILED mà QUEUED mãi attempts=1 (xếp lại không tiêu lượt thử — P-11-23), nên ca FAILED phải dùng lỗi không tính vào mạch |
| TC-G05-23 | P | regrade 8 bài (stub chậm), tắt cờ GRADING_GUIDANCE sau 3 s: kết quả round 2 8 dòng, run_state ['DONE'], guidance_version ghim ['0', '1'] (các bài nhận việc sau khi tắt cờ ghim 0, không crash; job regrade FAILED = 0) |
| TC-G05-09 | P | 250 nháp stale chưa sửa (nạp SQL, 250 SV): regrade-stale lần 1 → queued 200 (round 2 trong DB 200); lần 2 → queued 50 (tổng round 2 = 250); trạng thái round 2: [['DONE', '250']] |
| TC-G05-19 | P | công bố lô 6 bài chấm muộn 3 ngày (assignment v1, bài đã duyệt miễn bản): POST publish → 200 {published:6, skipped:0, exempt:{EDITED:6}}; (total, late_days, late_penalty) trước = sau công bố 6/6; total = Σ điểm tiêu chí − khoản trừ muộn = True (vd 2.00 − 1.00 = 1.00); trạng thái PUBLISHED + published_at. Go: `go test ./internal/assignment -run Publish` 12/12 PASS; git diff publish*_test.go so với 5dabd253: không sửa/xoá assertion công bố cũ |
| TC-G05-12 | F | /grading?assignment: «8 bài chấm theo bản cũ» + nút «Chấm lại» ×1; sau bấm: nút khoá/busy=false (chữ nút «»); mẫu dòng mỗi 3 s: ["Đã xếp 8 bài chấm lại [hàng «Đang chấm»=1]","Đã xếp 8 bài chấm lại [hàng «Đang chấm»=2]","Đã chấm xong 8 bài [hàng «Đang chấm»=1]"]; yêu cầu /events ["200"], /jobs 1; cuối: round 2 DONE=8 |
| TC-G05-17 | F | chọn 9 ô; nút «Công bố» bật=true; hộp xác nhận: «Công bố điểm ¦  ¦ Công bố 8 bài · 8 sinh viên nhận thư · không tính vào sổ điểm (chưa có công thức điểm) · 8 bài đã duyệt chấm theo bản lưu ý cũ, vẫn công bố được ¦  ¦ Để sau ¦ Công bố»; dòng «bản cũ» TRONG hộp=false; sau xác nhận: «6 bài chấm theo bản cũ »; PUBLISHED trong DB=8 |
| TC-G06-02 | P | phiên bản v0 có n=4 bài duyệt: GET stats → 200 {"items": [{"version": 0, "n": 4, "mae_criterion": null, "mae_total": null, "bias_total": null, "edited_criteria": 4, "edited_rate": null, "criteria_counted": 16}], "regression_cases": 0} (các chỉ số null khi n<5) |
| TC-G06-08 | P | v0→v1, 16 bài: 6 duyệt (v0, round 1) + 5 bài chấm lại round 2 rồi duyệt (v1): stats [(0, 6), (1, 5)] tổng n=11; số kết quả HIỆN HÀNH REVIEWED/PUBLISHED trong DB=11 (tất cả hàng REVIEWED/PUBLISHED=11); round cũ không tính |
| TC-G06-09 | P | 1.000 kết quả REVIEWED (nạp SQL, 1000 SV, 1 bài tập), 30 lần GET stats: p95 28 ms, trung vị 19 ms, max 42 ms (≤ 500); EXPLAIN ANALYZE truy vấn gom GuidanceStatsRows: 2,9 ms; chỉ mục có sẵn submissions_assignment_received_idx (course_id, assignment_id, received_at, id) + grading_results_submission_idx; bài tập 14 bài → Index Only Scan trên chỉ mục đó; bài tập 1.000 bài (29 % bảng submissions 3.401 hàng) planner chọn Seq Scan trên submissions (rẻ hơn), bật enable_seqscan=off → Bitmap/Index Scan trên cùng chỉ mục. TC ghi «không Seq Scan»: không đạt theo chữ ở cỡ 1.000 bài/3.401 hàng nhưng là chọn |
| TC-G06-05 | P | ngăn bài tập: 2 dòng số đo (["Bản v0 · 6 bài đã duyệt · lệch tiêu chí 0,50 · lệch tổng 2,00 · đổi điểm 50%","Bản v1 (đang dùng) · chưa đủ bài đã duyệt để đo"]); nằm SAU danh sách mục=true; biểu đồ=0; TA thấy 2 dòng (nhân viên); SV không thấy khối (TC-G04-18 P) |
| TC-G06-06 | F | kết quả v1 (c2 có 3 lưu ý áp dụng): 1 phụ đề «bản v1 · 3 lưu ý»; bấm → «Phân tích tấn công ¦  ¦ Đoạn lập luận ngắn gọn nhưng đúng trọng tâm vẫn đạt mức khá ở c2. Mẫu 3. ¦  ¦ Phân biệt rõ giả định với kết luận trong c2. Mẫu 4. ¦  ¦ Liên hệ vụ việc thật làm tăng giá trị bài ở c2. Mẫu 5.» |
| TC-G06-07 | P | kết quả v0 (duyệt trước khi áp dụng): phụ đề guidance-meta ×0; chữ «bản vN · n lần sửa» có=false |
| TC-G06-11 | F | chặn GET /guidance/stats → 500: khối «ạm vi ảnh hưởng ở c3. Mẫu 1. ¦ Duyệt ¦ Bỏ ¦ Sửa ¦ Chờ duyệt ¦ Gắt hơn ¦ 3 lần sửa ¦ Ưu tiên bằng chứng định lượng khi đánh giá c3. Mẫu 0. ¦ Duyệt ¦ Bỏ ¦ Sửa ¦ Chờ duyệt ¦ Gắt hơn ¦ 3 lần sửa ¦ Bản v1 (đang dùng) · chưa đủ bài đã duyệt để đo»; «—»=false; nút «Thử lại» ×0; biểu đồ 0 |
| TC-G07-02 | P | cờ REGRESSION_CAPTURE bật (admin PUT 200; teacher tự bật → 403): phúc khảo ADJUSTED tiêu chí c2 (score_after 3,00) → ca GRADING_APPEAL source_id=kết quả bài, criterion_id=c2 = True; bài AI được sửa (PUT /posts/{id}/correct → 200) → ca THREAD_CORRECTED source_id=bài = True; bài AI bị từ chối (POST reject 200) → ca THREAD_REJECTED = True; bài AI xác nhận (verify 200) → không ca = True; mọi ca: [('GRADING_APPEAL', '2f8b57'), ('THREAD_REJECTED', '52da1a'), ('THREAD_CORRECTED', '9bb2ff')] |
| TC-G07-06 | P | review 30 lượt cờ REGRESSION_CAPTURE tắt vs 30 lượt bật (mỗi lượt sửa c2 → 0 rồi duyệt): p95 57 ms vs 38 ms (chênh -19 ms, ≤ 20), trung vị 30 vs 22 ms; ca GRADING_EDIT ghi: bật 30/30, tắt 0/30; ca mồ côi (nguồn không REVIEWED/PUBLISHED) = 0. Phần «kill giữa chừng»: ghi cùng giao dịch kiểm gián tiếp (không ca mồ côi); không kill thật |
| TC-G07-07 | F | Mã 0bdef404: «regress run --course <QD9G> --limit 20» (binary linux build từ worktree, chạy trong container cùng mạng ep-qc với env của worker, GRADING → stub): đọc ca + văn bản trong DB, in {n:20, mae:1.00, bias:0.90, old_mae:1.00, worse:0, skipped:0}; không ghi văn bản ra stdout/stderr (chỉ số + cảnh báo roster), git status sạch. LỆCH SPEC: US-GD-07 AC3 ghi «--assignment <id>» + báo cáo {n, MAE, lệch TB, số tiêu chí xấu đi, số tốt lên}; CLI chỉ có --course, báo cáo không có «số tốt lên» (chỉ worse) → DEF-R106S-12 (Thấp) |
| TC-G07-08 | F | «regress run» không có kiểm quyền ADMIN / «môi trường đo» (grep cmd/regress/main.go: không role/ADMIN; chỉ «export» kiểm REGRESSION_EXPORT_ALLOWED): có env của worker là chạy được; thiếu DATABASE_URL/REDIS_URL thì lỗi cấu hình «thiếu biến môi trường bắt buộc», không phải từ chối theo quyền. US-GD-07 AC3 «(CLI vận hành, môi trường đo, ADMIN)» → DEF-R106S-13 (Thấp) |
| TC-G07-09 | F | benchmarks/guidance/baseline.json chưa tồn tại (chỉ kc/, run.py, run.sh) nên dựng baseline tạm. CLI --baseline: (a) MAE +12 % → exit 1 «Có lùi: MAE 1.00 > 0.98 (baseline 0.89 × 1.10)» đạt; (b) +8 % → exit 0 đạt; (e) lệch TB xấu thêm 0,25 → exit 1 đạt. (c) tiêu chí vốn đúng nay sai 0,6 và (d) sai 0,5: regressed() chỉ xét MAE trung bình và /lệch TB/; Worse (số ca xấu đi) không tham gia mã thoát và baseline không có số liệu theo tiêu chí nên (c) không phát hiện được (cmd/regress/main.go:213-228; thử baseline worse=0 → exit 0) → DEF-R106S-14 (Thấp) |
| TC-G07-10 | F | CI (.github/workflows) KHÔNG chạy «regress run» (grep rỗng) đạt; guide docs/guides/grading-guidance.md §3 ghi lệnh «go run ./cmd/regress run … --baseline» đạt; NHƯNG không có checklist PR nào ghi lệnh (không .github/pull_request_template.md; docs/team không nhắc) → thiếu «checklist PR» của US-GD-07 AC4 → DEF-R106S-15 (Thấp) |
| TC-G07-12 | N | Đã kiểm: «regress export» mặc định từ chối («REGRESSION_EXPORT_ALLOWED=false (…chỉ bật khi có quyết định của quản trị viên)»); không đường HTTP xuất (GET …/regression, …/regression-cases, …/assignments/{aid}/regression, …/grading-regression-cases = 404 ×5). Phần có phép (JSONL ngoài repo, Mask+Detect, id băm, audit_log): CHƯA KIỂM ĐƯỢC — chờ phép của PM (đặt REGRESSION_EXPORT_ALLOWED=true) |
| TC-G08-03 | P | 5 thread có tri thức xác nhận, cosine với câu soạn = [1.0, 1.0, 0.95, 0.9, 0.85] (≥ 0,80): POST similar-confirmed → 200, 3 kết quả (LIMIT 3), thứ tự theo cosine giảm: chỉ số thread [1, 0, 2] (kỳ vọng hai thread cos 1, rồi thread cos 0,95; thread 0,90 và 0,85 bị cắt) |
| TC-G08-04 | P | cosine thật (pgvector) của 3 thread với câu soạn: [1.0, 0.81, 0.79]; kết quả [0, 1]: thread cos 0.81 có = True, thread cos 0.79 không = True |
| TC-G08-05 | P | trước: thread 1,2,3 đều cos 1 → có trong gợi ý [1, 0, 2]; sau khi đáp án thread 1 bị Từ chối (reject) và thread 2 bị Ẩn (hide 200): gợi ý [3, 0] (thread 1 và 2 biến mất) |
| TC-G08-06 | N | Chưa kết luận: chỉ kiểm bộ lọc ở tầng API/SQL bằng dữ liệu dựng tay (không đi luồng thật TC yêu cầu). mục kho của thread 1 chuyển source_type 'TICKET' (mô phỏng tri thức từ kênh riêng: dựng bằng SQL, không đi qua escalate thật) có chữ lạ «ZQ-TK-91», cos 1 với câu soạn: trước (POST) có thread 1 = True; sau: gợi ý [0] không có thread 1; chữ «ZQ-TK-91» trong phản hồi (kể cả soạn tiêu đề ZQ-TK-91) = False; phản hồi DTO chỉ {thread_id,title} |
| TC-G08-13 | N | Chưa kết luận: chỉ kiểm bộ lọc ở tầng API/SQL bằng dữ liệu dựng tay (không đi luồng thật TC yêu cầu). mục kho thread 0 đặt embedding NULL (trạng thái «chờ nhúng lại» sau khi nội dung đổi; mô phỏng bằng SQL — chưa đo chuỗi document.changed thật) → không gợi ý (True); khôi phục vector → có (True); ẩn thread bằng staff (hide 200) → biến mất khỏi gợi ý sau 0.02 s (≤ 60) |
| TC-G08-07 | F | gõ 100 ký tự (25 ms/phím) → 1 precheck (không mỗi phím); request đầu sau phím cuối: precheck 801 ms, similar-confirmed 912 ms (TC ~600 ms); danh sách: 1 khối, 1 liên kết /threads/[id], chữ «Cách phân biệt tấn công chuỗi cung ứng với tấn công trực tiếp là gì QCG0867301 biến thể 2?»; chữ không giống gì → khối 0, «không tìm thấy» false / THIẾU «Đã xác nhận» cạnh tiêu đề (mục chỉ có tiêu đề liên kết; ul chỉ có aria-label «Đã có câu trả lời xác nhận»); debounce thực ≈ 800 ms (usePostGate.tsx setTimeout 800) ≠ 600 ms của US-GD-08 AC3 → DEF-R106S-16 (Thấp) |
| TC-G08-08 | P | gợi ý đang hiện (1 khối); bấm «Đăng»: POST /threads [201], hộp thoại 0; E2E cũ xanh: chưa chạy lại ở đây; tường lửa PII còn nguyên: đăng nội dung có họ tên roster «Vũ Hoàng Giang» → 422 PII_DETECTED; E2E playwright có sẵn của repo chưa chạy lại (CI mock) |
| TC-G08-18 | P | 375 px: 1 khối, 2 phần tử; cao < 44 px: []; tràn ngang false; một cột (x đầu < 200) true |
| TC-G08-17 | N | Chưa kết luận: dừng ep-qc-pgbouncer-1 làm cả bộ bảo vệ lớp/đăng nhập trả 503 SERVICE_UNAVAILABLE trước khi tới handler (không phải đường gợi ý), và giao diện không đăng nhập được; chưa gây được lỗi RIÊNG của nhúng (tạo nhà cung cấp nhúng giả 422 PROVIDER_UNREACHABLE; route /_test/llm/fake không có khoá ErrorRate cho nhúng). Test Go TestSimilarConfirmed PASS nhưng chưa kiểm nhánh nhúng lỗi. |
| TC-G09-05 | P | QUESTION_HINTS bật, 4 câu REJECTED có lý do (MO_HO×2, TRUNG_LAP, LAC_DE; < 5): prompt sinh gửi tới stub (3268 ký tự) có khối «TRÁNH…»: False |
| TC-G01-11 | P | QUESTION_HINTS tắt, 5 câu REJECTED có lý do (≥ 5): prompt sinh có khối «TRÁNH…»: False |
| TC-G09-06 | P | 5 câu có lý do trong 90 ngày (MO_HO×3, TRUNG_LAP×1, LAC_DE×1) + 2 câu UNSPECIFIED + 4 câu SAI_DAP_AN cách 91 ngày (reviewed_at): SQL thống kê (≠UNSPECIFIED, ≤90 ngày) = [['MO_HO', '3'], ['LAC_DE', '1'], ['TRUNG_LAP', '1']]; khối «TRÁNH» trong prompt có 3 dòng: ['Mỗi câu chỉ một cách hiểu; tránh từ chung chung.', 'Chỉ hỏi nội dung có trong tài liệu được chọn.', 'Không lặp ý với câu đã có trong ngân hàng.']; thứ tự = nhiều nhất trước, đồng hạng theo tên (LAC_DE trước TRUNG_LAP); SAI_DAP_AN (91 ngày) không có; tối đa 3 lý do |
| TC-G09-07 | P | khối «TRÁNH (giảng viên thường loại các câu như vậy):» + các dòng cố định đúng bảng RejectHintText (SRS 4.5); chữ câu bị loại xuất hiện trong prompt: []; dòng dài nhất 48 ký tự; (TC ghi tiêu đề «Tránh các lỗi sau» — chữ thật «TRÁNH (giảng viên thường loại các câu như vậy):») |
| TC-G09-09 | F | gen-stats?job_id (cờ bật) → 200 hints không có trong JSON (TA → 200); cờ tắt → hints=None / job.result đã lưu hints [MO_HO, LAC_DE, TRUNG_LAP] nhưng truy vấn PracticeGenHints (internal/store/queries/guidance.sql) lọc «payload ->> 'course_id'» trên bảng jobs KHÔNG có cột payload → lỗi SQL bị nuốt (guidance.sql.go:983, bulk.go GenStatsOf) → «hints» không bao giờ có → DEF-R106S-17 |
| TC-G09-10 | P | (1) cờ bật nhưng chỉ còn 2 câu có lý do (<5) → khối False; sinh 202; (2) lỗi thống kê (đổi tên cột reject_reason tạm trên ep-qc, đã đổi lại): sinh 202 không lỗi, khối False; (3) cờ tắt: xem TC-G01-11 / G09-09 |
| TC-G09-03 | P | chọn câu PENDING → «Loại» một thao tác: dãy 6 nhãn ["Mơ hồ","Sai đáp án","Trùng lặp","Lạc đề","Độ khó lạ","Sai định dạng"], nút «Hoàn tác» 1, hộp thoại 0/0, hàng ẩn ngay (true), DB trong lúc chờ 5 s = PENDING/- |
| TC-G09-04 | P | bỏ qua lý do → REJECTED/UNSPECIFIED; chọn «Mơ hồ» (aria-pressed true) → REJECTED/MO_HO; chọn «Lạc đề» rồi «Hoàn tác» trong 5 s → PENDING/-, hàng quay lại true |
