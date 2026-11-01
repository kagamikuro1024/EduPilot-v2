# DEV handoff — US-P6-02 (Grade Engine `internal/grade`)
Nhánh `sprint/8-p6`. Luật tính điểm theo SRS 4.3 + TLR-1, 2, 9, 13, 17 (chủ dự án chốt).

## Đã làm
- `engine.go`: `Compute(Scheme, Input) Result` (hàm thuần), `WhatIf`, `MinFinalFor` — cùng một `compute`. `DivRound(·, 20)` mọi phép chia; `HP_thô.Round(10)` ngay trước làm tròn; `HP = HP_thô.DivRound(step).Round(0).Mul(step)` (không hằng 0,5); nhóm trọng số 0 bỏ số hạng; `applicable=false` → B/P = 0 + cảnh báo `BONUS_NO_EFFECT` / `BONUS_NOT_IN_SCHEME` / `PENALTY_NOT_IN_SCHEME`; thiếu điểm → `INCOMPLETE` (kèm `MissingItems` lý do), 0 chỉ khi `zero_components` **và** `Item.Closed`; công thức thiếu `ROUNDING` / `LETTER_SCALE` → `INVALID` (không đoán). `Lines[]` đủ chữ số, dấu phẩy, `expr` chỉ gồm số đã hiển thị (cú pháp `+ − × ÷ ( ) min max round`).
- `rules.go`: parse tham số 6 loại luật, tham số lạ / thiếu / sai kiểu = `ParamError` (không mặc định thầm lặng), số là chuỗi.
- Quyết định nhỏ: `Result.Verdict` `INCOMPLETE:<key>+<key>` theo thứ tự thành phần; `LIET:final<{value}` dùng chữ gốc của `value` ("3.0"). Khi ghi thiếu cuối kỳ mà `w_CK = 0` thì không INCOMPLETE (TLR-9).

## Kiểm (`go test ./internal/grade`, `golangci-lint` 0 issue)
- `TestEngineExpected30`: **30/30** khớp `seed/expected_final_grades.csv` (hp, letter, verdict, qt, qt_before_clamp, hp_raw, bonus, penalty; `decimal.Equal`). CSV không bị sửa.
- `TestEngineCases`: 18/18 ca `cases-tong-hop.csv` (kể cả S16 chia không hết, S13–S15, S17, S18).
- `TestRoundHalfUpTable`, `TestNoIntermediateRounding`, `TestDivRoundNoDrift`, `TestMissingScore*`, `TestExpectedStudentC`, `TestFailConditions`, `TestFinalExactlyThreeNotFail`, `TestZeroWeightGroupNoDivide`, `TestBonusNoEffectWarning`, `TestPenaltyNotApplicable`, `TestPenaltyNotInSchemeWarning`, `TestBonusNotApplicable`, `TestParamsDriveResult`, `TestMissingParamIsError`, `TestExplain{OrderAndQuotes,UserFilledHasNoQuote,ExprRecomputesValue,FullPrecision}` (cột `explanation` của đáp án tính lại ra `hp_raw` 30/30), `TestWhatIfNoWrite`, `TestMinFinalFor`.
- Kiến trúc: `TestNoFloatInGrade` (AST), `TestForbiddenImports` (`go list -deps`), `TestEngineIsPure`, `TestNoPolicyConstants`.
- `BenchmarkCompute1000`: ≈ 13 ms / 1.000 sinh viên (ngưỡng 200 ms).
- AC10 `TestPreviewUsesCompute` và AC11 `TestPreviewRBAC`: chưa có route xem trước → làm ở US-P6-06 (route `gradebook/preview`), cùng `TestRBACMatrix`.

## Cần hỏi
Không.
