# QC test case — US-P9-09 Lịch sử, điểm yếu, Hôm nay, tín hiệu tổng hợp k ≥ 3
Nguồn: US-P9-09 + SRS FR-18–20, §6 #13–#14. Seed: ≈ 40 lượt luyện của SV mô phỏng, ≥ 3 SV sai cùng 2 chủ đề (đủ k-ẩn danh).

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-09-01 | AC1 | `sv.kha` có ≥ 3 lượt (TOPIC + MOCK) | `GET /courses/$C1/practice/history?limit=20`; mở `/practice/history` 1440 / 375 | Hàng theo thời gian giảm dần: ngày, loại (`Theo chủ đề`/`Thi thử`), chủ đề, `đúng/tổng`; không huy hiệu, không biểu đồ thành tích | API + E2E |
| TC-09-02 | AC1 | – | Bấm một hàng | Xem lại từng câu: đúng/sai, đáp án, giải thích; (MOCK `IN_PROGRESS` không bị lộ đáp án — đối chiếu TC-10-03) | E2E |
| TC-09-03 | AC1 | SV 45 lượt (tạo thêm bằng `start`/`submit` hoặc DB) | Duyệt `limit=20` theo `cursor` đến hết; `limit=101` | 20/20/5; không trùng, không thiếu; `limit` tối đa 100; thứ tự `(started_at DESC, id DESC)` ổn định khi có hai lượt cùng `started_at` | API |
| TC-09-04 | AC2 | `sv.kha` ≥ 10 câu đã làm, chủ đề A (sai 5/6), B (sai 3/5), C (sai 1/10), D (sai 2/2 nhưng chỉ 2 câu) | `GET /practice/weakness`; mở lịch sử | «Chủ đề cần ôn» tối đa 3 chủ đề xếp theo tỉ lệ sai: A (83%), B (60%); **D không** (chỉ 2 câu < 3); C hiện nếu < 3 chủ đề (tỉ lệ 10%: theo AC liệt kê tối đa 3 có ≥ 3 câu; ghi hành vi; P-10-7 Q18: không ngưỡng sai tối thiểu, hoà → nhiều câu sai hơn rồi tên chữ cái); mỗi chủ đề kèm ≤ 2 tài liệu gợi ý từ `citations` các câu sai, xếp theo số câu sai trỏ tới, bấm mở `/library` | API |
| TC-09-05 | AC2 | `sv.gioi` mới < 10 câu đã làm | Mở lịch sử | Khối «Chủ đề cần ôn» **ẩn** (không rỗng chữ) | E2E |
| TC-09-06 | AC3 | `sv.kha` có chủ đề yếu A, chưa luyện A trong 3 ngày | `GET /courses/$C1/today`; mở `/` | Có việc `PRACTICE_WEAK_TOPIC`: «Ôn chủ đề A», liên kết `/practice?topic=A`; bậc 85 (xếp giữa 80 và 90) | API + E2E |
| TC-09-07 | AC3 | Cùng; SV đã luyện A trong 2 ngày qua (ép `submitted_at` 2 ngày trước) | `today` | **Không** có việc A (chỉ hiện nếu chưa luyện trong 3 ngày); ép 4 ngày → có lại | API + DB |
| TC-09-08 | AC3 | Việc A đang hiện | Luyện A xong (`submit`) rồi poll `today` mỗi 1 s | Biến mất ≤ 5 s sau khi luyện xong («luyện xong» = `submit` lượt TOPIC của chủ đề; mốc 3 ngày theo `submitted_at`; P-10-7 Q19); sự kiện `practice.updated` kích hoạt làm mới | API |
| TC-09-09 | AC3 | `today` của TEACHER, TA, ADMIN | `GET today` | Không có `PRACTICE_WEAK_TOPIC` (chỉ STUDENT) | API |
| TC-09-10 | AC4 | Seed | `go test ./internal/practice -run TestWrongAnswerSignalsNoIdentity -v` | PASS | GO |
| TC-09-11 | AC4 | Dữ liệu: chủ đề P sai bởi 3 SV; chủ đề Q sai bởi 2 SV (tạo bằng `start`/`PUT` qua API nhiều tài khoản hoặc DB) | Gọi `WrongAnswerSignals(C1, from, to)` (không có route; đường gọi QC: `go test -run TestWrongAnswerSignalsNoIdentity -v` của dev + test bổ sung của QC trong bản sao gói `practice`; P-10-7 Q20: dev trả lời chi tiết ở **handoff**) | Kết quả chứa P (`students_affected≥3`) và **không** chứa Q (k-ẩn danh); mỗi mục `{topic, wrong_answers, students_affected, sample_question_ids}`; **không** `student_id`, tên, MSSV, email (đối chiếu chuỗi tên SV mô phỏng: 0 lần xuất hiện) | GO |
| TC-09-12 | AC4 | – | Khoảng `from/to` ngoài dữ liệu; `from>to` | Rỗng / lỗi rõ, không panic | GO |
| TC-09-13 | AC4 | – | Thêm SV thứ 3 trả lời sai chủ đề Q rồi gọi lại | Q xuất hiện ở lần gọi sau (`students_affected=3`) | GO |
| TC-09-14 | AC5 | GV, TA, ADMIN | `GET practice/history`, `weakness`, `overview`, `attempts/{aid}` của SV | `403` | API |
| TC-09-15 | AC5 | GV | Mở Hồ sơ 360 của `sv.kha` (UI + `GET …/students/{id}/360`) | **Không** hiển thị dữ liệu luyện đề (lượt, chủ đề yếu, điểm); chỉ `activity` heartbeat `PRACTICE` như cũ, theo công tắc đo của lớp | API + E2E |
| TC-09-16 | AC5 | Công tắc đo của lớp tắt | Luyện đề | Không ghi heartbeat `PRACTICE` (hoặc theo công tắc; kiểm `activity`) | DB |
| TC-09-17 | AC6 | `sv.kha` có lượt ở lớp 1; cũng ở lớp 2 (nếu SV thuộc cả hai; nếu seed thiếu, QC tạo lớp thử + SV qua API admin; P-10-7 Q1) | `GET /courses/$C2/practice/history` | Không có lượt của lớp 1 (lọc `course_id`) | API |
| TC-09-18 | AC6 | SV rời lớp 1 (`enrollments` không còn `ACTIVE`) hoặc lớp 1 lưu trữ (`POST /admin/courses/{id}/archive`) | `GET history` ở lớp 1 | Không hiện lượt (hành vi với chính lớp đó: mặc định QC `403` hoặc rỗng, ghi hành vi thật; P-10-7 Q21) | API |
| TC-09-19 | AC7 | `sv.moi` đã vào lớp, chưa luyện lượt nào | Mở `/practice/history` | «Chưa có lượt luyện» + nút `Luyện ngay` | E2E |
| TC-09-20 | AC7 | `route.abort` API history | Mở `/practice/history` | `PageState` lỗi + `Thử lại`; bấm `Thử lại` → tải được | E2E |
| TC-09-21 | AC8 | SV có 200 lượt (tạo bằng script `INSERT` hoặc API; ghi cách) | `GET history?limit=20` × 100 lần, đo | p95 ≤ 300 ms; `EXPLAIN (ANALYZE)` truy vấn dùng chỉ mục `(course_id, student_id, started_at DESC, id DESC)` (không `Seq Scan`) | API + DB |

## Phân quyền
TC-09-14, 15; SV khác không xem lịch sử người khác (không có tham số `student_id`: thử `?student_id=<A>` bởi B → bị bỏ qua, trả lịch sử của B).

## Lịch sử sửa TC
- 2026-11-14 — spec FEAT-practice v1.1 + PM trả lời P-10-7 (Q-S10-1…26): TC-09-04, 08, 11, 17, 18 ghi mặc định Q18, Q19, Q20, Q1, Q21.
