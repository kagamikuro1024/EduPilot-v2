# Kịch bản demo bảo vệ — 15 phút

Phiên bản 2 · 2026-11-15 · Luồng: `FLOWS.md` F2 → F3 → F5 → F7 (điểm danh bằng mã QR, sprint 10.8) → F9 → F10 → F17 (`/insights` thật) → nhật ký quản trị (`/admin/audit`) · Lớp trình bày: **761987** · Seed: `README.md` mục 5, `ARCHITECTURE.md` mục 9

Câu hỏi còn mở: `docs/specs/FEAT-demo-script/QUESTIONS.md` (Q1–Q9). «Lớp 1» ở các bảng dưới = lớp 761987, «lớp 2» = 761988.

Mạch truyện: một tuần của **Sinh viên B** (lớp 761987) và giảng viên phụ trách hai lớp. Sinh viên B hỏi điểm riêng tư → hỏi một câu tài liệu không có → giảng viên trả lời → cả lớp điểm danh bằng QR, B được ghi phát biểu → bài tập được chấm và công bố → điểm vào sổ → giảng viên xem lỗ hổng kiến thức của lớp → Admin đối chiếu nhật ký. Lớp 761988 dùng để demo vào lớp, công thức điểm chưa xác nhận và cách ly dữ liệu giữa hai lớp.

**Trạng thái kiểm (2026-11-15):** mục 2, 5 đã đối chiếu với mã `sprint/11-p10-eval`; Bước 4 (QR) viết theo SRS `FEAT-attendance-selfcheck` v1.6 vì 10.8 chưa vào `main`; **chưa chạy trọn trên stack demo ở mã mới** (xem mục 7, chạy sau merge). Chỗ chưa kiểm ghi `[CHƯA KIỂM]`.


## 1. Thời lượng

| Phần | Luồng | Bắt đầu | Thời lượng |
| --- | --- | --- | --- |
| Mở đầu | – | 00:00 | 0:30 |
| Bước 1. Mở lớp → vào lớp | F2 | 00:30 | 1:30 |
| Bước 2. Hỏi riêng tư | F3 | 02:00 | 1:30 |
| Bước 3. Cần giảng viên hỗ trợ | F5 | 03:30 | 1:45 |
| Bước 4. Điểm danh bằng QR và phát biểu | F7 | 05:15 | 2:15 |
| Bước 5. Duyệt và công bố bài tập | F9 | 07:30 | 1:30 |
| Bước 6. Công thức điểm và sổ điểm | F10 | 09:00 | 1:30 |
| Bước 7. Báo cáo lỗ hổng kiến thức | F17 | 10:30 | 1:00 |
| Bước 8. Nhật ký quản trị | – | 11:30 | 0:30 |
| Kết | – | 12:00 | 0:15 |
| Hết kịch bản (12:15); dự trữ 2:45 trong khung 15 phút | – | 12:15 | – |

Giới hạn cứng: kịch bản chính **≤ 13:30**, dự trữ **≥ 1:30**. Quy tắc giữ giờ: trễ quá 30 s so với mốc của một bước thì bỏ hàng nào trễ nhất của bước đó, không cắt bước. Còn sớm hơn mốc thì thêm các hàng ở «Nếu còn giờ» (mục 4), tối đa tới 13:30. Nếu hội đồng cho 20 phút thì thêm phần «Nếu hội đồng hỏi» của từng bước.

## 2. Chuẩn bị (T−30 phút, không tính vào 15 phút)

