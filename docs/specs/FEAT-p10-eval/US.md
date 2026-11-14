# FEAT-p10-eval Observation A + B, đánh giá, hoàn thiện
Nguồn: PRD M13 (A, B), M14 (nguồn việc), E1–E6 (PRD §Đánh giá) · FLOWS F14, F16, F17 · `docs/phases/P10.md` L1–L4 · `docs/sprints/11/plan.md` đợt 1–5 · `docs/ARCHITECTURE.md` §4 (`00002`, `00015`), §5 (Insight, Observability), §6 · `docs/design/DESIGN.md` §13, §14.21, §14.22 · `docs/UX.md` mục 6 · Nợ `docs/PROGRESS.md` (UX17-8/9, LCP, L6)
Phase: P10, sprint 11 (đợt 1–6; đợt 6 = P-11-10). Trạng thái: **APPROVED** v1.0 (2026-11-14; PM uỷ quyền; câu không đụng tính điểm / auth / PII: **PM chốt theo đề xuất BA** — `QUESTIONS.md`).

Phạm vi tải / chaos T1 (`seed-t1`, k6, `make load-t1`, `make chaos`) đã làm ở sprint 9.5; **không** lặp ở đây (Q-E-20).

Mã vai: ADMIN, TEACHER, TA, STUDENT. «Lớp» = `course_id` của đường dẫn.

---

## US-P10-01: Admin muốn thấy sức khoẻ hệ thống AI trong một dải gọn và bảng yêu cầu có lọc, để biết có gì bất thường mà không đọc nội dung ai hỏi
Ưu tiên: Must · Ước lượng: L · Sprint: 11 · **Đợt 1**

### Tiêu chí nghiệm thu
- AC1 (số thật). Given `/observability` Then mọi số lấy từ API thật (`llm_audit`, `escalation_tickets`, `chat_messages.feedback`, `pii_events`, điểm AI bị giảng viên sửa); **không còn import `@/mock/system`** trong `src/features/observability` (`grep` = 0).
- AC2 (dải trạng thái). Given Admin mở `/observability` Then khung nhìn đầu là MỘT dòng chữ «24 giờ qua»: số yêu cầu · p95 · tỷ lệ lỗi · tỷ lệ chuyển nhà cung cấp dự phòng · chi phí ngày (VND). Không thẻ KPI; mỗi chỉ số đúng một chỗ ở khung nhìn đầu (kế thừa US-UX91-17 AC3).
- AC3 (bảng 7 ngày). Given cùng màn Then bảng theo ngày × chỉ số (yêu cầu, p95, lỗi, chuyển giảng viên, lần che thông tin, chi phí) và bảng chi phí theo tác vụ; hàng đầu không trống. Biểu đồ chuỗi thời gian **chỉ** khi xu hướng có nghĩa (≥ 7 ngày dữ liệu).
- AC4 (bảng yêu cầu). Given Admin When lọc `Tất cả` / `Có vấn đề` (status ≠ `ok` hoặc `degraded`) / `Thông tin cá nhân` (`pii_masked_count > 0`), theo lớp, tác vụ, khoảng giờ Then bảng phân trang con trỏ (`limit` ≤ 100), mỗi hàng: giờ, tác vụ, nhà cung cấp / mô hình, độ trễ, trạng thái, lớp; **không có cột nội dung**.
- AC5 (TEACHER). Given Teacher mở `/observability` Then chỉ thấy bản tổng hợp của MỘT lớp mình phụ trách (số yêu cầu, tỷ lệ lỗi, chuyển giảng viên, lần che thông tin, Helpful / Not helpful, tỷ lệ sửa điểm AI); không bảng yêu cầu, không nội dung, không chi phí theo nhà cung cấp. `/admin/observability/*` → `403`.
- AC6 (phân quyền). Given TA / STUDENT When `GET …/observability/summary` hoặc `/admin/observability/*` Then `403`; Teacher lớp khác → `404` (không lộ tồn tại lớp); Admin xem tổng hợp lớp nào cũng được.
- AC7 (nhánh rỗng / lỗi). Given chưa có yêu cầu nào Then `EmptyState` một dòng, không bảng; API lỗi → `<PageState>` lỗi + `Thử lại`; thư gửi lỗi (`FailedMail`) vẫn hiện như prototype 9.1 và lấy từ `mail_outbox` thật.
- AC8 (hiệu năng). Given bảng `llm_audit` ≥ 1 triệu hàng (T1) Then `summary` p95 ≤ 500 ms, trang bảng ≤ 300 ms (dùng chỉ mục `created_at`, `course_id, created_at`; không `OFFSET`).
- AC9 (di động). Given 375 px Then dải thành các dòng xếp dọc, bảng cuộn trong khung của nó, không tràn trang.

### Ngoài phạm vi của story này
- Chi tiết một yêu cầu (US-P10-02); cảnh báo (US-P10-03); đồ thị Analytics.

### Phụ thuộc
- `llm_audit` (`00002`), US-UX91-17 (bố cục), `FEAT-llm-gateway`.

---

## US-P10-02: Admin muốn mở chi tiết một yêu cầu (prompt đã che, công cụ, độ tự tin) kèm lý do và dấu vết kiểm toán, để điều tra lỗi mà không phá cam kết riêng tư
Ưu tiên: Must · Ước lượng: L · Sprint: 11 · **Đợt 1** · Trả Nợ UX17-8

### Tiêu chí nghiệm thu
- AC1 (khoá mặc định). Given Admin chọn một hàng Then mở `Drawer` ở trạng thái **khoá**: hiện metadata (đã có ở bảng) + ô lý do; **chưa** gọi API chi tiết, chưa có nội dung.
- AC2 (lý do). Given Drawer khoá When Admin nhập lý do 10–200 ký tự và bấm `Xem chi tiết` Then `POST /admin/observability/requests/{id}/open {reason}` trả `200` kèm prompt **đã che**, công cụ đã gọi, độ tự tin, `trace_id`; **cùng giao dịch** ghi `audit_log(entity='llm_request', entity_id, action='VIEW_DETAIL', actor_id, course_id, after={reason}, trace_id)`.
- AC3 (thiếu lý do). Given lý do < 10 ký tự hoặc thiếu When gọi `POST …/open` Then `422 REASON_REQUIRED`, không trả nội dung, không ghi `audit_log` mở; ở giao diện nút khoá và báo lỗi tại ô sau khi rời ô (kế thừa prototype 9.1).
- AC4 (mỗi lần một dòng). Given Admin mở cùng yêu cầu hai lần Then `audit_log` có **2** dòng (không gộp, không bỏ).
- AC5 (không lộ PII). Given nội dung trả về Then không chứa tên / MSSV / email / số điện thoại của người dùng: chỉ placeholder (`[STUDENT_1]`…); `TestObservabilityDetailNoPII` kiểm trên dữ liệu seed có PII.
- AC6 (phân quyền). Given Teacher / TA / STUDENT When gọi `POST /admin/observability/requests/{id}/open` Then `403` và **không** ghi `audit_log` mở; Teacher dùng bản tổng hợp của lớp thì không bao giờ có nội dung (AC5 US-P10-01).
- AC7 (Jaeger). Given `JAEGER_BASE_URL` đã cấu hình When mở chi tiết Then có liên kết `Mở trace` tới `{JAEGER_BASE_URL}/trace/{trace_id}`; chưa cấu hình → hiện `trace_id` kèm nút sao chép, không liên kết gãy.
- AC8 (hết hạn lưu). Given yêu cầu cũ hơn 30 ngày (đã dọn payload) When mở Then `410 PAYLOAD_EXPIRED`; Drawer hiện «Nội dung đã hết hạn lưu» + metadata; vẫn ghi `audit_log` lần mở.
- AC9 (nhánh lỗi). Given `id` không tồn tại → `404`; DB lỗi → `500`, **không** ghi `audit_log` thành công giả; mất `audit_log` thì không trả nội dung (ghi trước, trả sau, cùng giao dịch).
- AC10 (ghi payload an toàn). Given gateway gọi LLM Then chỉ ghi **bản đã che** vào `llm_request_payloads`; test `TestPayloadStoresMaskedOnly`: gọi với tên + MSSV thật → bảng không chứa chuỗi gốc (kiểm cả trường `tools`).
- AC11 (kiểm tra UX17-8). Given Admin mở một hàng trên stack thật Then thấy đúng một hộp lý do → nội dung → có dòng `audit_log` đúng `actor_id`, `entity_id`, lý do (đối chiếu qua `GET /admin/audit`).

