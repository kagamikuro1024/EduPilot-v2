# Đợt R — lỗi tìm được (smoke hồi quy F1–F19, main 6b98a36e)

Mức: **Cao** (luồng chính hỏng / mất dữ liệu / lộ dữ liệu), **TB** (nhánh lỗi hỏng, sai quyền, lệch AC), **Thấp** (giao diện, chữ, nhỏ).

| ID | Mức | Luồng | Lỗi | Bước tái hiện | Trạng thái |
| --- | --- | --- | --- | --- | --- |
| D-R-01 | TB | F4 Threads | Tường lửa PII chặn nhầm câu thường: «Em nghĩ virus cần vật chủ.» / «Mình nghĩ vậy» bị xếp `PERSONAL_QUESTION` → không đăng được bài/bình luận | sv.nguyco: `POST /courses/{id}/threads/precheck {"title":"Worm","body":"Em nghĩ virus cần vật chủ."}` → `allowed:false`, `reasons:[{type:PERSONAL_QUESTION}]` (câu «Cho em hỏi worm khác virus…» thì qua) | **Đạt** ở main `6b30be37` (kiểm lại: «Em nghĩ…»/«Mình nghĩ vậy» qua; 5 câu cá nhân vẫn chặn, kể cả không dấu; `scripts/w11_fix.py`) |
| D-R-02 | Thấp | Công cụ seed | `check-docs-seed` đỏ: `ANSWER_KEY = 2 (cần 1)` (seed Luyện đề thêm 1 đáp án). `check-exam-seed bank` đỏ: `MCQ_SINGLE APPROVED ≠ 12`, `MCQ_MULTI ≠ 4`, `TRUE_FALSE ≠ 4`, `lớp 761988 phải KHÔNG có câu hỏi (có 20)`; `demo` đỏ: `D thuộc 2 lớp (cần 0)`. Script kiểm lệch số liệu seed hiện hành (không do QC ghi dữ liệu) | Dựng ep-qc trống → seed (`SEED_GUIDANCE_DEMO=1 SEED_DECISION_DEMO=1`) → `node scripts/check-docs-seed.mjs` / `check-exam-seed.mjs bank` / `demo` | Mở |
| D-R-03 | Thấp | F9 chấm | Bài nộp ngắn (40 ký tự, có nội dung thật) → điểm 0, nhận xét «Bài nộp không có nội dung để chấm.» (sai thông điệp; ngưỡng độ dài không nêu cho SV) | SV nộp PDF 1 dòng 40 ký tự → `grade-all` → kết quả `total=0.00` | **Đạt** ở main `6b30be37`: bài 40 ký tự → «Bài nộp quá ngắn để chấm (dưới 50 ký tự); giảng viên chấm tay.», cờ `EMPTY_SUBMISSION`. Nhánh «rỗng» **không kiểm được** (PDF không chữ rơi vào `EXTRACT_FAILED`, đường khác) |
| D-R-04 | Thấp | F19 Thi | Hợp đồng lệch: `openapi.yaml` ghi header `X-Exam-Tab` `required: false` cho `PUT …/attempts/{aid}/answers` và `POST …/submit`, nhưng thiếu header → 422 `VALIDATION_FAILED` (`field: X-Exam-Tab`); riêng `POST …/attempts` (bắt đầu) không cần | SV đang thi: `PUT /courses/{id}/exams/{eid}/attempts/{aid}/answers` không gửi `X-Exam-Tab` → 422; gửi uuid → 200 | Mở — ghi sprint 11 (PM) |
| D-R-05 | TB | F6 Tài liệu (UI) | Bảng Tài liệu ở 1440 vỡ khi có tài liệu **Lỗi**: cột Tên co còn ~1 ký tự/dòng («du·p-a-07 13 0.pd f», hàng cao 141–198 px, audit D3 ×4), cột «Trạng thái» chiếm hết bề rộng bằng câu lỗi dài, cột «Cập nhật» bị cắt. 375 bình thường. Audit 10.7 không bắt vì seed không có tài liệu lỗi | GV, lớp 761987: tải lên PDF 2 dòng ngắn (trích ~60 ký tự → trạng thái Lỗi «Không đọc được chữ trong tệp…») → mở `/documents` ở 1440. Ảnh: `audit-r/teacher-data-_documents-1440.jpg` | **Đạt** ở main `6b30be37`: `/documents` 1440 hàng 48 px, câu lỗi cắt «…», không tràn (`audit-r2/`) |
| D-R-06 | Thấp | F9 Chấm (UI) | `/grading` ở 1440: hàng có dòng phụ «Bài trống» cao 67 px (audit D3: cần 48/44 ±1); cùng nguồn gốc D-R-03 (bài ngắn bị coi trống) | GV: nộp bài 40 ký tự → `grade-all` → `/grading` tab Hàng chờ. Ảnh `audit-r/teacher-data-_grading-1440.jpg` | **Đạt** ở main `6b30be37`: `/grading` hàng 48 px, nhãn «Thiếu nội dung» (`audit-r2/`) |
| D-R-07 | TB | F3 Chat riêng | Câu ngoài phạm vi tài liệu («Cách nấu phở bò…», «đề tham khảo tuần 5» chưa có nội dung) khi **có truy xuất nhưng không liên quan**: LLM từ chối thành thật («ngữ cảnh chưa đủ thông tin…») nhưng `done.low_confidence=false`, `ticket=null`, `citations=[]` → không có đề nghị nhờ giảng viên (đường «không ngữ cảnh» thì có `low_confidence=true` + ticket). Cùng một tình huống người dùng, hai hành vi khác nhau | Embedding thật: SV `POST /chat/sessions/{sid}/messages {"content":"Cách nấu phở bò Hà Nội ngon như thế nào?"}` → SSE `done` | Lần 2 (`140a8b15`): OOD ĐẠT, IN ticket giả 7/12. **Lần 3 (`35591702`, P-11-21): OOD ĐẠT 12/12; IN ticket giả 3/12 = 25 % > 20 % — CHƯA ĐẠT (sát ngưỡng)**; xem «Đo P-11-18 lần 3» |
| D-R-08 | Thấp | F6 Tài liệu | Lập chỉ mục lại cả lớp (`POST /courses/{id}/documents/reindex`) → job `FAILED` với khoá Gemini bậc miễn phí: lô nhúng 100 chuỗi trúng hạn mức 100 yêu cầu/phút → 429 ×3 lần thử rồi bỏ (audit `rate_limited`/`RATE_LIMIT`). Từng tài liệu (`/documents/{id}/reindex`, cách nhau ≥ 60 s) thì xong. Không có lùi thời gian dài hơn / chia lô nhỏ khi 429 | Gemini `gemini-embedding-001` bậc miễn phí; `POST …/documents/reindex` lớp 761987 (≈ 100 chunk) | Mở (ghi nhận) |

