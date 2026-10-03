# SRS FEAT-weekly-exam Thi hằng tuần: ngân hàng câu hỏi, bài thi, sandbox chấm code, làm bài, liêm chính, công bố, phúc khảo
Phiên bản 1 · 2026-10-03 · Trạng thái: DRAFT (chờ PM duyệt)

Nguồn: `docs/phases/PE.md`; `docs/sprints/5/plan.md`; PRD M15, G8, §3; FLOWS F19, F14; `ARCHITECTURE.md` §4–§9; `DECISIONS.md` D45, D46, D47, D54–D58; `AGENTS.md`; research `docs/research/2026-10-03-code-judge.md`; spec nền `FEAT-pg-foundation` (v1.7), `FEAT-ui-foundation`, `FEAT-llm-gateway`, `FEAT-account-security`, `FEAT-course-foundation`.

## 1. Mục đích và phạm vi

Cho giảng viên giao **bài kiểm tra lấy điểm hằng tuần** ngay trên hệ thống (yêu cầu của thầy hướng dẫn): **trắc nghiệm** và **lập trình C / C++ chấm tự động bằng bộ test ẩn**; sinh viên làm trong khung giờ; máy chấm (Quiz Engine cho trắc nghiệm, sandbox `go-judge` cho code); điểm **tự công bố khi bài thi đóng với cả lớp** (D56). Điểm tính bằng **code thuần** (`shopspring/decimal`), **không LLM** (luật 5); AI chỉ **gợi ý nháp** câu trắc nghiệm / đầu vào test, giảng viên duyệt (D57).

**Trong phạm vi:** migration `00005_weekly_exam` (13 bảng); gói `internal/quiz` (Quiz Engine), `internal/exam` (ngân hàng, bài thi, lượt làm, khoá chat, sự kiện, độ giống, kết quả, phúc khảo, gợi ý AI, nguồn việc "Hôm nay"), `internal/judge` (client go-judge, checker, hàng đợi), `internal/exam/similarity`; container `judge` + `deploy/judge/`; route `/questions`, `/exams`, `/exams/[id]`, `/exams/[id]/results`, `/exams/[id]/take`; seed bài thi mẫu; k6 `exam-submit.js`; cổng PE.

**Ngoài phạm vi:** luyện đề / thi thử không tính điểm, trích câu hỏi từ đề cũ, sinh câu hỏi từ tài liệu có trích dẫn, `SHORT` / `ESSAY` (P9); nối điểm bài thi vào sổ điểm (P6); màn chat và việc từ chối câu hỏi nội dung (P3 — PE chỉ cung cấp `exam.Locker`); lịch và nhắc việc (P8); checker tuỳ chỉnh, đa luồng, nhiều tệp nguồn, ngôn ngữ ngoài C / C++; giám thị bằng camera; tách judge sang máy riêng (nợ PR); điểm "vắng = 0" vào sổ điểm (P6).

**Khái niệm.** *Câu hỏi* (`question_bank`) → *mục* (`exam_items`, có điểm) → *bài thi* (`exams`, có khung giờ) → *lượt làm* (`exam_attempts`, mỗi sinh viên một lượt mỗi bài thi) → *bản nộp code* (`code_submissions`, nhiều lần, lần cuối tính). *Test mẫu* (`is_sample`) là công khai với sinh viên (nằm trong đề); *test ẩn* thì không bao giờ lộ nội dung.

## 2. Người dùng và quyền

Chế độ `CourseAccessGuard` dùng đúng tên ở `FEAT-course-foundation` 4.1 (`Member`, `Staff`, `Teacher`); route của sinh viên thêm kiểm **vai trong lớp = STUDENT** (`ACTIVE`); **không** dùng chế độ có "hoặc ADMIN": ADMIN không đọc nội dung bài thi.

| Hành động | STUDENT | TA | TEACHER | ADMIN | Chế độ |
| --- | --- | --- | --- | --- | --- |
| Xem / lọc ngân hàng câu hỏi (kể cả đáp án, test ẩn, lời giải mẫu) | ✗ | ✓ | ✓ | ✗ | `Staff` |
| Tạo / sửa câu nháp, nhập test, nhân bản, lưu trữ, chạy lời giải mẫu, gợi ý AI | ✗ | ✓ | ✓ | ✗ | `Staff` |
| Duyệt / loại câu (Q28) | ✗ | ✓ | ✓ | ✗ | `Staff` |
| Sửa test của bài code **sau khi đóng** (dẫn tới chấm lại) | ✗ | ✗ | ✓ | ✗ | `Teacher` |
| Tạo / sửa bài thi nháp, chọn câu, xem trước, nhân bản | ✗ | ✓ | ✓ | ✗ | `Staff` |
| Lên lịch, bỏ lịch, gia hạn, hoãn / nhả công bố, xoá bài thi | ✗ | ✗ | ✓ | ✗ | `Teacher` |
| Xem danh sách bài thi (bản đầy đủ) | ✗ | ✓ | ✓ | ✗ | `Member` (+ vai) |
| Xem danh sách bài thi (bản giới hạn, chỉ bài đã lên lịch trở đi) | ✓ | – | – | ✗ | `Member` (+ vai) |
| Bắt đầu / làm / lưu / chạy thử / nộp lời giải / nộp bài, xem kết quả **của mình**, gửi phúc khảo | ✓ (chỉ của mình) | ✗ | ✗ | ✗ | `Member` + STUDENT |
| Xem tiến độ và bảng điểm lớp, thống kê, CSV, danh sách phúc khảo | ✗ | ✓ | ✓ | ✗ | `Staff` |
| Xem log rời tab / dán, cặp bài code nghi giống nhau, chạy lại so sánh, xử lý cặp | ✗ | ✗ | ✓ | ✗ | `Teacher` |
| Sửa điểm tay, đổi đáp án / bỏ câu, chấm lại, trả lời phúc khảo | ✗ | ✗ | ✓ | ✗ | `Teacher` |
| `GET /me/exam-lock` | ✓ | ✓ | ✓ | ✓ | JWT (chỉ chính mình) |

Quy tắc: danh tính (`student_id`, `actor_id`) **luôn từ JWT**, không từ tham số hay thân yêu cầu; giảng viên / TA **không** dùng route làm bài của sinh viên (xem trước dùng `…/preview`); sinh viên không có route nào đọc ngân hàng câu hỏi; mọi đường đọc của sinh viên lọc theo `student_id` = `sub`; Admin chỉ thấy chỉ số vận hành (PR), không nội dung; lớp `ARCHIVED` chỉ đọc (trừ nộp lượt đang chạy).

## 3. Luồng chính và các nhánh lỗi

### 3.1 Luồng chính (F19)

```mermaid
sequenceDiagram
  participant T as Giảng viên / TA
  participant GW as Gateway
  participant DB as Postgres
  participant WK as Worker
  participant JD as judge (go-judge)
  participant S as Sinh viên
  T->>GW: soạn câu + test + lời giải mẫu
  GW->>DB: question_bank, code_problems, code_testcases
  T->>GW: reference/verify (202)
  GW->>DB: jobs + outbox
  WK->>JD: POST /run (biên dịch + chạy mọi test)
  WK->>DB: reference_verified_version
  T->>GW: tạo bài thi, chọn câu, schedule
  GW->>DB: exams SCHEDULED + outbox exam.scheduled
  Note over WK: tick 5 s: SCHEDULED → OPEN → CLOSED
  S->>GW: POST attempts (Idempotency-Key)
  GW->>DB: exam_attempts (deadline_at) + ep:exam_lock
  loop mỗi ~2 s
    S->>GW: PUT answers / draft (X-Exam-Tab)
  end
  S->>GW: POST code submit (202)
  GW->>DB: code_submissions + outbox judge.enqueue
  WK->>JD: POST /run (biên dịch → từng test), DELETE /file/{id}
  WK->>DB: results, status DONE
  S->>GW: POST attempts/{id}/submit
  GW->>DB: GRADING; MCQ chấm ngay
  Note over WK: đóng bài → tự nộp mọi lượt → GRADED → PUBLISHED (nếu không hoãn)
  WK->>DB: exams PUBLISHED + outbox exam.published
  S->>GW: GET attempts/{id}/result
```

### 3.2 Máy trạng thái

**Bài thi** (`exams.status`): `DRAFT → SCHEDULED → OPEN → CLOSED → PUBLISHED`.

| Từ | Đến | Điều kiện | Ai / cái gì |
| --- | --- | --- | --- |
| `DRAFT` | `SCHEDULED` | qua mọi kiểm ở `schedule` (4.2.4) | Giảng viên |
| `SCHEDULED` | `DRAFT` | `now < opens_at` và chưa có lượt làm | Giảng viên (`unschedule`) |
| `SCHEDULED` | `OPEN` | `now ≥ opens_at` | tick (hoặc lười khi đọc) |
| `OPEN` | `CLOSED` | `now ≥ closes_at` | tick |
| `SCHEDULED` | `CLOSED` | `now ≥ closes_at` (máy chủ tắt dài) | tick |
| `CLOSED` | `PUBLISHED` | mọi lượt `GRADED`, không `regrading`, `publish_hold=false` | tick / nhả hoãn |

`effective_status(exam, now)`: nếu `SCHEDULED` và `now ≥ opens_at` → `OPEN`; nếu `OPEN` / `SCHEDULED` và `now ≥ closes_at` → `CLOSED`; còn lại giữ nguyên. **Mọi đường đọc / ghi dùng `effective_status`** nên bộ lập lịch chậm vài giây không cho ai làm bài ngoài giờ.

**Lượt làm** (`exam_attempts.status`): `IN_PROGRESS → GRADING → GRADED`.

| Từ | Đến | Điều kiện |
| --- | --- | --- |
| (không có) | `IN_PROGRESS` | `POST attempts` khi bài `OPEN` |
| `IN_PROGRESS` | `GRADING` | nộp tay / tự nộp (hết giờ, bài đóng) mà còn bài code chưa có kết quả `DONE` |
| `IN_PROGRESS` | `GRADED` | nộp mà không còn việc chấm (chỉ trắc nghiệm, hoặc mọi bài code đã `DONE`) |
| `GRADING` | `GRADED` | mọi bản nộp được tính đã `DONE` |
| `GRADED` | `GRADED` | chấm lại / sửa điểm / `override` (cập nhật tại chỗ) |

### 3.3 Nhánh lỗi

| Tình huống | Hệ thống phản ứng | Người dùng thấy |
| --- | --- | --- |
| Mất mạng giữa bài | Bản gõ giữ ở máy; hàng đợi gửi lại backoff 1/2/4/8/15 s; server giữ phần đã nhận | "Chưa lưu lên máy chủ ({n} thay đổi) — đang thử lại" / "Mất mạng — bài vẫn được giữ trên máy bạn" |
| Mất mạng tới hết giờ | Chỉ phần đã lên máy chủ được tính (+ `Gia hạn` bởi giảng viên) | Dải nói rõ điều này; sau hết giờ: "Hết giờ — bài của bạn đã được nộp" |
| Hết giờ khi đang gõ | Máy khách gửi bản cuối ở giây cuối; server nhận trong `EXAM_GRACE_SECONDS`; tự nộp; bài code chưa nộp mà có nháp → nộp nháp (`auto`) | "Hết giờ — phần bạn gõ sau giờ không được tính"; chữ còn trong ô (chỉ đọc) |
| Sandbox chết / quá tải | `Chạy thử` → 503 `JUDGE_UNAVAILABLE`; `Nộp lời giải` vẫn nhận + xếp hàng; sau retry + dead-letter → `IE` (không 0 điểm), lượt ở `GRADING`, việc `EXAM_GRADE_ERROR` | "Chưa chạy thử được, thử lại sau {n} giây" / "Hệ thống chưa chấm được bài của bạn. Bài đã được lưu và giảng viên sẽ xử lý." |
| Test sai do giảng viên | Sửa test → `tests_version` tăng → `regrade` (202, idempotent) → điểm cập nhật; sau công bố: `EXAM_REGRADED` | Sinh viên: "Điểm đã được cập nhật sau khi giảng viên sửa bộ test" |
| Hai tab / hai thiết bị | Một nơi ghi (`writer_tab`); nơi kia 409 `ATTEMPT_OTHER_TAB` → chỉ đọc + `Làm tiếp ở đây` (`takeover`); sự kiện `TAB_TAKEOVER` | "Bài đang mở ở nơi khác. [Làm tiếp ở đây]" |
| Nộp trùng | `Idempotency-Key` → phản hồi cũ; khoá khác → 409 `ATTEMPT_ALREADY_SUBMITTED` | "Bạn đã nộp bài này lúc …" |
| Lỗi biên dịch | `CE`, `compile_log` cắt + thay đường dẫn; 0 test đạt lần nộp đó | "Lỗi biên dịch" + dòng lỗi gọn; sửa và nộp lại trong giờ |
| Nộp dồn / vượt giới hạn | 429 `RATE_LIMITED` (giãn cách 15 s) / 409 `SUBMISSION_LIMIT_REACHED` (30 lần) | "Chờ {n} giây để nộp tiếp" / "Bạn đã dùng hết 30 lần nộp cho bài này" |
| `Chạy thử` quá 10 lần / 10 phút | 429 `RATE_LIMITED` + `retry_after` | "Bạn đã chạy thử 10 lần. Thử lại sau {mm:ss}" |
| Sinh viên không làm | Không có lượt; Staff thấy "Vắng" | "Bạn không làm bài này" |
| Gateway / worker khởi động lại giữa giờ | Mọi trạng thái ở DB, đồng hồ tính từ `deadline_at`; Stream `XAUTOCLAIM` | Tải lại → quay đúng câu đang làm |
| Redis chết | `exam.Locker` dùng DB; tự lưu / nộp không phụ thuộc Redis (Idempotency: PG trả 503 `SERVICE_UNAVAILABLE` ở POST bắt buộc khoá — máy khách thử lại) | "Đang kết nối lại…" |
| Bài còn `GRADING` khi đã đóng | Không công bố; Staff thấy "Đang chấm {x}/{y}" | Sinh viên: "Điểm đang được chấm" |
| Phúc khảo quá hạn / trùng | 409 `APPEAL_WINDOW_CLOSED` / `APPEAL_EXISTS` | "Đã hết hạn gửi yêu cầu xem lại" / "Bạn đã gửi yêu cầu cho bài này" |

## 4. Yêu cầu chức năng

### 4.1 Ngân hàng câu hỏi và Quiz Engine (US-PE-01, 03)

#### 4.1.1 Loại câu và `answer_key`

| `type` | `question_options` | `answer_key` (jsonb) | PE cho phép |
| --- | --- | --- | --- |
| `MCQ_SINGLE` | 2…8 | `{"option_ids":["<uuid>"]}` (đúng 1) | ✓ |
| `MCQ_MULTI` | 2…8 | `{"option_ids":["<uuid>",…]}` (1…n−1) | ✓ |
| `TRUE_FALSE` | không có | `{"value":true\|false}` | ✓ |
| `CODE` | không có | NULL (đáp án = bộ test, `code_problems`) | ✓ |
| `SHORT`, `ESSAY` | – | – | ✗ 422 `TYPE_NOT_SUPPORTED` (P9) |

Bảng `question_options` **không có** cờ đúng / sai: đáp án đúng chỉ nằm ở `answer_key` của `question_bank`; mọi truy vấn của đường sinh viên đọc `question_options` mà **không** đọc `answer_key` (rò đáp án bị chặn từ cấu trúc, ngoài `TestNoAnswerLeak`).

#### 4.1.2 Quy tắc kiểm hợp lệ

`title` 1…120; `topic` 1…80 (bắt buộc); `difficulty` ∈ `EASY|MEDIUM|HARD`; `stem` 1…8.000 ký tự Markdown, **lưu nguyên văn**; an toàn nằm ở **bộ render** (dựng React node từ cú pháp Markdown tối thiểu — đoạn, nhấn, mã, danh sách, bảng; HTML thô hiện thành chữ; không ảnh, không liên kết ngoài tự động; không `dangerouslySetInnerHTML`), không nằm ở việc cắt chuỗi lúc lưu; `explanation` ≤ 4.000. Đáp án: không rỗng, ≤ 1.000 ký tự, không trùng sau khi bỏ khoảng trắng và hạ chữ thường. Cảnh báo (không lỗi) cho đáp án kiểu "Tất cả các đáp án trên" / "Cả A và B" khi chưa `pinned_last`. Thao tác duyệt xem 4.1.4.

#### 4.1.3 Chấm trắc nghiệm (`quiz.Grade`)

```go
package quiz
type Type string // MCQ_SINGLE MCQ_MULTI TRUE_FALSE
type Mode string // ALL_OR_NOTHING PARTIAL
type Key struct { OptionIDs []string; Value *bool }
type Answer struct { OptionIDs []string; Value *bool }  // nil = bỏ trống
func Grade(t Type, points decimal.Decimal, key Key, a *Answer, mode Mode) (decimal.Decimal, error)
```

| Loại | Quy tắc | Điểm |
| --- | --- | --- |
| `MCQ_SINGLE` | chọn đúng đáp án duy nhất | `points`; còn lại (sai / trống / chọn nhiều) 0 |
| `TRUE_FALSE` | `Value` bằng `key.Value` | `points`; còn lại 0 |
| `MCQ_MULTI` + `ALL_OR_NOTHING` | tập chọn = tập đúng (sau khử trùng) | `points`; còn lại 0 |
| `MCQ_MULTI` + `PARTIAL` | `K` = số đáp án đúng; `TP` = chọn đúng; `FP` = chọn sai | `points × max(0, (TP − FP) ÷ K)` (`DivRound(…, 16)`) |

Không điểm âm; id không thuộc câu → `ErrInvalidOption` (API lưu: 422 `INVALID_OPTION_ID`; khi chấm lại: coi như chưa chọn). Không phụ thuộc thứ tự hiển thị.

#### 4.1.4 Duyệt, khoá sửa, chuyển trạng thái câu

`DRAFT --REQUEST--> PENDING --APPROVE--> APPROVED`; `DRAFT|PENDING --REJECT--> REJECTED`; `APPROVED|REJECTED --sửa--> DRAFT` (chỉ khi chưa bị bài thi `SCHEDULED+` dùng; sửa `stem` / đáp án / test đưa câu về `DRAFT` và **gỡ** `reference_verified_version`). Câu đang bị bài thi `SCHEDULED` / `OPEN` / `CLOSED` / `PUBLISHED` tham chiếu → 409 `QUESTION_IN_USE` (ngoại lệ ở `US.md` PE-03 AC8). Câu `AI_DRAFT` vào ở `PENDING`; không đường nào đặt `APPROVED` mà không có `reviewed_by`.

#### 4.1.5 Nhập test bằng zip

