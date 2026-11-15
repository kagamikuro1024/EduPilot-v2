# Sprint 10.8: kênh Tech Lead

## TL-1 (sprint 10.8): điểm danh tự phục vụ chống gian lận, web + app điện thoại

PM giao (2026-11-14), có bổ sung của chủ dự án: **sẽ có app điện thoại**. Cần:
- mối đe doạ;
- so sánh cơ chế;
- thiết kế nhiều lớp (chỉ gắn cờ, giảng viên sửa tay + audit);
- kiến trúc theo luật repo;
- so sánh PWA / Capacitor / React Native / native;
- PoC;
- chia lát + câu hỏi cho chủ dự án.

Bản đầy đủ có nguồn: `docs/research/2026-11-14-attendance-anti-cheat.md`. PoC: `docs/research/poc/attendance-checkin/` (`go run .`).

**TL trả lời:**

1. **Thiết kế: 5 lớp, chỉ gắn cờ.**
   - QR xoay vòng 10 s. Mã là `window_id.step.HMAC`, khoá suy ra từ `ATTENDANCE_TOKEN_KEY`, không lưu trạng thái, nhận bước hiện tại + bước trước.
   - Khoá thiết bị ký từng lượt. Web dùng WebCrypto P-256 không xuất được. App dùng Keystore có Key Attestation.
   - Vị trí / IP trường: tuỳ chọn theo lớp.
   - Đối chiếu do người làm: số đầu người, vòng 2 ngẫu nhiên, gọi tên ngẫu nhiên.
   - Cờ bất thường (§4.3 của research). Sinh viên không bao giờ thấy cờ.

   Chỉ từ chối khi: mã sai / hết hạn (`422`), cửa sổ đóng (`409`), không thuộc lớp (`403`).
2. **Không làm** selfie / khuôn mặt (sinh trắc, Luật 91 Đ31 k4). **Không làm** chặn chụp màn hình: QR nằm trên máy chiếu nên vô ích. **Không** dùng NFC: 200 người phải xếp hàng chạm.
3. **Lược đồ: `00024_attendance_checkin.sql`.**
   - Chỉ `CREATE`: `checkin_windows`, `user_devices` (không gắn user), `attendance_checkins` (UNIQUE `(course_id, window_id, student_id)`, index đầu `course_id`, không lưu toạ độ / IP).
   - Chính sách lớp nằm ở `courses.settings.attendance`, qua `PUT …/settings` có sẵn.
   - Đóng cửa sổ chạy một câu `INSERT … ON CONFLICT DO NOTHING` vào `attendance_records`. Không ghi đè dòng giảng viên đã sửa, nên lưới tay và khoá `version` giữ nguyên.
4. **Đường nóng không cần Redis.** Kiểm mã bằng HMAC, chữ ký ký trên mã tươi (không cần nonce), khử trùng bằng UNIQUE. Redis chỉ giữ nonce đăng ký thiết bị app (`ep:dev:chal:{uid}`, `GETDEL`).
   - PoC: 200 lượt kiểm mã + chữ ký song song mất 1,7 ms, đo trên MacBook Apple Silicon.
   - Màn chiếu poll `display` mỗi bước (1 req / 10 s) thay cho SSE, vì SSE cần bộ đếm nhịp theo cửa sổ trên N gateway.
5. **App: Capacitor tải web từ máy chủ (`server.url`), Android trước, APK sideload.**
   - Tái dùng 100% Next.js + `shared/`. Không đổi auth vì cookie vẫn cùng origin.
   - Cộng thêm: khoá phần cứng có Key Attestation (Go kiểm bằng `crypto/x509` + `encoding/asn1`; PoC đọc đúng `VERIFIED` / `UNVERIFIED` trên chuỗi chứng chỉ thật) và `Location.isMock()`.
   - Không chọn đóng gói tĩnh: phải đổi 9 route động và auth cross-origin (L, luật DỪNG). Không chọn React Native / native: phải viết lại UI.
   - iOS dùng web/PWA, vì App Attest cần gói 99 USD/năm.
   - Play Integrity không cần: Key Attestation miễn phí, không có hạn mức.
