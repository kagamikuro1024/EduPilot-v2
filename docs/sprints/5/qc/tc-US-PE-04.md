# QC test case — US-PE-04 (`/exams` giảng viên: tạo bài thi, chọn câu, xem trước, lên lịch, bỏ lịch, mở / đóng đúng giờ, gia hạn, "Hôm nay", danh sách của sinh viên)
Nguồn: `docs/specs/FEAT-weekly-exam/US.md` US-PE-04 AC1–AC13 + `SRS.md` 4.2 (bài thi, máy trạng thái 4.2.x, khoá sửa 4.2.5), 4.10 (nguồn việc, thông báo, outbox), 5.6–5.7, 6.2 #17–#28, 7.1–7.3, 7.5 (nav). Góp ý #1 (ACCEPTED): nav **8 / 13 / 16**; AC13 **áp dụng**. Hộp đen; stack như `tc-US-PE-02.md`.

Tiền điều kiện chung: stack riêng; `E` = id bài thi; `QM1…QM5` = 5 câu MCQ đã `APPROVED` (QC tạo qua API PE-03); `QC1` = câu CODE đã duyệt (≥ 1 test mẫu + ≥ 1 test ẩn, verify xong). Công cụ: **S** shell, **D** SQL, **A** Chrome, **G** `go test`, **R** Redis. Đồng hồ: QC không chỉnh giờ máy; dùng bài thi khung 2–3 phút ở tương lai gần (≥ 60 s) và DB (`update exams set opens_at=…`) **chỉ ở DB QC** để kiểm mốc; "đồng hồ giả" chỉ ở test Go của dev.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PE04-01 | AC1 | GV | **S** `POST $Q/exams` (có `Idempotency-Key`) `{"title":"Kiểm tra tuần 9","opens_at":"2026-12-01T01:00:00Z","closes_at":"2026-12-01T02:00:00Z","duration_minutes":45}`; đọc lại | `201`, `status=DRAFT`; mặc định `shuffle_questions=true`, `shuffle_options=true`, `max_score=10`, `rounding_step=0.01`, `multi_scoring=PARTIAL`, `reveal_answers=true`, `appeal_days=7` |
| TC-PE04-02 | AC1 (sai / biên) | – | **S** `title` rỗng / 121; `instructions` 4.000 / 4.001; `duration_minutes` 4 (`EXAM_MIN_DURATION_MINUTES`=5), 5, 300, 301; `duration` > khung (`DURATION_EXCEEDS_WINDOW`); `closes_at = opens_at`, `closes_at < opens_at` (`CLOSES_BEFORE_OPENS`); `rounding_step` 0,3; `max_score` 0 / 100 / 100,01; thiếu `Idempotency-Key`; client gửi `kind` | biên đúng qua, ngoài biên `422` đúng mã (`DURATION_TOO_SHORT`, `DURATION_EXCEEDS_WINDOW`, `CLOSES_BEFORE_OPENS`…); thiếu key `422 IDEMPOTENCY_KEY_REQUIRED`; `kind` do client gửi **bị bỏ qua** (suy ra từ mục) |
| TC-PE04-03 | AC1 (quyền, giờ) | – | **S** TA tạo/sửa nháp; SV, ADMIN, người ngoài lớp; `opens_at` có múi giờ `+07:00` và `Z` | TA `201`/`200`; SV/ADMIN `403`; ngoài lớp `403 course`; hai biểu diễn giờ cùng lưu một thời điểm UTC; giao diện hiển thị `Asia/Ho_Chi_Minh` |
| TC-PE04-04 | AC1 (idempotency) | – | **S** `POST` cùng `Idempotency-Key` hai lần; cùng key khác thân | cùng `id`, 1 dòng; khác thân → `422 IDEMPOTENCY_KEY_REUSED` |
| TC-PE04-05 | AC1 | – | **G** `-run 'TestCreateExamDefaults|TestExamValidation|TestExamKindDerived' -v` | `ok`; ≥ 20 ca |
| TC-PE04-06 | AC2 (chọn câu) | bài `DRAFT` | **S** `PUT …/items {items:[{question_id,points}],version}` với 5 MCQ + 1 CODE; đọc lại; đổi thứ tự | `position` 1…n theo thứ tự gửi; **thay toàn bộ**; `kind=MIXED` (chỉ MCQ → `MCQ`; chỉ CODE → `CODE`) |
| TC-PE04-07 | AC2 (từ chối) | – | **S** câu `DRAFT`/`PENDING`/`REJECTED`/đã lưu trữ → `ITEM_NOT_APPROVED`; câu lớp khác → `QUESTION_NOT_IN_COURSE`; trùng câu → `DUPLICATE_ITEM`; danh sách rỗng → `NO_ITEMS`; 101 mục; `points` 0, −1, 100,01, 1,234, 100 | `422` đúng mã; biên `points` 0,01 và 100 qua; không đổi dữ liệu khi lỗi (all-or-nothing) |
| TC-PE04-08 | AC2 | – | **S** `PUT …/items` khi bài đã `SCHEDULED` | `409 EXAM_LOCKED` (`reason=status`) |
| TC-PE04-09 | AC2 | – | **G** `-run 'TestPutItems|TestPutItemsRejects|TestPutItemsLockedWhenScheduled' -v` | `ok` |
| TC-PE04-10 | AC3 (xem trước) | bài `DRAFT` | **S** `GET …/preview` hai lần; so thứ tự câu và đáp án; **D** đếm `exam_attempts`, `exam_events`, Redis `ep:exam_lock:*` trước / sau | cùng cấu trúc DTO của lượt làm SV, `preview:true`, xáo **khác nhau** giữa hai lần; **không** `answer_key` ở trường của SV; `exam_attempts`/`exam_events` không đổi, **không** khoá chat |
| TC-PE04-11 | AC3 | – | **S** SV, ADMIN gọi `preview`; GV/TA gọi `POST …/attempts` | `403`; (GV/TA bắt đầu lượt `403 role`) |
| TC-PE04-12 | AC3 | – | **G** `-run 'TestPreviewNoAttemptCreated|TestPreviewShapeEqualsStudent' -v` | `ok` |
| TC-PE04-13 | AC4 (lên lịch) | bài hợp lệ | **S** `POST …/schedule` bằng GV; **D** `status`, `audit_log`, `outbox` topic `exam.scheduled`, bảng thông báo; **S** SVA, SVB `GET /me/notifications` (hoặc nơi hiển thị), SVD (`PENDING`) | `200`, `SCHEDULED`; 1 dòng `audit_log` `exam.schedule`; outbox cùng transaction; **mọi** SV `ACTIVE` có thông báo `EXAM_SCHEDULED` ("Bài thi {tiêu đề} mở lúc {Thứ…, HH:mm dd/MM}, làm trong {n} phút"), SV `PENDING`/`REMOVED` **không**; cache "Hôm nay" bị xoá |
| TC-PE04-14 | AC4 (đủ lỗi một lượt) | bài có **nhiều** lỗi cùng lúc: không câu / câu `PENDING` / câu CODE chưa verify / `Σweight=0` / `opens_at` quá khứ | **S** `schedule` | `422` với **toàn bộ** `details[]` (`OPENS_IN_PAST`, `NO_ITEMS` hoặc `ITEM_NOT_APPROVED`, `CODE_TESTS_MISSING`, `REFERENCE_NOT_VERIFIED`, `TOTAL_WEIGHT_ZERO`), không dừng ở lỗi đầu |
| TC-PE04-15 | AC4 (biên giờ) | – | **S** `opens_at = now + 59 s` rồi `now + 61 s` (`EXAM_MIN_LEAD_SECONDS`=60) | 59 s → `OPENS_IN_PAST`; 61 s → qua |
| TC-PE04-16 | AC4 (quyền + idempotent) | – | **S** TA gọi `schedule`; GV gọi lần 2 khi đã `SCHEDULED`; đếm thông báo | TA `403 reason=role`; lần 2 `200` **không** gửi thông báo thêm, `audit_log` không thêm dòng |
| TC-PE04-17 | AC4 (lớp) | lớp `ARCHIVED` | **S** `schedule` | `409 COURSE_ARCHIVED` |
| TC-PE04-18 | AC4 | – | **G** `-run 'TestScheduleValidationsAll|TestScheduleEffects|TestScheduleIdempotent|TestScheduleTeacherOnly' -v` | `ok` |
| TC-PE04-19 | AC5 (bỏ lịch) | bài `SCHEDULED`, `now < opens_at`, chưa lượt | **S** `POST …/unschedule`; xem thông báo SV | về `DRAFT`; `audit_log` `exam.unschedule`; SV có thông báo "Bài thi … đã bị hoãn, chờ lịch mới"; việc "Hôm nay" biến mất |
| TC-PE04-20 | AC5 (từ chối) | (a) đã tới `opens_at`; (b) đã có `exam_attempts` | **S** `unschedule` | `409 EXAM_LOCKED` `details.reason` = `opened` / `has_attempts` |
| TC-PE04-21 | AC5 (khoá sửa theo trạng thái) | bài `SCHEDULED`, `OPEN`, `CLOSED`, `PUBLISHED` | **S** QC dựng **bảng 4 trạng thái × 12 trường** tự đếm: sửa `opens_at`, `duration_minutes`, `shuffle_*`, `max_score`, `rounding_step`, `multi_scoring`, danh sách mục (khoá); `title`, `instructions`, `reveal_answers`, `appeal_days` (sửa được tới `PUBLISHED`); `closes_at` (chỉ qua `extend`) | khoá → `409 EXAM_LOCKED`; trường sửa được → `200`; mọi ô đúng bảng; `audit_log` mỗi lần |
| TC-PE04-22 | AC5 | – | **G** `-run 'TestUnscheduleRules|TestEditLockMatrix' -v` | `ok` |
| TC-PE04-23 | AC6 (mở / đóng đúng giờ) | bài `SCHEDULED` mở sau 70 s, đóng sau 3 phút; worker chạy | **S** poll `select status, effective…` mỗi 500 ms từ lúc lên lịch; ghi thời điểm `status` đổi `OPEN` và `CLOSED` so với `opens_at`/`closes_at` | `OPEN` ≤ `opens_at + 10 s`; `CLOSED` ≤ `closes_at + 10 s` (`EXAM_TICK_INTERVAL` 5 s); outbox `exam.opened`, `exam.closed` mỗi cái **1** dòng |
| TC-PE04-24 | AC6 (effective_status khi bộ lập lịch chết) | dừng worker trước `closes_at` | **S** SV `POST …/attempts/{aid}/submit` / lưu **sau** `closes_at + grace` khi worker tắt; xem `GET …/exams/{eid}` | API dùng `effective_status` (so với `now`): thấy `CLOSED`, lưu bị `409 ATTEMPT_CLOSED`/`EXAM_NOT_OPEN`; không ai làm thêm sau đóng dù cột `status` còn `OPEN` |
| TC-PE04-25 | AC6 (leader, idempotent) | 2 worker | **S** chạy hai worker; `redis-cli get ep:exam:tick:leader`; đếm outbox `exam.opened` sau mốc | **1** leader; **1** outbox mỗi mốc; kill leader → worker kia nhận trong ≤ TTL |
| TC-PE04-26 | AC6 (đóng quá hạn) | tắt worker qua `closes_at`, bật lại | **D** bài còn `SCHEDULED` mà `closes_at` đã qua | chuyển thẳng `CLOSED` (bỏ qua `OPEN`); không chuyển ngược |
| TC-PE04-27 | AC6 | – | **G** `-run 'TestExamStateMachine|TestEffectiveStatus|TestTickIdempotent|TestTickSkipsToClosed' -v`; `-tags integration -run TestTickLeaderLock -v` | `ok` |
| TC-PE04-28 | AC7 (gia hạn) | bài `OPEN`, 1 SV `IN_PROGRESS` có `deadline_at = closes_at cũ` | **S** `POST …/extend {closes_at: cũ + 10 phút}`; SV lưu bài; xem `deadline_at` | `200`; `deadline_at` SV = `min(started_at + duration, closes_at mới)`; SV thấy "Giảng viên đã gia hạn bài thi đến {HH:mm}" ở lần lưu kế; `audit_log` `exam.extend` |
| TC-PE04-29 | AC7 (từ chối) | – | **S** mốc mới = cũ; < cũ; cũ + 24 h + 1 s; cũ + 24 h; bài `CLOSED`; TA gọi | `422 CLOSES_NOT_LATER` (bằng / ngắn hơn); cũ + 24 h + 1 s `422`; cũ + 24 h `200`; `CLOSED` → `409 EXAM_LOCKED`; TA `403` |
| TC-PE04-30 | AC7 | – | **G** `-run 'TestExtendOnlyLater|TestExtendRecomputesRunningDeadlines|TestExtendAfterCloseRejected' -v` | `ok` |
| TC-PE04-31 | AC8 (xoá / nhân bản) | bài `DRAFT`, bài `SCHEDULED` | **S** `DELETE` `DRAFT` bằng GV; `DELETE` `SCHEDULED`; `DELETE` bằng TA; `clone` bằng TA | `204` + `exam_items` cascade (đếm 0); `409 EXAM_LOCKED`; `403`; bản sao `DRAFT`, tiêu đề + " (bản sao)", cùng mục / điểm / cài đặt, **không** giờ |
| TC-PE04-32 | AC8 | – | **G** `-run 'TestDeleteOnlyDraft|TestCloneExam' -v` | `ok` |
| TC-PE04-33 | AC9 (theo vai) | bài `DRAFT`, `SCHEDULED`, `OPEN`, `CLOSED`, `PUBLISHED` | **S** `GET $Q/exams` bằng GV, TA, SV; `GET …/exams/{eid}` từng bài bằng SV | Staff thấy cả 5 trạng thái với `effective_status`, `items_count`, `attempts{started,graded}`; SV **không** thấy `DRAFT` (`jq '[.items[].status]|unique'`), `GET` bài `DRAFT` bằng SV → `404`; `my_score` **chỉ** khi `PUBLISHED`; bản SV **không có** `items`, `override`, số câu theo loại; có `ETag` + `304` |
| TC-PE04-34 | AC9 (phân trang) | ≥ 120 bài | **S** `?status=OPEN`, `limit`, `cursor` | lọc đúng; cursor ổn định (xem `tc-US-PE-01` TC-PE01-30) |
| TC-PE04-35 | AC9 | – | **G** `-run 'TestExamListStaffVsStudent|TestExamDetailStudentProjection|TestStudentNeverSeesDraft' -v` | `ok` |
| TC-PE04-36 | AC10 (đồng thời) | 2 GV | **S** hai `PUT …/exams/{eid}` cùng `version`; GV A `schedule` trong lúc GV B `PUT` | đúng 1 thành công, 1 `409 VERSION_CONFLICT` kèm bản hiện tại |
| TC-PE04-37 | AC10 (đua với duyệt) | – | **S** `schedule` song song với `PUT …/review REJECT` cho 1 câu trong bài, lặp 30 lần | không bao giờ ra bài `SCHEDULED` chứa câu `REJECTED`; `schedule` kiểm lại trong transaction (lỗi `ITEM_NOT_APPROVED` hoặc thắng trước) |
| TC-PE04-38 | AC10 | – | **G** `-tags integration -run 'TestScheduleRaceWithQuestionReject|TestExamEditVersionConflict' -v` | `ok` |
| TC-PE04-39 | AC11 ("Hôm nay") | SVA, SVB; bài mở sau 2 giờ, mở ngay, đang làm | **S** `GET /me/today` bằng SVA tại 3 thời điểm: trước 48 giờ (không có), trong 48 giờ trước `opens_at`; đang mở chưa làm; đang làm dở; sau nộp; sau đóng | `EXAM_UPCOMING` (bậc 42, "{tiêu đề} · {Thứ…, HH:mm}" · "Làm trong {n} phút. Chuẩn bị máy tính nếu có bài lập trình."); `EXAM_OPEN` (bậc 8, "Mở đến {HH:mm dd/MM}. Bạn có {n} phút để làm.", chỉ khi chưa có lượt); `EXAM_IN_PROGRESS` (bậc 5, "Còn {m} phút. Làm tiếp.", `/exams/{id}/take`); biến mất khi nộp / đóng; `recommended` là việc bậc thấp nhất |
| TC-PE04-40 | AC11 (cache, người ngoài) | – | **D** `redis-cli ttl ep:today:<uid>` / key; bắt đầu / nộp / mở / đóng; GV, TA, SV lớp khác, SV `PENDING` | cache bị xoá ≤ 2 s sau `exam.scheduled|opened|closed` và khi bắt đầu / nộp; chỉ SV `ACTIVE` của lớp thấy việc; GV/TA không có việc SV |
| TC-PE04-41 | AC11 | – | **G** `-run 'TestExamProvidersStudent|TestExamTodayInvalidation' -v` | `ok` |
| TC-PE04-42 | AC12 (UI danh sách GV) | GV, 1440 | **A** `/exams`: nhóm `Đang mở` / `Sắp tới` / `Đã đóng` / `Nháp`; nút chính duy nhất `Tạo bài thi`; trạng thái rỗng "Chưa có bài thi nào. Tạo bài thi đầu tiên từ ngân hàng câu hỏi." | đúng; **1** `[data-variant=primary]`; chuỗi tiếng Việt theo SRS 7.2 |
| TC-PE04-43 | AC12 (trang soạn) | – | **A** `/exams/[id]`: 3 phần `Thông tin` · `Câu hỏi` · `Xem trước`; chọn câu từ ngân hàng; `Lên`/`Xuống` bằng bàn phím; `Lên lịch` thiếu điều kiện → **danh sách việc cần sửa** có liên kết tới đúng chỗ; `Lên lịch` thành công **không hộp thoại xác nhận**, có `Bỏ lịch` | khớp; mỗi lỗi lên lịch là liên kết focus đúng trường; không `confirm`/`alert`/dialog; sửa tại chỗ có tự lưu nháp (`useAutosaveDraft`), đóng tab giữa chừng không mất chữ |
| TC-PE04-44 | AC12 (TA, hành động theo trạng thái) | TA; bài `DRAFT`, `OPEN`, `CLOSED` | **A** TA xem `/exams/[id]`; GV xem 3 trạng thái | TA: nút `Lên lịch` **ẩn** + "Chỉ giảng viên lên lịch được"; GV: `Lên lịch` ở nháp, `Gia hạn` ở đang mở, `Xem kết quả` ở đã đóng; mỗi vùng 1 hành động chính |
| TC-PE04-45 | AC12 (đáp ứng, a11y) | – | **A** 1440 / 1024 / 390: `AUDIT_SRC`, `TOUCH_SRC`; chỉ bàn phím; axe; `bash scripts/ui-antipatterns.sh` | `{ox:0,cut:[],ell:[]}`; `[]`; 0 `serious`/`critical`; 19 phép ✓ |
| TC-PE04-46 | AC12 | – | **S** `$PW exam.spec.ts -g 'exam editor'` | `rc=0` |
| TC-PE04-47 | AC13 (SV) | SV, 375 | **A** `/exams`: nhóm `Đang mở`/`Sắp tới`/`Đã có điểm`; hành động mỗi hàng `Bắt đầu làm bài` / `Tiếp tục` / `Xem kết quả` / không có; rỗng "Lớp của bạn chưa có bài thi nào."; `AUDIT_SRC`, `TOUCH_SRC` | không tràn ngang; vùng chạm ≥ 44 px; đúng 1 hành động mỗi hàng |
| TC-PE04-48 | AC13 (nav, góp ý #1) | 4 vai, build thường | **A** đếm mục nav: SV, TA, GV, ADMIN; mobile 375: mục "Bài thi" ở menu "Thêm"; `tc-US-PU-04` TC-PU04-10/12/13 đã sửa | **SV 8, TA 13, GV 16, Admin 6** (Admin không có "Bài thi"); "Bài thi" `/exams` đúng vị trí SRS 7.5; mobile nằm dưới "Thêm" |
| TC-PE04-49 | AC13 | – | **S** `$PW exam.spec.ts -g 'student exam list'` ở 375 | `rc=0` |
| TC-PE04-50 | tổng | – | **S** `go vet ./... && golangci-lint run && go test -race -count=1 ./... && go test -count=1 ./internal/contract/...`; `pnpm -C frontend lint && build`; `audit.mjs` bốn vai (`tc_00_matrix` cập nhật nav 8/13/16) | `rc=0`; audit FAIL 0 |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Lên lịch với đề hỏng / câu bị loại ngay trước đó | 14, 37 |
| Bỏ lịch khi đã có người làm | 20 |
| Sửa cài đặt tính điểm khi đã có lượt làm | 21 |
| Bộ lập lịch chết / hai bộ lập lịch | 24, 25, 26 |
| Gia hạn ngắn hơn / quá 24 h / sau đóng | 29 |
| SV thấy nháp / đáp án / số câu theo loại | 33 |
| Thông báo lịch cho SV không còn thuộc lớp | 13 |
| TA lên lịch / gia hạn / xoá | 03, 16, 29, 31, 44 |

## Câu hỏi cho BA / PM
- **Q-QC-PE04-1** — AC6 đo "≤ 10 s" từ `opens_at`; QC đo bằng poll DB mỗi 500 ms trên máy dev; chấp nhận độ trễ đo ≤ 1 s? — *chờ BA*.
- **Q-QC-PE04-2** — AC4 "thông báo `EXAM_SCHEDULED` cho mọi SV `ACTIVE`": nơi hiển thị thông báo (chuông / `GET /me/notifications`) — QC sẽ đọc API thông báo của P2; nếu chưa có API đọc, QC chấm ở bảng `notifications`. Đúng? — *chờ BA*.
- **Q-QC-PE04-3** — AC13 và TC-PU04: mục "Bài thi" vị trí cụ thể trong thứ tự nav (SV: sau "Luyện đề"? sau "Lịch"?) chưa nêu ở AC; QC chấm theo SRS 7.5 sau khi BA sửa — *chờ BA*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-weekly-exam, APPROVED); AC13 áp dụng theo góp ý #1.

Tổng: 50 TC.
