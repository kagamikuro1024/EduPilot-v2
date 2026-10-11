# Sprint 6: báo cáo (P3 Hai kênh + PII, P8 Tài liệu + Lịch)

**Mục tiêu:**
- **P3:** câu hỏi cá nhân không lọt ra Threads; tên và MSSV không bao giờ tới LLM; sinh viên hỏi được dữ liệu của chính mình qua tool lấy danh tính từ JWT.
- **P8:** tài liệu được trích và nhúng bằng việc nền (`docling-serve`); có thư viện; có lịch với feed ICS và nhắc trước 24 h.

**Kết quả:** 11/11 story PASS. **Cổng P3 ĐẠT, cổng P8 ĐẠT**, chốt tại `cc44964`, CI xanh. Nhánh `sprint/6-p3-p8`.

| Story | Nội dung | QC |
| --- | --- | --- |
| US-P3-01 | Migration `00007_chat_threads`, `00008_privacy` (FK phức hợp theo lớp, `pii_events` chỉ được thêm), sqlc | PASS |
| US-P3-02 | `internal/privacy`: phát hiện / ẩn (regex + từ điển roster theo lớp: có dấu, không dấu, đảo thứ tự), che / khôi phục, `unmask_stream` chịu được token cắt ở mọi vị trí | PASS |
| US-P8-01 | Ingest nền (`docling-serve`, consumer riêng, gia hạn thuê, thử lại), chunk 800 ký tự, nhúng, tìm lai RRF (vector + từ khoá `tsv`), lọc `audience` ngay trong SQL | PASS, sau sửa |
| US-P3-03 | Che danh tính ở **một chỗ** trong `internal/llm`; route thử `_test/llm/payloads`; `TestNoPayloadLeak` | PASS, sau sửa |
| US-P3-04 | Phân loại kênh (luật + embedding, một lần mỗi tin nhắn); `internal/agent` định tuyến tất định; tool cá nhân không có tham số danh tính | PASS, sau sửa |
| US-P3-05 | Chat riêng `/chat`: SSE, tải lại giữa chừng không mất chữ, nút Dừng huỷ tới provider, khoá giờ thi, dùng được ở 375 px | PASS, sau sửa |
| US-P3-06 | Threads `/threads`: tường lửa PII, hộp thoại đúng hai lối, AI trả lời nền, giảng viên / TA xác nhận / sửa / loại | PASS, sau 2 vòng sửa |
| US-P3-07 | Độ tin cậy tất định (sinh viên chỉ thấy câu "chưa đủ chắc chắn"); bộ E1 200 mẫu + `eval_pii.py` | PASS, sau sửa |
| US-P8-02 | `/documents` (Staff), `/library` (tìm, xem trước, tải, "Hỏi AI về tài liệu này") | PASS |
| US-P8-03 | Lịch gộp bằng UNION (buổi học + bài thi + sự kiện), feed ICS ký bằng token băm, nhắc 24 h, tool lịch | PASS, sau sửa (#15) |
| US-P3-08 | Seed (hội thoại, thread, tài liệu, lịch), "Hôm nay" (`continue[]`, `AI_CONFIRM`), k6 `chat`, `gate-p3.sh`, `gate-p8.sh` | PASS, sau sửa |

**Spec:**
- `docs/specs/FEAT-private-chat-pii/` v1.6;
- `docs/specs/FEAT-docs-calendar/` v1.6.

Tech Lead thẩm định cả hai (P3: 16 TLR, 1 chặn; P8: 12 TLR, 1 chặn) trước khi `APPROVED`.

## Số liệu
- **Test case:**
  - 535 ở mức story;
  - cổng P3 56 mục, cổng P8 59 mục;
  - QC đặt 69 câu hỏi `Q-QC`, BA trả lời hết.
- **E1** (`eval_pii.py`, 200 mẫu): recall **0,960**, chặn nhầm **0,000**, precision 1,000, F1 0,980. Ngưỡng là recall ≥ 0,95 và chặn nhầm ≤ 0,05. Bộ mẫu do đội tự soạn từ dữ liệu mô phỏng nên kết quả có thể lạc quan.
- **Rò dữ liệu:** `TestNoPayloadLeak` quét 146 payload gửi provider giả, 0 MSSV / họ tên roster. Tên người ngoài danh sách lớp chưa bắt được (NER đã cắt theo D46).
- **Tốc độ** (k6, provider giả có trễ):
  - sự kiện SSE đầu tiên p95 **85,1 ms** (ngưỡng 300 ms) với 50 sinh viên, 0 lỗi;
  - TTFT khi trúng cache p95 44,3 ms, khi phải truy xuất p95 342,6 ms.
- **`docling-serve`** (CPU arm64):
  - bản có lớp chữ 0,78–0,84 s / trang;
  - bản scan (OCR Tesseract tiếng Việt) 4,52 s / trang;
  - đỉnh RAM 1,81 GiB.
- **Seed trên DB trống:** 392 s.
- **Khối lượng:** 91 commit; `backend-go` + `frontend` + `scripts` + `benchmarks` thêm 38.147 dòng.
- **Góp ý:**
  - đánh số #1–#17, kèm QC Q1–Q3 và dev D1–D3;
  - PM quyết hết;
  - chủ dự án chốt 4 câu: tên người đăng thread công khai, khoá giờ thi chặn chat + đăng thread, đổi schema, nút "Nhờ giảng viên" ẩn tới sprint 7.

## Lỗi thật tìm ra và đã sửa
- **Truy xuất hỏng sau PgBouncer** (P3-05, Cao): mọi câu hỏi nội dung môn lỗi vì `document_ids` kiểu `[]uuid` không mã hoá được trong chế độ exec của pool runtime. Đã sửa, kèm test chạy trên pool thật.
- **Docling tắt** (P8-01, Cao): tài liệu bị đánh `FAILED` ngay thay vì chờ và thử lại. Đã sửa: tài liệu giữ `QUEUED`, thử lại sau 5 s / 30 s / 2 phút. `retry` / `reindex` nay tôn trọng `Idempotency-Key`.
- **Sự kiện outbox không có handler** (P3-06): `thread.created`, `thread.post_decided` rơi vào dead-letter. Đã sửa và thêm `TestEveryEmittedTopicHasHandler`, test quét mọi topic được phát.
- **Định tuyến bỏ sót cách hỏi tự nhiên** (P3-04, P3-05, P8-03):
  - "Khi nào em thi cuối kỳ?" không vào lịch thi;
  - luật sửa xong lại quá rộng, bắt nhầm câu hỏi về nội dung quy chế thi;
  - "Tuần tới có gì không?" không vào lịch (#15).
- **Seed hỏng trên DB trống** (P3-08): một thread mẫu là câu hỏi cá nhân nên bị chính tường lửa chặn. Với provider giả thì không thread nào có bài AI. k6 đếm `409` là lỗi.
- **CI:**
  - JS mỗi route 279,7 KB vượt ngân sách 256 KB (hồi quy thật). Đã tách nạp lười màn chat / Threads.
  - Test "Redis chết" rò cấu hình sang test khác.
  - Test hạn mức theo phút và token lịch phụ thuộc đồng hồ thật.
  - Phải sửa **3 vòng** mới xanh, vượt quy ước 2 vòng của cổng. Cả 3 vòng đều là lỗi test hoặc ngân sách, không có lỗi chức năng.

## Quyết định trong sprint
- **Chủ dự án chốt:**
  - Threads hiện tên người đăng với cả lớp;
  - khoá giờ thi chặn chat riêng và đăng thread mới;
  - đồng ý migration thêm `content_chunks.tsv` + GIN và lưu `ics_token` dạng băm có CHECK;
  - các câu `[CHỦ DỰ ÁN]` còn lại theo mặc định của BA.
- **#7, `docling-serve`:**
  - một lần gọi async cho cả tệp, ngưỡng ≤ 1 s / trang;
  - `pypdfium2`, vì `docling_parse` làm chết server trên arm64;
  - OCR Tesseract tiếng Việt cho bản scan, ngưỡng nới thành ≤ 6 s / trang (#16).
- **#13, số migration:** lịch thành `00011_calendar` vì `00010` đã vào trước. Không có `00009`.
- **#17, CI:** áp dụng TL-CI #1, #2, #4 (lọc đường dẫn, tách job Frontend, sửa cache Go). #3, #5 để sprint 7.

## Nợ
- **Chưa kiểm được:**
  - `OVERLOADED` và quá hạn che (`mask-timeout`) không ép được với provider giả (TC-49/50 P3-02, TC-26 P3-05);
  - mạng 3G chậm chưa đo lại;
  - các bước phá cổng có chủ đích (`dev:down` stack dùng chung) không chạy.
- **Mã:**
  - nhiều test vẫn dùng `pgxpool.New` mặc định thay vì `RuntimePool` (lỗi PgBouncer có thể ẩn ở gói khác);
  - replay `Idempotency-Key` ở `complete` chưa có test HTTP riêng.
- **Hạ tầng:**
  - vai DB của gateway / worker là superuser (QC Q2), cần đổi sang `NOSUPERUSER NOBYPASSRLS` (PR);
  - báo upstream lỗi `docling_parse` trên arm64;
  - TL-CI #3 (shard Playwright + chia Lighthouse), #5 (matrix Go).
- **Sản phẩm:**
  - thu hồi chia sẻ tài liệu (TLR-5 P8);
  - "Nạp lại tri thức đã xác nhận" để P4 làm;
  - nút "Nhờ giảng viên" ở sprint 7;
  - tài liệu mới trên "Hôm nay" hoãn lại;
  - bộ E1 nên có người ngoài đội soạn thêm mẫu.

## Chủ dự án tự kiểm
- https://localhost: stack bản `main` sau merge, seed mới. Mặc định provider giả; muốn có câu trả lời thật thì thêm khoá vào `/settings/llm`.
  - Đăng nhập `sv.gioi@edupilot.local`, vào `/chat`, hỏi một câu về quy chế: câu trả lời có trích dẫn.
  - Gõ tên + MSSV của mình: câu trả lời hiện đúng tên, có dòng "Đã ẩn N thông tin cá nhân trước khi gửi cho AI". Payload gửi đi chỉ chứa `[[SV_…]]`; chỉ xem được bằng route thử `_test/llm/payloads` trên bản build `testroutes`.
- Ở `/threads`, đăng "Em Nguyễn Văn A 20221234 được mấy điểm?": bị chặn, hộp thoại có đúng hai lối; bấm chuyển sang chat riêng thì chữ còn nguyên.
- Tải lại trang giữa lúc AI đang trả lời: chữ không mất.
- Giảng viên vào `/documents`, tải một PDF lên: tiến độ chạy tới `READY`. `/calendar`: tạo link ICS và thêm vào Google Calendar.
