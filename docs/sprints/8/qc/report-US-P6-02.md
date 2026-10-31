# QC report — US-P6-02 (Grade Engine `internal/grade`: đúng tuyệt đối, decimal, giải trình) · Kết luận: **FAIL** (chỉ vì AC11 và nửa "xem trước cả lớp" của AC10 "KHÔNG KIỂM ĐƯỢC": chưa có route; mọi phần của engine PASS)
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `12fa90bf`. Bộ TC: `tc-US-P6-02.md` (không sửa chữ TC). Handoff: `dev-US-P6-02.md`. Đáp án độc lập: `seed/expected_final_grades.csv` + `fixtures/` (không ai sửa: `git diff 4c86d357.. -- seed/expected_final_grades.csv docs/specs/FEAT-gradebook/fixtures` chỉ có commit của BA).

Luật QC: "KHÔNG KIỂM ĐƯỢC" = FAIL. Dev ghi rõ ở handoff: "AC10 `TestPreviewUsesCompute` và AC11 `TestPreviewRBAC`: chưa có route xem trước → làm ở US-P6-06". Đề nghị PM chấp nhận `proposals.md` #8 (đã mở rộng cho story này): QC chấm lại hai phần đó ở US-P6-06, story coi PASS có điều kiện.

## 1. Cổng nghiệm thu đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `go test -race -count=1 -v ./internal/grade -run TestEngineExpected30` + đếm `--- PASS: TestEngineExpected30/` | PASS, đúng **30** ca con (không `SKIP`, không `DATA RACE`) |
| `go test -v ./internal/grade -run TestEngineCases` + đếm `--- PASS: TestEngineCases/` | PASS, đúng **18** ca con (S01…S18) |
| `p6-go-tests.sh ./internal/grade "<28 test của US-P6-02>" -race` | PASS 28/28 (không thiếu, không SKIP) |
| `! grep -rnE 'float(32\|64)\|math\.(Round\|Floor\|Ceil)\|ParseFloat\|RoundBank\|StringFixedBank' backend-go/internal/grade --include=*.go \| grep -v _test.go` | PASS rc=0 (lệnh cổng P6) |
| `grep -rn 'Sscanf\|"%f"'` ở mã không-test của `internal/grade` | rỗng |
| `go list -deps ./internal/grade \| grep -E 'internal/(activity\|crm\|llm\|grading\|privacy\|agent)\|openai-go'` | rỗng (rc=1) |
| `go doc ./internal/grade Compute` | `func Compute(s Scheme, in Input) Result` — không `context` |
| `grep -rn study_time_daily internal/grade internal/store/queries/grade*.sql` | rỗng (sau `7df6eff3`); `go test ./internal/activity -run 'TestGradeNeverImportsActivity\|TestGradeQueriesNeverReadStudyTime'` PASS |
| `go test -bench Compute1000 -benchtime 20x ./internal/grade` | PASS: `13,98 ms/op` cho 1.000 sinh viên (ngưỡng 200 ms) |
| `python3 docs/sprints/8/qc/scripts/p6-hand-calc.py A B C sv04 sv05 sv06 sv13 sv09 sv20` | PASS rc=0: `30/30` và `18/18` khớp — tính độc lập bằng `fractions`, không dùng script / mã của BA hay dev |
| QC probe (`scripts/p6_qc_probe_test.go.txt`, chạy tạm trong gói rồi xoá): `WhatIf` / `MinFinalFor` | PASS: B + CK 8,0 → HP thô **8,1**, HP **8,0**, **B+**; đầu vào không bị ghi; `MinFinalFor` B→A **8,25**, B→A+ **9,09**, B→B **5,75**, A→A+ **7,92**, C→A+ "không đạt được" — đều đúng các số QC tính tay trước (TC-P6-02-37/38) |
| `go vet ./internal/grade/...` | PASS |

