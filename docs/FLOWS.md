# EduPilot v2 — Danh mục luồng end-to-end

Mỗi luồng ở đây là một hành trình trọn vẹn của người thật, đi qua nhiều module. Đây là nguồn cho test E2E (Playwright) và là danh sách kiểm khi nghiệm thu: **một luồng chỉ "xong" khi đi được từ bước đầu đến bước cuối, kể cả các nhánh lỗi.**

## 0. Kết quả lần rà soát ngày 2026-09-20

Rà từng luồng theo câu hỏi "một trường dùng thật thì chỗ nào gãy?" tìm ra 17 lỗ hổng. Tất cả đã được vá vào PRD, ARCHITECTURE và phase file tương ứng.

| # | Lỗ hổng | Vì sao nghiêm trọng | Vá ở |
| --- | --- | --- | --- |
| 1 | Sinh viên tự đăng ký bằng MSSV của người khác rồi được nối vào danh sách lớp | **Xem được điểm và chuyên cần của người khác.** Lỗi bảo mật nặng nhất của bản trước | F1, P2 |
| 2 | Không có xác minh email, quên mật khẩu, khoá khi dò mật khẩu, mời giảng viên, thu hồi phiên | Không vận hành nổi với người dùng thật; JWT 24 giờ không thu hồi được | F1, P2 |
| 3 | Sinh viên không nộp bài trong app được; không có trang bài tập cho sinh viên; giảng viên không có màn tạo bài tập | Luồng chấm bài không có điểm bắt đầu tự nhiên | F9, P7 |
| 4 | Không có phúc khảo; chốt điểm rồi không sửa được | AI chấm mà không có đường khiếu nại thì không trường nào chấp nhận | F10, F11, P6, P7 |
| 5 | Trang Observability cho GIẢNG VIÊN xem prompt (đã che) của chat riêng | Mâu thuẫn với cam kết "chỉ khi escalate giảng viên mới đọc được hội thoại" | F16, P10 |
| 6 | Đang làm bài QUIZ tính điểm vẫn hỏi chat AI được | Gian lận ngay trong chính hệ thống | F12, P9 |
| 7 | Không có trần chi phí LLM | Một đợt chấm hoặc một vòng lặp lỗi có thể đốt hết ngân sách | F15, P1 |
| 8 | Không có công cụ tạo lịch buổi học | Điểm danh cần buổi học; không ai nhập tay 15 buổi × nhiều lớp | F7, P5 |
| 9 | Điểm giữa kỳ / cuối kỳ chỉ nhập từng ô | Thực tế giảng viên có file Excel | F10, P6 |
| 10 | Hết học kỳ: không lưu trữ, không nhân bản sang kỳ sau, không chính sách giữ dữ liệu | Hệ thống dùng được một kỳ rồi thành bãi rác | F18, PR |
| 11 | Chưa xử lý pháp lý dữ liệu cá nhân | Luật BVDLCN 91/2025/QH15 có hiệu lực từ 01/01/2026; gọi LLM nước ngoài là chuyển dữ liệu xuyên biên giới | `PRODUCTION_READINESS.md`, PR |
| 12 | Threads không có báo cáo vi phạm, ẩn bài, xoá bài của mình; không xử lý tin nhắn cho thấy sinh viên gặp khủng hoảng | Diễn đàn công khai của trường bắt buộc phải kiểm duyệt được | F4, F3, P4 |
| 13 | Không giám sát, cảnh báo, sao lưu tự động, runbook, môi trường staging, quét file | Không ai trực hệ thống thì phải có máy trực | PR |
| 14 | Ticket không ai nhận thì nằm im | Sinh viên chờ vô hạn, đúng vấn đề gốc của đề tài | F5, P4 |
| 15 | Người đăng thread không được báo khi có trả lời | Hỏi xong phải tự quay lại kiểm tra | F4, P4 |
| 16 | Chỉ đăng nhập bằng mật khẩu | Trường thường yêu cầu đăng nhập bằng tài khoản trường | F1, PR (tuỳ chọn) |
| 17 | Không có kế hoạch thí điểm, hướng dẫn người dùng, tuyên bố giới hạn của AI | Sản phẩm tốt vẫn thất bại khi triển khai thiếu chuẩn bị | `PRODUCTION_READINESS.md` |

## 1. Bản đồ luồng

