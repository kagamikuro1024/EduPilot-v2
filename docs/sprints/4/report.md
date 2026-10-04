# Sprint 4 — báo cáo (P2 Lớp học)

Mục tiêu: trọn `docs/phases/P2.md` — tài khoản thật (đăng nhập, đăng ký, xác minh email, đặt lại mật khẩu, chống dò, mời giảng viên / TA), lớp học (mở lớp, mã tham gia, duyệt, nạp danh sách lớp, `CourseAccessGuard`), "Hôm nay" cho mọi vai, seed bằng API thật · Kết quả: **12/12 story PASS · cổng P2 ĐẠT CÓ ĐIỀU KIỆN** sau 1 vòng sửa cổng · Nhánh `sprint/4-p2` (xếp chồng trên sprint 3)

| Story | Trạng thái | QC (TC) |
| --- | --- | --- |
| US-P2-01 Migration `00004 auth_hardening`, `internal/mail` (go-mail) + `mail_outbox` retry / dead-letter | PASS | `qc/report-US-P2-01.md` (35) |
| US-P2-02 Đăng nhập thật: access 15 phút + refresh xoay vòng cookie `httpOnly`, phát hiện dùng lại, `/login` | PASS có điều kiện | `qc/report-US-P2-02.md` (53) |
| US-P2-03 Đăng ký STUDENT + xác minh email (token băm, 24 h) | PASS có điều kiện | `qc/report-US-P2-03.md` (40) |
| US-P2-04 Quên / đặt lại / đổi mật khẩu, thiết bị đăng nhập, `/settings` | PASS có điều kiện | `qc/report-US-P2-04.md` (41) |
| US-P2-05 Chống dò: chờ tăng dần, khoá 15 phút + thư, giới hạn IP, mật khẩu ≥ 10 ký tự | PASS có điều kiện | `qc/report-US-P2-05.md` (41) |
| US-P2-06 Admin mời giảng viên / TA (link 72 h), khoá tài khoản, `/admin/users` | PASS có điều kiện | `qc/report-US-P2-06.md` (44) |
| US-P2-07 Migration `00003 course_foundation`, `CourseAccessGuard` thật, bộ chọn lớp | PASS có điều kiện | `qc/report-US-P2-07.md` (40) |
| US-P2-08 Admin mở / sửa / lưu trữ lớp, gán giảng viên / TA, chuông thông báo thật | PASS có điều kiện | `qc/report-US-P2-08.md` (43) |
| US-P2-09 Mã tham gia: xem trước + vào lớp, giới hạn đoán mã, duyệt thành viên | PASS sau sửa | `qc/report-US-P2-09.md` (55) |
| US-P2-10 Nạp danh sách lớp CSV / XLSX, **nối chỉ bằng email đã xác minh** | PASS sau sửa | `qc/report-US-P2-10.md` (41) |
| US-P2-11 "Hôm nay" cho mọi vai: `internal/today`, cache Redis xoá theo sự kiện | PASS có điều kiện | `qc/report-US-P2-11.md` (45) |
| US-P2-12 `scripts/seed.mjs` đi luồng API thật (2 lớp × 30 SV), bỏ phiên mô phỏng | PASS sau sửa | `qc/report-US-P2-12.md` (28) |

"Có điều kiện" = không lỗi chức năng; còn lệch nhỏ đã ghi hoặc TC phụ thuộc hạ tầng khác. Cổng: `qc/report-GATE-P2.md` (48 TC). Spec: `FEAT-account-security` v1.7, `FEAT-course-foundation` v1.7.

## Số liệu
- Test case: **554** (story 506 · GATE 48). Cổng vòng 1: CHƯA ĐẠT (CI Lighthouse TBT, seed B1) → vòng sửa 1 → ĐẠT CÓ ĐIỀU KIỆN, 0 FAIL.
- **0 lỗ hổng bảo mật** qua các đợt tấn công của QC: leo thang vai, mạo danh MSSV (chặn ở `PENDING | EMAIL_MISMATCH`), đoán mã lớp (100 mã ngẫu nhiên: 5 × `404`, 95 × `429`, 0 trúng), so thời gian đăng nhập, CSRF chéo site, rò mật khẩu / token / email trong log (0).
- `go test -race -tags integration`: 733 PASS, 0 FAIL; `golangci-lint` 0 issues; `sqlc diff` sạch. Playwright 327 pass (ngoài ảnh mốc). CI xanh ở HEAD.
- Seed từ DB trống: 37 s (ngưỡng 180 s).
- Lighthouse: JS 237–251 KB (< 256 KB), CLS 0; **LCP 3,2–4,1 s và TBT 184–239 ms trên runner CI** — cả hai đang `warn` (xem Nợ).
- Migration: `00003 course_foundation`, `00004 auth_hardening`, `00005 vn_fold`. Góp ý #1–#15, PM quyết cả 15 (1 chọn phương án thay thế).

