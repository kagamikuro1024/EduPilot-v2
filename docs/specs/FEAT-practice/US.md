# FEAT-practice Ngân hàng câu hỏi mở rộng, luyện đề, thi thử, form trắc nghiệm (sprint 10 · P9)
Nguồn: PRD M4, M8 (phần Form trắc nghiệm), M15 · FLOWS F12, F14, F19 · `docs/phases/P9.md` (L1–L3) · `docs/ARCHITECTURE.md` §4 (migration `practice`), §5 (route Luyện đề), §7 · `docs/design/DESIGN.md` §13, §14.17–§14.20 · `docs/UX.md` mục 6 (`/practice/[id]`) · spec đã có `FEAT-weekly-exam` (bảng `question_bank`, Quiz Engine, `/questions`, khoá chat trong giờ thi), `FEAT-docs-calendar` (tài liệu, chunk, `ANSWER_KEY`), `FEAT-private-chat-pii` (chat, khoá giờ thi), `FEAT-crm-360` (Hôm nay, heartbeat `PRACTICE`), `FEAT-gradebook`
Phase: P9, sprint 10. Trạng thái: **DRAFT** (câu hỏi mở ở `QUESTIONS.md`).

**Đã có từ PE (không làm lại):** `question_bank` / `question_options` (loại `MCQ_SINGLE`, `MCQ_MULTI`, `TRUE_FALSE`, `CODE`; enum `origin` đã có `EXTRACTED`, `GENERATED`; cột `citations jsonb` còn trống), duyệt câu (`PUT …/questions/{id}/review`), `/questions` + `QuestionReview` + `SuggestPanel`, `question.suggest` (AI nháp từ `source_text`), Quiz Engine `internal/quiz` (MCQ + đúng–sai), bài thi tính điểm (`/exams`, F19), khoá chat trong giờ thi, nguồn Hôm nay `QUESTION_REVIEW` (90). **Bài QUIZ tính điểm = bài thi của PE** (không có `assignments.type=QUIZ`).
**Còn là mock:** toàn bộ `/practice`, `/practice/[attemptId]`, `/practice/history` (`frontend/src/mock/practice.ts`, `features/practice/*`).

**Quy ước chung cho mọi story:**
- 375 px: không tràn ngang, vùng chạm ≥ 44 px. Màn sinh viên mobile-first.
- `prefers-reduced-motion: reduce` được tôn trọng. `bash scripts/ui-antipatterns.sh` sạch; axe 0 lỗi serious / critical.
- Chữ tiếng Việt, **sinh viên không thấy từ kỹ thuật AI** (DESIGN §13) và theo luật chữ UI của chủ dự án: không câu mô tả dưới tiêu đề, không tooltip thừa; mỗi dòng = nhãn + điều khiển + giá trị.
- Không gamify: không streak, bảng xếp hạng, huy hiệu, vòng tiến độ, confetti.
- Mọi API theo `ARCHITECTURE.md` §5: phân trang con trỏ (`limit` ≤ 100), POST có tác dụng phụ nhận `Idempotency-Key`, sửa đồng thời dùng `version` → 409.
- Mọi lời gọi LLM qua `internal/llm` + Scheduler: trích / sinh = **BATCH**, giải thích đáp án = **INTERACTIVE**. Tên và MSSV được che trước mọi lời gọi.
- Điểm luyện đề / thi thử **không vào sổ điểm**, không công bố, không tính điểm cuối kỳ.
- Chuỗi trạng thái và mã lỗi mới của API ghi vào `openapi.yaml` trong cùng story; contract test xanh.

Thứ tự thi công đề xuất (cắt theo `WORKFLOW.md` §6: `GenerateQuestions` và `FormsImportSource` bỏ trước): **P9-01 → 02 → 04 → 05 → 06 → 07 → 09 → 10 → 08 → 12; 03 và 11 cuối, bỏ được**.

---

## US-P9-01: Hệ thống muốn lược đồ luyện đề dạng cuối, Quiz Engine chấm cả câu trả lời ngắn và không rò đáp án, để mọi story sau dựng trên nền đúng
Ưu tiên: Must · Ước lượng: M · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given DB sạch When `goose up` Then migration mới (`00019_practice`) tạo `practice_exams`, `practice_attempts`, `practice_answers` (và bảng phụ ở SRS §5); mọi bảng có `course_id` đứng đầu mọi FK phức hợp tới `question_bank(course_id, id)` và `enrollments`; chèn dòng con khác `course_id` với cha bị DB từ chối (`23503`); `goose down` rồi `up` thành công; `sqlc generate && sqlc diff` sạch.
- AC2. Given Quiz Engine When chấm `SHORT` Then so khớp sau chuẩn hoá (bỏ dấu cách thừa đầu / cuối / giữa, hoa thường, bỏ dấu câu cuối, Unicode NFC; **không** bỏ dấu tiếng Việt) với tập đáp án chấp nhận được; sai → `0`, không gọi LLM; bộ test có ≥ 30 ca (gồm "Hà Nội" / "ha noi", khoảng trắng kép, dấu chấm cuối).
- AC3. Given bộ test Quiz Engine When `go test -race ./internal/quiz/...` Then đúng 100% (MCQ một / nhiều, đúng–sai, trả lời ngắn, câu bỏ trống, id lạ → `ErrInvalidOption`); điểm là `shopspring/decimal` (không `float64`); `internal/quiz` **không import `internal/llm`** (test kiểm danh sách import).
- AC4. Given `Validate` của PE (hiện trả `TYPE_NOT_SUPPORTED` cho `SHORT`, `ESSAY`) When tạo câu `SHORT` có đáp án chấp nhận được / `ESSAY` có đáp án mẫu + rubric ngắn Then lưu được với `review_status=DRAFT`; câu `SHORT`, `ESSAY` **không** được thêm vào bài thi tính điểm của PE (thêm vào `exam_items` → `422 TYPE_NOT_SUPPORTED_IN_EXAM`).
- AC5 (phân quyền / không rò). Given endpoint của sinh viên (practice) When quét `student_dto_allowlist`-tương đương (`practice/testdata/student_dto_allowlist.json`) Then DTO câu hỏi của sinh viên **không có** `answer_key`, `correct`, `explanation`, `citations` cho tới khi câu đó đã được trả lời (luyện theo chủ đề) hoặc lượt đã nộp (thi thử); `TestPracticeStudentDTOAllowlist` dùng `reflect` so thẻ `json`.
- AC6 (nhánh lỗi). Given câu `SHORT` có đáp án rỗng hoặc chuỗi chỉ gồm dấu câu When lưu Then `422 VALIDATION_FAILED` (`details[].code = ANSWER_EMPTY`).

