# Sprint 6: P3 Hai kênh + PII, P8 Tài liệu + Lịch

**Trạng thái:** **chủ dự án duyệt 2026-10-10**; nút "Nhờ giảng viên" ẩn tới sprint 7 (chủ dự án chốt).
**Nhánh và nơi làm:** `sprint/6-p3-p8` tách từ `main` (`48315e1`), worktree `../TA_Agent_v2-s6`.

## Mục tiêu
- **P3 (M1, M2):** câu hỏi cá nhân không lọt ra Threads. Tên và MSSV không bao giờ rời hệ thống tới LLM. Sinh viên hỏi được dữ liệu của chính mình qua tool lấy danh tính từ JWT.
- **P8 (M9, M10, M11):**
  - tài liệu được trích và nhúng bằng việc nền (`docling-serve`);
  - thư viện có tìm kiếm;
  - lịch có feed ICS và nhắc trước 24 h.
- **Luồng đi trọn:** F3 (chat riêng), F6 (Threads + tường lửa), F13 (tài liệu → hỏi AI).
- **Cổng:** `gate P3`, `gate P8`.
- **Quyển đồ án:** sau khi merge, `writer` viết phần sprint 6 (`docs/team/PM.md` §3.5 bước 9); PM + BA review.

Ghép P3 với P8 để chat có tài liệu thật cho RAG (ROADMAP "Vì sao ghép"). Vì vậy **không làm "đường nạp tối thiểu" tạm của P3 L0**: ingest nền của P8 làm trước, P3 dùng luôn.

## Story (lát dọc)

| # | Story | Lát | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 0 | **Tech Lead PoC** (trước US-P8-01): xem mục PoC bên dưới | – | S | – |
| 1 | **US-P3-01** Migration `00007_chat_threads`: `chat_sessions`, `chat_messages` (có `partial_content`, `stream_status`), `forum_threads`, `forum_posts` (có `hidden_at`, `hidden_reason` cho P4), tag. `00008_privacy`: `pii_events`. Store sqlc | P3 L0 | M | P2 |
| 2 | **US-P8-01** Ingest nền + `internal/rag`: upload → 202 + job → worker gọi `docling-serve` → chunk → embed qua `internal/llm` → `content_chunks`. Tiến độ qua SSE, idempotent theo `content_hash`. Tìm lai vector + từ khoá bằng SQL, **lọc `audience` ngay trong truy vấn**, `ANSWER_KEY` không bao giờ trả về (`TestAnswerKeyNeverRetrieved`). Seed nạp `seed/documents/*.pdf` | P8 L1 | L | 0, P1 |
| 3 | **US-P3-02** `internal/privacy`: `detect` / `redact` (regex MSSV / email / SĐT / CCCD + từ điển roster theo lớp: có dấu, không dấu, đảo thứ tự; cache Redis xoá theo sự kiện thành viên), `mask` / `unmask` / `unmask_stream` (buffer biên; ánh xạ Redis theo phiên, TTL 24 h, không log). `TestUnmaskStream` cắt token ở mọi vị trí | P3 L1, L3 | L | 1 |
| 4 | **US-P3-03** Cắm `mask` ở **một chỗ** trong `internal/llm` (tin nhắn, lịch sử, kết quả tool) và `unmask_stream` ngay trước SSE. Thêm bộ quét placeholder sót, `llm_audit.pii_masked_count`, `TestNoPayloadLeak` (0 MSSV / họ tên roster trong payload gửi provider giả) | P3 L3 | M | 3 |
| 5 | **US-P3-04** Phân loại kênh (luật + độ tương đồng embedding, **một lần mỗi tin nhắn**, D47 mục 2); `internal/agent` định tuyến tất định → tool Go → **một** lần sinh chữ. Tool theo kênh: agent Threads không có tool cá nhân (có test chứng minh). 6 tool cá nhân không có tham số danh tính, đọc `trusted_context`; tool của P5 / P6 trả "chưa có dữ liệu". Tắt semantic cache cho intent cá nhân. Test "hỏi hộ người khác bị từ chối" | P3 L2 | L | 2, 4 |
| 6 | **US-P3-05** Chat riêng `/chat` (thay mock):<br>• `internal/chat`, stream SSE (sự kiện đầu ≤ 300 ms);<br>• `partial_content` ≈ 1 s, `stream_status`; tải lại giữa chừng không mất; nút Dừng huỷ tới provider;<br>• `OVERLOADED` → thời gian chờ + Thử lại;<br>• dòng "Đã ẩn N thông tin cá nhân trước khi gửi cho AI";<br>• **khoá trong giờ thi** (`ep:exam_lock` của PE, nợ PE (6));<br>• 375 px | P3 L0, L3b | L | 5 |
| 7 | **US-P3-06** Threads `/threads` (thay mock):<br>• `internal/thread`; AI trả lời một lần, có trích nguồn;<br>• giảng viên / trợ giảng Xác nhận / Sửa / Loại; thread tương tự;<br>• `precheck` (debounce 800 ms) + Dialog bảo vệ đúng hai lối `Chuyển sang chat riêng` / `Ẩn thông tin rồi đăng`;<br>• `POST /chat/sessions/from-draft`; ghi `pii_events`; nháp tự lưu | P3 L1, L3b | L | 5, 6 |
| 8 | **US-P3-07** Confidence (rerank + groundedness + tool) trong `ResponseMetadata`; sinh viên không thấy con số. Bộ dữ liệu E1 200 bài gắn nhãn (`benchmarks/pii/`) + `benchmarks/eval_pii.py` (recall ≥ 0,95, chặn nhầm ≤ 0,05) | P3 L4 | M | 3, 7 |
| 9 | **US-P8-02** `/documents` (Staff): bảng, upload bằng vùng thả tại chỗ, cờ `use_for_rag` / `visible_to_students`, nhắc thiếu `COURSE_POLICY`, sửa chunk, thống kê. `/library`: tìm trước, xem trước PDF, đếm lượt tải, "Hỏi AI về tài liệu này" (chat giới hạn `document_id`), tool `search_library` | P8 L1, L2 | L | 2, 6 |
| 10 | **US-P8-03** Lịch:<br>• migration `00009_calendar` (`calendar_events`, `reminder_log`);<br>• buổi học (`class_sessions`) và bài thi (PE) ghép vào bằng UNION, không nhân bản; hạn bài tập nối ở P7;<br>• `/calendar` tháng / tuần / danh sách (điện thoại mặc định danh sách), `ETag`;<br>• feed ICS ký bằng `users.ics_token` (đã có từ `00001`);<br>• nhắc 24 h (chuông + mail) qua `reminder_log`;<br>• tool `get_exam_schedule`, `get_upcoming_events` | P8 L3 | L | 1 |
| 11 | **US-P3-08** Seed (hội thoại, thread mẫu, tài liệu đã nhúng). Provider "Hôm nay": `continue[]` của sinh viên (nợ PROGRESS) và thread chờ xác nhận. k6 `chat` (TTFT, sự kiện đầu). `gate-p3.sh`, `gate-p8.sh` | – | M | 1–10 |