## 2. Thử đột biến (bản sao mã `/tmp/qc-mut`, đã xoá; test của dev có bảo vệ hành vi không?)
| Đột biến | Kỳ vọng | Kết quả |
| --- | --- | --- |
| M1: làm tròn nửa-chẵn (`RoundBank`) thay nửa-lên | đỏ | **đỏ**: 20 ca (sv04, sv08, sv10, sv26, sv30, …) |
| M2: bỏ trần điểm cộng | đỏ | **đỏ**: sv.gioi, `S03-tran-cong-1.25`, `TestExplainExprRecomputesValue` |
| M3: miễn vắng thêm 1 buổi (`Absent − Free − 1`) | đỏ | **đỏ**: sv.nguyco, S05, S07, S08 (8 ca) |
| M4: làm tròn trung gian (`QT.Round(1)`) | đỏ | **đỏ**: 30/30 dòng |
| M5: liệt khi CK ≤ 3,0 (thay `<`) | đỏ | **đỏ**: sv23, sv25, `TestFinalExactlyThreeNotFail` |
| M6: thêm `var _ = float64(1)` vào gói | đỏ | **đỏ** (`TestNoFloatInGrade`) |
| M7: `import _ ".../internal/activity"` vào gói | đỏ | **đỏ** (`TestForbiddenImports`) |
| M9: bỏ chuẩn hoá `HP_thô.Round(10)` | đỏ (TLR-1) | **xanh** — xem BUG-P6-02-1 |
| M10: `DivPrecision` 20 → 16 | đỏ (TLR-1) | **xanh** — xem BUG-P6-02-1 |
| M11: chia cho bước làm tròn với độ chính xác 2 | đỏ | **đỏ** (2 test) |

## 3. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 đáp án BA đúng | PASS | `p6-hand-calc.py` rc=0, 30/30 + 18/18; bảng phép tính tay A, B, C, sv04, sv05, sv06, sv13, sv09, sv20, sv23 ở đầu `tc-US-P6-02.md` khớp từng bước |
| 02 30 ca | PASS | 30 ca con PASS |
| 03 30/30 qua API `preview` | KHÔNG KIỂM ĐƯỢC (chưa có route) | làm ở US-P6-06 / US-S8-SEED |
| 04 màn `/gradebook` A, B, C | KHÔNG KIỂM ĐƯỢC (chưa có màn thật) | làm ở US-P6-06 |
| 05–09 đột biến M1–M5 | PASS | bảng mục 2 |
| 10 18 ca | PASS | 18 ca con PASS |
| 11 làm tròn x,25/x,75/x,5/x,249/x,251, bước 0,1 / 1 | PASS | `TestRoundHalfUpTable`, `TestNoIntermediateRounding`, `TestDivRoundNoDrift` PASS; M1, M4, M11 làm đỏ |
| 12 `impact` đổi bước | KHÔNG KIỂM ĐƯỢC (chưa có route `impact`) | US-P6-04 |
| 13 S16 → 6,5 | PASS | `TestEngineCases/S16-chia-khong-het` PASS, `hp_raw 6,25`, `hp 6,5` |
| 14–15 không `float` | PASS | greps rỗng; chỉ chữ "float" nằm ở chú thích `types.go:2` (không khớp mẫu cổng) |
| 16 JSON điểm là chuỗi | KHÔNG KIỂM ĐƯỢC (chưa có API) | US-P6-06 |
| 17–18 lint, đột biến M6 | PASS | `go vet` xanh; M6 đỏ |
| 19–21 không import cấm, hàm thuần, M7 | PASS | `go list -deps` rỗng; chữ ký `Compute` không `context`; M7 đỏ |
| 22 hai công thức trên cùng engine | PASS ở mức engine (`TestParamsDriveResult`); qua `impact`: chờ US-P6-04 | |
| 23 mọi tham số từ `params`, không hằng quy chế | PASS | `TestParamsDriveResult`, `TestMissingParamIsError`, `TestNoPolicyConstants`; QC đọc từng dòng khớp `\b(0\.4\|0\.6\|0\.25\|0\.5\|3\.0\|7)\b` ở mã không-test: toàn chú thích / ví dụ định dạng, và `roundingSteps` = tập bước cho phép `{0,01; 0,1; 0,25; 0,5; 1}` do chính `SRS.md` 4.2.4 quy định (không phải hằng quy chế) |
| 24 thiếu `params` → lỗi rõ | PASS ở mức engine (`TestMissingParamIsError`: thiếu / lạ / sai kiểu = `ParamError`); `impact` API: chờ US-P6-04 | |
| 25 sinh viên C | PASS | `TestExpectedStudentC`; `qt 1,25`, `hp 3,0`, F không liệt; "không dự → 0 theo quy chế" trong `Lines` |
| 26–28 thiếu điểm → `INCOMPLETE` | PASS | `TestMissingScoreIncomplete`, `…ZeroRule`, `…ZeroOnlyWhenSourceClosed`; qua API/preview: chờ US-P6-06 |
| 29–31 liệt | PASS | `TestFailConditions`, `TestFinalExactlyThreeNotFail`; M5 đỏ |
| 32–35 giải trình | PASS ở mức engine | `TestExplain{OrderAndQuotes,ExprRecomputesValue,FullPrecision,UserFilledHasNoQuote}` PASS; M2 làm `ExprRecomputesValue` đỏ; qua `…/explain`: chờ US-P6-06 |
| 36 hiệu năng | PASS | 13,98 ms / 1.000 sinh viên |
| 37–38 `WhatIf`, `MinFinalFor` | PASS | QC probe, mọi số khớp tính tay |
| 39 một engine duy nhất | PASS một nửa | engine chỉ có `Compute` (`WhatIf`, `MinFinalFor` dùng chung); `TestPreviewUsesCompute` chưa có (xem trên) |
| 40–41 nhóm trọng số 0, `applicable=false` | PASS | 5/5 test; `impact` w_QT = 0 qua API: chờ US-P6-04 |
| 42 `FINAL_BELOW` khi `w_CK = 0` | PASS ở mức engine (không `INCOMPLETE`, TLR-9); mã `PARAM_INVALID` của bộ kiểm: tc-US-P6-03 |
| 43–45 nhánh lỗi | PASS ở mức engine; API: chờ route |
| 46–49 phân quyền | KHÔNG KIỂM ĐƯỢC (chưa có route) | `TestPreviewRBAC` chưa có |