### Ngoài phạm vi của story này
- Chấm tự luận (US-P9-08).
- Câu `CODE` trong luyện đề (không đưa vào luyện đề: Q-P9-6).

### Phụ thuộc
- `00017` là migration cuối; `00018` do sprint 9.5 giữ (chỉ mục); `00019` theo `ARCHITECTURE.md` §4 · `FEAT-weekly-exam` 5.2, 4.1.3 · luật 5, 6

---

## US-P9-02: Giảng viên / TA muốn trích câu hỏi từ đề cũ (và ghép đáp án từ tài liệu đáp án) bằng một việc nền, để có ngân hàng câu hỏi mà không gõ tay
Ưu tiên: Must · Ước lượng: L · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given tài liệu loại `EXAM_PAPER` đã `READY` When giảng viên / TA `POST …/questions/extract {document_id, answer_key_document_id?}` (có `Idempotency-Key`) Then `202 {job_id}`; tiến độ qua SSE `job.progress`; việc chạy ở làn BATCH, **không** làm chậm chat (kịch bản k6 chat + extract: TTFT chat xấu đi ≤ 20%).
- AC2. Given đề PDF mẫu `backend-go/internal/practice/testdata/de-mau-trac-nghiem.pdf` (dev tạo bằng `miniPdf` của `scripts/chat-seed-data.mjs`: 24 câu trắc nghiệm có phương án A–D; kèm `dap-an-mau.pdf`; đề `EXAM_PAPER` của seed hiện chỉ có câu tự luận nên không dùng được) và provider `fake` có bản ghi `QUESTION_EXTRACT` trong `fake_llm.json` When việc xong Then ≥ 20 câu lưu `origin=EXTRACTED`, `review_status=PENDING`, mỗi câu có `stem`, ≥ 2 `question_options`, `citations` trỏ tới `document_id` + số trang; **không câu nào thiếu `answer_key`**: câu không xác định được đáp án đúng **không lưu** vào ngân hàng mà vào danh sách "bỏ qua: không tìm thấy đáp án" kèm vị trí trang (`go test ./internal/practice -run TestExtractQuestions -v`).
- AC3. Given kèm `answer_key_document_id` (loại `ANSWER_KEY`) When ghép Then đáp án đúng lấy từ tài liệu đáp án (khớp theo số câu); câu không khớp được theo số → bỏ qua có lý do; **nội dung `ANSWER_KEY` không bao giờ vào prompt tới LLM của kênh sinh viên** và chỉ vào prompt trích khi người gọi là staff của lớp (kiểm `llm_audit`).
- AC4. Given đầu ra LLM When lưu Then qua `llm.Structured(json_schema)`; đầu ra sai schema bị loại trước khi lưu (đếm vào "bỏ qua"); không lưu câu có `stem` rỗng, có < 2 phương án, hoặc `answer_key` trỏ phương án không tồn tại.
- AC5 (idempotent). Given cùng `document_id` + `Idempotency-Key` When gửi lại Then trả cùng `job_id`, không tạo trùng câu; chạy lại **không** `Idempotency-Key` mới trên cùng tài liệu → câu trùng nội dung (cùng `stem` chuẩn hoá) bị bỏ qua ("đã có").
- AC6 (nhánh lỗi). Given tài liệu chưa `READY`, không phải `EXAM_PAPER`, hoặc thuộc lớp khác When gọi Then `409 DOCUMENT_NOT_READY` / `422 WRONG_DOCUMENT_TYPE` / `404` (không lộ tồn tại); LLM quá tải (`OVERLOADED`) → retry 3 lần rồi dead-letter, giảng viên thấy "Chưa trích được. Thử lại."; mất kết nối giữa chừng → câu đã lưu giữ nguyên, việc chạy tiếp theo trang chưa xong.
- AC7 (phân quyền). Given STUDENT, hoặc ADMIN không thuộc lớp When `POST …/questions/extract` Then `403 FORBIDDEN` (`reason: role|course`). TA gọi được (kiểm `Staff`).
- AC8 (riêng tư). Given đề có tên người ra đề / giảng viên When gửi LLM Then bị che; `stem` lưu **sau khi khôi phục** chỉ giữ chữ của đề (không thêm tên).

### Ngoài phạm vi của story này
- Đề dạng ảnh quét không có văn bản (đã trích bởi Docling; nếu rỗng → báo "Không đọc được văn bản").
- Câu `CODE`.

### Phụ thuộc
- US-P9-01 · `FEAT-docs-calendar` (loại tài liệu, chunk, trang) · D46, D47

---