6. **Lát:** chi tiết ở §8 research.

   | Lát | Nội dung | Cỡ |
   | --- | --- | --- |
   | L1 | Backend lõi | L |
   | L2 | Màn giảng viên + màn chiếu | M |
   | L3 | Màn sinh viên web + PWA | S |
   | L4 | Geofence / CIDR | S |
   | L5 | App Android + Key Attestation | M, rủi ro → L |
   | L6 | BLE (tuỳ chọn, cắt được) | M |
   | L7 | E2E + k6 + seed | M |

**Rủi ro:**
- **Demo cần HTTPS cho điện thoại.** WebCrypto, Geolocation và camera chỉ chạy trong ngữ cảnh an toàn. Android App Links cũng cần tên miền HTTPS xác minh được (Q8).
- Báo nhầm `DEVICE_NEW` khi sinh viên xoá dữ liệu trình duyệt.
- Ngưỡng `TOKEN_LATE` cần chỉnh theo dữ liệu thật.
- Gradle / WebView có thể tốn thời gian hơn dự kiến ở L5.

**→ PM (chủ dự án quyết):** Q1–Q9 ở §9 research.
- Q1: SV chưa check-in → `ABSENT`?
- Q2: lớp chọn "chỉ app" thì lượt từ web ghi + cờ hay từ chối?
- Q3: đổi máy chỉ gắn cờ hay phải duyệt?
- Q4: duyệt `uqr` (MIT), Capacitor 8 (MIT), ML Kit dự phòng, BLE nếu làm L6.
- Q5: geofence + câu thông báo dùng vị trí.
- Q6: iOS 99 USD/năm?
- Q7: Play Console 25 USD?
- Q8: HTTPS cho demo.
- Q9: thời hạn giữ dữ liệu.

Đề xuất `P-108-1` ở §11 research.

## TL-2 (sprint 10.8): tài liệu thiết kế + chia lát cho FEAT-attendance-selfcheck

PM giao (2026-11-15). Cần:
1. `DESIGN.md` §14 cho `/checkin` và `/attendance/display`, cập nhật §14.8 theo khung §23, nút chính dính đáy theo P-107-14;
2. `ARCHITECTURE.md`: migration 00024, khoá Redis / TTL, HMAC, kiểm Key Attestation bằng stdlib, không service mới;
3. `PRODUCTION_READINESS.md` §1.1: xoá dữ liệu thô sau 30 ngày (P-108-2), job purge;
4. chia lát L1–L5, L7 với acceptance đo được.

Đầu vào: SRS / US v1.1, QUESTIONS, P-108-1, P-108-2, research 2026-11-14, §23 trên `sprint/10.7-ui-r`.

**TL trả lời:**

### 1. Đã vá trên nhánh `sprint/10.8-attendance`

| Tệp | Mục | Nội dung |
| --- | --- | --- |
| `docs/design/DESIGN.md` | §14.8 (thêm phần «Self check-in»), §14.8.1 `/attendance/display`, §14.8.2 `/checkin`, §14.9 (một dòng) | Nút đầu trang, dòng trạng thái cửa sổ trong panel, hàng lưới có cờ, cài đặt lớp; màn chiếu ngoài khung + 3 token chữ chiếu; `/checkin` 9 trạng thái, nút chính dính đáy dưới 720 px |
| `docs/ARCHITECTURE.md` | §2, §3, §4 (một dòng), §5, §7, §8 (một dòng mỗi chỗ), **§11 mới** | Package, SQL hay sai, HMAC / mã ngắn / chữ ký / `assurance`, bảng khoá Redis, 10 bước kiểm Key Attestation, việc nền, frontend + app + CSP, tải |
| `docs/PRODUCTION_READINESS.md` | §1.1 (3 dòng + 1 đoạn vị trí), §3 «Vòng đời dữ liệu» | 30 ngày xoá hẳn, `attendance.ForgetStudent` cho job xoá của PR |
| `docs/sprints/10.8/proposals.md` | P-108-4 … P-108-10 | Mâu thuẫn / lỗ hổng spec (mục 6) |

