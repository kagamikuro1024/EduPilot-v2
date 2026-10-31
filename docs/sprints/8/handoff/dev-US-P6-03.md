# DEV handoff — US-P6-03 (trích công thức từ quy chế: `ExtractGradeScheme`)
Nhánh `sprint/8-p6`. Spec: US-P6-03 AC1–AC10, SRS 4.2, 6.1 #1. Proposals: #9 (dòng contract, ACCEPTED), #11 (guard 403 thay 404 — chờ PM).

## Đã làm
- `internal/grading` (gói MỚI, chứa LLM): `schema.go` (JSON schema `strict`-tương thích: mọi object `required` = mọi khoá, `params` là `anyOf` sáu nhánh, số là chuỗi decimal), `prompt.go` (chỉ dẫn hệ thống cố định; văn bản quy chế giữa hai dòng ranh giới, mỗi trang `[Trang N]`), `extract.go` (`Service.Request` kiểm đầu vào → 202; `Service.Run` việc nền `grade_scheme.extract`: đúng MỘT `llm.Structured` mỗi lần thử, task `UTILITY` → làn `NEAR_REALTIME`, `temperature 0`, `ctx` con 45 s / cả việc 90 s; chỉ lỗi tạm (`ErrUnavailable` do nhà cung cấp, `ErrOverloaded`, hết hạn riêng của lần thử) thử lại ≤ 3 lần rồi `LLM_UNAVAILABLE`; đầu ra không đọc được → `EXTRACTION_INVALID`, KHÔNG thử lại, KHÔNG gọi lần hai kèm "nhắc schema"; trích dẫn kiểm bằng MỘT câu SQL trên `content_chunks` (`vn_fold` + NFC + gộp khoảng trắng, không viết lại `vn_fold` ở Go), so trên đầu ra đã gỡ placeholder; lưu một giao dịch với khoá advisory theo lớp: kiểm lại `DRAFT` (sửa chen ngang / `version` khác lúc gọi → `FAILED DRAFT_EXISTS`), thay `DRAFT` chưa sửa, `scheme_no = max + 1`, một item `MANUAL` thang 10 mỗi thành phần, `audit_log`; `extract_job_id` duy nhất nên việc nhận lại không tạo bản đôi).
- `internal/grade/checker.go`: bộ kiểm 4.2.5 (`Check`) — thuần, không LLM; mã `NO_COMPONENTS`, `WEIGHT_SUM` (kèm tổng thật), `BONUS_MISSING`, `BONUS_CAP_MISSING`, `ABSENCE_RULE_MISSING`, `ROUNDING_MISSING`, `LETTER_SCALE_MISSING/INVALID`, `PARAM_INVALID`, `CITATION_NOT_FOUND` + cảnh báo `BONUS_NO_EFFECT`. Để ở `grade` (không ở `grading`) vì `PUT …/grade-scheme` (US-P6-04) cũng chạy nó mà `grade` không được import LLM.
- `internal/httpapi/gradehttp` (handler mỏng): `POST /courses/{id}/grade-scheme/extract` — guard `Teacher`, `Idempotency-Key` bắt buộc, `202 {job_id}`. Nối vào `routes.go` và `cmd/worker/registry.go`.
- `apierr`: 20 mã lỗi mới của sổ điểm (SRS 6.4) + enum trong `openapi.yaml`; route + schema + tag `gradebook` trong `openapi.yaml`.
- Store: `queries/grade_scheme.sql` (đọc tài liệu / đoạn, kiểm trích dẫn, bản nháp CRUD).
- Contract: dòng `want` của route trong `TestSpec_SecurityDeclared` (proposal #9) + `scenarios_grade_test.go` (202 / 401 / 403 / 404 / 409 / 422; test mới).

## Quyết định nhỏ
- Bản ghi của provider giả khoá theo sha256 của yêu cầu chuẩn hoá (gồm văn bản quy chế): nên bản ghi sẵn cho DEMO_MODE được sinh ở US-S8-SEED bằng công cụ ghi âm của `fake` trên đúng PDF seed. Test của story này dùng `Structurer` giả trả fixture.
- `llm_audit` không có cột nội dung (chỉ task, làn, token, trạng thái) → "không lưu toàn văn quy chế vào `llm_audit`" đúng theo cấu trúc; kiểm bằng `TestExtractUsesPrivacyMask` (cổng LLM thật + provider giả: payload tới provider không còn tên / MSSV, mỗi lời gọi một dòng `llm_audit` UTILITY / NEAR_REALTIME).
- Mục `unclear_items` LLM đưa vào được lưu ở `extraction` (source `LLM`); mục `CODE` tính lại mỗi lần kiểm (US-P6-04 `GET`). LLM đưa sai trọng số / luật trùng → giữ phần còn lại + `PARAM_INVALID` tĩnh.
- `TestConfirmOnlyFromHandler` (AC6) làm ở US-P6-04 khi có `ConfirmScheme`; hiện không có đường mã nào xác nhận (`TestExtractNeverConfirms` kiểm DB).

## Kiểm (`go test ./internal/grading ./internal/grade ./internal/contract ./internal/store`)
- AC1: `TestExtractOneStructuredCall`, `TestExtractTaskAndLane`, `TestExtractPromptHasPages`, `TestExtractSchemaStrictCompatible`, `TestExtractAttemptTimeout` · AC2: `TestExtractGradeScheme` · AC3: `TestExtractLop2OneUnclear` · AC4: `TestChecker{WeightSum,NoComponents,Bonus,BonusCap,Absence,Rounding,LetterScale,ParamInvalid}` · AC5: `TestCitation{Verified,NotFound,PageMissing,OCRTolerance,ComparedAfterUnmask}` · AC6: `TestExtractNeverConfirms` · AC7: `TestExtract{NotPolicy,NotReady,TooLong,LLMDown,InvalidSchema,FailureKeepsConfirmed}` · AC8: `TestExtract{IdempotentKey,KillWorkerMidRun,DraftExists,ReplaceDraft,DraftEditedDuringJob,ReplaceDraftVersionChanged}`, `TestExtractIdempotencyHTTP` · AC9: `TestPromptDelimitsDocument`, `TestPromptNoInstructionFollowing`, `TestExtractUsesPrivacyMask` · AC10: `TestExtractRBAC` (403 TA / STUDENT / ADMIN; ngoài lớp **403**, xem #11).
- `golangci-lint` 0 issue ở các gói đụng tới; `TestAllCourseRoutesGuarded` xanh; `go test ./internal/contract` xanh.
- Các lệnh `Kiểm:` của spec có thẻ `-tags integration`: các test này không đặt build tag (tiền lệ `crm`), chạy được với hoặc không có thẻ.

## Cần hỏi / nợ
- **#11** (404 → 403 ngoài lớp) chờ PM.
- Tool nhận `GET …/grade-scheme` (TC-46/47) thuộc US-P6-04.

## Vòng 2 (sau QC FAIL vòng 1 — commit `05633308`, push cùng US-P6-05)
- **BUG-P6-03-1** (`TestNoUnscopedChunkQuery`): `GradeDocForExtract` không còn chạm `content_chunks`; đọc văn bản tài liệu qua `DocChunks` (đường ĐỌC đã có trong danh sách cho phép, lọc `document_id` + `course_ids`) — `grade.DocPages`. Kiểm độ dài: `DocPages(maxChars)` dừng khi vượt trần (`POLICY_TOO_LONG`). `GradeDocPages` bị xoá. Test cũ không đổi.
- **BUG-P6-03-2** (`TestKnowledgeDocExcludedEverywhere`): mọi truy vấn `from documents` trong `grade_scheme.sql` (`GradeDocForExtract`, `GradeStatusFacts`) có `d.category is distinct from 'THREAD_KNOWLEDGE'`.
- **BUG-P6-03-3** (số thao tác OpenAPI): proposals #12 ACCEPTED, số cuối là 228 sau P6-05, 232 sau P6-04.
- `GradeCitationsCheck` nay nhận văn bản trang làm tham số (`vn_fold` vẫn ở SQL, không có `vnFold` ở Go); `grade.VerifyCitations` dùng chung.
- Kiểm: `go test -tags testroutes ./internal/contract ./internal/integration ./internal/rag` (xem dòng kết quả ở handoff US-P6-04).
