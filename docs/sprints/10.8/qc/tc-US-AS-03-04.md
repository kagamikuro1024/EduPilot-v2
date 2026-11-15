# QC test case — US-AS-03 (điểm danh bằng mã / mã ngắn), US-AS-04 (cờ, thiết bị, mức tin cậy)
Nguồn: US.md + SRS §3.1, 3.5, 4.2–4.5, 5, 6 #11, #14. Hộp đen; quy ước: `tc-00-chung.md`. `T`=token lấy từ `display.qr_url` sau `#`, `CODE`=`display.short_code`, `K`=khoá ECDSA do QC sinh, `SIG(msg)`=chữ ký. Thông điệp ký: web `"edupilot-checkin-v1|"+T+"|"+user_id`; mã ngắn `"edupilot-checkin-v1|code:"+CODE_CHUẨN_HOÁ+"|"+user_id`. `W`=cửa sổ đang mở, `rotate_seconds=5` ở lớp QC-AS.

## US-AS-03 Điểm danh bằng token / mã ngắn
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS03-01 | AC1 | `W` mở; `sv.gioi` ACTIVE | `POST /courses/{C}/checkins {token:T}` | `201 {checked_in_at, round:1}`; đúng một hàng `attendance_checkins` (`student_id`=`sv.gioi`, `channel='WEB'`) | API+DB |
| TC-AS03-02 | AC1 | – | Thân có cả `token` và `code`; thân rỗng; thân có `student_id` của người khác | Hai ca đầu `422 VALIDATION_FAILED`; ca thứ ba: `201` nhưng hàng ghi cho **người gọi** (JWT), `student_id` thân bị bỏ qua | API+DB |
| TC-AS03-03 | AC2 | – | Bước hiện tại và bước trước (gửi T cũ ≤ 1 bước = ≤ 5 s) | Cả hai `201` | API |
| TC-AS03-04 | AC2, AC3 | – | T cũ hơn 2 bước (chờ 12 s), cùng cửa sổ | `422 CHECKIN_TOKEN_EXPIRED` (không phải `TOKEN_EXPIRED`) | API |
| TC-AS03-05 | AC3 | – | Đổi 1 ký tự MAC; `window_id` lạ (UUID ngẫu nhiên); `window_id` của **lớp khác** nhưng MAC đúng; định dạng sai (thiếu phần, không base32) | Mỗi ca `422 CHECKIN_TOKEN_INVALID`; **cùng thân và độ dài phản hồi** giữa «cửa sổ lạ» và «MAC sai» (không lộ cửa sổ có tồn tại) | API |
| TC-AS03-06 | AC3 | Cửa sổ đã đóng, MAC đúng, trong bước | Gửi T | `409 WINDOW_CLOSED` (đóng bằng `close`; và đóng do quá `closes_at`) | API |
| TC-AS03-07 | AC3 | – | Thứ tự kiểm: (a) không JWT + token hỏng; (b) STUDENT ngoài lớp + token hỏng; (c) lớp tắt `self_checkin` + token hỏng; (d) token đúng cú pháp, cửa sổ đã đóng, MAC sai | (a) `401`; (b) `403` (trước cú pháp); (c) `409 SELF_CHECKIN_DISABLED`; (d) `422 CHECKIN_TOKEN_INVALID` (MAC trước đóng) | API |
| TC-AS03-08 | AC3a | `W` mở, UI/API cho SV | `POST … {code:CODE}` | `201`; `channel='CODE'` | API+DB |
| TC-AS03-09 | AC3a | – | Chuẩn hoá: `code` viết thường, có khoảng trắng / gạch giữa nhóm 3+3, thay `0→O`, `1→I`/`L` | Cùng kết quả như mã chuẩn (`201`) với ký tự `O→0`, `I`/`L→1` | API |
| TC-AS03-10 | AC3a | Lớp có hai cửa sổ: vòng 1 vừa đóng (< 15 phút), vòng 2 đang mở; cộng một cửa sổ đang mở ở **lớp khác** | Gửi mã ngắn của vòng 2; gửi mã ngắn của vòng 1 (còn trong bước hiện tại / trước); gửi mã ngắn của cửa sổ lớp khác | Mã vòng 2 → `201` ở vòng 2; mã vòng 1 (cửa sổ đã đóng) → `409 WINDOW_CLOSED`; mã của lớp khác hoặc không khớp → `422 CHECKIN_TOKEN_INVALID` (dò chỉ trong cửa sổ của lớp `{C}`) | API |
| TC-AS03-11 | AC3a | – | Mã ngắn đúng nhưng bước đã hết hạn (chờ 12 s) | `422 CHECKIN_TOKEN_INVALID` (không phân biệt hết hạn / sai) | API |
| TC-AS03-12 | AC3b | `sv.kha` | 5 lần nhập sai; lần thứ 6 sai; rồi **nhập đúng** trong cùng phút | 5 lần đầu `422`; lần 6 `429` + `Retry-After`; lần đúng trong phút đó vẫn `429`; sau hết phút (60 s) lần đúng `201`; Redis `ep:att:codefail:{uid}` TTL ≤ 60 | API |
| TC-AS03-13 | AC3b | `sv.kha` đã điểm danh đúng 3 lần ở 3 cửa sổ (không bị đếm) | 5 lần sai liên tiếp | Bộ đếm chỉ đếm lần **sai** (`429` ở lần sai thứ 6, không phải lần thứ 4); đường `token` sai nhiều lần không tăng bộ đếm này | API |
| TC-AS03-14 | AC4 | `sv.gioi` đã điểm danh | Gửi lại token / mã (F5, hai tab song song 10 yêu cầu) | `200` cùng `{checked_in_at, round}` của lượt đầu; `SELECT count(*)` = 1; cờ không đổi (`flags` giữ nguyên) | API+DB |
| TC-AS03-15 | AC4 | Vòng 1 đóng, vòng 2 mở | Cùng SV điểm danh vòng 2 | `201`, hàng thứ hai với `window_id` vòng 2 | API+DB |
| TC-AS03-16 | AC5 | Một SV lượt 0 cờ (có chữ ký web hợp lệ) và một SV lượt nhiều cờ (không chữ ký, ngoài vùng, ...) | So phản hồi hai SV, bỏ `checked_in_at` | Thân, mã, độ dài **giống hệt**; không có `flags`, `assurance`, `device`, `geo` ở bất kỳ route SV | API |
| TC-AS03-17 | AC6 | Bảng ca: không chữ ký / chữ ký sai / `device` hỏng / máy dùng chung / ngoài vùng / ngoài mạng / `geo.mock=true` / `geo.denied=true` / `app.version` lạ | Mỗi ca một SV | Luôn `201`; lượt ghi kèm cờ tương ứng (kiểm DB) | API+DB |
| TC-AS03-18 | AC7 | Sau điểm danh | `GET /courses/{C}/me/attendance` (SV) | Buổi tương ứng có `checked_in_at`; không `note`, cờ, mức tin cậy; buổi chưa điểm danh `null`; tool AI `get_my_attendance` (hỏi chat «tôi điểm danh lúc mấy giờ?») **không** trả `checked_in_at` (schema tool không đổi) | API |
| TC-AS03-19 | AC8 | – | Lớp tắt `self_checkin`; lớp lưu trữ; `geo.lat` kiểu chuỗi; `signature` không phải base64 | `409 SELF_CHECKIN_DISABLED`; `409 COURSE_ARCHIVED`; `422 VALIDATION_FAILED`; `422 VALIDATION_FAILED` | API |
| TC-AS03-20 | AC8 | – | 11 lượt trong 60 s (cùng SV, token đúng, cửa sổ khác nhau không khả thi → dùng token sai) | Từ lượt 11 `429` + `Retry-After` (đường `token`) | API |
| TC-AS03-21 | AC9 | – | Không JWT; TA, TEACHER, ADMIN gọi `POST …/checkins`; `sv.moi` (ngoài lớp); SV `PENDING`; SV đã bị mời ra | `401`; `403`; `403`; `403` `reason=course`; `403` `reason=course`; `403` `reason=course` (không `404`, không `NOT_IN_COURSE`, không `500`) | API |
| TC-AS03-22 | AC9 | – | Mời SV ra khỏi lớp **đúng lúc** đang `POST …/checkins` (đua; 30 vòng) | Không bao giờ `500`; kết quả `201` hoặc `403` | API |
| TC-AS03-23 | AC10 | Sau toàn bộ TC của US | `docker logs ep-qc-gateway-1 \| grep -E 'public_key\|signature\|"token"\|token='` | 0 dòng; không toạ độ; không IP máy khách | LOG |
| TC-AS03-24 | – | `apiClient` (web) | SV điểm danh bằng token đã hết hạn qua giao diện (`/checkin`) | Trình duyệt **không** gọi `/auth/refresh` và **không** bị đăng xuất; hiện «Mã đã hết hạn. Quét lại mã trên màn chiếu.» (P-108-11a) | E2E |
| TC-AS03-25 | – | – | Đọc `openapi.yaml` + mã lỗi | `CHECKIN_TOKEN_INVALID`, `CHECKIN_TOKEN_EXPIRED` có trong spec; `TOKEN_EXPIRED` / `TOKEN_INVALID` (xác thực) **không** dùng cho route điểm danh | DIFF |