Chỉ chèn ở vùng mà `sprint/10.7-ui-r` không đụng. `git merge-tree` với `origin/sprint/10.7-ui-r` không ra xung đột ở ba tệp trên.

### 2. Phụ thuộc nhánh (PM cần biết trước khi giao dev)

- Nhánh đã chứa `main` (gồm 10.6, `00021_grading_guidance`) nhưng **chưa có** khung §23 (`Page kind`, `HeaderAction.docked`, `frame.spec.ts`): `sprint/10.7-ui-r` còn đi trước 101 commit và chưa vào `main`.
- Hệ quả:
  - L1, L4-backend, L5 (Go + plugin) làm được ngay.
  - Phần giao diện L2 / L3 / L4 phải đợi `sprint/10.7-ui-r` vào `main` rồi nhánh 10.8 gộp lại `main`. Nếu không, dev dựng trên primitive cũ rồi phải làm lại.
  - Số migration `00024` vẫn đúng: `main` giữ chỗ `00022` / `00023`.

### 3. Quyết định kỹ thuật chính (lý do ở ARCHITECTURE §11)

1. **Đóng cửa sổ do worker (`SweepJob`) không dùng `FOR UPDATE SKIP LOCKED` ở câu chọn**, khác SRS FR-7. Lý do: khoá cửa sổ trước buổi là ngược thứ tự với `PUT …/attendance`, nên dễ deadlock. Thay bằng: chọn id không khoá, rồi gọi `Close` (khoá buổi → cửa sổ, idempotent). Kết quả với người dùng y hệt FR-7; đây là chuyện «làm thế nào».
2. **Upsert máy bằng `ON CONFLICT … DO UPDATE` rỗng**, không dùng `DO NOTHING`. Cách này trả `id` ở cả hai nhánh và giữ khoá hàng, nên tuần tự hoá `DEVICE_SHARED` và chặn purge xoá máy đang dùng (§11.2).
3. **Frontend không import `@capacitor/*`.** Trang gọi `window.Capacitor.nativePromise(...)`, do `native-bridge.ts` của Capacitor core 8.6.2 gắn vào mọi trang WebView nạp (dòng 915–920, 1114; đọc trực tiếp từ `raw.githubusercontent.com/ionic-team/capacitor/main/core/native-bridge.ts` ngày 2026-11-15).
4. **`#token` sống sót qua đăng nhập.**
   - `AuthGate` (`frontend/src/shared/session/AuthGate.tsx`) không gắn trang con khi chưa đăng nhập, và chuyển `/login?next=` chỉ với path + query, nên mất fragment.
   - Vì vậy `/checkin` nằm ở route group `app/(checkin)/` với layout riêng: cất `{c, token, at}` vào `sessionStorage` và xoá hash **trước** `AuthGate`.
5. **Màn chiếu ngoài khung** `app/(present)/`: ECC `L`. Đo bằng `uqr` 0.1.3 (`node`, `/tmp/qrtest`):

   | URL | ECC `L` | ECC `M` |
   | --- | --- | --- |
   | 128 ký tự (`https://localhost`) | phiên bản 6 | phiên bản 8 |
   | 162 ký tự (host quick tunnel) | phiên bản 8 | phiên bản 9 |

   Poll theo `step_expires_at` cộng độ lệch đo từ `server_time`, không tin đồng hồ laptop.
6. **CSP mới** (repo chưa có CSP nào): `script-src 'self' 'unsafe-inline'`, không `unsafe-eval`, chỉ ở bản dựng production, cho mọi route. Lý do phủ mọi route: mọi trang trên tên miền đều gọi được plugin. Đủ cho US-AS-11 AC5. Nonce thì buộc mọi trang kết xuất theo request.
7. **Danh sách thu hồi chứng thực đệm ở Redis**, không đệm trong tiến trình (luật 10). Đo 2026-11-15 `curl https://android.googleapis.com/attestation/status`: 181.137 byte, 1.775 mục, `cache-control: public, max-age=86400`.
8. **`SESSION_UNMARKED` không cần đổi.** Mục «Hôm nay» này dựa trên `class_sessions.attendance_done_at` (`internal/store/queries/today.sql:119`), chỉ đặt khi `complete`. Đóng cửa sổ không làm mục biến mất; `CHECKIN_UNRESOLVED` buộc xử lý người chưa điểm danh trước khi `complete`. Bậc 13 cho `CHECKIN_REVIEW` chưa ai dùng (`internal/today/today.go`: 10, 12, 15… không có 13).

