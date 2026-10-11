# Prompt cho `ba`: spec sprint 6 (P3 + P8)

Làm ở worktree `/Users/kuro/Documents/TA_Agent_v2-s6`, nhánh `sprint/6-p3-p8` (tách từ `main`).

## Đọc
- `docs/sprints/6/plan.md`: 12 story, thứ tự làm, quyết định đã chốt (nút "Nhờ giảng viên" ẩn tới sprint 7), nợ đem vào.
- `docs/phases/P3.md`, `docs/phases/P8.md`.
- `docs/PRD.md`: M1, M2, M9, M10, M11, mục quyền, mục phi chức năng.
- `docs/FLOWS.md`: F3, F6, F13, kèm các nhánh lỗi.
- `docs/ARCHITECTURE.md`: các bảng, API, gói `chat`, `thread`, `privacy`, `agent`, `rag`, `ingest`, `document`, `calendar`.
- `docs/SYSTEM_DESIGN.md` §3.2 (stream), §5 (SLO).
- `docs/DECISIONS.md`: D15, D44, D46, D47, D56 (khoá chat trong giờ thi).
- `docs/design/DESIGN.md`:
  - §14 cho `/chat`, `/threads`, `/documents`, `/library`, `/calendar`;
  - §13 cho lời văn, vì sinh viên không được thấy từ kỹ thuật AI;
  - quy tắc panel D59.
- `docs/UX.md` mục 6.
- Spec đã có, chỉ đọc: `FEAT-llm-gateway`, `FEAT-course-foundation`, `FEAT-weekly-exam` (`ep:exam_lock`).

## Việc
Viết hai feature:
- `docs/specs/FEAT-private-chat-pii/{US,SRS,QUESTIONS}.md` gồm US-P3-01…08. Story #11 của plan là US-P3-08.
- `docs/specs/FEAT-docs-calendar/{US,SRS,QUESTIONS}.md` gồm US-P8-01…03.

Mỗi story phải có ít nhất một AC nhánh lỗi và một AC phân quyền.

Bắt buộc có:
1. **Danh tính và tool.**
   - Tool cá nhân không có tham số danh tính; danh tính đọc từ `trusted_context` (JWT).
   - Agent của Threads không được đăng ký tool cá nhân.
   - Có AC "hỏi hộ người khác bị từ chối".
   - Tool của P5 / P6 trả "chưa có dữ liệu".
2. **Che danh tính.**
   - `mask` cắm ở một chỗ duy nhất trong `internal/llm`; `unmask_stream` chịu được token cắt ở mọi vị trí.
   - Ánh xạ lưu Redis, TTL 24 h, không log.
   - Có `TestNoPayloadLeak` và bộ quét placeholder sót.
   - Sinh viên không bao giờ thấy placeholder `[[…]]`.
3. **Tường lửa Threads.**
   - `precheck` debounce 800 ms.
   - Dialog chỉ có đúng hai lối.
   - Chuyển sang chat riêng giữ nguyên nội dung.
   - Ghi `pii_events`, không lưu nội dung cá nhân thô.
4. **D47.**
   - Mỗi tin nhắn chỉ phân loại một lần và chỉ có một lời gọi LLM sinh chữ.
   - Sự kiện SSE đầu tiên ≤ 300 ms.
   - Stream chịu mạng xấu: `partial_content`, tải lại trang không mất chữ, nút Dừng huỷ tới provider.
   - `OVERLOADED` hiện thời gian chờ ước tính.
5. **Khoá giờ thi.** Chat riêng bị khoá khi sinh viên đang có lượt làm bài mở (`ep:exam_lock`, D56). Đây là nợ của PE.
6. **RAG.**
   - Lọc `audience` ngay trong truy vấn SQL.
   - `ANSWER_KEY` không bao giờ được truy xuất cho AI của sinh viên.
   - Ingest là việc nền (202 + job + SSE), idempotent theo `content_hash`.
   - Giới hạn kích thước và loại tệp.
7. **Lịch.**
   - Ghép UNION với `class_sessions` và bài thi PE, không nhân bản dữ liệu.
   - Feed ICS ký bằng `users.ics_token`; token phải xoay được và lưu dạng băm, theo luật "token ở dạng băm". Kiểm xem cột hiện có lưu rõ hay băm; nếu cần đổi thì ghi vào SRS mục 10 cho PM quyết.
   - Nhắc trước 24 h, không gửi trùng (`reminder_log`).
8. **Confidence.** Sinh viên không thấy con số. Dưới ngưỡng chỉ hiện câu "AI chưa đủ chắc chắn về câu này", không có nút "Nhờ giảng viên" (chủ dự án chốt: ẩn tới sprint 7).
9. **E1.** Mô tả bộ 200 câu gắn nhãn (phân bố loại, nhãn, cách soạn từ dữ liệu mô phỏng) và chỉ số chấp nhận: recall ≥ 0,95, chặn nhầm ≤ 0,05.
10. **Migration.** Đánh số `00007`, `00008`, `00009`. Phase file ghi số cũ đã bị dùng, ghi chú điều này trong SRS.
11. **Nợ PROGRESS.** Mở một dòng `docs/sprints/6/proposals.md` đề nghị vá `PRD.md` §2 "Ngoài phạm vi" (còn ghi sandbox) và D3 → "thay bởi D55". Chỉ vá sau khi PM chấp nhận.
12. **`QUESTIONS.md`.** Câu nào đụng hành vi sản phẩm / quyền / dữ liệu cá nhân thì đánh **[CHỦ DỰ ÁN]**, kèm phương án mặc định và hệ quả.

Không mở subagent. Chỉ sửa `docs/specs/FEAT-private-chat-pii/**`, `docs/specs/FEAT-docs-calendar/**` và dòng proposals của bạn.

Xong thì commit `sprint 6: spec FEAT-private-chat-pii, FEAT-docs-calendar`, push, báo PM ≤ 5 dòng (số AC mỗi story, số câu [CHỦ DỰ ÁN]), rồi dừng. Sau đó Tech Lead thẩm định (`TL-REVIEW.md`).
