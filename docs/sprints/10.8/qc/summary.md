# QC sprint 10.8 — FEAT-attendance-selfcheck (điểm danh tự phục vụ): tổng kết

Bản kiểm: ep-qc @`a1c7ab30` (đợt 1, L1–L5) → @`b54d020e` (đợt 2: L7, sửa D-AS-01…03) → đợt 3 (@`b54d020e`): tương tác `/attendance` + TC biên → đợt 4 (@`f1b8602d`, ep-qc dựng lại frontend + gateway): kiểm lại D-AS-05/06 + một nút chính, chạy lại `qc-as-ui.spec` 1440/375 (36/36 PASS), hồi quy đường điểm danh. D-AS-04 dev sửa @`089b9d01` (QC kiểm tĩnh). Migration `00024` chạy trên DB đã có dữ liệu (goose v24, giữ DB). Không LLM, không tải nặng (k6 chờ bench). Chi tiết lỗi: `defects.md`; TC: `tc-*.md`; kết quả từng TC: `scripts/results-as.jsonl`, `results-ui*.jsonl`; ảnh: `shots/`.

## Kết luận theo story (chỉ PASS / FAIL; phần không kiểm được ghi riêng)
| Story | Kết luận | Căn cứ chính |
| --- | --- | --- |
| US-AS-01 chính sách lớp | **PASS** (sau sửa D-AS-06) | tắt `self_checkin` giữa buổi: cửa sổ đang mở vẫn nhận token + mã ngắn (201), rotate vẫn ghim, cửa sổ mới / sau khi đóng → 409. 13 ca sai 422, biên, `version` 409, PUT song song, phân quyền, SV chỉ thấy 3 trường, audit không toạ độ, UI cài đặt 1440/375 |
| US-AS-02 cửa sổ | **PASS** | mở / idempotent / `WINDOW_ALREADY_OPEN` / đua 6 vòng / 3 vòng + `ROUND_LIMIT` / display 429 / tự đóng 86 s / `COURSE_ARCHIVED` |
| US-AS-03 điểm danh (token + mã ngắn) | **PASS** | MAC / hết hạn / thứ tự kiểm / 5 lần sai → 429 / 10 lượt/phút / idempotent song song / vòng 2 / phản hồi giống nhau; `CHECKIN_TOKEN_*` không refresh (UI + app) |
| US-AS-04 cờ + mức tin cậy | **PASS** (sau sửa D-AS-01) | `WEB_KEY` r‖s + DER, `NO_DEVICE_KEY`, `DEVICE_UNTRUSTED`, `DEVICE_NEW`, `DEVICE_SHARED` (đua 4/4), `NOT_APP`, nhãn tiếng Việt |
| US-AS-05 đóng + đánh vắng | **PASS** | `PRESENT`/`LATE`, dòng tay giữ, đóng lần 2 không ghi gì, `absent-unchecked`, `CHECKIN_UNRESOLVED` |
| US-AS-06 danh sách lượt + Hôm nay | **PASS** | cursor, bộ lọc, `clear-flags` idempotent + audit, `CHECKIN_REVIEW` (≤ 7 ngày, biến mất khi hết cờ, SV không thấy) |
| US-AS-07 vòng 2, số đầu người, gọi tên | **PASS** | `RECHECK_MISS` (không nhân đôi, không đổi `attendance_records`), `over_headcount`, `spot-check` |
| US-AS-08 `/attendance` + màn chiếu | **PASS** (D-AS-05 đã sửa; sau đóng chỉ một nút chính, «Đánh vắng N SV» một chỗ) | tương tác Staff chạy thật 1440 + 375 (mở điểm danh → màn chiếu, đếm k/n, số đầu người tự lưu, `over_headcount`, Drawer gọi ngẫu nhiên, đóng, `CHECKIN_UNRESOLVED`, hộp xác nhận «Đánh vắng N sinh viên?», mở vòng 2, axe 0, không tràn ngang: 34/34); màn chiếu 1920×1080 / 1280×720 (14 / 8 / 4 vmin, QR 1016 px giải mã được bằng jsQR), đếm ≤ 14 s, mất kết nối P-108-7, đóng, axe 0 |
| US-AS-09 `/checkin` web | **PASS** | 375 px không tràn, token không vào request / `Referer` / storage, đăng nhập rồi quay lại, hết hạn, nhập mã + khoá 429, CSP, axe 0. KHÔNG KIỂM ĐƯỢC: iPhone Safari / PWA thật (TC-AS09-34) |
| US-AS-10 vị trí + mạng trường | **PASS** | haversine sai số 0 m (8 điểm), biên 249,5 m, `GEO_WEAK` 101, `GEO_DENIED` ×2, CIDR, tắt giữa buổi, không cột toạ độ / IP, log sạch; emulator: trong vùng / ngoài vùng 5 302 m / từ chối quyền |
| US-AS-11 app Android | **PASS** (phần kiểm được) | cấu hình tĩnh (cleartext off, `allowNavigation`, không `autoVerify`, không khoá trong git); emulator API 35: WebView có `BarcodeDetector`/`qr_code` → nút «Quét mã điểm danh», từ chối camera → «Chưa cấp quyền camera.», `ErrorPath` «Thử lại», điểm danh mức 2 (P-108-5), `NOT_APP` (P-108-6). KHÔNG KIỂM ĐƯỢC: máy thật (mức 3 thật, `boot_verified`, `GEO_MOCK` thật), quét QR thật bằng camera ảo. D-AS-04 đã được dev sửa (@`089b9d01`) |
| US-AS-12 chứng thực thiết bị | **PASS** | nonce 1 lần / 5 lần-phút / hết hạn, chuỗi rác → `ATTESTATION_FAILED`, 1 và 7 chứng chỉ → 422, mức 3 / `boot_verified=false` / `GEO_MOCK` / `key_id` lạ bằng fixture, Go test attest 4/4 (3 chuỗi thật, P-108-4 trong `TestAttestRejects`). KHÔNG KIỂM ĐƯỢC: chuỗi chứng thực từ máy thật, danh sách thu hồi khi mất mạng Google |
| US-AS-13 giữ 30 ngày | **PASS** | purge: A (31 ngày) xoá, B (29 ngày) + cửa sổ mở giữ, thiết bị mồ côi xoá, chạy lại + 2 tiến trình song song không đổi, cửa sổ còn. KHÔNG KIỂM ĐƯỢC: lô 2 500 hàng, ngắt Postgres giữa lô (TC-AS13-04, -09) |
| US-AS-14 seed / E2E / tải | **PASS** (phần kiểm được) | `seed-checkin-demo`: 43 lượt → 0 lượt (idempotent); 761987 giữ nguyên số `attendance_records`; đủ ca cờ (vòng 1 ≥ 20 lượt, `LATE` = 1, `RECHECK_MISS`, `over_headcount`); `sv.gioi` không lượt; `check-checkin-seed.mjs` phần snapshot xanh; `ui-antipatterns` sạch; CI dev xanh (Go, e2e, Lighthouse). **k6 trên ep-qc Docker + PgBouncer: PASS (xem mục «k6» bên dưới).** `check-checkin-seed` chạy lại seed đầy đủ thoát 1 ở bước 17/20 (409 `TICKET_STATE_INVALID` do dữ liệu chat/ticket tích luỹ của ep-qc, ngoài 10.8) |