### 4. Lát và acceptance đo được

Thứ tự: L1 → (L4-backend ∥ L5-Go) → [chờ 10.7-ui-r vào `main`] → L2 ∥ L3 → L4-UI → L5-app → L7. Mọi lát đều qua thêm: `go vet`, `golangci-lint`, `go test -race` các package đụng tới, `bash scripts/ui-antipatterns.sh` (lát có giao diện), không sửa test cũ.

| Lát | Phạm vi | Acceptance (máy đo được) |
| --- | --- | --- |
| **L1** Backend lõi (L) | `00024`, sqlc, `internal/attendance` (trừ vị trí / mạng và đăng ký thiết bị), chặn `CHECKIN_UNRESOLVED` ở `crm`, nguồn «Hôm nay», `checked_in_at`, `SweepJob`, `PurgeJob`, `ForgetStudent`, `openapi.yaml` | 1. `goose up` chạy được trên DB sạch **và** DB đã seed; `sqlc diff` sạch. 2. Các test L1 ở SRS §9 xanh (mọi `TestToken*`, `TestCheckin*`, `TestCode*`, `TestWindow*`, `TestRound*`, `TestClose*`, `TestSweepAutoClose`, `TestAbsentUnchecked`, `TestComplete*`, `TestFlag*` trừ Geo / Campus, `TestAssuranceLevels`, `TestSpotCheck`, `TestClearFlagsAudit`, `TestCheckinsListCursor`, `TestPurge*`, `TestLogsNoCheckinSecrets`, `TestTodayCheckinReview`, `TestMeAttendanceCheckedAt`, `TestAgentToolNoCheckinFlags`, `TestOpenapiCheckinRoutes`). 3. `TestCheckinQueryCount`: một lượt mới ≤ 6 câu SQL (+1 `UPDATE` khi có `DEVICE_SHARED`), đếm bằng tracer của pgx; lượt lặp ≤ 2. 4. `TestDeviceSharedRace`: 50 lần lặp, 2 goroutine, lần nào cả hai lượt cũng có `DEVICE_SHARED`. 5. `TestSweepVsCloseNoDeadlock`: `Close` thủ công song song `SweepJob` cùng cửa sổ 100 lần, không lỗi `40P01`. 6. `BenchmarkCheckinVerify` (2 HMAC + `ecdsa.VerifyASN1`, không DB) ≤ 150 µs/op. TL đo bản tối thiểu ngày 2026-11-15: 44–55 µs/op (Go 1.27.1, MacBook Apple Silicon, `-benchtime 2000x -count 3`). 7. `git diff main -- backend-go/internal/crm/*_test.go` chỉ có dòng thêm |
| **L2** Màn giảng viên + màn chiếu (M) | §14.8 phần self check-in, §14.8.1, `uqr` | 1. `attendance-display.spec.ts`: QR và mã ngắn đổi trong ≤ 1 s sau `step_expires_at`; trong 60 s có 6 ± 1 lần `GET …/display`; mất mạng → «Mất kết nối» trong ≤ 2 s, ô QR trống sau khi mã hết hạn (nếu P-108-7 được nhận); STUDENT → 0 lần gọi `/api/v1`. 2. `frame.spec.ts` cho `/attendance` 0 độ lệch ở 1440 / 375 × 4 trạng thái. 3. axe 0 lỗi nghiêm trọng ở `/attendance/display` 1280×720 và 1920×1080. 4. `next build`: First Load JS của mọi route khác `/attendance/display` không tăng quá 1 KB (`uqr` chỉ trong route chiếu). 5. Ảnh 1280×720: cạnh QR ≥ 600 px, mã ngắn ≥ 96 px |
| **L3** Màn sinh viên web + PWA (S) | §14.8.2 (đường web), khoá WebCrypto, `app/(checkin)/`, `/me` `checked_in_at`, `app/manifest.ts`, CSP + `Permissions-Policy` | 1. `attendance-checkin.spec.ts`, `checkin-code.spec.ts`, `checkin-token-leak.spec.ts` xanh ở 375 px. 2. Đã đăng nhập: từ `goto` tới «Đã điểm danh» 0 thao tác người dùng, p95 ≤ 1,5 s trên 20 lần (stack local). 3. Chưa đăng nhập → đăng nhập ≤ 60 s → tự gửi; `location.hash` rỗng ≤ 100 ms sau khi tải; chuỗi token không có trong URL / header của request nào. 4. `page.evaluate`: khoá trong IndexedDB có `extractable === false`. 5. `frame.spec.ts` đo `/checkin` (loại `read`) 0 độ lệch. 6. Lighthouse mobile `/checkin` đạt cổng 10.7; axe 0 lỗi nghiêm trọng. 7. Toàn bộ Playwright chạy trên bản dựng production **có CSP** vẫn xanh (CSP không làm vỡ trang cũ) |
| **L4** Vị trí / mạng (S) | Trường `geofence` / `campus_cidrs` ở settings, cờ GEO / OFF_CAMPUS, `geo_check` cho STUDENT (P-108-3), dòng thông báo | 1. `TestFlagGeo`, `TestFlagCampus`, `TestNoCoordsOrIPStored`, `TestSettingsAttendance*` xanh. 2. Sau E2E: `pg_dump --data-only` toàn DB + log gateway + `audit_log`, `grep` vĩ độ / kinh độ / IP đã gửi → 0 kết quả. 3. Playwright: lớp tắt vị trí → `navigator.geolocation.getCurrentPosition` được gọi 0 lần (spy); lớp bật → dòng «Lớp này kiểm tra vị trí khi điểm danh.» hiện **trước** lần gọi đầu |
| **L5** App Android (M, rủi ro → L) | `mobile/`, plugin `EduPilotAttest`, `POST /me/devices{,/challenge}`, `internal/attendance/attest`, đệm thu hồi | 1. `go test ./internal/attendance/attest`: `caiman` → `boot_verified=true`, StrongBox; `akita` → `false`, TEE; tự ký → `ATTESTATION_FAILED`; khoá không khớp lá → `ATTESTATION_FAILED` (P-108-4); serial bị thu hồi → `ATTESTATION_FAILED`; tải danh sách lỗi + không đệm → `503`; `TestDeviceChallengeOnce` (`GETDEL` hai lần → lần 2 `422`). 2. `./gradlew assembleRelease` xong trên Mac; `apksigner verify --print-certs` cho SHA-256 = `ATTESTATION_CERT_SHA256`. 3. `aapt2 dump xmltree` manifest: có `CAMERA`, không `autoVerify`, không `usesCleartextTraffic="true"`. 4. `frontend/package.json` không có `@capacitor/`. 5. `docs/sprints/10.8/qc/android-checklist.md` trên emulator: đăng nhập, nhập mã ngắn → lượt `channel=CODE`; với P-108-5 → `assurance=2`. Máy thật (nếu có) → `assurance=3` |
| **L7** Kiểm chứng (M) | seed, E2E F7b trọn, k6 | 1. `k6 run benchmarks/load/checkin.js`: 200 SV / 30 s rồi 200 lượt / 5 s → `POST …/checkins` p95 ≤ 500 ms, lỗi 0 %, đúng 200 hàng `attendance_checkins`; 20 màn chiếu poll song song không làm p95 của route khác vượt SLO `SYSTEM_DESIGN.md` §5; ghi «MacBook Apple Silicon, Colima 6 CPU / 10 GB». 2. Seed: đếm theo US-AS-14 AC1 bằng SQL; `git diff --exit-code seed/expected_crm.csv`; chạy seed hai lần → số hàng không đổi. 3. Mỗi US-AS-01…14 có ≥ 1 ca lỗi + 1 ca phân quyền (bảng truy vết SRS §11 đủ cột Test). 4. Cổng chung AGENTS.md xanh |

