# Kịch bản demo bảo vệ — 15 phút

Phiên bản 1 · 2026-10-01 · Nguồn: P0 lát L0, `FLOWS.md` F2 → F3 → F5 → F7 → F9 → F10 → F17, seed ở `ARCHITECTURE.md` mục 9 · Spec: `docs/specs/FEAT-demo-script/`

Kịch bản này là kim chỉ nam cho mọi phase: mỗi phase sau phải làm cho ít nhất một bước dưới đây chạy thật. Cuối mỗi phase, chạy lại kịch bản tới bước xa nhất có thể và ghi vào mục 7. Câu hỏi còn mở: `docs/specs/FEAT-demo-script/QUESTIONS.md` (Q1–Q9); chỗ nào trong kịch bản phụ thuộc một câu hỏi thì có ghi `(xem Qn)`.

Mạch truyện: một tuần của **Sinh viên B** (lớp 1) và giảng viên phụ trách hai lớp. Sinh viên B hỏi điểm riêng tư → hỏi một câu tài liệu không có → giảng viên trả lời → được ghi phát biểu → bài tập được chấm và công bố → điểm vào sổ. Lớp 2 dùng để demo vào lớp, công thức điểm chưa xác nhận và cách ly dữ liệu giữa hai lớp.

## 1. Thời lượng

| Phần | Luồng | Bắt đầu | Thời lượng |
| --- | --- | --- | --- |
| Mở đầu | – | 00:00 | 0:30 |
| Bước 1. Mở lớp → vào lớp | F2 | 00:30 | 1:45 |
| Bước 2. Hỏi riêng tư | F3 | 02:15 | 2:00 |
| Bước 3. Cần giảng viên hỗ trợ | F5 | 04:15 | 2:15 |
| Bước 4. Điểm danh và phát biểu | F7 | 06:30 | 1:30 |
| Bước 5. Duyệt và công bố bài tập | F9 | 08:00 | 2:00 |
| Bước 6. Công thức điểm và sổ điểm | F10 | 10:00 | 2:00 |
| Bước 7. Báo cáo lỗ hổng kiến thức | F17 | 12:00 | 1:15 |
| Kết | – | 13:15 | 0:30 |
| Hết kịch bản; 1:15 còn lại là dự trữ | – | 13:45 | – |

Quy tắc giữ giờ: trễ quá 30 s so với mốc của một bước thì bỏ các hàng ghi "(tuỳ chọn)" của bước kế tiếp. Không cắt bước.

## 2. Chuẩn bị (T−30 phút, không tính vào 15 phút)

| # | Việc | Lệnh / nơi làm | Kiểm |
| --- | --- | --- | --- |
| 1 | Dựng stack trên DB trống, seed tự chạy | `pnpm dev` (terminal riêng), `.env.local` có `SEED_ON_EMPTY_DB=true` | `pnpm dev:status`: mọi service chạy; `http://localhost:8025` mở được |
| 2 | Chọn tầng chạy: LLM thật hay `DEMO_MODE=true` (mục 5) | `.env.local` | Gửi câu D1 thử một lần, xem có trả lời |
| 3 | Tạo kết quả chấm nháp cho Bài tập 03 lớp 1 (chưa công bố) | Giảng viên chạy "Chấm tất cả" cho Bài tập 03 ở `/grading` | Bài của Sinh viên B ở trạng thái nháp, có cờ "cần xem kỹ" |
| 4 | Tạo sẵn báo cáo lỗ hổng kiến thức của lớp 2 | Giảng viên, lớp 2, `/insights` → `Tạo báo cáo mới` | Có báo cáo, chủ đề C đứng đầu |
| 5 | Có buổi học lớp 1 "đang diễn ra" vào giờ demo | Seed (xem Q1) | "Hôm nay" của giảng viên có việc điểm danh lớp 761987 |
| 6 | Cửa sổ trình duyệt desktop: Admin (hồ sơ trình duyệt 1), Giảng viên (hồ sơ 2) | Đăng nhập sẵn | – |
| 7 | Điện thoại thật hoặc giả lập 375 px: Sinh viên D, Sinh viên B, Giảng viên (cho điểm danh) | Đăng nhập sẵn, mỗi tài khoản một hồ sơ trình duyệt | – |
| 8 | Tab Mailpit `http://localhost:8025`, hộp thư trống | Xoá thư cũ trong Mailpit | Không còn thư |
| 9 | File quy chế lớp 2 trên máy demo | `seed/demo/quy-che-lop2.pdf` (xem mục 6) | File mở được |
| 10 | Video dự phòng sẵn trên laptop và USB | mục 5.3 | Phát thử 5 s |

