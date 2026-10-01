# QC test case — US-PROTO-01 (Sinh viên)
Nguồn: `docs/sprints/1.5/spec/US.md` + `SRS.md` 4.1, 4.3. Hộp đen. Công cụ **C** = `proto-curl.sh`, **B** = trình duyệt (sweep + tay), như `tc-US-PROTO-00.md`. Dữ liệu gốc: `Đặt lại dữ liệu demo` trước mỗi chuỗi TC.
Câu nhập nguyên văn D1–D3: `docs/DEMO_SCRIPT.md` mục 3 với `{HO_TEN_B}` = Trần Thu Uyên, `{MSSV_B}` = 20229002, `{HO_TEN_C}` = Lê Quang Huy.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-01-01 | AC1 | – | **C** `tc_01_01` (12 route SV B) | Đều `MO` |
| TC-01-02 | AC1 | – | **B** Sweep SV B 1440 px: `/`, `/chat`, `/threads`, `/threads/t-cbc`, `/practice`, `/practice/at-symmetric`, `/practice/history`, `/library`, `/calendar`, `/me`, `/assignments/bt03`, `/join` | Mỗi route đúng một `h1`, khung nhìn đầu khớp DESIGN §14.x, **một** nút chính đỏ mỗi vùng, ≤ 2 phụ hiện ra; ảnh `shots/` |
| TC-01-03 | AC1 / SRS 4.3 `/` | B, A, D | **B** `/` của A / B / D | A: "Ôn lại Mật mã đối xứng — bạn sai 4/7 câu gần nhất · 15 phút"; B: "QUIZ01 đóng sau 18 giờ — bạn chưa làm · 20 phút"; dòng thời gian có buổi 10 lớp 1 09:00–11:30 P.302 (đang diễn ra) và hạn QUIZ01; D: ô nhập mã thay khuyến nghị; bấm khuyến nghị → đúng màn |
| TC-01-04 | AC2 | B, `/chat` lớp 1, phiên mới | **B** Gõ D1, gửi | Dòng "Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI" + `Tìm hiểu` (mở giải thích ngắn tại chỗ); trả lời **chảy dần** (~3 s) nêu vắng 2 buổi (10/09, 08/10) và +0,75; khối gọn (Vắng 2 / Phát biểu 3 · +0,75); `Nguồn tham khảo (2)` mở sẵn (Quy chế môn học tr. 2; Sổ điểm danh lớp 761987); `Hữu ích` / `Không hữu ích` / `Nhờ giảng viên hỗ trợ`; **không có `[[`** trong trang (Ctrl+F, `innerText`) |
| TC-01-05 | AC2 | Sau TC-01-04 | **B** Gõ D2 | "Mình chỉ trả lời được thông tin của chính bạn. Nếu cần trao đổi về bạn khác, hãy hỏi giảng viên." Không con số nào về Lê Quang Huy (không "5", "vắng", "1,5") |
| TC-01-06 | AC2 / FR-X9 | – | **B** Bật `prefers-reduced-motion: reduce` (emulate) rồi gửi D1 | Trả lời hiện ngay toàn văn, không chảy |
| TC-01-07 | AC3 (Q1 mặc định: tự chuyển) | Sau TC-01-05 | **B** Gõ D3 | "AI chưa đủ chắc chắn về câu này"; câu hỏi tự chuyển giảng viên; chỗ nút thành "Đang chờ giảng viên · vừa gửi" |
| TC-01-08 | AC3 | Sau US-PROTO-02 TC-02-02 (GV trả lời D4) | **B** Đổi vai về B → `/chat` | Câu trả lời có nhãn giảng viên (D4 nguyên văn); `Đã rõ` đóng câu hỏi; sau bấm trạng thái đóng |
| TC-01-09 | AC4 | B ở `/threads` | **B** Soạn bài có `20229002` (và biến thể: "điểm của em là 8,3") → đăng | Mở Dialog đúng hai lối `Chuyển sang chat riêng` / `Ẩn thông tin rồi đăng`; không đăng khi chưa chọn |
| TC-01-10 | AC4 | Sau TC-01-09 | **B** Chọn `Chuyển sang chat riêng` | Sang `/chat` có sẵn bản nháp, chữ nguyên vẹn |
| TC-01-11 | AC4 | – | **B** Chọn `Ẩn thông tin rồi đăng`; thêm bài sạch | Bài đăng có "[đã ẩn]" thay MSSV; bài sạch đăng ngay, ~2 s sau có câu AI `Chờ xác nhận` |
| TC-01-12 | AC5 | – | **C** `tc_01_05` | Không từ kỹ thuật AI (lệnh AC5); không RAG/PII/LLM/prompt/placeholder/`[[SV_`; `/assignments/bt03` trước công bố không có 8,5 / 7,0 / "điểm nháp" |
| TC-01-13 | AC5 | – | **B** Sweep SV B/A/C/D: `innerText` mọi route SV (kể cả sau tương tác: sau D1–D3, Dialog, Drawer, `?state=error`) | Không từ cấm; không số độ tin cậy ("0,42", "0,80", "%"); không nhãn "Cần chú ý" / ghi chú / điểm nháp của chính mình (C có nhãn nhưng chỉ GV/TA thấy: **C đăng nhập ở `/me` không thấy "Cần chú ý"**) |
| TC-01-14 | AC6 | – | **B** Viewport 375 × 812: đi lần lượt các route của US | Không cuộn ngang; vùng chạm ≥ 44 px (sweep `nSmall = 0`); bottom nav ≤ 5 đích; lịch sử chat ẩn; ảnh `shots/` 375 |
| TC-01-15 | AC6 | – | **B** 375 px: Dialog hai lối, Drawer "Thêm", `/calendar` | Dialog / Drawer vừa màn hình, không tràn; `/calendar` mặc định **Danh sách** |
| TC-01-16 | AC7 | B ở `/me` | **B** Đọc `/me` | "Điểm quá trình hiện tại 8,3 (tạm tính)"; giải trình TB bài tập 7,5 + cộng 0,75 = 8,25 → 8,3; 2 vắng / 9 buổi; QUIZ01, giữa kỳ 05/11; dòng "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường"; **không** nhãn rủi ro / ghi chú / điểm nháp |
| TC-01-17 | AC7 | – | **B** What-if: nhập 8,0 → 11 → −1 → "abc" → xoá trống | 8,0 → "8,3"; 11, −1, abc → lỗi **tại ô** (không alert, không mất chữ); trống → không lỗi giả. 8,0 với QT 8,8 (sau công bố) cũng ra 8,3 |
| TC-01-18 | AC7 | Sau TC-02-03 (điểm danh) | **B** `/me` của B | QT = 8,5 |
| TC-01-19 | AC7 (xem proposals #13) | Sau TC-03-02 (công bố BT03) | **B** `/me` của B | QT = 8,8 theo spec hiện hành (nếu PM chọn phương án (b) của #13 → 8,7; TC sửa theo quyết định đó) |
| TC-01-20 | AC7 | – | **B** SV A đổi sang lớp 2 ở `/me` | "Lớp này chưa có công thức điểm chính thức" thay phần giải trình |
| TC-01-21 | AC8 (nhánh lỗi) | D, `/join` | **B** Nhập `ABCDEFG` | "Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên." — cùng câu cho mã sai chữ, mã hết hạn, mã rỗng (không lộ lý do) |
| TC-01-22 | AC8 | – | **B** Nhập sai 5 lần liên tiếp | "Thử lại sau 10 phút" |
| TC-01-23 | AC8 | – | **C** `tc_01_08`; **B** nhập `BX4P9TW` → xem trước → `Tham gia lớp` | Xem trước: An ninh mạng · 761988 · TS. Lê Thu Hà · HK1 2026–2027; sau bấm: "Đã gửi yêu cầu, chờ giảng viên duyệt"; `AN7K2MQ` → vào lớp ngay |
| TC-01-24 | AC8 | B ở `/chat` | **B** `/chat?state=error`, gõ chữ vào composer rồi lỗi | Chữ trong composer còn nguyên |
| TC-01-25 | AC9 (phân quyền) | – | **C** `tc_01_09` (TA / GV / Admin × 6 route SV; Admin × `/threads`, `/calendar`) | Đều `CHAN` |
| TC-01-26 | AC9 | – | **C** `tc_00_matrix` hàng SV | SV mở được đúng nhóm SV + SV_TA_GV; `ANSWER_KEY` không bao giờ hiện ở `/library` (**B**: đếm tài liệu = 6 bài giảng + quy chế trường + 2 đề cũ) |
| TC-01-27 | SRS 4.3 `/practice` | B | **B** Chọn chủ đề → `/practice/at-symmetric`; chọn đáp án | Phản hồi ngay + giải thích có nguồn; 10 câu (trắc nghiệm + 1 trả lời ngắn) |
| TC-01-28 | SRS 4.3 QUIZ01 | B | **B** Mở QUIZ01 | Không phản hồi từng câu; có giờ; ghi chú "Trong lúc làm bài, Chat riêng chỉ trả lời câu hỏi thủ tục"; `Nộp bài` qua xác nhận |
| TC-01-29 | SRS 4.3 | – | **B** `/library`: tìm "hash"/"zzzz"; `Xem`; `Hỏi AI về tài liệu` | Lọc tại chỗ; không kết quả → rỗng có hướng; `Hỏi AI về tài liệu` sang `/chat` có ngữ cảnh |
| TC-01-30 | SRS 4.3 | – | **B** `/threads/t-cbc` + thread đã xác nhận; `Báo cáo` | Câu AI `Chờ xác nhận` có nguồn (SV không có nút Xác nhận / Loại); câu "Đã được giảng viên xác nhận"; `Báo cáo` → "Đã gửi báo cáo" |
| TC-01-31 | SRS 4.3 `/assignments` | B | **B** `/assignments/bt03` trước công bố | Đã nộp 23/10 08:10, nhãn "Nộp muộn 1 ngày", file `bt03-tran-thu-uyen.pdf`, "Đang chấm" (không số); `/assignments/khong-co` → rỗng |
| TC-01-32 | SRS 4.3 `/assignments` | Sau công bố | **B** Tải lại | Điểm, nhận xét 4 tiêu chí, mỗi tiêu chí trích một đoạn bài của B; `Yêu cầu xem lại` → form chọn tiêu chí + lý do → "Đã gửi yêu cầu" |
| TC-01-33 | SRS 4.3 trạng thái | – | **B** Rỗng / lỗi / loading từng route SV (`?state=`) | Rỗng có một hành động ("Hôm nay bạn không có việc gấp" + `Luyện đề` …); lỗi có `Thử lại` |

## Nhánh lỗi
TC-01-21, 22, 24 (AC8); TC-01-17 (số ngoài 0–10); TC-01-33 (`?state=`).

## Phân quyền
TC-01-25, 26 (AC9, ma trận SRS 2); TC-01-13 (SV không thấy nhãn rủi ro / điểm nháp của chính mình).

## Kiểm chéo
- Lời văn SV: TC-01-12, 13 (DESIGN §13). Phản mẫu §21: ảnh từng route (sweep). §22 10 điều kiện: đánh dấu từng route trong report.
- 375 px: TC-01-14, 15. Không PII: MSSV chỉ xuất hiện ở chữ SV tự gõ (D1) và bị thay trước khi "gửi".
- Không `fetch`: `tc_00_static`.

## Điểm khó kiểm
- "Chữ chảy" cần quan sát thời gian (~3 s) trong trình duyệt; curl không thấy.
- AC7 con số 8,8 mâu thuẫn rubric (proposals #13): chấm theo spec hiện hành, ghi rõ trong report nếu dev chọn khác.
- Bước Mailpit bỏ (FR-X7): AC3 chỉ kiểm dòng xác nhận mô phỏng.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
(chưa có)
