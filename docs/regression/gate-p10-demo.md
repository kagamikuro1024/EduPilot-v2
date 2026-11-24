# Gate P10 — chạy DEMO_SCRIPT.md trên stack demo (main 23c97510)

Người chạy: QC cặp 3 · ngày 25/11/2026 02:20–03:30 (giờ VN) · stack demo `edupilot` (https://localhost, API https://localhost/api/v1, LLM giả, migrate 29, dữ liệu demo của chủ dự án giữ nguyên, bản sao lưu ở `~/Documents/EduPilot-demo-backups/`) · kịch bản: `docs/DEMO_SCRIPT.md` @`origin/main` (v2, 245 dòng).

Cách chạy: 5 trình duyệt Chromium headless RIÊNG (mỗi vai một hồ sơ, không dùng chung phiên): Admin 1440×900, Giảng viên 1440×900, Sinh viên B (`sv.kha`), Sinh viên D (`sv.moi` → dùng lại cho các sinh viên quét QR) và Giảng viên 375 px (phiên thứ hai của `teacher`) ở 375×812 cảm ứng; mọi thao tác bằng giao diện (bấm, gõ, tải tệp); không seed, không reset, không lệnh admin ghi dữ liệu; chỉ đọc DB (`psql select`) để đối chiếu. Camera không có → «quét QR» = mở đúng `qr_url` mà màn chiếu phát (tuổi ≤ 10 s). Ảnh từng bước: `evidence-gate-p10/` (cột «Ảnh»), nhật ký thô: `evidence-gate-p10/rec.json`.

## 1. Kết quả

**52 mục chấm: 39 P · 9 F · 4 N** (8 bước chính + chuẩn bị #3/#4/#5 + «Nếu hội đồng hỏi» + «Nếu còn giờ» kiểm được).

| Nhóm | P | F | N | Ghi chú |
| --- | --- | --- | --- | --- |
| Bước 1 (F2) | 4 | 0 | 0 | đạt |
| Bước 2 (F3) | 2 | 1 | 0 | D1 đúng số liệu nhưng khối «Kết quả» hiện khoá tiếng Anh + JSON thô |
| Bước 3 (F5) | 4 | 2 | 0 | không thấy tên người nhận; bấm chuông tại /chat → khung chat trống |
| Bước 4 (F7) | 4 | 1 | 1 | không có buổi học HÔM NAY (thiếu seed); mã ngắn bị cắt ở 1440 px |
| Bước 5 (F9) | 7 | 0 | 0 | đạt (chuẩn bị #3 phải làm tay ≈ 12–13 phút) |
| Bước 6 (F10) | 4 | 0 | 1 | Bài tập 03 chưa gắn cột điểm → điểm 7,50 không vào sổ |
| Bước 7 (F17) | 1 | 2 | 0 | **ĐỎ với embedding/LLM giả**: không có chủ đề A/B đứng đầu |
| Bước 8 | 1 | 1 | 0 | tên thực thể trong kịch bản khác thực tế |
| Hội đồng hỏi | 6 | 1 | 1 | |
| Còn giờ | 6 | 1 | 1 | |

**Kết luận cho PM:** luồng chạy được từ đầu tới cuối trừ Bước 7 (nội dung báo cáo vô nghĩa với embedding giả) và hai thiếu hụt dữ liệu seed (không có buổi học hôm nay; 761988 không có câu hỏi). Chưa nên bảo vệ ở «tầng 2» cho Bước 7: cần embedding thật hoặc bỏ/đổi bước. 4 phát hiện về giao diện/chữ cần quyết (mục 5).

## 2. Thời gian

Chỉ đo được độ trễ hệ thống (cột «Hệ thống phản hồi» trong các bảng); lời dẫn và thời gian gõ của người trình bày KHÔNG đo được trong lượt tự động, nên **không kết luận được «≤ 15 phút»**. Các điểm đo được:

| Việc | Đo được |
| --- | --- |
| Chat D1 / D2 / D3 (LLM giả) | 1,1 s / 0,6 s / 2,1 s (SSE đầu tiên < 1 s) |
| Nhận / trả lời ticket | 1,8 s / 2,7 s; mail tới Mailpit ≤ 1 s sau «Gửi trả lời» |
| Mở điểm danh → màn chiếu | 2,5 s; mã đổi mỗi 10 s; số đã điểm danh cập nhật ≤ 10 s |
| Đóng + đánh vắng 27 + phát biểu + Lưu (4 thao tác) | ≈ 45 s thao tác máy (kịch bản: < 60 s) |
| Tải quy chế → bản nháp công thức | ≈ 5 s (docling bật) |
| Xác nhận công thức → sổ điểm | 3,2 s + 6,8 s |
| Tạo báo cáo lỗ hổng (Toàn kỳ, 162 câu, embedding giả) | ≈ 3,6 s |
| **Chuẩn bị #3 «Chấm tất cả» BT03 (22 bài, LLM giả)** | **≈ 12–13 phút** (xếp hàng, worker chấm lần lượt) — PHẢI làm trước, không làm được trong 15 phút |

## 3. Chuẩn bị (T−30) — trạng thái thật của demo lúc bắt đầu

| # | Việc | Thực tế |
| --- | --- | --- |
| #1 | Stack + `COMPOSE_PROFILES=ingest` | **Thiếu docling** (không có container `edupilot-docling-1`) và `BLOB_PUBLIC_ENDPOINT=localhost:9000` (CSP `connect-src 'self'` chặn PUT URL ký sẵn): mọi tải tệp lỗi «Không kết nối được tới máy chủ». PM đã sửa giữa lượt (blob → `localhost` + SSL; bật docling); sau đó 6.2 chạy được. |
| #2 | Tầng LLM | Không có nhà cung cấp nào ở `/settings/llm` (chỉ «Quyết định»/decision-fake) → env `fake` (tầng 2). |
| #3 | Chấm nháp BT03 | **Chưa làm** (0/22 đã chấm). QC bấm «Chấm tất cả» (UI): xong sau ≈ 12–13 phút; bài của SV B có cờ «Hai lượt chấm lệch hơn 1 điểm» ✔. |
| #4 | Báo cáo lỗ hổng 2 lớp | Tạo bằng UI «Tạo báo cáo mới»: 761987 chỉ chạy được với khoảng «Toàn kỳ» (dữ liệu seed cũ hơn 14 ngày → «Chưa đủ câu hỏi (cần ≥ 20)»); 761988 có 0 câu hỏi → không tạo được. Kết quả vô nghĩa (mục Bước 7). |
| #5 | Điểm danh bằng mã + buổi học hôm nay | Công tắc «Điểm danh bằng mã» của 761987 TẮT → QC bật trong `/class/settings`; **không có buổi học ngày 25/11** (buổi 11 = 19/11, buổi 12 = 26/11) → dùng Buổi 11 (đã qua) thay thế, nên cả 3 SV quét ra «Muộn». |
| #6–#9, #11 | Hồ sơ trình duyệt, Mailpit, PDF, video | Mailpit hộp trống ✔; PDF `seed/demo/quy-che-lop2.pdf` có trong `main` ✔; video: không kiểm. |
| #10 | Có sẵn dòng nhật ký | Có sau khi QC chạy Bước 4–6 (27 dòng). |

## 4. Kết quả chi tiết từng bước

### Bước 1 — Mở lớp → vào lớp (F2)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 1.1 | Admin /admin/courses | **P** | 2.2 s | Danh sách lớp: 761987, 761988; không có «Chưa gán» | [ảnh](evidence-gate-p10/b1-1-admin-courses.png) |
| 1.2 | GV Hôm nay + chuông | **P** | – | Chuông có «Bạn được phân công lớp An ninh mạng – 761988» (Phân công lớp, 12/11 19:48) và «…761987»; Hôm nay (lớp 761988) có «Thiết lập lớp mới – 3/4 bước xong». LỆCH CHỮ nhỏ: dòng chuông KHÔNG kèm mã tham gia BX4P9TW như kịch bản (mã có ở màn khác). Lưu ý: bộ chọn lớp của GV mở mặc định ở 761988, kịch bản ghi 761987 | [ảnh](evidence-gate-p10/b1-2-teacher-bell-761988.png) |
| 1.3 | SV D (375) /join/BX4P9TW | **P** | 2.6 s | Xem trước: «An ninh mạng · 761988 · TS. Lê Thu Hà · HK1 2026–2027», «Lớp này cần giảng viên duyệt.»; sau bấm:  / Bản mô phỏng · dữ liệu giả / Tham gia lớp / Đã gửi yêu cầu, chờ giảng viên duyệt. / Về Hôm nay / Hôm nay | [ảnh](evidence-gate-p10/b1-3-svd-after-join.png) |
| 1.4 | GV /class/members duyệt SV D | **P** | 2.7 s | Tab «Chờ duyệt» liệt kê 4 (sv.moi + 3 seed); sau «Duyệt» (hàng sv.moi): còn 3 yêu cầu chờ (sv52–54); sv.moi rời danh sách chờ: true | [ảnh](evidence-gate-p10/b1-4-after-approve.png) |

### Bước 2 — Hỏi riêng tư (F3)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 2.1 | SV B (375) D1 /chat lớp 1 | **F** | 1.1 s | Nội dung ĐÚNG: «Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI»; câu dẫn không nêu số; khối dữ liệu từ sổ: absent 2, present 6, late 1, excused 0, points_total 0.50 (speak_count 2, 2×SPEAK 0.25 ngày 12/11); không có [[SV_n]]; toàn bộ phản hồi 1,1 s. LỆCH TRÌNH BÀY: khối «Kết quả» hiện KHOÁ KỸ THUẬT TIẾNG ANH + JSON thô (late/absent/excused/present/recorded/recent_absent {"date":…}, recent, speak_count, points_total) cho sinh viên, trái DESIGN §13/luật chữ UI; kịch bản chỉ nói «khối gọn nêu số buổi vắng (2) và điểm cộng». Ghi nhận (Thấp–Trung bình, PM xếp) | [ảnh](evidence-gate-p10/b2-1-d1.png) |
| 2.2 | SV B bấm «Hữu ích» | **P** | 0.9 s | Nút đổi trạng thái tại chỗ; thông báo nổi 'Thành công/Cảm ơn': không | [ảnh](evidence-gate-p10/b2-2-huuich.png) |
| 2.3 | SV B D2 (hỏi hộ bạn C) | **P** | 0.6 s | Trả lời: «Trợ lý AI / Mình chỉ xem được dữ liệu của chính bạn. / Hữu ích / Không hữu ích / Gửi / Hôm nay / Chat riêng / Hỏi đáp / Luyện đề / Thêm»; không con số về C: true | [ảnh](evidence-gate-p10/b2-3-d2.png) |

### Bước 3 — Cần giảng viên hỗ trợ (F5)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 3.1 | SV B D3 /chat | **P** | 2.1 s | «AI chưa đủ chắc chắn về câu này» + «Mình chưa tìm thấy nội dung này trong tài liệu của lớp.» + «Đang chờ giảng viên · vừa gửi»; tự chuyển ticket không cần bấm «Nhờ giảng viên hỗ trợ» (Q3); 2,1 s | [ảnh](evidence-gate-p10/b3-1-d3.png) |
| 3.2 | GV / → /inbox, mở ticket mới nhất | **P** | – | Hôm nay (lớp 761987) có «Câu hỏi chờ giảng viên» (Gấp·2); /inbox lọc «Đang chờ» có ticket «Bùi Thanh Khải · 0 phút · Thi cuối kỳ có được mang một tờ A4…»; chi tiết: câu hỏi gốc, «Đang chờ · 0 phút», «Độ tin cậy 0,00». Lưu ý: ticket mới nhất nằm DƯỚI 2 ticket seed cũ (20 và 12 ngày) — người trình bày phải chọn đúng dòng | [ảnh](evidence-gate-p10/b3-2-ticket-detail.png) |
| 3.3 | GV /inbox «Nhận» | **F** | 1.8 s | Sau «Nhận»: trạng thái «Đã nhận · 0 phút», hiện «Gửi trả lời / Bỏ nhận / Đóng không trả lời». KHÔNG thấy TÊN người nhận trên chi tiết/dòng (kịch bản: «Claimed, ghi tên người nhận»). Chỉ có một giảng viên nên không phân biệt được; ghi nhận Thấp (PM/BA: kịch bản hay UI) | [ảnh](evidence-gate-p10/b3-3-claimed.png) |
| 3.4 | GV D4 «Gửi trả lời» | **P** | 2.7 s | Sau gửi: Hộp thư hỗ trợ / Đang chờ / Đã nhận / Đã trả lời / Tất cả / Phạm Mai Chi / 20 ngày / Tấn công chuỗi cung ứng phần mềm là gì? / Đang chờ / Ngô Văn Hải / 12 ngày / Giải thích giúp em: Đánh giá rủi ro an ninh gồm những bước nào? / Đang chờ / Bùi Thanh Khải / Đã t | [ảnh](evidence-gate-p10/b3-4-answered.png) |
| 3.5 | SV B (375) /chat mở chuông | **F** | 6 s | Chuông có «Giảng viên đã trả lời câu hỏi của bạn · vừa xong». ĐÚNG khi mở từ «/» (vào /chat?session=… thấy «Giảng viên đã trả lời · Giảng viên · TS. Lê Thu Hà» + câu D4 + nút «Hỏi tiếp / Đã rõ»). SAI khi bấm thông báo lúc ĐANG Ở /chat (đúng kịch bản 04:50): URL đổi sang ?session=… nhưng khung chat trống «Hỏi bất cứ điều gì về lớp này», chỉ hiện câu trả lời sau khi tải lại trang. Ghi nhận Thấp–Trung bình (1 lần tái hiện, ảnh b3-5-svb-answer.png) | [ảnh](evidence-gate-p10/b3-5-svb-answer.png) |
| 3.6 | Mailpit thư mới nhất | **P** | – | Mailpit :8025 (hộp trống trước đó, 1 thư mới): tới sv.kha@edupilot.local, tiêu đề «Giảng viên đã trả lời câu hỏi của bạn», nội dung câu hỏi + lời GV TS. Lê Thu Hà (D4) + link «Xem và hỏi tiếp: https://localhost/chat?session=…&m=…» | [ảnh](evidence-gate-p10/b3-6-mailpit.png) |

### Bước 4 — Điểm danh bằng QR và phát biểu (F7)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 4.1 | GV chạm việc «Điểm danh buổi đang diễn ra» | **N** | – | THIẾU DỮ LIỆU SEED: không có buổi học NGÀY HÔM NAY (25/11) ở 761987 (buổi 11 = 19/11, buổi 12 = 26/11 09:00 — «Ngày mai»), nên Hôm nay không có việc «Điểm danh buổi đang diễn ra – 761987» và /attendance mở mặc định ở Buổi 11 · 19/11 · Chưa. Thêm: công tắc «Điểm danh bằng mã» của 761987 đang TẮT (chuẩn bị #5 chưa làm) — QC bật trong /class/settings (GV, UI) để đi tiếp; dùng Buổi 11 (19/11) thay buổi hôm nay | [ảnh](evidence-gate-p10/b4-1-attendance-after-enable.png) |
| 4.2 | GV /attendance «Mở điểm danh» → màn chiếu | **F** | 2.5 s | Đã làm được: 1 lần bấm «Mở điểm danh» (POST checkin-windows 201) và trình duyệt vào thẳng /attendance/display (CÙNG tab, không có nút «Mở màn chiếu» riêng, không cửa sổ mới): QR, mã ngắn 6 ký tự dạng «EA2 86J» ở BÊN PHẢI QR (không «ngay dưới»), số đã điểm danh, «còn 09:50» đếm ngược, mã đổi mỗi 10 s (round/short_code đổi, qr_url có token đổi). LỆCH: ở viewport 1440×900 mã ngắn (126 px) tràn ra ngoài khung (right 1485 > 1440, bị cắt «VQ/»; scrollWidth 1485); ở 1920×1080 vừa. Quan trọng cho máy chiếu 1280/1440. Ghi nhận Thấp–Trung bình | [ảnh](evidence-gate-p10/b4-2-display.png) |
| 4.3 | 2 SV (375) quét QR | **P** | 10 s | Camera hệ thống KHÔNG dùng được (headless): QC mở đúng qr_url «/checkin?c=<course>#<window>.<…>» mà màn chiếu phát (lấy từ API display, ≤ 10 s tuổi) cho 3 sinh viên: sv.kha (B), sv.nguyco, sv.gioi → mỗi người thấy «Điểm danh · Đã điểm danh 02:37/02:38» và KHÔNG có nút nào; màn chiếu: số đã điểm danh 0 → 3 (cập nhật sau ≤ 10 s). Phần quét bằng camera: N (không kiểm được) | [ảnh](evidence-gate-p10/b4-3-display-count.png) |
| 4.4 | GV (375) «Đóng», đánh vắng SV chưa quét, phát biểu +0,25 cho B | **P** | 45 s | Sau «Đóng»: «Đã ghi 0 có mặt · 3 muộn · 27 chưa điểm danh» (cả 3 SV quét ra «Muộn» vì QC phải dùng Buổi 11 · 19/11 đã qua, không phải buổi hôm nay); nút «Đánh vắng 27 SV chưa điểm danh» → hộp xác nhận «Đánh vắng 27 sinh viên?» → 27 dòng «Vắng»; «Ghi phát biểu · Bùi Thanh Khải» → dòng ghi «+0,25×1»; mỗi SV quét có «Điểm danh lúc 02:37». LỆCH nhẹ so kịch bản: vắng là thao tác hàng loạt có hộp xác nhận (không chạm từng dòng); «Muộn cho 1» không cần vì cả 3 đã Muộn | [ảnh](evidence-gate-p10/b4-4-speak.png) |
| 4.5 | GV «Lưu điểm danh» | **P** | 2.8 s | Nút «Lưu điểm danh · 3 có mặt · 27 vắng» → dòng tại chỗ «Đã lưu · 3 có mặt · 27 vắng»; Buổi 11 đổi sang «Đã điểm danh»; không thông báo nổi «Thành công»; 2,8 s. Nhánh «Còn n SV chưa điểm danh (422)» không xuất hiện vì đã đánh vắng hết (chưa kiểm riêng) | [ảnh](evidence-gate-p10/b4-5-saved.png) |
| 4.6 | SV B (375) /me | **P** | – | Hàng «Buổi 11 · 19/11 · Muộn · Điểm danh lúc 02:37» (không phải «hôm nay» vì QC dùng buổi 11); «Điểm cộng 0,75 · tối đa 1» (0,50 trước + 0,25 mới); giải trình dạng phép tính («Điểm trừ 0 · vắng 2 buổi, được miễn 2, mỗi buổi trừ 0,5», «Điểm quá trình trước điểm cộng/trừ 7,75 → 8,5») | [ảnh](evidence-gate-p10/b4-6-svb-me.png) |

### Bước 5 — Duyệt và công bố bài tập (F9)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 5.0 | Chuẩn bị #3 «Chấm tất cả» BT03 (GV, UI) | **P** | – | Bấm «Chấm tất cả» (đã chọn Bài tập 03) → «Đã xếp hàng chấm 22 bài.» (Bài tập 03 trước đó 0/22 đã chấm: chuẩn bị #3 chưa làm). Với LLM giả worker chấm xong 22 bài sau ≈ 12–13 phút (kịch bản coi là việc chuẩn bị); kết quả: 17 bài ở lọc «Cần xem kỹ» (lý do: «Trích dẫn không có trong bài», «Tổng điểm chưa khớp», «Nghi chèn lệnh», «Bài lạc đề», «Thiếu nội dung»), bài của Bùi Thanh Khải: «Hai lượt chấm lệch hơn 1 điểm», 6,50, «muộn 1 ngày» | [ảnh](evidence-gate-p10/b5-0-after-grade-all.png) |
| 5.1 | SV B (375) /assignments/[id] Bài tập 03 | **P** | – | Trang chi tiết: «Đã nộp Chủ Nhật, 08/11 · 20:04 · Nộp muộn 1 ngày»; KHÔNG có điểm/nhận xét nháp (đúng); danh sách /assignments ghi «Đã nộp, chờ điểm». LỆCH nhỏ: trang chi tiết không có dòng chữ «đang chấm/chờ điểm» (chỉ danh sách có); phần «Đề bài» hiện markdown thô («# Bài tập 03…», «## Đề bài») thay vì định dạng — Thấp | [ảnh](evidence-gate-p10/b5-1-svb-assignment03.png) |
| 5.2 | GV /grading hàng chờ (lọc mặc định) | **P** | 3 s | Lọc mặc định «Cần xem kỹ» + «Chưa duyệt»: 17 bài; dòng Bùi Thanh Khải: «Hai lượt chấm lệch hơn 1 điểm» (6,50, muộn 1 ngày) ✔. LỆCH: KHÔNG ở đầu danh sách mà ở vị trí 16/17 (xếp theo giờ nộp, không theo mức ưu tiên); 17 dòng cờ khác (Trích dẫn không có trong bài ×11, Tổng điểm chưa khớp ×3, …) — người trình bày phải tìm dòng B | [ảnh](evidence-gate-p10/b5-0-after-grade-all.png) |
| 5.3 | GV /grading/[id]: đọc bài + sửa điểm tiêu chí lệch + sửa nhận xét | **P** | 3.2 s | Trái: bài nộp; phải: 4 tiêu chí, mỗi tiêu chí có đoạn trích «“…”»; dòng cảnh báo tại «Phân tích tấn công»: «Lệch 1,5 điểm so với lượt 2 (1,00 và 2,50)»; sửa điểm tiêu chí lệch 1,00 → 2,00 và một nhận xét: Tổng điểm 6,50 → 7,50 ngay lập tức (trừ nộp muộn 0,50) | [ảnh](evidence-gate-p10/b5-3-edited.png) |
| 5.4 | GV «Duyệt bài» | **P** | 2.7 s | Sau «Duyệt bài»: Đã duyệt bài trước  / Đã duyệt 1/17  | [ảnh](evidence-gate-p10/b5-4-approved.png) |
| 5.5 | GV /grading: chọn bài đã duyệt → «Công bố» | **P** | – | Lọc «Đã duyệt» → tích «Chọn bài của Bùi Thanh Khải» → «Công bố» → hộp xác nhận «Công bố 1 bài · 1 sinh viên nhận thư · không tính vào sổ điểm (bài chưa gắn vào cột điểm)» → «Công bố»; lọc «Đã công bố» có dòng B (7,50). LƯU Ý: bài tập 03 CHƯA gắn vào cột điểm nên điểm KHÔNG vào sổ (ảnh hưởng Bước 6 dòng 10:20); hành động chỉ-Teacher: không kiểm bằng TA | [ảnh](evidence-gate-p10/b5-5-published-list.png) |
| 5.6 | SV B tải lại /assignments/[id] | **P** | – | «Kết quả 7,50 / 10,00»; từng tiêu chí «2,00 / 2,50» + nhận xét (kể cả câu GV sửa «thầy cần thêm bằng chứng…») + đoạn trích «“Bề mặt tấn công bị khai thác…”» từ bài của chính SV; nhãn dưới cùng «Đã có điểm» | [ảnh](evidence-gate-p10/b5-6-svb-result.png) |

### Bước 6 — Công thức điểm và sổ điểm (F10)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 6.1 | GV /gradebook lớp 2 | **P** | 4 s | Đổi sang 761988: banner «Công thức điểm chưa xác nhận» (+ «Chưa có công thức điểm đã xác nhận», nút «Tải quy chế» / «Thiết lập công thức»); «Chốt điểm» disabled; tooltip khi rê: «Chưa có công thức điểm đã xác nhận» | [ảnh](evidence-gate-p10/b6-1-gradebook-hover.png) |
| 6.2 | GV /gradebook/scheme lớp 2: «Tải quy chế» quy-che-lop2.pdf | **P** | – | SAU KHI PM sửa BLOB_PUBLIC_ENDPOINT=localhost / USE_SSL=true và bật docling (profile ingest), xoá tài liệu lỗi do lần tải hỏng trước và tải lại: tiến độ «Đang đọc · quy-che-lop2.pdf» rồi bản nháp sau ≈ 5 s: «Bản nháp», thành phần Giữa kỳ 25% / Bài thực hành 25% / Thi cuối kỳ 50% (Tổng 100%), Điểm cộng «+0,25 mỗi lần phát biểu · tối đa +1,5», Điểm trừ «−0,5 mỗi buổi vắng · miễn 1 buổi…», mỗi mục kèm «Trang 1/2», «Còn 1 mục chưa rõ: làm tròn» và nút «Xác nhận công thức». LƯU Ý (Thấp): lần tải đầu (docling chưa bật) sinh tài liệu «Sẵn sàng» nhưng KHÔNG có bản nháp công thức, tải lại bị chặn «Tệp này đã có trong lớp.» → phải xoá tài liệu mới đi tiếp (đường tự phục hồi thiếu) | [ảnh](evidence-gate-p10/b6-2-scheme-draft.png) |
| 6.3 | GV điền D5 → «Xác nhận công thức» → hộp thoại hậu quả → xác nhận | **P** | 3.2 s | Chọn «Bước làm tròn» = «Làm tròn đến 0,1» → nút «Xác nhận công thức» mở khoá (trước đó disabled); hộp thoại «Áp cho 25 sinh viên · điểm học phần của 0 sinh viên sẽ đổi so với công thức hiện tại · 0 điểm đã nhập được giữ · 0 điểm đã nhập bị bỏ» (Để sau / Xác nhận công thức); sau xác nhận: Công thức điểm / Tải quy chế / Đã xác nhận · Hôm nay, 03:13 / Thành phần / Tổng 100% / Trang 1 / Giữa kỳ · 25% · Quá trình / Bài thực hành · 25% · Quá trình / T. Lưu ý: «Làm tròn đến 0,1» là LỰA CHỌN trong danh sách (0,01/0,1/0,25/0,5/1), không phải ô gõ chữ | [ảnh](evidence-gate-p10/b6-3-confirmed.png) |
| 6.4 | GV /gradebook lớp 2 sau xác nhận | **P** | 6.8 s | Banner «Công thức điểm chưa xác nhận» biến mất; «Chốt điểm» mở | [ảnh](evidence-gate-p10/b6-4-gradebook-after.png) |
| 6.5 | GV /gradebook lớp 1: giải trình dòng SV B | **N** | – | ĐẠT một phần: giải trình «Bùi Thanh Khải» có «Điểm cộng min(0,75; 1) = 0,75» (đã gồm +0,25 từ Bước 4 ✔), «Điểm trừ», «Điểm quá trình (20×7,5+20×8)÷40 → 8,5», «Điểm học phần thô 7,3 → làm tròn 0,5 …» và dòng «Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.» ✔. KHÔNG ĐẠT: không có thành phần/điểm «Bài tập 03» (7,50 của Bước 5) vì công thức lớp 1 chỉ có Giữa kỳ / Kiểm tra hằng tuần / Thi cuối kỳ và hộp xác nhận «Công bố» đã báo «không tính vào sổ điểm (bài chưa gắn vào cột điểm)». THIẾU CẤU HÌNH SEED (bài tập chưa gắn cột điểm), không phải lỗi mã | [ảnh](evidence-gate-p10/b6-5-explain-B.png) |

### Bước 7 — Báo cáo lỗ hổng kiến thức (F17)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 7.1 | GV /insights lớp 761987 | **F** | – | Báo cáo tạo bằng UI «Tạo báo cáo mới» (khoảng «Toàn kỳ»: 162 câu · 30 SV; khoảng mặc định «14 ngày»/«7»/«30 ngày» của lớp 761987 báo «Chưa đủ câu hỏi để gom (cần ≥ 20)» vì câu hỏi seed cuối cùng ngày 08/11 — QUÁ 14 NGÀY so với 25/11; 761988 = 0 câu hỏi → «Chưa đủ câu hỏi để gom (cần ≥ 20)», nghĩa là kịch bản «lớp 761988: TCP/UDP (C) đứng đầu» không có dữ liệu). KẾT QUẢ SAI KỲ VỌNG: 152 chủ đề, 142 chủ đề 1 SV + 10 chủ đề 2 SV, KHÔNG có chủ đề nào ≥3 SV; đầu bảng là «huong, man, middle», «bang, cach, chan», «diem, doc, khac», «botnet, dung»… tên tạm theo từ khoá + «Chưa đặt được tên chủ đề», mọi chủ đề đều «Ít người hỏi» và nhãn chương «Quy chế học vụ»; KHÔNG có chủ đề A (mã độc/tấn công) / B (quy chế) đứng đầu. Nguyên nhân: embedding GIẢ (hash) trên demo không gom theo nghĩa (kịch bản đã ghi [CHƯA KIỂM: gom cụm với embedding giả]) — bước này ĐỎ ở tầng LLM giả; cần embedding thật hoặc dữ liệu/ngưỡng khác | [ảnh](evidence-gate-p10/prep4-insights-761987-after.png) |
| 7.2 | GV mở «Câu hỏi mẫu» của chủ đề A | **F** | – | Không có chủ đề A để mở. Nhánh phụ đúng kịch bản: chủ đề «Ít người hỏi» (<3 SV) KHÔNG có «Câu hỏi mẫu» và KHÔNG có gợi ý «Nên làm» (✔); nhưng 3–5 câu mẫu ẩn danh của chủ đề A không kiểm được | [ảnh](evidence-gate-p10/prep4-insights-761987-full.png) |
| 7.3 | GV «Tạo thread ghim» (chủ đề hạng 1) | **P** | 5 s | Form mở TẠI CHỖ dưới chủ đề (Tiêu đề điền sẵn «Chủ đề đang vướng: huong, man, middle», Nội dung «Nhiều bạn đang hỏi về “…”.», nút «Đăng»/«Huỷ»), không hộp thoại; «Đăng» → «Đã đăng và ghim thread. Xem ở Hỏi đáp công khai»; /threads có thread ghim: CÓ. LƯU Ý: dùng chủ đề hạng 1 của báo cáo giả («huong, man, middle», tên từ khoá) vì không có chủ đề A; thread đã tạo (liệt kê cuối báo cáo) | [ảnh](evidence-gate-p10/b7-3-threads.png) |

### Bước 8 — Nhật ký quản trị

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 8.1 | Admin /admin/audit (24 giờ) | **P** | – | Admin thấy các dòng vừa tạo ở Bước 3–6 (27 dòng): attendance (complete, absent_unchecked, course_attendance_updated), checkin_window.open/close, participation create, grading_results (grading_result.edit/review/publish), grade_schemes (extract/edit/confirm), escalation_ticket (created/claimed/answered/confirmed), thread.pin, enrollment (member_joined/member_approved), document.delete, gradebook export; mỗi dòng có Thời điểm / Người (kèm vai) / Thực thể / Hành động / Khoá đổi, KHÔNG có giá trị cũ/mới ✔. LỆCH TÊN so kịch bản: «assignments (publish)» thực tế là «grading_results · grading_result.publish»; «grade_entries (set)» KHÔNG có (Bước 6 sinh «grade_schemes · extract/edit/confirm») | [ảnh](evidence-gate-p10/b8-1-audit-2.png) |
| 8.2 | Admin ô «Thực thể» gõ grade_entries | **F** | – | Gõ «grade_entries» → «Không có» (0 dòng); chỉ khi gõ «grade_schemes» mới ra 3 dòng sổ điểm của Bước 6 (extract/edit/confirm). Kịch bản (và chuẩn bị #10: «xác nhận công thức tạo dòng grade_entries / set») sai tên thực thể — sửa kịch bản thành «grade_schemes» hoặc nhập điểm tay trong sổ để có grade_entries | [ảnh](evidence-gate-p10/b8-2-filter-grade_entries.png) |

### «Nếu hội đồng hỏi» kiểm được

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| H1 | Hội đồng hỏi B1: nhập sai mã tham gia | **P** | – | SV (sv.nguyco) vào /join/<mã sai> 5 lần: cùng một thông báo «Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.»; lần 6 và 7: POST /courses/join/preview 429 → «Bạn đã thử quá nhiều lần. Thử lại sau 09:47.» (khoá 10 phút ✔). «Tạo lại mã → mã cũ bị từ chối» KHÔNG chạy (tránh đổi mã AN7K2MQ/BX4P9TW của demo): N | [ảnh](evidence-gate-p10/h1-wrong-code.png) |
| H3 | Hội đồng hỏi B3: ticket chờ quá 72 giờ nổi lên Hôm nay | **P** | – | Hôm nay của GV (761987) có «Gấp · 2 — Câu hỏi chờ giảng viên: 1 câu hỏi chờ quá 72 giờ làm việc · lâu nhất 342 giờ» ✔. Hai người cùng «Nhận» → 409: chỉ có 1 giảng viên nên N; mail lỗi thử lại 3 lần: N (mail mock) | [ảnh](evidence-gate-p10/b3-2-teacher-today.png) |
| H4a | Hội đồng hỏi B4: mã QR cũ hết hạn | **P** | – | Dùng qr_url cũ 28 s (>2 chu kỳ 10 s): «Mã đã hết hạn. Quét lại mã trên màn chiếu.» (Vòng 2 của Buổi 11 mở bằng «Thêm hành động → Mở vòng 2») | [ảnh](evidence-gate-p10/h4-expired.png) |
| H4b | Hội đồng hỏi B4: SV không thuộc lớp quét QR | **F** | – | sv.moi (không thuộc 761987) quét QR hợp lệ → bị từ chối nhưng LỘ LÝ DO: «Bạn không thuộc lớp này.» (kịch bản: «bị từ chối, không lộ lý do»). Ghi nhận Thấp, PM/BA chốt chữ | [ảnh](evidence-gate-p10/h4-nonmember.png) |
| H4c | Hội đồng hỏi B4: hai SV cùng một điện thoại | **P** | – | Trên cùng một trình duyệt, sv04 nhập mã ngắn (Đã điểm danh 03:09) rồi sv05 quét QR: hàng của sv04 ở /attendance ghi «Cùng máy với bạn khác» và bộ lọc «Cần xem» tăng (3 → 5) | [ảnh](evidence-gate-p10/h4-flagged-list.png) |
| H5 | Hội đồng hỏi B5: nộp sau hạn khi không cho nộp muộn | **P** | – | SV B nộp «late-test.txt» vào Bài tập 04 (quá hạn 5 ngày; mô tả cho phép muộn tối đa 2 ngày) → «Đã hết hạn nộp bài.» và KHÔNG tạo bài nộp (Chưa nộp). Lưu ý nội dung Bài tập 04 trong demo là bản sao «Bài tập 03» (đề/tiêu chí/ngày muộn) — Thấp (seed). «Worker chết giữa chừng → không chấm đôi»: N | [ảnh](evidence-gate-p10/h5-bt04-submit.png) |
| H7 | Hội đồng hỏi B7: khoảng 7/14/30/toàn kỳ; 3 báo cáo/giờ; hai người cùng bấm | **N** | – | Khoảng 7 / 14 / 30 ngày / Toàn kỳ có đủ 4 lựa chọn; «14 ngày» (và 7/30) báo «Chưa đủ câu hỏi để gom (cần ≥ 20)» vì dữ liệu seed cũ hơn 14 ngày; «Toàn kỳ» tạo được báo cáo (✔ cùng tầng embedding giả). Bấm «Tạo báo cáo mới» 4 lần liên tiếp: chỉ lần đầu tạo (DB insight_reports 761987: 19:30:50 và 20:14:25 UTC), 3 lần sau KHÔNG tạo thêm và KHÔNG báo gì (không thấy «3 báo cáo mỗi giờ» hay «đang chạy») → giới hạn 3/giờ và «hai người cùng bấm» không dựng được (N); lưu ý bấm lặp im lặng | [ảnh](evidence-gate-p10/h7-limit.png) |
| H8 | Hội đồng hỏi B8: GV/SV vào /admin/audit; mở nhật ký được ghi | **P** | – | UI GV và SV: «Bạn không có quyền xem màn này · Trang này dành cho quản trị viên.»; API GET /admin/audit: teacher 403, sv.kha 403; Admin mở /admin/audit → audit_log có dòng VIEW_LIST (entity audit_log) = việc mở nhật ký được ghi lại | [ảnh](evidence-gate-p10/h8-teacher-audit.png) |

### «Nếu còn giờ» (dự trữ)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 1.x | Còn giờ B1: SV D tải lại / | **P** | 3.1 s | Bộ chọn lớp: 761988 ; ô nhập mã: không còn | [ảnh](evidence-gate-p10/b1-x-svd-reload.png) |
| x.B2 | Còn giờ B2: SV B mở «Nguồn tham khảo» dưới câu trả lời D1 | **F** | – | Dưới câu trả lời D1 KHÔNG có nút/khối «Nguồn tham khảo» (D1 là câu tra sổ điểm danh, không trích tài liệu); chỉ có «Tìm hiểu» (giải thích che thông tin cá nhân: «Tên, mã số sinh viên, email và số điện thoại được thay bằng ký hiệu trước khi gửi cho AI, rồi hiện lại cho bạn.») và «Hữu ích / Không hữu ích / Nhờ giảng viên hỗ trợ». Kịch bản sai chỗ hoặc D1 cần nguồn — PM/BA chốt | [ảnh](evidence-gate-p10/x-b2-timhieu.png) |
| x.B3 | Còn giờ B3: SV B bấm «Đã rõ» | **P** | – | Dưới câu trả lời GV có «Hỏi tiếp» / «Đã rõ»; bấm «Đã rõ» → «Đã đóng · Hoàn tác» tại chỗ (câu hỏi đóng), không hộp thoại/thông báo nổi | [ảnh](evidence-gate-p10/x-b3-darou.png) |
| x.B4a | Còn giờ B4: SV nhập mã ngắn khi không quét được | **P** | – | /checkin: ô «Mã điểm danh» + nút «Điểm danh»; nhập mã ngắn từ màn chiếu (VD «3TC 1MV») → «Đã điểm danh 03:09»; dòng ở /attendance có chip «Nhập mã» ✔ | [ảnh](evidence-gate-p10/h4-short-code.png) |
| x.B4b | Còn giờ B4: GV (375) tắt mạng → chạm một ô → bật mạng | **P** | – | Offline: bấm «Ghi phát biểu · Đỗ Bảo Hải» → dòng hiện ngay «+1 · +0,25×1», DB participation_events vẫn 67 (chưa gửi); bật mạng, sau ≤ 9 s DB = 68 và UI giữ nguyên → xếp hàng rồi tự đồng bộ, không mất ✔ | [ảnh](evidence-gate-p10/x-b4-online2.png) |
| x.B6 | Còn giờ B6: Xuất XLSX sổ điểm lớp 1 | **P** | – | Menu «Thêm hành động» → «Xuất XLSX» tải so-diem.xlsx (8,4 KB): tiêu đề «An ninh mạng · 761987 · 25/11/2026 03:04 · Bản xem trước — chưa chốt», cột MSSV/Họ tên/Giữa kỳ/Kiểm tra hằng tuần/Thi cuối kỳ/Điểm cộng/Điểm trừ/Điểm quá trình/Điểm thi cuối kỳ/Điểm học phần/Điểm chữ/Ghi chú, 33 dòng, và dòng «Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.» | – |
| x.B7 | Còn giờ B7: xuất Markdown báo cáo lỗ hổng | **P** | – | Menu «⋯» → «Xuất Markdown» tải lo-hong-kien-thuc.md (8,7 KB): tiêu đề «Lỗ hổng kiến thức — 120 ngày gần nhất», «Tạo lúc 24/11/2026 19:30 · 162 câu hỏi», danh sách chủ đề + số SV/câu; không có tên hay MSSV (kiểm regex: không). LỆCH nhỏ: giờ «Tạo lúc 19:30» là UTC trong khi màn báo «25/11/2026, 02:30» (giờ VN) — xuất ra lệch 7 giờ (Thấp). Chủ đề là tên từ khoá của báo cáo giả | – |
| x.B7b | Còn giờ B7: /insights đổi lớp 761988 → TCP/UDP (C) đứng đầu | **N** | – | THIẾU DỮ LIỆU SEED: 761988 không có câu hỏi chat nào (0 tin nhắn người dùng) → «Chưa đủ câu hỏi để gom (cần ≥ 20)» ở mọi khoảng; không có chủ đề C để so với 761987 (dữ liệu hai lớp không lẫn: chưa kiểm được) | [ảnh](evidence-gate-p10/prep4-insights-761988-after.png) |

## 5. Phát hiện cần xử lý (xếp mức đề xuất — PM chốt)

| # | Mức đề xuất | Mục | Mô tả | Loại |
| --- | --- | --- | --- | --- |
| G1 | **Cao (cho demo)** | 7.1, 7.2 | Báo cáo lỗ hổng với embedding giả: 152 chủ đề, 142 chủ đề 1 SV + 10 chủ đề 2 SV, mọi chủ đề «Ít người hỏi», nhãn «Quy chế học vụ», tên là từ khoá («huong, man, middle»…), KHÔNG có chủ đề A (mã độc/tấn công) / B đứng đầu, không có câu mẫu. Mặc định «14 ngày» luôn «Chưa đủ câu hỏi» vì dữ liệu seed già. | Kịch bản `[CHƯA KIỂM: embedding giả]` nay xác nhận ĐỎ; cần embedding thật hoặc đổi kịch bản/ngưỡng; dữ liệu seed cần câu hỏi mới nhất |
| G2 | Trung bình | 2.1 | Khối «Kết quả» dưới câu trả lời D1 hiện khoá tiếng Anh + JSON thô (`late`, `absent`, `recent_absent {"date":…}`, `speak_count`, `points_total`) cho sinh viên | Giao diện (DESIGN §13) |
| G3 | Trung bình | 4.2 | `/attendance/display` ở 1440×900: mã ngắn (126 px) tràn khỏi khung (right 1485 > 1440, bị cắt); ở 1920×1080 vừa. Mã nằm bên phải QR, không «ngay dưới QR» | Giao diện (máy chiếu) |
| G4 | Thấp–Trung bình | 3.5 | SV đang ở `/chat` bấm thông báo «Giảng viên đã trả lời» → URL đổi `?session=…` nhưng khung chat trống; chỉ hiện câu trả lời sau khi tải lại (mở từ `/` thì đúng) | Lỗi giao diện |
| G5 | Thấp | 3.3 | Sau «Nhận» chỉ ghi «Đã nhận», không thấy TÊN người nhận | Kịch bản/UI |
| G6 | Thấp | H4b | SV không thuộc lớp quét QR → «Bạn không thuộc lớp này.» (kịch bản: không lộ lý do) | Chữ |
| G7 | Thấp | 8.1, 8.2 | Tên thực thể: kịch bản ghi `assignments (publish)` / `grade_entries (set)`; thực tế `grading_results · grading_result.publish` và `grade_schemes · extract/edit/confirm`; lọc `grade_entries` → «Không có» (cả chuẩn bị #10 sai) | Kịch bản |
| G8 | Thấp | x.B2 | D1 là câu tra sổ nên không có «Nguồn tham khảo» (chỉ có «Tìm hiểu») | Kịch bản |
| G9 | Thấp | x.B7 | Xuất Markdown báo cáo ghi «Tạo lúc 24/11/2026 19:30» (UTC) trong khi màn báo «25/11/2026, 02:30» (giờ VN) — lệch 7 giờ | Lỗi nhỏ |
| G10 | Thấp | 6.2 | Lần tải quy chế đầu hỏng (docling chưa bật) để lại tài liệu «Sẵn sàng» nhưng KHÔNG có bản nháp công thức; tải lại bị chặn «Tệp này đã có trong lớp.» → phải xoá tài liệu mới đi tiếp | Đường phục hồi |
| G11 | Thấp | 5.1 | Đề bài BT03/BT04 hiện markdown thô («# Bài tập 03…», «## Đề bài»); Bài tập 04 là bản sao nội dung BT03 | Giao diện + seed |
| G12 | Thấp | H7 | Bấm «Tạo báo cáo mới» lặp (không có dữ liệu mới) im lặng, không báo «đã có báo cáo mới nhất» | UX |
| G13 | Lưu ý | – | Chế độ «Bản mô phỏng · dữ liệu giả»: tên trên đầu trang (vd. `sv.kha` hiện «Trần Thu Uyên», `sv.moi` «Phạm Ngọc Linh», admin «Đỗ Hoàng Nam») KHÁC tên thật trong DB/danh sách lớp (Bùi Thanh Khải, Lý Thu Minh…). D1/D2 phải gõ tên thật theo seed («Bùi Thanh Khải» / «Ngô Ngọc Cẩm», MSSV 20229002); người trình bày đọc tên từ đầu trang sẽ gõ sai | Kịch bản/seed |

**Thiếu dữ liệu seed (không tính lỗi sản phẩm):** (a) không có buổi học ngày hôm nay ở 761987 (4.1); (b) 761988 không có câu hỏi chat (7.1, x.B7b); (c) Bài tập 03 chưa gắn cột điểm nên điểm không vào sổ (6.5); (d) công tắc «Điểm danh bằng mã» tắt (chuẩn bị #5); (e) chưa chấm nháp BT03 (chuẩn bị #3); (f) câu hỏi seed già hơn 14 ngày; (g) một số thread Hỏi đáp công khai bằng tiếng Anh lẫn (vd. «Seasonality without Trend») trong lớp An ninh mạng.

**Không chạy được (N), lý do:** camera quét QR thật; «tạo lại mã → mã cũ bị từ chối» (đổi mã demo AN7K2MQ/BX4P9TW); hai người cùng «Nhận» ticket (chỉ có 1 giảng viên); mail lỗi thử lại 3 lần; worker chết giữa chừng; mất mạng giữa lúc AI đang trả lời (LLM giả trả < 1 s); mọi dịch vụ AI hỏng → trích tài liệu; hai người sửa cùng ô sổ điểm (409) và mở khoá sau chốt; 3 báo cáo/giờ và hai người cùng bấm tạo báo cáo; video dự phòng; Bước 6 `Chốt điểm` lớp 1 (kịch bản không làm trong 15 phút).

## 6. Dữ liệu QC đã TẠO / ĐỔI trên demo (PM quyết giữ / dọn)

| # | Dữ liệu | Chi tiết |
| --- | --- | --- |
| D1 | Lớp 761988 — thành viên | `sv.moi` (Lý Thu Minh) gửi yêu cầu `/join/BX4P9TW` rồi được GV duyệt → thành viên ACTIVE (24 → 25); còn 3 yêu cầu seed (sv52–54) chờ duyệt |
| D2 | Chat riêng 761987 của `sv.kha` | 1 phiên mới (`01a20f0c-6c77-…`) gồm D1, D2, D3; phản hồi «Hữu ích» cho D1; ticket D3 (Bùi Thanh Khải) ĐÃ TRẢ LỜI (D4) rồi SV bấm «Đã rõ» → đã đóng; thông báo + 2 thư Mailpit tới `sv.kha` («Giảng viên đã trả lời…», «Điểm bài tập đã được công bố») |
| D3 | Cài đặt lớp 761987 | công tắc «Điểm danh bằng mã» đã BẬT (trước đó tắt) |
| D4 | Điểm danh 761987 Buổi 11 (19/11) | 2 cửa sổ điểm danh (vòng 1, vòng 2); 5 lượt check-in (`sv.kha`, `sv.nguyco`, `sv.gioi` quét QR; `sv04` nhập mã ngắn; `sv05` quét QR cùng máy → cờ «Cùng máy»); 27 SV bị ĐÁNH VẮNG, 3 «Muộn»; Buổi 11 → «Đã điểm danh» (trước: «Chưa»); vòng 2 gắn «Vắng ở lượt kiểm lại» cho sv.kha, sv.gioi |
| D5 | Điểm cộng phát biểu +0,25 | `sv.kha` (+0,25 → điểm cộng 0,75), `sv23` (Bùi Minh Yến) và `sv06` (Đỗ Bảo Hải) mỗi người +0,25 (kiểm nhánh mất mạng); 3 dòng `participation_events` |
| D6 | Bài tập 03 (761987) | «Chấm tất cả» → 22 kết quả chấm nháp (DRAFT 21 + 1 PUBLISHED); bài `sv.kha` đã SỬA (điểm tiêu chí «Phân tích tấn công» 1,00 → 2,00, nhận xét), DUYỆT và CÔNG BỐ: 7,50/10 (SV thấy điểm) |
| D7 | Báo cáo lỗ hổng 761987 | 2 bản ghi `insight_reports` (19:30:50 và 20:14:25 UTC, «Toàn kỳ»); thông báo «Báo cáo lỗ hổng kiến thức đã sẵn sàng»; 1 thread ghim «Chủ đề đang vướng: huong, man, middle» ở Hỏi đáp công khai |
| D8 | Lớp 761988 — công thức điểm | tài liệu `quy-che-lop2.pdf` (đã xoá bản lỗi, tải lại bản mới, ĐÃ XÁC NHẬN công thức, «Làm tròn đến 0,1») — banner «chưa xác nhận» của lớp 2 đã biến mất, «Chốt điểm» đã mở; nhật ký: `document.delete`, `grade_schemes extract/edit/confirm` |
| D9 | Khoá thử | `sv.nguyco` bị khoá tham gia lớp 10 phút (bài thử mã sai; hết tự động) |
| D10 | Phiên/thông báo/tệp | nhiều phiên đăng nhập của `teacher`, `admin`, `sv.kha`, `sv.moi`, `sv.nguyco`, `sv.gioi`, `sv04`, `sv05`, `sv.chuaxm`; tệp tải xuống (so-diem.xlsx, lo-hong-kien-thuc.md) chỉ ở máy QC; 1 lần «Xuất XLSX» (audit `gradebook export`) |
| D11 | Hạ tầng (PM làm giữa lượt) | gateway/worker khởi động lại với `BLOB_PUBLIC_ENDPOINT=localhost`, `BLOB_PUBLIC_USE_SSL=true`; container `edupilot-docling-1` được bật |

Không tạo / không đổi: mã tham gia lớp, mật khẩu, tài khoản, bài nộp của SV (BT04 thử nộp bị chặn nên không tạo), điểm sổ điểm lớp 1 (chưa «Chốt điểm»).

Để khôi phục nhanh trạng thái trước demo (nếu PM muốn): bản sao lưu `~/Documents/EduPilot-demo-backups/` (QC không đụng). Các thay đổi D3, D4, D6, D8 là phần có tác động thấy được khi chủ dự án chạy lại kịch bản (Bước 4 «Chưa điểm danh» và Bước 5/6 «chưa chấm / chưa xác nhận» đã được làm sẵn).

## 7. Nhật ký chạy lại (DEMO_SCRIPT.md mục 7) — dòng đề xuất

| Ngày | Phase vừa xong | Bước xa nhất chạy thật | Tổng thời gian | Tầng | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| 2026-11-25 | Gate P10 (main 23c97510) | Bước 8 (đủ 8 bước) | không đo được trọn kịch bản (xem mục 2); chuẩn bị #3 ≈ 12–13 phút | Tầng 2 (LLM giả) | Bước 7 đỏ với embedding giả; 4 phát hiện giao diện; 7 điều kiện seed chưa đủ (mục 5) |