| Mã | Luồng | Người | Module | Phase hoàn tất |
| --- | --- | --- | --- | --- |
| F1 | Tài khoản: đăng ký, xác minh, đăng nhập, quên mật khẩu, mời giảng viên | Tất cả | M0 | P2 |
| F2 | Mở lớp → phân công → mã tham gia → vào lớp | Admin, GV, SV | M0, M14 | P2 |
| F3 | Hỏi riêng tư (điểm, quy chế, lịch thi) | SV | M1 | P3 → P6 |
| F4 | Hỏi bài công khai → AI trả lời → xác nhận → thành tri thức | SV, GV | M2, M1 | P4 |
| F5 | AI không chắc → escalate → giảng viên trả lời → mail | SV, GV | M3 | P4 |
| F6 | Tài liệu: tải lên → xử lý nền → RAG + thư viện | GV, SV | M9, M10 | P8 |
| F7 | Buổi học → điểm danh → phát biểu → sinh viên tra cứu | GV, SV | M5, M1 | P5 |
| F8 | Theo dõi sinh viên → ghi chú → can thiệp | GV | M6, M13-C | P5 |
| F9 | Bài tập: tạo → nộp → chấm nháp → duyệt → công bố | GV, SV | M8 | P7 |
| F10 | Quy chế môn học → công thức → sổ điểm → chốt → xuất | GV | M7 | P6 |
| F11 | Phúc khảo | SV, GV | M8, M7, M3 | P7 |
| F12 | Ngân hàng câu hỏi → luyện đề / bài QUIZ | GV, SV | M4, M8 | P9 |
| F13 | Lịch và nhắc việc | Tất cả | M11 | P8 |
| F14 | Hôm nay | SV, GV | M14 | P2 → P10 |
| F15 | Cấu hình LLM, ngân sách, tích hợp | Admin | M12, M8 | P1, P7 |
| F16 | Quan sát hệ thống AI | Admin | M13-A | P10 |
| F17 | Báo cáo lỗ hổng kiến thức → cập nhật giáo trình | GV | M13-B | P10 |
| F18 | Kết thúc học kỳ: lưu trữ, xuất, nhân bản, xoá theo chính sách | Admin, GV | M0 | PR |

## 2. Chi tiết từng luồng

### F1. Tài khoản
- **Đường chính (sinh viên):** đăng ký bằng email + mật khẩu + MSSV tự khai → nhận mail xác minh (link một lần, hạn 24 h) → xác minh → đăng nhập → `/` hiện ô nhập mã lớp.
- **Đường chính (giảng viên / TA):** Admin tạo tài khoản → hệ thống gửi **link mời** một lần (hạn 72 h) → giảng viên tự đặt mật khẩu. Admin không bao giờ biết mật khẩu của ai.
- **Quên mật khẩu:** nhập email → luôn trả cùng một thông báo (không lộ email có tồn tại không) → link đặt lại một lần, hạn 30 phút → mọi phiên cũ bị thu hồi.
- **Phiên:** access token 15 phút + refresh token xoay vòng trong cookie `httpOnly; Secure; SameSite=Lax`; bảng `auth_sessions` cho phép thu hồi; đổi mật khẩu / khoá tài khoản thu hồi mọi phiên. Trang `/settings` liệt kê thiết bị đang đăng nhập.
- **Chống dò:** 5 lần sai → chờ tăng dần; 10 lần → khoá 15 phút + mail cảnh báo; rate limit theo IP.
- **QUY TẮC AN TOÀN SỐ 1 — MSSV tự khai không bao giờ mở được dữ liệu.** Việc nối một tài khoản vào một dòng trong danh sách lớp import chỉ xảy ra khi **email đã xác minh trùng với email trong danh sách**. MSSV trùng mà email khác → vào trạng thái `PENDING`, giảng viên duyệt tay và thấy rõ cảnh báo lệch.
- **Tuỳ chọn (PR):** đăng nhập bằng tài khoản Microsoft của trường qua OIDC (chỉ xin `openid profile email`); có dùng được hay không tuỳ chính sách tenant của trường.
- **Nhánh lỗi:** link hết hạn / đã dùng → trang giải thích + gửi lại; mail không tới → gửi lại sau 60 s; email chưa xác minh → đăng nhập được nhưng không vào lớp được.
- **E2E:** `account.spec.ts` — đăng ký → xác minh qua MailHog → đăng nhập; quên mật khẩu thu hồi phiên cũ; mạo danh MSSV bị chặn.

