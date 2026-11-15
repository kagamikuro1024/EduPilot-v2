# QC test case — US-AS-11 (app Android), US-AS-14 (seed, E2E, tải, luật), truy vết P-108-1…11
Nguồn: US.md + SRS §3.3, 8, 9, 11 + `proposals.md`. Hộp đen; quy ước: `tc-00-chung.md`. Phần chạy tay trên điện thoại ở `android-checklist.md` (mã bước `AC-n`); ở đây là phần kiểm được bằng lệnh / Playwright / DIFF. `mobile/` = thư mục Capacitor trong repo.

## US-AS-11 App Android (phần kiểm tĩnh + giả lập WebView)
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS11-01 | AC1 | Repo | Đọc `mobile/capacitor.config.*`, `android/**/AndroidManifest.xml`, `network_security_config` | Capacitor 8 (MIT); `server.url` = `APP_SERVER_URL` (cùng gốc web), **không** đóng gói tĩnh; `allowNavigation` chỉ host của ứng dụng; `android.allowMixedContent=false`; không `usesCleartextTraffic`; quyền `CAMERA`; **không** intent filter `autoVerify`; **không** `assetlinks.json` (P-108-2) | DIFF |
| TC-AS11-02 | AC1 | – | `git ls-files mobile \| grep -E '\.(jks|keystore|p12)$|keystore.properties'`; `grep -rn "storePassword\|keyPassword" mobile` | 0 tệp / 0 chuỗi khoá ký; kho khoá ngoài repo; `.gitignore` chặn | DIFF |
| TC-AS11-03 | AC1 | – | `git diff <gốc 10.7>... -- frontend/package.json` | Không `@capacitor/*` trong `frontend/` (ARCHITECTURE §11.7); `@capacitor/*` chỉ ở `mobile/package.json`; thư viện mới chỉ `uqr` (frontend) và nhóm `@capacitor/*` (mobile) | DIFF |
| TC-AS11-04 | AC2 | – | Đọc `openapi.yaml` + route app gọi | Không route `app-only` cho điểm danh; app dùng đúng `POST …/checkins`, `POST /me/devices*` | DIFF |
| TC-AS11-05 | AC3, AC3a | Playwright, `BarcodeDetector` **giả** (`addInitScript`): QR đúng (`origin` app, `/checkin`, `c` UUID, có `#token`) | SV mở `/checkin` không token, bấm «Quét mã điểm danh» | Xin camera (`getUserMedia({video:{facingMode:"environment"}})` bị bắt), phát hiện QR → gửi **một** `POST`, `track.stop()` | E2E |
| TC-AS11-06 | AC3 | – | `BarcodeDetector` giả trả QR **lạ**: origin khác / path khác / `c` không phải UUID / không `#token` | Bỏ qua im lặng, tiếp tục quét; không `POST`; không thông báo | E2E |
| TC-AS11-07 | AC3a | `BarcodeDetector` / `qr_code` không hỗ trợ (`getSupportedFormats()` rỗng) | Mở `/checkin` | Không nút quét; ô nhập mã ngắn hiện ngay; lượt `channel=CODE` | E2E |
| TC-AS11-08 | AC3a | `getUserMedia` bị từ chối | Bấm quét | Ô nhập mã + «Chưa cấp quyền camera.» (cảnh báo); `Điểm danh` dock | E2E |
| TC-AS11-09 | AC3 | – | Rời trang / `visibilitychange→hidden` / `Thoát` khi đang quét | `MediaStreamTrack` dừng (`readyState="ended"`), đèn camera tắt | E2E |
| TC-AS11-10 | AC4 | Giả `window.Capacitor.isNativePlatform()=true` + plugin `EduPilotAttest` giả | Quét / điểm danh | Gọi plugin `sign` (DER) thay vì WebCrypto; **không** gọi `crypto.subtle` (spy); thân `device` đúng loại | E2E |
| TC-AS11-11 | AC4 | Đăng ký thiết bị trả `422 ATTESTATION_FAILED` / `503` (chặn mạng giả) | Điểm danh | App vẫn ký bằng khoá Keystore và gửi `WEB_KEY` (DER) → mức 2 (P-108-5); không báo lỗi cho SV | E2E |
| TC-AS11-12 | AC5 | – | Đọc `Content-Security-Policy` các trang | Không nguồn script ngoài tên miền ứng dụng; `/checkin` không `unsafe-eval` | E2E |
| TC-AS11-13 | AC6 | Plugin lỗi / không Keystore | Điểm danh | Không `device` (mức 1, `NO_DEVICE_KEY`), không treo; WebView lỗi tải → trang lỗi chuẩn + `Thử lại` | E2E |
| TC-AS11-14 | AC7 | TA / TEACHER đăng nhập (giả `isNativePlatform`) | Mở `/checkin` | Màn chặn chuẩn như web; không lưu mật khẩu / token ngoài cookie WebView | E2E |
| TC-AS11-15 | AC3b | – | `grep -n "Quét mã điểm danh" mobile/README.md` | Có dòng «Dùng nút Quét mã điểm danh trong app» | DIFF |
| TC-AS11-16 | AC8 | – | Có `docs/sprints/10.8/qc/android-checklist.md` (tệp này) và `mobile/README.md` lệnh một bước dựng lại APK theo `APP_SERVER_URL` | Có (P-108-9a) | DIFF |
| TC-AS11-17 | – | Máy thật / emulator | Chạy toàn bộ `android-checklist.md` | Điền kết quả từng bước; ghi KHÔNG KIỂM ĐƯỢC nếu không có máy | TAY |

