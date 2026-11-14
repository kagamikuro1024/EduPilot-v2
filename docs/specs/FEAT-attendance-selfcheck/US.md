# FEAT-attendance-selfcheck Điểm danh tự phục vụ chống gian lận (web + app Android)
Nguồn: PRD M5 (+ AC mới), FLOWS F7 và F7b (mới), phase P5 mở rộng; sprint 10.8 lát L1–L5, L7; `docs/sprints/10.8/techlead.md` (TL-1), `docs/research/2026-11-14-attendance-anti-cheat.md`, quyết định **P-108-1** (`docs/sprints/10.8/proposals.md`, PM chốt 2026-11-14 theo uỷ quyền chủ dự án).

Phiên bản 1.1 · 2026-11-14 · Trạng thái: **DRAFT** (PM đã chốt Q-A-1, Q-A-2, Q-A-4 ở **P-108-2**, chờ PM duyệt toàn bộ; v1.1 = vá theo P-108-2: mã ngắn dưới QR, quét trong WebView, bỏ App Links).

Quy ước: `Staff` = TA + TEACHER `ACTIVE` của lớp. ADMIN bị từ chối ở mọi route lớp (D39). Đường dẫn `…` = `/api/v1/courses/{id}`. «cờ» = mã bất thường chỉ giảng viên / TA thấy. Sinh viên **không bao giờ** thấy cờ, mức tin cậy, thiết bị, vị trí đã lưu.

**Luật cứng của feature (kiểm bằng test):**
1. **Chỉ gắn cờ, không tự phạt.** API điểm danh của sinh viên chỉ từ chối bằng đúng ba nhóm lỗi: mã sai / hết hạn (`422`), cửa sổ đóng (`409`), không thuộc lớp / sai vai (`403`) — cộng `429`, `409 SELF_CHECKIN_DISABLED`, `409 COURSE_ARCHIVED`. Mọi tín hiệu khác (máy, vị trí, IP, ứng dụng) chỉ thành cờ; lượt vẫn được ghi.
2. **Không tự ghi vắng.** Hết giờ / đóng cửa sổ không tạo `ABSENT`. Chỉ giảng viên bấm «Đánh vắng các SV chưa điểm danh» (P-108-1 Q1).
3. **Giảng viên quyết, có vết.** Mọi sửa trạng thái đi qua lưới `PUT …/attendance` có sẵn (khoá `version`, `audit_log`); mọi thao tác của Staff trong feature này ghi `audit_log` cùng giao dịch.
4. **Danh tính chỉ từ JWT.** Không route nào nhận `student_id` của sinh viên từ thân yêu cầu.
5. **Không sinh trắc, không lưu toạ độ, không lưu IP.** Chỉ lưu khoảng cách, độ chính xác, `in_campus`. Tín hiệu thô giữ **30 ngày** sau khi đóng buổi, sau đó chỉ còn trạng thái cuối trong `attendance_records` (P-108-1 Q9).
6. Web và app Android dùng **cùng một API**; app chỉ cộng thêm lớp bảo đảm (khoá phần cứng có chứng thực, phát hiện vị trí giả).

**Ngoài phạm vi của cả feature:** selfie / khuôn mặt, chặn chụp màn hình, NFC, BLE, cờ riêng cho «mã đọc to» (mã ngắn hiện dưới QR thì **có** — P-108-2), iOS native / App Attest / Play Integrity / Play Console, ML Kit, thông báo đẩy, SSE cho màn chiếu, tự phạt hay trừ điểm, đọc danh sách «cùng máy với ai» (nợ).

---

## US-AS-01: Giảng viên muốn bật điểm danh tự phục vụ cho lớp và đặt chính sách, để lớp nào dùng thì lớp đó bật
Ưu tiên: Must · Ước lượng: S · Sprint: 10.8 (L1; trường vị trí / mạng ở L4)

### Tiêu chí nghiệm thu
- AC1 (mặc định tắt). Given lớp có sẵn Then `courses.settings.attendance` thiếu hoặc `self_checkin=false`; mọi route mở cửa sổ và `POST …/checkins` trả `409 SELF_CHECKIN_DISABLED`; điểm danh tay (`PUT …/attendance`) hoạt động **y hệt** trước sprint (không đổi byte phản hồi).
- AC2 (đặt chính sách). Given TEACHER của lớp When `PUT …/settings {attendance:{self_checkin, rotate_seconds, late_after_min, require, geofence?, campus_cidrs?}, version}` Then `200 CourseDetail`; `version + 1`; ghi `audit_log` (`course_settings`, `attendance_policy`, trước / sau, **không** toạ độ trong `audit_log` — chỉ «geofence: bật / tắt, bán kính»); sự kiện outbox `course.settings_changed`.
- AC3 (miền giá trị). Given thân yêu cầu When `rotate_seconds ∉ [5,60]`, `late_after_min ∉ [0,120]`, `require ∉ {WEB, APP}`, `geofence.radius_m ∉ [20,2000]`, lat / lng ngoài miền, hơn 10 CIDR, CIDR sai cú pháp hoặc `/0` Then `422 VALIDATION_FAILED` kèm tên trường; không ghi gì. Mặc định: `rotate_seconds=10`, `late_after_min=15`, `require=WEB`.
- AC4 (nhánh lỗi). Given `version` cũ → `409 VERSION_CONFLICT` (không ghi); lớp lưu trữ → `409 COURSE_ARCHIVED`; hai `PUT` song song cùng `version` → đúng một thắng.
- AC5 (đổi giữa buổi không phá cửa sổ đang mở). Given cửa sổ đang mở When TEACHER đổi `rotate_seconds` hoặc tắt `self_checkin` Then cửa sổ đang mở giữ `rotate_seconds` đã ghim lúc mở; tắt `self_checkin` → cửa sổ đang mở vẫn nhận lượt tới khi đóng, không mở được cửa sổ mới.
- AC6 (phân quyền). Given TA / STUDENT / ADMIN / người ngoài lớp When `PUT …/settings` có `attendance` Then `403` (lớp khác: `404`); `GET /courses/{id}` trả chính sách đầy đủ cho Staff; STUDENT thấy duy nhất `self_checkin` (bool) và `require` (để giao diện biết có mở `/checkin`), **không** thấy `geofence`, `campus_cidrs`.
- AC7 (câu thông báo dùng vị trí — Luật 91 Đ31 k3). Given TEACHER bật `geofence` Then giao diện hiện **một dòng** «Lớp này kiểm tra vị trí khi điểm danh.» cạnh công tắc (đây là câu bắt buộc theo luật, không phải câu giải thích); sinh viên thấy đúng dòng đó trước khi trình duyệt / hệ điều hành hỏi quyền (US-AS-10).

### Ngoài phạm vi của story này
- Chính sách riêng từng buổi; mặc định theo học kỳ.

### Phụ thuộc
- `FEAT-course-foundation` `PUT …/settings`, `CourseDetail`; migration 00024 không tạo bảng cho chính sách.

---

## US-AS-02: Giảng viên / TA muốn mở và đóng cửa sổ điểm danh cho một buổi, tới tối đa 3 vòng, để sinh viên tự điểm danh trong giờ
Ưu tiên: Must · Ước lượng: M · Sprint: 10.8 (L1)

