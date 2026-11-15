# Dev — Sprint 10.8 (điểm danh tự phục vụ): L1 + L5-Go

Nhánh `sprint/10.8-attendance`. Phạm vi: `techlead.md` TL-2 §4 lát L1 (backend lõi) và phần Go của L5. `mobile/` (plugin Android), L2/L3/L4 (frontend) và L7 chưa làm.

## Đã làm
- Migration `00024_attendance_checkin.sql` (3 bảng + 2 enum, có Down). Truy vấn `internal/store/queries/attendance_checkin.sql` (sqlc sạch).
- `internal/attendance`: token HMAC xoay theo bước (`K_w` suy từ `ATTENDANCE_TOKEN_KEY`), mã ngắn Crockford, cửa sổ (mở / màn chiếu / đếm tay / đóng / quét quá hạn / gọi tên / đánh vắng), `Checkin` (thứ tự kiểm FR-10, cờ, mức tin cậy 1–3, vị trí, mạng trường, đếm mã sai 5 lần → 429), danh sách lượt theo cursor, bỏ cờ + audit, `PurgeCheckins`, `ForgetStudent` (P-108-10), `SweepJob` + `PurgeJob` (worker).
- `internal/attendance/attest`: Key Attestation đủ 10 bước ARCHITECTURE 11.5 bằng `crypto/x509` + `encoding/asn1`; `roots.pem` nhúng; danh sách thu hồi Redis (`ep:att:revoked`, khoá nạp `ep:lock:att.revoked`, TTL `ATTESTATION_REVOCATION_TTL_H`), nonce `ep:dev:chal:{uid}` GETDEL. Test dùng 3 chuỗi thật của github.com/android/keyattestation (`testdata/README.md`).
- 12 route (`internal/httpapi/attendancehttp`), mở rộng `PUT/GET /courses/{id}` (`attendance`; sinh viên chỉ `self_checkin`, `require`, `geo_check`), `GET …/me/attendance` (`checked_in_at`), `PutGrid` 422 `CHECKIN_UNRESOLVED {count}` (thay bước «điền PRESENT» chỉ khi buổi có cửa sổ), mục «Hôm nay» `CHECKIN_REVIEW` (bậc 13; một cột thêm vào truy vấn `TodayStaffPending`, không thêm truy vấn), lệnh dev `gateway admin seed-checkin-demo --course <mã>` / `purge-checkins`.
- Contract: 12 thao tác + `CourseDetail.attendance` khớp `openapi.yaml` (BA đã viết); `checkin_scenarios_test.go` gọi mọi status; miễn trừ có lý do cho `POST /me/devices` 200 / 201 / 503 (cần chuỗi chứng thực thật còn hạn) ở `exempt.go`.

## Quyết định đáng nhớ
- `ATTENDANCE_TOKEN_KEY`: production BẮT BUỘC (32 byte base64); dev / test có giá trị cố định (`config.devAttendanceKey`) để compose / test không phải đặt; tắt `ATTENDANCE_SELF_CHECKIN_ENABLED` thì không đòi. Đã thêm vào `.env.example`, `docker-compose.local.yml`. **Production compose / secret của PR phải đặt khoá khác.**
- `ATTESTATION_PACKAGE` + `ATTESTATION_CERT_SHA256` rỗng ⇒ mọi đăng ký thiết bị `ATTESTATION_FAILED` (chưa cấu hình = không tin ai). Phải đặt khi dựng APK thật (L5 mobile).
- Đường hot `POST …/checkins` KHÔNG BAO GIỜ từ chối vì tín hiệu (chỉ gắn cờ); phản hồi chỉ `{checked_in_at, round}`. `DEVICE_SHARED` tuần tự hoá bằng khoá hàng `user_devices`; cờ gắn cả lượt trước của người dùng chung máy. Lượt đầu với khoá mới KHÔNG bị `DEVICE_NEW`.
- `Close` lần hai trả kết quả tính lại từ DB, không ghi / audit / outbox. Cửa sổ KHÔNG bị purge (chỉ lượt thô + thiết bị mồ côi), để giữ số đếm tay / sĩ số.
- `WindowOut` không có `session_id` / `rotate_seconds` ở JSON (đúng `CheckinWindow`); `POST …/checkin-windows` trả riêng `rotate_seconds`.
- `GET …/checkin-windows` trên buổi không có thật → 404; mọi thao tác ghi của Staff trên lớp lưu trữ → 409 `COURSE_ARCHIVED`.
- `attest.Verifier.fail` ghi một dòng cảnh báo chỉ có TÊN BƯỚC (`step`), không nonce / chứng chỉ / người dùng.

## Đề xuất mới
- **P-108-11** (`proposals.md`): `apiClient` coi `TOKEN_EXPIRED` / `TOKEN_INVALID` là mã xác thực ⇒ QR quá hạn của sinh viên sẽ refresh vô ích / đăng xuất; đề nghị mã riêng `CHECKIN_TOKEN_*`. Cũng ghi: người ngoài lớp nhận 403 (guard), không 404 như SRS.

## Test đã sửa vì hợp đồng đổi (không nới ngưỡng)
- `db/migrations_test.go`: version 24. `crm/me_test.go`: phần tử `sessions` có 4 khoá (thêm `checked_in_at`, FR-17). `config_test.go` + `cmd/gateway/serve_test.go`: env production thêm `ATTENDANCE_TOKEN_KEY`. `contract_test.go`: 331 thao tác.

## Nợ / chưa làm
- L2 (màn chiếu), L3 (`/checkin`), L4-UI (cấu hình lớp, màn Staff), L5 mobile (Capacitor, plugin Keystore), L7 (E2E, k6 `checkin.js`, axe). `PRODUCTION_READINESS` §3: job xoá tài khoản (phase PR) phải gọi `attendance.ForgetStudent`.
- `attest` chưa chạy với emulator (chỉ gốc phần mềm ⇒ bước 4 hỏng; P-108-5).
- Chưa đo `go test -race` toàn bộ trên máy này (CI làm).
