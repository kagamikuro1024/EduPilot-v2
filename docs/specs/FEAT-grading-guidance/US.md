# FEAT-grading-guidance Harness học từ lần giảng viên sửa (M1 lưu ý chấm, M2 ca hồi quy, M4 câu tương tự Threads, M6 mẫu câu bị loại)
Nguồn: `docs/sprints/10.6/plan.md` · research `docs/research/2026-11-14-agent-memory.md` + thẩm định PM `docs/research/2026-11-14-agent-memory-tham-dinh-pm.html` (nhánh `research/agent-memory`) · ADR **D61**, **D62** (`docs/DECISIONS.md`, techlead-mem) · AGENTS.md nguyên tắc 2, 4, 5, 11, 12 · FEAT-assignments (F9: 4.6 Grading Engine, 4.7 hàng đợi, 4.8 duyệt / công bố, 4.10 phúc khảo) · FEAT-practice (F12: `GenerateQuestions`) · FLOWS F4, F9, F12 · PRD M1, M8 · `docs/design/DESIGN.md` §13, §14 · `docs/UX.md` mục 6 · FEAT-llm-gateway.
Phase: sprint 10.6. Trạng thái: **APPROVED** v1.0 (2026-11-14; PM uỷ quyền; câu không đụng điểm / phân quyền / dữ liệu cá nhân: **PM chốt theo đề xuất BA**; các câu đụng điểm, phân quyền, dữ liệu cá nhân ghi **PM xác nhận P-106-1** ở `QUESTIONS.md` (spec đã viết theo phương án an toàn nhất)).

**Ý tưởng một dòng.** Mỗi lần giảng viên sửa điểm nháp của AI là một tín hiệu có cấu trúc đã nằm trong DB; feature này **khép vòng** bằng một bước **người duyệt**: AI rút «lưu ý chấm» dạng nháp → giảng viên duyệt từng lưu ý → thành **phiên bản hướng dẫn chấm** gắn với bài tập → mỗi lượt chấm **ghim** một phiên bản → đổi phiên bản thì chấm lại lô chưa công bố. Không dùng hệ bộ nhớ agent bên ngoài (D61); lưu ý là dữ liệu Postgres có người duyệt (D62).

**Bất biến của feature (không thương lượng)**
1. **Không lưu ý nào tự áp.** Mọi lưu ý ở trạng thái `DRAFT` cho tới khi **TEACHER** duyệt và bấm `Áp dụng`. AI chỉ nháp, người quyết (CONTEXT mục 3).
2. **Không đọc văn bản bài làm** ở bước rút lưu ý; đầu ra qua `privacy.Detect`; khoá `(course_id, assignment_id)`; **không có bộ nhớ theo sinh viên** (M11, M12 không làm).
3. **Điểm tính bằng code thuần.** Lưu ý chỉ tác động điểm **từng tiêu chí** trong bản nháp LLM; tổng, trừ muộn, làm tròn giữ nguyên bằng `decimal` (nguyên tắc 5). Schema lưu ý không có trường tổng / phạt.
4. Mọi lời gọi LLM đi qua `internal/llm` (làn `BATCH`, `llm.Structured`, 9router); không SDK; không chạm đường hỏi–đáp (D47).
5. **Mặc định tắt theo lớp** (cờ), giống D60. Tắt cờ = hành vi hệt như trước sprint 10.6; dữ liệu giữ nguyên để đo.
6. Sinh viên **không bao giờ** thấy lưu ý, phiên bản, cờ hay số đo (không từ «AI», «nháp», «lưu ý chấm» ở màn sinh viên).

Mã vai: ADMIN, TEACHER, TA, STUDENT. Từ ngữ giao diện: «Lưu ý chấm» (guidance), «bản v2» (phiên bản), «Rút lưu ý», «Duyệt», «Bỏ», «Áp dụng», «Chấm lại».

---

## US-GD-01: Giảng viên muốn bật / tắt từng phần «học từ lần sửa» cho lớp mình, mặc định tắt, để thử an toàn
Ưu tiên: Must · Ước lượng: S · Sprint: 10.6 · **L1**

### Tiêu chí nghiệm thu
- AC1 (bốn cờ). Given lớp Then có bốn cờ theo lớp, **mặc định tắt**: `GRADING_GUIDANCE` (M1), `REGRESSION_CAPTURE` (M2), `THREAD_SUGGEST` (M4), `QUESTION_HINTS` (M6); không hàng = tắt.
- AC2 (ai bật). Given Teacher của lớp When bật / tắt `GRADING_GUIDANCE`, `THREAD_SUGGEST`, `QUESTION_HINTS` Then `200`, `version + 1`, `audit_log`; `REGRESSION_CAPTURE` **chỉ ADMIN** (liên quan dữ liệu bài làm, Q-G-3); TA / STUDENT → `403`; lớp khác → `404`. (Q-G-2, PM xác nhận P-106-1) vai bật cờ.
- AC3 (công tắc cứng toàn hệ thống). Given biến `GUIDANCE_ENABLED` (mặc định `true`) = `false` Then mọi cờ coi như tắt, API ghi trả `409 GUIDANCE_DISABLED`, không job nào chạy (tắt khẩn một chỗ).
- AC4 (hiệu lực tức thời). Given cờ tắt Then: không job `grading.reflect` mới; prompt chấm **không** thêm khối lưu ý; không gợi ý câu tương tự; prompt sinh câu hỏi không thêm mẫu âm; không ghi ca hồi quy. Dữ liệu cũ (lưu ý, phiên bản) giữ nguyên, **không** hiện trong prompt.
- AC5 (giao diện). Given `/class/settings` Then một nhóm «Học từ lần sửa» với bốn công tắc (nhãn: «Lưu ý chấm», «Ca kiểm thử», «Gợi ý câu hỏi tương tự», «Gợi ý khi sinh câu hỏi»); trạng thái 1–2 từ; không câu mô tả; tooltip chỉ trên công tắc bị khoá («Chỉ quản trị viên bật») ; cập nhật lạc quan + «Hoàn tác»; 375 px dùng được.
- AC6 (nhánh lỗi). Given `PUT` với `version` cũ → `409 VERSION_CONFLICT`; thiếu quyền → `403`; flag lạ → `422`.

### Ngoài phạm vi của story này
- Cấu hình ngưỡng theo lớp (hằng số hệ thống, SRS 4.1).

### Phụ thuộc
- `courses`, `audit_log`; D61, D62.

---

## US-GD-02: Đội muốn nền dữ liệu bất biến cho hướng dẫn chấm (phiên bản, ghim, `ai_comments`), để mọi lượt chấm truy vết được
Ưu tiên: Must · Ước lượng: M · Sprint: 10.6 · **L1**