### Tiêu chí nghiệm thu
- AC1 (mở). Given lớp bật `self_checkin`, buổi `sid` của lớp, chưa có cửa sổ đang mở When Staff `POST …/sessions/{sid}/checkin-windows {minutes?}` kèm `Idempotency-Key` Then `201 {id, round, rotate_seconds, closes_at}`; `round` = số vòng kế tiếp (1, 2, 3); `minutes` ∈ [1,60], mặc định 10 cho vòng 1 và 3 cho vòng ≥ 2; `rotate_seconds` ghim từ chính sách lớp; ghi `audit_log` (`checkin_window.open`); cùng `Idempotency-Key` → cùng `id`, không tạo cửa sổ thứ hai.
- AC2 (một cửa sổ mở mỗi buổi). Given buổi đã có cửa sổ mở (chưa đóng) When mở tiếp Then `409 WINDOW_ALREADY_OPEN` kèm `{window_id}`; hai yêu cầu mở song song khác `Idempotency-Key` → đúng một `201`, một `409` (chỉ mục duy nhất một phần).
- AC3 (giới hạn vòng). Given buổi đã có 3 vòng When mở tiếp Then `409 ROUND_LIMIT`. Vòng 2 / 3 **không** gửi thông báo, thư, chuông hay «Hôm nay» cho sinh viên (kiểm: không có sự kiện outbox nào tới sinh viên khi mở vòng ≥ 2).
- AC4 (hiển thị cho màn chiếu). Given cửa sổ đang mở When Staff `GET …/sessions/{sid}/checkin-windows/{wid}/display` Then `200 {qr_url, short_code, step_expires_at, server_time, rotate_seconds, round, checked_in, headcount|null, closes_at}`; `qr_url` = `{APP_PUBLIC_URL}/checkin?c={courseId}#{token}` (mã sau `#`, không vào access log, không đi theo Referer); `Cache-Control: no-store`; `short_code` = mã ngắn 6 ký tự (bảng chữ Crockford base32, SRS FR-3a) hiện **dưới QR**, đổi **cùng nhịp** với `qr_url`; hai lần gọi trong cùng bước cho **cùng** `qr_url` và `short_code`; bước kế cho mã khác.
- AC5 (liệt kê). Given Staff When `GET …/sessions/{sid}/checkin-windows` Then tối đa 3 cửa sổ `{id, round, opened_at, closes_at, closed_at|null, headcount|null, checked_in, over_headcount}`; không chứa mã.
- AC6 (số đầu người). Given cửa sổ (mở hoặc đã đóng) When Staff `PATCH …/checkin-windows/{wid} {headcount}` (`0 ≤ headcount ≤ 2000`) Then `200`; `audit_log` (`checkin_window.headcount`); `over_headcount = (số SV khác nhau đã điểm danh trong buổi) > headcount + max(2, ⌈5 % × headcount⌉)`.
- AC7 (tự đóng khi hết giờ). Given cửa sổ quá `closes_at` mà chưa ai đóng When tới nhịp quét của worker (≤ 60 s) Then cửa sổ được đóng bằng đúng thủ tục của US-AS-05 (`closed_by = NULL`, `audit_log` `checkin_window.auto_close`); đóng lại lần hai không đổi gì.
- AC8 (nhánh lỗi). Given `minutes` ngoài [1,60] hoặc `headcount` âm → `422`; `sid` không thuộc lớp → `404`; lớp lưu trữ → `409 COURSE_ARCHIVED`; `display` của cửa sổ đã đóng / hết hạn → `409 WINDOW_CLOSED`; `display` quá 30 lần / phút / người → `429`.
- AC9 (phân quyền). Given STUDENT / ADMIN / người ngoài lớp When bất kỳ route trong story này Then `403` (người ngoài lớp: `404`); TA và TEACHER đều được; ghi `audit_log` mang đúng người thao tác.

### Ngoài phạm vi của story này
- Gia hạn một cửa sổ (mở vòng mới thay thế); SSE cho màn chiếu (dùng poll, P-108-1).

### Phụ thuộc
- US-AS-01; migration 00024; `ATTENDANCE_TOKEN_KEY`.

---

## US-AS-03: Sinh viên muốn quét mã trên màn chiếu để điểm danh bằng một thao tác, để khỏi chờ gọi tên
Ưu tiên: Must · Ước lượng: M · Sprint: 10.8 (L1)

