# TC đợt 5 — Chat / Hỏi đáp / Kết quả của tôi / Tài liệu / Insights (US-UX91-13, 19, 20, 18, 17)

Phiên bản 1 · 2026-11-08. Quy ước và kiểm chung: `tc-00-chung.md`. Tên màn theo US-07: `/threads` = «Hỏi đáp công khai», `/insights` = «Lỗ hổng kiến thức», `/analytics` = «Số liệu lớp học».
LLM của stack `ep-qc` là provider giả (không tốn hạn mức). Kiểm riêng tư (PII) ở US-13: đọc `docker compose -p ep-qc logs gateway` và bảng `llm_audit` (qua `psql` trên VPS, chỉ đọc).

---
## US-UX91-13 — Chat riêng

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX13-1 | AC1 | sinh viên, 1440 | `/chat` có ≥ 3 phiên; đo vùng hội thoại | Một Panel hội thoại, rộng ≤ 840 px, căn giữa vùng nội dung (lề trái ≈ lề phải ±2 px); lịch sử là **cột phẳng** bên trái, ngăn bằng đường kẻ 1 px (không bo, không bóng, không nền Panel riêng) | E2E |
| TC-UX13-2 | AC1 | — | `bash scripts/ui-antipatterns.sh`; `panels.spec.ts` | Phép «nhiều Panel trong một vùng làm việc» sạch; `panels.spec` xanh | GREP + E2E |
| TC-UX13-3 | AC1 | sinh viên | Thu gọn cột lịch sử | Thu gọn được; nội dung giãn ra, không giật (CLS ≤ 0,05) | E2E |
| TC-UX13-4 | AC2 | sinh viên | Phiên có tiêu đề dài > 2 dòng; đọc từng hàng lịch sử | Tiêu đề ≤ 2 dòng, cắt `…`; ngày giờ theo US-02 («Hôm qua, 16:40»); **không** biểu tượng thùng rác trên mọi hàng | E2E |
| TC-UX13-5 | AC2 | sinh viên | Rê / Tab vào hàng, mở `⋯` | Xoá nằm trong menu `⋯` của hàng (vùng chạm ≥ 44 px ở 375) | E2E |
| TC-UX13-6 | AC3 | sinh viên | `⋯` → Xoá; quan sát hộp thoại, DOM, request | Không hộp thoại xác nhận; phiên biến ngay; dòng «Đã xoá · Hoàn tác» ≈ 5 s | E2E |
| TC-UX13-7 | AC3 | sinh viên | Bấm `Hoàn tác` trong 5 s | Phiên trở lại đúng vị trí; không có request `DELETE` | E2E |
| TC-UX13-8 | AC3 | sinh viên | Không bấm; đợi > 5 s; tải lại | Có `DELETE` sau ≈ 5 s; phiên mất vĩnh viễn sau tải lại | E2E |
| TC-UX13-9 | AC4 | sinh viên | Phiên trống | **Không** có nút `Phiên mới` (không phải nút khoá) | E2E |
| TC-UX13-10 | AC4 | sinh viên | Gửi 1 tin | Nút `Phiên mới` hiện, là nút phụ (không đỏ, `background` không `--ep-red`) | E2E |
| TC-UX13-11 | AC5 | sinh viên | Phiên trống | Lời chào ngắn + **tối đa 3** gợi ý dạng hàng chữ (không nút bo: `border-radius`≈0, không nền); gợi ý gồm hai câu cố định «Điểm quá trình của tôi hiện là bao nhiêu?» và «Tuần này tôi có những buổi học nào?» (+ ≤ 1 gợi ý dựng từ `GET /me/today`) | E2E |
| TC-UX13-12 | AC5 | sinh viên | Bấm một gợi ý; đếm request `POST` | Câu điền vào ô nhập; **không** tự gửi (0 `POST`) | E2E |
| TC-UX13-13 | AC5 | sinh viên | Đo khoảng trống trên ô nhập | ≤ 1/3 chiều cao khung hội thoại | E2E |
| TC-UX13-14 | AC6 | sinh viên | Quét chữ trên màn (quét 1440 + 375) | Không từ RAG / PII / provider / fallback / trace / confidence / token / LLM / embedding / prompt / model | E2E |
| TC-UX13-15 | AC6 | sinh viên | Gửi «Em tên Nguyễn Minh Trung, MSSV 20229002, email sv.kha@edupilot.local, hỏi điểm giữa kỳ» | Dòng «Đã ẩn N thông tin cá nhân trước khi gửi cho AI» (N ≥ 2) còn nguyên | E2E |
| TC-UX13-16 | AC6 | VPS | `docker compose -p ep-qc logs gateway` + `llm_audit` sau TC-15 | Payload gửi LLM chỉ có placeholder, **không** tên / MSSV / email thật; log không chứa PII | GREP |
| TC-UX13-17 | AC6 | sinh viên | Phiên trống: đọc lời giới thiệu | Còn cam kết «Phiên chat này chỉ bạn đọc được»; không thêm `meta` mới dưới `h1` | E2E |
| TC-UX13-18 | AC7 | sinh viên | Trong lúc AI đang trả lời: gõ tiếp; kéo cuộn lên | Gõ được; nút `Dừng` hiện; tự cuộn dừng khi kéo lên | E2E |
| TC-UX13-19 | AC7 | sinh viên | Gõ nháp, đợi 2,2 s, tải lại | Nháp còn (tự lưu 2 s) | E2E |
| TC-UX13-20 | AC8 | sinh viên | Cắt mạng giữa stream (`setOffline`) rồi bật lại | Tự nối lại, giữ phần đã sinh; dải mất mạng | E2E |
| TC-UX13-21 | AC8 | sinh viên | Chặn `POST` chat → gửi → bỏ chặn → `Gửi lại` | Nội dung ô nhập giữ nguyên; `Gửi lại` dùng **cùng** `Idempotency-Key`; chỉ 1 tin được ghi (idempotent) | E2E |
| TC-UX13-22 | AC9 | giảng viên / TA | `/chat` | Không đọc được chat riêng của SV (chỉ phản hồi ticket đã escalate) | E2E |
| TC-UX13-23 | AC9 | SV B | Dùng token SV B gọi API phiên của SV A (`id` đoán) | 403 / 404; không lộ nội dung | E2E |
| TC-UX13-24 | AC10 | sinh viên, 375 | `/chat` | Ô nhập sát đáy (trên thanh dưới, không bị che khi bàn phím ảo mở — mô phỏng `visualViewport` thu nhỏ); lịch sử là nút `Phiên trước (N)` mở bảng trượt từ đáy; vùng chạm ≥ 44 px; K4 | E2E |
| TC-UX13-25 | AC11 | sinh viên | Gửi; lấy mẫu `transform` / `opacity` tin mới | `opacity` + `translateY(4px)` trong 160 ms; **không** có hoạt cảnh khi token đang stream; nút `Gửi` có `:active` | E2E |
| TC-UX13-26 | AC11 | sinh viên, `reduce` | Như trên | Hiện tức thì | E2E |
| TC-UX13-27 | AC12 | — | `private-chat.spec.ts` diff + chạy; ảnh `chat-1440`, `chat-390` | Sửa theo AC1–AC5, không nới; ảnh mốc đổi | E2E |
| TC-UX13-28 | (K7) | — | Chụp trước (`audit/shots/chat__student__*`) và sau | Đủ ảnh | MAC |

