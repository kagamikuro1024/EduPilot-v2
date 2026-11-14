# QC test case — US-GD-07 (ca hồi quy, M2 — đụng dữ liệu cá nhân), US-GD-08 (câu tương tự Threads, M4), US-GD-09 (lý do loại câu hỏi, M6)
Nguồn: US.md + SRS 3.5–3.7, 4.3–4.5, 6 #10–12; Q-G-3, Q-G-5; P-106-1. Hộp đen; quy ước `tc-00-chung.md` (**embedding giả; không LLM thật**).

## US-GD-07 Ca hồi quy (theo tham chiếu)
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G07-01 | AC1 | `REGRESSION_CAPTURE` bật bởi ADMIN | Teacher `review` kết quả sửa tiêu chí c2 từ 2,0 → 3,5 (≥ 1,0); sửa c3 0,5 (< 1,0) | 1 hàng `GRADING_EDIT` (`source_id` = id kết quả, `criterion_id=c2`); **không** hàng cho c3 (< `REGRESSION_MIN_DELTA` 1,0); biên: lệch đúng 1,0 → ghi | API+DB |
| TC-G07-02 | AC1 | – | Phúc khảo `ADJUSTED`; Thread chuyển `CORRECTED`; `REJECTED` | `GRADING_APPEAL`, `THREAD_CORRECTED`, `THREAD_REJECTED` đúng `kind` / `source_id` | API+DB |
| TC-G07-03 | AC1 | Ca vừa ghi | `SELECT expected, to_jsonb(*)::text` các cột | `expected` chỉ số / enum (`ai_score`, `teacher_score`, `direction`, `rubric_hash` hoặc `state`); **không** chuỗi ≥ 40 ký tự; `grep` văn bản bài, câu hỏi, câu trả lời, tên, MSSV trong **toàn bảng** = 0; `TestRegressionCaseNoText` PASS | DB+GO |
| TC-G07-04 | AC1 | `REGRESSION_CAPTURE` tắt | Như TC-G07-01 | 0 hàng | API+DB |
| TC-G07-05 | AC2 | – | Sửa lại cùng tiêu chí 3,5 → 4,0; `review` nhiều lần | Một hàng (`UNIQUE`), `expected` cập nhật mới nhất; `TestRegressionCaseIdempotent` PASS | API+DB |
| TC-G07-06 | AC2 | – | Đo `review` bật / tắt cờ (30 lượt) | Không chậm đáng kể (chênh p95 ≤ 20 ms); ghi cùng giao dịch (kill giữa chừng → không ca mồ côi) | API |
| TC-G07-07 | AC3 | Stub LLM, ca có sẵn, `cmd/regress` (binary trong image hoặc `go run ./cmd/regress`) | `regress run --assignment $A` (ADMIN / môi trường đo) | In `{n, MAE, lệch TB, số tiêu chí xấu đi, số tốt lên}`; đọc ca + văn bản **trong DB**; chấm lại bằng prompt / model hiện hành qua stub / 9router; **không** ghi văn bản ra repo / log (`grep` log + `git status`) | TAY |
| TC-G07-08 | AC3 | – | `regress run` không phải ADMIN / không có env đo | Từ chối | TAY |
| TC-G07-09 | AC4 | `benchmarks/guidance/baseline.json` | Dựng kết quả: (a) MAE tăng 12 % (> 10 %); (b) MAE tăng 8 %; (c) một tiêu chí vốn đúng (|Δ| ≤ 0,25) nay sai 0,6; (d) sai 0,5 | (a) mã thoát ≠ 0 «Có lùi»; (b) 0; (c) ≠ 0; (d) 0 (> 0,5 mới tính) | TAY |
| TC-G07-10 | AC4 | CI | Đọc workflow | CI **không** chạy `regress run` thật; chỉ kiểm cấu trúc ca bằng provider `fake`; guide + checklist PR ghi lệnh | DIFF |
| TC-G07-11 | AC5 | `REGRESSION_EXPORT_ALLOWED=false` (mặc định) | `regress export` | Từ chối (mã ≠ 0, thông điệp rõ); không tệp nào tạo; `TestRegressCLIRefusesExport` PASS | TAY+GO |
| TC-G07-12 | AC5 | Chỉ khi PM cho phép (`true`) — **chờ phép PM** | `regress export` | «KHÔNG KIỂM ĐƯỢC (chờ phép PM)»; nếu có phép: JSONL **ngoài repo**, văn bản qua `Mask` + `Detect`, id băm, không `student_id`, ca còn PII bị bỏ; `audit_log` (ai, số ca); **không** đường HTTP xuất (thử `GET /…/regression*` → 404) | – |
| TC-G07-13 | AC6 | – | Teacher: `GET` số ca ở màn thống kê; teacher / ta / sv đọc / xuất ca | Teacher thấy **số ca** lớp mình, không nội dung; ADMIN đọc; TA, SV `403` (API nếu có) | API |
| TC-G07-14 | AC7 | Ca liên quan SV X; xoá / ẩn danh SV X (job xoá, PR) | Chạy job xoá | Ca tham chiếu bị xoá, không còn `source_id` mồ côi (`LEFT JOIN` = 0); xoá lớp → CASCADE | DB |
| TC-G07-15 | AC8 | – | `go test ./internal/guidance ./internal/assignment -run 'TestRegressionCaseNoText\|TestRegressionCaseIdempotent\|TestRegressCLIRefusesExport' -v` | PASS (tên test tồn tại thật) | GO |