## US-P9-03: Giảng viên / TA muốn sinh câu hỏi mới từ đoạn tài liệu kèm trích dẫn, để làm giàu ngân hàng mà vẫn kiểm được nguồn
Ưu tiên: Could (cắt đầu tiên nếu trễ) · Ước lượng: M · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given tài liệu `LECTURE` `READY` được chọn (hoặc chủ đề) When `POST …/questions/generate {document_ids[] | chunk_ids[], topic, difficulty, count ≤ 10, types[]}` (`Idempotency-Key`) Then `202 {job_id}`; việc BATCH; câu sinh `origin=GENERATED`, `review_status=PENDING`.
- AC2. Given câu sinh When lưu Then **mỗi câu có ≥ 1 trích dẫn thật** `{document_id, chunk_id, page}` mà đoạn đó chứa căn cứ của đáp án đúng (kiểm tất định: từ khoá của `explanation` / đáp án đúng xuất hiện trong chunk); câu không có trích dẫn hợp lệ bị loại và đếm vào "bỏ qua".
- AC3. Given dữ liệu vào prompt Then chỉ gồm chủ đề, độ khó, số câu, nội dung chunk đã chọn (loại `ANSWER_KEY` bị từ chối `422`); qua lớp che danh tính; staff được chọn cả tài liệu `visible_to_students=false` làm nguồn (câu sinh chỉ lộ cho sinh viên sau khi duyệt; Q-P9-9).
- AC4 (nhánh lỗi). Given `count > 10`, `types` rỗng, `document_ids` ngoài lớp Then `422 VALIDATION_FAILED`; lỗi LLM → như US-P9-02 AC6.
- AC5 (không trùng). Given câu mới trùng `stem` chuẩn hoá với câu đã có (mọi trạng thái, kể cả `REJECTED`) When lưu Then bỏ qua ("đã có").
- AC6 (phân quyền). Given STUDENT Then `403`.
- AC7 (số liệu E5). Given giảng viên duyệt xong lô Then API `GET …/questions/gen-stats?job_id=` trả `{nguyên, sửa nhẹ, loại}` (tính từ `version` và `review_status` của câu: `nguyên` = APPROVE không sửa; `sửa nhẹ` = có `PUT` trước APPROVE; `loại` = REJECT) để chủ dự án ghi số liệu E5.

### Ngoài phạm vi của story này
- Sinh câu `CODE` (đã có `suggest` của PE cho test đầu vào).
- Sinh từ ngoài tài liệu (kiến thức chung của mô hình).

### Phụ thuộc
- US-P9-01, US-P9-02 (dùng chung bộ khử trùng) · `internal/exam/suggest.go` (khung job)

---

## US-P9-04: Giảng viên / TA muốn một luồng "Tạo câu hỏi" và duyệt hàng loạt ở `/questions`, để xử lý hàng chục câu chờ duyệt trong vài phút
Ưu tiên: Must · Ước lượng: M · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given `/questions` (đã có từ PE) When mở Then khung nhìn đầu giữ nguyên (bảng + lọc trạng thái / chủ đề / độ khó / loại / nguồn); nút chính duy nhất `Tạo câu hỏi`; **không** có điều khiển sinh AI hiện mặc định (DESIGN §14.17).
- AC2. Given bấm `Tạo câu hỏi` When mở luồng Then ba lựa chọn: `Soạn tay`, `Trích từ đề` (chọn tài liệu `EXAM_PAPER` + tuỳ chọn tài liệu đáp án), `Sinh từ tài liệu` (chỉ hiện nếu US-P9-03 được làm); chọn xong gọi đúng API (US-P9-02, 03); đang chạy hiện tiến độ ngay trong luồng (không modal chặn), kết quả "bỏ qua N câu" mở rộng ra lý do từng câu.
- AC3. Given bảng câu chờ duyệt When chọn nhiều hàng (≤ 50) Then thanh hành động hiện `Duyệt`, `Loại`, và gắn chủ đề / độ khó hàng loạt; `POST …/questions/bulk {action: APPROVE|REJECT|SET_TAGS, ids[], topic?, difficulty?}` trả kết quả **từng id** (thành công / lỗi) — một id lỗi không huỷ cả lô; bảng cập nhật lạc quan + dòng `Hoàn tác` tại chỗ cho `Loại` (đảo ngược được); không toast "Thành công".
- AC4. Given câu `PENDING` When mở Drawer Then sửa tại chỗ (đề, phương án, đáp án đúng, giải thích, chủ đề, độ khó) và `Duyệt` / `Loại` trong cùng Drawer; bản gõ dở không mất khi đóng nhầm (`useAutosaveDraft`); sửa đồng thời → `409 VERSION_CONFLICT` kèm bản hiện tại để chọn.
- AC5. Given câu `origin ∈ {EXTRACTED, GENERATED}` When xem Then hiện nhãn nguồn + trích dẫn (tài liệu, trang) bấm được mở `/library` tại đoạn đó; câu `GENERATED` / `EXTRACTED` **không** duyệt được khi `citations` rỗng (`422 CITATION_REQUIRED`).
- AC6 (phân quyền). Given STUDENT gọi `…/questions*` Then `403`; TA duyệt được; sinh viên **chỉ** thấy câu `APPROVED` qua luyện đề, không bao giờ qua `/questions`.
- AC7 (nhánh lỗi / mạng). Given mất mạng khi duyệt hàng loạt Then các thao tác chờ gửi lại có `Idempotency-Key`, giao diện báo "Chưa gửi được" tại hàng và gửi lại khi có mạng, không gửi trùng.
- AC8 (Hôm nay). Given có câu `PENDING` Then nguồn việc `QUESTION_REVIEW` (đã có, bậc 90) hiển thị số đúng; duyệt hết thì biến mất trong ≤ 5 s.

### Ngoài phạm vi của story này
- Tạo bài thi (PE).
- Nhập câu từ file Forms (US-P9-11).

### Phụ thuộc
- US-P9-01, 02, (03) · `FEAT-weekly-exam` 6.1 (`PUT …/review`), DESIGN §14.17

---