---
## US-UX91-19 — Hỏi đáp công khai: danh sách

Fixture danh sách (mock `GET /courses/‹id›/threads`): T1 nội dung `- - Hướng dẫn nộp bài`, T2 `· Phisher - Individual uses email…`, T3 `* Ví dụ`, T4 `# Tiêu đề lớn`, T5 `> trích dẫn`, T6 `- - - ·` (chỉ ký tự markdown), T7 nội dung dài 300 ký tự có xuống dòng; 10 thread trạng thái «Chờ xác nhận» + T8 «Đã được giảng viên xác nhận» + T9 «Đã sửa bởi giảng viên» + T10 «Đã loại».

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX19-1 | AC1 | giảng viên, 1440 | Đọc đoạn trích T1–T5 | Mỗi đoạn **không** bắt đầu bằng `-`, `·`, `*`, `#`, `>` hay `- - `; ví dụ T1 → «Hướng dẫn nộp bài» | E2E |
| TC-UX19-2 | AC1 | giảng viên | Đọc T7 | Xuống dòng thành khoảng trắng; độ dài ≤ 140 ký tự kèm `…` | E2E |
| TC-UX19-3 | AC2 | giảng viên | Tab «Câu hỏi» ≥ 80 % «Chờ xác nhận» | Nhãn «Chờ xác nhận» **không** hiện trên từng thread đa số; dòng đầu «{N} câu chờ xác nhận» (N đúng); chỉ thread khác trạng thái (T10 «Đã loại») có nhãn | E2E |
| TC-UX19-4 | AC2 | giảng viên / SV | T8, T9 | «Đã được giảng viên xác nhận» và «Đã sửa bởi giảng viên» vẫn hiện trên **từng** thread có nó | E2E |
| TC-UX19-5 | AC2 | giảng viên | Dữ liệu < 80 % cùng trạng thái | Mọi thread hiện nhãn (không gom) | E2E |
| TC-UX19-6 | AC3 | sinh viên | Thread chưa xác nhận có câu trả lời AI | Danh sách **không** hiện nội dung câu trả lời AI chưa xác nhận | E2E |
| TC-UX19-7 | AC4 | sinh viên, 375 | `/threads` | «Bộ lọc» là nút có biểu tượng + số bộ lọc đang bật (đặt 2 bộ lọc → «2»), cao ≥ 44 px, mở bảng trượt từ đáy; không còn nhãn chữ trơ | E2E |
| TC-UX19-8 | AC5 | giảng viên | Thời gian thread | Theo US-02; không `00:10 02-11` | E2E |
| TC-UX19-9 | AC6 | giảng viên | T6 (chỉ ký tự markdown) và đoạn trích rỗng | Hiện tiêu đề, không dòng trống / dòng chỉ có ký tự thừa | E2E |
| TC-UX19-10 | AC6 | giảng viên | Huỷ API | `ApiErrorNotice` + `Thử lại` | E2E |
| TC-UX19-11 | AC7 | SV lớp khác | Mở `/threads/‹id thread lớp 761987›` bằng SV không thuộc lớp | 403 | E2E |
| TC-UX19-12 | AC7 | sinh viên | Gõ bài đăng có MSSV / email | `PIIChannelDialog` giữ nguyên, nói «Hỏi đáp công khai» | E2E |
| TC-UX19-13 | AC8 | mọi vai | `/threads` | `h1` «Hỏi đáp công khai» | E2E |
| TC-UX19-14 | AC9 | — | Chụp `threads` 1440 / 390 trước (`audit/shots/threads__*`) và sau | Đủ ảnh | MAC |
| TC-UX19-PG | (phân trang) | giảng viên | Quan sát request danh sách | Có `limit` ≤ 100 + `cursor`; cuộn tải thêm | E2E |