### 5. Trả lời trước các câu dev sẽ hỏi

- **Bảng chữ base32 của `mac`:** RFC 4648 chuẩn, chữ hoa, không đệm, cắt 16 ký tự. Mã ngắn dùng Crockford. Hai bảng khác nhau là có chủ ý: `mac` không bao giờ được gõ tay.
- **`assurance` hay cờ quyết định?** `assurance` tính từ chữ ký, rồi mới hạ theo vị trí (§11.3). Đọc SRS 4.5 theo US-AS-12 AC5 (P-108-5 sửa chữ).
- **Nhập mã ngắn có cần `window_id` không?** Không. Dò các cửa sổ của lớp có `closes_at > now() − 15 phút`, dùng khoá duy nhất `(course_id, session_id, round)` có sẵn. Mỗi lớp ≤ 45 cửa sổ / học kỳ, không cần thêm chỉ mục.
- **Có phải đổi `attendance_records`?** Không (SRS §5).

### 6. Mâu thuẫn / lỗ hổng spec → `proposals.md`

| # | Mức | Một dòng |
| --- | --- | --- |
| P-108-4 | **chặn** (bảo mật) | Đăng ký thiết bị không ràng `public_key` với khoá trong chứng chỉ lá → giả được mức 3 |
| (đã xong) | – | STUDENT cần biết lớp có kiểm vị trí: BA đã thêm `geo_check` ở P-108-3, DESIGN §14.8.2 dùng trường này |
| P-108-5 | nên sửa | App chưa chứng thực (emulator): SRS §9 nói mức 2, 3.5 / AC6 nói mức 1 → dùng khoá Keystore như `WEB_KEY` = mức 2 |
| P-108-6 | nên sửa | `NOT_APP` gắn nhầm cho lượt từ app thật (mở bootloader / vị trí giả) |
| P-108-7 | nên sửa | Màn chiếu mất mạng giữ QR đã hết hạn → cả lớp nhận `TOKEN_EXPIRED` |
| P-108-8 | nên sửa | Ngoại lệ §23 cho màn chiếu + 3 token chữ chiếu + `frame.spec` đo `/checkin` |
| P-108-9 | → PM | `server.url` đóng vào APK; quick tunnel đổi host → dựng lại APK mỗi lần (mặc định) hay mua tên miền |
| P-108-10 | nên sửa | Chưa có quy trình xoá tài khoản (thuộc PR) → 10.8 giao `ForgetStudent` + test gọi thẳng |