Tài khoản seed (mật khẩu `SEED_DEFAULT_PASSWORD`): `admin@edupilot.local`, `teacher@edupilot.local`, `sv.kha@edupilot.local` (Sinh viên B, lớp 1), `sv.moi@edupilot.local` (Sinh viên D, chưa vào lớp nào), `sv.nguyco@edupilot.local` (Sinh viên C, lớp 1). Lớp 1 mã lớp `761987`, mã tham gia `AN7K2MQ`; lớp 2 mã lớp `761988`, mã tham gia `BX4P9TW`, bật yêu cầu duyệt.

## 3. Câu nhập nguyên văn

Người trình bày gõ (hoặc dán) đúng các chuỗi này. Chế độ `DEMO_MODE` chỉ có câu trả lời ghi sẵn cho đúng các chuỗi này. `{…}` là giá trị seed cố định (xem Q2).

| Mã | Ai gõ | Ở đâu | Nội dung |
| --- | --- | --- | --- |
| D1 | Sinh viên B | `/chat`, lớp 1 | `Em là {HO_TEN_B}, MSSV {MSSV_B}. Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?` |
| D2 | Sinh viên B | `/chat`, lớp 1 | `Bạn {HO_TEN_C} nghỉ mấy buổi rồi ạ?` |
| D3 | Sinh viên B | `/chat`, lớp 1 | `Thi cuối kỳ có được mang một tờ A4 ghi chú viết tay vào phòng thi không ạ?` |
| D4 | Giảng viên | `/inbox` | `Được mang một tờ A4 viết tay, hai mặt, không dùng bản photo. Thầy sẽ nhắc lại trên lớp.` |
| D5 | Giảng viên | `/gradebook/scheme`, lớp 2 | Tải `seed/demo/quy-che-lop2.pdf`; mục chưa rõ "làm tròn" điền: `Làm tròn đến 0,1` |

## 4. Kịch bản

### Mở đầu (00:00–00:30)

> "EduPilot là nền tảng vận hành lớp học có AI cho một học phần. Dữ liệu hôm nay là dữ liệu mô phỏng: một giảng viên, hai lớp An ninh mạng, mỗi lớp 30 sinh viên. Lớp 1 đang ở tuần 10; lớp 2 vừa được phân công."

### Bước 1. Mở lớp → vào lớp — F2 (00:30–02:15)

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 00:30 | Admin | `/admin/courses` | Mở danh sách lớp, chỉ vào lớp 2 | Hai lớp An ninh mạng 761987 và 761988, cả hai đã gán giảng viên | P2 |
| 00:50 | Giảng viên | `/` (Hôm nay) | Mở chuông thông báo | "Bạn được phân công lớp An ninh mạng – 761988" kèm mã tham gia `BX4P9TW`; danh sách việc có "Thiết lập lớp mới" ghi tên lớp 2 | P2 (khung PU) |
| 01:10 | Sinh viên D (375 px) | `/join/BX4P9TW` | Mở link → đọc phần xem trước → `Tham gia lớp` | Xem trước: tên lớp, mã lớp 761988, giảng viên, học kỳ. Sau khi bấm: báo đang chờ giảng viên duyệt (lớp 2 bật yêu cầu duyệt) | P2 |
| 01:35 | Giảng viên | `/class/members` (bộ chọn lớp → "Quản lý lớp này") | Duyệt yêu cầu của Sinh viên D | Sinh viên D chuyển sang thành viên; 3 yêu cầu seed khác vẫn chờ | P2 |
| 01:55 | Sinh viên D (375 px) | `/` | Tải lại trang | Bộ chọn lớp có lớp 761988; không còn ô nhập mã tham gia | P2 |