Tệp zip (≤ `EXAM_TESTZIP_MAX_BYTES`, mặc định 10 MiB; ≤ `EXAM_TESTZIP_MAX_UNCOMPRESSED` 50 MiB khi giải nén — **đếm khi đọc**, không tin header): mỗi test là cặp `<tên>.in` + `<tên>.out` (hoặc `.ans`) ở **gốc** zip (không thư mục lồng); tên `^[A-Za-z0-9_-]{1,60}$`; tên bắt đầu `sample` → `is_sample=true`; sắp `position` theo tên (so tự nhiên: `t2` trước `t10`); `manifest.json` tuỳ chọn `{"tests":[{"name","weight","is_sample"}]}` (khoá lạ bị từ chối, tên phải có trong zip); từ chối: thiếu nửa cặp, tên trùng, đường dẫn có `..` / tuyệt đối / ký tự điều khiển, liên kết tượng trưng, tệp không UTF-8, mỗi tệp > 1 MiB, tổng > 100 test (kể cả test đã có nếu `mode=append`). **All-or-nothing**: có một lỗi thì không ghi gì. `dry_run=true` trả `{would_create,errors[]}` không ghi. Thành công: một transaction, `tests_version` tăng **một** lần, `audit_log` `testcases.change` (số lượng, không nội dung). Route có giới hạn thân riêng (override `MAX_BODY_BYTES` chỉ ở route này, như roster).

### 4.2 Bài thi, lên lịch, máy trạng thái (US-PE-04)

#### 4.2.1 Trường và mặc định

`title` 1…120; `instructions` ≤ 4.000 (Markdown, lưu nguyên văn); `opens_at` < `closes_at`; `duration_minutes` `EXAM_MIN_DURATION_MINUTES` (5) … 300 và ≤ khung giờ; `shuffle_questions`, `shuffle_options` = `true`; `max_score` 10,00 (0 < x ≤ 100); `rounding_step` 0,01; `multi_scoring` `ALL_OR_NOTHING`; `reveal_answers` `true`; `appeal_days` 7 (0…30; 0 = không nhận phúc khảo); `kind` suy ra: chỉ trắc nghiệm → `MCQ`; chỉ code → `CODE`; cả hai → `MIXED` (bài rỗng: `MCQ`).

#### 4.2.2 Mục của bài

`PUT …/items` thay toàn bộ danh sách khi `DRAFT`: 1…100 mục; câu `APPROVED`, chưa lưu trữ, cùng lớp, không trùng; `points` > 0, ≤ 100, 2 chữ số thập phân. Thứ tự gửi = `position` (ràng buộc `UNIQUE(exam_id, position)` hoãn tới cuối transaction để đổi chỗ).

#### 4.2.3 Xem trước

Dùng đúng DTO và hàm dựng của lượt làm sinh viên (`BuildStudentView`) với hạt giống ngẫu nhiên của yêu cầu; trường `preview:true`; không ghi DB, không đặt khoá, không ghi sự kiện.

#### 4.2.4 Lên lịch (`schedule`) — kiểm trong một transaction `FOR UPDATE` trên bài

| Mã `details[].code` | Điều kiện sai |
| --- | --- |
| `NO_ITEMS` | không có mục |
| `ITEM_NOT_APPROVED` | có câu không `APPROVED` hoặc đã lưu trữ |
| `CODE_TESTS_MISSING` | câu code không có test ẩn `approved` hoặc không có test mẫu |
| `TOTAL_WEIGHT_ZERO` | `Σweight` test `approved` bằng 0 |
| `REFERENCE_NOT_VERIFIED` | `reference_verified_version ≠ tests_version` |
| `OPENS_IN_PAST` | `opens_at < now + EXAM_MIN_LEAD_SECONDS` (60 s) |
| `CLOSES_BEFORE_OPENS`, `DURATION_EXCEEDS_WINDOW`, `DURATION_TOO_SHORT` | khung giờ / thời lượng |
| `COURSE_ARCHIVED` (409) | lớp lưu trữ |

Trả **toàn bộ** lỗi một lần. Đạt: `status=SCHEDULED`; `audit_log` `exam.schedule`; outbox `exam.scheduled`; khoá sửa (mục 4.2.5). Gọi lại khi đã `SCHEDULED`: 200, không gửi thông báo thêm.

#### 4.2.5 Khoá sửa (`EXAM_LOCKED`)

| Trường / thao tác | `DRAFT` | `SCHEDULED` | `OPEN` | `CLOSED` | `PUBLISHED` |
| --- | --- | --- | --- | --- | --- |
| `title`, `instructions`, `reveal_answers`, `appeal_days` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `opens_at`, `duration_minutes`, `shuffle_*`, `max_score`, `rounding_step`, `multi_scoring`, mục | ✓ | ✗ | ✗ | ✗ | ✗ |
| `closes_at` | ✓ | `extend` | `extend` | ✗ | ✗ |
| `unschedule` | – | ✓ (chưa mở, chưa có lượt) | ✗ | ✗ | ✗ |
| xoá | ✓ | ✗ | ✗ | ✗ | ✗ |

#### 4.2.6 Bộ lập lịch (`exam.tick`)

Chạy ở worker, mỗi `EXAM_TICK_INTERVAL` (5 s), chỉ một bản chạy nhờ khoá leader `ep:exam:tick:leader` (`SET NX PX 15000`, gia hạn mỗi vòng). Mỗi vòng, theo thứ tự và mỗi bước idempotent bằng `UPDATE … WHERE status=<cũ>`: (1) mở bài đến giờ; (2) đóng bài đến hạn (+ outbox `exam.closed`); (3) tự nộp lượt `IN_PROGRESS` có `deadline_at + EXAM_GRACE_SECONDS < now()` (lô 200); (4) hoàn tất chấm các lượt `GRADING` đủ điều kiện; (5) công bố bài đủ điều kiện (4.8.2). Một vòng lỗi không dừng vòng sau.

### 4.3 Lượt làm: đồng hồ, tự lưu, một nơi ghi (US-PE-05)

#### 4.3.1 Bắt đầu

`POST …/exams/{eid}/attempts` (`Idempotency-Key` bắt buộc), trong một transaction: khoá bài (`FOR SHARE`), kiểm `effective_status = OPEN`, `now + 60 s < closes_at` (còn dưới 60 s → `EXAM_NOT_OPEN` `reason=closing`), `INSERT … ON CONFLICT (exam_id, student_id) DO NOTHING RETURNING` (đã có lượt → trả lượt cũ: `IN_PROGRESS` 200; đã nộp → 409 `ATTEMPT_ALREADY_SUBMITTED`); `deadline_at = LEAST(started_at + duration_minutes, closes_at)`; claim `writer_tab` theo `X-Exam-Tab`; đặt khoá chat (4.7.1); outbox `exam.attempt_started` (xoá cache "Hôm nay"). Hai yêu cầu song song cho đúng một dòng nhờ `UNIQUE`.

#### 4.3.2 Hạt giống xáo trộn

`seed = SHA-256(attempt_id bytes)`; `h1 = seed[0:8]`, `h2 = seed[8:16]` (big-endian `uint64`); `r = rand.New(rand.NewPCG(h1, h2))` (`math/rand/v2`, luồng PCG được đặc tả ổn định); thứ tự mục: Fisher–Yates `for i := n-1; i > 0; i-- { j := r.IntN(i+1); swap }` trên danh sách theo `position`; thứ tự đáp án của câu `k`: cùng thuật toán với `rand.NewPCG(h1 ^ uint64(k+1)*0x9E3779B97F4A7C15, h2)` trên các đáp án **không** `pinned_last`, rồi nối các đáp án `pinned_last` (giữ thứ tự gốc) vào cuối. Không lưu thứ tự. `shuffle_questions=false` / `shuffle_options=false` → giữ `position` gốc. Nhãn A, B, C… gán theo thứ tự hiển thị ở **frontend**.

#### 4.3.3 Lưu câu trả lời

`PUT …/attempts/{aid}/answers` (kèm `X-Exam-Tab`): kiểm chủ lượt (`student_id = sub`, khác → 404), `IN_PROGRESS`, `now ≤ deadline_at + grace` (quá → 409 `ATTEMPT_CLOSED`), người ghi (4.3.5); mỗi mục `{item_id, answer}` thuộc bài; `answer` đúng dạng theo loại câu và id đáp án thuộc câu; thân ≤ 64 KiB; giới hạn `EXAM_SAVE_RATE_PER_MIN` (240) theo lượt (`ep:rl:exam:{attempt_id}:{phút}`); một lệnh SQL `INSERT … ON CONFLICT (attempt_id,item_id) DO UPDATE SET answer=EXCLUDED.answer, saved_at=now()` cho cả lô; trả `{saved_at, server_time, deadline_at}`. Không bao giờ trả đúng / sai.

#### 4.3.4 Nộp bài và tự nộp

`finish(attempt, reason)` dùng chung cho `POST …/submit` (`MANUAL`), tick (`TIMEOUT` nếu `deadline_at = started_at + duration`, `CLOSED` nếu `deadline_at = closes_at`): trong một transaction `UPDATE exam_attempts SET status='GRADING', submitted_at=…, submit_reason=… WHERE id=$1 AND status='IN_PROGRESS'` (0 dòng → đã nộp: idempotent); `submitted_at = LEAST(now, deadline_at)` khi tự nộp; tạo bản nộp tự động cho bài code chưa có `SUBMIT` mà có nháp không rỗng (4.4.5); chấm trắc nghiệm ngay (4.6); nếu không còn bài code chờ → `GRADED`; gỡ khoá chat nếu không còn lượt `IN_PROGRESS` khác; outbox `exam.attempt_submitted`. Phản hồi của nộp tay chỉ có `{status,submitted_at,answered,total}`.

#### 4.3.5 Một nơi được ghi (`writer_tab`)

Mọi `PUT` (answers, draft) và `POST run|submit` mang `X-Exam-Tab: <uuid do tab sinh ra, giữ ở `sessionStorage`>`. `exam_attempts.writer_tab` / `writer_seen_at`. Quy tắc: nếu `writer_tab` rỗng hoặc bằng tab gọi hoặc `writer_seen_at < now − EXAM_TAB_STALE` (20 s) → tab gọi trở thành người ghi (cập nhật `writer_seen_at` **tối đa một lần / 10 s**); ngược lại 409 `ATTEMPT_OTHER_TAB` (`details.writer_seen_at`). `POST …/takeover` đặt `writer_tab = tab gọi`, ghi `TAB_TAKEOVER`. `GET …/attempts/mine` trả `writer:{is_you}` (không đổi người ghi). Tab chỉ đọc vẫn đọc được trạng thái và tự thử lại ghi sau mỗi 10 s chỉ khi người dùng bấm `Làm tiếp ở đây`.

#### 4.3.6 Hàng đợi lưu phía máy khách

Khoá `exam:<attempt_id>` (`useAutosaveDraft`): mỗi thay đổi ghi ngay vào `localStorage` `{rev_local, dirty:{item_id:answer}, saved_at}`; bộ gửi chạy debounce 2 s và khi `online` / `visibilitychange`; thành công xoá phần đã gửi; thất bại giữ nguyên và backoff 1 → 2 → 4 → 8 → 15 s (tối đa 15 s); `beforeunload` chặn khi còn `dirty`; khi tải lại: nạp máy chủ rồi **áp lại `dirty`** lên trên (bản ở máy mới hơn bản máy chủ vì chỉ một nơi ghi); xoá khoá sau khi nộp. Tương tự cho bản nháp mã (khoá `exam-code:<attempt_id>:<item>:<language>`, kèm `base_rev`).

### 4.4 Làm bài code (US-PE-06)

#### 4.4.1 Bản nháp

`code_drafts` khoá `(attempt_id, item_id, language)`; `rev` tăng 1 mỗi lần ghi; `PUT draft {language,source,base_rev}`: `base_rev` phải bằng `rev` hiện tại (lần đầu `0`), khác → 409 `DRAFT_CONFLICT`; `source` ≤ 65.536 byte; ngôn ngữ thuộc `languages` của bài.

#### 4.4.2 `Chạy thử`

`POST …/code/{item}/run` (`Idempotency-Key` bắt buộc): kiểm lượt `IN_PROGRESS`, hạn giờ, người ghi, ngôn ngữ, `source` (≤ 64 KiB, không rỗng); hạn mức cửa sổ trượt `ep:exam:run:{user_id}` = `EXAM_RUN_LIMIT` (10) trong `EXAM_RUN_WINDOW` (10 phút) — `ZADD` / `ZREMRANGEBYSCORE` / `ZCARD` bằng Lua nguyên tử; vượt → 429 với `retry_after = giây tới khi phần tử cũ nhất hết hạn` (làm tròn lên); ghi `code_submissions` (`kind=RUN`) + outbox `judge.enqueue` (Stream `judge.run`); trả 202 `{run_id}`. Chỉ test `is_sample AND approved`. Kết quả (SSE `exam.run` + `GET …/runs/{id}`): `{status, compile_ok, compile_log?, samples:[{name,verdict,time_ms,memory_kb,input?,expected?,stdout_excerpt?}]}`; `stdout_excerpt` ≤ 2 KiB.

#### 4.4.3 `Nộp lời giải`

`POST …/code/{item}/submit` (`Idempotency-Key` bắt buộc): cùng kiểm như trên; giãn cách `ep:exam:submitcd:{attempt_id}:{item_id}` (`SET NX PX EXAM_SUBMIT_COOLDOWN` 15 s; còn khoá → 429 `retry_after`); số lần `SUBMIT` của `(attempt, item)` ≥ `EXAM_SUBMISSION_CAP` (30) → 409 `SUBMISSION_LIMIT_REACHED`; ghi `code_submissions` (`kind=SUBMIT`, `source_sha256`) + outbox `judge.enqueue` (Stream `judge.submit`) **cùng transaction**; 202 `{submission_id}`. Lần nộp trước còn `QUEUED` được `SUPERSEDED` khi consumer nhận nó (4.5.7).

#### 4.4.4 Sinh viên thấy gì trong giờ thi

`GET …/submissions/{sid}` (và danh sách) chỉ trả `{id,status,language,created_at,is_final,compile_ok,compile_log?,samples:[{name,verdict,time_ms,memory_kb}]}` khi bài chưa `PUBLISHED`. Mọi trường khác (`results` của test ẩn, `passed_weight`, `total_weight`, `verdict` chung, `tests_version`) chỉ dành cho Staff; sau `PUBLISHED` sinh viên thấy `hidden:{passed,total}` trong `result` (4.8.3), vẫn không thấy từng test ẩn.

#### 4.4.5 Chốt bản tính điểm

Với mỗi câu code: `final = SUBMIT` có `created_at` lớn nhất của `(attempt, item)` (không `SUPERSEDED`). Nếu **không có `SUBMIT`** và bản nháp (ngôn ngữ cập nhật gần nhất) không rỗng → tạo `SUBMIT` `auto=true` từ nháp, đưa vào hàng chấm; nếu đã có `SUBMIT` → **không** dùng nháp. Không có gì → `earned = 0` (không phải `IE`).

### 4.5 Sandbox `go-judge` và `internal/judge` (US-PE-02) — D55, D58

#### 4.5.1 Triển khai (D58)

| Hạng mục | Giá trị |
| --- | --- |
| Sản phẩm | `criyle/go-judge` **v1.13.0** (Go, MIT) + `g++` / `gcc` **14.2.0** (Debian trixie); `deploy/judge/Dockerfile` (`FROM debian:trixie-slim`, `apt-get install g++ gcc libc6-dev`, `COPY --from=criyle/go-judge:v1.13.0 /opt/go-judge /opt/mount.yaml /opt/`) |
| Chạy | chung máy, một container `judge`; `privileged: true`, `cgroup: host` (= `--cgroupns=host`), `shm_size: 256m` |
| Lệnh | `deploy/judge/start.sh`: `exec /opt/go-judge -http-addr=:5050 -parallelism="$JUDGE_PARALLELISM" -auth-token="$JUDGE_TOKEN" $JUDGE_EXTRA_ARGS` |
| seccomp | bật mặc định (amd64: CI, VPS); `JUDGE_EXTRA_ARGS=-no-seccomp` ở `.env.local` trên colima **arm64** (v1.13.0 + seccomp hỏng ở arm64: mọi `/run` trả `lstat stdout: operation not permitted`) |
| Mạng | `judge_net` (`internal: true`) gắn `judge` + `worker`; `judge` **không** gắn mạng mặc định (không thấy DB / Redis / blob / gateway, không ra internet); không `ports`; không volume |
| Tài nguyên | `cpus: 2.0`, `mem_limit: 2g` (chỉnh qua `JUDGE_CPUS`, `JUDGE_MEM_LIMIT`) — để judge không ăn hết máy chung |
| Bí mật | `JUDGE_TOKEN` từ `.env` (≥ 16 ký tự); container không nhận biến nào khác (không `DATABASE_URL`, `REDIS_URL`, `APP_ENCRYPTION_KEY`, …); `-enable-debug` tắt |
| Cách lùi | nếu v1.13.0 + seccomp hỏng ở amd64 → PM chọn v1.12.3 (research); API phía worker không đổi |

go-judge mặc định: mount **chỉ đọc** `/lib /usr /bin …` từ image, tmpfs `/w` và `/tmp` cho mỗi lệnh, user / pid / mount / **net** / ipc / uts namespace, cgroup v2, rlimit, `no_new_privs`, bỏ capability; file giữ trong bộ nhớ của container sandbox. Rủi ro thoát sandbox = root trên máy chung — **chủ dự án đã chấp nhận** cho demo / bảo vệ (D58); tách máy riêng trước khi cho sinh viên thật dùng (nợ PR).

#### 4.5.2 Hợp đồng REST (worker → judge)

Header chung: `Authorization: Bearer <JUDGE_TOKEN>`, `Content-Type: application/json`. Đơn vị: **thời gian ns**, **bộ nhớ byte**.

*Biên dịch* (một lệnh; C++ minh hoạ, C: `args ["/usr/bin/gcc","-O2","-std=c11","-pipe","-o","a","a.c","-lm"]`, tệp `a.c`):

```json
POST /run
{"cmd":[{
  "args":["/usr/bin/g++","-O2","-std=c++17","-pipe","-o","a","a.cc"],
  "env":["PATH=/usr/bin:/bin"],
  "files":[{"content":""},{"name":"stdout","max":4096,"pipe":true},{"name":"stderr","max":8192,"pipe":true}],
  "cpuLimit":10000000000,"clockLimit":20000000000,"memoryLimit":536870912,"procLimit":50,
  "copyIn":{"a.cc":{"content":"<mã nguồn>"}},
  "copyOutCached":["a"]
}]}
→ [{"status":"Accepted","exitStatus":0,"error":"","time":…,"runTime":…,"memory":…,"procPeak":…,
    "files":{"stdout":"","stderr":"…"},"fileIds":{"a":"<file id>"}}]
```

*Chạy một test* (mỗi test một lệnh; `stdin` = `files[0].content`):