### F2. Mở lớp → vào lớp
Đường chính như sơ đồ ở PRD M0. Bổ sung sau rà soát:
- Đổi giảng viên giữa kỳ: giảng viên mới nhận thông báo + toàn bộ ticket đang mở; giảng viên cũ mất quyền ngay; mọi thứ ghi `audit_log`.
- Giảng viên tự thêm / bớt TA cho lớp mình (Admin cũng làm được).
- Sinh viên bị mời ra khỏi lớp: mất quyền truy cập ngay, dữ liệu học tập giữ nguyên cho sổ điểm; vào lại bằng mã thì nối lại đúng bản ghi cũ.
- Lớp đã lưu trữ: mã tham gia tự tắt.
- **E2E:** `class-join.spec.ts`, và test tích hợp `TestCourseIsolation`.

### F3. Hỏi riêng tư
1. Sinh viên gõ câu hỏi trong `/chat` (lớp đang chọn) → 2. gateway gắn `trusted_context` từ phiên → 3. che tên / MSSV → 4. phân loại ý định + guardrails → 5. agent gọi tool cá nhân hoặc RAG → 6. tính độ tin cậy → 7. stream câu trả lời đã khôi phục placeholder → 8. sinh viên bấm Hữu ích / Không hữu ích / `Nhờ giảng viên hỗ trợ`.
- **Nhánh:** dưới ngưỡng → F5. Hệ thống quá tải → thông báo thời gian chờ + thử lại. Mọi LLM chết → trả lời trích xuất + nhãn. Công thức điểm chưa xác nhận mà hỏi cách tính điểm → nói rõ chưa có thông tin chính thức + F5. Hỏi về người khác → từ chối + ghi log. Mất mạng giữa chừng → tải lại vẫn thấy phần đã sinh.
- **Liêm chính học thuật:** câu hỏi trùng nội dung bài tập đang mở → chỉ gợi ý kiểu Socratic, không đưa lời giải. Đang trong phiên làm bài QUIZ tính điểm → chat từ chối câu hỏi nội dung, chỉ trả lời câu hỏi thủ tục (xem F12).
- **An toàn con người:** tin nhắn cho thấy sinh viên đang khủng hoảng hoặc có ý tự hại → AI không tư vấn chuyên môn, trả lời ngắn gọn ân cần, đưa thông tin hỗ trợ do trường cấu hình (phòng công tác sinh viên / đường dây hỗ trợ), và gợi ý nói chuyện với giảng viên. Không tự động báo cho ai nếu sinh viên không đồng ý, trừ khi trường có chính sách khác được ghi rõ trong thông báo quyền riêng tư.
- **Quyền của sinh viên với hội thoại:** xoá phiên chat của mình (xoá mềm, xoá hẳn theo chính sách giữ dữ liệu); xuất dữ liệu cá nhân (PR).
- **E2E:** `private-chat.spec.ts`.

### F4. Hỏi bài công khai
1. Soạn bài ở `/threads` → kiểm PII khi đang gõ → 2. bấm đăng → tường lửa; có nội dung cá nhân → hộp thoại hai lối → 3. bài được lưu → 4. AI trả lời kiểu Socratic kèm nguồn, ghi `Chờ xác nhận` → 5. giảng viên / TA nhận thông báo (gộp) → `Xác nhận` / `Chỉnh sửa` / `Loại` → 6. tri thức đã sửa nạp lại vector DB → 7. **người đăng được báo** khi có câu trả lời và khi câu trả lời được xác nhận.
- **Kiểm duyệt:** ai cũng `Báo cáo` được một bài (lý do chọn sẵn) → việc cho giảng viên; giảng viên / TA `Ẩn bài` (người đăng thấy lý do); sinh viên sửa / xoá bài của mình (sửa thì chạy lại tường lửa); 3 báo cáo → tự ẩn chờ duyệt.
- **Nhánh:** AI không đủ tin cậy → không tự trả lời, chỉ báo giảng viên. Bài bị `Loại` → biến mất với sinh viên và khỏi RAG.
- **E2E:** `threads.spec.ts`.