Nếu hội đồng hỏi (ngoài 15 phút): nhập sai mã → cùng một thông báo cho mọi trường hợp, khoá sau 5 lần / 10 phút; tạo lại mã → mã cũ bị từ chối. E2E: `class-join.spec.ts`.

### Bước 2. Hỏi riêng tư — F3 (02:15–04:15)

> "Sinh viên hỏi điểm của chính mình. Tên và MSSV được ẩn trước khi câu hỏi rời hệ thống; danh tính lấy từ phiên đăng nhập, không từ nội dung câu hỏi."

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 02:15 | Sinh viên B (375 px) | `/chat`, lớp 1 | Gõ D1, gửi | Dòng "Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI"; câu trả lời hiện dần; khối gọn nêu số buổi vắng (2) và điểm cộng phát biểu đúng số trong sổ; không thấy mã thay thế kiểu `[[SV_1]]` | P3 (số liệu: P5, P6) |
| 03:05 | Sinh viên B (375 px) | `/chat` | Bấm `Hữu ích` | Nút đổi trạng thái tại chỗ, không có thông báo nổi | P3 |
| 03:15 | Sinh viên B (375 px) | `/chat` | Gõ D2, gửi | Từ chối lịch sự: chỉ trả lời được thông tin của chính bạn; không có con số nào về Sinh viên C | P3 |
| 03:45 | Sinh viên B (375 px) | `/chat` | (tuỳ chọn) Mở `Nguồn tham khảo` dưới câu trả lời D1 | Nguồn mở ngay dưới câu trả lời | P3 |

Nếu hội đồng hỏi: mất mạng giữa lúc đang trả lời → tải lại vẫn thấy phần đã hiện; mọi dịch vụ AI hỏng → câu trả lời trích từ tài liệu kèm nhãn. E2E: `private-chat.spec.ts`.

### Bước 3. Cần giảng viên hỗ trợ — F5 (04:15–06:30)

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 04:15 | Sinh viên B (375 px) | `/chat`, lớp 1 | Gõ D3, gửi | "AI chưa đủ chắc chắn về câu này"; câu hỏi được chuyển cho giảng viên, trạng thái "Đang chờ giảng viên · vừa gửi" (xem Q3 về việc có cần bấm `Nhờ giảng viên hỗ trợ`) | P4 (chat: P3) |
| 04:45 | Giảng viên | `/` → `/inbox` | Mở việc "Cần xử lý" mới nhất | Ticket ở lọc Open: câu hỏi gốc, thời gian chờ, lý do (độ tin cậy thấp) | P4 |
| 05:00 | Giảng viên | `/inbox` | `Nhận` | Trạng thái Claimed, ghi tên người nhận | P4 |
| 05:10 | Giảng viên | `/inbox` | Gõ D4 → `Gửi trả lời` | Ticket chuyển Answered | P4 |
| 05:35 | Sinh viên B (375 px) | `/chat` | Mở chuông | Câu trả lời của giảng viên nằm ngay trong phiên chat, có nhãn người trả lời là giảng viên | P4 |
| 05:55 | Người trình bày | `http://localhost:8025` (Mailpit, công cụ dev, không phải màn sản phẩm) | Mở thư mới nhất | Thư tới `sv.kha@edupilot.local` chứa câu trả lời và link về app | P4 (Mailpit: P0 L1) |
| 06:10 | Sinh viên B (375 px) | `/chat` | Xác nhận đã rõ | Câu hỏi được đóng | P4 |

Nếu hội đồng hỏi: hai người cùng `Nhận` → người sau thấy ai đã nhận (409); mail lỗi → thử lại 3 lần, trả lời trong app không ảnh hưởng; không ai nhận 72 giờ → nổi lên đầu "Hôm nay". E2E: `escalation.spec.ts`.