| # | Việc | Lệnh / nơi làm | Kiểm |
| --- | --- | --- | --- |
| 1 | Dựng stack trên DB trống, seed tự chạy | `SEED_ON_EMPTY_DB=true COMPOSE_PROFILES=ingest pnpm dev` (terminal riêng; ≈ 10 phút, `README.md` 5.2) | `pnpm dev:status`: mọi service healthy; `https://localhost` đăng nhập được; `http://localhost:8025` mở được |
| 2 | Chọn tầng chạy: LLM thật (9router) hay bản ghi sẵn (mục 5) | `/settings/llm`; 9router bật: `curl -s http://127.0.0.1:20128/v1/models` | Gửi câu D1 thử một lần, xem có trả lời |
| 3 | **Làm ĐẦU TIÊN (trước #4–#11): bài chấm cần ≈ 12–13 phút** (đo 25/11: 22 bài, LLM giả, xếp hàng, worker chấm lần lượt; LLM thật có thể lâu hơn). Tạo kết quả chấm nháp cho Bài tập 03 lớp 761987 (chưa công bố); không thể làm trong lúc trình bày | Giảng viên chạy "Chấm tất cả" cho Bài tập 03 ở `/grading`; làm các việc #4–#11 trong lúc chờ | Đủ 22 bài có kết quả; bài của Sinh viên B ở trạng thái nháp, có cờ "cần xem kỹ" (trong hàng chờ, không nhất thiết ở đầu danh sách) |
| 4 | Có báo cáo lỗ hổng kiến thức ở cả hai lớp (không chờ nền trong lúc trình bày). **Tạo MỚI sau khi triển khai mã hiện tại**: báo cáo sinh bằng mã cũ trả 404 (P-11-34, `payload.schema` < 2) | Giảng viên, từng lớp, `/insights` → `Tạo báo cáo mới` | Lớp 761987: chủ đề về mã độc / tấn công (A) và quy chế học vụ (B) đứng đầu; lớp 761988: TCP / UDP (C) đứng đầu `[CHƯA KIỂM: gom cụm với embedding giả]` |
| 5 | Bật điểm danh bằng mã cho lớp 761987 và có một buổi học đang diễn ra vào giờ demo | Giảng viên, `/courses/{id}` → khối «Điểm danh bằng mã» bật; buổi học hôm nay từ seed (Q1) | "Hôm nay" của giảng viên có việc điểm danh lớp 761987; `/attendance` có nút `Mở điểm danh` `[CHƯA KIỂM: 10.8 chưa merge]` |
| 6 | Cửa sổ trình duyệt desktop: Admin (hồ sơ 1), Giảng viên (hồ sơ 2); máy chiếu thứ hai hoặc cửa sổ riêng cho `/attendance/display` | Đăng nhập sẵn | – |
| 7 | Điện thoại thật hoặc giả lập 375 px: Sinh viên D, Sinh viên B, Giảng viên (cho điểm danh); thêm 2 sinh viên khác để quét QR | Đăng nhập sẵn, mỗi tài khoản một hồ sơ trình duyệt | – |
| 8 | Tab Mailpit `http://localhost:8025`, hộp thư trống | Xoá thư cũ trong Mailpit | Không còn thư |
| 9 | File quy chế lớp 2 trên máy demo | `seed/demo/quy-che-lop2.pdf` (xem mục 6) | File mở được |
| 10 | Có sẵn vài dòng nhật ký để xem ở Bước 8 | Chạy thử Bước 5 và 6 một lần trước (công bố bài chấm tạo dòng `grading_results / grading_result.publish`; xác nhận công thức tạo `grade_schemes / confirm`; Bước 4 tạo `attendance`; P-11-112) | `/admin/audit` (24 giờ) có dòng |
| 11 | Video dự phòng sẵn trên laptop và USB | mục 5.3 | Phát thử 5 s |

Tài khoản seed (mật khẩu `SEED_DEFAULT_PASSWORD`, mặc định `Edupilot#Seed-2026`): `admin@edupilot.local`, `teacher@edupilot.local`, `sv.kha@edupilot.local` (Sinh viên B, lớp 761987), `sv.moi@edupilot.local` (Sinh viên D, chưa vào lớp nào), `sv.nguyco@edupilot.local` (Sinh viên C, lớp 761987). **Họ tên và MSSV để gõ D1, D2 (gõ đúng theo bảng này; KHÔNG đọc tên ở đầu trang: khi ứng dụng ở chế độ «Bản mô phỏng · dữ liệu giả» tên hiển thị khác tên trong dữ liệu):** `{HO_TEN_B}` = `Bùi Thanh Khải`, `{MSSV_B}` = `20229002` (`sv.kha`); `{HO_TEN_C}` = `Ngô Ngọc Cẩm` (`sv.nguyco`, MSSV `20229003`). Lớp 761987 mã tham gia `AN7K2MQ`; lớp 761988 mã `BX4P9TW`. Bộ chọn lớp của giảng viên mở sẵn ở **761987**; chỉ đổi sang 761988 ở các bước nói rõ.


## 3. Câu nhập nguyên văn

Người trình bày gõ (hoặc dán) đúng các chuỗi này. Tầng 2 (mục 5) không có câu trả lời ghi sẵn riêng cho các chuỗi này: dùng đúng chuỗi để kết quả lặp lại được. `{…}` là giá trị seed cố định (xem Q2).

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

### Bước 1. Mở lớp → vào lớp — F2 (00:30–02:00)

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 00:30 | Admin | `/admin/courses` | Mở danh sách lớp, chỉ vào lớp 2 | Hai lớp An ninh mạng 761987 và 761988, cả hai đã gán giảng viên | P2 |
| 00:50 | Giảng viên | `/` (Hôm nay) | Mở chuông thông báo | "Bạn được phân công lớp An ninh mạng – 761988" kèm mã tham gia `BX4P9TW`; danh sách việc có "Thiết lập lớp mới" ghi tên lớp 2 | P2 (khung PU) |
| 01:10 | Sinh viên D (375 px) | `/join/BX4P9TW` | Mở link → đọc phần xem trước → `Tham gia lớp` | Xem trước: tên lớp, mã lớp 761988, giảng viên, học kỳ. Sau khi bấm: báo đang chờ giảng viên duyệt (lớp 2 bật yêu cầu duyệt) | P2 |
| 01:35 | Giảng viên | `/class/members` (bộ chọn lớp → "Quản lý lớp này") | Duyệt yêu cầu của Sinh viên D | Sinh viên D chuyển sang thành viên; 3 yêu cầu seed khác vẫn chờ | P2 |

Nếu hội đồng hỏi (ngoài 15 phút): nhập sai mã → cùng một thông báo cho mọi trường hợp, khoá sau 5 lần / 10 phút; tạo lại mã → mã cũ bị từ chối. E2E: `class-join.spec.ts`.

### Bước 2. Hỏi riêng tư — F3 (02:00–03:30)

> "Sinh viên hỏi điểm của chính mình. Tên và MSSV được ẩn trước khi câu hỏi rời hệ thống; danh tính lấy từ phiên đăng nhập, không từ nội dung câu hỏi."

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 02:00 | Sinh viên B (375 px) | `/chat`, lớp 1 | Gõ D1, gửi | Dòng "Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI"; câu trả lời hiện dần; khối gọn nêu số buổi vắng (2) và điểm cộng phát biểu đúng số trong sổ; không thấy mã thay thế kiểu `[[SV_1]]` | P3 (số liệu: P5, P6) |
| 02:45 | Sinh viên B (375 px) | `/chat` | Bấm `Hữu ích` | Nút đổi trạng thái tại chỗ, không có thông báo nổi | P3 |
| 02:55 | Sinh viên B (375 px) | `/chat` | Gõ D2, gửi | Từ chối lịch sự: chỉ trả lời được thông tin của chính bạn; không có con số nào về Sinh viên C | P3 |

Nếu hội đồng hỏi: mất mạng giữa lúc đang trả lời → tải lại vẫn thấy phần đã hiện; mọi dịch vụ AI hỏng → câu trả lời trích từ tài liệu kèm nhãn. E2E: `private-chat.spec.ts`.

### Bước 3. Cần giảng viên hỗ trợ — F5 (03:30–05:15)

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 03:30 | Sinh viên B (375 px) | `/chat`, lớp 1 | Gõ D3, gửi | "AI chưa đủ chắc chắn về câu này"; câu hỏi được chuyển cho giảng viên, trạng thái "Đang chờ giảng viên · vừa gửi" (xem Q3 về việc có cần bấm `Nhờ giảng viên hỗ trợ`) | P4 (chat: P3) |
| 04:00 | Giảng viên | `/` → `/inbox` | Mở việc "Cần xử lý" mới nhất | Ticket ở lọc Open: câu hỏi gốc, thời gian chờ, lý do (độ tin cậy thấp) | P4 |
| 04:15 | Giảng viên | `/inbox` | `Nhận` | Trạng thái Claimed, ghi tên người nhận | P4 |
| 04:25 | Giảng viên | `/inbox` | Gõ D4 → `Gửi trả lời` | Ticket chuyển Answered | P4 |
| 04:50 | Sinh viên B (375 px) | `/chat` | Mở chuông | Câu trả lời của giảng viên nằm ngay trong phiên chat, có nhãn người trả lời là giảng viên | P4 |
| 05:05 | Người trình bày | `http://localhost:8025` (Mailpit, công cụ dev, không phải màn sản phẩm) | Mở thư mới nhất | Thư tới `sv.kha@edupilot.local` chứa câu trả lời và link về app | P4 (Mailpit: P0 L1) |

Nếu hội đồng hỏi: hai người cùng `Nhận` → người sau thấy ai đã nhận (409); mail lỗi → thử lại 3 lần, trả lời trong app không ảnh hưởng; không ai nhận 72 giờ → nổi lên đầu "Hôm nay". E2E: `escalation.spec.ts`.

### Bước 4. Điểm danh bằng QR và phát biểu — F7 (05:15–07:30)

> "Điểm danh không cần giảng viên gọi tên: sinh viên quét mã trên màn chiếu, mã đổi mỗi 10 giây nên chụp ảnh gửi bạn không dùng được."

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 05:15 | Giảng viên | `/` | Chạm việc "Điểm danh buổi đang diễn ra – 761987" | Mở `/attendance` đúng buổi hôm nay (Q1) | P5 (Hôm nay: P2) |
| 05:25 | Giảng viên | `/attendance` | `Mở điểm danh` → `Mở màn chiếu` | Cửa sổ mới `/attendance/display`: QR, **mã ngắn 6 ký tự** ngay dưới QR, số đã điểm danh, đồng hồ đếm ngược; QR và mã đổi cùng nhịp | 10.8 `[CHƯA KIỂM]` |
| 05:40 | 2 sinh viên (375 px) | camera hệ thống → `/checkin?c=…` | Quét QR trên màn chiếu | Màn «Đã điểm danh hh:mm»; không nút nào; số đã điểm danh trên màn chiếu tăng | 10.8 `[CHƯA KIỂM]` |
| 06:00 | Giảng viên (375 px) | `/attendance` | `Đóng`; dòng trạng thái đổi sang đã đóng. Chạm `Vắng` cho các sinh viên chưa quét, `Muộn` cho 1; ghi phát biểu +0,25 cho Sinh viên B ngay trên dòng | Mỗi dòng có «Điểm danh lúc»; sinh viên chưa quét ghi «Chưa điểm danh»; toàn bộ thao tác dưới 60 s | P5 + 10.8 |
| 06:40 | Giảng viên (375 px) | `/attendance` | `Lưu điểm danh` | Xác nhận đã lưu tại chỗ, không có thông báo nổi "Thành công"; còn sinh viên chưa có dòng thì báo «Còn n sinh viên chưa điểm danh» (422 `CHECKIN_UNRESOLVED`) | P5 + 10.8 |
| 06:55 | Sinh viên B (375 px) | `/me` | Mở | Hàng buổi hôm nay có «Điểm danh lúc hh:mm»; chuyên cần và điểm cộng có thêm +0,25; giải trình điểm dạng phép tính | P5 (giải trình: P6) |

Nếu hội đồng hỏi: mã cũ hết hạn sau 2 chu kỳ → «Mã đã hết hạn. Quét lại mã trên màn chiếu.»; sinh viên không thuộc lớp → bị từ chối bằng một câu «Bạn không thuộc lớp này.» (403; không nêu lớp nào khác, không cho điểm danh); hai sinh viên cùng một điện thoại → dòng bị gắn cờ «Cần xem»; hai người cùng điểm danh → khoá lạc quan; sửa sau buổi học → ghi nhật ký (Bước 8). E2E: `attendance.spec.ts`, `attendance-checkin.spec.ts`, `attendance-display.spec.ts`, `checkin-code.spec.ts`.


### Bước 5. Duyệt và công bố bài tập — F9 (07:30–09:00)

> Bài đã được nộp và chấm nháp ở bước chuẩn bị #3. Phần tạo bài và nộp bài có trong video phụ và E2E (xem Q4).

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 07:30 | Sinh viên B (375 px) | `/assignments/[id]` (Bài tập 03, mở từ `/me`) | Mở | Đã nộp, có dấu thời gian, nhãn nộp muộn; phần điểm ghi đang chấm — **không có điểm nháp** | P7 |
| 07:40 | Giảng viên | `/grading` (tab Hàng chờ chấm) | Giữ lọc mặc định `Cần xem kỹ` + `Chưa duyệt` | Bài của Sinh viên B ở đầu, lý do cờ: hai lượt chấm lệch hơn 1 điểm | P7 |
| 07:55 | Giảng viên | `/grading/[submissionId]` | Đọc bài bên trái, tiêu chí bên phải; sửa điểm tiêu chí bị lệch; sửa một câu nhận xét | Thông báo vàng tại tiêu chí lệch; mỗi tiêu chí có đoạn trích từ bài làm; tổng điểm tính lại ngay | P7 |
| 08:25 | Giảng viên | `/grading/[submissionId]` | `Duyệt bài` | Trạng thái đã duyệt | P7 |
| 08:35 | Giảng viên | `/grading` | Chọn bài vừa duyệt → `Công bố` | Trạng thái đã công bố; chỉ TEACHER thấy hành động này | P7 |
| 08:45 | Sinh viên B (375 px) | `/assignments/[id]` | Tải lại | Điểm, nhận xét theo từng tiêu chí, đoạn trích từ bài của chính mình làm bằng chứng | P7 |

Nếu hội đồng hỏi: nộp sau hạn khi bài không cho nộp muộn → bị chặn; worker chết giữa chừng → việc được nhận lại, không chấm đôi. E2E: `assignment-lifecycle.spec.ts`.

### Bước 6. Công thức điểm và sổ điểm — F10 (09:00–10:30)

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 09:00 | Giảng viên | `/gradebook`, lớp 2 | Đổi lớp ở bộ chọn lớp | Banner cố định "công thức điểm chưa xác nhận"; `Chốt điểm` bị khoá, lý do hiện khi rê chuột / focus | P6 |
| 09:10 | Giảng viên | `/gradebook/scheme`, lớp 2 | Tải `seed/demo/quy-che-lop2.pdf` | Tiến độ xử lý nền; bản nháp: thành phần, trọng số, điểm cộng, mỗi mục cạnh đoạn trích có số trang; mục "chưa rõ: quy tắc làm tròn" chặn nút xác nhận | P6 |
| 09:50 | Giảng viên | `/gradebook/scheme` | Điền D5 → `Xác nhận công thức` → xác nhận trong hộp thoại nêu hậu quả | Trạng thái đã xác nhận | P6 |
| 10:10 | Giảng viên | `/gradebook`, lớp 2 | Quay lại | Banner biến mất; `Chốt điểm` mở | P6 |
| 10:20 | Giảng viên | `/gradebook`, lớp 1 | Đổi lớp, mở giải trình dòng Sinh viên B | Thành phần điểm có +0,25 từ bước 4 và điểm Bài tập 03 từ bước 5; dòng ghi chú "điểm chính thức nằm ở hệ thống quản lý đào tạo của trường" | P6 (dữ liệu: P5, P7) |

`Chốt điểm` lớp 1 không làm trực tiếp trong 15 phút (xem Q5).

Nếu hội đồng hỏi: hai người sửa cùng ô → 409 và chọn giữ bên nào; mở khoá sau chốt cần lý do. E2E: `gradebook.spec.ts`.

### Bước 7. Báo cáo lỗ hổng kiến thức — F17 (10:30–11:30)

> "Hệ thống gom câu hỏi sinh viên đã gửi thành chủ đề và xếp hạng bằng công thức cố định; AI chỉ đặt tên chủ đề. Câu hỏi hiện ra đã ẩn danh."

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 10:30 | Giảng viên | `/insights`, lớp 761987 | Mở trang (báo cáo đã tạo ở chuẩn bị #4); nếu có dữ liệu mới bấm `Tạo báo cáo mới` | Danh sách «Chủ đề nên dạy lại» xếp hạng: chủ đề mã độc / tấn công (A) và quy chế học vụ (B) đứng đầu; mỗi chủ đề có số sinh viên hỏi, số câu và gợi ý «Nên làm» | P10 |
| 10:45 | Giảng viên | `/insights` | Mở «Câu hỏi mẫu» của chủ đề A | 3–5 câu đã ẩn danh, không có tên hay MSSV; chủ đề chỉ có dưới 3 sinh viên hỏi («Ít người hỏi») không có câu mẫu | P10 |
| 11:00 | Giảng viên | `/insights` | `Tạo thread ghim` của chủ đề A → xác nhận ngay tại chỗ | Form mở tại chỗ (không hộp thoại); thread ghim mới xuất hiện ở `/threads` của lớp | P10 (Threads: P4) |

Nếu hội đồng hỏi: lớp chưa đủ câu hỏi → «Chưa đủ dữ liệu» kèm số câu hiện có và số cần có (409); khoảng thời gian chọn 7 / 14 / 30 ngày hoặc toàn kỳ; mỗi lớp tối đa 3 báo cáo mỗi giờ; hai người cùng bấm tạo → chỉ một báo cáo chạy. E2E: `insights.spec.ts`; Go: `TestKnowledgeGap`.

### Bước 8. Nhật ký quản trị (11:30–12:00)

> "Mọi thay đổi về điểm, điểm danh và cấu hình đều có dấu vết. Nhật ký chỉ ghi ai đổi gì và khoá nào bị đổi, không ghi giá trị."

| Mốc | Vai trò | Route | Thao tác | Kết quả phải thấy | Phase |
| --- | --- | --- | --- | --- | --- |
| 11:30 | Admin | `/admin/audit` | Mở (khoảng 24 giờ) | Các dòng vừa tạo ở Bước 4–6: `attendance`, `grading_results` (`grading_result.publish`), `grade_schemes` (`extract` / `edit` / `confirm`; công thức lớp 2 ở Bước 6); mỗi dòng có thời điểm, người làm kèm vai, thực thể, hành động, khoá đổi; không có giá trị cũ / mới | P10 (P-11-30) |
| 11:45 | Admin | `/admin/audit` | Ô «Thực thể» gõ `grade_schemes` | Chỉ còn các dòng công thức điểm của Bước 6 | P10 |

Nếu hội đồng hỏi: chỉ Admin vào được (giảng viên / TA / sinh viên 403); chính việc mở nhật ký cũng được ghi lại. E2E: `audit.spec.ts`; Go: `TestAuditRBAC`.

### Nếu còn giờ (dự trữ, chỉ khi đang sớm hơn mốc)

Thêm theo thứ tự này; dừng khi tới mốc 13:30.

| Bước | Vai trò | Route | Thao tác | Kết quả phải thấy |
| --- | --- | --- | --- | --- |
| Bước 1 | Sinh viên D (375 px) | `/` | Tải lại trang | Bộ chọn lớp có lớp 761988; không còn ô nhập mã tham gia |
| Bước 3 | Sinh viên B (375 px) | `/chat` | Xác nhận đã rõ | Câu hỏi được đóng |
| Bước 4 | Sinh viên khác (375 px) | `/checkin` | không quét được: nhập mã ngắn dưới QR | «Đã điểm danh»; dòng ở `/attendance` có chip «Nhập mã» |
| Bước 4 | Giảng viên (375 px) | `/attendance` | Tắt mạng điện thoại → chạm một ô → bật mạng | Thay đổi xếp hàng rồi tự đồng bộ, không mất |
| Bước 6 | Giảng viên | `/gradebook`, lớp 1 | Xuất XLSX (menu) | File tải về có cột theo mẫu và dòng ghi chú điểm chính thức |
| Bước 7 | Giảng viên | `/insights`, lớp 761988 | Đổi lớp | Chủ đề TCP / UDP (C) đứng đầu — dữ liệu hai lớp không lẫn |
| Bước 7 | Giảng viên | `/insights` | Menu `⋯` → xuất Markdown | Tệp tải về: bảng chủ đề, không có tên hay MSSV |

### Kết (12:00–12:15)

> "Tám phần vừa xem đi qua cùng một dữ liệu: câu hỏi của sinh viên, câu trả lời của giảng viên, điểm danh, bài chấm và công thức điểm đều đổ về cùng một sổ điểm. Mọi quyết định về điểm do giảng viên bấm; AI chỉ soạn nháp."


## 5. Dự phòng cho ngày bảo vệ

Ba tầng, đi từ trên xuống. Người trình bày (chủ dự án) quyết định chuyển tầng.

| Tầng | Khi nào dùng | Bật ở đâu, thế nào | Mất gì |
| --- | --- | --- | --- |
| 1. LLM thật | Mặc định khi 9router chạy | Nhà cung cấp «9router» đang bật ở `/settings/llm` (`README.md` 5.3) | – |
| 2. Bản ghi sẵn (nhà cung cấp giả) | 9router hoặc mạng lỗi lúc chuẩn bị, hoặc một bước LLM lỗi / chậm quá 2 lần liên tiếp | Admin `/settings/llm` → công tắc «Đang bật» của nhà cung cấp thật sang **tắt** (nạp lại ≤ 60 s, không dựng lại container). Không còn nhà cung cấp bật thì gateway dùng nhà cung cấp giả của `.env.local` (`LLM_PROVIDER=fake`) | Câu trả lời chat là bản giả, ít tự nhiên; chấm bài / quy chế dùng bản ghi sẵn nên giống thật |
| 3. Video quay sẵn | Stack không lên, hoặc tầng 2 cũng lỗi | Mở file video trên laptop (bản sao trên USB), nhảy tới chương của bước đang dở | Không tương tác được |

### 5.1 Tầng 2 làm được gì

- Điều kiện cần: `.env.local` có `LLM_PROVIDER=fake` và `LLM_DEMO_FILE=/seed/demo/fake_llm.json` (mặc định của `.env.example`; compose gắn `./seed/demo` vào gateway và worker). Đổi `.env.local` thì phải tạo lại gateway và worker: `docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot up -d gateway worker` (≈ 30 s, nói lời dẫn trong lúc chờ). Tắt nhà cung cấp trên `/settings/llm` thì **không** cần (đã kiểm ở mức cổng LLM: `TestRegistryDisableProviderFallsBackToEnvFake` — tắt → mọi tác vụ dùng nhà cung cấp giả, bật lại → về nhà cung cấp thật; chưa bấm công tắc trên giao diện).
- Bản ghi sẵn `seed/demo/fake_llm.json` (theo nội dung yêu cầu, không theo băm) phủ: trích công thức quy chế lớp 1 / lớp 2 (D5), chấm Bài tập 03 hai lượt (Bước 5), các việc Luyện đề, và **chat D1, D3** (`chat-d1-so-lieu-ca-nhan`, `chat-d3-ngoai-tai-lieu`; test `TestDemoChatRecordings`). **Đã chạy trên stack (ep-qc, nhà cung cấp giả) 2026-11-15:** D1 ra câu dẫn + khối buổi vắng (2) và điểm phát biểu (0,50) lấy từ sổ; D2 ra «Mình chỉ xem được dữ liệu của chính bạn.»; D3 ra «chưa tìm thấy…» và mở ticket cho giảng viên.
  - D1: câu dẫn «đã tra sổ của bạn…» **không nêu số**; số buổi vắng và điểm phát biểu nằm ở khối dữ liệu lấy thẳng từ sổ, không từ LLM.
  - D2 (hỏi hộ bạn khác) là câu từ chối viết sẵn trong mã, không qua LLM. Phải gõ đúng họ tên trong danh sách lớp; tên lạ thì ra «chưa tìm thấy» và mở ticket.
  - D3: bản ghi là lời «chưa tìm thấy trong tài liệu của lớp»; chính lời này làm đường leo thang tới giảng viên chạy.
  - `/insights` ở tầng 2: bản ghi `insight-name-fallback` trả danh sách tên rỗng → báo cáo vẫn xong, chủ đề mang **tên tạm theo từ khoá** kèm dòng «Chưa đặt được tên chủ đề»; xếp hạng, số liệu, câu mẫu vẫn đúng (do mã, không do LLM). Không có bản ghi tên đẹp vì thứ tự chủ đề hoà điểm phụ thuộc thứ tự dữ liệu. Cách tốt nhất: tạo báo cáo ở chuẩn bị #4 khi LLM thật còn chạy; ở tầng 2 chỉ **xem** báo cáo đã có. `[CHƯA KIỂM trên stack: chạy cùng lượt dựng lại ep-qc sạch]`
- Đường đi còn nguyên: che tên / MSSV, Scheduler, `llm_audit`, ngưỡng độ tin cậy, escalation, mail. Chỉ lời gọi ra nhà cung cấp bị thay. Vì vậy dòng "Đã ẩn 2 thông tin cá nhân" và ticket ở Bước 3 vẫn sinh thật.
- Không cần mạng ra ngoài.
- LLM **chậm** (không lỗi): chat đã có câu trả lời hiện dần; chấm bài và báo cáo chạy nền có thanh tiến độ. Nói lời dẫn («đang chạy nền») rồi sang bước kế, quay lại sau; không đợi quá 20 s ở một chỗ.

### 5.2 Kiểm tầng 2 trước ngày bảo vệ

Chạy kịch bản trọn vẹn hai lần với nhà cung cấp thật đã tắt, một lần ngắt mạng hoàn toàn (kể cả 9router). Ghi kết quả vào mục 7.

### 5.3 Video

- Quay sau khi bản cuối vào `main`, theo đúng kịch bản này, có chương theo 8 bước, 1080p, tiếng thuyết minh.
- Thêm một video phụ ≤ 3 phút: tạo bài tập và sinh viên nộp bài (F9 bước 1–3), chốt điểm lớp 761987 (F10 bước 6), điểm danh bằng app Android (10.8) — phần không chạy trực tiếp trong 15 phút.
- Hai bản: ổ laptop demo và USB. Phát được không cần mạng.


## 6. Yêu cầu kịch bản đặt lên các phase sau

| Phase | Phải có để kịch bản chạy | Bước |
| --- | --- | --- |
| P1 | Provider `fake` phát lại bản ghi sẵn theo nội dung yêu cầu (`LLM_DEMO_FILE`); tắt nhà cung cấp thật ở `/settings/llm` để rơi về bản giả | 5.1 |
| P2 | Seed cố định: họ tên, MSSV của Sinh viên B, C (Q2); mã tham gia `AN7K2MQ` / `BX4P9TW`; lớp 2 bật yêu cầu duyệt; chuông phân công chưa đọc | 1, 2 |
| P3 | Dòng "Đã ẩn N thông tin cá nhân"; từ chối hỏi về người khác | 2 |
| P4 | Câu D3 thuộc nhóm "câu ngoài tài liệu" của seed (không tài liệu nào trả lời được); mail qua Mailpit | 3 |
| P5 | Một buổi học lớp 1 trùng giờ demo (Q1) | 4 |
| 10.8 | Lớp 761987 bật «Điểm danh bằng mã»; `/attendance/display`, `/checkin`, nhập mã ngắn; ≥ 3 tài khoản sinh viên đăng nhập sẵn trên điện thoại | 4 |
| P6 | File `seed/demo/quy-che-lop2.pdf` cố ý thiếu quy tắc làm tròn | 6 |
| P7 | Bài tập 03 lớp 1 có bài nộp muộn của Sinh viên B; hai lượt chấm của bài đó lệch > 1 điểm (bản ghi sẵn `bt03-*` trong `fake_llm.json`) | 5 |
| P10 | Câu hỏi seed lệch chủ đề A, B (lớp 1) và C (lớp 2), chủ đề D đúng 2 sinh viên; `/insights` thật; `/admin/audit`; video | 7, 8, 5.3 |

## 7. Nhật ký chạy lại

Cuối mỗi phase: chạy từ đầu tới bước xa nhất chạy được, bấm giờ, ghi một dòng.

| Ngày | Phase vừa xong | Bước xa nhất chạy thật | Tổng thời gian | Tầng | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| 2026-10-01 | P0 (đang làm) | – (chưa có bước nào chạy) | – | – | Bản kịch bản đầu tiên |
| 2026-11-15 | Sprint 11 (viết lại v2) | Chưa chạy trọn | – | – | Cập nhật theo mã `sprint/11-p10-eval`; Bước 4 (QR) theo SRS vì 10.8 chưa merge. Chạy trọn trên stack demo sau merge: ghi tại đây |