### F5. Escalation
1. `confidence` < ngưỡng lớp, hoặc sinh viên bấm nhờ hỗ trợ → 2. ticket OPEN (idempotent) + outbox → 3. chuông tới giảng viên và TA của lớp → 4. một người `Nhận` → 5. trả lời trong app → 6. câu trả lời vào chat riêng + chuông + **mail** cho sinh viên → 7. sinh viên hỏi tiếp (mở lại) hoặc xác nhận đã rõ → đóng; 7 ngày không phản hồi → tự đóng → 8. tuỳ chọn lưu thành tri thức (đã che thông tin cá nhân).
- **Không ai nhận:** sau 24 giờ làm việc → nhắc lại cho cả nhóm; sau 72 giờ → việc nổi lên đầu "Hôm nay" kèm số giờ đã chờ, và báo Admin nếu lớp không có ai hoạt động. Sinh viên luôn thấy trạng thái ticket ("Đang chờ giảng viên · đã gửi 3 giờ trước").
- **Nhánh:** hai người cùng nhận → người sau nhận 409 và thấy ai đã nhận; mail gửi lỗi → thử lại 3 lần, vẫn lỗi thì hiện trong `/observability`, câu trả lời trong app không bị ảnh hưởng.
- **E2E:** `escalation.spec.ts`.

### F6. Tài liệu
1. Giảng viên kéo thả file ở `/documents`, chọn loại + cờ → 2. client xin URL ký sẵn, đẩy thẳng lên object storage → 3. `ingest.jobs`: kiểm loại và kích thước, **quét mã độc (PR)**, trích văn bản, làm sạch, chia đoạn, nhúng → 4. tiến độ qua SSE; READY / FAILED kèm lý do đọc hiểu được → 5. xuất hiện trong RAG và / hoặc `/library` theo cờ.
- **Nhánh:** trùng nội dung → báo đã có, đề nghị chia sẻ sang lớp này; file scan không có lớp chữ → OCR, không đọc được thì FAILED rõ lý do; xoá tài liệu → xoá chunk + vô hiệu cache + câu trả lời cũ vẫn giữ citation nhưng đánh dấu "nguồn đã gỡ"; `COURSE_POLICY` → kích hoạt F10.
- **E2E:** `documents.spec.ts` + test `answer_key_never_retrieved`.

### F7. Điểm danh và phát biểu
1. **Tạo lịch buổi học** một lần: chọn thứ / giờ / phòng / ngày bắt đầu–kết thúc, loại trừ ngày nghỉ → sinh hàng loạt `class_sessions` (sửa từng buổi được: nghỉ, học bù) → 2. đến giờ, "Hôm nay" của giảng viên hiện việc điểm danh buổi đang diễn ra → 3. `/attendance`: chỉ chạm người vắng, ghi phát biểu ngay trên dòng → 4. `Lưu điểm danh` → 5. sinh viên xem ở `/me` hoặc hỏi AI.
- **Nhánh:** mất mạng → xếp hàng cục bộ; hai người cùng điểm danh → khoá lạc quan; sửa sau buổi học → được, ghi audit; sinh viên thấy sai → hỏi trong chat → F5 (ticket kèm ngữ cảnh buổi học).
- **E2E:** `attendance.spec.ts` (có bước ngắt mạng).

### F8. Theo dõi sinh viên
`/students` lọc "Cần chú ý" → mở hồ sơ → đọc câu nhận định rủi ro + lý do → thêm ghi chú riêng → tuỳ chọn tạo nhận xét AI → hành động: nhắn riêng cho sinh viên qua ticket do giảng viên mở (dùng lại F5 theo chiều ngược), hoặc tạo buổi ôn tập (F13).
- **Ràng buộc:** sinh viên không bao giờ thấy ghi chú hay nhãn rủi ro của mình; mọi lần xem hồ sơ ghi `audit_log`.