## Lỗi thật tìm ra và đã sửa
- Redis lỗi thì mỗi yêu cầu vào lớp ghi một dòng log và chờ ≈ 5 s (B1 P2-09) → log ≤ 1 dòng / 30 s, hạn Redis 250 ms để mở cửa nhanh.
- Seed lần đầu để giảng viên có 5 thông báo chưa đọc thay vì 1 (B1 P2-12) — do đánh dấu đọc trước khi worker tạo xong.
- Mã lỗi dòng khi nạp danh sách lớp chồng nghĩa (#11) → tách "lặp trong tệp" với "trùng ghi danh có sẵn".
- Bộ chọn lớp mặc định vào lớp đã lưu trữ (L6) → lớp đang hoạt động.
- Test chập chờn: `TestSSE_ReconnectRace`, `TestRateLimit_*`, `TestLoginTimingEqualized`, e2e chưa chờ hydrate.

## Quyết định trong sprint
- Mặc định sản phẩm (chủ dự án chưa chốt riêng, đổi được): mật khẩu ≥ 10 ký tự; khoá 15 phút sau 10 lần sai, đặt lại mật khẩu mở khoá ngay; phiên trượt 14 ngày, tối đa 30 ngày; Admin không thấy MSSV; sinh viên bị xoá vào lại thì về `PENDING`.
- Tìm thành viên theo MSSV chỉ cho giảng viên / TA của lớp (#10). `link` thông báo rỗng = mở "Hôm nay" (#9). Hai route `llm-budget` là ngoại lệ đóng của `CourseAccessGuard` (#7). Tìm không dấu bằng hàm SQL `vn_fold`, không extension (#6).
- Lighthouse đo bằng gateway giả chỉ trong công cụ đo (#13). LCP (#14) và TBT (#15) giữ `warn`, ngưỡng không đổi, **hạn trả về `error` là US-PU-06 đầu sprint 5** (khung vẽ phía máy chủ). Tech Lead đo và chứng minh 4× là chuẩn đúng, không hạ hệ số CPU (TL-1).

## Nợ
- **US-PU-06 (sprint 5):** LCP ≤ 2,5 s và TBT ≤ 200 ms về `error` trên CI.
- `gate-pg.sh` cố định project `edupilot` — cần nhận `EP_PORT_OFFSET` để QC chạy song song.
- Ảnh mốc chỉ so được trong container Linux (11 / 14 lệch trên macOS) — QC chạy `visual` trong image Playwright.
- `proto-curl.sh` / `sweep.mjs` của sprint 1.5 dùng cookie mô phỏng đã bỏ; QC thay bằng `audit-login.mjs` (130 hàng, 0 FAIL).
- Chưa chạy: seed bằng Bun, ca `@real`, ngắt seed giữa chừng.

## Chủ dự án tự kiểm
- `pnpm dev:down -v && SEED_ON_EMPTY_DB=true pnpm dev`, đăng nhập 7 tài khoản mẫu (mật khẩu ở `SEED_DEFAULT_PASSWORD`): mỗi vai thấy "Hôm nay" khác nhau.
- Đăng ký một tài khoản sinh viên mới, xác minh email qua Mailpit, vào lớp bằng mã `AN7K2MQ` → giảng viên thấy yêu cầu chờ duyệt.
- Sai mật khẩu 10 lần → bị khoá 15 phút, có thư báo; đặt lại mật khẩu → vào được ngay.
- Nạp một tệp CSV có MSSV trùng nhau → báo đúng dòng lỗi, không nối tài khoản theo MSSV.