## US-GD-08 Câu tương tự lúc soạn (Threads)
Dữ liệu: thread công khai đã có đáp án AI `VERIFIED` / `CORRECTED` (seed 10.x có ≥ 1; QC dựng thêm nếu thiếu). `SIM=/courses/$C/threads/similar-confirmed?q=`. Embedding giả: độ tương tự cosine tất định (QC dùng cùng chuỗi → 1,0; chuỗi khác hẳn → thấp).
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G08-01 | AC1 | `THREAD_SUGGEST` bật; thread T1 `VERIFIED` tiêu đề «Công thức tính điểm quá trình» | `GET $SIM` + chính tiêu đề T1 (SV) | ≤ 3 thread công khai cùng lớp, đáp án `VERIFIED` / `CORRECTED`, chưa ẩn / loại, cosine ≥ 0,80; xếp giảm dần; DTO `[{thread_id, title}]` | API |
| TC-G08-02 | AC1 | – | `q` dưới 3 từ; `q` rỗng; `q` 5.000 ký tự | Dưới 3 từ: `[]` (hoặc 422, ghi rõ); không `500`; dài: cắt / 422 | API |
| TC-G08-03 | AC1 | Có 5 thread giống nhau | `GET $SIM` | Đúng 3 (LIMIT 3); thứ tự giảm | API |
| TC-G08-04 | AC1 | Cosine 0,79 vs 0,81 (chuỗi dựng gần ngưỡng) | `GET` | 0,81 có; 0,79 không | API |
| TC-G08-05 | AC1 | Thread đáp án `UNVERIFIED` / `REJECTED`; thread `hidden` (Ẩn); thread bị loại | `GET` bằng tiêu đề giống | Không xuất hiện | API |
| TC-G08-06 | AC2 | Có mục kho `source_type='TICKET'` (từ kênh riêng) giống câu soạn | `GET` | **Không** hiện mục từ `TICKET`; chỉ `POST` (Q-G-5). QC dựng: hỏi riêng + escalate + GV trả lời có nội dung chứa chuỗi lạ «ZQ-TK-91»; soạn tiêu đề «ZQ-TK-91» → không lộ | API |
| TC-G08-07 | AC3 | `/threads` soạn tiêu đề ≥ 3 từ (debounce 600 ms) | Gõ; bắt gói mạng | Danh sách dưới ô tiêu đề: tiêu đề thread (liên kết `/threads/[id]`) + «Đã xác nhận»; không đoạn giải thích; không đẩy ô đăng xuống quá 3 dòng; **không có kết quả → không hiện gì** (không «không tìm thấy»); request sau ~600 ms (không mỗi phím) | E2E |
| TC-G08-08 | AC4 | – | Bấm đăng khi có gợi ý | Đăng bình thường (không hộp thoại); tường lửa PII + gộp thread tương tự (`THREAD_SIMILAR_MIN`) giữ nguyên (E2E hiện có xanh) | E2E |
| TC-G08-09 | AC5 | Đếm `llm_audit` / nhúng | 20 lần gợi ý | Đúng **một** lần nhúng chuỗi đang soạn mỗi lần (cùng chuỗi: dùng cache, 0 nhúng thêm); **0** lời gọi sinh chữ (`llm_audit` không dòng chat); một truy vấn SQL theo `course_id` (`log_statement`) | API+DB |
| TC-G08-10 | AC5 | – | 35 lần / phút (một người) | Từ lần 31: `429` (THREAD_SUGGEST_RPM=30); người khác không bị ảnh hưởng | API |
| TC-G08-11 | AC5 | – | p95 | ≤ 300 ms với embedding giả (30 lượt) | API |
| TC-G08-12 | AC5 | Cờ tắt | `GET $SIM` | `[]`; 0 nhúng; 0 SQL tri thức | API+DB |
| TC-G08-13 | AC6 | Tài liệu nguồn đổi (`document.changed`) → mục kho lỗi thời | `GET` | Không gợi ý mục lỗi thời; ẩn thread (staff) → biến mất khỏi gợi ý ≤ 60 s (đo) | API |
| TC-G08-14 | AC7 | – | Soạn chuỗi chứa «MSSV 20201234»; kiểm `pii_events`, `llm_audit`, bảng khác | Câu đang soạn **không** lưu, không `pii_events`, không gửi LLM; tường lửa `threads/precheck` vẫn chạy độc lập (kết quả như trước) | API+DB |
| TC-G08-15 | AC7 | Thread lớp 2 tiêu đề giống | SV lớp 1 `GET` | Không lộ thread lớp 2 (cô lập `course_id`; chuỗi lạ «ZQ-L2-…») | API |
| TC-G08-16 | AC8 | – | SV, TA, teacher lớp: `GET`; SV lớp 2 / `sv.moi`; không token | `200` ×3; `404`; `401` | API |
| TC-G08-17 | AC9 | Embedding / DB lỗi (dừng `pgbouncer` ngắn / `FAKE` lỗi nhúng) | `GET`; soạn trên giao diện | Giao diện **không hiện lỗi**, đăng vẫn được; API trả `[]` | API+E2E |
| TC-G08-18 | AC10 | 375 px | Soạn | Danh sách một cột; vùng chạm ≥ 44 px | E2E |

