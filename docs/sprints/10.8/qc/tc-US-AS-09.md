# QC test case — US-AS-09 (`/checkin` web, một thao tác, 375 px) — lát L3
Nguồn: US.md + SRS §3.1, 3.5, 7 + `DESIGN.md` §14.8.2; P-108-8, P-108-11. Hộp đen; quy ước: `tc-00-chung.md`. Playwright thật trên `ep-qc` (QC_REAL). **Mọi TC UI chạy 375×812 (`isMobile`, `hasTouch`) và 1440×900**; thêm 320×568 cho TC-AS09-30. SV `sv.gioi`/`sv.kha`; `T`=token từ `display.qr_url`; URL quét `U = {gốc}/checkin?c={C}#{T}`. Gốc `http://localhost:580` (ngữ cảnh an toàn). `rotate_seconds=5` khi cần chờ hết hạn.

## Luồng quét
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS09-01 | AC1 | SV đã đăng nhập; `W` mở | Mở `U` (như camera hệ thống: `page.goto(U)`) | Trong ≤ 3 s hiện «Đã điểm danh 07:03» (giờ thật, chữ lớn); đúng **một** `POST /checkins`; không nhập / bấm gì; không nút; một dấu `Check` đỏ (nghĩa «đã xác nhận»), **không** đỏ nào khác | E2E |
| TC-AS09-02 | AC1 | – | Sau khi trang tải | `location.hash === ""` (`history.replaceState` xoá `#token`); `history.length` không tăng; nút Back không quay về URL có token | E2E |
| TC-AS09-03 | AC9 | – | Ghi toàn bộ request HTTP trong luồng (`page.on('request')`, kể cả `Referer`, URL điều hướng, request tài nguyên) | `T` **không** xuất hiện trong URL/`Referer` của bất kỳ request nào; chỉ có trong **thân** `POST /checkins`; thêm kiểm `Referrer-Policy` của trang | E2E |
| TC-AS09-04 | AC2 | Chưa đăng nhập | Mở `U` | Chuyển `/login?next=/checkin…`; `sessionStorage.ep.checkin` chứa `{c,token,at}` (không `localStorage`); sau đăng nhập quay lại và **tự gửi một lần**; `sessionStorage` bị xoá sau khi dùng | E2E |
| TC-AS09-05 | AC2 | – | Đăng nhập chậm > 60 s rồi quay lại | Token bị bỏ khỏi `sessionStorage` quá 60 s; hiện «Mã đã hết hạn. Quét lại mã trên màn chiếu.»; không `POST` | E2E |
| TC-AS09-06 | AC2 | `rotate_seconds=5` | Quét `U`, chờ > 12 s mới đăng nhập xong | `POST` → `422 CHECKIN_TOKEN_EXPIRED` → «Mã đã hết hạn. Quét lại mã trên màn chiếu.» | E2E |
| TC-AS09-07 | AC5 | `sv.gioi` đã điểm danh | Mở lại cùng `U` / tải lại | «Đã điểm danh 07:03» (giờ của lượt đầu, từ `200`); không thông báo lỗi; không `POST` thứ hai thay đổi dữ liệu | E2E |
| TC-AS09-08 | AC3 | Trình duyệt có WebCrypto + IndexedDB, lần đầu | Quét `U` | IndexedDB `edupilot.device.v1` có cặp khoá ECDSA P-256 `extractable=false` (kiểm `exportKey` ném lỗi); gọi `navigator.storage.persist()`; thân `POST` có `device:{kind:"WEB_KEY", public_key, signature}`; chữ ký xác thực với `public_key` trên `"edupilot-checkin-v1|"+T+"|"+user_id` (QC tự verify); lượt DB `assurance=2` | E2E+DB |
| TC-AS09-09 | AC3 | – | Cùng trình duyệt, **đăng xuất rồi đăng nhập `sv.kha`**, quét | Cùng một khoá (không theo tài khoản; đăng xuất **không** xoá khoá); hai lượt cùng `device_id` ⇒ `DEVICE_SHARED` cả hai (Staff thấy) | E2E+DB |
| TC-AS09-10 | AC3 | Chế độ riêng tư / IndexedDB bị chặn / không WebCrypto (HTTP IP LAN) | Quét `U` | Gửi **không** `device`; vẫn điểm danh được; `NO_DEVICE_KEY`; trang không báo lỗi | E2E |
| TC-AS09-11 | AC4 | – | Mã hết hạn; mã sai; cửa sổ đã đóng; `SELF_CHECKIN_DISABLED`; `403 reason=course`; `429`; mất mạng | Đúng một câu cho mỗi mã: «Mã đã hết hạn. Quét lại mã trên màn chiếu.» / «Mã không đúng. Quét lại mã trên màn chiếu.» / «Đã đóng điểm danh.» / «Lớp này không điểm danh bằng mã.» / «Bạn không thuộc lớp này.» / «Thử lại sau {n} giây.» (đếm lùi, nút khoá kèm lý do) / «Chưa có mạng.» + `Thử lại`; lỗi theo đường token dùng `InlineNotice` | E2E |
| TC-AS09-12 | AC4, P-108-11 | – | Quét token **hết hạn** khi phiên còn hạn | Không gọi `/auth/refresh`; không bị đăng xuất; không điều hướng `/login` (mã `CHECKIN_TOKEN_*` không thuộc `AUTH_CODES`) | E2E |
| TC-AS09-13 | AC4 | Mất mạng giữa chừng | Offline rồi `Thử lại` | Gửi lại an toàn (AC4 của US-AS-03): một lượt, không trùng | E2E |
| TC-AS09-14 | AC8 | – | TA, TEACHER, ADMIN mở `U` | Màn chặn chuẩn, **0 request** tới `/api/v1` | E2E |
| TC-AS09-15 | AC8 | – | STUDENT không thuộc lớp `c`; `c` không phải UUID | «Bạn không thuộc lớp này.» (từ `403`); «Liên kết không đúng.» (không `POST`) | E2E |