```json
POST /run
{"cmd":[{
  "args":["a"],"env":["PATH=/usr/bin:/bin"],
  "files":[{"content":"<input>"},{"name":"stdout","max":<output_limit_kb*1024>,"pipe":true},{"name":"stderr","max":4096,"pipe":true}],
  "cpuLimit":<time_limit_ms*1000000>,"clockLimit":<3*cpuLimit>,"memoryLimit":<memory_limit_mb*1048576>,"procLimit":1,
  "copyIn":{"a":{"fileId":"<file id>"}}
}]}
→ [{"status":"Accepted|Time Limit Exceeded|Memory Limit Exceeded|Output Limit Exceeded|Nonzero Exit Status|Signalled|…",
    "exitStatus":0,"time":<cpu ns>,"runTime":<wall ns>,"memory":<peak bytes>,"procPeak":1,"files":{"stdout":"…"}}]
```

*Dọn:* `DELETE /file/<file id>` sau khi chấm xong (cả khi lỗi / huỷ). *Sức khoẻ:* `GET /version` (không cần token).

**Quy tắc phía worker:** biên dịch **một lần** mỗi bản nộp, chạy mọi test bằng `fileId` đã cache; mỗi lời gọi có `context` hạn = `clockLimit + JUDGE_HTTP_SLACK` (5 s); semaphore trong tiến trình = `JUDGE_PARALLELISM` (go-judge xếp hàng bên trong nhưng HTTP treo lâu — research); không ghi đĩa; tên tệp nguồn cố định `a.c` / `a.cc`; nội dung test gửi bằng `content` (đọc từ DB / blob); **không gửi danh tính sinh viên** hay bất kỳ định danh nào tới judge.

#### 4.5.3 Giới hạn mặc định (research mục 3)

| | Biên dịch | Chạy (mỗi test) |
| --- | --- | --- |
| CPU | 10 s | `time_limit_ms` của bài, mặc định **1.000 ms** (100…10.000) |
| Đồng hồ | 20 s | 3 × CPU |
| Bộ nhớ | 512 MiB | `memory_limit_mb`, mặc định **256 MiB** (16…1.024) |
| Tiến trình | 50 (cc1plus, as, ld) | **1** (không đa luồng) |
| Đầu ra | `stdout` 4 KiB, `stderr` 8 KiB | `stdout` ≤ `output_limit_kb`, mặc định **1.024 KiB** (1…16.384); `stderr` 4 KiB |
| Mã nguồn | ≤ `JUDGE_MAX_SOURCE_BYTES` 64 KiB | – |

Cờ cố định, **không** nhận cờ từ sinh viên: C `gcc -O2 -std=c11 -pipe -o a a.c -lm`; C++ `g++ -O2 -std=c++17 -pipe -o a a.cc`.

#### 4.5.4 Phân loại kết quả

Từ `status` của go-judge (chuỗi đã thấy ở PoC) và checker:

| `status` go-judge / tình huống | Verdict | Ghi chú |
| --- | --- | --- |
| biên dịch `status ≠ Accepted` | `CE` | `compile_log` = `stderr` cắt 8 KiB, thay đường dẫn bằng `main.c` / `main.cpp`; nếu bị `Time Limit Exceeded` / `Memory Limit Exceeded` khi biên dịch → `CE` kèm câu "Biên dịch quá thời gian / bộ nhớ" |
| `Accepted` + so khớp đạt | `AC` | |
| `Accepted` + so khớp không đạt | `WA` | |
| `Time Limit Exceeded` | `TLE` | CPU hoặc đồng hồ; vòng lặp in vô hạn cũng ra `TLE` (stdout bị cắt ở giới hạn rồi hết giờ) |
| `Memory Limit Exceeded` | `MLE` | |
| `Output Limit Exceeded` | `OLE` | in hữu hạn nhưng quá giới hạn |
| `Nonzero Exit Status`, `Signalled`, `Dangerous Syscall` | `RE` | segfault = `Signalled` |
| `Internal Error`, `File Error`, HTTP 5xx, mất kết nối, hết hạn gọi, phản hồi không giải mã được | **`IE`** | lỗi hệ thống: **không tính điểm**, thử lại theo hàng đợi; không bao giờ ghi thành 0 điểm |

Verdict chung của bản nộp: `CE` nếu biên dịch lỗi; `IE` nếu có test `IE` sau thử lại; `AC` nếu mọi test `AC`; ngược lại verdict của test lỗi đầu tiên theo `position`. Điểm **không** phụ thuộc verdict chung mà theo trọng số test đạt.

#### 4.5.5 Bộ so khớp (checker) — chạy trong worker, Go thuần

| `checker` | Quy tắc |
| --- | --- |
| `EXACT` (mặc định; = cách `norm` của PoC) | đổi `\r\n` → `\n`; bỏ khoảng trắng / tab ở **cuối mỗi dòng**; bỏ các dòng trống ở **cuối tệp**; so từng byte |
| `TOKENS` | tách theo khoảng trắng Unicode; so danh sách token |
| `FLOAT_EPS` | như `TOKENS`; hai token cùng là số thập phân hữu hạn (`strconv.ParseFloat`, không NaN / Inf) đạt khi `|a − b| ≤ eps × max(1, |b|)`; token khác so bằng chuỗi |

Đầu ra rỗng khớp `expected` rỗng. Checker tuỳ chỉnh (chương trình testlib chạy như lệnh thứ hai trong go-judge) có trong research nhưng **ngoài phạm vi PE** (ghi nợ); nếu cần, thêm sau mà không đổi hợp đồng này.

#### 4.5.6 Ca tấn công bắt buộc (`TestSandboxAttacks`)

Mỗi ca là một bản nộp C / C++ đi qua **đường chấm thật** (không gọi go-judge trực tiếp). "PoC" = đã chạy thật ở research trên colima (arm64, `-no-seccomp`, 4 CPU); "xác minh" = dev chạy và ghi số đo vào handoff.

| # | Ca | Kỳ vọng | PoC |
| --- | --- | --- | --- |
| A1 | vòng lặp vô hạn `for(;;) x++` | `TLE` (CPU ≈ 1.001 ms) | ✓ |
| A2 | in vô hạn `for(;;) puts(…)` | `TLE`; stdout bị cắt ở giới hạn; RAM ổn định | ✓ |
| A3 | in hữu hạn 5 MB | `OLE` (độ dài ghi nhận = giới hạn + 1) | ✓ |
| A4 | in 1 GB hữu hạn nhanh (`fwrite` lặp) | `OLE` hoặc `TLE`; RAM / đĩa máy chủ không tăng quá 10 MiB | xác minh |
| A5 | cấp phát dần tới 512 MiB (`malloc` + `memset` mỗi 1 MiB) | `MLE` | ✓ |
| A6 | cấp phát 2 GB một lần + `memset` | `MLE` hoặc `RE` (`malloc` trả NULL → chương trình thoát ≠ 0); container sống | xác minh |
| A7 | truy cập con trỏ NULL | `RE` (`Signalled`) | ✓ |
| A8 | fork bomb (`fork()` lặp, con cũng `fork()`) | không tạo được tiến trình thứ hai (`procPeak = 1`); `TLE` hoặc `RE`; không tiến trình mồ côi | ✓ |
| A9 | `fopen` đọc `/etc/shadow`, `/etc/passwd`, `/proc/1/environ`, `/opt/go-judge`, `/root/.bashrc` | **cả 5 đường bị chặn** (chương trình in `CHAN` cho từng đường; test kiểm đầu ra = `CHAN×5`) | ✓ |
| A10 | `socket` + `connect` tới `1.1.1.1:53` | `socket` được, `connect = -1` (không ra mạng) | ✓ |
| A10b | `connect` tới `postgres:5432` và `redis:6379` (tên dịch vụ compose) và địa chỉ IP của chúng | `connect = -1` (mạng riêng không thấy) | xác minh |
| A11 | `#include "/dev/random"` | `CE` (cc1plus bị giết vì hết bộ nhớ trong ≤ 25 s) | ✓ |
| A12 | bom biên dịch: mẫu `template` đệ quy sâu / `constexpr` vòng lặp lớn | `CE` "quá thời gian / bộ nhớ" trong ≤ 25 s; không ảnh hưởng bài khác | xác minh |
| A13 | ghi tệp 1 GB vào `/tmp` hoặc `/w` | bị chặn (hết chỗ tmpfs / `MLE` / `RE`); đĩa máy chủ không đổi | xác minh |
| A14 | `execve("/bin/sh")` / `system("ls /; cat /etc/shadow")` | không tạo được tiến trình con (`procLimit=1`); không đọc được ngoài vùng cho phép; kết thúc `RE` / `WA` | xác minh |
| A15 | rò dữ liệu giữa hai lượt chạy: lượt 1 ghi `/tmp/leak` và `/w/leak`, lượt 2 (ngay sau) đọc | lượt 2 **không** thấy (`CHAN`) | xác minh |

Điều kiện chung mọi ca: (a) trong lúc chạy ca tấn công, một bản nộp đúng của sinh viên khác vẫn `AC` trong ≤ 20 s; (b) `docker inspect … RestartCount` giữ nguyên; (c) bộ nhớ container `judge` về mức nền (≤ 100 MiB, PoC 35,6 MiB lúc rảnh) trong 30 s; (d) đĩa máy chủ đổi ≤ 10 MiB; (e) mọi `fileId` được xoá (`GET /file` không liệt kê gì còn lại sau 60 s — nếu go-judge hỗ trợ liệt kê; nếu không, bộ nhớ `/dev/shm` của container về mức nền).

#### 4.5.7 Hàng đợi và consumer

| Stream | Nhà sản xuất | Khoá idempotent | Ghi chú |
| --- | --- | --- | --- |
| `judge.submit` | outbox `judge.enqueue` (bản nộp `SUBMIT`, chấm lại) | `(submission_id, tests_version)` | consumer group `judge`; dead-letter `judge.submit.dead` |
| `judge.run` | outbox `judge.enqueue` (`Chạy thử`) | `submission_id` | cùng nhóm; **ít nhất 1** goroutine chỉ đọc `judge.run` (không chờ sau hàng nộp) |

Consumer (mỗi goroutine, `JUDGE_WORKERS` = 4): `XREADGROUP` chặn 2 s (goroutine chung ưu tiên `judge.run`) → nạp `code_submissions`; nếu `status ∈ {DONE, SUPERSEDED}` ở đúng `tests_version` → `XACK`; nếu `kind=SUBMIT`, đang `QUEUED` và tồn tại `SUBMIT` mới hơn của `(attempt, item)` → `status=SUPERSEDED`, `XACK`, **không** chạy sandbox; `UPDATE … SET status='RUNNING', attempts=attempts+1 WHERE id=$1 AND status IN ('QUEUED','RUNNING')`; đọc `code_problems.tests_version` và test `approved`; chấm (4.5.2–4.5.5) trong semaphore; ghi kết quả bằng `UPDATE … WHERE id=$1` (idempotent: cùng đầu vào cho cùng kết quả; ghi lại `tests_version` đã dùng); `XACK`; outbox `exam.submission_done` (SSE cho chủ + kiểm hoàn tất lượt 4.8.1). Lỗi → không `XACK`; thử lại 3 lần backoff 1 s / 5 s / 30 s (`attempts`); lần thứ 4 → `status=ERROR`, `verdict=IE`, `XADD judge.submit.dead`, `XACK`; tin treo quá `JUDGE_CLAIM_IDLE` (60 s) được `XAUTOCLAIM`. Không lưu `source` / `input` trong tin của Stream (chỉ `submission_id`).

### 4.6 Tính điểm (US-PE-01, US-PE-08)

```go
package exam
type ItemResult struct { Points, Earned decimal.Decimal; Void bool }
func Score(items []ItemResult, maxScore, step decimal.Decimal) decimal.Decimal
// raw = Σ (Void ? Points : Earned); total = Σ Points
// score = RoundToStep( raw.DivRound(total, 16) * maxScore , step ); kẹp [0, maxScore]
// RoundToStep(x, step) = x.Div(step).Round(0).Mul(step)   // Round: nửa lên (shopspring: ra xa số 0; điểm không âm)
```

`earned` của câu: trắc nghiệm theo 4.1.3 (đáp án đối chiếu = `override.answer_key` nếu có, nếu không `answer_key`); câu `void` = `points`; câu code = `points × Σweight(test AC) ÷ Σweight(test approved)` (`DivRound(…, 16)`); không bản nộp = 0. `Σweight = 0` không xảy ra ở bài đã lên lịch (kiểm ở `schedule`); nếu gặp → lỗi cấu hình, lượt ở `GRADING` + nhật ký, không chia cho 0. Điểm chính thức của lượt = `COALESCE(adjusted_score, auto_score)`.

**Ví dụ kiểm tay (có trong `TestScoringDecimal`):** `max_score=10`; Q1 `MCQ_SINGLE` 1 đ đúng → 1; Q2 `MCQ_MULTI` 2 đ, K = 3, chọn 2 đúng 0 sai, `PARTIAL` → `2 × 2/3 = 1,3333…`; Q3 code 3 đ, trọng số `[1,1,2]`, đạt test 1 và 3 → `3 × 3/4 = 2,25`; `raw = 4,58333…`, `total = 6` → `7,63888…`; bước `0.25` → `7,75`; bước `0.01` → `7,64`; bước `1` → `8`.

### 4.7 Liêm chính (US-PE-07) — D56, không phải giám thị

**4.7.1 Khoá chat.** `ep:exam_lock:{user_id}` (String) = `attempt_id`; TTL = `max(deadline_at của các lượt IN_PROGRESS của người đó) − now + EXAM_GRACE_SECONDS`; đặt khi bắt đầu / làm tiếp / `extend`; xoá khi **không còn** lượt `IN_PROGRESS`; sau mỗi lần đặt / xoá, DB là nguồn sự thật. **4.7.2 `exam.Locker`**:

```go
type Lock struct { AttemptID, ExamID uuid.UUID; Until time.Time }
type Locker interface { IsLocked(ctx context.Context, userID uuid.UUID) (Lock, bool, error) }
```

Thứ tự: Redis `GET` trúng → `true` (đọc `Until` từ TTL); trượt hoặc Redis lỗi → truy vấn `SELECT … FROM exam_attempts WHERE student_id=$1 AND status='IN_PROGRESS' AND deadline_at + grace > now() ORDER BY deadline_at DESC LIMIT 1` (chỉ mục từng phần `exam_attempts_running_idx`) rồi nạp lại Redis nếu sống; DB cũng lỗi → trả `error` — **người gọi (chat, P3) phải từ chối** (an toàn khi nghi ngờ). `GET /me/exam-lock` → `{locked,until?}`; route thử `GET /_test/chat-gate` → `{allowed: !locked}`.

**4.7.3 Sự kiện.** `POST …/attempts/{aid}/events {events:[{type,client_at,meta}]}` (≤ 50 / yêu cầu, máy khách gửi gộp ~15 s và khi `pagehide` bằng `navigator.sendBeacon`/`fetch keepalive`): loại `TAB_HIDDEN` (`meta.duration_ms` khi quay lại có ở `TAB_VISIBLE`), `TAB_VISIBLE` (`duration_ms` = thời gian ẩn), `PASTE` (`chars`, `item_id`), `OFFLINE`, `ONLINE`; máy chủ tự ghi `TAB_TAKEOVER`; P3 ghi `CHAT_BLOCKED`. Khoá `meta` cho phép: `duration_ms`, `chars`, `item_id` (khoá khác bị bỏ); `occurred_at` = giờ máy chủ; tối đa 500 / lượt (dư bị bỏ, 204). Không nhận / không lưu nội dung clipboard, mã, IP, user-agent. Lỗi ghi sự kiện không bao giờ làm hỏng lưu bài. Chỉ `TEACHER` đọc (`GET …/exams/{eid}/events`, tóm tắt trong chi tiết lượt). Máy khách gắn `paste` ở ô soạn mã và các `textarea` của bài; `visibilitychange` / `blur` cho rời tab. **Không có điểm trừ**: `score*.go` không đọc `exam_events` / `similarity_reports` (kiểm bằng `go/parser`).

**4.7.4 So độ giống mã (`internal/exam/similarity`).**

| Bước | Quy tắc |
| --- | --- |
| Đầu vào | `SUBMIT` cuối của từng sinh viên cho mỗi bài code (kể cả `auto`); bài `CLOSED`; bỏ cặp cùng sinh viên |
| Token hoá | bộ tách từ vựng C / C++ tự viết: bỏ chú thích (`//`, `/* */`) và khoảng trắng; **bỏ dòng `#include`** (các `#define` giữ, token hoá); định danh không phải từ khoá → `I`; hằng số nguyên / thực → `N`; chuỗi / ký tự → `S`; từ khoá (danh sách C11 + C++17 nhúng trong `keywords.go`) và toán tử / dấu câu giữ nguyên |
| Dấu vân tay | k-gram `k = 5` token, băm FNV-1a 64 bit; winnowing cửa sổ `w = 4` (chọn băm nhỏ nhất, hoà thì lấy bên phải nhất); tập dấu vân tay (không lặp) |
| Trừ khung | loại khỏi tập của mỗi bài các dấu vân tay có trong `starter_code` của bài (cùng ngôn ngữ); **không** trừ lời giải mẫu (không công khai) |
| Bỏ qua | bản nộp < 30 token sau chuẩn hoá |
| Độ giống | **Jaccard** `|Fa ∩ Fb| ÷ |Fa ∪ Fb|` (số thực 0…1, 3 chữ số; đây là ngoại lệ chủ ý của luật `float64`: không phải điểm); đối xứng, tất định |
| Lưu | cặp có `score ≥ 0,40` và ≥ 10 dấu vân tay chung; tối đa 200 cặp / bài (điểm cao nhất); `flagged = score ≥ max(SIMILARITY_MIN, mean + 3·stddev)` của phân phối điểm **mọi** cặp của bài trong lớp (research: nền ≈ 0,27 do khung chung, không dùng ngưỡng cố định); cặp lưu theo thứ tự chuẩn `attempt_a < attempt_b` |
| Quy mô | chỉ mục ngược `dấu vân tay → bản nộp`, chỉ so cặp chia sẻ ≥ 1 dấu vân tay; 1.000 bản nộp ≤ 30 s; chạy ở worker (job `exam.similarity`, 202) |
| Riêng tư | mã **không rời hệ thống**: gói không import `net/http`, không dịch vụ ngoài (MOSS không dùng; JPlag / Dolos chỉ dùng tay ngoài hệ thống) |

Kích hoạt: tự động một lần sau khi bài `CLOSED` và mọi lượt đã vào `GRADED` (outbox `exam.graded_all` → job); Giảng viên chạy lại bằng `POST …/similarity/run` (`run_id` mới, bản cũ giữ). Xử lý cặp: `NEW → CLEARED | FOLLOW_UP` (kèm ghi chú ≤ 500). Việc `EXAM_SIMILARITY` ở "Hôm nay" đếm cặp `flagged AND review_state='NEW'` của lần chạy mới nhất.