## US-AS-14 Seed, E2E, tải, luật
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS14-01 | AC1 | DB trống, `pnpm dev` / seed | `gateway admin seed-checkin-demo --course 761988` (chỉ `APP_ENV∈{dev,test}`); chạy lại | In số dòng đã đổi; chạy lại **không nhân đôi**; `APP_ENV=production` bị từ chối | CLI+DB |
| TC-AS14-02 | AC1 | Sau seed | `psql` đếm | Lớp 761988: `self_checkin=true`; vòng 1 đã đóng ≥ 20 lượt đủ ca: 0 cờ, `DEVICE_SHARED` (một cặp), `DEVICE_NEW`, `NO_DEVICE_KEY`, `TOKEN_LATE`, 1 `LATE`; vòng 2 đã đóng có 1 `RECHECK_MISS`; `headcount` thấp hơn số lượt (`over_headcount=true`); ≥ 1 lượt `CODE`; `CHECKIN_REVIEW` hiện ở `Hôm nay` của TA / TEACHER | DB+API |
| TC-AS14-03 | AC1 | Sau seed | `check-crm-seed.mjs` (hoặc so `seed/expected_crm.csv`); đếm `attendance_records` lớp 761987 | **Không đổi** `expected_crm.csv`; 270 `attendance_records` lớp 761987 y nguyên; chạy hai lần như nhau | CLI+DB |
| TC-AS14-04 | AC2 | `attendance-checkin.spec.ts` (E2E dev) | Chạy `pnpm -C frontend exec playwright test e2e/attendance-checkin.spec.ts --project=desktop --project=mobile --workers=1` | Xanh; **không bị sửa để xanh** (so `git diff` của spec giữa handoff và HEAD); phủ: mở điểm danh → màn chiếu QR + đếm; SV quét; mã hết hạn; cửa sổ đóng; nhập mã ngắn (đúng / sai / hết hạn / đóng / > 5 sai); ô quét WebView giả; `DEVICE_SHARED` hai tài khoản cùng trình duyệt; SV không thấy cờ; đóng + đánh vắng + `CHECKIN_UNRESOLVED`; vòng 2 + `RECHECK_MISS` | E2E |
| TC-AS14-05 | AC2 | – | Chạy `attendance-display.spec.ts`, `checkin-code.spec.ts`, `checkin-token-leak.spec.ts` | Xanh; `attendance-display` có ca «mất mạng giữ QR tới hạn rồi ô trống + Mất kết nối» (P-108-7), vai sai | E2E |
| TC-AS14-06 | AC3 | `ep-t1`/stack tải do PM chỉ định (không chạy trên `ep-qc` khi dev đang bench) | `k6 run benchmarks/load/checkin.js` (200 SV / 30 s, rồi 200 lượt / 5 s) | `POST …/checkins` p95 ≤ 500 ms, lỗi 0 %, **đúng 200 hàng** (đếm DB); poll `display` 20 lớp × 1 req / 10 s trong SLO `SYSTEM_DESIGN.md` | K6 |
| TC-AS14-07 | AC3 | – | Trong lúc tải | `docker stats` gateway / Postgres không vượt trần; không `5xx`; Redis chỉ dùng cho hạn mức (đường nóng không gọi Redis ngoài hạn mức — kiểm bằng `MONITOR` mẫu 10 s: không khoá `ep:att:*` ngoài `codefail`) | K6+LOG |
| TC-AS14-08 | AC4 | – | Lighthouse + axe cho `/checkin` (đã quay lại) và `/attendance/display` | Đạt cổng 10.7: LCP theo cổng, axe 0 lỗi nghiêm trọng | UI |
| TC-AS14-09 | AC5 | – | `bash scripts/ui-antipatterns.sh` (+ `--selftest`), `go vet ./...`, `golangci-lint run`, `go test -race ./...`, `go test ./internal/contract/...`, `sqlc diff`, `goose up` trên DB sạch **và** DB có dữ liệu | Tất cả xanh / sạch; migration `00024` một tệp, một giao dịch, **chỉ `CREATE`**; không sửa migration đã merge (`git diff --name-status <gốc> -- backend-go/db/migrations` chỉ có `A`) | CLI |
| TC-AS14-10 | AC5 | – | `git diff <gốc>... -- '*_test.go' 'frontend/e2e/*.spec.ts'` (phần test **cũ**) | Không test cũ bị đổi để xanh (chỉ thêm / sửa theo hợp đồng mới đã có proposal); phát hiện → lỗi nghiêm trọng báo PM | DIFF |
| TC-AS14-11 | AC6 | – | Truy vết bảng SRS §11 | Mỗi US-AS-0x có ≥ 1 ca lỗi và ≥ 1 ca phân quyền trong bộ test dev (tên test khớp bảng §11); QC liệt kê thiếu sót | DIFF |
| TC-AS14-12 | – | Hồi quy | Smoke F7 cũ: điểm danh tay lớp 761987 (buổi không cửa sổ) — mở `/attendance`, lưu | y nguyên trước sprint (ảnh + API); `PUT …/attendance` cùng byte | UI+API |
| TC-AS14-13 | – | Hồi quy toàn bộ | Chạy lại `qc-ui-audit-real.spec.ts` (đợt R) + `ui-antipatterns` + `check-*-seed` | Không lệch mới so với đợt R chốt (5 lệch đã biết: E2 chat N/A, D3 sửa ở fix/dr-02) | UI |