### Bước 4. Điểm danh và phát biểu — F7 (06:30–08:00)

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 06:30 | Giảng viên (375 px) | `/` | Chạm việc "Điểm danh buổi đang diễn ra – 761987" | Mở `/attendance` đúng buổi hôm nay (xem Q1) | P5 (Hôm nay: P2) |
| 06:40 | Giảng viên (375 px) | `/attendance` | Mọi người mặc định có mặt; chạm `Vắng` cho 2 sinh viên, `Muộn` cho 1; ghi phát biểu +0,25 cho Sinh viên B ngay trên dòng | Trạng thái lưu hiển thị; toàn bộ thao tác dưới 60 s | P5 |
| 07:10 | Giảng viên (375 px) | `/attendance` | `Lưu điểm danh` | Xác nhận đã lưu tại chỗ, không có thông báo nổi "Thành công" | P5 |
| 07:25 | Sinh viên B (375 px) | `/me` | Mở | Phần chuyên cần và điểm cộng có thêm +0,25 vừa ghi; giải trình điểm dạng phép tính | P5 (giải trình: P6) |
| 07:45 | Giảng viên (375 px) | `/attendance` | (tuỳ chọn) Tắt mạng điện thoại → chạm một ô → bật mạng | Thay đổi xếp hàng rồi tự đồng bộ, không mất | P5 |

Nếu hội đồng hỏi: hai người cùng điểm danh → khoá lạc quan; sửa sau buổi học → ghi audit. E2E: `attendance.spec.ts`.

### Bước 5. Duyệt và công bố bài tập — F9 (08:00–10:00)

> Bài đã được nộp và chấm nháp ở bước chuẩn bị #3. Phần tạo bài và nộp bài có trong video phụ và E2E (xem Q4).

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 08:00 | Sinh viên B (375 px) | `/assignments/[id]` (Bài tập 03, mở từ `/me`) | Mở | Đã nộp, có dấu thời gian, nhãn nộp muộn; phần điểm ghi đang chấm — **không có điểm nháp** | P7 |
| 08:15 | Giảng viên | `/grading` (tab Hàng chờ chấm) | Giữ lọc mặc định `Cần xem kỹ` + `Chưa duyệt` | Bài của Sinh viên B ở đầu, lý do cờ: hai lượt chấm lệch hơn 1 điểm | P7 |
| 08:35 | Giảng viên | `/grading/[submissionId]` | Đọc bài bên trái, tiêu chí bên phải; sửa điểm tiêu chí bị lệch; sửa một câu nhận xét | Thông báo vàng tại tiêu chí lệch; mỗi tiêu chí có đoạn trích từ bài làm; tổng điểm tính lại ngay | P7 |
| 09:05 | Giảng viên | `/grading/[submissionId]` | `Duyệt bài` | Trạng thái đã duyệt | P7 |
| 09:15 | Giảng viên | `/grading` | Chọn bài vừa duyệt → `Công bố` | Trạng thái đã công bố; chỉ TEACHER thấy hành động này | P7 |
| 09:35 | Sinh viên B (375 px) | `/assignments/[id]` | Tải lại | Điểm, nhận xét theo từng tiêu chí, đoạn trích từ bài của chính mình làm bằng chứng | P7 |

Nếu hội đồng hỏi: nộp sau hạn khi bài không cho nộp muộn → bị chặn; worker chết giữa chừng → việc được nhận lại, không chấm đôi. E2E: `assignment-lifecycle.spec.ts`.