### 4.8 Đóng, chấm, công bố, kết quả, phúc khảo (US-PE-08)

**4.8.1 Hoàn tất chấm một lượt (`exam.tryFinishGrading(attempt)`, idempotent).** Gọi khi một bản nộp `DONE` / `ERROR`, khi tick, sau `regrade`. Điều kiện: lượt `GRADING`; với mỗi câu code, bản nộp tính điểm (4.4.5) ở `DONE` (hoặc câu không có bản nộp). Có bản nộp `ERROR` / `IE` hoặc còn `QUEUED` / `RUNNING` → giữ `GRADING`. Đủ → tính `Score` (4.6), `UPDATE exam_attempts SET status='GRADED', auto_score=…, breakdown=…, graded_at=now() WHERE id=$1 AND status='GRADING'`, SSE `exam.attempt`.

**4.8.2 Công bố.** Mỗi vòng tick, với bài `CLOSED`:

```sql
UPDATE exams SET status='PUBLISHED', published_at=now(), version=version+1
WHERE id=$1 AND status='CLOSED' AND NOT publish_hold AND NOT regrading
  AND NOT EXISTS (SELECT 1 FROM exam_attempts a WHERE a.exam_id=exams.id AND a.status <> 'GRADED')
RETURNING id
```

(0 dòng → chưa đủ điều kiện hoặc đã công bố: không làm gì). Trúng → cùng transaction ghi outbox `exam.published`; handler: thông báo `EXAM_PUBLISHED` (khử trùng `exam.published:<exam_id>:<user_id>`) cho sinh viên **có lượt**, SSE `exam.published`, xoá cache "Hôm nay". `regrading=true` được đặt khi nhận yêu cầu chấm lại / `override` lớn và `false` khi job kết thúc (kể cả lỗi).

**4.8.3 `GET …/attempts/{aid}/result` (chỉ `PUBLISHED`).**

```json
{"exam":{"id":"…","title":"…","max_score":"10.00","published_at":"…","reveal_answers":true,"appeal_days":7,"appeal_open_until":"…"},
 "score":"7.75","score_adjusted":false,"appeal":{"status":null},
 "items":[
  {"item_id":"…","position":1,"type":"MCQ_SINGLE","stem":"…","options":[{"id":"…","body":"…"}],
   "earned":"1.00","max":"1.00","correct":true,"mine":{"option_ids":["…"]},
   "answer":{"option_ids":["…"]},"explanation":"…","overridden":false},
  {"item_id":"…","position":11,"type":"CODE","stem":"…","earned":"2.25","max":"3.00",
   "samples":[{"name":"sample1","verdict":"AC","time_ms":3,"memory_kb":1200,"input":"1 2\n","expected":"3\n"}],
   "hidden":{"passed":3,"total":4},"final_submission":{"id":"…","language":"cpp17","source":"…","created_at":"…","compile_ok":true}}]}
```

`answer` và `explanation` chỉ có khi `reveal_answers=true`; `correct` / `earned` luôn có sau công bố; câu `void` có `overridden:true`, `correct:null`; `hidden` chỉ **số**; không trọng số, không tên / input / expected / verdict từng test ẩn; không `reference*`; nếu `override` có thì chỉ cờ `overridden`. Điểm = điểm chính thức (`adjusted_score` nếu có).

**4.8.4 Sửa điểm tay, `override`, chấm lại.**

| Thao tác | Điều kiện | Tác dụng | Thông báo / audit |
| --- | --- | --- | --- |
| `PUT …/results/{aid}/score {score,reason,version}` | `GRADED`; Giảng viên | `adjusted_score` (đúng `rounding_step`, 0…`max_score`), `adjusted_reason` 1…500, `adjusted_by/at`; `auto_score` giữ nguyên; `score:null` gỡ điều chỉnh (cần lý do) | `audit_log exam.score_adjust`; sau `PUBLISHED`: `EXAM_REGRADED` (khử trùng `exam.regraded:<attempt_id>:v<version>`) |
| `PUT …/items/{itemId}/override {answer_key\|void,reason}` | `CLOSED` / `PUBLISHED`; câu MCQ / TRUE_FALSE; Giảng viên | `exam_items.override`; tính lại mọi lượt `GRADED` (≤ 200 lượt: đồng bộ; nhiều hơn: job 202); `void` = trọn điểm cho mọi lượt | `exam.item_override`; sau `PUBLISHED`: `EXAM_REGRADED` cho lượt đổi điểm |
| `POST …/regrade {scope,item_id?,attempt_id?,reason}` | `CLOSED` / `PUBLISHED`; Giảng viên; `Idempotency-Key` | 202; đặt `regrading=true`; job `exam.regrade` đặt lại `status=QUEUED`, `tests_version=NULL` cho bản nộp **cuối** trong phạm vi (`all` / `item` / `attempt` / `errors`) và xếp vào `judge.submit`; khi xong tính lại điểm; `regrading=false` | `exam.regrade`; sau `PUBLISHED`: `EXAM_REGRADED`; **không** huỷ công bố |

Chấm lại idempotent theo `(submission_id, tests_version)`; sandbox chết trong khi chấm lại → job ở lại, tick nhặt tiếp (không mất `regrading`).

**4.8.5 Phúc khảo.** `POST …/attempts/{aid}/appeal {reason}` (`Idempotency-Key`): bài `PUBLISHED`, lượt `GRADED` của người gọi, `appeal_days > 0`, `now ≤ published_at + appeal_days`, chưa có `exam_appeals` của lượt; `reason` 1…1.000; ghi `exam_appeals` `OPEN` + outbox `exam.appeal_created` → thông báo `EXAM_APPEAL_NEW` (Giảng viên + TA) và việc `EXAM_APPEAL` (Giảng viên). `POST …/appeals/{id}/answer {decision,response,score?,version}` (Giảng viên): `UPHELD` | `ADJUSTED` (+ `score`, áp như `PUT score` trong cùng transaction); một lần; outbox `exam.appeal_answered` → `EXAM_APPEAL_REPLIED` + việc `EXAM_APPEAL_REPLY` cho sinh viên. Không route nào của phúc khảo gọi LLM. P4 có thể chuyển sang `escalation_tickets` `GRADE_APPEAL` (đề xuất ở mục 10).

**4.8.6 CSV.** Streaming; BOM; `;`; số thập phân dấu phẩy; cột: `mssv;ho_ten;trang_thai;diem_tu_dong;diem_chinh_thuc;nop_luc;ly_do_nop` + `cau_<position>` (điểm từng câu); `trang_thai` ∈ `ABSENT|IN_PROGRESS|GRADING|GRADED`; ô bắt đầu bằng `= + - @` thêm `'` phía trước; tối đa 5.000 dòng; không cột liêm chính.

**4.8.7 Thống kê.** `distribution`: các khoảng độ rộng `max_score ÷ 10` (thang 10 → khoảng 1 điểm), nửa mở `[a,b)` trừ khoảng cuối đóng; `mean`, `median` (decimal, 2 chữ số); `hardest`: 5 câu trắc nghiệm có tỉ lệ đúng thấp nhất (loại câu `void`); `code`: điểm trung bình theo tỉ lệ và tỉ lệ `CE` mỗi bài code; chỉ từ lượt `GRADED`; không định danh.

### 4.9 Gợi ý nháp bằng AI (US-PE-03) — D57

Job `question.suggest` (không qua SSE riêng: `job.progress`). **Chỉ** gọi `internal/llm` ở `internal/exam/suggest.go`: `llm.Client.Structured(ctx, Request{Task: QUESTION_GEN, Lane: nil /* mặc định BATCH */, Messages, Params{Temperature: 0.4}}, schema)`; không tự viết thư viện JSON Schema — việc kiểm schema phía Go do `internal/llm` đảm bảo (`FEAT-llm-gateway` 4.2); PE kiểm thêm **quy tắc nghiệp vụ** của AC1 trên kết quả đã parse. Một job = đúng một lời gọi `Structured` (D47 không áp cho job nền, nhưng giữ tinh thần); `count` ≤ 10.

Schema MCQ (rút gọn): `{"questions":[{"type":"MCQ_SINGLE|MCQ_MULTI|TRUE_FALSE","title","stem","options":[{"body","correct":bool}],"value":bool?,"explanation","difficulty":"EASY|MEDIUM|HARD"}]}`. Lời nhắc hệ thống (tiếng Việt, cố định ở `prompts/suggest_mcq.txt`): vai trò "người ra đề trắc nghiệm", ràng buộc số đáp án, "không bịa số liệu", "chỉ dựa vào `source_text` nếu có", "không nêu tên người". **Dữ liệu vào prompt** chỉ gồm chủ đề, độ khó, số câu, `source_text` (người dùng nhập, ≤ 6.000 ký tự) — qua lớp che danh tính của `internal/llm` như mọi lời gọi. Câu hợp lệ → `question_bank` `origin=AI_DRAFT`, `review_status=PENDING`, `ai_job_id`; câu sai quy tắc → `dropped` kèm lý do.

Job `CODE_TESTS`: đầu vào cho AI: `stem` + giới hạn + (tuỳ chọn) 2 test mẫu của bài; schema `{"inputs":[{"name","input","note"}]}`; mỗi `input` ≤ 64 KiB, UTF-8; worker chạy **lời giải mẫu** qua sandbox với từng `input` (cùng giới hạn của bài); verdict chạy xong (`Accepted` về mặt sandbox) → `expected` = `stdout` đã chuẩn hoá cuối dòng; các verdict khác → `dropped[]`. Test lưu `source='AI_DRAFT'`, `approved=false`, `is_sample=false`, `weight=1`. AI không bao giờ sinh `expected`. Duyệt hàng loạt: `POST …/testcases/approve {ids[]}` (đặt `approved=true`, tăng `tests_version`, gỡ `reference_verified_version` — vì tập test đổi — người dùng chạy lại `verify`).

### 4.10 "Hôm nay", thông báo, sự kiện outbox, SSE

**Nguồn việc đăng ký thêm** (`FEAT-course-foundation` 4.7; số nhỏ = gấp hơn; `Item.Kind` mới):

| Vai | Bậc | Kind | Tiêu đề / lý do | `href` |
| --- | --- | --- | --- | --- |
| Sinh viên | 5 | `EXAM_IN_PROGRESS` | "Bài thi {tiêu đề} đang làm dở" · "Còn {m} phút. Làm tiếp." | `/exams/{id}/take` |
| | 8 | `EXAM_OPEN` | "Bài thi {tiêu đề} đang mở" · "Mở đến {HH:mm dd/MM}. Bạn có {n} phút để làm." | `/exams/{id}/take` |
| | 42 | `EXAM_UPCOMING` | "{tiêu đề} · {Thứ…, HH:mm}" · "Làm trong {n} phút. Chuẩn bị máy tính nếu có bài lập trình." | `/exams/{id}/take` |
| | 44 | `EXAM_RESULT` | "Điểm bài thi {tiêu đề} đã công bố" · "Xem kết quả và đáp án." (7 ngày kể từ công bố) | `/exams/{id}/take` |
| | 46 | `EXAM_APPEAL_REPLY` | "Giảng viên đã trả lời yêu cầu xem lại" | `/exams/{id}/take` |
| Giảng viên / TA | 15 | `EXAM_GRADE_ERROR` | "{N} bài code chấm lỗi hệ thống · {tên bài thi}" · "Điểm chưa công bố được. Chấm lại khi hệ thống chấm hoạt động." | `/exams/{id}/results` |
| Giảng viên | 22 | `EXAM_APPEAL` | "{N} yêu cầu xem lại điểm · {tên bài thi}" · "Cũ nhất đã chờ {tuổi}." | `/exams/{id}/results?tab=appeals` |
| | 32 | `EXAM_PUBLISH_HOLD` | "Bài thi {tiêu đề} đã chấm xong nhưng đang hoãn công bố" | `/exams/{id}/results` |
| | 55 | `EXAM_SIMILARITY` | "{N} cặp bài code nghi giống nhau · {tên bài thi}" · "Chỉ là gợi ý — xem và quyết định." | `/exams/{id}/results?tab=similarity` |
| Giảng viên / TA | 90 | `QUESTION_REVIEW` | "{N} câu hỏi chờ duyệt · lớp {mã lớp}" | `/questions?review_status=PENDING&course={id}` |

Provider chỉ đọc bằng một truy vấn tổng hợp mỗi nguồn (ngân sách truy vấn của 4.7 P2); `Overdue` do Provider đặt (`EXAM_APPEAL` có yêu cầu chờ > 48 giờ).

**Thông báo** (`notifications`, `type` mới): `EXAM_SCHEDULED`, `EXAM_PUBLISHED`, `EXAM_REGRADED`, `EXAM_APPEAL_NEW`, `EXAM_APPEAL_REPLIED` (không mail ở PE — chỉ chuông; mail do P4 nối). `dedupe_key`: `exam.scheduled:<exam_id>:<user_id>`, `exam.published:<exam_id>:<user_id>`, `exam.regraded:<attempt_id>:v<n>`, `exam.appeal_new:<appeal_id>:<user_id>`, `exam.appeal_replied:<appeal_id>`.

**Outbox** (mỗi topic cùng transaction với thay đổi; handler idempotent):

| Topic | Payload | Handler |
| --- | --- | --- |
| `exam.scheduled` / `exam.unscheduled` | `{exam_id,course_id}` | `notify.exam_scheduled`, `today.invalidate` |
| `exam.opened` / `exam.closed` | `{exam_id,course_id}` | `today.invalidate`; `exam.closed` → kích hoạt hoàn tất chấm |
| `exam.attempt_started` / `exam.attempt_submitted` | `{exam_id,attempt_id,user_id}` | `today.invalidate` (người đó) |
| `exam.graded_all` | `{exam_id}` | job `exam.similarity` |
| `exam.published` | `{exam_id,course_id}` | `notify.exam_published`, `today.invalidate`, SSE `exam.published` |
| `exam.regraded` | `{exam_id,attempt_ids[]}` | `notify.exam_regraded` (chỉ khi `PUBLISHED`) |
| `exam.appeal_created` / `exam.appeal_answered` | `{appeal_id,…}` | `notify.*`, `today.invalidate` |
| `question.reviewed` | `{course_id}` | `today.invalidate` (Staff) |
| `judge.enqueue` | `{submission_id,kind}` | `XADD judge.submit` hoặc `judge.run` |
| `exam.submission_done` | `{submission_id,attempt_id}` | SSE `exam.submission` / `exam.run` cho chủ; `tryFinishGrading` |

**SSE** (kênh theo người, `FEAT-pg-foundation` 6.8): `exam.run` `{run_id,status,…}`, `exam.submission` `{submission_id,item_id,status,compile_ok?}`, `exam.attempt` `{attempt_id,status}`, `exam.published` `{exam_id}`, và `job.progress` có sẵn cho `reference/verify`, `suggest`, `regrade`, `similarity`. Sự kiện **không** chứa mã, test, điểm hay verdict của test ẩn (chỉ trạng thái và dữ liệu của test mẫu cho `exam.run`).

### 4.11 Seed (US-PE-09)

Mở rộng `scripts/seed.mjs` (cùng quy tắc: HTTP API thật, không `_test`, không ghi DB, idempotent theo khoá tự nhiên, chặn `APP_ENV=production`):

| Dữ liệu (lớp 1; lớp 2 không có) | Số lượng |
| --- | --- |
| Câu trắc nghiệm `APPROVED` | 20: 12 `MCQ_SINGLE`, 4 `MCQ_MULTI`, 4 `TRUE_FALSE`; 5 chủ đề (Mật mã đối xứng, Mật mã công khai, Hàm băm, Xác thực, Mạng) |
| Câu `AI_DRAFT` `PENDING` | 5 (qua `questions/suggest`, provider `fake`) |
| Bài code `APPROVED` | 2 (`gcd-lon-nhat`, `dem-tu`): mỗi bài 2 test mẫu + 3 test ẩn; trọng số 1/1/2/2/2 và 1/1/1/1/1; lời giải mẫu đã xác minh |
| Bài thi | `Kiểm tra tuần 9 — Mật mã` (`MCQ`, 10 câu, 45 phút, `multi_scoring=PARTIAL`); `Kiểm tra tuần 9 — Lập trình` (`CODE`, 2 bài, 60 phút) |
| Lượt làm | 24 (trắc nghiệm) + 6 (code); A đúng 10/10; B 8/10; C không làm; D không thuộc lớp; cặp nộp mã giống nhau (đổi tên biến) |
| Kết quả mong đợi | `seed/expected_exam_scores.csv` (tính tay: 30 dòng + `expected_verdict`) |

Cửa sổ ngắn: `docker-compose.local.yml` / `docker-compose.test-seed.yml` đặt `EXAM_MIN_DURATION_MINUTES=1`, `EXAM_MIN_LEAD_SECONDS=5` (không có ở production). Seed tạo bài (mở sau 10 s, đóng sau 150 s), cho sinh viên làm ngay, rồi **thoát** — bộ lập lịch tự đóng và công bố trong ≤ 3 phút. `scripts/check-exam-seed.mjs bank|scores|demo` kiểm.

### 4.12 Danh sách FR