## US-AS-04 Cờ, thiết bị, mức tin cậy
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS04-01 | AC1 | `K` mới | `POST … {token:T, device:{kind:"WEB_KEY", public_key:SPKI_b64, signature:SIG(msg)}}` (chữ ký `r‖s` 64 byte) | `201`; `user_devices` có hàng `key_sha256=sha256(public_key)`, lượt có `device_id`; `assurance=2`; không cờ (lớp không bật vị trí) | API+DB |
| TC-AS04-02 | AC1 | – | Chữ ký **DER** hợp lệ trên cùng thông điệp | `201`, `assurance=2` (chấp nhận hai dạng) | API+DB |
| TC-AS04-03 | AC1 | – | Đường mã ngắn: ký `"edupilot-checkin-v1|code:"+CODE+"|"+uid` | `201`, `assurance=2`; ký nhầm thông điệp token → `DEVICE_UNTRUSTED` | API+DB |
| TC-AS04-04 | AC2 | – | `\d user_devices` + liệt kê bảng | **Không có cột `user_id`**; không bảng liên kết thiết bị–người; `label` luôn NULL (`CHECK`) | DB |
| TC-AS04-05 | AC3 | – | Không gửi `device` | Lượt ghi cờ `NO_DEVICE_KEY`, `assurance=1` | API+DB |
| TC-AS04-06 | AC3 | – | `device` có chữ ký sai (đổi 1 byte) / `public_key` hỏng / khoá khác | `DEVICE_UNTRUSTED`, `assurance=1`; vẫn `201` (AC9) | API+DB |
| TC-AS04-07 | AC4 | Hai SV (`sv.gioi`, `sv.kha`) cùng lớp dùng **cùng `K`** | `sv.gioi` điểm danh trước, `sv.kha` sau (cùng buổi) | Lượt `sv.kha` có `DEVICE_SHARED` **và** lượt `sv.gioi` được thêm `DEVICE_SHARED` (Staff thấy cả hai) | API+DB |
| TC-AS04-08 | AC4, AC9 | Hai SV cùng `K`, bắn song song 30 vòng (buổi mới) | `POST` đồng thời | Cả hai lượt luôn có `DEVICE_SHARED` (không mất do đua) | API+DB |
| TC-AS04-09 | AC4 | Cùng máy nhưng **hai lớp khác nhau** | SV lớp A và SV lớp B cùng `K` | Không `DEVICE_SHARED` (so trong phạm vi lớp; chỉ mục `(course_id, device_id)`) | API+DB |
| TC-AS04-10 | AC5 | `sv.gioi` đã có lượt bằng `K1` | Điểm danh buổi sau bằng `K2` (lần đầu `K2`); rồi buổi sau nữa bằng `K2` | Lượt `K2` đầu: `DEVICE_NEW`; lượt `K2` sau: không còn; lượt **đầu đời** của SV bằng `K1`: không `DEVICE_NEW` | API+DB |
| TC-AS04-11 | AC6 | Lớp `rotate_seconds=60` (ngưỡng `TOKEN_LATE` = 65 s) | Lấy T lúc đầu bước; gửi lúc tuổi mã ≈ 30 s (cùng bước) và ở ca khác lúc tuổi ≈ 70 s (bước trước, vẫn còn hiệu lực) | Tuổi ≈ 30 s: không cờ; tuổi ≈ 70 s (> `(60+5)×1000` ms): `TOKEN_LATE`, vẫn `201`; `token_age_ms` trong DB khớp (±1 s) | API+DB |
| TC-AS04-12 | AC7 | – | Các ca: không chữ ký (1); `WEB_KEY` hợp lệ (2); `WEB_KEY` hợp lệ + `GEO_OUT` (1) | `assurance` đúng bảng SRS 4.5 | API+DB |
| TC-AS04-13 | AC3, AC7 | Lớp `require=APP`; khoá app giả lập `attested=true` (Q-S108-2) | (a) `WEB_KEY` hợp lệ; (b) `ANDROID_KEY` chữ ký đúng, `boot_verified=true`; (c) `ANDROID_KEY` `boot_verified=false`; (d) `ANDROID_KEY` `key_id` chưa đăng ký | (a) `NOT_APP`, mức 2; (b) không `NOT_APP`, mức 3; (c) mức 2 + `DEVICE_UNTRUSTED`, **không** `NOT_APP` (P-108-6); (d) `DEVICE_UNTRUSTED` + `NOT_APP`, mức 1. Lớp `require=WEB`: không bao giờ `NOT_APP` | API+DB |
| TC-AS04-14 | AC7 | – | `app:{version:"x"}` và `channel` tự khai | Không đổi quyết định cờ / mức (kênh `APP` do máy khách khai không là căn cứ `NOT_APP`) | API+DB |
| TC-AS04-15 | AC8 | – | Staff `GET …/sessions/{S}/checkins` | Mỗi cờ `{code, label}` đúng SRS 4.4.1 («Cùng máy với bạn khác», «Máy mới», «Không xác minh được máy», «Máy không đáng tin», «Quét mã trễ», …); `label` không chứa `token`, `HMAC`, `attestation`, `key`, `device` (regex, không phân biệt hoa thường) | API |
| TC-AS04-16 | AC9 | – | `public_key` là chuỗi rác; IndexedDB bị chặn (Playwright, chế độ riêng tư) | `201` + `NO_DEVICE_KEY` hoặc `DEVICE_UNTRUSTED`, không lỗi | API+E2E |
| TC-AS04-17 | AC10 | – | `GET` mọi route SV (`me/attendance`, `GET /courses/{C}`, `me/today`, phản hồi `checkins`); STUDENT gọi `GET …/checkins`; ADMIN | Không `flags`, `assurance`, `device_id` ở route SV; `GET …/checkins` STUDENT/ADMIN `403` | API |
| TC-AS04-18 | – | – | `\d attendance_checkins` | Có `CHECK (flags <@ ARRAY[…12 mã…])`; chỉ mục `(course_id, session_id, created_at, id)`, `(course_id, device_id) WHERE device_id IS NOT NULL`, `(course_id, student_id, created_at)`; `UNIQUE (course_id, window_id, student_id)`; không cột toạ độ / IP | DB |