## US-P9-05: Hệ thống muốn dựng đề từ ngân hàng theo ma trận chủ đề × độ khó, không lặp câu và không dùng câu của bài thi chưa đóng, để luyện đề công bằng và không lộ đề thi thật
Ưu tiên: Must · Ước lượng: M · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given ngân hàng ≥ 80 câu `APPROVED` (seed) When dựng đề 20 câu theo ma trận Then trả trong ≤ 3 s (p95 trên 100 lần); số câu đúng ma trận (± 0 khi đủ câu; thiếu câu thì bù từ ô gần nhất theo độ khó rồi chủ đề, ghi vào `notes`); **không lặp câu** trong đề; `go test ./internal/quiz -run TestBuildMatrix`.
- AC2. Given sinh viên đã làm lượt trước When dựng đề mới cùng chủ đề Then ưu tiên câu **chưa làm hoặc làm sai** trong 30 ngày gần nhất; không lặp lại câu vừa làm đúng nếu còn câu khác.
- AC3. Given câu nằm trong `exam_items` của bài thi có trạng thái `DRAFT`, `SCHEDULED`, `OPEN` (chưa `CLOSED`) hoặc đã `CLOSED` nhưng chưa `PUBLISHED` When dựng đề luyện Then câu **bị loại**; câu của bài thi đã `PUBLISHED` được phép (Q-P9-3); câu `PENDING`, `DRAFT`, `REJECTED`, `archived`, `CODE`, `ESSAY` (khi đề không bật tự luận) **không bao giờ** xuất hiện.
- AC4. Given ma trận không thể đạt (ví dụ chủ đề chỉ có 4 câu mà đề xin 10) When dựng Then trả đề ngắn hơn kèm `notes: ["thiếu câu chủ đề X"]`, không lỗi 5xx; ngân hàng < 5 câu đủ điều kiện → `409 NOT_ENOUGH_QUESTIONS` (`{available}`), giao diện báo "Chưa đủ câu để luyện chủ đề này".
- AC5 (xáo). Given hai sinh viên cùng chủ đề When dựng Then thứ tự câu và phương án xáo riêng cho từng lượt, **ổn định** trong lượt (tải lại không đổi thứ tự); id phương án gốc giữ nguyên để chấm.
- AC6 (phân quyền). Given STUDENT lớp X When dựng đề lớp Y Then `403`/`404`; sinh viên chỉ dựng đề từ câu của lớp mình đang theo học `ACTIVE`.
- AC7 (nhánh lỗi). Given `count` ngoài `[5, 50]` hoặc `topic` không có trong ngân hàng Then `422 VALIDATION_FAILED`.

### Ngoài phạm vi của story này
- Điều chỉnh độ khó theo năng lực (không có mô hình năng lực; không gamify).

### Phụ thuộc
- US-P9-01 · `exam_items`, `exam_status` (PE) · PRD M4 AC (≤ 3 s)

---

## US-P9-06: Sinh viên muốn luyện theo chủ đề và được phản hồi ngay từng câu kèm giải thích bám tài liệu, để biết mình sai ở đâu
Ưu tiên: Must · Ước lượng: L · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given sinh viên đã vào lớp When mở `/practice` Then khung nhìn đầu gồm: MỘT khối luyện tập khuyến nghị theo lỗi gần đây (chủ đề + số câu + nút `Luyện ngay`), dưới đó hai lựa chọn `Theo chủ đề` và `Thi thử`, rồi dòng `Tiếp tục` nếu có lượt dở; không số liệu thành tích, không bảng xếp hạng; chưa có dữ liệu lỗi → khối khuyến nghị là chủ đề có nhiều câu nhất.
- AC2. Given chọn `Theo chủ đề` + chủ đề + số câu (mặc định 10) When `POST …/practice/start {mode: TOPIC, topic, count}` (`Idempotency-Key`) Then `201` lượt mới, hoặc `200` lượt `TOPIC` đang dở của cùng chủ đề (làm tiếp); chuyển sang `/practice/[attemptId]` ở **câu đang dở**.
- AC3. Given màn làm bài When hiển thị Then mỗi lần một câu (cả desktop, kèm danh sách số câu thu gọn ở ≥ 1024 px); tiêu đề gọn: tiến độ `3/10` + `Thoát`; không đồng hồ ở luyện theo chủ đề.
- AC4. Given chọn đáp án và bấm `Trả lời` When gửi Then phản hồi ngay (≤ 500 ms p95): đúng / sai, đáp án đúng, giải thích của câu, trích dẫn bấm được mở `/library`; đáp án **đã chốt, không đổi được** sau khi có phản hồi; mất mạng → hàng chờ ghi cục bộ, hiện "Chưa gửi được" và gửi lại khi có mạng (phản hồi hiện sau khi gửi được; không hiện trước).
- AC5 (ExplainAnswer). Given đã trả lời (đúng hoặc sai) When bấm `Giải thích thêm` Then `POST …/practice/attempts/{aid}/items/{itemId}/explain` stream SSE, sự kiện đầu ≤ 300 ms (D47), một lời gọi LLM, làn INTERACTIVE; nội dung chỉ dựa trên đề, đáp án đúng, giải thích và chunk trích dẫn của câu; kiểu Socratic ngắn, tiếng Việt; **chỉ gọi khi đã trả lời** (chưa trả lời → `409 NOT_ANSWERED`); nút bấm lại trong 10 s trả cùng nội dung từ cache theo `(question_id, id đáp án đã chọn, version câu)`, vô hiệu khi câu sửa.
- AC6 (che danh tính). Given payload tới LLM Then không có tên / MSSV / email; `llm_audit` kiểm.
- AC7 (kết thúc). Given trả lời xong câu cuối When `POST …/submit` (`Idempotency-Key`) Then `200` tóm tắt: số câu đúng / tổng, chủ đề yếu nhất, nút `Luyện tiếp chủ đề này` / `Về Luyện đề`; không điểm / xếp hạng / pháo hoa; gọi lại `submit` → cùng kết quả (idempotent).
- AC8 (phân quyền). Given sinh viên B When `GET …/practice/attempts/{id của A}` hoặc `PUT …/answers` Then `404` (không lộ tồn tại); TEACHER / TA / ADMIN gọi API sinh viên (`practice/start`, `answers`) → `403`.
- AC9 (nhánh lỗi). Given LLM `OVERLOADED` hoặc lỗi When bấm `Giải thích thêm` Then hiện "Chưa giải thích được. Thử lại." tại chỗ; phản hồi đúng / sai và giải thích có sẵn của câu **không bị ảnh hưởng** (chấm là code thuần); lượt không bị hỏng.
- AC10 (tải lại). Given đang ở câu 6 When tải lại trang Then quay đúng câu 6 với trạng thái các câu trước (đã trả lời, đúng / sai); không mất lựa chọn chưa gửi ở câu hiện tại nếu đã bấm chọn (lưu nháp phía máy).
- AC11 (rỗng). Given chủ đề đã chọn < 5 câu `APPROVED` Then `409 NOT_ENOUGH_QUESTIONS`; ngân hàng của lớp rỗng → `/practice` hiện "Giảng viên chưa mở câu hỏi" (một dòng, không nút).
- AC12 (mobile). Given 375 px Then làm trọn một lượt 10 câu không cuộn ngang; vùng chạm ≥ 44 px; thanh hành động cố định dưới.