---
## US-UX91-20 — Kết quả của tôi

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX20-1 | AC1 | sinh viên, 1440 | `/me` có điểm | Đúng **một** khối giải trình; mỗi dòng «Thành phần · trọng số → điểm → đóng góp» (ví dụ «Giữa kỳ · 20 % · 7,5 → 1,5»); «Căn cứ · tr. 1» cùng dòng; khối liệt kê thành phần riêng không còn | E2E |
| TC-UX20-2 | AC2 | sinh viên | Đọc dưới `h1` | Không dòng phụ chỉ có mã lớp trần «761987»; câu «Điểm quá trình hiện tại của bạn là X trên 10.» còn | E2E |
| TC-UX20-3 | AC3 | sinh viên | So từng số trên màn với API (`GET /courses/‹id›/gradebook/students/‹uid›/explain` của chính SV) | Khớp đúng giá trị máy chủ; không làm tròn, không tính lại | E2E |
| TC-UX20-4 | AC4 | sinh viên | Lớp có điểm nháp / ghi chú quan sát / nhãn rủi ro chưa công bố | Không xuất hiện; chỉ điểm đã công bố | E2E |
| TC-UX20-5 | AC5 | sinh viên | Điểm đã công bố → `Yêu cầu xem lại` | Hành vi cũ (mở biểu mẫu / gửi yêu cầu); đúng **một** nút phụ | E2E |
| TC-UX20-6 | AC6 | `sv.nguyco` hoặc SV chưa có điểm công bố | `/me` | `EmptyState` «Chưa có điểm nào được công bố.» + nêu nơi sẽ xuất hiện | E2E |
| TC-UX20-7 | AC6 | sinh viên | Huỷ API | `ApiErrorNotice` + `Thử lại` | E2E |
| TC-UX20-8 | AC7 | SV A | Token SV A gọi `…/gradebook/students/‹uid SV B›/explain` | 403 (danh tính từ JWT) | E2E |
| TC-UX20-9 | AC8 | sinh viên, 375 | Đo `document.documentElement.scrollHeight` lớp mẫu | ≤ 1.600 px (trước 2.645 px); K4; dòng giải trình xuống dòng gọn | E2E |
| TC-UX20-10 | AC9 | — | `me.spec.ts` diff + chạy | Sửa theo AC1–AC2, không nới | E2E |
| TC-UX20-11 | AC10 | — | Chụp `me-390` trước (`audit/shots/me__student__375.png`) và sau | Đủ ảnh | MAC |