### Ngoài phạm vi của story này
- Cho Teacher đọc nội dung (cấm, AGENTS.md); xuất hàng loạt prompt.

### Phụ thuộc
- US-P10-01; migration bảng `llm_request_payloads` (**đề xuất đổi ARCHITECTURE**, Q-E-1); `GET /admin/audit` (PR).

---

## US-P10-03: Admin muốn được báo khi tỷ lệ lỗi LLM cao hoặc chi phí ngày chạm hạn mức, để xử lý trước khi lớp bị ảnh hưởng
Ưu tiên: Must · Ước lượng: M · Sprint: 11 · **Đợt 1**

### Tiêu chí nghiệm thu
- AC1 (lỗi). Given trong **1 giờ** gần nhất có ≥ 20 yêu cầu và tỷ lệ lỗi (status ∉ {`ok`, `degraded`, `cancelled`}) > 5% Then có **một** thông báo `LLM_ERROR_RATE` cho mọi ADMIN (chuông), liên kết `/observability` đã lọc `Có vấn đề`.
- AC2 (chi phí). Given tổng `cost_est` trong ngày (múi giờ `Asia/Ho_Chi_Minh`) ≥ 100% `daily_limit` của `llm_budgets` toàn hệ thống (hoặc của một lớp) Then thông báo `LLM_BUDGET_REACHED` cho ADMIN, nêu phạm vi (toàn hệ thống / tên lớp).
- AC3 (không dội). Given điều kiện còn đúng Then cùng loại + cùng phạm vi **không** thông báo lại trong 6 giờ; hết điều kiện rồi tái phát sau 6 giờ → thông báo mới.
- AC4 (ngưỡng thấp). Given < 20 yêu cầu trong giờ (kể cả lỗi 100%) Then không thông báo lỗi (tránh báo giả lúc ít tải).
- AC5 (không ảnh hưởng đường chính). Given worker tính cảnh báo chậm hoặc lỗi Then chat / chấm bài không bị chậm thêm; cảnh báo chạy ở worker theo lịch (mỗi 5 phút), idempotent (chạy lại cùng cửa sổ không tạo thêm thông báo).
- AC6 (phân quyền). Given Teacher / TA / STUDENT Then không nhận thông báo loại này (`GET /notifications` không có); chỉ ADMIN.
- AC7 (kiểm bằng seed). Given seed có 30 yêu cầu trong giờ, 3 lỗi (10%) When worker chạy Then có 1 thông báo; chạy lại → vẫn 1.

### Ngoài phạm vi của story này
- Gửi mail / Teams cho cảnh báo; ngưỡng tuỳ chỉnh theo giao diện (cố định ở cấu hình).

### Phụ thuộc
- US-P10-01; `notifications` + `GET /api/v1/events` (P4); `llm_budgets` (P1).

---

## US-P10-04: Giảng viên muốn bấm «Tạo báo cáo mới» để biết sinh viên của lớp đang vướng chủ đề nào, để đổi cái mình dạy
Ưu tiên: Must · Ước lượng: L · Sprint: 11 · **Đợt 2** · F17

### Tiêu chí nghiệm thu
- AC1 (tạo). Given Teacher / TA lớp X When `POST …/insights/knowledge-gap/generate {period_days}` (`Idempotency-Key`) Then `202 {job_id}`; tiến độ qua SSE (các giai đoạn: `COLLECT`, `CLUSTER`, `NAME`, `DONE`); khi xong có bản ghi `insight_reports(kind='KNOWLEDGE_GAP')` của lớp X.
- AC2 (năm nguồn). Given lớp có dữ liệu Then báo cáo gom đủ: chat riêng (**đã che**), Threads, escalation, câu sai luyện đề (`practice` — tín hiệu tổng hợp, P-10-3), câu hỏi `no_context` (RAG không tìm được ngữ cảnh). Nguồn thiếu (lớp chưa dùng luyện đề) → bỏ qua tín hiệu đó, không lỗi.
- AC3 (xếp hạng bằng code). Given các cụm Then thứ hạng tính bằng code thuần từ 5 tín hiệu (số câu hỏi; số sinh viên khác nhau; tỷ lệ tự tin thấp / escalate; tỷ lệ Not helpful; tỷ lệ sai luyện đề) theo trọng số cố định (SRS 4); LLM chỉ **đặt tên chủ đề + khuyến nghị** qua `Structured(json_schema)`, không chấm điểm / xếp hạng.
- AC4 (đúng seed). Given seed lớp 1 (≈ 150 câu hỏi lệch chủ đề A, B + 10 câu ngoài tài liệu) When tạo báo cáo Then **đúng A và B đứng đầu** (thứ hạng 1–2), mục «Tài liệu chưa đề cập» có 10 câu `no_context`; `go test ./internal/insight -run TestKnowledgeGap` xanh.
- AC5 (k ≥ 3). Given chủ đề có < 3 sinh viên khác nhau hỏi Then chủ đề vẫn xếp hạng nhưng `examples = []` và nhãn «Ít người hỏi»; chủ đề ≥ 3 có 3–5 câu mẫu.
- AC6 (câu mẫu sạch). Given câu mẫu Then đã che + ẩn danh: không tên, MSSV, email, số điện thoại, không «tôi là …»; mỗi câu mẫu từ sinh viên khác nhau khi có thể; test seed có PII khẳng định 0 rò rỉ.
- AC7 (cách ly lớp). Given báo cáo lớp 1 Then không chứa câu hỏi / chủ đề nào của lớp 2; seed lớp 2 (≈ 40 câu lệch chủ đề C) → chủ đề C đứng đầu ở lớp 2 và **không** xuất hiện ở lớp 1.
- AC8 (data_watermark). Given không có dữ liệu mới từ báo cáo gần nhất với cùng `period_days` Then `generate` trả `200 {report_id, reused:true}`, **không** gọi LLM (`llm_audit` không thêm dòng), không tạo bản ghi mới; có dữ liệu mới → bản ghi mới với `data_watermark` mới.
- AC9 (quá ít dữ liệu). Given < 20 câu hỏi trong kỳ Then `409 NOT_ENOUGH_DATA` `{count}`; giao diện nói «Chưa đủ câu hỏi để gom (cần ≥ 20)».
- AC10 (LLM lỗi). Given LLM `OVERLOADED` / lỗi lúc đặt tên Then báo cáo vẫn tạo với tên tạm từ từ khoá cụm (đánh dấu `named_by:"fallback"`); khuyến nghị trống; việc **không** treo; giao diện báo «Chưa đặt được tên chủ đề». Không có báo cáo nửa vời (rollback nếu lỗi trước `NAME`).
- AC11 (trần). Given lớp đã có 3 việc `generate` trong 60 phút Then `429 RATE_LIMITED` + `Retry-After`; làn LLM `BATCH`.
- AC12 (phân quyền). Given STUDENT → `403`; Teacher / TA lớp khác → `404`; ADMIN gọi `generate` → `403` (chỉ staff lớp tạo); `GET …/latest` / `export`: Teacher, TA lớp đó; ADMIN `403` (báo cáo có câu mẫu).
- AC13 (riêng tư). Given payload gửi LLM Then không có tên / MSSV / email; `llm_audit.pii_masked_count` khớp; không có `student_id` trong `payload` của báo cáo (chỉ số đếm).

