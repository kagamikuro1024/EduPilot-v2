# Dev — Sprint 10.8 (điểm danh tự phục vụ): L1–L5, L7

Nhánh `sprint/10.8-attendance`. Phạm vi: `techlead.md` TL-2 §4 lát L1–L5 và L7. Mục «Đợt 2» ghi các số đo và việc còn nợ.

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

## Đợt 2 (dev2, 2026-11-15)
- **k6 `checkin.js` đã chạy** (không phải stack T1 Docker đầy đủ — RAM; xem «Cách đo»): spread 200 SV / 30 s và burst 200 lượt / 5 s đều qua mọi ngưỡng. `POST …/checkins` p95 **15,6 ms** (spread) / **9,8 ms** (burst), lỗi 0 %, đúng 200 hàng mỗi lần (teardown đếm 200; DB 400 hàng sau hai lần); `display` p95 11,8 / 12,7 ms; đọc nền p95 9,6 / 10,4 ms.
  Cách đo: gateway dựng native (`go build`, `APP_ENV=dev`, `fake` LLM, `k6-stack.yml` env) + Postgres 18 / Redis 8 / MinIO trong 3 container riêng, dữ liệu `seed-t1.mjs --no-vectors` (1.000 SV · 20 lớp, script lấy từ `sprint/9.5-nfr`), MacBook Apple Silicon, **không có worker, không Caddy/TLS, không pgbouncer**. Số này là chặn dưới của đường nóng, KHÔNG thay cho số đo trên stack T1 (còn QC/PM chạy khi có `ep-t1`).
- **Sửa lỗi script k6 (lỗi L7 của dev1):** `exec.vu.idInTest` đếm cả VU của kịch bản `display` / `background` nên 15 VU cuối rơi sang lớp T1-05 (không có token) → 15 × 401. Đổi sang `exec.scenario.iterationInTest + 1`.
- **Đường quét có test tự động:** `frontend/e2e/checkin-scan.spec.ts` (`BarcodeDetector` + `getUserMedia` giả): mã khác nguồn / không phải URL bị bỏ qua, mã đúng gửi token một lần và `track.stop()`, từ chối camera → «Chưa cấp quyền camera.» + ô nhập mã.
- **Icon manifest:** `public/brand/icon-{192,512}.png` (từ `logo-edupilot-mark.svg`, nền trắng) vào `manifest.ts`.
- **Kiểm toàn bộ:** `make lint` (vet ×3 + golangci-lint ×3) 0 issue; `go test -race -tags testroutes ./...` xanh toàn bộ gói; lần chạy song song `./...` trên máy này bị timeout 10 phút ở 11 gói (máy dùng chung với `ep-qc` / `ep-t1`), chạy lại các gói đó với `-p 3` đều xanh, `attendance` xanh riêng + 3/3 lần `TestMeAttendanceCheckedAtAndTodayReview` (một lần đỏ khi quá tải, không lặp lại). `pnpm lint` 0 lỗi (5 cảnh báo cũ, ngoài 10.8), `next build:gate` xanh, Playwright `checkin` + `attendance` 65 passed / 31 skipped (theo project), `scripts/ui-antipatterns.sh` sạch.

## Sửa lỗi QC đợt 1 (`qc/defects.md`)
- **D-AS-01 (cao):** `AttFlagShared` và `AttReviewSessions` nhận `[]uuid.UUID` → qua PgBouncer (`QueryExecModeExec`) không mã hoá được → 503, SV thứ hai mất lượt. Hai truy vấn đổi sang `sqlc.arg(...)::text[]::uuid[]` (Go `[]string`, helper `idStrings` ở `attendance/staff.go`). Rà toàn repo: chỉ hai chỗ này là tham số `[]uuid.UUID` trong `internal/store` (ba chỗ còn lại là cột kết quả, giải mã không lỗi); mã tay ngoài `store` không còn mảng uuid nào gửi qua pool runtime (`seedtool` dùng `pgxpool.New` trực tiếp, không qua PgBouncer). Test: rig `attendance` chạy `QueryExecModeExec` như runtime (trên mã cũ làm đỏ `TestCheckinFlags`, `TestSpotCheckPrefersFlagged`, `TestForgetStudentKeepsOthers`, `TestSeedCheckinDemo`) + `store.TestNoUUIDSliceParams` quét mọi `*Params` trong `*.sql.go`, cấm `[]uuid.UUID` cho truy vấn sau này.
- **D-AS-02:** KHÔNG đổi mã. `httpx.DecodeJSON` (`unknownFields`, 03-AC5) trả `422 VALIDATION_FAILED details[].code=unknown` cho khoá lạ ở MỌI route của hệ thống; `createCheckin` đi đúng quy ước. Lệch nằm ở SRS FR-10 / US-AS-03 AC1, AC9 và mô tả `createCheckin` trong `openapi.yaml` («bị bỏ qua») → BA vá thành «422 `unknown`; danh tính chỉ từ JWT».
- **D-AS-03:** SRS không nêu số chứng chỉ; ARCHITECTURE 11.5 bước 1 (và code) là 2–6, tổng ≤ 16 KB. Sửa `openapi.yaml` `attestation_chain` → `minItems: 2, maxItems: 6`; thêm ca hợp đồng 1 và 7 chứng chỉ → 422 ở `checkin_scenarios_test.go`.

## Nợ / chưa làm
- **Chưa chạy:** k6 trên stack T1 thật (Docker, TLS, pgbouncer, worker); `scripts/check-checkin-seed.mjs` (cần `seed.mjs` đầy đủ trên stack demo; phần logic đã có `TestSeedCheckinDemo`); Lighthouse `/checkin` (đã khai ở `lighthouserc.json`, chạy ở CI); kiểm tay emulator `android-checklist.md` (QC; **không có máy Android thật** — P-108-12).
- `attest` chưa chạy với emulator (chỉ gốc phần mềm ⇒ bước 4 hỏng; P-108-5).
- `PRODUCTION_READINESS` §3: job xoá tài khoản (phase PR) phải gọi `attendance.ForgetStudent`.
- Production compose / secret của PR phải đặt `ATTENDANCE_TOKEN_KEY`, `ATTESTATION_PACKAGE`, `ATTESTATION_CERT_SHA256` khác rỗng.