### Tiêu chí nghiệm thu
- AC1 (migration). Given migration mới (số kế tiếp sau migration cuối đã merge; file mới, không sửa file cũ) Then có `grading_guidance_items`, `grading_guidance_versions`, `course_guidance_flags`, `grading_regression_cases`; cột mới `assignments.guidance_version smallint NOT NULL DEFAULT 0`, `grading_results.guidance_version smallint NOT NULL DEFAULT 0`, `grading_results.ai_comments jsonb NULL`, `grading_results.first_reviewed_at timestamptz NULL`, `grading_results.first_reviewed_by uuid NULL` (P-106-5); thêm cột `question_bank.reject_reason`; FK phức hợp có `course_id`; index bắt đầu bằng `course_id` (SRS 5).
- AC2 (ghim). Given một lượt chấm bắt đầu (worker nhận việc) Then đọc `assignments.guidance_version` **trong cùng giao dịch** ghi `grading_results` và ghi vào `grading_results.guidance_version`; phiên bản đó dùng để dựng prompt của **cả hai lượt chấm** của việc đó; cờ `GRADING_GUIDANCE` tắt → ghim `0`.
- AC3 (bất biến). Given kết quả đã `DONE` Then `ai_scores`, `ai_comments`, `guidance_version` **không sửa được** (trigger, cùng cơ chế `ai_scores` ở `00017`); thử `UPDATE` → lỗi; việc thử lại (attempt) trước `DONE` được ghim lại.
- AC4 (`ai_comments`). Given worker ghi kết quả LLM lượt 1 đã kiểm Then lưu `ai_comments = {criteria:{<id>: "<nhận xét gốc của AI>"}, overall: "…"}` cùng lúc `ai_scores`; giảng viên sửa nhận xét **không** ghi đè bản gốc (chỉ `criteria[].comment` hiện hành đổi). Kết quả cũ trước migration: `ai_comments IS NULL` (không backfill).
- AC5 (tái dựng). Given `(assignment_id, guidance_version = v)` Then tập lưu ý áp dụng = các mục `status` đã duyệt có `approved_in_version ≤ v` và (`retired_in_version IS NULL` hoặc `> v`) — kết quả **tất định**; `TestGuidanceSetAtVersion` kiểm: dựng lại prompt của một lượt chấm cũ cho đúng chuỗi đã dùng.
- AC6 (không phá cái đang chạy). Given bài tập chưa từng có lưu ý Then `guidance_version = 0`, prompt chấm **giống từng byte** prompt hiện hành (`TestEssayPromptUnchangedAtV0`); toàn bộ test chấm của sprint 7 / 9 / 10.5 xanh.
- AC7a (dấu «đã duyệt» ghi một lần — P-106-5). Given giảng viên `review` một kết quả lần đầu Then `first_reviewed_at` / `first_reviewed_by` được ghi (`coalesce`, một lần) cùng lúc `reviewed_at` / `reviewed_by`; khi giảng viên sửa lại (`REVIEWED → DRAFT`, `reviewed_at` / `reviewed_by` về NULL) hai cột `first_*` **giữ nguyên**; duyệt lần hai không đổi chúng; mọi thử `UPDATE` đổi / xoá → lỗi (trigger, cùng cơ chế `ai_scores`); **không** thêm chặn nào ở bước `review`; `TestFirstReviewedWriteOnce`.
- AC7 (nhánh lỗi / quyền). Given Teacher lớp khác gọi bất kỳ API guidance → `404`; STUDENT → `403`; DB lỗi giữa chừng → không phiên bản nửa vời (giao dịch).

### Ngoài phạm vi của story này
- Rút lưu ý (US-GD-03); giao diện (US-GD-04, 06).

### Phụ thuộc
- `00017_assignments` (trigger `ai_scores`), `00006`; D62.

---

## US-GD-03: Giảng viên muốn hệ thống rút ra lưu ý chấm từ các lần tôi sửa điểm nháp, ở dạng nháp, để không phải sửa cùng một lỗi nhiều lần
Ưu tiên: Must · Ước lượng: L · Sprint: 10.6 · **L2** · M1