### 7. Rủi ro

- **Đăng nhập ở trình duyệt mặc định.** Camera hệ thống mở trình duyệt mặc định, có thể khác trình duyệt sinh viên đã đăng nhập. Đăng nhập lại mất hơn 20 s, nên mã hết hạn. Giảm bằng cách nhắc sinh viên đăng nhập trước trên điện thoại (hướng dẫn demo). Mã ngắn vẫn dùng được sau khi đăng nhập.
- **Đọc mã từ xa.** `[SUY LUẬN]` QR đọc được tới khoảng 15–20 m trên màn 3 m. Xa hơn thì sinh viên nhập mã ngắn, đọc được tới khoảng 17–34 m (DESIGN §14.8.1).
- **CSP mới.** Áp cho cả ứng dụng nên có thể làm vỡ trang cũ. Acceptance L3 #7 bắt điều này. Cách lùi: chỉ áp CSP cho `/checkin`. Cách này yếu hơn, vì plugin gọi được từ mọi trang.
- **`BarcodeDetector` trong Android WebView.** Có hay không tuỳ máy / bản WebView. Đã có đường nhập mã ngắn.
- **Gradle / Kotlin ở L5.** Lát duy nhất có thể trượt sang L. Cắt được mà không ảnh hưởng L1–L4, L7 (web mức 1–2 vẫn chạy).