### Tiêu chí nghiệm thu
- AC1 (điểm danh thành công). Given cửa sổ mở, STUDENT `ACTIVE` của lớp, mã trong bước hiện tại hoặc bước trước When `POST /courses/{id}/checkins {token | code, device?, geo?, app?}` (đúng **một** trong `token`, `code`; cả hai hoặc không có → `422 VALIDATION_FAILED`) Then `201 {checked_in_at, round}`; ghi đúng một hàng `attendance_checkins`; `student_id` lấy từ JWT.
- AC2 (mã). Given `token = {window_id}.{step}.{mac}`, `mac = base32(HMAC-SHA256(K_w, step))[:16]`, `K_w = HMAC(ATTENDANCE_TOKEN_KEY, "att-v1|"+window_id)`, `step = ⌊unix / rotate_seconds⌋` Then chấp nhận `step ∈ {hiện tại, trước}` (tuổi mã tối đa 2 × `rotate_seconds`); so sánh MAC bằng `hmac.Equal`; không lưu bí mật nào theo buổi.
- AC3 (lỗi mã). Given MAC sai, cửa sổ lạ, cửa sổ của lớp khác, định dạng sai → `422 TOKEN_INVALID`; MAC đúng nhưng `step` ngoài hai bước → `422 TOKEN_EXPIRED`; **không** phân biệt «cửa sổ lạ» với «MAC sai» (không lộ cửa sổ có tồn tại). Thứ tự kiểm: JWT → quyền lớp → cú pháp → cửa sổ (đúng lớp) → MAC → cửa sổ đã đóng / quá `closes_at` (`409 WINDOW_CLOSED`) → bước → khử trùng.
- AC3a (nhập mã ngắn — P-108-2). Given sinh viên không quét được QR (ví dụ app mà WebView không có `BarcodeDetector`) When gửi `{code}` (6 ký tự hiện dưới QR; bỏ khoảng trắng / gạch, không phân biệt hoa thường, `O→0`, `I`/`L→1`) Then server dò các cửa sổ **của lớp `{id}`** có `closes_at > now() − 15 phút`, so `code` với mã ngắn của bước hiện tại và bước trước (`subtle.ConstantTimeCompare`); khớp cửa sổ đang mở → ghi lượt với `channel = CODE` như AC1; khớp cửa sổ đã đóng → `409 WINDOW_CLOSED`; không khớp (sai **hoặc** hết hạn — không phân biệt) → `422 TOKEN_INVALID`. Lượt `CODE` không có cờ riêng (chỉ nhãn kênh «Nhập mã» cho Staff, US-AS-06 AC1); mọi quy tắc khác của US-AS-03 / US-AS-04 giữ nguyên (khử trùng, chữ ký, cờ, không lộ cờ, không tự phạt). Chữ ký thiết bị ký `"edupilot-checkin-v1|code:"+CODE_CHUẨN_HOÁ+"|"+user_id`.
- AC3b (chống dò mã ngắn). Given nhập sai Then đếm **chỉ lần sai** (Redis `ep:att:codefail:{uid}`, TTL 60 s); từ lần sai thứ 6 trong 1 phút → `429` (kèm `Retry-After`) ngay cả khi mã đúng sau đó, cho tới hết phút; lượt đúng không bị đếm; đường nhập `token` không dùng bộ đếm này.
- AC4 (khử trùng, idempotent theo khoá tự nhiên). Given cùng sinh viên đã có lượt ở cửa sổ này When gửi lại (làm mới trang, mạng chập chờn, hai tab) Then `200` cùng `{checked_in_at, round}` của lượt đầu; không hàng thứ hai (`UNIQUE (course_id, window_id, student_id)`); không đổi cờ. Vòng 2 là cửa sổ khác → hàng khác.
- AC5 (không bao giờ lộ cờ). Given lượt có 0 cờ và lượt có 5 cờ Then thân phản hồi, mã trạng thái và độ dài **giống hệt nhau** (kiểm bằng so sánh byte sau khi bỏ `checked_in_at`); không có trường `flags`, `assurance`, `device`, `geo` ở bất kỳ route nào của sinh viên (`TestCheckinResponseHidesFlags`).
- AC6 (không tự phạt). Given mọi tổ hợp tín hiệu: không chữ ký, chữ ký sai, máy dùng chung, ngoài vùng, ngoài mạng trường, vị trí giả, từ chối quyền vị trí Then luôn `201` / `200` và lượt được ghi kèm cờ (`TestCheckinNeverRejectsOnSignals`, bảng ca).
- AC7 (trạng thái điểm danh). Given lượt thành công When `GET /courses/{id}/me/attendance` Then thêm trường `checked_in_at` (hoặc `null`) cho buổi tương ứng; không có `note`, cờ, mức tin cậy; tool AI `get_my_attendance` **không** nhận trường mới nào (`TestAgentToolNoCheckinFlags`).
- AC8 (lỗi khác). Given lớp tắt `self_checkin` → `409 SELF_CHECKIN_DISABLED`; lớp lưu trữ → `409 COURSE_ARCHIVED`; quá 10 lượt / phút / người → `429` (kèm `Retry-After`; riêng đường `code` xem AC3b); thân sai kiểu (`geo.lat` chuỗi, `signature` không phải base64) → `422 VALIDATION_FAILED`; mất mạng giữa chừng: gửi lại an toàn theo AC4.
- AC9 (phân quyền). Given không JWT → `401`; TA / TEACHER / ADMIN → `403` (không điểm danh hộ qua API); STUDENT chưa ghi danh `ACTIVE` hoặc `PENDING` / đã bị mời ra → `403 NOT_IN_COURSE`; thân có `student_id` hoặc `user_id` → bị bỏ qua (danh tính chỉ từ JWT; `TestCheckinIdentityFromJWT`).
- AC10 (không rò). Given mọi dòng log của gateway trong luồng Then không có `token`, toạ độ, IP, `public_key`, `signature` (`TestLogsNoCheckinSecrets`: ghi log vào bộ đệm rồi `grep`).

### Ngoài phạm vi của story này
- Khoá thiết bị, cờ, mức tin cậy (US-AS-04); vị trí / IP (US-AS-10); màn hình (US-AS-09).

### Phụ thuộc
- US-AS-02; `CourseAccessGuard`; middleware hạn mức; migration 00024.

---

## US-AS-04: Giảng viên muốn mỗi lượt điểm danh được gắn cờ khi có dấu hiệu hộ / sai máy, để biết ai cần xem mà không phạt oan
Ưu tiên: Must · Ước lượng: M · Sprint: 10.8 (L1)

### Tiêu chí nghiệm thu
- AC1 (chữ ký thiết bị). Given `device:{kind:"WEB_KEY", public_key (SPKI base64), signature}` When chữ ký ECDSA P-256 hợp lệ trên chuỗi `"edupilot-checkin-v1|"+token+"|"+user_id` (đường `code`: `"edupilot-checkin-v1|code:"+CODE_CHUẨN_HOÁ+"|"+user_id`) Then `user_devices` có (hoặc tạo) hàng theo `sha256(public_key)`; lượt gắn `device_id`; chấp nhận chữ ký `r‖s` 64 byte (WebCrypto) và DER (Keystore). Không cần nonce, không cần Redis.
- AC2 (không gắn thiết bị vào người). Given bảng `user_devices` Then **không có cột `user_id`**; không bảng liên kết thiết bị–người; «máy thuộc về ai» chỉ suy ra từ `attendance_checkins` (kiểm lược đồ).
- AC3 (cờ — bảng). Given lượt Then cờ tính **tất định** như SRS 4.4: `NO_DEVICE_KEY` (không gửi chữ ký), `DEVICE_UNTRUSTED` (có chữ ký nhưng sai / khoá app chưa đăng ký / chứng thực hỏng), `DEVICE_SHARED`, `DEVICE_NEW`, `TOKEN_LATE`, `NOT_APP` (lớp `require=APP` mà mức < 3); vị trí / mạng xem US-AS-10, `GEO_MOCK` xem US-AS-12, `RECHECK_MISS` xem US-AS-07.
- AC4 (`DEVICE_SHARED`). Given hai sinh viên khác nhau trong cùng lớp điểm danh bằng cùng `device_id` Then lượt sau gắn `DEVICE_SHARED` **và** mọi lượt của sinh viên kia trong cùng buổi được thêm `DEVICE_SHARED` (giảng viên thấy cả hai người).
- AC5 (`DEVICE_NEW`). Given sinh viên đã có lượt bằng thiết bị X và chưa từng có lượt bằng thiết bị Y When điểm danh bằng Y Then lượt đó gắn `DEVICE_NEW`; lượt kế bằng Y **không** còn cờ (đã từng dùng); lượt đầu tiên trong đời của sinh viên không bị `DEVICE_NEW` (Q3: đổi máy chỉ gắn cờ một lần).
- AC6 (`TOKEN_LATE`). Given tuổi mã `token_age_ms = now − step × rotate_seconds × 1000 > (rotate_seconds + 5) × 1000` Then gắn `TOKEN_LATE`; ngưỡng `TOKEN_LATE_EXTRA_MS=5000` là hằng cấu hình để hiệu chỉnh theo dữ liệu thật.
- AC7 (mức tin cậy). Given lượt Then `assurance` ∈ {1,2,3} theo SRS 4.5 (1 = không có khoá hợp lệ hoặc trái vùng; 2 = khoá hợp lệ; 3 = app có khoá phần cứng chứng thực, máy đã khoá bootloader, vị trí không giả); lưu trên lượt; lớp `require=APP` và mức < 3 → cờ `NOT_APP`, **vẫn ghi** (Q2).
- AC8 (nhãn tiếng Việt, không từ kỹ thuật). Given Staff xem Then mỗi cờ hiện nhãn ở SRS 4.4.1 («Cùng máy với bạn khác», «Máy mới», «Không xác minh được máy», …); API trả `{code, label}`; nhãn **không** chứa từ «token», «HMAC», «attestation», «key», «device»; sinh viên không thấy.
- AC9 (nhánh lỗi). Given chữ ký hỏng / công khai hỏng / bộ nhớ trình duyệt bị chặn Then lượt vẫn ghi với `NO_DEVICE_KEY` hoặc `DEVICE_UNTRUSTED`, không lỗi; hai yêu cầu đồng thời của hai sinh viên trên cùng máy đều thấy `DEVICE_SHARED` (không mất cờ do đua; `TestDeviceSharedRace`).
- AC10 (phân quyền). Given STUDENT / ADMIN Then không route nào trả cờ, `assurance`, `device_id`; Staff chỉ đọc cờ của lớp mình (US-AS-06).