### Tiêu chí nghiệm thu
- AC1 (điều kiện). Given bài tập ESSAY ở lớp bật `GRADING_GUIDANCE` Then job `grading.reflect` chỉ chạy khi: có ≥ **5** kết quả hiện hành `REVIEWED` hoặc `PUBLISHED` **và** ≥ **3** điểm tiêu chí bị giảng viên sửa ≥ **0,5** so với `ai_scores`; hằng số `GUIDANCE_MIN_RESULTS`, `GUIDANCE_MIN_EDITS`, `GUIDANCE_MIN_DELTA`.
- AC2 (kích hoạt). Given Teacher bấm `Rút lưu ý` When đủ điều kiện Then `POST …/guidance/reflect` (`Idempotency-Key`) trả `202 {job_id}`; chưa đủ → `409 NOT_ENOUGH_DATA {results, edits}` (giao diện: «Cần thêm bài đã duyệt»); đã có việc cùng khoá → trả lại `job_id` cũ. (Q-G-4, PM chốt theo đề xuất BA) Không tự kích khi đủ ngưỡng: spec chốt **chỉ nút bấm** (một việc nền tốn lời gọi LLM do người quyết).
- AC3 (khoá idempotent). Given khoá `(assignment_id, max(reviewed_at), guidance_version)` Then cùng khoá không sinh lưu ý nháp lần hai; có bài mới được duyệt → khoá mới.
- AC4 (đầu vào — KHÔNG có văn bản bài). Given đầu vào của lời gọi Then chỉ gồm: rubric (tên + hướng dẫn từng tiêu chí), danh sách ca `{case_id, criterion_id, ai_score, teacher_score, ai_comment (từ ai_comments)}` tối đa 60 ca gần nhất, các lưu ý đã duyệt hiện hành (để không nhắc lại). **Không** có văn bản bài, nhận xét sửa tay của giảng viên, tên, MSSV, `student_id`, `submission_id`. `TestReflectInputNoSubmissionText`, `TestReflectInputNoIdentity`.
- AC5 (đầu ra có cấu trúc). Given `llm.Structured(json_schema)` Then mỗi mục `{criterion_id (enum id rubric), text (≤ 400 ký tự, tiếng Việt), direction ∈ {STRICTER, LENIENT, CLARIFY}, case_ids[] (≥ 2)}`; **không** trường tổng / phạt / làm tròn.
- AC6 (kiểm bằng code). Given đầu ra Then code: (i) `criterion_id` có trong rubric; (ii) mỗi `case_id` phải có trong đầu vào **và** thuộc đúng `criterion_id` của mục; **chỉ cần một `case_id` lạ (không có trong đầu vào, hoặc thuộc tiêu chí khác) là bỏ CẢ MỤC** (đếm `dropped.case`; không «bỏ riêng id lạ rồi đếm tiếp» — an toàn hơn trước đầu độc, P-106-7); (iii) `case_id` hợp lệ nhưng **ngược chiều** `direction` (STRICTER ⇔ `ai_score > teacher_score`) chỉ **không được đếm**; **`support` = số ca hợp lệ cùng chiều do code đếm**, không lấy từ LLM; bỏ mục `support < GUIDANCE_MIN_SUPPORT` (mặc định 2); (iv) bỏ mục chứa từ khoá tổng / trừ / muộn / làm tròn / phạt (`fold`, theo ranh giới từ) **sau khi miễn các cụm vô hại** «tổng quát», «tổng quan», «tổng thể», «tổng hợp», «tổng kết», «trừ khi», «ngoại trừ», «loại trừ» (P-106-12; mọi cách dùng khác vẫn bị loại; `TestReflectKeywordExempt`); (v) `privacy.Detect` (roster + regex MSSV / email / SĐT) khớp → **bỏ mục**; (vi) trùng ≥ 0,8 (Jaccard theo từ) với mục đã có cùng tiêu chí → bỏ; (vii) tối đa 8 mục mỗi lần.
- AC7 (lưu). Given mục hợp lệ Then lưu `DRAFT` với `support`, `source_result_ids[]` (id kết quả nguồn, **không** id sinh viên), `origin_job`; **không** tự áp, **không** đổi `assignments.guidance_version`; ghi `llm_audit` (task `GRADING`, làn `BATCH`, không nội dung; Q-G-15).
- AC8 (không có gì mới). Given 0 mục hợp lệ Then job `DONE` với `{drafts:0, dropped:[{criterion_id, reason, count}]}` (theo tiêu chí; `reason` ∈ `criterion, case, direction, support, keyword, pii, duplicate, overflow`; chỉ số, không chữ; `criterion_id` là `null` khi `reason = criterion`); giao diện: «Không có lưu ý mới»; khi tổng `count` N > 0 hiện thêm một dòng «N lưu ý bị loại» (cả khi `drafts > 0`), không câu mô tả (P-106-12).
- AC9 (nhánh lỗi). Given LLM `OVERLOADED` / lỗi / sai schema Then thử lại tối đa 3 lần (backoff của hàng đợi), vẫn lỗi → dead-letter + trạng thái job `FAILED`; không bản nháp nửa vời; giao diện: «Chưa rút được lưu ý. Thử lại.»; chấm bài và duyệt **không** bị ảnh hưởng.
- AC10 (trần). Given ≥ 3 việc `reflect` / giờ / bài tập Then `429 RATE_LIMITED` + `Retry-After`; làn `BATCH`, không làm chat treo.
- AC11 (phân quyền). Given TA / STUDENT gọi `reflect` → `403`; Teacher lớp khác → `404`; ADMIN → `403` (không có chỗ trong lớp); cờ tắt → `409 GUIDANCE_DISABLED`.
- AC12 (riêng tư). Given payload gửi LLM Then qua `Masker` như mọi lời gọi; `TestReflectOutputPII`: seed có tên / MSSV trong `ai_comments` → mục tương ứng bị bỏ; `TestReflectValidate` có ca «`case_id` lạ → bỏ cả mục» và «ca ngược chiều → chỉ không đếm».

### Ngoài phạm vi của story này
- Giọng nhận xét của giảng viên (M7); lưu ý cho QUIZ; đọc văn bản bài.

### Phụ thuộc
- US-GD-01, 02; FEAT-assignments 4.6, 4.8; `internal/privacy`; Redis Streams (jobs).

---

## US-GD-04: Giảng viên muốn duyệt từng lưu ý và bấm «Áp dụng» để thành bản mới, để tôi là người quyết cách chấm
Ưu tiên: Must · Ước lượng: L · Sprint: 10.6 · **L4** · M1