## Lỗi (xem `defects.md`)
| Mã | Mức | Trạng thái |
| --- | --- | --- |
| D-AS-01 máy dùng chung → 503, không ghi lượt (PgBouncer `[]uuid.UUID`) | Cao | **Đóng** (dev2 sửa; kiểm lại 4/4 đua + app emulator) |
| D-AS-02 trường lạ trong thân → 422 (spec nói «bỏ qua») | Thấp | **Đóng** (BA vá spec P-108-13) |
| D-AS-03 openapi `attestation_chain` 1–10 vs thực tế 2–6 | Thấp | **Đóng** |
| D-AS-04 `prepare.mjs`/README cho `http://localhost…` nhưng `cleartext:false` → APK trắng | Thấp | **Đóng** (dev @`089b9d01`; QC kiểm lại: `prepare.mjs` từ chối http, https tunnel OK) |
| **D-AS-06** tắt `self_checkin` giữa buổi → SV bị `409 SELF_CHECKIN_DISABLED` dù cửa sổ đang mở (US-AS-01 AC5) | **Trung bình** | **Đóng** (dev @`0cec750a`; kiểm lại: token + mã ngắn 201, mở mới 409, sau đóng 409) |
| D-AS-05 (`/attendance` 375 px) «Còn N sinh viên chưa điểm danh» xuống dòng từng chữ | Thấp | **Đóng** (dev @`1ce5619b`; đo: 261 × 22 px ở 375, 839 × 22 px ở 1440) |

Ghi nhận (không lỗi; mục trùng nút ở `/attendance` đã được dev sửa @`f1b8602d`, QC kiểm lại ĐẠT): (1) `/me/today` cache ~15–29 s sau thay đổi trực tiếp DB (chỉ lộ khi dùng fixture SQL); (2) hộp xin quyền vị trí lần đầu có thể làm token hết hạn (≤ 2 bước) → SV quét lại (đo: emulator `rotate=5 s`); (3) `qr_url` dùng `APP_PUBLIC_URL` (ep-qc mặc định `https://localhost`) — phải đặt đúng host/tunnel khi chạy máy thật; (4) `Referrer-Policy` của `/checkin` không đặt (SRS không yêu cầu; `#fragment` không vào `Referer`, đã đo); (5) emulator không gắn `GEO_MOCK` cho vị trí đặt bằng `adb emu geo fix`; (6) `TestAttestKeyMismatch` (tên trong P-108-4) không có — ca P-108-4 nằm trong `TestAttestRejects`; (7) mọi lượt trong app mang `channel=WEB` (app không gửi `app.version`).