| FR | Nội dung | AC |
| --- | --- | --- |
| FR-1 | Migration `00005` 13 bảng dạng cuối, enum, CHECK, chỉ mục (`course_id` đầu); không ALTER | 01-AC1, AC2, AC3 |
| FR-2 | Guard cho route PE: `Member` / `Staff` / `Teacher`; ADMIN bị chặn; sinh viên chỉ route của mình; cách ly lớp | 01-AC4, AC5 |
| FR-3 | Quiz Engine (`MCQ_SINGLE`, `MCQ_MULTI` hai chế độ, `TRUE_FALSE`) bằng `decimal`, không LLM | 01-AC6, AC8 |
| FR-4 | Công thức điểm bài thi (bước làm tròn, `void`, ví dụ 7,75 / 7,64) khớp bảng tính tay | 01-AC7 |
| FR-5 | `version` → 409; cursor; `audit_log` không lộ nội dung; lớp lưu trữ; mã lỗi + contract | 01-AC9…AC13 |
| FR-6 | Container `judge` (go-judge v1.13.0, privileged + cgroup host, mạng riêng, token, tài nguyên) | 02-AC1, AC2, AC3 |
| FR-7 | Client `internal/judge` theo hợp đồng REST, phân loại kết quả, checker, kết quả bản nộp | 02-AC4…AC7 |
| FR-8 | Hàng đợi `judge.submit` / `judge.run`: idempotent, retry 3, dead-letter, thay bản cũ, sandbox chết | 02-AC8, AC9, AC10 |
| FR-9 | 15 ca tấn công, hiệu năng, nhật ký sạch, sức khoẻ, seccomp theo môi trường | 02-AC11…AC15 |
| FR-10 | Soạn câu trắc nghiệm / đúng–sai / bài code; test; zip; lời giải mẫu | 03-AC1…AC6 |
| FR-11 | Duyệt, khoá sửa, nhân bản, danh sách / lọc / quyền, UI `/questions`, nguồn việc | 03-AC7, AC8, AC11…AC14 |
| FR-12 | AI gợi ý nháp (MCQ; đầu vào test + đầu ra từ lời giải mẫu) | 03-AC9, AC10 |
| FR-13 | Tạo / sửa / chọn câu / xem trước / lên lịch / bỏ lịch / gia hạn / xoá / nhân bản bài thi | 04-AC1…AC8, AC10 |
| FR-14 | Máy trạng thái + bộ lập lịch + nguồn việc sinh viên + UI `/exams` | 04-AC6, AC9, AC11, AC12, AC13 |
| FR-15 | Bắt đầu / làm tiếp / một lượt / đồng hồ máy chủ / xáo tất định | 05-AC1…AC5 |
| FR-16 | Tự lưu, hết giờ tự nộp, nộp idempotent, hai tab, mất mạng, UI 375 px, xác nhận nộp | 05-AC6…AC14 |
| FR-17 | Làm bài code: màn ≥ 1024 px, soạn mã, nháp, `Chạy thử`, `Nộp lời giải`, chốt bản tính điểm, hết giờ khi gõ | 06-AC1…AC13 |
| FR-18 | Không rò đáp án / test ẩn ở endpoint sinh viên | 05-AC4, 06-AC1, AC9, AC14, 08-AC18 |
| FR-19 | Khoá chat, `Locker`, sự kiện, quyền xem, không trừ điểm, thông báo cho sinh viên | 07-AC1…AC6, AC10, AC11 |
| FR-20 | So độ giống mã (winnowing), ngưỡng theo lớp, xử lý cặp, không gửi ra ngoài | 07-AC7, AC8, AC9 |
| FR-21 | Đóng bài, chấm, tự công bố đúng một lần, hoãn / nhả | 08-AC1…AC5, AC20 |
| FR-22 | Kết quả sinh viên (lọc trường) và màn kết quả; vắng | 08-AC6, AC7, AC8 |
| FR-23 | Kết quả Staff: API, UI, thống kê, CSV | 08-AC9…AC12 |
| FR-24 | Sửa điểm, `override`, chấm lại, phúc khảo, phân quyền | 08-AC13…AC17, AC19 |
| FR-25 | Seed, k6, E2E, cổng PE, cổng UX, bàn giao | 09-AC1…AC10 |

## 5. Dữ liệu

Migration **`backend-go/db/migrations/00005_weekly_exam.sql`** (goose; nộp sau `00004`; số thật ghi vào `PROGRESS.md` — đề xuất đổi đánh số ở mục 10). Quy ước PG 5: `id uuid DEFAULT uuidv7()`, `timestamptz`, trigger `set_updated_at` cho bảng có `updated_at`, `snake_case`, **không ALTER** sau khi merge. Bảng thuộc lớp có `course_id uuid NOT NULL REFERENCES courses(id)` và chỉ mục phức hợp **bắt đầu bằng `course_id`**. Mọi `created_by` / `student_id` / `*_by` tham chiếu `users(id)`. Điểm: `numeric(5,2)`; trọng số: số nguyên.

### 5.1 Enum

| Enum | Giá trị |
| --- | --- |
| `question_type` | `MCQ_SINGLE`, `MCQ_MULTI`, `TRUE_FALSE`, `CODE`, `SHORT`, `ESSAY` (PE chỉ dùng 4 giá trị đầu) |
| `question_difficulty` | `EASY`, `MEDIUM`, `HARD` |
| `question_origin` | `MANUAL`, `AI_DRAFT`, `EXTRACTED`, `GENERATED` (PE dùng 2 giá trị đầu) |
| `question_review_status` | `DRAFT`, `PENDING`, `APPROVED`, `REJECTED` |
| `checker_kind` | `EXACT`, `TOKENS`, `FLOAT_EPS` |
| `exam_kind` | `MCQ`, `CODE`, `MIXED` |
| `exam_status` | `DRAFT`, `SCHEDULED`, `OPEN`, `CLOSED`, `PUBLISHED` |
| `multi_scoring` | `ALL_OR_NOTHING`, `PARTIAL` |
| `attempt_status` | `IN_PROGRESS`, `GRADING`, `GRADED` |
| `attempt_submit_reason` | `MANUAL`, `TIMEOUT`, `CLOSED` |
| `submission_kind` | `RUN`, `SUBMIT` |
| `submission_status` | `QUEUED`, `RUNNING`, `DONE`, `SUPERSEDED`, `ERROR` |
| `judge_verdict` | `AC`, `WA`, `TLE`, `MLE`, `RE`, `CE`, `OLE`, `IE` |
| `exam_event_type` | `TAB_HIDDEN`, `TAB_VISIBLE`, `PASTE`, `OFFLINE`, `ONLINE`, `TAB_TAKEOVER`, `CHAT_BLOCKED` |
| `similarity_review_state` | `NEW`, `CLEARED`, `FOLLOW_UP` |
| `appeal_status` | `OPEN`, `UPHELD`, `ADJUSTED` |

### 5.2 `question_bank`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK |
| `course_id` | `uuid` | NOT NULL | — | FK `courses` |
| `type` | `question_type` | NOT NULL | — | |
| `title` | `text` | NOT NULL | — | `CHECK (char_length(title) BETWEEN 1 AND 120)` |
| `topic` | `text` | NOT NULL | — | `CHECK (char_length(topic) BETWEEN 1 AND 80)` |
| `difficulty` | `question_difficulty` | NOT NULL | `'MEDIUM'` | |
| `stem` | `text` | NOT NULL | — | `CHECK (char_length(stem) BETWEEN 1 AND 8000)`; Markdown lưu nguyên văn |
| `answer_key` | `jsonb` | NULL | — | `CHECK ((type IN ('MCQ_SINGLE','MCQ_MULTI','TRUE_FALSE')) = (answer_key IS NOT NULL) OR type IN ('SHORT','ESSAY'))`; `type='CODE'` ⇒ NULL |
| `explanation` | `text` | NULL | — | `CHECK (char_length(explanation) <= 4000)` |
| `citations` | `jsonb` | NOT NULL | `'[]'` | P9 dùng; PE để `[]` |
| `origin` | `question_origin` | NOT NULL | `'MANUAL'` | |
| `review_status` | `question_review_status` | NOT NULL | `'DRAFT'` | |
| `created_by` | `uuid` | NOT NULL | — | |
| `reviewed_by` | `uuid` | NULL | — | `CHECK ((review_status IN ('APPROVED','REJECTED')) = (reviewed_by IS NOT NULL))` |
| `reviewed_at` | `timestamptz` | NULL | — | |
| `ai_job_id` | `uuid` | NULL | — | job `question.suggest` đã tạo bản nháp (không FK) |
| `archived_at` | `timestamptz` | NULL | — | |
| `version` | `integer` | NOT NULL | `1` | |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `question_bank_course_review_idx` (course_id, review_status, created_at DESC, id DESC) WHERE archived_at IS NULL · `question_bank_course_topic_idx` (course_id, topic, difficulty, type) WHERE archived_at IS NULL · `question_bank_course_created_idx` (course_id, created_at DESC, id DESC).

### 5.3 `question_options`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK; là id đáp án dùng trong `answer_key` |
| `course_id` | `uuid` | NOT NULL | — | |
| `question_id` | `uuid` | NOT NULL | — | `REFERENCES question_bank(id) ON DELETE CASCADE` |
| `position` | `smallint` | NOT NULL | — | `CHECK (position BETWEEN 1 AND 8)` |
| `body` | `text` | NOT NULL | — | `CHECK (char_length(body) BETWEEN 1 AND 1000)` |
| `pinned_last` | `boolean` | NOT NULL | `false` | giữ cuối khi xáo |

Chỉ mục: PK · `question_options_question_position_key` UNIQUE (question_id, position) · `question_options_course_question_idx` (course_id, question_id, position). **Không có cột đúng / sai.**

### 5.4 `code_problems`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `question_id` | `uuid` | NOT NULL | — | PK; `REFERENCES question_bank(id) ON DELETE CASCADE`; câu phải là `CODE` (service) |
| `course_id` | `uuid` | NOT NULL | — | |
| `languages` | `text[]` | NOT NULL | `'{cpp17}'` | `CHECK (cardinality(languages) >= 1 AND languages <@ ARRAY['c11','cpp17'])` |
| `time_limit_ms` | `integer` | NOT NULL | `1000` | `CHECK (time_limit_ms BETWEEN 100 AND 10000)` |
| `memory_limit_mb` | `integer` | NOT NULL | `256` | `CHECK (memory_limit_mb BETWEEN 16 AND 1024)` |
| `output_limit_kb` | `integer` | NOT NULL | `1024` | `CHECK (output_limit_kb BETWEEN 1 AND 16384)` |
| `checker` | `checker_kind` | NOT NULL | `'EXACT'` | |
| `float_eps` | `numeric(12,10)` | NULL | — | `CHECK ((checker = 'FLOAT_EPS') = (float_eps IS NOT NULL))`, `CHECK (float_eps IS NULL OR (float_eps > 0 AND float_eps <= 0.1))` |
| `starter_code` | `jsonb` | NOT NULL | `'{}'` | `{"c11":"…","cpp17":"…"}`, mỗi giá trị ≤ 16 KiB (service) |
| `reference_language` | `text` | NULL | — | `CHECK ((reference_language IS NULL) = (reference_source IS NULL))` |
| `reference_source` | `text` | NULL | — | ≤ 64 KiB; **không bao giờ** ra khỏi đường Staff |
| `reference_verified_version` | `integer` | NULL | — | = `tests_version` lúc xác minh |
| `reference_verified_at` | `timestamptz` | NULL | — | |
| `tests_version` | `integer` | NOT NULL | `1` | tăng khi đổi test, giới hạn, checker, ngôn ngữ |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `code_problems_course_idx` (course_id).

### 5.5 `code_testcases`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK |
| `course_id` | `uuid` | NOT NULL | — | |
| `problem_id` | `uuid` | NOT NULL | — | `REFERENCES code_problems(question_id) ON DELETE CASCADE` |
| `position` | `integer` | NOT NULL | — | `CHECK (position >= 1)` |
| `name` | `text` | NOT NULL | — | `CHECK (name ~ '^[A-Za-z0-9_-]{1,60}$')` |
| `is_sample` | `boolean` | NOT NULL | `false` | |
| `weight` | `smallint` | NOT NULL | `1` | `CHECK (weight BETWEEN 0 AND 1000)` |
| `input` / `input_blob_key` | `text` / `text` | NULL / NULL | — | `CHECK ((input IS NULL) <> (input_blob_key IS NULL))`; ≤ 64 KiB inline |
| `expected` / `expected_blob_key` | `text` / `text` | NULL / NULL | — | tương tự |
| `input_bytes`, `expected_bytes` | `integer` | NOT NULL | — | `CHECK (input_bytes <= 1048576 AND expected_bytes <= 1048576)` |
| `source` | `text` | NOT NULL | `'MANUAL'` | `CHECK (source IN ('MANUAL','IMPORT','AI_DRAFT'))` |
| `approved` | `boolean` | NOT NULL | `true` | `AI_DRAFT` tạo với `false`; judge chỉ dùng `approved` |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `code_testcases_problem_position_key` UNIQUE (problem_id, position) DEFERRABLE INITIALLY DEFERRED · `code_testcases_course_problem_idx` (course_id, problem_id, position). Tối đa 100 test / bài (service, `TOO_MANY_TESTS`).

### 5.6 `exams`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK |
| `course_id` | `uuid` | NOT NULL | — | |
| `title` | `text` | NOT NULL | — | `CHECK (char_length(title) BETWEEN 1 AND 120)` |
| `instructions` | `text` | NULL | — | `CHECK (char_length(instructions) <= 4000)` |
| `kind` | `exam_kind` | NOT NULL | `'MCQ'` | suy ra từ mục |
| `status` | `exam_status` | NOT NULL | `'DRAFT'` | |
| `opens_at`, `closes_at` | `timestamptz` | NULL | — | `CHECK (status = 'DRAFT' OR (opens_at IS NOT NULL AND closes_at IS NOT NULL AND duration_minutes IS NOT NULL))`; `CHECK (opens_at IS NULL OR closes_at IS NULL OR closes_at > opens_at)` |
| `duration_minutes` | `smallint` | NULL | — | `CHECK (duration_minutes BETWEEN 1 AND 300)`; `CHECK (duration_minutes IS NULL OR opens_at IS NULL OR closes_at IS NULL OR duration_minutes * interval '1 minute' <= closes_at - opens_at)` (bản thấp nhất 5 phút do service) |
| `shuffle_questions`, `shuffle_options` | `boolean` | NOT NULL | `true` | |
| `max_score` | `numeric(5,2)` | NOT NULL | `10.00` | `CHECK (max_score > 0 AND max_score <= 100)` |
| `rounding_step` | `numeric(3,2)` | NOT NULL | `0.01` | `CHECK (rounding_step IN (0.01, 0.10, 0.25, 0.50, 1.00))` |
| `multi_scoring` | `multi_scoring` | NOT NULL | `'ALL_OR_NOTHING'` | |
| `reveal_answers` | `boolean` | NOT NULL | `true` | |
| `appeal_days` | `smallint` | NOT NULL | `7` | `CHECK (appeal_days BETWEEN 0 AND 30)` |
| `publish_hold` | `boolean` | NOT NULL | `false` | |
| `regrading` | `boolean` | NOT NULL | `false` | chặn công bố khi đang chấm lại |
| `published_at` | `timestamptz` | NULL | — | `CHECK ((status = 'PUBLISHED') = (published_at IS NOT NULL))` |
| `created_by` | `uuid` | NOT NULL | — | |
| `version` | `integer` | NOT NULL | `1` | |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `exams_course_status_idx` (course_id, status, opens_at DESC, id DESC) · `exams_course_created_idx` (course_id, created_at DESC, id DESC) · `exams_due_open_idx` (opens_at) WHERE status = 'SCHEDULED' · `exams_due_close_idx` (closes_at) WHERE status IN ('SCHEDULED','OPEN') · `exams_closed_idx` (course_id, id) WHERE status = 'CLOSED'.

### 5.7 `exam_items`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK |
| `course_id` | `uuid` | NOT NULL | — | |
| `exam_id` | `uuid` | NOT NULL | — | `REFERENCES exams(id) ON DELETE CASCADE` |
| `question_id` | `uuid` | NOT NULL | — | `REFERENCES question_bank(id)` |
| `position` | `smallint` | NOT NULL | — | `CHECK (position >= 1)` |
| `points` | `numeric(5,2)` | NOT NULL | `1.00` | `CHECK (points > 0 AND points <= 100)` |
| `override` | `jsonb` | NULL | — | `{"answer_key":{…}}` hoặc `{"void":true}` + `{"reason","by","at"}` |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `exam_items_exam_position_key` UNIQUE (exam_id, position) DEFERRABLE INITIALLY DEFERRED · `exam_items_exam_question_key` UNIQUE (exam_id, question_id) · `exam_items_course_exam_idx` (course_id, exam_id, position) · `exam_items_question_idx` (question_id, exam_id) (tra "đang dùng").

### 5.8 `exam_attempts`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK; hạt giống xáo |
| `course_id`, `exam_id` | `uuid` | NOT NULL | — | |
| `student_id` | `uuid` | NOT NULL | — | từ JWT |
| `status` | `attempt_status` | NOT NULL | `'IN_PROGRESS'` | |
| `started_at` | `timestamptz` | NOT NULL | `now()` | |
| `deadline_at` | `timestamptz` | NOT NULL | — | `CHECK (deadline_at > started_at)` |
| `submitted_at` | `timestamptz` | NULL | — | `CHECK ((status <> 'IN_PROGRESS') = (submitted_at IS NOT NULL))` |
| `submit_reason` | `attempt_submit_reason` | NULL | — | `CHECK ((submitted_at IS NOT NULL) = (submit_reason IS NOT NULL))` |
| `writer_tab` | `uuid` | NULL | — | |
| `writer_seen_at` | `timestamptz` | NULL | — | |
| `auto_score` | `numeric(5,2)` | NULL | — | `CHECK (auto_score IS NULL OR auto_score BETWEEN 0 AND 100)`; `CHECK ((status = 'GRADED') = (auto_score IS NOT NULL AND graded_at IS NOT NULL))` |
| `adjusted_score` | `numeric(5,2)` | NULL | — | `CHECK (adjusted_score IS NULL OR (adjusted_score BETWEEN 0 AND 100 AND adjusted_reason IS NOT NULL))` |
| `adjusted_reason` | `text` | NULL | — | `CHECK (char_length(adjusted_reason) BETWEEN 1 AND 500)` |
| `adjusted_by` | `uuid` | NULL | — | |
| `adjusted_at` | `timestamptz` | NULL | — | |
| `breakdown` | `jsonb` | NULL | — | `[{item_id,earned,max,void?}]` |
| `graded_at` | `timestamptz` | NULL | — | |
| `version` | `integer` | NOT NULL | `1` | |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `exam_attempts_exam_student_key` UNIQUE (exam_id, student_id) · `exam_attempts_course_exam_idx` (course_id, exam_id, status, student_id) · `exam_attempts_running_idx` (student_id, deadline_at) WHERE status = 'IN_PROGRESS' · `exam_attempts_due_idx` (deadline_at) WHERE status = 'IN_PROGRESS' · `exam_attempts_student_idx` (course_id, student_id, started_at DESC).

### 5.9 `exam_answers`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `attempt_id` | `uuid` | NOT NULL | — | `REFERENCES exam_attempts(id) ON DELETE CASCADE` |
| `item_id` | `uuid` | NOT NULL | — | `REFERENCES exam_items(id) ON DELETE CASCADE` |
| `course_id` | `uuid` | NOT NULL | — | |
| `answer` | `jsonb` | NOT NULL | — | `{"option_ids":[…]}` hoặc `{"value":bool}`; `CHECK (octet_length(answer::text) <= 4096)` |
| `saved_at` | `timestamptz` | NOT NULL | `now()` | |

PK `(attempt_id, item_id)`; chỉ mục `exam_answers_course_attempt_idx` (course_id, attempt_id).

### 5.10 `code_drafts`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `attempt_id`, `item_id` | `uuid` | NOT NULL | — | FK như 5.9 |
| `course_id` | `uuid` | NOT NULL | — | |
| `language` | `text` | NOT NULL | — | `CHECK (language IN ('c11','cpp17'))` |
| `source` | `text` | NOT NULL | — | `CHECK (octet_length(source) <= 65536)` |
| `rev` | `integer` | NOT NULL | `1` | tăng mỗi lần ghi |
| `updated_at` | `timestamptz` | NOT NULL | `now()` | |

