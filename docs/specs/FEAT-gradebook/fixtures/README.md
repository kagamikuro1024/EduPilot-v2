# Đáp án độc lập của sổ điểm (FEAT-gradebook)

Chủ sở hữu: **BA** (sprint 8). Dev và QC **không sửa** các tệp ở đây và `seed/expected_final_grades.csv`; lệch giữa mã và đáp án → sửa mã, hoặc dừng và ghi `proposals.md`. Dữ liệu mô phỏng hoàn toàn (D44).

| Tệp | Dùng để |
| --- | --- |
| `quy-che-lop1.md` | Nội dung gốc của `seed/demo/quy-che-lop1.pdf` (COURSE_POLICY lớp 1). Dev dựng PDF, không đổi chữ |
| `quy-che-lop2.md` | Nội dung gốc của `seed/demo/quy-che-lop2.pdf` (lớp 2, **cố ý thiếu quy tắc làm tròn**) |
| `extraction-lop1.json`, `extraction-lop2.json` | Kết quả `ExtractGradeScheme` **mong đợi sau bộ kiểm bằng code** cho hai quy chế. Bản ghi sẵn của provider giả (`seed/demo/fake_llm.json`) là các tệp này, bỏ các mục `unclear_items` có `source = "CODE"` (bộ kiểm tự thêm). Mọi `quote` là câu nguyên văn trong `quy-che-lopN.md`, đúng số trang |
| `inputs-lop1.csv` | Đầu vào điểm của 30 sinh viên lớp 1 (khoá `account`) |
| `../../../../seed/expected_final_grades.csv` | **Đáp án**: điểm học phần từng sinh viên + cột giải trình |
| `cases-tong-hop.csv` | 18 ca biên mà 30 sinh viên seed không chạm tới (sàn QT, trần cộng, ngưỡng vắng, liệt vì vắng, thiếu cuối kỳ, làm tròn 0,1, số 0 theo `MISSING_SCORE` chỉ khi nguồn đã chốt — TLR-2, phép chia không hết — TLR-1, nhóm trọng số 0 — TLR-9, `PENALTY` không áp dụng — TLR-17). Có thêm cột `w_*`, `midterm_max`, `weekly_state`, `penalty_rule` để đổi công thức từng ca |
| `../../../../seed/expected_risk.csv` | Đáp án at-risk của lớp 1 **sau khi có tín hiệu `GRADE`** (#5): điểm quá trình hiện tại từng sinh viên, `GRADE` thấp hay không, mức rủi ro mong đợi, ràng buộc hoạt động tuần 8 / 9 |
| `tinh_diem_tham_chieu.py` | Bảng tính tham chiếu của BA (decimal, không float). `python3 tinh_diem_tham_chieu.py` in kết quả; `--write` ghi hai CSV đáp án. Dev **không** import, không chạy trong CI |

## Quy chế lớp 1 → tham số (khớp `extraction-lop1.json`)

| Mục | Giá trị | Điều |
| --- | --- | --- |
| Trọng số (% điểm học phần) | Giữa kỳ 20 · Kiểm tra hằng tuần 20 · Cuối kỳ 60 (QT = 40) | 1.1–1.5 |
| Kiểm tra hằng tuần | bài trắc nghiệm đã công bố (`Kiểm tra tuần 9 — Mật mã`); bài lập trình **không** tính; không dự = **0 chỉ khi bài thi đã `PUBLISHED`** mà sinh viên không có lượt `GRADED`; bài chưa công bố → chưa đủ điểm (TLR-2) | 1.3, 1.4 |
| Điểm cộng B | Σ điểm `SPEAK` + `BONUS` ghi ở P5 (mặc định 0,25 / lần), tối đa **1,0** | 2.1 |
| Điểm trừ P | **0,5** × (số buổi `ABSENT` − 2) nếu dương, cộng `|Σ PENALTY|`; `EXCUSED` và `LATE` không tính | 2.2, 2.3 |
| Làm tròn | đến **0,5**, nửa lên (x,25 → x,5 · x,75 → x+1,0 · x,5 giữ nguyên); trước bước này `HP thô` được chuẩn hoá về 10 chữ số thập phân và mọi phép chia dừng ở 20 chữ số (TLR-1: chuẩn hoá độ chính xác máy, không phải làm tròn nghiệp vụ) | 3.1 |
| Thang chữ (trên điểm đã làm tròn) | A+ ≥ 9,0 · A ≥ 8,5 · B+ ≥ 8,0 · B ≥ 7,0 · C+ ≥ 6,5 · C ≥ 5,5 · D+ ≥ 5,0 · D ≥ 4,0 · F | 3.2 |
| Liệt (điểm chữ F) | cuối kỳ < 3,0, hoặc vắng không phép ≥ 7 buổi | 4.1, 4.2 |
| Điểm quá trình | luôn nằm trong [0; 10] (tối đa 10 theo công thức PRD M7; sàn 0 là mặc định ở QUESTIONS Q4) | – |

## Công thức

$$QT_0=\frac{20\cdot GK+20\cdot KT}{40},\qquad QT=\min\bigl(10,\max(0,\ QT_0+\min(B,1{,}0)-P)\bigr)$$
$$HP_{thô}=\frac{40\cdot QT+60\cdot CK}{100},\qquad HP=\text{làm tròn đến 0,5, nửa lên}(HP_{thô})$$

`GK` giữa kỳ, `KT` điểm trắc nghiệm tuần 9 (0 nếu không dự), `CK` cuối kỳ; điểm thành phần thang 10. Điểm chữ tính trên `HP`. Liệt → điểm chữ F, `HP` vẫn hiển thị số đã tính kèm lý do.

## Nguồn của `inputs-lop1.csv`

| Cột | Lấy từ |
| --- | --- |
| `account`, `absent`, `late`, `speak_count`, `speak_bonus_points` | `seed/crm_plan.json` (`attendance`, `speak`; điểm = 0,25 × số lần phát biểu, khớp `seed/expected_crm.csv` cho A, B, C) |
| `weekly_exam_t9` | `seed/expected_exam_scores.csv`, dòng `MCQ` (`expected_score`); **trống** = bài thi tuần 9 đã công bố và sinh viên không có lượt làm → 0 theo quy chế (C và `sv26`…`sv30`; TLR-2) |
| `student_code` (MSSV) | A, B, C cố định `20229001`–`20229003`; còn lại sinh bởi `scripts/seed.mjs` với `SEED_RNG=20261029` (giá trị mặc định). Đổi `SEED_RNG` thì MSSV đổi: so khớp bằng `account`, không bằng MSSV. Khi nhập XLSX, hệ thống khớp theo **`enrollments.student_code_snapshot`** (danh sách lớp), không theo MSSV tự khai ở `users` (TLR-3) |
| `midterm`, `final_exam` | **BA định nghĩa** (không có nguồn khác), chọn để sinh các ca biên dưới đây |
| `penalty_event_points` | 0,00 (seed không có sự kiện `PENALTY`) |

## Các ca biên có trong 30 dòng

| Ca | Sinh viên |
| --- | --- |
| Chạm trần điểm cộng (3,00 → 1,00) **và** trần QT 10 (10,500 → 10) | `sv.gioi` (A) |
| Điểm trừ vắng: 5 buổi → trừ 1,5 · không dự kiểm tra tuần (0 điểm) | `sv.nguyco` (C) |
| Vắng đúng 2 buổi (miễn trừ) | `sv.kha` (B) |
| Làm tròn nửa lên đúng x,25 → x,5 (chẵn-gần-nhất sẽ ra x,0): `HP` thô 6,25 · 8,25 · 5,25 · 3,25 · 7,25 | `sv04`, `sv08`, `sv10`, `sv26`, `sv30` |
| Làm tròn nửa lên đúng x,75: 6,75 · 7,75 · 8,75 · 4,75 · 3,75 · 9,75 | `sv05`, `sv12`, `sv07`, `sv15`, `sv25`, `sv06` |
| Đúng x,5 (giữ nguyên): 8,50 · 7,50 | `sv13`, `sv14` |
| Số lẻ ba chữ số do điểm trắc nghiệm 7,33 / 6,33 (chống `float64`): 8,066 · 6,466 · 4,916 · 8,566 | `sv09`, `sv11`, `sv20`, `sv22` |
| Sát dưới ngưỡng làm tròn: 6,20 → 6,0 · 6,70 → 6,5 · 4,20 → 4,0 · 5,40 → 5,5 | `sv16`, `sv17`, `sv24`, `sv21` |
| Ngưỡng điểm chữ sau làm tròn (9,0 / 8,5 / 8,0 / 7,0 / 6,5 / 6,0→C / 5,5 / 5,0 / 4,0 / 3,5→F) | nhiều dòng, xem cột `hp` và `letter` |
| Liệt vì cuối kỳ 2,75 (HP 5,0 nhưng điểm chữ F) | `sv20` |
| Cuối kỳ đúng 3,0 (**không** liệt) | `sv23`, `sv25` |
| Điểm kiểm tra tuần bằng 0 vì không dự (6 sinh viên) | `sv.nguyco`, `sv26`…`sv30` |

Không có sinh viên nào trong 30 chạm: vắng đúng 3 buổi, vắng ≥ 7 buổi, sàn QT = 0, trần cộng đúng 1,00, `PENALTY` thủ công, thiếu cuối kỳ, bước làm tròn 0,1, bài thi chưa công bố, item nhập tay trống, phép chia không hết, nhóm trọng số 0, `PENALTY` không áp dụng → đã phủ bằng `cases-tong-hop.csv` (S01…S18).

## Kiểm tay A, B, C (chủ dự án làm lại ở cổng)

**A** (`sv.gioi`): GK 9,00 · KT 10,00 · CK 9,50 · phát biểu 12 lần = 3,00 · vắng 0.
`QT₀ = (20×9 + 20×10)/40 = 9,50` → cộng `min(3,00; 1,00) = 1,00` → `10,50` → trần **10,00**. `HP thô = (40×10 + 60×9,5)/100 = 9,70` → làm tròn đến 0,5 = **9,5** → **A+**.

**B** (`sv.kha`): GK 7,50 · KT 8,00 · CK 6,50 · phát biểu 2 lần = 0,50 · vắng 2 · muộn 1.
`QT₀ = (20×7,5 + 20×8)/40 = 7,75` → cộng 0,50 → trừ 0 (vắng 2 ≤ 2) → `8,25`. `HP thô = (40×8,25 + 60×6,5)/100 = 7,20` → làm tròn đến 0,5 = **7,0** → **B**.

**C** (`sv.nguyco`): GK 5,00 · KT không dự → 0 · CK 4,50 · phát biểu 1 lần = 0,25 · vắng 5 · muộn 1.
`QT₀ = (20×5 + 20×0)/40 = 2,50` → cộng 0,25 → trừ `0,5 × (5 − 2) = 1,50` → `1,25`. `HP thô = (40×1,25 + 60×4,5)/100 = 3,20` → làm tròn đến 0,5 = **3,0** → **F** (dưới 4,0; không liệt: cuối kỳ 4,5 ≥ 3,0 và vắng 5 < 7).

Giải trình bằng lời cho từng dòng nằm ở cột `explanation` của `seed/expected_final_grades.csv`: **mọi số in đủ chữ số, không làm tròn** (bỏ số 0 thừa; TLR-5) nên tính lại từ chuỗi ra đúng `hp_raw`. Kiểm chéo: 30/30 dòng đã được tính lại bằng phân số (`fractions`) độc lập với script và khớp; Tech Lead cũng tính lại độc lập (TL-REVIEW) và khớp.

## Cột của `seed/expected_final_grades.csv`

`account, student_code, midterm, weekly_exam_t9, final_exam, speak_bonus_points, bonus_applied, absent, penalty, qt_before_clamp, qt, hp_raw, hp, letter, verdict, explanation`. Số dùng dấu chấm; `weekly_exam_t9` trống = không dự (bài đã công bố → 0); `verdict` = `OK` hoặc `LIET:<lý do>`. `qt_before_clamp`, `qt` (3 chữ số) và `hp_raw` (4 chữ số) là số **chính xác** với dữ liệu này (script kiểm không mất chữ số nào). Test Go so `hp`, `letter`, `verdict`, `qt`, `hp_raw` bằng `decimal.Equal`, không bằng chuỗi.

## Ghi chú còn mở

Các con số trên bám quy chế mô phỏng này và ba quy ước chưa có trong `phases/P6.md` (sàn QT = 0 · thiếu điểm thành phần ra sao · liệt giữ số hay không): xem `QUESTIONS.md` các câu **[ĐIỂM]**. Nếu chủ dự án đổi, BA tính lại hai CSV bằng `--write` và PM báo dev; không sửa tay.
