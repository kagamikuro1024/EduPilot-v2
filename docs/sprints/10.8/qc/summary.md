# QC sprint 10.8 — FEAT-attendance-selfcheck (điểm danh tự phục vụ): tổng kết

Bản kiểm: ep-qc @`a1c7ab30` (đợt 1, L1–L5) → @`b54d020e` (đợt 2: HEAD `sprint/10.8-attendance` gồm L7, sửa D-AS-01…03). Migration `00024` chạy trên DB đã có dữ liệu (goose v24, giữ DB). Không LLM, không tải nặng (k6 chờ bench). Chi tiết lỗi: `defects.md`; TC: `tc-*.md`; kết quả từng TC: `scripts/results-as.jsonl`, `results-ui*.jsonl`; ảnh: `shots/`.

## Kết luận theo story (chỉ PASS / FAIL; phần không kiểm được ghi riêng)
| Story | Kết luận | Căn cứ chính |
| --- | --- | --- |
| US-AS-01 chính sách lớp | **PASS** | 13 ca sai 422, biên, `version` 409, PUT song song, phân quyền, SV chỉ thấy 3 trường, audit không toạ độ, UI cài đặt 1440/375 |
| US-AS-02 cửa sổ | **PASS** | mở / idempotent / `WINDOW_ALREADY_OPEN` / đua 6 vòng / 3 vòng + `ROUND_LIMIT` / display 429 / tự đóng 86 s / `COURSE_ARCHIVED` |
| US-AS-03 điểm danh (token + mã ngắn) | **PASS** | MAC / hết hạn / thứ tự kiểm / 5 lần sai → 429 / 10 lượt/phút / idempotent song song / vòng 2 / phản hồi giống nhau; `CHECKIN_TOKEN_*` không refresh (UI + app) |
| US-AS-04 cờ + mức tin cậy | **PASS** (sau sửa D-AS-01) | `WEB_KEY` r‖s + DER, `NO_DEVICE_KEY`, `DEVICE_UNTRUSTED`, `DEVICE_NEW`, `DEVICE_SHARED` (đua 4/4), `NOT_APP`, nhãn tiếng Việt |
| US-AS-05 đóng + đánh vắng | **PASS** | `PRESENT`/`LATE`, dòng tay giữ, đóng lần 2 không ghi gì, `absent-unchecked`, `CHECKIN_UNRESOLVED` |
| US-AS-06 danh sách lượt + Hôm nay | **PASS** | cursor, bộ lọc, `clear-flags` idempotent + audit, `CHECKIN_REVIEW` (≤ 7 ngày, biến mất khi hết cờ, SV không thấy) |
| US-AS-07 vòng 2, số đầu người, gọi tên | **PASS** | `RECHECK_MISS` (không nhân đôi, không đổi `attendance_records`), `over_headcount`, `spot-check` |
| US-AS-08 `/attendance` + màn chiếu | **PASS** (phần đã chạy) | màn chiếu 1920×1080 / 1280×720 (14 / 8 / 4 vmin, QR 1016 px giải mã được bằng jsQR), đếm ≤ 14 s, mất kết nối P-108-7, đóng, axe 0. **Chưa chạy trên stack thật:** thao tác tương tác ở `/attendance` (menu `⋯`, đóng bằng lưới + hộp xác nhận, ô số đầu người, Drawer gọi tên) — chỉ chụp 1440/375 + axe; dev specs CI xanh |
| US-AS-09 `/checkin` web | **PASS** | 375 px không tràn, token không vào request / `Referer` / storage, đăng nhập rồi quay lại, hết hạn, nhập mã + khoá 429, CSP, axe 0. KHÔNG KIỂM ĐƯỢC: iPhone Safari / PWA thật (TC-AS09-34) |
| US-AS-10 vị trí + mạng trường | **PASS** | haversine sai số 0 m (8 điểm), biên 249,5 m, `GEO_WEAK` 101, `GEO_DENIED` ×2, CIDR, tắt giữa buổi, không cột toạ độ / IP, log sạch; emulator: trong vùng / ngoài vùng 5 302 m / từ chối quyền |
| US-AS-11 app Android | **PASS** (phần kiểm được) | cấu hình tĩnh (cleartext off, `allowNavigation`, không `autoVerify`, không khoá trong git); emulator API 35: WebView có `BarcodeDetector`/`qr_code` → nút «Quét mã điểm danh», từ chối camera → «Chưa cấp quyền camera.», `ErrorPath` «Thử lại», điểm danh mức 2 (P-108-5), `NOT_APP` (P-108-6). KHÔNG KIỂM ĐƯỢC: máy thật (mức 3 thật, `boot_verified`, `GEO_MOCK` thật), quét QR thật bằng camera ảo. Lỗi D-AS-04 (Thấp) còn mở |
| US-AS-12 chứng thực thiết bị | **PASS** | nonce 1 lần / 5 lần-phút / hết hạn, chuỗi rác → `ATTESTATION_FAILED`, 1 và 7 chứng chỉ → 422, mức 3 / `boot_verified=false` / `GEO_MOCK` / `key_id` lạ bằng fixture, Go test attest 4/4 (3 chuỗi thật, P-108-4 trong `TestAttestRejects`). KHÔNG KIỂM ĐƯỢC: chuỗi chứng thực từ máy thật, danh sách thu hồi khi mất mạng Google |
| US-AS-13 giữ 30 ngày | **PASS** | purge: A (31 ngày) xoá, B (29 ngày) + cửa sổ mở giữ, thiết bị mồ côi xoá, chạy lại + 2 tiến trình song song không đổi, cửa sổ còn. KHÔNG KIỂM ĐƯỢC: lô 2 500 hàng, ngắt Postgres giữa lô (TC-AS13-04, -09) |
| US-AS-14 seed / E2E / tải | **PASS** (phần kiểm được) | `seed-checkin-demo`: 43 lượt → 0 lượt (idempotent); 761987 giữ nguyên số `attendance_records`; đủ ca cờ (vòng 1 ≥ 20 lượt, `LATE` = 1, `RECHECK_MISS`, `over_headcount`); `sv.gioi` không lượt; `check-checkin-seed.mjs` phần snapshot xanh; `ui-antipatterns` sạch; CI dev xanh (Go, e2e, Lighthouse). **k6: chưa đo trên Docker (chờ bench); số của dev2 (native, không PgBouncer) không thay được.** `check-checkin-seed` chạy lại seed đầy đủ thoát 1 ở bước 17/20 (409 `TICKET_STATE_INVALID` do dữ liệu chat/ticket tích luỹ của ep-qc, ngoài 10.8) |