## Lượt embedding THẬT (PM duyệt P-11, trần 2.000 lời gọi)
- Cách nạp: khoá đọc từ `~/Documents/TA_Agent_v2-s95/.env.local`, đẩy qua `POST /admin/llm/providers` (type `gemini`, model `gemini-embedding-001`, 1536 chiều), lưu mã hoá trong DB ep-qc; **không** đưa vào env container, không in, không ghi log/commit. Route EMBEDDING → provider đó.
- Phạm vi: 6 tài liệu READY dùng cho AI của lớp 761987 (121 chunk, `reindex` từng tài liệu) + 1 tài liệu QC tải mới (đã xoá) + 8 câu hỏi chat + 1 bài Threads. Decision vẫn giả, chat vẫn 9router.
- **Số lời gọi thực** (`llm_audit`, provider `gemini-embed-qcR`): 29 dòng, 43 lần gọi HTTP (gồm 14 lần bị 429 ở lượt `reindex` cả lớp), ≈ 2 % trần. Cộng 1–2 lời gọi kiểm khoá khi tạo provider (không vào audit) ⇒ < 50.
- Gỡ: route EMBEDDING → provider `fake` (DB; API không cho để trống chuỗi — 422 `ROUTE_INVALID chain_empty`), provider Gemini đã `DELETE` (204, khoá bị xoá), reindex lại 6 tài liệu bằng embedding giả.

| D-R-09 | TB | F3 Chat riêng | Câu trả lời mở bằng câu mẫu từ chối nhưng vẫn `low_confidence=false`, không ticket, không trích dẫn: «Ví dụ về tấn công truy cập (access attacks) trong slide là gì?» → «Mình chưa tìm thấy nội dung này trong tài liệu của lớp. Trong tài liệu hiện chỉ liệt kê "Access Attacks" là một trong bốn loại tấn công… chưa cung cấp ví dụ chi tiết» (`confidence=1,000`). P-11-18(b) quy định mở bằng câu mẫu ⇒ xử lý như `no_context` (low=true, escalate) | Embedding thật, main `35591702`: SV `sv20` hỏi câu trên → SSE `done` (`message_id` ASSISTANT lớp 761987, 15/11 ~11:3x) | Mở |

## Đo P-11-18 (main `140a8b15`, 9router thật, embedding Gemini thật — lượt 2)
Script `scripts/w10_p1118.py`; 12 câu ngoài tài liệu (cùng chủ đề an ninh mạng: WannaCry, Zero-trust, WPA3, SQLi mù, Kerberos golden ticket, MITRE ATT&CK, stored XSS, SIEM/SOAR, honeypot, IPsec, Spectre, ISO 27001) và 12 câu trong tài liệu (bài giảng Network Security Threats), 4 SV riêng mỗi chiều (mỗi SV ≤ 5 ticket).