### Bước 6. Công thức điểm và sổ điểm — F10 (10:00–12:00)

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 10:00 | Giảng viên | `/gradebook`, lớp 2 | Đổi lớp ở bộ chọn lớp | Banner cố định "công thức điểm chưa xác nhận"; `Chốt điểm` bị khoá, lý do hiện khi rê chuột / focus | P6 |
| 10:15 | Giảng viên | `/gradebook/scheme`, lớp 2 | Tải `seed/demo/quy-che-lop2.pdf` | Tiến độ xử lý nền; bản nháp: thành phần, trọng số, điểm cộng, mỗi mục cạnh đoạn trích có số trang; mục "chưa rõ: quy tắc làm tròn" chặn nút xác nhận | P6 |
| 10:55 | Giảng viên | `/gradebook/scheme` | Điền D5 → `Xác nhận công thức` → xác nhận trong hộp thoại nêu hậu quả | Trạng thái đã xác nhận | P6 |
| 11:15 | Giảng viên | `/gradebook`, lớp 2 | Quay lại | Banner biến mất; `Chốt điểm` mở | P6 |
| 11:25 | Giảng viên | `/gradebook`, lớp 1 | Đổi lớp, mở giải trình dòng Sinh viên B | Thành phần điểm có +0,25 từ bước 4 và điểm Bài tập 03 từ bước 5; dòng ghi chú "điểm chính thức nằm ở hệ thống quản lý đào tạo của trường" | P6 (dữ liệu: P5, P7) |
| 11:45 | Giảng viên | `/gradebook`, lớp 1 | Xuất XLSX (menu) | File tải về có cột theo mẫu và dòng ghi chú điểm chính thức | P6 |

`Chốt điểm` lớp 1 không làm trực tiếp trong 15 phút (xem Q5). Nếu hội đồng hỏi: hai người sửa cùng ô → 409 và chọn giữ bên nào; mở khoá sau chốt cần lý do. E2E: `gradebook.spec.ts`.