## Lỗi (xem `defects.md`)
| Mã | Mức | Trạng thái |
| --- | --- | --- |
| D-AS-01 máy dùng chung → 503, không ghi lượt (PgBouncer `[]uuid.UUID`) | Cao | **Đóng** (dev2 sửa; kiểm lại 4/4 đua + app emulator) |
| D-AS-02 trường lạ trong thân → 422 (spec nói «bỏ qua») | Thấp | **Đóng** (BA vá spec P-108-13) |
| D-AS-03 openapi `attestation_chain` 1–10 vs thực tế 2–6 | Thấp | **Đóng** |
| D-AS-04 `mobile/README.md` + `prepare.mjs` cho `http://localhost|127.0.0.1|10.0.2.2` nhưng `cleartext:false` → APK trắng «Không kết nối được máy chủ» | Thấp | Mở |

Ghi nhận (không lỗi): (1) `/me/today` cache ~15–29 s sau thay đổi trực tiếp DB (chỉ lộ khi dùng fixture SQL); (2) hộp xin quyền vị trí lần đầu có thể làm token hết hạn (≤ 2 bước) → SV quét lại (đo: emulator `rotate=5 s`); (3) `qr_url` dùng `APP_PUBLIC_URL` (ep-qc mặc định `https://localhost`) — phải đặt đúng host/tunnel khi chạy máy thật; (4) `Referrer-Policy` của `/checkin` không đặt (SRS không yêu cầu; `#fragment` không vào `Referer`, đã đo); (5) emulator không gắn `GEO_MOCK` cho vị trí đặt bằng `adb emu geo fix`; (6) `TestAttestKeyMismatch` (tên trong P-108-4) không có — ca P-108-4 nằm trong `TestAttestRejects`; (7) mọi lượt trong app mang `channel=WEB` (app không gửi `app.version`).

## Chưa chạy / KHÔNG KIỂM ĐƯỢC (không tính FAIL)
- Máy Android thật (P-108-12): AC-5, 5b, 6, 9b, 11 (thật), 13, 17, 19 của `android-checklist.md`; quét QR thật qua camera trong app (AC-3); AC-7b.
- iPhone / Safari / PWA; k6 `checkin.js` trên Docker (chờ bench 1.000 bài); Lighthouse `/checkin` (CI); lô purge 2 500 hàng; chaos khi purge.
- Chưa chạy trên stack thật (đã có test dev xanh CI): tương tác `/attendance` (TC-AS08-02…24), TC-AS01-10 (đổi chính sách khi cửa sổ mở), TC-AS02-09 / -15 (outbox / `over_headcount` biên), TC-AS05-15 (đóng song song `PUT` lưới), TC-AS06-01 (≥ 120 lượt) / -03 (EXPLAIN), TC-AS03-10 / -13 / -22, TC-AS07-10 (UI gọi tên).
- `check-checkin-seed.mjs` đầy đủ (chạy lại toàn seed) — cần DB sạch.

## Cách tái chạy
`scripts/as_setup.py` (lớp QC-AS + sessions), `as_a.py`…`as_h.py` (API, cần `PYTHONPATH=…/10.5/qc/scripts`), `frontend/e2e/qc-as-real.spec.ts` + `playwright.qc-as.config.ts` (`QC_REAL=1 QC_CID QC_SIDS QC_PW`), `frontend/e2e/qc-as-android.spec.ts` (emulator + CDP; chưa ổn định: khoá Keystore dùng chung giữa các tài khoản nên cần `pm clear` mỗi tài khoản — kết quả Android lấy từ các lần chạy tay CDP cùng quy trình; cần `adb reverse tcp:580 tcp:580` sau `adb root`, APK debug dựng với `capacitor.config.js` `cleartext:true` chỉ để thử, KHÔNG commit).