| Chiều | Chỉ số | Kết quả | Ngưỡng PM | |
| --- | --- | --- | --- | --- |
| Ngoài tài liệu | có ticket | **12/12 (100 %)** | ≥ 90 % | ĐẠT |
| | LLM mở bằng câu mẫu «Mình chưa tìm thấy nội dung này trong tài liệu của lớp.» | 12/12 | – | |
| | `low_confidence=true` / có trích dẫn | 12/12 / 0/12; `confidence` 0,43–0,52 | – | |
| Trong tài liệu | ticket giả | **7/12 (58 %)**; 7/10 nếu bỏ 2 câu bị 9router rớt (1 «Trả lời tạm thời», 1 «AI đang gián đoạn») | ≤ 20 % | **CHƯA ĐẠT** |
| | có `[n]` (regex `\[\d+\]`) | 10/12 (1 câu dạng `[2, 5, 7]`, 1 câu 9router rớt) | – | |
| | câu mẫu (từ chối nhầm) | 0/12 | – | |
- `confidence` các câu trong tài liệu bị cờ thấp: 0,600 ×4 («bốn nhóm tấn công», DoS, worm/virus, smurf), 0,657 (Code Red), 0,680 (Trojan), 0,733 (Reconnaissance); các câu không cờ: 0,800–1,000. Ngưỡng thực tế quanh 0,75. Cả 7 câu đều trả lời đúng nội dung tài liệu, có `[n]` (trừ câu `[2, 5, 7]`).
- Gợi ý cho dev (P-11-19): nguồn tài liệu tiếng Anh + câu hỏi tiếng Việt cho cos nền ~0,5–0,6 → `RAG_SIM_FLOOR/CEIL` 0,25/0,65 chưa hiệu chỉnh; khớp dự đoán P-11-19.
- Lần chạy đầu (trước khi 9router ổn định) bị loại: 12/12 câu OOD đều «Trả lời tạm thời, trích nguyên văn» do `circuit_open` (9router rớt, watchdog khởi động lại 10:38–10:43); `ticket=0/12`. Số liệu trên là lần chạy lại.
- Lưu ý vận hành: 9router flap làm cầu dao LLM mở và câu trả lời giảm cấp «Trả lời tạm thời…»; `POST /admin/llm/providers/{id}/test` đóng lại sau khi 9router ổn định.

### Lượt embedding thật thứ 2 (PM duyệt, trần 2.000)
- Nạp / gỡ như lượt 1 (khoá → DB qua API admin, `DELETE` provider, route → provider `fake`, reindex lại bằng embedding giả).
- Số lời gọi thực (`llm_audit`, `gemini-embed-qcR`): lượt 2 = 34 dòng / 40 lần HTTP; **cộng cả hai lượt = 63 dòng / 83 lần HTTP (≈ 4 % trần)**; 9 dòng `rate_limited` (reindex tài liệu 82 chunk gặp 429 một lần, chạy lại xong).

## Đo P-11-18 lần 3 (main `35591702`, P-11-21; embedding Gemini thật lượt 3)
Cùng bộ 12 + 12 câu, SV mới (sv12–15 ngoài tài liệu, sv17–20 trong tài liệu).
| Chiều | Chỉ số | Lần 2 (`140a8b15`) | Lần 3 (`35591702`) | Ngưỡng |
| --- | --- | --- | --- | --- |
| Ngoài tài liệu | có ticket / theo câu mẫu | 12/12 · 12/12 | **12/12 · 12/12** (`confidence` ≤ ~0,52, 0 trích dẫn) | ≥ 90 % ĐẠT |
| Trong tài liệu | ticket giả | 7/12 (58 %) | **3/12 (25 %)** — Code Red 0,772, DoS 0,700, Trojan 0,760 (cờ thấp; ngưỡng quanh 0,78) | ≤ 20 % **CHƯA ĐẠT** |
| | có `[n]` | 10/12 | 12/12 | – |
| | từ chối nhầm (mở bằng câu mẫu) | 0/12 | 1/12 (D-R-09: vừa từ chối vừa không ticket) | – |
| Chữ thô `[1, 2]` | luồng token (SSE, đang gõ) | 1 | 2/24 (DoS, worm) | – |
| | nội dung cuối (DB / `done.content`, bản SV thấy sau khi xong) | – | **0/24** — đã tách `[1] [2]` (`ExtractCitations`); giao diện thay bằng `done.content` (`useChatRun.ts`) | không còn ĐẠT |
- Phần «chữ thô»: SV chỉ thấy `[1, 2]` thoáng qua khi chữ đang chảy, rồi được thay bằng `[1] [2]` lúc xong (không kiểm bằng mắt trên UI — suy ra từ mã + DB) [INFERENCE].
- 9router ổn định suốt lượt (không có `circuit_open`; 0 câu giảm cấp).
- Lượt embedding 3: gemini 34 dòng / 40 HTTP; **cộng 3 lượt = 97 dòng / 123 HTTP (≈ 6 % trần 2.000)**, 12 dòng `rate_limited` (reindex tài liệu 82 chunk 429 một lần mỗi lượt, chạy lại xong). Đã `DELETE` provider (204), route → `fake`, reindex lại bằng embedding giả.