---
## US-UX91-18 — Tài liệu giảng viên

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX18-1 | AC1 | giảng viên, 1440 và 1100 | `/documents`; đo từng ô «Loại» (`scrollWidth ≤ clientWidth` của phần chữ) | Mọi nhãn («Quy chế học phần», «Tài liệu khác», «Bài giảng», «Đáp án», «Đề tham khảo»…) hiển thị đủ, không cắt giữa từ | E2E |
| TC-UX18-2 | AC2 | giảng viên | Tài liệu thuộc lớp đang xem | Không có dòng «Chia sẻ từ {lớp}» | E2E |
| TC-UX18-3 | AC2 | giảng viên | Tài liệu chia sẻ **từ lớp khác** (mock hoặc chia sẻ trên `ep-qc`) | Hiện «Chia sẻ từ {lớp}» | E2E |
| TC-UX18-4 | AC3 | giảng viên | Công tắc «Dùng cho AI» / «Hiện cho sinh viên» | Tên truy cập «Dùng cho AI: bật» / «…: tắt»; bật / tắt khác nhau cả màu lẫn nhãn; màu bật / tắt đồng nhất mọi hàng | E2E |
| TC-UX18-5 | AC3 | giảng viên, 375 | Đo công tắc | Vùng chạm ≥ 44 px | E2E |
| TC-UX18-6 | AC4 | sinh viên | Tài liệu «Đáp án» (công tắc «Hiện cho sinh viên» tắt); `/library` khi bài còn mở | Không hiện trong Thư viện của SV | E2E |
| TC-UX18-7 | AC5 | giảng viên | Chặn API; đổi công tắc | Công tắc hoàn lại; dòng lỗi tại hàng + `Thử lại`; bỏ chặn bấm `Thử lại` → áp dụng | E2E |
| TC-UX18-8 | AC6 | STUDENT | `/documents` | 403 / không thấy; vào `/library` được | E2E |
| TC-UX18-9 | AC7 | giảng viên, 375 | `/documents` | Hàng xếp dọc; công tắc ≥ 44 px; K4 | E2E |
| TC-UX18-10 | AC8 | — | Chụp `documents-1440` trước / sau | Đủ ảnh | MAC |
| TC-UX18-PQ | (kiểm chéo) | giảng viên lớp khác | `GET /courses/‹lớp lạ›/documents` | 403 / 404 | E2E |
| TC-UX18-ID | (idempotency) | giảng viên | Bật / tắt công tắc nhiều lần nhanh | Trạng thái cuối khớp lần bấm cuối (không lẫn thứ tự) | E2E |

---
## US-UX91-17 — Lỗ hổng kiến thức và Quan sát AI

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX17-1 | AC1 | giảng viên, 1440 | `/insights` có báo cáo (lớp 761987) | «Chủ đề nên dạy lại», «Tài liệu chưa đề cập», «So với báo cáo ngày …» mỗi khối **một** Panel | E2E |
| TC-UX17-2 | AC1 | giảng viên | Quét phần tử con của Panel: có phần tử vừa có viền vừa bo góc riêng bên trong Panel không | Không (chủ đề ngăn bằng đường kẻ 1 px); `panels.spec` NEST xanh | E2E |
| TC-UX17-3 | AC2 | giảng viên | Chữ trên trang | Áp US-01 dòng 23–28: không còn câu giải thích; hộp riêng tư là `meta` ngắn (≤ 12 từ) | E2E |
| TC-UX17-4 | AC3 | giảng viên | `/observability` | Mỗi chỉ số (yêu cầu, p95, tỷ lệ lỗi, chuyển giảng viên, lần che) xuất hiện **đúng một chỗ**: số «hôm nay» là một dòng chữ (không dải 5 ô cùng cỡ); số 7 ngày là bảng | E2E |
| TC-UX17-5 | AC3 | giảng viên | Bảng 7 ngày | Hàng đầu không trống (có số liệu) | E2E |
| TC-UX17-6 | AC4 | giảng viên | Đo khoảng cách từ đáy Panel trước tới tiêu đề khối kế | ≥ 32 px; lớn hơn khoảng dưới tiêu đề (quy tắc §6) | E2E |
| TC-UX17-7 | AC5 | giảng viên | Mở `/observability`; chọn một hàng yêu cầu | Không đọc được nội dung câu hỏi / prompt (chỉ metadata đã che) | E2E |
| TC-UX17-8 | AC5 | admin | Admin mở nội dung prompt | Xem được; có một bản ghi `audit_log` mới (kiểm bằng `psql` chỉ đọc) | E2E |
| TC-UX17-9 | AC6 | giảng viên | Lớp chưa có báo cáo Insights | `EmptyState` + đúng một nút `Tạo báo cáo mới` | E2E |
| TC-UX17-10 | AC6 | giảng viên | Bấm `Tạo báo cáo mới` (provider giả) | Chỉ báo giai đoạn («Đang nhóm N câu hỏi…»); rời trang được; quay lại báo cáo cũ vẫn xem được | E2E |
| TC-UX17-11 | AC7 | STUDENT, TA | `/insights`, `/analytics`, `/observability` | STUDENT 403; TA theo quyền hiện có (không đọc nội dung chat riêng chưa escalate) | E2E |
| TC-UX17-12 | AC8 | giảng viên, 375 | Hai trang | Khối xếp dọc; K4 | E2E |
| TC-UX17-13 | AC9 | giảng viên | `getAnimations()` khi mở; `:active` của `Tạo báo cáo mới` | Không chuyển động mới; `:active` thu 0,97 | E2E |
| TC-UX17-14 | AC10 | — | Chụp `insights-1440`, `observability-1440` trước (`audit/shots/insights__*`, `observability__*`) và sau | Đủ ảnh | MAC |
| TC-UX17-PQ | (kiểm chéo) | giảng viên khác lớp | `GET /courses/‹lớp lạ›/insights` | 403 / 404 | E2E |

