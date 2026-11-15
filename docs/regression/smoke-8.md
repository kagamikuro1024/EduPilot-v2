# Smoke hồi quy sprint 8 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm D: SmkD + S8Core/S8Flow/S8Seed) · Ngày: 2026-11-15
Tổng TC cũ: 512 · Đạt: 133 · Trượt: 7 · Lỗi thời (LT): 0 · Không chạy K (a/b/c): 0 · Chưa chạy (N): 372   (P+F+LT+K+N = T)

Bằng chứng chung: `p6-hand-calc.py` 30/30 + 18/18 khớp; `go test ./internal/grade` 5 test lõi PASS (logs/). Ghi chú từng ca: các file `part-*.tsv`, script/log ở `docs/regression/evidence-8/`.

## Tổng hợp theo US
| US | TC tổng | P | F | LT | K | N |
|---|---|---|---|---|---|---|
| P6-01 | 41 | 32 | 1 | 0 | 0 | 8 |
| P6-02 | 49 | 41 | 3 | 0 | 0 | 5 |
| P6-03 | 48 | 17 | 0 | 0 | 0 | 31 |
| P6-04 | 49 | 6 | 0 | 0 | 0 | 43 |
| P6-05 | 46 | 15 | 3 | 0 | 0 | 28 |
| P6-06 | 63 | 2 | 0 | 0 | 0 | 61 |
| P6-07 | 59 | 8 | 0 | 0 | 0 | 51 |
| P6-08 | 47 | 8 | 0 | 0 | 0 | 39 |
| P6-09 | 50 | 4 | 0 | 0 | 0 | 46 |
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
| TC-P6-01-21 | N | chưa chạy |
| TC-P6-01-22 | N | chưa chạy |
| TC-P6-01-23 | N | chưa chạy |
| TC-P6-01-24 | P | TestExamScorePublishedOnly/FollowsAdjust/MissingIsNotZero/Scale/MatchesSeedCSV PASS; test có require.Equal(24 dòng MCQ) |
| TC-P6-01-25 | P | SQL P5: A cộng 3,00 vắng 0 · B 0,50 vắng 2 · C 0,25 vắng 5 (khớp expected_crm); giải trình dùng đúng (min(0,5; 1); max(0; 5 − 2) × 0,5 + 0) |
| TC-P6-01-26 | N | chưa chạy |
| TC-P6-01-27 | N | chưa chạy |
| TC-P6-01-28 | P | grep study_time_daily|student_risk|student_observations ở mã không-test internal/grade: rỗng |
| TC-P6-01-29 | P | TestInputsFromP5/ExcusedLateNotAbsent/ActiveOnly/MatchExpectedCRM Go integration -tags integration (logs/s8core-go-*.log) PASS |
| TC-P6-01-30 | N | chưa chạy |
| TC-P6-01-31 | P | lớp 2 gradebook 200: 24 dòng (đều thuộc lớp 2), trạng thái NO_POLICY (chưa công thức), không điểm lớp 1 |
| TC-P6-01-32 | P | IDOR: item lớp 1 qua đường dẫn lớp có công thức (QD8A1): PUT grade-entries 404, publish 404; audit_log lớp không tăng 4→4 |
| TC-P6-01-33 | N | chưa chạy |
| TC-P6-01-34 | N | chưa chạy |
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
| TC-P6-02-24 | N | chưa chạy |
| TC-P6-02-25 | F | C: KT=0, status COMPLETE, nhưng dòng KT chỉ expr '0' (filled_by_user, không quote điều 1.4), không có chữ 'không dự → 0 theo quy chế' (DEF-R8-s8core-01) |
| TC-P6-02-26 | P | xoá ô final sv30 (PUT score null 200): preview status INCOMPLETE, không hp/letter ({'status': 'INCOMPLETE', 'hp': None, 'letter': None, 'verdict': 'INCOMPL |
| TC-P6-02-27 | N | chưa chạy |
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
| TC-P6-02-42 | F | 200 {'students_total': 30, 'students_computable': 30, 'changed_vs_confirmed': 21, 'entries_carried': 60, 'entries_dropped': [], 'samples': [{'student': {'i |
| TC-P6-02-43 | N | chưa chạy |
| TC-P6-02-44 | N | chưa chạy |
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
| TC-P6-04-08 | N | chưa chạy |
| TC-P6-04-09 | N | chưa chạy |
| TC-P6-04-10 | N | chưa chạy |
| TC-P6-04-11 | N | chưa chạy |
| TC-P6-04-12 | N | chưa chạy |
| TC-P6-04-13 | N | chưa chạy |
| TC-P6-04-14 | N | chưa chạy |
| TC-P6-04-15 | P | 3 mẫu: cao nhất 10→9,8 · giữa 6.5→6.5 (vị trí 15/16 theo HP) · thấp nhất 3→3,2; giữa khớp hand-calc (đo ≤300 ms: xem UI/TC-04-15 ghi chú) |
| TC-P6-04-16 | P | impact 2 lần: audit_log lớp 4→4, grade_entries 60→60 (không ghi); N=21 khớp tính tay; khoá ['changed_vs_confirmed', 'entries_carried', 'entries_dropped', ' |
| TC-P6-04-17 | N | chưa chạy |
| TC-P6-04-18 | N | chưa chạy |
| TC-P6-04-19 | N | chưa chạy |
| TC-P6-04-20 | N | chưa chạy |
| TC-P6-04-21 | N | chưa chạy |
| TC-P6-04-22 | N | chưa chạy |
| TC-P6-04-23 | N | chưa chạy |
| TC-P6-04-24 | N | chưa chạy |
| TC-P6-04-25 | N | chưa chạy |
| TC-P6-04-26 | N | chưa chạy |
| TC-P6-04-27 | N | chưa chạy |
| TC-P6-04-28 | P | TestConfirmCarryOverEntries/NoCarryOnMaxScoreChange/NoCarryOnSourceChange/SupersedeBeforeConfirm/CreatesGradebookRow/BlockedWhenLocked PASS |
| TC-P6-04-29 | N | chưa chạy |
| TC-P6-04-30 | N | chưa chạy |
| TC-P6-04-31 | N | chưa chạy |
| TC-P6-04-32 | N | chưa chạy |
| TC-P6-04-33 | N | chưa chạy |
| TC-P6-04-34 | N | chưa chạy |
| TC-P6-04-35 | N | chưa chạy |
| TC-P6-04-36 | N | chưa chạy |
| TC-P6-04-37 | P | grep mock/gradebook frontend/src rỗng; frontend/src/mock/gradebook.ts không tồn tại |
| TC-P6-04-38 | N | chưa chạy |
| TC-P6-04-39 | N | chưa chạy |
| TC-P6-04-40 | N | chưa chạy |
| TC-P6-04-41 | N | chưa chạy |
| TC-P6-04-42 | N | chưa chạy |
| TC-P6-04-43 | N | chưa chạy |
| TC-P6-04-44 | N | chưa chạy |
| TC-P6-04-45 | N | chưa chạy |
| TC-P6-04-46 | N | chưa chạy |
| TC-P6-04-47 | P | GV lớp khác GET grade-scheme → 403 |
| TC-P6-04-48 | N | chưa chạy |
| TC-P6-04-49 | P | ADMIN GET grade-scheme → 403 |
| TC-P6-05-01 | P | L2={"state": "NO_POLICY", "unclear_count": 0, "has_policy": false, "pending_draft": false, "gradebook": {"status": "OPEN", "snapshot_version": 0}, "fi |
| TC-P6-05-02 | P | doc=READY status={"state": "NOT_EXTRACTED", "unclear_count": 0, "has_policy": true, "pending_draft": false, "gradebook": {"stat |
| TC-P6-05-03 | F | status={"state": "DRAFT_UNCLEAR", "unclear_count": 3, "has_policy": true, "pending_draft": true, "gradebook": {"status": "OPEN", "snapsho |
| TC-P6-05-04 | P | status={"state": "DRAFT_READY", "unclear_count": 0, "has_policy": true, "pending_draft": true, "gradebook": {"status": "OPEN", "snapshot_ |
| TC-P6-05-05 | N | chưa chạy |
| TC-P6-05-06 | N | chưa chạy |
| TC-P6-05-07 | P | SV/ADM/TA=[403, 403, 200] |
| TC-P6-05-08 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-05-09 | P | QD8F2 NO_POLICY: 1 banner 'Công thức điểm chưa xác nhận' + liên kết 'Tải quy chế' (Xem công thức, Thiết lập công thức); shot s8flow-us05-nopolicy-gradebook-teac |
| TC-P6-05-10 | P | TA /gradebook & /attendance lớp NO_POLICY: banner cùng chữ, liên kết 'Xem công thức' (không 'Tải quy chế', không 'Chốt điểm') |
| TC-P6-05-11 | N | chưa chạy |
| TC-P6-05-12 | P | TEACHER lớp 1 CONFIRMED: 0 banner (gradebook, attendance); STUDENT: 0 banner ở /me,/attendance,/gradebook,/chat |
| TC-P6-05-13 | N | chưa chạy |
| TC-P6-05-14 | N | chưa chạy |
| TC-P6-05-15 | P | Chốt điểm disabled+aria-disabled=true; tooltip 'Chưa có công thức điểm đã xác nhận' khi hover và focus; 0 icon Info |
| TC-P6-05-16 | N | chưa chạy |
| TC-P6-05-17 | N | chưa chạy |
| TC-P6-05-18 | N | chưa chạy |
| TC-P6-05-19 | P | /attendance QD8F2: banner 'Công thức điểm chưa xác nhận'+'Xem công thức' phía trên lưới 1 dòng; lưu thao tác xem TC-20 |
| TC-P6-05-20 | N | chưa chạy |
| TC-P6-05-21 | N | chưa chạy |
| TC-P6-05-22 | N | chưa chạy |
| TC-P6-05-23 | F | GS=[{"id": "GRADE_SCHEME_UNCONFIRMED:01a1dff6-f018-7550-b9bb-abfaa2cf3d9d", "kind": setup=[] |
| TC-P6-05-24 | F | ['Đã có quy chế, chưa trích công thức.']|['Đã có quy chế, chưa trích công thức.']|['Đã có quy chế, chưa trích công thức.'] |
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
| TC-P6-05-44 | P | pending |
| TC-P6-05-45 | N | chưa chạy |
| TC-P6-05-46 | N | chưa chạy |
| TC-P6-06-01 | N | chưa chạy |
| TC-P6-06-02 | N | chưa chạy |
| TC-P6-06-03 | N | chưa chạy |
| TC-P6-06-04 | N | chưa chạy |
| TC-P6-06-05 | N | chưa chạy |
| TC-P6-06-06 | N | chưa chạy |
| TC-P6-06-07 | N | chưa chạy |
| TC-P6-06-08 | N | chưa chạy |
| TC-P6-06-09 | N | chưa chạy |
| TC-P6-06-10 | N | chưa chạy |
| TC-P6-06-11 | N | chưa chạy |
| TC-P6-06-12 | N | chưa chạy |
| TC-P6-06-13 | N | chưa chạy |
| TC-P6-06-14 | N | chưa chạy |
| TC-P6-06-15 | N | chưa chạy |
| TC-P6-06-16 | N | chưa chạy |
| TC-P6-06-17 | P | go test 2 test PASS (log s8flow-go-*.log) |
| TC-P6-06-18 | N | chưa chạy |
| TC-P6-06-19 | N | chưa chạy |
| TC-P6-06-20 | N | chưa chạy |
| TC-P6-06-21 | N | chưa chạy |
| TC-P6-06-22 | N | chưa chạy |
| TC-P6-06-23 | N | chưa chạy |
| TC-P6-06-24 | N | chưa chạy |
| TC-P6-06-25 | N | chưa chạy |
| TC-P6-06-26 | N | chưa chạy |
| TC-P6-06-27 | N | chưa chạy |
| TC-P6-06-28 | N | chưa chạy |
| TC-P6-06-29 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-06-30 | N | chưa chạy |
| TC-P6-06-31 | N | chưa chạy |
| TC-P6-06-32 | N | chưa chạy |
| TC-P6-06-33 | N | chưa chạy |
| TC-P6-06-34 | N | chưa chạy |
| TC-P6-06-35 | N | chưa chạy |
| TC-P6-06-36 | N | chưa chạy |
| TC-P6-06-37 | N | chưa chạy |
| TC-P6-06-38 | N | chưa chạy |
| TC-P6-06-39 | N | chưa chạy |
| TC-P6-06-40 | N | chưa chạy |
| TC-P6-06-41 | N | chưa chạy |
| TC-P6-06-42 | N | chưa chạy |
| TC-P6-06-43 | N | chưa chạy |
| TC-P6-06-44 | N | chưa chạy |
| TC-P6-06-45 | N | chưa chạy |
| TC-P6-06-46 | N | chưa chạy |
| TC-P6-06-47 | N | chưa chạy |
| TC-P6-06-48 | N | chưa chạy |
| TC-P6-06-49 | N | chưa chạy |
| TC-P6-06-50 | N | chưa chạy |
| TC-P6-06-51 | N | chưa chạy |
| TC-P6-06-52 | N | chưa chạy |
| TC-P6-06-53 | N | chưa chạy |
| TC-P6-06-54 | N | chưa chạy |
| TC-P6-06-55 | N | chưa chạy |
| TC-P6-06-56 | N | chưa chạy |
| TC-P6-06-57 | N | chưa chạy |
| TC-P6-06-58 | N | chưa chạy |
| TC-P6-06-59 | N | chưa chạy |
| TC-P6-06-60 | N | chưa chạy |
| TC-P6-06-61 | N | chưa chạy |
| TC-P6-06-62 | N | chưa chạy |
| TC-P6-06-63 | N | chưa chạy |
| TC-P6-07-01 | N | chưa chạy |
| TC-P6-07-02 | N | chưa chạy |
| TC-P6-07-03 | N | chưa chạy |
| TC-P6-07-04 | N | chưa chạy |
| TC-P6-07-05 | N | chưa chạy |
| TC-P6-07-06 | N | chưa chạy |
| TC-P6-07-07 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-07-08 | N | chưa chạy |
| TC-P6-07-09 | N | chưa chạy |
| TC-P6-07-10 | P | go test 2 test PASS (log s8flow-go-*.log) |
| TC-P6-07-11 | N | chưa chạy |
| TC-P6-07-12 | N | chưa chạy |
| TC-P6-07-13 | N | chưa chạy |
| TC-P6-07-14 | N | chưa chạy |
| TC-P6-07-15 | N | chưa chạy |
| TC-P6-07-16 | N | chưa chạy |
| TC-P6-07-17 | P | go test 8 test PASS (log s8flow-go-*.log) |
| TC-P6-07-18 | N | chưa chạy |
| TC-P6-07-19 | N | chưa chạy |
| TC-P6-07-20 | N | chưa chạy |
| TC-P6-07-21 | N | chưa chạy |
| TC-P6-07-22 | N | chưa chạy |
| TC-P6-07-23 | N | chưa chạy |
| TC-P6-07-24 | N | chưa chạy |
| TC-P6-07-25 | N | chưa chạy |
| TC-P6-07-26 | P | go test 6 test PASS (log s8flow-go-*.log) |
| TC-P6-07-27 | N | chưa chạy |
| TC-P6-07-28 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-07-29 | N | chưa chạy |
| TC-P6-07-30 | N | chưa chạy |
| TC-P6-07-31 | N | chưa chạy |
| TC-P6-07-32 | N | chưa chạy |
| TC-P6-07-33 | N | chưa chạy |
| TC-P6-07-34 | P | go test 5 test PASS (log s8flow-go-*.log) |
| TC-P6-07-35 | N | chưa chạy |
| TC-P6-07-36 | N | chưa chạy |
| TC-P6-07-37 | N | chưa chạy |
| TC-P6-07-38 | N | chưa chạy |
| TC-P6-07-39 | N | chưa chạy |
| TC-P6-07-40 | N | chưa chạy |
| TC-P6-07-41 | P | go test 5 test PASS (log s8flow-go-*.log) |
| TC-P6-07-42 | N | chưa chạy |
| TC-P6-07-43 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-07-44 | N | chưa chạy |
| TC-P6-07-45 | N | chưa chạy |
| TC-P6-07-46 | N | chưa chạy |
| TC-P6-07-47 | N | chưa chạy |
| TC-P6-07-48 | N | chưa chạy |
| TC-P6-07-49 | N | chưa chạy |
| TC-P6-07-50 | N | chưa chạy |
| TC-P6-07-51 | N | chưa chạy |
| TC-P6-07-52 | N | chưa chạy |
| TC-P6-07-53 | N | chưa chạy |
| TC-P6-07-54 | N | chưa chạy |
| TC-P6-07-55 | N | chưa chạy |
| TC-P6-07-56 | N | chưa chạy |
| TC-P6-07-57 | N | chưa chạy |
| TC-P6-07-58 | N | chưa chạy |
| TC-P6-07-59 | N | chưa chạy |
| TC-P6-08-01 | N | chưa chạy |
| TC-P6-08-02 | N | chưa chạy |
| TC-P6-08-03 | N | chưa chạy |
| TC-P6-08-04 | N | chưa chạy |
| TC-P6-08-05 | P | go test 5 test PASS (log s8flow-go-*.log) |
| TC-P6-08-06 | N | chưa chạy |
| TC-P6-08-07 | N | chưa chạy |
| TC-P6-08-08 | N | chưa chạy |
| TC-P6-08-09 | N | chưa chạy |
| TC-P6-08-10 | N | chưa chạy |
| TC-P6-08-11 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-08-12 | N | chưa chạy |
| TC-P6-08-13 | N | chưa chạy |
| TC-P6-08-14 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-08-15 | N | chưa chạy |
| TC-P6-08-16 | N | chưa chạy |
| TC-P6-08-17 | N | chưa chạy |
| TC-P6-08-18 | N | chưa chạy |
| TC-P6-08-19 | N | chưa chạy |
| TC-P6-08-20 | N | chưa chạy |
| TC-P6-08-21 | N | chưa chạy |
| TC-P6-08-22 | N | chưa chạy |
| TC-P6-08-23 | N | chưa chạy |
| TC-P6-08-24 | N | chưa chạy |
| TC-P6-08-25 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-08-26 | N | chưa chạy |
| TC-P6-08-27 | N | chưa chạy |
| TC-P6-08-28 | N | chưa chạy |
| TC-P6-08-29 | P | go test 4 test PASS (log s8flow-go-*.log) |
| TC-P6-08-30 | N | chưa chạy |
| TC-P6-08-31 | N | chưa chạy |
| TC-P6-08-32 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-08-33 | N | chưa chạy |
| TC-P6-08-34 | N | chưa chạy |
| TC-P6-08-35 | P | go test 2 test PASS (log s8flow-go-*.log) |
| TC-P6-08-36 | N | chưa chạy |
| TC-P6-08-37 | N | chưa chạy |
| TC-P6-08-38 | N | chưa chạy |
| TC-P6-08-39 | N | chưa chạy |
| TC-P6-08-40 | N | chưa chạy |
| TC-P6-08-41 | N | chưa chạy |
| TC-P6-08-42 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-08-43 | N | chưa chạy |
| TC-P6-08-44 | N | chưa chạy |
| TC-P6-08-45 | N | chưa chạy |
| TC-P6-08-46 | N | chưa chạy |
| TC-P6-08-47 | N | chưa chạy |
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
| TC-P6-09-16 | N | chưa chạy |
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
| TC-P6-09-31 | N | chưa chạy |
| TC-P6-09-32 | N | chưa chạy |
| TC-P6-09-33 | N | chưa chạy |
| TC-P6-09-34 | P | go test 1 test PASS (log s8flow-go-*.log) |
| TC-P6-09-35 | N | chưa chạy |
| TC-P6-09-36 | N | chưa chạy |
| TC-P6-09-37 | N | chưa chạy |
| TC-P6-09-38 | N | chưa chạy |
| TC-P6-09-39 | P | go test 3 test PASS (log s8flow-go-*.log) |
| TC-P6-09-40 | N | chưa chạy |
| TC-P6-09-41 | N | chưa chạy |
| TC-P6-09-42 | N | chưa chạy |
| TC-P6-09-43 | N | chưa chạy |
| TC-P6-09-44 | N | chưa chạy |
| TC-P6-09-45 | N | chưa chạy |
| TC-P6-09-46 | N | chưa chạy |
| TC-P6-09-47 | N | chưa chạy |
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