PK `(attempt_id, item_id, language)`; chỉ mục `code_drafts_course_attempt_idx` (course_id, attempt_id).

### 5.11 `code_submissions`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK; `run_id` / `submission_id` |
| `course_id`, `exam_id`, `attempt_id`, `item_id`, `problem_id`, `student_id` | `uuid` | NOT NULL | — | `problem_id` = `question_id` |
| `kind` | `submission_kind` | NOT NULL | — | |
| `language` | `text` | NOT NULL | — | `CHECK (language IN ('c11','cpp17'))` |
| `source` | `text` | NOT NULL | — | `CHECK (octet_length(source) BETWEEN 1 AND 65536)` |
| `source_sha256` | `char(64)` | NOT NULL | — | |
| `auto` | `boolean` | NOT NULL | `false` | `CHECK (NOT auto OR kind = 'SUBMIT')` |
| `status` | `submission_status` | NOT NULL | `'QUEUED'` | |
| `verdict` | `judge_verdict` | NULL | — | `CHECK ((status IN ('DONE','ERROR')) = (verdict IS NOT NULL))` |
| `tests_version` | `integer` | NULL | — | phiên bản test đã chấm |
| `compile_ok` | `boolean` | NULL | — | |
| `compile_log` | `text` | NULL | — | `CHECK (char_length(compile_log) <= 8192)` |
| `results` | `jsonb` | NULL | — | `[{test_id,position,is_sample,verdict,time_ms,memory_kb,weight,stdout_excerpt?}]` |
| `passed_weight`, `total_weight` | `integer` | NULL | — | `CHECK (passed_weight <= total_weight)` |
| `time_ms_max`, `memory_kb_max` | `integer` | NULL | — | |
| `attempts` | `smallint` | NOT NULL | `0` | số lần consumer đã chạy |
| `judged_at` | `timestamptz` | NULL | — | |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `code_submissions_attempt_item_idx` (course_id, attempt_id, item_id, created_at DESC, id DESC) WHERE kind = 'SUBMIT' · `code_submissions_runs_idx` (attempt_id, created_at DESC) WHERE kind = 'RUN' · `code_submissions_queue_idx` (status, created_at) WHERE status IN ('QUEUED','RUNNING') · `code_submissions_exam_problem_idx` (course_id, exam_id, problem_id) WHERE kind = 'SUBMIT' (so độ giống, chấm lại). Bản `RUN` xoá sau 30 ngày kể từ `PUBLISHED` (job dọn — nợ PR).

### 5.12 `exam_events`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK |
| `course_id`, `exam_id` | `uuid` | NOT NULL | — | |
| `attempt_id` | `uuid` | NOT NULL | — | `REFERENCES exam_attempts(id) ON DELETE CASCADE` |
| `student_id` | `uuid` | NOT NULL | — | |
| `type` | `exam_event_type` | NOT NULL | — | |
| `occurred_at` | `timestamptz` | NOT NULL | `now()` | giờ máy chủ |
| `client_at` | `timestamptz` | NULL | — | tham khảo |
| `meta` | `jsonb` | NOT NULL | `'{}'` | `CHECK (octet_length(meta::text) <= 300)`; khoá `duration_ms`, `chars`, `item_id` |

Chỉ mục: PK · `exam_events_attempt_idx` (course_id, exam_id, attempt_id, occurred_at, id).

### 5.13 `similarity_reports` (một dòng một cặp)

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK |
| `course_id`, `exam_id`, `problem_id` | `uuid` | NOT NULL | — | |
| `run_id` | `uuid` | NOT NULL | — | một lần chạy |
| `submission_a`, `submission_b` | `uuid` | NOT NULL | — | `REFERENCES code_submissions(id) ON DELETE CASCADE` |
| `attempt_a`, `attempt_b` | `uuid` | NOT NULL | — | `CHECK (attempt_a < attempt_b)` |
| `score` | `numeric(4,3)` | NOT NULL | — | `CHECK (score BETWEEN 0 AND 1)` |
| `shared_fingerprints` | `integer` | NOT NULL | — | |
| `flagged` | `boolean` | NOT NULL | `false` | |
| `algorithm` | `text` | NOT NULL | `'winnow-k5-w4-jaccard'` | |
| `review_state` | `similarity_review_state` | NOT NULL | `'NEW'` | |
| `reviewed_by` | `uuid` | NULL | — | `CHECK ((review_state = 'NEW') = (reviewed_by IS NULL))` |
| `reviewed_at` | `timestamptz` | NULL | — | |
| `note` | `text` | NULL | — | `CHECK (char_length(note) <= 500)` |
| `created_at` | `timestamptz` | NOT NULL | `now()` | |

Chỉ mục: PK · `similarity_reports_run_pair_key` UNIQUE (run_id, submission_a, submission_b) · `similarity_reports_exam_idx` (course_id, exam_id, problem_id, score DESC, id) · `similarity_reports_run_idx` (run_id).

### 5.14 `exam_appeals`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK |
| `course_id`, `exam_id` | `uuid` | NOT NULL | — | |
| `attempt_id` | `uuid` | NOT NULL | — | `REFERENCES exam_attempts(id) ON DELETE CASCADE`; UNIQUE |
| `student_id` | `uuid` | NOT NULL | — | |
| `reason` | `text` | NOT NULL | — | `CHECK (char_length(reason) BETWEEN 1 AND 1000)` |
| `status` | `appeal_status` | NOT NULL | `'OPEN'` | |
| `response` | `text` | NULL | — | `CHECK (char_length(response) BETWEEN 1 AND 1000)` |
| `responded_by` | `uuid` | NULL | — | |
| `responded_at` | `timestamptz` | NULL | — | `CHECK ((status = 'OPEN') = (responded_at IS NULL))` |
| `score_before`, `score_after` | `numeric(5,2)` | NULL | — | `CHECK ((status = 'ADJUSTED') = (score_after IS NOT NULL))` |
| `version` | `integer` | NOT NULL | `1` | |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `exam_appeals_attempt_key` UNIQUE (attempt_id) · `exam_appeals_course_exam_idx` (course_id, exam_id, status, created_at) · `exam_appeals_open_idx` (course_id, created_at) WHERE status = 'OPEN'.

### 5.15 Khoá Redis và Stream (tiền tố `ep:` theo PG 5.6; Stream không tiền tố)

| Khoá | Kiểu | TTL | Dùng cho |
| --- | --- | --- | --- |
| `ep:exam_lock:{user_id}` | String (`attempt_id`) | = thời gian còn lại + `EXAM_GRACE_SECONDS` | khoá chat (4.7.1) |
| `ep:exam:run:{user_id}` | ZSET (member = `run_id`, score = ms) | 11 phút | cửa sổ trượt `Chạy thử` (10 / 10 phút) |
| `ep:exam:submitcd:{attempt_id}:{item_id}` | String `SET NX` | `EXAM_SUBMIT_COOLDOWN` (15 s) | giãn cách nộp |
| `ep:rl:exam:{attempt_id}:{phút}` | String (INCR) | 120 s | giới hạn lưu `EXAM_SAVE_RATE_PER_MIN` |
| `ep:exam:tick:leader` | String `SET NX PX` | 15 s | một bộ lập lịch |
| `ep:today:*` | (có sẵn) | 60 s | bị xoá theo sự kiện (4.10) |
| `judge.submit`, `judge.run` | Stream (nhóm `judge`) | — | hàng chấm |
| `judge.submit.dead`, `judge.run.dead` | Stream (`MAXLEN ~ 10000`) | — | dead-letter |

Không cache điểm / kết quả ở phía máy chủ (luật 15).

## 6. API

Tiền tố `/api/v1`; JSON (riêng `testcases/import`: `multipart/form-data`; `results.csv`: `text/csv`); `message` tiếng Việt; lỗi PG `{code,message,trace_id,details?,retry_after?}`; danh sách phân trang con trỏ `{items,next_cursor}` (30 mặc định, ≤ 100). Trong bảng, `…` = `/courses/{courseId}`; `…/attempts/{aid}` = `…/exams/{eid}/attempts/{aid}`. Thời gian RFC 3339 UTC; điểm là chuỗi thập phân.

### 6.1 Mã lỗi mới (12 mã; tổng sau PE = 37 + 12 = 49 nếu các phase giữa không thêm)

| Status | `code` | Khi nào | `details` |
| --- | --- | --- | --- |
| 409 | `EXAM_NOT_OPEN` | bắt đầu / làm khi bài chưa mở, đã đóng, hoặc còn dưới 60 s | `{reason: not_scheduled\|not_yet\|closed\|closing, opens_at?, closes_at?}` |
| 409 | `ATTEMPT_ALREADY_SUBMITTED` | lượt đã nộp mà còn bắt đầu / lưu / nộp | `{submitted_at}` |
| 409 | `ATTEMPT_CLOSED` | lưu / nộp sau `deadline_at + EXAM_GRACE_SECONDS` | `{deadline_at}` |
| 409 | `ATTEMPT_OTHER_TAB` | tab khác đang là người ghi | `{writer_seen_at}` |
| 409 | `DRAFT_CONFLICT` | `base_rev` ≠ `rev` của bản nháp mã | `{current_rev, updated_at}` |
| 409 | `EXAM_LOCKED` | sửa trường / thao tác bị khoá theo trạng thái (4.2.5) hoặc đã có lượt làm | `{reason: status\|has_attempts\|opened}` |
| 409 | `QUESTION_IN_USE` | sửa câu đang được bài thi `SCHEDULED+` dùng | `{exams:[{id,title}]}` |
| 409 | `RESULT_NOT_PUBLISHED` | xem kết quả trước công bố | — (thân không chứa dữ liệu) |
| 409 | `SUBMISSION_LIMIT_REACHED` | quá `EXAM_SUBMISSION_CAP` lần nộp cho một bài | `{cap}` |
| 503 | `JUDGE_UNAVAILABLE` | `Chạy thử` khi sandbox không phản hồi (kèm `Retry-After`, `retry_after`) | — |
| 409 | `APPEAL_WINDOW_CLOSED` | phúc khảo hết hạn / `appeal_days = 0` | `{closed_at}` |
| 409 | `APPEAL_EXISTS` | lượt đã có yêu cầu phúc khảo | — |

Dùng lại: `FORBIDDEN` (`reason` ∈ `role`, `course`), `NOT_FOUND`, `VALIDATION_FAILED` với `details[].code` ∈ `OPTION_COUNT`, `NO_CORRECT_OPTION`, `SINGLE_MULTIPLE_CORRECT`, `DUPLICATE_OPTION`, `STEM_TOO_LONG`, `TYPE_NOT_SUPPORTED`, `LANGUAGE_NOT_ALLOWED`, `LIMIT_OUT_OF_RANGE`, `FLOAT_EPS_REQUIRED`, `TOO_MANY_TESTS`, `TEST_ZIP_INVALID`, `REFERENCE_REQUIRED`, `REFERENCE_NOT_VERIFIED`, `CODE_TESTS_MISSING`, `TOTAL_WEIGHT_ZERO`, `ITEM_NOT_APPROVED`, `QUESTION_NOT_IN_COURSE`, `DUPLICATE_ITEM`, `NO_ITEMS`, `OPENS_IN_PAST`, `CLOSES_BEFORE_OPENS`, `CLOSES_NOT_LATER`, `DURATION_EXCEEDS_WINDOW`, `DURATION_TOO_SHORT`, `INVALID_OPTION_ID`, `SOURCE_TOO_LARGE`, `SOURCE_EMPTY`; `CONFLICT`, `VERSION_CONFLICT`, `RATE_LIMITED` (+ `retry_after`), `COURSE_ARCHIVED`, `IDEMPOTENCY_KEY_REQUIRED`, `IDEMPOTENCY_KEY_REUSED`, `PAYLOAD_TOO_LARGE`, `INVALID_CURSOR`, `SERVICE_UNAVAILABLE`.

### 6.2 Bảng thao tác (56 thao tác + 1 route thử)

| # | Thao tác | Quyền (mục 2) | `Idempotency-Key` | Thành công |
| --- | --- | --- | --- | --- |
| 1 | `GET …/questions?review_status&topic&difficulty&type&origin&q&archived&cursor&limit` | Staff | — | 200 cursor |
| 2 | `POST …/questions` | Staff | tuỳ chọn | 201 |
| 3 | `GET …/questions/{qid}` | Staff | — | 200 |
| 4 | `PUT …/questions/{qid}` (`version`) | Staff | — | 200 |
| 5 | `POST …/questions/{qid}/archive` | Staff | — | 200 |
| 6 | `POST …/questions/{qid}/duplicate` | Staff | — | 201 |
| 7 | `PUT …/questions/{qid}/review {decision: REQUEST\|APPROVE\|REJECT, version}` | Staff | — | 200 |
| 8 | `PUT …/questions/{qid}/code` (`version`) | Staff | — | 200 |
| 9 | `GET …/questions/{qid}/testcases` | Staff | — | 200 (cả test ẩn) |
| 10 | `POST …/questions/{qid}/testcases` | Staff | — | 201 |
| 11 | `PUT …/questions/{qid}/testcases/{tid}` | Staff (sau đóng: Teacher) | — | 200 |
| 12 | `DELETE …/questions/{qid}/testcases/{tid}` | Staff (sau đóng: Teacher) | — | 204 |
| 13 | `POST …/questions/{qid}/testcases/import?dry_run&mode=append\|replace` | Staff (sau đóng: Teacher) | — | 200 / 201 |
| 14 | `POST …/questions/{qid}/testcases/approve {ids[]}` | Staff | — | 200 |
| 15 | `POST …/questions/{qid}/reference/verify` | Staff | **bắt buộc** | 202 `{job_id}` |
| 16 | `POST …/questions/suggest {kind,…}` | Staff | **bắt buộc** | 202 `{job_id}` |
| 17 | `GET …/exams?status&cursor&limit` | Member (bản theo vai) | — | 200 cursor, `ETag` |
| 18 | `POST …/exams` | Staff | **bắt buộc** | 201 |
| 19 | `GET …/exams/{eid}` | Member (bản theo vai) | — | 200 |
| 20 | `PUT …/exams/{eid}` (`version`) | Staff | — | 200 |
| 21 | `PUT …/exams/{eid}/items {items[],version}` | Staff | — | 200 |
| 22 | `GET …/exams/{eid}/preview` | Staff | — | 200 |
| 23 | `POST …/exams/{eid}/schedule` | Teacher | tuỳ chọn | 200 |
| 24 | `POST …/exams/{eid}/unschedule` | Teacher | — | 200 |
| 25 | `POST …/exams/{eid}/extend {closes_at}` | Teacher | — | 200 |
| 26 | `PUT …/exams/{eid}/publish-hold {hold,version}` | Teacher | — | 200 |
| 27 | `DELETE …/exams/{eid}` | Teacher | — | 204 |
| 28 | `POST …/exams/{eid}/clone` | Staff | — | 201 |
| 29 | `POST …/exams/{eid}/attempts` | Member + STUDENT | **bắt buộc** | 201 mới / 200 làm tiếp |
| 30 | `GET …/exams/{eid}/attempts/mine` | Member + STUDENT | — | 200 (theo trạng thái) |
| 31 | `PUT …/attempts/{aid}/answers` (`X-Exam-Tab`) | Member + STUDENT | — | 200 |
| 32 | `PUT …/attempts/{aid}/code/{itemId}/draft` (`X-Exam-Tab`) | Member + STUDENT | — | 200 |
| 33 | `POST …/attempts/{aid}/code/{itemId}/run` | Member + STUDENT | **bắt buộc** | 202 `{run_id}` |
| 34 | `GET …/attempts/{aid}/runs/{runId}` | Member + STUDENT | — | 200 |
| 35 | `POST …/attempts/{aid}/code/{itemId}/submit` | Member + STUDENT | **bắt buộc** | 202 `{submission_id}` |
| 36 | `GET …/attempts/{aid}/code/{itemId}/submissions?cursor` | Member + STUDENT | — | 200 cursor |
| 37 | `GET …/attempts/{aid}/submissions/{sid}` | Member + STUDENT | — | 200 |
| 38 | `POST …/attempts/{aid}/events` | Member + STUDENT | — | 204 |
| 39 | `POST …/attempts/{aid}/takeover` | Member + STUDENT | — | 200 |
| 40 | `POST …/attempts/{aid}/submit` | Member + STUDENT | **bắt buộc** | 200 |
| 41 | `GET …/attempts/{aid}/result` | Member + STUDENT | — | 200 (chỉ `PUBLISHED`) |
| 42 | `POST …/attempts/{aid}/appeal {reason}` | Member + STUDENT | **bắt buộc** | 201 |
| 43 | `GET /me/exam-lock` | JWT | — | 200 |
| 44 | `GET …/exams/{eid}/results?status&sort&q&cursor&limit` | Staff | — | 200 cursor, `ETag` |
| 45 | `GET …/exams/{eid}/results/{aid}` | Staff | — | 200 |
| 46 | `PUT …/exams/{eid}/results/{aid}/score {score\|null,reason,version}` | Teacher | — | 200 |
| 47 | `PUT …/exams/{eid}/items/{itemId}/override {answer_key\|void,reason}` | Teacher | — | 200 / 202 |
| 48 | `POST …/exams/{eid}/regrade {scope,item_id?,attempt_id?,reason}` | Teacher | **bắt buộc** | 202 `{job_id}` |
| 49 | `GET …/exams/{eid}/stats` | Staff | — | 200 |
| 50 | `GET …/exams/{eid}/results.csv` | Staff | — | 200 `text/csv` |
| 51 | `GET …/exams/{eid}/events?attempt&cursor` | Teacher | — | 200 cursor |
| 52 | `GET …/exams/{eid}/similarity?run&flagged` | Teacher | — | 200 cursor |
| 53 | `POST …/exams/{eid}/similarity/run` | Teacher | tuỳ chọn | 202 `{job_id}` |
| 54 | `PUT …/exams/{eid}/similarity/{id}/review {state,note?}` | Teacher | — | 200 |
| 55 | `GET …/exams/{eid}/appeals?status&cursor` | Staff | — | 200 cursor |
| 56 | `POST …/exams/{eid}/appeals/{id}/answer {decision,response,score?,version}` | Teacher | tuỳ chọn | 200 |
| T1 | `GET /_test/chat-gate` (chỉ `testroutes`) | JWT | — | 200 `{allowed}` |