### Ngoài phạm vi của story này
- Khoá phần cứng và chứng thực (US-AS-12); danh sách «cùng máy với ai» (nợ).

### Phụ thuộc
- US-AS-03.

---

## US-AS-05: Giảng viên muốn đóng cửa sổ để lượt điểm danh thành trạng thái có mặt / muộn, và tự bấm đánh vắng người chưa điểm danh, để lưới điểm danh đúng mà không ai bị ghi vắng oan
Ưu tiên: Must · Ước lượng: M · Sprint: 10.8 (L1)

### Tiêu chí nghiệm thu
- AC1 (đóng). Given cửa sổ mở When Staff `POST …/checkin-windows/{wid}/close` Then `200 {window, written:{present, late}, unchecked, skipped_existing}`; đặt `closed_at`, `closed_by`; một giao dịch; `audit_log` (`checkin_window.close`, số đếm, **không** tên).
- AC2 (ghi bản ghi). Given sinh viên có ≥ 1 lượt trong buổi và chưa có `attendance_records` Then `INSERT … ON CONFLICT DO NOTHING` một câu: `PRESENT`, hoặc `LATE` khi **lượt sớm nhất** của buổi sau `starts_at + late_after_min`; `marked_by` = người đóng (tự đóng: người mở); `version = 1`; **không ghi đè** dòng đã tồn tại (giảng viên đã sửa tay) — dòng đó tính vào `skipped_existing`.
- AC3 (không tự vắng). Given sinh viên `ACTIVE` không có lượt nào Then **không** có hàng `ABSENT` nào được tạo khi đóng; `unchecked` = số sinh viên `ACTIVE` chưa có dòng điểm danh của buổi.
- AC4 (idempotent). Given cửa sổ đã đóng When `close` lại Then `200` cùng kết quả đã lưu, không ghi thêm, không audit thêm.
- AC5 (Đánh vắng các SV chưa điểm danh). Given buổi có ít nhất một cửa sổ và không cửa sổ nào đang mở When Staff `POST …/sessions/{sid}/absent-unchecked` (`Idempotency-Key`) Then ghi `ABSENT` (`ON CONFLICT DO NOTHING`, `marked_by` = người gọi) cho mọi sinh viên `ACTIVE` chưa có dòng của buổi; `200 {marked}`; `audit_log` (`attendance`, `absent_unchecked`, số đếm + `[{student_id, to:"ABSENT"}]`, không tên); sau đó mỗi hàng sửa được bằng lưới tay có `version` (Hoàn tác dùng lưới).
- AC6 (chặn hoàn tất lưới khi còn người chưa xử lý — P-108-2, PM đồng ý). Given buổi có ≥ 1 cửa sổ và còn sinh viên `ACTIVE` chưa có dòng When `PUT …/sessions/{sid}/attendance {complete:true}` mà thân không phủ hết họ Then `422 CHECKIN_UNRESOLVED {count}`; không ghi gì; buổi **không** có cửa sổ → hành vi cũ (điền `PRESENT`) nguyên vẹn (`TestCompleteUnchangedWithoutWindow`).
- AC7 (nhánh lỗi). Given `absent-unchecked` khi còn cửa sổ mở → `409 WINDOW_OPEN`; buổi chưa từng có cửa sổ → `409 NO_WINDOW`; `close` cửa sổ lớp khác → `404`; lớp lưu trữ → `409 COURSE_ARCHIVED`.
- AC8 (đua). Given `close` song song `PUT …/attendance` cùng buổi Then không mất ghi, không trùng dòng (khoá `FOR UPDATE` trên `class_sessions` cùng thứ tự như `PUT`); dòng giảng viên vừa lưu thắng.
- AC9 (phân quyền). Given STUDENT / ADMIN → `403`; người ngoài lớp → `404`; TA và TEACHER đều được.

### Ngoài phạm vi của story này
- Mở khoá buổi đã chốt sổ điểm (P6).

### Phụ thuộc
- US-AS-02, US-AS-03; `FEAT-crm-360` 4.3 (`PUT …/attendance`, `version`, `audit_log`).

---

## US-AS-06: Giảng viên / TA muốn xem danh sách lượt cần xem, bỏ cờ sau khi xác nhận, và thấy việc ở «Hôm nay», để không sót mà không bị ngập
Ưu tiên: Must · Ước lượng: S · Sprint: 10.8 (L1; giao diện ở US-AS-08)

### Tiêu chí nghiệm thu
- AC1 (danh sách, phân trang con trỏ). Given Staff When `GET …/sessions/{sid}/checkins?flagged=&round=&cursor=&limit=` Then `{items:[{id, window_id, round, student:{id, full_name, student_code}, created_at, channel (`WEB`/`APP`/`CODE`; giao diện chỉ hiện chip «Nhập mã» khi `CODE`), assurance, flags:[{code,label}], cleared, geo_distance_m?, in_campus?}], next_cursor}`; `limit` ≤ 100 (mặc định 50); `flagged=true` chỉ lượt còn cờ chưa bỏ; thứ tự `(created_at, id)`; không OFFSET; chỉ mục `(course_id, session_id, created_at)`.
- AC2 (bỏ cờ). Given lượt có cờ When Staff `POST …/checkins/{cid}/clear-flags` Then `flags_cleared_by`, `flags_cleared_at` đặt; `audit_log` (`checkin.clear_flags`, mã cờ, `student_id`, không tên); gọi lại → `200` không đổi; cờ **không bị xoá** khỏi hàng (chỉ đánh dấu đã xem).
- AC3 (Hôm nay). Given buổi có cửa sổ đã đóng ≤ 7 ngày (`CHECKIN_REVIEW_DAYS`) và k ≥ 1 lượt còn cờ chưa bỏ When TA / TEACHER gọi `GET /me/today` Then có mục `CHECKIN_REVIEW` (bậc 13): «Buổi {no}: {k} lượt điểm danh cần xem · {mã lớp}», `Href=/attendance?course=…&session=…&filter=review`; k = 0 hoặc quá 7 ngày → mục biến mất; STUDENT không có mục này.
- AC4 (nhánh lỗi). Given `limit > 100` → `422`; `cursor` hỏng → `422`; `cid` của lớp khác → `404`; lớp lưu trữ: đọc được, `clear-flags` → `409 COURSE_ARCHIVED`.
- AC5 (phân quyền). Given STUDENT → `403` ở mọi route; ADMIN → `403`; người ngoài lớp → `404`; sinh viên không bao giờ thấy mục `CHECKIN_REVIEW` hay cờ của mình.
- AC6 (dữ liệu quá hạn). Given lượt đã quá 30 ngày Then không còn trong danh sách (US-AS-13).

