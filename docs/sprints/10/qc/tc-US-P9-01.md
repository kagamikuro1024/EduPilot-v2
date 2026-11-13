# QC test case — US-P9-01 Lược đồ luyện đề, Quiz Engine, DTO không rò đáp án
Nguồn: `FEAT-practice/US.md` US-P9-01 + `SRS.md` §4.1, §5, §6, §9. Pha 1, không đọc mã dev. Loại: xem `tc-00-chung.md`.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-01-01 | AC1 | DB trống (stack mới) | `goose up` (qua `pnpm dev` hoặc lệnh trong AGENTS.md); `\dt` | Có `practice_attempts`, `practice_items`, `practice_answers` (SRS §5). **Không có `practice_exams`** (SRS §5 + P-10-2; US AC1 còn nhắc tên này: Q-S10-2). Có enum `practice_mode` (`TOPIC`,`MOCK`), `practice_status` (`IN_PROGRESS`,`SUBMITTED`,`EXPIRED`) | DB |
| TC-01-02 | AC1 | TC-01-01 | `\d practice_attempts`, `\d practice_items`, `\d practice_answers` | Mọi bảng có `course_id`; FK con trỏ `(course_id, …)`: `(course_id,student_id)→enrollments(course_id,user_id)`, `(course_id,question_id)→question_bank`; có CHECK `(mode='MOCK')=(deadline_at IS NOT NULL)`; UNIQUE từng phần `(course_id,student_id,mode,COALESCE(topic,'')) WHERE status='IN_PROGRESS'`; chỉ mục `(course_id,student_id,started_at DESC,id DESC)` và `(status,deadline_at) WHERE status='IN_PROGRESS' AND mode='MOCK'` | DB |
| TC-01-03 | AC1 | Có 1 lượt, 1 item của lớp 1 | `INSERT` (ghi DB, stack kiểm) một `practice_items` có `course_id`=lớp 2 nhưng `attempt_id` thuộc lớp 1; một `practice_items` có `question_id` của lớp 2 trong lượt lớp 1; một `practice_attempts` với SV chưa `enrollments` lớp đó | Cả ba bị từ chối `SQLSTATE 23503` | DB |
| TC-01-04 | AC1 | – | `goose down` (1 bước) rồi `goose up` | Cả hai thành công; bảng biến mất rồi trở lại; dữ liệu seed lớp khác không mất | DB |
| TC-01-05 | AC1 | repo | `cd backend-go && sqlc generate && sqlc diff` | `sqlc diff` rỗng, mã thoát 0 | GO |
| TC-01-06 | AC1 | – | Chèn hai lượt `IN_PROGRESS` cùng (SV, lớp, `TOPIC`, cùng chủ đề) | Lượt thứ hai bị UNIQUE từ chối (`23505`); khác chủ đề thì được; sau khi lượt 1 `SUBMITTED` thì được | DB |
| TC-01-07 | AC2 | – | Chạy bộ test chấm `SHORT` của dev: `go test ./internal/quiz -run 'GradeShort' -v`; đếm số ca con | ≥ 30 ca, tất cả PASS | GO |
| TC-01-08 | AC2 | Câu `SHORT`, đáp án chấp nhận `{"Hà Nội"}` (tạo qua API `POST …/questions` ở TC-01-12; học sinh trả lời qua lượt TOPIC) | Trả lời lần lượt: `Hà Nội`; `  hà   nội  `; `HÀ NỘI.`; `Hà Nội!`; `Ha Noi`; `ha noi`; `Hà  Nội` (hai dấu cách); chuỗi NFD của «Hà Nội» (`H\u0061\u0300 N\u006f\u0323\u0302i`) | `1` cho 4 câu đầu và câu NFD (khớp sau chuẩn hoá: trim, gộp khoảng trắng, hoa thường, bỏ dấu câu cuối, NFC); `0` cho `Ha Noi`, `ha noi` (**không bỏ dấu tiếng Việt**); `0` không gọi LLM (`llm_audit` 0 dòng mới) | API |
| TC-01-09 | AC2 | Câu `SHORT` đáp án chấp nhận `{"TCP/IP","TCP IP"}` | Trả lời `tcp/ip.`; `TCP-IP`; `tcp ip`; chuỗi rỗng | `1`; `0`; `1`; `0` (câu bỏ trống = 0, không lỗi 5xx) | API |
| TC-01-10 | AC3 | repo | `cd backend-go && go test -race ./internal/quiz/... 2>&1 \| tail -15` | `ok`; có ca MCQ một, MCQ nhiều, đúng–sai, `SHORT`, câu bỏ trống, id lạ → `ErrInvalidOption` | GO |
| TC-01-11 | AC3 | repo | `go list -deps ./internal/quiz \| grep -c 'internal/llm'`; `grep -rn '"float64"\|float64' backend-go/internal/quiz/*.go \| grep -v _test` (pha 2: DIFF) | 0 import `internal/llm`; không `float64` trong đường tính điểm (điểm = `decimal`) | DIFF |
| TC-01-12 | AC4 | GV | `api teacher POST /courses/$C1/questions` với câu `SHORT` có ≥ 1 đáp án chấp nhận (trường theo openapi; Q-S10-3) và câu `ESSAY` có đáp án mẫu + rubric ngắn | `201`; `review_status=DRAFT` cả hai | API |
| TC-01-13 | AC4 | Câu `SHORT` đã `APPROVED` (qua `PUT …/review`) | `POST /courses/$C1/exams` (hoặc thêm vào `exam_items` của bài thi nháp theo openapi) với câu `SHORT` rồi với câu `ESSAY` | `422 TYPE_NOT_SUPPORTED_IN_EXAM` cả hai; `exam_items` không có dòng mới | API |
| TC-01-14 | AC5 | `sv.kha` có lượt `TOPIC` và `MOCK` | `GO`: `go test ./internal/practice -run 'TestPracticeStudentDTOAllowlist' -v`; thêm quét đen: `GET /practice/attempts/{aid}` chưa trả lời câu, rồi sau khi trả lời | Test xanh; DTO chưa trả lời không chứa `answer_key`, `correct`, `explanation`, `citations`; sau khi trả lời (TOPIC) hoặc nộp (MOCK) chúng mới có, **chỉ** của câu đã trả lời | GO + API |
| TC-01-15 | AC5 | – | Sửa tạm `student_dto_allowlist.json` ở bản sao (không commit) bằng cách thêm trường cấm vào struct DTO ở bản sao mã | (kiểm «test có răng»): `TestPracticeStudentDTOAllowlist` phải FAIL. Không làm được thì ghi «không kiểm được độ nhạy», không tính FAIL | DIFF |
| TC-01-16 | AC6 | GV | `POST …/questions` câu `SHORT` với đáp án chấp nhận `[]`, `[""]`, `["  "]`, `["...","?!"]` | Mỗi lần `422 VALIDATION_FAILED`, `details[].code=ANSWER_EMPTY`; không tạo câu | API |

## Nhánh lỗi (SRS §3)
Không có tình huống lỗi riêng của story này ngoài TC-01-03, 06, 13, 16.

## Phân quyền
| TC | AC | Bước | Kết quả |
| --- | --- | --- | --- |
| TC-01-P1 | AC4 | STUDENT `POST /courses/$C1/questions` | `403` |
| TC-01-P2 | AC4 | TA tạo câu `SHORT` | `201` (Staff được) |
| TC-01-P3 | AC5 | GV, TA, ADMIN gọi `GET /practice/attempts/{aid}` của `sv.kha` | `403` (`reason: role`) |
| TC-01-P4 | AC5 | `sv.gioi` `GET /practice/attempts/{aid của sv.kha}` | `404` (không lộ tồn tại) |

## Script chạy được
`docs/sprints/10/qc/scripts/` (pha 2): `p9-01.sh` gom TC-01-08/09/12/13/16 bằng hàm `api`.

## Lịch sử sửa TC
(trống)