### Bước 7. Báo cáo lỗ hổng kiến thức — F17 (12:00–13:15)

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 12:00 | Giảng viên | `/insights`, lớp 1 | `Tạo báo cáo mới` | Tiến độ xử lý nền; danh sách chủ đề xếp hạng, chủ đề A và B đứng đầu, mỗi chủ đề có lý do và tín hiệu | P10 |
| 12:35 | Giảng viên | `/insights` | Mở câu mẫu của chủ đề A | 3–5 câu đã ẩn danh, không có tên hay MSSV; chủ đề dưới 3 sinh viên hỏi không có câu mẫu | P10 |
| 12:50 | Giảng viên | `/insights` | `Tạo thread ghim` | Thread ghim mới xuất hiện ở `/threads` của lớp 1 | P10 (Threads: P4) |
| 13:05 | Giảng viên | `/insights`, lớp 2 | Đổi lớp (báo cáo đã tạo ở chuẩn bị #4) | Chủ đề C đứng đầu — dữ liệu hai lớp không lẫn | P10 |

E2E / test: `pytest -q -k "knowledge_gap"` (P10).

### Kết (13:15–13:45)

> "Bảy luồng vừa xem đi qua cùng một dữ liệu: câu hỏi của sinh viên, câu trả lời của giảng viên, điểm danh, bài chấm và công thức điểm đều đổ về cùng một sổ điểm. Mọi quyết định về điểm do giảng viên bấm; AI chỉ soạn nháp."

## 5. Dự phòng cho ngày bảo vệ

Ba tầng, đi từ trên xuống. Người trình bày (chủ dự án) quyết định chuyển tầng.

| Tầng | Khi nào dùng | Ai bật | Bật ở đâu, thế nào | Mất gì |
| --- | --- | --- | --- | --- |
| 1. Chạy thật | Mặc định (xem Q6) | – | `.env.local` không đặt `DEMO_MODE` | – |
| 2. `DEMO_MODE=true` | Mạng hoặc API LLM lỗi lúc chuẩn bị, hoặc một bước lỗi 2 lần liên tiếp trong lúc demo | Chủ dự án | Sửa `.env.local`: `DEMO_MODE=true` → `docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot up -d gateway` (≈ 30 s, nói lời dẫn trong lúc chờ) | Câu trả lời là bản ghi sẵn; chỉ đúng với các câu ở mục 3 |
| 3. Video quay sẵn | Stack không lên, hoặc tầng 2 cũng lỗi | Chủ dự án | Mở file video trên laptop (bản sao trên USB), nhảy tới chương của bước đang dở | Không tương tác được |

### 5.1 `DEMO_MODE=true` phải làm gì

- Lớp gọi LLM trong gateway Go (`openai-go`, D46) dùng provider `fake` (P1 L1) cho mọi tác vụ trong kịch bản: CHAT, CLASSIFY, UTILITY, GRADING, INSIGHT, EMBEDDING. Worker Go dùng cùng cờ cho các việc nền.
- Provider `fake` trả câu trả lời ghi sẵn cho đúng các đầu vào ở mục 3 và các việc nền của bước 5, 6, 7 (chấm Bài tập 03, trích quy chế lớp 2, báo cáo lỗ hổng hai lớp). Đầu vào khác → câu trả lời trung tính "chưa có câu trả lời ghi sẵn", không lỗi.
- Đường đi còn nguyên: che tên / MSSV, Scheduler, `llm_audit`, ngưỡng độ tin cậy, escalation, mail. Chỉ lời gọi ra provider bị thay. Vì vậy dòng "Đã ẩn 2 thông tin cá nhân" và ticket ở bước 3 vẫn sinh thật.
- Không cần mạng ra ngoài. Kiểm: tắt Wi-Fi máy demo, chạy toàn kịch bản, mọi bước qua.
- Vị trí và định dạng bản ghi sẵn: đề xuất `seed/demo/fake_llm.json` (xem Q7).

### 5.2 Kiểm tầng 2 trước ngày bảo vệ

Chạy kịch bản trọn vẹn hai lần với `DEMO_MODE=true`, một lần ngắt mạng hoàn toàn. Ghi kết quả vào mục 7.

### 5.3 Video

- Quay sau khi xong P10, theo đúng kịch bản này, có chương theo 7 bước, 1080p, tiếng thuyết minh.
- Thêm một video phụ ≤ 3 phút: tạo bài tập và sinh viên nộp bài (F9 bước 1–3), chốt điểm lớp 1 (F10 bước 6) — phần không chạy trực tiếp trong 15 phút.
- Hai bản: ổ laptop demo và USB. Phát được không cần mạng.

## 6. Yêu cầu kịch bản đặt lên các phase sau

| Phase | Phải có để kịch bản chạy | Bước |
| --- | --- | --- |
| P1 | Provider `fake` phát lại bản ghi sẵn theo đầu vào; cờ `DEMO_MODE` | 5.1 |
| P2 | Seed cố định: họ tên, MSSV của Sinh viên B, C (Q2); mã tham gia `AN7K2MQ` / `BX4P9TW`; lớp 2 bật yêu cầu duyệt; chuông phân công chưa đọc | 1, 2 |
| P3 | Dòng "Đã ẩn N thông tin cá nhân"; từ chối hỏi về người khác | 2 |
| P4 | Câu D3 thuộc nhóm "câu ngoài tài liệu" của seed (không tài liệu nào trả lời được); mail qua Mailpit | 3 |
| P5 | Một buổi học lớp 1 trùng giờ demo (Q1) | 4 |
| P6 | File `seed/demo/quy-che-lop2.pdf` cố ý thiếu quy tắc làm tròn | 6 |
| P7 | Bài tập 03 lớp 1 có bài nộp muộn của Sinh viên B; hai lượt chấm của bài đó lệch > 1 điểm (bản ghi sẵn khi `DEMO_MODE`) | 5 |
| P10 | Câu hỏi seed lệch chủ đề A, B (lớp 1) và C (lớp 2); video | 7, 5.3 |

## 7. Nhật ký chạy lại

Cuối mỗi phase: chạy từ đầu tới bước xa nhất chạy được, bấm giờ, ghi một dòng.

| Ngày | Phase vừa xong | Bước xa nhất chạy thật | Tổng thời gian | Tầng | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| 2026-10-01 | P0 (đang làm) | – (chưa có bước nào chạy) | – | – | Bản kịch bản đầu tiên |