### F9. Bài tập: tạo → nộp → chấm → công bố
1. Giảng viên tạo bài ở **Chấm bài → tab Bài tập**: tiêu đề, mô tả, hạn, loại (ESSAY / QUIZ), cho nộp muộn + quy tắc trừ, thành phần điểm liên kết, đáp án chuẩn, rubric (AI gợi ý), kênh nhận bài cho phép → 2. công bố bài → sự kiện lịch + việc trên "Hôm nay" của sinh viên → 3. **sinh viên nộp trong app** ở `/assignments/[id]` (kéo thả file, thấy xác nhận + dấu thời gian + nộp lại được trước hạn); hoặc bài về qua Mail / ZIP / Forms / Teams → 4. khớp sinh viên → 5. `grading.jobs` làn BATCH → kết quả nháp → 6. giảng viên duyệt cạnh nhau → `Duyệt bài` → 7. `Công bố` (TEACHER) → sổ điểm + mail nhận xét + sinh viên thấy điểm, nhận xét theo tiêu chí, bằng chứng trích từ bài mình ở `/assignments/[id]`.
- **Nhánh:** nộp sau hạn → LATE + trừ theo quy tắc, hoặc bị chặn nếu bài không cho nộp muộn; file sai loại / quá cỡ → báo ngay trước khi tải; không khớp sinh viên → hàng Chưa khớp; AI gắn cờ → lên đầu hàng chờ; worker chết giữa chừng → việc được nhận lại, không chấm đôi; sinh viên KHÔNG BAO GIỜ thấy điểm nháp.
- **E2E:** `assignment-lifecycle.spec.ts`.

### F10. Sổ điểm
1. Tải quy chế môn học → 2. AI trích bản nháp + danh sách "chưa rõ" → 3. giảng viên điền, `Xác nhận công thức` → 4. điểm đổ về từ: bài tập đã công bố, QUIZ, **nhập file XLSX** (giữa kỳ, cuối kỳ: ánh xạ cột, xem trước, báo dòng lỗi), sửa ô trực tiếp → 5. xem trước + giải trình → 6. `Chốt điểm` → snapshot → 7. xuất XLSX theo mẫu cột cấu hình được.
- **Sửa sau khi chốt:** `Mở khoá có lý do` (TEACHER, bắt buộc ghi lý do) → sửa → chốt lại → snapshot phiên bản mới; phiên bản cũ giữ nguyên; sinh viên bị ảnh hưởng được báo.
- **Nhánh:** chưa có quy chế / chưa xác nhận → banner, khoá chốt, AI từ chối suy diễn; quy chế mới → bản nháp mới + so sánh; hai người sửa cùng ô → 409.
- **Ghi rõ trên giao diện và file xuất:** điểm chính thức là điểm trong hệ thống quản lý đào tạo của trường; EduPilot là công cụ hỗ trợ tính và giải trình.
- **E2E:** `gradebook.spec.ts`.

### F11. Phúc khảo
1. Ở `/assignments/[id]` (điểm đã công bố) sinh viên bấm `Yêu cầu xem lại`, chọn tiêu chí và ghi lý do (trong hạn phúc khảo của bài, mặc định 7 ngày) → 2. ticket loại `GRADE_APPEAL` kèm bài làm, rubric, kết quả AI, điểm đã duyệt → 3. giảng viên xem lại trong màn duyệt → giữ nguyên hoặc sửa, bắt buộc ghi phản hồi → 4. sinh viên nhận chuông + mail → 5. nếu sửa: sổ điểm cập nhật, audit, và nếu lớp đã chốt điểm thì đi qua "mở khoá có lý do".
- **Ràng buộc:** mỗi bài một lần phúc khảo; AI không tham gia quyết định phúc khảo; số liệu phúc khảo đổ vào M13-A như một chỉ số chất lượng chấm.
- **E2E:** `grade-appeal.spec.ts`.

### F12. Luyện đề và bài QUIZ
Trích / sinh câu hỏi (nền) → duyệt → sinh viên luyện theo chủ đề hoặc thi thử → phân tích điểm yếu → "Hôm nay" gợi ý ôn. Bài QUIZ tính điểm dùng cùng màn làm bài.
- **Chống gian lận mức cơ bản (không phải giám thị):** xáo câu và đáp án theo từng sinh viên; một lần làm; cửa sổ thời gian; đáp án và giải thích chỉ hiện sau khi bài đóng với cả lớp; **trong lúc có phiên QUIZ tính điểm đang mở, chat riêng của sinh viên đó từ chối câu hỏi nội dung** và tool `search_library` không trả tài liệu loại đề / đáp án; ghi nhận rời tab như một tín hiệu cho giảng viên xem, không tự trừ điểm.
- **E2E:** `practice.spec.ts`, `quiz-integrity.spec.ts`.

