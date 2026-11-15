# Smoke hồi quy sprint 8 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm D: SmkD + S8Core/S8Flow/S8Seed) · Ngày: 2026-11-15
Tổng TC cũ: 512 · Đạt: 290 · Trượt: 6 · Lỗi thời (LT): 0 · Không chạy K (a/b/c): 0 · Chưa chạy (N): 216   (P+F+LT+K+N = T)

Bằng chứng chung: `p6-hand-calc.py` 30/30 + 18/18 khớp; `go test ./internal/grade` 5 test lõi PASS (logs/). Ghi chú từng ca: các file `part-*.tsv`, script/log ở `docs/regression/evidence-8/`.

## Tổng hợp theo US
| US | TC tổng | P | F | LT | K | N |
|---|---|---|---|---|---|---|
| P6-01 | 41 | 40 | 1 | 0 | 0 | 0 |
| P6-02 | 49 | 46 | 2 | 0 | 0 | 1 |
| P6-03 | 48 | 17 | 0 | 0 | 0 | 31 |
| P6-04 | 49 | 25 | 0 | 0 | 0 | 24 |
| P6-05 | 46 | 19 | 0 | 0 | 0 | 27 |
| P6-06 | 63 | 42 | 2 | 0 | 0 | 19 |
| P6-07 | 59 | 51 | 1 | 0 | 0 | 7 |
| P6-08 | 47 | 42 | 0 | 0 | 0 | 5 |
| P6-09 | 50 | 8 | 0 | 0 | 0 | 42 |
| P6-SEED | 37 | 0 | 0 | 0 | 0 | 37 |
| S8-UI | 23 | 0 | 0 | 0 | 0 | 23 |

