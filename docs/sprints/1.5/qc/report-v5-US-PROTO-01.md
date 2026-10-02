# QC report v5 — US-PROTO-01 (Sinh viên)  · Kết luận: **FAIL**

Nhánh `sprint/1.5-mock-ui` · HEAD `8bb6861` · ngày 2026-10-01 · server `http://localhost:3400` · Chrome for Testing headless riêng (profile `/tmp/qcv5-*`)
TC: `docs/sprints/1.5/qc/tc-US-PROTO-01.md` — **153 TC**. Đối chiếu spec v5 + v5.1 + #20–#24.

**Tóm tắt:** 147/153 TC PASS; **6 FAIL** — TC-01-04, TC-01-15, TC-01-18, TC-01-33, TC-01-143, TC-01-153.

## Cổng đã chạy

| Lệnh / script | Kết quả |
| --- | --- |
| `proto-curl.sh` (`evidence-v5/proto-curl.log`) | 497 PASS / 0 FAIL toàn bộ; phần US-01: `tc_01_01`, `_05`, `_09`, `_10`, `_17`, `_24` PASS |
| `audit.mjs` (`audit.json`, `audit-wide.json`) | 677 + 412 dòng; vai SV: 1440 → 0 FAIL; 390/375 → FAIL duy nhất `/threads` (`wide`=382/396) — đã xác nhận bằng tay là lỗi thật |
| `threads-timeline.mjs` (`timeline.json`, T1…T16) | tất cả PASS sau sửa công cụ |
| `pii-matrix.mjs` (`pii.json`) | 48/48 PASS |
| `regress-v24.mjs` (`v24.json`) | TC-01-153 FAIL (7/8 ca) |
| `pnpm -C frontend lint` | rc=0 |
| `bash scripts/ui-antipatterns.sh` | rc=0 (mọi dòng ✓) |
| Chạy tay (Chrome riêng, chuột thật) | các TC loại **B**, đo lại các dòng machine FAIL; ảnh `docs/sprints/1.5/shots/qc-v5/owner/`, `v5q01a`, `v5q01b`, `pii`, `threads` |

## TC