### Ngoài phạm vi của story này
- Tự động đánh vắng theo cờ; gửi thư cho sinh viên về cờ.

### Phụ thuộc
- US-AS-04; `FEAT-course-foundation` 4.7 (`Hôm nay`).

---

## US-AS-07: Giảng viên muốn đối chiếu bằng người thật (đếm đầu người, vòng 2, gọi tên ngẫu nhiên), để bắt cái máy không bắt được
Ưu tiên: Should · Ước lượng: S · Sprint: 10.8 (L1 API, L2 giao diện)

### Tiêu chí nghiệm thu
- AC1 (số đầu người). Cửa sổ có `headcount`; buổi báo `over_headcount` theo US-AS-02 AC6; giao diện hiện nhãn «Số điểm danh nhiều hơn số người đếm» khi `true` (không chặn gì).
- AC2 (vòng 2). Given vòng 1 đã đóng When Staff mở vòng 2 lúc tuỳ ý (US-AS-02 AC1) Then sinh viên **không được báo trước** (US-AS-02 AC3); khi vòng ≥ 2 đóng → với mỗi sinh viên có lượt ở vòng trước mà **không** có lượt ở vòng này, thêm cờ `RECHECK_MISS` vào lượt vòng trước (một câu `UPDATE`, không đổi `attendance_records`).
- AC3 (gọi tên ngẫu nhiên). Given buổi có ≥ 1 lượt When Staff `POST …/checkin-windows/{wid}/spot-check {n}` (`1 ≤ n ≤ 10`, mặc định 5) Then `200 {students:[{student_id, full_name, student_code, reason}]}`; chọn **lượt còn cờ chưa bỏ trước**, thiếu thì bổ sung ngẫu nhiên bằng `crypto/rand`, không trùng; không lưu kết quả; không audit (không đổi dữ liệu); `n` lớn hơn số lượt → trả hết.
- AC4 (nhánh lỗi). Given `n` ngoài [1,10] → `422`; buổi chưa có lượt → `200 {students:[]}`; `wid` lớp khác → `404`.
- AC5 (phân quyền). Given STUDENT / ADMIN → `403`; người ngoài lớp → `404`; TA và TEACHER được.

### Phụ thuộc
- US-AS-02, US-AS-04.

---

## US-AS-08: Giảng viên / TA muốn mở điểm danh ngay trên `/attendance` và chiếu mã lên màn hình, để sinh viên quét trong lớp
Ưu tiên: Must · Ước lượng: M · Sprint: 10.8 (L2)

### Tiêu chí nghiệm thu
- AC1 (khởi động). Given lớp bật `self_checkin`, Staff mở `/attendance` chọn buổi Then có một hành động phụ `Mở điểm danh` (hành động chính vẫn `Lưu điểm danh`); bấm → `POST …/checkin-windows` rồi mở **màn chiếu** `/attendance/display?course=…&session=…` cùng tab; lớp tắt `self_checkin` → không hiện nút.
- AC2 (màn chiếu). Given `/attendance/display` Then toàn màn hình, không thanh điều hướng, nền sáng, QR do `uqr` vẽ SVG, **mã ngắn** 6 ký tự (nhóm 3 + 3, chữ lớn, đơn cách) ngay dưới QR và **đổi cùng nhịp** với QR, **số đã điểm danh** (chữ lớn), **đồng hồ đếm ngược** tới `closes_at`, vòng hiện tại; mã tự đổi theo `step_expires_at` (poll `display`, không SSE; lệch đồng hồ bù bằng `server_time`); mất mạng → giữ QR cuối cùng + chỉ báo «Mất kết nối» (một dòng) và thử lại; hai nút nhỏ `Đóng điểm danh`, `Thoát`; `Esc` thoát.
- AC3 (tại chỗ trên lưới). Given cửa sổ mở Then dòng trạng thái tĩnh «Đang mở · {k}/{n} · còn mm:ss» + `Mở màn chiếu`, `Đóng`; `Đóng` → `close`, kết quả «Đã ghi {a} có mặt · {b} muộn · {u} chưa điểm danh» tại chỗ (không toast) kèm `Đánh vắng {u} SV chưa điểm danh` (hộp xác nhận chuẩn của `shared/`, nêu số người).
- AC4 (lưới cho buổi có điểm danh). Given buổi có cửa sổ Then mỗi hàng: «Điểm danh lúc 07:03» (nếu có lượt), chip nhãn cờ cho Staff, SV chưa có dòng hiện «Chưa điểm danh» (**không** mặc định `Có mặt` — khác buổi không có cửa sổ); bộ lọc `Cần xem ({k})`; hàng có cờ có `Đã xem` (gọi `clear-flags` cho các lượt của người đó trong buổi); sửa trạng thái tại chỗ như cũ (khoá `version`, 5 giây «Hoàn tác»).
- AC5 (vòng 2, đếm đầu người, gọi tên). Given vòng 1 đã đóng Then menu `⋯` của vùng điểm danh có `Mở vòng 2`, `Nhập số đầu người` (ô số, tự lưu), `Gọi ngẫu nhiên 5 người` (hiện 5 tên + MSSV + lý do «Có cờ» / «Ngẫu nhiên» trong `Drawer`).
- AC6 (trạng thái). Given tải = khung xương; lỗi = nói chuyện gì xảy ra + `Thử lại`; `409 WINDOW_ALREADY_OPEN` → «Đang có cửa sổ mở» + `Mở màn chiếu`; 375 px dùng được (lưới là danh sách hàng, nút ≥ 44 px); màn chiếu dùng được ở 1280×720.
- AC7 (lỗi / mất mạng). Given mất mạng khi `Đóng` → hàng đợi, nhãn «Chưa đóng»; thử lại dùng cùng `Idempotency-Key`; `409` khi đua → tải lại trạng thái, không mất ô đã nhập.
- AC8 (phân quyền). Given STUDENT mở `/attendance` hoặc `/attendance/display` → màn chặn chuẩn, **không** gọi API; TA thấy đủ nút (như TEACHER); ADMIN → màn chặn; `PUT …/settings` chỉ TEACHER (US-AS-01).
- AC9 (kiểm phản mẫu). `bash scripts/ui-antipatterns.sh` sạch; chỉ token `--ep-*`; chữ UI theo luật cứng PO (không câu giải thích; tooltip chỉ khi cần).

### Ngoài phạm vi của story này
- Máy chiếu hai màn hình / chiếu từ máy khác; chặn mở `display` trên hai máy (rủi ro T10 chấp nhận, ghi ở SRS 8).

### Phụ thuộc
- US-AS-02, US-AS-05, US-AS-06, US-AS-07; thư viện `uqr` (P-108-1 Q4); `DESIGN.md` §14.8 (Tech Lead bổ sung §14 cho `/attendance/display` và `/checkin`).

---

## US-AS-09: Sinh viên muốn quét mã bằng camera điện thoại là điểm danh xong, ngay cả khi chưa cài app, để không phải làm thêm gì
Ưu tiên: Must · Ước lượng: S · Sprint: 10.8 (L3)

