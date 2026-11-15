# s8core: ca chỉ gồm lệnh Go test đã PASS (tất cả test nêu tên đều PASS trong logs/s8core-go-*.log; 02-02/10/36 từ logs/s8core-go2.log)
from s8core_lib import *
G = "(Go integration -tags integration, logs/s8core-go-*.log)"
for tc, n in [
 ("TC-P6-02-02", "TestEngineExpected30 -race: PASS, đúng 30 ca con, 0 SKIP, 0 DATA RACE (logs/s8core-go2.log)"),
 ("TC-P6-02-10", "TestEngineCases -race: PASS, đúng 18 ca con S01–S18, 0 SKIP"),
 ("TC-P6-02-11", "TestRoundHalfUpTable/NoIntermediateRounding/DivRoundNoDrift PASS; ca x,25 x,75 x,5 bước 0,5/0,1/1 (M1 làm đỏ)"),
 ("TC-P6-02-13", "TestDivRoundNoDrift + TestEngineCases/S16-chia-khong-het PASS (hp_raw 6,25 → 6,5)"),
 ("TC-P6-02-20", "TestForbiddenImports/TestEngineIsPure PASS; grep study_time_daily rỗng; Compute(s Scheme, in Input) không context"),
 ("TC-P6-02-23", "TestParamsDriveResult/MissingParamIsError/NoPolicyConstants PASS; grep hằng: chỉ chú thích, 7*3600 múi giờ, roundingSteps (tập bước SRS 4.2.4)"),
 ("TC-P6-02-28", "TestMissingScoreIncomplete/ZeroRule/ZeroOnlyWhenSourceClosed/ExpectedStudentC PASS"),
 ("TC-P6-02-31", "TestFailConditions/TestFinalExactlyThreeNotFail PASS"),
 ("TC-P6-02-35", "TestExplainOrderAndQuotes/ExprRecomputesValue/FullPrecision/UserFilledHasNoQuote PASS"),
 ("TC-P6-02-40", "TestZeroWeightGroupNoDivide/BonusNoEffectWarning/PenaltyNotApplicable/PenaltyNotInSchemeWarning/BonusNotApplicable PASS"),
 ("TC-P6-02-49", "TestPreviewRBAC PASS " + G),
 ("TC-P6-03-05", "TestExtractOneStructuredCall/TaskAndLane/PromptHasPages/SchemaStrictCompatible/AttemptTimeout PASS"),
 ("TC-P6-03-09", "TestExtractGradeScheme PASS"),
 ("TC-P6-03-15", "TestExtractLop2OneUnclear PASS"),
 ("TC-P6-03-16", "8 test TestChecker* PASS (WeightSum, NoComponents, Bonus, BonusCap, Absence, Rounding, LetterScale, ParamInvalid)"),
 ("TC-P6-03-24", "TestCitationVerified/NotFound/PageMissing/OCRTolerance/ComparedAfterUnmask PASS; không có hàm vnFold tự viết ở Go (vn_fold dùng ở SQL/textfold)"),
 ("TC-P6-03-27", "TestExtractNeverConfirms (grading) + TestConfirmOnlyFromHandler (grade) PASS"),
 ("TC-P6-03-29", "TestPromptDelimitsDocument/NoInstructionFollowing/ExtractUsesPrivacyMask PASS"),
 ("TC-P6-03-43", "12 test TestExtract* (NotPolicy…ReplaceDraftVersionChanged) PASS " + G),
 ("TC-P6-03-48", "TestExtractRBAC PASS"),
 ("TC-P6-04-28", "TestConfirmCarryOverEntries/NoCarryOnMaxScoreChange/NoCarryOnSourceChange/SupersedeBeforeConfirm/CreatesGradebookRow/BlockedWhenLocked PASS"),
]: rec(tc, "P", n)