| TC-id | PASS/FAIL | Cách kiểm | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-01-01 | PASS | C | proto-curl.log tc_01_01 (12 dòng) PASS |
| TC-01-02 | PASS | B | audit student 1440: 112 dòng/12 route, 0 FAIL (không tràn, không cắt); h1/nút chính theo audit + owner shots |
| TC-01-03 | PASS | B | B: "QUIZ01 Mật mã đối xứng đóng sau 18 giờ · Bạn chưa làm bài này · Khoảng 20 phút" + dòng thời gian Buổi 10 09:00–11:30 P.302 đang diễn ra + QUIZ01 03:20 30/10; A: "Ôn lại Mật mã đối xứng · Bạn sai 4/7 câu gần nhất · Khoảng 15 phút"; D: ô nhập mã |
| TC-01-04 | FAIL | B | D1: dòng "Đã ẩn 2 thông tin…", chảy dần, 10/09 + 08/10 + 0,75 đúng, Hữu ích/Không hữu ích/Nhờ giảng viên, không [[SV_ — PASS; nhưng "Nguồn tham khảo (1)" thu gọn (aria-expanded=false), chỉ "Quy chế môn học An ninh mạng – 761987 trang 2 · mục Điểm quá trình"; SRS 4.3.2/§14.2 + TC: "(2) mở ngay dưới (Quy chế tr. 2; Sổ điểm danh lớ |
| TC-01-05 | PASS | B | Bạn Lê Quang Huy nghỉ mấy buổi rồi ạ? Đã ẩn 1 thông tin cá nhân trước khi gửi cho AI Tìm hiểu Trợ lý AI Mình chỉ trả lời được thông tin của chính bạn. Nếu cần trao đổi về bạn khác, hãy hỏi giảng viên. |
| TC-01-06 | PASS | B | Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI Tìm hiểu Trợ lý AI Bạn đã vắng 2 buổi (10/09, 08/10) và được cộng 0,75 điểm cho 3 lần phát biểu. Vắng thêm 1 buổi không phép sẽ bị trừ 0,5 điểm. Buổi vắng 2 buổi Phát biểu 3 lần · +0,75 Nguồn tham khảo (1) Hữu ích Không hữu ích Nhờ giảng viên hỗ trợ Gửi |
| TC-01-07 | PASS | B | Thi cuối kỳ có được mang một tờ A4 ghi chú viết tay vào phòng thi không ạ? Trợ lý AI AI chưa đủ chắc chắn về câu này. Quy chế môn học không nói rõ về tài liệu được mang vào phòng thi, nên mình đã chuyển câu hỏi cho giảng |
| TC-01-08 | PASS | B | D4 hiển thị=true; nhãn GV gần D4=true; sau Đã rõ giữ D4=false (BUG-DEMO-4 nếu false) |
| TC-01-09 | PASS | B | checkbox: Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng |
| TC-01-10 | PASS | B | nút=["Chuyển sang chat riêng","Ẩn thông tin rồi đăng"] threads=12 |
| TC-01-11 | PASS | B | /chat; composer="Hỏi về mã số 20229002  Em là 20229002 muốn hỏi về CBC" |
| TC-01-12 | PASS | B | /threads/t-new-1; [đã ẩn]=true; MSSV còn=false; Chờ xác nhận sau 5 s=true; ngay lúc đầu: Threads Hỏi về mã số [đã ẩn] Trần Thu Uyên (bạn) Mật mã đối xứng · Tuần 10 · vừa |
| TC-01-13 | PASS | C | proto-curl.log tc_01_05 (18 dòng) PASS |
| TC-01-14 | PASS | B | quét 13 route + chat D1 + thread sau Hỏi AI: vi phạm=[] |
| TC-01-15 | FAIL | B | 375/390: /threads (SV, GV, TA) viewport bị nới tới 772 px (innerWidth 772, body.scrollWidth 771; hàng chip chủ đề data-scroll-x scrollWidth 1639 không bị bó) → ô tìm, tiêu đề thread, chip bị cắt phải; ảnh owner/threads-390-mobile.webp. Các route SV khác 375/390: 0 FAIL, TOUCH đạt |
| TC-01-16 | PASS | B | chế độ chọn=["Danh sách"] ; lưới tuần=false |
| TC-01-17 | PASS | B | /me B: QT 8,3 (tạm tính); "7,50 + 0,75 − 0,00 = 8,25 → làm tròn 0,1 → 8,3"; vắng, QUIZ01 "Đóng 30/10 · còn 18 giờ 20 phút", "Thi giữa kỳ Thứ Năm, 5 tháng 11", dòng "hệ thống quản lý đào tạo"; không nhãn rủi ro/ghi chú/điểm nháp |
| TC-01-18 | FAIL | B | 8,0 → "điểm học phần sẽ là 8,1 · 40% × 8,3 + 60% × 8,0" (đúng số học; TC ghi 8,3 chỉ đúng khi QT 8,7 — lỗi TC, đề nghị sửa); 11/−1/abc → lỗi tại ô (aria-invalid) OK; **ô trống vẫn hiện lỗi "Nhập một số từ 0 đến 10" + aria-invalid=true** (TC: trống → không lỗi giả); ô khởi tạo sẵn "8,0" |
| TC-01-19 | PASS | B | GV Lưu điểm danh buổi 10 (có Phát biểu cho B) → B /me "Điểm quá trình hiện tại 8,5" (xem TC-01-144) |
| TC-01-20 | PASS | B | chỉnh BT03 (8,50 điểm rubric − 0,5 nộp muộn 8,0 Điểm và nh) + công bố: Điểm quá trình hiện tại 8,7 |
| TC-01-21 | PASS | B | A đổi sang lớp 761988 → /me: "Lớp này chưa có công thức điểm chính thức" (xem TC-01-105) |
| TC-01-22 | PASS | B | mã sai chữ/hết hạn → đúng câu "Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên."; mã rỗng: nút "Xem lớp" khoá (không có câu lỗi riêng) — ghi chú TC |
| TC-01-23 | PASS | B | lần sai thứ 5 → ô khoá, "Thử lại sau 10 phút / Bạn đã nhập sai 5 lần"; còn lại sau reload giữ khoá |
| TC-01-24 | PASS | B | Nhập mã khác Tham gia lớp Kiểm tra thông tin lớp trước khi gửi yêu cầu. An ninh mạng Học phần An ninh mạng (INT1006) Mã lớp 761988 Giảng viên TS. Lê Thu Hà Học kỳ HK1 2026–2027 Lịch học Thứ Ba 07:00–0 ‖ Nhập mã khác Tham gia lớp Kiểm tra thông tin lớp trước khi gửi yêu cầu. An ninh mạng Học phần An ninh mạng (INT1006) Mã  ‖ http |
| TC-01-25 | PASS | B | value=Chữ dở dang của em · Chat riêng Hỏi về điểm, chuyên cần và bài học của chính bạn — An ninh mạng – 761987. Giảng viên không đọc được phiên cha |
| TC-01-26 | PASS | C | proto-curl.log tc_01_09 (20 dòng) PASS |
| TC-01-27 | PASS | C | proto-curl.log tc_00_matrix (128 dòng) PASS |
| TC-01-28 | PASS | B | câu đầu: Thoát về Luyện đề Luyện chủ đề: Mật mã đối xứng Câu 1/10 Đúng 0 Mật mã đối xứng Vì sao ảnh; sau chọn: Đúng |
| TC-01-29 | PASS | B | xác nhận nộp: true |
| TC-01-30 | PASS | B | zzzz: Thư viện Bài giảng, quy chế và đề cũ của lớp. Tìm theo tên, chủ đề hoặc tuần học. 10 tài liệu Tất cả 10 Bài giảng 6 Quy chế 2 Đề cũ 2 Không có tài liệu nào khớp ‖ http://localhost:3400/chat · composer="Về tài liệu “Chương 3 — Mật mã đối xứng và chế độ vận hành”: " |
| TC-01-31 | PASS | B | Báo cáo → dialog lý do; chọn "Sai kiến thức" → "Đã gửi báo cáo" (không có nút Gửi riêng) |
| TC-01-32 | PASS | B | Kết quả của tôi Bài tập 03 — Phân tích một vụ tấn công thực tế Chọn một vụ tấn công đã công bố, phân tích theo 4 tiêu chí của rubric. Hạn Thứ Năm, 22 tháng 10 23:59 Cho nộp muộn tối đa 2 ngày, trừ 0,5 điểm mỗi ngày Bài nộp của bạn bt03-tran-thu-uyen.pdf 418 KB ‖ Kết quả của tôi Bài tập Không tìm thấy bài tập này Bài có thể đã bị |
| TC-01-33 | FAIL | B | điểm 8,0=true; 4 tiêu chí=[true,true,false,false]; trích đoạn thuộc bài B (SolarWinds/Orion)=false; trích: ["“Nhóm tấn công vào hệ thống bán lẻ qua tài khoản VPN của một nhà thầu bảo trì, không bật xác thực đa yếu tố.”","“Sau khi vào mạng, kẻ tấn công dùng công cụ quản trị có sẵn để di chuyển ngang sang máy chủ quản lý bản vá.”" |
| TC-01-34 | PASS | B | [["/",true,true],["/chat",true,true],["/threads",true,true],["/practice",true,true],["/library",true,true],["/calendar",true,true],["/me",true,true]] |
| TC-01-35 | PASS | B | {"td":{"q":true,"ai":true,"src":true,"rep":true,"n":4},"order":[8,282,1384]} |
| TC-01-36 | PASS | B | xem bảng owner (đã chạy tay ở owner walkthrough) |
| TC-01-37 | PASS | B | ["Chuyển sang chat riêng","Ẩn thông tin rồi đăng"] |
| TC-01-38 | PASS | B | bài 4→5; hiện cuối danh sách=true |
| TC-01-39 | PASS | T | timeline.json T1a+T1b+T1c+T1d+T1e+T1f+T1g+T1h PASS |
| TC-01-40 | PASS | T | timeline.json T1i PASS |
| TC-01-41 | PASS | T | timeline.json T2a+T2b PASS |
| TC-01-42 | PASS | T | timeline.json T3a+T3b+T3c PASS |
| TC-01-43 | PASS | T | timeline.json T14 PASS |
| TC-01-44 | PASS | B | http://localhost:3400/threads/t-new-1 · AI block=false ‖ {"d":true,"h":"Còn thiếu: tiêu đề"} |
| TC-01-45 | PASS | B | khối độc lập, CBC trộn khối bản mã trước vào khối hiện tại rồi mới mã hoá. Với ECB, hai khối giống nhau cho ra hai khối bản mã thế nào — và điều đó làm lộ gì tr ‖ khối độc lập, CBC trộn khối bản mã trước vào khối hiện tại rồi mới mã hoá. Với ECB, hai khối giống nhau cho ra hai khối bản mã thế nào — và điều đó làm lộ gì tr |
| TC-01-46 | PASS | B | ã báo giảng viên và trợ giảng; câu trả lời sẽ hiện ngay trong thread này. Thảo luận (0) Phản hồi của bạn Nhập nội dung phản hồi Hỏi trợ lý AI Gửi phản hồi |
| TC-01-47 | PASS | B | an trọng hơn bí mật. Ở CTR, bộ đếm sinh ra dòng khoá rồi XOR với bản rõ. Nếu dùng lại cùng nonce với cùng khoá cho hai thông điệp, XOR hai bản mã với nhau sẽ cho ra gì? Nguồn tham khảo (1) Chương 3 —  |
| TC-01-48 | PASS | B | Hàm băm:  mà không có muối, giá trị băm của họ có gì đặc biệt, và bảng tra sẵn tận dụng điều đó ra sao? Nguồn tham khảo (1) Chươn \| MMĐX S1=true \| Thực hành S1=true |
| TC-01-49 | PASS | B | {"t-cbc":4,"t-salt":3,"t-sqli":3,"t-pin-rubric":4,"t-pin-lab":3,"t-rsa-key":2,"t-xss":2,"t-vpn":2,"t-pki":2,"t-phishing":2,"t-firewall":2,"t-wifi":2} |
| TC-01-50 | PASS | C | proto-curl.log tc_01_10 (72 dòng) PASS |
| TC-01-51 | PASS | B |  hoặc đi đến… ⌘ K U Trần Thu Uyên Threads CBC khác ECB ở điểm nào? Đặng Gia An Mật mã đối xứng · Tuần 10 · 1 giờ trước · 4 người tham gia Em đọc slide chương 3 thấy AES có nhiều chế độ. Em vẫn chưa hiểu vì sao ảnh mã hoá bằng ECB lại còn nhìn ra hình, còn CBC thì không. |
| TC-01-52 | PASS | B | Báo cáo \| Nguồn tham khảo (2) \| Trả lời \| Báo cáo \| Trả lời \| Báo cáo \| Trả lời \| Báo cáo \| Trả lời \| Báo cáo \| Hỏi trợ lý AI \| Gửi phản hồi \| Quay lại sửa \| Ẩn thông tin rồi đăng |
| TC-01-53 | PASS | B | Chương 4 — Hàm băm… trang 8; Dũng, GV (bcrypt còn cố tình chậm), B (bạn) đủ |
| TC-01-54 | PASS | B | [["t-sqli",true,false],["t-xss",true,false],["t-vpn",true,false],["t-pki",true,false],["t-phishing",true,false],["t-rsa-key",true,false]] |
| TC-01-55 | PASS | B | 4/3 |
| TC-01-56 | PASS | B | AI block: false ‖  |
| TC-01-57 | PASS | B | 12 thread: số phản hồi ở danh sách = số bài trong thread = n ở "Thảo luận (n)" (t-cbc 6 sau khi B đăng + TA trả lời muộn; các thread khác 2–4 khớp) |
| TC-01-58 | PASS | B | n 4→5; danh sách 5 |
| TC-01-59 | PASS | B | Nguồn tham khảo (2) \| Xác nhận \| Chỉnh sửa \| Trả lời \| Trả lời \| Trả lời |
| TC-01-60 | PASS | A | audit 01-AC11 phiên trống (1 dòng) → 0 FAIL |
| TC-01-61 | PASS | A | audit 01-AC11 phiên ≥ 6 (1 dòng) → 0 FAIL |
| TC-01-62 | PASS | A | audit 01-AC11 biên (1 dòng) → 0 FAIL |
| TC-01-63 | PASS | A | audit 01-AC11 375 (1 dòng) → 0 FAIL |
| TC-01-64 | PASS | T | timeline.json T10a+T10b+T10c+T10d PASS |
| TC-01-65 | PASS | T | timeline.json T11 PASS |
| TC-01-66 | PASS | T | timeline.json T12 PASS |
| TC-01-67 | PASS | T | timeline.json T3a+T3b+T3c PASS |
| TC-01-68 | PASS | B | t-wifi: Đang chờ giảng viên=true; t-firewall: true; (nhánh không khớp, không nguồn) |
| TC-01-69 | PASS | B | Hỏi AI có PII → hộp thoại ["Chuyển sang chat riêng","Ẩn thông tin rồi đăng"]; số bài vẫn 4; gửi phản hồi: ["Chuyển sang chat riêng","Ẩn thông tin rồi đăng"] |
| TC-01-70 | PASS | T | timeline.json T5a+T5b PASS |
| TC-01-71 | PASS | T | timeline.json T5c PASS |
| TC-01-72 | PASS | T | timeline.json T5d+T5e+T5f PASS |
| TC-01-73 | PASS | T | timeline.json T6a+T6b PASS |
| TC-01-74 | PASS | T | timeline.json T7a+T7b+T7c PASS |
| TC-01-75 | PASS | T | timeline.json T8 PASS |
| TC-01-76 | PASS | T | timeline.json T9 PASS |
| TC-01-77 | PASS | B | PQB trong t-salt=1 (H1 true); t-sqli=2 (W1 true) |
| TC-01-78 | PASS | B | H1 trước=0 sau Đặt lại + gửi=1 |
| TC-01-79 | PASS | T | timeline.json T4a+T4b+T4c PASS |
| TC-01-80 | PASS | B | Ivy league: true; không liên quan: true |
| TC-01-81 | PASS | T | timeline.json T4d PASS |
| TC-01-82 | PASS | T | timeline.json T13b PASS |
| TC-01-83 | PASS | T | timeline.json T13+T13b PASS |
| TC-01-84 | PASS | B | {"t":"Bài này có thông tin cá nhân Threads là nơi cả lớp cùng đọc. Chúng tôi tìm thấy: 1 mã số sinh viên. Chuyển sang chat riêng Ẩn thông tin rồi đăng","b":["Đóng","Chuyển sang chat riêng","Ẩn thông tin rồi đăng"]} n 3→3 |
| TC-01-85 | PASS | B | http://localhost:3400/chat · composer="MSSV của em là 20229002 ạ" n=3 |
| TC-01-86 | PASS | B | Chạy sạch: n +1, "[đã ẩn]" thay MSSV, 0 "20229002" trong body/html/ep_demo_state (lần đầu FAIL giả do bản nháp lối 1 nằm ở chat) |
| TC-01-87 | PASS | B | cùng quét; nhãn AI chỉ Chờ xác nhận / Đã được giảng viên xác nhận |
| TC-01-88 | PASS | B | đúng 7/10; "Câu cần ôn (3)" 8,9,10; dòng phụ "Mật mã đối xứng · 10 câu · vừa xong"; 0 pageerror |
| TC-01-89 | PASS | B | header=Câu 1/3 ‖ Luyện đề Bạn đúng 0/3 câu Mật mã đối xứng · 3 câu · vừa xong Câu cần ôn (3) Câu 1 Sai Mã hoá đối xứng và mã hoá khoá côn \|\| Luyện đề Lịch sử luyện tập 8 lượt gần đây. Bấm một lượt để xem lại từng câu và giải thích. Các lượt đã làm Mật mã đối xứng · 3 câu 0/3 vừa xong · Nên luyện lại chủ đề này Mật mã đối xứng  |
| TC-01-90 | PASS | B | 11/18→14/28 (sai +3, tổng +10), lịch sử 7/10 hàng đầu, khối dở biến mất ‖ trước: Mật mã đối xứng Bạn sai 11/18 câu ở các lượt gần nhất 10 câu · khoảng 15  → sau: Mật mã đối xứng Bạn sai 14/28 câu ở các lượt gần nhất 10 câu · khoảng 15  · khối dở: false |
| TC-01-91 | PASS | B | Câu 4/10 · Đúng 3 · state 284B |
| TC-01-92 | PASS | B | khối dở: Luyện đề Luyện theo chủ đề bạn hay sai, hoặc thi thử với đề cũ. Lịch sử luyện tập Nên luyện ngay Ôn lại Mật mã đối xứng Bạn sai 11/18 câu ở các lượt gần nhất 10 câu · khoảng 15 phút · có giải thích ngay sau mỗi câu Luyện 10 câu Chọn cách luyện Theo chủ đề 4 Thi thử 3 Mật mã đối xứng Bạn sai 11/18 câ → Câu 6/10 ‖ Tiếp tụ |
| TC-01-93 | PASS | B | [["Em chưa rõ",false,"sai","9"],["Các khối giống nhau cho bản mã giống nhau",true,"đúng?","10"],["CÁC KHỐI GIỐNG NHAU",true,"đúng?","10"],["logic",false,"sai","9"],["  ecb lộ mẫu  ",true,"đúng?","10"]] · rỗng: Kiểm tra khoá + "Nhập câu trả lời để kiểm tra" |
| TC-01-94 | PASS | B | Luyện đề Bạn đúng cả 10 câu Mật mã đối xứng · 10 câu · vừa xong Luyện chủ đề khác Xem lịch sử luyện tập \| Luyện đề,Luyện chủ đề khác,Xem lịch sử luyện tập |
| TC-01-95 | PASS | B | Luyện đề Bạn đúng cả 10 câu Mật mã đối xứng · 10 câu · vừa xong Luyện chủ đề khác Xem lịch sử luyện  \|\| Luyện đề Lịch sử luyện tập 7 lượt gần đây. Bấm một lượt để xem lại từng câu và giải thích. Các lượt đã làm Mật mã đối xứng · 10 câu 10/10 vừa xong · Nắm khá chắ errs=0 |
| TC-01-96 | PASS | B | Quét câu hỏi QUIZ01 (8 câu), luyện đề at-symmetric (10), hash, /questions, đề cũ: không cặp trùng, câu 1 luyện đề ("Vì sao ảnh mã hoá bằng chế độ ECB…") không còn ở QUIZ01 (làm tay ở lượt 1, script KEY) |
| TC-01-97 | PASS | B | Lượt Theo chủ đề + Ôn lại câu sai + đề cũ ở /library → Luyện đề này: không câu QUIZ01 nào lọt vào (đối chiếu stem) |
| TC-01-98 | PASS | C | proto-curl.log tc_01_17 (36 dòng) PASS |
| TC-01-99 | PASS | A | audit 01-AC17 (12 dòng) → 0 FAIL |
| TC-01-100 | PASS | B | /join/BX4P9TW: true; sai: Mã không hợp lệ hoặc đã hết hạn. |
| TC-01-101 | PASS | B | Chào Linh Thứ Năm, 29 tháng 10 · bạn chưa vào lớp nào Vào lớp của bạn Yêu cầu vào lớp 761988 đang chờ giảng viên duyệt · gửi lúc 09:20 Khi giảng viên duyệt, lớp |
| TC-01-102 | PASS | B | GV (lớp 761988) Duyệt D: true ‖ nav=7 chooser=761988 · An ninh mạng chat="Chat riêng Hỏi về điểm, chuyên cần và bài học của chính bạn — An ninh " me="Kết quả của tôi An ninh mạng – 761988 · Thứ Ba 07:00–08:30 Cập nhật Thứ Năm, 29 tháng 10 09:20 Lớp này chưa có" |
| TC-01-103 | PASS | B | threads="Threads Câu hỏi công khai của An ninh mạng – 761988. Câu trả lời có nhãn xác nhận là đã đư" lib="9 tài liệu" |
| TC-01-104 | PASS | B | threads=12 lib=10 tài liệu chat="Chat riêng Hỏi về điểm, chuyên cần và bài học của chính bạn — An ninh " history="Luyện đề Lịch sử luyện tập Chưa có lượt luyện nào. Chưa có lượt luyện nào Sau lượt đầu tiê" |
| TC-01-105 | PASS | B | A lớp 2 /me: đúng · chat A: Chat riêng Hỏi về điểm, chuyên cần và bài học của chính bạn  · chat C: Chat riêng Hỏi về điểm, chuyên cần và bài học của chính bạn  |
| TC-01-106 | PASS | C | proto-curl.log tc_01_17 (36 dòng) PASS |
| TC-01-107 | PASS | B | rj=true dlg=\|Để sau\|Tạo lại mã\|\|Để sau\|Mời ra khỏi lớp · Yêu cầu vào lớp 761988 chưa được chấp nhận Lớp học · vừa xong L Phạm Ngọc L |
| TC-01-108 | PASS | P | pii.json pii P1 (2) PASS |
| TC-01-109 | PASS | P | pii.json pii P1 (2) PASS |
| TC-01-110 | PASS | B | http://localhost:3400/chat composer có tiêu đề+mail=true threads=12 |
| TC-01-111 | PASS | P | pii.json pii P2 (2) PASS |
| TC-01-112 | PASS | P | pii.json pii P3,khong-sdt-2048,khong-sdt-0,25,khong-sdt-8443,khong-sdt-1.3.1,khong-sdt-800-57,khong-sdt-QUIZ01,khong-mssv,khong-ten-don,khong-ten-TA,khong-diem-1,khong-diem-2,mail-@AI,mail-lab@2 (14) PASS |
| TC-01-113 | PASS | B | Bài này có thông tin cá nhân Threads là nơi cả lớp cùng đọc. Chúng tôi tìm thấy: 1 họ tên, 1 điểm gắn với một người. Chuyển sang chat riêng Ẩn thông tin rồi đăng |
| TC-01-114 | PASS | P | pii.json pii P1,P1,P2,P2,P3,mail-hoa,mail-hoa,mail-cham-cuoi,mail-cham-cuoi,mail-@AI,mail-lab@2,sdt-10,sdt-10,sdt-cach,sdt-cach,sdt-cham,sdt-cham,sdt-gach,sdt-gach,sdt-+84,sdt-+84,sdt-84,sdt-84,khong-sdt-2048,khong-sdt-0,25,khong-sdt-8443,khong-sdt-1.3.1,khong-sdt-800-57,khong-sdt-QUIZ01,mssv,mssv,khong-mssv,ten-day-du,ten-day-d |
| TC-01-115 | PASS | P | pii.json pii mail-hoa,ten-hoa,ten-NFD,ten-chinh-minh (8) PASS |
| TC-01-116 | PASS | B | Hỏi trợ lý AI + SĐT → dialog=true ‖ tiêu đề PII → dialog=true; Hỏi AI/Gửi phản hồi dialog ok; nội dung PII (TC-01-108) |
| TC-01-117 | PASS | B | ["Đóng","Quay lại sửa","Ẩn thông tin rồi đăng"] ‖ Quay lại sửa giữ chữ=true |
| TC-01-118 | PASS | B | hiện khi gõ=true; mất khi xoá=true |
| TC-01-119 | PASS | B | [["Esc",true,"Bạn Lê Quang Hu"],["×",true,"Bạn Lê Quang Hu"],["ngoài",true,"Bạn Lê Quang Hu"]] lại hiện=true |
| TC-01-120 | PASS | B | rò ở: []; localStorage chứa giá trị gốc=false |
| TC-01-121 | PASS | A | audit 01-AC19 (2 dòng) → 0 FAIL |
| TC-01-122 | PASS | B | có sao không? Trần Thu Uyên (bạn) Mật mã đối xứng · Tuần 10 · vừa xong · 1 người tham gia Em hỏi thêm ạ. Sửa Xoá bài của tôi Thảo luận (0) Phản hồi của bạn Nhập nội dung  |
| TC-01-123 | PASS | B | Thread dài nhất Của bạn xNội dung dài rất dài rất dài rất dài rất dài rất dài rất dài rất dài rất dài rất dài rất dài rất dài rất dài rất dài rấ… Mật mã đối xứng · Tuần 10 · vừa xong · Chưa có câu trả ‖ sau Đặt lại thread đầu: http://localhost:3400/threads/t-new-1 |
| TC-01-124 | PASS | B | Bảo Trợ giảng 2 giờ trước Lê Ngọc Hoa · 3 giờ trước Có bảng khuyến nghị theo năm không ạ? Có, bảng độ dài khoá ở Chương 5 trang 5. Em đọc rồi tự trả lời câu AI hỏi nhé. Trả lời Báo cáo Trần Thu Uyên (bạn) vừa xong Em nghĩ 2048 bit an toàn đến 2030 theo NIST SP 800-57. Trả lời Sửa Xoá Trợ lý AI của lớp Chờ xác nhận ↳ trả lời Trần |
| TC-01-125 | PASS | B | Thử thêm: vì sao khoá ECC 256 bit được xem là tương đương RSA 3072 bit? |
| TC-01-126 | PASS | B | rsa-key có "lan truyền lỗi"=false |
| TC-01-127 | PASS | A | audit 01-AC21 hàng thread (1 dòng) → 0 FAIL |
| TC-01-128 | PASS | A | audit 01-AC21 (dòng nhắc\|"Đăng câu hỏi" khoá) (2 dòng) → 0 FAIL |
| TC-01-129 | PASS | A | audit 01-AC21 mọi nút (1 dòng) → 0 FAIL |
| TC-01-130 | PASS | B | {"d":true,"h":"Nhập nội dung phản hồi"} + "Nhập câu trả lời để kiểm tra"/"Chọn một đáp án để kiểm tra" (TC-01-92/93) |
| TC-01-131 | PASS | A | audit 01-AC22 nút (1 dòng) → 0 FAIL |
| TC-01-132 | PASS | A | audit 01-AC22 (bảng\|chọn phiên) (2 dòng) → 0 FAIL |
| TC-01-133 | PASS | A | audit 01-AC22 (\(#20b\)\|≥ 1100) (4 dòng) → 0 FAIL |
| TC-01-134 | PASS | B | Bài tập 03 đã nộp, đang chờ chấm Bài tập · 6 ngày trước U  \| Tài liệu mới: Chương 5 — Quản lý khoá và PKI  |
| TC-01-135 | PASS | B | duyệt=true chọn=true công bố→true; chuông B: Điểm Bài tập 03 đã được công bố Bài tập · vừa xong Tài  |
| TC-01-136 | PASS | B | chấm trước=false, sau mở-đóng=false, bấm mục=true, sau tải lại=false |
| TC-01-137 | PASS | B | Bạn đã được duyệt vào lớp An ninh mạng – 761988 Lớp học · vừa xong Tài liệu mới: Chương |
| TC-01-138 | PASS | B | chuông B: Giảng viên đã trả lời câu hỏi của bạn Chat riêng · vừa xong Tài liệu mớ → /chat |
| TC-01-139 | PASS | B | B ?course=lớp2 → url=/threads chooser=761987 · An ninh mạng |
| TC-01-140 | PASS | C | proto-curl.log tc_01_24 (11 dòng) PASS |
| TC-01-141 | PASS | B | Nguồn D1 "Quy chế môn học An ninh mạng – 761987 trang 2 · mục Điểm quá trình"; t-cbc "Chương 3 — Mật mã đối xứng và chế độ vận hành · trang 14–17" + "Modern Network Security Threats · mục 2.4"; chuông "Tài liệu mới: Chương 5 —…"; không tên tệp .pdf/.docx |
| TC-01-142 | PASS | B | Tuần: 7 cột rộng 168 px (≥140), thẻ sự kiện 129 px (đã trừ padding), lưới cao theo nội dung (đáy lưới 415, thẻ cuối 403); không còn ~500 px trống; ảnh owner/cal-week-1440 |
| TC-01-143 | FAIL | B | /calendar 1100, chế độ Tháng: nhãn "QUIZ01 đóng · Mật mã đối xứng" rộng 102 px, 1 dòng, bị cắt bằng dấu … ("QUIZ01 đóng ·…", "Buổi 6 · An nin…") thay vì xuống tối đa 2 dòng; Danh sách đủ chữ (247 px, 1 dòng); ảnh owner/cal-Thá-1100.webp |
| TC-01-144 | PASS | B | trước "Cập nhật Thứ Năm, 29 tháng 10 09:20" Điểm quá trình hiện tại 8,3 → GV "Đã lưu 09:21" → "Cập nhật Thứ Năm, 29 tháng 10 09:21" Điểm quá trình hiện tại 8,5 |
| TC-01-145 | PASS | B | mốc Cập nhật Thứ Năm, 29 tháng 10 09:21 → Cập nhật Thứ Năm, 29 tháng 10 09:22; QT Điểm quá trình hiện tại 8,7 |
| TC-01-146 | PASS | A | audit 01-AC27 (12 dòng) → 0 FAIL |
| TC-01-147 | PASS | T | timeline.json T15a+T15b PASS |
| TC-01-148 | PASS | T | timeline.json T15e+T15f PASS |
| TC-01-149 | PASS | T | timeline.json T15c+T15d PASS |
| TC-01-150 | PASS | T | timeline.json T16a+T16b+T16c PASS |
| TC-01-151 | PASS | T | timeline.json T16d+T16e+T16f+T16g PASS |
| TC-01-152 | PASS | B | [["t-pin-lab",21],["t-salt",21],["t-wifi",21],["t-cbc",21],["t-salt Sửa mở",21]] |
| TC-01-153 | FAIL | A + B | `regress-v24.mjs` (v24.json): 7/8 ca FAIL (SV 1440/1100/900/390, GV 1440/900/390); chip cuối bị cắt mép phải ("Tường lửa và phân đoạn mạng" ở ≥ 1100, "An toàn ứng dụng web" ở ≤ 900), hàng có `data-scroll-x` (scrollWidth 1639) nhưng không có mép mờ / nút cuộn nhìn thấy; chỉ GV 1100 đạt. Ảnh `shots/qc-v5/v24/` + `owner/threads-390-mobile.webp` |

## Lỗi

| Mã | Route | Vai | Bước | Thấy | Mong đợi | Mức | AC | Ảnh |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BUG-v5-01-1 | `/threads` | SV, GV, TA 390 / 375 | Mở `/threads` bằng thiết bị cảm ứng 390 | Khung nhìn bị nới tới **772 px** (`innerWidth` 772, `body.scrollWidth` 771); ô tìm, tiêu đề thread, chip chủ đề cắt mép phải; hàng chip `data-scroll-x` scrollWidth 1639 không được bó trong khung | Không tràn ngang ở 375 / 390; chip cuộn trong khung của nó | **cao** | 01-AC6 · TC-01-15 | `owner/threads-390-mobile.webp`, `audit-wide-fails.json` |
| BUG-v5-01-2 | `/threads` (hàng chip chủ đề) | SV, GV | Xem hàng chip ở 1440 → 390 | Chip cuối bị cắt mép phải; có `data-scroll-x` nhưng không mép mờ / nút cuộn thấy được | Có gợi ý cuộn nhìn thấy hoặc chip xuống dòng (#24c) | trung bình | #24c · TC-01-153 | `v24/` |
| BUG-v5-01-3 | `/chat` | SV B | Gõ D1 | `Nguồn tham khảo (1)` **thu gọn**, chỉ "Quy chế môn học … trang 2 · mục Điểm quá trình"; thiếu "Sổ điểm danh lớp 761987" | `Nguồn tham khảo (2)` mở ngay dưới câu trả lời (SRS §14.2) | trung bình | 01-AC2 · TC-01-04 | `owner/d1-sources-1440.webp` |
| BUG-v5-01-4 | `/me` (What-if) | SV B | Xoá chữ trong ô điểm giữa kỳ giả định | Ô rỗng vẫn báo "Nhập một số từ 0 đến 10", `aria-invalid=true` | Ô rỗng không báo lỗi giả (TC) | thấp | 01-AC7 · TC-01-18 | — |
| BUG-v5-01-5 | `/assignments/bt03` | SV B | Sau công bố, đọc 4 tiêu chí | Trích đoạn là bài của nhóm khác ("hệ thống bán lẻ", "VPN nhà thầu", "240 cửa hàng"), không phải bài SolarWinds của B | Mỗi tiêu chí trích đúng một đoạn bài của B | trung bình | TC-01-33 (= BUG-v5-DEMO-2) | `owner/bt03-published.webp` |
| BUG-v5-01-6 | `/calendar` (Tháng) | SV 1100 | Chế độ Tháng | Nhãn "QUIZ01 đóng · Mật mã đối xứng" rộng 102 px, 1 dòng, cắt bằng "…" ("Buổi 6 · An nin…") | Tối đa 2 dòng, không cắt | thấp | 01-AC25 · TC-01-143 | `owner/cal-Thá-1100.webp` |

Quan sát (không FAIL TC): (1) nháp phản hồi mất nếu rời trang < ~0,5 s sau phím cuối (≥ 0,9 s thì còn); (2) sau `Đã rõ` ở `/chat` câu trả lời của GV (D4) biến khỏi khung chat (= BUG-v5-DEMO-4); (3) TC-01-18: giá trị "8,0 → 8,3" chỉ đúng khi QT = 8,7 (ở QT 8,3 là 8,1) — nên sửa TC; (4) bộ lọc `Cần chú ý` của `/students` lưu qua tải lại (khoá `ep_demo_state`).

## Phản mẫu · phân quyền · 375–390

- `ui-antipatterns.sh` sạch; `lint` sạch. Sinh viên không thấy từ kỹ thuật AI (quét 13 route + chat sau D1 + thread sau `Hỏi trợ lý AI`: 0 vi phạm, TC-01-14 / 87).
- Phân quyền: `proto-curl` ma trận chặn route SV (TC-01-27/106) PASS; rò PII: GV, TA, SV không thấy `0912345678` / `uyen.tt229002@` ở Hôm nay, chuông, `/threads`, thread, "AI gốc", `/inbox`, `localStorage` (TC-01-120).
- 375–390: mọi route SV đạt (không tràn, vùng chạm ≥ 44, 0 FAIL TOUCH) **trừ `/threads`** (BUG-v5-01-1).

## Cảm nhận khi dùng như chủ dự án

Luồng Threads tạo → AI → xác nhận đọc mượt (đang soạn 0,05 s, chảy ~2,7 s, nguồn, `Chờ xác nhận`); chuông, hộp thoại PII và TA trả lời muộn hoạt động đúng. Lịch tuần giờ gọn. Mất điểm ở điện thoại: `/threads` bị phóng to, kéo mới thấy hết — đây là màn dùng nhiều nhất của SV nên nên sửa trước demo. `Nguồn tham khảo` ở chat D1 ít hơn spec nên câu trả lời trông kém căn cứ hơn.

## Đề nghị

1. Dev sửa BUG-v5-01-1 (bó hàng chip `/threads` trong `min-width:0`) và #24c (mép mờ / nút cuộn) cùng lúc.
2. PM / BA chốt D1 hiển thị 2 nguồn mở sẵn hay thu gọn; sửa TC-01-04 hoặc code.
3. BA sửa TC-01-18 (giá trị mẫu) và TC-01-33 (4 tên tiêu chí thật: "Xác định tác nhân…", "Phân tích tấn công", "Đánh giá tác động", "Biện pháp phòng thủ").

## Sửa công cụ QC v5 (theo proposals #21, #23)

| Công cụ | Sửa |
| --- | --- |
| `proto-curl.sh` | `${s}»` / `${k}»` đóng ngoặc đúng; `tc_04_08` đếm bằng `grep -o 'Quá 24 giờ' \| wc -l` |
| `audit.mjs` | `TOUCH` bỏ nhãn không bọc/không trỏ tới checkbox-radio; `HEADER đặc` nhận màu `lab()/oklab()`; bấm ưu tiên nút trong; cột thu gọn lấy bề rộng `aside`; kịch bản form thread mở `Đặt câu hỏi` trước; mới `wide` (viewport nở: `max(innerWidth, scrollWidth) − w`) — **bắt được lỗi /threads 390/375 mà `scrollWidth − innerWidth` bỏ sót**; `routesOnly`, `skipScenarios`, `skipEdge`, `specWhich` |
| `pii-matrix.mjs` | chỉ lấy hộp thoại `[role=dialog]` đang hiện có chữ "thông tin cá nhân" |
| `threads-timeline.mjs` | mở form qua `Đặt câu hỏi`; mở "Nguồn tham khảo" trước khi đọc; T4g chờ nền 4 việc của TA; T13 chờ 0,9 s trước khi rời trang; T15e cho +2/+3 rồi +2 |
| `regress-v24.mjs` (mới) | TC-02-90, TC-00-69, TC-01-153, TC-02-91 |
| Môi trường | Chrome riêng `--headless=new --no-first-run --user-data-dir` (cờ GPU/swiftshader làm treo ảnh trên máy này); `caffeinate -d -i` giữ màn hình |
| Nghi lỗi công cụ | audit "bộ chọn lớp mở" GV 1440 FAIL là dương tính giả (`display:none` bị tính là cắt); `/threads` 390/375 lúc đầu tôi nghi lỗi công cụ vì trang không cuộn — **đo tay xác nhận lỗi thật** (innerWidth 772) |