### Ngoài phạm vi của story này
- Thi thử (US-P9-07); tự luận (US-P9-08).
- Gợi ý tài liệu sau lượt (US-P9-09).

### Phụ thuộc
- US-P9-01, 05 · Scheduler INTERACTIVE · `FEAT-private-chat-pii` (che danh tính) · DESIGN §14.18–14.19, UX mục 6

---

## US-P9-07: Sinh viên muốn thi thử có tính giờ, tự lưu từng câu, mất mạng vẫn làm tiếp và nộp khi có mạng, để tập đúng điều kiện thi
Ưu tiên: Must · Ước lượng: L · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given chọn `Thi thử` When `POST …/practice/start {mode: MOCK, matrix?, count (mặc định 20), duration_minutes (mặc định 30)}` (`Idempotency-Key`) Then `201` lượt `MOCK` với `deadline_at` do **máy chủ** tính; còn lượt `MOCK` dở của lớp → `200` làm tiếp (không tạo lượt thứ hai).
- AC2. Given làm bài When hiển thị Then đồng hồ chạy theo máy chủ (`server_now` lệch được bù), chữ số cố định bề rộng (không nhảy layout), **không** hiện phản hồi đúng / sai và không có nút `Giải thích thêm` cho tới khi nộp.
- AC3. Given chọn / đổi đáp án When rời câu hoặc sau 1 s không gõ Then `PUT …/practice/attempts/{aid}/answers/{itemId}` tự lưu (`version`, `X-Practice-Tab`); mở lượt ở tab thứ hai → tab cũ nhận `409 TAB_TAKEN_OVER` và chỉ đọc, kèm nút `Làm tiếp ở tab này`.
- AC4. Given mất mạng When tiếp tục chọn Then mọi lựa chọn vào hàng chờ cục bộ (IndexedDB), giao diện hiện "Chưa gửi được N câu" tại thanh dưới (không chặn làm bài); có mạng → gửi lại theo thứ tự, thành công thì biến mất; hết giờ khi đang mất mạng → bài nộp tự động ngay khi có mạng, các lựa chọn trong hàng chờ **chỉ** được tính nếu có dấu thời gian cục bộ ≤ `deadline_at` + dung sai (mặc định 60 s, Q-P9-5).
- AC5. Given còn ≤ 5 phút When đồng hồ chạm Then hiện cảnh báo một lần, tại chỗ (không modal), màu đỏ theo luật "cần hành động"; đã cảnh báo thì không nhắc lại khi tải lại.
- AC6. Given tải lại trang giữa chừng Then quay đúng câu đang làm (`current_index` lưu máy chủ qua `PATCH …/attempts/{aid} {current_index}`), đồng hồ không đặt lại, mọi lựa chọn đã lưu còn nguyên.
- AC7. Given bấm `Nộp bài` (xác nhận một lần vì **không đảo ngược** — `ConfirmIrreversible`) hoặc hết giờ When `POST …/submit` (`Idempotency-Key`) Then chấm bằng Quiz Engine; trả kết quả: số đúng / tổng, điểm trên thang 10 (`decimal`, 2 chữ số), từng câu đúng / sai kèm đáp án + giải thích + trích dẫn, phân tích theo chủ đề; gửi hai lần → cùng kết quả, không chấm lại.
- AC8 (hết giờ do server). Given `deadline_at` đã qua trên máy chủ When nhận `PUT answers` mới Then `409 ATTEMPT_CLOSED`; lượt tự đóng bởi worker (cron 1 phút) nếu sinh viên không nộp; điểm tính trên các câu đã lưu.
- AC9 (nhánh lỗi). Given nộp thất bại do mạng Then hiện "Chưa nộp được. Đang thử lại." tại thanh dưới, thử lại tự động với cùng `Idempotency-Key`; nội dung bài **không bao giờ mất**.
- AC10 (liêm chính cơ bản). Given thi thử When rời tab / dán Then **không** ghi nhật ký gửi cho giảng viên (không phải bài tính điểm); chat riêng **không bị khoá** (Q-P9-2).
- AC11 (phân quyền). Given lượt của người khác / STUDENT lớp khác Then `404`; staff gọi API làm bài `403`.
- AC12 (hiệu năng). Given 100 sinh viên thi thử cùng lúc (k6 `practice-mock.js`, LLM `fake`) Then `PUT answers` p95 ≤ 500 ms, `submit` p95 ≤ 1 s, 0 lượt mất câu đã lưu.