### F13. Lịch và nhắc việc
Sự kiện sinh từ bài tập, buổi học, và sự kiện giảng viên tạo (thi, ôn tập) → tháng / tuần / danh sách → nhắc 24 giờ (chuông + mail, không gửi trùng, tắt được từng loại) → ICS. Đổi hạn → lịch, "Hôm nay", câu trả lời AI đổi ngay.

### F14. Hôm nay
Mỗi module đăng ký nguồn việc; bảng đầy đủ nguồn việc sau rà soát:

| Vai trò | Nguồn việc |
| --- | --- |
| Sinh viên | Chưa có lớp → nhập mã · email chưa xác minh · hạn nộp < 24 h chưa nộp · QUIZ / thi trong 48 h · điểm vừa công bố · phúc khảo có phản hồi · chủ đề sai nhiều · câu trả lời của giảng viên chưa đọc · tài liệu mới |
| Giảng viên / TA | Buổi học đang diễn ra chưa điểm danh · ticket chờ (kèm số giờ) · phúc khảo chờ · bài chấm "cần xem kỹ" · bài đã duyệt chưa công bố · câu trả lời AI chờ xác nhận · bài bị báo cáo · bài nộp chưa khớp · yêu cầu vào lớp chờ duyệt · email lệch MSSV chờ duyệt · công thức điểm chưa xác nhận · thiết lập lớp mới · câu hỏi chờ duyệt · sinh viên mới vào diện cần chú ý |
| Admin | Provider LLM lỗi · sắp chạm trần ngân sách · hàng dead-letter có việc · sao lưu gần nhất thất bại · lớp không có giảng viên hoạt động |

### F15. Cấu hình LLM, ngân sách, tích hợp
Thêm provider → test → gán model theo tác vụ → fallback. **Ngân sách:** trần chi phí theo ngày và theo tháng, cấu hình toàn hệ thống và theo lớp; 80% → cảnh báo Admin; 100% → làn BATCH dừng nhận việc mới (việc xếp hàng, không mất), làn INTERACTIVE chuyển sang model rẻ nhất đã cấu hình; không bao giờ tắt hẳn chat mà không báo. **LLM nội địa / tự host:** loại provider "OpenAI-compatible" cho phép trỏ vào máy chủ trong nước hoặc trong trường — đây là lối thoát khi trường không chấp nhận gửi dữ liệu ra nước ngoài.

### F16. Quan sát hệ thống AI
Dải trạng thái → bảng yêu cầu → Drawer chi tiết → trace Jaeger. **Phân quyền sau rà soát:** nội dung prompt (dù đã che) chỉ ADMIN xem được, mỗi lần mở ghi `audit_log` kèm lý do; TEACHER chỉ thấy số liệu tổng hợp của lớp mình. Thêm trang `/admin/audit` xem nhật ký kiểm toán và `/admin/health` (hàng đợi, dead-letter, sao lưu, dung lượng).

### F17. Báo cáo lỗ hổng kiến thức
Như PRD M13-B. Bổ sung: báo cáo chỉ dùng câu hỏi của lớp đang chọn; thông báo quyền riêng tư nói rõ câu hỏi (đã ẩn danh) được dùng để cải thiện giảng dạy; chủ đề có dưới 3 sinh viên hỏi không hiện câu mẫu (tránh suy ngược ra cá nhân).

### F18. Kết thúc học kỳ
1. Giảng viên chốt điểm → 2. Admin hoặc giảng viên `Lưu trữ lớp`: lớp thành chỉ-đọc, mã tham gia tắt, việc nền dừng, sinh viên vẫn xem được điểm và tài liệu đến hết hạn giữ dữ liệu → 3. `Xuất toàn bộ dữ liệu lớp` (ZIP: sổ điểm, điểm danh, bài nộp, nhận xét) cho giảng viên lưu hồ sơ → 4. `Nhân bản sang học kỳ mới`: tài liệu, ngân hàng câu hỏi, rubric, cấu trúc bài tập, bản nháp công thức; KHÔNG mang theo sinh viên, điểm, hội thoại → 5. job giữ dữ liệu: hội thoại và `llm_audit` xoá sau thời hạn cấu hình (mặc định 12 tháng sau khi lưu trữ); heartbeat thô 7 ngày; sổ điểm và audit giữ theo quy định lưu trữ của trường.
