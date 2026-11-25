# QC 10.8 — ỨNG DỤNG ĐIỆN THOẠI, PHA 1 (qc, cặp 3) — `tc-mobile-run-qc.md`

Tệp kết quả RIÊNG của qc cho pha 1 của `tc-mobile-app.md` (qc3 chủ tệp, gộp vào §9 sau). Nhóm của qc: **6.B, 6.C, 6.F, 6.G, 6.I, 6.K, 6.N, 6.R, 6.S, 6.X**.

## Môi trường

| Mục | Giá trị |
| --- | --- |
| Môi trường | **W** — Playwright **Chromium** trên VPS (`ep-vps-test`), thiết bị giả `Pixel 7` (412×915) và `iPhone 15` (393×852): **Chromium giả iPhone, KHÔNG WebKit** — Safari/WebKit chỉ có ở pha 3 |
| Nơi chạy | VPS `edupilot-vps`, HTTPS thật `https://v2202609419695521098.happysrv.de` (sha VPS: f6113582, 25/11 23:51 — có DEF-R10.8-07 dock theo chiều cao thật) |
| Lớp QC | `QCMOB1` (self_checkin bật, rotate 10 s, 41 buổi), `QCMOB2` (lớp thứ hai, bật), `QCMOB4B` (đã lưu trữ **qua giao diện Admin** thay QCMOB4), `QCMOB5` (self_checkin tắt), `QCMOB6` (sv.qc không thuộc) — không đụng 761987/761988, QCVPSA–D |
| Tài khoản | `sv.qc` (SV duy nhất qc dùng; **không** dùng sv.qc2/sv.qc2b của qc2), teacher, ta, admin |
| Số lượt | Cửa sổ điểm danh mở bằng API GV (`/sessions/{sid}/checkin-windows`), tối đa 3 vòng/buổi; bản ghi chỉ hiện ở lưới sau khi **đóng cửa sổ** |
| Kết quả | **P** đạt · **F** trượt (kèm DEF) · **N** không kiểm được (nêu lý do) |

## 6.B Phiên, vai, tài khoản

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| SES-01 | **P** | Đăng nhập giao diện 500 ms (đo cả tải /login), URL/localStorage không chứa token |
| SES-02 | **P** | 5/5 lần mở lại vào thẳng, không màn đăng nhập/trắng |
| SES-03 | **N** | cần `am force-stop` (native, pha 2) |
| SES-04 | **P** | Đăng xuất → /login, Back không vào lại (about:blank), `sessionStorage.ep.checkin` rỗng |
| SES-05 | **P** | teacher + ta bị chặn ở `/checkin` (không ô nhập, 0 POST `/checkins`), `/attendance` vẫn dùng được |
| SES-06 | **P** | admin bị chặn, 0 POST |
| SES-07 | **N** | cần 2 tài khoản SV cùng thiết bị (sv.qc2 thuộc qc2 — không dùng); pha 2 |
| SES-08 | **N** | native (thanh thông báo hệ thống) |
| SES-09 | **N** | cần access token hết hạn thật (TTL) — không đổi giờ máy chủ |
| SES-10 | **N** | cần thu hồi phiên từ thiết bị khác (thiết bị thật) |
| SES-11 | **P** | Chưa đăng nhập: chuyển `/login`, cất `{c,token,at}` vào sessionStorage (không localStorage) — cùng bằng chứng với ERR-06/SES-11a |
| SES-12 | **N** | cần Back của Android / vuốt iOS |
| SES-13 | **N** | cần `pm clear` |
| SES-14 | **N** | cần hai SV cùng trình duyệt (sv.qc2 không dùng) |

## 6.C Luồng điểm danh chính

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| FLW-01 | **P** | Token QR tự gửi một lần, «Đã điểm danh 00:11» sau **171 ms**; `#token` biến khỏi URL; chỉ dấu Check đỏ (svg/path), không chữ nào đỏ; cửa sổ ghi 1 lượt |
| FLW-02 | **N** | `simctl openurl` (pha 3) |
| FLW-03 | **P** | Ba kiểu mã `5mk9v5` / `5MK 9V5` / `5MK-9V5` đều nhận, cùng giờ; cửa sổ 1 lượt |
| FLW-04 | **P** | Ô «Mã điểm danh» + «Lớp QCMOB1» + nút Điểm danh; không nút quét (Chromium không BarcodeDetector) |
| FLW-05 | **P** | Mở lại cùng liên kết: hiện lại đúng giờ lượt đầu, không lỗi |
| FLW-06 | **P** | Chạm đúp < 300 ms: **1 POST**, nút khoá khi đang gửi, cửa sổ 1 lượt |
| FLW-07 | **P** | Xoay 412×915 → 915×412 giữa lúc gửi: 1 lượt, «Đã điểm danh» hiện sau xoay |
| FLW-08 | **P** | 31 yêu cầu ghi lại: token không có trong URL/Referer/yêu cầu nào |
| FLW-09 | **P** | Mã QCMOB2 khi đang chọn QCMOB1 → «Mã không đúng hoặc đã hết hạn…»; mã đúng lớp → «Đã điểm danh» |
| FLW-10 | **N** | không còn tài khoản nào không thuộc lớp (sv.qc* đều đã ghi danh) |
| FLW-11 | **P** | Vòng 2 (round 2) tạo lượt mới, cửa sổ ghi 1 lượt |
| FLW-12 | **P** | Lớp tắt self_checkin: «Lớp này không điểm danh bằng mã.», **không còn ô mã** (đúng AC16(d)), 0 POST |
| FLW-13 | **F (Thấp)** | Lớp **lưu trữ** hiện «Lớp này không điểm danh bằng mã.» — câu này nói sai nguyên nhân; AC (FLW-13) đòi câu về lớp đã lưu trữ. Máy chủ trả `attendance.self_checkin=false` cho lớp ARCHIVED nên giao diện rơi vào nhánh tắt điểm danh → **DEF-MOB-01** |
| FLW-14 | **N** | cần `pnpm build:gate` + bộ `attendance-checkin.spec.ts` trên build (chạy ở lượt riêng) |
| FLW-15/16/17/18 | **N** | hai AVD / camera hệ thống / LAN HTTP / IndexedDB chặn (native) |