## Chưa chạy / KHÔNG KIỂM ĐƯỢC (không tính FAIL)
- Máy Android thật (P-108-12): AC-5, 5b, 6, 9b, 11 (thật), 13, 17, 19 của `android-checklist.md`; quét QR thật qua camera trong app (AC-3); AC-7b.
- iPhone / Safari / PWA; Lighthouse `/checkin` (CI); lô purge 2 500 hàng; chaos khi purge.
- Đã chạy ở đợt 3 (trước đây «chưa chạy»): TC-AS01-10 (a: FAIL → D-AS-06; b, c PASS), TC-AS02-09, TC-AS02-15, TC-AS03-10, TC-AS03-13, TC-AS03-22 (8 vòng, không 5xx), TC-AS05-15 (8 vòng: đóng thắng, `PUT` lưới 409 `VERSION_CONFLICT`, không dòng trùng), TC-AS06-01 (90 lượt: 30 SV × 3 vòng, không phải 120), TC-AS06-03 (0,49 ms; chỉ mục theo quy mô T1 KHÔNG KIỂM ĐƯỢC), TC-AS07-10 (UI).
- Vẫn KHÔNG chạy: TC-AS06-01 với ≥ 120 lượt thật (chỉ 90), kiểm chỉ mục `EXPLAIN` ở quy mô T1 (bảng điểm danh chỉ ~400 hàng ngoài 400 hàng k6).
- `check-checkin-seed.mjs` đầy đủ (chạy lại toàn seed) — cần DB sạch.

## Cách tái chạy
`scripts/as_setup.py` (lớp QC-AS + sessions), `as_a.py`…`as_h.py` (API, cần `PYTHONPATH=…/10.5/qc/scripts`), `frontend/e2e/qc-as-real.spec.ts` + `playwright.qc-as.config.ts` (`QC_REAL=1 QC_CID QC_SIDS QC_PW`), `frontend/e2e/qc-as-android.spec.ts` (emulator + CDP; chưa ổn định: khoá Keystore dùng chung giữa các tài khoản nên cần `pm clear` mỗi tài khoản — kết quả Android lấy từ các lần chạy tay CDP cùng quy trình; cần `adb reverse tcp:580 tcp:580` sau `adb root`, APK debug dựng với `capacitor.config.js` `cleartext:true` chỉ để thử, KHÔNG commit).

## k6 `benchmarks/load/checkin.js` (US-AS-14 AC3) — ep-qc, Docker, PgBouncer
Cách đo: ep-qc dựng lại sang DB `edupilot_t1` (POSTGRES_DB / DATABASE_URL / PGBOUNCER_URL, gateway kèm `benchmarks/load/k6-stack.yml`, **qua PgBouncer**), `seed-t1.mjs --no-vectors` (1.000 SV · 20 lớp, script lấy từ `sprint/9.5-nfr`), k6 chạy trên máy host (Mac) → Caddy `https://localhost:943` → gateway (1) + worker (1) Docker; MacBook Apple Silicon, không bench khác chạy lúc đó. Xong đã trả ep-qc về DB QC (`edupilot`). Lượt gửi **không có `device`** (k6 không ký ECDSA): đường nóng không có bước chữ ký (cờ `NO_DEVICE_KEY`, vẫn 201); đường `DEVICE_SHARED` (D-AS-01) đã kiểm riêng ở API.

| Kịch bản | Tải | `POST …/checkins` p95 / p99 / max | `display` p95 | đọc nền p95 | lỗi | hàng DB | Ngưỡng |
| --- | --- | --- | --- | --- | --- | --- | --- |
| spread | 200 SV dàn đều 30 s (+ 20 màn chiếu poll, đọc nền 5/s) | **9,62** / 11,46 / 22,38 ms | 11,26 ms | 10,19 ms | 0 % (0/776) | 200 (buổi 1) | ≤ 500 ms, lỗi 0 %, 200 hàng, display < 300 ms: **PASS** |
| burst | 200 lượt dồn 5 s | **5,38** / 6,41 / 12,59 ms | 9,66 ms | 12,59 ms | 0 % (0/601) | 200 (buổi 2) | **PASS** |

Số của dev2 (native, không PgBouncer): 15,6 / 9,8 ms — cùng bậc. Nhật ký: `scripts/k6-spread.log|json`, `k6-burst.log|json`. Ghi chú: k6 chạy trên host nên có phí mạng loopback; đường Caddy TLS có tính vào số đo.

## Chốt (đợt 4)
Lỗi 10.8: D-AS-01…06 **tất cả đóng**, không lỗi mở. 14/14 story PASS ở phần kiểm được. Chưa kiểm được (không tính FAIL): máy Android thật, iPhone/PWA, chỉ mục `EXPLAIN` ở quy mô T1, 120 lượt thật, lô purge 2.500 hàng, kiểm tra Lighthouse `/checkin` (CI). Điều kiện merge theo QC: không còn.
