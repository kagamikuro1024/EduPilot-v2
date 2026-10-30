# DEV handoff dở dang — US-P6-02 (Grade Engine) — WIP, dừng theo lệnh PM (restart máy)
Nhánh `sprint/8-p6`. US-P6-01 đã xong và push (`ddbc8948`, handoff `dev-US-P6-01.md`).

## Đã có (US-P6-02)
- `internal/grade/types.go`, `internal/grade/inputs.go` (từ US-P6-01).
- `internal/grade/rules.go`: parse tham số 6 loại luật (`ParseBonus/Penalty/Rounding/LetterScale/Fail/MissingScore`, `BuildRules`), lỗi `ParamError` (thiếu / lạ / sai kiểu — không mặc định thầm lặng). Build xanh, **chưa có test**.

## Chưa làm (theo thứ tự)
1. `internal/grade/engine.go`: `Compute(Scheme, Input) Result` (hàm thuần; SRS 4.3.2: `DivRound(·,20)`, `HP_thô.Round(10)` trước bước làm tròn, `HP = HP_thô.Div(step).Round(0).Mul(step)` không hằng `0.5`; nhóm trọng số 0 bỏ số hạng; `PENALTY/BONUS.applicable=false` → 0 + cảnh báo `*_NOT_IN_SCHEME`; thiếu điểm → `INCOMPLETE` trừ `zero_components` khi `Item.Closed`), `WhatIf`, `MinFinalFor`, `Lines[]` (số đủ chữ số, dấu phẩy; `expr` tính lại ra `value`; verdict `LIET:final<3.0` / `LIET:absent>=7` dùng `FailCondition.Text`).
2. Test: `TestEngineExpected30` (đọc `seed/expected_final_grades.csv` + `fixtures/inputs-lop1.csv` + `fixtures/extraction-lop1.json`), `TestEngineCases` (18 ca `fixtures/cases-tong-hop.csv`), `TestRoundHalfUpTable`, `TestNoFloatInGrade` (AST), `TestForbiddenImports` (`go list -deps`), `TestEngineIsPure`, `TestParamsDriveResult`, `TestMissingParamIsError`, `TestNoPolicyConstants`, explain tests (bộ đánh giá `expr` trong test), `BenchmarkCompute1000`.
3. Kế tiếp theo prompt: US-P6-03 → 05 → 04 → 06 → 07 → 08 → 09 → US-S8-SEED.

## Trạng thái
Test của US-P6-02 chưa viết, chưa chạy. CI chưa chạy cho commit WIP này.