## US-GD-09 Lý do loại câu hỏi (mẫu âm)
Dùng `/courses/$C/questions/bulk`, `PUT …/questions/{id}/review`, `GET …/questions/gen-stats?job_id`, `GenerateQuestions` qua stub (`QUESTION_GEN`).
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G09-01 | AC1 | Câu PENDING | `PUT …/review {status:"REJECTED", reject_reason:"MO_HO"}` rồi từng lý do trong danh sách đóng (6) và không gửi lý do | Lưu `question_bank.reject_reason`; không gửi = `UNSPECIFIED`; mỗi nhãn đúng | API+DB |
| TC-G09-02 | AC1 | – | `bulk {action:"reject", ids[], reason:"TRUNG_LAP"}`; `reason:"XYZ"`; `reason` với `action:"approve"` | OK; `422` lý do lạ; `reason` kèm duyệt → `422` hoặc bỏ qua (ghi rõ); `CHECK` DB: `reject_reason` chỉ khi `REJECTED` | API+DB |
| TC-G09-03 | AC2 | `/questions` | Bấm `Loại` | Vẫn **một thao tác**; hiện dãy nhãn lý do chọn nhanh (bỏ qua được) + «Hoàn tác» 5 s; không hộp thoại bắt buộc | E2E |
| TC-G09-04 | AC2 | – | Loại rồi bỏ qua lý do; loại + chọn lý do; hoàn tác trong 5 s | `UNSPECIFIED`; lý do lưu; hoàn tác khôi phục `PENDING` và **xoá** `reject_reason` | E2E+DB |
| TC-G09-05 | AC3 | `QUESTION_HINTS` bật; 4 câu REJECTED có lý do ≠ UNSPECIFIED (< 5) | `GenerateQuestions`; bắt prompt (stub) | **Không** khối «Tránh…» (chưa đủ 5) | API |
| TC-G09-06 | AC3 | 5 câu REJECTED có lý do (MO_HO ×3, TRUNG_LAP ×1, LAC_DE ×1) trong 90 ngày; thêm câu `UNSPECIFIED` | `GenerateQuestions` | Thống kê SQL: chỉ lý do ≠ UNSPECIFIED, chỉ trong 90 ngày (câu 91 ngày không tính), lấy tối đa **3** lý do nhiều nhất (đồng hạng ổn định) | API+DB |
| TC-G09-07 | AC4 | Cùng | Bắt prompt sinh | Khối «Tránh các lỗi sau» gồm **câu cố định** (đúng chữ SRS 4.5: MO_HO «Mỗi câu chỉ một cách hiểu; tránh từ chung chung.» …); **không** có nội dung câu bị loại, **không** chữ tự do của GV (chèn nhãn có `'`/lệnh vào dữ liệu → không xuất hiện); ≤ 140 ký tự mỗi nhãn | API |
| TC-G09-08 | AC5 | stub sinh câu | `GenerateQuestions` | Kiểm code **không đổi** (một đáp án đúng, đáp án trong lựa chọn, `citations` bắt buộc, `origin=GENERATED`); câu mới `PENDING`; `go test ./internal/practice -run TestRejectReasonHints -v` PASS | API+GO |
| TC-G09-09 | AC6 | – | `gen-stats?job_id` | Thêm `hints:[reason…]` (nhân viên thấy lý do đã dùng); cờ tắt → `hints` rỗng / vắng | API |
| TC-G09-10 | AC7 | Cờ tắt / < 5 câu / lỗi thống kê (ép lỗi SQL: quyền cột) | `GenerateQuestions` | Prompt như trước; sinh không lỗi | API |
| TC-G09-11 | AC8 | – | SV: `GET …/questions`, `gen-stats`; GV lớp 2; không token | `403`; lý do loại **không bao giờ** trong DTO SV (`grep` `reject_reason`); `404`; `401` | API |
| TC-G09-12 | AC1 | Bulk loại 60 câu | `bulk reject` | Ghi lý do cho cả lô; không N+1 (một câu lệnh); `audit_log` như hiện hành | API+DB |

## Nhánh lỗi
TC-G07-08, 09, 11; TC-G08-02, 17; TC-G09-02, 10.

## Phân quyền
TC-G07-13; TC-G08-06, 15, 16; TC-G09-11.

## Lịch sử sửa TC
(chưa có)