**Thứ tự:** 0 ∥ 1 → 2 ∥ 3 → 4 → 5 → 6 → 7 → 8, rồi 9, 10 (10 chỉ cần 1, chen được khi dev chờ) → 11.

**Spec do BA viết:**
- `docs/specs/FEAT-private-chat-pii/`: P3, gồm cả Threads;
- `docs/specs/FEAT-docs-calendar/`: P8.

Tech Lead thẩm định cả hai trước khi `APPROVED`. TC ở `docs/sprints/6/qc/`.

## PoC của Tech Lead (story 0) → `docs/research/2026-10-1x-docling-arm64.md`
- **`docling-serve` trên colima arm64, chỉ CPU:**
  - image, RAM lúc nghỉ và lúc trích, thời gian cho 3 PDF trong `seed/documents/`;
  - cấu hình tối thiểu (tắt OCR / mô hình bảng nếu không cần).
  - Ngưỡng chấp nhận: RAM ≤ 3 GiB, mỗi PDF ≤ 60 s.
  - Không đạt thì đề xuất phương án, PM hỏi chủ dự án (D46 khoá `docling-serve`).
- **Mô hình nhúng:** 1536 chiều (khớp `content_chunks.embedding vector(1536)`) qua `internal/llm` (OpenAI `text-embedding-3-small` / Gemini). Provider giả cho test.
- **Đánh số migration:** phase file ghi `00005` / `00006` / `00013`, nhưng các số đó đã dùng. Sprint này dùng `00007`, `00008`, `00009`.

## Quyết định PM tự chốt (đổi được qua `proposals.md`)
- **Nút "Nhờ giảng viên hỗ trợ"** (khi AI chưa chắc) **chưa hiện ở sprint 6**: escalation là P4 (sprint 7). Không làm nút giả. Sprint 6 chỉ hiện câu "AI chưa đủ chắc chắn về câu này".
- **Lịch:** chỉ có buổi học + bài thi + sự kiện tự tạo. Hạn bài tập nối ở P7.
- **E1:** 200 câu gắn nhãn do BA / QC soạn từ dữ liệu mô phỏng (D44), không dùng câu hỏi sinh viên thật.

## Nợ đem vào
- **PE (6):** chat riêng tôn trọng khoá giờ thi. Story 6.
- **PROGRESS:** BA vá `PRD.md` §2 "Ngoài phạm vi" (còn ghi sandbox) và D3 → "thay bởi D55" qua `proposals.md`.
- **5.5 #2:** "Tiếp tục học" của sinh viên. Story 11.

## Rủi ro
- **RAM:** `docling-serve` nặng. Máy 18 GB và colima đang dùng chung với project khác (8 GiB). Cách xử lý: ingest chỉ chạy khi cần (profile compose); QC và dev không dựng hai stack cùng lúc; `docker volume prune -f` sau mỗi lượt.
- **Độ phủ PII:** NER đã bỏ (D46). Tên người ngoài danh sách lớp có thể lọt; ghi trung thực ở E1.
- **P3 lớn (ước lượng gốc 2 tuần + P8 1 tuần):** thứ tự cắt nếu trễ theo ROADMAP. #4 NER đã cắt; nếu cần cắt thêm thì cắt "thread tương tự" và thống kê tài liệu trước.
