# Dev — Sprint 10.8 (điểm danh tự phục vụ): L1–L5, L7 (chưa đo)

Nhánh `sprint/10.8-attendance`. Phạm vi: `techlead.md` TL-2 §4 lát L1–L5 và L7 (mã). Chưa chạy: k6, E2E trên stack thật, Lighthouse `/checkin`, kiểm tay emulator.

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

## Frontend + app (L2–L5)
- **L4-UI** `features/members/AttendanceSection.tsx` (mục «Điểm danh bằng mã» ở `/class/settings`): lưu tự động cả chính sách, 422 tại ô, TA chỉ xem. Chỉ dựng khi máy chủ trả `attendance.self_checkin` là boolean.
- **L2** `/attendance`: `checkinApi.ts`, `CheckinWindowLine.tsx`, tích hợp ở `RealAttendance.tsx` (ô trống «Chưa điểm danh», chip cờ, «Đã xem», lọc `filter=review`, đóng offline «Chưa đóng» tự thử lại, 422 `CHECKIN_UNRESOLVED`). `useAttendance({blank})`: hàng chưa có dòng không gửi khi lưu.
  Màn chiếu `app/(present)/attendance/display/` (`features/attendance/display/`): `uqr` ECC L; hỏi `…/display` một lần mỗi bước theo `server_time`; mất mạng giữ mã tới `step_expires_at + rotate_seconds` rồi «Mất kết nối». **Lớp lấy từ `?course=` trong URL**: `useCourseDeepLink` bỏ qua route này (không đổi lớp đang chọn, không bỏ tham số); lớp ngoài tài khoản → lỗi rõ, 0 yêu cầu.
- **L3** `/checkin` (`app/(checkin)/`, `features/checkin/`): layout cất `{c, token, at}` vào `sessionStorage` TRƯỚC `AuthGate`; khoá WebCrypto `edupilot.device.v1` (`extractable=false`); mã ngắn chuẩn hoá như máy chủ; geo sau câu thông báo; 429 đếm ngược; `manifest.ts` (màu từ `shared/styles/brand.ts`); `/me` «Điểm danh lúc». **CSP + Permissions-Policy** trong `next.config.ts` (chỉ khi không phải `next dev`; `NEXT_PUBLIC_BLOB_ORIGIN` khai origin kho tệp khác origin ứng dụng — `build:gate` đặt `http://blob.test`).
- **L5** `mobile/` (Capacitor 8.6.2, `vn.edupilot.app`): plugin Kotlin `EduPilotAttest` (`registerDevice`, `sign`, `isMockLocation`), `APP_SERVER_URL` đóng vào APK, `errorPath` có «Thử lại», quyền CAMERA + ACCESS_FINE_LOCATION, không `autoVerify`, không cleartext, `allowBackup=false`. `./gradlew assembleRelease` xong trên Mac; `apksigner verify --print-certs` SHA-256 = vân tay kho khoá dev (`~/.edupilot/edupilot-dev.jks`, ngoài repo). Frontend gọi plugin qua `window.Capacitor.nativePromise` (`features/checkin/nativeDevice.ts`); đăng ký lỗi ⇒ ký Keystore, gửi `WEB_KEY` (mức 2).
- **L7** seed `SeedCheckinDemo` (22 lượt vòng 1 đủ ca + LATE, vòng 2 + `RECHECK_MISS`, `over_headcount`, `sv.gioi` không có lượt; idempotent) gọi từ `scripts/seed.mjs`; `scripts/check-checkin-seed.mjs`; `benchmarks/load/checkin.js` (spread / burst, 20 màn chiếu, đọc nền, đếm 200 hàng ở teardown).

## Đối chiếu bảng truy vết SRS §11 (tên test thật; cột Test của BA dùng tên chưa có)
| US | Tên trong SRS | Test thật |
| --- | --- | --- |
| 01 | `TestSettingsAttendance*` | `TestSettingsAttendance` (+ contract `checkin_scenarios_test.go`) |
| 02 | `TestWindow*`, `TestRound*`, `TestDisplayNoStore`, `TestSweepAutoClose` | `TestCheckinFlow` (mở / màn chiếu / `no-store`), `TestRoundsRecheckAndLimit`, `TestSweepAutoCloseAndLate` |
| 03 | `TestToken*`, `TestCheckin*`, `TestMeAttendanceCheckedAt` | `TestTokenRoundTripAndTamper`, `TestTokenStepWindow`, `TestShortCodeNormalize`, `TestCheckinFlow`, `TestMeAttendanceCheckedAtAndTodayReview` |
| 04 | `TestFlag*`, `TestAssuranceLevels`, `TestDeviceSharedRace` | `TestCheckinFlags`, `TestGeoFlagsAndAssurance` |
| 05 | `TestClose*`, `TestAbsentUnchecked`, `TestComplete*` | `TestCloseAbsentComplete` |
| 06 | `TestCheckinsListCursor`, `TestClearFlagsAudit`, `TestTodayCheckinReview` | `TestCheckinsListCursor`, `TestCheckinFlow` (clear-flags), `TestMeAttendanceCheckedAtAndTodayReview` |
| 07 | `TestOverHeadcount`, `TestFlagRecheckMiss`, `TestSpotCheck` | `TestRoundsRecheckAndLimit` (RECHECK_MISS), `TestSeedCheckinDemo` (over_headcount), `TestSpotCheckPrefersFlagged` |
| 08, 09 | spec Playwright | `attendance-window.spec.ts`, `attendance-display.spec.ts`, `attendance-checkin.spec.ts`, `checkin-code.spec.ts`, `checkin-token-leak.spec.ts`, `checkin-native.spec.ts` |
| 10 | `TestFlagGeo`, `TestFlagCampus`, `TestNoCoordsOrIPStored` | `TestGeoFlagsAndAssurance`, `TestLogsNoCheckinSecrets` |
| 11 | `android-checklist.md` | QC; `checkin-native.spec.ts` (cầu nối giả) |
| 12 | `TestAttest*`, `TestDeviceChallengeOnce` | `TestAttestRealChains`, `TestAttestRejects`, `TestAttestRevocation`, `TestDeviceChallengeOnce` |
| 13 | `TestPurge*`, `TestAccountDeleteRemovesCheckins` | `TestPurgeKeepsRecords`, `TestForgetStudentKeepsOthers` |

## Nợ / chưa làm
- **Chưa chạy trên stack thật:** k6 `checkin.js` (cần stack T1 kèm `k6-stack.yml`), E2E F7b trọn trên stack (các spec Playwright dùng gateway giả theo hợp đồng thật), `check-checkin-seed.mjs`, Lighthouse / axe `/checkin` và `/attendance/display`, p95 «0 thao tác → Đã điểm danh ≤ 1,5 s».
- Icon manifest chỉ có SVG (chưa PNG 192 / 512). Kịch bản kiểm tay emulator `android-checklist.md` thuộc QC.
- Đường quét (`getUserMedia` + `BarcodeDetector`) chưa có test tự động (Chromium headless không có detector): chỉ có `parseCheckinQr` và kiểm tay.
- Playwright trên bản dựng có CSP: máy này chạy documents, a11y, class-join, checkin*, attendance*, panels, frame student; phần còn lại để CI.
- `attest` chưa chạy với emulator (chỉ gốc phần mềm ⇒ bước 4 hỏng; P-108-5). `go test -race` toàn bộ để CI.
- `PRODUCTION_READINESS` §3: job xoá tài khoản (phase PR) phải gọi `attendance.ForgetStudent`.
