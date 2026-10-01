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
| TC-01-09 | AC4 | B ở `/threads` | **B** Mở form tạo thread mới | Thấy đủ các trường: Tiêu đề (Input, bắt buộc), Chủ đề (Select danh sách chủ đề môn học), Nội dung chi tiết (Textarea, bắt buộc), checkbox "Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng" (mặc định bật) (Proposal #15) |
| TC-01-10 | AC4 | B ở `/threads` | **B** Soạn bài có `20229002` trong Tiêu đề hoặc Nội dung → bấm `Đăng câu hỏi` | Mở Dialog đúng hai lối: `Chuyển sang chat riêng` và `Ẩn thông tin rồi đăng`; không tạo thread khi chưa chọn lối xử lý |
| TC-01-11 | AC4 | Sau TC-01-10 | **B** Chọn lối 1: `Chuyển sang chat riêng` | Chuyển hướng sang `/chat`, mang toàn bộ nội dung bản nháp (tiêu đề + nội dung), chữ nguyên vẹn không bị mất |
| TC-01-12 | AC4 | B ở `/threads` | **B** Chọn lối 2: `Ẩn thông tin rồi đăng` (hoặc đăng bài sạch) | Thay MSSV thành `[đã ẩn]`; sau khi đăng thành công **chuyển hướng ngay lập tức sang `/threads/${id}`**; trang chi tiết hiển thị câu hỏi gốc và câu trả lời AI `Chờ xác nhận` |
| TC-01-13 | AC5 | – | **C** `tc_01_05` | Không từ kỹ thuật AI (lệnh AC5); không RAG/PII/LLM/prompt/placeholder/`[[SV_`; `/assignments/bt03` trước công bố không có 8,5 / 7,0 / "điểm nháp" |
| TC-01-14 | AC5 | – | **B** Sweep SV B/A/C/D: `innerText` mọi route SV (kể cả sau tương tác: sau D1–D3, Dialog, Drawer, `?state=error`) | Không từ cấm; không số độ tin cậy ("0,42", "0,80", "%"); không nhãn "Cần chú ý" / ghi chú / điểm nháp của chính mình (C có nhãn nhưng chỉ GV/TA thấy: **C đăng nhập ở `/me` không thấy "Cần chú ý"**) |
| TC-01-15 | AC6 | – | **B** Viewport 375 × 812: đi lần lượt các route của US | Không cuộn ngang; vùng chạm ≥ 44 px (sweep `nSmall = 0`); bottom nav ≤ 5 đích; lịch sử chat ẩn; ảnh `shots/` 375 |
| TC-01-16 | AC6 | – | **B** 375 px: Dialog hai lối, Drawer "Thêm", `/calendar` | Dialog / Drawer vừa màn hình, không tràn; `/calendar` mặc định **Danh sách** |
| TC-01-17 | AC7 | B ở `/me` | **B** Đọc `/me` | "Điểm quá trình hiện tại 8,3 (tạm tính)"; giải trình TB bài tập 7,5 + cộng 0,75 = 8,25 → 8,3; 2 vắng / 9 buổi; QUIZ01, giữa kỳ 05/11; dòng "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường"; **không** nhãn rủi ro / ghi chú / điểm nháp |
| TC-01-18 | AC7 | – | **B** What-if: nhập 8,0 → 11 → −1 → "abc" → xoá trống | 8,0 → "8,3"; 11, −1, abc → lỗi **tại ô** (không alert, không mất chữ); trống → không lỗi giả. 8,0 với QT 8,7 (sau công bố) cũng ra 8,3 |
| TC-01-19 | AC7 | Sau TC-02-03 (điểm danh) | **B** `/me` của B | QT = 8,5 |
| TC-01-20 | AC7 (xem proposals #13) | Sau TC-03-02 (công bố BT03) | **B** `/me` của B | QT = 8,7 (SRS 4.1 v3; proposals #13) |
| TC-01-21 | AC7 | – | **B** SV A đổi sang lớp 2 ở `/me` | "Lớp này chưa có công thức điểm chính thức" thay phần giải trình |
| TC-01-22 | AC8 (nhánh lỗi) | D, `/join` | **B** Nhập `ABCDEFG` | "Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên." — cùng câu cho mã sai chữ, mã hết hạn, mã rỗng (không lộ lý do) |
| TC-01-23 | AC8 | – | **B** Nhập sai 5 lần liên tiếp | "Thử lại sau 10 phút" |
| TC-01-24 | AC8 | – | **C** `tc_01_08`; **B** nhập `BX4P9TW` → xem trước → `Tham gia lớp` | Xem trước: An ninh mạng · 761988 · TS. Lê Thu Hà · HK1 2026–2027; sau bấm: "Đã gửi yêu cầu, chờ giảng viên duyệt"; `AN7K2MQ` → vào lớp ngay |
| TC-01-25 | AC8 | B ở `/chat` | **B** `/chat?state=error`, gõ chữ vào composer rồi lỗi | Chữ trong composer còn nguyên |
| TC-01-26 | AC9 (phân quyền) | – | **C** `tc_01_09` (TA / GV / Admin × 6 route SV; Admin × `/threads`, `/calendar`) | Đều `CHAN` |
| TC-01-27 | AC9 | – | **C** `tc_00_matrix` hàng SV | SV mở được đúng nhóm SV + SV_TA_GV; `ANSWER_KEY` không bao giờ hiện ở `/library` (**B**: đếm tài liệu = 6 bài giảng + quy chế trường + 2 đề cũ) |
| TC-01-28 | SRS 4.3 `/practice` | B | **B** Chọn chủ đề → `/practice/at-symmetric`; chọn đáp án | Phản hồi ngay + giải thích có nguồn; 10 câu (trắc nghiệm + 1 trả lời ngắn) |
| TC-01-29 | SRS 4.3 QUIZ01 | B | **B** Mở QUIZ01 | Không phản hồi từng câu; có giờ; ghi chú "Trong lúc làm bài, Chat riêng chỉ trả lời câu hỏi thủ tục"; `Nộp bài` qua xác nhận |
| TC-01-30 | SRS 4.3 | – | **B** `/library`: tìm "hash"/"zzzz"; `Xem`; `Hỏi AI về tài liệu` | Lọc tại chỗ; không kết quả → rỗng có hướng; `Hỏi AI về tài liệu` sang `/chat` có ngữ cảnh |
| TC-01-31 | SRS 4.3 | – | **B** `/threads/t-cbc` + thread đã xác nhận; `Báo cáo` | Câu AI `Chờ xác nhận` có nguồn (SV không có nút Xác nhận / Loại); câu "Đã được giảng viên xác nhận"; `Báo cáo` → "Đã gửi báo cáo" |
| TC-01-32 | SRS 4.3 `/assignments` | B | **B** `/assignments/bt03` trước công bố | Đã nộp 23/10 08:10, nhãn "Nộp muộn 1 ngày", file `bt03-tran-thu-uyen.pdf`, "Đang chấm" (không số); `/assignments/khong-co` → rỗng |
| TC-01-33 | SRS 4.3 `/assignments` | Sau công bố | **B** Tải lại | Điểm, nhận xét 4 tiêu chí, mỗi tiêu chí trích một đoạn bài của B; `Yêu cầu xem lại` → form chọn tiêu chí + lý do → "Đã gửi yêu cầu" |
| TC-01-34 | SRS 4.3 trạng thái | – | **B** Rỗng / lỗi / loading từng route SV (`?state=`) | Rỗng có một hành động ("Hôm nay bạn không có việc gấp" + `Luyện đề` …); lỗi có `Thử lại` |
| TC-01-35 | AC10 | B ở `/threads/[id]` | **B** Xem chi tiết thread bất kỳ | Bố cục panel rõ ràng (FR-X11): Khối câu hỏi gốc (người hỏi, vai trò, thời gian, chủ đề, nội dung), Khối câu trả lời AI (kèm nguồn trích dẫn mở rộng), Danh sách phản hồi thảo luận, Khối Reply Composer ở cuối trang (Proposal #15) |
| TC-01-36 | AC10 | B ở `/threads/[id]` | **B** Bấm `Hỏi trợ lý AI` trong Reply Composer | Sinh câu trả lời gợi ý định hướng (Socratic) kèm nguồn trích dẫn tài liệu |
| TC-01-37 | AC10 | B ở `/threads/[id]` | **B** Gõ nội dung thảo luận có MSSV `20229002` → `Gửi phản hồi` | Bắt PII mở Dialog 2 lối: chuyển chat riêng hoặc ẩn thông tin rồi gửi |
| TC-01-38 | AC10 | B ở `/threads/[id]` | **B** Gõ nội dung thảo luận hợp lệ → `Gửi phản hồi` | Phản hồi mới xuất hiện ngay ở cuối danh sách thảo luận của thread |
| TC-01-39 | AC4 (#18) | B, sau `Đặt lại dữ liệu` | **T** `T1` (hoặc **B** bấm giờ): tiêu đề "Dùng lại IV trong CTR có sao không?", chủ đề Mật mã đối xứng, nhờ AI bật → `Đăng câu hỏi` | Checkbox "Nhờ AI…" mặc định bật; chuyển ngay (≤ 0,8 s) sang `/threads/<id>`; bước 1 "Trợ lý AI đang soạn…" + 3 chấm ~1,2 s (1,0–1,6) → bước 2 chữ chảy có nút `Dừng` 2,5–3,5 s (±0,4) → bước 3 `Nguồn tham khảo (n)` ≤ 0,8 s sau bước 2 → bước 4 nhãn `Chờ xác nhận` ngay sau nguồn; trang tự cuộn tới khối AI; trong bước 1–2 `Hỏi trợ lý AI` và `Gửi phản hồi` khoá |
| TC-01-40 | AC4 | Sau TC-01-39 | **T** `T1i`; so với TC-01-41 | Nội dung mẫu S2: nêu nonce / XOR hai bản mã, kết bằng câu hỏi ngược "…XOR hai bản mã với nhau sẽ cho ra gì?", nguồn Chương 3 tr. 18–20; không đưa đáp án trọn |
| TC-01-41 | AC4 | B | **T** `T2`: tạo thêm "Vì sao ECB làm lộ ảnh?" (Mật mã đối xứng) | Mẫu S1 (so sánh ECB / CBC, kết bằng câu hỏi ngược về ECB và ảnh mã hoá), nguồn Chương 3 tr. 14–17; **khác** câu trả lời ở TC-01-40 |
| TC-01-42 | AC4 / E | Đang chảy (bước 2) | **T** `T3`: bấm `Dừng` sau ~1 s chảy | Giữ phần đã hiện (chữ không chảy tiếp), dòng "Đã dừng", nút `Hỏi lại`; `Hỏi lại` soạn lại và chảy tới `Chờ xác nhận` |
| TC-01-43 | AC4 / FR-X9 | B, emulate `prefers-reduced-motion: reduce` | **T** `T14` | Không chấm, không chảy: hiện ngay kết quả bước 4 (`Chờ xác nhận`) |
| TC-01-44 | AC4 (biên) | B ở `/threads` | **B** Bỏ tích "Nhờ AI…" rồi đăng; riêng: để trống Tiêu đề hoặc Nội dung rồi `Đăng câu hỏi` | Không tích: thread tạo, chuyển sang `/threads/<id>`, không có khối AI, không "đang soạn". Trống: lỗi **tại ô**, không tạo thread, không mất chữ ở ô còn lại |
| TC-01-45 | AC4 / SRS 4.3.1 D1–D2 | B, tạo thread hoặc `Hỏi trợ lý AI` | **B** Văn bản không dấu / hoa thường: "CBC KHAC ECB?" ; "che do cbc ecb" | Cả hai chọn mẫu S1 (chuẩn hoá bỏ dấu, `đ` → `d`, chữ thường) |
| TC-01-46 | AC4 / D2 (ranh giới từ) | B, thread chủ đề Mật mã đối xứng | **B** Đăng "Ivy league là gì?" (chứa "iv" chỉ như một phần của từ khác) | **Không** khớp S2 (khớp nguyên từ có ranh giới hai bên) → nhánh không khớp (xem TC-01-79) |
| TC-01-47 | AC4 / D3 | B | **B** Đăng "Còn CTR thì sao ạ, có cần IV ngẫu nhiên giống CBC không?" | S2 (2 từ khoá: ctr, iv > 1 của S1) |
| TC-01-48 | AC4 / D3 (hoà điểm) | B | **B** Ba thread cùng nội dung "cbc hash" (mỗi mẫu 1 điểm): chủ đề Hàm băm và chữ ký số; chủ đề Mật mã đối xứng; chủ đề Thực hành | Hàm băm → **H1** (ưu tiên mẫu của chủ đề thread); Mật mã đối xứng → **S1**; Thực hành (không có mẫu) → **S1** (mẫu đứng trước trong bảng C) |
| TC-01-49 | AC10 (#18) | B | **B** `/threads`: đọc số phản hồi của 12 thread | Đúng cột n của SRS 4.3.1 B: t-cbc 4, t-salt 3, t-sqli 3, t-pin-rubric 4, t-pin-lab 3, t-rsa-key 2, t-xss 2, t-vpn 2, t-pki 2, t-phishing 2, t-firewall 2, t-wifi 2 |
| TC-01-50 | AC10 | – | **C** `tc_01_10` (SV, TA, GV × 12 thread) | Không còn "Thảo luận của lớp" / "Phản hồi trong thread này"; mỗi thread có "Thảo luận (n)" đúng n của TC-01-49 |
| TC-01-51 | AC10 | B | **B** + **C** `tc_01_11` `/threads/t-cbc` | Từ trên xuống: câu hỏi gốc (Đặng Gia An, thời gian, chủ đề Mật mã đối xứng, tuần 10, "4 người tham gia") → khối AI `Chờ xác nhận` (S1) + `Nguồn tham khảo (2)` (Chương 3 tr. 14–17; Modern Network Security Threats mục 2.4) → **một** vùng "Thảo luận (4)" với 4 bài đúng thứ tự thời gian: Dương Thanh Hiếu ("Vậy IV có cần giữ bí mật không ạ?…"), Phạm Quốc Bảo (có khối trích bài của Hiếu), Đặng Gia An ("Em hiểu rồi: IV cố định…"), Nguyễn Thị Giang ("Còn CTR thì sao ạ…") → ô "Phản hồi của bạn" ở cuối chính vùng đó |
| TC-01-52 | AC10 | B | **B** `t-cbc`: các nút của SV | `Gửi phản hồi` là nút chính (đỏ); `Hỏi trợ lý AI` phụ; SV **không** có `Xác nhận`, `Chỉnh sửa`, `Loại khỏi tri thức`; một nút chính mỗi vùng |
| TC-01-53 | AC10 / 4.3.1 B | B | **B** `t-salt` | Hỏi: Sinh viên B (31 giờ); AI "Đã được giảng viên xác nhận · Lê Thu Hà", nguồn Chương 4 tr. 8; "Thảo luận (3)": Vũ Minh Dũng ("Muối lưu ở đâu ạ?…"), GV (trích bài của Dũng, kết bằng "vì sao bcrypt còn cố tình chậm?"), B ("Em nghĩ là để mỗi lần đoán thử…") — bài của B đánh dấu là của mình |
| TC-01-54 | AC10 | B | **B** `t-sqli`, `t-xss`, `t-vpn`, `t-pki`, `t-phishing`, `t-rsa-key` | Mỗi thread đủ bài theo 4.3.1 B (người, thời gian tương đối, bài có trích); nhãn AI: `t-rsa-key` `Chờ xác nhận` (K1), còn lại `Đã được giảng viên xác nhận` (W1, W2, V1, P1, L1); nguồn đúng bảng C (ví dụ `t-sqli` Chương 2 tr. 18–22) |
| TC-01-55 | AC10 | B | **B** `t-pin-rubric`, `t-pin-lab` | Thread ghim (Thông báo / Thực hành, GV đăng): 4 và 3 phản hồi; bài của TA / GV có khối trích đúng bài được trả lời (rubric: sv-12 → TA, sv-20 → GV) |
| TC-01-56 | AC10 | B | **B** `t-firewall`, `t-wifi` | Không có khối AI; danh sách ghi "Chưa có câu trả lời"; chi tiết vẫn "Thảo luận (2)" với 2 bài |
| TC-01-57 | AC10 / 4.3.1 A | B | **B** Hàng danh sách: "hoạt động gần nhất" | Bằng bài mới nhất của thread (t-cbc 31 phút trước); số phản hồi **tính từ dữ liệu** — mở thread, đếm bài trong vùng thảo luận = số ở danh sách = n ở tiêu đề (mọi 12 thread) |
| TC-01-58 | AC10 / 4.3.1 A | B ở `t-cbc` | **B** `Gửi phản hồi` một câu; sau đó `Hỏi trợ lý AI` với chữ; xem lại `/threads` | Mỗi phản hồi thêm +1 vào "Thảo luận (n)" và danh sách (câu AI sinh từ `Hỏi trợ lý AI` cũng được tính, **không** tính câu AI chính); n ở tiêu đề luôn bằng số ở danh sách |
| TC-01-59 | AC10 | TA / GV | **B** `/threads` và `/threads/t-cbc` | Cùng số phản hồi và cùng bài như SV; thêm `Xác nhận` / `Chỉnh sửa` / `Loại khỏi tri thức` ở khối AI |
| TC-01-60 | AC11 (#17e) | B, 1440 × 900, phiên trống | **A** "01-AC11" | Lịch sử phiên là panel (viền, nền, bo góc) rộng 240 px; hội thoại và composer cùng `left` và cùng `width`; composer cách đáy màn ≤ 24 px: `[true, true, true]` |
| TC-01-61 | AC11 | B, mở phiên "Cách chọn độ dài khoá RSA" (6 tin) hoặc sau D1–D3 | **A** | Như TC-01-60; hội thoại cuộn **bên trong** (trang không có thanh cuộn dọc: `scrollHeight` ≤ `innerHeight` + 1) |
| TC-01-62 | AC11 (biên) | – | **A** 1440 × 600; mở phiên cũ rồi `Phiên mới` | Composer vẫn ghim đáy (≤ 24 px) và cùng mép / rộng với hội thoại dù màn thấp, dù ít hay nhiều tin |
| TC-01-63 | AC11 | B, 375 × 812 | **A** + **B** `/chat` | Lịch sử ẩn (không nhìn thấy `chat-history`); composer vẫn ghim; không cuộn ngang |
| TC-01-64 | AC12 (#18) | B, `t-cbc` ô soạn trống | **T** `T10` | `Hỏi trợ lý AI` → AI đăng `Gợi ý thêm` S1 ("…bao nhiêu khối bản rõ bị ảnh hưởng?") theo trình tự E (soạn → chảy → Dừng → nguồn); hai nút khoá trong lúc AI soạn |
| TC-01-65 | AC12 | B, `t-cbc` | **T** `T11`: gõ "Còn CTR thì sao, có cần IV ngẫu nhiên không?" → `Hỏi trợ lý AI` | Phản hồi của B hiện **trước**, AI trả lời ngay dưới theo S2 với dòng "↳ trả lời Trần Thu Uyên"; ô soạn được xoá sau khi gửi |
| TC-01-66 | AC12 | B, `t-cbc` | **T** `T12`: gửi "@AI dùng lại nonce có sao không" bằng `Gửi phản hồi` | Xử lý như bấm `Hỏi trợ lý AI`: phản hồi của B, AI S2 dưới, "↳ trả lời Trần Thu Uyên" |
| TC-01-67 | AC12 (nhánh lỗi) | B, đang chảy | **T** `T3` trên khối `Hỏi trợ lý AI` (hoặc **B** bấm giờ) | `Dừng` giữ phần đã hiện + "Đã dừng" + `Hỏi lại`; sau `Dừng` hai nút mở lại |
| TC-01-68 | AC12 / F (biên) | B, thread chưa có câu AI: `t-wifi` rồi `t-firewall` | **B** `Hỏi trợ lý AI` với ô trống | AI trả lời **câu hỏi gốc** theo D; chủ đề không có mẫu và không từ khoá → nhánh không khớp (toàn văn ngay, không nguồn, `Đang chờ giảng viên`); luôn có phản hồi hiển thị, không bao giờ "bấm mà không có gì" |
| TC-01-69 | AC12 / F, AC14 | B | **B** Gõ chữ có `20229002` rồi `Hỏi trợ lý AI`; lặp với `Gửi phản hồi` | Quét PII: Dialog hai lối xuất hiện, chưa đăng gì cho tới khi chọn lối |
| TC-01-70 | AC13 (#18) | B, `t-salt`, sau `Đặt lại dữ liệu` | **T** `T5`: gửi "Em hiểu rồi ạ" | Phản hồi hiện ngay (≤ 0,8 s) ở cuối vùng thảo luận và trang cuộn tới (trong khung nhìn); ~2 s (1,5–2,8) sau: "Phạm Quốc Bảo đang trả lời…" |
| TC-01-71 | AC13 | Sau TC-01-70 | **T** `T5c` | ~6 s (5,4–7,0): phản hồi của TA đúng văn bản H1 "Em xem bảng so sánh SHA-256 với bcrypt ở trang 9 rồi trả lời câu AI hỏi nhé.", có khối trích phản hồi của B |
| TC-01-72 | AC13 | Sau TC-01-71 | **T** `T5d`, `T5e`, `T5f` | Chuông có chấm chưa đọc + "Phạm Quốc Bảo đã trả lời trong «Vì sao cần muối (salt) khi băm mật khẩu?»"; bấm → `/threads/t-salt` và cuộn tới đúng phản hồi; `/threads` ghi `t-salt` **5** phản hồi |
| TC-01-73 | AC13 (nhánh) | B, `t-salt` | **T** `T6`: gửi rồi sang `/calendar` trước 6 s | Tới hạn chuông vẫn có thông báo; quay lại thread thấy phản hồi TA (không phụ thuộc bộ hẹn giờ của trang) |
| TC-01-74 | AC13 (nhánh) | B, `t-salt` | **T** `T7`: gửi, tải lại ở giây 3, chờ | Phản hồi của B còn sau tải lại; phản hồi TA vẫn hiện ≤ ~7,5 s kể từ lúc gửi; chuông có thông báo |
| TC-01-75 | AC13 | Sau TC-01-71 | **T** `T8`: gửi phản hồi thứ hai trong cùng thread | Không còn phản hồi trễ (số khối "Phạm Quốc Bảo" không tăng sau 8 s) |
| TC-01-76 | AC13 / G (biên) | B, `t-firewall` (không có mẫu) | **T** `T9` | ~6 s: TA trả lời "Cảm ơn em, anh đã ghi nhận. Thầy cô sẽ trả lời chi tiết trong buổi học tới; em xem trước tài liệu tuần <tuần của thread> nhé." với đúng tuần hiển thị ở thread |
| TC-01-77 | AC13 / G | B | **B** Gửi phản hồi ở `t-salt` rồi ở `t-sqli` (hai thread khác nhau) | Mỗi thread một lần: cả hai đều có phản hồi trễ (nội dung theo mẫu chủ đề: H1 và W1 "Em thử với bảng `products` trong lab 2…"); chuông có hai thông báo |
| TC-01-78 | AC13 / G | Sau TC-01-77 | **B** `Đặt lại dữ liệu demo`, gửi lại ở `t-salt` | "Mỗi thread một lần mỗi phiên": sau Đặt lại lại có phản hồi trễ; thông báo cũ biến mất |
| TC-01-79 | AC14 (nhánh lỗi, #18) | B | **T** `T4`: tạo "WPA3 chặn được KRACK không?", chủ đề An toàn mạng không dây, nhờ AI bật | Toàn văn "Mình chưa đủ chắc chắn để gợi ý câu này từ tài liệu của lớp. Mình đã báo giảng viên và trợ giảng; câu trả lời sẽ hiện ngay trong thread này." hiện ngay, không "đang soạn", không chữ chảy, không `Nguồn tham khảo`; nhãn `Đang chờ giảng viên` (hổ phách); **không** có `Chờ xác nhận` |
| TC-01-80 | AC14 / D4 | B | **B** Đăng "Ivy league là gì?" (Mật mã đối xứng) và một câu không liên quan | Cùng nhánh không khớp như TC-01-79 (toàn văn, `Đang chờ giảng viên`) |
| TC-01-81 | AC14 / Q4 | Sau TC-01-79 | **T** `T4d`: đổi vai GV → `/inbox` | Số ticket **không đổi** ("Tất cả 6"), sidebar "Hộp thư hỗ trợ" vẫn 5; chat riêng không bị tạo ticket |
| TC-01-82 | AC14 | B, ô soạn trống | **T** `T13b` | `Gửi phản hồi` khoá khi ô trống và khi chỉ có khoảng trắng; mở khoá khi gõ chữ |
| TC-01-83 | AC14 | B, `t-cbc` | **T** `T13`: gõ dở "Em đang gõ dở" → `/threads` → quay lại; sang `t-salt`; tải lại | Chữ còn nguyên khi quay lại; `t-salt` ô trống (nháp theo thread, không lẫn); tải lại vẫn còn; sau khi gửi thì nháp bị xoá |
| TC-01-84 | AC14 / AC4 | B, `t-cbc` | **B** Gõ phản hồi chứa `20229002` → `Gửi phản hồi` | Dialog đúng hai lối; chưa có phản hồi mới trong thread khi chưa chọn |
| TC-01-85 | AC14 | Sau TC-01-84 | **B** Lối 1 `Chuyển sang chat riêng` | Sang `/chat` mang nguyên chữ phản hồi, thread không thêm bài |
| TC-01-86 | AC14 | Sau TC-01-84 | **B** Lối 2 `Ẩn thông tin rồi đăng` | Bài đăng có `[đã ẩn]` thay MSSV; số phản hồi +1; không còn `20229002` ở bất kỳ chỗ nào của thread |
| TC-01-87 | AC5 / AC10–14 | – | **B** + **C** `tc_01_05`; quét `innerText` các thread sau mọi bước Threads (AI soạn, Dừng, không khớp, phản hồi trễ, Dialog) | Không từ kỹ thuật AI (RAG, PII, provider, trace, confidence…), không số độ tin cậy; nhãn AI chỉ là `Chờ xác nhận`, `Đang chờ giảng viên`, `Đã được giảng viên xác nhận` |

## Công cụ bổ sung (spec v5)
**A** = `scripts/audit.mjs`; **T** = `scripts/threads-timeline.mjs` (mã bước `T1`…`T14`, chạy khi dev báo xong Threads; dung sai ở hằng `TOL` đầu file); **C** thêm `tc_01_10`, `tc_01_11`. Thời gian bấm tay cho phép sai ±0,5 s; script lấy mẫu ~40 ms.

## Nhánh lỗi
TC-01-21, 22, 24 (AC8); TC-01-17 (số ngoài 0–10); TC-01-33 (`?state=`). Spec v5: TC-01-44 (đăng thiếu trường), 46, 48 (khớp biên / hoà điểm), 67 (Dừng), 68 (thread không có mẫu), 73, 74 (phản hồi trễ qua điều hướng / tải lại), 76, 79–83 (nhánh không khớp, nút khoá, nháp), 84–86 (PII ở phản hồi).

## Phân quyền
TC-01-25, 26 (AC9, ma trận SRS 2); TC-01-13 (SV không thấy nhãn rủi ro / điểm nháp của chính mình); TC-01-52 (SV không có `Xác nhận` / `Chỉnh sửa` / `Loại`); TC-01-59 (TA/GV có).

## Kiểm chéo
- Lời văn SV: TC-01-12, 13 (DESIGN §13). Phản mẫu §21: ảnh từng route (sweep). §22 10 điều kiện: đánh dấu từng route trong report.
- 375 px: TC-01-14, 15. Không PII: MSSV chỉ xuất hiện ở chữ SV tự gõ (D1) và bị thay trước khi "gửi".
- Không `fetch`: `tc_00_static`.

## Điểm khó kiểm
- "Chữ chảy" cần quan sát thời gian (~3 s) trong trình duyệt; curl không thấy.
- AC7 con số 8,8 mâu thuẫn rubric (proposals #13): chấm theo spec hiện hành, ghi rõ trong report nếu dev chọn khác.
- Bước Mailpit bỏ (FR-X7): AC3 chỉ kiểm dòng xác nhận mô phỏng.
- Threads v5: dung sai thời gian là suy ra từ cột "Thời lượng" 4.3.1 E và G (1,2 s; 2,5–3,5 s; 0,3 s; 2 s; 6 s) cộng sai số đo ±0,4 s; nếu dev đặt số khác hẳn thì ghi nhận chênh lệch, không tự nới `TOL`.
- Spec chưa nói rõ (ghi nhận, không đoán): bấm `Hỏi trợ lý AI` lần thứ hai khi đã dùng `Gợi ý thêm` (TC-01-64 / T10d chỉ đòi "có phản hồi nhìn thấy", không đòi nội dung); `Hỏi trợ lý AI` có chữ có tính là "phản hồi đầu tiên" để kích hoạt phản hồi trễ hay không (TC-01-65 không kiểm); thứ tự của việc "Câu hỏi mới" trong Hôm nay của GV (TC-02-43 chỉ kiểm có mặt và đếm).
- Các TC dựa vào văn bản mẫu S1/S2/H1/W1… so khớp nguyên văn SRS 4.3.1 C; sai khác dấu câu là FAIL nhỏ, ghi rõ dòng nào lệch.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-01 · TC-01-17, 19: Cập nhật QT sau công bố = 8,7; What-if CK 8,0 = 8,3 theo spec v3 (proposals #13).
- 2026-10-01 · TC-01-09..12, 34..37: Cập nhật form tạo thread đầy đủ + chuyển hướng sang `/threads/${id}` (01-AC4); thêm Reply Composer, nút Hỏi trợ lý AI, quét PII phản hồi (01-AC10) theo spec v4 (proposals #15, #16).
- 2026-10-02 · Thêm TC-01-39…87 (không sửa TC cũ): spec v5 — 01-AC4 (trình tự E, mẫu S1/S2, quy tắc D), 01-AC10 (seed 12 thread, "Thảo luận (n)"), 01-AC11 (#17e, chat), 01-AC12 (Hỏi trợ lý AI), 01-AC13 (phản hồi trễ + chuông), 01-AC14 (nhánh không khớp, nháp, PII) — #17e, #18. Ghi chú: TC-01-36 ("Hỏi trợ lý AI sinh câu gợi ý") được TC-01-64…68 thay nghĩa chi tiết; giữ nguyên chữ TC-01-36, kết quả đúng là kết quả của TC-01-64.