### Tiêu chí nghiệm thu
- AC1 (vị trí). Given Teacher / TA mở `/assignments/[id]` (nhìn nhân viên) Then dưới rubric có **một** khối «Lưu ý chấm»: dòng `bản v2 · 3 chờ duyệt` + `Rút lưu ý` (phụ); bài tập chưa bật cờ → khối ẩn.
- AC2 (danh sách). Given có lưu ý Then danh sách, mỗi mục: tiêu chí (tên) · nội dung · `n lần sửa` (support) · chiều (Gắt hơn / Nhẹ hơn / Làm rõ) · `Duyệt` · `Bỏ`; ngăn bằng đường kẻ 1 px (không card lồng); trạng thái: `Chờ duyệt`, `Chờ áp dụng`, `Đã áp dụng`, `Chờ nghỉ`, `Đã bỏ`, `Đã nghỉ`.
- AC3 (duyệt / bỏ). Given Teacher bấm `Duyệt` / `Bỏ` Then `POST …/guidance/items/{iid}/decide {decision, version}`; chỉ `DRAFT`; `409 ITEM_NOT_DRAFT` / `VERSION_CONFLICT`; ghi `audit_log` (mục, quyết định, người); `Duyệt` hiện «Chờ áp dụng», `Bỏ` hiện «Đã bỏ»; cả hai cập nhật lạc quan kèm «Hoàn tác» (AC3a).
- AC4 (sửa trước khi duyệt). Given Teacher sửa chữ của một lưu ý rồi `Duyệt` Then chữ mới qua lại `privacy.Detect` + từ khoá cấm (AC6 US-GD-03); bản gốc giữ ở `text_original`; `422 GUIDANCE_TEXT_REJECTED` nếu không qua.
- AC3a (Hoàn tác `Duyệt` / `Bỏ` / `Cho nghỉ` trước `Áp dụng` — P-106-8). Given Teacher vừa `Duyệt`, `Bỏ` hoặc `Cho nghỉ` một lưu ý **chưa thuộc phiên bản đã áp dụng** (hoặc chưa có hiệu lực: mục «Chờ áp dụng», «Đã bỏ», «Chờ nghỉ») Then: (a) hiện dòng tĩnh lặng «Hoàn tác» tại chỗ theo `useUndoableAction` (không hộp xác nhận, không toast «Thành công»), bấm → `POST …/guidance/items/{iid}/decide {decision:"RESET", version}` đưa mục về trạng thái trước đó: «Chờ áp dụng» / «Đã bỏ» → `DRAFT` (xoá người quyết + thời điểm), «Chờ nghỉ» → bỏ cờ chờ nghỉ (trở lại «Đã áp dụng»); (b) sau khi dòng «Hoàn tác» tự mất, mục vẫn đặt lại được bằng `Đặt lại` trong menu của hàng **cho tới khi `Áp dụng`**; (c) chữ đã sửa trước khi `Duyệt` được giữ (`text`, `text_original` không đổi); (d) `RESET` ghi `audit_log` (mục, trạng thái trước / sau, người), `version + 1`; (e) mục **đã vào một phiên bản đã áp dụng** (`approved_in_version` khác rỗng, không chờ nghỉ) hoặc `RETIRED` → `409 ITEM_NOT_RESETTABLE`, giao diện báo tại chỗ «Đã áp dụng, không hoàn tác được» và tải lại hàng (kể cả khi `Áp dụng` chạy ở tab khác ngay trước đó); (f) `RESET` lên mục đang `DRAFT` → `200` không đổi (idempotent); (g) `VERSION_CONFLICT` → tải lại hàng; (h) chỉ TEACHER; TA / STUDENT → `403`; lớp khác → `404`; cờ tắt → `409 GUIDANCE_DISABLED`; (i) `Áp dụng` **không** có «Hoàn tác»: là thao tác tạo phiên bản mới nên dùng `ConfirmIrreversible` (AC5). `TestGuidanceItemReset`, `guidance-undo` trong `grading-guidance.spec.ts`.
- AC5 (Áp dụng). Given có ≥ 1 mục «Chờ áp dụng» hoặc «Chờ nghỉ» When Teacher bấm `Áp dụng` Then hiện `ConfirmIrreversible` (tên «Áp dụng bản v{N+1}?», số mục thêm / nghỉ; **không** có «Hoàn tác» sau khi xác nhận) và khi xác nhận `POST …/guidance/apply {expected_version}` (`Idempotency-Key`) tạo **phiên bản mới** `v+1` gồm các mục đã duyệt (trừ đã nghỉ), cập nhật `assignments.guidance_version`; một giao dịch; `409 VERSION_CONFLICT` nếu `expected_version` cũ; `409 NOTHING_TO_APPLY` nếu không đổi gì; `audit_log`.
- AC6 (chỉ TEACHER). Given TA When `Duyệt` / `Bỏ` / `Áp dụng` Then `403` (TA chỉ xem danh sách); STUDENT → `403`. (Q-G-2, PM xác nhận P-106-1) TA có được duyệt không — mặc định **không**.
- AC7 (cho nghỉ). Given lưu ý đã duyệt không còn đúng When Teacher `Cho nghỉ` Then mục sang «Chờ nghỉ»; chỉ có hiệu lực ở phiên bản **kế tiếp** khi `Áp dụng`; phiên bản cũ dựng lại vẫn có mục đó (AC5 US-GD-02).
- AC8 (sau Áp dụng). Given phiên bản mới Then hiện ngay ở `/grading` «N bài chấm theo bản cũ · Chấm lại» (US-GD-05, chỉ tính bài nháp chưa sửa) và `Áp dụng` trả `stale:{drafts, edited_drafts, reviewed}`; **không** tự chấm lại.
- AC9 (nhánh lỗi / rỗng). Given chưa có lưu ý → `EmptyState` một dòng; API lỗi → `<PageState>` + `Thử lại`; mất chữ đã sửa → `useAutosaveDraft`.
- AC10 (di động). Given 375 px Then danh sách xếp dọc, nút vùng chạm ≥ 44 px.
- AC11 (chữ UI). Given quét luật chữ UI Then không dòng mô tả dưới tiêu đề khối, không icon Info, không từ «AI / RAG / PII / nháp»; `bash scripts/ui-antipatterns.sh` sạch.

### Ngoài phạm vi của story này
- Chấm lại (US-GD-05); số đo (US-GD-06).

### Phụ thuộc
- US-GD-02, 03; `/assignments/[id]` (FEAT-assignments 7.3); DESIGN §13.

---

## US-GD-05: Giảng viên muốn mọi bài chấm theo đúng một bản hướng dẫn, chấm lại được lô chưa công bố, và không bị công bố lẫn hai bản
Ưu tiên: Must · Ước lượng: L · Sprint: 10.6 · **L3** · M1 · **đụng điểm: Q-G-1, PM xác nhận P-106-1; siết bởi P-106-2**