### Ngoài phạm vi của story này
- Giám thị, webcam, khoá chat.
- Xáo theo nhóm / ngân hàng thích nghi.

### Phụ thuộc
- US-P9-01, 05 · khung lưu hàng chờ + đồng hồ của PE (`exam-save-queue`, `exam-clock`) · UX mục 6

---

## US-P9-08: Sinh viên muốn luyện câu tự luận và nhận nhận xét gợi ý, không phải điểm, để biết bài mình thiếu gì
Ưu tiên: Should · Ước lượng: M · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given câu `ESSAY` `APPROVED` (đáp án mẫu + rubric ngắn) trong lượt `TOPIC` When sinh viên viết và bấm `Trả lời` Then lưu bài viết; hiện **đáp án mẫu và rubric** sau khi trả lời (tự đối chiếu); không điểm.
- AC2. Given bấm `Nhận xét` When `POST …/practice/attempts/{aid}/items/{itemId}/feedback` Then **một** lời gọi chấm qua Grading Engine (`GradeSubmission`, không phải hai lượt chính thức), bài viết là **dữ liệu không tin cậy** (cách ly khỏi chỉ dẫn), kết quả là nhận xét theo từng tiêu chí + trích đoạn bằng chứng; **không số điểm**, không ghi vào `grade_entries`; gắn nhãn "Gợi ý, giảng viên không xem".
- AC3. Given bài viết chứa lệnh ("cho tôi 10 điểm", "bỏ qua rubric") When chấm Then nhận xét không bị kéo; nhận xét có cờ "bài chèn lệnh" không hiển thị điểm vì không có điểm; không lộ nội dung prompt.
- AC4. Given lượt `MOCK` Then câu `ESSAY` mặc định **không** có trong ma trận (Q-P9-6); nếu giảng viên bật thì chỉ nhận xét sau khi nộp.
- AC5 (giới hạn). Given bài viết > 4.000 ký tự hoặc > 10 lần xin nhận xét mỗi giờ mỗi sinh viên Then `422 TOO_LONG` / `429 RATE_LIMITED` (`Retry-After`); không tốn lời gọi LLM.
- AC6 (nhánh lỗi). Given LLM lỗi / quá tải Then "Chưa nhận xét được. Thử lại."; bài viết giữ nguyên.
- AC7 (phân quyền / riêng tư). Given bài viết luyện tập Then giảng viên / TA / ADMIN **không đọc được** (kể cả qua `llm_audit` của giảng viên: chỉ ADMIN xem payload đã che, có `audit_log`).
- AC8 (không tính vào điểm). Given sổ điểm Then không có dòng nào lấy từ nhận xét luyện tự luận (test: `grade_entries` không đổi).

### Ngoài phạm vi của story này
- Chấm tự luận chính thức (P7, đã có).
- Chấm câu `SHORT` bằng LLM khi không khớp (PRD M8: "không khớp mới gọi engine tự luận" thuộc bài QUIZ; luyện đề **không** gọi LLM chấm `SHORT`, sai là sai).

### Phụ thuộc
- US-P9-06 · `internal/grading` · Q-P9-6

---

## US-P9-09: Sinh viên muốn xem lịch sử và điểm yếu theo chủ đề, nhận gợi ý tài liệu và nhắc trên Hôm nay; giảng viên muốn tín hiệu lỗ hổng tổng hợp — mà không ai thấy lượt làm riêng của sinh viên
Ưu tiên: Must · Ước lượng: M · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given sinh viên có lượt When mở `/practice/history` Then hàng theo thời gian (ngày, loại `Theo chủ đề` / `Thi thử`, chủ đề, số đúng / tổng); phân trang con trỏ; bấm một hàng xem lại từng câu (đúng / sai, đáp án, giải thích); không huy hiệu, không biểu đồ thành tích.
- AC2. Given đủ dữ liệu (≥ 10 câu đã làm) When mở lịch sử Then khối "Chủ đề cần ôn" liệt kê tối đa 3 chủ đề có tỉ lệ sai cao nhất (≥ 3 câu đã làm / chủ đề), mỗi chủ đề kèm tối đa 2 tài liệu gợi ý (từ `citations` của các câu sai, xếp theo số câu sai trỏ tới) bấm mở `/library`; chưa đủ dữ liệu → khối ẩn.
- AC3. Given sinh viên có ≥ 1 chủ đề yếu When mở `/` (Hôm nay) Then hiện việc `PRACTICE_WEAK_TOPIC` (bậc 85, chỉ STUDENT): "Ôn chủ đề {tên}" liên kết `/practice?topic=…`; chỉ hiện nếu chưa luyện chủ đề đó trong 3 ngày; biến mất ≤ 5 s sau khi luyện xong.
- AC4. Given dữ liệu luyện đề When tổng hợp cho giảng viên Then `internal/practice` công khai hàm `WrongAnswerSignals(courseID, from, to)` trả **theo chủ đề**: `{topic, wrong_answers, students_affected, sample_question_ids}`; chủ đề có `students_affected < 3` bị **loại khỏi kết quả** (k-ẩn danh); **không** có `student_id`, tên, MSSV trong đầu ra (`TestWrongAnswerSignalsNoIdentity`). Hàm này là đầu vào cho báo cáo lỗ hổng của P10 (không có UI cho giảng viên ở sprint này).
- AC5 (riêng tư). Given TEACHER / TA / ADMIN When gọi mọi API `…/practice/*` của sinh viên (lịch sử, lượt, câu trả lời, điểm yếu) Then `403`; Hồ sơ 360 **không** hiển thị dữ liệu luyện đề của từng sinh viên (Q-P9-1); `activity` heartbeat `PRACTICE` (đã có, P5) giữ nguyên và tuân công tắc đo theo lớp.
- AC6 (xoá). Given sinh viên rời lớp / lớp lưu trữ Then lượt làm của lớp đó không còn hiện trong `/practice/history` của lớp khác; không rò chéo lớp (`course_id`).
- AC7 (nhánh lỗi / rỗng). Given chưa có lượt Then `/practice/history` hiện "Chưa có lượt luyện" + nút `Luyện ngay`; lỗi tải → `PageState` lỗi có nút `Thử lại`.
- AC8 (hiệu năng). Given sinh viên 200 lượt Then `GET …/practice/history?limit=20` p95 ≤ 300 ms (chỉ mục `(course_id, student_id, started_at DESC, id DESC)`).