### Tiêu chí nghiệm thu
- AC1 (luồng). Given sinh viên đã đăng nhập, quét QR bằng camera hệ thống → mở `https://<host>/checkin?c={courseId}#{token}` Then trang tự gửi `POST …/checkins` một lần và hiện «Đã điểm danh 07:03»; không nhập gì, không bấm gì thêm; `#token` bị xoá khỏi thanh địa chỉ ngay sau khi đọc (`history.replaceState`).
- AC2 (chưa đăng nhập). Given chưa đăng nhập When mở `/checkin` Then chuyển tới đăng nhập, **giữ** `c` và token trong `sessionStorage` (không `localStorage`) và quay lại tự gửi sau đăng nhập; xoá khỏi `sessionStorage` sau khi dùng hoặc khi quá 60 s; mã đã hết hạn lúc quay lại → «Mã đã hết hạn. Quét lại mã trên màn chiếu.»
- AC3 (khoá thiết bị web). Given trình duyệt có WebCrypto + IndexedDB Then lần đầu tạo cặp khoá ECDSA P-256 **không xuất được** (`extractable=false`), lưu trong IndexedDB (một khoá cho cả trình duyệt, **không** theo tài khoản; đăng xuất không xoá), gọi `navigator.storage.persist()`; mỗi lượt ký `"edupilot-checkin-v1|"+token+"|"+user_id` và gửi `device`; không có WebCrypto / IndexedDB (ví dụ chế độ riêng tư) → gửi không `device`, vẫn điểm danh được.
- AC4 (thông báo lỗi, một câu mỗi mã). Given `TOKEN_EXPIRED` → «Mã đã hết hạn. Quét lại mã trên màn chiếu.»; `TOKEN_INVALID` → «Mã không đúng. Quét lại mã trên màn chiếu.»; `WINDOW_CLOSED` → «Đã đóng điểm danh.»; `SELF_CHECKIN_DISABLED` → «Lớp này không điểm danh bằng mã.»; `403 NOT_IN_COURSE` → «Bạn không thuộc lớp này.»; `429` → «Thử lại sau {n} giây.»; mất mạng → «Chưa có mạng.» + `Thử lại` (còn trong hạn mã) — không mất trang, không tải lại toàn bộ.
- AC5 (lặp). Given đã điểm danh When mở lại cùng liên kết Then hiện «Đã điểm danh 07:03» (từ phản hồi `200`), không thông báo lỗi.
- AC6 (PWA). Given trình duyệt di động Then `app/manifest.ts` (tên, `display: standalone`, icon, màu lấy từ token) có mặt; **không** có service worker đệm API; trên iOS Safari cài vào Màn hình chính giữ khoá WebCrypto quá 7 ngày; **không** hiện lời mời cài đặt.
- AC7 (di động 375 px, an toàn). Given 375 px Then toàn bộ trong một màn, nút ≥ 44 px; `/checkin` chạy trong ngữ cảnh an toàn (HTTPS hoặc `localhost`); HTTP thường → vẫn điểm danh được, không khoá thiết bị (cờ `NO_DEVICE_KEY`).
- AC8 (phân quyền). Given TA / TEACHER / ADMIN mở `/checkin` → màn chặn chuẩn, không gọi API; STUDENT không thuộc lớp `c` → AC4 («Bạn không thuộc lớp này.»); `c` không phải UUID → «Liên kết không đúng.»
- AC9 (nhật ký). Given Playwright Then `#token` không xuất hiện ở bất kỳ yêu cầu HTTP nào (kiểm danh sách request: chỉ trong thân `POST`).
- AC10 (không có token → nhập mã ngắn). Given mở `/checkin` không có `#token` (từ thanh điều hướng của app, hoặc dán tay) Then hiện ô «Mã điểm danh» (6 ký tự, `autocapitalize=characters`, `autocomplete=off`, `placeholder` là ví dụ cú pháp «ABC 123») + dòng «Lớp {mã lớp}» (lớp đang chọn ở thanh trên; đổi bằng bộ chọn có sẵn) + nút `Điểm danh`; gửi `{code}` theo US-AS-03 AC3a; lỗi: `TOKEN_INVALID` → «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.»; `429` → «Thử lại sau {n} giây.»; STUDENT chưa có lớp nào → màn «Bạn chưa vào lớp nào.» chuẩn.

### Ngoài phạm vi của story này
- Quét QR trong trang web (thư viện giải mã); chế độ ngoại tuyến.

### Phụ thuộc
- US-AS-03, US-AS-04; `FEAT-account-security` (đăng nhập, `next`); Demo cần HTTPS (Q8 PM: Cloudflare Tunnel).

---

## US-AS-10: Giảng viên muốn lớp có thể kiểm tra vị trí / mạng trường khi điểm danh, tuỳ chọn và chỉ là cờ, để bắt người ngồi ở xa
Ưu tiên: Should · Ước lượng: S · Sprint: 10.8 (L4)

### Tiêu chí nghiệm thu
- AC1 (chỉ khi lớp bật). Given lớp **không** khai `geofence` / `campus_cidrs` Then `/checkin` không xin quyền vị trí, server bỏ qua mọi `geo` gửi lên (không lưu), không cờ `GEO_*` / `OFF_CAMPUS`.
- AC2 (vị trí). Given lớp khai `geofence {lat,lng,radius_m}` When lượt có `geo {lat,lng,accuracy_m}` Then server tính khoảng cách (haversine); lưu **chỉ** `geo_distance_m`, `geo_accuracy_m`; cờ `GEO_OUT` khi `distance − accuracy > radius_m`; `GEO_WEAK` khi `accuracy > 100 m`; `GEO_DENIED` khi `geo.denied=true` hoặc thiếu `geo`. Toạ độ **không** vào DB, log, `audit_log` hay phản hồi.
- AC3 (mạng trường). Given lớp khai `campus_cidrs` When IP máy khách (qua proxy tin cậy) Then lưu **chỉ** `in_campus bool`; ngoài mọi CIDR → cờ `OFF_CAMPUS`; IP không vào DB / log.
- AC4 (câu thông báo). Given geofence bật When sinh viên mở `/checkin` Then hiện đúng một dòng «Lớp này kiểm tra vị trí khi điểm danh.» **trước** khi gọi `navigator.geolocation`; từ chối quyền → vẫn gửi `geo.denied=true` và điểm danh thành công + cờ.
- AC5 (không chặn). Given mọi cờ ở AC2–AC3 Then phản hồi cho sinh viên vẫn `201` (US-AS-03 AC6).
- AC6 (nhánh lỗi). Given `geo.accuracy_m < 0` hoặc lat / lng ngoài miền → `422 VALIDATION_FAILED`; geofence khai sai → `422` ở `PUT …/settings`.
- AC7 (phân quyền). Given Staff Then xem `geo_distance_m`, `in_campus` ở danh sách lượt; STUDENT không thấy gì; `PUT` cấu hình chỉ TEACHER.

### Phụ thuộc
- US-AS-01, US-AS-03; trình duyệt cần ngữ cảnh an toàn cho Geolocation.

---

## US-AS-11: Sinh viên dùng Android muốn có app EduPilot để quét mã điểm danh ngay trong app và điểm danh với mức bảo đảm cao hơn
Ưu tiên: Must · Ước lượng: M (rủi ro → L) · Sprint: 10.8 (L5)