### Tiêu chí nghiệm thu
- AC1 (prompt). Given lượt chấm ghim phiên bản `v ≥ 1` Then `BuildEssayPrompt` thêm **một khối** «Lưu ý của giảng viên (bản v)» sau rubric, nhóm theo tiêu chí, chỉ gồm lưu ý của phiên bản `v` (AC5 US-GD-02); khối nằm ngoài vùng «bài làm là dữ liệu không tin cậy»; lưu ý chỉ nói về **điểm từng tiêu chí**; mọi kiểm bằng code ở 4.6.6 (điểm ≤ max, bước 0,25, tổng, trích dẫn, muộn, làm tròn) **không đổi**.
- AC2 (nhất quán). Given hai bài có nội dung giống hệt, cùng `guidance_version` Then nhận cùng khối lưu ý (không recall theo độ tương tự bài làm; `TestGuidanceBlockDeterministic`).
- AC3 (nhãn cũ). Given `assignments.guidance_version = 3` và có kết quả hiện hành **chưa công bố** ghim `< 3` Then `/grading` hiện «N bài chấm theo bản cũ · Chấm lại» (`InlineNotice`, theo bài tập) với N = số bài **còn nháp và chưa ai sửa** (chấm lại được); kèm «M đang sửa dở» nếu có bài nháp đã có tiêu chí giảng viên sửa (không chấm lại hàng loạt; duyệt hoặc chấm lại từng bài, có xác nhận); bài **đã duyệt** (R) **không** tính vào N (P-106-2). Số lấy từ `GET …/guidance` (`stale:{drafts, edited_drafts, reviewed}`).
- AC4 (chấm lại lô — không đụng điểm giảng viên; P-106-2). Given Staff bấm `Chấm lại` When `POST …/assignments/{aid}/regrade-stale` (`Idempotency-Key`; không tham số phạm vi) Then tạo việc `round + 1` **chỉ** cho kết quả hiện hành `status = DRAFT`, **không** có tiêu chí giảng viên đã sửa (`edited = false` mọi tiêu chí), **chưa từng được duyệt (`first_reviewed_at IS NULL`; P-106-5)**, chưa công bố, `guidance_version < hiện hành`, bài nộp còn là bản mới nhất; **không bao giờ** chọn bài `REVIEWED` hay bài đã có điểm giảng viên sửa tay (không ghi đè điểm người đã chốt); `202 {job_id, queued, skipped:{reviewed, edited}}`; tiến độ `job.progress`; tối đa `REGRADE_MAX_BATCH` (200) mỗi lần; `round` > 20 → bỏ qua bài đó và báo. Chấm lại **một** bài đang sửa dở dùng đường chấm một bài sẵn có (`POST …/submissions/{sid}/grade`) kèm `ConfirmIrreversible` (điểm đã sửa nằm lại ở round cũ).
- AC5 (không đụng cái đã công bố). Given kết quả `PUBLISHED` Then **không bao giờ** bị chấm lại theo đường này; điểm đã công bố giữ nguyên; muốn đổi phải qua phúc khảo như hiện hành.
- AC6 (so phiên bản khi công bố — miễn cho bài giảng viên đã duyệt; P-106-2). Given `POST …/grading-results/publish` Then thứ tự: (1) mọi bài phải `REVIEWED` (`422 NOT_REVIEWED` như hiện hành); (2) **bài đã duyệt** (`first_reviewed_at IS NOT NULL` — ghi một lần, không bị xoá khi sửa lại; P-106-5) **miễn so phiên bản** — điểm cuối là quyết định của giảng viên (sửa tay hoặc duyệt nguyên nháp), không phải của bản lưu ý; (3) lưới an toàn: tập công bố có kết quả **chưa từng được duyệt** (`first_reviewed_at IS NULL`) (hiện không đưa vào được vì (1), giữ cho mọi đường công bố tương lai) mà `guidance_version` khác `assignments.guidance_version` hoặc lẫn nhiều bản → `409 MIXED_GUIDANCE {versions, counts}`, không công bố gì. Bài còn nháp (chưa duyệt) vẫn phải cùng phiên bản hiện hành → chấm lại. (Q-G-1, PM xác nhận P-106-1; siết bởi P-106-2.)
- AC6a (màn công bố nêu số bài miễn). Given công bố lô có bài chấm theo bản cũ đã duyệt Then phản hồi `{published, skipped, exempt:{EDITED, ACCEPTED_AS_IS}}` và hộp xác nhận công bố hiện một dòng «k bài chấm theo bản cũ: a đã sửa tay, b duyệt nguyên nháp» (`EDITED` = có tiêu chí giảng viên sửa; `ACCEPTED_AS_IS` = duyệt nguyên nháp); không chặn công bố; ghi vào `audit_log` của lần công bố.
- AC7 (đang chấm lại). Given đang có việc `regrade-stale` chạy Then nút `Chấm lại` khoá («Đang chấm lại …»); chấm lại lần hai cùng khoá không nhân đôi việc (`Idempotency-Key` + `UNIQUE (course_id, submission_id, round)`).
- AC8 (lịch sử). Given bài chấm lại Then kết quả round cũ giữ nguyên (xem được ở lịch sử), `guidance_version` mỗi round khác nhau hiển thị ở chi tiết; `ai_scores` / `ai_comments` mỗi round bất biến.
- AC9 (nhánh lỗi). Given worker chết giữa lô → việc được nhận lại, không chấm đôi (như 4.7); LLM lỗi → bài `FAILED` theo 4.7.4, các bài khác tiếp tục; cờ tắt giữa chừng → việc chưa nhận chạy với `guidance_version` hiện hành (0).
- AC10 (phân quyền). Given STUDENT → `403`; TA / TEACHER lớp → `202`; lớp khác → `404`.
- AC11 (không đổi điểm tổng, không ghi đè điểm giảng viên). Given so sánh trước / sau chấm lại Then `late_penalty`, làm tròn, `total` theo quy tắc hiện hành; **không** có đường chấm lại hàng loạt nào đổi `criteria` / `total` của kết quả `REVIEWED` hoặc đã có tiêu chí giảng viên sửa; `TestRegradeKeepsLatePolicy`, `TestRegradeNeverOverwritesTeacherScores`.

### Ngoài phạm vi của story này
- Chấm lại bài đã công bố; thông báo cho sinh viên (không có: bài chưa công bố sinh viên không thấy gì).

### Phụ thuộc
- US-GD-02, 04; FEAT-assignments 4.7, 4.8.5; `round` ≤ 20.

---

## US-GD-06: Giảng viên muốn biết lưu ý có giúp không: số đo trước / sau và nhãn trên từng tiêu chí khi duyệt bài
Ưu tiên: Should · Ước lượng: M · Sprint: 10.6 · **L4** · M1

### Tiêu chí nghiệm thu
- AC1 (số đo theo phiên bản). Given `GET …/assignments/{aid}/guidance/stats` (TEACHER, TA) Then theo từng `guidance_version`: số bài, **MAE điểm tiêu chí** và **MAE tổng** (|điểm nháp AI − điểm giảng viên chốt|, `decimal`), **lệch trung bình có dấu** (chống chấm gắt quá tay), số tiêu chí bị sửa tay, tỉ lệ bài sửa ≥ 1 điểm; chỉ tính kết quả `REVIEWED` / `PUBLISHED`; nhỏ hơn 5 bài → `—`.
- AC2 (hiển thị). Given khối «Lưu ý chấm» Then một bảng nhỏ phiên bản × (bài, sai số tiêu chí, lệch) dưới danh sách; **không** biểu đồ ở khung nhìn đầu; số chỉ cho nhân viên.
- AC3 (nhãn trên tiêu chí). Given `/grading/[submissionId]` Then nhãn mỗi tiêu chí có phụ đề `meta` «bản v2 · 3 lần sửa» **chỉ khi** bài chấm bằng `v ≥ 1` và tiêu chí đó có lưu ý áp dụng; bấm mở popover liệt kê các lưu ý áp dụng (chữ của lưu ý); bài `v = 0` → không phụ đề.
- AC4 (không tính lẫn). Given bài chấm lại Then thống kê dùng kết quả **hiện hành** của từng bài, ghi rõ phiên bản; round cũ không tính.
- AC5 (hiệu năng). Given 1.000 kết quả Then API p95 ≤ 500 ms (một truy vấn gom, chỉ mục `course_id, assignment_id`).
- AC6 (phân quyền). Given STUDENT → `403`; lớp khác → `404`.
- AC7 (rỗng / lỗi). Given chưa đủ dữ liệu → `—`, không biểu đồ rỗng; API lỗi → `Thử lại`.

### Ngoài phạm vi của story này
- Đo trên bộ E3 (US-GD-10); quyết định thay giảng viên.

### Phụ thuộc
- US-GD-02, 04, 05.

---

## US-GD-07: Đội muốn mỗi lần giảng viên sửa trở thành một ca kiểm thử hồi quy, để đổi prompt hoặc model không làm lùi
Ưu tiên: Should · Ước lượng: S · Sprint: 10.6 · **L5** · M2 · **đụng dữ liệu cá nhân: Q-G-3, PM xác nhận P-106-1**