### Ngoài phạm vi của story này
- Màn `/insights` (US-P10-05); thống kê điểm thi (US-P10-06).

### Phụ thuộc
- `insight_reports` (`00015`), embedding (`internal/rag`), `internal/privacy`, sprint 10 (câu sai luyện đề), jobs (P1).

---

## US-P10-05: Giảng viên muốn `/insights` hiện báo cáo như một danh sách biên tập kèm hành động, để quyết định dạy lại gì
Ưu tiên: Must · Ước lượng: L · Sprint: 11 · **Đợt 2** · Trả Nợ UX17-9

### Tiêu chí nghiệm thu
- AC1 (số thật). Given `/insights` Then đọc `GET …/insights/knowledge-gap/latest`; **không còn import `@/mock`** trong `src/features/insights` (`grep` = 0).
- AC2 (lớp chưa có báo cáo — UX17-9). Given lớp chưa có báo cáo (API `404`) Then `EmptyState` + đúng một nút `Tạo báo cáo mới`; **không** vẽ báo cáo mô phỏng. Kiểm bằng lớp 761987 và một lớp trống mới tạo.
- AC3 (khung nhìn đầu). Given có báo cáo Then đầu trang: ngày báo cáo + `Tạo báo cáo mới` (chỉ nổi bật khi có dữ liệu mới: API trả `has_new_data`), dưới là danh sách chủ đề xếp hạng; mỗi chủ đề: tiêu đề, một câu «vì sao xếp cao», tín hiệu chính, mục gập «Câu hỏi mẫu» (3–5 câu; không mục gập nếu «Ít người hỏi»), một dòng khuyến nghị dạy. Không thẻ điểm, không biểu đồ ở khung nhìn đầu.
- AC4 (đang tạo). Given bấm `Tạo báo cáo mới` Then chỉ báo giai đoạn («Đang nhóm 66 câu hỏi…»), rời trang được, báo cáo cũ vẫn xem được; xong có thông báo trong chuông (UX.md §5).
- AC5 (không dữ liệu mới). Given `reused:true` Then dòng một dòng «Chưa có câu hỏi mới từ ngày …»; không tạo bản ghi.
- AC6 (Tài liệu chưa đề cập). Given có câu `no_context` Then một Panel «Tài liệu chưa đề cập» liệt kê câu (đã che) kèm số lần.
- AC7 (so sánh). Given có ≥ 2 báo cáo Then cuối trang một Panel «So với báo cáo ngày …»: chủ đề mới / giảm / tăng hạng; một Panel mỗi khối, ngăn bằng đường kẻ 1 px (US-UX91-17 AC1).
- AC8 (tạo thread ghim). Given chủ đề + `Tạo thread ghim` Then mở nháp thread (tiêu đề + nội dung từ khuyến nghị, giảng viên sửa được) → bấm `Đăng` tạo thread **ghim** (`pinned_at`) tác giả = giảng viên, nội dung không chứa câu hỏi mẫu; thread xuất hiện ở `/threads` lớp đó. Không tự đăng không qua giảng viên.
- AC9 (buổi ôn tập). Given `Tạo buổi ôn tập` Then mở form sự kiện lịch (tiêu đề «Ôn tập: <chủ đề>», ngày giờ do giảng viên chọn) → lưu tạo `calendar_events` (type `OTHER`) của lớp; sinh viên thấy ở `/calendar`.
- AC10 (xuất). Given `Xuất` When chọn Markdown Then tải `GET …/insights/knowledge-gap/{id}/export?format=md` (có câu mẫu đã che); chọn PDF → in trang bằng định dạng in của trình duyệt (không thư viện mới, Q-E-8).
- AC11 (thông báo quyền riêng tư). Given trang Then một dòng `meta` «Câu hỏi đã ẩn danh, dùng để cải thiện giảng dạy» (F17); không thêm đoạn giải thích khác (luật chữ UI).
- AC12 (phân quyền). Given STUDENT mở `/insights` Then `403`, mục không có trong điều hướng; ADMIN `403`; TA và TEACHER của lớp có cùng quyền tạo báo cáo, tạo thread ghim, tạo buổi ôn tập (SRS 2).
- AC13 (nhánh lỗi). Given API lỗi → `<PageState>` lỗi + `Thử lại`; tạo thread thất bại → giữ nháp, báo tại chỗ; không mất chữ đã sửa (`useAutosaveDraft`).
- AC14 (di động). Given 375 px Then danh sách xếp dọc, mục gập đủ vùng chạm ≥ 44 px, không tràn ngang.
- AC15 (kiểm tra UX17-8/9 cùng lượt). Given QC chạy TC UX17-8, UX17-9 trên stack thật Then PASS (không còn «KHÔNG KIỂM ĐƯỢC»).

### Ngoài phạm vi của story này
- Biểu đồ phân tích sâu; gửi báo cáo qua mail.

### Phụ thuộc
- US-P10-04; US-UX91-17; `/threads` (P4), `/calendar` (P8).

---

## US-P10-06: Giảng viên muốn thấy thống kê điểm thi hằng tuần của lớp ở `/analytics`, để biết câu nào, tuần nào sinh viên yếu
Ưu tiên: Must · Ước lượng: M · Sprint: 11 · **Đợt 2** · chủ dự án yêu cầu 2026-10-09