## Truy vết P-108-1…11 (mỗi đề xuất PM chấp nhận → TC kiểm)
| P | Nội dung | TC kiểm |
| --- | --- | --- |
| P-108-1 | Phương án 5 lớp, chỉ gắn cờ không tự phạt; Q1–Q9 (không tự ghi vắng; hoàn tác qua lưới; xoá 30 ngày) | TC-AS03-17, TC-AS05-04, TC-AS05-07…09, TC-AS13-01, TC-AS01-01 |
| P-108-2 | Xoá hẳn 30 ngày; `CHECKIN_UNRESOLVED` khi `complete:true`; bỏ App Links; quét trong WebView + mã ngắn | TC-AS13-01, TC-AS05-10…12, TC-AS11-01/05…08, TC-AS09-20…24 |
| P-108-3 | 6 điểm hợp đồng (list cửa sổ ≤ 3; `close` idempotent; `WEB_KEY` kèm `createCheckin`) | TC-AS02-13/14, TC-AS05-06, TC-AS04-01, TC-AS01-12 |
| P-108-4 | `public_key` == SPKI chứng chỉ lá; `TestAttestKeyMismatch` | TC-AS12-06, TC-AS12-18 |
| P-108-5 | App chưa chứng thực ký Keystore gửi `WEB_KEY` DER → mức 2 | TC-AS12-11, TC-AS11-11, `android-checklist.md` AC-7 |
| P-108-6 | `NOT_APP` = `require=APP` và không có chữ ký `ANDROID_KEY` hợp lệ của khoá `attested=true` | TC-AS04-13 |
| P-108-7 | Màn chiếu giữ QR tới hạn theo giờ máy chủ rồi ô trống «Mất kết nối» | TC-AS08-36, TC-AS08-37, TC-AS14-05 |
| P-108-8 | Route trình chiếu ngoài khung; ba token `--ep-text-projector-*` chỉ trong `features/attendance/display/`; `frame.spec.ts` thêm `/checkin` | TC-AS08-30/31/45/46, TC-AS09-35/38 |
| P-108-9 | Lệnh một bước dựng lại APK theo `APP_SERVER_URL`; giữ tunnel | TC-AS11-16, `android-checklist.md` AC-0 |
| P-108-10 | `attendance.ForgetStudent`; job xoá thuộc phase PR phải gọi | TC-AS13-05, TC-AS13-06 |
| P-108-11 | Mã `CHECKIN_TOKEN_INVALID/EXPIRED` (không đăng xuất / refresh); người ngoài lớp `403 reason=course` | TC-AS03-04/05/24/25, TC-AS09-12, mọi TC phân quyền (`sv.moi`) |
