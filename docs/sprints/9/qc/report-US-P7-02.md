# QC report — US-P7-02 (tab Bài tập: CRUD, rubric, công bố, gợi ý tiêu chí, lịch / nhắc) · Kết luận vòng 1: **PASS** (AC10 / phần AC11 chuyển nơi kiểm theo #8 — chờ PM; Q-02-a, Q-02-b mở)
Ngày 2026-10-31. Commit kiểm: `sprint/9-p7` @ `cd189fea` (mã máy chủ + frontend tab; cùng cây `e424a20b`). Bộ TC: `tc-US-P7-02.md`. Handoff: `dev-US-P7-02.md`. CI `cd189fea` / `e424a20b`: **success**.

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh` theo gói, `-race -tags integration` (httpapi `-tags testroutes`), mọi tên test của dòng `Kiểm:` đang tồn tại | PASS: assignment 26/26, calendar 1, grading (Suggest) 6, httpapi 2. Thiếu `TestAssignmentsRBACMatrix` — chuyển US-P7-11 (#8) |
| Hộp đen trên stack seed thật (`scripts/p7-api-02.py`, `https://localhost:543`) | PASS 0 lỗi: tạo BT03 từ fixtures (201, tiêu chí `c1…c4` do máy chủ cấp, version 1), Idempotency-Key (một bài), 21 ca biên 422 (trừ hai ca ở Q-02-a / Q-02-b), tổng rubric 9,75 / 10,25 / bước 2,3 → 422, công bố → 30 thông báo `ASSIGNMENT_PUBLISHED` đúng số SV ACTIVE, công bố lần hai không thông báo lại, đổi hạn sau công bố → lịch SV đổi, sai `version` → 409, đổi `max_score` sau công bố bị khoá, `DELETE` bài sạch → 204 |
| Quét canary (`CANARY-BT03-ANSWER`) bằng `sv.gioi@`, `sv.kha@` ở `GET assignments`, `…/{id}`, `…/mine` | không có `reference_answer` / `guidance` / canary; TA đọc được đáp án (chỉ xem); ADMIN 403; TA tạo / sửa / xoá / công bố 403; STUDENT tạo / gợi ý 403 |
| Playwright desktop `assignments tab layout`, `drawer offline draft`, `drawer conflict`, `drawer double submit`, `assignments tab states` | 5/5 xanh |
| `ui-antipatterns.sh` | rc 0, sạch |
| `grep -rniE 'mock|assess'` ở `frontend/src/features/assignments`, `backend-go/internal/assignment` | rỗng |
| Đột biến (`/tmp/qc-mut9`, đã xoá) A1 bỏ kiểm tổng rubric · A2 bỏ `LATE_RULE_LOCKED` · A3 bỏ `RUBRIC_LOCKED` · A4 bỏ `HAS_SUBMISSIONS` · A5 bỏ `SCORE_STEP` | cả 5 **đỏ** đúng test |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 tab `Bài tập` trong `/grading` (một Panel, bảng năm cột, `Tạo bài tập` chỉ TEACHER, TA chỉ xem) | PASS | `assignments tab layout`, `…states` |
| AC2 Drawer / trường / biên | PASS | `p7-api-02.py` + `TestCreateAssignment*`; xem Q-02-a, Q-02-b |
| AC3 rubric: tổng = `max_score`, bước 0,25, id do máy chủ cấp | PASS | A1; API 9,75 / 10,25 / 2,3 → 422 |
| AC4 gợi ý tiêu chí: một lời gọi, phần dư lớn nhất | PASS ở mức Go (`TestSuggest*`, 6 test); **trên stack thật `rubric/suggest` trả `422 RUBRIC_SUGGEST_INVALID`** vì provider giả chưa có bản ghi demo cho gợi ý (không 5xx, không 403) → kiểm đầu-cuối bằng bản ghi demo ở US-S9-SEED |
| AC5 công bố: Confirm, thiếu rubric / đáp án chặn, thông báo một lần | PASS | API + `TestPublish*`; 30 thông báo đúng số SV |
| AC6 sửa sau công bố: khoá trường đúng, `409 VERSION_CONFLICT`, gia hạn trước công bố | PASS | A2, A3; API đổi hạn → lịch đổi |
| AC7 xoá: bài sạch 204; có bài nộp / gắn `grade_items` chặn | PASS | A4; `TestAssignmentDeleteRestrictedByGradeItem` (đã sửa ở US-P7-01 vòng 2) |
| AC8 gắn bài vào công thức sổ điểm | PASS | `TestLinkedToGradebook*`; Drawer ghi "Đã gắn công thức" |
| AC9 tự lưu nháp 2 s, offline, 409, bấm đúp | PASS | 3 spec Playwright |
| AC10 phân quyền theo route | PASS cho hàng đã có route; **`TestAssignmentsRBACMatrix`** chuyển US-P7-11 (#8) | ADMIN 403, TA chỉ xem, STUDENT 403 xác nhận trên stack |
| AC11 không `mock/assess` | PASS | grep rỗng |

## 3. Lỗi / ghi nhận
- Không có bug.
- **Q-02-a (BA):** mã nộp trùng trả `422 VALIDATION_FAILED` kèm `field: submission_code`; TC mong `409`. SRS 4.2.1 chỉ ghi "duy nhất trong lớp" → coi hợp lệ.
- **Q-02-b (BA / PM):** `channels: []` rõ ràng được máy chủ đổi thành mặc định `{IN_APP}` (`201`), TC mong `422`. SRS 4.2.1 ghi "≥ 1, mặc định {IN_APP}" — không nói `[]` rõ ràng; DB vẫn chặn `{}`. Đề nghị BA chốt: giữ (mặc định) hay `422`.
- Dev chưa làm `canary-scan.sh` (cần stack) — QC đã quét tay bằng canary (bảng trên); xoá ở UI chưa có e2e riêng, đã kiểm ở API.
- Handoff: OpenAPI 254 thao tác, `today.Topics()` 28 và `TestTodayQueryBudget` +1 — đã có proposal #8, #9 ACCEPTED.
