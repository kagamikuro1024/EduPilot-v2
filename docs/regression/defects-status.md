# Trạng thái lỗi (qc2, tự sinh từ defects-*.md + defects-closed.md; 2026-11-19)

ĐÓNG = đã có TC chạy lại đạt (xem defects-closed.md). ĐÃ SỬA — CHỜ CHẠY LẠI = dev báo sửa, chưa chạy lại. MỞ = chưa có sửa được xác nhận.

|Sprint|DEF|Mức|TC|Trạng thái|Tiêu đề|
|---|---|---|---|---|---|
|1|DEF-R1-01|Thấp||MỞ|DEMO_SCRIPT nêu spec E2E không có trong FLOWS.md|
|1.5|DEF-R1.5-01|Trung bình|TC-00-57, TC-00-59|ĐÓNG|Đã đăng nhập mà mở đường dẫn không có thật bị đưa về /login thay vì màn «Không t|
|1.5|DEF-R1.5-02|Thấp|TC-00-36|ĐÓNG|TA: nút hồ sơ ở topbar tràn ngang ở 375 và 390 px|
|2|DEF-R2-01|Trung bình|TC-PG06-01|ĐÓNG|`openapi.yaml` không qua `redocly lint` (1 lỗi struct)|
|2|DEF-R2-02|Cao|TC-PG06-10, TC-PG06-11, TC-PG06-28|ĐÓNG|`go test ./internal/contract` đỏ: TestStatusParity / TestSpec_EveryDocumentedSta|
|2|DEF-R2-03|Trung bình||ĐÃ SỬA — CHỜ CHẠY LẠI|`go test ./internal/store` đỏ: TestLLMSchema (6 bảng `llm_%` ≠ 9) — ĐÃ SỬA c272d|
|2|DEF-R2-04|Trung bình||MỞ|`go test ./internal/contract` KHÔNG có `-tags testroutes` đỏ cố định|
|3|DEF-R3-01|Thấp|TC-P105-44|MỞ|Bảng «Mức dùng theo tác vụ» hiện chữ thô «EMBEDDING»|
|3|DEF-R3-02|Thấp|TC-PU01-01|ĐÓNG|`docs/design/DESIGN_TOKENS.css` lệch `frontend/src/shared/styles/tokens.css` ở 2|
|3|DEF-R3-03|Thấp|TC-GATEPU-01, TC-PU01-05|MỞ|`pnpm lint` có 5 cảnh báo (nền sprint 3 = 0)|
|3|DEF-R3-04|Thấp|TC-P101-24, TC-P104-16|ĐÓNG|`DecryptKey(` được gọi ngoài `internal/llm/` và `internal/llmconfig/`|
|3|DEF-R3-05|Trung bình|TC-P102-14, TC-P103-19, TC-P103-20|ĐÓNG|Làn BATCH bị gọi lại/chờ bất thường trên ep-r (TC-P102-14, TC-P103-19, TC-P103-2|
|3|DEF-R3-SMK-01|?|TC-P103-08, TC-P103-09|ĐÓNG|RÚT LẠI (lỗi đo của QC, KHÔNG phải lỗi sản phẩm): «bucket RPM cho qua 50–65 lời |
|3|DEF-R3-SMK-02|Thấp|TC-PU05-29, TC-PU05-35|MỞ|Nhánh `ci/ui-drift` chưa được xoá (TC-PU05-35)|
|3|DEF-R3-SMK-03|Trung bình|TC-GATEPU-15, TC-PU05-40|ĐÓNG|[ĐÓNG @07a6b69c, TC PASS 2026-11-19] Sinh viên thấy «Chi tiết kỹ thuật … trace_i|
|4|DEF-R4-01|Trung bình|TC-P208-02|ĐÓNG|Mã lớp phân biệt hoa/thường: tạo được hai lớp `SK20D91` và `sk20d91`|
|4|DEF-R4-02|Thấp|TC-P210-03|ĐÓNG|Import roster nhận tệp `.xlsx` thực chất là văn bản (TC-P210-03)|
|4|DEF-R4-03|Thấp|TC-P202-46|MỞ|Thấp — TC-P202-46 (sprint gốc 4, US-P2-02 AC15)|
|4|DEF-R4-04|Thấp|TC-P206-28|MỞ|Thấp — TC-P206-28 (sprint gốc 4, US-P2-06 AC8)|
|4|DEF-R4-05|Trung bình|TC-P201-11|MỞ|Trung bình — TC-P201-11 (sprint gốc 4, US-P2-01 AC3; vi phạm US-P2-04 AC13)|
|4|DEF-R4-06|Thấp|TC-P202-47|MỞ|Thấp — TC-P202-47 (sprint gốc 4, US-P2-02 AC15)|
|4|DEF-R4-07|Thấp|TC-P201-19|MỞ|Thấp — TC-P201-19 (sprint gốc 4, US-P2-01 AC6)|
|4|DEF-R4-08|Thấp|TC-GATEP2-09, TC-P212-14|MỞ|Thấp — TC-GATEP2-09 / TC-P212-14 (sprint gốc 4, US-P2-12 AC7)|
|4|DEF-Q2-S4-01|Thấp|TC-P212-14|MỞ|Seed tự chạy trên DB trống mất 752 s, spec AC7 (US-P2-12) đòi ≤ 3 phút (TC-P212-|
|5|DEF-R5-01|Thấp|TC-PE03-52|ĐÓNG|Nút đầu trang AppShell: axe `label-content-name-mismatch` (chỉ khi bật thẻ wcag2|
|5|DEF-R5-02|Thấp|TC-PE06-08|ĐÓNG|Vùng cuộn ở màn làm bài code không focus được bằng bàn phím|
|5|DEF-R5-03|Trung bình|TC-PE03-49|MỞ|Ngân hàng câu hỏi: nháp tự lưu nhưng KHÔNG khôi phục khi mở lại (mất chữ gõ dở)|
|5|DEF-R5-04|Trung bình|TC-PE03-51|MỞ|Ngân hàng câu hỏi: Enter trên hàng mở Drawer rồi Drawer đóng ngay (bàn phím khôn|
|5|DEF-R5-05|Thấp|TC-PE01-39|ĐÃ SỬA — CHỜ CHẠY LẠI|`go test ./internal/contract/...` không tag testroutes đỏ (lệnh trong TC tổng và|
|5|DEF-R5-06|Thấp|TC-PE06-27|MỞ|Lần nộp code: nhãn «Đã nhận, đang xếp hàng» (QUEUED) không bao giờ hiện|
|5|DEF-R5-07|Thấp|TC-PE07-18|MỞ|Tín hiệu liêm chính cuối cùng bị mất khi đóng tab (không gửi lúc pagehide)|
|5|DEF-R5-08|Thấp|TC-PE08-31|MỞ|Màn kết quả SV thiếu dòng tóm tắt và từng câu không gập mặc định|
|5|DEF-R5-09|Thấp|TC-PE08-38|MỞ|Giao diện Staff `/exams/[id]/results` lệch US AC10 ở 3 điểm|
|5|DEF-R5-10|Thấp|TC-PE03-33|MỞ|Sửa test sau đóng: bài cũ không hiện «đã chấm theo bộ test cũ»|
|5|DEF-R5-11|Thấp|TC-PE08-04|MỞ|Chấm trắc nghiệm đồng bộ không phát sự kiện SSE `exam.attempt`|
|5|DEF-R5-12|Trung bình|TC-PU06-17, TC-UI7-06|MỞ|`pnpm lint` ở frontend rc=1 (TC-PU06-17, TC-UI7-06)|
|5.5|DEF-R5.5-01|Trung bình|TC-UI7-06|MỞ|`pnpm lint` frontend rc=1 (TC-UI7-06)|
|6|DEF-R6-01|Thấp|TC-P3-01-49|MỞ|Vai DB của ứng dụng là siêu quyền|
|6|DEF-R6-02|Cao|TC-P3-02-21|ĐÓNG|Bộ nhớ đệm roster PII không bị vô hiệu khi sinh viên vào lớp (duyệt hoặc vào bằn|
|6|DEF-R6-C1|Cao|TC-P8-03-11, TC-P8-03-11b|ĐÓNG|Sửa/xoá sự kiện từ giao diện lịch trả 404 (id gộp calendar_event:<uuid>); eb8f16a0, chạy lại P 21/11 (xoá) và 25/11 (sửa)|
|6|DEF-R6-C-01|Thấp|TC-P3-06-42|ĐÓNG|Dòng khoá ở /threads dùng chữ của chat thay vì "Đăng bài tạm khóa…"|
|6|DEF-R6-U1|Thấp|TC-P3-05-35|MỞ|Chữ «Tìm hiểu» của chat riêng khác SRS|
|6|DEF-R6-U2|Thấp|TC-P3-06-09|MỞ|Tiêu đề hộp thoại PII ở Threads khác SRS|
|6|DEF-R6-U3|Thấp|TC-P3-06-34|MỞ|Nút đăng lại khi mất mạng ở Threads khác SRS|
|6|DEF-R6-U4|Thấp|TC-P3-06-37|MỞ|Trạng thái rỗng /threads thiếu câu mời|
|6|DEF-R6-U5|Thấp|TC-P3-02-50|MỞ|Chữ lỗi MASK_FAILED của chat khác SRS|
|6|DEF-R6-U6|Thấp|TC-P3-05-26|MỞ|Chữ CHAT_UNAVAILABLE ở UI khác SRS|
|6|DEF-R6-D1|Trung bình|TC-P3-05-42, TC-P8-02-20|MỞ|Trích dẫn cũ vẫn hiện tên tài liệu đã bị xoá, không có «Nguồn đã gỡ»|
|6|DEF-R6-P1|Thấp|TC-P8-01-18|MỞ|Bộ chia đoạn không gộp mẩu < 200 rune qua ranh giới trang|
|6|DEF-R6-P2|Trung bình|TC-P3-01-05, TC-P3-03-31, TC-P3-04-45|MỞ|Chia sẻ tài liệu không tăng ep:rag:ver của lớp đích|
|7|DEF-R7-01|Trung bình||MỞ|CI chạy đủ job vượt ngưỡng cứng 10 phút của US-CI-02|
|7|DEF-R7-02|Trung bình|TC-P4-02-14|ĐÃ SỬA — CHỜ CHẠY LẠI|Chuông gộp TICKET_NEW không báo ticket mới sau khi hàng chờ về 0|
|7|DEF-R7-04|Thấp|TC-P4-03-36|MỞ|Ctrl+↵ gửi trả lời không sang phiếu kế khi bộ lọc là «Đang chờ»|
|7|DEF-R7-05|Thấp|TC-P4-03-33|MỞ|Ô trả lời của hộp thư không «dính đáy» ở 375 px|
|7|DEF-R7-06|Thấp||MỞ|Timeout của job CI không đạt «≥ 2× thời gian thực tế»; `changes` và `judge-attac|
|7|DEF-R7-07|Thấp||MỞ|Shard Playwright rỗng chạy lại CẢ BỘ test thay vì 0 test|
|7|DEF-R7-08|Trung bình|TC-P5-02-33|MỞ|Nút «Lưu điểm danh» bị khoá khi mất mạng, không xếp hàng `complete:true`|
|7|DEF-R7-09|Trung bình|TC-P5-03-29|MỞ|Nhịp heartbeat khi Redis dừng mất ~8 giây (SRS: ≤ 1 s)|
|7|DEF-R7-10|Thấp|TC-P5-01-26|MỞ|«Xem trước» tạo lịch buổi học gửi Idempotency-Key|
|7|DEF-R7-11|Thấp|TC-P4-03-38|MỞ|Trạng thái rỗng của Hộp thư hỗ trợ không đúng chữ / thiếu nút «Xem tất cả»|
|7|DEF-R7-12|Thấp|TC-P4-06-47|MỞ|Menu bài AI của giảng viên ghi «Loại», không phải «Loại khỏi tri thức»|
|8|DEF-R8-s8core-01|Trung bình|TC-P6-01-20, TC-P6-02-25|ĐÓNG|Giải trình không có dòng «không dự → 0 theo quy chế»|
|8|DEF-R8-s8core-02|Thấp|TC-P6-02-15|ĐÓNG|float64 chuyển điểm thi trong seed tool|
|8|DEF-R8-A1|Thấp|TC-P6-03-19, TC-P6-03-36|ĐÓNG|Trích công thức: đầu ra không phải JSON bị thử lại ×3 và báo «AI đang không dùng|
|8|DEF-R8-A2|Thấp|TC-P6-04-03, TC-P6-07-47|ĐÓNG|Đóng Drawer / Dialog làm focus rơi về <body> thay vì nút mở (TC-P6-04-03, TC-P6-|
|8|DEF-R8-A3|Trung bình|TC-P6-03-19|ĐÓNG|Trích công thức: trọng số ngoài [0;100] bị đổi thầm thành 0, không có mục PARAM_|
|8|DEF-R8-A4|Thấp|TC-P6-05-37|ĐÓNG|Chat riêng: «Cô tính điểm môn này thế nào?» không được nhận là GRADE_FORMULA (TC|
|8|DEF-R8-A5|Thấp|TC-P6-05-42|ĐÓNG|Xoá tài liệu nguồn của công thức đã xác nhận: UI không hiện «Tài liệu nguồn đã g|
|9|DEF-R9-01|Thấp|TC-P7-01-G06, TC-P7-02-G10, TC-P7-03-G03|ĐÓNG|Tên test ở dòng `Kiểm:` của US không có trong repo (hoặc ở gói khác)|
|9|DEF-R9-02|?||RÚT/LOẠI|ĐÃ LOẠI: `go test ./internal/contract/...` đỏ lúc tải cao|
|9.1|DEF-R9.1-01|Thấp|TC-UX04-3, TC-UX08-5|ĐÓNG|luật `:hover` ngoài `@media (hover: hover) and (pointer: fine)` (TC-UX04-3, TC-U|
|9.1|DEF-R9.1-02|Thấp|TC-UX07-8|ĐÓNG|chuỗi «Kiểm duyệt Threads» hiển thị (TC-UX07-8)|
|9.1|DEF-R9.1-03|Thấp|TC-UX08-50|ĐÓNG|axe color-contrast trên `/` ngay sau tải (TC-UX08-50)|
|9.1|DEF-R9.1-06|Thấp|TC-UX12-2|MỞ|chấm đã chọn (Muộn / Vắng phép / Vắng) xám khi chuột đang rê lên ô (TC-UX12-2)|
|9.1|DEF-R9.1-07|Thấp|TC-UX16-14|MỞ|nháy lưu ô sổ điểm khi `reduce`: giữ ≈ 160 ms thay vì ≥ 300 ms (TC-UX16-14; AC m|
|9.1|DEF-R9.1-08|Thấp|TC-UX09-16|MỞ|`scripts/landing-video/record.mjs` không chạy được ở HEAD (TC-UX09-16, Thấp)|
|9.5|DEF-R9.5-01|Thấp|TC-N01-17|ĐÓNG|`refuseDemo` bỏ lọt `[::1]` và `host.docker.internal`|
|9.5|DEF-R9.5-02|Thấp|TC-N01-18|ĐÓNG|`canary-scan.sh` không xoá nhà cung cấp canary|
|9.5|DEF-R9.5-03|Thấp|TC-N04-12|MỞ|load-a/c không có mẫu `docker stats` / ghi «không OOM, CPU < 95 %»|
|9.5|DEF-R9.5-04|Thấp|TC-N05-03|MỞ|Không có `00018_indexes_tuning.sql`, đề xuất P-95-8 (dev) chưa có quyết định PM|
|9.5|DEF-R9.5-05|Trung bình|TC-N10-10|ĐÓNG|9.5 sửa test có sẵn|
|9.5|DEF-R9.5-06|Thấp|TC-N10-06|MỞ|PROGRESS.md thiếu dòng Nợ cho TL-2c/d/e/f|
|9.5|DEF-R9.5-07|Thấp|TC-N04-15, TC-N09-11|MỞ|Báo cáo cuối không nêu hạn chế Mac ở từng mục|
|9.5|DEF-R9.5-08|Thấp|TC-N07-10|MỞ|Không ghi lần đo bị nhiễu|
|9.5|DEF-R9.5-09|Trung bình|TC-N08-01|MỞ|`meta.config` của tệp real-llm thiếu chuỗi model và RPM|
|9.5|DEF-R9.5-10|Trung bình|TC-N08-06|MỞ|R3 (mask) thiếu tỷ lệ placeholder và số lần mô hình đổi placeholder|
|9.5|DEF-R9.5-11|Trung bình|TC-N08-11|MỞ|R5 không đủ mẫu / AI Threads không có đầu ra|
|9.5|DEF-R9.5-12|Cao|TC-N08-16, TC-N08-17|ĐÓNG|`real_llm.py` không có `--resume`, không dừng sớm khi hết hạn mức|
|9.5|DEF-R9.5-13|Thấp|TC-N09-01|MỞ|Báo cáo research thiếu mục NFR-02/03 và «6 phần»|
|9.5|DEF-R9.5-14|Cao|TC-N09-04, TC-N09-05, TC-N09-13|MỞ|Báo cáo research lệch tệp nguồn sau khi đo lại qua 9router|
|9.5|DEF-R9.5-15|Thấp|TC-N09-07|MỞ|«Câu cần vá» không có tệp:dòng|
|9.5|DEF-R9.5-16|Thấp|TC-N01-12|MỞ|`check-meta.mjs` không kiểm giá trị `demo_stack`|
|9.5|DEF-R9.5-17|Trung bình|TC-N02-02, TC-N02-03|MỞ|Seed không chạy lại được sau khi dữ liệu bị sửa; check-*-seed đếm tuyệt đối|
|9.5|DEF-R9.5-18|Thấp|TC-N02-06|MỞ|Hôm nay của Admin trống ngay sau seed|
|9.5|DEF-R9.5-19|Thấp|TC-N02-09, TC-N02-11|ĐÓNG|Seed không từ chối khi thiếu hoặc rỗng `SEED_DEFAULT_PASSWORD`|
|9.5|DEF-R9.5-20|Trung bình|TC-N06-04|MỞ|Redis tắt: chat treo 39,5 s (TL-2d còn mở)|
|9.5|DEF-R9.5-21|Trung bình|TC-N08-17|MỞ|`real_llm.py` chết bằng Traceback khi server đóng kết nối; `--resume` chỉ ở mức |
|9.5|DEF-R9.5-22|Thấp|TC-N01-03|MỞ|README «Đo trên Mac» thiếu `--email E --name N` ở `gateway admin create` (TC-N01|
|9.5|DEF-R9.5-23|Thấp|TC-N05-02|MỞ|10 truy vấn «nặng nhất» của dev không khớp top theo total_exec_time (TC-N05-02, |
|10|DEF-R10-01|Trung bình|TC-12-14|ĐÓNG|go test ./internal/contract không xanh (TC-12-14)|
|10.5|DEF-R10.5-01|Thấp|TC-D07-02|ĐÓNG|Bảng «Quyết định nhanh» ở /observability nằm trên bảng «7 ngày qua»|
|10.5|DEF-R105S-01|Trung bình|TC-D08-06, TC-D08-07|MỞ|`cascade.spec.ts` không phủ các nhánh (a)–(f) của US-DM-08 AC3|
|10.6|DEF-R10.6-01|Thấp|TC-G03-24, TC-G05-24|ĐÓNG|reflect / regrade-stale qua lớp có cờ tắt trả 409 GUIDANCE_DISABLED trước 404|
|10.7|DEF-R10.7-01|Trung bình||MỞ|Trang Quan sát quyết định lệch khung (teacher: meta lặp tên lớp, lớp rỗng tự dựn|
|10.7|DEF-R10.7-02|Thấp||MỞ|Sổ điểm: đầu trang cao hơn 3,4 px và nút «Chốt điểm» mất khi đang tải / lỗi|
|10.7|DEF-R10.7-03|Thấp||MỞ|Luyện tập sinh viên: tab «Theo chủ đề / Thi thử» biến mất khi đang tải và khi lỗ|
|10.7|DEF-R10.7-04|Thấp||MỞ|/settings/llm: «Máy chủ riêng (tương thích OpenAI)» lặp 7 lần|
|10.8|DEF-R10.8-01|Trung bình|TC-AS08-25, TC-AS09-36, TC-AS09-39|ĐÓNG|`label-content-name-mismatch` (axe, serious) ở ba nút thanh trên của khung ứng d|
|10.8|DEF-R10.8-02|Thấp|TC-AS13-05, TC-AS14-11|MỞ|bảng truy vết SRS §11 trỏ tới 13 tên test không tồn tại|
|10.8|DEF-R10.8-03|Thấp|TC-AS08-20|ĐÃ SỬA — CHỜ CHẠY LẠI|«Đóng» khi mất mạng thử lại bằng Idempotency-Key mới|
|10.8|DEF-R10.8-04|Thấp|TC-AS11-07|ĐÃ SỬA — CHỜ CHẠY LẠI|nút «Quét mã điểm danh» hiện cả khi `qr_code` không được hỗ trợ|