### Ngoài phạm vi của story này
- Giao diện giảng viên xem lỗ hổng (P10 `/insights`).
- Gợi ý tài liệu bằng embedding.

### Phụ thuộc
- US-P9-06, 07 · `today.Aggregator.Register` + `allowed` (như các phase trước) · `FEAT-crm-360` 4.6 · Q-P9-1

---

## US-P9-10: Giảng viên muốn luyện đề không phá bài thi tính điểm (không lộ đề, không lộ đáp án, chat bị khoá đúng lúc), để kết quả thi đáng tin
Ưu tiên: Must · Ước lượng: M · Sprint: 10

### Tiêu chí nghiệm thu
- AC1 (không rò từ luyện đề). Given bài thi `OPEN` có câu X When sinh viên dựng lượt luyện (bất kỳ chủ đề) Then X **không** xuất hiện ở đề, ở phản hồi, ở `explain`; test `TestPracticeNoExamLeak` chạy 6 trạng thái bài thi (`DRAFT`, `SCHEDULED`, `OPEN`, `CLOSED` chưa công bố, `PUBLISHED`, đã xoá mềm) và xác nhận X chỉ xuất hiện khi `PUBLISHED` (Q-P9-3).
- AC2 (không rò đáp án). Given mọi endpoint của sinh viên trong `practice/*` When quét bằng `TestPracticeNoAnswerLeak` ở các trạng thái của lượt (`TOPIC` chưa trả lời câu, đã trả lời câu, đã nộp; `MOCK` đang làm, đã nộp) Then `answer_key`, `correct`, `explanation`, `citations` chỉ xuất hiện đúng lúc quy định ở US-P9-01 AC5; ở `MOCK` đang làm: không có trong **mọi** phản hồi (kể cả lỗi, kể cả `GET attempt` khi tải lại).
- AC3 (khoá chat bài thi tính điểm — F12). Given sinh viên đang có lượt bài thi PE `IN_PROGRESS` When hỏi chat riêng nội dung môn học Then chat từ chối (đã có từ PE / P3), hết giờ → chat mở lại; spec E2E `quiz-integrity.spec.ts` chứng minh cả hai chiều và kiểm trên **cùng** màn làm bài với một bài QUIZ do giảng viên giao.
- AC4 (`search_library` / tài liệu đề). Given sinh viên đang có lượt bài thi PE `IN_PROGRESS` When agent gọi `search_library` Then **không trả** đoạn thuộc tài liệu loại `EXAM_PAPER` hay `ANSWER_KEY` (kiểm bằng test tool); ngoài giờ thi, `EXAM_PAPER` vẫn tìm được như hiện tại (`FEAT-docs-calendar` bảng `audience`) (Q-P9-4).
- AC5 (luyện đề không khoá chat). Given sinh viên chỉ đang có lượt luyện `TOPIC` / `MOCK` Then chat **không** bị khoá; thi thử đang làm không làm đổi `GET /me/exam-lock`.
- AC6 (ghi nhận). Given thi thử When hết giờ Then không có sự kiện nào trong `exam_events` (vì không phải bài thi).
- AC7 (lộ đáp án qua `explain`). Given lượt `MOCK` chưa nộp When gọi `explain` Then `409 NOT_SUBMITTED`; không có đường nào khác trả giải thích.
- AC8 (phân quyền chéo lớp). Given sinh viên lớp X (có lượt) Then không đọc câu / trích dẫn của lớp Y; `question_id` của lớp khác trong `answers` → `422 INVALID_QUESTION`.

### Ngoài phạm vi của story này
- Giám thị, so mã giống (PE).

### Phụ thuộc
- US-P9-05, 06, 07 · `FEAT-weekly-exam` (Locker, `TestNoAnswerLeak`) · `FEAT-docs-calendar` (`audience`) · F12

---

## US-P9-11: Giảng viên muốn nhập kết quả Microsoft / Google Forms vào sổ điểm, khớp sinh viên theo email hoặc MSSV trong danh sách lớp, để dùng bài trắc nghiệm đã làm ngoài hệ thống
Ưu tiên: Could (cắt đầu tiên nếu trễ) · Ước lượng: L · Sprint: 10