---
## Kiểm chéo đặc biệt cuối sprint (đợt 5)
| TC | Nội dung | Chạy |
| --- | --- | --- |
| TC-UX91-X1 | **AC10a của US-08**: handoff đợt 5 có ba ảnh khung ghim, clip, poster chụp lại từ UI mới (xem TC-UX08-37) | TAY |
| TC-UX91-X2 | Chạy toàn bộ `pnpm -C frontend exec playwright test` + `bash scripts/ui-antipatterns.sh` + `pnpm lint && build`; Lighthouse `/` | E2E + LH |
| TC-UX91-X3 | `git diff 1b72f54... --stat`: không có `backend-go/db/migrations` mới; không `fetch` trần mới (`rg -n "\bfetch\(" frontend/src --glob '!**/shared/api/**'`); không sửa `Demo*.tsx` | GREP |
| TC-UX91-X4 | Quét chữ toàn app của SV ở 375 (mọi route): không từ kỹ thuật AI; nhãn nav mới nhất quán | E2E |

---
## Bảng phủ AC đợt 5
| Story | AC | TC |
| --- | --- | --- |
| US-13 | AC1 | 13-1, 2, 3 |
| | AC2 | 13-4, 5 |
| | AC3 | 13-6, 7, 8 |
| | AC4 | 13-9, 10 |
| | AC5 | 13-11, 12, 13 |
| | AC6 | 13-14…17 |
| | AC7 | 13-18, 19 |
| | AC8 | 13-20, 21 |
| | AC9 | 13-22, 23 |
| | AC10 | 13-24 |
| | AC11 | 13-25, 26 |
| | AC12 | 13-27, 28 |
| US-19 | AC1 | 19-1, 2 |
| | AC2 | 19-3, 4, 5 |
| | AC3 | 19-6 |
| | AC4 | 19-7 |
| | AC5 | 19-8 |
| | AC6 | 19-9, 10 |
| | AC7 | 19-11, 12 |
| | AC8 | 19-13 |
| | AC9 | 19-14 |
| US-20 | AC1 | 20-1 |
| | AC2 | 20-2 |
| | AC3 | 20-3 |
| | AC4 | 20-4 |
| | AC5 | 20-5 |
| | AC6 | 20-6, 7 |
| | AC7 | 20-8 |
| | AC8 | 20-9 |
| | AC9 | 20-10 |
| | AC10 | 20-11 |
| US-18 | AC1 | 18-1 |
| | AC2 | 18-2, 3 |
| | AC3 | 18-4, 5 |
| | AC4 | 18-6 |
| | AC5 | 18-7 |
| | AC6 | 18-8 |
| | AC7 | 18-9 |
| | AC8 | 18-10 |
| US-17 | AC1 | 17-1, 2 |
| | AC2 | 17-3 |
| | AC3 | 17-4, 5 |
| | AC4 | 17-6 |
| | AC5 | 17-7, 8 |
| | AC6 | 17-9, 10 |
| | AC7 | 17-11 |
| | AC8 | 17-12 |
| | AC9 | 17-13 |
| | AC10 | 17-14 |