### Tiêu chí nghiệm thu
- AC1 (khung nhìn đầu). Given Teacher / TA mở `/analytics`, tab `Điểm thi` Then chọn bài thi (mặc định bài `PUBLISHED` mới nhất) hiện: phân bố điểm (histogram recharts), trung bình / trung vị / tỉ lệ đạt (một dòng chữ, không thẻ), xu hướng qua các bài thi (đường), bảng «câu sai nhiều nhất» (trắc nghiệm + bài code).
- AC2 (nguồn). Given số liệu Then từ `exam_attempts` / `exam_answers` / `code_submissions` của bài thi PE đã công bố; **chỉ bài `PUBLISHED`** (điểm chưa công bố không vào thống kê); điểm tính lại bằng code thuần, không LLM.
- AC3 (theo tuần). Given Teacher chọn «Theo tuần» Then gộp các bài thi trong tuần; tuần không có bài thi → «Chưa có bài thi».
- AC4 (sai nhiều → lỗ hổng). Given một câu sai nhiều gắn `topic` trùng một chủ đề ở báo cáo `/insights` Then hàng có liên kết `Xem ở Báo cáo lỗ hổng`; không có báo cáo → không liên kết.
- AC5 (tụt điểm liên tiếp). Given sinh viên có điểm giảm ở **3 bài thi liên tiếp** Then hiện trong danh sách «Điểm thi đang giảm» (tên, 3 điểm, liên kết hồ sơ 360); chỉ staff lớp thấy; **không** đổi điểm rủi ro `student_risk` (chỉ gợi ý, Q-E-10).
- AC6 (so sánh lớp). Given Teacher phụ trách ≥ 2 lớp Then bảng so sánh các lớp mình dạy: trung bình, trung vị, tỉ lệ đạt, số bài; mỗi lớp cần ≥ 5 bài nộp mới hiện số (dưới ngưỡng → «Chưa đủ»). TA chỉ thấy lớp mình được gán; không thấy so sánh lớp khác.
- AC7 (phân quyền). Given ADMIN When gọi API thống kê lớp Then chỉ số tổng (số bài nộp, tỉ lệ đạt), **không** danh tính, không danh sách tụt điểm; STUDENT → `403`; sinh viên không bao giờ thấy thống kê của người khác (chỉ điểm của mình ở `/me`).
- AC8 (rỗng / lỗi). Given lớp chưa có bài thi công bố Then `EmptyState` một dòng; lỗi API → `<PageState>` + `Thử lại`.
- AC9 (hiệu năng). Given lớp 1.000 bài nộp Then API p95 ≤ 500 ms; cursor cho danh sách câu sai / tụt điểm; `Cache-Control` theo sự kiện công bố (không cache điểm phía máy chủ — AGENTS #15).
- AC10 (di động). Given 375 px Then biểu đồ co theo bề rộng, bảng cuộn trong khung.

### Ngoài phạm vi của story này
- Thống kê bài tập (`/grading`); xuất file; dự đoán điểm.

### Phụ thuộc
- `FEAT-weekly-exam` (`GET …/exams/{eid}/stats` — tái dùng phần có sẵn), `FEAT-crm-360` (liên kết hồ sơ), US-P10-04 (chủ đề).

---

## US-P10-07: Đội muốn seed, cổng nghiệm thu và E2E cho Observation A + B, để F16 và F17 đi trọn và đợt 1–2 đóng
Ưu tiên: Must · Ước lượng: M · Sprint: 11 · **Đợt 1–2**

### Tiêu chí nghiệm thu
- AC1 (seed A). Given stack trống When `node scripts/seed.mjs` Then `llm_audit` có ≥ 7 ngày dữ liệu (≥ 300 dòng, có `error`, `degraded`, `fallback_index > 0`, `pii_masked_count > 0`, ≥ 2 lớp, ≥ 3 tác vụ), `llm_request_payloads` có bản **đã che** cho ≥ 20 dòng gần nhất, 1 dòng thư gửi lỗi; chạy lại idempotent.
- AC2 (seed B). Given seed Then lớp 1 ≈ 150 câu hỏi lệch chủ đề A, B + 10 câu ngoài tài liệu (≥ 3 sinh viên mỗi chủ đề lớn, 1 chủ đề đúng 2 sinh viên); lớp 2 ≈ 40 câu lệch chủ đề C; ≥ 4 bài thi PE đã công bố với phân bố điểm có tụt 3 lần liên tiếp ở ≥ 1 sinh viên; `scripts/check-p10-seed.mjs` xanh.
- AC3 (cổng). Given repo When chạy:
  - `cd backend-go && go test ./internal/insight -run TestKnowledgeGap -v`
  - `cd backend-go && go test ./internal/observability -run 'TestObservabilityDetailNoPII|TestObservabilityAuditEveryOpen|TestAlertDedup' -v`
  - `cd backend-go && go test ./internal/llm -run TestPayloadStoresMaskedOnly -v`
  - `pnpm -C frontend exec playwright test insights.spec.ts observability.spec.ts analytics-exam.spec.ts`
  Then tất cả xanh.
- AC4 (E2E F17). Given `insights.spec.ts` When Teacher tạo báo cáo lớp 1 → mở câu mẫu → tạo thread ghim → đổi lớp 2 Then A, B đứng đầu lớp 1, C đứng đầu lớp 2, thread ghim ở `/threads`; nhánh lỗi: `generate` lần hai không dữ liệu mới → `reused`; LLM lỗi → tên tạm.
- AC5 (E2E F16). Given `observability.spec.ts` When Admin mở chi tiết với lý do Then có dòng `audit_log`; Teacher thấy bản tổng hợp, không nội dung; nhánh lỗi: lý do ngắn → `422`; nội dung hết hạn → `410`.
- AC6 (hợp đồng). Given `openapi.yaml` Then có mọi endpoint mới của US-P10-01…06; `go test ./internal/contract/...` xanh.
- AC7 (xoá mock). Given `frontend/src/mock/system.ts` Then không còn người dùng trong `src/` thì xoá cùng đợt; `bash scripts/ui-antipatterns.sh` sạch; ảnh mốc `insights-1440`, `observability-1440` cập nhật.
- AC8 (luồng). Given FLOWS F16, F17 Then đi trọn đường chính + nhánh lỗi trên stack thật; `docs/PROGRESS.md`: F17 ☑, M13 ghi «xong» khi cổng đạt.

### Ngoài phạm vi của story này
- Seed T1 (đã có ở 9.5).

### Phụ thuộc
- US-P10-01…06.

---

## US-P10-08: Đội muốn 8 trang đạt LCP ≤ 2,5 s trên build production, để giữ ngân sách hiệu năng không nới
Ưu tiên: Must · Ước lượng: L · Sprint: 11 · **Đợt 3** · Nợ G7-23 + 9.5 TL-2f; chủ dự án chốt 2026-11-14: **sửa mã, không nới ngưỡng**; PM P-11-5, P-11-6: đo ở trạng thái quay lại, chỉ hướng (a)

### Tiêu chí nghiệm thu
- AC1 (bộ đo). Given `benchmarks/lighthouse/run.sh` (devtools throttle, build production, đăng nhập thật, dữ liệu T1) Then đo 8 route: `/`, `/chat`, `/threads`, `/inbox`, `/gradebook`, `/me`, `/settings/llm`, `/assignments/[id]`; `/dev/ui` không đo (Q-E-11). **Trạng thái đo (P-11-5):** tài khoản đo đã bấm «Đã hiểu» ở hộp thoại thời gian học một lần trước lượt đo (người dùng quay lại; dữ liệu T1 không đổi); số đo **lần đầu** (chưa bấm) báo riêng, không tính vào ngưỡng.
- AC2 (đạt). Given lần đo Then `largest-contentful-paint` **trung vị 3 lượt ≤ 2.500 ms** ở cả 8 route; ngưỡng `lighthouserc.devtools.json` **giữ 2.500** (cấm nới).
- AC3 (nguyên nhân). Given mỗi route từng trượt (`/threads`, `/inbox`, `/gradebook`, `/me`, `/settings/llm`, `/assignments/[id]`) Then handoff nêu nguyên nhân từ waterfall (làm mới phiên → dữ liệu → vẽ) và cách sửa; sửa **ở mã** (ví dụ tải song song, bỏ chuỗi phụ thuộc, dữ liệu đầu trang theo mức mở dần), không cắt tính năng, không đổi dữ liệu đo.
- AC3a (đổi chữ khối lớn nhất, P-11-6a; BA xác nhận theo luật chữ UI). Given khối chữ lớn nhất sau phiên vượt PreShell Then đổi theo bảng, **không thêm câu giải thích, không tooltip mới**: `/settings/llm` trạng thái rỗng → «Chưa có nhà cung cấp» + nút `Thêm nhà cung cấp`; `/gradebook` thông báo điểm cộng ghi tay → nhãn «Ghi tay» gắn cạnh cột / ô liên quan (không câu dài); `/me` headline → một dòng `nhãn + giá trị` «Điểm quá trình  8,75 / 10» cỡ chữ ≤ h1 của PreShell; `/inbox` dòng ticket còn 1 dòng ở 412 px. Chữ khác trong các màn này giữ nguyên. Đổi chữ nào có trong SRS thì vá SRS cùng đợt (CLAUDE.md luật chữ UI).
- AC4 (không hồi quy). Given sau sửa Then TBT các route vẫn ≤ ngưỡng hiện có; e2e liên quan xanh; không route nào xấu đi > 10% LCP ở nhóm đang đạt (`/`, `/chat`).
- AC5 (CI). Given CI Then job Lighthouse đo cùng bộ route và đỏ nếu LCP > 2.500; số đo ghi «trên MacBook Apple Silicon, Colima 6 CPU / 10 GB».
- AC6 (route còn trượt; P-11-6). Given route nào còn LCP > 2.500 ms sau U1–U4 và (a) Then **không nới ngưỡng** và **không** làm SSR dữ liệu đầu trang / làm mới phiên phía máy chủ ((b) đụng auth, chưa có quyết định chủ dự án): ghi route, số đo thật (trung vị 3 lượt, cả trạng thái quay lại lẫn lần đầu), phần tử LCP, nguyên nhân vào handoff + `docs/PROGRESS.md` mục Nợ; **story không tính FAIL** vì route đó. Báo cáo ghi thêm thời điểm dữ liệu đầu trang vẽ xong để không trình bày LCP-theo-PreShell như thể dữ liệu đã hiện.

### Ngoài phạm vi của story này
- Đo trên điện thoại thật; tối ưu máy chủ (SLO máy chủ đã đạt ở 9.5).

### Phụ thuộc
- `benchmarks/lighthouse/run.sh` (9.5), `lighthouse-auth.cjs`.

---

## US-P10-09: Đội muốn đóng Nợ L6, để `start` từ chối số câu 0 như spec
Ưu tiên: Should · Ước lượng: S · Sprint: 11 · **Đợt 3** · P-10-10

### Tiêu chí nghiệm thu
- AC1. Given sinh viên When `POST …/practice/start {mode:"TOPIC", topic, count:0}` Then `422 VALIDATION_FAILED` (không phải `201`); `count` thiếu → mặc định 10 (như US-P9-06 AC2); `count` âm / > 50 → `422`.
- AC2 (không hồi quy). Given `start` hợp lệ Then `201` / `200` như cũ; `TC-05-16` đổi từ «Nợ chấp nhận» sang PASS; `go test ./internal/practice` xanh.
- AC3 (phân quyền). Given Teacher / TA / ADMIN gọi `start` Then `403` (không đổi).

### Ngoài phạm vi của story này
- Đổi hành vi `start` khác.

### Phụ thuộc
- Sprint 10 đã merge (`internal/practice`, `StartIn`).

---

## US-P10-10: Chủ dự án muốn `make eval` chạy E1–E6 và ra bảng số liệu có nguồn, để quyển đồ án có số
Ưu tiên: Must · Ước lượng: L · Sprint: 11 · **Đợt 4** · «KHÔNG ĐƯỢC CẮT» (P10)

### Tiêu chí nghiệm thu
- AC1 (một lệnh). Given repo When `make eval` Then chạy E1–E6 theo PRD §Đánh giá và ghi `benchmarks/reports/e{1..6}.json` + `.md` + biểu đồ **SVG** (sinh bằng thư viện chuẩn Python, không `matplotlib`; P-11-2); mục đang chờ dữ liệu người ghi `status:"WAITING_HUMAN"` (không chạy giả).
- AC2 (chỉ HTTP). Given mã đánh giá Then ở `benchmarks/`, Python, **chỉ gọi HTTP API của gateway**; không import mã Go; không service Python thường trực (D46). Kiểm: `grep -r "import.*backend" benchmarks` = 0.
- AC3 (E1 PII). Given bộ 200 bài gắn nhãn (nhãn do người gán) Then báo P / R / F1 kênh + từng loại PII, tỉ lệ chặn nhầm, rò rỉ payload; mục tiêu F1 kênh ≥ 0.90, **rò rỉ = 0** (đề xuất mục tiêu, Q-E-14).
- AC4 (E2 ngưỡng). Given golden set quy chế + 100 câu học thuật có nhãn Then đường cong escalate-rate / error-rate theo ngưỡng (≥ 7 điểm).
- AC5 (E3 chấm tự luận). Given ≥ 60 bài **chấm tay bởi người** theo rubric Then báo QWK, MAE, Pearson cho ≥ 2 provider; **chưa có điểm chuẩn người → `WAITING_HUMAN`**, báo cáo ghi rõ lý do (điểm LLM sinh không được dùng làm chuẩn: vòng tròn).
- AC6 (E4 provider). Given ≥ 2 provider Then chạy lại E2 (và E3 khi có) → chất lượng, độ trễ, chi phí / 100 yêu cầu; provider không khả dụng → ghi lý do, không bỏ trống im lặng.
- AC7 (E5 câu sinh tự động). Given 100 câu AI sinh được giảng viên duyệt (từ `GenerateQuestions`, dữ liệu `gen-stats`) Then tỉ lệ duyệt nguyên / sửa nhẹ / loại; < 100 câu có kết quả duyệt → ghi cỡ mẫu thật, không nội suy.
- AC8 (E6 nền Go). Given cùng máy, cùng seed Then RAM nghỉ, thời gian khởi động, kích thước image, p95 ≥ 5 endpoint; ghi cấu hình máy; không đối chiếu hệ cũ (D45).
- AC9 (tính hợp lệ). Given mỗi báo cáo `.md` Then có mục «Giới hạn»: dữ liệu mô phỏng, chưa kiểm chứng trên bài làm thật, E3 dùng chấm tay của ai; số đo LLM thật ghi nhà cung cấp + mô hình + ngày.
- AC10 (an toàn khoá). Given log / báo cáo Then không chứa khoá API, tên / MSSV thật; tôn trọng hạn mức nhà cung cấp (dừng nhẹ khi 429, không vòng lặp thử lại vô hạn).
- AC11 (tái lập). Given chạy lại `make eval` cùng seed + provider `fake` cho phần không cần LLM thật Then E1, E2, E6 cho kết quả trong sai số được ghi (E2 / E6 không bắt buộc bằng nhau tuyệt đối).
- AC12 (nhánh lỗi). Given gateway không chạy When `make eval` Then thoát mã ≠ 0 kèm một dòng «Gateway chưa chạy» (không ghi báo cáo rỗng).

### Ngoài phạm vi của story này
- Chấm tay E3 (việc của chủ dự án); mở rộng bộ nhãn; so sánh với hệ cũ.

### Phụ thuộc
- `benchmarks/pii/eval_pii.py`, `benchmarks/real-llm`, 9.5 (đo LLM thật), US-P10-07 (dữ liệu).

---

## US-P10-11: Đội muốn mỗi luồng F1–F17 có spec E2E đi trọn đường chính và một nhánh lỗi, để có bảng luồng × spec × kết quả cho luận văn
Ưu tiên: Must · Ước lượng: L · Sprint: 11 · **Đợt 4**

### Tiêu chí nghiệm thu
- AC1 (bảng). Given `docs/specs/FEAT-p10-eval/e2e-matrix.md` Then bảng F1…F17 × {spec Playwright, đường chính, nhánh lỗi, kết quả, ngày chạy}; không ô «—» ở cột spec.
- AC2 (đủ). Given mỗi luồng Then ≥ 1 test đi trọn đường chính **và** ≥ 1 nhánh lỗi liệt kê trong FLOWS; luồng đã có spec (F1…F12, F19) được ánh xạ, không viết lại; chỉ viết bù luồng thiếu (F13, F14, F15, F16, F17 nếu chưa đủ).
- AC3 (chạy). Given stack test When `pnpm -C frontend exec playwright test` Then toàn bộ xanh; test chập chờn (≥ 1 lần đỏ trong 3 lượt) bị ghi riêng, không che bằng retry vô hạn.
- AC4 (không nới). Given test đỏ Then sửa mã hoặc ghi lỗi; **không** nới assertion / sửa contract test (AGENTS.md).
- AC5 (luật đa vai). Given luồng có nhiều vai Then test dùng đúng tài khoản vai đó; ít nhất một test «vai sai → 403 / không thấy» cho F5, F9, F10, F16, F17.
- AC6 (xuất luận văn). Given bảng Then QC/`writer` dùng nguyên bảng cho luận văn; số liệu khớp lần chạy cuối (ghi commit).

### Ngoài phạm vi của story này
- F18 (PR); F19 đã có spec PE.

### Phụ thuộc
- US-P10-07; mọi sprint trước.

---

## US-P10-12: Đội muốn rà bảo mật, README chạy trên máy, visual QA hai lượt và kịch bản demo 15 phút chạy trọn, để sẵn sàng bảo vệ
Ưu tiên: Must · Ước lượng: L · Sprint: 11 · **Đợt 5**

### Tiêu chí nghiệm thu
- AC1 (secret). Given repo + lịch sử nhánh Then `gitleaks`-tương đương hoặc `grep` theo mẫu khoá = 0 khoá thật; `.env*` không commit; log không có PII (kiểm bằng chạy luồng chat + chấm rồi `grep` tên / MSSV seed trong log = 0).
- AC2 (CORS, rate limit). Given gateway Then CORS chỉ cho origin cấu hình; login / đăng ký / đặt lại / tham gia lớp có rate limit (Redis); vượt → `429` + `Retry-After`; kiểm bằng test.
- AC3 (quyền theo lớp). Given mỗi nhóm endpoint lớp học Then có test «sinh viên lớp khác → `404`, vai sai → `403`» (bảng nhóm × test trong handoff); mọi handler lớp học qua `CourseAccessGuard`.
- AC4 (báo cáo rà). Given kết quả Then `docs/sprints/11/qc/security-review.md` liệt kê mục đã rà, lỗi tìm thấy, mức độ, đã sửa / nợ; lỗi Cao phải sửa trong sprint.
- AC5 (README). Given máy sạch (macOS Apple Silicon + Colima) When làm theo README mới Then dựng được stack bằng `pnpm dev`, có đăng nhập demo, chạy được test cơ bản; có mục «Chạy trên máy», «Biến môi trường», «Dữ liệu demo mô phỏng (D44)», «Khắc phục sự cố ngắn». Kiểm: QC làm theo từng bước trên worktree sạch.
- AC6 (compose production). Given `docker-compose.prod.yml` Then khởi động được (kiểm `docker compose config` + dựng thử một lần), không khoá cứng; hướng dẫn triển khai máy chủ ngắn (VPS cũ đã thu hồi: ghi là hướng dẫn tham khảo).
- AC7 (chuyển động). Given Red Thread Transition Then < 300 ms, tôn trọng `prefers-reduced-motion` (bật → không chuyển động); đây là việc **đầu tiên bị cắt** nếu trễ (không chặn story).
- AC8 (visual QA hai lượt). Given Lượt A chụp desktop + mobile **7 route đại diện** (QC chọn, nêu trong plan) → sửa **một mẻ** → Lượt B chụp lại để xác nhận; **không lượt thứ ba**; ảnh và danh sách lỗi lưu ở `docs/sprints/11/qc/visual/`.
- AC9 (10 câu nghiệm thu). Given 10 câu hỏi nghiệm thu cuối (`AGENT_PROMPT.md`) Then mỗi câu có trả lời + bằng chứng (lệnh / ảnh / test) trong `docs/sprints/11/qc/acceptance-10.md`; câu không đạt ghi rõ.
- AC10 (demo 15 phút). Given `docs/DEMO_SCRIPT.md` When chạy trọn trên stack demo (7 bước F2…F17) Then tổng ≤ 15 phút (đồng hồ bấm), mọi «kết quả phải thấy» đúng; lệch → sửa kịch bản hoặc seed, không sửa kết quả mong đợi cho khớp lỗi.
- AC11 (kịch bản khớp seed). Given Bước 7 (F17) Then khớp US-P10-04/05 (A, B đứng đầu lớp 1; C lớp 2; chủ đề < 3 SV không câu mẫu); `DEMO_SCRIPT.md` mục «Nhật ký chạy lại» có dòng mới (ngày, commit, thời lượng).
- AC12 (video). Given video dự phòng Then **chủ dự án quay** (ngoài phạm vi agent); story ghi «chờ chủ dự án», không tính FAIL.
- AC13 (chữ UI). Given quét luật chữ UI Then `bash scripts/ui-antipatterns.sh` sạch; các màn cũ còn câu mô tả dưới tiêu đề (`/settings/llm`, `/settings/integrations`, khối bảo mật `/settings` — Nợ #42) được dọn **một lượt**.
- AC14 (cờ chèn lệnh bằng LLM, P-11-9). Given luồng chấm bài tập (P7) When chấm một bài Then ngoài regex hiện có (`grading/injection.go`, giữ làm lớp đầu) có **một** phép gắn cờ bằng `llm.Structured(json_schema)` qua `internal/llm`: **gộp vào lời gọi chấm sẵn có nếu được, không thêm vòng gọi** (không được thì một lời gọi riêng, làn `BATCH`); cờ cuối = regex **OR** LLM; chỉ gắn `INJECTION_SUSPECTED` + tô câu nghi (nếu có), **không** đổi điểm, không xoá chữ khỏi đầu vào, không chặn việc chấm (TLR-4).
- AC15 (bắt đủ bộ 9.5). Given bộ 6 biến thể chèn lệnh (5 tệp `benchmarks/real-llm/data/injection/v1…v5` + `docs/specs/FEAT-assignments/fixtures/submissions/s6-injection.md`) When chạy với LLM thật (cấu hình ở `/settings/llm`) Then **≥ 6/6** có `INJECTION_SUSPECTED`; ghi nhà cung cấp + model + ngày; regex một mình chỉ bắt 1/6 được ghi làm mốc so sánh. Chạy CI với provider `fake` chỉ kiểm đường mã (OR, không đổi điểm), không kiểm độ bắt.
- AC16 (không báo nhầm bài trống). Given bài trống / chỉ có khoảng trắng (`EMPTY_SUBMISSION`) Then **không** gọi LLM gắn cờ và **không** có `INJECTION_SUSPECTED`; bài hợp lệ không có lệnh (7 bài fixture khác trong `FEAT-assignments`) → không cờ, kiểm trên bộ đối chứng.
- AC17 (nhánh lỗi). Given LLM `OVERLOADED` / lỗi / trả sai schema Then cờ chỉ theo regex, việc chấm vẫn hoàn tất, điểm không đổi, không treo; `llm_audit` ghi lỗi.
- AC18 (riêng tư và phân quyền). Given payload gắn cờ Then qua che danh tính (không tên / MSSV / email) như lượt chấm; sinh viên không thấy cờ (như hiện hành); chỉ TA, TEACHER lớp thấy ở màn duyệt.

### Ngoài phạm vi của story này
- Triển khai thật lên máy chủ; kiểm thử trên điện thoại thật (chủ dự án tự kiểm).

### Phụ thuộc
- US-P10-07…11; Nợ #42; `docs/DEMO_SCRIPT.md`.

---

## US-P10-13: Đội muốn một cascade «decision model trước, phân vân thì LLM» cho ba cổng (tường lửa Threads, cờ chèn lệnh, định tuyến chat), để nhanh hơn và rẻ hơn mà không bỏ sót
Ưu tiên: Should · Ước lượng: L · Sprint: 11 · **Đợt 6** · P-11-10 (PM: chủ dự án muốn tăng tốc bằng decision model) · nguồn: `docs/research/2026-11-14-jev-decision-model.md` mục 6, `docs/research/poc/jev-decision/cascade.md`

**Luật chạy (PM 2026-11-14):** mọi lời gọi LLM sinh chữ chỉ qua 9router; `gemini-decision-1` và embedding thật **không gọi** khi chưa có phép của PM — thiết kế và test đợt 6 bằng provider `fake` (bảng ngưỡng × kết quả giả); các số đo thật trong story này (AC4 iii, AC7, AC9) là điều kiện **chạy khi PM cho phép**, trước đó ghi «chờ phép PM», không chặn merge phần mã.

Nguyên tắc: decision model **không bao giờ là cổng duy nhất** (tự công bố: «treat input as untrusted»); các cổng an toàn chỉ **thêm** cờ / chặn; lỗi → fail-closed. **Không dùng cho điểm, mức rubric, escalate** (chờ nhãn người E2 / E3).

### Tiêu chí nghiệm thu
- AC1 (`Decide`, ADR D53). Given `internal/llm` When gọi `Decide(ctx, DecideRequest{Task, Lane, Input, Questions})` Then gọi `POST /v1beta/decisions` bằng `net/http` (**không SDK**), provider loại `decision` riêng, **chỉ** model `gemini-decision-1`, khoá `GEMINI_API_KEY` sẵn có (không khoá / bên xử lý mới); `Questions` là kiểu Go đóng (`noul` / `choice` / `score`), không JSON tự do. Đi cùng đường ống với `Structured`: che PII (`Masker`) → Scheduler (task `DECIDE`; `INTERACTIVE` cho chat / precheck, `BATCH` cho bài nộp; token bucket riêng theo provider) → provider → `llm_audit` (task, token, trễ, `error_kind`, `path`) → cầu dao. Package khác không import SDK hay gọi endpoint này trực tiếp.
- AC2 (`Cascade` và chính sách). Given `Cascade(ctx, decide, policy, fallback)` Then hai chính sách: **đối xứng** `τ` (nhận kết quả decision khi `confidence ≥ τ`) và **chỉ-thêm-cờ** `ε` (decision «có» với p ≥ 0,5 nhận ngay; «không» chỉ nhận khi `P(có) ≤ ε`; vùng giữa → fallback); mọi lỗi của decision (hết giờ 1 s ở `INTERACTIVE`, 429, 5xx, 400) → fallback; fallback cũng lỗi → **giá trị fail-closed do chính sách quy định** (chặn / dương); trả `Path ∈ {FAST, FALLBACK, FAIL_CLOSED}`, ghi vào `llm_audit.path`. Ngưỡng `τ` / `ε` theo task ở `llmconfig`, có mặc định trong mã, chỉ ADMIN sửa. Kiểm: `TestCascadeTable` (bảng ngưỡng × kết quả giả), `TestCascadeFailClosed`.
- AC3 (400 safety). Given decision trả `400` thuộc lọc an toàn (`safety` / `prohibited_content`) Then coi là **dương** (fail-closed), không thử lại, không coi là lỗi hạ tầng.
- AC4 (Threads: tường lửa chỉ-thêm-cờ, ε = 0,05). Given bài đăng / bình luận công khai Then **luật tất định chạy trước, giữ nguyên** (MSSV / email / SĐT / CCCD / tên trong danh sách lớp / placeholder `[[…]]`); phần còn lại hỏi decision `noul` «câu hỏi riêng tư / nêu tên người cụ thể», vùng giữa → LLM `Structured`; **chỉ được thêm chặn, không bao giờ gỡ chặn** của luật. Cổng: trên tập `dev` của E1 (embedding giả mặc định; embedding thật chỉ khi PM cho phép) (i) mọi mẫu luật đã chặn vẫn bị chặn, (ii) không mẫu dương nào từng bị bắt mà nay lọt, (iii) tỉ lệ chặn nhầm `false_block ≤ 0,05`, (iv) rò rỉ payload = 0; tập `test` chỉ chạy một lần ở lần chốt. Câu chỉ có placeholder được luật chặn trước khi tới model (model đọc không ra placeholder, F1 0,883).
- AC5 (cờ chèn lệnh = regex OR `Structured` OR decision). Given luồng chấm (US-P10-12 AC14) Then thêm decision `noul` trên bài đã che, làn `BATCH`; vùng giữa → `Structured` (P-11-9); cờ cuối = OR cả ba; chỉ `INJECTION_SUSPECTED`, không đổi điểm; bộ 6 biến thể 9.5 vẫn ≥ 6/6, bài trống không gọi, bài đối chứng không cờ (AC15, AC16 của US-P10-12 giữ nguyên).
- AC6 (định tuyến chat: ngưỡng + fallback về luật + đo TTFT). Given tin chat không khớp luật intent cụ thể (rơi về mặc định `COURSE_QA`) Then gọi decision `choice` (11 nhãn ý định D47), **song song** với lần nhúng duy nhất (`TestOneEmbedPerMessage` giữ): `confidence ≥ τ` (mặc định 0,95, đối xứng) → dùng nhãn decision; dưới `τ` → **fallback = giữ kết quả luật** (không gọi LLM; P-11-11 / Q-E-24); luật intent cụ thể (`OTHER_PERSON`, `SMALLTALK`…) **không** bị decision ghi đè. Không thêm lời gọi sinh chữ: D47(1) giữ.
- AC6a (tin khủng hoảng không rời máy chủ; giữ US-P3-04 AC13 — P-11-11 / Q-E-23). Given tin chat hoặc bài Threads bị `IntentCrisis` bắt Then xử lý **trước** mọi cổng của story này; **không** gửi tới decision / `Structured` / bất kỳ LLM nào, không `llm_audit`, không log nội dung; chuỗi gọi trong mã kiểm bằng `TestCrisisNeverReachesDecide`. Lưới khủng hoảng **không** nằm trong cascade (không có `crisis_net_decision`).
- AC7 (cổng TTFT). Given cùng seed, cùng máy When đo trước / sau (≥ 30 câu hỏi, ≥ 3 lượt, provider thật cho sinh chữ qua 9router) Then sự kiện SSE đầu ≤ 300 ms **không đổi** (`received` phát trước phân loại); thời gian tới token đầu p95 không xấu đi quá 20 %; vượt → tắt `route_decision`, ghi số đo vào báo cáo, **không** nới ngưỡng; ghi «trên MacBook Apple Silicon, Colima 6 CPU / 10 GB».
- AC8 (không dùng cho điểm, rubric, escalate). Given mã Then `Decide` / `Cascade` chỉ được gọi từ **ba** nơi: tường lửa Threads (precheck), cờ chèn lệnh, định tuyến chat; **không** từ `internal/grade`, tính / làm tròn điểm, chọn mức rubric, quyết định escalate, đường khủng hoảng. Kiểm bằng test liệt kê nơi gọi (`TestDecideCallersAllowlist`). Trong luồng chấm, decision **chỉ** gắn cờ.
- AC9 (đo và hiển thị). Given `make eval` / `/observability` Then (a) `benchmarks/reports/cascade.md` có bảng mỗi tác vụ: % đường nhanh, độ đúng, sót dương, trễ p50 / p95, chi phí / 1.000 yêu cầu, so với chỉ-LLM và với luật một mình; (b) `/observability` (Admin) hiện tỉ lệ `FAST / FALLBACK / FAIL_CLOSED` theo tác vụ, **không** nội dung; Teacher không thấy.
- AC10 (nhánh lỗi). Given `GEMINI_API_KEY` trống / hết hạn mức / mạng lỗi Then cả ba cổng vẫn hoạt động: tường lửa Threads và cờ chèn lệnh rơi về fallback hoặc fail-closed (không bao giờ «mở cửa»: lỗi cả hai → chặn / gắn cờ), định tuyến chat rơi về luật; chat không treo; thông báo ADMIN theo US-P10-03 khi tỉ lệ `FAIL_CLOSED` cao bất thường.
- AC11 (riêng tư và triển khai). Given dữ liệu gửi decision Then luôn đã che (như mọi lời gọi LLM), `TestDecidePayloadMasked` (tên + MSSV thật → không có ở thân yêu cầu); gói Gemini **miễn phí** dùng dữ liệu để cải thiện sản phẩm nên chỉ chạy với dữ liệu mô phỏng đã che; triển khai thật phải dùng gói trả phí (`PRODUCTION_READINESS.md`, việc của Tech Lead).
- AC12 (phân quyền). Given cấu hình `τ` / `ε` / cờ bật tắt Then chỉ ADMIN sửa (cùng nhóm `PUT /admin/llm/routes`, có `audit_log`); Teacher / TA / STUDENT → `403`; sinh viên không bao giờ thấy từ kỹ thuật (decision, cascade, ngưỡng).
- AC13 (đóng đợt 6). Given cổng Then `go test ./internal/llm/... ./internal/privacy/... ./internal/agent/... ./internal/grading/...` xanh (provider `fake`); E1 dev đạt AC4 khi PM cho phép chạy thật; số đo TTFT / cascade có trong báo cáo khi PM cho phép; mọi cờ mặc định **tắt** cho tới khi cổng tương ứng đạt.

### Ngoài phạm vi của story này
- Điểm, mức rubric nháp, escalate (cần nhãn người E2 / E3, sau bảo vệ); loại nhà cung cấp Jev / `typesafe`; chạy decision trên bài làm của sinh viên ngoài cờ chèn lệnh.

### Phụ thuộc
- US-P3-04 AC13 giữ nguyên (khủng hoảng không gọi LLM; P-11-11).
- ADR D53 + sửa D47(2) + làm rõ AGENTS luật 5 + `PRODUCTION_READINESS.md` (Tech Lead, PM duyệt); P-11-9 (`Structured` gắn cờ); US-P10-12 AC14…16; E1 (`benchmarks/pii`); US-P10-01 / 03 (`/observability`, cảnh báo); `llm_audit.path` (đề xuất đổi ARCHITECTURE).