### Tiêu chí nghiệm thu
- AC1. Given tệp XLSX / CSV ≤ 5 MB từ Forms When tệp lên qua `uploads/presign` `purpose=forms_import` (cùng cơ chế `grade_import` của sổ điểm) rồi `POST …/forms-import/preview {upload_id}` Then trả 20 dòng đầu + cột phát hiện; giảng viên ánh xạ cột ↔ câu hỏi của ngân hàng (hoặc ↔ "đáp án chuẩn" nhập tay) và cột định danh (`Email` / `MSSV`); chưa ghi gì.
- AC2. Given ánh xạ xong When `POST …/forms-import {mapping, …}` (`Idempotency-Key`) Then chấm từng dòng bằng Quiz Engine (**không LLM**); mỗi sinh viên một kết quả; dòng không khớp được sinh viên vào danh sách "Chưa khớp" gán tay; ghi điểm vào một `grade_item` thủ công (`source MANUAL`) mà giảng viên chọn (`grade_entries.source = 'QUIZ'`) theo quy trình sổ điểm hiện có (chưa công bố mặc định).
- AC3. Given khớp sinh viên Then chỉ theo **email đã xác minh** hoặc **MSSV có trong danh sách lớp** (luật: MSSV tự khai không mở dữ liệu; không tạo tài khoản; không nối tài khoản vào danh sách lớp theo MSSV tự khai); email lệch MSSV → hàng "Chưa khớp".
- AC4. Given dòng trùng một sinh viên When nhập Then lấy dòng **mới nhất theo dấu thời gian** của Forms; ghi chú số dòng trùng; chạy lại cùng tệp (cùng hash) → idempotent, không đổi điểm.
- AC5. Given giảng viên bấm `Xem trước kết quả` Then thấy bảng `{sinh viên, điểm, đúng/tổng}` + số dòng chưa khớp trước khi ghi; điểm tính `decimal`.
- AC6 (nhánh lỗi). Given tệp sai định dạng / > 5 MB / thiếu cột định danh Then `422` kèm vị trí; đáp án trong tệp không khớp lựa chọn của câu → dòng đó `0` và ghi "không khớp" (không đoán).
- AC7 (phân quyền). Given STUDENT / TA Then `403` (chỉ TEACHER ghi vào sổ điểm); ADMIN `403`.
- AC8 (riêng tư). Given tệp gốc Then lưu qua `platform/blob` tối đa 90 ngày (mặc định, Q-P9-7), có `audit_log`; không vào log; không gửi tới LLM.
- AC9 (sổ điểm khoá). Given sổ điểm `LOCKED` Then `409` (đi qua "mở khoá có lý do" của P6).

### Ngoài phạm vi của story này
- Import bài tự luận / tải bài (P7).
- Tạo `exams` từ tệp Forms (không dùng `exams` của PE; xem Q-P9-8).

### Phụ thuộc
- `FEAT-gradebook` (`grade_entries.source = QUIZ`, `grade_items` MANUAL, khoá) · `FEAT-assignments` (đã chốt không làm Forms ở P7, Q8) · `platform/blob` · Q-P9-8

---

## US-P9-12: Đội muốn seed, cổng nghiệm thu và E2E của P9, và xoá mock luyện đề, để F12 đi trọn và phase đóng
Ưu tiên: Must · Ước lượng: M · Sprint: 10

### Tiêu chí nghiệm thu
- AC1 (seed). Given stack trống When `node scripts/seed.mjs` Then ngân hàng lớp 1 có **80 câu `APPROVED`** (chủ đề ≥ 6, đủ ba độ khó, có `citations` thật trỏ tài liệu seed), **20 câu `PENDING`** (có cả `EXTRACTED`), 1 bài QUIZ (bài thi PE đã `PUBLISHED`), ≈ **40 lượt luyện đề** của các sinh viên mô phỏng với ≥ 3 sinh viên sai cùng 2 chủ đề (đủ cho k-ẩn danh); lớp 2 có ngân hàng nhỏ khác (không chứa câu lớp 1); `scripts/check-practice-seed.mjs` xanh; chạy lại idempotent.
- AC2 (cổng). Given repo When chạy:
  - `cd backend-go && go test -race ./internal/quiz/...`
  - `cd backend-go && go test ./internal/practice -run 'TestExtractQuestions|TestBuildMatrix|TestPracticeNoAnswerLeak|TestPracticeNoExamLeak|TestWrongAnswerSignalsNoIdentity' -v`
  - `pnpm -C frontend exec playwright test quiz-integrity.spec.ts practice.spec.ts`
  Then tất cả xanh.
- AC3 (E2E `practice.spec.ts`). Given sinh viên mô phỏng When làm thi thử 20 câu, **tải lại trang giữa chừng** (câu 8) và tắt / bật mạng giữa chừng (câu 12–14) rồi nộp Then kết quả đúng với đáp án tính tay trong spec; luyện theo chủ đề 10 câu có phản hồi ngay; nhánh lỗi: hết giờ trên máy chủ → `ATTEMPT_CLOSED`, bài vẫn chấm; STUDENT B không mở lượt của A.
- AC4 (xoá mock). Given `frontend/src/mock/practice.ts`, `features/practice/*` Then không còn import `@/mock/practice` trong `src/` (`grep` = 0); `mock/questions.ts` nếu còn thì xoá cùng đợt nếu không còn người dùng; `bash scripts/ui-antipatterns.sh` sạch; ảnh mốc `practice` (nếu có) cập nhật.
- AC5 (hợp đồng). Given `openapi.yaml` Then có mọi endpoint mới; `go test ./internal/contract/...` xanh (response khớp spec).
- AC6 (luồng F12). Given FLOWS F12 Then đi trọn từ "trích câu hỏi → duyệt → luyện → phân tích điểm yếu → Hôm nay gợi ý ôn" trên stack thật, có nhánh lỗi (trích lỗi, mất mạng khi nộp); `docs/PROGRESS.md`: F12 ☑, P9 ghi "xong" khi cổng đạt.
- AC7 (hiệu năng giao diện). Given `/practice` và `/practice/[id]` Then LCP ≤ 2.500 ms (Lighthouse devtools), TBT ≤ 200 ms (mô phỏng), script ≤ 256 KB; lỗi trượt → ghi `proposals.md`, không nới ngưỡng.
- AC8 (tài liệu). Given xong Then `ARCHITECTURE.md` §4 / §5 khớp (số migration thật, route thật); `docs/DEMO_SCRIPT.md` thêm một bước ngắn luyện đề nếu PM đồng ý.

### Ngoài phạm vi của story này
- `make eval` E5 chính thức (P10); chỉ cung cấp số liệu thô qua `gen-stats`.

### Phụ thuộc
- US-P9-01…10 (US-P9-03, 11 nếu làm) · `scripts/seed.mjs` · `docs/sprints/9.5` (số đo) không phải điều kiện
