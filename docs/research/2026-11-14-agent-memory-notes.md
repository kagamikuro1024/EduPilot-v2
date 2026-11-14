# Ghi chép dở: bộ nhớ chung cho agent (Tech Lead cũ, 2026-11-14)

> Đây là ghi chép bàn giao, chưa phải tài liệu research. PM đã chuyển việc này cho `techlead-mem`. Tài liệu chính thức là `docs/research/2026-11-14-agent-memory.md`, do người nhận viết. Phần dưới chỉ ghi điều đã kiểm và nguồn của nó.

## 1. Hindsight (Vectorize): điều đã đọc
Nguồn: https://hindsight.vectorize.io/llms-full.txt (tải 14/11), README `github.com/vectorize-io/hindsight`, bài báo arXiv 2512.12818.

- **Giấy phép:** MIT. Ba model phụ cũng mở: `hs-decide-1-flash` (0,8B, Apache 2.0), `hs-rerank-1` (114M, MIT), `hs-embed-1` (112M, MIT, 384 chiều).
- **Máy chủ** viết bằng Python: gói pip `hindsight-api`, chạy FastAPI / uvicorn. Có Docker và Helm. Client có cho Python, Node.js, Go, CLI và MCP.
- **Lưu trữ:** Postgres + pgvector (hoặc Oracle 23ai). Có chế độ nhúng `pg0`.
- **retain** dùng LLM trích văn bản thành *fact* có kiểu: world, experience, observation, mental model, page. Fact gắn với entity và thời gian. Nó **không** lưu nguyên hội thoại. Một worker nền gom fact thành observation: khử trùng, có bằng chứng kèm trích dẫn, giữ lịch sử thay đổi.
- **recall:** 4 nhánh chạy song song (semantic, BM25, graph theo entity, temporal), hợp theo thứ hạng, rồi chấm lại bằng cross-encoder. Kết quả cắt theo **ngân sách token**, không theo top-k. Tài liệu ghi cross-encoder là nút cổ chai CPU.
- **reflect:** vòng lặp agent tối đa 10 lượt trên bank. Chỉ được trích dẫn ID đã truy xuất. Hành vi chỉnh bằng mission, directives và disposition.
- **Quên / xoá:** có `DELETE` cho documents, memories và bank. Re-retain cùng `document_id` sẽ upsert. Có webhook vòng đời.
- **Cách ly:** mỗi bank tách hẳn ("no cross-bank leakage"). Lọc thêm bằng metadata / tag.
- **Memory Defense** (tự bật): quét 45 mẫu secret / PII và prompt injection lúc retain, rồi che hoặc chặn; lý do ghi vào `security_events`.
- **Provider LLM** đặt qua `HINDSIGHT_API_LLM_PROVIDER`: openai, gemini, ollama, litellm… cùng `LLM_BASE_URL`. Như vậy trỏ được vào 9router (`[INFERENCE]`, chưa thử).
- **Benchmark** (tự công bố, Virginia Tech tái lập): LongMemEval 91,4 %, LoCoMo 89,61 %. Với backbone 20B mã mở: 39 % → 83,6 % so với đưa toàn bộ ngữ cảnh.

**Đụng luật EduPilot (đánh giá sơ bộ):**
- D46: máy chủ Python chạy thường trực.
- Nguyên tắc 3 / luật 11: Hindsight tự gọi LLM để trích fact và reflect, nên các lời gọi này không qua `internal/llm`, Scheduler, che PII hay `llm_audit`.
- D47: `reflect` là vòng lặp agent mở; `recall` có cross-encoder nằm trên đường nóng.
- PII: fact gắn entity (tên người). Memory Defense chỉ là regex.

## 2. Tín hiệu sửa sai đã có sẵn trong lược đồ (kiểm trên migration của main)
| Nguồn | Cột / bảng | Ghi chú |
| --- | --- | --- |
| GV sửa điểm nháp AI | `grading_results.ai_scores` (trigger cấm sửa sau khi ghi), `criteria` (điểm cuối), `deviation`, `teacher_changed_criteria[]`, `flags`, `confidence`, `pass_detail` (`00017`) | Tín hiệu mạnh nhất: có từng tiêu chí, có lệch |
| Phúc khảo | `grade_appeals.status` `UPHELD` / `ADJUSTED` (`00017`) | `ADJUSTED` nghĩa là điểm cuối sai |
| Công thức điểm | `grade_scheme_status` `DRAFT` → `CONFIRMED` / `SUPERSEDED` (`00016`) | So bản nháp LLM với bản GV xác nhận |
| Threads | `forum_posts.verification_state` `VERIFIED` / `CORRECTED` / `REJECTED` (`00007`) | Câu trả lời AI được GV / TA duyệt hoặc sửa |
| Chat | `chat_messages.feedback` `HELPFUL` / `NOT_HELPFUL` (`00007`) | Tín hiệu yếu, do sinh viên bấm |
| Escalation | `escalation_tickets.origin` `AUTO_LOW_CONFIDENCE` / `AUTO_NO_FORMULA` / `STUDENT_REQUEST`, cùng câu trả lời của GV (`00012`) | Cặp (câu hỏi đã che, đáp án GV) |
| Luyện đề | duyệt / sửa / bỏ câu sinh ra (`00019`, gen-stats) | Mẫu âm cho sinh câu hỏi |

## 3. Hướng đang nghiêng tới (chưa kiểm, để người nhận tự đánh giá)
- Với chấm điểm, "bài học" **không** nên là bộ nhớ ẩn. Nên biến nó thành **đề xuất sửa hướng dẫn rubric** có phiên bản, để GV duyệt, áp cho cả lượt chấm, không đổi giữa chừng. Làm vậy giữ được công bằng giữa sinh viên và giải thích được.
- Giá trị cao, rủi ro thấp: kho "đáp án đã duyệt" theo lớp, lấy từ Threads `VERIFIED` / `CORRECTED` và câu trả lời ticket mà GV đánh dấu dùng lại. Recall bằng SQL / pgvector, không gọi LLM, nên giữ được D47. Thêm: mỗi lần sửa sai thành một ca hồi quy trong golden set (phương án (c)).
- **PoC chấm "có / không bài học"** chưa làm được: chưa có điểm do người chấm (E3 đang `WAITING_HUMAN`). Nếu dùng nhãn thiết kế để đo thì vòng tròn, kết quả không có nghĩa.