## Kết quả từng ca
| TC | KQ | Ghi chú |
|---|---|---|
| TC-P6-01-01 | P | goose up→down(1)→up→status trên DB trống qcd8probe: cả 3 rc=0, 00016_gradebook Applied (logs/s8core-goose.log) |
| TC-P6-01-02 | P | grep ALTER TABLE trong 00016_gradebook.sql: 0 / 0 (cả Up lẫn Down) |
| TC-P6-01-03 | P | information_schema khớp SRS 5.1: version/unlock cols, không published ở grade_entries, numeric(5,2), PK uuid (gradebooks PK course_id uuid) (logs/s8core-sc |
| TC-P6-01-04 | P | docs/PROGRESS.md:98 có dòng '00016 | gradebook' |
| TC-P6-01-05 | P | goose down-to 15 (gỡ 00016+sau): 0 bảng, 0 enum grade*/gradebook_status; up lại sạch (không 'already exists') |
| TC-P6-01-06 | P | probe SQL: 2 DRAFT 23505; 1 CONFIRMED+nhiều SUPERSEDED ok; 2 CONFIRMED 23505 |
| TC-P6-01-07 | P | probe: weight -0,01/100,01 → 23514; 0 và 100 ok |
| TC-P6-01-08 | P | probe: trùng (scheme,key) → 23505 |
| TC-P6-01-09 | P | probe: EXAM thiếu exam_id / MANUAL có exam_id / max_score 0,-1 → 23514; EXAM hợp lệ ok |
| TC-P6-01-10 | P | probe: score -0,01 → 23514; 0 ok; trùng (item,student) → 23505 |
| TC-P6-01-11 | P | probe: 4 ca chéo lớp đều 23503 (FK phức hợp) |
| TC-P6-01-12 | P | 21 chỉ mục không-PK: ngoại lệ đều nằm trong SRS 5.1; cả 7 bảng course_id NOT NULL |
| TC-P6-01-13 | P | TestSchemaGradebook/OneDraftOneConfirmed/CrossCourseFK/Checks/IndexesStartWithCourseID PASS; sqlc generate + sqlc diff rc=0 rỗng |
| TC-P6-01-14 | P | app role=owner (superuser): UPDATE/DELETE/TRUNCATE → P0001 'snapshot is immutable'; số dòng không đổi (rollback) |
| TC-P6-01-15 | P | 2 trigger: final_grade_snapshots_immutable_row (row, BEFORE) + _truncate (statement, tgtype&32) |
| TC-P6-01-16 | P | probe: trùng (course,student,version) 23505; version mới ok; UPDATE bản mới P0001 |
| TC-P6-01-17 | P | 10.000 dòng: Index Scan using final_grade_snapshots_version_idx (200 dòng planner chọn Seq Scan vì bảng nhỏ — như báo cáo gốc) |
| TC-P6-01-18 | P | vai trò không phải chủ: 'must be owner'; vai trò gateway=edupilot (superuser, chủ bảng) gỡ được trigger → ghi RỦI RO (TC cho phép, không FAIL AC) |
| TC-P6-01-19 | P | TestSnapshotImmutable/TruncateBlocked/UniqueVersion/VersionIndex Go integration -tags integration (logs/s8core-go-*.log) PASS |
| TC-P6-01-20 | F | weekly A 10, B 8, C 0 và 24 dòng MCQ khớp CSV (đọc qua explain), nhưng giải trình C không có dòng 'không dự → 0 theo quy chế' (DEF-R8-s8core-01) |
| TC-P6-01-21 | P | bài thi chuyển CLOSED: cả lớp INCOMPLETE (30/30), missing ['weekly'], không có số 0 bịa; giải trình: [{'label': 'weekly', 'expr': '', 'value': 'Bài thi chư |
| TC-P6-01-22 | P | sửa adjusted_score B 8→7: giải trình ô weekly đổi ngay 8→7; grade_entries nguồn EXAM = 0 |
| TC-P6-01-23 | P | quy thang: item 20, điểm thi 8,0 → ô 16 (expr '16 × 10 ÷ 20') → điểm QT 8; item 30, 7,33 → 21,99 ('21,99 × 10 ÷ 30') → 7,33; decimal không sai số |
| TC-P6-01-24 | P | TestExamScorePublishedOnly/FollowsAdjust/MissingIsNotZero/Scale/MatchesSeedCSV PASS; test có require.Equal(24 dòng MCQ) |
| TC-P6-01-25 | P | SQL P5: A cộng 3,00 vắng 0 · B 0,50 vắng 2 · C 0,25 vắng 5 (khớp expected_crm); giải trình dùng đúng (min(0,5; 1); max(0; 5 − 2) × 0,5 + 0) |
| TC-P6-01-26 | P | B thêm 1 EXCUSED + 1 LATE: vắng vẫn 2 ('max(0; 2 − 2) × 0,5 + 0'), P=0 |
| TC-P6-01-27 | P | sv30 enrollment REMOVED: gradebook 29 hàng, preview students 29 (không tính sv30) |
| TC-P6-01-28 | P | grep study_time_daily|student_risk|student_observations ở mã không-test internal/grade: rỗng |
| TC-P6-01-29 | P | TestInputsFromP5/ExcusedLateNotAbsent/ActiveOnly/MatchExpectedCRM Go integration -tags integration (logs/s8core-go-*.log) PASS |
| TC-P6-01-30 | P | study_time_daily C = 600 phút/ngày (14 ngày × 3 khu vực, SQL tay): preview 30 SV & giải trình C không đổi (HP 3 · F); không dòng nào nhắc thời gian học; đã |
| TC-P6-01-31 | P | lớp 2 gradebook 200: 24 dòng (đều thuộc lớp 2), trạng thái NO_POLICY (chưa công thức), không điểm lớp 1 |
| TC-P6-01-32 | P | IDOR: item lớp 1 qua đường dẫn lớp có công thức (QD8A1): PUT grade-entries 404, publish 404; audit_log lớp không tăng 4→4 |
| TC-P6-01-33 | P | Go TestExamFromOtherCourseRejected PASS + probe SQL 11b (item lớp 2→exam lớp 1) 23503; API 422 chưa chạy riêng |
| TC-P6-01-34 | P | sv30 rời lớp rồi chốt lại: snapshot v1 của sv30 còn nguyên ['7.50', 'B']; v2 chỉ 29 SV, không có dòng v2 cho sv30 |
| TC-P6-01-35 | P | TestCrossCourseIsolation/ExamFromOtherCourseRejected/LeftStudentExcluded Go integration -tags integration (logs/s8core-go-*.log) PASS |
| TC-P6-01-36 | P | ma trận 20 route × 5 vai: 100 phép thử, 0 lệch |
| TC-P6-01-37 | P | GV lớp khác: GET gradebook/PUT entries/POST finalize → [403, 403, 403] |
| TC-P6-01-38 | P | TestAllCourseRoutesGuarded PASS (internal/integration); openapi: 45 dòng khớp, đúng 20 operation grade/gradebook/me-grades (≥19) |
| TC-P6-01-39 | P | TestRBACMatrix Go integration -tags integration (logs/s8core-go-*.log) PASS |
| TC-P6-01-40 | P | ngoài lớp / lớp không tồn tại → [403, 403, 403, 403] (cùng mã) |
| TC-P6-01-41 | P | id không phải uuid → [404, 404] |
| TC-P6-02-01 | P | p6-hand-calc.py rc=0: 30/30 + 18/18 khớp; các bước 10 dòng khớp bảng tay (logs/s8core-handcalc.log) |
| TC-P6-02-02 | P | TestEngineExpected30 -race: PASS, đúng 30 ca con, 0 SKIP, 0 DATA RACE (logs/s8core-go2.log) |
| TC-P6-02-03 | P | preview lớp sao QD8A1 (đầu vào sạch = seed): 30/30 khớp qt, hp_raw, hp, letter, verdict theo MSSV; lớp 1 gốc lệch 8 do dữ liệu bị ghi đè (xem DEF/ghi chú) |
| TC-P6-02-04 | N | chưa chạy |
| TC-P6-02-05 | P | M1 nửa-chẵn: TestEngineExpected30 sv04/08/10/26/30, TestEngineCases S10/S11/S16, TestRoundHalfUpTable đỏ (sv05 không đổi vì 6,75→7,0 cả hai cách) |
| TC-P6-02-06 | P | M2 trần cộng 1,5: TestEngineCases/S03 + TestEngineExpected30/sv.gioi đỏ |
| TC-P6-02-07 | P | M3 miễn 3 buổi: sv.nguyco, S05, S07, S08 đỏ |
| TC-P6-02-08 | P | M4 làm tròn trung gian QT: 23 ca sv + S10/S11 + TestNoIntermediateRounding đỏ |
| TC-P6-02-09 | P | M5 liệt CK<=3,0: sv23, sv25, TestFinalExactlyThreeNotFail đỏ |
| TC-P6-02-10 | P | TestEngineCases -race: PASS, đúng 18 ca con S01–S18, 0 SKIP |
| TC-P6-02-11 | P | TestRoundHalfUpTable/NoIntermediateRounding/DivRoundNoDrift PASS; ca x,25 x,75 x,5 bước 0,5/0,1/1 (M1 làm đỏ) |
| TC-P6-02-12 | P | impact step 0.1: samples 9,8 / 6.5 / 3,2 khớp hand-calc step=0.1; step 1 khớp hand-calc step=1 (6,5→7) |
| TC-P6-02-13 | P | TestDivRoundNoDrift + TestEngineCases/S16-chia-khong-het PASS (hp_raw 6,25 → 6,5) |
| TC-P6-02-14 | P | grep float/math.Round/ParseFloat/RoundBank/StringFixedBank + Sscanf/%f ở mã không-test internal/grade: rỗng (rc grep=1) |
| TC-P6-02-15 | F | quét float64: internal/seedtool/examhistory.go:130 đưa điểm thi qua float64(c)*2.5 (seed tool); mã sản phẩm sạch |
| TC-P6-02-16 | P | preview: số JSON chỉ ở ['A', 'A+', 'B', 'B+', 'C', 'C+', 'D', 'D+', 'F', 'complete', 'incomplete', 'liet', 'students']; điểm đều là chuỗi decimal (8 giá tr |
| TC-P6-02-17 | P | go vet ./internal/grade/... ./internal/grading/... rc=0; golangci-lint run ./internal/grade/...: 0 issues; TestNoFloatInGrade PASS (logs/s8core-go2.log) |
| TC-P6-02-18 | P | M6 thêm float64: TestNoFloatInGrade đỏ; grep cổng P6 có dòng (cổng đỏ) |
| TC-P6-02-19 | P | go list -deps ./internal/grade | grep activity|crm|llm|grading|privacy|agent|openai-go: rỗng (rc=1) |
| TC-P6-02-20 | P | TestForbiddenImports/TestEngineIsPure PASS; grep study_time_daily rỗng; Compute(s Scheme, in Input) không context |
| TC-P6-02-21 | P | M7 import activity: TestForbiddenImports đỏ (lần chạy lại riêng; lần đầu bị kill rc=143 do tải) |
| TC-P6-02-22 | P | impact với cap=1,5 free=1 (không sửa mã giữa hai lần gọi): 3 mẫu khớp hand-calc --table cap=1.5 free=1; changed=0 |
| TC-P6-02-23 | P | TestParamsDriveResult/MissingParamIsError/NoPolicyConstants PASS; grep hằng: chỉ chú thích, 7*3600 múi giờ, roundingSteps (tập bước SRS 4.2.4) |
| TC-P6-02-24 | P | xoá PENALTY.params.free: PUT 200 + unclear ABSENCE_RULE_MISSING (chặn xác nhận); impact students_computable=0, samples rỗng — không HP mặc định; (impact kh |
| TC-P6-02-25 | F | C: KT=0, status COMPLETE, nhưng dòng KT chỉ expr '0' (filled_by_user, không quote điều 1.4), không có chữ 'không dự → 0 theo quy chế' (DEF-R8-s8core-01) |
| TC-P6-02-26 | P | xoá ô final sv30 (PUT score null 200): preview status INCOMPLETE, không hp/letter ({'status': 'INCOMPLETE', 'hp': None, 'letter': None, 'verdict': 'INCOMPL |
| TC-P6-02-27 | P | exam CLOSED: preview cả lớp INCOMPLETE (30/30); finalize lần chạy lại → 409 GRADES_INCOMPLETE (details missing weekly). Lần đầu trả 200: nghi tick tự công  |
| TC-P6-02-28 | P | TestMissingScoreIncomplete/ZeroRule/ZeroOnlyWhenSourceClosed/ExpectedStudentC PASS |
| TC-P6-02-29 | P | sv20 hp 5,0 F LIET:final<3.0; sv23 D 4,0 OK; sv25 D 4,0 (hp_raw 3.75) không liệt |
| TC-P6-02-30 | P | giải trình sv20 có dòng liệt với quote nguyên văn điều 4.1: 'Không đạt' |
| TC-P6-02-31 | P | TestFailConditions/TestFinalExactlyThreeNotFail PASS |
| TC-P6-02-32 | P | B: thứ tự dòng đúng (thành phần→cộng→trừ→QT→CK→HP thô→làm tròn→chữ); QT₀ 7,75 → +0,50 → 8,25 → HP thô 7,2 → 7 → B |
| TC-P6-02-33 | P | mọi expr của B, C, sv09 tính lại bằng Fraction ra đúng value; HP thô B 7,2 · C 3,2 · sv09 8,066 (không làm tròn giữa chừng) |
| TC-P6-02-34 | P | 10 SV ngẫu nhiên: mọi số trong explanation CSV có trong lines[]; 181 quote của 30 SV đều có thật (grep -F) trong quy-che-lop1.md; dòng B không quote = số h |
| TC-P6-02-35 | P | TestExplainOrderAndQuotes/ExprRecomputesValue/FullPrecision/UserFilledHasNoQuote PASS |
| TC-P6-02-36 | P | TestWhatIfNoWrite/MinFinalFor/PreviewUsesCompute PASS; BenchmarkCompute1000 = 27,9 ms/op (≤200 ms) trên máy QC (tải cao) |
| TC-P6-02-37 | P | what-if B: CK 8,0 → HP 8 B+; target A → min CK 8.25; A+ → 9.09; audit_log/grade_entries không đổi (script đếm trước/sau) |
| TC-P6-02-38 | P | what-if C target A+ → {"reachable": false} |
| TC-P6-02-39 | P | chỉ có 1 hàm tính điểm Compute(engine.go:115); Preview (gradebook.go:646) gọi nó; TestPreviewUsesCompute PASS |
| TC-P6-02-40 | P | TestZeroWeightGroupNoDivide/BonusNoEffectWarning/PenaltyNotApplicable/PenaltyNotInSchemeWarning/BonusNotApplicable PASS |
| TC-P6-02-41 | P | w_QT=0: 200, HP = CK (3 mẫu ['10', '6.5', '3']), không NaN/Infinity/chia 0; ghi chú: impact không trả cảnh báo BONUS_NO_EFFECT (có ở TestBonusNoEffectWarni |
| TC-P6-02-42 | P | w_CK=0 + luật FINAL_BELOW: PUT 200, unclear_items ['PARAM_INVALID'] (mục chưa rõ, chặn xác nhận) — impact trả tính bình thường, mục chưa rõ nằm ở bản nháp |
| TC-P6-02-43 | P | INCOMPLETE không hp: xem 02-26/02-27 (P) |
| TC-P6-02-44 | P | lỗi rõ không mặc định: xem 02-23/02-24 (P) |
| TC-P6-02-45 | P | weight 'abc','-5','101','',1e999 → [('abc', 422), ('-5', 422), ('101', 422), ('', 422), ('inf', 400)]; không 5xx, panic trong log gateway 0→0 |
| TC-P6-02-46 | P | STUDENT preview/explain người khác/impact → 403 403 403 |
| TC-P6-02-47 | P | TA preview 200, finalize 403 |
| TC-P6-02-48 | P | ADMIN preview 403 |
| TC-P6-02-49 | P | TestPreviewRBAC PASS (Go integration -tags integration, logs/s8core-go-*.log) |
| TC-P6-03-01 | N | chưa chạy |
| TC-P6-03-02 | N | chưa chạy |
| TC-P6-03-03 | N | chưa chạy |
| TC-P6-03-04 | P | grep: 3 file ngoài internal/llm chỉ chứa tên enum 'openai anthropic' trong validate/oneof (llmhttp, llmconfig); không SDK/endpoint |
| TC-P6-03-05 | P | TestExtractOneStructuredCall/TaskAndLane/PromptHasPages/SchemaStrictCompatible/AttemptTimeout PASS |
| TC-P6-03-06 | N | chưa chạy |
| TC-P6-03-07 | N | chưa chạy |
| TC-P6-03-08 | N | chưa chạy |
| TC-P6-03-09 | P | TestExtractGradeScheme PASS |
| TC-P6-03-10 | N | chưa chạy |
| TC-P6-03-11 | N | chưa chạy |
| TC-P6-03-12 | N | chưa chạy |
| TC-P6-03-13 | N | chưa chạy |
| TC-P6-03-14 | P | PDF quy-che-lop2 (pypdf, 2 trang): 0 lần làm tròn/quy tròn/bước điểm; md có 1 dòng là chú thích HTML nói về chính điều đó |
| TC-P6-03-15 | P | TestExtractLop2OneUnclear PASS |
| TC-P6-03-16 | P | 8 test TestChecker* PASS (WeightSum, NoComponents, Bonus, BonusCap, Absence, Rounding, LetterScale, ParamInvalid) |
| TC-P6-03-17 | N | chưa chạy |
| TC-P6-03-18 | N | chưa chạy |
| TC-P6-03-19 | N | chưa chạy |
| TC-P6-03-20 | N | chưa chạy |
| TC-P6-03-21 | N | chưa chạy |
| TC-P6-03-22 | N | chưa chạy |
| TC-P6-03-23 | N | chưa chạy |
| TC-P6-03-24 | P | TestCitationVerified/NotFound/PageMissing/OCRTolerance/ComparedAfterUnmask PASS; không có hàm vnFold tự viết ở Go (vn_fold dùng ở SQL/textfold) |
| TC-P6-03-25 | N | chưa chạy |
| TC-P6-03-26 | P | ConfirmScheme: định nghĩa confirm.go:31 + 1 lời gọi gradehttp/handler.go:202; không ở job/seed/agent |
| TC-P6-03-27 | P | TestExtractNeverConfirms (grading) + TestConfirmOnlyFromHandler (grade) PASS |
| TC-P6-03-28 | P | seed.mjs xác nhận qua POST …/grade-scheme/confirm bằng token TEACHER; không UPDATE grade_schemes. (seed-t1.mjs là seed tải DB riêng edupilot_t1, INSERT thẳ |
| TC-P6-03-29 | P | TestPromptDelimitsDocument/NoInstructionFollowing/ExtractUsesPrivacyMask PASS |
| TC-P6-03-30 | N | chưa chạy |
| TC-P6-03-31 | N | chưa chạy |
| TC-P6-03-32 | N | chưa chạy |
| TC-P6-03-33 | N | chưa chạy |
| TC-P6-03-34 | N | chưa chạy |
| TC-P6-03-35 | N | chưa chạy |
| TC-P6-03-36 | N | chưa chạy |
| TC-P6-03-37 | N | chưa chạy |
| TC-P6-03-38 | N | chưa chạy |
| TC-P6-03-39 | N | chưa chạy |
| TC-P6-03-40 | N | chưa chạy |
| TC-P6-03-41 | N | chưa chạy |
| TC-P6-03-42 | N | chưa chạy |
| TC-P6-03-43 | P | 12 test TestExtract* (NotPolicy…ReplaceDraftVersionChanged) PASS (Go integration -tags integration, logs/s8core-go-*.log) |
| TC-P6-03-44 | P | TA/SV/ADMIN extract → [403, 403, 403]; llm_audit 3649→3649 không đổi |
| TC-P6-03-45 | P | GV lớp khác extract lớp 1 → 403 |
| TC-P6-03-46 | P | TA GET grade-scheme 200, PUT 403 |
| TC-P6-03-47 | P | STUDENT GET grade-scheme → 403 |
| TC-P6-03-48 | P | TestExtractRBAC PASS |
| TC-P6-04-01 | N | chưa chạy |
| TC-P6-04-02 | N | chưa chạy |
| TC-P6-04-03 | N | chưa chạy |
| TC-P6-04-04 | N | chưa chạy |
| TC-P6-04-05 | N | chưa chạy |
| TC-P6-04-06 | N | chưa chạy |
| TC-P6-04-07 | N | chưa chạy |
| TC-P6-04-08 | P | confirm khi còn mục chưa rõ → 422 SCHEME_HAS_UNCLEAR details {"codes": ["WEIGHT_SUM"], "unclear_items": [{"source": "CODE; status vẫn DRAFT |
| TC-P6-04-09 | P | TestConfirmBlockedByUnclear PASS + e2e 'unclear blocks confirm' xanh (13 test dev Playwright gradebo) |
| TC-P6-04-10 | P | PUT rời ô: version 9→10, unclear_items recomputed []=[], ROUNDING filled_by_user=true, không trích dẫn (UI ghi 'do giảng viên đặt'; dòng mạng ở UI-TC) |
| TC-P6-04-11 | P | trọng số abc/-5/101 → [422, 422, 422]; 12,5 → 422 VALIDATION_FAILED; tổng 110 → unclear WEIGHT_SUM kèm tổng 110 (giữ chữ: UI, xem spec) |
| TC-P6-04-12 | P | PUT với version cũ → 409 VERSION_CONFLICT kèm giá trị hiện tại (True) |
| TC-P6-04-13 | N | chưa chạy |
| TC-P6-04-14 | P | TestSchemePutRecheck/VersionConflict/OnlyDraft/FilledByUserNoCitation PASS + e2e 'scheme edit optimistic' xanh |
| TC-P6-04-15 | P | 3 mẫu: cao nhất 10→9,8 · giữa 6.5→6.5 (vị trí 15/16 theo HP) · thấp nhất 3→3,2; giữa khớp hand-calc (đo ≤300 ms: xem UI/TC-04-15 ghi chú) |
| TC-P6-04-16 | P | impact 2 lần: audit_log lớp 4→4, grade_entries 60→60 (không ghi); N=21 khớp tính tay; khoá ['changed_vs_confirmed', 'entries_carried', 'entries_dropped', ' |
| TC-P6-04-17 | N | chưa chạy |
| TC-P6-04-18 | P | TestImpactThreeSamples/NoWrite/NoCompleteStudents PASS + e2e 'impact preview' (+ 'empty') xanh |
| TC-P6-04-19 | N | chưa chạy |
| TC-P6-04-20 | N | chưa chạy |
| TC-P6-04-21 | P | confirm 200: DRAFT→CONFIRMED(confirmed_by), cũ→SUPERSEDED [['SUPERSEDED', '1'], ['CONFIRMED', '2']]; audit_log confirm có số liệu {"version": 10, "scheme_n |
| TC-P6-04-22 | N | chưa chạy |
| TC-P6-04-23 | N | chưa chạy |
| TC-P6-04-24 | P | TestConfirmFlow/Idempotent/Audit/RBAC PASS + e2e 'confirm irreversible' xanh |
| TC-P6-04-25 | P | confirm giữ khoá/thang/nguồn: 60 điểm CHUYỂN sang item mới (midterm 30, final 30; item cũ SUPERSEDED còn trong DB, 0 điểm); /gradebook 30/30 đủ điểm; entri |
| TC-P6-04-26 | P | midterm max 10→100: impact/confirm entries_dropped=[midterm: 30, MAX_CHANGED]; điểm KHÔNG sang item thang 100 (mới 0 điểm), 30 điểm còn ở item cũ SUPERSEDE |
| TC-P6-04-27 | P | sổ LOCKED: confirm → 409 GRADEBOOK_LOCKED; DRAFT không đổi [['DRAFT', '2']] |
| TC-P6-04-28 | P | TestConfirmCarryOverEntries/NoCarryOnMaxScoreChange/NoCarryOnSourceChange/SupersedeBeforeConfirm/CreatesGradebookRow/BlockedWhenLocked PASS |
| TC-P6-04-29 | N | chưa chạy |
| TC-P6-04-30 | N | chưa chạy |
| TC-P6-04-31 | N | chưa chạy |
| TC-P6-04-32 | P | TestNewPolicyMakesNewDraft/DiffAgainstConfirmed/ConfirmedUnchangedUntilConfirm PASS + e2e 'new policy diff' xanh |
| TC-P6-04-33 | P | confirm với version cũ → 409 VERSION_CONFLICT; status vẫn DRAFT; sau đó confirm version mới (bước kế) → 200 |
| TC-P6-04-34 | N | chưa chạy |
| TC-P6-04-35 | N | chưa chạy |
| TC-P6-04-36 | P | TestSchemeEditRace/TestConfirmStaleVersion PASS + e2e 'scheme offline retry' xanh |
| TC-P6-04-37 | P | grep mock/gradebook frontend/src rỗng; frontend/src/mock/gradebook.ts không tồn tại |
| TC-P6-04-38 | N | chưa chạy |
| TC-P6-04-39 | N | chưa chạy |
| TC-P6-04-40 | P | e2e 'scheme states' + 'scheme mobile' xanh |
| TC-P6-04-41 | N | chưa chạy |
| TC-P6-04-42 | P | confirm thiếu Idempotency-Key → 422 IDEMPOTENCY_KEY_REQUIRED; DRAFT không đổi |
| TC-P6-04-43 | P | confirm thân {} thiếu version → 422 VALIDATION_FAILED |
| TC-P6-04-44 | N | chưa chạy |
| TC-P6-04-45 | N | chưa chạy |
| TC-P6-04-46 | N | chưa chạy |
| TC-P6-04-47 | P | GV lớp khác GET grade-scheme → 403 |
| TC-P6-04-48 | P | e2e 'scheme rbac' xanh |
| TC-P6-04-49 | P | ADMIN GET grade-scheme → 403 |
| TC-P6-05-01 | P | L2={"state": "NO_POLICY", "unclear_count": 0, "has_policy": false, "pending_draft": false, "gradebook": {"status": "OPEN", "snapshot_version": 0}, "fi |
| TC-P6-05-02 | P | doc=READY status={"state": "NOT_EXTRACTED", "unclear_count": 0, "has_policy": true, "pending_draft": false, "gradebook": {"stat |
| TC-P6-05-03 | N | chưa chạy |
| TC-P6-05-04 | P | status={"state": "DRAFT_READY", "unclear_count": 0, "has_policy": true, "pending_draft": true, "gradebook": {"status": "OPEN", "snapshot_ |
| TC-P6-05-05 | N | chưa chạy |
| TC-P6-05-06 | N | chưa chạy |
| TC-P6-05-07 | P | SV/ADM/TA=[403, 403, 200] |
| TC-P6-05-08 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-05-09 | P | QD8F2 NO_POLICY: 1 banner 'Công thức điểm chưa xác nhận' + liên kết 'Tải quy chế' (Xem công thức, Thiết lập công thức); shot s8flow-us05-nopolicy-gradebook-teac |
| TC-P6-05-10 | P | TA /gradebook & /attendance lớp NO_POLICY: banner cùng chữ, liên kết 'Xem công thức' (không 'Tải quy chế', không 'Chốt điểm') |
| TC-P6-05-11 | P | QD8F2 DRAFT_READY: banner + liên kết 'Xác nhận công thức' (TEACHER) |
| TC-P6-05-12 | P | TEACHER lớp 1 CONFIRMED: 0 banner (gradebook, attendance); STUDENT: 0 banner ở /me,/attendance,/gradebook,/chat |
| TC-P6-05-13 | N | chưa chạy |
| TC-P6-05-14 | P | dev e2e ['banner unconfirmed'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-05-15 | P | Chốt điểm disabled+aria-disabled=true; tooltip 'Chưa có công thức điểm đã xác nhận' khi hover và focus; 0 icon Info |
| TC-P6-05-16 | N | chưa chạy |
| TC-P6-05-17 | N | chưa chạy |
| TC-P6-05-18 | P | dev e2e ['finalize locked reason'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-05-19 | P | /attendance QD8F2: banner 'Công thức điểm chưa xác nhận'+'Xem công thức' phía trên lưới 1 dòng; lưu thao tác xem TC-20 |
| TC-P6-05-20 | N | chưa chạy |
| TC-P6-05-21 | N | chưa chạy |
| TC-P6-05-22 | P | dev e2e ['attendance notice'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-05-23 | N | chưa chạy |
| TC-P6-05-24 | N | chưa chạy |
| TC-P6-05-25 | N | chưa chạy |
| TC-P6-05-26 | N | chưa chạy |
| TC-P6-05-27 | N | chưa chạy |
| TC-P6-05-28 | N | chưa chạy |
| TC-P6-05-29 | P | go test 2 test PASS (log s8flow-go-*.log) |
| TC-P6-05-30 | N | chưa chạy |
| TC-P6-05-31 | N | chưa chạy |
| TC-P6-05-32 | N | chưa chạy |
| TC-P6-05-33 | N | chưa chạy |
| TC-P6-05-34 | N | chưa chạy |
| TC-P6-05-35 | N | chưa chạy |
| TC-P6-05-36 | P | go test 5 test PASS (log s8flow-go-*.log) |
| TC-P6-05-37 | N | chưa chạy |
| TC-P6-05-38 | N | chưa chạy |
| TC-P6-05-39 | N | chưa chạy |
| TC-P6-05-40 | N | chưa chạy |
| TC-P6-05-41 | P | go test 4 test PASS (log s8flow-go-*.log) |
| TC-P6-05-42 | N | chưa chạy |
| TC-P6-05-43 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-05-44 | P | GV lớp khác (teacher@ gọi lớp QD8F8 do gv.s8f dạy) → 403 FORBIDDEN reason=course |
| TC-P6-05-45 | N | chưa chạy |
| TC-P6-05-46 | N | chưa chạy |
| TC-P6-06-01 | N | chưa chạy |
| TC-P6-06-02 | P | API sổ: A ('1', '0', '10', '9.5', 'A+', 'COMPLETE'); B ('0.5', '0', '8.25', '7', 'B', 'COMPLETE'); C ('0.25', '1.5', '1.25', '3', 'F', 'COMPLETE') khớp bảng tay (UI: |
| TC-P6-06-03 | P | dev e2e ['gradebook layout'] xanh (logs/s8r-dev-gradebook-me.log); ui-antipatterns rc=0 |
| TC-P6-06-04 | N | chưa chạy |
| TC-P6-06-05 | P | dev e2e ['official note'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-06-06 | N | chưa chạy |
| TC-P6-06-07 | N | chưa chạy |
| TC-P6-06-08 | N | chưa chạy |
| TC-P6-06-09 | P | PUT 10.5 → 422 SCORE_OUT_OF_RANGE; DB giữ 9.00 v2 |
| TC-P6-06-10 | P | PUT 8.333 → 422 SCORE_PRECISION (Điểm có quá 2 chữ số thập phân.); DB không đổi, không làm tròn 8,33 |
| TC-P6-06-11 | F | 8/9 chuỗi sai bị từ chối 4xx (abc,-1,1e2,8,,5,1000,8,5x,+5,٥) nhưng ' ' (một dấu cách) → 200 và XOÁ điểm (coi như rỗng); không có 5xx |
| TC-P6-06-12 | P | PUT score null → 200; DB sau: (None, None); trạng thái sinh viên: INCOMPLETE (thiếu điểm) |
| TC-P6-06-13 | P | PUT vào item EXAM (weekly) → 409 ITEM_NOT_EDITABLE |
| TC-P6-06-14 | P | Go 5 test PASS; dev e2e ['cell edit keys'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-06-15 | P | B cuối kỳ 6,5→8,0: row_result qt 8.25 hp 8 B+ (tay 8,25 / 8,10→8,0 B+); khoá trình bày '…' xem UI s8r-ui-06 |
| TC-P6-06-16 | P | 30 PUT tuần tự (lượt 1/4 khi máy tải ~10): p50 91 ms, p95 169 ms, max 349 ms (≤300); phản hồi có row_result; grep công thức trong features/gradebook: chỉ phép nhân t |
| TC-P6-06-17 | P | Go TestPutEntryReturnsRowResult/Under300ms PASS (logs/s8flow-go-results.json) + đo thật p95 169 ms |
| TC-P6-06-18 | P | phiên 2 cùng version → 409 VERSION_CONFLICT, current {'score': '7.5', 'updated_at': '2026-11-15T18:39:59.009766Z', 'updated_by': {'name': 'TS. Lê Thu Hà'}, 'version' |
| TC-P6-06-19 | P | Giữ của tôi: gửi lại với version 2 → 200; DB ('6.00', 3); audit_log có dòng set 7,5 (1); UI nút: s8r-ui-06 |
| TC-P6-06-20 | N | chưa chạy |
| TC-P6-06-21 | P | TA thấy updated_by='TS. Lê Thu Hà'; /me/grades của sinh viên không có khoá updated_by (HTTP 200) |
| TC-P6-06-22 | P | Go 1 test PASS; dev e2e ['cell conflict choose'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-06-23 | P | giải trình B: thứ tự ['Giữa kỳ (20%)', 'Kiểm tra hằng tuần (20%)', 'Thi cuối kỳ (60%)']…; bonus 0,5; QT 8,25; HP thô 7,2 → ['7'] → B |
| TC-P6-06-24 | P | Căn cứ: làm tròn → tr.2 (quote 3.1), điểm cộng → tr.1; dòng do GV đặt: filled_by_user của 'Kiểm tra hằng tuần'=True; UI 'do giảng viên đặt': s8r-ui-06 |
| TC-P6-06-25 | P | sv20: HP 5 (không 0), F; dòng liệt 'Không đạt'=F kèm trích điều 4.1 tr.2 «dưới 3,0»; verdict LIET:final<3.0; dòng không viết số 2,75 trong expr (đúng ở dòng Thi cuối |
| TC-P6-06-26 | P | explain 30/30 sinh viên: Điểm quá trình, HP, chữ khớp cột qt/hp/letter của expected_final_grades.csv |
| TC-P6-06-27 | P | dev e2e ['explain student B'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-06-28 | P | history sv04: 3 mục, mục đầu {"at": "2026-11-15T18:40:06.490124Z", "by": "TS. Lê Thu Hà", "action": "set", "item": "midterm", "from": "7", "to": "6.25"}; next_cursor |
| TC-P6-06-29 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-06-30 | P | LOAD1000 trang đầu limit=100 ×20: p50 20 ms, p95 51 ms (≤500); limit=101 → 422; thứ tự vn_fold(tên),id đúng trên lớp 30 SV có dấu: True |
| TC-P6-06-31 | N | chưa chạy |
| TC-P6-06-32 | N | chưa chạy |
| TC-P6-06-33 | N | chưa chạy |
| TC-P6-06-34 | P | LOAD1000 limit=100: seq_scan +0 trên grade_entries/participation_events/attendance_records (pg_stat_user_tables, min 3 lượt); đếm câu SQL không đo được qua log DB dù |
| TC-P6-06-35 | P | Go 3 test PASS; dev e2e ['virtualized 1000\\|type then scroll keeps text'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-06-36 | N | chưa chạy |
| TC-P6-06-37 | N | chưa chạy |
| TC-P6-06-38 | N | chưa chạy |
| TC-P6-06-39 | N | chưa chạy |
| TC-P6-06-40 | P | PUT khi LOCKED → 409 GRADEBOOK_LOCKED; state {'status': 'LOCKED', 'snapshot_version': 1} (UI 'Đã chốt v1', menu Mở khoá: s8r-ui-08) |
| TC-P6-06-41 | N | chưa chạy |
| TC-P6-06-42 | P | Go 1 test PASS; dev e2e ['gradebook states\\|offline cell'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-06-43 | P | audit mới nhất: grade_entries/set before {"score": "6.75", "item_id": "01a1e082-05b2-7b42-9ba4-ee1046 after {"score": "6", "item_id": "01a1e082-05b2-7b42-9ba4-ee1046 |
| TC-P6-06-44 | P | audit_log chặn INSERT (trigger riêng lớp, thay vì đổi tên bảng dùng chung) → PUT 500; điểm không ghi: ('6.75', 1) = ('6.75', 1); đã gỡ trigger |
| TC-P6-06-45 | P | Go 2 test PASS (logs/s8r-dev-gradebook-me.log); grep mock/gradebook rỗng |
| TC-P6-06-46 | N | chưa chạy |
| TC-P6-06-47 | P | publish true → 200, published_at có; gọi lại cùng thân → 200 (idempotent), audit 2→2 |
| TC-P6-06-48 | P | publish false → 200, published_at NULL; audit grade_items có dòng |
| TC-P6-06-49 | P | publish cột EXAM → 409 ITEM_NOT_PUBLISHABLE |
| TC-P6-06-50 | P | nhập 8 cho sv.kha khi cột đã công bố → /me/grades hiện 8 ngay không thao tác thêm; khôi phục 7.50 |
| TC-P6-06-51 | P | khi LOCKED: GV publish → 409 GRADEBOOK_LOCKED; TA publish → 403 (UI TA không thấy menu: s8r-ui-06) |
| TC-P6-06-52 | P | Go 4 test PASS; dev e2e ['publish column'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-06-53 | N | chưa chạy |
| TC-P6-06-54 | N | chưa chạy |
| TC-P6-06-55 | P | Go 1 test PASS; dev e2e ['penalty not in scheme notice'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-06-56 | F | version kiểu chữ → 422 VALIDATION_FAILED đúng; NHƯNG thiếu version: ô chưa có → 200 (tạo ô, version=0 ngầm); ô đã có → 409 VERSION_CONFLICT (không phải 422 VALIDATIO |
| TC-P6-06-57 | P | PUT cho sinh viên ngoài lớp → 422 VALIDATION_FAILED; số dòng grade_entries giữ 60 |
| TC-P6-06-58 | P | score số JSON 5 → 422; 8.333 → 422 VALIDATION_FAILED; '5.005' → 422; 0.1+0.2 → 422; DB ('7.00', 1) |
| TC-P6-06-59 | P | 20 PUT song song cùng version 1: 200×1, 409×19, khác []; audit +1; version 2 |
| TC-P6-06-60 | N | chưa chạy |
| TC-P6-06-61 | P | PUT bằng TA/STUDENT/ADMIN → {'ta': 403, 'sv.gioi': 403, 'admin': 403}; DB không đổi |
| TC-P6-06-62 | P | GV lớp khác GET gradebook → 403; STUDENT → 403; không token → 401 |
| TC-P6-06-63 | N | chưa chạy |
| TC-P6-07-01 | P | presign → 200 upload_id + URL ký (kho MinIO :9700), PUT thành công; `docker diff` gateway: tệp .xlsx mới 0→0 (không ghi đĩa cục bộ) |
| TC-P6-07-02 | P | đuôi .pdf / .csv / .xlsx+mime pdf → [(422, 'FILE_TYPE_NOT_ALLOWED'), (422, 'FILE_TYPE_NOT_ALLOWED'), (422, 'FILE_TYPE_NOT_ALLOWED')] |
| TC-P6-07-03 | P | 11 MiB khai thật → 422 FILE_TOO_LARGE; khai 2000 B nhưng PUT 11 MiB → kho MinIO từ chối (ngắt kết nối) |
| TC-P6-07-04 | P | 11 presign trong ~1 s (teacher): [200, 200, 200, 200, 200, 200, 200, 200, 200, 200, 429]; lần 11 = 429; GV khác (gv.s8f) cùng lúc → 200 (không bị ảnh hưởng) |
| TC-P6-07-05 | P | presign khai sha256 sai + PUT tệp thật → preview 422 FILE_INVALID; không ghi (grade_entries 60) |
| TC-P6-07-06 | P | TA/STUDENT/ADMIN × (presign, preview, commit) → [403, 403, 403, 403, 403, 403, 403, 403, 403] |
| TC-P6-07-07 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-07-08 | P | preview midterm.xlsx → rows_total 30, rows_ok 30, overwrites 0, unchanged 30, errors [], sample 0 (≤20); grade_entries 60→60, audit_log 3→3 không đổi |
| TC-P6-07-09 | N | chưa chạy |
| TC-P6-07-10 | P | go test 2 test PASS (log s8flow-go-*.log) |
| TC-P6-07-11 | F | {2: 'STUDENT_NOT_IN_COURSE', 4: 'DUPLICATE_IN_FILE', 5: 'SCORE_OUT_OF_RANGE', 6: 'SCORE_PRECISION', 7: 'SCORE_NOT_NUMBER', 10: 'SCORE_NOT_NUMBER'} vs {2: 'STUDENT_NO |
| TC-P6-07-12 | P | formula.xlsx: dòng B (=22/3 định dạng 0.00 → '7.33') hợp lệ; dòng C (=22/3 chung) → {3: 'SCORE_PRECISION'}; rows_ok 1 |
| TC-P6-07-13 | P | MSSV ô số thực 20229002.0 → rows_ok 1, 0 lỗi (khớp B: điểm 6 vs hiện có 7,5 → overwrites 1) |
| TC-P6-07-14 | P | X.users.student_code = MSSV của Y nhưng snapshot X = XSELF0001; dòng '20229555, 9' → điểm vào Y (snapshot khớp, ROSTER) = 9.00, X = None (không nhận) |
| TC-P6-07-15 | P | 2 SV ROSTER cùng snapshot 20229666 → preview ['AMBIGUOUS_STUDENT_CODE'] |
| TC-P6-07-16 | P | '00123456' (chuỗi) khớp đúng người (điểm 7.00); '123456' → {3: 'STUDENT_NOT_IN_COURSE'} (không mất số 0) |
| TC-P6-07-17 | P | go test 8 test PASS (log s8flow-go-*.log) |
| TC-P6-07-18 | P | 3 ô bị sửa tay → commit midterm.xlsx: 200 (ok 30, overwrites 3, unchanged 27); 3 ô đổi version+1 source IMPORT; 27 ô giống cũ không version mới; audit mới {'import': |
| TC-P6-07-19 | P | commit lần hai cùng tệp, khoá mới: unchanged 30 = rows_ok 30; 0 version mới; audit 11→11 (0 dòng mới) |
| TC-P6-07-20 | P | commit lại cùng Idempotency-Key → 200, phản hồi y hệt lần đầu; version/audit không đổi |
| TC-P6-07-21 | P | errors.xlsx commit không skip_errors → 409 IMPORT_HAS_ERRORS; không ghi dòng nào (kể cả dòng hợp lệ): version/entries/audit giữ nguyên |
| TC-P6-07-22 | P | skip_errors:true → 200: ghi 2 dòng hợp lệ (B=7, sv06=8,5) trong một lô, bỏ 6 dòng lỗi; ô khác không đổi (đổi: 2); UI nút 'Bỏ qua N dòng lỗi và ghi M dòng': dev e2e ' |
| TC-P6-07-23 | P | item EXAM làm đích: preview 409 ITEM_NOT_EDITABLE, commit 409 ITEM_NOT_EDITABLE |
| TC-P6-07-24 | P | sổ LOCKED: commit → 409 GRADEBOOK_LOCKED (preview 200) |
| TC-P6-07-25 | P | 6 vòng commit(10 ô) ∥ finalize: (commit, finalize, lý do) = [(409, 200, 'GRADEBOOK_LOCKED'), (409, 200, 'GRADEBOOK_LOCKED'), (409, 200, 'GRADEBOOK_LOCKED'), (409, 20 |
| TC-P6-07-26 | P | go test 6 test PASS (log s8flow-go-*.log) |
| TC-P6-07-27 | P | DB trống GK/CK → nhập midterm.xlsx + final.xlsx (commit): 60 ô; từng giá trị khớp inputs CSV (0 lệch); preview 30/30 khớp expected_final_grades.csv (hp, letter, verd |
| TC-P6-07-28 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-07-29 | P | zipbomb.xlsx (170 KiB nén, XML ~64 MiB) → 422 FILE_INVALID; RSS gateway 19→19 MiB (tăng <100); không restart (StartedAt giữ) |
| TC-P6-07-30 | P | 5.000 dòng → 200 rows_total 5000 (lỗi 5000 mục, ghi 0); 5.001 dòng → 422 TOO_MANY_ROWS |
| TC-P6-07-31 | P | corrupt.xlsx / notxlsx.xlsx (MIME khai đúng) → [(422, 'FILE_INVALID'), (422, 'FILE_INVALID')]; log gateway 2 phút gần nhất: 'panic' 0 |
| TC-P6-07-32 | P | formula.xlsx + inject.xlsx preview → [200, 200] (không thực thi công thức: 22/3 đọc theo giá trị đã định dạng); log gateway 2 phút: tên tệp gốc/nội dung ô độc xuất h |
| TC-P6-07-33 | P | preview OK (200); thay đối tượng trong kho bằng tệp B cùng độ dài (1625 B): PUT lại → 200; commit → 422 FILE_INVALID; sum(version) giữ 81→81 |
| TC-P6-07-34 | P | go test 5 test PASS (log s8flow-go-*.log) |
| TC-P6-07-35 | P | export OPEN → 200; Cache-Control no-store; Content-Disposition attachment; filename="so-diem-QD8R7.xlsx"; 8445 B mở được |
| TC-P6-07-36 | P | export OPEN: hàng đầu [tên lớp, mã QD8R7, thời điểm, 'Bản xem trước — chưa chốt']; hàng tiêu đề MSSV,Họ tên,Giữa kỳ,Kiểm tra hằng tuần,Thi cuối kỳ,Điểm cộng,Điểm trừ |
| TC-P6-07-37 | P | export khi LOCKED v1 sau khi nguồn sống đổi: hàng đầu 'QC s8core QD8R8 QD8R8 16/11/2026 01:44 Bản chốt v1'; HP sv04 trong tệp 6.5 = snapshot 6.5 (sống 5) |
| TC-P6-07-38 | P | 4 sinh viên tên bắt đầu = + - @ → export có 4 ô tên đúng chữ gốc, <f> trong sheet = 0 (không công thức); ô kiểu inlineStr/s |
| TC-P6-07-39 | P | audit_log action='export': 1 dòng; mới nhất {"rows": 30, "locked": false, "version": 0} |
| TC-P6-07-40 | P | export bởi TA/STUDENT/ADMIN → [403, 403, 403] |
| TC-P6-07-41 | P | go test 5 test PASS (log s8flow-go-*.log) |
| TC-P6-07-42 | P | LOAD1000 (5 lần, max): xuất 0.08s (≤3), xem trước 1.000 dòng 0.04s (≤2), ghi 1.000 dòng 0.08s (≤3); số câu SQL 1.000 dòng = 10 dòng: TestImportNoNPlusOne (test Go đế |
| TC-P6-07-43 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-07-44 | N | chưa chạy |
| TC-P6-07-45 | N | chưa chạy |
| TC-P6-07-46 | N | chưa chạy |
| TC-P6-07-47 | N | chưa chạy |
| TC-P6-07-48 | N | chưa chạy |
| TC-P6-07-49 | P | dev e2e ['import xlsx flow\\|import errors listed'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-07-50 | P | upload_id lớp khác → 404 UPLOAD_NOT_FOUND; upload_id không tồn tại → 404; không ghi |
| TC-P6-07-51 | P | item_id của lớp khác → 404 NOT_FOUND |
| TC-P6-07-52 | P | commit thiếu Idempotency-Key → 422 IDEMPOTENCY_KEY_REQUIRED; không ghi |
| TC-P6-07-53 | P | mapping cột điểm = cột MSSV → 200; rows_ok 0; mã lỗi ['SCORE_NOT_NUMBER']; không 500 |
| TC-P6-07-54 | P | 2 commit song song (khoá khác nhau) → [200, 200]; sau cùng bằng một lần commit: 2 ô đổi, mỗi ô version+1 (không đôi); không 5xx/deadlock |
| TC-P6-07-55 | P | Z (CODE, tự khai 20229777): preview {2: 'STUDENT_CODE_UNVERIFIED'}; commit không skip → 409 IMPORT_HAS_ERRORS; skip_errors → 200, ghi B=6, entries của Z 0→0 |
| TC-P6-07-56 | P | 20229888 có ROSTER (R) + CODE (Z2): preview không lỗi/AMBIGUOUS; điểm vào R=8.00, Z2=None |
| TC-P6-07-57 | P | 20229999 joined_via=ADMIN → khớp, điểm 7.00 |
| TC-P6-07-58 | P | PUT tay cho Z (CODE) → 200, điểm 6.50; sổ vẫn liệt kê Z (39 dòng) |
| TC-P6-07-59 | N | chưa chạy |
| TC-P6-08-01 | P | finalize → 200 {'version': 1, 'students': 30, 'locked_at': '2026-11-15T18:43:09.999815Z'}; snapshot v1 30 dòng; gradebooks LOCKED v1; scheme_id/locked_by/locked_at c |
| TC-P6-08-02 | P | snapshot v1 30/30 khớp expected_final_grades.csv (hp↔numeric_grade, letter, verdict↔failed_reason; sv20 5,0 F kèm lý do liệt) |
| TC-P6-08-03 | P | numeric_grade của snapshot = hp trong preview lưu trước khi chốt, 30/30 |
| TC-P6-08-04 | P | gửi lại cùng Idempotency-Key → 200, phản hồi giống hệt; count(*)=30, snapshot_version=1 |
| TC-P6-08-05 | P | go test 5 test PASS (log s8flow-go-*.log) |
| TC-P6-08-06 | P | finalize lớp NO_POLICY (QD8F3) → 409 SCHEME_NOT_CONFIRMED |
| TC-P6-08-07 | P | xoá cuối kỳ sv30, sv27 → finalize 409 GRADES_INCOMPLETE total=2 students=[{"id": "01a1df27-6553-7c98-bc34-a0c697597f85", "missing": ["final"]}, {"id": "01a1df27-654e |
| TC-P6-08-08 | P | LOAD1000 thiếu 40 ô cuối kỳ → finalize 409 GRADES_INCOMPLETE: details.total=40, students.length=30 (≤30) |
| TC-P6-08-09 | P | finalize lần hai khoá mới → 409 GRADEBOOK_LOCKED |
| TC-P6-08-10 | N | chưa chạy |
| TC-P6-08-11 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-08-12 | P | khi LOCKED: PUT/publish/confirm → {'put': 409, 'publish': 409, 'import': 404, 'confirm': 409} (['GRADEBOOK_LOCKED', 'GRADEBOOK_LOCKED', 'GRADEBOOK_LOCKED']); import  |
| TC-P6-08-13 | P | sửa điểm bài thi sv04 (nguồn sống 6→0): băm SNAP(1) không đổi; hàng sv04 drift=true & locked.version=1 (xem trước sống HP 5 ≠ snapshot 6.5); chỉ 1 hàng drift; STUDEN |
| TC-P6-08-14 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-08-15 | N | chưa chạy |
| TC-P6-08-16 | P | unlock → 200; gradebooks OPEN, unlock_reason đúng, unlocked_by=teacher, unlocked_at có; audit unlock +1; SNAP(1) không đổi |
| TC-P6-08-17 | P | thiếu/''/9/501/10 dấu cách → 422 VALIDATION_FAILED (5/5), sổ vẫn LOCKED; biên 10 ký tự → 200, 500 ký tự → 200 |
| TC-P6-08-18 | P | unlock khi OPEN → 409 NOT_LOCKED |
| TC-P6-08-19 | P | TA/SV/ADMIN finalize+unlock → {'ta': (403, 403), 'sv.gioi': (403, 403), 'admin': (403, 403)}; GV lớp khác → (403, 403); 0 snapshot |
| TC-P6-08-20 | P | Go 4 test PASS; dev e2e ['unlock reason'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-08-21 | P | sv08 cuối kỳ 8.25→9,25 (200) xem trước HP 9 A+ (tay 9,0 A+); sv04 giữa kỳ 6.25→7,25 (200) HP 6.5 C+ (tay 6,5 C+ không đổi) |
| TC-P6-08-22 | P | finalize lần hai → 200 version 2; snapshot v2 30 dòng, unlock_reason = 'Phúc khảo sv04: sửa điểm giữa kỳ'; gradebooks LOCKED v2 |
| TC-P6-08-23 | P | băm md5 SNAP(1) trước = sau chốt lại v2: 6adad003 = 6adad003 |
| TC-P6-08-24 | P | gradebook state LOCKED v2; export hàng đầu '26 01:45 Bản chốt v2', sv08 9 A+; snapshots list 200, snapshots/1 200; UI 'Lịch sử chốt' + chọn v1 (8,5 A): s8r-ui-08 |
| TC-P6-08-25 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-08-26 | P | GRADE_CHANGED: đúng 1 thông báo (sv08, đến sau ~0,5 s qua outbox); nội dung 'Điểm học phần lớp QD8R8 đã được cập nhật.' không có số/chữ điểm; sv04 (HP không đổi) và  |
| TC-P6-08-27 | P | GRADE_FINALIZED = 30 cho cả lớp 30 sau lần chốt đầu; gửi lại cùng khoá không thêm (08-04: vẫn 30 snapshot) |
| TC-P6-08-28 | P | Mailpit: 0 thư có 'điểm học phần'; thông báo chỉ nằm bảng notifications (chuông); mục 'Hôm nay' sinh viên: kiểm ở 09/UI |
| TC-P6-08-29 | P | go test 4 test PASS (log s8flow-go-*.log) |
| TC-P6-08-30 | P | 2 finalize song song (khoá khác nhau) → [200, 409] (409 = GRADEBOOK_LOCKED); count(distinct version)=1, 30 dòng |
| TC-P6-08-31 | P | 20 vòng (mở khoá→finalize ∥ PUT sv05): PUT 200 trước chốt 0 lần (giá trị có trong snapshot), 409 GRADEBOOK_LOCKED 20 lần (snapshot không có); 20/20 snapshot khớp gra |
| TC-P6-08-32 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-08-33 | P | LOAD1000 finalize ×3 (max): 0.22s (≤3); đọc bản chốt trang đầu snapshots/3?limit=100: p90 14 ms (≤500), 100 dòng; số câu SQL 1.000 = 30 SV: TestFinalizeNoNPlusOne (t |
| TC-P6-08-34 | P | đọc bản chốt LOAD1000 limit=100: seq_scan final_grade_snapshots tăng [0, 0, 0] (min 0 = dùng chỉ mục final_grade_snapshots_version_idx); EXPLAIN truy vấn: TestFinali |
| TC-P6-08-35 | P | go test 2 test PASS (log s8flow-go-*.log) |
| TC-P6-08-36 | N | chưa chạy |
| TC-P6-08-37 | N | chưa chạy |
| TC-P6-08-38 | N | chưa chạy |
| TC-P6-08-39 | P | dev e2e ['finalize flow\\|finalize incomplete list'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-08-40 | P | sau chốt: finalize bởi TA/SV/ADMIN/GV lớp khác → {'ta': 403, 'sv.gioi': 403, 'admin': 403, 'g7t': 403} |
| TC-P6-08-41 | P | sinh viên B GET snapshots / snapshots/{1,2,3} → [403, 403, 403, 403]; đoán snapshot_id (A + 20 id) → [403]: không dòng nào lộ |
| TC-P6-08-42 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-08-43 | P | finalize/unlock thiếu Idempotency-Key → 422/422 (IDEMPOTENCY_KEY_REQUIRED); sổ vẫn OPEN, 0 snapshot |
| TC-P6-08-44 | P | sv12 rời lớp → chốt lại v3: 29 snapshot (29, không có sv12); snapshot cũ của sv12 còn 2 (v1,v2); thông báo GRADE* của sv12 1→1 (không nhận mới) |
| TC-P6-08-45 | P | đưa bài thi tuần 9 về CLOSED (chưa công bố) → finalize 409 GRADES_INCOMPLETE (total 30, missing ['weekly']), không phải chốt cả lớp 0 |
| TC-P6-08-46 | P | audit_log chặn INSERT (trigger riêng lớp này, thay REVOKE để không ảnh hưởng nhóm khác) → finalize 500; không snapshot mới (89→89), sổ vẫn OPEN; đã gỡ trigger |
| TC-P6-08-47 | P | v1→v2→v3 liên tiếp: version 1,2,3 không nhảy/trùng; v3 mang unlock_reason của lần mở trước nó; v1, v2 giữ nguyên băm; (v2 mang 'Phúc khảo sv04: sửa điểm giữa kỳ') |
| TC-P6-09-01 | N | chưa chạy |
| TC-P6-09-02 | N | chưa chạy |
| TC-P6-09-03 | N | chưa chạy |
| TC-P6-09-04 | N | chưa chạy |
| TC-P6-09-05 | N | chưa chạy |
| TC-P6-09-06 | P | go test 4 test PASS (log s8flow-go-*.log) |
| TC-P6-09-07 | N | chưa chạy |
| TC-P6-09-08 | N | chưa chạy |
| TC-P6-09-09 | N | chưa chạy |
| TC-P6-09-10 | N | chưa chạy |
| TC-P6-09-11 | N | chưa chạy |
| TC-P6-09-12 | N | chưa chạy |
| TC-P6-09-13 | N | chưa chạy |
| TC-P6-09-14 | N | chưa chạy |
| TC-P6-09-15 | N | chưa chạy |
| TC-P6-09-16 | P | dev e2e ['grade section\\|calculation linear\\|no gamify'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-09-17 | N | chưa chạy |
| TC-P6-09-18 | N | chưa chạy |
| TC-P6-09-19 | N | chưa chạy |
| TC-P6-09-20 | N | chưa chạy |
| TC-P6-09-21 | N | chưa chạy |
| TC-P6-09-22 | N | chưa chạy |
| TC-P6-09-23 | N | chưa chạy |
| TC-P6-09-24 | P | go test 4 test PASS (log s8flow-go-*.log) |
| TC-P6-09-25 | N | chưa chạy |
| TC-P6-09-26 | N | chưa chạy |
| TC-P6-09-27 | N | chưa chạy |
| TC-P6-09-28 | N | chưa chạy |
| TC-P6-09-29 | N | chưa chạy |
| TC-P6-09-30 | N | chưa chạy |
| TC-P6-09-31 | P | Go 5 test PASS; dev e2e ['my grade question'] xanh (logs/s8r-dev-gradebook-me.log); |
| TC-P6-09-32 | N | chưa chạy |
| TC-P6-09-33 | N | chưa chạy |
| TC-P6-09-34 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-09-35 | N | chưa chạy |
| TC-P6-09-36 | N | chưa chạy |
| TC-P6-09-37 | N | chưa chạy |
| TC-P6-09-38 | P | sau chốt, B gọi snapshots/1, snapshots/<id của A>, 20 id lân cận → chỉ [403]; không rò dòng của A |
| TC-P6-09-39 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-09-40 | N | chưa chạy |
| TC-P6-09-41 | N | chưa chạy |
| TC-P6-09-42 | N | chưa chạy |
| TC-P6-09-43 | N | chưa chạy |
| TC-P6-09-44 | N | chưa chạy |
| TC-P6-09-45 | N | chưa chạy |
| TC-P6-09-46 | N | chưa chạy |
| TC-P6-09-47 | P | dev e2e ['grade states\\|grade mobile 375'] xanh (logs/s8r-dev-gradebook-me.log); grep mock ở features/me rỗng |
| TC-P6-09-48 | N | chưa chạy |
| TC-P6-09-49 | N | chưa chạy |
| TC-P6-09-50 | N | chưa chạy |
| TC-P6-SEED-01 | N | chưa chạy |
| TC-P6-SEED-02 | N | chưa chạy |
| TC-P6-SEED-03 | N | chưa chạy |
| TC-P6-SEED-04 | N | chưa chạy |
| TC-P6-SEED-05 | N | chưa chạy |
| TC-P6-SEED-06 | N | chưa chạy |
| TC-P6-SEED-07 | N | chưa chạy |
| TC-P6-SEED-08 | N | chưa chạy |
| TC-P6-SEED-09 | N | chưa chạy |
| TC-P6-SEED-10 | N | chưa chạy |
| TC-P6-SEED-11 | N | chưa chạy |
| TC-P6-SEED-12 | N | chưa chạy |
| TC-P6-SEED-13 | N | chưa chạy |
| TC-P6-SEED-14 | N | chưa chạy |
| TC-P6-SEED-15 | N | chưa chạy |
| TC-P6-SEED-16 | N | chưa chạy |
| TC-P6-SEED-17 | N | chưa chạy |
| TC-P6-SEED-18 | N | chưa chạy |
| TC-P6-SEED-19 | N | chưa chạy |
| TC-P6-SEED-20 | N | chưa chạy |
| TC-P6-SEED-21 | N | chưa chạy |
| TC-P6-SEED-22 | N | chưa chạy |
| TC-P6-SEED-23 | N | chưa chạy |
| TC-P6-SEED-24 | N | chưa chạy |
| TC-P6-SEED-25 | N | chưa chạy |
| TC-P6-SEED-26 | N | chưa chạy |
| TC-P6-SEED-27 | N | chưa chạy |
| TC-P6-SEED-28 | N | chưa chạy |
| TC-P6-SEED-29 | N | chưa chạy |
| TC-P6-SEED-30 | N | chưa chạy |
| TC-P6-SEED-31 | N | chưa chạy |
| TC-P6-SEED-32 | N | chưa chạy |
| TC-P6-SEED-33 | N | chưa chạy |
| TC-P6-SEED-34 | N | chưa chạy |
| TC-P6-SEED-35 | N | chưa chạy |
| TC-P6-SEED-36 | N | chưa chạy |
| TC-P6-SEED-37 | N | chưa chạy |
| TC-S8-UI-01 | N | chưa chạy |
| TC-S8-UI-02 | N | chưa chạy |
| TC-S8-UI-03 | N | chưa chạy |
| TC-S8-UI-04 | N | chưa chạy |
| TC-S8-UI-05 | N | chưa chạy |
| TC-S8-UI-06 | N | chưa chạy |
| TC-S8-UI-07 | N | chưa chạy |
| TC-S8-UI-08 | N | chưa chạy |
| TC-S8-UI-09 | N | chưa chạy |
| TC-S8-UI-10 | N | chưa chạy |
| TC-S8-UI-11 | N | chưa chạy |
| TC-S8-UI-12 | N | chưa chạy |
| TC-S8-UI-13 | N | chưa chạy |
| TC-S8-UI-14 | N | chưa chạy |
| TC-S8-UI-15 | N | chưa chạy |
| TC-S8-UI-16 | N | chưa chạy |
| TC-S8-UI-17 | N | chưa chạy |
| TC-S8-UI-18 | N | chưa chạy |
| TC-S8-UI-19 | N | chưa chạy |
| TC-S8-UI-20 | N | chưa chạy |
| TC-S8-UI-21 | N | chưa chạy |
| TC-S8-UI-22 | N | chưa chạy |
| TC-S8-UI-23 | N | chưa chạy |