Mọi thao tác trên (trừ #43, T1) đi qua `CourseAccessGuard` ở chế độ nêu; ADMIN → 403. Thao tác nhận `Idempotency-Key` theo PG 6.6 (Redis 24 h; thiếu khi bắt buộc → 422 `IDEMPOTENCY_KEY_REQUIRED`).

### 6.3 Thân chính (rút gọn)

**Câu trắc nghiệm** (`POST …/questions`): `{"type":"MCQ_MULTI","title":"…","topic":"Mật mã đối xứng","difficulty":"MEDIUM","stem":"…","options":[{"body":"AES","pinned_last":false},{"body":"RSA"},{"body":"3DES"}],"correct":[0,2],"explanation":"…"}` → 201 `{id,…,review_status:"DRAFT",version:1,options:[{id,position,body}]}` (`correct` là **chỉ số** trong `options` lúc gửi; trả về cho Staff kèm `answer_key`).

**Bài thi** (`POST …/exams`): `{"title":"Kiểm tra tuần 9","opens_at":"…","closes_at":"…","duration_minutes":45,"shuffle_questions":true,"shuffle_options":true,"max_score":"10.00","rounding_step":"0.25","multi_scoring":"PARTIAL","reveal_answers":true,"appeal_days":7}`.

**Bắt đầu / làm tiếp** (`POST …/attempts`, `GET …/attempts/mine` khi `IN_PROGRESS`) — xem danh sách trường ở 6.4:

```json
{"attempt":{"id":"…","exam_id":"…","status":"IN_PROGRESS","started_at":"…","deadline_at":"…","server_time":"…","writer":{"is_you":true}},
 "exam":{"id":"…","title":"…","instructions":"…","kind":"MIXED","duration_minutes":45,"closes_at":"…"},
 "items":[
  {"item_id":"…","position":1,"type":"MCQ_SINGLE","points":"1.00","stem":"…","options":[{"id":"…","body":"…"}],"answer":{"option_ids":["…"]}},
  {"item_id":"…","position":2,"type":"TRUE_FALSE","points":"1.00","stem":"…","answer":{"value":true}},
  {"item_id":"…","position":3,"type":"CODE","points":"3.00","stem":"…",
   "code":{"languages":["c11","cpp17"],"time_limit_ms":1000,"memory_limit_mb":256,
           "starter_code":{"cpp17":"…"},"samples":[{"name":"sample1","input":"1 2\n","expected":"3\n"}]},
   "draft":{"language":"cpp17","source":"…","rev":4},"submissions":{"count":2,"final_id":"…"}}]}
```

**Lưu** (`PUT …/answers`): `{"items":[{"item_id":"…","answer":{"option_ids":["…"]}}]}` → `{"saved_at":"…","server_time":"…","deadline_at":"…"}`. **Nháp mã**: `{"language":"cpp17","source":"…","base_rev":4}` → `{"rev":5,"saved_at":"…"}`.

**Chạy thử** → `{"run_id":"…"}`; `GET …/runs/{id}` → `{"status":"DONE","compile_ok":true,"samples":[{"name":"sample1","verdict":"AC","time_ms":3,"memory_kb":1200},{"name":"sample2","verdict":"WA","time_ms":2,"memory_kb":1180,"input":"…","expected":"…","stdout_excerpt":"…"}]}` (`input` / `expected` của test mẫu; `compile_log` khi `CE`).

**Nộp lời giải** → `{"submission_id":"…"}`; `GET …/submissions/{sid}` (trong giờ) → `{"id","status","language","created_at","is_final","compile_ok","compile_log?","samples":[{"name","verdict","time_ms","memory_kb"}]}`.

**Kết quả của sinh viên**: xem 4.8.3. **Kết quả Staff** (`GET …/results`): `{"progress":{"not_started":3,"in_progress":22,"grading":0,"graded":5,"absent":0},"items":[{"attempt_id","student":{"id","full_name","student_code"},"status","auto_score","score","adjusted":false,"submitted_at","submit_reason","flags":{"similarity":1,"tab_hidden":4,"paste":2}}],"next_cursor":null}` (`flags` chỉ với Giảng viên).

### 6.4 Danh sách trường cho phép ở phía sinh viên và cách giữ

DTO của sinh viên là **struct riêng** trong `internal/exam/dto_student.go` (không tái dùng struct của Staff, không `json:",omitempty"` che trường nhạy cảm — trường nhạy cảm **không tồn tại** trong struct). Tệp `internal/exam/testdata/student_dto_allowlist.json` liệt kê khoá JSON cho phép của từng DTO (`ExamStudentView`, `AttemptView`, `ItemView`, `CodeItemView`, `SubmissionView`, `RunView`, `ResultView`, `ResultItemView`, `ExamLockView`); `TestStudentDTOAllowlist` dùng `reflect` so thẻ `json` với tệp — thêm trường mới bắt buộc sửa tệp (nên PR phải qua mắt người rà). `ItemView` **không** có: `answer_key`, `correct`, `is_correct`, `explanation`, `override`, `original_position`, `pinned_last`. `CodeItemView` **không** có: `hidden*`, `weight`, `tests_version`, `checker`, `reference*`. `SubmissionView` trong giờ **không** có: `results`, `passed_weight`, `total_weight`, `verdict`, `tests_version`. `ResultItemView` chỉ có `answer` / `explanation` khi `reveal_answers`. Không DTO nào có `events`, `similarity*`, `flags`, dữ liệu của sinh viên khác.

`TestNoAnswerLeak` (US-PE-08 AC18) là lớp kiểm thứ hai: seed canary (chuỗi duy nhất) vào `answer_key`, `explanation`, tên / input / expected của test ẩn, lời giải mẫu, bài nộp và đáp án của sinh viên khác, `override`, sự kiện, độ giống; gọi mọi endpoint sinh viên (kể cả đường lỗi, SSE) ở 6 tình huống (bài `SCHEDULED`; `OPEN` chưa bắt đầu; `OPEN` đang làm; `OPEN` đã nộp; `CLOSED` chưa công bố; `PUBLISHED` với `reveal_answers=true` / `false`); quét **từng byte** thân + header + `details` + khung SSE; ngoại lệ được khai báo cụ thể (đáp án và giải thích MCQ khi `reveal_answers`, test mẫu, mã của chính mình); nhật ký ứng dụng cũng quét.

### 6.5 Route thử

`GET /api/v1/_test/chat-gate` (build `testroutes`): `{"allowed":false}` khi người gọi có `exam.Locker` trả `true`, ngược lại `{"allowed":true}`; mô tả trong `api/openapi.test.yaml`; binary mặc định trả 404.

### 6.6 `openapi.yaml` và contract

Thêm đủ 56 thao tác (không thao tác thử), schema cho mọi DTO, 12 mã lỗi, `ETag` ở #17, #44; `TestExamContract` (kin-openapi — D52) kiểm phản hồi thật khớp schema ở trạng thái bài thi đại diện; `TestExamErrorCodes` kiểm mọi mã ở 6.1 xuất hiện đúng. Số dòng "56 thao tác" kiểm bằng `chi.Walk` so với `openapi.yaml`.

## 7. Giao diện

Mọi màn dùng nền chung `frontend/src/shared/` (TanStack Query + `apiClient`, `<PageState>`, `DataTable`, `useAutosaveDraft`, `useSSE`, `<ConfirmIrreversible>`, `Drawer`, `Dialog`, `Tabs`, `SegmentedControl`, `InlineNotice`, `StatusText`), token `--ep-*` (D53), CSS Modules; **cấm** `fetch` trần, spinner / confirm / bảng riêng; `bash scripts/ui-antipatterns.sh` sạch. Giao diện tiếng Việt; "bạn" với sinh viên, "thầy/cô" với giảng viên. Tính năng mới đặt ở `frontend/src/features/exam/` và `features/questions/`.

### 7.1 Route

| Route | Vai | Khung nhìn đầu | Primitive | Trạng thái | Mobile |
| --- | --- | --- | --- | --- | --- |
| `/questions` | TA, GV | Bảng câu hỏi có bộ lọc; nút chính `Tạo câu hỏi`; hàng mở Drawer rộng `QuestionReview` (`DESIGN.md` §14.17) | `DataTable`, `Drawer`, `Tabs`, `Field`, `Menu`, `StatusText` | tải = khung xương; rỗng = "Chưa có câu hỏi nào. Tạo câu đầu tiên hoặc nhờ AI gợi ý nháp." + `Tạo câu hỏi`; lỗi = vấn đề + dữ liệu an toàn + `Thử lại` | < 720 px: danh sách → chi tiết |
| `/exams` (Staff) | TA, GV | Danh sách hàng nhóm `Đang mở` / `Sắp tới` / `Đã đóng` / `Nháp`; nút chính `Tạo bài thi` | `ActionList`, `StatusText`, `PageHeader` | rỗng = "Chưa có bài thi nào. Tạo bài thi đầu tiên từ ngân hàng câu hỏi." | danh sách dọc |
| `/exams` (Sinh viên — Q25) | SV | Bài `Đang mở` / `Sắp tới` / `Đã có điểm`, mỗi hàng một hành động | `ActionList` | rỗng = "Lớp của bạn chưa có bài thi nào." | dùng tốt ở 375 px |
| `/exams/[id]` | TA, GV | Soạn: `Thông tin` · `Câu hỏi` · `Xem trước`; hành động chính theo trạng thái | `Tabs`, `Field`, `ActionList`, `InlineNotice` | lỗi lên lịch = danh sách việc cần sửa có liên kết | ≥ 720 px |
| `/exams/[id]/results` | TA, GV | Bảng kết quả + thanh tiến độ; tab `Phân bố`, `Câu sai nhiều`, `Phúc khảo`, `Nghi giống nhau` (chỉ GV); Drawer chi tiết lượt | `DataTable`, `Drawer`, `Tabs`, biểu đồ cột (recharts) | rỗng = "Chưa có sinh viên nào bắt đầu làm bài." | cuộn ngang có chủ đích, cột tên cố định |
| `/exams/[id]/take` | SV | theo trạng thái: **trước giờ** màn giới thiệu + `Bắt đầu làm bài`; **đang làm** màn làm bài; **đã nộp** màn xác nhận; **đã công bố** kết quả | `Field` (radio / checkbox), `ConfirmIrreversible`, `InlineNotice`, `StatusText` | tải / lỗi / vắng ("Bạn không làm bài này") | trắc nghiệm 375 px; code ≥ 1.024 px |

### 7.2 Chuỗi chính (tiếng Việt)

| Ngữ cảnh | Chuỗi |
| --- | --- |
| Giới thiệu bài | "{tiêu đề}" · "Bạn có {n} phút. Đồng hồ chạy ngay khi bạn bấm Bắt đầu và không dừng lại nếu bạn thoát." · nếu còn ít hơn thời lượng: "Bài thi đóng lúc {HH:mm}, bạn chỉ còn {m} phút." |
| Liêm chính (bắt buộc) | "Trong giờ làm bài, chat AI tạm khoá. Hệ thống ghi lại số lần bạn rời trang hoặc dán nội dung để giảng viên xem khi cần; đây không phải giám thị và không tự trừ điểm của bạn." |
| Nút | `Bắt đầu làm bài` · `Tiếp tục làm bài` · `Câu trước` · `Câu sau` · `Danh sách câu` · `Chạy thử` · `Nộp lời giải` · `Nộp bài` · `Làm tiếp ở đây` · `Gửi yêu cầu xem lại` |
| Lưu | "Đã lưu lúc {hh:mm:ss}" · "Chưa lưu lên máy chủ ({n} thay đổi) — đang thử lại" · "Mất mạng — bài vẫn được giữ trên máy bạn" · (khi offline) "Nếu hết giờ khi chưa có mạng, chỉ phần đã lưu lên máy chủ được tính." |
| Cảnh báo giờ | "Còn 5 phút." · "Còn 1 phút." (dải bình tĩnh, không chớp, không âm thanh) |
| Xác nhận nộp | "Nộp bài thi?" · "Bạn đã trả lời {a}/{n} câu. Còn {u} câu chưa trả lời. Sau khi nộp bạn không sửa được." (+ "{c} bài lập trình chưa có lần nộp nào.") |
| Sau nộp | "Đã nộp lúc {hh:mm:ss}. Điểm sẽ hiện khi bài thi đóng với cả lớp ({Thứ…, HH:mm dd/MM})." · hết giờ: "Hết giờ — bài của bạn đã được nộp lúc {hh:mm:ss}." · đã đóng chưa công bố: "Điểm đang được chấm." |
| Hai tab | "Bài đang mở ở nơi khác. [Làm tiếp ở đây]" · "Có thay đổi chưa lưu — chép ra nếu cần." |
| Màn hẹp | "Bài lập trình cần màn hình rộng hơn (từ 1.024 px). Hãy mở bài thi này trên máy tính — bài của bạn vẫn là một lượt duy nhất và tự đồng bộ." |
| Nhãn kết quả code | `AC` "Đúng" · `WA` "Sai kết quả" · `TLE` "Quá thời gian" · `MLE` "Quá bộ nhớ" · `RE` "Lỗi khi chạy" · `OLE` "In ra quá nhiều" · `CE` "Lỗi biên dịch" · `IE` "Hệ thống chưa chạy được" |
| Hệ thống chưa chấm | "Hệ thống chưa chấm được bài của bạn. Bài đã được lưu và giảng viên sẽ xử lý." · Chạy thử: "Chưa chạy thử được, thử lại sau {n} giây." |
| Giới hạn | "Chờ {n} giây để nộp tiếp." · "Bạn đã chạy thử 10 lần. Thử lại sau {mm:ss}." · "Bạn đã dùng hết 30 lần nộp cho bài này." |
| Kết quả | "{7,75} / {10}" · "Bạn đúng {a}/{n} câu trắc nghiệm · bài lập trình: {p}/{t} test ẩn đạt" · "Test ẩn: đạt {p} trên {t}" · "Câu này đã được điều chỉnh bởi giảng viên" · "Điểm đã được điều chỉnh" |
| Vắng | "Bạn không làm bài này." |
| Phúc khảo | "Đã hết hạn gửi yêu cầu xem lại" · "Bạn đã gửi yêu cầu cho bài này" · phản hồi của giảng viên hiển thị nguyên văn kèm "Giảng viên đã trả lời lúc …" |
| Giảng viên | "Độ giống chỉ là gợi ý — nhiều bài đúng cùng một cách làm tự nhiên giống nhau. Quyết định là của thầy/cô." · "Đã chấm theo bộ test v{a}; hiện là v{b}. [Chấm lại]" · "Chỉ giảng viên lên lịch được" (cho TA) |

Sinh viên **không bao giờ** thấy: `sandbox`, `judge`, `verdict`, `go-judge`, `RAG`, `PII`, `provider`, `trace`, mã trạng thái trần, `tests_version`; nhãn nguồn `AI` của câu hỏi (Q12). Đề xuất bổ sung bảng dịch khái niệm của `DESIGN.md` §13: `Verdict` → (SV) "Kết quả chấm" / (GV) "Kết quả chấm"; `Sandbox` → — (ẩn).

### 7.3 Hành vi chính

- **Làm bài** (`/exams/[id]/take`): khung nhìn đầu render sẵn từ máy chủ (không chờ vòng fetch); tiêu đề, đồng hồ cố định bề rộng chữ số, dòng trạng thái lưu; một câu mỗi lần ở < 720 px, ≥ 1100 px có danh sách câu bên cạnh; điều hướng bàn phím (`←` `→` đổi câu; `1`…`8` / `A`…`H` chọn đáp án); cảnh báo `aria-live="polite"` ở mốc 5 phút / 1 phút; `beforeunload` khi còn thay đổi chưa lên máy chủ; xác nhận nộp bằng `ConfirmIrreversible` có con số.
- **Code** (≥ 1.024 px): hai cột — trái đề + test mẫu + lần nộp; phải ô soạn mã + `Chạy thử` / `Nộp lời giải` + kết quả; ô soạn mã là `textarea` có cột số dòng đồng bộ cuộn; `Tab` chèn 4 dấu cách, `Esc` rồi `Tab` rời ô; đếm byte `x/65.536`.
- **Kết quả Staff**: `DataTable` ảo hoá (1.000 dòng, 60 fps), cột tên cố định, Drawer chi tiết giữ vị trí cuộn; tab liêm chính chỉ hiện với Giảng viên.
- **Tải / rỗng / lỗi** theo `<PageState>`; thao tác đảo ngược được (lên lịch, hoãn công bố) dùng dòng "Đã … · Hoàn tác" thay vì hộp thoại; hộp thoại xác nhận (`ConfirmIrreversible`, có con số) **chỉ** cho việc không đảo ngược: nộp bài thi, nhả hoãn công bố, xoá bài thi nháp ("Bài thi nháp có {n} câu sẽ bị xoá.").

### 7.4 Thành phần mới ở `shared/` (đề xuất bổ sung `DESIGN.md` §10 / §19 — PM quyết)

`shared/ui/CodeEditor` (textarea + số dòng + Tab), `shared/domain/ExamTimer` (đồng hồ máy chủ + offset + cảnh báo), `shared/domain/QuestionNavigator` (danh sách câu + trạng thái), `shared/domain/TestResultList` (bảng verdict từng test mẫu + diff khoảng trắng), `shared/domain/QuestionReview` (đã có tên ở `DESIGN.md` §14.17), `shared/lib/markdown.ts` (bộ render Markdown tối thiểu, an toàn — Q21). Không tạo bản sao nút / trường / bảng / hộp thoại ở route.

### 7.5 Điều hướng và khác biệt mock ↔ thật (D51: spec thật thắng)

- `/questions` hết là mock (P9 trong `FEAT-ui-foundation` 7.6 → **PE**); `/exams`, `/exams/[id]`, `/exams/[id]/results`, `/exams/[id]/take` là route mới.
- `nav.ts`: nhóm "Đánh giá" của TA / GV thêm `/exams` "Bài thi" (TA 12 → **13**, GV 15 → **16**); Sinh viên thêm `/exams` "Bài thi" (7 → **8**, dưới "Thêm" ở thanh dưới điện thoại; Q25); `ACCESS`: `/questions` → TA, GV; `/exams` → TA, GV, SV (SV chỉ danh sách và `/exams/[id]/take`; `/exams/[id]`, `/exams/[id]/results` → TA, GV); Admin: không. Các hằng số đếm mục của `FEAT-ui-foundation` 7.5 (7 / 12 / 15 / 6) và test của US-PU-04 AC3 phải cập nhật **cùng commit** với PE (đề xuất đổi — mục 10).
- `/practice` vẫn là mock cho tới P9.

## 8. Phi chức năng

### 8.1 Biến môi trường (bổ sung vào PG 8.1; sai → thoát 1 nêu tên biến)

| Biến | Mặc định | Dùng cho | Thành phần |
| --- | --- | --- | --- |
| `JUDGE_URL` | `http://judge:5050` | địa chỉ go-judge (mạng `judge_net`) | worker |
| `JUDGE_TOKEN` | (bắt buộc, ≥ 16 ký tự, secret) | `-auth-token` của go-judge | worker, judge |
| `JUDGE_PARALLELISM` | `2` (VPS) / `4` (colima 4 CPU, `.env.local`) | semaphore + `-parallelism` | worker, judge |
| `JUDGE_WORKERS` | `4` | số goroutine consumer (≥ 1 chỉ đọc `judge.run`) | worker |
| `JUDGE_EXTRA_ARGS` | trống (amd64) / `-no-seccomp` (colima arm64) | cờ thêm cho go-judge | judge (compose) |
| `JUDGE_CPUS`, `JUDGE_MEM_LIMIT` | `2.0`, `2g` | giới hạn container judge | compose |
| `JUDGE_HTTP_SLACK` | `5s` | hạn gọi = `clockLimit` + slack | worker |
| `JUDGE_CLAIM_IDLE` | `60s` | `XAUTOCLAIM` tin treo | worker |
| `JUDGE_MAX_SOURCE_BYTES` | `65536` | mã nguồn tối đa | gateway, worker |
| `EXAM_GRACE_SECONDS` | `10` | độ trễ chấp nhận ghi sau `deadline_at` | gateway, worker |
| `EXAM_MIN_DURATION_MINUTES` | `5` (`1` ở seed / local) | thời lượng tối thiểu | gateway |
| `EXAM_MIN_LEAD_SECONDS` | `60` (`5` ở seed / local) | lên lịch phải cách hiện tại | gateway |
| `EXAM_TICK_INTERVAL` | `5s` | bộ lập lịch | worker |
| `EXAM_SAVE_RATE_PER_MIN` | `240` | giới hạn lưu theo lượt | gateway |
| `EXAM_RUN_LIMIT`, `EXAM_RUN_WINDOW` | `10`, `10m` | `Chạy thử` | gateway |
| `EXAM_SUBMIT_COOLDOWN`, `EXAM_SUBMISSION_CAP` | `15s`, `30` | nộp lời giải | gateway |
| `EXAM_TAB_STALE` | `20s` | tab ghi coi như bỏ | gateway |
| `EXAM_EVENTS_MAX` | `500` | sự kiện / lượt | gateway |
| `EXAM_TESTZIP_MAX_BYTES`, `EXAM_TESTZIP_MAX_UNCOMPRESSED` | `10485760`, `52428800` | nhập test zip | gateway |
| `SIMILARITY_MIN` | `0.60` | ngưỡng tối thiểu của cờ nghi giống | worker |

Sai (không số, ngoài khoảng): thoát 1, nêu tên biến. `JUDGE_TOKEN` và mọi `JUDGE_*` có trong `.env.example` (không giá trị thật).

### 8.2 Compose (dev + test + VPS)

```yaml
judge:
  build: { context: ./deploy/judge }          # Dockerfile ghim go-judge v1.13.0 + g++/gcc 14.2.0
  privileged: true
  cgroup: host                                # = --cgroupns=host; thiếu → "cgroup path is empty", container thoát
  shm_size: 256m
  cpus: ${JUDGE_CPUS:-2.0}
  mem_limit: ${JUDGE_MEM_LIMIT:-2g}
  environment:                                # chỉ các biến này
    JUDGE_PARALLELISM: ${JUDGE_PARALLELISM:-2}
    JUDGE_TOKEN: ${JUDGE_TOKEN}
    JUDGE_EXTRA_ARGS: ${JUDGE_EXTRA_ARGS:-}   # colima arm64: -no-seccomp
  entrypoint: ["/opt/start.sh"]
  networks: [judge_net]                       # internal: true; KHÔNG ở mạng mặc định; không ports; không volumes
worker:
  networks: [default, judge_net]
networks:
  judge_net: { internal: true }
```

`deploy/judge/start.sh` là script `sh` nằm trong image judge (luật "không shell" chỉ áp cho image gateway / worker). Image gateway / worker **không** thêm gì của PE ngoài binary; ngưỡng kích thước (`FEAT-pg-foundation` 07-AC10: gateway < 80 MB, worker < 40 MB) giữ nguyên. Lưu ý: job `question.suggest` gọi `internal/llm` — nếu chạy ở worker mà SDK làm binary worker vượt 40 MB thì dev chạy job này ở **gateway** (job 202 trong tiến trình gateway) hoặc ghi `proposals.md`; không nới ngưỡng im lặng.

### 8.3 Hiệu năng và SLO

| Chỉ số | Mục tiêu | Đo bằng |
| --- | --- | --- |
| Lưu câu trả lời (`PUT answers`) | p95 ≤ 150 ms ở 300 phiên đồng thời, lỗi < 0,5 % | k6 `autosave` |
| Bắt đầu làm bài | p95 ≤ 300 ms (1.000 sinh viên mở cùng phút ≈ 17/s) | k6 / `TestStartAttemptLoad` |
| API đọc (danh sách, kết quả, `attempts/mine`) | p95 ≤ 300 ms | k6 |
| Chấm 60 bài code dồn (1 biên dịch + 10 test, `<bits/stdc++.h>`) | xong p95 ≤ 60 s (PoC `parallelism=4`: 18,2 s) | k6 `judge_burst` |
| `Chạy thử` lúc rảnh | p95 ≤ 5 s | `TestRunLatencyIdle` |
| Thông lượng chấm | ≈ 190 bài/phút (4 CPU); 240 bài/5 phút (20 lớp × 60) hết trong ≤ 8 phút | research; k6 |
| Mở / đóng / tự nộp / công bố đúng giờ | ≤ 10 s sau mốc | `TestExamStateMachine`, E2E |
| Chat INTERACTIVE khi chấm dồn | TTFT không chậm hơn +20 % | k6 `mixed` (SYSTEM_DESIGN §5) |
| Kết quả `GET …/results` (1.000 dòng) | p95 ≤ 300 ms; cuộn 60 fps | k6 / Playwright |
| Container `judge` lúc rảnh | ≤ 100 MiB RAM (PoC 35,6 MiB) | `docker stats` |

Luật mở rộng: gateway không trạng thái (đồng hồ ở DB, khoá ở Redis, tab ở DB); việc nặng ở Stream (luật 12); mọi danh sách cursor (luật 13); POST có tác dụng phụ idempotent (luật 14); không cache điểm (luật 15); mọi lời gọi LLM (chỉ `suggest`) qua Scheduler làn BATCH (luật 11); thời gian chấm **không** chung làn LLM.

### 8.4 Bảo mật

- Mã sinh viên chỉ chạy trong container `judge` (D55, D58): không mạng, không volume, không secret, giới hạn CPU / RAM / thời gian / output / tiến trình; gateway / worker không `exec`; judge chỉ nhận mã nguồn + dữ liệu test + giới hạn (không danh tính); token Bearer; không công bố cổng.
- Danh tính từ JWT; mọi truy vấn của sinh viên lọc `student_id = sub`; id của người khác → 404; ADMIN → 403.
- Chống rò: DTO riêng (6.4), `question_options` không có cờ đúng, `TestStudentDTOAllowlist`, `TestNoAnswerLeak`, lượt đã nộp không đọc lại đề trước khi công bố, canary trong log.
- Liêm chính (D56): xem 4.7; **không tự trừ điểm**.
- Nội dung do người dùng nhập (đề, đáp án, ghi chú) hiển thị bằng bộ render an toàn (không HTML thô); CSV chống chèn công thức; zip chống zip-slip / zip bomb; mọi giới hạn kích thước có kiểm.
- Không log: mã nguồn, input / output test, `stderr` của chương trình, tên / MSSV / email, token judge, `answer_key`.
- Dữ liệu cá nhân phát sinh: bài làm, mã nguồn, log hành vi (loại + thời điểm + độ dài dán), điểm — cùng vòng đời lớp; `RUN` xoá 30 ngày sau công bố; PR quyết hạn xoá (Q10, Q20).

### 8.5 Thư viện

Không thêm thư viện ngoài bảng `ARCHITECTURE.md` §3: phía Go chỉ `net/http`, `encoding/json`, `archive/zip`, `crypto/sha256`, `hash/fnv`, `math/rand/v2`, `go-redis`, `pgx`, `sqlc`, `shopspring/decimal`, `chi`; phía web **không** thư viện editor, **không** thư viện Markdown ngoài (bộ render tối giản tự viết — Q21); biểu đồ phân bố dùng `recharts` đã có.

### 8.6 CI

Thêm job `judge-attacks` (runner amd64): dựng `deploy/judge` với seccomp bật, chạy `TestSandboxAttacks`; thêm `exam.spec.ts` (không `@real`) vào job Playwright; `sqlc diff` và contract như PG.

## 9. Kiểm thử

| Tầng | Gói / tệp | Test chính |
| --- | --- | --- |
| Đơn vị Go | `internal/quiz` | `TestQuizGrade`, `TestQuizPartialFormula`, `TestQuizInvalidOption` |
| | `internal/exam` | `TestScoringDecimal` (`testdata/exam_scoring.csv`), `TestExamStateMachine`, `TestEffectiveStatus`, `TestShuffle*`, `TestSave*`, `TestSubmit*`, `TestAutoSubmit*`, `TestSingleWriterTab`, `TestScheduleValidationsAll`, `TestEditLockMatrix`, `TestExamGuardMatrix`, `TestStudentDTOAllowlist`, `TestResultPayloadWhitelist`, `TestCSV*`, `TestAudit*`, `TestScoreIgnoresIntegritySignals` |
| | `internal/exam/similarity` | `TestWinnowing*`, `TestLexerCpp` |
| | `internal/judge` | `TestVerdictMapping`, `TestChecker*`, `TestClient*`, `TestSubmissionResultAssembly`, `TestNoSourceInLogs` |
| Tích hợp (`-tags integration`; Postgres + Redis + **judge thật**) | `internal/judge`, `internal/exam`, `internal/integration` | `TestSandboxAttacks` (15 ca), `TestJudgeConsumerIdempotent`, `TestJudgeKillWorkerMidRun`, `TestRunWhenJudgeDown503`, `TestPublishFlowConcurrency`, `TestNoAnswerLeak`, `TestExamLockContract`, `TestExamIsolation`, `TestAllExamRoutesGuarded`, `TestRegrade*`, `TestVerifyReference*`, `TestSuggest*` (provider `fake`) |
| Contract | `internal/contract` | `TestExamContract`, `TestExamErrorCodes` (56 thao tác ↔ `openapi.yaml`) |
| Schema | `internal/store` | `TestExamSchema`, `TestExamConstraints`, `TestExamIndexes` |
| Frontend đơn vị | `vitest` | `clock.test.ts`, `saveQueue.test.ts`, `markdown.test.ts` |
| E2E | `frontend/e2e/exam.spec.ts` | 12 ca của US-PE-09 AC7 |
| Tải | `benchmarks/load/exam-submit.js` | `judge_burst`, `autosave`, `mixed` |
| Seed | `scripts/check-exam-seed.mjs` | `bank`, `scores`, `demo` |
| Cổng | `scripts/gate-pe.sh` | toàn bộ trên |

**Dữ liệu seed cần:** 20 câu trắc nghiệm đã duyệt, 5 câu AI nháp, 2 bài code có test + lời giải mẫu đã xác minh, 2 bài thi đã công bố với 30 lượt, 1 cặp mã giống nhau (US-PE-09). Test thời gian dùng đồng hồ giả; test E2E hết giờ dùng `EXAM_MIN_DURATION_MINUTES=1` + `page.clock`.

## 10. Câu hỏi mở và quyết định đã chốt

**Câu hỏi mở** → `QUESTIONS.md` (31 câu; 15 đánh **[CHỦ DỰ ÁN]**: điểm số, quyền, dữ liệu cá nhân). Mặc định của BA cho phép dev **không bị chặn**.

**Đã chốt (nguồn):** D54 (PE sau P2, tạo `question_bank`); D55 + D58 (sandbox `go-judge` v1.13.0, chung máy, mạng riêng, `--privileged --cgroupns=host`, seccomp bật trên amd64 / tắt trên colima arm64); D56 (tự công bố khi đóng; khoá chat, một lượt, log chỉ giảng viên, so độ giống, xáo trộn); D57 (AI chỉ nháp, giảng viên duyệt); D45 (bảng dạng cuối); luật 5 (điểm = code thuần); nộp code nhiều lần, lần cuối tính (`plan.md`); textarea không thư viện editor (PM); Admin không đọc nội dung lớp.

**Đề xuất đổi tài liệu nền (cần PM quyết; BA không sửa file ngoài `docs/specs/FEAT-weekly-exam/**`, FLOWS F19, PRD M15):**

1. `ARCHITECTURE.md` §7 / `FEAT-ui-foundation` 7.5: thêm `/exams` cho sinh viên + mục nav "Bài thi" (Q25); đổi đếm mục 7 / 12 / 15 → 8 / 13 / 16 và `ACCESS`; test US-PU-04 AC3 cập nhật cùng commit.
2. `ARCHITECTURE.md` §5: thêm 56 thao tác ở 6.2 (nhóm mới "Bài thi" và "Ngân hàng câu hỏi"); nhóm "Luyện đề" giữ `GET/POST …/questions` nhưng **PE** sở hữu `…/questions` (P9 dùng lại, thêm `extract` / `generate`); `POST …/questions/suggest` là đường mới của PE (khác `generate` của P9: không dùng chunk / trích dẫn).
3. `ARCHITECTURE.md` §2 (cấu trúc `backend-go`): thêm `internal/exam`, `internal/exam/similarity`, `internal/judge`; §4: đổi dòng "(kế tiếp sau P2) exams" thành `00005_weekly_exam` (13 bảng — thêm `code_drafts`, `exam_appeals`) và đẩy `00005…00016` hiện có lùi một số (hoặc ghi ánh xạ ở `PROGRESS.md` theo D45); §8: thêm biến `JUDGE_*`, `EXAM_*`, `SIMILARITY_MIN`; §9: seed thêm dữ liệu PE; §10: dòng "Sandbox" ở bảng kiểm thử.
4. `SYSTEM_DESIGN.md` §3.4: thêm Stream `judge.submit`, `judge.run` (+ `.dead`); S1 thêm container `judge` (D58).
5. `FEAT-course-foundation` 4.7: bảng bậc "Hôm nay" thêm 10 `Kind` mới (4.10); 4.9: thêm topic outbox của PE; `notifications.type` thêm 5 giá trị.
6. `FEAT-course-foundation` US-P2-12 AC7: ngân sách seed "≤ 3 phút" → "≤ 5 phút" khi gồm dữ liệu PE (US-PE-09 AC5).
7. `FEAT-pg-foundation` 07-AC10 (ngưỡng image) — xem 8.2: nếu worker vượt 40 MB vì `internal/llm`, chạy `question.suggest` ở gateway.
8. `DESIGN.md` §10 / §13 / §14: thêm hợp đồng bố cục `/exams`, `/exams/[id]`, `/exams/[id]/results`, `/exams/[id]/take`; thành phần mới ở 7.4; bảng dịch nhãn kết quả (7.2).
9. Phúc khảo bài thi dùng bảng `exam_appeals` riêng ở PE; khi P4 có `escalation_tickets` (`GRADE_APPEAL`), PM quyết gộp.
10. `submission_id` đóng vai job id cho `Chạy thử` / `Nộp lời giải` (không tạo dòng `jobs` mỗi lần) — luật 12 thoả "202 + id + SSE"; `jobs` dành cho việc lâu của giảng viên (verify, suggest, regrade, similarity) (Q29).
11. `DECISIONS.md` D46: câu chữ "chỉ Go + `docling-serve`" → thêm `go-judge` (đã nêu ở D55 / D58).

## 11. Truy vết: PRD → FLOWS → phase → US → FR → test

| PRD | FLOWS | Phase PE | US | FR | Test chính |
| --- | --- | --- | --- | --- | --- |
| M15 (ngân hàng câu hỏi); M4 (bảng dùng lại) | F19 b1 | L1, L3 | US-PE-01, US-PE-03 | FR-1…FR-5, FR-10…FR-12 | `TestExamSchema`, `TestQuizGrade`, `TestReviewApprove`, `TestSuggest*` |
| M15 (chấm code, sandbox); D55, D58 | F19 b5 + nhánh "sandbox chết" | L2 | US-PE-02 | FR-6…FR-9 | `TestSandboxAttacks`, `TestJudgeConsumerIdempotent` |
| M15 (bài thi, lên lịch); M14 | F19 b2; F14 | L3, L4 | US-PE-04 | FR-13, FR-14 | `TestScheduleValidationsAll`, `TestExamStateMachine` |
| M15 (làm bài trắc nghiệm) | F19 b3–b4 + nhánh mất mạng / hết giờ / hai tab / nộp trùng | L4 | US-PE-05 | FR-15, FR-16, FR-18 | `TestAutoSubmitOnTimeout`, `TestSingleWriterTab`, `exam.spec.ts` |
| M15 (làm bài code) | F19 b3 + nhánh lỗi biên dịch / hết giờ khi gõ | L4 | US-PE-06 | FR-17, FR-18 | `TestRunSamplesOnly`, `TestLastSubmissionCounts`, `TestHiddenTestsNeverInPayload` |
| M15 (liêm chính); D56; M4 | F19 liêm chính; F12 | L5 | US-PE-07 | FR-19, FR-20 | `TestExamLockContract`, `TestWinnowing*`, `TestScoreIgnoresIntegritySignals` |
| M15 (công bố, kết quả, phúc khảo); G8; M8 | F19 b6–b8 + nhánh "test sai"; F11 | L6 | US-PE-08 | FR-21…FR-24, FR-18 | `TestAutoPublishOnce`, `TestNoAnswerLeak`, `TestRegrade*`, `TestAppeal*` |
| G8, M15 AC (seed, tải, cổng) | F19 đầu-cuối | cổng PE | US-PE-09 | FR-25 | `gate-pe.sh`, `check-exam-seed.mjs`, `exam-submit.js` |

**Yêu cầu của `PE.md` → AC:** L1 bảng + quyền → 01-AC1…AC5; Quiz Engine → 01-AC6…AC8; L2 một container sandbox / `internal/judge` / hàng đợi / chấm lại / `Chạy thử` / test tấn công → 02-AC1…AC15, 06-AC6, 08-AC15; L3 `/questions`, gợi ý AI, lời giải mẫu, `/exams` → 03-AC1…AC14, 04-AC1…AC13; L4 "Hôm nay", đồng hồ, tự lưu, hết giờ tự nộp, xáo trộn, màn 375 / 1.024 px → 04-AC11, 05-AC1…AC14, 06-AC2; L5 khoá chat, log, so độ giống → 07-AC1…AC11; L6 tự công bố, kết quả sinh viên, `/exams/[id]/results`, CSV, sửa điểm, phúc khảo → 08-AC1…AC20; cổng PE (`TestSandboxAttacks`, `TestScoringDecimal`, `TestNoAnswerLeak`, contract, Playwright, k6) → 09-AC6…AC8. **Ghi chú:** `PE.md` ghi "sửa điểm thủ công có lý do + `audit_log`" → 08-AC13; "ghi rõ trên giao diện đây không phải giám thị" → 07-AC6.