### Tiêu chí nghiệm thu
- AC1 (ghi ca theo tham chiếu). Given lớp bật `REGRESSION_CAPTURE` (chỉ ADMIN bật) When giảng viên `review` một kết quả có tiêu chí sửa ≥ `REGRESSION_MIN_DELTA` (1,0), hoặc phúc khảo `ADJUSTED`, hoặc bài Threads chuyển `CORRECTED` / `REJECTED` Then ghi một hàng `grading_regression_cases(kind, source_id, criterion_id?, expected jsonb)` với **chỉ số / enum** (điểm AI, điểm giảng viên, chiều, id rubric, băm rubric) và **tham chiếu** `source_id`; **không** chứa văn bản bài, câu hỏi, câu trả lời, tên, MSSV.
- AC2 (idempotent). Given cùng `(kind, source_id, criterion_id)` Then một hàng (`UNIQUE`); sửa lại thì cập nhật `expected` mới nhất; ghi ca **không** làm chậm `review` (cùng giao dịch, một `INSERT … ON CONFLICT`).
- AC3 (chạy hồi quy ở nơi dữ liệu nằm). Given `cmd/regress run --assignment <id> [--prompt-ref …]` (CLI vận hành, môi trường đo, ADMIN) Then đọc ca + văn bản bài **ngay trong DB**, chấm lại bằng prompt / model hiện hành qua 9router, so với `expected.teacher_score`, in báo cáo `{n, MAE, lệch TB, số tiêu chí xấu đi, số tốt lên}`; **văn bản không ra khỏi DB / máy chạy CLI**, không ghi vào repo.
- AC4 (không cho lùi). Given `benchmarks/guidance/baseline.json` Then báo cáo so với đường cơ sở: MAE tăng quá `REGRESSION_MAX_REL` (10 %) **hoặc** có tiêu chí vốn đúng (|Δ| ≤ 0,25) nay sai > 0,5 → mã thoát ≠ 0 «Có lùi»; đổi prompt / model phải chạy lệnh này (ghi vào guide và checklist PR); không chạy thật trong CI (CI chỉ kiểm cấu trúc ca bằng provider `fake`).
- AC5 (xuất bản đã che — mặc định **tắt**). Given `REGRESSION_EXPORT_ALLOWED=false` (mặc định) Then `cmd/regress export` từ chối. (Q-G-3, PM xác nhận P-106-1) nếu PM cho phép (`true`): ghi JSONL **ngoài repo**, văn bản qua `privacy.Mask` + `Detect` (khớp còn sót → bỏ ca), id băm, không `student_id`; mỗi lần xuất ghi `audit_log` (ai, bao nhiêu ca); không có đường xuất qua HTTP.
- AC6 (phân quyền). Given chỉ ADMIN đọc / xuất ca; Teacher chỉ thấy **số ca** — trường `regression_cases` (số nguyên, đếm theo bài tập: ca `GRADING_EDIT` / `GRADING_APPEAL` có kết quả nguồn thuộc bài tập đó) trong response `GET …/guidance/stats` (#8), **không endpoint mới** (P-106-10); TA đọc được con số này như phần còn lại của #8 nhưng không đọc / xuất được ca; STUDENT → `403`. Xuất ca và chạy `cmd/regress` chỉ ADMIN.
- AC7 (xoá). Given xoá dữ liệu một sinh viên / lớp Then ca liên quan bị xoá theo `ON DELETE` / job xoá (không còn tham chiếu mồ côi).
- AC8 (kiểm). Given test Then `TestRegressionCaseNoText` (cột / `expected` không có chuỗi văn bản), `TestRegressionCaseIdempotent`, `TestRegressCLIRefusesExport`.

### Ngoài phạm vi của story này
- Đưa ca vào CI chạy LLM thật; E2 từ nhãn người (golden set E2 dựng tay, `benchmarks/e2`).

### Phụ thuộc
- US-GD-01, 02; FEAT-assignments 4.8, 4.10; Threads (F4); E2 / E3 (`benchmarks`).

---

## US-GD-08: Sinh viên muốn thấy câu hỏi tương tự đã có đáp án được xác nhận ngay khi soạn câu hỏi trên Threads, để có câu trả lời tức thì
Ưu tiên: Should · Ước lượng: S · Sprint: 10.6 · **L5** · M4 · F4

### Tiêu chí nghiệm thu
- AC1 (khi nào). Given lớp bật `THREAD_SUGGEST` When sinh viên đang soạn tiêu đề ≥ 3 từ ở `/threads` (debounce 600 ms) Then `POST …/threads/similar-confirmed {q}` (chỉ đọc, không cần `Idempotency-Key`; P-106-3) trả tối đa **3** thread **công khai** cùng lớp, câu trả lời AI đã `VERIFIED` / `CORRECTED`, chưa `Ẩn`/`Loại`, độ tương tự ≥ `THREAD_SUGGEST_MIN` (0,80, cosine); xếp theo độ tương tự giảm dần.
- AC2 (nguồn tin cậy). Given nguồn Then chỉ thread **đăng công khai** qua tường lửa (`knowledge_items.source_type='POST'`); **không** dùng mục `TICKET` (nguồn từ kênh riêng) để hiện ra công khai. (Q-G-5, PM xác nhận P-106-1) mặc định chỉ `POST`.
- AC3 (hiển thị). Given có kết quả Then một danh sách dưới ô tiêu đề: tiêu đề thread (liên kết mở `/threads/[id]`) + `Đã xác nhận`; không đoạn giải thích; không đẩy ô đăng xuống quá 3 dòng; không có kết quả → **không hiện gì** (không «không tìm thấy»).
- AC4 (không chặn đăng). Given sinh viên vẫn bấm đăng Then đăng bình thường (gợi ý không bắt buộc, không hộp thoại); luồng tường lửa / gộp thread tương tự (`THREAD_SIMILAR_MIN`) giữ nguyên.
- AC5 (hiệu năng và chi phí). Given mỗi lần gợi ý Then đúng **một** lần nhúng câu đang soạn (không LLM sinh chữ) + một truy vấn SQL theo `course_id` (HNSW, lọc trong cùng truy vấn); p95 ≤ 300 ms với embedding giả; rate limit 30 lần / phút / người; không gọi khi cờ tắt; dùng lại cache nhúng nếu cùng chuỗi.
- AC6 (đúng và mới). Given tài liệu nguồn của đáp án đã đổi (`document.changed`) Then mục kho tri thức liên quan đã bị đánh dấu lỗi thời theo cơ chế sẵn có và **không** được gợi ý; thread bị ẩn sau đó biến mất khỏi gợi ý trong ≤ 60 s.
- AC7 (riêng tư). Given câu đang soạn Then **không** lưu, không ghi `pii_events`, không gửi LLM; câu **nằm trong thân `POST`, không nằm trong URL** (bản nháp chưa qua tường lửa có thể chứa MSSV / tên; access log của Caddy ghi nguyên URI nên không dùng `GET ?q=`); thân yêu cầu **không bị ghi log** ở cả Caddy và log ứng dụng (`TestSimilarNoBodyLog`), phản hồi `Cache-Control: no-store`; tường lửa PII lúc gõ (`threads/precheck`) vẫn chạy độc lập; gợi ý không tiết lộ thread của lớp khác (`course_id` trong truy vấn).
- AC8 (phân quyền). Given STUDENT, TA, TEACHER của lớp → `200`; ngoài lớp → `404`; không đăng nhập → `401`.
- AC9 (nhánh lỗi). Given lỗi nhúng / DB → bỏ qua im lặng (không hiện lỗi, đăng vẫn được).
- AC10 (di động). Given 375 px Then danh sách một cột, vùng chạm ≥ 44 px.

### Ngoài phạm vi của story này
- Tự trả lời bằng câu cũ; gợi ý trong chat riêng; gợi ý từ nguồn ngoài lớp.

### Phụ thuộc
- F4, `knowledge_items`, embedding (`internal/rag`), US-GD-01.

---

## US-GD-09: Giảng viên muốn câu hỏi AI sinh lần sau tránh các lỗi mà tôi từng loại, để tỉ lệ duyệt nguyên cao hơn
Ưu tiên: Should · Ước lượng: S · Sprint: 10.6 · **L5** · M6 · F12

### Tiêu chí nghiệm thu
- AC1 (ghi lý do loại). Given Teacher / TA `Loại` câu (một câu hoặc `bulk`) Then có thể **chọn tuỳ ý** một lý do trong danh sách đóng: `MO_HO`, `SAI_DAP_AN`, `TRUNG_LAP`, `LAC_DE`, `KHO_DE_BAT_THUONG`, `SAI_DINH_DANG` (nhãn hiển thị tiếng Việt); không chọn = `UNSPECIFIED`; lưu `question_bank.reject_reason`; `POST …/questions/bulk {action:"reject", reason?}` và `PUT …/questions/{id}/review` nhận `reject_reason?`; lý do lạ → `422`.
- AC2 (giao diện một chạm). Given `/questions` Then `Loại` vẫn là **một thao tác**; lý do là dãy nhãn chọn nhanh hiện sau khi loại (có hoàn tác 5 giây), bỏ qua được; không hộp thoại bắt buộc.
- AC3 (thống kê tất định). Given lớp bật `QUESTION_HINTS` When `GenerateQuestions` chạy Then code (SQL) đếm câu `REJECTED` có lý do ≠ `UNSPECIFIED` của lớp trong 90 ngày; chỉ khi ≥ 5 câu; lấy tối đa 3 lý do nhiều nhất.
- AC4 (vào prompt — chỉ nhãn). Given thống kê Then prompt sinh thêm một khối «Tránh các lỗi sau» gồm **câu cố định** ánh xạ từ nhãn (ví dụ `MO_HO` → «Mỗi câu chỉ một cách hiểu; tránh từ chung chung»); **không** chèn nội dung câu bị loại, không văn bản tự do của giảng viên (không PII, không chèn lệnh).
- AC5 (không đổi kiểm). Given kiểm bằng code của `GenerateQuestions` (một đáp án đúng, đáp án nằm trong lựa chọn, `citations` bắt buộc, `origin=GENERATED`, duyệt trước khi tới sinh viên) Then **không đổi**; câu sinh ra vẫn `PENDING`.
- AC6 (đo). Given `gen-stats` Then thêm `hints:[reason…]` cho job (nhân viên thấy lý do đã dùng); E5 so tỉ lệ duyệt nguyên trước / sau (báo cáo, không bắt buộc đạt).
- AC7 (nhánh lỗi). Given cờ tắt / chưa đủ 5 câu / lỗi thống kê Then prompt **như trước** (không khối), việc sinh không lỗi.
- AC8 (phân quyền). Given STUDENT → `403`; chỉ Staff lớp thấy lý do loại; lớp khác → `404`; sinh viên không bao giờ thấy lý do loại.

### Ngoài phạm vi của story này
- Học từ câu bị sửa (`edited`); sinh lý do bằng LLM.

### Phụ thuộc
- FEAT-practice (`GenerateQuestions`, `bulk`, `gen-stats`); `question_bank`; US-GD-01.

---

## US-GD-10: Đội muốn seed, cổng nghiệm thu, đo trên bộ E3 và kiểm chứng với «Giáo viên», để quyết định bật theo số liệu
Ưu tiên: Must · Ước lượng: L · Sprint: 10.6 · **L6**

### Tiêu chí nghiệm thu
- AC1 (seed). Given stack trống When `node scripts/seed.mjs` Then provider `fake` có sẵn; bài tập BT03 lớp 1 có ≥ 8 kết quả `REVIEWED` với ≥ 4 tiêu chí bị sửa ≥ 0,5 theo một «sở thích» (ví dụ c2 chấm gắt hơn); `SEED_GUIDANCE_DEMO=1` bật `GRADING_GUIDANCE` cho lớp 1 và nạp bản nháp lưu ý mẫu; bài tập khác để nguyên; `scripts/check-guidance-seed.mjs` xanh; chạy lại idempotent.
- AC2 (cổng). Given repo When chạy:
  - `cd backend-go && go test ./internal/grading -run 'TestGuidanceSetAtVersion|TestEssayPromptUnchangedAtV0|TestGuidanceBlockDeterministic|TestReflect|TestRegradeKeepsLatePolicy' -v`
  - `cd backend-go && go test ./internal/assignment ./internal/guidance -run 'TestPublishMixedGuidance|TestApplyVersion|TestRegradeStale|TestGuidanceRBAC' -v`
  - `cd backend-go && go test ./internal/thread ./internal/practice -run 'TestSimilarConfirmed|TestRejectReasonHints' -v`
  - `pnpm -C frontend exec playwright test grading-guidance.spec.ts thread-suggest.spec.ts question-hints.spec.ts`
  Then tất cả xanh (provider `fake`, không gọi mạng thật).
- AC3 (E2E `grading-guidance.spec.ts`). Given lớp bật cờ, seed có lệch When (a) `Rút lưu ý` → có nháp, mỗi mục `n lần sửa`; (b) `Duyệt` hai, `Bỏ` một; (c) `Áp dụng` → bản v1; (d) `/grading` hiện «N bài chấm theo bản cũ»; (e) `Chấm lại` → các bài nháp chưa sửa ghim v1, bài đã duyệt / đã sửa tay **không bị đụng**; (f) công bố bài đã duyệt chấm theo v0 → được, hộp xác nhận nêu «k bài chấm theo bản cũ: a đã sửa tay, b duyệt nguyên nháp»; nhánh lỗi: LLM lỗi lúc rút → «Chưa rút được lưu ý»; TA bấm `Duyệt` → `403`; sinh viên mở `/assignments/[id]` → không thấy gì về lưu ý.
- AC4 (hợp đồng và xoá mock). Given `openapi.yaml` Then có mọi endpoint SRS 6; `go test ./internal/contract/...` xanh; mock liên quan nếu có bị xoá cùng đợt; `bash scripts/ui-antipatterns.sh` sạch.
- AC5 (đo trên E3 — kiểm chứng; cần nhãn người). Given bộ E3 (≥ 60 bài, nhãn người) hoặc nhãn của «Giáo viên» tạm (kết quả ghi rõ nguồn nhãn) When chạy `benchmarks/guidance/run.py` (HTTP, chỉ thư viện chuẩn) theo giao thức SRS 9: lưu ý rút từ 20 bài đầu (qua duyệt), đo trên 40 bài sau, bài tập thứ hai cùng lớp Then báo cáo `benchmarks/reports/guidance-before-after.{json,md}` + SVG: MAE tiêu chí / tổng trước và sau, **lệch trung bình có dấu**, số tiêu chí xấu đi, số lần sửa tay (đề xuất), nguồn nhãn, cỡ mẫu, mục «Giới hạn».
- AC6 (tiêu chí quyết định bật). Given báo cáo Then dòng kết luận **Bật được / Giữ tắt**: MAE tổng giảm ≥ 20 % **và** |lệch trung bình| không xấu hơn đường cơ sở quá 0,20 điểm **và** số tiêu chí vốn đúng nay sai ≤ 1,5× đường cơ sở; không đạt → giữ tắt, không nới ngưỡng. PoC (mô phỏng) giảm 49 % chỉ là tham chiếu, **không** là cam kết.
- AC7 (kiểm chứng với «Giáo viên», nếu PM giao). Given PM dựng agent tạm Then QC ghi lại: ai là nguồn nhãn, tiêu chí duyệt / bỏ từng lưu ý, nhãn S7 (Q-G-S7); kết quả không dùng làm nhãn người thật; PM tắt agent sau kiểm chứng.
- AC8 (hướng dẫn vận hành). Given `docs/guides/grading-guidance.md` Then có: bật cờ, khi nào bấm `Rút lưu ý`, cách duyệt, `Áp dụng`, `Chấm lại`, đọc số đo, lùi (tắt cờ), quên / xoá dữ liệu, giới hạn, chi phí (số lời gọi mỗi lần rút / chấm lại); kiểm: QC làm theo từng bước.
- AC9 (tài liệu). Given `ARCHITECTURE.md`, `SYSTEM_DESIGN.md`, `DECISIONS.md` D61, D62 khớp mã; FLOWS F9, F4, F12 rà lại sau khi có mã; `PROGRESS.md` ghi Nợ (M3, M5, M7, M8 để sau).
- AC10 (nhánh lỗi). Given `benchmarks/guidance/run.py` khi gateway không chạy → thoát ≠ 0 «Gateway chưa chạy», không ghi báo cáo rỗng; thiếu nhãn người → `WAITING_HUMAN`, không chạy giả.

### Ngoài phạm vi của story này
- Bench 1.000 bài chung và sprint 11 (chạy sau 10.6); M3, M5, M7, M8; điểm tổng.

### Phụ thuộc
- US-GD-01…09; E3 (`benchmarks`), `docs/specs/FEAT-assignments/fixtures`.

---

## US-GD-11: Chủ dự án muốn chắc chắn «học từ lần sửa» không đụng điểm, không lộ dữ liệu sinh viên và không tự thay giảng viên
Ưu tiên: Must · Ước lượng: S · Sprint: 10.6 · **xuyên suốt**

### Tiêu chí nghiệm thu
- AC1 (không tự áp). Given mọi đường mã Then không có hàm nào đổi `assignments.guidance_version` ngoài `POST …/guidance/apply` của TEACHER (`TestGuidanceVersionOnlyViaApply`); job `reflect`, worker chấm, xoá dữ liệu sinh viên **không** đổi được.
- AC2 (không điểm tổng). Given schema lưu ý và prompt Then không có tổng / trừ muộn / làm tròn; từ khoá cấm (US-GD-03 AC6 iv) áp cả lúc sửa tay; `TestGuidanceNoTotalsInPrompt`.
- AC3 (không văn bản bài, không danh tính). Given đầu vào `reflect` và mọi bảng mới Then không văn bản bài, không `student_id`, không tên / MSSV / email; `source_result_ids` là id kết quả, **không** map ngược ra sinh viên ngoài người có quyền xem kết quả đó; log không PII.
- AC4 (cách ly). Given hai lớp / hai bài tập Then lưu ý của bài A **không** vào prompt bài B (khoá `(course_id, assignment_id)`); `TestGuidanceIsolation`.
- AC5 (sinh viên không thấy). Given mọi DTO phía STUDENT (`/assignments/[id]`, kết quả đã công bố, `/threads`, thông báo) Then không trường / chữ về lưu ý, phiên bản, cờ, số đo; kiểm hợp đồng.
- AC6 (đường lùi). Given tắt cờ hoặc `GUIDANCE_ENABLED=false` Then hệ thống như trước 10.6, kể cả prompt (byte-for-byte); dữ liệu còn để đo.
- AC7 (xoá dữ liệu sinh viên). Given xoá / ẩn danh một sinh viên Then id kết quả của em được bỏ khỏi `source_result_ids`; mục còn `support < 2` đánh dấu «Cần xem lại» cho giảng viên (không tự nghỉ, không tự đổi hành vi chấm); ca hồi quy liên quan xoá; (Q-G-6, PM xác nhận P-106-1): mặc định như vậy.
- AC8 (không bộ nhớ ngoài). Given repo Then không thư viện / dịch vụ bộ nhớ agent (Hindsight…); `go.mod` / `package.json` không thêm mục ngoài bảng `ARCHITECTURE.md`.
- AC9 (kiểm toán). Given mọi quyết định (duyệt, bỏ, áp dụng, nghỉ, bật cờ, xuất ca) Then có dòng `audit_log` (ai, gì, trước / sau); `reflect` có `llm_audit`.

### Ngoài phạm vi của story này
- Chính sách giữ dữ liệu toàn hệ thống (PR).

### Phụ thuộc
- US-GD-01…09; D61, D62; AGENTS.md.