### Tiêu chí nghiệm thu
- AC1 (vỏ app). Given thư mục `mobile/` Then Capacitor 8 (MIT), Android, `server.url` = `APP_SERVER_URL` (cùng nguồn gốc với web; không đóng gói tĩnh; không đổi auth, không đổi route Next.js), `allowNavigation` chỉ host của ứng dụng, `android.allowMixedContent=false`, không `usesCleartextTraffic`; quyền `CAMERA` trong `AndroidManifest`; **không** intent filter `autoVerify`, **không** `assetlinks.json` (P-108-2); APK ký bằng khoá của chủ dự án (kho khoá **không** nằm trong repo); lệnh dựng, cài và cách dùng nút «Quét mã điểm danh» ở `mobile/README.md`.
- AC2 (cùng API). Given app Then gọi đúng các route của US-AS-03 / US-AS-12; không có route `app-only` cho điểm danh; web và app cho cùng kết quả khi thiếu bảo đảm (chỉ khác `assurance` / cờ).
- AC3 (quét trong WebView — P-108-2). Given app, sinh viên mở `/checkin` (không `#token`) Then có nút chính «Quét mã điểm danh»; bấm → xin quyền camera (lần đầu) → `getUserMedia({video:{facingMode:"environment"}})` + `BarcodeDetector` định dạng `qr_code` của Chromium (**không thêm thư viện** quét, không ML Kit); khi thấy QR, chỉ chấp nhận nếu `origin` = `origin` của ứng dụng, đường dẫn `/checkin`, `c` là UUID và có `#token`; khác → bỏ qua, tiếp tục quét (không báo lỗi); chấp nhận → **dừng camera ngay** (`track.stop()`), gửi `POST …/checkins {token}` thẳng (không điều hướng) và hiện «Đã điểm danh hh:mm»; rời trang / bấm `Thoát` cũng dừng camera.
- AC3a (không quét được → nhập mã ngắn). Given `BarcodeDetector` hoặc `qr_code` không có (`getSupportedFormats()`), hoặc sinh viên từ chối quyền camera, hoặc camera lỗi Then không hiện nút quét; hiện ngay ô nhập mã ngắn của US-AS-09 AC10 (kèm một dòng «Chưa cấp quyền camera.» khi bị từ chối); lượt ghi `channel = CODE`.
- AC3b (camera hệ thống). Given sinh viên đã cài app nhưng quét QR bằng camera hệ thống Then liên kết mở **trình duyệt** (không chuyển vào app); lượt là lượt web (mức 1–2; cờ `NOT_APP` nếu lớp `require=APP`); `mobile/README.md` ghi một dòng «Dùng nút Quét mã điểm danh trong app».
- AC4 (plugin ký). Given app Then trang `/checkin` phát hiện nền tảng native, gọi plugin Kotlin `EduPilotAttest` để (a) ký `"edupilot-checkin-v1|"+token+"|"+user_id` bằng khoá Keystore không xuất được (chữ ký DER), (b) đọc `Location.isMock()` khi lớp yêu cầu vị trí; **không** dùng WebCrypto trong app.
- AC5 (CSP). Given `Content-Security-Policy` của web Then không cho nguồn script ngoài tên miền ứng dụng (mọi trang trên tên miền đều gọi được plugin); `script-src` không `unsafe-eval` ở `/checkin`.
- AC6 (nhánh lỗi). Given thiếu Keystore / plugin lỗi → app điểm danh như web (mức 1, `NO_DEVICE_KEY`), không treo; WebView lỗi tải → trang lỗi chuẩn + `Thử lại`; APK sai chữ ký → xem US-AS-12 AC4; `getUserMedia` lỗi → AC3a.
- AC7 (phân quyền). Given tài khoản TA / TEACHER đăng nhập trong app → các màn như web (`/checkin` chặn theo vai); app không lưu mật khẩu / token ngoài cơ chế cookie của WebView hiện có.
- AC8 (kiểm). Có kịch bản kiểm tay trên emulator (khoá phần mềm, `boot_verified=false`) và trên máy thật nếu có (`docs/sprints/10.8/qc/android-checklist.md`); không bắt buộc máy thật để PASS story, nhưng AC mức 3 của US-AS-12 chỉ chứng minh được bằng chuỗi chứng chỉ thật (`testdata`).

### Ngoài phạm vi của story này
- iOS, Play Store, ML Kit, BLE, đóng gói tĩnh.

### Phụ thuộc
- US-AS-09; thư viện `@capacitor/*` (P-108-1 Q4); `/checkin` chạy trong ngữ cảnh an toàn (HTTPS — `getUserMedia`, WebCrypto) nhưng **không** cần tên miền cố định (P-108-2).

---

## US-AS-12: Sinh viên dùng app muốn khoá của máy được chứng thực, để lượt điểm danh đạt mức tin cậy cao nhất mà không bị phiền
Ưu tiên: Must · Ước lượng: M · Sprint: 10.8 (L5)

### Tiêu chí nghiệm thu
- AC1 (thử thách). Given app đã đăng nhập When `POST /me/devices/challenge` Then `200 {nonce}` (32 byte ngẫu nhiên, Redis `ep:dev:chal:{uid}` TTL 300 s, ghi đè nonce cũ); quá 5 lần / phút / người → `429`.
- AC2 (đăng ký). Given app sinh khoá Keystore với `setAttestationChallenge(nonce)` When `POST /me/devices {kind:"ANDROID_KEY", public_key, attestation_chain[]}` Then `GETDEL` nonce; Go (`crypto/x509` + `encoding/asn1`, **không thư viện mới**) kiểm: (i) chuỗi tới một gốc Google trong danh sách gốc nhúng sẵn; (ii) `attestationChallenge == nonce`; (iii) `attestationApplicationId` = tên gói + vân tay chứng chỉ ký APK khớp cấu hình (`ATTESTATION_PACKAGE`, `ATTESTATION_CERT_SHA256`); (iv) khoá nằm trong TEE / StrongBox; ghi `user_devices {kind, attested=true, boot_verified, security_level}`; `201 {key_id}`. Khoá đã có → `200` cùng `key_id` (idempotent).
- AC3 (danh sách thu hồi). Given đăng ký Then đối chiếu danh sách thu hồi của Google (bản tải về lưu đệm tối đa 24 giờ); chuỗi có chứng chỉ bị thu hồi → `422 ATTESTATION_FAILED`; không tải được danh sách và chưa có bản đệm → từ chối an toàn `503 ATTESTATION_UNAVAILABLE` (lượt điểm danh **không** phụ thuộc bước này, chỉ đăng ký).
- AC4 (nhánh lỗi — đăng ký). Given nonce sai / hết hạn / đã dùng → `422 DEVICE_CHALLENGE_INVALID`; chuỗi giả (tự ký) → `422 ATTESTATION_FAILED`; APK ký bằng khoá khác → `422 ATTESTATION_FAILED`; bootloader mở (`UNVERIFIED`) → **vẫn đăng ký** với `boot_verified=false` (để gắn cờ, không chặn người dùng); thân quá lớn (chuỗi > 6 chứng chỉ hoặc > 16 KB) → `422`.
- AC5 (điểm danh bằng khoá app). Given `device:{kind:"ANDROID_KEY", key_id, signature}` Then khoá đã đăng ký + chữ ký đúng + `boot_verified=true` + `geo.mock != true` (nếu có vị trí) + không cờ vị trí → `assurance = 3`, không cờ; `boot_verified=false` → `assurance=2` + cờ `DEVICE_UNTRUSTED`; `geo.mock=true` → cờ `GEO_MOCK`, `assurance ≤ 2`; `key_id` chưa đăng ký → `DEVICE_UNTRUSTED`, `assurance=1`.
- AC6 (không chặn). Given mọi lỗi ở AC5 Then điểm danh vẫn `201` (US-AS-03 AC6).
- AC7 (riêng tư). Given `user_devices` Then chỉ lưu khoá công khai + băm; không IMEI, không số serial, không nhãn người dùng (cột `label` do Staff không nhập — để trống); không gắn người.
- AC8 (phân quyền). Given STUDENT chỉ đăng ký cho **chính mình** (nonce theo `uid` từ JWT); TA / TEACHER / ADMIN → `403`; không JWT → `401`.
- AC9 (kiểm). Given chuỗi mẫu thật `caiman…` (StrongBox, `VERIFIED`, khoá) và `akita…` (TEE, `UNVERIFIED`) trong `testdata` Then `TestAttestChain` cho `boot_verified` đúng 2 trường hợp; chứng chỉ tự ký giả → `ATTESTATION_FAILED`.