## 4. AC
| AC | Kết quả |
| --- | --- |
| AC1 30/30 | PASS |
| AC2 18 ca | PASS |
| AC3 làm tròn | PASS |
| AC4 decimal, cấm float | PASS (M6 đỏ) |
| AC5 không biết thời gian học / LLM, hàm thuần | PASS (M7 đỏ) |
| AC6 mọi quy tắc từ `params` | PASS |
| AC7 thiếu điểm không bịa 0 | PASS |
| AC8 điều kiện liệt | PASS |
| AC9 giải trình có căn cứ, `expr` tính lại ra `value` | PASS ở mức engine (qua API: US-P6-06) |
| AC10 xem trước / what-if cùng engine | **PASS một nửa:** `WhatIf`, `MinFinalFor`, hiệu năng PASS; `TestPreviewUsesCompute` (route) KHÔNG KIỂM ĐƯỢC |
| AC11 phân quyền | **KHÔNG KIỂM ĐƯỢC → FAIL** (chưa có route) |
| AC12 nhóm trọng số 0, luật không áp dụng | PASS |

## 5. Lỗi
- **BUG-P6-02-1 (Thấp, khoảng trống bảo vệ):** hai đột biến M9 (bỏ `HP_thô.Round(10)`) và M10 (`DivPrecision` 20 → 16) đều **không làm đỏ test nào**, kể cả cặp M9+M10 cùng lúc. `US.md` US-P6-02 AC3 / TLR-1 yêu cầu chính hai điều này vì "làm tròn trung gian ở 16 chữ số có thể ra 6,0". Ca S16 như dựng không tái hiện được lỗi nên `TestDivRoundNoDrift` / `TestEngineCases` không bảo vệ chúng; sau này ai đổi hằng số hoặc bỏ dòng chuẩn hoá sẽ không bị test bắt. **Hành vi hiện tại đúng** (`DivPrecision = 20`, `Round(NormalizeDigits=10)` có mặt, `engine.go:307, 310`). Đề nghị dev **thêm** (không sửa test cũ) một ca có đầu vào làm lộ sai số khi chia ở 16 chữ số (ví dụ item `10/30` kết hợp trọng số cho HP thô 6,25 mà chia 16 chữ số cho 6,2499… ) hoặc ghi rõ ở handoff vì sao không dựng được. Không chặn story.
- Không phát hiện lỗi tính điểm nào: 30/30, 18/18, 5 ca QC tự tính (A, B, C, hai ca biên x,25 / x,75: sv04 6,25 → 6,5; sv05 6,75 → 7,0), sv13 x,5, sv09 ba chữ số, sv20 liệt giữ số: khớp.

## 6. Kiểm phản mẫu / luật mở rộng / phân quyền
- Không route / UI mới ở story này: `ui-antipatterns` không áp dụng; không `fetch` trần; không thư viện mới (`go.mod` không đổi).
- Test cũ: không sửa thêm (chỉ `db/migrations_test.go` ở US-P6-01, chờ PM #7). `seed/expected_final_grades.csv`, `fixtures/`: không bị dev sửa.
- Không PII / secret.

## Đề nghị
1. PM chấp nhận `proposals.md` #8 (đã mở rộng cho US-P6-02): QC chấm lại AC10 (route) + AC11 ở US-P6-06; story coi PASS có điều kiện.
2. Dev: thêm một ca bảo vệ TLR-1 (BUG-P6-02-1) khi tiện; không chặn.
