# Fixtures FEAT-assignments (đáp án độc lập của BA)

Dev và QC **không sửa** các tệp này; lệch → ghi `proposals.md` / `QUESTIONS.md`. Nếu BA đổi dữ liệu, chạy lại `python3 build_fixtures.py --write` rồi `python3 build_fixtures.py` (kiểm).

| Tệp | Nội dung | Dùng ở |
| --- | --- | --- |
| `de-bai-bt03.md` | Đề Bài tập 03 "Phân tích một vụ tấn công thực tế" | seed, US-S9-SEED AC1 |
| `dap-an-chuan-bt03.md` | Đáp án chuẩn (có canary `CANARY-BT03-ANSWER`) | prompt chấm; test canary không lộ ra sinh viên |
| `rubric-bt03.json` | 4 tiêu chí c1–c4, mỗi tiêu chí tối đa 2,5; Σ = 10; bước 0,25 | US-P7-02, `SuggestRubric` giả |
| `create-bt03.json` | Body `POST …/assignments` của Bài tập 03 (`due_at` ví dụ; seed đặt hạn đã qua theo `seed-plan.csv`; `id` tiêu chí do máy chủ cấp) | US-P7-02 AC2, seed |
| `submissions/s1…s7` | 7 mẫu bài làm (tốt, thiếu bằng chứng, tổng sai, trống, lạc đề, chèn lệnh, lệch hai lượt) | US-P7-06 |
| `expected_grading.json` | Điểm / cờ kỳ vọng của 7 mẫu (S7 = bài Sinh viên B: nháp 7,00 − 0,50 muộn = 6,50; sau giảng viên sửa tiêu chí 2 → 8,00 − 0,50 = 7,50) | US-P7-06, SEED AC2, AC5 |
| `expected_checks.json` | 13 ca kiểm một lượt (K01–K13) + 4 ca hai lượt (D01–D04) | US-P7-06 AC (kiểm bằng code) |
| `seed-plan.csv` | 30 sinh viên × Bài tập 03: mẫu, kênh, giờ so với hạn, khớp ban đầu, điểm công bố giả định | US-S9-SEED AC1, AC4, AC5 |
| `teams-plan.csv` | 30 bản ghi Graph (25 `submitted` gồm 1 người ngoài lớp, 3 `working`, 2 `returned`) | US-P7-11, mock-graph |
| `quy-che-lop1-v2.md`, `extraction-lop1-v2.json` | Quy chế bản 2 (bản nháp demo) + kết quả trích của provider giả | SEED AC4 |
| `build_fixtures.py` | Kiểm nhất quán + `--write` sinh lại số liệu | BA |
| `../../../../seed/expected_final_grades_v2.csv` | Điểm học phần 30 sinh viên theo quy chế v2 (22/30 khác bản 1) | SEED AC4 |

## Cách tính (tóm tắt)
- Điểm bài = Σ điểm tiêu chí (bước 0,25) − khoản trừ muộn; khoản trừ = `0,5 × min(⌈giờ trễ ÷ 24⌉, 2)`; không âm.
- Điểm học phần v2 = giữa kỳ 10 % + kiểm tra tuần 10 % + bài tập 20 % + cuối kỳ 60 % (+ cộng / trừ như bản 1, sinh viên không nộp bài tập = 0 theo `MISSING_SCORE`); đã kiểm chéo bằng số thập phân và phân số, 0 lệch.
- `seed/expected_final_grades.csv` (bản 1, sprint 8) **không đổi**; băm ở `seed/expected_final_grades.csv.sha256`.

## Ca biên
S4 (38 ký tự < 50) không gọi LLM; S6 mất câu chèn lệnh khỏi đầu vào; S2 trích dẫn bịa bị loại; S3 tổng LLM báo sai nên dùng Σ; sv12 nộp hai phiên bản (chỉ bản 2 được chấm); sv29, sv30 và một thư email lạ là `UNMATCHED`; sv08 nộp muộn 2 ngày (trừ 1,0); sv14 ở `partial_failure` của Teams là tệp hỏng.