### Ngoài phạm vi của story này
- Play Integrity, App Attest, kiểm thu hồi hằng ngày cho khoá đã đăng ký (nợ: kiểm một lần khi đăng ký).

### Phụ thuộc
- US-AS-04, US-AS-11; migration 00024; Redis.

---

## US-AS-13: Quản trị / chủ dự án muốn dữ liệu thô của điểm danh chỉ giữ 30 ngày, để giữ đúng luật dữ liệu cá nhân
Ưu tiên: Must · Ước lượng: S · Sprint: 10.8 (L1 + L7)

### Tiêu chí nghiệm thu
- AC1 (xoá sau 30 ngày). Given cửa sổ đóng quá `CHECKIN_RAW_RETENTION_DAYS` (30) ngày When worker chạy việc `attendance.purge` (mỗi ngày, idempotent, theo lô ≤ 1.000) Then xoá các hàng `attendance_checkins` của cửa sổ đó, rồi xoá `user_devices` không còn hàng nào tham chiếu và `created_at` quá 30 ngày; **không** đụng `attendance_records`, `checkin_windows`, `audit_log` (P-108-2, PM chốt: xoá hẳn).
- AC2 (sau khi xoá). Given đã purge Then `GET /me/attendance.checked_in_at = null` cho buổi đó, trạng thái cuối (`PRESENT` / `LATE` / `ABSENT`) còn nguyên; danh sách lượt của buổi trống; mục `CHECKIN_REVIEW` không còn.
- AC3 (an toàn khi chạy lại / đua). Given purge chạy hai lần / hai gateway Then kết quả như một; không xoá lượt của cửa sổ chưa đóng; không xoá khi cửa sổ `closed_at IS NULL`.
- AC4 (xoá tài khoản). Given quy trình xoá dữ liệu cá nhân của người dùng (`PRODUCTION_READINESS`) Then xoá ngay các hàng `attendance_checkins` của người đó (không chờ 30 ngày); thiết bị mồ côi bị dọn ở lần purge kế.
- AC5 (không lưu thô ngoài hạn). Given kiểm lược đồ Then `attendance_checkins` không có cột toạ độ, IP, tên máy; `user_devices.label` luôn NULL.
- AC6 (nhật ký). Given purge Then log chỉ có số hàng đã xoá (không `student_id`).
- AC7 (nhánh lỗi / phân quyền). Given purge lỗi giữa chừng Then lô đã xoá giữ nguyên, lô sau chạy lại; không có route HTTP kích hoạt purge (chỉ worker + lệnh `gateway admin purge-checkins` ở `APP_ENV ∈ {dev,test}`).

### Phụ thuộc
- migration 00024; `PRODUCTION_READINESS.md` §1.1 (Tech Lead thêm dòng dữ liệu).

---

## US-AS-14: QC / đội muốn seed, E2E và đo tải cho luồng điểm danh, để chứng minh F7b chạy trọn
Ưu tiên: Must · Ước lượng: M · Sprint: 10.8 (L7)

### Tiêu chí nghiệm thu
- AC1 (seed). Given `pnpm dev` seed Then có dữ liệu mẫu **không đổi** `seed/expected_crm.csv` và 270 `attendance_records` của lớp 761987 (SRS 4.10.1 FEAT-crm-360): buổi demo thuộc lớp 761988 có (a) lớp bật `self_checkin`, (b) cửa sổ vòng 1 đã đóng với ≥ 20 lượt gồm đủ các ca: không cờ, `DEVICE_SHARED` (một cặp), `DEVICE_NEW`, `NO_DEVICE_KEY`, `TOKEN_LATE`, 1 `LATE`; (c) vòng 2 đã đóng có 1 `RECHECK_MISS`; (d) `headcount` thấp hơn số lượt để có `over_headcount`; (e) không cửa sổ mở (mở tay khi demo); tài khoản `sv.gioi` điểm danh được; lệnh seed chỉ chạy khi `APP_ENV ∈ {dev,test}`.
- AC2 (E2E Playwright `attendance-checkin.spec.ts`, web). Gồm: TEACHER mở điểm danh, màn chiếu hiện QR + đếm; SV quét (mở URL) → «Đã điểm danh»; mã hết hạn; cửa sổ đã đóng; nhập mã ngắn (đúng / sai / hết hạn / cửa sổ đóng / vượt 5 lần sai); ô quét trong WebView giả lập (`BarcodeDetector` giả: QR đúng / QR lạ / không hỗ trợ → ô nhập mã); hai tài khoản cùng trình duyệt → cờ `DEVICE_SHARED` thấy ở lưới Staff, sinh viên không thấy; vòng 2 + `RECHECK_MISS`; `Đánh vắng`; `complete` bị chặn khi còn người chưa xử lý; ngắt mạng khi `Đóng`; STUDENT vào `/attendance/display` bị chặn; 375 px cho `/checkin`.
- AC3 (tải k6 `benchmarks/load/checkin.js`). Given 200 SV điểm danh trong 30 s rồi dồn 200 lượt trong 5 s Then `POST …/checkins` p95 ≤ 500 ms, lỗi 0 %, đúng 200 hàng; poll `display` 20 lớp × 1 req / 10 s không vượt SLO của `SYSTEM_DESIGN.md`; kết quả dán vào báo cáo sprint.
- AC4 (Lighthouse / axe). `/checkin` và `/attendance/display` đạt cổng của sprint 10.7 (LCP ở trạng thái «đã quay lại»; axe không lỗi nghiêm trọng).
- AC5 (luật). `bash scripts/ui-antipatterns.sh`, `go vet`, `golangci-lint`, `go test -race ./...`, test hợp đồng khớp `openapi.yaml` đều xanh; không sửa test cũ để xanh.
- AC6 (nhánh lỗi / phân quyền). Mỗi US-AS-0x có ít nhất một ca lỗi và một ca phân quyền trong bộ test (bảng truy vết SRS mục 11).

### Phụ thuộc
- US-AS-01…13.