## 6.F Nhánh lỗi (ERR) + 6.S câu chữ (phần web)

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| ERR-01 | **P** | `c=khong-phai-uuid` → «Liên kết không đúng.», 0 POST |
| ERR-02 | **P** | Token cũ > 2 bước rotate → «Mã đã hết hạn. Quét lại mã trên màn chiếu.» |
| ERR-03 | **P** | Sửa ký tự cuối → «Mã không đúng. Quét lại mã trên màn chiếu.» |
| ERR-04 | **P** | Cửa sổ đã đóng → «Đã đóng điểm danh.» |
| ERR-05 | **P** | sv.qc mở liên kết QCMOB6 (không thuộc) → «Bạn không thuộc lớp này.», `checked_in=0` |
| ERR-06 | **P** | Để màn đăng nhập > 60 s rồi đăng nhập → «Mã đã hết hạn…», 0 POST. Ghi chú: kho cất `ep.checkin` **không bị xoá** (không thuộc AC) |
| ERR-07 | **P** | teacher mở liên kết: màn chặn, 0 POST, token không lộ trong DOM |
| ERR-08 | **P** | ta + admin: như ERR-07 |
| ERR-09 | **N** | đếm ngược 429 cần 6 lần nhập sai liên tiếp — để ca riêng (chưa chạy: sợ chạm hạn mức của sv.qc cho các ca còn lại) |
| ERR-10 | **N** | cần tải 11 lượt/phút rồi quét ngay (native+API) |
| ERR-11 | **P** | Mã sai → «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» dưới ô, `aria-invalid=true`, không hộp thoại |
| ERR-12 | **P** | Ô rỗng → nút khoá; ký tự lạ `😀@#$` → cùng một câu, 0 yêu cầu, không crash |
| ERR-13 | **P** | Dán 40 ký tự → cắt còn 12; mã có khoảng trắng đầu/cuối vẫn điểm danh được |
| ERR-14 | **P** | 500 JSON / 200 HTML / 500 rỗng → «Máy chủ chưa phản hồi đúng. Dữ liệu của bạn vẫn an toàn.» (câu BAD_GATEWAY của SRS ui-foundation), không mã HTTP/JSON; nút còn dùng. *Khác câu gợi ý «Chưa điểm danh được. Thử lại.» trong TC — đề nghị qc3/PM chỉnh TC theo SRS, không phải lỗi sản phẩm* |
| ERR-15 | **N** | offline lần đầu mở liên kết (Safari iOS, pha 3) |
| ERR-16/17 | **N** | đổi giờ/múi giờ máy (native) |
| ERR-18 | **P** | `POST {code, student_id}` → 422 `VALIDATION_FAILED`, `details[].code=unknown`, không ghi lượt |
| ERR-19 | **P** | Lớp không tồn tại → cùng một câu, không lộ khác biệt |
| ERR-20 | **N** | cửa sổ đóng > 15 phút — chờ dài, để lượt sau |
| MSG-10 | **P** | Không `BarcodeDetector`: không câu lỗi nào, ô nhập + nút hiện ngay |
| MSG-01…09, 11…16 | một phần **P** qua ERR-01…05/11/12/14/18 (đúng câu); còn lại **N** (native) |  |

## 6.G / 6.I / 6.K / 6.N / 6.R / 6.X

(chờ lô C chạy xong — điền ở lượt sau)

## DEF mới

| DEF | Mức | Nội dung | Bằng chứng |
| --- | --- | --- | --- |
| DEF-MOB-01 | Thấp | Lớp đã lưu trữ hiện «Lớp này không điểm danh bằng mã.» thay vì câu về lớp lưu trữ (FLW-13) | Máy chủ: lớp ARCHIVED trả `attendance.self_checkin=false`; giao diện chỉ có nhánh "tắt điểm danh" |

## Ghi chú

- Quan sát tĩnh §5 của `tc-mobile-app.md` không chấm ở pha 1 trừ ca W tương ứng.
- Mọi ca dựng trạng thái bằng `page.route` (gửi/đã gửi/lỗi/429) chỉ đo bố cục, không tạo lượt trong DB.