## Nhập mã ngắn (không có token)
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS09-20 | AC10 | – | Mở `/checkin` không `#token` | Ô «Mã điểm danh» (`autocapitalize=characters`, `autocomplete=off`, `autocorrect=off`, `spellcheck=false`, `enterkeyhint=go`, placeholder `ABC 123`); dòng «Lớp {mã lớp}» (lớp đang chọn ở thanh trên; đổi bằng bộ chọn có sẵn); nút `Điểm danh` (dock dưới ở 375) | UI |
| TC-AS09-21 | AC10 | `W` mở | Gõ `abc 123`-kiểu mã thật (chữ thường, có khoảng trắng) → `Điểm danh` | `POST {code}`; «Đã điểm danh 07:03»; lượt `channel='CODE'` | E2E |
| TC-AS09-22 | AC10 | – | Mã sai / hết hạn → «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» (lỗi **tại ô**); cửa sổ đóng → «Đã đóng điểm danh.»; 6 lần sai → «Thử lại sau {n} giây.» | Đúng câu; nút khoá có lý do khi `429`; ô giữ chữ đã gõ | E2E |
| TC-AS09-23 | AC10 | Lớp khác đang chọn | Nhập mã của lớp A khi đang chọn lớp B | `422` («Mã không đúng…»); đổi sang lớp A bằng bộ chọn rồi nhập → thành công | E2E |
| TC-AS09-24 | AC10 | Bàn phím ảo mở (375×568 giả lập `visualViewport`) | Chạm ô mã | Ô cuộn vào khung nhìn **trên** thanh dock; nút `Điểm danh` vẫn bấm được | E2E |

## 375 px, PWA, an toàn, giao diện
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS09-30 | AC7 | 375×812 và 320×568 | Mọi trạng thái: đang gửi, xong, nhập mã, lỗi (mỗi mã), 429 | Toàn bộ trong **một màn** (không cuộn cần thiết ở trạng thái thành công); không tràn ngang (`scrollWidth ≤ clientWidth`); nút / ô ≥ 44×44 px; chữ ≥ 16 px ở ô nhập (không zoom iOS) | UI |
| TC-AS09-31 | AC7 | Trang | `h1` «Điểm danh», không header meta, **một** `Panel`, không card lồng, không câu giải thích / mẹo / lời mời cài; chỉ token `--ep-*` | Khớp DESIGN §14.8.2 | UI |
| TC-AS09-32 | AC7 | HTTP thường (IP LAN, không an toàn) | Quét `U` | Vẫn điểm danh; không khoá thiết bị (`NO_DEVICE_KEY`); không lỗi giao diện | E2E |
| TC-AS09-33 | AC6 | – | `GET /manifest.webmanifest`; đọc đăng ký service worker | Manifest: `name`, `display: standalone`, icon, màu từ token; **không** service worker đệm API (`navigator.serviceWorker.getRegistrations()` rỗng); không lời mời cài | UI |
| TC-AS09-34 | AC6; US-AS-15 AC17 (P-11-137) | iPhone thật (hoặc Simulator cho phần tạo khoá); PWA đã cài vào Màn hình chính; `sv.gioi`; cửa sổ mở | 1) Trong PWA nhập mã → «Đã điểm danh», ghi `device_id` (SQL). 2) Lượt sau trong PWA (kể cả sau hơn 7 ngày không mở Safari nhưng có mở PWA). 3) Quét QR bằng camera hệ thống (mở Safari) điểm danh. 4) Staff xem lưới | 1) khoá ECDSA tạo và lưu ở IndexedDB của PWA, `DEVICE_NEW` một lần; 2) cùng `device_id`, không `DEVICE_NEW`; 3) Safari dùng khoá của Safari (kho khác PWA): `DEVICE_NEW` một lần cho khoá đó, có thể lại sau 7 ngày không mở Safari; khoá Safari không chuyển vào PWA; 4) hai khoá cùng `user_id` không sinh `DEVICE_SHARED`; mọi cờ không chặn, SV không thấy. WebKit Playwright chỉ kiểm `navigator.storage.persist` được gọi; không có iPhone → KHÔNG KIỂM ĐƯỢC (không tính F). Chi tiết thiết bị: `tc-mobile-app.md` PWA-10, PWA-13, PWA-17…23 | TAY |
| TC-AS09-35 | – | Khung 10.7 | `frame.spec.ts` có `/checkin` (loại `read`); `qc-ui-audit-real.spec.ts` thêm `/checkin` (student) 1440 + 375 | 0 lệch khung (E/H/P/D…); audit `read` | UI |
| TC-AS09-36 | – | axe + Lighthouse | `/checkin` ở trạng thái «đã quay lại» (xong) | axe 0 lỗi nghiêm trọng; LCP theo cổng 10.7 | UI |
| TC-AS09-37 | – | CSP | Đọc `Content-Security-Policy` của `/checkin` | Không nguồn script ngoài tên miền ứng dụng; `script-src` không `unsafe-eval` ở `/checkin` (kiểm bằng console CSP violation = 0) | E2E |
| TC-AS09-38 | – | `ui-antipatterns.sh` | Chạy | Sạch (kể cả kiểm 32 token projector không lọt vào `/checkin`) | DIFF |
| TC-AS09-39 | – | Bản chụp | Chụp ảnh các trạng thái 375 + 1440 vào `shots/` | Ảnh khớp DESIGN §14.8.2 (một dấu `Check` đỏ duy nhất; lỗi dùng danger notice, không đỏ nền) | UI |
